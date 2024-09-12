package models

import (
	"time"

	"github.com/uptrace/bun"
)

type LifelineHistory struct {
	bun.BaseModel `bun:"lifeline_history,alias:lifeline_history"`

	ID        int64     `bun:"id,pk,autoincrement" json:"id"`
	UserID    string    `bun:"user_id" json:"user_id"`
	Change    int       `bun:"change" json:"change"`
	Action    string    `bun:"action" json:"action"`
	Timestamp time.Time `bun:"timestamp,default:current_timestamp" json:"use_date"`
}
