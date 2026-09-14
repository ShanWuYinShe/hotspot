package moments

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	platformauth "hotspot/backend-go/internal/platform/auth"
)

// 本文件验证 moments/search 的契约对齐（spec api_spec_moment.md moments/search）：
//   - sort_by='popular' 含时间衰减因子（算法为实现细节，存在性为契约）；
//   - 游标不透明、keyset 语义，popular 模式翻页不重不漏；
//   - 登录查看者 is_favorited 反映真实收藏态（spec api_spec_types.md MomentItem）。

func seedSearchMoment(t *testing.T, repo *InMemoryRepository, id, description string, createdAt time.Time, likes, comments int) {
	t.Helper()
	repo.moments[id] = MomentItem{
		MomentID:     id,
		Description:  description,
		PrivacyLevel: "public",
		CreatedAt:    createdAt.UTC().Format(time.RFC3339),
		LikesCount:   likes, CommentsCount: comments,
		User: UserInfo{UserID: "author"},
	}
}

// TestMemorySearchPopularTimeDecayApplies：时间衰减生效——新发布低热度瞬间应排在
// 久远高热度瞬间之前（无衰减时后者恒在前）。
func TestMemorySearchPopularTimeDecayApplies(t *testing.T) {
	repo := NewInMemoryRepository()
	now := time.Now().UTC()
	// 240 小时前、10 次互动：衰减分值 ≈ 10 / 242^1.5 ≈ 0.0027
	seedSearchMoment(t, repo, "moment_old", "old hot post", now.Add(-240*time.Hour), 8, 2)
	// 1 小时前、5 次互动：衰减分值 ≈ 5 / 3^1.5 ≈ 0.96
	seedSearchMoment(t, repo, "moment_fresh", "fresh quiet post", now.Add(-1*time.Hour), 3, 2)
	// 无输入条件的热门模式
	req := SearchMomentsRequest{SortBy: "popular"}
	result, err := repo.Search(context.Background(), req, "")
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(result.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(result.Items))
	}
	if result.Items[0].MomentID != "moment_fresh" {
		t.Fatalf("expected fresh low-engagement moment to outrank stale hot moment under time decay, got %s", result.Items[0].MomentID)
	}
}

// TestMemorySearchPopularCursorPaginationNoDupNoSkip：热门模式 keyset 翻页
// 不重不漏（游标分值含衰减浮点，编码/解码保持元组语义）。
func TestMemorySearchPopularCursorPaginationNoDupNoSkip(t *testing.T) {
	repo := NewInMemoryRepository()
	now := time.Now().UTC()
	total := 5
	want := make([]string, 0, total)
	for i := 0; i < total; i++ {
		id := fmt.Sprintf("moment_%02d", i)
		// 互动数随序号递减、发布时间错开，制造互异的 (score, created_at, id) 元组
		seedSearchMoment(t, repo, id, "post", now.Add(-time.Duration(i*7)*time.Hour), total-i, 0)
		want = append(want, id)
	}
	ctx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "viewer"})
	got := make([]string, 0, total)
	cursor := ""
	for page := 0; page < total+1; page++ {
		req := SearchMomentsRequest{SortBy: "popular", Pagination: &PaginationParams{Limit: 1}}
		if cursor != "" {
			req.Pagination.Cursor = cursor
		}
		result, err := repo.Search(ctx, req, "")
		if err != nil {
			t.Fatalf("page %d: %v", page, err)
		}
		if len(result.Items) == 0 {
			break
		}
		got = append(got, result.Items[0].MomentID)
		if result.NextCursor == nil {
			break
		}
		cursor = *result.NextCursor
	}
	if len(got) != total {
		t.Fatalf("expected %d items across pages, got %v", total, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("pagination order drift: want %v got %v", want, got)
		}
	}
}

