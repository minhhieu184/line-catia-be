package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"millionaire/internal/datastore"
	"millionaire/internal/datastore/redis_store"
	"millionaire/internal/models"
	"millionaire/internal/pkg/caching"
	"time"

	"github.com/go-redsync/redsync/v4"
	"github.com/redis/go-redis/v9"
	"github.com/samber/do"
	"github.com/uptrace/bun"
)

const (
	GEM_AMOUNT      = 5
	LIFELINE_AMOUNT = 1
	STAR_AMOUNT     = 1
)

type ServiceUserFreebies struct {
	container          *do.Injector
	redisDB            redis.UniversalClient
	postgresDB         *bun.DB
	readonlyPostgresDB *bun.DB
	cache              caching.Cache
	rs                 *redsync.Redsync
	readonlyCache      caching.ReadOnlyCache

	serviceConfig *ServiceConfig
}

func NewServiceUserFreebies(container *do.Injector) (*ServiceUserFreebies, error) {
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

	serviceConfig, err := do.Invoke[*ServiceConfig](container)
	if err != nil {
		return nil, err
	}

	readonlyCache, err := do.Invoke[caching.ReadOnlyCache](container)
	if err != nil {
		return nil, err
	}

	return &ServiceUserFreebies{container, db, postgresDB, readonlyPostgresDB, cache, rs, readonlyCache, serviceConfig}, nil
}

func (service *ServiceUserFreebies) GetOrNewUserFreebie(ctx context.Context, name string, action string, userID string, icon string, amount int) (*models.UserFreebie, error) {
	freebie, err := service.GetUserFreebie(ctx, userID, action)
	if err != nil && err != redis.Nil && err != sql.ErrNoRows {
		return nil, err
	}

	if freebie == nil || err == redis.Nil || err == sql.ErrNoRows {
		freebie := &models.UserFreebie{
			UserID:    userID,
			Name:      name,
			Countdown: time.Now(),
			Action:    action,
			Icon:      icon,
			Amount:    amount,
		}
		err := datastore.InsertUserFreebies(ctx, service.postgresDB, freebie)
		if err != nil {
			return nil, err
		}
	}

	return freebie, nil
}

func (service *ServiceUserFreebies) GetOrNewUserFreebies(ctx context.Context, userID string) ([]*models.UserFreebie, error) {
	freebies, err := service.GetAllUserFreebies(ctx, userID)
	if err != nil && err != redis.Nil && err != sql.ErrNoRows {
		return nil, err
	}

	if len(freebies) < len(models.Freebies) {
		newFreebies, err := datastore.InsertMultipleUserFreebies(ctx, service.postgresDB, userID)
		if err != nil {
			return nil, err
		}

		err = service.cache.Delete(ctx, DBKeyUserAllFreebies(userID))

		freebies = append(freebies, newFreebies...)
	}

	return freebies, nil
}

func (service *ServiceUserFreebies) GetUserFreebie(ctx context.Context, userID string, action string) (*models.UserFreebie, error) {
	callback := func() (*models.UserFreebie, error) {
		return datastore.GetUserFreebies(ctx, service.readonlyPostgresDB, userID, action)
	}

	return caching.UseCacheWithRO(ctx, service.readonlyCache, service.cache, DBKeyUserFreebies(userID, action), CACHE_TTL_5_MINS, callback)
}

func (service *ServiceUserFreebies) GetAllUserFreebies(ctx context.Context, userID string) ([]*models.UserFreebie, error) {
	callback := func() ([]*models.UserFreebie, error) {
		return datastore.GetAllUserFreebies(ctx, service.readonlyPostgresDB, userID)
	}

	return caching.UseCacheWithRO(ctx, service.readonlyCache, service.cache, DBKeyUserAllFreebies(userID), CACHE_TTL_5_MINS, callback)
}

