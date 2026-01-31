/**
 * WebSocket 事件处理器
 * 统一处理 WebSocket 推送的事件，更新应用状态
 */

import { websocketStore } from './store.svelte';
import { toast } from '$lib/stores/toast.svelte';
import type {
	PrivateMessageNewPayload,
	GroupMessageNewPayload,
	NewFollowerPayload,
	MomentLikedPayload,
	MomentCommentedPayload,
	SystemAnnouncementPayload,
	SubscriptionSourceId,
	SubscriptionOperation
} from './types';

// ============ 事件处理器类型 ============

export type EventHandler<T = unknown> = (payload: T) => void | Promise<void>;

// ============ 订阅类型 ============

export type SubscriptionType = 'user' | 'group' | 'moment';

// ============ 订阅管理器 ============

class SubscriptionManager {
	// 当前订阅（按类型分组）
	private subscriptions = new Map<SubscriptionType, Set<SubscriptionSourceId>>();

	// 待确认的订阅操作
	private pendingAcks = new Map<
		string,
		{
			operation: SubscriptionOperation;
			type: SubscriptionType;
			ids: SubscriptionSourceId[];
			resolve: (success: boolean) => void;
		}
	>();

	constructor() {
		// 初始化订阅集合
		this.subscriptions.set('user', new Set());
		this.subscriptions.set('group', new Set());
		this.subscriptions.set('moment', new Set());

		// 监听 SubscribeAck
		websocketStore.on('SubscribeAck', (payload: any) => {
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
			this.pendingAcks.set(clientMsgId, {
				operation: 'subscribe',
				type,
				ids,
				resolve
			});

			// 超时处理
			setTimeout(() => {
				if (this.pendingAcks.has(clientMsgId)) {
					console.warn('[WebSocket] 订阅超时:', type, ids);
					this.pendingAcks.delete(clientMsgId);
					resolve(false);
				}
			}, 10000); // 10秒超时
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
			this.pendingAcks.set(clientMsgId, {
				operation: 'unsubscribe',
				type,
				ids,
				resolve
			});

			setTimeout(() => {
				if (this.pendingAcks.has(clientMsgId)) {
					console.warn('[WebSocket] 取消订阅超时:', type, ids);
					this.pendingAcks.delete(clientMsgId);
					resolve(false);
				}
			}, 10000);
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
	 * 处理订阅确认
	 */
	private handleSubscribeAck(payload: any) {
		const { client_msg_id, operation, type, ids, code, failed_ids } = payload;

		const pending = this.pendingAcks.get(client_msg_id);
		if (!pending) {
			console.warn('[WebSocket] 收到未知的 SubscribeAck:', client_msg_id);
			return;
		}

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
		websocketStore.on<PrivateMessageNewPayload>('PrivateMessageNew', (payload) => {
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

		// 群聊消息
		websocketStore.on<GroupMessageNewPayload>('GroupMessageNew', (payload) => {
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

		// 关注关系
		websocketStore.on<NewFollowerPayload>('NewFollower', (payload) => {
			console.log('[WS] 新增关注者:', payload);
			toast.info(`${payload.user.nickname} 关注了你`);
			this.notificationHandlers.forEach((handler) => handler(payload));
		});

		websocketStore.on('MutualFollow', (payload: any) => {
			console.log('[WS] 互相关注:', payload);
			toast.success(`你和 ${payload.user.nickname} 成为了好友`);
			this.notificationHandlers.forEach((handler) => handler(payload));
		});

		// 生活瞬间互动
		websocketStore.on<MomentLikedPayload>('MomentLiked', (payload) => {
			console.log('[WS] 动态被点赞:', payload);
			toast.info(`${payload.user.nickname} 赞了你的动态`);
			this.notificationHandlers.forEach((handler) => handler(payload));
		});

		websocketStore.on<MomentCommentedPayload>('MomentCommented', (payload) => {
			console.log('[WS] 动态被评论:', payload);
			toast.info(`${payload.user.nickname} 评论了你的动态`);
			this.notificationHandlers.forEach((handler) => handler(payload));
		});

		websocketStore.on('CommentReply', (payload: any) => {
			console.log('[WS] 评论被回复:', payload);
			toast.info(`${payload.user.nickname} 回复了你的评论`);
			this.notificationHandlers.forEach((handler) => handler(payload));
		});

		// 群组管理
		websocketStore.on('GroupInviteReceived', (payload: any) => {
			console.log('[WS] 收到群组邀请:', payload);
			toast.info(`${payload.inviter.nickname} 邀请你加入群组`);
			this.notificationHandlers.forEach((handler) => handler(payload));
		});

		websocketStore.on('GroupApplicationReceived', (payload: any) => {
			console.log('[WS] 收到入群申请:', payload);
			toast.info(`${payload.applicant.nickname} 申请加入群组`);
			this.notificationHandlers.forEach((handler) => handler(payload));
		});

		// 系统公告
		websocketStore.on<SystemAnnouncementPayload>('SystemAnnouncement', (payload) => {
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
		websocketStore.on('UserOnline', (payload) => {
			console.log('[WS] 用户上线:', payload);
			this.userOnlineHandlers.forEach((handler) => handler(payload));
		});

		websocketStore.on('UserOffline', (payload) => {
			console.log('[WS] 用户离线:', payload);
			this.userOfflineHandlers.forEach((handler) => handler(payload));
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
