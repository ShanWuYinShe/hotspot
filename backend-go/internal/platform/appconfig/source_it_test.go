package appconfig

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/testdb"

	"github.com/jackc/pgx/v5"
)

// appConfigRow 是 app_configs.default 行的原貌，供提交型 IT 结束后恢复现场。
type appConfigRow struct {
	payload   string
	updatedAt time.Time
	updatedBy *string
}

func readAppConfigRow(t *testing.T, qdb persistence.QueryableDB) (appConfigRow, bool) {
	t.Helper()
	var row appConfigRow
	err := qdb.QueryRow(context.Background(),
		"SELECT payload_json, updated_at, updated_by_user_id FROM app_configs WHERE config_id = $1",
		appConfigConfigID).Scan(&row.payload, &row.updatedAt, &row.updatedBy)
	if errors.Is(err, pgx.ErrNoRows) {
		return appConfigRow{}, false
	}
	if err != nil {
		t.Fatalf("read app_configs row: %v", err)
	}
	return row, true
}

// restoreAppConfigRow 把 app_configs.default 恢复为测试前状态（原不存在则删除）。
func restoreAppConfigRow(t *testing.T, qdb persistence.QueryableDB, row appConfigRow, exists bool) {
	t.Helper()
	ctx := context.Background()
	if !exists {
		if err := qdb.Exec(ctx, "DELETE FROM app_configs WHERE config_id = $1", appConfigConfigID); err != nil {
			t.Logf("cleanup app_configs row: %v", err)
		}
		return
	}
	var updatedBy any
	if row.updatedBy != nil {
		updatedBy = *row.updatedBy
	}
	if err := qdb.Exec(ctx, `INSERT INTO app_configs (config_id, payload_json, updated_at, updated_by_user_id)
VALUES ($1,$2,$3,$4)
ON CONFLICT (config_id) DO UPDATE
SET payload_json = EXCLUDED.payload_json,
    updated_at = EXCLUDED.updated_at,
    updated_by_user_id = EXCLUDED.updated_by_user_id`,
		appConfigConfigID, row.payload, row.updatedAt, updatedBy); err != nil {
		t.Logf("restore app_configs row: %v", err)
	}
}

// waitForAppConfigLockWait 等待出现「本库某后端在等 app_configs 行锁」：这是
// 并发用例确定性时序的前提（T2 必须先阻塞在行锁上，T1 才能提交）。只看
// wait_event_type 与 app_configs 关键字，新旧实现的阻塞语句（FOR UPDATE 读 /
// upsert）都能命中。
func waitForAppConfigLockWait(t *testing.T, qdb persistence.QueryableDB) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		var waiting int
		if err := qdb.QueryRow(context.Background(), `
SELECT COUNT(*) FROM pg_stat_activity
WHERE datname = current_database()
  AND wait_event_type = 'Lock'
  AND query ILIKE '%app_configs%'
  AND pid <> pg_backend_pid()`).Scan(&waiting); err != nil {
			t.Fatalf("query pg_stat_activity: %v", err)
		}
		if waiting > 0 {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("未观察到 T2 阻塞在 app_configs 行锁上：并发时序前提不成立")
}

