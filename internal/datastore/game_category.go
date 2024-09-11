package datastore

import (
	"context"
	"github.com/uptrace/bun"
	"millionaire/internal/models"
)

func CreateTableGameCategory(ctx context.Context, db *bun.DB) error {
	_, err := db.NewCreateTable().Model((*models.GameCategory)(nil)).IfNotExists().Exec(ctx)
	if err != nil {
		return err
	}

	_, err = db.NewCreateIndex().Model((*models.GameCategory)(nil)).Index("index_game_category_game_slug").Unique().IfNotExists().Column("game_slug", "category").Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}

func GetGameCategory(ctx context.Context, db *bun.DB, gameSlug string) ([]*models.GameCategory, error) {
	var gameCategories []*models.GameCategory
	err := db.NewSelect().Model(&gameCategories).Where("game_slug = ?", gameSlug).Scan(ctx)
	if err != nil {
		return nil, err
	}
	return gameCategories, nil
}

func SetGameCategory(ctx context.Context, db *bun.DB, gameCategory *models.GameCategory) error {
	_, err := db.NewInsert().Model(gameCategory).Exec(ctx)
	if err != nil {
		return err
	}
	return nil
}
