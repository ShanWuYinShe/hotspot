package moments

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	platformauth "hotspot/backend-go/internal/platform/auth"
	httpx "hotspot/backend-go/internal/platform/httpx"
	idempotency "hotspot/backend-go/internal/platform/idempotency"
)

// spec_align_test.go 验证 moments 对齐新契约（spec/api_spec_moment.md）：list_my → list_created、OwnerMomentItem、is_favorited 恒定、location 恒定 nullable、非作者 10200+not_resource_owner、被屏蔽标签 10400。

func mustReader(s string) *strings.Reader { return strings.NewReader(s) }

func TestSpecListCreatedReturnsOwnerView(t *testing.T) {
	idempotency.SetStore(idempotency.NewInMemoryStore())
	repo := NewInMemoryRepository()
	handler := NewHandler(NewServiceWithRepo(repo))
	ctx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "u1"})
	if _, err := repo.Create(context.Background(), "u1", CreateMomentRequest{Description: "hi", PrivacyLevel: "private", Location: &Location{LocationName: "Here"}}); err != nil {
		t.Fatalf("create: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/list_created", mustReader("{}"))
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()
	handler.ListCreated(rr, req)
	var resp struct {
		Code httpx.Code `json:"code"`
		Data struct {
			Items      []OwnerMomentItem `json:"items"`
			NextCursor string            `json:"next_cursor"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v body=%s", err, rr.Body.String())
	}
	if resp.Code != 0 {
		t.Fatalf("expected success, got %d", resp.Code)
	}
	if len(resp.Data.Items) != 1 {
		t.Fatalf("unexpected items: %#v", resp.Data.Items)
	}
	if resp.Data.Items[0].PrivacyLevel != "private" {
		t.Fatalf("expected owner privacy_level=private, got %#v", resp.Data.Items[0])
	}
	if resp.Data.NextCursor != "" {
		t.Fatalf("expected null next_cursor, got %q", resp.Data.NextCursor)
	}
}

func TestSpecCreateReturnsOwnerView(t *testing.T) {
	idempotency.SetStore(idempotency.NewInMemoryStore())
	handler := NewHandler(NewServiceWithRepo(NewInMemoryRepository()))
	ctx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "u1"})
	body := `{"description":"hello","location":{"point":{"latitude":1,"longitude":2},"location_name":"Somewhere"},"privacy_level":"private"}`
	req := httptest.NewRequest(http.MethodPost, "/create", mustReader(body))
	req.Header.Set("Idempotency-Key", "a1b2c3d4a1b2c3d4a1b2c3d4a1b2c3d4")
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()
	handler.Create(rr, req)
	var resp struct {
		Code httpx.Code      `json:"code"`
		Data OwnerMomentItem `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v body=%s", err, rr.Body.String())
	}
	if resp.Code != 0 {
		t.Fatalf("expected success, got %d %s", resp.Code, rr.Body.String())
	}
	if resp.Data.PrivacyLevel != "private" {
		t.Fatalf("expected create owner privacy_level, got %#v", resp.Data)
	}
}

func TestSpecPublicMomentSerialization(t *testing.T) {
	raw, err := json.Marshal(MomentItem{MomentID: "m1", Description: "hi", PrivacyLevel: "private"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := m["is_favorited"]; !ok {
		t.Fatalf("is_favorited must always be present: %s", raw)
	}
	if m["is_favorited"] != false {
		t.Fatalf("expected is_favorited=false, got %v", m["is_favorited"])
	}
	// recommendation 恒定存在（spec api_spec_types.md MomentItem：Feed 场景返回推荐
	// 理由，非 Feed 场景为 null；Feed 域已废除，当前恒 null）。
	rec, ok := m["recommendation"]
	if !ok {
		t.Fatalf("recommendation must always be present: %s", raw)
	}
	if rec != nil {
		t.Fatalf("expected recommendation=null outside feed scenarios, got %v", rec)
	}
	loc, ok := m["location"]
	if !ok {
		t.Fatalf("location must always be present: %s", raw)
	}
	if loc != nil {
		t.Fatalf("expected location=null for nil pointer, got %v", loc)
	}
	if _, ok := m["privacy_level"]; ok {
		t.Fatalf("privacy_level must NOT be serialized on public MomentItem: %s", raw)
	}
	if _, ok := m["reactions"]; !ok {
		t.Fatalf("reactions must always be present: %s", raw)
	}
}

// TestSpecLocationSerializesRelativeFields：location 非空时 RelativeLocation 形状
// 恒定（spec api_spec_types.md RelativeLocation）——distance / bearing_degrees 恒在，
// 无参考点场景为 null（服务端计算，客户端不得本地计算）。
func TestSpecLocationSerializesRelativeFields(t *testing.T) {
	item := MomentItem{MomentID: "m1"}
	item.Location = &Location{}
	item.Location.Point.Latitude = 35.0
	item.Location.Point.Longitude = 112.0
	raw, err := json.Marshal(item)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var m struct {
		Location struct {
			Distance       *float64 `json:"distance"`
			BearingDegrees *float64 `json:"bearing_degrees"`
		} `json:"location"`
	}
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("unmarshal: %v body=%s", err, raw)
	}
	if m.Location.Distance != nil || m.Location.BearingDegrees != nil {
		t.Fatalf("expected distance/bearing_degrees=null without resolved reference point: %s", raw)
	}
}

func TestSpecDeleteNonOwnerForbiddenReason(t *testing.T) {
	repo := NewInMemoryRepository()
	handler := NewHandler(NewServiceWithRepo(repo))
	ctxOther := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "u2"})
	created, err := repo.Create(context.Background(), "u1", CreateMomentRequest{Description: "hi", Location: &Location{LocationName: "Here"}})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/delete", mustReader(`{"moment_id":"`+created.MomentID+`"}`))
	req = req.WithContext(ctxOther)
	rr := httptest.NewRecorder()
	handler.Delete(rr, req)
	var resp struct {
		Code httpx.Code `json:"code"`
		Data struct {
			ForbiddenReason string `json:"forbidden_reason"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Code != httpx.CodeForbidden {
		t.Fatalf("expected 10200, got %d", resp.Code)
	}
	if resp.Data.ForbiddenReason != string(httpx.ReasonNotResourceOwner) {
		t.Fatalf("expected forbidden_reason=not_resource_owner, got %q", resp.Data.ForbiddenReason)
	}
}

func TestSpecCreateBlockedTagReturns10400(t *testing.T) {
	idempotency.SetStore(idempotency.NewInMemoryStore())
	repo := NewInMemoryRepository()
	repo.SetBlockedTags("spam")
	handler := NewHandler(NewServiceWithRepo(repo))
	ctx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "u1"})
	body := `{"description":"hello","location":{"point":{"latitude":1,"longitude":2},"location_name":"Somewhere"},"tags":["ok","spam"]}`
	req := httptest.NewRequest(http.MethodPost, "/create", mustReader(body))
	req.Header.Set("Idempotency-Key", "e1f2a3b4e1f2a3b4e1f2a3b4e1f2a3b4")
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()
	handler.Create(rr, req)
	var resp struct {
		Code httpx.Code `json:"code"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Code != httpx.CodeConflict {
		t.Fatalf("expected 10400, got %d body=%s", resp.Code, rr.Body.String())
	}
}