// TestPostgresIT_ConcurrentUpdateMergesCommittedFields 覆盖第 82 轮审计 P1 的
// 并发丢更新回归：批次 5 把进程内快照更新推迟到 aftercommit 后，事务内 Update
// 仍以提交前的进程快照为底稿，两个并发事务（或 reloadTTL 内的多实例）会让后
// 提交者整份覆盖先提交者已写入的字段。
//
// 确定性时序：
//  1. 实例 A 的事务先 Update（持有 app_configs 行锁）且不提交；
//  2. 实例 B 的事务随后 Update 另一个分区，阻塞在行锁上；
//  3. A 提交后 B 才能继续，且必须基于 A 已提交的最新行合并；
//  4. 最终库内 payload 与 B 的进程快照同时包含两处改动。
func TestPostgresIT_ConcurrentUpdateMergesCommittedFields(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	qdb, ok := factory.DB.(persistence.QueryableDB)
	if !ok {
		t.Fatal("factory.DB is not persistence.QueryableDB")
	}
	ctx := context.Background()

	// 保存现场并在结束后恢复（提交型 IT 不得把测试配置留在共享库里）。
	original, originalExists := readAppConfigRow(t, qdb)
	t.Cleanup(func() { restoreAppConfigRow(t, qdb, original, originalExists) })

	base := defaultConfig(config.Config{SupportEmail: "old@example.com"})
	basePayload, err := json.Marshal(base)
	if err != nil {
		t.Fatalf("marshal baseline config: %v", err)
	}
	if err := qdb.Exec(ctx, `INSERT INTO app_configs (config_id, payload_json, updated_at, updated_by_user_id)
VALUES ($1,$2,now(),NULL)
ON CONFLICT (config_id) DO UPDATE
SET payload_json = EXCLUDED.payload_json, updated_at = now(), updated_by_user_id = NULL`,
		appConfigConfigID, string(basePayload)); err != nil {
		t.Fatalf("seed baseline config row: %v", err)
	}

	// 两个实例各持一份进程内快照：实例 B 在 T1 提交前完成加载（旧快照），
	// 模拟「多实例 reloadTTL 窗口内的过期快照」+ 同实例并发两种场景。
	runtimeCfg := config.Config{SupportEmail: "old@example.com"}
	sourceA := NewPersistenceSource(runtimeCfg, factory)
	sourceB := NewPersistenceSource(runtimeCfg, factory)
	if got := sourceB.Get(ctx); got.Contact.SupportEmail != "old@example.com" {
		t.Fatalf("实例 B 初始快照 support_email = %q", got.Contact.SupportEmail)
	}
	if got := sourceA.Get(ctx); !got.Features.EnableMoments {
		t.Fatal("实例 A 初始快照 enable_moments 应为 true")
	}

	t1Holding := make(chan struct{})
	t1Commit := make(chan struct{})
	t1Done := make(chan error, 1)
	go func() {
		t1Done <- persistence.RunInTransaction(ctx, factory, func(txCtx context.Context) error {
			if _, err := sourceA.Update(txCtx, func(cfg AppConfig) AppConfig {
				cfg.Features.EnableMoments = false
				return cfg
			}); err != nil {
				return err
			}
			close(t1Holding) // 已写库（持有行锁）且未提交
			<-t1Commit
			return nil
		})
	}()
	select {
	case <-t1Holding:
	case err := <-t1Done:
		t.Fatalf("T1 事务提前结束: %v", err)
	case <-time.After(10 * time.Second):
		t.Fatal("T1 未能在 10s 内完成事务内 Update")
	}

	// 实例 B 在另一个事务里改另一个分区：应先阻塞在行锁上。
	type updateResult struct {
		cfg AppConfig
		err error
	}
	t2Done := make(chan updateResult, 1)
	go func() {
		var res updateResult
		res.err = persistence.RunInTransaction(ctx, factory, func(txCtx context.Context) error {
			cfg, err := sourceB.Update(txCtx, func(cfg AppConfig) AppConfig {
				cfg.Contact.SupportEmail = "new@example.com"
				return cfg
			})
			res.cfg = cfg
			return err
		})
		t2Done <- res
	}()
	waitForAppConfigLockWait(t, qdb) // T2 已被行锁阻塞（确定性时序的关键断言）
	close(t1Commit)                  // 放行 T1 提交，T2 才能继续
	if err := <-t1Done; err != nil {
		t.Fatalf("T1 事务失败: %v", err)
	}
	select {
	case res := <-t2Done:
		if res.err != nil {
			t.Fatalf("T2 事务失败: %v", res.err)
		}
		// T2 的合并底稿必须是 T1 已提交的最新行：否则这里能看到 T1 字段丢失。
		if res.cfg.Features.EnableMoments {
			t.Fatal("T2 Update 未基于 T1 已提交配置合并：enable_moments 被整份覆盖为 true")
		}
	case <-time.After(10 * time.Second):
		t.Fatal("T1 提交后 T2 仍被阻塞超过 10s")
	}

	// 库内最终 payload 必须同时包含两个事务的字段。
	stored, exists := readAppConfigRow(t, qdb)
	if !exists {
		t.Fatal("app_configs.default 行不存在")
	}
	var final AppConfig
	if err := json.Unmarshal([]byte(stored.payload), &final); err != nil {
		t.Fatalf("unmarshal final payload: %v", err)
	}
	if final.Features.EnableMoments {
		t.Fatal("丢更新：T1 已提交的 enable_moments=false 在库内被 T2 覆盖")
	}
	if final.Contact.SupportEmail != "new@example.com" {
		t.Fatalf("库内 support_email = %q, want new@example.com", final.Contact.SupportEmail)
	}

	// T2 提交后 aftercommit 更新其进程快照：同样必须同时包含两处改动。
	snapshot := sourceB.Get(ctx)
	if snapshot.Features.EnableMoments {
		t.Fatal("丢更新：T2 的进程快照中 enable_moments 为 true")
	}
	if snapshot.Contact.SupportEmail != "new@example.com" {
		t.Fatalf("T2 进程快照 support_email = %q, want new@example.com", snapshot.Contact.SupportEmail)
	}
}
