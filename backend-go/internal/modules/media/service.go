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
	// ErrUploadQuotaExceeded 该用户未完成上传会话数/声明字节总量已达配额
	// （见 checkPresignQuota）：拒绝签发新上传 URL，客户端应完成已有上传或
	// 等待弃置会话过期（handler 映射 spec 10400 业务冲突）。
	ErrUploadQuotaExceeded = errors.New("media: pending upload quota exceeded")
	// ErrStorageQuotaExceeded 该账号「已完成媒体」总字节已达上限
	// （见 checkCompletedStorageQuota，上限 completedMediaStorageLimitBytes）：
	// complete 拒绝提交（handler 映射 spec 10400 业务冲突）。与未完成上传配额
	// 分开是因为语义与释放方式不同——完成态媒体是永久业务数据，配额只增不减，
	// 客户端无法通过「完成已有上传」释放，只能等运维调整上限。
	ErrStorageQuotaExceeded = errors.New("media: completed media storage quota exceeded")
)

type Service struct {
	appConfig appconfig.Source
	repo      Repository
	provider  Provider
	// eventOutbox 是异步后置副作用的事件生产者（bootstrap 注入通用 outbox
	// writer）：账户注销级联的媒体对象清理由请求生命周期内改为入队投递；nil
	// （单测/内存装配）或入队失败时回退同步删除，见 user_purge.go。
	eventOutbox outboxWriter
	// orphanPublishedCursor 是 published/ 前缀孤儿清理跨轮推进的 ListObjectsV2
	// continuation token：published/ 含全部完成态媒体、随业务单调增长，保留期
	// sweeper 若每轮都从首页开始在固定页数内扫描，窗口外对象永远扫不到。仅保留期
	// sweeper goroutine 读写；进程内状态，重启后从首页重新开始（见 retention.go）。
	orphanPublishedCursor string
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

// BatchLimitForFiles 返回整批的上限 = 批次内各类别上限的最大值（spec
// api_spec_media.md presign_upload：上限 = 批次内各类别上限的最大值）。此前
// handler 只取 files[0] 的类别，混合类别批次会被首个文件的类别误限（第 82 轮
// 审计 P2-1 备注）：files[0] 是小上限类别会把整批误拒，files[0] 是大上限类别
// 又可能放过超出其余类别上限的批次。
func (s *Service) BatchLimitForFiles(files []PresignFileInput) int {
	limit := 0
	for _, file := range files {
		if categoryLimit := s.BatchLimitFor(string(file.Category)); categoryLimit > limit {
			limit = categoryLimit
		}
	}
	if limit <= 0 {
		// 空批次由 handler 先行拒绝；此处兜底默认上限，调用方不应依赖 0。
		return 9
	}
	return limit
}

// CompleteBatchLimit 返回 complete_upload 的批量上限（spec api_spec_media.md
// complete_upload：upload_ids 1..N，上限与 presign_upload 一致）。upload_ids
// 不带类别，按「所有可能来源类别的上限最大值」取：presign 侧每批的上限本身
// 就是批次内各类别上限的最大值，故 complete 用全类别最大值即可覆盖任意合法
// 批次的来源；avatar 等未受 AppConfig 约束的类别固定 9，因此结果不低于 9
// （默认 9，管理员调高 moment/chat 档位时随之放大）。
func (s *Service) CompleteBatchLimit() int {
	limit := 9
	for _, category := range []MediaCategory{
		CategoryAvatar, CategoryGroupAvatar, CategoryMomentImage,
		CategoryChatImage, CategoryChatVideo, CategoryChatAudio, CategoryChatFile,
	} {
		if categoryLimit := s.BatchLimitFor(string(category)); categoryLimit > limit {
			limit = categoryLimit
		}
	}
	return limit
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
	if err := s.checkPresignQuota(ctx, principal.UserID, []PresignFileInput{req}); err != nil {
		if !errors.Is(err, ErrUploadQuotaExceeded) {
			observability.FromContext(ctx).Error("media.presign quota check failed", "error", err)
		}
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
	// 预签名配额：按用户聚合未完成会话数与声明字节总量，超限整批拒绝（与
	// spec「任一文件校验失败则整批失败、不部分预签名」的原子语义一致）。
	if err := s.checkPresignQuota(ctx, principal.UserID, files); err != nil {
		if !errors.Is(err, ErrUploadQuotaExceeded) {
			observability.FromContext(ctx).Error("media.batch_presign quota check failed", "error", err)
		}
		return PresignedUploads{}, err
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
//
// 返回值是发布对象的实际字节数（第 82 轮审计 P2-6：final_file_size 此前记录的是
// 清洗前大小，与发布对象不一致）。清洗 PUT 成功取清洗后长度；复制原对象（读取/
// 解析失败、无需清洗、清洗 PUT 失败）取 objectSize（调用方 HEAD 得到的原始长度）。
// 不采用「写后再 HEAD 发布键」：发布链路本来就知道写入了哪份字节，额外一次 S3
// HEAD 只会增加延迟与失败面（且 HEAD 失败时反而无法记录真实大小）。
func (s *Service) publishImageObject(ctx context.Context, uploadKey, publishedKey, mimeType string, objectSize int) (int, error) {
	copyOriginal := func() (int, error) {
		if err := s.provider.CopyObject(ctx, uploadKey, publishedKey); err != nil {
			return 0, err
		}
		return objectSize, nil
	}
	data, err := s.provider.GetObject(ctx, uploadKey)
	if err != nil {
		observability.FromContext(ctx).Warn("image metadata scrub: get object failed, publishing original", "error", err, "storage_key", uploadKey)
		return copyOriginal()
	}
	cleaned, changed, err := stripImageMetadata(data)
	if err != nil {
		observability.FromContext(ctx).Warn("image metadata scrub: parse failed, publishing original", "error", err, "storage_key", uploadKey)
		return copyOriginal()
	}
	if !changed {
		return copyOriginal()
	}
	if err := s.provider.PutObject(ctx, publishedKey, mimeType, cleaned); err != nil {
		observability.FromContext(ctx).Warn("image metadata scrub: put cleaned object failed, publishing original", "error", err, "storage_key", publishedKey)
		return copyOriginal()
	}
	observability.FromContext(ctx).Info("image metadata scrubbed", "upload_key", uploadKey, "published_key", publishedKey, "bytes_before", len(data), "bytes_after", len(cleaned))
	return len(cleaned), nil
}

// 预签名配额（第 81 轮审计残留：预签名 PUT 的 body 大小不受签名约束——
// 声明 1 字节仍可 PUT 到 nginx client_max_body_size 100m；未完成会话在保留期
// 内持续占用存储/带宽，连续 presign 弃置即是存储放大面）。阈值只是抑制
// 「海量 presign 弃置」的粗闸门：declared_file_size 由客户端声明、不可信，
// 真实 body 由网关 100m 上限兜底，弃置会话由保留期 GC（2 天）兜底清理。
//
// 释放路径（第 82 轮审计 P1-3）：配额此前只增不减——complete 校验失败不删行、
// 弃置会话要等 2 天保留期，正常用户弱网/取消发送 2~3 次即可把 20 个槽位占满后
// 上传功能停摆。现在有两条释放路径：
//  1. complete 的确定性失败（大小/类型/魔数/扩展名不匹配）由
//     releaseRejectedUpload 立即释放该会话（对象+行）——失败与当前对象内容绑定，
//     重试 complete 不会成功，只应让客户端重新走 presign；
//  2. presign 因配额被拒时，对该用户超过 pendingUploadCleanupGrace 仍未完成的
//     会话做一次限量 best-effort 清理后重试配额判断（cleanupStalePendingUploads），
//     不必等 2 天保留期。
//
// 单账号上界（审计报告口径，第 82 轮 P2-2/P2-3 修复后）：
//   - 未完成会话：≤ pendingUploadSessionLimit 条、声明字节 ≤ pendingUploadBytesLimit；
//     弃置会话最长占 2 天（保留期 GC），但配额拒绝时会提前清理超过宽限期的
//     弃置会话，实际「锁死上传功能」的窗口从约 2 天压到分钟级；
//   - 完成态媒体：≤ completedMediaStorageLimitBytes（2GiB，complete 阶段校验，
//     见 checkCompletedStorageQuota）。此前完成态媒体不计入任何配额，单账号可经
//     presign → PUT → complete 循环无限占用存储（complete 即重置 pending 配额，
//     第 82 轮审计 P2-3）；现在上界收敛为「账号数 × 2GiB」。仍无「完成后未被
//     业务引用」的回收策略（上传后不挂载到任何实体也计入配额）——那需要业务侧
//     引用登记，属后续工作，但与账号总配额叠加后存储上界已可预测；
//   - uploads/ 与 published/ 孤儿对象（complete 后旧上传键被重 PUT / 批量 complete
//     中途失败留下的发布副本，第 82 轮审计 P2-2 + 批次 10）：保留期 sweeper 每轮的
//     孤儿清理阶段按「对象早于保留期且 DB 无对应 storage_key 行」删除（见
//     retention.go sweepOrphanObjects），不再是永久孤儿。
const (
	// pendingUploadSessionLimit 单用户未完成上传会话数上限（正常单次业务
	// 上传 1~9 个文件且立即 complete，20 留足慢网/多业务并行余量）。
	pendingUploadSessionLimit = 20
	// pendingUploadBytesLimit 单用户未完成会话声明字节总量上限（200MB）。
	pendingUploadBytesLimit = 200 * 1024 * 1024
	// pendingUploadCleanupGrace 配额拒绝时提前清理弃置会话的宽限期：预签名
	// PUT URL 有效期即 mediaPresignTTL（15 分钟），超过后客户端无法再直传。
	// 仍存在「已直传成功但迟迟没有 complete」的会话被清理的可能，客户端需
	// 重新上传——仅在配额已耗尽时触发，用一次重传换上传功能不被锁 2 天。
	pendingUploadCleanupGrace = mediaPresignTTL
	// pendingUploadCleanupLimit 单次 presign 请求最多清理的弃置会话数：足以
	// 覆盖一整份配额（20 条），其余留给后续请求/保留期 GC。
	pendingUploadCleanupLimit = pendingUploadSessionLimit
	// pendingUploadCleanupTimeout 清理预算：清理是 best-effort 旁路，不能让
	// 配额拒绝路径拖长 presign 响应（弃置会话的对象多半从未上传，DELETE 404
	// 由 provider 视为已删，通常很快）。
	pendingUploadCleanupTimeout = 5 * time.Second
)

// completedMediaStorageLimitBytes 单账号「已完成媒体」总字节上限（2GiB）：
// 完成态媒体（头像/瞬间图/聊天附件等）是业务数据，永不回收，此前没有账号级
// 上界——complete 即重置未完成配额，攻击者可在限流允许的节奏下无限上传
// （第 82 轮审计 P2-3）。本上限把单账号完成态存储收敛到 2GiB：按契约单文件
// 上限（头像 5MB / 图片 10MB / 视频与 chat_file 100MB）折算，约合 2000 张
// 原图、200 个视频或数千张头像，足够单账号正常积累，同时把「账号数 × 上限」
// 作为存储预算的硬上界。
//
// 调整方式：直接改此常量并随版本发布——不新增迁移/AppConfig 字段。媒体存储
// 配额不在 spec 契约内（spec 只定义单文件上限），走 AppConfig 会形成契约外
// 运营配置且引入「配置误配放宽安全上界」的入口；若后续产品化需要运营可调，
// 应先扩展 spec AppLimits 再接线。
//
// 精度说明：按 complete 阶段 HEAD 到的对象大小（meta.FileSize）判定；图片
// EXIF 清洗后实际发布字节通常更小，故判定偏保守（不会低估）。检查发生在
// 发布/拷贝对象之前，被拒的 upload 不产生发布副本。检查与提交非原子：并发
// complete、同一批量 complete 的多个 upload 各自读取同一总量，可能小幅超限
// （上界约为并发数/批大小 × 单文件上限）。配额是存储滥用的粗闸门，不承担精确
// 计费语义。
const completedMediaStorageLimitBytes int64 = 2 << 30 // 2GiB

// checkPresignQuota 按用户校验「未完成上传」配额：会话数或声明字节总量
// （已有 + 本次批量）超限时返回 ErrUploadQuotaExceeded，调用方整批拒绝。
// 限额判断包含本次批量，避免「先签满再逐次追加」绕过单文件上限。
// 配额是粗粒度闸门而非精确计费：检查与 Save 非原子，并发请求可小幅超限，
// 目标是抑制单账号无限放大，不是保证严格上限。
//
// 超限时不直接拒绝：先对超宽限期的弃置会话做一次 best-effort 清理（释放路径
// 2），再重试一次配额判断，避免「历史失败批次」把用户上传功能锁到保留期。
func (s *Service) checkPresignQuota(ctx context.Context, userID string, files []PresignFileInput) error {
	usage, err := s.repo.PendingUploadUsage(ctx, userID)
	if err != nil {
		return err
	}
	batchSessions := int64(len(files))
	var batchBytes int64
	for _, file := range files {
		batchBytes += int64(file.FileSize)
	}
	if pendingUploadWithinQuota(usage, batchSessions, batchBytes) {
		return nil
	}
	s.cleanupStalePendingUploads(ctx, userID)
	usage, err = s.repo.PendingUploadUsage(ctx, userID)
	if err != nil {
		return err
	}
	if pendingUploadWithinQuota(usage, batchSessions, batchBytes) {
		return nil
	}
	observability.FromContext(ctx).Warn("media.presign quota exceeded",
		"pending_sessions", usage.Count,
		"pending_declared_bytes", usage.DeclaredBytes,
		"batch_sessions", batchSessions,
		"batch_declared_bytes", batchBytes)
	return ErrUploadQuotaExceeded
}

// pendingUploadWithinQuota 判断「已有未完成用量 + 本次批量」是否仍在配额内。
func pendingUploadWithinQuota(usage PendingUploadUsage, batchSessions, batchBytes int64) bool {
	return usage.Count+batchSessions <= pendingUploadSessionLimit &&
		usage.DeclaredBytes+batchBytes <= pendingUploadBytesLimit
}

// checkCompletedStorageQuota 校验「该账号已完成媒体总字节 + 本次文件大小」是否
// 仍在 completedMediaStorageLimitBytes 内（第 82 轮审计 P2-3：完成态媒体此前
// 无账号级上界，complete 即重置未完成配额，单账号存储可无限放大）。超限返回
// ErrStorageQuotaExceeded，由 handler 映射 spec 10400 业务冲突。
//
// 调用点在 PrepareComplete 的发布/拷贝之前：超限拒绝不会为本次 upload 产生发布
// 副本（同批已 Prepare 的 upload 的发布副本与其它 pre-commit 失败路径同语义：
// 未被任何行引用、重试会覆盖同一键）。会话行保持 pending（不释放）——上限是
// 账号级状态，释放单个会话也不会腾出空间，行留给保留期 GC 回收（对象与行）。
//
// 查询本身失败时返回原始错误（映射 10600）：不能把「无法判断用量」当成未超限
// 放行，否则存储上界在 DB 故障期间直接失效。
func (s *Service) checkCompletedStorageQuota(ctx context.Context, userID string, additionalBytes int64) error {
	used, err := s.repo.CompletedStorageBytes(ctx, userID)
	if err != nil {
		observability.FromContext(ctx).Error("media.complete storage usage query failed", "error", err, "user_id", userID)
		return err
	}
	if used+additionalBytes <= completedMediaStorageLimitBytes {
		return nil
	}
	observability.FromContext(ctx).Warn("media.complete storage quota exceeded",
		"user_id", userID,
		"used_bytes", used,
		"additional_bytes", additionalBytes,
		"limit_bytes", completedMediaStorageLimitBytes)
	return ErrStorageQuotaExceeded
}

// cleanupStalePendingUploads 清理指定用户「超过宽限期仍未完成」的会话（释放
// 路径 2）：按 created_at 升序限量取会话，先删对象、成功后再删行（与保留期
// GC 同序，避免「行删了对象还在」的永久孤儿）；对象删除失败的行保留，交给
// 下一轮清理或保留期 GC。全部失败仅告警——清理是旁路，不得影响 presign 的
// 正常错误语义。
func (s *Service) cleanupStalePendingUploads(ctx context.Context, userID string) {
	cleanupCtx, cancel := context.WithTimeout(ctx, pendingUploadCleanupTimeout)
	defer cancel()
	cutoff := time.Now().UTC().Add(-pendingUploadCleanupGrace)
	stale, err := s.repo.ListStaleUploadsForUser(cleanupCtx, userID, cutoff, pendingUploadCleanupLimit)
	if err != nil {
		observability.FromContext(ctx).Warn("media.presign quota cleanup list failed", "error", err)
		return
	}
	var released int
	for _, item := range stale {
		if err := s.provider.Delete(cleanupCtx, item.StorageKey); err != nil {
			observability.FromContext(ctx).Warn("media.presign quota cleanup object delete failed", "error", err, "upload_id", item.UploadID)
			continue
		}
		if err := s.repo.DeleteUploadRow(cleanupCtx, item.UploadID); err != nil {
			observability.FromContext(ctx).Warn("media.presign quota cleanup row delete failed", "error", err, "upload_id", item.UploadID)
			continue
		}
		released++
	}
	if len(stale) > 0 {
		observability.FromContext(ctx).Info("media.presign quota cleanup",
			"stale", len(stale), "released", released)
	}
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
		// 大小/类型/扩展名不匹配与对象内容绑定，重试不会成功：立即释放该
		// pending 会话（对象+行），把配额槽位还给用户（第 82 轮审计 P1-3）。
		if errors.Is(err, ErrUploadConflict) {
			s.releaseRejectedUpload(ctx, upload, "object metadata mismatch")
		}
		return preparedComplete{}, err
	}
	// 账号级完成态媒体配额（第 82 轮审计 P2-3）：按 HEAD 到的对象大小校验
	// 「已完成媒体总字节 + 本次文件」，超限在发布/拷贝之前拒绝——此时尚未产生
	// 发布副本，拒绝零副作用；完成态媒体因此有账号级上界（见
	// completedMediaStorageLimitBytes）。
	if err := s.checkCompletedStorageQuota(ctx, upload.UserID, int64(meta.FileSize)); err != nil {
		return preparedComplete{}, err
	}
	// 魔数嗅探（P2-1）：图片类别在 complete 阶段读取对象前 512 字节校验魔数，
	// 与声明/实际 Content-Type 比对——预签名直传的 Content-Type 由客户端控制，
	// 仅凭声明无法防止把非图片内容（如 HTML）伪装成图片（存储型 XSS/解析器混淆）。
	// 校验通过后发布到不可写发布键：图片清洗后字节直接写发布键，无需清洗/清洗
	// 链路失败时复制原对象；非图片类别用服务端 CopyObject（不把 100MB 视频读进内存）。
	// finalSize 用发布对象的实际字节数：图片清洗会删段/块使字节数变小（P2-6）。
	publishedKey := publishedStorageKeyFor(upload.UserID, upload.UploadID, upload.FileName)
	finalSize := meta.FileSize
	if isImageCategory(upload.Category) {
		if err := s.sniffImageMagic(ctx, upload.StorageKey, normalizeMimeType(meta.MimeType)); err != nil {
			// 魔数不匹配同属确定性失败：对象内容永远不可能通过 magic 校验。
			if errors.Is(err, ErrUploadConflict) {
				s.releaseRejectedUpload(ctx, upload, "image magic mismatch")
			}
			return preparedComplete{}, err
		}
		publishedSize, err := s.publishImageObject(ctx, upload.StorageKey, publishedKey, normalizeMimeType(meta.MimeType), meta.FileSize)
		if err != nil {
			observability.FromContext(ctx).Error("media.complete publish image failed", "error", err, "upload_id", uploadID, "published_key", publishedKey)
			return preparedComplete{}, err
		}
		finalSize = publishedSize
	} else {
		if err := s.provider.CopyObject(ctx, upload.StorageKey, publishedKey); err != nil {
			observability.FromContext(ctx).Error("media.complete copy object failed", "error", err, "upload_id", uploadID, "published_key", publishedKey)
			return preparedComplete{}, err
		}
	}
	return preparedComplete{
		file: UploadedMedia{
			FileID:   buildFileID(upload.UploadID, upload.UserID),
			FileSize: finalSize,
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

// releaseRejectedUpload 释放「确定性失败」的上传会话（对象+行），立即归还配额
// 槽位（第 82 轮审计 P1-3）：魔数/大小/类型/扩展名校验失败与当前对象内容绑定，
// 重试 complete 永远不会成功，客户端必须重新走 presign；若行一直留在 pending，
// 用户会在保留期（2 天）内被配额锁死上传功能。
//
// 只释放失败的那一条会话：同批其它 upload_id 可能已直传成功、客户端可单独
// 重试或由 presign 的过期清理兜底；整批删除会丢弃可用对象，且 PrepareComplete
// 已为前面通过的会话发布过副本，整批回滚反而放大孤儿面。
//
// 删除顺序与保留期 GC 一致：先删对象、成功后再删行（对象删除失败时保留行，
// 下次 complete/过期清理再释放），避免「行删了对象还在」的永久孤儿。
func (s *Service) releaseRejectedUpload(ctx context.Context, upload pendingUpload, reason string) {
	if err := s.provider.Delete(ctx, upload.StorageKey); err != nil {
		observability.FromContext(ctx).Warn("media.complete release rejected upload object failed",
			"error", err, "upload_id", upload.UploadID, "reason", reason)
		return
	}
	if err := s.repo.DeleteUploadRow(ctx, upload.UploadID); err != nil {
		observability.FromContext(ctx).Warn("media.complete release rejected upload row failed",
			"error", err, "upload_id", upload.UploadID, "reason", reason)
		return
	}
	observability.FromContext(ctx).Info("media.complete released rejected upload",
		"upload_id", upload.UploadID, "reason", reason)
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
	// 扩展名契约（spec api_spec_media.md「chat_file 安全约束」）：黑名单扩展名与
	// 「已知扩展名 × MIME 不一致」在 presign 阶段就拒绝，不签发上传 URL。
	if err := validateUploadFileContract(req.FileName, normalizeMimeType(req.MimeType)); err != nil {
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

// isActiveContentMime 报告浏览器会执行脚本的 MIME（HTML/SVG/XML/JS 系）。
func isActiveContentMime(mime string) bool {
	switch strings.ToLower(strings.TrimSpace(mime)) {
	case "text/html", "image/svg+xml", "application/xhtml+xml", "application/xml", "text/xml", "application/rss+xml",
		// JS 系：单独打开即执行（.js 扩展名已在黑名单，此处按 MIME 双判定兜底）。
		"text/javascript", "application/javascript", "application/x-javascript",
		"application/ecmascript", "text/ecmascript":
		return true
	default:
		return false
	}
}

// ── 文件扩展名契约（spec api_spec_media.md「chat_file 安全约束」）────────────
//
// spec 要求：校验基于「文件扩展名 + MIME 类型双判定」（任一命中黑名单即拒绝），
// 且 MIME 类型与扩展名不一致时拒绝。黑名单对所有上传类别生效：图片/视频/音频
// 类别虽然 MIME 白名单已挡住 HTML 等，但文件名仍可能以 .html/.exe 落库并作为
// 下载名下发，故统一在 presign 拒绝；complete 阶段再用会话行的 file_name 复核
// 一次（防历史会话或绕过 presign 的写入）。

// forbiddenUploadExtensions 是 spec 明确禁止的扩展名集合（可执行类/脚本标记类/
// 服务端脚本类，含同类常见变体；判定用最后一段扩展名，大小写不敏感）。
var forbiddenUploadExtensions = map[string]struct{}{
	// 可执行类
	".exe": {}, ".bat": {}, ".cmd": {}, ".com": {}, ".scr": {}, ".msi": {}, ".app": {},
	".sh": {}, ".bash": {}, ".zsh": {}, ".ps1": {}, ".psm1": {}, ".psd1": {},
	".vbs": {}, ".vbe": {}, ".wsf": {}, ".wsh": {}, ".hta": {}, ".cpl": {}, ".msc": {},
	".reg": {}, ".inf": {}, ".jar": {}, ".apk": {}, ".ipa": {}, ".dmg": {}, ".pkg": {},
	".deb": {}, ".rpm": {},
	// 脚本/标记类
	".html": {}, ".htm": {}, ".xhtml": {}, ".svg": {}, ".js": {}, ".mjs": {}, ".cjs": {}, ".jsx": {},
	// 服务端脚本类
	".php": {}, ".php3": {}, ".php4": {}, ".php5": {}, ".php7": {}, ".php8": {}, ".phtml": {},
	".jsp": {}, ".jspx": {}, ".jspf": {}, ".asp": {}, ".aspx": {}, ".ashx": {}, ".asmx": {},
	".cgi": {}, ".pl": {},
}

// mimeTypeByUploadExtension 是「已知扩展名 → 唯一 MIME」对照表，用于
// 「MIME 与扩展名不一致即拒绝」。不在表中的扩展名按 spec「未明确禁止的类型
// 默认允许」放行（仍须通过类别的 MIME 白名单/活跃内容黑名单）。
var mimeTypeByUploadExtension = map[string]string{
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".png":  "image/png",
	".webp": "image/webp",
	".gif":  "image/gif",
	".mp4":  "video/mp4",
	".mov":  "video/quicktime",
	".mp3":  "audio/mpeg",
	".aac":  "audio/aac",
	".m4a":  "audio/aac",
	".ogg":  "audio/ogg",
	".oga":  "audio/ogg",
	".wav":  "audio/wav",
	".pdf":  "application/pdf",
	".zip":  "application/zip",
	".txt":  "text/plain",
	".csv":  "text/csv",
	".json": "application/json",
	".md":   "text/markdown",
	".doc":  "application/msword",
	".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	".xls":  "application/vnd.ms-excel",
	".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	".ppt":  "application/vnd.ms-powerpoint",
	".pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
}

// uploadFileExtension 返回文件名的最后一段扩展名（小写，含点）。尾随的点与
// 空白先去掉，避免 "page.html." / "evil.exe " 之类的写法绕过黑名单；只取最后
// 一个扩展名——存储键（storageKeyFor）也只用最后一段，多扩展名（a.php.jpg）
// 按最终类型落库。
func uploadFileExtension(fileName string) string {
	name := strings.TrimRight(strings.TrimSpace(fileName), ". ")
	if name == "" {
		return ""
	}
	return strings.ToLower(filepath.Ext(name))
}

// validateUploadFileContract 按 spec 做「扩展名 × MIME」双判定：黑名单扩展名
// 一律拒绝；扩展名在已知对照表中但 MIME 不一致时拒绝；未知扩展名默认放行
// （由类别 MIME 白名单/活跃内容黑名单继续把关）。参数 mimeType 须已归一化。
func validateUploadFileContract(fileName, mimeType string) error {
	ext := uploadFileExtension(fileName)
	if ext == "" {
		return nil
	}
	if _, forbidden := forbiddenUploadExtensions[ext]; forbidden {
		return ErrInvalidMediaRequest
	}
	expected, known := mimeTypeByUploadExtension[ext]
	if !known {
		return nil
	}
	if normalizeMimeType(mimeType) != expected {
		return ErrInvalidMediaRequest
	}
	return nil
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
//
// 收紧语义（spec api_spec_media.md「限制优先级（契约级）」）：AppConfig 的
// 大小上限只能收紧、不能放宽，实际生效值 = min(配置值, 契约硬上限)。契约硬
// 上限即各类别的兜底常量（avatar 5MB / image 10MB / video 100MB / audio 20MB /
// chat_file 100MB）；管理员误配成更大值时必须被夹紧，否则一次配置即可突破
// spec 承诺与客诉口径（第 82 轮审计 P2-1）。
func (s *Service) maxFileSize(category MediaCategory) int {
	var upload appconfig.UploadConfig
	if s.appConfig != nil {
		upload = s.appConfig.Get(context.Background()).Upload
	}
	pick := func(configured, contractMax int) int {
		if configured <= 0 || configured > contractMax {
			return contractMax
		}
		return configured
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
	// 扩展名复核：presign 已校验声明的「扩展名 × MIME」，这里用会话行保存的
	// file_name 与对象实际的 Content-Type 再判一次，覆盖历史会话（修复前写入、
	// 未经扩展名校验）与任何绕过 presign 的会话。失败属确定性错误，调用方会
	// 释放该会话。
	if err := validateUploadFileContract(upload.FileName, completedMimeType); err != nil {
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
