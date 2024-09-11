package models

import "github.com/uptrace/bun"

type Partner struct {
	bun.BaseModel `bun:"table:partner"`
	ID            int64  `bun:"id,pk,autoincrement" json:"id"`
	Name          string `bun:"name" json:"name"`
	APIKey        string `bun:"api_key" json:"api_key"`
	Slug          string `bun:"slug" json:"slug"`
	Enabled       bool   `bun:"enabled" json:"enabled"`
}

type PartnerResponse struct {
	User bool `json:"user"`
	Gem  bool `json:"gem"`
	Ref  bool `json:"ref"`
}
