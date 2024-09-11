package models

import (
	"github.com/uptrace/bun"
)

type SocialType string

const (
	SocialTypeTelegramChannel SocialType = "telegram_channel"
	SocialTypeTelegramGroup   SocialType = "telegram_group"
	SocialTypeTelegramApp     SocialType = "telegram_app"
	SocialTypeTwitter         SocialType = "twitter"
	SocialTypeTeletop         SocialType = "teletop"
)

type SocialTask struct {
	bun.BaseModel `bun:"table:social_task"`
	ID            int64   `bun:"id,pk,autoincrement" json:"id"`
	Title         string  `bun:"title" json:"title"`
	Description   *string `bun:"description" json:"description"`
	GameSlug      string  `bun:"game_slug" json:"game_slug"`
	Logo          string  `bun:"logo" json:"logo"`
	Priority      int     `bun:"priority" json:"priority"`
	QuestionIndex int     `bun:"question_index" json:"question_index"`
	SessionIndex  int     `bun:"session_index" json:"session_index"`
	Enabled       bool    `bun:"enabled" json:"-"`
	Links         []Link  `bun:"links,type:jsonb" json:"links"`
	IsPublic      bool    `bun:"is_public" json:"is_public"` //if is public = false -> it's in arena
}

type Link struct {
	ID          int        `json:"id"`
	LinkType    SocialType `json:"link_type"`
	Url         string     `json:"url"`
	RefUrl      string     `json:"ref_url"`
	Required    bool       `json:"required"`
	Joined      bool       `json:"joined"`
	Gem         int        `json:"star,omitempty"`
	Description string     `json:"description,omitempty"`
	Priority    int        `json:"priority,omitempty"`
}

type UserTask struct {
	ID        int64 `bun:"id,pk,autoincrement" json:"id"`
	UserID    int64 `bun:"user_id" json:"user_id"`
	TaskID    int64 `bun:"task_id" json:"task_id"`
	Completed bool  `bun:"completed" json:"completed"`
}
