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
	AccountBannedPayload,
	GroupMemberJoinedPayload
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

	/** 增量添加订阅 */
	subscribe(type: SubscriptionType, ids: string[]): Promise<boolean> {
		return ids.length === 0 ? Promise.resolve(true) : this.sendRequest('subscribe', type, ids);
	}

	/** 增量移除订阅 */
	unsubscribe(type: SubscriptionType, ids: string[]): Promise<boolean> {
		return ids.length === 0 ? Promise.resolve(true) : this.sendRequest('unsubscribe', type, ids);
	}

	/** 全量覆盖订阅（重连时使用） */
	setSubscriptions(type: SubscriptionType, ids: string[]): Promise<boolean> {
		return this.sendRequest('set', type, ids);
	}

	/** 统一的订阅请求发送 + 超时等待 */
	private sendRequest(
		operation: SubscriptionOperation,
		type: SubscriptionType,
		ids: string[]
	): Promise<boolean> {
		const clientMsgId = this.generateClientMsgId();

		websocketStore.send({
			type: 'SetSubscriptions',
			payload: { client_msg_id: clientMsgId, operation, type, ids }
		});

		return new Promise((resolve) => {
			const timer = setTimeout(() => {
				if (this.pendingAcks.has(clientMsgId)) {
					warn(`[WebSocket] ${operation}超时:`, type, ids);
					this.pendingAcks.delete(clientMsgId);
					resolve(false);
				}
			}, 10000);

			this.pendingAcks.set(clientMsgId, { operation, type, ids, resolve, timer });
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

		// 计算成功的 ID（排除 failed_ids 中的项）
		const failedIdSet = new Set(failed_ids?.map((f) => f.subscription_id) ?? []);
		const succeededIds = ids.filter((id) => !failedIdSet.has(id));

		// 无论 code 是否为 0，都更新成功项的本地订阅状态
		// spec: failed_ids 仅包含失败项，其余为成功
		const subscriptionSet = this.subscriptions.get(type as SubscriptionType);
		if (subscriptionSet && succeededIds.length > 0) {
			switch (operation) {
				case 'subscribe':
					succeededIds.forEach((id) => subscriptionSet.add(id));
					break;
				case 'unsubscribe':
					succeededIds.forEach((id) => subscriptionSet.delete(id));
					break;
				case 'set':
					subscriptionSet.clear();
					succeededIds.forEach((id) => subscriptionSet.add(id));
					break;
			}
		}

		if (code === 0) {
			log(`[WebSocket] 订阅${operation}成功:`, type, ids.length, '个');
		} else {
			logError('[WebSocket] 订阅操作部分失败:', payload.message, failed_ids);
			if (succeededIds.length > 0) {
				log(`[WebSocket] 部分成功: ${succeededIds.length}/${ids.length} 个`);
			}
		}

		// 全部成功时 resolve(true)，部分失败时 resolve(false)
		pending.resolve(code === 0);
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
	 * 生成客户端消息 ID（spec 建议使用 UUID）
	 */
	private generateClientMsgId(): string {
		return crypto.randomUUID();
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

	// ============ 辅助方法 ============

	/** 注册通知类事件：log + toast + 转发到 notificationHandlers */
	private onNotify<K extends keyof import('./types').WebSocketEventMap>(
		event: K,
		getMessage: (payload: import('./types').WebSocketEventMap[K]) => string,
		toastType: 'info' | 'success' | 'warning' | 'error' = 'info'
	) {
		websocketStore.on(event, (payload) => {
			log(`[WS] ${event}:`, payload);
			toast[toastType](getMessage(payload));
			this.notificationHandlers.forEach((handler) => handler(payload));
		});
	}

	/** 注册消息类事件：log + 转发到自定义 handlers */
	private onMessage<K extends keyof import('./types').WebSocketEventMap>(
		event: K,
		handlers: Set<EventHandler<import('./types').WebSocketEventMap[K]>>
	) {
		websocketStore.on(event, (payload) => {
			log(`[WS] ${event}:`, payload);
			handlers.forEach((handler) => handler(payload));
		});
	}

	/**
	 * 注册默认处理器
	 */
	private registerDefaultHandlers() {
		// 私聊消息
		this.onMessage('PrivateMessageNew', this.privateMessageHandlers);
		this.onMessage('PrivateMessageEdited', this.privateMessageEditedHandlers);
		this.onMessage('PrivateMessageRecalled', this.privateMessageRecalledHandlers);

		// 群聊消息
		this.onMessage('GroupMessageNew', this.groupMessageHandlers);
		this.onMessage('GroupMessageEdited', this.groupMessageEditedHandlers);
		this.onMessage('GroupMessageRecalled', this.groupMessageRecalledHandlers);

		// 关注关系
		this.onNotify('NewFollower', (p) => `${p.user.nickname} 关注了你`);
		this.onNotify('MutualFollow', (p) => `你和 ${p.user.nickname} 成为了好友`, 'success');

		// 生活瞬间互动
		this.onNotify('MomentLiked', (p) => `${p.user.nickname} 赞了你的动态`);
		this.onNotify('MomentCommented', (p) => `${p.user.nickname} 评论了你的动态`);
		this.onNotify('CommentLiked', (p) => `${p.user.nickname} 赞了你的评论`);
		this.onNotify('CommentReply', (p) => `${p.user.nickname} 回复了你的评论`);

		// 群组邀请/申请
		this.onNotify('GroupInviteReceived', (p) => `${p.inviter.nickname} 邀请你加入群组`);
		this.onNotify('GroupApplicationReceived', (p) => `${p.applicant.nickname} 申请加入群组`);
		this.onNotify('GroupAnnouncementCreated', (p) => `群组发布了新公告: ${p.title}`);
		this.onNotify(
			'MessageMentioned',
			(p) => `${p.user.nickname} 在${p.target_type === 'moment' ? '动态' : '评论'}中提到了你`
		);

		// 群组邀请响应（条件 toast）
		websocketStore.on('GroupInviteResponse', (payload) => {
			log('[WS] GroupInviteResponse:', payload);
			if (payload.status === 'accepted') {
				toast.success(`${payload.user.nickname} 接受了群组邀请`);
			}
			this.notificationHandlers.forEach((handler) => handler(payload));
		});

		// 群组申请响应（条件 toast）
		websocketStore.on('GroupApplicationResponse', (payload) => {
			log('[WS] GroupApplicationResponse:', payload);
			if (payload.status === 'accepted') {
				toast.success(`你的入群申请已通过 - ${payload.group.name}`);
			}
			this.notificationHandlers.forEach((handler) => handler(payload));
		});

		// 群组成员加入（仅 toast，不转发 notification）
		websocketStore.on('GroupMemberJoined', (payload: GroupMemberJoinedPayload) => {
			log('[WS] GroupMemberJoined:', payload);
			toast.info(`${payload.user.nickname} 加入了群组`);
		});

		// 私聊已读回执（spec: PrivateMessageRead）
		websocketStore.on('PrivateMessageRead', (payload) => {
			log('[WS] 私聊消息已读:', payload);
		});

		// 订阅状态快照（spec: SubscriptionsSnapshot）
		websocketStore.on('SubscriptionsSnapshot', (payload) => {
			log('[WS] 订阅状态快照:', payload);
		});

		// 群组成员离开（GroupMemberLeft）
		websocketStore.on('GroupMemberLeft', (payload: GroupMemberLeftPayload) => {
			log('[WS] 群组成员离开:', payload);
			toast.info(`${payload.user.nickname} 离开了群组`);
		});

		// 群组成员被移除（GroupMemberRemoved）
		websocketStore.on('GroupMemberRemoved', (payload: GroupMemberRemovedPayload) => {
			log('[WS] 群组成员被移除:', payload);

			const isCurrentUser = payload.user.user_id === authStore.currentUserId;

			if (isCurrentUser) {
				toast.warning('你已被移出群组');
				// 如果在群组相关页面，跳转到群组列表
				if (browser) {
					const currentPath = window.location.pathname;
					if (
						currentPath.includes(`/groups/${payload.group_id}`) ||
						currentPath.includes(`/chat/groups/${payload.group_id}`)
					) {
						goto('/groups');
					}
				}
			} else {
				const reasonText = payload.reason === 'banned' ? '被封禁' : '被移除';
				toast.info(`${payload.user.nickname} ${reasonText}`);
			}
		});

		// 群组成员被禁言（GroupMemberMuted）
		websocketStore.on('GroupMemberMuted', (payload: GroupMemberMutedPayload) => {
			log('[WS] 群组成员被禁言:', payload);

			const isCurrentUser = payload.user.user_id === authStore.currentUserId;
			const mutedUntil = new Date(payload.muted_until).toLocaleString('zh-CN');

			if (isCurrentUser) {
				toast.warning(`你已被禁言，直到 ${mutedUntil}`);
			} else {
				toast.info(`${payload.user.nickname} 已被禁言，直到 ${mutedUntil}`);
			}
		});

		// 群组成员解除禁言（GroupMemberUnmuted）
		websocketStore.on('GroupMemberUnmuted', (payload: GroupMemberUnmutedPayload) => {
			log('[WS] 群组成员解除禁言:', payload);

			const isCurrentUser = payload.user.user_id === authStore.currentUserId;

			if (isCurrentUser) {
				toast.success('你的禁言已解除');
			} else {
				toast.info(`${payload.user.nickname} 的禁言已解除`);
			}
		});

		// 群组信息更新（GroupUpdated）
		websocketStore.on('GroupUpdated', (payload: GroupUpdatedPayload) => {
			log('[WS] 群组信息更新:', payload);

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
		});

		// 群组解散（GroupDismissed，spec 命名）
		websocketStore.on('GroupDismissed', (payload: GroupDismissedPayload) => {
			log('[WS] 群组解散:', payload);
			toast.warning('群组已解散');

			// 如果在群组相关页面，跳转到群组列表
			if (browser) {
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
				case 'error':
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

			// 先断开 WebSocket（设置 shouldReconnect=false，阻止自动重连），再登出
			websocketStore.disconnect();
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

		// 私聊正在输入（spec: PrivateUserTyping）
		websocketStore.on('PrivateUserTyping', (payload) => {
			log('[WS] 私聊正在输入:', payload);
		});

		// 群聊正在输入（spec: GroupUserTyping）
		websocketStore.on('GroupUserTyping', (payload) => {
			log('[WS] 群聊正在输入:', payload);
		});

		// 多端同步事件（⏳ 延期实现）
		websocketStore.on('SelfReadSync', (_payload) => {
			// ⏳ 延期实现
		});
		websocketStore.on('SelfChatSettingsSync', (_payload) => {
			// ⏳ 延期实现
		});
		websocketStore.on('SelfSettingsUpdated', (_payload) => {
			// ⏳ 延期实现
		});
		websocketStore.on('SelfRelationSync', (_payload) => {
			// ⏳ 延期实现
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

	// ============ 客户端事件发送 ============

	// Typing 节流状态（spec: 最多每 3 秒一次，停止 3 秒后自动发 false）
	private static readonly TYPING_THROTTLE_MS = 3000;
	private typingThrottleTimers = new Map<string, ReturnType<typeof setTimeout>>();
	private typingLastSent = new Map<string, number>();
	private typingDebounceTimers = new Map<string, ReturnType<typeof setTimeout>>();
	// 记录当前正在输入的目标，用于切换会话时取消
	private activeTypingTargets = new Set<string>();

	/** 发送私聊正在输入状态（带节流 + 自动取消） */
	sendPrivateTyping(targetUserId: string, isTyping: boolean) {
		this.dispatchTyping(`private:${targetUserId}`, isTyping, (typing) => ({
			type: 'PrivateTyping',
			payload: { target_user_id: targetUserId, is_typing: typing }
		}));
	}

	/** 发送群聊正在输入状态（带节流 + 自动取消） */
	sendGroupTyping(groupId: string, isTyping: boolean) {
		this.dispatchTyping(`group:${groupId}`, isTyping, (typing) => ({
			type: 'GroupTyping',
			payload: { group_id: groupId, is_typing: typing }
		}));
	}

	/** 统一的 typing 分发（节流 + 防抖逻辑复用） */
	private dispatchTyping(
		key: string,
		isTyping: boolean,
		createEvent: (is_typing: boolean) => import('./types').ClientEvent
	) {
		if (isTyping) {
			this.sendTypingThrottled(
				key,
				() => websocketStore.send(createEvent(true)),
				() => websocketStore.send(createEvent(false))
			);
		} else {
			this.cancelTyping(key, () => websocketStore.send(createEvent(false)));
		}
	}

	/**
	 * 切换会话时取消所有活跃的输入状态
	 * spec: 切换会话时对上一个对象立即发 is_typing: false
	 */
	cancelAllTyping() {
		for (const key of this.activeTypingTargets) {
			const colonIndex = key.indexOf(':');
			if (colonIndex === -1) continue;
			const type = key.substring(0, colonIndex);
			const id = key.substring(colonIndex + 1);
			if (type === 'private') {
				websocketStore.send({
					type: 'PrivateTyping',
					payload: { target_user_id: id, is_typing: false }
				});
			} else if (type === 'group') {
				websocketStore.send({
					type: 'GroupTyping',
					payload: { group_id: id, is_typing: false }
				});
			}
		}
		// 清理所有定时器和状态
		for (const timer of this.typingThrottleTimers.values()) clearTimeout(timer);
		for (const timer of this.typingDebounceTimers.values()) clearTimeout(timer);
		this.typingThrottleTimers.clear();
		this.typingDebounceTimers.clear();
		this.typingLastSent.clear();
		this.activeTypingTargets.clear();
	}

	/**
	 * 节流发送 typing（内部方法）
	 */
	private sendTypingThrottled(key: string, sendFn: () => void, cancelFn: () => void) {
		const now = Date.now();
		const lastSent = this.typingLastSent.get(key) ?? 0;
		const elapsed = now - lastSent;

		// 节流：距上次发送不足 3 秒则跳过
		if (elapsed >= WebSocketEventHandlers.TYPING_THROTTLE_MS) {
			sendFn();
			this.typingLastSent.set(key, now);
			this.activeTypingTargets.add(key);
		}

		// 重置防抖定时器：停止输入 3 秒后自动发 false
		const existingDebounce = this.typingDebounceTimers.get(key);
		if (existingDebounce) clearTimeout(existingDebounce);

		this.typingDebounceTimers.set(key, setTimeout(() => {
			cancelFn();
			this.typingDebounceTimers.delete(key);
			this.typingLastSent.delete(key);
			this.activeTypingTargets.delete(key);
		}, WebSocketEventHandlers.TYPING_THROTTLE_MS));
	}

	/**
	 * 取消 typing（内部方法）
	 */
	private cancelTyping(key: string, sendFn: () => void) {
		// 清理节流和防抖定时器
		const throttleTimer = this.typingThrottleTimers.get(key);
		if (throttleTimer) {
			clearTimeout(throttleTimer);
			this.typingThrottleTimers.delete(key);
		}
		const debounceTimer = this.typingDebounceTimers.get(key);
		if (debounceTimer) {
			clearTimeout(debounceTimer);
			this.typingDebounceTimers.delete(key);
		}

		// 只有当我们之前确实发送过 typing:true 才发 false
		if (this.activeTypingTargets.has(key)) {
			sendFn();
			this.activeTypingTargets.delete(key);
		}
		this.typingLastSent.delete(key);
	}
}

// ============ 导出单例 ============

// 必须先创建实例，因为 WebSocketEventHandlers 依赖 websocketStore
export const subscriptionManager = new SubscriptionManager();
export const eventHandlers = new WebSocketEventHandlers();
