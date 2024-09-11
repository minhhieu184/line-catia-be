package datastore

import (
	"context"
	"millionaire/internal/models"
	"time"

	"github.com/uptrace/bun"
)

func CreateTableUserFreebies(ctx context.Context, db *bun.DB) error {
	_, err := db.NewCreateTable().Model((*models.UserFreebie)(nil)).IfNotExists().Exec(ctx)
	if err != nil {
		return err
	}

	_, err = db.NewCreateIndex().Model((*models.UserFreebie)(nil)).Index("index_user_freebie_user_id").IfNotExists().Column("user_id").Exec(ctx)
	if err != nil {
		return err
	}

	_, err = db.NewCreateIndex().Model((*models.UserFreebie)(nil)).Index("index_user_freebie_user_id_name").IfNotExists().Unique().Column("user_id", "name").Exec(ctx)
	if err != nil {
		return err
	}

	_, err = db.NewRaw(`
		alter table "user_freebie"
			add if not exists amount int default 0;`).Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}

func InsertUserFreebies(ctx context.Context, db *bun.DB, userFreebie *models.UserFreebie) error {
	_, err := db.NewInsert().Model(userFreebie).Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}

func InsertMultipleUserFreebies(ctx context.Context, db *bun.DB, userId int64) ([]*models.UserFreebie, error) {
	freebies := make([]*models.UserFreebie, 0)
	now := time.Now()
	for _, freebie := range models.Freebies {
		item := &models.UserFreebie{
			UserID:    userId,
			Name:      freebie.Name,
			Countdown: now,
			Action:    freebie.Action,
			Icon:      freebie.Icon,
			Amount:    freebie.Amount,
		}
		freebies = append(freebies, item)
	}

	_, err := db.NewInsert().Model(&freebies).On("conflict (user_id, name) DO nothing").Exec(ctx)
	if err != nil {
		return nil, err
	}

	return freebies, nil
}

func GetUserFreebies(ctx context.Context, db *bun.DB, userID int64, action string) (*models.UserFreebie, error) {
	var userFreebie models.UserFreebie
	err := db.NewSelect().Model(&userFreebie).Where("user_id = ?", userID).Where("action = ?", action).Scan(ctx)
	if err != nil {
		return nil, err
	}

	return &userFreebie, nil
}

func UpdateUserFreebies(ctx context.Context, db *bun.DB, userFreebie *models.UserFreebie) error {
	_, err := db.NewUpdate().Model(userFreebie).WherePK().Exec(ctx)
	if err != nil {
		return err
	}
	return nil
}

func GetAllUserFreebies(ctx context.Context, db *bun.DB, userID int64) ([]*models.UserFreebie, error) {
	var userFreebies []*models.UserFreebie
	err := db.NewSelect().Model(&userFreebies).Where("user_id = ?", userID).Scan(ctx)
	if err != nil {
		return nil, err
	}

	return userFreebies, nil
}
