package privatemessages

import (
	"context"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/testdb"
)

// 真实库口径验证私聊消息的单条查询/举报快照/审核撤回与媒体批量回填的 SQL
// 语义（stub 单测只能证明 Go 分支，无法证明列序/谓词与真实表结构一致）。
// 所有数据在 testdb.TxScope 事务内写入并回滚，ID 统一带
// privatemessagescov_ 前缀，避免与其他用例互相污染。
const queryInsertCovMediaSession = `
INSERT INTO media_upload_sessions (
    upload_id, user_id, storage_key, file_name, declared_file_size,
    declared_mime_type, category, status, file_id, created_at
) VALUES ($1, $2, $3, $4, $5, $6, 'image', 'completed', $7, $8)`

func TestPostgresIT_MessageLookupHelpers(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		alice := testdb.InsertUser(t, txFactory, "privatemessagescov_alice", "Alice")
		bob := testdb.InsertUser(t, txFactory, "privatemessagescov_bob", "Bob")
		carol := testdb.InsertUser(t, txFactory, "privatemessagescov_carol", "Carol")
		repo := newITRepo(txFactory)

		now := time.Now().UTC()
		fileID := "privatemessagescov_file_1"
		db, ok := txFactory.DB.(persistence.QueryableDB)
		if !ok {
			t.Fatal("tx factory is not queryable")
		}
		if err := db.Exec(ctx, queryInsertCovMediaSession,
			"privatemessagescov_upload_1", alice, "uploads/privatemessagescov/1.png",
			"1.png", 1024, "image/png", fileID, now); err != nil {
			t.Fatalf("insert media session: %v", err)
		}

		text := mustSendMessage(t, repo, alice, bob, SendPrivateMessageRequest{Type: "text", Content: "hello"})
		image := mustSendMessage(t, repo, alice, bob, SendPrivateMessageRequest{Type: "image", FileID: fileID})
		share := mustSendMessage(t, repo, alice, bob, SendPrivateMessageRequest{Type: "share", ShareCode: "SHARECODE01"})

		t.Run("get message for reply", func(t *testing.T) {
			result, found, err := repo.GetMessageForReply(ctx, text.MessageID, bob)
			if err != nil || !found {
				t.Fatalf("found=%v err=%v", found, err)
			}
			if result["message_id"] != text.MessageID || result["user_id"] != alice || result["nickname"] != "Alice" || result["content"] != "hello" {
				t.Fatalf("unexpected reply payload: %#v", result)
			}
			if _, hasShareCode := result["share_code"]; hasShareCode {
				t.Fatalf("text message must not expose share_code: %#v", result)
			}

			shareReply, found, err := repo.GetMessageForReply(ctx, share.MessageID, bob)
			if err != nil || !found {
				t.Fatalf("share found=%v err=%v", found, err)
			}
			if shareReply["share_code"] != "SHARECODE01" {
				t.Fatalf("expected share_code in reply payload, got %#v", shareReply)
			}

			// 非会话参与方不可见（回复目标校验按参与者过滤）。
			if _, found, err := repo.GetMessageForReply(ctx, text.MessageID, carol); err != nil || found {
				t.Fatalf("outsider must not resolve reply target: found=%v err=%v", found, err)
			}
			if _, found, err := repo.GetMessageForReply(ctx, "privatemessagescov_missing", bob); err != nil || found {
				t.Fatalf("missing message must not resolve: found=%v err=%v", found, err)
			}
		})

		t.Run("get media for message", func(t *testing.T) {
			result, found, err := repo.GetMediaForMessage(ctx, fileID, alice)
			if err != nil || !found {
				t.Fatalf("found=%v err=%v", found, err)
			}
			if result["url"] != "uploads/privatemessagescov/1.png" || result["file_name"] != "1.png" || result["file_size"] != 1024 || result["mime_type"] != "image/png" {
				t.Fatalf("unexpected media payload: %#v", result)
			}
			// 非上传者不可复用他人附件。
			if _, found, err := repo.GetMediaForMessage(ctx, fileID, bob); err != nil || found {
				t.Fatalf("other owner must not resolve media: found=%v err=%v", found, err)
			}
			if _, found, err := repo.GetMediaForMessage(ctx, "privatemessagescov_missing", alice); err != nil || found {
				t.Fatalf("missing media must not resolve: found=%v err=%v", found, err)
			}
		})

		t.Run("message exists and participants", func(t *testing.T) {
			found, sender, err := repo.MessageExists(ctx, text.MessageID)
			if err != nil || !found || sender != alice {
				t.Fatalf("exists=%v sender=%q err=%v", found, sender, err)
			}
			if found, _, err := repo.MessageExists(ctx, "privatemessagescov_missing"); err != nil || found {
				t.Fatalf("missing message must not exist: found=%v err=%v", found, err)
			}

			sender, receiver, found, err := repo.MessageParticipants(ctx, text.MessageID)
			if err != nil || !found || sender != alice || receiver != bob {
				t.Fatalf("participants=%q/%q found=%v err=%v", sender, receiver, found, err)
			}
			if _, _, found, err := repo.MessageParticipants(ctx, "privatemessagescov_missing"); err != nil || found {
				t.Fatalf("missing message must have no participants: found=%v err=%v", found, err)
			}
		})

		t.Run("report snapshot", func(t *testing.T) {
			snapshot, found, err := repo.GetMessageReportSnapshot(ctx, image.MessageID)
			if err != nil || !found {
				t.Fatalf("found=%v err=%v", found, err)
			}
			if snapshot.SenderUserID != alice || snapshot.TargetUserID != bob || snapshot.Type != "image" || snapshot.FileID != fileID {
				t.Fatalf("unexpected snapshot: %#v", snapshot)
			}
			if _, err := time.Parse(time.RFC3339Nano, snapshot.SentAt); err != nil {
				t.Fatalf("snapshot sent_at must be RFC3339Nano: %q (%v)", snapshot.SentAt, err)
			}
			if _, found, err := repo.GetMessageReportSnapshot(ctx, "privatemessagescov_missing"); err != nil || found {
				t.Fatalf("missing message must have no snapshot: found=%v err=%v", found, err)
			}
		})

		t.Run("history and batch reads backfill media", func(t *testing.T) {
			history, err := repo.History(ctx, bob, ListPrivateMessagesRequest{TargetUserID: alice})
			if err != nil {
				t.Fatalf("history: %v", err)
			}
			var mediaItem *PrivateMessageItem
			for i := range history.Items {
				if history.Items[i].MessageID == image.MessageID {
					mediaItem = &history.Items[i]
				}
			}
			if mediaItem == nil {
				t.Fatalf("image message missing from history: %#v", history.Items)
			}
			if mediaItem.Media == nil || mediaItem.Media.URL != "uploads/privatemessagescov/1.png" {
				t.Fatalf("expected media backfill with storage_key url, got %#v", mediaItem.Media)
			}
			if mediaItem.Media.FileName == nil || *mediaItem.Media.FileName != "1.png" || mediaItem.Media.FileSize == nil || *mediaItem.Media.FileSize != 1024 {
				t.Fatalf("expected full media contract fields, got %#v", mediaItem.Media)
			}

			key := conversationKey(alice, bob)
			lasts, err := repo.LastByChatKeys(ctx, bob, []string{key}, nil)
			if err != nil {
				t.Fatalf("last by chat keys: %v", err)
			}
			if item := lasts[key]; item.MessageID != share.MessageID {
				t.Fatalf("expected latest share message, got %#v", item)
			}

			unread, err := repo.UnreadCountsByChatKeys(ctx, bob, []string{key}, nil)
			if err != nil {
				t.Fatalf("unread counts: %v", err)
			}
			if unread[key] != 3 {
				t.Fatalf("unread = %d, want 3", unread[key])
			}
		})

		t.Run("recall as admin", func(t *testing.T) {
			hit, err := repo.RecallAsAdmin(ctx, image.MessageID)
			if err != nil || !hit {
				t.Fatalf("hit=%v err=%v", hit, err)
			}
			// 已撤回的消息不再接受 reaction（参与者查询按 is_recalled 过滤）。
			if _, _, found, err := repo.MessageParticipants(ctx, image.MessageID); err != nil || found {
				t.Fatalf("recalled message must not resolve participants: found=%v err=%v", found, err)
			}
			// 重复撤回幂等返回 false。
			if hit, err := repo.RecallAsAdmin(ctx, image.MessageID); err != nil || hit {
				t.Fatalf("repeat recall hit=%v err=%v, want false/nil", hit, err)
			}
		})
	})
}

func mustSendMessage(t *testing.T, repo *PostgresRepository, sender, target string, req SendPrivateMessageRequest) MessageSentResult {
	t.Helper()
	req.TargetUserID = target
	sent, err := repo.Send(context.Background(), sender, target, req)
	if err != nil {
		t.Fatalf("send %q: %v", req.Type, err)
	}
	return sent
}
