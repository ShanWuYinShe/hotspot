package media

import (
	"context"
	"hotspot/backend-go/internal/platform/appconfig"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
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
	repo     Repository
	provider Provider
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
	if err := validatePresignUploadRequest(req); err != nil {
		return PresignedUpload{}, err
	}
	upload, err := s.provider.Presign(ctx, req, principal.UserID)
	if err != nil {
		observability.L().Error("media.presign provider failed", "error", err)
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
		observability.L().Error("media.presign save failed", "error", err, "upload_id", upload.UploadID)
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
		if err := validatePresignUploadRequest(file); err != nil {
			return PresignedUploads{}, err
		}
	}
	uploads := make([]PresignedUpload, 0, len(files))
	for _, file := range files {
		upload, err := s.provider.Presign(ctx, file, principal.UserID)
		if err != nil {
			observability.L().Error("media.batch_presign provider failed", "error", err)
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
			observability.L().Error("media.batch_presign save failed", "error", err, "upload_id", upload.UploadID)
			return PresignedUploads{}, err
		}
		upload.StorageKey = ""
		uploads = append(uploads, upload)
	}
	return PresignedUploads{Uploads: uploads}, nil
}

func (s *Service) Complete(ctx context.Context, uploadID string) (UploadedMedia, error) {
	principal, ok := platformauth.PrincipalFromContext(ctx)
	if !ok || principal.UserID == "" {
		return UploadedMedia{}, ErrInvalidMediaRequest
	}
	upload, ok, err := s.repo.Get(ctx, uploadID)
	if err != nil {
		observability.L().Error("media.complete get failed", "error", err, "upload_id", uploadID)
		return UploadedMedia{}, err
	}
	if !ok || upload.UserID != principal.UserID {
		return UploadedMedia{}, ErrUploadNotFound
	}
	if upload.Completed {
		// 自然幂等（spec api_spec_media.md）：upload_id → file_id 映射在首次 complete 成功后
		// 持久化，同一 upload_id 重复调用直接返回相同 file_id（等价成功），不依赖
		// Idempotency-Key——覆盖"complete 成功但响应丢失后重试"场景。
		return upload.File, nil
	}
	key := idempotency.KeyFromContext(ctx) // 仅记录用途（repo 存储），不参与幂等判定
	meta, err := s.provider.Head(ctx, upload.StorageKey)
	if err != nil {
		if errors.Is(err, ErrUploadNotReady) {
			return UploadedMedia{}, ErrUploadNotReady
		}
		observability.L().Error("media.complete head failed", "error", err, "upload_id", uploadID, "storage_key", upload.StorageKey)
		return UploadedMedia{}, err
	}
	if err := validateCompletedObjectMetadata(upload, meta); err != nil {
		return UploadedMedia{}, err
	}
	file := UploadedMedia{
		FileID:   buildFileID(upload.UploadID, upload.UserID),
		FileSize: meta.FileSize,
		MimeType: meta.MimeType,
	}
	completed, ok, err := s.repo.Complete(ctx, uploadID, file, key)
	if err != nil {
		observability.L().Error("media.complete save failed", "error", err, "upload_id", uploadID)
		return UploadedMedia{}, err
	}
	if !ok {
		return UploadedMedia{}, ErrUploadNotFound
	}
	return completed.File, nil
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
		observability.L().Error("media.resolve_moment_images failed", "error", err)
		return nil, err
	}
	for _, id := range fileIDs {
		if _, ok := assets[id]; !ok {
			return nil, ErrMediaNotOwned
		}
	}
	return assets, nil
}

// PresignGet signs a short-term client-facing read URL for the given storage key.
func (s *Service) PresignGet(ctx context.Context, storageKey string, ttl time.Duration) (string, error) {
	return s.provider.PresignGet(ctx, storageKey, ttl)
}

func validatePresignUploadRequest(req PresignFileInput) error {
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
	if !isMimeAllowed(req.Category, req.MimeType) {
		return ErrInvalidMediaRequest
	}
	if req.FileSize > maxFileSize(req.Category) {
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
		return true
	default:
		return false
	}
}

func maxFileSize(category MediaCategory) int {
	switch category {
	case CategoryAvatar, CategoryGroupAvatar:
		return 5 * 1024 * 1024
	case CategoryMomentImage, CategoryChatImage:
		return 10 * 1024 * 1024
	case CategoryChatVideo:
		return 100 * 1024 * 1024
	case CategoryChatAudio:
		return 20 * 1024 * 1024
	case CategoryChatFile:
		return 100 * 1024 * 1024
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
