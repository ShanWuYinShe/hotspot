package media

import (
	"bytes"
	"context"
	"errors"
	"testing"
)

// copy-on-complete 端到端（服务层）：complete 必须把对象发布到不可写发布键
// （published/<user>/<uploadID>.<ext>），把 media_upload_sessions.storage_key
// 改写为该键，并尽力删除原可写上传键。上传键的预签名 PUT URL 在 TTL 内仍有效，
// 发布键从不签发给客户端，因此发布后内容不可变（第 81 轮审计修复的核心不变量）。
func saveCopyOnCompleteUpload(t *testing.T, repo Repository, upload pendingUpload) {
	t.Helper()
	if err := repo.Save(context.Background(), upload); err != nil {
		t.Fatalf("save %s: %v", upload.UploadID, err)
	}
}

func TestCompleteNonImageCopiesToPublishedKey(t *testing.T) {
	repo := NewInMemoryRepository()
	provider := &mediaStubProvider{headResp: ObjectMetadata{FileSize: 2048, MimeType: "application/pdf"}}
	svc := NewServiceWithRepoAndProvider(repo, provider)
	const uploadKey = "uploads/u1/upload_pub_file.pdf"
	saveCopyOnCompleteUpload(t, repo, pendingUpload{
		UploadID: "upload_pub_file", UserID: "u1", StorageKey: uploadKey,
		FileName: "doc.pdf", DeclaredFileSize: 2048, DeclaredMimeType: "application/pdf",
		Category: CategoryChatFile,
	})

	file, err := svc.Complete(mediaContextWithUserAndKey("u1", "key1"), "upload_pub_file")
	if err != nil {
		t.Fatalf("Complete returned error: %v", err)
	}
	const wantPublished = "published/u1/upload_pub_file.pdf"
	if len(provider.copies) != 1 || provider.copies[0] != [2]string{uploadKey, wantPublished} {
		t.Fatalf("non-image must be copied with CopyObject, copies = %#v", provider.copies)
	}
	if provider.putBody != nil {
		t.Fatal("non-image must not be read into application memory (no PutObject)")
	}
	stored, found, err := repo.Get(context.Background(), "upload_pub_file")
	if err != nil || !found {
		t.Fatalf("get completed upload: found=%v err=%v", found, err)
	}
	if !stored.Completed || stored.StorageKey != wantPublished {
		t.Fatalf("completed row = %#v, want storage_key %q", stored, wantPublished)
	}
	if len(provider.deletedKeys) != 1 || provider.deletedKeys[0] != uploadKey {
		t.Fatalf("upload key must be deleted after commit, deleted = %#v", provider.deletedKeys)
	}
	if file.FileSize != 2048 || file.MimeType != "application/pdf" || file.FileID == "" {
		t.Fatalf("unexpected file metadata: %#v", file)
	}
}

func TestCompleteImageScrubsDirectlyIntoPublishedKey(t *testing.T) {
	data := coveragePNGWithEXIF()
	repo := NewInMemoryRepository()
	provider := &mediaStubProvider{
		headResp: ObjectMetadata{FileSize: len(data), MimeType: "image/png"},
		prefix:   data,
	}
	svc := NewServiceWithRepoAndProvider(repo, provider)
	const uploadKey = "uploads/u1/upload_pub_image.png"
	saveCopyOnCompleteUpload(t, repo, pendingUpload{
		UploadID: "upload_pub_image", UserID: "u1", StorageKey: uploadKey,
		FileName: "photo.png", DeclaredFileSize: len(data), DeclaredMimeType: "image/png",
		Category: CategoryMomentImage,
	})

	if _, err := svc.Complete(mediaContextWithUserAndKey("u1", "key1"), "upload_pub_image"); err != nil {
		t.Fatalf("Complete returned error: %v", err)
	}
	const wantPublished = "published/u1/upload_pub_image.png"
	if provider.putKey != wantPublished {
		t.Fatalf("cleaned bytes must be written directly to published key, put key = %q", provider.putKey)
	}
	if bytes.Contains(provider.putBody, []byte("GPS-LOCATION")) {
		t.Fatalf("published object still contains EXIF GPS payload: %q", provider.putBody)
	}
	if !bytes.Contains(provider.putBody, []byte("imagedata")) {
		t.Fatal("image payload must survive scrubbing")
	}
	if len(provider.copies) != 0 {
		t.Fatalf("must not copy original after scrubbing, copies = %#v", provider.copies)
	}
	stored, found, err := repo.Get(context.Background(), "upload_pub_image")
	if err != nil || !found {
		t.Fatalf("get completed upload: found=%v err=%v", found, err)
	}
	if stored.StorageKey != wantPublished {
		t.Fatalf("completed storage_key = %q, want %q", stored.StorageKey, wantPublished)
	}
	if len(provider.deletedKeys) != 1 || provider.deletedKeys[0] != uploadKey {
		t.Fatalf("upload key must be deleted after commit, deleted = %#v", provider.deletedKeys)
	}
}

