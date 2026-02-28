/**
 * 私聊服务
 */

import { post, paginate, generateIdempotencyKey } from '../client';
import { PRIVATE_CHATS, PRIVATE_MESSAGES } from '../endpoints';
import type {
	PrivateChatItem,
	PrivateChatOpenResult,
	PrivateMessageItem,
	MessageSent,
	SendMessageType,
	PaginationParams
} from '$lib/types/api';

export const privateChatsService = {
	/**
	 * 获取私聊列表
	 */
	getChats(params?: { pagination?: PaginationParams }) {
		return paginate<PrivateChatItem>(PRIVATE_CHATS.ME.LIST, params);
	},

	/**
	 * 获取/创建与指定用户的私聊（返回聊天详情 + 最近消息）
	 */
	openChat(data: { target_user_id: string }) {
		return post<PrivateChatOpenResult>(PRIVATE_CHATS.OPEN_THE_DOOR, data);
	},

	/**
	 * 更新私聊设置
	 */
	updateSettings(data: { target_user_id: string; is_muted?: boolean; is_pinned?: boolean }) {
		return post<void>(PRIVATE_CHATS.ME.UPDATE_SETTINGS, data);
	},

	/**
	 * 删除我的私聊会话（单边删除）
	 */
	deleteChat(data: { target_user_id: string }) {
		return post<void>(PRIVATE_CHATS.ME.DELETE, data);
	}
};

export const privateMessagesService = {
	/**
	 * 发送私聊消息
	 * spec: MUST 幂等，自动生成 Idempotency-Key
	 */
	send(data: {
		target_user_id: string;
		type: SendMessageType;
		content?: string;
		file_id?: string;
		share_code?: string;
		reply_to_message_id?: string;
	}) {
		return post<MessageSent>(PRIVATE_MESSAGES.SEND, data, { idempotencyKey: generateIdempotencyKey() });
	},

	/**
	 * 获取消息历史
	 */
	getHistory(params: { target_user_id: string; pagination?: PaginationParams }) {
		return paginate<PrivateMessageItem>(PRIVATE_MESSAGES.GET_HISTORY, params);
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
	 * 标记已读
	 */
	markAsRead(data: { target_user_id: string; up_to_message_id?: string }) {
		return post<void>(PRIVATE_MESSAGES.ME.MARK_AS_READ, data);
	},

	/**
	 * 标记全部私聊已读
	 */
	markAllRead() {
		return post<void>(PRIVATE_MESSAGES.ME.MARK_ALL_READ, {});
	}
};
