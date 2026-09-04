package reports

import (
	"context"

	"hotspot/backend-go/internal/modules/groupmessages"
	"hotspot/backend-go/internal/modules/privatemessages"
)

// snapshot_source.go —— MessageSnapshotSource 的默认实现（spec api_spec_types.md
// ReportSnapshot：服务端在举报受理时基于消息真值生成快照，客户端不提交）。
//
// 该适配器属于 reports 模块（消费方），而非 bootstrap 装配层：参与者校验
// （群聊=在群成员；私聊=消息作者或会话对方）与快照组装规则是举报域的业务规则。
// media 元数据经 mediaFileMetaSource 窄接口读取，reports 不 import media/chat
// 的具体类型以外的依赖；聊天模块依赖方向 reports → groupmessages/privatemessages
// 合规（消费其公开 Service 方法）。

// MediaFileMeta 是快照媒体元数据的契约形态（与 media.FileMeta 字段对齐，
// 避免 reports 直接 import media 包）。
type MediaFileMeta struct {
	StorageKey   string
	ThumbnailURL string
	Width        int
	Height       int
	MimeType     string
	FileName     string
	FileSize     int
	Duration     int
}

// MediaFileMetaSource 按 file_id 读取完成态文件的展示元数据。由 bootstrap 用
// media 模块 Service 适配注入。
type MediaFileMetaSource interface {
	CompletedFileMetaByID(ctx context.Context, fileID string) (MediaFileMeta, bool, error)
}

// NewMessageSnapshotSource 组装举报快照源。groupMessages/privateMessages 任一
// 为 nil 时跳过对应消息源；mediaFiles 为 nil 时媒体元数据降级为「查不到」，
// snapshot_media 回退为不可成立（按目标不存在处理）。
func NewMessageSnapshotSource(groupMessages *groupmessages.Service, privateMessages privatemessages.ReadService, mediaFiles MediaFileMetaSource) MessageSnapshotSource {
	return messageSnapshotSource{
		groupMessages:   groupMessages,
		privateMessages: privateMessages,
		mediaFiles:      mediaFiles,
	}
}

type messageSnapshotSource struct {
	groupMessages   *groupmessages.Service
	privateMessages privatemessages.ReadService
	mediaFiles      MediaFileMetaSource
}

var _ MessageSnapshotSource = messageSnapshotSource{}

// GetMessageSnapshot 先探群聊、再探私聊，命中后先校验举报人是会话参与方
// （群聊 = 在群成员；私聊 = 消息作者或会话对方），非参与者按目标不存在处理
// （防止持有任意 message_id 即可"举报后自读"冻结的私聊内容快照）。
// snapshot_content 仅 text 类型非 null；snapshot_media 仅媒体消息非 null
// （完整 MessageMedia）；target_chat_id：group = group_id，private = 相对发送者的对方 user_id。
func (s messageSnapshotSource) GetMessageSnapshot(ctx context.Context, messageID, reporterUserID string) (*ReportSnapshot, bool, error) {
	if s.groupMessages != nil {
		snap, found, err := s.groupMessages.GetMessageReportSnapshot(ctx, messageID)
		if err != nil {
			return nil, false, err
		}
		if found {
			isMember, err := s.groupMessages.IsGroupMember(ctx, snap.GroupID, reporterUserID)
			if err != nil {
				return nil, false, err
			}
			if !isMember {
				return nil, false, nil
			}
			return s.buildReportSnapshot(ctx, "group", snap.GroupID, snap.Type, snap.Content, snap.FileID, snap.SentAt)
		}
	}
	if s.privateMessages != nil {
		snap, found, err := s.privateMessages.GetMessageReportSnapshot(ctx, messageID)
		if err != nil {
			return nil, false, err
		}
		if found {
			if reporterUserID != snap.SenderUserID && reporterUserID != snap.TargetUserID {
				return nil, false, nil
			}
			return s.buildReportSnapshot(ctx, "private", snap.TargetUserID, snap.Type, snap.Content, snap.FileID, snap.SentAt)
		}
	}
	return nil, false, nil
}

// buildReportSnapshot 组装冻结快照：content 仅文本消息保留；媒体消息基于
// file_id 解析存储元数据生成完整 MessageMedia（spec api_spec_types.md：
// 管理员审核基于快照，不回查消息库——故元数据在受理时冻结进快照）。
func (s messageSnapshotSource) buildReportSnapshot(ctx context.Context, chatType, targetChatID, messageType, content, fileID, sentAt string) (*ReportSnapshot, bool, error) {
	snapshot := &ReportSnapshot{
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
		if s.mediaFiles == nil {
			// 无媒体元数据源（测试/降级装配）：快照无法完整成立。
			return nil, false, nil
		}
		raw, found, err := s.mediaFiles.CompletedFileMetaByID(ctx, fileID)
		if err != nil {
			return nil, false, err
		}
		if !found {
			// 消息引用的文件已不存在：按目标不存在处理（快照无法成立）。
			return nil, false, nil
		}
		snapshot.SnapshotMedia = raw.toMessageMedia()
	}
	return snapshot, true, nil
}

// toMessageMedia 映射为 spec MessageMedia：未知/不适用字段按契约置 null。
// url 使用 storage key，与聊天 WS media 负载（GetMediaForMessage）口径一致；
// duration 仅音视频非 null（0 = 不适用），width/height 仅图片/视频非 null。
func (m MediaFileMeta) toMessageMedia() *MessageMedia {
	media := &MessageMedia{URL: m.StorageKey}
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
