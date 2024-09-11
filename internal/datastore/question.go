package datastore

import (
	"context"
	"millionaire/internal/models"

	"github.com/uptrace/bun"
)

func CreateTableQuestion(ctx context.Context, db *bun.DB) error {
	_, err := db.NewCreateTable().Model((*models.Question)(nil)).IfNotExists().Exec(ctx)
	if err != nil {
		return err
	}

	_, err = db.NewRaw(`
		create sequence if not exists question_id_seq
			as int;

		alter table question
			alter column id type int using id::int;

		alter table question
			alter column id set default nextval('public.question_id_seq'::regclass);

		alter sequence question_id_seq owned by question.id;

		alter table question
			add if not exists question_bank_id int;
			
		alter table question
    		add if not exists enabled bool default true;`).Exec(ctx)

	if err != nil {
		return err
	}

	return nil
}

func GetQuestion(ctx context.Context, db *bun.DB, questionID int) (*models.Question, error) {
	var question models.Question
	err := db.NewSelect().Model(&question).Where("id = ?", questionID).Where("enabled = ?", true).Scan(ctx)
	if err != nil {
		return nil, err
	}
	return &question, nil
}

func GetQuestionIdsAndDifficulty(ctx context.Context, db *bun.DB, category []*models.GameCategory) ([]models.ToRandomQuestion, error) {
	var questions []models.Question

	var categories []string
	for _, c := range category {
		categories = append(categories, c.Category)
	}

	err := db.NewSelect().Model(&questions).Where("category IN (?)", bun.In(categories)).Where("enabled = ?", true).Scan(ctx)
	if err != nil {
		return nil, err
	}

	var toRandomQuestions []models.ToRandomQuestion
	for _, question := range questions {
		toRandomQuestions = append(toRandomQuestions, models.ToRandomQuestion{
			QuestionId: question.ID,
			Difficulty: question.Difficulty,
		})
	}
	return toRandomQuestions, nil
}

func SetQuestion(ctx context.Context, db *bun.DB, question *models.Question) error {
	_, err := db.NewInsert().Model(question).Ignore().Exec(ctx)
	if err != nil {
		return err
	}
	return nil
}
