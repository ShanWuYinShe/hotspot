package mapmodule

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCanvasReturnsHeatmapForLargeViewport(t *testing.T) {
	handler := NewHandler(NewService())
	req := httptest.NewRequest(http.MethodPost, "/canvas", strings.NewReader(`{"viewport":{"bounds":{"north":10,"south":0,"east":10,"west":0}}}`))
	rr := httptest.NewRecorder()
	handler.Canvas(rr, req)

	var resp struct {
		Code int               `json:"code"`
		Data MapCanvasResponse `json:"data"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != 0 || resp.Data.Heatmap == nil || resp.Data.Hotspots != nil {
		t.Fatalf("unexpected response: %#v", resp)
	}
}

func TestCanvasReturnsHotspotsForSmallViewport(t *testing.T) {
	handler := NewHandler(NewService())
	req := httptest.NewRequest(http.MethodPost, "/canvas", strings.NewReader(`{"viewport":{"bounds":{"north":0.2,"south":0,"east":0.2,"west":0}}}`))
	rr := httptest.NewRecorder()
	handler.Canvas(rr, req)

	var resp struct {
		Code int               `json:"code"`
		Data MapCanvasResponse `json:"data"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != 0 || resp.Data.Hotspots == nil || resp.Data.Heatmap != nil {
		t.Fatalf("unexpected response: %#v", resp)
	}
}
