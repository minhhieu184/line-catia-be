package datastore

import (
	"context"
	"millionaire/internal/models"

	"github.com/uptrace/bun"
)

func CreateTableLifelineHistory(ctx context.Context, db *bun.DB) error {
	_, err := db.NewCreateTable().Model((*models.LifelineHistory)(nil)).IfNotExists().Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}

func GetLifelineHistory(ctx context.Context, db *bun.DB, userID string) ([]models.LifelineHistory, error) {
	var history []models.LifelineHistory
	err := db.NewSelect().Model(&history).Where("user_id = ?", userID).Scan(ctx)
	if err != nil {
		return nil, err
	}
	return history, nil
}

func InsertLifelineHistory(ctx context.Context, db *bun.DB, history *models.LifelineHistory) error {
	_, err := db.NewInsert().Model(history).Exec(ctx)
	if err != nil {
		return err
	}
	return nil
}
