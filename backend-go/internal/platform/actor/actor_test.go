package actor

import (
	"context"
	"testing"
)

type mockProvider struct {
	info Info
	err  error
}

func (m mockProvider) GetActorInfo(_ context.Context, _ string) (Info, error) {
	return m.info, m.err
}

func TestActorInfoAndProviderInterface(t *testing.T) {
	expected := Info{
		UserID:    "u_1001",
		Nickname:  "测试用户",
		AvatarURL: "https://example.com/avatar.jpg",
	}

	var provider Provider = mockProvider{info: expected}
	got, err := provider.GetActorInfo(context.Background(), "u_1001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != expected {
		t.Fatalf("expected %+v, got %+v", expected, got)
	}
}
