/**
 * 个人中心页面
 * 三级用户体系：
 * - 匿名（无 token）：游客浏览模式，引导登录/注册
 * - 临时用户（temp）：已认证，显示升级入口
 * - 注册用户（normal）：完整功能
 */

import { GlobalStateManager, KEY_CURRENT_USER, KEY_IS_LOGGED_IN, KEY_USER_TYPE, KEY_CURRENT_LOCATION } from '../../../core/state/GlobalStateManager';
import { InteractionStateStore, UserRelationState } from '../../../core/state/InteractionStateStore';
import { interactionActions } from '../../../core/state/InteractionActions';
import { curves } from '@kit.ArkUI';
import { LocationInfo } from '../../../core/location/LocationUtil';
import { ApiClient } from '../../../core/api/ApiClient';
import { HttpUtil } from '../../../core/network/HttpUtil';
import { WebSocketService } from '../../../core/network/WebSocketService';
import { AppRouter, ChatRouteParam, LoginRouteParam, MomentDetailRouteParam } from '../../../core/router/AppRouter';
import { PageNames } from '../../../constants/AppConstants';
import { MediaServiceManager } from '../../../core/media/MediaServiceManager';
import { ViewedMomentThumb } from '../components/ViewedMomentThumb';
import { SystemAvatar } from '../../../components/SystemAvatar';
import { useMasterDetail } from '../../../core/responsive/Breakpoint';
import type { BreakpointType } from '../../../core/responsive/Breakpoint';
import { PageShellContainer } from '../../../components/PageShellContainer';
import { SystemToast } from '../../../components/SystemToast';
import { SystemInput } from '../../../components/SystemInput';
import { SystemTextArea } from '../../../components/SystemTextArea';
import { SystemLoadingView } from '../../../components/SystemLoadingView';
import { SystemDialog } from '../../../components/SystemDialog';
import { LoginPromptView } from '../../../components/LoginPromptView';
import { AccountInfo, UpdateLocationRequest, UpgradeAccountRequest, UserItem, OpenUserDoorRequest, UserSharePayload, MomentItem, FetchUserMomentsRequest } from '../../../core/types/api';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { common } from '@kit.AbilityKit';
import { DesignConstants } from '../../../constants/DesignConstants';
import { AppIcon } from '../../../components/SystemIcon';
import { LucideIcon } from '../../../components/LucideIcon';
import { EmptyState } from '../../../components/EmptyState';
import { SystemPageHeader } from '../../../components/SystemPageHeader';
import { SystemButton } from '../../../components/SystemButton';
import { TabsPillGroup } from '../../../components/TabsPillGroup';
import { LoadingUtil } from '../../../core/utils/LoadingUtil';
import { MediaUrlUtil } from '../../../core/utils/MediaUrlUtil';
import { ShareActionHelper } from '../../../core/utils/ShareActionHelper';
import { NfcShareService } from '../../../core/services/NfcShareService';
import { SystemUiController } from '../../../core/utils/SystemUiController';
import { KeyboardDismissManager } from '../../../core/utils/KeyboardDismissManager';
import { SafeAreaUtils } from '../../../core/utils/SafeAreaUtils';
import { formatCount } from '../../../core/utils/FormatUtil';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { AppThemeVisuals } from '../../../core/theme/AppThemeVisuals';
import { CompactStatItem, ProfileQuickAction, ProfileIntentCard, ProfileMenuItem } from '../components/ProfileContentAtoms';
import { ProfileMomentGallery } from '../components/ProfileMomentGallery';
import { KEY_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';
import { getErrorMessage } from '../../../core/network/ApiErrorUtil';

const DOMAIN_NUMBER: number = 0xFF03;
const TAG: string = 'ProfilePage';
const PASSWORD_POLICY_HINT: string = '密码需为8-128位且同时包含字母和数字';
// 备注长度上限（对齐前端 remarkDraft.slice(0, 24)）
const REMARK_MAX: number = 24;

interface ProfilePageParams {
  userId?: string;
  user_id?: string;
}

interface StatItem {
  label: string;
  value: string;
}

@Preview
@Component
export struct ProfilePage {
  pathStack: NavPathStack = new NavPathStack();
  @StorageLink(KEY_CURRENT_USER) currentUser: AccountInfo | null = null;
  @StorageLink(KEY_IS_LOGGED_IN) isLoggedIn: boolean = false;
  @StorageLink(KEY_USER_TYPE) userType: string = '';
  @StorageLink(KEY_CURRENT_LOCATION) currentLocation: LocationInfo | null = null;
  @StorageLink('safeAreaBottom') safeAreaBottom: number = 0;
  @StorageLink(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'vibrant_light';
  @StorageProp('currentBreakpoint') currentBreakpoint: BreakpointType = 'sm';

  private stateManager = GlobalStateManager.getInstance();
  @State isEditing: boolean = false;
  @State editNickname: string = '';
  @State editBio: string = '';
  @State editAvatarPreviewUrl: string = '';
  @State editAvatarLocalUri: string = '';
  @State editAvatarFileId: string = '';
  @State isAvatarUploading: boolean = false;
  @State showLogoutConfirm: boolean = false;
  @State isLoading: boolean = false;
  @State myMomentsCount: number = 0;
  @State myGroupsCount: number = 0;
  @State myFriendsCount: number = 0;
  @State myMoments: MomentItem[] = [];
  @State myFollowersCount: number = 0;
  @State myFollowingCount: number = 0;
  @State selectedTab: number = 0;
  /** 数据行按压/hover（对齐 ProfileMenuItem 按压契约；原行点击零反馈） */
  @State private profileRowPressed: string = '';
  @State private profileRowHovered: string = '';
  // 快速体验弹窗
  // 升级表单
  @State showUpgradeForm: boolean = false;
  @State upgradeLoginId: string = '';
  @State upgradePassword: string = '';
  @State upgradePasswordConfirm: string = '';
  @State isUpgrading: boolean = false;
  @State hasAdminPrivileges: boolean = false;
  @State pressedThumbKey: string = '';
  @State viewedUserId: string = '';
  @State viewedUserProfile: UserItem | null = null;
  @StorageLink('interaction_user_state') viewedUserStateMap: Record<string, UserRelationState> = {};
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
    const trimmed = loginId.trim();
    // spec user.md → upgrade：仅支持用户名型 login_id（手机号/邮箱需验证码服务，接入前拒绝）
    if (trimmed.includes('@') || /^\d+$/.test(trimmed)) {
      return false;
    }
    return trimmed.length >= 3 && trimmed.length <= 64;
  }

  private isValidNickname(nickname: string): boolean {
    const trimmed = nickname.trim();
    return trimmed.length >= 1 && trimmed.length <= 30;
  }

  private hasLetterAndDigit(value: string): boolean {
    let hasLetter: boolean = false;
    let hasDigit: boolean = false;
    for (let i = 0; i < value.length; i++) {
      const code = value.charCodeAt(i);
      if ((code >= 65 && code <= 90) || (code >= 97 && code <= 122)) {
        hasLetter = true;
      }
      if (code >= 48 && code <= 57) {
        hasDigit = true;
      }
      if (hasLetter && hasDigit) {
        return true;
      }
    }
    return false;
  }

  private isValidPassword(value: string): boolean {
    return value.length >= 8 && value.length <= 128 && this.hasLetterAndDigit(value);
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
    return this.viewedUserStateMap[uid]?.relation?.is_following ?? false;
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

  private async shareProfileByCode(): Promise<void> {
    const userId = this.viewedUserProfile?.user?.user_id;
    if (!userId) return;
    try {
      const context = this.getUIContext().getHostContext() as common.UIAbilityContext;
      const nickname = this.viewedUserProfile?.user?.nickname ?? '用户';
      const payload: UserSharePayload = { shared_user_id: userId };
      await ShareActionHelper.shareResource(
        context,
        'user',
        payload,
        `来认识 ${nickname}`,
        this.viewedUserProfile?.bio ?? '快来 HotSpot 和附近的人交流吧！'
      );
    } catch {
      // 用户取消分享
    }
  }

  async aboutToAppear() {
    SystemUiController.syncWithCurrentTheme();
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
    await this.loadMyMoments();
    await this.refreshAdminPrivileges();
  }

  aboutToDisappear(): void {
    this.isDestroyed = true;
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

  private async loadMyMoments(): Promise<void> {
    if (!this.isLoggedIn || !this.currentUser) {
      return;
    }
    try {
      const response = await ApiClient.getUserMoments({
        user_id: this.currentUser.user_id,
        pagination: { limit: 9 }
      });
      if (response.code === 0) {
        const items: MomentItem[] = response.data?.items ?? [];
        this.myMoments = items.filter((m: MomentItem) => {
          const imgs = m.images;
          return imgs !== undefined && imgs !== null && imgs.length > 0;
        });
      }
    } catch (error) {
      // 静默失败，画廊降级为空占位
    }
  }


  private openMyMoments(): void {
    AppRouter.push(PageNames.MOMENT);
  }

  private openMyGroups(): void {
    AppRouter.push(PageNames.GROUP);
  }

  private openSearchPeople(): void {
    AppRouter.push(PageNames.SEARCH);
  }

  private openSettingsCenter(): void {
    AppRouter.push(PageNames.SETTINGS);
  }

  private openPrivacySettings(): void {
    AppRouter.push(PageNames.PRIVACY_SETTINGS);
  }

  private openRelationshipHub(): void {
    AppRouter.push(PageNames.RELATIONSHIP_HUB);
  }

  private openSafetySettings(): void {
    AppRouter.push(PageNames.SAFETY_SETTINGS);
  }

  private openAnnouncements(): void {
    AppRouter.push(PageNames.ANNOUNCEMENTS);
  }

  private openMyShares(): void {
    AppRouter.push(PageNames.MY_SHARES);
  }

  private openSubscriptions(): void {
    AppRouter.push(PageNames.SUBSCRIPTIONS);
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
    AppRouter.push(PageNames.DEBUG_PANEL);
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

  build() {
    NavDestination() {
      PageShellContainer({ variant: 'default', currentBreakpoint: this.currentBreakpoint, ambientGlow: false }) {
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
            } else {
              SystemPageHeader({
                title: this.resolveHeaderTitle(),
                titleSize: this.isExternalProfileMode() ? DesignConstants.FONT.SIZE.XL : DesignConstants.FONT.SIZE.XXL,
                titleWeight: DesignConstants.FONT.WEIGHT.MEDIUM,
                showBackButton: this.isExternalProfileMode(),
                onBack: this.isExternalProfileMode() ? () => { AppRouter.pop(); } : undefined,
                rightAction: (!this.isExternalProfileMode() && this.isLoggedIn)
                  ? { icon: AppIcon.SETTINGS, onClick: () => { AppRouter.push(PageNames.SETTINGS); }, accessibilityLabel: '设置' }
                  : undefined
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
              SystemLoadingView({ message: '正在更新...' })
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

        // 退出登录二次确认（危险操作统一样式：danger 确认键）
        SystemDialog({
          visible: this.showLogoutConfirm,
          title: '退出登录',
          description: '退出后需要重新登录',
          confirmText: '退出登录',
          confirmVariant: 'danger',
          onCancel: () => { this.showLogoutConfirm = false; },
          onConfirm: () => {
            this.showLogoutConfirm = false;
            this.logout();
          }
        })
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

  @Builder
  private buildExternalProfileView(): void {
    if (this.viewedUserLoading) {
      SystemLoadingView({ message: '正在加载用户资料...' })
        .layoutWeight(1)
    } else if (this.viewedUserError.length > 0 || !this.viewedUserProfile) {
      EmptyState({
        icon: AppIcon.WARNING,
        message: this.viewedUserError.length > 0 ? this.viewedUserError : '未找到该用户',
        actionLabel: '重试',
        onAction: () => { this.loadViewedUserProfile(); }
      })
      .layoutWeight(1)
    } else {
      // 对齐前端 users/[userId] 移动端：左对齐身份行 + bio + 标签 + 三段统计
      // + 近期瞬间九宫格；底部操作条 私信 + 关注
      Column() {
        Scroll() {
          Column({ space: DesignConstants.SPACING.MD }) {
            Row({ space: DesignConstants.SPACING.MD }) {
              Stack({ alignContent: Alignment.BottomEnd }) {
                SystemAvatar({
                  avatarUrl: this.viewedUserProfile?.user?.avatar_url ?? '',
                  name: this.viewedUserProfile?.user?.nickname ?? '',
                  avatarSize: DesignConstants.COMPONENT.AVATAR_SIZE.LARGE,
                  shape: 'circle',
                  isOnline: this.viewedUserProfile?.presence_status === 'online'
                })
              }
              .width(DesignConstants.COMPONENT.AVATAR_SIZE.LARGE)
              .height(DesignConstants.COMPONENT.AVATAR_SIZE.LARGE)

              Column({ space: DesignConstants.SPACING.XS }) {
                Text(this.viewedUserProfile?.user?.nickname ?? '未知用户')
                  .fontSize(DesignConstants.FONT.SIZE.XXL)
                  .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
                  .fontColor(this.getPrimaryTextColor())
                  .maxLines(1)
                  .textOverflow({ overflow: TextOverflow.Ellipsis })
                Row({ space: DesignConstants.SPACING.XS }) {
                  LucideIcon({
                      icon: AppIcon.LOCATION,
                      iconSize: DesignConstants.FONT.SIZE.SM,
                      color: this.getAccentBackground()
                    })
                  Text(this.viewedUserProfile?.location?.location_name ?? '未设置位置')
                    .fontSize(DesignConstants.FONT.SIZE.XS)
                    .fontColor(this.getSecondaryTextColor())
                    .maxLines(1)
                    .textOverflow({ overflow: TextOverflow.Ellipsis })
                }
                .alignItems(VerticalAlign.Center)
              }
              .alignItems(HorizontalAlign.Start)
              .layoutWeight(1)
            }
            .width('100%')
            .alignItems(VerticalAlign.Center)

            Text(this.viewedUserProfile?.bio ?? '这个人很神秘，暂时还没有留下简介')
              .fontSize(DesignConstants.FONT.SIZE.SM)
              .lineHeight(DesignConstants.FONT.LINE_HEIGHT.MD)
              .fontColor(this.getSecondaryTextColor())
              .width('100%')

            if ((this.viewedUserProfile?.tags?.length ?? 0) > 0) {
              Flex({ wrap: FlexWrap.Wrap }) {
                ForEach((this.viewedUserProfile?.tags ?? []).slice(0, 8), (tag: string) => {
                  Text(`#${tag}`)
                    .fontSize(DesignConstants.FONT.SIZE.XS)
                    .fontColor(this.getAccentBackground())
                    .padding({ left: DesignConstants.SPACING.MD, right: DesignConstants.SPACING.MD, top: DesignConstants.SPACING.XS, bottom: DesignConstants.SPACING.XS })
                    .margin({ right: DesignConstants.SPACING.SM, bottom: DesignConstants.SPACING.SM })
                    .backgroundColor(this.getCurrentThemePalette().accentMuted)
                    .borderRadius(DesignConstants.RADIUS.FULL)
                }, (tag: string) => tag)
              }
              .width('100%')
            }

            Row() {
              this.ViewedStatItem(`${this.viewedUserProfile?.stats?.moments_count ?? 0}`, '瞬间')
              this.ViewedStatItem(`${this.viewedUserProfile?.stats?.followers_count ?? 0}`, '粉丝')
              this.ViewedStatItem(`${this.viewedUserProfile?.stats?.following_count ?? 0}`, '关注')
            }
            .width('100%')
            .margin({ top: DesignConstants.SPACING.SM })

            // 近期瞬间九宫格
            if (this.viewedUserMoments.length > 0) {
              Column({ space: DesignConstants.SPACING.SM }) {
                Text('近期瞬间')
                  .fontSize(DesignConstants.FONT.SIZE.SM)
                  .fontColor(this.getSecondaryTextColor())

                Grid() {
                  ForEach(this.viewedUserMoments, (moment: MomentItem) => {
                    GridItem() {
                      ViewedMomentThumb({
                        moment: moment,
                        palette: this.getCurrentThemePalette()
                      })
                    }
                  }, (moment: MomentItem) => moment.moment_id)
                }
                .columnsTemplate('1fr 1fr 1fr')
                .columnsGap(DesignConstants.SPACING.SM)
                .rowsGap(DesignConstants.SPACING.SM)
                .width('100%')
              }
              .width('100%')
              .margin({ top: DesignConstants.SPACING.SM })
            }
          }
          .width('100%')
          .justifyContent(FlexAlign.Start)
          .padding({
            left: DesignConstants.SPACING.LG2,
            right: DesignConstants.SPACING.LG2,
            top: DesignConstants.SPACING.MD,
            bottom: DesignConstants.SPACING.LG
          })
          .alignItems(HorizontalAlign.Start)
        }
        .width('100%')
        .layoutWeight(1)
        .align(Alignment.TopStart)
        .scrollBar(BarState.Off)

        // 底部操作条（对齐前端 footer snippet：私信 + 关注 下沉拇指热区；
        // 备注入口对齐前端「更多操作」sheet 的备注名称，仅已关注可保存）
        Row({ space: DesignConstants.SPACING.MD }) {
          SystemButton({
            label: '私信',
            variant: 'outline',
            buttonSize: 'small',
            onPress: () => {
              this.startPrivateChatWithViewedUser();
            }
          })
            .layoutWeight(1)

          SystemButton({
            label: this.isViewedUserFollowing() ? '已关注' : '关注',
            variant: this.isViewedUserFollowing() ? 'outline' : 'primary',
            buttonSize: 'small',
            disabled: this.viewedFollowPending,
            onPress: () => {
              this.toggleViewedUserFollow();
            }
          })
            .layoutWeight(1)

          SystemButton({
            label: '备注',
            variant: 'outline',
            buttonSize: 'small',
            onPress: () => {
              this.openRemarkSheet();
            }
          })
            .layoutWeight(1)
        }
        .width('100%')
        .padding({
          left: DesignConstants.SPACING.LG2,
          right: DesignConstants.SPACING.LG2,
          top: DesignConstants.SPACING.MD,
          bottom: DesignConstants.SPACING.MD + this.safeAreaBottom
        })
        .backgroundColor(this.getCurrentThemePalette().surfaceBackground)
        .border({ width: { top: DesignConstants.BORDER_WIDTH.HAIRLINE }, color: this.getSurfaceBorder() })
      }
      .width('100%')
      .height('100%')
    }
  }

  @Builder
  private ViewedStatItem(value: string, label: string) {
    Column({ space: DesignConstants.SPACING.HALF }) {
      Text(value)
        .fontSize(DesignConstants.FONT.SIZE.MD)
        .fontWeight(DesignConstants.FONT.WEIGHT.SEMIBOLD)
        .fontColor(this.getPrimaryTextColor())
      Text(label)
        .fontSize(DesignConstants.FONT.SIZE.XS)
        .fontColor(this.getSecondaryTextColor())
    }
    .alignItems(HorizontalAlign.Center)
    .layoutWeight(1)
  }

  // ==================== 备注编辑半模态（对齐前端备注名称输入，仅已关注可保存） ====================
  @Builder
  private buildRemarkSheet() {
    Column({ space: DesignConstants.SPACING.MD }) {
      Text('设置备注')
        .fontSize(DesignConstants.FONT.SIZE.XL)
        .fontWeight(DesignConstants.FONT.WEIGHT.BOLD)
        .fontColor(this.getCurrentThemePalette().textPrimary)

      Text(`备注${this.viewedUserProfile?.user?.nickname ?? ''}，只对你可见`)
        .fontSize(DesignConstants.FONT.SIZE.SM)
        .fontColor(this.getCurrentThemePalette().textSecondary)

      SystemInput({
        value: this.remarkDraft,
        placeholder: '备注名称',
        fontSize: DesignConstants.FONT.SIZE.MD,
        inputHeight: DesignConstants.COMPONENT.BUTTON_HEIGHT,
        maxLength: REMARK_MAX,
        onChange: (value: string) => { this.remarkDraft = value; }
      })
        .backgroundColor(this.getCurrentThemePalette().surfaceSubtle)
        .shadow(this.getCurrentThemePalette().controlShadow)
        .borderRadius(DesignConstants.RADIUS.MD)
        .padding({ left: DesignConstants.SPACING.MD, right: DesignConstants.SPACING.MD })

      Text(`${this.remarkDraft.trim().length}/${REMARK_MAX}`)
        .fontSize(DesignConstants.FONT.SIZE.XS)
        .fontColor(this.getCurrentThemePalette().textTertiary)
        .width('100%')
        .textAlign(TextAlign.End)

      Row({ space: DesignConstants.SPACING.SM }) {
        SystemButton({
          label: '取消',
          variant: 'surface',
          buttonSize: 'sheet',
          fullWidth: true,
          onPress: () => { this.showRemarkSheet = false; }
        })
        SystemButton({
          label: '保存',
          variant: 'primary',
          buttonSize: 'sheet',
          fullWidth: true,
          loading: this.isSavingRemark,
          disabled: !this.isViewedUserFollowing(),
          onPress: () => { this.saveRemark(); }
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

  @Builder
  private buildGuestHighlight(title: string, subtitle: string) {
    Column({ space: DesignConstants.SPACING.XS }) {
      Text(title)
        .fontSize(DesignConstants.FONT.SIZE.SM)
        .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
        .fontColor(this.getPrimaryTextColor())
      Text(subtitle)
        .fontSize(DesignConstants.FONT.SIZE.XS)
        .fontColor(this.getSecondaryTextColor())
    }
    .layoutWeight(1)
    .alignItems(HorizontalAlign.Center)
    .padding({ left: DesignConstants.SPACING.SM, right: DesignConstants.SPACING.SM, top: DesignConstants.SPACING.SM, bottom: DesignConstants.SPACING.SM })
    .backgroundColor(this.getSurfaceSubtle())
    .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getSurfaceBorder() })
    .borderRadius(DesignConstants.SHAPE.COMPACT_CARD)
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
      } else {
        Scroll() {
          Column() {
            if (useMasterDetail(this.currentBreakpoint)) {
              // ======== 宽端（≥960）：保留垂直居中形态 ========
              this.buildProfileAvatar()

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
                    iconSize: DesignConstants.FONT.SIZE.XS,
                    color: this.getAccentBackground()
                  })
                  Text(this.currentUser.location.location_name)
                    .fontSize(DesignConstants.FONT.SIZE.XS)
                    .fontColor(this.getSecondaryTextColor())
                }
                .margin({ top: DesignConstants.SPACING.SM })
              }

              Button() {
                Row({ space: DesignConstants.SPACING.SM2 }) {
                  LucideIcon({
                    icon: AppIcon.EDIT,
                    iconSize: DesignConstants.FONT.SIZE.SM,
                    color: this.getSecondaryTextColor()
                  })
                  Text('编辑资料')
                    .fontSize(DesignConstants.FONT.SIZE.SM)
                    .fontColor(this.getSecondaryTextColor())
                }
              }
              .type(ButtonType.Normal)
              .width('100%')
              .height(DesignConstants.COMPONENT.SHEET_ACTION_HEIGHT)
              .borderRadius(DesignConstants.SHAPE.BUTTON)
              .backgroundColor(this.getSurfaceBackground())
              .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getSurfaceBorder() })
              .shadow(this.getCurrentThemePalette().controlShadow)
              .margin({ top: DesignConstants.SPACING.LG, bottom: DesignConstants.SPACING.LG2 })
              .onClick(() => this.startEdit())

              this.buildFigmaStatsRow()
            } else {
              // ======== 移动/平板：前端 ProfileMobile 形态（身份卡 + 个人数据列表） ========
              this.buildIdentityCard()
              this.buildProfileDataList()
            }

            // 标签栏 —— 统一 TabsPillGroup；「喜欢/收藏」无数据源
            // （spec UserProfileBundle 仅 recent_moments），移除恒空假 tab。
            // 无瞬间时不渲染 pill+画廊：单 tab 悬浮在个人数据卡下方视觉上像错位元素
            if (this.myMoments.length > 0) {
              TabsPillGroup({
                tabs: [{ label: '帖子' }],
                activeIndex: 0
              })
                .margin({ bottom: DesignConstants.SPACING.MD })

              // 画廊 (根据选中的 tab 展示不同内容)
              ProfileMomentGallery({
                selectedTab: 0,
                moments: this.myMoments,
                columns: this.getMomentColumns(),
                resolvedThemeId: this.resolvedThemeId
              })
            }

            // 菜单项
            Column({ space: 0 }) {
              if (this.userType === 'temp') {
                // 游客升级入口（原「临时用户视图」的唯一功能，保留在菜单）
                ProfileMenuItem({ title: '升级为完整账号', icon: AppIcon.EDIT, onTap: () => { this.showUpgradeForm = true; }, resolvedThemeId: this.resolvedThemeId })
              }
              ProfileMenuItem({ title: '更新位置', icon: AppIcon.LOCATION, onTap: () => this.updateLocation(), resolvedThemeId: this.resolvedThemeId })
              ProfileMenuItem({ title: '设置', icon: AppIcon.SETTINGS, onTap: () => { AppRouter.push(PageNames.SETTINGS); }, resolvedThemeId: this.resolvedThemeId })
              if (this.isAdminUser()) {
                ProfileMenuItem({ title: '管理员工具台', icon: AppIcon.SHIELD, onTap: () => this.openAdminPanel(), resolvedThemeId: this.resolvedThemeId })
              }
              ProfileMenuItem({ title: '退出登录', icon: AppIcon.CLOSE, onTap: () => { this.showLogoutConfirm = true; }, resolvedThemeId: this.resolvedThemeId })
            }
            .backgroundColor(this.getSurfaceBackground())
            .borderRadius(DesignConstants.SHAPE.CARD)
            .shadow(this.getCurrentThemePalette().elevatedShadow)
            .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getSurfaceBorder() })
            .margin({ top: DesignConstants.SPACING.XL, bottom: DesignConstants.SPACING.XL })
          }
          .alignItems(HorizontalAlign.Center)
          .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, top: DesignConstants.SPACING.SM, bottom: DesignConstants.SPACING.XL })
        }
        .layoutWeight(1)
        .scrollBar(BarState.Off)
      }
    }
    .width('100%')
    .height('100%')
    .backgroundColor(this.getCurrentThemePalette().pageBackground)
  }

  // ==================== 移动/平板身份卡（对齐前端 ProfileMobile 身份卡） ====================
  @Builder
  private buildIdentityCard() {
    Column() {
      Row({ space: DesignConstants.SPACING.MD }) {
        Stack({ alignContent: Alignment.BottomEnd }) {
          SystemAvatar({
            avatarUrl: this.currentUser?.avatar_url || '',
            name: this.currentUser?.nickname || '',
            avatarSize: DesignConstants.COMPONENT.AVATAR_SIZE.PROFILE,
            shape: 'circle',
            showBorder: true
          })

          // 相机角钮（24vp accent 实底，点击换头像）
          Button() {
            LucideIcon({
              icon: AppIcon.CAMERA,
              iconSize: DesignConstants.FONT.SIZE.XS2,
              paletteRole: 'accentText'
            })
          }
          .type(ButtonType.Normal)
          .width(DesignConstants.COMPONENT.ICON_SIZE_SM)
          .height(DesignConstants.COMPONENT.ICON_SIZE_SM)
          .borderRadius(DesignConstants.RADIUS.FULL)
          .backgroundColor(this.getAccentBackground())
          .border({
            width: DesignConstants.BORDER_WIDTH.STRONG,
            color: this.getCurrentThemePalette().pageBackground
          })
                    .onClick(() => this.changeAvatarDirectly())
          .accessibilityText('更换头像')
          // 触控热区：视觉 24 → 命中区 44（responseRegion 不占布局）
          .responseRegion({
            x: -DesignConstants.SPACING.SM,
            y: -DesignConstants.SPACING.SM,
            width: DesignConstants.LAYOUT.MIN_HIT_TARGET,
            height: DesignConstants.LAYOUT.MIN_HIT_TARGET
          })
        }

        Column({ space: DesignConstants.SPACING.THREE }) {
          Text(this.currentUser?.nickname || '未命名用户')
            .fontSize(DesignConstants.FONT.SIZE.LG)
            .fontWeight(DesignConstants.FONT.WEIGHT.SEMIBOLD)
            .fontColor(this.getPrimaryTextColor())
            .maxLines(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
          if (this.currentUser?.nickname) {
            // 对齐前端：handle 显示 @昵称（前端 ProfileMobile 用 nickname，非 user_id）
            Text(`@${this.currentUser.nickname}`)
              .fontSize(DesignConstants.FONT.SIZE.XS)
              .fontColor(this.getSecondaryTextColor())
          }
          if (this.currentUser?.location?.location_name) {
            Row({ space: DesignConstants.SPACING.HALF }) {
              LucideIcon({
                icon: AppIcon.LOCATION,
                iconSize: DesignConstants.FONT.SIZE.XS,
                color: this.getAccentBackground()
              })
              Text(this.currentUser.location.location_name)
                .fontSize(DesignConstants.FONT.SIZE.XS)
                .fontColor(this.getSecondaryTextColor())
            }
            .alignItems(VerticalAlign.Center)
          }
        }
        .alignItems(HorizontalAlign.Start)
        .layoutWeight(1)

        // 编辑资料描边小钮
        Button('编辑资料')
          .type(ButtonType.Normal)
          .fontSize(DesignConstants.FONT.SIZE.XS)
          .fontColor(this.getPrimaryTextColor())
          .backgroundColor(this.getSurfaceBackground())
          .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getSurfaceBorder() })
          .borderRadius(DesignConstants.SHAPE.INPUT)
          .padding({
            left: DesignConstants.SPACING.MD2,
            right: DesignConstants.SPACING.MD2,
            top: DesignConstants.SPACING.SM,
            bottom: DesignConstants.SPACING.SM
          })
          .onClick(() => this.startEdit())
      }
      .width('100%')
      .alignItems(VerticalAlign.Top)

      if (this.currentUser?.bio) {
        Text(this.currentUser.bio)
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .fontColor(this.getSecondaryTextColor())
          .lineHeight(DesignConstants.FONT.LINE_HEIGHT.MD)
          .width('100%')
          .margin({ top: DesignConstants.SPACING.LG })
      } else {
        // 对齐前端：无简介时显示占位文案（自己的资料页用第一人称）
        Text('还没有填写简介，介绍一下自己吧')
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .fontColor(this.getSecondaryTextColor())
          .width('100%')
          .margin({ top: DesignConstants.SPACING.LG })
      }

      // 卡底三列统计（border-t，列间留白——竖线在小屏三列里显得切割局促）
      Row() {
        this.buildIdentityStat('瞬间', this.myMomentsCount, 0)
        this.buildIdentityStat('关注者', this.myFollowersCount, 1)
        this.buildIdentityStat('群组', this.myGroupsCount, 2)
      }
      .width('100%')
      .border({
        width: { top: DesignConstants.BORDER_WIDTH.HAIRLINE },
        color: { top: this.getSurfaceBorder() }
      })
      .margin({ top: DesignConstants.SPACING.LG })
    }
    .width('100%')
    .alignItems(HorizontalAlign.Start)
    .padding(DesignConstants.SPACING.LG2)
    .borderRadius(DesignConstants.SHAPE.CARD)
    .backgroundColor(this.getSurfaceBackground())
    .shadow(this.getCurrentThemePalette().elevatedShadow)
  }

  @Builder
  private buildIdentityStat(label: string, value: number, index: number) {
    Row() {
      Column({ space: DesignConstants.SPACING.HALF }) {
        Text(formatCount(value))
          .fontSize(DesignConstants.FONT.SIZE.MD)
          .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
          .fontColor(this.getPrimaryTextColor())
        Text(label)
          .fontSize(DesignConstants.FONT.SIZE.XS)
          .fontColor(this.getSecondaryTextColor())
      }
      .layoutWeight(1)
      .justifyContent(FlexAlign.Center)
      .alignItems(HorizontalAlign.Center)
    }
    .layoutWeight(1)
    .padding({ top: DesignConstants.SPACING.MD, bottom: DesignConstants.SPACING.SM })
  }

  // ==================== 移动/平板「个人数据」列表（对齐前端 ProfileMobile 5 行） ====================
  @Builder
  private buildProfileDataList() {
    Column() {
      // 分组小标（对齐前端 ProfileMobile 改版：eyebrow + 副文案在列表外，列表去外框）
      Text('个人数据')
        .fontSize(DesignConstants.FONT.SIZE.XS2)
        .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
        .fontColor(this.getSecondaryTextColor())
        .letterSpacing(DesignConstants.FONT.LETTER_SPACING.LABEL)
        .textCase(TextCase.UpperCase)
        .width('100%')
        .padding({ left: DesignConstants.SPACING.SM2, bottom: DesignConstants.SPACING.SM })
      Text('在这里查看你的内容与关系')
        .fontSize(DesignConstants.FONT.SIZE.XS)
        .fontColor(this.getSecondaryTextColor())
        .width('100%')
        .padding({ left: DesignConstants.SPACING.SM2, bottom: DesignConstants.SPACING.SM })

      // 数据行白卡容器（表面分层：底色差 + 阴影浮起，无边框线，对齐前端 card-raised）
      Column() {
        this.buildProfileDataRow(AppIcon.PHOTO, '我的瞬间', '查看已发布的记录', () => {
          AppRouter.push(PageNames.MOMENT);
        }, true)
        this.buildProfileDataRow(AppIcon.GROUP, '我的群组', `${formatCount(this.myGroupsCount)} 个正在参与`, () => {
          AppRouter.push(PageNames.GROUP);
        }, false)
        this.buildProfileDataRow(AppIcon.PERSON, '好友', `${formatCount(this.myFriendsCount)} 位好友`, () => {
          AppRouter.push(PageNames.RELATIONSHIP_HUB);
        }, false)
        this.buildProfileDataRow(AppIcon.STAR, '关注', `${formatCount(this.myFollowingCount)} 人`, () => {
          AppRouter.push(PageNames.RELATIONSHIP_HUB);
        }, false)
        this.buildProfileDataRow(AppIcon.HEART, '粉丝', `${formatCount(this.myFollowersCount)} 人`, () => {
          AppRouter.push(PageNames.RELATIONSHIP_HUB);
        }, false)
      }
      .width('100%')
      .backgroundColor(this.getSurfaceBackground())
      .borderRadius(DesignConstants.SHAPE.CARD)
      .shadow(this.getCurrentThemePalette().elevatedShadow)
      .clip(true)
    }
    .width('100%')
    .alignItems(HorizontalAlign.Start)
    .margin({ top: DesignConstants.SPACING.LG })
  }

  @Builder
  private buildProfileDataRow(icon: AppIcon, label: string, detail: string, onTap: () => void, divider: boolean) {
    // 对齐前端 ProfileMobile：圆形 icon-badge(32) + 左侧两行（标题/副标题）+ 右箭头
    Row({ space: DesignConstants.SPACING.MD }) {
      Row() {
        LucideIcon({
          icon: icon,
          iconSize: DesignConstants.FONT.SIZE.SM,
          color: this.getAccentBackground()
        })
      }
      .width(DesignConstants.COMPONENT.AVATAR_SIZE.XS)
      .height(DesignConstants.COMPONENT.AVATAR_SIZE.XS)
      .borderRadius(DesignConstants.RADIUS.FULL)
      .backgroundColor(this.getCurrentThemePalette().accentMuted)
      .justifyContent(FlexAlign.Center)

      Column({ space: DesignConstants.SPACING.HALF }) {
        Text(label)
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .fontColor(this.getPrimaryTextColor())
          .maxLines(1)
          .textOverflow({ overflow: TextOverflow.Ellipsis })
        Text(detail)
          .fontSize(DesignConstants.FONT.SIZE.XS)
          .fontColor(this.getSecondaryTextColor())
          .maxLines(1)
          .textOverflow({ overflow: TextOverflow.Ellipsis })
      }
      .alignItems(HorizontalAlign.Start)
      .layoutWeight(1)

      LucideIcon({
        icon: AppIcon.ARROW_RIGHT,
        iconSize: DesignConstants.FONT.SIZE.SM,
        color: this.getSecondaryTextColor()
      })
    }
    .width('100%')
    .padding({
      left: DesignConstants.SPACING.LG2,
      right: DesignConstants.SPACING.LG2,
      top: DesignConstants.SPACING.MD3,
      bottom: DesignConstants.SPACING.MD3
    })
    .border(divider ? {
      width: { bottom: DesignConstants.BORDER_WIDTH.HAIRLINE },
      color: { bottom: this.getSurfaceBorder() }
    } : { width: 0 })
    .backgroundColor(this.profileRowPressed === label ?
      this.getCurrentThemePalette().surfaceSubtle :
      (this.profileRowHovered === label ? this.getCurrentThemePalette().accentMuted :
      DesignConstants.COLOR.TRANSPARENT))
    .scale(this.profileRowPressed === label ?
      { x: DesignConstants.INTERACTION.SCALE_PRESSED_INPUT_ROW, y: DesignConstants.INTERACTION.SCALE_PRESSED_INPUT_ROW } :
      { x: 1, y: 1 })
    .animation({ duration: DesignConstants.ANIMATION.DURATION_FAST, curve: Curve.EaseOut })
    .onTouch((event: TouchEvent) => {
      if (event.type === TouchType.Down) {
        this.profileRowPressed = label;
      } else if (event.type === TouchType.Up || event.type === TouchType.Cancel) {
        this.profileRowPressed = '';
      }
    })
    .onHover((isHover: boolean) => {
      this.profileRowHovered = isHover ? label : '';
    })
    .onClick(() => { onTap(); })
  }

  @Builder
  private buildProfileAvatar() {    Column() {
      Stack({ alignContent: Alignment.BottomEnd }) {
        // 头像
        SystemAvatar({
          avatarUrl: this.currentUser?.avatar_url || '',
          name: this.currentUser?.nickname || '',
          avatarSize: DesignConstants.COMPONENT.AVATAR_SIZE.XLARGE,
          shape: 'circle'
        })

        // 相机按钮
        Button() {
          LucideIcon({
            icon: AppIcon.CAMERA,
            iconSize: DesignConstants.FONT.SIZE.XS2,
            paletteRole: 'accentText'
          })
        }
        .type(ButtonType.Normal)
        .width(DesignConstants.COMPONENT.STATUS_ICON_SIZE)
        .height(DesignConstants.COMPONENT.STATUS_ICON_SIZE)
        .borderRadius(DesignConstants.RADIUS.FULL)
        .backgroundColor(this.getAccentBackground())
        .border({ width: DesignConstants.BORDER_WIDTH.HEAVY, color: this.getCurrentThemePalette().pageBackground })
                .onClick(() => this.changeAvatarDirectly())
        .accessibilityText('更换头像')
        .responseRegion({
          x: -DesignConstants.SPACING.SM,
          y: -DesignConstants.SPACING.SM,
          width: DesignConstants.LAYOUT.MIN_HIT_TARGET,
          height: DesignConstants.LAYOUT.MIN_HIT_TARGET
        })
      }
      .margin({ top: DesignConstants.SPACING.LG2, bottom: DesignConstants.SPACING.MD })

      Text(this.currentUser?.nickname || '')
        .fontSize(DesignConstants.FONT.SIZE.XL)
        .fontWeight(DesignConstants.FONT.WEIGHT.BOLD)
        .fontColor(this.getPrimaryTextColor())
    }
    .alignItems(HorizontalAlign.Center)
  }

  @Builder
  private buildFigmaStatsRow() {
    Row() {
      ForEach([
        { label: '瞬间', value: formatCount(this.myMomentsCount) },
        { label: '关注者', value: formatCount(this.myFollowersCount) },
        { label: '群组', value: formatCount(this.myGroupsCount) }
      ] as StatItem[], (stat: StatItem, index: number) => {
        Row() {
          if (index > 0) {
            Column()
              .width(DesignConstants.BORDER_WIDTH.HAIRLINE)
              .height(DesignConstants.COMPONENT.AVATAR_SIZE.XS)
              .backgroundColor(this.getSurfaceBorder())
              .margin({ left: 0, right: 0 })
          }
          Column({ space: DesignConstants.SPACING.XS }) {
            Text(stat.value)
              .fontSize(DesignConstants.FONT.SIZE.XXL)
              .fontWeight(DesignConstants.FONT.WEIGHT.BOLD)
              .fontColor(this.getPrimaryTextColor())
            Text(stat.label)
              .fontSize(DesignConstants.FONT.SIZE.XS2)
              .fontColor(this.getAccentBackground())
              .textCase(TextCase.UpperCase)
              .letterSpacing(DesignConstants.FONT.LETTER_SPACING.FIELD)
              .opacity(DesignConstants.OPACITY.EMPHASIZED)
          }
        }
        .layoutWeight(1)
        .justifyContent(FlexAlign.Center)
        .padding({ top: DesignConstants.SPACING.LG, bottom: DesignConstants.SPACING.LG })
      }, (stat: StatItem) => stat.label)
    }
    .width('100%')
    .borderRadius(DesignConstants.SHAPE.CARD)
    .backgroundColor(this.getSurfaceBackground())
    .shadow(this.getCurrentThemePalette().elevatedShadow)
    .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getSurfaceBorder() })
    .margin({ bottom: DesignConstants.SPACING.LG2 })
  }

  // 我的瞬间九宫格：按列数分行（避开 ArkUI 不支持的 calc，用 Row + layoutWeight 等宽）。
  // 列数随断点变化：宽屏(master-detail) 3 列，窄屏 2 列（对齐前端 grid-cols-2 md:grid-cols-3）。
  private getMomentColumns(): number {
    return useMasterDetail(this.currentBreakpoint) ? 3 : 2;
  }




  // ==================== 升级表单界面 ====================
  @Builder
  private buildUpgradeFormView() {
    Scroll() {
      Column({ space: DesignConstants.SPACING.LG }) {
        Text('升级为完整账号')
          .fontSize(DesignConstants.FONT.SIZE.XL)
          .fontWeight(DesignConstants.FONT.WEIGHT.BOLD)
          .fontColor(this.getPrimaryTextColor())
          .alignSelf(ItemAlign.Start)

        Text('设置用户名与密码后，临时账号将升级为完整账号，所有数据都会保留。')
          .fontSize(DesignConstants.FONT.SIZE.MD)
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
            .borderRadius(this.getProfileVisualStyle().secondaryButton.borderRadius)
            .shadow(this.getProfileVisualStyle().secondaryButton.shadow)
            .border({ width: this.getProfileVisualStyle().secondaryButton.borderWidth, color: this.getProfileVisualStyle().secondaryButton.borderColor })
            .enabled(!this.isUpgrading)
            .opacity(this.isUpgrading ? DesignConstants.OPACITY.LOADING : 1)
            .onClick(() => {
              this.showUpgradeForm = false;
            })

          Button(this.isUpgrading ? '升级中...' : '确认升级')
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
      .padding({ left: DesignConstants.SPACING.XXL, right: DesignConstants.SPACING.XXL, top: DesignConstants.SPACING.XL })
    }
    .layoutWeight(1)
  }

  // ==================== 公共 Builder ====================

  @Builder
  private UserInfoContent(): void {
    Column({ space: DesignConstants.SPACING.XL }) {
      Column({ space: DesignConstants.SPACING.MD }) {
        // 身份卡头部：对齐前端 ProfileMobile —— 头像(68)+相机角标 | 昵称(xl)+位置 | 编辑资料按钮
        Row({ space: DesignConstants.SPACING.MD }) {
          Stack({ alignContent: Alignment.BottomEnd }) {
            SystemAvatar({
              avatarUrl: this.currentUser?.avatar_url || '',
              name: this.currentUser?.nickname || '',
              avatarSize: DesignConstants.COMPONENT.AVATAR_SIZE.PROFILE,
              isOnline: this.currentUser?.presence_status === 'online',
              inset: true
            })
            // 相机角标：accent 圆 + 白描边（对齐前端 size-6 camera badge），进编辑态
            Row() {
              LucideIcon({
                icon: AppIcon.CAMERA,
                iconSize: DesignConstants.FONT.SIZE.XS2,
                color: this.getAccentText()
              })
            }
            .width(DesignConstants.COMPONENT.AVATAR_SIZE.XXS + 2)
            .height(DesignConstants.COMPONENT.AVATAR_SIZE.XXS + 2)
            .borderRadius(DesignConstants.RADIUS.FULL)
            .backgroundColor(this.getAccentBackground())
            .border({ width: DesignConstants.BORDER_WIDTH.STRONG, color: this.getCurrentThemePalette().pageBackground })
            .justifyContent(FlexAlign.Center)
            .onClick(() => this.startEdit())
            .accessibilityText('编辑头像')
            .responseRegion({
              x: -DesignConstants.SPACING.SM,
              y: -DesignConstants.SPACING.SM,
              width: DesignConstants.LAYOUT.MIN_HIT_TARGET,
              height: DesignConstants.LAYOUT.MIN_HIT_TARGET
            })
          }
          .width(DesignConstants.COMPONENT.AVATAR_SIZE.PROFILE)
          .height(DesignConstants.COMPONENT.AVATAR_SIZE.PROFILE)

          Column({ space: DesignConstants.SPACING.XS }) {
            Text(this.currentUser?.nickname || '未设置昵称')
              .fontSize(DesignConstants.FONT.SIZE.XL)
              .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
              .fontColor(this.getPrimaryTextColor())
              .maxLines(1)
              .textOverflow({ overflow: TextOverflow.Ellipsis })
            if (this.userType === 'temp') {
              Text('临时身份')
                .fontSize(DesignConstants.FONT.SIZE.XS)
                .fontColor(this.getWarningTextColor())
                .maxLines(1)
            } else if (this.isAdminUser()) {
              Text('管理员')
                .fontSize(DesignConstants.FONT.SIZE.XS)
                .fontColor(this.getAccentBackground())
                .maxLines(1)
            }
            Row({ space: DesignConstants.SPACING.XS }) {
              LucideIcon({
                icon: AppIcon.LOCATION,
                iconSize: DesignConstants.FONT.SIZE.SM,
                color: this.getAccentBackground()
              })
              Text(this.currentUser?.location?.location_name || '未设置位置')
                .fontSize(DesignConstants.FONT.SIZE.XS)
                .fontColor(this.getSecondaryTextColor())
                .maxLines(1)
                .textOverflow({ overflow: TextOverflow.Ellipsis })
            }
            .alignItems(VerticalAlign.Center)
          }
          .alignItems(HorizontalAlign.Start)
          .layoutWeight(1)

          Row({ space: DesignConstants.SPACING.SM2 }) {
            LucideIcon({
              icon: AppIcon.EDIT,
              iconSize: DesignConstants.FONT.SIZE.SM,
              color: this.getPrimaryTextColor()
            })
            Text('编辑资料')
              .fontSize(DesignConstants.FONT.SIZE.XS)
              .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
              .fontColor(this.getPrimaryTextColor())
          }
          .padding({
            left: DesignConstants.SPACING.MD,
            right: DesignConstants.SPACING.MD,
            top: DesignConstants.SPACING.SM,
            bottom: DesignConstants.SPACING.SM
          })
          .borderRadius(DesignConstants.RADIUS.MD)
          .backgroundColor(this.getSurfaceBackground())
          .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getSurfaceBorder() })
          .onClick(() => this.startEdit())
        }
        .width('100%')
        .alignItems(VerticalAlign.Top)

        if (this.currentUser?.bio) {
          // 自己的资料页用第一人称占位（「这个人」是他人视角文案）
          Text(this.currentUser.bio || '还没有填写简介，介绍一下自己吧')
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .lineHeight(DesignConstants.FONT.LINE_HEIGHT.LG)
            .fontColor(this.getSecondaryTextColor())
            .textAlign(TextAlign.Start)
            .maxLines(3)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
            .width('100%')
            .margin({ top: DesignConstants.SPACING.MD })
        }
      }
      .width('100%')
      .alignItems(HorizontalAlign.Start)

      // 个人数据列表（对齐前端 ProfileMobile：header + 5 行带 detail 副标题 + 计数）
      Column() {
        // header（对齐前端 ListContainer 头：「个人数据 / 在这里查看你的内容与关系」）
        Column({ space: DesignConstants.SPACING.HALF }) {
          Text('个人数据')
            .fontSize(DesignConstants.FONT.SIZE.MD)
            .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
            .fontColor(this.getPrimaryTextColor())
          Text('在这里查看你的内容与关系')
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .fontColor(this.getSecondaryTextColor())
        }
        .alignItems(HorizontalAlign.Start)
        .width('100%')
        .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, top: DesignConstants.SPACING.MD2, bottom: DesignConstants.SPACING.MD2 })

        ProfileMenuItem({ title: '我的瞬间', icon: AppIcon.CAMERA, value: formatCount(this.myMomentsCount), detail: '查看已发布的记录', onTap: () => this.openMyMoments(), resolvedThemeId: this.resolvedThemeId })
        ProfileMenuItem({ title: '我的群组', icon: AppIcon.GROUP, value: formatCount(this.myGroupsCount), detail: `${formatCount(this.myGroupsCount)} 个正在参与`, onTap: () => this.openMyGroups(), resolvedThemeId: this.resolvedThemeId })
        ProfileMenuItem({ title: '好友', icon: AppIcon.HEART, value: formatCount(this.myFriendsCount), detail: `${formatCount(this.myFriendsCount)} 位好友`, onTap: () => this.openRelationshipHub(), resolvedThemeId: this.resolvedThemeId })
        ProfileMenuItem({ title: '关注', icon: AppIcon.PERSON, value: formatCount(this.myFollowingCount), detail: `${formatCount(this.myFollowingCount)} 人`, onTap: () => this.openRelationshipHub(), resolvedThemeId: this.resolvedThemeId })
        ProfileMenuItem({ title: '粉丝', icon: AppIcon.COMPASS, value: formatCount(this.myFollowersCount), detail: `${formatCount(this.myFollowersCount)} 人`, onTap: () => this.openRelationshipHub(), resolvedThemeId: this.resolvedThemeId })
      }
      .width('100%')
      .padding(DesignConstants.SPACING.MD)
      .backgroundColor(this.getProfileVisualStyle().card.backgroundColor)
      .borderRadius(this.getProfileVisualStyle().card.borderRadius)
      .shadow(this.getProfileVisualStyle().card.shadow)
      .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getSurfaceBorder() })
    }
    .width('100%')
  }

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
          Text('头像上传中...')
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .fontColor(this.getSecondaryTextColor())
        } else {
          Text('点击头像更换')
            .fontSize(DesignConstants.FONT.SIZE.SM)
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
          .borderRadius(this.getProfileVisualStyle().card.borderRadius)
          .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getSurfaceBorder() })
          .enabled(!this.isLoading && !this.isAvatarUploading)
          .opacity(this.isLoading || this.isAvatarUploading ? DesignConstants.OPACITY.LOADING : 1)
          .onClick(() => this.cancelEdit())

        Button(this.isLoading ? '保存中...' : '保存')
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

  @Builder
  private StatsContent(): void {
    Column() {
      Row({ space: DesignConstants.SPACING.SM }) {
        CompactStatItem({ icon: AppIcon.CAMERA, value: formatCount(this.myMomentsCount), label: '瞬间', onTap: () => this.openMyMoments(), resolvedThemeId: this.resolvedThemeId })
        CompactStatItem({ icon: AppIcon.HEART, value: formatCount(this.myFollowersCount), label: '关注者', onTap: () => this.openRelationshipHub(), resolvedThemeId: this.resolvedThemeId })
        CompactStatItem({ icon: AppIcon.GROUP, value: formatCount(this.myGroupsCount), label: '群组', onTap: () => this.openMyGroups(), resolvedThemeId: this.resolvedThemeId })
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