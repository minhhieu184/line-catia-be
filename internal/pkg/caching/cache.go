package caching

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-redis/cache/v9"
	"github.com/redis/go-redis/v9"
)

type ReadOnlyCache interface {
	Get(ctx context.Context, key string, target any) error
}

type Cache interface {
	ReadOnlyCache
	Set(ctx context.Context, key string, value any, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}

func UseCache[T any](ctx context.Context, cash Cache, key string, ttl time.Duration, callback func() (T, error)) (T, error) {
	var v T
	err := cash.Get(ctx, key, &v)
	if !errors.Is(err, cache.ErrCacheMiss) {
		return v, err
	}

	v, err = callback()
	if err != nil {
		return v, err
	}

	// fire and forget
	//nolint:errcheck
	cash.Set(ctx, key, v, ttl)
	return v, nil
}

func UseCacheWithRO[T any](ctx context.Context, roCash ReadOnlyCache, cash Cache, key string, ttl time.Duration, callback func() (T, error)) (T, error) {
	var v T
	err := roCash.Get(ctx, key, &v)
	if !errors.Is(err, cache.ErrCacheMiss) {
		return v, err
	}

	v, err = callback()
	if err != nil {
		return v, err
	}

	// fire and forget
	//nolint:errcheck
	cash.Set(ctx, key, v, ttl)
	return v, nil
}

type CacheRedis struct {
	instance *cache.Cache
}

func (c *CacheRedis) Get(ctx context.Context, key string, target any) error {
	return c.instance.Get(ctx, key, target)
}

func (c *CacheRedis) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	return c.instance.Set(&cache.Item{
		Ctx:   ctx,
		Key:   key,
		Value: value,
		TTL:   ttl,
	})
}

func (c *CacheRedis) Delete(ctx context.Context, key string) error {
	return c.instance.Delete(ctx, key)
}

func NewCacheRedis(client redis.UniversalClient, withLocalCache bool) (*CacheRedis, error) {
	var localCache cache.LocalCache
	if withLocalCache {
		localCache = cache.NewTinyLFU(10000, time.Minute)
	}
	return &CacheRedis{cache.New(&cache.Options{
		Redis:      client,
		LocalCache: localCache,
	})}, nil
}

type RedisClient interface {
	Keys(ctx context.Context, pattern string) *redis.StringSliceCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
}

func DeleteKeys(ctx context.Context, client redis.UniversalClient, pattern string) error {
	clusterClient, ok := client.(*redis.ClusterClient)
	if ok {
		_ = clusterClient.ForEachMaster(ctx, func(ctx context.Context, c *redis.Client) error {
			deleteKeys(ctx, c, pattern)
			return nil
		})
	} else {
		deleteKeys(ctx, client, pattern)
	}

	return nil
}

func deleteKeys(ctx context.Context, client RedisClient, pattern string) {
	keys := client.Keys(ctx, pattern).Val()
	for _, key := range keys {
		err := client.Del(ctx, key).Err()
		fmt.Println("Deleted key", key, err)
	}
}
