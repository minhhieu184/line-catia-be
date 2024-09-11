package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"millionaire/internal/pkg/ton_utils"
	"strconv"
	"strings"
	"time"

	"github.com/tonkeeper/tongo"

	"github.com/go-redsync/redsync/v4"
	"github.com/hiendaovinh/toolkit/pkg/errorx"
	"github.com/redis/go-redis/v9"
	"github.com/samber/do"
	"github.com/uptrace/bun"

	"millionaire/internal/datastore"
	"millionaire/internal/datastore/redis_store"
	"millionaire/internal/models"
	"millionaire/internal/pkg/caching"
)

var ErrUserLock = errors.New("user locked")

const MessageNewUser = `🎉 Great news! %s has just joined Catia Eduverse.

You will get 1 Star 🌟 when your invited friend gets Gem. Check button "Frens" in the app for more details.

Let's refer more friends to multiply your Stars for game power. ✨`

type ServiceUser struct {
	container          *do.Injector
	redisDB            redis.UniversalClient
	redisDBCache       redis.UniversalClient
	rs                 *redsync.Redsync
	postgresDB         *bun.DB
	readonlyPostgresDB *bun.DB
	cache              caching.Cache
	readonlyCache      caching.ReadOnlyCache

	serviceUserGame     *ServiceUserGame
	bot                 *Bot
	serviceConfig       *ServiceConfig
	serviceUserFreebies *ServiceUserFreebies
	serviceReward       *ServiceReward
}

func NewServiceUser(container *do.Injector) (*ServiceUser, error) {
	db, err := do.InvokeNamed[redis.UniversalClient](container, "redis-db")
	if err != nil {
		return nil, err
	}

	rs, err := do.Invoke[*redsync.Redsync](container)
	if err != nil {
		return nil, err
	}

	postgresDB, err := do.Invoke[*bun.DB](container)
	if err != nil {
		return nil, err
	}

	cache, err := do.Invoke[caching.Cache](container)
	if err != nil {
		return nil, err
	}

	readonlyCache, err := do.Invoke[caching.ReadOnlyCache](container)
	if err != nil {
		return nil, err
	}

	serviceUserGame, err := do.Invoke[*ServiceUserGame](container)
	if err != nil {
		return nil, err
	}

	bot, err := do.Invoke[*Bot](container)
	if err != nil {
		return nil, err
	}

	serviceConfig, err := do.Invoke[*ServiceConfig](container)
	if err != nil {
		return nil, err
	}

	serviceUserFreebies, err := do.Invoke[*ServiceUserFreebies](container)
	if err != nil {
		return nil, err
	}

	serviceReward, err := do.Invoke[*ServiceReward](container)
	if err != nil {
		return nil, err
	}

	readonlyPostgresDB, err := do.InvokeNamed[*bun.DB](container, "db-readonly")
	if err != nil {
		return nil, err
	}

	dbRedisCache, err := do.InvokeNamed[redis.UniversalClient](container, "redis-cache")
	if err != nil {
		return nil, err
	}

	return &ServiceUser{container, db, dbRedisCache, rs, postgresDB, readonlyPostgresDB, cache, readonlyCache, serviceUserGame, bot, serviceConfig, serviceUserFreebies, serviceReward}, nil
}

