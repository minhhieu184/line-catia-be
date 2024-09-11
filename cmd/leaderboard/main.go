package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"millionaire/internal/datastore"
	"millionaire/internal/datastore/redis_store"
	"millionaire/internal/models"
	"millionaire/internal/services"
	"os"

	"github.com/hiendaovinh/toolkit/pkg/db"
	"github.com/hiendaovinh/toolkit/pkg/env"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
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
		Name: "leaderboard",
		Commands: []*cli.Command{
			commandOverallLeaderboard(),
			commandReferralLeaderboard(),
			commandGamesLeaderboard(),
			commandCatiaArenaLeaderboard(),
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func commandOverallLeaderboard() *cli.Command {
	return &cli.Command{
		Name:        "overall-leaderboard",
		Description: "Used to update overall leaderboard",
		Action: func(c *cli.Context) error {
			// TODO: add option to flush entire leaderboard before updating
			vs, err := env.EnvsRequired(
				"DB_DSN",
				"DB_PASSWORD",
				"REDIS_QUESTIONNAIRE",
			)
			if err != nil {
				return err
			}

			var dbRedis redis.UniversalClient

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

			sqldb := sql.OpenDB(pgdriver.NewConnector(
				pgdriver.WithDSN(os.Getenv("DB_DSN")),
				pgdriver.WithPassword(os.Getenv("DB_PASSWORD")),
			))
			db := bun.NewDB(sqldb, pgdialect.New())

			ctx := context.Background()

			games, err := datastore.GetEnabledGames(ctx, db)
			if err != nil {
				fmt.Println(err)
				return err
			}

			limit := 100
			offset := 0

			for {
				users, err := datastore.GetUsersSortedByCreatedAt(ctx, db, limit, offset)
				offset += limit
				if err != nil {
					fmt.Println(err)
					continue
				}
				if len(users) == 0 {
					fmt.Println("no new user, DONE")
					break
				}

				for _, user := range users {
					point := float64(0)
					for _, game := range games {
						redisScore, err := redis_store.GetRankWithScore(ctx, dbRedis, game.Slug, user)
						if err != nil && err != redis.Nil {
							fmt.Println(err)
							continue
						}

						point += redisScore.Score
					}

					leaderboardItem := &models.LeaderboardItem{
						UserId: user.ID,
						Score:  point,
					}

					_, err = redis_store.SetLeaderboard(ctx, dbRedis, services.LEADERBOARD_OVERALL, leaderboardItem)
					if err != nil {
						fmt.Println(err)
						continue
					}
				}

			}

			fmt.Println("done", offset, limit)

			return nil
		},
	}
}

func commandGamesLeaderboard() *cli.Command {
	return &cli.Command{
		Name:        "sync-game-leaderboard",
		Description: "Used to update games leaderboard",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name: "gameslug",
			},
		},
		Action: func(c *cli.Context) error {
			// TODO: add option to flush entire leaderboard before updating
			vs, err := env.EnvsRequired(
				"DB_DSN",
				"DB_PASSWORD",
				"REDIS_QUESTIONNAIRE",
			)
			if err != nil {
				return err
			}

			var dbRedis redis.UniversalClient

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

			sqldb := sql.OpenDB(pgdriver.NewConnector(
				pgdriver.WithDSN(os.Getenv("DB_DSN")),
				pgdriver.WithPassword(os.Getenv("DB_PASSWORD")),
			))
			db := bun.NewDB(sqldb, pgdialect.New())

			ctx := context.Background()

			gameSlug := c.String("gameslug")

			game, err := datastore.GetGame(ctx, db, gameSlug)
			if err != nil {
				fmt.Println(err)
				return err
			}

			limit := 100
			offset := 0

			for {
				fmt.Println("start", offset, limit)
				users, err := datastore.GetUsersSortedByCreatedAt(ctx, db, limit, offset)
				offset += limit
				if err != nil {
					fmt.Println(err)
					continue
				}
				if len(users) == 0 {
					fmt.Println("no new user, DONE")
					break
				}

				for _, user := range users {
					//get all game sessions of an user
					gameSessions, err := datastore.GetUserGameSessionSumary(ctx, db, game.Slug, user.ID)
					if err != nil {
						fmt.Println(err)
						continue
					}

					if (gameSessions.TotalScore) > 0 {
						leaderboardItem := &models.LeaderboardItem{
							UserId: user.ID,
							Score:  float64(gameSessions.TotalScore),
						}

						_, err = redis_store.SetLeaderboard(ctx, dbRedis, game.Slug, leaderboardItem)
						if err != nil {
							fmt.Println(err)
						}
					}

				}
			}

			fmt.Println("done", offset, limit)

			return nil
		},
	}
}

