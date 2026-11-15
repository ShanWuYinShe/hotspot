package media

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
)

// 保留期孤儿清理（第 82 轮审计 P2-2；批次 10 泛化）：uploads/ 前缀（complete 后
// 重放预签名 PUT 写回的上传键）与 published/ 前缀（Prepare→Commit 中途失败留下的
// 发布副本）共用同一套判定——「对象早于保留期（2 天）且 DB 无任何行引用
// storage_key」才删除；单轮按前缀分别有界（页数/删除/时长），失败下一轮重试。

func newOrphanSweepService(t *testing.T) (*Service, *InMemoryRepository, *coverageProvider) {
	t.Helper()
	repo := NewInMemoryRepository()
	provider := &coverageProvider{
		listPagesByPrefix:  map[string][]ObjectListPage{},
		listTokensByPrefix: map[string][]string{},
	}
	return NewServiceWithRepoAndProvider(repo, provider), repo, provider
}

func orphanObject(key string, age time.Duration) ObjectSummary {
	return ObjectSummary{Key: key, LastModified: time.Now().UTC().Add(-age)}
}

// orphanSweepOldAge 返回必定早于保留期的对象年龄（多 1 小时，避免临界抖动）。
func orphanSweepOldAge() time.Duration {
	return staleUploadRetentionDays*24*time.Hour + time.Hour
}

// seedStorageKeyRow 写入一条引用 storageKey 的会话行（completed 控制状态）：
// 孤儿判定必须保留任何被行引用的对象。
func seedStorageKeyRow(t *testing.T, repo *InMemoryRepository, uploadID, storageKey string, completed bool) {
	t.Helper()
	upload := pendingUpload{
		UploadID: uploadID, UserID: "u1", StorageKey: storageKey, FileName: "a.bin",
		DeclaredFileSize: 10, DeclaredMimeType: "application/octet-stream", Category: CategoryChatFile,
		Completed: completed,
		CreatedAt: time.Now().UTC().Add(-72 * time.Hour).Format(time.RFC3339),
	}
	if completed {
		upload.File = UploadedMedia{FileID: "file_" + uploadID, FileSize: 10, MimeType: "application/octet-stream"}
	}
	if err := repo.Save(context.Background(), upload); err != nil {
		t.Fatalf("seed storage key row %s: %v", uploadID, err)
	}
}

// 引用判定与保留期（uploads/ 前缀，保持批次 9 语义）：只删「超过 2 天且无任何会话行
// 引用」的对象；未过期对象、pending 行/历史完成态行（storage_key 仍为 uploads/ 键）
// 引用的对象、LastModified 缺失（零值，年龄未知）的对象都必须保留。
func TestSweepOrphanObjectsDeletesOnlyUnreferencedExpiredUploadObjects(t *testing.T) {
	svc, repo, provider := newOrphanSweepService(t)
	ctx := context.Background()
	old := orphanSweepOldAge()
	pendingKey := "uploads/u1/upload_pending.bin"
	legacyKey := "uploads/u1/upload_legacy_completed.bin"
	seedStorageKeyRow(t, repo, "upload_pending", pendingKey, false)
	seedStorageKeyRow(t, repo, "upload_legacy_completed", legacyKey, true)
	orphanKey := "uploads/u1/upload_orphan.bin"
	freshKey := "uploads/u1/upload_fresh.bin"
	provider.listPagesByPrefix[orphanUploadObjectPrefix] = []ObjectListPage{{
		Objects: []ObjectSummary{
			orphanObject(orphanKey, old),
			orphanObject(pendingKey, old),
			orphanObject(legacyKey, old),
			orphanObject(freshKey, time.Minute),
			{Key: "uploads/u1/upload_no_time.bin"},
		},
	}}

	svc.sweepOrphanObjects(ctx)

	if got := provider.deletedKeys; len(got) != 1 || got[0] != orphanKey {
		t.Fatalf("deleted = %v, want only the unreferenced expired key %q", got, orphanKey)
	}
	// 两个前缀都必须被枚举，且 uploads/ 先跑（published/ 的扫描量不得挤占 uploads/）。
	wantPrefixes := []string{orphanUploadObjectPrefix, orphanPublishedObjectPrefix}
	if fmt.Sprint(provider.listPrefixes) != fmt.Sprint(wantPrefixes) {
		t.Fatalf("list prefixes = %v, want %v", provider.listPrefixes, wantPrefixes)
	}
	for i, maxKeys := range provider.listMaxKeys {
		if maxKeys != orphanSweepPageSize {
			t.Fatalf("list call %d page size = %d, want %d", i, maxKeys, orphanSweepPageSize)
		}
	}
	// 引用行一律保留（行删除是保留期会话阶段的职责，孤儿阶段只删对象）。
	for _, uploadID := range []string{"upload_pending", "upload_legacy_completed"} {
		if _, found, err := repo.Get(ctx, uploadID); err != nil || !found {
			t.Fatalf("referenced row %s must be kept: found=%v err=%v", uploadID, found, err)
		}
	}
}

