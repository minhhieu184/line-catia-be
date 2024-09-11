package datastore

import (
	"context"
	"millionaire/internal/models"

	"github.com/uptrace/bun"
)

func CreateTableSocialTask(ctx context.Context, db *bun.DB) error {
	_, err := db.NewCreateTable().Model((*models.SocialTask)(nil)).IfNotExists().Exec(ctx)
	if err != nil {
		return err
	}

	_, err = db.NewCreateIndex().Model((*models.SocialTask)(nil)).Index("index_social_task_game_slug_question_index_session_index").Unique().IfNotExists().Column("game_slug", "question_index", "session_index").Exec(ctx)
	if err != nil {
		return err
	}

	_, err = db.NewCreateIndex().Model((*models.SocialTask)(nil)).Index("index_social_task_game_slug").IfNotExists().Column("game_slug").Exec(ctx)
	if err != nil {
		return err
	}

	_, err = db.NewRaw(`
		alter table social_task
			add if not exists logo varchar;
			
		alter table social_task
			add if not exists priority integer default 0;
			
		alter table social_task
			add if not exists title varchar;

		alter table social_task 
			add if not exists is_public bool default true;
		alter table social_task
			add if not exists description varchar;`).Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}

func GetAvailableSocialTask(ctx context.Context, db *bun.DB, gameSlug string) (*models.SocialTask, error) {
	var socialTask models.SocialTask
	err := db.NewSelect().Model(&socialTask).Where("game_slug = ?", gameSlug).Where("enabled=?", true).Scan(ctx)
	if err != nil {
		return nil, err
	}

	return &socialTask, nil
}

func GetSocialTask(ctx context.Context, db *bun.DB, gameSlug string) (*models.SocialTask, error) {
	var socialTask models.SocialTask
	err := db.NewSelect().Model(&socialTask).Where("game_slug = ?", gameSlug).Scan(ctx)
	if err != nil {
		return nil, err
	}

	return &socialTask, nil
}

func CreateSocialTask(ctx context.Context, db *bun.DB, socialTask *models.SocialTask) error {
	_, err := db.NewInsert().Model(socialTask).Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}

func GetAvailableSocialTasks(ctx context.Context, db *bun.DB) ([]models.SocialTask, error) {
	var socialTasks []models.SocialTask
	err := db.NewSelect().Model(&socialTasks).Where("enabled = true AND is_public = true").Scan(ctx)
	if err != nil {
		return nil, err
	}

	return socialTasks, nil
}

func GetAllSocialTasks(ctx context.Context, db *bun.DB) ([]models.SocialTask, error) {
	var socialTasks []models.SocialTask
	err := db.NewSelect().Model(&socialTasks).Scan(ctx)
	if err != nil {
		return nil, err
	}

	return socialTasks, nil
}
