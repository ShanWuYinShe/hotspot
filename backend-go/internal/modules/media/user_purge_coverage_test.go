package media

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/observability"
	"hotspot/backend-go/internal/platform/outbox"
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
// 对象删除 handler 内部有界并发执行，add 必须加锁；需要断言顺序的用例用
// snapshot 取副本再排序。
type purgeCallLog struct {
	mu    sync.Mutex
	calls []string
}

func (l *purgeCallLog) add(call string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.calls = append(l.calls, call)
}

func (l *purgeCallLog) snapshot() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]string(nil), l.calls...)
}

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

// purgeFakeProvider 记录对象删除顺序与被删键，并可对指定 storage_key 注入
// 删除失败；Delete 会被 handler 并发调用，deleted 访问加锁。
type purgeFakeProvider struct {
	*coverageProvider
	mu       sync.Mutex
	log      *purgeCallLog
	failKeys map[string]error
	deleted  []string
}

func (p *purgeFakeProvider) Delete(_ context.Context, storageKey string) error {
	if p.log != nil {
		p.log.add("delete:" + storageKey)
	}
	if err, ok := p.failKeys[storageKey]; ok {
		return err
	}
	p.mu.Lock()
	p.deleted = append(p.deleted, storageKey)
	p.mu.Unlock()
	return nil
}

func (p *purgeFakeProvider) deletedStorageKeys() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]string(nil), p.deleted...)
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

// purgeEnqueueCall 记录一次 outbox 入队（topic + payload），用于断言异步路径。
type purgeEnqueueCall struct {
	topic   string
	payload PurgeObjectsPayload
}

// purgeRecordingOutbox 是 outboxWriter 的测试替身：记录入队事件，可对第 errAt
// 次调用注入失败（1-based，0 表示不失败）。
type purgeRecordingOutbox struct {
	calls []purgeEnqueueCall
	errAt int
	err   error
}

func (o *purgeRecordingOutbox) Enqueue(_ context.Context, topic string, payload any) error {
	call := purgeEnqueueCall{topic: topic}
	if event, ok := payload.(PurgeObjectsPayload); ok {
		call.payload = event
	}
	o.calls = append(o.calls, call)
	if o.errAt > 0 && len(o.calls) == o.errAt {
		if o.err != nil {
			return o.err
		}
		return errors.New("outbox enqueue boom")
	}
	return nil
}

// enqueuedKeys 按入队顺序展开所有事件里的 keys。
func (o *purgeRecordingOutbox) enqueuedKeys() []string {
	keys := make([]string, 0)
	for _, call := range o.calls {
		keys = append(keys, call.payload.Keys...)
	}
	return keys
}

// 注入 outbox writer 后注销级联不再同步删对象：删行后按批入队
// TopicMediaPurge（payload 带 user_id 与完整 keys），对象清理交给 dispatcher。
func TestServicePurgeUserDataEnqueuesObjectsInsteadOfDeleting(t *testing.T) {
	log := &purgeCallLog{}
	keys := []string{"published/u1/a.jpg", "uploads/u1/b.png"}
	repo := &purgeFakeRepo{InMemoryRepository: NewInMemoryRepository(), storageKeys: keys, log: log}
	provider := &purgeFakeProvider{coverageProvider: &coverageProvider{}, log: log}
	writer := &purgeRecordingOutbox{}
	svc := NewServiceWithRepoAndProvider(repo, provider)
	svc.SetEventOutbox(writer)
	if err := svc.PurgeUserData(context.Background(), "u1"); err != nil {
		t.Fatalf("purge: %v", err)
	}
	if want := []string{"list", "purge"}; !slices.Equal(log.calls, want) {
		t.Fatalf("异步路径不得同步删对象，调用顺序 = %v, want %v", log.calls, want)
	}
	if len(provider.deletedKeys) != 0 {
		t.Fatalf("异步路径不得调用 provider.Delete: %v", provider.deletedKeys)
	}
	if len(writer.calls) != 1 {
		t.Fatalf("enqueued events = %d, want 1", len(writer.calls))
	}
	call := writer.calls[0]
	if call.topic != outbox.TopicMediaPurge {
		t.Fatalf("topic = %q, want %q", call.topic, outbox.TopicMediaPurge)
	}
	if call.payload.UserID != "u1" {
		t.Fatalf("payload user_id = %q, want u1", call.payload.UserID)
	}
	if !slices.Equal(call.payload.Keys, keys) {
		t.Fatalf("payload keys = %v, want %v", call.payload.Keys, keys)
	}
}

