package mapmodule

import (
	"sort"

	"hotspot/backend-go/internal/platform/geo"
)

// 以下辅助从生产代码下沉到测试：生产路径已不使用，仅测试替身实现需要。

func fallbackHeatBucketCenter(token string, fallback GeoPoint) GeoPoint {
	center := geo.CellCenter(token)
	if center.Latitude == 0 && center.Longitude == 0 {
		return fallback
	}
	return GeoPoint{Latitude: center.Latitude, Longitude: center.Longitude}
}

func aggregateHeatBuckets(items []mapCandidate, level int) []MapHeatBucket {
	byToken := map[string]*MapHeatBucket{}
	members := map[string][]geo.Point{}
	for _, item := range items {
		point := geo.Point{Latitude: item.Location.Latitude, Longitude: item.Location.Longitude}
		token := geo.CellTokenAtLevel(point, level)
		bucket, ok := byToken[token]
		if !ok {
			bucket = &MapHeatBucket{CellToken: token, Level: level, Location: item.Location, Breakdown: &MapHeatBreakdown{}}
			byToken[token] = bucket
		}
		members[token] = append(members[token], point)
		bucket.TotalCount++
		switch item.ObjectType {
		case "user":
			if bucket.Breakdown.User == nil {
				zero := 0
				bucket.Breakdown.User = &zero
			}
			*bucket.Breakdown.User++
		case "group":
			if bucket.Breakdown.Group == nil {
				zero := 0
				bucket.Breakdown.Group = &zero
			}
			*bucket.Breakdown.Group++
		case "moment":
			if bucket.Breakdown.Moment == nil {
				zero := 0
				bucket.Breakdown.Moment = &zero
			}
			*bucket.Breakdown.Moment++
		}
	}
	out := make([]MapHeatBucket, 0, len(byToken))
	for token, bucket := range byToken {
		if center, ok := geo.SphericalCentroid(members[token]); ok {
			bucket.Location = GeoPoint{Latitude: center.Latitude, Longitude: center.Longitude}
		}
		out = append(out, *bucket)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].TotalCount != out[j].TotalCount {
			return out[i].TotalCount > out[j].TotalCount
		}
		return out[i].CellToken < out[j].CellToken
	})
	if len(out) > 200 {
		out = out[:200]
	}
	return out
}
