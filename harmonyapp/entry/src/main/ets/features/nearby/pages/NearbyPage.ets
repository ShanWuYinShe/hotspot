/**
 * NearbyPage — 附近综合探索门户页（对齐前端 /nearby 顶级导航 tab）
 *
 * 结构：顶部搜索 + 4 大分区快捷导航芯片 + 身边新朋友横向滑轨 + 身边兴趣圈 + 周边新鲜事动态流。
 * 消除空洞孤立按钮，展示真实周边数据。
 */

import { SystemPageHeader } from '../../../components/SystemPageHeader';
import { SurfaceIconButton } from '../../../components/SurfaceIconButton';
import { LucideIcon } from '../../../components/LucideIcon';
import { curves } from '@kit.ArkUI';
import { PageShellContainer } from '../../../components/PageShellContainer';
import type { BreakpointType } from '../../../core/responsive/Breakpoint';
import { AppRouter, MomentDetailRouteParam } from '../../../core/router/AppRouter';
import { PageNames } from '../../../constants/AppConstants';
import { AppIcon } from '../../../components/SystemIcon';
import { DesignConstants } from '../../../constants/DesignConstants';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { KEY_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';
import { UserItem, GroupItem, MomentItem, SearchUsersRequest, SearchGroupsRequest, FetchNearbyMomentsRequest } from '../../../core/types/api';
import { ApiClient } from '../../../core/api/ApiClient';
import { UserCard } from '../components/UserCard';
import { GroupCard } from '../../groups/components/GroupCard';
import { MomentWallCard } from '../../moments/components/MomentWallCard';
import { EmptyState } from '../../../components/EmptyState';

interface NearbySectionEntry {
  key: string;
  icon: AppIcon;
  title: string;
  description: string;
}

@Preview
@Component
export struct NearbyPage {
  @StorageProp('currentBreakpoint') currentBreakpoint: BreakpointType = 'sm';
  @StorageLink(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'vibrant_light';
  @State private searchEntryPressed: boolean = false;
  @State private pressedSectionKey: string = '';

  @State private nearbyUsers: UserItem[] = [];
  @State private nearbyGroups: GroupItem[] = [];
  @State private nearbyMoments: MomentItem[] = [];
  @State private isLoading: boolean = true;
  @State private loadError: string = '';

  private static readonly SECTION_ENTRIES: NearbySectionEntry[] = [
    { key: 'people', icon: AppIcon.PERSON, title: '附近的人', description: '新朋友' },
    { key: 'groups', icon: AppIcon.GROUP, title: '附近群组', description: '兴趣圈' },
    { key: 'moments', icon: AppIcon.SPARKLES, title: '附近动态', description: '新鲜事' },
    { key: 'map', icon: AppIcon.MAP, title: '地图浏览', description: '全景看' }
  ];

  async aboutToAppear() {
    await this.loadData();
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

  private routeForSection(key: string): string {
    if (key === 'people') return PageNames.NEARBY_USERS;
    if (key === 'groups') return PageNames.NEARBY_GROUPS;
    if (key === 'moments') return PageNames.NEARBY_MOMENTS;
    return PageNames.LOCATION_PICKER;
  }

  build() {
    PageShellContainer({ variant: 'default', currentBreakpoint: this.currentBreakpoint, noHorizontalPadding: true }) {
      Column() {
        SystemPageHeader({
          title: '附近',
          subtitle: '5km 实时探索',
          titleSize: DesignConstants.FONT.SIZE.XXL,
          titleWeight: DesignConstants.FONT.WEIGHT.MEDIUM,
          backgroundVariant: 'transparent'
        }) {
          // trailing: 地图全景入口
          SurfaceIconButton({
            icon: AppIcon.MAP,
            kind: 'neutral',
            onPress: () => AppRouter.push(PageNames.LOCATION_PICKER)
          })
        }

        Scroll() {
          Column({ space: DesignConstants.SPACING.LG }) {
            // 搜索卡片入口
            Row({ space: DesignConstants.SPACING.MD2 }) {
              LucideIcon({
                icon: AppIcon.SEARCH,
                iconSize: DesignConstants.FONT.SIZE.LG,
                paletteRole: 'accent'
              })
              Text('搜索用户、群组、动态、地点…')
                .fontSize(DesignConstants.FONT.SIZE.SM)
                .fontColor(this.getCurrentThemePalette().textSecondary)
                .layoutWeight(1)
                .maxLines(1)
                .textOverflow({ overflow: TextOverflow.Ellipsis })
            }
            .height(DesignConstants.COMPONENT.INPUT_HEIGHT)
            .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG })
            .margin({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG })
            .backgroundColor(this.getCurrentThemePalette().surfaceBackground)
            .borderRadius(DesignConstants.SHAPE.INPUT)
            .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getSurfaceBorder() })
            .shadow(this.getCurrentThemePalette().surfaceShadow)
            .scale(this.searchEntryPressed ?
              { x: DesignConstants.INTERACTION.SCALE_PRESSED_CARD, y: DesignConstants.INTERACTION.SCALE_PRESSED_CARD } :
              { x: 1, y: 1 })
            .animation({ curve: curves.springMotion(
              DesignConstants.INTERACTION.SPRING_RESPONSE_PRESS,
              DesignConstants.INTERACTION.SPRING_DAMPING_PRESS
            ) })
            .onTouch((event: TouchEvent) => {
              if (event.type === TouchType.Down) {
                this.searchEntryPressed = true;
              } else if (event.type === TouchType.Up || event.type === TouchType.Cancel) {
                this.searchEntryPressed = false;
              }
            })
            .onClick(() => { AppRouter.push(PageNames.SEARCH); })

            // 4 大分区快捷入口芯片（紧凑 4 栏网格）
            Grid() {
              ForEach(NearbyPage.SECTION_ENTRIES, (entry: NearbySectionEntry) => {
                GridItem() {
                  this.buildSectionChip(entry)
                }
              }, (entry: NearbySectionEntry) => entry.key)
            }
            .columnsTemplate('1fr 1fr 1fr 1fr')
            .columnsGap(DesignConstants.SPACING.SM)
            .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG })
            .height(76)

            // ═══ 板块一：身边的新朋友 ═══
            if (this.nearbyUsers.length > 0) {
              Column({ space: DesignConstants.SPACING.MD }) {
                this.buildSectionHeader('身边的新朋友', '查看全部', () => {
                  AppRouter.push(PageNames.NEARBY_USERS);
                })

                List({ space: DesignConstants.SPACING.MD }) {
                  ForEach(this.nearbyUsers, (user: UserItem) => {
                    ListItem() {
                      Column() {
                        UserCard({ user: user })
                      }
                      .width(140)
                    }
                  }, (user: UserItem) => user.user?.user_id ?? '')
                }
                .listDirection(Axis.Horizontal)
                .scrollBar(BarState.Off)
                .edgeEffect(EdgeEffect.Spring)
                .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG })
                .height(190)
              }
              .width('100%')
            }

            // ═══ 板块二：身边的兴趣圈 ═══
            if (this.nearbyGroups.length > 0) {
              Column({ space: DesignConstants.SPACING.MD }) {
                this.buildSectionHeader('身边兴趣圈', '探索群组', () => {
                  AppRouter.push(PageNames.NEARBY_GROUPS);
                })

                Column({ space: DesignConstants.SPACING.MD }) {
                  ForEach(this.nearbyGroups.slice(0, 2), (group: GroupItem) => {
                    GroupCard({ group: group, featured: false })
                  }, (group: GroupItem) => group.group?.group_id ?? '')
                }
                .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG })
              }
              .width('100%')
            }

            // ═══ 板块三：周边新鲜事 ═══
            if (this.nearbyMoments.length > 0) {
              Column({ space: DesignConstants.SPACING.MD }) {
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
                .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG })
              }
              .width('100%')
            }

            if (!this.isLoading && this.nearbyUsers.length === 0 && this.nearbyGroups.length === 0 && this.nearbyMoments.length === 0) {
              EmptyState({
                title: '周边暂时还没有动态',
                message: '成为第一个点亮这片区域的人吧！'
              })
              .padding({ top: DesignConstants.SPACING.XL })
            }

            Row()
              .height(DesignConstants.SPACING.XXL)
          }
          .width('100%')
        }
        .width('100%')
        .layoutWeight(1)
        .scrollBar(BarState.Off)
        .edgeEffect(EdgeEffect.Spring)
        .align(Alignment.Top)
      }
      .width('100%')
      .height('100%')
    }
  }

  @Builder
  private buildSectionHeader(title: string, actionText: string, onAction: () => void) {
    Row() {
      Text(title)
        .fontSize(DesignConstants.FONT.SIZE.MD)
        .fontWeight(DesignConstants.FONT.WEIGHT.BOLD)
        .fontColor(this.getCurrentThemePalette().textPrimary)

      Blank()

      Row({ space: DesignConstants.SPACING.XS }) {
        Text(actionText)
          .fontSize(DesignConstants.FONT.SIZE.XS)
          .fontColor(this.getCurrentThemePalette().accentText)
        LucideIcon({
          icon: AppIcon.ARROW_RIGHT,
          iconSize: DesignConstants.FONT.SIZE.XS,
          paletteRole: 'accent'
        })
      }
      .onClick(onAction)
    }
    .width('100%')
    .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG })
  }

  @Builder
  private buildSectionChip(entry: NearbySectionEntry) {
    Column({ space: DesignConstants.SPACING.XS }) {
      Column() {
        LucideIcon({
          icon: entry.icon,
          iconSize: DesignConstants.FONT.SIZE.MD,
          paletteRole: 'accent'
        })
      }
      .width(28)
      .height(28)
      .borderRadius(DesignConstants.RADIUS.FULL)
      .backgroundColor(this.getCurrentThemePalette().accentMuted)
      .justifyContent(FlexAlign.Center)

      Text(entry.title)
        .fontSize(DesignConstants.FONT.SIZE.XS2)
        .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
        .fontColor(this.getCurrentThemePalette().textPrimary)
        .maxLines(1)
      Text(entry.description)
        .fontSize(8)
        .fontColor(this.getCurrentThemePalette().textSecondary)
        .maxLines(1)
    }
    .width('100%')
    .height('100%')
    .padding({ top: DesignConstants.SPACING.SM, bottom: DesignConstants.SPACING.SM })
    .backgroundColor(this.getCurrentThemePalette().surfaceBackground)
    .borderRadius(DesignConstants.SHAPE.CARD)
    .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getSurfaceBorder() })
    .shadow(this.getCurrentThemePalette().surfaceShadow)
    .justifyContent(FlexAlign.Center)
    .scale(this.pressedSectionKey === entry.key ?
      { x: DesignConstants.INTERACTION.SCALE_PRESSED_CARD, y: DesignConstants.INTERACTION.SCALE_PRESSED_CARD } :
      { x: 1, y: 1 })
    .animation({ curve: curves.springMotion(
      DesignConstants.INTERACTION.SPRING_RESPONSE_PRESS,
      DesignConstants.INTERACTION.SPRING_DAMPING_PRESS
    ) })
    .onTouch((event: TouchEvent) => {
      if (event.type === TouchType.Down) {
        this.pressedSectionKey = entry.key;
      } else if (event.type === TouchType.Up || event.type === TouchType.Cancel) {
        this.pressedSectionKey = '';
      }
    })
    .onClick(() => { AppRouter.push(this.routeForSection(entry.key)); })
  }
}

@Builder
export function NearbyPageBuilder() {
  NearbyPage()
}

