/**
 * 聊天实时同步通用工具
 *
 * 纯函数模块，无框架依赖。
 * - 统一处理 WS 连接状态变化，识别"重连成功"时机
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
  thresholdMs = 10000,
): boolean {
  if (disconnectDurationMs === null) return false;
  return disconnectDurationMs > thresholdMs;
}

export function createReconnectTracker() {
  let wsWasConnected = false;
  let wsHasConnectedOnce = false;

  function update(
    connected: boolean,
    disconnectDurationMs: number | null,
    thresholdMs = 10000,
  ): ReconnectUpdateResult {
    const justReconnected = connected && wsHasConnectedOnce && !wsWasConnected;
    wsWasConnected = connected;
    if (connected && !wsHasConnectedOnce) {
      wsHasConnectedOnce = true;
    }
    return {
      justReconnected,
      shouldBackfill:
        justReconnected &&
        shouldBackfillAfterReconnect(disconnectDurationMs, thresholdMs),
    };
  }

  function reset() {
    wsWasConnected = false;
    wsHasConnectedOnce = false;
  }

  return { update, reset };
}

export function mergeMessagesByIdAndSentAt<T extends MessageWithIdAndTime>(
  currentMessages: T[],
  recentMessages: T[],
): T[] {
  const merged = new Map(
    currentMessages.map((message) => [message.message_id, message] as const),
  );
  for (const message of recentMessages) {
    merged.set(message.message_id, message);
  }
  return Array.from(merged.values()).sort((a, b) => {
    const timeDiff =
      new Date(a.sent_at).getTime() - new Date(b.sent_at).getTime();
    if (timeDiff !== 0) return timeDiff;
    return a.message_id.localeCompare(b.message_id);
  });
}

export function resolveLatestVisibleMessageId<T extends { message_id: string }>(
  messages: T[],
  visibleMessageIds: string[],
): string | undefined {
  if (visibleMessageIds.length === 0) return undefined;
  const indexMap = new Map(
    messages.map((message, index) => [message.message_id, index] as const),
  );
  let latestId: string | undefined;
  let latestIndex = -1;
  for (const messageId of visibleMessageIds) {
    const index = indexMap.get(messageId);
    if (index !== undefined && index > latestIndex) {
      latestIndex = index;
      latestId = messageId;
    }
  }
  return latestId ?? visibleMessageIds[visibleMessageIds.length - 1];
}
