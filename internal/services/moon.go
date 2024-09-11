package services

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"millionaire/internal/datastore/redis_store"
	"millionaire/internal/models"
	"millionaire/internal/pkg/caching"
	"time"

	"github.com/go-redsync/redsync/v4"
	"github.com/hiendaovinh/toolkit/pkg/errorx"
	"github.com/mroth/weightedrand/v2"
	"github.com/redis/go-redis/v9"
	"github.com/samber/do"
	"github.com/uptrace/bun"
)

type ServiceMoon struct {
	container          *do.Injector
	redisDB            redis.UniversalClient
	rs                 *redsync.Redsync
	readonlyPostgresDB *bun.DB
	cache              caching.Cache

	serviceConfig *ServiceConfig
	serviceGacha  *ServiceGacha[int]

	extraSetups []models.ExtraSetup
}

func NewServiceMoon(container *do.Injector) (*ServiceMoon, error) {
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

	serviceConfig, err := do.Invoke[*ServiceConfig](container)
	if err != nil {
		return nil, err
	}

	extraSetups := []models.ExtraSetup{
		{
			Type:        models.ExtraSetupTypeNothing.String(),
			Chance:      50,
			Description: "Good luck",
		},
		{
			Type:        models.ExtraSetupType1Gem.String(),
			Chance:      20,
			Description: "Receive 1 gem",
		},
		{
			Type:        models.ExtraSetupType3Gem.String(),
			Chance:      10,
			Description: "Receive 3 gems",
		},
		{
			Type:        models.ExtraSetupType5Gem.String(),
			Chance:      7,
			Description: "Receive 5 gems",
		},
		{
			Type:        models.ExtraSetupType10Gem.String(),
			Chance:      5,
			Description: "Receive 10 gems",
		},
		{
			Type:        models.ExtraSetupType1Lifeline.String(),
			Chance:      5,
			Description: "Receive 1 lifeline",
		},
		{
			Type:        models.ExtraSetupType2Lifeline.String(),
			Chance:      2,
			Description: "Receive 2 lifelines",
		},
		{
			Type:        models.ExtraSetupType1Star.String(),
			Chance:      2,
			Description: "Receive 1 star",
		},
		{
			Type:        models.ExtraSetupType2Star.String(),
			Chance:      1,
			Description: "Receive 2 stars",
		},
	}

	choices := []weightedrand.Choice[int, int]{}
	for i, v := range extraSetups {
		choices = append(choices, weightedrand.NewChoice(i, v.Chance))
	}

	serviceGacha, err := NewServiceGacha[int](choices)

	return &ServiceMoon{container, db, rs, readonlyPostgresDB, cache, serviceConfig, serviceGacha, extraSetups}, nil
}

func (service *ServiceMoon) GetUserMoon(ctx context.Context, user *models.User) (*models.UserMoon, error) {
	moon, err := service.GetMoon(ctx)

	if err != nil {
		return nil, err
	}

	spinned, err := service.CheckSpin(ctx, user, moon)

	if err != nil {
		return nil, err
	}

	return &models.UserMoon{
		Moon:    *moon,
		Claimed: spinned,
	}, nil
}

func (service *ServiceMoon) GetMoon(ctx context.Context) (*models.Moon, error) {
	moon, err := redis_store.GetMoon(ctx, service.redisDB)
	if err != nil && err != redis.Nil {
		fmt.Println("GetMoon", err)
		return nil, err
	}

	timePerRangeInMinutes, _ := service.serviceConfig.GetIntConfig(ctx, CONFIG_MOON_TIME_PER_RANGE_IN_MINUTES, 180)
	timeFrameDuration := time.Duration(timePerRangeInMinutes) * time.Minute
	randomUnitInMinutes, _ := service.serviceConfig.GetIntConfig(ctx, CONFIG_MOON_RANDOM_UNIT_IN_MINUTES, 15)
	randomUnit := time.Duration(randomUnitInMinutes) * time.Minute
	currentTimeFrame := time.Now().UTC().Truncate(timeFrameDuration)
	fmt.Println("currentTimeFrame", currentTimeFrame, timeFrameDuration)

	if moon != nil && moon.CurrentTimeFrame.Equal(currentTimeFrame) {
		return moon, nil
	}

	// moon does not exist or expired

	mutex := service.rs.NewMutex(LockKeyFullMoon())
	if err := mutex.Lock(); err != nil {
		return nil, errorx.Wrap(ErrFullMoonLock, errorx.Invalid)
	}
	// nolint:errcheck
	defer mutex.Unlock()

	livingTimeInMinutes, _ := service.serviceConfig.GetIntConfig(ctx, CONFIG_MOON_EXPIRED_TIME_IN_MINUTES, 4)
	timeLines := service.genCurrentAndNextAppearanceTime(ctx, currentTimeFrame, timeFrameDuration, randomUnit, livingTimeInMinutes)
	fmt.Printf("timeLines: %+v\n", timeLines)

	moon = &models.Moon{
		NumberOfTaps:     500,
		ExpiredAt:        timeLines[0].Add(time.Duration(livingTimeInMinutes) * time.Minute),
		CurrentFullMoon:  timeLines[0],
		NextFullMoon:     timeLines[1],
		CurrentTimeFrame: currentTimeFrame,
	}

	err = redis_store.SetMoon(ctx, service.redisDB, moon)
	return moon, err
}

