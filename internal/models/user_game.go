package models

import (
	"time"

	"github.com/uptrace/bun"
)

type UserGame struct {
	bun.BaseModel         `bun:"table:user_game"`
	ID                    int64      `bun:"id,pk,autoincrement" json:"id"`
	UserID                int64      `bun:"user_id" json:"user_id"`
	GameSlug              string     `bun:"game_slug" json:"game_slug"`
	ExtraSessions         int        `bun:"extra_session" json:"extra_sessions"`
	Countdown             *time.Time `bun:"countdown" json:"countdown"`
	CurrentSessionID      *string    `bun:"current_session_id" json:"current_session_id"`
	Lifeline              *Lifeline  `bun:"lifeline,type:jsonb" json:"lifeline"`
	GiftPoints            int        `bun:"gift_points" json:"gift_points"`
	CurrentBonusMilestone int        `bun:"current_bonus_milestone" json:"current_bonus_milestone"`
	TotalSessions         int        `bun:"total_sessions" json:"total_sessions"`
	TotalScore            int        `bun:"total_score" json:"total_score"`
	CurrentStreak         int        `bun:"current_streak" json:"current_streak"`
	CreatedAt             time.Time  `bun:"created_at,default:current_timestamp" json:"-"`
	UpdatedAt             time.Time  `bun:"updated_at" json:"-"`

	HistorySessions []*GameSession `bun:"-" json:"history_sessions"`
	IsNew           bool           `bun:"-" json:"is_new"`
	GameStartTime   *time.Time     `bun:"-" json:"game_start_time"`
	GameEndTime     *time.Time     `bun:"-" json:"game_end_time"`
}

func (userGame *UserGame) CountdownEnded() bool {
	if userGame.Countdown == nil {
		return true
	}
	return time.Now().After(*userGame.Countdown)
}
