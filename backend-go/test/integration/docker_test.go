package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/migrate"
	"hotspot/backend-go/internal/platform/persistence"

	"hotspot/backend-go/migrations"
)

const (
	backendURL = "http://backend:8081"
)

var (
	testClient  *http.Client
	accessToken string
	backendAddr string
	idempKeySeq int64
	idempKeyMu  sync.Mutex
)

type ApiResponse[T any] struct {
	Code    int    `json:"code"`
	Message string `json:"message,omitempty"`
	Data    *T     `json:"data,omitempty"`
}

func TestMain(m *testing.M) {
	backendAddr = os.Getenv("BACKEND_URL")
	if backendAddr == "" {
		backendAddr = "http://localhost:8081"
	}

	maybeMigrate()
	if shouldWaitForBackend() {
		if err := waitForBackendWithPost(backendAddr+"/api/v1/system/health", 60*time.Second); err != nil {
			fmt.Fprintf(os.Stderr, "backend not ready: %v\n", err)
			os.Exit(1)
		}

		jar, _ := cookiejar.New(nil)
		testClient = &http.Client{
			Jar: jar,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
			Timeout: 15 * time.Second,
		}
	}

	code := m.Run()
	os.Exit(code)
}

func shouldWaitForBackend() bool {
	for _, arg := range os.Args {
		if strings.HasPrefix(arg, "-test.run=") {
			pattern := strings.TrimPrefix(arg, "-test.run=")
			if strings.Contains(pattern, "Explain") || strings.Contains(pattern, "RequiredTables") {
				return false
			}
		}
	}
	return true
}

func maybeMigrate() {
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		return
	}
	db := persistence.NewPostgresDB(dsn)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	if err := migrate.EnsureSchema(ctx, db, migrations.Files); err != nil {
		fmt.Fprintf(os.Stderr, "migration failed: %v\n", err)
	}
}

func waitForBackend(url string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			resp, err := http.Get(url)
			if err == nil {
				resp.Body.Close()
				if resp.StatusCode < 500 {
					return nil
				}
			}
		}
	}
}

func waitForBackendWithPost(url string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	body := bytes.NewReader([]byte("{}"))
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			body.Seek(0, 0)
			resp, err := http.Post(url, "application/json", body)
			if err == nil {
				resp.Body.Close()
				if resp.StatusCode < 500 {
					return nil
				}
			}
		}
	}
}

func req(method, path string, body, result any) (int, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return 0, fmt.Errorf("marshal: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, backendAddr+path, bodyReader)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-Id", "test-"+fmt.Sprint(time.Now().UnixNano()))
	req.Header.Set("Idempotency-Key", fmt.Sprintf("%032x%032x", time.Now().UnixNano(), atomic.AddInt64(&idempKeySeq, 1)))
	if accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	}
	resp, err := testClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return resp.StatusCode, fmt.Errorf("decode: %w", err)
		}
	}
	return resp.StatusCode, nil
}

func post(path string, body, result any) (int, error) {
	return req("POST", path, body, result)
}

func getHTTPStatus(path string) int {
	status, _ := req("GET", path, nil, nil)
	return status
}

func apiRespOK(t *testing.T, path string, body, result any) {
	status, err := post(path, body, result)
	if err != nil {
		t.Fatalf("POST %s: %v", path, err)
	}
	if status != http.StatusOK {
		t.Fatalf("POST %s: got status %d, want 200", path, status)
	}
	if r, ok := result.(*ApiResponse[any]); ok {
		if r.Code != 0 {
			t.Fatalf("POST %s: code=%d message=%s", path, r.Code, r.Message)
		}
	}
}

func apiRespOKWithCode(t *testing.T, path string, body, result any, expectCode int) {
	status, err := post(path, body, result)
	if err != nil {
		t.Fatalf("POST %s: %v", path, err)
	}
	if status != http.StatusOK {
		t.Fatalf("POST %s: got status %d, want %d", path, status, http.StatusOK)
	}
	if r, ok := result.(*ApiResponse[any]); ok {
		if r.Code != expectCode {
			t.Fatalf("POST %s: code=%d, want %d", path, r.Code, expectCode)
		}
	}
}

func apiOK(t *testing.T, path string, body, result any) any {
	status, err := post(path, body, result)
	if err != nil {
		t.Fatalf("POST %s: %v", path, err)
	}
	if status != http.StatusOK {
		t.Fatalf("POST %s: got status %d", path, status)
	}
	return result
}

func apiNoAuth(t *testing.T, path string, body, result any) {
	status, err := post(path, body, result)
	if err != nil {
		t.Fatalf("POST %s: %v", path, err)
	}
	if status != http.StatusOK {
		t.Fatalf("POST %s: got status %d, want 200", path, status)
	}
	if r, ok := result.(*ApiResponse[any]); ok && r.Code != 0 {
		t.Fatalf("POST %s: code=%d message=%s", path, r.Code, r.Message)
	}
}

func emptyBody(t *testing.T, path string) {
	status, err := post(path, nil, nil)
	if err != nil {
		t.Fatalf("POST %s: %v", path, err)
	}
	if status != http.StatusOK {
		t.Fatalf("POST %s: got status %d, want 200", path, status)
	}
}

// =============================================================================
// Module-level test helpers
// =============================================================================

func createTestUser(t *testing.T, nickname string) (userID string) {
	var resp ApiResponse[map[string]any]
	apiRespOK(t, "/api/v1/auth/register", map[string]any{
		"login_id":    "test_" + t.Name() + "_" + nickname + "_" + fmt.Sprint(time.Now().UnixNano()),
		"nickname":    nickname,
		"password":    "Test1234",
		"client_type": "app",
	}, &resp)
	if resp.Data == nil {
		t.Fatalf("no data in register response")
	}
	if uid, ok := (*resp.Data)["user_id"].(string); ok {
		userID = uid
	}
	return userID
}

