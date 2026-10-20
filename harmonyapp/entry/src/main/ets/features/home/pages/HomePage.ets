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
import { LoadingUtil } from '../../../core/utils/LoadingUtil';
import { SystemUiController } from '../../../core/utils/SystemUiController';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { AppThemeVisuals } from '../../../core/theme/AppThemeVisuals';
import { DEFAULT_RESOLVED_THEME_ID, KEY_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';
import { getErrorMessage } from '../../../core/network/ApiErrorUtil';
import { curves } from '@kit.ArkUI';

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
  @State moments: MomentItem[] = [];
  /** 关系流（我关注的人，需登录） */
  @State followingMoments: MomentItem[] = [];
  @State momentsLoading: boolean = true;
  @State followingLoading: boolean = false;
  @State momentsError: string = '';
  @State followingError: string = '';

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

        // 整页下拉刷新：Refresh 包内容区（sticky header 保持粘性在外）
        Refresh({ refreshing: $$this.isRefreshing, builder: RefreshIndicator }) {
          Column() {
            // 一条流：关系流在前、发现流去重续后；无标题、无切换器、无分组小标题
            if (this.feedLoading() && this.displayedMoments().length === 0) {
              SystemLoadingView({ message: '正在加载…' })
                .width('100%')
            } else if (this.feedError().length > 0 && this.displayedMoments().length === 0) {
              EmptyState({
                icon: AppIcon.WARNING,
                title: '内容加载失败',
                message: this.feedError(),
                fillMode: true,
                actionLabel: '重试',
                onAction: () => { this.reloadFeed(); }
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

  /** 空态：两个流都为空时才出现（一条流的通用空态） */
  @Builder
  private buildEmptyState() {
    EmptyState({
      icon: AppIcon.NEARBY,
      title: '还没有内容',
      message: '去附近看看，或者发布第一条动态。',
      fillMode: true,
      actionLabel: '去附近看看',
      onAction: () => AppRouter.push(PageNames.NEARBY)
    })
  }

  /** 一条流的动态列表（复用既有活动行原语，与「附近瞬间」行样式一致；无分组小标题） */
  @Builder
  private buildFeed() {
    Scroll() {
      Column({ space: DesignConstants.SPACING.LG }) {
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
  }

  @Builder
  private buildStickyHeader() {
    // 无可见标题：只留右侧搜索/通知/地图三个动作，内容直接开头（对齐 web 首页 2026-10-20）
    Row() {
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
    .width('100%')
    .justifyContent(FlexAlign.End)
    .padding({
      left: DesignConstants.SPACING.LG,
      right: DesignConstants.SPACING.LG,
      top: DesignConstants.SPACING.SM,
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
   */
  private displayedMoments(): MomentItem[] {
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
    return merged;
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