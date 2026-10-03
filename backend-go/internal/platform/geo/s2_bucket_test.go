package geo

import (
	"math"
	"testing"
)

// BucketCellToken 必须与 CellTokenAtLevel（内存聚合路径）对同一批点产出一致
// 的祖先 token——SQL GROUP BY（bucket = leaf >> shift）与 Go 内存聚合是同一
// 语义的两条实现，出现分叉会让两条路径的统计口径漂移。
func TestBucketCellTokenMatchesCellTokenAtLevel(t *testing.T) {
	// 覆盖 face 0-5：含负 int64 的 leaf ID（face 4/5）与赤道/高纬点。
	points := []Point{
		{Latitude: 39.9042, Longitude: 116.4074}, // 北京（face 4 附近）
		{Latitude: 31.2304, Longitude: 121.4737}, // 上海
		{Latitude: -33.87, Longitude: 151.21},    // 悉尼（南半球）
		{Latitude: 40.7128, Longitude: -74.006},  // 纽约（西经）
		{Latitude: -1.29, Longitude: 36.82},      // 内罗毕（赤道附近）
		{Latitude: 64.14, Longitude: -21.94},     // 雷克雅未克（高纬）
		{Latitude: -38.44, Longitude: -70.12},    // 南美（face 3/2 边界带）
		{Latitude: 51.5, Longitude: -0.12},       // 伦敦（本初子午线附近）
	}
	for _, level := range []int{4, 6, 8, 10, 12} {
		for _, p := range points {
			leaf := LeafCellID(p)
			want := CellTokenAtLevel(p, level)
			got := BucketCellToken(leaf>>CellLevelShift(level), level)
			if got != want {
				t.Errorf("level=%d lat=%f lng=%f: BucketCellToken=%q want %q", level, p.Latitude, p.Longitude, got, want)
				continue
			}
			// 同一桶内的点必须解析回同一祖先中心（SQL 聚合结果的代表坐标）。
			center := CellCenter(got)
			if center.Latitude == 0 && center.Longitude == 0 {
				t.Errorf("level=%d token=%q: CellCenter returned zero point", level, got)
			}
		}
	}
}

func TestSphericalCentroidUsesSamePoint(t *testing.T) {
	point := Point{Latitude: 35.154922, Longitude: 112.682524}
	got, ok := SphericalCentroid([]Point{point, point})
	if !ok {
		t.Fatal("expected centroid for identical points")
	}
	if math.Abs(got.Latitude-point.Latitude) > 1e-12 || math.Abs(got.Longitude-point.Longitude) > 1e-12 {
		t.Fatalf("expected centroid %v, got %v", point, got)
	}
}

func TestSphericalCentroidAcrossAntimeridian(t *testing.T) {
	got, ok := SphericalCentroid([]Point{{Latitude: 0, Longitude: 179}, {Latitude: 0, Longitude: -179}})
	if !ok {
		t.Fatal("expected centroid across antimeridian")
	}
	if math.Abs(got.Latitude) > 1e-12 || math.Abs(math.Abs(got.Longitude)-180) > 1e-9 {
		t.Fatalf("expected centroid near (0, ±180), got %v", got)
	}
}

func TestSphericalCentroidRejectsEmptyAndDegenerateInput(t *testing.T) {
	if _, ok := SphericalCentroid(nil); ok {
		t.Fatal("expected no centroid for empty input")
	}
	if _, ok := SphericalCentroid([]Point{{Latitude: 0, Longitude: 0}, {Latitude: 0, Longitude: 180}}); ok {
		t.Fatal("expected no centroid for exactly antipodal input")
	}
}

func TestBucketCellTokenInvalidInput(t *testing.T) {
	if got := BucketCellToken(123, -1); got != "" {
		t.Errorf("level=-1 should return empty, got %q", got)
	}
	if got := BucketCellToken(123, 31); got != "" {
		t.Errorf("level=31 should return empty, got %q", got)
	}
	// face 位为 6/7（int64 负数桶、全 1 高位）不是合法 S2 cell，
	// 必须拒绝而不是产出伪 token。
	if got := BucketCellToken(-1, 12); got != "" {
		t.Errorf("face-7 bucket should return empty, got %q", got)
	}
	if got := BucketCellToken(-1, 0); got != "" {
		t.Errorf("face-7 bucket at level 0 should return empty, got %q", got)
	}
}
