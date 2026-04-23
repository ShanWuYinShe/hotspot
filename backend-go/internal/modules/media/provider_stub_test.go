package media

import (
	"context"
	"testing"
)

func TestStubProviderHeadSupportsCompletedUploadFlow(t *testing.T) {
	provider := NewStubProvider()
	meta, err := provider.Head(context.Background(), "uploads/u1/upload_1.png")
	if err != nil {
		t.Fatalf("Head returned error: %v", err)
	}
	if meta.FileSize <= 0 {
		t.Fatalf("expected positive file size, got %#v", meta)
	}
	if meta.MimeType != "image/png" {
		t.Fatalf("expected image/png mime type, got %#v", meta)
	}
}
