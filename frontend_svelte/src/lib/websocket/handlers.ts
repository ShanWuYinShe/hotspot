/**
 * WebSocket 事件处理器
 * 统一处理 WebSocket 推送的事件，更新应用状态
 * 严格遵循 @spec/api_spec_websocket.mdc
 */

import { websocketStore } from './store.svelte';
import { toast } from '$lib/stores/toast.svelte';
import { browser, dev } from '$app/environment';
import { goto } from '$app/navigation';

// Dev-only 日志工具
const log = dev ? console.log.bind(console) : () => {};
const warn = dev ? console.warn.bind(console) : () => {};
const logError = dev ? console.error.bind(console) : () => {};
import { authStore } from '$lib/stores/auth.svelte';
import { presenceStore } from '$lib/stores/presence.svelte';
import type {
	PrivateMessageNewPayload,
	GroupMessageNewPayload,
	SubscriptionOperation,
	SubscribeAckPayload,
	UserOnlinePayload,
	UserOfflinePayload,
	GroupMemberLeftPayload,
	GroupMemberRemovedPayload,
	GroupMemberMutedPayload,
	GroupMemberUnmutedPayload,
	GroupUpdatedPayload,
	GroupDismissedPayload,
	AccountBannedPayload
} from './types';

// ============ 事件处理器类型 ============

export type EventHandler<T = unknown> = (payload: T) => void | Promise<void>;

// ============ 订阅类型 ============

export type SubscriptionType = 'user' | 'group';

// ============ 订阅管理器 ============

class SubscriptionManager {
	// 当前订阅（按类型分组）
	private subscriptions = new Map<SubscriptionType, Set<string>>();

	// 待确认的订阅操作（包含定时器引用）
	private pendingAcks = new Map<
		string,
		{
			operation: SubscriptionOperation;
			type: SubscriptionType;
			ids: string[];
			resolve: (success: boolean) => void;
			timer: ReturnType<typeof setTimeout>;
		}
	>();

	constructor() {
		// 初始化订阅集合
		this.subscriptions.set('user', new Set());
		this.subscriptions.set('group', new Set());

		// 监听 SubscribeAck
		websocketStore.on('SubscribeAck', (payload) => {
			this.handleSubscribeAck(payload);
		});

		// 监听连接建立，恢复订阅
		websocketStore.on('ConnectionEstablished', () => {
			this.restoreSubscriptions();
		});
	}

