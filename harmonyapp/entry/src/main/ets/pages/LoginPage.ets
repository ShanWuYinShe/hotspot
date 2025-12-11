/**
 * 极简登录页面
 */

import { GlobalState, GlobalStateManager } from '../common/state/GlobalStateManager';
import { ApiClient } from '../common/api/ApiClient';
import { PreferencesUtil } from '../common/storage/PreferencesUtil';
import { HttpUtil } from '../common/network/HttpUtil';
import { WebSocketService } from '../common/websocket/WebSocketService';
import { PageNames } from '../constants/AppConstants';
import { SystemLoadingView } from '../components/SystemLoadingView';
import { SystemToast } from '../components/SystemToast';
import { PasswordSecurityUtil } from '../common/security/PasswordSecurityUtil';
import { AuthData, CreateTempRequest, MyProfileData, LoginRequest } from '../common/types/api';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { I18nManager } from '../common/i18n/I18nManager';

const DOMAIN_NUMBER: number = 0xFF02;
const TAG: string = 'LoginPage';


@Entry
@Component
struct LoginPage {
  @State globalState: GlobalState = GlobalStateManager.getInstance().getState();
  @State nickname: string = '';
  @State loginId: string = '';
  @State password: string = '';
  @State isLoading: boolean = false;
  @State loadingStage: string = '';
  @State loadingProgress: number = 0;
  @State loginMode: 'temp' | 'register' | 'login' = 'temp';
  private stateManager = GlobalStateManager.getInstance();
  private webSocketService = WebSocketService.getInstance();
  private i18nManager = I18nManager.getInstance();
  private loadingViewRef: SystemLoadingView | null = null;

  aboutToAppear() {
    // 订阅状态变化
    this.stateManager.subscribe('login_page', (state: GlobalState) => {
      this.globalState = state;
    });
  }

  aboutToDisappear() {
    // 取消订阅
    this.stateManager.unsubscribe('login_page');
  }

  /**
   * 延迟辅助方法
   */
  private delay(ms: number): Promise<void> {
    return new Promise(resolve => setTimeout(resolve, ms));
  }

  build() {
    Column() {
      if (this.isLoading) {
        SystemLoadingView({ 
          message: this.loadingStage || this.i18nManager.t('auth.login'), 
          progressType: 'determinate',
          showProgress: true,
          externalProgress: this.loadingProgress
        })
      } else {
        Column({ space: 32 }) {
          // 标题
          Column({ space: 16 }) {
            Text(this.i18nManager.t('app.name'))
              .fontSize(28)
              .fontWeight(FontWeight.Bold)
              .fontColor($r('sys.color.ohos_id_color_text_primary'))

            Text(this.i18nManager.t('app.description'))
              .fontSize(16)
              .fontColor($r('sys.color.ohos_id_color_text_secondary'))
          }

          // 登录模式切换
          Row() {
            Button(this.i18nManager.t('auth.tempUser'))
              .type(this.loginMode === 'temp' ? ButtonType.Capsule : ButtonType.Normal)
              .backgroundColor(this.loginMode === 'temp' ?
              $r('sys.color.ohos_id_color_emphasize') :
              $r('sys.color.ohos_id_color_background'))
              .fontColor(this.loginMode === 'temp' ? Color.White :
              $r('sys.color.ohos_id_color_text_primary'))
              .fontSize(14)
              .layoutWeight(1)
              .onClick(() => {
                this.loginMode = 'temp';
              })

            Button(this.i18nManager.t('auth.login'))
              .type(this.loginMode === 'login' ? ButtonType.Capsule : ButtonType.Normal)
              .backgroundColor(this.loginMode === 'login' ?
              $r('sys.color.ohos_id_color_emphasize') :
              $r('sys.color.ohos_id_color_background'))
              .fontColor(this.loginMode === 'login' ? Color.White :
              $r('sys.color.ohos_id_color_text_primary'))
              .fontSize(14)
              .layoutWeight(1)
              .margin({ left: 8 })
              .onClick(() => {
                this.loginMode = 'login';
              })
          }
          .width('100%')

          // 输入区域
          if (this.loginMode === 'temp') {
            this.buildTempUserForm()
          } else {
            this.buildUserLoginForm()
          }
        }
        .width('100%')
        .justifyContent(FlexAlign.Center)
        .layoutWeight(1)
        .padding(32)
      }
    }
    .width('100%')
    .height('100%')
    .backgroundColor($r('sys.color.ohos_id_color_sub_background'))
  }

  /**
   * 创建临时用户
   */
  private async createTempUser() {
    if (!this.nickname.trim()) {
      SystemToast.showWarning(this.i18nManager.t('auth.nickname'), this.getUIContext());
      return;
    }

    this.isLoading = true;
    this.loadingProgress = 0;

    try {
      // 阶段1：准备创建临时用户
      this.loadingStage = this.i18nManager.t('common.loading');
      this.loadingProgress = 20;
      await this.delay(300);

      // 阶段2：发送创建请求
      this.loadingStage = this.i18nManager.t('auth.register');
      this.loadingProgress = 50;
      const request: CreateTempRequest = {
        nickname: this.nickname.trim()
      };
      const response = await ApiClient.createTempUser(request);
      
      if (response.code === 0 && response.data) {
        // 阶段3：处理登录成功
        this.loadingStage = this.i18nManager.t('auth.loginSuccess');
        this.loadingProgress = 80;
        await this.handleAuthSuccess(response.data);
        this.loadingProgress = 100;
      } else {
        SystemToast.showError(response.message || this.i18nManager.t('auth.registerFailed'), this.getUIContext());
      }
    } catch (error) {
      SystemToast.showError(error instanceof Error ? error.message : this.i18nManager.t('auth.registerFailed'), this.getUIContext());
    } finally {
      this.isLoading = false;
      this.loadingStage = '';
      this.loadingProgress = 0;
    }
  }