// TestBuildSearchMomentsQueryPopularDecayAndViewerState：SQL 形状对齐——
// popular 分值含时间衰减 + reactions 聚合点赞数（likes_count 已无写路径，
// 见 F-08）；登录查看者追加 EXISTS(moment_favorites) 尾列，匿名查询不含该列；
// keyset 谓词以 float8 比较衰减分值。
func TestBuildSearchMomentsQueryPopularDecayAndViewerState(t *testing.T) {
	sql, args := buildSearchMomentsQuery(nil, nil, "popular", 21, nil, "viewer_1")
	if !strings.Contains(sql, "SELECT COUNT(*) FROM reactions r WHERE r.target_type = 'moment'") ||
		!strings.Contains(sql, "m.comments_count") || !strings.Contains(sql, "POWER(") {
		t.Fatalf("popular score must aggregate reactions and embed time-decay POWER expression: %s", sql)
	}
	if !strings.Contains(sql, "GREATEST(EXTRACT(EPOCH FROM") {
		t.Fatalf("decay hours must be clamped non-negative: %s", sql)
	}
	// 排序表达式不得再用已无写路径的 likes_count（SELECT 列表仍投影该兼容列，
	// json:"-" 不对外，属有意保留），只看 ORDER BY 段。
	orderBy := sql[strings.Index(sql, "ORDER BY"):]
	if strings.Contains(orderBy, "likes_count") {
		t.Fatalf("popular ORDER BY must not use the dead likes_count column: %s", orderBy)
	}
	if !strings.Contains(sql, "EXISTS(SELECT 1 FROM moment_favorites mf WHERE mf.moment_id = m.moment_id AND mf.user_id = $1)") {
		t.Fatalf("viewer search must fill is_favorited via parameterized EXISTS: %s", sql)
	}
	if len(args) == 0 || args[0] != "viewer_1" {
		t.Fatalf("viewer must be bound as first query arg, got %#v", args)
	}

	// 带游标：谓词首元素按 float8 比较（衰减分值为浮点）
	decoded := &momentSearchCursor{Sort: "popular", Score: 1.25, At: "2026-01-01T00:00:00Z", ID: "moment_9"}
	sqlCursor, _ := buildSearchMomentsQuery(nil, nil, "popular", 21, decoded, "")
	if !strings.Contains(sqlCursor, "< ($") || !strings.Contains(sqlCursor, "::float8") {
		t.Fatalf("keyset predicate must compare decayed score as float8: %s", sqlCursor)
	}
	if strings.Contains(sqlCursor, "::bigint") {
		t.Fatalf("score cast must not remain bigint after decay migration: %s", sqlCursor)
	}
	if strings.Contains(sqlCursor, "EXISTS(SELECT 1 FROM moment_favorites") {
		t.Fatalf("anonymous search must omit viewer favorites column: %s", sqlCursor)
	}
}

// TestScanSearchMomentRowReadsViewerFavoriteFlag：搜索行扫描读取尾列收藏态。
func TestScanSearchMomentRowReadsViewerFavoriteFlag(t *testing.T) {
	base := []any{"moment_1", "user_a", "user_a", "desc", "2026-01-01T00:00:00Z", 0, 2, 0, "somewhere", 35.0, 112.0, "public", false}
	item, err := scanSearchMomentRow(rowValues{values: append(append([]any{}, base...), true)})
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if !item.IsFavorited {
		t.Fatalf("expected viewer is_favorited=true, got %#v", item)
	}
	item, err = scanSearchMomentRow(rowValues{values: append(append([]any{}, base...), false)})
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if item.IsFavorited {
		t.Fatalf("expected viewer is_favorited=false, got %#v", item)
	}
}

// TestMemorySearchFillsViewerFavoriteState：内存实现与 Postgres 同语义——登录查看者
// 按收藏记录填充 is_favorited，匿名恒 false。
func TestMemorySearchFillsViewerFavoriteState(t *testing.T) {
	repo := NewInMemoryRepository()
	now := time.Now().UTC()
	seedSearchMoment(t, repo, "moment_1", "alpha post", now.Add(-time.Hour), 0, 0)
	seedSearchMoment(t, repo, "moment_2", "beta post", now.Add(-2*time.Hour), 0, 0)
	if err := repo.Favorite(context.Background(), "viewer", "moment_1"); err != nil {
		t.Fatalf("favorite: %v", err)
	}

	authed := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "viewer"})
	result, err := repo.Search(authed, SearchMomentsRequest{Keyword: strPtrMoments("post")}, "viewer")
	if err != nil {
		t.Fatalf("authed search: %v", err)
	}
	if len(result.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(result.Items))
	}
	byID := map[string]bool{}
	for _, it := range result.Items {
		byID[it.MomentID] = it.IsFavorited
	}
	if !byID["moment_1"] || byID["moment_2"] {
		t.Fatalf("expected only favorited moment flagged, got %#v", byID)
	}

	anon, err := repo.Search(context.Background(), SearchMomentsRequest{Keyword: strPtrMoments("post")}, "")
	if err != nil {
		t.Fatalf("anonymous search: %v", err)
	}
	for _, it := range anon.Items {
		if it.IsFavorited {
			t.Fatalf("anonymous viewer must always see is_favorited=false, got %#v", it)
		}
	}
}

func strPtrMoments(s string) *string { return &s }
