package main

import (
	"context"
	"fmt"
	"log"
	"millionaire/internal/datastore"
	"millionaire/internal/datastore/redis_store"
	"millionaire/internal/models"
	"millionaire/internal/pkg"
	"millionaire/internal/services"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/robfig/cron/v3"
	"github.com/uptrace/bun"
)

type LeaderboardJob struct {
	Redis redis.UniversalClient
	Db    *bun.DB
}

func NewLeaderboardJob(redis redis.UniversalClient, db *bun.DB) *LeaderboardJob {
	return &LeaderboardJob{
		Redis: redis,
		Db:    db,
	}
}

func (j *LeaderboardJob) Start(cronRunner *cron.Cron) {
	timeline, err := datastore.GetConfigByKey(context.Background(), j.Db, "CRONJOB_TIME_LEADERBOARD")
	if err != nil {
		fmt.Println(err)
		return
	}

	if timeline == nil || timeline.Value == "" {
		fmt.Println("No timeline found")
		return
	}

	_, err = cronRunner.AddFunc(timeline.Value, j.runScheduledTask)
	log.Println("Leaderboard Cronjob start at:", time.Now().Format("2006-01-02 15:04:05"), "cron:", timeline.Value, err)
	j.initLeaderboard()
}

func (j *LeaderboardJob) runScheduledTask() {
	ctx := context.Background()
	log.Println("Start cleaning weekly leaderboard ...")
	err := redis_store.ClearLeaderboard(ctx, j.Redis, services.LEADERBOARD_OVERALL_WEEKLY)
	if err != nil {
		fmt.Println(err)
		return
	}
	log.Println("Weekly leaderboard cleaned")
}

func (j *LeaderboardJob) initLeaderboard() {
	// loop through list usser and score to add to leaderboard
	ctx := context.Background()
	limit := 100
	offset := 0

	startTimeOfWeek := pkg.GetFirstTimeOfCurrentWeek()
	log.Println("Start loading user gem from time:", startTimeOfWeek)

	for {
		log.Println("limit:", limit, "offset:", offset)
		userGems, err := datastore.GetUserTotalGemListFromTime(ctx, j.Db, &startTimeOfWeek, limit, offset)
		offset += limit
		if err != nil {
			log.Println(err)
			continue
		}

		if len(userGems) == 0 {
			log.Println("No more user gem found. Finish loading user gem")
			break
		}

		for _, userGem := range userGems {
			_, err := redis_store.SetLeaderboard(ctx, j.Redis, services.LEADERBOARD_OVERALL_WEEKLY, &models.LeaderboardItem{
				UserId: userGem.UserID,
				Score:  float64(userGem.TotalGems),
			})
			if err != nil {
				log.Println(err)
			}
		}
	}
}