// 单事件 key 数上限：超过批大小时拆多个事件，事件并集与原清单一致。
func TestServicePurgeUserDataEnqueueBatchesKeys(t *testing.T) {
	keys := make([]string, 0, 2*purgeObjectEventBatchSize+1)
	for i := 0; i < 2*purgeObjectEventBatchSize+1; i++ {
		keys = append(keys, fmt.Sprintf("published/u1/%d.jpg", i))
	}
	repo := &purgeFakeRepo{InMemoryRepository: NewInMemoryRepository(), storageKeys: keys}
	provider := &coverageProvider{}
	writer := &purgeRecordingOutbox{}
	svc := NewServiceWithRepoAndProvider(repo, provider)
	svc.SetEventOutbox(writer)
	if err := svc.PurgeUserData(context.Background(), "u1"); err != nil {
		t.Fatalf("purge: %v", err)
	}
	if len(writer.calls) != 3 {
		t.Fatalf("enqueued events = %d, want 3", len(writer.calls))
	}
	wantSizes := []int{purgeObjectEventBatchSize, purgeObjectEventBatchSize, 1}
	for i, call := range writer.calls {
		if len(call.payload.Keys) != wantSizes[i] {
			t.Fatalf("event %d keys = %d, want %d", i, len(call.payload.Keys), wantSizes[i])
		}
	}
	if got := writer.enqueuedKeys(); !slices.Equal(got, keys) {
		t.Fatalf("事件 keys 并集与清单不一致: got %d keys, want %d", len(got), len(keys))
	}
	if len(provider.deletedKeys) != 0 {
		t.Fatalf("异步路径不得调用 provider.Delete: %v", provider.deletedKeys)
	}
}

// 入队失败回退同步删除（注销不因异步化失败而失败，也不留孤儿对象），并记
// Warn 日志（带失败原因与 user_id）。
func TestServicePurgeUserDataEnqueueFailureFallsBackToSyncDelete(t *testing.T) {
	keys := []string{"published/u1/a.jpg", "uploads/u1/b.png", "uploads/u1/c.png"}
	repo := &purgeFakeRepo{InMemoryRepository: NewInMemoryRepository(), storageKeys: keys}
	provider := &coverageProvider{}
	writer := &purgeRecordingOutbox{errAt: 1, err: errors.New("outbox db down")}
	svc := NewServiceWithRepoAndProvider(repo, provider)
	svc.SetEventOutbox(writer)
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
		purgeErr = svc.PurgeUserData(ctx, "u1")
	})
	if loggerErr != nil {
		t.Fatalf("new logger: %v", loggerErr)
	}
	if purgeErr != nil {
		t.Fatalf("入队失败必须回退同步删除而不是让注销失败，got %v", purgeErr)
	}
	if len(writer.calls) != 1 {
		t.Fatalf("入队尝试次数 = %d, want 1", len(writer.calls))
	}
	if !slices.Equal(provider.deletedKeys, keys) {
		t.Fatalf("回退同步删除应删全部键: %v, want %v", provider.deletedKeys, keys)
	}
	if !strings.Contains(stderr, "outbox enqueue failed") {
		t.Fatalf("入队失败应记录 Warn，stderr = %q", stderr)
	}
	if !strings.Contains(stderr, "user_id") {
		t.Fatalf("Warn 应带 user_id，stderr = %q", stderr)
	}
}