	/**
	 * 订阅（增量添加）
	 */
	async subscribe(type: SubscriptionType, ids: string[]): Promise<boolean> {
		if (ids.length === 0) return true;

		const clientMsgId = this.generateClientMsgId();

		websocketStore.send({
			type: 'SetSubscriptions',
			payload: {
				client_msg_id: clientMsgId,
				operation: 'subscribe',
				type,
				ids
			}
		});

		// 等待 SubscribeAck
		return new Promise((resolve) => {
			// 创建超时定时器
			const timer = setTimeout(() => {
				if (this.pendingAcks.has(clientMsgId)) {
					warn('[WebSocket] 订阅超时:', type, ids);
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
	 * 取消订阅（增量移除）
	 */
	async unsubscribe(type: SubscriptionType, ids: string[]): Promise<boolean> {
		if (ids.length === 0) return true;

		const clientMsgId = this.generateClientMsgId();

		websocketStore.send({
			type: 'SetSubscriptions',
			payload: {
				client_msg_id: clientMsgId,
				operation: 'unsubscribe',
				type,
				ids
			}
		});

		return new Promise((resolve) => {
			// 创建超时定时器
			const timer = setTimeout(() => {
				if (this.pendingAcks.has(clientMsgId)) {
					warn('[WebSocket] 取消订阅超时:', type, ids);
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
	 * 设置订阅（全量覆盖，重连时使用）
	 */
	async setSubscriptions(type: SubscriptionType, ids: string[]): Promise<boolean> {
		const clientMsgId = this.generateClientMsgId();

		websocketStore.send({
			type: 'SetSubscriptions',
			payload: {
				client_msg_id: clientMsgId,
				operation: 'set',
				type,
				ids
			}
		});

		return new Promise((resolve) => {
			const timer = setTimeout(() => {
				if (this.pendingAcks.has(clientMsgId)) {
					warn('[WebSocket] 设置订阅超时:', type, ids);
					this.pendingAcks.delete(clientMsgId);
					resolve(false);
				}
			}, 10000);

			this.pendingAcks.set(clientMsgId, {
				operation: 'set',
				type,
				ids,
				resolve,
				timer
			});
		});
	}

	/**
	 * 获取当前订阅
	 */
	getSubscriptions(type: SubscriptionType): string[] {
		const subscriptions = this.subscriptions.get(type);
		return subscriptions ? Array.from(subscriptions) : [];
	}

	/**
	 * 检查是否已订阅
	 */
	isSubscribed(type: SubscriptionType, id: string): boolean {
		return this.subscriptions.get(type)?.has(id) ?? false;
	}

	/**
	 * 处理订阅确认
	 */
	private handleSubscribeAck(payload: SubscribeAckPayload) {
		const { client_msg_id, operation, type, ids, code, failed_ids } = payload;

		const pending = this.pendingAcks.get(client_msg_id);
		if (!pending) {
			warn('[WebSocket] 收到未知的 SubscribeAck:', client_msg_id);
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
						ids.forEach((id: string) => subscriptionSet.add(id));
						break;
					case 'unsubscribe':
						ids.forEach((id: string) => subscriptionSet.delete(id));
						break;
					case 'set':
						subscriptionSet.clear();
						ids.forEach((id: string) => subscriptionSet.add(id));
						break;
				}
			}

			log(`[WebSocket] 订阅${operation}成功:`, type, ids.length, '个');
		} else {
			logError('[WebSocket] 订阅操作失败:', payload.message, failed_ids);
		}

		pending.resolve(success);
	}

	/**
	 * 恢复订阅（重连后，使用 set 操作全量覆盖）
	 */
	private async restoreSubscriptions() {
		log('[WebSocket] 恢复订阅...');

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
		log('[SubscriptionManager] 清理所有待确认的订阅操作');

		// 清理所有待确认的定时器
		for (const [_clientMsgId, pending] of this.pendingAcks) {
			clearTimeout(pending.timer);
			// 拒绝所有待确认的 Promise
			pending.resolve(false);
		}

		this.pendingAcks.clear();
		log('[SubscriptionManager] 清理完成');
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
		// 私聊消息
		websocketStore.on('PrivateMessageNew', (payload) => {
			log('[WS] 收到私聊消息:', payload);
			this.privateMessageHandlers.forEach((handler) => handler(payload));
		});

		websocketStore.on('PrivateMessageEdited', (payload) => {
			log('[WS] 私聊消息已编辑:', payload);
			this.privateMessageEditedHandlers.forEach((handler) => handler(payload));
		});

		websocketStore.on('PrivateMessageRecalled', (payload) => {
			log('[WS] 私聊消息已撤回:', payload);
			this.privateMessageRecalledHandlers.forEach((handler) => handler(payload));
		});

		// 群聊消息
		websocketStore.on('GroupMessageNew', (payload) => {
			log('[WS] 收到群聊消息:', payload);
			this.groupMessageHandlers.forEach((handler) => handler(payload));
		});

		websocketStore.on('GroupMessageEdited', (payload) => {
			log('[WS] 群聊消息已编辑:', payload);
			this.groupMessageEditedHandlers.forEach((handler) => handler(payload));
		});

		websocketStore.on('GroupMessageRecalled', (payload) => {
			log('[WS] 群聊消息已撤回:', payload);
			this.groupMessageRecalledHandlers.forEach((handler) => handler(payload));
		});

		// 关注关系
		websocketStore.on('NewFollower', (payload) => {
			log('[WS] 新增关注者:', payload);
			toast.info(`${payload.user.nickname} 关注了你`);
			this.notificationHandlers.forEach((handler) => handler(payload));
		});

		websocketStore.on('MutualFollow', (payload) => {
			log('[WS] 互相关注:', payload);
			toast.success(`你和 ${payload.user.nickname} 成为了好友`);
			this.notificationHandlers.forEach((handler) => handler(payload));
		});

		// 生活瞬间互动
		websocketStore.on('MomentLiked', (payload) => {
			log('[WS] 动态被点赞:', payload);
			toast.info(`${payload.user.nickname} 赞了你的动态`);
			this.notificationHandlers.forEach((handler) => handler(payload));
		});

		websocketStore.on('MomentCommented', (payload) => {
			log('[WS] 动态被评论:', payload);
			toast.info(`${payload.user.nickname} 评论了你的动态`);
			this.notificationHandlers.forEach((handler) => handler(payload));
		});

		// 评论被点赞（spec 新增事件）
		websocketStore.on('CommentLiked', (payload) => {
			log('[WS] 评论被点赞:', payload);
			toast.info(`${payload.user.nickname} 赞了你的评论`);
			this.notificationHandlers.forEach((handler) => handler(payload));
		});

		websocketStore.on('CommentReply', (payload) => {
			log('[WS] 评论被回复:', payload);
			toast.info(`${payload.user.nickname} 回复了你的评论`);
			this.notificationHandlers.forEach((handler) => handler(payload));
		});

		// 群组管理
		websocketStore.on('GroupInviteReceived', (payload) => {
			log('[WS] 收到群组邀请:', payload);
			toast.info(`${payload.inviter.nickname} 邀请你加入群组`);
			this.notificationHandlers.forEach((handler) => handler(payload));
		});

		websocketStore.on('GroupApplicationReceived', (payload) => {
			log('[WS] 收到入群申请:', payload);
			toast.info(`${payload.applicant.nickname} 申请加入群组`);
			this.notificationHandlers.forEach((handler) => handler(payload));
		});

		// 群组成员离开（GroupMemberLeft）
		websocketStore.on('GroupMemberLeft', (payload: GroupMemberLeftPayload) => {
			log('[WS] 群组成员离开:', payload);

			// 显示提示
			toast.info(`${payload.user.nickname} 离开了群组`);

			// 分发自定义事件，让页面组件更新成员列表
			if (browser) {
				window.dispatchEvent(
					new CustomEvent('group-member-left', {
						detail: {
							group_id: payload.group_id,
							user_id: payload.user.user_id,
							member_count: payload.member_count
						}
					})
				);
			}
		});

		// 群组成员被移除（GroupMemberRemoved）
		websocketStore.on('GroupMemberRemoved', (payload: GroupMemberRemovedPayload) => {
			log('[WS] 群组成员被移除:', payload);

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
							group_id: payload.group_id,
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
					if (
						currentPath.includes(`/groups/${payload.group_id}`) ||
						currentPath.includes(`/chat/groups/${payload.group_id}`)
					) {
						goto('/groups');
					}
				}
			}
		});

		// 群组成员被禁言（GroupMemberMuted）
		websocketStore.on('GroupMemberMuted', (payload: GroupMemberMutedPayload) => {
			log('[WS] 群组成员被禁言:', payload);

			const isCurrentUser = payload.user.user_id === authStore.currentUserId;
			const mutedUntil = new Date(payload.muted_until).toLocaleString('zh-CN');

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
							group_id: payload.group_id,
							user_id: payload.user.user_id,
							muted_by: payload.muted_by,
							muted_until: payload.muted_until,
							is_current_user: isCurrentUser
						}
					})
				);
			}
		});

		// 群组成员解除禁言（GroupMemberUnmuted）
		websocketStore.on('GroupMemberUnmuted', (payload: GroupMemberUnmutedPayload) => {
			log('[WS] 群组成员解除禁言:', payload);

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
							group_id: payload.group_id,
							user_id: payload.user.user_id,
							unmuted_by: payload.unmuted_by,
							is_current_user: isCurrentUser
						}
					})
				);
			}
		});

		// 群组信息更新（GroupUpdated）
		websocketStore.on('GroupUpdated', (payload: GroupUpdatedPayload) => {
			log('[WS] 群组信息更新:', payload);

			// 显示提示
			const updatedFields = Object.keys(payload.updated_fields);
			if (updatedFields.length > 0) {
				const fieldLabels: Record<string, string> = {
					name: '名称',
					avatar_url: '头像',
					description: '描述',
					tags: '标签',
					join_method: '加入方式',
					is_searchable: '可搜索性',
					max_members: '成员上限',
					has_password: '密码设置',
					location: '位置'
				};
				const fieldsText = updatedFields.map((f) => fieldLabels[f] || f).join('、');
				toast.success(`群组${fieldsText}已更新`);
			}

			// 分发自定义事件
			if (browser) {
				window.dispatchEvent(
					new CustomEvent('group-updated', {
						detail: {
							group_id: payload.group_id,
							updated_fields: payload.updated_fields,
							updated_by: payload.updated_by
						}
					})
				);
			}
		});

		// 群组解散（GroupDismissed，spec 命名）
		websocketStore.on('GroupDismissed', (payload: GroupDismissedPayload) => {
			log('[WS] 群组解散:', payload);

			// 显示提示
			toast.warning('群组已解散');

			// 分发自定义事件
			if (browser) {
				window.dispatchEvent(
					new CustomEvent('group-dismissed', {
						detail: {
							group_id: payload.group_id,
							reason: payload.reason
						}
					})
				);

				// 检查当前路由，如果在群组相关页面，跳转到群组列表
				const currentPath = window.location.pathname;
				if (
					currentPath.includes(`/groups/${payload.group_id}`) ||
					currentPath.includes(`/chat/groups/${payload.group_id}`)
				) {
					goto('/groups');
				}
			}
		});

		// 系统公告
		websocketStore.on('SystemAnnouncement', (payload) => {
			log('[WS] 系统公告:', payload);
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

		// 账号被封禁（spec 新增事件）
		websocketStore.on('AccountBanned', (payload: AccountBannedPayload) => {
			log('[WS] 账号被封禁:', payload);

			const durationText =
				payload.duration_days === 0 ? '永久' : `${payload.duration_days} 天`;
			toast.error(`账号已被封禁（${durationText}）`, payload.reason);

			// 强制登出
			authStore.logout();
			goto('/auth');
		});

		// 在线状态（spec 简化：只有 user_id + 时间戳）
		websocketStore.on('UserOnline', (payload: UserOnlinePayload) => {
			log('[WS] 用户上线:', payload);

			// 更新 presence store（简化为 online 状态）
			presenceStore.setOnline(payload.user_id, 'online', undefined, payload.online_at);

			// 调用自定义处理器
			this.userOnlineHandlers.forEach((handler) => handler(payload));
		});

		websocketStore.on('UserOffline', (payload: UserOfflinePayload) => {
			log('[WS] 用户离线:', payload);

			// 更新 presence store
			presenceStore.setOffline(payload.user_id, payload.offline_at);

			// 调用自定义处理器
			this.userOfflineHandlers.forEach((handler) => handler(payload));
		});

		// 用户资料变更（spec 新增事件）
		websocketStore.on('UserProfileUpdated', (payload) => {
			log('[WS] 用户资料变更:', payload);
			// 可用于更新缓存的用户信息
		});

		// 群用户进入/离开群聊（spec 新增事件）
		websocketStore.on('GroupUserEntered', (payload) => {
			log('[WS] 用户进入群聊:', payload);
		});

		websocketStore.on('GroupUserExited', (payload) => {
			log('[WS] 用户离开群聊:', payload);
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
}

// ============ 导出单例 ============

// 必须先创建实例，因为 WebSocketEventHandlers 依赖 websocketStore
export const subscriptionManager = new SubscriptionManager();
export const eventHandlers = new WebSocketEventHandlers();
