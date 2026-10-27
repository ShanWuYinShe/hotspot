/**
 * 附近瞬间页面（术语与 web 统一：spec 权威术语为「生活瞬间」；原「附近动态」已改名）
 * 展示热门话题、推荐用户、热门瞬间
 */

import { GlobalStateManager, KEY_CURRENT_LOCATION, KEY_BROWSE_ANCHOR, KEY_IS_LOGGED_IN } from '../../../core/state/GlobalStateManager';
import { PageShellContainer } from '../../../components/PageShellContainer';
import type { BreakpointType } from '../../../core/responsive/Breakpoint';
import { useMasterDetail, isDesktopForm } from '../../../core/responsive/Breakpoint';
import { LocationInfo } from '../../../core/location/LocationUtil';
import { ApiClient } from '../../../core/api/ApiClient';
import { getErrorMessage } from '../../../core/network/ApiErrorUtil';
import { AppRouter, ChatRouteParam, GroupDetailRouteParam, LoginRouteParam, MomentDetailRouteParam, PhotoPreviewRouteParam, ProfileRouteParam } from '../../../core/router/AppRouter';
import { PageNames, DefaultConfig } from '../../../constants/AppConstants';
import { interactionActions } from '../../../core/state/InteractionActions';
import { InteractionStateStore, UserRelationState } from '../../../core/state/InteractionStateStore';
import {
  MomentItem,
  GeoPoint, PaginationParams,
  FetchNearbyMomentsRequest,
  ApiResponse, PaginatedData
} from '../../../core/types/api';
import { curves } from '@kit.ArkUI';
import { LocationUtil } from '../../../core/location/LocationUtil';
import { SystemAvatar } from '../../../components/SystemAvatar';
import { SystemCard, CardVariant } from '../../../components/SystemCard';
import { SystemLoadingView } from '../../../components/SystemLoadingView';
import { SystemToast } from '../../../components/SystemToast';
import { SystemPageHeader } from '../../../components/SystemPageHeader';
import { TabsPillGroup, TabItem } from '../../../components/TabsPillGroup';
import { DesignConstants, textLineHeight } from '../../../constants/DesignConstants';
import { MotionUtil } from '../../../core/accessibility/MotionUtil';
import { BackToTopButton } from '../../../components/BackToTopButton';
import { ListEnd } from '../../../components/ListEnd';
import { AppIcon } from '../../../components/SystemIcon';
import { MomentFeedCard } from '../../../components/MomentFeedCard';
import { MomentWallCard } from '../components/MomentWallCard';
import { LoadingUtil } from '../../../core/utils/LoadingUtil';
import { BasicDataSource } from '../../../core/utils/BasicDataSource';
import { SystemUiController } from '../../../core/utils/SystemUiController';
import { SafeAreaUtils } from '../../../core/utils/SafeAreaUtils';
import { EmptyState, RefreshIndicator, LoadingMoreIndicator } from '../../../components/EmptyState';
import { SystemButton } from '../../../components/SystemButton';
import { hilog } from '@kit.PerformanceAnalysisKit';
import pasteboard from '@ohos.pasteboard';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { DEFAULT_RESOLVED_THEME_ID, KEY_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';
import { LucideIcon } from '../../../components/LucideIcon';

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
  @StorageLink(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = DEFAULT_RESOLVED_THEME_ID;
  @StorageLink('interaction_user_state') userStateMap: Record<string, UserRelationState> = {};
  @StorageProp('isTablet') isTablet: boolean = false;
  @StorageProp('currentBreakpoint') currentBreakpoint: BreakpointType = 'sm';
  /** 作为路由页 push 时显示返回按钮（tab 场景隐藏，对齐前端 /nearby/moments 独立页） */
  @Prop showBackButton: boolean = false;

  @State isLoading: boolean = true;
  @State errorMessage: string = '';
  @State hotMoments: MomentItem[] = [];
  private hotMomentsDataSource: BasicDataSource<MomentItem> = new BasicDataSource<MomentItem>([]);
  private interactionStore = InteractionStateStore.getInstance();
  private isDestroyed: boolean = false;
  @State isRefreshing: boolean = false;
  /** 触底加载更多状态（cursor 分页，照 NearbyUsersListPage 模式） */
  @State isLoadingMore: boolean = false;
  @State hasMore: boolean = false;
  @State nextCursor: string = '';
  @State activeTag: string = '';
  /** R7 内容进场（只包内容区；首屏为导航进入，播放） */
  @State private enterOpacity: number = 1;
  @State private enterOffset: number = 0;
  @State private showBackToTop: boolean = false;
  @State private viewportHeight: number = 0;
  private backToTopScroller: Scroller = new Scroller();
  private enterPlayed: boolean = false;
  /** 动态 tag 列表（从前 N 条 moment.tags 聚合去重，对齐前端 moments tagsList 派生模式） */
  @State hotTags: string[] = [];

  // 2026-10-19 信息架构调整（与 web 同口径）：本页是**地理叶子页**，只承担「周边」流——
  // 分类切换器、「热门/关注」数据分支、推荐用户/群组与桌面右栏均已移除；
  // 「热门」（算法轴）与「关注」（关系轴）归首页两个段（首页「发现」/「关注」）。
  // 标签药片（hotTags）保留为筛选维度。

  /** 位置行副标题（对齐前端 nearby/moments subtitle：{locationName} · 5km 内 / 5km 内） */
  private getLocationSubtitle(): string {
    const name = (this.browseAnchor ?? this.currentLocation)?.locationName ?? '';
    return name.length > 0 ? `${name} · 5km 内` : '5km 内';
  }

  /** 标签药片客户端过滤：activeTag 为空（未选/再次点击取消）时恢复全部 */
  private getVisibleMoments(): MomentItem[] {
    if (!this.activeTag) {
      return this.hotMoments;
    }
    const tag = this.activeTag;
    return this.hotMoments.filter((moment: MomentItem) => (moment.tags ?? []).includes(tag));
  }

  /** 动态聚合 tag 药片（从前 N 条 moment.tags 去重 + 取前 8；对齐前端 tagsList 派生） */
  private deriveHotTags(): void {
    const collected: Set<string> = new Set();
    for (const moment of this.hotMoments) {
      for (const tag of moment.tags ?? []) {
        if (tag && tag.length > 0 && tag.length <= 12) {
          collected.add(tag);
          if (collected.size >= 8) break;
        }
      }
      if (collected.size >= 8) break;
    }
    this.hotTags = Array.from(collected);
    // 当前选中的 tag 已不在新列表 → 清空，恢复全部
    if (this.activeTag && !this.hotTags.includes(this.activeTag)) {
      this.activeTag = '';
    }
  }

  /** 同步过滤到 LazyForEach 数据源（activeTag 变更/数据加载后调用，保持虚拟化列表与筛选一致） */
  private refreshVisibleDataSource(): void {
    this.hotMomentsDataSource.setData(this.getVisibleMoments());
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
    return this.hotMoments.length > 0;
  }

  /** 唯一数据源：附近瞬间（moments/fetch_nearby，距离序；reference_point 可选） */
  private async loadNearbyFeed(): Promise<void> {
    const version = this.loadVersion;
    const refLoc: LocationInfo | null = this.browseAnchor ?? this.currentLocation;
    // 契约：reference_point 可选，无定位时不传（服务端按回退链解析），不再用 0,0 兜底
    const refPoint: GeoPoint | undefined = refLoc ? { latitude: refLoc.latitude, longitude: refLoc.longitude } : undefined;
    const request: FetchNearbyMomentsRequest = {
      reference_point: refPoint,
      radius: DefaultConfig.NEARBY_RADIUS,
      pagination: { limit: 20 }
    };
    const response = await LoadingUtil.withMinimumTime(() => ApiClient.fetchNearbyMoments(request));
    if (this.isDestroyed || version !== this.loadVersion) return;
    if (response.code === 0) {
      this.hotMoments = response.data?.items ?? [];
      this.storeCursor(response);
      this.deriveHotTags();
      this.hotMomentsDataSource.setData(this.getVisibleMoments());
      this.hotMoments.forEach((item: MomentItem) => this.interactionStore.patchMomentFromItem(item));
    } else if (!this.hasAnyData()) {
      this.errorMessage = getErrorMessage(response, '加载附近瞬间失败');
    }
  }

  /** 记录分页游标（spec：next_cursor 恒在，null = 没有更多） */
  private storeCursor(response: ApiResponse<PaginatedData<MomentItem>>): void {
    this.hasMore = !!(response.data?.next_cursor);
    this.nextCursor = response.data?.next_cursor ?? '';
  }

  /** 触底加载更多（cursor 分页）：按当前 tab 走同源接口，追加并同步 tag 过滤与互动态 */
  private async loadMoreMoments(): Promise<void> {
    if (this.isLoadingMore || !this.hasMore || this.isLoading) return;
    const version = this.loadVersion;
    this.isLoadingMore = true;
    try {
      const refLoc: LocationInfo | null = this.browseAnchor ?? this.currentLocation;
      const refPoint: GeoPoint | undefined = refLoc ? { latitude: refLoc.latitude, longitude: refLoc.longitude } : undefined;
      const pagination: PaginationParams = { limit: 20, cursor: this.nextCursor };
      const request: FetchNearbyMomentsRequest = {
        reference_point: refPoint,
        radius: DefaultConfig.NEARBY_RADIUS,
        pagination: pagination
      };
      const response: ApiResponse<PaginatedData<MomentItem>> = await ApiClient.fetchNearbyMoments(request);
      if (this.isDestroyed || version !== this.loadVersion) return;
      if (response.code === 0) {
        // 按 moment_id 去重后追加，避免游标边界重复项打崩 LazyForEach 键
        const known = new Set<string>(this.hotMoments.map((m: MomentItem) => m.moment_id));
        const fresh = (response.data?.items ?? []).filter((m: MomentItem) => m.moment_id && !known.has(m.moment_id));
        this.hotMoments = this.hotMoments.concat(fresh);
        fresh.forEach((item: MomentItem) => this.interactionStore.patchMomentFromItem(item));
        this.storeCursor(response);
        this.deriveHotTags();
        this.refreshVisibleDataSource();
      }
    } catch {
      // 静默失败：保持游标不变，用户再次触底即可重试
    } finally {
      this.isLoadingMore = false;
    }
  }

  private getMomentsSectionTitle(): string {
    // 本页只承担地理流（原先随 tab 切换三套标题，tab 已移除）
    return '周边动态';
  }

  /** 数据加载版本号：快速切 tab 时旧响应不覆盖新 tab 内容 */
  private loadVersion: number = 0;

  private async loadData(refresh: boolean = false): Promise<void> {
    const version = ++this.loadVersion;
    if (!refresh) { this.isLoading = true; }
    this.errorMessage = '';
    // 首屏/刷新重置游标，避免残留上一轮 hasMore 误触发触底加载
    this.hasMore = false;
    this.nextCursor = '';
    try {
      await this.loadNearbyFeed();
    } catch {
      hilog.error(DOMAIN_NUMBER, TAG, '加载附近瞬间失败');
      if (!this.hasAnyData()) {
        this.errorMessage = '加载附近瞬间失败';
      }
    } finally {
      this.isLoading = false;
      this.isRefreshing = false;
      if (this.hasAnyData()) {
        this.resetContentEnter();
        this.playContentEnter();
      }
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
  NavDestination() {
  PageShellContainer({ variant: 'feed-medium', currentBreakpoint: this.currentBreakpoint}) {
    Column() {
      // 粘性头部：标题(走 SystemPageHeader) + 搜索栏 + 标签栏（对齐前端 nearby/moments 页 headerContent）
      Column() {
        // 标题行 —— 统一走 SystemPageHeader（标题 24pt + 位置行副标题 + 内容数 trailing，
        // 对齐前端 nearby/moments PageHeader：titleSize 2xl + MapPin 图标 + {位置} · 5km 内）
        SystemPageHeader({
          title: '附近瞬间',
          subtitle: this.getLocationSubtitle(),
          subtitleIcon: AppIcon.LOCATION,
          titleSize: DesignConstants.FONT.SIZE.XXL,
          titleWeight: DesignConstants.FONT.WEIGHT.MEDIUM,
          backgroundVariant: 'transparent',
          showBackButton: this.showBackButton,
          onBack: this.showBackButton ? () => { AppRouter.pop(); } : null
        }) {
          if (this.getVisibleMoments().length > 0) {
            // 已加载数量（cursor 分页）：还有更多时以「+」标注，计数诚实（审查 P2）
            Text(this.hasMore ?
              `${this.getVisibleMoments().length}+ 条内容` :
              `${this.getVisibleMoments().length} 条内容`)
              .fontSize(DesignConstants.FONT.SIZE.SM)
              .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.SM))
              .fontColor(this.getCurrentThemePalette().textSecondary)
          }
        }

        // 搜索栏（对齐前端：search icon + input + clear button）
        Row({ space: DesignConstants.SPACING.MD }) {
          LucideIcon({
              icon: AppIcon.SEARCH,
              iconSize: DesignConstants.COMPONENT.ICON_SIZE_2XS,
              color: this.getCurrentThemePalette().textSecondary
            })
          Text('搜索用户、动态、地点或话题')
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.SM))
            .fontColor(this.getCurrentThemePalette().textSecondary)
            .layoutWeight(1)
            .maxLines(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
        }
        .height(DesignConstants.COMPONENT.INPUT_HEIGHT)
        .backgroundColor(this.getCurrentThemePalette().inputBackgroundColor)
        .borderRadius(DesignConstants.SHAPE.INPUT)
        .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getSurfaceBorder() })
        .onClick(() => { AppRouter.push(PageNames.SEARCH); })
        .margin({ top: DesignConstants.SPACING.SM, bottom: DesignConstants.SPACING.XS })

        // 动态 tag 药片（对齐前端 nearby/moments tags 药片栏；hotTags 由 hotMoments 派生）
        if (this.hotTags.length > 0) {
          TabsPillGroup({
            variant: 'pills',
            tabs: this.hotTags.map((tag: string): TabItem => ({ label: tag })),
            activeIndex: this.activeTag ? Math.max(0, this.hotTags.indexOf(this.activeTag)) : -1,
            actions: {
              onChange: (index: number) => {
                const tag = this.hotTags[index];
                // 再次点击已激活项可取消筛选（对齐前端 activeTag === id ? null : id）
                this.activeTag = this.activeTag === tag ? '' : (tag ?? '');
                this.refreshVisibleDataSource();
                // 内容替换（标签筛选）：重播进场
                this.resetContentEnter();
                this.playContentEnter();
              }
            }
          })
            .margin({ top: DesignConstants.SPACING.SM })
        }
      }
      .width('100%')
      .padding({ bottom: 0 })
      .backgroundColor(this.getCurrentThemePalette().pageBackground)
      .border({ width: { bottom: DesignConstants.BORDER_WIDTH.HAIRLINE }, color: this.getSurfaceBorder() })

      // 内容区
      if (this.isLoading) {
        SystemLoadingView({ message: '正在加载附近瞬间…' })
          .layoutWeight(1)
          .transition(TransitionEffect.OPACITY.animation({ duration: DesignConstants.ANIMATION.DURATION_SHEET, curve: DesignConstants.ANIMATION.CURVE_FAST_OUT_SLOW_IN })
            .combine(TransitionEffect.move(TransitionEdge.BOTTOM).animation({ duration: DesignConstants.ANIMATION.DURATION_SHEET, curve: DesignConstants.ANIMATION.CURVE_FAST_OUT_SLOW_IN })))
      } else if (!this.hasAnyData() && this.errorMessage) {
        EmptyState({
          bare: true,
          icon: AppIcon.WARNING,
          message: this.errorMessage,
          actionLabel: '重试',
          onAction: () => { this.loadData(); }
        })
        .layoutWeight(1)
        .transition(TransitionEffect.OPACITY.animation({ duration: DesignConstants.ANIMATION.DURATION_SHEET, curve: DesignConstants.ANIMATION.CURVE_FAST_OUT_SLOW_IN })
          .combine(TransitionEffect.move(TransitionEdge.BOTTOM).animation({ duration: DesignConstants.ANIMATION.DURATION_SHEET, curve: DesignConstants.ANIMATION.CURVE_FAST_OUT_SLOW_IN })))
      } else {
        // 下拉刷新由 buildMomentFeed 内包 WaterFlow 的 Refresh 承担。
        // 单栏瀑布墙（宽屏与移动同款）：右栏「活跃用户/热门话题」随「热门」tab 移除——
        // 推荐用户/群组属发现语义，归首页「发现」段与附近枢纽；本页只做地理流。
        // 注意：这里**不能**用裸 `{ }` 块包裹（2026-10-19 端上实测缺陷）：ArkUI 的块不是
        // 容器，内部的 layoutWeight(1) 不会生效，Column 高度塌陷 → WaterFlow 高度为 0 →
        // 页头计数正常但一条内容都不渲染（表现为"内容区全空"）。
        Column() {
          this.buildMomentFeed()
        }
        .width('100%')
        .layoutWeight(1)
      }
    }
    .width('100%')
    .height('100%')
    }
  }
  .hideTitleBar(true)
  }

  // ==================== 动态瀑布墙（对齐前端 BSP 卡片墙：移动/平板 2 列，桌面 3 列） ====================
  // tag 胶囊条固定在瀑布上方（不随流滚动）以保留 WaterFlow 虚拟化；无首卡 hero，全部项进统一墙。
  @Builder
  buildMomentFeed() {
    Column() {
      // section 标题（对齐前端：eyebrow 小标 + 主标题层级）
      Row() {
        Text(this.getMomentsSectionTitle())
          .fontSize(DesignConstants.FONT.SIZE.LG_PLUS) // 18，对齐前端 page-title
          .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.LG_PLUS))
          .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
          .fontColor(this.getCurrentThemePalette().textPrimary)
      }
      .width('100%')
      .padding({
        top: DesignConstants.SPACING.LG,
        bottom: DesignConstants.SPACING.SM
      })

      // R7 内容进场（只包内容区，不含粘性头）
      Column() {
      if (this.getVisibleMoments().length > 0) {
        // Refresh 直接包可滚动的 WaterFlow（子组件需为滚动组件下拉刷新才生效）
        Refresh({ refreshing: $$this.isRefreshing, builder: RefreshIndicator }) {
          Column() {
            WaterFlow({ scroller: this.backToTopScroller }) {
              // LazyForEach 虚拟化（审查 P1-9：替换全量 ForEach，长列表滚动不掉帧）；
              // 数据源由 refreshVisibleDataSource 保持与标签过滤同步
              LazyForEach(this.hotMomentsDataSource, (moment: MomentItem, index: number) => {
                FlowItem() {
                  MomentWallCard({
                    moment: moment,
                    // R6 首屏优先：前 2 张高清档（LazyForEach 离屏不创建）
                    priority: index < 2,
                    // 2 列窄墙用 3:4 竖版（对齐 web 移动端）；3 列用 4:3
                    cardAspectRatio: this.getWallColumns() === 2 ? 0.75 : 1.333,
                    onOpenDetail: (item: MomentItem) => this.openMomentDetail(item),
                    onImagePreview: (momentId: string, images: string[], index: number) => {
                      this.previewImage(momentId, images, index);
                    }
                  })
                }
              }, (moment: MomentItem) => moment.moment_id)
            }
            .width('100%')
            .layoutWeight(1)
            .columnsTemplate(this.getWallColumns() === 3 ? '1fr 1fr 1fr' : '1fr 1fr')
            .columnsGap(DesignConstants.SPACING.LG)
            .rowsGap(DesignConstants.SPACING.LG)
            .padding({
              bottom: SafeAreaUtils.resolveBottomPadding(this.safeAreaBottom, DesignConstants.SPACING.LG, 0)
            })
            .cachedCount(4)
            .edgeEffect(EdgeEffect.None)
            .scrollBar(BarState.Off)
            // 触底加载更多（cursor 分页，P2-3：突破首屏 20 条封顶）
            .onReachEnd(() => { this.loadMoreMoments(); })
            .onDidScroll(() => { this.updateBackToTop(); })
            .onAreaChange((oldArea: Area, newArea: Area) => { this.viewportHeight = Number(newArea.height); })

            // 分页状态条（WaterFlow 外常驻，规避 FlowItem 跨列占位；照 NearbyUsersListPage 模式）
            if (this.hasMore) {
              LoadingMoreIndicator()
            } else {
              ListEnd()
            }
          }
          .width('100%')
          .height('100%')
        }
        .width('100%')
        .layoutWeight(1)
        .pullToRefresh(false)
        .refreshOffset(64)
        .onRefreshing(() => { this.loadData(true); })
      } else {
        // 空态（对齐前端 EmptyState：图标 + 文案）
        if (this.activeTag) {
          // 标签过滤后为空：提供「查看全部」退出过滤
          EmptyState({
            bare: true,
            icon: AppIcon.RECOMMEND,
            message: `「${this.activeTag}」标签下暂时没有动态`,
            actionLabel: '查看全部',
            onAction: () => { this.activeTag = ''; }
          })
            .layoutWeight(1)
        } else {
          EmptyState({
            bare: true,
            icon: AppIcon.RECOMMEND,
            message: '附近暂时没有瞬间',
            // 「刷新」对"这片区域暂时没人发"帮助有限；直接给发布入口，对齐 web 空态的引导
            // 文案「去附近看看，或者发布第一条动态」（2026-10-21 端上走查）。
            actionLabel: '发布瞬间',
            onAction: () => { AppRouter.push(PageNames.CREATE_MOMENT); }
          })
            .layoutWeight(1)
        }
      }
      }
      .width('100%')
      .layoutWeight(1)
      .opacity(this.enterOpacity)
      .translate({ y: this.enterOffset })
    }
    .overlay(this.buildBackToTopOverlay(), { align: Alignment.BottomEnd })
    .width('100%')
    .height('100%')
  }

  private getWallColumns(): number {
    // 对齐前端 /nearby/moments gridColumns：mobile 2 / tablet 2 / desktop 3
    return isDesktopForm(this.currentBreakpoint) ? 3 : 2;
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
}

@Builder
export function MomentsFeedPageBuilder() {
  MomentsFeedPage({ showBackButton: true })
}