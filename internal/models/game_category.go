package models

import "github.com/uptrace/bun"

type GameCategory struct {
	bun.BaseModel `bun:"table:game_category"`
	ID            int    `bun:"id,pk,autoincrement" json:"id"`
	GameSlug      string `bun:"game_slug" json:"game_slug"`
	Category      string `bun:"category" json:"category"`
}
