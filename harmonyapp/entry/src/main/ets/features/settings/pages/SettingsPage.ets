/**
 * 设置页面
 * 使用 HarmonyOS 原生 List 组件 + 系统颜色资源
 */

import { GlobalStateManager, KEY_CURRENT_USER, KEY_IS_LOGGED_IN, KEY_USER_TYPE } from '../../../core/state/GlobalStateManager';
import { PageNames } from '../../../constants/AppConstants';
import { SystemToast } from '../../../components/SystemToast';
import { DesignConstants } from '../../../constants/DesignConstants';
import { AppIcon } from '../../../components/SystemIcon';
import { HttpUtil } from '../../../core/network/HttpUtil';
import { WebSocketService } from '../../../core/network/WebSocketService';
import { MyProfileData } from '../../../core/types/api';
import { AuthAccountManager } from '../../../core/state/AuthAccountManager';
import { MessageCacheService } from '../../../core/database/MessageCacheService';

@Entry
@Component
struct SettingsPage {
  @StorageLink(KEY_CURRENT_USER) currentUser: MyProfileData | null = null;
  @StorageLink(KEY_IS_LOGGED_IN) isLoggedIn: boolean = false;
  @StorageLink(KEY_USER_TYPE) userType: string = '';
  private stateManager = GlobalStateManager.getInstance();
  private webSocketService = WebSocketService.getInstance();

  private async logout() {
    try {
      this.webSocketService.disconnect();
      await MessageCacheService.getInstance().clearAll();
      await AuthAccountManager.lockCurrent();
      await HttpUtil.clearToken();
      this.stateManager.updateUser(null);

      const routerInstance = this.getUIContext().getRouter();
      routerInstance.clear();
      await routerInstance.pushUrl({ url: `features/main/pages/${PageNames.INDEX}` });
      SystemToast.show('已退出登录', this.getUIContext());
    } catch (error) {
      SystemToast.show('退出失败', this.getUIContext());
    }
  }

  build() {
    Column() {
      // 标题栏
      Row() {
        Text() {
          SymbolSpan($r(AppIcon.BACK))
            .fontSize(24)
            .fontColor([$r('sys.color.ohos_id_color_text_primary')])
        }
        .onClick(() => {
          const routerInstance = this.getUIContext().getRouter();
          routerInstance.back();
        })
        .width(48)
        .height(48)
        .textAlign(TextAlign.Center)

        Text('设置')
          .fontSize(DesignConstants.FONT.SIZE.XL)
          .fontWeight(FontWeight.Medium)
          .fontColor($r('sys.color.ohos_id_color_text_primary'))
          .layoutWeight(1)
          .textAlign(TextAlign.Center)

        Blank().width(48)
      }
      .width('100%')
      .height(56)
      .padding({ left: DesignConstants.SPACING.SM, right: DesignConstants.SPACING.SM })
      .backgroundColor($r('sys.color.ohos_id_color_background'))

      // 设置列表
      List({ space: DesignConstants.SPACING.MD }) {
        // 账号设置组
        ListItemGroup({ header: this.ListHeader('账号') }) {
          ListItem() {
            this.SettingItem('账号管理', '切换/删除本地已保存账号', () => {
              const routerInstance = this.getUIContext().getRouter();
              routerInstance.pushUrl({ url: `features/settings/pages/${PageNames.ACCOUNTS}` });
            })
          }
          .backgroundColor($r('sys.color.ohos_id_color_background'))

          ListItem() {
            this.SettingItem('个人资料', '修改头像、昵称', async () => {
              if (this.currentUser) {
                const userId = await AuthAccountManager.getCurrentUserId();
                const routerInstance = this.getUIContext().getRouter();
                routerInstance.pushUrl({
                  url: `features/profile/pages/${PageNames.PROFILE}`,
                  params: { userId: userId }
                });
              } else {
                SystemToast.showWarning('请先登录', this.getUIContext());
                const routerInstance = this.getUIContext().getRouter();
                routerInstance.pushUrl({
                  url: `features/auth/pages/${PageNames.LOGIN}`,
                  params: { reason: 'action_required', action: '查看个人资料', redirectUrl: `features/profile/pages/${PageNames.PROFILE}` }
                });
              }
            })
          }
          .backgroundColor($r('sys.color.ohos_id_color_background'))
        }
        .borderRadius(DesignConstants.RADIUS.LG)
        .divider({ strokeWidth: 0.5, color: $r('sys.color.ohos_id_color_list_separator'), startMargin: 16, endMargin: 16 })
        .margin({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG })

        // 通用设置组
        ListItemGroup({ header: this.ListHeader('通用') }) {
          ListItem() {
            this.SettingItem('消息通知', '', () => {
              const routerInstance = this.getUIContext().getRouter();
              routerInstance.pushUrl({ url: `features/subscriptions/pages/${PageNames.SUBSCRIPTIONS}` });
            })
          }
          .backgroundColor($r('sys.color.ohos_id_color_background'))

          ListItem() {
            this.SettingItem('高级设置', '开发者选项、网络诊断', async () => {
              const routerInstance = this.getUIContext().getRouter();
              await routerInstance.pushUrl({ url: `features/settings/pages/${PageNames.ADVANCED_SETTINGS}` });
            })
          }
          .backgroundColor($r('sys.color.ohos_id_color_background'))
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
          .backgroundColor($r('sys.color.ohos_id_color_background'))
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
              .backgroundColor($r('sys.color.ohos_id_color_background'))
              .fontColor('#FF3B30')
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
              .onClick(async () => {
                const routerInstance = this.getUIContext().getRouter();
                await routerInstance.pushUrl({ url: `features/auth/pages/${PageNames.LOGIN}` });
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
    .backgroundColor($r('sys.color.ohos_id_color_sub_background'))
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