func (service *ServiceUser) FindOrCreateUser(ctx context.Context, userAuth *models.UserFromAuth) (*models.User, error) {
	if userAuth == nil {
		return nil, errors.New("userAuth is nil")
	}
	user, _ := service.FindUserByID(ctx, userAuth.ID)
	b, _ := json.MarshalIndent(user, "", "    ")
	fmt.Println(string(b))

	if user != nil {
		if (user.Username != strings.ToLower(userAuth.Username)) ||
			(user.FirstName != userAuth.FirstName) ||
			(user.LastName != userAuth.LastName) ||
			(user.PhotoURL != userAuth.PhotoURL) {
			user.Username = strings.ToLower(userAuth.Username)
			user.FirstName = userAuth.FirstName
			user.LastName = userAuth.LastName
			user.PhotoURL = userAuth.PhotoURL
			datastore.UpdateUserProfile(ctx, service.postgresDB, user)
			_ = service.cache.Delete(ctx, DBKeyUser(user.ID))
		}
		return user, nil
	}

	now := time.Now()
	newUser := &models.User{
		ID:           userAuth.ID,
		FirstName:    userAuth.FirstName,
		IsBot:        userAuth.IsBot,
		IsPremium:    userAuth.IsPremium,
		LastName:     userAuth.LastName,
		Username:     strings.ToLower(userAuth.Username),
		LanguageCode: userAuth.LanguageCode,
		PhotoURL:     userAuth.PhotoURL,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	log.Println("Create new user:", "user:", newUser.ID, "username:", newUser.Username)
	user, err := datastore.CreateUser(ctx, service.postgresDB, newUser)
	if err != nil {
		return nil, err
	}

	user.IsNewUser = true

	//create new user freebie - 3 total
	_, err = service.serviceUserFreebies.GetOrNewUserFreebies(ctx, user.ID)
	if err != nil {
		return user, err
	}

	go func() {
		err = service.bot.SendWelcomeMsg(user.ID)
		if err != nil {
			log.Println(err)
		}
	}()

	return user, nil
}

func (service *ServiceUser) FindUserByID(ctx context.Context, userID int64) (*models.User, error) {
	callback := func() (*models.User, error) {
		return datastore.FindUserByID(ctx, service.readonlyPostgresDB, userID)
	}
	return caching.UseCacheWithRO(ctx, service.readonlyCache, service.cache, DBKeyUser(userID), CACHE_TTL_5_MINS, callback)
}

func (service *ServiceUser) FindUserByIDNoCache(ctx context.Context, userID int64) (*models.User, error) {
	return datastore.FindUserByID(ctx, service.readonlyPostgresDB, userID)
}

func (service *ServiceUser) UpdateUser(ctx context.Context, user *models.User) (*models.User, error) {
	if user == nil {
		return nil, errors.New("user is nil")
	}

	user.Username = strings.ToLower(user.Username)

	user, err := datastore.EditUser(ctx, service.readonlyPostgresDB, user)
	if err != nil {
		return nil, err
	}

	// delete key user in redis
	err = service.cache.Delete(ctx, DBKeyUser(user.ID))
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (service *ServiceUser) AddReferenceCode(ctx context.Context, user *models.User, inviter *models.User) error {
	if inviter == nil {
		return errors.New("inviter is nil")
	}

	if user.ID == inviter.ID {
		return errors.New("user cannot invite himself")
	}

	if user.InviterID != nil {
		return errors.New("user already has an inviter")
	}

	// TODO check user total score from db
	if user.TotalScore >= 1 {
		return errors.New("user has already played and earned more than 200 scores")
	}

	err := datastore.AddInviteRef(ctx, service.postgresDB, user.ID, inviter.ID)
	if err != nil {
		return err
	}

	username := fmt.Sprintf("@%s", user.Username)
	if user.Username == "" {
		username = strings.TrimSpace(fmt.Sprintf("%s %s", user.FirstName, user.LastName))
	}

	go func() {
		if inviter.ID < 100 {
			return
		}
		err = service.bot.SendMsg(inviter.ID, fmt.Sprintf(MessageNewUser, username))
		if err != nil {
			log.Println(err)
		}
	}()

	_, err = redis_store.SetLeaderboard(ctx, service.redisDB, LEADERBOARD_REFERRAL, &models.LeaderboardItem{
		UserId: inviter.ID,
		Score:  float64(inviter.TotalInvites + 1),
	})

	err = service.cache.Delete(ctx, DBKeyUser(user.ID))
	if err != nil {
		log.Println(err)
	}

	err = service.cache.Delete(ctx, DBKeyUser(inviter.ID))
	if err != nil {
		log.Println(err)
	}

	log.Println("AddReferenceCode updated:", "user:", user.ID, "username:", user.Username, "inviterID:", inviter.ID)

	return err
}

func (service *ServiceUser) CountBoosts(ctx context.Context, userID int64) (int, error) {
	count, err := datastore.CountUserBoosts(ctx, service.readonlyPostgresDB, userID)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (service *ServiceUser) IsWinner(ctx context.Context, userID int64) (bool, error) {
	user, err := service.FindUserByID(ctx, userID)
	if err != nil {
		return false, err
	}

	if user == nil {
		return false, nil
	}

	userRank, err := redis_store.GetRank(ctx, service.redisDB, "catia", user)
	if err != nil {
		return false, err
	}

	return userRank < 50, nil
}

func (service *ServiceUser) Me(ctx context.Context, user *models.User, refCode string) (*models.User, error) {
	if user == nil {
		return nil, errors.New("user not found")
	}

	callback := func() (*models.User, error) {
		me, err := datastore.FindUserByID(ctx, service.readonlyPostgresDB, user.ID)
		if err != nil {
			return me, err
		}
		count, err := service.CountBoosts(ctx, me.ID)
		if err != nil {
			return me, err
		}
		me.Boosts = count

		gem, err := service.GetUserGem(ctx, user.ID)
		if err == nil {
			me.TotalScore = gem
		}

		wallet, _ := service.FindUserWalletByUserID(ctx, me.ID)

		if wallet != nil {
			me.EVMWallet = wallet.EVMWallet
			me.TONWallet = wallet.TONWallet
		}

		if user.IsNewUser && refCode != "" {
			if me.InviterID != nil {
				log.Println("AddReferenceCode abort: user already has refCode", "user:", me.ID, "username:", me.Username, "refCode:", refCode)
				return me, nil
			}

			// refcode is userID
			inviter, err := service.GetUserIdByRefCode(ctx, refCode)

			if inviter == nil {
				log.Println("AddReferenceCode abort: cannot parse refcode", "user:", me.ID, "username:", me.Username, "refCode:", refCode)
				return me, nil
			}

			err = service.AddReferenceCode(ctx, me, inviter)
			if err != nil {
				log.Println("AddReferenceCode error:", err, "user:", me.ID, "username:", me.Username, "refCode:", refCode)
			}
		} else if refCode != "" {
			log.Println("AddReferenceCode abort: user is not new user", "user:", me.ID, "username:", me.Username, "refCode:", refCode)

		}

		_, err = service.serviceUserFreebies.GetOrNewUserFreebies(ctx, me.ID)
		if err != nil {
			return me, err
		}

		userGem, err := service.GetUserGem(ctx, me.ID)
		if err != nil {
			return me, err
		}

		me.TotalScore = userGem
		return me, nil
	}

	me, err := caching.UseCacheWithRO(ctx, service.readonlyCache, service.cache, DBKeyMe(user.ID), CACHE_TTL_5_MINS, callback)

	rewards, err := service.serviceReward.GetAvailableRewardByUserID(ctx, user.ID)

	if rewards != nil {
		me.AvailableRewards = rewards

		// TODO claim all reward, used for aethir campaign only, remove later
		for _, r := range rewards {
			service.serviceReward.ClaimReward(ctx, r.ID)
		}
		service.serviceReward.ClearUserAvailableRewardCache(ctx, user.ID)
	}
	if me != nil && user.IsNewUser {
		me.IsNewUser = user.IsNewUser
	}

	return me, err
}

func (service *ServiceUser) GetUserGem(ctx context.Context, userID int64) (int, error) {
	callback := func() (int, error) {
		return datastore.GetUserTotalGem(ctx, service.readonlyPostgresDB, userID)
	}

	//TODO clear DBKeyUserGems
	return caching.UseCacheWithRO(ctx, service.readonlyCache, service.cache, DBKeyUserGems(userID), CACHE_TTL_1_HOUR, callback)
}

func (service *ServiceUser) GetUserGemByAction(ctx context.Context, userID int64, action string) (*models.UserGem, error) {
	callback := func() (*models.UserGem, error) {
		return datastore.GetUserGemByAction(ctx, service.readonlyPostgresDB, userID, action)
	}

	return caching.UseCacheWithRO(ctx, service.readonlyCache, service.cache, DBKeyUserGemAction(userID, action), CACHE_TTL_1_HOUR, callback)
}

func (service *ServiceUser) GetUserGemNoCache(ctx context.Context, userID int64) (int, error) {
	// User write db to prevent replica lag
	return datastore.GetUserTotalGem(ctx, service.postgresDB, userID)
}

func (service *ServiceUser) GetUserGemFromTimeNoCache(ctx context.Context, userID int64, from *time.Time) (int, error) {
	// User write db to prevent replica lag
	return datastore.GetUserTotalGemFromTime(ctx, service.postgresDB, userID, from)
}

func (service *ServiceUser) InsertUserGem(ctx context.Context, user *models.User, gems int, action string) error {
	var userGem models.UserGem
	userGem.UserID = user.ID
	userGem.Gems = gems
	userGem.Action = action

	err := datastore.InsertUserGem(ctx, service.postgresDB, &userGem)
	if err != nil {
		return err
	}

	serviceLeaderboard, err := do.Invoke[*ServiceLeaderboard](service.container)
	if err != nil {
		return err
	}

	_, err = serviceLeaderboard.UpdateOverallLeaderboard(ctx, user)
	if err != nil {
		return err
	}

	err = service.ClearUserGemCache(ctx, user.ID)
	if err != nil {
		log.Println(err)
	}

	return nil
}

func (service *ServiceUser) ClaimUserBoost(ctx context.Context, source string, userId int64, page int, limit int) error {
	mutex := service.rs.NewMutex(LockKeyUserClaimBoost(userId, source))
	if err := mutex.TryLock(); err != nil {
		return errorx.Wrap(ErrUserBoostLock, errorx.Invalid)
	}

	// nolint:errcheck
	defer mutex.Unlock()

	inviteeId, err := strconv.ParseInt(source, 10, 64)

	// TODO more condition when boost can come from other source in the future
	if err != nil {
		return errors.New("invalid source")
	}

	gem, err := service.GetUserGem(ctx, inviteeId)
	if err != nil {
		return err
	}

	minScore, _ := service.serviceConfig.GetIntConfig(ctx, CONFIG_MIN_GEM_TO_CLAIM_REF_BOOST, MIN_GEM_TO_CLAIM_REF_BOOST)

	if gem < minScore {
		return errors.New("Your friend must have at least 16 gems to claim the boost")
	}

	// check if user already have a boost
	hasBoost, err := service.CheckUserBoostExists(ctx, userId, source)
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	if !hasBoost {
		userBoost := &models.UserBoost{
			UserID:    userId,
			UsedAt:    nil,
			CreatedAt: time.Now(),
			Source:    source,
			Used:      false,
			UsedFor:   "",
			Validated: true,
		}

		err = service.CreateBoost(ctx, userBoost)
		if err != nil {
			log.Println("error create boost", err)
			return err
		}

		friendCache, err := service.GetUserFriendListPaging(ctx, userId, page, limit)
		if err != nil {
			log.Println(err)
		}

		//update this friend
		for _, friend := range friendCache {
			if friend.ID == inviteeId {
				friend.Claimed = true
			}
		}

		service.cache.Set(ctx, DBKeyUserFriendList(userId, page, limit), friendCache, CACHE_TTL_5_MINS)
	}

	return nil
}

func (service *ServiceUser) CreateBoost(ctx context.Context, userBoost *models.UserBoost) error {
	if userBoost == nil {
		return errors.New("userBoost is nil")
	}

	err := datastore.CreateBoost(ctx, service.postgresDB, userBoost)
	if err != nil {
		log.Println("error when creating boost", err)
		return err
	}

	return service.ClearUserCache(ctx, userBoost.UserID)
}

func (service *ServiceUser) ClaimAllAvailableBoostFromFriends(ctx context.Context, user *models.User) (int, error) {
	mutex := service.rs.NewMutex(LockKeyUserClaimAllBoost(user.ID))
	if err := mutex.TryLock(); err != nil {
		return 0, errorx.Wrap(ErrUserBoostLock, errorx.Invalid)
	}

	// nolint:errcheck
	defer mutex.Unlock()
	friendList, err := datastore.GetOnlyClaimableFriends(ctx, service.readonlyPostgresDB, user.ID)
	if err != nil {
		return 0, err
	}

	if len(friendList) == 0 {
		return 0, errorx.Wrap(errors.New("no friend to claim"), errorx.NotExist)
	}

	var sources []string

	for _, friend := range friendList {
		sources = append(sources, fmt.Sprintf("%d", friend.ID))
	}

	err = service.InsertBoostsWithSources(ctx, user, sources)
	if err != nil {
		return 0, err
	}

	err = service.DeleteFriendListCaching(ctx, user.ID)
	if err != nil {
		return 0, err
	}

	return len(friendList), nil
}

func (service *ServiceUser) InsertBoosts(ctx context.Context, user *models.User, source string, amount int) error {
	if user == nil {
		return errors.New("user is nil")
	}
	userBoosts := []*models.UserBoost{}

	now := time.Now()
	for i := 0; i < amount; i++ {
		userBoost := models.UserBoost{
			UserID:    user.ID,
			Source:    fmt.Sprintf("%s:%d", source, i),
			CreatedAt: now,
			Validated: true,
			Used:      false,
		}
		userBoosts = append(userBoosts, &userBoost)
	}

	err := datastore.CreateMultipleBoost(ctx, service.postgresDB, userBoosts)
	if err != nil {
		log.Println("error when creating boost", err)
		return err
	}

	return service.ClearUserCache(ctx, user.ID)
}

func (service *ServiceUser) InsertBoostsWithSources(ctx context.Context, user *models.User, sources []string) error {
	if user == nil {
		return errors.New("user is nil")
	}
	userBoosts := []*models.UserBoost{}

	now := time.Now()
	for _, source := range sources {
		userBoost := models.UserBoost{
			UserID:    user.ID,
			Source:    source,
			CreatedAt: now,
			Validated: true,
			Used:      false,
		}
		userBoosts = append(userBoosts, &userBoost)
	}

	err := datastore.CreateMultipleBoost(ctx, service.postgresDB, userBoosts)
	if err != nil {
		log.Println("error when creating boost", err)
		return err
	}

	return service.ClearUserCache(ctx, user.ID)
}

func (service *ServiceUser) ChangeLifelineBalance(ctx context.Context, user *models.User, action string, changedAmount int) error {
	err := datastore.ChangeUserLifelineBalance(ctx, service.postgresDB, user.ID, changedAmount)
	if err != nil {
		return err
	}

	history := &models.LifelineHistory{
		UserID: user.ID,
		Action: action,
		Change: changedAmount,
	}

	err = datastore.InsertLifelineHistory(ctx, service.postgresDB, history)
	if err != nil {
		return err
	}

	return service.ClearUserCache(ctx, user.ID)
}

func (service *ServiceUser) GetUserFriendListPaging(ctx context.Context, userID int64, page int, limit int) ([]*models.Friend, error) {
	callback := func() ([]*models.Friend, error) {
		offset := page * limit
		claimableFriends, err := datastore.GetUserFriendListPaging(ctx, service.readonlyPostgresDB, userID, limit, offset)
		if err != nil {
			return nil, err
		}

		return claimableFriends, nil
	}
	return caching.UseCacheWithRO(ctx, service.readonlyCache, service.cache, DBKeyUserFriendList(userID, page, limit), CACHE_TTL_5_MINS, callback)
}

func (service *ServiceUser) CountUserFriends(ctx context.Context, userID int64) (int, error) {
	callback := func() (int, error) {
		return datastore.CountFriends(ctx, service.readonlyPostgresDB, userID)
	}

	return caching.UseCacheWithRO(ctx, service.readonlyCache, service.cache, DBKeyFriendCount(userID), CACHE_TTL_5_MINS, callback)
}

func (service *ServiceUser) CheckUserBoostExists(ctx context.Context, userID int64, source string) (bool, error) {
	callback := func() (bool, error) {
		return datastore.CheckUserBoostExists(ctx, service.readonlyPostgresDB, userID, source)
	}

	return caching.UseCacheWithRO(ctx, service.readonlyCache, service.cache, DBKeyBoostExist(userID, source), CACHE_TTL_5_MINS, callback)
}

func (service *ServiceUser) DeleteFriendListCaching(ctx context.Context, userID int64) error {
	caching.DeleteKeys(ctx, service.redisDBCache, fmt.Sprintf("user_friend_list:%d:*", userID))
	return nil
}

func (service *ServiceUser) FindUserWalletByUserID(ctx context.Context, userID int64) (*models.UserWallet, error) {
	callback := func() (*models.UserWallet, error) {
		return datastore.FindUserWalletByUserID(ctx, service.readonlyPostgresDB, userID)
	}

	return caching.UseCacheWithRO(ctx, service.readonlyCache, service.cache, DBKeyUserWallet(userID), CACHE_TTL_5_MINS, callback)
}

func (service *ServiceUser) ClearUserGemCache(ctx context.Context, userID int64) error {
	err := service.cache.Delete(ctx, DBKeyUserGems(userID))
	if err != nil {
		log.Println(err)
	}

	_ = service.ClearUserCache(ctx, userID)
	return nil
}

func (service *ServiceUser) ConnectTonWallet(ctx context.Context, user *models.User, payload *models.TonProof) error {
	if payload == nil {
		return errors.New("invalid payload")
	}

	userWallet, err := service.FindUserWalletByUserID(ctx, user.ID)

	if err != nil && err != sql.ErrNoRows {
		return err
	}

	if userWallet != nil && userWallet.TONWallet != nil {
		return errorx.Wrap(errors.New("already connected"), errorx.Invalid)
	}

	parsed, err := ton_utils.ParseTonProofMessage(payload)
	if err != nil {
		return err
	}

	addr, err := tongo.ParseAddress(payload.Address)
	if err != nil {
		return errorx.Wrap(errors.New("invalid account"), errorx.Invalid)
	}

	vs := do.MustInvokeNamed[map[string]string](service.container, "envs")

	check, err := ton_utils.CheckProof(ctx, service.redisDB, addr.ID, user.ID, vs["TON_APP_DOMAIN"], payload.Nonce, parsed)
	if err != nil {
		log.Println(err)
		return errorx.Wrap(errors.New("proof checking error"), errorx.Invalid)
	}
	if !check {
		return errorx.Wrap(errors.New("invalid proof"), errorx.Invalid)
	}

	tonWallet := addr.ID.String()

	if userWallet == nil {
		userWallet = &models.UserWallet{
			ID:        user.ID,
			TONWallet: &tonWallet,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		_, err = datastore.CreateUserWallet(ctx, service.postgresDB, userWallet)
		if err != nil {
			return err
		}
	} else {
		userWallet.TONWallet = &tonWallet

		_, err = datastore.UpdateUserWallet(ctx, service.postgresDB, userWallet)
		if err != nil {
			return err
		}
	}

	err = service.cache.Delete(ctx, DBKeyUserWallet(user.ID))
	if err != nil {
		log.Println(err)
	}

	return service.ClearMeCache(ctx, user.ID)
}

func (service *ServiceUser) ClearUserCache(ctx context.Context, userID int64) error {
	err := service.cache.Delete(ctx, DBKeyMe(userID))
	if err != nil {
		log.Println(err)
	}

	err = service.cache.Delete(ctx, DBKeyUser(userID))
	if err != nil {
		log.Println(err)
	}

	return nil
}

func (service *ServiceUser) ClearMeCache(ctx context.Context, userID int64) error {
	err := service.cache.Delete(ctx, DBKeyMe(userID))
	if err != nil {
		log.Println(err)
	}

	return nil
}

func (service *ServiceUser) GetUserIdByRefCode(ctx context.Context, refCode string) (*models.User, error) {
	callback := func() (*models.User, error) {
		return datastore.GetUserByCustomRefCode(ctx, service.readonlyPostgresDB, refCode)
	}

	return caching.UseCacheWithRO(ctx, service.readonlyCache, service.cache, DBKeyUserByRefCode(refCode), CACHE_TTL_5_MINS, callback)
}
