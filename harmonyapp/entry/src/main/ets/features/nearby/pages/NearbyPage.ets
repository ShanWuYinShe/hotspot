/**
 * NearbyPage — 附近板块段位容器（对齐前端 /nearby 三段内容页 IA）
 *
 * 结构（对齐 web /nearby 390 移动端）：header（附近 + 位置胶囊 + 搜索/地图）+ 统计卡
 * + **三段分区平铺**，顺序与 web 一致：附近的人 → 附近瞬间 → 附近群组，
 * 区块头统一 icon + count + 「查看全部」+ chevron（对齐 web SectionHead）：
 * - 附近的人 = 行式列表软岛卡（avatar 44 + 名/副行 + 关注 pill）
 * - 附近瞬间 = 双列 3:4 预览（前 3 条）
 * - 附近群组 = 横向滑动 140×140 方形图卡
 * 各段均带「查看全部」入口进叶子页 NearbyUsersListPage / MomentsFeedPage / NearbyGroupsListPage。
 */

import { SystemPageHeader } from '../../../components/SystemPageHeader';
import { SurfaceIconButton } from '../../../components/SurfaceIconButton';
import { LucideIcon } from '../../../components/LucideIcon';
import { PageShellContainer } from '../../../components/PageShellContainer';
import { ListGroup } from '../../../components/ListGroup';
import type { BreakpointType } from '../../../core/responsive/Breakpoint';
import { AppRouter, MomentDetailRouteParam } from '../../../core/router/AppRouter';
import { curves } from '@kit.ArkUI';
import { GlobalStateManager, KEY_CURRENT_LOCATION, KEY_BROWSE_ANCHOR } from '../../../core/state/GlobalStateManager';
import type { LocationInfo } from '../../../core/location/LocationUtil';
import { PageNames } from '../../../constants/AppConstants';
import { AppIcon } from '../../../components/SystemIcon';
import { DesignConstants, textLineHeight } from '../../../constants/DesignConstants';
import { MotionUtil } from '../../../core/accessibility/MotionUtil';
import { BackToTopButton } from '../../../components/BackToTopButton';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { AppThemeVisuals } from '../../../core/theme/AppThemeVisuals';
import { DEFAULT_RESOLVED_THEME_ID, KEY_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';
import { UserItem, GroupItem, MomentItem, SearchUsersRequest, SearchGroupsRequest, FetchNearbyMomentsRequest } from '../../../core/types/api';
import { ApiClient } from '../../../core/api/ApiClient';
import { UserCard } from '../components/UserCard';
import { GroupCard } from '../../groups/components/GroupCard';
import { MomentWallCard } from '../../moments/components/MomentWallCard';
import { EmptyState, RefreshIndicator } from '../../../components/EmptyState';
import { SystemLoadingView } from '../../../components/SystemLoadingView';

@Preview
@Component
export struct NearbyPage {
  @StorageProp('currentBreakpoint') currentBreakpoint: BreakpointType = 'sm';
  @StorageLink(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = DEFAULT_RESOLVED_THEME_ID;

  @State private nearbyUsers: UserItem[] = [];
  @State private nearbyGroups: GroupItem[] = [];
  @State private nearbyMoments: MomentItem[] = [];
  @State private isLoading: boolean = true;
  @State private enterOpacity: number = 1;
  @State private enterOffset: number = 0;
  private enterPlayed: boolean = false;

  @State private loadError: string = '';
  @State private isRefreshing: boolean = false;
  /** 统计卡（2026-10-19 由首页迁来）：位置与按压态 */
  @StorageLink(KEY_CURRENT_LOCATION) currentLocation: LocationInfo | null = null;
  @StorageLink(KEY_BROWSE_ANCHOR) browseAnchor: LocationInfo | null = null;
  @State private locationCardPressed: boolean = false;

  /** 分区预览条数（对齐 web 枢纽：人 4（行式）/ 瞬间 3（双列预览）/ 群组 4（横滑）） */
  private static readonly PREVIEW_USERS: number = 4;
  private static readonly PREVIEW_GROUPS: number = 4;
  private static readonly PREVIEW_MOMENTS: number = 3;
  /** 附近瞬间双列预览卡宽高比（3:4，对齐 web 枢纽 HomeFeedItem 预览） */
  private static readonly MOMENT_PREVIEW_ASPECT_RATIO: number = 0.75;

  private isDestroyed: boolean = false;

  async aboutToAppear() {
    this.isDestroyed = false;
    await this.loadData();
  }

  aboutToDisappear() {
    this.isDestroyed = true;
  }

  /** 下拉刷新（P2-1：错误文案承诺了「下拉重试」，此前页面无 Refresh） */
  private async refreshNearby(): Promise<void> {
    try {
      await this.loadData();
    } finally {
      if (!this.isDestroyed) {
        this.isRefreshing = false;
      }
    }
  }

  private async loadData(): Promise<void> {
    this.isLoading = true;
    this.loadError = '';
    try {
      const usersReq: SearchUsersRequest = { radius: 5000, sort_by: 'distance', pagination: { limit: 8 } };
      const groupsReq: SearchGroupsRequest = { radius: 5000, sort_by: 'distance', pagination: { limit: 4 } };
      const momentsReq: FetchNearbyMomentsRequest = { radius: 5000, pagination: { limit: 6 } };
      const results = await Promise.all([
        ApiClient.searchUsers(usersReq).catch(() => null),
        ApiClient.searchGroups(groupsReq).catch(() => null),
        ApiClient.fetchNearbyMoments(momentsReq).catch(() => null)
      ]);
      if (this.isDestroyed) return;
      const usersRes = results[0];
      const groupsRes = results[1];
      const momentsRes = results[2];

      if (usersRes && usersRes.code === 0 && usersRes.data?.items) {
        this.nearbyUsers = usersRes.data.items;
      }
      if (groupsRes && groupsRes.code === 0 && groupsRes.data?.items) {
        this.nearbyGroups = groupsRes.data.items;
      }
      if (momentsRes && momentsRes.code === 0 && momentsRes.data?.items) {
        this.nearbyMoments = momentsRes.data.items;
      }

      if (this.nearbyUsers.length === 0 && this.nearbyGroups.length === 0 && this.nearbyMoments.length === 0) {
        if (usersRes?.code !== 0 && groupsRes?.code !== 0 && momentsRes?.code !== 0) {
          this.loadError = '加载周边数据失败';
        }
      }
    } catch (e) {
      if (!this.isDestroyed) {
        this.loadError = '网络异常，请稍后重试';
      }
    } finally {
      if (!this.isDestroyed) {
        this.isLoading = false;
        if (this.loadError.length === 0) {
          this.resetContentEnter();
          this.playContentEnter();
        }
      }
    }
  }

  private getCurrentThemePalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  /** 位置行副标题（对齐前端 nearby 页 PageHeader subtitle：{位置} · 5km 内 / 5km 内） */
  private getLocationSubtitle(): string {
    const name = (this.browseAnchor ?? this.currentLocation)?.locationName ?? '';
    return name.length > 0 ? `${name} · 5km 内` : '5km 内';
  }

  /** 三段是否全空（全空才走整页空态；否则逐段渲染，空段自行隐藏） */
  private hasAnyNearbyData(): boolean {
    return this.nearbyUsers.length > 0 || this.nearbyGroups.length > 0 || this.nearbyMoments.length > 0;
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

  @State private showBackToTop: boolean = false;
  @State private viewportHeight: number = 0;
  private backToTopScroller: Scroller = new Scroller();
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
    if (this.showBackToTop) {
      BackToTopButton({
        visible: true,
        onTap: () => { this.scrollToTop(); }
      })
        .margin({ right: DesignConstants.SPACING.LG, bottom: DesignConstants.SPACING.XXL })
    }
  }

  build() {
    PageShellContainer({ variant: 'default', currentBreakpoint: this.currentBreakpoint}) {
      Column() {
        SystemPageHeader({
          title: '附近',
          subtitle: this.getLocationSubtitle(),
          // 附近域页头标题对齐 web 18px/600（audit §2.3 页头分档）
          subtitleIcon: AppIcon.LOCATION,
          titleSize: DesignConstants.FONT.SIZE.LG_PLUS,
          titleWeight: DesignConstants.FONT.WEIGHT.SEMIBOLD,
          backgroundVariant: 'transparent'
        }) {
          // trailing: 搜索/地图是页面级工具，收敛在标题右侧（对齐前端 nearby 页 trailing）
          Row({ space: DesignConstants.SPACING.SM }) {
            SurfaceIconButton({
              icon: AppIcon.SEARCH,
              accessibilityLabel: '搜索',
              kind: 'surface',
              onPress: () => AppRouter.push(PageNames.SEARCH)
            })
            SurfaceIconButton({
              icon: AppIcon.MAP,
              accessibilityLabel: '地图',
              kind: 'surface',
              onPress: () => AppRouter.push(PageNames.LOCATION_PICKER)
            })
          }
        }

        Refresh({ refreshing: $$this.isRefreshing, builder: RefreshIndicator }) {
        Scroll(this.backToTopScroller) {
          Column({ space: DesignConstants.PAGE_SHELL.SECTION_GAP }) {
            // 统计卡：统计属地理轴，跟「附近」走（2026-10-19 由首页迁来，对齐 web 枢纽页）
            if (!this.isLoading && this.loadError === '') {
              this.buildHeroStats()
            }
            if (this.isLoading) {
              SystemLoadingView({ message: '正在探索周边…' })
                .width('100%')
                .padding({ top: DesignConstants.SPACING.XL })
            } else if (this.loadError !== '') {
              EmptyState({
                bare: true,
                title: this.loadError,
                message: '下拉或稍后重试',
                // 错误态必须给出可点的下一步（2026-10-21 端上走查：此前只有文字提示，无重试入口）
                actionLabel: '重试',
                onAction: () => { void this.loadData(); }
              })
              .width('100%')
            } else if (!this.hasAnyNearbyData()) {
              EmptyState({
                bare: true,
                title: '附近还没有内容',
                message: '去发布第一条动态，或到附近的人认识新朋友。',
                actionLabel: '去发瞬间',
                onAction: () => AppRouter.push(PageNames.CREATE_MOMENT)
              })
              .width('100%')
              .padding({ top: DesignConstants.SPACING.XL })
            } else {
              // ═══ 分区一：附近的人（行式列表软岛卡，预览前 4 位）═══
              if (this.nearbyUsers.length > 0) {
                Column() {
                  this.buildSectionHeader(AppIcon.PERSON, '附近的人', this.nearbyUsers.length, '查看全部',
                    this.nearbyUsers.length > NearbyPage.PREVIEW_USERS, () => {
                      AppRouter.push(PageNames.NEARBY_USERS);
                    })
                  ListGroup() {
                    ForEach(this.nearbyUsers.slice(0, NearbyPage.PREVIEW_USERS), (user: UserItem) => {
                      UserCard({ user: user, variant: 'row' })
                    }, (user: UserItem) => user.user?.user_id ?? '')
                  }
                }
                .width('100%')
              }

              // ═══ 分区二：附近瞬间（双列 3:4 预览，前 3 条）═══
              if (this.nearbyMoments.length > 0) {
                Column() {
                  this.buildSectionHeader(AppIcon.SPARKLES, '附近瞬间', this.nearbyMoments.length, '查看全部',
                    this.nearbyMoments.length > NearbyPage.PREVIEW_MOMENTS, () => {
                      AppRouter.push(PageNames.NEARBY_MOMENTS);
                    })
                  this.buildMomentPreview()
                }
                .width('100%')
              }

              // ═══ 分区三：附近群组（横向滑动方形图卡）═══
              if (this.nearbyGroups.length > 0) {
                Column() {
                  this.buildSectionHeader(AppIcon.GROUP, '附近群组', this.nearbyGroups.length, '查看全部',
                    this.nearbyGroups.length > NearbyPage.PREVIEW_GROUPS, () => {
                      AppRouter.push(PageNames.NEARBY_GROUPS);
                    })
                  this.buildGroupCarousel()
                }
                .width('100%')
              }
            }

            Row()
              .height(DesignConstants.SPACING.XXL)
          }
          .width('100%')
        }
        .width('100%')
        .height('100%')
        .scrollBar(BarState.Off)
        .edgeEffect(EdgeEffect.None)
        .align(Alignment.Top)
        .onDidScroll(() => { this.updateBackToTop(); })
        .onAreaChange((oldArea: Area, newArea: Area) => { this.viewportHeight = Number(newArea.height); })
        }
        .width('100%')
        .layoutWeight(1)
        .pullToRefresh(false)
        .refreshOffset(64)
        .onRefreshing(() => { this.refreshNearby(); })
        .opacity(this.enterOpacity)
        .translate({ y: this.enterOffset })
      }
      .overlay(this.buildBackToTopOverlay(), { align: Alignment.BottomEnd })
      .width('100%')
      .height('100%')
    }
  }

  /** 附近瞬间双列预览（按 2 条切行，行内 layoutWeight 均分；最后一行单卡补 Blank） */
  private getMomentPreviewRows(): MomentItem[][] {
    const list: MomentItem[] = this.nearbyMoments.slice(0, NearbyPage.PREVIEW_MOMENTS);
    const rows: MomentItem[][] = [];
    for (let i = 0; i < list.length; i += 2) {
      rows.push(list.slice(i, i + 2));
    }
    return rows;
  }

  @Builder
  private buildMomentPreview() {
    Column({ space: DesignConstants.SPACING.MD }) {
      ForEach(this.getMomentPreviewRows(), (rowMoments: MomentItem[], rowIndex: number) => {
        Row({ space: DesignConstants.SPACING.MD }) {
          ForEach(rowMoments, (moment: MomentItem) => {
            MomentWallCard({
              moment: moment,
              cardAspectRatio: NearbyPage.MOMENT_PREVIEW_ASPECT_RATIO,
              onOpenDetail: (m: MomentItem) => {
                const param = new MomentDetailRouteParam();
                param.momentId = m.moment_id;
                AppRouter.push(PageNames.MOMENT_DETAIL, param);
              }
            })
              .layoutWeight(1)
          }, (moment: MomentItem) => moment.moment_id)
          if (rowMoments.length === 1) {
            Blank()
              .layoutWeight(1)
          }
        }
        .width('100%')
        .alignItems(VerticalAlign.Top)
      }, (rowMoments: MomentItem[], rowIndex: number) => `nearby-moment-row-${rowIndex}`)
    }
    .width('100%')
  }

  /** 附近群组横滑卡（对齐 web GroupCarousel：140×140 方形图卡、横向滚动） */
  @Builder
  private buildGroupCarousel() {
    Scroll() {
      Row({ space: DesignConstants.SPACING.MD }) {
        ForEach(this.nearbyGroups.slice(0, NearbyPage.PREVIEW_GROUPS), (group: GroupItem) => {
          GroupCard({ group: group, variant: 'carousel' })
        }, (group: GroupItem) => group.group?.group_id ?? '')
      }
    }
    .scrollable(ScrollDirection.Horizontal)
    .scrollBar(BarState.Off)
    .align(Alignment.Start)
    .width('100%')
  }

  /** 统计卡的在线人数（枢纽已加载的用户里在线者） */
  private getOnlineCount(): number {
    if (!this.nearbyUsers || this.nearbyUsers.length === 0) return 0;
    let count: number = 0;
    for (const u of this.nearbyUsers) {
      if (u.presence_status === 'online') count += 1;
    }
    return count;
  }

  private getHeroRangeLabel(): string {
    const locationName = this.browseAnchor?.locationName ?? this.currentLocation?.locationName ?? '';
    return locationName ? `${locationName} · 5km 内` : '5km 内 · 实时更新';
  }

  @Builder
  private buildRadarBadge() {
    // 绿底「实时雷达」pill（对齐 web HeroStats Badge tone=online）
    Row({ space: DesignConstants.SPACING.HALF }) {
      Row()
        .width(DesignConstants.COMPONENT.INDICATOR_DOT_SIZE)
        .height(DesignConstants.COMPONENT.INDICATOR_DOT_SIZE)
        .borderRadius(DesignConstants.RADIUS.FULL)
        .backgroundColor(this.getCurrentThemePalette().online)
      Text('实时雷达')
        .fontSize(DesignConstants.FONT.SIZE.XS2)
        .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS2))
        .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
        .fontColor(this.getCurrentThemePalette().textStatusOnlineOnSoft)
        .maxLines(1)
    }
    .height(DesignConstants.SPACING.LG2)
    .padding({ left: DesignConstants.SPACING.SM, right: DesignConstants.SPACING.SM })
    // online 淡底（theme 派生 + 水印档透明度，不新增固定色）
    .backgroundColor(AppThemeVisuals.withAlpha(this.getCurrentThemePalette().online, DesignConstants.OPACITY.WATERMARK))
    .borderRadius(DesignConstants.RADIUS.FULL)
    .alignItems(VerticalAlign.Center)
  }

  @Builder
  private buildHeroStats() {
    // 实时附近统计卡（对齐前端 HeroStats.svelte：白卡 + shadow-xs + 20 圆角；
    // 标题 14/600 + 绿底实时雷达 pill + 位置副行；发丝分隔的三列统计 24/600）
    Column() {
      Row() {
        Row({ space: DesignConstants.SPACING.SM2 }) {
          // brand 圆形图标（对齐前端 size-10 圆 + accent 实底，无投影）
          Row() {
            LucideIcon({
              icon: AppIcon.NEARBY,
              iconSize: DesignConstants.COMPONENT.ICON_SIZE_SM,
              paletteRole: 'accentText'
            })
          }
          .width(DesignConstants.COMPONENT.BRAND_ICON_SIZE)
          .height(DesignConstants.COMPONENT.BRAND_ICON_SIZE)
          .borderRadius(DesignConstants.RADIUS.FULL)
          .backgroundColor(this.getCurrentThemePalette().accentBackground)
          .justifyContent(FlexAlign.Center)

          Column() {
            Row({ space: DesignConstants.SPACING.SM2 }) {
              Text('实时附近')
                .fontSize(DesignConstants.FONT.SIZE.MD) // 14 对齐前端 text-md
                .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.MD))
                .fontWeight(DesignConstants.FONT.WEIGHT.SEMIBOLD)
                .fontColor(this.getCurrentThemePalette().textPrimary)
              this.buildRadarBadge()
            }
            .alignItems(VerticalAlign.Center)

            // 副标题：定位名 · 范围（对齐前端 HeroStats rangeLabel）
            Text(this.getHeroRangeLabel())
              .fontSize(DesignConstants.FONT.SIZE.XS) // 11 对齐前端 text-xs
              .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS))
              .fontColor(this.getCurrentThemePalette().textSecondary)
              .margin({ top: DesignConstants.SPACING.HALF })
              .maxLines(1)
              .textOverflow({ overflow: TextOverflow.Ellipsis })
          }
          .alignItems(HorizontalAlign.Start)
          .layoutWeight(1)
        }
        .alignItems(VerticalAlign.Center)
        .layoutWeight(1)

        LucideIcon({
          icon: AppIcon.ARROW_RIGHT,
          iconSize: DesignConstants.COMPONENT.ICON_SIZE_XS,
          paletteRole: 'textSecondary'
        })
      }
      .width('100%')
      .alignItems(VerticalAlign.Center)

      // Stats grid：三段统计 + 竖分割线（对齐前端 bg-border-default gap-px 网格语义）
      Row({ space: DesignConstants.SPACING.MD }) {
        // 人（对齐前端 stats.nearby）
        Column({ space: DesignConstants.SPACING.SM2 }) {
          Text(`${this.nearbyUsers.length}`)
            .fontSize(DesignConstants.FONT.SIZE.XXL)
            .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XXL))
            .fontWeight(DesignConstants.FONT.WEIGHT.SEMIBOLD)
            .fontColor(this.getCurrentThemePalette().textPrimary)
          Text('人')
            .fontSize(DesignConstants.FONT.SIZE.XS)
            .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS))
            .fontColor(this.getCurrentThemePalette().textSecondary)
        }
        .alignItems(HorizontalAlign.Center)
        .layoutWeight(1)
        .padding({ top: DesignConstants.SPACING.MD, bottom: DesignConstants.SPACING.MD })

        // 分割竖线
        Column()
          .width(DesignConstants.BORDER_WIDTH.HAIRLINE)
          .height(40) // 对齐前端 gap-px 网格线（随统计格内容高：数字 24 + 标签 11 + 间距 6）
          .backgroundColor(this.getCurrentThemePalette().borderColor)
          .opacity(DesignConstants.OPACITY.MUTED)

        // 瞬间（对齐前端 heroStats.active = moments.length）
        Column({ space: DesignConstants.SPACING.SM2 }) {
          Text(`${this.nearbyMoments.length}`)
            .fontSize(DesignConstants.FONT.SIZE.XXL)
            .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XXL))
            .fontWeight(DesignConstants.FONT.WEIGHT.SEMIBOLD)
            .fontColor(this.getCurrentThemePalette().textPrimary)
          Text('瞬间')
            .fontSize(DesignConstants.FONT.SIZE.XS)
            .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS))
            .fontColor(this.getCurrentThemePalette().textSecondary)
        }
        .alignItems(HorizontalAlign.Center)
        .layoutWeight(1)
        .padding({ top: DesignConstants.SPACING.MD, bottom: DesignConstants.SPACING.MD })

        // 分割竖线
        Column()
          .width(DesignConstants.BORDER_WIDTH.HAIRLINE)
          .height(40)
          .backgroundColor(this.getCurrentThemePalette().borderColor)
          .opacity(DesignConstants.OPACITY.MUTED)

        // 在线
        Column({ space: DesignConstants.SPACING.SM2 }) {
          Text(`${this.getOnlineCount()}`)
            .fontSize(DesignConstants.FONT.SIZE.XXL)
            .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XXL))
            .fontWeight(DesignConstants.FONT.WEIGHT.SEMIBOLD)
            .fontColor(this.getCurrentThemePalette().textPrimary)
          Text('在线')
            .fontSize(DesignConstants.FONT.SIZE.XS)
            .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS))
            .fontColor(this.getCurrentThemePalette().textSecondary)
        }
        .alignItems(HorizontalAlign.Center)
        .layoutWeight(1)
        .padding({ top: DesignConstants.SPACING.MD, bottom: DesignConstants.SPACING.MD })
      }
      .width('100%')
      .margin({ top: DesignConstants.SPACING.MD3 })
    }
    .width('100%')
    // 对齐 web HeroStats：pt-4/pb-4(16) px-5(20) + rounded-lg(20) + shadow-xs
    .padding({
      top: DesignConstants.SPACING.LG,
      bottom: DesignConstants.SPACING.LG,
      left: DesignConstants.SPACING.LG2,
      right: DesignConstants.SPACING.LG2
    })
    .borderRadius(DesignConstants.SHAPE.CARD)
    .backgroundColor(this.getCurrentThemePalette().surfaceIsland)
    .shadow(this.getCurrentThemePalette().surfaceShadow)
    .scale(this.locationCardPressed
      ? { x: DesignConstants.INTERACTION.SCALE_PRESSED_CARD, y: DesignConstants.INTERACTION.SCALE_PRESSED_CARD }
      : { x: 1, y: 1 })
    .animation({
      curve: curves.springMotion(
        DesignConstants.INTERACTION.SPRING_RESPONSE_PRESS,
        DesignConstants.INTERACTION.SPRING_DAMPING_PRESS
      )
    })
    .onTouch((event: TouchEvent) => {
      if (event.type === TouchType.Down) {
        this.locationCardPressed = true;
      } else if (event.type === TouchType.Up || event.type === TouchType.Cancel) {
        this.locationCardPressed = false;
      }
    })
    .onClick(() => AppRouter.push(PageNames.NEARBY_USERS))
  }

  /** 分区头（对齐 web SectionHead）：icon + 标题 14/600 + count 11 + 右侧「查看全部」+ chevron */
  @Builder
  private buildSectionHeader(icon: AppIcon, title: string, count: number, actionText: string,
    showAction: boolean, onAction: () => void) {
    Row() {
      Row({ space: DesignConstants.SPACING.SM }) {
        LucideIcon({
          icon: icon,
          iconSize: DesignConstants.COMPONENT.ICON_SIZE_XS,
          color: this.getCurrentThemePalette().textSecondary
        })
        .flexShrink(0)

        Text(title)
          .fontSize(DesignConstants.FONT.SIZE.MD)
          .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.MD))
          .fontWeight(DesignConstants.FONT.WEIGHT.SEMIBOLD)
          .fontColor(this.getCurrentThemePalette().textPrimary)
          .maxLines(1)

        Text(`${count}`)
          .fontSize(DesignConstants.FONT.SIZE.XS)
          .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS))
          .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
          .fontColor(this.getCurrentThemePalette().textSecondary)
          .maxLines(1)
          .flexShrink(0)
      }
      .alignItems(VerticalAlign.Center)
      .layoutWeight(1)

      // 与 web SectionHead 一致：预览未覆盖全部内容时才出现「查看全部」入口
      if (showAction) {
        Row({ space: DesignConstants.SPACING.HALF }) {
          Text(actionText)
            .fontSize(DesignConstants.FONT.SIZE.XS)
            .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS))
            .fontColor(this.getCurrentThemePalette().accentBackground)
          LucideIcon({
            icon: AppIcon.ARROW_RIGHT,
            iconSize: DesignConstants.COMPONENT.ICON_SIZE_2XS,
            color: this.getCurrentThemePalette().accentBackground
          })
        }
        .alignItems(VerticalAlign.Center)
        .flexShrink(0)
        .onClick(onAction)
        // 触控热区：文字链接视觉高约 15vp，垂直外扩到 MIN_HIT_TARGET（responseRegion 不占布局）
        .responseRegion({
          x: -DesignConstants.SPACING.MD,
          y: -(DesignConstants.LAYOUT.MIN_HIT_TARGET - DesignConstants.COMPONENT.FILTER_CHIP_HEIGHT) / 2,
          width: '130%',
          height: DesignConstants.LAYOUT.MIN_HIT_TARGET
        })
      }
    }
    .width('100%')
    .alignItems(VerticalAlign.Center)
    // 区块标题 → 其内容：SECTION_INNER_GAP(12)，与区块之间的 SECTION_GAP(24) 形成 2:1 层次
    .margin({ bottom: DesignConstants.PAGE_SHELL.SECTION_INNER_GAP })
  }
}

@Builder
export function NearbyPageBuilder() {
  NearbyPage()
}
