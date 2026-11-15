package outbox

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/testdb"
)

// legacyClaimOutbox 是迁移 072 之前 inline 领取的反向过滤写法（排除事务外 topic）。
// 生产代码已改为白名单领取（queryClaimOutbox），这里保留旧写法仅用于 IT 量化基线：
// 外部 topic 极端积压时它必须按 event_id 顺序扫过全部积压行才能找到 inline 事件。
const legacyClaimOutbox = `SELECT event_id, topic, payload, attempts
FROM outbox_events
WHERE processed_at IS NULL AND available_at <= now() AND topic <> ALL($2)
ORDER BY event_id
LIMIT $1
FOR UPDATE SKIP LOCKED`

// explainPlanNode 是 EXPLAIN (FORMAT JSON) 计划树中测试用到的字段子集。
type explainPlanNode struct {
	NodeType            string            `json:"Node Type"`
	RelationName        string            `json:"Relation Name"`
	IndexName           string            `json:"Index Name"`
	IndexCond           string            `json:"Index Cond"`
	RecheckCond         string            `json:"Recheck Cond"`
	ActualRows          float64           `json:"Actual Rows"`
	ActualLoops         float64           `json:"Actual Loops"`
	RowsRemovedByFilter float64           `json:"Rows Removed by Filter"`
	Plans               []explainPlanNode `json:"Plans"`
}

// claimPlanStats 汇总一次领取查询计划中 outbox_events 相关扫描节点的实际代价：
// fetchedRows/removedRows 是实际输出/被过滤丢弃的行数（已乘 loops）。
type claimPlanStats struct {
	fetchedRows float64
	removedRows float64
	seqScanned  bool
	indexNames  []string
	indexCond   string
}

func (s claimPlanStats) hasIndex(name string) bool {
	for _, index := range s.indexNames {
		if index == name {
			return true
		}
	}
	return false
}

func collectClaimPlanStats(root explainPlanNode) claimPlanStats {
	var stats claimPlanStats
	var walk func(explainPlanNode)
	walk = func(node explainPlanNode) {
		loops := node.ActualLoops
		if loops <= 0 {
			loops = 1
		}
		if node.IndexName != "" {
			stats.indexNames = append(stats.indexNames, node.IndexName)
		}
		if strings.Contains(node.IndexCond, "topic = ANY") || strings.Contains(node.RecheckCond, "topic = ANY") {
			stats.indexCond = node.IndexCond + node.RecheckCond
		}
		if node.RelationName == "outbox_events" {
			if node.NodeType == "Seq Scan" {
				stats.seqScanned = true
			}
			stats.fetchedRows += node.ActualRows * loops
			stats.removedRows += node.RowsRemovedByFilter * loops
		}
		for _, child := range node.Plans {
			walk(child)
		}
	}
	walk(root)
	return stats
}

