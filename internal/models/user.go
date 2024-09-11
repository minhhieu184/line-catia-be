package models

import (
	"time"

	"github.com/uptrace/bun"
)

type User struct {
	bun.BaseModel         `bun:"table:user"`
	ID                    int64      `bun:"id,pk" json:"id"`
	FirstName             string     `bun:"first_name" json:"first_name"`
	IsBot                 bool       `bun:"is_bot" json:"-"`
	IsPremium             bool       `bun:"is_premium" json:"-"`
	LastName              string     `bun:"last_name" json:"last_name"`
	Username              string     `bun:"username" json:"username"`
	LanguageCode          string     `bun:"language_code" json:"language_code"`
	PhotoURL              string     `bun:"photo_url" json:"photo_url"`
	CreatedAt             time.Time  `bun:"created_at,default:current_timestamp" json:"created_at"`
	UpdatedAt             time.Time  `bun:"updated_at" json:"updated_at"`
	InviterID             *int64     `bun:"inviter_id" json:"inviter_id"`
	TotalInvites          int64      `bun:"total_invites" json:"total_invites"`
	RefCode               *string    `bun:"ref_code" json:"ref_code"`
	ExtraSessions         int        `bun:"extra_session" json:"-"`           // deprecated, moved to UserGame
	Countdown             *time.Time `bun:"countdown" json:"-"`               // deprecated, moved to UserGame
	CurrentSessionID      *string    `bun:"current_session_id" json:"-"`      // deprecated, moved to UserGame
	Lifeline              *Lifeline  `bun:"lifeline,type:jsonb" json:"-"`     // deprecated, moved to UserGame
	GiftPoints            int        `bun:"gift_points" json:"gift_points"`   // deprecated, moved to UserGame
	CurrentBonusMilestone int        `bun:"current_bonus_milestone" json:"-"` // deprecated, moved to UserGame
	LifelineBalance       int        `bun:"lifeline_balance" json:"lifeline_balance"`
	Avatar                *string    `bun:"avatar" json:"avatar"`
	ChatStatus            *string    `bun:"chat_status" json:"chat_status"`

	Boosts           int      `bun:"-" json:"boosts"`
	IsWinner         bool     `bun:"-" json:"is_winner"`
	TotalScore       int      `bun:"-" json:"total_score"`
	EVMWallet        *string  `bun:"-" json:"evm_wallet"`
	TONWallet        *string  `bun:"-" json:"ton_wallet"`
	IsNewUser        bool     `bun:"-" json:"is_new_user"`
	AvailableRewards []Reward `bun:"-" json:"available_rewards"`
}

type Lifeline struct {
	ChangeQuestion bool `json:"change_question"`
	FiftyFifty     bool `json:"fifty_fifty"`
}

// UserFromAuth only use in middleware
type UserFromAuth struct {
	ID           int64  `json:"id"`
	FirstName    string `json:"first_name"`
	IsBot        bool   `json:"is_bot"`
	IsPremium    bool   `json:"is_premium"`
	LastName     string `json:"last_name"`
	Username     string `json:"username"`
	LanguageCode string `json:"language_code"`
	PhotoURL     string `json:"photo_url"`
}

type Friend struct {
	ID        int64  `bun:"id" json:"id"`
	FirstName string `bun:"first_name" json:"first_name"`
	LastName  string `bun:"last_name" json:"last_name"`
	Username  string `bun:"username" json:"username"`
	Gems      int    `bun:"gems" json:"gems"`
	Claimed   bool   `bun:"claimed" json:"claimed"`
	Validated bool   `bun:"validated" json:"validated"`
}

type Ref struct {
	ID       int64  `bun:"id" json:"id"`
	Username string `bun:"username" json:"username"`
	Count    int    `bun:"count" json:"count"`
}
