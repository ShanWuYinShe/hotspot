package media

import (
	"errors"
	"testing"

	"hotspot/backend-go/internal/platform/config"
)

func TestNewModuleRequiresS3Configuration(t *testing.T) {
	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatal("expected panic when S3 configuration is missing")
		}
		var err error
		switch v := recovered.(type) {
		case error:
			err = v
		case string:
			err = errors.New(v)
		default:
			t.Fatalf("unexpected panic type: %T", recovered)
		}
		if err.Error() != "media object storage not configured" {
			t.Fatalf("unexpected panic error: %v", err)
		}
	}()

	_ = NewModule(config.Config{}, nil)
}

func TestNewModuleAcceptsS3CompatibleConfiguration(t *testing.T) {
	module := NewModule(config.Config{
		S3Endpoint:       "http://minio:9000",
		S3Bucket:         "hotspot",
		S3Region:         "us-east-1",
		S3AccessKey:      "minioadmin",
		S3SecretKey:      "minioadmin123",
		S3ForcePathStyle: true,
	}, nil)
	if module == nil || module.handler == nil {
		t.Fatalf("expected module with handler, got %#v", module)
	}
}
