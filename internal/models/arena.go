package models

import (
	"time"

	"github.com/uptrace/bun"
)

type Arena struct {
	bun.BaseModel `bun:"table:arena"`
	ID            int64      `bun:"id,pk,autoincrement" json:"id"`
	Name          string     `bun:"name" json:"name"`
	Slug          string     `bun:"slug" json:"slug"`
	GameSlug      string     `bun:"game_slug" json:"game_slug"`
	Enabled       bool       `bun:"enabled" json:"enabled"`
	StartDate     *time.Time `bun:"start_date" json:"start_date"`
	EndDate       *time.Time `bun:"end_date" json:"end_date"`
	Rewards       any        `bun:"rewards,type:jsonb" json:"rewards"`
	Description   string     `bun:"description" json:"description"`
	Logo          string     `bun:"logo" json:"logo"`
	Banner        string     `bun:"banner" json:"banner"`
	Priority      int        `bun:"priority" json:"priority"`

	PaticipantsCount int64 `bun:"-" json:"participants_count"`
}

func (a *Arena) IsEnded() bool {
	return a.EndDate != nil && a.EndDate.Before(time.Now())
}
