import { MessageType } from '../../../core/types/api';

/**
 * ChatMessageRules —— 聊天消息操作规则（撤回/编辑/可操作判定）的纯函数模块。
 * 从 ChatPage 抽取：入参为消息展示模型与群角色上下文，无状态、无 UI 依赖，
 * 与前端 MessageBubble 的 canRecall/canEdit 口径逐条对齐。
 */

/** 撤回时间窗（spec message_recall_window_seconds 默认 300s；无时间戳放行，服务端兜底） */
const RECALL_WINDOW_MS: number = 300000;

/** 规则输入：消息展示模型的最小字段集（ChatMessage / ChatMessageView 均满足） */
export interface ChatActionRulesInput {
  sentAt: string;
  isSelf: boolean;
  isSystem: boolean;
  type: MessageType;
  recalled?: boolean;
  pending?: boolean;
}

export function isInRecallWindow(sentAt: string): boolean {
  if (!sentAt) {
    return true;
  }
  const sentAtMs = Date.parse(sentAt);
  if (Number.isNaN(sentAtMs)) {
    return true;
  }
  return Date.now() - sentAtMs < RECALL_WINDOW_MS;
}

/** 自己的消息：时限内可撤回（对齐前端 canRecall = sent && inRecallWindow） */
export function canRecallMessage(msg: ChatActionRulesInput): boolean {
  return msg.isSelf && !msg.recalled && msg.pending !== true && isInRecallWindow(msg.sentAt);
}

/** 群主/管理员撤回他人消息：无时间窗（对齐前端 recallGroupMessageAsAdmin 分支） */
export function canAdminRecallMessage(msg: ChatActionRulesInput, isAdminOrOwner: boolean): boolean {
  return !msg.isSelf && !msg.recalled && !msg.isSystem && isAdminOrOwner;
}

/** 编辑：仅本人的文本消息且未撤回（对齐前端 canEdit = sent；spec 限定 type=text） */
export function canEditMessage(msg: ChatActionRulesInput): boolean {
  return msg.isSelf && !msg.recalled && msg.type === 'text' && msg.pending !== true;
}

export function hasMessageActions(msg: ChatActionRulesInput, isAdminOrOwner: boolean): boolean {
  return canEditMessage(msg) || canRecallMessage(msg) || canAdminRecallMessage(msg, isAdminOrOwner);
}