// published/ 前缀（批次 10）：发布副本无行引用且超过保留期才删除；有行引用（任意状态）、
// 未过期、年龄未知的对象一律保留。
func TestSweepOrphanObjectsDeletesOnlyUnreferencedExpiredPublishedObjects(t *testing.T) {
	svc, repo, provider := newOrphanSweepService(t)
	ctx := context.Background()
	old := orphanSweepOldAge()
	completedKey := "published/u1/upload_completed.png"
	pendingRefKey := "published/u1/upload_pending_ref.png"
	seedStorageKeyRow(t, repo, "upload_completed", completedKey, true)
	// 任意状态的行都算引用（与 uploads/ 阶段同一口径）：pending 行即使 storage_key
	// 人为写成发布键，也必须保留该对象，不得只看 completed 行。
	seedStorageKeyRow(t, repo, "upload_pending_ref", pendingRefKey, false)
	orphanKey := "published/u1/upload_orphan.png"
	freshKey := "published/u1/upload_fresh.png"
	provider.listPagesByPrefix[orphanPublishedObjectPrefix] = []ObjectListPage{{
		Objects: []ObjectSummary{
			orphanObject(orphanKey, old),
			orphanObject(completedKey, old),
			orphanObject(pendingRefKey, old),
			orphanObject(freshKey, time.Minute),
			{Key: "published/u1/upload_no_time.png"},
		},
	}}

	svc.sweepOrphanObjects(ctx)

	if got := provider.deletedKeys; len(got) != 1 || got[0] != orphanKey {
		t.Fatalf("deleted = %v, want only the unreferenced expired published key %q", got, orphanKey)
	}
	if _, found, err := repo.Get(ctx, "upload_completed"); err != nil || !found {
		t.Fatalf("completed row must be kept: found=%v err=%v", found, err)
	}
	if _, found, err := repo.Get(ctx, "upload_pending_ref"); err != nil || !found {
		t.Fatalf("pending row must be kept: found=%v err=%v", found, err)
	}
}

// published/ 引用判定额外包含「同 upload 上传键」：未完成会话行引用上传键时，发布副本
// 可能被在途 complete 重新发布并提交（行提交后就会引用该发布键），必须保留；上传键也
// 没有行引用时发布孤儿正常删除。
func TestSweepOrphanObjectsPublishedChecksPendingSiblingUploadKey(t *testing.T) {
	base := NewInMemoryRepository()
	repo := &recordingExistingKeysRepo{InMemoryRepository: base}
	provider := &coverageProvider{
		listPagesByPrefix:  map[string][]ObjectListPage{},
		listTokensByPrefix: map[string][]string{},
	}
	svc := NewServiceWithRepoAndProvider(repo, provider)
	old := orphanSweepOldAge()
	// 在途未完成会话：行引用 uploads 兄弟键（complete 尚未提交）；发布副本是早先一次
	// 失败的 complete 尝试留下的，按「无行引用」会误判为孤儿。
	inFlightPublishedKey := "published/u1/upload_in_flight.png"
	inFlightUploadKey := "uploads/u1/upload_in_flight.png"
	seedStorageKeyRow(t, base, "upload_in_flight", inFlightUploadKey, false)
	orphanKey := "published/u1/upload_crashed.png"
	provider.listPagesByPrefix[orphanPublishedObjectPrefix] = []ObjectListPage{{
		Objects: []ObjectSummary{orphanObject(inFlightPublishedKey, old), orphanObject(orphanKey, old)},
	}}

	svc.sweepOrphanObjects(context.Background())

	if got := provider.deletedKeys; len(got) != 1 || got[0] != orphanKey {
		t.Fatalf("deleted = %v, want only %q (in-flight sibling must be kept)", got, orphanKey)
	}
	if len(repo.calls) != 1 {
		t.Fatalf("reference queries = %d, want 1 (only published/ has candidates)", len(repo.calls))
	}
	wantQuery := []string{
		inFlightPublishedKey, inFlightUploadKey,
		orphanKey, "uploads/u1/upload_crashed.png",
	}
	if fmt.Sprint(repo.calls[0]) != fmt.Sprint(wantQuery) {
		t.Fatalf("reference query keys = %v, want %v (发布键 + 同 upload 上传键)", repo.calls[0], wantQuery)
	}
}

