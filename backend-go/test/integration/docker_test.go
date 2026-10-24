package integration

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"os"
	"reflect"
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
	testClient   *http.Client
	accessToken  string
	backendAddr  string
	backendReady bool
	idempKeySeq  int64
	idempKeyMu   sync.Mutex
)

type ApiResponse[T any] struct {
	Code    int    `json:"code"`
	Message string `json:"message,omitempty"`
	Data    *T     `json:"data,omitempty"`
}

// responseCode 使 *ApiResponse[T] 满足 codeGetter（值接收者 → 指针方法集）。
func (r ApiResponse[T]) responseCode() int { return r.Code }

func TestMain(m *testing.M) {
	// 必须显式指定 BACKEND_URL 才会执行集成测试：默认地址可能指向共享的
	// 主实例后端（8081），裸跑 go test ./... 会污染其数据库。
	backendAddr = os.Getenv("BACKEND_URL")
	if backendAddr == "" {
		fmt.Fprintf(os.Stderr, "BACKEND_URL not set; skipping integration tests\n")
		os.Exit(m.Run())
	}

	maybeMigrate()
	if shouldWaitForBackend() {
		if err := waitForBackendWithPost(backendAddr+"/public/v1/system/health", 60*time.Second); err != nil {
			fmt.Fprintf(os.Stderr, "backend not ready, skipping backend tests: %v\n", err)
		} else {
			backendReady = true
			jar, _ := cookiejar.New(nil)
			testClient = &http.Client{
				Jar: jar,
				CheckRedirect: func(*http.Request, []*http.Request) error {
					return http.ErrUseLastResponse
				},
				Timeout: 15 * time.Second,
			}
		}
	}

	code := m.Run()
	os.Exit(code)
}

func shouldWaitForBackend() bool {
	for _, arg := range os.Args {
		if arg == "-short" || strings.HasPrefix(arg, "-test.short") {
			return false
		}
	}
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
	return reqWithHost("", method, path, body, result)
}