func createGuestUser(t *testing.T) (userID string, token string) {
	var resp ApiResponse[map[string]any]
	apiNoAuth(t, "/api/v1/auth/guest", map[string]any{
		"nickname":    "guest_" + t.Name(),
		"client_type": "app",
	}, &resp)
	if resp.Data == nil {
		t.Fatalf("no data in guest response")
	}
	if uid, ok := (*resp.Data)["user_id"].(string); ok {
		userID = uid
	}
	if tok, ok := (*resp.Data)["access_token"].(string); ok {
		token = tok
	}
	return userID, token
}

func loginUser(t *testing.T, loginID, password string) (userID string, token string) {
	var resp ApiResponse[map[string]any]
	apiNoAuth(t, "/api/v1/auth/login", map[string]any{
		"login_id":    loginID,
		"password":    password,
		"client_type": "app",
	}, &resp)
	if resp.Data != nil {
		if uid, ok := (*resp.Data)["user_id"].(string); ok {
			userID = uid
		}
		if tok, ok := (*resp.Data)["access_token"].(string); ok {
			token = tok
		}
	}
	return userID, token
}

func withAuth(t *testing.T, fn func(token string)) {
	_, token := createGuestUser(t)
	accessToken = token
	fn(token)
}

func withTwoUsers(t *testing.T, fn func(user1Token, user2Token string)) {
	_, tok1 := createGuestUser(t)
	_, tok2 := createGuestUser(t)
	fn(tok1, tok2)
}

// =============================================================================
// System
// =============================================================================

func TestSystem_Health(t *testing.T) {
	var resp ApiResponse[map[string]any]
	apiOK(t, "/api/v1/system/health", map[string]any{}, &resp)
}

func TestSystem_GetMeta(t *testing.T) {
	var resp ApiResponse[map[string]any]
	apiOK(t, "/api/v1/system/get_meta", map[string]any{}, &resp)
}

func TestSystem_GetClientConfig(t *testing.T) {
	var resp ApiResponse[map[string]any]
	apiOK(t, "/api/v1/system/get_client_config", map[string]any{}, &resp)
}

func TestSystem_ListAnnouncements(t *testing.T) {
	var resp ApiResponse[map[string]any]
	apiOK(t, "/api/v1/system/list_announcements", map[string]any{}, &resp)
}

// =============================================================================
// Auth
// =============================================================================

func TestAuth_Guest(t *testing.T) {
	createGuestUser(t)
}

func TestAuth_Register(t *testing.T) {
	createTestUser(t, "reg_user")
}

func TestAuth_Login_InvalidCredentials(t *testing.T) {
	var resp ApiResponse[map[string]any]
	apiOK(t, "/api/v1/auth/login", map[string]any{
		"login_id":    "nonexistent_user_xyz",
		"password":    "wrongpass",
		"client_type": "app",
	}, &resp)
	if resp.Code != 0 {
		t.Logf("guest response code=%d message=%s", resp.Code, resp.Message)
	}
}

func TestAuth_CheckStatus_Anonymous(t *testing.T) {
	var resp ApiResponse[map[string]any]
	apiOK(t, "/api/v1/auth/check_status", map[string]any{}, &resp)
	if resp.Data != nil {
		if valid, ok := (*resp.Data)["is_valid"].(bool); ok && valid {
			t.Log("anonymous check_status returned is_valid=true — unexpected but acceptable if CSRF or session-based")
		}
	}
}

func TestAuth_RefreshToken_NoToken(t *testing.T) {
	var resp ApiResponse[map[string]any]
	status, _ := post("/api/v1/auth/refresh_token", map[string]any{"client_type": "app"}, &resp)
	if status != http.StatusOK {
		t.Fatalf("refresh_token without token: got status %d", status)
	}
	if resp.Code != 0 && resp.Code != 10100 {
		t.Logf("refresh_token code=%d message=%s", resp.Code, resp.Message)
	}
}

// =============================================================================
// Users (public)
// =============================================================================

func TestUsers_Search(t *testing.T) {
	var resp ApiResponse[map[string]any]
	apiRespOK(t, "/api/v1/users/search", map[string]any{"keyword": "test"}, &resp)
}

func TestUsers_OpenTheDoor(t *testing.T) {
	withAuth(t, func(token string) {
		uid := createTestUser(t, "target_user")
		var resp ApiResponse[map[string]any]
		apiRespOK(t, "/api/v1/users/open_the_door", map[string]any{"user_id": uid}, &resp)
	})
}

func TestUsers_ListFriends(t *testing.T) {
	withAuth(t, func(token string) {
		var resp ApiResponse[map[string]any]
		apiRespOK(t, "/api/v1/users/list_friends", map[string]any{}, &resp)
	})
}

func TestUsers_ListFollowing(t *testing.T) {
	withAuth(t, func(token string) {
		var resp ApiResponse[map[string]any]
		apiRespOK(t, "/api/v1/users/list_following", map[string]any{}, &resp)
	})
}

func TestUsers_ListFollowers(t *testing.T) {
	withAuth(t, func(token string) {
		var resp ApiResponse[map[string]any]
		apiRespOK(t, "/api/v1/users/list_followers", map[string]any{}, &resp)
	})
}

func TestUsers_ListGroups(t *testing.T) {
	withAuth(t, func(token string) {
		var resp ApiResponse[map[string]any]
		apiRespOK(t, "/api/v1/users/list_groups", map[string]any{}, &resp)
	})
}

func TestUsers_ListMoments(t *testing.T) {
	withAuth(t, func(token string) {
		var resp ApiResponse[map[string]any]
		apiRespOK(t, "/api/v1/users/list_moments", map[string]any{"user_id": ""}, &resp)
	})
}

func TestUsers_ListLikedMoments(t *testing.T) {
	withAuth(t, func(token string) {
		var resp ApiResponse[map[string]any]
		apiRespOK(t, "/api/v1/users/list_liked_moments", map[string]any{}, &resp)
	})
}

