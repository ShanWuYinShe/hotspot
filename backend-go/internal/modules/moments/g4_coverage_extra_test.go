package moments

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/appconfig"
	"hotspot/backend-go/internal/platform/config"
	platformcursor "hotspot/backend-go/internal/platform/cursor"
	"hotspot/backend-go/internal/platform/geo"
	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/types"
)

// ── discovery.go：分页 limit / 排序比较 / 游标编解码分支 ────────────────

func TestG4NormalizeFetchLimitBranches(t *testing.T) {
	cases := []struct {
		name string
		in   *PaginationParams
		want int
	}{
		{"nil", nil, 20},
		{"zero", &PaginationParams{Limit: 0}, 20},
		{"negative", &PaginationParams{Limit: -3}, 20},
		{"over max", &PaginationParams{Limit: 101}, 100},
		{"in range", &PaginationParams{Limit: 55}, 55},
	}
	for _, tc := range cases {
		if got := normalizeFetchLimit(tc.in); got != tc.want {
			t.Fatalf("%s: normalizeFetchLimit = %d, want %d", tc.name, got, tc.want)
		}
	}
}

func TestG4CompareFollowingCandidatesBranches(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	later := momentCandidate{Item: MomentItem{MomentID: "m_b"}, CreatedAt: now.Add(time.Minute)}
	earlier := momentCandidate{Item: MomentItem{MomentID: "m_a"}, CreatedAt: now}
	if got := compareFollowingCandidates(later, earlier); got != -1 {
		t.Fatalf("later vs earlier = %d, want -1", got)
	}
	if got := compareFollowingCandidates(earlier, later); got != 1 {
		t.Fatalf("earlier vs later = %d, want 1", got)
	}
	highID := momentCandidate{Item: MomentItem{MomentID: "m_b"}, CreatedAt: now}
	lowID := momentCandidate{Item: MomentItem{MomentID: "m_a"}, CreatedAt: now}
	if got := compareFollowingCandidates(highID, lowID); got != -1 {
		t.Fatalf("same time high id = %d, want -1", got)
	}
	if got := compareFollowingCandidates(lowID, highID); got != 1 {
		t.Fatalf("same time low id = %d, want 1", got)
	}
	if got := compareFollowingCandidates(lowID, lowID); got != 0 {
		t.Fatalf("identical candidate = %d, want 0", got)
	}
}

func TestG4CompareNearbyCandidatesBranches(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	base := momentCandidate{Item: MomentItem{MomentID: "m_b"}, Distance: 100, Score: 5, CreatedAt: now}

	closer := momentCandidate{Item: MomentItem{MomentID: "m_a"}, Distance: 50, Score: 1, CreatedAt: now}
	if got := compareNearbyCandidates(closer, base); got != -50 {
		t.Fatalf("closer vs base = %d, want -50", got)
	}
	if got := compareNearbyCandidates(base, closer); got != 50 {
		t.Fatalf("base vs closer = %d, want 50", got)
	}

	hotter := momentCandidate{Item: MomentItem{MomentID: "m_c"}, Distance: 100, Score: 9, CreatedAt: now}
	if got := compareNearbyCandidates(hotter, base); got != -4 {
		t.Fatalf("hotter vs base = %d, want -4", got)
	}
	if got := compareNearbyCandidates(base, hotter); got != 4 {
		t.Fatalf("base vs hotter = %d, want 4", got)
	}

	older := momentCandidate{Item: MomentItem{MomentID: "m_d"}, Distance: 100, Score: 5, CreatedAt: now.Add(-time.Minute)}
	if got := compareNearbyCandidates(base, older); got != -1 {
		t.Fatalf("newer vs older = %d, want -1", got)
	}
	if got := compareNearbyCandidates(older, base); got != 1 {
		t.Fatalf("older vs newer = %d, want 1", got)
	}

	highID := momentCandidate{Item: MomentItem{MomentID: "m_b"}, Distance: 100, Score: 5, CreatedAt: now}
	lowID := momentCandidate{Item: MomentItem{MomentID: "m_a"}, Distance: 100, Score: 5, CreatedAt: now}
	if got := compareNearbyCandidates(highID, lowID); got != -1 {
		t.Fatalf("same tuple high id = %d, want -1", got)
	}
	if got := compareNearbyCandidates(lowID, highID); got != 1 {
		t.Fatalf("same tuple low id = %d, want 1", got)
	}
	if got := compareNearbyCandidates(lowID, lowID); got != 0 {
		t.Fatalf("identical candidate = %d, want 0", got)
	}
}

