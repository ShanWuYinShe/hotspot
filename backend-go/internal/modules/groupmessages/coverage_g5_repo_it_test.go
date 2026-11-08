package groupmessages

import (
	"context"
	"errors"
	"testing"

	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/testdb"
)

// gmCovPrefix 是本文件所有真实库测试数据的唯一前缀：与其他 agent/组的
// 固定 ID 隔离，事务回滚保证不污染测试库。
const gmCovPrefix = "groupmessagescov_"

// TestPostgresIT_HistoryPayloadAndMediaBackfill 覆盖 History 的载荷列回填：
// share_code / reply_to / mentions / media 批量回填（mediaByFileIDs 单次往返）。
// 这些分支此前 0% 覆盖，必须用真实库才能验证 SQL 列与 Scan 布局一致。
func TestPostgresIT_HistoryPayloadAndMediaBackfill(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		owner := testdb.InsertUser(t, txFactory, gmCovPrefix+"media_owner", "MediaOwner")
		groupID := createTestGroup(t, ctx, txFactory, owner, "GM Cov Media Group")
		repo := NewPostgresRepository(txFactory)
		qdb, ok := txFactory.DB.(persistence.QueryableDB)
		if !ok {
			t.Fatal("factory.DB is not QueryableDB")
		}

		const fileID = gmCovPrefix + "file_1"
		const storageKey = "uploads/" + gmCovPrefix + "media_1.jpg"
		const thumbURL = "https://cdn.example/" + gmCovPrefix + "thumb.jpg"
		if err := qdb.Exec(ctx, `INSERT INTO media_upload_sessions
			(upload_id, user_id, storage_key, file_name, declared_file_size, declared_mime_type,
			 category, status, file_id, thumbnail_url, duration, width, height, created_at, completed_at)
			VALUES ($1,$2,$3,$4,$5,$6,'image','completed',$7,$8,$9,$10,$11,NOW(),NOW())`,
			gmCovPrefix+"upload_1", owner, storageKey, "cov.jpg", 4096, "image/jpeg",
			fileID, thumbURL, 0, 800, 600); err != nil {
			t.Fatalf("insert media session: %v", err)
		}

		mediaMsg, err := repo.Send(ctx, SendGroupMessageRequest{
			GroupID:      groupID,
			SenderUserID: owner,
			Type:         "image",
			FileID:       fileID,
		})
		if err != nil {
			t.Fatalf("send media message: %v", err)
		}
		replyMsg, err := repo.Send(ctx, SendGroupMessageRequest{
			GroupID:          groupID,
			SenderUserID:     owner,
			Type:             "text",
			Content:          "reply payload",
			ShareCode:        "gmcovshare1",
			ReplyToMessageID: mediaMsg.MessageID,
			Mentions:         []string{owner},
		})
		if err != nil {
			t.Fatalf("send reply message: %v", err)
		}

		history, err := repo.History(ctx, groupID, nil)
		if err != nil {
			t.Fatalf("history: %v", err)
		}
		var mediaItem, replyItem *GroupMessageItem
		for i := range history.Items {
			switch history.Items[i].MessageID {
			case mediaMsg.MessageID:
				mediaItem = &history.Items[i]
			case replyMsg.MessageID:
				replyItem = &history.Items[i]
			}
		}
		if mediaItem == nil || replyItem == nil {
			t.Fatalf("history items = %#v", history.Items)
		}

		// 媒体消息：file_id 非空 → 触发批量回填（mediaByFileIDs）。
		if mediaItem.Media == nil {
			t.Fatal("media backfill must be present for image message")
		}
		if got := mediaItem.Media["url"]; got != storageKey {
			t.Fatalf("media url = %v, want %q", got, storageKey)
		}
		if w, ok := mediaItem.Media["width"].(*int); !ok || w == nil || *w != 800 {
			t.Fatalf("media width = %#v", mediaItem.Media["width"])
		}
		if h, ok := mediaItem.Media["height"].(*int); !ok || h == nil || *h != 600 {
			t.Fatalf("media height = %#v", mediaItem.Media["height"])
		}
		if got, ok := mediaItem.Media["mime_type"].(*string); !ok || got == nil || *got != "image/jpeg" {
			t.Fatalf("media mime_type = %#v", mediaItem.Media["mime_type"])
		}
		if got, ok := mediaItem.Media["file_name"].(*string); !ok || got == nil || *got != "cov.jpg" {
			t.Fatalf("media file_name = %#v", mediaItem.Media["file_name"])
		}
		if got, ok := mediaItem.Media["thumbnail_url"].(*string); !ok || got == nil || *got != thumbURL {
			t.Fatalf("media thumbnail_url = %#v", mediaItem.Media["thumbnail_url"])
		}
		if size, ok := mediaItem.Media["file_size"].(int); !ok || size != 4096 {
			t.Fatalf("media file_size = %#v", mediaItem.Media["file_size"])
		}

		// 回复消息：share_code / reply_to / mentions 恒在。
		if replyItem.Media != nil {
			t.Fatalf("text message must not carry media: %#v", replyItem.Media)
		}
		if replyItem.ShareCode == nil || *replyItem.ShareCode != "gmcovshare1" {
			t.Fatalf("share_code = %#v", replyItem.ShareCode)
		}
		if replyItem.ReplyTo == nil || replyItem.ReplyTo.MessageID != mediaMsg.MessageID {
			t.Fatalf("reply_to = %#v", replyItem.ReplyTo)
		}
		if replyItem.ReplyTo.Sender.UserID != owner || replyItem.ReplyTo.Type != "image" {
			t.Fatalf("reply_to payload = %#v", replyItem.ReplyTo)
		}
		if len(replyItem.Mentions) != 1 || replyItem.Mentions[0] != owner {
			t.Fatalf("mentions = %#v", replyItem.Mentions)
		}
		if replyItem.ReadCount == nil || *replyItem.ReadCount != 0 {
			t.Fatalf("read_count = %#v, want 0", replyItem.ReadCount)
		}

		// GetMediaForMessage 的 8 键负载（WS media 同构）：命中路径。
		media, found, err := repo.GetMediaForMessage(ctx, fileID, owner)
		if err != nil || !found {
			t.Fatalf("GetMediaForMessage found=%v err=%v", found, err)
		}
		if media["url"] != storageKey {
			t.Fatalf("GetMediaForMessage url = %v", media["url"])
		}
		for _, key := range []string{"url", "thumbnail_url", "width", "height", "mime_type", "file_name", "file_size", "duration"} {
			if _, ok := media[key]; !ok {
				t.Fatalf("GetMediaForMessage missing key %q: %#v", key, media)
			}
		}
		// owner 过滤：别的用户不能引用该文件。
		if _, found, err := repo.GetMediaForMessage(ctx, fileID, gmCovPrefix+"other_user"); err != nil || found {
			t.Fatalf("GetMediaForMessage other owner found=%v err=%v", found, err)
		}
	})
}

