package privatemessages

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"hotspot/backend-go/internal/modules/users"
	platformcursor "hotspot/backend-go/internal/platform/cursor"
	"hotspot/backend-go/internal/platform/httpx"
)

// 本文件补齐 handler 的参数校验/错误码映射分支、模块装配 accessor 的防御分支，
// 以及游标编解码与仓储小工具函数的边界分支。

// --- handler: send ---

func TestHandlerSendRequiresIdempotencyKey(t *testing.T) {
	handler := NewHandler(NewServiceWithDeps(NewInMemoryRepository(), &privateMessageUserStub{}))

	req := withPrivateMessagePrincipal(httptest.NewRequest(http.MethodPost, "/send", strings.NewReader(`{"target_user_id":"user_bob","type":"text","content":"hi"}`)))
	rr := httptest.NewRecorder()
	handler.Send(rr, req)

	assertPrivateMessageCode(t, rr, httpx.CodeInvalidRequest)
}

func TestHandlerSendRejectsMalformedJSON(t *testing.T) {
	handler := NewHandler(NewServiceWithDeps(NewInMemoryRepository(), &privateMessageUserStub{}))

	req := withPrivateMessagePrincipal(httptest.NewRequest(http.MethodPost, "/send", strings.NewReader(`{"target_user_id":`)))
	req.Header.Set("Idempotency-Key", "pm_cov_send_bad_json_00000000000001")
	rr := httptest.NewRecorder()
	handler.Send(rr, req)

	assertPrivateMessageCode(t, rr, httpx.CodeInvalidRequest)
}

// APIError 无 Data 时不得生成 data 字段（契约：仅 10200 带 forbidden_reason）。
func TestHandlerSendMapsAPIErrorWithoutData(t *testing.T) {
	userStub := &privateMessageUserStub{
		item:     users.UserItem{Relation: &users.Relationship{}},
		ok:       true,
		settings: users.UserSettings{AllowPrivateChat: users.ChatPermEveryone},
	}
	handler := NewHandler(NewServiceWithDeps(
		&privateMessageErrorRepo{sendErr: &httpx.APIError{Code: httpx.CodeConflict, Message: "冲突"}}, userStub))

	req := withPrivateMessagePrincipal(httptest.NewRequest(http.MethodPost, "/send", strings.NewReader(`{"target_user_id":"user_bob","type":"text","content":"hi"}`)))
	req.Header.Set("Idempotency-Key", "pm_cov_send_api_error_000000000001")
	rr := httptest.NewRecorder()
	handler.Send(rr, req)

	assertPrivateMessageCode(t, rr, httpx.CodeConflict)
	if strings.Contains(rr.Body.String(), "forbidden_reason") {
		t.Fatalf("APIError without data must not emit data: %s", rr.Body.String())
	}
}

func TestHandlerSendMapsInvalidShareCode(t *testing.T) {
	userStub := &privateMessageUserStub{ok: true, settings: users.UserSettings{AllowPrivateChat: users.ChatPermEveryone}}
	handler := NewHandler(NewServiceWithDeps(&privateMessageErrorRepo{sendErr: ErrInvalidShareCode}, userStub))

	req := withPrivateMessagePrincipal(httptest.NewRequest(http.MethodPost, "/send", strings.NewReader(`{"target_user_id":"user_bob","type":"share","share_code":"sc_bad"}`)))
	req.Header.Set("Idempotency-Key", "pm_cov_send_share_code_00000000001")
	rr := httptest.NewRecorder()
	handler.Send(rr, req)

	assertPrivateMessageCode(t, rr, httpx.CodeConflict)
}

func TestHandlerSendMapsFileNotOwned(t *testing.T) {
	userStub := &privateMessageUserStub{ok: true, settings: users.UserSettings{AllowPrivateChat: users.ChatPermEveryone}}
	handler := NewHandler(NewServiceWithDeps(&privateMessageErrorRepo{sendErr: ErrFileNotOwned}, userStub))

	req := withPrivateMessagePrincipal(httptest.NewRequest(http.MethodPost, "/send", strings.NewReader(`{"target_user_id":"user_bob","type":"image","file_id":"file_other"}`)))
	req.Header.Set("Idempotency-Key", "pm_cov_send_file_owned_000000000001")
	rr := httptest.NewRecorder()
	handler.Send(rr, req)

	assertPrivateMessageCode(t, rr, httpx.CodeNotFound)
}

// --- handler: update ---

func TestHandlerUpdateRejectsMalformedJSON(t *testing.T) {
	handler := NewHandler(NewServiceWithDeps(NewInMemoryRepository(), &privateMessageUserStub{}))

	req := withPrivateMessagePrincipal(httptest.NewRequest(http.MethodPost, "/update", strings.NewReader(`{"message_id":`)))
	rr := httptest.NewRecorder()
	handler.Update(rr, req)

	assertPrivateMessageCode(t, rr, httpx.CodeInvalidRequest)
}

