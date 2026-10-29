package seed

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/config"
)

// plan_test.go — 计划层单测：幂等计划、表行数口径、SQL 形态、dry-run 输出。

// mustBuildPlan 构建测试计划（固定基准时间 + 最低 bcrypt 成本）。
// 未显式指定素材目录时用 TestMain 生成的临时素材（单测不依赖仓库 devdata）。
func mustBuildPlan(t *testing.T, opts Options) *Plan {
	t.Helper()
	if opts.AssetsDir == "" {
		opts.AssetsDir = testAssetsDirPath
	}
	plan, err := BuildPlan(testConfig, opts)
	if err != nil {
		t.Fatalf("BuildPlan 失败: %v", err)
	}
	return plan
}

// rowsOfTable 返回某表的全部行（comments 分两批写入时合并）。
func rowsOfTable(t *testing.T, plan *Plan, table string) [][]any {
	t.Helper()
	rows := make([][]any, 0)
	for _, step := range plan.steps {
		if step.Table == table {
			rows = append(rows, step.Rows...)
		}
	}
	return rows
}

// columnsOfTable 返回某表的列定义（取首次出现的批次）。
func columnsOfTable(t *testing.T, plan *Plan, table string) []column {
	t.Helper()
	for _, step := range plan.steps {
		if step.Table == table {
			return step.Columns
		}
	}
	t.Fatalf("计划中不存在表 %s", table)
	return nil
}

// columnIndex 返回列下标（不存在时 Fatal）。
func columnIndex(t *testing.T, columns []column, name string) int {
	t.Helper()
	for index, column := range columns {
		if column.name == name {
			return index
		}
	}
	t.Fatalf("列 %s 不存在", name)
	return -1
}

// expectedTableCounts 是规格 §3 的写入行数口径（同时是交付说明里的数字来源）。
var expectedTableCounts = map[string]int{
	"users": 10, "user_settings": 10, "user_profile_tags": 17,
	"user_relations": 13, "auth_accounts": 10, "media_upload_sessions": 38,
	"groups": 4, "group_members": 13, "group_announcements": 4,
	"group_bans": 1, "group_invites": 1, "group_applications": 2,
	"group_messages": 10, "group_message_read_state": 7,
	"private_chats": 3, "private_messages": 20,
	"private_chat_user_state": 6, "private_chat_read_state": 6,
	"moments": 24, "moment_images": 22, "moment_tags": 42, "moment_mentions": 2,
	"moment_likes": 29, "moment_favorites": 2, "comments": 14, "comment_likes": 9,
	"reactions": 32, "notifications": 9, "shares": 2, "reports": 4,
	"admin_roles": 2, "admin_role_permissions": 15, "admin_users": 1,
	"admin_user_roles": 1, "admin_audit_logs": 6, "admin_blocked_tags": 2,
	"system_announcements": 2, "app_configs": 1,
}

// TestBuildPlanTableCounts 断言写入规模与规格一致（dry-run 输出与交付说明
// 的「预期行数」都以本表为准）。
func TestBuildPlanTableCounts(t *testing.T) {
	plan := mustBuildPlan(t, testPlanOptions())
	counts := make(map[string]int)
	for _, count := range plan.TableCounts() {
		counts[count.Table] = count.Rows
	}
	for table, want := range expectedTableCounts {
		if got := counts[table]; got != want {
			t.Errorf("表 %s 行数 = %d，期望 %d", table, got, want)
		}
	}
	if len(counts) != len(expectedTableCounts) {
		t.Errorf("表数量 = %d，期望 %d（多出的表: %v）", len(counts), len(expectedTableCounts), diffKeys(counts, expectedTableCounts))
	}
	if plan.RowCount() != 396 {
		t.Errorf("数据总行数 = %d，期望 396", plan.RowCount())
	}
	if plan.MediaCount() != 38 {
		t.Errorf("媒体对象数 = %d，期望 38", plan.MediaCount())
	}
}

// diffKeys 返回 a 中不存在于 b 的键（错误提示用）。
func diffKeys(a, b map[string]int) []string {
	out := make([]string, 0)
	for key := range a {
		if _, ok := b[key]; !ok {
			out = append(out, key)
		}
	}
	return out
}

// TestBuildPlanDeterministic 同一基准时刻重复构建得到同一计划（--reset 可复现）。
func TestBuildPlanDeterministic(t *testing.T) {
	first := mustBuildPlan(t, testPlanOptions())
	second := mustBuildPlan(t, testPlanOptions())
	if first.RowCount() != second.RowCount() || first.MediaCount() != second.MediaCount() {
		t.Fatalf("两次构建行数/媒体数不一致: %d/%d vs %d/%d",
			first.RowCount(), first.MediaCount(), second.RowCount(), second.MediaCount())
	}
	if !first.BaseTime.Equal(second.BaseTime) {
		t.Fatalf("基准时间不一致: %s vs %s", first.BaseTime, second.BaseTime)
	}
	for table := range expectedTableCounts {
		firstRows := len(rowsOfTable(t, first, table))
		secondRows := len(rowsOfTable(t, second, table))
		if firstRows != secondRows {
			t.Fatalf("表 %s 两次构建行数不一致: %d vs %d", table, firstRows, secondRows)
		}
	}
}

