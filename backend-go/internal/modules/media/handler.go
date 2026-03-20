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
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.FileName == "" || req.FileSize <= 0 || req.MimeType == "" || req.Category == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(h.service.Presign(r.Context(), req)))
}

func (h *Handler) BatchPresignUpload(w http.ResponseWriter, r *http.Request) {
	var req BatchPresignUploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.Files) == 0 || len(req.Files) > 9 {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(h.service.BatchPresign(r.Context(), req)))
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
		file, ok, _ := h.service.Complete(r.Context(), req.UploadID)
		if !ok {
			return httpx.ApiResponse[UploadedMedia]{Code: 10300, Message: "upload_id 不存在"}
		}
		return httpx.Success(file)
	})
}
