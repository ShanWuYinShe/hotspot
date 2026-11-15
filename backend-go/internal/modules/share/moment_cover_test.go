package share

import (
	"context"
	"encoding/json"
	"errors"
	"html"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// 本文件锁定「moment 分享封面用瞬间首图」的产品决策（原实现恒为作者头像）：
// 仓储把该瞬间第一张已完成媒体的 storage key 带在 preview 上（未导出、不进
// JSON），Service 在 Get/List 返回前 presign 成绝对读 URL 覆盖 cover_url；
// 无图（纯文字瞬间）、未注入 resolver、签名失败时保留作者头像兜底。

// shareStubCoverResolver 记录 Service 传给 MomentCoverResolver 的参数，返回
// 固定签名 URL；err 非空时模拟对象存储不可用。
type shareStubCoverResolver struct {
	keys []string
	ttls []time.Duration
	url  string
	err  error
}

func (r *shareStubCoverResolver) PresignGet(_ context.Context, storageKey string, ttl time.Duration) (string, error) {
	r.keys = append(r.keys, storageKey)
	r.ttls = append(r.ttls, ttl)
	if r.err != nil {
		return "", r.err
	}
	return r.url, nil
}

func strPtrTest(value string) *string { return &value }

func TestShareServiceGetSignsMomentCover(t *testing.T) {
	const (
		avatar  = "https://example.com/avatar.png"
		key     = "uploads/user_bob/first.jpg"
		signed  = "https://storage.example.com/bucket/uploads/user_bob/first.jpg?X-Amz-Signature=abc"
		shareID = "share_moment_cover"
	)
	resolver := &shareStubCoverResolver{url: signed}
	svc := NewServiceWithRepo(&shareStaticGetRepo{
		shareStubRepo: &shareStubRepo{},
		detail: ShareDetail{
			Share:   ShareBase{ShareID: shareID, ShareCode: "ABCDEFGHJKLM", ShareType: ShareTypeMoment, Creator: ShareUserInfo{UserID: "user_bob", Nickname: "Bob"}},
			Preview: &SharePreview{Title: "标题", CoverURL: strPtrTest(avatar), coverStorageKey: key},
		},
	})
	svc.SetCoverResolver(resolver)

	detail, err := svc.Get(context.Background(), GetShareRequest{ShareCode: "ABCDEFGHJKLM"})
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if detail.Preview == nil || detail.Preview.CoverURL == nil || *detail.Preview.CoverURL != signed {
		t.Fatalf("cover_url 必须是瞬间首图的签名 URL: %#v", detail.Preview)
	}
	if len(resolver.keys) != 1 || resolver.keys[0] != key {
		t.Fatalf("resolver 入参 = %#v，期望 [%s]", resolver.keys, key)
	}
	if resolver.ttls[0] != shareMomentCoverTTL || shareMomentCoverTTL != 7*24*time.Hour {
		t.Fatalf("封面 TTL = %v，期望 7 天（与 moments 读 URL 一致）", resolver.ttls[0])
	}
}

// TestShareServiceGetMomentCoverFallback：签名失败或未注入 resolver 时保留
// 头像兜底且不影响分享解析（封面是展示增强，不能变成请求失败）。
func TestShareServiceGetMomentCoverFallback(t *testing.T) {
	const avatar = "https://example.com/avatar.png"

	cases := []struct {
		name     string
		resolver MomentCoverResolver
	}{
		{"presign error", &shareStubCoverResolver{err: errors.New("storage down")}},
		{"resolver missing", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := NewServiceWithRepo(&shareStaticGetRepo{
				shareStubRepo: &shareStubRepo{},
				detail: ShareDetail{
					Share:   ShareBase{ShareID: "share_1", ShareCode: "ABCDEFGHJKLM", ShareType: ShareTypeMoment, Creator: ShareUserInfo{UserID: "user_bob", Nickname: "Bob"}},
					Preview: &SharePreview{Title: "标题", CoverURL: strPtrTest(avatar), coverStorageKey: "uploads/user_bob/first.jpg"},
				},
			})
			if tc.resolver != nil {
				svc.SetCoverResolver(tc.resolver)
			}
			detail, err := svc.Get(context.Background(), GetShareRequest{ShareCode: "ABCDEFGHJKLM"})
			if err != nil {
				t.Fatalf("封面签名失败不得让 Get 失败: %v", err)
			}
			if detail.Preview == nil || detail.Preview.CoverURL == nil || *detail.Preview.CoverURL != avatar {
				t.Fatalf("应回退作者头像，实际 %#v", detail.Preview)
			}
		})
	}
}

// TestShareServiceGetTextOnlyMomentKeepsAvatar：纯文字瞬间（仓储未带出封面
// key）不调用 resolver，头像兜底原样返回。
func TestShareServiceGetTextOnlyMomentKeepsAvatar(t *testing.T) {
	const avatar = "https://example.com/avatar.png"
	resolver := &shareStubCoverResolver{url: "https://storage.example.com/signed.jpg"}
	svc := NewServiceWithRepo(&shareStaticGetRepo{
		shareStubRepo: &shareStubRepo{},
		detail: ShareDetail{
			Share:   ShareBase{ShareID: "share_1", ShareCode: "ABCDEFGHJKLM", ShareType: ShareTypeMoment, Creator: ShareUserInfo{UserID: "user_bob", Nickname: "Bob"}},
			Preview: &SharePreview{Title: "纯文字", CoverURL: strPtrTest(avatar)},
		},
	})
	svc.SetCoverResolver(resolver)

	detail, err := svc.Get(context.Background(), GetShareRequest{ShareCode: "ABCDEFGHJKLM"})
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if detail.Preview == nil || detail.Preview.CoverURL == nil || *detail.Preview.CoverURL != avatar {
		t.Fatalf("纯文字瞬间应保留头像封面: %#v", detail.Preview)
	}
	if len(resolver.keys) != 0 {
		t.Fatalf("无封面 key 时不应调用 resolver: %#v", resolver.keys)
	}
}

