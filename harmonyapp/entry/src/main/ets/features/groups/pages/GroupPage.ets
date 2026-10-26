/**
 * 群组页面
 * 使用 HarmonyOS 原生能力 + 系统颜色资源（支持深色模式）
 */

import { GlobalStateManager, KEY_CURRENT_LOCATION, KEY_BROWSE_ANCHOR } from '../../../core/state/GlobalStateManager';
import { LocationInfo, LocationUtil } from '../../../core/location/LocationUtil';
import { ApiClient } from '../../../core/api/ApiClient';
import { PageNames, DefaultConfig } from '../../../constants/AppConstants';
import { formatDateTime } from '../../../core/utils/FormatUtil';
import {
  GroupApplicationItem,
  GroupInviteItem,
  GroupItem,
  FetchJoinedGroupsRequest
} from '../../../core/types/api';
import { SystemAvatar } from '../../../components/SystemAvatar';

import { SystemLoadingView } from '../../../components/SystemLoadingView';
import { PageShellContainer } from '../../../components/PageShellContainer';
import type { BreakpointType } from '../../../core/responsive/Breakpoint';
import { SystemToast } from '../../../components/SystemToast';
import { SystemPageHeader } from '../../../components/SystemPageHeader';
import { SystemButton } from '../../../components/SystemButton';
import { EmptyState, LoadingMoreIndicator, RefreshIndicator } from '../../../components/EmptyState';
import { DesignConstants, textLineHeight } from '../../../constants/DesignConstants';
import { ListEnd } from '../../../components/ListEnd';
import { curves } from '@kit.ArkUI';
import { AppIcon } from '../../../components/SystemIcon';
import { LoadingUtil } from '../../../core/utils/LoadingUtil';
import { BasicDataSource } from '../../../core/utils/BasicDataSource';
import { AppRouter, LoginRouteParam, GroupDetailRouteParam } from '../../../core/router/AppRouter';
import { InteractionStateStore } from '../../../core/state/InteractionStateStore';
import { SystemUiController } from '../../../core/utils/SystemUiController';
import { SafeAreaUtils } from '../../../core/utils/SafeAreaUtils';
import { formatCount } from '../../../core/utils/FormatUtil';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { DEFAULT_RESOLVED_THEME_ID, KEY_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';
import { getErrorMessage } from '../../../core/network/ApiErrorUtil';
import { ListGroup, ListRowDivider } from '../../../components/ListGroup';

@Preview
@Component
export struct GroupPage {
  @StorageProp('currentBreakpoint') currentBreakpoint: BreakpointType = 'sm';
  @StorageLink(KEY_BROWSE_ANCHOR) browseAnchor: LocationInfo | null = null;
  @StorageLink(KEY_CURRENT_LOCATION) currentLocation: LocationInfo | null = null;
  @StorageLink('safeAreaTop') safeAreaTop: number = 0;
  @StorageLink('safeAreaBottom') safeAreaBottom: number = 0;
  @StorageLink(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = DEFAULT_RESOLVED_THEME_ID;

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

  async aboutToAppear() {
    SystemUiController.syncWithCurrentTheme();
    await this.loadMyGroups();
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

  private async loadMyGroups(refresh: boolean = false) {
    if (!refresh) { this.isLoading = true; }
    this.errorMessage = '';

    try {
      const request: FetchJoinedGroupsRequest = {
        pagination: { limit: DefaultConfig.PAGE_SIZE_LARGE }
      };
      const response = await LoadingUtil.withMinimumTime(() => ApiClient.getJoinedGroups(request));
      if (this.isDestroyed) return;

      if (response.code === 0) {
        this.groups = response.data?.items ?? [];
        this.groupsDataSource.setData(this.groups);
        this.hasMore = !!(response.data?.next_cursor);
        this.nextCursor = response.data?.next_cursor ?? '';
      } else {
        this.errorMessage = getErrorMessage(response, '加载群组失败');
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
    if (this.isLoadingMore || !this.hasMore) return;
    this.isLoadingMore = true;
    try {
      const request: FetchJoinedGroupsRequest = {
        pagination: { limit: DefaultConfig.PAGE_SIZE_LARGE, cursor: this.nextCursor }
      };
      const response = await ApiClient.getJoinedGroups(request);
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
        this.invitesError = getErrorMessage(response, '加载邀请失败');
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
          await this.loadMyGroups();
        }
      } else {
        SystemToast.showError(getErrorMessage(response, '处理邀请失败'), this.getUIContext());
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
        this.applicationsError = getErrorMessage(response, '加载申请失败');
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
      const response = await ApiClient.cancelGroupApplication({
        group_id: application.group?.group_id ?? '',
        application_id: application.application_id
      });
      if (response.code === 0) {
        this.myApplications = this.myApplications.filter((item: GroupApplicationItem) => item.application_id !== application.application_id);
        this.applicationsDataSource.setData(this.myApplications);
        SystemToast.showSuccess('已取消申请', this.getUIContext());
      } else {
        SystemToast.showError(getErrorMessage(response, '取消申请失败'), this.getUIContext());
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

    if (group.join_method === 'password') {
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
          await this.loadMyGroups();
        } else if (response.data.status === 'pending_approval') {
          SystemToast.show('已提交申请，等待管理员审批', this.getUIContext());
          await this.loadMyGroups();
        }
      } else if (response.code === 10400) {
        SystemToast.show(getErrorMessage(response, '加入群组失败'), this.getUIContext());
        await this.loadMyGroups();
      } else {
        SystemToast.showError(getErrorMessage(response, '加入群组失败'), this.getUIContext());
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
    NavDestination() {
      PageShellContainer({ variant: 'default', currentBreakpoint: this.currentBreakpoint }) {
        Column() {
          // 标题栏 —— 对齐前端 /me/groups：页头只留标题 + 返回 + 主操作；
          // 邀请/申请下移到内容区入口行。三枚按钮挤在 trailing 会把标题压成「我…」
          SystemPageHeader({
            title: '我的群组',
            titleSize: DesignConstants.FONT.SIZE.XXL,
            titleWeight: DesignConstants.FONT.WEIGHT.MEDIUM,
            showBackButton: true,
            onBack: () => { AppRouter.pop(); },
            backgroundVariant: 'transparent'
          }) {
            SystemButton({
              label: '创建',
              variant: 'primary',
              buttonSize: 'small',
              pill: true,
              onPress: () => { this.createGroup(); }
            })
          }

          // 邀请/申请入口行（页头放不下，收进内容区；作用与原页头按钮一致）
          Row({ space: DesignConstants.SPACING.SM }) {
            SystemButton({
              label: '收到的邀请',
              variant: 'surface',
              buttonSize: 'small',
              onPress: () => { this.loadMyInvites(); }
            })

            SystemButton({
              label: '我的申请',
              variant: 'surface',
              buttonSize: 'small',
              onPress: () => { this.loadMyApplications(); }
            })

            Blank()
          }
          .width('100%')
          // 横向留白由 PageShellContainer gutter 统一提供（页头已改为不自带横向 padding）
          .padding({ bottom: DesignConstants.SPACING.SM })

          // 内容区域
          if (this.isLoading) {
            SystemLoadingView({ message: '正在加载我的群组…' })
              .transition(
                TransitionEffect.OPACITY.animation({ duration: DesignConstants.ANIMATION.DURATION_SHEET, curve: DesignConstants.ANIMATION.CURVE_FAST_OUT_SLOW_IN })
                  .combine(TransitionEffect.move(TransitionEdge.BOTTOM).animation({ duration: DesignConstants.ANIMATION.DURATION_SHEET, curve: DesignConstants.ANIMATION.CURVE_FAST_OUT_SLOW_IN }))
              )
          } else if (this.errorMessage && this.groups.length === 0) {
            EmptyState({
          bare: true, icon: AppIcon.WARNING, message: this.errorMessage, actionLabel: '重试', onAction: () => { this.loadMyGroups(); } })
              .layoutWeight(1)
              .transition(
                TransitionEffect.OPACITY.animation({ duration: DesignConstants.ANIMATION.DURATION_SHEET, curve: DesignConstants.ANIMATION.CURVE_FAST_OUT_SLOW_IN })
                  .combine(TransitionEffect.move(TransitionEdge.BOTTOM).animation({ duration: DesignConstants.ANIMATION.DURATION_SHEET, curve: DesignConstants.ANIMATION.CURVE_FAST_OUT_SLOW_IN }))
              )
          } else if (this.groups.length === 0) {
            Refresh({ refreshing: $$this.isRefreshing, builder: RefreshIndicator }) {
              EmptyState({
          spotlight: true, bare: true, icon: AppIcon.GROUP, message: '还没有加入任何群组\n去附近发现感兴趣的小组吧！' })
            }
            .layoutWeight(1)
            .pullToRefresh(false)
            .refreshOffset(64)
            .onRefreshing(() => { this.loadMyGroups(true); })
            .transition(
              TransitionEffect.OPACITY.animation({ duration: DesignConstants.ANIMATION.DURATION_SHEET, curve: DesignConstants.ANIMATION.CURVE_FAST_OUT_SLOW_IN })
                .combine(TransitionEffect.move(TransitionEdge.BOTTOM).animation({ duration: DesignConstants.ANIMATION.DURATION_SHEET, curve: DesignConstants.ANIMATION.CURVE_FAST_OUT_SLOW_IN }))
            )
          } else {
            Refresh({ refreshing: $$this.isRefreshing, builder: RefreshIndicator }) {
              this.buildGroupList()
            }
            .layoutWeight(1)
            .pullToRefresh(false)
            .refreshOffset(64)
            .onRefreshing(() => { this.loadMyGroups(true); })
            .transition(
              TransitionEffect.OPACITY.animation({ duration: DesignConstants.ANIMATION.DURATION_SHEET, curve: DesignConstants.ANIMATION.CURVE_FAST_OUT_SLOW_IN })
                .combine(TransitionEffect.move(TransitionEdge.BOTTOM).animation({ duration: DesignConstants.ANIMATION.DURATION_SHEET, curve: DesignConstants.ANIMATION.CURVE_FAST_OUT_SLOW_IN }))
            )
          }
        }
        .width('100%')
        .height('100%')
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
    }
    .hideTitleBar(true)
  }

  @Builder
  private buildMyInvitesSheet() {
    Column() {
      Row() {
        Text('我的群邀请')
          .fontSize(DesignConstants.FONT.SIZE.XL)
          .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XL))
          .fontWeight(DesignConstants.FONT.WEIGHT.BOLD)
          .fontColor(this.getCurrentThemePalette().textPrimary)
          .layoutWeight(1)

        Text(`${this.myInvites.length} 条`)
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.SM))
          .fontColor(this.getCurrentThemePalette().textSecondary)
      }
      .width('100%')
      .padding(DesignConstants.SPACING.LG)

      if (this.invitesLoading && this.myInvites.length === 0) {
        SystemLoadingView({ message: '加载邀请中…' }).layoutWeight(1)
      } else if (this.myInvites.length === 0 && this.invitesError) {
        EmptyState({
          bare: true, icon: AppIcon.WARNING, message: this.invitesError, actionLabel: '重试', onAction: () => this.loadMyInvites() })
          .layoutWeight(1)
      } else if (this.myInvites.length === 0) {
        EmptyState({
          bare: true, icon: AppIcon.EMPTY, message: '暂无待处理邀请' }).layoutWeight(1)
      } else {
        List() {
          ListItemGroup() {
            LazyForEach(this.invitesDataSource, (invite: GroupInviteItem) => {
              ListItem() {
                this.buildMyInviteItem(invite)
              }
            }, (invite: GroupInviteItem) => invite.invite_id)
          }
          .divider({
            strokeWidth: DesignConstants.BORDER_WIDTH.HAIRLINE,
            color: this.getCurrentThemePalette().borderColor,
            startMargin: DesignConstants.SPACING.MD,
            endMargin: DesignConstants.SPACING.MD
          })

          if (this.invitesLoadingMore) {
            ListItem() {
              LoadingMoreIndicator({ message: '加载更多邀请…' })
            }
          }
        }
        .divider({ strokeWidth: 0 })
        .width('100%')
        .layoutWeight(1)
        .scrollBar(BarState.Off)
        .cachedCount(5)
        // Sheet 内集合按 ListGroup bordered=false 契约：无外框投影，行间分隔线承担分组
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
    Column({ space: DesignConstants.SPACING.SM }) {
      Row({ space: DesignConstants.SPACING.MD }) {
        SystemAvatar({
          avatarUrl: invite.group?.avatar_url ?? '',
          name: invite.group?.name ?? '群组',
          avatarSize: DesignConstants.COMPONENT.AVATAR_SIZE.SMALL,
          shape: 'squircle'
        })
        Column({ space: DesignConstants.SPACING.HALF }) {
          Text(invite.group?.name ?? '群组')
            .fontSize(DesignConstants.FONT.SIZE.MD)
            .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.MD))
            .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
            .fontColor(this.getCurrentThemePalette().textPrimary)
            .maxLines(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
          Text(`邀请人：${invite.inviter?.nickname ?? '未知用户'}`)
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.SM))
            .fontColor(this.getCurrentThemePalette().textSecondary)
            .maxLines(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
          if (invite.message) {
            Text(invite.message)
              .fontSize(DesignConstants.FONT.SIZE.SM)
              .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.SM))
              .fontColor(this.getCurrentThemePalette().textSecondary)
              .maxLines(2)
              .textOverflow({ overflow: TextOverflow.Ellipsis })
          }
        }
        .layoutWeight(1)
        .alignItems(HorizontalAlign.Start)
      }
      .width('100%')

      Row({ space: DesignConstants.SPACING.SM }) {
        SystemButton({
          label: this.isInviteHandling(invite.invite_id) ? '处理中…' : '接受',
          variant: 'primary',
          buttonSize: 'default',
          pill: true,
          disabled: this.isInviteHandling(invite.invite_id),
          onPress: () => { this.handleInviteAction(invite, 'accept'); }
        })
          .layoutWeight(1)

        SystemButton({
          label: '拒绝',
          variant: 'surface',
          buttonSize: 'default',
          pill: true,
          shadowed: true,
          disabled: this.isInviteHandling(invite.invite_id),
          onPress: () => { this.handleInviteAction(invite, 'reject'); }
        })
          .layoutWeight(1)
      }
      .width('100%')
    }
    .width('100%')
    .padding(DesignConstants.SPACING.MD)
    // 行不再自带卡片样式：分组外框/行间分隔线由 ListGroup 承担
    // （原为每行白卡 + 圆角 + elevatedShadow，连续条目叠成一摞浮块）
  }

  @Builder
  private buildMyApplicationsSheet() {
    Column() {
      Row() {
        Text('我的入群申请')
          .fontSize(DesignConstants.FONT.SIZE.XL)
          .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XL))
          .fontWeight(DesignConstants.FONT.WEIGHT.BOLD)
          .fontColor(this.getCurrentThemePalette().textPrimary)
          .layoutWeight(1)

        Text(`${this.myApplications.length} 条`)
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.SM))
          .fontColor(this.getCurrentThemePalette().textSecondary)
      }
      .width('100%')
      .padding(DesignConstants.SPACING.LG)

      if (this.applicationsLoading && this.myApplications.length === 0) {
        SystemLoadingView({ message: '加载申请中…' }).layoutWeight(1)
      } else if (this.myApplications.length === 0 && this.applicationsError) {
        EmptyState({
          bare: true, icon: AppIcon.WARNING, message: this.applicationsError, actionLabel: '重试', onAction: () => this.loadMyApplications() })
          .layoutWeight(1)
      } else if (this.myApplications.length === 0) {
        EmptyState({
          bare: true, icon: AppIcon.EMPTY, message: '暂无待处理申请' }).layoutWeight(1)
      } else {
        List() {
          ListItemGroup() {
            LazyForEach(this.applicationsDataSource, (application: GroupApplicationItem) => {
              ListItem() {
                this.buildMyApplicationItem(application)
              }
            }, (application: GroupApplicationItem) => application.application_id)
          }
          .divider({
            strokeWidth: DesignConstants.BORDER_WIDTH.HAIRLINE,
            color: this.getCurrentThemePalette().borderColor,
            startMargin: DesignConstants.SPACING.MD,
            endMargin: DesignConstants.SPACING.MD
          })

          if (this.applicationsLoadingMore) {
            ListItem() {
              LoadingMoreIndicator({ message: '加载更多申请…' })
            }
          }
        }
        .divider({ strokeWidth: 0 })
        .width('100%')
        .layoutWeight(1)
        .scrollBar(BarState.Off)
        .cachedCount(5)
        // Sheet 内集合按 ListGroup bordered=false 契约：无外框投影，行间分隔线承担分组；
        // 横向留白只由 PageShellContainer 提供
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
    Column({ space: DesignConstants.SPACING.SM }) {
      Row({ space: DesignConstants.SPACING.MD }) {
        SystemAvatar({
          avatarUrl: '',
          name: '群组申请',
          avatarSize: DesignConstants.COMPONENT.AVATAR_SIZE.SMALL,
          shape: 'squircle'
        })
        Column({ space: DesignConstants.SPACING.HALF }) {
          Text('群组申请')
            .fontSize(DesignConstants.FONT.SIZE.MD)
            .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.MD))
            .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
            .fontColor(this.getCurrentThemePalette().textPrimary)
            .maxLines(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
          if (application.message) {
            Text(application.message)
              .fontSize(DesignConstants.FONT.SIZE.SM)
              .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.SM))
              .fontColor(this.getCurrentThemePalette().textSecondary)
              .maxLines(2)
              .textOverflow({ overflow: TextOverflow.Ellipsis })
          }
          Text(`申请时间：${formatDateTime(application.created_at)}`)
            .fontSize(DesignConstants.FONT.SIZE.XS)
            .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS))
            .fontColor(this.getCurrentThemePalette().textSecondary)
            .maxLines(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
        }
        .layoutWeight(1)
        .alignItems(HorizontalAlign.Start)
      }
      .width('100%')

      Row() {
        SystemButton({
          label: '取消申请',
          variant: 'surface',
          buttonSize: 'small',
          shadowed: true,
          loading: this.isApplicationHandling(application.application_id),
          onPress: () => { this.cancelMyApplication(application); }
        })
          .layoutWeight(1)
      }
      .width('100%')
    }
    .width('100%')
    .padding(DesignConstants.SPACING.MD)
    // 同 buildMyInviteItem：行级卡片样式收敛到 ListGroup 容器
  }

  @Builder
  private buildGroupList() {
    List() {
      ListItemGroup() {
        LazyForEach(this.groupsDataSource, (group: GroupItem) => {
          ListItem() {
            this.buildGroupCard(group)
          }
        }, (group: GroupItem) => group.group?.group_id ?? '')
      }
      .divider({
        strokeWidth: DesignConstants.BORDER_WIDTH.HAIRLINE,
        color: this.getCurrentThemePalette().borderColor,
        startMargin: DesignConstants.SPACING.LG,
        endMargin: DesignConstants.SPACING.LG
      })
      if (this.isLoadingMore) {
        ListItem() {
          LoadingMoreIndicator()
        }
      } else if (!this.hasMore && this.groups.length > 0) {
        ListItem() {
          ListEnd()
        }
      }
    }
    .divider({ strokeWidth: 0 })
    .width('100%')
    .edgeEffect(EdgeEffect.None)
    // 横向留白由外层 PageShellContainer 统一提供，这里不再叠加（否则双重 16vp 缩进）
    .padding({
      top: DesignConstants.SPACING.SM,
      bottom: SafeAreaUtils.resolveBottomPadding(this.safeAreaBottom, DesignConstants.SPACING.LG, 0)
    })
    .scrollBar(BarState.Off)
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

      Column({ space: DesignConstants.SPACING.HALF }) {
        Text(group.group.name)
          .fontSize(DesignConstants.FONT.SIZE.MD)
          .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.MD))
          .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
          .fontColor(this.getCurrentThemePalette().textPrimary)
          .maxLines(1)
          .textOverflow({ overflow: TextOverflow.Ellipsis })

        Row({ space: DesignConstants.SPACING.SM }) {
          Text(`${formatCount(group.group.members_count)}人`)
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.SM))
            .fontColor(this.getCurrentThemePalette().textSecondary)

          if (group.online_members_count !== undefined && group.online_members_count !== null && group.online_members_count > 0) {
            // 在线人数（对齐前端 SectionGroups「N 人在线」语义，用 online 状态色）
            Row({ space: DesignConstants.SPACING.XS }) {
              Row()
                .width(DesignConstants.COMPONENT.INDICATOR_DOT_SIZE)
                .height(DesignConstants.COMPONENT.INDICATOR_DOT_SIZE)
                .borderRadius(DesignConstants.RADIUS.FULL)
                .backgroundColor(this.getCurrentThemePalette().online)
              Text(`${group.online_members_count} 在线`)
                .fontSize(DesignConstants.FONT.SIZE.SM)
                .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.SM))
                .fontColor(this.getCurrentThemePalette().online)
            }
            .alignItems(VerticalAlign.Center)
          }

          if (group.location?.distance !== undefined && group.location?.distance !== null) {
            Text(LocationUtil.formatDistance(group.location.distance))
              .fontSize(DesignConstants.FONT.SIZE.SM)
              .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.SM))
              .fontColor(this.getCurrentThemePalette().textSecondary)
          }
        }

        if (group.description) {
          Text(group.description)
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.SM))
            .fontColor(this.getCurrentThemePalette().textSecondary)
            .maxLines(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
        }
      }
      .alignItems(HorizontalAlign.Start)
      .layoutWeight(1)

      // 本列表为已加入群组（getJoinedGroups），操作为「进入」详情而非「加入」
      SystemButton({
        label: '进入',
        variant: 'surface-accent',
        buttonSize: 'small',
        shadowed: true,
        onPress: () => {
          const detailParams = new GroupDetailRouteParam();
          detailParams.groupId = group.group.group_id;
          AppRouter.push(PageNames.GROUP_DETAIL, detailParams);
        }
      })
    }
    .width('100%')
    .padding({
      left: DesignConstants.SPACING.LG,
      right: DesignConstants.SPACING.LG,
      top: DesignConstants.SPACING.LG,
      bottom: DesignConstants.SPACING.LG
    })
    // 行级卡片样式（圆角 + elevatedShadow + 白底 + 行间距）已移除：
    // 外框与投影归分组容器，行间靠 ListItemGroup.divider
    .alignItems(VerticalAlign.Center)
    // 按压缩放（消费 pressedGroupId 状态；原状态只写不读，卡片点击零按压反馈）
    .scale(this.pressedGroupId === group.group.group_id ?
      { x: DesignConstants.INTERACTION.SCALE_PRESSED_CARD, y: DesignConstants.INTERACTION.SCALE_PRESSED_CARD } :
      { x: 1, y: 1 })
    .animation({ curve: curves.springMotion(
      DesignConstants.INTERACTION.SPRING_RESPONSE_PRESS,
      DesignConstants.INTERACTION.SPRING_DAMPING_PRESS
    ) })
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
  GroupPage()
}