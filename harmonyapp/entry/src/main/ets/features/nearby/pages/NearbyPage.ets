/**
 * NearbyPage — 附近聚合页（对齐前端 /nearby 顶级导航 tab）
 *
 * 结构：搜索入口 + 附近的人 / 附近群组 / 附近动态三个预览区块，
 * 各区块「查看全部」跳对应专区（NEARBY_USERS / NEARBY_GROUPS）。
 * 数据复用首页同一组接口（searchUsers / searchGroups / searchMoments），
 * 与前端 fetchHomeFeed 信息架构一致。
 */

import { SystemPageHeader } from '../../../components/SystemPageHeader';
import { SurfaceIconButton } from '../../../components/SurfaceIconButton';
import { ResponsiveContainer } from '../../../components/ResponsiveContainer';
import { SystemAvatar } from '../../../components/SystemAvatar';
import { SystemLoadingView } from '../../../components/SystemLoadingView';
import { EmptyState } from '../../../components/EmptyState';
import { Pressable } from '../../../components/Pressable';
import { AppIcon } from '../../../components/SystemIcon';
import { ApiClient } from '../../../core/api/ApiClient';
import { AppRouter, ProfileRouteParam, GroupDetailRouteParam, MomentDetailRouteParam } from '../../../core/router/AppRouter';
import { PageNames, DefaultConfig } from '../../../constants/AppConstants';
import { KEY_CURRENT_LOCATION } from '../../../core/state/GlobalStateManager';
import { LocationInfo, LocationUtil } from '../../../core/location/LocationUtil';
import { DesignConstants } from '../../../constants/DesignConstants';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { KEY_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';
import { LoadingUtil } from '../../../core/utils/LoadingUtil';
import {
  SearchUsersRequest,
  SearchGroupsRequest,
  SearchMomentsRequest,
  UserItem,
  GroupItem,
  MomentItem,
  GeoPoint
} from '../../../core/types/api';

const DOMAIN_NUMBER: number = 0xFF2A;
const TAG: string = 'NearbyPage';

@Preview
@Component
export struct NearbyPage {
  @StorageLink(KEY_CURRENT_LOCATION) currentLocation: LocationInfo | null = null;
  @StorageLink(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'vibrant_light';
  @StorageLink('safeAreaTop') safeAreaTop: number = 0;

  @State users: UserItem[] = [];
  @State groups: GroupItem[] = [];
  @State moments: MomentItem[] = [];
  @State isLoading: boolean = true;
  @State errorMessage: string = '';

  private isDestroyed: boolean = false;
  private loadVersion: number = 0;

  private getCurrentThemePalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  private getSurfaceBorder(): string {
    return this.getCurrentThemePalette().borderColor;
  }

  private getReferencePoint(): GeoPoint | null {
    const loc: LocationInfo | null = this.currentLocation;
    if (!loc) return null;
    return { latitude: loc.latitude, longitude: loc.longitude };
  }

  aboutToAppear() {
    this.loadData();
  }

  aboutToDisappear() {
    this.isDestroyed = true;
  }

  private async loadData(): Promise<void> {
    const capturedVersion = ++this.loadVersion;
    this.isLoading = true;
    this.errorMessage = '';
    try {
      const ref = this.getReferencePoint();
      if (!ref) {
        if (!this.isDestroyed && capturedVersion === this.loadVersion) {
          this.isLoading = false;
          this.errorMessage = '正在获取位置…';
        }
        return;
      }
      const results = await LoadingUtil.withMinimumTime(() =>
        Promise.allSettled([
          ApiClient.searchUsers({ reference_point: ref, radius: DefaultConfig.NEARBY_RADIUS, sort_by: 'distance', pagination: { limit: 4 } } as SearchUsersRequest),
          ApiClient.searchGroups({ reference_point: ref, radius: DefaultConfig.NEARBY_RADIUS, sort_by: 'distance', pagination: { limit: 4 } } as SearchGroupsRequest),
          ApiClient.searchMoments({ reference_point: ref, radius: DefaultConfig.NEARBY_RADIUS, sort_by: 'distance', pagination: { limit: 3 } } as SearchMomentsRequest)
        ])
      );
      if (this.isDestroyed || capturedVersion !== this.loadVersion) return;

      const usersResult = results[0];
      const groupsResult = results[1];
      const momentsResult = results[2];

      if (usersResult.status === 'fulfilled' && usersResult.value.code === 0) {
        this.users = (usersResult.value.data?.items ?? []).filter((u: UserItem) => u.user?.user_id);
      }
      if (groupsResult.status === 'fulfilled' && groupsResult.value.code === 0) {
        this.groups = (groupsResult.value.data?.items ?? []).filter((g: GroupItem) => g.group?.group_id);
      }
      if (momentsResult.status === 'fulfilled' && momentsResult.value.code === 0) {
        this.moments = (momentsResult.value.data?.items ?? []).filter((m: MomentItem) => m.moment_id);
      }
      if (this.users.length === 0 && this.groups.length === 0 && this.moments.length === 0) {
        this.errorMessage = '附近暂时没有内容';
      }
    } catch {
      if (!this.isDestroyed && capturedVersion === this.loadVersion) {
        this.errorMessage = '加载附近内容失败';
      }
    } finally {
      if (!this.isDestroyed && capturedVersion === this.loadVersion) {
        this.isLoading = false;
      }
    }
  }

  private openProfile(userId: string): void {
    const params = new ProfileRouteParam();
    params.userId = userId;
    AppRouter.push(PageNames.PROFILE, params);
  }

  private openGroup(groupId: string): void {
    const params = new GroupDetailRouteParam();
    params.groupId = groupId;
    AppRouter.push(PageNames.GROUP_DETAIL, params);
  }

  private openMoment(momentId: string): void {
    const params = new MomentDetailRouteParam();
    params.momentId = momentId;
    AppRouter.push(PageNames.MOMENT_DETAIL, params);
  }

  @Builder
  private buildSectionHeader(title: string, actionLabel: string, onAction: () => void) {
    Row() {
      Text(title)
        .fontSize(DesignConstants.FONT.SIZE.SM_PLUS)
        .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
        .fontColor(this.getCurrentThemePalette().textPrimary)
        .layoutWeight(1)
      Text(actionLabel)
        .fontSize(DesignConstants.FONT.SIZE.SM)
        .fontColor(this.getCurrentThemePalette().accentBackground)
        .onClick(onAction)
    }
    .width('100%')
    .padding({
      left: DesignConstants.SPACING.LG,
      right: DesignConstants.SPACING.LG,
      top: DesignConstants.SPACING.LG2,
      bottom: DesignConstants.SPACING.MD
    })
  }

  build() {
    ResponsiveContainer() {
      Column() {
        SystemPageHeader({
          title: '附近',
          subtitle: `${this.users.length} 人 · ${this.groups.length} 群`,
          titleSize: DesignConstants.FONT.SIZE.XL_PLUS,
          titleWeight: DesignConstants.FONT.WEIGHT.MEDIUM,
          backgroundVariant: 'transparent'
        }) {
          // trailing: 地图入口（移动/平板补齐地图入口，对齐前端 PageHeader trailing）
          SurfaceIconButton({
            icon: AppIcon.MAP,
            kind: 'neutral',
            onPress: () => AppRouter.push(PageNames.LOCATION_PICKER)
          })
        }

        // 搜索入口（对齐前端 /nearby 搜索卡片）
        Row({ space: DesignConstants.SPACING.MD2 }) {
          Text() {
            SymbolSpan($r(AppIcon.SEARCH))
              .fontSize(DesignConstants.FONT.SIZE.MD_PLUS)
              .fontColor([this.getCurrentThemePalette().textSecondary])
          }
          Text('搜索用户、群组、动态、地点…')
            .fontSize(DesignConstants.FONT.SIZE.SM_PLUS)
            .fontColor(this.getCurrentThemePalette().textSecondary)
            .layoutWeight(1)
            .maxLines(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
        }
        .height(DesignConstants.COMPONENT.INPUT_HEIGHT)
        .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG })
        .margin({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG })
        .backgroundColor(this.getCurrentThemePalette().inputBackgroundColor)
        .borderRadius(DesignConstants.SHAPE.INPUT)
        .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getSurfaceBorder() })
        .onClick(() => { AppRouter.push(PageNames.SEARCH); })

        if (this.isLoading) {
          SystemLoadingView({ message: '正在加载附近内容…' })
            .layoutWeight(1)
        } else if (this.errorMessage && this.users.length === 0 && this.groups.length === 0 && this.moments.length === 0) {
          EmptyState({
            icon: AppIcon.WARNING,
            message: this.errorMessage,
            actionLabel: '重试',
            onAction: () => { this.loadData(); }
          })
          .layoutWeight(1)
        } else {
          Scroll() {
            Column() {
              // 附近的人
              this.buildSectionHeader('附近的人', '查看全部', () => { AppRouter.push(PageNames.NEARBY_USERS); })
              ForEach(this.users, (user: UserItem) => {
                this.buildUserRow(user)
              }, (user: UserItem) => user.user?.user_id ?? '')

              // 附近群组
              this.buildSectionHeader('附近群组', '查看全部', () => { AppRouter.push(PageNames.NEARBY_GROUPS); })
              ForEach(this.groups, (group: GroupItem) => {
                this.buildGroupRow(group)
              }, (group: GroupItem) => group.group?.group_id ?? '')

              // 附近动态
              this.buildSectionHeader('附近动态', '查看全部', () => { AppRouter.push(PageNames.NEARBY_MOMENTS); })
              ForEach(this.moments, (moment: MomentItem) => {
                this.buildMomentRow(moment)
              }, (moment: MomentItem) => moment.moment_id)

              Row()
                .height(DesignConstants.SPACING.XXL)
            }
            .width('100%')
            .padding({ bottom: DesignConstants.SPACING.LG })
          }
          .width('100%')
          .layoutWeight(1)
          .scrollBar(BarState.Auto)
          .edgeEffect(EdgeEffect.Spring)
          .align(Alignment.Top)
        }
      }
      .width('100%')
      .height('100%')
      .backgroundColor(this.getCurrentThemePalette().pageBackground)
    }
  }

  @Builder
  private buildUserRow(user: UserItem) {
    Pressable({ onPress: () => {
      const userId = user.user?.user_id ?? '';
      if (userId) this.openProfile(userId);
    } }) {
      Row({ space: DesignConstants.SPACING.MD }) {
        SystemAvatar({
          avatarUrl: user.user?.avatar_url ?? '',
          name: user.user?.nickname ?? '',
          avatarSize: DesignConstants.COMPONENT.AVATAR_SIZE.SMALL_PLUS,
          shape: 'circle'
        })
        Column({ space: DesignConstants.SPACING.HALF }) {
          Text(user.user?.nickname ?? '')
            .fontSize(DesignConstants.FONT.SIZE.SM_PLUS)
            .fontColor(this.getCurrentThemePalette().textPrimary)
            .maxLines(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
          Text(`${user.tags?.[0] ?? '附近'} · ${LocationUtil.formatDistance(user.location?.distance ?? 0)}`)
            .fontSize(DesignConstants.FONT.SIZE.XS)
            .fontColor(this.getCurrentThemePalette().textSecondary)
            .maxLines(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
        }
        .layoutWeight(1)
        .alignItems(HorizontalAlign.Start)
      }
      .width('100%')
      .padding({
        left: DesignConstants.SPACING.LG,
        right: DesignConstants.SPACING.LG,
        top: DesignConstants.SPACING.SM2,
        bottom: DesignConstants.SPACING.SM2
      })
    }
  }

  @Builder
  private buildGroupRow(group: GroupItem) {
    Pressable({ onPress: () => {
      const groupId = group.group?.group_id ?? '';
      if (groupId) this.openGroup(groupId);
    } }) {
      Row({ space: DesignConstants.SPACING.MD }) {
        SystemAvatar({
          avatarUrl: group.group?.avatar_url ?? '',
          name: group.group?.name ?? '',
          avatarSize: DesignConstants.COMPONENT.AVATAR_SIZE.SMALL_PLUS,
          shape: 'squircle'
        })
        Column({ space: DesignConstants.SPACING.HALF }) {
          Text(group.group?.name ?? '')
            .fontSize(DesignConstants.FONT.SIZE.SM_PLUS)
            .fontColor(this.getCurrentThemePalette().textPrimary)
            .maxLines(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
          Text(`${group.group?.members_count ?? 0} 位成员 · ${LocationUtil.formatDistance(group.location?.distance ?? 0)}`)
            .fontSize(DesignConstants.FONT.SIZE.XS)
            .fontColor(this.getCurrentThemePalette().textSecondary)
            .maxLines(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
        }
        .layoutWeight(1)
        .alignItems(HorizontalAlign.Start)
      }
      .width('100%')
      .padding({
        left: DesignConstants.SPACING.LG,
        right: DesignConstants.SPACING.LG,
        top: DesignConstants.SPACING.SM2,
        bottom: DesignConstants.SPACING.SM2
      })
    }
  }

  @Builder
  private buildMomentRow(moment: MomentItem) {
    Pressable({ onPress: () => {
      this.openMoment(moment.moment_id);
    } }) {
      Column({ space: DesignConstants.SPACING.XS }) {
        Text(moment.description)
          .fontSize(DesignConstants.FONT.SIZE.SM_PLUS)
          .fontColor(this.getCurrentThemePalette().textPrimary)
          .maxLines(2)
          .textOverflow({ overflow: TextOverflow.Ellipsis })
          .width('100%')
        Text(moment.location?.location_name ?? '附近')
          .fontSize(DesignConstants.FONT.SIZE.XS)
          .fontColor(this.getCurrentThemePalette().textSecondary)
          .width('100%')
      }
      .width('100%')
      .alignItems(HorizontalAlign.Start)
      .padding({
        left: DesignConstants.SPACING.LG,
        right: DesignConstants.SPACING.LG,
        top: DesignConstants.SPACING.SM2,
        bottom: DesignConstants.SPACING.SM2
      })
    }
  }
}

@Builder
export function NearbyPageBuilder() {
  NearbyPage()
}