// quoteSQLString 把测试用 topic 字面量转成 SQL 单引号字符串（仅测试代码构造
// EXPLAIN EXECUTE 的常量参数使用；生产 SQL 一律 $N 参数化）。
func quoteSQLString(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

// explainClaimPlan 在与生产领取事务一致的前提下对领取 SQL 做 EXPLAIN ANALYZE：
// 事务内先 SET LOCAL force_custom_plan（生产 dispatcher 同样在领取事务里设置），
// 再用预编译语句 + 实际数组参数执行（与 pgx 预编译语句路径一致），返回根计划。
func explainClaimPlan(t *testing.T, factory *persistence.Factory, query string, limit int, topics []string) explainPlanNode {
	t.Helper()
	txFactory, cleanup := testdb.BeginScope(t)
	defer cleanup()
	qdb, ok := txFactory.DB.(persistence.QueryableDB)
	if !ok {
		t.Fatal("tx factory.DB is not QueryableDB")
	}
	ctx := context.Background()
	if err := qdb.Exec(ctx, queryForceCustomPlan); err != nil {
		t.Fatalf("set local plan_cache_mode: %v", err)
	}
	const stmt = "it_outbox_claim_plan"
	if err := qdb.Exec(ctx, "PREPARE "+stmt+"(int, text[]) AS "+query); err != nil {
		t.Fatalf("prepare claim SQL: %v", err)
	}
	defer func() { _ = qdb.Exec(context.Background(), "DEALLOCATE "+stmt) }()
	literals := make([]string, 0, len(topics))
	for _, topic := range topics {
		literals = append(literals, quoteSQLString(topic))
	}
	explainSQL := fmt.Sprintf("EXPLAIN (ANALYZE, BUFFERS, FORMAT JSON) EXECUTE %s(%d, ARRAY[%s]::text[])",
		stmt, limit, strings.Join(literals, ", "))
	var raw []byte
	if err := qdb.QueryRow(ctx, explainSQL).Scan(&raw); err != nil {
		t.Fatalf("explain claim SQL: %v", err)
	}
	// EXPLAIN (FORMAT JSON) 返回单元素数组：[{"Plan": {...}}]。
	var decoded []struct {
		Plan explainPlanNode `json:"Plan"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("decode explain JSON: %v (raw=%s)", err, raw)
	}
	if len(decoded) != 1 {
		t.Fatalf("explain 返回 %d 个计划, want 1", len(decoded))
	}
	return decoded[0].Plan
}

// TestPostgresIT_OutboxInlineClaimPlanIgnoresExternalBacklog 覆盖第 82 轮审计残留
// 的修复：inline 领取从「排除外部 topic」反向过滤改为「inline topic 白名单」+ 迁移
// 072 的 (topic, available_at, event_id) 部分索引后，外部 topic（媒体清理）极端
// 积压不再进入 inline 领取的扫描路径（旧写法 O(积压)，10 万行约 10.5ms）。
//
// 断言口径用计划形态与实际行数（不是耗时）：新领取查询必须走 idx_outbox_topic_due、
// 索引条件带 topic = ANY、实际只触碰少量 inline 行；旧反向过滤基线必须扫过全部
// 10 万积压行（量化修复前的代价）。全程提交型数据由 t.Cleanup 删除。
func TestPostgresIT_OutboxInlineClaimPlanIgnoresExternalBacklog(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	qdb, ok := factory.DB.(persistence.QueryableDB)
	if !ok {
		t.Fatal("factory.DB is not QueryableDB")
	}
	ctx := context.Background()
	suffix := time.Now().UnixNano()
	backlogTopic := fmt.Sprintf("it_outbox_plan_backlog_%d", suffix)
	inlineTopic := fmt.Sprintf("it_outbox_plan_inline_%d", suffix)
	t.Cleanup(func() {
		_ = qdb.Exec(ctx, "DELETE FROM outbox_events WHERE topic = ANY($1)", []string{backlogTopic, inlineTopic})
	})

	// 外部 topic 极端积压（10 万行）先入队、inline 事件（60 行）在后：旧查询按
	// event_id 升序取 50 条时必然扫过全部积压行。
	const insertSQL = `INSERT INTO outbox_events (topic, payload)
SELECT $1, '{}'::jsonb FROM generate_series(1, $2)`
	if err := qdb.Exec(ctx, insertSQL, backlogTopic, 100000); err != nil {
		t.Fatalf("insert external backlog: %v", err)
	}
	if err := qdb.Exec(ctx, insertSQL, inlineTopic, 60); err != nil {
		t.Fatalf("insert inline events: %v", err)
	}
	// 让 planner 的统计反映本次构造的积压规模（否则可能沿用空表统计导致计划失真）。
	if err := qdb.Exec(ctx, "ANALYZE outbox_events"); err != nil {
		t.Fatalf("analyze outbox_events: %v", err)
	}

	newPlan := explainClaimPlan(t, factory, queryClaimOutbox, defaultBatchSize, []string{inlineTopic})
	newStats := collectClaimPlanStats(newPlan)
	if newPlan.ActualRows != float64(defaultBatchSize) {
		t.Fatalf("新领取查询实际行数 = %.0f, want %d", newPlan.ActualRows, defaultBatchSize)
	}
	if !newStats.hasIndex("idx_outbox_topic_due") {
		t.Fatalf("新领取查询未使用迁移 072 的 idx_outbox_topic_due（indexes=%v）", newStats.indexNames)
	}
	if !strings.Contains(newStats.indexCond, "topic = ANY") {
		t.Fatalf("新领取查询的索引条件未包含 topic = ANY，indexCond=%q", newStats.indexCond)
	}
	if newStats.seqScanned {
		t.Fatal("新领取查询不得对 outbox_events 做顺序扫描")
	}
	// 10 万外部积压之下只应触碰 60 条 inline 行；任何 O(积压) 扫描都会远超该阈值。
	if newStats.fetchedRows > 1000 || newStats.removedRows > 1000 {
		t.Fatalf("新领取查询仍扫过外部积压：fetched=%.0f removed=%.0f（应只触碰少量 inline 行）",
			newStats.fetchedRows, newStats.removedRows)
	}

	// 旧反向过滤基线：排除列表取「除本测试 inline topic 之外的全部 topic」，即旧
	// 实现遇到「所有其它 topic 都是外部 topic」的最坏情况。这样断言不受共享测试库
	// 中其它用例残留 outbox 行（event_id 更小、可能先占满 LIMIT）的影响。
	otherTopics := []string{}
	otherRows, err := qdb.Query(ctx, "SELECT DISTINCT topic FROM outbox_events WHERE topic <> $1", inlineTopic)
	if err != nil {
		t.Fatalf("list other outbox topics: %v", err)
	}
	for otherRows.Next() {
		var topic string
		if err := otherRows.Scan(&topic); err != nil {
			otherRows.Close()
			t.Fatalf("scan other outbox topic: %v", err)
		}
		otherTopics = append(otherTopics, topic)
	}
	if err := otherRows.Err(); err != nil {
		otherRows.Close()
		t.Fatalf("iterate other outbox topics: %v", err)
	}
	otherRows.Close()
	if len(otherTopics) == 0 {
		t.Fatal("未找到除 inline topic 之外的 topic：积压构造失败")
	}
	legacyPlan := explainClaimPlan(t, factory, legacyClaimOutbox, defaultBatchSize, otherTopics)
	legacyStats := collectClaimPlanStats(legacyPlan)
	if legacyPlan.ActualRows != float64(defaultBatchSize) {
		t.Fatalf("旧领取查询实际行数 = %.0f, want %d", legacyPlan.ActualRows, defaultBatchSize)
	}
	if legacyStats.removedRows < 100000 {
		t.Fatalf("旧反向过滤基线应扫过全部 10 万积压行，实际丢弃行数 = %.0f", legacyStats.removedRows)
	}

	// 真实 dispatcher 路径回归：积压之下只领取并投递注册的 inline topic，
	// 外部积压行（未注册/外部 topic）保持未处理。
	dispatcher := NewDispatcher(factory, nil)
	delivered := 0
	dispatcher.Register(inlineTopic, func(context.Context, json.RawMessage) error {
		delivered++
		return nil
	})
	if processed, err := dispatcher.DrainOnce(ctx); err != nil || processed != defaultBatchSize {
		t.Fatalf("drain: processed=%d err=%v, want %d", processed, err, defaultBatchSize)
	}
	if delivered != defaultBatchSize {
		t.Fatalf("inline handler 投递次数 = %d, want %d", delivered, defaultBatchSize)
	}
	var inlinePending, backlogPending int
	if err := qdb.QueryRow(ctx,
		"SELECT COUNT(*) FROM outbox_events WHERE processed_at IS NULL AND topic = $1", inlineTopic).Scan(&inlinePending); err != nil {
		t.Fatalf("count inline pending: %v", err)
	}
	if inlinePending != 10 {
		t.Fatalf("inline 剩余未处理 = %d, want 10（60 条只领 50 条上限）", inlinePending)
	}
	if err := qdb.QueryRow(ctx,
		"SELECT COUNT(*) FROM outbox_events WHERE processed_at IS NULL AND topic = $1", backlogTopic).Scan(&backlogPending); err != nil {
		t.Fatalf("count backlog pending: %v", err)
	}
	if backlogPending != 100000 {
		t.Fatalf("外部积压行被误处理：pending = %d, want 100000（白名单只领取注册 topic）", backlogPending)
	}
}
