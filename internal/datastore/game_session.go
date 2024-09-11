package datastore

import (
	"context"
	"millionaire/internal/models"

	"github.com/uptrace/bun"
)

func CreateTableGameSession(ctx context.Context, db *bun.DB) error {
	_, err := db.NewCreateTable().Model((*models.GameSession)(nil)).IfNotExists().Exec(ctx)
	if err != nil {
		return err
	}

	_, err = db.NewCreateIndex().Model((*models.GameSession)(nil)).Index("index_game_session_legacy_id").Unique().IfNotExists().Column("legacy_id").Exec(ctx)
	if err != nil {
		return err
	}

	_, err = db.NewCreateIndex().Model((*models.GameSession)(nil)).Index("index_game_session_game_slug_user_id").IfNotExists().Column("game_slug", "user_id").Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}

func CreateTableQuestionHistory(ctx context.Context, db *bun.DB) error {
	_, err := db.NewCreateTable().Model((*models.QuestionHistory)(nil)).IfNotExists().Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}

func SaveGameSession(ctx context.Context, db *bun.DB, gameSession *models.GameSession) error {
	err := CreateGameSession(ctx, db, gameSession)
	if err != nil {
		return err
	}

	for index, questionHistory := range gameSession.History {
		questionHistory.GameSessionID = gameSession.ID
		questionHistory.Index = index
		err = CreateQuestionHistory(ctx, db, questionHistory)
		if err != nil {
			continue
		}
	}

	return nil
}

func CreateGameSession(ctx context.Context, db *bun.DB, gameSession *models.GameSession) error {
	_, err := db.NewInsert().Model(gameSession).Returning("*").Exec(ctx)
	if err != nil {
		return err
	}
	return nil
}

func GetGameListSessions(ctx context.Context, db *bun.DB, gameSlug string, userID int64) ([]models.GameSession, error) {
	var gameSessions []models.GameSession
	err := db.NewSelect().Model(&gameSessions).Where("user_id = ?", userID).Where("game_slug = ?", gameSlug).Scan(ctx)
	if err != nil {
		return nil, err
	}
	return gameSessions, nil
}

func GetGameSessionById(ctx context.Context, db *bun.DB, gameSessionID string) (*models.GameSession, error) {
	var gameSession models.GameSession
	err := db.NewSelect().Model(&gameSession).Where("id = ?", gameSessionID).Scan(ctx)
	if err != nil {
		return nil, err
	}
	return &gameSession, nil
}

func GetLastUserGameSession(ctx context.Context, db *bun.DB, gameSlug string, userID int64) (*models.GameSession, error) {
	var gameSession models.GameSession
	err := db.NewSelect().Model(&gameSession).Where("user_id = ?", userID).Where("game_slug = ?", gameSlug).Order("ended_at DESC NULLS LAST").Limit(1).Scan(ctx)
	if err != nil {
		return nil, err
	}
	return &gameSession, nil
}

func UpdateGameSession(ctx context.Context, db *bun.DB, gameSession *models.GameSession) error {
	_, err := db.NewUpdate().Model(gameSession).WherePK().Exec(ctx)
	if err != nil {
		return err
	}
	return nil
}

func CreateQuestionHistory(ctx context.Context, db *bun.DB, questionHistory models.QuestionHistory) error {
	_, err := db.NewInsert().Model(&questionHistory).Exec(ctx)
	if err != nil {
		return err
	}
	return nil
}

func GetQuestionHistory(ctx context.Context, db *bun.DB, gameSessionID string, index int) (*models.QuestionHistory, error) {
	var questionHistory models.QuestionHistory
	err := db.NewSelect().Model(&questionHistory).Where("game_session_id = ?", gameSessionID).Where("index = ?", index).Scan(ctx)
	if err != nil {
		return nil, err
	}
	return &questionHistory, nil
}

func GetQuestionHistories(ctx context.Context, db *bun.DB, gameSessionID string) ([]models.QuestionHistory, error) {
	var questionHistories []models.QuestionHistory
	err := db.NewSelect().Model(&questionHistories).Where("game_session_id = ?", gameSessionID).Scan(ctx)
	if err != nil {
		return nil, err
	}
	return questionHistories, nil
}

func UpdateQuestionHistory(ctx context.Context, db *bun.DB, questionHistory *models.QuestionHistory) error {
	_, err := db.NewUpdate().Model(questionHistory).WherePK().Exec(ctx)
	if err != nil {
		return err
	}
	return nil
}

func GetUserGameSessionSumary(ctx context.Context, db *bun.DB, gameSlug string, userID int64) (*models.UserGameSessionSumary, error) {
	var gameSessionSumary models.UserGameSessionSumary
	// rows, err := db.QueryContext(ctx, "select sum(total_score) total_score, count(*) session_count from game_session ere user_id=? and game_slug=?", userID, gameSlug)
	err := db.
		NewSelect().
		ColumnExpr("sum(total_score) total_score").
		ColumnExpr("sum(correct_answer_count) correct_answer_count").
		ColumnExpr("max(correct_answer_count) max_correct_answer_count").
		ColumnExpr("count(*) session_count").
		TableExpr("game_session").
		Where("user_id = ?", userID).
		Where("game_slug = ?", gameSlug).
		Scan(ctx, &gameSessionSumary)

	if err != nil {
		return nil, err
	}

	return &gameSessionSumary, nil
}