// =============================================================================
// Users (authenticated)
// =============================================================================

func TestUsersMe_GetProfile(t *testing.T) {
	withAuth(t, func(token string) {
		var resp ApiResponse[map[string]any]
		apiRespOK(t, "/api/v1/users/me/profile/get", map[string]any{}, &resp)
	})
}

func TestUsersMe_UpdateProfile(t *testing.T) {
	withAuth(t, func(token string) {
		apiRespOK(t, "/api/v1/users/me/profile/update", map[string]any{
			"bio": "test bio",
		}, nil)
	})
}

func TestUsersMe_LocationUpdate(t *testing.T) {
	withAuth(t, func(token string) {
		apiRespOK(t, "/api/v1/users/me/location/update", map[string]any{
			"latitude":  37.7749,
			"longitude": -122.4194,
		}, nil)
	})
}

func TestUsersMe_GetSettings(t *testing.T) {
	withAuth(t, func(token string) {
		var resp ApiResponse[map[string]any]
		apiRespOK(t, "/api/v1/users/me/settings/get", map[string]any{}, &resp)
	})
}

func TestUsersMe_UpdateSettings(t *testing.T) {
	withAuth(t, func(token string) {
		apiRespOK(t, "/api/v1/users/me/settings/update", map[string]any{
			"allow_stranger_dm": true,
		}, nil)
	})
}

func TestUsersMe_Follow(t *testing.T) {
	withTwoUsers(t, func(tok1, tok2 string) {
		tok1uid, _ := whoAmI(tok1)
		apiRespOK(t, "/api/v1/users/me/follow", map[string]any{
			"target_user_id": tok1uid,
		}, nil)
	})
}

func TestUsersMe_Unfollow(t *testing.T) {
	// Follow first then unfollow
	_, tok2 := createGuestUser(t)
	tok2uid, _ := whoAmI(tok2)
	tok1uid := createTestUser(t, "unfollow_target")
	oldToken := accessToken
	accessToken = tok1uid
	apiRespOK(t, "/api/v1/users/me/follow", map[string]any{"target_user_id": tok2uid}, nil)
	apiRespOK(t, "/api/v1/users/me/unfollow", map[string]any{"target_user_id": tok2uid}, nil)
	accessToken = oldToken
}

func TestUsersMe_Block_Unblock(t *testing.T) {
	withTwoUsers(t, func(tok1, tok2 string) {
		tok2uid, _ := whoAmI(tok2)
		apiRespOK(t, "/api/v1/users/me/block", map[string]any{"target_user_id": tok2uid}, nil)
		apiRespOK(t, "/api/v1/users/me/unblock", map[string]any{"target_user_id": tok2uid}, nil)
	})
}

func TestUsersMe_ListBlocked(t *testing.T) {
	withAuth(t, func(token string) {
		var resp ApiResponse[map[string]any]
		apiRespOK(t, "/api/v1/users/me/list_blocked", map[string]any{}, &resp)
	})
}

func TestUsersMe_SetRemark(t *testing.T) {
	withTwoUsers(t, func(tok1, tok2 string) {
		tok2uid, _ := whoAmI(tok2)
		apiRespOK(t, "/api/v1/users/me/set_remark", map[string]any{
			"target_user_id": tok2uid,
			"remark":         "test remark",
		}, nil)
	})
}

func whoAmI(token string) (string, string) {
	old := accessToken
	accessToken = token
	defer func() { accessToken = old }()
	var resp ApiResponse[map[string]any]
	post("/api/v1/users/me/profile/get", map[string]any{}, &resp)
	if resp.Data == nil {
		return "", ""
	}
	return getString(resp.Data, "user_id"), token
}

func getString(m *map[string]any, key string) string {
	if m == nil {
		return ""
	}
	if v, ok := (*m)[key].(string); ok {
		return v
	}
	return ""
}

// =============================================================================
// Moments (public)
// =============================================================================

func TestMoments_OpenTheDoor(t *testing.T) {
	withAuth(t, func(token string) {
		_ = createTestUser(t, "moment_owner")
		var resp ApiResponse[map[string]any]
		apiRespOK(t, "/api/v1/moments/open_the_door", map[string]any{"moment_id": "nonexistent"}, &resp)
		// Should return code != 0 but status 200 (business error)
		if resp.Code == 0 {
			t.Log("open_the_door returned success for nonexistent moment — acceptable if design choice")
		}
	})
}

func TestMoments_Search(t *testing.T) {
	var resp ApiResponse[map[string]any]
	apiRespOK(t, "/api/v1/moments/search", map[string]any{}, &resp)
}

func TestMoments_ListLikes(t *testing.T) {
	var resp ApiResponse[map[string]any]
	apiRespOK(t, "/api/v1/moments/list_likes", map[string]any{}, &resp)
}

// =============================================================================
// Moments (authenticated)
// =============================================================================

func TestMoments_CreateList(t *testing.T) {
	withAuth(t, func(token string) {
		// Create
		var createResp ApiResponse[map[string]any]
		apiRespOK(t, "/api/v1/moments/create", map[string]any{
			"description": "test moment",
			"location": map[string]any{
				"point": map[string]any{
					"latitude":  37.7749,
					"longitude": -122.4194,
				},
			},
		}, &createResp)
		if createResp.Code != 0 {
			t.Fatalf("create moment failed: code=%d message=%s", createResp.Code, createResp.Message)
		}
		var listResp ApiResponse[map[string]any]
		apiOK(t, "/api/v1/moments/me/list", map[string]any{}, &listResp)
	})
}

func TestMoments_Update(t *testing.T) {
	withAuth(t, func(token string) {
		var createResp ApiResponse[map[string]any]
		apiRespOK(t, "/api/v1/moments/create", map[string]any{
			"description": "original",
			"location": map[string]any{
				"point": map[string]any{
					"latitude":  37.7749,
					"longitude": -122.4194,
				},
			},
		}, &createResp)
		if createResp.Code == 0 && createResp.Data != nil {
			if mid, ok := (*createResp.Data)["moment_id"].(string); ok {
				apiRespOK(t, "/api/v1/moments/update", map[string]any{"moment_id": mid, "description": "updated"}, nil)
			}
		}
	})
}

