package moments

import (
	"math"
	"strings"
	"time"

	"hotspot/backend-go/internal/platform/cursor"
)

const momentSearchCursorNamespace = "moments.search"

// searchDecayBucket 是 popular 热度衰减时钟的量化粒度：衰减时钟取
// floor(epoch/bucket)*bucket（UTC 绝对对齐），桶内排序/游标分值恒定，
// 使 popular 分页在同一桶内严格一致（消除实时衰减的跨请求漂移窗口）。
// 与 searchScoreSQL 里参数化的 FLOOR($clock/900)*900 对齐（$clock 由 Search
// 传入同一量化时钟，SQL 不使用 now()）。
const searchDecayBucket = 15 * time.Minute

// SearchDecayClock 把查询时钟量化到衰减桶：排序、游标编码、内存打分
// 必须使用同一量化时钟，保证 Go 游标分值与 Postgres ORDER BY 分值一致。
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

// momentSearchScore is the keyset score used by all three moment search sort
// modes. relevance counts keyword/tag matches; newest keeps a constant score so
// the keyset degenerates to (created_at, moment_id); popular applies a time
// decay factor（spec api_spec_moment.md：热度排序含时间衰减因子，算法为实现细节）：
//
//	score = (reaction 计数列 + comments_count) / POWER(发布小时龄 + 2, 1.5)
//
// 与 searchScoreSQL 的 Postgres 表达式逐项对应；now 由调用方在单次查询内取同一
// 量化值（SearchDecayClock），并作为 SQL 参数下传，保证排序、游标谓词与游标编码
// 三者同源。
func momentSearchScore(item MomentItem, sortBy, keyword, tag string, now time.Time) float64 {
	switch normalizeMomentSearchSort(sortBy) {
	case "popular":
		raw := float64(item.LikesCount + item.CommentsCount)
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

// compareSearchTuple 比较 keyset 元组 (score, created_at, moment_id) 与解码游标
// （全 DESC 排序语义）：返回 <0 表示该元素严格排在游标之后，即满足 SQL 谓词
// `(score, created_at, moment_id) < (cursor.Score, cursor.At, cursor.ID)`。
// 内存实现据此定位下一页起点，不依赖边界项仍存在（与 Postgres keyset 同语义）。
func compareSearchTuple(score float64, at, id string, cur momentSearchCursor) int {
	if score != cur.Score {
		if score < cur.Score {
			return -1
		}
		return 1
	}
	if at != cur.At {
		if at < cur.At {
			return -1
		}
		return 1
	}
	return strings.Compare(id, cur.ID)
}

// encodeMomentSearchCursor 编码 keyset 游标。now 为本次查询的统一时钟，保证
// 边界项分值与排序时使用的衰减分值一致（popular 模式）。
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
