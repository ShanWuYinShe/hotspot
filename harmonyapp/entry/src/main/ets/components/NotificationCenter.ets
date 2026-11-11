/**
 * 通知中心（HTTP + WebSocket 联动）
 */
import { NotificationSyncService } from '../core/services/NotificationSyncService';
import { ApiClient } from '../core/api/ApiClient';
import { NotifyItem, NotifyPayload, NotifyType } from '../core/types/api';
import { SystemIcon, AppIcon } from './SystemIcon';
import { SystemButton } from './SystemButton';
import { TabsPillGroup, TabItem } from './TabsPillGroup';
import { AppRouter, AnnouncementsRouteParam, GroupDetailRouteParam, MomentDetailRouteParam, ProfileRouteParam } from '../core/router/AppRouter';
import { PageNames } from '../constants/AppConstants';
import { SystemToast } from './SystemToast';
import { LoginPromptView } from './LoginPromptView';
import { SystemAvatar } from './SystemAvatar';
import { BackToTopButton } from './BackToTopButton';
import { SurfaceIconButton } from './SurfaceIconButton';
import { SystemDialog } from './SystemDialog';
import { AccessibilityManager } from '../core/accessibility/AccessibilityManager';
import { MotionUtil } from '../core/accessibility/MotionUtil';
import { DesignConstants, lineHeightFor } from '../constants/DesignConstants';
import { AppThemeVisuals } from '../core/theme/AppThemeVisuals';
import { curves } from '@kit.ArkUI';
import { SmartNotificationService, NotificationEvent, SmartNotification } from '../core/services/SmartNotificationService';
import { LoadingMoreIndicator, RefreshIndicator } from './EmptyState';
import { AppThemeManager } from '../core/theme/AppThemeManager';
import { DEFAULT_RESOLVED_THEME_ID, KEY_RESOLVED_THEME_ID } from '../core/theme/AppThemeTypes';
import { LucideIcon } from '../components/LucideIcon';
import { MediaUrlUtil } from '../core/utils/MediaUrlUtil';
import { ImageUrlUtil } from '../core/utils/ImageUrlUtil';

// 过滤维度对齐前端 /notify FILTER_TABS（全部/点赞/关注/评论/附近）：
// 按「语义类型」分栏，而非「业务域」——域分组（互动/群组/举报/系统）是桌面 Rail 的视角，
// 移动端 TabBar 只给 5 项，两端信息架构必须一致。
type NotificationFilterKey = 'all' | 'reaction' | 'follow' | 'comment' | 'nearby';

/**
 * 列表条目：kind='header' 是卡外的分区标题（页面画布上），kind='group' 是收在
 * 软岛卡里的通知行集合。List 里用一条扁平条目流承载，标题与卡片才能交替出现
 * （对齐 web /notify：SectionHead 在卡外，ListContainer 是卡）。
 */
interface NotificationListEntry {
  key: string;
  kind: 'header' | 'group';
  label: string;
  count: number;
  showBell: boolean;
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
  @StorageLink('isLoggedIn') private isLoggedIn: boolean = false;
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = DEFAULT_RESOLVED_THEME_ID;
  onClose?: () => void;
  @State private visibleNotifications: NotifyItem[] = [];
  @State private selectedFilter: NotificationFilterKey = 'all';
  @State private isLoading: boolean = false;
  @State private enterOpacity: number = 1;
  @State private enterOffset: number = 0;
  @State private showBackToTop: boolean = false;
  @State private viewportHeight: number = 0;
  private backToTopScroller: Scroller = new Scroller();
  private enterPlayed: boolean = false;

  @State private isRefreshing: boolean = false;
  @State private isLoadingMore: boolean = false;
  @State private hasMore: boolean = false;
  @State private errorMessage: string = '';
  @State private actionLoadingIds: string[] = [];
  // 全部已读确认弹窗（对齐全站 SystemDialog 范式，替代系统 showAlertDialog）
  @State private markAllDialogVisible: boolean = false;
  @State private pressedNotificationId: string = '';

