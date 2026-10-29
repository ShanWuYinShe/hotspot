package seed

import (
	"context"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"hotspot/backend-go/internal/platform/config"
)

// execute_test.go — 执行层单测（fake DB + fake 对象存储）：
// 幂等语句、--reset 先删后插、口令哈希可校验、环境守卫。

// insertAuthAccountsFromPlan 执行计划并返回 login_id → bcrypt 哈希。
func insertAuthAccountsFromPlan(t *testing.T, plan *Plan) map[string]string {
	t.Helper()
	fake := &fakeQueryable{}
	if err := insertAll(context.Background(), fake, plan.steps, bcryptMinCostForTest); err != nil {
		t.Fatalf("insertAll 失败: %v", err)
	}
	hashes := map[string]string{}
	for _, exec := range fake.execs {
		if !strings.Contains(exec.sql, "INSERT INTO auth_accounts") {
			continue
		}
		if len(exec.args) != 5 {
			t.Fatalf("auth_accounts 参数个数 = %d，期望 5", len(exec.args))
		}
		loginID, ok := exec.args[0].(string)
		if !ok {
			t.Fatalf("auth_accounts login_id 参数类型 = %T", exec.args[0])
		}
		hash, ok := exec.args[2].(string)
		if !ok {
			t.Fatalf("auth_accounts password_hash 参数类型 = %T", exec.args[2])
		}
		// 断言「哈希不是明文」时不能用短子串 "Qa" 判断：bcrypt 的 base64 字符集
		// 含大小写字母，每个哈希约 1% 概率偶然出现 "Qa"（10 个账号时每轮全量测试
		// 约 1/10 概率随机失败，2026-10-29 实测复现 1 次）。按完整明文口令判断才可靠。
		if hash == loginID || strings.Contains(hash, defaultQAPassword) {
			t.Errorf("口令哈希疑似明文: %q", hash)
		}
		hashes[loginID] = hash
	}
	return hashes
}

// TestPasswordsAreBcryptHashedAndVerifiable 账号口令必须 bcrypt 哈希且能通过登录校验。
func TestPasswordsAreBcryptHashedAndVerifiable(t *testing.T) {
	plan := mustBuildPlan(t, testPlanOptions())
	hashes := insertAuthAccountsFromPlan(t, plan)
	want := map[string]string{
		"qa@hotspot.local":     defaultQAPassword,
		"admin@hotspot.local":  defaultAdminPassword,
		"sophia@hotspot.local": legacyAccountPassword,
	}
	for loginID, plain := range want {
		hash, ok := hashes[loginID]
		if !ok {
			t.Errorf("缺少账号 %s 的哈希", loginID)
			continue
		}
		if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)); err != nil {
			t.Errorf("账号 %s 的哈希校验失败: %v", loginID, err)
		}
		if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte("WrongPassword1")); err == nil {
			t.Errorf("账号 %s 的错误口令竟然通过校验", loginID)
		}
	}
}

// TestPasswordEnvOverride Options 中的口令覆盖缺省值（SEED_QA_PASSWORD 链路）。
func TestPasswordEnvOverride(t *testing.T) {
	opts := testPlanOptions()
	opts.QAPassword = "CustomQA1234"
	opts.AdminPassword = "CustomAdmin99"
	plan := mustBuildPlan(t, opts)
	hashes := insertAuthAccountsFromPlan(t, plan)
	if err := bcrypt.CompareHashAndPassword([]byte(hashes["qa@hotspot.local"]), []byte("CustomQA1234")); err != nil {
		t.Errorf("QA 账号应使用覆盖口令: %v", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hashes["admin@hotspot.local"]), []byte("CustomAdmin99")); err != nil {
		t.Errorf("管理员账号应使用覆盖口令: %v", err)
	}
}

