package services

import (
	"context"
	"database/sql"
	"errors"
	"millionaire/internal/datastore"
	"millionaire/internal/models"
	"millionaire/internal/pkg/caching"
	"time"

	"github.com/go-redsync/redsync/v4"
	"github.com/hiendaovinh/toolkit/pkg/errorx"
	"github.com/redis/go-redis/v9"
	"github.com/samber/do"
	"github.com/uptrace/bun"
)

type ServiceUserGame struct {
	container          *do.Injector
	redisDB            redis.UniversalClient
	postgresDB         *bun.DB
	readonlyPostgresDB *bun.DB
	cache              caching.Cache
	rs                 *redsync.Redsync

	readonlyCache caching.ReadOnlyCache
}

func NewServiceUserGame(container *do.Injector) (*ServiceUserGame, error) {
	db, err := do.InvokeNamed[redis.UniversalClient](container, "redis-db")
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

	rs, err := do.Invoke[*redsync.Redsync](container)
	if err != nil {
		return nil, err
	}

	readonlyPostgresDB, err := do.InvokeNamed[*bun.DB](container, "db-readonly")
	if err != nil {
		return nil, err
	}

	readonlyCache, err := do.Invoke[caching.ReadOnlyCache](container)
	if err != nil {
		return nil, err
	}

	return &ServiceUserGame{container, db, postgresDB, readonlyPostgresDB, cache, rs, readonlyCache}, nil
}

func (service *ServiceUserGame) GetUserGame(ctx context.Context, user *models.User, game *models.Game) (*models.UserGame, error) {
	if game == nil {
		return nil, errors.New("game not found")
	}

	mutex := service.rs.NewMutex(LockKeyUserGame(game.Slug, user.ID))
	if err := mutex.Lock(); err != nil {
		return nil, errorx.Wrap(ErrUserGameLock, errorx.Invalid)
	}
	// nolint:errcheck
	defer mutex.Unlock()

	callback := func() (*models.UserGame, error) {
		return datastore.GetUserGame(ctx, service.readonlyPostgresDB, user.ID, game.Slug)
	}

	userGame, err := caching.UseCacheWithRO(ctx, service.readonlyCache, service.cache, DBKeyUserGame(game.Slug, user.ID), CACHE_TTL_5_MINS, callback)
	if err == sql.ErrNoRows {
		//create new user game
		userGame, err = service.AddNewUserGame(ctx, user.ID, game.Slug)
	} else if err != nil {
		return nil, err
	}

	if userGame != nil {
		userGame.GameStartTime = game.StartTime
		userGame.GameEndTime = game.EndTime
		return userGame, nil
	}

	return nil, err
}

func (service *ServiceUserGame) AddNewUserGame(ctx context.Context, userID int64, gameSlug string) (*models.UserGame, error) {
	now := time.Now()
	userGame := &models.UserGame{
		UserID:                userID,
		GameSlug:              gameSlug,
		ExtraSessions:         0,
		Countdown:             &now,
		CurrentSessionID:      nil,
		GiftPoints:            0,
		CurrentBonusMilestone: 0,
	}

	err := datastore.SetUserGame(ctx, service.postgresDB, userGame)
	if err != nil {
		return nil, err
	}

	userGame.IsNew = true

	return userGame, nil
}

func (service *ServiceUserGame) GetUserGameSessionSumary(ctx context.Context, userGame *models.UserGame) (*models.UserGameSessionSumary, error) {
	callback := func() (*models.UserGameSessionSumary, error) {
		return datastore.GetUserGameSessionSumary(ctx, service.readonlyPostgresDB, userGame.GameSlug, userGame.UserID)
	}

	return caching.UseCacheWithRO(ctx, service.readonlyCache, service.cache, DBKeyUserGameSessionSumary(userGame.GameSlug, userGame.UserID), CACHE_TTL_5_MINS, callback)
}

