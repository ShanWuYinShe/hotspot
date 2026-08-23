/**
 * 登录/注册页面
 * 两种模式：登录、注册
 * 认证成功后获取完整 profile（含 user_type）
 */

import { GlobalStateManager } from '../../../core/state/GlobalStateManager';
import { PageShellContainer } from '../../../components/PageShellContainer';
import type { BreakpointType } from '../../../core/responsive/Breakpoint';
import { ApiClient } from '../../../core/api/ApiClient';
import { HttpUtil } from '../../../core/network/HttpUtil';
import { WebSocketService } from '../../../core/network/WebSocketService';
import { AppRouter, ChatRouteParam } from '../../../core/router/AppRouter';
import { PageNames } from '../../../constants/AppConstants';
import { SystemLoadingView } from '../../../components/SystemLoadingView';
import { SystemInput } from '../../../components/SystemInput';
import { SystemButton } from '../../../components/SystemButton';
import { TabsPillGroup } from '../../../components/TabsPillGroup';
import { SystemToast } from '../../../components/SystemToast';
import { PasswordSecurityUtil } from '../../../core/security/PasswordSecurityUtil';
import { AuthData, AccountInfo, LoginRequest, RegisterRequest, SendRegisterCodeRequest, GuestRequest } from '../../../core/types/api';
import { DeviceIdUtil } from '../../../core/util/DeviceIdUtil';
import { AuthAccountManager } from '../../../core/state/AuthAccountManager';
import { getErrorMessage } from '../../../core/network/ApiErrorUtil';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { DesignConstants } from '../../../constants/DesignConstants';
import { curves } from '@kit.ArkUI';
import { LoadingUtil } from '../../../core/utils/LoadingUtil';
import { AppIcon } from '../../../components/SystemIcon';
import { SystemUiController } from '../../../core/utils/SystemUiController';
import { KeyboardDismissManager } from '../../../core/utils/KeyboardDismissManager';
import { SafeAreaUtils } from '../../../core/utils/SafeAreaUtils';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { KEY_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';
import { AppThemeVisuals, InputBorderResolution } from '../../../core/theme/AppThemeVisuals';
import { validateField, FieldRule } from '../../../core/util/FormValidator';

const LOGIN_ID_RULES: FieldRule[] = [
  { required: true, requiredMessage: '请输入账号' },
  { minLength: 3, minLengthMessage: '账号需 3-64 个字符' },
  { maxLength: 64, maxLengthMessage: '账号需 3-64 个字符' }
];
const NICKNAME_RULES: FieldRule[] = [
  { required: true, requiredMessage: '请输入昵称' },
  { maxLength: 30, maxLengthMessage: '昵称最多 30 个字符' }
];
// 访客昵称可选（null 时服务端自动生成），仅限长度校验
const GUEST_NICKNAME_RULES: FieldRule[] = [
  { maxLength: 30, maxLengthMessage: '昵称最多 30 个字符' }
];

const DOMAIN_NUMBER: number = 0xFF02;
const TAG: string = 'LoginPage';
const PASSWORD_POLICY_HINT: string = '密码需为8-128位且同时包含字母和数字';

@Preview
@Component
export struct LoginPage {
  @StorageProp('currentBreakpoint') currentBreakpoint: BreakpointType = 'sm';
  pathStack: NavPathStack = new NavPathStack();
  @StorageLink('isLoggedIn') isLoggedIn: boolean = false;
  @StorageLink('safeAreaTop') safeAreaTop: number = 0;
  @StorageLink('safeAreaBottom') safeAreaBottom: number = 0;
  @StorageLink(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'vibrant_light';

  @State loginId: string = '';
  @State password: string = '';
  @State registerNickname: string = '';
  @State registerLoginId: string = '';
  @State registerPassword: string = '';
  @State registerPasswordConfirm: string = '';
  @State registerVerificationCode: string = '';
  @State guestNickname: string = '';
  @State isLoading: boolean = false;
  @State loadingStage: string = '';
  @State loadingProgress: number = 0;
  @State loginMode: 'login' | 'register' | 'guest' = 'login';
  @State isSendingRegisterCode: boolean = false;
  @State registerCodeCountdown: number = 0;
  @State redirectUrl: string = '';
  @State redirectAction: string = '';
  @State restoreAction: string = '';
  @State restoreTargetId: string = '';
  @State restoreTitle: string = '';
  @State isBackPressed: boolean = false;
  @State isLoginBtnPressed: boolean = false;
  @State isRegisterBtnPressed: boolean = false;
  @State isGuestBtnPressed: boolean = false;
  @State loginIdFocused: boolean = false;
  @State passwordFocused: boolean = false;
  @State registerNicknameFocused: boolean = false;
  @State registerLoginIdFocused: boolean = false;
  @State registerPasswordFocused: boolean = false;
  @State registerPasswordConfirmFocused: boolean = false;
  @State registerVerificationCodeFocused: boolean = false;
  @State guestNicknameFocused: boolean = false;
  @State loginIdTouched: boolean = false;
  @State passwordTouched: boolean = false;
  @State registerNicknameTouched: boolean = false;
  @State registerLoginIdTouched: boolean = false;
  @State registerPasswordTouched: boolean = false;
  @State registerPasswordConfirmTouched: boolean = false;
  @State registerVerificationCodeTouched: boolean = false;
  @State guestNicknameTouched: boolean = false;

  private stateManager = GlobalStateManager.getInstance();
  private webSocketService = WebSocketService.getInstance();
  private isDestroyed: boolean = false;
  private registerCodeTimerId: number = -1;

  private isValidLoginId(loginId: string): boolean {
    const trimmed = loginId.trim();
    return trimmed.length >= 3 && trimmed.length <= 64;
  }

  private isValidNickname(nickname: string): boolean {
    const trimmed = nickname.trim();
    return trimmed.length >= 1 && trimmed.length <= 30;
  }

  /** 访客昵称可选：留空合法（提交 null，服务端自动生成），非空时限 30 字符 */
  private isValidOptionalNickname(nickname: string): boolean {
    const trimmed = nickname.trim();
    return trimmed.length === 0 || trimmed.length <= 30;
  }

  /**
   * 手机号（纯数字）或邮箱（含 '@'）型 login_id 需验证码（spec public.md →
   * auth/register 所有权验证；用户名型 verification_code 必须不传）
   */
  private isPhoneOrEmailLoginId(loginId: string): boolean {
    const trimmed = loginId.trim();
    if (trimmed.includes('@')) {
      return true;
    }
    return trimmed.length > 0 && /^[0-9]+$/.test(trimmed);
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
    this.stopRegisterCodeCountdown();
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

  private fieldBorder(focused: boolean, hasError: boolean): InputBorderResolution {
    return AppThemeVisuals.resolveInputBorder({
      backgroundColor: this.getSurfaceSubtle(),
      borderColor: this.getSurfaceBorder(),
      borderWidth: 1,
      borderRadius: DesignConstants.SHAPE.INPUT,
      shadow: { radius: 0, color: DesignConstants.COLOR.TRANSPARENT, offsetX: 0, offsetY: 0 },
      height: DesignConstants.COMPONENT.INPUT_HEIGHT,
      horizontalPadding: DesignConstants.SPACING.MD
    }, focused, hasError, this.resolvedThemeId);
  }

  private getLoginIdError(): string | undefined {
    return this.loginIdTouched ? validateField(this.loginId, LOGIN_ID_RULES) : undefined;
  }

  private getPasswordError(): string | undefined {
    if (!this.passwordTouched) {
      return undefined;
    }
    return validateField(this.password, [
      { required: true, requiredMessage: '请输入密码' },
      { minLength: 8, minLengthMessage: '密码需为 8-128 位且同时包含字母和数字' },
      { maxLength: 128, maxLengthMessage: '密码需为 8-128 位且同时包含字母和数字' },
      { custom: (v: string) => (v && !this.hasLetterAndDigit(v) ? '密码需为 8-128 位且同时包含字母和数字' : undefined) }
    ]);
  }

  private getRegisterNicknameError(): string | undefined {
    return this.registerNicknameTouched ? validateField(this.registerNickname, NICKNAME_RULES) : undefined;
  }

  private getRegisterLoginIdError(): string | undefined {
    return this.registerLoginIdTouched ? validateField(this.registerLoginId, LOGIN_ID_RULES) : undefined;
  }

  private getRegisterPasswordError(): string | undefined {
    if (!this.registerPasswordTouched) {
      return undefined;
    }
    return validateField(this.registerPassword, [
      { required: true, requiredMessage: '请输入密码' },
      { minLength: 8, minLengthMessage: '密码需为 8-128 位且同时包含字母和数字' },
      { maxLength: 128, maxLengthMessage: '密码需为 8-128 位且同时包含字母和数字' },
      { custom: (v: string) => (v && !this.hasLetterAndDigit(v) ? '密码需为 8-128 位且同时包含字母和数字' : undefined) }
    ]);
  }

  private getRegisterPasswordConfirmError(): string | undefined {
    if (!this.registerPasswordConfirmTouched) {
      return undefined;
    }
    return validateField(this.registerPasswordConfirm, [
      { required: true, requiredMessage: '请确认密码' },
      { custom: (v: string) => (v && v !== this.registerPassword ? '两次密码不一致' : undefined) }
    ]);
  }

  private needsRegisterVerificationCode(): boolean {
    return this.isPhoneOrEmailLoginId(this.registerLoginId);
  }

  private getRegisterVerificationCodeError(): string | undefined {
    if (!this.needsRegisterVerificationCode()) {
      return undefined;
    }
    return this.registerVerificationCodeTouched
      ? validateField(this.registerVerificationCode, [
          { required: true, requiredMessage: '请输入验证码' }
        ])
      : undefined;
  }

  private getGuestNicknameError(): string | undefined {
    return this.guestNicknameTouched ? validateField(this.guestNickname, GUEST_NICKNAME_RULES) : undefined;
  }

  build() {
    NavDestination() {
    PageShellContainer({ variant: 'form', currentBreakpoint: this.currentBreakpoint, noHorizontalPadding: true }) {
      Column() {
      if (this.isLoading) {
        SystemLoadingView({
          message: this.loadingStage || '加载中...',
          progressType: 'determinate',
          showProgress: true,
          externalProgress: this.loadingProgress
        })
          .transition(
            TransitionEffect.OPACITY.animation({ duration: DesignConstants.ANIMATION.DURATION_SHEET, curve: DesignConstants.ANIMATION.CURVE_FAST_OUT_SLOW_IN })
              .combine(TransitionEffect.move(TransitionEdge.BOTTOM).animation({ duration: DesignConstants.ANIMATION.DURATION_SHEET, curve: DesignConstants.ANIMATION.CURVE_FAST_OUT_SLOW_IN }))
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
            .width(DesignConstants.COMPONENT.AVATAR_SIZE.MEDIUM)
            .height(DesignConstants.COMPONENT.AVATAR_SIZE.MEDIUM)
            .textAlign(TextAlign.Center)
            .scale(this.isBackPressed ?
              { x: DesignConstants.INTERACTION.SCALE_PRESSED_ICON, y: DesignConstants.INTERACTION.SCALE_PRESSED_ICON } :
              { x: 1, y: 1 })
            .animation({
              curve: curves.springMotion(
                DesignConstants.INTERACTION.SPRING_RESPONSE_PRESS,
                DesignConstants.INTERACTION.SPRING_DAMPING_PRESS
              )
            })
            .onTouch((event: TouchEvent) => {
              if (event.type === TouchType.Down) {
                this.isBackPressed = true;
              } else if (event.type === TouchType.Up || event.type === TouchType.Cancel) {
                this.isBackPressed = false;
              }
            })

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
          .backgroundColor(DesignConstants.COLOR.TRANSPARENT)
          .expandSafeArea([SafeAreaType.SYSTEM], [SafeAreaEdge.TOP])

          // 标题区域（固定）
          Column({ space: DesignConstants.SPACING.SM }) {
            Image($r('app.media.foreground'))
              .width(DesignConstants.COMPONENT.LOADING_CONTAINER_SIZE)
              .height(DesignConstants.COMPONENT.LOADING_CONTAINER_SIZE)
              .objectFit(ImageFit.Contain)

            Text($r('app.string.app_name'))
              .fontSize(DesignConstants.FONT.SIZE.XXL)
              .fontWeight(DesignConstants.FONT.WEIGHT.BOLD)
              .fontColor(this.getCurrentThemePalette().textPrimary)

            Text($r('app.string.app_description'))
              .fontSize(DesignConstants.FONT.SIZE.LG)
              .fontColor(this.getCurrentThemePalette().textSecondary)
          }
          .width('100%')
          .alignItems(HorizontalAlign.Center)
          .padding({ left: DesignConstants.SPACING.XXL, right: DesignConstants.SPACING.XXL,
            bottom: DesignConstants.SPACING.LG })

          // 模式切换标签（统一 TabsPillGroup）
          TabsPillGroup({
            tabs: [{ label: '登录' }, { label: '注册' }, { label: '访客' }],
            activeIndex: Math.max(0, ['login', 'register', 'guest'].indexOf(this.loginMode)),
            actions: {
              onChange: (index: number) => {
                const mode = ['login', 'register', 'guest'][index] || 'login';
                if (mode === 'login') {
                  this.loginMode = 'login';
                  this.registerNickname = '';
                  this.registerLoginId = '';
                  this.registerPassword = '';
                  this.registerPasswordConfirm = '';
                  this.resetRegisterCodeState();
                } else if (mode === 'register') {
                  this.loginMode = 'register';
                  this.loginId = '';
                  this.password = '';
                } else {
                  this.loginMode = 'guest';
                  this.loginId = '';
                  this.password = '';
                  this.registerNickname = '';
                  this.registerLoginId = '';
                  this.registerPassword = '';
                  this.registerPasswordConfirm = '';
                  this.resetRegisterCodeState();
                }
              }
            }
          })
            .margin({ left: DesignConstants.SPACING.XXL, right: DesignConstants.SPACING.XXL })

          // 表单区域（可滚动，凸起卡片包裹）
          Scroll() {
            Column() {
              Column() {
                if (this.loginMode === 'login') {
                  this.buildLoginForm()
                } else if (this.loginMode === 'register') {
                  this.buildRegisterForm()
                } else {
                  this.buildGuestForm()
                }
              }
              .width('100%')
              .padding(DesignConstants.SPACING.LG)
              .backgroundColor(this.getSurfaceBackground())
              .borderRadius(DesignConstants.RADIUS.LG)
              .shadow(this.getCurrentThemePalette().elevatedShadow)
              .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getSurfaceBorder() })
              .margin({ bottom: DesignConstants.SPACING.MD3 })
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
  private buildLoginForm() {
    Column({ space: DesignConstants.SPACING.LG }) {
      SystemInput({
        value: this.loginId,
        placeholder: '邮箱 / 用户名',
        fontSize: DesignConstants.FONT.SIZE.MD,
        hasError: this.getLoginIdError() !== undefined,
        onChange: (value: string) => { this.loginId = value; },
        onFocusChange: (focused: boolean) => {
          this.loginIdFocused = focused;
          if (!focused) {
            this.loginIdTouched = true;
            KeyboardDismissManager.getInstance().dismiss();
          }
        }
      })

      if (this.getLoginIdError()) {
        Text(this.getLoginIdError())
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .fontColor(this.getCurrentThemePalette().error)
          .width('100%')
          .padding({ left: DesignConstants.SPACING.MD })
      }

      SystemInput({
        value: this.password,
        placeholder: '密码',
        passwordMode: true,
        fontSize: DesignConstants.FONT.SIZE.MD,
        hasError: this.getPasswordError() !== undefined,
        onChange: (value: string) => { this.password = value; },
        onFocusChange: (focused: boolean) => {
          this.passwordFocused = focused;
          if (!focused) {
            this.passwordTouched = true;
            KeyboardDismissManager.getInstance().dismiss();
          }
        }
      })

      if (this.getPasswordError()) {
        Text(this.getPasswordError())
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .fontColor(this.getCurrentThemePalette().error)
          .width('100%')
          .padding({ left: DesignConstants.SPACING.MD })
      }

      SystemButton({
        label: '登录',
        variant: 'primary',
        buttonSize: 'large',
        fullWidth: true,
        disabled: !(this.isValidLoginId(this.loginId) && this.isValidPassword(this.password)),
        onPress: () => { this.userLogin(); }
      })
        .scale(this.isLoginBtnPressed ?
          { x: DesignConstants.INTERACTION.SCALE_PRESSED_BUTTON, y: DesignConstants.INTERACTION.SCALE_PRESSED_BUTTON } :
          { x: 1, y: 1 })
        .animation({
          curve: curves.springMotion(
            DesignConstants.INTERACTION.SPRING_RESPONSE_PRESS,
            DesignConstants.INTERACTION.SPRING_DAMPING_PRESS
          )
        })
        .onTouch((event: TouchEvent) => {
          if (event.type === TouchType.Down) {
            this.isLoginBtnPressed = true;
          } else if (event.type === TouchType.Up || event.type === TouchType.Cancel) {
            this.isLoginBtnPressed = false;
          }
        })
        .onClick(() => this.userLogin())

      Text('没有账号？切换到「注册」标签')
        .fontSize(DesignConstants.FONT.SIZE.SM)
        .fontColor(this.getCurrentThemePalette().textSecondary)
        .textAlign(TextAlign.Center)
    }
    .width('100%')
    .transition(
      TransitionEffect.OPACITY.animation({ duration: DesignConstants.ANIMATION.DURATION_SHEET, curve: DesignConstants.ANIMATION.CURVE_FAST_OUT_SLOW_IN })
        .combine(TransitionEffect.move(TransitionEdge.BOTTOM).animation({ duration: DesignConstants.ANIMATION.DURATION_SHEET, curve: DesignConstants.ANIMATION.CURVE_FAST_OUT_SLOW_IN }))
    )
  }

  @Builder
  private buildRegisterForm() {
    Column({ space: DesignConstants.SPACING.LG }) {
      SystemInput({
        value: this.registerNickname,
        placeholder: '昵称（必填）',
        fontSize: DesignConstants.FONT.SIZE.MD,
        hasError: this.getRegisterNicknameError() !== undefined,
        onChange: (value: string) => { this.registerNickname = value; },
        onFocusChange: (focused: boolean) => {
          this.registerNicknameFocused = focused;
          if (!focused) {
            this.registerNicknameTouched = true;
            KeyboardDismissManager.getInstance().dismiss();
          }
        }
      })

      if (this.getRegisterNicknameError()) {
        Text(this.getRegisterNicknameError())
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .fontColor(this.getCurrentThemePalette().error)
          .width('100%')
          .padding({ left: DesignConstants.SPACING.MD })
      }

      SystemInput({
        value: this.registerLoginId,
        placeholder: '邮箱 / 用户名（登录凭证）',
        fontSize: DesignConstants.FONT.SIZE.MD,
        hasError: this.getRegisterLoginIdError() !== undefined,
        onChange: (value: string) => { this.registerLoginId = value; },
        onFocusChange: (focused: boolean) => {
          this.registerLoginIdFocused = focused;
          if (!focused) {
            this.registerLoginIdTouched = true;
            KeyboardDismissManager.getInstance().dismiss();
          }
        }
      })

      if (this.getRegisterLoginIdError()) {
        Text(this.getRegisterLoginIdError())
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .fontColor(this.getCurrentThemePalette().error)
          .width('100%')
          .padding({ left: DesignConstants.SPACING.MD })
      }

      // 手机号/邮箱型 login_id 需验证码（所有权验证）；用户名型不显示
      if (this.needsRegisterVerificationCode()) {
        Row({ space: DesignConstants.SPACING.SM }) {
          SystemInput({
            value: this.registerVerificationCode,
            placeholder: '验证码',
            inputType: 'number',
            maxLength: 6,
            fontSize: DesignConstants.FONT.SIZE.MD,
            hasError: this.getRegisterVerificationCodeError() !== undefined,
            onChange: (value: string) => { this.registerVerificationCode = value; },
            onFocusChange: (focused: boolean) => {
              this.registerVerificationCodeFocused = focused;
              if (!focused) {
                this.registerVerificationCodeTouched = true;
                KeyboardDismissManager.getInstance().dismiss();
              }
            }
          })
            .layoutWeight(1)

          SystemButton({
            label: this.registerCodeCountdown > 0 ? `${this.registerCodeCountdown}s 后重发` : '发送验证码',
            variant: 'secondary',
            buttonSize: 'small',
            loading: this.isSendingRegisterCode,
            disabled: this.isSendingRegisterCode || this.registerCodeCountdown > 0,
            onPress: () => { this.sendRegisterVerificationCode(); }
          })
        }
        .width('100%')
        .alignItems(VerticalAlign.Center)

        if (this.getRegisterVerificationCodeError()) {
          Text(this.getRegisterVerificationCodeError())
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .fontColor(this.getCurrentThemePalette().error)
            .width('100%')
            .padding({ left: DesignConstants.SPACING.MD })
        }
      }

      SystemInput({
        value: this.registerPassword,
        placeholder: '密码（8-128位，含字母+数字）',
        passwordMode: true,
        fontSize: DesignConstants.FONT.SIZE.MD,
        hasError: this.getRegisterPasswordError() !== undefined,
        onChange: (value: string) => { this.registerPassword = value; },
        onFocusChange: (focused: boolean) => {
          this.registerPasswordFocused = focused;
          if (!focused) {
            this.registerPasswordTouched = true;
            KeyboardDismissManager.getInstance().dismiss();
          }
        }
      })

      if (this.getRegisterPasswordError()) {
        Text(this.getRegisterPasswordError())
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .fontColor(this.getCurrentThemePalette().error)
          .width('100%')
          .padding({ left: DesignConstants.SPACING.MD })
      }

      SystemInput({
        value: this.registerPasswordConfirm,
        placeholder: '确认密码',
        passwordMode: true,
        fontSize: DesignConstants.FONT.SIZE.MD,
        hasError: this.getRegisterPasswordConfirmError() !== undefined,
        onChange: (value: string) => { this.registerPasswordConfirm = value; },
        onFocusChange: (focused: boolean) => {
          this.registerPasswordConfirmFocused = focused;
          if (!focused) {
            this.registerPasswordConfirmTouched = true;
            KeyboardDismissManager.getInstance().dismiss();
          }
        }
      })

      if (this.getRegisterPasswordConfirmError()) {
        Text(this.getRegisterPasswordConfirmError())
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .fontColor(this.getCurrentThemePalette().error)
          .width('100%')
          .padding({ left: DesignConstants.SPACING.MD })
      }

      SystemButton({
        label: '注册',
        variant: 'primary',
        buttonSize: 'large',
        fullWidth: true,
        disabled: !(this.isValidNickname(this.registerNickname)
          && this.isValidLoginId(this.registerLoginId)
          && this.isValidPassword(this.registerPassword)
          && this.registerPasswordConfirm === this.registerPassword
          && (!this.needsRegisterVerificationCode() || this.registerVerificationCode.trim().length > 0)),
        onPress: () => { this.userRegister(); }
      })

      Text('已有账号？切换到「登录」标签')
        .fontSize(DesignConstants.FONT.SIZE.SM)
        .fontColor(this.getCurrentThemePalette().textSecondary)
        .textAlign(TextAlign.Center)
    }
    .width('100%')
    .transition(
      TransitionEffect.OPACITY.animation({ duration: DesignConstants.ANIMATION.DURATION_SHEET, curve: DesignConstants.ANIMATION.CURVE_FAST_OUT_SLOW_IN })
        .combine(TransitionEffect.move(TransitionEdge.BOTTOM).animation({ duration: DesignConstants.ANIMATION.DURATION_SHEET, curve: DesignConstants.ANIMATION.CURVE_FAST_OUT_SLOW_IN }))
    )
  }

  // ==================== 认证逻辑 ====================

  @Builder
  private buildGuestForm() {
    Column({ space: DesignConstants.SPACING.LG }) {
      Text('访客模式可快速体验，部分功能受限。昵称可留空，系统将自动生成。')
        .fontSize(DesignConstants.FONT.SIZE.SM)
        .fontColor(this.getCurrentThemePalette().textSecondary)
        .width('100%')
        .padding(DesignConstants.SPACING.MD)
        .backgroundColor(this.getSurfaceBackground())
        .borderRadius(DesignConstants.RADIUS.LG)
        .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getSurfaceBorder() })

      SystemInput({
        value: this.guestNickname,
        placeholder: '昵称（可选）',
        fontSize: DesignConstants.FONT.SIZE.MD,
        hasError: this.getGuestNicknameError() !== undefined,
        onChange: (value: string) => { this.guestNickname = value; },
        onFocusChange: (focused: boolean) => {
          this.guestNicknameFocused = focused;
          if (!focused) {
            this.guestNicknameTouched = true;
            KeyboardDismissManager.getInstance().dismiss();
          }
        }
      })

      if (this.getGuestNicknameError()) {
        Text(this.getGuestNicknameError())
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .fontColor(this.getCurrentThemePalette().error)
          .width('100%')
          .padding({ left: DesignConstants.SPACING.MD })
      }

      SystemButton({
        label: '访客登录',
        variant: 'primary',
        buttonSize: 'large',
        fullWidth: true,
        disabled: !this.isValidOptionalNickname(this.guestNickname),
        onPress: () => { this.userGuestLogin(); }
      })

      Text('随时可在「登录」或「注册」升级为正式账号')
        .fontSize(DesignConstants.FONT.SIZE.SM)
        .fontColor(this.getCurrentThemePalette().textSecondary)
        .textAlign(TextAlign.Center)
    }
    .width('100%')
    .transition(
      TransitionEffect.OPACITY.animation({ duration: DesignConstants.ANIMATION.DURATION_SHEET, curve: DesignConstants.ANIMATION.CURVE_FAST_OUT_SLOW_IN })
        .combine(TransitionEffect.move(TransitionEdge.BOTTOM).animation({ duration: DesignConstants.ANIMATION.DURATION_SHEET, curve: DesignConstants.ANIMATION.CURVE_FAST_OUT_SLOW_IN }))
    )
  }

  private async userGuestLogin() {
    // 昵称可选：留空提交 null，服务端自动生成默认昵称（spec auth.md → guest）
    const nickname = this.guestNickname.trim();
    if (!this.isValidOptionalNickname(nickname)) {
      SystemToast.showWarning('昵称最多 30 个字符', this.getUIContext());
      return;
    }

    this.isLoading = true;
    this.loadingProgress = 0;

    try {
      await LoadingUtil.withMinimumTime(async () => {
        this.loadingStage = '正在创建访客身份...';
        this.loadingProgress = 40;

        const request: GuestRequest = {
          nickname: nickname.length > 0 ? nickname : null,
          client_type: 'app',
          device_id: DeviceIdUtil.getDeviceId()
        };
        const response = await ApiClient.createTempUser(request);
        if (response.code === 0 && response.data) {
          this.loadingStage = '创建成功，正在初始化...';
          this.loadingProgress = 70;
          await this.handleAuthSuccess(response.data);
          this.loadingProgress = 100;
        } else {
          SystemToast.showError(getErrorMessage(response, '创建失败'), this.getUIContext());
        }
      }, LoadingUtil.DEFAULT_ACTION_MS);
    } catch (error) {
      SystemToast.showError(error instanceof Error ? error.message : '创建失败', this.getUIContext());
    } finally {
      this.isLoading = false;
      this.loadingStage = '';
      this.loadingProgress = 0;
    }
  }

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
          client_type: 'app',
          device_id: DeviceIdUtil.getDeviceId()
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

  // ==================== 注册验证码 ====================

  private resetRegisterCodeState(): void {
    this.registerVerificationCode = '';
    this.registerVerificationCodeTouched = false;
    this.stopRegisterCodeCountdown();
  }

  private stopRegisterCodeCountdown(): void {
    if (this.registerCodeTimerId >= 0) {
      clearInterval(this.registerCodeTimerId);
      this.registerCodeTimerId = -1;
    }
    this.registerCodeCountdown = 0;
  }

  private startRegisterCodeCountdown(): void {
    this.stopRegisterCodeCountdown();
    this.registerCodeCountdown = 60;
    this.registerCodeTimerId = setInterval(() => {
      if (this.registerCodeCountdown <= 1) {
        this.stopRegisterCodeCountdown();
        return;
      }
      this.registerCodeCountdown -= 1;
    }, 1000);
  }

  private async sendRegisterVerificationCode(): Promise<void> {
    const trimmedLoginId = this.registerLoginId.trim();
    if (!this.isValidLoginId(trimmedLoginId)) {
      SystemToast.showWarning('请输入有效的手机号或邮箱', this.getUIContext());
      return;
    }
    if (this.isSendingRegisterCode || this.registerCodeCountdown > 0) {
      return;
    }
    this.isSendingRegisterCode = true;
    try {
      const request: SendRegisterCodeRequest = { login_id: trimmedLoginId, client_type: 'app' };
      const response = await ApiClient.sendRegisterCode(request);
      if (response.code === 0) {
        SystemToast.showSuccess('验证码已发送', this.getUIContext());
        this.startRegisterCodeCountdown();
      } else {
        SystemToast.showError(getErrorMessage(response, '验证码发送失败'), this.getUIContext());
      }
    } catch (error) {
      SystemToast.showError(error instanceof Error ? error.message : '验证码发送失败', this.getUIContext());
    } finally {
      this.isSendingRegisterCode = false;
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
    // 手机号/邮箱型 login_id 必须携带验证码（所有权验证）；用户名型不传
    const needsCode = this.isPhoneOrEmailLoginId(trimmedLoginId);
    const trimmedCode = this.registerVerificationCode.trim();
    if (needsCode && !trimmedCode) {
      this.registerVerificationCodeTouched = true;
      SystemToast.showWarning('请输入验证码', this.getUIContext());
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
          verification_code: needsCode ? trimmedCode : null,
          password: this.registerPassword,
          client_type: 'app',
          device_id: DeviceIdUtil.getDeviceId()
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

    const basicUserInfo: AccountInfo = {
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
      last_online_at: '',
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
      const profileResp = await ApiClient.getAccountInfo();
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