/**
 * 极简登录页面
 * 重构：使用 List 优化表单布局
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
import { AppAuthData, CreateTempRequest, MyProfileData, LoginRequest } from '../../../core/types/api';
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
  @State isLoading: boolean = false;
  @State loadingStage: ResourceStr = '';
  @State loadingProgress: number = 0;
  @State loginMode: 'temp' | 'register' | 'login' = 'temp';
  @State redirectUrl: string = '';
  
  private stateManager = GlobalStateManager.getInstance();
  private webSocketService = WebSocketService.getInstance();

  private delay(ms: number): Promise<void> {
    return new Promise(resolve => setTimeout(resolve, ms));
  }

  aboutToAppear() {
    try {
      const routerInstance = this.getUIContext().getRouter();
      const params = routerInstance.getParams() as Record<string, Object>;
      const redirect = params?.redirectUrl as string;
      if (redirect) {
        this.redirectUrl = redirect;
      }
    } catch (error) {
      // ignore
    }
  }

  build() {
    Column() {
      if (this.isLoading) {
        SystemLoadingView({ 
          message: this.loadingStage || $r('app.string.auth_login'), 
          progressType: 'determinate',
          showProgress: true,
          externalProgress: this.loadingProgress
        })
      } else {
        List() {
          ListItem() {
            Column({ space: 16 }) {
              Text($r('app.string.app_name'))
                .fontSize(28)
                .fontWeight(FontWeight.Bold)
                .fontColor(DesignConstants.COLOR.TEXT_PRIMARY_LIGHT)

              Text($r('app.string.app_description'))
                .fontSize(16)
                .fontColor(DesignConstants.COLOR.TEXT_SECONDARY_LIGHT)
            }
            .width('100%')
            .alignItems(HorizontalAlign.Start)
            .padding({ top: 48, bottom: 48 })
          }

          ListItem() {
            Row() {
              Button($r('app.string.auth_temp_user'))
                .type(this.loginMode === 'temp' ? ButtonType.Capsule : ButtonType.Normal)
                .backgroundColor(this.loginMode === 'temp' ?
                DesignConstants.COLOR.BRAND_PRIMARY :
                DesignConstants.COLOR.TERTIARY_BG_LIGHT)
                .fontColor(this.loginMode === 'temp' ? Color.White :
                DesignConstants.COLOR.TEXT_PRIMARY_LIGHT)
                .fontSize(14)
                .layoutWeight(1)
                .onClick(() => {
                  this.loginMode = 'temp';
                })

              Button($r('app.string.auth_login'))
                .type(this.loginMode === 'login' ? ButtonType.Capsule : ButtonType.Normal)
                .backgroundColor(this.loginMode === 'login' ?
                DesignConstants.COLOR.BRAND_PRIMARY :
                DesignConstants.COLOR.TERTIARY_BG_LIGHT)
                .fontColor(this.loginMode === 'login' ? Color.White :
                DesignConstants.COLOR.TEXT_PRIMARY_LIGHT)
                .fontSize(14)
                .layoutWeight(1)
                .margin({ left: 8 })
                .onClick(() => {
                  this.loginMode = 'login';
                })
            }
            .width('100%')
            .margin({ bottom: 32 })
          }

          ListItem() {
            if (this.loginMode === 'temp') {
              this.buildTempUserForm()
            } else {
              this.buildUserLoginForm()
            }
          }
        }
        .width('100%')
        .layoutWeight(1)
        .padding(32)
      }
    }
    .width('100%')
    .height('100%')
    .backgroundColor(DesignConstants.COLOR.SECONDARY_BG_LIGHT)
  }

  @Builder
  private buildTempUserForm() {
    Column({ space: 16 }) {
      TextInput({ placeholder: $r('app.string.auth_nickname_placeholder') })
        .fontSize(16)
        .padding(16)
        .backgroundColor(DesignConstants.COLOR.TERTIARY_BG_LIGHT)
        .borderRadius(8)
        .onChange((value: string) => {
          this.nickname = value;
        })

      Button($r('app.string.auth_start_experience'))
        .type(ButtonType.Capsule)
        .backgroundColor(DesignConstants.COLOR.BRAND_PRIMARY)
        .fontColor(Color.White)
        .fontSize(16)
        .width('100%')
        .height(48)
        .onClick(() => this.createTempUser())

      Text($r('app.string.auth_temp_user_desc'))
        .fontSize(14)
        .fontColor(DesignConstants.COLOR.TEXT_SECONDARY_LIGHT)
        .textAlign(TextAlign.Center)
    }
    .width('100%')
  }

  @Builder
  private buildUserLoginForm() {
    Column({ space: 16 }) {
      TextInput({ placeholder: $r('app.string.auth_login_id_placeholder') })
        .fontSize(16)
        .padding(16)
        .backgroundColor(DesignConstants.COLOR.TERTIARY_BG_LIGHT)
        .borderRadius(8)
        .type(InputType.Email)
        .onChange((value: string) => {
          this.loginId = value;
        })

      TextInput({ placeholder: $r('app.string.auth_password_placeholder') })
        .fontSize(16)
        .padding(16)
        .backgroundColor(DesignConstants.COLOR.TERTIARY_BG_LIGHT)
        .borderRadius(8)
        .type(InputType.Password)
        .onChange((value: string) => {
          this.password = value;
        })

      Button($r('app.string.auth_login_btn'))
        .type(ButtonType.Capsule)
        .backgroundColor(DesignConstants.COLOR.BRAND_PRIMARY)
        .fontColor(Color.White)
        .fontSize(16)
        .width('100%')
        .height(48)
        .onClick(() => this.userLogin())

      Text($r('app.string.auth_register_not_supported'))
        .fontSize(14)
        .fontColor(DesignConstants.COLOR.TEXT_SECONDARY_LIGHT)
        .textAlign(TextAlign.Center)
    }
    .width('100%')
  }

  private async createTempUser() {
    if (!this.nickname.trim()) {
      SystemToast.showWarning('请输入昵称', this.getUIContext());
      return;
    }

    this.isLoading = true;
    this.loadingProgress = 0;

    try {
      this.loadingStage = $r('app.string.common_loading');
      this.loadingProgress = 20;
      await this.delay(300);

      this.loadingStage = '正在创建用户...';
      this.loadingProgress = 50;
      const request: CreateTempRequest = {
        nickname: this.nickname.trim()
      };
      const response = await ApiClient.createTempUser(request);
      
      if (response.code === 0 && response.data) {
        this.loadingStage = $r('app.string.auth_login_success');
        this.loadingProgress = 80;
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

  private async userLogin() {
    if (!this.loginId.trim() || !this.password.trim()) {
      SystemToast.showWarning('请输入邮箱和密码', this.getUIContext());
      return;
    }

    this.isLoading = true;
    this.loadingProgress = 0;

    try {
      this.loadingStage = '正在验证登录信息...';
      this.loadingProgress = 60;
      const loginRequest: LoginRequest = {
        login_id: this.loginId,
        password: this.password,
        client_type: 'app'
      };

      const response = await ApiClient.login(loginRequest);
      if (response.code === 0 && response.data) {
        this.loadingStage = '登录成功，正在初始化...';
        this.loadingProgress = 90;
        
        PasswordSecurityUtil.clearSensitiveString(this.password);
        
        await this.handleAuthSuccess(response.data);
        this.loadingProgress = 100;
      } else {
        SystemToast.showError(response.message || $r('app.string.auth_login_failed'), this.getUIContext());
      }
    } catch (error) {
      SystemToast.showError(error instanceof Error ? error.message : '登录失败', this.getUIContext());
    } finally {
      this.isLoading = false;
      this.loadingStage = '';
      this.loadingProgress = 0;
    }
  }

  private async handleAuthSuccess(authData: AppAuthData) {
    // 多账号：保存并激活为当前账号（同时写回 access_token/user_info 以兼容旧逻辑）
    await AuthAccountManager.upsertAndActivate(authData);

    const userInfo: MyProfileData = {
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
    this.stateManager.updateUser(userInfo);
    await HttpUtil.setToken(authData.access_token);

    try {
      await this.webSocketService.connect(authData.access_token);
    } catch (error) {
      hilog.warn(DOMAIN_NUMBER, TAG, `WebSocket连接失败: ${JSON.stringify(error)}`);
    }

    try {
      const routerInstance = this.getUIContext().getRouter();
      const targetUrl = this.redirectUrl || `features/main/pages/${PageNames.INDEX}`;
      await routerInstance.replaceUrl({ url: targetUrl });
      SystemToast.showSuccess('登录成功', this.getUIContext());
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `跳转主页失败: ${error}`);
    }
  }
}
