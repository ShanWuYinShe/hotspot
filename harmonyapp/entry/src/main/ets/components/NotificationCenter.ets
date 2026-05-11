/**
 * 通知中心（HTTP + WebSocket 联动）
 */
import { NotificationSyncService } from '../core/services/NotificationSyncService';
import { ApiClient } from '../core/api/ApiClient';
import { NotificationItem, NotificationPayload, NotificationType } from '../core/types/api';
import { SystemIcon, AppIcon } from './SystemIcon';
import { AppRouter, AnnouncementsRouteParam, GroupDetailRouteParam, MomentDetailRouteParam, ProfileRouteParam } from '../core/router/AppRouter';
import { PageNames } from '../constants/AppConstants';
import { SystemToast } from './SystemToast';
import { AccessibilityManager } from '../core/accessibility/AccessibilityManager';
import { DesignConstants } from '../constants/DesignConstants';
import { SmartNotificationService, NotificationEvent, SmartNotification } from '../core/services/SmartNotificationService';
import { LoadingMoreIndicator, RefreshIndicator } from './EmptyState';
import { AppThemeManager } from '../core/theme/AppThemeManager';
import { KEY_RESOLVED_THEME_ID } from '../core/theme/AppThemeTypes';

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
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'vibrant_light';
  onClose?: () => void;
  @State private visibleNotifications: NotificationItem[] = [];
  @State private selectedFilter: NotificationFilterKey = 'all';
  @State private isLoading: boolean = false;
  @State private isRefreshing: boolean = false;
  @State private isLoadingMore: boolean = false;
  @State private hasMore: boolean = false;
  @State private errorMessage: string = '';
  @State private actionLoadingIds: string[] = [];

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

  private getTimeGroupLabel(isoDate: string): string {
    const date = new Date(isoDate);
    if (Number.isNaN(date.getTime())) return '更早';
    const now = new Date();
    const todayStart = new Date(now.getFullYear(), now.getMonth(), now.getDate()).getTime();
    const yesterdayStart = todayStart - 86400000;
    const itemTime = date.getTime();
    if (itemTime >= todayStart) return '今天';
    if (itemTime >= yesterdayStart) return '昨天';
    return '更早';
  }

  private getGroupedNotifications(): { label: string; items: NotificationItem[] }[] {
    const groups: { label: string; items: NotificationItem[] }[] = [];
    const order = ['今天', '昨天', '更早'];
    for (const notification of this.visibleNotifications) {
      const label = this.getTimeGroupLabel(notification.created_at);
      let group = groups.find(g => g.label === label);
      if (!group) {
        group = { label, items: [] };
        groups.push(group);
      }
      group.items.push(notification);
    }
    groups.sort((a, b) => order.indexOf(a.label) - order.indexOf(b.label));
    return groups;
  }

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

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
    if (this.navigateByNotification(notification)) {
      if (this.onClose) {
        this.onClose();
      }
      return;
    }
    const message = this.buildNotificationMessage(notification);
    if (message.trim().length > 0) {
      SystemToast.showInfo(message, this.getUIContext());
    }
  }

  private navigateByNotification(notification: NotificationItem): boolean {
    switch (notification.type) {
      case 'new_follower':
      case 'mutual_follow': {
        const userId = notification.payload?.actor?.user_id ?? notification.payload?.user?.user_id ?? '';
        if (!userId) return false;
        const params = new ProfileRouteParam();
        params.userId = userId;
        AppRouter.push(PageNames.PROFILE, params);
        return true;
      }
      case 'moment_liked':
      case 'moment_commented':
      case 'comment_liked':
      case 'comment_reply':
      case 'mentioned': {
        const momentId = notification.payload?.moment_id ?? '';
        if (!momentId) return false;
        const params = new MomentDetailRouteParam();
        params.momentId = momentId;
        AppRouter.push(PageNames.MOMENT_DETAIL, params);
        return true;
      }
      case 'group_invite_received':
      case 'group_invite_accepted':
      case 'group_application_received':
      case 'group_application_accepted': {
        const groupId = notification.payload?.group_id ?? '';
        if (!groupId) return false;
        const params = new GroupDetailRouteParam();
        params.groupId = groupId;
        AppRouter.push(PageNames.GROUP_DETAIL, params);
        return true;
      }
      case 'system_announcement': {
        const params = new AnnouncementsRouteParam();
        params.focusAnnouncementId = notification.payload?.announcement_id ?? '';
        AppRouter.push(PageNames.ANNOUNCEMENTS, params);
        return true;
      }
      default:
        return false;
    }
  }

  private isActionLoading(notificationId: string): boolean {
    return this.actionLoadingIds.includes(notificationId);
  }

  private setActionLoading(notificationId: string, loading: boolean): void {
    if (loading) {
      if (!this.actionLoadingIds.includes(notificationId)) {
        this.actionLoadingIds = [...this.actionLoadingIds, notificationId];
      }
      return;
    }
    this.actionLoadingIds = this.actionLoadingIds.filter((id: string) => id !== notificationId);
  }

  private canHandleGroupInvite(notification: NotificationItem): boolean {
    return notification.type === 'group_invite_received' && !!notification.payload?.invite_id;
  }

  private canHandleGroupApplication(notification: NotificationItem): boolean {
    return notification.type === 'group_application_received' && !!notification.payload?.application_id;
  }

  private async handleGroupInviteAction(notification: NotificationItem, action: 'accept' | 'reject'): Promise<void> {
    if (!this.canHandleGroupInvite(notification) || this.isActionLoading(notification.notification_id)) {
      return;
    }
    const inviteId = notification.payload?.invite_id ?? '';
    if (!inviteId) {
      SystemToast.showWarning('邀请信息缺失，无法处理', this.getUIContext());
      return;
    }
    this.setActionLoading(notification.notification_id, true);
    try {
      const response = action === 'accept'
        ? await ApiClient.acceptGroupInvite({ invite_id: inviteId })
        : await ApiClient.rejectGroupInvite({ invite_id: inviteId });
      if (response.code !== 0) {
        SystemToast.showError(response.message || (action === 'accept' ? '接受邀请失败' : '拒绝邀请失败'), this.getUIContext());
        return;
      }
      await this.onMarkRead(notification);
      SystemToast.showSuccess(action === 'accept' ? '已接受群组邀请' : '已拒绝群组邀请', this.getUIContext());
      await this.refreshNotifications();
    } catch (error) {
      const message = error instanceof Error ? error.message : (action === 'accept' ? '接受邀请失败' : '拒绝邀请失败');
      SystemToast.showError(message, this.getUIContext());
    } finally {
      this.setActionLoading(notification.notification_id, false);
    }
  }

  private async handleGroupApplicationAction(notification: NotificationItem, action: 'approve' | 'reject'): Promise<void> {
    if (!this.canHandleGroupApplication(notification) || this.isActionLoading(notification.notification_id)) {
      return;
    }
    const applicationId = notification.payload?.application_id ?? '';
    if (!applicationId) {
      SystemToast.showWarning('申请信息缺失，无法处理', this.getUIContext());
      return;
    }
    this.setActionLoading(notification.notification_id, true);
    try {
      const response = action === 'approve'
        ? await ApiClient.approveGroupApplication({ application_id: applicationId })
        : await ApiClient.rejectGroupApplication({ application_id: applicationId });
      if (response.code !== 0) {
        SystemToast.showError(response.message || (action === 'approve' ? '批准申请失败' : '拒绝申请失败'), this.getUIContext());
        return;
      }
      await this.onMarkRead(notification);
      SystemToast.showSuccess(action === 'approve' ? '已批准入群申请' : '已拒绝入群申请', this.getUIContext());
      await this.refreshNotifications();
    } catch (error) {
      const message = error instanceof Error ? error.message : (action === 'approve' ? '批准申请失败' : '拒绝申请失败');
      SystemToast.showError(message, this.getUIContext());
    } finally {
      this.setActionLoading(notification.notification_id, false);
    }
  }

  build() {
    Column({ space: 16 }) {
      // 标题栏
      Row() {
        Row({ space: 12 }) {
          if (this.onClose) {
            Row() {
              Text() {
                SymbolSpan($r(AppIcon.BACK))
                  .fontSize(15)
                  .fontColor([this.getPalette().textSecondary])
              }
            }
            .width(36)
            .height(36)
            .justifyContent(FlexAlign.Center)
            .borderRadius(18)
            .backgroundColor(this.getPalette().surfaceBackground)
            .shadow(this.getPalette().controlShadow)
            .border({ width: 1, color: this.getPalette().borderColor })
            .onClick(() => {
              if (this.onClose) {
                this.onClose();
              }
            })
          }

          Column({ space: 1 }) {
            Text('通知')
              .fontSize(20)
              .fontWeight(FontWeight.Bold)
              .fontColor(this.getPalette().textPrimary)
            if (this.getUnreadCount() > 0) {
              Text(`${this.getUnreadCount()} 条未读`)
                .fontSize(12)
                .fontColor(this.getPalette().textSecondary)
            }
          }
        }
        .layoutWeight(1)

        if (this.getUnreadCount() > 0) {
          Button('全部已读')
            .type(ButtonType.Normal)
            .backgroundColor(this.getPalette().surfaceBackground)
            .fontColor(this.getPalette().accentBackground)
            .fontSize(12)
            .borderRadius(DesignConstants.RADIUS.FULL)
            .padding({ left: 12, right: 12, top: 6, bottom: 6 })
            .shadow(this.getPalette().controlShadow)
            .border({ width: 1, color: this.getPalette().borderColor })
            .onClick(() => this.onMarkAllRead())
        }
      }
      .width('100%')
      .padding({ left: 16, right: 16 })

      // 过滤器标签
      Scroll() {
        Row({ space: 8 }) {
          ForEach(this.filterOptions, (filter: NotificationFilterOption) => {
            Text(filter.label)
              .fontSize(12)
              .fontColor(this.selectedFilter === filter.key
                ? Color.White
                : this.getPalette().textSecondary)
              .backgroundColor(this.selectedFilter === filter.key
                ? this.getPalette().accentBackground
                : this.getPalette().surfaceBackground)
              .borderRadius(DesignConstants.RADIUS.FULL)
              .padding({ left: 14, right: 14, top: 6, bottom: 6 })
              .shadow(this.selectedFilter === filter.key
                ? {
                  radius: 10,
                  color: this.getPalette().accentShadow.color,
                  offsetX: 0,
                  offsetY: 3
                }
                : this.getPalette().controlShadow)
              .border({
                width: 1,
                color: this.selectedFilter === filter.key
                  ? Color.Transparent
                  : this.getPalette().borderColor
              })
              .onClick(() => this.onFilterSelected(filter.key as NotificationFilterKey))
          }, (filter: NotificationFilterOption) => filter.key + '_' + (this.selectedFilter === filter.key))
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
            .color(this.getPalette().accentBackground)
          Text('加载中...')
            .fontSize(14)
            .fontColor(this.getPalette().textSecondary)
        }
        .width('100%')
        .height(200)
        .justifyContent(FlexAlign.Center)
      } else if (this.visibleNotifications.length === 0 && this.errorMessage) {
        Column({ space: DesignConstants.SPACING.LG }) {
          SystemIcon({
            icon: AppIcon.WARNING,
            iconSize: 48,
            iconColor: this.getPalette().textSecondary
          })
          Text(this.errorMessage)
            .fontSize(16)
            .fontColor(this.getPalette().textSecondary)
            .textAlign(TextAlign.Center)
          Button('重试')
            .type(ButtonType.Normal)
            .backgroundColor(this.getPalette().accentBackground)
            .fontColor(this.getPalette().accentText)
            .height(40)
            .borderRadius(DesignConstants.RADIUS.XL)
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
              iconColor: this.getPalette().textTertiary
            })
            Text('暂无通知')
              .fontSize(16)
              .fontColor(this.getPalette().textSecondary)
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
            ForEach(this.getGroupedNotifications(), (group: { label: string; items: NotificationItem[] }) => {
              ListItemGroup({
                header: this.buildGroupHeader(group.label),
                space: 8
              }) {
                ForEach(group.items, (notification: NotificationItem) => {
                  ListItem() {
                    this.buildNotificationItem(notification)
                  }
                }, (notification: NotificationItem) =>
                  notification.notification_id + '_p' + (this.actionLoadingIds.indexOf(notification.notification_id) >= 0))
              }
            }, (group: { label: string; items: NotificationItem[] }) => group.label)

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
    .backgroundColor(this.getPalette().pageBackground)
  }

  @Builder
  private buildGroupHeader(label: string) {
    Text(label)
      .fontSize(DesignConstants.FONT.SIZE.SM)
      .fontWeight(FontWeight.Medium)
      .fontColor(this.getPalette().textSecondary)
      .padding({ left: 4, top: DesignConstants.SPACING.SM, bottom: DesignConstants.SPACING.XS })
  }

  @Builder
  private buildNotificationItem(notification: NotificationItem) {
    Row({ space: 14 }) {
      // 头像 + 类型图标徽章
      Stack() {
        if (notification.payload?.actor?.avatar_url) {
          Image(notification.payload.actor.avatar_url)
            .width(44)
            .height(44)
            .borderRadius(22)
            .objectFit(ImageFit.Cover)
        } else {
          Row() {
            Text() {
              SymbolSpan($r(this.getNotificationIcon(notification.type)))
                .fontSize(18)
                .fontColor([this.getTypeColor(notification.type)])
            }
          }
          .width(44)
          .height(44)
          .borderRadius(22)
          .justifyContent(FlexAlign.Center)
          .backgroundColor(this.getTypeColor(notification.type) + '18')
        }

        // 类型图标徽章
        Row() {
          Text() {
            SymbolSpan($r(this.getNotificationIcon(notification.type)))
              .fontSize(9)
              .fontColor([Color.White])
          }
        }
        .width(18)
        .height(18)
        .borderRadius(9)
        .justifyContent(FlexAlign.Center)
        .backgroundColor(this.getTypeColor(notification.type))
        .border({ width: 2, color: this.getPalette().pageBackground })
        .position({ x: 30, y: 30 })
      }

      // 文本内容
      Column({ space: 2 }) {
        Text() {
          Span(this.buildNotificationTitle(notification))
            .fontSize(13)
            .fontColor(this.getPalette().textPrimary)
            .fontWeight(notification.is_read ? FontWeight.Normal : FontWeight.Medium)
          Span(' ' + this.buildNotificationMessage(notification))
            .fontSize(13)
            .fontColor(this.getPalette().textPrimary)
        }
        .maxLines(2)
        .textOverflow({ overflow: TextOverflow.Ellipsis })
        .lineHeight(18)

        Text(this.formatTimestamp(notification.created_at))
          .fontSize(11)
          .fontColor(this.getPalette().textSecondary)
          .margin({ top: 2 })

        // 群组操作按钮
        if (this.canHandleGroupInvite(notification)) {
          Row({ space: 8 }) {
            Button(this.isActionLoading(notification.notification_id) ? '处理中...' : '接受')
              .type(ButtonType.Normal)
              .backgroundColor(this.getPalette().accentBackground)
              .fontColor(this.getPalette().accentText)
              .fontSize(11)
              .height(28)
              .borderRadius(DesignConstants.RADIUS.FULL)
              .enabled(!this.isActionLoading(notification.notification_id))
              .onClick(() => this.handleGroupInviteAction(notification, 'accept'))

            Button('拒绝')
              .type(ButtonType.Normal)
              .backgroundColor(this.getPalette().surfaceSubtle)
              .fontColor(this.getPalette().textPrimary)
              .fontSize(11)
              .height(28)
              .borderRadius(DesignConstants.RADIUS.FULL)
              .borderWidth(1)
              .borderColor(this.getPalette().borderColor)
              .enabled(!this.isActionLoading(notification.notification_id))
              .onClick(() => this.handleGroupInviteAction(notification, 'reject'))
          }
          .margin({ top: 6 })
        } else if (this.canHandleGroupApplication(notification)) {
          Row({ space: 8 }) {
            Button(this.isActionLoading(notification.notification_id) ? '处理中...' : '批准')
              .type(ButtonType.Normal)
              .backgroundColor(this.getPalette().accentBackground)
              .fontColor(this.getPalette().accentText)
              .fontSize(11)
              .height(28)
              .borderRadius(DesignConstants.RADIUS.FULL)
              .enabled(!this.isActionLoading(notification.notification_id))
              .onClick(() => this.handleGroupApplicationAction(notification, 'approve'))

            Button('拒绝')
              .type(ButtonType.Normal)
              .backgroundColor(this.getPalette().surfaceSubtle)
              .fontColor(this.getPalette().textPrimary)
              .fontSize(11)
              .height(28)
              .borderRadius(DesignConstants.RADIUS.FULL)
              .borderWidth(1)
              .borderColor(this.getPalette().borderColor)
              .enabled(!this.isActionLoading(notification.notification_id))
              .onClick(() => this.handleGroupApplicationAction(notification, 'reject'))
          }
          .margin({ top: 6 })
        }
      }
      .layoutWeight(1)

      // 未读圆点
      if (!notification.is_read) {
        Row()
          .width(8)
          .height(8)
          .borderRadius(4)
          .backgroundColor(this.getTypeColor(notification.type))
      }
    }
    .width('100%')
    .padding({ left: 12, right: 12, top: 12, bottom: 12 })
    .backgroundColor(notification.is_read ? Color.Transparent : this.getTypeColor(notification.type) + '07')
    .borderRadius(18)
    .border({
      width: 1,
      color: notification.is_read ? Color.Transparent : this.getTypeColor(notification.type) + '18'
    })
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

  private getUnreadCount(): number {
    let count = 0;
    for (const notif of this.visibleNotifications) {
      if (!notif.is_read) {
        count++;
      }
    }
    return count;
  }

  private getTypeColor(type: string): string {
    switch (this.mapTypeToCategory(type as NotificationType)) {
      case 'social':
        return '#6366f1';
      case 'interaction':
        return '#EF4444';
      case 'group':
        return '#22c55e';
      case 'system':
        return '#F59E0B';
      default:
        return '#3B82F6';
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
        return payload?.comment_preview ? `${actor} 赞了你的评论：${payload.comment_preview}` : `${actor} 赞了你的评论`;
      case 'comment_reply':
        return payload?.replied_comment_preview
          ? `${actor} 回复了你的评论：${payload.replied_comment_preview}`
          : `${actor}: ${payload?.reply_content_preview ?? '回复了你的评论'}`;
      case 'mentioned':
        return payload?.target_type === 'comment'
          ? `${actor} 在评论中提到了你`
          : `${actor} 在动态中提到了你`;
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
