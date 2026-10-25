/**
 * 群组详情页面
 * 重构：使用 HarmonyOS 原生 List 组件
 */

import { GlobalStateManager, KEY_CURRENT_USER } from '../../../core/state/GlobalStateManager';
import { curves } from '@kit.ArkUI';
import { AccountInfo } from '../../../core/types/api';
import { ApiClient } from '../../../core/api/ApiClient';
import { PageNames } from '../../../constants/AppConstants';
import { SystemAvatar } from '../../../components/SystemAvatar';
import { GroupMemberRow } from '../components/GroupMemberRow';
import { GroupApplicationRow } from '../components/GroupApplicationRow';
import { AnnouncementEditorSheet } from '../components/AnnouncementEditorSheet';
import { GroupMemberListSheet } from '../components/GroupMemberListSheet';
import { GroupApplicationListSheet } from '../components/GroupApplicationListSheet';
import { PageShellContainer } from '../../../components/PageShellContainer';
import type { BreakpointType } from '../../../core/responsive/Breakpoint';
import { SystemLoadingView } from '../../../components/SystemLoadingView';
import { EmptyState } from '../../../components/EmptyState';
import { SystemPageHeader } from '../../../components/SystemPageHeader';
import { SystemToast } from '../../../components/SystemToast';
import { SystemButton } from '../../../components/SystemButton';
import { SystemDialog } from '../../../components/SystemDialog';
import { SystemInput } from '../../../components/SystemInput';
import { SystemTextArea } from '../../../components/SystemTextArea';
import { SystemToggle } from '../../../components/SystemToggle';
import { AppIcon } from '../../../components/SystemIcon';
import { SurfaceIconButton } from '../../../components/SurfaceIconButton';
import { ShareActionHelper } from '../../../core/utils/ShareActionHelper';
import { SystemShareService } from '../../../core/services/SystemShareService';
import { common } from '@kit.AbilityKit';
import pasteboard from '@ohos.pasteboard';
import { DesignConstants, lineHeightFor, textLineHeight } from '../../../constants/DesignConstants';
import {
  GroupItem,
  GroupApplicationItem,
  GroupMemberItem,
  GroupSharePayload,
  GroupAnnouncementItem,
  RelativeLocation,
  SettableGroupRole,
  UpdateGroupRequest,
  AdminBannedMemberItem,
  UserItem,
  CreateReportRequest
} from '../../../core/types/api';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { InteractionStateStore } from '../../../core/state/InteractionStateStore';
import { SmartNotificationService } from '../../../core/services/SmartNotificationService';
import { AccessibilityManager } from '../../../core/accessibility/AccessibilityManager';
import { AuthAccountManager } from '../../../core/state/AuthAccountManager';
import { LoadingUtil } from '../../../core/utils/LoadingUtil';
import { BasicDataSource } from '../../../core/utils/BasicDataSource';
import { NfcShareService } from '../../../core/services/NfcShareService';
import { AppRouter, LoginRouteParam, ChatRouteParam } from '../../../core/router/AppRouter';
import { SystemUiController } from '../../../core/utils/SystemUiController';
import { SafeAreaUtils } from '../../../core/utils/SafeAreaUtils';
import { MediaServiceManager } from '../../../core/media/MediaServiceManager';
import { formatCount, formatRelativeTime } from '../../../core/utils/FormatUtil';
import { LocationUtil } from '../../../core/location/LocationUtil';
import { KeyboardDismissManager } from '../../../core/utils/KeyboardDismissManager';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { DEFAULT_RESOLVED_THEME_ID, KEY_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';
import { getErrorMessage } from '../../../core/network/ApiErrorUtil';
import { ListRowDivider } from '../../../components/ListGroup';
import { ErrorCode } from '../../../core/constants/ErrorCodes';
import { LucideIcon } from '../../../components/LucideIcon';
import { TabsPillGroup } from '../../../components/TabsPillGroup';
import { AppThemeVisuals } from '../../../core/theme/AppThemeVisuals';


const DOMAIN_NUMBER: number = 0xFF17;
const TAG: string = 'GroupDetailPage';
const GROUP_NAME_MAX: number = 50;
const GROUP_DESCRIPTION_MAX: number = 500;
// 群公告字数上限（对齐前端 GroupAnnouncementDialog：标题 50 / 内容 1000）
// 禁言时长档（分钟）
const MUTE_DURATION_MINUTES: number = 30;

interface GroupDetailParams {
  groupId: string;
}

interface MemberActionMenuItem {
  value: string;
  action: () => void;
}

@Preview
@Component
export struct GroupDetailPage {
  @StorageProp('currentBreakpoint') currentBreakpoint: BreakpointType = 'sm';
  pathStack: NavPathStack = new NavPathStack();
  @StorageLink(KEY_CURRENT_USER) currentUser: AccountInfo | null = null;
  @StorageLink('safeAreaTop') safeAreaTop: number = 0;
  @StorageLink('safeAreaBottom') safeAreaBottom: number = 0;
  @StorageLink(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = DEFAULT_RESOLVED_THEME_ID;
  
  private stateManager = GlobalStateManager.getInstance();
  private smartNotificationService = SmartNotificationService.getInstance();
  private accessibilityManager = AccessibilityManager.getInstance();
  @State groupInfo: GroupItem | null = null;
  @State members: GroupMemberItem[] = [];
  @State private pressedEntryKey: string = '';
  private membersDataSource: BasicDataSource<GroupMemberItem> = new BasicDataSource<GroupMemberItem>([]);
  @State applications: GroupApplicationItem[] = [];
  private applicationsDataSource: BasicDataSource<GroupApplicationItem> = new BasicDataSource<GroupApplicationItem>([]);
  @State isLoading: boolean = true;
  @State isLoadingMembers: boolean = false;
  @State isLoadingApplications: boolean = false;
  @State membersErrorMessage: string = '';
  @State applicationsErrorMessage: string = '';
  @State errorMessage: string = '';
  @State showMemberList: boolean = false;
  @State showApplicationList: boolean = false;
  @State membersHasMore: boolean = false;
  @State membersNextCursor: string = '';
  @State membersLoadingMore: boolean = false;
  @State applicationsHasMore: boolean = false;
  @State applicationsNextCursor: string = '';
  @State applicationsLoadingMore: boolean = false;
  @State isJoining: boolean = false;
  @State isLeaving: boolean = false;
  /** 离开群组二次确认显隐（对齐 GroupCard / Web 同页：危险操作防误触）。 */
  @State showLeaveConfirm: boolean = false;
  @State joinPassword: string = '';
  private groupId: string = '';
  @State handlingApplicationIds: string[] = [];

  @State isMember: boolean = false;
  /** 内容 Tab（对齐前端 groups/[groupId] 三 Tab：简介/成员/公告） */
  @State activeTab: number = 0;
  /** 群公告列表（对齐前端 AnnouncementsList） */
  @State announcements: GroupAnnouncementItem[] = [];
  @State showEditGroupSheet: boolean = false;
  @State editGroupName: string = '';
  @State editGroupDescription: string = '';
  @State editGroupAvatarPreviewUrl: string = '';
  @State editGroupAvatarLocalUri: string = '';
  @State editGroupAvatarFileId: string = '';
  @State isSavingGroupInfo: boolean = false;
  @State isEditGroupAvatarUploading: boolean = false;
  // 群设置扩展（对齐前端 GroupSettingsSheet：私密/新人审核开关）
  @State editGroupPrivate: boolean = false;
  @State editGroupRequiresApproval: boolean = false;
  // 群公告管理（对齐前端 GroupAnnouncementDialog：owner/admin 可创建/编辑/删除）
  @State showAnnouncementSheet: boolean = false;
  @State announcementEditing: GroupAnnouncementItem | null = null;
  @State announcementTitle: string = '';
  @State announcementContent: string = '';
  @State announcementPinned: boolean = false;
  @State isSavingAnnouncement: boolean = false;
  @State showAnnouncementDeleteConfirm: boolean = false;
  @State deleteAnnouncementTarget: GroupAnnouncementItem | null = null;
  // 邀请成员（对齐前端 InviteMembersSheet：搜索 + 多选 + 批量邀请）
  @State showInviteSheet: boolean = false;
  @State inviteKeyword: string = '';
  @State inviteResults: UserItem[] = [];
  @State isSearchingInviteUsers: boolean = false;
  @State inviteSearchError: string = '';
  @State inviteSelectedIds: string[] = [];
  @State isInviting: boolean = false;
  private inviteSearchTimer: number = -1;
  private inviteSearchSeq: number = 0;
  /** 转让群主：候选成员（非群主、非自己，对齐前端 transferableMembers） */
  @State transferSectionOpen: boolean = false;
  @State transferCandidates: GroupMemberItem[] = [];
  @State isLoadingTransferCandidates: boolean = false;
  // 封禁管理（对齐前端 banned 子 Tab：名单 + 解封）
  @State showBannedListSheet: boolean = false;
  /** 更多操作 sheet + 举报确认（对齐前端「更多操作」bottom sheet：封面右上角 ⋯） */
  @State showGroupMoreSheet: boolean = false;
  @State showGroupReportConfirm: boolean = false;
  @State groupReportSubmitting: boolean = false;
  @State bannedMembers: AdminBannedMemberItem[] = [];
  @State isLoadingBannedMembers: boolean = false;
  /** 封禁二次确认目标（危险操作防误触，对齐前端 MemberActionSheet ban 语义） */
  @State showBanConfirm: boolean = false;
  @State banTargetMember: GroupMemberItem | null = null;
  /** 转让群主确认目标（仅群主；对齐前端危险区 ConfirmDialog 语义） */
  @State showTransferConfirm: boolean = false;
  @State transferTargetMember: GroupMemberItem | null = null;
  /** 解散群组二次确认（仅群主；不可逆，对齐前端危险区） */
  @State showDismissConfirm: boolean = false;
  @State isDismissingGroup: boolean = false;
  @State isTransferringOwnership: boolean = false;

  aboutToAppear(): void {
    SystemUiController.syncWithCurrentTheme();
  }

  aboutToDisappear(): void {
    if (this.inviteSearchTimer !== -1) {
      clearTimeout(this.inviteSearchTimer);
      this.inviteSearchTimer = -1;
    }
    KeyboardDismissManager.getInstance().dismiss();
  }

  private getCurrentThemePalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  /** 封面返回按钮顶部安全区避让（对齐 SystemPageHeader 内聚逻辑） */
  private getHeaderTopPadding(): number {
    // extra 0 对齐 SystemPageHeader：页头内容贴状态栏下缘（2026-10-10 布局审查）。
    return SafeAreaUtils.resolveTopPadding(this.safeAreaTop, 0);
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

  private ensureLoggedIn(actionName: string): boolean {
    if (this.currentUser) return true;

    SystemToast.showWarning(`请先登录后再${actionName}`, this.getUIContext());
    const loginParams = new LoginRouteParam();
    loginParams.reason = 'action_required';
    loginParams.action = actionName;
    AppRouter.push(PageNames.LOGIN, loginParams);
    return false;
  }

  private closeOverlaySheets(): void {
    this.showMemberList = false;
    this.showApplicationList = false;
    this.showEditGroupSheet = false;
    this.showAnnouncementSheet = false;
    this.showInviteSheet = false;
    this.showBannedListSheet = false;
  }

  private async loadGroupDetail(): Promise<void> {
    this.isLoading = true;
    this.errorMessage = '';

    try {
      await LoadingUtil.withMinimumTime(async () => {
        const response = await ApiClient.getGroupDetail({
          group_id: this.groupId
        });

        if (response.code === 0 && response.data) {
          this.groupInfo = response.data;
          this.isMember = this.groupInfo.viewer_state != null;
          await this.accessibilityManager.announceText(`已加载群组：${this.groupInfo.group?.name ?? ''}`);
          // 公告 Tab 数据（对齐前端 AnnouncementsList）
          this.loadGroupAnnouncements();
        } else {
          this.errorMessage = getErrorMessage(response, '加载群组详情失败');
        }
      });
    } catch {
      this.errorMessage = '加载群组详情失败';
    } finally {
      this.isLoading = false;
    }
  }

  /** 加载群公告（对齐前端 AnnouncementsList；spec: /public/v1/groups/announcements/fetch） */
  private async loadGroupAnnouncements(): Promise<void> {
    if (!this.groupId) return;
    try {
      const response = await ApiClient.getGroupAnnouncements({ group_id: this.groupId });
      if (response.code === 0) {
        this.announcements = response.data?.items ?? [];
      }
    } catch (error) {
      // 静默失败：公告 Tab 显示空态，不阻塞主流程
    }
  }

  // ==================== 群公告管理（对齐前端 GroupAnnouncementDialog，owner/admin） ====================

  /** 公告管理权限（对齐前端 canManageAnnouncements = joined && (owner|admin)） */
  private canManageAnnouncements(): boolean {
    return this.isMember && this.isAdminOrOwnerRole();
  }

  private openCreateAnnouncement(): void {
    if (!this.canManageAnnouncements()) {
      SystemToast.showWarning('仅群主或管理员可管理群公告', this.getUIContext());
      return;
    }
    this.announcementEditing = null;
    this.announcementTitle = '';
    this.announcementContent = '';
    this.announcementPinned = false;
    this.closeOverlaySheets();
    this.showAnnouncementSheet = true;
  }

  private openEditAnnouncement(item: GroupAnnouncementItem): void {
    if (!this.canManageAnnouncements()) {
      SystemToast.showWarning('仅群主或管理员可管理群公告', this.getUIContext());
      return;
    }
    this.announcementEditing = item;
    this.announcementTitle = item.title;
    this.announcementContent = item.content;
    this.announcementPinned = item.is_pinned;
    this.closeOverlaySheets();
    this.showAnnouncementSheet = true;
  }

  private closeAnnouncementSheet(): void {
    if (this.isSavingAnnouncement) {
      return;
    }
    this.showAnnouncementSheet = false;
    this.announcementEditing = null;
  }

  private async submitAnnouncement(): Promise<void> {
    if (!this.ensureLoggedIn('管理群公告')) return;
    if (!this.canManageAnnouncements()) {
      SystemToast.showWarning('无权限管理群公告', this.getUIContext());
      return;
    }
    const title = this.announcementTitle.trim();
    const content = this.announcementContent.trim();
    if (!title || !content) {
      SystemToast.showWarning('公告标题和内容不能为空', this.getUIContext());
      return;
    }
    const editing = this.announcementEditing;
    this.isSavingAnnouncement = true;
    try {
      if (editing === null) {
        const response = await ApiClient.createGroupAnnouncement({
          group_id: this.groupId,
          title: title,
          content: content,
          is_pinned: this.announcementPinned
        });
        if (response.code !== 0) {
          SystemToast.showError(getErrorMessage(response, '发布失败'), this.getUIContext());
          return;
        }
      } else {
        const response = await ApiClient.updateGroupAnnouncement({
          group_id: this.groupId,
          announcement_id: editing.announcement_id,
          title: title,
          content: content,
          is_pinned: this.announcementPinned
        });
        if (response.code !== 0) {
          SystemToast.showError(getErrorMessage(response, '保存失败'), this.getUIContext());
          return;
        }
      }
      SystemToast.showSuccess(editing === null ? '公告已发布' : '公告已更新', this.getUIContext());
      this.showAnnouncementSheet = false;
      this.announcementEditing = null;
      await this.loadGroupAnnouncements();
    } catch {
      SystemToast.showError('保存失败，请重试', this.getUIContext());
    } finally {
      this.isSavingAnnouncement = false;
    }
  }

  private requestDeleteAnnouncement(item: GroupAnnouncementItem): void {
    if (!this.canManageAnnouncements()) {
      SystemToast.showWarning('无权限管理群公告', this.getUIContext());
      return;
    }
    this.deleteAnnouncementTarget = item;
    this.showAnnouncementDeleteConfirm = true;
  }

  private async handleDeleteAnnouncement(): Promise<void> {
    const target = this.deleteAnnouncementTarget;
    this.showAnnouncementDeleteConfirm = false;
    this.deleteAnnouncementTarget = null;
    if (target === null) {
      return;
    }
    try {
      const response = await ApiClient.deleteGroupAnnouncement({
        group_id: this.groupId,
        announcement_id: target.announcement_id
      });
      if (response.code === 0) {
        SystemToast.showSuccess('公告已删除', this.getUIContext());
        await this.loadGroupAnnouncements();
      } else {
        SystemToast.showError(getErrorMessage(response, '删除失败'), this.getUIContext());
      }
    } catch {
      SystemToast.showError('删除失败，请重试', this.getUIContext());
    }
  }

  private async loadGroupMembers(): Promise<void> {
    if (!this.groupInfo) return;

    this.isLoadingMembers = true;
    this.membersErrorMessage = '';
    this.closeOverlaySheets();
    this.showMemberList = true;
    try {
      await LoadingUtil.withMinimumTime(async () => {
        const response = await ApiClient.getGroupMembers({
          group_id: this.groupId,
          pagination: { limit: 100 }
        });

        if (response.code === 0) {
          this.members = response.data?.items ?? [];
          this.membersDataSource.setData(this.members);
          this.membersHasMore = !!(response.data?.next_cursor);
          this.membersNextCursor = response.data?.next_cursor ?? '';
        } else {
          if (this.members.length === 0) {
            this.membersErrorMessage = getErrorMessage(response, '加载成员列表失败');
          }
        }
      });
    } catch {
      if (this.members.length === 0) {
        this.membersErrorMessage = '加载成员列表失败';
      }
    } finally {
      this.isLoadingMembers = false;
    }
  }

  private async loadMoreMembers(): Promise<void> {
    if (this.membersLoadingMore || !this.membersHasMore) return;
    this.membersLoadingMore = true;
    try {
      const response = await ApiClient.getGroupMembers({
        group_id: this.groupId,
        pagination: { limit: 100, cursor: this.membersNextCursor }
      });
      if (response.code === 0) {
        const items = response.data?.items ?? [];
        this.members = this.members.concat(items);
        this.membersDataSource.appendData(items);
        this.membersHasMore = !!(response.data?.next_cursor);
        this.membersNextCursor = response.data?.next_cursor ?? '';
      }
    } catch {
      // 静默失败
    } finally {
      this.membersLoadingMore = false;
    }
  }

  private canManageApplications(): boolean {
    return this.isMember && this.isAdminOrOwnerRole();
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

  private async loadGroupApplications(): Promise<void> {
    if (!this.canManageApplications()) {
      return;
    }
    this.isLoadingApplications = true;
    this.applicationsErrorMessage = '';
    this.closeOverlaySheets();
    this.showApplicationList = true;
    try {
      const response = await ApiClient.getGroupApplications({
        group_id: this.groupId,
        status: 'pending',
        pagination: { limit: 50 }
      });
      if (response.code === 0) {
        this.applications = response.data?.items ?? [];
        this.applicationsDataSource.setData(this.applications);
        this.applicationsHasMore = !!(response.data?.next_cursor);
        this.applicationsNextCursor = response.data?.next_cursor ?? '';
      } else if (this.applications.length === 0) {
        this.applicationsErrorMessage = getErrorMessage(response, '加载入群申请失败');
      }
    } catch {
      if (this.applications.length === 0) {
        this.applicationsErrorMessage = '加载入群申请失败';
      }
    } finally {
      this.isLoadingApplications = false;
    }
  }

  private async loadMoreGroupApplications(): Promise<void> {
    if (this.applicationsLoadingMore || !this.applicationsHasMore || !this.canManageApplications()) {
      return;
    }
    this.applicationsLoadingMore = true;
    try {
      const response = await ApiClient.getGroupApplications({
        group_id: this.groupId,
        status: 'pending',
        pagination: { limit: 50, cursor: this.applicationsNextCursor }
      });
      if (response.code === 0) {
        const items = response.data?.items ?? [];
        this.applications = this.applications.concat(items);
        this.applicationsDataSource.appendData(items);
        this.applicationsHasMore = !!(response.data?.next_cursor);
        this.applicationsNextCursor = response.data?.next_cursor ?? '';
      }
    } catch {
      // 静默失败
    } finally {
      this.applicationsLoadingMore = false;
    }
  }

  private removeApplicationById(applicationId: string): void {
    this.applications = this.applications.filter((item) => item.application_id !== applicationId);
    this.applicationsDataSource.setData(this.applications);
  }

  private async approveApplication(application: GroupApplicationItem): Promise<void> {
    if (!this.ensureLoggedIn('审核入群申请')) return;
    if (!this.canManageApplications()) {
      SystemToast.showWarning('无权限处理入群申请', this.getUIContext());
      return;
    }
    const applicationId = application.application_id;
    if (!applicationId || this.isApplicationHandling(applicationId)) return;
    this.setApplicationHandling(applicationId, true);
    try {
      const response = await ApiClient.approveGroupApplication({
        group_id: this.groupInfo?.group.group_id ?? '',
        application_id: applicationId
      });
      if (response.code === 0) {
        this.removeApplicationById(applicationId);
        if (this.groupInfo?.group) {
          this.groupInfo.group.members_count = (this.groupInfo.group.members_count ?? 0) + 1;
        }
        SystemToast.showSuccess('已批准入群申请', this.getUIContext());
      } else {
        SystemToast.showError(getErrorMessage(response, '批准申请失败'), this.getUIContext());
      }
    } catch {
      SystemToast.showError('批准申请失败', this.getUIContext());
    } finally {
      this.setApplicationHandling(applicationId, false);
    }
  }

  private async rejectApplication(application: GroupApplicationItem): Promise<void> {
    if (!this.ensureLoggedIn('审核入群申请')) return;
    if (!this.canManageApplications()) {
      SystemToast.showWarning('无权限处理入群申请', this.getUIContext());
      return;
    }
    const applicationId = application.application_id;
    if (!applicationId || this.isApplicationHandling(applicationId)) return;
    this.setApplicationHandling(applicationId, true);
    try {
      const response = await ApiClient.rejectGroupApplication({
        group_id: this.groupInfo?.group.group_id ?? '',
        application_id: applicationId
      });
      if (response.code === 0) {
        this.removeApplicationById(applicationId);
        SystemToast.showSuccess('已拒绝入群申请', this.getUIContext());
      } else {
        SystemToast.showError(getErrorMessage(response, '拒绝申请失败'), this.getUIContext());
      }
    } catch {
      SystemToast.showError('拒绝申请失败', this.getUIContext());
    } finally {
      this.setApplicationHandling(applicationId, false);
    }
  }

  private async joinGroup(): Promise<void> {
    if (!this.groupInfo) return;
    if (!this.ensureLoggedIn('加入群组')) return;

    if (this.groupInfo.join_method === 'password' && !this.joinPassword.trim()) {
      SystemToast.showWarning('请输入群组密码', this.getUIContext());
      return;
    }

    const groupInfoRef = this.groupInfo;
    this.isJoining = true;
    try {
      await LoadingUtil.withMinimumTime(async () => {
        const response = await ApiClient.joinGroup({
          group_id: this.groupId,
          password: groupInfoRef.join_method === 'password' ? this.joinPassword.trim() : undefined
        });

        if (response.code === 0 && response.data) {
          if (response.data.status === 'joined') {
            SystemToast.showSuccess('加入群组成功', this.getUIContext());
            this.isMember = true;
            InteractionStateStore.getInstance().updateGroupMembership(this.groupId, { is_member: true });
            await this.loadGroupDetail();
          } else if (response.data.status === 'pending_approval') {
            SystemToast.showSuccess('已提交申请，等待管理员审批', this.getUIContext());
          }
        } else if (response.code === ErrorCode.BUSINESS_CONFLICT && (response.message ?? '').includes('已是群成员')) {
          // 10400 为通用业务冲突（密码错误/群已满/已申请过/仅限邀请均为此码），
          // 仅「已是群成员」语义放行进入聊天，其余按错误展示、不跳转
          SystemToast.show('你已是群成员', this.getUIContext());
          this.isMember = true;
          InteractionStateStore.getInstance().updateGroupMembership(this.groupId, { is_member: true });
          await this.enterChat();
        } else {
          SystemToast.showError(getErrorMessage(response, '加入群组失败'), this.getUIContext());
        }
      }, LoadingUtil.DEFAULT_ACTION_MS);
    } catch {
      SystemToast.showError('加入群组失败', this.getUIContext());
    } finally {
      this.isJoining = false;
    }
  }

  private async leaveGroup(): Promise<void> {
    if (!this.groupInfo) return;
    if (!this.ensureLoggedIn('离开群组')) return;

    this.isLeaving = true;
    try {
      await LoadingUtil.withMinimumTime(async () => {
        const response = await ApiClient.leaveGroup({
          group_id: this.groupId
        });

        if (response.code === 0) {
          SystemToast.showSuccess('已离开群组', this.getUIContext());
          this.isMember = false;
          InteractionStateStore.getInstance().updateGroupMembership(this.groupId, { is_member: false });
          AppRouter.pop();
        } else {
          SystemToast.showError(getErrorMessage(response, '离开群组失败'), this.getUIContext());
        }
      }, LoadingUtil.DEFAULT_ACTION_MS);
    } catch {
      SystemToast.showError('离开群组失败', this.getUIContext());
    } finally {
      this.isLeaving = false;
    }
  }

  private async shareGroup(): Promise<void> {
    if (!this.groupInfo) return;
    try {
      const context = this.getUIContext().getHostContext() as common.UIAbilityContext;
      const groupName = this.groupInfo.group?.name ?? '群组';
      const payload: GroupSharePayload = { shared_group_id: this.groupId };
      await ShareActionHelper.shareResource(
        context,
        'group',
        payload,
        `加入群组「${groupName}」`,
        this.groupInfo.description ?? '在 HotSpot 发现有趣的附近群组'
      );
    } catch {
      // 用户取消分享
    }
  }

  private canEditGroupInfo(): boolean {
    return this.isMember && this.isAdminOrOwnerRole();
  }

  private openEditGroupSheet(): void {
    if (!this.groupInfo) {
      return;
    }
    if (!this.canEditGroupInfo()) {
      SystemToast.showWarning('仅群主或管理员可编辑群组信息', this.getUIContext());
      return;
    }

    this.editGroupName = this.groupInfo.group?.name ?? '';
    this.editGroupDescription = this.groupInfo.description ?? '';
    this.editGroupAvatarPreviewUrl = this.groupInfo.group?.avatar_url ?? '';
    this.editGroupAvatarLocalUri = '';
    this.editGroupAvatarFileId = '';
    // 群设置扩展回填（对齐前端 GroupSettingsSheet：仅 invite_only 二元，密码群编辑保持公开形态）
    this.editGroupPrivate = this.groupInfo.join_method === 'invite_only';
    this.editGroupRequiresApproval = this.groupInfo.requires_approval;
    this.closeOverlaySheets();
    this.showEditGroupSheet = true;
  }

  private closeEditGroupSheet(): void {
    if (this.isSavingGroupInfo || this.isEditGroupAvatarUploading) {
      return;
    }
    this.showEditGroupSheet = false;
  }

  private async selectEditGroupAvatar(): Promise<void> {
    if (this.isSavingGroupInfo || this.isEditGroupAvatarUploading) {
      return;
    }
    try {
      const context = this.getUIContext().getHostContext() as common.UIAbilityContext;
      const selectedUris = await MediaServiceManager.selectImages(context, 1);
      if (selectedUris.length === 0) {
        return;
      }
      this.editGroupAvatarLocalUri = selectedUris[0];
      this.editGroupAvatarPreviewUrl = selectedUris[0];
      this.editGroupAvatarFileId = '';
      SystemToast.show('已选择新头像，保存时自动上传', this.getUIContext());
    } catch {
      SystemToast.showError('选择群头像失败，请重试', this.getUIContext());
    }
  }

  private resetEditGroupAvatarSelection(): void {
    this.editGroupAvatarLocalUri = '';
    this.editGroupAvatarFileId = '';
    this.editGroupAvatarPreviewUrl = this.groupInfo?.group?.avatar_url ?? '';
  }

  private async ensureEditGroupAvatarUploaded(context: common.UIAbilityContext): Promise<string | undefined> {
    if (!this.editGroupAvatarLocalUri) {
      return undefined;
    }
    if (this.editGroupAvatarFileId) {
      return this.editGroupAvatarFileId;
    }

    this.isEditGroupAvatarUploading = true;
    try {
      const compressedPath = await MediaServiceManager.compressImage(
        context,
        this.editGroupAvatarLocalUri,
        80,
        1024 * 1024 * 2
      );
      const uploadUri = await MediaServiceManager.prepareFileForUpload(context, compressedPath);
      const uploadResult = await ApiClient.uploadFile(context, uploadUri, 'group_avatar');
      if (!uploadResult.file_id) {
        throw new Error('群头像上传失败');
      }
      this.editGroupAvatarFileId = uploadResult.file_id;
      return this.editGroupAvatarFileId;
    } finally {
      this.isEditGroupAvatarUploading = false;
    }
  }

  private async saveGroupInfo(): Promise<void> {
    if (!this.groupInfo) {
      return;
    }
    if (!this.ensureLoggedIn('编辑群组信息')) {
      return;
    }
    if (!this.canEditGroupInfo()) {
      SystemToast.showWarning('无权限编辑群组信息', this.getUIContext());
      return;
    }

    const trimmedName = this.editGroupName.trim();
    const trimmedDescription = this.editGroupDescription.trim();
    const currentName = this.groupInfo.group?.name ?? '';
    const currentDescription = this.groupInfo.description ?? '';
    const hasAvatarChange = this.editGroupAvatarLocalUri.length > 0;
    const nextJoinMethod = this.editGroupPrivate ? 'invite_only' : 'direct';
    const hasSettingsChange = nextJoinMethod !== this.groupInfo.join_method ||
      this.editGroupRequiresApproval !== this.groupInfo.requires_approval;

    if (!trimmedName) {
      SystemToast.showWarning('群组名称不能为空', this.getUIContext());
      return;
    }
    if (trimmedName.length > GROUP_NAME_MAX) {
      SystemToast.showWarning(`群组名称最多 ${GROUP_NAME_MAX} 字符`, this.getUIContext());
      return;
    }
    if (trimmedDescription.length > GROUP_DESCRIPTION_MAX) {
      SystemToast.showWarning(`群组描述最多 ${GROUP_DESCRIPTION_MAX} 字符`, this.getUIContext());
      return;
    }
    if (trimmedName === currentName && trimmedDescription === currentDescription &&
      !hasAvatarChange && !hasSettingsChange) {
      SystemToast.show('未检测到可保存的变更', this.getUIContext());
      return;
    }

    this.isSavingGroupInfo = true;
    try {
      const context = this.getUIContext().getHostContext() as common.UIAbilityContext;
      const avatarFileId = await this.ensureEditGroupAvatarUploaded(context);
      const request: UpdateGroupRequest = {
        group_id: this.groupId,
        name: trimmedName,
        description: trimmedDescription,
        avatar_file_id: avatarFileId,
        join_method: nextJoinMethod,
        requires_approval: this.editGroupRequiresApproval
      };
      const response = await ApiClient.updateGroup(request);
      if (response.code === 0) {
        this.showEditGroupSheet = false;
        await this.loadGroupDetail();
        SystemToast.showSuccess('群组信息已更新', this.getUIContext());
      } else {
        SystemToast.showError(getErrorMessage(response, '更新群组信息失败'), this.getUIContext());
      }
    } catch {
      SystemToast.showError('更新群组信息失败', this.getUIContext());
    } finally {
      this.isSavingGroupInfo = false;
    }
  }

  // ==================== 群管理面板补齐（对齐前端 InviteMembersSheet / banned / 危险区） ====================

  // ---- 邀请成员：搜索用户 + 多选 + 批量邀请（对齐前端 InviteMembersSheet） ----

  private openInviteSheet(): void {
    if (!this.ensureLoggedIn('邀请成员')) return;
    if (!this.canManageApplications()) {
      SystemToast.showWarning('仅群主或管理员可邀请成员', this.getUIContext());
      return;
    }
    this.inviteKeyword = '';
    this.inviteResults = [];
    this.inviteSearchError = '';
    this.inviteSelectedIds = [];
    this.isInviting = false;
    this.inviteSearchSeq += 1;
    this.closeOverlaySheets();
    this.showInviteSheet = true;
  }

  private isInviteUserSelected(userId: string): boolean {
    return this.inviteSelectedIds.includes(userId);
  }

  private toggleInviteUser(userId: string): void {
    if (this.isInviteUserSelected(userId)) {
      this.inviteSelectedIds = this.inviteSelectedIds.filter((id: string) => id !== userId);
    } else {
      this.inviteSelectedIds = this.inviteSelectedIds.concat([userId]);
    }
  }

  /** 邀请搜索输入防抖（350ms，对齐前端 InviteMembersSheet onInput） */
  private handleInviteKeywordChange(value: string): void {
    this.inviteKeyword = value;
    if (this.inviteSearchTimer !== -1) {
      clearTimeout(this.inviteSearchTimer);
      this.inviteSearchTimer = -1;
    }
    const keyword = value.trim();
    if (!keyword) {
      this.inviteResults = [];
      this.inviteSearchError = '';
      return;
    }
    this.inviteSearchTimer = setTimeout(() => {
      this.inviteSearchTimer = -1;
      this.searchInviteUsers(keyword);
    }, 350);
  }

  private async searchInviteUsers(keyword: string): Promise<void> {
    const seq = ++this.inviteSearchSeq;
    this.isSearchingInviteUsers = true;
    this.inviteSearchError = '';
    try {
      const response = await ApiClient.searchUsers({ keyword: keyword, pagination: { limit: 20 } });
      // 竞态保护：仅最新一次搜索的结果允许落地（对齐前端 searchSeq 模式）
      if (seq !== this.inviteSearchSeq) {
        return;
      }
      if (response.code === 0) {
        this.inviteResults = response.data?.items ?? [];
      } else {
        this.inviteResults = [];
        this.inviteSearchError = getErrorMessage(response, '搜索失败');
      }
    } catch {
      if (seq !== this.inviteSearchSeq) {
        return;
      }
      // 失败必须清空结果：避免用户把上一次关键词的命中项误当作当前结果邀请
      this.inviteResults = [];
      this.inviteSearchError = '网络异常，请稍后重试';
    } finally {
      if (seq === this.inviteSearchSeq) {
        this.isSearchingInviteUsers = false;
      }
    }
  }

  private async submitInvites(): Promise<void> {
    if (this.inviteSelectedIds.length === 0 || this.isInviting) {
      return;
    }
    this.isInviting = true;
    let okCount = 0;
    try {
      for (const userId of this.inviteSelectedIds) {
        const response = await ApiClient.inviteMember({ group_id: this.groupId, user_id: userId });
        if (response.code === 0) {
          okCount += 1;
        }
      }
      const total = this.inviteSelectedIds.length;
      if (okCount === total) {
        SystemToast.showSuccess(`已邀请 ${okCount} 人`, this.getUIContext());
      } else if (okCount > 0) {
        SystemToast.showSuccess(`已邀请 ${okCount}/${total} 人`, this.getUIContext());
      } else {
        SystemToast.showError('邀请失败', this.getUIContext());
      }
      this.showInviteSheet = false;
    } catch {
      SystemToast.showError('邀请失败，请重试', this.getUIContext());
    } finally {
      this.isInviting = false;
    }
  }

  // ---- 封禁管理：名单 + 解封 + 成员封禁（对齐前端 banned 子 Tab / MemberActionSheet） ----

  private async loadBannedMembers(): Promise<void> {
    if (!this.canManageApplications()) {
      return;
    }
    this.isLoadingBannedMembers = true;
    try {
      const response = await ApiClient.getBannedMembers({
        group_id: this.groupId,
        pagination: { limit: 100 }
      });
      if (response.code === 0) {
        this.bannedMembers = response.data?.items ?? [];
      }
    } catch {
      // 静默失败：列表显示空态
    } finally {
      this.isLoadingBannedMembers = false;
    }
  }

  private openBannedListSheet(): void {
    if (!this.canManageApplications()) {
      SystemToast.showWarning('仅群主或管理员可查看封禁名单', this.getUIContext());
      return;
    }
    this.closeOverlaySheets();
    this.showBannedListSheet = true;
    this.loadBannedMembers();
  }

  private async unbanMember(banned: AdminBannedMemberItem): Promise<void> {
    if (!this.ensureLoggedIn('解封成员')) return;
    const userId = banned.user?.user_id;
    if (!userId) return;
    try {
      const response = await ApiClient.unbanMember({ group_id: this.groupId, user_id: userId });
      if (response.code === 0) {
        this.bannedMembers = this.bannedMembers.filter((item: AdminBannedMemberItem) =>
          item.user?.user_id !== userId);
        SystemToast.showSuccess('已解封', this.getUIContext());
      } else {
        SystemToast.showError(getErrorMessage(response, '解封失败'), this.getUIContext());
      }
    } catch {
      SystemToast.showError('解封失败，请重试', this.getUIContext());
    }
  }

  /** 成员菜单「封禁并移出」入口：二次确认（危险操作防误触） */
  private requestBanMember(member: GroupMemberItem): void {
    if (!this.canManageMember(member)) {
      SystemToast.showWarning('无权限封禁该成员', this.getUIContext());
      return;
    }
    this.banTargetMember = member;
    this.showBanConfirm = true;
  }

  private async confirmBanMember(): Promise<void> {
    const member = this.banTargetMember;
    this.showBanConfirm = false;
    this.banTargetMember = null;
    if (member === null) {
      return;
    }
    const userId = member.user?.user_id;
    if (!userId) return;
    try {
      const response = await ApiClient.banMember({ group_id: this.groupId, user_id: userId });
      if (response.code === 0) {
        // 封禁后成员移出群组（对齐前端 ban 后 loadMembers 刷新语义）
        this.members = this.members.filter((item: GroupMemberItem) => item.user?.user_id !== userId);
        this.membersDataSource.setData(this.members);
        if (this.groupInfo?.group) {
          this.groupInfo.group.members_count = Math.max(0, (this.groupInfo.group.members_count ?? 0) - 1);
        }
        SystemToast.showSuccess('已封禁并移出群组', this.getUIContext());
      } else {
        SystemToast.showError(getErrorMessage(response, '封禁失败'), this.getUIContext());
      }
    } catch {
      SystemToast.showError('封禁失败，请重试', this.getUIContext());
    }
  }

  // ---- 危险区（仅群主）：转让群主 / 解散群组（对齐前端 GroupSettingsSheet 危险区） ----

  /** 展开转让成员选择（懒加载候选成员：非群主、非自己） */
  private toggleTransferSection(): void {
    this.transferSectionOpen = !this.transferSectionOpen;
    if (this.transferSectionOpen && this.transferCandidates.length === 0) {
      this.loadTransferCandidates();
    }
  }

  private async loadTransferCandidates(): Promise<void> {
    this.isLoadingTransferCandidates = true;
    try {
      const response = await ApiClient.getGroupMembers({
        group_id: this.groupId,
        pagination: { limit: 100 }
      });
      if (response.code === 0) {
        this.transferCandidates = (response.data?.items ?? []).filter((member: GroupMemberItem) =>
          member.role !== 'owner' && !this.isCurrentUser(member.user?.user_id));
      }
    } catch {
      // 静默失败：展示「暂无可转让成员」空态
    } finally {
      this.isLoadingTransferCandidates = false;
    }
  }

  private getRoleShortText(role: string): string {
    if (role === 'admin') {
      return '管理员';
    }
    return '成员';
  }

  private requestTransferOwnership(member: GroupMemberItem): void {
    if (!this.isOwnerRole()) {
      SystemToast.showWarning('仅群主可转让群主身份', this.getUIContext());
      return;
    }
    if (this.isCurrentUser(member.user?.user_id)) {
      return;
    }
    this.transferTargetMember = member;
    this.showTransferConfirm = true;
  }

  private async confirmTransferOwnership(): Promise<void> {
    const member = this.transferTargetMember;
    this.showTransferConfirm = false;
    this.transferTargetMember = null;
    if (member === null) {
      return;
    }
    const userId = member.user?.user_id;
    if (!userId) return;
    this.isTransferringOwnership = true;
    try {
      const response = await ApiClient.transferOwnership({ group_id: this.groupId, new_owner_id: userId });
      if (response.code === 0) {
        SystemToast.showSuccess('群主已转让', this.getUIContext());
        this.showEditGroupSheet = false;
        await this.loadGroupDetail();
      } else {
        SystemToast.showError(getErrorMessage(response, '转让失败'), this.getUIContext());
      }
    } catch {
      SystemToast.showError('转让失败，请重试', this.getUIContext());
    } finally {
      this.isTransferringOwnership = false;
    }
  }

  private requestDismissGroup(): void {
    if (!this.isOwnerRole()) {
      SystemToast.showWarning('仅群主可解散群组', this.getUIContext());
      return;
    }
    this.showDismissConfirm = true;
  }

  private async confirmDismissGroup(): Promise<void> {
    if (!this.isOwnerRole()) {
      this.showDismissConfirm = false;
      return;
    }
    this.showDismissConfirm = false;
    this.isDismissingGroup = true;
    try {
      const response = await ApiClient.dismissGroup({ group_id: this.groupId });
      if (response.code === 0) {
        SystemToast.showSuccess('群组已解散', this.getUIContext());
        InteractionStateStore.getInstance().updateGroupMembership(this.groupId, { is_member: false });
        AppRouter.pop();
      } else {
        SystemToast.showError(getErrorMessage(response, '解散失败'), this.getUIContext());
      }
    } catch {
      SystemToast.showError('解散失败，请重试', this.getUIContext());
    } finally {
      this.isDismissingGroup = false;
    }
  }

  private async enterChat(): Promise<void> {
    if (!this.groupInfo) return;
    if (!this.ensureLoggedIn('进入群聊')) return;
    const chatParams = new ChatRouteParam();
    chatParams.chatId = this.groupId;
    chatParams.chatType = 'group_chat';
    chatParams.title = this.groupInfo.group?.name ?? '';
    AppRouter.push(PageNames.CHAT, chatParams);
  }

  private openPrivateChat(member: GroupMemberItem): void {
    if (!this.ensureLoggedIn('发起私聊')) return;
    if (!member.user?.user_id) return;
    const myId = this.currentUser?.user_id ?? '';
    if (member.user.user_id === myId) {
      SystemToast.showWarning('不能和自己发起私聊', this.getUIContext());
      return;
    }
    const chatParams = new ChatRouteParam();
    chatParams.chatId = member.user.user_id;
    chatParams.chatType = 'private_chat';
    chatParams.targetId = member.user.user_id;
    chatParams.title = member.user.nickname ?? '';
    AppRouter.push(PageNames.CHAT, chatParams);
  }

  private currentGroupRole(): string {
    return this.groupInfo?.viewer_state?.role ?? 'member';
  }

  private isOwnerRole(): boolean {
    return this.currentGroupRole() === 'owner';
  }

  private isAdminOrOwnerRole(): boolean {
    const role = this.currentGroupRole();
    return role === 'owner' || role === 'admin';
  }

  private isCurrentUser(userId?: string): boolean {
    if (!userId) return false;
    return userId === (this.currentUser?.user_id ?? '');
  }

  private canManageMember(member: GroupMemberItem): boolean {
    if (!this.isMember || !this.isAdminOrOwnerRole()) return false;
    const targetUserId = member.user?.user_id;
    if (!targetUserId) return false;
    if (this.isCurrentUser(targetUserId)) return false;
    if (member.role === 'owner') return false;
    if (!this.isOwnerRole() && member.role === 'admin') return false;
    return true;
  }

  private canChangeRole(member: GroupMemberItem): boolean {
    return this.canManageMember(member);
  }

  private isMemberMuted(member: GroupMemberItem): boolean {
    const mutedUntil = member.muted_until;
    if (!mutedUntil) return false;
    const ts = new Date(mutedUntil).getTime();
    return Number.isFinite(ts) && ts > Date.now();
  }

  private upsertMember(updated: GroupMemberItem): void {
    const targetUserId = updated.user?.user_id;
    if (!targetUserId) return;
    this.members = this.members.map((item) => {
      if (item.user?.user_id === targetUserId) {
        return updated;
      }
      return item;
    });
    this.membersDataSource.setData(this.members);
  }

  private withMemberRole(member: GroupMemberItem, role: SettableGroupRole): GroupMemberItem {
    return {
      user: member.user,
      role: role,
      joined_at: member.joined_at,
      muted_until: member.muted_until,
      presence_status: member.presence_status,
      last_online_at: member.last_online_at
    };
  }

  private withMemberMutedUntil(member: GroupMemberItem, mutedUntil?: string): GroupMemberItem {
    return {
      user: member.user,
      role: member.role,
      joined_at: member.joined_at,
      muted_until: mutedUntil,
      presence_status: member.presence_status,
      last_online_at: member.last_online_at
    };
  }

  private async setMemberRole(member: GroupMemberItem, role: SettableGroupRole): Promise<void> {
    if (!this.ensureLoggedIn('调整成员角色')) return;
    if (!this.canChangeRole(member)) {
      SystemToast.showWarning('无权限调整该成员角色', this.getUIContext());
      return;
    }
    const userId = member.user?.user_id;
    if (!userId) return;
    try {
      const response = await ApiClient.setMemberRole({
        group_id: this.groupId,
        user_id: userId,
        role: role
      });
      if (response.code === 0) {
        this.upsertMember(this.withMemberRole(member, role));
        SystemToast.showSuccess('成员角色已更新', this.getUIContext());
      } else {
        SystemToast.showError(getErrorMessage(response, '成员角色更新失败'), this.getUIContext());
      }
    } catch {
      SystemToast.showError('成员角色更新失败', this.getUIContext());
    }
  }

  private async muteMember(member: GroupMemberItem, durationMinutes: number): Promise<void> {
    if (!this.ensureLoggedIn('禁言成员')) return;
    if (!this.canManageMember(member)) {
      SystemToast.showWarning('无权限禁言该成员', this.getUIContext());
      return;
    }
    const userId = member.user?.user_id;
    if (!userId) return;
    try {
      const response = await ApiClient.muteMember({
        group_id: this.groupId,
        user_id: userId,
        active: true,
        duration_minutes: durationMinutes
      });
      if (response.code === 0) {
        const mutedUntil = new Date(Date.now() + durationMinutes * 60 * 1000).toISOString();
        this.upsertMember(this.withMemberMutedUntil(member, mutedUntil));
        SystemToast.showSuccess('已设置禁言', this.getUIContext());
      } else {
        SystemToast.showError(getErrorMessage(response, '禁言失败'), this.getUIContext());
      }
    } catch {
      SystemToast.showError('禁言失败', this.getUIContext());
    }
  }

  private async unmuteMember(member: GroupMemberItem): Promise<void> {
    if (!this.ensureLoggedIn('解除禁言')) return;
    if (!this.canManageMember(member)) {
      SystemToast.showWarning('无权限解除禁言', this.getUIContext());
      return;
    }
    const userId = member.user?.user_id;
    if (!userId) return;
    try {
      const response = await ApiClient.unmuteMember({
        group_id: this.groupId,
        user_id: userId,
        active: false
      });
      if (response.code === 0) {
        this.upsertMember(this.withMemberMutedUntil(member, undefined));
        SystemToast.showSuccess('已解除禁言', this.getUIContext());
      } else {
        SystemToast.showError(getErrorMessage(response, '解除禁言失败'), this.getUIContext());
      }
    } catch {
      SystemToast.showError('解除禁言失败', this.getUIContext());
    }
  }

  private async removeMember(member: GroupMemberItem): Promise<void> {
    if (!this.ensureLoggedIn('移除成员')) return;
    if (!this.canManageMember(member)) {
      SystemToast.showWarning('无权限移除该成员', this.getUIContext());
      return;
    }
    const userId = member.user?.user_id;
    if (!userId) return;
    try {
      const response = await ApiClient.removeMember({
        group_id: this.groupId,
        user_id: userId
      });
      if (response.code === 0) {
        this.members = this.members.filter(item => item.user?.user_id !== userId);
        this.membersDataSource.setData(this.members);
        if (this.groupInfo?.group) {
          const nextMembersCount = Math.max(0, (this.groupInfo.group.members_count ?? 0) - 1);
          this.groupInfo.group.members_count = nextMembersCount;
        }
        SystemToast.showSuccess('已移除成员', this.getUIContext());
      } else {
        SystemToast.showError(getErrorMessage(response, '移除成员失败'), this.getUIContext());
      }
    } catch {
      SystemToast.showError('移除成员失败', this.getUIContext());
    }
  }

  private buildMemberActionMenus(member: GroupMemberItem): MemberActionMenuItem[] {
    const menus: MemberActionMenuItem[] = [];
    if (member.user?.user_id && !this.isCurrentUser(member.user.user_id)) {
      menus.push({
        value: '发起私聊',
        action: () => { this.openPrivateChat(member); }
      });
    }
    if (!this.canManageMember(member)) {
      return menus;
    }
    if (this.canChangeRole(member)) {
      if (member.role === 'member') {
        menus.push({
          value: '设为管理员',
          action: () => { this.setMemberRole(member, 'admin'); }
        });
      } else if (member.role === 'admin') {
        menus.push({
          value: '设为成员',
          action: () => { this.setMemberRole(member, 'member'); }
        });
      }
    }
    if (this.isMemberMuted(member)) {
      menus.push({
        value: '解除禁言',
        action: () => { this.unmuteMember(member); }
      });
    } else {
      menus.push({
        value: '禁言 30 分钟',
        action: () => { this.muteMember(member, MUTE_DURATION_MINUTES); }
      });
    }
    menus.push({
      value: '移出群组',
      action: () => { this.removeMember(member); }
    });
    // 封禁并移出（对齐前端 MemberActionSheet ban 项；二次确认在 requestBanMember 内）
    menus.push({
      value: '封禁并移出',
      action: () => { this.requestBanMember(member); }
    });
    return menus;
  }

  /** 行内是否展示操作菜单（轻量布尔判断；替代渲染期构造完整菜单数组判空） */
  private hasMemberActions(member: GroupMemberItem): boolean {
    const userId = member.user?.user_id;
    if (userId && !this.isCurrentUser(userId)) {
      return true;
    }
    return this.canManageMember(member);
  }

  private getRoleDisplayText(role: string): string {
    switch (role) {
      case 'owner': return '群主';
      case 'admin': return '管理员';
      case 'member': return '成员';
      default: return '未知';
    }
  }

  private getRoleColor(role: string): ResourceColor {
    switch (role) {
      case 'owner': return this.getCurrentThemePalette().accentBackground;
      case 'admin': return this.getCurrentThemePalette().accentBackground;
      case 'member': return this.getCurrentThemePalette().textSecondary;
      default: return this.getCurrentThemePalette().textSecondary;
    }
  }


  build() {
    NavDestination() {
    PageShellContainer({ variant: 'default', currentBreakpoint: this.currentBreakpoint }) {
    Column() {
      // 封面型头部（对齐前端 groups/[groupId]：全幅封面 + 返回浮层 + 左下分类胶囊 + 群名白字；
      // 鸿蒙 GroupInfo 无 banner 字段，用 accent 渐变占位，待后端支持后接真图）
      // 子节点定位说明：ArkUI Stack 的子级 .align() 只影响其自身内容对齐、
      // 不改变自身在 Stack 中的位置（曾致标题/分享钮全部叠在左上）。
      // 因此：标题列放进内层 Stack（BottomStart 锚到封面左下），
      // 分享按钮用通栏 Row 右对齐，返回按钮保持 TopStart + margin。
      Stack({ alignContent: Alignment.TopStart }) {
        // 封面（accent 渐变占位 + 左下分类胶囊/群名）
        // 通栏封面：用负 margin 抵消容器 gutter（对齐前端 -mx-page-gutter），
        // 而不是让页面另写一层 padding——否则正文会比页头窄一截
        Stack({ alignContent: Alignment.BottomStart }) {
          Column()
            .width('100%')
            .height(DesignConstants.LAYOUT.COVER_HEIGHT)
            // 封面底部加深 scrim：文字区（群名白字）落在深色终止段，
            // 浅色主题下纯 accent 淡渐变近似白底白字（对比度不足）
            .linearGradient({
              direction: GradientDirection.Bottom,
              colors: [
                [AppThemeVisuals.withAlpha(this.getCurrentThemePalette().accentBackground, 0.38), 0.0],
                [AppThemeVisuals.withAlpha(this.getCurrentThemePalette().accentBackground, 0.28), 0.55],
                [AppThemeVisuals.withAlpha(DesignConstants.COLOR.BLACK, 0.42), 1.0]
              ]
            })

          // 左下：分类胶囊 + 群名（对齐前端封面底部：玻璃徽章 + h1 白字）
          Column({ space: DesignConstants.SPACING.SM }) {
            if (this.groupInfo?.tags && this.groupInfo.tags.length > 0) {
              // 玻璃分类徽章（对齐前端 bg-surface-glass-badge + backdrop-blur-sm）
              Text(`#${this.groupInfo.tags[0]}`)
                .fontSize(DesignConstants.FONT.SIZE.XS2)
                .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS2))
                .fontColor(DesignConstants.COLOR.WHITE)
                .padding({ left: DesignConstants.SPACING.MD, right: DesignConstants.SPACING.MD, top: DesignConstants.SPACING.XS, bottom: DesignConstants.SPACING.XS })
                .borderRadius(DesignConstants.RADIUS.FULL)
                .backgroundBlurStyle(BlurStyle.Thin)
                .backgroundColor(AppThemeVisuals.withAlpha(DesignConstants.COLOR.BLACK, 0.25))
            }
            if (this.groupInfo?.group?.name) {
              Text(this.groupInfo.group.name)
                .fontSize(DesignConstants.FONT.SIZE.XXL)
                .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XXL))
                .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
                .fontColor(DesignConstants.COLOR.WHITE)
                .maxLines(1)
                .textOverflow({ overflow: TextOverflow.Ellipsis })
            }
          }
          .alignItems(HorizontalAlign.Start)
          .width('100%')
          // 封面内文字沿用容器 gutter（这里不再自加，避免与容器叠加）
          .padding({ bottom: DesignConstants.SPACING.LG })
        }
        .width('100%')

        // 顶部按钮行（返回左 / 分享右）：全宽透明 Row 会遮挡返回钮点击（已实测），
        // 两钮必须同行；标题列已在内层 Stack 锚定封面左下，不与此行冲突。
        // hero 封面上的媒体镀铬钮：白字 + 透明底（iconColor 逃逸口），a11y 补齐
        Row() {
          SurfaceIconButton({
            icon: AppIcon.BACK,
            kind: 'neutral',
            buttonSize: DesignConstants.LAYOUT.ICON_BUTTON_SIZE,
            glyphSize: DesignConstants.COMPONENT.ICON_SIZE_XS,
            iconColor: DesignConstants.COLOR.WHITE,
            accessibilityLabel: '返回',
            onPress: () => { AppRouter.pop(); }
          })
          Blank()
          // web 群详情封面右上角是「更多操作」(MoreHorizontal overlay) —— 内含举报群组；
          // 原「分享」在前端群详情页不存在（分享入口在群分享码/我的分享）
          SurfaceIconButton({
            icon: AppIcon.MORE,
            kind: 'neutral',
            buttonSize: DesignConstants.LAYOUT.ICON_BUTTON_SIZE,
            glyphSize: DesignConstants.COMPONENT.ICON_SIZE_XS,
            iconColor: DesignConstants.COLOR.WHITE,
            accessibilityLabel: '更多操作',
            onPress: () => { this.showGroupMoreSheet = true; }
          })
        }
        .width('100%')
        .margin({ top: this.getHeaderTopPadding() })
      }
      .width('100%')
      // 封面与下方 Tab/卡片同宽（都落在容器 gutter 内）。
      // 注：这里**不能**用「负 margin 抵消 gutter」做通栏——ArkUI 下子元素
      // width('100%') 相对父内容盒解析，负 margin 只会位移不会加宽，实测仍是
      // 左右各内缩一个 gutter（本机 2026-10-11 实测：封面 x=70..1186 / 屏宽 1256）。
      // 真要通栏需把本页改为 noHorizontalPadding + 各区块自加 gutter（已登记例外），
      // 或改用 calc 宽度；当前选择与下方内容对齐的「收束式 hero」，视觉更一致。
      .expandSafeArea([SafeAreaType.SYSTEM], [SafeAreaEdge.TOP])

      // 举报确认弹窗（更多操作 sheet → 举报群组；danger 确认，对齐 ChatPage 举报范式）。
      // 挂在常驻 Stack 上：任何 tab / 加载态下都可用（更多 sheet 入口在封面右上角）。
      SystemDialog({
        visible: this.showGroupReportConfirm,
        title: '举报群组',
        description: '将对该群组提交举报，我们会尽快审核处理',
        confirmText: '提交举报',
        confirmVariant: 'danger',
        onCancel: () => { this.showGroupReportConfirm = false; },
        onConfirm: () => {
          this.showGroupReportConfirm = false;
          this.submitGroupReport();
        }
      })

      if (this.isLoading) {
        SystemLoadingView({ message: '加载群组信息中…' })
          .layoutWeight(1)
          .transition(
            TransitionEffect.OPACITY.animation({ duration: DesignConstants.ANIMATION.DURATION_SHEET, curve: DesignConstants.ANIMATION.CURVE_FAST_OUT_SLOW_IN })
              .combine(TransitionEffect.move(TransitionEdge.BOTTOM).animation({ duration: DesignConstants.ANIMATION.DURATION_SHEET, curve: DesignConstants.ANIMATION.CURVE_FAST_OUT_SLOW_IN }))
          )
      } else if (this.errorMessage) {
        EmptyState({
          bare: true,
          icon: AppIcon.WARNING,
          title: this.errorMessage,
          actionLabel: '重试',
          fillMode: true,
          onAction: () => { this.loadGroupDetail(); }
        })
        } else if (this.groupInfo) {
        // 元信息（成员/在线/距你）不再在此重复展示：简介 tab 四宫格数据卡承载
        // 同一份数据（大版优先），切 tab 时由「群组成员」入口行延续上下文。

        // 三 Tab（对齐前端 TabBar 胶囊：简介/成员/公告）
        TabsPillGroup({
          tabs: [
            { label: '简介' },
            { label: '成员' },
            { label: '公告' }
          ],
          activeIndex: this.activeTab,
          actions: {
            onChange: (index: number) => {
              this.activeTab = index;
              // 公告 Tab 首次进入时懒加载
              if (index === 2 && this.announcements.length === 0) {
                this.loadGroupAnnouncements();
              }
            }
          }
        })
          .padding({ top: DesignConstants.SPACING.MD, bottom: DesignConstants.SPACING.SM })

        // 使用 List 布局（扁平风格）
        List() {
          // 简介/成员/操作/公告统一收在一个分组容器里（外框 + 行间分隔线由 ListItemGroup
          // 承担，行本身不再各自带白卡/圆角/投影）
          ListItemGroup() {
          // Tab 0 简介区（封面已承载群名/分类，元信息行已承载成员/在线/距离；
          //    位置名以 MapPin 行展示，对齐前端 users/[userId] 位置行模式）
          if (this.activeTab === 0) {
          ListItem() {
            Column({ space: DesignConstants.SPACING.MD }) {
              if (this.groupInfo?.description) {
                Text(this.groupInfo?.description)
                  .fontSize(DesignConstants.FONT.SIZE.MD)
                  .fontColor(this.getCurrentThemePalette().textSecondary)
                  .textAlign(TextAlign.Start)
                  .width('100%')
                  // 群简介是卡片内多行正文：1.57 → 1.7（对齐前端 --leading-card）
                  .lineHeight(lineHeightFor(DesignConstants.FONT.SIZE.MD, DesignConstants.FONT.LINE_RATIO.CARD))
              }
              Row({ space: DesignConstants.SPACING.XS }) {
                LucideIcon({
                  icon: AppIcon.LOCATION,
                  iconSize: DesignConstants.COMPONENT.ICON_SIZE_2XS,
                  color: this.getCurrentThemePalette().accentBackground
                })
                Text(this.groupInfo?.location?.location_name || '未知位置')
                  .fontSize(DesignConstants.FONT.SIZE.XS)
                  .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS))
                  .fontColor(this.getCurrentThemePalette().textSecondary)
                  .maxLines(1)
                  .textOverflow({ overflow: TextOverflow.Ellipsis })
                  .layoutWeight(1)
                // 交互暗示（用户重设计：位置行需可交互 affordance）：
                // 鸿蒙端暂无地图详情路由，先提供「复制位置名」真实功能 + 尾随箭头
                LucideIcon({
                  icon: AppIcon.FORWARD,
                  iconSize: DesignConstants.COMPONENT.ICON_SIZE_2XS,
                  color: this.getCurrentThemePalette().textSecondary
                })
              }
              .alignItems(VerticalAlign.Center)
              .onClick(() => {
                const name = this.groupInfo?.location?.location_name;
                if (!name) return;
                const data = pasteboard.createData('text/plain', name);
                pasteboard.getSystemPasteboard().setData(data);
                SystemToast.showSuccess(`位置已复制：${name}`, this.getUIContext());
              })
            }
            .width('100%')
            .alignItems(HorizontalAlign.Start)
            .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, top: DesignConstants.SPACING.MD, bottom: DesignConstants.SPACING.MD })
          }
          // 行级卡片样式移除：外框/分隔线由 ListItemGroup 承担
          }

          // 四宫格数据卡（对齐前端 groups/[groupId] 简介区 2×2 stats：成员/在线/距你/分类）
          if (this.activeTab === 0) {
            ListItem() {
              this.buildStatsGrid()
            }
            .margin({ bottom: DesignConstants.SPACING.MD })
          }

          // 成员列表入口（Tab 1 成员 或 Tab 0 简介均显示）
          ListItem() {
            Row() {
              Column() {
                 Text('群组成员')
                  .fontSize(DesignConstants.FONT.SIZE.MD)
                  .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.MD))
                  .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
                  .fontColor(this.getCurrentThemePalette().textPrimary)
                 Text(`${formatCount(this.groupInfo?.group?.members_count)} 名成员`)
                  .fontSize(DesignConstants.FONT.SIZE.SM)
                  .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.SM))
                  .fontColor(this.getCurrentThemePalette().textSecondary)
                  .margin({ top: DesignConstants.SPACING.HALF })
              }
              .layoutWeight(1)
              .alignItems(HorizontalAlign.Start)

              LucideIcon({
                  icon: AppIcon.FORWARD,
                  iconSize: DesignConstants.COMPONENT.ICON_SIZE_XS,
                  color: this.getCurrentThemePalette().textSecondary
                })
            }
            .width('100%')
            .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG,
              top: DesignConstants.SPACING.MD, bottom: DesignConstants.SPACING.MD })
          }
          // 行级卡片样式移除：外框/分隔线由 ListItemGroup 承担
          // 整卡可点（扩大热区，chevron 仅作视觉指示）+ 按压反馈（对齐 tap-97 语义）
          .scale(this.pressedEntryKey === 'members' ?
            { x: DesignConstants.INTERACTION.SCALE_PRESSED_CARD, y: DesignConstants.INTERACTION.SCALE_PRESSED_CARD } :
            { x: 1, y: 1 })
          .animation({ curve: curves.springMotion(
            DesignConstants.INTERACTION.SPRING_RESPONSE_PRESS,
            DesignConstants.INTERACTION.SPRING_DAMPING_PRESS
          ) })
          .onTouch((event: TouchEvent) => {
            if (event.type === TouchType.Down) {
              this.pressedEntryKey = 'members';
            } else if (event.type === TouchType.Up || event.type === TouchType.Cancel) {
              this.pressedEntryKey = '';
            }
          })
          .onClick(() => this.loadGroupMembers())

          if (this.canManageApplications()) {
            // 3.5 入群申请入口（仅群主/管理员可见）
            ListItem() {
              Row() {
                Column() {
                  Text('入群申请')
                    .fontSize(DesignConstants.FONT.SIZE.MD)
                    .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.MD))
                    .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
                    .fontColor(this.getCurrentThemePalette().textPrimary)
                  Text('处理待审核的入群请求')
                    .fontSize(DesignConstants.FONT.SIZE.SM)
                    .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.SM))
                    .fontColor(this.getCurrentThemePalette().textSecondary)
                    .margin({ top: DesignConstants.SPACING.HALF })
                }
                .layoutWeight(1)
                .alignItems(HorizontalAlign.Start)

                LucideIcon({
                    icon: AppIcon.FORWARD,
                    iconSize: DesignConstants.COMPONENT.ICON_SIZE_XS,
                    color: this.getCurrentThemePalette().textSecondary
                  })
              }
              .width('100%')
              .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG,
                top: DesignConstants.SPACING.MD, bottom: DesignConstants.SPACING.MD })
            }
            // 行级卡片样式移除：外框/分隔线由 ListItemGroup 承担
            // 整卡可点（扩大热区，chevron 仅作视觉指示）+ 按压反馈（对齐 tap-97 语义）
            .scale(this.pressedEntryKey === 'applications' ?
              { x: DesignConstants.INTERACTION.SCALE_PRESSED_CARD, y: DesignConstants.INTERACTION.SCALE_PRESSED_CARD } :
              { x: 1, y: 1 })
            .animation({ curve: curves.springMotion(
              DesignConstants.INTERACTION.SPRING_RESPONSE_PRESS,
              DesignConstants.INTERACTION.SPRING_DAMPING_PRESS
            ) })
            .onTouch((event: TouchEvent) => {
              if (event.type === TouchType.Down) {
                this.pressedEntryKey = 'applications';
              } else if (event.type === TouchType.Up || event.type === TouchType.Cancel) {
                this.pressedEntryKey = '';
              }
            })
            .onClick(() => this.loadGroupApplications())
          }
          }  // 导航行分组结束
          .divider({
            strokeWidth: DesignConstants.BORDER_WIDTH.HAIRLINE,
            color: this.getSurfaceBorder(),
            startMargin: DesignConstants.SPACING.LG,
            endMargin: DesignConstants.SPACING.LG
          })

          // 4. 操作按钮区域（独立分组）：按钮是页面级主操作、不是「列表行」，
          // 混在导航行分组里会被行分隔线上下夹住，形成「按钮被框住」的怪异观感
          ListItemGroup() {
          ListItem() {
             Column({ space: DesignConstants.SPACING.SM }) {
                if (this.isMember) {
                  if (this.canEditGroupInfo()) {
                    SystemButton({
                      label: '编辑群组信息',
                      variant: 'surface-accent',
                      buttonSize: 'default',
                      pill: true,
                      fullWidth: true,
                      shadowed: true,
                      loading: this.isSavingGroupInfo,
                      onPress: () => { this.openEditGroupSheet(); }
                    })

                    // 邀请成员 / 封禁名单（对齐前端成员 Tab 管理入口，owner/admin 可见）
                    SystemButton({
                      label: '邀请成员',
                      variant: 'surface',
                      buttonSize: 'default',
                      pill: true,
                      fullWidth: true,
                      shadowed: true,
                      onPress: () => { this.openInviteSheet(); }
                    })

                    SystemButton({
                      label: '封禁名单',
                      variant: 'surface',
                      buttonSize: 'default',
                      pill: true,
                      fullWidth: true,
                      shadowed: true,
                      onPress: () => { this.openBannedListSheet(); }
                    })
                  }

                  SystemButton({
                    label: '进入聊天',
                    variant: 'primary',
                    buttonSize: 'default',
                    pill: true,
                    fullWidth: true,
                    onPress: () => { this.enterChat(); }
                  })

                  SystemButton({
                    label: '离开群组',
                    variant: 'surface',
                    buttonSize: 'default',
                    pill: true,
                    fullWidth: true,
                    shadowed: true,
                    disabled: this.isLeaving,
                    onPress: () => { this.showLeaveConfirm = true; }
                  })
                    .opacity(this.isLeaving ? DesignConstants.OPACITY.LOADING : 1)

                  // 离开群组二次确认（对齐 GroupCard.ets / Web 同页规范形态）
                  SystemDialog({
                    visible: this.showLeaveConfirm,
                    title: '退出群组',
                    description: `退出「${this.groupInfo?.group?.name ?? '该群组'}」后需要重新申请加入`,
                    confirmText: '退出',
                    confirmVariant: 'danger',
                    onCancel: () => { this.showLeaveConfirm = false; },
                    onConfirm: () => {
                      this.showLeaveConfirm = false;
                      this.leaveGroup();
                    }
                  })
                } else {
                  if (this.groupInfo?.join_method === 'password') {
                    SystemInput({
                      value: this.joinPassword,
                      placeholder: '请输入群组密码',
                      passwordMode: true,
                      fontSize: DesignConstants.FONT.SIZE.MD,
                      onChange: (value: string) => { this.joinPassword = value; }
                    })
                      .padding(DesignConstants.SPACING.MD)
                      .backgroundColor(this.getSurfaceSubtle())
                      .shadow(this.getCurrentThemePalette().controlShadow)
                      .borderRadius(DesignConstants.RADIUS.XL)
                      .onBlur(() => KeyboardDismissManager.getInstance().dismiss())
                  }

                    SystemButton({
                      label: this.groupInfo?.join_method === 'password' ? '加入群组（需要密码）' : '加入群组',
                      variant: 'primary',
                      buttonSize: 'default',
                      pill: true,
                      fullWidth: true,
                      loading: this.isJoining,
                      onPress: () => { this.joinGroup(); }
                    })
                }
             }
             .padding(DesignConstants.SPACING.LG)
          }
          }  // 操作区分组结束

          // Tab 2 公告（对齐前端 AnnouncementsList：标题/内容/时间/置顶徽章；
          // owner/admin 可发布/编辑/删除，对齐前端 canManageAnnouncements 门控）
          ListItemGroup() {
          if (this.activeTab === 2) {
            if (this.canManageAnnouncements()) {
              ListItem() {
                SystemButton({
                  label: '发布新公告',
                  variant: 'surface-accent',
                  buttonSize: 'default',
                  pill: true,
                  fullWidth: true,
                  shadowed: true,
                  onPress: () => { this.openCreateAnnouncement(); }
                })
              }
              .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, bottom: DesignConstants.SPACING.MD })
            }
            if (this.announcements.length === 0) {
              ListItem() {
                EmptyState({
                  icon: AppIcon.NOTIFICATION,
                  message: '暂无公告',
                  bare: true
                })
                  .width('100%')
                  .padding({ top: DesignConstants.SPACING.XXL, bottom: DesignConstants.SPACING.XXL })
              }
            } else {
              ForEach(this.announcements, (item: GroupAnnouncementItem, index: number) => {
                if (index > 0) {
                  ListItem() {
                    ListRowDivider({ startMargin: DesignConstants.SPACING.LG, endMargin: DesignConstants.SPACING.LG })
                  }
                }
                ListItem() {
                  Column({ space: DesignConstants.SPACING.XS }) {
                    Row({ space: DesignConstants.SPACING.XS }) {
                      if (item.is_pinned) {
                        LucideIcon({
                          icon: AppIcon.STAR_FILL,
                          iconSize: DesignConstants.COMPONENT.ICON_SIZE_2XS,
                          color: this.getCurrentThemePalette().accentBackground
                        })
                      }
                      Text(item.title)
                        .fontSize(DesignConstants.FONT.SIZE.MD)
                        .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.MD))
                        .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
                        .fontColor(this.getCurrentThemePalette().textPrimary)
                        .maxLines(1)
                        .textOverflow({ overflow: TextOverflow.Ellipsis })
                        .layoutWeight(1)
                    }
                    .width('100%')

                    Text(item.content)
                      .fontSize(DesignConstants.FONT.SIZE.SM)
                      // 公告正文是多行长文：用前端 --leading-copy(1.75)，此前无行高
                      .lineHeight(lineHeightFor(DesignConstants.FONT.SIZE.SM, DesignConstants.FONT.LINE_RATIO.COPY))
                      .fontColor(this.getCurrentThemePalette().textSecondary)
                      .width('100%')
                      .maxLines(3)
                      .textOverflow({ overflow: TextOverflow.Ellipsis })

                    Text(formatRelativeTime(item.created_at))
                      .fontSize(DesignConstants.FONT.SIZE.XS)
                      .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS))
                      .fontColor(this.getCurrentThemePalette().textTertiary)
                      .width('100%')

                    if (this.canManageAnnouncements()) {
                      Row({ space: DesignConstants.SPACING.SM }) {
                        SystemButton({
                          label: '编辑',
                          variant: 'surface',
                          buttonSize: 'small',
                          onPress: () => { this.openEditAnnouncement(item); }
                        })
                        SystemButton({
                          label: '删除',
                          variant: 'surface-danger',
                          buttonSize: 'small',
                          onPress: () => { this.requestDeleteAnnouncement(item); }
                        })
                      }
                      .width('100%')
                      .justifyContent(FlexAlign.End)
                      .margin({ top: DesignConstants.SPACING.XS })
                    }
                  }
                  .width('100%')
                  .alignItems(HorizontalAlign.Start)
                  .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, top: DesignConstants.SPACING.MD, bottom: DesignConstants.SPACING.MD })
                }
                // 行级卡片样式移除
              }, (item: GroupAnnouncementItem) => item.announcement_id)

              // 删除公告二次确认（危险操作统一样式，对齐前端 ConfirmDialog 语义）
              ListItem() {
                SystemDialog({
                  visible: this.showAnnouncementDeleteConfirm,
                  title: '删除公告',
                  description: `确定删除「${this.deleteAnnouncementTarget?.title ?? ''}」？删除后不可恢复`,
                  confirmText: '删除',
                  confirmVariant: 'danger',
                  onCancel: () => {
                    this.showAnnouncementDeleteConfirm = false;
                    this.deleteAnnouncementTarget = null;
                  },
                  onConfirm: () => { this.handleDeleteAnnouncement(); }
                })
              }
            }
          }
          }  // 公告分组结束

        }
        .width('100%')
        .layoutWeight(1)
        .scrollBar(BarState.Off)
        .divider({ strokeWidth: 0, color: this.getSurfaceBorder() })
        .padding({
          bottom: SafeAreaUtils.resolveBottomPadding(this.safeAreaBottom, DesignConstants.SPACING.LG, DesignConstants.SPACING.LG)
        })
        // 分组卡（surfaceShadow 表达层级；描边已按用户反馈移除——深色下呈裁剪线观感）
        .backgroundColor(this.getSurfaceBackground())
        .borderRadius(DesignConstants.SHAPE.COMPACT_CARD)
        .shadow(this.getCurrentThemePalette().surfaceShadow)
        .transition(
          TransitionEffect.OPACITY.animation({ duration: DesignConstants.ANIMATION.DURATION_SHEET, curve: DesignConstants.ANIMATION.CURVE_FAST_OUT_SLOW_IN })
            .combine(TransitionEffect.move(TransitionEdge.BOTTOM).animation({ duration: DesignConstants.ANIMATION.DURATION_SHEET, curve: DesignConstants.ANIMATION.CURVE_FAST_OUT_SLOW_IN }))
        )
      }
    }
    .width('100%')
    .height('100%')
    }
    }
    .hideTitleBar(true)
    .onReady((context: NavDestinationContext) => {
      this.pathStack = context.pathStack;
      const param = context.pathInfo.param as Record<string, Object> | undefined;
      if (param && param.groupId) {
        this.groupId = param.groupId as string;
        (async () => {
          await this.loadGroupDetail();
        })();
      } else {
        this.errorMessage = '缺少群组ID参数';
      }
    })
    .bindSheet($$this.showMemberList, this.buildMemberListSheet(), {
      height: '70%',
      showClose: false,
      dragBar: true,
      backgroundColor: this.getCurrentThemePalette().sheetBackground
    })
    .bindSheet($$this.showApplicationList, this.buildApplicationListSheet(), {
      height: '70%',
      showClose: false,
      dragBar: true,
      backgroundColor: this.getCurrentThemePalette().sheetBackground
    })
    .bindSheet($$this.showEditGroupSheet, this.buildEditGroupSheet(), {
      height: '78%',
      showClose: false,
      dragBar: true,
      backgroundColor: this.getCurrentThemePalette().sheetBackground
    })
    .bindSheet($$this.showAnnouncementSheet, this.buildAnnouncementSheet(), {
      height: SheetSize.FIT_CONTENT,
      showClose: false,
      dragBar: true,
      backgroundColor: this.getCurrentThemePalette().sheetBackground
    })
    .bindSheet($$this.showInviteSheet, this.buildInviteSheet(), {
      height: '70%',
      showClose: false,
      dragBar: true,
      backgroundColor: this.getCurrentThemePalette().sheetBackground
    })
    .bindSheet($$this.showBannedListSheet, this.buildBannedListSheet(), {
      height: '70%',
      showClose: false,
      dragBar: true,
      backgroundColor: this.getCurrentThemePalette().sheetBackground
    })
    .bindSheet($$this.showBanConfirm, this.buildBanConfirmSheet(), {
      height: SheetSize.FIT_CONTENT,
      showClose: false,
      dragBar: true,
      backgroundColor: this.getCurrentThemePalette().sheetBackground
    })
    .bindSheet($$this.showGroupMoreSheet, this.buildGroupMoreSheet(), {
      height: SheetSize.FIT_CONTENT,
      showClose: false,
      dragBar: true,
      backgroundColor: this.getCurrentThemePalette().sheetBackground
    })
  }

  // 四宫格数据卡（对齐前端 groups/[groupId] 简介区 2×2 stats：成员/在线/距你/分类）
  @Builder
  private buildStatsGrid(): void {
    Column({ space: DesignConstants.SPACING.SM }) {
      Row({ space: DesignConstants.SPACING.SM }) {
        this.StatsCell(formatCount(this.groupInfo?.group?.members_count ?? 0), '成员')
        this.StatsCell(this.formatOnlineCount(), '在线')
      }
      .width('100%')
      Row({ space: DesignConstants.SPACING.SM }) {
        this.StatsCell(this.formatDistanceCell(), '距你')
        this.StatsCell(this.formatCategoryCell(), '分类')
      }
      .width('100%')
    }
    .width('100%')
    // 与卡内位置行的 LG(16) 对齐：瓦片原先 0 内边距贴卡缘（实拍反馈「紧贴边缘」）
    .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG })
  }

  private formatOnlineCount(): string {
    const online = this.groupInfo?.online_members_count;
    return online !== null && online !== undefined ? formatCount(online) : '—';
  }

  private formatDistanceCell(): string {
    const loc = this.groupInfo?.location as RelativeLocation | null;
    return loc?.distance !== undefined && loc?.distance !== null
      ? LocationUtil.formatDistance(loc.distance)
      : '—';
  }

  private formatCategoryCell(): string {
    const tags = this.groupInfo?.tags;
    return tags !== undefined && tags !== null && tags.length > 0 ? `#${tags[0]}` : '—';
  }

  @Builder
  private StatsCell(value: string, label: string) {
    // 按用户重设计方案：去瓦片盒（surfaceSubtle 底+圆角），纯「数值+标签」列
    // 直接落在卡面上，留足呼吸感（数值 XL 半粗 / 标签 XS 次要色，间距放宽）
    Column({ space: DesignConstants.SPACING.SM }) {
      Text(value)
        .fontSize(DesignConstants.FONT.SIZE.XL)
        .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XL))
        .fontWeight(DesignConstants.FONT.WEIGHT.SEMIBOLD)
        .fontColor(this.getCurrentThemePalette().textPrimary)
        .maxLines(1)
        .textOverflow({ overflow: TextOverflow.Ellipsis })
      Text(label)
        .fontSize(DesignConstants.FONT.SIZE.XS)
        .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS))
        .fontColor(this.getCurrentThemePalette().textSecondary)
    }
    .justifyContent(FlexAlign.Center)
    .alignItems(HorizontalAlign.Center)
    .layoutWeight(1)
    .padding({ top: DesignConstants.SPACING.MD, bottom: DesignConstants.SPACING.MD })
  }

  @Builder
  private buildEditGroupSheet() {
    Column({ space: DesignConstants.SPACING.MD }) {
      Row({ space: DesignConstants.SPACING.SM }) {
        Text('编辑群组信息')
          .fontSize(DesignConstants.FONT.SIZE.XL)
          .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XL))
          .fontWeight(DesignConstants.FONT.WEIGHT.BOLD)
          .fontColor(this.getCurrentThemePalette().textPrimary)
          .layoutWeight(1)

        SystemButton({
          label: '取消',
          variant: 'surface',
          buttonSize: 'large',
          shadowed: true,
          disabled: this.isSavingGroupInfo || this.isEditGroupAvatarUploading,
          onPress: () => { this.closeEditGroupSheet(); }
        })

        SystemButton({
          label: '保存',
          variant: 'primary',
          buttonSize: 'large',
          loading: this.isSavingGroupInfo,
          disabled: this.isEditGroupAvatarUploading,
          onPress: () => { this.saveGroupInfo(); }
        })
      }
      .width('100%')
      .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, top: DesignConstants.SPACING.SM })

      Scroll() {
        Column({ space: DesignConstants.SPACING.LG }) {
          Column({ space: DesignConstants.SPACING.MD }) {
            Text('群组头像')
              .fontSize(DesignConstants.FONT.SIZE.SM)
              .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.SM))
              .fontColor(this.getCurrentThemePalette().textSecondary)
              .width('100%')

            Row({ space: DesignConstants.SPACING.MD }) {
              Button() {
                SystemAvatar({
                  src: this.editGroupAvatarPreviewUrl,
                  name: this.editGroupName,
                  avatarSize: DesignConstants.COMPONENT.AVATAR_SIZE.LARGE
                })
              }
              .type(ButtonType.Normal)
              .backgroundColor(DesignConstants.COLOR.TRANSPARENT)
              .onClick(() => this.selectEditGroupAvatar())

              Column({ space: DesignConstants.SPACING.SM }) {
                Text(this.isEditGroupAvatarUploading ? '头像上传中…' : '点击头像可重新选择')
                  .fontSize(DesignConstants.FONT.SIZE.SM)
                  .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.SM))
                  .fontColor(this.getCurrentThemePalette().textSecondary)

                if (this.editGroupAvatarLocalUri) {
                  SystemButton({
                    label: '恢复原头像',
                    variant: 'surface',
                    buttonSize: 'small',
                    onPress: () => this.resetEditGroupAvatarSelection()
                  })
                }
              }
              .alignItems(HorizontalAlign.Start)
              .layoutWeight(1)
            }
            .width('100%')
          }
          .padding(DesignConstants.SPACING.LG)
          .backgroundColor(this.getSurfaceBackground())
          .borderRadius(DesignConstants.RADIUS.XL)
          .shadow(this.getCurrentThemePalette().controlShadow)

          Column({ space: DesignConstants.SPACING.SM }) {
            Text('群组名称')
              .fontSize(DesignConstants.FONT.SIZE.SM)
              .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.SM))
              .fontColor(this.getCurrentThemePalette().textSecondary)
              .width('100%')

            SystemInput({
              value: this.editGroupName,
              placeholder: '请输入群组名称',
              fontSize: DesignConstants.FONT.SIZE.MD,
              inputHeight: DesignConstants.COMPONENT.BUTTON_HEIGHT,
              maxLength: GROUP_NAME_MAX,
              onChange: (value: string) => { this.editGroupName = value; }
            })
               .backgroundColor(this.getSurfaceSubtle())
              .shadow(this.getCurrentThemePalette().controlShadow)
              .borderRadius(DesignConstants.RADIUS.MD)
              .padding({ left: DesignConstants.SPACING.MD, right: DesignConstants.SPACING.MD })
              .onBlur(() => KeyboardDismissManager.getInstance().dismiss())

            Text(`${this.editGroupName.trim().length}/${GROUP_NAME_MAX}`)
              .fontSize(DesignConstants.FONT.SIZE.XS)
              .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS))
              .fontColor(this.getCurrentThemePalette().textTertiary)
              .width('100%')
              .textAlign(TextAlign.End)
          }
          .padding(DesignConstants.SPACING.LG)
          .backgroundColor(this.getSurfaceBackground())
          .borderRadius(DesignConstants.RADIUS.XL)
          .shadow(this.getCurrentThemePalette().controlShadow)

          Column({ space: DesignConstants.SPACING.SM }) {
            Text('群组描述')
              .fontSize(DesignConstants.FONT.SIZE.SM)
              .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.SM))
              .fontColor(this.getCurrentThemePalette().textSecondary)
              .width('100%')

            SystemTextArea({
              value: this.editGroupDescription,
              placeholder: '请输入群组描述',
              minHeight: DesignConstants.COMPONENT.PLACEHOLDER_AREA_HEIGHT,
              maxLength: GROUP_DESCRIPTION_MAX,
              fontSize: DesignConstants.FONT.SIZE.MD,
              onChange: (value: string) => { this.editGroupDescription = value; }
            })

            Text(`${this.editGroupDescription.trim().length}/${GROUP_DESCRIPTION_MAX}`)
              .fontSize(DesignConstants.FONT.SIZE.XS)
              .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS))
              .fontColor(this.getCurrentThemePalette().textTertiary)
              .width('100%')
              .textAlign(TextAlign.End)
          }
          .padding(DesignConstants.SPACING.LG)
          .backgroundColor(this.getSurfaceBackground())
          .borderRadius(DesignConstants.RADIUS.XL)
          .shadow(this.getCurrentThemePalette().controlShadow)

          // 群设置扩展（对齐前端 GroupSettingsSheet：私密群组 / 新人需审核）
          Column({ space: DesignConstants.SPACING.MD }) {
            Text('入群设置')
              .fontSize(DesignConstants.FONT.SIZE.SM)
              .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.SM))
              .fontColor(this.getCurrentThemePalette().textSecondary)
              .width('100%')

            Row({ space: DesignConstants.SPACING.MD }) {
              Column({ space: DesignConstants.SPACING.HALF }) {
                Text(this.editGroupPrivate ? '私密群组' : '公开群组')
                  .fontSize(DesignConstants.FONT.SIZE.MD)
                  .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.MD))
                  .fontColor(this.getCurrentThemePalette().textPrimary)
                Text(this.editGroupPrivate ? '仅受邀请的人可加入' : '所有人都可以找到并加入')
                  .fontSize(DesignConstants.FONT.SIZE.XS)
                  .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS))
                  .fontColor(this.getCurrentThemePalette().textSecondary)
              }
              .alignItems(HorizontalAlign.Start)
              .layoutWeight(1)

              SystemToggle({
                label: '私密群组',
                isOn: this.editGroupPrivate,
                onToggle: (next: boolean) => { this.editGroupPrivate = next; }
              })
            }
            .width('100%')

            Row({ space: DesignConstants.SPACING.MD }) {
              Column({ space: DesignConstants.SPACING.HALF }) {
                Text('新人需审核')
                  .fontSize(DesignConstants.FONT.SIZE.MD)
                  .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.MD))
                  .fontColor(this.getCurrentThemePalette().textPrimary)
                Text(this.editGroupRequiresApproval ? '申请需群主/管理员审核通过后加入' : '申请后直接加入')
                  .fontSize(DesignConstants.FONT.SIZE.XS)
                  .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS))
                  .fontColor(this.getCurrentThemePalette().textSecondary)
              }
              .alignItems(HorizontalAlign.Start)
              .layoutWeight(1)

              SystemToggle({
                label: '入群需审核',
                isOn: this.editGroupRequiresApproval,
                onToggle: (next: boolean) => { this.editGroupRequiresApproval = next; }
              })
            }
            .width('100%')
          }
          .padding(DesignConstants.SPACING.LG)
          .backgroundColor(this.getSurfaceBackground())
          .borderRadius(DesignConstants.RADIUS.XL)
          .shadow(this.getCurrentThemePalette().controlShadow)

          // 危险区（仅群主）：转让群主 / 解散群组（对齐前端 GroupSettingsSheet 危险区）
          if (this.isOwnerRole()) {
            Column({ space: DesignConstants.SPACING.MD }) {
              Row({ space: DesignConstants.SPACING.XS }) {
                LucideIcon({
                  icon: AppIcon.WARNING,
                  iconSize: DesignConstants.COMPONENT.ICON_SIZE_2XS,
                  color: this.getCurrentThemePalette().error
                })
                Text('危险区')
                  .fontSize(DesignConstants.FONT.SIZE.MD)
                  .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.MD))
                  .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
                  .fontColor(this.getCurrentThemePalette().error)
              }
              .width('100%')

              // 转让群主：展开候选成员选择 + 二次确认
              Column({ space: DesignConstants.SPACING.SM }) {
                Row() {
                  Text('转让群主')
                    .fontSize(DesignConstants.FONT.SIZE.MD)
                    .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.MD))
                    .fontColor(this.getCurrentThemePalette().textPrimary)
                    .layoutWeight(1)
                  SystemButton({
                    label: this.transferSectionOpen ? '收起' : '选择成员',
                    variant: 'surface',
                    buttonSize: 'small',
                    onPress: () => { this.toggleTransferSection(); }
                  })
                }
                .width('100%')

                if (this.transferSectionOpen) {
                  if (this.isLoadingTransferCandidates) {
                    Text('成员加载中…')
                      .fontSize(DesignConstants.FONT.SIZE.SM)
                      .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.SM))
                      .fontColor(this.getCurrentThemePalette().textSecondary)
                      .width('100%')
                  } else if (this.transferCandidates.length === 0) {
                    Text('群内暂无其他成员可转让')
                      .fontSize(DesignConstants.FONT.SIZE.SM)
                      .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.SM))
                      .fontColor(this.getCurrentThemePalette().textSecondary)
                      .width('100%')
                  } else {
                    ForEach(this.transferCandidates, (member: GroupMemberItem) => {
                      Row({ space: DesignConstants.SPACING.SM }) {
                        Text(member.user?.nickname ?? '')
                          .fontSize(DesignConstants.FONT.SIZE.SM)
                          .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.SM))
                          .fontColor(this.getCurrentThemePalette().textPrimary)
                          .maxLines(1)
                          .textOverflow({ overflow: TextOverflow.Ellipsis })
                          .layoutWeight(1)
                        Text(this.getRoleShortText(member.role))
                          .fontSize(DesignConstants.FONT.SIZE.XS)
                          .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS))
                          .fontColor(this.getCurrentThemePalette().textSecondary)
                        SystemButton({
                          label: '转让',
                          variant: 'surface',
                          buttonSize: 'small',
                          onPress: () => { this.requestTransferOwnership(member); }
                        })
                      }
                      .width('100%')
                    }, (member: GroupMemberItem) => member.user?.user_id ?? member.joined_at ?? '')
                  }
                }
              }
              .width('100%')

              // 解散群组：不可逆，系统确认弹窗二次确认
              Column({ space: DesignConstants.SPACING.SM }) {
                Text('解散群组')
                  .fontSize(DesignConstants.FONT.SIZE.MD)
                  .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.MD))
                  .fontColor(this.getCurrentThemePalette().textPrimary)
                  .width('100%')
                Text('解散后不可恢复，所有成员将被移出')
                  .fontSize(DesignConstants.FONT.SIZE.XS)
                  .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS))
                  .fontColor(this.getCurrentThemePalette().textSecondary)
                  .width('100%')
                SystemButton({
                  label: '解散群组',
                  variant: 'surface-danger',
                  buttonSize: 'small',
                  disabled: this.isDismissingGroup,
                  onPress: () => { this.requestDismissGroup(); }
                })
              }
              .width('100%')
            }
            .padding(DesignConstants.SPACING.LG)
            .backgroundColor(this.getCurrentThemePalette().dangerSoft)
            .borderRadius(DesignConstants.RADIUS.XL)
            .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getCurrentThemePalette().error })

            // 转让二次确认（危险操作确认弹窗，对齐前端 ConfirmDialog 语义）
            SystemDialog({
              visible: this.showTransferConfirm,
              title: '确定转让群主？',
              description: `转让后将变为普通成员，无法再执行群管理操作。接收者：${this.transferTargetMember?.user?.nickname ?? ''}`,
              confirmText: '确认转让',
              confirmVariant: 'danger',
              onCancel: () => {
                this.showTransferConfirm = false;
                this.transferTargetMember = null;
              },
              onConfirm: () => { this.confirmTransferOwnership(); }
            })

            // 解散二次确认（不可逆操作，系统确认弹窗）
            SystemDialog({
              visible: this.showDismissConfirm,
              title: '解散群组',
              description: '解散后不可恢复，所有成员将被移出。确定解散该群组？',
              confirmText: '解散',
              confirmVariant: 'danger',
              onCancel: () => { this.showDismissConfirm = false; },
              onConfirm: () => { this.confirmDismissGroup(); }
            })
          }
        }
        .width('100%')
        .padding({
          left: DesignConstants.SPACING.LG,
          right: DesignConstants.SPACING.LG,
          top: DesignConstants.SPACING.SM,
          bottom: SafeAreaUtils.resolveBottomPadding(this.safeAreaBottom, DesignConstants.SPACING.LG, 0)
        })
      }
      .width('100%')
      .layoutWeight(1)
      .scrollBar(BarState.Off)
    }
    .width('100%')
    .height('100%')
    .backgroundColor(this.getCurrentThemePalette().pageBackground)
  }

  // ==================== 群公告编辑面板（创建/编辑共用，对齐前端 GroupAnnouncementDialog） ====================
  @Builder
  private buildAnnouncementSheet() {
    // 已提取为子域组件（状态经 @Link 双向同步；保存/取消回调交回页面）
    AnnouncementEditorSheet({
      editing: this.announcementEditing,
      title: this.announcementTitle,
      content: this.announcementContent,
      pinned: this.announcementPinned,
      saving: this.isSavingAnnouncement,
      safeAreaBottom: this.safeAreaBottom,
      onCancel: () => { this.closeAnnouncementSheet(); },
      onSubmit: () => { this.submitAnnouncement(); }
    })
  }

  /** 取消封禁确认（关闭面板并清空目标，避免残留状态误触） */
  private cancelBanMember(): void {
    this.showBanConfirm = false;
    this.banTargetMember = null;
  }

  /** 封禁成员二次确认面板（危险操作防误触；确认后走 confirmBanMember 落库） */
  /** 更多操作 sheet（对齐前端「更多操作」bottom sheet：眉标 + 群名 + 举报群组） */
  @Builder
  private buildGroupMoreSheet() {
    Column({ space: DesignConstants.SPACING.MD }) {
      Row({ space: DesignConstants.SPACING.HALF }) {
        Text('更多操作')
          .fontSize(DesignConstants.FONT.SIZE.XS2)
          .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS2))
          .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
          .fontColor(this.getCurrentThemePalette().textSecondary)
          .letterSpacing(DesignConstants.FONT.LETTER_SPACING.EYEBROW)
          .textCase(TextCase.UpperCase)
      }
      .width('100%')

      Text(this.groupInfo?.group?.name ?? '群组')
        .fontSize(DesignConstants.FONT.SIZE.MD)
        .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.MD))
        .fontColor(this.getCurrentThemePalette().textPrimary)
        .width('100%')
        .maxLines(1)
        .textOverflow({ overflow: TextOverflow.Ellipsis })

      SystemButton({
        label: '举报群组',
        variant: 'surface-danger',
        buttonSize: 'sheet',
        fullWidth: true,
        onPress: () => {
          this.showGroupMoreSheet = false;
          this.showGroupReportConfirm = true;
        }
      })

      SystemButton({
        label: '关闭',
        variant: 'surface-accent',
        buttonSize: 'sheet',
        fullWidth: true,
        onPress: () => { this.showGroupMoreSheet = false; }
      })
    }
    .width('100%')
    .padding({
      left: DesignConstants.SPACING.LG,
      right: DesignConstants.SPACING.LG,
      top: DesignConstants.SPACING.LG,
      bottom: DesignConstants.SPACING.XL
    })
  }

  /** 举报确认（对齐聊天页举报范式：SystemDialog danger 确认 → createReport） */
  private async submitGroupReport(): Promise<void> {
    if (this.groupReportSubmitting) {
      return;
    }
    this.groupReportSubmitting = true;
    try {
      const reportReq: CreateReportRequest = {
        target_type: 'group',
        target_id: this.groupInfo?.group?.group_id ?? '',
        reason: 'other',
        description: '来自群详情页面的举报'
      };
      await ApiClient.createReport(reportReq);
      SystemToast.show('举报已提交', this.getUIContext());
    } catch {
      SystemToast.showError('举报提交失败', this.getUIContext());
    } finally {
      this.groupReportSubmitting = false;
    }
  }

  @Builder
  private buildBanConfirmSheet() {
    Column({ space: DesignConstants.SPACING.LG }) {
      Text('封禁并移出')
        .fontSize(DesignConstants.FONT.SIZE.XL)
        .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XL))
        .fontWeight(DesignConstants.FONT.WEIGHT.BOLD)
        .fontColor(this.getCurrentThemePalette().textPrimary)
        .width('100%')

      Text(`确定将「${this.banTargetMember?.user?.nickname ?? '该成员'}」封禁并移出群组？封禁后对方将无法再加入。`)
        .fontSize(DesignConstants.FONT.SIZE.SM)
        .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.SM))
        .fontColor(this.getCurrentThemePalette().textSecondary)
        .width('100%')

      Row({ space: DesignConstants.SPACING.SM }) {
        SystemButton({
          label: '取消',
          variant: 'surface',
          buttonSize: 'sheet',
          fullWidth: true,
          onPress: () => { this.cancelBanMember(); }
        })
        SystemButton({
          label: '封禁并移出',
          variant: 'danger',
          buttonSize: 'sheet',
          fullWidth: true,
          onPress: () => { this.confirmBanMember(); }
        })
      }
      .width('100%')
    }
    .width('100%')
    .padding({
      left: DesignConstants.SPACING.XXL,
      right: DesignConstants.SPACING.XXL,
      top: DesignConstants.SPACING.LG,
      bottom: DesignConstants.SPACING.XXL
    })
  }

  // ==================== 邀请成员面板（搜索 + 多选 + 批量邀请，对齐前端 InviteMembersSheet） ====================
  @Builder
  private buildInviteSheet() {
    Column() {
      Row() {
        Text('邀请成员入群')
          .fontSize(DesignConstants.FONT.SIZE.XL)
          .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XL))
          .fontWeight(DesignConstants.FONT.WEIGHT.BOLD)
          .fontColor(this.getCurrentThemePalette().textPrimary)
          .layoutWeight(1)
        Text(`${this.inviteSelectedIds.length} 人已选`)
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.SM))
          .fontColor(this.getCurrentThemePalette().textSecondary)
      }
      .width('100%')
      .padding(DesignConstants.SPACING.LG)

      SystemInput({
        value: this.inviteKeyword,
        placeholder: '搜索用户昵称…',
        fontSize: DesignConstants.FONT.SIZE.MD,
        inputHeight: DesignConstants.COMPONENT.BUTTON_HEIGHT,
        onChange: (value: string) => { this.handleInviteKeywordChange(value); }
      })
        .margin({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG })
        .backgroundColor(this.getSurfaceSubtle())
        .shadow(this.getCurrentThemePalette().controlShadow)
        .borderRadius(DesignConstants.RADIUS.MD)

      if (this.isSearchingInviteUsers) {
        Column() {
          SystemLoadingView({ message: '搜索中…' })
        }
        .layoutWeight(1)
      } else if (this.inviteSearchError.length > 0) {
        Column() {
          EmptyState({
            bare: true,
            icon: AppIcon.WARNING,
            title: this.inviteSearchError,
            actionLabel: '重试',
            onAction: () => { this.searchInviteUsers(this.inviteKeyword.trim()); }
          })
        }
        .layoutWeight(1)
      } else if (this.inviteResults.length === 0) {
        Column() {
          EmptyState({
            icon: AppIcon.EMPTY,
            message: this.inviteKeyword.trim().length > 0 ? '无匹配用户' : '输入昵称搜索用户',
            bare: true
          })
        }
        .layoutWeight(1)
      } else {
        List() {
          ForEach(this.inviteResults, (user: UserItem) => {
            ListItem() {
              Row({ space: DesignConstants.SPACING.SM }) {
                SystemAvatar({
                  src: user.user?.avatar_url ?? '',
                  name: user.user?.nickname ?? '',
                  avatarSize: DesignConstants.COMPONENT.AVATAR_SIZE.XS
                })
                Text(user.user?.nickname ?? '')
                  .fontSize(DesignConstants.FONT.SIZE.MD)
                  .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.MD))
                  .fontColor(this.getCurrentThemePalette().textPrimary)
                  .maxLines(1)
                  .textOverflow({ overflow: TextOverflow.Ellipsis })
                  .layoutWeight(1)
                SystemButton({
                  label: this.isInviteUserSelected(user.user?.user_id ?? '') ? '已选择' : '选择',
                  variant: this.isInviteUserSelected(user.user?.user_id ?? '') ? 'surface-accent' : 'surface',
                  buttonSize: 'small',
                  onPress: () => { this.toggleInviteUser(user.user?.user_id ?? ''); }
                })
              }
              .width('100%')
              .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, top: DesignConstants.SPACING.SM, bottom: DesignConstants.SPACING.SM })
            }
          }, (user: UserItem) => user.user?.user_id ?? '')
        }
        .width('100%')
        .layoutWeight(1)
        .scrollBar(BarState.Off)
      }

      Column() {
        SystemButton({
          label: this.isInviting ? '邀请中…' :
            (this.inviteSelectedIds.length > 0 ? `邀请 ${this.inviteSelectedIds.length} 人` : '邀请'),
          variant: 'primary',
          buttonSize: 'default',
          pill: true,
          fullWidth: true,
          loading: this.isInviting,
          disabled: this.inviteSelectedIds.length === 0,
          onPress: () => { this.submitInvites(); }
        })
      }
      .width('100%')
      .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, bottom: SafeAreaUtils.resolveBottomPadding(this.safeAreaBottom, DesignConstants.SPACING.LG, 0) })
    }
    .width('100%')
    .height('100%')
    .backgroundColor(this.getCurrentThemePalette().pageBackground)
  }

  // ==================== 封禁名单面板（解封，对齐前端 BannedList） ====================
  @Builder
  private buildBannedListSheet() {
    Column() {
      Row() {
        Text('封禁名单')
          .fontSize(DesignConstants.FONT.SIZE.XL)
          .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XL))
          .fontWeight(DesignConstants.FONT.WEIGHT.BOLD)
          .fontColor(this.getCurrentThemePalette().textPrimary)
          .layoutWeight(1)
        Text(`${this.bannedMembers.length} 人`)
          .fontSize(DesignConstants.FONT.SIZE.MD)
          .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.MD))
          .fontColor(this.getCurrentThemePalette().textSecondary)
      }
      .width('100%')
      .padding(DesignConstants.SPACING.LG)

      if (this.isLoadingBannedMembers) {
        Column() {
          SystemLoadingView({ message: '加载封禁名单中…' })
        }
        .layoutWeight(1)
      } else if (this.bannedMembers.length === 0) {
        Column() {
          EmptyState({
            icon: AppIcon.EMPTY,
            title: '暂无封禁成员',
            bare: true,
            fillMode: true
          })
        }
        .layoutWeight(1)
      } else {
        List() {
          ForEach(this.bannedMembers, (banned: AdminBannedMemberItem) => {
            ListItem() {
              Row({ space: DesignConstants.SPACING.SM }) {
                SystemAvatar({
                  src: banned.user?.avatar_url ?? '',
                  name: banned.user?.nickname ?? '',
                  avatarSize: DesignConstants.COMPONENT.AVATAR_SIZE.XS
                })
                Column({ space: DesignConstants.SPACING.HALF }) {
                  Text(banned.user?.nickname ?? '')
                    .fontSize(DesignConstants.FONT.SIZE.MD)
                    .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.MD))
                    .fontColor(this.getCurrentThemePalette().textPrimary)
                    .maxLines(1)
                    .textOverflow({ overflow: TextOverflow.Ellipsis })
                  if (banned.reason !== null && banned.reason !== undefined && banned.reason.length > 0) {
                    Text(`原因：${banned.reason}`)
                      .fontSize(DesignConstants.FONT.SIZE.XS)
                      .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS))
                      .fontColor(this.getCurrentThemePalette().textSecondary)
                      .maxLines(1)
                      .textOverflow({ overflow: TextOverflow.Ellipsis })
                  }
                }
                .alignItems(HorizontalAlign.Start)
                .layoutWeight(1)
                SystemButton({
                  label: '解封',
                  variant: 'surface',
                  buttonSize: 'small',
                  onPress: () => { this.unbanMember(banned); }
                })
              }
              .width('100%')
              .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, top: DesignConstants.SPACING.SM, bottom: DesignConstants.SPACING.SM })
            }
          }, (banned: AdminBannedMemberItem) => banned.user?.user_id ?? banned.banned_at)
        }
        .width('100%')
        .layoutWeight(1)
        .scrollBar(BarState.Off)
      }
    }
    .padding({ bottom: SafeAreaUtils.resolveBottomPadding(this.safeAreaBottom, DesignConstants.SPACING.LG, 0) })
    .width('100%')
    .height('100%')
    .backgroundColor(this.getCurrentThemePalette().pageBackground)
  }

  @Builder
  buildMemberListSheet() {
    GroupMemberListSheet({
      members: this.members,
      isLoadingMembers: this.isLoadingMembers,
      membersErrorMessage: this.membersErrorMessage,
      membersLoadingMore: this.membersLoadingMore,
      membersHasMore: this.membersHasMore,
      safeAreaBottom: this.safeAreaBottom,
      membersDataSource: this.membersDataSource,
      onRetryLoad: () => { this.loadGroupMembers(); },
      onLoadMore: () => { this.loadMoreMembers(); },
      roleTextFor: (member: GroupMemberItem): string => this.getRoleDisplayText(member.role),
      roleColorFor: (member: GroupMemberItem): ResourceColor => this.getRoleColor(member.role),
      showMenuFor: (member: GroupMemberItem): boolean => this.hasMemberActions(member),
      menuItemsFor: (member: GroupMemberItem): MenuElement[] => this.buildMemberActionMenus(member)
    })
  }

  @Builder
  buildApplicationListSheet() {
    GroupApplicationListSheet({
      applications: this.applications,
      isLoadingApplications: this.isLoadingApplications,
      applicationsErrorMessage: this.applicationsErrorMessage,
      applicationsLoadingMore: this.applicationsLoadingMore,
      applicationsHasMore: this.applicationsHasMore,
      safeAreaBottom: this.safeAreaBottom,
      applicationsDataSource: this.applicationsDataSource,
      onRetryLoad: () => { this.loadGroupApplications(); },
      onLoadMore: () => { this.loadMoreGroupApplications(); },
      handlingFor: (applicationId: string): boolean => this.isApplicationHandling(applicationId),
      onApprove: (application: GroupApplicationItem): void => { this.approveApplication(application); },
      onReject: (application: GroupApplicationItem): void => { this.rejectApplication(application); }
    })
  }
}

@Builder
export function GroupDetailPageBuilder() {
  GroupDetailPage()
}