// 编辑与 send 同受 content 1-5000 字符契约约束（防先发短消息再编辑绕过上限）。
func TestHandlerUpdateRejectsOverlongContent(t *testing.T) {
	handler := NewHandler(NewServiceWithDeps(NewInMemoryRepository(), &privateMessageUserStub{}))

	body := fmt.Sprintf(`{"message_id":"pm_1","content":"%s"}`, strings.Repeat("好", messageContentMaxRunes+1))
	req := withPrivateMessagePrincipal(httptest.NewRequest(http.MethodPost, "/update", strings.NewReader(body)))
	rr := httptest.NewRecorder()
	handler.Update(rr, req)

	assertPrivateMessageCode(t, rr, httpx.CodeInvalidRequest)
}

func TestHandlerUpdateMapsGenericError(t *testing.T) {
	handler := NewHandler(NewServiceWithDeps(&privateMessageErrorRepo{updateErr: errors.New("db reset")}, &privateMessageUserStub{}))

	req := withPrivateMessagePrincipal(httptest.NewRequest(http.MethodPost, "/update", strings.NewReader(`{"message_id":"pm_1","content":"edited"}`)))
	rr := httptest.NewRecorder()
	handler.Update(rr, req)

	assertPrivateMessageCode(t, rr, httpx.CodeInternalUnavailable)
}

func TestHandlerUpdateSucceeds(t *testing.T) {
	handler := NewHandler(NewServiceWithDeps(
		&privateMessageErrorRepo{updateErr: nil}, &privateMessageUserStub{}))

	req := withPrivateMessagePrincipal(httptest.NewRequest(http.MethodPost, "/update", strings.NewReader(`{"message_id":"pm_1","content":"edited"}`)))
	rr := httptest.NewRecorder()
	handler.Update(rr, req)

	assertPrivateMessageCode(t, rr, httpx.CodeOK)
}

// --- handler: recall ---

func TestHandlerRecallRejectsMalformedJSON(t *testing.T) {
	handler := NewHandler(NewServiceWithDeps(NewInMemoryRepository(), &privateMessageUserStub{}))

	req := withPrivateMessagePrincipal(httptest.NewRequest(http.MethodPost, "/recall", strings.NewReader(`{"message_id":`)))
	rr := httptest.NewRecorder()
	handler.Recall(rr, req)

	assertPrivateMessageCode(t, rr, httpx.CodeInvalidRequest)
}

func TestHandlerRecallMapsGenericError(t *testing.T) {
	handler := NewHandler(NewServiceWithDeps(&privateMessageErrorRepo{recallErr: errors.New("db reset")}, &privateMessageUserStub{}))

	req := withPrivateMessagePrincipal(httptest.NewRequest(http.MethodPost, "/recall", strings.NewReader(`{"message_id":"pm_1"}`)))
	rr := httptest.NewRecorder()
	handler.Recall(rr, req)

	assertPrivateMessageCode(t, rr, httpx.CodeInternalUnavailable)
}

func TestHandlerRecallSucceeds(t *testing.T) {
	handler := NewHandler(NewServiceWithDeps(&privateMessageErrorRepo{}, &privateMessageUserStub{}))

	req := withPrivateMessagePrincipal(httptest.NewRequest(http.MethodPost, "/recall", strings.NewReader(`{"message_id":"pm_1"}`)))
	rr := httptest.NewRecorder()
	handler.Recall(rr, req)

	assertPrivateMessageCode(t, rr, httpx.CodeOK)
}

// --- handler: mark_read ---

// 契约 api_spec_base.md「参数必显式」：scope 影响行为范围，缺省不得静默翻转语义。
func TestHandlerMarkReadRequiresScope(t *testing.T) {
	handler := NewHandler(NewServiceWithDeps(&privateMessageErrorRepo{}, &privateMessageUserStub{}))

	req := withPrivateMessagePrincipal(httptest.NewRequest(http.MethodPost, "/mark_read", strings.NewReader(`{"target_user_id":"user_bob"}`)))
	rr := httptest.NewRecorder()
	handler.MarkRead(rr, req)

	assertPrivateMessageCode(t, rr, httpx.CodeInvalidRequest)
}

// 契约 api_spec_chat.md → mark_read：scope='all' 与 up_to_message_id 互斥。
func TestHandlerMarkReadRejectsUpToMessageIDWithScopeAll(t *testing.T) {
	handler := NewHandler(NewServiceWithDeps(&privateMessageErrorRepo{}, &privateMessageUserStub{}))

	req := withPrivateMessagePrincipal(httptest.NewRequest(http.MethodPost, "/mark_read", strings.NewReader(`{"scope":"all","up_to_message_id":"pm_1"}`)))
	rr := httptest.NewRecorder()
	handler.MarkRead(rr, req)

	assertPrivateMessageCode(t, rr, httpx.CodeInvalidRequest)
}

func TestHandlerMarkReadSessionMapsGenericError(t *testing.T) {
	handler := NewHandler(NewServiceWithDeps(&privateMessageErrorRepo{markErr: errors.New("db reset")}, &privateMessageUserStub{}))

	req := withPrivateMessagePrincipal(httptest.NewRequest(http.MethodPost, "/mark_read", strings.NewReader(`{"scope":"session","target_user_id":"user_bob"}`)))
	rr := httptest.NewRecorder()
	handler.MarkRead(rr, req)

	assertPrivateMessageCode(t, rr, httpx.CodeInternalUnavailable)
}

