/**
 * WebSocket Store - 连接管理、鉴权、心跳
 * 严格遵循 @spec/api_spec_websocket.mdc
 */

import { authStore } from '$lib/stores/auth.svelte';
import type { ServerEvent, ClientEvent } from './types';

// ============ 常量配置（来自规范）============

const PING_INTERVAL = 30000; // 30秒
const PONG_TIMEOUT = 10000; // 10秒
const SERVER_TIMEOUT = 90000; // 90秒
const AUTH_TIMEOUT = 5000; // 5秒
const MAX_RECONNECT_ATTEMPTS = 5;

// 指数退避重连延迟（秒）
const RECONNECT_DELAYS = [1, 2, 4, 8, 16, 30];

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
	// WebSocket 连接
	private ws = $state<WebSocket | null>(null);
	private wsUrl = $state('');

	// 连接状态
	connectionState = $state<ConnectionState>('disconnected');
	connectionId = $state<string | null>(null);
	lastError = $state<string | null>(null);

	// 重连控制
	private reconnectAttempts = 0;
	private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
	private shouldReconnect = true;

	// 心跳控制
	private pingTimer: ReturnType<typeof setInterval> | null = null;
	private pongTimer: ReturnType<typeof setTimeout> | null = null;
	private lastPingTime = 0;

	// 事件监听器
	private eventListeners = new Map<string, Set<(payload: unknown) => void>>();

	// 待发送队列（连接建立前的消息）
	private pendingMessages: ClientEvent[] = [];

	// ============ 公共方法 ============

	/**
	 * 连接 WebSocket
	 */
	connect(url?: string) {
		if (this.connectionState === 'connected' || this.connectionState === 'connecting') {
			console.warn('[WebSocket] 已经连接或正在连接中');
			return;
		}

		// 检查是否已登录
		if (!authStore.isAuthenticated || !authStore.currentAccount?.access_token) {
			console.warn('[WebSocket] 未登录，无法建立连接');
			return;
		}

		// 使用提供的 URL 或默认 URL
		if (url) {
			this.wsUrl = url;
		} else if (!this.wsUrl) {
			// 从当前 HTTP URL 推导 WebSocket URL
			const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
			const host = window.location.host;
			this.wsUrl = `${protocol}//${host}/ws`;
		}

		this.connectionState = 'connecting';
		this.lastError = null;
		this.shouldReconnect = true;

		try {
			console.log('[WebSocket] 正在连接...', this.wsUrl);
			this.ws = new WebSocket(this.wsUrl);
			this.setupWebSocketHandlers();
		} catch (error) {
			console.error('[WebSocket] 连接失败:', error);
			this.handleConnectionError('连接失败');
		}
	}

	/**
	 * 断开连接
	 */
	disconnect() {
		console.log('[WebSocket] 主动断开连接');
		this.shouldReconnect = false;
		this.cleanup();
	}

	/**
	 * 发送事件
	 */
	send(event: ClientEvent) {
		if (this.connectionState !== 'connected') {
			// 连接未建立，加入待发送队列
			this.pendingMessages.push(event);
			console.warn('[WebSocket] 连接未建立，消息已加入队列:', event.type);
			return;
		}

		try {
			const message = JSON.stringify(event);
			this.ws?.send(message);
			console.log('[WebSocket] 发送消息:', event.type);
		} catch (error) {
			console.error('[WebSocket] 发送消息失败:', error);
		}
	}

	/**
	 * 监听事件
	 */
	on<T = unknown>(eventType: string, handler: (payload: T) => void) {
		if (!this.eventListeners.has(eventType)) {
			this.eventListeners.set(eventType, new Set());
		}
		this.eventListeners.get(eventType)!.add(handler as (payload: unknown) => void);

		// 返回取消监听函数
		return () => {
			this.eventListeners.get(eventType)?.delete(handler as (payload: unknown) => void);
		};
	}

	/**
	 * 移除事件监听
	 */
	off(eventType: string, handler?: (payload: unknown) => void) {
		if (!handler) {
			this.eventListeners.delete(eventType);
		} else {
			this.eventListeners.get(eventType)?.delete(handler);
		}
	}

	// ============ 私有方法 ============

	/**
	 * 设置 WebSocket 事件处理器
	 */
	private setupWebSocketHandlers() {
		if (!this.ws) return;

		this.ws.onopen = () => {
			console.log('[WebSocket] 连接已建立，开始鉴权...');
			this.connectionState = 'authenticating';
			this.reconnectAttempts = 0; // 重置重连次数
			this.authenticate();
		};

		this.ws.onmessage = (event) => {
			try {
				const message: ServerEvent = JSON.parse(event.data);
				this.handleServerEvent(message);
			} catch (error) {
				console.error('[WebSocket] 解析消息失败:', error);
			}
		};

		this.ws.onerror = (error) => {
			console.error('[WebSocket] 连接错误:', error);
			this.handleConnectionError('连接错误');
		};

		this.ws.onclose = (event) => {
			console.log('[WebSocket] 连接已关闭:', event.code, event.reason);
			this.handleConnectionClose(event.code, event.reason);
		};
	}

	/**
	 * 鉴权
	 */
	private authenticate() {
		const accessToken = authStore.currentAccount?.access_token;
		if (!accessToken) {
			console.error('[WebSocket] 无法鉴权：缺少 access_token');
			this.handleConnectionError('缺少 access_token');
			return;
		}

		// 发送鉴权消息
		const authEvent: ClientEvent = {
			type: 'Auth',
			payload: {
				access_token: accessToken,
				device_fingerprint: this.getDeviceFingerprint()
			}
		};

		try {
			const message = JSON.stringify(authEvent);
			this.ws?.send(message);
			console.log('[WebSocket] 已发送鉴权请求');
		} catch (error) {
			console.error('[WebSocket] 发送鉴权消息失败:', error);
			this.handleConnectionError('鉴权失败');
		}
	}

	/**
	 * 处理服务端事件
	 */
	private handleServerEvent(event: ServerEvent) {
		console.log('[WebSocket] 收到事件:', event.type);

		switch (event.type) {
			case 'ConnectionEstablished':
				this.handleConnectionEstablished(event.payload);
				break;

			case 'Pong':
				this.handlePong();
				break;

			case 'Error':
				this.handleServerError(event.payload);
				break;

			default:
				// 触发事件监听器
				this.emitEvent(event.type, event.payload);
		}
	}

	/**
	 * 处理连接建立
	 */
	private handleConnectionEstablished(payload: any) {
		this.connectionState = 'connected';
		this.connectionId = payload.connection_id ?? null;
		this.lastError = null;

		console.log('[WebSocket] 连接已建立:', payload);

		// 启动心跳
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
		console.log('[WebSocket] 收到 Pong，延迟:', latency, 'ms');
	}

	/**
	 * 处理服务端错误
	 */
	private handleServerError(payload: any) {
		console.error('[WebSocket] 服务端错误:', payload);
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
		this.cleanup(false); // 不关闭 WebSocket（已经出错）
		this.scheduleReconnect();
	}

	/**
	 * 处理连接关闭
	 */
	private handleConnectionClose(code: number, reason: string) {
		this.cleanup(false);

		if (this.shouldReconnect) {
			this.scheduleReconnect();
		} else {
			this.connectionState = 'disconnected';
		}
	}

	/**
	 * 启动心跳
	 */
	private startHeartbeat() {
		// 清除旧的定时器
		this.stopHeartbeat();

		// 每 30 秒发送一次 Ping
		this.pingTimer = setInterval(() => {
			this.sendPing();
		}, PING_INTERVAL);

		console.log('[WebSocket] 心跳已启动');
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
			payload: {
				timestamp: new Date().toISOString()
			}
		};

		this.send(pingEvent);

		// 设置 Pong 超时
		this.pongTimer = setTimeout(() => {
			console.error('[WebSocket] Pong 超时，连接异常');
			this.handleConnectionError('Pong 超时');
		}, PONG_TIMEOUT);
	}

	/**
	 * 安排重连
	 */
	private scheduleReconnect() {
		if (!this.shouldReconnect || this.reconnectAttempts >= MAX_RECONNECT_ATTEMPTS) {
			console.log('[WebSocket] 达到最大重连次数，停止重连');
			this.connectionState = 'failed';
			return;
		}

		this.connectionState = 'reconnecting';
		const delay = RECONNECT_DELAYS[Math.min(this.reconnectAttempts, RECONNECT_DELAYS.length - 1)];
		this.reconnectAttempts++;

		console.log(
			`[WebSocket] ${delay}秒后重连（第 ${this.reconnectAttempts}/${MAX_RECONNECT_ATTEMPTS} 次）`
		);

		this.reconnectTimer = setTimeout(() => {
			this.connect();
		}, delay * 1000);
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
	}

	/**
	 * 发送待发送队列中的消息
	 */
	private flushPendingMessages() {
		if (this.pendingMessages.length === 0) return;

		console.log('[WebSocket] 发送待发送队列中的消息:', this.pendingMessages.length);

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
					console.error(`[WebSocket] 事件监听器执行失败 (${eventType}):`, error);
				}
			}
		}
	}

	/**
	 * 获取设备指纹
	 */
	private getDeviceFingerprint(): string {
		// 简单的设备指纹实现
		return `web_${navigator.userAgent}_${screen.width}x${screen.height}`;
	}
}

// ============ 导出单例 ============

export const websocketStore = new WebSocketStore();
