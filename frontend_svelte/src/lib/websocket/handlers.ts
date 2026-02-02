/**
 * WebSocket 事件处理器
 * 统一处理 WebSocket 推送的事件，更新应用状态
 */

import { websocketStore } from './store.svelte';
import { toast } from '$lib/stores/toast.svelte';
import { browser } from '$app/environment';
import { goto } from '$app/navigation';
import { authStore } from '$lib/stores/auth.svelte';
import { presenceStore } from '$lib/stores/presence.svelte';
import type {
	PrivateMessageNewPayload,
	GroupMessageNewPayload,
	NewFollowerPayload,
	MomentLikedPayload,
	MomentCommentedPayload,
	SystemAnnouncementPayload,
	SubscriptionSourceId,
	SubscriptionOperation,
	SubscribeAckPayload,
	UserPresenceUpdatePayload,
	UserOnlinePayload,
	UserOfflinePayload,
	GroupMemberLeftPayload,
	GroupMemberRemovedPayload,
	GroupMemberRoleChangedPayload,
	GroupMemberMutedPayload,
	GroupMemberUnmutedPayload,
	GroupAnnouncementUpdatedPayload,
	GroupUpdatedPayload,
	GroupDeletedPayload
} from './types';

// ============ 事件处理器类型 ============

export type EventHandler<T = unknown> = (payload: T) => void | Promise<void>;

// ============ 订阅类型 ============

export type SubscriptionType = 'user' | 'group' | 'moment';

// ============ 订阅管理器 ============

class SubscriptionManager {
	// 当前订阅（按类型分组）
	private subscriptions = new Map<SubscriptionType, Set<SubscriptionSourceId>>();

	// 待确认的订阅操作（包含定时器引用）
	private pendingAcks = new Map<
		string,
		{
			operation: SubscriptionOperation;
			type: SubscriptionType;
			ids: SubscriptionSourceId[];
			resolve: (success: boolean) => void;
			timer: ReturnType<typeof setTimeout>;
		}
	>();

	constructor() {
		// 初始化订阅集合
		this.subscriptions.set('user', new Set());
		this.subscriptions.set('group', new Set());
		this.subscriptions.set('moment', new Set());

		// 监听 SubscribeAck（现在有类型提示了！）
		websocketStore.on('SubscribeAck', (payload) => {
			this.handleSubscribeAck(payload);
		});

		// 监听连接建立，恢复订阅
		websocketStore.on('ConnectionEstablished', () => {
			this.restoreSubscriptions();
		});
	}

	/**
	 * 订阅（增量）
	 */
	async subscribe(type: SubscriptionType, ids: SubscriptionSourceId[]): Promise<boolean> {
		if (ids.length === 0) return true;

		const clientMsgId = this.generateClientMsgId();

		websocketStore.send({
			type: 'Subscribe',
			payload: {
				client_msg_id: clientMsgId,
				type,
				ids
			}
		});

		// 等待 SubscribeAck
		return new Promise((resolve) => {
			// 创建超时定时器
			const timer = setTimeout(() => {
				if (this.pendingAcks.has(clientMsgId)) {
					console.warn('[WebSocket] 订阅超时:', type, ids);
					this.pendingAcks.delete(clientMsgId);
					resolve(false);
				}
			}, 10000); // 10秒超时
			
			this.pendingAcks.set(clientMsgId, {
				operation: 'subscribe',
				type,
				ids,
				resolve,
				timer
			});
		});
	}

	/**
	 * 取消订阅
	 */
	async unsubscribe(type: SubscriptionType, ids: SubscriptionSourceId[]): Promise<boolean> {
		if (ids.length === 0) return true;

		const clientMsgId = this.generateClientMsgId();

		websocketStore.send({
			type: 'Unsubscribe',
			payload: {
				client_msg_id: clientMsgId,
				type,
				ids
			}
		});

		return new Promise((resolve) => {
			// 创建超时定时器
			const timer = setTimeout(() => {
				if (this.pendingAcks.has(clientMsgId)) {
					console.warn('[WebSocket] 取消订阅超时:', type, ids);
					this.pendingAcks.delete(clientMsgId);
					resolve(false);
				}
			}, 10000);
			
			this.pendingAcks.set(clientMsgId, {
				operation: 'unsubscribe',
				type,
				ids,
				resolve,
				timer
			});
		});
	}

