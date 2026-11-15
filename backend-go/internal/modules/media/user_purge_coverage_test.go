package media

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/observability"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/testdb"
)

// purgerRepo 在内存仓储基础上实现 userDataPurger，验证 Service 的类型断言转发。
type purgerRepo struct {
	*InMemoryRepository
	purged []string
	err    error
}

func (r *purgerRepo) PurgeUserData(_ context.Context, userID string) error {
	r.purged = append(r.purged, userID)
	return r.err
}

// purgeCallLog 记录注销级联各步骤的先后顺序（查询 keys → 删行 → 删对象）。
type purgeCallLog struct {
	calls []string
}

func (l *purgeCallLog) add(call string) { l.calls = append(l.calls, call) }

// purgeFakeRepo 在内存仓储基础上同时实现 userDataPurger 与 userStorageKeyLister：
// 返回预设的待删存储键并记录调用顺序，用于验证 Service 层顺序编排与对象删除。
type purgeFakeRepo struct {
	*InMemoryRepository
	storageKeys []string
	listErr     error
	purgeErr    error
	log         *purgeCallLog
}

func (r *purgeFakeRepo) ListUserStorageKeys(context.Context, string) ([]string, error) {
	if r.log != nil {
		r.log.add("list")
	}
	return r.storageKeys, r.listErr
}

func (r *purgeFakeRepo) PurgeUserData(context.Context, string) error {
	if r.log != nil {
		r.log.add("purge")
	}
	return r.purgeErr
}

// purgeFakeProvider 记录对象删除顺序，并可对指定 storage_key 注入删除失败。
type purgeFakeProvider struct {
	*coverageProvider
	log      *purgeCallLog
	failKeys map[string]error
}

func (p *purgeFakeProvider) Delete(_ context.Context, storageKey string) error {
	if p.log != nil {
		p.log.add("delete:" + storageKey)
	}
	if err, ok := p.failKeys[storageKey]; ok {
		return err
	}
	return nil
}

// captureStderr 捕获 fn 执行期间写入进程 stderr 的内容（用于断言 Warn 日志）；
// 结束后恢复原 stderr，读 goroutine 与 buf 通过 channel 关闭同步。
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	original := os.Stderr
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stderr pipe: %v", err)
	}
	os.Stderr = writer
	defer func() { os.Stderr = original }()
	var (
		buf  bytes.Buffer
		done = make(chan struct{})
	)
	go func() {
		defer close(done)
		_, _ = buf.ReadFrom(reader)
	}()
	fn()
	if err := writer.Close(); err != nil {
		t.Fatalf("close stderr writer: %v", err)
	}
	<-done
	_ = reader.Close()
	return buf.String()
}

func TestPostgresRepositoryPurgeUserDataWithoutDatabase(t *testing.T) {
	if err := NewPostgresRepository(nil).PurgeUserData(context.Background(), "user_mediacov"); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("expected ErrNotImplemented, got %v", err)
	}
	if _, err := NewPostgresRepository(nil).ListUserStorageKeys(context.Background(), "user_mediacov"); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("expected ErrNotImplemented from ListUserStorageKeys, got %v", err)
	}
}

// ListUserStorageKeys 的防御分支（SQL 语义由真实库 IT 覆盖；此处只覆盖查询/
// 扫描/迭代错误传播与正常路径）。
func TestPostgresRepositoryListUserStorageKeysPaths(t *testing.T) {
	ctx := context.Background()

	queryErr := errors.New("list keys query boom")
	queryBackend := &mediaScriptedBackend{rowsErr: map[string]error{queryListUserStorageKeys: queryErr}}
	if _, err := NewPostgresRepository(newMediaFactory(queryBackend)).ListUserStorageKeys(ctx, "u1"); !errors.Is(err, queryErr) {
		t.Fatalf("query error = %v, want %v", err, queryErr)
	}

	scanErr := errors.New("list keys scan boom")
	scanBackend := &mediaScriptedBackend{rowsByQuery: map[string]persistence.Rows{
		queryListUserStorageKeys: &mediaScriptedRows{values: [][]any{{"published/u1/a.jpg"}}, scanErr: scanErr},
	}}
	if _, err := NewPostgresRepository(newMediaFactory(scanBackend)).ListUserStorageKeys(ctx, "u1"); !errors.Is(err, scanErr) {
		t.Fatalf("scan error = %v, want %v", err, scanErr)
	}

	rowsErr := errors.New("list keys rows boom")
	errBackend := &mediaScriptedBackend{rowsByQuery: map[string]persistence.Rows{
		queryListUserStorageKeys: &mediaScriptedRows{finalErr: rowsErr},
	}}
	if _, err := NewPostgresRepository(newMediaFactory(errBackend)).ListUserStorageKeys(ctx, "u1"); !errors.Is(err, rowsErr) {
		t.Fatalf("rows error = %v, want %v", err, rowsErr)
	}

	okBackend := &mediaScriptedBackend{rowsByQuery: map[string]persistence.Rows{
		queryListUserStorageKeys: &mediaScriptedRows{values: [][]any{
			{"published/u1/a.jpg"},
			{"uploads/u1/b.png"},
		}},
	}}
	keys, err := NewPostgresRepository(newMediaFactory(okBackend)).ListUserStorageKeys(ctx, "u1")
	if err != nil {
		t.Fatalf("list keys: %v", err)
	}
	if !slices.Equal(keys, []string{"published/u1/a.jpg", "uploads/u1/b.png"}) {
		t.Fatalf("keys = %v", keys)
	}
}

