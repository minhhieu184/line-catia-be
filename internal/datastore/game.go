package datastore

import (
	"context"

	"millionaire/internal/models"

	"github.com/uptrace/bun"
)

func CreateTableGame(ctx context.Context, db *bun.DB) error {
	_, err := db.NewCreateTable().Model((*models.Game)(nil)).IfNotExists().Exec(ctx)
	if err != nil {
		return err
	}

	_, err = db.NewCreateIndex().Model((*models.Game)(nil)).Index("index_game_slug").Unique().IfNotExists().Column("slug").Exec(ctx)
	if err != nil {
		return err
	}

	_, err = db.NewRaw(`
		create sequence if not exists game_id_seq
			as int;	

		alter table game
			alter column id type int using id::int;

		alter table game
			alter column id set default nextval('public.game_id_seq'::regclass);

		alter sequence game_id_seq owned by game.id;`).Exec(ctx)
	if err != nil {
		return err
	}

	_, err = db.NewRaw(`
		alter table game
			add if not exists logo varchar;

		alter table game
			add if not exists start_time timestamp;

		alter table game
			add if not exists end_time timestamp;
		
		alter table game
			add if not exists enabled bool default false;

		alter table game
			add if not exists extra_setup jsonb default '[{"type":"nothing","description":"Good luck","chance":65},{"type":"plus_15","description":"Plus 15 points","chance":10},{"type":"minus_15","description":"Minus 15 points","chance":12},{"type":"plus_30","description":"Plus 30 points","chance":5},{"type":"half","description":"Halve score","chance":5},{"type":"double","description":"Double score","chance":2},{"type":"to_0","description":"Lose all","chance":2},{"type":"to_110","description":"Score upto 110 points","chance":1}]';
		
		alter table game
			add if not exists config jsonb;
			
		alter table game
    		add if not exists difficulty varchar;
			
		alter table game
    		add if not exists social_links jsonb;

		alter table game
    		add if not exists priority int default 0;
		
		alter table game
    		add if not exists bonus_privilege varchar;

		alter table game 
			add if not exists is_public bool default true;
		`).Exec(ctx)
	if err != nil {
		return err
	}

	_, _ = db.NewRaw(`
		alter table game
    		rename column enable to enabled;`).Exec(ctx)

	return nil
}

func GetGame(ctx context.Context, db *bun.DB, slug string) (*models.Game, error) {
	var game models.Game
	err := db.NewSelect().Model(&game).Where("slug = ?", slug).Where("enabled = ?", true).Scan(ctx)
	if err != nil {
		return nil, err
	}
	return &game, nil
}

func SetGame(ctx context.Context, db *bun.DB, game *models.Game) error {
	_, err := db.NewInsert().Model(game).Exec(ctx)
	if err != nil {
		return err
	}
	return nil
}

func GetEnabledGames(ctx context.Context, db *bun.DB) ([]models.Game, error) {
	var games []models.Game
	err := db.NewSelect().Model(&games).Where("enabled = ? and is_public = ?", true, true).Order("priority DESC").Scan(ctx)
	println("games LEN", len(games))
	if err != nil {
		return nil, err
	}
	return games, nil
}

func GetGameConfig(ctx context.Context, db *bun.DB, slug string) ([]models.GameConfig, error) {
	game, err := GetGame(ctx, db, slug)
	if err != nil {
		return nil, err
	}

	return game.Config, nil
}
