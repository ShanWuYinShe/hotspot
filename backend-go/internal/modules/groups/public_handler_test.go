package groups

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"hotspot/backend-go/internal/platform/httpx"
)

// spec api_spec_public.md：匿名 GroupItem 不得出现 viewer_state/friends_in_group/
// last_message（查看者状态字段），tags 恒为数组。
func TestToPublicGroupItemStripsViewerFields(t *testing.T) {
	viewerState := &GroupViewerState{Role: "owner"}
	friends := 3
	item := GroupItem{
		Group:          GroupInfo{GroupID: "g1", Name: "Group"},
		ViewerState:    viewerState,
		FriendsInGroup: &friends,
		LastMessage:    &GroupLastMessage{MessageID: "m1"},
	}
	pub := toPublicGroupItem(item)
	if pub.Tags == nil {
		t.Fatal("tags must normalize to empty slice")
	}
	blob, _ := json.Marshal(pub)
	for _, forbidden := range []string{"viewer_state", "friends_in_group", "last_message"} {
		if strings.Contains(string(blob), forbidden) {
			t.Fatalf("public group must not expose %q: %s", forbidden, blob)
		}
	}
	if !strings.Contains(string(blob), "group_id") {
		t.Fatalf("unexpected public group payload: %s", blob)
	}
}

func publicGroupFixture() *stubGroupsRepo {
	return &stubGroupsRepo{
		getResult: GroupItem{Group: GroupInfo{GroupID: "g1", Name: "Group"}, MembersSearchable: true, Tags: []string{"x"}},
		getFound:  true,
		searchResult: PaginatedGroups{Items: []GroupItem{{
			Group:          GroupInfo{GroupID: "g1", Name: "Group"},
			Tags:           []string{"x"},
			Location:       &RelativeLocation{},
			ViewerState:    &GroupViewerState{Role: "owner"},
			FriendsInGroup: func() *int { n := 1; return &n }(),
		}}},
		members:         []GroupMemberItem{{User: UserInfo{UserID: "u1", Nickname: "Alice"}, Role: GroupRoleMember, JoinedAt: "2026-01-01T00:00:00Z"}},
		searchMemResult: []GroupMemberItem{{User: UserInfo{UserID: "u2", Nickname: "Bob"}, Role: GroupRoleMember, JoinedAt: "2026-01-02T00:00:00Z"}},
	}
}

func TestPublicGroupHandlersReturnPublicShape(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(publicGroupFixture()))

	cases := []struct {
		name string
		call func(w http.ResponseWriter, r *http.Request)
		body string
	}{
		{"open", handler.PublicOpenTheDoor, `{"group_id":"g1"}`},
		{"search", handler.PublicSearch, `{"tag":"x"}`},
		{"members", handler.PublicListMembers, `{"group_id":"g1"}`},
		{"member_search", handler.PublicSearchMembers, `{"group_id":"g1","keyword":"ali"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			tc.call(rr, httptest.NewRequest(http.MethodPost, "/public/v1/groups/"+tc.name, strings.NewReader(tc.body)))
			var resp struct {
				Code httpx.Code `json:"code"`
				Data any        `json:"data"`
			}
			if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
				t.Fatalf("decode response: %v body=%s", err, rr.Body.String())
			}
			if resp.Code != 0 {
				t.Fatalf("unexpected code: %d body=%s", resp.Code, rr.Body.String())
			}
			forbidden := []string{"viewer_state", "friends_in_group", "last_message"}
			for _, key := range forbidden {
				if strings.Contains(rr.Body.String(), key) {
					t.Fatalf("public response must not contain %q: %s", key, rr.Body.String())
				}
			}
		})
	}
}

func TestPublicGroupHandlersMapErrors(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&stubGroupsRepo{}))
	// 不存在的群 → 10300（search 用 tag 条件走 repo.Search，不受 getFound 影响）。
	rr := httptest.NewRecorder()
	handler.PublicOpenTheDoor(rr, httptest.NewRequest(http.MethodPost, "/public/v1/groups/open_the_door", strings.NewReader(`{"group_id":"missing"}`)))
	var envelope struct {
		Code httpx.Code `json:"code"`
	}
	_ = json.NewDecoder(rr.Body).Decode(&envelope)
	if envelope.Code != httpx.CodeNotFound {
		t.Fatalf("expected 10300 for missing group, got %d", envelope.Code)
	}

	// 搜索缺少条件 → 10000。
	rr = httptest.NewRecorder()
	handler.PublicSearch(rr, httptest.NewRequest(http.MethodPost, "/public/v1/groups/search", strings.NewReader(`{}`)))
	_ = json.NewDecoder(rr.Body).Decode(&envelope)
	if envelope.Code != httpx.CodeInvalidRequest {
		t.Fatalf("expected 10000 for empty search, got %d", envelope.Code)
	}

	// 成员搜索缺 keyword → 10000。
	rr = httptest.NewRecorder()
	handler.PublicSearchMembers(rr, httptest.NewRequest(http.MethodPost, "/public/v1/groups/members/search", strings.NewReader(`{"group_id":"g1"}`)))
	_ = json.NewDecoder(rr.Body).Decode(&envelope)
	if envelope.Code != httpx.CodeInvalidRequest {
		t.Fatalf("expected 10000 for missing keyword, got %d", envelope.Code)
	}
}
