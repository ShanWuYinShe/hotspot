/**
 * 我的瞬间（/me/moments）
 *
 * 对齐 web ProfilePage「我的瞬间」分区（components/profile/SectionMoments.svelte）：
 * - 页头 18/600，下方 pills tab（发布 / 喜欢 / 收藏，带计数）+ 右侧记录数
 * - 2 列卡片网格（复用 MomentFeedCard overlay 3:4，卡片本身即软岛）
 * - 空态 / 错误 / 首屏骨架与 web 同语义；发布动作保留页头右侧入口
 *
 * 数据分别来自 moments/fetch_created、users/fetch_reacted_moments、moments/fetch_favorites。
 */

import { GlobalStateManager } from '../../../core/state/GlobalStateManager';
import { ApiClient } from '../../../core/api/ApiClient';
import { PageNames, DefaultConfig } from '../../../constants/AppConstants';
import {
  MomentItem,
  FetchCreatedMomentsRequest,
  FetchFavoritesRequest,
  FetchUserReactedMomentsRequest,
  PaginationParams
} from '../../../core/types/api';
import { SystemPageHeader } from '../../../components/SystemPageHeader';
import { EmptyState, LoadingMoreIndicator, RefreshIndicator } from '../../../components/EmptyState';
import { Skeleton } from '../../../components/states/Skeleton';
import { SkeletonShimmer } from '../../../components/states/SkeletonShimmer';
import { ListEnd } from '../../../components/ListEnd';
import { PageShellContainer } from '../../../components/PageShellContainer';
import { TabsPillGroup, TabItem } from '../../../components/TabsPillGroup';
import { SystemButton } from '../../../components/SystemButton';
import { AppIcon } from '../../../components/SystemIcon';
import { MomentFeedCard } from '../../../components/MomentFeedCard';
import { DesignConstants, textLineHeight } from '../../../constants/DesignConstants';
import type { BreakpointType } from '../../../core/responsive/Breakpoint';
import { BasicDataSource } from '../../../core/utils/BasicDataSource';
import { SystemToast } from '../../../components/SystemToast';
import { AppRouter, LoginRouteParam, PhotoPreviewRouteParam } from '../../../core/router/AppRouter';
import { SafeAreaUtils } from '../../../core/utils/SafeAreaUtils';
import { SystemUiController } from '../../../core/utils/SystemUiController';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { DEFAULT_RESOLVED_THEME_ID, KEY_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';
import { getErrorMessage } from '../../../core/network/ApiErrorUtil';
import { hilog } from '@kit.PerformanceAnalysisKit';

const DOMAIN_NUMBER: number = 0xFF05;
const TAG: string = 'MomentPage';
const PAGE_SIZE: number = DefaultConfig.PAGE_SIZE_LARGE;
const TAB_POSTS: number = 0;
const TAB_LIKED: number = 1;
const TAB_SAVED: number = 2;

interface TabFetchResult {
  code: number;
  message: string;
  items: MomentItem[];
  nextCursor: string;
}

@Preview
@Component
export struct MomentPage {
  @StorageProp('currentBreakpoint') currentBreakpoint: BreakpointType = 'sm';
  @StorageLink('safeAreaTop') safeAreaTop: number = 0;
  @StorageLink('safeAreaBottom') safeAreaBottom: number = 0;
  @StorageLink(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = DEFAULT_RESOLVED_THEME_ID;

  private stateManager = GlobalStateManager.getInstance();
  private momentsDataSource: BasicDataSource<MomentItem> = new BasicDataSource<MomentItem>([]);
  private isDestroyed: boolean = false;

  @State activeTab: number = TAB_POSTS;
  @State posts: MomentItem[] = [];
  @State liked: MomentItem[] = [];
  @State saved: MomentItem[] = [];
  @State postsCursor: string = '';
  @State likedCursor: string = '';
  @State savedCursor: string = '';
  @State postsHasMore: boolean = false;
  @State likedHasMore: boolean = false;
  @State savedHasMore: boolean = false;
  @State loadedTabs: boolean[] = [false, false, false];
  @State isLoggedIn: boolean = false;
  @State isLoading: boolean = true;
  @State isRefreshing: boolean = false;
  @State isLoadingMore: boolean = false;
  @State errorMessage: string = '';

  async aboutToAppear() {
    SystemUiController.syncWithCurrentTheme();
    const user = this.stateManager.getCurrentUser();
    if (!user) {
      this.isLoggedIn = false;
      this.isLoading = false;
      return;
    }
    this.isLoggedIn = true;
    await this.loadAllTabs();
  }

  aboutToDisappear(): void {
    this.isDestroyed = true;
  }

  private getCurrentThemePalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  private getUserId(): string {
    return this.stateManager.getCurrentUser()?.user_id ?? '';
  }

  private activeItems(): MomentItem[] {
    if (this.activeTab === TAB_LIKED) {
      return this.liked;
    }
    if (this.activeTab === TAB_SAVED) {
      return this.saved;
    }
    return this.posts;
  }

  private activeHasMore(): boolean {
    if (this.activeTab === TAB_LIKED) {
      return this.likedHasMore;
    }
    if (this.activeTab === TAB_SAVED) {
      return this.savedHasMore;
    }
    return this.postsHasMore;
  }

  private setTabData(tab: number, items: MomentItem[], cursor: string, hasMore: boolean): void {
    if (tab === TAB_LIKED) {
      this.liked = items;
      this.likedCursor = cursor;
      this.likedHasMore = hasMore;
    } else if (tab === TAB_SAVED) {
      this.saved = items;
      this.savedCursor = cursor;
      this.savedHasMore = hasMore;
    } else {
      this.posts = items;
      this.postsCursor = cursor;
      this.postsHasMore = hasMore;
    }
    const loaded: boolean[] = this.loadedTabs.slice();
    loaded[tab] = true;
    this.loadedTabs = loaded;
    if (tab === this.activeTab) {
      this.refreshDataSource();
    }
  }

  private refreshDataSource(): void {
    this.momentsDataSource.setData(this.activeItems());
  }

  private async fetchTab(tab: number, cursor: string): Promise<TabFetchResult> {
    const pagination: PaginationParams = { limit: PAGE_SIZE };
    if (cursor.length > 0) {
      pagination.cursor = cursor;
    }
    try {
      if (tab === TAB_LIKED) {
        const userId = this.getUserId();
        if (!userId) {
          return { code: 10100, message: '请先登录', items: [], nextCursor: '' };
        }
        const request: FetchUserReactedMomentsRequest = { user_id: userId, pagination };
        const response = await ApiClient.getUserReactedMoments(request);
        if (response.code === 0) {
          return {
            code: 0,
            message: '',
            items: response.data?.items ?? [],
            nextCursor: response.data?.next_cursor ?? ''
          };
        }
        return { code: response.code, message: getErrorMessage(response, '加载喜欢的瞬间失败'), items: [], nextCursor: '' };
      }
      if (tab === TAB_SAVED) {
        const request: FetchFavoritesRequest = { pagination };
        const response = await ApiClient.getMyFavorites(request);
        if (response.code === 0) {
          return {
            code: 0,
            message: '',
            items: response.data?.items ?? [],
            nextCursor: response.data?.next_cursor ?? ''
          };
        }
        return { code: response.code, message: getErrorMessage(response, '加载收藏的瞬间失败'), items: [], nextCursor: '' };
      }
      const request: FetchCreatedMomentsRequest = { pagination };
      const response = await ApiClient.getMyMoments(request);
      if (response.code === 0) {
        return {
          code: 0,
          message: '',
          items: response.data?.items ?? [],
          nextCursor: response.data?.next_cursor ?? ''
        };
      }
      return { code: response.code, message: getErrorMessage(response, '加载我的瞬间失败'), items: [], nextCursor: '' };
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `加载我的瞬间失败: ${JSON.stringify(error)}`);
      return {
        code: -1,
        message: error instanceof Error ? error.message : '加载我的瞬间失败',
        items: [],
        nextCursor: ''
      };
    }
  }

  private async loadAllTabs(): Promise<void> {
    this.isLoading = true;
    this.errorMessage = '';
    await this.loadTab(TAB_POSTS, true);
    await this.loadTab(TAB_LIKED, true);
    await this.loadTab(TAB_SAVED, true);
    if (!this.isDestroyed) {
      this.isLoading = false;
      this.refreshDataSource();
    }
  }

  private async loadTab(tab: number, replace: boolean): Promise<void> {
    const cursor = replace ? '' : this.cursorForTab(tab);
    const result = await this.fetchTab(tab, cursor);
    if (this.isDestroyed) {
      return;
    }
    if (result.code === 10100) {
      this.isLoggedIn = false;
      return;
    }
    if (result.code !== 0) {
      if (replace) {
        this.errorMessage = result.message;
        this.setTabData(tab, [], '', false);
      }
      return;
    }
    const existing = replace ? [] : this.itemsForTab(tab);
    const nextItems = existing.concat(result.items);
    this.setTabData(tab, nextItems, result.nextCursor, result.nextCursor.length > 0);
  }

  private cursorForTab(tab: number): string {
    if (tab === TAB_LIKED) {
      return this.likedCursor;
    }
    if (tab === TAB_SAVED) {
      return this.savedCursor;
    }
    return this.postsCursor;
  }

  private itemsForTab(tab: number): MomentItem[] {
    if (tab === TAB_LIKED) {
      return this.liked;
    }
    if (tab === TAB_SAVED) {
      return this.saved;
    }
    return this.posts;
  }

  private switchTab(index: number): void {
    if (index === this.activeTab || this.isLoading) {
      return;
    }
    this.activeTab = index;
    this.refreshDataSource();
    if (!this.loadedTabs[index]) {
      this.loadTab(index, true);
    }
  }

  private async refreshActive(): Promise<void> {
    if (this.isRefreshing) {
      return;
    }
    this.isRefreshing = true;
    await this.loadTab(this.activeTab, true);
    if (!this.isDestroyed) {
      this.isRefreshing = false;
      this.refreshDataSource();
    }
  }

  private async loadMore(): Promise<void> {
    if (this.isLoadingMore || !this.activeHasMore()) {
      return;
    }
    this.isLoadingMore = true;
    await this.loadTab(this.activeTab, false);
    if (!this.isDestroyed) {
      this.isLoadingMore = false;
    }
  }

  private openComposer(): void {
    const currentUser = this.stateManager.getCurrentUser();
    if (!currentUser) {
      SystemToast.showWarning('请先登录后再发布', this.getUIContext());
      const lp = new LoginRouteParam();
      lp.reason = 'action_required';
      lp.action = '发布瞬间';
      lp.redirectUrl = PageNames.CREATE_MOMENT;
      AppRouter.push(PageNames.LOGIN, lp);
      return;
    }
    AppRouter.push(PageNames.CREATE_MOMENT);
  }

  private openLogin(): void {
    const lp = new LoginRouteParam();
    lp.reason = 'action_required';
    lp.action = '查看我的瞬间';
    lp.redirectUrl = PageNames.MOMENT;
    AppRouter.push(PageNames.LOGIN, lp);
  }

  private previewImage(momentId: string, images: string[], index: number) {
    const previewParams = new PhotoPreviewRouteParam();
    previewParams.momentId = momentId;
    previewParams.images = images;
    previewParams.index = index;
    AppRouter.push(PageNames.PHOTO_PREVIEW, previewParams);
  }

  private getTabCountLabel(): string {
    if (this.activeTab === TAB_LIKED) {
      return '最近收藏的灵感';
    }
    if (this.activeTab === TAB_SAVED) {
      return '可随时继续整理';
    }
    return `共 ${this.posts.length} 则记录`;
  }

  private getEmptyTitle(): string {
    if (this.activeTab === TAB_LIKED) {
      return '还没有喜欢的瞬间';
    }
    if (this.activeTab === TAB_SAVED) {
      return '还没有收藏的瞬间';
    }
    return '这里还没有瞬间';
  }

  private getEmptyMessage(): string {
    if (this.activeTab === TAB_LIKED) {
      return '在附近遇到心动的瞬间，点亮爱心后会出现在这里。';
    }
    if (this.activeTab === TAB_SAVED) {
      return '看到想留住的内容时，点击收藏即可在这里继续整理。';
    }
    return '从一次路过、一次相遇开始记录。';
  }

  build() {
    NavDestination() {
      PageShellContainer({ variant: 'default', currentBreakpoint: this.currentBreakpoint }) {
        Column() {
          SystemPageHeader({
            title: '我的瞬间',
            titleSize: DesignConstants.FONT.SIZE.LG_PLUS,
            titleWeight: DesignConstants.FONT.WEIGHT.SEMIBOLD,
            backgroundVariant: 'transparent',
            showBackButton: true,
            onBack: () => { AppRouter.pop(); },
            hasBelowContent: true
          }) {
            SystemButton({
              label: '发布',
              variant: 'primary',
              buttonSize: 'small',
              pill: true,
              disabled: this.isLoading,
              onPress: () => { this.openComposer(); }
            })
          }

          Row({ space: DesignConstants.SPACING.MD }) {
            Scroll() {
              TabsPillGroup({
                variant: 'pills',
                freeWidth: true,
                tabs: [
                  { label: '发布', count: this.posts.length },
                  { label: '喜欢', count: this.liked.length },
                  { label: '收藏', count: this.saved.length }
                ] as TabItem[],
                activeIndex: this.activeTab,
                actions: {
                  onChange: (index: number) => { this.switchTab(index); }
                }
              })
            }
            .scrollable(ScrollDirection.Horizontal)
            .scrollBar(BarState.Off)
            .align(Alignment.Start)
            .layoutWeight(1)

            if (!this.isLoading && this.isLoggedIn && this.errorMessage.length === 0) {
              Text(this.getTabCountLabel())
                .fontSize(DesignConstants.FONT.SIZE.XS)
                .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS))
                .fontColor(this.getCurrentThemePalette().textSecondary)
                .maxLines(1)
                .flexShrink(0)
            }
          }
          .width('100%')
          .alignItems(VerticalAlign.Center)
          .padding({
            top: DesignConstants.SPACING.SM,
            bottom: DesignConstants.SPACING.MD
          })

          if (this.isLoading) {
            this.buildSkeletonGrid()
          } else if (!this.isLoggedIn) {
            EmptyState({
              spotlight: true,
              icon: AppIcon.PERSON,
              title: '登录后查看',
              message: '登录后即可查看你的瞬间与收藏',
              actionLabel: '去登录',
              onAction: () => { this.openLogin(); }
            })
              .layoutWeight(1)
          } else if (this.errorMessage.length > 0 && this.activeItems().length === 0) {
            EmptyState({
              bare: true,
              icon: AppIcon.WARNING,
              message: this.errorMessage,
              actionLabel: '重试',
              onAction: () => { this.loadTab(this.activeTab, true); }
            })
              .layoutWeight(1)
          } else if (this.activeItems().length === 0) {
            EmptyState({
              spotlight: true,
              icon: this.activeTab === TAB_LIKED ? AppIcon.HEART : (this.activeTab === TAB_SAVED ? AppIcon.BOOKMARK : AppIcon.PHOTO),
              title: this.getEmptyTitle(),
              message: this.getEmptyMessage(),
              actionLabel: this.activeTab === TAB_POSTS ? '发布第一条' : '',
              onAction: () => { if (this.activeTab === TAB_POSTS) { this.openComposer(); } }
            })
              .layoutWeight(1)
          } else {
            Refresh({ refreshing: $$this.isRefreshing, builder: RefreshIndicator }) {
              Column() {
                this.buildMomentGrid()
                if (this.isLoadingMore) {
                  LoadingMoreIndicator()
                } else if (!this.activeHasMore()) {
                  ListEnd()
                }
              }
              .width('100%')
              .height('100%')
            }
            .layoutWeight(1)
            .pullToRefresh(false)
            .refreshOffset(64)
            .onRefreshing(() => { this.refreshActive(); })
          }
        }
        .width('100%')
        .height('100%')
      }
    }
    .hideTitleBar(true)
  }

  @Builder
  private buildSkeletonGrid() {
    // 与真实 2 列卡同栅格（骨架→内容零位移）。用行布局而非 Grid：
    // SkeletonShimmer 的外层 Column 不锁定高度，子 Grid 的 layoutWeight 会塌成 0。
    Column() {
      SkeletonShimmer({ active: true }) {
        Column({ space: DesignConstants.SPACING.MD2 }) {
          ForEach([0, 1, 2], (row: number) => {
            Row({ space: DesignConstants.SPACING.MD2 }) {
              ForEach([0, 1], (col: number) => {
                Column({ space: DesignConstants.SPACING.SM }) {
                  Skeleton({
                    blockWidth: '100%',
                    blockHeight: DesignConstants.COMPONENT.SKELETON_MEDIA_HEIGHT,
                    cornerRadius: DesignConstants.SHAPE.CARD
                  })
                  Skeleton({
                    blockWidth: '72%',
                    blockHeight: DesignConstants.FONT.SIZE.SM,
                    cornerRadius: DesignConstants.RADIUS.SM
                  })
                }
                .layoutWeight(1)
                .alignItems(HorizontalAlign.Start)
              }, (col: number) => `my-moment-skeleton-${row}-${col}`)
            }
            .width('100%')
            .alignItems(VerticalAlign.Top)
          }, (row: number) => `my-moment-skeleton-row-${row}`)
        }
        .width('100%')
        .padding({ top: DesignConstants.SPACING.SM })
        .accessibilityText('正在加载我的瞬间')
      }
    }
    .layoutWeight(1)
    .width('100%')
  }


  @Builder
  private buildMomentGrid() {
    WaterFlow() {
      LazyForEach(this.momentsDataSource, (moment: MomentItem, index: number) => {
        FlowItem() {
          MomentFeedCard({
            moment: moment,
            variant: 'overlay',
            overlayAspectRatio: 0.75,
            priority: index < 2,
            onImagePreview: (momentId: string, images: string[], imageIndex: number) => {
              this.previewImage(momentId, images, imageIndex);
            }
          })
        }
      }, (moment: MomentItem) => moment.moment_id)
    }
    .columnsTemplate('1fr 1fr')
    .columnsGap(DesignConstants.SPACING.MD2)
    .rowsGap(DesignConstants.SPACING.MD2)
    .width('100%')
    .layoutWeight(1)
    .cachedCount(4)
    .edgeEffect(EdgeEffect.None)
    .scrollBar(BarState.Off)
    .padding({
      bottom: SafeAreaUtils.resolveBottomPadding(this.safeAreaBottom, DesignConstants.SPACING.LG, 0)
    })
    .onReachEnd(() => { this.loadMore(); })
  }
}

@Builder
export function MomentPageBuilder() {
  MomentPage()
}
