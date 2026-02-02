/**
 * 消息未读计数 Store
 * 监听 WebSocket 推送的消息事件来更新未读计数
 */

import { websocketStore } from '$lib/websocket';
import { browser } from '$app/environment';
import { authStore } from './auth.svelte';

// ============ 消息类型 ============

export interface UnreadConversation {
	id: string; // 对话ID（私聊为user_id，群聊为group_id）
	type: 'private' | 'group';
	count: number;
	lastMessage?: {
		content: string;
		timestamp: string;
		senderName: string;
	};
}

// ============ 消息未读计数 Store ============

class MessageStore {
	// 未读对话列表
	unreadConversations = $state<Map<string, UnreadConversation>>(new Map());

	// 总未读消息数
	totalUnreadCount = $derived(
		Array.from(this.unreadConversations.values()).reduce((sum, conv) => sum + conv.count, 0)
	);

	// 未读对话数
	unreadConversationCount = $derived(this.unreadConversations.size);

	// 清理函数数组（用于取消 WebSocket 监听）
	private cleanupFunctions: (() => void)[] = [];

	constructor() {
		if (!browser) return;

		// 监听消息事件
		this.registerMessageHandlers();
	}

	/**
	 * 注册消息事件处理器
	 */
	private registerMessageHandlers() {
		// 监听私聊新消息（现在有类型提示了！）
		this.cleanupFunctions.push(
			websocketStore.on('PrivateMessageNew', (payload) => {
				// ✅ 只统计别人发给我的消息（过滤掉自己发送的消息）
				if (payload.sender.user_id === authStore.currentUserId) {
					return;
				}

				const conversationId = `private_${payload.sender.user_id}`;
				const existing = this.unreadConversations.get(conversationId);

				if (existing) {
					this.unreadConversations.set(conversationId, {
						...existing,
						count: existing.count + 1,
						lastMessage: {
							content: payload.content ?? '',
							timestamp: payload.sent_at,
							senderName: payload.sender.nickname
						}
					});
				} else {
					this.unreadConversations.set(conversationId, {
						id: conversationId,
						type: 'private',
						count: 1,
						lastMessage: {
							content: payload.content ?? '',
							timestamp: payload.sent_at,
							senderName: payload.sender.nickname
						}
					});
				}

				// 触发响应式更新
				this.unreadConversations = this.unreadConversations;
			})
		);

		// 监听群聊新消息（现在有类型提示了！）
		this.cleanupFunctions.push(
			websocketStore.on('GroupMessageNew', (payload) => {
				// ✅ 只统计别人发的消息（过滤掉自己发送的消息）
				if (payload.sender.user_id === authStore.currentUserId) {
					return;
				}

				const conversationId = `group_${payload.group_chat_id}`;
				const existing = this.unreadConversations.get(conversationId);

				if (existing) {
					this.unreadConversations.set(conversationId, {
						...existing,
						count: existing.count + 1,
						lastMessage: {
							content: payload.content ?? '',
							timestamp: payload.sent_at,
							senderName: payload.sender.nickname
						}
					});
				} else {
					this.unreadConversations.set(conversationId, {
						id: conversationId,
						type: 'group',
						count: 1,
						lastMessage: {
							content: payload.content ?? '',
							timestamp: payload.sent_at,
							senderName: payload.sender.nickname
						}
					});
				}

				// 触发响应式更新
				this.unreadConversations = this.unreadConversations;
			})
		);
	}

	/**
	 * 标记对话为已读
	 */
	markConversationAsRead(conversationId: string) {
		this.unreadConversations.delete(conversationId);
		this.unreadConversations = this.unreadConversations;
	}

	/**
	 * 标记所有消息为已读
	 */
	markAllAsRead() {
		this.unreadConversations.clear();
		this.unreadConversations = this.unreadConversations;
	}

	/**
	 * 清理所有 WebSocket 监听器
	 */
	destroy() {
		this.cleanupFunctions.forEach((cleanup) => cleanup());
		this.cleanupFunctions = [];
	}
}

// ============ 导出单例 ============

export const messageStore = new MessageStore();
