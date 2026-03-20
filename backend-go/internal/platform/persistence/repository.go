package persistence

import "context"

type Repository[T any] interface {
	GetByID(context.Context, string) (T, bool, error)
	Save(context.Context, T) error
	Delete(context.Context, string) error
}