// uploadsSiblingKeyForPublished：发布键 → 同 upload 上传键；非 published/ 前缀返回空串。
func TestUploadsSiblingKeyForPublished(t *testing.T) {
	cases := map[string]string{
		"published/u1/upload_1.png": "uploads/u1/upload_1.png",
		"uploads/u1/upload_1.png":   "",
		"published/":                "uploads/",
		"":                          "",
	}
	for input, want := range cases {
		if got := uploadsSiblingKeyForPublished(input); got != want {
			t.Fatalf("uploadsSiblingKeyForPublished(%q) = %q, want %q", input, got, want)
		}
	}
}

// 两前缀同轮处理：uploads/ 与 published/ 的孤儿在同一轮都被删除，且各自有引用的对象
// 都被保留（展示顺序固定 uploads/ → published/）。
func TestSweepOrphanObjectsCoversBothPrefixesInOneRound(t *testing.T) {
	svc, repo, provider := newOrphanSweepService(t)
	ctx := context.Background()
	old := orphanSweepOldAge()
	uploadRefKey := "uploads/u1/upload_in_flight.bin"
	seedStorageKeyRow(t, repo, "upload_in_flight", uploadRefKey, false)
	publishedRefKey := "published/u1/upload_done.png"
	seedStorageKeyRow(t, repo, "upload_done", publishedRefKey, true)
	uploadOrphan := "uploads/u1/upload_replayed.bin"
	publishedOrphan := "published/u1/upload_crashed_before_commit.png"
	provider.listPagesByPrefix[orphanUploadObjectPrefix] = []ObjectListPage{{
		Objects: []ObjectSummary{orphanObject(uploadOrphan, old), orphanObject(uploadRefKey, old)},
	}}
	provider.listPagesByPrefix[orphanPublishedObjectPrefix] = []ObjectListPage{{
		Objects: []ObjectSummary{orphanObject(publishedOrphan, old), orphanObject(publishedRefKey, old)},
	}}

	svc.sweepOrphanObjects(ctx)

	deleted := map[string]int{}
	for _, key := range provider.deletedKeys {
		deleted[key]++
	}
	if deleted[uploadOrphan] != 1 || deleted[publishedOrphan] != 1 || len(provider.deletedKeys) != 2 {
		t.Fatalf("deleted = %v, want exactly [%q %q]", provider.deletedKeys, uploadOrphan, publishedOrphan)
	}
	if got, want := fmt.Sprint(provider.listPrefixes), fmt.Sprint([]string{orphanUploadObjectPrefix, orphanPublishedObjectPrefix}); got != want {
		t.Fatalf("list prefixes = %v, want %v", got, want)
	}
}

