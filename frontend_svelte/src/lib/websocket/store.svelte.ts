/**
 * WebSocket Store - 连接管理、鉴权、心跳
 * 严格遵循 @spec/api_spec_websocket.mdc
 */

import { browser, dev } from '$app/environment';
import { authStore } from '$lib/stores/auth.svelte';
import { untrack } from 'svelte';
import type { 
	ServerWebSocketEvent,
	ClientEvent, 
	WebSocketEventMap,
	ConnectionEstablishedPayload,
	ErrorPayload
} from './types';

// ============ Dev-only 日志工具 ============
const log = dev ? console.log.bind(console) : () => {};
const warn = dev ? console.warn.bind(console) : () => {};
const logError = dev ? console.error.bind(console) : () => {};

// ============ 常量配置（来自规范）============

const PING_INTERVAL = 30000; // 30秒
const PONG_TIMEOUT = 10000; // 10秒
// SERVER_TIMEOUT = 90000 和 AUTH_TIMEOUT = 5000 保留在注释中，未来用于超时检测

// 指数退避重连延迟（秒）- 符合规范：1s、2s、4s、8s、16s、32s、60s（最大）
const RECONNECT_DELAYS = [1, 2, 4, 8, 16, 32, 60];
const MAX_DELAY = 60; // 最大延迟 60 秒
const JITTER_PERCENT = 0.2; // 抖动 ±20%

// 最大重连次数（约 25 分钟后放弃）
const MAX_RECONNECT_ATTEMPTS = 30;
// 连续失败通知阈值
const CONSECUTIVE_FAILURE_THRESHOLD = 10;

// ============ 连接状态 ============

export type ConnectionState =
	| 'disconnected'
	| 'connecting'
	| 'authenticating'
	| 'connected'
	| 'reconnecting'
	| 'failed';

// ============ WebSocket Store ============

class WebSocketStore {
	// ============ 响应式状态（使用 $state） ============
	
	// WebSocket 连接
	private ws = $state<WebSocket | null>(null);
	private wsUrl = $state('');

	// 连接状态（公开，供外部响应式使用）
	connectionState = $state<ConnectionState>('disconnected');
	connectionId = $state<string | null>(null);
	lastError = $state<string | null>(null);

	// ============ 派生状态（使用 $derived） ============
	
	/** 是否已连接 */
	isConnected = $derived(this.connectionState === 'connected');
	
	/** 是否正在连接中 */
	isConnecting = $derived(
		this.connectionState === 'connecting' || 
		this.connectionState === 'authenticating' ||
		this.connectionState === 'reconnecting'
	);
	
	/** 是否可以发送消息 */
	canSendMessage = $derived(this.connectionState === 'connected');

	// ============ 内部状态（非响应式） ============
	
	// 重连控制
	private reconnectAttempts = 0;
	private consecutiveFailures = 0; // 连续失败次数（用于用户通知）
	private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
	private shouldReconnect = true;
	private userNotified = false; // 是否已通知用户

	// 心跳控制
	private pingTimer: ReturnType<typeof setInterval> | null = null;
	private pongTimer: ReturnType<typeof setTimeout> | null = null;
	private lastPingTime = 0;

	// 事件监听器
	private eventListeners = new Map<string, Set<(payload: unknown) => void>>();

	// 待发送队列（连接建立前的消息）
	private pendingMessages: ClientEvent[] = [];
	
	// 全局事件监听器引用（用于清理）
	private visibilityChangeHandler: (() => void) | null = null;

	// 事件去重：维护最近 1000 个已处理的 event_id
	private processedEventIds = new Set<string>();
	private readonly MAX_PROCESSED_EVENT_IDS = 1000;

	// 服务端配置（从 ConnectionEstablished 获取）
	private serverConfig = $state<{
		ping_interval: number;
		ping_timeout: number;
		max_subscriptions: {
			user: number;
			group: number;
		};
	} | null>(null);

	// serverTime 保留于注释中，未来用于断线恢复逻辑
	private disconnectTime = 0; // 断开时间戳

	// 防止并发连接的锁
	private connectingLock = false;
	
