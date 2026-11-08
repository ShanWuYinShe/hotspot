package media

import (
	"context"
	"errors"
	"testing"
	"time"

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

func TestPostgresRepositoryPurgeUserDataWithoutDatabase(t *testing.T) {
	if err := NewPostgresRepository(nil).PurgeUserData(context.Background(), "user_mediacov"); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("expected ErrNotImplemented, got %v", err)
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

// 真实库：账户注销只删除目标用户的上传会话行（范围外行不动），且幂等。
func TestPostgresRepositoryPurgeUserDataDeletesOnlyTargetUser(t *testing.T) {
	testdb.SkipIfShort(t)
	testdb.Open(t)
	factory, cleanup := testdb.BeginScope(t)
	defer cleanup()

	repo := NewPostgresRepository(factory)
	ctx := context.Background()
	const targetUser = "mediacov_purge_user"
	const otherUser = "mediacov_purge_other"

	seed := func(t *testing.T, uploadID, userID string) {
		t.Helper()
		if err := repo.Save(ctx, pendingUpload{
			UploadID:         uploadID,
			UserID:           userID,
			StorageKey:       "uploads/" + userID + "/" + uploadID + ".png",
			FileName:         uploadID + ".png",
			DeclaredFileSize: 100,
			DeclaredMimeType: "image/png",
			Category:         CategoryAvatar,
			CreatedAt:        time.Now().UTC().Format(time.RFC3339),
		}); err != nil {
			t.Fatalf("save %s: %v", uploadID, err)
		}
	}
	seed(t, "mediacov_purge_upload_a", targetUser)
	seed(t, "mediacov_purge_upload_b", targetUser)
	seed(t, "mediacov_purge_upload_other", otherUser)

	if err := repo.PurgeUserData(ctx, targetUser); err != nil {
		t.Fatalf("purge user data: %v", err)
	}
	for _, uploadID := range []string{"mediacov_purge_upload_a", "mediacov_purge_upload_b"} {
		if _, found, err := repo.Get(ctx, uploadID); err != nil || found {
			t.Fatalf("upload %s must be purged: found=%v err=%v", uploadID, found, err)
		}
	}
	if _, found, err := repo.Get(ctx, "mediacov_purge_upload_other"); err != nil || !found {
		t.Fatalf("other user's upload must survive: found=%v err=%v", found, err)
	}
	// 幂等：重复调用不报错。
	if err := repo.PurgeUserData(ctx, targetUser); err != nil {
		t.Fatalf("second purge must be idempotent, got %v", err)
	}

	// Service 层在真实仓储上同样转发到 Postgres 实现。
	svc := NewServiceWithRepoAndProvider(repo, &coverageProvider{})
	if err := svc.PurgeUserData(ctx, otherUser); err != nil {
		t.Fatalf("service purge: %v", err)
	}
	if _, found, err := repo.Get(ctx, "mediacov_purge_upload_other"); err != nil || found {
		t.Fatalf("service purge must delete other user's rows: found=%v err=%v", found, err)
	}
}