// 分页：IsTruncated 驱动 continuation-token 翻页，跨页候选都参与判定。
func TestSweepOrphanObjectsPaginatesUploadsPrefix(t *testing.T) {
	svc, _, provider := newOrphanSweepService(t)
	old := orphanSweepOldAge()
	provider.listPagesByPrefix[orphanUploadObjectPrefix] = []ObjectListPage{
		{Objects: []ObjectSummary{orphanObject("uploads/u1/a.bin", old)}, IsTruncated: true, NextContinuationToken: "token-1"},
		{Objects: []ObjectSummary{orphanObject("uploads/u1/b.bin", old)}, IsTruncated: true, NextContinuationToken: "token-2"},
		{Objects: []ObjectSummary{orphanObject("uploads/u1/c.bin", old)}},
	}

	svc.sweepOrphanObjects(context.Background())

	wantTokens := []string{"", "token-1", "token-2"}
	if got := provider.listTokensByPrefix[orphanUploadObjectPrefix]; fmt.Sprint(got) != fmt.Sprint(wantTokens) {
		t.Fatalf("uploads list tokens = %v, want %v", got, wantTokens)
	}
	if len(provider.deletedKeys) != 3 {
		t.Fatalf("deleted = %v, want 3 objects", provider.deletedKeys)
	}
}

// 单轮删除上限：一页内候选过多时只删 orphanSweepDeleteLimit 条并停止（剩余对象仍满足
// 条件，下一轮继续）。
func TestSweepOrphanObjectsStopsAtDeleteLimit(t *testing.T) {
	svc, _, provider := newOrphanSweepService(t)
	old := orphanSweepOldAge()
	objects := make([]ObjectSummary, 0, orphanSweepDeleteLimit+10)
	for i := 0; i < orphanSweepDeleteLimit+10; i++ {
		objects = append(objects, orphanObject(fmt.Sprintf("uploads/u1/orphan_%03d.bin", i), old))
	}
	provider.listPagesByPrefix[orphanUploadObjectPrefix] = []ObjectListPage{
		{Objects: objects, IsTruncated: true, NextContinuationToken: "more"},
	}

	svc.sweepOrphanObjects(context.Background())

	if len(provider.deletedKeys) != orphanSweepDeleteLimit {
		t.Fatalf("deleted = %d, want delete limit %d", len(provider.deletedKeys), orphanSweepDeleteLimit)
	}
	if got := len(provider.listTokensByPrefix[orphanUploadObjectPrefix]); got != 1 {
		t.Fatalf("delete limit must stop uploads pagination, list calls = %d", got)
	}
}

// 单轮扫描页数上限：异常多页时不会无限翻页，最多 orphanSweepListMaxPages 页。
func TestSweepOrphanObjectsBoundedUploadPages(t *testing.T) {
	svc, _, provider := newOrphanSweepService(t)
	old := orphanSweepOldAge()
	for i := 0; i < orphanSweepListMaxPages+5; i++ {
		provider.listPagesByPrefix[orphanUploadObjectPrefix] = append(
			provider.listPagesByPrefix[orphanUploadObjectPrefix],
			ObjectListPage{
				Objects:               []ObjectSummary{orphanObject(fmt.Sprintf("uploads/u1/page_%02d.bin", i), old)},
				IsTruncated:           true,
				NextContinuationToken: fmt.Sprintf("token-%02d", i),
			})
	}

	svc.sweepOrphanObjects(context.Background())

	if got := len(provider.listTokensByPrefix[orphanUploadObjectPrefix]); got != orphanSweepListMaxPages {
		t.Fatalf("uploads list calls = %d, want page cap %d", got, orphanSweepListMaxPages)
	}
	if len(provider.deletedKeys) != orphanSweepListMaxPages {
		t.Fatalf("deleted = %d, want %d", len(provider.deletedKeys), orphanSweepListMaxPages)
	}
}

