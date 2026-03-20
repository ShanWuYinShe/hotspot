package integration

import (
	"os"
	"testing"
)

func requireIntegrationEnv(t *testing.T) {
	t.Helper()
	if os.Getenv("POSTGRES_DSN") == "" {
		t.Skip("POSTGRES_DSN not set")
	}
}

func TestHarnessSkipsWithoutPostgres(t *testing.T) {
	if os.Getenv("POSTGRES_DSN") != "" {
		t.Skip("integration env available")
	}
	requireIntegrationEnv(t)
}
