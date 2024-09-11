package main

import (
	"database/sql"
	"log"
	"os"

	"github.com/hiendaovinh/toolkit/pkg/db"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"github.com/robfig/cron/v3"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/urfave/cli/v2"
)

const gameID = "catia"
const NOTIFY_AUTO_CONTENT = "NOTIFY_AUTO_CONTENT"

func init() {
	// for development
	//nolint:errcheck
	godotenv.Load("../../.env")

	// for production
	//nolint:errcheck
	godotenv.Load("./.env")
}

type CronJob interface {
	start()
}

func main() {
	app := &cli.App{
		Name: "cronjob",
		Commands: []*cli.Command{
			commandCronjob(),
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func commandCronjob() *cli.Command {
	return &cli.Command{
		Name: "cron",
		Action: func(c *cli.Context) error {
			db, err := getDb()
			if err != nil {
				return err
			}
			redis, err := getRedis()
			if err != nil {
				return err
			}

			cronRunner := cron.New()

			leaderboardJob := NewLeaderboardJob(redis, db)
			leaderboardJob.Start(cronRunner)
			log.Println("Start cronjob")
			cronRunner.Run()
			return nil
		},
	}
}

func getDb() (*bun.DB, error) {
	godotenv.Load()
	sqldb := sql.OpenDB(pgdriver.NewConnector(
		pgdriver.WithDSN(os.Getenv("DB_DSN")),
		pgdriver.WithPassword(os.Getenv("DB_PASSWORD")),
	))

	db := bun.NewDB(sqldb, pgdialect.New())
	return db, nil
}

func getRedis() (redis.UniversalClient, error) {
	var dbRedis redis.UniversalClient
	var err error

	clusterRedisQuestionnaire := os.Getenv("CLUSTER_REDIS_QUESTIONNAIRE")
	if clusterRedisQuestionnaire != "" {
		clusterOpts, err := redis.ParseClusterURL(clusterRedisQuestionnaire)
		if err != nil {
			return nil, err
		}
		dbRedis = redis.NewClusterClient(clusterOpts)
	} else {
		dbRedis, err = db.InitRedis(&db.RedisConfig{
			URL: os.Getenv("REDIS_QUESTIONNAIRE"),
		})
		if err != nil {
			return nil, err
		}
	}
	return dbRedis, nil
}
