package models

import (
	"strconv"

	"github.com/uptrace/bun"
)

type QuestionDifficulty string

const (
	QuestionEasy   = "easy"
	QuestionMedium = "medium"
	QuestionHard   = "hard"
)

func (v QuestionDifficulty) Valid() bool {
	switch v {
	case QuestionEasy, QuestionMedium, QuestionHard:
		return true
	default:
		return false
	}
}

type ToRandomQuestion struct {
	QuestionId int                `json:"id"`
	Difficulty QuestionDifficulty `json:"difficulty"`
}

// db
type Question struct {
	bun.BaseModel  `bun:"table:question"`
	ID             int                `bun:"id,pk,autoincrement" json:"id"`
	QuestionBankID int                `bun:"question_bank_id" json:"question_bank_id"`
	Question       string             `bun:"question" json:"question,omitempty"`
	Choices        []*Choice          `bun:"choices,type:jsonb" json:"choices,omitempty"`
	CorrectAnswer  int                `bun:"correct_answer" json:"-"`
	Difficulty     QuestionDifficulty `bun:"difficulty" json:"difficulty,omitempty"`
	Extra          bool               `bun:"-" json:"extra"`
	Category       string             `bun:"category" json:"category"`
	Enabled        bool               `bun:"enabled" json:"-"`

	Translations []*QuestionTranslation `bun:"-" json:"translations,omitempty"`
}

type QuestionLegacy struct {
	bun.BaseModel `bun:"table:question"`
	ID            string             `bun:"id,pk" json:"id"`
	Question      string             `bun:"question" json:"question,omitempty"`
	Choices       []*Choice          `bun:"choices,type:jsonb" json:"choices,omitempty"`
	CorrectAnswer int                `bun:"correct_answer" json:"-"`
	Difficulty    QuestionDifficulty `bun:"difficulty" json:"difficulty,omitempty"`
	Extra         bool               `bun:"-" json:"extra"`
	Category      string             `bun:"category" json:"category"`
}

func (q *QuestionLegacy) ToQuestion() *Question {
	id := 0
	if q.ID == "extra" {
		id = -1
	} else {
		id, _ = strconv.Atoi(q.ID)
	}
	return &Question{
		ID:             id,
		QuestionBankID: id,
		Question:       q.Question,
		Choices:        q.Choices,
		CorrectAnswer:  q.CorrectAnswer,
		Difficulty:     q.Difficulty,
		Extra:          q.Extra,
		Category:       q.Category,
	}
}

type Choice struct {
	Content string `json:"content"`
	Key     int    `json:"key"`
}