func TestHandlerMarkReadSessionSucceeds(t *testing.T) {
	handler := NewHandler(NewServiceWithDeps(&privateMessageErrorRepo{}, &privateMessageUserStub{}))

	req := withPrivateMessagePrincipal(httptest.NewRequest(http.MethodPost, "/mark_read", strings.NewReader(`{"scope":"session","target_user_id":"user_bob"}`)))
	rr := httptest.NewRecorder()
	handler.MarkRead(rr, req)

	assertPrivateMessageCode(t, rr, httpx.CodeOK)
}

// --- isValidSendRequest：必填字段缺失 ---

func TestIsValidSendRequestRejectsMissingTargetOrType(t *testing.T) {
	if isValidSendRequest(SendPrivateMessageRequest{Type: "text", Content: "hi"}) {
		t.Fatal("missing target_user_id must be invalid")
	}
	if isValidSendRequest(SendPrivateMessageRequest{TargetUserID: "u2", Content: "hi"}) {
		t.Fatal("missing type must be invalid")
	}
	if !isValidSendRequest(SendPrivateMessageRequest{TargetUserID: "u2", Type: "text", Content: "hi"}) {
		t.Fatal("well-formed text request must be valid")
	}
}

// --- module accessors 防御分支 ---

// covReadServiceStub 只为制造「ReadService 不是 *Service」的装配异常。
type covReadServiceStub struct{ ReadService }

type covChatEnsurerRepo struct {
	Repository
	got []ChatEnsurer
}

func (r *covChatEnsurerRepo) SetChatEnsurer(ensurer ChatEnsurer) { r.got = append(r.got, ensurer) }

func TestModuleServiceAccessorReturnsNilForUnexpectedImpl(t *testing.T) {
	module := &Module{runtime: &Runtime{readService: covReadServiceStub{}}}
	if svc := module.Service(); svc != nil {
		t.Fatalf("expected nil service for non-*Service read impl, got %#v", svc)
	}
}

func TestModuleSetChatEnsurerDelegatesToRepo(t *testing.T) {
	repo := &covChatEnsurerRepo{Repository: NewInMemoryRepository()}
	module := &Module{runtime: &Runtime{readService: NewServiceWithRepo(repo), repo: repo}}

	ensurer := &fakeChatEnsurer{}
	module.SetChatEnsurer(ensurer)
	if len(repo.got) != 1 || repo.got[0] != ensurer {
		t.Fatalf("expected chat ensurer to be injected once, got %#v", repo.got)
	}
}

// --- 游标编解码边界 ---

// 游标内 sent_at 不是 RFC3339Nano 时必须返回 ErrInvalidCursor（fail-closed，
// 不得把非法时间当零值用于翻页）。
func TestDecodeMessageCursorRejectsNonRFC3339SentAt(t *testing.T) {
	encoded, err := platformcursor.Encode("private_messages.history", messageCursor{SentAt: "not-a-timestamp", MessageID: "pm_1"})
	if err != nil {
		t.Fatalf("encode fixture: %v", err)
	}
	if _, _, err := decodeMessageCursor(encoded); !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("expected ErrInvalidCursor, got %v", err)
	}
}

func TestEncodeDecodeMessageCursorRoundTrip(t *testing.T) {
	cursor := encodeMessageCursor("2026-03-30T10:00:00.123456Z", "pm_1")
	if cursor == "" {
		t.Fatal("cursor must be encoded")
	}
	sentAt, messageID, err := decodeMessageCursor(cursor)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if messageID != "pm_1" || sentAt.Format("2006-01-02T15:04:05.999999Z07:00") != "2026-03-30T10:00:00.123456Z" {
		t.Fatalf("round trip mismatch: %s %s", messageID, sentAt)
	}
	zeroAt, zeroID, zeroErr := decodeMessageCursor("")
	if !zeroAt.IsZero() || zeroID != "" || zeroErr != nil {
		t.Fatalf("empty cursor must decode to zero value: %v %q %v", zeroAt, zeroID, zeroErr)
	}
}

// --- 小工具函数 ---

func TestOtherUserIDReturnsEmptyForMalformedChatKey(t *testing.T) {
	if got := otherUserID("usr_a", "usr_a"); got != "" {
		t.Fatalf("malformed chat key must yield empty peer, got %q", got)
	}
	if got := otherUserID("usr_a:usr_b", "usr_a"); got != "usr_b" {
		t.Fatalf("expected usr_b, got %q", got)
	}
	if got := otherUserID("usr_a:usr_b", "usr_b"); got != "usr_a" {
		t.Fatalf("expected usr_a, got %q", got)
	}
}

func TestStringOrNil(t *testing.T) {
	if value := stringOrNil(""); value != nil {
		t.Fatalf("empty string must map to nil, got %#v", value)
	}
	if value := stringOrNil("file_1"); value != "file_1" {
		t.Fatalf("non-empty string must pass through, got %#v", value)
	}
}