// TestBuildPlanTimeTruncatedToHour 基准时间向下取整到小时（同一小时内重复执行稳定）。
func TestBuildPlanTimeTruncatedToHour(t *testing.T) {
	plan := mustBuildPlan(t, Options{Now: time.Date(2026, 10, 20, 12, 34, 56, 0, time.UTC), BcryptCost: bcryptMinCostForTest})
	want := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	if !plan.BaseTime.Equal(want) {
		t.Fatalf("基准时间 = %s，期望 %s", plan.BaseTime, want)
	}
}

// TestEveryRowArityMatchesColumns 每行参数个数必须与列数一致（否则运行时 SQL 报错）。
func TestEveryRowArityMatchesColumns(t *testing.T) {
	plan := mustBuildPlan(t, testPlanOptions())
	for _, step := range plan.steps {
		for rowIndex, row := range step.Rows {
			if len(row) != len(step.Columns) {
				t.Fatalf("表 %s 第 %d 行参数 %d 个，列 %d 个", step.Table, rowIndex+1, len(row), len(step.Columns))
			}
		}
	}
}

// TestInsertStatementsAreIdempotentAndParameterized 所有 INSERT 都参数化且
// 带 ON CONFLICT DO NOTHING；jsonb 列显式转型。
func TestInsertStatementsAreIdempotentAndParameterized(t *testing.T) {
	plan := mustBuildPlan(t, testPlanOptions())
	for _, step := range plan.steps {
		if len(step.Rows) == 0 {
			continue
		}
		statement := step.insertSQL()
		if !strings.HasPrefix(statement, "INSERT INTO "+step.Table+" (") {
			t.Errorf("表 %s 的 INSERT 前缀异常: %s", step.Table, statement)
		}
		if !strings.HasSuffix(statement, "ON CONFLICT DO NOTHING") {
			t.Errorf("表 %s 的 INSERT 缺少 ON CONFLICT DO NOTHING: %s", step.Table, statement)
		}
		if strings.Count(statement, "$") != len(step.Columns) {
			t.Errorf("表 %s 占位符数量 %d 与列数 %d 不一致: %s",
				step.Table, strings.Count(statement, "$"), len(step.Columns), statement)
		}
		if strings.Contains(statement, "schema_migrations") {
			t.Errorf("表 %s 不应写入 schema_migrations", step.Table)
		}
		for index, column := range step.Columns {
			if column.cast != "" {
				want := fmt.Sprintf("$%d%s", index+1, column.cast)
				if !strings.Contains(statement, want) {
					t.Errorf("表 %s 列 %s 缺少转型 %s: %s", step.Table, column.name, column.cast, statement)
				}
			}
		}
	}
}

// TestNoSchemaMigrationsWrite seed 不维护迁移版本表（由 platform/migrate 负责）。
func TestNoSchemaMigrationsWrite(t *testing.T) {
	plan := mustBuildPlan(t, testPlanOptions())
	for _, step := range plan.steps {
		if strings.Contains(step.Table, "schema_migrations") || strings.Contains(step.Table, "data_migrations") {
			t.Fatalf("seed 不应写入迁移表: %s", step.Table)
		}
	}
}

