package moments

// 本文件保留 moments/search 的 Go 侧打分/游标编码实现，仅供内存仓储（测试
// 构建）使用：生产 Postgres 路径的排序分值与游标时间直接来自 SQL 输出
// （sort_score + 全精度 created_at，见 search_cursor.go 的
// encodeMomentSearchCursorSQL），不再由 Go 侧重算——两套运算路径的浮点/
// 精度差异会造成 keyset 页边界重复或丢行（审计 R65）。
//
// 内存实现与 Postgres 实现的语义对齐由契约测试覆盖；此处公式与
// searchScoreSQL 的单项逐项对应，内存实现内部排序与游标编码同源，因此
// 内存侧翻页自洽。

import (
	"math"
	"time"

	"hotspot/backend-go/internal/platform/cursor"
)

// momentSearchScore is the keyset score used by the in-memory moment repository
// (test build). Production search uses the SQL-produced sort_score column.
//
//	score = interaction_count / POWER(发布小时龄 + 2, 1.5)
func momentSearchScore(item MomentItem, sortBy, keyword, tag string, now time.Time) float64 {
	switch normalizeMomentSearchSort(sortBy) {
	case "popular":
		raw := float64(item.InteractionCount)
		hours := 0.0
		if createdAt, err := time.Parse(time.RFC3339Nano, item.CreatedAt); err == nil {
			hours = now.Sub(createdAt).Hours()
			if hours < 0 {
				// 未来时间戳（时钟偏移）钳制为 0，与 SQL 的 GREATEST(..., 0.0) 对齐。
				hours = 0
			}
		}
		return raw / math.Pow(hours+2, 1.5)
	case "newest":
		return 0
	default:
		score := float64(0)
		if keyword != "" && containsFold(item.Description, keyword) {
			score++
		}
		if tag != "" && containsTag(item.Tags, tag) {
			score++
		}
		return score
	}
}

// encodeMomentSearchCursor 编码内存实现（测试构建）的 keyset 游标。now 为本次
// 查询的统一时钟，保证边界项分值与排序时使用的衰减分值一致（popular 模式）。
func encodeMomentSearchCursor(sortBy string, item MomentItem, keyword, tag string, now time.Time) string {
	value, err := cursor.Encode(momentSearchCursorNamespace, momentSearchCursor{
		Sort:  normalizeMomentSearchSort(sortBy),
		Score: momentSearchScore(item, sortBy, keyword, tag, now),
		At:    item.CreatedAt,
		ID:    item.MomentID,
	})
	if err != nil {
		return ""
	}
	return value
}
