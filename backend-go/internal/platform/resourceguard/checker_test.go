package resourceguard

import (
	"context"
	"errors"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/persistence"

	"github.com/jackc/pgx/v5"
)

type mockQueryableDB struct {
	queryRowFunc func(ctx context.Context, sql string, args ...any) persistence.Row
}

func (m *mockQueryableDB) Ping(context.Context) error { return nil }
func (m *mockQueryableDB) Close() error               { return nil }

func (m *mockQueryableDB) Exec(ctx context.Context, sql string, args ...any) error {
	return nil
}

func (m *mockQueryableDB) Query(ctx context.Context, sql string, args ...any) (persistence.Rows, error) {
	return nil, nil
}

func (m *mockQueryableDB) QueryRow(ctx context.Context, sql string, args ...any) persistence.Row {
	if m.queryRowFunc != nil {
		return m.queryRowFunc(ctx, sql, args...)
	}
	return &mockRow{err: pgx.ErrNoRows}
}

type mockRow struct {
	scanFunc func(dest ...any) error
	err      error
}

func (r *mockRow) Scan(dest ...any) error {
	if r.scanFunc != nil {
		return r.scanFunc(dest...)
	}
	return r.err
}

func TestNoopChecker(t *testing.T) {
	var c NoopChecker
	ctx := context.Background()
	if !c.Ready(ctx) {
		t.Fatal("expected NoopChecker to be ready")
	}
	allowed, err := c.MayExist(ctx, ResourceUser, "u1")
	if err != nil || !allowed {
		t.Fatalf("expected allowed=true, err=nil, got %v, %v", allowed, err)
	}
	if err := c.Record(ctx, ResourceUser, "u1"); err != nil {
		t.Fatalf("expected record err=nil, got %v", err)
	}
}

func TestDBCheckerLifecycleAndCache(t *testing.T) {
	ctx := context.Background()

	// 1. Ready 检查
	var nilChecker *DBChecker
	if nilChecker.Ready(ctx) {
		t.Fatal("nil checker should not be ready")
	}

	checkerWithoutDB := NewDBChecker(nil)
	if checkerWithoutDB.Ready(ctx) {
		t.Fatal("checker with nil db should not be ready")
	}

	// 2. 模拟 DB 交互
	queryCount := 0
	var dbErr error

	db := &mockQueryableDB{
		queryRowFunc: func(ctx context.Context, sql string, args ...any) persistence.Row {
			queryCount++
			if dbErr != nil {
				return &mockRow{err: dbErr}
			}
			return &mockRow{
				scanFunc: func(dest ...any) error {
					if len(dest) > 0 {
						if p, ok := dest[0].(*int); ok {
							*p = 1
						}
					}
					return nil
				},
			}
		},
	}

	checker := NewDBChecker(db)
	if !checker.Ready(ctx) {
		t.Fatal("checker with db should be ready")
	}

	// 空 ID 安全性
	allowed, err := checker.MayExist(ctx, ResourceUser, "")
	if err != nil || !allowed {
		t.Fatalf("empty id should return allowed=true, err=nil")
	}

	// 3. 查询存在资源
	queryCount = 0
	dbErr = nil
	allowed, err = checker.MayExist(ctx, ResourceUser, "u_exist")
	if err != nil || !allowed {
		t.Fatalf("expected user to exist, err=%v", err)
	}
	if queryCount != 1 {
		t.Fatalf("expected 1 db query, got %d", queryCount)
	}

	// 4. 查询不存在资源（pgx.ErrNoRows）并触发 markMissing 缓存
	queryCount = 0
	dbErr = pgx.ErrNoRows
	allowed, err = checker.MayExist(ctx, ResourceGroup, "g_missing")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if allowed {
		t.Fatal("expected allowed=false for missing group")
	}
	if queryCount != 1 {
		t.Fatalf("expected 1 db query, got %d", queryCount)
	}

	// 5. 再次查询相同缺失资源（命中 missing 缓存，不应再查询 DB）
	queryCount = 0
	allowed, err = checker.MayExist(ctx, ResourceGroup, "g_missing")
	if err != nil || allowed {
		t.Fatalf("expected cached allowed=false, err=nil, got allowed=%v, err=%v", allowed, err)
	}
	if queryCount != 0 {
		t.Fatalf("expected 0 db query due to missing cache, got %d", queryCount)
	}

	// 6. Record 该资源（主动清空 missing 缓存）
	_ = checker.Record(ctx, ResourceGroup, "g_missing")
	// 切换 DB 模拟资源被创建已存在
	dbErr = nil
	queryCount = 0
	allowed, err = checker.MayExist(ctx, ResourceGroup, "g_missing")
	if err != nil || !allowed {
		t.Fatalf("expected resource to exist after record and db creation, err=%v", err)
	}
	if queryCount != 1 {
		t.Fatalf("expected 1 db query after cache invalidated, got %d", queryCount)
	}

	// 7. DB 发生不可预料错误（如连接断开），应 fail-open（返回 true, err）保证业务韧性
	dbErr = errors.New("db network error")
	allowed, err = checker.MayExist(ctx, ResourceMoment, "m_error")
	if err == nil || !allowed {
		t.Fatalf("expected allowed=true and err returned on db failure, got allowed=%v, err=%v", allowed, err)
	}

	// 8. 缓存过期逻辑测试
	checker.mu.Lock()
	checker.missing[cacheKey(ResourceMoment, "m_expired")] = time.Now().UTC().Add(-1 * time.Second)
	checker.mu.Unlock()

	dbErr = nil
	queryCount = 0
	allowed, err = checker.MayExist(ctx, ResourceMoment, "m_expired")
	if err != nil || !allowed {
		t.Fatalf("expected allowed=true after expired cache re-query, err=%v", err)
	}
	if queryCount != 1 {
		t.Fatalf("expected re-query to db after cache expired, got %d", queryCount)
	}
}