func TestServicePurgeUserDataBranches(t *testing.T) {
	ctx := context.Background()
	// 无 purger 能力的仓储（内存实现）：账户注销级联为 no-op。
	if err := NewServiceWithRepoAndProvider(NewInMemoryRepository(), &coverageProvider{}).PurgeUserData(ctx, "user_mediacov"); err != nil {
		t.Fatalf("non-purger repository must be a no-op, got %v", err)
	}

	repo := &purgerRepo{InMemoryRepository: NewInMemoryRepository()}
	svc := NewServiceWithRepoAndProvider(repo, &coverageProvider{})
	if err := svc.PurgeUserData(ctx, "user_mediacov"); err != nil {
		t.Fatalf("purger repository must succeed, got %v", err)
	}
	if len(repo.purged) != 1 || repo.purged[0] != "user_mediacov" {
		t.Fatalf("purge calls = %v", repo.purged)
	}

	purgeErr := errors.New("purge boom")
	failing := &purgerRepo{InMemoryRepository: NewInMemoryRepository(), err: purgeErr}
	if err := NewServiceWithRepoAndProvider(failing, &coverageProvider{}).PurgeUserData(ctx, "user_mediacov"); !errors.Is(err, purgeErr) {
		t.Fatalf("purge error must propagate, got %v", err)
	}
}

// 注销级联顺序：先查询待删对象键，再删行，最后逐条删对象；completed 的发布键
// 与未完成的临时上传键都在清单内。
func TestServicePurgeUserDataListsKeysThenDeletesRowsThenObjects(t *testing.T) {
	log := &purgeCallLog{}
	keys := []string{"published/u1/upload_a.jpg", "uploads/u1/upload_b.png"}
	repo := &purgeFakeRepo{InMemoryRepository: NewInMemoryRepository(), storageKeys: keys, log: log}
	provider := &purgeFakeProvider{coverageProvider: &coverageProvider{}, log: log}
	if err := NewServiceWithRepoAndProvider(repo, provider).PurgeUserData(context.Background(), "u1"); err != nil {
		t.Fatalf("purge: %v", err)
	}
	want := []string{"list", "purge", "delete:" + keys[0], "delete:" + keys[1]}
	if !slices.Equal(log.calls, want) {
		t.Fatalf("调用顺序 = %v, want %v", log.calls, want)
	}
}

// 单个对象删除失败不阻塞注销：仍返回 nil、继续删除其余对象、并记录 Warn。
func TestServicePurgeUserDataObjectDeleteFailureWarnsAndContinues(t *testing.T) {
	log := &purgeCallLog{}
	keys := []string{"published/u1/a.jpg", "uploads/u1/b.png", "uploads/u1/c.png"}
	repo := &purgeFakeRepo{InMemoryRepository: NewInMemoryRepository(), storageKeys: keys, log: log}
	provider := &purgeFakeProvider{
		coverageProvider: &coverageProvider{},
		log:              log,
		failKeys:         map[string]error{keys[0]: errors.New("s3 delete boom")},
	}
	var (
		purgeErr  error
		loggerErr error
	)
	// logger 必须在 stderr 重定向之后创建：zap 在 Build 时按路径打开输出，
	// 提前创建会绑定到原 stderr 而捕获不到 Warn。
	stderr := captureStderr(t, func() {
		logger, err := observability.NewLogger("development", "warn")
		if err != nil {
			loggerErr = err
			return
		}
		ctx := observability.WithLogger(context.Background(), logger)
		purgeErr = NewServiceWithRepoAndProvider(repo, provider).PurgeUserData(ctx, "u1")
	})
	if loggerErr != nil {
		t.Fatalf("new logger: %v", loggerErr)
	}
	if purgeErr != nil {
		t.Fatalf("对象删除失败必须不阻塞注销，got %v", purgeErr)
	}
	want := []string{"list", "purge", "delete:" + keys[0], "delete:" + keys[1], "delete:" + keys[2]}
	if !slices.Equal(log.calls, want) {
		t.Fatalf("失败后必须继续删除其余对象，调用顺序 = %v, want %v", log.calls, want)
	}
	if !strings.Contains(stderr, "account purge media object delete failed") {
		t.Fatalf("对象删除失败应记录 Warn，stderr = %q", stderr)
	}
	if !strings.Contains(stderr, keys[0]) {
		t.Fatalf("Warn 应带失败的 storage_key，stderr = %q", stderr)
	}
}

