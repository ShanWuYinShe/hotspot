/**
 * 登录/注册页面
 * 三种模式：临时用户、登录、注册
 * 认证成功后获取完整 profile（含 user_type）
 */

import { GlobalStateManager } from '../../../core/state/GlobalStateManager';
import { ApiClient } from '../../../core/api/ApiClient';
import { PreferencesUtil } from '../../../core/storage/PreferencesUtil';
import { HttpUtil } from '../../../core/network/HttpUtil';
import { WebSocketService } from '../../../core/network/WebSocketService';
import { PageNames } from '../../../constants/AppConstants';
import { SystemLoadingView } from '../../../components/SystemLoadingView';
import { SystemToast } from '../../../components/SystemToast';
import { PasswordSecurityUtil } from '../../../core/security/PasswordSecurityUtil';
import { AuthData, GuestRequest, MyProfileData, LoginRequest, RegisterRequest } from '../../../core/types/api';
import { AuthAccountManager } from '../../../core/state/AuthAccountManager';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { DesignConstants } from '../../../constants/DesignConstants';

const DOMAIN_NUMBER: number = 0xFF02;
const TAG: string = 'LoginPage';

@Entry
@Component
struct LoginPage {
  @StorageLink('isLoggedIn') isLoggedIn: boolean = false;

  @State nickname: string = '';
  @State loginId: string = '';
  @State password: string = '';
  @State registerNickname: string = '';
  @State registerLoginId: string = '';
  @State registerPassword: string = '';
  @State registerPasswordConfirm: string = '';
  @State isLoading: boolean = false;
  @State loadingStage: string = '';
  @State loadingProgress: number = 0;
  @State loginMode: 'temp' | 'login' | 'register' = 'temp';
  @State redirectUrl: string = '';
  @State redirectAction: string = '';

  private stateManager = GlobalStateManager.getInstance();
  private webSocketService = WebSocketService.getInstance();

  private delay(ms: number): Promise<void> {
    return new Promise(resolve => setTimeout(resolve, ms));
  }

  aboutToAppear() {
    try {
      const routerInstance = this.getUIContext().getRouter();
      const params = routerInstance.getParams() as Record<string, Object>;
      if (params) {
        const redirect = params.redirectUrl as string;
        if (redirect) {
          this.redirectUrl = redirect;
        }
        const action = params.action as string;
        if (action) {
          this.redirectAction = action;
        }
      }
    } catch (error) {
      // ignore
    }
  }

  build() {
    Column() {
      if (this.isLoading) {
        SystemLoadingView({
          message: this.loadingStage || '加载中...',
          progressType: 'determinate',
          showProgress: true,
          externalProgress: this.loadingProgress
        })
      } else {
        // 返回按钮（游客可直接返回浏览）
        Row() {
          Text('← 返回浏览')
            .fontSize(14)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
            .onClick(async () => {
              try {
                const routerInstance = this.getUIContext().getRouter();
                await routerInstance.back();
              } catch (error) {
                // fallback: 跳转主页
                try {
                  const routerInstance = this.getUIContext().getRouter();
                  await routerInstance.replaceUrl({ url: `features/main/pages/${PageNames.INDEX}` });
                } catch (e) {
                  hilog.error(DOMAIN_NUMBER, TAG, `跳转失败: ${e}`);
                }
              }
            })
        }
        .width('100%')
        .padding({ top: 16, left: 16, right: 16 })

        List() {
          // 标题区域
          ListItem() {
            Column({ space: 8 }) {
              Text($r('app.string.app_name'))
                .fontSize(28)
                .fontWeight(FontWeight.Bold)
                .fontColor($r('sys.color.ohos_id_color_text_primary'))

              Text($r('app.string.app_description'))
                .fontSize(16)
                .fontColor($r('sys.color.ohos_id_color_text_secondary'))
            }
            .width('100%')
            .alignItems(HorizontalAlign.Start)
            .padding({ top: 24, bottom: 32 })
          }

          // 模式切换
          ListItem() {
            Row({ space: 4 }) {
              this.ModeTab('快速体验', 'temp')
              this.ModeTab('登录', 'login')
              this.ModeTab('注册', 'register')
            }
            .width('100%')
            .margin({ bottom: 24 })
          }

          // 表单区域
          ListItem() {
            if (this.loginMode === 'temp') {
              this.buildTempUserForm()
            } else if (this.loginMode === 'login') {
              this.buildLoginForm()
            } else {
              this.buildRegisterForm()
            }
          }
        }
        .width('100%')
        .layoutWeight(1)
        .padding({ left: 32, right: 32 })
      }
    }
    .width('100%')
    .height('100%')
    .backgroundColor($r('sys.color.ohos_id_color_sub_background'))
  }

