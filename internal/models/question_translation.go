package models

import "github.com/uptrace/bun"

type QuestionTranslation struct {
	bun.BaseModel  `bun:"table:question_translation"`
	ID             int       `bun:"id,pk,autoincrement" json:"id"`
	QuestionBankID int       `bun:"question_bank_id" json:"question_bank_id"`
	LanguageCode   string    `bun:"language_code" json:"language_code"`
	Question       string    `bun:"question" json:"question,omitempty"`
	Choices        []*Choice `bun:"choices,type:jsonb" json:"choices,omitempty"`
	Category       string    `bun:"category" json:"category"`
	Enabled        bool      `bun:"enabled" json:"-"`
}
