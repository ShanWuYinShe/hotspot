package seed

import (
	"testing"

	"hotspot/backend-go/internal/platform/geo"
)

// helpers_test.go — 单测公共辅助：行值类型断言与 spec 级公共数据。

func mustString(t *testing.T, value any) string {
	t.Helper()
	text, ok := value.(string)
	if !ok {
		t.Fatalf("值类型 = %T，期望 string", value)
	}
	return text
}

func mustInt(t *testing.T, value any) int {
	t.Helper()
	number, ok := value.(int)
	if !ok {
		t.Fatalf("值类型 = %T，期望 int", value)
	}
	return number
}

func mustFloat(t *testing.T, value any) float64 {
	t.Helper()
	number, ok := value.(float64)
	if !ok {
		t.Fatalf("值类型 = %T，期望 float64", value)
	}
	return number
}

func mustBool(t *testing.T, value any) bool {
	t.Helper()
	flag, ok := value.(bool)
	if !ok {
		t.Fatalf("值类型 = %T，期望 bool", value)
	}
	return flag
}

// lumosPoint 是 QA 主账号的位置（距离断言基准）。
var lumosPoint = geo.Point{Latitude: 34.2604, Longitude: 117.1855}
