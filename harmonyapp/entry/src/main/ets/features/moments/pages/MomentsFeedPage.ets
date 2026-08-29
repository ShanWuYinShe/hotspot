/**
 * 附近动态页面
 * 展示热门话题、推荐用户、热门瞬间
 */

import { GlobalStateManager, KEY_CURRENT_LOCATION, KEY_BROWSE_ANCHOR, KEY_IS_LOGGED_IN } from '../../../core/state/GlobalStateManager';
import { PageShellContainer } from '../../../components/PageShellContainer';
import type { BreakpointType } from '../../../core/responsive/Breakpoint';
import { useMasterDetail } from '../../../core/responsive/Breakpoint';
import { LocationInfo } from '../../../core/location/LocationUtil';
import { ApiClient } from '../../../core/api/ApiClient';
import { getErrorMessage } from '../../../core/network/ApiErrorUtil';
import { AppRouter, ChatRouteParam, GroupDetailRouteParam, LoginRouteParam, MomentDetailRouteParam, PhotoPreviewRouteParam, ProfileRouteParam } from '../../../core/router/AppRouter';
import { PageNames, DefaultConfig } from '../../../constants/AppConstants';
import { interactionActions } from '../../../core/state/InteractionActions';
import { InteractionStateStore, UserRelationState } from '../../../core/state/InteractionStateStore';
import {
  UserItem, GroupItem, MomentItem,
  SearchUsersRequest, SearchGroupsRequest, CreateReportRequest, Relationship, GeoPoint, PaginationParams,
  SearchMomentsRequest, FetchFollowingMomentsRequest, FetchNearbyMomentsRequest
} from '../../../core/types/api';
import { curves } from '@kit.ArkUI';
import { LocationUtil } from '../../../core/location/LocationUtil';
import { SystemAvatar } from '../../../components/SystemAvatar';
import { SystemCard, CardVariant } from '../../../components/SystemCard';
import { SystemLoadingView } from '../../../components/SystemLoadingView';
import { SystemToast } from '../../../components/SystemToast';
import { SystemPageHeader } from '../../../components/SystemPageHeader';
import { TabsPillGroup, TabItem } from '../../../components/TabsPillGroup';
import { DesignConstants } from '../../../constants/DesignConstants';
import { AppIcon } from '../../../components/SystemIcon';
import { MomentFeedCard } from '../../../components/MomentFeedCard';
import { LoadingUtil } from '../../../core/utils/LoadingUtil';
import { BasicDataSource } from '../../../core/utils/BasicDataSource';
import { SystemUiController } from '../../../core/utils/SystemUiController';
import { SafeAreaUtils } from '../../../core/utils/SafeAreaUtils';
import { EmptyState, RefreshIndicator } from '../../../components/EmptyState';
import { SystemButton } from '../../../components/SystemButton';
import { hilog } from '@kit.PerformanceAnalysisKit';
import pasteboard from '@ohos.pasteboard';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { KEY_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';

const DOMAIN_NUMBER: number = 0xFF11;
const TAG: string = 'MomentsFeedPage';

class SearchParams {
  keyword: string = '';
}


@Preview
@Component
export struct MomentsFeedPage {
  @State userRowPressed: boolean = false;
  @StorageLink(KEY_IS_LOGGED_IN) isLoggedIn: boolean = false;
  @StorageLink(KEY_BROWSE_ANCHOR) browseAnchor: LocationInfo | null = null;
  @StorageLink(KEY_CURRENT_LOCATION) currentLocation: LocationInfo | null = null;
  @StorageLink('safeAreaTop') safeAreaTop: number = 0;
  @StorageLink('safeAreaBottom') safeAreaBottom: number = 0;
  @StorageLink(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'vibrant_light';
  @StorageLink('interaction_user_state') userStateMap: Record<string, UserRelationState> = {};
  @StorageProp('isTablet') isTablet: boolean = false;
  @StorageProp('currentBreakpoint') currentBreakpoint: BreakpointType = 'sm';
  /** 作为路由页 push 时显示返回按钮（tab 场景隐藏，对齐前端 /nearby/moments 独立页） */
  @Prop showBackButton: boolean = false;

  @State isLoading: boolean = true;
  @State errorMessage: string = '';
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

  private getAvailableTags(): string[] {
    const tagSet = new Set<string>();
    for (const moment of this.hotMoments) {
      for (const tag of (moment.tags ?? [])) {
        tagSet.add(tag);
      }
    }
    return Array.from(tagSet);
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
    return this.recommendedUsers.length > 0 ||
      this.recommendedGroups.length > 0 || this.hotMoments.length > 0;
  }

  private async loadFeedMoments(source: string): Promise<void> {
    const refLoc: LocationInfo | null = this.browseAnchor ?? this.currentLocation;
    // 契约：reference_point 可选，无定位时不传（服务端按回退链解析），不再用 0,0 兜底
    const refPoint: GeoPoint | undefined = refLoc ? { latitude: refLoc.latitude, longitude: refLoc.longitude } : undefined;
    const pagination: PaginationParams = { limit: 20 };

    if (source === 'following') {
      const request: FetchFollowingMomentsRequest = {
        reference_point: refPoint,
        pagination: pagination
      };
      const response = await LoadingUtil.withMinimumTime(() => ApiClient.fetchFollowingMoments(request));
      if (this.isDestroyed) return;
      if (response.code === 0) {
        this.hotMoments = response.data?.items ?? [];
        this.hotMomentsDataSource.setData(this.hotMoments);
        this.hotMoments.forEach((item: MomentItem) => this.interactionStore.patchMomentFromItem(item));
        this.recommendedUsers = [];
        this.recommendedGroups = [];
        this.recommendedUsersDataSource.setData([]);
      } else if (!this.hasAnyData()) {
        this.errorMessage = getErrorMessage(response, '加载失败');
      }
    } else {
      const request: FetchNearbyMomentsRequest = {
        reference_point: refPoint,
        radius: DefaultConfig.NEARBY_RADIUS,
        pagination: pagination
      };
      const response = await LoadingUtil.withMinimumTime(() => ApiClient.fetchNearbyMoments(request));
      if (this.isDestroyed) return;
      if (response.code === 0) {
        this.hotMoments = response.data?.items ?? [];
        this.hotMomentsDataSource.setData(this.hotMoments);
        this.hotMoments.forEach((item: MomentItem) => this.interactionStore.patchMomentFromItem(item));
        this.recommendedUsers = [];
        this.recommendedGroups = [];
        this.recommendedUsersDataSource.setData([]);
      } else if (!this.hasAnyData()) {
        this.errorMessage = getErrorMessage(response, '加载失败');
      }
    }
  }

  private getMomentsSectionTitle(): string {
    if (this.activeTab === '关注') {
      return '关注的动态';
    }
    if (this.activeTab === '附近') {
      return '附近的动态';
    }
    return '热门动态';
  }

  private async loadData(refresh: boolean = false): Promise<void> {
    if (!refresh) { this.isLoading = true; }
    this.errorMessage = '';
    try {
      if (this.activeTab === '关注' || this.activeTab === '附近') {
        await this.loadFeedMoments(this.activeTab === '关注' ? 'following' : 'nearby');
        return;
      }
      // 热门内容不依赖位置，推荐用户需要位置
      const refLoc: LocationInfo | null = this.browseAnchor ?? this.currentLocation;
      // spec：热门模式走 moments/search 的 sort_by='popular'（原 feed/trending 收口）
      const trendingRequest: SearchMomentsRequest = { sort_by: 'popular', pagination: { limit: 20 } };
      const trendingPromise = ApiClient.searchMoments(trendingRequest);

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
          this.hotMoments = trendingResponse.data?.items ?? [];
          this.hotMomentsDataSource.setData(this.hotMoments);
          this.hotMoments.forEach((item: MomentItem) => this.interactionStore.patchMomentFromItem(item));
        }
        if (usersResponse.code !== 0 && trendingResponse.code !== 0 && !this.hasAnyData()) {
          this.errorMessage = getErrorMessage(trendingResponse, getErrorMessage(usersResponse, '加载附近动态失败'));
        }
      } else {
        const trendingResponse = await LoadingUtil.withMinimumTime(() => trendingPromise);
        if (this.isDestroyed) return;
        if (trendingResponse.code === 0) {
          this.hotMoments = trendingResponse.data?.items ?? [];
          this.hotMomentsDataSource.setData(this.hotMoments);
          this.hotMoments.forEach((item: MomentItem) => this.interactionStore.patchMomentFromItem(item));
        } else if (!this.hasAnyData()) {
          this.errorMessage = getErrorMessage(trendingResponse, '加载附近动态失败');
        }
      }
    } catch {
      hilog.error(DOMAIN_NUMBER, TAG, '加载附近动态失败');
      if (!this.hasAnyData()) {
        this.errorMessage = '加载附近动态失败';
      }
    } finally {
      this.isLoading = false;
      this.isRefreshing = false;
    }
  }

  build() {
  PageShellContainer({ variant: 'feed-medium', currentBreakpoint: this.currentBreakpoint, noHorizontalPadding: true }) {
    Column() {
      // 粘性头部：标题(走 SystemPageHeader) + 搜索栏 + 标签栏（对齐前端 nearby/moments 页 headerContent）
      Column() {
        // 标题行 —— 统一走 SystemPageHeader（标题 + 内容数 trailing，透明背景自带安全区）
        SystemPageHeader({
          title: '附近动态',
          titleSize: DesignConstants.FONT.SIZE.XL,
          titleWeight: DesignConstants.FONT.WEIGHT.MEDIUM,
          backgroundVariant: 'transparent',
          showBackButton: this.showBackButton,
          onBack: this.showBackButton ? () => { AppRouter.pop(); } : null
        }) {
          if (this.hotMoments.length > 0) {
            Text(`${this.hotMoments.length} 条内容`)
              .fontSize(DesignConstants.FONT.SIZE.SM)
              .fontColor(this.getCurrentThemePalette().textSecondary)
          }
        }

        // 搜索栏（对齐前端：search icon + input + clear button）
        Row({ space: DesignConstants.SPACING.MD2 }) {
          Text() {
            SymbolSpan($r(AppIcon.SEARCH))
              .fontSize(DesignConstants.FONT.SIZE.MD)
              .fontColor([this.getCurrentThemePalette().textSecondary])
          }
          Text('搜索用户、动态、地点或话题')
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .fontColor(this.getCurrentThemePalette().textSecondary)
            .layoutWeight(1)
            .maxLines(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
        }
        .height(DesignConstants.COMPONENT.INPUT_HEIGHT)
        .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG })
        .backgroundColor(this.getCurrentThemePalette().inputBackgroundColor)
        .borderRadius(DesignConstants.SHAPE.INPUT)
        .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getSurfaceBorder() })
        .onClick(() => { AppRouter.push(PageNames.SEARCH); })
        .margin({ top: DesignConstants.SPACING.SM, bottom: DesignConstants.SPACING.XS })

        // 统一 TabsPillGroup 分段胶囊（对齐前端 TabBar）
        TabsPillGroup({
          tabs: this.TABS.map((tab: string) => ({ label: tab } as TabItem)),
          activeIndex: Math.max(0, this.TABS.indexOf(this.activeTab)),
          actions: {
            onChange: (index: number) => {
              const tab = this.TABS[index];
              if (tab && tab !== this.activeTab) {
                this.activeTab = tab;
                this.loadData(true);
              }
            }
          }
        })
      }
      .width('100%')
      .padding({
        left: DesignConstants.SPACING.LG,
        right: DesignConstants.SPACING.LG,
        bottom: 0
      })
      .backgroundColor(this.getCurrentThemePalette().pageBackground)
      .border({ width: { bottom: DesignConstants.BORDER_WIDTH.HAIRLINE }, color: this.getSurfaceBorder() })

      // 内容区
      if (this.isLoading) {
        SystemLoadingView({ message: '正在加载附近动态...' })
          .layoutWeight(1)
          .transition(TransitionEffect.OPACITY.animation({ duration: DesignConstants.ANIMATION.DURATION_SHEET, curve: DesignConstants.ANIMATION.CURVE_FAST_OUT_SLOW_IN })
            .combine(TransitionEffect.move(TransitionEdge.BOTTOM).animation({ duration: DesignConstants.ANIMATION.DURATION_SHEET, curve: DesignConstants.ANIMATION.CURVE_FAST_OUT_SLOW_IN })))
      } else if (!this.hasAnyData() && this.errorMessage) {
        EmptyState({
          icon: AppIcon.WARNING,
          message: this.errorMessage,
          actionLabel: '重试',
          onAction: () => { this.loadData(); }
        })
        .layoutWeight(1)
        .transition(TransitionEffect.OPACITY.animation({ duration: DesignConstants.ANIMATION.DURATION_SHEET, curve: DesignConstants.ANIMATION.CURVE_FAST_OUT_SLOW_IN })
          .combine(TransitionEffect.move(TransitionEdge.BOTTOM).animation({ duration: DesignConstants.ANIMATION.DURATION_SHEET, curve: DesignConstants.ANIMATION.CURVE_FAST_OUT_SLOW_IN })))
      } else {
        Refresh({ refreshing: $$this.isRefreshing, builder: RefreshIndicator }) {
          if (useMasterDetail(this.currentBreakpoint)) {
            // ======== 宽屏两栏布局（master-detail，对齐前端 useMasterDetail） ========
            Row() {
              // 左栏：标签药片 + 动态网格
              Scroll() {
                Column() {
                  this.buildTagChipsStrip()
                  this.buildMomentFeed()
                }
                .width('100%')
              }
              .layoutWeight(1)
              .scrollBar(BarState.Auto)
              // 桌面分支同样给滚动尾部留呼吸（对齐移动分支的 bottom 安全区语义）
              .padding({ bottom: DesignConstants.SPACING.LG })

              // 右栏：热门话题 + 活跃用户（对齐前端 rightSidebar，宽 220vp + border-left）
              Column() {
                this.buildRightSidebar()
              }
              .width(DesignConstants.LAYOUT.RIGHT_SIDEBAR_WIDTH)
              .height('100%')
              .backgroundColor(this.getCurrentThemePalette().surfaceBackground)
              .border({ width: { left: DesignConstants.BORDER_WIDTH.HAIRLINE }, color: this.getSurfaceBorder() })
            }
            .layoutWeight(1)
            .width('100%')
          } else {
            // ======== 移动端单栏布局（对齐前端 !isDesktop 分支） ========
            Scroll() {
              Column() {
                this.buildTagChipsStrip()
                this.buildMomentFeed()
              }
              .width('100%')
            }
            .layoutWeight(1)
            .scrollBar(BarState.Auto)
            .padding({
              bottom: SafeAreaUtils.resolveBottomPadding(this.safeAreaBottom, DesignConstants.SPACING.LG, 0)
            })
          }
        }
        .layoutWeight(1)
        .onRefreshing(() => { this.loadData(true); })
        .transition(TransitionEffect.OPACITY.animation({ duration: DesignConstants.ANIMATION.DURATION_SHEET, curve: DesignConstants.ANIMATION.CURVE_FAST_OUT_SLOW_IN })
          .combine(TransitionEffect.move(TransitionEdge.BOTTOM).animation({ duration: DesignConstants.ANIMATION.DURATION_SHEET, curve: DesignConstants.ANIMATION.CURVE_FAST_OUT_SLOW_IN })))
      }
    }
    .width('100%')
    .height('100%')
    .backgroundColor(this.getCurrentThemePalette().pageBackground)
  }
  }

  // ==================== 动态列表/网格（对齐前端 feedGrid） ====================
  @Builder
  buildMomentFeed() {
    Column() {
      // section 标题（对齐前端：eyebrow 小标 + 主标题层级）
      Row() {
        Text(this.getMomentsSectionTitle())
          .fontSize(DesignConstants.FONT.SIZE.LG_PLUS) // 18，对齐前端 page-title
          .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
          .fontColor(this.getCurrentThemePalette().textPrimary)
      }
      .width('100%')
      .padding({
        left: DesignConstants.SPACING.LG,
        right: DesignConstants.SPACING.LG,
        top: DesignConstants.SPACING.LG,
        bottom: DesignConstants.SPACING.SM
      })

      // Hero first card（对齐前端 ambient-glow 首个卡片突出）
      if (this.hotMoments.length > 0) {
        Column() {
          MomentFeedCard({
            moment: this.hotMoments[0],
            variant: 'postCard',
            showFavorite: false,
            onOpenDetail: (item: MomentItem) => this.openMomentDetail(item),
            onImagePreview: (momentId: string, images: string[], index: number) => {
              this.previewImage(momentId, images, index);
            }
          })
        }
        .width('100%')
        .padding({
          left: DesignConstants.SPACING.LG,
          right: DesignConstants.SPACING.LG,
          bottom: DesignConstants.SPACING.XS
        })
      }

      // 剩余动态用紧凑卡片列表（对齐前端 feedGrid 的 dense 模式）
      if (this.hotMoments.length > 1) {
        ForEach(this.hotMoments.slice(1), (moment: MomentItem) => {
          MomentFeedCard({
            moment: moment,
            variant: 'default',
            showFavorite: false,
            onOpenDetail: (item: MomentItem) => this.openMomentDetail(item),
            onImagePreview: (momentId: string, images: string[], index: number) => {
              this.previewImage(momentId, images, index);
            }
          })
          .padding({
            left: DesignConstants.SPACING.LG,
            right: DesignConstants.SPACING.LG,
            top: DesignConstants.SPACING.XS,
            bottom: DesignConstants.SPACING.XS
          })
        }, (moment: MomentItem) => moment.moment_id)
      }

      // 空态（对齐前端 EmptyState：图标 + 文案，不再裸灰块）
      if (this.hotMoments.length === 0) {
        Column() {
          EmptyState({
            icon: AppIcon.RECOMMEND,
            message: '附近暂时没有瞬间',
            actionLabel: '刷新',
            onAction: () => { this.loadData(true); }
          })
        }
        .width('100%')
        .padding({ top: DesignConstants.SPACING.XXL, bottom: DesignConstants.SPACING.XXL })
      }
    }
    .width('100%')
  }

  // ==================== 右侧栏（桌面端，对齐前端 rightSidebar） ====================
  @Builder
  buildRightSidebar() {
    Column() {
      // 活跃用户（对齐前端 active users：eyebrow + 列表项）
      if (this.recommendedUsers.length > 0) {
        Column() {
          Text('活跃用户')
            .fontSize(DesignConstants.FONT.SIZE.XS2)
            .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
            .fontColor(this.getCurrentThemePalette().textSecondary)
            .textCase(TextCase.UpperCase)
            .letterSpacing(DesignConstants.FONT.LETTER_SPACING.LABEL)
            .padding({
              left: DesignConstants.SPACING.LG,
              right: DesignConstants.SPACING.LG,
              top: DesignConstants.SPACING.XL,
              bottom: DesignConstants.SPACING.SM
            })

          ForEach(this.recommendedUsers.slice(0, 5), (user: UserItem) => {
            Row() {
              SystemAvatar({
                avatarUrl: user.user?.avatar_url ?? '',
                name: user.user?.nickname ?? '',
                avatarSize: DesignConstants.COMPONENT.AVATAR_SIZE.XS,
                shape: 'circle'
              })
              Text(user.user?.nickname ?? '')
                .fontSize(DesignConstants.FONT.SIZE.SM) // 12 对齐前端 text-meta
                .fontColor(this.getCurrentThemePalette().textPrimary)
                .maxLines(1)
                .textOverflow({ overflow: TextOverflow.Ellipsis })
                .layoutWeight(1)
                .margin({ left: DesignConstants.SPACING.SM })

              SystemButton({
                label: '关注',
                variant: 'primary',
                buttonSize: 'compact',
                onPress: () => { this.followUser(user); }
              })
            }
            .width('100%')
            .alignItems(VerticalAlign.Center)
            .padding({
              left: DesignConstants.SPACING.LG,
              right: DesignConstants.SPACING.LG,
              top: DesignConstants.SPACING.XS,
              bottom: DesignConstants.SPACING.XS
            })
          }, (user: UserItem) => user.user?.user_id ?? '')
        }
        .width('100%')
      }

      Blank().layoutWeight(1)
    }
    .width('100%')
    .height('100%')
  }

  // ==================== 标签药片条 ====================
  @Builder
  buildTagChipsStrip() {
    Scroll() {
      Row({ space: DesignConstants.SPACING.SM }) {
        ForEach(this.getAvailableTags(), (tag: string) => {
          Text(tag)
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .fontColor(this.activeTag === tag
              ? this.getCurrentThemePalette().accentText
              : this.getCurrentThemePalette().textSecondary)
            .backgroundColor(this.activeTag === tag
              ? this.getCurrentThemePalette().accentBackground
              : this.getCurrentThemePalette().surfaceBackground)
            .borderRadius(DesignConstants.RADIUS.FULL)
            .padding({
              left: DesignConstants.SPACING.MD3,
              right: DesignConstants.SPACING.MD3,
              top: DesignConstants.SPACING.SM2,
              bottom: DesignConstants.SPACING.SM2
            })
            .shadow(this.activeTag === tag
              ? this.getCurrentThemePalette().accentShadow
              : this.getCurrentThemePalette().controlShadow)
            .border({
              width: DesignConstants.BORDER_WIDTH.HAIRLINE,
              color: this.activeTag === tag
                ? DesignConstants.COLOR.TRANSPARENT
                : this.getSurfaceBorder()
            })
            // 选中底色/字色/阴影切换过渡（对齐 DesktopLayout 导航项，避免硬跳）
            .animation({ duration: DesignConstants.ANIMATION.DURATION_FAST, curve: DesignConstants.ANIMATION.CURVE_SMOOTH })
            .onClick(() => {
              this.activeTag = this.activeTag === tag ? '' : tag;
            })
        }, (tag: string) => tag)
      }
      .padding({ left: DesignConstants.SPACING.XS, right: DesignConstants.SPACING.XS })
    }
    .scrollable(ScrollDirection.Horizontal)
    .scrollBar(BarState.Off)
    .width('100%')
    .padding({
      left: DesignConstants.SPACING.LG,
      right: DesignConstants.SPACING.LG,
      top: DesignConstants.SPACING.XS,
      bottom: DesignConstants.SPACING.XS
    })
  }

  // ==================== section 标题 ====================
  @Builder
  buildSectionHeader(title: string) {
    Row() {
      Text(title)
        .fontSize(DesignConstants.FONT.SIZE.MD)
        .fontWeight(DesignConstants.FONT.WEIGHT.BOLD)
        .fontColor(this.getCurrentThemePalette().textPrimary)
    }
    .width('100%')
    .padding({
      left: DesignConstants.SPACING.LG,
      right: DesignConstants.SPACING.LG,
      top: DesignConstants.SPACING.LG,
      bottom: DesignConstants.SPACING.XS
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
          Row({ space: DesignConstants.SPACING.MD3 }) {
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
    Column({ space: DesignConstants.SPACING.SM2 }) {
      Stack({ alignContent: Alignment.Center }) {
        SystemAvatar({
          avatarUrl: user.user?.avatar_url ?? '',
          name: user.user?.nickname ?? '',
          avatarSize: DesignConstants.COMPONENT.AVATAR_SIZE.MEDIUM,
          shape: 'squircle'
        })

        // 距离徽标：贴头像左下角（用 alignContent 取代硬编码坐标）
        if (user.location?.distance !== undefined && user.location?.distance !== null) {
          Stack({ alignContent: Alignment.BottomStart }) {
            Row() {
              Text(LocationUtil.formatDistance(user.location.distance))
                .fontSize(DesignConstants.FONT.SIZE.XS2)
                .fontColor(this.getCurrentThemePalette().accentText)
            }
            .padding({ left: DesignConstants.SPACING.SM2, right: DesignConstants.SPACING.SM2, top: DesignConstants.SPACING.HALF, bottom: DesignConstants.SPACING.HALF })
            .borderRadius(DesignConstants.RADIUS.FULL)
            .backgroundColor(this.getCurrentThemePalette().accentBackground)
          }
          .width('100%')
          .height('100%')
        }
      }
      .width(DesignConstants.LAYOUT.LABEL_WIDTH)
      .height(DesignConstants.COMPONENT.EMPTY_ICON_SIZE)

      Text(user.user?.nickname ?? '')
        .fontSize(DesignConstants.FONT.SIZE.XS)
        .fontColor(this.getCurrentThemePalette().textPrimary)
        .maxLines(1)
        .textOverflow({ overflow: TextOverflow.Ellipsis })
        .constraintSize({ maxWidth: 60 })
    }
    .alignItems(HorizontalAlign.Center)
    .scale(this.userRowPressed
      ? { x: DesignConstants.INTERACTION.SCALE_PRESSED_INPUT_ROW, y: DesignConstants.INTERACTION.SCALE_PRESSED_INPUT_ROW }
      : { x: 1, y: 1 })
    .animation({
      curve: curves.springMotion(
        DesignConstants.INTERACTION.SPRING_RESPONSE_PRESS,
        DesignConstants.INTERACTION.SPRING_DAMPING_PRESS
      )
    })
    .onTouch((event: TouchEvent) => {
      if (event.type === TouchType.Down) {
        this.userRowPressed = true;
      } else if (event.type === TouchType.Up || event.type === TouchType.Cancel) {
        this.userRowPressed = false;
      }
    })
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
        Column({ space: DesignConstants.SPACING.MD }) {
          this.buildGroupRow(this.recommendedGroups.slice(0, 2))
          if (this.recommendedGroups.length > 2) {
            this.buildGroupRow(this.recommendedGroups.slice(2, 4))
          }
        }
        .width('100%')
      }
    }
    .margin({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG })
  }

  // 两列等宽群组卡片行（前端 grid-cols-2 的 ArkUI 等价：Row + layoutWeight，避开 calc）
  @Builder
  buildGroupRow(items: GroupItem[]) {
    Row({ space: DesignConstants.SPACING.MD }) {
      ForEach(items, (group: GroupItem) => {
        Column() {
          this.buildGroupCard(group)
        }
        .layoutWeight(1)
      }, (group: GroupItem) => group.group?.group_id ?? '')
      if (items.length === 1) {
        Column().layoutWeight(1)
      }
    }
    .width('100%')
  }

  @Builder
  buildGroupCard(group: GroupItem) {
    Column() {
      Row() {
        Text() {
          SymbolSpan($r(AppIcon.GROUP))
            .fontSize(DesignConstants.FONT.SIZE.XXL)
            .fontColor([this.getCurrentThemePalette().accentBackground])
        }
      }
      .width('100%')
      .height(DesignConstants.COMPONENT.BUTTON_HEIGHT_LG)
      .justifyContent(FlexAlign.Center)
      .backgroundColor(this.getCurrentThemePalette().accentMuted)
      .borderRadius({ topLeft: DesignConstants.RADIUS.LG, topRight: DesignConstants.RADIUS.LG })

      Column({ space: DesignConstants.SPACING.XS }) {
        Text(group.group?.name ?? '')
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
          .fontColor(this.getCurrentThemePalette().textPrimary)
          .maxLines(1)
          .textOverflow({ overflow: TextOverflow.Ellipsis })

        Row({ space: DesignConstants.SPACING.SM2 }) {
          Text(`${group.group?.members_count ?? 0} 人`)
            .fontSize(DesignConstants.FONT.SIZE.XS2)
            .fontColor(this.getCurrentThemePalette().textSecondary)
          if (group.location?.distance !== undefined && group.location?.distance !== null) {
            Text(LocationUtil.formatDistance(group.location.distance))
              .fontSize(DesignConstants.FONT.SIZE.XS2)
              .fontColor(this.getCurrentThemePalette().accentBackground)
          }
        }
      }
      .padding({ left: DesignConstants.SPACING.MD2, right: DesignConstants.SPACING.MD2, top: DesignConstants.SPACING.SM, bottom: DesignConstants.SPACING.MD2 })
      .alignItems(HorizontalAlign.Start)
    }
    .width('100%')
    .borderRadius(DesignConstants.RADIUS.LG)
    .backgroundColor(this.getCurrentThemePalette().surfaceBackground)
    .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getCurrentThemePalette().borderColor })
    .shadow(this.getCurrentThemePalette().controlShadow)
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

  // 卡片图片点击直达全屏大图预览（与"我的瞬间"/详情页入口一致）
  private previewImage(momentId: string, images: string[], index: number): void {
    const previewParams = new PhotoPreviewRouteParam();
    previewParams.momentId = momentId;
    previewParams.images = images;
    previewParams.index = index;
    AppRouter.push(PageNames.PHOTO_PREVIEW, previewParams);
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
        SystemToast.showError(getErrorMessage(response, '举报失败'), this.getUIContext());
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

@Builder
export function MomentsFeedPageBuilder() {
  MomentsFeedPage({ showBackButton: true })
}