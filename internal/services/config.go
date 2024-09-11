package services

import (
	"context"
	"strconv"

	"millionaire/internal/datastore"
	"millionaire/internal/pkg/caching"

	"github.com/go-redsync/redsync/v4"
	"github.com/redis/go-redis/v9"
	"github.com/samber/do"
	"github.com/uptrace/bun"
)

type ServiceConfig struct {
	container          *do.Injector
	redisDB            redis.UniversalClient
	rs                 *redsync.Redsync
	readonlyPostgresDB *bun.DB
	cache              caching.Cache
	readonlyCache      caching.ReadOnlyCache
}

func NewServiceConfig(container *do.Injector) (*ServiceConfig, error) {
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

	return &ServiceConfig{container, db, rs, readonlyPostgresDB, cache, readOnlyCache}, nil
}

func (service *ServiceConfig) GetStringConfig(ctx context.Context, key string, defaultValue string) (string, error) {
	callback := func() (string, error) {
		config, err := datastore.GetConfigByKey(ctx, service.readonlyPostgresDB, key)
		if err != nil {
			return defaultValue, err
		}
		return config.Value, nil
	}

	value, err := caching.UseCacheWithRO(ctx, service.readonlyCache, service.cache, DBKeyConfig(key), CACHE_TTL_5_MINS, callback)
	if err != nil {
		return defaultValue, err
	}

	return value, nil
}

func (service *ServiceConfig) GetIntConfig(ctx context.Context, key string, defaultValue int) (int, error) {
	callback := func() (int, error) {
		config, err := datastore.GetConfigByKey(ctx, service.readonlyPostgresDB, key)
		if err != nil {
			return defaultValue, err
		}

		intValue, err := strconv.Atoi(config.Value)
		if err != nil {
			return defaultValue, err
		}

		return intValue, nil
	}

	value, err := caching.UseCacheWithRO(ctx, service.readonlyCache, service.cache, DBKeyConfig(key), CACHE_TTL_5_MINS, callback)
	if err != nil {
		return defaultValue, err
	}

	return value, nil
}

// func (service *ServiceConfig) CreateConfig(ctx context.Context, key string, value string) (*models.Config, error) {
// 	config, _ := redis_store.GetConfigRedis(ctx, service.redisDB, key)
// 	if config != nil {
// 		return config, nil
// 	}

// 	config = &models.Config{
// 		Key:   key,
// 		Value: value,
// 	}
// 	err := datastore.InsertConfig(ctx, *service.postgresDB, *config)
// 	if err != nil {
// 		return nil, err
// 	}

// 	err = redis_store.SetConfigRedis(ctx, service.redisDB, config)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return config, nil
// }

// func (service *ServiceConfig) EditConfig(ctx context.Context, key string, value string) (*models.Config, error) {
// 	config, err := datastore.GetConfigByKey(ctx, *service.postgresDB, key)
// 	if err != nil {
// 		return nil, err
// 	}

// 	config.Value = value
// 	_, err = datastore.EditConfig(ctx, *service.postgresDB, config)
// 	if err != nil {
// 		return nil, err
// 	}

// 	err = redis_store.SetConfigRedis(ctx, service.redisDB, config)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return config, nil
// }
