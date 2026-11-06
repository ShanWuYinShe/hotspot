package bootstrap

import (
	"context"
	"errors"
	"testing"

	"hotspot/backend-go/internal/modules/media"
)

// stubCompletedFileMetaReader 是 reportsMediaSource 依赖的最小能力桩：
// 装配层只依赖窄接口（架构审计 P0-1），媒体服务自身行为由 media 包测试覆盖。
type stubCompletedFileMetaReader struct {
	meta  media.FileMeta
	found bool
	err   error
}

func (s stubCompletedFileMetaReader) CompletedFileMetaByID(context.Context, string) (media.FileMeta, bool, error) {
	return s.meta, s.found, s.err
}

func TestReportsMediaSourceMapsAllFields(t *testing.T) {
	ctx := context.Background()
	fm := media.FileMeta{
		Category:     media.CategoryChatFile,
		StorageKey:   "uploads/u1/evidence.pdf",
		ThumbnailURL: "thumb/evidence.jpg",
		Width:        800,
		Height:       600,
		MimeType:     "application/pdf",
		FileName:     "evidence.pdf",
		FileSize:     2048,
		Duration:     12,
	}
	src := reportsMediaSource{svc: stubCompletedFileMetaReader{meta: fm, found: true}}

	got, found, err := src.CompletedFileMetaByID(ctx, "file_1")
	if err != nil || !found {
		t.Fatalf("expected found=true err=nil, got found=%v err=%v", found, err)
	}
	// FileMeta.Category 不在 reports.MediaFileMeta 中（reports 用不到类别），
	// 其余字段逐一比对：漏拷或错位都会在这里失败（举报证据不完整）。
	if got.StorageKey != fm.StorageKey {
		t.Fatalf("StorageKey mismatch: got %q want %q", got.StorageKey, fm.StorageKey)
	}
	if got.ThumbnailURL != fm.ThumbnailURL {
		t.Fatalf("ThumbnailURL mismatch: got %q want %q", got.ThumbnailURL, fm.ThumbnailURL)
	}
	if got.Width != fm.Width || got.Height != fm.Height {
		t.Fatalf("dimensions mismatch: got %dx%d want %dx%d", got.Width, got.Height, fm.Width, fm.Height)
	}
	if got.MimeType != fm.MimeType {
		t.Fatalf("MimeType mismatch: got %q want %q", got.MimeType, fm.MimeType)
	}
	if got.FileName != fm.FileName {
		t.Fatalf("FileName mismatch: got %q want %q", got.FileName, fm.FileName)
	}
	if got.FileSize != fm.FileSize {
		t.Fatalf("FileSize mismatch: got %d want %d", got.FileSize, fm.FileSize)
	}
	if got.Duration != fm.Duration {
		t.Fatalf("Duration mismatch: got %d want %d", got.Duration, fm.Duration)
	}
}

func TestReportsMediaSourceNotFoundPassesThrough(t *testing.T) {
	ctx := context.Background()
	src := reportsMediaSource{svc: stubCompletedFileMetaReader{found: false}}
	if _, found, err := src.CompletedFileMetaByID(ctx, "file_missing"); err != nil || found {
		t.Fatalf("expected found=false err=nil, got found=%v err=%v", found, err)
	}

	boom := errors.New("boom")
	src = reportsMediaSource{svc: stubCompletedFileMetaReader{err: boom}}
	if _, _, err := src.CompletedFileMetaByID(ctx, "file_err"); !errors.Is(err, boom) {
		t.Fatalf("expected error propagated, got %v", err)
	}
}
