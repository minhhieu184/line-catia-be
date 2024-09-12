package main

import (
	"context"
	"database/sql"
	"encoding/csv"
	"fmt"
	"log"
	"millionaire/internal/services"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/hiendaovinh/toolkit/pkg/db"
	"github.com/hiendaovinh/toolkit/pkg/env"
	"github.com/joho/godotenv"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/urfave/cli/v2"

	"millionaire/internal/datastore"
	"millionaire/internal/datastore/redis_store"
	"millionaire/internal/models"
)

const (
	STREAK_STEP_SCORE         = 20
	BONUS_NEW_USER            = 200
	BONUS_POINTS_5_QUESTIONS  = 500
	BONUS_POINTS_9_QUESTIONS  = 800
	BONUS_POINTS_30_QUESTIONS = 1000
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
		Name: "migrate",
		Commands: []*cli.Command{
			commandMigration(),
			commandUserGameMigration(),
			commandConfigMigration(),
			commandSessionMigration(),
			commandUserGemMigrate(),
			commandUserBoostMigrate(),
			commandUserInviteesMigrate(),
			commandInsertBoosts(),
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func commandMigration() *cli.Command {
	return &cli.Command{
		Name: "migrate",
		Action: func(c *cli.Context) error {
			ctx := context.Background()
			db, err := getDb()
			if err != nil {
				log.Fatal(err)
			}

			err = datastore.CreateTableGame(ctx, db)
			if err != nil {
				log.Fatal(err)
			}

			err = datastore.CreateTableQuestion(ctx, db)
			if err != nil {
				log.Fatal(err)
			}

			err = datastore.CreateTableQuestionTranslation(ctx, db)
			if err != nil {
				log.Fatal(err)
			}

			err = datastore.CreateTableUser(ctx, db)
			if err != nil {
				log.Fatal(err)
			}

			err = datastore.CreateTableConfig(ctx, db)
			if err != nil {
				log.Fatal(err)
			}

			err = datastore.CreateTableGameCategory(ctx, db)
			if err != nil {
				log.Fatal(err)
			}

			err = datastore.CreateTableUserGame(ctx, db)
			if err != nil {
				log.Fatal(err)
			}

			err = datastore.CreateTableUserBoost(ctx, db)
			if err != nil {
				log.Fatal(err)
			}

			err = datastore.CreateTableUserWallet(ctx, db)
			if err != nil {
				log.Fatal(err)
			}

			err = datastore.CreateTableGameSession(ctx, db)
			if err != nil {
				log.Fatal(err)
			}

			err = datastore.CreateTableQuestionHistory(ctx, db)
			if err != nil {
				log.Fatal(err)
			}

			err = datastore.CreateTableSocialTask(ctx, db)
			if err != nil {
				log.Fatal(err)
			}

			err = datastore.CreateTableLifelineHistory(ctx, db)
			if err != nil {
				log.Fatal(err)
			}

			err = datastore.CreateTableUserFreebies(ctx, db)
			if err != nil {
				log.Fatal(err)
			}

			err = datastore.CreateTableUserGem(ctx, db)
			if err != nil {
				log.Fatal(err)
			}

			err = datastore.CreateTablePartner(ctx, db)
			if err != nil {
				log.Fatal(err)
			}

			err = datastore.CreateTableReward(ctx, db)
			if err != nil {
				log.Fatal(err)
			}

			err = datastore.CreateTableArena(ctx, db)
			if err != nil {
				log.Fatal(err)
			}

			fmt.Println("Migration success")

			return nil
		},
	}
}

func commandUserGameMigration() *cli.Command {
	return &cli.Command{
		Name: "migrate-user-game",
		Action: func(c *cli.Context) error {
			ctx := context.Background()
			db, err := getDb()
			if err != nil {
				log.Fatal(err)
			}

			limit := 100
			offset := 0

			for {
				users, err := datastore.GetUsersByLimit(ctx, db, limit, offset)
				if err != nil {
					log.Fatal(err)
				}

				if len(users) == 0 {
					break
				}

				for _, user := range users {
					userGame := &models.UserGame{
						UserID:                user.ID,
						GameSlug:              "catia",
						ExtraSessions:         user.ExtraSessions,
						CurrentBonusMilestone: user.CurrentBonusMilestone,
						Countdown:             user.Countdown,
						CurrentSessionID:      user.CurrentSessionID,
						GiftPoints:            user.GiftPoints,
					}
					err = datastore.SetUserGame(ctx, db, userGame)
					if err != nil {
						fmt.Println(err)
					}
				}

				fmt.Println("Done", offset, limit)

				offset += limit
			}

			fmt.Println("Migration success")

			return nil
		},
	}
}

// insert default configs to db
func commandConfigMigration() *cli.Command {
	return &cli.Command{
		Name:        "migrate-config",
		Description: "Insert default configs to db",
		Action: func(c *cli.Context) error {
			ctx := context.Background()
			db, err := getDb()
			if err != nil {
				log.Fatal(err)
			}

			configs := []models.Config{
				{Key: services.CONFIG_SERVER_MODE, Value: "production"},
				{Key: services.CONFIG_GAME_LEADERBOARD_LIMIT, Value: "53"},
				{Key: "NOTIFY_ALL_CONTENT", Value: "More friends - More fun ✨✨\n\n1 friend invited will shorten your wait time by 4 hours (activated from now)! Guess what: 3 new friends joined so you don't have to wait any longer for the new turn 🎉\n\nSend your refer link now!!! Hurry to win the race and bring rewards home ❤️‍🔥"},
				{Key: "NOTIFY_AUTO_CONTENT", Value: "🎉A new turn is available! Hurry to climb the Leaderboard before <b>\"Millionaire, Now or Never\"</b> is over!! Try hard and give the Lucky Wheel a whirl to see what you get today✨"},
				{Key: "TEXT_CONGRATULATION", Value: "➡️ Here is the reward for your hard work: Fan badges - NFT of Catia\n\n» Guaranteed ticket to join Catia's 1st APE\n» Membership key to onboard \"Catia Scholars\" department  \n» Hidden privileges for Top 3 \n\n❣️❣️\"Catia Scholars\" department is widely open for you: https://t.me/+lyyCQvZff704MmI9 \n\n▪️Press on the link above\n▪️Click on \"Request to Join\"\n▪️Sit down and take some tea \n▪️Once you become a member, you will know more about your privileges of the upcoming APEs\n\nJoin now and get ready to conquer new campaigns from \"Millionaire, Now or Never\"!!!"},
				{Key: "TOP_50", Value: ""},
				{Key: "CRONJOB_TIME_SEND_MESSAGE", Value: "@every 1d"},
				{Key: "NOTIFY_TIME", Value: "48"},
				{Key: services.CONFIG_BONUS_TASK_POINT, Value: "5"},
				{Key: services.CONFIG_TEXT_NEW_USER, Value: `🎮 Welcome to "Millionaire, Now or Never"!

Join us in this exciting adventure where you can earn Gems, play minigames, climb the leaderboard, and unlock access to exclusive Asset Publishing Events!

Meet Catia - an asset publishing protocol for premium web3 projects. Crafted by Icetea Labs with sustainability, transparency and community-driven spirit.

🚀 Ready to learn more about us and win big rewards? Game ON!`},
				{Key: services.CONFIG_MIN_GEM_TO_CLAIM_REF_BOOST, Value: "16"},
				{Key: services.CONFIG_FREEBIE_GEM_COUNTDOWN, Value: "5"},
				{Key: services.CONFIG_FREEBIE_STAR_COUNTDOWN, Value: "5"},
				{Key: services.CONFIG_FREEBIE_LIFELINE_COUNTDOWN, Value: "5"},
				{Key: services.CONFIG_CRONJOB_TIME_FULL_MOON, Value: "@every 3h"},
				{Key: services.CONFIG_FULL_MOON_START_TIME, Value: "2024-07-10T08:00:00Z07:00"},
				{Key: services.CONFIG_FULL_MOON_END_TIME, Value: "2024-07-20T08:00:00Z07:00"},
				{Key: services.CONFIG_MOON_TIME_PER_RANGE_IN_MINUTES, Value: "180"},
				{Key: services.CONFIG_MOON_EXPIRED_TIME_IN_MINUTES, Value: "10"},
				{Key: services.CONFIG_MOON_RANDOM_UNIT_IN_MINUTES, Value: "15"},
				{Key: services.CONFIG_OVERALL_LEADERBOARD_LIMIT, Value: "53"},
				{Key: services.CONFIG_REFERRAL_LEADERBOARD_LIMIT, Value: "53"},
				{Key: services.CONFIG_ARENA_LEADERBOARD_LIMIT, Value: "53"},
				{Key: "CRONJOB_TIME_LEADERBOARD", Value: "0 0 * * 1"},
				{Key: "ADMIN_CHAT_ID", Value: ""},
			}

			for _, config := range configs {
				_, err = db.NewInsert().Model(&config).Exec(ctx)
				if err != nil {
					log.Println(err)
				}
			}

			fmt.Println("Migration success")

			return nil
		},
	}
}

func commandSessionMigration() *cli.Command {
	return &cli.Command{
		Name: "migrate-session",
		Action: func(c *cli.Context) error {
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
			limit := 100
			offset := 0
			ctx := context.Background()
			for {
				fmt.Println(time.Now(), "start", offset, limit)
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
					sessions, err := redis_store.GetGameListSessions(ctx, dbRedis, "catia", user.ID)
					if err != nil {
						fmt.Println(err)
						continue
					}

					sort.Slice(sessions, func(i, j int) bool {
						if sessions[i].StartedAt == nil {
							return false
						}

						if sessions[j].StartedAt == nil {
							return true
						}
						return sessions[i].StartedAt.Before(*sessions[j].StartedAt)
					})

					totalCorrectAnswers := 0

					currentBonusMilestone := 0

					for index, session := range sessions {
						// if session.ID == *userGame.CurrentSessionID {
						// 	continue
						// }

						gameSlug := session.GameSlug
						if gameSlug == "" {
							gameSlug = "catia"
						}

						bonusScore := 0
						sessionTotalScore := 0
						sessionCorrectAnswers := 0
						if index == 0 {
							bonusScore = BONUS_NEW_USER
						}

						if session.TotalScore > 0 {
							for _, history := range session.History {
								if history.Correct != nil && *history.Correct {
									sessionCorrectAnswers++
									totalCorrectAnswers++
								}
							}

							if currentBonusMilestone < 1 && sessionCorrectAnswers >= 5 {
								bonusScore += BONUS_POINTS_5_QUESTIONS
								currentBonusMilestone += 1
							}

							if currentBonusMilestone < 2 && sessionCorrectAnswers >= 9 {
								bonusScore += BONUS_POINTS_9_QUESTIONS
								currentBonusMilestone += 1
							}

							if currentBonusMilestone < 3 && totalCorrectAnswers >= 30 && len(sessions) >= 5 {
								currentBonusMilestone += 1
								bonusScore += BONUS_POINTS_30_QUESTIONS
							}

							sessionTotalScore += session.TotalScore + bonusScore + session.StreakPoint*STREAK_STEP_SCORE
						}

						newSession := &models.GameSession{
							LegacyID:             session.ID,
							GameSlug:             gameSlug,
							UserID:               session.UserID,
							NextStep:             session.NextStep,
							CurrentQuestion:      session.CurrentQuestion,
							CurrentQuestionScore: session.CurrentQuestionScore,
							Score:                session.TotalScore,
							BonusScore:           bonusScore,
							TotalScore:           sessionTotalScore,
							QuestionStartedAt:    session.QuestionStartedAt,
							StartedAt:            session.StartedAt,
							EndedAt:              session.EndedAt,
							StreakPoint:          session.StreakPoint,
							UsedBoostCount:       session.UsedBoostCount,
							CorrectAnswerCount:   sessionCorrectAnswers,
						}

						err = datastore.CreateGameSession(ctx, db, newSession)
						if err != nil {
							fmt.Println(err)
						}

						if newSession.ID < 1 {
							continue
						}

						correctCount := 0

						for key, history := range session.History {
							if history.Correct != nil && *history.Correct {
								correctCount++
							}
							history := models.QuestionHistory{
								GameSessionID: newSession.ID,
								Index:         key,
								TotalScore:    history.TotalScore,
								QuestionID:    history.Question.ID,
								QuestionScore: history.QuestionScore,
								StartedAt:     history.StartedAt,
								Answer:        history.Answer,
								AnsweredAt:    history.AnsweredAt,
								Correct:       history.Correct,
								Checkpoint:    history.Checkpoint,
							}

							//insert question history
							err = datastore.CreateQuestionHistory(ctx, db, history)
							if err != nil {
								fmt.Println(err)
								continue
							}
						}
					}
				}
			}
			return nil
		},
	}
}

func commandUserGemMigrate() *cli.Command {
	return &cli.Command{
		Name: "migrate-user-gem",
		Action: func(c *cli.Context) error {
			vs, err := env.EnvsRequired(
				"DB_DSN",
				"DB_PASSWORD",
				"REDIS_QUESTIONNAIRE",
			)
			if err != nil {
				return err
			}
			ctx := context.Background()

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

			dbPostgres, err := getDb()
			if err != nil {
				log.Fatal(err)
			}

			limit := 100
			offset := 0

			for {
				users, err := datastore.GetUsersByLimit(ctx, dbPostgres, limit, offset)
				if err != nil {
					log.Fatal(err)
				}

				if len(users) == 0 {
					break
				}

				for _, user := range users {
					//get user leaderboard
					rank, err := redis_store.GetRankWithScore(ctx, dbRedis, "catia", user)
					if err != nil {
						fmt.Println("err get rank", err)
						continue
					}

					userGem := &models.UserGem{
						UserID:    user.ID,
						Gems:      int(rank.Score),
						Action:    "gem-from-legacy:catia",
						CreatedAt: time.Now(),
					}

					err = datastore.InsertUserGem(ctx, dbPostgres, userGem)
					if err != nil {
						fmt.Println(err)
					}
				}

				fmt.Println("Done", offset, limit)

				offset += limit
			}

			fmt.Println("Migration success")

			return nil
		},
	}
}

func commandUserBoostMigrate() *cli.Command {
	return &cli.Command{
		Name: "migrate-user-boost",
		Action: func(c *cli.Context) error {
			ctx := context.Background()

			dbPostgres, err := getDb()
			if err != nil {
				log.Fatal(err)
			}

			limit := 100
			offset := 0

			now := time.Now()

			refCount := map[string]int{}

			for {
				users, err := datastore.GetUsersSortedByCreatedAt(ctx, dbPostgres, limit, offset)
				if err != nil {
					log.Println(err)
				}

				if len(users) == 0 {
					break
				}

				for _, user := range users {
					if user.InviterID != nil {
						used := true
						validated := true

						u, err := datastore.GetUserGameSessionSumary(ctx, dbPostgres, "catia", user.ID)
						if err != nil {
							fmt.Println(err)
							continue
						}

						if u.TotalScore >= 216 {
							refCount[*user.InviterID]++
							if refCount[*user.InviterID] > 2 {
								used = false
								refCount[*user.InviterID] = 0
							}
						}

						userBoost := &models.UserBoost{
							UserID:    *user.InviterID,
							CreatedAt: now,
							Source:    user.ID,
							Validated: validated,
						}

						if used {
							userBoost.UsedAt = &now
							userBoost.Used = used
							userBoost.UsedFor = "burn_out"
						}
						err = datastore.CreateBoost(ctx, dbPostgres, userBoost)
						if err != nil {
							fmt.Println(err)
						}
					}
				}

				fmt.Println("Done", offset, limit)

				offset += limit
			}

			fmt.Println("Migration success")

			return nil
		},
	}
}

func commandUserInviteesMigrate() *cli.Command {
	return &cli.Command{
		Name: "migrate-user-invitees",
		Action: func(c *cli.Context) error {
			ctx := context.Background()

			dbPostgres, err := getDb()
			if err != nil {
				log.Fatal(err)
			}

			limit := 100
			offset := 0

			refCount := map[string]int{}

			for {
				users, err := datastore.GetUsersSortedByCreatedAt(ctx, dbPostgres, limit, offset)
				if err != nil {
					log.Println(err)
				}

				if len(users) == 0 {
					break
				}

				for _, user := range users {
					if user.InviterID != nil {
						refCount[*user.InviterID]++
					}
				}

				fmt.Println("Done get info", offset, limit)

				offset += limit
			}

			fmt.Println("start updating invitees")

			//loop the refcount
			for refID, count := range refCount {
				//update user invitees
				if count < 1 {
					continue
				}

				err = datastore.UpdateUserInvitees(ctx, dbPostgres, refID, count)
				if err != nil {
					fmt.Println(err)
					continue
				}
			}

			fmt.Println("Migration success")

			return nil
		},
	}
}

func getDb() (*bun.DB, error) {
	fmt.Println(os.Getenv("DB_DSN"))
	sqldb := sql.OpenDB(pgdriver.NewConnector(
		pgdriver.WithDSN(os.Getenv("DB_DSN")),
		pgdriver.WithPassword(os.Getenv("DB_PASSWORD")),
	))

	db := bun.NewDB(sqldb, pgdialect.New())
	return db, nil
}

func commandInsertBoosts() *cli.Command {
	return &cli.Command{
		Name: "insert-boosts",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "input",
				Value: "./data.csv",
			},
			&cli.StringFlag{
				Name: "source",
			},
		},
		Action: func(c *cli.Context) error {
			ctx := context.Background()

			db, err := getDb()
			if err != nil {
				log.Fatal(err)
			}

			//row 1 is userId, row2 is number of boosts
			inputPath := c.String("input")
			if _, err := os.Stat(inputPath); os.IsNotExist(err) {
				return err
			}

			source := c.String("source")

			file, err := os.Open(inputPath)
			if err != nil {
				return err
			}

			r := csv.NewReader(file)

			for {
				row, err := r.Read()
				if err != nil {

					break
				}

				// userID, err := strconv.ParseInt(row[0], 10, 64)
				userID := row[0]
				if err != nil {
					return err
				}

				count, err := strconv.Atoi(row[1])
				if err != nil {
					return err
				}

				for i := 0; i < count; i++ {
					boost := &models.UserBoost{
						UserID:    userID,
						Validated: true,
						Source:    fmt.Sprintf("%s:%d", source, i+1),
					}

					err = datastore.CreateBoost(ctx, db, boost)
					if err != nil {
						fmt.Println(err)
					}
				}
			}

			return nil
		},
	}
}