func TestG4DecodeFetchCursorInvalidBranches(t *testing.T) {
	// 可解码但字段缺失（ID 为空）→ ErrInvalidCursor
	forged, err := fetchCursorCodec.Encode(itemCursor{})
	if err != nil {
		t.Fatalf("encode forged cursor: %v", err)
	}
	if _, err := decodeFetchCursor(forged); !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("missing id err = %v, want ErrInvalidCursor", err)
	}

	// created_at 非法时间格式 → ErrInvalidCursor
	forgedAt, err := fetchCursorCodec.Encode(itemCursor{ID: "m1", CreatedAt: "not-a-time"})
	if err != nil {
		t.Fatalf("encode forged at cursor: %v", err)
	}
	if _, err := decodeFetchCursor(forgedAt); !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("invalid created_at err = %v, want ErrInvalidCursor", err)
	}

	if _, err := decodeFetchCursor("clearly-not-a-cursor"); !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("garbage cursor err = %v, want ErrInvalidCursor", err)
	}
	if decoded, err := decodeFetchCursor(""); decoded != nil || err != nil {
		t.Fatalf("empty cursor must yield nil,nil, got %#v err=%v", decoded, err)
	}
}

func TestG4PaginateInvalidCursor(t *testing.T) {
	if _, err := paginateFollowingMoments(nil, 20, "bad-cursor"); !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("following invalid cursor err = %v, want ErrInvalidCursor", err)
	}
	if _, err := paginateNearbyMoments(nil, 20, "bad-cursor"); !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("nearby invalid cursor err = %v, want ErrInvalidCursor", err)
	}
}

func TestG4FindNearbyCursorStartBranches(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	items := []momentCandidate{
		{Item: MomentItem{MomentID: "m1"}, Distance: 10, CreatedAt: now},
		{Item: MomentItem{MomentID: "m2"}, Distance: 20, CreatedAt: now},
		{Item: MomentItem{MomentID: "m3"}, Distance: 30, CreatedAt: now},
	}
	if got := findNearbyCursorStart(items, nil); got != 0 {
		t.Fatalf("nil cursor start = %d, want 0", got)
	}
	// 游标比首项更近：首项严格排在游标之后 → 从下标 0 开始
	closerCursor := &itemCursor{Distance: 5, CreatedAt: now.Format(time.RFC3339Nano), ID: "m0"}
	if got := findNearbyCursorStart(items, closerCursor); got != 0 {
		t.Fatalf("closer cursor start = %d, want 0", got)
	}
	// 游标比所有项都远 → 落在末尾（返回 len）
	behindCursor := &itemCursor{Distance: 500, CreatedAt: now.Format(time.RFC3339Nano), ID: "m9"}
	if got := findNearbyCursorStart(items, behindCursor); got != len(items) {
		t.Fatalf("behind cursor start = %d, want %d", got, len(items))
	}
	// 游标与首项完全相等 → 从下一项开始（cmp==0 分支）
	equalCursor := &itemCursor{Distance: 10, CreatedAt: now.Format(time.RFC3339Nano), ID: "m1"}
	if got := findNearbyCursorStart(items, equalCursor); got != 1 {
		t.Fatalf("equal cursor start = %d, want 1", got)
	}
}

type g4LocationProvider struct {
	point geo.Point
	found bool
}

