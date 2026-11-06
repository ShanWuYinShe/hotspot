package persistence_test

import (
	"context"
	"testing"

	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/testdb"
)

// WarmPlanner：真实库上并发预热池连接（PGroonga 首查规划数据落在每个后端
// 连接上），行为幂等、失败可忽略。
func TestWarmPlannerWarmsPoolConnections(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	db, ok := factory.DB.(*persistence.PostgresDB)
	if !ok {
		t.Fatalf("expected PostgresDB, got %T", factory.DB)
	}
	if err := db.WarmPlanner(context.Background(), "SELECT 1 FROM users WHERE nickname &@ '__hotspot_warmup__' LIMIT 1", 4); err != nil {
		t.Fatalf("warm planner: %v", err)
	}
	if got := db.Pool.Stat().TotalConns(); got < 4 {
		t.Fatalf("expected >=4 warm pool connections, got %d", got)
	}
	// 再次调用幂等（多取同样的连接执行）。
	if err := db.WarmPlanner(context.Background(), "SELECT 1", 2); err != nil {
		t.Fatalf("second warm: %v", err)
	}
}

func TestWarmPlannerSkipsStubBackend(t *testing.T) {
	db := persistence.NewPostgresDB("")
	db.Backend = stubWarmBackend{}
	if err := db.WarmPlanner(context.Background(), "SELECT 1", 4); err != nil {
		t.Fatalf("stub backend warm planner must be no-op: %v", err)
	}
}

type stubWarmBackend struct{}

func (stubWarmBackend) Query(context.Context, string, ...any) (persistence.Rows, error) {
	return nil, nil
}
func (stubWarmBackend) QueryRow(context.Context, string, ...any) persistence.Row { return nil }
func (stubWarmBackend) Exec(context.Context, string, ...any) error               { return nil }
