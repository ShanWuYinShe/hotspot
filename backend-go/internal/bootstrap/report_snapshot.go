package bootstrap

import (
	"context"
	"errors"

	"hotspot/backend-go/internal/modules/groupmessages"
	"hotspot/backend-go/internal/modules/privatemessages"
	"hotspot/backend-go/internal/modules/reports"
	"hotspot/backend-go/internal/platform/persistence"

	"github.com/jackc/pgx/v5"
)

// reportMessageSnapshotSource 把私聊/群聊两个消息源适配为
// reports.MessageSnapshotSource（spec api_spec_types.md ReportSnapshot：
// 服务端在举报受理时基于消息真值生成快照，客户端不提交）。两个消息模块
// 各暴露一个按 message_id 读消息的适配函数，reports 只依赖最小接口，
// 不直接 import 聊天模块。mediaFiles 按 file_id 补全媒体元数据，使
// snapshot_media 为完整 MessageMedia（spec api_spec_types.md ~MessageMedia）。
type reportMessageSnapshotSource struct {
	groupMessages   *groupmessages.Service
	privateMessages privatemessages.ReadService
	mediaFiles      reportMediaFileSource
}

var _ reports.MessageSnapshotSource = reportMessageSnapshotSource{}

// newReportMessageSnapshotSource 组装举报快照源；factory 为 nil 或 DB 不可用时
// mediaFiles 降级为「查不到元数据」，snapshot_media 回退为仅 url 可知的形态。
func newReportMessageSnapshotSource(factory *persistence.Factory, groupMessages *groupmessages.Service, privateMessages privatemessages.ReadService) reportMessageSnapshotSource {
	var queryable persistence.QueryableDB
	if factory != nil {
		queryable, _ = factory.DB.(persistence.QueryableDB)
	}
	return reportMessageSnapshotSource{
		groupMessages:   groupMessages,
		privateMessages: privateMessages,
		mediaFiles:      reportMediaFileSource{queryable: queryable},
	}
}

// GetMessageSnapshot 先探群聊、再探私聊，命中即生成快照。
// snapshot_content 仅 text 类型非 null；snapshot_media 仅媒体消息非 null
// （完整 MessageMedia）；target_chat_id：group = group_id，private = 相对发送者的对方 user_id。
func (s reportMessageSnapshotSource) GetMessageSnapshot(ctx context.Context, messageID string) (*reports.ReportSnapshot, bool, error) {
	if s.groupMessages != nil {
		snap, found, err := s.groupMessages.GetMessageReportSnapshot(ctx, messageID)
		if err != nil {
			return nil, false, err
		}
		if found {
			return s.buildReportSnapshot(ctx, "group", snap.GroupID, snap.Type, snap.Content, snap.FileID, snap.SentAt)
		}
	}
	if s.privateMessages != nil {
		snap, found, err := s.privateMessages.GetMessageReportSnapshot(ctx, messageID)
		if err != nil {
			return nil, false, err
		}
		if found {
			return s.buildReportSnapshot(ctx, "private", snap.TargetUserID, snap.Type, snap.Content, snap.FileID, snap.SentAt)
		}
	}
	return nil, false, nil
}

// buildReportSnapshot 组装冻结快照：content 仅文本消息保留；媒体消息基于
// file_id 解析存储元数据生成完整 MessageMedia（spec api_spec_types.md：
// 管理员审核基于快照，不回查消息库——故元数据在受理时冻结进快照）。
func (s reportMessageSnapshotSource) buildReportSnapshot(ctx context.Context, chatType, targetChatID, messageType, content, fileID, sentAt string) (*reports.ReportSnapshot, bool, error) {
	snapshot := &reports.ReportSnapshot{
		ChatType:     chatType,
		TargetChatID: targetChatID,
		MessageType:  messageType,
		SentAt:       sentAt,
	}
	if messageType == "text" {
		snapshotContent := content
		snapshot.SnapshotContent = &snapshotContent
	}
	if fileID != "" {
		meta, found, err := s.mediaFiles.get(ctx, fileID)
		if err != nil {
			return nil, false, err
		}
		if !found {
			// 消息引用的文件已不存在：按目标不存在处理（快照无法成立）。
			return nil, false, nil
		}
		snapshot.SnapshotMedia = meta.toMessageMedia()
	}
	return snapshot, true, nil
}

// reportMediaMeta 是 snapshot_media 冻结所需的媒体文件真值（来自
// media_upload_sessions 完成态记录）。
type reportMediaMeta struct {
	URL          string
	ThumbnailURL string
	Width        int
	Height       int
	MimeType     string
	FileName     string
	FileSize     int
	Duration     int
}

// toMessageMedia 映射为 spec MessageMedia：未知/不适用字段按契约置 null。
// url 使用 storage key，与聊天 WS media 负载（GetMediaForMessage）口径一致；
// duration 仅音视频非 null（0 = 不适用），width/height 仅图片/视频非 null。
func (m reportMediaMeta) toMessageMedia() *reports.MessageMedia {
	media := &reports.MessageMedia{URL: m.URL}
	media.ThumbnailURL = strPtrOrNull(m.ThumbnailURL)
	media.MimeType = strPtrOrNull(m.MimeType)
	media.FileName = strPtrOrNull(m.FileName)
	if m.FileSize > 0 {
		fileSize := m.FileSize
		media.FileSize = &fileSize
	}
	if m.Width > 0 {
		width := m.Width
		media.Width = &width
	}
	if m.Height > 0 {
		height := m.Height
		media.Height = &height
	}
	if m.Duration > 0 {
		duration := m.Duration
		media.Duration = &duration
	}
	return media
}

func strPtrOrNull(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

// reportMediaFileSource 按 file_id 读取完成态上传文件的展示元数据。它是
// bootstrap 装配层的最小适配器（与 image_resolver_adapter.go 同一模式）：
// reports 包不得直接依赖 media/chat 模块，而 media Service 现有查询均按
// 「owner+category」过滤（ResolveMomentImages），不适用举报快照按 file_id
// 的跨类别读取，故此处以参数化 SQL 直查 media_upload_sessions。
type reportMediaFileSource struct {
	queryable persistence.QueryableDB
}

// queryReportMediaFileByFileID 参数化查询：file_id 唯一定位，仅取 completed
// 文件；final_* 为 complete 校验后的真值，空缺时回退声明值。
const queryReportMediaFileByFileID = `
SELECT storage_key,
       COALESCE(thumbnail_url, ''),
       COALESCE(width, 0),
       COALESCE(height, 0),
       COALESCE(final_mime_type, declared_mime_type, ''),
       COALESCE(file_name, ''),
       COALESCE(final_file_size, declared_file_size, 0),
       COALESCE(duration, 0)
FROM media_upload_sessions
WHERE file_id = $1 AND status = 'completed'`

func (s reportMediaFileSource) get(ctx context.Context, fileID string) (reportMediaMeta, bool, error) {
	if s.queryable == nil {
		return reportMediaMeta{}, false, nil
	}
	var meta reportMediaMeta
	if err := s.queryable.QueryRow(ctx, queryReportMediaFileByFileID, fileID).Scan(
		&meta.URL,
		&meta.ThumbnailURL,
		&meta.Width,
		&meta.Height,
		&meta.MimeType,
		&meta.FileName,
		&meta.FileSize,
		&meta.Duration,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return reportMediaMeta{}, false, nil
		}
		return reportMediaMeta{}, false, err
	}
	return meta, true, nil
}
