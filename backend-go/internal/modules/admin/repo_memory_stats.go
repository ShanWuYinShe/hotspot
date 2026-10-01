package admin

import (
	"context"
	"sort"
	"time"

	"hotspot/backend-go/internal/platform/geo"
)

// StatsGeo 内存实现：聚合 adminUsers/groups 夹具数据（moments 夹具无位置字段，
// 恒计 0）。时间窗在此实现中不生效（夹具近似）——时间窗 SQL 语义由
// 真库集成测试（repo_postgres_stats_it_test.go）覆盖。
// 排名/合并/Top-N 语义与 postgres 实现一致，供 Service 单测断言。
func (r *InMemoryRepository) StatsGeo(_ context.Context, query geoStatsQuery) (AdminGeoStatsResponse, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	resp := AdminGeoStatsResponse{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Regions:     []AdminRegionRankItem{},
		Places:      []AdminPlaceRankItem{},
	}

	buckets := map[string]*geoBucketCount{}
	names := map[string]*geoBucketCount{}
	zeroPoint := func(lat, lng float64) bool { return lat == 0 && lng == 0 }

	addCount := func(b *geoBucketCount, kind string) {
		b.Total++
		switch kind {
		case "user":
			b.User++
		case "group":
			b.Group++
		case "moment":
			b.Moment++
		}
	}
	regionBucket := func(lat, lng float64) *geoBucketCount {
		token := geo.CellTokenAtLevel(geo.Point{Latitude: lat, Longitude: lng}, query.Level)
		b, ok := buckets[token]
		if !ok {
			center := geo.CellCenter(token)
			b = &geoBucketCount{Key: token, Center: GeoPoint{Latitude: center.Latitude, Longitude: center.Longitude}}
			buckets[token] = b
		}
		return b
	}
	placeBucket := func(name string) *geoBucketCount {
		b, ok := names[name]
		if !ok {
			b = &geoBucketCount{Key: name}
			names[name] = b
		}
		return b
	}

	for _, user := range r.adminUsers {
		if user.Location == nil || zeroPoint(user.Location.Point.Latitude, user.Location.Point.Longitude) {
			continue
		}
		lat, lng := user.Location.Point.Latitude, user.Location.Point.Longitude
		addCount(regionBucket(lat, lng), "user")
		if name := user.Location.LocationName; name != "" {
			addCount(placeBucket(name), "user")
		}
	}
	for _, group := range r.groups {
		lat, lng := group.Location.Point.Latitude, group.Location.Point.Longitude
		if zeroPoint(lat, lng) {
			continue
		}
		addCount(regionBucket(lat, lng), "group")
		if name := group.Location.LocationName; name != "" {
			addCount(placeBucket(name), "group")
		}
	}

	resp.Totals = AdminGeoStatsTotals{
		User:   len(r.adminUsers),
		Group:  len(r.groups),
		Moment: 0,
		Total:  len(r.adminUsers) + len(r.groups),
	}

	// Top-N + 确定性排序（计数降序、key 升序 tie-breaker），与 SQL ORDER BY 对齐。
	collect := func(src map[string]*geoBucketCount, limit int) []*geoBucketCount {
		out := make([]*geoBucketCount, 0, len(src))
		for _, b := range src {
			out = append(out, b)
		}
		sort.Slice(out, func(i, j int) bool {
			if out[i].Total != out[j].Total {
				return out[i].Total > out[j].Total
			}
			return out[i].Key < out[j].Key
		})
		if len(out) > limit {
			out = out[:limit]
		}
		return out
	}

	for _, b := range collect(buckets, query.Limit) {
		resp.Regions = append(resp.Regions, AdminRegionRankItem{
			CellToken:  b.Key,
			Level:      query.Level,
			Center:     b.Center,
			TotalCount: b.Total,
			Breakdown:  StatsEntityBreakdown{User: b.User, Group: b.Group, Moment: b.Moment},
		})
	}
	for _, b := range collect(names, query.Limit) {
		resp.Places = append(resp.Places, AdminPlaceRankItem{
			LocationName: b.Key,
			TotalCount:   b.Total,
			Breakdown:    StatsEntityBreakdown{User: b.User, Group: b.Group, Moment: b.Moment},
		})
	}
	return resp, nil
}
