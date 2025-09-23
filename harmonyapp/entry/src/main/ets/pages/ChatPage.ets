/**
 * 聊天页面
 * 支持私聊和群聊功能
 */

import { GlobalStateManager, GlobalState } from '../common/state/GlobalStateManager';
import { ApiClient } from '../common/api/ApiClient';
import { WebSocketService } from '../common/websocket/WebSocketService';
import { 
  NewMessageData,
  TypingIndicatorData, 
  MessageReadData,
  UserOnlineData,
  UserOfflineData,
  GroupMemberJoinedData,
  GroupMemberLeftData,
  SystemNotificationData,
  WebSocketMessageType,
  WebSocketMessageConverter,
  WebSocketParseResult
} from '../common/types/WebSocketTypes';
import { SystemAvatar } from '../components/SystemAvatar';
import { SystemLoadingView } from '../components/SystemLoadingView';
import { SystemToast } from '../components/SystemToast';
import { IconButton, AppIcon, IconSize } from '../components/SystemIcon';
import { hilog } from '@kit.PerformanceAnalysisKit';

const DOMAIN_NUMBER: number = 0xFF04;
const TAG: string = 'ChatPage';

interface ChatParams {
  groupId: string;        // 统一使用群组ID，私聊和群聊都是群组
  groupName?: string;     // 群组名称（私聊群组可能为空或显示对方昵称）
  groupType?: string;     // 群组类型：'public_chat' | 'private_chat'
}

/**
 * 聊天页面显示用的消息接口
 * 统一WebSocket实时消息和API历史消息的显示格式
 */
interface ChatMessage {
  message_id: string;       // 消息ID
  group_id: string;         // 群组ID
  sender_id?: string;       // 发送者ID
  sender_name?: string;     // 发送者昵称
  sender_avatar?: string;   // 发送者头像URL
  content: string;          // 消息内容
  message_type: string;     // 消息类型
  reply_to_message_id?: string; // 回复的消息ID
  mentioned_users: string[]; // 提及的用户ID数组
  sent_at: string;          // 发送时间
  // WebSocket特有字段
  group_type?: string;      // 群组类型（WebSocket消息有）
  system_event_type?: string; // 系统事件类型（WebSocket消息有）
  affected_users?: string[]; // 受影响的用户ID数组（WebSocket消息有）
  file_url?: string;        // 文件URL（WebSocket消息有）
  file_name?: string;       // 文件名（WebSocket消息有）
  file_size?: number;       // 文件大小（WebSocket消息有）
  latitude?: number;        // 纬度（WebSocket消息有）
  longitude?: number;       // 经度（WebSocket消息有）
  location_name?: string;   // 位置名称（WebSocket消息有）
  // API特有字段
  is_edited?: boolean;      // 是否已编辑（API消息有）
  reaction_count?: number;  // 反应数量（API消息有）
  edited_at?: string;       // 编辑时间（API消息有）
}


@Entry
@Component
struct ChatPage {
  @State globalState: GlobalState = GlobalStateManager.getInstance().getState();
  private stateManager = GlobalStateManager.getInstance();
  @State messages: ChatMessage[] = [];
  @State inputText: string = '';
  @State isLoading: boolean = true;
  @State isSending: boolean = false;
  @State isTyping: boolean = false;
  private chatParams: ChatParams = { groupId: '' };
  private webSocketService = WebSocketService.getInstance();
  private scrollController: Scroller = new Scroller();

  async aboutToAppear() {
    // 订阅状态变化
    this.stateManager.subscribe('chat_page', (state: GlobalState) => {
      this.globalState = state;
    });

    // 获取路由参数
    try {
      const routerInstance = this.getUIContext().getRouter();
      const params = routerInstance.getParams() as Record<string, Object>;
      if (params && params.groupId) {
        this.chatParams = {
          groupId: params.groupId as string,
          groupName: params.groupName as string,
          groupType: params.groupType as string
        };
        
        await this.loadChatHistory();
        this.setupWebSocketListeners();
      } else {
        SystemToast.showError('缺少群组参数', this.getUIContext());
        const routerInstance = this.getUIContext().getRouter();
        routerInstance.back();
      }
    } catch (error) {
      SystemToast.showError('初始化聊天失败', this.getUIContext());
    }
  }

  aboutToDisappear() {
    this.stateManager.unsubscribe('chat_page');
    this.removeWebSocketListeners();
  }

