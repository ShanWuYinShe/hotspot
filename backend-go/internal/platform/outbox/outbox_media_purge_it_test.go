package outbox_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"hotspot/backend-go/internal/modules/media"
	"hotspot/backend-go/internal/platform/outbox"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/testdb"
)

// purgeDeleteProvider 是注销级联对象删除的对象存储替身：只关心 Delete，
// 记录被删 storage_key（DeletePurgeObjects 并发调用，必须加锁）。
type purgeDeleteProvider struct {
	mu      sync.Mutex
	deleted []string
	failOn  map[string]error
}

func (p *purgeDeleteProvider) Presign(context.Context, media.PresignFileInput, string) (media.PresignedUpload, error) {
	return media.PresignedUpload{}, errors.New("not implemented")
}

func (p *purgeDeleteProvider) PresignGet(context.Context, string, time.Duration) (string, error) {
	return "", errors.New("not implemented")
}

func (p *purgeDeleteProvider) Head(context.Context, string) (media.ObjectMetadata, error) {
	return media.ObjectMetadata{}, errors.New("not implemented")
}

func (p *purgeDeleteProvider) ReadPrefix(context.Context, string, int64) ([]byte, error) {
	return nil, errors.New("not implemented")
}

func (p *purgeDeleteProvider) Delete(_ context.Context, storageKey string) error {
	p.mu.Lock()
	p.deleted = append(p.deleted, storageKey)
	err := p.failOn[storageKey]
	p.mu.Unlock()
	return err
}

func (p *purgeDeleteProvider) GetObject(context.Context, string) ([]byte, error) {
	return nil, errors.New("not implemented")
}

func (p *purgeDeleteProvider) PutObject(context.Context, string, string, []byte) error {
	return errors.New("not implemented")
}

func (p *purgeDeleteProvider) CopyObject(context.Context, string, string) error {
	return errors.New("not implemented")
}

func (p *purgeDeleteProvider) deletedKeys() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]string(nil), p.deleted...)
}

// mediaPurgeTopicHandler 复刻 bootstrap 的 mediaPurgeOutboxHandler 装配：
// 解码 media.PurgeObjectsPayload 后调用 Service.DeletePurgeObjects。
func mediaPurgeTopicHandler(svc *media.Service) outbox.Handler {
	return func(ctx context.Context, payload json.RawMessage) error {
		var event media.PurgeObjectsPayload
		if err := json.Unmarshal(payload, &event); err != nil {
			return fmt.Errorf("decode media purge payload: %w", err)
		}
		if event.UserID == "" {
			return errors.New("invalid media purge payload: missing user_id")
		}
		return svc.DeletePurgeObjects(ctx, event.UserID, event.Keys)
	}
}

// 真实库用例：注销媒体对象清理事件以事务外 topic（RegisterExternal）投递时，
// dispatcher 能正常领取/租约/执行/写回，handler 完成对象删除且事件被标记
// processed（第 82 轮 P1-4 改造后的端到端行为）。
func TestPostgresIT_MediaPurgeHandlerDeletesObjectsViaExternalTopic(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	ctx := context.Background()
	qdb, ok := factory.DB.(persistence.QueryableDB)
	if !ok {
		t.Fatal("factory.DB is not QueryableDB")
	}
	const userID = "it_outbox_media_purge_user"
	t.Cleanup(func() {
		_ = qdb.Exec(ctx, "DELETE FROM outbox_events WHERE topic = $1 AND payload->>'user_id' = $2", outbox.TopicMediaPurge, userID)
	})

	keys := []string{
		"published/" + userID + "/a.jpg",
		"published/" + userID + "/b.jpg",
		"uploads/" + userID + "/c.png",
	}
	provider := &purgeDeleteProvider{}
	// 只测 handler 的对象删除路径（不触仓储）：repo 传 nil，Service 只用 provider。
	svc := media.NewServiceWithRepoAndProvider(nil, provider)
	if err := outbox.NewWriter(factory).Enqueue(ctx, outbox.TopicMediaPurge, media.PurgeObjectsPayload{
		UserID: userID,
		Keys:   keys,
	}); err != nil {
		t.Fatalf("enqueue purge event: %v", err)
	}

	dispatcher := outbox.NewDispatcher(factory, nil)
	dispatcher.RegisterExternal(outbox.TopicMediaPurge, mediaPurgeTopicHandler(svc))
	if _, err := dispatcher.DrainOnce(ctx); err != nil {
		t.Fatalf("drain: %v", err)
	}
	dispatcher.Wait()

	got := provider.deletedKeys()
	slices.Sort(got)
	want := append([]string(nil), keys...)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Fatalf("deleted keys = %v, want %v", got, want)
	}

	var attempts int
	var processedAt *time.Time
	var lastError *string
	if err := qdb.QueryRow(ctx,
		"SELECT attempts, processed_at, last_error FROM outbox_events WHERE topic = $1 AND payload->>'user_id' = $2",
		outbox.TopicMediaPurge, userID).Scan(&attempts, &processedAt, &lastError); err != nil {
		t.Fatalf("read purge event row: %v", err)
	}
	if processedAt == nil || attempts != 1 || lastError != nil {
		t.Fatalf("事件行 = processed:%v attempts:%d last_error:%v, want 非空/1/nil", processedAt, attempts, lastError)
	}
}