  private syncService = NotificationSyncService.getInstance();
  private accessibilityManager = AccessibilityManager.getInstance();
  private smartNotificationService = SmartNotificationService.getInstance();
  private filterOptions: NotificationFilterOption[] = [
    new NotificationFilterOption('all', '全部'),
    // 点赞语义 = reaction_added（契约已删 moment_liked），Tab 文案对齐前端「点赞」
    new NotificationFilterOption('reaction', '点赞'),
    new NotificationFilterOption('follow', '关注'),
    new NotificationFilterOption('comment', '评论'),
    new NotificationFilterOption('nearby', '附近')
  ];

  private syncListener: (() => void) | null = null;
  private notificationCreatedListener: ((event: NotificationEvent, notification: SmartNotification, extra?: Object) => void) | null = null;

  /**
   * 入群邀请 / 入群申请：行尾带接受·拒绝双按钮，单独成组（对齐 web InvitesPendingCard
   * 把「待处理邀请」从通知流里提出来，避免邀请被已读通知淹没）。
   */
  private isInviteNotification(notification: NotifyItem): boolean {
    return this.canHandleGroupInvite(notification) || this.canHandleGroupApplication(notification);
  }

  /**
   * 分组对齐前端 /notify：入群邀请 → 新消息（未读）→ 已读。
   * 原实现按「今天 / 昨天 / 更早」时间分组，信息架构与 web 不同：同一屏既读不出
   * 「哪条还没处理」，也读不出「哪些是新消息」。
   * 每个分区输出两条：卡外标题 + 收行的软岛卡。
   */
  private buildNotificationListEntries(): NotificationListEntry[] {
    const invites: NotifyItem[] = [];
    const unread: NotifyItem[] = [];
    const read: NotifyItem[] = [];
    for (const notification of this.visibleNotifications) {
      if (this.isInviteNotification(notification)) {
        invites.push(notification);
      } else if (!notification.is_read) {
        unread.push(notification);
      } else {
        read.push(notification);
      }
    }
    const entries: NotificationListEntry[] = [];
    const pushSection = (key: string, label: string, items: NotifyItem[], showBell: boolean): void => {
      entries.push({ key: `${key}-header`, kind: 'header', label: label, count: items.length, showBell: showBell, items: [] });
      entries.push({ key: key, kind: 'group', label: label, count: items.length, showBell: showBell, items: items });
    };
    if (invites.length > 0) {
      pushSection('invites', '入群邀请', invites, false);
    }
    if (unread.length > 0) {
      pushSection('unread', '新消息', unread, true);
    }
    if (read.length > 0) {
      pushSection('read', '已读', read, false);
    }
    return entries;
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
      this.resetContentEnter();
      this.playContentEnter();
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

    if (this.selectedFilter !== 'all') {
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

  private playContentEnter(): void {
    if (this.enterPlayed) return;
    this.enterPlayed = true;
    if (MotionUtil.reduced()) {
      this.enterOpacity = 1;
      this.enterOffset = 0;
      return;
    }
    this.enterOpacity = 0;
    this.enterOffset = MotionUtil.enterOffset();
    MotionUtil.contentEnter(this.getUIContext(), () => {
      this.enterOpacity = 1;
      this.enterOffset = 0;
    });
  }

  private resetContentEnter(): void {
    this.enterPlayed = false;
    this.enterOpacity = 1;
    this.enterOffset = 0;
  }

  private updateBackToTop(): void {
    const offset: OffsetResult | undefined = this.backToTopScroller.currentOffset();
    const y: number = offset ? offset.yOffset : 0;
    const threshold: number = this.viewportHeight > 0 ? this.viewportHeight * 1.5 : 1200;
    const next: boolean = y > threshold;
    if (next !== this.showBackToTop) {
      this.showBackToTop = next;
    }
  }

  private scrollToTop(): void {
    this.showBackToTop = false;
    this.backToTopScroller.scrollTo({
      xOffset: 0,
      yOffset: 0,
      animation: { duration: MotionUtil.reduced() ? 0 : 300, curve: Curve.EaseInOut }
    });
  }

  @Builder
  private buildBackToTopOverlay() {
    BackToTopButton({
      visible: this.showBackToTop,
      onTap: () => { this.scrollToTop(); }
    })
      .margin({ right: DesignConstants.SPACING.LG, bottom: DesignConstants.SPACING.XXL })
  }

  build() {
    Column({ space: DesignConstants.SPACING.LG }) {
      // 标题栏（对齐 web /notify PageHeader）：36 圆返回钮 + 标题 20/600 +
      // 未读胶囊（web subtitle pill）+ 右侧「全部已读」安静动作
      Row({ space: DesignConstants.SPACING.MD }) {
        if (this.onClose) {
          // 返回钮统一走 SurfaceIconButton（36 圆钮 + 发丝描边 + 轻投影 + 44 热区 + 无障碍名）
          SurfaceIconButton({
            icon: AppIcon.BACK,
            kind: 'surface',
            buttonSize: DesignConstants.LAYOUT.ICON_BUTTON_SIZE,
            accessibilityLabel: '返回',
            onPress: () => {
              if (this.onClose) {
                this.onClose();
              }
            }
          })
        }

        Column({ space: DesignConstants.SPACING.XS }) {
          Text('通知')
            .fontSize(DesignConstants.FONT.SIZE.XL)
            .fontWeight(DesignConstants.FONT.WEIGHT.SEMIBOLD)
            .fontColor(this.getPalette().textPrimary)
          // 对齐 web /notify PageHeader：副标题为标题下方裸文本（非胶囊）
          Text(this.getUnreadCount() > 0 ? `${this.getUnreadCount()} 条未读消息` : '暂无新通知')
            .fontSize(DesignConstants.FONT.SIZE.XS)
            .fontColor(this.getPalette().textSecondary)
            .maxLines(1)
        }
        .alignItems(HorizontalAlign.Start)
        .layoutWeight(1)

        if (this.getUnreadCount() > 0) {
          SystemButton({
            label: '全部已读',
            icon: AppIcon.CHECK_ALL,
            showIcon: true,
            // web 是安静的文字 + 图标动作（trailing soft/ghost），不是页头主操作
            variant: 'ghost',
            buttonSize: 'small',
            // 批量标记不可撤销：二次确认防误触
            onPress: () => {
              this.markAllDialogVisible = true;
            }
          })
        }
      }
      .width('100%')
      .alignItems(VerticalAlign.Center)
      .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG })

      // 过滤器标签（游客态隐藏，对齐 web 通知访客引导无筛选行）
      if (this.isLoggedIn) {
        TabsPillGroup({
          variant: 'pills',
          tabs: this.filterOptions.map((filter: NotificationFilterOption) => ({ label: filter.label } as TabItem)),
          activeIndex: this.filterOptions.findIndex((filter: NotificationFilterOption) => filter.key === this.selectedFilter),
          actions: {
            onChange: (index: number) => {
              const option = this.filterOptions[index];
              if (option) {
                this.onFilterSelected(option.key as NotificationFilterKey);
              }
            }
          }
        })
          .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, bottom: DesignConstants.SPACING.SM })
      }