// 查询待删对象键失败时中止，不删行也不删对象（编排层可整体重试）。
func TestServicePurgeUserDataListFailureAbortsBeforeRowDelete(t *testing.T) {
	log := &purgeCallLog{}
	listErr := errors.New("list storage keys boom")
	repo := &purgeFakeRepo{InMemoryRepository: NewInMemoryRepository(), listErr: listErr, log: log}
	provider := &purgeFakeProvider{coverageProvider: &coverageProvider{}, log: log}
	err := NewServiceWithRepoAndProvider(repo, provider).PurgeUserData(context.Background(), "u1")
	if !errors.Is(err, listErr) {
		t.Fatalf("list error must propagate, got %v", err)
	}
	if !slices.Equal(log.calls, []string{"list"}) {
		t.Fatalf("查询失败不得继续删行/删对象，调用顺序 = %v", log.calls)
	}
}

// 删行失败时不删对象（保留对象与行一致的完整状态供整体重试）。
func TestServicePurgeUserDataRowDeleteFailureSkipsObjectDeletes(t *testing.T) {
	log := &purgeCallLog{}
	purgeErr := errors.New("delete rows boom")
	repo := &purgeFakeRepo{
		InMemoryRepository: NewInMemoryRepository(),
		storageKeys:        []string{"published/u1/a.jpg"},
		purgeErr:           purgeErr,
		log:                log,
	}
	provider := &purgeFakeProvider{coverageProvider: &coverageProvider{}, log: log}
	err := NewServiceWithRepoAndProvider(repo, provider).PurgeUserData(context.Background(), "u1")
	if !errors.Is(err, purgeErr) {
		t.Fatalf("purge error must propagate, got %v", err)
	}
	if !slices.Equal(log.calls, []string{"list", "purge"}) {
		t.Fatalf("删行失败不得删对象，调用顺序 = %v", log.calls)
	}
}

// 对象删除按批推进且全部删完；空 storage_key 防御性跳过（不向存储发空键请求）。
func TestServicePurgeUserDataDeletesObjectsInBatches(t *testing.T) {
	keys := make([]string, 0, 2*purgeObjectDeleteBatchSize+5)
	for i := 0; i < 2*purgeObjectDeleteBatchSize+4; i++ {
		keys = append(keys, fmt.Sprintf("published/u1/%d.jpg", i))
	}
	keys = append(keys, "")
	repo := &purgeFakeRepo{InMemoryRepository: NewInMemoryRepository(), storageKeys: keys}
	provider := &coverageProvider{}
	if err := NewServiceWithRepoAndProvider(repo, provider).PurgeUserData(context.Background(), "u1"); err != nil {
		t.Fatalf("purge: %v", err)
	}
	if len(provider.deletedKeys) != len(keys)-1 {
		t.Fatalf("deleted objects = %d, want %d", len(provider.deletedKeys), len(keys)-1)
	}
	if slices.Contains(provider.deletedKeys, "") {
		t.Fatalf("空 storage_key 不得发送删除请求: %v", provider.deletedKeys)
	}
}

