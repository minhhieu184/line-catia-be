package models

import (
	"time"

	"github.com/uptrace/bun"
)

type Reward struct {
	bun.BaseModel `bun:"table:reward"`
	ID            int                    `bun:"id,pk,autoincrement" json:"id"`
	Campaign      string                 `bun:"campaign" json:"campaign"`
	UserID        int64                  `bun:"user_id" json:"user_id"`
	Gem           int                    `bun:"gem" json:"gem"`
	Star          int                    `bun:"star" json:"star"`
	Lifeline      int                    `bun:"lifeline" json:"lifeline"`
	Claimed       bool                   `bun:"claimed" json:"claimed"`
	Metadata      map[string]interface{} `bun:"metadata,type:jsonb" json:"metadata"`
	CreatedAt     time.Time              `bun:"created_at,default:current_timestamp" json:"created_at"`
	UpdatedAt     time.Time              `bun:"updated_at" json:"updated_at"`
}
