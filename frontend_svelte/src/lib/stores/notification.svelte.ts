/**
 * 通知 Store
 * HTTP API 负责冷启动数据，WebSocket 负责实时增量
 */

import { browser } from '$app/environment';
import { authStore } from '$lib/stores/auth.svelte';
import { notificationsService } from '$lib/api/services';
import { isSuccess } from '$lib/api/client';
import type { ApiResponse, PaginatedData, NotificationItem, NotificationListRequest } from '$lib/types/api';
import { websocketStore } from '$lib/websocket';

// ============ 通知类型 ============

/** WebSocket 推送的实时通知项（用于类型兼容，列表数据来自 API） */
export interface RealtimeNotification {
	id: string;
	type: string;
	title: string;
	content: string;
	timestamp: string;
	read: boolean;
}

// ============ 通知 Store ============

class NotificationStore {
	// 未读数量（来自 HTTP API + WebSocket 增量）
	unreadCount = $state.raw(0);

	private cleanupFunctions: (() => void)[] = [];

	constructor() {
		if (!browser) return;

		this.registerNotificationHandlers();

		// 认证后拉取未读数量
		$effect.root(() => {
			$effect(() => {
				if (authStore.isAuthenticated) {
					this.fetchUnreadCount();
				}
			});
		});
	}

	/**
	 * 拉取未读数量（冷启动 / 重连时调用）
	 */
	async fetchUnreadCount(): Promise<void> {
		const res = await notificationsService.unreadCount();
		if (isSuccess(res) && res.data) {
			this.unreadCount = res.data.total;
		}
	}

	/**
	 * 分页获取通知列表
	 */
	async fetchHistory(params?: NotificationListRequest): Promise<ApiResponse<PaginatedData<NotificationItem>>> {
		return notificationsService.list(params ?? {});
	}

	/**
	 * 标记指定通知为已读
	 */
	async markAsRead(notificationIds: string[]): Promise<boolean> {
		if (notificationIds.length === 0) return true;
		const res = await notificationsService.markAsRead(notificationIds);
		if (isSuccess(res)) {
			this.unreadCount = Math.max(0, this.unreadCount - notificationIds.length);
			return true;
		}
		return false;
	}

	/**
	 * 标记全部通知为已读
	 */
	async markAllAsRead(): Promise<boolean> {
		const res = await notificationsService.markAllRead();
		if (isSuccess(res)) {
			this.unreadCount = 0;
			return true;
		}
		return false;
	}

	/**
	 * WebSocket 推送时增加未读计数
	 */
	private registerNotificationHandlers() {
		const increment = () => {
			this.unreadCount += 1;
		};

		this.cleanupFunctions.push(websocketStore.on('NewFollower', increment));
		this.cleanupFunctions.push(websocketStore.on('MutualFollow', increment));
		this.cleanupFunctions.push(websocketStore.on('MomentLiked', increment));
		this.cleanupFunctions.push(websocketStore.on('MomentCommented', increment));
		this.cleanupFunctions.push(websocketStore.on('CommentLiked', increment));
		this.cleanupFunctions.push(websocketStore.on('CommentReply', increment));
		this.cleanupFunctions.push(websocketStore.on('MessageMentioned', increment));
		this.cleanupFunctions.push(websocketStore.on('GroupInviteReceived', increment));
		this.cleanupFunctions.push(websocketStore.on('GroupApplicationReceived', increment));
		this.cleanupFunctions.push(websocketStore.on('SystemAnnouncement', increment));
	}

	destroy() {
		this.cleanupFunctions.forEach((cleanup) => cleanup());
		this.cleanupFunctions = [];
	}
}

export const notificationStore = new NotificationStore();
