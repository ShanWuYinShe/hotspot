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
import { SystemAvatar } from '../../../components/SystemAvatar';
import { SystemToast } from '../../../components/SystemToast';
import { SystemLoadingView } from '../../../components/SystemLoadingView';
import { MyProfile, UpdateLocationRequest, UpgradeAccountRequest, GuestRequest } from '../../../core/types/api';
import { AuthAccountManager } from '../../../core/state/AuthAccountManager';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { DesignConstants } from '../../../constants/DesignConstants';
import { AppIcon } from '../../../components/SystemIcon';
import { SystemPageHeader } from '../../../components/SystemPageHeader';
import { LoadingUtil } from '../../../core/utils/LoadingUtil';
import { NfcShareService } from '../../../core/services/NfcShareService';
import { SystemUiController } from '../../../core/utils/SystemUiController';

const DOMAIN_NUMBER: number = 0xFF03;
const TAG: string = 'ProfilePage';

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
  @State isLoading: boolean = false;
  @State myMomentsCount: number = 0;
  @State myGroupsCount: number = 0;
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

  private webSocketService = WebSocketService.getInstance();

  async aboutToAppear() {
    SystemUiController.useSurface({
      baseColor: DesignConstants.COLOR.PRIMARY_BG_LIGHT,      preferDarkContent: true
    });
    if (this.currentUser) {
      this.editNickname = this.currentUser.nickname;
      this.editBio = this.currentUser.bio || '';
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
    }
  }

  private startEdit(): void {
    if (this.currentUser) {
      this.editNickname = this.currentUser.nickname;
      this.editBio = this.currentUser.bio || '';
      this.isEditing = true;
    }
  }

  private cancelEdit(): void {
    this.isEditing = false;
  }

  private async saveEdit(): Promise<void> {
    if (!this.editNickname.trim()) {
      SystemToast.showWarning('昵称不能为空', this.getUIContext());
      return;
    }

    this.isLoading = true;
    try {
      await LoadingUtil.withMinimumTime(async () => {
        if (this.currentUser) {
          await ApiClient.updateProfile({
            nickname: this.editNickname,
            bio: this.editBio
          });

          const response = await ApiClient.getMyProfile();
          if (response.code === 0 && response.data) {
            this.stateManager.updateUser(response.data);
          }
        }

        this.isEditing = false;
        SystemToast.showSuccess('资料更新成功', this.getUIContext());
      }, LoadingUtil.DEFAULT_ACTION_MS);
    } catch (error) {
      SystemToast.showError('更新失败', this.getUIContext());
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
    if (!this.upgradeLoginId.trim()) {
      SystemToast.showWarning('请输入邮箱/用户名', this.getUIContext());
      return;
    }
    if (this.upgradePassword.length < 8) {
      SystemToast.showWarning('密码至少需要8个字符', this.getUIContext());
      return;
    }
    if (this.upgradePassword !== this.upgradePasswordConfirm) {
      SystemToast.showWarning('两次输入的密码不一致', this.getUIContext());
      return;
    }

    this.isUpgrading = true;
    try {
      const request: UpgradeAccountRequest = {
        login_id: this.upgradeLoginId.trim(),
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
              .fill($r('sys.color.ohos_id_color_component_normal'))
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
        .backgroundColor($r('sys.color.ohos_id_color_background'))
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
        .backgroundColor($r('sys.color.ohos_id_color_background'))
        .borderRadius(DesignConstants.RADIUS.LG)
        .padding(DesignConstants.SPACING.LG)

        Column({ space: 12 }) {
          // 快速体验按钮（弹窗模式）
          Button('快速体验（无需注册）')
            .type(ButtonType.Capsule)
            .width('100%')
            .height(48)
            .backgroundColor($r('sys.color.ohos_id_color_emphasize'))
            .fontColor(Color.White)
            .fontSize(16)
            .onClick(() => {
              this.showQuickStart = true;
            })

          // 登录/注册按钮
          Button('登录 / 注册')
            .type(ButtonType.Capsule)
            .width('100%')
            .height(48)
            .backgroundColor($r('sys.color.ohos_id_color_component_normal'))
            .fontColor($r('sys.color.ohos_id_color_text_primary'))
            .fontSize(16)
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
      backgroundColor: $r('sys.color.ohos_id_color_background')
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
            .type(ButtonType.Capsule)
            .backgroundColor($r('sys.color.ohos_id_color_emphasize'))
            .fontColor(Color.White)
            .fontSize(14)
            .width('100%')
            .height(40)
            .onClick(() => {
              this.showUpgradeForm = true;
            })
        }
        .width('100%')
        .padding(DesignConstants.SPACING.LG)
      }
      .backgroundColor($r('sys.color.ohos_id_color_background'))
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
      .backgroundColor($r('sys.color.ohos_id_color_background'))
      .borderRadius(DesignConstants.RADIUS.LG)
      .margin({ left: 16, right: 16 })
      .padding(DesignConstants.SPACING.LG)

      // 统计数据
      ListItem() {
        this.StatsContent()
      }
      .backgroundColor($r('sys.color.ohos_id_color_background'))
      .borderRadius(DesignConstants.RADIUS.LG)
      .margin({ left: 16, right: 16 })
      .padding(DesignConstants.SPACING.LG)

      // 菜单
      ListItemGroup() {
        ListItem() {
          this.MenuItemContent('更新位置', AppIcon.LOCATION, () => this.updateLocation())
        }
        .backgroundColor($r('sys.color.ohos_id_color_background'))

        ListItem() {
          this.MenuItemContent('设置', AppIcon.SETTINGS, () => {
            AppRouter.push(PageNames.SETTINGS);
          })
        }
        .backgroundColor($r('sys.color.ohos_id_color_background'))

        ListItem() {
          this.MenuItemContent('退出登录', AppIcon.CLOSE, () => this.logout())
        }
        .backgroundColor($r('sys.color.ohos_id_color_background'))
      }
      .borderRadius(DesignConstants.RADIUS.LG)
      .divider({ strokeWidth: 0.5, color: $r('sys.color.ohos_id_color_list_separator'), startMargin: 52, endMargin: DesignConstants.SPACING.LG })
      .margin({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, bottom: 24 })
    }
    .width('100%')
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
      .backgroundColor($r('sys.color.ohos_id_color_background'))
      .borderRadius(DesignConstants.RADIUS.LG)
      .margin({ left: 16, right: 16 })
      .padding(DesignConstants.SPACING.LG)

      // 统计数据卡片
      ListItem() {
        this.StatsContent()
      }
      .backgroundColor($r('sys.color.ohos_id_color_background'))
      .borderRadius(DesignConstants.RADIUS.LG)
      .margin({ left: 16, right: 16 })
      .padding(DesignConstants.SPACING.LG)

      // 菜单列表
      ListItemGroup() {
        ListItem() {
          this.MenuItemContent('更新位置', AppIcon.LOCATION, () => this.updateLocation())
        }
        .backgroundColor($r('sys.color.ohos_id_color_background'))

        ListItem() {
          this.MenuItemContent('设置', AppIcon.SETTINGS, () => {
            AppRouter.push(PageNames.SETTINGS);
          })
        }
        .backgroundColor($r('sys.color.ohos_id_color_background'))

        ListItem() {
          this.MenuItemContent('退出登录', AppIcon.CLOSE, () => this.logout())
        }
        .backgroundColor($r('sys.color.ohos_id_color_background'))
      }
      .borderRadius(DesignConstants.RADIUS.LG)
      .divider({ strokeWidth: 0.5, color: $r('sys.color.ohos_id_color_list_separator'), startMargin: 52, endMargin: DesignConstants.SPACING.LG })
      .margin({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, bottom: 24 })
    }
    .width('100%')
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
          .backgroundColor($r('sys.color.ohos_id_color_background'))
          .borderRadius(12)
          .type(InputType.Email)
          .onChange((value: string) => {
            this.upgradeLoginId = value;
          })

        TextInput({ placeholder: '密码（至少8位）' })
          .fontSize(16)
          .padding(16)
          .backgroundColor($r('sys.color.ohos_id_color_background'))
          .borderRadius(12)
          .type(InputType.Password)
          .onChange((value: string) => {
            this.upgradePassword = value;
          })

        TextInput({ placeholder: '确认密码' })
          .fontSize(16)
          .padding(16)
          .backgroundColor($r('sys.color.ohos_id_color_background'))
          .borderRadius(12)
          .type(InputType.Password)
          .onChange((value: string) => {
            this.upgradePasswordConfirm = value;
          })

        Row({ space: 12 }) {
          Button('取消')
            .type(ButtonType.Capsule)
            .backgroundColor($r('sys.color.ohos_id_color_component_normal'))
            .fontColor($r('sys.color.ohos_id_color_text_primary'))
            .layoutWeight(1)
            .height(44)
            .onClick(() => {
              this.showUpgradeForm = false;
            })

          Button(this.isUpgrading ? '升级中...' : '确认升级')
            .type(ButtonType.Capsule)
            .backgroundColor($r('sys.color.ohos_id_color_emphasize'))
            .fontColor(Color.White)
            .layoutWeight(1)
            .height(44)
            .enabled(!this.isUpgrading)
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
          .backgroundColor($r('sys.color.ohos_id_color_component_normal'))
          .borderRadius(8)
          .padding({ left: 8, right: 8, top: 2, bottom: 2 })
      }

      if (this.currentUser?.bio) {
        Text(this.currentUser.bio)
          .fontSize(DesignConstants.FONT.SIZE.MD)
          .fontColor($r('sys.color.ohos_id_color_text_secondary'))
          .textAlign(TextAlign.Center)
          .maxLines(3)
      }
    }
    .width('100%')
  }

  @Builder
  private EditFormContent(): void {
    Column({ space: 16 }) {
      SystemAvatar({
        avatarUrl: this.currentUser?.avatar_url || '',
        name: this.editNickname,
        avatarSize: 80
      })

      TextInput({ placeholder: '请输入昵称', text: this.editNickname })
        .fontSize(16)
        .height(48)
        .backgroundColor($r('sys.color.ohos_id_color_component_normal'))
        .borderRadius(8)
        .onChange((value: string) => {
          this.editNickname = value;
        })

      TextArea({ placeholder: '请输入个人简介', text: this.editBio })
        .fontSize(14)
        .backgroundColor($r('sys.color.ohos_id_color_component_normal'))
        .borderRadius(8)
        .height(80)
        .onChange((value: string) => {
          this.editBio = value;
        })

      Row({ space: 12 }) {
        Button('取消')
          .type(ButtonType.Capsule)
          .backgroundColor($r('sys.color.ohos_id_color_component_normal'))
          .fontColor($r('sys.color.ohos_id_color_text_primary'))
          .layoutWeight(1)
          .onClick(() => this.cancelEdit())

        Button('保存')
          .type(ButtonType.Capsule)
          .backgroundColor($r('sys.color.ohos_id_color_emphasize'))
          .fontColor(Color.White)
          .layoutWeight(1)
          .onClick(() => this.saveEdit())
      }
      .width('100%')
    }
    .width('100%')
  }

  @Builder
  private StatsContent(): void {
    Column({ space: 12 }) {
      Text('我的数据')
        .fontSize(16)
        .fontWeight(FontWeight.Medium)
        .fontColor($r('sys.color.ohos_id_color_text_primary'))
        .alignSelf(ItemAlign.Start)

      Row() {
        Column({ space: 4 }) {
          Text(this.myMomentsCount.toString())
            .fontSize(20)
            .fontWeight(FontWeight.Bold)
            .fontColor($r('sys.color.ohos_id_color_emphasize'))

          Text('我的瞬间')
            .fontSize(12)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
        }
        .layoutWeight(1)

        Column({ space: 4 }) {
          Text(this.myGroupsCount.toString())
            .fontSize(20)
            .fontWeight(FontWeight.Bold)
            .fontColor($r('sys.color.ohos_id_color_emphasize'))

          Text('加入群组')
            .fontSize(12)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
        }
        .layoutWeight(1)

        Column({ space: 4 }) {
          Text(this.currentLocation ? '已获取' : '未获取')
            .fontSize(14)
            .fontWeight(FontWeight.Bold)
            .fontColor(this.currentLocation
              ? $r('sys.color.ohos_id_color_emphasize')
              : $r('sys.color.ohos_id_color_warning'))

          Text('位置信息')
            .fontSize(12)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
        }
        .layoutWeight(1)
      }
      .width('100%')
    }
    .width('100%')
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
        .backgroundColor($r('app.color.bg_primary'))
        .borderRadius(DesignConstants.RADIUS.LG)
        .enabled(!this.isQuickStarting)
        .onChange((value: string) => { this.quickNickname = value; })

      Button(this.isQuickStarting ? '正在创建...' : '开始体验')
        .type(ButtonType.Capsule)
        .backgroundColor($r('sys.color.ohos_id_color_emphasize'))
        .fontColor(Color.White)
        .fontSize(DesignConstants.FONT.SIZE.LG)
        .width('100%')
        .height(48)
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
