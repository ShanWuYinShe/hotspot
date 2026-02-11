/**
 * 智能通知中心组件
 * 展示和管理所有类型的智能通知
 */
import { SmartNotification, SmartNotificationService, NotificationFilter, NotificationAction, NotificationEvent } from '../core/services/SmartNotificationService';
import { SystemIcon, AppIcon } from './SystemIcon';
import { SystemToast } from './SystemToast';
import { AccessibilityManager } from '../core/accessibility/AccessibilityManager';
import { DesignConstants } from '../constants/DesignConstants';

interface FilterOption {
  key: string;
  label: string;
}

@Component
export struct NotificationCenter {
  @State private notifications: SmartNotification[] = [];
  @State private selectedFilter: 'all' | 'unread' | 'performance' | 'social' = 'all';
  @State private isLoading: boolean = false;
  @State private showActions: boolean = false;

  private notificationService = SmartNotificationService.getInstance();
  private accessibilityManager = AccessibilityManager.getInstance();
  /** 通知创建事件监听器引用，用于 aboutToDisappear 清理 */
  private notificationCreatedListener: ((event: NotificationEvent, notification: SmartNotification, extra?: Object) => void) | null = null;

  aboutToAppear() {
    this.loadNotifications();
    this.setupNotificationListener();
  }

  aboutToDisappear() {
    // 移除通知事件监听器，避免组件销毁后监听器堆积
    if (this.notificationCreatedListener) {
      this.notificationService.removeEventListener(NotificationEvent.CREATED, this.notificationCreatedListener);
      this.notificationCreatedListener = null;
    }
  }

  /**
   * 加载通知
   */
  private loadNotifications(): void {
    try {
      this.isLoading = true;
      
      let filter: NotificationFilter | undefined = undefined;
      if (this.selectedFilter !== 'all') {
        filter = {
          type: this.selectedFilter === 'unread' ? undefined : this.selectedFilter as ('system' | 'performance' | 'social' | 'security'),
          read: this.selectedFilter === 'unread' ? false : undefined
        };
      }
      
      this.notifications = this.notificationService.getNotifications(filter);
      
    } catch (error) {
      SystemToast.showError(`加载通知失败: ${error}`, this.getUIContext());
    } finally {
      this.isLoading = false;
    }
  }

  /**
   * 设置通知监听器
   */
  private setupNotificationListener(): void {
    this.notificationCreatedListener = (_event: NotificationEvent, notification: SmartNotification): void => {
      this.loadNotifications();

      // 无障碍播报新通知
      this.accessibilityManager.announceText(
        `收到新通知：${notification.title}`,
        notification.priority === 'critical' ? 'high' : 'medium'
      );
    };
    this.notificationService.addEventListener(NotificationEvent.CREATED, this.notificationCreatedListener);
  }

  /**
   * 处理通知点击
   */
  private async onNotificationClick(notification: SmartNotification): Promise<void> {
    try {
      // 标记为已读
      if (!notification.read) {
        await this.notificationService.markAsRead(notification.id);
        this.loadNotifications();
      }

      // 无障碍播报
      await this.accessibilityManager.announceText(`已打开通知：${notification.title}`);

      // 执行通知相关的操作
      if (notification.actionable && notification.actions && notification.actions.length > 0) {
        // 如果有操作，显示操作菜单
        this.showNotificationActions(notification);
      } else {
        // 否则显示通知详情
        this.showNotificationDetail(notification);
      }
      
    } catch (error) {
      SystemToast.showError(`处理通知失败: ${error}`, this.getUIContext());
    }
  }

  /**
   * 执行通知操作
   */
  private async executeNotificationAction(notification: SmartNotification, actionId: string): Promise<void> {
    try {
      const action = notification.actions?.find(a => a.id === actionId);
      if (action) {
        await action.handler();
        SystemToast.showSuccess(`已执行操作：${action.label}`, this.getUIContext());
        this.loadNotifications();
      }
    } catch (error) {
      SystemToast.showError(`执行操作失败: ${error}`, this.getUIContext());
    }
  }

  /**
   * 删除通知
   */
  private async deleteNotification(notification: SmartNotification): Promise<void> {
    try {
      await this.notificationService.deleteNotification(notification.id);
      this.loadNotifications();
      SystemToast.showSuccess('通知已删除', this.getUIContext());
      
      // 无障碍播报
      await this.accessibilityManager.announceText('通知已删除');
      
    } catch (error) {
      SystemToast.showError(`删除通知失败: ${error}`, this.getUIContext());
    }
  }

