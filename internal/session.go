package internal

import (
	"time"

	"millionaire/internal/models"
)

type GameScore struct {
	GameID                   string `json:"game_id"`
	UserID                   int64  `json:"user_id"`
	TotalSessions            int    `json:"total_sessions"`
	TotalScore               int    `json:"total_score"`
	TotalScoreCurrentSession int    `json:"total_score_current_session"`
	CurrentStreak            int    `json:"current_streak"`
	Milestone                int    `json:"milestone"`
	AchievedNewMilestone     bool   `json:"achieved_new_milestone"`
	LastMilestone            int    `json:"last_milestone"`
}

type QuestionHistory struct {
	TotalScore    int             `json:"total_score"`
	Question      models.Question `json:"-"`
	QuestionScore int             `json:"question_score"`
	StartedAt     time.Time       `json:"started_at"`
	Answer        *int            `json:"answer"`
	AnsweredAt    *time.Time      `json:"answered_at"`
	Correct       *bool           `json:"correct"`
	Checkpoint    *int            `json:"checkpoint"`
}

type QuestionHistoryLegacy struct {
	TotalScore    int                   `json:"total_score"`
	Question      models.QuestionLegacy `json:"question"`
	QuestionScore int                   `json:"question_score"`
	StartedAt     time.Time             `json:"started_at"`
	Answer        *int                  `json:"answer"`
	AnsweredAt    *time.Time            `json:"answered_at"`
	Correct       *bool                 `json:"correct"`
	Checkpoint    *int                  `json:"checkpoint"`
}

type GameSession struct {
	ID                   string                  `json:"id"`
	Key                  string                  `json:"-"`
	GameSlug             string                  `json:"game_id"`
	UserID               int64                   `json:"user_id"`
	NextStep             int                     `json:"next_step"`
	CurrentQuestion      *models.Question        `json:"current_question"`
	CurrentQuestionScore int                     `json:"current_question_score"`
	TotalScore           int                     `json:"total_score"`
	QuestionStartedAt    *time.Time              `json:"question_started_at"`
	StartedAt            *time.Time              `json:"started_at"`
	EndedAt              *time.Time              `json:"ended_at"`
	History              map[int]QuestionHistory `json:"history"`
	StreakPoint          int                     `json:"streak_point"`
	UsedBoostCount       int                     `json:"used_boost_count"`
}

type GameSessionLegacy struct {
	ID                   string                        `json:"id"`
	Key                  string                        `json:"-"`
	GameSlug             string                        `json:"game_id"`
	UserID               int64                         `json:"user_id"`
	NextStep             int                           `json:"next_step"`
	CurrentQuestion      *models.QuestionLegacy        `json:"current_question"`
	CurrentQuestionScore int                           `json:"current_question_score"`
	TotalScore           int                           `json:"total_score"`
	QuestionStartedAt    *time.Time                    `json:"question_started_at"`
	StartedAt            *time.Time                    `json:"started_at"`
	EndedAt              *time.Time                    `json:"ended_at"`
	History              map[int]QuestionHistoryLegacy `json:"history"`
	StreakPoint          int                           `json:"streak_point"`
}

func (session *GameSessionLegacy) ToGameSession() *GameSession {
	var currentQuestion *models.Question
	if session.CurrentQuestion != nil {
		currentQuestion = session.CurrentQuestion.ToQuestion()
	}
	gameSession := &GameSession{
		ID:                   session.ID,
		Key:                  session.Key,
		GameSlug:             session.GameSlug,
		UserID:               session.UserID,
		NextStep:             session.NextStep,
		CurrentQuestion:      currentQuestion,
		CurrentQuestionScore: session.CurrentQuestionScore,
		TotalScore:           session.TotalScore,
		QuestionStartedAt:    session.QuestionStartedAt,
		StartedAt:            session.StartedAt,
		EndedAt:              session.EndedAt,
		History:              make(map[int]QuestionHistory),
		StreakPoint:          session.StreakPoint,
		UsedBoostCount:       0,
	}

	for key, history := range session.History {
		gameSession.History[key] = QuestionHistory{
			TotalScore:    history.TotalScore,
			Question:      *history.Question.ToQuestion(),
			QuestionScore: history.QuestionScore,
			StartedAt:     history.StartedAt,
			Answer:        history.Answer,
			AnsweredAt:    history.AnsweredAt,
			Correct:       history.Correct,
			Checkpoint:    history.Checkpoint,
		}
	}

	return gameSession
}

func (session *GameSession) ToPostgresGameSession() *models.GameSession {
	gameSession := &models.GameSession{
		LegacyID:             session.ID,
		GameSlug:             session.GameSlug,
		UserID:               session.UserID,
		NextStep:             session.NextStep,
		CurrentQuestion:      session.CurrentQuestion,
		CurrentQuestionScore: session.CurrentQuestionScore,
		Score:                session.TotalScore,
		QuestionStartedAt:    session.QuestionStartedAt,
		StartedAt:            session.StartedAt,
		EndedAt:              session.EndedAt,
		History:              make(map[int]models.QuestionHistory),
		StreakPoint:          session.StreakPoint,
		UsedBoostCount:       0,
	}

	for key, history := range session.History {
		gameSession.History[key] = models.QuestionHistory{
			TotalScore:    history.TotalScore,
			QuestionID:    history.Question.ID,
			Question:      history.Question,
			QuestionScore: history.QuestionScore,
			StartedAt:     history.StartedAt,
			Answer:        history.Answer,
			AnsweredAt:    history.AnsweredAt,
			Correct:       history.Correct,
			Checkpoint:    history.Checkpoint,
		}
	}

	return gameSession
}

type GameSessionParams struct {
	RefCode int64 `json:"refCode"`
}
