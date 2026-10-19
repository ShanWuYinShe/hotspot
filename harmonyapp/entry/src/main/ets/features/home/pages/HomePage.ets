/**
 * HomePage — 首页「关注 / 发现」双轴（2026-10-19 信息架构调整，对齐 web 首页）
 *
 * 背景：此前首页是「热门瞬间 + 附近用户 + 附近群组」聚合，事实上与「附近」重复
 * （web 同形问题已修）。调整后三条轴各占一个表面，内容不再重复：
 *   首页「关注」= moments/fetch_following（关系轴，需登录）
 *   首页「发现」= moments/search 热门模式（算法轴；spec：无输入的热门发现承接发现页首屏）
 *   附近        = NearbyPage 枢纽（地理轴：人 / 群组 / 瞬间）
 * 默认段与「关注」段空态由 HomeFeedMode 的纯函数决定（含冷启动矩阵，与 web 同口径）。
 * 附近用户/群组及其关注、入群交互随区块移出首页，由附近枢纽与叶子页承担。
 */

import { PageShellContainer } from '../../../components/PageShellContainer';
import type { BreakpointType } from '../../../core/responsive/Breakpoint';
import { ApiClient } from '../../../core/api/ApiClient';
import { AppRouter, MomentDetailRouteParam } from '../../../core/router/AppRouter';
import { PageNames } from '../../../constants/AppConstants';
import { MomentItem, SearchMomentsRequest, FetchFollowingMomentsRequest } from '../../../core/types/api';
import { SystemPageHeader } from '../../../components/SystemPageHeader';
import { SurfaceIconButton } from '../../../components/SurfaceIconButton';
import { TabsPillGroup, TabItem } from '../../../components/TabsPillGroup';
import { LoginPromptView } from '../../../components/LoginPromptView';
import { SystemLoadingView } from '../../../components/SystemLoadingView';
import { DesignConstants, textLineHeight } from '../../../constants/DesignConstants';
import { AppIcon } from '../../../components/SystemIcon';
import { LucideIcon } from '../../../components/LucideIcon';
import { EmptyState, RefreshIndicator } from '../../../components/EmptyState';
import { LoadingUtil } from '../../../core/utils/LoadingUtil';
import { SystemUiController } from '../../../core/utils/SystemUiController';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { AppThemeVisuals } from '../../../core/theme/AppThemeVisuals';
import { DEFAULT_RESOLVED_THEME_ID, KEY_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';
import { getErrorMessage } from '../../../core/network/ApiErrorUtil';
import { curves } from '@kit.ArkUI';
import {
  HomeFeedSegment,
  resolveHomeSegment,
  resolveFollowingEmpty,
  FollowingEmptyCopy
} from '../HomeFeedMode';

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

  /** 当前段：关注 / 发现 */
  @State activeSegment: HomeFeedSegment = 'discover';
  /** 发现段（热门） */
  @State moments: MomentItem[] = [];
  /** 关注段 */
  @State followingMoments: MomentItem[] = [];
  @State momentsLoading: boolean = true;
  @State followingLoading: boolean = false;
  @State momentsError: string = '';
  @State followingError: string = '';
  /** 关注数（-1 = 尚未探测）；只为「零关系」与「有关注但没更新」的空态分流 */
  @State followingCount: number = -1;
  /** 首屏判定期间显示加载态，避免「先关注后跳发现」的段位翻动 */
  @State segmentReady: boolean = false;

  @State headerElevated: boolean = false;
  /** 「查看全部」链接的按下态索引（-1 = 无；对齐前端 link 按压反馈） */
  @State viewAllPressedIndex: number = -1;
  @State pressedMomentKey: string = '';
  @State isRefreshing: boolean = false;

  private isDestroyed: boolean = false;

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  async aboutToAppear() {
    SystemUiController.syncWithCurrentTheme();
    this.loadDiscoverMoments();
    this.bootstrapSegment();
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

  /**
   * 首屏判定默认段：未登录直接进「发现」；已登录则先取关注流首屏 + 关注数探测
   * （users/fetch_following limit:1，足够分辨零与非零），判定在加载态内完成。
   * 用顺序 await 而非 Promise.all 解构：ArkTS 对异构元组解构支持有限。
   */
  private async bootstrapSegment(): Promise<void> {
    if (!this.isLoggedIn) {
      this.activeSegment = resolveHomeSegment(false, 0);
      this.segmentReady = true;
      return;
    }
    this.followingLoading = true;
    try {
      const request: FetchFollowingMomentsRequest = { pagination: { limit: FEED_PAGE_SIZE } };
      const response = await ApiClient.fetchFollowingMoments(request);
      if (this.isDestroyed) return;
      if (response.code === 0) {
        this.followingMoments = (response.data?.items ?? []).filter((m: MomentItem) => m.moment_id && m.user);
      } else {
        this.followingError = getErrorMessage(response, '加载失败');
      }
      const followRes = await ApiClient.getFollowingList({ pagination: { limit: 1 } });
      if (this.isDestroyed) return;
      this.followingCount = followRes.code === 0 ? (followRes.data?.items?.length ?? 0) : 0;
      this.activeSegment = resolveHomeSegment(true, this.followingCount);
    } catch {
      // 探测失败不阻塞首页：退回发现轴（关注段仍可手动进入）
      this.activeSegment = 'discover';
    } finally {
      if (!this.isDestroyed) {
        this.followingLoading = false;
        this.segmentReady = true;
      }
    }
  }

  private openMomentDetail(moment: MomentItem): void {
    const params = new MomentDetailRouteParam();
    params.momentId = moment.moment_id;
    AppRouter.push(PageNames.MOMENT_DETAIL, params);
  }

  private segmentLoading(): boolean {
    return this.activeSegment === 'following' ? this.followingLoading : this.momentsLoading;
  }

  private segmentError(): string {
    return this.activeSegment === 'following' ? this.followingError : this.momentsError;
  }

  private reloadSegment(): void {
    if (this.activeSegment === 'following') {
      this.loadFollowingMoments();
    } else {
      this.loadDiscoverMoments();
    }
  }

  private async refreshHome(): Promise<void> {
    try {
      if (this.activeSegment === 'following') {
        await this.loadFollowingMoments();
      } else {
        await this.loadDiscoverMoments();
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

        // 整页下拉刷新：Refresh 包内容区（sticky header 保持粘性在外）
        Refresh({ refreshing: $$this.isRefreshing, builder: RefreshIndicator }) {
          Column() {
            // 段位切换常驻（空态/错误态下也可切换，避免被状态困住）
            this.buildSegmentSwitcher()

            if (!this.segmentReady) {
              SystemLoadingView({ message: '正在加载…' })
                .width('100%')
            } else if (this.activeSegment === 'following' && !this.isLoggedIn) {
              // 关注流无匿名版：未登录是正常业务态，走登录引导而非错误态
              LoginPromptView({
                title: '登录后查看关注动态',
                subtitle: '关注的人发布的动态会出现在这里',
                icon: AppIcon.PERSON,
                loginAction: '查看关注动态'
              })
                .width('100%')
            } else if (this.segmentLoading()) {
              SystemLoadingView({
                message: this.activeSegment === 'following' ? '正在加载关注动态…' : '正在加载热门内容…'
              })
                .width('100%')
            } else if (this.segmentError().length > 0) {
              EmptyState({
                icon: AppIcon.WARNING,
                title: '内容加载失败',
                message: this.segmentError(),
                fillMode: true,
                actionLabel: '重试',
                onAction: () => { this.reloadSegment(); }
              })
            } else if (this.displayedMoments().length === 0) {
              this.buildEmptyState()
            } else {
              this.buildFeed()
            }
          }
          .width('100%')
          .height('100%')
        }
        .width('100%')
        .layoutWeight(1)
        .onRefreshing(() => { this.refreshHome(); })
      }
      .width('100%')
      .height('100%')
    }
  }

  /** 段位切换（对齐 web 首页 TabBar segmented：关注 / 发现） */
  @Builder
  private buildSegmentSwitcher() {
    TabsPillGroup({
      tabs: [{ label: '关注' } as TabItem, { label: '发现' } as TabItem],
      activeIndex: this.activeSegment === 'following' ? 0 : 1,
      actions: {
        onChange: (index: number) => {
          const next: HomeFeedSegment = index === 0 ? 'following' : 'discover';
          if (next === this.activeSegment) return;
          this.activeSegment = next;
          // 懒加载：切到「关注」首次进入时补拉（首屏已由 bootstrapSegment 拉过）
          if (next === 'following' && this.isLoggedIn && this.followingMoments.length === 0
            && this.followingError.length === 0 && !this.followingLoading) {
            this.loadFollowingMoments();
          }
        }
      }
    })
      .width('100%')
      .padding({
        left: DesignConstants.SPACING.LG,
        right: DesignConstants.SPACING.LG,
        top: DesignConstants.SPACING.SM,
        bottom: DesignConstants.SPACING.XS
      })
  }

  /** 空态：关注段按关注计数分流（见 HomeFeedMode），发现段为通用空态 */
  @Builder
  private buildEmptyState() {
    if (this.activeSegment === 'following' && this.isLoggedIn) {
      EmptyState({
        icon: AppIcon.PERSON,
        title: resolveFollowingEmpty(this.followingCount).title,
        message: resolveFollowingEmpty(this.followingCount).message,
        fillMode: true,
        actionLabel: resolveFollowingEmpty(this.followingCount).actionLabel,
        onAction: () => {
          const copy: FollowingEmptyCopy = resolveFollowingEmpty(this.followingCount);
          if (copy.noRelations) {
            AppRouter.push(PageNames.NEARBY);
          } else {
            this.activeSegment = 'discover';
          }
        }
      })
    } else {
      EmptyState({
        icon: AppIcon.NEARBY,
        title: '暂时还没有热门内容',
        message: '去附近看看，或者发布第一条动态。',
        fillMode: true,
        actionLabel: '去附近看看',
        onAction: () => AppRouter.push(PageNames.NEARBY)
      })
    }
  }

  /** 当前段的动态列表（复用既有活动行原语，与「附近瞬间」行样式一致） */
  @Builder
  private buildFeed() {
    Scroll() {
      Column({ space: DesignConstants.SPACING.LG }) {
        this.buildSectionHeader(
          this.activeSegment === 'following' ? '关注动态' : '热门动态',
          this.activeSegment === 'following' ? '附近' : '更多',
          0,
          () => {
            AppRouter.push(this.activeSegment === 'following' ? PageNames.NEARBY : PageNames.NEARBY_MOMENTS);
          }
        )
        Column() {
          ForEach(this.homeActivityRows(), (moment: MomentItem, index: number) => {
            this.buildActivityRow(moment, index)
          }, (moment: MomentItem) => moment.moment_id)
        }
        .width('100%')
      }
      .width('100%')
      .padding({ top: DesignConstants.SPACING.LG, bottom: DesignConstants.SPACING.XL })
    }
    .layoutWeight(1)
    .scrollBar(BarState.Off)
    .align(Alignment.TopStart)
    .onDidScroll((xOffset: number, yOffset: number) => {
      // 粘性头部滚动态：滚动 > 4vp 出现投影与边线，回顶恢复融合
      this.headerElevated = yOffset > 4;
    })
  }

  @Builder
  private buildStickyHeader() {
    // 统一走 SystemPageHeader（标题 + 副标题 + 位置 chip trailing）；
    // 顶级 Tab 页标题对齐前端 titleSize=3xl（30）+ semibold（移动端，+page.svelte）
    SystemPageHeader({
      title: '首页',
      subtitle: '关注与发现',
      titleSize: DesignConstants.FONT.SIZE.XXL,
      titleWeight: DesignConstants.FONT.WEIGHT.SEMIBOLD,
      elevated: this.headerElevated
    }) {
      // trailing: 在线计数 chip + 搜索/通知/地图入口（对齐前端 PageHeader trailing）。
      // 溢出优先级（对齐前端弹性契约）：trailing 簇 flexShrink(0) 锁定不被压缩，
      // 窄屏下由标题副标题先省略让位（SystemPageHeader 内 Text 已 maxLines(1)+Ellipsis），
      // chip 文本同样 maxLines(1) 单行不折行——此前 trailing 无收缩保护，
      // 窄屏下 chip 会被压折行（对齐前端 360px 下「4 在线」折成两行的同款问题）。
      Row({ space: DesignConstants.SPACING.SM2 }) {
        // 搜索/通知/地图入口：中性（透明底 + 主文字色图标），对齐前端 PageHeader trailing
        // （affa411a 为非桌面端补齐通知 Bell 快捷入口，本端同步；搜索/通知/地图三连）
        SurfaceIconButton({
          icon: AppIcon.SEARCH,
          accessibilityLabel: '搜索',
          kind: 'neutral',
          buttonSize: 36,
          onPress: () => AppRouter.push(PageNames.SEARCH)
        })

        SurfaceIconButton({
          icon: AppIcon.NOTIFICATION,
          accessibilityLabel: '通知',
          kind: 'neutral',
          buttonSize: 36,
          onPress: () => AppStorage.setOrCreate<boolean>('openNotificationCenter', true)
        })

        SurfaceIconButton({
          icon: AppIcon.MAP,
          accessibilityLabel: '地图',
          kind: 'neutral',
          buttonSize: 36,
          onPress: () => AppRouter.push(PageNames.LOCATION_PICKER)
        })
      }
      .flexShrink(0)
    }
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

  // ==================== 附近的人 ====================

  /** 当前段的数据（关注 / 发现） */
  private displayedMoments(): MomentItem[] {
    return this.activeSegment === 'following' ? this.followingMoments : this.moments;
  }

  private homeActivityRows(): MomentItem[] {
    return this.displayedMoments();
  }

  // 活动行配色轮换（对齐前端 ActivityRow ICONS/COLORS/BGS 四档循环）：
  // [warning+咖啡, accent+音乐, marker-group+人群, danger+星标]，淡底全部走 token。
  private activityAccent(index: number): string {
    const accents: string[] = [
      this.getPalette().warning,
      this.getPalette().accentBackground,
      this.getPalette().mapGroup,
      this.getPalette().error
    ];
    return accents[index % 4];
  }

  private activitySoftBg(index: number): string {
    const bgs: string[] = [
      this.getPalette().warningSoft,
      this.getPalette().accentMuted,
      AppThemeVisuals.withAlpha(this.getPalette().mapGroup, 0.12),
      this.getPalette().dangerSoft
    ];
    return bgs[index % 4];
  }

  private activityIcon(index: number): AppIcon {
    const icons: AppIcon[] = [AppIcon.COFFEE, AppIcon.MUSIC, AppIcon.GROUP, AppIcon.STAR];
    return icons[index % 4];
  }

  @Builder
  private buildActivityRow(moment: MomentItem, index: number) {
    Row({ space: DesignConstants.SPACING.MD }) {
      Column() {
        LucideIcon({
          icon: this.activityIcon(index),
          iconSize: DesignConstants.COMPONENT.ICON_SIZE_XS,
          color: this.activityAccent(index)
        })
      }
      // 对齐前端 ActivityRow size-8.5(34px) 图标圆底
      .width(DesignConstants.COMPONENT.ACTIVITY_ICON_TILE_SIZE)
      .height(DesignConstants.COMPONENT.ACTIVITY_ICON_TILE_SIZE)
      .borderRadius(DesignConstants.RADIUS.SM)
      .backgroundColor(this.activitySoftBg(index))
      .justifyContent(FlexAlign.Center)

      Column({ space: DesignConstants.SPACING.XS }) {
        Text(moment.description || '分享了一条动态')
          .fontSize(DesignConstants.FONT.SIZE.SM) // 对齐前端 ActivityRow text-sm=12
          .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.SM))
          .fontColor(this.getPalette().textPrimary)
          .maxLines(2)
          .textOverflow({ overflow: TextOverflow.Ellipsis })
        Text(moment.location?.location_name ?? '')
          .fontSize(DesignConstants.FONT.SIZE.XS) // 对齐前端 text-xs=11
          .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS))
          .fontColor(this.getPalette().textSecondary)
          .maxLines(1)
          .textOverflow({ overflow: TextOverflow.Ellipsis })
          .margin({ top: 1 })
      }
      .alignItems(HorizontalAlign.Start)
      .layoutWeight(1)

      LucideIcon({
        icon: AppIcon.ARROW_RIGHT,
        iconSize: DesignConstants.COMPONENT.ICON_SIZE_2XS,
        paletteRole: 'textSecondary'
      })
        // 标题折行时（行高对齐前端 17 后更易两行）文字与 chevron 会贴在一起，
        // 补一个左间距把两者分开
        .margin({ left: DesignConstants.SPACING.SM2 })
    }
    .width('100%')
    // 白卡容器内行：px-3 py-2.5（对齐前端 ActivityRow ListItem px-4 py-3 扣除容器 p-1.5）
    .padding({ left: DesignConstants.SPACING.MD2, right: DesignConstants.SPACING.MD2, top: DesignConstants.SPACING.SM, bottom: DesignConstants.SPACING.SM })
    // 整行按压反馈（对齐 tap-97 语义）
    .scale(this.pressedMomentKey === moment.moment_id ?
      { x: DesignConstants.INTERACTION.SCALE_PRESSED_CARD, y: DesignConstants.INTERACTION.SCALE_PRESSED_CARD } :
      { x: 1, y: 1 })
    .animation({ curve: curves.springMotion(
      DesignConstants.INTERACTION.SPRING_RESPONSE_PRESS,
      DesignConstants.INTERACTION.SPRING_DAMPING_PRESS
    ) })
    .onTouch((event: TouchEvent) => {
      if (event.type === TouchType.Down) {
        this.pressedMomentKey = moment.moment_id;
      } else if (event.type === TouchType.Up || event.type === TouchType.Cancel) {
        this.pressedMomentKey = '';
      }
    })
    .onClick(() => this.openMomentDetail(moment))
  }
}

class GroupDetailParams {
  groupId: string = '';
}