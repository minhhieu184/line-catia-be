package services

import (
	"context"
	"fmt"
	"millionaire/internal/datastore/redis_store"
	"millionaire/internal/models"
	"millionaire/internal/pkg"
	"millionaire/internal/pkg/caching"

	"github.com/go-redsync/redsync/v4"
	"github.com/redis/go-redis/v9"
	"github.com/samber/do"
	"github.com/uptrace/bun"
)

type ServiceLeaderboard struct {
	container     *do.Injector
	redisDB       redis.UniversalClient
	redisDBCache  redis.UniversalClient
	rs            *redsync.Redsync
	postgresDB    *bun.DB
	cache         caching.Cache
	readonlyCache caching.ReadOnlyCache

	serviceUser   *ServiceUser
	serviceConfig *ServiceConfig
	serviceSocial *ServiceSocial
}

func NewServiceLeaderboard(container *do.Injector) (*ServiceLeaderboard, error) {
	db, err := do.InvokeNamed[redis.UniversalClient](container, "redis-db")
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

	cache, err := do.Invoke[caching.Cache](container)
	if err != nil {
		return nil, err
	}

	readonlyCache, err := do.Invoke[caching.ReadOnlyCache](container)
	if err != nil {
		return nil, err
	}

	serviceUser, err := do.Invoke[*ServiceUser](container)
	if err != nil {
		return nil, err
	}
	serviceConfig, err := do.Invoke[*ServiceConfig](container)
	if err != nil {
		return nil, err
	}

	serviceSocial, err := do.Invoke[*ServiceSocial](container)
	if err != nil {
		return nil, err
	}

	return &ServiceLeaderboard{container, db, dbRedisCache, rs, postgresDB, cache, readonlyCache, serviceUser, serviceConfig, serviceSocial}, nil
}

func (service *ServiceLeaderboard) GetTopReferralLeaderboard(ctx context.Context, user *models.User) (*models.LeaderboardResponse, error) {
	limit, _ := service.serviceConfig.GetIntConfig(ctx, CONFIG_REFERRAL_LEADERBOARD_LIMIT, REFERRAL_LEADERBOARD_DEFAULT_LIMIT)
	return service.getLeaderboard(ctx, user, LEADERBOARD_REFERRAL, limit)
}

func (service *ServiceLeaderboard) GetOverallLeaderboard(ctx context.Context, user *models.User) (*models.LeaderboardResponse, error) {
	limit, _ := service.serviceConfig.GetIntConfig(ctx, CONFIG_OVERALL_LEADERBOARD_LIMIT, OVERALL_LEADERBOARD_DEFAULT_LIMIT)
	return service.getLeaderboard(ctx, user, LEADERBOARD_OVERALL, limit)
}

func (service *ServiceLeaderboard) GetWeeklyOverallLeaderboard(ctx context.Context, user *models.User) (*models.LeaderboardResponse, error) {
	limit, _ := service.serviceConfig.GetIntConfig(ctx, CONFIG_OVERALL_LEADERBOARD_LIMIT, OVERALL_LEADERBOARD_DEFAULT_LIMIT)
	return service.getLeaderboard(ctx, user, LEADERBOARD_OVERALL_WEEKLY, limit)
}

func (service *ServiceLeaderboard) GetGameLeaderboard(ctx context.Context, gameID string, user *models.User) (*models.LeaderboardResponse, error) {
	numLeaderboard, _ := service.serviceConfig.GetIntConfig(ctx, CONFIG_GAME_LEADERBOARD_LIMIT, 50)

	return service.getLeaderboard(ctx, user, gameID, numLeaderboard)
}

func (service *ServiceLeaderboard) GetArenaLeaderboard(ctx context.Context, arenaSlug string, user *models.User) (*models.LeaderboardResponse, error) {
	limit, _ := service.serviceConfig.GetIntConfig(ctx, CONFIG_ARENA_LEADERBOARD_LIMIT, 50)
	return service.getLeaderboard(ctx, user, DBKeyArena(arenaSlug), limit)
}