// TestShareServiceListSignsMomentCovers：「我的分享」列表与单条入口同口径——
// 有图的 moment 项签名，纯文字/用户项保持不变，preview 为 nil 的项安全跳过。
func TestShareServiceListSignsMomentCovers(t *testing.T) {
	const (
		avatar   = "https://example.com/avatar.png"
		groupImg = "https://example.com/group.png"
		signed   = "https://storage.example.com/signed-first.jpg"
	)
	resolver := &shareStubCoverResolver{url: signed}
	svc := NewServiceWithRepo(&shareStaticGetRepo{
		shareStubRepo: &shareStubRepo{},
		list: PaginatedShares{Items: []ShareListItem{
			{ShareID: "share_moment_cover", ShareType: ShareTypeMoment, Preview: &SharePreview{Title: "带图", CoverURL: strPtrTest(avatar), coverStorageKey: "uploads/user_bob/first.jpg"}},
			{ShareID: "share_moment_text", ShareType: ShareTypeMoment, Preview: &SharePreview{Title: "纯文字", CoverURL: strPtrTest(avatar)}},
			{ShareID: "share_group", ShareType: ShareTypeGroup, Preview: &SharePreview{Title: "群组", CoverURL: strPtrTest(groupImg)}},
			{ShareID: "share_unavailable", ShareType: ShareTypeMoment, Preview: nil},
		}},
	})
	svc.SetCoverResolver(resolver)

	page, err := svc.List(shareContextWithUser("user_bob"), ListSharesRequest{})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(page.Items) != 4 {
		t.Fatalf("items = %d", len(page.Items))
	}
	if got := page.Items[0].Preview.CoverURL; got == nil || *got != signed {
		t.Fatalf("列表 moment 封面 = %v，期望签名 URL", got)
	}
	if got := page.Items[1].Preview.CoverURL; got == nil || *got != avatar {
		t.Fatalf("纯文字瞬间封面 = %v，期望头像兜底", got)
	}
	if got := page.Items[2].Preview.CoverURL; got == nil || *got != groupImg {
		t.Fatalf("群组分享封面不得被改写 = %v", got)
	}
	if page.Items[3].Preview != nil {
		t.Fatalf("nil preview 必须保持 nil: %#v", page.Items[3].Preview)
	}
	if len(resolver.keys) != 1 || resolver.keys[0] != "uploads/user_bob/first.jpg" {
		t.Fatalf("列表只应对带封面的 moment 调用一次 resolver: %#v", resolver.keys)
	}
}

// TestSharePreviewCoverStorageKeyNotSerialized：coverStorageKey 是仓储→Service
// 的内部字段，绝不能出现在对外 JSON 里（契约 SharePreview 仅 3 个字段）。
func TestSharePreviewCoverStorageKeyNotSerialized(t *testing.T) {
	cover := "https://example.com/avatar.png"
	data, err := json.Marshal(SharePreview{Title: "标题", CoverURL: &cover, coverStorageKey: "uploads/user_bob/first.jpg"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(data), "coverStorageKey") || strings.Contains(string(data), "uploads/user_bob") {
		t.Fatalf("内部 coverStorageKey 泄露进 JSON: %s", data)
	}
	var fields map[string]any
	if err := json.Unmarshal(data, &fields); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(fields) != 3 || fields["cover_url"] == nil {
		t.Fatalf("SharePreview JSON 字段 = %#v，期望仅 title/subtitle/cover_url", fields)
	}
}

// TestServeCrawlerHTMLUsesSignedMomentCover：OG 落地页的 og:image 必须是注入
// resolver 后的签名绝对 URL（爬虫只抓一次、不解析相对路径），头像兜底不再出现。
func TestServeCrawlerHTMLUsesSignedMomentCover(t *testing.T) {
	const (
		avatar = "/public/v1/media/avatars/file_avatar"
		signed = "https://storage.example.com/bucket/uploads/user_bob/first.jpg?X-Amz-Signature=abc&X-Amz-Expires=604800"
	)
	svc := NewServiceWithRepo(&shareStaticGetRepo{
		shareStubRepo: &shareStubRepo{},
		detail: ShareDetail{
			Share:   ShareBase{ShareID: "share_1", ShareCode: "ABCDEFGHJKLM", ShareType: ShareTypeMoment, Creator: ShareUserInfo{UserID: "user_bob", Nickname: "Bob"}},
			Preview: &SharePreview{Title: "标题", CoverURL: strPtrTest(avatar), coverStorageKey: "uploads/user_bob/first.jpg"},
		},
	})
	svc.SetCoverResolver(&shareStubCoverResolver{url: signed})
	handler := NewHandler(svc)

	rr := httptest.NewRecorder()
	handler.ServeCrawlerHTML(rr, ogPageRequest("ABCDEFGHJKLM", "hotspot.example.com", "https"))
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	// 签名 URL 是绝对 URL：absoluteURL 原样保留（不拼当前 Host）；HTML 属性值
	// 中的 & 会按 html.EscapeString 转义成 &amp;（浏览器/爬虫解析回 &）。
	if want := `<meta property="og:image" content="` + html.EscapeString(signed) + `">`; !strings.Contains(body, want) {
		t.Fatalf("og:image 必须是签名绝对 URL %q: %s", want, body)
	}
	if strings.Contains(body, avatar) {
		t.Fatalf("有瞬间图时 og:image 不得回退头像: %s", body)
	}
}