// TestGroupPasswordHashed 密码群的 password_hash 必须是 bcrypt("run123456")。
func TestGroupPasswordHashed(t *testing.T) {
	plan := mustBuildPlan(t, testPlanOptions())
	fake := &fakeQueryable{}
	if err := insertAll(context.Background(), fake, plan.steps, bcryptMinCostForTest); err != nil {
		t.Fatalf("insertAll 失败: %v", err)
	}
	columns := columnsOfTable(t, plan, "groups")
	hasPasswordIndex := columnIndex(t, columns, "has_password")
	passwordHashIndex := columnIndex(t, columns, "password_hash")
	found := false
	for _, exec := range fake.execs {
		if !strings.Contains(exec.sql, "INSERT INTO groups") {
			continue
		}
		groupID := exec.args[0].(string)
		if groupID != "group_seed_2" {
			continue
		}
		found = true
		if got, ok := exec.args[hasPasswordIndex].(bool); !ok || !got {
			t.Errorf("group_seed_2 has_password 应为 true")
		}
		hash, ok := exec.args[passwordHashIndex].(string)
		if !ok {
			t.Fatalf("group_seed_2 password_hash 类型 = %T，期望 string", exec.args[passwordHashIndex])
		}
		if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(groupSeedPassword)); err != nil {
			t.Errorf("群口令哈希校验失败: %v", err)
		}
	}
	if !found {
		t.Fatal("未找到 group_seed_2 的写入语句")
	}
}

// TestExecuteIfMissing 默认模式：不删除、上传全部对象、逐行 INSERT（ON CONFLICT DO NOTHING）。
func TestExecuteIfMissing(t *testing.T) {
	plan := mustBuildPlan(t, testPlanOptions())
	fake := &fakeQueryable{}
	storage := &fakeStorage{}
	if err := execute(context.Background(), fakeTxRunner(fake), storage, plan); err != nil {
		t.Fatalf("execute 失败: %v", err)
	}
	if len(storage.deletes) != 0 {
		t.Errorf("--if-missing 不应删除对象，实际删除 %d 个", len(storage.deletes))
	}
	if len(storage.puts) != plan.MediaCount() {
		t.Errorf("上传对象数 = %d，期望 %d", len(storage.puts), plan.MediaCount())
	}
	if fake.queryCount != 0 {
		t.Errorf("--if-missing 不应查询历史媒体键，Query 次数 = %d", fake.queryCount)
	}
	if len(fake.execs) != plan.RowCount() {
		t.Errorf("INSERT 次数 = %d，期望行数 %d", len(fake.execs), plan.RowCount())
	}
	for _, exec := range fake.execs {
		if strings.Contains(exec.sql, "DELETE FROM") {
			t.Errorf("--if-missing 不应执行 DELETE: %s", exec.sql)
		}
		if !strings.Contains(exec.sql, "ON CONFLICT DO NOTHING") {
			t.Errorf("INSERT 缺少幂等子句: %s", exec.sql)
		}
	}
}

// TestExecuteReset 重置模式：先按前缀删除（并回收历史对象），再上传、写入。
func TestExecuteReset(t *testing.T) {
	opts := testPlanOptions()
	opts.Mode = ModeReset
	plan := mustBuildPlan(t, opts)
	fake := &fakeQueryable{storageKeys: []string{"uploads/legacy_seed/object.jpg"}}
	storage := &fakeStorage{}
	if err := execute(context.Background(), fakeTxRunner(fake), storage, plan); err != nil {
		t.Fatalf("execute 失败: %v", err)
	}
	// 历史对象 + 计划对象都要删（去重后）。
	wantDeletes := plan.MediaCount() + 1
	if len(storage.deletes) != wantDeletes {
		t.Errorf("删除对象数 = %d，期望 %d", len(storage.deletes), wantDeletes)
	}
	if fake.queryCount != 1 {
		t.Errorf("应查询一次历史媒体键，实际 %d 次", fake.queryCount)
	}
	if got := fake.countStatements("DELETE FROM"); got != len(resetStatements) {
		t.Errorf("DELETE 语句数 = %d，期望 %d", got, len(resetStatements))
	}
	// 顺序：全部 DELETE 在第一条 INSERT 之前。
	firstInsert := fake.firstExecIndex("INSERT INTO")
	lastDelete := -1
	for index, exec := range fake.execs {
		if strings.Contains(exec.sql, "DELETE FROM") {
			lastDelete = index
		}
	}
	if firstInsert < 0 {
		t.Fatal("未找到任何 INSERT")
	}
	if lastDelete > firstInsert {
		t.Errorf("存在 INSERT 之后才执行的 DELETE（顺序错误）: insert=%d delete=%d", firstInsert, lastDelete)
	}
	// 写入行数不因重置而减少。
	if got := fake.countStatements("INSERT INTO"); got != plan.RowCount() {
		t.Errorf("INSERT 次数 = %d，期望 %d", got, plan.RowCount())
	}
}

