/**
 * 设置页面
 * 使用 HarmonyOS 原生 List 组件重构，提供更原生、流畅的体验
 */

import { GlobalStateManager, KEY_CURRENT_USER } from '../../../core/state/GlobalStateManager';
import { PageNames } from '../../../constants/AppConstants';
import { SystemToast } from '../../../components/SystemToast';
import { DesignConstants } from '../../../constants/DesignConstants';
import { AppIcon, IconSize, IconButton } from '../../../components/SystemIcon';
import { PreferencesUtil } from '../../../core/storage/PreferencesUtil';
import { HttpUtil } from '../../../core/network/HttpUtil';
import { WebSocketService } from '../../../core/network/WebSocketService';
import { MyProfileData } from '../../../core/types/api';
import { AuthAccountManager } from '../../../core/state/AuthAccountManager';
import { MessageCacheService } from '../../../core/database/MessageCacheService';

@Entry
@Component
struct SettingsPage {
  @StorageLink(KEY_CURRENT_USER) currentUser: MyProfileData | null = null;
  private stateManager = GlobalStateManager.getInstance();
  private webSocketService = WebSocketService.getInstance();

  /**
   * 退出登录
   */
  private async logout() {
    try {
      this.webSocketService.disconnect();
      // 多账号：默认仅“锁定当前账号”，保留本地已保存的其他账号
      // 同时清空本地消息缓存，避免切换账号后看到上一个账号的消息
      await MessageCacheService.getInstance().clearAll();
      await AuthAccountManager.lockCurrent();
      await HttpUtil.clearToken(); // 兼容：清除 Authorization 使用的 token
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
        IconButton({
          icon: AppIcon.BACK,
          iconSize: IconSize.LARGE,
          iconColor: DesignConstants.COLOR.TEXT_PRIMARY_LIGHT,
          bgColor: Color.Transparent,
          onButtonClick: () => {
            const routerInstance = this.getUIContext().getRouter();
            routerInstance.back();
          }
        })

        Text('设置')
          .fontSize(DesignConstants.FONT.SIZE.XL)
          .fontWeight(FontWeight.Medium)
          .fontColor(DesignConstants.COLOR.TEXT_PRIMARY_LIGHT)
          .layoutWeight(1)
          .textAlign(TextAlign.Center)

        Blank().width(48)
      }
      .width('100%')
      .height(56)
      .padding({ left: 8, right: 8 })
      .backgroundColor(DesignConstants.COLOR.PRIMARY_BG_LIGHT)
      .zIndex(1)

      // 设置列表
      List({ space: 12 }) {
        // 账号设置组
        ListItemGroup({ header: this.ListHeader('账号') }) {
          ListItem() {
            this.SettingItem('账号管理', '切换/删除本地已保存账号', () => {
              const routerInstance = this.getUIContext().getRouter();
              routerInstance.pushUrl({ url: `features/settings/pages/${PageNames.ACCOUNTS}` });
            })
          }
          .backgroundColor(DesignConstants.COLOR.PRIMARY_BG_LIGHT)

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
          .backgroundColor(DesignConstants.COLOR.PRIMARY_BG_LIGHT)
        }
        .borderRadius(DesignConstants.RADIUS.LG)
        .divider({ strokeWidth: 1, color: DesignConstants.COLOR.TERTIARY_BG_LIGHT, startMargin: 16, endMargin: 16 })
        .margin({ left: 16, right: 16 })

        // 通用设置组
        ListItemGroup({ header: this.ListHeader('通用') }) {
          ListItem() {
            this.SettingItem('消息通知', '', () => {
              const routerInstance = this.getUIContext().getRouter();
              routerInstance.pushUrl({ url: `features/subscriptions/pages/${PageNames.SUBSCRIPTIONS}` });
            })
          }
          .backgroundColor(DesignConstants.COLOR.PRIMARY_BG_LIGHT)

          ListItem() {
            this.SettingItem('高级设置', '开发者选项、网络诊断', async () => {
              const routerInstance = this.getUIContext().getRouter();
              await routerInstance.pushUrl({ url: `features/settings/pages/${PageNames.ADVANCED_SETTINGS}` });
            })
          }
          .backgroundColor(DesignConstants.COLOR.PRIMARY_BG_LIGHT)
        }
        .borderRadius(DesignConstants.RADIUS.LG)
        .divider({ strokeWidth: 1, color: DesignConstants.COLOR.TERTIARY_BG_LIGHT, startMargin: 16, endMargin: 16 })
        .margin({ left: 16, right: 16 })

        // 关于组
        ListItemGroup({ header: this.ListHeader('关于') }) {
          ListItem() {
            this.SettingItem('关于HotSpot', 'v1.0.0', () => {
              SystemToast.show('HotSpot v1.0.0', this.getUIContext());
            })
          }
          .backgroundColor(DesignConstants.COLOR.PRIMARY_BG_LIGHT)
        }
        .borderRadius(DesignConstants.RADIUS.LG)
        .divider({ strokeWidth: 1, color: DesignConstants.COLOR.TERTIARY_BG_LIGHT, startMargin: 16, endMargin: 16 })
        .margin({ left: 16, right: 16 })

        // 退出登录
        ListItem() {
          Button('退出登录')
            .width('100%')
            .height(50)
            .backgroundColor(DesignConstants.COLOR.PRIMARY_BG_LIGHT)
            .fontColor(DesignConstants.COLOR.ERROR)
            .fontSize(16)
            .type(ButtonType.Normal)
            .onClick(() => {
              this.logout();
            })
        }
        .borderRadius(DesignConstants.RADIUS.LG)
        .margin({ left: 16, right: 16, top: 12, bottom: 24 })
      }
      .width('100%')
      .layoutWeight(1)
      .scrollBar(BarState.Auto)
      .sticky(StickyStyle.Header)
    }
    .width('100%')
    .height('100%')
    .backgroundColor(DesignConstants.COLOR.SECONDARY_BG_LIGHT)
  }

  @Builder
  ListHeader(title: string) {
    Text(title)
      .fontSize(14)
      .fontColor(DesignConstants.COLOR.TEXT_SECONDARY_LIGHT)
      .margin({ left: 12, bottom: 8, top: 16 })
      .width('100%')
  }

  @Builder
  SettingItem(title: string, subtitle: string, onClick: () => void) {
    Row() {
      Column({ space: 4 }) {
        Text(title)
          .fontSize(16)
          .fontColor(DesignConstants.COLOR.TEXT_PRIMARY_LIGHT)
        
        if (subtitle) {
          Text(subtitle)
            .fontSize(12)
            .fontColor(DesignConstants.COLOR.TEXT_TERTIARY_LIGHT)
        }
      }
      .layoutWeight(1)
      .alignItems(HorizontalAlign.Start)
      
      Text('>')
        .fontSize(16)
        .fontColor(DesignConstants.COLOR.TEXT_TERTIARY_LIGHT)
    }
    .width('100%')
    .padding({ left: 16, right: 16, top: 12, bottom: 12 })
    .onClick(onClick)
  }
}
