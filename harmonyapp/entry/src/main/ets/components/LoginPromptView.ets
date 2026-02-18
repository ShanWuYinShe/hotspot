/**
 * LoginPromptView - 可复用的登录提示组件
 * 用于需要登录才能访问的页面/标签，提供统一的未登录状态展示
 * - "快速体验"：底部弹窗内完成临时用户创建（无需跳转页面）
 * - "登录 / 注册"：跳转到 LoginPage
 */

import { DesignConstants } from '../constants/DesignConstants';
import { PageNames } from '../constants/AppConstants';
import { AppRouter } from '../core/router/AppRouter';
import { AppIcon } from './SystemIcon';
import { SystemToast } from './SystemToast';
import { ApiClient } from '../core/api/ApiClient';
import { AuthAccountManager } from '../core/state/AuthAccountManager';
import { HttpUtil } from '../core/network/HttpUtil';
import { WebSocketService } from '../core/network/WebSocketService';
import { GlobalStateManager } from '../core/state/GlobalStateManager';
import { AuthData, GuestRequest, MyProfile } from '../core/types/api';
import { hilog } from '@kit.PerformanceAnalysisKit';

const DOMAIN_NUMBER: number = 0xFF20;
const TAG: string = 'LoginPromptView';

@Reusable
@Component
export struct LoginPromptView {
  /** 主标题，如 "附近动态需要登录" */
  @Prop title: string = '此功能需要登录';
  /** 副标题说明 */
  @Prop subtitle: string = '登录后即可查看完整内容';
  /** 底部提示文字 */
  @Prop hint: string = '';
  /** 图标（默认锁图标） */
  @Prop icon: AppIcon = AppIcon.PERSON;

  /** 快速体验弹窗状态 */
  @State showQuickStart: boolean = false;
  @State quickNickname: string = '';
  @State isQuickStarting: boolean = false;

  build() {
    Column({ space: 20 }) {
      // 图标
      Text() {
        SymbolSpan($r(this.icon))
          .fontSize(56)
          .fontColor([$r('sys.color.ohos_id_color_text_secondary')])
      }
      .margin({ bottom: 4 })

      // 主标题
      Text(this.title)
        .fontSize(DesignConstants.FONT.SIZE.XL)
        .fontWeight(FontWeight.Bold)
        .fontColor($r('sys.color.ohos_id_color_text_primary'))
        .textAlign(TextAlign.Center)

      // 副标题
      Text(this.subtitle)
        .fontSize(DesignConstants.FONT.SIZE.MD)
        .fontColor($r('sys.color.ohos_id_color_text_secondary'))
        .textAlign(TextAlign.Center)
        .lineHeight(22)

      // 按钮区
      Column({ space: 12 }) {
        Button('快速体验（无需注册）')
          .type(ButtonType.Capsule)
          .backgroundColor($r('sys.color.ohos_id_color_emphasize'))
          .fontColor(Color.White)
          .fontSize(DesignConstants.FONT.SIZE.LG)
          .width('75%')
          .height(44)
          .onClick(() => {
            this.showQuickStart = true;
          })

        Button('登录 / 注册')
          .type(ButtonType.Capsule)
          .backgroundColor($r('sys.color.ohos_id_color_component_normal'))
          .fontColor($r('sys.color.ohos_id_color_text_primary'))
          .fontSize(DesignConstants.FONT.SIZE.LG)
          .width('75%')
          .height(44)
          .onClick(() => {
            this.navigateToLogin();
          })
      }
      .margin({ top: 8 })

      // 底部提示
      if (this.hint.length > 0) {
        Text(this.hint)
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .fontColor($r('sys.color.ohos_id_color_text_secondary'))
          .textAlign(TextAlign.Center)
          .margin({ top: 4 })
      }
    }
    .width('100%')
    .height('100%')
    .justifyContent(FlexAlign.Center)
    .padding(32)
    .backgroundColor($r('app.color.bg_primary'))
    .expandSafeArea([SafeAreaType.SYSTEM], [SafeAreaEdge.TOP])
    .bindSheet($$this.showQuickStart, this.buildQuickStartSheet(), {
      height: SheetSize.FIT_CONTENT,
      showClose: true,
      dragBar: true,
      backgroundColor: $r('app.color.bg_secondary')
    })
  }

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
        .backgroundColor($r('sys.color.ohos_id_color_sub_background'))
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
        await this.handleQuickStartSuccess(response.data);
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

  /** 临时用户创建成功后的认证初始化 */
  private async handleQuickStartSuccess(authData: AuthData): Promise<void> {
    await AuthAccountManager.upsertAndActivate(authData);
    await HttpUtil.setToken(authData.access_token);

    const stateManager = GlobalStateManager.getInstance();
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
    stateManager.updateUser(basicUserInfo);

    try {
      const profileResp = await ApiClient.getMyProfile();
      if (profileResp.code === 0 && profileResp.data) {
        stateManager.updateUser(profileResp.data);
      }
    } catch (error) {
      hilog.warn(DOMAIN_NUMBER, TAG, `获取用户资料失败: ${JSON.stringify(error)}`);
    }

    try {
      await WebSocketService.getInstance().connect(authData.access_token);
    } catch (error) {
      hilog.warn(DOMAIN_NUMBER, TAG, `WebSocket连接失败: ${JSON.stringify(error)}`);
    }
  }

  private navigateToLogin(): void {
    AppRouter.push(PageNames.LOGIN);
  }
}
