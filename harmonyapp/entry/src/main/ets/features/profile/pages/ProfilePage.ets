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
import { AppRouter } from '../../../core/router/AppRouter';
import { PageNames } from '../../../constants/AppConstants';
import { MediaServiceManager } from '../../../core/media/MediaServiceManager';
import { SystemAvatar } from '../../../components/SystemAvatar';
import { SystemToast } from '../../../components/SystemToast';
import { SystemLoadingView } from '../../../components/SystemLoadingView';
import { MyProfile, UpdateLocationRequest, UpgradeAccountRequest, GuestRequest } from '../../../core/types/api';
import { AuthAccountManager } from '../../../core/state/AuthAccountManager';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { common } from '@kit.AbilityKit';
import { DesignConstants } from '../../../constants/DesignConstants';
import { AppIcon } from '../../../components/SystemIcon';
import { SystemPageHeader } from '../../../components/SystemPageHeader';
import { LoadingUtil } from '../../../core/utils/LoadingUtil';
import { NfcShareService } from '../../../core/services/NfcShareService';
import { SystemUiController } from '../../../core/utils/SystemUiController';

const DOMAIN_NUMBER: number = 0xFF03;
const TAG: string = 'ProfilePage';
const PASSWORD_POLICY_HINT: string = '密码需为8-128位且同时包含字母和数字';

@Preview
@Component
export struct ProfilePage {
  @StorageLink(KEY_CURRENT_USER) currentUser: MyProfile | null = null;
  @StorageLink(KEY_IS_LOGGED_IN) isLoggedIn: boolean = false;
  @StorageLink(KEY_USER_TYPE) userType: string = '';
  @StorageLink(KEY_CURRENT_LOCATION) currentLocation: LocationInfo | null = null;

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
  // 升级表单
  @State showUpgradeForm: boolean = false;
  @State upgradeLoginId: string = '';
  @State upgradePassword: string = '';
  @State upgradePasswordConfirm: string = '';
  @State isUpgrading: boolean = false;
  @State hasAdminPrivileges: boolean = false;
  @State pressedMenuItemKey: string = '';

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

