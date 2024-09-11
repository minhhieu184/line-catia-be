package models

import (
	"github.com/uptrace/bun"
)

type Config struct {
	bun.BaseModel `bun:"table:config"`
	Key           string `bun:"key,pk" json:"key"`
	Value         string `bun:"value" json:"value"`
}
