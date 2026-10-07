package users

import (
	"testing"

	momenttypes "hotspot/backend-go/internal/modules/moments"
	"hotspot/backend-go/internal/platform/types"
)

// normalizePagination / normalizeMomentPagination 是「我的瞬间」「收藏动态」
// 等用户侧列表的 limit/cursor 归一化入口。spec api_spec_base.md：
// limit default=20、max=100。此前零覆盖，钳制逻辑被删时不会被发现。
func TestNormalizePagination(t *testing.T) {
	tests := []struct {
		name       string
		limit      int
		wantLimit  int
		wantCursor string
	}{
		{"nil pagination → default", 0, 20, ""},
		{"limit <= 0 → default", -1, 20, ""},
		{"normal limit passes through", 5, 5, "cur_1"},
		{"limit at max boundary", 100, 100, ""},
		{"limit above max → clamped", 100000, 100, ""},
	}
	_ = types.DefaultPageSize
	_ = types.MaxPageSize

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var p *PaginationParams
			if tt.limit != 0 {
				p = &PaginationParams{Limit: tt.limit, Cursor: tt.wantCursor}
			}
			gotLimit, gotCursor := normalizePagination(p)
			if gotLimit != tt.wantLimit {
				t.Fatalf("limit = %d, want %d", gotLimit, tt.wantLimit)
			}
			if gotCursor != tt.wantCursor {
				t.Fatalf("cursor = %q, want %q", gotCursor, tt.wantCursor)
			}
		})
	}
}

func TestNormalizeMomentPagination(t *testing.T) {
	tests := []struct {
		name      string
		limit     int
		wantLimit int
	}{
		{"nil → default 20", 0, 20},
		{"negative → default 20", -5, 20},
		{"normal → passthrough", 10, 10},
		{"at max → passthrough", 100, 100},
		{"above max → clamped to 100", 50000, 100},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var p *momenttypes.PaginationParams
			if tt.limit != 0 {
				p = &momenttypes.PaginationParams{Limit: tt.limit}
			}
			gotLimit, _ := normalizeMomentPagination(p)
			if gotLimit != tt.wantLimit {
				t.Fatalf("limit = %d, want %d", gotLimit, tt.wantLimit)
			}
		})
	}
}
