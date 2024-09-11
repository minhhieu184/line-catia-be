package interfaces

import (
	"context"

	"github.com/go-redis/redis_rate/v10"
)

type Limiter interface {
	Allow(ctx context.Context, key string, limit redis_rate.Limit) error
}
