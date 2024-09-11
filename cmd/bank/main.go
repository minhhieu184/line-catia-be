package main

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/hiendaovinh/toolkit/pkg/db"
	"github.com/redis/go-redis/v9"

	"millionaire/internal/datastore"
	"millionaire/internal/datastore/redis_store"
	"millionaire/internal/models"

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
		Name: "api",
		Commands: []*cli.Command{
			commandImport(),
			commandImportTranslatedQuestion(),
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func commandImport() *cli.Command {
	return &cli.Command{
		Name: "import",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "input",
				Value: "./quiz.csv",
			},
			&cli.StringFlag{
				Name: "category",
			},
			&cli.BoolFlag{
				Name: "test",
			},
		},
		Action: func(c *cli.Context) error {
			postgresDB, err := getDb()
			if err != nil {
				return err
			}

			dbRedis, err := getRedis()
			if err != nil {
				return err
			}

			if err != nil {
				return err
			}

			inputPath := c.String("input")
			if _, err := os.Stat(inputPath); os.IsNotExist(err) {
				return err
			}

			file, err := os.Open(inputPath)
			if err != nil {
				return err
			}

			r := csv.NewReader(file)

			_, err = r.Read()
			if err != nil {
				return err
			}

			questions := []models.Question{}

			//get from cli
			category := c.String("category")

			for {
				record, err := r.Read()
				if err == io.EOF {
					break
				}
				if err != nil {
					return err
				}

				if len(record) < 8 {
					log.Println("invalid record length", record[0])
					continue
				}

				answerStr := strings.TrimSpace(record[6])
				if len(answerStr) != 1 {
					log.Println("invalid record answer", record[0])
					continue
				}

				answer := int([]rune(strings.ToUpper(record[6]))[0]) - 65 // A = 0, B = 1, C = 2, D = 3
				if c.Bool("test") {
					answer = 0
				}

				difficultyStr := strings.ToLower(strings.TrimSpace(record[7]))
				difficulty := models.QuestionDifficulty(difficultyStr)
				if !difficulty.Valid() {
					log.Println("invalid record difficulty", record[0])
					continue
				}
				choices := []*models.Choice{{Content: record[2], Key: 0}, {Content: record[3], Key: 1}, {Content: record[4], Key: 2}, {Content: record[5], Key: 3}}

				id, _ := strconv.Atoi(record[0])
				questions = append(questions, models.Question{
					QuestionBankID: id,
					Question:       record[1],
					Choices:        choices,
					CorrectAnswer:  answer,
					Difficulty:     difficulty,
					Category:       category, // default == catia
					Enabled:        true,
				})

			}

			// ctx := context.Background()
			ctx := c.Context
			// pipe := redis.Pipeline()

			groupDifficulty := map[string][]int{}
			for _, question := range questions {
				err := datastore.SetQuestion(ctx, postgresDB, &question)
				if err != nil {
					fmt.Println("Error when set question", err)
					continue
				}

				group := groupDifficulty[string(question.Difficulty)]
				if group == nil {
					group = []int{}
				}

				group = append(group, question.ID)
				groupDifficulty[string(question.Difficulty)] = group
			}

			redis_store.DeleteGameQuestionGroups(ctx, dbRedis, category)
			for difficulty, group := range groupDifficulty {
				redis_store.AddQuestionsToGroup(ctx, dbRedis, category, difficulty, group)
			}
			// _, err = pipe.Exec(ctx)
			// if err != nil {
			// 	return err
			// }

			return nil
		},
	}
}

func commandImportTranslatedQuestion() *cli.Command {
	return &cli.Command{
		Name: "import-question-translation",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "input",
				Value: "./quiz-translated.csv",
			},
			&cli.StringFlag{
				Name: "category",
			},
			&cli.StringFlag{
				Name: "language",
			},
		},
		Action: func(c *cli.Context) error {
			postgresDB, err := getDb()
			if err != nil {
				return err
			}

			inputPath := c.String("input")
			if _, err := os.Stat(inputPath); os.IsNotExist(err) {
				return err
			}

			file, err := os.Open(inputPath)
			if err != nil {
				return err
			}

			r := csv.NewReader(file)

			_, err = r.Read()
			if err != nil {
				return err
			}

			translatedQuestions := []models.QuestionTranslation{}

			category := c.String("category")
			language := c.String("language")

			for {
				record, err := r.Read()
				if err == io.EOF {
					break
				}
				if err != nil {
					return err
				}

				if len(record) != 8 {
					log.Println("invalid record length", record[0])
					continue
				}

				answerStr := strings.TrimSpace(record[6])
				if len(answerStr) != 1 {
					log.Println("invalid record answer", record[0])
					continue
				}

				choices := []*models.Choice{{Content: record[2], Key: 0}, {Content: record[3], Key: 1}, {Content: record[4], Key: 2}, {Content: record[5], Key: 3}}

				id, _ := strconv.Atoi(record[0])
				translatedQuestions = append(translatedQuestions, models.QuestionTranslation{
					QuestionBankID: id,
					LanguageCode:   language,
					Question:       record[1],
					Choices:        choices,
					Category:       category,
					Enabled:        true,
				})
			}

			ctx := c.Context

			for _, question := range translatedQuestions {
				err := datastore.SetQuestionTranslation(ctx, postgresDB, &question)
				if err != nil {
					fmt.Println("Error when set question", err)
					continue
				}
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
