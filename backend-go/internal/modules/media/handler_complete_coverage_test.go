package media

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"

	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/testdb"
)

// 批量 complete 第二阶段（落库）失败属内部错误：Prepare 已通过、Commit 报错
// 时必须返回 10600，而不是把内部错误细节透给客户端。
func TestMediaHandlerCompleteMapsCommitWriteError(t *testing.T) {
	repo := &mediaRepoOverride{InMemoryRepository: NewInMemoryRepository(), completeErr: errors.New("commit boom")}
	if err := repo.Save(context.Background(), pendingUpload{
		UploadID:         "upload_mediacov_commit_fail",
		UserID:           "u1",
		StorageKey:       "uploads/u1/commit.png",
		FileName:         "commit.png",
		DeclaredFileSize: 100,
		DeclaredMimeType: "image/png",
		Category:         CategoryAvatar,
	}); err != nil {
		t.Fatalf("save upload: %v", err)
	}
	handler := NewHandler(NewServiceWithRepoAndProvider(repo, &coverageProvider{
		headResp:   ObjectMetadata{FileSize: 100, MimeType: "image/png"},
		readPrefix: magicPNG,
	}))

	rr := httptest.NewRecorder()
	handler.CompleteUpload(rr, mediaAuthedCompleteRequest("upload_mediacov_commit_fail", ""))
	if code := testdb.DecodeAPICode(t, rr); code != httpx.CodeInternalUnavailable {
		t.Fatalf("expected CodeInternalUnavailable, got %d", code)
	}
}