func (service *ServiceUserFreebies) ClaimFreebies(ctx context.Context, user *models.User, action string) error {
	userFreebie, err := service.GetUserFreebie(ctx, user.ID, action)
	if err != nil {
		return err
	}

	b, _ := json.MarshalIndent(userFreebie, "", "    ")
	println("userFreebie", string(b))

	if userFreebie.Action == action {
		println("userFreebie 1")
		if userFreebie.Countdown.After(time.Now()) {
			println("userFreebie 2")
			return errors.New("freebie is not available to claim now")
		}

		serviceUser, err := do.Invoke[*ServiceUser](service.container)
		if err != nil {
			return err
		}

		println("userFreebie 3")

		if action == models.ACTION_CLAIM_GEM {
			println("userFreebie 4")
			if userFreebie.Amount == 0 {
				userFreebie.Amount = GEM_AMOUNT
			}
			println("userFreebie 5")
			serviceUser.InsertUserGem(ctx, user, userFreebie.Amount, fmt.Sprintf("freebies:%s:%s", models.ACTION_CLAIM_GEM, time.Now().Format("2006-01-02T15:04:05")))
			println("userFreebie 6")
			timeGem, err := service.serviceConfig.GetIntConfig(ctx, CONFIG_FREEBIE_GEM_COUNTDOWN, 5)
			if err != nil {
				println("err xxx", err.Error())
				return err
			}
			println("userFreebie 7", timeGem)
			userFreebie.Countdown = time.Now().Add(time.Duration(timeGem) * time.Minute)
		}

		if action == models.ACTION_CLAIM_STAR {
			if userFreebie.Amount == 0 {
				userFreebie.Amount = STAR_AMOUNT
			}

			var userStar models.UserBoost
			userStar.UserID = user.ID
			userStar.CreatedAt = time.Now()
			userStar.Source = fmt.Sprintf("freebies:%s:%v", models.ACTION_CLAIM_STAR, time.Now().Format("2006-01-02T15:04:05"))
			userStar.Validated = true

			err = serviceUser.CreateBoost(ctx, &userStar)
			if err != nil {
				return err
			}

			timeStar, err := service.serviceConfig.GetIntConfig(ctx, CONFIG_FREEBIE_STAR_COUNTDOWN, 5)
			if err != nil {
				return err
			}

			userFreebie.Countdown = time.Now().Add(time.Duration(timeStar) * time.Minute)

			err = serviceUser.ClearUserCache(ctx, user.ID)
			if err != nil {
				return err
			}
		}

		if action == models.ACTION_CLAIM_LIFELINE {
			if userFreebie.Amount == 0 {
				userFreebie.Amount = LIFELINE_AMOUNT
			}

			err = serviceUser.ChangeLifelineBalance(ctx, user, fmt.Sprintf("freebies:%s:%s", models.ACTION_CLAIM_LIFELINE, time.Now().Format("2006-01-02T15:04:05")), userFreebie.Amount)
			if err != nil {
				return err
			}

			timeLifeline, err := service.serviceConfig.GetIntConfig(ctx, CONFIG_FREEBIE_LIFELINE_COUNTDOWN, 5)
			if err != nil {
				return err
			}

			userFreebie.Countdown = time.Now().Add(time.Duration(timeLifeline) * time.Minute)
		}

		err = datastore.UpdateUserFreebies(ctx, service.postgresDB, userFreebie)
		if err != nil {
			println("userFreebie 8")
			return err
		}

		//delete cache
		err = service.cache.Delete(ctx, DBKeyUserFreebies(user.ID, action))
		if err != nil {
			println("userFreebie 9")
			log.Println(err)
		}

		err = service.cache.Delete(ctx, DBKeyUserAllFreebies(user.ID))
		if err != nil {
			println("userFreebie 10")
			log.Println(err)
		}
	}
	println("userFreebie 11")

	return nil
}

func (service *ServiceUserFreebies) GetLatestMessage(ctx context.Context) (*models.LastMessage, error) {
	message, err := redis_store.GetLastMessage(ctx, service.redisDB)
	if err != nil {
		return nil, err
	}

	return message, nil
}