  /**
   * 全部标记为已读
   */
  private async markAllAsRead(): Promise<void> {
    try {
      let filter: NotificationFilter | undefined = undefined;
      if (this.selectedFilter !== 'all' && this.selectedFilter !== 'unread') {
        filter = {
          type: this.selectedFilter as ('system' | 'performance' | 'social' | 'security')
        };
      } else if (this.selectedFilter === 'unread') {
        filter = {
          read: false
        };
      }
      
      await this.notificationService.markAllAsRead(filter);
      this.loadNotifications();
      SystemToast.showSuccess('已全部标记为已读', this.getUIContext());
      
      // 无障碍播报
      await this.accessibilityManager.announceText('所有通知已标记为已读');
      
    } catch (error) {
      SystemToast.showError(`标记失败: ${error}`, this.getUIContext());
    }
  }

  /**
   * 获取通知类型图标
   */
  private getNotificationIcon(notification: SmartNotification): AppIcon {
    switch (notification.type) {
      case 'system':
        return AppIcon.SETTINGS;
      case 'performance':
        return AppIcon.CHART;
      case 'social':
        return AppIcon.PERSON;
      case 'security':
        return AppIcon.SHIELD;
      default:
        return AppIcon.INFO;
    }
  }

  /**
   * 获取优先级颜色
   */
  private getPriorityColor(priority: 'low' | 'medium' | 'high' | 'critical'): ResourceColor {
    switch (priority) {
      case 'low':
        return Color.Gray;
      case 'medium':
        return Color.Blue;
      case 'high':
        return Color.Orange;
      case 'critical':
        return Color.Red;
      default:
        return Color.Gray;
    }
  }

  /**
   * 获取类型文本
   */
  private getTypeText(type: 'system' | 'performance' | 'social' | 'security'): string {
    switch (type) {
      case 'system':
        return '系统';
      case 'performance':
        return '性能';
      case 'social':
        return '社交';
      case 'security':
        return '安全';
      default:
        return '未知';
    }
  }

  /**
   * 显示通知操作菜单
   */
  private showNotificationActions(notification: SmartNotification): void {
    SystemToast.showInfo(`显示${notification.title}的操作菜单`, this.getUIContext());
  }

  /**
   * 显示通知详情
   */
  private showNotificationDetail(notification: SmartNotification): void {
    SystemToast.showInfo(`显示${notification.title}的详情`, this.getUIContext());
  }

  build() {
    Column({ space: 16 }) {
      // 标题栏
      Row() {
        Text('通知中心')
          .fontSize(20)
          .fontWeight(FontWeight.Bold)
          .fontColor($r('sys.color.ohos_id_color_text_primary'))
          .layoutWeight(1)

        // 操作按钮
        Row({ space: 12 }) {
          SystemIcon({
            icon: AppIcon.CHECK_ALL,
            iconSize: 20,
            onIconClick: () => this.markAllAsRead()
          })

          SystemIcon({
            icon: AppIcon.REFRESH,
            iconSize: 20,
            onIconClick: () => this.loadNotifications()
          })
        }
      }
      .width('100%')
      .padding({ left: 16, right: 16 })

      // 过滤器
      Scroll(new Scroller()) {
        Row({ space: 8 }) {
          ForEach([
            { key: 'all', label: '全部' },
            { key: 'unread', label: '未读' },
            { key: 'performance', label: '性能' },
            { key: 'social', label: '社交' },
          ], (filter: FilterOption) => {
            Button(filter.label)
              .type(this.selectedFilter === filter.key ? ButtonType.Capsule : ButtonType.Normal)
              .backgroundColor(this.selectedFilter === filter.key 
                ? $r('sys.color.ohos_id_color_emphasize') 
                : $r('sys.color.ohos_id_color_background'))
              .fontColor(this.selectedFilter === filter.key 
                ? Color.White 
                : $r('sys.color.ohos_id_color_text_primary'))
              .fontSize(14)
              .padding({ left: 16, right: 16, top: 8, bottom: 8 })
              .onClick(() => {
                this.selectedFilter = filter.key as ('all' | 'unread' | 'performance' | 'social');
                this.loadNotifications();
              })
          })
        }
        .padding({ left: 16, right: 16 })
      }
      .scrollable(ScrollDirection.Horizontal)
      .scrollBar(BarState.Off)

      // 通知列表
      if (this.isLoading && this.notifications.length === 0) {
        Column() {
          LoadingProgress()
            .width(32)
            .height(32)
            .color($r('sys.color.ohos_id_color_emphasize'))
          
          Text('加载中...')
            .fontSize(14)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
            .margin({ top: 8 })
        }
        .width('100%')
        .height(200)
        .justifyContent(FlexAlign.Center)
      } else if (this.notifications.length === 0) {
        Refresh({ refreshing: $$this.isLoading, builder: this.RefreshBuilder() }) {
          Column() {
            SystemIcon({
              icon: AppIcon.EMPTY,
              iconSize: 48,
              iconColor: $r('sys.color.ohos_id_color_text_tertiary')
            })
            
            Text('暂无通知')
              .fontSize(16)
              .fontColor($r('sys.color.ohos_id_color_text_secondary'))
              .margin({ top: 16 })
          }
          .width('100%')
          .height('100%')
          .justifyContent(FlexAlign.Center)
        }
        .onRefreshing(() => {
          this.loadNotifications();
        })
      } else {
        Refresh({ refreshing: $$this.isLoading, builder: this.RefreshBuilder() }) {
          List({ space: 8 }) {
            ForEach(this.notifications, (notification: SmartNotification) => {
              ListItem() {
                this.buildNotificationItem(notification)
              }
            }, (item: SmartNotification) => item.id)
          }
          .width('100%')
          .layoutWeight(1)
          .padding({ left: 16, right: 16 })
        }
        .onRefreshing(() => {
          this.loadNotifications();
        })
      }
    }
    .width('100%')
    .height('100%')
    .backgroundColor($r('sys.color.ohos_id_color_background'))
  }