func TestDefaultCheckerAndGlobalRecordHelpers(t *testing.T) {
	orig := DefaultChecker()
	defer func() { SetDefaultChecker(orig) }()

	// 测试 SetDefaultChecker(nil) 会回退为 NoopChecker
	SetDefaultChecker(nil)
	if DefaultChecker() == nil {
		t.Fatal("DefaultChecker should never return nil")
	}

	mockC := &mockCheckRecorder{recorded: map[string]ResourceKind{}}
	SetDefaultChecker(mockC)

	ctx := context.Background()
	RecordUser(ctx, "usr_1")
	RecordGroup(ctx, "grp_1")
	RecordMoment(ctx, "mom_1")

	if mockC.recorded["usr_1"] != ResourceUser {
		t.Fatalf("expected user recorded, got %v", mockC.recorded["usr_1"])
	}
	if mockC.recorded["grp_1"] != ResourceGroup {
		t.Fatalf("expected group recorded, got %v", mockC.recorded["grp_1"])
	}
	if mockC.recorded["mom_1"] != ResourceMoment {
		t.Fatalf("expected moment recorded, got %v", mockC.recorded["mom_1"])
	}
}

type mockCheckRecorder struct {
	NoopChecker
	recorded map[string]ResourceKind
}

func (m *mockCheckRecorder) Record(_ context.Context, kind ResourceKind, id string) error {
	m.recorded[id] = kind
	return nil
}

func TestQueryForKind(t *testing.T) {
	if q := queryForKind(ResourceUser); q != "SELECT 1 FROM users WHERE user_id = $1" {
		t.Fatalf("unexpected user query: %s", q)
	}
	if q := queryForKind(ResourceGroup); q != "SELECT 1 FROM groups WHERE group_id = $1" {
		t.Fatalf("unexpected group query: %s", q)
	}
	if q := queryForKind(ResourceMoment); q != "SELECT 1 FROM moments WHERE moment_id = $1" {
		t.Fatalf("unexpected moment query: %s", q)
	}
	if q := queryForKind("unknown"); q != "SELECT 1" {
		t.Fatalf("unexpected default query: %s", q)
	}
}
