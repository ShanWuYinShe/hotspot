/**
 * 通知中心（HTTP + WebSocket 联动）
 */
import { NotificationSyncService } from '../core/services/NotificationSyncService';
import { NotificationItem, NotificationPayload, NotificationType } from '../core/types/api';
import { SystemIcon, AppIcon } from './SystemIcon';
import { SystemToast } from './SystemToast';
import { AccessibilityManager } from '../core/accessibility/AccessibilityManager';
import { DesignConstants } from '../constants/DesignConstants';
import { SmartNotificationService, NotificationEvent, SmartNotification } from '../core/services/SmartNotificationService';
import { LoadingMoreIndicator, RefreshIndicator } from './EmptyState';

type NotificationFilterKey = 'all' | 'unread' | 'social' | 'interaction' | 'group' | 'system';

class NotificationFilterOption {
  key: string = '';
  label: string = '';
  constructor(key: string, label: string) {
    this.key = key;
    this.label = label;
  }
}

@Component
export struct NotificationCenter {
  @State private visibleNotifications: NotificationItem[] = [];
  @State private selectedFilter: NotificationFilterKey = 'all';
  @State private isLoading: boolean = false;
  @State private isRefreshing: boolean = false;
  @State private isLoadingMore: boolean = false;
  @State private hasMore: boolean = false;
  @State private errorMessage: string = '';

  private syncService = NotificationSyncService.getInstance();
  private accessibilityManager = AccessibilityManager.getInstance();
  private smartNotificationService = SmartNotificationService.getInstance();
  private filterOptions: NotificationFilterOption[] = [
    new NotificationFilterOption('all', '全部'),
    new NotificationFilterOption('unread', '未读'),
    new NotificationFilterOption('social', '社交'),
    new NotificationFilterOption('interaction', '互动'),
    new NotificationFilterOption('group', '群组'),
    new NotificationFilterOption('system', '系统')
  ];

  private syncListener: (() => void) | null = null;
  private notificationCreatedListener: ((event: NotificationEvent, notification: SmartNotification, extra?: Object) => void) | null = null;

  aboutToAppear() {
    this.attachSyncListener();
    this.refreshNotifications();
    this.setupNotificationListener();
  }

  aboutToDisappear() {
    if (this.syncListener) {
      this.syncService.unsubscribe(this.syncListener);
      this.syncListener = null;
    }
    if (this.notificationCreatedListener) {
      this.smartNotificationService.removeEventListener(NotificationEvent.CREATED, this.notificationCreatedListener);
      this.notificationCreatedListener = null;
    }
  }

  private attachSyncListener(): void {
    if (this.syncListener) {
      return;
    }
    this.syncListener = () => {
      this.applyFilter();
      this.hasMore = this.syncService.hasMoreData();
      this.errorMessage = this.syncService.getLastError();
    };
    this.syncService.subscribe(this.syncListener);
  }

  private setupNotificationListener(): void {
    this.notificationCreatedListener = () => {
      if (!this.isLoading && !this.isRefreshing) {
        this.refreshNotifications();
      }
    };
    this.smartNotificationService.addEventListener(NotificationEvent.CREATED, this.notificationCreatedListener);
  }

  private async refreshNotifications(): Promise<void> {
    if (this.isLoading) {
      return;
    }

    this.isLoading = true;
    this.errorMessage = '';

    try {
      await this.syncService.refresh();
      await this.syncService.syncUnreadCount();
      this.applyFilter();
      this.hasMore = this.syncService.hasMoreData();
    } catch (error) {
      const message = error instanceof Error ? error.message : '加载通知失败';
      if (this.visibleNotifications.length === 0) {
        this.errorMessage = message;
      }
      SystemToast.showError(message, this.getUIContext());
    } finally {
      this.isLoading = false;
      this.isRefreshing = false;
    }
  }

  private async loadMoreNotifications(): Promise<void> {
    if (this.isLoadingMore || !this.hasMore) {
      return;
    }

    this.isLoadingMore = true;
    try {
      await this.syncService.loadMore();
      this.applyFilter();
      this.hasMore = this.syncService.hasMoreData();
    } catch (error) {
      const message = error instanceof Error ? error.message : '加载更多通知失败';
      SystemToast.showError(message, this.getUIContext());
    } finally {
      this.isLoadingMore = false;
    }
  }