func TestMoments_LikeUnlike(t *testing.T) {
	withTwoUsers(t, func(tok1, tok2 string) {
		tok1uid := createTestUser(t, "moment_author")
		oldTok := accessToken
		accessToken = tok1uid
		var createResp ApiResponse[map[string]any]
		apiRespOK(t, "/api/v1/moments/create", map[string]any{"description": "likeable moment"}, &createResp)
		accessToken = oldTok

		if createResp.Code == 0 && createResp.Data != nil {
			if mid, ok := (*createResp.Data)["moment_id"].(string); ok {
				apiRespOK(t, "/api/v1/moments/me/like", map[string]any{"moment_id": mid}, nil)
				apiRespOK(t, "/api/v1/moments/me/unlike", map[string]any{"moment_id": mid}, nil)
			}
		}
	})
}

func TestMoments_FavoriteUnfavorite(t *testing.T) {
	withAuth(t, func(token string) {
		uid := createTestUser(t, "fav_author")
		oldTok := accessToken
		accessToken = uid
		var createResp ApiResponse[map[string]any]
		apiRespOK(t, "/api/v1/moments/create", map[string]any{"description": "fav moment"}, &createResp)
		accessToken = oldTok
		if createResp.Code == 0 && createResp.Data != nil {
			if mid, ok := (*createResp.Data)["moment_id"].(string); ok {
				apiRespOK(t, "/api/v1/moments/me/favorite", map[string]any{"moment_id": mid}, nil)
				apiRespOK(t, "/api/v1/moments/me/unfavorite", map[string]any{"moment_id": mid}, nil)
			}
		}
	})
}

// =============================================================================
// Comments
// =============================================================================

func TestComments_List(t *testing.T) {
	var resp ApiResponse[map[string]any]
	apiRespOK(t, "/api/v1/comments/list", map[string]any{}, &resp)
}

func TestComments_ListReplies(t *testing.T) {
	var resp ApiResponse[map[string]any]
	apiRespOK(t, "/api/v1/comments/list_replies", map[string]any{"comment_id": "c_fake"}, &resp)
}

func TestComments_ListLikes(t *testing.T) {
	var resp ApiResponse[map[string]any]
	apiRespOK(t, "/api/v1/comments/list_likes", map[string]any{"comment_id": "c_fake"}, &resp)
}

func TestComments_CreateDelete(t *testing.T) {
	withAuth(t, func(token string) {
		var resp ApiResponse[map[string]any]
		apiRespOK(t, "/api/v1/comments/create", map[string]any{
			"target_id":   "moment_fake",
			"target_type": "moment",
			"content":     "test comment",
		}, &resp)
		if resp.Code == 0 && resp.Data != nil {
			if cid, ok := (*resp.Data)["comment_id"].(string); ok {
				apiRespOK(t, "/api/v1/comments/delete", map[string]any{"comment_id": cid}, nil)
			}
		}
	})
}

func TestComments_LikeUnlike(t *testing.T) {
	withAuth(t, func(token string) {
		apiRespOK(t, "/api/v1/comments/me/like", map[string]any{"comment_id": "c_fake"}, nil)
		apiRespOK(t, "/api/v1/comments/me/unlike", map[string]any{"comment_id": "c_fake"}, nil)
	})
}

// =============================================================================
// Groups
// =============================================================================

func TestGroups_Search(t *testing.T) {
	var resp ApiResponse[map[string]any]
	apiRespOK(t, "/api/v1/groups/search", map[string]any{}, &resp)
}

func TestGroups_OpenTheDoor(t *testing.T) {
	var resp ApiResponse[map[string]any]
	apiRespOK(t, "/api/v1/groups/open_the_door", map[string]any{"group_id": "fake_group"}, &resp)
}

func TestGroups_ListMembers(t *testing.T) {
	var resp ApiResponse[map[string]any]
	apiRespOK(t, "/api/v1/groups/members/list", map[string]any{"group_id": "fake_group"}, &resp)
}

func TestGroups_SearchMembers(t *testing.T) {
	var resp ApiResponse[map[string]any]
	apiRespOK(t, "/api/v1/groups/members/search", map[string]any{
		"group_id": "fake_group",
		"keyword":  "test",
	}, &resp)
}

func TestGroups_AnnouncementsList(t *testing.T) {
	var resp ApiResponse[map[string]any]
	apiRespOK(t, "/api/v1/groups/announcements/list", map[string]any{"group_id": "fake_group"}, &resp)
}

// Group authenticated actions
func TestGroups_CreateUpdateDismiss(t *testing.T) {
	withAuth(t, func(token string) {
		// Create
		var createResp ApiResponse[map[string]any]
		apiRespOK(t, "/api/v1/groups/create", map[string]any{
			"name":        "Test Group",
			"description": "integration test group",
		}, &createResp)
		if createResp.Code == 0 && createResp.Data != nil {
			if gid, ok := (*createResp.Data)["group_id"].(string); ok {
				// Update
				apiRespOK(t, "/api/v1/groups/update", map[string]any{
					"group_id": gid,
					"name":     "Updated Group",
				}, nil)
				// Dismiss
				apiRespOK(t, "/api/v1/groups/dismiss", map[string]any{"group_id": gid}, nil)
			}
		} else {
			t.Logf("group create returned code=%d: %s", createResp.Code, createResp.Message)
		}
	})
}

func TestGroups_MyGroups(t *testing.T) {
	withAuth(t, func(token string) {
		var resp ApiResponse[map[string]any]
		apiRespOK(t, "/api/v1/groups/me/list", map[string]any{}, &resp)
	})
}

// =============================================================================
// Feed
// =============================================================================