func (service *ServiceMoon) SpinGacha(ctx context.Context, user *models.User) (*models.Gift, error) {
	mutex := service.rs.NewMutex(LockKeyUserMoon(user.ID))
	if err := mutex.TryLock(); err != nil {
		return nil, errorx.Wrap(ErrGameSessionLock, errorx.Invalid)
	}
	// nolint:errcheck
	defer mutex.Unlock()

	moon, err := service.GetMoon(ctx)
	if err != nil {
		return nil, err
	}

	userSpin, err := redis_store.GetUserMoonGacha(ctx, service.redisDB, user.ID, moon.CurrentTimeFrame)
	if err != nil && err != redis.Nil {
		return nil, err
	}

	if userSpin {
		return nil, errorx.Wrap(errors.New("prize already claimed"), errorx.Validation)
	}

	now := time.Now()

	if now.Before(moon.CurrentFullMoon) || now.After(moon.ExpiredAt) {
		return nil, errorx.Wrap(errors.New("the moonlight time is over"), errorx.Validation)
	}

	prizeIndex := service.serviceGacha.Pick()
	prize := models.ExtraSetup{
		Type: models.ExtraSetupTypeNothing.String(),
	}
	if prizeIndex >= 0 && prizeIndex < len(service.extraSetups) {
		prize = service.extraSetups[prizeIndex]
	}
	prizeType := models.ToExtraSetupType(prize.Type)

	if !prizeType.Valid() {
		return nil, errors.New("invalid prize type")
	}

	gift := prizeType.ToGift()
	if gift == nil {
		return nil, errors.New("invalid gift")
	}

	serviceUser, err := do.Invoke[*ServiceUser](service.container)
	if err != nil {
		return nil, err
	}

	if gift.Type == models.GiftTypeGem {
		err = serviceUser.InsertUserGem(ctx, user, gift.Amout, fmt.Sprintf("moon_gacha:gem:%s", time.Now().Format("2006-01-02T15:04:05")))
		if err != nil {
			return nil, err
		}
	} else if gift.Type == models.GiftTypeStar {
		err = serviceUser.InsertBoosts(ctx, user, fmt.Sprintf("moon_gacha:boost:%s", time.Now().Format("2006-01-02T15:04:05")), gift.Amout)
		if err != nil {
			return nil, err
		}
	} else if gift.Type == models.GiftTypeLifeline {
		err = serviceUser.ChangeLifelineBalance(ctx, user, fmt.Sprintf("moon_gacha:lifeline:%s", time.Now().Format("2006-01-02T15:04:05")), gift.Amout)
		if err != nil {
			return nil, err
		}
	}

	err = redis_store.SetUserMoonGacha(ctx, service.redisDB, user.ID, moon.CurrentTimeFrame)
	if err != nil {
		return nil, err
	}

	return gift, nil
}

func (service *ServiceMoon) CheckSpin(ctx context.Context, user *models.User, moon *models.Moon) (bool, error) {
	spinned, err := redis_store.GetUserMoonGacha(ctx, service.redisDB, user.ID, moon.CurrentTimeFrame)
	if err != nil && err != redis.Nil {
		return false, err
	}

	return spinned, nil
}

func (service *ServiceMoon) genCurrentAndNextAppearanceTime(ctx context.Context, dateNow time.Time, duration time.Duration, randomTimeFrame time.Duration, livingTimeInMinutes int) []time.Time {
	wdRand := rand.New(rand.NewSource(dateNow.Unix()))
	numberRand := wdRand.Intn(int(duration / randomTimeFrame)) // get random hour
	fmt.Println("numberRand 1:", numberRand, duration.Hours())
	out := []time.Time{
		dateNow.Add(time.Duration(numberRand) * randomTimeFrame),
	}
	nextTime := dateNow.Add(duration)
	wdRand = rand.New(rand.NewSource(nextTime.Unix()))
	numberRand = wdRand.Intn(int(duration / randomTimeFrame))
	fmt.Println("numberRand 2:", numberRand)
	out = append(out, nextTime.Add(time.Duration(numberRand)*randomTimeFrame))
	return out
}
