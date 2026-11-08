package share

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// shareStaticGetRepo 返回固定 ShareDetail，用于覆盖 OG 页面在 preview 缺失/
// cover 存在时的回退与摘要分支。
type shareStaticGetRepo struct {
	*shareStubRepo
	detail ShareDetail
}

func (r *shareStaticGetRepo) Get(context.Context, getShareQuery) (ShareDetail, error) {
	return r.detail, nil
}

// TestServeCrawlerHTMLFallbackBranches 覆盖 preview 缺失时的类型标签回退描述，
// 以及 Preview.CoverURL 非空时输出 og:image。
func TestServeCrawlerHTMLFallbackBranches(t *testing.T) {
	fallback := NewHandler(NewServiceWithRepo(&shareStaticGetRepo{
		shareStubRepo: &shareStubRepo{},
		detail: ShareDetail{
			Share: ShareBase{ShareID: "share_1", ShareCode: "ABCDEFGHJKLM", ShareType: ShareTypeUser, Creator: ShareUserInfo{UserID: "user_bob", Nickname: "Bob"}},
		},
	}))
	rr := httptest.NewRecorder()
	fallback.ServeCrawlerHTML(rr, ogCrawlerRequest("ABCDEFGHJKLM"))
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "用户主页 · 来自 Bob") {
		t.Fatalf("missing type-label fallback description: %s", body)
	}
	if strings.Contains(body, "og:image") {
		t.Fatalf("unexpected og:image without cover: %s", body)
	}

	cover := "https://example.com/cover.png"
	subtitle := "摘要副标题"
	withPreview := NewHandler(NewServiceWithRepo(&shareStaticGetRepo{
		shareStubRepo: &shareStubRepo{},
		detail: ShareDetail{
			Share:   ShareBase{ShareID: "share_2", ShareCode: "ABCDEFGHJKMN", ShareType: ShareTypeMoment, Creator: ShareUserInfo{UserID: "user_bob", Nickname: "Bob"}},
			Preview: &SharePreview{Title: "摘要标题", Subtitle: &subtitle, CoverURL: &cover},
		},
	}))
	rr = httptest.NewRecorder()
	withPreview.ServeCrawlerHTML(rr, ogCrawlerRequest("ABCDEFGHJKMN"))
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	body = rr.Body.String()
	for _, want := range []string{"摘要标题", "摘要副标题", "og:image", "https://example.com/cover.png"} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in crawler page: %s", want, body)
		}
	}
}