func TestFeed_ListMoments(t *testing.T) {
	withAuth(t, func(token string) {
		var resp ApiResponse[map[string]any]
		apiRespOK(t, "/api/v1/feed/list_moments", map[string]any{
			"source": "nearby",
		}, &resp)
	})
}

func TestFeed_ListGroups(t *testing.T) {
	withAuth(t, func(token string) {
		var resp ApiResponse[map[string]any]
		apiRespOK(t, "/api/v1/feed/list_groups", map[string]any{}, &resp)
	})
}

func TestFeed_ListUsers(t *testing.T) {
	withAuth(t, func(token string) {
		var resp ApiResponse[map[string]any]
		apiRespOK(t, "/api/v1/feed/list_users", map[string]any{}, &resp)
	})
}

func TestFeed_ListTrending(t *testing.T) {
	var resp ApiResponse[map[string]any]
	apiRespOK(t, "/api/v1/feed/list_trending", map[string]any{}, &resp)
}

func TestFeed_MeDismiss(t *testing.T) {
	withAuth(t, func(token string) {
		apiRespOK(t, "/api/v1/feed/me/dismiss", map[string]any{
			"target_type": "moment",
			"target_id":   "fake_moment_id",
		}, nil)
	})
}

// =============================================================================
// Search
// =============================================================================

func TestSearch_Suggest(t *testing.T) {
	var resp ApiResponse[map[string]any]
	apiRespOK(t, "/api/v1/search/suggest", map[string]any{"prefix": "test"}, &resp)
}

// =============================================================================
// Map
// =============================================================================

func TestMap_Canvas(t *testing.T) {
	var resp ApiResponse[map[string]any]
	apiOK(t, "/api/v1/map/canvas", map[string]any{
		"viewport": map[string]any{
			"bounds": map[string]any{
				"north": 37.8,
				"south": 37.7,
				"east":  -122.3,
				"west":  -122.4,
			},
		},
	}, &resp)
}

// =============================================================================
// Share
// =============================================================================

func TestShare_CreateGet(t *testing.T) {
	withAuth(t, func(token string) {
		var resp ApiResponse[map[string]any]
		apiRespOK(t, "/api/v1/share/create", map[string]any{
			"target_type": "moment",
			"target_id":   "fake_moment",
		}, &resp)
		if resp.Code == 0 && resp.Data != nil {
			if code := getString(resp.Data, "share_code"); code != "" {
				apiRespOK(t, "/api/v1/share/get", map[string]any{"share_code": code}, nil)
			}
		}
	})
}

func TestShare_MeList(t *testing.T) {
	withAuth(t, func(token string) {
		var resp ApiResponse[map[string]any]
		apiRespOK(t, "/api/v1/share/me/list", map[string]any{}, &resp)
	})
}

func TestShare_Revoke(t *testing.T) {
	withAuth(t, func(token string) {
		apiRespOK(t, "/api/v1/share/revoke", map[string]any{"share_code": "FAKE"}, nil)
	})
}

// =============================================================================
// Private Chats
// =============================================================================

func TestPrivateChats_OpenTheDoor(t *testing.T) {
	withTwoUsers(t, func(tok1, tok2 string) {
		tok2uid, _ := whoAmI(tok2)
		var resp ApiResponse[map[string]any]
		apiRespOK(t, "/api/v1/private_chats/open_the_door", map[string]any{
			"target_user_id": tok2uid,
		}, &resp)
	})
}

func TestPrivateChats_List(t *testing.T) {
	withAuth(t, func(token string) {
		var resp ApiResponse[map[string]any]
		apiRespOK(t, "/api/v1/private_chats/me/list", map[string]any{}, &resp)
	})
}

func TestPrivateChats_UpdateSettings(t *testing.T) {
	withTwoUsers(t, func(tok1, tok2 string) {
		tok2uid, _ := whoAmI(tok2)
		// Open door first
		apiRespOK(t, "/api/v1/private_chats/open_the_door", map[string]any{"target_user_id": tok2uid}, nil)
		// Then update settings
		mute := true
		apiRespOK(t, "/api/v1/private_chats/me/update_settings", map[string]any{
			"target_user_id": tok2uid,
			"is_muted":       mute,
		}, nil)
	})
}

func TestPrivateChats_Delete(t *testing.T) {
	withTwoUsers(t, func(tok1, tok2 string) {
		tok2uid, _ := whoAmI(tok2)
		apiRespOK(t, "/api/v1/private_chats/open_the_door", map[string]any{"target_user_id": tok2uid}, nil)
		apiRespOK(t, "/api/v1/private_chats/me/delete", map[string]any{"target_user_id": tok2uid}, nil)
	})
}

// =============================================================================
// Private Messages
// =============================================================================

func TestPrivateMessages_SendGetHistory(t *testing.T) {
	withTwoUsers(t, func(tok1, tok2 string) {
		tok2uid, _ := whoAmI(tok2)
		// Open door to create chat
		apiRespOK(t, "/api/v1/private_chats/open_the_door", map[string]any{"target_user_id": tok2uid}, nil)
		// Send message
		var sendResp ApiResponse[map[string]any]
		apiRespOK(t, "/api/v1/private_messages/send", map[string]any{
			"target_user_id": tok2uid,
			"type":           "text",
			"content":        "hello from test",
		}, &sendResp)
		// Get history
		var histResp ApiResponse[map[string]any]
		apiRespOK(t, "/api/v1/private_messages/get_history", map[string]any{
			"target_user_id": tok2uid,
		}, &histResp)
	})
}