// TestPostgresIT_MuteStatusAndLatestMessageNotFound 覆盖禁言状态命中/未命中
// 与空群 LatestMessageID（pgx.ErrNoRows → ErrMessageNotFound）分支。
func TestPostgresIT_MuteStatusAndLatestMessageNotFound(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		owner := testdb.InsertUser(t, txFactory, gmCovPrefix+"mute_owner", "MuteOwner")
		groupID := createTestGroup(t, ctx, txFactory, owner, "GM Cov Mute Group")
		repo := NewPostgresRepository(txFactory)
		qdb, ok := txFactory.DB.(persistence.QueryableDB)
		if !ok {
			t.Fatal("factory.DB is not QueryableDB")
		}

		// 未禁言成员：found=true, isMuted=false。
		muted, until, found, err := repo.MemberMuteStatus(ctx, groupID, owner)
		if err != nil || !found || muted || until != "" {
			t.Fatalf("unmuted member = (%v,%q,%v,%v)", muted, until, found, err)
		}

		// 永久禁言（muted_until 为空）：is_muted 直接生效。
		if err := qdb.Exec(ctx, `UPDATE group_members SET is_muted = TRUE, muted_until = NULL
			WHERE group_id = $1 AND user_id = $2`, groupID, owner); err != nil {
			t.Fatalf("mute member: %v", err)
		}
		muted, _, found, err = repo.MemberMuteStatus(ctx, groupID, owner)
		if err != nil || !found || !muted {
			t.Fatalf("permanently muted member = (%v,%v,%v)", muted, found, err)
		}

		// 禁言已到期：自动恢复（is_muted 但 muted_until 已过 → false）。
		if err := qdb.Exec(ctx, `UPDATE group_members SET muted_until = NOW() - INTERVAL '1 minute'
			WHERE group_id = $1 AND user_id = $2`, groupID, owner); err != nil {
			t.Fatalf("expire mute: %v", err)
		}
		muted, _, found, err = repo.MemberMuteStatus(ctx, groupID, owner)
		if err != nil || !found || muted {
			t.Fatalf("expired mute member = (%v,%v,%v)", muted, found, err)
		}

		// 非成员：found=false（不再泄漏禁言状态）。
		muted, until, found, err = repo.MemberMuteStatus(ctx, groupID, gmCovPrefix+"non_member")
		if err != nil || found || muted || until != "" {
			t.Fatalf("non-member mute status = (%v,%q,%v,%v)", muted, until, found, err)
		}

		// 空群没有消息：LatestMessageID 返回 ErrMessageNotFound（mark_read 推送兜底）。
		if _, err := repo.LatestMessageID(ctx, gmCovPrefix+"empty_group"); !errors.Is(err, ErrMessageNotFound) {
			t.Fatalf("LatestMessageID empty group err = %v, want ErrMessageNotFound", err)
		}
	})
}
