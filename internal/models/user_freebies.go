package models

import (
	"time"

	"github.com/uptrace/bun"
)

const (
	ACTION_CLAIM_GEM      = "claim-gem"
	ACTION_CLAIM_STAR     = "claim-star"
	ACTION_CLAIM_LIFELINE = "claim-lifeline"

	NAME_GEM      = "freebie-gem"
	NAME_STAR     = "freebie-star"
	NAME_LIFELINE = "freebie-lifeline"

	ICON_GEM      = "https://cdn-icons-png.flaticon.com/512/541/541415.png"
	ICON_STAR     = "https://cdn-icons-png.flaticon.com/512/541/541415.png"
	ICON_LIFELINE = "https://cdn-icons-png.flaticon.com/512/541/541415.png"
)

var Freebies = []UserFreebie{
	{
		Name:   NAME_GEM,
		Action: ACTION_CLAIM_GEM,
		Icon:   ICON_GEM,
		Amount: 5,
	},
	{
		Name:   NAME_STAR,
		Action: ACTION_CLAIM_STAR,
		Icon:   ICON_STAR,
		Amount: 1,
	},
	{
		Name:   NAME_LIFELINE,
		Action: ACTION_CLAIM_LIFELINE,
		Icon:   ICON_LIFELINE,
		Amount: 1,
	},
}

type UserFreebie struct {
	bun.BaseModel `bun:"table:user_freebie"`
	ID            int64     `bun:"id,pk,autoincrement" json:"id"`
	UserID        int64     `bun:"user_id" json:"user_id"`
	Name          string    `bun:"name" json:"name"`
	Countdown     time.Time `bun:"countdown" json:"countdown"`
	Action        string    `bun:"action" json:"action"`
	Icon          string    `bun:"icon" json:"icon"`
	Amount        int       `bun:"amount" json:"amount"`
	ClaimSchedule int       `bun:"-" json:"claim_schedule"`
}

//type Freebie struct {
//	Name      string    `json:"name"`
//	Countdown time.Time `json:"countdown"`
//	Action    string    `json:"action"`
//}

type LastMessage struct {
	ID              int       `json:"id"`
	TimeLastMessage time.Time `json:"time_last_message"`
	FreebieName     string    `json:"freebie_name"`
}
