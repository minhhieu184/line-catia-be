package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"millionaire/internal/datastore"
	"millionaire/internal/interfaces"
	"millionaire/internal/models"
	"millionaire/internal/pkg/caching"
	"net/http"
	"os"
	"regexp"

	"github.com/go-redis/redis_rate/v10"
	"github.com/hiendaovinh/toolkit/pkg/errorx"
	"github.com/hiendaovinh/toolkit/pkg/limiter"

	"github.com/uptrace/bun"

	"millionaire/internal/datastore/redis_store"

	"github.com/redis/go-redis/v9"
	"github.com/samber/do"
)

type ServiceSocial struct {
	*ServiceHTTP
	container     *do.Injector
	redisDB       redis.UniversalClient
	cache         caching.Cache
	readonlyCache caching.ReadOnlyCache

	db                 *bun.DB
	readonlyPostgresDB *bun.DB
	limiter            interfaces.Limiter
	baseURL            string
}

func NewServiceSocial(container *do.Injector) (*ServiceSocial, error) {
	db, err := do.InvokeNamed[redis.UniversalClient](container, "redis-db")
	if err != nil {
		return nil, err
	}

	cache, err := do.Invoke[caching.Cache](container)
	if err != nil {
		return nil, err
	}

	readOnlyCache, err := do.Invoke[caching.ReadOnlyCache](container)
	if err != nil {
		return nil, err
	}

	postgresDB, err := do.Invoke[*bun.DB](container)
	if err != nil {
		return nil, err
	}

	limiter, err := do.Invoke[interfaces.Limiter](container)
	if err != nil {
		return nil, err
	}

	readonlyPostgresDB, err := do.InvokeNamed[*bun.DB](container, "db-readonly")
	if err != nil {
		return nil, err
	}

	return &ServiceSocial{&ServiceHTTP{}, container, db, cache, readOnlyCache, postgresDB, readonlyPostgresDB, limiter, TELEGRAM_API_BASE_URL}, nil
}

func (service *ServiceSocial) GetTasks(ctx context.Context, gameSlug string) (*models.SocialTask, error) {
	callback := func() (*models.SocialTask, error) {
		return datastore.GetAvailableSocialTask(ctx, service.readonlyPostgresDB, gameSlug)
	}

	task, err := caching.UseCacheWithRO(ctx, service.readonlyCache, service.cache, DBKeyGameSocialTasks(gameSlug), CACHE_TTL_15_MINS, callback)
	if err != nil {
		return nil, err
	}

	return task, nil
}

func (service *ServiceSocial) GetUserTasks(ctx context.Context, userID int64, gameSlug string) (*models.SocialTask, error) {
	callback := func() (*models.SocialTask, error) {
		tasks, err := service.GetTasks(ctx, gameSlug)
		if err != nil {
			return nil, err
		}

		if tasks == nil {
			return nil, errorx.Wrap(errors.New("task not found"), errorx.NotExist)
		}
		for index, link := range tasks.Links {
			valid, _ := service.IsJoinSocial(ctx, userID, link.Url)
			link.Joined = valid
			tasks.Links[index] = link
		}

		return tasks, nil
	}

	return caching.UseCacheWithRO(ctx, service.readonlyCache, service.cache, DBKeyUserSocialTasks(userID, gameSlug), CACHE_TTL_1_DAY, callback)
}

func (service *ServiceSocial) GetAvailableSocialTasks(ctx context.Context) ([]models.SocialTask, error) {
	callback := func() ([]models.SocialTask, error) {
		return datastore.GetAvailableSocialTasks(ctx, service.readonlyPostgresDB)
	}

	return caching.UseCacheWithRO(ctx, service.readonlyCache, service.cache, DBKeyAllSocialTasks(), CACHE_TTL_5_MINS, callback)
}

func (service *ServiceSocial) GetAvailableSocialTasksByUser(ctx context.Context, userID int64) ([]models.SocialTask, error) {
	callback := func() ([]models.SocialTask, error) {
		tasks, err := service.GetAvailableSocialTasks(ctx)
		if err != nil {
			return nil, err
		}

		for index, task := range tasks {
			for linkIndex, link := range task.Links {
				valid, _ := service.IsJoinSocial(ctx, userID, link.Url)
				task.Links[linkIndex].Joined = valid
			}
			tasks[index] = task
		}

		return tasks, nil
	}
	return caching.UseCacheWithRO(ctx, service.readonlyCache, service.cache, DBKeyUserSocialTask(userID), CACHE_TTL_5_MINS, callback)
}

