package datastore

import (
	"context"
	"millionaire/internal/models"
	"time"

	"github.com/uptrace/bun"
)

func CreateTableUserGame(ctx context.Context, db *bun.DB) error {
	_, err := db.NewCreateTable().Model((*models.UserGame)(nil)).IfNotExists().Exec(ctx)
	if err != nil {
		return err
	}

	_, err = db.NewCreateIndex().Model((*models.UserGame)(nil)).Index("index_user_id_game_slug").Unique().IfNotExists().Column("user_id", "game_slug").Exec(ctx)
	if err != nil {
		return err
	}

	_, err = db.NewRaw(`
		alter table user_game
			add if not exists total_sessions int;

		alter table user_game
			add if not exists total_score int;

		alter table user_game
			add if not exists current_streak int;
		alter table user_game
			add if not exists created_at timestamp default current_timestamp;

		alter table user_game
			add if not exists updated_at timestamp;`).Exec(ctx)

	if err != nil {
		return err
	}

	return nil
}

func GetUserGame(ctx context.Context, db *bun.DB, userID int64, gameSlug string) (*models.UserGame, error) {
	var userGame models.UserGame
	err := db.NewSelect().Model(&userGame).Where("user_id = ?", userID).Where("game_slug = ?", gameSlug).Scan(ctx)
	if err != nil {
		return nil, err
	}
	return &userGame, nil
}

func SetUserGame(ctx context.Context, db *bun.DB, userGame *models.UserGame) error {
	_, err := db.NewInsert().Model(userGame).Exec(ctx)
	if err != nil {
		return err
	}
	return nil
}

func UpdateUserGame(ctx context.Context, db *bun.DB, userGame *models.UserGame) error {
	userGame.UpdatedAt = time.Now()
	_, err := db.NewUpdate().Model(userGame).Where("id = ?", userGame.ID).Exec(ctx)
	if err != nil {
		return err
	}
	return nil
}

func UpdateUserGameCountdown(ctx context.Context, db *bun.DB, userGame *models.UserGame) (*models.UserGame, error) {
	_, err := db.NewUpdate().Model(userGame).
		Set("countdown = ?", userGame.Countdown).
		Set("updated_at = current_timestamp").WherePK().Exec(ctx)
	if err != nil {
		return nil, err
	}

	return userGame, nil
}

func UpdateCountdownAndExtraSession(ctx context.Context, db *bun.DB, userGame *models.UserGame, countdownTime time.Time, extraSession int) (*models.UserGame, error) {
	_, err := db.NewUpdate().Model(userGame).
		Set("countdown = ?", countdownTime).
		Set("extra_session = ?", extraSession).
		Set("updated_at = current_timestamp").WherePK().Returning("*").Exec(ctx)
	if err != nil {
		return nil, err
	}

	return userGame, nil
}
