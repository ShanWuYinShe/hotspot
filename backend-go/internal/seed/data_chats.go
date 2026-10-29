package seed

import (
	"fmt"
	"time"
)

// data_chats.go — 私聊相关表行构建（private_chats / private_messages /
// private_chat_user_state / private_chat_read_state）。
//
// chat_key 规则与 privatechats 模块一致：双方 user_id 字典序用 ":" 连接
// （见 naming.go chatKey）。未读数由「对方消息中 sent_at 晚于本人已读游标」
// 动态计算，因此已读状态直接决定 QA 账号看到的未读角标。

func buildPrivateChatsTable(baseTime time.Time, chats []chatSpec) tableData {
	table := tableData{
		Table:   "private_chats",
		Columns: []column{col("chat_key"), col("user_a_id"), col("user_b_id"), col("created_at")},
	}
	for _, chat := range chats {
		userA, userB := chatUsers(chat.UserID, chat.TargetUserID)
		table.addRow(chatKey(chat.UserID, chat.TargetUserID), userA, userB, baseTime.Add(-chat.CreatedAgo))
	}
	return table
}

func buildPrivateMessagesTable(baseTime time.Time, chats []chatSpec, media *mediaBundle) tableData {
	table := tableData{
		Table: "private_messages",
		Columns: []column{
			col("message_id"), col("chat_key"), col("sender_user_id"), col("type"), col("content"),
			col("file_id"), col("share_code"), col("reply_to_message_id"), col("sent_at"),
			col("is_recalled"), col("recalled_at"), col("content_prev"), col("edited_at"),
		},
	}
	for _, chat := range chats {
		key := chatKey(chat.UserID, chat.TargetUserID)
		for _, message := range chat.Messages {
			content := any(message.Content)
			if message.Recalled {
				content = nil
			}
			var fileID any
			if message.MediaKey != "" {
				resolved, ok := media.chatImageFileIDs[message.MediaKey]
				if !ok {
					panic(fmt.Sprintf("私聊消息 %s 引用了未生成的聊天媒体 %s", message.MessageID, message.MediaKey))
				}
				fileID = resolved
			}
			var recalledAt any
			if message.Recalled {
				recalledAt = baseTime.Add(-message.RecalledAgo)
			}
			var editedAt any
			if message.EditedAgo > 0 {
				editedAt = baseTime.Add(-message.EditedAgo)
			}
			table.addRow(
				message.MessageID, key, message.SenderID, message.Type, content,
				fileID, nullableString(message.ShareCode), nullableString(message.ReplyTo), baseTime.Add(-message.SentAgo),
				message.Recalled, recalledAt, nullableString(message.ContentPrev), editedAt,
			)
		}
	}
	return table
}

func buildChatUserStatesTable(baseTime time.Time, chats []chatSpec) tableData {
	table := tableData{
		Table: "private_chat_user_state",
		Columns: []column{
			col("chat_key"), col("user_id"), col("is_muted"), col("is_pinned"),
			col("visible_since"), col("deleted_at"), col("updated_at"),
		},
	}
	for _, chat := range chats {
		key := chatKey(chat.UserID, chat.TargetUserID)
		updatedAt := baseTime.Add(-chat.CreatedAgo)
		states := make(map[string]chatUserStateSpec, 2)
		for _, state := range chat.States {
			states[state.UserID] = state
		}
		for _, userID := range []string{chat.UserID, chat.TargetUserID} {
			state := states[userID]
			table.addRow(key, userID, state.Muted, state.Pinned, any(nil), any(nil), updatedAt)
		}
	}
	return table
}

func buildChatReadStatesTable(baseTime time.Time, chats []chatSpec) tableData {
	table := tableData{
		Table: "private_chat_read_state",
		Columns: []column{
			col("chat_key"), col("user_id"), col("last_read_message_id"),
			col("last_read_sent_at"), col("updated_at"),
		},
	}
	for _, chat := range chats {
		key := chatKey(chat.UserID, chat.TargetUserID)
		sentAtByMessageID := make(map[string]time.Time, len(chat.Messages))
		for _, message := range chat.Messages {
			sentAtByMessageID[message.MessageID] = baseTime.Add(-message.SentAgo)
		}
		for _, state := range chat.ReadStates {
			sentAt, ok := sentAtByMessageID[state.LastReadMessageID]
			if !ok {
				panic(fmt.Sprintf("私聊已读游标引用了不存在的消息 %s", state.LastReadMessageID))
			}
			table.addRow(key, state.UserID, state.LastReadMessageID, sentAt, sentAt)
		}
	}
	return table
}