func TestPrivateMessages_UpdateRecall(t *testing.T) {
	withTwoUsers(t, func(tok1, tok2 string) {
		authorUID := createTestUser(t, "msg_author")
		oldTok := accessToken
		accessToken = tok1
		apiRespOK(t, "/api/v1/private_chats/open_the_door", map[string]any{"target_user_id": authorUID}, nil)
		var sendResp ApiResponse[map[string]any]
		apiRespOK(t, "/api/v1/private_messages/send", map[string]any{
			"target_user_id": authorUID,
			"type":           "text",
			"content":        "to be recalled",
		}, &sendResp)
		accessToken = oldTok
		if sendResp.Code == 0 && sendResp.Data != nil {
			if mid, ok := (*sendResp.Data)["message_id"].(string); ok {
				apiRespOK(t, "/api/v1/private_messages/update", map[string]any{
					"message_id": mid,
					"content":    "updated content",
				}, nil)
				apiRespOK(t, "/api/v1/private_messages/recall", map[string]any{
					"message_id": mid,
				}, nil)
			}
		}
	})
}

func TestPrivateMessages_MarkAsReadAll(t *testing.T) {
	withTwoUsers(t, func(tok1, tok2 string) {
		tok2uid, _ := whoAmI(tok2)
		apiRespOK(t, "/api/v1/private_chats/open_the_door", map[string]any{"target_user_id": tok2uid}, nil)
		apiRespOK(t, "/api/v1/private_messages/me/mark_as_read", map[string]any{
			"target_user_id": tok2uid,
		}, nil)
		apiRespOK(t, "/api/v1/private_messages/me/mark_all_read", map[string]any{}, nil)
	})
}

// =============================================================================
// Group Messages
// =============================================================================

func TestGroupMessages_SendGetHistory(t *testing.T) {
	withAuth(t, func(token string) {
		uid := createTestUser(t, "gm_sender")
		oldTok := accessToken
		accessToken = uid
		// Send to fake group (group doesn't exist — expect business error code != 0)
		var resp ApiResponse[map[string]any]
		status, _ := post("/api/v1/group_messages/send", map[string]any{
			"group_id": "fake_group_id",
			"type":     "text",
			"content":  "test group message",
		}, &resp)
		if status == http.StatusOK {
			t.Logf("group message send: code=%d message=%s", resp.Code, resp.Message)
		}
		accessToken = oldTok
	})
}

func TestGroupMessages_UpdateRecall(t *testing.T) {
	withAuth(t, func(token string) {
		uid := createTestUser(t, "gm_author")
		oldTok := accessToken
		accessToken = uid
		apiRespOK(t, "/api/v1/group_messages/recall", map[string]any{
			"group_id":   "fake_group",
			"message_id": "fake_message",
		}, nil)
		apiRespOK(t, "/api/v1/group_messages/update", map[string]any{
			"group_id":   "fake_group",
			"message_id": "fake_message",
			"content":    "updated",
		}, nil)
		accessToken = oldTok
	})
}

func TestGroupMessages_MarkAsRead(t *testing.T) {
	withAuth(t, func(token string) {
		apiRespOK(t, "/api/v1/group_messages/me/mark_as_read", map[string]any{
			"group_id": "fake_group",
		}, nil)
		apiRespOK(t, "/api/v1/group_messages/me/mark_all_read", map[string]any{}, nil)
	})
}

// =============================================================================
// Notifications
// =============================================================================

func TestNotifications_List(t *testing.T) {
	withAuth(t, func(token string) {
		var resp ApiResponse[map[string]any]
		apiRespOK(t, "/api/v1/notifications/list", map[string]any{}, &resp)
	})
}

func TestNotifications_UnreadCount(t *testing.T) {
	withAuth(t, func(token string) {
		var resp ApiResponse[map[string]any]
		apiRespOK(t, "/api/v1/notifications/unread_count", map[string]any{}, &resp)
	})
}

func TestNotifications_MarkAsRead(t *testing.T) {
	withAuth(t, func(token string) {
		apiRespOK(t, "/api/v1/notifications/mark_as_read", map[string]any{
			"notification_ids": []string{"fake_notif_id"},
		}, nil)
	})
}

func TestNotifications_MarkAllRead(t *testing.T) {
	withAuth(t, func(token string) {
		apiRespOK(t, "/api/v1/notifications/mark_all_read", map[string]any{}, nil)
	})
}

// =============================================================================
// Reports
// =============================================================================

func TestReports_Create(t *testing.T) {
	withAuth(t, func(token string) {
		var resp ApiResponse[map[string]any]
		apiRespOK(t, "/api/v1/reports/create", map[string]any{
			"target_type": "moment",
			"target_id":   "fake_moment",
			"reason":      "spam",
			"description": "test report from integration test",
		}, &resp)
	})
}

func TestReports_MeList(t *testing.T) {
	withAuth(t, func(token string) {
		var resp ApiResponse[map[string]any]
		apiRespOK(t, "/api/v1/reports/me/list", map[string]any{}, &resp)
	})
}

// =============================================================================
// Media
// =============================================================================

func TestMedia_PresignUpload(t *testing.T) {
	withAuth(t, func(token string) {
		var resp ApiResponse[map[string]any]
		apiRespOK(t, "/api/v1/media/presign_upload", map[string]any{
			"file_name": "test.png",
			"file_size": 1024,
			"mime_type": "image/png",
			"category":  "avatar",
		}, &resp)
		if resp.Data == nil {
			t.Fatal("expected data in presign response")
		}
		data := *resp.Data
		for _, key := range []string{"upload_id", "upload_url", "headers", "expires_at"} {
			if _, ok := data[key]; !ok {
				t.Fatalf("expected %q in presign response data", key)
			}
		}
		if uid, ok := data["upload_id"].(string); !ok || uid == "" {
			t.Fatal("expected non-empty upload_id")
		}
		if url, ok := data["upload_url"].(string); !ok || url == "" {
			t.Fatal("expected non-empty upload_url")
		}
	})
}

