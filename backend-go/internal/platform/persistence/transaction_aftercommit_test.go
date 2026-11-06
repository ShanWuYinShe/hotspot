package persistence_test

import (
	"context"
	"testing"

	"hotspot/backend-go/internal/platform/aftercommit"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/testdb"
)

// RunInTransaction 的 aftercommit 语义：COMMIT 后回调执行一次、以事务前 ctx
// 执行（不含已关闭的事务句柄，能继续查库）；回滚时丢弃。
func TestRunInTransactionAfterCommitHooks(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	queryable, ok := factory.DB.(persistence.QueryableDB)
	if !ok {
		t.Fatal("expected QueryableDB backend")
	}

	var hookCtx context.Context
	hookRan := false
	err := persistence.RunInTransaction(context.Background(), factory, func(txCtx context.Context) error {
		if _, ok := persistence.QueryableFromContext(txCtx); !ok {
			t.Fatal("expected tx queryable inside transaction")
		}
		aftercommit.Register(txCtx, func(postCtx context.Context) {
			hookRan = true
			hookCtx = postCtx
			if _, ok := persistence.QueryableFromContext(postCtx); ok {
				t.Error("post-commit hook must not see the closed transaction queryable")
			}
			var one int
			if err := queryable.QueryRow(postCtx, "SELECT 1").Scan(&one); err != nil || one != 1 {
				t.Errorf("post-commit hook must be able to query with the ctx: one=%d err=%v", one, err)
			}
		})
		return queryable.Exec(txCtx, "SELECT 1")
	})
	if err != nil {
		t.Fatalf("run in transaction: %v", err)
	}
	if !hookRan || hookCtx == nil {
		t.Fatal("expected after-commit hook to run after COMMIT")
	}
}

func TestRunInTransactionDiscardsHooksOnRollback(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)

	hookRan := false
	err := persistence.RunInTransaction(context.Background(), factory, func(txCtx context.Context) error {
		aftercommit.Register(txCtx, func(context.Context) { hookRan = true })
		return context.Canceled
	})
	if err == nil {
		t.Fatal("expected rollback error")
	}
	if hookRan {
		t.Fatal("after-commit hook must not run when transaction rolls back")
	}
}
