package datastore

import (
	"context"
	"millionaire/internal/models"

	"github.com/uptrace/bun"
)

func CreateTablePartner(ctx context.Context, db *bun.DB) error {
	_, err := db.NewCreateTable().Model((*models.Partner)(nil)).IfNotExists().Exec(ctx)
	if err != nil {
		return err
	}

	_, err = db.NewCreateIndex().Model((*models.Partner)(nil)).Index("index_partner_slug").IfNotExists().Unique().Column("slug").Exec(ctx)
	if err != nil {
		return err
	}

	_, err = db.NewCreateIndex().Model((*models.Partner)(nil)).Index("index_partner_api_key").IfNotExists().Unique().Column("api_key").Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}

func CreateNewPartner(ctx context.Context, db *bun.DB, partner models.Partner) error {
	_, err := db.NewInsert().Model(&partner).Exec(ctx)
	if err != nil {
		return err
	}
	return nil
}

func GetPartner(ctx context.Context, db *bun.DB, slug string) (*models.Partner, error) {
	var partner models.Partner
	err := db.NewSelect().Model(&partner).Where("slug = ?", slug).Scan(ctx)
	if err != nil {
		return nil, err
	}
	return &partner, nil
}

func FindPartnerByAPIKey(ctx context.Context, db *bun.DB, apiKey string) (*models.Partner, error) {
	var partner models.Partner
	err := db.NewSelect().Model(&partner).Where("api_key = ?", apiKey).Scan(ctx)
	if err != nil {
		return nil, err
	}

	return &partner, nil
}

func GetEnabledPartner(ctx context.Context, db *bun.DB) ([]models.Partner, error) {
	var partners []models.Partner
	err := db.NewSelect().Model(&partners).Where("enabled = ?", true).Scan(ctx)
	if err != nil {
		return partners, err
	}
	return partners, nil
}
