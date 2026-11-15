package appconfig

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
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
	querySQL []string
	rowFunc  func() persistence.Row
}

func (q *txRecordingQueryable) Exec(_ context.Context, sql string, args ...any) error {
	q.execSQL = append(q.execSQL, sql)
	q.execArgs = append(q.execArgs, append([]any(nil), args...))
	return nil
}

func (q *txRecordingQueryable) QueryRow(_ context.Context, sql string, _ ...any) persistence.Row {
	q.querySQL = append(q.querySQL, sql)
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
	// 预加载进程快照（池上无配置行 → 快照为缺省配置），使后面的事务内 Update
	// 与读取不会触发 30s 重载而掩盖快照断言。
	_ = src.Get(ctx)

	// 事务内底稿必须来自 DB 行（FOR UPDATE 读到的最新已提交配置），而不是进程
	// 快照：快照里的 support_email 为空，DB 行里是 db@example.com。
	storedCfg := defaultConfig(config.Config{})
	storedCfg.Contact.SupportEmail = "db@example.com"
	storedCfg.Contact.PrivacyPolicyURL = "https://db.example/privacy"
	storedPayload, err := json.Marshal(storedCfg)
	if err != nil {
		t.Fatalf("marshal stored config: %v", err)
	}
	tx := &txRecordingQueryable{rowFunc: func() persistence.Row {
		return &mockRow{scanFunc: func(dest ...any) error {
			if len(dest) >= 3 {
				*dest[0].(*string) = string(storedPayload)
				*dest[1].(*time.Time) = time.Now().UTC()
				*dest[2].(*string) = "admin_tx"
			}
			return nil
		}}
	}}
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
	// DB 行独有字段必须保留：证明合并底稿是事务内 DB 行而非进程快照。
	if updated.Contact.PrivacyPolicyURL != "https://db.example/privacy" {
		t.Fatalf("merge base must be the locked DB row, got privacy_policy_url=%q", updated.Contact.PrivacyPolicyURL)
	}
	// 事务内 upsert 必须落在 tx 连接上，池连接一次都不应被写。
	if poolExecSQL != "" {
		t.Fatalf("pool exec must not be used inside tx, got %q", poolExecSQL)
	}
	// 底稿读取必须在事务内加行锁：FOR UPDATE 是并发/跨实例更新串行化的前提。
	if len(tx.querySQL) != 1 || !strings.Contains(tx.querySQL[0], "FOR UPDATE") {
		t.Fatalf("tx query = %#v, want one SELECT ... FOR UPDATE", tx.querySQL)
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

// txRowReturning 返回一个事务连接替身：其 FOR UPDATE 读返回 payload 对应的行
// （模拟事务内读到的最新已提交配置）。
func txRowReturning(t *testing.T, cfg AppConfig) *txRecordingQueryable {
	t.Helper()
	payload, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal tx row: %v", err)
	}
	return &txRecordingQueryable{rowFunc: func() persistence.Row {
		return &mockRow{scanFunc: func(dest ...any) error {
			if len(dest) >= 3 {
				*dest[0].(*string) = string(payload)
				*dest[1].(*time.Time) = time.Now().UTC()
				*dest[2].(*string) = "admin_tx"
			}
			return nil
		}}
	}}
}

func TestPersistenceSourceDefersMemorySnapshotUntilCommit(t *testing.T) {
	mockDB := &mockQueryableDB{
		queryRowFunc: func(context.Context, string, ...any) persistence.Row { return &mockRow{err: pgx.ErrNoRows} },
	}
	fac := &persistence.Factory{Config: persistence.Config{PostgresDSN: "postgres://mock"}, DB: mockDB}
	src := NewPersistenceSource(config.Config{SupportEmail: "old@example.com"}, fac)
	committed := defaultConfig(config.Config{SupportEmail: "old@example.com"})
	tx := txRowReturning(t, committed)
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
	rollbackBase := committed
	rollbackBase.Contact.SupportEmail = "new@example.com"
	rollbackCtx := persistence.WithTx(aftercommit.WithHooks(context.Background()), txRowReturning(t, rollbackBase), nil)
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

// TestPersistenceSourceUpdateInTxSeedsMissingRow：新库（app_configs 无行）首次
// 事务内更新时，先 INSERT ... ON CONFLICT DO NOTHING 落一行基线再 FOR UPDATE
// 重读，以基线为底稿合并——没有这一步，两个并发「首写」会各自以缺省配置为底稿
// 整份覆盖（行锁只对已存在的行生效）。
func TestPersistenceSourceUpdateInTxSeedsMissingRow(t *testing.T) {
	base := defaultConfig(config.Config{SupportEmail: "seed@example.com"})
	base.Limits.MaxGroupMembers = 500
	basePayload, err := json.Marshal(base)
	if err != nil {
		t.Fatalf("marshal base config: %v", err)
	}
	queries := 0
	tx := &txRecordingQueryable{
		rowFunc: func() persistence.Row {
			queries++
			if queries == 1 {
				return &mockRow{err: pgx.ErrNoRows} // 第一次 FOR UPDATE：行不存在
			}
			// 占位 INSERT 之后重读：返回刚落下的基线行。
			return &mockRow{scanFunc: func(dest ...any) error {
				if len(dest) >= 3 {
					*dest[0].(*string) = string(basePayload)
					*dest[1].(*time.Time) = time.Now().UTC()
					*dest[2].(*string) = ""
				}
				return nil
			}}
		},
	}
	// 带一个「查不到行」的池连接：让加载状态标记为已成功（lastLoadSuccess），
	// 后面的快照断言才不会被无 DB 场景下每次都重载 loadedCfg 的行为掩盖。
	mockDB := &mockQueryableDB{
		queryRowFunc: func(context.Context, string, ...any) persistence.Row { return &mockRow{err: pgx.ErrNoRows} },
	}
	fac := &persistence.Factory{Config: persistence.Config{PostgresDSN: "postgres://mock"}, DB: mockDB}
	src := NewPersistenceSource(config.Config{SupportEmail: "seed@example.com"}, fac)
	txCtx := persistence.WithTx(aftercommit.WithHooks(context.Background()), tx, nil)

	updated, err := src.Update(txCtx, func(c AppConfig) AppConfig {
		c.Limits.MaxGroupMembers = 4321
		return c
	})
	if err != nil {
		t.Fatalf("update in tx with missing row: %v", err)
	}
	if updated.Contact.SupportEmail != "seed@example.com" || updated.Limits.MaxGroupMembers != 4321 {
		t.Fatalf("config not merged onto seeded baseline: support_email=%q max_group_members=%d",
			updated.Contact.SupportEmail, updated.Limits.MaxGroupMembers)
	}
	if len(tx.execSQL) != 2 {
		t.Fatalf("tx exec = %#v, want seed insert + upsert", tx.execSQL)
	}
	if !strings.Contains(tx.execSQL[0], "INSERT INTO app_configs") || !strings.Contains(tx.execSQL[0], "DO NOTHING") {
		t.Fatalf("first exec should seed the missing row, got %q", tx.execSQL[0])
	}
	if tx.execSQL[1] != queryAppConfigUpsert {
		t.Fatalf("second exec = %q, want upsert", tx.execSQL[1])
	}
	if len(tx.querySQL) != 2 || !strings.Contains(tx.querySQL[1], "FOR UPDATE") {
		t.Fatalf("tx queries = %#v, want two FOR UPDATE reads", tx.querySQL)
	}
	// 提交前快照不更新（回滚不污染），提交后才写入。
	if got := src.Get(context.Background()).Limits.MaxGroupMembers; got == 4321 {
		t.Fatal("uncommitted config leaked into memory snapshot")
	}
	aftercommit.Run(txCtx)
	if got := src.Get(context.Background()).Limits.MaxGroupMembers; got != 4321 {
		t.Fatalf("committed config missing from memory snapshot: %d", got)
	}
}

// blockingTxQueryable 的 QueryRow 阻塞在 entered/release 通道上，用于模拟事务内
// 「SELECT ... FOR UPDATE 等行锁」。
type blockingTxQueryable struct {
	entered chan struct{}
	release chan struct{}
	payload []byte
	once    sync.Once
}

func (q *blockingTxQueryable) Exec(context.Context, string, ...any) error { return nil }

func (q *blockingTxQueryable) QueryRow(context.Context, string, ...any) persistence.Row {
	q.once.Do(func() { close(q.entered) })
	<-q.release
	return &mockRow{scanFunc: func(dest ...any) error {
		if len(dest) >= 3 {
			*dest[0].(*string) = string(q.payload)
			*dest[1].(*time.Time) = time.Now().UTC()
			*dest[2].(*string) = ""
		}
		return nil
	}}
}

func (q *blockingTxQueryable) Query(context.Context, string, ...any) (persistence.Rows, error) {
	return nil, persistence.ErrNotImplemented
}

// TestPersistenceSourceUpdateInTxDoesNotBlockReaders：事务内 Update 等行锁时不得
// 持有 loadMu——否则所有 AppConfig 读取（限流中间件每请求一次）都会排到行锁之后，
// 阻塞时长取决于对方事务持有时长。
func TestPersistenceSourceUpdateInTxDoesNotBlockReaders(t *testing.T) {
	payload, err := json.Marshal(defaultConfig(config.Config{}))
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	tx := &blockingTxQueryable{entered: make(chan struct{}), release: make(chan struct{}), payload: payload}
	src := NewPersistenceSource(config.Config{}, nil)
	txCtx := persistence.WithTx(aftercommit.WithHooks(context.Background()), tx, nil)

	updateDone := make(chan error, 1)
	go func() {
		_, updateErr := src.Update(txCtx, func(c AppConfig) AppConfig {
			c.Features.EnableMoments = false
			return c
		})
		updateDone <- updateErr
	}()
	select {
	case <-tx.entered:
	case <-time.After(5 * time.Second):
		t.Fatal("Update 未进入事务内底稿读取（FOR UPDATE）")
	}

	readDone := make(chan AppConfig, 1)
	go func() { readDone <- src.Get(context.Background()) }()
	select {
	case <-readDone:
		// 读取路径未被 Update 阻塞：符合预期。
	case <-time.After(2 * time.Second):
		close(tx.release)
		<-updateDone
		t.Fatal("AppConfig 读取被事务内 Update 阻塞：Update 等行锁时不应持有 loadMu")
	}

	close(tx.release)
	if err := <-updateDone; err != nil {
		t.Fatalf("update in tx: %v", err)
	}
}
