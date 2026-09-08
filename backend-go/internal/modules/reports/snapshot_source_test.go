package reports

import (
	"context"
	"errors"
	"testing"
	"time"

	"hotspot/backend-go/internal/modules/groupmessages"
	"hotspot/backend-go/internal/modules/privatemessages"
)

type mockPrivateMessagesReadService struct {
	snapshotFunc func(ctx context.Context, messageID string) (privatemessages.MessageReportSnapshot, bool, error)
}

func (m mockPrivateMessagesReadService) Recent(context.Context, string, int, *time.Time) (privatemessages.PaginatedMessages, error) {
	return privatemessages.PaginatedMessages{}, nil
}
func (m mockPrivateMessagesReadService) Last(context.Context, string, *time.Time) (*privatemessages.PrivateMessageItem, error) {
	return nil, nil
}
func (m mockPrivateMessagesReadService) UnreadCount(context.Context, string, *time.Time) (int, error) {
	return 0, nil
}
func (m mockPrivateMessagesReadService) LookupMessage(context.Context, string) (bool, string, error) {
	return false, "", nil
}
func (m mockPrivateMessagesReadService) IsReactionTargetVisible(context.Context, string) (bool, bool, error) {
	return false, false, nil
}
func (m mockPrivateMessagesReadService) GetMessageReportSnapshot(ctx context.Context, messageID string) (privatemessages.MessageReportSnapshot, bool, error) {
	if m.snapshotFunc != nil {
		return m.snapshotFunc(ctx, messageID)
	}
	return privatemessages.MessageReportSnapshot{}, false, nil
}

type mockMediaFileMetaSource struct {
	metaFunc func(ctx context.Context, fileID string) (MediaFileMeta, bool, error)
}

func (m mockMediaFileMetaSource) CompletedFileMetaByID(ctx context.Context, fileID string) (MediaFileMeta, bool, error) {
	if m.metaFunc != nil {
		return m.metaFunc(ctx, fileID)
	}
	return MediaFileMeta{}, false, nil
}

func TestMessageSnapshotSourcePrivateMessage(t *testing.T) {
	ctx := context.Background()

	// 1. 正常文本私聊消息快照（举报人为接收者）
	pms := mockPrivateMessagesReadService{
		snapshotFunc: func(ctx context.Context, messageID string) (privatemessages.MessageReportSnapshot, bool, error) {
			return privatemessages.MessageReportSnapshot{
				SenderUserID: "user_sender",
				TargetUserID: "user_receiver",
				Type:         "text",
				Content:      "敏感违法内容",
				SentAt:       "2026-09-08T12:00:00Z",
			}, true, nil
		},
	}

	source := NewMessageSnapshotSource(nil, pms, nil)
	snap, found, err := source.GetMessageSnapshot(ctx, "msg_1", "user_receiver")
	if err != nil || !found || snap == nil {
		t.Fatalf("expected snapshot found, got snap=%v, found=%v, err=%v", snap, found, err)
	}
	if snap.ChatType != "private" || snap.MessageType != "text" || snap.SnapshotContent == nil || *snap.SnapshotContent != "敏感违法内容" {
		t.Fatalf("unexpected snapshot data: %+v", snap)
	}

	// 2. 正常文本私聊消息快照（举报人为发送者自己）
	snapSender, found, err := source.GetMessageSnapshot(ctx, "msg_1", "user_sender")
	if err != nil || !found || snapSender == nil {
		t.Fatalf("expected sender can report message, got %v", err)
	}

	// 3. 非会话参与者举报私聊消息（防止探测和越权，必须返回 found=false）
	snapThirdParty, found, err := source.GetMessageSnapshot(ctx, "msg_1", "user_third_party")
	if err != nil || found || snapThirdParty != nil {
		t.Fatalf("expected third party cannot snapshot private message, got found=%v", found)
	}

	// 4. 私聊带媒体附件的消息快照
	mediaSource := mockMediaFileMetaSource{
		metaFunc: func(ctx context.Context, fileID string) (MediaFileMeta, bool, error) {
			return MediaFileMeta{
				StorageKey:   "photos/report_evidence.png",
				ThumbnailURL: "photos/thumb.png",
				Width:        1920,
				Height:       1080,
				MimeType:     "image/png",
				FileName:     "evidence.png",
				FileSize:     102400,
			}, true, nil
		},
	}
	pmsMedia := mockPrivateMessagesReadService{
		snapshotFunc: func(ctx context.Context, messageID string) (privatemessages.MessageReportSnapshot, bool, error) {
			return privatemessages.MessageReportSnapshot{
				SenderUserID: "user_sender",
				TargetUserID: "user_receiver",
				Type:         "image",
				FileID:       "file_123",
				SentAt:       "2026-09-08T12:00:00Z",
			}, true, nil
		},
	}
	sourceWithMedia := NewMessageSnapshotSource(nil, pmsMedia, mediaSource)
	mediaSnap, found, err := sourceWithMedia.GetMessageSnapshot(ctx, "msg_media", "user_receiver")
	if err != nil || !found || mediaSnap == nil {
		t.Fatalf("expected media snapshot found, got %v", err)
	}
	if mediaSnap.SnapshotMedia == nil || mediaSnap.SnapshotMedia.URL != "photos/report_evidence.png" {
		t.Fatalf("unexpected media snapshot: %+v", mediaSnap.SnapshotMedia)
	}
	if mediaSnap.SnapshotContent != nil {
		t.Fatal("image message should not have snapshot_content")
	}

	// 5. 媒体文件已不存在（found=false）
	mediaSourceMissing := mockMediaFileMetaSource{
		metaFunc: func(ctx context.Context, fileID string) (MediaFileMeta, bool, error) {
			return MediaFileMeta{}, false, nil
		},
	}
	sourceMissingMedia := NewMessageSnapshotSource(nil, pmsMedia, mediaSourceMissing)
	snapMissing, found, err := sourceMissingMedia.GetMessageSnapshot(ctx, "msg_media", "user_receiver")
	if err != nil || found || snapMissing != nil {
		t.Fatalf("expected missing media to return found=false, got found=%v, err=%v", found, err)
	}

	// 6. 消息服务报错向外透传
	pmsErr := mockPrivateMessagesReadService{
		snapshotFunc: func(ctx context.Context, messageID string) (privatemessages.MessageReportSnapshot, bool, error) {
			return privatemessages.MessageReportSnapshot{}, false, errors.New("db error")
		},
	}
	sourceErr := NewMessageSnapshotSource(nil, pmsErr, mediaSource)
	_, _, err = sourceErr.GetMessageSnapshot(ctx, "msg_err", "user_receiver")
	if err == nil {
		t.Fatal("expected error propagated from private messages service")
	}
}