	// 记录最后一次连接尝试的时间，防止频繁连接
	private lastConnectAttempt = 0;
	private readonly MIN_CONNECT_INTERVAL = 2000; // 最小连接间隔 2 秒

	// ============ 构造函数 ============

	constructor() {
		// 只在浏览器环境初始化
		if (!browser) return;

		log('[WebSocket] 初始化自动生命周期管理');

		// 使用 $effect.root() 在模块顶层创建 effect（独立于组件生命周期）
		$effect.root(() => {
			// 监听认证状态变化，自动管理连接
			$effect(() => {
				const isAuthenticated = authStore.isAuthenticated && !!authStore.currentAccount?.access_token;

				log('[WebSocket] 认证状态变化:', isAuthenticated);

				if (isAuthenticated) {
					// 用户已登录，检查是否需要连接
					untrack(() => {
						const state = this.connectionState;
						if (state === 'disconnected' || state === 'failed') {
							log('[WebSocket] 用户已登录，自动连接');
							this.connect();
						} else {
							log('[WebSocket] 已在状态:', state);
						}
					});
				} else {
					// 用户未登录，断开连接
					untrack(() => {
						const state = this.connectionState;
						if (state !== 'disconnected' && state !== 'failed') {
							log('[WebSocket] 用户已登出，自动断开');
							this.disconnect();
						}
					});
				}
			});
		});

		// 监听页面可见性变化
		this.visibilityChangeHandler = () => {
			if (document.hidden) {
				log('[WebSocket] 页面隐藏');
			} else {
				log('[WebSocket] 页面显示');
				// 页面重新显示时，检查连接是否需要恢复
				untrack(() => {
					const isAuthenticated = authStore.isAuthenticated && !!authStore.currentAccount?.access_token;
					const state = this.connectionState;

					if (isAuthenticated && state === 'failed') {
						log('[WebSocket] 页面显示，尝试恢复连接');
						this.connect();
					}
				});
			}
		};

		document.addEventListener('visibilitychange', this.visibilityChangeHandler);

		// 暴露调试方法到浏览器控制台（仅开发环境）
		if (dev && typeof window !== 'undefined') {
			(window as any).__debugWebSocket = () => this.debug();
			(window as any).__wsStore = this;
		}
	}

	// ============ 公共方法 ============

	/**
	 * 连接 WebSocket
	 * 使用防护模式（早返回）确保连接的唯一性和安全性
	 */
	connect(url?: string): void {
		// 防护 1: 频率限制（至少间隔 2 秒）
		const now = Date.now();
		const timeSinceLastAttempt = now - this.lastConnectAttempt;
		if (timeSinceLastAttempt < this.MIN_CONNECT_INTERVAL) {
			const remaining = Math.ceil((this.MIN_CONNECT_INTERVAL - timeSinceLastAttempt) / 1000);
			warn(`[WebSocket] 连接过于频繁，需等待 ${remaining}秒`);
			return;
		}

		// 防护 2: 并发锁
		if (this.connectingLock) {
			warn('[WebSocket] 连接锁激活中，忽略重复请求');
			return;
		}

		// 防护 3: 状态检查（使用 derived 状态）
		if (this.isConnecting || this.isConnected) {
			warn(`[WebSocket] 当前状态 ${this.connectionState}，无需重连`);
			return;
		}

		// 防护 4: 认证检查
		if (!authStore.isAuthenticated || !authStore.currentAccount?.access_token) {
			warn('[WebSocket] 未登录，无法建立连接');
			return;
		}

		// 防护 5: 清理残留连接
		if (this.ws?.readyState !== WebSocket.CLOSED) {
			warn('[WebSocket] 清理残留连接');
			this.cleanup();
		}

		// 开始连接流程
		this.lastConnectAttempt = now;
		this.connectingLock = true;
		this.shouldReconnect = true;
		this.lastError = null;

		// 手动触发连接（非重连循环）时重置重连计数器
		if (this.connectionState === 'failed' || this.connectionState === 'disconnected') {
			this.reconnectAttempts = 0;
			this.consecutiveFailures = 0;
			this.userNotified = false;
		}

		// 确定 WebSocket URL
		if (url) {
			this.wsUrl = url;
		} else if (!this.wsUrl) {
			const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
			this.wsUrl = `${protocol}//${window.location.host}/ws`;
		}

		// 更新状态并建立连接
		this.connectionState = 'connecting';
		log(`[WebSocket] 连接到 ${this.wsUrl}`);

		try {
			this.ws = new WebSocket(this.wsUrl);
			this.setupWebSocketHandlers();
		} catch (error) {
			logError('[WebSocket] 连接失败:', error);
			this.connectingLock = false;
			this.handleConnectionError('连接初始化失败');
		}
	}

