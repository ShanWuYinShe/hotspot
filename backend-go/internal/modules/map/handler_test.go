package mapmodule

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type mapErrorRepo struct {
	heatmapErr  error
	hotspotsErr error
}

func (r *mapErrorRepo) Heatmap(context.Context, canvasQuery) (MapHeatCanvas, error) {
	return MapHeatCanvas{}, r.heatmapErr
}

func (r *mapErrorRepo) Hotspots(context.Context, canvasQuery) (MapHotspotsCanvas, error) {
	return MapHotspotsCanvas{}, r.hotspotsErr
}

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

func TestCanvasReturns10600OnServiceError(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&mapErrorRepo{heatmapErr: errors.New("boom"), hotspotsErr: errors.New("boom")}))

	largeReq := httptest.NewRequest(http.MethodPost, "/canvas", strings.NewReader(`{"viewport":{"bounds":{"north":10,"south":0,"east":10,"west":0}}}`))
	largeRR := httptest.NewRecorder()
	handler.Canvas(largeRR, largeReq)
	assertMapCode(t, largeRR, 10600)

	smallReq := httptest.NewRequest(http.MethodPost, "/canvas", strings.NewReader(`{"viewport":{"bounds":{"north":0.2,"south":0,"east":0.2,"west":0}}}`))
	smallRR := httptest.NewRecorder()
	handler.Canvas(smallRR, smallReq)
	assertMapCode(t, smallRR, 10600)
}

func assertMapCode(t *testing.T, rr *httptest.ResponseRecorder, expected int) {
	t.Helper()
	var resp struct {
		Code int `json:"code"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != expected {
		t.Fatalf("expected code %d, got %#v", expected, resp)
	}
}