// published/ 前缀跨轮推进游标：单轮翻页上限耗尽后保存 continuation token，下一轮从
// 上次位置继续（published/ 含全部完成态媒体、无界增长，固定从首页开始会永远扫不到
// 窗口外），扫到末页后复位，下一轮再从首页开始。
func TestSweepOrphanObjectsPublishedPrefixRotatesCursor(t *testing.T) {
	svc, _, provider := newOrphanSweepService(t)
	ctx := context.Background()
	old := orphanSweepOldAge()
	// 第 1 轮：orphanSweepListMaxPages 个截断页，每页一个无引用旧对象。
	for i := 0; i < orphanSweepListMaxPages; i++ {
		provider.listPagesByPrefix[orphanPublishedObjectPrefix] = append(
			provider.listPagesByPrefix[orphanPublishedObjectPrefix],
			ObjectListPage{
				Objects:               []ObjectSummary{orphanObject(fmt.Sprintf("published/u1/page_%02d.png", i), old)},
				IsTruncated:           true,
				NextContinuationToken: fmt.Sprintf("pub-token-%02d", i),
			})
	}
	// 第 2 轮：从保存的 token 开始，最后一页（不再截断）里再放一个孤儿对象。
	provider.listPagesByPrefix[orphanPublishedObjectPrefix] = append(
		provider.listPagesByPrefix[orphanPublishedObjectPrefix],
		ObjectListPage{Objects: []ObjectSummary{orphanObject("published/u1/last.png", old)}},
	)

	svc.sweepOrphanObjects(ctx) // 第 1 轮

	if svc.orphanPublishedCursor != fmt.Sprintf("pub-token-%02d", orphanSweepListMaxPages-1) {
		t.Fatalf("cursor after capped round = %q, want last next-token", svc.orphanPublishedCursor)
	}
	if len(provider.deletedKeys) != orphanSweepListMaxPages {
		t.Fatalf("round 1 deleted = %d, want %d", len(provider.deletedKeys), orphanSweepListMaxPages)
	}
	tokensAfterRound1 := len(provider.listTokensByPrefix[orphanPublishedObjectPrefix])

	svc.sweepOrphanObjects(ctx) // 第 2 轮：从游标继续

	round2Tokens := provider.listTokensByPrefix[orphanPublishedObjectPrefix][tokensAfterRound1:]
	if len(round2Tokens) != 1 || round2Tokens[0] != fmt.Sprintf("pub-token-%02d", orphanSweepListMaxPages-1) {
		t.Fatalf("round 2 published tokens = %v, want [last next-token]", round2Tokens)
	}
	if svc.orphanPublishedCursor != "" {
		t.Fatalf("cursor must reset after reaching the end, got %q", svc.orphanPublishedCursor)
	}
	if len(provider.deletedKeys) != orphanSweepListMaxPages+1 {
		t.Fatalf("deleted after round 2 = %d, want %d", len(provider.deletedKeys), orphanSweepListMaxPages+1)
	}

	tokensAfterRound2 := len(provider.listTokensByPrefix[orphanPublishedObjectPrefix])
	svc.sweepOrphanObjects(ctx) // 第 3 轮：游标为空 → 从首页重新开始
	round3Tokens := provider.listTokensByPrefix[orphanPublishedObjectPrefix][tokensAfterRound2:]
	if len(round3Tokens) != 1 || round3Tokens[0] != "" {
		t.Fatalf("round 3 published tokens = %v, want first page after reset", round3Tokens)
	}
}

// 列举失败时保存的 continuation token 可能已失效：必须复位到首页，避免坏 token 让
// published/ 前缀清理永久停摆。
func TestSweepOrphanObjectsPublishedCursorResetsOnListFailure(t *testing.T) {
	svc, _, provider := newOrphanSweepService(t)
	ctx := context.Background()
	for i := 0; i < orphanSweepListMaxPages; i++ {
		provider.listPagesByPrefix[orphanPublishedObjectPrefix] = append(
			provider.listPagesByPrefix[orphanPublishedObjectPrefix],
			ObjectListPage{IsTruncated: true, NextContinuationToken: fmt.Sprintf("pub-token-%02d", i)})
	}
	svc.sweepOrphanObjects(ctx) // 第 1 轮：游标推进到窗口末尾
	if svc.orphanPublishedCursor == "" {
		t.Fatal("cursor must advance after a capped round")
	}

	provider.listErrByPrefix = map[string]error{orphanPublishedObjectPrefix: errors.New("list boom")}
	svc.sweepOrphanObjects(ctx) // 第 2 轮：published/ 列举失败
	if svc.orphanPublishedCursor != "" {
		t.Fatalf("cursor must reset on list failure, got %q", svc.orphanPublishedCursor)
	}
	if len(provider.deletedKeys) != 0 {
		t.Fatalf("no object may be deleted when listing fails, got %v", provider.deletedKeys)
	}

	provider.listErrByPrefix = nil
	tokensBefore := len(provider.listTokensByPrefix[orphanPublishedObjectPrefix])
	svc.sweepOrphanObjects(ctx) // 第 3 轮：从首页重新开始
	if got := provider.listTokensByPrefix[orphanPublishedObjectPrefix][tokensBefore:]; len(got) != 1 || got[0] != "" {
		t.Fatalf("tokens after reset = %v, want [empty] (首页)", got)
	}
}

