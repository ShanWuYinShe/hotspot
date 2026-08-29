package media

import (
	"errors"
	"net/http"

	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/observability"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) PresignUpload(w http.ResponseWriter, r *http.Request) {
	var req PresignUploadRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || len(req.Files) == 0 {
		httpx.WriteInvalidRequest(w, r, "PresignUpload", "参数错误")
		return
	}
	// 批次上限 = 批次内各类别上限的最大值（spec api_spec_media.md：moment_image → max_moment_images；chat_* → max_chat_files）
	if limit := h.service.BatchLimitFor(string(req.Files[0].Category)); len(req.Files) > limit {
		httpx.WriteInvalidRequest(w, r, "PresignUpload", "批量文件数超出上限")
		return
	}
	uploads, err := h.service.BatchPresign(r.Context(), req.Files)
	if err != nil {
		if errors.Is(err, ErrInvalidMediaRequest) {
			httpx.WriteInvalidRequest(w, r, "PresignUpload", "参数错误")
			return
		}
		httpx.WriteAPIError(w, r, "PresignUpload", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(uploads))
}

func (h *Handler) CompleteUpload(w http.ResponseWriter, r *http.Request) {
	var req CompleteUploadRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || len(req.UploadIDs) == 0 {
		httpx.WriteInvalidRequest(w, r, "CompleteUpload", "参数错误")
		return
	}
	// 原子语义：任一 upload_id 校验失败则整批失败，不部分完成。
	uploads := make([]UploadedMedia, 0, len(req.UploadIDs))
	for _, uploadID := range req.UploadIDs {
		file, err := h.service.Complete(r.Context(), uploadID)
		if err != nil {
			switch {
			case errors.Is(err, ErrUploadNotFound):
				httpx.WriteBusinessError(w, r, "CompleteUpload", httpx.CodeNotFound, "upload_id 不存在")
			case errors.Is(err, ErrUploadNotReady), errors.Is(err, ErrUploadConflict):
				httpx.WriteBusinessError(w, r, "CompleteUpload", httpx.CodeConflict, "上传状态冲突")
			case errors.Is(err, ErrInvalidMediaRequest):
				httpx.WriteBusinessError(w, r, "CompleteUpload", httpx.CodeInvalidRequest, "参数错误")
			default:
				observability.LogServerError(r, "CompleteUpload", err)
				httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInternalUnavailable, "服务暂时不可用"))
			}
			return
		}
		uploads = append(uploads, file)
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(CompletedUploads{Uploads: uploads}))
}
