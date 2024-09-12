package datastore

import (
	"context"
	"millionaire/internal/models"

	"github.com/uptrace/bun"
)

func CreateTableUserWallet(ctx context.Context, db *bun.DB) error {
	_, err := db.NewCreateTable().Model((*models.UserWallet)(nil)).IfNotExists().Exec(ctx)
	if err != nil {
		return err
	}

	_, err = db.NewRaw(`
		alter table user_wallet
			add if not exists ton_wallet text;`).Exec(ctx)

	if err != nil {
		return err
	}

	_, err = db.NewCreateIndex().Model((*models.UserWallet)(nil)).Index("index_user_wallet_evm").Unique().IfNotExists().Column("evm_wallet").Exec(ctx)
	if err != nil {
		return err
	}

	_, err = db.NewCreateIndex().Model((*models.UserWallet)(nil)).Index("index_user_wallet_ton").Unique().IfNotExists().Column("ton_wallet").Exec(ctx)
	if err != nil {
		return err
	}
	return nil
}

func FindUserWalletByEVMWallet(ctx context.Context, db *bun.DB, evmWallet string) (*models.UserWallet, error) {
	var userWallet models.UserWallet
	err := db.NewSelect().Model(&userWallet).Where("evm_wallet = ?", evmWallet).Scan(ctx)
	if err != nil {
		return nil, err
	}
	return &userWallet, nil
}

func FindUserWalletByUserID(ctx context.Context, db *bun.DB, userID string) (*models.UserWallet, error) {
	var userWallet models.UserWallet
	err := db.NewSelect().Model(&userWallet).Where("id = ?", userID).Scan(ctx)
	if err != nil {
		return nil, err
	}
	return &userWallet, nil
}

func CreateUserWallet(ctx context.Context, db *bun.DB, userWallet *models.UserWallet) (*models.UserWallet, error) {
	_, err := db.NewInsert().Model(userWallet).Exec(ctx)
	if err != nil {
		return nil, err
	}

	return userWallet, nil
}

func UpdateUserWallet(ctx context.Context, db *bun.DB, userWallet *models.UserWallet) (*models.UserWallet, error) {
	_, err := db.NewUpdate().Model(userWallet).WherePK().Exec(ctx)
	if err != nil {
		return nil, err
	}

	return userWallet, nil
}
