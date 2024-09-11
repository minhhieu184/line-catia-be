package services

import (
	"context"
	"database/sql"
	"millionaire/internal/datastore"
	"millionaire/internal/models"
	"millionaire/internal/pkg/caching"

	"github.com/go-redsync/redsync/v4"
	"github.com/redis/go-redis/v9"
	"github.com/samber/do"
	"github.com/uptrace/bun"
)

type ServiceReward struct {
	container          *do.Injector
	redisDB            redis.UniversalClient
	redisDBCache       redis.UniversalClient
	rs                 *redsync.Redsync
	postgresDB         *bun.DB
	readonlyPostgresDB *bun.DB
	cache              caching.Cache
	readonlyCache      caching.ReadOnlyCache
}

func NewServiceReward(container *do.Injector) (*ServiceReward, error) {
	dbRedis, err := do.InvokeNamed[redis.UniversalClient](container, "redis-db")
	if err != nil {
		return nil, err
	}

	dbRedisCache, err := do.InvokeNamed[redis.UniversalClient](container, "redis-cache")
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

	readonlyPostgresDB, err := do.InvokeNamed[*bun.DB](container, "db-readonly")
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

	return &ServiceReward{container, dbRedis, dbRedisCache, rs, postgresDB, readonlyPostgresDB, cache, readonlyCache}, nil
}

func (service *ServiceReward) GetAvailableRewardByUserID(ctx context.Context, userID int64) ([]models.Reward, error) {
	callback := func() ([]models.Reward, error) {
		rewards, err := datastore.GetAvaiableRewardByUserID(context.Background(), service.postgresDB, userID)
		if err == sql.ErrNoRows {
			return nil, nil
		}

		return rewards, err
	}

	return caching.UseCacheWithRO(ctx, service.readonlyCache, service.cache, DBKeyUserAvailableReward(userID), CACHE_TTL_5_MINS, callback)
}

func (service *ServiceReward) ClaimReward(ctx context.Context, rewardID int) error {
	// TODO: add lock
	// TODO: check ownership of reward
	// TODO: update user assets
	return datastore.ClaimReward(ctx, service.postgresDB, rewardID)
}

func (service *ServiceReward) ClearUserAvailableRewardCache(ctx context.Context, userID int64) error {
	return service.cache.Delete(ctx, DBKeyUserAvailableReward(userID))
}
