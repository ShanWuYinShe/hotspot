package media

import (
	"encoding/json"
	"net/http"

	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/idempotency"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) PresignUpload(w http.ResponseWriter, r *http.Request) {
	var req PresignUploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || !isValidPresignUploadRequest(req) {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	upload, err := h.service.Presign(r.Context(), req)
	if err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(upload))
}

func (h *Handler) BatchPresignUpload(w http.ResponseWriter, r *http.Request) {
	var req BatchPresignUploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.Files) == 0 || len(req.Files) > 9 {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	for _, file := range req.Files {
		if !isValidPresignUploadRequest(file) {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
			return
		}
	}
	uploads, err := h.service.BatchPresign(r.Context(), req)
	if err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(uploads))
}

func isValidPresignUploadRequest(req PresignUploadRequest) bool {
	return req.FileName != "" && req.FileSize > 0 && req.MimeType != "" && req.Category != ""
}

func (h *Handler) CompleteUpload(w http.ResponseWriter, r *http.Request) {
	if err := idempotency.RequireKey(r); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(err.Code, err.Message))
		return
	}
	var req CompleteUploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.UploadID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	idempotency.ReplayOrStore(w, r, "media/complete_upload", func() httpx.ApiResponse[UploadedMedia] {
		file, ok, err := h.service.Complete(r.Context(), req.UploadID)
		if err != nil {
			return httpx.ApiResponse[UploadedMedia]{Code: 10600, Message: "服务暂时不可用"}
		}
		if !ok {
			return httpx.ApiResponse[UploadedMedia]{Code: 10300, Message: "upload_id 不存在"}
		}
		return httpx.Success(file)
	})
}