	/**
	 * 断开连接
	 */
	disconnect() {
		log('[WebSocket] 主动断开连接');
		this.shouldReconnect = false;
		this.reconnectAttempts = 0; // 重置重连次数
		this.connectingLock = false; // 释放连接锁
		this.cleanup();
		this.connectionState = 'disconnected';
	}

	/**
	 * 发送事件
	 * 如果未连接，消息会被加入队列，连接建立后自动发送
	 */
	send(event: ClientEvent): void {
		// 使用 derived 状态检查
		if (!this.canSendMessage) {
			this.pendingMessages.push(event);
			warn(`[WebSocket] 未连接(${this.connectionState})，消息已入队: ${event.type}`);
			return;
		}

		try {
			const message = JSON.stringify(event);
			this.ws?.send(message);
			log(`[WebSocket] ✓ 发送: ${event.type}`);
		} catch (error) {
			logError(`[WebSocket] ✗ 发送失败: ${event.type}`, error);
		}
	}

	/**
	 * 监听 WebSocket 事件（类型安全版本）
	 * 返回取消监听的函数（符合 Svelte 清理模式）
	 * 
	 * @example
	 * ```ts
	 * // 现在 payload 会自动推断类型！
	 * const unsubscribe = websocketStore.on('PrivateMessageNew', (payload) => {
	 *   log('收到新消息:', payload.message_id);
	 * });
	 * 
	 * // 在组件卸载时取消监听
	 * onDestroy(unsubscribe);
	 * ```
	 */
	on<K extends keyof WebSocketEventMap>(
		eventType: K,
		handler: (payload: WebSocketEventMap[K]) => void
	): () => void {
		if (!this.eventListeners.has(eventType)) {
			this.eventListeners.set(eventType, new Set());
		}
		
		const typedHandler = handler as (payload: unknown) => void;
		this.eventListeners.get(eventType)!.add(typedHandler);
		log(`[WebSocket] 监听事件: ${eventType}`);

		// 返回清理函数（符合 Svelte 模式）
		return () => {
			this.eventListeners.get(eventType)?.delete(typedHandler);
			log(`[WebSocket] 取消监听: ${eventType}`);
		};
	}

	/**
	 * 移除事件监听（手动方式，推荐使用 on() 返回的清理函数）
	 */
	off(eventType: string, handler?: (payload: unknown) => void): void {
		if (!handler) {
			// 移除该事件的所有监听器
			this.eventListeners.delete(eventType);
			log(`[WebSocket] 移除所有监听: ${eventType}`);
		} else {
			// 移除特定监听器
			this.eventListeners.get(eventType)?.delete(handler);
			log(`[WebSocket] 移除监听: ${eventType}`);
		}
	}

	// ============ 私有方法 ============

