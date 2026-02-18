/**
 * 设置页面
 * 使用 HarmonyOS 原生 List 组件 + 系统颜色资源
 */

import { GlobalStateManager, KEY_CURRENT_USER, KEY_IS_LOGGED_IN, KEY_USER_TYPE } from '../../../core/state/GlobalStateManager';
import { SystemToast } from '../../../components/SystemToast';
import { DesignConstants } from '../../../constants/DesignConstants';
import { AppIcon, IconSize, IconButton } from '../../../components/SystemIcon';
import { HttpUtil } from '../../../core/network/HttpUtil';
import { WebSocketService } from '../../../core/network/WebSocketService';
import { MyProfile } from '../../../core/types/api';
import { AuthAccountManager } from '../../../core/state/AuthAccountManager';
import { MessageCacheService } from '../../../core/database/MessageCacheService';
import { AppRouter, ProfileRouteParam, LoginRouteParam } from '../../../core/router/AppRouter';
import { PageNames } from '../../../constants/AppConstants';
import { SystemUiController } from '../../../core/utils/SystemUiController';
import { SafeAreaUtils } from '../../../core/utils/SafeAreaUtils';

@Component
export struct SettingsPage {
  pathStack: NavPathStack = new NavPathStack();
  @StorageLink(KEY_CURRENT_USER) currentUser: MyProfile | null = null;
  @StorageLink(KEY_IS_LOGGED_IN) isLoggedIn: boolean = false;
  @StorageLink(KEY_USER_TYPE) userType: string = '';
  @StorageLink('safeAreaTop') safeAreaTop: number = 0;
  private stateManager = GlobalStateManager.getInstance();
  private webSocketService = WebSocketService.getInstance();

  aboutToAppear(): void {
    SystemUiController.useSurface({
      baseColor: DesignConstants.COLOR.PRIMARY_BG_LIGHT,      preferDarkContent: true
    });
  }

  private async logout() {
    try {
      this.webSocketService.disconnect();
      await MessageCacheService.getInstance().clearAll();
      await AuthAccountManager.lockCurrent();
      await HttpUtil.clearToken();
      this.stateManager.updateUser(null);

      AppRouter.clear();
      SystemToast.show('已退出登录', this.getUIContext());
    } catch (error) {
      SystemToast.show('退出失败', this.getUIContext());
    }
  }

