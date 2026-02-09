/**
 * 通知 Store
 * 处理 WebSocket 推送的通知事件
 */

import { websocketStore } from '$lib/websocket';
import { browser } from '$app/environment';

// ============ 通知类型 ============

/** WebSocket 推送的实时通知项（区别于 API 返回的 NotificationItem） */
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
	// 未读通知列表
	unreadNotifications = $state.raw<RealtimeNotification[]>([]);

	// 未读数量
	unreadCount = $derived(this.unreadNotifications.length);

	// 清理函数数组（用于取消 WebSocket 监听）
	private cleanupFunctions: (() => void)[] = [];

	constructor() {
		if (!browser) return;

		// 监听各种通知事件
		this.registerNotificationHandlers();
	}

	/**
	 * 注册通知事件处理器（现在所有 payload 都有类型提示了！）
	 */
	private registerNotificationHandlers() {
		// 新增关注者
		this.cleanupFunctions.push(websocketStore.on('NewFollower', (payload) => {
			this.addNotification({
				id: `follower_${Date.now()}`,
				type: 'follower',
				title: '新增关注者',
				content: `${payload.user.nickname} 关注了你`,
				timestamp: payload.followed_at,
				read: false
			});
		}));

		// 互相关注
		this.cleanupFunctions.push(websocketStore.on('MutualFollow', (payload) => {
			this.addNotification({
				id: `mutual_${Date.now()}`,
				type: 'mutual',
				title: '新好友',
				content: `你和 ${payload.user.nickname} 成为了好友`,
				timestamp: payload.mutual_at,
				read: false
			});
		}));

		// 动态被点赞
		this.cleanupFunctions.push(websocketStore.on('MomentLiked', (payload) => {
			this.addNotification({
				id: `like_${payload.moment_id}_${Date.now()}`,
				type: 'like',
				title: '动态被点赞',
				content: `${payload.user.nickname} 赞了你的动态`,
				timestamp: payload.liked_at,
				read: false
			});
		}));

		// 动态被评论
		this.cleanupFunctions.push(websocketStore.on('MomentCommented', (payload) => {
			this.addNotification({
				id: `comment_${payload.comment_id}`,
				type: 'comment',
				title: '动态被评论',
				content: `${payload.user.nickname}: ${payload.content}`,
				timestamp: payload.commented_at,
				read: false
			});
		}));

		// 评论被回复
		this.cleanupFunctions.push(websocketStore.on('CommentReply', (payload) => {
			this.addNotification({
				id: `reply_${payload.reply_id}`,
				type: 'reply',
				title: '评论被回复',
				content: `${payload.user.nickname} 回复了你`,
				timestamp: payload.replied_at,
				read: false
			});
		}));

		// 群组邀请
		this.cleanupFunctions.push(websocketStore.on('GroupInviteReceived', (payload) => {
			this.addNotification({
				id: `invite_${payload.invite_id}`,
				type: 'group_invite',
				title: '群组邀请',
				content: `${payload.inviter.nickname} 邀请你加入 ${payload.group_name}`,
				timestamp: payload.created_at,
				read: false
			});
		}));

		// 入群申请
		this.cleanupFunctions.push(websocketStore.on('GroupApplicationReceived', (payload) => {
			this.addNotification({
				id: `application_${payload.application_id}`,
				type: 'group_application',
				title: '入群申请',
				content: `${payload.applicant.nickname} 申请加入群组`,
				timestamp: payload.created_at,
				read: false
			});
		}));

		// 系统公告
		this.cleanupFunctions.push(websocketStore.on('SystemAnnouncement', (payload) => {
			this.addNotification({
				id: `announcement_${payload.announcement_id}`,
				type: 'system',
				title: payload.title,
				content: payload.content,
				timestamp: payload.created_at,
				read: false
			});
		}));
	}

	/**
	 * 添加通知
	 */
	private addNotification(notification: RealtimeNotification) {
		// 检查是否已存在
		const exists = this.unreadNotifications.some((n) => n.id === notification.id);
		if (!exists) {
			this.unreadNotifications = [notification, ...this.unreadNotifications];

			// 最多保留 100 条未读通知
			if (this.unreadNotifications.length > 100) {
				this.unreadNotifications = this.unreadNotifications.slice(0, 100);
			}
		}
	}

	/**
	 * 标记所有通知为已读
	 */
	markAllAsRead() {
		this.unreadNotifications = [];
	}

	/**
	 * 标记单个通知为已读
	 */
	markAsRead(id: string) {
		this.unreadNotifications = this.unreadNotifications.filter((n) => n.id !== id);
	}

	/**
	 * 清理所有 WebSocket 监听器
	 * 注：虽然这是单例通常不需要清理，但为了一致性和可测试性提供此方法
	 */
	destroy() {
		this.cleanupFunctions.forEach((cleanup) => cleanup());
		this.cleanupFunctions = [];
	}
}

// ============ 导出单例 ============

export const notificationStore = new NotificationStore();
