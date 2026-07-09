package geo

import (
	"fmt"
	"math"

	"github.com/golang/geo/s1"
	"github.com/golang/geo/s2"
)

type Bounds struct {
	North float64
	South float64
	East  float64
	West  float64
}

type Point struct {
	Latitude  float64
	Longitude float64
}

// IsValidPoint reports whether a geographic coordinate is within the valid
// WGS-84 range. Out-of-range values flow into s2.CellIDFromLatLng and produce
// invalid/zero cell IDs, silently corrupting the geo index — so callers must
// reject them before indexing.
func IsValidPoint(p Point) bool {
	return p.Latitude >= -90 && p.Latitude <= 90 && p.Longitude >= -180 && p.Longitude <= 180
}

func ValidateBounds(b Bounds) error {
	if b.North < b.South || b.East < b.West {
		return fmt.Errorf("invalid bounds")
	}
	if b.North > 90 || b.South < -90 || b.East > 180 || b.West < -180 {
		return fmt.Errorf("invalid bounds")
	}
	return nil
}

func Center(b Bounds) Point {
	return Point{Latitude: (b.North + b.South) / 2, Longitude: (b.East + b.West) / 2}
}

func ApproxViewportSpanMeters(b Bounds) float64 {
	center := Center(b)
	latMeters := math.Abs(b.North-b.South) * 111000
	lngMeters := math.Abs(b.East-b.West) * 111000 * math.Cos(center.Latitude*math.Pi/180)
	return math.Hypot(latMeters, lngMeters)
}

func SuggestedLevel(spanMeters float64) int {
	switch {
	case spanMeters > 1_000_000:
		return 8
	case spanMeters > 200_000:
		return 10
	case spanMeters > 50_000:
		return 12
	case spanMeters > 10_000:
		return 14
	case spanMeters > 2_000:
		return 16
	default:
		return 18
	}
}

func CellTokenAtLevel(p Point, level int) string {
	ll := s2.LatLngFromDegrees(p.Latitude, p.Longitude)
	cellID := s2.CellIDFromLatLng(ll).Parent(level)
	return cellID.ToToken()
}

func CoveringTokens(b Bounds, level int, maxCells int) []string {
	rect := s2.RectFromLatLng(s2.LatLngFromDegrees(b.South, b.West))
	rect = rect.AddPoint(s2.LatLngFromDegrees(b.North, b.East))
	coverer := &s2.RegionCoverer{MinLevel: level, MaxLevel: level, MaxCells: maxCells}
	ids := coverer.Covering(rect)
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, id.ToToken())
	}
	return out
}

// LeafCellID returns the level-30 (leaf) S2 cell ID for a point as int64.
// Stored in the database for efficient B-tree range scans.
func LeafCellID(p Point) int64 {
	ll := s2.LatLngFromDegrees(p.Latitude, p.Longitude)
	cellID := s2.CellIDFromLatLng(ll)
	if !cellID.IsValid() {
		return 0
	}
	return int64(cellID)
}

// CoveringArrays returns the S2 covering as two separate slices
// suitable for passing to PostgreSQL's unnest($1::bigint[], $2::bigint[]).
func CoveringArrays(bounds Bounds) (mins []int64, maxs []int64) {
	ranges := SearchCovering(bounds)
	mins = make([]int64, len(ranges))
	maxs = make([]int64, len(ranges))
	for i, r := range ranges {
		mins[i], maxs[i] = r[0], r[1]
	}
	return
}