// published/ 引用查询失败时游标必须钉在本页起点：下一轮重扫该页（不能推进 token 跳过
// 本页尾部候选），恢复后同一页的孤儿对象被删除。
func TestSweepOrphanObjectsPublishedRetriesPageAfterReferenceFailure(t *testing.T) {
	base := NewInMemoryRepository()
	repo := &orphanReferenceFailOnceRepo{InMemoryRepository: base}
	provider := &coverageProvider{
		listPagesByPrefix:  map[string][]ObjectListPage{},
		listTokensByPrefix: map[string][]string{},
	}
	svc := NewServiceWithRepoAndProvider(repo, provider)
	ctx := context.Background()
	old := orphanSweepOldAge()
	orphanKey := "published/u1/upload_orphan.png"
	// 第 1 页无候选、截断到 pub-token-1；第 2 页（由 pub-token-1 取回）含孤儿候选，
	// 其引用查询第一次失败。
	provider.listPagesByPrefix[orphanPublishedObjectPrefix] = []ObjectListPage{
		{IsTruncated: true, NextContinuationToken: "pub-token-1"},
		{Objects: []ObjectSummary{orphanObject(orphanKey, old)}},
	}

	svc.sweepOrphanObjects(ctx) // 第 1 轮：第 2 页引用查询失败
	if len(provider.deletedKeys) != 0 {
		t.Fatalf("no object may be deleted when reference check fails, got %v", provider.deletedKeys)
	}
	if svc.orphanPublishedCursor != "pub-token-1" {
		t.Fatalf("cursor = %q, want stuck at the failing page token", svc.orphanPublishedCursor)
	}
	// 下一轮的同一页再次返回该孤儿对象（对象未被删，仍在存储侧）。
	provider.listPagesByPrefix[orphanPublishedObjectPrefix] = append(
		provider.listPagesByPrefix[orphanPublishedObjectPrefix],
		ObjectListPage{Objects: []ObjectSummary{orphanObject(orphanKey, old)}})

	tokensBefore := len(provider.listTokensByPrefix[orphanPublishedObjectPrefix])
	svc.sweepOrphanObjects(ctx) // 第 2 轮：恢复后重扫同一页并删除孤儿
	if got := provider.listTokensByPrefix[orphanPublishedObjectPrefix][tokensBefore:]; len(got) == 0 || got[0] != "pub-token-1" {
		t.Fatalf("round 2 published tokens = %v, want retry from pub-token-1", got)
	}
	if len(provider.deletedKeys) != 1 || provider.deletedKeys[0] != orphanKey {
		t.Fatalf("deleted = %v, want %q", provider.deletedKeys, orphanKey)
	}
}

// published/ 删除上限触顶（异常超大页）时游标必须留在本页起点：不能用下一页 token
// 跳过本页尾部候选。
func TestSweepOrphanObjectsPublishedDeleteLimitKeepsPageCursor(t *testing.T) {
	svc, _, provider := newOrphanSweepService(t)
	old := orphanSweepOldAge()
	objects := make([]ObjectSummary, 0, orphanSweepDeleteLimit+10)
	for i := 0; i < orphanSweepDeleteLimit+10; i++ {
		objects = append(objects, orphanObject(fmt.Sprintf("published/u1/orphan_%03d.png", i), old))
	}
	provider.listPagesByPrefix[orphanPublishedObjectPrefix] = []ObjectListPage{
		{Objects: objects, IsTruncated: true, NextContinuationToken: "pub-next"},
	}
	svc.orphanPublishedCursor = "" // 本页起点即首页

	svc.sweepOrphanObjects(context.Background())

	if len(provider.deletedKeys) != orphanSweepDeleteLimit {
		t.Fatalf("deleted = %d, want delete limit %d", len(provider.deletedKeys), orphanSweepDeleteLimit)
	}
	if svc.orphanPublishedCursor != "" {
		t.Fatalf("cursor = %q, want page start (empty), not next token", svc.orphanPublishedCursor)
	}
}

