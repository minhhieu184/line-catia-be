package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"millionaire/internal/datastore"
	"millionaire/internal/models"
	"os"

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
		Name: "migrate-game",
		Commands: []*cli.Command{
			commandGame(),
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func commandGame() *cli.Command {
	return &cli.Command{
		Name: "migrate-game",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name: "name",
			},
		},
		Action: func(c *cli.Context) error {
			ctx := context.Background()
			db, err := getDb()
			if err != nil {
				log.Fatal(err)
			}

			gameName := c.String("name")
			println("gameName", gameName)

			game := &models.Game{
				Name:        gameName,
				Slug:        gameName,
				Description: "",
				Questions:   nil,
				Checkpoints: nil,
				Logo:        "",
				Enabled:     true,
				Config:      nil,
				IsPublic:    true,
			}

			err = datastore.SetGame(ctx, db, game)
			if err != nil {
				log.Fatal(err)
			}

			fmt.Println("New Game created")

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