// TestPlanMediaSummary 媒体对象分类数量与规格 §4 一致，file_id/storage_key 唯一。
func TestPlanMediaSummary(t *testing.T) {
	plan := mustBuildPlan(t, testPlanOptions())
	counts := map[string]int{}
	uploadIDs := map[string]struct{}{}
	fileIDs := map[string]struct{}{}
	storageKeys := map[string]struct{}{}
	for _, obj := range plan.media {
		counts[obj.Category]++
		if _, ok := uploadIDs[obj.UploadID]; ok {
			t.Fatalf("upload_id 重复: %s", obj.UploadID)
		}
		uploadIDs[obj.UploadID] = struct{}{}
		if _, ok := fileIDs[obj.FileID]; ok {
			t.Fatalf("file_id 重复: %s", obj.FileID)
		}
		fileIDs[obj.FileID] = struct{}{}
		if _, ok := storageKeys[obj.StorageKey]; ok {
			t.Fatalf("storage_key 重复: %s", obj.StorageKey)
		}
		storageKeys[obj.StorageKey] = struct{}{}
		if !strings.HasPrefix(obj.UploadID, uploadIDPrefix) {
			t.Errorf("upload_id 缺少固定前缀: %s", obj.UploadID)
		}
		if obj.FileID != buildFileID(obj.UserID, obj.UploadID) {
			t.Errorf("媒体 %s 的 file_id 规则不符", obj.UploadID)
		}
		if obj.StorageKey != storageKeyFor(obj.UserID, obj.UploadID, obj.FileName) {
			t.Errorf("媒体 %s 的 storage_key 规则不符", obj.UploadID)
		}
		if obj.MimeType != seedJPEGMimeType || obj.FileSize <= 0 || obj.Width <= 0 || obj.Height <= 0 {
			t.Errorf("媒体 %s 元数据不完整: %+v", obj.UploadID, obj)
		}
	}
	want := map[string]int{"avatar": 9, "group_avatar": 4, "moment_image": 22, "chat_image": 3}
	for category, wantCount := range want {
		if counts[category] != wantCount {
			t.Errorf("媒体分类 %s 数量 = %d，期望 %d", category, counts[category], wantCount)
		}
	}
}

// TestPrintPlan dry-run 输出包含账号、明细与提示，且不含口令明文。
func TestPrintPlan(t *testing.T) {
	plan := mustBuildPlan(t, Options{Mode: ModeIfMissing, DryRun: true, Now: testBaseTime, BcryptCost: bcryptMinCostForTest})
	var buffer bytes.Buffer
	if err := PrintPlan(&buffer, plan); err != nil {
		t.Fatalf("PrintPlan 失败: %v", err)
	}
	output := buffer.String()
	for _, want := range []string{
		"HotSpot 开发种子数据计划", "qa@hotspot.local", "admin@hotspot.local",
		"users", "moments", "notifications", "38 个", "--dry-run",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("dry-run 输出缺少 %q", want)
		}
	}
	for _, secret := range []string{defaultQAPassword, defaultAdminPassword, legacyAccountPassword, groupSeedPassword} {
		if strings.Contains(output, secret) {
			t.Errorf("dry-run 输出不应包含口令明文 %q", secret)
		}
	}
}

// TestOptionsNormalize 校验模式/缺省值规范化。
func TestOptionsNormalize(t *testing.T) {
	normalized, err := Options{}.normalize()
	if err != nil {
		t.Fatalf("normalize 失败: %v", err)
	}
	if normalized.Mode != ModeIfMissing {
		t.Errorf("默认模式 = %q，期望 %q", normalized.Mode, ModeIfMissing)
	}
	if normalized.QAPassword != defaultQAPassword || normalized.AdminPassword != defaultAdminPassword {
		t.Errorf("默认口令未回退规格值")
	}
	if normalized.BcryptCost != config.DefaultBcryptCost() {
		t.Errorf("默认 bcrypt 成本 = %d，期望 %d", normalized.BcryptCost, config.DefaultBcryptCost())
	}
	if _, err := (Options{Mode: "bogus"}).normalize(); err == nil {
		t.Error("非法模式应返回错误")
	}
	if _, err := (Options{Mode: ModeReset}).normalize(); err != nil {
		t.Errorf("reset 模式应合法: %v", err)
	}
}

// TestAppConfigPayloadIsValidAppConfig 应用配置 JSON 必须是完整 AppConfig
// （管理端配置页按该结构渲染，缺字段会显示为零值/空开关）。
func TestAppConfigPayloadIsValidAppConfig(t *testing.T) {
	plan := mustBuildPlan(t, testPlanOptions())
	rows := rowsOfTable(t, plan, "app_configs")
	if len(rows) != 1 {
		t.Fatalf("app_configs 行数 = %d，期望 1", len(rows))
	}
	payload, ok := rows[0][1].(string)
	if !ok {
		t.Fatalf("app_configs.payload_json 参数类型 = %T，期望 string", rows[0][1])
	}
	var decoded map[string]any
	if err := json.Unmarshal([]byte(payload), &decoded); err != nil {
		t.Fatalf("app_configs payload 不是合法 JSON: %v", err)
	}
	for _, key := range []string{"features", "limits", "contact", "upload", "defaults", "rate_limits", "response_defaults", "auth"} {
		if _, ok := decoded[key]; !ok {
			t.Errorf("appconfig 缺少字段 %s", key)
		}
	}
	features, _ := decoded["features"].(map[string]any)
	if features["enable_moments"] != true || features["enable_public_registration"] != true {
		t.Errorf("features 关键开关不正确: %v", features)
	}
	limits, _ := decoded["limits"].(map[string]any)
	emojis, _ := limits["reaction_emojis"].([]any)
	if len(emojis) == 0 {
		t.Errorf("limits.reaction_emojis 为空")
	}
}