func TestCompleteImageWithoutMetadataUsesServerSideCopy(t *testing.T) {
	repo := NewInMemoryRepository()
	provider := &mediaStubProvider{
		headResp: ObjectMetadata{FileSize: 100, MimeType: "image/png"},
		prefix:   magicPNG,
	}
	svc := NewServiceWithRepoAndProvider(repo, provider)
	const uploadKey = "uploads/u1/upload_pub_clean.png"
	saveCopyOnCompleteUpload(t, repo, pendingUpload{
		UploadID: "upload_pub_clean", UserID: "u1", StorageKey: uploadKey,
		FileName: "clean.png", DeclaredFileSize: 100, DeclaredMimeType: "image/png",
		Category: CategoryAvatar,
	})
	if _, err := svc.Complete(mediaContextWithUserAndKey("u1", "key1"), "upload_pub_clean"); err != nil {
		t.Fatalf("Complete returned error: %v", err)
	}
	if len(provider.copies) != 1 || provider.copies[0] != [2]string{uploadKey, "published/u1/upload_pub_clean.png"} {
		t.Fatalf("clean image must be copied server-side, copies = %#v", provider.copies)
	}
	if provider.putBody != nil {
		t.Fatal("clean image must not be rewritten through PutObject")
	}
}

// 复制失败（存储端/网络错误）：complete 不得提交，行保持 pending 且 storage_key
// 仍指向可写上传键，客户端可安全重试。
func TestCompleteCopyFailureDoesNotCommit(t *testing.T) {
	copyErr := errors.New("copy boom")
	repo := NewInMemoryRepository()
	provider := &mediaStubProvider{
		headResp: ObjectMetadata{FileSize: 100, MimeType: "application/pdf"},
		copyErr:  copyErr,
	}
	svc := NewServiceWithRepoAndProvider(repo, provider)
	const uploadKey = "uploads/u1/upload_copyfail.pdf"
	saveCopyOnCompleteUpload(t, repo, pendingUpload{
		UploadID: "upload_copyfail", UserID: "u1", StorageKey: uploadKey,
		FileName: "a.pdf", DeclaredFileSize: 100, DeclaredMimeType: "application/pdf",
		Category: CategoryChatFile,
	})
	if _, err := svc.Complete(mediaContextWithUserAndKey("u1", "key1"), "upload_copyfail"); !errors.Is(err, copyErr) {
		t.Fatalf("expected copy error, got %v", err)
	}
	stored, found, err := repo.Get(context.Background(), "upload_copyfail")
	if err != nil || !found {
		t.Fatalf("get upload: found=%v err=%v", found, err)
	}
	if stored.Completed || stored.StorageKey != uploadKey {
		t.Fatalf("failed copy must not commit, row = %#v", stored)
	}
	if len(provider.deletedKeys) != 0 {
		t.Fatalf("no object may be deleted on failed copy, deleted = %#v", provider.deletedKeys)
	}
}

