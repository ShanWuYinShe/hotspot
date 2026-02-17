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

// ============ 通知 Store ============

class NotificationStore {
	// 未读数量（来自 HTTP API + WebSocket 增量）
	unreadCount = $state(0);

	private cleanupFunctions: (() => void)[] = [];
	// $effect.root() 清理函数
	private destroyEffectRoot: (() => void) | null = null;
	// 竞态保护：当前请求版本号
	private fetchVersion = 0;

	constructor() {
		if (!browser) return;

		this.registerNotificationHandlers();

		// 响应式监听认证状态：登录时拉取未读数量，登出时清空
		this.destroyEffectRoot = $effect.root(() => {
			$effect(() => {
				if (authStore.isAuthenticated) {
					this.fetchUnreadCount();
				} else {
					// 登出时清空未读计数，防止跨用户数据泄漏
					this.unreadCount = 0;
					// 使之前的请求结果无效
					this.fetchVersion++;
				}
			});
		});
	}

	/**
	 * 拉取未读数量（冷启动 / 重连时调用）
	 */
	async fetchUnreadCount(): Promise<void> {
		const version = ++this.fetchVersion;
		const res = await notificationsService.unreadCount();
		// 竞态保护：仅接受最新请求的结果（防止登出后过期响应写入）
		if (version !== this.fetchVersion) return;
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
		const res = await notificationsService.markAsRead({ notification_ids: notificationIds });
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
		// 清理 $effect.root()
		if (this.destroyEffectRoot) {
			this.destroyEffectRoot();
			this.destroyEffectRoot = null;
		}
		this.cleanupFunctions.forEach((cleanup) => cleanup());
		this.cleanupFunctions = [];
	}
}

export const notificationStore = new NotificationStore();
