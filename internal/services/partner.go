package services

import (
	"context"
	"database/sql"
	"errors"
	"millionaire/internal/datastore"
	"millionaire/internal/interfaces"
	"millionaire/internal/models"
	"millionaire/internal/pkg/caching"

	"github.com/go-redis/redis_rate/v10"
	"github.com/go-redsync/redsync/v4"
	"github.com/hiendaovinh/toolkit/pkg/errorx"
	"github.com/hiendaovinh/toolkit/pkg/limiter"
	"github.com/redis/go-redis/v9"
	"github.com/samber/do"
	"github.com/uptrace/bun"
)

type ServicePartner struct {
	container          *do.Injector
	redisDB            redis.UniversalClient
	rs                 *redsync.Redsync
	readonlyPostgresDB *bun.DB
	cache              caching.Cache
	readonlyCache      caching.ReadOnlyCache

	limiter interfaces.Limiter
}

func NewServicePartner(container *do.Injector) (*ServicePartner, error) {
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

	readonlyCache, err := do.Invoke[caching.ReadOnlyCache](container)
	if err != nil {
		return nil, err
	}

	readonlyPostgresDB, err := do.InvokeNamed[*bun.DB](container, "db-readonly")
	if err != nil {
		return nil, err
	}

	limiter, err := do.Invoke[interfaces.Limiter](container)
	if err != nil {
		return nil, err
	}

	return &ServicePartner{container, db, rs, readonlyPostgresDB, cache, readonlyCache, limiter}, nil
}

func (service *ServicePartner) ValidateAPIKey(apiKey string) (*models.Partner, error) {
	ctx := context.Background()
	callback := func() (*models.Partner, error) {
		return datastore.FindPartnerByAPIKey(ctx, service.readonlyPostgresDB, apiKey)
	}
	partner, err := caching.UseCacheWithRO(ctx, service.readonlyCache, service.cache, DBKeyPartner(apiKey), CACHE_TTL_15_MINS, callback)
	if err != nil {
		return nil, err
	}

	if partner == nil {
		return nil, errors.New("wrong api key")
	}

	return partner, nil
}

func (service *ServicePartner) GetPartner(ctx context.Context, slug string) (*models.Partner, error) {
	callback := func() (*models.Partner, error) {
		partner, err := datastore.GetPartner(ctx, service.readonlyPostgresDB, slug)
		if err != nil {
			return nil, err
		}
		return partner, nil
	}

	return caching.UseCacheWithRO(ctx, service.readonlyCache, service.cache, DBKeyPartner(slug), CACHE_TTL_5_MINS, callback)
}

func (service *ServicePartner) CheckJoinedUser(ctx context.Context, partner *models.Partner, userID string, refCode string, minGem int) (*models.PartnerResponse, error) {
	err := service.limiter.Allow(ctx, LimitKeyParner(partner.Slug), redis_rate.PerMinute(PARTNER_RATE_LIMIT_PER_MINUTE))
	if err != nil {
		if err.Error() == limiter.ErrRateLimited.Error() {
			return nil, errorx.Wrap(err, errorx.RateLimiting)
		}
		return nil, err
	}

	callback := func() (*models.PartnerResponse, error) {
		return service.getJoinedUserInfo(ctx, userID, refCode, minGem)
	}

	response, err := caching.UseCacheWithRO(ctx, service.readonlyCache, service.cache, DBKeyUserJoined(userID, refCode, minGem), CACHE_TTL_5_SECONDS, callback)

	if err == sql.ErrNoRows {
		return &models.PartnerResponse{User: false}, nil
	}

	return response, err
}

func (service *ServicePartner) getJoinedUserInfo(ctx context.Context, userID string, refCode string, minGem int) (*models.PartnerResponse, error) {
	var res = &models.PartnerResponse{}

	serviceUser, err := do.Invoke[*ServiceUser](service.container)
	if err != nil {
		return res, err
	}

	user, err := serviceUser.FindUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	res.User = (user != nil)

	if !res.User {
		return res, nil
	}

	if minGem >= 0 {
		userGem, err := serviceUser.GetUserGem(ctx, userID)
		if err == nil {
			res.Gem = userGem >= minGem
		}
	}

	if refCode != "-1" {
		user, err := serviceUser.FindUserByID(ctx, userID)
		if err == nil {
			res.Ref = (user.InviterID != nil) && (*user.InviterID == refCode)
		}
	}

	return res, nil
}