	/**
	 * 设置订阅（替换式）
	 */
	async setSubscriptions(type: SubscriptionType, ids: SubscriptionSourceId[]): Promise<boolean> {
		const clientMsgId = this.generateClientMsgId();

		websocketStore.send({
			type: 'SetSubscriptions',
			payload: {
				client_msg_id: clientMsgId,
				type,
				ids
			}
		});

		return new Promise((resolve) => {
			this.pendingAcks.set(clientMsgId, {
				operation: 'set',
				type,
				ids,
				resolve
			});

			setTimeout(() => {
				if (this.pendingAcks.has(clientMsgId)) {
					console.warn('[WebSocket] 设置订阅超时:', type, ids);
					this.pendingAcks.delete(clientMsgId);
					resolve(false);
				}
			}, 10000);
		});
	}

	/**
	 * 获取当前订阅
	 */
	getSubscriptions(type: SubscriptionType): SubscriptionSourceId[] {
		const subscriptions = this.subscriptions.get(type);
		return subscriptions ? Array.from(subscriptions) : [];
	}

	/**
	 * 检查是否已订阅
	 */
	isSubscribed(type: SubscriptionType, id: SubscriptionSourceId): boolean {
		return this.subscriptions.get(type)?.has(id) ?? false;
	}

	/**
	 * 处理订阅确认（现在有类型提示了！）
	 */
	private handleSubscribeAck(payload: SubscribeAckPayload) {
		const { client_msg_id, operation, type, ids, code, failed_ids } = payload;

		const pending = this.pendingAcks.get(client_msg_id);
		if (!pending) {
			console.warn('[WebSocket] 收到未知的 SubscribeAck:', client_msg_id);
			return;
		}

		// 清理定时器并删除待确认记录
		clearTimeout(pending.timer);
		this.pendingAcks.delete(client_msg_id);

		const success = code === 0;

		if (success) {
			// 更新本地订阅状态
			const subscriptionSet = this.subscriptions.get(type as SubscriptionType);
			if (subscriptionSet) {
				switch (operation) {
					case 'subscribe':
						ids.forEach((id: SubscriptionSourceId) => subscriptionSet.add(id));
						break;
					case 'unsubscribe':
						ids.forEach((id: SubscriptionSourceId) => subscriptionSet.delete(id));
						break;
					case 'set':
						subscriptionSet.clear();
						ids.forEach((id: SubscriptionSourceId) => subscriptionSet.add(id));
						break;
				}
			}

			console.log(`[WebSocket] 订阅${operation}成功:`, type, ids.length, '个');
		} else {
			console.error('[WebSocket] 订阅操作失败:', payload.message, failed_ids);
		}

		pending.resolve(success);
	}

	/**
	 * 恢复订阅（重连后）
	 */
	private async restoreSubscriptions() {
		console.log('[WebSocket] 恢复订阅...');

		for (const [type, ids] of this.subscriptions.entries()) {
			if (ids.size > 0) {
				await this.setSubscriptions(type, Array.from(ids));
			}
		}
	}

	/**
	 * 清理所有待确认的订阅操作
	 */
	cleanup() {
		console.log('[SubscriptionManager] 清理所有待确认的订阅操作');
		
		// 清理所有待确认的定时器
		for (const [clientMsgId, pending] of this.pendingAcks) {
			clearTimeout(pending.timer);
			// 拒绝所有待确认的 Promise
			pending.resolve(false);
		}
		
		this.pendingAcks.clear();
		console.log('[SubscriptionManager] 清理完成');
	}

