/**
 * NearbyPage — 附近板块段位容器（对齐前端 /nearby 三段内容页 IA）
 *
 * 结构：header（附近 + 搜索/地图工具图标）+ TabsPillGroup 三段段位导航
 * （附近的人/群组/动态，对齐前端 NearbySegmentNav）+ 当前段真实周边数据
 * 预览流，「查看全部」进入对应完整列表页（NearbyUsersListPage /
 * NearbyGroupsListPage / MomentsFeedPage）。
 * 原搜索卡片与 4 分区入口芯片已收敛：搜索/地图降为 header 右上角工具入口。
 */

import { SystemPageHeader } from '../../../components/SystemPageHeader';
import { SurfaceIconButton } from '../../../components/SurfaceIconButton';
import { LucideIcon } from '../../../components/LucideIcon';
import { TabsPillGroup, TabItem } from '../../../components/TabsPillGroup';
import { PageShellContainer } from '../../../components/PageShellContainer';
import type { BreakpointType } from '../../../core/responsive/Breakpoint';
import { AppRouter, MomentDetailRouteParam } from '../../../core/router/AppRouter';
import { curves } from '@kit.ArkUI';
import { GlobalStateManager, KEY_CURRENT_LOCATION, KEY_BROWSE_ANCHOR } from '../../../core/state/GlobalStateManager';
import type { LocationInfo } from '../../../core/location/LocationUtil';
import { PageNames } from '../../../constants/AppConstants';
import { AppIcon } from '../../../components/SystemIcon';
import { DesignConstants, textLineHeight } from '../../../constants/DesignConstants';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
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
  /** 当前段位（0 人 / 1 群组 / 2 动态），默认「附近的人」——对齐前端 /nearby → people */
  @State private activeSection: number = 0;

  @State private nearbyUsers: UserItem[] = [];
  @State private nearbyGroups: GroupItem[] = [];
  @State private nearbyMoments: MomentItem[] = [];
  @State private isLoading: boolean = true;
  @State private loadError: string = '';
  @State private isRefreshing: boolean = false;
  /** 统计卡（2026-10-19 由首页迁来）：位置与按压态 */
  @StorageLink(KEY_CURRENT_LOCATION) currentLocation: LocationInfo | null = null;
  @StorageLink(KEY_BROWSE_ANCHOR) browseAnchor: LocationInfo | null = null;
  @State private locationCardPressed: boolean = false;

  private static readonly SECTION_TABS: string[] = ['附近的人', '附近群组', '附近动态'];

  /** 附近的人双列卡墙行分块（对齐前端 nearby/people 2 列网格；行分块避开 Grid 嵌 Scroll 高度测量） */
  private getChunkedNearbyUsers(): UserItem[][] {
    const rows: UserItem[][] = [];
    for (let i = 0; i < this.nearbyUsers.length; i += 2) {
      rows.push(this.nearbyUsers.slice(i, i + 2));
    }
    return rows;
  }

  async aboutToAppear() {
    await this.loadData();
  }

  /** 下拉刷新（P2-1：错误文案承诺了「下拉重试」，此前页面无 Refresh） */
  private async refreshNearby(): Promise<void> {
    try {
      await this.loadData();
    } finally {
      this.isRefreshing = false;
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
      this.loadError = '网络异常，请稍后重试';
    } finally {
      this.isLoading = false;
    }
  }

  private getCurrentThemePalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  private getSurfaceBorder(): string {
    return this.getCurrentThemePalette().borderColor;
  }

  /** 当前段位的空态文案（区分三种内容，引导语义对齐前端各段空态） */
  private emptyTitleForSection(section: number): string {
    if (section === 0) return '附近暂时没有其他人';
    if (section === 1) return '附近暂时没有群组';
    return '附近暂时没有瞬间';
  }

  /** 当前段位是否有内容（决定渲染预览流还是空态） */
  private sectionHasContent(section: number): boolean {
    if (section === 0) return this.nearbyUsers.length > 0;
    if (section === 1) return this.nearbyGroups.length > 0;
    return this.nearbyMoments.length > 0;
  }

  build() {
    PageShellContainer({ variant: 'default', currentBreakpoint: this.currentBreakpoint}) {
      Column() {
        SystemPageHeader({
          title: '附近',
          subtitle: '5km 实时探索',
          // 顶级 Tab 页标题对齐前端 titleSize=3xl（30）+ semibold（移动端）
          titleSize: DesignConstants.FONT.SIZE.XXL,
          titleWeight: DesignConstants.FONT.WEIGHT.SEMIBOLD,
          backgroundVariant: 'transparent'
        }) {
          // trailing: 搜索/地图是页面级工具，收敛在标题右侧（对齐前端 nearby 页 trailing）
          Row({ space: DesignConstants.SPACING.SM }) {
            SurfaceIconButton({
              icon: AppIcon.SEARCH,
              accessibilityLabel: '搜索',
              kind: 'neutral',
              onPress: () => AppRouter.push(PageNames.SEARCH)
            })
            SurfaceIconButton({
              icon: AppIcon.MAP,
              accessibilityLabel: '地图',
              kind: 'neutral',
              onPress: () => AppRouter.push(PageNames.LOCATION_PICKER)
            })
          }
        }

        // 段位导航（对齐前端 NearbySegmentNav：附近的人/群组/动态三段吸顶）
        TabsPillGroup({
          tabs: NearbyPage.SECTION_TABS.map((label: string) => ({ label: label } as TabItem)),
          activeIndex: this.activeSection,
          actions: {
            onChange: (index: number) => {
              this.activeSection = index;
            }
          }
        })
          .padding({ top: DesignConstants.SPACING.SM, bottom: DesignConstants.SPACING.SM })

        Refresh({ refreshing: $$this.isRefreshing, builder: RefreshIndicator }) {
        Scroll() {
          Column({ space: DesignConstants.SPACING.LG }) {
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
                title: this.loadError,
                message: '下拉或稍后重试'
              })
              .width('100%')
            } else if (!this.sectionHasContent(this.activeSection)) {
              // 空态引导动作（P2-5）：按段位给语义一致的 CTA——群组段直达创建群组，
              // 人/动态段引导发动态点亮区域（r45：群组空态配「去发动态」语义不闭环）
              EmptyState({
                title: this.emptyTitleForSection(this.activeSection),
                message: this.activeSection === 1 ? '成为第一个创建群组的人吧！' : '成为第一个点亮这片区域的人吧！',
                actionLabel: this.activeSection === 1 ? '创建群组' : '去发动态',
                onAction: () => AppRouter.push(this.activeSection === 1 ? PageNames.CREATE_GROUP : PageNames.CREATE_MOMENT)
              })
              .width('100%')
              .padding({ top: DesignConstants.SPACING.XL })
            } else if (this.activeSection === 0) {
              // ═══ 段位一：附近的人（图片优先双列卡墙，对齐前端 nearby/people 2 列网格）═══
              Column({ space: DesignConstants.SPACING.LG }) {
                this.buildSectionHeader('身边的新朋友', '查看全部', () => {
                  AppRouter.push(PageNames.NEARBY_USERS);
                })

                ForEach(this.getChunkedNearbyUsers(), (rowUsers: UserItem[], rowIndex: number) => {
                  Row({ space: DesignConstants.SPACING.LG }) {
                    ForEach(rowUsers, (user: UserItem) => {
                      Column() {
                        // portrait 3:4 图片卡（与查看全部页 NearbyUsersListPage 同卡同契约）
                        UserCard({ user: user })
                      }
                      .layoutWeight(1)
                    }, (user: UserItem) => user.user?.user_id ?? '')
                  }
                  .width('100%')
                  .alignItems(VerticalAlign.Top)
                }, (rowUsers: UserItem[], rowIndex: number) => `nearby-user-row-${rowIndex}`)
              }
              .width('100%')
            } else if (this.activeSection === 1) {
              // ═══ 段位二：附近群组 ═══
              Column() {
                this.buildSectionHeader('身边兴趣圈', '探索群组', () => {
                  AppRouter.push(PageNames.NEARBY_GROUPS);
                })

                Column({ space: DesignConstants.SPACING.MD }) {
                  ForEach(this.nearbyGroups, (group: GroupItem) => {
                    GroupCard({ group: group, featured: false })
                  }, (group: GroupItem) => group.group?.group_id ?? '')
                }
              }
              .width('100%')
            } else {
              // ═══ 段位三：附近动态 ═══
              Column() {
                this.buildSectionHeader('周边新鲜事', '更多动态', () => {
                  AppRouter.push(PageNames.NEARBY_MOMENTS);
                })

                Column({ space: DesignConstants.SPACING.MD }) {
                  ForEach(this.nearbyMoments, (moment: MomentItem) => {
                    MomentWallCard({
                      moment: moment,
                      onOpenDetail: (m) => {
                        const param = new MomentDetailRouteParam();
                        param.momentId = m.moment_id;
                        AppRouter.push(PageNames.MOMENT_DETAIL, param);
                      }
                    })
                  }, (moment: MomentItem) => moment.moment_id)
                }
              }
              .width('100%')
            }

            Row()
              .height(DesignConstants.SPACING.XXL)
          }
          .width('100%')
        }
        .width('100%')
        .height('100%')
        .scrollBar(BarState.Off)
        .edgeEffect(EdgeEffect.Spring)
        .align(Alignment.Top)
        }
        .width('100%')
        .layoutWeight(1)
        .onRefreshing(() => { this.refreshNearby(); })
      }
      .width('100%')
      .height('100%')
    }
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
    return locationName ? `${locationName} · 5km 内 · 持续更新` : '5km 内 · 持续更新';
  }

  @Builder
  private buildHeroStats() {
    // 实时附近统计卡（对齐前端 HeroStats.svelte 当前实现：白卡 + 发丝描边 + raised 投影，
    // 无 ambient 光晕——前端 09-07 去炫光治理已删除 glow 层，本端同步删除）
    Column() {
      // Header row：图标 + 标题/范围 + 右箭头
      // 弹性契约（对齐前端 HeroStats）：左侧文本簇 layoutWeight(1) 先收缩，
      // 右箭头 flexShrink(0) 锁定；定位名过长时副标题单行省略，不顶出箭头。
      // （此前靠 Blank 撑开，文本簇无宽度约束，副标题 maxLines(1) 的省略永远触发不了，
      // 长定位名会把箭头顶出卡外——对齐前端同款问题。）
      Row() {
        Row({ space: DesignConstants.SPACING.SM2 }) {
          // brand 圆形图标（对齐前端 size-9.5 圆 + accent 实底，无投影）
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
            Text('实时附近')
              .fontSize(DesignConstants.FONT.SIZE.SM) // 12 对齐前端 text-sm
              .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.SM))
              .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
              .fontColor(this.getCurrentThemePalette().textPrimary)
            // 副标题：定位名 · 范围 · 更新节奏（对齐前端 HeroStats rangeLabel）
            Text(this.getHeroRangeLabel())
              .fontSize(DesignConstants.FONT.SIZE.XS) // 11 对齐前端 text-xs
              .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS))
              .fontColor(this.getCurrentThemePalette().textSecondary)
              .margin({ top: 1 })
              .maxLines(1)
              .textOverflow({ overflow: TextOverflow.Ellipsis })
          }
          .alignItems(HorizontalAlign.Start)
          .layoutWeight(1)
        }
        .alignItems(VerticalAlign.Center)
        .layoutWeight(1)

        Text() {
          LucideIcon({
            icon: AppIcon.ARROW_RIGHT,
            iconSize: DesignConstants.COMPONENT.ICON_SIZE_XS,
            paletteRole: 'textSecondary'
          })
        }
        .flexShrink(0)
      }
      .width('100%')
      .alignItems(VerticalAlign.Center)

      // Stats grid：三段统计 + 竖分割线（对齐前端 bg-border-default gap-px 网格语义）
      // 数字统一 text-2xl(24) bold，对齐前端 display 化 stats（0d528548）
      Row({ space: DesignConstants.SPACING.MD }) {
        // 人附近（对齐前端：显示实际返回条数，limit 12）
        Column({ space: DesignConstants.SPACING.SM2 }) {
          Text(`${this.nearbyUsers.length}`)
            .fontSize(DesignConstants.FONT.SIZE.XXL)
            .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XXL))
            .fontWeight(DesignConstants.FONT.WEIGHT.SEMIBOLD) // 对齐前端 font-semibold(600)
            .fontColor(this.getCurrentThemePalette().textPrimary)
          Text('人附近')
            .fontSize(DesignConstants.FONT.SIZE.XS2) // 10 对齐前端 text-xxs
            .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS2))
            .fontColor(this.getCurrentThemePalette().textSecondary)
        }
        .alignItems(HorizontalAlign.Center)
        .layoutWeight(1)
        .padding({ top: DesignConstants.SPACING.MD, bottom: DesignConstants.SPACING.MD })

        // 分割竖线（Gemini 审查：缩短至 16 + 降不透明度退居幕后）
        Column()
          .width(DesignConstants.BORDER_WIDTH.HAIRLINE)
          .height(40) // 对齐前端 gap-px 网格线(随统计格内容高,40=数字24+标签11+间距6)
          .backgroundColor(this.getCurrentThemePalette().borderColor)
          .opacity(DesignConstants.OPACITY.MUTED)

        // 活动中（对齐前端 heroStats.active = moments.length）
        Column({ space: DesignConstants.SPACING.SM2 }) {
          Text(`${this.nearbyMoments.length}`)
            .fontSize(DesignConstants.FONT.SIZE.XXL)
            .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XXL))
            .fontWeight(DesignConstants.FONT.WEIGHT.SEMIBOLD) // 对齐前端 font-semibold(600)
            .fontColor(this.getCurrentThemePalette().textPrimary)
          Text('动态')
            .fontSize(DesignConstants.FONT.SIZE.XS2)
            .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS2))
            .fontColor(this.getCurrentThemePalette().textSecondary)
        }
        .alignItems(HorizontalAlign.Center)
        .layoutWeight(1)
        .padding({ top: DesignConstants.SPACING.MD, bottom: DesignConstants.SPACING.MD })

        // 分割竖线（Gemini 审查：缩短至 16 + 降不透明度退居幕后）
        Column()
          .width(DesignConstants.BORDER_WIDTH.HAIRLINE)
          .height(40) // 对齐前端 gap-px 网格线(随统计格内容高,40=数字24+标签11+间距6)
          .backgroundColor(this.getCurrentThemePalette().borderColor)
          .opacity(DesignConstants.OPACITY.MUTED)

        // 在线
        Column({ space: DesignConstants.SPACING.SM2 }) {
          Text(`${this.getOnlineCount()}`)
            .fontSize(DesignConstants.FONT.SIZE.XXL)
            .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XXL))
            .fontWeight(DesignConstants.FONT.WEIGHT.SEMIBOLD) // 对齐前端 font-semibold(600)
            .fontColor(this.getCurrentThemePalette().textPrimary)
          Text('在线')
            .fontSize(DesignConstants.FONT.SIZE.XS2)
            .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS2))
            .fontColor(this.getCurrentThemePalette().textSecondary)
        }
        .alignItems(HorizontalAlign.Center)
        .layoutWeight(1)
        .padding({ top: DesignConstants.SPACING.MD, bottom: DesignConstants.SPACING.MD })
      }
      .width('100%')
      .margin({ top: DesignConstants.SPACING.LG })
    }
    .width('100%')
    // 页边距对齐前端 px-5(20)：卡本体不可贴屏幕边缘（此前 100% 宽导致边框贴边不可见）
    // 对齐前端 HeroStats 卡：py-4.5(18, 布局几何豁免) px-5(LG2=20) + rounded-xl(24) + raised-sm 投影
    .padding({
      top: 18,
      bottom: 18,
      left: DesignConstants.SPACING.LG2,
      right: DesignConstants.SPACING.LG2
    })
    .borderRadius(DesignConstants.SHAPE.HERO_CARD)
    .backgroundColor(this.getCurrentThemePalette().surfaceBackground)
    .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getCurrentThemePalette().borderColor })
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
    // 首块与后续区块同节奏：统一底距，使下一个区块标题的上方间距与其他区块一致
    .margin({ bottom: DesignConstants.PAGE_SHELL.SECTION_GAP })
  }

  @Builder
  private buildSectionHeader(title: string, actionText: string, onAction: () => void) {
    Row() {
      Text(title)
        .fontSize(DesignConstants.FONT.SIZE.MD)
        .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.MD))
        // 区块标题字重对齐前端 SectionHead（text-md font-semibold）与首页：
        // 此前用 BOLD(700)，比首页的 SEMIBOLD(600) 更重，同类标题跨页粗细不一致
        .fontWeight(DesignConstants.FONT.WEIGHT.SEMIBOLD)
        .fontColor(this.getCurrentThemePalette().textPrimary)

      Blank()

      Row({ space: DesignConstants.SPACING.XS }) {
        Text(actionText)
          .fontSize(DesignConstants.FONT.SIZE.XS)
          .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS))
          // 页面底色上的链接用 accent 本色（accentText 是「accent 底上的文字色」，
          // 深色主题下为近黑色，直接落在深色页面上不可读）
          .fontColor(this.getCurrentThemePalette().accentBackground)
        LucideIcon({
          icon: AppIcon.ARROW_RIGHT,
          iconSize: DesignConstants.FONT.SIZE.XS,
          paletteRole: 'accent'
        })
      }
      .onClick(onAction)
    }
    .width('100%')
    // 与首页同契约：区块标题 → 其内容用 SECTION_INNER_GAP(12)，
    // 区块之间用 SECTION_GAP(24)，保证两个页面纵向节奏一致
    .margin({ bottom: DesignConstants.PAGE_SHELL.SECTION_INNER_GAP })
  }
}

@Builder
export function NearbyPageBuilder() {
  NearbyPage()
}
