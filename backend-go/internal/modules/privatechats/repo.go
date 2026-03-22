package privatechats

import (
	"context"
	"time"
)

type Repository interface {
	GetOrCreate(context.Context, string, string) (time.Time, error)
	List(context.Context, string) (map[string]time.Time, error)
}
