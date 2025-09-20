/**
 * 消息页面 - 会话列表
 * 显示所有会话（私聊和群聊统一架构）
 * 
 * 设计理念：
 * - 私聊和群聊本质上都是群组，只是通过 group_type 字段区分
 * - 'private_chat': 私聊群组，不可搜索，通常只有2个成员
 * - 'public_chat': 群聊群组，可被搜索发现，支持多成员
 * - UI层面仅在显示上有所区别（图标、默认名称等），逻辑完全统一
 */

import { GlobalStateManager, GlobalState } from '../common/state/GlobalStateManager';
import { ApiClient } from '../common/api/ApiClient';
import { PageNames } from '../constants/AppConstants';
import { SystemLoadingView } from '../components/SystemLoadingView';
import { SystemAvatar } from '../components/SystemAvatar';
import { SystemToast } from '../components/SystemToast';
import { UserResponse, GetConversationsResponse, SearchUserByLocationResponse } from '../common/types/BackendApiTypes';
import { hilog } from '@kit.PerformanceAnalysisKit';

const DOMAIN_NUMBER: number = 0xFF05;
const TAG: string = 'MessagePage';

interface LastMessage {
  message_id: string;
  sender_id: string;
  sender_name: string;
  content: string;
  message_type: string;
  sent_at: string;
}

interface Conversation {
  group_id: string;
  group_name: string;
  group_type: string;  // 'public_chat' | 'private_chat'
  group_avatar?: string;
  last_message?: LastMessage;
  unread_count: number;
  last_activity_at: string;
  is_muted: boolean;
  is_archived: boolean;
  member_count: number;
}


@Entry
@Component
struct MessagePage {
  @State globalState: GlobalState = GlobalStateManager.getInstance().getState();
  private stateManager = GlobalStateManager.getInstance();
  @State conversations: Conversation[] = [];
  @State isLoading: boolean = false;
  @State errorMessage: string = '';

  async aboutToAppear() {
    // 订阅状态变化
    this.stateManager.subscribe('message_page', (state: GlobalState) => {
      this.globalState = state;
    });
    
    // 如果用户已登录，加载会话列表
    if (this.globalState.isLoggedIn) {
      await this.loadConversations();
    }
  }

  aboutToDisappear() {
    // 取消订阅
    this.stateManager.unsubscribe('message_page');
  }

  /**
   * 加载会话列表
   */
  private async loadConversations() {
    if (!this.globalState.isLoggedIn) {
      return;
    }

    this.isLoading = true;
    this.errorMessage = '';

    try {
      const response = await ApiClient.getConversations({ limit: 50 });
      
      if (response.code === 0 && response.data) {
        // 转换 GetConversationsResponse 到 Conversation 格式
        this.conversations = response.data.items.map((item): Conversation => {
          const lastMessage: LastMessage | undefined = item.last_message_content ? {
            message_id: '',
            sender_id: '',
            sender_name: '',
            content: item.last_message_content,
            message_type: 'text',
            sent_at: item.last_message_at || new Date().toISOString()
          } : undefined;

          return {
            group_id: item.group_id,
            group_name: item.group_name,
            group_type: item.group_type,
            group_avatar: item.avatar_url,
            last_message: lastMessage,
            unread_count: item.unread_count,
            last_activity_at: item.last_message_at || new Date().toISOString(),
            is_muted: item.is_muted,
            is_archived: false,
            member_count: 0
          };
        });
      } else {
        this.errorMessage = response.msg || '加载会话失败';
        this.conversations = [];
      }
    } catch (error) {
      this.errorMessage = error instanceof Error ? error.message : '加载会话失败';
      this.conversations = [];
    } finally {
      this.isLoading = false;
    }
  }

