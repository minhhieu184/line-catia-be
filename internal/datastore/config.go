package datastore

import (
	"context"

	"github.com/uptrace/bun"
	"millionaire/internal/models"
)

func CreateTableConfig(ctx context.Context, db *bun.DB) error {
	_, err := db.NewCreateTable().Model((*models.Config)(nil)).IfNotExists().Exec(ctx)
	if err != nil {
		return err
	}
	return nil
}

func InsertConfig(ctx context.Context, db *bun.DB, config models.Config) error {
	_, err := db.NewInsert().Model(&config).Exec(ctx)
	if err != nil {
		return err
	}
	return nil
}

func GetConfigByKey(ctx context.Context, db *bun.DB, key string) (*models.Config, error) {
	var config models.Config
	err := db.NewSelect().Model(&config).Where("key = ?", key).Scan(ctx)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func EditConfig(ctx context.Context, db *bun.DB, config *models.Config) (*models.Config, error) {
	_, err := db.NewUpdate().Model(config).WherePK().Exec(ctx)
	if err != nil {
		return nil, err
	}

	return config, nil
}
