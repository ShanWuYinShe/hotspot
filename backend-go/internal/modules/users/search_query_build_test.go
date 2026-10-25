package users

import (
	"strings"
	"testing"

	"hotspot/backend-go/internal/platform/geo"
)

// TestBuildSearchUsersQueryVariants 覆盖半径/手机号/标签三类可选谓词的 SQL 组装分支。
func TestBuildSearchUsersQueryVariants(t *testing.T) {
	bounds := geo.BoundsAround(geo.Point{Latitude: 39.9, Longitude: 116.4}, 5000)
	sql, args := buildSearchUsersQuery(bounds, "13800138000", "美食", true)
	if sql == "" {
		t.Fatal("radius search SQL is empty")
	}
	if !strings.Contains(sql, "unnest(") || !strings.Contains(sql, "user_profile_tags") || !strings.Contains(sql, "login_id") {
		t.Fatalf("radius/phone/tag predicates missing: %s", sql)
	}
	if len(args) < 6 {
		t.Fatalf("args = %d, want >=6 (mins,maxs,keyword,phone,tag,limit)", len(args))
	}
	plain, plainArgs := buildSearchUsersQuery(geo.Bounds{}, "", "", false)
	if strings.Contains(plain, "unnest(") || strings.Contains(plain, "EXISTS (SELECT 1 FROM user_profile_tags") {
		t.Fatalf("plain search should not include radius/tag filter predicates: %s", plain)
	}
	if len(plainArgs) != 1 {
		t.Fatalf("plain args = %v, want only limit", plainArgs)
	}
}

// TestContainsFold 覆盖搜索 Go 层 tag 双保险的大小写归一比较。
func TestContainsFold(t *testing.T) {
	if !containsFold([]string{"Food", "美食"}, "food") {
		t.Fatal("case-insensitive match failed")
	}
	if !containsFold([]string{"美食"}, "美食") {
		t.Fatal("exact match failed")
	}
	if containsFold([]string{"美食"}, "travel") {
		t.Fatal("unexpected match")
	}
	if containsFold(nil, "") {
		t.Fatal("empty tags must not match")
	}
}