  build() {
    NavDestination() {
    Column() {
      // 标题栏 - 使用 SafeArea 避免与状态栏重叠
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

        Text('设置')
          .fontSize(DesignConstants.FONT.SIZE.XL)
          .fontWeight(FontWeight.Medium)
          .fontColor($r('sys.color.ohos_id_color_text_primary'))
          .layoutWeight(1)
          .textAlign(TextAlign.Center)

        Blank().width(48)
      }
      .width('100%')
      .alignItems(VerticalAlign.Center)
      .padding({
        left: DesignConstants.SPACING.LG,
        right: DesignConstants.SPACING.LG,
        top: SafeAreaUtils.resolveTopPadding(this.safeAreaTop),
        bottom: DesignConstants.SPACING.SM
      })
      .backgroundColor($r('app.color.bg_secondary'))
      .expandSafeArea([SafeAreaType.SYSTEM], [SafeAreaEdge.TOP])

      // 设置列表
      List({ space: DesignConstants.SPACING.MD }) {
        // 账号设置组
        ListItemGroup({ header: this.ListHeader('账号') }) {
          ListItem() {
            this.SettingItem('账号管理', '切换/删除本地已保存账号', () => {
              AppRouter.push(PageNames.ACCOUNTS);
            })
          }
          .backgroundColor($r('app.color.bg_secondary'))

          ListItem() {
            this.SettingItem('个人资料', '修改头像、昵称', async () => {
              if (this.currentUser) {
                const userId = await AuthAccountManager.getCurrentUserId();
                const profileParams = new ProfileRouteParam();
                profileParams.userId = userId ?? '';
                AppRouter.push(PageNames.PROFILE, profileParams);
              } else {
                SystemToast.showWarning('请先登录', this.getUIContext());
                const loginParams = new LoginRouteParam();
                loginParams.reason = 'action_required';
                loginParams.action = '查看个人资料';
                loginParams.redirectUrl = PageNames.PROFILE;
                AppRouter.push(PageNames.LOGIN, loginParams);
              }
            })
          }
          .backgroundColor($r('app.color.bg_secondary'))
        }
        .borderRadius(DesignConstants.RADIUS.LG)
        .divider({ strokeWidth: 0.5, color: $r('sys.color.ohos_id_color_list_separator'), startMargin: 16, endMargin: 16 })
        .margin({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG })

        // 通用设置组
        ListItemGroup({ header: this.ListHeader('通用') }) {
          ListItem() {
            this.SettingItem('消息通知', '', () => {
              if (!this.isLoggedIn) {
                SystemToast.showWarning('请先登录', this.getUIContext());
                const loginParams = new LoginRouteParam();
                loginParams.reason = 'action_required';
                loginParams.action = '管理通知订阅';
                loginParams.redirectUrl = PageNames.SUBSCRIPTIONS;
                AppRouter.push(PageNames.LOGIN, loginParams);
                return;
              }
              AppRouter.push(PageNames.SUBSCRIPTIONS);
            })
          }
          .backgroundColor($r('app.color.bg_secondary'))

          ListItem() {
            this.SettingItem('高级设置', '开发者选项、网络诊断', () => {
              AppRouter.push(PageNames.ADVANCED_SETTINGS);
            })
          }
          .backgroundColor($r('app.color.bg_secondary'))
        }
        .borderRadius(DesignConstants.RADIUS.LG)
        .divider({ strokeWidth: 0.5, color: $r('sys.color.ohos_id_color_list_separator'), startMargin: 16, endMargin: 16 })
        .margin({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG })

        // 关于组
        ListItemGroup({ header: this.ListHeader('关于') }) {
          ListItem() {
            this.SettingItem('关于 HotSpot', 'v1.0.0', () => {
              SystemToast.show('HotSpot v1.0.0', this.getUIContext());
            })
          }
          .backgroundColor($r('app.color.bg_secondary'))
        }
        .borderRadius(DesignConstants.RADIUS.LG)
        .divider({ strokeWidth: 0.5, color: $r('sys.color.ohos_id_color_list_separator'), startMargin: 16, endMargin: 16 })
        .margin({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG })

        // 退出登录
        if (this.isLoggedIn) {
          ListItem() {
            Button('退出登录')
              .width('100%')
              .height(50)
              .backgroundColor($r('app.color.bg_secondary'))
              .fontColor($r('app.color.color_danger'))
              .fontSize(DesignConstants.FONT.SIZE.LG)
              .type(ButtonType.Normal)
              .borderRadius(DesignConstants.RADIUS.LG)
              .onClick(() => { this.logout(); })
          }
          .margin({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, top: DesignConstants.SPACING.MD, bottom: DesignConstants.SPACING.XXL })
        }

        // 未登录时显示登录入口
        if (!this.isLoggedIn) {
          ListItem() {
            Button('登录 / 注册')
              .width('100%')
              .height(50)
              .backgroundColor($r('sys.color.ohos_id_color_emphasize'))
              .fontColor(Color.White)
              .fontSize(DesignConstants.FONT.SIZE.LG)
              .type(ButtonType.Capsule)
              .onClick(() => {
                AppRouter.push(PageNames.LOGIN);
              })
          }
          .margin({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, top: DesignConstants.SPACING.MD, bottom: DesignConstants.SPACING.XXL })
        }
      }
      .width('100%')
      .layoutWeight(1)
      .scrollBar(BarState.Auto)
      .sticky(StickyStyle.Header)
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

  @Builder
  ListHeader(title: string) {
    Text(title)
      .fontSize(DesignConstants.FONT.SIZE.MD)
      .fontColor($r('sys.color.ohos_id_color_text_secondary'))
      .margin({ left: DesignConstants.SPACING.MD, bottom: DesignConstants.SPACING.SM, top: DesignConstants.SPACING.LG })
      .width('100%')
  }

  @Builder
  SettingItem(title: string, subtitle: string, onClick: () => void) {
    Row() {
      Column({ space: 4 }) {
        Text(title)
          .fontSize(DesignConstants.FONT.SIZE.LG)
          .fontColor($r('sys.color.ohos_id_color_text_primary'))

        if (subtitle) {
          Text(subtitle)
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
        }
      }
      .layoutWeight(1)
      .alignItems(HorizontalAlign.Start)

      Text() {
        SymbolSpan($r(AppIcon.ARROW_RIGHT))
          .fontSize(16)
          .fontColor([$r('sys.color.ohos_id_color_text_secondary')])
      }
    }
    .width('100%')
    .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, top: DesignConstants.SPACING.MD, bottom: DesignConstants.SPACING.MD })
    .onClick(onClick)
  }
}

@Builder
export function SettingsPageBuilder() {
  SettingsPage()
}