// 失败路径：列举失败 / 引用查询失败时不得删除任何对象（否则可能误删在途上传/发布
// 副本），只告警留待下一轮。
func TestSweepOrphanObjectsKeepsObjectsOnFailures(t *testing.T) {
	old := orphanSweepOldAge()

	t.Run("list failure", func(t *testing.T) {
		svc, _, provider := newOrphanSweepService(t)
		provider.listErr = errors.New("list boom")
		svc.sweepOrphanObjects(context.Background())
		if len(provider.deletedKeys) != 0 {
			t.Fatalf("no object may be deleted when listing fails, got %v", provider.deletedKeys)
		}
	})

	t.Run("budget exhausted mid-sweep", func(t *testing.T) {
		svc, _, provider := newOrphanSweepService(t)
		provider.listPagesByPrefix[orphanUploadObjectPrefix] = []ObjectListPage{{
			Objects: []ObjectSummary{orphanObject("uploads/u1/orphan.bin", old)},
		}}
		// 已取消的 ctx：列举/引用查询（fake 不检查 ctx）之后、删除之前检查预算，
		// 不得继续向已取消的 ctx 发删除请求。
		canceled, cancel := context.WithCancel(context.Background())
		cancel()
		svc.sweepOrphanObjects(canceled)
		if len(provider.deletedKeys) != 0 {
			t.Fatalf("canceled budget must not delete objects, got %v", provider.deletedKeys)
		}
	})

	t.Run("reference query failure", func(t *testing.T) {
		base := NewInMemoryRepository()
		provider := &coverageProvider{
			listPagesByPrefix: map[string][]ObjectListPage{
				orphanUploadObjectPrefix: {{Objects: []ObjectSummary{orphanObject("uploads/u1/orphan.bin", old)}}},
			},
			listTokensByPrefix: map[string][]string{},
		}
		repo := &orphanReferenceErrorRepo{InMemoryRepository: base, err: errors.New("reference boom")}
		NewServiceWithRepoAndProvider(repo, provider).sweepOrphanObjects(context.Background())
		if len(provider.deletedKeys) != 0 {
			t.Fatalf("no object may be deleted when reference check fails, got %v", provider.deletedKeys)
		}
	})
}

// 删除失败只告警、下一轮重试：对象仍满足「无行 + 旧」，恢复后下一轮成功删除。
func TestSweepOrphanObjectsRetriesDeleteFailureNextRound(t *testing.T) {
	svc, _, provider := newOrphanSweepService(t)
	ctx := context.Background()
	orphanKey := "published/u1/upload_orphan.png"
	provider.listPagesByPrefix[orphanPublishedObjectPrefix] = []ObjectListPage{{
		Objects: []ObjectSummary{orphanObject(orphanKey, orphanSweepOldAge())},
	}}
	provider.deleteErr = errors.New("storage temporarily unavailable")

	svc.sweepOrphanObjects(ctx)
	if len(provider.deletedKeys) != 1 || provider.deletedKeys[0] != orphanKey {
		t.Fatalf("expected one delete attempt for %q, got %v", orphanKey, provider.deletedKeys)
	}

	// 下一轮同一对象仍在（上一轮删除失败），恢复后必须重试并成功删除。
	provider.listPagesByPrefix[orphanPublishedObjectPrefix] = append(
		provider.listPagesByPrefix[orphanPublishedObjectPrefix],
		ObjectListPage{Objects: []ObjectSummary{orphanObject(orphanKey, orphanSweepOldAge())}})
	provider.deleteErr = nil
	svc.sweepOrphanObjects(ctx)
	if deleted := provider.deletedKeys[1:]; len(deleted) != 1 || deleted[0] != orphanKey {
		t.Fatalf("next round must retry %q, got %v", orphanKey, deleted)
	}
}

