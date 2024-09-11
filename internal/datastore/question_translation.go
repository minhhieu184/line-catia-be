package datastore

import (
	"context"
	"millionaire/internal/models"

	"github.com/uptrace/bun"
)

func CreateTableQuestionTranslation(ctx context.Context, db *bun.DB) error {
	_, err := db.NewCreateTable().Model((*models.QuestionTranslation)(nil)).IfNotExists().Exec(ctx)
	if err != nil {
		return err
	}

	_, err = db.NewCreateIndex().
		Model((*models.QuestionTranslation)(nil)).
		Index("index_question_translation_question_bank_id_language_code_category").
		Column("question_bank_id", "language_code", "category").
		Unique().IfNotExists().Exec(ctx)

	if err != nil {
		return err
	}

	_, err = db.NewCreateIndex().
		Model((*models.QuestionTranslation)(nil)).
		Index("index_question_translation_question_bank_id_category").
		Column("question_bank_id", "category").IfNotExists().Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}

func GetQuestionTranslation(ctx context.Context, db *bun.DB, questionBankId int, category string) ([]*models.QuestionTranslation, error) {
	var questions []*models.QuestionTranslation
	err := db.NewSelect().Model(&questions).
		Where("question_bank_id = ?", questionBankId).
		Where("category = ?", category).
		Where("enabled=?", true).Scan(ctx)
	if err != nil {
		return nil, err
	}
	return questions, nil

}

func SetQuestionTranslation(ctx context.Context, db *bun.DB, tq *models.QuestionTranslation) error {
	_, err := db.NewInsert().Model(tq).Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}
