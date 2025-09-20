/**
 * 通知组件
 * 显示应用内通知和消息提醒
 */

import { GlobalStateManager, GlobalState } from '../common/state/GlobalStateManager';
import { WebSocketService } from '../common/websocket/WebSocketService';
import { SystemToast } from './SystemToast';
import { PageNames } from '../constants/AppConstants';
import { hilog } from '@kit.PerformanceAnalysisKit';

const DOMAIN_NUMBER: number = 0xFF15;
const TAG: string = 'NotificationComponent';

type NotificationData = Record<string, string | number | boolean>;

interface NotificationItem {
  id: string;
  type: 'message' | 'group' | 'moment' | 'system';
  title: string;
  content: string;
  timestamp: number;
  read: boolean;
  data?: NotificationData;
}

@Component
export struct NotificationComponent {
  @State notifications: NotificationItem[] = [];
  @State showNotifications: boolean = false;
  @State unreadCount: number = 0;
  private globalState = GlobalStateManager.getInstance().getState();
  private webSocketService = WebSocketService.getInstance();

  aboutToAppear() {
    hilog.info(DOMAIN_NUMBER, TAG, '通知组件初始化');
    
    // 监听WebSocket消息
    this.setupWebSocketListeners();
    
    // 加载本地通知
    this.loadLocalNotifications();
  }

  aboutToDisappear() {
    // 清理监听器
    this.cleanupListeners();
  }

  /**
   * 设置WebSocket监听器
   */
  private setupWebSocketListeners() {
    // 这里应该监听WebSocket服务的消息事件
    // 由于WebSocketService的具体实现可能不同，这里提供示例
    
    // 监听新消息
    // this.webSocketService.on('new_message', (message) => {
    //   this.addNotification({
    //     id: `msg_${Date.now()}`,
    //     type: 'message',
    //     title: `来自 ${message.sender_name} 的消息`,
    //     content: message.content,
    //     timestamp: Date.now(),
    //     read: false,
    //     data: message
    //   });
    // });

    // 监听群组通知
    // this.webSocketService.on('group_notification', (notification) => {
    //   this.addNotification({
    //     id: `group_${Date.now()}`,
    //     type: 'group',
    //     title: notification.title,
    //     content: notification.content,
    //     timestamp: Date.now(),
    //     read: false,
    //     data: notification
    //   });
    // });
  }

  /**
   * 清理监听器
   */
  private cleanupListeners() {
    // 移除WebSocket监听器
  }

  /**
   * 加载本地通知
   */
  private loadLocalNotifications() {
    // 从本地存储加载通知历史
    // 这里可以集成 PreferencesUtil
    
    // 示例通知
    this.notifications = [
      {
        id: 'welcome',
        type: 'system',
        title: '欢迎使用 GeoTalk',
        content: '开始探索附近的精彩世界吧！',
        timestamp: Date.now() - 3600000, // 1小时前
        read: false
      },
      {
        id: 'location_tip',
        type: 'system',
        title: '位置服务提示',
        content: '开启位置服务以获得更好的体验',
        timestamp: Date.now() - 1800000, // 30分钟前
        read: false
      }
    ];
    
    this.updateUnreadCount();
  }

  /**
   * 添加新通知
   */
  private addNotification(notification: NotificationItem) {
    this.notifications.unshift(notification);
    
    // 限制通知数量
    if (this.notifications.length > 50) {
      this.notifications = this.notifications.slice(0, 50);
    }
    
    this.updateUnreadCount();
    
    // 显示Toast提示
    SystemToast.show(notification.title);
    
    // 保存到本地存储
    this.saveNotifications();
  }

  /**
   * 更新未读数量
   */
  private updateUnreadCount() {
    this.unreadCount = this.notifications.filter(n => !n.read).length;
  }

  /**
   * 标记通知为已读
   */
  private markAsRead(notificationId: string) {
    const notification = this.notifications.find(n => n.id === notificationId);
    if (notification && !notification.read) {
      notification.read = true;
      this.updateUnreadCount();
      this.saveNotifications();
    }
  }