  /**
   * 加载聊天历史（统一接口）1· @
   */
  private async loadChatHistory(): Promise<void> {
    this.isLoading = true;
    try {
      // 统一使用群组ID获取消息历史，不再区分私聊和群聊
      const response = await ApiClient.getHistory({
        group_id: this.chatParams.groupId,
        limit: 50,
        include_system_messages: true
      });

      if (response && response.code === 0 && response.data) {
        this.messages = response.data.items.reverse(); // 消息按时间正序显示
        this.scrollToBottom();
        
        // 标记消息为已读
        await this.markMessagesAsRead();
      } else if (response) {
        hilog.error(DOMAIN_NUMBER, TAG, `加载聊天记录失败: ${response.msg}`);
        SystemToast.showError(response.msg || '加载聊天记录失败', this.getUIContext());
      }
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `加载聊天记录异常: ${JSON.stringify(error)}`);
      SystemToast.showError('加载聊天记录失败', this.getUIContext());
    } finally {
      this.isLoading = false;
    }
  }

  /**
   * 标记消息为已读
   */
  private async markMessagesAsRead(): Promise<void> {
    try {
      if (this.messages.length > 0) {
        const lastMessage = this.messages[this.messages.length - 1];
        await ApiClient.markRead({
          group_id: this.chatParams.groupId,
          message_ids: [lastMessage.message_id]
        });
      }
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `标记已读失败: ${JSON.stringify(error)}`);
    }
  }

  /**
   * 设置WebSocket监听器（根据后端文档规范）
   */
  private setupWebSocketListeners(): void {
    // 监听所有WebSocket消息
    this.webSocketService.onMessage('message', (rawData: Record<string, Object>) => {
      // 使用转换工具解析消息
      const parseResult = WebSocketMessageConverter.parseWebSocketMessage(rawData);
      if (parseResult.message) {
        this.handleWebSocketMessage(parseResult);
      }
    });
  }

  /**
   * 处理WebSocket消息（根据消息类型分发）
   */
  private handleWebSocketMessage(parseResult: WebSocketParseResult): void {
    if (!parseResult.message) {
      return;
    }

    switch (parseResult.message.type) {
      case WebSocketMessageType.NEW_MESSAGE:
        if (parseResult.newMessageData) {
          this.handleNewMessage(parseResult.newMessageData);
        }
        break;
      case WebSocketMessageType.TYPING_INDICATOR:
        if (parseResult.typingIndicatorData) {
          this.handleTypingIndicator(parseResult.typingIndicatorData);
        }
        break;
      case WebSocketMessageType.MESSAGE_READ:
        if (parseResult.messageReadData) {
          this.handleMessageRead(parseResult.messageReadData);
        }
        break;
      case WebSocketMessageType.USER_ONLINE:
        if (parseResult.userOnlineData) {
          this.handleUserStatusChange(parseResult.userOnlineData);
        }
        break;
      case WebSocketMessageType.USER_OFFLINE:
        if (parseResult.userOfflineData) {
          this.handleUserStatusChange(parseResult.userOfflineData);
        }
        break;
      case WebSocketMessageType.GROUP_MEMBER_JOINED:
        if (parseResult.groupMemberJoinedData) {
          this.handleGroupEvent(parseResult.groupMemberJoinedData);
        }
        break;
      case WebSocketMessageType.GROUP_MEMBER_LEFT:
        if (parseResult.groupMemberLeftData) {
          this.handleGroupEvent(parseResult.groupMemberLeftData);
        }
        break;
      case WebSocketMessageType.SYSTEM_NOTIFICATION:
        if (parseResult.systemNotificationData) {
          this.handleSystemNotification(parseResult.systemNotificationData);
        }
        break;
        default:
          hilog.warn(DOMAIN_NUMBER, TAG, `未处理的WebSocket消息类型: ${parseResult.message.type}`);
    }
  }

  /**
   * 处理新消息（WebSocket实时消息）
   */
  private handleNewMessage(messageData: NewMessageData): void {
    if (this.isRelevantMessage(messageData)) {
      // 将WebSocket消息转换为ChatMessage格式
      const message: NewMessageData = {
        message_id: messageData.message_id,
        group_id: messageData.group_id,
        sender_id: messageData.sender_id,
        sender_name: messageData.sender_name,
        sender_avatar: messageData.sender_avatar,
        content: messageData.content,
        message_type: messageData.message_type,
        reply_to_message_id: messageData.reply_to_message_id,
        mentioned_users: messageData.mentioned_users || [],
        sent_at: messageData.sent_at,
        // WebSocket特有字段
        group_type: messageData.group_type,
        system_event_type: messageData.system_event_type,
        affected_users: messageData.affected_users,
        file_url: messageData.file_url,
        file_name: messageData.file_name,
        file_size: messageData.file_size,
        latitude: messageData.latitude,
        longitude: messageData.longitude,
        location_name: messageData.location_name
      };
      this.messages.push(message as ChatMessage);
      this.scrollToBottom();
    }
  }

  /**
   * 处理输入状态指示器
   */
  private handleTypingIndicator(typingData: TypingIndicatorData): void {
    if (this.isRelevantTyping(typingData)) {
      this.isTyping = typingData.is_typing;
    }
  }

  /**
   * 处理消息已读状态
   */
  private handleMessageRead(data: MessageReadData): void {
    // 可以在这里更新消息的已读状态
    hilog.debug(DOMAIN_NUMBER, TAG, `消息已读状态更新: ${data.message_id}`);
  }

  /**
   * 处理用户在线状态变化
   */
  private handleUserStatusChange(data: UserOnlineData | UserOfflineData): void {
    // 可以在这里更新用户的在线状态
    hilog.debug(DOMAIN_NUMBER, TAG, `用户状态变化: ${data.user_id}`);
  }

  /**
   * 处理群组事件
   */
  private handleGroupEvent(data: GroupMemberJoinedData | GroupMemberLeftData): void {
    // 可以在这里处理群组成员变化、角色变更等事件
    hilog.debug(DOMAIN_NUMBER, TAG, '群组事件处理');
  }

  /**
   * 处理系统通知
   */
  private handleSystemNotification(data: SystemNotificationData): void {
    // 可以在这里显示系统通知
    hilog.debug(DOMAIN_NUMBER, TAG, `系统通知: ${data.title}`);
  }

  /**
   * 移除WebSocket监听器
   */
  private removeWebSocketListeners(): void {
    // WebSocket服务会在组件销毁时自动清理
  }

  /**
   * 判断消息是否与当前聊天相关（统一判断）
   */
  private isRelevantMessage(message: NewMessageData): boolean {
    // 统一架构：所有消息都基于群组，直接比较群组ID
    return message.group_id === this.chatParams.groupId;
  }

  /**
   * 判断打字状态是否与当前聊天相关（统一判断）
   */
  private isRelevantTyping(data: TypingIndicatorData): boolean {
    // 统一架构：所有打字状态都基于群组，直接比较群组ID
    // 排除自己的打字状态
    return data.group_id === this.chatParams.groupId &&
           data.user_id !== this.globalState.currentUser?.user_id;
  }

  /**
   * 发送消息
   */
  private async sendMessage(): Promise<void> {
    const content = this.inputText.trim();
    if (!content || this.isSending) return;

    if (!this.globalState.currentUser) {
      SystemToast.show('请先登录', this.getUIContext());
      return;
    }

    this.isSending = true;
    const originalText = this.inputText;
    this.inputText = ''; // 立即清空输入框

    try {
      // 统一发送消息接口，不再区分私聊和群聊
      const response = await ApiClient.sendMessage({
        group_id: this.chatParams.groupId,
        content: content,
        message_type: 'text'
      });

      if (response && response.code === 0 && response.data) {
        // 消息发送成功，通过WebSocket会收到消息推送
        // 这里不需要手动添加到消息列表
        hilog.info(DOMAIN_NUMBER, TAG, `消息发送成功: ${response.data.message_id}`);
      } else if (response) {
        SystemToast.showError(response.msg || '发送失败', this.getUIContext());
        this.inputText = originalText; // 恢复输入内容
      }
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `发送消息异常: ${JSON.stringify(error)}`);
      SystemToast.showError('发送失败', this.getUIContext());
      this.inputText = originalText; // 恢复输入内容
    } finally {
      this.isSending = false;
    }
  }

  /**
   * 滚动到底部
   */
  private scrollToBottom(): void {
    setTimeout(() => {
      this.scrollController.scrollEdge(Edge.Bottom);
    }, 100);
  }

  /**
   * 获取聊天标题（统一处理）
   */
  private getChatTitle(): string {
    // 根据群组类型和名称显示标题
    if (this.chatParams.groupName) {
      return this.chatParams.groupName;
    } else if (this.chatParams.groupType === 'private_chat') {
      return '私聊';
    } else {
      return '群聊';
    }
  }

  /**
   * 构建消息项
   */
  @Builder
  buildMessageItem(message: ChatMessage) {
    Row() {
      if (!(message.sender_id === this.globalState.currentUser?.user_id)) {
        // 其他人的消息，显示头像
        SystemAvatar({
          avatarUrl: undefined,
          nickname: message.sender_name,
          avatarSize: 36
        })
          .margin({ right: 8 })
      }

      Column() {
        // 发送者名称（群聊中显示，私聊中不显示）
        if (this.chatParams.groupType === 'public_chat' && !(message.sender_id === this.globalState.currentUser?.user_id)) {
          Text(message.sender_name || '未知用户')
            .fontSize(12)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
            .margin({ bottom: 4 })
            .width('100%')
            .textAlign((message.sender_id === this.globalState.currentUser?.user_id) ? TextAlign.End : TextAlign.Start)
        }

        // 消息内容
        Text(message.content)
          .fontSize(16)
          .fontColor((message.sender_id === this.globalState.currentUser?.user_id) ? Color.White : $r('sys.color.ohos_id_color_text_primary'))
          .backgroundColor((message.sender_id === this.globalState.currentUser?.user_id) ? $r('sys.color.ohos_id_color_emphasize') : $r('sys.color.ohos_id_color_background'))
          .padding({ left: 12, right: 12, top: 8, bottom: 8 })
          .borderRadius(12)
          .maxLines(10)
          .textOverflow({ overflow: TextOverflow.Ellipsis })

        // 发送时间
        Text(new Date(message.sent_at).toLocaleTimeString())
          .fontSize(10)
          .fontColor($r('sys.color.ohos_id_color_text_secondary'))
          .margin({ top: 4 })
          .width('100%')
          .textAlign((message.sender_id === this.globalState.currentUser?.user_id) ? TextAlign.End : TextAlign.Start)
      }
      .alignItems((message.sender_id === this.globalState.currentUser?.user_id) ? HorizontalAlign.End : HorizontalAlign.Start)
      .layoutWeight(1)

      if (message.sender_id === this.globalState.currentUser?.user_id) {
        // 我的消息，显示头像
        SystemAvatar({
          avatarUrl: this.globalState.currentUser?.avatar_url,
          nickname: this.globalState.currentUser?.nickname || '',
          avatarSize: 36
        })
          .margin({ left: 8 })
      }
    }
    .width('100%')
    .justifyContent((message.sender_id === this.globalState.currentUser?.user_id) ? FlexAlign.End : FlexAlign.Start)
    .padding({ left: 16, right: 16, top: 8, bottom: 8 })
  }

  build() {
    Column() {
      // 标题栏
      Row() {
        IconButton({
          icon: AppIcon.BACK,
          iconSize: IconSize.LARGE,
          bgColor: '#00000000',
          onButtonClick: () => {
            const routerInstance = this.getUIContext().getRouter();
            routerInstance.back();
          }
        })

        Text(this.getChatTitle())
          .fontSize(18)
          .fontWeight(FontWeight.Medium)
          .layoutWeight(1)
          .textAlign(TextAlign.Center)

        // 更多操作按钮
        IconButton({
          icon: AppIcon.MORE,
          iconSize: IconSize.LARGE,
          bgColor: '#00000000',
          onButtonClick: () => {
            SystemToast.show('更多功能待开发', this.getUIContext());
          }
        })
      }
      .width('100%')
      .height(56)
      .padding({ left: 16, right: 16 })
      .backgroundColor($r('sys.color.ohos_id_color_background'))

      // 消息列表
      if (this.isLoading) {
        SystemLoadingView({ message: '加载聊天记录...' })
          .layoutWeight(1)
      } else {
        Scroll(this.scrollController) {
          Column() {
            if (this.messages.length === 0) {
              Column() {
                Text('暂无消息')
                  .fontSize(14)
                  .fontColor($r('sys.color.ohos_id_color_text_secondary'))
                
                Text('开始你们的第一次对话吧')
                  .fontSize(12)
                  .fontColor($r('sys.color.ohos_id_color_text_secondary'))
                  .margin({ top: 8 })
              }
              .width('100%')
              .height(200)
              .justifyContent(FlexAlign.Center)
            } else {
              ForEach(this.messages, (message: ChatMessage) => {
                this.buildMessageItem(message)
              })
            }

            // 打字指示器
            if (this.isTyping) {
              Row() {
                Text('对方正在输入...')
                  .fontSize(12)
                  .fontColor($r('sys.color.ohos_id_color_text_secondary'))
                  .margin({ left: 52 })
              }
              .width('100%')
              .padding({ top: 8, bottom: 8 })
            }
          }
          .width('100%')
        }
        .layoutWeight(1)
        .scrollBar(BarState.Auto)
        .backgroundColor($r('sys.color.ohos_id_color_sub_background'))
      }

      // 输入栏
      Row() {
        TextInput({ placeholder: '输入消息...', text: this.inputText })
          .onChange((value: string) => {
            this.inputText = value;
          })
          .layoutWeight(1)
          .backgroundColor($r('sys.color.ohos_id_color_background'))
          .borderRadius(20)
          .padding({ left: 16, right: 16 })

        IconButton({
          icon: AppIcon.SEND,
          iconSize: IconSize.MEDIUM,
          iconColor: '#FFFFFF',
          bgColor: this.inputText.trim().length > 0 && !this.isSending ? 
            $r('sys.color.ohos_id_color_emphasize') : 
            $r('sys.color.ohos_id_color_secondary'),
          onButtonClick: () => {
            if (this.inputText.trim().length > 0 && !this.isSending) {
              this.sendMessage();
            }
          }
        })
        .margin({ left: 8 })
      }
      .width('100%')
      .padding(16)
      .backgroundColor($r('sys.color.ohos_id_color_background'))
    }
    .width('100%')
    .height('100%')
    .backgroundColor($r('sys.color.ohos_id_color_sub_background'))
  }
}
