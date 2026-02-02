/**
 * 私聊服务
 */

import { post, paginate } from '../client';
import { PRIVATE_CHATS, PRIVATE_MESSAGES } from '../endpoints';
import type {
	Conversation,
	MessageReactionItem,
	MessageLocation,
	PrivateChatListItem,
	PrivateMessageItem,
	ReactionType,
	SendMessageType
} from '$lib/types/api';

export const privateChatsService = {
	/**
	 * 获取私聊列表
	 */
	getChats(params?: { limit?: number; cursor?: string }) {
		return paginate<PrivateChatListItem>(PRIVATE_CHATS.ME, params);
	},

	/**
	 * 获取/创建与指定用户的私聊
	 */
	openChat(data: { target_user_id: number }) {
		return post<Conversation>(PRIVATE_CHATS.OPEN_THE_DOOR, data);
	},

	/**
	 * 更新私聊设置
	 */
	updateSettings(data: { target_user_id: number; is_muted?: boolean; is_pinned?: boolean }) {
		return post<Conversation>(PRIVATE_CHATS.UPDATE_SETTINGS, data);
	},

	/**
	 * 删除私聊
	 */
	deleteChat(data: { target_user_id: number }) {
		return post<void>(PRIVATE_CHATS.DELETE, data);
	}
};

export const privateMessagesService = {
	/**
	 * 发送私聊消息
	 */
	send(data: {
		target_user_id: number;
		type: SendMessageType;
		content?: string;
		file_id?: string;
		location?: MessageLocation;
		reply_to_message_id?: string;
		idempotencyKey?: string;
	}) {
		const { idempotencyKey, ...payload } = data;
		return post<void>(PRIVATE_MESSAGES.SEND, payload, { idempotencyKey });
	},

	/**
	 * 获取消息历史
	 */
	getHistory(params: { target_user_id: number; limit?: number; cursor?: string }) {
		return paginate<PrivateMessageItem>(PRIVATE_MESSAGES.HISTORY, params);
	},

	/**
	 * 编辑消息
	 */
	update(data: { message_id: string; content: string }) {
		return post<void>(PRIVATE_MESSAGES.UPDATE, data);
	},

	/**
	 * 撤回消息
	 */
	recall(data: { message_id: string }) {
		return post<void>(PRIVATE_MESSAGES.RECALL, data);
	},

	/**
	 * 删除消息（仅自己可见）
	 */
	delete(data: { message_id: string }) {
		return post<void>(PRIVATE_MESSAGES.DELETE, data);
	},

	/**
	 * 标记已读
	 */
	markAsRead(data: { target_user_id: number; up_to_message_id?: string }) {
		return post<void>(PRIVATE_MESSAGES.MARK_AS_READ, data);
	},

	/**
	 * 添加消息反应
	 */
	addReaction(data: { message_id: string; type: ReactionType }) {
		return post<void>(PRIVATE_MESSAGES.ADD_REACTION, data);
	},

	/**
	 * 移除消息反应
	 */
	removeReaction(data: { message_id: string; type: ReactionType }) {
		return post<void>(PRIVATE_MESSAGES.REMOVE_REACTION, data);
	},

	/**
	 * 获取消息反应列表
	 */
	getReactions(params: { message_id: string; limit?: number; cursor?: string }) {
		return paginate<MessageReactionItem>(PRIVATE_MESSAGES.REACTIONS, params);
	}
};
