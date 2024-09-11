package datastore

import (
	"context"
	"millionaire/internal/models"

	"github.com/uptrace/bun"
)

func CreateTableReward(ctx context.Context, db *bun.DB) error {
	_, err := db.NewCreateTable().Model((*models.Reward)(nil)).IfNotExists().Exec(ctx)
	if err != nil {
		return err
	}

	_, err = db.NewCreateIndex().Model((*models.Reward)(nil)).Index("index_reward_user_id").IfNotExists().Column("user_id").Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}

func GetAvaiableRewardByUserID(ctx context.Context, db *bun.DB, userID int64) ([]models.Reward, error) {
	var rewards []models.Reward
	err := db.NewSelect().Model(&rewards).
		Where("user_id = ?", userID).
		Where("claimed = ?", false).Scan(ctx)
	if err != nil {
		return nil, err
	}

	return rewards, nil
}

func ClaimReward(ctx context.Context, db *bun.DB, rewardId int) error {
	_, err := db.NewUpdate().Model((*models.Reward)(nil)).
		Set("claimed = ?", true).
		Where("id=?", rewardId).Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}
