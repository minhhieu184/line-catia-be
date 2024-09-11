package datastore

import (
	"context"
	"millionaire/internal/models"
	"time"

	"github.com/uptrace/bun"
)

func CreateTableUserGem(ctx context.Context, db *bun.DB) error {
	_, err := db.NewCreateTable().Model((*models.UserGem)(nil)).IfNotExists().Exec(ctx)
	if err != nil {
		return err
	}

	_, err = db.NewCreateIndex().Model((*models.UserGem)(nil)).Index("index_user_gem_user_id").IfNotExists().Column("user_id").Exec(ctx)
	if err != nil {
		return err
	}

	_, err = db.NewCreateIndex().Model((*models.UserGem)(nil)).Index("index_user_gem_user_id_action").IfNotExists().Unique().Column("user_id", "action").Exec(ctx)
	if err != nil {
		return err
	}

	_, err = db.NewCreateIndex().Model((*models.UserGem)(nil)).Index("index_user_gem_created_at").IfNotExists().Column("created_at").Exec(ctx)
	if err != nil {
		return err
	}

	_, err = db.NewCreateIndex().Model((*models.UserGem)(nil)).Index("index_user_gem_action").IfNotExists().Column("action").Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}

func InsertUserGem(ctx context.Context, db *bun.DB, userGem *models.UserGem) error {
	_, err := db.NewInsert().Model(userGem).On("CONFLICT (user_id, action) DO NOTHING").Exec(ctx)
	if err != nil {
		return err
	}
	return nil
}

func GetUserTotalGem(ctx context.Context, db *bun.DB, userID int64) (int, error) {
	var totalGem models.TotalGem
	err := db.NewSelect().
		ColumnExpr("SUM(gems) as total_gems").
		ColumnExpr("user_id").
		TableExpr("user_gem").
		Where("user_id = ?", userID).
		GroupExpr("user_id").
		Scan(ctx, &totalGem)
	if err != nil {
		return 0, err
	}

	return totalGem.TotalGems, nil
}

func GetUserTotalGemFromTime(ctx context.Context, db *bun.DB, userID int64, from *time.Time) (int, error) {
	var totalGem models.TotalGem
	err := db.NewSelect().
		ColumnExpr("SUM(gems) as total_gems").
		ColumnExpr("user_id").
		TableExpr("user_gem").
		Where("user_id = ?", userID).
		Where("created_at >=?", from).
		GroupExpr("user_id").
		Scan(ctx, &totalGem)
	if err != nil {
		return 0, err
	}

	return totalGem.TotalGems, nil
}

func GetUserTotalGemListFromTime(ctx context.Context, db *bun.DB, from *time.Time, limit, offset int) ([]*models.TotalGem, error) {
	var totalGem []*models.TotalGem
	err := db.NewSelect().
		ColumnExpr("SUM(gems) as total_gems").
		ColumnExpr("user_id").
		TableExpr("user_gem").
		Where("created_at >=?", from).
		GroupExpr("user_id").
		OrderExpr("total_gems DESC").
		Limit(limit).
		Offset(offset).
		Scan(ctx, &totalGem)
	if err != nil {
		return nil, err
	}

	return totalGem, nil
}

func GetUserGemByAction(ctx context.Context, db *bun.DB, userID int64, action string) (*models.UserGem, error) {
	var userGem models.UserGem
	err := db.NewSelect().Model(&userGem).Where("user_id = ? AND action = ?", userID, action).Scan(ctx)
	if err != nil {
		return nil, err
	}

	return &userGem, nil
}

func CountByAction(ctx context.Context, db *bun.DB, action string) (int, error) {
	count, err := db.NewSelect().Model((*models.UserGem)(nil)).Where("action = ?", action).Count(ctx)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func CountByActionFromTime(ctx context.Context, db *bun.DB, action string, from *time.Time) (int, error) {
	count, err := db.NewSelect().Model((*models.UserGem)(nil)).Where("action = ?", action).Where("created_at >=?", from).Count(ctx)
	if err != nil {
		return 0, err
	}

	return count, nil
}
