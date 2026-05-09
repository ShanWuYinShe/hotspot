/**
 * 发现页面
 * 展示热门话题、推荐用户、热门瞬间
 */

import { GlobalStateManager, KEY_CURRENT_LOCATION, KEY_BROWSE_ANCHOR, KEY_IS_LOGGED_IN } from '../../../core/state/GlobalStateManager';
import { LocationInfo } from '../../../core/location/LocationUtil';
import { ApiClient } from '../../../core/api/ApiClient';
import { AppRouter, ChatRouteParam, LoginRouteParam, MomentDetailRouteParam, ProfileRouteParam } from '../../../core/router/AppRouter';
import { PageNames, DefaultConfig } from '../../../constants/AppConstants';
import { interactionActions } from '../../../core/state/InteractionActions';
import { InteractionStateStore, UserRelationState } from '../../../core/state/InteractionStateStore';
import {
  UserItem, TrendingTagItem, MomentItem, TrendingBundle,
  SearchUsersRequest, CreateReportRequest, Relationship
} from '../../../core/types/api';
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
    return this.trendingTags.length > 0 || this.recommendedUsers.length > 0 || this.hotMoments.length > 0;
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
        const results = await LoadingUtil.withMinimumTime(() =>
          Promise.all([trendingPromise, ApiClient.searchUsers(usersRequest)])
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
            // 标签药片
            if (this.getAvailableTags().length > 0) {
              ListItem() {
                this.buildTagChipsStrip()
              }
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
              : this.getCurrentThemePalette().background)
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
      top: 12,
      bottom: 12
    })
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