func (service *ServiceSocial) VerifySocialTask(ctx context.Context, user *models.User, gameSlug string, linkID int) (bool, error) {
	link, err := service.GetSocialLink(ctx, linkID, gameSlug)
	if err != nil {
		return false, err
	}
	callback := func() (bool, error) {
		joined, err := service.checkSocialLink(ctx, user, link)
		if err != nil {
			return false, err
		}

		if joined {
			serviceUser, err := do.Invoke[*ServiceUser](service.container)
			if err != nil {
				return joined, err
			}
			_, err = serviceUser.GetUserGemByAction(ctx, user.ID, fmt.Sprintf(KEY_SOCIAL_TASK, gameSlug, link.Url))
			if err != nil && err != sql.ErrNoRows {
				return joined, err
			}

			err = serviceUser.InsertUserGem(ctx, user, link.Gem, fmt.Sprintf(KEY_SOCIAL_TASK, gameSlug, link.Url))
			if err != nil {
				return joined, err
			}
			err = service.cache.Delete(ctx, DBKeyUserSocialTasks(user.ID, gameSlug))
			err = service.cache.Delete(ctx, DBKeyUserSocialTask(user.ID))

			serviceArena, err := do.Invoke[*ServiceArena](service.container)
			if err != nil || serviceArena == nil {
				return joined, nil
			}

			arena, _ := serviceArena.GetArenaByGameSlug(ctx, gameSlug)
			if arena != nil {
				_ = serviceArena.UpdateArenaLeaderboard(ctx, user, arena)
			}

			return joined, nil
		}
		return false, errorx.Wrap(errors.New("unverified"), errorx.Invalid)
	}

	return caching.UseCacheWithRO(ctx, service.readonlyCache, service.cache, DBKeyUserSocialTaskVerify(user.ID, gameSlug, link.Url), CACHE_TTL_15_MINS, callback)
}

func (service *ServiceSocial) checkSocialLink(ctx context.Context, user *models.User, link *models.Link) (bool, error) {
	joined := false

	var err error
	if link.LinkType == models.SocialTypeTelegramChannel || link.LinkType == models.SocialTypeTelegramGroup {
		joined, err = service.VerifyJoinTelegram(ctx, user.ID, link.Url)
	} else if link.LinkType == models.SocialTypeTwitter || link.LinkType == models.SocialTypeTelegramApp {
		joined, err = service.VerifySocialLinkWithoutChecking(ctx, user.ID, link.Url)
	} else if link.LinkType == models.SocialTypeTeletop {
		joined, err = service.VerifyAddedTeletop(ctx, user.ID, link.Url)
		fmt.Print(joined, err)
	} else {
		err = errorx.Wrap(errors.New("invalid link type"), errorx.Invalid)
	}

	return joined, err
}

func (service *ServiceSocial) VerifyJoinTelegram(ctx context.Context, userID int64, socialLink string) (bool, error) {
	verify, err := redis_store.GetJoinSocial(ctx, service.redisDB, userID, socialLink)
	if err == nil {
		return verify, nil
	}
	if err != nil && err != redis.Nil {
		return false, err
	}

	err = service.limiter.Allow(ctx, LimitKeyUserSocialTask(userID), redis_rate.PerMinute(TELEGRAM_TASK_RATE_LIMIT_PER_MINUTE))
	if err != nil {
		if err.Error() == limiter.ErrRateLimited.Error() {
			return false, errorx.Wrap(err, errorx.RateLimiting)
		}
		return false, err
	}

	matches := reTelegramLink.FindStringSubmatch(socialLink)
	if len(matches) != 6 {
		return false, errors.New("invalid subject")
	}
	slug := matches[5]

	telegramUserChannel, err := service.apiUserChannel(ctx, userID, slug)
	if err != nil || telegramUserChannel == nil {
		return false, fmt.Errorf("%w: unable to get user channel (%d: %s)", err, userID, slug)
	}

	if telegramUserChannel.Status != "member" &&
		telegramUserChannel.Status != "admin" &&
		telegramUserChannel.Status != "restricted" &&
		telegramUserChannel.Status != "creator" &&
		telegramUserChannel.Status != "administrator" {
		return false, nil
	}

	err = redis_store.SetJoinSocial(ctx, service.redisDB, userID, socialLink)
	if err != nil {
		return false, err
	}

	return true, nil
}