  @Builder
  private ModeTab(label: string, mode: string) {
    Button(label)
      .type(this.loginMode === mode ? ButtonType.Capsule : ButtonType.Normal)
      .backgroundColor(this.loginMode === mode
        ? $r('sys.color.ohos_id_color_emphasize')
        : $r('sys.color.ohos_id_color_component_normal'))
      .fontColor(this.loginMode === mode ? Color.White : $r('sys.color.ohos_id_color_text_primary'))
      .fontSize(14)
      .height(36)
      .layoutWeight(1)
      .onClick(() => {
        if (mode === 'temp') {
          this.loginMode = 'temp';
        } else if (mode === 'login') {
          this.loginMode = 'login';
        } else {
          this.loginMode = 'register';
        }
      })
  }

  @Builder
  private buildTempUserForm() {
    Column({ space: 16 }) {
      Text('无需注册，输入昵称即可快速体验')
        .fontSize(14)
        .fontColor($r('sys.color.ohos_id_color_text_secondary'))

      TextInput({ placeholder: '请输入你的昵称' })
        .fontSize(16)
        .padding(16)
        .backgroundColor($r('sys.color.ohos_id_color_background'))
        .borderRadius(12)
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

      Text('临时用户可使用大部分功能，随时可升级为完整账号')
        .fontSize(12)
        .fontColor($r('sys.color.ohos_id_color_text_secondary'))
        .textAlign(TextAlign.Center)
    }
    .width('100%')
  }

  @Builder
  private buildLoginForm() {
    Column({ space: 16 }) {
      TextInput({ placeholder: '邮箱 / 用户名' })
        .fontSize(16)
        .padding(16)
        .backgroundColor($r('sys.color.ohos_id_color_background'))
        .borderRadius(12)
        .type(InputType.Email)
        .onChange((value: string) => {
          this.loginId = value;
        })

      TextInput({ placeholder: '密码' })
        .fontSize(16)
        .padding(16)
        .backgroundColor($r('sys.color.ohos_id_color_background'))
        .borderRadius(12)
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

      Text('没有账号？切换到"注册"或"快速体验"')
        .fontSize(12)
        .fontColor($r('sys.color.ohos_id_color_text_secondary'))
        .textAlign(TextAlign.Center)
    }
    .width('100%')
  }

  @Builder
  private buildRegisterForm() {
    Column({ space: 16 }) {
      TextInput({ placeholder: '昵称（必填）' })
        .fontSize(16)
        .padding(16)
        .backgroundColor($r('sys.color.ohos_id_color_background'))
        .borderRadius(12)
        .onChange((value: string) => {
          this.registerNickname = value;
        })

      TextInput({ placeholder: '邮箱 / 用户名（登录凭证）' })
        .fontSize(16)
        .padding(16)
        .backgroundColor($r('sys.color.ohos_id_color_background'))
        .borderRadius(12)
        .type(InputType.Email)
        .onChange((value: string) => {
          this.registerLoginId = value;
        })

      TextInput({ placeholder: '密码（至少8位）' })
        .fontSize(16)
        .padding(16)
        .backgroundColor($r('sys.color.ohos_id_color_background'))
        .borderRadius(12)
        .type(InputType.Password)
        .onChange((value: string) => {
          this.registerPassword = value;
        })

      TextInput({ placeholder: '确认密码' })
        .fontSize(16)
        .padding(16)
        .backgroundColor($r('sys.color.ohos_id_color_background'))
        .borderRadius(12)
        .type(InputType.Password)
        .onChange((value: string) => {
          this.registerPasswordConfirm = value;
        })

      Button('注册')
        .type(ButtonType.Capsule)
        .backgroundColor($r('sys.color.ohos_id_color_emphasize'))
        .fontColor(Color.White)
        .fontSize(16)
        .width('100%')
        .height(48)
        .onClick(() => this.userRegister())

      Text('已有账号？切换到"登录"标签')
        .fontSize(12)
        .fontColor($r('sys.color.ohos_id_color_text_secondary'))
        .textAlign(TextAlign.Center)
    }
    .width('100%')
  }

  // ==================== 认证逻辑 ====================

