/**
 * 附近页面 - Figma Make HomeMobile 对齐版
 * 粘性头部 + 地图卡片 + 附近用户列表 + 活动网格
 */

import { GlobalStateManager, KEY_IS_LOGGED_IN, KEY_BROWSE_ANCHOR, KEY_CURRENT_LOCATION } from '../../../core/state/GlobalStateManager';
import { LocationInfo, LocationUtil } from '../../../core/location/LocationUtil';
import { ApiClient } from '../../../core/api/ApiClient';
import { AppRouter, ChatRouteParam, LoginRouteParam, MomentDetailRouteParam, ProfileRouteParam } from '../../../core/router/AppRouter';
import { PageNames, DefaultConfig } from '../../../constants/AppConstants';
import {
  MomentItem, UserItem, GroupItem, GeoPoint,
  SearchMomentsRequest, SearchUsersRequest, SearchGroupsRequest,
  JoinGroupRequest,
  BlockRequest, UnblockRequest, CreateReportRequest, Relationship
} from '../../../core/types/api';
import { SystemAvatar } from '../../../components/SystemAvatar';
import { MomentFeedCard } from '../../../components/MomentFeedCard';
import { SystemLoadingView } from '../../../components/SystemLoadingView';
import { SystemToast } from '../../../components/SystemToast';
import { InteractionStateStore, UserRelationState } from '../../../core/state/InteractionStateStore';
import { interactionActions } from '../../../core/state/InteractionActions';
import { EmptyState } from '../../../components/EmptyState';
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
  @StorageLink(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'figma_neu_light';
  @StorageLink('interaction_user_state') userStateMap: Record<string, UserRelationState> = {};
  @StorageProp('isTablet') isTablet: boolean = false;

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
    Column() {
      // 粘性头部
      this.buildStickyHeader()

      // 内容区
      Scroll() {
        Column() {
          this.buildMapCard()
          this.buildNearbyPeople()
          this.buildActivityGrid()
          this.buildMomentsSection()

          if (this.users.length === 0 && this.groups.length === 0 && this.moments.length === 0) {
            if (this.momentsLoading || this.usersLoading || this.groupsLoading) {
              SystemLoadingView({ message: '正在加载附近内容...' })
                .height(400)
            } else {
              EmptyState({
                icon: AppIcon.NEARBY,
                message: '附近暂无内容\n发布第一条动态吧！',
                actionLabel: '重试',
                onAction: () => { this.loadNearbyMoments(); this.loadNearbyUsers(); this.loadNearbyGroups(); }
              })
              .height(300)
            }
          }
        }
        .width('100%')
        .constraintSize({ maxWidth: this.isTablet ? 680 : Infinity })
        .padding({ top: 16, bottom: 24 })
      }
      .layoutWeight(1)
      .scrollBar(BarState.Auto)
      .align(Alignment.TopStart)
    }
    .width('100%')
    .height('100%')
    .constraintSize({ maxWidth: this.isTablet ? 720 : Infinity })
    .backgroundColor(this.getPalette().pageBackground)
  }

  // ==================== 粘性头部 ====================

  @Builder
  private buildStickyHeader() {
    Row() {
      Column({ space: 2 }) {
        Text('附近')
          .fontSize(22)
          .fontWeight(FontWeight.Bold)
          .fontColor(this.getPalette().textPrimary)
        Text('发现周边的有趣人和事')
          .fontSize(12)
          .fontColor(this.getPalette().textTertiary)
      }
      .alignItems(HorizontalAlign.Start)

      Blank()

      Row({ space: 6 }) {
        Text() {
          SymbolSpan($r(AppIcon.LOCATION))
            .fontSize(12)
            .fontColor([this.getPalette().accentBackground])
        }

        Text(this.browseAnchor?.locationName ?? this.currentLocation?.locationName ?? '定位中...')
          .fontSize(11)
          .fontColor(this.getPalette().textTertiary)
          .maxLines(1)
          .textOverflow({ overflow: TextOverflow.Ellipsis })
      }
      .padding({ left: 12, right: 14, top: 6, bottom: 6 })
      .borderRadius(DesignConstants.RADIUS.FULL)
      .shadow(this.getPalette().controlShadow)
      .border({ width: 1, color: this.getPalette().borderColor })
      .backgroundColor(this.getPalette().surfaceBackground)
    }
    .width('100%')
    .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, top: this.getHeaderTopPadding(), bottom: 12 })
    .backgroundColor(this.getPalette().pageBackground)
    .border({ width: { bottom: 1 }, color: this.getPalette().borderColor })
  }

  // ==================== 地图卡片 ====================

  @Builder
  private buildMapCard() {
    Stack() {
      Column()
        .width('100%')
        .aspectRatio(2.4)
        .borderRadius(24)
        .linearGradient({
          direction: GradientDirection.Bottom,
          colors: [
            ['rgba(0,0,0,0.0)', 0.3],
            ['rgba(0,0,0,0.3)', 1.0]
          ]
        })
        .backgroundColor(this.getPalette().surfaceSubtle)

      // 脉冲定位点
      Row()
        .width(14)
        .height(14)
        .borderRadius(7)
        .backgroundColor(this.getPalette().accentBackground)
        .shadow({ radius: 4, color: this.getPalette().accentShadow.color, offsetX: 0, offsetY: 0 })
        .position({ x: '50%', y: '50%' })
        .translate({ x: -7, y: -7 })

      // 右下控制按钮
      Column({ space: 8 }) {
        Button() {
          Text() {
            SymbolSpan($r(AppIcon.LOCATION))
              .fontSize(14)
              .fontColor([this.getPalette().textPrimary])
          }
        }
        .width(34)
        .height(34)
        .type(ButtonType.Normal)
        .borderRadius(12)
        .backgroundColor(this.getPalette().surfaceBackground)
        .opacity(0.88)
        .border({ width: 1, color: 'rgba(255,255,255,0.25)' })
      }
      .position({ x: 'calc(100% - 42vp)', y: 'calc(100% - 46vp)' })

      // 左下人数统计
      Row({ space: 4 }) {
        Text(`${this.users.length}`)
          .fontSize(11)
          .fontColor(this.getPalette().accentBackground)
          .fontWeight(FontWeight.Bold)
        Text('人附近')
          .fontSize(10)
          .fontColor(this.getPalette().textTertiary)
      }
      .padding({ left: 10, right: 10, top: 4, bottom: 4 })
      .borderRadius(DesignConstants.RADIUS.FULL)
      .backgroundColor(this.getPalette().surfaceBackground)
      .opacity(0.88)
      .border({ width: 1, color: 'rgba(255,255,255,0.2)' })
      .position({ x: 12, y: 'calc(100% - 34vp)' })
    }
    .width('100%')
    .borderRadius(24)
    .margin({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, bottom: 20 })
    .shadow(this.getPalette().elevatedShadow)
    .border({ width: 1, color: this.getPalette().borderColor })
  }

  // ==================== 附近的人 ====================

  @Builder
  private buildNearbyPeople() {
    Column() {
      Row() {
        Text('附近的人')
          .fontSize(16)
          .fontWeight(FontWeight.Medium)
          .fontColor(this.getPalette().textPrimary)

        Blank()

        Row({ space: 2 }) {
          Text('全部')
            .fontSize(12)
            .fontColor(this.getPalette().accentBackground)
          Text() {
            SymbolSpan($r(AppIcon.ARROW_RIGHT))
              .fontSize(11)
              .fontColor([this.getPalette().accentBackground])
          }
        }
        .onClick(() => {
          // navigate to nearby users
        })
      }
      .width('100%')
      .margin({ bottom: 12 })

      Scroll() {
        Row({ space: 12 }) {
          if (this.usersLoading) {
            ForEach(['1', '2', '3', '4', '5'], (_: string) => {
              Column({ space: 6 }) {
                Row()
                  .width(66)
                  .height(66)
                  .borderRadius(20)
                  .backgroundColor(this.getPalette().surfaceSubtle)
              }
              .alignItems(HorizontalAlign.Center)
            }, (_: string) => _)
          } else {
            ForEach(this.users.slice(0, 8), (user: UserItem) => {
              Column({ space: 6 }) {
                Stack() {
                  SystemAvatar({
                    avatarUrl: user.user?.avatar_url ?? '',
                    name: user.user?.nickname ?? '',
                    avatarSize: 66,
                    shape: 'circle'
                  })
                  .border({ width: 2, color: this.getPalette().borderColor })
                  .borderRadius(33)

                  if (user.presence_status === 'online') {
                    Row()
                      .width(12)
                      .height(12)
                      .borderRadius(6)
                      .backgroundColor(DesignConstants.COLOR.SUCCESS)
                      .border({ width: 2, color: this.getPalette().surfaceBackground })
                      .position({ x: 50, y: 50 })
                  }

                  if (user.location?.distance !== undefined && user.location?.distance !== null) {
                    Row() {
                      Text(LocationUtil.formatDistance(user.location.distance))
                        .fontSize(9)
                        .fontColor(Color.White)
                    }
                    .padding({ left: 6, right: 6, top: 1, bottom: 1 })
                    .borderRadius(DesignConstants.RADIUS.FULL)
                    .backgroundColor(this.getPalette().accentBackground)
                    .shadow({
                      radius: 8,
                      color: this.getPalette().accentShadow.color,
                      offsetX: 0,
                      offsetY: 2
                    })
                    .position({ x: '50%', y: 62 })
                    .translate({ x: '-50%' })
                  }
                }

                Text(user.user?.nickname ?? '')
                  .fontSize(12)
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
    .width('100%')
    .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG })
    .margin({ bottom: 20 })
  }

  // ==================== 附近活动 ====================

  @Builder
  private buildActivityGrid() {
    Column() {
      Row() {
        Text('附近活动')
          .fontSize(16)
          .fontWeight(FontWeight.Medium)
          .fontColor(this.getPalette().textPrimary)

        Blank()

        Row({ space: 2 }) {
          Text('更多')
            .fontSize(12)
            .fontColor(this.getPalette().accentBackground)
          Text() {
            SymbolSpan($r(AppIcon.ARROW_RIGHT))
              .fontSize(11)
              .fontColor([this.getPalette().accentBackground])
          }
        }
      }
      .width('100%')
      .margin({ bottom: 12 })

      Flex({ wrap: FlexWrap.Wrap }) {
        ForEach(this.groups.slice(0, 4), (group: GroupItem) => {
          Column() {
            this.buildActivityCard(group)
          }
          .margin({ left: 6, right: 6, top: 6, bottom: 6 })
        }, (group: GroupItem) => group.group?.group_id ?? '')
      }
      .width('100%')
    }
    .width('100%')
    .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG })
    .margin({ bottom: 20 })
  }

  @Builder
  private buildActivityCard(group: GroupItem) {
    Column() {
      Row() {
        Text() {
          SymbolSpan($r(AppIcon.GROUP))
            .fontSize(28)
            .fontColor([this.getPalette().accentBackground])
        }
      }
      .width('100%')
      .height(64)
      .justifyContent(FlexAlign.Center)
      .backgroundColor(this.getPalette().accentMuted)
      .borderRadius({ topLeft: 22, topRight: 22 })

      Column({ space: 4 }) {
        Text(group.group?.name ?? '')
          .fontSize(13)
          .fontWeight(FontWeight.Medium)
          .fontColor(this.getPalette().textPrimary)
          .maxLines(1)
          .textOverflow({ overflow: TextOverflow.Ellipsis })

        Row({ space: 6 }) {
          Text(`${formatCount(group.group?.members_count)} 人`)
            .fontSize(11)
            .fontColor(this.getPalette().textTertiary)
          if (group.location?.distance !== undefined) {
            Text(LocationUtil.formatDistance(group.location.distance))
              .fontSize(11)
              .fontColor(this.getPalette().accentBackground)
          }
        }
      }
      .padding({ left: 12, right: 12, top: 12, bottom: 12 })
      .alignItems(HorizontalAlign.Start)
    }
    .width('calc(50% - 12vp)')
    .borderRadius(22)
    .backgroundColor(this.getPalette().surfaceBackground)
    .border({ width: 1, color: this.getPalette().borderColor })
    .shadow(this.getPalette().elevatedShadow)
    .onClick(() => {
      if (group.group?.group_id) {
        const params = new GroupDetailParams();
        params.groupId = group.group.group_id;
        AppRouter.push(PageNames.GROUP_DETAIL, params);
      }
    })
  }

  // ==================== 附近动态 ====================

  @Builder
  private buildMomentsSection() {
    Column() {
      Row() {
        Text('附近动态')
          .fontSize(10)
          .fontColor(this.getPalette().textTertiary)
          .textCase(TextCase.UpperCase)
          .letterSpacing(0.14 * 16)

        Blank()

        Row({ space: 2 }) {
          Text('更多')
            .fontSize(12)
            .fontColor(this.getPalette().accentBackground)
          Text() {
            SymbolSpan($r(AppIcon.ARROW_RIGHT))
              .fontSize(11)
              .fontColor([this.getPalette().accentBackground])
          }
        }
      }
      .width('100%')
      .margin({ bottom: 12 })

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
    .width('100%')
    .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG })
    .margin({ bottom: 24 })
  }
}

class GroupDetailParams {
  groupId: string = '';
}
