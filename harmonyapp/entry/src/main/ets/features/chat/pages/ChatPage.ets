/**
 * 聊天详情页面
 * 重构：使用 List 组件优化消息列表，支持自动滚动
 * 优化：集成消息历史缓存 (RDB)
 */

import { GlobalStateManager, KEY_CURRENT_USER } from '../../../core/state/GlobalStateManager';
import { ApiClient } from '../../../core/api/ApiClient';
import { WebSocketService } from '../../../core/network/WebSocketService';
import { MyProfile, MessageType, PrivateMessageItem, GroupMessageItem, SendPrivateMessageRequest, SendGroupMessageRequest, GetPrivateMessagesHistoryRequest, GetGroupMessagesHistoryRequest, MarkPrivateMessagesReadRequest, MarkGroupMessagesReadRequest } from '../../../core/types/api';
import type { ServerEvent, PrivateMessageNewEvent, GroupMessageNewEvent } from '../../../core/types/WebSocketTypes';
import { PageNames } from '../../../constants/AppConstants';
import { SystemAvatar } from '../../../components/SystemAvatar';
import { SystemLoadingView } from '../../../components/SystemLoadingView';
import { SystemToast } from '../../../components/SystemToast';
import { DesignConstants } from '../../../constants/DesignConstants';
import { AppIcon, IconSize, IconButton } from '../../../components/SystemIcon';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { MessageCacheService, MessageCacheItem } from '../../../core/database/MessageCacheService';
import { AuthAccountManager } from '../../../core/state/AuthAccountManager';
import { LoadingUtil } from '../../../core/utils/LoadingUtil';
import { BasicDataSource } from '../../../core/utils/BasicDataSource';
import { AppRouter, LoginRouteParam, GroupDetailRouteParam, ProfileRouteParam } from '../../../core/router/AppRouter';
import { SafeAreaUtils } from '../../../core/utils/SafeAreaUtils';
import { SystemUiController } from '../../../core/utils/SystemUiController';

const DOMAIN_NUMBER: number = 0xFF09;
const TAG: string = 'ChatPage';

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

@Component
export struct ChatPage {
  pathStack: NavPathStack = new NavPathStack();
  @StorageLink(KEY_CURRENT_USER) currentUser: MyProfile | null = null;
  @StorageLink('safeAreaTop') safeAreaTop: number = 0;
  @StorageLink('safeAreaBottom') safeAreaBottom: number = 0;
  
  @State messages: ChatMessage[] = [];
  private messagesDataSource: BasicDataSource<ChatMessage> = new BasicDataSource<ChatMessage>([]);
  @State inputText: string = '';
  @State isLoading: boolean = true;
  @State errorMessage: string = '';
  @State isSending: boolean = false;
  @State title: string = '聊天';
  @State hasMoreHistory: boolean = false;
  @State historyNextCursor: string = '';
  @State isLoadingMoreHistory: boolean = false;
  
  private chatId: string = '';
  private chatType: 'private_chat' | 'group_chat' = 'private_chat';
  private targetId?: string;
  private webSocketService = WebSocketService.getInstance();
  private messageCacheService = MessageCacheService.getInstance();
  private scroller: Scroller = new Scroller();

