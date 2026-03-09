/**
 * 消息同步管理器
 * 负责监听 WebSocket 新消息事件，触发消息列表刷新，管理未读数
 *
 * 职责：
 * 1. 订阅 PrivateMessageNew / GroupMessageNew 事件
 * 2. 通过防抖（debounce）合并短时间内多条消息为一次刷新通知
 * 3. 通过 GlobalStateManager.notifyMessageSync() 触发 MessagePage 刷新
 * 4. 维护总未读数 badge
 */

import { WebSocketService } from '../network/WebSocketService';
import { GlobalStateManager } from './GlobalStateManager';
import { hilog } from '@kit.PerformanceAnalysisKit';
import type { ServerEvent } from '../types/WebSocketTypes';
import type {
  PrivateMessageNewPayload,
  GroupMessageNewPayload,
} from '../types/WebSocketTypes';

const DOMAIN_NUMBER: number = 0xFF08;
const TAG: string = 'MessageSyncManager';

/** 防抖窗口（毫秒）：多条消息在此窗口内合并为一次刷新 */
const DEBOUNCE_MS: number = 500;

export class MessageSyncManager {
  private static instance: MessageSyncManager | null = null;

  private debounceTimer: number = -1;
  private initialized: boolean = false;

  /** 当前用户是否在聊天详情页（若在则不触发列表刷新，聊天页自行处理） */
  private activeChatId: string = '';

  // 保存绑定后的 handler 引用，用于 off 注销
  private privateMessageHandler: (event: ServerEvent) => void =
    (event: ServerEvent): void => this.onPrivateMessageNew(event);
  private groupMessageHandler: (event: ServerEvent) => void =
    (event: ServerEvent): void => this.onGroupMessageNew(event);

  private constructor() {}

  static getInstance(): MessageSyncManager {
    if (!MessageSyncManager.instance) {
      MessageSyncManager.instance = new MessageSyncManager();
    }
    return MessageSyncManager.instance;
  }

  /**
   * 初始化：订阅 WebSocket 事件
   * 应在 WebSocketService 连接之前或 GlobalStateManager.initialize 之后调用
   */
  initialize(): void {
    if (this.initialized) {
      return;
    }

    const ws = WebSocketService.getInstance();
    ws.on('PrivateMessageNew', this.privateMessageHandler);
    ws.on('GroupMessageNew', this.groupMessageHandler);

    this.initialized = true;
    hilog.info(DOMAIN_NUMBER, TAG, 'MessageSyncManager 初始化完成，已订阅消息事件');
  }

  /**
   * 销毁：注销 WebSocket 事件监听
   * 应在用户登出时调用
   */
  destroy(): void {
    if (!this.initialized) {
      return;
    }

    const ws = WebSocketService.getInstance();
    ws.off('PrivateMessageNew', this.privateMessageHandler);
    ws.off('GroupMessageNew', this.groupMessageHandler);

    this.clearDebounce();
    this.activeChatId = '';
    this.initialized = false;
    hilog.info(DOMAIN_NUMBER, TAG, 'MessageSyncManager 已销毁');
  }

  /**
   * 标记当前用户进入了某个聊天会话
   * 进入聊天详情页时调用，避免重复刷新消息列表
   */
  enterChat(chatId: string): void {
    this.activeChatId = chatId;
  }

  /**
   * 标记用户离开聊天会话
   * 离开聊天详情页时调用
   */
  leaveChat(): void {
    this.activeChatId = '';
  }

  // ============================================================================
  // WebSocket 事件处理
  // ============================================================================

  private onPrivateMessageNew(event: ServerEvent): void {
    const payload = event.payload as PrivateMessageNewPayload;
    const senderId = payload.sender?.user_id ?? '';

    hilog.info(DOMAIN_NUMBER, TAG, `收到私聊新消息: sender=${senderId}, message_id=${payload.message_id}`);

    // 如果当前正在查看该会话，跳过列表刷新（ChatPage 自行处理）
    if (this.activeChatId === payload.target_user_id || this.activeChatId === senderId) {
      hilog.debug(DOMAIN_NUMBER, TAG, '用户正在查看当前会话，跳过列表刷新');
      return;
    }

    this.scheduleSync();
  }

  private onGroupMessageNew(event: ServerEvent): void {
    const payload = event.payload as GroupMessageNewPayload;

    hilog.info(DOMAIN_NUMBER, TAG, `收到群聊新消息: group_id=${payload.group_id}, message_id=${payload.message_id}`);

    // 如果当前正在查看该群会话，跳过列表刷新
    if (this.activeChatId === payload.group_id) {
      hilog.debug(DOMAIN_NUMBER, TAG, '用户正在查看当前群聊，跳过列表刷新');
      return;
    }

    this.scheduleSync();
  }

  // ============================================================================
  // 防抖刷新
  // ============================================================================

  /**
   * 安排一次防抖刷新
   * 多条消息在 DEBOUNCE_MS 窗口内只触发一次 notifyMessageSync
   */
  private scheduleSync(): void {
    this.clearDebounce();
    this.debounceTimer = setTimeout((): void => {
      this.debounceTimer = -1;
      this.doSync();
    }, DEBOUNCE_MS);
  }

  private doSync(): void {
    hilog.info(DOMAIN_NUMBER, TAG, '触发消息列表刷新');
    GlobalStateManager.getInstance().notifyMessageSync();
  }

  private clearDebounce(): void {
    if (this.debounceTimer !== -1) {
      clearTimeout(this.debounceTimer);
      this.debounceTimer = -1;
    }
  }
}