// TestResetStatementsCoverSeedPrefixes 每条 seed 前缀都有对应清理语句，且不碰迁移表。
func TestResetStatementsCoverSeedPrefixes(t *testing.T) {
	statements := make([]string, 0, len(resetStatements))
	for _, statement := range resetStatements {
		if statement.description == "" || statement.sql == "" {
			t.Errorf("清理语句缺少描述/SQL")
		}
		statements = append(statements, statement.sql)
	}
	joined := strings.Join(statements, "\n")
	prefixes := []string{
		userIDPrefix, momentIDPrefix, commentIDPrefix, groupIDPrefix,
		groupMessageIDPrefix, chatMessageIDPrefix, notifyIDPrefix, shareIDPrefix,
		reportIDPrefix, uploadIDPrefix, inviteIDPrefix, applicationIDPrefix,
		groupAnnouncementIDPrefix, systemAnnouncementIDPrefix, adminRoleIDPrefix, adminAuditIDPrefix,
	}
	for _, prefix := range prefixes {
		if !strings.Contains(joined, prefix) {
			t.Errorf("--reset 清理语句未覆盖前缀 %q", prefix)
		}
	}
	if strings.Contains(joined, "schema_migrations") {
		t.Errorf("--reset 不应触碰 schema_migrations")
	}
}

// TestExecuteUploadFailurePropagates 对象上传失败必须中断执行（不能只写一半）。
func TestExecuteUploadFailurePropagates(t *testing.T) {
	plan := mustBuildPlan(t, testPlanOptions())
	fake := &fakeQueryable{}
	storage := &fakeStorage{putErr: context.DeadlineExceeded}
	err := execute(context.Background(), fakeTxRunner(fake), storage, plan)
	if err == nil {
		t.Fatal("上传失败时 execute 应返回错误")
	}
	if !strings.Contains(err.Error(), "上传媒体") {
		t.Errorf("错误信息应包含上传失败上下文: %v", err)
	}
	if len(fake.execs) != 0 {
		t.Errorf("上传失败后不应写数据库，实际执行 %d 条 SQL", len(fake.execs))
	}
}

// TestRunDryRunWithoutDependencies dry-run 不要求数据库/对象存储。
func TestRunDryRunWithoutDependencies(t *testing.T) {
	opts := testPlanOptions()
	opts.DryRun = true
	cfg := config.Config{AppEnv: "dev"}
	if err := Run(context.Background(), cfg, nil, opts); err != nil {
		t.Fatalf("dry-run 不应依赖 DB/对象存储: %v", err)
	}
}

// TestRunRejectsProductionEnvironment 生产环境必须拒绝执行（固定口令 dev 账号）。
func TestRunRejectsProductionEnvironment(t *testing.T) {
	opts := testPlanOptions()
	cfg := config.Config{AppEnv: "production"}
	err := Run(context.Background(), cfg, nil, opts)
	if err == nil {
		t.Fatal("生产环境应拒绝执行 seed")
	}
	if !strings.Contains(err.Error(), "dev/test") {
		t.Errorf("错误信息应说明仅限 dev/test 环境: %v", err)
	}
}

// TestRunRequiresDatabase 非 dry-run 且无数据库连接时给出明确错误。
func TestRunRequiresDatabase(t *testing.T) {
	opts := testPlanOptions()
	cfg := config.Config{AppEnv: "dev"}
	err := Run(context.Background(), cfg, nil, opts)
	if err == nil {
		t.Fatal("缺少数据库连接时应报错")
	}
	if !strings.Contains(err.Error(), "POSTGRES_DSN") {
		t.Errorf("错误信息应提示 POSTGRES_DSN: %v", err)
	}
}

// TestMergeStorageKeys 合并去重且稳定排序。
func TestMergeStorageKeys(t *testing.T) {
	merged := mergeStorageKeys([]string{"b", "a"}, []string{"a", "c", ""})
	if len(merged) != 3 || merged[0] != "a" || merged[1] != "b" || merged[2] != "c" {
		t.Fatalf("mergeStorageKeys = %v", merged)
	}
}