  /**
   * 标记所有通知为已读
   */
  private markAllAsRead() {
    this.notifications.forEach(n => n.read = true);
    this.updateUnreadCount();
    this.saveNotifications();
  }

  /**
   * 删除通知
   */
  private deleteNotification(notificationId: string) {
    const index = this.notifications.findIndex(n => n.id === notificationId);
    if (index > -1) {
      this.notifications.splice(index, 1);
      this.updateUnreadCount();
      this.saveNotifications();
    }
  }

  /**
   * 清空所有通知
   */
  private clearAllNotifications() {
    this.notifications = [];
    this.updateUnreadCount();
    this.saveNotifications();
  }

  /**
   * 保存通知到本地存储
   */
  private saveNotifications() {
    // 这里可以使用 PreferencesUtil 保存通知
    hilog.debug(DOMAIN_NUMBER, TAG, `保存 ${this.notifications.length} 条通知`);
  }

  /**
   * 处理通知点击
   */
  private handleNotificationClick(notification: NotificationItem) {
    // 标记为已读
    this.markAsRead(notification.id);
    
    // 根据通知类型进行导航
    switch (notification.type) {
      case 'message':
        // 导航到聊天页面
        if (notification.data) {
          this.getUIContext().getRouter().pushUrl({
            url: `pages/${PageNames.CHAT}`,
            params: notification.data
          });
        }
        break;
      case 'group':
        // 导航到群组页面
        if (notification.data) {
          this.getUIContext().getRouter().pushUrl({
            url: `pages/${PageNames.GROUP_DETAIL}`,
            params: { groupId: notification.data.group_id }
          });
        }
        break;
      case 'moment':
        // 导航到瞬间页面
        this.getUIContext().getRouter().pushUrl({
          url: `pages/${PageNames.MOMENT}`
        });
        break;
      case 'system':
        // 系统通知，可能不需要导航
        break;
    }
    
    // 关闭通知面板
    this.showNotifications = false;
  }

  /**
   * 格式化时间
   */
  private formatTime(timestamp: number): string {
    const now = Date.now();
    const diff = now - timestamp;
    
    if (diff < 60000) { // 1分钟内
      return '刚刚';
    } else if (diff < 3600000) { // 1小时内
      return `${Math.floor(diff / 60000)}分钟前`;
    } else if (diff < 86400000) { // 24小时内
      return `${Math.floor(diff / 3600000)}小时前`;
    } else {
      const date = new Date(timestamp);
      return `${date.getMonth() + 1}/${date.getDate()}`;
    }
  }

  /**
   * 获取通知图标
   */
  private getNotificationIcon(type: 'message' | 'group' | 'moment' | 'system'): string {
    switch (type) {
      case 'message': return '💬';
      case 'group': return '👥';
      case 'moment': return '📸';
      case 'system': return '🔔';
      default: return '📢';
    }
  }