func (p g4LocationProvider) RecentLocation(context.Context, string) (geo.Point, bool) {
	return p.point, p.found
}

func TestG4ResolveReferencePointFallbacks(t *testing.T) {
	repo := &stubMomentRepo{}
	svc := NewServiceWithRepo(repo)

	// 回退 2：已登录用户最近位置
	svc.SetLocationProvider(g4LocationProvider{point: geo.Point{Latitude: 35.5, Longitude: 112.5}, found: true})
	got := svc.resolveReferencePoint(momentCtx("u1"), nil)
	if got == nil || got.Latitude != 35.5 || got.Longitude != 112.5 {
		t.Fatalf("user recent location fallback = %#v", got)
	}

	// provider 未命中 → 回退 3（无 appConfig 时平台默认位置）
	svcMiss := NewServiceWithRepo(&stubMomentRepo{})
	svcMiss.SetLocationProvider(g4LocationProvider{found: false})
	if got := svcMiss.resolveReferencePoint(momentCtx("u1"), nil); got == nil {
		t.Fatal("expected platform default fallback point")
	}

	// 回退 3：appConfig default_location
	svcCfg := NewServiceWithRepo(&stubMomentRepo{})
	svcCfg.appConfig = appconfig.NewStaticSource(config.Config{})
	if got := svcCfg.resolveReferencePoint(context.Background(), nil); got == nil {
		t.Fatal("expected appConfig default location fallback point")
	}

	// 显式非法坐标 → nil
	invalid := GeoPoint{Latitude: 91, Longitude: 0}
	if got := svc.resolveReferencePoint(context.Background(), &invalid); got != nil {
		t.Fatalf("invalid explicit reference point = %#v, want nil", got)
	}
}

// ── search_cursor.go：搜索分值与 keyset 比较/解码分支 ──────────────────

func TestG4MomentSearchScoreTagBranch(t *testing.T) {
	now := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	item := MomentItem{Description: "hello world", Tags: []string{"golang"}, CreatedAt: now.Format(time.RFC3339Nano)}

	// relevance：标签精确命中计 1 分（keyword 为空）
	if score := momentSearchScore(item, "relevance", "", "golang", now); score != 1 {
		t.Fatalf("tag match score = %v, want 1", score)
	}
	// relevance：keyword + tag 同时命中计 2 分
	if score := momentSearchScore(item, "relevance", "hello", "golang", now); score != 2 {
		t.Fatalf("keyword+tag score = %v, want 2", score)
	}
	// relevance：均不命中 0 分
	if score := momentSearchScore(item, "relevance", "absent", "rust", now); score != 0 {
		t.Fatalf("no match score = %v, want 0", score)
	}
	// newest 恒 0 分
	if score := momentSearchScore(item, "newest", "hello", "golang", now); score != 0 {
		t.Fatalf("newest score = %v, want 0", score)
	}
}

func TestG4CompareSearchTupleBranches(t *testing.T) {
	cur := momentSearchCursor{Score: 2, At: "2026-01-01T00:00:00Z", ID: "m2"}
	if got := compareSearchTuple(1, cur.At, "m1", cur); got != -1 {
		t.Fatalf("lower score = %d, want -1", got)
	}
	if got := compareSearchTuple(3, cur.At, "m1", cur); got != 1 {
		t.Fatalf("higher score = %d, want 1", got)
	}
	if got := compareSearchTuple(2, "2025-12-31T00:00:00Z", "m1", cur); got != -1 {
		t.Fatalf("older at = %d, want -1", got)
	}
	if got := compareSearchTuple(2, "2026-01-02T00:00:00Z", "m1", cur); got != 1 {
		t.Fatalf("newer at = %d, want 1", got)
	}
	if got := compareSearchTuple(2, cur.At, "m1", cur); got != -1 {
		t.Fatalf("lower id = %d, want -1", got)
	}
	if got := compareSearchTuple(2, cur.At, "m3", cur); got != 1 {
		t.Fatalf("higher id = %d, want 1", got)
	}
	if got := compareSearchTuple(2, cur.At, "m2", cur); got != 0 {
		t.Fatalf("identical tuple = %d, want 0", got)
	}
}