      // 内容区域
      if (!this.isLoggedIn) {
        // 游客态：对齐 web /notify 的 NeedsAuthNotice 内联引导，而不是把 10100 当普通错误
        LoginPromptView({
          title: '登录后查看通知',
          subtitle: '收到的点赞、评论与关注会出现在这里',
          icon: AppIcon.PERSON,
          loginAction: '查看通知',
          highlights: [
            { icon: AppIcon.HEART, text: '谁点赞了你的瞬间' },
            { icon: AppIcon.PERSON_ADD, text: '新的关注与互相关注' },
            { icon: AppIcon.GROUP, text: '入群邀请与群申请结果' }
          ]
        })
          .layoutWeight(1)
      } else if (this.isLoading) {
        Column({ space: DesignConstants.SPACING.MD }) {
          LoadingProgress()
            .width(DesignConstants.COMPONENT.AVATAR_SIZE.XS)
            .height(DesignConstants.COMPONENT.AVATAR_SIZE.XS)
            .color(this.getPalette().accentBackground)
          Text('加载中…')
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
            iconSize: DesignConstants.COMPONENT.ICON_SIZE_EMPTY,
            iconColor: this.getPalette().textSecondary
          })
          Text(this.errorMessage)
            .fontSize(DesignConstants.FONT.SIZE.LG)
            .fontColor(this.getPalette().textSecondary)
            .textAlign(TextAlign.Center)
          SystemButton({
            label: '重试',
            variant: 'primary',
            buttonSize: 'small',
            onPress: () => this.refreshNotifications()
          })
        }
        .width('100%')
        .layoutWeight(1)
        .justifyContent(FlexAlign.Center)
      } else if (this.visibleNotifications.length === 0) {
        Refresh({ refreshing: $$this.isRefreshing, builder: RefreshIndicator }) {
          Column({ space: DesignConstants.SPACING.MD }) {
            SystemIcon({
              icon: AppIcon.EMPTY,
              iconSize: DesignConstants.COMPONENT.ICON_SIZE_EMPTY,
              iconColor: this.getPalette().textTertiary
            })
            Text('暂无通知')
              .fontSize(DesignConstants.FONT.SIZE.LG)
              .fontColor(this.getPalette().textSecondary)
            // 空态指引（r39）：告知接收范围可在应用内调整
            Text('可在「设置-通知偏好」调整要接收的通知')
              .fontSize(DesignConstants.FONT.SIZE.SM)
              .fontColor(this.getPalette().textTertiary)
          }
          .width('100%')
          .height('100%')
          .justifyContent(FlexAlign.Center)
        }
        .pullToRefresh(false)
        .refreshOffset(64)
        .onRefreshing(() => {
          if (!this.isRefreshing) {
            this.isRefreshing = true;
            this.refreshNotifications();
          }
        })
      } else {
        Refresh({ refreshing: $$this.isRefreshing, builder: RefreshIndicator }) {
          List({ space: DesignConstants.SPACING.MD, scroller: this.backToTopScroller }) {
            ForEach(this.buildNotificationListEntries(), (entry: NotificationListEntry) => {
              if (entry.kind === 'header') {
                // 分区标题在卡外（页面画布上），与 web SectionHead 位置一致：
                // 卡外标题 + 卡内行，标题读起来是「分组」而不是「卡片标题」
                ListItem() {
                  this.buildGroupHeader(entry.label, entry.count, entry.showBell)
                }
              } else {
                ListItemGroup({
                  // web 软岛内行间距 gap-y-0.5(2)：行本身已有 14 上下内距
                  space: DesignConstants.SPACING.HALF
                }) {
                  ForEach(entry.items, (notification: NotifyItem) => {
                    ListItem() {
                      this.buildNotifyItem(notification)
                    }
                  }, (notification: NotifyItem) =>
                    notification.notify_id + '_p' + (this.actionLoadingIds.indexOf(notification.notify_id) >= 0))
                }
                .width('100%')
                // r3 容器式软岛：每组通知一张软岛卡（surfaceIsland + 20 圆角 + 轻投影），
                // 水平留白由 List 的 gutter 承担（卡与外层内容同宽，不再自加 margin）
                .backgroundColor(this.getPalette().surfaceIsland)
                .borderRadius(DesignConstants.SHAPE.CARD)
                .shadow(this.getPalette().surfaceShadow)
                // 岛内上下留 8（web ListContainer p-2）；横向由行自身 margin 8 承担，
                // 未读竖条因此离岛边 8，与 web 行内状态条位置一致
                .padding({ top: DesignConstants.SPACING.SM, bottom: DesignConstants.SPACING.SM })
                .clip(true)
              }
            }, (entry: NotificationListEntry) => entry.key)

            if (this.isLoadingMore) {
              ListItem() {
                LoadingMoreIndicator({ message: '正在加载更多通知…' })
              }
            }
          }
          .edgeEffect(EdgeEffect.None)
          .scrollBar(BarState.Off)
          .width('100%')
          .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, bottom: DesignConstants.SPACING.LG })
          .onReachEnd(() => this.loadMoreNotifications())
          .onDidScroll(() => { this.updateBackToTop(); })
          .onAreaChange((oldArea: Area, newArea: Area) => { this.viewportHeight = Number(newArea.height); })
        }
        .layoutWeight(1)
        .pullToRefresh(false)
        .refreshOffset(64)
