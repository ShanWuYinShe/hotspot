/**
 * 消息未读计数 Store
 * 监听 WebSocket 推送的消息事件来更新未读计数
 */

import { websocketStore } from '$lib/websocket';
import { browser } from '$app/environment';
import { authStore } from './auth.svelte';
import type { PrivateMessageNewPayload, GroupMessageNewPayload } from '$lib/websocket/types';

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
	// 未读对话列表（使用 $state.raw 避免 Map 自赋值反模式）
	unreadConversations = $state.raw<Map<string, UnreadConversation>>(new Map());

	// 总未读消息数
	totalUnreadCount = $derived(
		Array.from(this.unreadConversations.values()).reduce((sum, conv) => sum + conv.count, 0)
	);

	// 未读对话数
	unreadConversationCount = $derived(this.unreadConversations.size);

	// 清理函数数组（用于取消 WebSocket 监听）
	private cleanupFunctions: (() => void)[] = [];
	// $effect.root() 清理函数
	private destroyEffectRoot: (() => void) | null = null;

	constructor() {
		if (!browser) return;

		this.registerMessageHandlers();

		// 响应式监听认证状态：登出时清空消息未读，防止跨用户数据泄漏
		this.destroyEffectRoot = $effect.root(() => {
			$effect(() => {
				if (!authStore.isAuthenticated) {
					this.unreadConversations = new Map();
				}
			});
		});
	}

	// ============ 内部工具方法 ============

	/** 不可变更新 Map：clone → mutate → reassign */
	private updateMap(mutator: (map: Map<string, UnreadConversation>) => void) {
		const next = new Map(this.unreadConversations);
		mutator(next);
		this.unreadConversations = next;
	}

	/** 统一的消息未读递增逻辑（私聊/群聊共用） */
	private incrementUnread(
		conversationId: string,
		type: 'private' | 'group',
		payload: { content?: string; sent_at: string; sender: { nickname: string } }
	) {
		const lastMessage = {
			content: payload.content ?? '',
			timestamp: payload.sent_at,
			senderName: payload.sender.nickname
		};

		this.updateMap((map) => {
			const existing = map.get(conversationId);
			map.set(conversationId, {
				id: conversationId,
				type,
				count: (existing?.count ?? 0) + 1,
				lastMessage
			});
		});
	}

	// ============ WebSocket 事件注册 ============

	private registerMessageHandlers() {
		this.cleanupFunctions.push(
			websocketStore.on('PrivateMessageNew', (payload: PrivateMessageNewPayload) => {
				if (payload.sender.user_id === authStore.currentUserId) return;
				this.incrementUnread(`private_${payload.sender.user_id}`, 'private', payload);
			})
		);

		this.cleanupFunctions.push(
			websocketStore.on('GroupMessageNew', (payload: GroupMessageNewPayload) => {
				if (payload.sender.user_id === authStore.currentUserId) return;
				this.incrementUnread(`group_${payload.group_id}`, 'group', payload);
			})
		);
	}

	// ============ 公共方法 ============

	markConversationAsRead(conversationId: string) {
		this.updateMap((map) => map.delete(conversationId));
	}

	markAllAsRead() {
		this.unreadConversations = new Map();
	}

	destroy() {
		if (this.destroyEffectRoot) {
			this.destroyEffectRoot();
			this.destroyEffectRoot = null;
		}
		this.cleanupFunctions.forEach((cleanup) => cleanup());
		this.cleanupFunctions = [];
	}
}

// ============ 导出单例 ============

export const messageStore = new MessageStore();
