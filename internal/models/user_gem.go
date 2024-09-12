package models

import (
	"time"

	"github.com/uptrace/bun"
)

type UserGem struct {
	bun.BaseModel `bun:"table:user_gem"`
	ID            int64     `bun:"id,pk,autoincrement" json:"id"`
	UserID        string    `bun:"user_id" json:"user_id"`
	Gems          int       `bun:"gems" json:"gems"`
	Action        string    `bun:"action" json:"action"`
	CreatedAt     time.Time `bun:"created_at,default:current_timestamp" json:"created_at"`
}

type TotalGem struct {
	UserID    string `json:"user_id"`
	TotalGems int    `json:"total_gems"`
}