// 临时库真实库用例（BeginScope：事务内写入、结束回滚，不提交任何行）。
// 断言：删行前 ListUserStorageKeys 同时返回 completed 的发布键与未完成的临时
// 上传键；Service 删除行后逐条删除全部对象；范围外用户的行与键不受影响；幂等。
func TestPostgresRepositoryPurgeUserDataDeletesOnlyTargetUser(t *testing.T) {
	testdb.SkipIfShort(t)
	testdb.Open(t)
	factory, cleanup := testdb.BeginScope(t)
	defer cleanup()

	repo := NewPostgresRepository(factory)
	ctx := context.Background()
	const targetUser = "mediacov_purge_user"
	const otherUser = "mediacov_purge_other"

	// seed 写入一条上传会话；completed 的会话 storage_key 为 copy-on-complete
	// 的发布键，未完成会话为临时上传键。返回该行最终引用的存储键。
	seed := func(t *testing.T, uploadID, userID string, completed bool) string {
		t.Helper()
		uploadKey := "uploads/" + userID + "/" + uploadID + ".png"
		if err := repo.Save(ctx, pendingUpload{
			UploadID:         uploadID,
			UserID:           userID,
			StorageKey:       uploadKey,
			FileName:         uploadID + ".png",
			DeclaredFileSize: 100,
			DeclaredMimeType: "image/png",
			Category:         CategoryAvatar,
			CreatedAt:        time.Now().UTC().Format(time.RFC3339),
		}); err != nil {
			t.Fatalf("save %s: %v", uploadID, err)
		}
		if !completed {
			return uploadKey
		}
		publishedKey := "published/" + userID + "/" + uploadID + ".png"
		updated, err := repo.Complete(ctx, uploadID, UploadedMedia{
			FileID:   "file_" + uploadID,
			FileSize: 100,
			MimeType: "image/png",
		}, publishedKey, "")
		if err != nil {
			t.Fatalf("complete %s: %v", uploadID, err)
		}
		if !updated {
			t.Fatalf("complete %s: row not updated", uploadID)
		}
		return publishedKey
	}
	targetPendingKey := seed(t, "mediacov_purge_upload_a", targetUser, false)
	targetPublishedKey := seed(t, "mediacov_purge_upload_b", targetUser, true)
	otherPendingKey := seed(t, "mediacov_purge_upload_other", otherUser, false)
	otherPublishedKey := seed(t, "mediacov_purge_upload_other_done", otherUser, true)

	targetKeys, err := repo.ListUserStorageKeys(ctx, targetUser)
	if err != nil {
		t.Fatalf("list target storage keys: %v", err)
	}
	slices.Sort(targetKeys)
	wantTargetKeys := []string{targetPendingKey, targetPublishedKey}
	slices.Sort(wantTargetKeys)
	if !slices.Equal(targetKeys, wantTargetKeys) {
		t.Fatalf("target storage keys = %v, want %v", targetKeys, wantTargetKeys)
	}

	// Service 层在真实仓储上执行注销级联：删行 + fake provider 删除全部对象。
	provider := &coverageProvider{}
	svc := NewServiceWithRepoAndProvider(repo, provider)
	if err := svc.PurgeUserData(ctx, targetUser); err != nil {
		t.Fatalf("purge user data: %v", err)
	}
	for _, uploadID := range []string{"mediacov_purge_upload_a", "mediacov_purge_upload_b"} {
		if _, found, err := repo.Get(ctx, uploadID); err != nil || found {
			t.Fatalf("upload %s must be purged: found=%v err=%v", uploadID, found, err)
		}
	}
	if remaining, err := repo.ListUserStorageKeys(ctx, targetUser); err != nil || len(remaining) != 0 {
		t.Fatalf("target storage keys after purge = %v, err=%v", remaining, err)
	}
	deletedKeys := append([]string(nil), provider.deletedKeys...)
	slices.Sort(deletedKeys)
	if !slices.Equal(deletedKeys, wantTargetKeys) {
		t.Fatalf("deleted objects = %v, want %v", deletedKeys, wantTargetKeys)
	}
	// 范围外用户：行与键都必须保留。
	otherKeys, err := repo.ListUserStorageKeys(ctx, otherUser)
	if err != nil {
		t.Fatalf("list other storage keys: %v", err)
	}
	slices.Sort(otherKeys)
	wantOtherKeys := []string{otherPendingKey, otherPublishedKey}
	slices.Sort(wantOtherKeys)
	if !slices.Equal(otherKeys, wantOtherKeys) {
		t.Fatalf("other user's storage keys = %v, want %v", otherKeys, wantOtherKeys)
	}
	if _, found, err := repo.Get(ctx, "mediacov_purge_upload_other"); err != nil || !found {
		t.Fatalf("other user's upload must survive: found=%v err=%v", found, err)
	}

	// 幂等：重复调用不报错，且行已删后不会再产生删除请求。
	deletedBefore := len(provider.deletedKeys)
	if err := svc.PurgeUserData(ctx, targetUser); err != nil {
		t.Fatalf("second purge must be idempotent, got %v", err)
	}
	if len(provider.deletedKeys) != deletedBefore {
		t.Fatalf("second purge must not delete objects again: %v", provider.deletedKeys[deletedBefore:])
	}

	// 另一用户经 Service 级联清理同样删行删对象。
	if err := svc.PurgeUserData(ctx, otherUser); err != nil {
		t.Fatalf("service purge other user: %v", err)
	}
	if _, found, err := repo.Get(ctx, "mediacov_purge_upload_other"); err != nil || found {
		t.Fatalf("service purge must delete other user's rows: found=%v err=%v", found, err)
	}
}
