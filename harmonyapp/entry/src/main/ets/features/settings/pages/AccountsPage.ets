/**
 * 账号管理（多账号切换）
 *
 * - 展示本地保存的账号列表
 * - 支持切换当前账号（切换后重连 WS，并清空本地消息缓存，避免串数据）
 * - 支持删除账号
 */

import { AuthAccountManager } from '../../../core/state/AuthAccountManager';
import { GlobalStateManager, KEY_CURRENT_USER } from '../../../core/state/GlobalStateManager';
import { WebSocketService } from '../../../core/network/WebSocketService';
import { MessageCacheService } from '../../../core/database/MessageCacheService';
import { MyProfile, AuthData } from '../../../core/types/api';
import { SystemToast } from '../../../components/SystemToast';
import { DesignConstants } from '../../../constants/DesignConstants';
import { AppIcon, IconSize, IconButton } from '../../../components/SystemIcon';
import { LoadingUtil } from '../../../core/utils/LoadingUtil';
import { AppRouter } from '../../../core/router/AppRouter';
import { PageNames } from '../../../constants/AppConstants';
import { SystemUiController } from '../../../core/utils/SystemUiController';
import { SafeAreaUtils } from '../../../core/utils/SafeAreaUtils';

@Component
export struct AccountsPage {
  pathStack: NavPathStack = new NavPathStack();
  @StorageLink(KEY_CURRENT_USER) currentUser: MyProfile | null = null;
  @StorageLink('safeAreaBottom') safeAreaBottom: number = 0;

  @State accounts: AuthData[] = [];
  @State currentUserId: string | null = null;
  @State isLoading: boolean = false;

  private stateManager = GlobalStateManager.getInstance();
  private ws = WebSocketService.getInstance();
  private cache = MessageCacheService.getInstance();

  async aboutToAppear() {
    SystemUiController.useSurface({
      baseColor: DesignConstants.COLOR.PRIMARY_BG_LIGHT,      preferDarkContent: true
    });
    await this.load();
  }

  private async load() {
    this.isLoading = true;
    try {
      await LoadingUtil.withMinimumTime(async () => {
        this.accounts = await AuthAccountManager.listAccounts();
        const uid = await AuthAccountManager.getCurrentUserId();
        this.currentUserId = uid !== null ? String(uid) : null;
      });
    } finally {
      this.isLoading = false;
    }
  }

  private getAccountDisplayName(auth: AuthData): string {
    // AuthData 只有 user_id/token 等字段，用 user_id 作为显示名
    return auth.user_id;
  }

