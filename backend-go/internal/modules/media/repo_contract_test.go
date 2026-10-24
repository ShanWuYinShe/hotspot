package media

import (
	"context"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/testdb"
)

// 媒体仓库契约：上传会话状态机（pending → completed）、完成态解析/元数据、
// 保留期 GC（只清理未完成、按 created_at 升序）必须双实现一致。
// media_upload_sessions.user_id 无外键，PG 侧无需预置用户。
type mediaContractState struct {
	ctx   context.Context
	userA string
	userB string
}

func TestMediaRepositoryContract(t *testing.T) {
	userA := "it_media_contract_a"
	userB := "it_media_contract_b"
	base := time.Now().UTC().Truncate(time.Second)

	impls := []testdb.ContractImpl[Repository, mediaContractState]{
		{
			Name: "memory",
			New: func(t *testing.T) (Repository, mediaContractState, func()) {
				return NewInMemoryRepository(), mediaContractState{ctx: context.Background(), userA: userA, userB: userB}, nil
			},
		},
		{
			Name: "postgres",
			New: func(t *testing.T) (Repository, mediaContractState, func()) {
				testdb.SkipIfShort(t)
				testdb.Open(t)
				txFactory, cleanup := testdb.BeginScope(t)
				return NewPostgresRepository(txFactory), mediaContractState{ctx: context.Background(), userA: userA, userB: userB}, cleanup
			},
		},
	}

	pending := func(uploadID, userID string, category MediaCategory, createdAt time.Time) pendingUpload {
		return pendingUpload{
			UploadID:         uploadID,
			UserID:           userID,
			StorageKey:       "uploads/" + uploadID,
			FileName:         uploadID + ".png",
			DeclaredFileSize: 1024,
			DeclaredMimeType: "image/png",
			Category:         category,
			MD5:              "d41d8cd98f00b204e9800998ecf8427e",
			CreatedAt:        createdAt.UTC().Format(time.RFC3339),
		}
	}

	file := func(id string) UploadedMedia {
		return UploadedMedia{
			FileID:       id,
			ThumbnailURL: "https://cdn.example.com/" + id + "_thumb.webp",
			FileSize:     2048,
			MimeType:     "image/webp",
			Duration:     0,
			Width:        1080,
			Height:       1920,
		}
	}

	saveComplete := func(t *testing.T, repo Repository, state mediaContractState, uploadID string, category MediaCategory) pendingUpload {
		t.Helper()
		upload := pending(uploadID, state.userA, category, base)
		// file_id 与 upload_id 同值：内存实现的 CompletedFileMetaByID 以 upload_id 作键。
		upload.File = file(uploadID)
		if err := repo.Save(state.ctx, upload); err != nil {
			t.Fatalf("save %s: %v", uploadID, err)
		}
		completed, err := repo.Complete(state.ctx, uploadID, upload.File, "idem-"+uploadID)
		if err != nil || !completed {
			t.Fatalf("complete %s = %v err=%v", uploadID, completed, err)
		}
		return upload
	}

	testdb.RunContract(t, impls, []testdb.ContractCase[Repository, mediaContractState]{
		{
			Name: "save and get pending upload round-trips fields",
			Run: func(t *testing.T, repo Repository, state mediaContractState) {
				upload := pending("upload_roundtrip", state.userA, CategoryAvatar, base)
				if err := repo.Save(state.ctx, upload); err != nil {
					t.Fatalf("save: %v", err)
				}
				got, found, err := repo.Get(state.ctx, "upload_roundtrip")
				if err != nil || !found {
					t.Fatalf("get: found=%v err=%v", found, err)
				}
				if got.UserID != state.userA || got.StorageKey != upload.StorageKey || got.FileName != upload.FileName {
					t.Fatalf("upload = %#v", got)
				}
				if got.DeclaredFileSize != 1024 || got.DeclaredMimeType != "image/png" || got.Category != CategoryAvatar || got.MD5 != upload.MD5 {
					t.Fatalf("declared fields = %#v", got)
				}
				if got.Completed || got.CompletedAt != "" || got.CompletedIdempotencyKey != "" {
					t.Fatalf("pending session marked completed: %#v", got)
				}
				if _, found, err := repo.Get(state.ctx, "upload_missing"); err != nil || found {
					t.Fatalf("get missing: found=%v err=%v", found, err)
				}
			},
		},
		{
			Name: "complete transitions once and stores final metadata",
			Run: func(t *testing.T, repo Repository, state mediaContractState) {
				upload := pending("upload_complete", state.userA, CategoryMomentImage, base)
				if err := repo.Save(state.ctx, upload); err != nil {
					t.Fatalf("save: %v", err)
				}
				final := file("file_complete")
				if updated, err := repo.Complete(state.ctx, "upload_complete", final, "idem-key-1"); err != nil || !updated {
					t.Fatalf("complete = %v err=%v", updated, err)
				}
				got, found, err := repo.Get(state.ctx, "upload_complete")
				if err != nil || !found {
					t.Fatalf("get completed: found=%v err=%v", found, err)
				}
				if !got.Completed || got.File.FileID != "file_complete" || got.File.Width != 1080 || got.File.Height != 1920 {
					t.Fatalf("completed upload = %#v", got)
				}
				if got.CompletedIdempotencyKey != "idem-key-1" || got.CompletedAt == "" {
					t.Fatalf("completion audit fields = %#v", got)
				}
				if _, err := time.Parse(time.RFC3339, got.CompletedAt); err != nil {
					t.Fatalf("completed_at must be RFC3339: %q", got.CompletedAt)
				}
				// 已完成会话不可被再次改写（并发完成只有一个生效）。
				if updated, err := repo.Complete(state.ctx, "upload_complete", file("file_other"), "idem-key-2"); err != nil || updated {
					t.Fatalf("second complete = %v err=%v", updated, err)
				}
				again, _, err := repo.Get(state.ctx, "upload_complete")
				if err != nil {
					t.Fatalf("get after second complete: %v", err)
				}
				if again.File.FileID != "file_complete" || again.CompletedIdempotencyKey != "idem-key-1" {
					t.Fatalf("completion overwritten: %#v", again)
				}
				if updated, err := repo.Complete(state.ctx, "upload_missing", final, "idem"); err != nil || updated {
					t.Fatalf("complete missing = %v err=%v", updated, err)
				}
			},
		},
		{
			Name: "resolve completed files is scoped to owner and category",
			Run: func(t *testing.T, repo Repository, state mediaContractState) {
				saveComplete(t, repo, state, "upload_avatar_1", CategoryAvatar)
				saveComplete(t, repo, state, "upload_avatar_2", CategoryAvatar)
				saveComplete(t, repo, state, "upload_moment_1", CategoryMomentImage)
				pendingOnly := pending("upload_pending", state.userA, CategoryAvatar, base)
				if err := repo.Save(state.ctx, pendingOnly); err != nil {
					t.Fatalf("save pending: %v", err)
				}
				otherOwner := pending("upload_other_owner", state.userB, CategoryAvatar, base)
				otherOwner.File = file("file_other_owner")
				if err := repo.Save(state.ctx, otherOwner); err != nil {
					t.Fatalf("save other owner: %v", err)
				}
				if completed, err := repo.Complete(state.ctx, "upload_other_owner", otherOwner.File, "idem-other"); err != nil || !completed {
					t.Fatalf("complete other owner: %v err=%v", completed, err)
				}

				assets, err := repo.ResolveByFileIDs(state.ctx,
					[]string{"upload_avatar_1", "upload_avatar_2", "upload_moment_1", "upload_pending", "file_other_owner", "file_missing"},
					state.userA, CategoryAvatar)
				if err != nil {
					t.Fatalf("resolve: %v", err)
				}
				if len(assets) != 2 {
					t.Fatalf("assets = %#v, want 2", assets)
				}
				for _, id := range []string{"upload_avatar_1", "upload_avatar_2"} {
					asset, ok := assets[id]
					if !ok {
						t.Fatalf("asset %s missing: %#v", id, assets)
					}
					if asset.StorageKey != "uploads/"+id || asset.Width != 1080 || asset.Height != 1920 || asset.MimeType != "image/webp" {
						t.Fatalf("asset %s = %#v", id, asset)
					}
				}
				otherAssets, err := repo.ResolveByFileIDs(state.ctx, []string{"upload_avatar_1"}, state.userB, CategoryAvatar)
				if err != nil || len(otherAssets) != 0 {
					t.Fatalf("other owner assets = %#v err=%v", otherAssets, err)
				}
				emptyAssets, err := repo.ResolveByFileIDs(state.ctx, nil, state.userA, CategoryAvatar)
				if err != nil || len(emptyAssets) != 0 {
					t.Fatalf("empty ids assets = %#v err=%v", emptyAssets, err)
				}
			},
		},
		{
			Name: "completed file meta is readable without owner scoping",
			Run: func(t *testing.T, repo Repository, state mediaContractState) {
				upload := pending("upload_meta", state.userA, CategoryMomentImage, base)
				upload.FileName = "report-target.png"
				upload.DeclaredFileSize = 4096
				upload.File = UploadedMedia{
					FileID:       "file_meta",
					FileSize:     4096,
					MimeType:     "image/png",
					ThumbnailURL: "https://cdn.example.com/file_meta_thumb.webp",
					Width:        720,
					Height:       1280,
					Duration:     0,
				}
				if err := repo.Save(state.ctx, upload); err != nil {
					t.Fatalf("save: %v", err)
				}
				if meta, found, err := repo.CompletedFileMetaByID(state.ctx, "file_meta"); err != nil || found {
					t.Fatalf("meta before complete = %#v found=%v err=%v", meta, found, err)
				}
				if completed, err := repo.Complete(state.ctx, "upload_meta", upload.File, "idem-meta"); err != nil || !completed {
					t.Fatalf("complete: %v err=%v", completed, err)
				}
				meta, found, err := repo.CompletedFileMetaByID(state.ctx, "file_meta")
				if err != nil || !found {
					t.Fatalf("meta: found=%v err=%v", found, err)
				}
				if meta.Category != CategoryMomentImage || meta.FileName != "report-target.png" || meta.FileSize != 4096 {
					t.Fatalf("meta = %#v", meta)
				}
				if meta.MimeType != "image/png" || meta.Width != 720 || meta.Height != 1280 || meta.ThumbnailURL == "" {
					t.Fatalf("meta media fields = %#v", meta)
				}
				if _, found, err := repo.CompletedFileMetaByID(state.ctx, "file_missing"); err != nil || found {
					t.Fatalf("meta missing: found=%v err=%v", found, err)
				}
			},
		},
		{
			Name: "stale upload GC lists oldest pending sessions first",
			Run: func(t *testing.T, repo Repository, state mediaContractState) {
				cutoff := base.Add(-30 * time.Minute)
				for _, step := range []struct {
					id        string
					createdAt time.Time
				}{
					{"upload_stale_3h", base.Add(-3 * time.Hour)},
					{"upload_stale_2h", base.Add(-2 * time.Hour)},
					{"upload_stale_1h", base.Add(-time.Hour)},
				} {
					if err := repo.Save(state.ctx, pending(step.id, state.userA, CategoryAvatar, step.createdAt)); err != nil {
						t.Fatalf("save %s: %v", step.id, err)
					}
				}
				completed := pending("upload_stale_completed", state.userA, CategoryAvatar, base.Add(-4*time.Hour))
				completed.File = file("upload_stale_completed")
				if err := repo.Save(state.ctx, completed); err != nil {
					t.Fatalf("save completed: %v", err)
				}
				if ok, err := repo.Complete(state.ctx, "upload_stale_completed", completed.File, "idem-stale"); err != nil || !ok {
					t.Fatalf("complete: %v err=%v", ok, err)
				}

				stale, err := repo.ListStaleUploads(state.ctx, cutoff, 2)
				if err != nil {
					t.Fatalf("list stale: %v", err)
				}
				if len(stale) != 2 || stale[0].UploadID != "upload_stale_3h" || stale[1].UploadID != "upload_stale_2h" {
					t.Fatalf("stale = %#v", staleIDs(stale))
				}
				if stale[0].StorageKey != "uploads/upload_stale_3h" || stale[0].UserID != state.userA {
					t.Fatalf("stale fields = %#v", stale[0])
				}
				if err := repo.DeleteUploadRow(state.ctx, "upload_stale_3h"); err != nil {
					t.Fatalf("delete stale: %v", err)
				}
				if _, found, err := repo.Get(state.ctx, "upload_stale_3h"); err != nil || found {
					t.Fatalf("deleted stale still present: found=%v err=%v", found, err)
				}
				remaining, err := repo.ListStaleUploads(state.ctx, cutoff, 10)
				if err != nil {
					t.Fatalf("list stale after delete: %v", err)
				}
				if len(remaining) != 2 || remaining[0].UploadID != "upload_stale_2h" || remaining[1].UploadID != "upload_stale_1h" {
					t.Fatalf("remaining stale = %#v", staleIDs(remaining))
				}
				// GC 不得删除已完成会话（保留期 GC 只清理未完成上传）。
				if err := repo.DeleteUploadRow(state.ctx, "upload_stale_completed"); err != nil {
					t.Fatalf("delete completed: %v", err)
				}
				got, found, err := repo.Get(state.ctx, "upload_stale_completed")
				if err != nil || !found || !got.Completed {
					t.Fatalf("completed row must survive GC: found=%v err=%v record=%#v", found, err, got)
				}
			},
		},
		{
			Name: "save upserts basic session fields",
			Run: func(t *testing.T, repo Repository, state mediaContractState) {
				first := pending("upload_upsert", state.userA, CategoryAvatar, base)
				if err := repo.Save(state.ctx, first); err != nil {
					t.Fatalf("save first: %v", err)
				}
				second := pending("upload_upsert", state.userA, CategoryMomentImage, base.Add(-time.Hour))
				second.StorageKey = "uploads/resaved"
				second.FileName = "resaved.jpg"
				second.DeclaredMimeType = "image/jpeg"
				if err := repo.Save(state.ctx, second); err != nil {
					t.Fatalf("save second: %v", err)
				}
				got, found, err := repo.Get(state.ctx, "upload_upsert")
				if err != nil || !found {
					t.Fatalf("get: found=%v err=%v", found, err)
				}
				if got.StorageKey != "uploads/resaved" || got.FileName != "resaved.jpg" || got.Category != CategoryMomentImage {
					t.Fatalf("upserted record = %#v", got)
				}
				if got.DeclaredMimeType != "image/jpeg" {
					t.Fatalf("declared_mime_type = %q", got.DeclaredMimeType)
				}
			},
		},
	})
}

func staleIDs(items []pendingUpload) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, item.UploadID)
	}
	return out
}
