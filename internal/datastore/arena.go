package datastore

import (
	"context"
	"millionaire/internal/models"

	"github.com/uptrace/bun"
)

func CreateTableArena(ctx context.Context, db *bun.DB) error {
	_, err := db.NewCreateTable().Model((*models.Arena)(nil)).IfNotExists().Exec(ctx)
	if err != nil {
		return err
	}

	_, err = db.NewCreateIndex().Model((*models.Arena)(nil)).Index("index_arena_slug").IfNotExists().Unique().Column("slug").Exec(ctx)
	if err != nil {
		return err
	}

	_, err = db.NewRaw(`
		alter table "arena"
			add if not exists banner varchar;
		
		alter table "arena"
			add if not exists priority int default 0;
		`).Exec(ctx)
	if err != nil {
		return err
	}

	_, err = db.NewCreateIndex().Model((*models.Arena)(nil)).Index("index_arena_slug").IfNotExists().Unique().Column("game_slug").Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}

func GetEnabledArenas(ctx context.Context, db *bun.DB) ([]models.Arena, error) {
	var arenas []models.Arena
	err := db.NewSelect().Model(&arenas).Where("enabled = ?", true).Scan(ctx)
	if err != nil {
		return nil, err
	}
	return arenas, nil
}

func GetArenaBySlug(ctx context.Context, db *bun.DB, slug string) (*models.Arena, error) {
	var arena models.Arena
	err := db.NewSelect().Model(&arena).Where("slug = ? and enabled = ?", slug, true).Scan(ctx)
	if err != nil {
		return nil, err
	}
	return &arena, nil
}

func GetArenaByGameSlug(ctx context.Context, db *bun.DB, gameSlug string) (*models.Arena, error) {
	var arena models.Arena
	err := db.NewSelect().Model(&arena).Where("game_slug = ? and enabled = ?", gameSlug, true).Scan(ctx)
	if err != nil {
		return nil, err
	}
	return &arena, nil
}