// SearchCovering returns [RangeMin, RangeMax] pairs of leaf cell IDs
// for a mixed-level S2 covering of the given bounds.
// Each pair maps to a BETWEEN clause on the leaf cell ID column,
// turning a 2D spatial search into efficient 1D range scans.
// The covering is intentionally coarse (pre-filter) — distance filter handles precision.
func SearchCovering(bounds Bounds) [][2]int64 {
	span := ApproxViewportSpanMeters(bounds)
	level := SuggestedLevel(span)
	maxLevel := level
	if maxLevel > 16 {
		maxLevel = 16
	}
	minLevel := maxLevel - 4
	if minLevel < 4 {
		minLevel = 4
	}
	coverer := &s2.RegionCoverer{MinLevel: minLevel, MaxLevel: maxLevel, MaxCells: 64}
	var covering []s2.CellID
	if bounds.East > 180 {
		// Crossing from positive side: split [West, 180] and [-180, East-360]
		rect1 := s2.RectFromLatLng(s2.LatLngFromDegrees(bounds.South, bounds.West))
		rect1 = rect1.AddPoint(s2.LatLngFromDegrees(bounds.North, 180))
		rect2 := s2.RectFromLatLng(s2.LatLngFromDegrees(bounds.South, -180))
		rect2 = rect2.AddPoint(s2.LatLngFromDegrees(bounds.North, bounds.East-360))
		covering = append(covering, coverer.Covering(rect1)...)
		covering = append(covering, coverer.Covering(rect2)...)
	} else if bounds.West < -180 {
		// Crossing from negative side: split [-180, East] and [West+360, 180]
		rect1 := s2.RectFromLatLng(s2.LatLngFromDegrees(bounds.South, -180))
		rect1 = rect1.AddPoint(s2.LatLngFromDegrees(bounds.North, bounds.East))
		rect2 := s2.RectFromLatLng(s2.LatLngFromDegrees(bounds.South, bounds.West+360))
		rect2 = rect2.AddPoint(s2.LatLngFromDegrees(bounds.North, 180))
		covering = append(covering, coverer.Covering(rect1)...)
		covering = append(covering, coverer.Covering(rect2)...)
	} else {
		rect := s2.RectFromLatLng(s2.LatLngFromDegrees(bounds.South, bounds.West))
		rect = rect.AddPoint(s2.LatLngFromDegrees(bounds.North, bounds.East))
		covering = coverer.Covering(rect)
	}
	ranges := make([][2]int64, 0, len(covering))
	for _, id := range covering {
		if !id.IsValid() {
			continue
		}
		ranges = append(ranges, [2]int64{int64(id.RangeMin()), int64(id.RangeMax())})
	}
	return ranges
}

// ParentCellToken maps a leaf cell ID to its ancestor's token at the given level.
func ParentCellToken(leafCellID int64, level int) string {
	id := s2.CellID(leafCellID)
	if !id.IsValid() {
		return ""
	}
	return id.Parent(level).ToToken()
}

// CellCenter returns the center point of an S2 cell identified by its token.
func CellCenter(token string) Point {
	id := s2.CellIDFromToken(token)
	if !id.IsValid() {
		return Point{}
	}
	ll := id.LatLng()
	return Point{Latitude: ll.Lat.Degrees(), Longitude: ll.Lng.Degrees()}
}

func DistanceMeters(a, b Point) int {
	p1 := s2.LatLngFromDegrees(a.Latitude, a.Longitude)
	p2 := s2.LatLngFromDegrees(b.Latitude, b.Longitude)
	return int(s1.Angle(p1.Distance(p2)).Radians() * 6371000)
}

func BearingDegrees(a, b Point) int {
	lat1 := a.Latitude * math.Pi / 180
	lat2 := b.Latitude * math.Pi / 180
	dLng := (b.Longitude - a.Longitude) * math.Pi / 180
	y := math.Sin(dLng) * math.Cos(lat2)
	x := math.Cos(lat1)*math.Sin(lat2) - math.Sin(lat1)*math.Cos(lat2)*math.Cos(dLng)
	bearing := math.Atan2(y, x) * 180 / math.Pi
	if bearing < 0 {
		bearing += 360
	}
	return int(math.Round(bearing))
}

// HaversineDistanceSQL returns a SQL expression fragment computing the Haversine
// distance in meters between the row's (latitude, longitude) and reference params.
// refLatParam/refLngParam are $N placeholders for the reference coordinates.
func HaversineDistanceSQL(refLatParam, refLngParam string) string {
	return fmt.Sprintf(
		`CAST(6371000 * 2 * ASIN(SQRT(POWER(SIN(RADIANS(latitude - %s) / 2), 2) + COS(RADIANS(%s)) * COS(RADIANS(latitude)) * POWER(SIN(RADIANS(longitude - %s) / 2), 2))) AS INTEGER)`,
		refLatParam, refLatParam, refLngParam,
	)
}
