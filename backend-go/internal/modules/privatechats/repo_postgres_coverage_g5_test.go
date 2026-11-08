package privatechats

import (
	"context"
	"errors"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/testdb"
)

// g5ChatScanErrorRow 让 scanChatState 在首列扫描即失败。
type g5ChatScanErrorRow struct{}

func (g5ChatScanErrorRow) Scan(...any) error { return errors.New("scan failed") }

// g5ChatFailingRows 模拟 rows.Scan 失败（列类型/数量不匹配）。
type g5ChatFailingRows struct{ scanErr error }

func (r *g5ChatFailingRows) Next() bool        { return true }
func (r *g5ChatFailingRows) Scan(...any) error { return r.scanErr }
func (r *g5ChatFailingRows) Close() error      { return nil }
func (r *g5ChatFailingRows) Err() error        { return nil }

// g5ChatTailErrorRows 模拟迭代结束后 rows.Err() 非 nil。
type g5ChatTailErrorRows struct{ err error }

func (r *g5ChatTailErrorRows) Next() bool        { return false }
func (r *g5ChatTailErrorRows) Scan(...any) error { return nil }
func (r *g5ChatTailErrorRows) Close() error      { return nil }
func (r *g5ChatTailErrorRows) Err() error        { return r.err }

// g5ChatListRowsBackend 固定 List 查询返回的 rows，用于覆盖迭代错误分支。
type g5ChatListRowsBackend struct{ rows persistence.Rows }

func (b *g5ChatListRowsBackend) Exec(context.Context, string, ...any) error { return nil }
func (b *g5ChatListRowsBackend) QueryRow(context.Context, string, ...any) persistence.Row {
	return persistence.InMemoryRow{}
}
func (b *g5ChatListRowsBackend) Query(context.Context, string, ...any) (persistence.Rows, error) {
	return b.rows, nil
}

func TestCoverageG5PostgresEnsureChatNotConfigured(t *testing.T) {
	repo := NewPostgresRepository(&persistence.Factory{
		Config: persistence.Config{PostgresDSN: "postgres://example"},
		DB:     privateChatNonQueryableDB{},
	})
	if err := repo.EnsureChat(context.Background(), "a:b", "a", "b", time.Now().UTC()); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("expected ErrNotImplemented, got %v", err)
	}
}

// TestCoverageG5PostgresGetStateAndRestoreScanErrors 覆盖 GetState 与
// RestoreForUser 内部 getStateImpl 的扫描错误路径。
func TestCoverageG5PostgresGetStateAndRestoreScanErrors(t *testing.T) {
	backend := &chatStateStub{rowErr: errors.New("scan failed")}
	repo := NewPostgresRepository(newChatFactory(backend))

	if _, err := repo.GetState(context.Background(), "u1", "u2"); err == nil {
		t.Fatal("expected GetState scan error")
	}
	if err := repo.RestoreForUser(context.Background(), "u1", "u2", time.Now().UTC()); err == nil {
		t.Fatal("expected RestoreForUser scan error")
	}
}

func TestCoverageG5PostgresScanChatStateError(t *testing.T) {
	if _, err := scanChatState(g5ChatScanErrorRow{}); err == nil {
		t.Fatal("expected scan error")
	}
}

// TestCoverageG5PostgresUpsertAndGetStateFallbacks 覆盖未接库时
// upsertStateImpl/getStateImpl 的 ErrNotImplemented 分支，以及带
// visible_since/deleted_at 的 upsert 参数组装。
func TestCoverageG5PostgresUpsertAndGetStateFallbacks(t *testing.T) {
	ctx := context.Background()
	notConfigured := NewPostgresRepository(persistence.NewFactory(config.Config{}))
	if err := notConfigured.upsertStateImpl(ctx, "u1", ChatState{}); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("expected ErrNotImplemented for upsert, got %v", err)
	}
	if _, err := notConfigured.getStateImpl(ctx, "u1", "u2"); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("expected ErrNotImplemented for get state, got %v", err)
	}

	backend := &chatStateStub{}
	repo := NewPostgresRepository(newChatFactory(backend))
	visible := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	deleted := visible.Add(time.Hour)
	state := ChatState{ChatKey: "chat_1", IsMuted: true, IsPinned: true, VisibleSince: &visible, DeletedAt: &deleted}
	if err := repo.upsertStateImpl(ctx, "u1", state); err != nil {
		t.Fatalf("upsert state: %v", err)
	}
	if backend.execs != 1 {
		t.Fatalf("expected one upsert exec, got %d", backend.execs)
	}
}

// TestCoverageG5PostgresScanChatStateWithDeletedAt 覆盖 scanChatState 的
// deleted_at 非空映射分支。
func TestCoverageG5PostgresScanChatStateWithDeletedAt(t *testing.T) {
	now := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	deleted := now.Add(2 * time.Hour)
	backend := &chatStateStub{rowValues: []any{"chat_1", "u2", now, true, false, nil, deleted}}
	repo := NewPostgresRepository(newChatFactory(backend))

	state, err := repo.GetState(context.Background(), "u1", "u2")
	if err != nil {
		t.Fatalf("get state: %v", err)
	}
	if state.DeletedAt == nil || !state.DeletedAt.Equal(deleted) {
		t.Fatalf("deleted_at mapping: %v", state.DeletedAt)
	}
	if !state.IsMuted {
		t.Fatalf("expected is_muted=true, got %#v", state)
	}
}

