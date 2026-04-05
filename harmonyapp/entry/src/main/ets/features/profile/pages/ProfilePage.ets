/**
 * 个人中心页面
 * 三级用户体系：
 * - 匿名（无 token）：游客浏览模式，引导登录/注册
 * - 临时用户（temp）：已认证，显示升级入口
 * - 注册用户（normal）：完整功能
 */

import { GlobalStateManager, KEY_CURRENT_USER, KEY_IS_LOGGED_IN, KEY_USER_TYPE, KEY_CURRENT_LOCATION } from '../../../core/state/GlobalStateManager';
import { LocationInfo } from '../../../core/location/LocationUtil';
import { ApiClient } from '../../../core/api/ApiClient';
import { PreferencesUtil } from '../../../core/storage/PreferencesUtil';
import { HttpUtil } from '../../../core/network/HttpUtil';
import { WebSocketService } from '../../../core/network/WebSocketService';
import { AppRouter, ChatRouteParam, LoginRouteParam } from '../../../core/router/AppRouter';
import { PageNames } from '../../../constants/AppConstants';
import { MediaServiceManager } from '../../../core/media/MediaServiceManager';
import { SystemAvatar } from '../../../components/SystemAvatar';
import { SystemToast } from '../../../components/SystemToast';
import { SystemLoadingView } from '../../../components/SystemLoadingView';
import { MyProfile, UpdateLocationRequest, UpgradeAccountRequest, GuestRequest, UserItem, UserOpenTheDoorRequest } from '../../../core/types/api';
import { AuthAccountManager } from '../../../core/state/AuthAccountManager';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { common } from '@kit.AbilityKit';
import { DesignConstants } from '../../../constants/DesignConstants';
import { AppIcon } from '../../../components/SystemIcon';
import { EmptyState } from '../../../components/EmptyState';
import { SystemPageHeader } from '../../../components/SystemPageHeader';
import { LoadingUtil } from '../../../core/utils/LoadingUtil';
import { NfcShareService } from '../../../core/services/NfcShareService';
import { SystemUiController } from '../../../core/utils/SystemUiController';
import { KeyboardDismissManager } from '../../../core/utils/KeyboardDismissManager';
import { SafeAreaUtils } from '../../../core/utils/SafeAreaUtils';
import { formatCount } from '../../../core/utils/FormatUtil';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { AppThemeVisuals } from '../../../core/theme/AppThemeVisuals';
import { KEY_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';

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
  @StorageLink(KEY_CURRENT_USER) currentUser: MyProfile | null = null;
  @StorageLink(KEY_IS_LOGGED_IN) isLoggedIn: boolean = false;
  @StorageLink(KEY_USER_TYPE) userType: string = '';
  @StorageLink(KEY_CURRENT_LOCATION) currentLocation: LocationInfo | null = null;
  @StorageLink('safeAreaBottom') safeAreaBottom: number = 0;
  @StorageLink(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'default_clean_light';

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
  @State myFollowersCount: number = 0;
  @State myFollowingCount: number = 0;
  // 快速体验弹窗
  @State showQuickStart: boolean = false;
  @State quickNickname: string = '';
  @State isQuickStarting: boolean = false;
  @State isQuickInputFocused: boolean = false;
  // 升级表单
  @State showUpgradeForm: boolean = false;
  @State upgradeLoginId: string = '';
  @State upgradePassword: string = '';
  @State upgradePasswordConfirm: string = '';
  @State isUpgrading: boolean = false;
  @State hasAdminPrivileges: boolean = false;
  @State pressedMenuItemKey: string = '';
  @State viewedUserId: string = '';
  @State viewedUserProfile: UserItem | null = null;
  @State viewedUserLoading: boolean = false;
  @State viewedUserError: string = '';
  private isDestroyed: boolean = false;

  private webSocketService = WebSocketService.getInstance();

  private isValidLoginId(loginId: string): boolean {
    const trimmed = loginId.trim();
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
    if (!this.isExternalProfileMode()) {
      return '个人中心';
    }
    const nickname = this.viewedUserProfile?.user?.nickname ?? '';
    return nickname.length > 0 ? nickname : '用户资料';
  }

  private async loadViewedUserProfile(): Promise<void> {
    if (!this.viewedUserId) {
      return;
    }
    this.viewedUserLoading = true;
    this.viewedUserError = '';
    this.viewedUserProfile = null;
    try {
      const response = await LoadingUtil.withMinimumTime(() =>
        ApiClient.getUserProfile({ user_id: this.viewedUserId } as UserOpenTheDoorRequest),
      DesignConstants.ANIMATION.DURATION_CONTENT_SWITCH);
      if (this.isDestroyed) return;
      if (response.code === 0 && response.data?.user?.user_id) {
        this.viewedUserProfile = response.data;
      } else {
        this.viewedUserError = response.message || '无法获取该用户资料';
      }
    } catch {
      this.viewedUserError = '加载用户资料失败';
    } finally {
      this.viewedUserLoading = false;
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
    return this.getCurrentThemePalette().accentBackground;
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
            throw new Error(updateProfileResponse.message || '资料更新失败');
          }

          const response = await ApiClient.getMyProfile();
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

      // 3. 清理本地持久化数据（Preferences + token）
      await PreferencesUtil.clear();
      await HttpUtil.clearToken();

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
        await HttpUtil.clearToken();
      } catch (_) {
        // ignore
      }
      SystemToast.show('退出登录遇到问题，请重启应用', this.getUIContext());
    }
  }

  private async upgradeAccount(): Promise<void> {
    const trimmedLoginId = this.upgradeLoginId.trim();
    if (!trimmedLoginId) {
      SystemToast.showWarning('请输入邮箱/用户名', this.getUIContext());
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
        const profileResp = await ApiClient.getMyProfile();
        if (profileResp.code === 0 && profileResp.data) {
          this.stateManager.updateUser(profileResp.data);
        }
        this.showUpgradeForm = false;
        this.upgradeLoginId = '';
        this.upgradePassword = '';
        this.upgradePasswordConfirm = '';
        SystemToast.showSuccess('账号升级成功！', this.getUIContext());
      } else {
        SystemToast.showError(response.message || '升级失败', this.getUIContext());
      }
    } catch (error) {
      SystemToast.showError(error instanceof Error ? error.message : '升级失败', this.getUIContext());
    } finally {
      this.isUpgrading = false;
    }
  }

  build() {
    NavDestination() {
      Stack({ alignContent: Alignment.Top }) {
        Column() {}
          .width('50%')
          .height(180)
          .borderRadius(DesignConstants.RADIUS.FULL)
          .backgroundColor(this.getCurrentThemePalette().pageGlow)
          .opacity(this.getCurrentThemePalette().isDarkTheme ? 0.06 : 0.18)
          .blur(80)
          .align(Alignment.Top)
          .margin({ top: 12 })

        Column() {}
          .width('32%')
          .height(120)
          .borderRadius(DesignConstants.RADIUS.FULL)
          .backgroundColor(this.getCurrentThemePalette().heroGlow)
          .opacity(this.getCurrentThemePalette().isDarkTheme ? 0.04 : 0.12)
          .blur(64)
          .align(Alignment.TopEnd)
          .margin({ top: 64, right: 12 })

        Column() {
          SystemPageHeader({
            title: this.resolveHeaderTitle(),
            showBackButton: this.isExternalProfileMode(),
            onBack: this.isExternalProfileMode() ? () => { AppRouter.pop(); } : undefined,
            action: (!this.isExternalProfileMode() && this.isLoggedIn && this.currentUser && !this.isEditing && !this.showUpgradeForm)
              ? { label: '编辑', onClick: () => this.startEdit() }
              : undefined
          })

          if (this.isExternalProfileMode()) {
            Column() {
              this.buildExternalProfileView()
            }
            .width('100%')
            .justifyContent(FlexAlign.Start)
            .alignItems(HorizontalAlign.Start)
            .layoutWeight(1)
            .transition(
              TransitionEffect.OPACITY.animation({ duration: 320, curve: Curve.FastOutSlowIn })
                .combine(TransitionEffect.move(TransitionEdge.BOTTOM).animation({ duration: 320, curve: Curve.FastOutSlowIn }))
            )
          } else if (this.isLoading) {
            SystemLoadingView({ message: '正在更新...' })
              .layoutWeight(1)
              .transition(
                TransitionEffect.OPACITY.animation({ duration: 320, curve: Curve.FastOutSlowIn })
                  .combine(TransitionEffect.move(TransitionEdge.BOTTOM).animation({ duration: 320, curve: Curve.FastOutSlowIn }))
              )
          } else if (!this.isLoggedIn) {
            Column() {
              this.buildGuestView()
            }
            .layoutWeight(1)
            .transition(
              TransitionEffect.OPACITY.animation({ duration: 320, curve: Curve.FastOutSlowIn })
                .combine(TransitionEffect.move(TransitionEdge.BOTTOM).animation({ duration: 320, curve: Curve.FastOutSlowIn }))
            )
          } else if (this.showUpgradeForm) {
            Column() {
              this.buildUpgradeFormView()
            }
            .layoutWeight(1)
            .transition(
              TransitionEffect.OPACITY.animation({ duration: 320, curve: Curve.FastOutSlowIn })
                .combine(TransitionEffect.move(TransitionEdge.BOTTOM).animation({ duration: 320, curve: Curve.FastOutSlowIn }))
            )
          } else if (this.userType === 'temp') {
            Column() {
              this.buildTempUserView()
            }
            .layoutWeight(1)
            .transition(
              TransitionEffect.OPACITY.animation({ duration: 320, curve: Curve.FastOutSlowIn })
                .combine(TransitionEffect.move(TransitionEdge.BOTTOM).animation({ duration: 320, curve: Curve.FastOutSlowIn }))
            )
          } else {
            Column() {
              this.buildRegisteredUserView()
            }
            .layoutWeight(1)
            .transition(
              TransitionEffect.OPACITY.animation({ duration: 320, curve: Curve.FastOutSlowIn })
                .combine(TransitionEffect.move(TransitionEdge.BOTTOM).animation({ duration: 320, curve: Curve.FastOutSlowIn }))
            )
          }
        }
        .width('100%')
        .height('100%')
      }
      .width('100%')
      .height('100%')
      .backgroundColor(this.getCurrentThemePalette().pageBackground)
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
      Scroll() {
        Column({ space: DesignConstants.SPACING.MD }) {
          Column({ space: DesignConstants.SPACING.SM }) {
            SystemAvatar({
              avatarUrl: this.viewedUserProfile?.user?.avatar_url ?? '',
              name: this.viewedUserProfile?.user?.nickname ?? '',
              avatarSize: this.getProfileVisualStyle().identity.avatarSize,
              shape: 'circle'
            })

            Text(this.viewedUserProfile?.user?.nickname ?? '未知用户')
              .fontSize(DesignConstants.FONT.SIZE.XXL)
              .fontWeight(FontWeight.Bold)
              .fontColor(this.getPrimaryTextColor())
              .textAlign(TextAlign.Center)
              .maxLines(1)
              .textOverflow({ overflow: TextOverflow.Ellipsis })

            if (this.viewedUserProfile?.presence_status === 'online') {
              Text('在线')
                .fontSize(DesignConstants.FONT.SIZE.SM)
                .fontColor(this.getAccentBackground())
            } else {
              Text('离线')
                .fontSize(DesignConstants.FONT.SIZE.SM)
                .fontColor(this.getSecondaryTextColor())
            }
          }
          .width('100%')
          .alignItems(HorizontalAlign.Center)
          .padding({
            left: DesignConstants.SPACING.MD,
            right: DesignConstants.SPACING.MD,
            top: DesignConstants.SPACING.LG,
            bottom: DesignConstants.SPACING.LG
          })
          .backgroundColor(this.getProfileVisualStyle().card.backgroundColor)
          .borderRadius(this.getProfileVisualStyle().card.borderRadius)
          .border({ width: this.getProfileVisualStyle().card.borderWidth, color: this.getProfileVisualStyle().card.borderColor })
          .shadow(this.getProfileVisualStyle().card.shadow)

          Column({ space: DesignConstants.SPACING.SM }) {
            Text('个人简介')
              .fontSize(DesignConstants.FONT.SIZE.SM)
              .fontColor(this.getSecondaryTextColor())
            Text(this.viewedUserProfile?.bio?.length ? this.viewedUserProfile?.bio : '这个人很神秘，暂时还没有留下简介')
              .fontSize(DesignConstants.FONT.SIZE.MD)
              .lineHeight(22)
              .fontColor(this.getPrimaryTextColor())
          }
          .width('100%')
          .padding(DesignConstants.SPACING.LG)
          .backgroundColor(this.getProfileVisualStyle().card.backgroundColor)
          .borderRadius(this.getProfileVisualStyle().card.borderRadius)
          .border({ width: this.getProfileVisualStyle().card.borderWidth, color: this.getProfileVisualStyle().card.borderColor })
          .shadow(this.getProfileVisualStyle().card.shadow)

          if ((this.viewedUserProfile?.tags?.length ?? 0) > 0) {
            Flex({ wrap: FlexWrap.Wrap }) {
              ForEach((this.viewedUserProfile?.tags ?? []).slice(0, 8), (tag: string) => {
                Text(`#${tag}`)
                  .fontSize(DesignConstants.FONT.SIZE.SM)
                  .fontColor(this.getAccentBackground())
                  .padding({ left: DesignConstants.SPACING.SM, right: DesignConstants.SPACING.SM, top: DesignConstants.SPACING.XS, bottom: DesignConstants.SPACING.XS })
                  .margin({ right: DesignConstants.SPACING.SM, bottom: DesignConstants.SPACING.SM })
                  .backgroundColor(this.getSurfaceSubtle())
                  .borderRadius(DesignConstants.RADIUS.XL)
              }, (tag: string) => tag)
            }
            .width('100%')
            .padding(DesignConstants.SPACING.LG)
            .backgroundColor(this.getSurfaceBackground())
            .borderRadius(this.getProfileVisualStyle().card.borderRadius)
            .border({ width: 1, color: this.getSurfaceBorder() })
            .shadow(this.getProfileVisualStyle().card.shadow)
          }

          Row({ space: DesignConstants.SPACING.SM }) {
            Button('私聊')
              .type(ButtonType.Normal)
              .layoutWeight(1)
              .height(this.getProfileVisualStyle().secondaryButton.height)
              .fontSize(DesignConstants.FONT.SIZE.MD)
              .fontWeight(FontWeight.Medium)
              .backgroundColor(this.getAccentBackground())
              .fontColor(this.getAccentText())
              .borderRadius(this.getProfileVisualStyle().primaryButton.borderRadius)
              .shadow(this.getProfileVisualStyle().primaryButton.shadow)
              .onClick(() => {
                this.startPrivateChatWithViewedUser();
              })

            Button('返回')
              .type(ButtonType.Normal)
              .layoutWeight(1)
              .height(this.getProfileVisualStyle().secondaryButton.height)
              .fontSize(DesignConstants.FONT.SIZE.MD)
              .fontWeight(FontWeight.Medium)
              .backgroundColor(this.getProfileVisualStyle().secondaryButton.backgroundColor)
              .fontColor(this.getProfileVisualStyle().secondaryButton.textColor)
              .borderRadius(this.getProfileVisualStyle().secondaryButton.borderRadius)
              .shadow(this.getProfileVisualStyle().secondaryButton.shadow)
              .border({ width: this.getProfileVisualStyle().secondaryButton.borderWidth, color: this.getProfileVisualStyle().secondaryButton.borderColor })
              .onClick(() => {
                AppRouter.pop();
              })
          }
          .width('100%')
        }
        .width('100%')
        .justifyContent(FlexAlign.Start)
        .padding({
          left: DesignConstants.SPACING.LG,
          right: DesignConstants.SPACING.LG,
          top: DesignConstants.SPACING.SM,
          bottom: DesignConstants.SPACING.LG
        })
        .alignItems(HorizontalAlign.Start)
      }
      .width('100%')
      .layoutWeight(1)
      .align(Alignment.TopStart)
      .scrollBar(BarState.Auto)
    }
  }

  // ==================== 匿名用户 - 游客界面 ====================
  @Builder
  private buildGuestView() {
    Scroll() {
      Column({ space: DesignConstants.SPACING.LG }) {
        Column({ space: DesignConstants.SPACING.LG }) {
          Stack() {
            Circle()
              .width(this.getProfileVisualStyle().identity.avatarSize)
              .height(this.getProfileVisualStyle().identity.avatarSize)
              .fill(this.getSurfaceSubtle())
              .shadow(this.getProfileVisualStyle().identity.haloShadow)
            Text() {
              SymbolSpan($r(AppIcon.PERSON))
                .fontSize(this.getProfileVisualStyle().identity.avatarSize * 0.46)
                .fontColor([this.getSecondaryTextColor()])
            }
          }

          Column({ space: DesignConstants.SPACING.SM }) {
            Text('游客模式')
              .fontSize(this.getProfileVisualStyle().identity.titleFontSize)
              .fontWeight(FontWeight.Medium)
              .fontColor(this.getPrimaryTextColor())
              .textAlign(TextAlign.Center)

            Text('先体验附近探索、内容发现与基础互动；登录后再保存资料、偏好与关系。')
              .fontSize(DesignConstants.FONT.SIZE.MD)
              .lineHeight(24)
              .fontColor(this.getSecondaryTextColor())
              .textAlign(TextAlign.Center)
          }
          .width('100%')

          Row({ space: DesignConstants.SPACING.SM }) {
            this.buildGuestHighlight('无需注册', '先看再决定')
            this.buildGuestHighlight('圆角轻界面', '统一视觉触感')
          }
          .width('100%')
        }
        .width('100%')
        .padding(this.getProfileVisualStyle().card.padding)
        .backgroundColor(this.getProfileVisualStyle().card.backgroundColor)
        .border({ width: this.getProfileVisualStyle().card.borderWidth, color: this.getProfileVisualStyle().card.borderColor })
        .borderRadius(this.getProfileVisualStyle().card.borderRadius)
        .shadow(this.getProfileVisualStyle().card.shadow)
        .alignItems(HorizontalAlign.Center)

        Column({ space: DesignConstants.SPACING.MD }) {
          Text('可先体验这些能力')
            .width('100%')
            .fontSize(DesignConstants.FONT.SIZE.LG)
            .fontWeight(FontWeight.Medium)
            .fontColor(this.getPrimaryTextColor())

          Grid() {
            GridItem() {
              this.buildFeatureCard($r(AppIcon.MAP), '附近', '发现附近热点与地图灵感', () => AppRouter.push(PageNames.SEARCH))
            }
            GridItem() {
              this.buildFeatureCard($r(AppIcon.SEARCH), '发现', '找到兴趣用户与内容线索', () => AppRouter.push(PageNames.SEARCH))
            }
            GridItem() {
              this.buildFeatureCard($r(AppIcon.MESSAGE), '消息', '登录后开启实时对话', () => {
                if (this.isLoggedIn) {
                  AppRouter.push(PageNames.CHAT);
                } else {
                  AppRouter.push(PageNames.LOGIN);
                }
              })
            }
            GridItem() {
              this.buildFeatureCard($r(AppIcon.EDIT), '发布', '记录当下灵感与生活切片', () => {
                if (this.isLoggedIn) {
                  this.showCreateSheet();
                } else {
                  AppRouter.push(PageNames.LOGIN);
                }
              })
            }
          }
          .columnsTemplate('1fr 1fr')
          .rowsGap(DesignConstants.SPACING.MD)
          .columnsGap(DesignConstants.SPACING.MD)
          .width('100%')
        }
        .width('100%')
        .padding(this.getProfileVisualStyle().card.padding)
        .backgroundColor(this.getProfileVisualStyle().card.backgroundColor)
        .border({ width: this.getProfileVisualStyle().card.borderWidth, color: this.getProfileVisualStyle().card.borderColor })
        .borderRadius(this.getProfileVisualStyle().card.borderRadius)
        .shadow(this.getProfileVisualStyle().card.shadow)

        Column({ space: DesignConstants.SPACING.SM }) {
          Text('登录后可同步资料、保存偏好并开启完整互动。')
            .width('100%')
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .lineHeight(20)
            .fontColor(this.getSecondaryTextColor())
            .textAlign(TextAlign.Start)

          Button('快速体验（无需注册）')
            .type(ButtonType.Normal)
            .width('100%')
            .height(this.getProfileVisualStyle().primaryButton.height)
            .backgroundColor(this.getProfileVisualStyle().primaryButton.backgroundColor)
            .fontColor(this.getProfileVisualStyle().primaryButton.textColor)
            .fontSize(DesignConstants.FONT.SIZE.MD)
            .fontWeight(FontWeight.Bold)
            .shadow(this.getProfileVisualStyle().primaryButton.shadow)
            .border({ width: this.getProfileVisualStyle().primaryButton.borderWidth, color: this.getProfileVisualStyle().primaryButton.borderColor })
            .borderRadius(this.getProfileVisualStyle().primaryButton.borderRadius)
            .padding({ left: this.getProfileVisualStyle().primaryButton.horizontalPadding, right: this.getProfileVisualStyle().primaryButton.horizontalPadding })
            .onClick(() => {
              this.showQuickStart = true;
            })

          Button('登录 / 注册')
            .type(ButtonType.Normal)
            .width('100%')
            .height(this.getProfileVisualStyle().secondaryButton.height)
            .backgroundColor(this.getProfileVisualStyle().secondaryButton.backgroundColor)
            .fontColor(this.getProfileVisualStyle().secondaryButton.textColor)
            .fontSize(DesignConstants.FONT.SIZE.MD)
            .fontWeight(FontWeight.Medium)
            .shadow(this.getProfileVisualStyle().secondaryButton.shadow)
            .border({ width: this.getProfileVisualStyle().secondaryButton.borderWidth, color: this.getProfileVisualStyle().secondaryButton.borderColor })
            .borderRadius(this.getProfileVisualStyle().secondaryButton.borderRadius)
            .padding({ left: this.getProfileVisualStyle().secondaryButton.horizontalPadding, right: this.getProfileVisualStyle().secondaryButton.horizontalPadding })
            .onClick(() => {
              AppRouter.push(PageNames.LOGIN);
            })
        }
        .width('100%')
        .padding(this.getProfileVisualStyle().card.padding)
        .backgroundColor(this.getProfileVisualStyle().card.backgroundColor)
        .border({ width: this.getProfileVisualStyle().card.borderWidth, color: this.getProfileVisualStyle().card.borderColor })
        .borderRadius(this.getProfileVisualStyle().card.borderRadius)
        .shadow(this.getProfileVisualStyle().card.shadow)

        this.MenuItemContent('设置', AppIcon.SETTINGS, () => {
          AppRouter.push(PageNames.SETTINGS);
        })
      }
      .width('100%')
      .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, top: DesignConstants.SPACING.SM, bottom: DesignConstants.SPACING.LG })
    }
    .width('100%')
    .layoutWeight(1)
    .scrollBar(BarState.Auto)
    .bindSheet($$this.showQuickStart, this.buildQuickStartSheet(), {
      height: SheetSize.FIT_CONTENT,
      showClose: true,
      dragBar: true,
      backgroundColor: this.getCurrentThemePalette().sheetBackground,
      keyboardAvoidMode: SheetKeyboardAvoidMode.TRANSLATE_AND_SCROLL
    })
  }

  @Builder
  private buildGuestHighlight(title: string, subtitle: string) {
    Column({ space: 4 }) {
      Text(title)
        .fontSize(DesignConstants.FONT.SIZE.SM)
        .fontWeight(FontWeight.Medium)
        .fontColor(this.getPrimaryTextColor())
      Text(subtitle)
        .fontSize(DesignConstants.FONT.SIZE.XS)
        .fontColor(this.getSecondaryTextColor())
    }
    .layoutWeight(1)
    .alignItems(HorizontalAlign.Center)
    .padding({ left: DesignConstants.SPACING.SM, right: DesignConstants.SPACING.SM, top: DesignConstants.SPACING.SM, bottom: DesignConstants.SPACING.SM })
    .backgroundColor(this.getSurfaceSubtle())
    .border({ width: 1, color: this.getSurfaceBorder() })
    .borderRadius(DesignConstants.SHAPE.COMPACT_CARD)
  }

  private showCreateSheet(): void {
    // 委托父级 Index 处理创建动作（通过 AppStorage 通信）
    AppStorage.setOrCreate('triggerCreateAction', Date.now());
  }

  @Builder
  private buildFeatureCard(icon: Resource, title: string, subtitle: string, onTap?: () => void) {
    Column({ space: DesignConstants.SPACING.SM }) {
      Row() {
        Text() {
          SymbolSpan(icon)
            .fontSize(this.getProfileVisualStyle().quickAction.iconSize)
            .fontColor([this.getAccentBackground()])
        }
      }
      .width(this.getProfileVisualStyle().quickAction.iconWrapperSize)
      .height(this.getProfileVisualStyle().quickAction.iconWrapperSize)
      .borderRadius(DesignConstants.RADIUS.FULL)
      .backgroundColor(this.getSurfaceSubtle())
      .border({ width: 1, color: this.getSurfaceBorder() })
      .shadow(this.getProfileVisualStyle().quickAction.shadow)
      .justifyContent(FlexAlign.Center)

      Text(title)
        .fontSize(DesignConstants.FONT.SIZE.MD)
        .fontWeight(FontWeight.Medium)
        .fontColor(this.getPrimaryTextColor())
      Text(subtitle)
        .fontSize(DesignConstants.FONT.SIZE.SM)
        .lineHeight(20)
        .fontColor(this.getSecondaryTextColor())
        .textAlign(TextAlign.Center)
    }
    .width('100%')
    .height('100%')
    .alignItems(HorizontalAlign.Center)
    .justifyContent(FlexAlign.Center)
    .padding(this.getProfileVisualStyle().quickAction.padding)
    .backgroundColor(this.getProfileVisualStyle().quickAction.backgroundColor)
    .borderRadius(this.getProfileVisualStyle().quickAction.borderRadius)
    .border({ width: this.getProfileVisualStyle().quickAction.borderWidth, color: this.getProfileVisualStyle().quickAction.borderColor })
    .shadow(this.getProfileVisualStyle().quickAction.shadow)
    .onClick(() => {
      if (onTap) {
        onTap();
      }
    })
  }

  // ==================== 临时用户界面 ====================
  @Builder
  private buildTempUserView() {
    List() {
      ListItem() {
        if (this.isEditing) {
          this.EditFormContent()
        } else {
          this.UserInfoContent()
        }
      }
      .backgroundColor(this.getProfileVisualStyle().card.backgroundColor)
      .borderRadius(this.getProfileVisualStyle().card.borderRadius)
      .shadow(this.getProfileVisualStyle().card.shadow)
      .border({ width: this.getProfileVisualStyle().card.borderWidth, color: this.getProfileVisualStyle().card.borderColor })
      .padding(this.getProfileVisualStyle().card.padding)
      .margin({ bottom: DesignConstants.SPACING.MD })

      ListItem() {
        this.StatsContent()
      }
      .backgroundColor(this.getProfileVisualStyle().card.backgroundColor)
      .borderRadius(this.getProfileVisualStyle().card.borderRadius)
      .shadow(this.getProfileVisualStyle().card.shadow)
      .border({ width: this.getProfileVisualStyle().card.borderWidth, color: this.getProfileVisualStyle().card.borderColor })
      .padding(this.getProfileVisualStyle().card.padding)
      .margin({ bottom: DesignConstants.SPACING.MD })

      ListItem() {
        Column({ space: DesignConstants.SPACING.MD }) {
          Row({ space: DesignConstants.SPACING.SM }) {
            Text() {
              SymbolSpan($r(AppIcon.INFO))
                .fontSize(DesignConstants.FONT.SIZE.LG)
                .fontColor([this.getWarningTextColor()])
            }
            Text('临时身份')
              .fontSize(DesignConstants.FONT.SIZE.MD)
              .fontWeight(FontWeight.Medium)
              .fontColor(this.getWarningTextColor())
          }
          .width('100%')

          Text('当前为临时身份，继续注册可保留资料、关系与个性化偏好，让你的触感主题长期保存。')
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .lineHeight(22)
            .fontColor(this.getSecondaryTextColor())

          Button('升级为完整账号')
            .type(ButtonType.Normal)
            .backgroundColor(this.getProfileVisualStyle().primaryButton.backgroundColor)
            .fontColor(this.getProfileVisualStyle().primaryButton.textColor)
            .fontSize(DesignConstants.FONT.SIZE.MD)
            .fontWeight(FontWeight.Bold)
            .width('100%')
            .height(this.getProfileVisualStyle().primaryButton.height)
            .padding({
              left: this.getProfileVisualStyle().primaryButton.horizontalPadding,
              right: this.getProfileVisualStyle().primaryButton.horizontalPadding
            })
            .borderRadius(this.getProfileVisualStyle().primaryButton.borderRadius)
            .border({ width: this.getProfileVisualStyle().primaryButton.borderWidth, color: this.getProfileVisualStyle().primaryButton.borderColor })
            .shadow(this.getProfileVisualStyle().primaryButton.shadow)
            .onClick(() => {
              this.showUpgradeForm = true;
            })
        }
        .width('100%')
        .padding(DesignConstants.SPACING.LG)
      }
      .backgroundColor(this.getProfileVisualStyle().card.backgroundColor)
      .borderRadius(this.getProfileVisualStyle().card.borderRadius)
      .shadow(this.getProfileVisualStyle().card.shadow)
      .border({ width: 1, color: this.getSurfaceBorder() })
      .margin({ bottom: DesignConstants.SPACING.MD })

      ListItemGroup() {
        ListItem() {
          this.MenuItemContent('更新位置', AppIcon.LOCATION, () => this.updateLocation())
        }
        .backgroundColor(Color.Transparent)

        ListItem() {
          this.MenuItemContent('设置', AppIcon.SETTINGS, () => {
            AppRouter.push(PageNames.SETTINGS);
          })
        }
        .backgroundColor(Color.Transparent)

        if (this.isAdminUser()) {
          ListItem() {
            this.MenuItemContent('管理员工具台', AppIcon.SHIELD, () => this.openAdminPanel())
          }
          .backgroundColor(Color.Transparent)
        }

        ListItem() {
          this.MenuItemContent('退出登录', AppIcon.CLOSE, () => this.logout())
        }
        .backgroundColor(Color.Transparent)
      }
      .divider({ strokeWidth: 0.5, color: this.getSurfaceBorder(), startMargin: 52, endMargin: DesignConstants.SPACING.LG })
      .backgroundColor(this.getSurfaceBackground())
      .borderRadius(this.getProfileVisualStyle().card.borderRadius)
      .shadow(this.getProfileVisualStyle().card.shadow)
      .border({ width: 1, color: this.getSurfaceBorder() })
      .margin({ bottom: DesignConstants.SPACING.MD })
    }
    .width('100%')
    .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, top: DesignConstants.SPACING.SM, bottom: DesignConstants.SPACING.LG })
    .backgroundColor(this.getCurrentThemePalette().pageBackground)
    .backgroundBlurStyle(BlurStyle.Thin)
    .layoutWeight(1)
    .scrollBar(BarState.Auto)
  }

  // ==================== 注册用户界面 ====================
  @Builder
  private buildRegisteredUserView() {
    List() {
      ListItem() {
        if (this.isEditing) {
          this.EditFormContent()
        } else {
          this.UserInfoContent()
        }
      }
      .backgroundColor(this.getProfileVisualStyle().card.backgroundColor)
      .borderRadius(this.getProfileVisualStyle().card.borderRadius)
      .shadow(this.getProfileVisualStyle().card.shadow)
      .border({ width: this.getProfileVisualStyle().card.borderWidth, color: this.getProfileVisualStyle().card.borderColor })
      .padding(this.getProfileVisualStyle().card.padding)
      .margin({ bottom: DesignConstants.SPACING.MD })

      ListItem() {
        this.StatsContent()
      }
      .backgroundColor(this.getProfileVisualStyle().card.backgroundColor)
      .borderRadius(this.getProfileVisualStyle().card.borderRadius)
      .shadow(this.getProfileVisualStyle().card.shadow)
      .border({ width: this.getProfileVisualStyle().card.borderWidth, color: this.getProfileVisualStyle().card.borderColor })
      .padding(this.getProfileVisualStyle().card.padding)
      .margin({ bottom: DesignConstants.SPACING.MD })

      ListItemGroup() {
        ListItem() {
          this.MenuItemContent('更新位置', AppIcon.LOCATION, () => this.updateLocation())
        }
        .backgroundColor(Color.Transparent)

        ListItem() {
          this.MenuItemContent('设置', AppIcon.SETTINGS, () => {
            AppRouter.push(PageNames.SETTINGS);
          })
        }
        .backgroundColor(Color.Transparent)

        if (this.isAdminUser()) {
          ListItem() {
            this.MenuItemContent('管理员工具台', AppIcon.SHIELD, () => this.openAdminPanel())
          }
          .backgroundColor(Color.Transparent)
        }

        ListItem() {
          this.MenuItemContent('退出登录', AppIcon.CLOSE, () => this.logout())
        }
        .backgroundColor(Color.Transparent)
      }
      .divider({ strokeWidth: 0.5, color: this.getSurfaceBorder(), startMargin: 52, endMargin: DesignConstants.SPACING.LG })
      .backgroundColor(this.getSurfaceBackground())
      .borderRadius(this.getProfileVisualStyle().card.borderRadius)
      .shadow(this.getProfileVisualStyle().card.shadow)
      .border({ width: 1, color: this.getSurfaceBorder() })
      .margin({ bottom: DesignConstants.SPACING.MD })
    }
    .width('100%')
    .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, top: DesignConstants.SPACING.SM, bottom: DesignConstants.SPACING.LG })
    .backgroundColor(this.getCurrentThemePalette().pageBackground)
    .backgroundBlurStyle(BlurStyle.Thin)
    .layoutWeight(1)
    .scrollBar(BarState.Auto)
  }

  // ==================== 升级表单界面 ====================
  @Builder
  private buildUpgradeFormView() {
    Scroll() {
      Column({ space: DesignConstants.SPACING.LG }) {
        Text('升级为完整账号')
          .fontSize(DesignConstants.FONT.SIZE.XL)
          .fontWeight(FontWeight.Bold)
          .fontColor(this.getPrimaryTextColor())
          .alignSelf(ItemAlign.Start)

        Text('设置登录凭证后，你的临时账号将升级为完整账号，所有数据都会保留。')
          .fontSize(DesignConstants.FONT.SIZE.MD)
          .fontColor(this.getSecondaryTextColor())

        TextInput({ placeholder: '邮箱 / 用户名（登录凭证）' })
          .fontSize(DesignConstants.FONT.SIZE.LG)
          .padding(DesignConstants.SPACING.LG)
          .backgroundColor(this.getSurfaceSubtle())
          .borderRadius(this.getProfileVisualStyle().input.borderRadius)
          .type(InputType.Email)
          .onChange((value: string) => {
            this.upgradeLoginId = value;
          })
          .onBlur(() => KeyboardDismissManager.getInstance().dismiss())

        TextInput({ placeholder: '密码（至少8位）' })
          .fontSize(DesignConstants.FONT.SIZE.LG)
          .padding(DesignConstants.SPACING.LG)
          .backgroundColor(this.getSurfaceSubtle())
          .borderRadius(this.getProfileVisualStyle().input.borderRadius)
          .type(InputType.Password)
          .onChange((value: string) => {
            this.upgradePassword = value;
          })
          .onBlur(() => KeyboardDismissManager.getInstance().dismiss())

        TextInput({ placeholder: '确认密码' })
          .fontSize(DesignConstants.FONT.SIZE.LG)
          .padding(DesignConstants.SPACING.LG)
          .backgroundColor(this.getSurfaceSubtle())
          .borderRadius(this.getProfileVisualStyle().input.borderRadius)
          .type(InputType.Password)
          .onChange((value: string) => {
            this.upgradePasswordConfirm = value;
          })
          .onBlur(() => KeyboardDismissManager.getInstance().dismiss())

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
        SystemAvatar({
          avatarUrl: this.currentUser?.avatar_url || '',
          name: this.currentUser?.nickname || '',
          avatarSize: this.getProfileVisualStyle().identity.avatarSize,
          isOnline: this.currentUser?.presence_status === 'online',
          inset: true
        })

        Column({ space: DesignConstants.SPACING.SM }) {
          Text(this.currentUser?.nickname || '')
            .fontSize(this.getProfileVisualStyle().identity.titleFontSize)
            .fontWeight(FontWeight.Medium)
            .fontColor(this.getPrimaryTextColor())
            .textAlign(TextAlign.Center)
            .maxLines(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
        }
        .alignItems(HorizontalAlign.Center)
        .width('100%')

        if (this.userType === 'temp') {
          Text('临时身份')
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .fontColor(this.getWarningTextColor())
            .backgroundColor(this.getSurfaceSubtle())
            .borderRadius(DesignConstants.SHAPE.CHIP)
            .padding({ left: DesignConstants.SPACING.MD, right: DesignConstants.SPACING.MD, top: DesignConstants.SPACING.XS, bottom: DesignConstants.SPACING.XS })
        } else if (this.isAdminUser()) {
          Text('管理员')
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .fontColor(this.getAccentBackground())
            .backgroundColor(this.getSurfaceSubtle())
            .borderRadius(DesignConstants.SHAPE.CHIP)
            .padding({ left: DesignConstants.SPACING.MD, right: DesignConstants.SPACING.MD, top: DesignConstants.SPACING.XS, bottom: DesignConstants.SPACING.XS })
        }

        if (this.currentUser?.bio) {
          Text(this.currentUser.bio)
            .fontSize(DesignConstants.FONT.SIZE.MD)
            .lineHeight(24)
            .fontColor(this.getSecondaryTextColor())
            .textAlign(TextAlign.Center)
            .maxLines(3)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
        }
      }
      .width('100%')
      .alignItems(HorizontalAlign.Center)

      Button('编辑资料')
        .type(ButtonType.Normal)
        .width('100%')
        .height(this.getProfileVisualStyle().primaryButton.height)
        .padding({
          left: this.getProfileVisualStyle().primaryButton.horizontalPadding,
          right: this.getProfileVisualStyle().primaryButton.horizontalPadding
        })
        .backgroundColor(this.getProfileVisualStyle().primaryButton.backgroundColor)
        .fontColor(this.getProfileVisualStyle().primaryButton.textColor)
        .fontSize(DesignConstants.FONT.SIZE.XL)
        .borderRadius(this.getProfileVisualStyle().primaryButton.borderRadius)
        .shadow(this.getProfileVisualStyle().primaryButton.shadow)
        .border({ width: this.getProfileVisualStyle().primaryButton.borderWidth, color: this.getProfileVisualStyle().primaryButton.borderColor })
        .onClick(() => this.startEdit())

      Row({ space: DesignConstants.SPACING.MD }) {
        this.ProfileQuickAction(AppIcon.HEART, '关系', () => this.openRelationshipHub())
        this.ProfileQuickAction(AppIcon.STAR, '收藏', () => this.openMyShares())
        this.ProfileQuickAction(AppIcon.SETTINGS, '设置', () => this.openSettingsCenter())
      }
      .width('100%')

      Row({ space: DesignConstants.SPACING.MD }) {
        this.ProfileIntentCard('隐私偏好', '控制资料可见范围与互动设置', AppIcon.SHIELD, () => this.openPrivacySettings())
        this.ProfileIntentCard('安全设置', '管理密码、登录设备与账号安全', AppIcon.LOCK, () => this.openSafetySettings())
      }
      .width('100%')
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
            avatarSize: 80
          })
        }
        .type(ButtonType.Normal)
        .backgroundColor(Color.Transparent)
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

        TextInput({ placeholder: '请输入昵称', text: this.editNickname })
        .fontSize(DesignConstants.FONT.SIZE.LG)
        .height(48)
        .backgroundColor(this.getSurfaceSubtle())
        .borderRadius(DesignConstants.SHAPE.CHIP)
        .onChange((value: string) => {
          this.editNickname = value;
        })
        .onBlur(() => KeyboardDismissManager.getInstance().dismiss())

      TextArea({ placeholder: '请输入个人简介', text: this.editBio })
        .fontSize(DesignConstants.FONT.SIZE.MD)
        .backgroundColor(this.getSurfaceSubtle())
        .borderRadius(DesignConstants.SHAPE.CHIP)
        .height(80)
        .onChange((value: string) => {
          this.editBio = value;
        })

      Row({ space: DesignConstants.SPACING.MD }) {
        Button('取消')
          .type(ButtonType.Normal)
          .backgroundColor(Color.Transparent)
          .fontColor(this.getPrimaryTextColor())
          .layoutWeight(1)
          .borderRadius(this.getProfileVisualStyle().card.borderRadius)
          .border({ width: 1, color: this.getSurfaceBorder() })
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
    Column({ space: DesignConstants.SPACING.MD }) {
      Row({ space: DesignConstants.SPACING.SM }) {
        this.CompactStatItem(AppIcon.CAMERA, formatCount(this.myMomentsCount), '瞬间', () => this.openMyMoments())
        this.CompactStatItem(AppIcon.GROUP, formatCount(this.myGroupsCount), '群组', () => this.openMyGroups())
        this.CompactStatItem(AppIcon.PERSON, formatCount(this.myFollowingCount), '关注', () => this.openRelationshipHub())
        this.CompactStatItem(AppIcon.HEART, formatCount(this.myFollowersCount), '粉丝', () => this.openRelationshipHub())
      }
      .width('100%')
    }
    .width('100%')
  }

  @Builder
  private CompactStatItem(icon: AppIcon, value: string, label: string, onTap: () => void): void {
    Button() {
      Column({ space: DesignConstants.SPACING.XS }) {
        Text() {
          SymbolSpan($r(icon))
            .fontSize(16)
            .fontColor([this.getAccentBackground()])
        }
        Text(value)
          .fontSize(DesignConstants.FONT.SIZE.XL)
          .fontWeight(FontWeight.Bold)
          .fontColor(this.getPrimaryTextColor())
        Text(label)
          .fontSize(DesignConstants.FONT.SIZE.XS)
          .fontColor(this.getSecondaryTextColor())
      }
      .width('100%')
      .alignItems(HorizontalAlign.Center)
      .justifyContent(FlexAlign.Center)
    }
    .type(ButtonType.Normal)
    .backgroundColor(this.getSurfaceBackground())
    .border({ width: 1, color: this.getSurfaceBorder() })
    .borderRadius(DesignConstants.SHAPE.COMPACT_CARD)
    .shadow(this.getProfileVisualStyle().card.shadow)
    .padding({ top: DesignConstants.SPACING.MD, bottom: DesignConstants.SPACING.MD })
    .layoutWeight(1)
    .stateEffect(true)
    .onClick(onTap)
  }

  @Builder
  private ProfileQuickAction(icon: AppIcon, label: string, onTap: () => void): void {
    Button() {
      Column({ space: DesignConstants.SPACING.SM }) {
        Row() {
          Text() {
            SymbolSpan($r(icon))
              .fontSize(this.getProfileVisualStyle().quickAction.iconSize)
              .fontColor([this.getAccentBackground()])
          }
        }
        .width(this.getProfileVisualStyle().quickAction.iconWrapperSize)
        .height(this.getProfileVisualStyle().quickAction.iconWrapperSize)
        .borderRadius(DesignConstants.RADIUS.FULL)
        .backgroundColor(this.getSurfaceSubtle())
        .justifyContent(FlexAlign.Center)
        .alignItems(VerticalAlign.Center)

        Text(label)
          .fontSize(this.getProfileVisualStyle().quickAction.labelFontSize)
          .fontColor(this.getSecondaryTextColor())
      }
      .width('100%')
      .alignItems(HorizontalAlign.Center)
    }
    .type(ButtonType.Normal)
    .backgroundColor(this.getProfileVisualStyle().quickAction.backgroundColor)
    .borderRadius(this.getProfileVisualStyle().quickAction.borderRadius)
    .border({ width: this.getProfileVisualStyle().quickAction.borderWidth, color: this.getProfileVisualStyle().quickAction.borderColor })
    .shadow(this.getProfileVisualStyle().quickAction.shadow)
    .padding(this.getProfileVisualStyle().quickAction.padding)
    .layoutWeight(1)
    .onClick(onTap)
  }

  @Builder
  private ProfileIntentCard(title: string, description: string, icon: AppIcon, onTap: () => void): void {
    Button() {
      Column({ space: DesignConstants.SPACING.MD }) {
        Row() {
          Text() {
            SymbolSpan($r(icon))
              .fontSize(20)
              .fontColor([this.getAccentBackground()])
          }
        }
        .width(this.getProfileVisualStyle().intentCard.iconWrapperSize)
        .height(this.getProfileVisualStyle().intentCard.iconWrapperSize)
        .borderRadius(DesignConstants.RADIUS.FULL)
        .backgroundColor(this.getSurfaceSubtle())
        .justifyContent(FlexAlign.Center)
        .alignItems(VerticalAlign.Center)

        Text(title)
          .fontSize(this.getProfileVisualStyle().intentCard.titleFontSize)
          .fontWeight(FontWeight.Medium)
          .fontColor(this.getPrimaryTextColor())
          .width('100%')
          .textAlign(TextAlign.Start)

        Text(description)
          .fontSize(this.getProfileVisualStyle().intentCard.descriptionFontSize)
          .lineHeight(24)
          .fontColor(this.getSecondaryTextColor())
          .width('100%')
          .textAlign(TextAlign.Start)
      }
      .width('100%')
      .alignItems(HorizontalAlign.Start)
    }
    .type(ButtonType.Normal)
    .layoutWeight(1)
    .backgroundColor(this.getProfileVisualStyle().intentCard.backgroundColor)
    .borderRadius(this.getProfileVisualStyle().intentCard.borderRadius)
    .shadow(this.getProfileVisualStyle().intentCard.shadow)
    .border({ width: this.getProfileVisualStyle().intentCard.borderWidth, color: this.getProfileVisualStyle().intentCard.borderColor })
    .padding(this.getProfileVisualStyle().intentCard.padding)
    .height(this.getProfileVisualStyle().intentCard.minHeight)
    .onClick(onTap)
  }

  @Builder
  private MenuItemContent(title: string, icon: AppIcon, onTap: () => void): void {
    Row({ space: DesignConstants.SPACING.MD }) {
      Row() {
        Text() {
          SymbolSpan($r(icon))
            .fontSize(DesignConstants.FONT.SIZE.LG)
            .fontColor([this.getAccentBackground()])
        }
      }
      .width(40)
      .height(40)
      .borderRadius(DesignConstants.RADIUS.FULL)
      .backgroundColor(this.getSurfaceSubtle())
      .justifyContent(FlexAlign.Center)
      .alignItems(VerticalAlign.Center)

      Text(title)
        .fontSize(DesignConstants.FONT.SIZE.LG)
        .fontColor(this.getPrimaryTextColor())
        .layoutWeight(1)

      Text() {
        SymbolSpan($r(AppIcon.FORWARD))
          .fontSize(DesignConstants.FONT.SIZE.LG)
          .fontColor([this.getSecondaryTextColor()])
      }
    }
    .width('100%')
    .padding({
      left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG,
      top: DesignConstants.SPACING.MD + 2, bottom: DesignConstants.SPACING.MD + 2
    })
    .scale(this.pressedMenuItemKey === title ? { x: 0.985, y: 0.985 } : { x: 1, y: 1 })
    .animation({ duration: 100, curve: Curve.EaseOut })
    .onTouch((event: TouchEvent) => {
      if (event.type === TouchType.Down) {
        this.pressedMenuItemKey = title;
      } else if (event.type === TouchType.Up || event.type === TouchType.Cancel) {
        this.pressedMenuItemKey = '';
      }
    })
    .onClick(onTap)
  }

