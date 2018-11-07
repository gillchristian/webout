package types

import (
	"time"
)

type Channel struct {
	ID        string    `json:"id"`    // uuid - created by postres
	Token     string    `json:"token"` // uuid - created by postres
	CreatedAt time.Time `json:"createdAt"`
}