  aboutToAppear(): void {
    SystemUiController.useSurface({
      baseColor: DesignConstants.COLOR.PRIMARY_BG_LIGHT,      preferDarkContent: true
    });
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

  private handleNewMessage(newMessage: ChatMessage) {
    this.messages.push(newMessage);
    this.messagesDataSource.appendData([newMessage]);
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
    this.unsubscribeFromEvents();
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
          const request: GetPrivateMessagesHistoryRequest = { target_user_id: targetUserId, pagination: { limit: 20 } };
          const response = await ApiClient.getPrivateMessages(request);
          if (response.code === 0) {
            items = response.data?.items ?? [];
            nextCursor = response.data?.next_cursor;
          }
        } else {
          const request: GetGroupMessagesHistoryRequest = { group_id: this.chatId, pagination: { limit: 20 } };
          const response = await ApiClient.getGroupMessages(request);
          if (response.code === 0) {
            items = response.data?.items ?? [];
            nextCursor = response.data?.next_cursor;
          }
        }

        const newMessages = await Promise.all(items.reverse().map(item => this.convertMessage(item)));
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
      // 将更早的消息插入到列表开头
      this.messages = olderMessages.concat(this.messages);
      this.messagesDataSource.prependData(olderMessages);
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
      const messages = await Promise.all(cachedMessages.reverse().map(async item => {
        const rawData = JSON.parse(item.data) as Object;
        return await this.convertMessage(rawData as PrivateMessageItem | GroupMessageItem);
      }));
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

  private subscribeToEvents() {
    this.webSocketService.on('PrivateMessageNew', this.onPrivateMessageNew);
    this.webSocketService.on('GroupMessageNew', this.onGroupMessageNew);
  }

  private unsubscribeFromEvents() {
    this.webSocketService.off('PrivateMessageNew', this.onPrivateMessageNew);
    this.webSocketService.off('GroupMessageNew', this.onGroupMessageNew);
  }

  private scrollToBottom() {
    setTimeout(() => {
      this.scroller.scrollEdge(Edge.Bottom);
    }, 100);
  }

  private async sendMessage() {
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
        await LoadingUtil.withMinimumTime(async () => ApiClient.sendPrivateMessage(request), LoadingUtil.DEFAULT_ACTION_MS);
      } else {
        const request: SendGroupMessageRequest = {
          group_id: this.chatId,
          type: 'text',
          content: content
        };
        await LoadingUtil.withMinimumTime(async () => ApiClient.sendGroupMessage(request), LoadingUtil.DEFAULT_ACTION_MS);
      }

      this.inputText = '';
    } catch (error) {
      SystemToast.showError('发送失败', this.getUIContext());
    } finally {
      this.isSending = false;
    }
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
    await this.loadHistory();
    this.subscribeToEvents();
    const last = this.messages.length > 0 ? this.messages[this.messages.length - 1] : undefined;
    if (last) {
      this.markAsRead(last.id);
    }
  }

  build() {
    NavDestination() {
    Column() {
      Row() {
        IconButton({
          icon: AppIcon.BACK,
          iconSize: IconSize.LARGE,
          iconColor: $r('sys.color.ohos_id_color_text_primary'),
          bgColor: Color.Transparent,
          onButtonClick: () => {
            AppRouter.pop();
          }
        })

        Text(this.title)
          .fontSize(DesignConstants.FONT.SIZE.XL)
          .fontWeight(FontWeight.Medium)
          .fontColor($r('sys.color.ohos_id_color_text_primary'))
          .layoutWeight(1)
          .textAlign(TextAlign.Center)
        
        IconButton({
          icon: AppIcon.MORE,
          iconSize: IconSize.LARGE,
          iconColor: $r('sys.color.ohos_id_color_text_primary'),
          bgColor: Color.Transparent,
          onButtonClick: () => {
            if (this.chatType === 'group_chat') {
              const p = new GroupDetailRouteParam();
              p.groupId = this.chatId;
              AppRouter.push(PageNames.GROUP_DETAIL, p);
            } else if (this.targetId) {
              const p = new ProfileRouteParam();
              p.userId = this.targetId;
              AppRouter.push(PageNames.PROFILE, p);
            }
          }
        })
      }
      .width('100%')
      .padding({
        left: DesignConstants.SPACING.LG,
        right: DesignConstants.SPACING.LG,
        top: SafeAreaUtils.resolveTopPadding(this.safeAreaTop),
        bottom: DesignConstants.SPACING.SM
      })
      .backgroundColor($r('app.color.bg_secondary'))
      .expandSafeArea([SafeAreaType.SYSTEM], [SafeAreaEdge.TOP])
      .zIndex(1)

      if (this.isLoading) {
        SystemLoadingView({ message: '加载消息...' })
          .layoutWeight(1)
          .transition(TransitionEffect.OPACITY.animation({ duration: 300, curve: Curve.FastOutSlowIn }))
      } else if (this.messages.length === 0 && this.errorMessage) {
        Column({ space: DesignConstants.SPACING.LG }) {
          Text() {
            SymbolSpan($r(AppIcon.WARNING))
              .fontSize(48)
              .fontColor([$r('sys.color.ohos_id_color_text_secondary')])
          }
          Text(this.errorMessage)
            .fontSize(DesignConstants.FONT.SIZE.LG)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
            .textAlign(TextAlign.Center)
          Button('重试')
            .type(ButtonType.Capsule)
            .backgroundColor($r('sys.color.ohos_id_color_emphasize'))
            .fontColor(Color.White)
            .height(40)
            .onClick(() => { this.loadHistory(); })
        }
        .width('100%')
        .layoutWeight(1)
        .justifyContent(FlexAlign.Center)
        .transition(TransitionEffect.OPACITY.animation({ duration: 300, curve: Curve.FastOutSlowIn }))
      } else if (this.messages.length === 0) {
        Column({ space: DesignConstants.SPACING.LG }) {
          Text() {
            SymbolSpan($r(AppIcon.MESSAGE))
              .fontSize(48)
              .fontColor([$r('sys.color.ohos_id_color_text_tertiary')])
          }
          Text('暂无消息')
            .fontSize(DesignConstants.FONT.SIZE.LG)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
          Text('发送一条消息开始聊天吧')
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .fontColor($r('sys.color.ohos_id_color_text_tertiary'))
        }
        .width('100%')
        .layoutWeight(1)
        .justifyContent(FlexAlign.Center)
      } else {
      List({ scroller: this.scroller, initialIndex: this.messages.length - 1 }) {
        // 顶部加载更多历史指示器
        if (this.isLoadingMoreHistory) {
          ListItem() {
            Row({ space: DesignConstants.SPACING.SM }) {
              LoadingProgress().width(20).height(20)
              Text('加载更早消息...')
                .fontSize(DesignConstants.FONT.SIZE.SM)
                .fontColor($r('sys.color.ohos_id_color_text_secondary'))
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
      .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, top: DesignConstants.SPACING.LG, bottom: DesignConstants.SPACING.LG })
      .backgroundColor($r('sys.color.ohos_id_color_sub_background'))
      .onScrollIndex((start: number) => {
        // 当滚动到列表顶部附近时加载更早的消息
        if (start <= 1 && this.hasMoreHistory && !this.isLoadingMoreHistory) {
          this.loadMoreHistory();
        }
      })
      .transition(TransitionEffect.OPACITY.animation({ duration: 300, curve: Curve.FastOutSlowIn }))
      }

      Row() {
        TextInput({ text: this.inputText, placeholder: '发送消息...' })
          .layoutWeight(1)
          .height(40)
          .backgroundColor($r('sys.color.ohos_id_color_component_normal'))
          .borderRadius(20)
          .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG })
          .onChange((value) => {
            this.inputText = value;
          })
          .onSubmit(() => {
            this.sendMessage();
          })

        Button('发送')
          .type(ButtonType.Capsule)
          .backgroundColor($r('sys.color.ohos_id_color_emphasize'))
          .fontColor(Color.White)
          .height(36)
          .margin({ left: DesignConstants.SPACING.MD })
          .enabled(!this.isSending && this.inputText.trim().length > 0)
          .onClick(() => this.sendMessage())
      }
      .width('100%')
      .padding({
        left: DesignConstants.SPACING.LG,
        right: DesignConstants.SPACING.LG,
        top: DesignConstants.SPACING.SM,
        bottom: SafeAreaUtils.resolveBottomPadding(this.safeAreaBottom, 0, DesignConstants.SPACING.SM)
      })
      .backgroundColor($r('app.color.bg_secondary'))
    }
    .width('100%')
    .height('100%')
    .backgroundColor($r('sys.color.ohos_id_color_sub_background'))
    }
    .hideTitleBar(true)
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
  buildMessageItem(msg: ChatMessage) {
    if (msg.isSystem) {
      Row() {
        Text(msg.content)
          .fontSize(12)
          .fontColor($r('sys.color.ohos_id_color_text_secondary'))
          .backgroundColor($r('sys.color.ohos_id_color_component_normal'))
          .padding({ left: 12, right: 12, top: 4, bottom: 4 })
          .borderRadius(12)
      }
      .width('100%')
      .justifyContent(FlexAlign.Center)
      .margin({ top: 8, bottom: 8 })
    } else {
      Row() {
        if (!msg.isSelf) {
          SystemAvatar({
            avatarUrl: msg.senderAvatar,
            name: msg.senderName,
            avatarSize: 40
          })
          .margin({ right: 8 })
        }

        Column() {
          if (!msg.isSelf && this.chatType === 'group_chat') {
            Text(msg.senderName)
              .fontSize(12)
              .fontColor($r('sys.color.ohos_id_color_text_secondary'))
              .margin({ bottom: 4, left: 4 })
          }

          Row() {
            if (msg.type === 'image') {
              Image(msg.content)
                .width(200)
                .borderRadius(8)
                .margin(4)
            } else {
              Text(msg.content)
                .fontSize(16)
                .fontColor(msg.isSelf ? Color.White : $r('sys.color.ohos_id_color_text_primary'))
                .backgroundColor(msg.isSelf ? $r('sys.color.ohos_id_color_emphasize') : $r('app.color.bg_secondary'))
                .padding(12)
                .borderRadius(16)
                .constraintSize({ maxWidth: '70%' })
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
            avatarSize: 40
          })
          .margin({ left: 8 })
        }
      }
      .width('100%')
      .justifyContent(msg.isSelf ? FlexAlign.End : FlexAlign.Start)
      .alignItems(VerticalAlign.Top)
      .margin({ bottom: 16 })
    }
  }
}

@Builder
export function ChatPageBuilder() {
  ChatPage()
}
