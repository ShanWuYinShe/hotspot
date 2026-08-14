/**
 * 通知中心（HTTP + WebSocket 联动）
 */
import { NotificationSyncService } from '../core/services/NotificationSyncService';
import { ApiClient } from '../core/api/ApiClient';
import { NotifyItem, NotifyPayload, NotifyType } from '../core/types/api';
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

type NotificationFilterKey = 'all' | 'unread' | 'social' | 'interaction' | 'group' | 'report' | 'nearby' | 'system';

interface GroupedNotification {
  label: string;
  items: NotifyItem[];
}

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
  @State private visibleNotifications: NotifyItem[] = [];
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
    new NotificationFilterOption('report', '举报'),
    new NotificationFilterOption('nearby', '附近'),
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

  private getGroupedNotifications(): GroupedNotification[] {
    const groups: GroupedNotification[] = [];
    const order = ['今天', '昨天', '更早'];
    for (const notification of this.visibleNotifications) {
      const label = this.getTimeGroupLabel(notification.created_at);
      let group = groups.find(g => g.label === label);
      if (!group) {
        group = { label: label, items: [] as NotifyItem[] } as GroupedNotification;
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

  private async onMarkRead(notification: NotifyItem): Promise<void> {
    if (notification.is_read) {
      return;
    }
    try {
      await this.syncService.markAsRead([notification.notify_id]);
      await this.syncService.syncUnreadCount();
      this.applyFilter();
      await this.accessibilityManager.announceText('通知已标记为已读');
    } catch (error) {
      const message = error instanceof Error ? error.message : '标记失败';
      SystemToast.showError(message, this.getUIContext());
    }
  }

  private async handleNotificationClick(notification: NotifyItem): Promise<void> {
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

  private navigateByNotification(notification: NotifyItem): boolean {
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
      case 'moment_commented':
      case 'comment_reply':
      case 'mentioned': {
        const momentId = notification.payload?.moment_id ?? '';
        if (!momentId) return false;
        const params = new MomentDetailRouteParam();
        params.momentId = momentId;
        AppRouter.push(PageNames.MOMENT_DETAIL, params);
        return true;
      }
      case 'reaction_added': {
        // 表情反应目标可能是 moment/comment/message；仅 moment 可直达详情
        if (notification.payload?.target_type !== 'moment') {
          return false;
        }
        const targetMomentId = notification.payload?.target_id ?? '';
        if (!targetMomentId) return false;
        const params = new MomentDetailRouteParam();
        params.momentId = targetMomentId;
        AppRouter.push(PageNames.MOMENT_DETAIL, params);
        return true;
      }
      case 'nearby_new_moment': {
        const momentId = notification.payload?.moment_id ?? '';
        if (!momentId) return false;
        const params = new MomentDetailRouteParam();
        params.momentId = momentId;
        AppRouter.push(PageNames.MOMENT_DETAIL, params);
        return true;
      }
      case 'group_invite_received':
      case 'group_mentioned':
      case 'group_application_received':
      case 'group_application_accepted':
      case 'group_application_rejected': {
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

  private canHandleGroupInvite(notification: NotifyItem): boolean {
    return notification.type === 'group_invite_received' && !!notification.payload?.invite_id;
  }

  private canHandleGroupApplication(notification: NotifyItem): boolean {
    return notification.type === 'group_application_received' && !!notification.payload?.application_id;
  }

  private async handleGroupInviteAction(notification: NotifyItem, action: 'accept' | 'reject'): Promise<void> {
    if (!this.canHandleGroupInvite(notification) || this.isActionLoading(notification.notify_id)) {
      return;
    }
    const inviteId = notification.payload?.invite_id ?? '';
    if (!inviteId) {
      SystemToast.showWarning('邀请信息缺失，无法处理', this.getUIContext());
      return;
    }
    this.setActionLoading(notification.notify_id, true);
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
      this.setActionLoading(notification.notify_id, false);
    }
  }

  private async handleGroupApplicationAction(notification: NotifyItem, action: 'approve' | 'reject'): Promise<void> {
    if (!this.canHandleGroupApplication(notification) || this.isActionLoading(notification.notify_id)) {
      return;
    }
    const applicationId = notification.payload?.application_id ?? '';
    const groupId = notification.payload?.group_id ?? '';
    if (!applicationId || !groupId) {
      SystemToast.showWarning('申请信息缺失，无法处理', this.getUIContext());
      return;
    }
    this.setActionLoading(notification.notify_id, true);
    try {
      const response = action === 'approve'
        ? await ApiClient.approveGroupApplication({ group_id: groupId, application_id: applicationId })
        : await ApiClient.rejectGroupApplication({ group_id: groupId, application_id: applicationId });
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
      this.setActionLoading(notification.notify_id, false);
    }
  }

  build() {
    Column({ space: DesignConstants.SPACING.LG }) {
      // 标题栏
      Row() {
        Row({ space: DesignConstants.SPACING.MD }) {
          if (this.onClose) {
            Row() {
              Text() {
                SymbolSpan($r(AppIcon.BACK))
                  .fontSize(DesignConstants.FONT.SIZE.LG)
                  .fontColor([this.getPalette().textSecondary])
              }
            }
            .width(DesignConstants.LAYOUT.ICON_BUTTON_SIZE)
            .height(DesignConstants.LAYOUT.ICON_BUTTON_SIZE)
            .justifyContent(FlexAlign.Center)
            .borderRadius(DesignConstants.RADIUS.FULL)
            .backgroundColor(this.getPalette().surfaceBackground)
            .shadow(this.getPalette().controlShadow)
            .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getPalette().borderColor })
            .onClick(() => {
              if (this.onClose) {
                this.onClose();
              }
            })
          }

          Column({ space: 1 }) {
            Text('通知')
              .fontSize(DesignConstants.FONT.SIZE.XL)
              .fontWeight(DesignConstants.FONT.WEIGHT.BOLD)
              .fontColor(this.getPalette().textPrimary)
            if (this.getUnreadCount() > 0) {
              Text(`${this.getUnreadCount()} 条未读`)
                .fontSize(DesignConstants.FONT.SIZE.SM)
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
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .borderRadius(DesignConstants.RADIUS.FULL)
            .padding({ left: DesignConstants.SPACING.MD, right: DesignConstants.SPACING.MD, top: DesignConstants.SPACING.SM2, bottom: DesignConstants.SPACING.SM2 })
            .shadow(this.getPalette().controlShadow)
            .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getPalette().borderColor })
            .onClick(() => this.onMarkAllRead())
        }
      }
      .width('100%')
      .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG })

      // 过滤器标签
      Scroll() {
        Row({ space: DesignConstants.SPACING.SM }) {
          ForEach(this.filterOptions, (filter: NotificationFilterOption) => {
            Text(filter.label)
              .fontSize(DesignConstants.FONT.SIZE.SM)
              .fontColor(this.selectedFilter === filter.key
                ? this.getPalette().accentText
                : this.getPalette().textSecondary)
              .backgroundColor(this.selectedFilter === filter.key
                ? this.getPalette().accentBackground
                : this.getPalette().surfaceBackground)
              .borderRadius(DesignConstants.RADIUS.FULL)
              .padding({ left: DesignConstants.SPACING.MD, right: DesignConstants.SPACING.MD, top: DesignConstants.SPACING.SM2, bottom: DesignConstants.SPACING.SM2 })
              .shadow(this.selectedFilter === filter.key
                ? this.getPalette().accentShadow
                : this.getPalette().controlShadow)
              .border({
                width: 1,
                color: this.selectedFilter === filter.key
                  ? DesignConstants.COLOR.TRANSPARENT
                  : this.getPalette().borderColor
              })
              .onClick(() => this.onFilterSelected(filter.key as NotificationFilterKey))
          }, (filter: NotificationFilterOption) => filter.key + '_' + (this.selectedFilter === filter.key))
        }
        .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG })
      }
      .scrollable(ScrollDirection.Horizontal)
      .scrollBar(BarState.Off)

      // 内容区域
      if (this.isLoading) {
        Column({ space: DesignConstants.SPACING.MD }) {
          LoadingProgress()
            .width(DesignConstants.COMPONENT.AVATAR_SIZE.XS)
            .height(DesignConstants.COMPONENT.AVATAR_SIZE.XS)
            .color(this.getPalette().accentBackground)
          Text('加载中...')
            .fontSize(DesignConstants.FONT.SIZE.MD)
            .fontColor(this.getPalette().textSecondary)
        }
        .width('100%')
        .height(DesignConstants.COMPONENT.LOADING_AREA_HEIGHT)
        .justifyContent(FlexAlign.Center)
      } else if (this.visibleNotifications.length === 0 && this.errorMessage) {
        Column({ space: DesignConstants.SPACING.LG }) {
          SystemIcon({
            icon: AppIcon.WARNING,
            iconSize: 48,
            iconColor: this.getPalette().textSecondary
          })
          Text(this.errorMessage)
            .fontSize(DesignConstants.FONT.SIZE.LG)
            .fontColor(this.getPalette().textSecondary)
            .textAlign(TextAlign.Center)
          Button('重试')
            .type(ButtonType.Normal)
            .backgroundColor(this.getPalette().accentBackground)
            .fontColor(this.getPalette().accentText)
            .height(DesignConstants.COMPONENT.ACTION_BUTTON_HEIGHT_MD)
            .borderRadius(DesignConstants.RADIUS.XL)
            .onClick(() => this.refreshNotifications())
        }
        .width('100%')
        .layoutWeight(1)
        .justifyContent(FlexAlign.Center)
      } else if (this.visibleNotifications.length === 0) {
        Refresh({ refreshing: $$this.isRefreshing, builder: RefreshIndicator }) {
          Column({ space: DesignConstants.SPACING.MD }) {
            SystemIcon({
              icon: AppIcon.EMPTY,
              iconSize: 48,
              iconColor: this.getPalette().textTertiary
            })
            Text('暂无通知')
              .fontSize(DesignConstants.FONT.SIZE.LG)
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
          List({ space: DesignConstants.SPACING.MD }) {
            ForEach(this.getGroupedNotifications(), (group: GroupedNotification) => {
              ListItemGroup({
                header: this.buildGroupHeader(group.label),
                space: DesignConstants.SPACING.SM
              }) {
                ForEach(group.items, (notification: NotifyItem) => {
                  ListItem() {
                    this.buildNotifyItem(notification)
                  }
                }, (notification: NotifyItem) =>
                  notification.notify_id + '_p' + (this.actionLoadingIds.indexOf(notification.notify_id) >= 0))
              }
            }, (group: GroupedNotification) => group.label)

            if (this.isLoadingMore) {
              ListItem() {
                LoadingMoreIndicator({ message: '正在加载更多通知...' })
              }
            }
          }
          .width('100%')
          .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, bottom: DesignConstants.SPACING.XS })
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
      .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
      .fontColor(this.getPalette().textSecondary)
      .padding({ left: DesignConstants.SPACING.XS, top: DesignConstants.SPACING.SM, bottom: DesignConstants.SPACING.XS })
  }

  @Builder
  private buildNotifyItem(notification: NotifyItem) {
    Row({ space: DesignConstants.SPACING.MD3 }) {
      // 头像 + 类型图标徽章
      Stack({ alignContent: Alignment.Center }) {
        if (notification.payload?.actor?.avatar_url) {
          Image(notification.payload.actor.avatar_url)
            .width(DesignConstants.COMPONENT.AVATAR_SIZE.SMALL_PLUS)
            .height(DesignConstants.COMPONENT.AVATAR_SIZE.SMALL_PLUS)
            .borderRadius(DesignConstants.RADIUS.FULL)
            .objectFit(ImageFit.Cover)
        } else {
          Row() {
            Text() {
              SymbolSpan($r(this.getNotificationIcon(notification.type)))
                .fontSize(DesignConstants.FONT.SIZE.XL)
                .fontColor([this.getTypeColor(notification.type)])
            }
          }
          .width(DesignConstants.COMPONENT.AVATAR_SIZE.SMALL_PLUS)
          .height(DesignConstants.COMPONENT.AVATAR_SIZE.SMALL_PLUS)
          .borderRadius(DesignConstants.RADIUS.FULL)
          .justifyContent(FlexAlign.Center)
          .backgroundColor(this.getTypeColor(notification.type) + '18')
        }

        // 类型图标徽章：贴头像右下角（用 alignContent 取代硬编码坐标）
        Stack({ alignContent: Alignment.BottomEnd }) {
          Row() {
            Text() {
              SymbolSpan($r(this.getNotificationIcon(notification.type)))
                .fontSize(DesignConstants.FONT.SIZE.XS)
                .fontColor([this.getPalette().accentText])
            }
          }
          .width(DesignConstants.COMPONENT.LOADING_ICON_SIZE)
          .height(DesignConstants.COMPONENT.LOADING_ICON_SIZE)
          .borderRadius(DesignConstants.RADIUS.FULL)
          .justifyContent(FlexAlign.Center)
          .backgroundColor(this.getTypeColor(notification.type))
          .border({ width: DesignConstants.BORDER_WIDTH.STRONG, color: this.getPalette().pageBackground })
        }
        .width('100%')
        .height('100%')
      }

      // 文本内容
      Column({ space: DesignConstants.SPACING.HALF }) {
        Text() {
          Span(this.buildNotificationTitle(notification))
            .fontSize(DesignConstants.CHAT.BUBBLE_FONT_SIZE)
            .fontColor(this.getPalette().textPrimary)
            .fontWeight(notification.is_read ? DesignConstants.FONT.WEIGHT.REGULAR : DesignConstants.FONT.WEIGHT.MEDIUM)
          Span(' ' + this.buildNotificationMessage(notification))
            .fontSize(DesignConstants.CHAT.BUBBLE_FONT_SIZE)
            .fontColor(this.getPalette().textPrimary)
        }
        .maxLines(2)
        .textOverflow({ overflow: TextOverflow.Ellipsis })
        .lineHeight(DesignConstants.FONT.LINE_HEIGHT.XS)

        Text(this.formatTimestamp(notification.created_at))
          .fontSize(DesignConstants.FONT.SIZE.XS)
          .fontColor(this.getPalette().textSecondary)
          .margin({ top: DesignConstants.SPACING.HALF })

        // 群组操作按钮
        if (this.canHandleGroupInvite(notification)) {
          Row({ space: DesignConstants.SPACING.SM }) {
            Button(this.isActionLoading(notification.notify_id) ? '处理中...' : '接受')
              .type(ButtonType.Normal)
              .backgroundColor(this.getPalette().accentBackground)
              .fontColor(this.getPalette().accentText)
              .fontSize(DesignConstants.FONT.SIZE.XS)
              .height(DesignConstants.COMPONENT.ACTION_BUTTON_HEIGHT)
              .borderRadius(DesignConstants.RADIUS.FULL)
              .enabled(!this.isActionLoading(notification.notify_id))
              .onClick(() => this.handleGroupInviteAction(notification, 'accept'))

            Button('拒绝')
              .type(ButtonType.Normal)
              .backgroundColor(this.getPalette().surfaceSubtle)
              .fontColor(this.getPalette().textPrimary)
              .fontSize(DesignConstants.FONT.SIZE.XS)
              .height(DesignConstants.COMPONENT.ACTION_BUTTON_HEIGHT)
              .borderRadius(DesignConstants.RADIUS.FULL)
              .borderWidth(1)
              .borderColor(this.getPalette().borderColor)
              .enabled(!this.isActionLoading(notification.notify_id))
              .onClick(() => this.handleGroupInviteAction(notification, 'reject'))
          }
          .margin({ top: DesignConstants.SPACING.SM2 })
        } else if (this.canHandleGroupApplication(notification)) {
          Row({ space: DesignConstants.SPACING.SM }) {
            Button(this.isActionLoading(notification.notify_id) ? '处理中...' : '批准')
              .type(ButtonType.Normal)
              .backgroundColor(this.getPalette().accentBackground)
              .fontColor(this.getPalette().accentText)
              .fontSize(DesignConstants.FONT.SIZE.XS)
              .height(DesignConstants.COMPONENT.ACTION_BUTTON_HEIGHT)
              .borderRadius(DesignConstants.RADIUS.FULL)
              .enabled(!this.isActionLoading(notification.notify_id))
              .onClick(() => this.handleGroupApplicationAction(notification, 'approve'))

            Button('拒绝')
              .type(ButtonType.Normal)
              .backgroundColor(this.getPalette().surfaceSubtle)
              .fontColor(this.getPalette().textPrimary)
              .fontSize(DesignConstants.FONT.SIZE.XS)
              .height(DesignConstants.COMPONENT.ACTION_BUTTON_HEIGHT)
              .borderRadius(DesignConstants.RADIUS.FULL)
              .borderWidth(1)
              .borderColor(this.getPalette().borderColor)
              .enabled(!this.isActionLoading(notification.notify_id))
              .onClick(() => this.handleGroupApplicationAction(notification, 'reject'))
          }
          .margin({ top: DesignConstants.SPACING.SM2 })
        }
      }
      .layoutWeight(1)

      // 未读圆点
      if (!notification.is_read) {
        Row()
          .width(DesignConstants.SPACING.SM)
          .height(DesignConstants.SPACING.SM)
          .borderRadius(DesignConstants.RADIUS.FULL)
          .backgroundColor(this.getTypeColor(notification.type))
      }
    }
    .width('100%')
    .padding({ left: DesignConstants.SPACING.MD, right: DesignConstants.SPACING.MD, top: DesignConstants.SPACING.MD, bottom: DesignConstants.SPACING.MD })
    .backgroundColor(notification.is_read ? DesignConstants.COLOR.TRANSPARENT : this.getTypeColor(notification.type) + '07')
    .borderRadius(DesignConstants.SHAPE.COMPACT_CARD)
    .border({
      width: 1,
      color: notification.is_read ? DesignConstants.COLOR.TRANSPARENT : this.getTypeColor(notification.type) + '18'
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
      case 'report':
        return '举报';
      case 'nearby':
        return '附近';
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
    switch (this.mapTypeToCategory(type as NotifyType)) {
      case 'social':
        return this.getPalette().accentBackground;
      case 'interaction':
        return this.getPalette().error;
      case 'group':
        return this.getPalette().success;
      case 'system':
        return this.getPalette().warning;
      default:
        return this.getPalette().accentBackground;
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

  private buildNotificationTitle(notification: NotifyItem): string {
    switch (notification.type) {
      case 'new_follower':
        return '新的关注者';
      case 'mutual_follow':
        return '互相关注';
      case 'moment_commented':
        return '新的评论';
      case 'comment_reply':
        return '评论被回复';
      case 'mentioned':
        return '被提及';
      case 'reaction_added':
        return '收到表情反应';
      case 'group_invite_received':
        return '群组邀请';
      case 'group_mentioned':
        return '群聊提及';
      case 'group_application_received':
        return '新的入群申请';
      case 'group_application_accepted':
        return '申请已通过';
      case 'group_application_rejected':
        return '申请被拒绝';
      case 'report_resolved':
        return '举报已处理';
      case 'report_dismissed':
        return '举报被驳回';
      case 'nearby_new_moment':
        return '附近新动态';
      case 'system_announcement':
        return '系统通知';
      default:
        return '通知';
    }
  }

  private buildNotificationMessage(notification: NotifyItem): string {
    const payload = notification.payload;
    const actor = this.getActorName(payload);
    switch (notification.type) {
      case 'new_follower':
        return `${actor} 关注了你`;
      case 'mutual_follow':
        return `你与 ${actor} 已互相关注`;
      case 'moment_commented':
        return `${actor}: ${payload?.comment_preview ?? '发表了评论'}`;
      case 'comment_reply':
        return payload?.replied_comment_preview
          ? `${actor} 回复了你的评论：${payload.replied_comment_preview}`
          : `${actor}: ${payload?.reply_content_preview ?? '回复了你的评论'}`;
      case 'mentioned':
        return payload?.target_type === 'comment'
          ? `${actor} 在评论中提到了你`
          : `${actor} 在动态中提到了你`;
      case 'reaction_added':
        return payload?.emoji
          ? `${actor} 对你的内容送出了 ${payload.emoji}`
          : `${actor} 对你的内容添加了表情反应`;
      case 'group_invite_received':
        return `${payload?.group_name ?? '某群组'} 邀请你加入`;
      case 'group_mentioned':
        return payload?.content_preview
          ? `${actor} 在「${payload?.group_name ?? '群组'}」中 @ 了你：${payload.content_preview}`
          : `${actor} 在「${payload?.group_name ?? '群组'}」中 @ 了你`;
      case 'group_application_received':
        return `${actor} 申请加入 ${payload?.group_name ?? '群组'}`;
      case 'group_application_accepted':
        return `你加入了 ${payload?.group_name ?? '群组'}`;
      case 'group_application_rejected':
        return `你加入 ${payload?.group_name ?? '群组'} 的申请被拒绝`;
      case 'report_resolved':
        return payload?.note ?? '你的举报已处理完成';
      case 'report_dismissed':
        return payload?.note ?? '你的举报未通过审核';
      case 'nearby_new_moment':
        return `${actor} 在附近发布了新瞬间`;
      case 'system_announcement':
        return payload?.message ?? '请查看系统公告';
      default:
        return payload?.message ?? '';
    }
  }

  private getNotificationIcon(type: NotifyType): AppIcon {
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

  private getActorName(payload?: NotifyPayload): string {
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
    if (payload.rejected_by?.nickname) {
      return payload.rejected_by.nickname;
    }
    return '有人';
  }

  private mapTypeToCategory(type: NotifyType): NotificationFilterKey {
    switch (type) {
      case 'new_follower':
      case 'mutual_follow':
        return 'social';
      case 'moment_commented':
      case 'comment_reply':
      case 'mentioned':
      case 'reaction_added':
        return 'interaction';
      case 'group_invite_received':
      case 'group_mentioned':
      case 'group_application_received':
      case 'group_application_accepted':
      case 'group_application_rejected':
        return 'group';
      case 'report_resolved':
      case 'report_dismissed':
        return 'report';
      case 'nearby_new_moment':
        return 'nearby';
      case 'system_announcement':
        return 'system';
      default:
        return 'social';
    }
  }
}
