import { MessageType } from '../../../core/types/api';
import { ChatActionRulesInput } from './ChatMessageRules';

/**
 * ChatMessageListUtils —— 聊天消息列表纯函数工具（去重/合并/时间格式化）。
 * 从 ChatPage 抽取：入参为消息展示模型，无状态、无 UI 依赖。
 */

/** 本地乐观消息去重时间窗（同内容签名在该窗口内视为重复） */
export const DUPLICATE_MESSAGE_TIME_WINDOW_MS: number = 1500;

/** 消息列表项最小字段集（页面 ChatMessage 接口 extends 本接口，规避结构化类型限制） */
export interface ChatMessageListItem extends ChatActionRulesInput {
  id: string;
  senderId: string;
  type: MessageType;
  sentAt: string;
  content: string;
  isSelf: boolean;
  isSystem: boolean;
  recalled?: boolean;
  pending?: boolean;
}

/** 禁言解除时间展示（MM-DD HH:mm；无效时间戳原样返回） */
export function formatMutedUntil(isoString: string): string {
  const date = new Date(isoString);
  if (Number.isNaN(date.getTime())) {
    return isoString;
  }
  const month = date.getMonth() + 1;
  const day = date.getDate();
  const hours = date.getHours();
  const minutes = date.getMinutes();
  const pad = (n: number): string => (n < 10 ? `0${n}` : `${n}`);
  return `${pad(month)}-${pad(day)} ${pad(hours)}:${pad(minutes)}`;
}

/** 消息时间行展示（HH:mm；无效时间戳返回空串） */
export function formatMessageTime(iso: string): string {
  if (!iso) return '';
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return '';
  return `${d.getHours().toString().padStart(2, '0')}:${d.getMinutes().toString().padStart(2, '0')}`;
}

/** 本地日期 key（用于判断两条消息是否跨天，对齐前端 groupMessagesByDay） */
export function getLocalDayKey(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return '';
  return `${d.getFullYear()}-${d.getMonth()}-${d.getDate()}`;
}

/** 最后一条自己发送的消息 id（对齐前端 lastSentId 已读勾选） */
export function getLastSentMessageId(messages: ChatMessageListItem[]): string {
  for (let i = messages.length - 1; i >= 0; i--) {
    if (messages[i].isSelf) {
      return messages[i].id;
    }
  }
  return '';
}

/** 该索引消息是否需要展示日期分隔胶囊（与上一条不同日 / 首条） */
export function needsDateSeparator(messages: ChatMessageListItem[], index: number): boolean {
  if (index < 0 || index >= messages.length) return false;
  if (index === 0) return true;
  return getLocalDayKey(messages[index].sentAt) !== getLocalDayKey(messages[index - 1].sentAt);
}

export function buildMessageFingerprint(message: ChatMessageListItem): string {
  const sentAtMs = Date.parse(message.sentAt);
  const normalizedSentAt = Number.isNaN(sentAtMs) ? message.sentAt : String(sentAtMs);
  return `${message.senderId}|${message.type}|${normalizedSentAt}|${message.content}`;
}

export function buildMessageContentSignature(message: ChatMessageListItem): string {
  return `${message.senderId}|${message.type}|${message.content}`;
}

/** 去重：优先按 id 精确去重；无 id/新内容按指纹与「近期同内容」时间窗去重（泛型保元素静态类型） */
export function dedupeMessages<T extends ChatMessageListItem>(messages: T[]): T[] {
  const seenIds: Set<string> = new Set();
  const seenFingerprints: Set<string> = new Set();
  const recentContentSentAt: Map<string, number> = new Map();
  const uniqueMessages: T[] = [];

  messages.forEach((message: T) => {
    const normalizedId = message.id.trim();
    if (normalizedId.length > 0) {
      if (seenIds.has(normalizedId)) {
        return;
      }
      seenIds.add(normalizedId);
    }

    const fingerprint = buildMessageFingerprint(message);
    if (seenFingerprints.has(fingerprint)) {
      return;
    }

    const sentAtMs = Date.parse(message.sentAt);
    if (!Number.isNaN(sentAtMs)) {
      const contentSignature = buildMessageContentSignature(message);
      const previousSentAt = recentContentSentAt.get(contentSignature);
      if (previousSentAt !== undefined &&
        Math.abs(sentAtMs - previousSentAt) <= DUPLICATE_MESSAGE_TIME_WINDOW_MS) {
        return;
      }
      recentContentSentAt.set(contentSignature, sentAtMs);
    }
    seenFingerprints.add(fingerprint);
    uniqueMessages.push(message);
  });

  return uniqueMessages;
}

/** 按 sentAt 升序合并两段消息（先整体去重再排序；id 为次序 tie-break） */
export function mergeMessagesById<T extends ChatMessageListItem>(currentMessages: T[], recentMessages: T[]): T[] {
  const mergedMessages = dedupeMessages(currentMessages.concat(recentMessages));
  mergedMessages.sort((a: ChatMessageListItem, b: ChatMessageListItem) => {
    const timeDiff = new Date(a.sentAt).getTime() - new Date(b.sentAt).getTime();
    if (timeDiff !== 0) {
      return timeDiff;
    }
    return a.id.localeCompare(b.id);
  });
  return mergedMessages;
}