	/**
	 * 生成客户端消息 ID
	 */
	private generateClientMsgId(): string {
		return `${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
	}
}

// ============ 全局事件处理器 ============

class WebSocketEventHandlers {
	// 私聊消息处理器
	private privateMessageHandlers = new Set<EventHandler<PrivateMessageNewPayload>>();
	private privateMessageEditedHandlers = new Set<EventHandler>();
	private privateMessageRecalledHandlers = new Set<EventHandler>();

	// 群聊消息处理器
	private groupMessageHandlers = new Set<EventHandler<GroupMessageNewPayload>>();
	private groupMessageEditedHandlers = new Set<EventHandler>();
	private groupMessageRecalledHandlers = new Set<EventHandler>();

	// 通知处理器
	private notificationHandlers = new Set<EventHandler>();

	// 在线状态处理器
	private userOnlineHandlers = new Set<EventHandler>();
	private userOfflineHandlers = new Set<EventHandler>();

	constructor() {
		this.registerDefaultHandlers();
	}

	/**
	 * 注册默认处理器
	 */
	private registerDefaultHandlers() {
		// 私聊消息（类型自动推断！）
		websocketStore.on('PrivateMessageNew', (payload) => {
			console.log('[WS] 收到私聊消息:', payload);
			this.privateMessageHandlers.forEach((handler) => handler(payload));
		});

		websocketStore.on('PrivateMessageEdited', (payload) => {
			console.log('[WS] 私聊消息已编辑:', payload);
			this.privateMessageEditedHandlers.forEach((handler) => handler(payload));
		});

		websocketStore.on('PrivateMessageRecalled', (payload) => {
			console.log('[WS] 私聊消息已撤回:', payload);
			this.privateMessageRecalledHandlers.forEach((handler) => handler(payload));
		});

		// 群聊消息（类型自动推断！）
		websocketStore.on('GroupMessageNew', (payload) => {
			console.log('[WS] 收到群聊消息:', payload);
			this.groupMessageHandlers.forEach((handler) => handler(payload));
		});

		websocketStore.on('GroupMessageEdited', (payload) => {
			console.log('[WS] 群聊消息已编辑:', payload);
			this.groupMessageEditedHandlers.forEach((handler) => handler(payload));
		});

		websocketStore.on('GroupMessageRecalled', (payload) => {
			console.log('[WS] 群聊消息已撤回:', payload);
			this.groupMessageRecalledHandlers.forEach((handler) => handler(payload));
		});

		// 关注关系（类型自动推断！）
		websocketStore.on('NewFollower', (payload) => {
			console.log('[WS] 新增关注者:', payload);
			toast.info(`${payload.user.nickname} 关注了你`);
			this.notificationHandlers.forEach((handler) => handler(payload));
		});

		websocketStore.on('MutualFollow', (payload) => {
			console.log('[WS] 互相关注:', payload);
			toast.success(`你和 ${payload.user.nickname} 成为了好友`);
			this.notificationHandlers.forEach((handler) => handler(payload));
		});

		// 生活瞬间互动（类型自动推断！）
		websocketStore.on('MomentLiked', (payload) => {
			console.log('[WS] 动态被点赞:', payload);
			toast.info(`${payload.user.nickname} 赞了你的动态`);
			this.notificationHandlers.forEach((handler) => handler(payload));
		});

		websocketStore.on('MomentCommented', (payload) => {
			console.log('[WS] 动态被评论:', payload);
			toast.info(`${payload.user.nickname} 评论了你的动态`);
			this.notificationHandlers.forEach((handler) => handler(payload));
		});

		websocketStore.on('CommentReply', (payload) => {
			console.log('[WS] 评论被回复:', payload);
			toast.info(`${payload.user.nickname} 回复了你的评论`);
			this.notificationHandlers.forEach((handler) => handler(payload));
		});

		// 群组管理（类型自动推断！）
		websocketStore.on('GroupInviteReceived', (payload) => {
			console.log('[WS] 收到群组邀请:', payload);
			toast.info(`${payload.inviter.nickname} 邀请你加入群组`);
			this.notificationHandlers.forEach((handler) => handler(payload));
		});

		websocketStore.on('GroupApplicationReceived', (payload) => {
			console.log('[WS] 收到入群申请:', payload);
			toast.info(`${payload.applicant.nickname} 申请加入群组`);
			this.notificationHandlers.forEach((handler) => handler(payload));
		});

		// 群组成员离开（GroupMemberLeft）
		websocketStore.on('GroupMemberLeft', (payload: GroupMemberLeftPayload) => {
			console.log('[WS] 群组成员离开:', payload);
			
			// 显示提示
			toast.info(`${payload.user.nickname} 离开了群组`);
			
			// 分发自定义事件，让页面组件更新成员列表
			if (browser) {
				window.dispatchEvent(
					new CustomEvent('group-member-left', {
						detail: {
							group_chat_id: payload.group_chat_id,
							user_id: payload.user.user_id,
							member_count: payload.member_count
						}
					})
				);
			}
		});

		// 群组成员被移除（GroupMemberRemoved）
		websocketStore.on('GroupMemberRemoved', (payload: GroupMemberRemovedPayload) => {
			console.log('[WS] 群组成员被移除:', payload);
			
			const isCurrentUser = payload.user.user_id === authStore.currentUserId;
			
			// 显示提示
			if (isCurrentUser) {
				toast.warning('你已被移出群组');
			} else {
				const reasonText = payload.reason === 'banned' ? '被封禁' : '被移除';
				toast.info(`${payload.user.nickname} ${reasonText}`);
			}
			
			// 分发自定义事件
			if (browser) {
				window.dispatchEvent(
					new CustomEvent('group-member-removed', {
						detail: {
							group_chat_id: payload.group_chat_id,
							user_id: payload.user.user_id,
							removed_by: payload.removed_by,
							reason: payload.reason,
							member_count: payload.member_count,
							is_current_user: isCurrentUser
						}
					})
				);
				
				// 如果被移除的是当前用户，检查是否需要跳转
				if (isCurrentUser) {
					const currentPath = window.location.pathname;
					// 如果在群组相关页面，跳转到群组列表
					if (currentPath.includes(`/groups/${payload.group_chat_id}`) || 
					    currentPath.includes(`/chat/groups/${payload.group_chat_id}`)) {
						goto('/groups');
					}
				}
			}
		});

		// 群组成员角色变更（GroupMemberRoleChanged）
		websocketStore.on('GroupMemberRoleChanged', (payload: GroupMemberRoleChangedPayload) => {
			console.log('[WS] 群组成员角色变更:', payload);
			
			const roleLabels: Record<string, string> = {
				owner: '群主',
				admin: '管理员',
				member: '成员'
			};
			
			const isCurrentUser = payload.user.user_id === authStore.currentUserId;
			const oldRoleLabel = roleLabels[payload.old_role] || payload.old_role;
			const newRoleLabel = roleLabels[payload.new_role] || payload.new_role;
			
			// 显示提示
			if (isCurrentUser) {
				toast.success(`你的角色已变更为 ${newRoleLabel}`);
			} else {
				toast.info(`${payload.user.nickname} 的角色已从 ${oldRoleLabel} 变更为 ${newRoleLabel}`);
			}
			
			// 分发自定义事件
			if (browser) {
				window.dispatchEvent(
					new CustomEvent('group-member-role-changed', {
						detail: {
							group_chat_id: payload.group_chat_id,
							user_id: payload.user.user_id,
							old_role: payload.old_role,
							new_role: payload.new_role,
							changed_by: payload.changed_by,
							is_current_user: isCurrentUser
						}
					})
				);
			}
		});

		// 群组成员被禁言（GroupMemberMuted）
		websocketStore.on('GroupMemberMuted', (payload: GroupMemberMutedPayload) => {
			console.log('[WS] 群组成员被禁言:', payload);
			
			const isCurrentUser = payload.user.user_id === authStore.currentUserId;
			const mutedUntil = payload.muted_until 
				? new Date(payload.muted_until).toLocaleString('zh-CN')
				: '永久';
			
			// 显示提示
			if (isCurrentUser) {
				toast.warning(`你已被禁言，直到 ${mutedUntil}`);
			} else {
				toast.info(`${payload.user.nickname} 已被禁言，直到 ${mutedUntil}`);
			}
			
			// 分发自定义事件
			if (browser) {
				window.dispatchEvent(
					new CustomEvent('group-member-muted', {
						detail: {
							group_chat_id: payload.group_chat_id,
							user_id: payload.user.user_id,
							muted_by: payload.muted_by,
							muted_until: payload.muted_until,
							reason: payload.reason,
							is_current_user: isCurrentUser
						}
					})
				);
			}
		});

		// 群组成员解除禁言（GroupMemberUnmuted）
		websocketStore.on('GroupMemberUnmuted', (payload: GroupMemberUnmutedPayload) => {
			console.log('[WS] 群组成员解除禁言:', payload);
			
			const isCurrentUser = payload.user.user_id === authStore.currentUserId;
			
			// 显示提示
			if (isCurrentUser) {
				toast.success('你的禁言已解除');
			} else {
				toast.info(`${payload.user.nickname} 的禁言已解除`);
			}
			
			// 分发自定义事件
			if (browser) {
				window.dispatchEvent(
					new CustomEvent('group-member-unmuted', {
						detail: {
							group_chat_id: payload.group_chat_id,
							user_id: payload.user.user_id,
							unmuted_by: payload.unmuted_by,
							is_current_user: isCurrentUser
						}
					})
				);
			}
		});

		// 群公告更新（GroupAnnouncementUpdated）
		websocketStore.on('GroupAnnouncementUpdated', (payload: GroupAnnouncementUpdatedPayload) => {
			console.log('[WS] 群公告更新:', payload);
			
			// 显示提示
			toast.info('群公告已更新');
			
			// 分发自定义事件
			if (browser) {
				window.dispatchEvent(
					new CustomEvent('group-announcement-updated', {
						detail: {
							group_chat_id: payload.group_chat_id,
							announcement_id: payload.announcement_id,
							content: payload.content,
							updated_by: payload.updated_by
						}
					})
				);
			}
		});

		// 群组信息更新（GroupUpdated）
		websocketStore.on('GroupUpdated', (payload: GroupUpdatedPayload) => {
			console.log('[WS] 群组信息更新:', payload);
			
			// 显示提示
			const updatedFields = Object.keys(payload.updated_fields);
			if (updatedFields.length > 0) {
				const fieldLabels: Record<string, string> = {
					name: '名称',
					avatar_url: '头像',
					description: '描述'
				};
				const fieldsText = updatedFields.map(f => fieldLabels[f] || f).join('、');
				toast.success(`群组${fieldsText}已更新`);
			}
			
			// 分发自定义事件
			if (browser) {
				window.dispatchEvent(
					new CustomEvent('group-updated', {
						detail: {
							group_chat_id: payload.group_chat_id,
							updated_fields: payload.updated_fields,
							updated_by: payload.updated_by
						}
					})
				);
			}
		});

		// 群组解散（GroupDeleted）
		websocketStore.on('GroupDeleted', (payload: GroupDeletedPayload) => {
			console.log('[WS] 群组解散:', payload);
			
			// 显示提示
			toast.warning('群组已解散');
			
			// 分发自定义事件
			if (browser) {
				window.dispatchEvent(
					new CustomEvent('group-deleted', {
						detail: {
							group_chat_id: payload.group_chat_id,
							deleted_by: payload.deleted_by,
							reason: payload.reason
						}
					})
				);
				
				// 检查当前路由，如果在群组相关页面，跳转到群组列表
				const currentPath = window.location.pathname;
				if (currentPath.includes(`/groups/${payload.group_chat_id}`) || 
				    currentPath.includes(`/chat/groups/${payload.group_chat_id}`)) {
					goto('/groups');
				}
			}
		});

		// 系统公告（类型自动推断！）
		websocketStore.on('SystemAnnouncement', (payload) => {
			console.log('[WS] 系统公告:', payload);
			switch (payload.severity) {
				case 'critical':
					toast.error(payload.title, payload.content);
					break;
				case 'warning':
					toast.warning(payload.title);
					break;
				default:
					toast.info(payload.title);
			}
			this.notificationHandlers.forEach((handler) => handler(payload));
		});

		// 在线状态
		websocketStore.on('UserOnline', (payload: UserOnlinePayload) => {
			console.log('[WS] 用户上线:', payload);
			
			// 更新 presence store
			presenceStore.setOnline(
				payload.user_id,
				payload.status,
				payload.status_text,
				payload.online_at
			);
			
			// 调用自定义处理器
			this.userOnlineHandlers.forEach((handler) => handler(payload));
		});

		websocketStore.on('UserOffline', (payload: UserOfflinePayload) => {
			console.log('[WS] 用户离线:', payload);
			
			// 更新 presence store
			presenceStore.setOffline(
				payload.user_id,
				payload.offline_at,
				payload.last_seen
			);
			
			// 调用自定义处理器
			this.userOfflineHandlers.forEach((handler) => handler(payload));
		});

		// 用户状态更新（UserPresenceUpdate）
		websocketStore.on('UserPresenceUpdate', (payload: UserPresenceUpdatePayload) => {
			console.log('[WS] 用户状态更新:', payload);
			
			// 更新 presence store
			presenceStore.updatePresence(
				payload.user_id,
				payload.status,
				payload.status_text,
				payload.updated_at
			);
		});
	}

	// ============ 添加自定义处理器 ============

	/**
	 * 添加私聊消息处理器
	 */
	onPrivateMessage(handler: EventHandler<PrivateMessageNewPayload>) {
		this.privateMessageHandlers.add(handler);
		return () => this.privateMessageHandlers.delete(handler);
	}

	/**
	 * 添加群聊消息处理器
	 */
	onGroupMessage(handler: EventHandler<GroupMessageNewPayload>) {
		this.groupMessageHandlers.add(handler);
		return () => this.groupMessageHandlers.delete(handler);
	}

	/**
	 * 添加通知处理器
	 */
	onNotification(handler: EventHandler) {
		this.notificationHandlers.add(handler);
		return () => this.notificationHandlers.delete(handler);
	}

	/**
	 * 添加用户在线状态处理器
	 */
	onUserOnline(handler: EventHandler) {
		this.userOnlineHandlers.add(handler);
		return () => this.userOnlineHandlers.delete(handler);
	}

	onUserOffline(handler: EventHandler) {
		this.userOfflineHandlers.add(handler);
		return () => this.userOfflineHandlers.delete(handler);
	}

	/**
	 * 生成客户端消息 ID
	 */
	private generateClientMsgId(): string {
		return `${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
	}
}

// ============ 导出单例 ============

// 必须先创建实例，因为 WebSocketEventHandlers 依赖 websocketStore
export const subscriptionManager = new SubscriptionManager();
export const eventHandlers = new WebSocketEventHandlers();
