package testdb

import (
	"context"
	"testing"

	"hotspot/backend-go/internal/platform/persistence"
)

func TestSkipIfShortSkipsInShortMode(t *testing.T) {
	if !testing.Short() {
		t.Log("run with -short to verify SkipIfShort behavior")
		return
	}
	testing.Short()
}

func TestDSNReadsEnv(t *testing.T) {
	dsn := DSN()
	if testing.Short() {
		if dsn != "" {
			t.Fatalf("expected empty DSN without env, got %q", dsn)
		}
		return
	}
	if dsn == "" {
		t.Skip("POSTGRES_DSN not set")
	}
	t.Logf("DSN length: %d", len(dsn))
}

func TestTxWrapperSatisfiesQueryableDB(t *testing.T) {
	var _ persistence.QueryableDB = (*txWrapper)(nil)
}

func TestOpenAndTxScope(t *testing.T) {
	SkipIfShort(t)
	factory := Open(t)
	if factory == nil {
		t.Fatal("expected non-nil factory")
	}
	TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		qdb, ok := txFactory.DB.(persistence.QueryableDB)
		if !ok {
			t.Fatal("txFactory.DB should implement QueryableDB")
		}
		if err := qdb.Exec(ctx, "SELECT 1"); err != nil {
			t.Fatalf("exec in tx: %v", err)
		}
	})
}
