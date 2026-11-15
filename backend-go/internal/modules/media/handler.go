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
	// 批次上限 = 批次内各类别上限的最大值（spec api_spec_media.md presign_upload：
	// moment_image → max_moment_images；chat_* → max_chat_files；取批次内最大值）。
	// 不能只看 files[0] 的类别：混合类别批次会被首个文件的类别误限（第 82 轮审计
	// P2-1 备注）。
	if limit := h.service.BatchLimitForFiles(req.Files); len(req.Files) > limit {
		httpx.WriteInvalidRequest(w, r, "PresignUpload", "批量文件数超出上限")
		return
	}
	// 直传 URL 的入口 origin 由 ClientEndpointMiddleware 全局注入 context
	// （见 client_endpoint.go），此处无需再处理。
	uploads, err := h.service.BatchPresign(r.Context(), req.Files)
	if err != nil {
		if errors.Is(err, ErrInvalidMediaRequest) {
			httpx.WriteInvalidRequest(w, r, "PresignUpload", "参数错误")
			return
		}
		if errors.Is(err, ErrUploadQuotaExceeded) {
			// 10400 业务冲突（spec api_spec_errors.md「满员/前置状态不满足」）：
			// 未完成上传已达配额，客户端提示用户完成已有上传或稍后重试；不引导
			// 立即重试（10500 的 Retry-After 语义不适用，配额不会短时自动恢复）。
			httpx.WriteBusinessError(w, r, "PresignUpload", httpx.CodeConflict, "未完成上传过多，请完成上传后重试")
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
	// 批量上限与 presign_upload 一致（spec api_spec_media.md complete_upload：
	// upload_ids 1..N，上限与 presign_upload 一致）。此前没有上限，单请求可带
	// 数万个 upload_id 放大成同量级 DB/S3 操作（第 82 轮审计 P1-2）。
	if limit := h.service.CompleteBatchLimit(); len(req.UploadIDs) > limit {
		httpx.WriteInvalidRequest(w, r, "CompleteUpload", "批量 upload_id 数超出上限")
		return
	}
	// 整批原子语义（spec api_spec_media.md）：第一阶段对全部 upload_id 执行
	// 校验（归属/对象存在性/元数据/魔数/EXIF 清洗），任一失败整批失败且未产生
	// completed 状态；第二阶段才逐个落库（仅剩 DB 写入，失败属内部错误，
	// 靠 upload 级自然幂等重试恢复）。
	uploads := make([]UploadedMedia, 0, len(req.UploadIDs))
	writeCompleteErr := func(err error) {
		switch {
		case errors.Is(err, ErrUploadNotFound):
			httpx.WriteBusinessError(w, r, "CompleteUpload", httpx.CodeNotFound, "upload_id 不存在")
		case errors.Is(err, ErrUploadNotReady), errors.Is(err, ErrUploadConflict):
			httpx.WriteBusinessError(w, r, "CompleteUpload", httpx.CodeConflict, "上传状态冲突")
		case errors.Is(err, ErrInvalidMediaRequest):
			httpx.WriteBusinessError(w, r, "CompleteUpload", httpx.CodeInvalidRequest, "参数错误")
		default:
			observability.LogServerError(r, "CompleteUpload", err)
			httpx.WriteServerError(w, r, err)
		}
	}
	prepared := make([]preparedComplete, 0, len(req.UploadIDs))
	for _, uploadID := range req.UploadIDs {
		file, err := h.service.PrepareComplete(r.Context(), uploadID)
		if err != nil {
			writeCompleteErr(err)
			return
		}
		prepared = append(prepared, file)
	}
	for i, uploadID := range req.UploadIDs {
		file, err := h.service.CommitComplete(r.Context(), uploadID, prepared[i])
		if err != nil {
			writeCompleteErr(err)
			return
		}
		uploads = append(uploads, file)
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(CompletedUploads{Uploads: uploads}))
}