  private async createTempUser() {
    if (!this.nickname.trim()) {
      SystemToast.showWarning('请输入昵称', this.getUIContext());
      return;
    }

    this.isLoading = true;
    this.loadingProgress = 0;

    try {
      this.loadingStage = '正在创建临时用户...';
      this.loadingProgress = 30;

      const request: GuestRequest = {
        nickname: this.nickname.trim(),
        client_type: 'app'
      };
      const response = await ApiClient.createTempUser(request);

      if (response.code === 0 && response.data) {
        this.loadingStage = '初始化中...';
        this.loadingProgress = 70;
        await this.handleAuthSuccess(response.data);
        this.loadingProgress = 100;
      } else {
        SystemToast.showError(response.message || '创建失败', this.getUIContext());
      }
    } catch (error) {
      SystemToast.showError(error instanceof Error ? error.message : '创建失败', this.getUIContext());
    } finally {
      this.isLoading = false;
      this.loadingStage = '';
      this.loadingProgress = 0;
    }
  }

  private async userLogin() {
    if (!this.loginId.trim() || !this.password.trim()) {
      SystemToast.showWarning('请输入邮箱和密码', this.getUIContext());
      return;
    }

    this.isLoading = true;
    this.loadingProgress = 0;

    try {
      this.loadingStage = '正在验证登录信息...';
      this.loadingProgress = 40;

      const loginRequest: LoginRequest = {
        login_id: this.loginId,
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

  private async userRegister() {
    if (!this.registerNickname.trim()) {
      SystemToast.showWarning('请输入昵称', this.getUIContext());
      return;
    }
    if (!this.registerLoginId.trim()) {
      SystemToast.showWarning('请输入邮箱或用户名', this.getUIContext());
      return;
    }
    if (this.registerPassword.length < 8) {
      SystemToast.showWarning('密码至少需要8个字符', this.getUIContext());
      return;
    }
    if (this.registerPassword !== this.registerPasswordConfirm) {
      SystemToast.showWarning('两次输入的密码不一致', this.getUIContext());
      return;
    }

    this.isLoading = true;
    this.loadingProgress = 0;

    try {
      this.loadingStage = '正在注册...';
      this.loadingProgress = 40;

      const registerRequest: RegisterRequest = {
        nickname: this.registerNickname.trim(),
        login_id: this.registerLoginId.trim(),
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
        SystemToast.showError(response.message || '注册失败', this.getUIContext());
      }
    } catch (error) {
      SystemToast.showError(error instanceof Error ? error.message : '注册失败', this.getUIContext());
    } finally {
      this.isLoading = false;
      this.loadingStage = '';
      this.loadingProgress = 0;
    }
  }

  /**
   * 认证成功后的统一处理
   * 1. 保存 token 到多账户管理器
   * 2. 获取完整 profile（含 user_type）
   * 3. 连接 WebSocket
   * 4. 跳转
   */
  private async handleAuthSuccess(authData: AuthData) {
    // 保存多账户
    await AuthAccountManager.upsertAndActivate(authData);
    await HttpUtil.setToken(authData.access_token);

    // 先用 authData 设置基础用户信息，保证 isLoggedIn 立刻为 true
    const basicUserInfo: MyProfileData = {
      user_id: authData.user_id,
      nickname: authData.nickname,
      avatar_url: '',
      bio: '',
      tags: [],
      email: '',
      phone: '',
      location: {
        latitude: 0,
        longitude: 0,
        location_name: ''
      },
      is_online: true,
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

    // 拉取完整 profile（含 user_type 等完整信息）
    try {
      const profileResp = await ApiClient.getMyProfile();
      if (profileResp.code === 0 && profileResp.data) {
        this.stateManager.updateUser(profileResp.data);
        hilog.info(DOMAIN_NUMBER, TAG, `用户类型: ${profileResp.data.user_type || 'unknown'}`);
      }
    } catch (error) {
      hilog.warn(DOMAIN_NUMBER, TAG, `获取用户资料失败，使用基础信息: ${JSON.stringify(error)}`);
    }

    // 连接 WebSocket
    try {
      await this.webSocketService.connect(authData.access_token);
    } catch (error) {
      hilog.warn(DOMAIN_NUMBER, TAG, `WebSocket连接失败: ${JSON.stringify(error)}`);
    }

    // 跳转
    try {
      const routerInstance = this.getUIContext().getRouter();
      const targetUrl = this.redirectUrl || `features/main/pages/${PageNames.INDEX}`;
      await routerInstance.replaceUrl({ url: targetUrl });
      SystemToast.showSuccess('欢迎！', this.getUIContext());
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `跳转主页失败: ${error}`);
    }
  }
}