func (service *ServiceUserGame) GetUserGameSessionSumaryByGameSlug(ctx context.Context, gameSlug string, userId int64) (*models.UserGameSessionSumary, error) {
	callback := func() (*models.UserGameSessionSumary, error) {
		return datastore.GetUserGameSessionSumary(ctx, service.readonlyPostgresDB, gameSlug, userId)
	}

	return caching.UseCacheWithRO(ctx, service.readonlyCache, service.cache, DBKeyUserGameSessionSumary(gameSlug, userId), CACHE_TTL_5_MINS, callback)
}

func (service *ServiceUserGame) GetLastUserGameSession(ctx context.Context, userGame *models.UserGame) (*models.GameSession, error) {
	callback := func() (*models.GameSession, error) {
		return datastore.GetLastUserGameSession(ctx, service.readonlyPostgresDB, userGame.GameSlug, userGame.UserID)
	}

	return caching.UseCacheWithRO(ctx, service.readonlyCache, service.cache, DBKeyLastUserGameSession(userGame.GameSlug, userGame.UserID), CACHE_TTL_5_MINS, callback)
}

func (service *ServiceUserGame) ResetCountdown(ctx context.Context, userGame *models.UserGame, countdownDuration time.Duration) (*models.UserGame, error) {
	if userGame == nil {
		return nil, errors.New("userGame is nil")
	}
	now := time.Now()
	countdownTime := now.Add(countdownDuration)
	userGame.Countdown = &countdownTime

	_, err := datastore.UpdateUserGameCountdown(ctx, service.postgresDB, userGame)
	_ = service.cache.Delete(ctx, DBKeyUserGame(userGame.GameSlug, userGame.UserID))
	if err != nil {
		return nil, err
	}

	return userGame, nil
}

func (service *ServiceUserGame) ReduceCountdown(ctx context.Context, userGame *models.UserGame, reduceTime time.Duration) (*models.UserGame, error) {
	if userGame == nil {
		return nil, errors.New("userGame is nil")
	}
	//minus user cooldown
	if userGame.Countdown != nil {
		if time.Until(*userGame.Countdown) < reduceTime {
			now := time.Now()
			userGame.Countdown = &now
		} else {
			newCountDown := userGame.Countdown.Add(-reduceTime)
			userGame.Countdown = &newCountDown
		}
	}

	_, err := datastore.UpdateUserGameCountdown(ctx, service.postgresDB, userGame)
	_ = service.cache.Delete(ctx, DBKeyUserGame(userGame.GameSlug, userGame.UserID))

	return userGame, err
}

func (service *ServiceUserGame) UpdateCountdownAndExtraSession(ctx context.Context, userGame *models.UserGame, countdownTime time.Time, extraSession int) (*models.UserGame, error) {
	if userGame == nil {
		return nil, errors.New("userGame is nil")
	}

	_, err := datastore.UpdateCountdownAndExtraSession(ctx, service.postgresDB, userGame, countdownTime, extraSession)
	_ = service.cache.Delete(ctx, DBKeyUserGame(userGame.GameSlug, userGame.UserID))
	if err != nil {
		return nil, err
	}

	return userGame, nil
}

func (service *ServiceUserGame) UpdateUserBonusMilestone(ctx context.Context, userGame *models.UserGame) (*models.UserGame, error) {
	if userGame == nil {
		return nil, errors.New("userGame is nil")
	}

	_, err := datastore.UpdateUserBonusMilestone(ctx, service.postgresDB, userGame)
	_ = service.cache.Delete(ctx, DBKeyUserGame(userGame.GameSlug, userGame.UserID))
	if err != nil {
		return nil, err
	}

	return userGame, nil
}

func (service *ServiceUserGame) GetUserGameList(ctx context.Context, userID *models.User, games []models.Game) ([]*models.UserGame, error) {
	var userGames []*models.UserGame
	for _, game := range games {
		userGame, err := service.GetUserGame(ctx, userID, &game)
		if err != nil {
			return nil, err
		}

		userGames = append(userGames, userGame)
	}

	return userGames, nil
}