  private applyFilter(): void {
    const items = this.syncService.getNotifications();
    let filtered = items;

    if (this.selectedFilter === 'unread') {
      filtered = items.filter(item => !item.is_read);
    } else if (this.selectedFilter !== 'all') {
      filtered = items.filter(item => this.mapTypeToCategory(item.type) === this.selectedFilter);
    }

    this.visibleNotifications = filtered;
  }

  private onFilterSelected(filter: NotificationFilterKey): void {
    if (this.selectedFilter === filter) {
      return;
    }
    this.selectedFilter = filter;
    this.applyFilter();
  }

  private async onMarkAllRead(): Promise<void> {
    try {
      await this.syncService.markAllAsRead();
      await this.syncService.syncUnreadCount();
      this.applyFilter();
      SystemToast.showSuccess('已全部标记为已读', this.getUIContext());
    } catch (error) {
      const message = error instanceof Error ? error.message : '标记失败';
      SystemToast.showError(message, this.getUIContext());
    }
  }

  private async onMarkRead(notification: NotificationItem): Promise<void> {
    if (notification.is_read) {
      return;
    }
    try {
      await this.syncService.markAsRead([notification.notification_id]);
      await this.syncService.syncUnreadCount();
      this.applyFilter();
      await this.accessibilityManager.announceText('通知已标记为已读');
    } catch (error) {
      const message = error instanceof Error ? error.message : '标记失败';
      SystemToast.showError(message, this.getUIContext());
    }
  }

  private async handleNotificationClick(notification: NotificationItem): Promise<void> {
    await this.onMarkRead(notification);
    const message = this.buildNotificationMessage(notification);
    if (message.trim().length > 0) {
      SystemToast.showInfo(message, this.getUIContext());
    }
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

        Row({ space: 12 }) {
          SystemIcon({
            icon: AppIcon.CHECK_ALL,
            iconSize: 20,
            onIconClick: () => this.onMarkAllRead()
          })

          SystemIcon({
            icon: AppIcon.REFRESH,
            iconSize: 20,
            onIconClick: () => this.refreshNotifications()
          })
        }
      }
      .width('100%')
      .padding({ left: 16, right: 16 })

      // 过滤器
      Scroll(new Scroller()) {
        Row({ space: 8 }) {
          ForEach(this.filterOptions, (filter: NotificationFilterOption) => {
            Button(filter.label)
              .type(this.selectedFilter === filter.key ? ButtonType.Capsule : ButtonType.Normal)
              .backgroundColor(this.selectedFilter === filter.key
                ? $r('sys.color.ohos_id_color_emphasize')
                : $r('app.color.bg_secondary'))
              .fontColor(this.selectedFilter === filter.key ? Color.White : $r('sys.color.ohos_id_color_text_primary'))
              .fontSize(14)
              .padding({ left: 16, right: 16, top: 8, bottom: 8 })
              .onClick(() => this.onFilterSelected(filter.key as NotificationFilterKey))
          }, (filter: NotificationFilterOption) => filter.key)
        }
        .padding({ left: 16, right: 16 })
      }
      .scrollable(ScrollDirection.Horizontal)
      .scrollBar(BarState.Off)