// 入队中途失败：只回退「尚未入队」的剩余键，已入队批次不重复同步删除。
func TestServicePurgeUserDataEnqueuePartialFailureDeletesOnlyRemainingKeys(t *testing.T) {
	keys := make([]string, 0, 2*purgeObjectEventBatchSize+1)
	for i := 0; i < 2*purgeObjectEventBatchSize+1; i++ {
		keys = append(keys, fmt.Sprintf("published/u1/%d.jpg", i))
	}
	repo := &purgeFakeRepo{InMemoryRepository: NewInMemoryRepository(), storageKeys: keys}
	provider := &coverageProvider{}
	writer := &purgeRecordingOutbox{errAt: 2, err: errors.New("outbox db down")}
	svc := NewServiceWithRepoAndProvider(repo, provider)
	svc.SetEventOutbox(writer)
	if err := svc.PurgeUserData(context.Background(), "u1"); err != nil {
		t.Fatalf("purge: %v", err)
	}
	if len(writer.calls) != 2 {
		t.Fatalf("入队尝试次数 = %d, want 2", len(writer.calls))
	}
	// 第二次调用已失败，其 keys 未入队：只有第一个事件的 keys 生效。
	wantEnqueued := keys[0:purgeObjectEventBatchSize]
	if got := writer.calls[0].payload.Keys; !slices.Equal(got, wantEnqueued) {
		t.Fatalf("已入队 keys 数 = %d, want %d", len(got), len(wantEnqueued))
	}
	wantDeleted := keys[purgeObjectEventBatchSize:]
	if !slices.Equal(provider.deletedKeys, wantDeleted) {
		t.Fatalf("回退只删剩余键: got %d keys, want %d", len(provider.deletedKeys), len(wantDeleted))
	}
}

// 空 storage_key 不产生事件也不发删除请求（防御空键）。
func TestServicePurgeUserDataEmptyStorageKeysSkipsCleanup(t *testing.T) {
	repo := &purgeFakeRepo{InMemoryRepository: NewInMemoryRepository(), storageKeys: []string{"", ""}}
	provider := &coverageProvider{}
	writer := &purgeRecordingOutbox{}
	svc := NewServiceWithRepoAndProvider(repo, provider)
	svc.SetEventOutbox(writer)
	if err := svc.PurgeUserData(context.Background(), "u1"); err != nil {
		t.Fatalf("purge: %v", err)
	}
	if len(writer.calls) != 0 {
		t.Fatalf("空键不得入队: %#v", writer.calls)
	}
	if len(provider.deletedKeys) != 0 {
		t.Fatalf("空键不得触发删除: %v", provider.deletedKeys)
	}
}

// DeletePurgeObjects 是 outbox handler 的删除逻辑：空键跳过、空清单成功、非空键
// 全部删除。handler 内部有界并发，删除完成顺序不保证，按集合比较。
func TestServiceDeletePurgeObjects(t *testing.T) {
	provider := &purgeFakeProvider{failKeys: map[string]error{}}
	svc := NewServiceWithRepoAndProvider(NewInMemoryRepository(), provider)
	ctx := context.Background()
	if err := svc.DeletePurgeObjects(ctx, "u1", nil); err != nil {
		t.Fatalf("空 keys 直接成功: %v", err)
	}
	if err := svc.DeletePurgeObjects(ctx, "u1", []string{"", ""}); err != nil {
		t.Fatalf("全空 keys 直接成功: %v", err)
	}
	if got := provider.deletedStorageKeys(); len(got) != 0 {
		t.Fatalf("空清单不得发删除请求: %v", got)
	}
	if err := svc.DeletePurgeObjects(ctx, "u1", []string{"published/u1/a.jpg", "", "uploads/u1/b.png"}); err != nil {
		t.Fatalf("delete purge objects: %v", err)
	}
	got := provider.deletedStorageKeys()
	sort.Strings(got)
	want := []string{"published/u1/a.jpg", "uploads/u1/b.png"}
	if !slices.Equal(got, want) {
		t.Fatalf("deleted keys = %v, want %v", got, want)
	}
}

