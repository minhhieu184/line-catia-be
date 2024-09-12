package models

import (
	"time"

	"github.com/uptrace/bun"
)

type UserWallet struct {
	bun.BaseModel `bun:"table:user_wallet"`
	ID            string     `bun:"id,pk" json:"id"`
	EVMWallet     *string   `bun:"evm_wallet" json:"evm_wallet"`
	TONWallet     *string   `bun:"ton_wallet" json:"ton_wallet"`
	CreatedAt     time.Time `bun:"created_at" json:"created_at"`
	UpdatedAt     time.Time `bun:"updated_at" json:"updated_at"`
}