func commandReferralLeaderboard() *cli.Command {
	return &cli.Command{
		Name:        "referral-leaderboard",
		Description: "Used to update referral leaderboard",
		Action: func(c *cli.Context) error {
			// TODO: add option to flush entire leaderboard before updating
			vs, err := env.EnvsRequired(
				"DB_DSN",
				"DB_PASSWORD",
				"REDIS_QUESTIONNAIRE",
			)
			if err != nil {
				return err
			}

			var dbRedis redis.UniversalClient

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

			sqldb := sql.OpenDB(pgdriver.NewConnector(
				pgdriver.WithDSN(os.Getenv("DB_DSN")),
				pgdriver.WithPassword(os.Getenv("DB_PASSWORD")),
			))
			db := bun.NewDB(sqldb, pgdialect.New())

			ctx := context.Background()

			limit := 100
			offset := 0

			for {
				users, err := datastore.GetUsersSortedByCreatedAt(ctx, db, limit, offset)
				offset += limit
				if err != nil {
					fmt.Println(err)
					continue
				}
				if len(users) == 0 {
					fmt.Println("no new user, DONE")
					break
				}

				for _, user := range users {
					referralLeaderboard := &models.LeaderboardItem{
						UserId: user.ID,
						Score:  float64(user.TotalInvites),
					}

					_, err = redis_store.SetLeaderboard(ctx, dbRedis, services.LEADERBOARD_REFERRAL, referralLeaderboard)
					if err != nil {
						fmt.Println(err)
						continue
					}
				}

			}

			fmt.Println("done", offset, limit)

			return nil
		},
	}
}

func commandCatiaArenaLeaderboard() *cli.Command {
	return &cli.Command{
		Name:        "catia-arena-leaderboard",
		Description: "Set leaderboard for catia arena",
		Action: func(c *cli.Context) error {
			dbRedis, err := getRedis()
			if err != nil {
				return err
			}

			leaderboardData := []models.LeaderboardItem{
				{UserId: 225167411, Score: 97766},
				{UserId: 1262995839, Score: 79674},
				{UserId: 1713913994, Score: 74488},
				{UserId: 5185766852, Score: 61042},
				{UserId: 5114958024, Score: 26515},
				{UserId: 1565419194, Score: 18847},
				{UserId: 319932362, Score: 16948},
				{UserId: 1116899344, Score: 16802},
				{UserId: 907618817, Score: 15597},
				{UserId: 894821039, Score: 15003},
				{UserId: 711370079, Score: 14447},
				{UserId: 697766192, Score: 13658},
				{UserId: 1113275370, Score: 13477},
				{UserId: 1733587848, Score: 13258},
				{UserId: 5177232376, Score: 13045},
				{UserId: 1446024164, Score: 12608},
				{UserId: 6561395918, Score: 12555},
				{UserId: 1691036644, Score: 12135},
				{UserId: 965438346, Score: 12098},
				{UserId: 1107779311, Score: 12076},
				{UserId: 1777243963, Score: 11988},
				{UserId: 139207726, Score: 11979},
				{UserId: 6482545906, Score: 11100},
				{UserId: 7137730161, Score: 10933},
				{UserId: 483765468, Score: 10773},
				{UserId: 1176724768, Score: 10677},
				{UserId: 6993960154, Score: 10564},
				{UserId: 390821526, Score: 10527},
				{UserId: 1559144766, Score: 10340},
				{UserId: 1932831353, Score: 10252},
				{UserId: 245278757, Score: 10053},
				{UserId: 238855076, Score: 9984},
				{UserId: 859661175, Score: 9842},
				{UserId: 7064099931, Score: 9697},
				{UserId: 1916254355, Score: 9586},
				{UserId: 6609779241, Score: 9430},
				{UserId: 1067492691, Score: 9181},
				{UserId: 5395796480, Score: 8938},
				{UserId: 5532087062, Score: 8828},
				{UserId: 7004686108, Score: 8760},
				{UserId: 1562336135, Score: 8739},
				{UserId: 2026128805, Score: 8538},
				{UserId: 1907930754, Score: 8508},
				{UserId: 359763599, Score: 8496},
				{UserId: 809996094, Score: 8484},
				{UserId: 1043327818, Score: 8461},
				{UserId: 1608882768, Score: 8278},
				{UserId: 489762129, Score: 8154},
				{UserId: 1859407001, Score: 8126},
				{UserId: 785567171, Score: 7921},
			}

			_ = redis_store.ClearLeaderboard(context.Background(), dbRedis, services.DBKeyArena("catia"))

			for _, item := range leaderboardData {
				_, err = redis_store.SetLeaderboard(context.Background(), dbRedis, services.DBKeyArena("catia"), &item)
				if err != nil {
					fmt.Println(err)
					continue
				}
			}

			fmt.Println("done")

			return nil
		},
	}
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