// 图片发布链路（清洗 PUT + 兜底 CopyObject）失败：同样不得提交。
func TestCompleteImagePublishFailureDoesNotCommit(t *testing.T) {
	data := coveragePNGWithEXIF()
	copyErr := errors.New("copy boom")
	repo := NewInMemoryRepository()
	provider := &mediaStubProvider{
		headResp: ObjectMetadata{FileSize: len(data), MimeType: "image/png"},
		prefix:   data,
		putErr:   errors.New("put boom"),
		copyErr:  copyErr,
	}
	svc := NewServiceWithRepoAndProvider(repo, provider)
	saveCopyOnCompleteUpload(t, repo, pendingUpload{
		UploadID: "upload_imgfail", UserID: "u1", StorageKey: "uploads/u1/upload_imgfail.png",
		FileName: "p.png", DeclaredFileSize: len(data), DeclaredMimeType: "image/png",
		Category: CategoryAvatar,
	})
	if _, err := svc.Complete(mediaContextWithUserAndKey("u1", "key1"), "upload_imgfail"); !errors.Is(err, copyErr) {
		t.Fatalf("expected publish fallback error, got %v", err)
	}
	stored, found, err := repo.Get(context.Background(), "upload_imgfail")
	if err != nil || !found || stored.Completed {
		t.Fatalf("failed publish must not commit, found=%v err=%v row=%#v", found, err, stored)
	}
}

// 自然幂等：重复 complete 返回同一 file，且不重复复制/删除对象。
func TestCompletePublishReplayHasNoSideEffects(t *testing.T) {
	repo := NewInMemoryRepository()
	provider := &mediaStubProvider{headResp: ObjectMetadata{FileSize: 100, MimeType: "application/pdf"}}
	svc := NewServiceWithRepoAndProvider(repo, provider)
	saveCopyOnCompleteUpload(t, repo, pendingUpload{
		UploadID: "upload_replay", UserID: "u1", StorageKey: "uploads/u1/upload_replay.pdf",
		FileName: "a.pdf", DeclaredFileSize: 100, DeclaredMimeType: "application/pdf",
		Category: CategoryChatFile,
	})
	first, err := svc.Complete(mediaContextWithUserAndKey("u1", "key1"), "upload_replay")
	if err != nil {
		t.Fatalf("first Complete: %v", err)
	}
	copiesAfterFirst := len(provider.copies)
	deletesAfterFirst := len(provider.deletedKeys)
	second, err := svc.Complete(mediaContextWithUserAndKey("u1", "key2"), "upload_replay")
	if err != nil || second.FileID != first.FileID {
		t.Fatalf("replay = %#v err=%v", second, err)
	}
	if len(provider.copies) != copiesAfterFirst || len(provider.deletedKeys) != deletesAfterFirst {
		t.Fatalf("replay must not touch objects: copies %d->%d deletes %d->%d",
			copiesAfterFirst, len(provider.copies), deletesAfterFirst, len(provider.deletedKeys))
	}
}

