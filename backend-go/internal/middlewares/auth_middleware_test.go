package middlewares

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"backend-go/internal/auth"
	"backend-go/internal/repository"
)

// fakeTokenManager 实现 auth.TokenManager，用于中间件测试。
type fakeTokenManager struct {
	accessToken string
	userID      string
	sessionID   string
	expiresAt   time.Time
}

func (m *fakeTokenManager) GenerateTokens(userID, sessionID string) (string, string, time.Time, time.Time, error) {
	return "", "", time.Time{}, time.Time{}, nil
}

func (m *fakeTokenManager) ParseAccessToken(token string) (string, string, time.Time, error) {
	if token != m.accessToken {
		return "", "", time.Time{}, http.ErrNoCookie
	}
	return m.userID, m.sessionID, m.expiresAt, nil
}

func (m *fakeTokenManager) ParseRefreshToken(token string) (string, string, time.Time, error) {
	return "", "", time.Time{}, http.ErrNoCookie
}

var _ auth.TokenManager = (*fakeTokenManager)(nil)

// fakeTokenBlacklistRepository 实现 repository.TokenBlacklistRepository，仅用于记录传入 token。
type fakeTokenBlacklistRepository struct {
	blacklisted map[string]bool
}

func newFakeTokenBlacklistRepository() *fakeTokenBlacklistRepository {
	return &fakeTokenBlacklistRepository{blacklisted: make(map[string]bool)}
}

func (r *fakeTokenBlacklistRepository) BlacklistToken(_ context.Context, token string, _ time.Time) error {
	if token == "" {
		return nil
	}
	r.blacklisted[token] = true
	return nil
}

func (r *fakeTokenBlacklistRepository) IsTokenBlacklisted(_ context.Context, token string) (bool, error) {
	return r.blacklisted[token], nil
}

var _ repository.TokenBlacklistRepository = (*fakeTokenBlacklistRepository)(nil)

// helper：构造带中间件的测试路由
func newTestRouter(mw func(http.Handler) http.Handler, handler http.HandlerFunc) *chi.Mux {
	r := chi.NewRouter()
	if mw != nil {
		r.Use(mw)
	}
	r.Get("/protected", handler)
	return r
}

func TestAuthOptional_NoToken_AllowsRequestWithoutAuthInfo(t *testing.T) {
	manager := &fakeTokenManager{}
	black := newFakeTokenBlacklistRepository()

	var called bool
	r := newTestRouter(AuthOptional(manager, black), func(w http.ResponseWriter, r *http.Request) {
		called = true
		if info, ok := auth.FromContext(r.Context()); ok || info != nil {
			t.Fatalf("expected no AuthInfo in context when no token provided")
		}
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if !called {
		t.Fatalf("expected handler to be called when token is optional")
	}
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestAuthOptional_ValidToken_AttachesAuthInfo(t *testing.T) {
	exp := time.Now().UTC().Add(5 * time.Minute)
	manager := &fakeTokenManager{
		accessToken: "valid-access",
		userID:      "user-1",
		sessionID:   "sess-1",
		expiresAt:   exp,
	}
	black := newFakeTokenBlacklistRepository()

	var called bool
	r := newTestRouter(AuthOptional(manager, black), func(w http.ResponseWriter, r *http.Request) {
		called = true
		info, ok := auth.FromContext(r.Context())
		if !ok || info == nil {
			t.Fatalf("expected AuthInfo in context for valid token")
		}
		if info.UserID != "user-1" || info.SessionID != "sess-1" {
			t.Fatalf("unexpected auth info: %+v", info)
		}
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer valid-access")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if !called {
		t.Fatalf("expected handler to be called for valid token")
	}
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestAuthRequired_NoToken_ReturnsAuthFailed(t *testing.T) {
	manager := &fakeTokenManager{}
	black := newFakeTokenBlacklistRepository()

	r := newTestRouter(AuthRequired(manager, black), func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("handler should not be called when token is required but missing")
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200 envelope, got %d", w.Code)
	}
}

func TestAuthRequired_BlacklistedToken_ReturnsAuthFailed(t *testing.T) {
	exp := time.Now().UTC().Add(5 * time.Minute)
	manager := &fakeTokenManager{
		accessToken: "valid-access",
		userID:      "user-1",
		sessionID:   "sess-1",
		expiresAt:   exp,
	}
	black := newFakeTokenBlacklistRepository()
	_ = black.BlacklistToken(context.Background(), "valid-access", exp)

	r := newTestRouter(AuthRequired(manager, black), func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("handler should not be called for blacklisted token")
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer valid-access")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200 envelope, got %d", w.Code)
	}
}
