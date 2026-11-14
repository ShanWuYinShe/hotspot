package media

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"hotspot/backend-go/internal/platform/appconfig"
	"mime"
	"path/filepath"
	"strings"
	"time"

	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/idempotency"
	"hotspot/backend-go/internal/platform/observability"
)

var (
	ErrInvalidMediaRequest = errors.New("media: invalid request")
	ErrUploadNotFound      = errors.New("media: upload not found")
	ErrUploadNotReady      = errors.New("media: upload not ready")
	ErrUploadConflict      = errors.New("media: upload conflict")
	ErrMediaNotOwned       = errors.New("media: file not owned by user or not ready")
)

type Service struct {
	appConfig appconfig.Source
	repo      Repository
	provider  Provider
}

func NewService() *Service {
	return NewServiceWithRepoAndProvider(NewPostgresRepository(nil), NewObjectStorageProvider(config.Load()))
}
func NewServiceWithRepo(repo Repository) *Service {
	return NewServiceWithRepoAndProvider(repo, NewObjectStorageProvider(config.Load()))
}

// SetAppConfig wires the dynamic app config source (batch limits, spec types.md AppLimits).
func (s *Service) SetAppConfig(src appconfig.Source) { s.appConfig = src }

// BatchLimitFor returns the per-category batch size limit (spec media.md：
// moment_image → max_moment_images；chat_* → max_chat_files；默认 9)。
func (s *Service) BatchLimitFor(category string) int {
	if s.appConfig == nil {
		return 9
	}
	limits := s.appConfig.Get(context.Background()).Limits
	switch category {
	case "moment_image":
		if limits.MaxMomentImages > 0 {
			return limits.MaxMomentImages
		}
	case "chat_image", "chat_video", "chat_audio", "chat_file":
		if limits.MaxChatFiles > 0 {
			return limits.MaxChatFiles
		}
	}
	return 9
}

func NewServiceWithRepoAndProvider(repo Repository, provider Provider) *Service {
	return &Service{repo: repo, provider: provider}
}