// TestCoverageG5PostgresListRowsErrors 覆盖 List 迭代扫描失败与 rows.Err()
// 非空两个错误分支。
func TestCoverageG5PostgresListRowsErrors(t *testing.T) {
	scanRepo := NewPostgresRepository(newChatFactory(&g5ChatListRowsBackend{
		rows: &g5ChatFailingRows{scanErr: errors.New("scan failed")},
	}))
	if _, _, err := scanRepo.List(context.Background(), "u1", ListOptions{}); err == nil {
		t.Fatal("expected list scan error")
	}

	tailRepo := NewPostgresRepository(newChatFactory(&g5ChatListRowsBackend{
		rows: &g5ChatTailErrorRows{err: errors.New("rows tail failed")},
	}))
	if _, _, err := tailRepo.List(context.Background(), "u1", ListOptions{}); err == nil {
		t.Fatal("expected rows.Err error")
	}
}

// TestPostgresIT_CoverageG5ChatStateLifecycle 真实库覆盖会话状态行生命周期：
// EnsureChat 幂等 → GetOrCreate 复读创建时间 → 条件更新 mute/pin 互不覆盖
// → 删除可见性 → Restore 恢复 → PurgeUserData 清空本模块用户态。全部在事务
// 内执行并回滚，数据 ID 使用 privatechatscov_ 前缀。
func TestPostgresIT_CoverageG5ChatStateLifecycle(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		viewer := testdb.InsertUser(t, txFactory, "privatechatscov_g5_viewer", "Viewer")
		target := testdb.InsertUser(t, txFactory, "privatechatscov_g5_target", "Target")
		repo := NewPostgresRepository(txFactory)

		createdAt := time.Date(2026, 5, 2, 0, 0, 0, 0, time.UTC)
		chatKey := key(viewer, target)
		userA, userB := orderedChatUsers(viewer, target)
		if err := repo.EnsureChat(ctx, chatKey, userA, userB, createdAt); err != nil {
			t.Fatalf("ensure chat: %v", err)
		}
		// EnsureChat 幂等：重复写入不改变创建时间。
		if err := repo.EnsureChat(ctx, chatKey, userA, userB, createdAt.Add(time.Hour)); err != nil {
			t.Fatalf("ensure chat twice: %v", err)
		}

		state, err := repo.GetState(ctx, viewer, target)
		if err != nil {
			t.Fatalf("get state: %v", err)
		}
		if state.TargetID != target || !state.CreatedAt.Equal(createdAt) {
			t.Fatalf("unexpected state after ensure: %#v", state)
		}

		gotCreatedAt, err := repo.GetOrCreate(ctx, target, viewer)
		if err != nil {
			t.Fatalf("get or create existing chat: %v", err)
		}
		if !gotCreatedAt.Equal(createdAt) {
			t.Fatalf("get_or_create created_at: got %v want %v", gotCreatedAt, createdAt)
		}

		muted, pinned := true, true
		if err := repo.UpdateSettings(ctx, viewer, target, UpdateChatSettings{IsMuted: &muted}); err != nil {
			t.Fatalf("mute: %v", err)
		}
		if err := repo.UpdateSettings(ctx, viewer, target, UpdateChatSettings{IsPinned: &pinned}); err != nil {
			t.Fatalf("pin: %v", err)
		}
		state, err = repo.GetState(ctx, viewer, target)
		if err != nil {
			t.Fatalf("get state after settings: %v", err)
		}
		if !state.IsMuted || !state.IsPinned {
			t.Fatalf("conditional update must preserve mute and pin: %#v", state)
		}

		deletedAt := createdAt.Add(time.Hour)
		if err := repo.DeleteForUser(ctx, viewer, target, &deletedAt); err != nil {
			t.Fatalf("delete for user: %v", err)
		}
		state, err = repo.GetState(ctx, viewer, target)
		if err != nil {
			t.Fatalf("get state after delete: %v", err)
		}
		if state.DeletedAt == nil || !state.DeletedAt.Equal(deletedAt) {
			t.Fatalf("deleted_at: %v", state.DeletedAt)
		}

		restoreAt := deletedAt.Add(time.Hour)
		if err := repo.RestoreForUser(ctx, viewer, target, restoreAt); err != nil {
			t.Fatalf("restore for user: %v", err)
		}
		state, err = repo.GetState(ctx, viewer, target)
		if err != nil {
			t.Fatalf("get state after restore: %v", err)
		}
		if state.DeletedAt != nil {
			t.Fatalf("restore must clear deleted_at: %v", state.DeletedAt)
		}
		if state.VisibleSince == nil || !state.VisibleSince.Equal(restoreAt) {
			t.Fatalf("visible_since: %v", state.VisibleSince)
		}

		if err := repo.PurgeUserData(ctx, viewer); err != nil {
			t.Fatalf("purge user data: %v", err)
		}
		q := txFactory.DB.(persistence.QueryableDB)
		var stateRows int
		if err := q.QueryRow(ctx, `SELECT COUNT(*) FROM private_chat_user_state WHERE user_id = $1`, viewer).Scan(&stateRows); err != nil {
			t.Fatalf("count user state: %v", err)
		}
		if stateRows != 0 {
			t.Fatalf("expected user state purged, got %d rows", stateRows)
		}
		var chatRows int
		if err := q.QueryRow(ctx, `SELECT COUNT(*) FROM private_chats WHERE chat_key = $1`, chatKey).Scan(&chatRows); err != nil {
			t.Fatalf("count chat: %v", err)
		}
		if chatRows != 1 {
			t.Fatalf("conversation identity must be preserved, got %d rows", chatRows)
		}
	})
}