func TestG4DecodeMomentSearchCursorInvalidBranches(t *testing.T) {
	missingFields, err := platformcursor.Encode(momentSearchCursorNamespace, momentSearchCursor{Sort: "relevance"})
	if err != nil {
		t.Fatalf("encode missing-fields cursor: %v", err)
	}
	if _, err := decodeMomentSearchCursor(missingFields); !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("missing fields err = %v, want ErrInvalidCursor", err)
	}

	badAt, err := platformcursor.Encode(momentSearchCursorNamespace, momentSearchCursor{Sort: "relevance", At: "bad-time", ID: "m1"})
	if err != nil {
		t.Fatalf("encode bad-at cursor: %v", err)
	}
	if _, err := decodeMomentSearchCursor(badAt); !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("invalid at err = %v, want ErrInvalidCursor", err)
	}

	if _, err := decodeMomentSearchCursor("garbage-cursor"); !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("garbage err = %v, want ErrInvalidCursor", err)
	}
}

// ── nearby_notify.go：目标查询失败 / 发布失败分支 ──────────────────────

type g4NearbyPublisher struct {
	err        error
	calls      int
	lastUserID string
}

func (p *g4NearbyPublisher) PublishNearbyNewMomentPreChecked(_ context.Context, targetUserID, _, _ string, _ int, _ types.UserInfo, _ types.MomentSummary) error {
	p.calls++
	p.lastUserID = targetUserID
	return p.err
}

func TestG4NotifyNearbyNewMomentBranches(t *testing.T) {
	svc := NewServiceWithRepo(&stubMomentRepo{})
	pub := &g4NearbyPublisher{}
	svc.SetNearbyNotifyPublisher(pub)

	// 目标查询失败 → 仅记录 WARN，不发布
	svc.SetNearbyNotifyTargetsReader(func(context.Context, float64, float64, int, string, int) ([]NearbyNotifyTarget, error) {
		return nil, errors.New("targets boom")
	})
	svc.notifyNearbyNewMoment(context.Background(), "u1", "m1", 35.0, 112.0, types.UserInfo{}, types.MomentSummary{})
	if pub.calls != 0 {
		t.Fatalf("publisher must not be called on targets error, calls = %d", pub.calls)
	}

	// 目标发布失败 → 记录 WARN 后继续（尽力而为，不中断）
	failingPub := &g4NearbyPublisher{err: errors.New("publish boom")}
	svc.SetNearbyNotifyPublisher(failingPub)
	svc.SetNearbyNotifyTargetsReader(func(context.Context, float64, float64, int, string, int) ([]NearbyNotifyTarget, error) {
		return []NearbyNotifyTarget{{UserID: "u2", Latitude: 35.001, Longitude: 112.001}}, nil
	})
	svc.notifyNearbyNewMoment(context.Background(), "u1", "m1", 35.0, 112.0, types.UserInfo{}, types.MomentSummary{})
	if failingPub.calls != 1 || failingPub.lastUserID != "u2" {
		t.Fatalf("publish failure must still be attempted once, got calls=%d target=%q", failingPub.calls, failingPub.lastUserID)
	}

	// 未接线（publisher/targets 均为 nil）→ 静默返回
	NewServiceWithRepo(&stubMomentRepo{}).notifyNearbyNewMoment(context.Background(), "u1", "m1", 35.0, 112.0, types.UserInfo{}, types.MomentSummary{})
}

// ── images.go：缩略图 presign 失败回退与已加载图片水合早退 ──────────────

type g4ImageResolver struct {
	failThumb bool
}

func (g4ImageResolver) ResolveMomentImages(context.Context, []string, string) (map[string]ImageAsset, error) {
	return nil, nil
}

