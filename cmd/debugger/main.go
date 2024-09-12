package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"millionaire/internal/datastore"
	"millionaire/internal/datastore/redis_store"
	"millionaire/internal/models"
	"millionaire/internal/pkg/caching"
	"millionaire/internal/services"
	"os"

	"github.com/redis/go-redis/v9"

	"github.com/hiendaovinh/toolkit/pkg/db"
	"github.com/hiendaovinh/toolkit/pkg/env"
	"github.com/joho/godotenv"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/urfave/cli/v2"
)

func init() {
	// for development
	//nolint:errcheck
	godotenv.Load("../../.env")

	// for production
	//nolint:errcheck
	godotenv.Load("./.env")
}

func main() {
	app := &cli.App{
		Name: "debugger",
		Commands: []*cli.Command{
			commandDeleteUser(),
			commandClearUserSocialTask(),
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func commandDeleteUser() *cli.Command {
	return &cli.Command{
		Name: "delete-user",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name: "userid",
			},
		},
		Action: func(c *cli.Context) error {
			ctx := context.Background()
			vs, err := env.EnvsRequired(
				"DB_DSN",
				"DB_PASSWORD",
				"REDIS_QUESTIONNAIRE",
				"REDIS_CACHE",
			)
			if err != nil {
				return err
			}

			dbPostgres, err := getDb()
			if err != nil {
				log.Fatal(err)
			}

			var dbRedis redis.UniversalClient
			var dbRedisCache redis.UniversalClient

			clusterRedisQuestionnaire := os.Getenv("CLUSTER_REDIS_QUESTIONNAIRE")
			if clusterRedisQuestionnaire != "" {
				clusterOpts, err := redis.ParseClusterURL(clusterRedisQuestionnaire)
				if err != nil {
					return err
				}
				dbRedis = redis.NewClusterClient(clusterOpts)
			} else {
				dbRedis, err = db.InitRedis(&db.RedisConfig{
					URL: vs["REDIS_QUESTIONNAIRE"],
				})
				if err != nil {
					return err
				}
			}

			clusterCacheRedisURL := os.Getenv("CLUSTER_REDIS_CACHE")
			if clusterCacheRedisURL != "" {
				clusterOpts, err := redis.ParseClusterURL(clusterCacheRedisURL)
				if err != nil {
					return err
				}
				dbRedisCache = redis.NewClusterClient(clusterOpts)
			} else {
				dbRedisCache, err = db.InitRedis(&db.RedisConfig{
					URL: vs["REDIS_CACHE"],
				})
				if err != nil {
					log.Fatal(err)
				}
			}

			// userid, err := strconv.ParseInt(c.String("userid"), 10, 64)
			userid := c.String("userid")

			if err != nil {
				log.Fatal(err)
			}

			user, _ := datastore.FindUserByID(ctx, dbPostgres, userid)

			if user != nil {
				fmt.Println("User found, deleting...")
				_, err := dbPostgres.NewDelete().Model(user).WherePK().Exec(ctx)
				if err != nil {
					log.Fatal(err)
				}
			}
			_, err = dbPostgres.NewDelete().TableExpr("user_game").Where("user_id = ?", userid).Exec(ctx)
			if err != nil {
				log.Println(err)
			}
			_, err = dbPostgres.NewDelete().TableExpr("game_session").Where("user_id = ?", userid).Exec(ctx)
			if err != nil {
				log.Println(err)
			}

			_, err = dbPostgres.NewDelete().TableExpr("user_gem").Where("user_id = ?", userid).Exec(ctx)
			if err != nil {
				log.Println(err)
			}

			_, err = dbPostgres.NewDelete().TableExpr("user_freebie").Where("user_id = ?", userid).Exec(ctx)
			if err != nil {
				log.Println(err)
			}

			_, err = dbPostgres.NewDelete().TableExpr("user_boost").Where("user_id = ?", userid).Exec(ctx)
			if err != nil {
				log.Println(err)
			}

			caching.DeleteKeys(ctx, dbRedis, fmt.Sprintf("game_session:*:%d", userid))
			caching.DeleteKeys(ctx, dbRedis, fmt.Sprintf("user:joined_social_link:%d:*", userid))

			games, _ := datastore.GetEnabledGames(ctx, dbPostgres)
			for _, game := range games {
				_, err = redis_store.SetLeaderboard(ctx, dbRedis, game.Slug, &models.LeaderboardItem{
					UserId: userid,
					Score:  0,
				})
				fmt.Println("Deleted leaderboard score", game.Slug, err)
			}

			_, err = redis_store.SetLeaderboard(ctx, dbRedis, services.LEADERBOARD_OVERALL, &models.LeaderboardItem{
				UserId: userid,
				Score:  0,
			})
			fmt.Println("Deleted leaderboard score overall", err)

			caching.DeleteKeys(ctx, dbRedisCache, fmt.Sprintf("*%d*", userid))
			return nil
		},
	}
}

func commandClearUserSocialTask() *cli.Command {
	return &cli.Command{
		Name: "clear-user-social-task",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name: "userid",
			},
		},
		Action: func(c *cli.Context) error {
			ctx := context.Background()
			vs, err := env.EnvsRequired(
				"DB_DSN",
				"DB_PASSWORD",
				"REDIS_QUESTIONNAIRE",
				"REDIS_CACHE",
			)
			if err != nil {
				return err
			}

			dbPostgres, err := getDb()
			if err != nil {
				log.Fatal(err)
			}

			var dbRedis redis.UniversalClient
			var dbRedisCache redis.UniversalClient

			clusterRedisQuestionnaire := os.Getenv("CLUSTER_REDIS_QUESTIONNAIRE")
			if clusterRedisQuestionnaire != "" {
				clusterOpts, err := redis.ParseClusterURL(clusterRedisQuestionnaire)
				if err != nil {
					return err
				}
				dbRedis = redis.NewClusterClient(clusterOpts)
			} else {
				dbRedis, err = db.InitRedis(&db.RedisConfig{
					URL: vs["REDIS_QUESTIONNAIRE"],
				})
				if err != nil {
					return err
				}
			}

			clusterCacheRedisURL := os.Getenv("CLUSTER_REDIS_CACHE")
			if clusterCacheRedisURL != "" {
				clusterOpts, err := redis.ParseClusterURL(clusterCacheRedisURL)
				if err != nil {
					return err
				}
				dbRedisCache = redis.NewClusterClient(clusterOpts)
			} else {
				dbRedisCache, err = db.InitRedis(&db.RedisConfig{
					URL: vs["REDIS_CACHE"],
				})
				if err != nil {
					log.Fatal(err)
				}
			}

			// userid, err := strconv.ParseInt(c.String("userid"), 10, 64)
			useridString := c.String("userid")

			if useridString == "" {
				log.Fatal("userid is required")
			}

			// userid, err := strconv.ParseInt(useridString, 10, 64)
			userid := useridString

			if err != nil {
				log.Fatal(err)
			}

			user, _ := datastore.FindUserByID(ctx, dbPostgres, userid)

			if user != nil {
				fmt.Println("User found, deleting...")
				caching.DeleteKeys(ctx, dbRedis, fmt.Sprintf("user:joined_social_link:%d:*", user.ID))

				_, err = redis_store.SetLeaderboard(ctx, dbRedis, services.LEADERBOARD_OVERALL, &models.LeaderboardItem{
					UserId: user.ID,
					Score:  0,
				})
				fmt.Println("Deleted leaderboard score overall", err)
				caching.DeleteKeys(ctx, dbRedisCache, fmt.Sprintf("user:verify_join_social_link:%d*", user.ID))
				caching.DeleteKeys(ctx, dbRedisCache, fmt.Sprintf("user:%d*", user.ID))
				caching.DeleteKeys(ctx, dbRedisCache, fmt.Sprintf("me:%d*", user.ID))
				caching.DeleteKeys(ctx, dbRedisCache, fmt.Sprintf("social_task:user:%d*", user.ID))
			}
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