	/**
	 * 设置 WebSocket 事件处理器
	 */
	private setupWebSocketHandlers() {
		if (!this.ws) return;

		this.ws.onopen = () => {
			log('[WebSocket] 连接已建立，开始鉴权...');
			this.connectionState = 'authenticating';
			this.reconnectAttempts = 0; // 重置重连次数
			this.consecutiveFailures = 0; // 重置连续失败次数
			this.userNotified = false; // 重置通知状态
			this.authenticate();
		};

		this.ws.onmessage = (event) => {
			try {
				const message: ServerWebSocketEvent = JSON.parse(event.data);
				this.handleServerEvent(message);
			} catch (error) {
				logError('[WebSocket] 解析消息失败:', error);
			}
		};

		this.ws.onerror = (error) => {
			logError('[WebSocket] 连接错误:', error);
			// 注意：不在这里调用 handleConnectionError，等待 onclose 处理
		};

		this.ws.onclose = (event) => {
			log('[WebSocket] 连接已关闭:', event.code, event.reason);
			
			// 检查关闭原因，某些情况下不应重连
			// 1000: 正常关闭
			// 1001: 端点离开
			// 1006: 异常关闭（网络错误）
			// 4000+: 自定义错误码
			
			if (event.code === 1000 || event.code === 1001) {
				// 正常关闭，不重连
				log('[WebSocket] 正常关闭，不重连');
				this.shouldReconnect = false;
			}
			
			this.handleConnectionClose(event.code, event.reason);
		};
	}

	/**
	 * 鉴权
	 */
	private authenticate() {
		// Web 端 access_token 在 Cookie 中，可选传递
		const accessToken = authStore.currentAccount?.access_token;

		// 发送鉴权消息（spec: client_type 必填，access_token Web 端不传）
		const authEvent: ClientEvent = {
			type: 'Auth',
			payload: {
				client_type: 'web',
				access_token: accessToken || undefined
			}
		};

		try {
			const message = JSON.stringify(authEvent);
			this.ws?.send(message);
			log('[WebSocket] 已发送鉴权请求');
		} catch (error) {
			logError('[WebSocket] 发送鉴权消息失败:', error);
			this.handleConnectionError('鉴权失败');
		}
	}

	/**
	 * 处理服务端事件（带 event_id 去重）
	 */
	private handleServerEvent(event: ServerWebSocketEvent) {
		// 检查 event_id 去重
		if (this.processedEventIds.has(event.event_id)) {
			warn(`[WebSocket] 重复事件已忽略: ${event.event_id} (${event.type})`);
			return;
		}

		// 记录已处理的事件 ID
		this.processedEventIds.add(event.event_id);
		
		// 维护最近 1000 个 event_id
		if (this.processedEventIds.size > this.MAX_PROCESSED_EVENT_IDS) {
			// 删除最旧的（FIFO，但 Set 没有顺序，这里简化处理：清空重建）
			// 实际场景中，可以使用 LRU Cache 或循环队列
			const ids = Array.from(this.processedEventIds);
			this.processedEventIds.clear();
			// 保留最后 1000 个
			ids.slice(-this.MAX_PROCESSED_EVENT_IDS).forEach(id => {
				this.processedEventIds.add(id);
			});
		}

		log('[WebSocket] 收到事件:', event.type, 'event_id:', event.event_id);

		switch (event.type) {
			case 'ConnectionEstablished':
				this.handleConnectionEstablished(event.payload as ConnectionEstablishedPayload);
				break;

			case 'Pong':
				this.handlePong();
				break;

			case 'Error':
				this.handleServerError(event.payload as ErrorPayload);
				break;

			default:
				// 触发事件监听器
				this.emitEvent(event.type, event.payload);
		}
	}

	/**
	 * 处理连接建立（类型安全！）
	 */
	private handleConnectionEstablished(payload: ConnectionEstablishedPayload) {
		this.connectionState = 'connected';
		this.connectionId = payload.connection_id ?? null;
		this.lastError = null;
		this.connectingLock = false; // 释放连接锁

		// 保存服务端配置
		this.serverConfig = payload.config;

		// 计算断开时长（如果之前有断开时间）
		if (this.disconnectTime > 0) {
			const disconnectDuration = Date.now() - this.disconnectTime;
			log(`[WebSocket] 断开时长: ${(disconnectDuration / 1000).toFixed(1)}秒`);
			
			// 断开超过 10 秒，建议通过 HTTP API 补拉离线数据
			if (disconnectDuration > 10000) {
				log('[WebSocket] 断开超过 10 秒，建议补拉离线数据');
			}
		}

		// 更新未读计数到 UI store
		// 注意：这里需要更新 notification store 或其他 UI store
		// 由于 notification store 可能没有直接暴露未读计数更新方法，
		// 这里先记录，后续可以通过事件或直接调用更新
		log('[WebSocket] 未读摘要:', payload.unread_summary);

		log('[WebSocket] 连接已建立:', {
			connection_id: payload.connection_id,
			user_id: payload.user_id,
			server_time: payload.server_time,
			config: payload.config,
			unread_summary: payload.unread_summary
		});

		// 启动心跳（使用服务端配置的间隔）
		this.startHeartbeat();

		// 发送待发送队列中的消息
		this.flushPendingMessages();

		// 触发连接成功事件
		this.emitEvent('ConnectionEstablished', payload);
	}

