/**
 * 首页 - 对齐前端 Home 页面布局
 * 粘性头部 + 地图卡片 + 附近用户列表 + 活动网格
 */

import { GlobalStateManager, KEY_IS_LOGGED_IN, KEY_BROWSE_ANCHOR, KEY_CURRENT_LOCATION } from '../../../core/state/GlobalStateManager';
import { ResponsiveContainer } from '../../../components/ResponsiveContainer';
import { isWideForm } from '../../../core/responsive/Breakpoint';
import type { BreakpointType } from '../../../core/responsive/Breakpoint';
import { LocationInfo, LocationUtil } from '../../../core/location/LocationUtil';
import { ApiClient } from '../../../core/api/ApiClient';
import { AppRouter, ChatRouteParam, GroupDetailRouteParam, LoginRouteParam, MomentDetailRouteParam, ProfileRouteParam, SearchRouteParam } from '../../../core/router/AppRouter';
import { PageNames, DefaultConfig } from '../../../constants/AppConstants';
import {
  MomentItem, UserItem, GroupItem, GeoPoint,
  SearchMomentsRequest, SearchUsersRequest, SearchGroupsRequest,
  JoinGroupRequest,
  BlockRequest, UnblockRequest, CreateReportRequest, Relationship
} from '../../../core/types/api';
import { SystemAvatar } from '../../../components/SystemAvatar';
import { MomentFeedCard } from '../../../components/MomentFeedCard';
import { SystemToast } from '../../../components/SystemToast';
import { SystemPageHeader } from '../../../components/SystemPageHeader';
import { InteractionStateStore, UserRelationState } from '../../../core/state/InteractionStateStore';
import { interactionActions } from '../../../core/state/InteractionActions';
import { DesignConstants } from '../../../constants/DesignConstants';
import { AppIcon } from '../../../components/SystemIcon';
import { LoadingUtil } from '../../../core/utils/LoadingUtil';
import { BasicDataSource } from '../../../core/utils/BasicDataSource';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { WidgetDataService } from '../../../core/services/WidgetDataService';
import { SystemUiController } from '../../../core/utils/SystemUiController';
import { SafeAreaUtils } from '../../../core/utils/SafeAreaUtils';
import { formatCount } from '../../../core/utils/FormatUtil';
import pasteboard from '@ohos.pasteboard';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { KEY_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';

const DOMAIN_NUMBER: number = 0xFF10;
const TAG: string = 'NearbyPage';

interface StatItem {
  label: string;
  value: string;
}

@Preview
@Component
export struct NearbyPage {
  @StorageLink(KEY_IS_LOGGED_IN) isLoggedIn: boolean = false;
  @StorageLink(KEY_CURRENT_LOCATION) @Watch('onCurrentLocationChange') currentLocation: LocationInfo | null = null;
  @StorageLink(KEY_BROWSE_ANCHOR) @Watch('onBrowseAnchorChange') browseAnchor: LocationInfo | null = null;
  @StorageLink('safeAreaTop') safeAreaTop: number = 0;
  @StorageLink('safeAreaBottom') safeAreaBottom: number = 0;
  @StorageLink(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'default_clean_light';
  @StorageLink('interaction_user_state') userStateMap: Record<string, UserRelationState> = {};
  @StorageProp('currentBreakpoint') currentBreakpoint: BreakpointType = 'sm';

  @State moments: MomentItem[] = [];
  @State users: UserItem[] = [];
  @State groups: GroupItem[] = [];

  @State momentsLoading: boolean = true;
  @State usersLoading: boolean = true;
  @State groupsLoading: boolean = true;

  @State momentsError: string = '';
  @State usersError: string = '';
  @State groupsError: string = '';

  @State userActionPendingIds: string[] = [];
  @State groupActionPendingIds: string[] = [];

  private interactionStore = InteractionStateStore.getInstance();
  private isDestroyed: boolean = false;
  private initialLoadDone: boolean = false;

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  private getOnlineCount(): number {
    if (!this.users || this.users.length === 0) return 0;
    return this.users.filter((u: UserItem) => u.presence_status === 'online').length;
  }

  async aboutToAppear() {
    SystemUiController.syncWithCurrentTheme();
    this.loadNearbyMoments();
    this.loadNearbyUsers();
    this.loadNearbyGroups();
    this.initialLoadDone = true;
  }

  aboutToDisappear() {
    this.isDestroyed = true;
  }

  private onCurrentLocationChange(): void {
    if (!this.initialLoadDone || this.browseAnchor || !this.currentLocation) return;
    if (this.moments.length === 0 && this.users.length === 0 && this.groups.length === 0) {
      this.loadNearbyMoments();
      this.loadNearbyUsers();
      this.loadNearbyGroups();
    }
  }

  private onBrowseAnchorChange(): void {
    if (!this.initialLoadDone) return;
    this.loadNearbyMoments();
    this.loadNearbyUsers();
    this.loadNearbyGroups();
  }

  private getReferencePoint(): GeoPoint | null {
    const loc: LocationInfo | null = this.browseAnchor ?? this.currentLocation;
    if (!loc) return null;
    return { latitude: loc.latitude, longitude: loc.longitude };
  }

  private getHeaderTopPadding(): number {
    return SafeAreaUtils.resolveTopPadding(this.safeAreaTop);
  }

  // ==================== 数据加载 ====================

  private async loadNearbyMoments(): Promise<void> {
    const ref = this.getReferencePoint();
    if (!ref) { this.momentsLoading = false; return; }
    this.momentsLoading = true;
    this.momentsError = '';
    try {
      const request: SearchMomentsRequest = {
        reference_point: ref, radius: DefaultConfig.NEARBY_RADIUS,
        sort_by: 'distance', pagination: { limit: 10 }
      };
      const response = await LoadingUtil.withMinimumTime(() => ApiClient.searchMoments(request));
      if (this.isDestroyed) return;
      if (response.code === 0) {
        const items = response.data?.items ?? [];
        this.moments = items.filter((m: MomentItem) => m.moment_id && m.user);
        this.moments.forEach((item: MomentItem) => this.interactionStore.patchMomentFromItem(item));
      } else {
        this.momentsError = response.message || '加载失败';
      }
    } catch {
      this.momentsError = '加载附近动态失败';
    } finally {
      this.momentsLoading = false;
    }
  }

  private async loadNearbyUsers(): Promise<void> {
    const ref = this.getReferencePoint();
    if (!ref) { this.usersLoading = false; return; }
    this.usersLoading = true;
    this.usersError = '';
    try {
      const request: SearchUsersRequest = {
        reference_point: ref, radius: DefaultConfig.NEARBY_RADIUS,
        sort_by: 'distance', pagination: { limit: 10 }
      };
      const response = await LoadingUtil.withMinimumTime(() => ApiClient.searchUsers(request));
      if (this.isDestroyed) return;
      if (response.code === 0) {
        const items = response.data?.items ?? [];
        const me = GlobalStateManager.getInstance().getCurrentUser();
        const myId = me?.user_id;
        const valid = items.filter((u: UserItem) => u.user?.user_id);
        this.users = typeof myId === 'string'
          ? valid.filter((u: UserItem) => u.user.user_id !== myId)
          : valid;
        this.users.forEach((u: UserItem) => this.interactionStore.patchUserFromItem(u));
      } else {
        this.usersError = response.message || '加载失败';
      }
    } catch {
      this.usersError = '加载附近用户失败';
    } finally {
      this.usersLoading = false;
    }
  }

  private async loadNearbyGroups(): Promise<void> {
    const ref = this.getReferencePoint();
    if (!ref) { this.groupsLoading = false; return; }
    this.groupsLoading = true;
    this.groupsError = '';
    try {
      const request: SearchGroupsRequest = {
        reference_point: ref, radius: DefaultConfig.NEARBY_RADIUS,
        sort_by: 'distance', pagination: { limit: 10 }
      };
      const response = await LoadingUtil.withMinimumTime(() => ApiClient.searchGroups(request));
      if (this.isDestroyed) return;
      if (response.code === 0) {
        const items = response.data?.items ?? [];
        this.groups = items.filter((g: GroupItem) => g.group?.group_id);
      } else {
        this.groupsError = response.message || '加载失败';
      }
    } catch {
      this.groupsError = '加载附近群组失败';
    } finally {
      this.groupsLoading = false;
    }
  }

  // ==================== 操作方法 ====================

  private isUserActionPending(userId: string): boolean {
    return this.userActionPendingIds.indexOf(userId) >= 0;
  }

  private setUserActionPending(userId: string, pending: boolean): void {
    const next: string[] = [];
    for (let i = 0; i < this.userActionPendingIds.length; i++) {
      if (this.userActionPendingIds[i] !== userId) next.push(this.userActionPendingIds[i]);
    }
    if (pending) next.push(userId);
    this.userActionPendingIds = next;
  }

  private isUserFollowing(userId: string): boolean {
    const state = this.userStateMap[userId];
    return state?.relation?.is_following ?? false;
  }

  private ensureUserRelation(user: UserItem): Relationship {
    if (!user.relation) {
      user.relation = { is_following: false, is_followed_by: false, is_blocked: false };
    }
    return user.relation;
  }

  private async followUser(user: UserItem): Promise<void> {
    if (!user.user?.user_id) return;
    const userId = user.user.user_id;
    if (this.isUserActionPending(userId)) return;
    this.setUserActionPending(userId, true);
    try {
      const isFollowing = this.isUserFollowing(userId);
      const success = await interactionActions.setFollowUser(userId, !isFollowing, '关注用户');
      if (success) {
        this.patchUserFollowing(userId, !isFollowing);
        SystemToast.showSuccess(isFollowing ? '已取消关注' : '已关注', this.getUIContext());
      } else {
        SystemToast.showError('操作失败', this.getUIContext());
      }
    } finally {
      this.setUserActionPending(userId, false);
    }
  }

  private patchUserFollowing(userId: string, isFollowing: boolean): void {
    const users = this.users;
    for (let i = 0; i < users.length; i++) {
      if ((users[i].user?.user_id ?? '') !== userId) continue;
      this.ensureUserRelation(users[i]).is_following = isFollowing;
      break;
    }
    this.users = users;
  }

  private async joinGroup(group: GroupItem): Promise<void> {
    if (!this.isLoggedIn) {
      SystemToast.showWarning('请先登录后再加入群组', this.getUIContext());
      return;
    }
    const groupId = group.group?.group_id ?? '';
    if (!groupId || group.my_state?.is_joined) {
      if (group.my_state?.is_joined) this.enterGroupChat(group);
      return;
    }
    try {
      const joinReq: JoinGroupRequest = { group_id: groupId };
      const response = await ApiClient.joinGroup(joinReq);
      if (response.code === 0 && response.data) {
        if (response.data.status === 'joined') {
          InteractionStateStore.getInstance().updateGroupMembership(groupId, { is_member: true });
          SystemToast.showSuccess('加入成功', this.getUIContext());
          this.loadNearbyGroups();
        } else if (response.data.status === 'pending_approval') {
          SystemToast.show('已提交申请，等待审批', this.getUIContext());
        }
      } else {
        SystemToast.showError(response.message || '加入失败', this.getUIContext());
      }
    } catch {
      SystemToast.showError('加入群组失败', this.getUIContext());
    }
  }

  private openUserProfile(user: UserItem): void {
    const userId = user.user?.user_id;
    if (!userId) return;
    const params = new ProfileRouteParam();
    params.userId = userId;
    AppRouter.push(PageNames.PROFILE, params);
  }

  private openMomentDetail(moment: MomentItem): void {
    const params = new MomentDetailRouteParam();
    params.momentId = moment.moment_id;
    AppRouter.push(PageNames.MOMENT_DETAIL, params);
  }

  private enterGroupChat(group: GroupItem): void {
    if (!this.isLoggedIn) return;
    const groupId = group.group?.group_id;
    if (!groupId) return;
    const params = new ChatRouteParam();
    params.chatId = groupId;
    params.chatType = 'group_chat';
    params.title = group.group?.name ?? '群聊';
    AppRouter.push(PageNames.CHAT, params);
  }

  private async copyText(text: string, successText: string): Promise<void> {
    if (!text) return;
    try {
      const systemPasteboard = pasteboard.getSystemPasteboard();
      const pasteData = pasteboard.createData(pasteboard.MIMETYPE_TEXT_PLAIN, text);
      await systemPasteboard.setData(pasteData);
      SystemToast.show(successText, this.getUIContext());
    } catch {
      SystemToast.showError('复制失败', this.getUIContext());
    }
  }

  // ==================== UI ====================

  build() {
  ResponsiveContainer() {
    Column() {
      // 粘性头部
      this.buildStickyHeader()

      if (isWideForm(this.currentBreakpoint)) {
        // ======== 宽屏（≥960）：双栏工作区（对齐前端首页 isWide 分支）========
        // 左栏：HeroStats 统计 + 附近群组横滑
        Row() {
          // 左栏
          Scroll() {
            Column({ space: DesignConstants.SPACING.LG }) {
              this.buildHeroStats()
              this.buildGroupCarousel()
            }
            .width('100%')
            .padding({ top: 16, bottom: 24 })
          }
          .layoutWeight(1)
          .scrollBar(BarState.Auto)
          .align(Alignment.TopStart)

          // 分割线
          Column()
            .width(1)
            .height('100%')
            .backgroundColor(this.getPalette().borderColor)

          // 右栏（固定宽度，对齐前端右栏语义）
          Scroll() {
            Column({ space: DesignConstants.SPACING.LG }) {
              this.buildNearbyPeople()
              this.buildMomentsSection()
            }
            .width('100%')
            .padding({ top: 16, bottom: 24 })
          }
          .width(420)
          .scrollBar(BarState.Auto)
          .align(Alignment.TopStart)
        }
        .layoutWeight(1)
      } else {
        // ======== 窄屏（<960）：单栏纵向堆叠 ========
        Scroll() {
          Column() {
            this.buildHeroStats()
            this.buildNearbyPeople()
            this.buildGroupCarousel()
            this.buildMomentsSection()
          }
          .width('100%')
          .padding({ top: 16, bottom: 24 })
        }
        .layoutWeight(1)
        .scrollBar(BarState.Auto)
        .align(Alignment.TopStart)
      }
    }
    .width('100%')
    .height('100%')
    .backgroundColor(this.getPalette().pageBackground)
  }
  }

  // ==================== 粘性头部 ====================

  @Builder
  private buildStickyHeader() {
    // 统一走 SystemPageHeader（标题 + 副标题 + 位置 chip trailing）
    SystemPageHeader({
      title: '首页',
      subtitle: '发现周边的有趣人和事',
      titleSize: DesignConstants.FONT.SIZE.XL_PLUS,
      titleWeight: FontWeight.Normal
    }) {
      // trailing: 位置 chip
      Row({ space: DesignConstants.SPACING.XS }) {
        Text() {
          SymbolSpan($r(AppIcon.LOCATION))
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .fontColor([this.getPalette().accentBackground])
        }

        Text(this.browseAnchor?.locationName ?? this.currentLocation?.locationName ?? '定位中...')
          .fontSize(DesignConstants.FONT.SIZE.XS)
          .fontColor(this.getPalette().textSecondary)
          .maxLines(1)
          .textOverflow({ overflow: TextOverflow.Ellipsis })
      }
      .padding({ left: DesignConstants.SPACING.MD, right: DesignConstants.SPACING.MD, top: DesignConstants.SPACING.XS, bottom: DesignConstants.SPACING.XS })
      .borderRadius(DesignConstants.RADIUS.FULL)
      .border({ width: 1, color: this.getPalette().borderColor })
      .backgroundColor(this.getPalette().surfaceBackground)
    }
  }

  // ==================== 地图卡片 ====================

  @Builder
  private buildHeroStats() {
    // 实时附近统计卡（对齐前端 HeroStats：radar 图标 + 标题/范围 + 三段统计）
    Column() {
      // Header row：图标 + 标题/范围 + 右箭头
      Row() {
        Row({ space: DesignConstants.SPACING.SM }) {
          // brand 圆形图标（对齐前端 38px brand 圆 + shadow-brand-btn）
          Row() {
            Text() {
              SymbolSpan($r(AppIcon.NEARBY))
                .fontSize(DesignConstants.FONT.SIZE.MD_PLUS)
                .fontColor([this.getPalette().accentText])
            }
          }
          .width(38)
          .height(38)
          .borderRadius(DesignConstants.RADIUS.FULL)
          .backgroundColor(this.getPalette().accentBackground)
          .justifyContent(FlexAlign.Center)

          Column() {
            Text('实时附近')
              .fontSize(DesignConstants.FONT.SIZE.SM_PLUS) // 13 对齐前端 text-body
              .fontWeight(FontWeight.Medium)
              .fontColor(this.getPalette().textPrimary)
            Text('持续更新')
              .fontSize(DesignConstants.FONT.SIZE.XS) // 11 对齐前端 text-caption
              .fontColor(this.getPalette().textSecondary)
              .margin({ top: 1 })
          }
          .alignItems(HorizontalAlign.Start)
        }
        .alignItems(VerticalAlign.Center)

        Blank()

        Text() {
          SymbolSpan($r(AppIcon.ARROW_RIGHT))
            .fontSize(DesignConstants.FONT.SIZE.MD)
            .fontColor([this.getPalette().textSecondary])
        }
      }
      .width('100%')
      .alignItems(VerticalAlign.Center)

      // Stats grid：三段统计 + 竖分割线
      Row({ space: DesignConstants.SPACING.MD }) {
        // 人附近
        Column({ space: 2 }) {
          Text(`${this.users.length}`)
            .fontSize(DesignConstants.FONT.SIZE.XXL)
            .fontWeight(FontWeight.Bold)
            .fontColor(this.getPalette().accentBackground)
          Text('人附近')
            .fontSize(DesignConstants.FONT.SIZE.XS)
            .fontColor(this.getPalette().textSecondary)
        }
        .alignItems(HorizontalAlign.Center)
        .layoutWeight(1)

        // 分割竖线
        Column()
          .width(1)
          .height(32)
          .backgroundColor(this.getPalette().borderColor)

        // 活动中
        Column({ space: 2 }) {
          Text(`${this.groups.length}`)
            .fontSize(DesignConstants.FONT.SIZE.XXL)
            .fontWeight(FontWeight.Bold)
            .fontColor(this.getPalette().warning)
          Text('活动中')
            .fontSize(DesignConstants.FONT.SIZE.XS)
            .fontColor(this.getPalette().textSecondary)
        }
        .alignItems(HorizontalAlign.Center)
        .layoutWeight(1)

        // 分割竖线
        Column()
          .width(1)
          .height(32)
          .backgroundColor(this.getPalette().borderColor)

        // 在线
        Column({ space: 2 }) {
          Text(`${this.getOnlineCount()}`)
            .fontSize(DesignConstants.FONT.SIZE.XXL)
            .fontWeight(FontWeight.Bold)
            .fontColor(this.getPalette().success)
          Text('在线')
            .fontSize(DesignConstants.FONT.SIZE.XS)
            .fontColor(this.getPalette().textSecondary)
        }
        .alignItems(HorizontalAlign.Center)
        .layoutWeight(1)
      }
      .width('100%')
      .margin({ top: DesignConstants.SPACING.LG })
    }
    .width('100%')
    .padding({ left: DesignConstants.SPACING.XL, right: DesignConstants.SPACING.XL, top: DesignConstants.SPACING.LG, bottom: DesignConstants.SPACING.LG })
    .borderRadius(DesignConstants.SHAPE.HERO_CARD)
    .backgroundColor(this.getPalette().surfaceBackground)
    .border({ width: 1, color: this.getPalette().borderColor })
    .onClick(() => AppRouter.push(PageNames.LOCATION_PICKER))
  }

  @Builder
  // ==================== 区块标题 (eyebrow) ====================

  @Builder
  private buildSectionHeader(title: string, linkText: string, onViewAll: () => void) {
    Row() {
      Text(title)
        .fontSize(DesignConstants.FONT.SIZE.XS2)
        .fontColor(this.getPalette().textSecondary)
        .textCase(TextCase.UpperCase)
        .letterSpacing(1.4)
        .opacity(0.75)

      Blank()

      Row({ space: 2 }) {
        Text(linkText)
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .fontColor(this.getPalette().accentBackground)
        Text() {
          SymbolSpan($r(AppIcon.ARROW_RIGHT))
            .fontSize(DesignConstants.FONT.SIZE.SM_PLUS)
            .fontColor([this.getPalette().accentBackground])
        }
      }
      .onClick(onViewAll)
    }
    .width('100%')
    .margin({ bottom: 12 })
  }

  // ==================== 附近的人 ====================

  @Builder
  private buildNearbyPeople() {
    Column() {
      this.buildSectionHeader('附近的人', '查看全部', () => {
        AppRouter.push(PageNames.NEARBY_USERS);
      })

      if (!this.usersLoading && this.usersError.length > 0) {
        Column({ space: DesignConstants.SPACING.SM }) {
          Text(this.usersError)
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .fontColor(this.getPalette().textSecondary)
            .width('100%')
            .textAlign(TextAlign.Center)
          Text('重试')
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .fontColor(this.getPalette().accentText)
            .padding({ left: DesignConstants.SPACING.MD, right: DesignConstants.SPACING.MD, top: 6, bottom: 6 })
            .borderRadius(DesignConstants.RADIUS.FULL)
            .border({ width: 1, color: this.getPalette().accentText })
            .onClick(() => this.loadNearbyUsers())
        }
        .width('100%')
        .alignItems(HorizontalAlign.Center)
        .margin({ top: DesignConstants.SPACING.SM })
      } else if (!this.usersLoading && this.users.length === 0) {
        Text('附近暂无其他用户，可稍后再来或调整浏览位置。')
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .fontColor(this.getPalette().textSecondary)
          .width('100%')
          .margin({ top: 2 })
      } else {
        Scroll() {
          Row({ space: 12 }) {
            if (this.usersLoading) {
              ForEach(['1', '2', '3', '4', '5'], (_: string) => {
              Column({ space: 6 }) {
                Row()
                  .width(66)
                  .height(66)
                  .borderRadius(DesignConstants.RADIUS.FULL)
                  .backgroundColor(this.getPalette().surfaceSubtle)
              }
              .alignItems(HorizontalAlign.Center)
            }, (_: string) => _)
          } else {
            ForEach(this.users.slice(0, 8), (user: UserItem) => {
              Column({ space: 6 }) {
                Stack({ alignContent: Alignment.Center }) {
                  SystemAvatar({
                    avatarUrl: user.user?.avatar_url ?? '',
                    name: user.user?.nickname ?? '',
                    avatarSize: 80,
                    shape: 'circle'
                  })
                  .border({ width: 2, color: this.getPalette().borderColor })
                  .borderRadius(DesignConstants.RADIUS.FULL)

                  // 在线状态点：贴头像右下角（用 alignContent 取代硬编码坐标）
                  if (user.presence_status === 'online') {
                    Stack({ alignContent: Alignment.BottomEnd }) {
                      Row()
                        .width(12)
                        .height(12)
                        .borderRadius(DesignConstants.RADIUS.FULL)
                        .backgroundColor(this.getPalette().success)
                        .border({ width: 2, color: this.getPalette().surfaceBackground })
                    }
                    .width('100%')
                    .height('100%')
                  }

                  // 距离徽标：贴头像底部居中（取代百分比 + translate 硬定位）
                  if (user.location?.distance !== undefined && user.location?.distance !== null) {
                    Stack({ alignContent: Alignment.Bottom }) {
                      Row() {
                        Text(LocationUtil.formatDistance(user.location.distance))
                          .fontSize(DesignConstants.FONT.SIZE.XS_MINUS)
                          .fontColor(this.getPalette().accentText)
                      }
                      .padding({ left: 6, right: 6, top: 1, bottom: 1 })
                      .borderRadius(DesignConstants.RADIUS.FULL)
                      .backgroundColor(this.getPalette().accentBackground)
                      .shadow(this.getPalette().accentShadow)
                    }
                    .width('100%')
                    .height('100%')
                  }
                }

                Text(user.user?.nickname ?? '')
                  .fontSize(DesignConstants.FONT.SIZE.SM)
                  .fontColor(this.getPalette().textPrimary)
                  .maxLines(1)
                  .textOverflow({ overflow: TextOverflow.Ellipsis })
                  .constraintSize({ maxWidth: 70 })
                  .margin({ top: 4 })
              }
              .alignItems(HorizontalAlign.Center)
              .onClick(() => {
                const userId = user.user?.user_id;
                if (userId) {
                  const params = new ProfileRouteParam();
                  params.userId = userId;
                  AppRouter.push(PageNames.PROFILE, params);
                }
              })
            }, (user: UserItem) => user.user?.user_id ?? '')
          }
        }
      }
      .scrollable(ScrollDirection.Horizontal)
      .scrollBar(BarState.Off)
      .width('100%')
      }
    }
    .width('100%')
    .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG })
    .margin({ bottom: 20 })
  }

  // ==================== 附近活动 ====================

  @Builder
  private buildGroupCarousel() {
    // 附近群组横滑条（对齐前端 GroupCarousel：水平滑动的群组卡片条）
    if (this.groupsLoading || this.groups.length > 0) {
      Column() {
        this.buildSectionHeader('附近群组', '查看全部', () => {
          const param: SearchRouteParam = { initialTab: 2 } as SearchRouteParam;
          AppRouter.push(PageNames.SEARCH, param);
        })

        Scroll() {
          Row({ space: DesignConstants.SPACING.SM }) {
            if (this.groupsLoading && this.groups.length === 0) {
              // loading skeleton
              Column()
                .width(140)
                .height(140)
                .borderRadius(DesignConstants.SHAPE.CARD)
                .backgroundColor(this.getPalette().surfaceSubtle)
              Column()
                .width(140)
                .height(140)
                .borderRadius(DesignConstants.SHAPE.CARD)
                .backgroundColor(this.getPalette().surfaceSubtle)
            } else {
              ForEach(this.groups, (group: GroupItem) => {
                this.buildGroupCarouselCard(group)
              }, (group: GroupItem) => group.group?.group_id ?? '')
            }
          }
          .width('100%')
          .padding({ left: DesignConstants.SPACING.XS })
        }
        .scrollBar(BarState.Auto)
        .scrollable(ScrollDirection.Horizontal)
        .width('100%')
      }
      .width('100%')
      .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG })
      .margin({ bottom: 20 })
    }
  }

  @Builder
  private buildGroupCarouselCard(group: GroupItem) {
    Column({ space: DesignConstants.SPACING.XS }) {
      Column() {
        Text() {
          SymbolSpan($r(AppIcon.GROUP))
            .fontSize(DesignConstants.FONT.SIZE.XXL)
            .fontColor([this.getPalette().accentBackground])
        }
      }
      .width(140)
      .height(100)
      .justifyContent(FlexAlign.Center)
      .borderRadius(DesignConstants.SHAPE.CARD)
      .backgroundColor(this.getPalette().accentMuted)
      .clip(true)

      Text(group.group?.name ?? '')
        .fontSize(DesignConstants.FONT.SIZE.XS)
        .fontColor(this.getPalette().textPrimary)
        .maxLines(1)
        .textOverflow({ overflow: TextOverflow.Ellipsis })
        .width(140)

      Row({ space: 4 }) {
        Text(`${formatCount(group.group?.members_count ?? 0)} 人`)
          .fontSize(DesignConstants.FONT.SIZE.XS2)
          .fontColor(this.getPalette().textTertiary)
        if (group.location?.distance !== undefined) {
          Text(LocationUtil.formatDistance(group.location.distance))
            .fontSize(DesignConstants.FONT.SIZE.XS2)
            .fontColor(this.getPalette().accentText)
        }
      }
      .width(140)
    }
    .alignItems(HorizontalAlign.Start)
    .borderRadius(DesignConstants.SHAPE.CARD)
    .onClick(() => {
      const groupId = group.group?.group_id;
      if (groupId) {
        const params = new GroupDetailRouteParam();
        params.groupId = groupId;
        AppRouter.push(PageNames.GROUP_DETAIL, params);
      }
    })
  }

  // ==================== 附近动态 ====================

  @Builder
  private buildMomentsSection() {
    if (this.momentsLoading || this.moments.length > 0) {
      Column() {
        this.buildSectionHeader('附近动态', '查看全部', () => {
          AppRouter.push(PageNames.MOMENT);
        })

        if (this.momentsLoading && this.moments.length === 0) {
          Column({ space: 12 }) {
            ForEach(['1', '2'], (_: string) => {
              Column()
                .width('100%')
                .height(96)
                .borderRadius(DesignConstants.SHAPE.CARD)
                .backgroundColor(this.getPalette().surfaceSubtle)
            }, (_: string) => _)
          }
          .width('100%')
        } else {
          Column({ space: 16 }) {
            ForEach(this.moments.slice(0, 5), (moment: MomentItem) => {
              MomentFeedCard({
                moment: moment,
                onOpenDetail: (item: MomentItem) => this.openMomentDetail(item)
              })
            }, (moment: MomentItem) => moment.moment_id)
          }
          .width('100%')
        }
      }
      .width('100%')
      .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG })
      .margin({ bottom: 24 })
    }
  }
}

class GroupDetailParams {
  groupId: string = '';
}
