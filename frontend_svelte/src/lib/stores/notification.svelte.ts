/**
 * 通知 Store
 * 处理 WebSocket 推送的通知事件
 */

import { websocketStore } from '$lib/websocket';
import { browser } from '$app/environment';

// ============ 通知类型 ============

export interface NotificationItem {
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
	unreadNotifications = $state<NotificationItem[]>([]);

	// 未读数量
	unreadCount = $derived(this.unreadNotifications.length);

	constructor() {
		if (!browser) return;

		// 监听各种通知事件
		this.registerNotificationHandlers();
	}

	/**
	 * 注册通知事件处理器
	 */
	private registerNotificationHandlers() {
		// 新增关注者
		websocketStore.on('NewFollower', (payload: any) => {
			this.addNotification({
				id: `follower_${Date.now()}`,
				type: 'follower',
				title: '新增关注者',
				content: `${payload.user.nickname} 关注了你`,
				timestamp: payload.followed_at,
				read: false
			});
		});

		// 互相关注
		websocketStore.on('MutualFollow', (payload: any) => {
			this.addNotification({
				id: `mutual_${Date.now()}`,
				type: 'mutual',
				title: '新好友',
				content: `你和 ${payload.user.nickname} 成为了好友`,
				timestamp: payload.mutual_at,
				read: false
			});
		});

		// 动态被点赞
		websocketStore.on('MomentLiked', (payload: any) => {
			this.addNotification({
				id: `like_${payload.moment_id}_${Date.now()}`,
				type: 'like',
				title: '动态被点赞',
				content: `${payload.user.nickname} 赞了你的动态`,
				timestamp: payload.liked_at,
				read: false
			});
		});

		// 动态被评论
		websocketStore.on('MomentCommented', (payload: any) => {
			this.addNotification({
				id: `comment_${payload.comment_id}`,
				type: 'comment',
				title: '动态被评论',
				content: `${payload.user.nickname}: ${payload.content}`,
				timestamp: payload.commented_at,
				read: false
			});
		});

		// 评论被回复
		websocketStore.on('CommentReply', (payload: any) => {
			this.addNotification({
				id: `reply_${payload.reply_id}`,
				type: 'reply',
				title: '评论被回复',
				content: `${payload.user.nickname} 回复了你`,
				timestamp: payload.replied_at,
				read: false
			});
		});

		// 群组邀请
		websocketStore.on('GroupInviteReceived', (payload: any) => {
			this.addNotification({
				id: `invite_${payload.invite_id}`,
				type: 'group_invite',
				title: '群组邀请',
				content: `${payload.inviter.nickname} 邀请你加入 ${payload.group_name}`,
				timestamp: payload.created_at,
				read: false
			});
		});

		// 入群申请
		websocketStore.on('GroupApplicationReceived', (payload: any) => {
			this.addNotification({
				id: `application_${payload.application_id}`,
				type: 'group_application',
				title: '入群申请',
				content: `${payload.applicant.nickname} 申请加入群组`,
				timestamp: payload.created_at,
				read: false
			});
		});

		// 系统公告
		websocketStore.on('SystemAnnouncement', (payload: any) => {
			this.addNotification({
				id: `announcement_${payload.announcement_id}`,
				type: 'system',
				title: payload.title,
				content: payload.content,
				timestamp: payload.created_at,
				read: false
			});
		});
	}

	/**
	 * 添加通知
	 */
	private addNotification(notification: NotificationItem) {
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
}

// ============ 导出单例 ============

export const notificationStore = new NotificationStore();