	/**
	 * 处理 Pong 响应
	 */
	private handlePong() {
		// 清除 pong 超时定时器
		if (this.pongTimer) {
			clearTimeout(this.pongTimer);
			this.pongTimer = null;
		}

		const latency = Date.now() - this.lastPingTime;
		log('[WebSocket] 收到 Pong，延迟:', latency, 'ms');
	}

	/**
	 * 处理服务端错误（类型安全！）
	 */
	private handleServerError(payload: ErrorPayload) {
		logError('[WebSocket] 服务端错误:', payload);
		this.lastError = payload.message ?? '未知错误';
		this.emitEvent('Error', payload);

		// 鉴权失败或严重错误，断开连接
		if (payload.code === 20001) {
			this.shouldReconnect = false;
			this.cleanup();
		}
	}

	/**
	 * 处理连接错误
	 */
	private handleConnectionError(error: string) {
		this.lastError = error;
		this.connectionState = 'failed';
		this.connectingLock = false; // 释放连接锁
		this.cleanup(false); // 不关闭 WebSocket（已经出错）
		this.scheduleReconnect();
	}

	/**
	 * 处理连接关闭
	 */
	private handleConnectionClose(_code: number, _reason: string) {
		// 记录断开时间
		this.disconnectTime = Date.now();
		
		this.connectingLock = false; // 释放连接锁
		this.cleanup(false);

		if (this.shouldReconnect) {
			this.scheduleReconnect();
		} else {
			this.connectionState = 'disconnected';
		}
	}

	/**
	 * 启动心跳（使用服务端配置的间隔）
	 */
	private startHeartbeat() {
		// 清除旧的定时器
		this.stopHeartbeat();

		// 使用服务端配置的 ping_interval，如果没有则使用默认值
		const pingInterval = this.serverConfig?.ping_interval 
			? this.serverConfig.ping_interval * 1000 // 转换为毫秒
			: PING_INTERVAL;

		// 使用服务端配置的 ping_timeout，如果没有则使用默认值
		const pongTimeout = this.serverConfig?.ping_timeout
			? this.serverConfig.ping_timeout * 1000 // 转换为毫秒
			: PONG_TIMEOUT;

		// 每 pingInterval 毫秒发送一次 Ping
		this.pingTimer = setInterval(() => {
			this.sendPing();
		}, pingInterval);

		log(`[WebSocket] 心跳已启动 (间隔: ${pingInterval}ms, 超时: ${pongTimeout}ms)`);
	}

	/**
	 * 停止心跳
	 */
	private stopHeartbeat() {
		if (this.pingTimer) {
			clearInterval(this.pingTimer);
			this.pingTimer = null;
		}
		if (this.pongTimer) {
			clearTimeout(this.pongTimer);
			this.pongTimer = null;
		}
	}

	/**
	 * 发送 Ping
	 */
	private sendPing() {
		if (this.connectionState !== 'connected') {
			return;
		}

		this.lastPingTime = Date.now();

		const pingEvent: ClientEvent = {
			type: 'Ping',
			payload: {}
		};

		this.send(pingEvent);

		// 设置 Pong 超时（使用服务端配置的超时时间）
		const pongTimeout = this.serverConfig?.ping_timeout
			? this.serverConfig.ping_timeout * 1000 // 转换为毫秒
			: PONG_TIMEOUT;

		this.pongTimer = setTimeout(() => {
			logError('[WebSocket] Pong 超时，连接异常');
			this.handleConnectionError('Pong 超时');
		}, pongTimeout);
	}

