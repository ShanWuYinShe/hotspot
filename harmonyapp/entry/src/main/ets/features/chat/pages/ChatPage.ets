/**
 * 聊天详情页面
 * 重构：使用 List 组件优化消息列表，支持自动滚动
 * 优化：集成消息历史缓存 (RDB)
 */

import { GlobalStateManager, KEY_CURRENT_USER } from '../../../core/state/GlobalStateManager';
import { ApiClient } from '../../../core/api/ApiClient';
import { WebSocketService } from '../../../core/network/WebSocketService';
import { MyProfileData, MessageType, PrivateMessageItem, GroupMessageItem, SendMessageType, UserInfo } from '../../../core/types/api';
import type { ServerEvent, PrivateMessageNewEvent, GroupMessageNewEvent } from '../../../core/types/WebSocketTypes';
import { PageNames } from '../../../constants/AppConstants';
import { SystemAvatar } from '../../../components/SystemAvatar';
import { SystemLoadingView } from '../../../components/SystemLoadingView';
import { SystemToast } from '../../../components/SystemToast';
import { DesignConstants } from '../../../constants/DesignConstants';
import { AppIcon, SystemIcon, IconSize, IconButton } from '../../../components/SystemIcon';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { MessageCacheService, MessageCacheItem } from '../../../core/database/MessageCacheService';
import { AuthAccountManager } from '../../../core/state/AuthAccountManager';

const DOMAIN_NUMBER: number = 0xFF09;
const TAG: string = 'ChatPage';

// 统一消息接口用于UI展示
interface ChatMessage {
  id: string;
  content: string;
  senderId: number;
  senderName: string;
  senderAvatar?: string;
  type: MessageType;
  sentAt: string;
  isSelf: boolean;
  isSystem: boolean;
  rawData?: unknown; // 原始数据，用于缓存
}

interface ChatParams {
  chatId: string;
  chatType: 'private_chat' | 'group_chat';
  title: string;
  targetId?: number; // 私聊对方的用户ID
}

@Entry
@Component
struct ChatPage {
  @StorageLink(KEY_CURRENT_USER) currentUser: MyProfileData | null = null;
  
  @State messages: ChatMessage[] = [];
  @State inputText: string = '';
  @State isLoading: boolean = true;
  @State isSending: boolean = false;
  @State title: string = '聊天';
  
  private chatId: string = '';
  private chatType: 'private_chat' | 'group_chat' = 'private_chat';
  private targetId?: number;
  private webSocketService = WebSocketService.getInstance();
  private messageCacheService = MessageCacheService.getInstance();
  private scroller: Scroller = new Scroller();

  private onPrivateMessageNew = async (event: ServerEvent): Promise<void> => {
    if (event.type !== 'PrivateMessageNew') return;
    if (this.chatType !== 'private_chat') return;

    const payload = (event as PrivateMessageNewEvent).payload;
    const currentUserId = await AuthAccountManager.getCurrentUserId();
    const targetId = this.targetId;

    if (!currentUserId || !targetId) return;

    // Check relevance
    const isRelevant = 
      (payload.sender.user_id === currentUserId && payload.target_user_id === targetId) ||
      (payload.sender.user_id === targetId && payload.target_user_id === currentUserId);
    
    if (!isRelevant) return;

    this.handleNewMessage({
      id: payload.message_id,
      content: payload.media?.url || payload.content || (payload.type === 'image' ? '[图片]' : '[非文本消息]'),
      senderId: payload.sender.user_id,
      senderName: payload.sender.nickname,
      senderAvatar: payload.sender.avatar_url,
      type: payload.type,
      sentAt: payload.sent_at,
      isSelf: payload.sender.user_id === currentUserId,
      isSystem: payload.type === 'system',
      rawData: payload
    });
  }

