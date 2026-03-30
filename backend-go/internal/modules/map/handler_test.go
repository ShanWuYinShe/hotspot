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

type handlerMapRepo struct {
	heatmapResp  MapHeatCanvas
	hotspotsResp MapHotspotsCanvas
	heatmapErr   error
	hotspotsErr  error
}

func (r *handlerMapRepo) Heatmap(context.Context, canvasQuery) (MapHeatCanvas, error) {
	return r.heatmapResp, r.heatmapErr
}

func (r *handlerMapRepo) Hotspots(context.Context, canvasQuery) (MapHotspotsCanvas, error) {
	return r.hotspotsResp, r.hotspotsErr
}

func TestCanvasRejectsInvalidJSON(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&handlerMapRepo{}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/canvas", strings.NewReader(`{"viewport":`))

	handler.Canvas(rr, req)
	assertMapCode(t, rr, 10000)
}

func TestCanvasRejectsInvalidBounds(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&handlerMapRepo{}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/canvas", strings.NewReader(`{"viewport":{"bounds":{"north":0,"south":1,"east":10,"west":0}}}`))

	handler.Canvas(rr, req)
	assertMapCode(t, rr, 10000)
}

func TestCanvasRejectsInvalidDataType(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&handlerMapRepo{}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/canvas", strings.NewReader(`{"viewport":{"bounds":{"north":1,"south":0,"east":1,"west":0}},"filter":{"data_type":"weird"}}`))

	handler.Canvas(rr, req)
	assertMapCode(t, rr, 10000)
}

func TestCanvasRejectsInvalidTimeFormat(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&handlerMapRepo{}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/canvas", strings.NewReader(`{"viewport":{"bounds":{"north":1,"south":0,"east":1,"west":0}},"filter":{"time_range":{"start_time":"bad-time"}}}`))

	handler.Canvas(rr, req)
	assertMapCode(t, rr, 10000)
}

func TestCanvasRejectsReversedTimeRange(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&handlerMapRepo{}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/canvas", strings.NewReader(`{"viewport":{"bounds":{"north":1,"south":0,"east":1,"west":0}},"filter":{"time_range":{"start_time":"2026-03-02T00:00:00Z","end_time":"2026-03-01T00:00:00Z"}}}`))

	handler.Canvas(rr, req)
	assertMapCode(t, rr, 10000)
}

func TestCanvasRejectsInvalidCursor(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&handlerMapRepo{hotspotsErr: ErrInvalidCursor}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/canvas", strings.NewReader(`{"viewport":{"bounds":{"north":0.2,"south":0,"east":0.2,"west":0}},"pagination":{"cursor":"bad-cursor"}}`))

	handler.Canvas(rr, req)
	assertMapCode(t, rr, 10000)
}

func TestCanvasReturns10600OnServiceError(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&handlerMapRepo{heatmapErr: errors.New("boom"), hotspotsErr: errors.New("boom")}))

	largeReq := httptest.NewRequest(http.MethodPost, "/canvas", strings.NewReader(`{"viewport":{"bounds":{"north":10,"south":0,"east":10,"west":0}}}`))
	largeRR := httptest.NewRecorder()
	handler.Canvas(largeRR, largeReq)
	assertMapCode(t, largeRR, 10600)

	smallReq := httptest.NewRequest(http.MethodPost, "/canvas", strings.NewReader(`{"viewport":{"bounds":{"north":0.2,"south":0,"east":0.2,"west":0}}}`))
	smallRR := httptest.NewRecorder()
	handler.Canvas(smallRR, smallReq)
	assertMapCode(t, smallRR, 10600)
}

func TestCanvasReturnsHeatmapOnlyForLargeViewport(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&handlerMapRepo{heatmapResp: MapHeatCanvas{Items: []MapHeatBucket{{CellToken: "a", Level: 8, TotalCount: 1}}}}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/canvas", strings.NewReader(`{"viewport":{"bounds":{"north":10,"south":0,"east":10,"west":0}}}`))

	handler.Canvas(rr, req)
	resp := decodeMapResponse(t, rr)
	if resp.Code != 0 || resp.Data == nil || resp.Data.Heatmap == nil || resp.Data.Hotspots != nil {
		t.Fatalf("unexpected response: %#v", resp)
	}
}

func TestCanvasReturnsHotspotsOnlyForSmallViewport(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&handlerMapRepo{hotspotsResp: MapHotspotsCanvas{Items: []MapHotspotItem{{ObjectType: "user", ObjectID: "u1"}}}}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/canvas", strings.NewReader(`{"viewport":{"bounds":{"north":0.2,"south":0,"east":0.2,"west":0}}}`))

	handler.Canvas(rr, req)
	resp := decodeMapResponse(t, rr)
	if resp.Code != 0 || resp.Data == nil || resp.Data.Hotspots == nil || resp.Data.Heatmap != nil {
		t.Fatalf("unexpected response: %#v", resp)
	}
}

type mapAPIResponse struct {
	Code int                `json:"code"`
	Data *MapCanvasResponse `json:"data,omitempty"`
}

func decodeMapResponse(t *testing.T, rr *httptest.ResponseRecorder) mapAPIResponse {
	t.Helper()
	var resp mapAPIResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return resp
}

func assertMapCode(t *testing.T, rr *httptest.ResponseRecorder, expected int) {
	t.Helper()
	resp := decodeMapResponse(t, rr)
	if resp.Code != expected {
		t.Fatalf("expected code %d, got %#v", expected, resp)
	}
}
