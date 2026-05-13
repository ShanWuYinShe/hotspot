/**
 * 聊天详情页面
 * 重构：使用 List 组件优化消息列表，支持自动滚动
 * 优化：集成消息历史缓存 (RDB)
 */

import { GlobalStateManager, KEY_CURRENT_USER } from '../../../core/state/GlobalStateManager';
import { ApiClient } from '../../../core/api/ApiClient';
import { WebSocketService } from '../../../core/network/WebSocketService';
import { MessageSyncManager } from '../../../core/state/MessageSyncManager';
import { MyProfile, MessageType, PrivateMessageItem, GroupMessageItem, SendPrivateMessageRequest, SendGroupMessageRequest, GetPrivateMessagesHistoryRequest, GetGroupMessagesHistoryRequest, MarkPrivateMessagesReadRequest, MarkGroupMessagesReadRequest, UserOpenTheDoorRequest } from '../../../core/types/api';
import type {
  ServerEvent,
  PrivateMessageNewEvent,
  GroupMessageNewEvent,
  PrivateUserTypingEvent,
  GroupUserTypingEvent
} from '../../../core/types/WebSocketTypes';
import { PageNames } from '../../../constants/AppConstants';
import { SystemAvatar } from '../../../components/SystemAvatar';
import { SystemLoadingView } from '../../../components/SystemLoadingView';
import { SystemToast } from '../../../components/SystemToast';
import { DesignConstants } from '../../../constants/DesignConstants';
import { AppIcon } from '../../../components/SystemIcon';
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
import { KeyboardAvoidMode as UIContextKeyboardAvoidMode } from '@ohos.arkui.UIContext';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { AppThemeVisuals } from '../../../core/theme/AppThemeVisuals';
import { ChatThemeManager } from '../../../core/theme/ChatTheme';
import { KEY_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';

const DOMAIN_NUMBER: number = 0xFF09;
const TAG: string = 'ChatPage';
const MESSAGE_CONTENT_MAX: number = 5000;
const TYPING_THROTTLE_MS: number = 3000;
const TYPING_INDICATOR_TTL_MS: number = 4500;
const GROUP_TYPING_NAME_PREVIEW_LIMIT: number = 3;
const RECONNECT_BACKFILL_THRESHOLD_MS: number = 10000;
const SAFE_AREA_KEYBOARD_SPIKE_THRESHOLD: number = 120;
const SAFE_AREA_SPIKE_DELTA: number = 48;
const DUPLICATE_MESSAGE_TIME_WINDOW_MS: number = 1500;
const SEND_DUPLICATE_GUARD_WINDOW_MS: number = 1200;
const KEYBOARD_DISMISS_NAV_GUARD_MS: number = 260;

// 统一消息接口用于UI展示
interface ChatMessage {
  id: string;
  content: string;
  senderId: string;
  senderName: string;
  senderAvatar?: string;
  type: MessageType;
  sentAt: string;
  isSelf: boolean;
  isSystem: boolean;
  rawData?: Object;
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
  @StorageLink(KEY_CURRENT_USER) currentUser: MyProfile | null = null;
  @StorageLink('safeAreaTop') safeAreaTop: number = 0;
  @StorageLink('safeAreaBottom') safeAreaBottom: number = 0;
  @StorageLink(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'vibrant_light';
  
  @State messages: ChatMessage[] = [];
  private messagesDataSource: BasicDataSource<ChatMessage> = new BasicDataSource<ChatMessage>([]);
  @State inputText: string = '';
  @State isLoading: boolean = true;
  @State errorMessage: string = '';
  @State isSending: boolean = false;
  @State isInputFocused: boolean = false;
  @State isOtherUserTyping: boolean = false;
  @State groupTypingUserIds: string[] = [];
  @State title: string = '聊天';
  @State hasMoreHistory: boolean = false;
  @State historyNextCursor: string = '';
  @State isLoadingMoreHistory: boolean = false;
  @State showConversationActionsSheet: boolean = false;
  @State targetAvatar: string = '';
  @State isMuted: boolean = false;

  private chatId: string = '';
  private chatType: 'private_chat' | 'group_chat' = 'private_chat';
  private targetId?: string;
  private webSocketService = WebSocketService.getInstance();
  private messageCacheService = MessageCacheService.getInstance();
  private scroller: Scroller = new Scroller();
  private typingStopTimer: number = -1;
  private lastTypingSentAt: number = 0;
  private typingStateReported: boolean = false;
  private privateTypingResetTimer: number = -1;
  private groupTypingTimers: Map<string, number> = new Map();
  private wsHasConnectedOnce: boolean = false;
  private isReconnectReconcileInFlight: boolean = false;
  private stableSafeAreaBottom: number = 0;
  private lastSendSignature: string = '';
  private lastSendAtMs: number = 0;
  private lastKeyboardDismissAtMs: number = 0;

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

  private getStableBottomInset(): number {
    const currentInset = Math.max(0, this.safeAreaBottom);
    if (this.stableSafeAreaBottom <= 0) {
      this.stableSafeAreaBottom = currentInset;
      return currentInset;
    }
    const isKeyboardSpike = currentInset >= SAFE_AREA_KEYBOARD_SPIKE_THRESHOLD &&
      currentInset > this.stableSafeAreaBottom + SAFE_AREA_SPIKE_DELTA;
    if (!isKeyboardSpike) {
      this.stableSafeAreaBottom = currentInset;
      return currentInset;
    }
    return this.stableSafeAreaBottom;
  }

  private getComposerBottomPadding(): number {
    const stableInset = this.getStableBottomInset();
    return SafeAreaUtils.resolveBottomPadding(stableInset, 0, DesignConstants.SPACING.SM);
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

  private buildMessageFingerprint(message: ChatMessage): string {
    const sentAtMs = Date.parse(message.sentAt);
    const normalizedSentAt = Number.isNaN(sentAtMs) ? message.sentAt : String(sentAtMs);
    return `${message.senderId}|${message.type}|${normalizedSentAt}|${message.content}`;
  }

  private buildMessageContentSignature(message: ChatMessage): string {
    return `${message.senderId}|${message.type}|${message.content}`;
  }

  private dedupeMessages(messages: ChatMessage[]): ChatMessage[] {
    const seenIds: Set<string> = new Set();
    const seenFingerprints: Set<string> = new Set();
    const recentContentSentAt: Map<string, number> = new Map();
    const uniqueMessages: ChatMessage[] = [];

    messages.forEach((message: ChatMessage) => {
      const normalizedId = message.id.trim();
      if (normalizedId.length > 0) {
        if (seenIds.has(normalizedId)) {
          return;
        }
        seenIds.add(normalizedId);
      }

      const fingerprint = this.buildMessageFingerprint(message);
      if (seenFingerprints.has(fingerprint)) {
        return;
      }

      const sentAtMs = Date.parse(message.sentAt);
      if (!Number.isNaN(sentAtMs)) {
        const contentSignature = this.buildMessageContentSignature(message);
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

  private onPrivateUserTyping = (event: ServerEvent): void => {
    if (event.type !== 'PrivateUserTyping') return;
    if (this.chatType !== 'private_chat') return;

    const payload = (event as PrivateUserTypingEvent).payload;
    const targetId = this.targetId;
    if (!targetId) return;

    if (payload.user_id !== targetId) return;

    if (payload.is_typing) {
      this.isOtherUserTyping = true;
      this.schedulePrivateTypingAutoClear();
      return;
    }
    this.resetPrivateTypingState();
  }

  private onGroupUserTyping = (event: ServerEvent): void => {
    if (event.type !== 'GroupUserTyping') return;
    if (this.chatType !== 'group_chat') return;

    const payload = (event as GroupUserTypingEvent).payload;
    if (payload.group_id !== this.chatId) return;
    if (payload.user_id === (this.currentUser?.user_id ?? '')) return;

    if (payload.is_typing) {
      if (this.groupTypingUserIds.indexOf(payload.user_id) === -1) {
        this.groupTypingUserIds = this.groupTypingUserIds.concat([payload.user_id]);
      }
      this.scheduleGroupTypingAutoClear(payload.user_id);
      return;
    }
    this.clearGroupTypingTimer(payload.user_id);
    this.groupTypingUserIds = this.groupTypingUserIds.filter((id: string) => id !== payload.user_id);
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
    const newFingerprint = this.buildMessageFingerprint(newMessage);
    const duplicated = this.messages.some((item: ChatMessage) => {
      const itemId = item.id.trim();
      if (normalizedId.length > 0 && itemId === normalizedId) {
        return true;
      }
      return this.buildMessageFingerprint(item) === newFingerprint;
    });
    if (duplicated) {
      return;
    }

    const nextMessages = this.messages.concat([newMessage]);
    this.messages = nextMessages;
    this.messagesDataSource.setData(nextMessages);
    this.scrollToBottom();

    if (!newMessage.isSelf) {
      this.markAsRead(newMessage.id);
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
    this.isLoading = true;
    this.errorMessage = '';
    
    await this.loadFromCache();

    try {
      await LoadingUtil.withMinimumTime(async () => {
        let items: (PrivateMessageItem | GroupMessageItem)[] = [];
        let nextCursor: string | undefined;
        
        if (this.chatType === 'private_chat') {
          const targetUserId = this.targetId;
          if (!targetUserId) {
            throw new Error('target_user_id is required for private chat');
          }
          const response = await ApiClient.privateChatOpenTheDoor({ target_user_id: targetUserId });
          if (response.code === 0 && response.data) {
            items = response.data.recent_messages?.items ?? [];
            nextCursor = response.data.recent_messages?.next_cursor;
          }
        } else {
          const request: GetGroupMessagesHistoryRequest = { group_id: this.chatId, pagination: { limit: 20 } };
          const response = await ApiClient.getGroupMessages(request);
          if (response.code === 0) {
            items = response.data?.items ?? [];
            nextCursor = response.data?.next_cursor;
          }
        }

        const convertedMessages = await Promise.all(items.reverse().map(item => this.convertMessage(item)));
        const newMessages = this.dedupeMessages(convertedMessages);
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
      this.isLoading = false;
    }
  }

  private async loadMoreHistory(): Promise<void> {
    if (this.isLoadingMoreHistory || !this.hasMoreHistory) return;
    this.isLoadingMoreHistory = true;
    try {
      let items: (PrivateMessageItem | GroupMessageItem)[] = [];
      let nextCursor: string | undefined;

      if (this.chatType === 'private_chat') {
        const targetUserId = this.targetId;
        if (!targetUserId) return;
        const request: GetPrivateMessagesHistoryRequest = {
          target_user_id: targetUserId,
          pagination: { limit: 20, cursor: this.historyNextCursor }
        };
        const response = await ApiClient.getPrivateMessages(request);
        if (response.code === 0) {
          items = response.data?.items ?? [];
          nextCursor = response.data?.next_cursor;
        }
      } else {
        const request: GetGroupMessagesHistoryRequest = {
          group_id: this.chatId,
          pagination: { limit: 20, cursor: this.historyNextCursor }
        };
        const response = await ApiClient.getGroupMessages(request);
        if (response.code === 0) {
          items = response.data?.items ?? [];
          nextCursor = response.data?.next_cursor;
        }
      }

      const olderMessages = await Promise.all(items.reverse().map(item => this.convertMessage(item)));
      const mergedMessages = this.dedupeMessages(olderMessages.concat(this.messages));
      this.messages = mergedMessages;
      this.messagesDataSource.setData(mergedMessages);
      this.hasMoreHistory = nextCursor !== undefined && nextCursor !== '';
      this.historyNextCursor = nextCursor ?? '';
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
      const messages = this.dedupeMessages(cachedMapped);
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

  private async convertMessage(item: PrivateMessageItem | GroupMessageItem): Promise<ChatMessage> {
    const currentUserId = await AuthAccountManager.getCurrentUserId();
    
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
      isSelf = senderId === currentUserId;
    } else {
      senderId = '';
      senderName = '';
      senderAvatar = undefined;
      isSelf = false;
    }
    
    const messageType: MessageType = item.type as MessageType;
    return {
      id: item.message_id,
      content: item.media?.url || item.content || (item.type === 'image' ? '[图片]' : '[非文本消息]'),
      senderId: senderId,
      senderName: senderName,
      senderAvatar: senderAvatar,
      type: messageType,
      sentAt: item.sent_at,
      isSelf: isSelf,
      isSystem: item.type === 'system',
      rawData: item
    };
  }

  private mergeMessagesById(currentMessages: ChatMessage[], recentMessages: ChatMessage[]): ChatMessage[] {
    const mergedMessages = this.dedupeMessages(currentMessages.concat(recentMessages));
    mergedMessages.sort((a: ChatMessage, b: ChatMessage) => {
      const timeDiff = new Date(a.sentAt).getTime() - new Date(b.sentAt).getTime();
      if (timeDiff !== 0) {
        return timeDiff;
      }
      return a.id.localeCompare(b.id);
    });
    return mergedMessages;
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
      const recentMessages = await Promise.all(items.reverse().map(item => this.convertMessage(item)));
      const mergedMessages = this.mergeMessagesById(this.messages, recentMessages);
      this.messages = mergedMessages;
      this.messagesDataSource.setData(mergedMessages);
    } catch (error) {
      hilog.warn(DOMAIN_NUMBER, TAG, `重连后消息回补失败: ${JSON.stringify(error)}`);
    } finally {
      this.isReconnectReconcileInFlight = false;
    }
  }

  private subscribeToEvents() {
    this.webSocketService.on('PrivateMessageNew', this.onPrivateMessageNew);
    this.webSocketService.on('GroupMessageNew', this.onGroupMessageNew);
    this.webSocketService.on('PrivateUserTyping', this.onPrivateUserTyping);
    this.webSocketService.on('GroupUserTyping', this.onGroupUserTyping);
    this.webSocketService.on('ConnectionEstablished', this.onConnectionEstablished);
  }

  private unsubscribeFromEvents() {
    this.webSocketService.off('PrivateMessageNew', this.onPrivateMessageNew);
    this.webSocketService.off('GroupMessageNew', this.onGroupMessageNew);
    this.webSocketService.off('PrivateUserTyping', this.onPrivateUserTyping);
    this.webSocketService.off('GroupUserTyping', this.onGroupUserTyping);
    this.webSocketService.off('ConnectionEstablished', this.onConnectionEstablished);
  }

  private scrollToBottom() {
    setTimeout(() => {
      this.scroller.scrollEdge(Edge.Bottom);
    }, 100);
  }

  private async pickAttachment(): Promise<void> {
    try {
      const context = AppStorage.get<common.UIAbilityContext>('context');
      if (!context) return;
      const uris = await MediaServiceManager.selectImages(context, 1);
      if (uris.length > 0) {
        SystemToast.show('图片功能开发中', this.getUIContext());
      }
    } catch (e) {
      hilog.info(DOMAIN_NUMBER, TAG, `附件选择取消或失败: ${e}`);
    }
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
        const request: SendGroupMessageRequest = {
          group_id: this.chatId,
          type: 'text',
          content: content
        };
        await ApiClient.sendGroupMessage(request);
      }

      this.inputText = '';
    } catch (error) {
      this.lastSendSignature = '';
      this.lastSendAtMs = 0;
      SystemToast.showError('发送失败', this.getUIContext());
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

  private groupTypingHintText(): string {
    const count = this.groupTypingUserIds.length;
    if (count <= 0) {
      return '';
    }
    const previewCount = count > GROUP_TYPING_NAME_PREVIEW_LIMIT ? GROUP_TYPING_NAME_PREVIEW_LIMIT : count;
    const previewNames: string[] = [];
    for (let index = 0; index < previewCount; index++) {
      previewNames.push(this.resolveGroupTypingDisplayName(this.groupTypingUserIds[index]));
    }
    if (count === 1) {
      return `${previewNames[0]} 正在输入...`;
    }
    if (count <= GROUP_TYPING_NAME_PREVIEW_LIMIT) {
      return `${previewNames.join('、')} 正在输入...`;
    }
    return `${previewNames.join('、')} 等 ${count} 人正在输入...`;
  }

  private resolveGroupTypingDisplayName(userId: string): string {
    for (let index = this.messages.length - 1; index >= 0; index--) {
      const message = this.messages[index];
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

  private markAsRead(upToMessageId: string): void {
    if (!upToMessageId) return;

    (async () => {
      try {
        if (this.chatType === 'private_chat') {
          const targetUserId = this.targetId;
          if (!targetUserId) return;
          const request: MarkPrivateMessagesReadRequest = { target_user_id: targetUserId, up_to_message_id: upToMessageId };
          await ApiClient.markPrivateMessagesRead(request);
        } else {
          const request: MarkGroupMessagesReadRequest = { group_id: this.chatId, up_to_message_id: upToMessageId };
          await ApiClient.markGroupMessagesRead(request);
        }
        GlobalStateManager.getInstance().notifyMessageSync();
      } catch (e) {
        // ignore
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

    // spec: PrivateUserTyping 依赖 user 订阅
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
      const resp = await ApiClient.getUserProfile({ user_id: this.targetId } as UserOpenTheDoorRequest);
      if (resp.code === 0 && resp.data) {
        this.targetAvatar = resp.data.user.avatar_url ?? '';
      }
    } catch (e) {
      hilog.warn(DOMAIN_NUMBER, TAG, `加载目标用户头像失败: ${e}`);
    }
  }

  build() {
    NavDestination() {
    Column() {
      Row() {
        Row() {
          Button() {
            Text() {
              SymbolSpan($r(AppIcon.BACK))
                .fontSize(20)
                .fontColor([this.getCurrentThemePalette().textPrimary])
            }
          }
          .type(ButtonType.Circle)
          .width(36)
          .height(36)
          .backgroundColor(Color.Transparent)
          .onClick(() => this.handleBackNavigation())

          if (this.chatType === 'private_chat' && this.targetAvatar) {
            SystemAvatar({
              avatarUrl: this.targetAvatar,
              name: this.title,
              avatarSize: 32,
              shape: 'circle'
            })
              .margin({ left: DesignConstants.SPACING.XS })
          }

          Text(this.title)
            .fontSize(DesignConstants.FONT.SIZE.LG)
            .fontWeight(FontWeight.Bold)
            .fontColor(this.getCurrentThemePalette().textPrimary)
            .maxLines(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
            .layoutWeight(1)
            .margin({ left: DesignConstants.SPACING.SM })
        }
        .layoutWeight(1)

        Button() {
          Text() {
            SymbolSpan($r(AppIcon.INFO))
              .fontSize(20)
              .fontColor([this.getCurrentThemePalette().textSecondary])
          }
        }
        .type(ButtonType.Circle)
        .width(36)
        .height(36)
        .backgroundColor(Color.Transparent)
        .onClick(() => this.openConversationActions())
      }
        .width('100%')
        .margin({
          bottom: DesignConstants.SPACING.XS
        })
        .shadow(this.getCurrentThemePalette().elevatedShadow)
        .border({ width: 1, color: this.getChatTheme().titleBarBorder })
        .borderRadius(DesignConstants.RADIUS.XL)
        .zIndex(1)
        .onTouch((event: TouchEvent) => {
          if (event.type === TouchType.Down) {
            this.dismissKeyboardOnOutsideTap();
          }
        })

      if (this.isLoading) {
        SystemLoadingView({ message: '加载消息...' })
          .layoutWeight(1)
          .onTouch((event: TouchEvent) => {
            if (event.type === TouchType.Down) {
              this.dismissKeyboardOnOutsideTap();
            }
          })
          .transition(
            TransitionEffect.OPACITY.animation({ duration: 320, curve: Curve.FastOutSlowIn })
              .combine(TransitionEffect.move(TransitionEdge.BOTTOM).animation({ duration: 320, curve: Curve.FastOutSlowIn }))
          )
      } else if (this.messages.length === 0 && this.errorMessage) {
        Column({ space: DesignConstants.SPACING.LG }) {
          Text() {
            SymbolSpan($r(AppIcon.WARNING))
              .fontSize(48)
              .fontColor([this.getCurrentThemePalette().textSecondary])
          }
          Text(this.errorMessage)
            .fontSize(DesignConstants.FONT.SIZE.LG)
            .fontColor(this.getCurrentThemePalette().textSecondary)
            .textAlign(TextAlign.Center)
          Button('重试')
            .type(ButtonType.Normal)
            .backgroundColor(this.getCurrentThemePalette().accentBackground)
            .fontColor(this.getCurrentThemePalette().accentText)
            .height(40)
            .borderRadius(DesignConstants.RADIUS.XL)
            .onClick(() => { this.loadHistory(); })
        }
        .width('100%')
        .layoutWeight(1)
        .justifyContent(FlexAlign.Center)
        .onTouch((event: TouchEvent) => {
          if (event.type === TouchType.Down) {
            this.dismissKeyboardOnOutsideTap();
          }
        })
        .transition(
          TransitionEffect.OPACITY.animation({ duration: 320, curve: Curve.FastOutSlowIn })
            .combine(TransitionEffect.move(TransitionEdge.BOTTOM).animation({ duration: 320, curve: Curve.FastOutSlowIn }))
        )
      } else if (this.messages.length === 0) {
        Column({ space: DesignConstants.SPACING.LG }) {
          Text() {
            SymbolSpan($r(AppIcon.MESSAGE))
              .fontSize(48)
              .fontColor([this.getCurrentThemePalette().textTertiary])
          }
          Text('暂无消息')
            .fontSize(DesignConstants.FONT.SIZE.LG)
            .fontColor(this.getCurrentThemePalette().textSecondary)
          Text('发送一条消息开始聊天吧')
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .fontColor(this.getCurrentThemePalette().textTertiary)
        }
        .width('100%')
        .layoutWeight(1)
        .justifyContent(FlexAlign.Center)
        .onTouch((event: TouchEvent) => {
          if (event.type === TouchType.Down) {
            this.dismissKeyboardOnOutsideTap();
          }
        })
      } else {
      List({ scroller: this.scroller, initialIndex: this.messages.length - 1 }) {
        // 顶部加载更多历史指示器
        if (this.isLoadingMoreHistory) {
          ListItem() {
            Row({ space: DesignConstants.SPACING.SM }) {
              LoadingProgress().width(20).height(20)
              Text('加载更早消息...')
                .fontSize(DesignConstants.FONT.SIZE.SM)
                .fontColor(this.getCurrentThemePalette().textSecondary)
            }
            .width('100%')
            .justifyContent(FlexAlign.Center)
            .padding(DesignConstants.SPACING.MD)
          }
        }
        LazyForEach(this.messagesDataSource, (msg: ChatMessage) => {
          ListItem() {
            this.buildMessageItem(msg)
          }
        }, (msg: ChatMessage) => msg.id)
      }
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
        // 当滚动到列表顶部附近时加载更早的消息
        if (start <= 1 && this.hasMoreHistory && !this.isLoadingMoreHistory) {
          this.loadMoreHistory();
        }
      })
      .onTouch((event: TouchEvent) => {
        if (event.type === TouchType.Down) {
          this.dismissKeyboardOnOutsideTap();
        }
      })
      .transition(
        TransitionEffect.OPACITY.animation({ duration: 320, curve: Curve.FastOutSlowIn })
          .combine(TransitionEffect.move(TransitionEdge.BOTTOM).animation({ duration: 320, curve: Curve.FastOutSlowIn }))
      )
      }

      if (this.chatType === 'private_chat' && this.isOtherUserTyping) {
        Row() {
          Text('对方正在输入...')
            .fontSize(DesignConstants.FONT.SIZE.SM)
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

      Row() {
        Button() {
          Text() {
            SymbolSpan($r(AppIcon.ADD))
              .fontSize(22)
              .fontColor([this.getCurrentThemePalette().textSecondary])
          }
        }
        .type(ButtonType.Circle)
        .width(36)
        .height(36)
        .backgroundColor(this.getChatTheme().inputFieldBg)
        .borderRadius(DesignConstants.RADIUS.FULL)
        .shadow(this.getChatVisualStyle().input.shadow)
        .border({ width: 1, color: this.getChatTheme().inputBorder })
        .margin({ right: DesignConstants.SPACING.SM })
        .onClick(() => this.pickAttachment())

        TextInput({ text: this.inputText, placeholder: '发送消息...' })
          .layoutWeight(1)
          .maxLength(MESSAGE_CONTENT_MAX)
          .height(this.getChatVisualStyle().input.height)
          .backgroundColor(this.getChatTheme().inputFieldBg)
          .borderRadius(this.getChatVisualStyle().input.borderRadius)
          .shadow(this.getChatVisualStyle().input.shadow)
          .padding({ left: this.getChatVisualStyle().input.horizontalPadding, right: this.getChatVisualStyle().input.horizontalPadding })
          .onChange((value) => {
            this.inputText = value;
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

        Button(this.isSending ? '发送中...' : '发送')
          .type(ButtonType.Normal)
          .backgroundColor(
            this.isSending || this.inputText.trim().length === 0
              ? this.getChatTheme().sendButtonDisabled
              : this.getChatTheme().sendButtonActive
          )
          .fontColor(
            this.isSending || this.inputText.trim().length === 0
              ? this.getCurrentThemePalette().textSecondary
              : this.getChatTheme().sendButtonActiveText
          )
          .height(this.getChatVisualStyle().sendButton.height)
          .borderRadius(this.getChatVisualStyle().sendButton.borderRadius)
          .shadow(this.getChatVisualStyle().sendButton.shadow)
          .border({ width: 1, color: this.getChatTheme().inputBorder })
          .padding({ left: this.getChatVisualStyle().sendButton.horizontalPadding, right: this.getChatVisualStyle().sendButton.horizontalPadding })
          .margin({ left: DesignConstants.SPACING.MD })
          .opacity(this.isSending || this.inputText.trim().length === 0 ? DesignConstants.OPACITY.PRESSED : 1)
          .enabled(!this.isSending && this.inputText.trim().length > 0)
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
      .border({ width: { top: 1 }, color: this.getChatTheme().inputBorder })
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
    .bindSheet($$this.showConversationActionsSheet, this.buildConversationActionsSheet(), {
      height: SheetSize.FIT_CONTENT,
      showClose: true,
      dragBar: true,
      backgroundColor: this.getCurrentThemePalette().sheetBackground
    })
    }
    .hideTitleBar(true)
    .onBackPressed((): boolean => this.handleBackNavigation())
    .onReady((context: NavDestinationContext) => {
      this.pathStack = context.pathStack;
      const param = context.pathInfo.param as Record<string, Object> | undefined;
      if (param) {
        this.chatId = String(param.chatId ?? '');
        this.chatType = String(param.chatType ?? 'private_chat') as 'private_chat' | 'group_chat';
        this.title = String(param.title ?? '聊天');
        this.targetId = param.targetId ? String(param.targetId) : undefined;
        this.initChatData();
      } else {
        hilog.error(DOMAIN_NUMBER, TAG, 'ChatPage 缺少路由参数');
      }
    })
  }

  @Builder
  private buildConversationActionsSheet() {
    Column({ space: DesignConstants.SPACING.MD }) {
      Text('会话设置')
        .fontSize(DesignConstants.FONT.SIZE.XL)
        .fontWeight(FontWeight.Bold)
        .fontColor(this.getCurrentThemePalette().textPrimary)

      Text(this.chatType === 'group_chat' ? '群聊会话' : '私聊会话')
        .fontSize(DesignConstants.FONT.SIZE.SM)
        .fontColor(this.getCurrentThemePalette().textSecondary)

      Divider().color(this.getSurfaceBorder())

      if (this.chatType === 'private_chat' && this.targetId) {
        Button('查看个人主页')
          .type(ButtonType.Normal)
          .width('100%')
          .height(44)
          .backgroundColor(this.getSurfaceSubtle())
          .fontColor(this.getCurrentThemePalette().textPrimary)
          .borderRadius(DesignConstants.RADIUS.FULL)
          .border({ width: 1, color: this.getSurfaceBorder() })
          .onClick(() => {
            this.showConversationActionsSheet = false;
            const profileParam = new ProfileRouteParam();
            profileParam.userId = this.targetId ?? '';
            AppRouter.push(PageNames.PROFILE, profileParam);
          })
      }

      Button(this.isMuted ? '取消静音' : '静音通知')
        .type(ButtonType.Normal)
        .width('100%')
        .height(44)
        .backgroundColor(this.getSurfaceSubtle())
        .fontColor(this.getCurrentThemePalette().textPrimary)
        .borderRadius(DesignConstants.RADIUS.FULL)
        .border({ width: 1, color: this.getSurfaceBorder() })
        .onClick(() => {
          this.isMuted = !this.isMuted;
          SystemToast.show(this.isMuted ? '已静音' : '已取消静音', this.getUIContext());
        })

      Button('回到最新消息')
        .type(ButtonType.Normal)
        .width('100%')
        .height(44)
        .backgroundColor(this.getSurfaceSubtle())
        .fontColor(this.getCurrentThemePalette().textPrimary)
        .borderRadius(DesignConstants.RADIUS.FULL)
        .border({ width: 1, color: this.getSurfaceBorder() })
        .onClick(() => {
          this.scrollToBottom();
          this.showConversationActionsSheet = false;
        })

      Button('清空输入框')
        .type(ButtonType.Normal)
        .width('100%')
        .height(44)
        .backgroundColor(this.getSurfaceSubtle())
        .fontColor(this.getCurrentThemePalette().textPrimary)
        .borderRadius(DesignConstants.RADIUS.FULL)
        .border({ width: 1, color: this.getSurfaceBorder() })
        .onClick(() => {
          this.inputText = '';
          this.showConversationActionsSheet = false;
        })

      Divider().color(this.getSurfaceBorder())

      Button('举报')
        .type(ButtonType.Normal)
        .width('100%')
        .height(44)
        .backgroundColor(this.getSurfaceSubtle())
        .fontColor('#E74C3C')
        .borderRadius(DesignConstants.RADIUS.FULL)
        .border({ width: 1, color: this.getSurfaceBorder() })
        .onClick(() => {
          this.showConversationActionsSheet = false;
          SystemToast.show('举报功能开发中', this.getUIContext());
        })

      Button('关闭')
        .type(ButtonType.Normal)
        .width('100%')
        .height(44)
        .backgroundColor(this.getSurfaceBackground())
        .fontColor(this.getAccentBackground())
        .borderRadius(DesignConstants.RADIUS.FULL)
        .border({ width: 1, color: this.getSurfaceBorder() })
        .onClick(() => {
          this.showConversationActionsSheet = false;
        })
    }
    .width('100%')
    .padding({
      left: DesignConstants.SPACING.XXL,
      right: DesignConstants.SPACING.XXL,
      top: DesignConstants.SPACING.LG,
      bottom: DesignConstants.SPACING.XXL
    })
    .backgroundColor(this.getSurfaceBackground())
    .borderRadius(DesignConstants.RADIUS.LG)
  }

  @Builder
  buildMessageItem(msg: ChatMessage) {
    if (msg.isSystem) {
      Row() {
        Text(msg.content)
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .fontColor(this.getCurrentThemePalette().textSecondary)
          .backgroundColor(this.getChatTheme().bubbleSystem)
          .padding({
            left: DesignConstants.SPACING.MD + 2,
            right: DesignConstants.SPACING.MD + 2,
            top: DesignConstants.SPACING.XS + 2,
            bottom: DesignConstants.SPACING.XS + 2
          })
          .borderRadius(DesignConstants.CHAT.BUBBLE_RADIUS)
          .shadow(this.getCurrentThemePalette().controlShadow)
          .border({ width: 1, color: this.getChatTheme().divider })
      }
      .width('100%')
      .justifyContent(FlexAlign.Center)
      .margin({ top: DesignConstants.SPACING.SM, bottom: DesignConstants.SPACING.SM })
    } else {
      Row() {
        if (!msg.isSelf) {
          SystemAvatar({
            avatarUrl: msg.senderAvatar,
            name: msg.senderName,
            avatarSize: DesignConstants.CHAT.AVATAR_SIZE
          })
          .margin({ right: DesignConstants.SPACING.SM })
        }

        Column() {
          if (!msg.isSelf && this.chatType === 'group_chat') {
            Text(msg.senderName)
              .fontSize(DesignConstants.FONT.SIZE.SM)
              .fontColor(this.getCurrentThemePalette().textSecondary)
              .margin({ bottom: DesignConstants.SPACING.XS, left: DesignConstants.SPACING.XS })
          }

            Row() {
            if (msg.type === 'image') {
              Image(MediaUrlUtil.normalize(msg.content))
                .width(200)
                .objectFit(ImageFit.Cover)
                .backgroundColor(this.getChatTheme().listBackground)
                .borderRadius(DesignConstants.CHAT.BUBBLE_RADIUS)
                .shadow(this.getCurrentThemePalette().controlShadow)
                .border({ width: 1, color: this.getChatTheme().divider })
                .margin(DesignConstants.SPACING.XS)
                .accessibilityText('[图片消息]')
            } else {
              Text(msg.content)
                .fontSize(DesignConstants.CHAT.BUBBLE_FONT_SIZE)
                .fontColor(msg.isSelf ? this.getChatTheme().sendButtonActiveText : this.getCurrentThemePalette().textPrimary)
                .backgroundColor(msg.isSelf ? this.getChatTheme().bubbleSelf : this.getChatTheme().bubblePeer)
                .padding({
                  left: DesignConstants.CHAT.BUBBLE_PADDING_H + 2,
                  right: DesignConstants.CHAT.BUBBLE_PADDING_H + 2,
                  top: DesignConstants.CHAT.BUBBLE_PADDING_V + 1,
                  bottom: DesignConstants.CHAT.BUBBLE_PADDING_V + 1
                })
                .borderRadius(DesignConstants.CHAT.BUBBLE_RADIUS)
                .constraintSize({ maxWidth: DesignConstants.CHAT.BUBBLE_MAX_WIDTH })
                .shadow(msg.isSelf ? this.getCurrentThemePalette().accentShadow : this.getCurrentThemePalette().controlShadow)
            }
          }
          .justifyContent(msg.isSelf ? FlexAlign.End : FlexAlign.Start)
        }
        .alignItems(msg.isSelf ? HorizontalAlign.End : HorizontalAlign.Start)
        .layoutWeight(1)

        if (msg.isSelf) {
          SystemAvatar({
            avatarUrl: msg.senderAvatar,
            name: msg.senderName,
            avatarSize: DesignConstants.CHAT.AVATAR_SIZE
          })
          .margin({ left: DesignConstants.SPACING.SM })
        }
      }
      .width('100%')
      .justifyContent(msg.isSelf ? FlexAlign.End : FlexAlign.Start)
      .alignItems(VerticalAlign.Top)
      .margin({ bottom: DesignConstants.CHAT.MESSAGE_GAP })
    }
  }
}

@Builder
export function ChatPageBuilder() {
  ChatPage()
}
