package users

// ws_payloads_spec_test.go — users 模块 WS payload 形状与搜索入参校验契约
//（spec api_spec_websocket.md / api_spec_base.md「搜索与地理约束」）。

import (
	"context"
	"encoding/json"
	"testing"

	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/realtime"
	"hotspot/backend-go/internal/platform/types"
)

func assertUserPayloadKeys(t *testing.T, payload any, wantKeys ...string) map[string]json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatalf("payload is not a JSON object: %v (%s)", err, raw)
	}
	if len(fields) != len(wantKeys) {
		keys := make([]string, 0, len(fields))
		for key := range fields {
			keys = append(keys, key)
		}
		t.Fatalf("payload keys = %v, want %v (raw: %s)", keys, wantKeys, raw)
	}
	for _, key := range wantKeys {
		if _, ok := fields[key]; !ok {
			t.Fatalf("payload missing key %q (raw: %s)", key, raw)
		}
	}
	return fields
}

// TestUserProfileUpdatedPayloadCarriesUpdatedUser：spec 要求事件携带「更新后的
// UserInfo」而不仅是 user_id（契约审计 B01）。
func TestUserProfileUpdatedPayloadCarriesUpdatedUser(t *testing.T) {
	hub := realtime.NewHub()
	realtime.SetDefaultHub(hub)
	repo := NewInMemoryRepository()
	if err := repo.PutProfile(context.Background(), defaultTestProfile("u1")); err != nil {
		t.Fatalf("put profile: %v", err)
	}
	service := NewServiceWithRepo(repo)
	sink := &testRealtimeSink{}
	hub.RegisterUser("u1", "sess_u1", sink)
	ctx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "u1", SessionID: "sess_u1"})
	nickname := "鹿鸣改名"
	if err := service.UpdateMyProfile(ctx, UpdateAccountRequest{Nickname: &nickname}); err != nil {
		t.Fatalf("update profile: %v", err)
	}
	if len(sink.events) != 1 || sink.events[0].Type != realtime.EventTypeUserProfileUpdated {
		t.Fatalf("unexpected events: %#v", sink.events)
	}
	payload, ok := sink.events[0].Payload.(UserProfileUpdatedPayload)
	if !ok {
		t.Fatalf("unexpected payload type: %T", sink.events[0].Payload)
	}
	if payload.User.UserID != "u1" || payload.User.Nickname != nickname {
		t.Fatalf("payload must carry updated UserInfo: %#v", payload)
	}
	if payload.UpdatedAt == "" {
		t.Fatal("updated_at must not be empty")
	}
	fields := assertUserPayloadKeys(t, payload, "user", "updated_at")
	var userFields map[string]json.RawMessage
	if err := json.Unmarshal(fields["user"], &userFields); err != nil {
		t.Fatalf("user is not an object: %v (%s)", err, fields["user"])
	}
	for _, key := range []string{"user_id", "nickname", "avatar_url"} {
		if _, ok := userFields[key]; !ok {
			t.Fatalf("user missing key %q: %s", key, fields["user"])
		}
	}
}

// TestNewFollowerPayloadJSONShape：source/source_moment_id 恒定 nullable。
func TestNewFollowerPayloadJSONShape(t *testing.T) {
	fields := assertUserPayloadKeys(t, NewFollowerPayload{
		NotifyID:   "notif_1",
		Actor:      types.UserInfo{UserID: "u2", Nickname: "关注者"},
		FollowedAt: "2026-11-28T00:00:00Z",
	}, "notify_id", "actor", "followed_at", "source", "source_moment_id")
	if string(fields["source"]) != "null" || string(fields["source_moment_id"]) != "null" {
		t.Fatalf("source fields must be null when absent: %s / %s", fields["source"], fields["source_moment_id"])
	}
}

func userSearchPtr(value int) *int { return &value }

// TestValidateSearchUsersRequestRadiusBounds：radius 100..50000 米，越界返回
// 10000（与 groups/search 同口径；此前静默裁剪，契约审计 B11）。
func TestValidateSearchUsersRequestRadiusBounds(t *testing.T) {
	cases := []struct {
		name   string
		radius int
		wantOK bool
	}{
		{"below min", 99, false},
		{"min", 100, true},
		{"max", 50000, true},
		{"above max", 50001, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := SearchUsersRequest{Keyword: "鹿", Radius: userSearchPtr(tc.radius)}
			msg := validateSearchUsersRequest(req)
			if tc.wantOK && msg != "" {
				t.Fatalf("radius %d should be accepted, got %q", tc.radius, msg)
			}
			if !tc.wantOK && msg == "" {
				t.Fatalf("radius %d should be rejected", tc.radius)
			}
		})
	}
}

// TestValidateSearchUsersRequestSortByEnum：sort_by ∈ relevance|distance|newest，
// 非法枚举返回 10000（此前被静默当作 relevance，契约审计 B11）。
func TestValidateSearchUsersRequestSortByEnum(t *testing.T) {
	cases := []struct {
		name   string
		sortBy string
		wantOK bool
	}{
		{"empty", "", true},
		{"relevance", "relevance", true},
		{"distance", "distance", true},
		{"newest", "newest", true},
		{"popular", "popular", false},
		{"spaces trimmed", "  newest ", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := SearchUsersRequest{Keyword: "鹿", SortBy: tc.sortBy}
			msg := validateSearchUsersRequest(req)
			if tc.wantOK && msg != "" {
				t.Fatalf("sort_by %q should be accepted, got %q", tc.sortBy, msg)
			}
			if !tc.wantOK && msg == "" {
				t.Fatalf("sort_by %q should be rejected", tc.sortBy)
			}
		})
	}
}
