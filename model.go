package main

import (
	"github.com/uptrace/bun"
	"time"
)

type Message struct {
	bun.BaseModel `bun:"table:message"`
	ID            int64     `bun:"id,pk,autoincrement"`
	Message       string    `bun:"message"`
	Status        string    `bun:"status,notnull"`
	Details       string    `bun:"details,nullzero"`
	CreatedAt     time.Time `bun:",nullzero,notnull,default:current_timestamp"`
	UpdatedAt     bun.NullTime
}