func (service *ServiceSocial) IsJoinSocial(ctx context.Context, userID int64, socialLink string) (bool, error) {
	return redis_store.GetJoinSocial(ctx, service.redisDB, userID, socialLink)
}

func (service *ServiceSocial) VerifySocialLinkWithoutChecking(ctx context.Context, userID int64, socialLink string) (bool, error) {
	joined, _ := service.IsJoinSocial(ctx, userID, socialLink)
	if joined {
		return true, nil
	}
	err := redis_store.SetJoinSocial(ctx, service.redisDB, userID, socialLink)
	if err != nil {
		return false, err
	}

	return true, err
}

func (service *ServiceSocial) GetSocialLink(ctx context.Context, linkID int, gameSlug string) (*models.Link, error) {
	callback := func() (*models.Link, error) {
		task, err := service.GetTasks(ctx, gameSlug)
		if err != nil {
			return nil, err
		}

		if task == nil {
			return nil, errorx.Wrap(errors.New("task not found"), errorx.NotExist)
		}

		if linkID < 0 || linkID >= len(task.Links) {
			return nil, errorx.Wrap(errors.New("invalid link id"), errorx.Invalid)
		}

		for _, link := range task.Links {
			if link.ID == linkID {
				return &link, nil
			}
		}

		return nil, errorx.Wrap(errors.New("link not found"), errorx.NotExist)
	}

	return caching.UseCacheWithRO(ctx, service.readonlyCache, service.cache, DBKeySocialLink(linkID, gameSlug), CACHE_TTL_5_MINS, callback)
}

func (service *ServiceSocial) apiUserChannel(ctx context.Context, userID int64, channel string) (*TelegramUserChannel, error) {
	resp, err := service.httpClient(0).Get(
		fmt.Sprintf("%s/bot%s/getChatMember?chat_id=@%s&user_id=%d", service.baseURL, os.Getenv("BOT_TOKEN"), channel, userID),
		http.Header{},
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var body TelegramUserChannelResp
	err = json.NewDecoder(resp.Body).Decode(&body)
	if err != nil {
		return nil, err
	}

	return body.Result, nil
}

func (service *ServiceSocial) VerifyAddedTeletop(ctx context.Context, userId int64, url string) (bool, error) {
	//getjoin social and set join social
	verify, err := redis_store.GetJoinSocial(ctx, service.redisDB, userId, url)
	if err == nil {
		return verify, nil
	}
	if err != nil && err != redis.Nil {
		return false, err
	}

	added, err := service.apiTeletopAddedTask(userId)
	if err != nil {
		return false, err
	}

	if added {
		err = redis_store.SetJoinSocial(ctx, service.redisDB, userId, url)
		if err != nil {
			return false, err
		}

		return true, nil
	}

	return false, nil
}

func (service *ServiceSocial) apiTeletopAddedTask(userId int64) (bool, error) {
	resp, err := service.httpClient(0).Get(
		fmt.Sprintf("https://api.teletop.xyz/users/%d/verify/%d", userId, TELETOP_CATIA_APP_ID),
		http.Header{},
	)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	var body TeletopResponse
	err = json.NewDecoder(resp.Body).Decode(&body)
	if err != nil {
		return false, err
	}

	return body.Success, nil
}

var reTelegramLink = regexp.MustCompile(`^(?:|(https?:\/\/)?(|www)[.]?((t|telegram)\.me)\/)([a-zA-Z0-9_+-]+)$`)

type TelegramRespError struct {
	OK          bool   `json:"ok"`
	ErrorCode   int    `json:"error_code"`
	Description string `json:"description"`
}

type TelegramUserChannelResp struct {
	*TelegramRespError
	OK     bool                 `json:"ok"`
	Result *TelegramUserChannel `json:"result"`
}

type TelegramUserChannel struct {
	User struct {
		ID           int    `json:"id"`
		IsBot        bool   `json:"is_bot"`
		FirstName    string `json:"first_name"`
		Username     string `json:"username"`
		LanguageCode string `json:"language_code"`
	} `json:"user"`
	Status string `json:"status"`
}

type TeletopResponse struct {
	Success bool `json:"success"`
}