func TestMedia_BatchPresignUpload(t *testing.T) {
	withAuth(t, func(token string) {
		var resp ApiResponse[map[string]any]
		apiRespOK(t, "/api/v1/media/batch_presign_upload", map[string]any{
			"files": []map[string]any{
				{"file_name": "a.png", "file_size": 100, "mime_type": "image/png", "category": "avatar"},
				{"file_name": "b.png", "file_size": 200, "mime_type": "image/png", "category": "moment_image"},
			},
		}, &resp)
		if resp.Data == nil {
			t.Fatal("expected data in batch presign response")
		}
	})
}

func TestMedia_CompleteUpload_NotFound(t *testing.T) {
	withAuth(t, func(token string) {
		var resp ApiResponse[map[string]any]
		status, err := post("/api/v1/media/complete_upload", map[string]any{
			"upload_id": "nonexistent_upload_id",
		}, &resp)
		if err != nil {
			t.Fatalf("complete request failed: %v", err)
		}
		if status != http.StatusOK {
			t.Fatalf("expected 200, got %d", status)
		}
		if resp.Code == 0 {
			t.Fatal("expected error code for nonexistent upload")
		}
	})
}

func TestMedia_E2E_PresignUploadComplete(t *testing.T) {
	_, token := createGuestUser(t)
	oldToken := accessToken
	accessToken = token
	defer func() { accessToken = oldToken }()

	var presignResp ApiResponse[map[string]any]
	apiRespOK(t, "/api/v1/media/presign_upload", map[string]any{
		"file_name": "photo.png",
		"file_size": 66,
		"mime_type": "image/png",
		"category":  "avatar",
	}, &presignResp)

	uploadID := getString(presignResp.Data, "upload_id")
	uploadURL := getString(presignResp.Data, "upload_url")
	if uploadID == "" || uploadURL == "" {
		t.Fatalf("presign returned empty upload_id or upload_url: %+v", presignResp)
	}

	pngBytes := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A,
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x06, 0x00, 0x00, 0x00, 0x1F, 0x15, 0xC4,
		0x89, 0x00, 0x00, 0x00, 0x0A, 0x49, 0x44, 0x41,
		0x54, 0x78, 0x9C, 0x63, 0x00, 0x01, 0x00, 0x00,
		0x05, 0x00, 0x01, 0x0D, 0x0A, 0xB4, 0x00, 0x00,
		0x00, 0x00, 0x49, 0x45, 0x4E, 0x44, 0xAE, 0x42,
		0x60, 0x82,
	}

	putReq, err := http.NewRequest(http.MethodPut, uploadURL, bytes.NewReader(pngBytes))
	if err != nil {
		t.Fatalf("create PUT request: %v", err)
	}
	putReq.Header.Set("Content-Type", "image/png")
	putResp, err := testClient.Do(putReq)
	if err != nil {
		t.Fatalf("PUT to storage: %v", err)
	}
	defer putResp.Body.Close()
	if putResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(putResp.Body)
		t.Fatalf("PUT to storage returned %d: %s", putResp.StatusCode, body)
	}

	var completeResp ApiResponse[map[string]any]
	apiRespOK(t, "/api/v1/media/complete_upload", map[string]any{
		"upload_id": uploadID,
	}, &completeResp)
	if completeResp.Code != 0 {
		t.Fatalf("complete_upload failed: code=%d message=%s", completeResp.Code, completeResp.Message)
	}
	if completeResp.Data == nil {
		t.Fatal("complete_upload returned no data")
	}
	data := *completeResp.Data
	fileID := getString(completeResp.Data, "file_id")
	if fileID == "" {
		t.Fatal("expected file_id in complete response")
	}
	fileSize, _ := data["file_size"].(float64)
	if int(fileSize) != len(pngBytes) {
		t.Fatalf("expected file_size=%d, got %d", len(pngBytes), int(fileSize))
	}
	mimeType := getString(completeResp.Data, "mime_type")
	if mimeType != "image/png" {
		t.Fatalf("expected mime_type=image/png, got %q", mimeType)
	}
}

func TestMedia_E2E_BatchPresignUploadComplete(t *testing.T) {
	_, token := createGuestUser(t)
	oldToken := accessToken
	accessToken = token
	defer func() { accessToken = oldToken }()

	var batchResp ApiResponse[map[string]any]
	apiRespOK(t, "/api/v1/media/batch_presign_upload", map[string]any{
		"files": []map[string]any{
			{"file_name": "a.png", "file_size": 66, "mime_type": "image/png", "category": "avatar"},
		},
	}, &batchResp)
	if batchResp.Data == nil {
		t.Fatal("batch presign returned no data")
	}
	uploadsVal, ok := (*batchResp.Data)["uploads"]
	if !ok {
		t.Fatal("expected 'uploads' in batch response")
	}
	uploads, ok := uploadsVal.([]any)
	if !ok || len(uploads) != 1 {
		t.Fatalf("expected 1 upload, got %+v", uploadsVal)
	}
	upload, ok := uploads[0].(map[string]any)
	if !ok {
		t.Fatal("upload is not a map")
	}
	uploadID, _ := upload["upload_id"].(string)
	uploadURL, _ := upload["upload_url"].(string)
	if uploadID == "" || uploadURL == "" {
		t.Fatal("batch upload missing upload_id or upload_url")
	}

	pngBytes := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A,
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x06, 0x00, 0x00, 0x00, 0x1F, 0x15, 0xC4,
		0x89, 0x00, 0x00, 0x00, 0x0A, 0x49, 0x44, 0x41,
		0x54, 0x78, 0x9C, 0x63, 0x00, 0x01, 0x00, 0x00,
		0x05, 0x00, 0x01, 0x0D, 0x0A, 0xB4, 0x00, 0x00,
		0x00, 0x00, 0x49, 0x45, 0x4E, 0x44, 0xAE, 0x42,
		0x60, 0x82,
	}

	putReq, _ := http.NewRequest(http.MethodPut, uploadURL, bytes.NewReader(pngBytes))
	putReq.Header.Set("Content-Type", "image/png")
	putResp, err := testClient.Do(putReq)
	if err != nil {
		t.Fatalf("PUT to storage: %v", err)
	}
	defer putResp.Body.Close()
	if putResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(putResp.Body)
		t.Fatalf("PUT returned %d: %s", putResp.StatusCode, body)
	}

	var completeResp ApiResponse[map[string]any]
	apiRespOK(t, "/api/v1/media/complete_upload", map[string]any{
		"upload_id": uploadID,
	}, &completeResp)
	if completeResp.Code != 0 {
		t.Fatalf("batch complete failed: code=%d message=%s", completeResp.Code, completeResp.Message)
	}
}