  /**
   * 打开聊天页面
   */
  private async openChat(conversation: Conversation) {
    try {
      const routerInstance = this.getUIContext().getRouter();
      
      await routerInstance.pushUrl({
        url: 'pages/ChatPage',
        params: {
          groupId: conversation.group_id,
          groupName: this.getDisplayName(conversation),
          groupType: conversation.group_type
        }
      });
    } catch (error) {
      SystemToast.showError('打开聊天页面失败');
    }
  }

  /**
   * 获取会话显示名称
   * 根据群组类型和名称生成合适的显示名称
   */
  private getDisplayName(conversation: Conversation): string {
    if (conversation.group_name) {
      return conversation.group_name;
    }
    
    // 如果没有群组名称，根据类型返回默认名称
    return conversation.group_type === 'private_chat' ? '私聊' : '群聊';
  }

  /**
   * 搜索附近用户开始聊天
   */
  private async searchNearbyUsers() {
    if (!this.globalState.currentLocation) {
      SystemToast.show('请先获取位置信息');
      return;
    }

    try {
      const response = await ApiClient.searchUsersByLocation({
        latitude: this.globalState.currentLocation.latitude,
        longitude: this.globalState.currentLocation.longitude,
        radius: 1000, // 1km 范围
        limit: 20     // 最多20个用户
      });
      
      if (response.code === 0 && response.data && response.data.items.length > 0) {
        // 打开用户列表选择对话
        SystemToast.show(`发现 ${response.data.items.length} 个附近用户`);
        // 显示用户选择对话框
        this.showNearbyUsersDialog(response.data.items);
      } else {
        SystemToast.show('附近暂无其他用户');
      }
    } catch (error) {
      SystemToast.showError('搜索用户失败');
    }
  }

  /**
   * 显示附近用户选择对话框
   */
  private showNearbyUsersDialog(users: SearchUserByLocationResponse[]): void {
    // 这里可以实现用户选择对话框
    // 暂时使用简单的提示，实际项目中可以实现自定义对话框
    const userNames = users.map(user => user.nickname).join(', ');
    SystemToast.show(`附近用户: ${userNames}`);
    
    // 可以在这里添加导航到聊天页面的逻辑
    // 例如：选择第一个用户开始聊天
    if (users.length > 0) {
      const firstUser = users[0];
      hilog.info(DOMAIN_NUMBER, TAG, `可以与用户 ${firstUser.nickname} 开始聊天`);
    }
  }

  build() {
    Column() {
      // 标题栏
      Row() {
        Text('消息')
          .fontSize(20)
          .fontWeight(FontWeight.Bold)
          .fontColor($r('sys.color.ohos_id_color_text_primary'))
          .layoutWeight(1)

        Button('新聊天')
          .type(ButtonType.Capsule)
          .backgroundColor($r('sys.color.ohos_id_color_emphasize'))
          .fontColor(Color.White)
          .fontSize(14)
          .padding({ left: 16, right: 16, top: 8, bottom: 8 })
          .onClick(() => this.searchNearbyUsers())
      }
      .width('100%')
      .padding(16)
      .backgroundColor($r('sys.color.ohos_id_color_background'))

      // 内容区域
      if (!this.globalState.isLoggedIn) {
        // 未登录状态
        Column({ space: 16 }) {
          Text('🔐')
            .fontSize(48)
          
          Text('请先登录\n才能查看消息')
            .fontSize(16)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
            .textAlign(TextAlign.Center)
          
            Button('去登录')
              .type(ButtonType.Capsule)
              .backgroundColor($r('sys.color.ohos_id_color_emphasize'))
              .fontColor(Color.White)
              .onClick(async () => {
                try {
                  const routerInstance = this.getUIContext().getRouter();
                  await routerInstance.pushUrl({ url: `pages/${PageNames.LOGIN}` });
                } catch (error) {
                  SystemToast.showError('跳转失败');
                }
              })
        }
        .width('100%')
        .height('100%')
        .justifyContent(FlexAlign.Center)
      } else if (this.isLoading) {
        SystemLoadingView({ message: '正在加载会话...' })
      } else if (this.errorMessage) {
        // 错误状态
        Column({ space: 16 }) {
          Text('⚠️')
            .fontSize(48)
          
          Text(this.errorMessage)
            .fontSize(16)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
            .textAlign(TextAlign.Center)
          
          Button('重试')
            .type(ButtonType.Capsule)
            .backgroundColor($r('sys.color.ohos_id_color_emphasize'))
            .fontColor(Color.White)
            .onClick(() => this.loadConversations())
        }
        .width('100%')
        .height('100%')
        .justifyContent(FlexAlign.Center)
        .padding(24)
      } else if (this.conversations.length === 0) {
        // 空状态
        Column({ space: 16 }) {
          Text('💬')
            .fontSize(48)
          
          Text('暂无消息\n点击"新聊天"开始对话吧！')
            .fontSize(16)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
            .textAlign(TextAlign.Center)
        }
        .width('100%')
        .height('100%')
        .justifyContent(FlexAlign.Center)
      } else {
        this.buildConversationList()
      }
    }
    .width('100%')
    .height('100%')
    .backgroundColor($r('sys.color.ohos_id_color_sub_background'))
  }

