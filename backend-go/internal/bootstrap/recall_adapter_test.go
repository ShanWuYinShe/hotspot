package bootstrap

import (
	"context"
	"errors"
	"testing"

	"hotspot/backend-go/internal/modules/groups"
	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/testdb"
)

// TestRecallMessageAsAdminAdapterMapsMissingMessage — 装配层适配器把
// groupmessages.ErrMessageNotFound 翻译为 10300（spec admin recall），
// 而非落到 groups 服务的 10600 兜底。需真实 PG（groupmessages 无内存回退）。
func TestRecallMessageAsAdminAdapterMapsMissingMessage(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	cfg := config.Config{
		S3Endpoint:       "http://seaweedfs:8333",
		S3Bucket:         "hotspot",
		S3Region:         "us-east-1",
		S3AccessKey:      "minioadmin",
		S3SecretKey:      "minioadmin123",
		S3ForcePathStyle: true,
	}
	_ = NewHTTPRouter(BuildModules(cfg, factory))
	if groups.RecallMessageAsAdmin == nil {
		t.Fatal("expected RecallMessageAsAdmin wired")
	}
	err := groups.RecallMessageAsAdmin(context.Background(), "g1", "missing", "admin1")
	var apiErr *httpx.APIError
	if !errors.As(err, &apiErr) || apiErr.Code != httpx.CodeNotFound {
		t.Fatalf("expected 10300, got %v", err)
	}
}
