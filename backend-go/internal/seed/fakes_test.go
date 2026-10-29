package seed

import (
	"context"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/persistence"
)

// bcryptMinCostForTest 是单测里生成口令哈希的成本（最低边界，验证哈希可校验即可）。
const bcryptMinCostForTest = bcrypt.MinCost

// fakes_test.go — 单测替身：内存 Queryable（记录 SQL）与对象存储（记录键）。
// 目的是在不依赖真实 Postgres / SeaweedFS 的前提下验证：
//   - INSERT 语句形态（ON CONFLICT DO NOTHING / 参数化 / jsonb 转型）；
//   - --reset 的删除顺序与「先删后插」；
//   - 媒体对象 PUT / DELETE 的次数与键。

// execCall 记录一次 Exec。
type execCall struct {
	sql  string
	args []any
}

// fakeQueryable 是 persistence.Queryable 的内存实现。
type fakeQueryable struct {
	execs       []execCall
	storageKeys []string // Query 返回的 storage_key（--reset 清理对象用）
	queryErr    error
	queryCount  int
}

func (f *fakeQueryable) Exec(_ context.Context, sql string, args ...any) error {
	f.execs = append(f.execs, execCall{sql: sql, args: args})
	return nil
}

func (f *fakeQueryable) QueryRow(_ context.Context, _ string, _ ...any) persistence.Row {
	return fakeRow{}
}

func (f *fakeQueryable) Query(_ context.Context, _ string, _ ...any) (persistence.Rows, error) {
	f.queryCount++
	if f.queryErr != nil {
		return nil, f.queryErr
	}
	return &fakeRows{values: f.storageKeys}, nil
}

// statements 返回全部已执行 SQL（便于断言顺序）。
func (f *fakeQueryable) statements() []string {
	out := make([]string, 0, len(f.execs))
	for _, exec := range f.execs {
		out = append(out, exec.sql)
	}
	return out
}

// countStatements 统计包含 substr 的语句数量。
func (f *fakeQueryable) countStatements(substr string) int {
	count := 0
	for _, exec := range f.execs {
		if strings.Contains(exec.sql, substr) {
			count++
		}
	}
	return count
}

// firstExecIndex 返回第一条包含 substr 的语句下标，找不到返回 -1。
func (f *fakeQueryable) firstExecIndex(substr string) int {
	for index, exec := range f.execs {
		if strings.Contains(exec.sql, substr) {
			return index
		}
	}
	return -1
}

type fakeRow struct{}

func (fakeRow) Scan(...any) error { return nil }

type fakeRows struct {
	values []string
	index  int
}

func (r *fakeRows) Next() bool {
	if r.index >= len(r.values) {
		return false
	}
	r.index++
	return true
}

func (r *fakeRows) Scan(dest ...any) error {
	if len(dest) != 1 {
		return fmt.Errorf("fakeRows.Scan 期望 1 个目标，得到 %d", len(dest))
	}
	target, ok := dest[0].(*string)
	if !ok {
		return fmt.Errorf("fakeRows.Scan 目标必须是 *string")
	}
	*target = r.values[r.index-1]
	return nil
}

func (r *fakeRows) Close() error { return nil }
func (r *fakeRows) Err() error   { return nil }

// fakeStorage 是 ObjectStorage 的内存实现。
type fakeStorage struct {
	puts    []string
	deletes []string
	putErr  error
}

func (f *fakeStorage) PutObject(_ context.Context, storageKey, _ string, _ []byte) error {
	if f.putErr != nil {
		return f.putErr
	}
	f.puts = append(f.puts, storageKey)
	return nil
}

func (f *fakeStorage) Delete(_ context.Context, storageKey string) error {
	f.deletes = append(f.deletes, storageKey)
	return nil
}

// fakeTxRunner 在无真实事务的情况下把 fakeQueryable 注入 context
// （persistence.WithTx 是导出 API，repo/seed 共用同一取连接入口）。
func fakeTxRunner(queryable persistence.Queryable) txRunner {
	return func(ctx context.Context, fn func(ctx context.Context) error) error {
		return fn(persistence.WithTx(ctx, queryable, nil))
	}
}

// testPlanOptions 返回测试用的 Options：固定基准时间 + 最低 bcrypt 成本
// （单测只验证「哈希可校验」，成本与生产无关，min cost 让测试秒级完成）；
// 素材目录指向 TestMain 生成的临时目录（单测不依赖仓库 devdata/seed/media）。
func testPlanOptions() Options {
	return Options{
		Mode:       ModeIfMissing,
		Now:        testBaseTime,
		BcryptCost: bcryptMinCostForTest,
		AssetsDir:  testAssetsDirPath,
	}
}

// testBaseTime 是测试固定基准时刻（会被 normalize 取整到小时）。
var testBaseTime = time.Date(2026, 10, 20, 12, 34, 56, 0, time.UTC)

// testConfig 是空配置（appconfig 用默认值，不含任何真实凭据）。
var testConfig = config.Config{}
