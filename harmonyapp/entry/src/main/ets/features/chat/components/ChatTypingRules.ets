/**
 * ChatTypingRules —— 群聊「正在输入」提示的纯规则（从 ChatPage 抽取）。
 * 只依赖入参（用户 id 列表 + 消息列表），无状态、无副作用，便于复用与推演。
 */

import { TypingMessageSource } from './ChatMessageListUtils';

/** 提示文案里最多列举的名字数（超出则折叠为「等 N 人」） */
export const GROUP_TYPING_NAME_PREVIEW_LIMIT: number = 3;

/**
 * 解析输入中成员的展示名：优先取该成员最近一条消息的昵称，
 * 查不到时回退为「成员+userId 末 4 位」。
 */
export function resolveTypingDisplayName(userId: string, messages: TypingMessageSource[]): string {
  for (let index = messages.length - 1; index >= 0; index--) {
    const message = messages[index];
    if (!message || message.senderId !== userId) {
      continue;
    }
    const nickname = message.senderName ? message.senderName.trim() : '';
    if (nickname.length > 0) {
      return nickname;
    }
  }
  const length = userId.length;
  const suffix = length > 4 ? userId.substring(length - 4, length) : userId;
  return `成员${suffix}`;
}

/**
 * 群聊「正在输入」提示文案：
 * 1 人 → 「A 正在输入…」；≤ 上限 → 「A、B 正在输入…」；超出 → 「A、B、C 等 N 人正在输入…」。
 */
export function buildGroupTypingHint(userIds: string[], messages: TypingMessageSource[]): string {
  const count = userIds.length;
  if (count <= 0) {
    return '';
  }
  const previewCount = count > GROUP_TYPING_NAME_PREVIEW_LIMIT ? GROUP_TYPING_NAME_PREVIEW_LIMIT : count;
  const previewNames: string[] = [];
  for (let index = 0; index < previewCount; index++) {
    previewNames.push(resolveTypingDisplayName(userIds[index], messages));
  }
  if (count === 1) {
    return `${previewNames[0]} 正在输入…`;
  }
  if (count <= GROUP_TYPING_NAME_PREVIEW_LIMIT) {
    return `${previewNames.join('、')} 正在输入…`;
  }
  return `${previewNames.join('、')} 等 ${count} 人正在输入…`;
}