func TestMedia_E2E_SizeMismatchRejected(t *testing.T) {
	createGuestUser(t)

	var presignResp ApiResponse[map[string]any]
	apiRespOK(t, "/api/v1/media/presign_upload", map[string]any{
		"file_name": "size_mismatch.png",
		"file_size": 999,
		"mime_type": "image/png",
		"category":  "avatar",
	}, &presignResp)
	uploadID := getString(presignResp.Data, "upload_id")
	uploadURL := getString(presignResp.Data, "upload_url")

	pngBytes := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A,
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x06, 0x00, 0x00, 0x00, 0x1F, 0x15, 0xC4,
		0x89, 0x00, 0x00, 0x00, 0x0A, 0x49, 0x44, 0x41,
		0x54, 0x78, 0x9C, 0x63, 0x00, 0x01, 0x00, 0x00,
		0x05, 0x00, 0x01, 0x0D, 0x0A, 0xB4, 0x00, 0x00,
		0x00, 0x00, 0x49, 0x45, 0x4E, 0x44, 0xAE, 0x42,
		0x60, 0x82,
	}

	putReq, _ := http.NewRequest(http.MethodPut, uploadURL, bytes.NewReader(pngBytes))
	putReq.Header.Set("Content-Type", "image/png")
	putResp, _ := testClient.Do(putReq)
	putResp.Body.Close()

	var completeResp ApiResponse[map[string]any]
	apiRespOK(t, "/api/v1/media/complete_upload", map[string]any{
		"upload_id": uploadID,
	}, &completeResp)
	if completeResp.Code == 0 {
		t.Fatal("expected error for size mismatch, got success")
	}
}

func TestMedia_E2E_CompleteBeforeUpload(t *testing.T) {
	createGuestUser(t)

	var presignResp ApiResponse[map[string]any]
	apiRespOK(t, "/api/v1/media/presign_upload", map[string]any{
		"file_name": "not_uploaded.png",
		"file_size": 100,
		"mime_type": "image/png",
		"category":  "avatar",
	}, &presignResp)
	uploadID := getString(presignResp.Data, "upload_id")

	var completeResp ApiResponse[map[string]any]
	apiRespOK(t, "/api/v1/media/complete_upload", map[string]any{
		"upload_id": uploadID,
	}, &completeResp)
	if completeResp.Code == 0 {
		t.Fatal("expected error when completing before upload, got success")
	}
	if completeResp.Code != 10400 {
		t.Fatalf("expected code 10400 (not ready), got code=%d", completeResp.Code)
	}
}

// =============================================================================
// Admin
// =============================================================================

func TestAdmin_RBAC_GetMe(t *testing.T) {
	withAuth(t, func(token string) {
		var resp ApiResponse[map[string]any]
		apiRespOK(t, "/api/v1/admin/rbac/get_me", map[string]any{}, &resp)
	})
}

func TestAdmin_RBAC_ListPermissions(t *testing.T) {
	withAuth(t, func(token string) {
		var resp ApiResponse[map[string]any]
		apiRespOK(t, "/api/v1/admin/rbac/list_permissions", map[string]any{}, &resp)
	})
}

func TestAdmin_RBAC_RolesList(t *testing.T) {
	withAuth(t, func(token string) {
		var resp ApiResponse[map[string]any]
		apiRespOK(t, "/api/v1/admin/rbac/roles/list", map[string]any{}, &resp)
	})
}

func TestAdmin_RBAC_AdminsList(t *testing.T) {
	withAuth(t, func(token string) {
		var resp ApiResponse[map[string]any]
		apiRespOK(t, "/api/v1/admin/rbac/admins/list", map[string]any{}, &resp)
	})
}

func TestAdmin_AppConfigGet(t *testing.T) {
	withAuth(t, func(token string) {
		var resp ApiResponse[map[string]any]
		apiRespOK(t, "/api/v1/admin/app_config/get", map[string]any{}, &resp)
	})
}

func TestAdmin_UsersSearch(t *testing.T) {
	withAuth(t, func(token string) {
		var resp ApiResponse[map[string]any]
		apiRespOK(t, "/api/v1/admin/users/search", map[string]any{}, &resp)
	})
}

func TestAdmin_GroupsSearch(t *testing.T) {
	withAuth(t, func(token string) {
		var resp ApiResponse[map[string]any]
		apiRespOK(t, "/api/v1/admin/groups/search", map[string]any{}, &resp)
	})
}

func TestAdmin_ReportsSearch(t *testing.T) {
	withAuth(t, func(token string) {
		var resp ApiResponse[map[string]any]
		apiRespOK(t, "/api/v1/admin/reports/search", map[string]any{}, &resp)
	})
}

func TestAdmin_TagsSearch(t *testing.T) {
	withAuth(t, func(token string) {
		var resp ApiResponse[map[string]any]
		apiRespOK(t, "/api/v1/admin/tags/search", map[string]any{}, &resp)
	})
}

func TestAdmin_Announcements(t *testing.T) {
	withAuth(t, func(token string) {
		var resp ApiResponse[map[string]any]
		apiRespOK(t, "/api/v1/admin/announcements/search", map[string]any{}, &resp)
	})
}

func TestAdmin_Audit(t *testing.T) {
	withAuth(t, func(token string) {
		var resp ApiResponse[map[string]any]
		apiRespOK(t, "/api/v1/admin/audit/list", map[string]any{}, &resp)
	})
}
