package reactions

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/httpx"
)

// apiCode extracts the envelope code from a response body.
func apiCode(t *testing.T, body string) httpx.Code {
	t.Helper()
	var env struct {
		Code httpx.Code `json:"code"`
	}
	if err := json.Unmarshal([]byte(body), &env); err != nil {
		t.Fatalf("unmarshal response %q: %v", body, err)
	}
	return env.Code
}

func do(t *testing.T, handler http.HandlerFunc, userID, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	if userID != "" {
		req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: userID, SessionID: "s1"}))
	}
	rr := httptest.NewRecorder()
	handler(rr, req)
	return rr
}

func TestHandlerSetHappyPath(t *testing.T) {
	svc := NewServiceWithDeps(NewInMemoryRepository(), []TargetProvider{
		stubTargetProvider{kind: ReactTargetMoment, exists: map[string]bool{"m1": true}, owners: map[string]string{"m1": "owner"}},
	}, nil, nil)
	h := NewHandler(svc)

	rr := do(t, h.Set, "u_bob", `{"target_type":"moment","target_id":"m1","emoji":"👍","active":true}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("status: %d", rr.Code)
	}
	if code := apiCode(t, rr.Body.String()); code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
}

func TestHandlerSetRejectsMissingFields(t *testing.T) {
	svc := NewServiceWithDeps(NewInMemoryRepository(), nil, nil, nil)
	h := NewHandler(svc)

	cases := []string{
		`{}`, // missing target_id and emoji
		`{"target_type":"moment","emoji":"👍"}`,      // missing target_id
		`{"target_type":"moment","target_id":"m1"}`, // missing emoji
	}
	for i, body := range cases {
		rr := do(t, h.Set, "u", body)
		if code := apiCode(t, rr.Body.String()); code != httpx.CodeInvalidRequest {
			t.Fatalf("case %d: expected httpx.CodeInvalidRequest, got %d", i, code)
		}
	}
}

func TestHandlerSetRejectsBadTargetType(t *testing.T) {
	svc := NewServiceWithDeps(NewInMemoryRepository(), nil, nil, nil)
	h := NewHandler(svc)

	rr := do(t, h.Set, "u", `{"target_type":"bogus","target_id":"x","emoji":"👍","active":true}`)
	if code := apiCode(t, rr.Body.String()); code != httpx.CodeInvalidRequest {
		t.Fatalf("bad target_type: expected httpx.CodeInvalidRequest, got %d", code)
	}
}

func TestHandlerSetMapsTargetNotFoundTo10300(t *testing.T) {
	provider := stubTargetProvider{kind: ReactTargetMoment, exists: map[string]bool{}} // nothing exists
	svc := NewServiceWithDeps(NewInMemoryRepository(), []TargetProvider{provider}, nil, nil)
	h := NewHandler(svc)

	rr := do(t, h.Set, "u", `{"target_type":"moment","target_id":"missing","emoji":"👍","active":true}`)
	if code := apiCode(t, rr.Body.String()); code != httpx.CodeNotFound {
		t.Fatalf("missing target: expected httpx.CodeNotFound, got %d", code)
	}
}

func TestHandlerListReturnsItems(t *testing.T) {
	svc := NewServiceWithDeps(NewInMemoryRepository(), nil, nil, nil)
	// Seed two reactions directly via repo.
	if _, err := svc.repo.React(context.Background(), ReactTargetMoment, "m1", "u1", "👍"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.repo.React(context.Background(), ReactTargetMoment, "m1", "u2", "👍"); err != nil {
		t.Fatal(err)
	}
	h := NewHandler(svc)

	rr := do(t, h.List, "", `{"target_type":"moment","target_id":"m1"}`)
	if code := apiCode(t, rr.Body.String()); code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	// Confirm items array carries 2 reactions.
	var env struct {
		Data struct {
			Items []ReactionItem `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(rr.Body.String()), &env); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(env.Data.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(env.Data.Items))
	}
}

func TestHandlerListRejectsMissingTargetID(t *testing.T) {
	svc := NewServiceWithDeps(NewInMemoryRepository(), nil, nil, nil)
	h := NewHandler(svc)

	rr := do(t, h.List, "", `{"target_type":"moment"}`)
	if code := apiCode(t, rr.Body.String()); code != httpx.CodeInvalidRequest {
		t.Fatalf("missing target_id: expected httpx.CodeInvalidRequest, got %d", code)
	}
}

func TestHandlerListByMessage(t *testing.T) {
	svc := NewServiceWithDeps(NewInMemoryRepository(), nil, nil, nil)
	if _, err := svc.repo.React(context.Background(), ReactTargetMessage, "msg1", "u1", "❤️"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.repo.React(context.Background(), ReactTargetMessage, "msg1", "u2", "❤️"); err != nil {
		t.Fatal(err)
	}
	h := NewHandler(svc)

	rr := do(t, h.List, "", `{"target_type":"message","target_id":"msg1"}`)
	if code := apiCode(t, rr.Body.String()); code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}

	var env struct {
		Data struct {
			Items []ReactionItem `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(rr.Body.String()), &env); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(env.Data.Items) != 2 {
		t.Fatalf("expected 2 items for message target, got %d", len(env.Data.Items))
	}
	if env.Data.Items[0].Emoji != "❤️" {
		t.Fatalf("expected emoji ❤️, got %s", env.Data.Items[0].Emoji)
	}
}

func TestHandlerListByComment(t *testing.T) {
	svc := NewServiceWithDeps(NewInMemoryRepository(), nil, nil, nil)
	if _, err := svc.repo.React(context.Background(), ReactTargetComment, "c1", "u1", "😂"); err != nil {
		t.Fatal(err)
	}
	h := NewHandler(svc)

	rr := do(t, h.List, "", `{"target_type":"comment","target_id":"c1"}`)
	if code := apiCode(t, rr.Body.String()); code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}

	var env struct {
		Data struct {
			Items []ReactionItem `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(rr.Body.String()), &env); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(env.Data.Items) != 1 {
		t.Fatalf("expected 1 item for comment target, got %d", len(env.Data.Items))
	}
	if env.Data.Items[0].Emoji != "😂" {
		t.Fatalf("expected emoji 😂, got %s", env.Data.Items[0].Emoji)
	}
}

func TestHandlerListRejectsBadTargetType(t *testing.T) {
	svc := NewServiceWithDeps(NewInMemoryRepository(), nil, nil, nil)
	h := NewHandler(svc)

	rr := do(t, h.List, "", `{"target_type":"bogus","target_id":"x"}`)
	if code := apiCode(t, rr.Body.String()); code != httpx.CodeInvalidRequest {
		t.Fatalf("bad target_type: expected httpx.CodeInvalidRequest, got %d", code)
	}
}
