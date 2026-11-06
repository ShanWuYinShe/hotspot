package comments

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"hotspot/backend-go/internal/platform/httpx"
)

// spec api_spec_public.md：匿名 CommentItem 不含 reacted_by_me；reactions 换
// PublicReaction，top_replies 递归使用 Public 类型，数组字段恒为数组。
func TestToPublicCommentItemStripsViewerStateAndNormalizesArrays(t *testing.T) {
	item := CommentItem{
		CommentID: "c1",
		User:      UserInfo{UserID: "u1", Nickname: "Alice"},
		Content:   "hello",
		Reactions: []Reaction{{Emoji: "like", Count: 2, ReactedByMe: true}},
		TopReplies: []CommentItem{{
			CommentID: "c2",
			Content:   "reply",
			Reactions: []Reaction{{Emoji: "love", Count: 1, ReactedByMe: true}},
		}},
	}
	pub := toPublicCommentItem(item)
	if pub.Mentions == nil || pub.Reactions == nil || pub.TopReplies == nil {
		t.Fatalf("arrays must normalize to empty slices: %#v", pub)
	}
	if len(pub.Reactions) != 1 || pub.Reactions[0].Emoji != "like" || pub.Reactions[0].Count != 2 {
		t.Fatalf("unexpected public reactions: %#v", pub.Reactions)
	}
	if len(pub.TopReplies) != 1 || pub.TopReplies[0].CommentID != "c2" || len(pub.TopReplies[0].Reactions) != 1 {
		t.Fatalf("top_replies must map recursively: %#v", pub.TopReplies)
	}
	blob, _ := json.Marshal(pub)
	if strings.Contains(string(blob), "reacted_by_me") {
		t.Fatalf("public comment must not expose reacted_by_me: %s", blob)
	}
	page := toPublicPaginatedComments(PaginatedComments{Items: []CommentItem{item}})
	if len(page.Items) != 1 || page.Items[0].CommentID != "c1" {
		t.Fatalf("unexpected public page: %#v", page)
	}
}

func decodePublicCode(t *testing.T, rr *httptest.ResponseRecorder) httpx.Code {
	t.Helper()
	var envelope struct {
		Code httpx.Code `json:"code"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&envelope); err != nil {
		t.Fatalf("decode envelope: %v body=%s", err, rr.Body.String())
	}
	return envelope.Code
}

func TestPublicCommentHandlersMapNotFoundTo10300(t *testing.T) {
	// 默认 stub：moment 不存在 / 评论不存在 → 公开读统一 10300（防枚举口径）。
	handler := newTestHandler(&stubCommentRepo{}, &stubMomentProvider{})
	cases := []struct {
		name string
		call func(w http.ResponseWriter, r *http.Request)
		body string
	}{
		{"fetch", handler.PublicFetch, `{"moment_id":"missing"}`},
		{"replies", handler.PublicFetchReplies, `{"comment_id":"missing"}`},
		{"open", handler.PublicOpenTheDoor, `{"comment_id":"missing"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			tc.call(rr, httptest.NewRequest(http.MethodPost, "/public/v1/comments/"+tc.name, strings.NewReader(tc.body)))
			if got := decodePublicCode(t, rr); got != httpx.CodeNotFound {
				t.Fatalf("expected 10300, got %d body=%s", got, rr.Body.String())
			}
		})
	}
	// 参数缺失统一 10000。
	rr := httptest.NewRecorder()
	handler.PublicFetch(rr, httptest.NewRequest(http.MethodPost, "/public/v1/comments/fetch", strings.NewReader(`{}`)))
	if got := decodePublicCode(t, rr); got != httpx.CodeInvalidRequest {
		t.Fatalf("expected 10000 for missing moment_id, got %d", got)
	}
}