  @Builder
  RefreshBuilder() {
    Row() {
      LoadingProgress().width(24).height(24)
      Text('正在刷新...').fontSize(14).margin({left: 8}).fontColor($r('sys.color.ohos_id_color_text_secondary'))
    }
    .width('100%')
    .justifyContent(FlexAlign.Center)
    .padding(10)
  }

  /**
   * 构建通知项
   */
  @Builder
  private buildNotificationItem(notification: SmartNotification) {
    Row({ space: 12 }) {
      // 通知图标和优先级指示器
      Stack() {
        SystemIcon({
          icon: this.getNotificationIcon(notification),
          iconSize: 24,
          iconColor: $r('sys.color.ohos_id_color_emphasize')
        })

        // 优先级指示器
        if (notification.priority === 'high' || notification.priority === 'critical') {
          Circle({ width: 8, height: 8 })
            .fill(this.getPriorityColor(notification.priority))
            .position({ x: 20, y: 0 })
        }

        // 未读指示器
        if (!notification.read) {
          Circle({ width: 6, height: 6 })
            .fill($r('sys.color.ohos_id_color_emphasize'))
            .position({ x: 22, y: 22 })
        }
      }
      .width(32)
      .height(32)

      // 通知内容
      Column({ space: 4 }) {
        Row() {
          Text(notification.title)
            .fontSize(16)
            .fontWeight(notification.read ? FontWeight.Normal : FontWeight.Medium)
            .fontColor($r('sys.color.ohos_id_color_text_primary'))
            .maxLines(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
            .layoutWeight(1)

          Text(this.getTypeText(notification.type))
            .fontSize(12)
            .fontColor($r('sys.color.ohos_id_color_text_tertiary'))
            .padding({ left: DesignConstants.SPACING.SM, right: DesignConstants.SPACING.SM, top: 2, bottom: 2 })
            .backgroundColor($r('sys.color.ohos_id_color_background'))
            .borderRadius(DesignConstants.SPACING.XS)
        }
        .width('100%')

        Text(notification.message)
          .fontSize(14)
          .fontColor($r('sys.color.ohos_id_color_text_secondary'))
          .maxLines(2)
          .textOverflow({ overflow: TextOverflow.Ellipsis })
          .alignSelf(ItemAlign.Start)

        Row() {
          Text(new Date(notification.timestamp).toLocaleString())
            .fontSize(12)
            .fontColor($r('sys.color.ohos_id_color_text_tertiary'))
            .layoutWeight(1)

          // 操作按钮
          if (notification.actionable && notification.actions) {
            Row({ space: 8 }) {
              ForEach(notification.actions.slice(0, 2), (action: NotificationAction) => {
                Button(action.label)
                  .type(ButtonType.Capsule)
                  .backgroundColor(action.type === 'primary' 
                    ? $r('sys.color.ohos_id_color_emphasize')
                    : action.type === 'danger'
                    ? Color.Red
                    : $r('sys.color.ohos_id_color_background'))
                  .fontColor(action.type === 'secondary' 
                    ? $r('sys.color.ohos_id_color_text_primary')
                    : Color.White)
                  .fontSize(12)
                  .padding({ left: 12, right: 12, top: 4, bottom: 4 })
                  .onClick(() => this.executeNotificationAction(notification, action.id))
              })
            }
          }

          // 删除按钮
          SystemIcon({
            icon: AppIcon.DELETE,
            iconSize: 16,
            iconColor: $r('sys.color.ohos_id_color_text_tertiary'),
            onIconClick: () => this.deleteNotification(notification)
          })
        }
        .width('100%')
      }
      .alignItems(HorizontalAlign.Start)
      .layoutWeight(1)
    }
    .width('100%')
    .padding(16)
    .backgroundColor($r('sys.color.ohos_id_color_background'))
    .borderRadius(8)
    .onClick(() => this.onNotificationClick(notification))
    // 无障碍属性
    .accessibilityGroup(true)
    .accessibilityText(`${notification.read ? '' : '未读'}${this.getTypeText(notification.type)}通知：${notification.title}。${notification.message}`)
    .accessibilityDescription(`发送时间：${new Date(notification.timestamp).toLocaleString()}，点击查看详情`)
  }
}
