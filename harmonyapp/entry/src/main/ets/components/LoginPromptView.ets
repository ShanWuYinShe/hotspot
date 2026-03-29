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
import { SafeAreaUtils } from '../core/utils/SafeAreaUtils';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { KeyboardDismissManager } from '../core/utils/KeyboardDismissManager';
import { AppThemeManager } from '../core/theme/AppThemeManager';
import { KEY_RESOLVED_THEME_ID } from '../core/theme/AppThemeTypes';

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
  @State isQuickInputFocused: boolean = false;
  @StorageLink('safeAreaBottom') safeAreaBottom: number = 0;
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'default_clean_light';

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  build() {
    Column({ space: 20 }) {
      // 图标
      Text() {
        SymbolSpan($r(this.icon))
          .fontSize(56)
          .fontColor([this.getPalette().iconMuted])
      }
      .margin({ bottom: 4 })

      // 主标题
      Text(this.title)
        .fontSize(DesignConstants.FONT.SIZE.XL)
        .fontWeight(FontWeight.Bold)
        .fontColor(this.getPalette().textPrimary)
        .textAlign(TextAlign.Center)

      // 副标题
      Text(this.subtitle)
        .fontSize(DesignConstants.FONT.SIZE.MD)
        .fontColor(this.getPalette().textSecondary)
        .textAlign(TextAlign.Center)
        .lineHeight(22)

      // 按钮区（胶囊化加阴影）
      Column({ space: 12 }) {
        Button('快速体验（无需注册）')
          .type(ButtonType.Capsule)
          .backgroundColor(this.getPalette().accentBackground)
          .fontColor(this.getPalette().accentText)
          .fontSize(DesignConstants.FONT.SIZE.LG)
          .width('100%')
          .height(48)
          .onClick(() => {
            this.showQuickStart = true;
          })

        Button('登录 / 注册')
          .type(ButtonType.Capsule)
          .backgroundColor(this.getPalette().buttonSecondaryBackground)
          .fontColor(this.getPalette().buttonSecondaryText)
          .fontSize(DesignConstants.FONT.SIZE.LG)
          .border({ width: 1, color: this.getPalette().borderColor })
          .width('100%')
          .height(48)
          .onClick(() => {
            this.navigateToLogin();
          })
      }
      .width('80%')
      .margin({ top: 8 })

      // 底部提示
      if (this.hint.length > 0) {
        Text(this.hint)
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .fontColor(this.getPalette().textSecondary)
          .textAlign(TextAlign.Center)
          .margin({ top: 4 })
      }
    }
    .width('100%')
    .height('100%')
    .justifyContent(FlexAlign.Center)
    .padding(32)
    .backgroundColor(this.getPalette().pageBackground)
    .expandSafeArea([SafeAreaType.SYSTEM], [SafeAreaEdge.TOP])
    .bindSheet($$this.showQuickStart, this.buildQuickStartSheet(), {
      height: SheetSize.FIT_CONTENT,
      showClose: true,
      dragBar: true,
      backgroundColor: this.getPalette().sheetBackground,
      keyboardAvoidMode: SheetKeyboardAvoidMode.TRANSLATE_AND_SCROLL
    })
  }

  @Builder
  private buildQuickStartSheet() {
    Scroll() {
      Column({ space: DesignConstants.SPACING.LG }) {
        Text('快速体验')
          .fontSize(DesignConstants.FONT.SIZE.XL)
          .fontWeight(FontWeight.Bold)
          .fontColor(this.getPalette().textPrimary)

        Text('无需注册，输入昵称即可快速体验')
          .fontSize(DesignConstants.FONT.SIZE.MD)
          .fontColor(this.getPalette().textSecondary)

        TextInput({ placeholder: '请输入你的昵称' })
          .fontSize(DesignConstants.FONT.SIZE.LG)
          .padding(DesignConstants.SPACING.LG)
          .backgroundColor(this.getPalette().surfaceSubtle)
          .borderRadius(DesignConstants.RADIUS.XL)
          .shadow(this.getPalette().controlShadow)
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
          .type(ButtonType.Capsule)
          .backgroundColor(this.getPalette().accentBackground)
          .fontColor(this.getPalette().accentText)
          .fontSize(DesignConstants.FONT.SIZE.LG)
          .width('100%')
          .height(48)
          .enabled(!this.isQuickStarting)
          .opacity(this.isQuickStarting ? DesignConstants.OPACITY.LOADING : 1)
          .onClick(() => {
            this.dismissQuickStartKeyboard();
            this.startQuickExperience();
          })

        Text('临时用户可使用大部分功能，随时可升级为完整账号')
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .fontColor(this.getPalette().textSecondary)
          .textAlign(TextAlign.Center)
      }
      .alignItems(HorizontalAlign.Start)
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
    .backgroundColor(this.getPalette().surfaceBackground)
    .shadow(this.getPalette().elevatedShadow)
    .border({ width: 1, color: this.getPalette().borderColor })
  }

  private dismissQuickStartKeyboard(): void {
    if (!this.isQuickInputFocused) {
      return;
    }
    KeyboardDismissManager.getInstance().dismiss();
    this.isQuickInputFocused = false;
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
