/**
 * 聊天详情页面
 * 重构：使用 List 组件优化消息列表，支持自动滚动
 * 优化：集成消息历史缓存 (RDB)
 */
// 聊天发送钮属已登记例外 — 已在 harmonyapp/AGENTS.md 组件原语例外清单登记，手写 Button('发送') 保留因 ChatTheme/getChatVisualStyle 定制样式与 isSendInteractive 动态交互（需主题派生背景/禁用态/按压缩放），未收敛为 SystemButton({variant:'primary', pill:true})

import { GlobalStateManager, KEY_CURRENT_USER } from '../../../core/state/GlobalStateManager';
import { ApiClient } from '../../../core/api/ApiClient';
import { formatDateLabel } from '../../../core/utils/FormatUtil';
import { WebSocketService } from '../../../core/network/WebSocketService';
import { ApiError } from '../../../core/network/ApiError';
import { MessageSyncManager } from '../../../core/state/MessageSyncManager';
import { AccountInfo, MessageType, PrivateMessageItem, GroupMessageItem, GroupMemberItem, SendPrivateMessageRequest, SendGroupMessageRequest, FetchPrivateMessagesRequest, FetchGroupMessagesRequest, MarkPrivateReadRequest, MarkGroupReadRequest, OpenUserDoorRequest, CreateReportRequest, MediaCategory } from '../../../core/types/api';
import type {
  ServerEvent,
  PrivateMessageNewEvent,
  PrivateMessageEditedEvent,
  PrivateMessageRecalledEvent,
  GroupMessageNewEvent,
  GroupMessageEditedEvent,
  GroupMessageRecalledEvent,
  PrivateTypingEvent,
  GroupTypingEvent
} from '../../../core/types/WebSocketTypes';
import { PageNames } from '../../../constants/AppConstants';
import { Pressable } from '../../../components/Pressable';
import { SystemAvatar } from '../../../components/SystemAvatar';
import { ChatPageHeader } from '../../../components/ChatPageHeader';
import { ChatMessageItem, ChatMessageView } from '../components/ChatMessageItem';
import { DatePill } from '../components/DatePill';
import { canRecallMessage, canAdminRecallMessage, canEditMessage, hasMessageActions } from '../components/ChatMessageRules';
import { mapMessageById } from '../components/ChatMessageListUtils';
import { buildGroupTypingHint } from '../components/ChatTypingRules';
import { resolveStableBottomInset, isNearBottom as isMessageListNearBottom } from '../components/ChatViewportRules';
import { ChatMessageListItem, dedupeMessages, mergeMessagesById, buildMessageFingerprint, buildMessageContentSignature, formatMessageTime, formatMutedUntil, getLocalDayKey, getLastSentMessageId, needsDateSeparator } from '../components/ChatMessageListUtils';
import type { ChatActionRulesInput } from '../components/ChatMessageRules';
import { ChatActionsSheet } from '../components/ChatActionsSheet';
import { MessageActionsSheet } from '../components/MessageActionsSheet';
import { ResponsiveContainer } from '../../../components/ResponsiveContainer';
import { SystemLoadingView } from '../../../components/SystemLoadingView';
import { EmptyState, LoadingMoreIndicator } from '../../../components/EmptyState';
import { SystemToast } from '../../../components/SystemToast';
import { SystemButton } from '../../../components/SystemButton';
import { SystemDialog } from '../../../components/SystemDialog';
import { DesignConstants, textLineHeight } from '../../../constants/DesignConstants';
import { ListEnd } from '../../../components/ListEnd';
import { AppIcon } from '../../../components/SystemIcon';
import { SoftDivider } from '../../../components/SoftDivider';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { MessageCacheService, MessageCacheItem } from '../../../core/database/MessageCacheService';
import { AuthAccountManager } from '../../../core/state/AuthAccountManager';
import { LoadingUtil } from '../../../core/utils/LoadingUtil';
import { BasicDataSource } from '../../../core/utils/BasicDataSource';
import { AppRouter, LoginRouteParam, GroupDetailRouteParam, ProfileRouteParam } from '../../../core/router/AppRouter';
import { SafeAreaUtils } from '../../../core/utils/SafeAreaUtils';
import { SystemUiController } from '../../../core/utils/SystemUiController';
import { KeyboardDismissManager } from '../../../core/utils/KeyboardDismissManager';
import { MediaUrlUtil } from '../../../core/utils/MediaUrlUtil';
import { MediaServiceManager } from '../../../core/media/MediaServiceManager';
import { common } from '@kit.AbilityKit';
import { curves } from '@kit.ArkUI';
import { KeyboardAvoidMode as UIContextKeyboardAvoidMode } from '@ohos.arkui.UIContext';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { AppThemeVisuals } from '../../../core/theme/AppThemeVisuals';
import { ChatThemeManager } from '../../../core/theme/ChatTheme';
import { DEFAULT_RESOLVED_THEME_ID, KEY_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';
import { LucideIcon } from '../../../components/LucideIcon';

const DOMAIN_NUMBER: number = 0xFF09;
const TAG: string = 'ChatPage';
const MESSAGE_CONTENT_MAX: number = 5000;
const TYPING_THROTTLE_MS: number = 3000;
const TYPING_INDICATOR_TTL_MS: number = 4500;
const RECONNECT_BACKFILL_THRESHOLD_MS: number = 10000;
const SEND_DUPLICATE_GUARD_WINDOW_MS: number = 1200;
const KEYBOARD_DISMISS_NAV_GUARD_MS: number = 260;
// 撤回时间窗（spec message_recall_window_seconds，默认 300s；服务端 10400 兜底，
const RECALLED_MESSAGE_TEXT: string = '消息已撤回';

// 统一消息接口用于UI展示（直接复用 ChatMessageView：extends 列表项最小集 + 共享扩展）
interface ChatMessage extends ChatMessageView {
}

/** 群聊 @ 提及候选（对齐前端 ChatInputBar 的 mentionCandidates） */
interface MentionCandidate {
  id: string;
  name: string;
}

interface ChatParams {
  chatId: string;
  chatType: 'private_chat' | 'group_chat';
  title: string;
  targetId?: string;
}

@Preview
@Component
export struct ChatPage {
  pathStack: NavPathStack = new NavPathStack();
  @StorageLink(KEY_CURRENT_USER) currentUser: AccountInfo | null = null;
  @StorageLink('safeAreaTop') safeAreaTop: number = 0;
  @StorageLink('safeAreaBottom') safeAreaBottom: number = 0;
  @StorageLink(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = DEFAULT_RESOLVED_THEME_ID;
  
  @State messages: ChatMessage[] = [];
  private messagesDataSource: BasicDataSource<ChatMessage> = new BasicDataSource<ChatMessage>([]);
  @State inputText: string = '';

  // @ 提及（对齐前端 s2-08：输入 `@` 弹成员浮层，发送时透传 mentions）
  @State private mentionCandidates: MentionCandidate[] = [];
  @State private mentionOpen: boolean = false;
  @State private mentionQuery: string = '';
  @State isLoading: boolean = true;
  /** loadHistory 在飞标志：与 UI 的 isLoading 分离——isLoading 初始即为 true（首屏
   * 加载态），不能当在飞判据，否则首次进入直接 return、聊天永远卡在「加载消息…」
   * （ff9ee4fb 引入的回归，模拟器实测 10-18） */
  private loadHistoryInFlight: boolean = false;
  @State errorMessage: string = '';
  @State isSending: boolean = false;
  @State isSendPressed: boolean = false;
  @State isInputFocused: boolean = false;
  @State isOtherUserTyping: boolean = false;
  @State groupTypingUserIds: string[] = [];
  @State title: string = '聊天';
  @State hasMoreHistory: boolean = false;
  @State historyNextCursor: string = '';
  @State isLoadingMoreHistory: boolean = false;
  @State showConversationActionsSheet: boolean = false;
  @State showReportConfirm: boolean = false;
  @State reportSubmitting: boolean = false;
  @State targetAvatar: string = '';
  @State targetPresenceText: string = '';
  @State targetPresenceOnline: boolean = false;
  /** 群聊页头数据（对齐前端 GroupChatView：头像 + 副文本「N 位成员」） */
  @State groupAvatarUrl: string = '';
  @State groupMembersCount: number = 0;
  @State isGroupServerMuted: boolean = false;
  @State groupMutedUntil: string = '';
  /** 单条消息操作面板（编辑/撤回，对齐前端 MessageBubble ⋮ 菜单） */
  @State showMessageActionsSheet: boolean = false;
  @State actionMessage: ChatMessage | null = null;

  private chatId: string = '';
  private chatType: 'private_chat' | 'group_chat' = 'private_chat';
  private targetId?: string;
  private webSocketService = WebSocketService.getInstance();
  private messageCacheService = MessageCacheService.getInstance();
  private scroller: Scroller = new Scroller();
  private firstVisibleIndex: number = 0;
  /** 上翻读历史时收到新消息：显示「↓新消息」悬浮钮（对齐审查 P1 聊天三件套） */
  @State showNewMessageBadge: boolean = false;
  private typingStopTimer: number = -1;
  private lastTypingSentAt: number = 0;
  private typingStateReported: boolean = false;
  private privateTypingResetTimer: number = -1;
  private groupTypingTimers: Map<string, number> = new Map();
  private wsHasConnectedOnce: boolean = false;
  private isReconnectReconcileInFlight: boolean = false;
  private scrollTimer: number = -1;
  private stableSafeAreaBottom: number = 0;
  private lastSendSignature: string = '';
  private lastSendAtMs: number = 0;
  private lastKeyboardDismissAtMs: number = 0;
  /** 当前用户在群聊中的角色（owner/admin 可撤回他人消息；私聊不使用） */
  private groupRole: string = 'member';

  aboutToAppear(): void {
    SystemUiController.syncWithCurrentTheme();

    this.stableSafeAreaBottom = Math.max(0, this.safeAreaBottom);
    this.applyKeyboardAvoidMode(UIContextKeyboardAvoidMode.RESIZE);
  }

  private getCurrentThemePalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  private getSurfaceBackground(): string {
    return this.getCurrentThemePalette().surfaceBackground;
  }

  private getSurfaceSubtle(): string {
    return this.getCurrentThemePalette().surfaceSubtle;
  }

  private getSurfaceBorder(): string {
    return this.getCurrentThemePalette().borderColor;
  }

  private getAccentBackground(): string {
    return this.getCurrentThemePalette().accentBackground;
  }

  private getChatTheme() {
    return ChatThemeManager.current();
  }

  private getChatVisualStyle() {
    return AppThemeVisuals.getChatVisualStyle(this.resolvedThemeId);
  }

  /** 底部安全区稳定性（键盘尖峰抑制）：规则实现在 ChatViewportRules */
  private getStableBottomInset(): number {
    const next = resolveStableBottomInset(this.safeAreaBottom, this.stableSafeAreaBottom);
    this.stableSafeAreaBottom = next;
    return next;
  }

  private getComposerBottomPadding(): number {
    const stableInset = this.getStableBottomInset();
    return SafeAreaUtils.resolveBottomPadding(stableInset, 0, DesignConstants.SPACING.SM);
  }

  /** 本地日期 key：用于判断两条消息是否跨天（对齐前端 groupMessagesByDay） */
  /** 该索引消息是否需要展示日期分隔胶囊（与上一条不同日 / 首条）——实现在 ChatMessageListUtils */
  private shouldShowDatePill(index: number): boolean {
    return needsDateSeparator(this.messages, index);
  }

  private dismissKeyboardOnOutsideTap(): void {
    if (!this.isInputFocused) {
      return;
    }
    KeyboardDismissManager.getInstance().dismiss();
    this.isInputFocused = false;
    this.lastKeyboardDismissAtMs = Date.now();
    this.stopTyping();
  }

  private handleBackNavigation(): boolean {
    if (this.isInputFocused) {
      this.dismissKeyboardOnOutsideTap();
      return true;
    }
    if (this.shouldConsumePostDismissNavigation()) {
      return true;
    }
    AppRouter.pop();
    return true;
  }

  private shouldConsumePostDismissNavigation(): boolean {
    if (this.lastKeyboardDismissAtMs <= 0) {
      return false;
    }
    return Date.now() - this.lastKeyboardDismissAtMs <= KEYBOARD_DISMISS_NAV_GUARD_MS;
  }

  private applyKeyboardAvoidMode(mode: UIContextKeyboardAvoidMode): void {
    try {
      this.getUIContext().setKeyboardAvoidMode(mode);
    } catch (error) {
      hilog.warn(DOMAIN_NUMBER, TAG, `设置键盘避让模式失败: ${JSON.stringify(error)}`);
    }
  }

  private onPrivateMessageNew = async (event: ServerEvent): Promise<void> => {
    if (event.type !== 'PrivateMessageNew') return;
    if (this.chatType !== 'private_chat') return;

    const payload = (event as PrivateMessageNewEvent).payload;
    const currentUserId = await AuthAccountManager.getCurrentUserId();
    const targetId = this.targetId;

    if (!currentUserId || !targetId) return;

    // PrivateMessageNewPayload uses sender (UserInfo)
    if (!payload.sender) return;
    const isRelevant = 
      (payload.sender.user_id === currentUserId && payload.target_user_id === targetId) ||
      (payload.sender.user_id === targetId && payload.target_user_id === currentUserId);
    
    if (!isRelevant) return;
    if (payload.sender.user_id === targetId) {
      this.resetPrivateTypingState();
    }

    const newMessage: ChatMessage = {
      id: payload.message_id,
      content: payload.media?.url || payload.content || (payload.type === 'image' ? '[图片]' : '[非文本消息]'),
      senderId: payload.sender.user_id,
      senderName: payload.sender.nickname ?? '',
      senderAvatar: payload.sender.avatar_url,
      type: payload.type as MessageType,
      sentAt: payload.sent_at,
      isSelf: payload.sender.user_id === currentUserId,
      isSystem: payload.type === 'system',
      rawData: payload
    };
    this.handleNewMessage(newMessage);
  }

  private onGroupMessageNew = async (event: ServerEvent): Promise<void> => {
    if (event.type !== 'GroupMessageNew') return;
    if (this.chatType !== 'group_chat') return;

    const payload = (event as GroupMessageNewEvent).payload;
    if (payload.group_id !== this.chatId) return;
    if (!payload.sender) return;
    const senderId = payload.sender.user_id ?? '';
    if (senderId.length > 0) {
      this.clearGroupTypingTimer(senderId);
      this.groupTypingUserIds = this.groupTypingUserIds.filter((id: string) => id !== senderId);
    }

    const currentUserId = await AuthAccountManager.getCurrentUserId();
    const newMessage: ChatMessage = {
      id: payload.message_id,
      content: payload.media?.url || payload.content || (payload.type === 'image' ? '[图片]' : '[非文本消息]'),
      senderId: payload.sender.user_id ?? '',
      senderName: payload.sender.nickname ?? '',
      senderAvatar: payload.sender.avatar_url,
      type: payload.type as MessageType,
      sentAt: payload.sent_at,
      isSelf: payload.sender.user_id === currentUserId,
      isSystem: payload.type === 'system',
      rawData: payload
    };
    this.handleNewMessage(newMessage);
  }

  private onPrivateTyping = (event: ServerEvent): void => {
    if (event.type !== 'PrivateTyping') return;
    if (this.chatType !== 'private_chat') return;

    const payload = (event as PrivateTypingEvent).payload;
    const targetId = this.targetId;
    if (!targetId) return;

    // 契约：payload 统一 actor: UserInfo，按 actor.user_id 匹配会话对端
    const actorUserId = payload.actor?.user_id ?? '';
    if (actorUserId !== targetId) return;

    if (payload.is_typing) {
      this.isOtherUserTyping = true;
      this.schedulePrivateTypingAutoClear();
      return;
    }
    this.resetPrivateTypingState();
  }

  private onGroupTyping = (event: ServerEvent): void => {
    if (event.type !== 'GroupTyping') return;
    if (this.chatType !== 'group_chat') return;

    const payload = (event as GroupTypingEvent).payload;
    if (payload.group_id !== this.chatId) return;
    const actorUserId = payload.actor?.user_id ?? '';
    if (actorUserId === (this.currentUser?.user_id ?? '')) return;

    if (payload.is_typing) {
      if (this.groupTypingUserIds.indexOf(actorUserId) === -1) {
        this.groupTypingUserIds = this.groupTypingUserIds.concat([actorUserId]);
      }
      this.scheduleGroupTypingAutoClear(actorUserId);
      return;
    }
    this.clearGroupTypingTimer(actorUserId);
    this.groupTypingUserIds = this.groupTypingUserIds.filter((id: string) => id !== actorUserId);
  }

  private onConnectionEstablished = (_event: ServerEvent): void => {
    if (!this.wsHasConnectedOnce) {
      this.wsHasConnectedOnce = true;
      return;
    }
    if (!this.shouldBackfillAfterReconnect()) {
      return;
    }
    this.reconcileRecentMessagesAfterReconnect();
  }

  private handleNewMessage(newMessage: ChatMessage) {
    const normalizedId = newMessage.id.trim();
    const newFingerprint = buildMessageFingerprint(newMessage);
    const duplicated = this.messages.some((item: ChatMessage) => {
      const itemId = item.id.trim();
      if (normalizedId.length > 0 && itemId === normalizedId) {
        return true;
      }
      return buildMessageFingerprint(item) === newFingerprint;
    });
    if (duplicated) {
      return;
    }

    // 乐观气泡确认：WS 回显与本地 pending 消息匹配（本人 + 同类型 + 同内容）时，
    // 用服务端消息替换 pending（保留真实 id），避免重复展示
    if (newMessage.isSelf) {
      const pendingIndex = this.messages.findIndex((m: ChatMessage) =>
        m.pending === true && m.content === newMessage.content && m.type === newMessage.type);
      if (pendingIndex >= 0) {
        this.messages[pendingIndex] = newMessage;
        // BasicDataSource 无 replaceData，用 setData 全量刷新
        this.messagesDataSource.setData(this.messages);
        this.scrollToBottom();
        this.messageCacheService.addMessage({
          id: newMessage.id,
          chatId: this.chatId,
          type: newMessage.type,
          senderId: newMessage.senderId,
          content: newMessage.content,
          sentAt: newMessage.sentAt,
          data: JSON.stringify(newMessage.rawData ?? null)
        });
        return;
      }
    }

    this.messages.push(newMessage);
    this.messagesDataSource.addData([newMessage]);

    if (!newMessage.isSelf) {
      // 仅贴底时自动滚动到底（上翻读历史不拽回底部）；否则显示「↓新消息」悬浮钮
      if (this.isNearBottom()) {
        this.scrollToBottom();
      } else {
        this.showNewMessageBadge = true;
      }
      this.markAsRead(newMessage.id);
    } else {
      // 自己的消息（含乐观气泡确认）总是滚动到底
      this.scrollToBottom();
    }

    this.messageCacheService.addMessage({
      id: newMessage.id,
      chatId: this.chatId,
      type: newMessage.type,
      senderId: newMessage.senderId,
      content: newMessage.content,
      sentAt: newMessage.sentAt,
      data: JSON.stringify(newMessage.rawData ?? null)
    });
  }

  aboutToDisappear() {
    this.applyKeyboardAvoidMode(UIContextKeyboardAvoidMode.OFFSET);
    this.isInputFocused = false;
    this.stopTyping();
    if (this.scrollTimer !== -1) {
      clearTimeout(this.scrollTimer);
      this.scrollTimer = -1;
    }
    MessageSyncManager.getInstance().leaveChat();
    if (this.chatType === 'private_chat' && this.targetId) {
      this.webSocketService.unsubscribeUsers([this.targetId]);
    }
    this.resetPrivateTypingState();
    this.resetGroupTypingState();
    this.unsubscribeFromEvents();
    if (this.chatType === 'group_chat' && this.chatId) {
      this.webSocketService.unsubscribeGroupChats([this.chatId]);
    }
  }

  private async loadHistory() {
    if (this.loadHistoryInFlight) return; // 在飞守卫：错误态重试按钮可与慢速首屏并发，避免双 setData + 双 scrollToBottom
    this.loadHistoryInFlight = true;
    this.isLoading = true;
    this.errorMessage = '';
    
    await this.loadFromCache();

    try {
      await LoadingUtil.withMinimumTime(async () => {
        let items: (PrivateMessageItem | GroupMessageItem)[] = [];
        let nextCursor: string | null = null;

        if (this.chatType === 'private_chat') {
          const targetUserId = this.targetId;
          if (!targetUserId) {
            throw new Error('target_user_id is required for private chat');
          }
          const response = await ApiClient.privateChatOpenTheDoor({ target_user_id: targetUserId });
          if (response.code === 0 && response.data) {
            items = response.data.recent_messages?.items ?? [];
            nextCursor = response.data.recent_messages?.next_cursor ?? null;
          }
        } else {
          const request: FetchGroupMessagesRequest = { group_id: this.chatId, pagination: { limit: 20 } };
          const response = await ApiClient.getGroupMessages(request);
          if (response.code === 0) {
            items = response.data?.items ?? [];
            nextCursor = response.data?.next_cursor ?? null;
          }

          // 获取群组详情中的 viewer_state，填充服务端禁言状态与角色（角色用于管理员撤回）
          try {
            const groupResp = await ApiClient.getGroupDetail({ group_id: this.chatId });
            if (groupResp.code === 0 && groupResp.data?.viewer_state) {
              this.isGroupServerMuted = groupResp.data.viewer_state.is_muted;
              this.groupMutedUntil = groupResp.data.viewer_state.muted_until ?? '';
              this.groupRole = groupResp.data.viewer_state.role;
            }
            // 群名/头像/成员数 → 页头（对齐前端 GroupChatView：副文本「N 位成员」，
            // 群名以详情接口为准，路由参数里的会话名可能滞后）
            if (groupResp.code === 0 && groupResp.data?.group) {
              const groupInfo = groupResp.data.group;
              if (groupInfo.name && groupInfo.name !== this.title) {
                this.title = groupInfo.name;
              }
              this.groupAvatarUrl = groupInfo.avatar_url ?? '';
              this.groupMembersCount = groupInfo.members_count ?? 0;
            }
            // @ 提及候选：群成员首屏（limit 100 = 分页上限）；失败静默——提及是增强，
            // 不应影响聊天主流程（对齐前端 GroupChatView 的 getGroupMembers 处理）
            try {
              const memResp = await ApiClient.getGroupMembers({
                group_id: this.chatId,
                pagination: { limit: 100 }
              });
              if (memResp.code === 0 && memResp.data?.items) {
                this.mentionCandidates = memResp.data.items.map((m: GroupMemberItem) => {
                  const cand: MentionCandidate = {
                    id: m.user.user_id,
                    name: m.user.nickname && m.user.nickname.length > 0 ? m.user.nickname : m.user.user_id
                  };
                  return cand;
                });
              }
            } catch (e) {
              hilog.warn(DOMAIN_NUMBER, TAG, `加载群成员（提及候选）失败: ${JSON.stringify(e)}`);
            }
          } catch (e) {
            hilog.warn(DOMAIN_NUMBER, TAG, `加载群组禁言状态失败: ${JSON.stringify(e)}`);
          }
        }

        const currentUserId = await AuthAccountManager.getCurrentUserId();
        const convertedMessages = await Promise.all(items.reverse().map(item => this.convertMessage(item, currentUserId)));
        const newMessages = dedupeMessages(convertedMessages);
        this.messages = newMessages;
        this.messagesDataSource.setData(newMessages);
        this.hasMoreHistory = !!(nextCursor);
        this.historyNextCursor = nextCursor ?? '';
        
        this.updateCache(newMessages);
        this.scrollToBottom();
      });
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `加载历史消息失败: ${error}`);
      if (this.messages.length === 0) {
        this.errorMessage = '加载消息失败';
      }
    } finally {
      this.loadHistoryInFlight = false;
      this.isLoading = false;
    }
  }

  private async loadMoreHistory(): Promise<void> {
    if (this.isLoadingMoreHistory || !this.hasMoreHistory) return;
    // 加载前记录锚点：首条可见 index（此时顶部无指示项，List index 与消息 index 一致）与当前条数
    const anchorIndex = this.firstVisibleIndex;
    this.isLoadingMoreHistory = true;
    try {
      let items: (PrivateMessageItem | GroupMessageItem)[] = [];
      let nextCursor: string | null = null;

      if (this.chatType === 'private_chat') {
        const targetUserId = this.targetId;
        if (!targetUserId) return;
        const request: FetchPrivateMessagesRequest = {
          target_user_id: targetUserId,
          pagination: { limit: 20, cursor: this.historyNextCursor }
        };
        const response = await ApiClient.getPrivateMessages(request);
        if (response.code === 0) {
          items = response.data?.items ?? [];
          nextCursor = response.data?.next_cursor ?? null;
        }
      } else {
        const request: FetchGroupMessagesRequest = {
          group_id: this.chatId,
          pagination: { limit: 20, cursor: this.historyNextCursor }
        };
        const response = await ApiClient.getGroupMessages(request);
        if (response.code === 0) {
          items = response.data?.items ?? [];
          nextCursor = response.data?.next_cursor ?? null;
        }
      }

      const currentUserId = await AuthAccountManager.getCurrentUserId();
      const olderMessages = await Promise.all(items.reverse().map(item => this.convertMessage(item, currentUserId)));
      // 合并基于 fetch 后的 this.messages 快照：fetch 期间 WS 新消息会追加到
      // 尾部，若用 fetch 前的 previousCount 计数会使 addedCount 多算、视口锚点
      // 错位跳变（修复：按合并前后差值取「顶部前插条数」）。
      const mergedMessages = dedupeMessages(olderMessages.concat(this.messages));
      const prependedCount = mergedMessages.length - this.messages.length;
      this.messages = mergedMessages;
      this.messagesDataSource.setData(mergedMessages);
      // spec：next_cursor 为 null 表示没有更多数据
      this.hasMoreHistory = !!(nextCursor);
      this.historyNextCursor = nextCursor ?? '';
      // setData 走 notifyDataReload 全量重建，按「旧首条可见 index + 前插条数」恢复视口锚点，避免跳变
      this.restoreHistoryAnchor(anchorIndex, prependedCount);
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `加载更早消息失败: ${error}`);
    } finally {
      this.isLoadingMoreHistory = false;
    }
  }

  private async loadFromCache() {
    const cachedMessages = await this.messageCacheService.getCachedMessages(this.chatId);
    if (cachedMessages.length > 0) {
      const cachedMapped = await Promise.all(cachedMessages.reverse().map(async item => {
        const rawData = JSON.parse(item.data) as Object;
        return await this.convertMessage(rawData as PrivateMessageItem | GroupMessageItem);
      }));
      const messages = dedupeMessages(cachedMapped);
      this.messages = messages;
      this.messagesDataSource.setData(messages);
      this.scrollToBottom();
      hilog.info(DOMAIN_NUMBER, TAG, `Loaded ${messages.length} messages from cache`);
    }
  }

  private async updateCache(messages: ChatMessage[]): Promise<void> {
    const cacheItems: MessageCacheItem[] = messages.map(msg => {
      const cacheItem: MessageCacheItem = {
        id: msg.id,
        chatId: this.chatId,
        type: msg.type,
        senderId: msg.senderId,
        content: msg.content,
        sentAt: msg.sentAt,
        data: JSON.stringify(msg.rawData ?? null)
      };
      return cacheItem;
    });
    await this.messageCacheService.cacheMessages(this.chatId, cacheItems);
  }

  // currentUserId 允许批量调用方一次解析后透传（Preferences 读取 + JSON.parse
  // 索引较重，逐条 await 会在批量转换时放大成 N 次重复 IO）。
  private async convertMessage(item: PrivateMessageItem | GroupMessageItem, currentUserId?: string | null): Promise<ChatMessage> {
    const uid = currentUserId ?? await AuthAccountManager.getCurrentUserId();
    
    let senderId: string;
    let senderName: string;
    let senderAvatar: string | undefined;
    let isSelf: boolean;
    
    // Both PrivateMessageItem and GroupMessageItem use sender: UserInfo
    const msgItem = item as PrivateMessageItem | GroupMessageItem;
    if (msgItem.sender) {
      senderId = msgItem.sender.user_id ?? '';
      senderName = msgItem.sender.nickname ?? '';
      senderAvatar = msgItem.sender.avatar_url;
      isSelf = senderId === uid;
    } else {
      senderId = '';
      senderName = '';
      senderAvatar = undefined;
      isSelf = false;
    }
    
    const messageType: MessageType = item.type as MessageType;
    const isRecalled = item.recalled_at !== null && item.recalled_at !== undefined;
    return {
      id: item.message_id,
      // 已撤回消息统一渲染占位文案（对齐前端：m.recalled_at ? '消息已撤回' : content）
      content: isRecalled ? RECALLED_MESSAGE_TEXT :
        (item.media?.url || item.content || (item.type === 'image' ? '[图片]' : '[非文本消息]')),
      senderId: senderId,
      senderName: senderName,
      senderAvatar: senderAvatar,
      type: messageType,
      sentAt: item.sent_at,
      isSelf: isSelf,
      isSystem: item.type === 'system',
      recalled: isRecalled,
      edited: item.edited_at !== null && item.edited_at !== undefined,
      rawData: item
    };
  }

  private shouldBackfillAfterReconnect(): boolean {
    const disconnectDurationMs = this.webSocketService.getLastDisconnectDurationMs();
    return disconnectDurationMs > RECONNECT_BACKFILL_THRESHOLD_MS;
  }

  private async reconcileRecentMessagesAfterReconnect(): Promise<void> {
    if (this.isReconnectReconcileInFlight || this.isLoading) {
      return;
    }
    this.isReconnectReconcileInFlight = true;
    try {
      let items: (PrivateMessageItem | GroupMessageItem)[] = [];
      if (this.chatType === 'private_chat') {
        const targetUserId = this.targetId;
        if (!targetUserId) {
          return;
        }
        const response = await ApiClient.getPrivateMessages({
          target_user_id: targetUserId,
          pagination: { limit: 20 }
        });
        if (response.code === 0) {
          items = response.data?.items ?? [];
        }
      } else {
        const response = await ApiClient.getGroupMessages({
          group_id: this.chatId,
          pagination: { limit: 20 }
        });
        if (response.code === 0) {
          items = response.data?.items ?? [];
        }
      }

      if (items.length === 0) {
        return;
      }
      const currentUserId = await AuthAccountManager.getCurrentUserId();
      const recentMessages = await Promise.all(items.reverse().map(item => this.convertMessage(item, currentUserId)));
      const mergedMessages = mergeMessagesById(this.messages, recentMessages);
      this.messages = mergedMessages;
      this.messagesDataSource.setData(mergedMessages);
    } catch (error) {
      hilog.warn(DOMAIN_NUMBER, TAG, `重连后消息回补失败: ${JSON.stringify(error)}`);
      SystemToast.show('部分消息可能未同步，下拉刷新', this.getUIContext());
    } finally {
      this.isReconnectReconcileInFlight = false;
    }
  }

  private subscribeToEvents() {
    this.webSocketService.on('PrivateMessageNew', this.onPrivateMessageNew);
    this.webSocketService.on('GroupMessageNew', this.onGroupMessageNew);
    this.webSocketService.on('PrivateMessageEdited', this.onPrivateMessageEdited);
    this.webSocketService.on('PrivateMessageRecalled', this.onPrivateMessageRecalled);
    this.webSocketService.on('GroupMessageEdited', this.onGroupMessageEdited);
    this.webSocketService.on('GroupMessageRecalled', this.onGroupMessageRecalled);
    this.webSocketService.on('PrivateTyping', this.onPrivateTyping);
    this.webSocketService.on('GroupTyping', this.onGroupTyping);
    this.webSocketService.on('ConnectionEstablished', this.onConnectionEstablished);
  }

  private unsubscribeFromEvents() {
    this.webSocketService.off('PrivateMessageNew', this.onPrivateMessageNew);
    this.webSocketService.off('GroupMessageNew', this.onGroupMessageNew);
    this.webSocketService.off('PrivateMessageEdited', this.onPrivateMessageEdited);
    this.webSocketService.off('PrivateMessageRecalled', this.onPrivateMessageRecalled);
    this.webSocketService.off('GroupMessageEdited', this.onGroupMessageEdited);
    this.webSocketService.off('GroupMessageRecalled', this.onGroupMessageRecalled);
    this.webSocketService.off('PrivateTyping', this.onPrivateTyping);
    this.webSocketService.off('GroupTyping', this.onGroupTyping);
    this.webSocketService.off('ConnectionEstablished', this.onConnectionEstablished);
  }

  private scrollToBottom() {
    if (this.scrollTimer !== -1) clearTimeout(this.scrollTimer);
    this.scrollTimer = setTimeout(() => {
      this.scrollTimer = -1;
      this.scroller.scrollEdge(Edge.Bottom);
    }, 100);
  }

  /** 是否贴底（末尾 3 条内视为贴底）：上翻读历史时收新消息不拽回底部 */
  /** 是否接近底部：规则实现在 ChatViewportRules（末尾 3 条以内或空列表） */
  private isNearBottom(): boolean {
    return isMessageListNearBottom(this.messages.length, this.firstVisibleIndex);
  }

  /** 悬浮钮点击：滚动到底并清除标记 */
  private jumpToBottom(): void {
    this.showNewMessageBadge = false;
    this.scrollToBottom();
  }

  /** 加载更早历史后恢复视口锚点：等 reload 布局与顶部指示项移除完成后，定位到「旧 index + 新增条数」 */
  private restoreHistoryAnchor(anchorIndex: number, addedCount: number): void {
    if (addedCount <= 0) return;
    if (this.scrollTimer !== -1) clearTimeout(this.scrollTimer);
    this.scrollTimer = setTimeout(() => {
      this.scrollTimer = -1;
      this.scroller.scrollToIndex(anchorIndex + addedCount, false);
    }, 100);
  }



  private openConversationActions(): void {
    if (this.isInputFocused) {
      this.dismissKeyboardOnOutsideTap();
      return;
    }
    if (this.shouldConsumePostDismissNavigation()) {
      return;
    }

    if (this.chatType === 'group_chat') {
      const groupParam = new GroupDetailRouteParam();
      groupParam.groupId = this.chatId;
      AppRouter.push(PageNames.GROUP_DETAIL, groupParam);
      return;
    }

    this.showConversationActionsSheet = true;
  }

  /** 发送钮可交互条件（背景/字色/透明度/缩放/enabled 统一入口） */
  private isSendInteractive(): boolean {
    return !this.isSending && this.inputText.trim().length > 0 && !this.isGroupServerMuted;
  }

  /** 当前匹配的提及候选（最多 6 条，对齐前端 MENTION_MAX） */
  private filteredMentionCandidates(): MentionCandidate[] {
    const q: string = this.mentionQuery.toLowerCase();
    const out: MentionCandidate[] = [];
    for (const c of this.mentionCandidates) {
      if (c.name.toLowerCase().includes(q) || c.id.toLowerCase().includes(q)) {
        out.push(c);
        if (out.length >= 6) {
          break;
        }
      }
    }
    return out;
  }

  /** 输入变化时同步 `@查询` 状态（按整串尾部判定，与前端的光标兜底口径一致） */
  private syncMention(value: string): void {
    if (this.mentionCandidates.length === 0) {
      this.mentionOpen = false;
      return;
    }
    const m = /(?:^|\s)@([^\s@]*)$/.exec(value);
    if (m) {
      this.mentionQuery = m[1];
      this.mentionOpen = this.filteredMentionCandidates().length > 0;
    } else {
      this.mentionQuery = '';
      this.mentionOpen = false;
    }
  }

  /** 选中成员：把尾部 `@查询` 替换为 `@昵称 `（对齐前端 applyMention） */
  private applyMention(name: string): void {
    this.inputText = this.inputText.replace(/(^|\s)@([^\s@]*)$/, `$1@${name} `);
    this.mentionQuery = '';
    this.mentionOpen = false;
  }

  /** 发送时把文本里仍存在的 @昵称 映射回 user_id（去重、上限 20，spec mentions 上限） */
  private collectMentions(text: string): string[] {
    const ids: string[] = [];
    for (const c of this.mentionCandidates) {
      if (text.includes(`@${c.name}`) && !ids.includes(c.id)) {
        ids.push(c.id);
      }
    }
    return ids.slice(0, 20);
  }

  @Builder
  mentionPanel() {
    Column() {
      ForEach(this.filteredMentionCandidates(), (c: MentionCandidate) => {
        Pressable({
          onPress: () => {
            this.applyMention(c.name);
          },
          hoverFill: this.getCurrentThemePalette().surfaceSubtle
        }) {
          Row({ space: DesignConstants.SPACING.SM }) {
            Text(c.name)
              .fontSize(DesignConstants.FONT.SIZE.MD)
              .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.MD))
              .fontColor(this.getCurrentThemePalette().textPrimary)
              .layoutWeight(1)
              .maxLines(1)
              .textOverflow({ overflow: TextOverflow.Ellipsis })
            Text(c.id)
              .fontSize(DesignConstants.FONT.SIZE.XS2)
              .fontColor(this.getCurrentThemePalette().textTertiary)
          }
          .width('100%')
          .padding({ left: DesignConstants.SPACING.MD, right: DesignConstants.SPACING.MD, top: DesignConstants.SPACING.SM, bottom: DesignConstants.SPACING.SM })
        }
      }, (c: MentionCandidate) => c.id)
    }
    .width('100%')
    .backgroundColor(this.getCurrentThemePalette().surfaceBackground)
    .borderRadius(DesignConstants.RADIUS.MD)
    .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getCurrentThemePalette().borderColor })
    .clip(true)
    .margin({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, bottom: DesignConstants.SPACING.XS })
  }

  private async sendMessage() {
    if (this.isSending) {
      return;
    }
    if (!this.currentUser) {
      SystemToast.showWarning('请先登录后再发送消息', this.getUIContext());
      const loginParams = new LoginRouteParam();
      loginParams.reason = 'action_required';
      loginParams.action = '发送消息';
      AppRouter.push(PageNames.LOGIN, loginParams);
      return;
    }

    const content = this.inputText.trim();
    if (!content) return;
    if (content.length > MESSAGE_CONTENT_MAX) {
      SystemToast.showError(`消息内容最多 ${MESSAGE_CONTENT_MAX} 字符`, this.getUIContext());
      return;
    }

    const sendSignature = `${this.chatType}|${this.chatId}|${this.targetId ?? ''}|${content}`;
    const nowMs = Date.now();
    if (this.lastSendSignature === sendSignature &&
      nowMs - this.lastSendAtMs <= SEND_DUPLICATE_GUARD_WINDOW_MS) {
      hilog.warn(DOMAIN_NUMBER, TAG, '检测到短时间重复发送，已拦截');
      return;
    }
    this.lastSendSignature = sendSignature;
    this.lastSendAtMs = nowMs;

    this.isSending = true;
    try {
      if (this.chatType === 'private_chat') {
        const targetUserId = this.targetId;
        if (!targetUserId) {
          SystemToast.showError('无法获取目标用户ID', this.getUIContext());
          return;
        }
        const request: SendPrivateMessageRequest = {
          target_user_id: targetUserId,
          type: 'text',
          content: content
        };
        await ApiClient.sendPrivateMessage(request);
      } else {
        // @ 提及：文本里仍存在的 @昵称 映射回 user_id（对齐前端与 spec
        // api_spec_chat.md:135 —— 仅非空时下发 mentions）
        const mentions: string[] = this.collectMentions(content);
        const request: SendGroupMessageRequest = {
          group_id: this.chatId,
          type: 'text',
          content: content
        };
        if (mentions.length > 0) {
          request.mentions = mentions;
        }
        await ApiClient.sendGroupMessage(request);
      }

      // 乐观气泡：API 确认后本地立即上屏（pending 标记），WS 回显时由
      // handleNewMessage 替换为服务端消息；弱网下不再「发了没影」
      const pendingMessage: ChatMessage = {
        id: `pending-${Date.now()}`,
        content: content,
        senderId: this.currentUser.user_id,
        senderName: this.currentUser.nickname ?? '',
        senderAvatar: this.currentUser.avatar_url ?? '',
        type: 'text',
        sentAt: new Date().toISOString(),
        isSelf: true,
        isSystem: false,
        pending: true
      };
      this.messages.push(pendingMessage);
      this.messagesDataSource.addData([pendingMessage]);
      this.scrollToBottom();

      this.inputText = '';
    } catch (error) {
      this.lastSendSignature = '';
      this.lastSendAtMs = 0;
      // 契约 errors.md：send 返回 10200 时 data = { forbidden_reason: ForbiddenReason }（blocked_by_user/privacy_setting/stranger_quota_exceeded）
      const reason = ApiError.forbiddenReasonOf(error as Object);
      let failMessage = '发送失败';
      if (reason === 'blocked_by_user') {
        failMessage = '你已被对方拉黑';
      } else if (reason === 'privacy_setting') {
        failMessage = '对方隐私设置不允许发送消息';
      } else if (reason === 'stranger_quota_exceeded') {
        failMessage = '今日陌生人消息额度已用完';
      }
      SystemToast.showError(failMessage, this.getUIContext());
    } finally {
      this.stopTyping();
      this.isSending = false;
    }
  }

  private handleTypingInput(value: string): void {
    const isTyping = value.trim().length > 0;
    if (!isTyping) {
      this.stopTyping();
      return;
    }

    const now = Date.now();
    if (now - this.lastTypingSentAt >= TYPING_THROTTLE_MS) {
      this.sendTypingState(true);
      this.lastTypingSentAt = now;
    }

    // 防抖：停止输入 3 秒后自动发送 is_typing=false
    if (this.typingStopTimer !== -1) {
      clearTimeout(this.typingStopTimer);
      this.typingStopTimer = -1;
    }
    this.typingStopTimer = setTimeout(() => {
      this.stopTyping();
    }, TYPING_THROTTLE_MS);
  }

  private sendTypingState(isTyping: boolean): void {
    if (this.typingStateReported === isTyping) {
      return;
    }
    if (this.chatType === 'private_chat' && this.targetId) {
      this.webSocketService.sendPrivateTyping(this.targetId, isTyping);
      this.typingStateReported = isTyping;
      return;
    }
    if (this.chatType === 'group_chat' && this.chatId) {
      this.webSocketService.sendGroupTyping(this.chatId, isTyping);
      this.typingStateReported = isTyping;
      return;
    }
    if (!isTyping) {
      this.typingStateReported = false;
    }
  }

  private stopTyping(): void {
    if (this.typingStopTimer !== -1) {
      clearTimeout(this.typingStopTimer);
      this.typingStopTimer = -1;
    }
    this.sendTypingState(false);
    this.lastTypingSentAt = 0;
  }

  private schedulePrivateTypingAutoClear(): void {
    if (this.privateTypingResetTimer !== -1) {
      clearTimeout(this.privateTypingResetTimer);
      this.privateTypingResetTimer = -1;
    }
    this.privateTypingResetTimer = setTimeout(() => {
      this.privateTypingResetTimer = -1;
      this.isOtherUserTyping = false;
    }, TYPING_INDICATOR_TTL_MS);
  }

  private resetPrivateTypingState(): void {
    if (this.privateTypingResetTimer !== -1) {
      clearTimeout(this.privateTypingResetTimer);
      this.privateTypingResetTimer = -1;
    }
    this.isOtherUserTyping = false;
  }

  private clearGroupTypingTimer(userId: string): void {
    const timer = this.groupTypingTimers.get(userId);
    if (timer === undefined) {
      return;
    }
    clearTimeout(timer);
    this.groupTypingTimers.delete(userId);
  }

  private scheduleGroupTypingAutoClear(userId: string): void {
    this.clearGroupTypingTimer(userId);
    const timer = setTimeout(() => {
      this.groupTypingTimers.delete(userId);
      this.groupTypingUserIds = this.groupTypingUserIds.filter((id: string) => id !== userId);
    }, TYPING_INDICATOR_TTL_MS);
    this.groupTypingTimers.set(userId, timer);
  }

  private resetGroupTypingState(): void {
    this.groupTypingTimers.forEach((timer: number, _userId: string) => {
      clearTimeout(timer);
    });
    this.groupTypingTimers.clear();
    this.groupTypingUserIds = [];
  }

  /** 群聊「正在输入」提示：规则实现在 ChatTypingRules（纯函数，入参为成员列表与消息列表） */
  private groupTypingHintText(): string {
    return buildGroupTypingHint(this.groupTypingUserIds, this.messages);
  }

  private markAsRead(upToMessageId: string): void {
    if (!upToMessageId) return;

    (async () => {
      try {
        if (this.chatType === 'private_chat') {
          const targetUserId = this.targetId;
          if (!targetUserId) return;
          const request: MarkPrivateReadRequest = { scope: "session", target_user_id: targetUserId, up_to_message_id: upToMessageId };
          await ApiClient.markPrivateMessagesRead(request);
        } else {
          const request: MarkGroupReadRequest = { scope: "group", group_id: this.chatId, up_to_message_id: upToMessageId };
          await ApiClient.markGroupMessagesRead(request);
        }
        GlobalStateManager.getInstance().notifyMessageSync();
      } catch (e) {
        // 已读上报 best-effort：失败不阻塞用户，但留痕便于排障（角标不清零类反馈）
        hilog.warn(DOMAIN_NUMBER, TAG, `已读上报失败: ${JSON.stringify(e)}`);
      }
    })();
  }

  /** 从 onReady 参数初始化后，异步加载聊天数据 */
  private async initChatData(): Promise<void> {
    // 通知 MessageSyncManager 当前正在查看此会话，避免重复刷新消息列表
    const activeChatId = this.chatType === 'private_chat' ? (this.targetId ?? this.chatId) : this.chatId;
    MessageSyncManager.getInstance().enterChat(activeChatId);

    await this.loadHistory();
    this.subscribeToEvents();
    this.wsHasConnectedOnce = this.webSocketService.isConnected();

    // spec: PrivateTyping 依赖 user 订阅
    if (this.chatType === 'private_chat' && this.targetId) {
      this.webSocketService.subscribeUsers([this.targetId]);
    }

    // spec: GroupMessageNew 需要先订阅对应群聊才能收到推送
    if (this.chatType === 'group_chat' && this.chatId) {
      this.webSocketService.subscribeGroupChats([this.chatId]);
    }

    const last = this.messages.length > 0 ? this.messages[this.messages.length - 1] : undefined;
    if (last) {
      this.markAsRead(last.id);
    }

    if (this.chatType === 'private_chat' && this.targetId) {
      this.loadTargetProfile();
    }
  }

  private async loadTargetProfile(): Promise<void> {
    if (!this.targetId) return;
    try {
      const resp = await ApiClient.getUserProfile({ user_id: this.targetId } as OpenUserDoorRequest);
      if (resp.code === 0 && resp.data) {
        this.targetAvatar = resp.data.user.user.avatar_url ?? '';
      }
    } catch (e) {
      hilog.warn(DOMAIN_NUMBER, TAG, `加载目标用户头像失败: ${e}`);
    }
  }

  build() {
    NavDestination() {
    Stack() {
    ResponsiveContainer({ fill: true }) {
    Column() {
      // 头部 —— 统一走 ChatPageHeader（浮动卡片 + 返回 + 头像 + 标题 + 信息操作；
      // 群聊对齐前端 GroupChatView：群头像圆角方 + 「N 位成员」副文本 + 头部点击进群详情）
      ChatPageHeader({
        title: this.title,
        showAvatar: this.chatType === 'private_chat',
        targetAvatar: this.targetAvatar ?? '',
        presenceText: this.chatType === 'private_chat' ? this.targetPresenceText : '',
        presenceOnline: this.targetPresenceOnline,
        isGroup: this.chatType === 'group_chat',
        groupAvatar: this.groupAvatarUrl,
        membersCount: this.groupMembersCount,
        typingHint: this.groupTypingHintText(),
        showTrailing: this.chatType !== 'group_chat',
        onBack: () => this.handleBackNavigation(),
        onTrailingAction: () => this.openConversationActions(),
        onOpenGroupDetail: () => {
          const groupParam = new GroupDetailRouteParam();
          groupParam.groupId = this.chatId;
          AppRouter.push(PageNames.GROUP_DETAIL, groupParam);
        }
      })

      if (this.isLoading) {
        SystemLoadingView({ message: '加载消息…' })
          .layoutWeight(1)
          .onTouch((event: TouchEvent) => {
            if (event.type === TouchType.Down) {
              this.dismissKeyboardOnOutsideTap();
            }
          })
          .transition(
            TransitionEffect.OPACITY.animation({ duration: DesignConstants.ANIMATION.DURATION_SHEET, curve: DesignConstants.ANIMATION.CURVE_FAST_OUT_SLOW_IN })
              .combine(TransitionEffect.move(TransitionEdge.BOTTOM).animation({ duration: DesignConstants.ANIMATION.DURATION_SHEET, curve: DesignConstants.ANIMATION.CURVE_FAST_OUT_SLOW_IN }))
          )
      } else if (this.messages.length === 0 && this.errorMessage) {
        EmptyState({
          bare: true,
          icon: AppIcon.WARNING,
          message: this.errorMessage,
          actionLabel: '重试',
          fillMode: true,
          onAction: () => { this.loadHistory(); }
        })
          .layoutWeight(1)
          .onTouch((event: TouchEvent) => {
            if (event.type === TouchType.Down) {
              this.dismissKeyboardOnOutsideTap();
            }
          })
          .transition(
            TransitionEffect.OPACITY.animation({ duration: DesignConstants.ANIMATION.DURATION_SHEET, curve: DesignConstants.ANIMATION.CURVE_FAST_OUT_SLOW_IN })
              .combine(TransitionEffect.move(TransitionEdge.BOTTOM).animation({ duration: DesignConstants.ANIMATION.DURATION_SHEET, curve: DesignConstants.ANIMATION.CURVE_FAST_OUT_SLOW_IN }))
          )
      } else if (this.messages.length === 0) {
        EmptyState({
          bare: true,
          icon: AppIcon.MESSAGE,
          title: '暂无消息',
          message: '发送一条消息开始聊天吧',
          fillMode: true
        })
          .layoutWeight(1)
          .onTouch((event: TouchEvent) => {
            if (event.type === TouchType.Down) {
              this.dismissKeyboardOnOutsideTap();
            }
          })
      } else {
      List({ scroller: this.scroller, initialIndex: this.messages.length - 1 + (this.hasMoreHistory ? 0 : 1) }) {
        // 列表头部：没有更早历史时展示（语义=顶端收尾，原误置列表尾部）
        if (!this.hasMoreHistory) {
          ListItem() {
                        ListEnd()
          }
        }
        // 顶部加载更多历史指示器（向上分页）
        // 复用 LoadingMoreIndicator 原语（它本就是「加载中 + 文案」的居中胶囊），
        // 此前是逐字重写：Row + LoadingProgress + Text + 居中 + padding MD，
        // 与向下分页的实现不一致（同角色两种写法）。
        if (this.isLoadingMoreHistory) {
          ListItem() {
            LoadingMoreIndicator({ message: '加载更早消息…' })
          }
        }
            LazyForEach(this.messagesDataSource, (msg: ChatMessage, index: number) => {
              ListItem() {
                Column() {
                  if (this.shouldShowDatePill(index)) {
                    this.buildDatePill(formatDateLabel(msg.sentAt))
                  }
                  ChatMessageItem({
                    msg: msg,
                    isGroupChat: this.chatType === 'group_chat',
                    isLastSent: msg.isSelf && msg.id === getLastSentMessageId(this.messages),
                    timeText: formatMessageTime(msg.sentAt),
                    hasActions: hasMessageActions(msg, this.isAdminRecallContext()),
                    onMessageClick: (view: ChatMessageView) => { this.openMessageActions(view); },
                    palette: this.getCurrentThemePalette(),
                    chatTheme: this.getChatTheme()
                  })
                }
                .width('100%')
              }
            }, (msg: ChatMessage) => `${msg.id}_${msg.content}_${msg.edited ? 1 : 0}_${msg.recalled ? 1 : 0}_${msg.pending ? 1 : 0}`)

      }
      .scrollBar(BarState.Off)
      .width('100%')
      .layoutWeight(1)
      .cachedCount(8)
      .padding({
        left: DesignConstants.SPACING.LG,
        right: DesignConstants.SPACING.LG,
        top: DesignConstants.SPACING.MD,
        bottom: DesignConstants.SPACING.MD
      })
      .backgroundColor(this.getChatTheme().listBackground)
      .borderRadius(this.getChatVisualStyle().listRadius)
      .onScrollIndex((start: number) => {
        this.firstVisibleIndex = start;
        // 当滚动到列表顶部附近时加载更早的消息
        if (start <= 1 && this.hasMoreHistory && !this.isLoadingMoreHistory) {
          this.loadMoreHistory();
        }
        // 贴底（接近末尾）时清除「↓新消息」悬浮钮
        if (this.isNearBottom()) {
          this.showNewMessageBadge = false;
        }
      })
      .onTouch((event: TouchEvent) => {
        if (event.type === TouchType.Down) {
          this.dismissKeyboardOnOutsideTap();
        }
      })
      .transition(
        TransitionEffect.OPACITY.animation({ duration: DesignConstants.ANIMATION.DURATION_SHEET, curve: DesignConstants.ANIMATION.CURVE_FAST_OUT_SLOW_IN })
          .combine(TransitionEffect.move(TransitionEdge.BOTTOM).animation({ duration: DesignConstants.ANIMATION.DURATION_SHEET, curve: DesignConstants.ANIMATION.CURVE_FAST_OUT_SLOW_IN }))
      )
      }

      if (this.chatType === 'private_chat' && this.isOtherUserTyping) {
        Row() {
          Text('对方正在输入…')
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.SM))
            .fontColor(this.getCurrentThemePalette().textSecondary)
        }
        .width('100%')
        .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, top: DesignConstants.SPACING.XS })
        .onTouch((event: TouchEvent) => {
          if (event.type === TouchType.Down) {
            this.dismissKeyboardOnOutsideTap();
          }
        })
      }
      if (this.chatType === 'group_chat' && this.groupTypingUserIds.length > 0) {
        Row() {
          Text(this.groupTypingHintText())
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.SM))
            .fontColor(this.getCurrentThemePalette().textSecondary)
        }
        .width('100%')
        .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, top: DesignConstants.SPACING.XS })
        .onTouch((event: TouchEvent) => {
          if (event.type === TouchType.Down) {
            this.dismissKeyboardOnOutsideTap();
          }
        })
      }

      // 服务端禁言横幅
      if (this.isGroupServerMuted) {
        Row({ space: DesignConstants.SPACING.XS }) {
          LucideIcon({
              icon: AppIcon.WARNING,
              iconSize: DesignConstants.COMPONENT.ICON_SIZE_2XS,
              color: this.getCurrentThemePalette().error
            })
          Text(this.groupMutedUntil ? `你已被禁言，解除时间：${formatMutedUntil(this.groupMutedUntil)}` : '你已被禁言')
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.SM))
            .fontColor(this.getCurrentThemePalette().error)
        }
        .width('100%')
        .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, top: DesignConstants.SPACING.XS, bottom: DesignConstants.SPACING.XS })
        .justifyContent(FlexAlign.Center)
      }

      // @ 提及浮层（对齐前端 ChatInputBar：输入 `@` 触发，选中插入 `@昵称 `）
      if (this.mentionOpen) {
        this.mentionPanel()
      }

      // 对齐前端 ChatInputBar：仅输入框 + 发送钮，无「添加图片」钮
      // （web 私聊/群聊输入栏均无图片入口；图片消息为 app 单端功能，git 历史可恢复）
      Row() {
        TextInput({ text: this.inputText, placeholder: '发送消息…' })
          .layoutWeight(1)
          .maxLength(MESSAGE_CONTENT_MAX)
          .height(this.getChatVisualStyle().input.height)
          .backgroundColor(this.getChatTheme().inputFieldBg)
          .placeholderColor(this.getCurrentThemePalette().textTertiary)
          .borderRadius(this.getChatVisualStyle().input.borderRadius)
          .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: AppThemeVisuals.resolveInputBorder(this.getChatVisualStyle().input, this.isInputFocused, false, this.resolvedThemeId).borderColor })
          .shadow(AppThemeVisuals.resolveInputBorder(this.getChatVisualStyle().input, this.isInputFocused, false, this.resolvedThemeId).shadow)
          .padding({ left: this.getChatVisualStyle().input.horizontalPadding, right: this.getChatVisualStyle().input.horizontalPadding })
          .onChange((value) => {
            this.inputText = value;
            this.syncMention(value);
            this.handleTypingInput(value);
          })
          .onFocus(() => {
            this.isInputFocused = true;
          })
          .onBlur(() => {
            this.isInputFocused = false;
            this.stopTyping();
            KeyboardDismissManager.getInstance().dismiss();
          })
          .onSubmit(() => {
            this.sendMessage();
          })

        // 对齐前端：圆形发送钮（纸飞机 icon）；交互态 accent 底，非交互灰底；
        // 发送中旋转指示（LoadingProgress 替代静态时钟图标）
        Button() {
          if (this.isSending) {
            LoadingProgress()
              .width(DesignConstants.COMPONENT.LOADING_ICON_SIZE)
              .height(DesignConstants.COMPONENT.LOADING_ICON_SIZE)
              .color(this.getChatTheme().sendButtonActiveText)
          } else {
            LucideIcon({
              icon: AppIcon.SEND,
              iconSize: DesignConstants.COMPONENT.ICON_SIZE_XS,
              color: this.isSendInteractive() ? this.getChatTheme().sendButtonActiveText : this.getCurrentThemePalette().textSecondary
            })
          }
        }
        .type(ButtonType.Circle)
        // 纯图标钮：无障碍文案随发送态变化（skill mobile-ui-conventions 要求
        // 动态状态文案随之变化，而不是固定一句）
        .accessibilityText(this.isSending ? '发送中' : '发送消息')
        .stateEffect(false)
        .width(this.getChatVisualStyle().sendButton.height)
        .height(this.getChatVisualStyle().sendButton.height)
        // 发送钮热区扩到 44（视觉 36 不变，对齐同排「+」钮 MIN_HIT_TARGET 契约）
        .responseRegion({
          x: -(DesignConstants.LAYOUT.MIN_HIT_TARGET - this.getChatVisualStyle().sendButton.height) / 2,
          y: -(DesignConstants.LAYOUT.MIN_HIT_TARGET - this.getChatVisualStyle().sendButton.height) / 2,
          width: DesignConstants.LAYOUT.MIN_HIT_TARGET,
          height: DesignConstants.LAYOUT.MIN_HIT_TARGET
        })
        .backgroundColor(this.isSendInteractive() ? this.getChatTheme().sendButtonActive : this.getChatTheme().sendButtonDisabled)
        .shadow(this.getChatVisualStyle().sendButton.shadow)
        .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getChatTheme().inputBorder })
        .margin({ left: DesignConstants.SPACING.MD })
          .opacity(this.isSendInteractive() ? 1 : DesignConstants.OPACITY.DISABLED)
          .scale(this.isSendPressed && this.isSendInteractive() ?
            { x: DesignConstants.INTERACTION.SCALE_PRESSED_BUTTON, y: DesignConstants.INTERACTION.SCALE_PRESSED_BUTTON } :
            { x: 1, y: 1 })
          .animation({
            curve: curves.springMotion(
              DesignConstants.INTERACTION.SPRING_RESPONSE_PRESS,
              DesignConstants.INTERACTION.SPRING_DAMPING_PRESS
            )
          })
          .enabled(this.isSendInteractive())
          .onTouch((event: TouchEvent) => {
            if (!this.isSendInteractive()) {
              return;
            }
            if (event.type === TouchType.Down) {
              this.isSendPressed = true;
            } else if (event.type === TouchType.Up || event.type === TouchType.Cancel) {
              this.isSendPressed = false;
            }
          })
          .onClick(() => this.sendMessage())
      }
      .width('100%')
      .padding({
        left: DesignConstants.SPACING.LG,
        right: DesignConstants.SPACING.LG,
        top: DesignConstants.SPACING.SM,
        bottom: this.getComposerBottomPadding()
      })
      .backgroundColor(this.getChatTheme().inputBarBg)
      .shadow(this.getCurrentThemePalette().elevatedShadow)
      .border({ width: { top: DesignConstants.BORDER_WIDTH.HAIRLINE }, color: this.getChatTheme().inputBorder })
      .borderRadius({
        topLeft: this.getChatVisualStyle().composerRadius,
        topRight: this.getChatVisualStyle().composerRadius
      })

      // 焦点落点：用于将 TextInput 失焦，从而收起软键盘
      Button('')
        .id('chatKeyboardDismissTarget')
        .width(1)
        .height(1)
        .opacity(0)
        .focusable(true)
        .onClick(() => {})
    }
    .width('100%')
    .height('100%')
    .backgroundColor(this.getCurrentThemePalette().chatBackground)
    .bindSheet($$this.showConversationActionsSheet, this.conversationActionsSheetBuilder(), {
      height: SheetSize.FIT_CONTENT,
      showClose: true,
      dragBar: true,
      backgroundColor: this.getCurrentThemePalette().sheetBackground
    })
    .bindSheet($$this.showMessageActionsSheet, this.messageActionsSheetBuilder(), {
      height: SheetSize.FIT_CONTENT,
      showClose: true,
      dragBar: true,
      backgroundColor: this.getCurrentThemePalette().sheetBackground
    })
    }

    // 举报二次确认
    SystemDialog({
      visible: this.showReportConfirm,
      title: '举报',
      description: '将对该用户及相关聊天内容提交举报，我们会尽快审核处理',
      confirmText: '提交举报',
      confirmVariant: 'danger',
      onCancel: () => { this.showReportConfirm = false; },
      onConfirm: () => {
        this.showReportConfirm = false;
        this.submitReport();
      }
    })

    // 「↓新消息」悬浮钮（外层 Stack child，position 定位右下；上翻读历史时收到新消息才显示）
    if (this.showNewMessageBadge) {
      Row({ space: DesignConstants.SPACING.XS }) {
        LucideIcon({
          icon: AppIcon.ARROW_DOWN,
          iconSize: DesignConstants.COMPONENT.ICON_SIZE_2XS,
          color: this.getCurrentThemePalette().accentText
        })
        Text('新消息')
          .fontSize(DesignConstants.FONT.SIZE.XS)
          .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS))
          .fontColor(this.getCurrentThemePalette().accentText)
      }
      .padding({ left: DesignConstants.SPACING.MD, right: DesignConstants.SPACING.MD, top: DesignConstants.SPACING.SM, bottom: DesignConstants.SPACING.SM })
      .borderRadius(DesignConstants.RADIUS.FULL)
      .backgroundColor(this.getCurrentThemePalette().accentBackground)
      .shadow(this.getCurrentThemePalette().surfaceShadow) // 对齐前端 raised-sm（无彩色光晕）
      .position({ x: '50%', y: '78%' })
      .translate({ x: -56, y: 0 }) // 半宽回中（markAnchor 亦可；pill 约 112vp 宽）
      .onClick(() => { this.jumpToBottom(); })
      .accessibilityText('滚动到最新消息')
    }
    }
    }
    .hideTitleBar(true)
    .onBackPressed((): boolean => this.handleBackNavigation())
    .onReady((context: NavDestinationContext) => {
      this.pathStack = context.pathStack;
      const param = context.pathInfo.param as Record<string, Object> | undefined;
      if (param && param.chatId) {
        this.chatId = String(param.chatId ?? '');
        this.chatType = String(param.chatType ?? 'private_chat') as 'private_chat' | 'group_chat';
        this.title = String(param.title ?? '聊天');
        this.targetId = param.targetId ? String(param.targetId) : undefined;
        this.targetPresenceText = String(param.presenceText ?? '');
        this.targetPresenceOnline = this.targetPresenceText === '在线';
        this.initChatData();
      } else {
        // 缺参兜底（对齐 GroupDetailPage 缺群组ID模式）：直接进错误态，
        // 不调 initChatData，否则 isLoading 恒 true 陷入无限加载。
        hilog.error(DOMAIN_NUMBER, TAG, 'ChatPage 缺少路由参数');
        this.errorMessage = '缺少聊天参数';
        this.isLoading = false;
      }
    })
  }


  /** 日期分隔胶囊：渲染在 DatePill 组件（props 注入主题） */
  @Builder
  buildDatePill(label: string) {
    DatePill({
      label: label,
      palette: this.getCurrentThemePalette(),
      surfaceBorder: this.getSurfaceBorder()
    })
  }

  private async submitReport(): Promise<void> {
    if (this.reportSubmitting) {
      return;
    }
    this.reportSubmitting = true;
    try {
      const reportReq: CreateReportRequest = {
        target_type: 'user',
        target_id: this.targetId ?? this.chatId,
        reason: 'harassment',
        description: '来自聊天页面的举报'
      };
      await ApiClient.createReport(reportReq);
      SystemToast.show('举报已提交', this.getUIContext());
    } catch {
      SystemToast.showError('举报提交失败', this.getUIContext());
    } finally {
      this.reportSubmitting = false;
    }
  }

  // ==================== 消息撤回/编辑（对齐前端 MessageBubble ⋮ 菜单） ====================
  // 规则实现在 ../components/ChatMessageRules.ets（纯函数模块）；此处仅保留
  // 依赖页面状态的上下文判定（群角色 / 群聊类型）。

  private isGroupAdminOrOwner(): boolean {
    return this.groupRole === 'owner' || this.groupRole === 'admin';
  }

  private isAdminRecallContext(): boolean {
    return this.chatType === 'group_chat' && this.isGroupAdminOrOwner();
  }

  /** 气泡点击：定位原始消息并唤起操作面板 */
  private openMessageActions(view: ChatMessageView): void {
    const target = this.messages.find((m: ChatMessage) => m.id === view.id);
    if (target === undefined || !hasMessageActions(target, this.isAdminRecallContext())) {
      return;
    }
    this.dismissKeyboardOnOutsideTap();
    this.actionMessage = target;
    this.showMessageActionsSheet = true;
  }

  /** 面板回填内容（避免 Builder 内可空链式求值） */
  private resolveActionMessageContent(): string {
    const msg = this.actionMessage;
    if (msg === null) {
      return '';
    }
    return msg.content;
  }

  private isActionMessageEditable(): boolean {
    const msg = this.actionMessage;
    if (msg === null) {
      return false;
    }
    return canEditMessage(msg);
  }

  private isActionMessageRecallable(): boolean {
    const msg = this.actionMessage;
    if (msg === null) {
      return false;
    }
    return canRecallMessage(msg) || canAdminRecallMessage(msg, this.isAdminRecallContext());
  }

  /** 本地标记撤回（HTTP 成功后先落本地，WS 回环再幂等覆盖；对齐前端 handleRecall） */
  private applyRecallLocally(messageId: string): void {
    if (!messageId) {
      return;
    }
    this.messages = mapMessageById(this.messages, messageId, (m: ChatMessage): ChatMessage => {
      const next: ChatMessage = {
        id: m.id,
        content: RECALLED_MESSAGE_TEXT,
        senderId: m.senderId,
        senderName: m.senderName,
        senderAvatar: m.senderAvatar,
        type: m.type,
        sentAt: m.sentAt,
        isSelf: m.isSelf,
        isSystem: m.isSystem,
        recalled: true,
        rawData: m.rawData
      };
      return next;
    });
    this.messagesDataSource.setData(this.messages);
  }

  /** 本地应用编辑结果（HTTP 成功后先落本地，WS 回环再幂等覆盖；对齐前端 handleEdit） */
  private applyEditLocally(messageId: string, content: string): void {
    if (!messageId) {
      return;
    }
    this.messages = mapMessageById(this.messages, messageId, (m: ChatMessage): ChatMessage => {
      const next: ChatMessage = {
        id: m.id,
        content: content,
        senderId: m.senderId,
        senderName: m.senderName,
        senderAvatar: m.senderAvatar,
        type: m.type,
        sentAt: m.sentAt,
        isSelf: m.isSelf,
        isSystem: m.isSystem,
        recalled: m.recalled,
        edited: true,
        rawData: m.rawData
      };
      return next;
    });
    this.messagesDataSource.setData(this.messages);
  }

  private async handleRecallMessage(): Promise<void> {
    const msg = this.actionMessage;
    this.showMessageActionsSheet = false;
    this.actionMessage = null;
    if (!msg) {
      return;
    }
    try {
      if (msg.isSelf) {
        // 自己的消息走消息撤回端点（仅作者，时间窗内）
        if (this.chatType === 'private_chat') {
          await ApiClient.recallPrivateMessage({ message_id: msg.id });
        } else {
          await ApiClient.recallGroupMessage({ message_id: msg.id });
        }
      } else {
        // 群主/管理员撤回他人消息走 groups/admin/messages/recall（无时间窗）
        await ApiClient.recallGroupMessageAsAdmin({ group_id: this.chatId, message_id: msg.id });
      }
      this.applyRecallLocally(msg.id);
    } catch {
      SystemToast.showError('撤回失败', this.getUIContext());
    }
  }

  private async handleEditMessageSubmit(content: string): Promise<void> {
    const msg = this.actionMessage;
    if (!msg || !content) {
      return;
    }
    try {
      if (this.chatType === 'private_chat') {
        await ApiClient.updatePrivateMessage({ message_id: msg.id, content: content });
      } else {
        await ApiClient.updateGroupMessage({ message_id: msg.id, content: content });
      }
      this.showMessageActionsSheet = false;
      this.actionMessage = null;
      this.applyEditLocally(msg.id, content);
    } catch {
      SystemToast.showError('编辑失败', this.getUIContext());
    }
  }

  // ==================== 消息编辑/撤回 WS 事件（实时同步对端操作） ====================

  private onPrivateMessageEdited = (event: ServerEvent): void => {
    if (event.type !== 'PrivateMessageEdited') return;
    if (this.chatType !== 'private_chat') return;
    const payload = (event as PrivateMessageEditedEvent).payload;
    if (this.targetId && payload.target_user_id !== this.targetId) return;
    this.applyEditLocally(payload.message_id, payload.content);
  }

  private onPrivateMessageRecalled = (event: ServerEvent): void => {
    if (event.type !== 'PrivateMessageRecalled') return;
    if (this.chatType !== 'private_chat') return;
    const payload = (event as PrivateMessageRecalledEvent).payload;
    if (this.targetId && payload.target_user_id !== this.targetId) return;
    this.applyRecallLocally(payload.message_id);
  }

  private onGroupMessageEdited = (event: ServerEvent): void => {
    if (event.type !== 'GroupMessageEdited') return;
    if (this.chatType !== 'group_chat') return;
    const payload = (event as GroupMessageEditedEvent).payload;
    if (payload.group_id !== this.chatId) return;
    this.applyEditLocally(payload.message_id, payload.content);
  }

  private onGroupMessageRecalled = (event: ServerEvent): void => {
    if (event.type !== 'GroupMessageRecalled') return;
    if (this.chatType !== 'group_chat') return;
    const payload = (event as GroupMessageRecalledEvent).payload;
    if (payload.group_id !== this.chatId) return;
    this.applyRecallLocally(payload.message_id);
  }

  @Builder
  private conversationActionsSheetBuilder() {
    ChatActionsSheet({
      isGroupChat: this.chatType === 'group_chat',
      targetId: this.targetId ?? '',
      palette: this.getCurrentThemePalette(),
      surfaceBackground: this.getSurfaceBackground(),
      surfaceBorder: this.getSurfaceBorder(),
      onOpenProfile: () => {
        this.showConversationActionsSheet = false;
        const profileParam = new ProfileRouteParam();
        profileParam.userId = this.targetId ?? '';
        AppRouter.push(PageNames.PROFILE, profileParam);
      },
      onScrollToBottom: () => {
        this.scrollToBottom();
        this.showConversationActionsSheet = false;
      },
      onClearInput: () => {
        this.inputText = '';
        this.showConversationActionsSheet = false;
      },
      onReport: () => {
        this.showConversationActionsSheet = false;
        this.showReportConfirm = true;
      },
      onClose: () => { this.showConversationActionsSheet = false; }
    })
  }

  @Builder
  private messageActionsSheetBuilder() {
    MessageActionsSheet({
      messageContent: this.resolveActionMessageContent(),
      canEdit: this.isActionMessageEditable(),
      canRecall: this.isActionMessageRecallable(),
      palette: this.getCurrentThemePalette(),
      surfaceBackground: this.getSurfaceBackground(),
      surfaceBorder: this.getSurfaceBorder(),
      onEdit: (content: string) => { this.handleEditMessageSubmit(content); },
      onRecall: () => { this.handleRecallMessage(); },
      onClose: () => {
        this.showMessageActionsSheet = false;
        this.actionMessage = null;
      }
    })
  }


}

@Builder
export function ChatPageBuilder() {
  ChatPage()
}