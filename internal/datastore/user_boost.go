package datastore

import (
	"context"
	"millionaire/internal/models"
	"time"

	"github.com/uptrace/bun"
)

func CreateTableUserBoost(ctx context.Context, db *bun.DB) error {
	_, err := db.NewCreateTable().Model((*models.UserBoost)(nil)).IfNotExists().Exec(ctx)
	if err != nil {
		return err
	}

	_, err = db.NewRaw(`
		alter table user_boost
			add if not exists validated bool default false;
		alter table user_boost
   			alter column used set default false;
		alter table user_boost
    		alter column created_at set default current_timestamp;`).Exec(ctx)
	if err != nil {
		return err
	}

	_, err = db.NewCreateIndex().Model((*models.UserBoost)(nil)).Index("index_user_boost_user_id_used_validated").IfNotExists().Column("user_id", "used", "validated").Exec(ctx)
	if err != nil {
		return err
	}

	_, err = db.NewCreateIndex().Model((*models.UserBoost)(nil)).Index("index_user_boost_user_id_source_validated").IfNotExists().Column("user_id", "source", "validated").Exec(ctx)
	if err != nil {
		return err
	}

	_, err = db.NewCreateIndex().Model((*models.UserBoost)(nil)).Index("index_user_boost_user_id_source").Unique().IfNotExists().Column("user_id", "source").Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}

func CountUserBoosts(ctx context.Context, db *bun.DB, userID string) (int, error) {
	count, err := db.NewSelect().Model((*models.UserBoost)(nil)).Where("user_id = ?", userID).Where("used = false").Where("validated = true").Count(ctx)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func UseBoost(ctx context.Context, db *bun.DB, userId string, usedFor string) error {
	// Get the avaiable boost that used_at is null
	// TODO: lock the row
	var boost models.UserBoost
	err := db.NewSelect().Model(&boost).Where("user_id = ? and used = false and validated = true", userId).OrderExpr("created_at desc").Limit(1).Scan(ctx)
	if err != nil {
		return err
	}

	now := time.Now()

	// Update the boost used_at
	boost.UsedAt = &now
	boost.Used = true
	boost.UsedFor = usedFor

	_, err = db.NewUpdate().Model(&boost).WherePK().Where("used_at is null").Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}

func SetValidateUserBoost(ctx context.Context, db *bun.DB, userId string, source string) error {
	_, err := db.NewUpdate().Model((*models.UserBoost)(nil)).Set("validated = true").Where("user_id = ? and source = ? and validated = false", userId, source).Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}

func CreateBoost(ctx context.Context, db *bun.DB, userBoost *models.UserBoost) error {
	_, err := db.NewInsert().Model(userBoost).Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}

func CreateMultipleBoost(ctx context.Context, db *bun.DB, userBoosts []*models.UserBoost) error {
	_, err := db.NewInsert().Model(&userBoosts).Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}

func CheckUserBoostExists(ctx context.Context, db *bun.DB, userId string, source string) (bool, error) {
	var boost models.UserBoost
	err := db.NewSelect().Model(&boost).Where("user_id = ? and source = ?", userId, source).Scan(ctx)
	if err != nil {
		return false, err
	}

	return true, nil
}