func (r g4ImageResolver) PresignGet(_ context.Context, storageKey string, _ time.Duration) (string, error) {
	if r.failThumb && storageKey == "thumb-key" {
		return "", errors.New("presign thumb boom")
	}
	return "https://cdn.example/" + storageKey, nil
}

func TestG4PresignMomentImageRowsThumbnailFallback(t *testing.T) {
	rows := []momentImageRow{{StorageKey: "img-key", ThumbnailURL: "thumb-key", Width: 100, Height: 80, MimeType: "image/jpeg"}}
	images := presignMomentImageRows(context.Background(), g4ImageResolver{failThumb: true}, rows)
	if len(images) != 1 {
		t.Fatalf("images = %d, want 1", len(images))
	}
	img := images[0]
	if img.URL != "https://cdn.example/img-key" {
		t.Fatalf("image url = %q", img.URL)
	}
	if img.ThumbnailURL == nil || *img.ThumbnailURL != "https://cdn.example/img-key" {
		t.Fatalf("thumbnail presign failure must fall back to full url, got %#v", img.ThumbnailURL)
	}
	if img.Width == nil || *img.Width != 100 || img.Height == nil || *img.Height != 80 || img.MimeType == nil || *img.MimeType != "image/jpeg" {
		t.Fatalf("image metadata = %#v", img)
	}

	if got := presignMomentImageRows(context.Background(), nil, rows); got != nil {
		t.Fatalf("nil resolver must yield nil images, got %#v", got)
	}
	if got := presignMomentImageRows(context.Background(), g4ImageResolver{}, nil); got != nil {
		t.Fatalf("empty rows must yield nil images, got %#v", got)
	}
}

func TestG4HydrateLoadedMomentImagesEarlyReturn(t *testing.T) {
	// resolver 为 nil / items 为空均在入口直接返回
	HydrateLoadedMomentImages(context.Background(), nil, []MomentItem{{MomentID: "m1"}})
	HydrateLoadedMomentImages(context.Background(), g4ImageResolver{}, nil)

	items := []MomentItem{{MomentID: "m1", imageRows: []momentImageRow{{StorageKey: "key-img"}}}}
	HydrateLoadedMomentImages(context.Background(), g4ImageResolver{}, items)
	if len(items[0].Images) != 1 || items[0].Images[0].URL != "https://cdn.example/key-img" {
		t.Fatalf("loaded image hydration = %#v", items[0].Images)
	}
}

// ── handler.go：剩余分支（超长正文 / 列表解码错误 / 关注流成功）─────────

func TestG4HandlerCreateDescriptionTooLongBranch(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&stubMomentRepo{}))
	body := fmt.Sprintf(`{"description":%q,"location":{"point":{"latitude":35.0,"longitude":112.0},"location_name":"x"}}`, strings.Repeat("a", defaultMomentDescriptionMaxRunes+1))
	req := httptest.NewRequest(http.MethodPost, "/create", strings.NewReader(body))
	req.Header.Set("Idempotency-Key", "g4longdesc000000000000000000000001")
	req = req.WithContext(momentCtx("u1"))
	rr := httptest.NewRecorder()
	handler.Create(rr, req)
	assertMomentCode(t, rr, httpx.CodeInvalidRequest)
}

func TestG4HandlerListCreatedDecodeErrorBranch(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&stubMomentRepo{}))
	req := httptest.NewRequest(http.MethodPost, "/fetch_created", strings.NewReader("{bad json"))
	rr := httptest.NewRecorder()
	handler.ListCreated(rr, req)
	assertMomentCode(t, rr, httpx.CodeInvalidRequest)
}

func TestG4HandlerFetchFollowingSuccessBranch(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&stubMomentRepo{}))
	req := httptest.NewRequest(http.MethodPost, "/fetch_following", strings.NewReader("{}"))
	req = req.WithContext(momentCtx("u1"))
	rr := httptest.NewRecorder()
	handler.FetchFollowing(rr, req)
	assertMomentCode(t, rr, 0)
}