  @Builder
  private buildConversationList() {
    List({ space: 1 }) {
      ForEach(this.conversations, (conversation: Conversation) => {
        ListItem() {
          this.buildConversationItem(conversation)
        }
      }, (conversation: Conversation) => conversation.group_id)
    }
    .width('100%')
    .layoutWeight(1)
    .backgroundColor($r('sys.color.ohos_id_color_background'))
  }

  @Builder
  private buildConversationItem(conversation: Conversation) {
    Row({ space: 12 }) {
      // 头像
      SystemAvatar({
        avatarUrl: conversation.group_avatar || '',
        nickname: this.getDisplayName(conversation),
        avatarSize: 48
      })

      // 会话信息
      Column({ space: 4 }) {
        Row() {
          Row() {
            // 根据群组类型显示不同图标
            Text(conversation.group_type === 'private_chat' ? '👤' : '👥')
              .fontSize(14)
              .margin({ right: 4 })
            
            Text(this.getDisplayName(conversation))
              .fontSize(16)
              .fontWeight(FontWeight.Medium)
              .fontColor($r('sys.color.ohos_id_color_text_primary'))
          }
          .layoutWeight(1)
          
          Text(this.formatTime(conversation.last_activity_at))
            .fontSize(12)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
        }
        .width('100%')

        Row() {
          Text(conversation.last_message?.content || '暂无消息')
            .fontSize(14)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
            .maxLines(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
            .layoutWeight(1)
          
          if (conversation.unread_count > 0) {
            Text(conversation.unread_count.toString())
              .fontSize(12)
              .fontColor(Color.White)
              .backgroundColor(Color.Red)
              .borderRadius(10)
              .padding({ left: 6, right: 6, top: 2, bottom: 2 })
              .textAlign(TextAlign.Center)
              .width(20)
              .height(20)
          }
        }
        .width('100%')
      }
      .alignItems(HorizontalAlign.Start)
      .layoutWeight(1)
    }
    .width('100%')
    .padding(16)
    .backgroundColor($r('sys.color.ohos_id_color_background'))
    .onClick(() => this.openChat(conversation))
  }

  /**
   * 格式化时间
   */
  private formatTime(timeStr: string): string {
    const time = new Date(timeStr);
    const now = new Date();
    const diff = now.getTime() - time.getTime();
    
    const minutes = Math.floor(diff / (1000 * 60));
    const hours = Math.floor(diff / (1000 * 60 * 60));
    const days = Math.floor(diff / (1000 * 60 * 60 * 24));

    if (minutes < 1) return '刚刚';
    if (minutes < 60) return `${minutes}分钟前`;
    if (hours < 24) return `${hours}小时前`;
    if (days < 7) return `${days}天前`;
    
    return time.toLocaleDateString();
  }
}