  build() {
    Stack() {
      // 通知按钮
      Button() {
        Stack() {
          Text('🔔')
            .fontSize(20)
          
          // 未读数量徽章
          if (this.unreadCount > 0) {
            Text(this.unreadCount > 99 ? '99+' : this.unreadCount.toString())
              .fontSize(10)
              .fontColor(Color.White)
              .backgroundColor(Color.Red)
              .borderRadius(8)
              .padding({ left: 4, right: 4, top: 1, bottom: 1 })
              .position({ x: '60%', y: '10%' })
          }
        }
      }
      .width(40)
      .height(40)
      .backgroundColor(Color.Transparent)
      .onClick(() => {
        this.showNotifications = !this.showNotifications;
      })

      // 通知面板
      if (this.showNotifications) {
        Column() {
          // 标题栏
          Row() {
            Text('通知')
              .fontSize(18)
              .fontWeight(FontWeight.Bold)
              .layoutWeight(1)

            if (this.unreadCount > 0) {
              Button('全部已读')
                .fontSize(12)
                .backgroundColor(Color.Transparent)
                .fontColor($r('sys.color.ohos_id_color_emphasize'))
                .onClick(() => this.markAllAsRead())
            }

            Button('清空')
              .fontSize(12)
              .backgroundColor(Color.Transparent)
              .fontColor($r('sys.color.ohos_id_color_text_secondary'))
              .onClick(() => this.clearAllNotifications())

            Button('✕')
              .width(30)
              .height(30)
              .fontSize(16)
              .backgroundColor(Color.Transparent)
              .fontColor($r('sys.color.ohos_id_color_text_primary'))
              .onClick(() => {
                this.showNotifications = false;
              })
          }
          .width('100%')
          .padding(16)

          Divider()

          // 通知列表
          if (this.notifications.length > 0) {
            List() {
              ForEach(this.notifications, (notification: NotificationItem) => {
                ListItem() {
                  Row() {
                    // 通知图标
                    Text(this.getNotificationIcon(notification.type))
                      .fontSize(24)
                      .margin({ right: 12 })

                    // 通知内容
                    Column() {
                      Row() {
                        Text(notification.title)
                          .fontSize(14)
                          .fontWeight(notification.read ? FontWeight.Normal : FontWeight.Bold)
                          .fontColor(notification.read ? 
                            $r('sys.color.ohos_id_color_text_secondary') : 
                            $r('sys.color.ohos_id_color_text_primary'))
                          .maxLines(1)
                          .textOverflow({ overflow: TextOverflow.Ellipsis })
                          .layoutWeight(1)

                        Text(this.formatTime(notification.timestamp))
                          .fontSize(10)
                          .fontColor($r('sys.color.ohos_id_color_text_secondary'))
                      }
                      .width('100%')
                      .margin({ bottom: 4 })

                      Text(notification.content)
                        .fontSize(12)
                        .fontColor($r('sys.color.ohos_id_color_text_secondary'))
                        .maxLines(2)
                        .textOverflow({ overflow: TextOverflow.Ellipsis })
                    }
                    .alignItems(HorizontalAlign.Start)
                    .layoutWeight(1)

                    // 未读标识
                    if (!notification.read) {
                      Circle()
                        .width(8)
                        .height(8)
                        .fill($r('sys.color.ohos_id_color_emphasize'))
                        .margin({ left: 8 })
                    }

                    // 删除按钮
                    Button('✕')
                      .width(24)
                      .height(24)
                      .fontSize(12)
                      .backgroundColor(Color.Transparent)
                      .fontColor($r('sys.color.ohos_id_color_text_secondary'))
                      .margin({ left: 8 })
                      .onClick(() => this.deleteNotification(notification.id))
                  }
                  .width('100%')
                  .padding(12)
                  .backgroundColor(notification.read ? 
                    Color.Transparent : 
                    $r('sys.color.ohos_id_color_background_transparent'))
                  .onClick(() => this.handleNotificationClick(notification))
                }
              })
            }
            .width('100%')
            .layoutWeight(1)
          } else {
            // 空状态
            Column() {
              Text('🔔')
                .fontSize(48)
                .margin({ bottom: 16 })
              
              Text('暂无通知')
                .fontSize(16)
                .fontColor($r('sys.color.ohos_id_color_text_secondary'))
            }
            .width('100%')
            .layoutWeight(1)
            .justifyContent(FlexAlign.Center)
            .alignItems(HorizontalAlign.Center)
          }
        }
        .width(320)
        .height(400)
        .backgroundColor($r('sys.color.ohos_id_color_background'))
        .borderRadius(12)
        .shadow({
          radius: 20,
          color: Color.Black,
          offsetX: 0,
          offsetY: 4
        })
        .position({ x: -280, y: 50 }) // 相对于按钮的位置
      }
    }
    .alignContent(Alignment.TopEnd)
  }
}