// ==================== 快速体验弹窗 ====================

  @Builder
  private buildQuickStartSheet() {
    Scroll() {
      Column({ space: DesignConstants.SPACING.LG }) {
        Text('快速体验')
          .fontSize(DesignConstants.FONT.SIZE.XL)
          .fontWeight(FontWeight.Bold)
          .fontColor(this.getPrimaryTextColor())

        Text('无需注册，输入昵称即可快速体验')
          .fontSize(DesignConstants.FONT.SIZE.MD)
          .fontColor(this.getSecondaryTextColor())

        TextInput({ placeholder: '请输入你的昵称' })
          .fontSize(DesignConstants.FONT.SIZE.LG)
          .padding(DesignConstants.SPACING.LG)
          .backgroundColor(this.getSurfaceSubtle())
          .borderRadius(this.getProfileVisualStyle().input.borderRadius)
          .enabled(!this.isQuickStarting)
          .onChange((value: string) => { this.quickNickname = value; })
          .onFocus(() => {
            this.isQuickInputFocused = true;
          })
          .onBlur(() => {
            this.isQuickInputFocused = false;
            KeyboardDismissManager.getInstance().dismiss();
          })

        Button(this.isQuickStarting ? '正在创建...' : '开始体验')
          .type(ButtonType.Normal)
          .backgroundColor(this.getProfileVisualStyle().primaryButton.backgroundColor)
          .fontColor(this.getProfileVisualStyle().primaryButton.textColor)
          .fontSize(DesignConstants.FONT.SIZE.LG)
          .width('100%')
          .height(this.getProfileVisualStyle().primaryButton.height)
          .borderRadius(this.getProfileVisualStyle().primaryButton.borderRadius)
          .shadow(this.getProfileVisualStyle().primaryButton.shadow)
          .enabled(!this.isQuickStarting)
          .opacity(this.isQuickStarting ? DesignConstants.OPACITY.LOADING : 1.0)
          .onClick(() => this.startQuickExperience())

        Text('临时用户可使用大部分功能，随时可升级为完整账号')
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .fontColor(this.getSecondaryTextColor())
          .textAlign(TextAlign.Center)
      }
      .width('100%')
      .padding({
        left: DesignConstants.SPACING.XXL,
        right: DesignConstants.SPACING.XXL,
        top: DesignConstants.SPACING.LG,
        bottom: SafeAreaUtils.resolveBottomPadding(this.safeAreaBottom, DesignConstants.SPACING.XXL, DesignConstants.SPACING.LG)
      })
    }
    .width('100%')
    .scrollBar(BarState.Off)
  }

  private async startQuickExperience(): Promise<void> {
    const nickname = this.quickNickname.trim();
    if (!nickname) {
      SystemToast.showWarning('请输入昵称', this.getUIContext());
      return;
    }
    if (!this.isValidNickname(nickname)) {
      SystemToast.showWarning('昵称长度需在1-30字符之间', this.getUIContext());
      return;
    }

    this.isQuickStarting = true;
    try {
      const request: GuestRequest = { nickname, client_type: 'app' };
      const response = await ApiClient.createTempUser(request);

      if (response.code === 0 && response.data) {
        const authData = response.data;
        await AuthAccountManager.upsertAndActivate(authData);
        await HttpUtil.setToken(authData.access_token);

        const basicUserInfo: MyProfile = {
          user_id: authData.user_id, nickname: '', avatar_url: '', user_type: 'temp',
          bio: '', tags: [], email: '', phone: '',
          location: { point: { latitude: 0, longitude: 0 }, location_name: '' },
          presence_status: 'online', last_online: '', created_at: new Date().toISOString(),
          stats: { moments_count: 0, friends_count: 0, followers_count: 0, following_count: 0, groups_count: 0 }
        };
        this.stateManager.updateUser(basicUserInfo);

        try {
          const profileResp = await ApiClient.getMyProfile();
          if (profileResp.code === 0 && profileResp.data) {
            this.stateManager.updateUser(profileResp.data);
          }
        } catch (error) {
          hilog.warn(DOMAIN_NUMBER, TAG, `获取用户资料失败: ${JSON.stringify(error)}`);
        }

        try {
          await this.webSocketService.connect(authData.access_token);
        } catch (error) {
          hilog.warn(DOMAIN_NUMBER, TAG, `WebSocket连接失败: ${JSON.stringify(error)}`);
        }

        this.showQuickStart = false;
        SystemToast.showSuccess('欢迎！', this.getUIContext());
      } else {
        SystemToast.showError(response.message || '创建失败', this.getUIContext());
      }
    } catch (error) {
      SystemToast.showError(error instanceof Error ? error.message : '创建失败', this.getUIContext());
    } finally {
      this.isQuickStarting = false;
    }
  }
}

@Builder
export function ProfilePageBuilder() {
  ProfilePage()
}
