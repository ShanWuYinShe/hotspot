package media

import (
	"context"
	"errors"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/persistence"
)

// mediaScriptedRows 是可控的 Rows 实现：按脚本吐行，可注入 Scan 错误或迭代
// 结束后的 Err（SQL 语义由真实库契约测试覆盖，这里只覆盖仓库防御分支）。
type mediaScriptedRows struct {
	values   [][]any
	index    int
	scanErr  error
	finalErr error
}

func (r *mediaScriptedRows) Next() bool {
	if r.index >= len(r.values) {
		return false
	}
	r.index++
	return true
}

func (r *mediaScriptedRows) Scan(dest ...any) error {
	if r.scanErr != nil {
		return r.scanErr
	}
	if r.index == 0 || r.index > len(r.values) {
		return errors.New("mediaScriptedRows: scan without next")
	}
	return mediaRowValues{values: r.values[r.index-1]}.Scan(dest...)
}

func (r *mediaScriptedRows) Close() error { return nil }
func (r *mediaScriptedRows) Err() error   { return r.finalErr }

// mediaScriptedBackend 按查询串注入行/错误的后端替身。
type mediaScriptedBackend struct {
	rowsByQuery map[string]persistence.Rows
	rowsErr     map[string]error
	rowByQuery  map[string]persistence.Row
}

func (b *mediaScriptedBackend) Exec(context.Context, string, ...any) error { return nil }

func (b *mediaScriptedBackend) QueryRow(_ context.Context, query string, _ ...any) persistence.Row {
	if row, ok := b.rowByQuery[query]; ok {
		return row
	}
	return mediaRowValues{err: persistence.ErrNotImplemented}
}

func (b *mediaScriptedBackend) Query(_ context.Context, query string, _ ...any) (persistence.Rows, error) {
	if err, ok := b.rowsErr[query]; ok {
		return nil, err
	}
	if rows, ok := b.rowsByQuery[query]; ok {
		return rows, nil
	}
	return &persistence.InMemoryRows{}, nil
}

func TestPostgresRepositoryWithoutQueryableReturnsNotImplemented(t *testing.T) {
	ctx := context.Background()
	repo := NewPostgresRepository(nil)
	if _, _, err := repo.Get(ctx, "upload_mediacov"); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("Get = %v, want ErrNotImplemented", err)
	}
	if _, err := repo.Complete(ctx, "upload_mediacov", UploadedMedia{FileID: "file_mediacov"}, "published/u1/upload_mediacov.png", ""); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("Complete = %v, want ErrNotImplemented", err)
	}
	if _, err := repo.ResolveByFileIDs(ctx, []string{"file_mediacov"}, "u1", CategoryAvatar); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("ResolveByFileIDs = %v, want ErrNotImplemented", err)
	}
	if _, _, err := repo.CompletedFileMetaByID(ctx, "file_mediacov"); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("CompletedFileMetaByID = %v, want ErrNotImplemented", err)
	}
	if _, err := repo.ListStaleUploads(ctx, time.Now().UTC(), 10); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("ListStaleUploads = %v, want ErrNotImplemented", err)
	}
	if err := repo.DeleteUploadRow(ctx, "upload_mediacov"); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("DeleteUploadRow = %v, want ErrNotImplemented", err)
	}
}

func TestPostgresRepositoryPropagatesScanErrors(t *testing.T) {
	scanErr := errors.New("scan boom")
	backend := &mediaScriptedBackend{rowByQuery: map[string]persistence.Row{
		queryGetUploadSession:      mediaRowValues{err: scanErr},
		queryCompleteUploadSession: mediaRowValues{err: scanErr},
	}}
	repo := NewPostgresRepository(newMediaFactory(backend))
	ctx := context.Background()
	if _, _, err := repo.Get(ctx, "upload_mediacov"); !errors.Is(err, scanErr) {
		t.Fatalf("Get scan error = %v, want %v", err, scanErr)
	}
	if _, err := repo.Complete(ctx, "upload_mediacov", UploadedMedia{FileID: "file_mediacov"}, "published/u1/upload_mediacov.png", "key"); !errors.Is(err, scanErr) {
		t.Fatalf("Complete scan error = %v, want %v", err, scanErr)
	}
	// CompletedFileMetaByID 的查询串是函数内局部常量，兜底行返回非 ErrNoRows 错误。
	if _, _, err := repo.CompletedFileMetaByID(ctx, "file_mediacov"); err == nil {
		t.Fatal("CompletedFileMetaByID must propagate non-ErrNoRows scan errors")
	}
}

