/**
 * 登录/注册页面
 * 两种模式：登录、注册
 * 认证成功后获取完整 profile（含 user_type）
 */

import { GlobalStateManager } from '../../../core/state/GlobalStateManager';
import { ApiClient } from '../../../core/api/ApiClient';
import { HttpUtil } from '../../../core/network/HttpUtil';
import { WebSocketService } from '../../../core/network/WebSocketService';
import { AppRouter, ChatRouteParam } from '../../../core/router/AppRouter';
import { PageNames } from '../../../constants/AppConstants';
import { SystemLoadingView } from '../../../components/SystemLoadingView';
import { SystemToast } from '../../../components/SystemToast';
import { PasswordSecurityUtil } from '../../../core/security/PasswordSecurityUtil';
import { AuthData, MyProfile, LoginRequest, RegisterRequest } from '../../../core/types/api';
import { AuthAccountManager } from '../../../core/state/AuthAccountManager';
import { getErrorMessage } from '../../../core/network/ApiErrorUtil';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { DesignConstants } from '../../../constants/DesignConstants';
import { LoadingUtil } from '../../../core/utils/LoadingUtil';
import { AppIcon } from '../../../components/SystemIcon';
import { SystemUiController } from '../../../core/utils/SystemUiController';
import { KeyboardDismissManager } from '../../../core/utils/KeyboardDismissManager';
import { SafeAreaUtils } from '../../../core/utils/SafeAreaUtils';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { KEY_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';

const DOMAIN_NUMBER: number = 0xFF02;
const TAG: string = 'LoginPage';
const PASSWORD_POLICY_HINT: string = '密码需为8-128位且同时包含字母和数字';

@Preview
@Component
export struct LoginPage {
  pathStack: NavPathStack = new NavPathStack();
  @StorageLink('isLoggedIn') isLoggedIn: boolean = false;
  @StorageLink('safeAreaTop') safeAreaTop: number = 0;
  @StorageLink('safeAreaBottom') safeAreaBottom: number = 0;
  @StorageLink(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'default_clean_light';

  @State loginId: string = '';
  @State password: string = '';
  @State registerNickname: string = '';
  @State registerLoginId: string = '';
  @State registerPassword: string = '';
  @State registerPasswordConfirm: string = '';
  @State isLoading: boolean = false;
  @State loadingStage: string = '';
  @State loadingProgress: number = 0;
  @State loginMode: 'login' | 'register' = 'login';
  @State redirectUrl: string = '';
  @State redirectAction: string = '';
  @State restoreAction: string = '';
  @State restoreTargetId: string = '';
  @State restoreTitle: string = '';

  private stateManager = GlobalStateManager.getInstance();
  private webSocketService = WebSocketService.getInstance();
  private isDestroyed: boolean = false;

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

  private getHeaderTopPadding(): number {
    return (this.safeAreaTop > 0 ? this.safeAreaTop : DesignConstants.SPACING.LG) + DesignConstants.SPACING.SM;
  }

  aboutToAppear(): void {
    SystemUiController.syncWithCurrentTheme();
  }

  aboutToDisappear(): void {
    this.isDestroyed = true;
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

  build() {
    NavDestination() {
      Column() {
      if (this.isLoading) {
        SystemLoadingView({
          message: this.loadingStage || '加载中...',
          progressType: 'determinate',
          showProgress: true,
          externalProgress: this.loadingProgress
        })
          .transition(
            TransitionEffect.OPACITY.animation({ duration: 320, curve: Curve.FastOutSlowIn })
              .combine(TransitionEffect.move(TransitionEdge.BOTTOM).animation({ duration: 320, curve: Curve.FastOutSlowIn }))
          )
      } else {
        Column() {
          // 顶部导航栏（固定）- 使用 SafeArea 避免与状态栏重叠
          Row() {
            Text() {
              SymbolSpan($r(AppIcon.BACK))
                .fontSize(DesignConstants.FONT.SIZE.XXL)
                .fontColor([this.getCurrentThemePalette().textPrimary])
            }
            .onClick(() => {
              AppRouter.pop();
            })
            .width(48)
            .height(48)
            .textAlign(TextAlign.Center)

            Blank()
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

          // 标题区域（固定）
          Column({ space: DesignConstants.SPACING.SM }) {
            Image($r('app.media.foreground'))
              .width(72)
              .height(72)
              .objectFit(ImageFit.Contain)

            Text($r('app.string.app_name'))
              .fontSize(28)
              .fontWeight(FontWeight.Bold)
              .fontColor(this.getCurrentThemePalette().textPrimary)

            Text($r('app.string.app_description'))
              .fontSize(DesignConstants.FONT.SIZE.LG)
              .fontColor(this.getCurrentThemePalette().textSecondary)
          }
          .width('100%')
          .alignItems(HorizontalAlign.Center)
          .padding({ left: DesignConstants.SPACING.XXL, right: DesignConstants.SPACING.XXL,
            bottom: DesignConstants.SPACING.LG })

          // 模式切换标签（固定）
          Row({ space: DesignConstants.SPACING.SM }) {
            this.ModeTab('登录', 'login')
            this.ModeTab('注册', 'register')
          }
          .backgroundColor(this.getSurfaceSubtle())
          .borderRadius(DesignConstants.RADIUS.XXL)
          .border({ width: 1, color: this.getSurfaceBorder() })
          .padding(3)
          .margin({ left: DesignConstants.SPACING.XXL, right: DesignConstants.SPACING.XXL })

          // 表单区域（可滚动，凸起卡片包裹）
          Scroll() {
            Column() {
              Column() {
                if (this.loginMode === 'login') {
                  this.buildLoginForm()
                } else {
                  this.buildRegisterForm()
                }
              }
              .width('100%')
              .padding(DesignConstants.SPACING.LG)
              .backgroundColor(this.getSurfaceBackground())
              .borderRadius(DesignConstants.RADIUS.XXL)
              .shadow(this.getCurrentThemePalette().elevatedShadow)
              .border({ width: 1, color: this.getSurfaceBorder() })
              .margin({ bottom: 14 })
            }
            .width('100%')
            .padding({ left: DesignConstants.SPACING.XXL, right: DesignConstants.SPACING.XXL,
              top: DesignConstants.SPACING.XL,
              bottom: SafeAreaUtils.resolveBottomPadding(this.safeAreaBottom, DesignConstants.SPACING.XXL, 0) })
          }
          .align(Alignment.Top)
          .layoutWeight(1)
          .scrollBar(BarState.Off)
        }
        .width('100%')
        .height('100%')
        .transition(
          TransitionEffect.OPACITY.animation({ duration: 320, curve: Curve.FastOutSlowIn })
            .combine(TransitionEffect.move(TransitionEdge.BOTTOM).animation({ duration: 320, curve: Curve.FastOutSlowIn }))
        )
      }
    }
    .width('100%')
    .height('100%')
    .backgroundColor(this.getCurrentThemePalette().pageBackground)
    }
    .hideTitleBar(true)
    .onReady((context: NavDestinationContext) => {
      this.pathStack = context.pathStack;
      const params = context.pathInfo?.param as Record<string, Object> | undefined;
      if (params) {
        const redirect = params.redirectUrl as string;
        if (redirect) {
          this.redirectUrl = redirect;
        }
        const action = params.action as string;
        if (action) {
          this.redirectAction = action;
        }
        const restoreAction = params.restoreAction as string;
        if (restoreAction) {
          this.restoreAction = restoreAction;
        }
        const restoreTargetId = params.restoreTargetId as string;
        if (restoreTargetId) {
          this.restoreTargetId = restoreTargetId;
        }
        const restoreTitle = params.restoreTitle as string;
        if (restoreTitle) {
          this.restoreTitle = restoreTitle;
        }
      }
    })
  }

  @Builder
  private ModeTab(label: string, mode: string) {
    Text(label)
      .fontSize(DesignConstants.FONT.SIZE.MD)
      .fontWeight(this.loginMode === mode ? FontWeight.Bold : FontWeight.Medium)
      .fontColor(this.loginMode === mode ? this.getCurrentThemePalette().accentText : this.getCurrentThemePalette().buttonSecondaryText)
      .backgroundColor(this.loginMode === mode ? this.getCurrentThemePalette().accentBackground : Color.Transparent)
      .borderRadius(DesignConstants.RADIUS.XL)
      .height(36)
      .layoutWeight(1)
      .textAlign(TextAlign.Center)
      .shadow(this.loginMode === mode ? this.getCurrentThemePalette().controlShadow : { radius: 0, color: '#00000000', offsetX: 0, offsetY: 0 })
      .animation({ duration: 250, curve: Curve.FastOutSlowIn })
      .onClick(() => {
        if (mode === 'login') {
          this.loginMode = 'login';
          this.registerNickname = '';
          this.registerLoginId = '';
          this.registerPassword = '';
          this.registerPasswordConfirm = '';
        } else {
          this.loginMode = 'register';
          this.loginId = '';
          this.password = '';
        }
      })
  }

  @Builder
  private buildLoginForm() {
    Column({ space: DesignConstants.SPACING.LG }) {
      TextInput({ placeholder: '邮箱 / 用户名', text: $$this.loginId })
        .fontSize(DesignConstants.FONT.SIZE.LG)
        .padding(DesignConstants.SPACING.LG)
        .backgroundColor(this.getSurfaceSubtle())
        .borderRadius(DesignConstants.RADIUS.XL)
        .shadow(this.getCurrentThemePalette().controlShadow)
        .type(InputType.Email)
        .onBlur(() => KeyboardDismissManager.getInstance().dismiss())

      TextInput({ placeholder: '密码', text: $$this.password })
        .fontSize(DesignConstants.FONT.SIZE.LG)
        .padding(DesignConstants.SPACING.LG)
        .backgroundColor(this.getSurfaceSubtle())
        .borderRadius(DesignConstants.RADIUS.XL)
        .shadow(this.getCurrentThemePalette().controlShadow)
        .type(InputType.Password)
        .onBlur(() => KeyboardDismissManager.getInstance().dismiss())

      Button('登录')
        .type(ButtonType.Normal)
        .backgroundColor(this.isValidLoginId(this.loginId) && this.isValidPassword(this.password)
          ? this.getCurrentThemePalette().accentBackground
          : this.getCurrentThemePalette().buttonSecondaryBackground)
        .fontColor(this.isValidLoginId(this.loginId) && this.isValidPassword(this.password)
          ? this.getCurrentThemePalette().accentText
          : this.getCurrentThemePalette().textSecondary)
        .fontSize(DesignConstants.FONT.SIZE.LG)
        .width('100%')
        .height(48)
        .borderRadius(DesignConstants.RADIUS.FULL)
        .shadow(this.getCurrentThemePalette().controlShadow)
        .border({ width: 1, color: this.getSurfaceBorder() })
        .enabled(this.isValidLoginId(this.loginId) && this.isValidPassword(this.password))
        .onClick(() => this.userLogin())

      Text('没有账号？切换到「注册」标签')
        .fontSize(DesignConstants.FONT.SIZE.SM)
        .fontColor(this.getCurrentThemePalette().textSecondary)
        .textAlign(TextAlign.Center)
    }
    .width('100%')
    .transition(
      TransitionEffect.OPACITY.animation({ duration: 320, curve: Curve.FastOutSlowIn })
        .combine(TransitionEffect.move(TransitionEdge.BOTTOM).animation({ duration: 320, curve: Curve.FastOutSlowIn }))
    )
  }

  @Builder
  private buildRegisterForm() {
    Column({ space: DesignConstants.SPACING.LG }) {
      TextInput({ placeholder: '昵称（必填）', text: $$this.registerNickname })
        .fontSize(DesignConstants.FONT.SIZE.LG)
        .padding(DesignConstants.SPACING.LG)
        .backgroundColor(this.getSurfaceSubtle())
        .borderRadius(DesignConstants.RADIUS.XL)
        .shadow(this.getCurrentThemePalette().controlShadow)
        .onBlur(() => KeyboardDismissManager.getInstance().dismiss())

      TextInput({ placeholder: '邮箱 / 用户名（登录凭证）', text: $$this.registerLoginId })
        .fontSize(DesignConstants.FONT.SIZE.LG)
        .padding(DesignConstants.SPACING.LG)
        .backgroundColor(this.getSurfaceSubtle())
        .borderRadius(DesignConstants.RADIUS.XL)
        .shadow(this.getCurrentThemePalette().controlShadow)
        .type(InputType.Email)
        .onBlur(() => KeyboardDismissManager.getInstance().dismiss())

      TextInput({ placeholder: '密码（8-128位，含字母+数字）', text: $$this.registerPassword })
        .fontSize(DesignConstants.FONT.SIZE.LG)
        .padding(DesignConstants.SPACING.LG)
        .backgroundColor(this.getSurfaceSubtle())
        .borderRadius(DesignConstants.RADIUS.XL)
        .shadow(this.getCurrentThemePalette().controlShadow)
        .type(InputType.Password)
        .onBlur(() => KeyboardDismissManager.getInstance().dismiss())

      TextInput({ placeholder: '确认密码', text: $$this.registerPasswordConfirm })
        .fontSize(DesignConstants.FONT.SIZE.LG)
        .padding(DesignConstants.SPACING.LG)
        .backgroundColor(this.getSurfaceSubtle())
        .borderRadius(DesignConstants.RADIUS.XL)
        .shadow(this.getCurrentThemePalette().controlShadow)
        .type(InputType.Password)
        .onBlur(() => KeyboardDismissManager.getInstance().dismiss())

      Button('注册')
        .type(ButtonType.Normal)
        .backgroundColor(this.isValidNickname(this.registerNickname)
          && this.isValidLoginId(this.registerLoginId)
          && this.isValidPassword(this.registerPassword)
          && this.registerPasswordConfirm === this.registerPassword
          ? this.getCurrentThemePalette().accentBackground
          : this.getCurrentThemePalette().buttonSecondaryBackground)
        .fontColor(this.isValidNickname(this.registerNickname)
          && this.isValidLoginId(this.registerLoginId)
          && this.isValidPassword(this.registerPassword)
          && this.registerPasswordConfirm === this.registerPassword
          ? this.getCurrentThemePalette().accentText
          : this.getCurrentThemePalette().textSecondary)
        .fontSize(DesignConstants.FONT.SIZE.LG)
        .width('100%')
        .height(48)
        .borderRadius(DesignConstants.RADIUS.FULL)
        .shadow(this.getCurrentThemePalette().controlShadow)
        .border({ width: 1, color: this.getSurfaceBorder() })
        .enabled(this.isValidNickname(this.registerNickname)
          && this.isValidLoginId(this.registerLoginId)
          && this.isValidPassword(this.registerPassword)
          && this.registerPasswordConfirm === this.registerPassword)
        .onClick(() => this.userRegister())

      Text('已有账号？切换到「登录」标签')
        .fontSize(DesignConstants.FONT.SIZE.SM)
        .fontColor(this.getCurrentThemePalette().textSecondary)
        .textAlign(TextAlign.Center)
    }
    .width('100%')
    .transition(
      TransitionEffect.OPACITY.animation({ duration: 320, curve: Curve.FastOutSlowIn })
        .combine(TransitionEffect.move(TransitionEdge.BOTTOM).animation({ duration: 320, curve: Curve.FastOutSlowIn }))
    )
  }

  // ==================== 认证逻辑 ====================

  private async userLogin() {
    const trimmedLoginId = this.loginId.trim();
    if (!trimmedLoginId || !this.password.trim()) {
      SystemToast.showWarning('请输入邮箱和密码', this.getUIContext());
      return;
    }
    if (!this.isValidLoginId(trimmedLoginId)) {
      SystemToast.showWarning('登录账号长度需在3-64字符之间', this.getUIContext());
      return;
    }
    if (!this.isValidPassword(this.password)) {
      SystemToast.showWarning(PASSWORD_POLICY_HINT, this.getUIContext());
      return;
    }

    this.isLoading = true;
    this.loadingProgress = 0;

    try {
      await LoadingUtil.withMinimumTime(async () => {
        this.loadingStage = '正在验证登录信息...';
        this.loadingProgress = 40;

        const loginRequest: LoginRequest = {
          login_id: trimmedLoginId,
          password: this.password,
          client_type: 'app'
        };

        const response = await ApiClient.login(loginRequest);
        if (response.code === 0 && response.data) {
          this.loadingStage = '登录成功，正在初始化...';
          this.loadingProgress = 70;

          PasswordSecurityUtil.clearSensitiveString(this.password);
          await this.handleAuthSuccess(response.data);
          this.loadingProgress = 100;
        } else {
          SystemToast.showError(getErrorMessage(response, '登录失败'), this.getUIContext());
        }
      }, LoadingUtil.DEFAULT_ACTION_MS);
    } catch (error) {
      SystemToast.showError(error instanceof Error ? error.message : '登录失败', this.getUIContext());
    } finally {
      this.isLoading = false;
      this.loadingStage = '';
      this.loadingProgress = 0;
    }
  }

  private async userRegister() {
    const trimmedNickname = this.registerNickname.trim();
    const trimmedLoginId = this.registerLoginId.trim();
    if (!trimmedNickname) {
      SystemToast.showWarning('请输入昵称', this.getUIContext());
      return;
    }
    if (!trimmedLoginId) {
      SystemToast.showWarning('请输入邮箱或用户名', this.getUIContext());
      return;
    }
    if (!this.isValidNickname(trimmedNickname)) {
      SystemToast.showWarning('昵称长度需在1-30字符之间', this.getUIContext());
      return;
    }
    if (!this.isValidLoginId(trimmedLoginId)) {
      SystemToast.showWarning('登录账号长度需在3-64字符之间', this.getUIContext());
      return;
    }
    if (!this.isValidPassword(this.registerPassword)) {
      SystemToast.showWarning(PASSWORD_POLICY_HINT, this.getUIContext());
      return;
    }
    if (this.registerPassword !== this.registerPasswordConfirm) {
      SystemToast.showWarning('两次输入的密码不一致', this.getUIContext());
      return;
    }

    this.isLoading = true;
    this.loadingProgress = 0;

    try {
      await LoadingUtil.withMinimumTime(async () => {
        this.loadingStage = '正在注册...';
        this.loadingProgress = 40;

        const registerRequest: RegisterRequest = {
          nickname: trimmedNickname,
          login_id: trimmedLoginId,
          password: this.registerPassword,
          client_type: 'app'
        };

        const response = await ApiClient.register(registerRequest);
        if (response.code === 0 && response.data) {
          this.loadingStage = '注册成功，正在初始化...';
          this.loadingProgress = 70;

          PasswordSecurityUtil.clearSensitiveString(this.registerPassword);
          PasswordSecurityUtil.clearSensitiveString(this.registerPasswordConfirm);
          await this.handleAuthSuccess(response.data);
          this.loadingProgress = 100;
        } else {
          SystemToast.showError(getErrorMessage(response, '注册失败'), this.getUIContext());
        }
      }, LoadingUtil.DEFAULT_ACTION_MS);
    } catch (error) {
      SystemToast.showError(error instanceof Error ? error.message : '注册失败', this.getUIContext());
    } finally {
      this.isLoading = false;
      this.loadingStage = '';
      this.loadingProgress = 0;
    }
  }

  private finishAuthNavigation(): void {
    if (this.restoreAction === 'open_private_chat' && this.restoreTargetId.trim().length > 0) {
      const params = new ChatRouteParam();
      params.chatId = this.restoreTargetId.trim();
      params.chatType = 'private_chat';
      params.targetId = this.restoreTargetId.trim();
      params.title = this.restoreTitle.trim().length > 0 ? this.restoreTitle.trim() : '私聊';
      AppRouter.push(PageNames.CHAT, params);
      return;
    }

    const redirectTarget = this.redirectUrl.trim();
    if (redirectTarget.length > 0) {
      AppRouter.replacePath(redirectTarget);
      if (this.restoreAction === 'open_create_sheet' && redirectTarget === PageNames.INDEX) {
        AppStorage.setOrCreate<boolean>('openCreateSheet', true);
      }
      return;
    }
    AppRouter.pop();
  }

  private async handleAuthSuccess(authData: AuthData) {
    await AuthAccountManager.upsertAndActivate(authData);
    await HttpUtil.setToken(authData.access_token);

    const basicUserInfo: MyProfile = {
      user_id: authData.user_id,
      nickname: '',
      avatar_url: '',
      user_type: 'temp',
      bio: '',
      tags: [],
      email: '',
      phone: '',
      location: { point: { latitude: 0, longitude: 0 }, location_name: '' },
      presence_status: 'online',
      last_online: '',
      created_at: new Date().toISOString(),
      stats: {
        moments_count: 0,
        friends_count: 0,
        followers_count: 0,
        following_count: 0,
        groups_count: 0
      }
    };
    this.stateManager.updateUser(basicUserInfo);

    try {
      const profileResp = await ApiClient.getMyProfile();
      if (this.isDestroyed) return;
      if (profileResp.code === 0 && profileResp.data) {
        this.stateManager.updateUser(profileResp.data);
        hilog.info(DOMAIN_NUMBER, TAG, `用户类型: ${profileResp.data.user_type || 'unknown'}`);
      }
    } catch (error) {
      hilog.warn(DOMAIN_NUMBER, TAG, `获取用户资料失败，使用基础信息: ${JSON.stringify(error)}`);
    }

    try {
      await this.webSocketService.connect(authData.access_token);
    } catch (error) {
      hilog.warn(DOMAIN_NUMBER, TAG, `WebSocket连接失败: ${JSON.stringify(error)}`);
    }

    this.finishAuthNavigation();
    SystemToast.showSuccess('欢迎！', this.getUIContext());
  }
}

@Builder
export function LoginPageBuilder() {
  LoginPage()
}