// 真实库用例：注销清理 handler 删除失败时返回错误，outbox 按既有语义写回
// attempts/退避/last_error（事务外路径的重试语义与事务内路径一致）。
func TestPostgresIT_MediaPurgeHandlerFailureSchedulesRetry(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	ctx := context.Background()
	qdb, ok := factory.DB.(persistence.QueryableDB)
	if !ok {
		t.Fatal("factory.DB is not QueryableDB")
	}
	const userID = "it_outbox_media_purge_retry_user"
	t.Cleanup(func() {
		_ = qdb.Exec(ctx, "DELETE FROM outbox_events WHERE topic = $1 AND payload->>'user_id' = $2", outbox.TopicMediaPurge, userID)
	})
	boomKey := "published/" + userID + "/boom.jpg"
	provider := &purgeDeleteProvider{failOn: map[string]error{boomKey: errors.New("s3 unavailable")}}
	svc := media.NewServiceWithRepoAndProvider(nil, provider)
	if err := outbox.NewWriter(factory).Enqueue(ctx, outbox.TopicMediaPurge, media.PurgeObjectsPayload{
		UserID: userID,
		Keys:   []string{boomKey, "published/" + userID + "/ok.jpg"},
	}); err != nil {
		t.Fatalf("enqueue purge event: %v", err)
	}

	dispatcher := outbox.NewDispatcher(factory, nil)
	dispatcher.RegisterExternal(outbox.TopicMediaPurge, mediaPurgeTopicHandler(svc))
	if _, err := dispatcher.DrainOnce(ctx); err != nil {
		t.Fatalf("drain: %v", err)
	}
	dispatcher.Wait()

	// 单条失败不中断其余删除：另一个对象仍被删。
	if got := provider.deletedKeys(); !slices.Contains(got, "published/"+userID+"/ok.jpg") {
		t.Fatalf("单条失败不得中断其余删除，deleted = %v", got)
	}
	var attempts int
	var processedAt, availableAt *time.Time
	var lastError *string
	if err := qdb.QueryRow(ctx,
		"SELECT attempts, processed_at, available_at, last_error FROM outbox_events WHERE topic = $1 AND payload->>'user_id' = $2",
		outbox.TopicMediaPurge, userID).Scan(&attempts, &processedAt, &availableAt, &lastError); err != nil {
		t.Fatalf("read purge event row: %v", err)
	}
	if attempts != 1 || processedAt != nil {
		t.Fatalf("失败后事件行 = attempts:%d processed:%v, want 1/nil", attempts, processedAt)
	}
	if availableAt == nil || !availableAt.After(time.Now()) {
		t.Fatalf("失败后 available_at = %v, 应退避到未来", availableAt)
	}
	if lastError == nil || !strings.Contains(*lastError, "s3 unavailable") {
		t.Fatalf("last_error = %v, 应包含删除错误", lastError)
	}
}
