package persistence

import "context"

type DB interface {
	Ping(context.Context) error
	Close() error
}

type InMemoryDB struct{}

func NewInMemoryDB() *InMemoryDB                  { return &InMemoryDB{} }
func (db *InMemoryDB) Ping(context.Context) error { return nil }
func (db *InMemoryDB) Close() error               { return nil }