  private toProfile(auth: AuthData): MyProfile {
    const nickname = auth.user_id;
    return {
      user_id: auth.user_id,
      nickname: nickname,
      avatar_url: '',
      user_type: 'temp',
      bio: '',
      tags: [],
      email: '',
      phone: '',
      location: {
        point: { latitude: 0, longitude: 0 },
        location_name: ''
      },
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
  }

  private async switchTo(auth: AuthData) {
    try {
      // 1) 清空缓存（避免串数据）
      await this.cache.clearAll();

      // 2) 切换 token
      const next = await AuthAccountManager.switchTo(auth.user_id);
      if (!next) {
        SystemToast.showError('账号不存在', this.getUIContext());
        return;
      }

      // 3) 更新 UI 层 currentUser
      this.stateManager.updateUser(this.toProfile(next));

      // 4) 重连 WS
      this.ws.disconnect();
      try {
        await this.ws.connect(next.access_token);
      } catch {
        // ignore
      }

      SystemToast.showSuccess(`已切换到：${next.user_id}`, this.getUIContext());
      await this.load();
    } catch {
      SystemToast.showError('切换失败', this.getUIContext());
    }
  }

  private async remove(auth: AuthData) {
    try {
      await AuthAccountManager.removeAccount(auth.user_id);
      if (this.currentUserId !== null && this.currentUserId === auth.user_id) {
        // 删除当前账号 => 锁定
        await this.cache.clearAll();
        this.ws.disconnect();
        this.stateManager.updateUser(null);
      }
      SystemToast.showSuccess('已删除账号', this.getUIContext());
      await this.load();
    } catch {
      SystemToast.showError('删除失败', this.getUIContext());
    }
  }

  build() {
    NavDestination() {
    Column() {
      // 标题栏
      Row() {
        IconButton({
          icon: AppIcon.BACK,
          iconSize: IconSize.LARGE,
          iconColor: $r('sys.color.ohos_id_color_text_primary'),
          bgColor: Color.Transparent,
          onButtonClick: () => {
            AppRouter.pop();
          }
        })

        Text('账号管理')
          .fontSize(DesignConstants.FONT.SIZE.XL)
          .fontWeight(FontWeight.Medium)
          .fontColor($r('sys.color.ohos_id_color_text_primary'))
          .layoutWeight(1)
          .textAlign(TextAlign.Center)

        Blank().width(48)
      }
      .width('100%')
      .height(56)
      .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG })
      .backgroundColor($r('app.color.bg_secondary'))
      .expandSafeArea([SafeAreaType.SYSTEM], [SafeAreaEdge.TOP])

      if (this.isLoading) {
        Text('加载中...')
          .fontSize(DesignConstants.FONT.SIZE.MD)
          .fontColor($r('sys.color.ohos_id_color_text_secondary'))
          .margin({ top: 24 })
          .transition(TransitionEffect.OPACITY.animation({ duration: 300, curve: Curve.FastOutSlowIn }))
      } else if (this.accounts.length === 0) {
        Column({ space: 12 }) {
          Text('暂无已保存账号')
            .fontSize(DesignConstants.FONT.SIZE.MD)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))

          Button('去登录/创建访客')
            .type(ButtonType.Capsule)
            .backgroundColor($r('sys.color.ohos_id_color_emphasize'))
            .fontColor(Color.White)
            .height(44)
            .onClick(() => {
              AppRouter.push(PageNames.LOGIN);
            })
        }
        .width('100%')
        .padding(DesignConstants.SPACING.XL)
        .transition(TransitionEffect.OPACITY.animation({ duration: 300, curve: Curve.FastOutSlowIn }))
      } else {
        List({ space: 12 }) {
          ForEach(this.accounts, (acc: AuthData) => {
            ListItem() {
              Row() {
                Column({ space: 4 }) {
                  Row({ space: 8 }) {
                    Text(this.getAccountDisplayName(acc))
                      .fontSize(DesignConstants.FONT.SIZE.LG)
                      .fontColor($r('sys.color.ohos_id_color_text_primary'))
                    if (this.currentUserId === acc.user_id) {
                      Text('当前')
                        .fontSize(DesignConstants.FONT.SIZE.SM)
                        .fontColor(Color.White)
                        .backgroundColor($r('sys.color.ohos_id_color_emphasize'))
                        .borderRadius(DesignConstants.RADIUS.MD)
                        .padding({ left: DesignConstants.SPACING.SM, right: DesignConstants.SPACING.SM, top: 2, bottom: 2 })
                    }
                  }
                  Text(`ID: ${acc.user_id}`)
                    .fontSize(DesignConstants.FONT.SIZE.SM)
                    .fontColor($r('sys.color.ohos_id_color_text_secondary'))
                }
                .layoutWeight(1)
                .alignItems(HorizontalAlign.Start)

                Row({ space: 8 }) {
                  Button('切换')
                    .type(ButtonType.Capsule)
                    .fontSize(DesignConstants.FONT.SIZE.SM)
                    .height(32)
                    .backgroundColor($r('sys.color.ohos_id_color_component_normal'))
                    .fontColor($r('sys.color.ohos_id_color_text_primary'))
                    .onClick(() => this.switchTo(acc))

                  Button('删除')
                    .type(ButtonType.Capsule)
                    .fontSize(DesignConstants.FONT.SIZE.SM)
                    .height(32)
                    .backgroundColor($r('app.color.color_danger'))
                    .fontColor(Color.White)
                    .onClick(() => this.remove(acc))
                }
              }
              .width('100%')
              .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, top: DesignConstants.SPACING.MD, bottom: DesignConstants.SPACING.MD })
            }
            .backgroundColor($r('app.color.bg_secondary'))
            .borderRadius(DesignConstants.RADIUS.LG)
            .margin({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG })
          }, (acc: AuthData) => acc.user_id)
        }
        .width('100%')
        .layoutWeight(1)
        .scrollBar(BarState.Auto)
        .padding({ bottom: SafeAreaUtils.resolveBottomPadding(this.safeAreaBottom, DesignConstants.SPACING.LG, 0) })
        .transition(TransitionEffect.OPACITY.animation({ duration: 300, curve: Curve.FastOutSlowIn }))
      }
    }
    .width('100%')
    .height('100%')
    .backgroundColor($r('app.color.bg_primary'))
    }
    .hideTitleBar(true)
    .onReady((context: NavDestinationContext) => {
      this.pathStack = context.pathStack;
    })
  }
}

@Builder
export function AccountsPageBuilder() {
  AccountsPage()
}
