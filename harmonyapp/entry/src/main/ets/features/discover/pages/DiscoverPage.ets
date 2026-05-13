/**
 * 发现页面
 * 展示热门话题、推荐用户、热门瞬间
 */

import { GlobalStateManager, KEY_CURRENT_LOCATION, KEY_BROWSE_ANCHOR, KEY_IS_LOGGED_IN } from '../../../core/state/GlobalStateManager';
import { LocationInfo } from '../../../core/location/LocationUtil';
import { ApiClient } from '../../../core/api/ApiClient';
import { AppRouter, ChatRouteParam, GroupDetailRouteParam, LoginRouteParam, MomentDetailRouteParam, ProfileRouteParam } from '../../../core/router/AppRouter';
import { PageNames, DefaultConfig } from '../../../constants/AppConstants';
import { interactionActions } from '../../../core/state/InteractionActions';
import { InteractionStateStore, UserRelationState } from '../../../core/state/InteractionStateStore';
import {
  UserItem, GroupItem, TrendingTagItem, MomentItem, TrendingBundle,
  SearchUsersRequest, SearchGroupsRequest, CreateReportRequest, Relationship
} from '../../../core/types/api';
import { LocationUtil } from '../../../core/location/LocationUtil';
import { SystemAvatar } from '../../../components/SystemAvatar';
import { SystemCard, CardVariant } from '../../../components/SystemCard';
import { SystemLoadingView } from '../../../components/SystemLoadingView';
import { SystemToast } from '../../../components/SystemToast';
import { DesignConstants } from '../../../constants/DesignConstants';
import { AppIcon } from '../../../components/SystemIcon';
import { MomentFeedCard } from '../../../components/MomentFeedCard';
import { LoadingUtil } from '../../../core/utils/LoadingUtil';
import { BasicDataSource } from '../../../core/utils/BasicDataSource';
import { SystemUiController } from '../../../core/utils/SystemUiController';
import { SafeAreaUtils } from '../../../core/utils/SafeAreaUtils';
import { EmptyState, RefreshIndicator } from '../../../components/EmptyState';
import { hilog } from '@kit.PerformanceAnalysisKit';
import pasteboard from '@ohos.pasteboard';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { KEY_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';

const DOMAIN_NUMBER: number = 0xFF11;
const TAG: string = 'DiscoverPage';

class SearchParams {
  keyword: string = '';
}


@Preview
@Component
export struct DiscoverPage {
  @StorageLink(KEY_IS_LOGGED_IN) isLoggedIn: boolean = false;
  @StorageLink(KEY_BROWSE_ANCHOR) browseAnchor: LocationInfo | null = null;
  @StorageLink(KEY_CURRENT_LOCATION) currentLocation: LocationInfo | null = null;
  @StorageLink('safeAreaTop') safeAreaTop: number = 0;
  @StorageLink('safeAreaBottom') safeAreaBottom: number = 0;
  @StorageLink(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'vibrant_light';
  @StorageLink('interaction_user_state') userStateMap: Record<string, UserRelationState> = {};
  @StorageProp('isTablet') isTablet: boolean = false;

  @State isLoading: boolean = true;
  @State errorMessage: string = '';
  @State trendingTags: TrendingTagItem[] = [];
  @State recommendedUsers: UserItem[] = [];
  private recommendedUsersDataSource: BasicDataSource<UserItem> = new BasicDataSource<UserItem>([]);
  @State recommendedGroups: GroupItem[] = [];
  @State hotMoments: MomentItem[] = [];
  private hotMomentsDataSource: BasicDataSource<MomentItem> = new BasicDataSource<MomentItem>([]);
  private interactionStore = InteractionStateStore.getInstance();
  private isDestroyed: boolean = false;
  @State isRefreshing: boolean = false;
  @State userActionPendingIds: string[] = [];
  @State activeTab: string = '热门';
  @State activeTag: string = '';

  private readonly TABS: string[] = ['热门', '附近', '活动', '关注'];

  private getFilteredMoments(): MomentItem[] {
    let filtered = this.hotMoments;
    if (this.activeTag) {
      filtered = filtered.filter((m: MomentItem) => (m.tags ?? []).indexOf(this.activeTag) >= 0);
    }
    return filtered;
  }

  private getAvailableTags(): string[] {
    const tagSet = new Set<string>();
    for (const moment of this.hotMoments) {
      for (const tag of (moment.tags ?? [])) {
        tagSet.add(tag);
      }
    }
    return Array.from(tagSet);
  }

  private getHeaderTopPadding(): number {
    return SafeAreaUtils.resolveTopPadding(this.safeAreaTop);
  }

  private isUserActionPending(userId: string): boolean {
    return this.userActionPendingIds.indexOf(userId) >= 0;
  }

  async aboutToAppear() {
    SystemUiController.syncWithCurrentTheme();
    await this.loadData();
  }

  aboutToDisappear(): void {
    this.isDestroyed = true;
  }

  private getCurrentThemePalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  private getSurfaceBorder(): string {
    return this.getCurrentThemePalette().borderColor;
  }

  private hasAnyData(): boolean {
    return this.trendingTags.length > 0 || this.recommendedUsers.length > 0 ||
      this.recommendedGroups.length > 0 || this.hotMoments.length > 0;
  }

  private async loadData(refresh: boolean = false): Promise<void> {
    if (!refresh) { this.isLoading = true; }
    this.errorMessage = '';
    try {
      // 热门内容不依赖位置，推荐用户需要位置
      const refLoc: LocationInfo | null = this.browseAnchor ?? this.currentLocation;
      const trendingPromise = ApiClient.getFeedTrending({});

      if (refLoc) {
        const usersRequest: SearchUsersRequest = {
          reference_point: { latitude: refLoc.latitude, longitude: refLoc.longitude },
          radius: DefaultConfig.NEARBY_RADIUS,
          pagination: { limit: 10 }
        };
        const groupsRequest: SearchGroupsRequest = {
          reference_point: { latitude: refLoc.latitude, longitude: refLoc.longitude },
          radius: DefaultConfig.NEARBY_RADIUS,
          sort_by: 'popular',
          pagination: { limit: 6 }
        };
        const results = await LoadingUtil.withMinimumTime(() =>
          Promise.all([trendingPromise, ApiClient.searchUsers(usersRequest), ApiClient.searchGroups(groupsRequest)])
        );
        if (this.isDestroyed) return;
        const usersResponse = results[1];
        if (usersResponse.code === 0) {
          const items = usersResponse.data?.items ?? [];
          const me = GlobalStateManager.getInstance().getCurrentUser();
          const myId = me?.user_id;
          const validUsersAll = items.filter((u: UserItem) => u.user?.user_id);
          this.recommendedUsers = typeof myId === 'string'
            ? validUsersAll.filter((u: UserItem) => u.user.user_id !== myId)
            : validUsersAll;
          this.recommendedUsersDataSource.setData(this.recommendedUsers);
        }
        const groupsResponse = results[2];
        if (groupsResponse.code === 0) {
          this.recommendedGroups = groupsResponse.data?.items ?? [];
        }
        const trendingResponse = results[0];
        if (trendingResponse.code === 0) {
          const bundle = trendingResponse.data;
          this.trendingTags = bundle?.trending_tags ?? [];
          this.hotMoments = bundle?.hot_moments ?? [];
          this.hotMomentsDataSource.setData(this.hotMoments);
          this.hotMoments.forEach((item: MomentItem) => this.interactionStore.patchMomentFromItem(item));
        }
        if (usersResponse.code !== 0 && trendingResponse.code !== 0 && !this.hasAnyData()) {
          this.errorMessage = trendingResponse.message ?? usersResponse.message ?? '加载发现数据失败';
        }
      } else {
        const trendingResponse = await LoadingUtil.withMinimumTime(() => trendingPromise);
        if (this.isDestroyed) return;
        if (trendingResponse.code === 0) {
          const bundle = trendingResponse.data;
          this.trendingTags = bundle?.trending_tags ?? [];
          this.hotMoments = bundle?.hot_moments ?? [];
          this.hotMomentsDataSource.setData(this.hotMoments);
          this.hotMoments.forEach((item: MomentItem) => this.interactionStore.patchMomentFromItem(item));
        } else if (!this.hasAnyData()) {
          this.errorMessage = trendingResponse.message ?? '加载发现数据失败';
        }
      }
    } catch {
      hilog.error(DOMAIN_NUMBER, TAG, '加载发现数据失败');
      if (!this.hasAnyData()) {
        this.errorMessage = '加载发现数据失败';
      }
    } finally {
      this.isLoading = false;
      this.isRefreshing = false;
    }
  }

  build() {
    Column() {
      // 粘性头部：标题 + 搜索栏 + 标签栏
      Column() {
        Row() {
          Text('发现')
            .fontSize(22)
            .fontWeight(FontWeight.Bold)
            .fontColor(this.getCurrentThemePalette().textPrimary)

          Blank()

          Text(`${this.getFilteredMoments().length} 条内容`)
            .fontSize(12)
            .fontColor(this.getCurrentThemePalette().textSecondary)
        }
        .width('100%')
        .margin({ bottom: 10 })

        // 搜索栏
        Row({ space: 10 }) {
          Text() {
            SymbolSpan($r(AppIcon.SEARCH))
              .fontSize(15)
              .fontColor([this.getCurrentThemePalette().textSecondary])
          }
          Text('地点、人物、关键词…')
            .fontSize(13)
            .fontColor(this.getCurrentThemePalette().textSecondary)
            .layoutWeight(1)
            .maxLines(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
        }
        .height(42)
        .padding({ left: 16, right: 16 })
        .backgroundColor(this.getCurrentThemePalette().inputBackgroundColor)
        .borderRadius(18)
        .shadow(this.getCurrentThemePalette().controlShadow)
        .border({ width: 1, color: this.getSurfaceBorder() })
        .onClick(() => {
          AppRouter.push(PageNames.SEARCH);
        })
        .margin({ bottom: 10 })

        // 标签栏
        this.buildTabBar()
      }
      .width('100%')
      .padding({
        left: DesignConstants.SPACING.LG,
        right: DesignConstants.SPACING.LG,
        top: this.getHeaderTopPadding(),
        bottom: 0
      })
      .backgroundColor(this.getCurrentThemePalette().pageBackground)
      .border({ width: { bottom: 1 }, color: this.getSurfaceBorder() })
      .expandSafeArea([SafeAreaType.SYSTEM], [SafeAreaEdge.TOP])

      // 内容区
      if (this.isLoading) {
        SystemLoadingView({ message: '正在加载发现内容...' })
          .layoutWeight(1)
          .transition(
            TransitionEffect.OPACITY.animation({ duration: 320, curve: Curve.FastOutSlowIn })
              .combine(TransitionEffect.move(TransitionEdge.BOTTOM).animation({ duration: 320, curve: Curve.FastOutSlowIn }))
          )
      } else if (!this.hasAnyData() && this.errorMessage) {
        EmptyState({
          icon: AppIcon.WARNING,
          message: this.errorMessage,
          actionLabel: '重试',
          onAction: () => { this.loadData(); }
        })
        .layoutWeight(1)
        .transition(
          TransitionEffect.OPACITY.animation({ duration: 320, curve: Curve.FastOutSlowIn })
            .combine(TransitionEffect.move(TransitionEdge.BOTTOM).animation({ duration: 320, curve: Curve.FastOutSlowIn }))
        )
      } else {
        Refresh({ refreshing: $$this.isRefreshing, builder: RefreshIndicator }) {
          List() {
            // 标签药片 — 热门话题
            if (this.getAvailableTags().length > 0) {
              ListItem() {
                this.buildSectionHeader('热门话题')
              }
              ListItem() {
                this.buildTagChipsStrip()
              }
            }

            // 推荐用户
            if (this.recommendedUsers.length > 0) {
              ListItem() {
                this.buildSectionHeader('推荐用户')
              }
              ListItem() {
                this.buildRecommendedUsersSection()
              }
            }

            // 推荐群组
            if (this.recommendedGroups.length > 0) {
              ListItem() {
                this.buildSectionHeader('推荐群组')
              }
              ListItem() {
                this.buildRecommendedGroupsSection()
              }
            }

            // 动态流标题
            ListItem() {
              this.buildSectionHeader('热门动态')
            }

            // 动态卡片列表（PostCard 风格）
            LazyForEach(this.hotMomentsDataSource, (moment: MomentItem) => {
              ListItem() {
                Column() {
                  MomentFeedCard({
                    moment: moment,
                    variant: 'postCard',
                    showFavorite: false,
                    onOpenDetail: (item: MomentItem) => this.openMomentDetail(item)
                  })
                }
                .width('100%')
                .padding({
                  left: DesignConstants.SPACING.LG,
                  right: DesignConstants.SPACING.LG,
                  top: 4,
                  bottom: 4
                })
              }
              .width('100%')
            }, (moment: MomentItem) => moment.moment_id)

            // 没有任何内容
            if (this.hotMoments.length === 0) {
              ListItem() {
                EmptyState({ icon: AppIcon.RECOMMEND, message: '暂无推荐内容' })
                  .height(300)
              }
            }
          }
          .width('100%')
          .lanes(this.isTablet ? 2 : 1, 12)
          .padding({
            bottom: SafeAreaUtils.resolveBottomPadding(this.safeAreaBottom, DesignConstants.SPACING.LG, 0)
          })
          .scrollBar(BarState.Auto)
        }
        .layoutWeight(1)
        .onRefreshing(() => { this.loadData(true); })
        .transition(
          TransitionEffect.OPACITY.animation({ duration: 320, curve: Curve.FastOutSlowIn })
            .combine(TransitionEffect.move(TransitionEdge.BOTTOM).animation({ duration: 320, curve: Curve.FastOutSlowIn }))
        )
      }
    }
    .width('100%')
    .height('100%')
    .constraintSize({ maxWidth: this.isTablet ? 720 : Infinity })
    .backgroundColor(this.getCurrentThemePalette().pageBackground)
  }

  // ==================== 标签栏 ====================
  @Builder
  buildTabBar() {
    Row({ space: 0 }) {
      ForEach(this.TABS, (tab: string) => {
        Column({ space: 8 }) {
          Text(tab)
            .fontSize(13)
            .fontWeight(this.activeTab === tab ? FontWeight.Medium : FontWeight.Normal)
            .fontColor(this.activeTab === tab
              ? this.getCurrentThemePalette().accentBackground
              : this.getCurrentThemePalette().textSecondary)

          if (this.activeTab === tab) {
            Divider()
              .strokeWidth(2)
              .width('72%')
              .color(this.getCurrentThemePalette().accentBackground)
              .borderRadius(1)
          } else {
            Divider()
              .strokeWidth(0)
              .height(2)
              .width('72%')
          }
        }
        .padding({ left: 18, right: 18, top: 4, bottom: 12 })
        .onClick(() => {
          this.activeTab = tab;
        })
      }, (tab: string) => tab)
    }
    .width('100%')
  }

  // ==================== 标签药片条 ====================
  @Builder
  buildTagChipsStrip() {
    Scroll() {
      Row({ space: 8 }) {
        ForEach(this.getAvailableTags(), (tag: string) => {
          Text(tag)
            .fontSize(12)
            .fontColor(this.activeTag === tag
              ? Color.White
              : this.getCurrentThemePalette().textSecondary)
            .backgroundColor(this.activeTag === tag
              ? this.getCurrentThemePalette().accentBackground
              : this.getCurrentThemePalette().surfaceBackground)
            .borderRadius(DesignConstants.RADIUS.FULL)
            .padding({
              left: 14,
              right: 14,
              top: 6,
              bottom: 6
            })
            .shadow(this.activeTag === tag
              ? {
                radius: 10,
                color: this.getCurrentThemePalette().accentShadow.color,
                offsetX: 0,
                offsetY: 3
              }
              : this.getCurrentThemePalette().controlShadow)
            .border({
              width: 1,
              color: this.activeTag === tag
                ? Color.Transparent
                : this.getSurfaceBorder()
            })
            .onClick(() => {
              this.activeTag = this.activeTag === tag ? '' : tag;
            })
        }, (tag: string) => tag)
      }
      .padding({ left: 4, right: 4 })
    }
    .scrollable(ScrollDirection.Horizontal)
    .scrollBar(BarState.Off)
    .width('100%')
    .padding({
      left: DesignConstants.SPACING.LG,
      right: DesignConstants.SPACING.LG,
      top: 4,
      bottom: 4
    })
  }

  // ==================== section 标题 ====================
  @Builder
  buildSectionHeader(title: string) {
    Row() {
      Text(title)
        .fontSize(16)
        .fontWeight(FontWeight.Bold)
        .fontColor(this.getCurrentThemePalette().textPrimary)
    }
    .width('100%')
    .padding({
      left: DesignConstants.SPACING.LG,
      right: DesignConstants.SPACING.LG,
      top: 16,
      bottom: 4
    })
  }

  // ==================== 推荐用户横向滚动 ====================
  @Builder
  buildRecommendedUsersSection() {
    Column() {
      SystemCard({
        cardPadding: DesignConstants.SPACING.MD,
        variant: CardVariant.OUTLINED
      }) {
        Scroll() {
          Row({ space: 14 }) {
            ForEach(this.recommendedUsers.slice(0, 10), (user: UserItem) => {
              this.buildUserCard(user)
            }, (user: UserItem) => user.user?.user_id ?? '')
          }
        }
        .scrollable(ScrollDirection.Horizontal)
        .scrollBar(BarState.Off)
      }
    }
    .margin({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG })
  }

  @Builder
  buildUserCard(user: UserItem) {
    Column({ space: 6 }) {
      Stack() {
        SystemAvatar({
          avatarUrl: user.user?.avatar_url ?? '',
          name: user.user?.nickname ?? '',
          avatarSize: 56,
          shape: 'squircle'
        })

        if (user.location?.distance !== undefined && user.location?.distance !== null) {
          Row() {
            Text(LocationUtil.formatDistance(user.location.distance))
              .fontSize(9)
              .fontColor(Color.White)
          }
          .padding({ left: 6, right: 6, top: 2, bottom: 2 })
          .borderRadius(DesignConstants.RADIUS.FULL)
          .backgroundColor(this.getCurrentThemePalette().accentBackground)
          .position({ x: 6, y: 50 })
        }
      }
      .width(60)
      .height(64)

      Text(user.user?.nickname ?? '')
        .fontSize(11)
        .fontColor(this.getCurrentThemePalette().textPrimary)
        .maxLines(1)
        .textOverflow({ overflow: TextOverflow.Ellipsis })
        .constraintSize({ maxWidth: 60 })
    }
    .alignItems(HorizontalAlign.Center)
    .onClick(() => { this.openUserProfile(user); })
  }

  // ==================== 推荐群组网格 ====================
  @Builder
  buildRecommendedGroupsSection() {
    Column() {
      SystemCard({
        cardPadding: DesignConstants.SPACING.MD,
        variant: CardVariant.OUTLINED
      }) {
        Flex({ wrap: FlexWrap.Wrap, justifyContent: FlexAlign.Start }) {
          ForEach(this.recommendedGroups.slice(0, 4), (group: GroupItem) => {
            this.buildGroupCard(group)
          }, (group: GroupItem) => group.group?.group_id ?? '')
        }
        .width('100%')
      }
    }
    .margin({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG })
  }

  @Builder
  buildGroupCard(group: GroupItem) {
    Column() {
      Row() {
        Text() {
          SymbolSpan($r(AppIcon.GROUP))
            .fontSize(24)
            .fontColor([this.getCurrentThemePalette().accentBackground])
        }
      }
      .width('100%')
      .height(48)
      .justifyContent(FlexAlign.Center)
      .backgroundColor(this.getCurrentThemePalette().accentMuted)
      .borderRadius({ topLeft: 16, topRight: 16 })

      Column({ space: 4 }) {
        Text(group.group?.name ?? '')
          .fontSize(12)
          .fontWeight(FontWeight.Medium)
          .fontColor(this.getCurrentThemePalette().textPrimary)
          .maxLines(1)
          .textOverflow({ overflow: TextOverflow.Ellipsis })

        Row({ space: 6 }) {
          Text(`${group.group?.members_count ?? 0} 人`)
            .fontSize(10)
            .fontColor(this.getCurrentThemePalette().textSecondary)
          if (group.location?.distance !== undefined) {
            Text(LocationUtil.formatDistance(group.location.distance))
              .fontSize(10)
              .fontColor(this.getCurrentThemePalette().accentBackground)
          }
        }
      }
      .padding({ left: 10, right: 10, top: 8, bottom: 10 })
      .alignItems(HorizontalAlign.Start)
    }
    .width('calc(50% - 6px)')
    .borderRadius(16)
    .backgroundColor(this.getCurrentThemePalette().surfaceBackground)
    .border({ width: 1, color: this.getCurrentThemePalette().borderColor })
    .shadow(this.getCurrentThemePalette().controlShadow)
    .margin({ bottom: 8 })
    .onClick(() => { this.openGroupDetail(group); })
  }

  // ==================== 操作 ====================
  private async followUser(user: UserItem): Promise<void> {
    if (!user.user?.user_id) return;
    const userId = user.user.user_id;
    if (this.isUserActionPending(userId)) return;
    this.setUserActionPending(userId, true);
    try {
      const isFollowing = this.isUserFollowing(userId);
      const success = await interactionActions.setFollowUser(userId, !isFollowing, '关注用户');
      if (success) {
        this.patchRecommendedUserFollowing(userId, !isFollowing);
        SystemToast.showSuccess(isFollowing ? '已取消关注' : '已关注', this.getUIContext());
      } else {
        SystemToast.showError('操作失败', this.getUIContext());
      }
    } finally {
      this.setUserActionPending(userId, false);
    }
  }

  private isUserFollowing(userId: string): boolean {
    const state = this.userStateMap[userId];
    if (state?.relation) {
      return state.relation.is_following;
    }
    return false;
  }

  private ensureUserRelation(user: UserItem): Relationship {
    if (!user.relation) {
      const relation: Relationship = {
        is_following: false,
        is_followed_by: false,
        is_blocked: false
      };
      user.relation = relation;
    }
    return user.relation;
  }

  private patchRecommendedUserFollowing(userId: string, isFollowing: boolean): void {
    const users = this.recommendedUsers;
    for (let i = 0; i < users.length; i++) {
      if ((users[i].user?.user_id ?? '') !== userId) {
        continue;
      }
      const relation = this.ensureUserRelation(users[i]);
      relation.is_following = isFollowing;
      break;
    }
    this.recommendedUsers = users;
    this.recommendedUsersDataSource.setData(users);
  }

  private setUserActionPending(userId: string, pending: boolean): void {
    const next: string[] = [];
    for (let i = 0; i < this.userActionPendingIds.length; i++) {
      const id = this.userActionPendingIds[i];
      if (id !== userId) {
        next.push(id);
      }
    }
    if (pending) {
      next.push(userId);
    }
    this.userActionPendingIds = next;
  }

  private openMomentDetail(moment: MomentItem): void {
    if (!moment.moment_id) return;
    const params = new MomentDetailRouteParam();
    params.momentId = moment.moment_id;
    AppRouter.push(PageNames.MOMENT_DETAIL, params);
  }

  private openUserProfile(user: UserItem): void {
    const userId = user.user?.user_id;
    if (!userId) {
      return;
    }
    const params = new ProfileRouteParam();
    params.userId = userId;
    AppRouter.push(PageNames.PROFILE, params);
  }

  private openGroupDetail(group: GroupItem): void {
    const groupId = group.group?.group_id;
    if (!groupId) return;
    const params = new GroupDetailRouteParam();
    params.groupId = groupId;
    AppRouter.push(PageNames.GROUP_DETAIL, params);
  }

  private startPrivateChat(user: UserItem): void {
    if (!this.isLoggedIn) {
      SystemToast.showWarning('请先登录后再发起私聊', this.getUIContext());
      const loginParams = new LoginRouteParam();
      loginParams.reason = 'action_required';
      loginParams.action = '发起私聊';
      AppRouter.push(PageNames.LOGIN, loginParams);
      return;
    }
    const userId = user.user?.user_id;
    if (!userId) return;
    const params = new ChatRouteParam();
    params.chatId = userId;
    params.chatType = 'private_chat';
    params.targetId = userId;
    params.title = user.user?.nickname ?? '私聊';
    AppRouter.push(PageNames.CHAT, params);
  }

  private async reportUser(user: UserItem): Promise<void> {
    if (!this.isLoggedIn) {
      SystemToast.showWarning('请先登录后再举报', this.getUIContext());
      const loginParams = new LoginRouteParam();
      loginParams.reason = 'action_required';
      loginParams.action = '举报用户';
      AppRouter.push(PageNames.LOGIN, loginParams);
      return;
    }
    const userId = user.user?.user_id;
    if (!userId) return;
    try {
      const req: CreateReportRequest = {
        target_type: 'user',
        target_id: userId,
        reason: 'other'
      };
      const response = await ApiClient.createReport(req);
      if (response.code === 0) {
        SystemToast.showSuccess('举报已提交', this.getUIContext());
      } else {
        SystemToast.showError(response.message || '举报失败', this.getUIContext());
      }
    } catch {
      SystemToast.showError('举报失败', this.getUIContext());
    }
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


}