// reqWithHost 支持覆写 Host 头（media presign 直传 URL 按请求 Host 签名——
// 集成环境用它对齐 /s3/ 网关域，保证 PUT 验签通过）。host 为空=默认 backendAddr。
func reqWithHost(host, method, path string, body, result any) (int, error) {
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
	if host != "" {
		req.Host = host
		// 媒体直传 URL 的入口 origin 由后端从 Origin/Referer 提取（全局中间件
		// media.ClientEndpointMiddleware，见 internal/modules/media/client_endpoint.go）；
		// 只设 Host 会回退静态 S3_PUBLIC_ENDPOINT，故两者必须一起给。
		req.Header.Set("Origin", "http://"+host)
	}
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

func requireBackend(t *testing.T) {
	t.Helper()
	if !backendReady {
		t.Skip("backend not available")
	}
}

// codeGetter 统一提取任意 ApiResponse[T] 的业务码——避免对具体泛型类型做
// type assertion（*ApiResponse[map[string]any] 无法断言为 *ApiResponse[any]，
// 曾导致 apiRespOK 的 code==0 校验对所有调用静默失效）。
type codeGetter interface {
	responseCode() int
}

func responseCodeOf(result any) (int, bool) {
	if result == nil {
		return 0, false
	}
	switch r := result.(type) {
	case codeGetter:
		// 所有 *ApiResponse[T] 均实现 codeGetter（值接收者方法提升到指针方法集）。
		return r.responseCode(), true
	}
	// 兜底：反射读取 Code 字段（覆盖其余元素类型的 ApiResponse[T]）。
	v := reflect.ValueOf(result)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() == reflect.Struct && v.Type().Name() == "ApiResponse" {
		if f := v.FieldByName("Code"); f.IsValid() && f.CanInt() {
			return int(f.Int()), true
		}
	}
	return 0, false
}

// responseMessageOf 提取业务错误信息用于失败诊断（与 responseCodeOf 同样的兜底策略）。
func responseMessageOf(result any) string {
	switch r := result.(type) {
	case *ApiResponse[any]:
		return r.Message
	case *ApiResponse[map[string]any]:
		return r.Message
	}
	v := reflect.ValueOf(result)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() == reflect.Struct && v.Type().Name() == "ApiResponse" {
		if f := v.FieldByName("Message"); f.IsValid() && f.Kind() == reflect.String {
			return f.String()
		}
	}
	return ""
}

func apiRespOK(t *testing.T, path string, body, result any) {
	requireBackend(t)
	status, err := post(path, body, result)
	if err != nil {
		t.Fatalf("POST %s: %v", path, err)
	}
	if status != http.StatusOK {
		t.Fatalf("POST %s: got status %d, want 200", path, status)
	}
	if code, ok := responseCodeOf(result); ok && code != 0 {
		t.Fatalf("POST %s: code=%d message=%s", path, code, responseMessageOf(result))
	}
}

// apiRespOKWithHost 以指定 Host 头发起 POST（media presign 直传 URL 按请求
// Host 签名——集成环境用它对齐 /s3/ 网关域，保证 PUT 验签通过）。其余同 apiRespOK。
func apiRespOKWithHost(t *testing.T, host, path string, body, result any) {
	requireBackend(t)
	status, err := reqWithHost(host, "POST", path, body, result)
	if err != nil {
		t.Fatalf("POST %s: %v", path, err)
	}
	if status != http.StatusOK {
		t.Fatalf("POST %s: got status %d, want 200", path, status)
	}
	if code, ok := responseCodeOf(result); ok && code != 0 {
		t.Fatalf("POST %s: code=%d message=%s", path, code, responseMessageOf(result))
	}
}

// mediaPresignClientOrigin 是媒体 E2E 声明的「客户端入口 origin」：presign 返回的
// 直传 URL 必须与该 origin 同源（SigV4 签名含 Host，浏览器跨源 PUT 也会被 CORS 拦）。
// 端口跟随 NGINX_PORT（默认 8888）：重映射端口起 stack 时（NGINX_PORT=8889 docker
// compose up …）用同名变量跑测试，presign/PUT/complete 才会全部落在同一套
// seaweedfs——origin 固定 8888 会把 PUT 打进默认端口那套栈，complete_upload
// 在本栈查不到对象而报 10400（2026-10-03 worktree 实测）。
var mediaPresignClientOrigin = func() string {
	port := os.Getenv("NGINX_PORT")
	if port == "" {
		port = "8888"
	}
	return "127.0.0.1:" + port
}()

func apiRespOKWithCode(t *testing.T, path string, body, result any, expectCode int) {
	requireBackend(t)
	status, err := post(path, body, result)
	if err != nil {
		t.Fatalf("POST %s: %v", path, err)
	}
	if status != http.StatusOK {
		t.Fatalf("POST %s: got status %d, want %d", path, status, http.StatusOK)
	}
	if code, ok := responseCodeOf(result); ok && code != expectCode {
		t.Fatalf("POST %s: code=%d, want %d", path, code, expectCode)
	}
}

func apiOK(t *testing.T, path string, body, result any) any {
	requireBackend(t)
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
	requireBackend(t)
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
	requireBackend(t)
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

// createTestUser 的 login_id 必须符合 spec api_spec_base.md「通用字段约束」
// （3-64 字符）；"test_"+t.Name() 拼接可能超长（如 TestGroupMessages_SendGetHistory），
// 故对测试名取短哈希保证唯一且合规。
func createTestUser(t *testing.T, nickname string) (userID string) {
	uid, _ := createTestUserWithToken(t, nickname)
	return uid
}

// createTestUserWithToken 注册测试用户并同时返回 user_id 与 access_token，
// 供需要「以该用户身份」发请求的用例切换 accessToken 使用。
func createTestUserWithToken(t *testing.T, nickname string) (userID, token string) {
	var resp ApiResponse[map[string]any]
	nameSum := sha256.Sum256([]byte(t.Name()))
	apiRespOK(t, "/public/v1/auth/register", map[string]any{
		"login_id":    fmt.Sprintf("t_%x_%s_%d", nameSum[:5], nickname, time.Now().UnixNano()),
		"nickname":    nickname,
		"password":    "Test1234",
		"client_type": "app",
		"device_id":   uniqueDeviceID(t),
	}, &resp)
	if resp.Data == nil {
		t.Fatalf("no data in register response")
	}
	if uid, ok := (*resp.Data)["user_id"].(string); ok {
		userID = uid
	}
	if tok, ok := (*resp.Data)["access_token"].(string); ok {
		token = tok
	}
	return userID, token
}

// withAdminAuth 以「系统管理员」身份执行 fn：注册普通用户后向 admin_users
// 表播种 grant_all_permissions 记录（spec api_spec_admin.md 管理员判定来源），
// 结束后清理。无 POSTGRES_DSN 时跳过（无法播种）。
func withAdminAuth(t *testing.T, fn func(token string)) {
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_DSN not set: cannot seed admin user")
	}
	userID, token := createGuestUser(t)
	db := persistence.NewPostgresDB(dsn)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := db.Exec(ctx,
		`INSERT INTO admin_users (user_id, grant_all_permissions) VALUES ($1, TRUE)
		 ON CONFLICT (user_id) DO UPDATE SET revoked_at = NULL, grant_all_permissions = TRUE`,
		userID); err != nil {
		t.Fatalf("seed admin user: %v", err)
	}
	defer func() {
		delCtx, delCancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer delCancel()
		_ = db.Exec(delCtx, `DELETE FROM admin_users WHERE user_id = $1`, userID)
	}()
	oldTok := accessToken
	accessToken = token
	defer func() { accessToken = oldTok }()
	fn(token)
}

// createMomentAsCurrentUser 以当前 accessToken 创建一条瞬间并返回 moment_id。
func createMomentAsCurrentUser(t *testing.T) string {
	var createResp ApiResponse[map[string]any]
	apiRespOK(t, "/api/v1/moments/create", map[string]any{
		"description": "integration test moment",
		"location": map[string]any{
			"point":         map[string]any{"latitude": 37.7749, "longitude": -122.4194},
			"location_name": "Test City",
		},
	}, &createResp)
	if createResp.Data == nil {
		t.Fatalf("create moment: no data")
	}
	mid, _ := (*createResp.Data)["moment_id"].(string)
	if mid == "" {
		t.Fatalf("create moment: missing moment_id")
	}
	return mid
}

func createGuestUser(t *testing.T) (userID string, token string) {
	var resp ApiResponse[map[string]any]
	// 昵称上限 30 字符（spec api_spec_auth.md）：t.Name() 可能超长，用短哈希代替
	nameSum := sha256.Sum256([]byte(t.Name()))
	apiNoAuth(t, "/public/v1/auth/guest", map[string]any{
		"nickname":    fmt.Sprintf("gst_%x", nameSum[:5]),
		"client_type": "app",
		"device_id":   uniqueDeviceID(t),
	}, &resp)
	if resp.Code != 0 {
		t.Fatalf("guest register failed: code=%d message=%s", resp.Code, resp.Message)
	}
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
	apiNoAuth(t, "/public/v1/auth/signin", map[string]any{
		"login_id":    loginID,
		"password":    password,
		"client_type": "app",
		"device_id":   uniqueDeviceID(t),
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

// uniqueDeviceID returns a per-test device id so concurrent/sequential tests
// never share a session (spec: (user_id, device_id) 唯一会话，同设备新会话会驱逐旧会话)。
func uniqueDeviceID(t *testing.T) string {
	return fmt.Sprintf("test-device-%s-%d", t.Name(), time.Now().UnixNano())
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
	apiOK(t, "/public/v1/system/health", map[string]any{}, &resp)
}

func TestSystem_GetMeta(t *testing.T) {
	var resp ApiResponse[map[string]any]
	apiOK(t, "/public/v1/system/meta", map[string]any{}, &resp)
}

func TestSystem_GetClientConfig(t *testing.T) {
	var resp ApiResponse[map[string]any]
	apiOK(t, "/public/v1/system/client_config", map[string]any{}, &resp)
}

func TestSystem_ListAnnouncements(t *testing.T) {
	var resp ApiResponse[map[string]any]
	apiOK(t, "/public/v1/system/announcements/fetch", map[string]any{}, &resp)
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
	apiOK(t, "/public/v1/auth/signin", map[string]any{
		"login_id":    "nonexistent_user_xyz",
		"password":    "wrongpass",
		"client_type": "app",
		"device_id":   uniqueDeviceID(t),
	}, &resp)
	if resp.Code != 0 {
		t.Logf("guest response code=%d message=%s", resp.Code, resp.Message)
	}
}

func TestAuth_CheckStatus_Anonymous(t *testing.T) {
	var resp ApiResponse[map[string]any]
	apiOK(t, "/api/v1/auth/session", map[string]any{}, &resp)
	if resp.Data != nil {
		if valid, ok := (*resp.Data)["is_valid"].(bool); ok && valid {
			t.Log("anonymous get_status returned is_valid=true — unexpected but acceptable if CSRF or session-based")
		}
	}
}

func TestAuth_RefreshToken_NoToken(t *testing.T) {
	requireBackend(t)
	var resp ApiResponse[map[string]any]
	status, _ := post("/public/v1/auth/refresh_token", map[string]any{"client_type": "app"}, &resp)
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
	// users/search 需登录（spec api_spec_discovery.md 认证档位）
	withAuth(t, func(token string) {
		var resp ApiResponse[map[string]any]
		apiRespOK(t, "/api/v1/users/search", map[string]any{"keyword": "test"}, &resp)
	})
}

func TestUsers_OpenTheDoor(t *testing.T) {
	withAuth(t, func(token string) {
		uid := createTestUser(t, "target_user")
		var resp ApiResponse[map[string]any]
		apiRespOK(t, "/api/v1/users/open_the_door", map[string]any{"user_id": uid}, &resp)
	})
}

func TestUsers_ListFriends(t *testing.T) {
	// spec FetchFriendsRequest：user_id 必传
	withAuth(t, func(token string) {
		uid, _ := createGuestUser(t)
		var resp ApiResponse[map[string]any]
		apiRespOK(t, "/api/v1/users/fetch_friends", map[string]any{"user_id": uid}, &resp)
	})
}

func TestUsers_ListFollowing(t *testing.T) {
	withAuth(t, func(token string) {
		uid, _ := createGuestUser(t)
		var resp ApiResponse[map[string]any]
		apiRespOK(t, "/api/v1/users/fetch_following", map[string]any{"user_id": uid}, &resp)
	})
}

func TestUsers_ListFollowers(t *testing.T) {
	withAuth(t, func(token string) {
		uid, _ := createGuestUser(t)
		var resp ApiResponse[map[string]any]
		apiRespOK(t, "/api/v1/users/fetch_followers", map[string]any{"user_id": uid}, &resp)
	})
}

func TestUsers_ListGroups(t *testing.T) {
	withAuth(t, func(token string) {
		uid, _ := createGuestUser(t)
		var resp ApiResponse[map[string]any]
		apiRespOK(t, "/api/v1/users/fetch_groups", map[string]any{"user_id": uid}, &resp)
	})
}

func TestUsers_ListMoments(t *testing.T) {
	withAuth(t, func(token string) {
		uid := createTestUser(t, "moments_owner")
		var resp ApiResponse[map[string]any]
		apiRespOK(t, "/api/v1/users/fetch_moments", map[string]any{"user_id": uid}, &resp)
	})
}

func TestUsers_ListReactedMoments(t *testing.T) {
	withAuth(t, func(token string) {
		uid, _ := createGuestUser(t)
		var resp ApiResponse[map[string]any]
		apiRespOK(t, "/api/v1/users/fetch_reacted_moments", map[string]any{"user_id": uid}, &resp)
	})
}

// =============================================================================
// Users (authenticated)
// =============================================================================

func TestUsersMe_GetProfile(t *testing.T) {
	withAuth(t, func(token string) {
		var resp ApiResponse[map[string]any]
		apiRespOK(t, "/api/v1/users/account", map[string]any{}, &resp)
	})
}

func TestUsersMe_UpdateProfile(t *testing.T) {
	withAuth(t, func(token string) {
		apiRespOK(t, "/api/v1/users/update_account", map[string]any{
			"bio": "test bio",
		}, nil)
	})
}

func TestUsersMe_LocationUpdate(t *testing.T) {
	withAuth(t, func(token string) {
		apiRespOK(t, "/api/v1/users/update_location", map[string]any{
			"location": map[string]any{
				"point": map[string]any{
					"latitude":  37.7749,
					"longitude": -122.4194,
				},
				"location_name": "San Francisco",
			},
		}, nil)
	})
}

func TestUsersMe_GetSettings(t *testing.T) {
	withAuth(t, func(token string) {
		var resp ApiResponse[map[string]any]
		apiRespOK(t, "/api/v1/users/settings", map[string]any{}, &resp)
	})
}

func TestUsersMe_UpdateSettings(t *testing.T) {
	withAuth(t, func(token string) {
		apiRespOK(t, "/api/v1/users/update_settings", map[string]any{
			"settings": map[string]any{
				"allow_private_chat": "everyone",
			},
		}, nil)
	})
}

func TestUsersMe_Follow(t *testing.T) {
	withTwoUsers(t, func(tok1, tok2 string) {
		tok1uid, _ := whoAmI(tok1)
		apiRespOK(t, "/api/v1/users/set_follow", map[string]any{
			"user_id": tok1uid,
			"active":  true,
		}, nil)
	})
}

func TestUsersMe_Unfollow(t *testing.T) {
	// Follow first then unfollow
	_, tok2 := createGuestUser(t)
	tok2uid, _ := whoAmI(tok2)
	_, tok1 := createTestUserWithToken(t, "unfollow_target")
	oldToken := accessToken
	accessToken = tok1
	apiRespOK(t, "/api/v1/users/set_follow", map[string]any{"user_id": tok2uid, "active": true}, nil)
	apiRespOK(t, "/api/v1/users/set_follow", map[string]any{"user_id": tok2uid, "active": false}, nil)
	accessToken = oldToken
}

func TestUsersMe_Block_Unblock(t *testing.T) {
	withTwoUsers(t, func(tok1, tok2 string) {
		tok2uid, _ := whoAmI(tok2)
		apiRespOK(t, "/api/v1/users/set_block", map[string]any{"user_id": tok2uid, "active": true}, nil)
		apiRespOK(t, "/api/v1/users/set_block", map[string]any{"user_id": tok2uid, "active": false}, nil)
	})
}

func TestUsersMe_ListBlocked(t *testing.T) {
	withAuth(t, func(token string) {
		var resp ApiResponse[map[string]any]
		apiRespOK(t, "/api/v1/users/fetch_blocked", map[string]any{}, &resp)
	})
}

func TestUsersMe_SetRemark(t *testing.T) {
	withTwoUsers(t, func(tok1, tok2 string) {
		tok2uid, _ := whoAmI(tok2)
		apiRespOK(t, "/api/v1/users/set_remark", map[string]any{
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
	post("/api/v1/users/account", map[string]any{}, &resp)
	if resp.Data == nil {
		return "", ""
	}
	return getString(resp.Data, "user_id"), token
}

// firstUploadEntry 取 presign/complete 响应 data.uploads[0]（数组语义，spec api_spec_media.md）。
func firstUploadEntry(t *testing.T, m *map[string]any) map[string]any {
	t.Helper()
	if m == nil {
		t.Fatal("no data in response")
	}
	uploadsVal, ok := (*m)["uploads"]
	if !ok {
		t.Fatalf("expected 'uploads' in response data: %+v", *m)
	}
	uploads, ok := uploadsVal.([]any)
	if !ok || len(uploads) == 0 {
		t.Fatalf("expected non-empty uploads, got %+v", uploadsVal)
	}
	upload, ok := uploads[0].(map[string]any)
	if !ok {
		t.Fatal("uploads[0] is not a map")
	}
	return upload
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
		var resp ApiResponse[map[string]any]
		// 不存在的瞬间 → 10300 瞬间不存在（spec api_spec_moment.md 错误码）
		apiRespOKWithCode(t, "/api/v1/moments/open_the_door", map[string]any{"moment_id": "nonexistent"}, &resp, 10300)
	})
}

func TestMoments_Search(t *testing.T) {
	withAuth(t, func(token string) {
		// spec SearchMomentsRequest：keyword/tag 至少其一（全局搜索）
		var resp ApiResponse[map[string]any]
		apiRespOK(t, "/api/v1/moments/search", map[string]any{"keyword": "integration"}, &resp)
	})
}

func TestMoments_ListLikes(t *testing.T) {
	withAuth(t, func(token string) {
		mid := createMomentAsCurrentUser(t)
		var resp ApiResponse[map[string]any]
		apiRespOK(t, "/api/v1/reactions/fetch", map[string]any{"target_type": "moment", "target_id": mid}, &resp)
	})
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
				"location_name": "Test City",
			},
		}, &createResp)
		if createResp.Code != 0 {
			t.Fatalf("create moment failed: code=%d message=%s", createResp.Code, createResp.Message)
		}
		var listResp ApiResponse[map[string]any]
		apiOK(t, "/api/v1/moments/fetch_created", map[string]any{}, &listResp)
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
				"location_name": "Test City",
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
		// 作者身份需用其真实 access_token（此前误把 user_id 塞进 Authorization）
		_, authorTok := createTestUserWithToken(t, "moment_author")
		oldTok := accessToken
		accessToken = authorTok
		var createResp ApiResponse[map[string]any]
		apiRespOK(t, "/api/v1/moments/create", map[string]any{
			"description": "likeable moment",
			"location": map[string]any{
				"point": map[string]any{
					"latitude":  37.7749,
					"longitude": -122.4194,
				},
				"location_name": "Test City",
			},
		}, &createResp)
		accessToken = oldTok

		if createResp.Code == 0 && createResp.Data != nil {
			if mid, ok := (*createResp.Data)["moment_id"].(string); ok {
				apiRespOK(t, "/api/v1/reactions/set", map[string]any{"target_type": "moment", "target_id": mid, "emoji": "👍", "active": true}, nil)
				apiRespOK(t, "/api/v1/reactions/set", map[string]any{"target_type": "moment", "target_id": mid, "emoji": "👍", "active": false}, nil)
			}
		}
	})
}

func TestMoments_FavoriteUnfavorite(t *testing.T) {
	withAuth(t, func(token string) {
		_, authorTok := createTestUserWithToken(t, "fav_author")
		oldTok := accessToken
		accessToken = authorTok
		var createResp ApiResponse[map[string]any]
		apiRespOK(t, "/api/v1/moments/create", map[string]any{
			"description": "fav moment",
			"location": map[string]any{
				"point": map[string]any{
					"latitude":  37.7749,
					"longitude": -122.4194,
				},
				"location_name": "Test City",
			},
		}, &createResp)
		accessToken = oldTok
		if createResp.Code == 0 && createResp.Data != nil {
			if mid, ok := (*createResp.Data)["moment_id"].(string); ok {
				apiRespOK(t, "/api/v1/moments/set_favorite", map[string]any{"moment_id": mid, "active": true}, nil)
				apiRespOK(t, "/api/v1/moments/set_favorite", map[string]any{"moment_id": mid, "active": false}, nil)
			}
		}
	})
}

// =============================================================================
// Comments
// =============================================================================

func TestComments_List(t *testing.T) {
	withAuth(t, func(token string) {
		mid := createMomentAsCurrentUser(t)
		// spec ListCommentsRequest：moment_id 必传
		var resp ApiResponse[map[string]any]
		apiRespOK(t, "/api/v1/comments/fetch", map[string]any{"moment_id": mid}, &resp)
	})
}

func TestComments_ListReplies(t *testing.T) {
	withAuth(t, func(token string) {
		mid := createMomentAsCurrentUser(t)
		var createResp ApiResponse[map[string]any]
		apiRespOK(t, "/api/v1/comments/create", map[string]any{
			"moment_id": mid,
			"content":   "root comment for replies",
		}, &createResp)
		if createResp.Data == nil {
			t.Fatalf("create comment: no data")
		}
		cid, _ := (*createResp.Data)["comment_id"].(string)
		var resp ApiResponse[map[string]any]
		apiRespOK(t, "/api/v1/comments/fetch_replies", map[string]any{"comment_id": cid}, &resp)
	})
}

func TestComments_ListLikes(t *testing.T) {
	withAuth(t, func(token string) {
		mid := createMomentAsCurrentUser(t)
		var createResp ApiResponse[map[string]any]
		apiRespOK(t, "/api/v1/comments/create", map[string]any{
			"moment_id": mid,
			"content":   "comment to be liked",
		}, &createResp)
		if createResp.Data == nil {
			t.Fatalf("create comment: no data")
		}
		cid, _ := (*createResp.Data)["comment_id"].(string)
		var resp ApiResponse[map[string]any]
		apiRespOK(t, "/api/v1/reactions/fetch", map[string]any{"target_type": "comment", "target_id": cid}, &resp)
	})
}

func TestComments_CreateDelete(t *testing.T) {
	withAuth(t, func(token string) {
		mid := createMomentAsCurrentUser(t)
		var resp ApiResponse[map[string]any]
		apiRespOK(t, "/api/v1/comments/create", map[string]any{
			"moment_id": mid,
			"content":   "test comment",
		}, &resp)
		if resp.Data == nil {
			t.Fatalf("create comment: no data")
		}
		if cid, ok := (*resp.Data)["comment_id"].(string); ok {
			apiRespOK(t, "/api/v1/comments/delete", map[string]any{"comment_id": cid}, nil)
		}
	})
}

func TestComments_LikeUnlike(t *testing.T) {
	withAuth(t, func(token string) {
		mid := createMomentAsCurrentUser(t)
		var createResp ApiResponse[map[string]any]
		apiRespOK(t, "/api/v1/comments/create", map[string]any{
			"moment_id": mid,
			"content":   "comment to like",
		}, &createResp)
		if createResp.Data == nil {
			t.Fatalf("create comment: no data")
		}
		cid, _ := (*createResp.Data)["comment_id"].(string)
		apiRespOK(t, "/api/v1/reactions/set", map[string]any{"target_type": "comment", "target_id": cid, "emoji": "👍", "active": true}, nil)
		apiRespOK(t, "/api/v1/reactions/set", map[string]any{"target_type": "comment", "target_id": cid, "emoji": "👍", "active": false}, nil)
	})
}

// =============================================================================
// Groups
// =============================================================================

func TestGroups_Search(t *testing.T) {
	// spec GroupsSearchRequest：keyword/tag/radius 至少其一
	var resp ApiResponse[map[string]any]
	apiRespOK(t, "/api/v1/groups/search", map[string]any{"keyword": "test"}, &resp)
}

func TestGroups_OpenTheDoor(t *testing.T) {
	var resp ApiResponse[map[string]any]
	// 不存在的群组 → 10300 群组不存在（spec api_spec_group.md 错误码）
	apiRespOKWithCode(t, "/api/v1/groups/open_the_door", map[string]any{"group_id": "fake_group"}, &resp, 10300)
}

func TestGroups_ListMembers(t *testing.T) {
	var resp ApiResponse[map[string]any]
	apiRespOKWithCode(t, "/api/v1/groups/members/fetch", map[string]any{"group_id": "fake_group"}, &resp, 10300)
}

func TestGroups_SearchMembers(t *testing.T) {
	var resp ApiResponse[map[string]any]
	apiRespOKWithCode(t, "/api/v1/groups/members/search", map[string]any{
		"group_id": "fake_group",
		"keyword":  "test",
	}, &resp, 10300)
}

func TestGroups_AnnouncementsList(t *testing.T) {
	var resp ApiResponse[map[string]any]
	apiRespOK(t, "/public/v1/groups/announcements/fetch", map[string]any{"group_id": "fake_group"}, &resp)
}

// Group authenticated actions
func TestGroups_CreateUpdateDismiss(t *testing.T) {
	withAuth(t, func(token string) {
		// Create
		var createResp ApiResponse[map[string]any]
		apiRespOK(t, "/api/v1/groups/create", map[string]any{
			"name":        "Test Group",
			"description": "integration test group",
			"location": map[string]any{
				"point": map[string]any{
					"latitude":  37.7749,
					"longitude": -122.4194,
				},
				"location_name": "Test City",
			},
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
		apiRespOK(t, "/api/v1/groups/fetch_joined", map[string]any{}, &resp)
	})
}

// =============================================================================
// Moments 发现流（feed 域废除后迁移至 moments/fetch_following、moments/fetch_nearby，
// spec api_spec_moment.md / api_spec_discovery.md）
// =============================================================================

func TestMoments_DiscoveryFetchFollowing(t *testing.T) {
	withAuth(t, func(token string) {
		var resp ApiResponse[map[string]any]
		apiRespOK(t, "/api/v1/moments/fetch_following", map[string]any{
			"reference_point": map[string]any{
				"latitude":  37.7749,
				"longitude": -122.4194,
			},
		}, &resp)
	})
}

func TestMoments_DiscoveryFetchNearby(t *testing.T) {
	withAuth(t, func(token string) {
		var resp ApiResponse[map[string]any]
		apiRespOK(t, "/api/v1/moments/fetch_nearby", map[string]any{
			"reference_point": map[string]any{
				"latitude":  37.7749,
				"longitude": -122.4194,
			},
		}, &resp)
	})
}

func TestPublic_MomentsFetchNearbyAnonymous(t *testing.T) {
	// 匿名版拉新路径（spec api_spec_public.md）：仅 public 瞬间，无查看者字段。
	var resp ApiResponse[map[string]any]
	apiRespOK(t, "/public/v1/moments/fetch_nearby", map[string]any{
		"reference_point": map[string]any{
			"latitude":  37.7749,
			"longitude": -122.4194,
		},
	}, &resp)
}

// =============================================================================
// Search
// =============================================================================

func TestSearch_Suggest(t *testing.T) {
	// search/suggest 需登录（spec api_spec_discovery.md 认证档位）
	withAuth(t, func(token string) {
		var resp ApiResponse[map[string]any]
		apiRespOK(t, "/api/v1/search/suggest", map[string]any{"prefix": "test"}, &resp)
	})
}

// =============================================================================
// Map
// =============================================================================

func TestMap_Canvas(t *testing.T) {
	var resp ApiResponse[map[string]any]
	apiOK(t, "/public/v1/map/canvas", map[string]any{
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
		// 分享目标必须真实存在（spec：资源不存在 → 10300）
		mid := createMomentAsCurrentUser(t)
		var resp ApiResponse[map[string]any]
		apiRespOK(t, "/api/v1/share/create", map[string]any{
			"share_type": "moment",
			"payload": map[string]any{
				"shared_moment_id": mid,
			},
		}, &resp)
		if resp.Data == nil {
			t.Fatalf("share create: no data")
		}
		if code := getString(resp.Data, "share_code"); code != "" {
			apiRespOK(t, "/public/v1/share/open_the_door", map[string]any{"share_code": code}, nil)
		}
	})
}

func TestShare_MeList(t *testing.T) {
	withAuth(t, func(token string) {
		var resp ApiResponse[map[string]any]
		apiRespOK(t, "/api/v1/share/fetch_created", map[string]any{}, &resp)
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
		apiRespOK(t, "/api/v1/private_chats/fetch", map[string]any{}, &resp)
	})
}

func TestPrivateChats_UpdateSettings(t *testing.T) {
	withTwoUsers(t, func(tok1, tok2 string) {
		tok2uid, _ := whoAmI(tok2)
		// Open door first
		apiRespOK(t, "/api/v1/private_chats/open_the_door", map[string]any{"target_user_id": tok2uid}, nil)
		// Then update settings
		mute := true
		apiRespOK(t, "/api/v1/private_chats/update_preferences", map[string]any{
			"target_user_id": tok2uid,
			"is_muted":       mute,
		}, nil)
	})
}

func TestPrivateChats_Delete(t *testing.T) {
	withTwoUsers(t, func(tok1, tok2 string) {
		tok2uid, _ := whoAmI(tok2)
		apiRespOK(t, "/api/v1/private_chats/open_the_door", map[string]any{"target_user_id": tok2uid}, nil)
		apiRespOK(t, "/api/v1/private_chats/delete", map[string]any{"target_user_id": tok2uid}, nil)
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
		apiRespOK(t, "/api/v1/private_messages/fetch", map[string]any{
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
		apiRespOK(t, "/api/v1/private_messages/mark_read", map[string]any{
			"scope":          "session",
			"target_user_id": tok2uid,
		}, nil)
		apiRespOK(t, "/api/v1/private_messages/mark_read", map[string]any{"scope": "all"}, nil)
	})
}

// =============================================================================
// Group Messages
// =============================================================================

func TestGroupMessages_SendGetHistory(t *testing.T) {
	withAuth(t, func(token string) {
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
	})
}

func TestGroupMessages_UpdateRecall(t *testing.T) {
	withAuth(t, func(token string) {
		// 不存在的消息 → 业务错误（10300/10400），HTTP 仍 200；此处仅验证可达
		var recallResp ApiResponse[map[string]any]
		status, _ := post("/api/v1/group_messages/recall", map[string]any{
			"message_id": "fake_message",
		}, &recallResp)
		if status != http.StatusOK || recallResp.Code == 0 {
			t.Fatalf("recall fake message: status=%d code=%d", status, recallResp.Code)
		}
		var updateResp ApiResponse[map[string]any]
		status, _ = post("/api/v1/group_messages/update", map[string]any{
			"message_id": "fake_message",
			"content":    "updated",
		}, &updateResp)
		if status != http.StatusOK || updateResp.Code == 0 {
			t.Fatalf("update fake message: status=%d code=%d", status, updateResp.Code)
		}
	})
}

func TestGroupMessages_MarkAsRead(t *testing.T) {
	withAuth(t, func(token string) {
		apiRespOK(t, "/api/v1/group_messages/mark_read", map[string]any{
			"scope":    "group",
			"group_id": "fake_group",
		}, nil)
		apiRespOK(t, "/api/v1/group_messages/mark_read", map[string]any{"scope": "all"}, nil)
	})
}

// =============================================================================
// Notifications
// =============================================================================

func TestNotifications_List(t *testing.T) {
	withAuth(t, func(token string) {
		var resp ApiResponse[map[string]any]
		apiRespOK(t, "/api/v1/notify/fetch", map[string]any{}, &resp)
	})
}

func TestNotifications_UnreadCount(t *testing.T) {
	withAuth(t, func(token string) {
		var resp ApiResponse[map[string]any]
		apiRespOK(t, "/api/v1/notify/unread_summary", map[string]any{}, &resp)
	})
}

func TestNotifications_MarkReadSelected(t *testing.T) {
	withAuth(t, func(token string) {
		apiRespOK(t, "/api/v1/notify/mark_read", map[string]any{
			"notify_ids": []string{"fake_notif_id"},
		}, nil)
	})
}

func TestNotifications_MarkReadAll(t *testing.T) {
	withAuth(t, func(token string) {
		apiRespOK(t, "/api/v1/notify/mark_read", map[string]any{}, nil)
	})
}

// =============================================================================
// Reports
// =============================================================================

func TestReports_Create(t *testing.T) {
	withAuth(t, func(token string) {
		// 目标必须真实存在（10300）且不能举报自己（10400）→ 以另一作者身份创建瞬间
		_, authorTok := createTestUserWithToken(t, "reported_author")
		oldTok := accessToken
		accessToken = authorTok
		mid := createMomentAsCurrentUser(t)
		accessToken = oldTok

		var resp ApiResponse[map[string]any]
		apiRespOK(t, "/api/v1/reports/create", map[string]any{
			"target_type": "moment",
			"target_id":   mid,
			"reason":      "spam",
			"description": "test report from integration test",
		}, &resp)
	})
}

func TestReports_MeList(t *testing.T) {
	withAuth(t, func(token string) {
		var resp ApiResponse[map[string]any]
		apiRespOK(t, "/api/v1/reports/fetch", map[string]any{}, &resp)
	})
}

// =============================================================================
// Media
// =============================================================================

func TestMedia_PresignUpload(t *testing.T) {
	withAuth(t, func(token string) {
		var resp ApiResponse[map[string]any]
		apiRespOK(t, "/api/v1/media/presign_upload", map[string]any{
			"files": []map[string]any{
				{"file_name": "test.png", "file_size": 1024, "mime_type": "image/png", "category": "avatar"},
			},
		}, &resp)
		if resp.Data == nil {
			t.Fatal("expected data in presign response")
		}
		data := firstUploadEntry(t, resp.Data)
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
		apiRespOK(t, "/api/v1/media/presign_upload", map[string]any{
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
			"upload_ids": []string{"nonexistent_upload_id"},
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
	apiRespOKWithHost(t, mediaPresignClientOrigin, "/api/v1/media/presign_upload", map[string]any{
		"files": []map[string]any{
			{"file_name": "photo.png", "file_size": 66, "mime_type": "image/png", "category": "avatar"},
		},
	}, &presignResp)

	presignUpload := firstUploadEntry(t, presignResp.Data)
	uploadID, _ := presignUpload["upload_id"].(string)
	uploadURL, _ := presignUpload["upload_url"].(string)
	if uploadID == "" || uploadURL == "" {
		t.Fatalf("presign returned empty upload_id or upload_url: %+v", presignResp)
	}
	// 直传 URL 的 origin 跟随 presign 请求的 Host（nginx /s3/ 以 $http_host 转发，
	// SigV4 验签依赖该一致性）；集成环境以网关 host 发起 presign，PUT 即同 URL。
	// 直传 URL 必须与请求入口 origin 同源，并沿用网关的 /s3 挂载前缀。
	wantPrefix := "http://" + mediaPresignClientOrigin + "/s3/"
	if !strings.HasPrefix(uploadURL, wantPrefix) {
		t.Fatalf("upload_url should follow presign request origin, want prefix %s got %s", wantPrefix, uploadURL)
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
		"upload_ids": []string{uploadID},
	}, &completeResp)
	if completeResp.Code != 0 {
		t.Fatalf("complete_upload failed: code=%d message=%s", completeResp.Code, completeResp.Message)
	}
	if completeResp.Data == nil {
		t.Fatal("complete_upload returned no data")
	}
	data := firstUploadEntry(t, completeResp.Data)
	fileID, _ := data["file_id"].(string)
	if fileID == "" {
		t.Fatal("expected file_id in complete response")
	}
	fileSize, _ := data["file_size"].(float64)
	if int(fileSize) != len(pngBytes) {
		t.Fatalf("expected file_size=%d, got %d", len(pngBytes), int(fileSize))
	}
	mimeType, _ := data["mime_type"].(string)
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
	apiRespOKWithHost(t, mediaPresignClientOrigin, "/api/v1/media/presign_upload", map[string]any{
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
	// 直传 URL 的 origin 跟随 presign 请求的 Host（同单文件 E2E 断言）。
	// 直传 URL 必须与请求入口 origin 同源，并沿用网关的 /s3 挂载前缀。
	wantPrefix := "http://" + mediaPresignClientOrigin + "/s3/"
	if !strings.HasPrefix(uploadURL, wantPrefix) {
		t.Fatalf("upload_url should follow presign request origin, want prefix %s got %s", wantPrefix, uploadURL)
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
		"upload_ids": []string{uploadID},
	}, &completeResp)
	if completeResp.Code != 0 {
		t.Fatalf("batch complete failed: code=%d message=%s", completeResp.Code, completeResp.Message)
	}
}

func TestMedia_E2E_SizeMismatchRejected(t *testing.T) {
	_, token := createGuestUser(t)
	oldToken := accessToken
	accessToken = token
	defer func() { accessToken = oldToken }()

	var presignResp ApiResponse[map[string]any]
	apiRespOK(t, "/api/v1/media/presign_upload", map[string]any{
		"files": []map[string]any{
			{"file_name": "size_mismatch.png", "file_size": 999, "mime_type": "image/png", "category": "avatar"},
		},
	}, &presignResp)
	mismatchUpload := firstUploadEntry(t, presignResp.Data)
	uploadID, _ := mismatchUpload["upload_id"].(string)
	uploadURL, _ := mismatchUpload["upload_url"].(string)

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
	putResp.Body.Close()

	var completeResp ApiResponse[map[string]any]
	apiOK(t, "/api/v1/media/complete_upload", map[string]any{
		"upload_ids": []string{uploadID},
	}, &completeResp)
	if completeResp.Code == 0 {
		t.Fatal("expected error for size mismatch, got success")
	}
}

func TestMedia_E2E_CompleteBeforeUpload(t *testing.T) {
	_, token := createGuestUser(t)
	oldToken := accessToken
	accessToken = token
	defer func() { accessToken = oldToken }()

	var presignResp ApiResponse[map[string]any]
	apiRespOK(t, "/api/v1/media/presign_upload", map[string]any{
		"files": []map[string]any{
			{"file_name": "not_uploaded.png", "file_size": 100, "mime_type": "image/png", "category": "avatar"},
		},
	}, &presignResp)
	notUploaded := firstUploadEntry(t, presignResp.Data)
	uploadID, _ := notUploaded["upload_id"].(string)

	var completeResp ApiResponse[map[string]any]
	apiOK(t, "/api/v1/media/complete_upload", map[string]any{
		"upload_ids": []string{uploadID},
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
	withAdminAuth(t, func(token string) {
		var resp ApiResponse[map[string]any]
		apiRespOK(t, "/admin/api/v1/rbac/me", map[string]any{}, &resp)
	})
}

func TestAdmin_RBAC_ListPermissions(t *testing.T) {
	withAdminAuth(t, func(token string) {
		var resp ApiResponse[map[string]any]
		apiRespOK(t, "/admin/api/v1/rbac/permissions/fetch", map[string]any{}, &resp)
	})
}

func TestAdmin_RBAC_RolesList(t *testing.T) {
	withAdminAuth(t, func(token string) {
		var resp ApiResponse[map[string]any]
		apiRespOK(t, "/admin/api/v1/rbac/roles/fetch", map[string]any{}, &resp)
	})
}

func TestAdmin_RBAC_AdminsList(t *testing.T) {
	withAdminAuth(t, func(token string) {
		var resp ApiResponse[map[string]any]
		apiRespOK(t, "/admin/api/v1/rbac/admins/fetch", map[string]any{}, &resp)
	})
}

func TestAdmin_AppConfigGet(t *testing.T) {
	withAdminAuth(t, func(token string) {
		var resp ApiResponse[map[string]any]
		apiRespOK(t, "/admin/api/v1/app_config", map[string]any{}, &resp)
	})
}

func TestAdmin_Stats_Geo(t *testing.T) {
	withAdminAuth(t, func(token string) {
		var resp ApiResponse[map[string]any]
		apiRespOK(t, "/admin/api/v1/stats/geo", map[string]any{}, &resp)
		if resp.Data == nil {
			t.Fatal("expected data")
		}
		for _, key := range []string{"generated_at", "totals", "regions", "places"} {
			if _, ok := (*resp.Data)[key]; !ok {
				t.Fatalf("missing %s in response", key)
			}
		}
		// 层级越界 → 10000（参数必显式）。
		var badResp ApiResponse[map[string]any]
		apiRespOKWithCode(t, "/admin/api/v1/stats/geo", map[string]any{"level": 99}, &badResp, 10000)

		// 趋势：trend_days=7 → trends 恒 7 点。
		var trendResp ApiResponse[map[string]any]
		apiRespOK(t, "/admin/api/v1/stats/geo", map[string]any{"trend_days": 7}, &trendResp)
		trends, ok := (*trendResp.Data)["trends"].([]any)
		if !ok || len(trends) != 7 {
			t.Fatalf("expected 7 trend points, got %+v", (*trendResp.Data)["trends"])
		}
		if _, ok := trends[0].(map[string]any)["date"].(string); !ok {
			t.Fatalf("trend point missing date: %+v", trends[0])
		}
	})
}

func TestAdmin_Stats_Geo_Drilldown(t *testing.T) {
	withAdminAuth(t, func(token string) {
		// 管理员自己就有默认位置（guest 注册即带位置），regions 至少 1 个。
		var statsResp ApiResponse[map[string]any]
		apiRespOK(t, "/admin/api/v1/stats/geo", map[string]any{}, &statsResp)
		regions, ok := (*statsResp.Data)["regions"].([]any)
		if !ok || len(regions) == 0 {
			t.Fatalf("expected at least 1 region, got %+v", (*statsResp.Data)["regions"])
		}
		region0 := regions[0].(map[string]any)
		cellToken := region0["cell_token"].(string)

		var dd ApiResponse[map[string]any]
		apiRespOK(t, "/admin/api/v1/stats/geo/drilldown", map[string]any{
			"cell_token": cellToken, "entity_type": "user",
			"pagination": map[string]any{"limit": 5},
		}, &dd)
		items, ok := (*dd.Data)["items"].([]any)
		if !ok {
			t.Fatalf("expected items array, got %+v", (*dd.Data)["items"])
		}
		for idx, raw := range items {
			item := raw.(map[string]any)
			for _, key := range []string{"object_id", "title", "created_at"} {
				if _, ok := item[key]; !ok {
					t.Fatalf("drilldown item %d missing %s", idx, key)
				}
			}
		}
		// next_cursor 键恒在。
		if _, ok := (*dd.Data)["next_cursor"]; !ok {
			t.Fatal("missing next_cursor key")
		}
		// 非法 token / 类型白名单外 → 10000。
		var bad ApiResponse[map[string]any]
		apiRespOKWithCode(t, "/admin/api/v1/stats/geo/drilldown", map[string]any{"cell_token": "zz", "entity_type": "user"}, &bad, 10000)
		apiRespOKWithCode(t, "/admin/api/v1/stats/geo/drilldown", map[string]any{"cell_token": cellToken, "entity_type": "comment"}, &bad, 10000)
	})
}

func TestAdmin_Stats_Geo_RejectsNonAdmin(t *testing.T) {
	userID, token := createGuestUser(t)
	_ = userID
	oldTok := accessToken
	accessToken = token
	defer func() { accessToken = oldTok }()
	var resp ApiResponse[map[string]any]
	apiRespOKWithCode(t, "/admin/api/v1/stats/geo", map[string]any{}, &resp, 10200)
}

func TestAdmin_UsersSearch(t *testing.T) {
	withAdminAuth(t, func(token string) {
		var resp ApiResponse[map[string]any]
		apiRespOK(t, "/admin/api/v1/users/search", map[string]any{}, &resp)
	})
}

func TestAdmin_GroupsSearch(t *testing.T) {
	withAdminAuth(t, func(token string) {
		var resp ApiResponse[map[string]any]
		apiRespOK(t, "/admin/api/v1/groups/search", map[string]any{}, &resp)
	})
}

func TestAdmin_ReportsSearch(t *testing.T) {
	withAdminAuth(t, func(token string) {
		var resp ApiResponse[map[string]any]
		apiRespOK(t, "/admin/api/v1/reports/fetch", map[string]any{}, &resp)
	})
}

func TestAdmin_TagsSearch(t *testing.T) {
	withAdminAuth(t, func(token string) {
		var resp ApiResponse[map[string]any]
		apiRespOK(t, "/admin/api/v1/tags/search", map[string]any{}, &resp)
	})
}

func TestAdmin_Announcements(t *testing.T) {
	withAdminAuth(t, func(token string) {
		var resp ApiResponse[map[string]any]
		apiRespOK(t, "/admin/api/v1/announcements/fetch", map[string]any{}, &resp)
	})
}

func TestAdmin_Audit(t *testing.T) {
	withAdminAuth(t, func(token string) {
		var resp ApiResponse[map[string]any]
		apiRespOK(t, "/admin/api/v1/audit/fetch", map[string]any{}, &resp)
	})
}