      // 内容区域
      if (this.isLoading) {
        Column({ space: 12 }) {
          LoadingProgress()
            .width(32)
            .height(32)
            .color($r('sys.color.ohos_id_color_emphasize'))
          Text('加载中...')
            .fontSize(14)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
        }
        .width('100%')
        .height(200)
        .justifyContent(FlexAlign.Center)
      } else if (this.visibleNotifications.length === 0 && this.errorMessage) {
        Column({ space: DesignConstants.SPACING.LG }) {
          SystemIcon({
            icon: AppIcon.WARNING,
            iconSize: 48,
            iconColor: $r('sys.color.ohos_id_color_text_secondary')
          })
          Text(this.errorMessage)
            .fontSize(16)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
            .textAlign(TextAlign.Center)
          Button('重试')
            .type(ButtonType.Capsule)
            .backgroundColor($r('sys.color.ohos_id_color_emphasize'))
            .fontColor(Color.White)
            .height(40)
            .onClick(() => this.refreshNotifications())
        }
        .width('100%')
        .layoutWeight(1)
        .justifyContent(FlexAlign.Center)
      } else if (this.visibleNotifications.length === 0) {
        Refresh({ refreshing: $$this.isRefreshing, builder: RefreshIndicator }) {
          Column({ space: 12 }) {
            SystemIcon({
              icon: AppIcon.EMPTY,
              iconSize: 48,
              iconColor: $r('sys.color.ohos_id_color_text_tertiary')
            })
            Text('暂无通知')
              .fontSize(16)
              .fontColor($r('sys.color.ohos_id_color_text_secondary'))
          }
          .width('100%')
          .height('100%')
          .justifyContent(FlexAlign.Center)
        }
        .onRefreshing(() => {
          if (!this.isRefreshing) {
            this.isRefreshing = true;
            this.refreshNotifications();
          }
        })
      } else {
        Refresh({ refreshing: $$this.isRefreshing, builder: RefreshIndicator }) {
          List({ space: 12 }) {
            ForEach(this.visibleNotifications, (notification: NotificationItem) => {
              ListItem() {
                this.buildNotificationItem(notification)
              }
            }, (notification: NotificationItem) => notification.notification_id)

            if (this.isLoadingMore) {
              ListItem() {
                LoadingMoreIndicator({ message: '正在加载更多通知...' })
              }
            }
          }
          .width('100%')
          .padding({ left: 16, right: 16, bottom: 4 })
          .onReachEnd(() => this.loadMoreNotifications())
        }
        .layoutWeight(1)
        .onRefreshing(() => {
          if (!this.isRefreshing) {
            this.isRefreshing = true;
            this.refreshNotifications();
          }
        })
      }
    }
    .width('100%')
    .height('100%')
    .backgroundColor($r('app.color.bg_secondary'))
  }

  @Builder
  private buildNotificationItem(notification: NotificationItem) {
    Row({ space: 12 }) {
      Stack() {
        SystemIcon({
          icon: this.getNotificationIcon(notification.type),
          iconSize: 24,
          iconColor: $r('sys.color.ohos_id_color_emphasize')
        })

        if (!notification.is_read) {
          Circle({ width: 6, height: 6 })
            .fill($r('sys.color.ohos_id_color_emphasize'))
            .position({ x: 22, y: 22 })
        }
      }
      .width(32)
      .height(32)

      Column({ space: 6 }) {
        Row() {
          Text(this.buildNotificationTitle(notification))
            .fontSize(16)
            .fontWeight(notification.is_read ? FontWeight.Normal : FontWeight.Medium)
            .fontColor($r('sys.color.ohos_id_color_text_primary'))
            .layoutWeight(1)
            .maxLines(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })

          Text(this.getFilterLabel(this.mapTypeToCategory(notification.type)))
            .fontSize(12)
            .fontColor($r('sys.color.ohos_id_color_text_tertiary'))
            .padding({ left: 8, right: 8, top: 2, bottom: 2 })
            .backgroundColor($r('app.color.bg_secondary'))
            .borderRadius(DesignConstants.RADIUS.SM)
        }

        Text(this.buildNotificationMessage(notification))
          .fontSize(14)
          .fontColor($r('sys.color.ohos_id_color_text_secondary'))
          .maxLines(2)
          .textOverflow({ overflow: TextOverflow.Ellipsis })

        Row() {
          Text(this.formatTimestamp(notification.created_at))
            .fontSize(12)
            .fontColor($r('sys.color.ohos_id_color_text_tertiary'))
            .layoutWeight(1)

          if (!notification.is_read) {
            Button('标记已读')
              .type(ButtonType.Capsule)
              .backgroundColor($r('sys.color.ohos_id_color_emphasize'))
              .fontColor(Color.White)
              .fontSize(12)
              .height(32)
              .padding({ left: 12, right: 12 })
              .onClick(() => this.onMarkRead(notification))
          }
        }
      }
      .layoutWeight(1)
    }
    .width('100%')
    .padding(16)
    .backgroundColor($r('app.color.bg_primary'))
    .borderRadius(DesignConstants.RADIUS.LG)
    .onClick(() => this.handleNotificationClick(notification))
    .accessibilityGroup(true)
    .accessibilityText(`${notification.is_read ? '' : '未读'}通知：${this.buildNotificationTitle(notification)}。${this.buildNotificationMessage(notification)}`)
  }

  private getFilterLabel(filter: NotificationFilterKey): string {
    switch (filter) {
      case 'unread':
        return '未读';
      case 'social':
        return '社交';
      case 'interaction':
        return '互动';
      case 'group':
        return '群组';
      case 'system':
        return '系统';
      default:
        return '全部';
    }
  }

  private formatTimestamp(iso: string): string {
    try {
      const date = new Date(iso);
      if (Number.isNaN(date.getTime())) {
        return iso;
      }
      const month = this.padNumber(date.getMonth() + 1);
      const day = this.padNumber(date.getDate());
      const hour = this.padNumber(date.getHours());
      const minute = this.padNumber(date.getMinutes());
      return `${month}-${day} ${hour}:${minute}`;
    } catch (_) {
      return iso;
    }
  }

  private padNumber(value: number): string {
    return value < 10 ? `0${value}` : `${value}`;
  }

  private buildNotificationTitle(notification: NotificationItem): string {
    switch (notification.type) {
      case 'new_follower':
        return '新的关注者';
      case 'mutual_follow':
        return '互相关注';
      case 'moment_liked':
        return '瞬间获赞';
      case 'moment_commented':
        return '新的评论';
      case 'comment_liked':
        return '评论获赞';
      case 'comment_reply':
        return '评论被回复';
      case 'mentioned':
        return '被提及';
      case 'group_invite_received':
        return '群组邀请';
      case 'group_invite_accepted':
        return '邀请已接受';
      case 'group_application_received':
        return '新的入群申请';
      case 'group_application_accepted':
        return '申请已通过';
      case 'system_announcement':
        return '系统通知';
      default:
        return '通知';
    }
  }

  private buildNotificationMessage(notification: NotificationItem): string {
    const payload = notification.payload;
    const actor = this.getActorName(payload);
    switch (notification.type) {
      case 'new_follower':
        return `${actor} 关注了你`;
      case 'mutual_follow':
        return `你与 ${actor} 已互相关注`;
      case 'moment_liked':
        return `${actor} 赞了你的动态`;
      case 'moment_commented':
        return `${actor}: ${payload?.comment_preview ?? '发表了评论'}`;
      case 'comment_liked':
        return `${actor} 赞了你的评论`;
      case 'comment_reply':
        return `${actor}: ${payload?.reply_content_preview ?? '回复了你的评论'}`;
      case 'mentioned':
        return `${actor} 在动态中提到了你`;
      case 'group_invite_received':
        return `${payload?.group_name ?? '某群组'} 邀请你加入`;
      case 'group_invite_accepted':
        return `${actor} 接受了群组邀请`;
      case 'group_application_received':
        return `${actor} 申请加入 ${payload?.group_name ?? '群组'}`;
      case 'group_application_accepted':
        return `你加入了 ${payload?.group_name ?? '群组'}`;
      case 'system_announcement':
        return payload?.message ?? '请查看系统公告';
      default:
        return payload?.message ?? '';
    }
  }

  private getNotificationIcon(type: NotificationType): AppIcon {
    switch (this.mapTypeToCategory(type)) {
      case 'social':
        return AppIcon.PERSON;
      case 'interaction':
        return AppIcon.MESSAGE;
      case 'group':
        return AppIcon.GROUP;
      case 'system':
        return AppIcon.INFO;
      default:
        return AppIcon.NOTIFICATION;
    }
  }

  private getActorName(payload?: NotificationPayload): string {
    if (!payload) {
      return '有人';
    }
    if (payload.actor?.nickname) {
      return payload.actor.nickname;
    }
    if (payload.user?.nickname) {
      return payload.user.nickname;
    }
    if (payload.applicant?.nickname) {
      return payload.applicant.nickname;
    }
    return '有人';
  }

  private mapTypeToCategory(type: NotificationType): NotificationFilterKey {
    switch (type) {
      case 'new_follower':
      case 'mutual_follow':
        return 'social';
      case 'moment_liked':
      case 'moment_commented':
      case 'comment_liked':
      case 'comment_reply':
      case 'mentioned':
        return 'interaction';
      case 'group_invite_received':
      case 'group_invite_accepted':
      case 'group_application_received':
      case 'group_application_accepted':
        return 'group';
      case 'system_announcement':
        return 'system';
      default:
        return 'social';
    }
  }
}