// 并发完成（Prepare 与 Commit 之间另一请求先落库）：
// - 同发布键：本次复制出的对象已被完成行引用，不得删除；
// - 不同发布键：本次复制出的对象无人引用，必须回滚删除，并返回既有 file 保持幂等。
func TestCompleteConcurrentCompletionRollsBackOnlyUnreferencedKey(t *testing.T) {
	t.Run("same published key is kept", func(t *testing.T) {
		repo := NewInMemoryRepository()
		provider := &mediaStubProvider{headResp: ObjectMetadata{FileSize: 100, MimeType: "application/pdf"}}
		svc := NewServiceWithRepoAndProvider(repo, provider)
		saveCopyOnCompleteUpload(t, repo, pendingUpload{
			UploadID: "upload_race_same", UserID: "u1", StorageKey: "uploads/u1/upload_race_same.pdf",
			FileName: "a.pdf", DeclaredFileSize: 100, DeclaredMimeType: "application/pdf",
			Category: CategoryChatFile,
		})
		ctx := mediaContextWithUserAndKey("u1", "key1")
		prepared, err := svc.PrepareComplete(ctx, "upload_race_same")
		if err != nil {
			t.Fatalf("PrepareComplete: %v", err)
		}
		// 模拟并发请求先完成：与本次相同的确定性发布键。
		if ok, err := repo.Complete(context.Background(), "upload_race_same", prepared.file, prepared.publishedKey, "concurrent-key"); err != nil || !ok {
			t.Fatalf("concurrent complete: ok=%v err=%v", ok, err)
		}
		got, err := svc.CommitComplete(ctx, "upload_race_same", prepared)
		if err != nil || got.FileID != prepared.file.FileID {
			t.Fatalf("replay after concurrent completion = %#v err=%v", got, err)
		}
		for _, key := range provider.deletedKeys {
			if key == prepared.publishedKey {
				t.Fatalf("published key referenced by completed row must not be deleted: %#v", provider.deletedKeys)
			}
		}
	})

	t.Run("different published key is rolled back", func(t *testing.T) {
		repo := NewInMemoryRepository()
		provider := &mediaStubProvider{headResp: ObjectMetadata{FileSize: 100, MimeType: "application/pdf"}}
		svc := NewServiceWithRepoAndProvider(repo, provider)
		saveCopyOnCompleteUpload(t, repo, pendingUpload{
			UploadID: "upload_race_diff", UserID: "u1", StorageKey: "uploads/u1/upload_race_diff.pdf",
			FileName: "a.pdf", DeclaredFileSize: 100, DeclaredMimeType: "application/pdf",
			Category: CategoryChatFile,
		})
		ctx := mediaContextWithUserAndKey("u1", "key1")
		prepared, err := svc.PrepareComplete(ctx, "upload_race_diff")
		if err != nil {
			t.Fatalf("PrepareComplete: %v", err)
		}
		// 模拟并发请求用另一发布键先完成（历史/异常版本产生的键不同）。
		if ok, err := repo.Complete(context.Background(), "upload_race_diff", UploadedMedia{FileID: "file_winner", FileSize: 1, MimeType: "application/pdf"}, "published/u1/upload_race_diff_legacy.pdf", "concurrent-key"); err != nil || !ok {
			t.Fatalf("concurrent complete: ok=%v err=%v", ok, err)
		}
		got, err := svc.CommitComplete(ctx, "upload_race_diff", prepared)
		if err != nil || got.FileID != "file_winner" {
			t.Fatalf("idempotent replay must return winning file, got %#v err=%v", got, err)
		}
		if len(provider.deletedKeys) != 1 || provider.deletedKeys[0] != prepared.publishedKey {
			t.Fatalf("unreferenced published key must be rolled back, deleted = %#v", provider.deletedKeys)
		}
	})
}

// 源对象在 complete 与 copy 之间消失（保留期 GC 等）：按未就绪返回，可重试。
func TestCompleteCopyNotFoundSurfacesNotReady(t *testing.T) {
	repo := NewInMemoryRepository()
	provider := &mediaStubProvider{
		headResp: ObjectMetadata{FileSize: 100, MimeType: "application/pdf"},
		copyErr:  ErrUploadNotReady,
	}
	svc := NewServiceWithRepoAndProvider(repo, provider)
	saveCopyOnCompleteUpload(t, repo, pendingUpload{
		UploadID: "upload_gone", UserID: "u1", StorageKey: "uploads/u1/upload_gone.pdf",
		FileName: "a.pdf", DeclaredFileSize: 100, DeclaredMimeType: "application/pdf",
		Category: CategoryChatFile,
	})
	if _, err := svc.Complete(mediaContextWithUserAndKey("u1", "key1"), "upload_gone"); !errors.Is(err, ErrUploadNotReady) {
		t.Fatalf("vanished source must surface ErrUploadNotReady, got %v", err)
	}
	if stored, found, _ := repo.Get(context.Background(), "upload_gone"); !found || stored.Completed {
		t.Fatalf("row must stay pending, found=%v row=%#v", found, stored)
	}
}
