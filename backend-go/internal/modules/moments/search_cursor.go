package moments

import (
	"time"

	"hotspot/backend-go/internal/platform/cursor"
)

const momentSearchCursorNamespace = "moments.search"

// searchDecayBucket 是 popular 热度衰减时钟的量化粒度：衰减时钟取
// floor(epoch/bucket)*bucket（UTC 绝对对齐），桶内排序分值恒定，
// 使 popular 分页在同一桶内严格一致（消除实时衰减的跨请求漂移窗口）。
// 与 searchScoreSQL 里参数化的 FLOOR($clock/900)*900 对齐（$clock 由 Search
// 传入同一量化时钟，SQL 不使用 now()）。
const searchDecayBucket = 15 * time.Minute

// SearchDecayClock 把查询时钟量化到衰减桶：排序与游标必须使用同一量化时钟。
// 排序分值本身由 SQL 计算并随行返回（sort_score），Go 侧不重算。
func SearchDecayClock(now time.Time) time.Time {
	return now.Truncate(searchDecayBucket)
}

// momentSearchCursor 是 moments/search 的 keyset 游标载荷（经 cursor.Encode 加密，
// 对客户端不透明）。Score 为排序分值：relevance/newest 为整数语义，popular 含
// 时间衰减为浮点（spec api_spec_moment.md moments/search 排序）。
type momentSearchCursor struct {
	Sort  string  `json:"sort"`
	Score float64 `json:"score"`
	At    string  `json:"at"`
	ID    string  `json:"id"`
}

func normalizeMomentSearchSort(sortBy string) string {
	switch sortBy {
	case "newest", "popular":
		return sortBy
	default:
		return "relevance"
	}
}

// encodeMomentSearchCursorSQL 编码 moments/search 的 keyset 游标：分值与时间
// 必须直接来自本次 SQL 查询的实际排序输出（SELECT ... AS sort_score 与全精度
// created_at），不得由 Go 侧用近似公式重算——popular 的浮点分值在不同
// IEEE-754 运算路径下存在 ulp 级差异，relevance/newest 若用秒级展示时间则与
// 微秒级排序列不等，二者都会让下一页元组比较的首分量对不上，造成页边界重复
// 或整段数据被跳过（审计 R65 在 dev 环境实测复现）。
func encodeMomentSearchCursorSQL(sortBy string, score float64, at time.Time, momentID string) string {
	value, err := cursor.Encode(momentSearchCursorNamespace, momentSearchCursor{
		Sort:  normalizeMomentSearchSort(sortBy),
		Score: score,
		At:    at.UTC().Format(time.RFC3339Nano),
		ID:    momentID,
	})
	if err != nil {
		return ""
	}
	return value
}

func decodeMomentSearchCursor(value string) (*momentSearchCursor, error) {
	if value == "" {
		return nil, nil
	}
	var decoded momentSearchCursor
	if err := cursor.Decode(momentSearchCursorNamespace, value, &decoded); err != nil {
		return nil, ErrInvalidCursor
	}
	if decoded.Sort == "" || decoded.At == "" || decoded.ID == "" {
		return nil, ErrInvalidCursor
	}
	if _, err := time.Parse(time.RFC3339Nano, decoded.At); err != nil {
		return nil, ErrInvalidCursor
	}
	return &decoded, nil
}
