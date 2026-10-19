/**
 * 个人中心页面
 * 三级用户体系：
 * - 匿名（无 token）：游客浏览模式，引导登录/注册
 * - 临时用户（temp）：已认证，显示升级入口
 * - 注册用户（normal）：完整功能
 */

import { GlobalStateManager, KEY_CURRENT_USER, KEY_IS_LOGGED_IN, KEY_USER_TYPE, KEY_CURRENT_LOCATION } from '../../../core/state/GlobalStateManager';
import { isValidUsernameLoginId, isValidNickname as isValidNicknameShared, isValidPassword as isValidPasswordShared } from '../../../core/utils/AccountValidation';
import { interactionEventHub, InteractionStateStore, UserRelationState, AuthUserChangedState } from '../../../core/state/InteractionStateStore';
import { interactionActions } from '../../../core/state/InteractionActions';
import { LocationInfo } from '../../../core/location/LocationUtil';
import { ApiClient } from '../../../core/api/ApiClient';
import { HttpUtil } from '../../../core/network/HttpUtil';
import { WebSocketService } from '../../../core/network/WebSocketService';
import { AppRouter, ChatRouteParam, LoginRouteParam } from '../../../core/router/AppRouter';
import { PageNames } from '../../../constants/AppConstants';
import { MediaServiceManager } from '../../../core/media/MediaServiceManager';
import { ViewedMomentThumb } from '../components/ViewedMomentThumb';
import { SystemAvatar } from '../../../components/SystemAvatar';
import { Pressable } from '../../../components/Pressable';
import { useMasterDetail } from '../../../core/responsive/Breakpoint';
import type { BreakpointType } from '../../../core/responsive/Breakpoint';
import { PageShellContainer } from '../../../components/PageShellContainer';
import { SystemToast } from '../../../components/SystemToast';
import { SystemInput } from '../../../components/SystemInput';
import { SystemTextArea } from '../../../components/SystemTextArea';
import { SystemLoadingView } from '../../../components/SystemLoadingView';
import { LoginPromptView } from '../../../components/LoginPromptView';
import { AccountInfo, UpdateLocationRequest, UpgradeAccountRequest, UserItem, OpenUserDoorRequest, MomentItem } from '../../../core/types/api';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { common } from '@kit.AbilityKit';
import { DesignConstants, textLineHeight } from '../../../constants/DesignConstants';
import { AppIcon } from '../../../components/SystemIcon';
import { LucideIcon } from '../../../components/LucideIcon';
import { EmptyState } from '../../../components/EmptyState';
import { SystemPageHeader } from '../../../components/SystemPageHeader';
import { SystemButton } from '../../../components/SystemButton';
import { LoadingUtil } from '../../../core/utils/LoadingUtil';
import { MediaUrlUtil } from '../../../core/utils/MediaUrlUtil';
import { ShareActionHelper } from '../../../core/utils/ShareActionHelper';
import { NfcShareService } from '../../../core/services/NfcShareService';
import { SystemUiController } from '../../../core/utils/SystemUiController';
import { KeyboardDismissManager } from '../../../core/utils/KeyboardDismissManager';
import { SafeAreaUtils } from '../../../core/utils/SafeAreaUtils';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { AppThemeVisuals } from '../../../core/theme/AppThemeVisuals';
import { ListGroup } from '../../../components/ListGroup';
import { ProfileMenuItem } from '../components/ProfileContentAtoms';
import { ProfileDataSection } from '../components/ProfileDataSection';
import { ProfileIdentityCard } from '../components/ProfileIdentityCard';
import { ProfileAvatarBlock, ProfileStatsRow } from '../components/ProfileHeaderAtoms';
import { ProfileRemarkSheet } from '../components/ProfileRemarkSheet';
import { ExternalProfileView } from '../components/ExternalProfileView';
import { DEFAULT_RESOLVED_THEME_ID, KEY_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';
import { getErrorMessage } from '../../../core/network/ApiErrorUtil';

const DOMAIN_NUMBER: number = 0xFF03;
const TAG: string = 'ProfilePage';
const PASSWORD_POLICY_HINT: string = '密码需为8-128位且同时包含字母和数字';

interface ProfilePageParams {
  userId?: string;
  user_id?: string;
}

@Preview
@Component
export struct ProfilePage {
  pathStack: NavPathStack = new NavPathStack();
  @StorageLink(KEY_CURRENT_USER) currentUser: AccountInfo | null = null;
  @StorageLink(KEY_IS_LOGGED_IN) isLoggedIn: boolean = false;
  @StorageLink(KEY_USER_TYPE) userType: string = '';
  @StorageLink(KEY_CURRENT_LOCATION) currentLocation: LocationInfo | null = null;
  @StorageLink('safeAreaTop') safeAreaTop: number = 0;
  @StorageLink('safeAreaBottom') safeAreaBottom: number = 0;
  @StorageLink(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = DEFAULT_RESOLVED_THEME_ID;
  @StorageProp('currentBreakpoint') currentBreakpoint: BreakpointType = 'sm';

  private stateManager = GlobalStateManager.getInstance();
  @State isEditing: boolean = false;
  @State editNickname: string = '';
  @State editBio: string = '';
  @State editAvatarPreviewUrl: string = '';
  @State editAvatarLocalUri: string = '';
  @State editAvatarFileId: string = '';
  @State isAvatarUploading: boolean = false;
  @State isLoading: boolean = false;
  @State myMomentsCount: number = 0;
  @State myGroupsCount: number = 0;
  @State myFriendsCount: number = 0;
  @State myFollowersCount: number = 0;
  @State myFollowingCount: number = 0;
  // 快速体验弹窗
  // 升级表单
  @State showUpgradeForm: boolean = false;
  @State upgradeLoginId: string = '';
  @State upgradePassword: string = '';
  @State upgradePasswordConfirm: string = '';
  @State isUpgrading: boolean = false;
  @State hasAdminPrivileges: boolean = false;
  @State viewedUserId: string = '';
  @State viewedUserProfile: UserItem | null = null;
  // 仅持被查看用户的关系态（单实体定向订阅，同 UserCard 范式）
  @State viewedUserRelation: UserRelationState | undefined = undefined;
  @State viewedFollowPending: boolean = false;
  @State viewedUserMoments: MomentItem[] = [];
  @State viewedUserLoading: boolean = false;
  @State viewedUserError: string = '';
  // 设置备注（对齐前端 users/[userId] 备注名称；仅已关注可保存）
  @State showRemarkSheet: boolean = false;
  @State remarkDraft: string = '';
  @State isSavingRemark: boolean = false;
  private isDestroyed: boolean = false;

  private webSocketService = WebSocketService.getInstance();

  private isValidLoginId(loginId: string): boolean {
    // spec user.md → upgrade：仅支持用户名型 login_id（手机号/邮箱需验证码服务，接入前拒绝）
    return isValidUsernameLoginId(loginId);
  }

  private isValidNickname(nickname: string): boolean {
    return isValidNicknameShared(nickname);
  }

  private isValidPassword(value: string): boolean {
    return isValidPasswordShared(value);
  }

  private isExternalProfileMode(): boolean {
    return this.viewedUserId.length > 0;
  }

  private resolveHeaderTitle(): string {
    if (this.isExternalProfileMode()) {
      const nickname = this.viewedUserProfile?.user?.nickname ?? '';
      return nickname.length > 0 ? nickname : '用户资料';
    }
    // 编辑态是就地切换的子视图，页头需随之切换语义，
    // 否则标题仍是「个人中心」+ 设置入口，与下方编辑表单相互矛盾
    if (this.isEditing) {
      return '编辑资料';
    }
    // 对齐前端 /me 顶栏：固定「个人中心」小标题（不随昵称变化）
    return '个人中心';
  }

  private async loadViewedUserProfile(): Promise<void> {
    if (!this.viewedUserId) {
      return;
    }
    this.viewedUserLoading = true;
    this.viewedUserError = '';
    this.viewedUserProfile = null;
    this.viewedUserMoments = [];
    try {
      const response = await LoadingUtil.withMinimumTime(() =>
        ApiClient.getUserProfile({ user_id: this.viewedUserId, include_recent_moments: true } as OpenUserDoorRequest),
      DesignConstants.ANIMATION.DURATION_CONTENT_SWITCH);
      if (this.isDestroyed) return;
      if (response.code === 0 && response.data?.user?.user?.user_id) {
        this.viewedUserProfile = response.data.user;
        this.viewedUserMoments = response.data.recent_moments?.items ?? [];
        InteractionStateStore.getInstance().patchUserFromItem(this.viewedUserProfile);
        // 播种：patch 已写入 store，读取快照恢复本页关系态
        const existing = interactionEventHub.getUserState(this.viewedUserProfile.user?.user_id ?? '');
        if (existing) {
          this.viewedUserRelation = existing;
        }
      } else {
        this.viewedUserError = getErrorMessage(response, '无法获取该用户资料');
      }
    } catch {
      this.viewedUserError = '加载用户资料失败';
    } finally {
      this.viewedUserLoading = false;
    }
  }

  private isViewedUserFollowing(): boolean {
    const uid = this.viewedUserProfile?.user?.user_id ?? '';
    return (this.viewedUserRelation?.user_id === uid ? this.viewedUserRelation?.relation?.is_following : undefined) ?? false;
  }

  private async toggleViewedUserFollow(): Promise<void> {
    const uid = this.viewedUserProfile?.user?.user_id ?? '';
    if (!uid || this.viewedFollowPending) {
      return;
    }
    this.viewedFollowPending = true;
    try {
      const isFollowing = this.isViewedUserFollowing();
      const ok = await interactionActions.setFollowUser(uid, !isFollowing, isFollowing ? '已取消关注' : '已关注');
      if (!ok) {
        SystemToast.showError('操作失败', this.getUIContext());
      }
    } finally {
      this.viewedFollowPending = false;
    }
  }

  // ==================== 设置备注（对齐前端 users/[userId] set_remark） ====================

  private openRemarkSheet(): void {
    if (!this.ensureLoggedIn('设置备注')) {
      return;
    }
    // 对齐前端：仅已关注可设置备注（disabled={!followed}）
    if (!this.isViewedUserFollowing()) {
      SystemToast.showWarning('关注对方后可设备注', this.getUIContext());
      return;
    }
    this.remarkDraft = '';
    this.showRemarkSheet = true;
  }

  private async saveRemark(): Promise<void> {
    const uid = this.viewedUserProfile?.user?.user_id ?? '';
    if (!uid || !this.isViewedUserFollowing() || this.isSavingRemark) {
      return;
    }
    this.isSavingRemark = true;
    try {
      const response = await ApiClient.setFriendRemark({ user_id: uid, remark: this.remarkDraft.trim() });
      if (response.code === 0) {
        this.showRemarkSheet = false;
        SystemToast.showSuccess('备注已保存', this.getUIContext());
      } else {
        SystemToast.showError(getErrorMessage(response, '保存失败'), this.getUIContext());
      }
    } catch {
      SystemToast.showError('保存失败，请重试', this.getUIContext());
    } finally {
      this.isSavingRemark = false;
    }
  }

  private ensureLoggedIn(actionName: string): boolean {
    if (this.isLoggedIn) {
      return true;
    }
    const loginParams = new LoginRouteParam();
    loginParams.reason = 'action_required';
    loginParams.action = actionName;
    loginParams.redirectUrl = PageNames.PROFILE;
    if (actionName === '发起私聊') {
      const profile = this.viewedUserProfile;
      const userId = profile?.user?.user_id ?? '';
      if (userId) {
        loginParams.restoreAction = 'open_private_chat';
        loginParams.restoreTargetId = userId;
        loginParams.restoreTitle = profile?.user?.nickname ?? '私聊';
      }
    }
    AppRouter.push(PageNames.LOGIN, loginParams);
    return false;
  }

  private startPrivateChatWithViewedUser(): void {
    const profile = this.viewedUserProfile;
    const userId = profile?.user?.user_id ?? '';
    if (!userId) {
      return;
    }
    if (!this.ensureLoggedIn('发起私聊')) {
      return;
    }
    const params = new ChatRouteParam();
    params.chatId = userId;
    params.chatType = 'private_chat';
    params.targetId = userId;
    params.title = profile?.user?.nickname ?? '私聊';
    AppRouter.push(PageNames.CHAT, params);
  }

  // 登录/登出/切号后刷新管理员行：常驻 Tab 不会重新 mount，
  // aboutToAppear 只在首次挂载跑一次，此处补上身份变更时的刷新。
  // 注意：必须经箭头函数持有 this 后再订阅——hub 以裸函数调用 handler，
  // 直接传 this.handleX 会丢绑定（实测 EventHub handler error），刷新永不执行。
  private authUserChangedListener: (payload: Object) => void = (payload: Object): void => {
    this.handleAuthUserChanged(payload);
  };

  private handleAuthUserChanged(payload: Object): void {
    const state = payload as AuthUserChangedState;
    if (!state) {
      return;
    }
    // 有意不等待：后台刷新 rbac/me，结果经 @State 自动刷 UI；
    // 方法内部全路径捕获异常，无未处理 rejection。
    this.refreshAdminPrivileges();
  }

  // 被查看用户的关系态定向更新（单实体，同 UserCard 范式）
  private handleViewedUserStateUpdate(payload: Object): void {
    const state = payload as UserRelationState;
    const uid = this.viewedUserProfile?.user?.user_id ?? '';
    if (state && uid && state.user_id === uid) {
      this.viewedUserRelation = state;
    }
  }

  async aboutToAppear() {
    // appear 即存活：常驻 Tab 被全屏页覆盖会触发 aboutToDisappear，
    // 返回时 aboutToAppear 重跑，若不复位，后续刷新（含事件触发）将永久早退。
    this.isDestroyed = false;
    SystemUiController.syncWithCurrentTheme();
    interactionEventHub.on('user_relation_updated', this.handleViewedUserStateUpdate);
    interactionEventHub.on('auth_user_changed', this.authUserChangedListener);
    if (this.currentUser) {
      this.editNickname = this.currentUser.nickname;
      this.editBio = this.currentUser.bio || '';
      this.editAvatarPreviewUrl = this.currentUser.avatar_url || '';
      try {
        NfcShareService.getInstance().setShareData(
          NfcShareService.buildUserProfileShare(
            this.currentUser.user_id,
            this.currentUser.nickname,
            this.currentUser.bio,
            this.currentUser.avatar_url
          )
        );
      } catch (_) {
        // NFC 不可用或失败时静默忽略
      }
    }
    await this.loadUserStats();
    await this.refreshAdminPrivileges();
  }

  aboutToDisappear(): void {
    this.isDestroyed = true;
    interactionEventHub.off('user_relation_updated', this.handleViewedUserStateUpdate);
    interactionEventHub.off('auth_user_changed', this.authUserChangedListener);
    KeyboardDismissManager.getInstance().dismiss();
    try {
      NfcShareService.getInstance().clearShareData();
    } catch (_) {
      // 静默忽略
    }
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

  private getAccentBackground(): string {
    return this.getCurrentThemePalette().accentBackground;
  }

  private getAccentText(): string {
    return this.getCurrentThemePalette().accentText;
  }

  private getPrimaryTextColor(): string {
    return this.getCurrentThemePalette().textPrimary;
  }

  private getSecondaryTextColor(): string {
    return this.getCurrentThemePalette().textSecondary;
  }

  private getWarningTextColor(): string {
    return this.getCurrentThemePalette().warning;
  }

  private getProfileVisualStyle() {
    return AppThemeVisuals.getProfileVisualStyle(this.resolvedThemeId);
  }

  private async loadUserStats(): Promise<void> {
    if (!this.isLoggedIn || !this.currentUser) {
      return;
    }
    if (this.currentUser.stats) {
      this.myMomentsCount = this.currentUser.stats.moments_count || 0;
      this.myGroupsCount = this.currentUser.stats.groups_count || 0;
      this.myFriendsCount = this.currentUser.stats.friends_count || 0;
      this.myFollowersCount = this.currentUser.stats.followers_count || 0;
      this.myFollowingCount = this.currentUser.stats.following_count || 0;
    }
  }


  private openMyMoments(): void {
    AppRouter.push(PageNames.MOMENT);
  }

  private openMyGroups(): void {
    AppRouter.push(PageNames.GROUP);
  }

  private openRelationshipHub(): void {
    AppRouter.push(PageNames.RELATIONSHIP_HUB);
  }

  private isAdminUser(): boolean {
    return this.hasAdminPrivileges;
  }

  private async refreshAdminPrivileges(): Promise<void> {
    if (!this.isLoggedIn || !this.currentUser || this.userType === 'temp') {
      this.hasAdminPrivileges = false;
      return;
    }

    try {
      const response = await ApiClient.getMyAdminPermissions();
      if (this.isDestroyed) return;
      if (response.code === 0 && response.data) {
        this.hasAdminPrivileges = true;
        return;
      }
    } catch (_) {
      // 非管理员返回 10200，静默降级为普通用户视图
    }

    this.hasAdminPrivileges = false;
  }

  private openAdminPanel(): void {
    if (!this.isAdminUser()) {
      SystemToast.showWarning('仅管理员可访问', this.getUIContext());
      return;
    }
    // 管理员工具台落点为管理后台概览页（此前误指向调试面板，功能名与落点不符）；
    // 调试面板保留在高级设置的 DEBUG_BUILD 门控入口内。
    AppRouter.push(PageNames.ADMIN_OVERVIEW);
  }

  private startEdit(): void {
    if (this.currentUser) {
      this.editNickname = this.currentUser.nickname;
      this.editBio = this.currentUser.bio || '';
      this.editAvatarPreviewUrl = this.currentUser.avatar_url || '';
      this.editAvatarLocalUri = '';
      this.editAvatarFileId = '';
      this.isEditing = true;
    }
  }

  private cancelEdit(): void {
    this.editAvatarLocalUri = '';
    this.editAvatarFileId = '';
    this.editAvatarPreviewUrl = this.currentUser?.avatar_url || '';
    this.isEditing = false;
  }

  private async selectAvatarForEdit(): Promise<void> {
    if (this.isLoading || this.isAvatarUploading) {
      return;
    }
    try {
      const context = this.getUIContext().getHostContext() as common.UIAbilityContext;
      const selectedUris = await MediaServiceManager.selectImages(context, 1);
      if (selectedUris.length === 0) {
        return;
      }
      this.editAvatarLocalUri = selectedUris[0];
      this.editAvatarPreviewUrl = selectedUris[0];
      this.editAvatarFileId = '';
      SystemToast.show('头像已选择，保存后生效', this.getUIContext());
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `选择头像失败: ${JSON.stringify(error)}`);
      SystemToast.showError('选择头像失败，请重试', this.getUIContext());
    }
  }

  /**
   * 身份卡/头像相机按钮（非编辑态）：选图后直接上传并更新头像。
   * 此前复用 selectAvatarForEdit，仅写入编辑态暂存字段，
   * 未进入编辑态时用户没有保存入口，头像变更会丢失。
   */
  private async changeAvatarDirectly(): Promise<void> {
    if (this.isLoading || this.isAvatarUploading) {
      return;
    }
    try {
      const context = this.getUIContext().getHostContext() as common.UIAbilityContext;
      const selectedUris = await MediaServiceManager.selectImages(context, 1);
      if (selectedUris.length === 0) {
        return;
      }
      this.isAvatarUploading = true;
      const compressedPath = await MediaServiceManager.compressImage(context, selectedUris[0], 80, 1024 * 1024 * 2);
      const uploadUri = await MediaServiceManager.prepareFileForUpload(context, compressedPath);
      const uploadResult = await ApiClient.uploadFile(context, uploadUri, 'avatar');
      if (!uploadResult.file_id) {
        throw new Error('头像上传失败');
      }
      const updateResponse = await ApiClient.updateProfile({ avatar_file_id: uploadResult.file_id });
      if (updateResponse.code !== 0) {
        throw new Error(getErrorMessage(updateResponse, '头像更新失败'));
      }
      const accountResponse = await ApiClient.getAccountInfo();
      if (accountResponse.code === 0 && accountResponse.data) {
        this.stateManager.updateUser(accountResponse.data);
      }
      SystemToast.showSuccess('头像已更新', this.getUIContext());
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `更换头像失败: ${JSON.stringify(error)}`);
      const message = error instanceof Error ? error.message : '更换头像失败，请重试';
      SystemToast.showError(message, this.getUIContext());
    } finally {
      this.isAvatarUploading = false;
    }
  }

  private async ensureEditAvatarUploaded(context: common.UIAbilityContext): Promise<string | undefined> {
    if (!this.editAvatarLocalUri) {
      return undefined;
    }
    if (this.editAvatarFileId) {
      return this.editAvatarFileId;
    }
    this.isAvatarUploading = true;
    try {
      const compressedPath = await MediaServiceManager.compressImage(context, this.editAvatarLocalUri, 80, 1024 * 1024 * 2);
      const uploadUri = await MediaServiceManager.prepareFileForUpload(context, compressedPath);
      const uploadResult = await ApiClient.uploadFile(context, uploadUri, 'avatar');
      if (!uploadResult.file_id) {
        throw new Error('头像上传失败');
      }
      this.editAvatarFileId = uploadResult.file_id;
      return this.editAvatarFileId;
    } finally {
      this.isAvatarUploading = false;
    }
  }

  private async saveEdit(): Promise<void> {
    const trimmedNickname = this.editNickname.trim();
    const trimmedBio = this.editBio.trim();
    if (!trimmedNickname) {
      SystemToast.showWarning('昵称不能为空', this.getUIContext());
      return;
    }
    if (!this.isValidNickname(trimmedNickname)) {
      SystemToast.showWarning('昵称长度需在1-30字符之间', this.getUIContext());
      return;
    }
    if (trimmedBio.length > 500) {
      SystemToast.showWarning('个人简介不能超过500字符', this.getUIContext());
      return;
    }

    this.isLoading = true;
    try {
      await LoadingUtil.withMinimumTime(async () => {
        if (this.currentUser) {
          const context = this.getUIContext().getHostContext() as common.UIAbilityContext;
          const avatarFileId = await this.ensureEditAvatarUploaded(context);
          const updateProfileResponse = await ApiClient.updateProfile({
            nickname: trimmedNickname,
            bio: trimmedBio,
            avatar_file_id: avatarFileId
          });
          if (updateProfileResponse.code !== 0) {
            throw new Error(getErrorMessage(updateProfileResponse, '资料更新失败'));
          }

          const response = await ApiClient.getAccountInfo();
          if (response.code === 0 && response.data) {
            this.stateManager.updateUser(response.data);
          }
        }

        this.editAvatarLocalUri = '';
        this.editAvatarFileId = '';
        this.isEditing = false;
        SystemToast.showSuccess('资料更新成功', this.getUIContext());
      }, LoadingUtil.DEFAULT_ACTION_MS);
    } catch (error) {
      const message = error instanceof Error ? error.message : '更新失败';
      SystemToast.showError(message, this.getUIContext());
    } finally {
      this.isLoading = false;
    }
  }

  private async updateLocation(): Promise<void> {
    if (!this.currentLocation) {
      SystemToast.show('当前位置不可用', this.getUIContext());
      return;
    }

    try {
      const request: UpdateLocationRequest = {
        location: {
          point: {
            latitude: this.currentLocation.latitude,
            longitude: this.currentLocation.longitude
          },
          location_name: this.currentLocation.locationName || '未知位置'
        }
      };
      await ApiClient.updateLocation(request);
      SystemToast.showSuccess('位置更新成功', this.getUIContext());
    } catch (error) {
      SystemToast.showError('位置更新失败', this.getUIContext());
    }
  }

  private async logout(): Promise<void> {
    try {
      // 1. 先尝试通知服务端（失败不阻塞本地清理）
      try {
        await ApiClient.logout();
      } catch (e) {
        hilog.warn(DOMAIN_NUMBER, TAG, `服务端登出失败: ${JSON.stringify(e)}`);
      }

      // 2. 断开 WebSocket
      this.webSocketService.disconnect();

      // 3. 清理本地认证数据：只清 token/会话账号，不动 device_id、服务器地址、
      //    主题等用户配置（spec：device_id 登出不清除；PreferencesUtil.clear()
      //    全量清空会连带抹掉它们，导致幂等 client_id 漂移、自定义服务器丢失）
      await HttpUtil.clearAuthData();

      // 4. 重置 AppStorage（PersistentStorage 会自动同步到磁盘）
      this.stateManager.clear();
      // 同步清零管理员行：clear() 经 auth_user_changed 事件触发的刷新是异步的，
      // 先行清零避免登出瞬间残留上一账号的管理员入口。
      this.hasAdminPrivileges = false;

      // 5. 跳转到首页
      AppRouter.clear();
      SystemToast.show('已退出登录', this.getUIContext());
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `退出登录异常: ${JSON.stringify(error)}`);
      // 即使出现异常，也强制清理状态，避免用户卡在"已登录但无法操作"的死胡同
      try {
        this.stateManager.clear();
        await HttpUtil.clearAuthData();
      } catch (_) {
        // ignore
      }
      SystemToast.show('退出登录遇到问题，请重启应用', this.getUIContext());
    }
  }

  private async upgradeAccount(): Promise<void> {
    const trimmedLoginId = this.upgradeLoginId.trim();
    if (!trimmedLoginId) {
      SystemToast.showWarning('请输入用户名（手机号/邮箱升级需验证码，暂未开放）', this.getUIContext());
      return;
    }
    if (!this.isValidLoginId(trimmedLoginId)) {
      SystemToast.showWarning('登录账号长度需在3-64字符之间', this.getUIContext());
      return;
    }
    if (!this.isValidPassword(this.upgradePassword)) {
      SystemToast.showWarning(PASSWORD_POLICY_HINT, this.getUIContext());
      return;
    }
    if (this.upgradePassword !== this.upgradePasswordConfirm) {
      SystemToast.showWarning('两次输入的密码不一致', this.getUIContext());
      return;
    }

    this.isUpgrading = true;
    try {
      const request: UpgradeAccountRequest = {
        login_id: trimmedLoginId,
        password: this.upgradePassword
      };
      const response = await ApiClient.upgradeAccount(request);
      if (response.code === 0) {
        // 升级成功，刷新用户信息
        const profileResp = await ApiClient.getAccountInfo();
        if (profileResp.code === 0 && profileResp.data) {
          this.stateManager.updateUser(profileResp.data);
        }
        this.showUpgradeForm = false;
        this.upgradeLoginId = '';
        this.upgradePassword = '';
        this.upgradePasswordConfirm = '';
        SystemToast.showSuccess('账号升级成功！', this.getUIContext());
      } else {
        SystemToast.showError(getErrorMessage(response, '升级失败'), this.getUIContext());
      }
    } catch (error) {
      SystemToast.showError(error instanceof Error ? error.message : '升级失败', this.getUIContext());
    } finally {
      this.isUpgrading = false;
    }
  }

  // 端侧补充分组（前端 ProfileMobile 无对应区块）：位置上报 / 临时账号升级 / 管理员工具台。
  // 前端位置由浏览器地理定位自动上报、无临时账号形态、亦无移动端管理台入口，故三项属端侧独有；
  // 2026-10-17 移动端对齐：从「个人数据」卡拆出单列一组（保持个人数据卡 = 前端 5 行契约），
  // 设置入口上移身份卡操作行、退出登录移至设置页（P0-4：全端单入口，本组不再承载导航与登出）。
  @Builder
  private buildAppOnlyActionRows() {
    ListGroup() {
      if (this.userType === 'temp') {
        ProfileMenuItem({ title: '升级为完整账号', icon: AppIcon.EDIT, divider: true, onTap: () => { this.showUpgradeForm = true; }, resolvedThemeId: this.resolvedThemeId })
      }
      ProfileMenuItem({ title: '更新位置', icon: AppIcon.LOCATION, divider: this.isAdminUser(), onTap: () => this.updateLocation(), resolvedThemeId: this.resolvedThemeId })
      if (this.isAdminUser()) {
        ProfileMenuItem({ title: '管理员工具台', icon: AppIcon.SHIELD, onTap: () => this.openAdminPanel(), resolvedThemeId: this.resolvedThemeId })
      }
    }
  }

  build() {
    NavDestination() {
      PageShellContainer({ variant: 'default', currentBreakpoint: this.currentBreakpoint }) {
        Stack({ alignContent: Alignment.Top }) {
          Column() {
          if (!this.isLoggedIn && !this.isExternalProfileMode() && !this.isLoading) {
            // 未登录：全页面展示 LoginPromptView（与前端 protected route 对齐）
            this.buildGuestView()
          } else {
            // 编辑态与常态用 if 分支重建 SystemPageHeader：其 onBack/rightAction
            // 是普通成员（非 @Prop），组件复用时不会被父级刷新，必须整颗重建
            if (this.isEditing) {
              SystemPageHeader({
                title: '编辑资料',
                titleSize: DesignConstants.FONT.SIZE.XXL,
                titleWeight: DesignConstants.FONT.WEIGHT.MEDIUM,
                showBackButton: true,
                onBack: () => { this.cancelEdit(); }
              })
            } else if (this.isExternalProfileMode()) {
              SystemPageHeader({
                title: this.resolveHeaderTitle(),
                titleSize: DesignConstants.FONT.SIZE.XXL,
                titleWeight: DesignConstants.FONT.WEIGHT.MEDIUM,
                showBackButton: true,
                onBack: () => { AppRouter.pop(); }
              })
            } else if (this.showUpgradeForm) {
              // 升级态：与编辑态同规格的页头（返回 = 取消升级），
              // 否则此聚焦任务页没有返回钮、且顶部空洞
              SystemPageHeader({
                title: '升级为完整账号',
                titleSize: DesignConstants.FONT.SIZE.XXL,
                titleWeight: DesignConstants.FONT.WEIGHT.MEDIUM,
                showBackButton: true,
                onBack: () => { this.cancelUpgrade(); }
              })
            } else {
              // 对齐前端 ProfileMobile 页头条：eyebrow 小标 + 底部发丝分隔线（bg-surface-page）。
              // 横向留白由 PageShellContainer gutter 提供，页头不再额外缩进 —— 原 left/right SM2(6)
              // 会让「个人中心」比正文多缩进 6，违反页头与内容对齐；下内边距对齐前端 py-3(12)。
              // 设置入口上移身份卡操作行（Web 同款契约：同页仅一处设置入口）。
              Row() {
                Text('个人中心')
                  .fontSize(DesignConstants.FONT.SIZE.XS2)
                  .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS2))
                  .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
                  .fontColor(this.getCurrentThemePalette().textSecondary)
                  .letterSpacing(DesignConstants.FONT.LETTER_SPACING.LABEL)
              }
              .width('100%')
              .alignItems(VerticalAlign.Center)
              .padding({
                // 与 SystemPageHeader 同口径：贴状态栏下缘（extra 0）
                top: SafeAreaUtils.resolveTopPadding(this.safeAreaTop, 0),
                bottom: DesignConstants.SPACING.MD
              })
              .border({
                width: { bottom: DesignConstants.BORDER_WIDTH.HAIRLINE },
                color: { bottom: this.getCurrentThemePalette().borderColor }
              })
            }

            if (this.isExternalProfileMode()) {
              Column() {
                this.buildExternalProfileView()
              }
              .width('100%')
              .justifyContent(FlexAlign.Start)
              .alignItems(HorizontalAlign.Start)
              .layoutWeight(1)
              .transition(
                TransitionEffect.OPACITY.animation({ duration: DesignConstants.ANIMATION.DURATION_SHEET, curve: DesignConstants.ANIMATION.CURVE_FAST_OUT_SLOW_IN })
                  .combine(TransitionEffect.move(TransitionEdge.BOTTOM).animation({ duration: DesignConstants.ANIMATION.DURATION_SHEET, curve: DesignConstants.ANIMATION.CURVE_FAST_OUT_SLOW_IN }))
              )
            } else if (this.isLoading) {
              SystemLoadingView({ message: '正在更新…' })
                .layoutWeight(1)
                .transition(
                  TransitionEffect.OPACITY.animation({ duration: DesignConstants.ANIMATION.DURATION_SHEET, curve: DesignConstants.ANIMATION.CURVE_FAST_OUT_SLOW_IN })
                    .combine(TransitionEffect.move(TransitionEdge.BOTTOM).animation({ duration: DesignConstants.ANIMATION.DURATION_SHEET, curve: DesignConstants.ANIMATION.CURVE_FAST_OUT_SLOW_IN }))
                )
            } else if (this.showUpgradeForm) {
              Column() {
                this.buildUpgradeFormView()
              }
              .layoutWeight(1)
              .transition(
                TransitionEffect.OPACITY.animation({ duration: DesignConstants.ANIMATION.DURATION_SHEET, curve: DesignConstants.ANIMATION.CURVE_FAST_OUT_SLOW_IN })
                  .combine(TransitionEffect.move(TransitionEdge.BOTTOM).animation({ duration: DesignConstants.ANIMATION.DURATION_SHEET, curve: DesignConstants.ANIMATION.CURVE_FAST_OUT_SLOW_IN }))
              )
            } else {
              // 对齐前端：游客(temp)与注册用户共用同一套 ProfileMobile 形态，
              // 不再单独渲染「临时用户视图」（前端 /me 无此分叉）
              Column() {
                this.buildRegisteredUserView()
              }
              .layoutWeight(1)
              .transition(
                TransitionEffect.OPACITY.animation({ duration: DesignConstants.ANIMATION.DURATION_SHEET, curve: DesignConstants.ANIMATION.CURVE_FAST_OUT_SLOW_IN })
                  .combine(TransitionEffect.move(TransitionEdge.BOTTOM).animation({ duration: DesignConstants.ANIMATION.DURATION_SHEET, curve: DesignConstants.ANIMATION.CURVE_FAST_OUT_SLOW_IN }))
              )
            }
          }
        }
        .width('100%')
        .height('100%')
        .constraintSize({ maxWidth: useMasterDetail(this.currentBreakpoint) ? 720 : '100%' })

        // 退出登录已按 P0-4 收敛到设置页（全端单入口），本页不再承载确认弹窗
      }
      .width('100%')
      .height('100%')
      .backgroundColor(this.getCurrentThemePalette().pageBackground)
      .bindSheet($$this.showRemarkSheet, this.buildRemarkSheet(), {
        height: SheetSize.FIT_CONTENT,
        showClose: true,
        dragBar: true,
        backgroundColor: this.getCurrentThemePalette().sheetBackground
      })
      }
    }
    .hideTitleBar(true)
    .onReady((context: NavDestinationContext) => {
      this.pathStack = context.pathStack;
      const params = context.pathInfo?.param as ProfilePageParams | undefined;
      const routeUserId = params?.userId ?? params?.user_id ?? '';
      const selfUserId = this.currentUser?.user_id ?? '';
      if (routeUserId && routeUserId !== selfUserId) {
        this.viewedUserId = routeUserId;
        this.loadViewedUserProfile();
      } else {
        this.viewedUserId = '';
        this.viewedUserProfile = null;
        this.viewedUserLoading = false;
        this.viewedUserError = '';
      }
    })
  }

  /** 他人资料视图：渲染在 ExternalProfileView 组件（资料/统计/回调经 props 注入） */
  @Builder
  private buildExternalProfileView(): void {
    ExternalProfileView({
      resolvedThemeId: this.resolvedThemeId,
      loading: this.viewedUserLoading,
      errorMessage: this.viewedUserError,
      hasProfile: this.viewedUserProfile !== null,
      avatarUrl: this.viewedUserProfile?.user?.avatar_url ?? '',
      nickname: this.viewedUserProfile?.user?.nickname ?? '',
      locationName: this.viewedUserProfile?.location?.location_name ?? '',
      bio: this.viewedUserProfile?.bio ?? '',
      isOnline: this.viewedUserProfile?.presence_status === 'online',
      tags: this.viewedUserProfile?.tags ?? [],
      momentsCount: `${this.viewedUserProfile?.stats?.moments_count ?? 0}`,
      followersCount: `${this.viewedUserProfile?.stats?.followers_count ?? 0}`,
      followingCount: `${this.viewedUserProfile?.stats?.following_count ?? 0}`,
      moments: this.viewedUserMoments,
      isFollowing: this.isViewedUserFollowing(),
      followPending: this.viewedFollowPending,
      safeAreaBottom: this.safeAreaBottom,
      onRetry: () => { this.loadViewedUserProfile(); },
      onPrivateChat: () => { this.startPrivateChatWithViewedUser(); },
      onToggleFollow: () => { this.toggleViewedUserFollow(); },
      onOpenRemark: () => { this.openRemarkSheet(); }
    })
  }

  // ==================== 备注编辑半模态（对齐前端备注名称输入，仅已关注可保存） ====================
  /** 备注半模态：渲染在 ProfileRemarkSheet 组件（草稿/保存态/回调经 props 注入） */
  @Builder
  private buildRemarkSheet() {
    ProfileRemarkSheet({
      resolvedThemeId: this.resolvedThemeId,
      nickname: this.viewedUserProfile?.user?.nickname ?? '',
      draft: this.remarkDraft,
      isSaving: this.isSavingRemark,
      canSave: this.isViewedUserFollowing(),
      onDraftChange: (value: string) => { this.remarkDraft = value; },
      onCancel: () => { this.showRemarkSheet = false; },
      onSave: () => { this.saveRemark(); }
    })
  }

  // ==================== 匿名用户 - 游客界面 ====================
  @Builder
  private buildGuestView() {
    LoginPromptView({
      title: '个人中心需要登录',
      subtitle: '登录后可管理资料、查看互动数据与收藏内容',
      icon: AppIcon.PERSON,
      loginAction: '查看个人中心'
    })
  }

  private showCreateSheet(): void {
    // 委托父级 Index 处理创建动作（通过 AppStorage 通信）
    AppStorage.setOrCreate('triggerCreateAction', Date.now());
  }


  // ==================== 临时用户界面 ====================
  // ==================== 注册用户界面 ====================
  @Builder
  private buildRegisteredUserView() {
    Column() {
      if (this.isEditing) {
        Scroll() {
          this.EditFormContent()
        }
        .layoutWeight(1)
        .scrollBar(BarState.Off)
        // 同升级表单：内容短于视口时 Scroll 默认居中，表单一律顶部对齐
        .align(Alignment.Top)
      } else {
        Scroll() {
          Column() {
            if (useMasterDetail(this.currentBreakpoint)) {
              // ======== 宽端（≥960）：保留垂直居中形态 ========
              ProfileAvatarBlock({
                resolvedThemeId: this.resolvedThemeId,
                avatarUrl: this.currentUser?.avatar_url || '',
                nickname: this.currentUser?.nickname || '',
                onChangeAvatar: () => { this.changeAvatarDirectly(); }
              })

              if (this.currentUser?.bio) {
                Text(this.currentUser.bio)
                  .fontSize(DesignConstants.FONT.SIZE.SM)
                  .fontColor(this.getSecondaryTextColor())
                  .textAlign(TextAlign.Center)
                  .lineHeight(DesignConstants.FONT.LINE_HEIGHT.SM)
                  .margin({ top: DesignConstants.SPACING.XS })
              }

              if (this.currentUser?.location?.location_name) {
                Row({ space: DesignConstants.SPACING.XS }) {
                  LucideIcon({
                    icon: AppIcon.LOCATION,
                    iconSize: DesignConstants.COMPONENT.ICON_SIZE_2XS,
                    color: this.getAccentBackground()
                  })
                  Text(this.currentUser.location.location_name)
                    .fontSize(DesignConstants.FONT.SIZE.XS)
                    .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS))
                    .maxLines(1)
                    .textOverflow({ overflow: TextOverflow.Ellipsis })
                    .fontColor(this.getSecondaryTextColor())
                }
                .margin({ top: DesignConstants.SPACING.SM })
              }

              Button() {
                Row({ space: DesignConstants.SPACING.SM2 }) {
                  LucideIcon({
                    icon: AppIcon.EDIT,
                    iconSize: DesignConstants.COMPONENT.ICON_SIZE_2XS,
                    color: this.getSecondaryTextColor()
                  })
                  Text('编辑资料')
                    .fontSize(DesignConstants.FONT.SIZE.SM)
                    .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.SM))
                    .fontColor(this.getSecondaryTextColor())
                }
              }
              .type(ButtonType.Normal)
              .width('100%')
              .height(DesignConstants.COMPONENT.SHEET_ACTION_HEIGHT)
              // .border 先于 .borderRadius：ArkUI 后设属性覆盖先设，border 写在后面会重置圆角
              .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getSurfaceBorder() })
              .borderRadius(DesignConstants.SHAPE.BUTTON)
              .backgroundColor(this.getSurfaceBackground())
              .shadow(this.getCurrentThemePalette().controlShadow)
              .margin({ top: DesignConstants.SPACING.LG, bottom: DesignConstants.SPACING.LG2 })
              .onClick(() => this.startEdit())

              ProfileStatsRow({
                resolvedThemeId: this.resolvedThemeId,
                momentsCount: this.myMomentsCount,
                followersCount: this.myFollowersCount,
                groupsCount: this.myGroupsCount
              })
            } else {
              // ======== 移动/平板：前端 ProfileMobile 形态（身份卡 + 个人数据列表） ========
              ProfileIdentityCard({
                resolvedThemeId: this.resolvedThemeId,
                avatarUrl: this.currentUser?.avatar_url || '',
                nickname: this.currentUser?.nickname || '',
                locationName: this.currentUser?.location?.location_name || '',
                bio: this.currentUser?.bio || '',
                momentsCount: this.myMomentsCount,
                followersCount: this.myFollowersCount,
                groupsCount: this.myGroupsCount,
                onEditProfile: () => { this.startEdit(); },
                onOpenSettings: () => { AppRouter.push(PageNames.SETTINGS); }
              })
              // 个人数据：只保留前端 ProfileMobile 的 5 行（我的瞬间/我的群组/好友/关注/粉丝）
              ProfileDataSection({
                resolvedThemeId: this.resolvedThemeId,
                myMomentsCount: this.myMomentsCount,
                myGroupsCount: this.myGroupsCount,
                myFriendsCount: this.myFriendsCount,
                myFollowingCount: this.myFollowingCount,
                myFollowersCount: this.myFollowersCount
              })
              // 端侧补充分组：见 buildAppOnlyActionRows 注释（前端无对应区块）
              Column() {
                Text('账号与位置')
                  .fontSize(DesignConstants.FONT.SIZE.XS2)
                  .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS2))
                  .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
                  .fontColor(this.getCurrentThemePalette().textSecondary)
                  .letterSpacing(DesignConstants.FONT.LETTER_SPACING.LABEL)
                  .textCase(TextCase.UpperCase)
                  .width('100%')
                  .padding({ left: DesignConstants.SPACING.XS, bottom: DesignConstants.SPACING.SM })
                Text('端侧独有：同步当前位置、账号升级与管理入口')
                  .fontSize(DesignConstants.FONT.SIZE.XS)
                  .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS))
                  .fontColor(this.getCurrentThemePalette().textSecondary)
                  .width('100%')
                  .padding({ left: DesignConstants.SPACING.XS, bottom: DesignConstants.SPACING.SM })
                this.buildAppOnlyActionRows()
              }
              .width('100%')
              .alignItems(HorizontalAlign.Start)
              .margin({ top: DesignConstants.PAGE_SHELL.SECTION_GAP })
            }

            // 2026-10-17 移动端对齐：移除「帖子」Tab + 我的瞬间画廊。
            // 前端 ProfileMobile 只有「身份卡 + 个人数据列表」两卡（无 Tab/画廊），
            // 端侧画廊是 app 独有形态；且「喜欢/收藏」无数据源（spec UserProfileBundle
            // 仅 recent_moments），单 tab 悬浮在数据卡下方视觉上像错位元素。

          }
          .alignItems(HorizontalAlign.Center)
          // 横向留白统一由容器 gutter 提供，这里只保留纵向节奏
          // （顶部 PAGE_TOP_GAP(16) 对齐前端 pt-4：卡片顶边距页头分隔线 16）
          .padding({ top: DesignConstants.PAGE_SHELL.PAGE_TOP_GAP, bottom: DesignConstants.SPACING.XL })
        }
        .layoutWeight(1)
        .scrollBar(BarState.Off)
      }
    }
    .width('100%')
    .height('100%')
    .backgroundColor(this.getCurrentThemePalette().pageBackground)
  }




  // ==================== 升级表单界面 ====================
  /** 取消升级：关表单并清空输入（页头返回与「取消」按钮共用） */
  private cancelUpgrade(): void {
    this.showUpgradeForm = false;
    this.upgradeLoginId = '';
    this.upgradePassword = '';
    this.upgradePasswordConfirm = '';
  }

  @Builder
  private buildUpgradeFormView() {
    Scroll() {
      Column({ space: DesignConstants.SPACING.LG }) {
        Text('设置用户名与密码后，临时账号将升级为完整账号，所有数据都会保留。')
          .fontSize(DesignConstants.FONT.SIZE.MD)
          .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.MD))
          .fontColor(this.getSecondaryTextColor())

        SystemInput({
          value: this.upgradeLoginId,
          // spec user.md → upgrade：仅用户名型 login_id（手机/邮箱需验证码服务），
          // placeholder 直接言明，避免用户填完密码才被拒
          placeholder: '用户名（暂不支持手机号 / 邮箱）',
          inputType: 'email',
          fontSize: DesignConstants.FONT.SIZE.LG,
          onChange: (value: string) => { this.upgradeLoginId = value; },
          onFocusChange: (focused: boolean) => {
            if (!focused) KeyboardDismissManager.getInstance().dismiss();
          }
        })

        SystemInput({
          value: this.upgradePassword,
          placeholder: '密码（8-128位，含字母+数字）',
          passwordMode: true,
          fontSize: DesignConstants.FONT.SIZE.LG,
          onChange: (value: string) => { this.upgradePassword = value; },
          onFocusChange: (focused: boolean) => {
            if (!focused) KeyboardDismissManager.getInstance().dismiss();
          }
        })

        SystemInput({
          value: this.upgradePasswordConfirm,
          placeholder: '确认密码',
          passwordMode: true,
          fontSize: DesignConstants.FONT.SIZE.LG,
          onChange: (value: string) => { this.upgradePasswordConfirm = value; },
          onFocusChange: (focused: boolean) => {
            if (!focused) KeyboardDismissManager.getInstance().dismiss();
          }
        })

        Row({ space: DesignConstants.SPACING.MD }) {
          Button('取消')
            .type(ButtonType.Normal)
            .backgroundColor(this.getProfileVisualStyle().secondaryButton.backgroundColor)
            .fontColor(this.getProfileVisualStyle().secondaryButton.textColor)
            .layoutWeight(1)
            .height(this.getProfileVisualStyle().secondaryButton.height)
            // .border 先于 .borderRadius：border 写在后面会把圆角重置为直角（实拍确认）
            .border({ width: this.getProfileVisualStyle().secondaryButton.borderWidth, color: this.getProfileVisualStyle().secondaryButton.borderColor })
            .borderRadius(this.getProfileVisualStyle().secondaryButton.borderRadius)
            .shadow(this.getProfileVisualStyle().secondaryButton.shadow)
            .enabled(!this.isUpgrading)
            .opacity(this.isUpgrading ? DesignConstants.OPACITY.LOADING : 1)
            .onClick(() => {
              this.cancelUpgrade();
            })

          Button(this.isUpgrading ? '升级中…' : '确认升级')
            .type(ButtonType.Normal)
            .backgroundColor(this.getProfileVisualStyle().primaryButton.backgroundColor)
            .fontColor(this.getProfileVisualStyle().primaryButton.textColor)
            .layoutWeight(1)
            .height(this.getProfileVisualStyle().primaryButton.height)
            .borderRadius(this.getProfileVisualStyle().primaryButton.borderRadius)
            .shadow(this.getProfileVisualStyle().primaryButton.shadow)
            .enabled(!this.isUpgrading)
            .opacity(this.isUpgrading ? DesignConstants.OPACITY.LOADING : 1)
            .onClick(() => this.upgradeAccount())
        }
        .width('100%')
      }
      .width('100%')
      // 升级表单无卡片外框 → 横向留白由容器 gutter 提供（此前自带 XXL 32 会叠成 52，
      // 比页头多缩进一大截）；只保留纵向节奏
      .padding({ top: DesignConstants.SPACING.XL })
    }
    .layoutWeight(1)
    // 内容短于视口时 Scroll 默认垂直居中——表单会飘在页中段，
    // 页头下方留出大空隙；表单一律顶部对齐
    .align(Alignment.Top)
  }

  // ==================== 公共 Builder ====================

  @Builder
  private EditFormContent(): void {
    Column({ space: DesignConstants.SPACING.LG }) {
      Column({ space: DesignConstants.SPACING.SM }) {
        Button() {
          SystemAvatar({
            avatarUrl: this.editAvatarPreviewUrl || this.currentUser?.avatar_url || '',
            name: this.editNickname,
            avatarSize: DesignConstants.COMPONENT.AVATAR_SIZE.LARGE
          })
        }
        .type(ButtonType.Normal)
        .backgroundColor(DesignConstants.COLOR.TRANSPARENT)
        .enabled(!this.isLoading && !this.isAvatarUploading)
        .opacity(this.isLoading || this.isAvatarUploading ? DesignConstants.OPACITY.LOADING : 1)
        .onClick(() => {
          this.selectAvatarForEdit();
        })

        if (this.isAvatarUploading) {
          Text('头像上传中…')
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.SM))
            .fontColor(this.getSecondaryTextColor())
        } else {
          Text('点击头像更换')
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.SM))
            .fontColor(this.getSecondaryTextColor())
        }
      }

        SystemInput({
          value: this.editNickname,
          placeholder: '请输入昵称',
          inputHeight: DesignConstants.COMPONENT.BUTTON_HEIGHT_LG,
          fontSize: DesignConstants.FONT.SIZE.LG,
          onChange: (value: string) => { this.editNickname = value; },
          onFocusChange: (focused: boolean) => {
            if (!focused) KeyboardDismissManager.getInstance().dismiss();
          }
        })

      SystemTextArea({
        value: this.editBio,
        placeholder: '请输入个人简介',
        minHeight: DesignConstants.COMPONENT.AVATAR_SIZE.LARGE,
        fontSize: DesignConstants.FONT.SIZE.MD,
        onChange: (value: string) => {
          this.editBio = value;
        }
      })

      Row({ space: DesignConstants.SPACING.MD }) {
        Button('取消')
          .type(ButtonType.Normal)
          .backgroundColor(DesignConstants.COLOR.TRANSPARENT)
          .fontColor(this.getPrimaryTextColor())
          .layoutWeight(1)
          // .border 先于 .borderRadius（同升级表单取消，避免圆角被重置）
          .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getSurfaceBorder() })
          .borderRadius(this.getProfileVisualStyle().card.borderRadius)
          .enabled(!this.isLoading && !this.isAvatarUploading)
          .opacity(this.isLoading || this.isAvatarUploading ? DesignConstants.OPACITY.LOADING : 1)
          .onClick(() => this.cancelEdit())

        Button(this.isLoading ? '保存中…' : '保存')
          .type(ButtonType.Normal)
          .backgroundColor(this.getAccentBackground())
          .fontColor(this.getAccentText())
          .layoutWeight(1)
          .borderRadius(this.getProfileVisualStyle().card.borderRadius)
          .enabled(!this.isLoading && !this.isAvatarUploading)
          .opacity(this.isLoading || this.isAvatarUploading ? DesignConstants.OPACITY.LOADING : 1)
          .onClick(() => this.saveEdit())
      }
      .width('100%')
    }
    .width('100%')
  }

}

@Builder
export function ProfilePageBuilder() {
  ProfilePage()
}