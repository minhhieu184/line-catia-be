package api

import (
	"context"

	"github.com/go-redis/redis_rate/v10"
)

type Limiter interface {
	AllowUser(ctx context.Context, key string, limit redis_rate.Limit) error
	Allow(ctx context.Context, key string, limit redis_rate.Limit) error
}
