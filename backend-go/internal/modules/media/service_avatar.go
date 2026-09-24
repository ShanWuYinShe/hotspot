package media

import (
	"context"
	"time"

	"hotspot/backend-go/internal/platform/observability"
)

// service_avatar.go — 头像公开访问路径与签名读 URL。
//
// 背景：avatar_url 此前存裸 file_id（users/groups 直存），客户端 <img src> /
// Image() 拿到无效相对路径，全站头像实际全部坏图（首字母兜底）。修复口径：
// avatar_url 存「稳定代理路径」，GET /public/v1/media/avatars/{file_id} 匿名
// 302 到短时签名读 URL（仅放行 avatar/group_avatar 公开资料类）。相对路径
// 环境无关、可用 SQL 迁移修补存量；web 同源直用，鸿蒙经 MediaUrlUtil 拼 origin。

// avatarPublicPathPrefix 是头像对外访问的稳定代理路径前缀（挂载见 module.go
// MountPublic；spec「仅 POST RPC」的第二个登记 GET 例外，同类先例 share/og.go）。
const avatarPublicPathPrefix = "/public/v1/media/avatars/"

// avatarReadTTL 是头像 302 目标的签名有效期。代理端点每次请求实时签名，
// 短 TTL 即可覆盖一次图片加载。
const avatarReadTTL = 15 * time.Minute

// AvatarPublicPath 返回 file_id 在 avatar_url 字段中的稳定存储形态
// （相对代理路径——域名随部署环境变化，不入库）。
func (s *Service) AvatarPublicPath(fileID string) string {
	return avatarPublicPathPrefix + fileID
}

// ResolveGroupAvatarImage 校验单个 group_avatar 类 file_id：存在、归属 userID、
// 类别 group_avatar 且已完成（groups Create/Update 的头像入参校验，
// 经 groups.GroupAvatarMediaResolver 接口注入调用）。
func (s *Service) ResolveGroupAvatarImage(ctx context.Context, fileID, userID string) error {
	assets, err := s.repo.ResolveByFileIDs(ctx, []string{fileID}, userID, CategoryGroupAvatar)
	if err != nil {
		observability.L().Error("media.resolve_group_avatar failed", "error", err)
		return err
	}
	if _, ok := assets[fileID]; !ok {
		return ErrMediaNotOwned
	}
	return nil
}

// AvatarReadURL 解析头像类（avatar/group_avatar）已完成文件的短时签名读 URL，
// 供 GET /public/v1/media/avatars/{file_id} 匿名代理端点 302 使用。非头像类、
// 未完成或不存在一律 ErrMediaNotOwned，不泄露文件存在性。
func (s *Service) AvatarReadURL(ctx context.Context, fileID string) (string, error) {
	meta, ok, err := s.repo.CompletedFileMetaByID(ctx, fileID)
	if err != nil {
		observability.L().Error("media.avatar_read_url failed", "error", err)
		return "", err
	}
	if !ok || (meta.Category != CategoryAvatar && meta.Category != CategoryGroupAvatar) {
		return "", ErrMediaNotOwned
	}
	return s.provider.PresignGet(ctx, meta.StorageKey, avatarReadTTL)
}