	/**
	 * 安排重连
	 * 
	 * 规范要求：
	 * - 指数退避：1s、2s、4s、8s、16s、32s、60s（最大）
	 * - 随机抖动：延迟 ± 20%
	 * - 最大重连次数：30 次（约 25 分钟），超过后停止
	 * - 连续失败 10 次后提示用户
	 */
	private scheduleReconnect() {
		// 如果已经在重连中，不要重复安排
		if (this.reconnectTimer) {
			warn('[WebSocket] 重连已安排，忽略重复调用');
			return;
		}

		if (!this.shouldReconnect) {
			log('[WebSocket] 不应重连，停止重连');
			this.connectionState = 'failed';
			this.connectingLock = false; // 释放锁
			return;
		}

		// 达到最大重连次数，放弃自动重连
		if (this.reconnectAttempts >= MAX_RECONNECT_ATTEMPTS) {
			logError(`[WebSocket] 已达最大重连次数 ${MAX_RECONNECT_ATTEMPTS}，停止自动重连`);
			this.shouldReconnect = false;
			this.connectionState = 'failed';
			this.connectingLock = false;
			this.emitEvent('ConnectionFailed', {
				message: '连接失败，请检查网络后重试',
				attempts: this.reconnectAttempts
			});
			return;
		}

		this.connectionState = 'reconnecting';
		
		// 计算延迟（带抖动）
		const delay = this.getReconnectDelay(this.reconnectAttempts);
		this.reconnectAttempts++;
		this.consecutiveFailures++;

		// 连续失败 10 次后提示用户
		if (this.consecutiveFailures >= CONSECUTIVE_FAILURE_THRESHOLD && !this.userNotified) {
			this.userNotified = true;
			this.notifyUserConnectionIssue();
			warn('[WebSocket] 连续失败次数达到阈值，已通知用户');
		}

		log(
			`[WebSocket] ${delay.toFixed(1)}秒后重连（第 ${this.reconnectAttempts}/${MAX_RECONNECT_ATTEMPTS} 次，连续失败 ${this.consecutiveFailures} 次）`
		);

		this.reconnectTimer = setTimeout(() => {
			this.reconnectTimer = null; // 清除定时器引用
			this.connectingLock = false; // 释放锁，准备重连
			this.connect();
		}, delay * 1000);
	}

	/**
	 * 计算重连延迟（带随机抖动）
	 */
	private getReconnectDelay(attempt: number): number {
		const baseDelay = RECONNECT_DELAYS[Math.min(attempt, RECONNECT_DELAYS.length - 1)] ?? MAX_DELAY;
		const jitter = baseDelay * JITTER_PERCENT * (Math.random() * 2 - 1); // ±20%
		const finalDelay = Math.min(baseDelay + jitter, MAX_DELAY);
		return Math.max(finalDelay, 0.1); // 最小 0.1 秒
	}

	/**
	 * 通知用户连接问题
	 */
	private notifyUserConnectionIssue() {
		// 触发自定义事件，让 UI 层显示通知
		this.emitEvent('ConnectionIssue', {
			message: '网络连接不稳定，正在尝试重连...',
			consecutiveFailures: this.consecutiveFailures
		});
	}

	/**
	 * 清理资源
	 */
	private cleanup(closeWebSocket = true) {
		this.stopHeartbeat();

		if (this.reconnectTimer) {
			clearTimeout(this.reconnectTimer);
			this.reconnectTimer = null;
		}

		if (closeWebSocket && this.ws) {
			this.ws.close();
		}

		this.ws = null;
		this.connectionId = null;
		// 注意：不在这里释放 connectingLock，因为可能需要重连
	}
	
	/**
	 * 彻底清理所有资源（用于页面卸载）
	 */
	destroy() {
		log('[WebSocket] 彻底清理所有资源');
		
		// 清理 WebSocket 连接和定时器
		this.shouldReconnect = false;
		this.cleanup();
		
		// 移除全局事件监听器
		if (this.visibilityChangeHandler) {
			document.removeEventListener('visibilitychange', this.visibilityChangeHandler);
			this.visibilityChangeHandler = null;
		}
		
		// 清空所有事件监听器
		this.eventListeners.clear();
		
		log('[WebSocket] 资源清理完成');
	}