  private onGroupMessageNew = async (event: ServerEvent): Promise<void> => {
    if (event.type !== 'GroupMessageNew') return;
    if (this.chatType !== 'group_chat') return;

    const payload = (event as GroupMessageNewEvent).payload;
    if (payload.group_id !== this.chatId) return;

    const currentUserId = await AuthAccountManager.getCurrentUserId();
    this.handleNewMessage({
      id: payload.message_id,
      content: payload.media?.url || payload.content || (payload.type === 'image' ? '[图片]' : '[非文本消息]'),
      senderId: payload.sender.user_id,
      senderName: payload.sender.nickname,
      senderAvatar: payload.sender.avatar_url,
      type: payload.type,
      sentAt: payload.sent_at,
      isSelf: payload.sender.user_id === currentUserId,
      isSystem: payload.type === 'system',
      rawData: payload
    });
  }

  private handleNewMessage(newMessage: ChatMessage) {
    this.messages.push(newMessage);
    this.scrollToBottom();

    // 如果当前就在这个会话里，收到新消息后立即标记已读（避免会话列表一直显示未读）
    if (!newMessage.isSelf) {
      this.markAsRead(newMessage.id);
    }

    // 追加缓存
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

  async aboutToAppear() {
    try {
      const routerInstance = this.getUIContext().getRouter();
      const params = routerInstance.getParams() as ChatParams;
      if (params) {
        this.chatId = params.chatId;
        this.chatType = params.chatType;
        this.title = params.title;
        this.targetId = params.targetId;
        
        await this.loadHistory();
        this.subscribeToEvents();

        // 进入会话后尝试标记已读（使用当前页面最后一条消息作为 up_to）
        const last = this.messages.length > 0 ? this.messages[this.messages.length - 1] : undefined;
        if (last) {
          this.markAsRead(last.id);
        }
      }
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `获取参数失败: ${JSON.stringify(error)}`);
    }
  }

  aboutToDisappear() {
    this.unsubscribeFromEvents();
  }

