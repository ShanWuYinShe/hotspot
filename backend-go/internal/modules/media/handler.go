package media

import (
	"errors"
	"net/http"

	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/idempotency"
	"hotspot/backend-go/internal/platform/observability"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func logServerError(r *http.Request, method string, err error) {
	observability.L().Error("handler error", "method", method, "error", err, "request_id", observability.RequestIDFromContext(r.Context()))
}

func logBusinessError(r *http.Request, method string, code int, message string) {
	observability.L().Warn("handler business error", "method", method, "code", code, "message", message, "request_id", observability.RequestIDFromContext(r.Context()))
}

func (h *Handler) PresignUpload(w http.ResponseWriter, r *http.Request) {
	var req PresignUploadRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		logBusinessError(r, "PresignUpload", 10000, "参数错误")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	upload, err := h.service.Presign(r.Context(), req)
	if err != nil {
		if errors.Is(err, ErrInvalidMediaRequest) {
			logBusinessError(r, "PresignUpload", 10000, "参数错误")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
			return
		}
		logServerError(r, "PresignUpload", err)
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(upload))
}

func (h *Handler) BatchPresignUpload(w http.ResponseWriter, r *http.Request) {
	var req BatchPresignUploadRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || len(req.Files) == 0 || len(req.Files) > 9 {
		logBusinessError(r, "BatchPresignUpload", 10000, "参数错误")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	uploads, err := h.service.BatchPresign(r.Context(), req)
	if err != nil {
		if errors.Is(err, ErrInvalidMediaRequest) {
			logBusinessError(r, "BatchPresignUpload", 10000, "参数错误")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
			return
		}
		logServerError(r, "BatchPresignUpload", err)
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(uploads))
}

func (h *Handler) CompleteUpload(w http.ResponseWriter, r *http.Request) {
	if err := idempotency.RequireKey(r); err != nil {
		logBusinessError(r, "CompleteUpload", err.Code, err.Message)
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(err.Code, err.Message))
		return
	}
	var req CompleteUploadRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.UploadID == "" {
		logBusinessError(r, "CompleteUpload", 10000, "参数错误")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	idempotency.ReplayOrStore(w, r, "media/complete_upload", func() httpx.ApiResponse[UploadedMedia] {
		file, err := h.service.Complete(r.Context(), req.UploadID)
		if err != nil {
			switch {
			case errors.Is(err, ErrUploadNotFound):
				logBusinessError(r, "CompleteUpload", 10300, "upload_id 不存在")
				return httpx.ApiResponse[UploadedMedia]{Code: 10300, Message: "upload_id 不存在"}
			case errors.Is(err, ErrUploadNotReady), errors.Is(err, ErrUploadConflict):
				logBusinessError(r, "CompleteUpload", 10400, "上传状态冲突")
				return httpx.ApiResponse[UploadedMedia]{Code: 10400, Message: "上传状态冲突"}
			case errors.Is(err, ErrInvalidMediaRequest):
				logBusinessError(r, "CompleteUpload", 10000, "参数错误")
				return httpx.ApiResponse[UploadedMedia]{Code: 10000, Message: "参数错误"}
			default:
				logServerError(r, "CompleteUpload", err)
				return httpx.ApiResponse[UploadedMedia]{Code: 10600, Message: "服务暂时不可用"}
			}
		}
		return httpx.Success(file)
	})
}

func (h *Handler) CompleteUploadBatch(w http.ResponseWriter, r *http.Request) {
	var req BatchCompleteUploadRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || len(req.UploadIDs) == 0 || len(req.UploadIDs) > 9 {
		logBusinessError(r, "CompleteUploadBatch", 10000, "参数错误")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	uploads := make([]UploadedMedia, 0, len(req.UploadIDs))
	for _, uploadID := range req.UploadIDs {
		file, err := h.service.Complete(r.Context(), uploadID)
		if err != nil {
			switch {
			case errors.Is(err, ErrUploadNotFound):
				logBusinessError(r, "CompleteUploadBatch", 10300, "upload_id 不存在")
				httpx.WriteJSON(w, http.StatusOK, httpx.Error(10300, "upload_id 不存在"))
			case errors.Is(err, ErrUploadNotReady), errors.Is(err, ErrUploadConflict):
				logBusinessError(r, "CompleteUploadBatch", 10400, "上传状态冲突")
				httpx.WriteJSON(w, http.StatusOK, httpx.Error(10400, "上传状态冲突"))
			case errors.Is(err, ErrInvalidMediaRequest):
				logBusinessError(r, "CompleteUploadBatch", 10000, "参数错误")
				httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
			default:
				logServerError(r, "CompleteUploadBatch", err)
				httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
			}
			return
		}
		uploads = append(uploads, file)
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(BatchCompleteUpload{Uploads: uploads}))
}
