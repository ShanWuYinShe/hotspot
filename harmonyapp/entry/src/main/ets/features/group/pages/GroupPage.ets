/**
 * 群组页面
 * 使用 HarmonyOS 原生能力 + 系统颜色资源（支持深色模式）
 */

import { GlobalStateManager, KEY_CURRENT_LOCATION, KEY_BROWSE_ANCHOR } from '../../../core/state/GlobalStateManager';
import { LocationInfo, LocationUtil } from '../../../core/location/LocationUtil';
import { ApiClient } from '../../../core/api/ApiClient';
import { PageNames, DefaultConfig } from '../../../constants/AppConstants';
import {
  GroupApplicationItem,
  GroupInviteItem,
  GroupItem,
  SearchGroupsRequest
} from '../../../core/types/api';
import { SystemAvatar } from '../../../components/SystemAvatar';

import { SystemLoadingView } from '../../../components/SystemLoadingView';
import { SystemToast } from '../../../components/SystemToast';
import { EmptyState, LoadingMoreIndicator, RefreshIndicator } from '../../../components/EmptyState';
import { DesignConstants } from '../../../constants/DesignConstants';
import { AppIcon } from '../../../components/SystemIcon';
import { LoadingUtil } from '../../../core/utils/LoadingUtil';
import { BasicDataSource } from '../../../core/utils/BasicDataSource';
import { AppRouter, LoginRouteParam, GroupDetailRouteParam } from '../../../core/router/AppRouter';
import { InteractionStateStore } from '../../../core/state/InteractionStateStore';
import { SystemUiController } from '../../../core/utils/SystemUiController';
import { SafeAreaUtils } from '../../../core/utils/SafeAreaUtils';
import { formatCount } from '../../../core/utils/FormatUtil';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { AppThemeVisuals } from '../../../core/theme/AppThemeVisuals';
import { KEY_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';

@Preview
@Component
export struct GroupPage {
  @StorageLink(KEY_BROWSE_ANCHOR) browseAnchor: LocationInfo | null = null;
  @StorageLink(KEY_CURRENT_LOCATION) currentLocation: LocationInfo | null = null;
  @StorageLink('safeAreaTop') safeAreaTop: number = 0;
  @StorageLink('safeAreaBottom') safeAreaBottom: number = 0;
  @StorageLink(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'figma_neu_light';

  private stateManager = GlobalStateManager.getInstance();
  @State groups: GroupItem[] = [];
  private groupsDataSource: BasicDataSource<GroupItem> = new BasicDataSource<GroupItem>([]);
  @State isLoading: boolean = true;
  @State isRefreshing: boolean = false;
  @State errorMessage: string = '';
  @State hasMore: boolean = false;
  @State nextCursor: string = '';
  @State isLoadingMore: boolean = false;

  @State myInvites: GroupInviteItem[] = [];
  private invitesDataSource: BasicDataSource<GroupInviteItem> = new BasicDataSource<GroupInviteItem>([]);
  @State invitesLoading: boolean = false;
  @State invitesError: string = '';
  @State invitesHasMore: boolean = false;
  @State invitesNextCursor: string = '';
  @State invitesLoadingMore: boolean = false;

  @State myApplications: GroupApplicationItem[] = [];
  private applicationsDataSource: BasicDataSource<GroupApplicationItem> = new BasicDataSource<GroupApplicationItem>([]);
  @State applicationsLoading: boolean = false;
  @State applicationsError: string = '';
  @State applicationsHasMore: boolean = false;
  @State applicationsNextCursor: string = '';
  @State applicationsLoadingMore: boolean = false;

  @State showMyInvitesSheet: boolean = false;
  @State showMyApplicationsSheet: boolean = false;
  @State handlingInviteIds: string[] = [];
  @State handlingApplicationIds: string[] = [];
  @State joiningGroupIds: string[] = [];
  @State pressedGroupId: string = '';
  private isDestroyed: boolean = false;

  private getHeaderTopPadding(): number {
    return SafeAreaUtils.resolveTopPadding(this.safeAreaTop);
  }

  async aboutToAppear() {
    SystemUiController.syncWithCurrentTheme();
    await this.loadNearbyGroups();
  }

  private getCurrentThemePalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  private getSurfaceBackground(): string {
    return this.getCurrentThemePalette().surfaceBackground;
  }

  private getSurfaceSubtle(): string {
    return this.getCurrentThemePalette().surfaceSubtle;
  }

  private getSurfaceBorder(): string {
    return this.getCurrentThemePalette().borderColor;
  }

  aboutToDisappear() {
    this.isDestroyed = true;
  }

  /** 获取搜索参考点：优先浏览锚点，回退到当前定位 */
  private getRefLocation(): LocationInfo | null {
    return this.browseAnchor ?? this.currentLocation;
  }

  private ensureLoggedIn(actionName: string): boolean {
    const currentUser = this.stateManager.getCurrentUser();
    if (currentUser) {
      return true;
    }
    SystemToast.showWarning(`请先登录后再${actionName}`, this.getUIContext());
    const loginParams = new LoginRouteParam();
    loginParams.reason = 'action_required';
    loginParams.action = actionName;
    AppRouter.push(PageNames.LOGIN, loginParams);
    return false;
  }

  private closeOverlaySheets(): void {
    this.showMyInvitesSheet = false;
    this.showMyApplicationsSheet = false;
  }

  private isInviteHandling(inviteId: string): boolean {
    return this.handlingInviteIds.includes(inviteId);
  }

  private setInviteHandling(inviteId: string, handling: boolean): void {
    if (handling) {
      if (!this.handlingInviteIds.includes(inviteId)) {
        const nextIds: string[] = [];
        for (const id of this.handlingInviteIds) {
          nextIds.push(id);
        }
        nextIds.push(inviteId);
        this.handlingInviteIds = nextIds;
      }
      return;
    }
    const nextIds: string[] = [];
    for (const id of this.handlingInviteIds) {
      if (id !== inviteId) {
        nextIds.push(id);
      }
    }
    this.handlingInviteIds = nextIds;
  }

  private isApplicationHandling(applicationId: string): boolean {
    return this.handlingApplicationIds.includes(applicationId);
  }

  private setApplicationHandling(applicationId: string, handling: boolean): void {
    if (handling) {
      if (!this.handlingApplicationIds.includes(applicationId)) {
        const nextIds: string[] = [];
        for (const id of this.handlingApplicationIds) {
          nextIds.push(id);
        }
        nextIds.push(applicationId);
        this.handlingApplicationIds = nextIds;
      }
      return;
    }
    const nextIds: string[] = [];
    for (const id of this.handlingApplicationIds) {
      if (id !== applicationId) {
        nextIds.push(id);
      }
    }
    this.handlingApplicationIds = nextIds;
  }

  private isGroupJoining(groupId: string): boolean {
    return this.joiningGroupIds.includes(groupId);
  }

  private setGroupJoining(groupId: string, joining: boolean): void {
    if (joining) {
      if (!this.joiningGroupIds.includes(groupId)) {
        const nextIds: string[] = [];
        for (const id of this.joiningGroupIds) {
          nextIds.push(id);
        }
        nextIds.push(groupId);
        this.joiningGroupIds = nextIds;
      }
      return;
    }
    const nextIds: string[] = [];
    for (const id of this.joiningGroupIds) {
      if (id !== groupId) {
        nextIds.push(id);
      }
    }
    this.joiningGroupIds = nextIds;
  }

  private async loadNearbyGroups(refresh: boolean = false) {
    const refLoc = this.getRefLocation();
    if (!refLoc) {
      this.isLoading = false;
      this.isRefreshing = false;
      return;
    }

    if (!refresh) { this.isLoading = true; }
    this.errorMessage = '';

    try {
      const request: SearchGroupsRequest = {
        reference_point: {
          latitude: refLoc.latitude,
          longitude: refLoc.longitude
        },
        radius: DefaultConfig.NEARBY_RADIUS,
        sort_by: 'distance',
        pagination: { limit: DefaultConfig.PAGE_SIZE_LARGE }
      };
      const response = await LoadingUtil.withMinimumTime(() => ApiClient.searchGroups(request));
      if (this.isDestroyed) return;

      if (response.code === 0) {
        this.groups = response.data?.items ?? [];
        this.groupsDataSource.setData(this.groups);
        this.hasMore = !!(response.data?.next_cursor);
        this.nextCursor = response.data?.next_cursor ?? '';
      } else {
        this.errorMessage = response.message || '加载群组失败';
        this.groups = [];
        this.groupsDataSource.clear();
      }
    } catch (error) {
      this.errorMessage = error instanceof Error ? error.message : '加载群组失败';
    } finally {
      this.isLoading = false;
      this.isRefreshing = false;
    }
  }

  private async loadMoreGroups(): Promise<void> {
    const refLoc = this.getRefLocation();
    if (this.isLoadingMore || !this.hasMore || !refLoc) return;
    this.isLoadingMore = true;
    try {
      const request: SearchGroupsRequest = {
        reference_point: {
          latitude: refLoc.latitude,
          longitude: refLoc.longitude
        },
        radius: DefaultConfig.NEARBY_RADIUS,
        sort_by: 'distance',
        pagination: { limit: DefaultConfig.PAGE_SIZE_LARGE, cursor: this.nextCursor }
      };
      const response = await ApiClient.searchGroups(request);
      if (this.isDestroyed) return;
      if (response.code === 0) {
        const items = response.data?.items ?? [];
        this.groups = this.groups.concat(items);
        this.groupsDataSource.appendData(items);
        this.hasMore = !!(response.data?.next_cursor);
        this.nextCursor = response.data?.next_cursor ?? '';
      }
    } catch (error) {
      // 静默失败，用户可再次触底重试
    } finally {
      this.isLoadingMore = false;
    }
  }

  private async loadMyInvites(): Promise<void> {
    if (!this.ensureLoggedIn('处理群组邀请')) return;
    this.invitesLoading = true;
    this.invitesError = '';
    this.closeOverlaySheets();
    this.showMyInvitesSheet = true;
    try {
      const response = await ApiClient.getGroupInvites({
        status: 'pending',
        pagination: { limit: DefaultConfig.PAGE_SIZE_LARGE }
      });
      if (this.isDestroyed) return;
      if (response.code === 0) {
        this.myInvites = response.data?.items ?? [];
        this.invitesDataSource.setData(this.myInvites);
        this.invitesHasMore = !!(response.data?.next_cursor);
        this.invitesNextCursor = response.data?.next_cursor ?? '';
      } else if (this.myInvites.length === 0) {
        this.invitesError = response.message || '加载邀请失败';
      }
    } catch (error) {
      this.invitesError = error instanceof Error ? error.message : '加载邀请失败';
    } finally {
      this.invitesLoading = false;
    }
  }

  private async loadMoreMyInvites(): Promise<void> {
    if (this.invitesLoadingMore || !this.invitesHasMore) return;
    this.invitesLoadingMore = true;
    try {
      const response = await ApiClient.getGroupInvites({
        status: 'pending',
        pagination: { limit: DefaultConfig.PAGE_SIZE_LARGE, cursor: this.invitesNextCursor }
      });
      if (this.isDestroyed) return;
      if (response.code === 0) {
        const items = response.data?.items ?? [];
        this.myInvites = this.myInvites.concat(items);
        this.invitesDataSource.appendData(items);
        this.invitesHasMore = !!(response.data?.next_cursor);
        this.invitesNextCursor = response.data?.next_cursor ?? '';
      }
    } catch (_) {
      // 静默失败，保留当前列表
    } finally {
      this.invitesLoadingMore = false;
    }
  }

  private async handleInviteAction(invite: GroupInviteItem, action: 'accept' | 'reject'): Promise<void> {
    if (!this.ensureLoggedIn('处理群组邀请')) return;
    if (this.isInviteHandling(invite.invite_id)) return;
    this.setInviteHandling(invite.invite_id, true);
    try {
      const response = action === 'accept'
        ? await ApiClient.acceptGroupInvite({ invite_id: invite.invite_id })
        : await ApiClient.rejectGroupInvite({ invite_id: invite.invite_id });
      if (response.code === 0) {
        this.myInvites = this.myInvites.filter((item: GroupInviteItem) => item.invite_id !== invite.invite_id);
        this.invitesDataSource.setData(this.myInvites);
        SystemToast.showSuccess(action === 'accept' ? '已接受群邀请' : '已拒绝群邀请', this.getUIContext());
        if (action === 'accept') {
          await this.loadNearbyGroups();
        }
      } else {
        SystemToast.showError(response.message || '处理邀请失败', this.getUIContext());
      }
    } catch {
      SystemToast.showError('处理邀请失败', this.getUIContext());
    } finally {
      this.setInviteHandling(invite.invite_id, false);
    }
  }

  private async loadMyApplications(): Promise<void> {
    if (!this.ensureLoggedIn('管理入群申请')) return;
    this.applicationsLoading = true;
    this.applicationsError = '';
    this.closeOverlaySheets();
    this.showMyApplicationsSheet = true;
    try {
      const response = await ApiClient.getMyGroupApplications({
        status: 'pending',
        pagination: { limit: DefaultConfig.PAGE_SIZE_LARGE }
      });
      if (this.isDestroyed) return;
      if (response.code === 0) {
        this.myApplications = response.data?.items ?? [];
        this.applicationsDataSource.setData(this.myApplications);
        this.applicationsHasMore = !!(response.data?.next_cursor);
        this.applicationsNextCursor = response.data?.next_cursor ?? '';
      } else if (this.myApplications.length === 0) {
        this.applicationsError = response.message || '加载申请失败';
      }
    } catch (error) {
      this.applicationsError = error instanceof Error ? error.message : '加载申请失败';
    } finally {
      this.applicationsLoading = false;
    }
  }

  private async loadMoreMyApplications(): Promise<void> {
    if (this.applicationsLoadingMore || !this.applicationsHasMore) return;
    this.applicationsLoadingMore = true;
    try {
      const response = await ApiClient.getMyGroupApplications({
        status: 'pending',
        pagination: { limit: DefaultConfig.PAGE_SIZE_LARGE, cursor: this.applicationsNextCursor }
      });
      if (response.code === 0) {
        const items = response.data?.items ?? [];
        this.myApplications = this.myApplications.concat(items);
        this.applicationsDataSource.appendData(items);
        this.applicationsHasMore = !!(response.data?.next_cursor);
        this.applicationsNextCursor = response.data?.next_cursor ?? '';
      }
    } catch (_) {
      // 静默失败
    } finally {
      this.applicationsLoadingMore = false;
    }
  }

  private async cancelMyApplication(application: GroupApplicationItem): Promise<void> {
    if (!this.ensureLoggedIn('取消入群申请')) return;
    if (this.isApplicationHandling(application.application_id)) return;
    this.setApplicationHandling(application.application_id, true);
    try {
      const response = await ApiClient.cancelGroupApplication({ application_id: application.application_id });
      if (response.code === 0) {
        this.myApplications = this.myApplications.filter((item: GroupApplicationItem) => item.application_id !== application.application_id);
        this.applicationsDataSource.setData(this.myApplications);
        SystemToast.showSuccess('已取消申请', this.getUIContext());
      } else {
        SystemToast.showError(response.message || '取消申请失败', this.getUIContext());
      }
    } catch {
      SystemToast.showError('取消申请失败', this.getUIContext());
    } finally {
      this.setApplicationHandling(application.application_id, false);
    }
  }

  private async joinGroup(group: GroupItem) {
    const groupId = group.group.group_id;
    if (this.isGroupJoining(groupId)) {
      return;
    }
    const currentUser = this.stateManager.getCurrentUser();
    if (!currentUser) {
      SystemToast.showWarning('请先登录后再加入群组', this.getUIContext());
      const loginParams = new LoginRouteParam();
      loginParams.reason = 'action_required';
      loginParams.action = '加入群组';
      AppRouter.push(PageNames.LOGIN, loginParams);
      return;
    }

    if (group.has_password) {
      SystemToast.show('此群组需要密码，请进入详情页加入', this.getUIContext());
      const detailParams = new GroupDetailRouteParam();
      detailParams.groupId = groupId;
      AppRouter.push(PageNames.GROUP_DETAIL, detailParams);
      return;
    }

    this.setGroupJoining(groupId, true);
    try {
      const response = await ApiClient.joinGroup({ group_id: groupId });
      if (response.code === 0 && response.data) {
        if (response.data.status === 'joined') {
          SystemToast.show('加入群组成功', this.getUIContext());
          InteractionStateStore.getInstance().updateGroupMembership(groupId, { is_member: true });
          await this.loadNearbyGroups();
        } else if (response.data.status === 'pending_approval') {
          SystemToast.show('已提交申请，等待管理员审批', this.getUIContext());
          await this.loadNearbyGroups();
        }
      } else if (response.code === 10400) {
        SystemToast.show(response.message || '加入群组失败', this.getUIContext());
        await this.loadNearbyGroups();
      } else {
        SystemToast.showError(response.message || '加入群组失败', this.getUIContext());
      }
    } catch (error) {
      const errMsg = error instanceof Error ? error.message : '加入群组失败';
      SystemToast.showError(errMsg, this.getUIContext());
    } finally {
      this.setGroupJoining(groupId, false);
    }
  }


  private async createGroup() {
    const currentUser = this.stateManager.getCurrentUser();
    if (!currentUser) {
      SystemToast.showWarning('请先登录后再创建群组', this.getUIContext());
      const loginParams = new LoginRouteParam();
      loginParams.reason = 'action_required';
      loginParams.action = '创建群组';
      AppRouter.push(PageNames.LOGIN, loginParams);
      return;
    }

    AppRouter.push(PageNames.CREATE_GROUP);
  }

  build() {
    Column() {
      // 标题栏 - 使用 SafeArea 避免与状态栏重叠
      Row() {
        Text('附近群组')
          .fontSize(DesignConstants.FONT.SIZE.XL)
          .fontWeight(FontWeight.Medium)
          .fontColor(this.getCurrentThemePalette().textPrimary)
          .layoutWeight(1)

        Row({ space: 8 }) {
          Button('邀请')
            .type(ButtonType.Capsule)
            .backgroundColor(this.getSurfaceSubtle())
            .fontColor(this.getCurrentThemePalette().textPrimary)
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .height(36)
            .shadow(this.getCurrentThemePalette().controlShadow)
            .onClick(() => this.loadMyInvites())

          Button('申请')
            .type(ButtonType.Capsule)
            .backgroundColor(this.getSurfaceSubtle())
            .fontColor(this.getCurrentThemePalette().textPrimary)
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .height(36)
            .shadow(this.getCurrentThemePalette().controlShadow)
            .onClick(() => this.loadMyApplications())

          Button('创建')
            .type(ButtonType.Capsule)
            .backgroundColor(this.getCurrentThemePalette().accentBackground)
            .fontColor(this.getCurrentThemePalette().accentText)
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .height(36)
            .shadow(this.getCurrentThemePalette().controlShadow)
            .onClick(() => this.createGroup())
        }
      }
      .width('100%')
      .alignItems(VerticalAlign.Center)
      .padding({
        left: DesignConstants.SPACING.LG,
        right: DesignConstants.SPACING.LG,
        top: this.getHeaderTopPadding(),
        bottom: DesignConstants.SPACING.SM
      })
      .backgroundColor(Color.Transparent)
      .expandSafeArea([SafeAreaType.SYSTEM], [SafeAreaEdge.TOP])

      // 内容区域
      if (this.isLoading) {
        SystemLoadingView({ message: '正在加载附近群组...' })
          .transition(
            TransitionEffect.OPACITY.animation({ duration: 320, curve: Curve.FastOutSlowIn })
              .combine(TransitionEffect.move(TransitionEdge.BOTTOM).animation({ duration: 320, curve: Curve.FastOutSlowIn }))
          )
      } else if (this.errorMessage && this.groups.length === 0) {
        EmptyState({ icon: AppIcon.WARNING, message: this.errorMessage, actionLabel: '重试', onAction: () => { this.loadNearbyGroups(); } })
          .layoutWeight(1)
          .transition(
            TransitionEffect.OPACITY.animation({ duration: 320, curve: Curve.FastOutSlowIn })
              .combine(TransitionEffect.move(TransitionEdge.BOTTOM).animation({ duration: 320, curve: Curve.FastOutSlowIn }))
          )
      } else if (this.groups.length === 0) {
        Refresh({ refreshing: $$this.isRefreshing, builder: RefreshIndicator }) {
          EmptyState({ icon: AppIcon.GROUP, message: '附近暂无群组\n快来创建第一个吧！' })
        }
        .onRefreshing(() => { this.loadNearbyGroups(true); })
        .transition(
          TransitionEffect.OPACITY.animation({ duration: 320, curve: Curve.FastOutSlowIn })
            .combine(TransitionEffect.move(TransitionEdge.BOTTOM).animation({ duration: 320, curve: Curve.FastOutSlowIn }))
        )
      } else {
        Refresh({ refreshing: $$this.isRefreshing, builder: RefreshIndicator }) {
          this.buildGroupList()
        }
        .layoutWeight(1)
        .onRefreshing(() => { this.loadNearbyGroups(true); })
        .transition(
          TransitionEffect.OPACITY.animation({ duration: 320, curve: Curve.FastOutSlowIn })
            .combine(TransitionEffect.move(TransitionEdge.BOTTOM).animation({ duration: 320, curve: Curve.FastOutSlowIn }))
        )
      }
    }
    .width('100%')
    .height('100%')
    .backgroundColor(this.getCurrentThemePalette().pageBackground)
    .bindSheet($$this.showMyInvitesSheet, this.buildMyInvitesSheet(), {
      height: '68%',
      showClose: false,
      dragBar: true,
      backgroundColor: this.getCurrentThemePalette().sheetBackground
    })
    .bindSheet($$this.showMyApplicationsSheet, this.buildMyApplicationsSheet(), {
      height: '68%',
      showClose: false,
      dragBar: true,
      backgroundColor: this.getCurrentThemePalette().sheetBackground
    })
  }

  @Builder
  private buildMyInvitesSheet() {
    Column() {
      Row() {
        Text('我的群邀请')
          .fontSize(DesignConstants.FONT.SIZE.XL)
          .fontWeight(FontWeight.Bold)
          .fontColor(this.getCurrentThemePalette().textPrimary)
          .layoutWeight(1)

        Text(`${this.myInvites.length} 条`)
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .fontColor(this.getCurrentThemePalette().textSecondary)
      }
      .width('100%')
      .padding(DesignConstants.SPACING.LG)

      if (this.invitesLoading && this.myInvites.length === 0) {
        SystemLoadingView({ message: '加载邀请中...' }).layoutWeight(1)
      } else if (this.myInvites.length === 0 && this.invitesError) {
        EmptyState({ icon: AppIcon.WARNING, message: this.invitesError, actionLabel: '重试', onAction: () => this.loadMyInvites() })
          .layoutWeight(1)
      } else if (this.myInvites.length === 0) {
        EmptyState({ icon: AppIcon.EMPTY, message: '暂无待处理邀请' }).layoutWeight(1)
      } else {
        List() {
          LazyForEach(this.invitesDataSource, (invite: GroupInviteItem) => {
            ListItem() {
              this.buildMyInviteItem(invite)
            }
          }, (invite: GroupInviteItem) => invite.invite_id)

          if (this.invitesLoadingMore) {
            ListItem() {
              LoadingMoreIndicator({ message: '加载更多邀请...' })
            }
          }
        }
        .divider({ strokeWidth: 0 })
        .width('100%')
        .layoutWeight(1)
        .scrollBar(BarState.Auto)
        .cachedCount(5)
        .onReachEnd(() => { this.loadMoreMyInvites(); })
      }
    }
    .padding({ bottom: SafeAreaUtils.resolveBottomPadding(this.safeAreaBottom, DesignConstants.SPACING.LG, 0) })
    .width('100%')
    .height('100%')
    .backgroundColor(this.getCurrentThemePalette().pageBackground)
  }

  @Builder
  private buildMyInviteItem(invite: GroupInviteItem) {
    Column({ space: 8 }) {
      Row({ space: DesignConstants.SPACING.MD }) {
        SystemAvatar({
          avatarUrl: invite.group?.avatar_url ?? '',
          name: invite.group?.name ?? '群组',
          avatarSize: 40,
          shape: 'squircle'
        })
        Column({ space: 2 }) {
          Text(invite.group?.name ?? '群组')
            .fontSize(DesignConstants.FONT.SIZE.MD)
            .fontWeight(FontWeight.Medium)
            .fontColor(this.getCurrentThemePalette().textPrimary)
            .maxLines(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
          Text(`邀请人：${invite.inviter?.nickname ?? '未知用户'}`)
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .fontColor(this.getCurrentThemePalette().textSecondary)
            .maxLines(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
          if (invite.message) {
            Text(invite.message)
              .fontSize(DesignConstants.FONT.SIZE.SM)
              .fontColor(this.getCurrentThemePalette().textSecondary)
              .maxLines(2)
              .textOverflow({ overflow: TextOverflow.Ellipsis })
          }
        }
        .layoutWeight(1)
        .alignItems(HorizontalAlign.Start)
      }
      .width('100%')

      Row({ space: 8 }) {
        Button(this.isInviteHandling(invite.invite_id) ? '处理中...' : '接受')
          .type(ButtonType.Capsule)
          .backgroundColor(this.getCurrentThemePalette().accentBackground)
          .fontColor(this.getCurrentThemePalette().accentText)
          .shadow(this.getCurrentThemePalette().controlShadow)
          .border({ width: 1, color: this.getSurfaceBorder() })
          .height(38)
          .layoutWeight(1)
          .enabled(!this.isInviteHandling(invite.invite_id))
          .opacity(this.isInviteHandling(invite.invite_id) ? DesignConstants.OPACITY.LOADING : 1)
          .onClick(() => this.handleInviteAction(invite, 'accept'))

        Button('拒绝')
          .type(ButtonType.Capsule)
          .backgroundColor(this.getSurfaceSubtle())
          .fontColor(this.getCurrentThemePalette().textPrimary)
          .shadow(this.getCurrentThemePalette().controlShadow)
          .border({ width: 1, color: this.getSurfaceBorder() })
          .height(38)
          .layoutWeight(1)
          .enabled(!this.isInviteHandling(invite.invite_id))
          .opacity(this.isInviteHandling(invite.invite_id) ? DesignConstants.OPACITY.LOADING : 1)
          .onClick(() => this.handleInviteAction(invite, 'reject'))
      }
      .width('100%')
    }
    .width('100%')
    .padding(DesignConstants.SPACING.MD)
    .borderRadius(AppThemeVisuals.getListItemCardStyle(this.resolvedThemeId).borderRadius)
    .shadow(this.getCurrentThemePalette().elevatedShadow)
    .backgroundColor(this.getSurfaceBackground())
    .border({ width: 1, color: this.getSurfaceBorder() })
    .margin({ bottom: 12 })
  }

  @Builder
  private buildMyApplicationsSheet() {
    Column() {
      Row() {
        Text('我的入群申请')
          .fontSize(DesignConstants.FONT.SIZE.XL)
          .fontWeight(FontWeight.Bold)
          .fontColor(this.getCurrentThemePalette().textPrimary)
          .layoutWeight(1)

        Text(`${this.myApplications.length} 条`)
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .fontColor(this.getCurrentThemePalette().textSecondary)
      }
      .width('100%')
      .padding(DesignConstants.SPACING.LG)

      if (this.applicationsLoading && this.myApplications.length === 0) {
        SystemLoadingView({ message: '加载申请中...' }).layoutWeight(1)
      } else if (this.myApplications.length === 0 && this.applicationsError) {
        EmptyState({ icon: AppIcon.WARNING, message: this.applicationsError, actionLabel: '重试', onAction: () => this.loadMyApplications() })
          .layoutWeight(1)
      } else if (this.myApplications.length === 0) {
        EmptyState({ icon: AppIcon.EMPTY, message: '暂无待处理申请' }).layoutWeight(1)
      } else {
        List() {
          LazyForEach(this.applicationsDataSource, (application: GroupApplicationItem) => {
            ListItem() {
              this.buildMyApplicationItem(application)
            }
          }, (application: GroupApplicationItem) => application.application_id)

          if (this.applicationsLoadingMore) {
            ListItem() {
              LoadingMoreIndicator({ message: '加载更多申请...' })
            }
          }
        }
        .divider({ strokeWidth: 0 })
        .width('100%')
        .layoutWeight(1)
        .scrollBar(BarState.Auto)
        .cachedCount(5)
        .onReachEnd(() => { this.loadMoreMyApplications(); })
      }
    }
    .padding({ bottom: SafeAreaUtils.resolveBottomPadding(this.safeAreaBottom, DesignConstants.SPACING.LG, 0) })
    .width('100%')
    .height('100%')
    .backgroundColor(this.getCurrentThemePalette().pageBackground)
  }

  @Builder
  private buildMyApplicationItem(application: GroupApplicationItem) {
    Column({ space: 8 }) {
      Row({ space: DesignConstants.SPACING.MD }) {
        SystemAvatar({
          avatarUrl: '',
          name: '群组申请',
          avatarSize: 40,
          shape: 'squircle'
        })
        Column({ space: 2 }) {
          Text('群组申请')
            .fontSize(DesignConstants.FONT.SIZE.MD)
            .fontWeight(FontWeight.Medium)
            .fontColor(this.getCurrentThemePalette().textPrimary)
            .maxLines(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
          if (application.message) {
            Text(application.message)
              .fontSize(DesignConstants.FONT.SIZE.SM)
              .fontColor(this.getCurrentThemePalette().textSecondary)
              .maxLines(2)
              .textOverflow({ overflow: TextOverflow.Ellipsis })
          }
          Text(`申请时间：${new Date(application.created_at).toLocaleString()}`)
            .fontSize(DesignConstants.FONT.SIZE.XS)
            .fontColor(this.getCurrentThemePalette().textSecondary)
            .maxLines(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
        }
        .layoutWeight(1)
        .alignItems(HorizontalAlign.Start)
      }
      .width('100%')

      Row() {
        Button(this.isApplicationHandling(application.application_id) ? '处理中...' : '取消申请')
          .type(ButtonType.Capsule)
          .backgroundColor(this.getCurrentThemePalette().surfaceSubtle)
          .fontColor(this.getCurrentThemePalette().textPrimary)
          .shadow(this.getCurrentThemePalette().controlShadow)
          .border({ width: 1, color: this.getCurrentThemePalette().borderColor })
          .height(38)
          .layoutWeight(1)
          .enabled(!this.isApplicationHandling(application.application_id))
          .opacity(this.isApplicationHandling(application.application_id) ? DesignConstants.OPACITY.LOADING : 1)
          .onClick(() => this.cancelMyApplication(application))
      }
      .width('100%')
    }
    .width('100%')
    .padding(DesignConstants.SPACING.MD)
    .borderRadius(AppThemeVisuals.getListItemCardStyle(this.resolvedThemeId).borderRadius)
    .shadow(this.getCurrentThemePalette().elevatedShadow)
    .backgroundColor(this.getCurrentThemePalette().surfaceBackground)
    .border({ width: 1, color: this.getCurrentThemePalette().borderColor })
    .margin({ bottom: 12 })
  }

  @Builder
  private buildGroupList() {
    List() {
      LazyForEach(this.groupsDataSource, (group: GroupItem) => {
        ListItem() {
          this.buildGroupCard(group)
        }
      }, (group: GroupItem) => group.group?.group_id ?? '')
      if (this.isLoadingMore) {
        ListItem() {
          LoadingMoreIndicator()
        }
      }
    }
    .divider({ strokeWidth: 0 })
    .width('100%')
    .padding({
      left: DesignConstants.SPACING.LG,
      right: DesignConstants.SPACING.LG,
      top: DesignConstants.SPACING.SM,
      bottom: SafeAreaUtils.resolveBottomPadding(this.safeAreaBottom, DesignConstants.SPACING.LG, 0)
    })
    .scrollBar(BarState.Auto)
    .cachedCount(5)
    .onReachEnd(() => { this.loadMoreGroups(); })
  }

  @Builder
  private buildGroupCard(group: GroupItem) {
    Row({ space: DesignConstants.SPACING.MD }) {
      SystemAvatar({
        avatarUrl: group.group.avatar_url || '',
        name: group.group.name,
        avatarSize: DesignConstants.COMPONENT.AVATAR_SIZE.MEDIUM,
        shape: 'squircle'
      })

      Column({ space: 2 }) {
        Text(group.group.name)
          .fontSize(DesignConstants.FONT.SIZE.LG)
          .fontWeight(FontWeight.Medium)
          .fontColor(this.getCurrentThemePalette().textPrimary)
          .maxLines(1)
          .textOverflow({ overflow: TextOverflow.Ellipsis })

        Row({ space: DesignConstants.SPACING.SM }) {
          Text(`${formatCount(group.group.members_count)}人`)
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .fontColor(this.getCurrentThemePalette().textSecondary)

          if (group.location?.distance !== undefined && group.location?.distance !== null) {
            Text(LocationUtil.formatDistance(group.location.distance))
              .fontSize(DesignConstants.FONT.SIZE.SM)
              .fontColor(this.getCurrentThemePalette().textSecondary)
          }
        }

        if (group.description) {
          Text(group.description)
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .fontColor(this.getCurrentThemePalette().textSecondary)
            .maxLines(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
        }
      }
      .alignItems(HorizontalAlign.Start)
      .layoutWeight(1)

      Button(this.isGroupJoining(group.group.group_id) ? '处理中...' : '加入')
        .type(ButtonType.Capsule)
        .backgroundColor(this.getCurrentThemePalette().surfaceSubtle)
        .fontColor(this.getCurrentThemePalette().accentBackground)
        .fontSize(DesignConstants.FONT.SIZE.SM)
        .fontWeight(FontWeight.Medium)
        .height(34)
        .shadow(this.getCurrentThemePalette().controlShadow)
        .border({ width: 1, color: this.getCurrentThemePalette().borderColor })
        .enabled(!this.isGroupJoining(group.group.group_id))
        .opacity(this.isGroupJoining(group.group.group_id) ? DesignConstants.OPACITY.LOADING : 1)
        .onClick(() => this.joinGroup(group))
    }
    .width('100%')
    .padding({
      left: DesignConstants.SPACING.LG,
      right: DesignConstants.SPACING.LG,
      top: DesignConstants.SPACING.LG,
      bottom: DesignConstants.SPACING.LG
    })
    .borderRadius(AppThemeVisuals.getListItemCardStyle(this.resolvedThemeId).borderRadius)
    .shadow(this.getCurrentThemePalette().elevatedShadow)
    .backgroundColor(this.getCurrentThemePalette().surfaceBackground)
    .margin({ bottom: 12 })
    .alignItems(VerticalAlign.Center)
    .onTouch((event: TouchEvent) => {
      if (event.type === TouchType.Down) {
        this.pressedGroupId = group.group.group_id;
      } else if (event.type === TouchType.Up || event.type === TouchType.Cancel) {
        this.pressedGroupId = '';
      }
    })
    .onClick(() => {
      const detailParams = new GroupDetailRouteParam();
      detailParams.groupId = group.group.group_id;
      AppRouter.push(PageNames.GROUP_DETAIL, detailParams);
    })
  }
}

@Builder
export function GroupPageBuilder() {
  NavDestination() {
    GroupPage()
  }
  .hideTitleBar(true)
}
