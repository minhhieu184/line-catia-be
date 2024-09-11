package services

import (
	"context"
	"errors"
	"millionaire/internal/datastore"
	"millionaire/internal/datastore/redis_store"
	"millionaire/internal/models"
	"millionaire/internal/pkg/caching"

	"github.com/go-redsync/redsync/v4"
	"github.com/redis/go-redis/v9"
	"github.com/samber/do"
	"github.com/uptrace/bun"
)

type ServiceArena struct {
	container          *do.Injector
	redisDB            redis.UniversalClient
	rs                 *redsync.Redsync
	readonlyPostgresDB *bun.DB
	cache              caching.Cache
	readonlyCache      caching.ReadOnlyCache
}

func NewServiceArena(container *do.Injector) (*ServiceArena, error) {
	db, err := do.InvokeNamed[redis.UniversalClient](container, "redis-db")
	if err != nil {
		return nil, err
	}

	rs, err := do.Invoke[*redsync.Redsync](container)
	if err != nil {
		return nil, err
	}

	cache, err := do.Invoke[caching.Cache](container)
	if err != nil {
		return nil, err
	}

	readonlyPostgresDB, err := do.InvokeNamed[*bun.DB](container, "db-readonly")
	if err != nil {
		return nil, err
	}

	readOnlyCache, err := do.Invoke[caching.ReadOnlyCache](container)
	if err != nil {
		return nil, err
	}

	return &ServiceArena{
		container:          container,
		redisDB:            db,
		rs:                 rs,
		readonlyPostgresDB: readonlyPostgresDB,
		cache:              cache,
		readonlyCache:      readOnlyCache,
	}, nil
}

func (service *ServiceArena) GetEnabledArenas(ctx context.Context) ([]models.Arena, error) {
	callback := func() ([]models.Arena, error) {
		return datastore.GetEnabledArenas(ctx, service.readonlyPostgresDB)
	}

	return caching.UseCacheWithRO(ctx, service.readonlyCache, service.cache, DBKeyArenaList(), CACHE_TTL_5_MINS, callback)
}

func (service *ServiceArena) GetArena(ctx context.Context, slug string) (*models.Arena, error) {
	callback := func() (*models.Arena, error) {
		arena, err := datastore.GetArenaBySlug(ctx, service.readonlyPostgresDB, slug)
		if err != nil {
			return nil, err
		}

		paticipantsCount := int64(0)

		switch slug {
		case "seed":
			paticipantsCount, _ = redis_store.GetLeaderboardPaticipantsCount(ctx, service.redisDB, DBKeyArena(slug))
			paticipantsCount += 100000
		case "catia":
			paticipantsCount = 30000
		default:
			paticipantsCount, _ = redis_store.GetLeaderboardPaticipantsCount(ctx, service.redisDB, DBKeyArena(slug))
		}

		arena.PaticipantsCount = paticipantsCount
		return arena, nil
	}

	return caching.UseCacheWithRO(ctx, service.readonlyCache, service.cache, DBKeyArena(slug), CACHE_TTL_5_MINS, callback)
}

func (service *ServiceArena) GetArenaByGameSlug(ctx context.Context, gameSlug string) (*models.Arena, error) {
	callback := func() (*models.Arena, error) {
		return datastore.GetArenaByGameSlug(ctx, service.readonlyPostgresDB, gameSlug)
	}

	return caching.UseCacheWithRO(ctx, service.readonlyCache, service.cache, DBKeyArenaByGameSlug(gameSlug), CACHE_TTL_5_MINS, callback)
}

func (service *ServiceArena) UpdateArenaLeaderboard(ctx context.Context, user *models.User, arena *models.Arena) error {
	if arena == nil {
		return errors.New("arena not found")
	}

	if user == nil {
		return errors.New("user not found")
	}

	if arena.IsEnded() {
		return errors.New("arena is ended")
	}

	serviceUserGame, err := do.Invoke[*ServiceUserGame](service.container)
	if err != nil {
		return err
	}

	sessionSumary, err := serviceUserGame.GetUserGameSessionSumaryByGameSlug(ctx, arena.GameSlug, user.ID)
	if err != nil {
		return err
	}

	totalGem := sessionSumary.TotalScore

	serviceSocial, err := do.Invoke[*ServiceSocial](service.container)
	if err != nil {
		return err
	}

	socialTask, _ := serviceSocial.GetUserTasks(ctx, user.ID, arena.GameSlug)
	if socialTask != nil {
		for _, link := range socialTask.Links {
			//if task is not completed, return error
			if link.Joined {
				totalGem += link.Gem
			}
		}
	}

	_, err = redis_store.SetLeaderboard(ctx, service.redisDB, DBKeyArena(arena.Slug), &models.LeaderboardItem{
		UserId: user.ID,
		Score:  float64(totalGem),
	})

	if err != nil {
		return err
	}

	serviceLeaderboard, err := do.Invoke[*ServiceLeaderboard](service.container)

	if err != nil {
		return nil
	}

	return serviceLeaderboard.ClearLeaderboardCache(ctx, DBKeyArena(arena.Slug))
}
