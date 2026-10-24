/**
 * HomePage — 纯内容流（2026-10-20，对齐 web 首页）
 *
 * 用户审查结论：「不需要标题块，直接就显示内容；没必要让用户专门去关注当前看的内容是什么」。
 * 因此本页**没有标题、没有 tab、没有分组小标题**，就是一条流：
 *   - 关系流（moments/fetch_following，需登录）在前 —— 与你最相关的内容优先；
 *   - 发现流（moments/search 热门，匿名可用）续后，按 moment_id 去重；
 *   - 附近用户/群组及其关注、入群交互由附近枢纽与叶子页承担。
 * 视觉上无标题，但 ArkUI 的页面语义由路由与 accessibilityText 承担（不做伪 h1）。
 */

import { PageShellContainer } from '../../../components/PageShellContainer';
import type { BreakpointType } from '../../../core/responsive/Breakpoint';
import { ApiClient } from '../../../core/api/ApiClient';
import { AppRouter, MomentDetailRouteParam } from '../../../core/router/AppRouter';
import { PageNames } from '../../../constants/AppConstants';
import { MomentItem, SearchMomentsRequest, FetchFollowingMomentsRequest } from '../../../core/types/api';
import { SurfaceIconButton } from '../../../components/SurfaceIconButton';
import { SystemLoadingView } from '../../../components/SystemLoadingView';
import { DesignConstants, textLineHeight } from '../../../constants/DesignConstants';
import { AppIcon } from '../../../components/SystemIcon';
import { LucideIcon } from '../../../components/LucideIcon';
import { EmptyState, RefreshIndicator } from '../../../components/EmptyState';
import { Pressable } from '../../../components/Pressable';
import { LoadingUtil } from '../../../core/utils/LoadingUtil';
import { SystemUiController } from '../../../core/utils/SystemUiController';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { AppThemeVisuals } from '../../../core/theme/AppThemeVisuals';
import { DEFAULT_RESOLVED_THEME_ID, KEY_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';
import { getErrorMessage } from '../../../core/network/ApiErrorUtil';
import { curves } from '@kit.ArkUI';
import { MomentFeedCard } from '../../../components/MomentFeedCard';
import { TabsPillGroup, TabItem } from '../../../components/TabsPillGroup';

const DOMAIN_NUMBER: number = 0xFF10;
const TAG: string = 'HomePage';

/** 每段首屏条数（对齐 web PAGE_SIZE=20 的首屏语义；本端暂不做无限续载） */
const FEED_PAGE_SIZE: number = 20;

@Preview
@Component
export struct HomePage {
  @StorageLink('isLoggedIn') isLoggedIn: boolean = false;
  @StorageLink('safeAreaTop') safeAreaTop: number = 0;
  @StorageLink('safeAreaBottom') safeAreaBottom: number = 0;
  @StorageLink(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = DEFAULT_RESOLVED_THEME_ID;
  @StorageProp('currentBreakpoint') currentBreakpoint: BreakpointType = 'sm';

  /** 发现流（热门，匿名可用） */
  @Watch('onMomentsChanged')
  @State moments: MomentItem[] = [];
  /** 关系流（我关注的人，需登录） */
  @Watch('onMomentsChanged')
  @State followingMoments: MomentItem[] = [];
  @State private cachedDisplayedMoments: MomentItem[] = [];
  @State momentsLoading: boolean = true;
  @State followingLoading: boolean = false;
  @State momentsError: string = '';
  @State followingError: string = '';

  /** 分段流：0 = 为你推荐, 1 = 正在关注 */
  @State private activeTab: number = 0;
  private readonly feedTabs: TabItem[] = [
    { label: '为你推荐' },
    { label: '正在关注' }
  ];

  /** 「查看全部」链接的按下态索引（-1 = 无；对齐前端 link 按压反馈） */
  @State viewAllPressedIndex: number = -1;
  @State isRefreshing: boolean = false;

  private isDestroyed: boolean = false;

  private onMomentsChanged(): void {
    const merged: MomentItem[] = [];
    const seen: Set<string> = new Set<string>();
    for (const m of this.followingMoments) {
      if (seen.has(m.moment_id)) continue;
      seen.add(m.moment_id);
      merged.push(m);
    }
    for (const m of this.moments) {
      if (seen.has(m.moment_id)) continue;
      seen.add(m.moment_id);
      merged.push(m);
    }
    this.cachedDisplayedMoments = merged;
  }

  private currentTabMoments(): MomentItem[] {
    if (this.activeTab === 1) {
      return this.followingMoments;
    }
    return this.moments;
  }

  private isTabLoading(): boolean {
    if (this.activeTab === 1) {
      return this.followingLoading;
    }
    return this.momentsLoading;
  }

  private currentTabError(): string {
    if (this.activeTab === 1) {
      return this.followingError;
    }
    return this.momentsError;
  }

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  async aboutToAppear() {
    SystemUiController.syncWithCurrentTheme();
    this.onMomentsChanged();
    // 一条流 = 关系流 + 发现流：两者并行加载，渲染时关系流在前、发现流去重续后
    this.loadDiscoverMoments();
    if (this.isLoggedIn) {
      this.loadFollowingMoments();
    }
  }

  aboutToDisappear() {
    this.isDestroyed = true;
  }

  // ==================== 数据加载 ====================

  /** 发现段：moments/search 热门模式（spec：承接发现页首屏；匿名可用） */
  private async loadDiscoverMoments(): Promise<void> {
    this.momentsLoading = true;
    this.momentsError = '';
    try {
      const request: SearchMomentsRequest = {
        sort_by: 'popular',
        pagination: { limit: FEED_PAGE_SIZE }
      };
      const response = await LoadingUtil.withMinimumTime(() => ApiClient.searchMoments(request));
      if (this.isDestroyed) return;
      if (response.code === 0) {
        this.moments = (response.data?.items ?? []).filter((m: MomentItem) => m.moment_id && m.user);
      } else {
        this.momentsError = getErrorMessage(response, '加载失败');
      }
    } catch {
      this.momentsError = '加载热门内容失败';
    } finally {
      if (!this.isDestroyed) this.momentsLoading = false;
    }
  }

  /** 关注段：moments/fetch_following（无匿名版，未登录走登录引导） */
  private async loadFollowingMoments(): Promise<void> {
    this.followingLoading = true;
    this.followingError = '';
    try {
      const request: FetchFollowingMomentsRequest = { pagination: { limit: FEED_PAGE_SIZE } };
      const response = await LoadingUtil.withMinimumTime(() => ApiClient.fetchFollowingMoments(request));
      if (this.isDestroyed) return;
      if (response.code === 0) {
        this.followingMoments = (response.data?.items ?? []).filter((m: MomentItem) => m.moment_id && m.user);
      } else {
        this.followingError = getErrorMessage(response, '加载失败');
      }
    } catch {
      this.followingError = '加载关注动态失败';
    } finally {
      if (!this.isDestroyed) this.followingLoading = false;
    }
  }

  private openMomentDetail(moment: MomentItem): void {
    const params = new MomentDetailRouteParam();
    params.momentId = moment.moment_id;
    AppRouter.push(PageNames.MOMENT_DETAIL, params);
  }

  private feedLoading(): boolean {
    return this.momentsLoading || this.followingLoading;
  }

  private feedError(): string {
    return this.momentsError.length > 0 ? this.momentsError : this.followingError;
  }

  private reloadFeed(): void {
    this.loadDiscoverMoments();
    if (this.isLoggedIn) {
      this.loadFollowingMoments();
    }
  }

  private async refreshHome(): Promise<void> {
    try {
      await this.loadDiscoverMoments();
      if (this.isLoggedIn) {
        await this.loadFollowingMoments();
      }
    } finally {
      this.isRefreshing = false;
    }
  }

  // ==================== UI ====================

  build() {
    // 横向留白由容器 gutter 统一提供（页面不再自加左右 padding）
    PageShellContainer({ variant: 'default', currentBreakpoint: this.currentBreakpoint }) {
      Column() {
        this.buildStickyHeader()

        // 分段切换：为你推荐 / 正在关注
        Row() {
          TabsPillGroup({
            tabs: this.feedTabs,
            activeIndex: this.activeTab,
            freeWidth: true,
            actions: {
              onChange: (index: number) => {
                this.activeTab = index;
              }
            }
          })
        }
        .width('100%')
        .padding({ bottom: DesignConstants.SPACING.XS })

        // 整页下拉刷新：Refresh 包内容区（sticky header 保持粘性在外）
        // 关键配置：
        // 1. builder: RefreshIndicator 保证全周期使用同一固定高度指示器（无跳变）
        // 2. pullToRefresh(false) 强制松手才刷新，彻底消除下拉位移超过阈值时系统吸附回弹与手指拖拽的冲突（鬼畜晃动根因）
        // 3. refreshOffset(64) 明确触发阈值
        Refresh({
          refreshing: $$this.isRefreshing,
          builder: RefreshIndicator
        }) {
          if (this.activeTab === 1 && !this.isLoggedIn) {
            Scroll() {
              this.buildLoginPrompt()
            }
            .width('100%')
            .height('100%')
            .align(Alignment.Top)
            .scrollBar(BarState.Off)
          } else if (this.isTabLoading() && this.currentTabMoments().length === 0) {
            Column() {
              SystemLoadingView({ message: '正在加载…' })
                .width('100%')
            }
            .width('100%')
            .height('100%')
            .justifyContent(FlexAlign.Center)
          } else if (this.currentTabError().length > 0 && this.currentTabMoments().length === 0) {
            Scroll() {
              EmptyState({
                icon: AppIcon.WARNING,
                title: '内容加载失败',
                message: this.currentTabError(),
                bare: true,
                actionLabel: '重试',
                onAction: () => { this.reloadFeed(); }
              })
              .padding({ top: DesignConstants.SPACING.XXL })
            }
            .width('100%')
            .height('100%')
            .align(Alignment.Top)
            .scrollBar(BarState.Off)
          } else if (this.currentTabMoments().length === 0) {
            Scroll() {
              this.buildEmptyState()
            }
            .width('100%')
            .height('100%')
            .align(Alignment.Top)
            .scrollBar(BarState.Off)
          } else {
            this.buildFeed()
          }
        }
        .width('100%')
        .layoutWeight(1)
        .pullToRefresh(false)
        .refreshOffset(64)
        .onRefreshing(() => { this.refreshHome(); })
      }
      .width('100%')
      .height('100%')
    }
  }

  /** 关注流未登录引导卡片 */
  @Builder
  private buildLoginPrompt() {
    Column({ space: DesignConstants.SPACING.MD }) {
      LucideIcon({
        icon: AppIcon.PERSON,
        iconSize: DesignConstants.COMPONENT.ICON_SIZE_LG,
        color: this.getPalette().accentBackground
      })
      Text('登录后查看关注动态')
        .fontSize(DesignConstants.FONT.SIZE.MD)
        .fontWeight(DesignConstants.FONT.WEIGHT.SEMIBOLD)
        .fontColor(this.getPalette().textPrimary)
      Text('关注同频的好友与活跃邻居，即时获取 TA 们的最新瞬间与生活记录。')
        .fontSize(DesignConstants.FONT.SIZE.SM)
        .fontColor(this.getPalette().textSecondary)
        .textAlign(TextAlign.Center)
        .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG })
      Button('立即登录')
        .fontSize(DesignConstants.FONT.SIZE.SM)
        .fontColor(this.getPalette().accentText)
        .backgroundColor(this.getPalette().accentBackground)
        .borderRadius(DesignConstants.RADIUS.MD)
        .height(36)
        .padding({ left: DesignConstants.SPACING.XL, right: DesignConstants.SPACING.XL })
        .onClick(() => AppRouter.push(PageNames.LOGIN))
    }
    .width('100%')
    .padding({ top: DesignConstants.SPACING.XXL })
    .alignItems(HorizontalAlign.Center)
  }

  /** 空态：分段感知友好文案 */
  @Builder
  private buildEmptyState() {
    EmptyState({
      icon: AppIcon.NEARBY,
      title: this.activeTab === 1 ? '暂无关注动态' : '还没有内容',
      message: this.activeTab === 1 ? '去「为你推荐」发现更多有趣的人，或去附近看看身边正在发生什么。' : '去附近看看，或者发布第一条动态。',
      bare: true,
      actionLabel: this.activeTab === 1 ? '去看看推荐' : '去附近看看',
      onAction: () => {
        if (this.activeTab === 1) {
          this.activeTab = 0;
        } else {
          AppRouter.push(PageNames.NEARBY);
        }
      }
    })
    .padding({ top: DesignConstants.SPACING.XXL })
  }

  /** 动态列表：全面采用高质感富媒体社交卡片（MomentFeedCard） */
  @Builder
  private buildFeed() {
    List({ space: DesignConstants.PAGE_SHELL.SECTION_INNER_GAP }) {
      ForEach(this.currentTabMoments(), (moment: MomentItem) => {
        ListItem() {
          MomentFeedCard({
            moment: moment,
            variant: 'default',
            showTags: true,
            showTime: true,
            textMaxLines: 3,
            onOpenDetail: (m: MomentItem) => this.openMomentDetail(m)
          })
        }
      }, (moment: MomentItem) => moment.moment_id)
    }
    .width('100%')
    .height('100%')
    .edgeEffect(EdgeEffect.None)
    .padding({ top: DesignConstants.SPACING.XS, bottom: DesignConstants.SPACING.XL })
    .scrollBar(BarState.Off)
  }

  @Builder
  private buildStickyHeader() {
    // 沉浸流顶栏：左侧品牌字标锚点，右侧搜索/通知/地图动作，平衡首屏视觉重心
    Row() {
      Row({ space: DesignConstants.SPACING.XS }) {
        LucideIcon({
          icon: AppIcon.NEARBY,
          iconSize: DesignConstants.COMPONENT.ICON_SIZE_SM,
          color: this.getPalette().accentBackground
        })
        Text('HotSpot')
          .fontSize(DesignConstants.FONT.SIZE.LG_PLUS)
          .fontWeight(DesignConstants.FONT.WEIGHT.BOLD)
          .fontColor(this.getPalette().textPrimary)
      }
      .alignItems(VerticalAlign.Center)
      .accessibilityText('HotSpot 首页')

      // Blank 撑开使左右两端对称锚定
      Blank()
      Row({ space: DesignConstants.SPACING.SM2 }) {
        // 搜索/通知/地图入口：中性（透明底 + 主文字色图标），对齐前端 PageHeader trailing
        // （affa411a 为非桌面端补齐通知 Bell 快捷入口，本端同步；搜索/通知/地图三连）
        SurfaceIconButton({
          icon: AppIcon.SEARCH,
          accessibilityLabel: '搜索',
          // surface：表面底 + hairline 描边，对齐 web 的 .action-icon-btn（此前透明底显得又小又散）
          kind: 'surface',
          buttonSize: 36,
          onPress: () => AppRouter.push(PageNames.SEARCH)
        })

        SurfaceIconButton({
          icon: AppIcon.NOTIFICATION,
          accessibilityLabel: '通知',
          // surface：表面底 + hairline 描边，对齐 web 的 .action-icon-btn（此前透明底显得又小又散）
          kind: 'surface',
          buttonSize: 36,
          onPress: () => AppStorage.setOrCreate<boolean>('openNotificationCenter', true)
        })

        SurfaceIconButton({
          icon: AppIcon.MAP,
          accessibilityLabel: '地图',
          // surface：表面底 + hairline 描边，对齐 web 的 .action-icon-btn（此前透明底显得又小又散）
          kind: 'surface',
          buttonSize: 36,
          onPress: () => AppRouter.push(PageNames.LOCATION_PICKER)
        })
      }
      .flexShrink(0)
    }
    .width('100%')
    // 安全区：页头此前由 SystemPageHeader 承担 immersive 的顶部内缩，换成自绘 Row 后
    // 必须自己补 safeAreaTop，否则动作图标压在状态栏上（2026-10-20 端上实测）。
    // 横向留白由 PageShellContainer 的 gutter 统一提供（此处再加会双重内缩）。
    .padding({
      top: this.safeAreaTop + DesignConstants.SPACING.SM,
      bottom: DesignConstants.SPACING.XS
    })
  }

  // ==================== 地图卡片 ====================

  @Builder
  private buildSectionHeader(title: string, linkText: string, pressedIndex: number, onViewAll: () => void) {
    // 对齐前端 SectionHead.svelte：分区标题用真实标题层级（15px semibold），
    // eyebrow 仅用于表单/分组 label；右侧 link-btn 文字链接 + 右箭头
    Row() {
      Text(title)
        .fontSize(DesignConstants.FONT.SIZE.MD)
        .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.MD))
        .fontWeight(DesignConstants.FONT.WEIGHT.SEMIBOLD)
        .fontColor(this.getPalette().textPrimary)

      Blank()

      Row({ space: DesignConstants.SPACING.HALF }) {
        Text(linkText)
          // 对齐前端 SectionHead 的 action（text-xs=11）与附近页：首页此前用 SM(12)，
          // 同一角色跨页字号不一致
          .fontSize(DesignConstants.FONT.SIZE.XS)
          .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS))
          .fontColor(this.getPalette().accentBackground)
        LucideIcon({
          icon: AppIcon.ARROW_RIGHT,
          iconSize: DesignConstants.COMPONENT.ICON_SIZE_2XS,
          color: this.getPalette().accentBackground
        })
      }
      .scale(this.viewAllPressedIndex === pressedIndex
        ? { x: DesignConstants.INTERACTION.SCALE_PRESSED_PILL, y: DesignConstants.INTERACTION.SCALE_PRESSED_PILL }
        : { x: 1, y: 1 })
      .animation({
        curve: curves.springMotion(
          DesignConstants.INTERACTION.SPRING_RESPONSE_PRESS,
          DesignConstants.INTERACTION.SPRING_DAMPING_PRESS
        )
      })
      .onTouch((event: TouchEvent) => {
        if (event.type === TouchType.Down) {
          this.viewAllPressedIndex = pressedIndex;
        } else if (event.type === TouchType.Up || event.type === TouchType.Cancel) {
          this.viewAllPressedIndex = -1;
        }
      })
      .onClick(onViewAll)
      // 触控热区：行高 ~20vp，垂直外扩到 MIN_HIT_TARGET（responseRegion 不占布局）
      .responseRegion({
        x: -DesignConstants.SPACING.MD,
        y: -(DesignConstants.LAYOUT.MIN_HIT_TARGET - DesignConstants.COMPONENT.FILTER_CHIP_HEIGHT) / 2,
        width: '120%',
        height: DesignConstants.LAYOUT.MIN_HIT_TARGET
      })
    }
    .width('100%')
    // 区块标题 → 其内容：SECTION_INNER_GAP(12)，与区块之间的 SECTION_GAP(24) 形成
    // 2:1 层次——标题才在视觉上归属下方内容，而不是飘在两块之间
    .margin({ bottom: DesignConstants.PAGE_SHELL.SECTION_INNER_GAP })
  }

  // ==================== 一条流 ====================

  /**
   * 一条流的数据：关系流（我关注的人）在前，发现流按 moment_id 去重续后。
   * 两个流各自失败不影响另一条：任一为空就只展示另一条。
   * 使用 cachedDisplayedMoments 响应式缓存，避免 build() 周期内 4 次重复遍历与数组分配。
   */
  private displayedMoments(): MomentItem[] {
    return this.cachedDisplayedMoments;
  }
}

class GroupDetailParams {
  groupId: string = '';
}