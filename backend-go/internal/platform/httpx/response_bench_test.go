package httpx

import (
	"net/http/httptest"
	"testing"
)

type benchMomentItem struct {
	MomentID    string    `json:"moment_id"`
	User        benchUser `json:"user"`
	Description string    `json:"description"`
	CreatedAt   string    `json:"created_at"`
	Images      []string  `json:"images"`
}

type benchUser struct {
	UserID   string `json:"user_id"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar_url"`
}

func benchPage() []benchMomentItem {
	items := make([]benchMomentItem, 20)
	for i := range items {
		items[i] = benchMomentItem{
			MomentID:    "moment_bench_0001",
			User:        benchUser{UserID: "user_bench_a", Nickname: "压测用户", Avatar: "https://cdn.example.com/a.jpg"},
			Description: "用于压测统一响应 envelope 编码开销的瞬间正文内容，长度取真实场景中位数附近。",
			CreatedAt:   "2026-01-01T00:00:00.000000Z",
			Images:      []string{"https://cdn.example.com/1.jpg", "https://cdn.example.com/2.jpg"},
		}
	}
	return items
}

// BenchmarkWriteJSONPage 测列表响应（20 条瞬间）的 envelope 编码成本：
// 统一 WriteJSON 是每个成功响应的必经路径，关注 alloc 与 ns/op 回归。
func BenchmarkWriteJSONPage(b *testing.B) {
	page := Success(benchPage())
	rr := httptest.NewRecorder()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr.Body.Reset()
		WriteJSON(rr, 200, page)
	}
}