	/**
	 * 发送待发送队列中的消息
	 */
	private flushPendingMessages() {
		if (this.pendingMessages.length === 0) return;

		log('[WebSocket] 发送待发送队列中的消息:', this.pendingMessages.length);

		for (const message of this.pendingMessages) {
			this.send(message);
		}

		this.pendingMessages = [];
	}

	/**
	 * 触发事件
	 */
	private emitEvent(eventType: string, payload: unknown) {
		const listeners = this.eventListeners.get(eventType);
		if (listeners) {
			for (const listener of listeners) {
				try {
					listener(payload);
				} catch (error) {
					logError(`[WebSocket] 事件监听器执行失败 (${eventType}):`, error);
				}
			}
		}
	}

	/**
	 * 调试方法：打印当前状态
	 * 在浏览器控制台使用：window.__debugWebSocket()
	 */
	debug() {
		const now = Date.now();
		const timeSinceLastConnect = now - this.lastConnectAttempt;
		
		console.group('🔍 WebSocket Debug Info');
		
		// 连接状态
		console.group('📡 连接状态');
		log('状态:', this.connectionState);
		log('连接 ID:', this.connectionId ?? '无');
		log('WebSocket URL:', this.wsUrl || '未设置');
		log('原生状态:', this.ws ? 
			['🟡 CONNECTING', '🟢 OPEN', '🟠 CLOSING', '🔴 CLOSED'][this.ws.readyState] : 
			'❌ 无连接'
		);
		console.groupEnd();
		
		// 派生状态
		console.group('🎯 派生状态');
		log('isConnected:', this.isConnected);
		log('isConnecting:', this.isConnecting);
		log('canSendMessage:', this.canSendMessage);
		console.groupEnd();
		
		// 重连信息
		console.group('🔄 重连控制');
		log('重连次数:', `${this.reconnectAttempts}/${MAX_RECONNECT_ATTEMPTS}`);
		log('是否重连:', this.shouldReconnect);
		log('连接锁:', this.connectingLock ? '🔒 已锁定' : '🔓 未锁定');
		log('距上次连接:', `${(timeSinceLastConnect / 1000).toFixed(1)}秒`);
		log('重连定时器:', this.reconnectTimer ? '⏰ 运行中' : '⏹️ 未运行');
		console.groupEnd();
		
		// 心跳信息
		console.group('💓 心跳状态');
		log('Ping 定时器:', this.pingTimer ? '⏰ 运行中' : '⏹️ 未运行');
		log('Pong 定时器:', this.pongTimer ? '⏰ 运行中' : '⏹️ 未运行');
		if (this.lastPingTime > 0) {
			const pingSince = ((now - this.lastPingTime) / 1000).toFixed(1);
			log('距上次 Ping:', `${pingSince}秒`);
		}
		console.groupEnd();
		
		// 消息队列
		console.group('📨 消息队列');
		log('待发送消息:', this.pendingMessages.length);
		if (this.pendingMessages.length > 0) {
			console.table(this.pendingMessages.map((msg, i) => ({ 序号: i + 1, 类型: msg.type })));
		}
		console.groupEnd();
		
		// 事件监听器
		console.group('🎧 事件监听器');
		const listeners = Array.from(this.eventListeners.entries()).map(([type, set]) => ({
			事件类型: type,
			监听器数量: set.size
		}));
		if (listeners.length > 0) {
			console.table(listeners);
		} else {
			log('无监听器');
		}
		console.groupEnd();
		
		// 错误信息
		if (this.lastError) {
			console.group('❌ 最后错误');
			logError(this.lastError);
			console.groupEnd();
		}
		
		// 操作提示
		console.group('🛠️ 调试命令');
		log('手动连接: window.__wsStore.connect()');
		log('手动断开: window.__wsStore.disconnect()');
		log('发送测试: window.__wsStore.send({ type: "Ping", payload: {} })');
		console.groupEnd();
		
		console.groupEnd();
		
		return this;
	}
}

// ============ 导出单例 ============

export const websocketStore = new WebSocketStore();
