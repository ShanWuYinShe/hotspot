package geo

import (
	"fmt"
	"math"
	"sync"

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

// DefaultFallbackLatitude / DefaultFallbackLongitude 是 appConfig 缺失时的
// reference_point 回退锚点(回退链第 3 级 default_location 的服务端兜底,
// 见 api_spec_types.md「reference_point 解析规则」)。与
// ClientConfig.defaults.default_location 的默认值一致。
const (
	DefaultFallbackLatitude  = 35.154922
	DefaultFallbackLongitude = 112.682524
)

// IsValidPoint reports whether a geographic coordinate is within the valid
// WGS-84 range. Out-of-range values flow into s2.CellIDFromLatLng and produce
// invalid/zero cell IDs, silently corrupting the geo index — so callers must
// reject them before indexing.
func IsValidPoint(p Point) bool {
	return p.Latitude >= -90 && p.Latitude <= 90 && p.Longitude >= -180 && p.Longitude <= 180
}

// ValidateBounds 校验视口边界。East < West 一律拒绝：spec api_spec_public.md
// map/canvas 明确「暂不支持跨国际日期变更线视口（需由客户端拆分请求）」，
// 因此下方 computeSearchCovering 的 antimeridian 拆分分支当前从 HTTP 入口
// 不可达——它们是 spec 放开该限制时的预留实现，勿删。
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

// ParentShift 返回把 leaf cell ID 归并到 level 层祖先的右移位数 s：
// (cell >> s) << s 即该 cell 在 level 层的祖先 cell ID。S2 CellID 布局为
// 3 位 face + 60 位位置 + 末位 1 标记，level 层 cell 的标记位恰在第 s 位，
// 移位掩码后自然保留；负值（face 6/7 溢出符号位）在 Postgres 算术右移下
// 同样成立（符号扩展位会被左移清回，仅低 s 位被清零）。
// 供 SQL GROUP BY 直聚合计数使用（避免把全量候选行拉到 Go 再聚合）。
func ParentShift(level int) int {
	return 60 - 2*level
}

// CellIDCenter 返回原始 cell ID（数值形态）所在格心的经纬度。
func CellIDCenter(id uint64) Point {
	ll := s2.CellID(id).LatLng()
	return Point{Latitude: ll.Lat.Degrees(), Longitude: ll.Lng.Degrees()}
}

// CellIDToken 把原始 cell ID 转为 token 串（与 CellTokenAtLevel 同一编码）。
func CellIDToken(id uint64) string {
	return s2.CellID(id).ToToken()
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
//
// Results are memoised by quantized bounds (rounded to 5 decimal places, ~1m)
// because identical client viewports repeatedly produce identical coverings.
// The s2.RegionCoverer construction itself is the expensive part (~μs of
// allocation per call), and map/feed endpoints call this multiple times per
// request, so caching yields a high hit rate at effectively zero cost. The
// cache is bounded via simple LRU eviction to avoid unbounded growth when
// viewports drift.
func SearchCovering(bounds Bounds) [][2]int64 {
	key := quantizeBounds(bounds)
	coverCacheMu.RLock()
	if cached, ok := coverCache[key]; ok {
		coverCacheMu.RUnlock()
		// 返回副本：调用方原地修改会污染缓存（coverings 是范围对切片）。
		out := make([][2]int64, len(cached))
		copy(out, cached)
		return out
	}
	coverCacheMu.RUnlock()

	ranges := computeSearchCovering(bounds)

	coverCacheMu.Lock()
	if coverCache == nil {
		coverCache = make(map[coverCacheKey][][2]int64, coverCacheMaxEntries)
	}
	// Bounded eviction: drop a random-ish entry once we hit the cap. Using
	// the map iteration order is fine here because correctness does not
	// depend on which entry is dropped — only the cache hit rate.
	if len(coverCache) >= coverCacheMaxEntries {
		for k := range coverCache {
			delete(coverCache, k)
			break
		}
	}
	coverCache[key] = ranges
	coverCacheMu.Unlock()
	out := make([][2]int64, len(ranges))
	copy(out, ranges)
	return out
}

// coverCacheMaxEntries bounds the memoisation table. Each entry is a few
// hundred bytes; 4096 entries caps memory at single-digit MB.
const coverCacheMaxEntries = 4096

var (
	coverCacheMu sync.RWMutex
	coverCache   = make(map[coverCacheKey][][2]int64, coverCacheMaxEntries)
)

// coverCacheKey 是缓存的精确键（4 个量化分量直接作结构体键）：
// 此前把 ~100 bit 输入折叠成单个 uint64 非加密哈希当精确键用，碰撞时
// SearchCovering 会返回完全不同视口的覆盖区间（地理查询静默查错区域），
// 且 South/North 负浮点转 uint32 是未定义行为（南半球视口键不可预测）。
type coverCacheKey struct {
	South uint32
	North uint32
	West  uint32
	East  uint32
}

// quantizeBounds rounds bounds to ~5 decimal places (~1m at the equator)
// and packs them into the cache key. Identical client viewports
// (which the map endpoint sees repeatedly while panning) collapse to the
// same key, giving a high cache hit rate.
func quantizeBounds(b Bounds) coverCacheKey {
	// Multiply by 1e5 and round to integer; 5 decimals ≈ 1.1m at equator.
	// 全部偏移到非负区间再转 uint32（Go 规范中负浮点转无符号是
	// implementation-dependent）。
	south := uint32(int64(math.Round(b.South*1e5)) + 90*1e5)
	north := uint32(int64(math.Round(b.North*1e5)) + 90*1e5)
	west := uint32(int64(math.Round(b.West*1e5)) + 180*1e5)
	east := uint32(int64(math.Round(b.East*1e5)) + 180*1e5)
	return coverCacheKey{South: south, North: north, West: west, East: east}
}

func computeSearchCovering(bounds Bounds) [][2]int64 {
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

// SphericalCentroid 返回一组地理坐标的球面质心。
//
// 与简单的经纬度算术平均不同，它先把坐标换算成三维单位向量再平均，
// 因此可以正确处理跨 ±180° 经线的点集。返回 false 的情况只有输入为空、
// 全部非法，或者向量完全抵消（例如精确对跖点对）。
func SphericalCentroid(points []Point) (Point, bool) {
	var x, y, z float64
	valid := 0
	for _, p := range points {
		if !IsValidPoint(p) {
			continue
		}
		lat := p.Latitude * math.Pi / 180
		lng := p.Longitude * math.Pi / 180
		cosLat := math.Cos(lat)
		x += cosLat * math.Cos(lng)
		y += cosLat * math.Sin(lng)
		z += math.Sin(lat)
		valid++
	}
	if valid == 0 {
		return Point{}, false
	}
	norm := math.Sqrt(x*x + y*y + z*z)
	if norm < 1e-12 {
		return Point{}, false
	}
	return Point{
		Latitude:  math.Asin(math.Max(-1, math.Min(1, z/norm))) * 180 / math.Pi,
		Longitude: math.Atan2(y, x) * 180 / math.Pi,
	}, true
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

// CellLevelShift returns the right-shift amount that maps a leaf S2 cell ID to
// its level-L ancestor bucket for SQL GROUP BY aggregation:
//
//	bucket = s2_cell_id >> CellLevelShift(level)
//
// groups every leaf sharing the same level-L parent. Shifts use arithmetic
// right shift on both Go and PostgreSQL signed bigint, so leaf IDs on faces
// 4-5 (negative as int64) still group by their true high bits.
func CellLevelShift(level int) int {
	return 2*(maxS2Level-level) + 1
}

// maxS2Level is the deepest S2 cell level (leaf). Kept local to avoid
// importing s2 constants into call sites.
const maxS2Level = 30

// CellTokenLevel returns the S2 level of a cell token, or -1 when the token is
// invalid. Used to validate client-provided cell tokens at the API boundary.
func CellTokenLevel(token string) int {
	id := s2.CellIDFromToken(token)
	if !id.IsValid() {
		return -1
	}
	return id.Level()
}

// CellTokenLeafID returns the leaf cell ID (level 30) a token resolves to,
// or 0 for an invalid token. Drilldown queries derive their bucket from it.
func CellTokenLeafID(token string) int64 {
	id := s2.CellIDFromToken(token)
	if !id.IsValid() {
		return 0
	}
	return int64(id)
}

// BucketCellToken reconstructs the S2 cell token of a level-L bucket produced
// by grouping leaf IDs with `s2_cell_id >> CellLevelShift(level)`. The bucket
// is the shifted integer; the dropped low bits are irrelevant, so the
// canonical ancestor (lsb set) is rebuilt before converting to a token. The
// int64→uint64 round-trip preserves the bit pattern for negative buckets.
// Returns "" for out-of-range levels or an invalid reconstructed cell.
func BucketCellToken(bucket int64, level int) string {
	if level < 0 || level > maxS2Level {
		return ""
	}
	shift := uint(CellLevelShift(level))
	id := s2.CellID(uint64(bucket)<<shift | uint64(1)<<(shift-1))
	if !id.IsValid() {
		return ""
	}
	return id.ToToken()
}