// 单条删除失败：继续删其余对象并返回错误（outbox 据此重试，已删对象 404 幂等）。
func TestServiceDeletePurgeObjectsFailureReturnsErrorForRetry(t *testing.T) {
	boom := errors.New("s3 delete boom")
	log := &purgeCallLog{}
	provider := &purgeFakeProvider{
		coverageProvider: &coverageProvider{},
		log:              log,
		failKeys:         map[string]error{"b": boom},
	}
	svc := NewServiceWithRepoAndProvider(NewInMemoryRepository(), provider)
	err := svc.DeletePurgeObjects(context.Background(), "u1", []string{"a", "b", "c"})
	if !errors.Is(err, boom) {
		t.Fatalf("删除失败必须返回错误供 outbox 重试, got %v", err)
	}
	if !strings.Contains(err.Error(), "u1") {
		t.Fatalf("错误应带 user_id 便于排查: %v", err)
	}
	gotCalls := log.snapshot()
	sort.Strings(gotCalls)
	want := []string{"delete:a", "delete:b", "delete:c"}
	if !slices.Equal(gotCalls, want) {
		t.Fatalf("单条失败不得中断其余删除: %v, want %v", gotCalls, want)
	}
}

// 真实库用例（BeginScope 事务内写入、结束回滚，不提交任何行）：注入 outbox
// writer 后 PurgeUserData 把对象清理按批写进 outbox_events（注销请求不再同步
// 删对象）——事件 topic/payload 完整覆盖全部 storage_key，且会话行已删。
func TestPostgresRepositoryPurgeUserDataEnqueuesOutboxEvents(t *testing.T) {
	testdb.SkipIfShort(t)
	testdb.Open(t)
	factory, cleanup := testdb.BeginScope(t)
	defer cleanup()

	repo := NewPostgresRepository(factory)
	ctx := context.Background()
	const targetUser = "mediacov_purge_outbox_user"
	for _, uploadID := range []string{"mediacov_purge_outbox_a", "mediacov_purge_outbox_b"} {
		if err := repo.Save(ctx, pendingUpload{
			UploadID:         uploadID,
			UserID:           targetUser,
			StorageKey:       "uploads/" + targetUser + "/" + uploadID + ".png",
			FileName:         uploadID + ".png",
			DeclaredFileSize: 100,
			DeclaredMimeType: "image/png",
			Category:         CategoryAvatar,
			CreatedAt:        time.Now().UTC().Format(time.RFC3339),
		}); err != nil {
			t.Fatalf("save %s: %v", uploadID, err)
		}
	}
	wantKeys, err := repo.ListUserStorageKeys(ctx, targetUser)
	if err != nil {
		t.Fatalf("list storage keys: %v", err)
	}
	slices.Sort(wantKeys)

	provider := &coverageProvider{}
	svc := NewServiceWithRepoAndProvider(repo, provider)
	svc.SetEventOutbox(outbox.NewWriter(factory))
	if err := svc.PurgeUserData(ctx, targetUser); err != nil {
		t.Fatalf("purge user data: %v", err)
	}
	if len(provider.deletedKeys) != 0 {
		t.Fatalf("异步路径不得同步删对象: %v", provider.deletedKeys)
	}
	if remaining, err := repo.ListUserStorageKeys(ctx, targetUser); err != nil || len(remaining) != 0 {
		t.Fatalf("会话行必须已删: keys=%v err=%v", remaining, err)
	}

	qdb, ok := factory.DB.(persistence.QueryableDB)
	if !ok {
		t.Fatal("factory.DB is not QueryableDB")
	}
	rows, err := qdb.Query(ctx, "SELECT payload FROM outbox_events WHERE topic = $1 AND payload->>'user_id' = $2", outbox.TopicMediaPurge, targetUser)
	if err != nil {
		t.Fatalf("query outbox events: %v", err)
	}
	defer rows.Close()
	payloads := make([]PurgeObjectsPayload, 0)
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			t.Fatalf("scan outbox payload: %v", err)
		}
		var payload PurgeObjectsPayload
		if err := json.Unmarshal(raw, &payload); err != nil {
			t.Fatalf("decode outbox payload: %v", err)
		}
		payloads = append(payloads, payload)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate outbox events: %v", err)
	}
	if len(payloads) != 1 {
		t.Fatalf("outbox events = %d, want 1", len(payloads))
	}
	gotKeys := append([]string(nil), payloads[0].Keys...)
	slices.Sort(gotKeys)
	if payloads[0].UserID != targetUser || !slices.Equal(gotKeys, wantKeys) {
		t.Fatalf("outbox payload = user:%q keys:%v, want user:%q keys:%v", payloads[0].UserID, gotKeys, targetUser, wantKeys)
	}
}
