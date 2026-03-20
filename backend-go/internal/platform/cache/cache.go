package cache

import (
	"context"
	"time"
)

type Entry struct {
	Value     []byte
	ExpiresAt time.Time
}

type Store interface {
	Get(context.Context, string) ([]byte, bool, error)
	Set(context.Context, string, []byte, time.Duration) error
	Delete(context.Context, string) error
}