.onRefreshing(() => {
          if (!this.isRefreshing) {
            this.isRefreshing = true;
            this.refreshNotifications();
          }
        })
        .opacity(this.enterOpacity)
        .translate({ y: this.enterOffset })
        .overlay(this.buildBackToTopOverlay(), { align: Alignment.BottomEnd })
      }

      // 全部已读二次确认（与全站确认弹窗视觉一致）
      SystemDialog({
        visible: this.markAllDialogVisible,
        title: '全部标记已读',
        description: `将 ${this.getUnreadCount()} 条通知标记为已读，此操作不可撤销。`,
        confirmText: '标记',
        showCancel: true,
        onCancel: () => {
          this.markAllDialogVisible = false;
        },
        onConfirm: () => {
          this.markAllDialogVisible = false;
          this.onMarkAllRead();
        }
      })
    }
    .width('100%')
    .height('100%')
    .backgroundColor(this.getPalette().pageBackground)
  }

  @Builder
  private buildGroupHeader(label: string, count: number, showBell: boolean) {
    // 分组标题对齐 web SectionHead variant=eyebrow：10vp 次色小标 + 右侧计数（11vp），
    // 「新消息」组带铃铛图标。旧实现是「accent 圆点 + 12vp 主文字」，比正文还抢眼。
    Row({ space: DesignConstants.SPACING.SM }) {
      if (showBell) {
        LucideIcon({
          icon: AppIcon.NOTIFICATION,
          iconSize: DesignConstants.COMPONENT.ICON_SIZE_XS,
          paletteRole: 'textSecondary'
        })
      }
      Text(label)
        .fontSize(DesignConstants.FONT.SIZE.XS2)
        .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
        .fontColor(this.getPalette().textSecondary)
        .textCase(TextCase.UpperCase)
        .letterSpacing(DesignConstants.FONT.LETTER_SPACING.EYEBROW)
        .lineHeight(lineHeightFor(DesignConstants.FONT.SIZE.XS2, DesignConstants.FONT.LINE_RATIO.DEFAULT))
      Text(`${count}`)
        .fontSize(DesignConstants.FONT.SIZE.XS)
        .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
        .fontColor(this.getPalette().textSecondary)
    }
    .alignItems(VerticalAlign.Center)
    // 与下方软岛卡左缘对齐（横向留白由 List 的 gutter 承担，分区标题不再自加 left）
    .padding({
      top: DesignConstants.SPACING.SM,
      bottom: DesignConstants.SPACING.SM2
    })
  }

  @Builder
  private buildNotifyItem(notification: NotifyItem) {
    Stack({ alignContent: Alignment.Start }) {
      // 未读条：左侧 3vp accent 竖条（对齐 web NotifyRow；静态、不抢注意力，
      // 与列表的扫读方向一致）。替代旧的「整行底色 + 右侧类型色圆点」双信号。
      if (!notification.is_read) {
        Row()
          .width(DesignConstants.SPACING.THREE)
          .height(DesignConstants.COMPONENT.AVATAR_SIZE.SMALL_PLUS)
          .borderRadius(DesignConstants.RADIUS.FULL)
          .backgroundColor(this.getPalette().accentBackground)
      }

      Row({ space: DesignConstants.SPACING.MD3 }) {
        // 头像 + 类型图标徽章
        Stack({ alignContent: Alignment.Center }) {
          if (notification.payload?.actor?.avatar_url) {
            // R8：统一 Avatar 原语（底层首字母/人形图标占位 + 渐显 + 失败保留占位）
            // 相邻正文已含 actor 姓名 → decorative 移出无障碍树
            SystemAvatar({
              avatarUrl: notification.payload.actor.avatar_url,
              name: notification.payload.actor.nickname ?? '',
              avatarSize: DesignConstants.COMPONENT.AVATAR_SIZE.SMALL_PLUS,
              shape: 'circle',
              decorative: true
            })
          } else {
            Row() {
              LucideIcon({
                icon: this.getNotificationIcon(notification.type),
                iconSize: DesignConstants.COMPONENT.ICON_SIZE_SM,
                color: this.getTypeOnSoftColor(notification.type)
              })
            }
            .width(DesignConstants.COMPONENT.AVATAR_SIZE.SMALL_PLUS)
            .height(DesignConstants.COMPONENT.AVATAR_SIZE.SMALL_PLUS)
            .borderRadius(DesignConstants.RADIUS.FULL)
            .justifyContent(FlexAlign.Center)
            .backgroundColor(this.getSoftStatusColor(notification.type))
          }

          // 类型图标徽章：贴头像右下角（用 alignContent 取代硬编码坐标）
          Stack({ alignContent: Alignment.BottomEnd }) {
            Row() {
              LucideIcon({
                icon: this.getNotificationIcon(notification.type),
                iconSize: DesignConstants.COMPONENT.ICON_SIZE_2XS,
                color: this.getPalette().accentText
              })
            }
            .width(DesignConstants.COMPONENT.LOADING_ICON_SIZE)
            .height(DesignConstants.COMPONENT.LOADING_ICON_SIZE)
            .borderRadius(DesignConstants.RADIUS.FULL)
            .justifyContent(FlexAlign.Center)
            .backgroundColor(this.getTypeColor(notification.type))
            .border({ width: DesignConstants.BORDER_WIDTH.STRONG, color: this.getPalette().pageBackground })
          }
          // 徽章层必须与头像同尺寸：用 100% 会撑满整行/整卡，导致通知行高度塌陷成空白大卡
          .width(DesignConstants.COMPONENT.AVATAR_SIZE.SMALL_PLUS)
          .height(DesignConstants.COMPONENT.AVATAR_SIZE.SMALL_PLUS)
        }

        // 文本内容：未读 = 标题 semibold 主色；已读 = 标题 medium 次色（对齐 web）
        Column({ space: DesignConstants.SPACING.HALF }) {
          Text() {
            Span(this.buildNotificationTitle(notification))
              .fontSize(DesignConstants.FONT.SIZE.SM)
              .fontColor(notification.is_read ? this.getPalette().textSecondary : this.getPalette().textPrimary)
              .fontWeight(notification.is_read ? DesignConstants.FONT.WEIGHT.MEDIUM : DesignConstants.FONT.WEIGHT.SEMIBOLD)
            if (this.buildNotificationMessage(notification).length > 0) {
              Span(' ' + this.buildNotificationMessage(notification))
                .fontSize(DesignConstants.FONT.SIZE.SM)
                .fontColor(this.getPalette().textPrimary)
            }
          }
          .maxLines(2)
          .textOverflow({ overflow: TextOverflow.Ellipsis })
          .lineHeight(DesignConstants.FONT.LINE_HEIGHT.XS)

          Text(this.formatTimestamp(notification.created_at))
            .fontSize(DesignConstants.FONT.SIZE.XS)
            .fontColor(this.getPalette().textSecondary)
            .margin({ top: DesignConstants.SPACING.HALF })
        }
        .layoutWeight(1)
        .alignItems(HorizontalAlign.Start)

        // 入群邀请 / 入群申请双按钮：行尾竖排（对齐 web 移动端 flex-col gap-1.5），
        // 主 = accent 实底 pill、次 = soft 软填 pill，两枚同档视觉重量。
        if (this.canHandleGroupInvite(notification)) {
          Column({ space: DesignConstants.SPACING.SM2 }) {
            SystemButton({
              label: '接受',
              variant: 'primary',
              buttonSize: 'small',
              pill: true,
              loading: this.isActionLoading(notification.notify_id),
              disabled: this.isActionLoading(notification.notify_id),
              onPress: () => this.handleGroupInviteAction(notification, 'accept')
            })

            SystemButton({
              label: '拒绝',
              variant: 'surface',
              buttonSize: 'small',
              pill: true,
              disabled: this.isActionLoading(notification.notify_id),
              onPress: () => this.handleGroupInviteAction(notification, 'reject')
            })
          }
          .justifyContent(FlexAlign.Center)
        } else if (this.canHandleGroupApplication(notification)) {
          Column({ space: DesignConstants.SPACING.SM2 }) {
            SystemButton({
              label: '批准',
              variant: 'primary',
              buttonSize: 'small',
              pill: true,
              loading: this.isActionLoading(notification.notify_id),
              disabled: this.isActionLoading(notification.notify_id),
              onPress: () => this.handleGroupApplicationAction(notification, 'approve')
            })

            SystemButton({
              label: '拒绝',
              variant: 'surface',
              buttonSize: 'small',
              pill: true,
              disabled: this.isActionLoading(notification.notify_id),
              onPress: () => this.handleGroupApplicationAction(notification, 'reject')
            })
          }
          .justifyContent(FlexAlign.Center)
        }
      }
      .width('100%')
      .alignItems(VerticalAlign.Center)
      // web NotifyRow：px-4 py-3.5（16/14）
      .padding({
        left: DesignConstants.SPACING.LG,
        right: DesignConstants.SPACING.LG,
        top: DesignConstants.SPACING.MD3,
        bottom: DesignConstants.SPACING.MD3
      })
    }
    .width('100%')
    // 行底色不再随未读变化（未读语义只由左侧竖条 + 标题字重承担）；行内缩 8 离岛边
    .backgroundColor(DesignConstants.COLOR.TRANSPARENT)
    .borderRadius(DesignConstants.RADIUS.SM)
    .margin({ left: DesignConstants.SPACING.SM, right: DesignConstants.SPACING.SM })
    // 整行按压反馈（对齐 DesktopLayout 导航项 tap-97 语义）
    .scale(this.pressedNotificationId === notification.notify_id ?
      { x: DesignConstants.INTERACTION.SCALE_PRESSED_CARD, y: DesignConstants.INTERACTION.SCALE_PRESSED_CARD } :
      { x: 1, y: 1 })
    .animation({ curve: curves.springMotion(
      DesignConstants.INTERACTION.SPRING_RESPONSE_PRESS,
      DesignConstants.INTERACTION.SPRING_DAMPING_PRESS
    ) })
    .onTouch((event: TouchEvent) => {
      if (event.type === TouchType.Down) {
        this.pressedNotificationId = notification.notify_id;
      } else if (event.type === TouchType.Up || event.type === TouchType.Cancel) {
        this.pressedNotificationId = '';
      }
    })
    .onClick(() => this.handleNotificationClick(notification))
    .accessibilityGroup(true)
    .accessibilityText(`${notification.is_read ? '' : '未读'}通知：${this.buildNotificationTitle(notification)}。${this.buildNotificationMessage(notification)}`)
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

  /** 语义分类 → 主色（对齐前端 TYPE_CONFIG 的 color 角色）：
   *  reaction=statusDanger（点赞爱心红）、follow=accent、comment=markerUser、nearby=statusOnline */
  private getTypeColor(type: string): string {
    switch (this.mapTypeToCategory(type as NotifyType)) {
      case 'reaction':
        return this.getPalette().error;
      case 'comment':
        return this.getPalette().accentBackground;
      case 'nearby':
        return this.getPalette().online;
      case 'follow':
      default:
        return this.getPalette().accentBackground;
    }
  }

  /** R12：soft 底上的类型图标用 on-soft 色（web text-status-*-on-soft 同口径） */
  private getTypeOnSoftColor(type: string): string {
    switch (this.mapTypeToCategory(type as NotifyType)) {
      case 'reaction':
        return this.getPalette().textStatusDangerOnSoft;
      case 'nearby':
        return this.getPalette().textStatusOnlineOnSoft;
      default:
        return this.getPalette().textAccentOnSoft;
    }
  }

  private getSoftStatusColor(type: string): string {
    switch (this.mapTypeToCategory(type as NotifyType)) {
      case 'reaction':
        return AppThemeManager.getSoftStatusBackground('danger');
      case 'nearby':
        return AppThemeManager.getSoftStatusBackground('success');
      default:
        return AppThemeManager.getSoftStatusBackground('info');
    }
  }

  private formatTimestamp(iso: string): string {
    try {
      const date = new Date(iso);
      if (Number.isNaN(date.getTime())) {
        return iso;
      }
      // 对齐 web：11月10日 09:50（月/日不补零，时分补零）
      const month = date.getMonth() + 1;
      const day = date.getDate();
      const hour = this.padNumber(date.getHours());
      const minute = this.padNumber(date.getMinutes());
      return `${month}月${day}日 ${hour}:${minute}`;
    } catch (_) {
      return iso;
    }
  }

  private padNumber(value: number): string {
    return value < 10 ? `0${value}` : `${value}`;
  }

  private buildNotificationTitle(notification: NotifyItem): string {
    switch (notification.type) {
      // web 关注类通知标题就是「actor 关注了你 / actor 和你互相关注了」
      case 'new_follower':
        return `${this.getActorName(notification.payload)} 关注了你`;
      case 'mutual_follow':
        return `${this.getActorName(notification.payload)} 和你互相关注了`;
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
        return '';
      case 'mutual_follow':
        return '';
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

  /** 语义分类 → 图标（对齐前端 TYPE_CONFIG：Heart / UserPlus / MessageCircle / MapPin） */
  private getNotificationIcon(type: NotifyType): AppIcon {
    switch (this.mapTypeToCategory(type)) {
      case 'reaction':
        return AppIcon.HEART;
      case 'follow':
        return AppIcon.PERSON_ADD;
      case 'comment':
        return AppIcon.MESSAGE;
      case 'nearby':
        return AppIcon.LOCATION;
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

  /**
   * 后端 NotifyType → 语义分类键。
   * 与前端 frontend/src/routes/(app)/notify/shared/notify-data.ts 的
   * NOTIFY_TYPE_TO_CATEGORY 逐条一致（含群相关通知归入「评论」、举报/公告归入默认档）：
   * 两端同一后端事件在两端的过滤维度必须落在同一格，否则同一个通知在 web 与 app 里
   * 会被归到不同 Tab，用户切端后找不到刚看过的通知。
   */
  private mapTypeToCategory(type: NotifyType): NotificationFilterKey {
    switch (type) {
      case 'reaction_added':
        return 'reaction';
      case 'new_follower':
      case 'mutual_follow':
        return 'follow';
      case 'moment_commented':
      case 'comment_reply':
      case 'mentioned':
      // 群内被提及：前端归一为 comment（评论语义），不是群组域
      case 'group_mentioned':
        return 'comment';
      case 'nearby_new_moment':
        return 'nearby';
      default:
        // group_* / report_* / system_announcement 等无独立 Tab，归入「全部」
        return 'all';
    }
  }
}
