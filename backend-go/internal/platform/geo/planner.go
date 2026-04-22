package geo

import "math"

type SearchPlan struct {
	Reference Point
	Radius    int
	SortBy    string
	Level     int
	ranges    [][2]int64
}

func NewSearchPlan(reference Point, radius int, sortBy string) SearchPlan {
	if radius <= 0 {
		radius = 5000
	}
	if radius > 50000 {
		radius = 50000
	}
	bounds := BoundsAround(reference, radius)
	ranges := SearchCovering(bounds)
	level := SuggestedLevel(float64(radius * 2))
	if sortBy == "" {
		sortBy = "relevance"
	}
	return SearchPlan{Reference: reference, Radius: radius, SortBy: sortBy, Level: level, ranges: ranges}
}

func BoundsAround(reference Point, radiusMeters int) Bounds {
	latDelta := float64(radiusMeters) / 111000.0
	cosLat := math.Cos(reference.Latitude * math.Pi / 180)
	if math.Abs(cosLat) < 0.01 {
		cosLat = 0.01
	}
	lngDelta := float64(radiusMeters) / (111000.0 * cosLat)
	return Bounds{
		North: clamp(reference.Latitude+latDelta, -90, 90),
		South: clamp(reference.Latitude-latDelta, -90, 90),
		East:  reference.Longitude + lngDelta,
		West:  reference.Longitude - lngDelta,
	}
}

func (p SearchPlan) Match(candidate Point) (distance int, bearing int, ok bool) {
	cellID := LeafCellID(candidate)
	for _, r := range p.ranges {
		if cellID >= r[0] && cellID <= r[1] {
			distance = DistanceMeters(p.Reference, candidate)
			if distance > p.Radius {
				return 0, 0, false
			}
			bearing = BearingDegrees(p.Reference, candidate)
			return distance, bearing, true
		}
	}
	return 0, 0, false
}

func clamp(v, minV, maxV float64) float64 {
	if v < minV {
		return minV
	}
	if v > maxV {
		return maxV
	}
	return v
}