  private async loadHistory() {
    this.isLoading = true;
    
    // 1. 先尝试加载缓存
    await this.loadFromCache();

    try {
      let items: (PrivateMessageItem | GroupMessageItem)[] = [];
      
      if (this.chatType === 'private_chat') {
        const response = await ApiClient.getPrivateMessages({ id: this.chatId, limit: 20 });
        if (response.code === 0 && response.data) {
          items = response.data.items;
        }
      } else {
        const response = await ApiClient.getGroupMessages({ id: this.chatId, limit: 20 });
        if (response.code === 0 && response.data) {
          items = response.data.items;
        }
      }

      const newMessages = await Promise.all(items.reverse().map(item => this.convertMessage(item)));
      this.messages = newMessages;
      
      // 3. 更新缓存
      this.updateCache(newMessages);
      
      // 滚动到底部
      this.scrollToBottom();

    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `加载历史消息失败: ${error}`);
      if (this.messages.length === 0) {
        SystemToast.showError('加载消息失败', this.getUIContext());
      }
    } finally {
      this.isLoading = false;
    }
  }

  private async loadFromCache() {
    const cachedMessages = await this.messageCacheService.getCachedMessages(this.chatId);
    if (cachedMessages.length > 0) {
      // 缓存是倒序取出的（最新的在前），需要反转回正序（旧的在前）
      const messages = await Promise.all(cachedMessages.reverse().map(async item => {
        const rawData = JSON.parse(item.data);
        return await this.convertMessage(rawData);
      }));
      this.messages = messages;
      this.scrollToBottom();
      hilog.info(DOMAIN_NUMBER, TAG, `Loaded ${messages.length} messages from cache`);
    }
  }

  private async updateCache(messages: ChatMessage[]) {
    // 转换UI模型回缓存模型
    const cacheItems: MessageCacheItem[] = messages.map(msg => ({
      id: msg.id,
      chatId: this.chatId,
      type: msg.type,
      senderId: msg.senderId,
      content: msg.content,
      sentAt: msg.sentAt,
      data: JSON.stringify(msg.rawData ?? null)
    }));
    await this.messageCacheService.cacheMessages(this.chatId, cacheItems);
  }

  private async convertMessage(item: PrivateMessageItem | GroupMessageItem): Promise<ChatMessage> {
    const currentUserId = await AuthAccountManager.getCurrentUserId();
    
    // PrivateMessageItem 使用 sender_id，GroupMessageItem 使用 sender
    let senderId: number;
    let senderName: string;
    let senderAvatar: string | undefined;
    let isSelf: boolean;
    
    if ('sender_id' in item) {
      // PrivateMessageItem
      senderId = item.sender_id;
      isSelf = senderId === currentUserId;
      // 对于私聊消息，sender_id 需要从其他地方获取用户信息
      // 这里暂时使用占位符，实际应该从缓存或API获取
      senderName = `User ${senderId}`;
      senderAvatar = undefined;
    } else {
      // GroupMessageItem
      senderId = item.sender.user_id;
      senderName = item.sender.nickname;
      senderAvatar = item.sender.avatar_url;
      isSelf = senderId === currentUserId;
    }
    
    return {
      id: item.message_id,
      content: item.media?.url || item.content || (item.type === 'image' ? '[图片]' : '[非文本消息]'),
      senderId: senderId,
      senderName: senderName,
      senderAvatar: senderAvatar,
      type: item.type,
      sentAt: item.sent_at,
      isSelf: isSelf,
      isSystem: item.type === 'system',
      rawData: item // 保存原始数据以便缓存
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
      try {
        const routerInstance = this.getUIContext().getRouter();
        await routerInstance.pushUrl({
          url: `features/auth/pages/${PageNames.LOGIN}`,
          params: { reason: 'action_required', action: '发送消息' }
        });
      } catch (error) {
        // ignore
      }
      return;
    }

    const content = this.inputText.trim();
    if (!content) return;

    this.isSending = true;
    try {
      if (this.chatType === 'private_chat') {
        await ApiClient.sendPrivateMessage({
          id: this.chatId,
          type: 'text',
          content: content
        });
      } else {
        await ApiClient.sendGroupMessage({
          id: this.chatId,
          type: 'text',
          content: content
        });
      }
      
      this.inputText = '';
      // 遵循 @harmony_app_rule.mdc 消息发送规范：
      // 1. API 仅负责发送消息，不处理本地数据更新
      // 2. 消息上屏完全依赖 WebSocket 推送的 ChatMessageNew 事件
      // 3. 确保显示的数据（如媒体URL、时间戳）与服务端完全一致
    } catch (error) {
      SystemToast.showError('发送失败', this.getUIContext());
    } finally {
      this.isSending = false;
    }
  }

  /**
   * 标记当前会话已读（API + WS），并通知消息列表刷新
   */
  private markAsRead(upToMessageId: string) {
    if (!upToMessageId) return;

    // 1) WebSocket 读状态（实时）
    this.webSocketService.markAsRead(this.chatType, this.chatId, upToMessageId);

    // 2) HTTP 标记已读（权威，影响 unread_count）
    (async () => {
      try {
        if (this.chatType === 'private_chat') {
          await ApiClient.markPrivateMessagesRead({ id: this.chatId, up_to: upToMessageId });
        } else {
          await ApiClient.markGroupMessagesRead({ id: this.chatId, up_to: upToMessageId });
        }
        GlobalStateManager.getInstance().notifyMessageSync();
      } catch (e) {
        // ignore：不阻塞 UI
      }
    })();
  }

  build() {
    Column() {
      // 标题栏
      Row() {
        IconButton({
          icon: AppIcon.BACK,
          iconSize: IconSize.LARGE,
          iconColor: DesignConstants.COLOR.TEXT_PRIMARY_LIGHT,
          bgColor: Color.Transparent,
          onButtonClick: () => {
            const routerInstance = this.getUIContext().getRouter();
            routerInstance.back();
          }
        })

        Text(this.title)
          .fontSize(DesignConstants.FONT.SIZE.XL)
          .fontWeight(FontWeight.Medium)
          .fontColor(DesignConstants.COLOR.TEXT_PRIMARY_LIGHT)
          .layoutWeight(1)
          .textAlign(TextAlign.Center)
        
        IconButton({
          icon: AppIcon.MORE,
          iconSize: IconSize.LARGE,
          iconColor: DesignConstants.COLOR.TEXT_PRIMARY_LIGHT,
          bgColor: Color.Transparent,
          onButtonClick: () => {
            if (this.chatType === 'group_chat') {
              const routerInstance = this.getUIContext().getRouter();
              routerInstance.pushUrl({
                url: `features/group/pages/${PageNames.GROUP_DETAIL}`,
                params: { groupId: this.chatId }
              });
            } else if (this.targetId) {
              const routerInstance = this.getUIContext().getRouter();
              routerInstance.pushUrl({
                url: `features/profile/pages/${PageNames.PROFILE}`,
                params: { userId: this.targetId }
              });
            }
          }
        })
      }
      .width('100%')
      .height(56)
      .padding({ left: 8, right: 8 })
      .backgroundColor(DesignConstants.COLOR.PRIMARY_BG_LIGHT)
      .zIndex(1)

      // 消息列表
      List({ scroller: this.scroller, initialIndex: this.messages.length - 1 }) {
        ForEach(this.messages, (msg: ChatMessage) => {
          ListItem() {
            this.buildMessageItem(msg)
          }
        }, (msg: ChatMessage) => msg.id)
      }
      .width('100%')
      .layoutWeight(1)
      .padding({ left: 16, right: 16, top: 16, bottom: 16 })
      .backgroundColor(DesignConstants.COLOR.SECONDARY_BG_LIGHT)

      // 输入框
      Row() {
        TextInput({ text: this.inputText, placeholder: '发送消息...' })
          .layoutWeight(1)
          .height(40)
          .backgroundColor(DesignConstants.COLOR.TERTIARY_BG_LIGHT)
          .borderRadius(20)
          .padding({ left: 16, right: 16 })
          .onChange((value) => {
            this.inputText = value;
          })
          .onSubmit(() => {
            this.sendMessage();
          })

        Button('发送')
          .type(ButtonType.Capsule)
          .backgroundColor(DesignConstants.COLOR.BRAND_PRIMARY)
          .fontColor(Color.White)
          .height(36)
          .margin({ left: 12 })
          .enabled(!this.isSending && this.inputText.trim().length > 0)
          .onClick(() => this.sendMessage())
      }
      .width('100%')
      .padding({ left: 16, right: 16, top: 8, bottom: 8 })
      .backgroundColor(DesignConstants.COLOR.PRIMARY_BG_LIGHT)
    }
    .width('100%')
    .height('100%')
    .backgroundColor(DesignConstants.COLOR.SECONDARY_BG_LIGHT)
  }

  @Builder
  buildMessageItem(msg: ChatMessage) {
    if (msg.isSystem) {
      Row() {
        Text(msg.content)
          .fontSize(12)
          .fontColor(DesignConstants.COLOR.TEXT_TERTIARY_LIGHT)
          .backgroundColor(DesignConstants.COLOR.TERTIARY_BG_LIGHT)
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
            nickname: msg.senderName,
            size: 40
          })
          .margin({ right: 8 })
        }

        Column() {
          if (!msg.isSelf && this.chatType === 'group_chat') {
            Text(msg.senderName)
              .fontSize(12)
              .fontColor(DesignConstants.COLOR.TEXT_SECONDARY_LIGHT)
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
                .fontColor(msg.isSelf ? Color.White : DesignConstants.COLOR.TEXT_PRIMARY_LIGHT)
                .backgroundColor(msg.isSelf ? DesignConstants.COLOR.BRAND_PRIMARY : DesignConstants.COLOR.PRIMARY_BG_LIGHT)
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
            nickname: msg.senderName,
            size: 40
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