  /**
   * 用户登录
   * 使用 SHA256(password) 进行密码哈希
   */
  private async userLogin() {
    if (!this.loginId.trim() || !this.password.trim()) {
      SystemToast.showWarning('请输入邮箱和密码', this.getUIContext());
      return;
    }

    this.isLoading = true;
    this.loadingProgress = 0;

    try {
      // 阶段1：加密密码
      this.loadingStage = '正在加密密码...';
      this.loadingProgress = 30;
      const passwordHash = await PasswordSecurityUtil.calculatePasswordHash(this.password);

      // 阶段2：发送登录请求
      this.loadingStage = '正在验证登录信息...';
      this.loadingProgress = 60;
      const loginRequest: LoginRequest = {
        login_id: this.loginId,
        password_hash: passwordHash
      };

      const response = await ApiClient.login(loginRequest);
      if (response.code === 0 && response.data) {
        // 阶段3：登录成功，正在初始化
        this.loadingStage = '登录成功，正在初始化...';
        this.loadingProgress = 90;
        
        // 清除敏感数据
        PasswordSecurityUtil.clearSensitiveString(this.password);
        PasswordSecurityUtil.clearSensitiveString(passwordHash);
        
        await this.handleAuthSuccess(response.data);
        this.loadingProgress = 100;
      } else {
        SystemToast.showError(response.message || '登录失败', this.getUIContext());
      }
    } catch (error) {
      SystemToast.showError(error instanceof Error ? error.message : '登录失败', this.getUIContext());
    } finally {
      this.isLoading = false;
      this.loadingStage = '';
      this.loadingProgress = 0;
    }
  }

  /**
   * 处理认证成功
   */
  private async handleAuthSuccess(authData: AuthData) {
    await PreferencesUtil.setObject('user_info', authData);
    await PreferencesUtil.set('access_token', authData.access_token);

    // 更新全局状态
    const userInfo: MyProfileData = {
      user_id: authData.user_id,
      nickname: authData.nickname,
      location_name: '',
      is_online: true,
      created_at: new Date().toISOString()
    };
    this.stateManager.updateUser(userInfo);
    await HttpUtil.setToken(authData.access_token);

    // 连接WebSocket
    try {
      await this.webSocketService.connect(authData.access_token);
    } catch (error) {
      hilog.warn(DOMAIN_NUMBER, TAG, `WebSocket连接失败: ${JSON.stringify(error)}`);
    }

    // 导航到主页
    const routerInstance = this.getUIContext().getRouter();
    await routerInstance.replaceUrl({ url: `pages/${PageNames.INDEX}` });
    SystemToast.showSuccess('登录成功', this.getUIContext());
  }

  @Builder
  private buildTempUserForm() {
    Column({ space: 16 }) {
      TextInput({ placeholder: '请输入昵称' })
        .fontSize(16)
        .padding(16)
        .backgroundColor($r('sys.color.ohos_id_color_background'))
        .borderRadius(8)
        .onChange((value: string) => {
          this.nickname = value;
        })

      Button('开始体验')
        .type(ButtonType.Capsule)
        .backgroundColor($r('sys.color.ohos_id_color_emphasize'))
        .fontColor(Color.White)
        .fontSize(16)
        .width('100%')
        .height(48)
        .onClick(() => this.createTempUser())

      Text('创建临时账户，无需注册即可体验')
        .fontSize(14)
        .fontColor($r('sys.color.ohos_id_color_text_secondary'))
        .textAlign(TextAlign.Center)
    }
    .width('100%')
  }

  @Builder
  private buildUserLoginForm() {
    Column({ space: 16 }) {
      TextInput({ placeholder: '请输入邮箱或手机号' })
        .fontSize(16)
        .padding(16)
        .backgroundColor($r('sys.color.ohos_id_color_background'))
        .borderRadius(8)
        .type(InputType.Email)
        .onChange((value: string) => {
          this.loginId = value;
        })

      TextInput({ placeholder: '请输入密码' })
        .fontSize(16)
        .padding(16)
        .backgroundColor($r('sys.color.ohos_id_color_background'))
        .borderRadius(8)
        .type(InputType.Password)
        .onChange((value: string) => {
          this.password = value;
        })

      Button('登录')
        .type(ButtonType.Capsule)
        .backgroundColor($r('sys.color.ohos_id_color_emphasize'))
        .fontColor(Color.White)
        .fontSize(16)
        .width('100%')
        .height(48)
        .onClick(() => this.userLogin())

      Text('暂不支持注册，请使用临时账户体验')
        .fontSize(14)
        .fontColor($r('sys.color.ohos_id_color_text_secondary'))
        .textAlign(TextAlign.Center)
    }
    .width('100%')
  }
}
