package admin

import (
	"context"
	"sort"
	"time"

	"hotspot/backend-go/internal/platform/geo"
	"hotspot/backend-go/internal/platform/types"
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
		Trends:      []AdminGeoStatsTrendPoint{},
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
	if query.TrendDays > 0 {
		// 每日新增：按 UTC+8 日界分桶（postgres 同口径）；夹具 CreatedAt 为
		// RFC3339 字符串，解析失败的行不计入。
		sh := time.FixedZone("UTC+8", 8*60*60)
		byDay := map[string]geoBucketCount{}
		bucketDay := func(raw string, kind string) {
			t, err := time.Parse(time.RFC3339, raw)
			if err != nil {
				return
			}
			key := t.In(sh).Format("2006-01-02")
			b := byDay[key]
			b.Total++
			switch kind {
			case "user":
				b.User++
			case "group":
				b.Group++
			case "moment":
				b.Moment++
			}
			byDay[key] = b
		}
		for _, user := range r.adminUsers {
			bucketDay(user.CreatedAt, "user")
		}
		for _, group := range r.groups {
			bucketDay(group.CreatedAt, "group")
		}
		resp.Trends = fillTrendDays(byDay, query.TrendStart, query.TrendDays)
	}
	return resp, nil
}

// StatsGeoDrilldown 内存实现：聚合 adminUsers/groups 夹具（moments 夹具无位置，
// 恒空页）。排序/cursor 语义与 postgres 实现对齐（created_at DESC + id tie-break）。
func (r *InMemoryRepository) StatsGeoDrilldown(_ context.Context, query geoDrilldownQuery) (geoDrilldownPage, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	page := geoDrilldownPage{Items: []AdminGeoDrilldownItem{}}
	target := geo.BucketCellToken(query.Bucket, query.Level)
	if target == "" {
		return page, nil
	}
	zeroPoint := func(lat, lng float64) bool { return lat == 0 && lng == 0 }
	type entry struct {
		item      AdminGeoDrilldownItem
		createdAt time.Time
	}
	entries := make([]entry, 0, 16)
	parse := func(raw string) time.Time {
		t, _ := time.Parse(time.RFC3339, raw)
		return t
	}
	for id, user := range r.adminUsers {
		if user.Location == nil || zeroPoint(user.Location.Point.Latitude, user.Location.Point.Longitude) {
			continue
		}
		if query.EntityType != "user" || geo.CellTokenAtLevel(geo.Point{Latitude: user.Location.Point.Latitude, Longitude: user.Location.Point.Longitude}, query.Level) != target {
			continue
		}
		entries = append(entries, entry{
			item: AdminGeoDrilldownItem{
				ObjectID:     id,
				Title:        user.User.Nickname,
				LocationName: types.StrPtr(user.Location.LocationName),
				Latitude:     user.Location.Point.Latitude,
				Longitude:    user.Location.Point.Longitude,
				CreatedAt:    user.CreatedAt,
			},
			createdAt: parse(user.CreatedAt),
		})
	}
	for id, group := range r.groups {
		if zeroPoint(group.Location.Point.Latitude, group.Location.Point.Longitude) {
			continue
		}
		if query.EntityType != "group" || geo.CellTokenAtLevel(geo.Point{Latitude: group.Location.Point.Latitude, Longitude: group.Location.Point.Longitude}, query.Level) != target {
			continue
		}
		entries = append(entries, entry{
			item: AdminGeoDrilldownItem{
				ObjectID:     id,
				Title:        group.Group.Name,
				LocationName: types.StrPtr(group.Location.LocationName),
				Latitude:     group.Location.Point.Latitude,
				Longitude:    group.Location.Point.Longitude,
				CreatedAt:    group.CreatedAt,
			},
			createdAt: parse(group.CreatedAt),
		})
	}
	sort.Slice(entries, func(i, j int) bool {
		if !entries[i].createdAt.Equal(entries[j].createdAt) {
			return entries[i].createdAt.After(entries[j].createdAt)
		}
		return entries[i].item.ObjectID > entries[j].item.ObjectID
	})
	start := 0
	if query.Cursor != "" {
		c, apiErr := decodeGeoDrilldownCursor(query.Cursor)
		if apiErr != nil {
			return geoDrilldownPage{}, apiErr
		}
		for idx, e := range entries {
			if e.createdAt.Equal(c.CreatedAt) && e.item.ObjectID == c.ObjectID {
				start = idx + 1
				break
			}
			if e.createdAt.Before(c.CreatedAt) || (e.createdAt.Equal(c.CreatedAt) && e.item.ObjectID < c.ObjectID) {
				start = idx
				break
			}
		}
	}
	end := min(start+query.Limit, len(entries))
	for _, e := range entries[start:end] {
		page.Items = append(page.Items, e.item)
	}
	if end < len(entries) && end > start {
		last := page.Items[len(page.Items)-1]
		t, _ := time.Parse(time.RFC3339, last.CreatedAt)
		page.Next = &geoDrilldownCursor{CreatedAt: t, ObjectID: last.ObjectID}
	}
	return page, nil
}