func (s *Service) Presign(ctx context.Context, req PresignFileInput) (PresignedUpload, error) {
	principal, ok := platformauth.PrincipalFromContext(ctx)
	if !ok || principal.UserID == "" {
		return PresignedUpload{}, ErrInvalidMediaRequest
	}
	if err := validatePresignUploadRequest(req, s.maxFileSize(req.Category)); err != nil {
		return PresignedUpload{}, err
	}
	upload, err := s.provider.Presign(ctx, req, principal.UserID)
	if err != nil {
		observability.FromContext(ctx).Error("media.presign provider failed", "error", err)
		return PresignedUpload{}, err
	}
	if err := s.repo.Save(ctx, pendingUpload{
		UploadID:         upload.UploadID,
		UserID:           principal.UserID,
		StorageKey:       upload.StorageKey,
		FileName:         req.FileName,
		DeclaredFileSize: req.FileSize,
		DeclaredMimeType: req.MimeType,
		Category:         req.Category,
		MD5:              req.MD5,
		CreatedAt:        time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		observability.FromContext(ctx).Error("media.presign save failed", "error", err, "upload_id", upload.UploadID)
		return PresignedUpload{}, err
	}
	upload.StorageKey = ""
	return upload, nil
}

func (s *Service) BatchPresign(ctx context.Context, files []PresignFileInput) (PresignedUploads, error) {
	principal, ok := platformauth.PrincipalFromContext(ctx)
	if !ok || principal.UserID == "" {
		return PresignedUploads{}, ErrInvalidMediaRequest
	}
	for _, file := range files {
		if err := validatePresignUploadRequest(file, s.maxFileSize(file.Category)); err != nil {
			return PresignedUploads{}, err
		}
	}
	uploads := make([]PresignedUpload, 0, len(files))
	for _, file := range files {
		upload, err := s.provider.Presign(ctx, file, principal.UserID)
		if err != nil {
			observability.FromContext(ctx).Error("media.batch_presign provider failed", "error", err)
			return PresignedUploads{}, err
		}
		if err := s.repo.Save(ctx, pendingUpload{
			UploadID:         upload.UploadID,
			UserID:           principal.UserID,
			StorageKey:       upload.StorageKey,
			FileName:         file.FileName,
			DeclaredFileSize: file.FileSize,
			DeclaredMimeType: file.MimeType,
			Category:         file.Category,
			MD5:              file.MD5,
			CreatedAt:        time.Now().UTC().Format(time.RFC3339),
		}); err != nil {
			observability.FromContext(ctx).Error("media.batch_presign save failed", "error", err, "upload_id", upload.UploadID)
			return PresignedUploads{}, err
		}
		upload.StorageKey = ""
		uploads = append(uploads, upload)
	}
	return PresignedUploads{Uploads: uploads}, nil
}

// publishImageObject 把图片从可写上传键发布到不可写发布键，同时清洗图片元数据
// （EXIF/XMP/IPTC，含 GPS）：直传对象原样入库会把发布坐标之外的住址级位置暴露
// 给一切可见者。需要清洗时把清洗后字节直接 PUT 到发布键——不再回写上传键，
// 上传键仍可被原预签名 PUT 覆盖，而发布键从不签发给客户端，因此发布后内容不可变
// （第 81 轮审计：原实现发布后仍可被原 URL 重 PUT 替换）。
// fail-open：读取/解析失败保留原文件并告警（复制原对象），清洗不可用不能阻塞
// 合法上传；PUT 清洗版失败时同样退回复制原对象并告警。返回错误表示发布键未就绪，
// complete 不得提交。
func (s *Service) publishImageObject(ctx context.Context, uploadKey, publishedKey, mimeType string) error {
	data, err := s.provider.GetObject(ctx, uploadKey)
	if err != nil {
		observability.FromContext(ctx).Warn("image metadata scrub: get object failed, publishing original", "error", err, "storage_key", uploadKey)
		return s.provider.CopyObject(ctx, uploadKey, publishedKey)
	}
	cleaned, changed, err := stripImageMetadata(data)
	if err != nil {
		observability.FromContext(ctx).Warn("image metadata scrub: parse failed, publishing original", "error", err, "storage_key", uploadKey)
		return s.provider.CopyObject(ctx, uploadKey, publishedKey)
	}
	if !changed {
		return s.provider.CopyObject(ctx, uploadKey, publishedKey)
	}
	if err := s.provider.PutObject(ctx, publishedKey, mimeType, cleaned); err != nil {
		observability.FromContext(ctx).Warn("image metadata scrub: put cleaned object failed, publishing original", "error", err, "storage_key", publishedKey)
		return s.provider.CopyObject(ctx, uploadKey, publishedKey)
	}
	observability.FromContext(ctx).Info("image metadata scrubbed", "upload_key", uploadKey, "published_key", publishedKey, "bytes_before", len(data), "bytes_after", len(cleaned))
	return nil
}

func (s *Service) Complete(ctx context.Context, uploadID string) (UploadedMedia, error) {
	prepared, err := s.PrepareComplete(ctx, uploadID)
	if err != nil {
		return UploadedMedia{}, err
	}
	return s.CommitComplete(ctx, uploadID, prepared)
}

// preparedComplete 是 PrepareComplete 的产物：待落库的 file 元数据 + copy-on-complete
// 的键信息。published=false 表示 Prepare 阶段即发现已完成（自然幂等），Commit 无副作用。
type preparedComplete struct {
	file         UploadedMedia
	uploadKey    string // 可写上传键，提交成功后尽力删除
	publishedKey string // 不可写发布键，提交时写入 media_upload_sessions.storage_key
	published    bool   // 本次是否已把对象复制/落盘到发布键
}

// PrepareComplete 执行 complete 的全部校验与副作用前置（归属、对象存在性、
// 元数据、图片魔数与 EXIF 清洗），并把对象发布到不可写发布键（copy-on-complete）：
// 上传键仍可被客户端凭原预签名 PUT 覆盖，发布键从不签发给客户端，因此发布后
// 内容不可变。批量 complete 以「先整批校验、再整批提交」两阶段满足 spec 的整批
// 原子语义（api_spec_media.md：任一 upload_id 校验失败则整批失败，不部分完成）；
// 提交前失败留下的发布副本未被任何行引用，重试会覆盖同一键（幂等），不产生可见
// 影响。已完成 upload 直接返回既有 file（自然幂等，不做复制/删除等副作用）。
func (s *Service) PrepareComplete(ctx context.Context, uploadID string) (preparedComplete, error) {
	principal, ok := platformauth.PrincipalFromContext(ctx)
	if !ok || principal.UserID == "" {
		return preparedComplete{}, ErrInvalidMediaRequest
	}
	upload, ok, err := s.repo.Get(ctx, uploadID)
	if err != nil {
		observability.FromContext(ctx).Error("media.complete get failed", "error", err, "upload_id", uploadID)
		return preparedComplete{}, err
	}
	if !ok || upload.UserID != principal.UserID {
		return preparedComplete{}, ErrUploadNotFound
	}
	if upload.Completed {
		// 自然幂等（spec api_spec_media.md）：upload_id → file_id 映射在首次 complete 成功后
		// 持久化，同一 upload_id 重复调用直接返回相同 file_id（等价成功），不依赖
		// Idempotency-Key——覆盖"complete 成功但响应丢失后重试"场景。
		return preparedComplete{file: upload.File}, nil
	}
	meta, err := s.provider.Head(ctx, upload.StorageKey)
	if err != nil {
		if errors.Is(err, ErrUploadNotReady) {
			return preparedComplete{}, ErrUploadNotReady
		}
		observability.FromContext(ctx).Error("media.complete head failed", "error", err, "upload_id", uploadID, "storage_key", upload.StorageKey)
		return preparedComplete{}, err
	}
	if err := validateCompletedObjectMetadata(upload, meta); err != nil {
		return preparedComplete{}, err
	}
	// 魔数嗅探（P2-1）：图片类别在 complete 阶段读取对象前 512 字节校验魔数，
	// 与声明/实际 Content-Type 比对——预签名直传的 Content-Type 由客户端控制，
	// 仅凭声明无法防止把非图片内容（如 HTML）伪装成图片（存储型 XSS/解析器混淆）。
	// 校验通过后发布到不可写发布键：图片清洗后字节直接写发布键，无需清洗/清洗
	// 链路失败时复制原对象；非图片类别用服务端 CopyObject（不把 100MB 视频读进内存）。
	publishedKey := publishedStorageKeyFor(upload.UserID, upload.UploadID, upload.FileName)
	if isImageCategory(upload.Category) {
		if err := s.sniffImageMagic(ctx, upload.StorageKey, normalizeMimeType(meta.MimeType)); err != nil {
			return preparedComplete{}, err
		}
		if err := s.publishImageObject(ctx, upload.StorageKey, publishedKey, normalizeMimeType(meta.MimeType)); err != nil {
			observability.FromContext(ctx).Error("media.complete publish image failed", "error", err, "upload_id", uploadID, "published_key", publishedKey)
			return preparedComplete{}, err
		}
	} else {
		if err := s.provider.CopyObject(ctx, upload.StorageKey, publishedKey); err != nil {
			observability.FromContext(ctx).Error("media.complete copy object failed", "error", err, "upload_id", uploadID, "published_key", publishedKey)
			return preparedComplete{}, err
		}
	}
	return preparedComplete{
		file: UploadedMedia{
			FileID:   buildFileID(upload.UploadID, upload.UserID),
			FileSize: meta.FileSize,
			MimeType: meta.MimeType,
		},
		uploadKey:    upload.StorageKey,
		publishedKey: publishedKey,
		published:    true,
	}, nil
}

// CommitComplete 将 PrepareComplete 通过的 upload 落为 completed：条件 UPDATE
// 同时写入发布键；成功后尽力删除原上传键（失败仅告警——发布键已是唯一被引用的
// 对象，旧键残留只是垃圾）。条件 UPDATE 未命中（并发/重复完成）时回滚本次复制
// 出的发布键（若已完成行引用同一键则保留，避免删掉并发方刚提交的对象），并回读
// 既有 file 保持自然幂等。仅剩 DB 写入 + 清理；此时失败属内部错误，靠 upload 级
// 幂等重试恢复。
func (s *Service) CommitComplete(ctx context.Context, uploadID string, prepared preparedComplete) (UploadedMedia, error) {
	principal, ok := platformauth.PrincipalFromContext(ctx)
	if !ok || principal.UserID == "" {
		return UploadedMedia{}, ErrInvalidMediaRequest
	}
	if !prepared.published {
		// Prepare 阶段即已完成：自然幂等返回既有 file，不做复制/删除等副作用。
		return prepared.file, nil
	}
	// 单次往返：条件 UPDATE（status <> 'completed'）成功即返回——file 元数据
	// 由 PrepareComplete 校验后传入，正是本次写入库的值（审查 P2-19）。
	updated, err := s.repo.Complete(ctx, uploadID, prepared.file, prepared.publishedKey, idempotency.KeyFromContext(ctx))
	if err != nil {
		observability.FromContext(ctx).Error("media.complete save failed", "error", err, "upload_id", uploadID)
		return UploadedMedia{}, err
	}
	if updated {
		// 发布键已随 UPDATE 入库，上传键不再被引用：尽力删除（失败仅告警）。
		s.discardObject(ctx, prepared.uploadKey)
		return prepared.file, nil
	}
	// 未更新：会话不存在，或已被并发/重复请求完成——回滚本次复制出的发布键并
	// 回读一次区分，保持幂等。
	upload, found, err := s.repo.Get(ctx, uploadID)
	if err != nil {
		observability.FromContext(ctx).Error("media.complete get failed", "error", err, "upload_id", uploadID)
		return UploadedMedia{}, err
	}
	if !found {
		s.discardObject(ctx, prepared.publishedKey)
		return UploadedMedia{}, ErrUploadNotFound
	}
	if upload.Completed {
		if upload.StorageKey != prepared.publishedKey {
			// 并发方用别的发布键完成了该 upload：本次复制出的对象无人引用，删除。
			s.discardObject(ctx, prepared.publishedKey)
		}
		return upload.File, nil
	}
	s.discardObject(ctx, prepared.publishedKey)
	return UploadedMedia{}, ErrUploadNotFound
}

// discardObject 尽力删除对象（complete 清理路径）：失败仅告警，不影响已被条件
// UPDATE 保护的完成态；残留对象由保留期/人工清理兜底。
func (s *Service) discardObject(ctx context.Context, storageKey string) {
	if storageKey == "" {
		return
	}
	if err := s.provider.Delete(ctx, storageKey); err != nil {
		observability.FromContext(ctx).Warn("media.complete object cleanup failed", "error", err, "storage_key", storageKey)
	}
}

// ResolveMomentImages resolves the given file_ids to their stored media assets,
// enforcing that every id belongs to userID, has category 'moment_image' and is
// completed. The returned map is keyed by file_id. If any requested file_id is
// missing (not owned / wrong category / not completed) ErrMediaNotOwned is
// returned — callers must treat attachment as all-or-nothing.
func (s *Service) ResolveMomentImages(ctx context.Context, fileIDs []string, userID string) (map[string]MediaAsset, error) {
	if len(fileIDs) == 0 {
		return map[string]MediaAsset{}, nil
	}
	assets, err := s.repo.ResolveByFileIDs(ctx, fileIDs, userID, CategoryMomentImage)
	if err != nil {
		observability.FromContext(ctx).Error("media.resolve_moment_images failed", "error", err)
		return nil, err
	}
	for _, id := range fileIDs {
		if _, ok := assets[id]; !ok {
			return nil, ErrMediaNotOwned
		}
	}
	return assets, nil
}

// ResolveAvatarImage 校验单个 avatar_file_id：存在、归属 userID、category 为
// avatar 且上传已完成（spec api_spec_user.md update_account 错误码 10300：
// avatar_file_id 对应的媒体资源不存在或不属于当前用户）。不满足时返回
// ErrMediaNotOwned；users 模块经 AccountAvatarMediaResolver 接口注入调用。
func (s *Service) ResolveAvatarImage(ctx context.Context, fileID, userID string) error {
	assets, err := s.repo.ResolveByFileIDs(ctx, []string{fileID}, userID, CategoryAvatar)
	if err != nil {
		observability.FromContext(ctx).Error("media.resolve_avatar_image failed", "error", err)
		return err
	}
	if _, ok := assets[fileID]; !ok {
		return ErrMediaNotOwned
	}
	return nil
}

// PresignGet signs a short-term client-facing read URL for the given storage key.
func (s *Service) PresignGet(ctx context.Context, storageKey string, ttl time.Duration) (string, error) {
	return s.provider.PresignGet(ctx, storageKey, ttl)
}

// validatePresignUploadRequest 校验直传预签名请求。maxSize 由调用方按
// Service.maxFileSize(category) 传入——AppConfig 注入在 Service 上，自由函数
// 无法直接读取。
func validatePresignUploadRequest(req PresignFileInput, maxSize int) error {
	if strings.TrimSpace(req.FileName) == "" || len(req.FileName) > 255 {
		return ErrInvalidMediaRequest
	}
	if req.FileSize <= 0 {
		return ErrInvalidMediaRequest
	}
	if !isAllowedMediaCategory(req.Category) {
		return ErrInvalidMediaRequest
	}
	if strings.TrimSpace(req.MimeType) == "" {
		return ErrInvalidMediaRequest
	}
	// 必须先归一化再判定：带参数的 MIME（如 "text/html; charset=utf-8"）会绕过
	// active-content 精确匹配黑名单，而该原始串会被直接写进预签名 PUT 的
	// Content-Type，最终在存储域名上以 HTML 执行（存储型 XSS）。
	if !isMimeAllowed(req.Category, normalizeMimeType(req.MimeType)) {
		return ErrInvalidMediaRequest
	}
	if req.FileSize > maxSize {
		return ErrInvalidMediaRequest
	}
	if req.MD5 != "" {
		decoded, err := hex.DecodeString(req.MD5)
		if err != nil || len(decoded) != md5.Size {
			return ErrInvalidMediaRequest
		}
	}
	return nil
}

func isAllowedMediaCategory(category MediaCategory) bool {
	switch category {
	case CategoryAvatar, CategoryGroupAvatar, CategoryMomentImage, CategoryChatImage, CategoryChatVideo, CategoryChatAudio, CategoryChatFile:
		return true
	default:
		return false
	}
}

func isMimeAllowed(category MediaCategory, mime string) bool {
	switch category {
	case CategoryAvatar, CategoryGroupAvatar:
		return mime == "image/jpeg" || mime == "image/png" || mime == "image/webp"
	case CategoryMomentImage, CategoryChatImage:
		return mime == "image/jpeg" || mime == "image/png" || mime == "image/webp" || mime == "image/gif"
	case CategoryChatVideo:
		return mime == "video/mp4" || mime == "video/quicktime"
	case CategoryChatAudio:
		return mime == "audio/mpeg" || mime == "audio/aac" || mime == "audio/ogg"
	case CategoryChatFile:
		// chat_file 是开放类别，但必须黑名单 active content：预签名 GET 按对象
		// Content-Type 原样回给浏览器，text/html、SVG 等在存储域名上可直接
		// 执行，构成存储型 XSS。
		return !isActiveContentMime(mime)
	default:
		return false
	}
}

// isActiveContentMime 报告浏览器会执行脚本的 MIME（HTML/SVG/XML 系）。
func isActiveContentMime(mime string) bool {
	switch strings.ToLower(strings.TrimSpace(mime)) {
	case "text/html", "image/svg+xml", "application/xhtml+xml", "application/xml", "text/xml", "application/rss+xml":
		return true
	default:
		return false
	}
}

// 各类别的兜底上限（AppConfig 未接线或字段为 0 时使用）；数值与既有
// 硬编码常量完全一致，保证存量配置（无 avatar/chat_file 新字段）行为不变。
const (
	defaultAvatarMaxSize   = 5 * 1024 * 1024
	defaultImageMaxSize    = 10 * 1024 * 1024
	defaultVideoMaxSize    = 100 * 1024 * 1024
	defaultAudioMaxSize    = 20 * 1024 * 1024
	defaultChatFileMaxSize = 100 * 1024 * 1024
)

// maxFileSize 返回类别的上传大小上限。上限统一从 AppConfig.Upload 读取
// （AGENTS「配置」节：应用限制禁止硬编码），字段为 0/未接线时回退同值常量。
// 审查 P2-23：此前六类全部硬编码，与同文件 BatchLimitFor 走 AppConfig
// 形成「同类参数两种来源」，调整上限需要发版。
func (s *Service) maxFileSize(category MediaCategory) int {
	var upload appconfig.UploadConfig
	if s.appConfig != nil {
		upload = s.appConfig.Get(context.Background()).Upload
	}
	pick := func(configured, fallback int) int {
		if configured > 0 {
			return configured
		}
		return fallback
	}
	switch category {
	case CategoryAvatar, CategoryGroupAvatar:
		return pick(upload.AvatarMaxSize, defaultAvatarMaxSize)
	case CategoryMomentImage, CategoryChatImage:
		return pick(upload.ImageMaxSize, defaultImageMaxSize)
	case CategoryChatVideo:
		return pick(upload.VideoMaxSize, defaultVideoMaxSize)
	case CategoryChatAudio:
		return pick(upload.AudioMaxSize, defaultAudioMaxSize)
	case CategoryChatFile:
		return pick(upload.ChatFileMaxSize, defaultChatFileMaxSize)
	default:
		return 0
	}
}

func validateCompletedObjectMetadata(upload pendingUpload, meta ObjectMetadata) error {
	if meta.FileSize <= 0 {
		return ErrUploadConflict
	}
	if upload.DeclaredFileSize > 0 && meta.FileSize != upload.DeclaredFileSize {
		return ErrUploadConflict
	}
	completedMimeType := normalizeMimeType(meta.MimeType)
	if completedMimeType == "" {
		return ErrUploadConflict
	}
	if upload.DeclaredMimeType != "" && completedMimeType != normalizeMimeType(upload.DeclaredMimeType) {
		return ErrUploadConflict
	}
	if !isMimeAllowed(upload.Category, completedMimeType) {
		return ErrUploadConflict
	}
	return nil
}

func normalizeMimeType(value string) string {
	mediaType, _, err := mime.ParseMediaType(strings.TrimSpace(value))
	if err != nil {
		return strings.TrimSpace(value)
	}
	return mediaType
}

// magicSniffLen 魔数嗅探读取长度（覆盖 PNG/WebP 文件头 + RIFF 头部）。
const magicSniffLen = 512

// sniffImageMagic 读取对象前 magicSniffLen 字节，校验魔数与完成的 MIME 一致；
// 读取失败按错误返回（fail-closed），不匹配返回 ErrUploadConflict。
func (s *Service) sniffImageMagic(ctx context.Context, storageKey, mimeType string) error {
	prefix, err := s.provider.ReadPrefix(ctx, storageKey, magicSniffLen)
	if err != nil {
		observability.FromContext(ctx).Error("media.complete read prefix failed", "error", err, "storage_key", storageKey)
		return err
	}
	if !magicMatchesImageType(prefix, mimeType) {
		observability.FromContext(ctx).Warn("media.complete magic mismatch", "mime_type", mimeType, "storage_key", storageKey)
		return ErrUploadConflict
	}
	return nil
}

// isImageCategory 报告该类别是否为图片白名单类别（仅图片启用魔数校验）。
func isImageCategory(category MediaCategory) bool {
	switch category {
	case CategoryAvatar, CategoryGroupAvatar, CategoryMomentImage, CategoryChatImage:
		return true
	default:
		return false
	}
}

// magicMatchesImageType 手写魔数表：判断前缀字节是否与图片 MIME 一致。
// 非图片 MIME（当前不会传入）一律放行。
func magicMatchesImageType(prefix []byte, mimeType string) bool {
	switch mimeType {
	case "image/jpeg":
		return len(prefix) >= 3 && prefix[0] == 0xFF && prefix[1] == 0xD8 && prefix[2] == 0xFF
	case "image/png":
		return bytes.HasPrefix(prefix, []byte("\x89PNG\r\n\x1a\n"))
	case "image/gif":
		return bytes.HasPrefix(prefix, []byte("GIF87a")) || bytes.HasPrefix(prefix, []byte("GIF89a"))
	case "image/webp":
		return len(prefix) >= 12 && string(prefix[0:4]) == "RIFF" && string(prefix[8:12]) == "WEBP"
	default:
		return true
	}
}

func buildFileID(uploadID, userID string) string {
	sum := md5.Sum([]byte(userID + ":" + uploadID))
	return fmt.Sprintf("file_%x", sum[:])
}

func storageKeyFor(userID, uploadID, fileName string) string {
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(fileName), "."))
	if ext == "" {
		ext = "bin"
	}
	return fmt.Sprintf("uploads/%s/%s.%s", userID, uploadID, ext)
}

// publishedStorageKeyFor 返回 copy-on-complete 的发布键：仅前缀由 uploads/ 换成
// published/，用户/上传 ID/扩展名与上传键同构（扩展名口径复用 storageKeyFor，
// 避免两处规则漂移）。发布键从不由预签名 PUT 下发给客户端，因此发布后对象不可
// 被原上传 URL 覆盖；同一 upload 重复 complete 也幂等落到同一键。
func publishedStorageKeyFor(userID, uploadID, fileName string) string {
	return "published/" + strings.TrimPrefix(storageKeyFor(userID, uploadID, fileName), "uploads/")
}

// CompletedFileMetaByID 按 file_id 读取完成态文件的展示元数据（跨模块只读
// 场景：举报快照冻结媒体真值；不限定 owner/category，见 Repository 同名方法）。
func (s *Service) CompletedFileMetaByID(ctx context.Context, fileID string) (FileMeta, bool, error) {
	return s.repo.CompletedFileMetaByID(ctx, fileID)
}
