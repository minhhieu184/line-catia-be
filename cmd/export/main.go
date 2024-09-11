package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"millionaire/internal/datastore"
	"millionaire/internal/datastore/redis_store"
	"millionaire/internal/models"
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
		Name: "export",
		Commands: []*cli.Command{
			commandExport(),
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func commandExport() *cli.Command {
	return &cli.Command{
		Name: "export",
		Action: func(c *cli.Context) error {
			vs, err := env.EnvsRequired(
				"BOT_TOKEN",
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

			limit := 100
			offset := 0
			ctx := context.Background()

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
					//get latest game session of the user
					gameListSession, err := redis_store.GetGameListSessions(ctx, dbRedis, "catia", user.ID)
					if err == redis.Nil {
						fmt.Printf("%d: no session", user.ID)
						continue
					}
					if err != nil {
						fmt.Println(err)
						continue
					}

					longestStreak := 0
					bonusPoint := 0
					minusPoint := 0

					//get the longest streak of the user
					for _, session := range gameListSession {
						if session.StreakPoint > longestStreak {
							longestStreak = session.StreakPoint
						}

						if len(session.History) == 10 {
							if session.History[9].TotalScore == session.History[8].TotalScore {
								continue
							}
							if session.History[9].TotalScore > session.History[8].TotalScore {
								points := session.History[9].TotalScore - session.History[8].TotalScore
								bonusPoint += points
							} else {
								points := session.History[8].TotalScore - session.History[9].TotalScore
								minusPoint += points
							}
						}
					}

					err = redis_store.SetLongestStreak(ctx, dbRedis, "catia", &models.LongestStreak{
						Username:    user.Username,
						StreakPoint: longestStreak,
					})
					if err != nil {
						fmt.Println(err)
						continue
					}

					err = redis_store.SetMostPlayedSession(ctx, dbRedis, "catia", &models.MostSessions{
						Username:      user.Username,
						TotalSessions: len(gameListSession),
					})
					if err != nil {
						fmt.Println(err)
						continue
					}

					err = redis_store.SetMostBonusPoint(ctx, dbRedis, "catia", &models.MostBonusPoint{
						Username:        user.Username,
						BonusPointsTime: bonusPoint,
					})
					if err != nil {
						fmt.Println(err)
						continue
					}

					err = redis_store.SetMostMinusPoint(ctx, dbRedis, "catia", &models.MostMinusPoint{
						Username:        user.Username,
						MinusPointsTime: minusPoint,
					})
					if err != nil {
						fmt.Println(err)
						continue
					}
				}

				fmt.Println("done", offset, limit)
			}

			mostBonusPoints, err := redis_store.GetMostBonusPoint(ctx, dbRedis, "catia", 5)
			if err != nil {
				fmt.Println(err)
			}
			fmt.Println("MOST BONUS POINTS LIST - TOP 5:")
			for _, mostBonusPoint := range mostBonusPoints {
				fmt.Printf("Username: %s, Bonus Points: %d\n", mostBonusPoint.Username, mostBonusPoint.BonusPointsTime)
			}

			mostMinusPoints, err := redis_store.GetMostMinusPoint(ctx, dbRedis, "catia", 5)
			if err != nil {
				fmt.Println(err)
			}
			fmt.Println("MOST MINUS POINTS LIST - TOP 5:")
			for _, mostMinusPoint := range mostMinusPoints {
				fmt.Printf("Username: %s, Minus Points: %d\n", mostMinusPoint.Username, mostMinusPoint.MinusPointsTime)
			}

			mostSessions, err := redis_store.GetMostPlayedSession(ctx, dbRedis, "catia", 5)
			if err != nil {
				fmt.Println(err)
			}

			fmt.Println("MOST PLAYED SESSIONS LIST - TOP 5:")
			for _, mostSession := range mostSessions {
				fmt.Printf("Username: %s, Total Sessions: %d\n", mostSession.Username, mostSession.TotalSessions)
			}

			longestStreaks, err := redis_store.GetLongestStreak(ctx, dbRedis, "catia", 5)
			if err != nil {
				fmt.Println(err)
			}

			fmt.Println("LONGEST STREAKS LIST - TOP 5:")
			for _, longestStreak := range longestStreaks {
				fmt.Printf("Username: %s, Streak Point: %d\n", longestStreak.Username, longestStreak.StreakPoint)
			}

			fmt.Println("DONE ALL")
			return nil
		},
	}
}