func (service *ServiceLeaderboard) ClearLeaderboardCache(ctx context.Context, leaderboardName string) error {
	caching.DeleteKeys(ctx, service.redisDBCache, fmt.Sprintf("leaderboard_by_user:%s*", leaderboardName))
	return nil
}

func (service *ServiceLeaderboard) UpdateOverallLeaderboard(ctx context.Context, user *models.User) (*models.LeaderboardItem, error) {
	point, err := service.serviceUser.GetUserGemNoCache(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	leaderboard, err := redis_store.SetLeaderboard(ctx, service.redisDB, LEADERBOARD_OVERALL, &models.LeaderboardItem{
		UserId: user.ID,
		Score:  float64(point),
	})

	service.updateWeeklyOverallLeaderboard(ctx, user)

	//delete leaderboard cache
	service.ClearLeaderboardCache(ctx, LEADERBOARD_OVERALL)

	// NO need to clear inviter's friendlist cache here, 5 mins delay by caching is acceptable

	return leaderboard, err
}

func (service *ServiceLeaderboard) updateWeeklyOverallLeaderboard(ctx context.Context, user *models.User) (*models.LeaderboardItem, error) {
	thisWeek := pkg.GetFirstTimeOfCurrentWeek()
	point, err := service.serviceUser.GetUserGemFromTimeNoCache(ctx, user.ID, &thisWeek)
	if err != nil {
		return nil, err
	}

	leaderboard, err := redis_store.SetLeaderboard(ctx, service.redisDB, LEADERBOARD_OVERALL_WEEKLY, &models.LeaderboardItem{
		UserId: user.ID,
		Score:  float64(point),
	})

	service.ClearLeaderboardCache(ctx, LEADERBOARD_OVERALL_WEEKLY)

	return leaderboard, err
}

func (service *ServiceLeaderboard) getLeaderboard(ctx context.Context, user *models.User, leaderboardName string, limit int) (*models.LeaderboardResponse, error) {
	callback := func() (*models.LeaderboardResponse, error) {
		leaderboard, err := redis_store.GetLeaderboard(ctx, service.redisDB, leaderboardName, limit)
		if err != nil {
			return nil, err
		}

		rank, err := redis_store.GetRank(ctx, service.redisDB, leaderboardName, user)

		score := float64(0)
		if err == redis.Nil {
			rank = -1
		} else {
			score, err = redis_store.GetScore(ctx, service.redisDB, leaderboardName, user)
		}

		if err != nil && err != redis.Nil {
			return nil, err
		}

		for _, item := range leaderboard {
			// censor username
			u, _ := service.serviceUser.FindUserByID(ctx, item.UserId)
			if u != nil {
				if u.Username == "" {
					item.Username = censorUsername(fmt.Sprintf("%s %s", u.FirstName, u.LastName))
				} else {
					item.Username = censorUsername(u.Username)
				}

				if u.Avatar != nil {
					item.Avatar = u.Avatar
				}
			}
		}

		response := &models.LeaderboardResponse{
			Leaderboard: leaderboard,
			Me: &models.LeaderboardItem{
				Username: user.Username,
				UserId:   user.ID,
				Score:    float64(score),
				Rank:     int(rank + 1),
				Avatar:   user.Avatar,
			},
		}

		if user.Username == "" {
			response.Me.Username = fmt.Sprintf("%s %s", user.FirstName, user.LastName)
		}

		return response, nil
	}

	// TODO increase TTTL to optimize performance
	return caching.UseCacheWithRO(ctx, service.readonlyCache, service.cache, DBKeyLeaderboardByUser(leaderboardName, user.ID, limit), CACHE_TTL_1_MIN, callback)
}

func censorUsername(username string) string {
	// Get the first and last characters
	if len(username) < 3 {
		return username
	}
	firstTwo := username[:2]
	lastChar := string(username[len(username)-1])

	// Censor the middle characters
	middle := "*****"

	// Combine the censored username
	censoredUsername := firstTwo + middle + lastChar

	return censoredUsername
}
