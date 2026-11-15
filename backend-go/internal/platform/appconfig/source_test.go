package appconfig

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/aftercommit"
	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/persistence"

	"github.com/jackc/pgx/v5"
)

func TestNewStaticSourceUsesRuntimeDefaults(t *testing.T) {
	src := NewStaticSource(config.Config{
		SupportEmail:       "support@example.com",
		PrivacyPolicyURL:   "https://example.com/privacy",
		TermsOfServiceURL:  "https://example.com/tos",
		OfficialWebsiteURL: "https://example.com",
		DefaultUserAvatar:  "user.png",
		DefaultGroupAvatar: "group.png",
	})
	cfg := src.Get(context.Background())
	if cfg.Contact.SupportEmail != "support@example.com" {
		t.Fatalf("unexpected support email: %s", cfg.Contact.SupportEmail)
	}
	if cfg.Defaults.DefaultUserAvatar != "user.png" {
		t.Fatalf("unexpected default user avatar: %s", cfg.Defaults.DefaultUserAvatar)
	}
	if cfg.Defaults.DefaultGroupAvatar != "group.png" {
		t.Fatalf("unexpected default group avatar: %s", cfg.Defaults.DefaultGroupAvatar)
	}
}

func TestStaticSourceUpdateAppliesFunction(t *testing.T) {
	src := NewStaticSource(config.Config{})
	updated, err := src.Update(context.Background(), func(cfg AppConfig) AppConfig {
		cfg.Features.EnableMoments = false
		cfg.Contact.SupportEmail = "new@example.com"
		return cfg
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Features.EnableMoments {
		t.Fatal("expected moments disabled")
	}
	if updated.Contact.SupportEmail != "new@example.com" {
		t.Fatalf("unexpected support email: %s", updated.Contact.SupportEmail)
	}
	got := src.Get(context.Background())
	if got.Contact.SupportEmail != "new@example.com" {
		t.Fatalf("expected persisted update, got %s", got.Contact.SupportEmail)
	}
}

func TestStaticSourceGetReturnsClone(t *testing.T) {
	src := NewStaticSource(config.Config{})
	cfg := src.Get(context.Background())
	cfg.Upload.AllowedImageTypes[0] = "mutated/type"
	again := src.Get(context.Background())
	if again.Upload.AllowedImageTypes[0] == "mutated/type" {
		t.Fatal("expected clone isolation for slices")
	}
}

func TestCloneConfigCopiesSlices(t *testing.T) {
	cfg := AppConfig{Upload: UploadConfig{AllowedImageTypes: []string{"a", "b"}, AllowedVideoTypes: []string{"v1"}}}
	cloned := cloneConfig(cfg)
	cloned.Upload.AllowedImageTypes[0] = "changed"
	cloned.Upload.AllowedVideoTypes[0] = "changed-video"
	if cfg.Upload.AllowedImageTypes[0] != "a" {
		t.Fatalf("original image slice mutated: %#v", cfg.Upload.AllowedImageTypes)
	}
	if cfg.Upload.AllowedVideoTypes[0] != "v1" {
		t.Fatalf("original video slice mutated: %#v", cfg.Upload.AllowedVideoTypes)
	}
}

func TestNullableString(t *testing.T) {
	if got := nullableString(""); got != nil {
		t.Fatalf("expected nil, got %#v", got)
	}
	if got := nullableString("x"); got != "x" {
		t.Fatalf("expected x, got %#v", got)
	}
}

func TestParseTimeOrNow(t *testing.T) {
	parsed := parseTimeOrNow("2025-01-02T03:04:05Z")
	if parsed.UTC().Format(time.RFC3339) != "2025-01-02T03:04:05Z" {
		t.Fatalf("unexpected parsed time: %s", parsed.UTC().Format(time.RFC3339))
	}
	fallback := parseTimeOrNow("not-a-time")
	if fallback.IsZero() {
		t.Fatal("expected fallback current time")
	}
}

type mockQueryableDB struct {
	queryRowFunc func(ctx context.Context, sql string, args ...any) persistence.Row
	execFunc     func(ctx context.Context, sql string, args ...any) error
}

func (m *mockQueryableDB) Ping(context.Context) error { return nil }
func (m *mockQueryableDB) Close() error               { return nil }

func (m *mockQueryableDB) Exec(ctx context.Context, sql string, args ...any) error {
	if m.execFunc != nil {
		return m.execFunc(ctx, sql, args...)
	}
	return nil
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

func TestPersistenceSourceGetAndLoadFromDB(t *testing.T) {
	ctx := context.Background()

	// 1. 无 Factory 回退到默认配置
	pSourceNoFactory := NewPersistenceSource(config.Config{SupportEmail: "no-fac@example.com"}, nil)
	cfg := pSourceNoFactory.Get(ctx)
	if cfg.Contact.SupportEmail != "no-fac@example.com" {
		t.Fatalf("expected fallback support email, got %s", cfg.Contact.SupportEmail)
	}

	// 2. DB 查询返回 ErrNoRows，回退到默认配置
	mockDB := &mockQueryableDB{
		queryRowFunc: func(ctx context.Context, sql string, args ...any) persistence.Row {
			return &mockRow{err: pgx.ErrNoRows}
		},
	}
	fac := &persistence.Factory{
		Config: persistence.Config{PostgresDSN: "postgres://mock"},
		DB:     mockDB,
	}
	pSource := NewPersistenceSource(config.Config{SupportEmail: "default@example.com"}, fac)
	cfg = pSource.Get(ctx)
	if cfg.Contact.SupportEmail != "default@example.com" {
		t.Fatalf("expected default email when no row, got %s", cfg.Contact.SupportEmail)
	}

	// 3. DB 查询返回有效 JSON 数据
	storedCfg := defaultConfig(config.Config{})
	storedCfg.Contact.SupportEmail = "db-stored@example.com"
	storedCfg.Limits.MaxGroupMembers = 888
	payloadBytes, _ := json.Marshal(storedCfg)

	mockDB.queryRowFunc = func(ctx context.Context, sql string, args ...any) persistence.Row {
		return &mockRow{
			scanFunc: func(dest ...any) error {
				if len(dest) >= 3 {
					*dest[0].(*string) = string(payloadBytes)
					*dest[1].(*time.Time) = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
					*dest[2].(*string) = "admin_user_42"
				}
				return nil
			},
		}
	}
	pSourceLoaded := NewPersistenceSource(config.Config{}, fac)
	cfg = pSourceLoaded.Get(ctx)
	if cfg.Contact.SupportEmail != "db-stored@example.com" {
		t.Fatalf("expected loaded email db-stored@example.com, got %s", cfg.Contact.SupportEmail)
	}
	if cfg.Limits.MaxGroupMembers != 888 {
		t.Fatalf("expected MaxGroupMembers 888, got %d", cfg.Limits.MaxGroupMembers)
	}
	if cfg.UpdatedByUserID != "admin_user_42" {
		t.Fatalf("expected UpdatedByUserID admin_user_42, got %s", cfg.UpdatedByUserID)
	}
}

func TestPersistenceSourceUpdateAndErrors(t *testing.T) {
	ctx := context.Background()

	var execSQL string
	var execArgs []any

	mockDB := &mockQueryableDB{
		queryRowFunc: func(ctx context.Context, sql string, args ...any) persistence.Row {
			return &mockRow{err: pgx.ErrNoRows}
		},
		execFunc: func(ctx context.Context, sql string, args ...any) error {
			execSQL = sql
			execArgs = args
			return nil
		},
	}
	fac := &persistence.Factory{
		Config: persistence.Config{PostgresDSN: "postgres://mock"},
		DB:     mockDB,
	}

	pSource := NewPersistenceSource(config.Config{}, fac)
	updated, err := pSource.Update(ctx, func(c AppConfig) AppConfig {
		c.Features.EnableMoments = false
		c.UpdatedByUserID = "updater_1"
		return c
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.Features.EnableMoments {
		t.Fatal("expected EnableMoments to be false")
	}
	if execSQL != queryAppConfigUpsert {
		t.Fatalf("unexpected upsert SQL: %s", execSQL)
	}
	if len(execArgs) < 4 || execArgs[0] != appConfigConfigID {
		t.Fatalf("unexpected execArgs: %+v", execArgs)
	}

	// 验证 Update 在 DB 执行报错时向上抛出错误
	mockDB.execFunc = func(ctx context.Context, sql string, args ...any) error {
		return errors.New("db write timeout")
	}
	_, err = pSource.Update(ctx, func(c AppConfig) AppConfig {
		c.Features.EnableGuestRegister = false
		return c
	})
	if err == nil {
		t.Fatal("expected error on DB exec failure, got nil")
	}
}

func TestPersistenceSourceRetryBackoff(t *testing.T) {
	ctx := context.Background()

	queryAttempts := 0
	dbErr := errors.New("connection refused")

	mockDB := &mockQueryableDB{
		queryRowFunc: func(ctx context.Context, sql string, args ...any) persistence.Row {
			queryAttempts++
			return &mockRow{err: dbErr}
		},
	}
	fac := &persistence.Factory{
		Config: persistence.Config{PostgresDSN: "postgres://mock"},
		DB:     mockDB,
	}

	pSource := NewPersistenceSource(config.Config{SupportEmail: "fallback@example.com"}, fac)

	// 第一次调用：执行 tryLoad，失败并记录 loadErr
	cfg := pSource.Get(ctx)
	if cfg.Contact.SupportEmail != "fallback@example.com" {
		t.Fatalf("expected fallback email, got %s", cfg.Contact.SupportEmail)
	}
	if queryAttempts != 1 {
		t.Fatalf("expected 1 attempt, got %d", queryAttempts)
	}

	// 第二次立即调用：处于 30s 冷却期内，不应重复查询 DB
	_ = pSource.Get(ctx)
	if queryAttempts != 1 {
		t.Fatalf("expected no additional query during backoff window, got %d", queryAttempts)
	}

	// 人工模拟 30 秒冷却已过
	pSource.loadMu.Lock()
	pSource.lastLoadFailure = time.Now().Add(-35 * time.Second)
	pSource.loadMu.Unlock()

	// 恢复 DB 正常响应
	storedCfg := defaultConfig(config.Config{})
	storedCfg.Contact.SupportEmail = "recovered@example.com"
	payloadBytes, _ := json.Marshal(storedCfg)
	dbErr = nil
	mockDB.queryRowFunc = func(ctx context.Context, sql string, args ...any) persistence.Row {
		queryAttempts++
		return &mockRow{
			scanFunc: func(dest ...any) error {
				if len(dest) >= 3 {
					*dest[0].(*string) = string(payloadBytes)
					*dest[1].(*time.Time) = time.Now().UTC()
					*dest[2].(*string) = "admin_reconnect"
				}
				return nil
			},
		}
	}

	cfg = pSource.Get(ctx)
	if queryAttempts != 2 {
		t.Fatalf("expected retry attempt after backoff expired, got %d", queryAttempts)
	}
	if cfg.Contact.SupportEmail != "recovered@example.com" {
		t.Fatalf("expected recovered email, got %s", cfg.Contact.SupportEmail)
	}
}

func TestLoadFromDBMalformedJSON(t *testing.T) {
	ctx := context.Background()
	mockDB := &mockQueryableDB{
		queryRowFunc: func(ctx context.Context, sql string, args ...any) persistence.Row {
			return &mockRow{
				scanFunc: func(dest ...any) error {
					*dest[0].(*string) = "{invalid json content"
					*dest[1].(*time.Time) = time.Now().UTC()
					*dest[2].(*string) = "admin"
					return nil
				},
			}
		},
	}
	_, ok, err := loadFromDB(ctx, mockDB)
	if err == nil || ok {
		t.Fatalf("expected error on malformed json, got ok=%v, err=%v", ok, err)
	}
}

// ── 第 81 轮审计批次 4 残留：config.update 事务化 ────────────────────────────
//
// 以下用例锁定 PersistenceSource 的 ctx 事务感知：
//   1. Update/tryLoad 在 ctx 已有事务时复用该事务（而非另取池连接），使 admin
//      的「配置写 + config.update 审计」落在同一事务；
//   2. 事务内的配置 upsert 不在 COMMIT 前更新进程内快照——回滚（审计失败）时
//      不会把未提交配置缓存进内存。

// txRecordingQueryable 是 ctx 事务连接替身：记录经事务执行的 SQL，并按需
// 返回预设行（模拟事务内读取到的已提交配置）。
type txRecordingQueryable struct {
	execSQL  []string
	execArgs [][]any
	rowFunc  func() persistence.Row
}

func (q *txRecordingQueryable) Exec(_ context.Context, sql string, args ...any) error {
	q.execSQL = append(q.execSQL, sql)
	q.execArgs = append(q.execArgs, append([]any(nil), args...))
	return nil
}

func (q *txRecordingQueryable) QueryRow(context.Context, string, ...any) persistence.Row {
	if q.rowFunc != nil {
		return q.rowFunc()
	}
	return &mockRow{err: pgx.ErrNoRows}
}

// Query 只为满足 persistence.Queryable（WithTx 入参）——本包不使用多行查询。
func (q *txRecordingQueryable) Query(context.Context, string, ...any) (persistence.Rows, error) {
	return nil, persistence.ErrNotImplemented
}

func TestPersistenceSourceUpdateJoinsContextTx(t *testing.T) {
	ctx := context.Background()
	poolExecSQL := ""
	mockDB := &mockQueryableDB{
		queryRowFunc: func(context.Context, string, ...any) persistence.Row { return &mockRow{err: pgx.ErrNoRows} },
		execFunc: func(_ context.Context, sql string, _ ...any) error {
			poolExecSQL = sql
			return nil
		},
	}
	fac := &persistence.Factory{Config: persistence.Config{PostgresDSN: "postgres://mock"}, DB: mockDB}
	src := NewPersistenceSource(config.Config{}, fac)

	tx := &txRecordingQueryable{}
	txCtx := persistence.WithTx(ctx, tx, nil)
	updated, err := src.Update(txCtx, func(c AppConfig) AppConfig {
		c.Contact.SupportEmail = "tx@example.com"
		return c
	})
	if err != nil {
		t.Fatalf("update in tx: %v", err)
	}
	if updated.Contact.SupportEmail != "tx@example.com" {
		t.Fatalf("updated support email = %q", updated.Contact.SupportEmail)
	}
	// 事务内 upsert 必须落在 tx 连接上，池连接一次都不应被写。
	if poolExecSQL != "" {
		t.Fatalf("pool exec must not be used inside tx, got %q", poolExecSQL)
	}
	if len(tx.execSQL) != 1 || tx.execSQL[0] != queryAppConfigUpsert {
		t.Fatalf("tx exec = %#v, want one upsert", tx.execSQL)
	}
	if got := src.Get(ctx).Contact.SupportEmail; got != "tx@example.com" {
		t.Fatalf("fallback email = %q", got)
	}

	// 无事务时保持旧行为：走 factory 共享池。
	if _, err := src.Update(ctx, func(c AppConfig) AppConfig {
		c.Contact.SupportEmail = "pool@example.com"
		return c
	}); err != nil {
		t.Fatalf("update without tx: %v", err)
	}
	if poolExecSQL != queryAppConfigUpsert {
		t.Fatalf("pool exec sql = %q, want upsert", poolExecSQL)
	}
}

func TestPersistenceSourceGetLoadsThroughContextTx(t *testing.T) {
	poolQueries := 0
	mockDB := &mockQueryableDB{
		queryRowFunc: func(context.Context, string, ...any) persistence.Row {
			poolQueries++
			return &mockRow{err: errors.New("pool must not be queried inside tx")}
		},
	}
	fac := &persistence.Factory{Config: persistence.Config{PostgresDSN: "postgres://mock"}, DB: mockDB}
	storedCfg := defaultConfig(config.Config{})
	storedCfg.Contact.SupportEmail = "tx-loaded@example.com"
	payloadBytes, _ := json.Marshal(storedCfg)
	tx := &txRecordingQueryable{
		rowFunc: func() persistence.Row {
			return &mockRow{scanFunc: func(dest ...any) error {
				if len(dest) >= 3 {
					*dest[0].(*string) = string(payloadBytes)
					*dest[1].(*time.Time) = time.Now().UTC()
					*dest[2].(*string) = "admin_tx"
				}
				return nil
			}}
		},
	}
	src := NewPersistenceSource(config.Config{}, fac)
	cfg := src.Get(persistence.WithTx(context.Background(), tx, nil))
	if poolQueries != 0 {
		t.Fatalf("pool queried %d times inside tx", poolQueries)
	}
	if cfg.Contact.SupportEmail != "tx-loaded@example.com" {
		t.Fatalf("support email = %q, want tx-loaded@example.com", cfg.Contact.SupportEmail)
	}
	if cfg.UpdatedByUserID != "admin_tx" {
		t.Fatalf("updated by = %q", cfg.UpdatedByUserID)
	}
}

func TestPersistenceSourceDefersMemorySnapshotUntilCommit(t *testing.T) {
	mockDB := &mockQueryableDB{
		queryRowFunc: func(context.Context, string, ...any) persistence.Row { return &mockRow{err: pgx.ErrNoRows} },
	}
	fac := &persistence.Factory{Config: persistence.Config{PostgresDSN: "postgres://mock"}, DB: mockDB}
	src := NewPersistenceSource(config.Config{SupportEmail: "old@example.com"}, fac)
	tx := &txRecordingQueryable{}
	txCtx := persistence.WithTx(aftercommit.WithHooks(context.Background()), tx, nil)

	if _, err := src.Update(txCtx, func(c AppConfig) AppConfig {
		c.Contact.SupportEmail = "new@example.com"
		return c
	}); err != nil {
		t.Fatalf("update in tx: %v", err)
	}
	// 未 COMMIT：进程内快照必须仍是旧值（回滚时不会暴露未提交配置）。
	if got := src.Get(context.Background()).Contact.SupportEmail; got != "old@example.com" {
		t.Fatalf("uncommitted config leaked into memory snapshot: %q", got)
	}

	// COMMIT：提交后回调把新配置写入快照。
	aftercommit.Run(txCtx)
	if got := src.Get(context.Background()).Contact.SupportEmail; got != "new@example.com" {
		t.Fatalf("committed config missing from memory snapshot: %q", got)
	}

	// ROLLBACK：再次事务内更新后丢弃回调，快照保持提交后的值。
	rollbackCtx := persistence.WithTx(aftercommit.WithHooks(context.Background()), &txRecordingQueryable{}, nil)
	if _, err := src.Update(rollbackCtx, func(c AppConfig) AppConfig {
		c.Contact.SupportEmail = "rolled-back@example.com"
		return c
	}); err != nil {
		t.Fatalf("update in rollback tx: %v", err)
	}
	aftercommit.Discard(rollbackCtx)
	if got := src.Get(context.Background()).Contact.SupportEmail; got != "new@example.com" {
		t.Fatalf("rolled-back config leaked into memory snapshot: %q", got)
	}
}