func TestMessageSnapshotSourceGroupMessage(t *testing.T) {
	ctx := context.Background()

	// 使用内存版 groupmessages.InMemoryRepository
	repo := groupmessages.NewInMemoryRepository()
	res, err := repo.Send(ctx, groupmessages.SendGroupMessageRequest{
		GroupID:      "group_999",
		SenderUserID: "user_member_a",
		Type:         "text",
		Content:      "群内的不当言论",
	})
	if err != nil {
		t.Fatalf("send group message failed: %v", err)
	}

	gmSvc := groupmessages.NewServiceWithRepo(repo)
	source := NewMessageSnapshotSource(gmSvc, nil, nil)

	// 1. 发送者（群成员）举报：快照成功生成
	snap, found, err := source.GetMessageSnapshot(ctx, res.MessageID, "user_member_a")
	if err != nil || !found || snap == nil {
		t.Fatalf("expected snapshot found for group member, got %v", err)
	}
	if snap.ChatType != "group" || snap.TargetChatID != "group_999" || snap.SnapshotContent == nil || *snap.SnapshotContent != "群内的不当言论" {
		t.Fatalf("unexpected group snapshot: %+v", snap)
	}

	// 2. 非群成员举报：必须拒绝（防止外部遍历 messageID 自读群聊记录）
	snapNonMember, found, err := source.GetMessageSnapshot(ctx, res.MessageID, "user_outsider")
	if err != nil || found || snapNonMember != nil {
		t.Fatalf("expected outsider cannot snapshot group message, got found=%v", found)
	}

	// 3. 不存在的消息 ID
	snapMissing, found, err := source.GetMessageSnapshot(ctx, "gm_nonexistent", "user_member_a")
	if err != nil || found || snapMissing != nil {
		t.Fatalf("expected nonexistent message to return found=false, got found=%v", found)
	}
}

func TestMediaFileMetaToMessageMedia(t *testing.T) {
	meta := MediaFileMeta{
		StorageKey:   "videos/bad_video.mp4",
		ThumbnailURL: "videos/thumb.jpg",
		Width:        1280,
		Height:       720,
		MimeType:     "video/mp4",
		FileName:     "bad.mp4",
		FileSize:     5000000,
		Duration:     120,
	}

	msgMedia := meta.toMessageMedia()
	if msgMedia.URL != "videos/bad_video.mp4" {
		t.Fatalf("unexpected URL: %s", msgMedia.URL)
	}
	if msgMedia.ThumbnailURL == nil || *msgMedia.ThumbnailURL != "videos/thumb.jpg" {
		t.Fatalf("unexpected ThumbnailURL: %v", msgMedia.ThumbnailURL)
	}
	if msgMedia.Duration == nil || *msgMedia.Duration != 120 {
		t.Fatalf("unexpected Duration: %v", msgMedia.Duration)
	}
	if msgMedia.Width == nil || *msgMedia.Width != 1280 {
		t.Fatalf("unexpected Width: %v", msgMedia.Width)
	}

	// 最小空字段验证
	zeroMeta := MediaFileMeta{
		StorageKey: "files/plain.txt",
	}
	zeroMedia := zeroMeta.toMessageMedia()
	if zeroMedia.URL != "files/plain.txt" {
		t.Fatalf("unexpected zeroMedia URL: %s", zeroMedia.URL)
	}
	if zeroMedia.ThumbnailURL != nil || zeroMedia.Duration != nil || zeroMedia.Width != nil || zeroMedia.FileSize != nil {
		t.Fatalf("expected nil for non-existent dimensions, got %+v", zeroMedia)
	}
}

func TestStrPtrOrNull(t *testing.T) {
	if got := strPtrOrNull(""); got != nil {
		t.Fatalf("expected nil for empty string, got %v", got)
	}
	if got := strPtrOrNull("hello"); got == nil || *got != "hello" {
		t.Fatalf("expected ptr to hello, got %v", got)
	}
}