// orphanReferenceErrorRepo 注入 ExistingStorageKeys 错误：引用状态不可确认时
// 孤儿清理必须整体放弃本轮（可能误删在途上传/发布副本的对象）。
type orphanReferenceErrorRepo struct {
	*InMemoryRepository
	err error
}

func (r *orphanReferenceErrorRepo) ExistingStorageKeys(context.Context, []string) (map[string]struct{}, error) {
	return nil, r.err
}

// orphanReferenceFailOnceRepo 第一次 ExistingStorageKeys 返回错误（之后走内存实现）：
// 用于验证 published/ 前缀引用查询失败后「游标钉在本页、下轮重扫」的语义。
type orphanReferenceFailOnceRepo struct {
	*InMemoryRepository
	mu     sync.Mutex
	failed bool
}

func (r *orphanReferenceFailOnceRepo) ExistingStorageKeys(ctx context.Context, keys []string) (map[string]struct{}, error) {
	r.mu.Lock()
	shouldFail := !r.failed
	if shouldFail {
		r.failed = true
	}
	r.mu.Unlock()
	if shouldFail {
		return nil, errors.New("reference boom")
	}
	return r.InMemoryRepository.ExistingStorageKeys(ctx, keys)
}

// recordingExistingKeysRepo 记录 ExistingStorageKeys 每次收到的候选键（验证 published/
// 阶段把「发布键 + 同 upload 上传键」一起提交引用判定）。
type recordingExistingKeysRepo struct {
	*InMemoryRepository
	calls [][]string
}

func (r *recordingExistingKeysRepo) ExistingStorageKeys(ctx context.Context, keys []string) (map[string]struct{}, error) {
	r.calls = append(r.calls, append([]string(nil), keys...))
	return r.InMemoryRepository.ExistingStorageKeys(ctx, keys)
}

// runRetentionSweep 两阶段串联：先回收超期未完成会话（对象+行），再清理
// uploads/ 与 published/ 无行引用的孤儿对象；同一轮内两种残留都被回收。
func TestRunRetentionSweepCoversSessionsAndOrphans(t *testing.T) {
	svc, repo, provider := newOrphanSweepService(t)
	ctx := context.Background()
	staleAt := time.Now().UTC().AddDate(0, 0, -(staleUploadRetentionDays + 1))
	if err := repo.Save(ctx, pendingUpload{
		UploadID: "upload_stale_session", UserID: "u1", StorageKey: "uploads/u1/upload_stale_session.bin",
		FileName: "a.bin", DeclaredFileSize: 10, DeclaredMimeType: "application/octet-stream",
		Category: CategoryChatFile, CreatedAt: staleAt.Format(time.RFC3339),
	}); err != nil {
		t.Fatalf("seed stale session: %v", err)
	}
	// 孤儿阶段看到的是会话阶段删除后的列举结果：只剩无行引用的孤儿对象。
	old := orphanSweepOldAge()
	provider.listPagesByPrefix[orphanUploadObjectPrefix] = []ObjectListPage{{
		Objects: []ObjectSummary{orphanObject("uploads/u1/upload_replayed.bin", old)},
	}}
	provider.listPagesByPrefix[orphanPublishedObjectPrefix] = []ObjectListPage{{
		Objects: []ObjectSummary{orphanObject("published/u1/upload_crashed.png", old)},
	}}

	svc.runRetentionSweep(ctx)

	if _, found, err := repo.Get(ctx, "upload_stale_session"); err != nil || found {
		t.Fatalf("stale session must be swept: found=%v err=%v", found, err)
	}
	deleted := map[string]int{}
	for _, key := range provider.deletedKeys {
		deleted[key]++
	}
	if deleted["uploads/u1/upload_stale_session.bin"] != 1 {
		t.Fatalf("stale session object must be deleted once, deleted=%v", provider.deletedKeys)
	}
	if deleted["uploads/u1/upload_replayed.bin"] != 1 {
		t.Fatalf("uploads orphan object must be deleted once, deleted=%v", provider.deletedKeys)
	}
	if deleted["published/u1/upload_crashed.png"] != 1 {
		t.Fatalf("published orphan object must be deleted once, deleted=%v", provider.deletedKeys)
	}
}