func TestPostgresRepositoryResolveByFileIDsPaths(t *testing.T) {
	ctx := context.Background()

	queryErr := errors.New("query boom")
	queryBackend := &mediaScriptedBackend{rowsErr: map[string]error{queryResolveByFileIDs: queryErr}}
	if _, err := NewPostgresRepository(newMediaFactory(queryBackend)).ResolveByFileIDs(ctx, []string{"file_mediacov"}, "u1", CategoryAvatar); !errors.Is(err, queryErr) {
		t.Fatalf("query error = %v, want %v", err, queryErr)
	}

	// 空 file_ids 直接返回空 map（不查库）。
	if assets, err := NewPostgresRepository(newMediaFactory(queryBackend)).ResolveByFileIDs(ctx, nil, "u1", CategoryAvatar); err != nil || len(assets) != 0 {
		t.Fatalf("empty ids = %#v err=%v", assets, err)
	}

	scanErr := errors.New("scan boom")
	scanBackend := &mediaScriptedBackend{rowsByQuery: map[string]persistence.Rows{
		queryResolveByFileIDs: &mediaScriptedRows{
			values:  [][]any{{"file_mediacov", "uploads/u1/file_mediacov.png", "", 10, 20, "image/webp"}},
			scanErr: scanErr,
		},
	}}
	if _, err := NewPostgresRepository(newMediaFactory(scanBackend)).ResolveByFileIDs(ctx, []string{"file_mediacov"}, "u1", CategoryAvatar); !errors.Is(err, scanErr) {
		t.Fatalf("scan error = %v, want %v", err, scanErr)
	}

	rowsErr := errors.New("rows boom")
	errBackend := &mediaScriptedBackend{rowsByQuery: map[string]persistence.Rows{
		queryResolveByFileIDs: &mediaScriptedRows{finalErr: rowsErr},
	}}
	if _, err := NewPostgresRepository(newMediaFactory(errBackend)).ResolveByFileIDs(ctx, []string{"file_mediacov"}, "u1", CategoryAvatar); !errors.Is(err, rowsErr) {
		t.Fatalf("rows error = %v, want %v", err, rowsErr)
	}

	okBackend := &mediaScriptedBackend{rowsByQuery: map[string]persistence.Rows{
		queryResolveByFileIDs: &mediaScriptedRows{values: [][]any{{"file_mediacov", "uploads/u1/file_mediacov.png", "https://cdn.example.com/t.webp", 10, 20, "image/webp"}}},
	}}
	assets, err := NewPostgresRepository(newMediaFactory(okBackend)).ResolveByFileIDs(ctx, []string{"file_mediacov"}, "u1", CategoryAvatar)
	if err != nil || len(assets) != 1 {
		t.Fatalf("resolve assets = %#v err=%v", assets, err)
	}
	asset := assets["file_mediacov"]
	if asset.StorageKey != "uploads/u1/file_mediacov.png" || asset.ThumbnailURL != "https://cdn.example.com/t.webp" || asset.Width != 10 || asset.Height != 20 || asset.MimeType != "image/webp" {
		t.Fatalf("resolved asset = %#v", asset)
	}
}

func TestPostgresRepositoryListStaleUploadsPaths(t *testing.T) {
	ctx := context.Background()
	cutoff := time.Now().UTC()

	queryErr := errors.New("query boom")
	queryBackend := &mediaScriptedBackend{rowsErr: map[string]error{queryListStaleUploads: queryErr}}
	if _, err := NewPostgresRepository(newMediaFactory(queryBackend)).ListStaleUploads(ctx, cutoff, 10); !errors.Is(err, queryErr) {
		t.Fatalf("query error = %v, want %v", err, queryErr)
	}

	scanErr := errors.New("scan boom")
	scanBackend := &mediaScriptedBackend{rowsByQuery: map[string]persistence.Rows{
		queryListStaleUploads: &mediaScriptedRows{
			values:  [][]any{{"upload_mediacov", "uploads/u1/upload_mediacov.png", "u1"}},
			scanErr: scanErr,
		},
	}}
	if _, err := NewPostgresRepository(newMediaFactory(scanBackend)).ListStaleUploads(ctx, cutoff, 10); !errors.Is(err, scanErr) {
		t.Fatalf("scan error = %v, want %v", err, scanErr)
	}

	rowsErr := errors.New("rows boom")
	errBackend := &mediaScriptedBackend{rowsByQuery: map[string]persistence.Rows{
		queryListStaleUploads: &mediaScriptedRows{finalErr: rowsErr},
	}}
	if _, err := NewPostgresRepository(newMediaFactory(errBackend)).ListStaleUploads(ctx, cutoff, 10); !errors.Is(err, rowsErr) {
		t.Fatalf("rows error = %v, want %v", err, rowsErr)
	}

	okBackend := &mediaScriptedBackend{rowsByQuery: map[string]persistence.Rows{
		queryListStaleUploads: &mediaScriptedRows{values: [][]any{{"upload_mediacov", "uploads/u1/upload_mediacov.png", "u1"}}},
	}}
	items, err := NewPostgresRepository(newMediaFactory(okBackend)).ListStaleUploads(ctx, cutoff, 10)
	if err != nil || len(items) != 1 {
		t.Fatalf("stale items = %#v err=%v", items, err)
	}
	if items[0].UploadID != "upload_mediacov" || items[0].StorageKey != "uploads/u1/upload_mediacov.png" || items[0].UserID != "u1" {
		t.Fatalf("stale item = %#v", items[0])
	}
}

func TestStatusFromCompletedAndNullableTime(t *testing.T) {
	if got := statusFromCompleted(true); got != "completed" {
		t.Fatalf("statusFromCompleted(true) = %q", got)
	}
	if got := statusFromCompleted(false); got != "pending" {
		t.Fatalf("statusFromCompleted(false) = %q", got)
	}

	parsed := nullableTime("2026-01-02T03:04:05.123Z")
	stamp, ok := parsed.(time.Time)
	if !ok || !stamp.Equal(time.Date(2026, 1, 2, 3, 4, 5, 123000000, time.UTC)) {
		t.Fatalf("nullableTime valid = %#v", parsed)
	}
	if nullableTime("not-a-time") != nil {
		t.Fatal("nullableTime must drop unparsable timestamps")
	}
	if nullableTime("") != nil {
		t.Fatal("nullableTime must map empty string to nil")
	}
}
