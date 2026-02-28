/**
 * 聊天实时同步通用工具
 *
 * 目标：
 * - 统一处理 WS 连接状态变化，识别“重连成功”时机
 * - 统一处理 HTTP 回补消息与本地消息的去重合并
 */

interface MessageWithIdAndTime {
	message_id: string;
	sent_at: string;
}

export interface ReconnectUpdateResult {
	justReconnected: boolean;
	shouldBackfill: boolean;
}

export function shouldBackfillAfterReconnect(
	disconnectDurationMs: number | null,
	thresholdMs = 10000
): boolean {
	if (disconnectDurationMs === null) return false;
	return disconnectDurationMs > thresholdMs;
}

/**
 * 跟踪连接状态，识别是否为“重连后首次连通”
 */
export function createReconnectTracker() {
	let wsWasConnected = false;
	let wsHasConnectedOnce = false;

	function update(
		connected: boolean,
		disconnectDurationMs: number | null,
		thresholdMs = 10000
	): ReconnectUpdateResult {
		const justReconnected = connected && wsHasConnectedOnce && !wsWasConnected;
		wsWasConnected = connected;
		if (connected && !wsHasConnectedOnce) {
			wsHasConnectedOnce = true;
		}
		return {
			justReconnected,
			shouldBackfill: justReconnected && shouldBackfillAfterReconnect(disconnectDurationMs, thresholdMs)
		};
	}

	function reset() {
		wsWasConnected = false;
		wsHasConnectedOnce = false;
	}

	return {
		update,
		reset
	};
}

/**
 * 将回补消息与当前消息按主键去重后合并，并按 sent_at/message_id 稳定排序
 */
export function mergeMessagesByIdAndSentAt<T extends MessageWithIdAndTime>(
	currentMessages: T[],
	recentMessages: T[]
): T[] {
	const merged = new Map(currentMessages.map((message) => [message.message_id, message] as const));
	for (const message of recentMessages) {
		merged.set(message.message_id, message);
	}
	return Array.from(merged.values()).sort((a, b) => {
		const timeDiff = new Date(a.sent_at).getTime() - new Date(b.sent_at).getTime();
		if (timeDiff !== 0) return timeDiff;
		return a.message_id.localeCompare(b.message_id);
	});
}
