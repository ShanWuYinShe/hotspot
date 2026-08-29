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
import { PageShellContainer } from '../../../components/PageShellContainer';
import type { BreakpointType } from '../../../core/responsive/Breakpoint';
import { SystemLoadingView } from '../../../components/SystemLoadingView';
import { EmptyState } from '../../../components/EmptyState';
import { SystemPageHeader } from '../../../components/SystemPageHeader';
import { SystemToast } from '../../../components/SystemToast';
import { SystemButton } from '../../../components/SystemButton';
import { SystemInput } from '../../../components/SystemInput';
import { SystemTextArea } from '../../../components/SystemTextArea';
import { IconButton, AppIcon, IconSize } from '../../../components/SystemIcon';
import { ShareActionHelper } from '../../../core/utils/ShareActionHelper';
import { SystemShareService } from '../../../core/services/SystemShareService';
import { common } from '@kit.AbilityKit';
import { DesignConstants } from '../../../constants/DesignConstants';
import {
  GroupItem,
  GroupApplicationItem,
  GroupMemberItem,
  GroupSharePayload,
  RelativeLocation,
  SettableGroupRole,
  UpdateGroupRequest
} from '../../../core/types/api';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { InteractionStateStore } from '../../../core/state/InteractionStateStore';
import { DistributedSyncService, SyncData } from '../../../core/services/DistributedSyncService';
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
import { formatCount } from '../../../core/utils/FormatUtil';
import { LocationUtil } from '../../../core/location/LocationUtil';
import { KeyboardDismissManager } from '../../../core/utils/KeyboardDismissManager';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { KEY_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';
import { getErrorMessage } from '../../../core/network/ApiErrorUtil';

interface GroupSyncData {
  groupId: string;
  groupInfo: Object;
  members: Object[];
  lastViewed: number;
}

const DOMAIN_NUMBER: number = 0xFF17;
const TAG: string = 'GroupDetailPage';
const GROUP_NAME_MAX: number = 50;
const GROUP_DESCRIPTION_MAX: number = 500;

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
  @StorageLink(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'vibrant_light';
  
  private stateManager = GlobalStateManager.getInstance();
  private distributedSyncService = DistributedSyncService.getInstance();
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
  @State joinPassword: string = '';
  private groupId: string = '';
  @State handlingApplicationIds: string[] = [];

  @State isMember: boolean = false;
  @State showEditGroupSheet: boolean = false;
  @State editGroupName: string = '';
  @State editGroupDescription: string = '';
  @State editGroupAvatarPreviewUrl: string = '';
  @State editGroupAvatarLocalUri: string = '';
  @State editGroupAvatarFileId: string = '';
  @State isSavingGroupInfo: boolean = false;
  @State isEditGroupAvatarUploading: boolean = false;

  aboutToAppear(): void {
    SystemUiController.syncWithCurrentTheme();
  }

  aboutToDisappear(): void {
    KeyboardDismissManager.getInstance().dismiss();
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
          await this.syncGroupData();
          await this.accessibilityManager.announceText(`已加载群组：${this.groupInfo.group?.name ?? ''}`);
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
        } else if (response.code === 10400) {
          SystemToast.show(getErrorMessage(response, '加入群组失败'), this.getUIContext());
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
    if (trimmedName === currentName && trimmedDescription === currentDescription && !hasAvatarChange) {
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
        avatar_file_id: avatarFileId
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
        action: () => { this.muteMember(member, 30); }
      });
    }
    menus.push({
      value: '移出群组',
      action: () => { this.removeMember(member); }
    });
    return menus;
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

  private async syncGroupData(): Promise<void> {
    if (!this.groupInfo) return;
    const userId = await AuthAccountManager.getCurrentUserId();
    if (!userId) return;
    try {
      const syncDataObject: GroupSyncData = {
        groupId: this.groupId,
        groupInfo: this.groupInfo,
        members: this.members,
        lastViewed: Date.now()
      };
      
      const syncData: SyncData = {
        userId: userId,
        dataType: 'groups',
        data: syncDataObject,
        timestamp: Date.now(),
        deviceId: 'current_device'
      };
      
      await this.distributedSyncService.syncUserData(syncData);
    } catch {
      hilog.error(DOMAIN_NUMBER, TAG, '同步群组数据失败');
    }
  }

  build() {
    NavDestination() {
    PageShellContainer({ variant: 'default', currentBreakpoint: this.currentBreakpoint }) {
    Column() {
      // 标题栏 —— 统一走 SystemPageHeader（居中标题 + 返回 + 分享）
      SystemPageHeader({
        title: '群组详情',
        showBackButton: true,
        onBack: () => { AppRouter.pop(); },
        titleAlign: TextAlign.Center,
        titleSize: DesignConstants.FONT.SIZE.XL,
        titleWeight: DesignConstants.FONT.WEIGHT.MEDIUM,
        rightAction: {
          icon: AppIcon.SHARE,
          onClick: () => this.shareGroup()
        }
      })

      if (this.isLoading) {
        SystemLoadingView({ message: '加载群组信息中...' })
          .layoutWeight(1)
          .transition(
            TransitionEffect.OPACITY.animation({ duration: DesignConstants.ANIMATION.DURATION_SHEET, curve: DesignConstants.ANIMATION.CURVE_FAST_OUT_SLOW_IN })
              .combine(TransitionEffect.move(TransitionEdge.BOTTOM).animation({ duration: DesignConstants.ANIMATION.DURATION_SHEET, curve: DesignConstants.ANIMATION.CURVE_FAST_OUT_SLOW_IN }))
          )
      } else if (this.errorMessage) {
        EmptyState({
          icon: AppIcon.WARNING,
          title: this.errorMessage,
          actionLabel: '重试',
          fillMode: true,
          onAction: () => { this.loadGroupDetail(); }
        })
        } else if (this.groupInfo) {
        // 使用 List 布局（扁平风格）
        List() {
          // 1. 基本信息区域
          ListItem() {
            Column() {
              // 头像和名称
              Column() {
                SystemAvatar({
                  src: this.groupInfo?.group?.avatar_url ?? '',
                  name: this.groupInfo?.group?.name ?? '',
                  avatarSize: DesignConstants.COMPONENT.AVATAR_SIZE.XLARGE
                })
                Text(this.groupInfo?.group?.name ?? '')
                  .fontSize(DesignConstants.FONT.SIZE.XL)
                  .fontWeight(DesignConstants.FONT.WEIGHT.BOLD)
                  .fontColor(this.getCurrentThemePalette().textPrimary)
                  .margin({ top: DesignConstants.SPACING.MD })

                if (this.groupInfo?.description) {
                  Text(this.groupInfo?.description)
                    .fontSize(DesignConstants.FONT.SIZE.MD)
                    .fontColor(this.getCurrentThemePalette().textSecondary)
                    .textAlign(TextAlign.Center)
                    .maxLines(3)
                    .textOverflow({ overflow: TextOverflow.Ellipsis })
                    .margin({ top: DesignConstants.SPACING.SM })
                }
              }
              .width('100%')
              .padding({ top: DesignConstants.SPACING.XL, bottom: DesignConstants.SPACING.XL })

              // 统计信息
              Row() {
                Column() {
                  Text(formatCount(this.groupInfo?.group?.members_count))
                    .fontSize(DesignConstants.FONT.SIZE.LG)
                    .fontWeight(DesignConstants.FONT.WEIGHT.BOLD)
                    .fontColor(this.getCurrentThemePalette().textPrimary)
                  Text('成员')
                    .fontSize(DesignConstants.FONT.SIZE.SM)
                    .fontColor(this.getCurrentThemePalette().textSecondary)
                    .margin({ top: DesignConstants.SPACING.XS })
                }
                .layoutWeight(1)

                Column() {
                  Text((this.groupInfo?.location as RelativeLocation)?.distance
                    ? `${Math.round((this.groupInfo!.location as RelativeLocation).distance!)}m` : '--')
                    .fontSize(DesignConstants.FONT.SIZE.LG)
                    .fontWeight(DesignConstants.FONT.WEIGHT.BOLD)
                    .fontColor(this.getCurrentThemePalette().textPrimary)
                  Text('距离')
                    .fontSize(DesignConstants.FONT.SIZE.SM)
                    .fontColor(this.getCurrentThemePalette().textSecondary)
                    .margin({ top: DesignConstants.SPACING.XS })
                }
                .layoutWeight(1)
              }
              .width('100%')
              .padding({ bottom: DesignConstants.SPACING.XL })
              .justifyContent(FlexAlign.SpaceAround)
            }
            .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG })
          }
          .backgroundColor(this.getSurfaceBackground())
          .borderRadius(DesignConstants.RADIUS.LG)
          .shadow(this.getCurrentThemePalette().elevatedShadow)
          .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getSurfaceBorder() })
          .margin({ bottom: DesignConstants.SPACING.MD })

          // 2. 位置信息
          ListItem() {
             Column() {
                Row() {
                  Text() {
                    SymbolSpan($r(AppIcon.LOCATION))
                      .fontSize(DesignConstants.FONT.SIZE.LG_PLUS)
                      .fontColor([this.getCurrentThemePalette().accentBackground])
                  }
                  Text('位置信息')
                    .fontSize(DesignConstants.FONT.SIZE.MD)
                    .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
                    .fontColor(this.getCurrentThemePalette().textPrimary)
                    .margin({ left: DesignConstants.SPACING.SM })
                }
                .width('100%')
                .margin({ bottom: DesignConstants.SPACING.SM })

                Text(this.groupInfo?.location?.location_name || '未知位置')
                  .fontSize(DesignConstants.FONT.SIZE.MD)
                  .fontColor(this.getCurrentThemePalette().textSecondary)
                  .width('100%')

                if ((this.groupInfo?.location as RelativeLocation)?.distance !== undefined &&
                  (this.groupInfo!.location as RelativeLocation).distance! >= 0) {
                  Text(`距你 ${LocationUtil.formatDistance((this.groupInfo!.location as RelativeLocation).distance!)}`)
                    .fontSize(DesignConstants.FONT.SIZE.SM)
                    .fontColor(this.getCurrentThemePalette().textTertiary)
                    .width('100%')
                    .margin({ top: DesignConstants.SPACING.XS })
                }
             }
             .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG,
               top: DesignConstants.SPACING.MD, bottom: DesignConstants.SPACING.MD })
          }
          .backgroundColor(this.getSurfaceBackground())
          .borderRadius(DesignConstants.RADIUS.LG)
          .shadow(this.getCurrentThemePalette().elevatedShadow)
          .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getSurfaceBorder() })
          .margin({ bottom: DesignConstants.SPACING.MD })

          // 3. 成员列表入口
          ListItem() {
            Row() {
              Column() {
                 Text('群组成员')
                  .fontSize(DesignConstants.FONT.SIZE.MD)
                  .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
                  .fontColor(this.getCurrentThemePalette().textPrimary)
                 Text(`${formatCount(this.groupInfo?.group?.members_count)} 名成员`)
                  .fontSize(DesignConstants.FONT.SIZE.SM)
                  .fontColor(this.getCurrentThemePalette().textSecondary)
                  .margin({ top: DesignConstants.SPACING.HALF })
              }
              .layoutWeight(1)
              .alignItems(HorizontalAlign.Start)

              Text() {
                SymbolSpan($r(AppIcon.FORWARD))
                  .fontSize(DesignConstants.FONT.SIZE.LG)
                  .fontColor([this.getCurrentThemePalette().textSecondary])
              }
            }
            .width('100%')
            .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG,
              top: DesignConstants.SPACING.MD, bottom: DesignConstants.SPACING.MD })
          }
          .backgroundColor(this.getSurfaceBackground())
          .borderRadius(DesignConstants.RADIUS.LG)
          .shadow(this.getCurrentThemePalette().elevatedShadow)
          .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getSurfaceBorder() })
          .margin({ bottom: DesignConstants.SPACING.MD })
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
                    .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
                    .fontColor(this.getCurrentThemePalette().textPrimary)
                  Text('处理待审核的入群请求')
                    .fontSize(DesignConstants.FONT.SIZE.SM)
                    .fontColor(this.getCurrentThemePalette().textSecondary)
                    .margin({ top: DesignConstants.SPACING.HALF })
                }
                .layoutWeight(1)
                .alignItems(HorizontalAlign.Start)

                Text() {
                  SymbolSpan($r(AppIcon.FORWARD))
                    .fontSize(DesignConstants.FONT.SIZE.LG)
                    .fontColor([this.getCurrentThemePalette().textSecondary])
                }
              }
              .width('100%')
              .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG,
                top: DesignConstants.SPACING.MD, bottom: DesignConstants.SPACING.MD })
            }
            .backgroundColor(this.getSurfaceBackground())
            .borderRadius(DesignConstants.RADIUS.LG)
            .shadow(this.getCurrentThemePalette().elevatedShadow)
            .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getSurfaceBorder() })
            .margin({ bottom: DesignConstants.SPACING.MD })
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

          // 4. 操作按钮区域
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
                    onPress: () => { this.leaveGroup(); }
                  })
                    .opacity(this.isLeaving ? DesignConstants.OPACITY.LOADING : 1)
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
          .backgroundColor(this.getSurfaceBackground())
          .borderRadius(DesignConstants.RADIUS.LG)
          .shadow(this.getCurrentThemePalette().elevatedShadow)
          .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getSurfaceBorder() })

        }
        .width('100%')
        .layoutWeight(1)
        .scrollBar(BarState.Auto)
        .divider({ strokeWidth: 0, color: this.getSurfaceBorder() })
        .padding({
          bottom: SafeAreaUtils.resolveBottomPadding(this.safeAreaBottom, DesignConstants.SPACING.LG, 0)
        })
        .transition(
          TransitionEffect.OPACITY.animation({ duration: DesignConstants.ANIMATION.DURATION_SHEET, curve: DesignConstants.ANIMATION.CURVE_FAST_OUT_SLOW_IN })
            .combine(TransitionEffect.move(TransitionEdge.BOTTOM).animation({ duration: DesignConstants.ANIMATION.DURATION_SHEET, curve: DesignConstants.ANIMATION.CURVE_FAST_OUT_SLOW_IN }))
        )
      }
    }
    .width('100%')
    .height('100%')
    .backgroundColor(this.getCurrentThemePalette().pageBackground)
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
  }

  @Builder
  private buildEditGroupSheet() {
    Column({ space: DesignConstants.SPACING.MD }) {
      Row({ space: DesignConstants.SPACING.SM }) {
        Text('编辑群组信息')
          .fontSize(DesignConstants.FONT.SIZE.XL)
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
                Text(this.isEditGroupAvatarUploading ? '头像上传中...' : '点击头像可重新选择')
                  .fontSize(DesignConstants.FONT.SIZE.SM)
                  .fontColor(this.getCurrentThemePalette().textSecondary)

                if (this.editGroupAvatarLocalUri) {
                  Button('恢复原头像')
                    .type(ButtonType.Normal)
                    .backgroundColor(DesignConstants.COLOR.TRANSPARENT)
                    .fontColor(this.getCurrentThemePalette().textSecondary)
                    .borderRadius(DesignConstants.RADIUS.XL)
                    .height(DesignConstants.COMPONENT.FILTER_CHIP_HEIGHT)
                    .onClick(() => this.resetEditGroupAvatarSelection())
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
              .fontColor(this.getCurrentThemePalette().textTertiary)
              .width('100%')
              .textAlign(TextAlign.End)
          }
          .padding(DesignConstants.SPACING.LG)
          .backgroundColor(this.getSurfaceBackground())
          .borderRadius(DesignConstants.RADIUS.XL)
          .shadow(this.getCurrentThemePalette().controlShadow)
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
      .scrollBar(BarState.Auto)
    }
    .width('100%')
    .height('100%')
    .backgroundColor(this.getCurrentThemePalette().pageBackground)
  }

  @Builder
  buildMemberListSheet() {
    Column() {
      // 标题
      Row() {
        Text('群组成员')
          .fontSize(DesignConstants.FONT.SIZE.XL)
          .fontWeight(DesignConstants.FONT.WEIGHT.BOLD)
          .fontColor(this.getCurrentThemePalette().textPrimary)
          .layoutWeight(1)

        Text(`${this.members.length} 人`)
          .fontSize(DesignConstants.FONT.SIZE.MD)
          .fontColor(this.getCurrentThemePalette().textSecondary)
      }
      .width('100%')
      .padding(DesignConstants.SPACING.LG)

      // 成员列表
      if (this.isLoadingMembers && this.members.length === 0) {
        SystemLoadingView({ message: '加载成员列表中...' }).layoutWeight(1)
      } else if (this.members.length === 0 && this.membersErrorMessage) {
        EmptyState({
          icon: AppIcon.WARNING,
          title: this.membersErrorMessage,
          actionLabel: '重试',
          fillMode: true,
          onAction: () => { this.loadGroupMembers(); }
        })
      } else if (this.members.length > 0) {
        List() {
          LazyForEach(this.membersDataSource, (member: GroupMemberItem) => {
            ListItem() {
              GroupMemberRow({
                member: member,
                roleText: this.getRoleDisplayText(member.role),
                roleColor: this.getRoleColor(member.role),
                showMenu: this.buildMemberActionMenus(member).length > 0,
                palette: this.getCurrentThemePalette(),
                surfaceBackground: this.getSurfaceBackground(),
                menuItems: this.buildMemberActionMenus(member)
              })
            }
          }, (member: GroupMemberItem) => member.user?.user_id ?? member.joined_at ?? '')
          if (this.membersLoadingMore) {
            ListItem() {
              Row({ space: DesignConstants.SPACING.SM }) {
                LoadingProgress().width(DesignConstants.COMPONENT.LOADING_ICON_SIZE).height(DesignConstants.COMPONENT.LOADING_ICON_SIZE)
                Text('加载更多...')
                  .fontSize(DesignConstants.FONT.SIZE.SM)
                  .fontColor(this.getCurrentThemePalette().textSecondary)
              }
              .width('100%')
              .justifyContent(FlexAlign.Center)
              .padding(DesignConstants.SPACING.MD)
            }
          }
        }
        .width('100%')
        .layoutWeight(1)
        .padding({ left: DesignConstants.SPACING.MD, right: DesignConstants.SPACING.MD })
        .scrollBar(BarState.Auto)
        .cachedCount(5)
        .onReachEnd(() => { this.loadMoreMembers(); })
      } else {
        EmptyState({
          icon: AppIcon.EMPTY,
          title: '暂无成员信息',
          bare: true,
          fillMode: true
        })
      }
    }
    .padding({ bottom: SafeAreaUtils.resolveBottomPadding(this.safeAreaBottom, DesignConstants.SPACING.LG, 0) })
    .width('100%')
    .height('100%')
    .backgroundColor(this.getCurrentThemePalette().pageBackground)
  }

  @Builder
  buildApplicationListSheet() {
    Column() {
      Row() {
        Text('入群申请')
          .fontSize(DesignConstants.FONT.SIZE.XL)
          .fontWeight(DesignConstants.FONT.WEIGHT.BOLD)
          .fontColor(this.getCurrentThemePalette().textPrimary)
          .layoutWeight(1)

        Text(`${this.applications.length} 条`)
          .fontSize(DesignConstants.FONT.SIZE.MD)
          .fontColor(this.getCurrentThemePalette().textSecondary)
      }
      .width('100%')
      .padding(DesignConstants.SPACING.LG)

      if (this.isLoadingApplications && this.applications.length === 0) {
        SystemLoadingView({ message: '加载入群申请中...' }).layoutWeight(1)
      } else if (this.applications.length === 0 && this.applicationsErrorMessage) {
        EmptyState({
          icon: AppIcon.WARNING,
          title: this.applicationsErrorMessage,
          actionLabel: '重试',
          fillMode: true,
          onAction: () => { this.loadGroupApplications(); }
        })
      } else if (this.applications.length > 0) {
        List() {
          LazyForEach(this.applicationsDataSource, (application: GroupApplicationItem) => {
            ListItem() {
              GroupApplicationRow({
                application: application,
                handling: this.isApplicationHandling(application.application_id),
                palette: this.getCurrentThemePalette(),
                surfaceBackground: this.getSurfaceBackground(),
                onApprove: () => { this.approveApplication(application); },
                onReject: () => { this.rejectApplication(application); }
              })
            }
          }, (application: GroupApplicationItem) => application.application_id)

          if (this.applicationsLoadingMore) {
            ListItem() {
              Row({ space: DesignConstants.SPACING.SM }) {
                LoadingProgress().width(DesignConstants.COMPONENT.LOADING_ICON_SIZE).height(DesignConstants.COMPONENT.LOADING_ICON_SIZE)
                Text('加载更多...')
                  .fontSize(DesignConstants.FONT.SIZE.SM)
                  .fontColor(this.getCurrentThemePalette().textSecondary)
              }
              .width('100%')
              .justifyContent(FlexAlign.Center)
              .padding(DesignConstants.SPACING.MD)
            }
          }
        }
        .width('100%')
        .layoutWeight(1)
        .padding({ left: DesignConstants.SPACING.MD, right: DesignConstants.SPACING.MD })
        .scrollBar(BarState.Auto)
        .cachedCount(5)
        .onReachEnd(() => { this.loadMoreGroupApplications(); })
      } else {
        EmptyState({
          icon: AppIcon.EMPTY,
          title: '暂无待处理申请',
          bare: true,
          fillMode: true
        })
      }
    }
    .padding({ bottom: SafeAreaUtils.resolveBottomPadding(this.safeAreaBottom, DesignConstants.SPACING.LG, 0) })
    .width('100%')
    .height('100%')
    .backgroundColor(this.getCurrentThemePalette().pageBackground)
  }
}

@Builder
export function GroupDetailPageBuilder() {
  GroupDetailPage()
}