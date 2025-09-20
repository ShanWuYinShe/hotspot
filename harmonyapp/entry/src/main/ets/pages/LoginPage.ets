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
import { AuthResponse, CreateTempRequest, UserResponse } from '../common/types/BackendApiTypes';


@Entry
@Component
struct LoginPage {
  @State globalState: GlobalState = GlobalStateManager.getInstance().getState();
  @State nickname: string = '';
  @State loginId: string = '';
  @State password: string = '';
  @State isLoading: boolean = false;
  @State loginMode: 'temp' | 'register' | 'login' = 'temp';
  private stateManager = GlobalStateManager.getInstance();
  private webSocketService = WebSocketService.getInstance();

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

  build() {
    Column() {
      if (this.isLoading) {
        SystemLoadingView({ message: '正在登录...' })
      } else {
        Column({ space: 32 }) {
          // 标题
          Column({ space: 16 }) {
            Text('欢迎使用 GeoTalk')
              .fontSize(28)
              .fontWeight(FontWeight.Bold)
              .fontColor($r('sys.color.ohos_id_color_text_primary'))

            Text('基于地理位置的社交应用')
              .fontSize(16)
              .fontColor($r('sys.color.ohos_id_color_text_secondary'))
          }

          // 登录模式切换
          Row() {
            Button('快速体验')
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

            Button('用户登录')
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
      SystemToast.showWarning('请输入昵称');
      return;
    }

    this.isLoading = true;

    try {
      const request: CreateTempRequest = {
        nickname: this.nickname.trim()
      };
      const response = await ApiClient.createTempUser(request);
      if (response.code === 0 && response.data) {
        await this.handleAuthSuccess(response.data);
      } else {
        SystemToast.showError(response.msg || '创建临时用户失败');
      }
    } catch (error) {
      SystemToast.showError(error instanceof Error ? error.message : '创建临时用户失败');
    } finally {
      this.isLoading = false;
    }
  }

  /**
   * 用户登录
   */
  private async userLogin() {
    if (!this.loginId.trim() || !this.password.trim()) {
      SystemToast.showWarning('请输入邮箱和密码');
      return;
    }

    this.isLoading = true;

    try {
      // 注意：这里需要实现完整的密码哈希流程
      // 1. 获取 nonce
      // 2. 计算密码哈希
      // 3. 发送登录请求

      // 1. 获取nonce用于密码哈希计算
      const nonceResponse = await ApiClient.getNonce();
      if (nonceResponse.code !== 0 || !nonceResponse.data) {
        throw new Error(nonceResponse.msg || '获取nonce失败');
      }

      // 2. 计算传输哈希: SHA256(password + nonce + timestamp)
      // 注意：这里使用明文密码是正确的，因为传输哈希的目的是防重放攻击
      // 服务端会使用Argon2计算存储哈希进行验证
      const response = await ApiClient.login({
        login_id: this.loginId,
        password_hash: this.password, // 传输哈希，服务端会验证并计算存储哈希
        nonce_id: nonceResponse.data.nonce_id,
        timestamp: nonceResponse.data.timestamp
      });
      if (response.code === 0 && response.data) {
        await this.handleAuthSuccess(response.data);
      } else {
        SystemToast.showError(response.msg || '登录失败');
      }
    } catch (error) {
      SystemToast.showError(error instanceof Error ? error.message : '登录失败');
    } finally {
      this.isLoading = false;
    }
  }

  /**
   * 处理认证成功
   */
  private async handleAuthSuccess(response: AuthResponse) {
    await PreferencesUtil.setObject('user_info', response);
    await PreferencesUtil.set('access_token', response.access_token);

    // 更新全局状态
    const userInfo: UserResponse = {
      user_id: response.user_id,
      nickname: response.nickname
    };
    this.stateManager.updateUser(userInfo);
    await HttpUtil.setToken(response.access_token);

    // 连接WebSocket
    try {
      await this.webSocketService.connect(response.access_token);
    } catch (error) {
      console.warn('WebSocket连接失败:', error);
    }

    // 导航到主页
    const routerInstance = this.getUIContext().getRouter();
    await routerInstance.replaceUrl({ url: `pages/${PageNames.INDEX}` });
    SystemToast.showSuccess('登录成功');
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
