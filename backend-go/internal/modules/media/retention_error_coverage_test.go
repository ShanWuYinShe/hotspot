package media

import (
	"context"
	"errors"
	"testing"
	"time"
)

// 保留期 GC 的仓储失败分支：列举失败不得误删对象；行删除失败必须保留行
// （对象已删、行留待下一轮重试），不能静默丢数据。
func TestSweepStaleUploadsHandlesRepositoryFailures(t *testing.T) {
	ctx := context.Background()

	t.Run("list failure stops the sweep", func(t *testing.T) {
		repo := &mediaRepoOverride{InMemoryRepository: NewInMemoryRepository(), listErr: errors.New("list boom")}
		provider := &coverageProvider{}
		NewServiceWithRepoAndProvider(repo, provider).sweepStaleUploads(ctx)
		if len(provider.deletedKeys) != 0 {
			t.Fatalf("no object may be deleted when listing fails, got %v", provider.deletedKeys)
		}
	})

	t.Run("row delete failure keeps row for retry", func(t *testing.T) {
		repo := &mediaRepoOverride{InMemoryRepository: NewInMemoryRepository(), deleteRowErr: errors.New("delete boom")}
		provider := &coverageProvider{}
		seedPendingUpload(t, repo, "upload_mediacov_retention_row", time.Now().UTC().AddDate(0, 0, -(staleUploadRetentionDays+1)))
		NewServiceWithRepoAndProvider(repo, provider).sweepStaleUploads(ctx)

		if len(provider.deletedKeys) != 1 {
			t.Fatalf("expected one object delete attempt, got %v", provider.deletedKeys)
		}
		if _, found, _ := repo.Get(ctx, "upload_mediacov_retention_row"); !found {
			t.Fatal("row must survive when row deletion fails")
		}
	})
}
