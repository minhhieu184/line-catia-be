package models

import (
	"time"

	"github.com/uptrace/bun"
)

// TODO: save full question for tracking?
type QuestionHistory struct {
	bun.BaseModel `bun:"table:question_history"`
	GameSessionID int        `bun:"game_session_id,pk" json:"game_session_id"`
	Index         int        `bun:"index,pk" json:"index"`
	TotalScore    int        `bun:"total_score" json:"total_score"`
	QuestionID    int        `bun:"question_id" json:"question_id"`
	QuestionScore int        `bun:"question_score" json:"question_score"`
	StartedAt     time.Time  `bun:"started_at" json:"started_at"`
	Answer        *int       `bun:"answer" json:"answer"`
	AnsweredAt    *time.Time `bun:"answered_at" json:"answered_at"`
	Correct       *bool      `bun:"correct" json:"correct"`
	CorrectAnswer *int       `bun:"-" json:"correct_answer"`
	Checkpoint    *int       `bun:"checkpoint" json:"checkpoint"`

	Question Question `bun:"-" json:"-"`
}

type GameSession struct {
	bun.BaseModel      `bun:"table:game_session"`
	ID                 int        `bun:"id,pk,autoincrement" json:"id"`
	LegacyID           string     `bun:"legacy_id" json:"legacy_id"`
	GameSlug           string     `bun:"game_slug" json:"game_slug"`
	UserID             int64      `bun:"user_id" json:"user_id"`
	Score              int        `bun:"score" json:"score"`
	BonusScore         int        `bun:"bonus_score" json:"bonus_score"`
	TotalScore         int        `bun:"total_score" json:"total_score"`
	QuestionStartedAt  *time.Time `bun:"question_started_at" json:"question_started_at"`
	StartedAt          *time.Time `bun:"started_at" json:"started_at"`
	EndedAt            *time.Time `bun:"ended_at" json:"ended_at"`
	StreakPoint        int        `bun:"streak_point" json:"streak_point"`
	UsedBoostCount     int        `bun:"used_boost_count" json:"used_boost_count"`
	CorrectAnswerCount int        `bun:"correct_answer_count" json:"correct_answer_count"`

	NextStep             int                     `bun:"-" json:"next_step"`
	CurrentQuestion      *Question               `bun:"-" json:"current_question"`
	CurrentQuestionScore int                     `bun:"-" json:"current_question_score"`
	History              map[int]QuestionHistory `bun:"-" json:"history"`
}

type GameSessionParams struct {
	RefCode int64 `json:"refCode"`
}

type UserGameSessionSumary struct {
	TotalScore            int `bun:"total_score" json:"total_score"`
	CorrectAnswerCount    int `bun:"correct_answer_count" json:"correct_answer_count"`
	MaxCorrectAnswerCount int `bun:"max_correct_answer_count" json:"max_correct_answer_count"`
	SessionCount          int `bun:"session_count" json:"session_count"`
}