  async aboutToAppear() {
    SystemUiController.useSurface({
      baseColor: DesignConstants.COLOR.PRIMARY_BG_LIGHT,      preferDarkContent: true
    });
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
    try {
      NfcShareService.getInstance().clearShareData();
    } catch (_) {
      // 静默忽略
    }
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

  private openSubscriptions(): void {
    AppRouter.push(PageNames.SUBSCRIPTIONS);
  }

  private openMyShares(): void {
    AppRouter.push(PageNames.MY_SHARES);
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
    Column() {
      SystemPageHeader({
        title: '个人中心',
        action: (this.isLoggedIn && this.currentUser && !this.isEditing && !this.showUpgradeForm)
          ? { label: '编辑', onClick: () => this.startEdit() }
          : undefined
      })

      if (this.isLoading) {
        SystemLoadingView({ message: '正在更新...' })
          .layoutWeight(1)
          .transition(TransitionEffect.OPACITY.animation({ duration: 300, curve: Curve.FastOutSlowIn }))
      } else if (!this.isLoggedIn) {
        // 匿名用户 - 游客模式
        Column() {
          this.buildGuestView()
        }
        .layoutWeight(1)
        .transition(TransitionEffect.OPACITY.animation({ duration: 300, curve: Curve.FastOutSlowIn }))
      } else if (this.showUpgradeForm) {
        // 升级表单
        Column() {
          this.buildUpgradeFormView()
        }
        .layoutWeight(1)
        .transition(TransitionEffect.OPACITY.animation({ duration: 300, curve: Curve.FastOutSlowIn }))
      } else if (this.userType === 'temp') {
        // 临时用户
        Column() {
          this.buildTempUserView()
        }
        .layoutWeight(1)
        .transition(TransitionEffect.OPACITY.animation({ duration: 300, curve: Curve.FastOutSlowIn }))
      } else {
        // 注册用户
        Column() {
          this.buildRegisteredUserView()
        }
        .layoutWeight(1)
        .transition(TransitionEffect.OPACITY.animation({ duration: 300, curve: Curve.FastOutSlowIn }))
      }
    }
    .width('100%')
    .height('100%')
    .backgroundColor($r('app.color.bg_primary'))
  }

  // ==================== 匿名用户 - 游客界面 ====================
  @Builder
  private buildGuestView() {
    Scroll() {
      Column({ space: 24 }) {
        // 头像占位 + 欢迎语
        Column({ space: 12 }) {
          Stack() {
            Circle()
              .width(80)
              .height(80)
              .fill($r('app.color.bg_secondary'))
            Text() {
              SymbolSpan($r(AppIcon.PERSON))
                .fontSize(36)
                .fontColor([$r('sys.color.ohos_id_color_text_secondary')])
            }
          }

          Text('游客模式')
            .fontSize(20)
            .fontWeight(FontWeight.Medium)
            .fontColor($r('sys.color.ohos_id_color_text_primary'))

          Text('无需登录即可浏览附近内容')
            .fontSize(14)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
        }
        .width('100%')
        .padding({ top: 32, bottom: 16 })

        // 当前可用功能
        Column({ space: 8 }) {
          Text('当前可用')
            .fontSize(14)
            .fontWeight(FontWeight.Medium)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
            .alignSelf(ItemAlign.Start)

          this.GuestFeatureItem('浏览附近群组', AppIcon.GROUP)
          this.GuestFeatureItem('查看附近瞬间', AppIcon.CAMERA)
          this.GuestFeatureItem('探索地图', AppIcon.MAP)
          this.GuestFeatureItem('搜索用户和内容', AppIcon.SEARCH)
        }
        .width('100%')
        .backgroundColor($r('app.color.bg_secondary'))
        .borderRadius(DesignConstants.RADIUS.LG)
        .padding(DesignConstants.SPACING.LG)

        // 登录解锁功能
        Column({ space: 8 }) {
          Text('登录后解锁')
            .fontSize(14)
            .fontWeight(FontWeight.Medium)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
            .alignSelf(ItemAlign.Start)

          this.GuestFeatureItem('发布瞬间、点赞互动', AppIcon.HEART)
          this.GuestFeatureItem('创建和加入群组', AppIcon.ADD)
          this.GuestFeatureItem('实时消息聊天', AppIcon.MESSAGE)
          this.GuestFeatureItem('关注好友、管理资料', AppIcon.PERSON)
        }
        .width('100%')
        .backgroundColor($r('app.color.bg_secondary'))
        .borderRadius(DesignConstants.RADIUS.LG)
        .padding(DesignConstants.SPACING.LG)

        Column({ space: 12 }) {
          // 快速体验按钮（弹窗模式）
          Button('快速体验（无需注册）')
            .type(ButtonType.Normal)
            .width('100%')
            .height(48)
            .backgroundColor($r('sys.color.ohos_id_color_emphasize'))
            .fontColor(Color.White)
            .fontSize(16)
            .borderRadius(DesignConstants.RADIUS.XL)
            .onClick(() => {
              this.showQuickStart = true;
            })

          // 登录/注册按钮
          Button('登录 / 注册')
            .type(ButtonType.Normal)
            .width('100%')
            .height(48)
            .backgroundColor($r('app.color.bg_secondary'))
            .fontColor($r('sys.color.ohos_id_color_text_primary'))
            .fontSize(16)
            .borderRadius(DesignConstants.RADIUS.XL)
            .onClick(() => {
              AppRouter.push(PageNames.LOGIN);
            })
        }

        // 设置入口
        Row({ space: 8 }) {
          Text() {
            SymbolSpan($r(AppIcon.SETTINGS))
              .fontSize(16)
              .fontColor([$r('sys.color.ohos_id_color_text_secondary')])
          }
          Text('设置')
            .fontSize(14)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
        }
        .justifyContent(FlexAlign.Center)
        .width('100%')
        .padding({ top: 8, bottom: 16 })
        .onClick(() => {
          AppRouter.push(PageNames.SETTINGS);
        })
      }
      .width('100%')
      .padding({ left: 16, right: 16 })
    }
    .layoutWeight(1)
    .bindSheet($$this.showQuickStart, this.buildQuickStartSheet(), {
      height: SheetSize.FIT_CONTENT,
      showClose: true,
      dragBar: true,
      backgroundColor: $r('app.color.bg_secondary')
    })
  }

  // ==================== 临时用户界面 ====================
  @Builder
  private buildTempUserView() {
    List({ space: 12 }) {
      // 升级提示卡片
      ListItem() {
        Column({ space: 12 }) {
          Row({ space: 8 }) {
            Text() {
              SymbolSpan($r(AppIcon.INFO))
                .fontSize(16)
                .fontColor([$r('sys.color.ohos_id_color_warning')])
            }
            Text('临时用户')
              .fontSize(14)
              .fontWeight(FontWeight.Medium)
              .fontColor($r('sys.color.ohos_id_color_warning'))
          }
          .width('100%')

          Text('当前为临时用户，数据可能不会永久保留。注册账号可保留所有数据并解锁完整功能。')
            .fontSize(13)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))

          Button('升级为完整账号')
            .type(ButtonType.Normal)
            .backgroundColor($r('sys.color.ohos_id_color_emphasize'))
            .fontColor(Color.White)
            .fontSize(14)
            .width('100%')
            .height(40)
            .borderRadius(DesignConstants.RADIUS.XL)
            .onClick(() => {
              this.showUpgradeForm = true;
            })
        }
        .width('100%')
        .padding(DesignConstants.SPACING.LG)
      }
      .backgroundColor($r('app.color.bg_secondary'))
      .borderRadius(DesignConstants.RADIUS.LG)
      .margin({ left: 16, right: 16 })

      // 用户信息卡片
      ListItem() {
        if (this.isEditing) {
          this.EditFormContent()
        } else {
          this.UserInfoContent()
        }
      }
      .backgroundColor($r('app.color.bg_secondary'))
      .borderRadius(DesignConstants.RADIUS.LG)
      .margin({ left: 16, right: 16 })
      .padding(DesignConstants.SPACING.LG)

      // 统计数据
      ListItem() {
        this.StatsContent()
      }
      .backgroundColor($r('app.color.bg_secondary'))
      .borderRadius(DesignConstants.RADIUS.LG)
      .margin({ left: 16, right: 16 })
      .padding(DesignConstants.SPACING.LG)

      // 快捷操作
      ListItem() {
        this.QuickActionsSection()
      }
      .backgroundColor($r('app.color.bg_secondary'))
      .borderRadius(DesignConstants.RADIUS.LG)
      .margin({ left: 16, right: 16 })
      .padding(DesignConstants.SPACING.LG)

      // 菜单
      ListItemGroup() {
        ListItem() {
          this.MenuItemContent('更新位置', AppIcon.LOCATION, () => this.updateLocation())
        }
        .backgroundColor($r('app.color.bg_secondary'))

        ListItem() {
          this.MenuItemContent('设置', AppIcon.SETTINGS, () => {
            AppRouter.push(PageNames.SETTINGS);
          })
        }
        .backgroundColor($r('app.color.bg_secondary'))

        if (this.isAdminUser()) {
          ListItem() {
            this.MenuItemContent('管理员工具台', AppIcon.SHIELD, () => this.openAdminPanel())
          }
          .backgroundColor($r('app.color.bg_secondary'))
        }

        ListItem() {
          this.MenuItemContent('退出登录', AppIcon.CLOSE, () => this.logout())
        }
        .backgroundColor($r('app.color.bg_secondary'))
      }
      .borderRadius(DesignConstants.RADIUS.LG)
      .divider({ strokeWidth: 0.5, color: $r('sys.color.ohos_id_color_list_separator'), startMargin: 52, endMargin: DesignConstants.SPACING.LG })
      .margin({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, bottom: 24 })
    }
    .width('100%')
    .padding({ top: DesignConstants.SPACING.MD, bottom: DesignConstants.SPACING.LG })
    .layoutWeight(1)
    .scrollBar(BarState.Auto)
  }

  // ==================== 注册用户界面 ====================
  @Builder
  private buildRegisteredUserView() {
    List({ space: 12 }) {
      // 用户信息卡片
      ListItem() {
        if (this.isEditing) {
          this.EditFormContent()
        } else {
          this.UserInfoContent()
        }
      }
      .backgroundColor($r('app.color.bg_secondary'))
      .borderRadius(DesignConstants.RADIUS.LG)
      .margin({ left: 16, right: 16 })
      .padding(DesignConstants.SPACING.LG)

      // 统计数据卡片
      ListItem() {
        this.StatsContent()
      }
      .backgroundColor($r('app.color.bg_secondary'))
      .borderRadius(DesignConstants.RADIUS.LG)
      .margin({ left: 16, right: 16 })
      .padding(DesignConstants.SPACING.LG)

      // 快捷操作
      ListItem() {
        this.QuickActionsSection()
      }
      .backgroundColor($r('app.color.bg_secondary'))
      .borderRadius(DesignConstants.RADIUS.LG)
      .margin({ left: 16, right: 16 })
      .padding(DesignConstants.SPACING.LG)

      // 菜单列表
      ListItemGroup() {
        ListItem() {
          this.MenuItemContent('更新位置', AppIcon.LOCATION, () => this.updateLocation())
        }
        .backgroundColor($r('app.color.bg_secondary'))

        ListItem() {
          this.MenuItemContent('设置', AppIcon.SETTINGS, () => {
            AppRouter.push(PageNames.SETTINGS);
          })
        }
        .backgroundColor($r('app.color.bg_secondary'))

        if (this.isAdminUser()) {
          ListItem() {
            this.MenuItemContent('管理员工具台', AppIcon.SHIELD, () => this.openAdminPanel())
          }
          .backgroundColor($r('app.color.bg_secondary'))
        }

        ListItem() {
          this.MenuItemContent('退出登录', AppIcon.CLOSE, () => this.logout())
        }
        .backgroundColor($r('app.color.bg_secondary'))
      }
      .borderRadius(DesignConstants.RADIUS.LG)
      .divider({ strokeWidth: 0.5, color: $r('sys.color.ohos_id_color_list_separator'), startMargin: 52, endMargin: DesignConstants.SPACING.LG })
      .margin({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, bottom: 24 })
    }
    .width('100%')
    .padding({ top: DesignConstants.SPACING.MD, bottom: DesignConstants.SPACING.LG })
    .layoutWeight(1)
    .scrollBar(BarState.Auto)
  }

  // ==================== 升级表单界面 ====================
  @Builder
  private buildUpgradeFormView() {
    Scroll() {
      Column({ space: 16 }) {
        Text('升级为完整账号')
          .fontSize(20)
          .fontWeight(FontWeight.Bold)
          .fontColor($r('sys.color.ohos_id_color_text_primary'))
          .alignSelf(ItemAlign.Start)

        Text('设置登录凭证后，你的临时账号将升级为完整账号，所有数据都会保留。')
          .fontSize(14)
          .fontColor($r('sys.color.ohos_id_color_text_secondary'))

        TextInput({ placeholder: '邮箱 / 用户名（登录凭证）' })
          .fontSize(16)
          .padding(16)
          .backgroundColor($r('app.color.bg_tertiary'))
          .borderRadius(DesignConstants.RADIUS.LG)
          .type(InputType.Email)
          .onChange((value: string) => {
            this.upgradeLoginId = value;
          })

        TextInput({ placeholder: '密码（至少8位）' })
          .fontSize(16)
          .padding(16)
          .backgroundColor($r('app.color.bg_tertiary'))
          .borderRadius(DesignConstants.RADIUS.LG)
          .type(InputType.Password)
          .onChange((value: string) => {
            this.upgradePassword = value;
          })

        TextInput({ placeholder: '确认密码' })
          .fontSize(16)
          .padding(16)
          .backgroundColor($r('app.color.bg_tertiary'))
          .borderRadius(DesignConstants.RADIUS.LG)
          .type(InputType.Password)
          .onChange((value: string) => {
            this.upgradePasswordConfirm = value;
          })

        Row({ space: 12 }) {
          Button('取消')
            .type(ButtonType.Normal)
            .backgroundColor($r('app.color.bg_secondary'))
            .fontColor($r('sys.color.ohos_id_color_text_primary'))
            .layoutWeight(1)
            .height(44)
            .borderRadius(DesignConstants.RADIUS.XL)
            .enabled(!this.isUpgrading)
            .opacity(this.isUpgrading ? 0.58 : 1)
            .onClick(() => {
              this.showUpgradeForm = false;
            })

          Button(this.isUpgrading ? '升级中...' : '确认升级')
            .type(ButtonType.Normal)
            .backgroundColor($r('sys.color.ohos_id_color_emphasize'))
            .fontColor(Color.White)
            .layoutWeight(1)
            .height(44)
            .borderRadius(DesignConstants.RADIUS.XL)
            .enabled(!this.isUpgrading)
            .opacity(this.isUpgrading ? 0.58 : 1)
            .onClick(() => this.upgradeAccount())
        }
        .width('100%')
      }
      .width('100%')
      .padding({ left: 32, right: 32, top: 24 })
    }
    .layoutWeight(1)
  }

  // ==================== 公共 Builder ====================

  @Builder
  private UserInfoContent(): void {
    Column({ space: 16 }) {
      SystemAvatar({
        avatarUrl: this.currentUser?.avatar_url || '',
        name: this.currentUser?.nickname || '',
        avatarSize: 80,
        isOnline: this.currentUser?.presence_status === 'online'
      })

      Text(this.currentUser?.nickname || '')
        .fontSize(DesignConstants.FONT.SIZE.XL)
        .fontWeight(FontWeight.Medium)
        .fontColor($r('sys.color.ohos_id_color_text_primary'))

      if (this.userType === 'temp') {
        Text('临时用户')
          .fontSize(12)
          .fontColor($r('sys.color.ohos_id_color_warning'))
          .backgroundColor($r('app.color.bg_tertiary'))
          .borderRadius(DesignConstants.RADIUS.MD)
          .padding({ left: 8, right: 8, top: 2, bottom: 2 })
      } else if (this.isAdminUser()) {
        Text('管理员')
          .fontSize(12)
          .fontColor($r('sys.color.ohos_id_color_emphasize'))
          .backgroundColor($r('app.color.bg_tertiary'))
          .borderRadius(DesignConstants.RADIUS.MD)
          .padding({ left: 8, right: 8, top: 2, bottom: 2 })
      }

      if (this.currentUser?.bio) {
        Text(this.currentUser.bio)
          .fontSize(DesignConstants.FONT.SIZE.MD)
          .fontColor($r('sys.color.ohos_id_color_text_secondary'))
          .textAlign(TextAlign.Center)
          .maxLines(3)
          .textOverflow({ overflow: TextOverflow.Ellipsis })
      }
    }
    .width('100%')
  }

  @Builder
  private EditFormContent(): void {
    Column({ space: 16 }) {
      Column({ space: 6 }) {
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
        .opacity(this.isLoading || this.isAvatarUploading ? 0.58 : 1)
        .onClick(() => {
          this.selectAvatarForEdit();
        })

        if (this.isAvatarUploading) {
          Text('头像上传中...')
            .fontSize(12)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
        } else {
          Text('点击头像更换')
            .fontSize(12)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
        }
      }

      TextInput({ placeholder: '请输入昵称', text: this.editNickname })
        .fontSize(16)
        .height(48)
        .backgroundColor($r('app.color.bg_tertiary'))
        .borderRadius(DesignConstants.RADIUS.MD)
        .onChange((value: string) => {
          this.editNickname = value;
        })

      TextArea({ placeholder: '请输入个人简介', text: this.editBio })
        .fontSize(14)
        .backgroundColor($r('app.color.bg_tertiary'))
        .borderRadius(DesignConstants.RADIUS.MD)
        .height(80)
        .onChange((value: string) => {
          this.editBio = value;
        })

      Row({ space: 12 }) {
        Button('取消')
          .type(ButtonType.Normal)
          .backgroundColor($r('app.color.bg_secondary'))
          .fontColor($r('sys.color.ohos_id_color_text_primary'))
          .layoutWeight(1)
          .borderRadius(DesignConstants.RADIUS.XL)
          .enabled(!this.isLoading && !this.isAvatarUploading)
          .opacity(this.isLoading || this.isAvatarUploading ? 0.58 : 1)
          .onClick(() => this.cancelEdit())

        Button(this.isLoading ? '保存中...' : '保存')
          .type(ButtonType.Normal)
          .backgroundColor($r('sys.color.ohos_id_color_emphasize'))
          .fontColor(Color.White)
          .layoutWeight(1)
          .borderRadius(DesignConstants.RADIUS.XL)
          .enabled(!this.isLoading && !this.isAvatarUploading)
          .opacity(this.isLoading || this.isAvatarUploading ? 0.58 : 1)
          .onClick(() => this.saveEdit())
      }
      .width('100%')
    }
    .width('100%')
  }

  @Builder
  private StatsContent(): void {
    Column({ space: 16 }) {
      Text('我的数据')
        .fontSize(16)
        .fontWeight(FontWeight.Medium)
        .fontColor($r('sys.color.ohos_id_color_text_primary'))
        .alignSelf(ItemAlign.Start)

      Row({ space: 12 }) {
        this.StatCard('我的瞬间', this.myMomentsCount.toString(), '浏览和管理动态', AppIcon.CAMERA, () => this.openMyMoments())
        this.StatCard('加入群组', this.myGroupsCount.toString(), '查看我加入的圈子', AppIcon.GROUP, () => this.openMyGroups())
      }
      .width('100%')

      Row({ space: 12 }) {
        this.StatCard('关注', this.myFollowingCount.toString(), '我关注的用户', AppIcon.PERSON, () => this.openSearchPeople())
        this.StatCard('粉丝', this.myFollowersCount.toString(), '谁关注了我', AppIcon.HEART, () => this.openSearchPeople())
      }
      .width('100%')
    }
    .width('100%')
  }

  @Builder
  private StatCard(title: string, value: string, description: string, icon: AppIcon, onTap: () => void): void {
    Button() {
      Column({ space: 6 }) {
        Row({ space: 6 }) {
          Text() {
            SymbolSpan($r(icon))
              .fontSize(18)
              .fontColor([$r('sys.color.ohos_id_color_emphasize')])
          }
          Text(title)
            .fontSize(14)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
            .layoutWeight(1)
        }
        Text(value)
          .fontSize(22)
          .fontWeight(FontWeight.Bold)
          .fontColor($r('sys.color.ohos_id_color_text_primary'))
        Text(description)
          .fontSize(12)
          .fontColor($r('sys.color.ohos_id_color_text_secondary'))
      }
      .width('100%')
    }
    .type(ButtonType.Normal)
    .backgroundColor(Color.Transparent)
    .borderRadius(DesignConstants.RADIUS.XL)
    .padding(DesignConstants.SPACING.MD)
    .layoutWeight(1)
    .stateEffect(true)
    .onClick(onTap)
  }

  @Builder
  private GuestFeatureItem(label: string, icon: AppIcon): void {
    Row({ space: 12 }) {
      Text() {
        SymbolSpan($r(icon))
          .fontSize(16)
          .fontColor([$r('sys.color.ohos_id_color_emphasize')])
      }
      Text(label)
        .fontSize(14)
        .fontColor($r('sys.color.ohos_id_color_text_primary'))
    }
    .width('100%')
    .padding({ top: 8, bottom: 8 })
  }

  @Builder
  private MenuItemContent(title: string, icon: string, onTap: () => void): void {
    Row({ space: DesignConstants.SPACING.MD }) {
      Text() {
        SymbolSpan($r(icon))
          .fontSize(20)
          .fontColor([$r('sys.color.ohos_id_color_emphasize')])
      }
      .width(24)
      .textAlign(TextAlign.Center)

      Text(title)
        .fontSize(DesignConstants.FONT.SIZE.LG)
        .fontColor($r('sys.color.ohos_id_color_text_primary'))
        .layoutWeight(1)

      Text() {
        SymbolSpan($r(AppIcon.FORWARD))
          .fontSize(16)
          .fontColor([$r('sys.color.ohos_id_color_text_secondary')])
      }
    }
    .width('100%')
    .padding({
      left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG,
      top: 14, bottom: 14
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

    @Builder
  private QuickActionsSection(): void {
    Column({ space: 12 }) {
      Text('快速操作')
        .fontSize(16)
        .fontWeight(FontWeight.Medium)
        .fontColor($r('sys.color.ohos_id_color_text_primary'))
        .alignSelf(ItemAlign.Start)

      this.QuickActionItem('我的分享', '查看并管理我的分享链接', AppIcon.SHARE, () => this.openMyShares())
      this.QuickActionItem('发现周边用户', '查看附近的人并发起互动', AppIcon.PERSON, () => this.openSearchPeople())
      this.QuickActionItem('浏览附近群组', '加入或管理兴趣圈子', AppIcon.GROUP, () => this.openMyGroups())
      this.QuickActionItem('查看附近动态', '浏览我的瞬间和周边动态', AppIcon.CAMERA, () => this.openMyMoments())
      if (this.isAdminUser()) {
        this.QuickActionItem('管理员工具台', '查看管理员专属工具入口', AppIcon.SHIELD, () => this.openAdminPanel())
      }
    }
    .width('100%')
  }

  @Builder
  private QuickActionItem(title: string, description: string, icon: AppIcon, onTap: () => void): void {
    Button() {
      Row({ space: 12 }) {
        Stack() {
          Circle()
            .width(40)
            .height(40)
            .fill($r('app.color.bg_tertiary'))
          Text() {
            SymbolSpan($r(icon))
              .fontSize(18)
              .fontColor([$r('sys.color.ohos_id_color_text_primary')])
          }
          .align(Alignment.Center)
        }
        .width(40)
        .height(40)

        Column({ space: 4 }) {
          Text(title)
            .fontSize(14)
            .fontWeight(FontWeight.Medium)
            .fontColor($r('sys.color.ohos_id_color_text_primary'))
          Text(description)
            .fontSize(12)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
        }
        .layoutWeight(1)

        Text() {
          SymbolSpan($r(AppIcon.FORWARD))
            .fontSize(16)
            .fontColor([$r('sys.color.ohos_id_color_text_secondary')])
        }
      }
      .width('100%')
    }
    .type(ButtonType.Normal)
    .backgroundColor(Color.Transparent)
    .borderRadius(DesignConstants.RADIUS.XL)
    .padding({
      left: DesignConstants.SPACING.MD,
      right: DesignConstants.SPACING.MD,
      top: DesignConstants.SPACING.MD,
      bottom: DesignConstants.SPACING.MD
    })
    .stateEffect(true)
    .onClick(onTap)
  }

// ==================== 快速体验弹窗 ====================

  @Builder
  private buildQuickStartSheet() {
    Column({ space: DesignConstants.SPACING.LG }) {
      Text('快速体验')
        .fontSize(DesignConstants.FONT.SIZE.XL)
        .fontWeight(FontWeight.Bold)
        .fontColor($r('sys.color.ohos_id_color_text_primary'))

      Text('无需注册，输入昵称即可快速体验')
        .fontSize(DesignConstants.FONT.SIZE.MD)
        .fontColor($r('sys.color.ohos_id_color_text_secondary'))

      TextInput({ placeholder: '请输入你的昵称' })
        .fontSize(DesignConstants.FONT.SIZE.LG)
        .padding(DesignConstants.SPACING.LG)
        .backgroundColor($r('app.color.bg_tertiary'))
        .borderRadius(DesignConstants.RADIUS.LG)
        .enabled(!this.isQuickStarting)
        .onChange((value: string) => { this.quickNickname = value; })

      Button(this.isQuickStarting ? '正在创建...' : '开始体验')
        .type(ButtonType.Normal)
        .backgroundColor($r('sys.color.ohos_id_color_emphasize'))
        .fontColor(Color.White)
        .fontSize(DesignConstants.FONT.SIZE.LG)
        .width('100%')
        .height(48)
        .borderRadius(DesignConstants.RADIUS.XL)
        .enabled(!this.isQuickStarting)
        .opacity(this.isQuickStarting ? 0.6 : 1.0)
        .onClick(() => this.startQuickExperience())

      Text('临时用户可使用大部分功能，随时可升级为完整账号')
        .fontSize(DesignConstants.FONT.SIZE.SM)
        .fontColor($r('sys.color.ohos_id_color_text_secondary'))
        .textAlign(TextAlign.Center)
    }
    .width('100%')
    .padding({
      left: DesignConstants.SPACING.XXL,
      right: DesignConstants.SPACING.XXL,
      top: DesignConstants.SPACING.LG,
      bottom: DesignConstants.SPACING.XXL
    })
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
  NavDestination() {
    ProfilePage()
  }
  .hideTitleBar(true)
}
