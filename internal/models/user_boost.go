package models

import (
	"time"

	"github.com/uptrace/bun"
)

type UserBoost struct {
	bun.BaseModel `bun:"table:user_boost"`
	ID            int        `bun:"id,pk,autoincrement" json:"id"`
	UserID        int64      `bun:"user_id" json:"user_id"`
	UsedAt        *time.Time `bun:"used_at" json:"used_at"`
	CreatedAt     time.Time  `bun:"created_at,default:current_timestamp" json:"created_at"`
	Source        string     `bun:"source" json:"source"`
	Used          bool       `bun:"used,default:false" json:"used"`
	UsedFor       string     `bun:"used_for" json:"used_for"`
	Validated     bool       `bun:"validated,default:false" json:"validated"`
}

const ReduceTimeCountdown = "reduce-time-countdown"
