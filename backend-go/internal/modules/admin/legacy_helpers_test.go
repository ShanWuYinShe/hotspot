package admin

import (
	"time"
)

// 以下辅助从生产代码下沉到测试：生产路径已不使用，仅测试替身实现需要。

func fillTrendDays(rows map[string]geoBucketCount, start time.Time, days int) []AdminGeoStatsTrendPoint {
	sh := time.FixedZone("UTC+8", 8*60*60)
	out := make([]AdminGeoStatsTrendPoint, 0, days)
	dayStart := time.Date(start.In(sh).Year(), start.In(sh).Month(), start.In(sh).Day(), 0, 0, 0, 0, sh)
	for i := 0; i < days; i++ {
		key := dayStart.AddDate(0, 0, i).Format("2006-01-02")
		b := rows[key]
		out = append(out, AdminGeoStatsTrendPoint{Date: key, User: b.User, Group: b.Group, Moment: b.Moment})
	}
	return out
}

func limitPageItems[T any](items []T, limit int) ([]T, bool) {
	if len(items) <= limit {
		return items, false
	}
	return items[:limit], true
}

type geoBucketCount struct {
	Key    string // 区域维度为 S2 token；地点维度为 location_name
	Center GeoPoint
	Total  int
	User   int
	Group  int
	Moment int
}
