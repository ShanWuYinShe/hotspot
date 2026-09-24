package media

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/observability"
)

// handler_avatar.go — GET /public/v1/media/avatars/{file_id}（匿名，302 到
// 签名读 URL）。挂载见 module.go MountPublic；spec「仅 POST RPC」的登记
// GET 例外（同类先例 share/og.go）：它服务 <img src>/Image() 直接消费的
// avatar_url 字段，非 RPC 动作面。
//
// 失败一律 404 统一 envelope（不存在/非头像类/未完成不区分——不泄露
// 文件存在性），客户端 img onerror 走首字母兜底。
func (h *Handler) ServeAvatar(w http.ResponseWriter, r *http.Request) {
	fileID := chi.URLParam(r, "file_id")
	readURL, err := h.service.AvatarReadURL(r.Context(), fileID)
	if err != nil || readURL == "" {
		if err != nil && err != ErrMediaNotOwned {
			observability.L().Warn("media.avatar resolve failed", "method", "ServeAvatar")
		}
		httpx.NotFoundHandler(w, r)
		return
	}
	http.Redirect(w, r, readURL, http.StatusFound)
}
