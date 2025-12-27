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
import { MyProfileData, AppAuthData } from '../../../core/types/api';
import { PageNames } from '../../../constants/AppConstants';
import { SystemToast } from '../../../components/SystemToast';
import { DesignConstants } from '../../../constants/DesignConstants';
import { AppIcon, IconSize, IconButton } from '../../../components/SystemIcon';

@Entry
@Component
struct AccountsPage {
  @StorageLink(KEY_CURRENT_USER) currentUser: MyProfileData | null = null;

  @State accounts: AppAuthData[] = [];
  @State currentUserId: number | null = null;
  @State isLoading: boolean = false;

  private stateManager = GlobalStateManager.getInstance();
  private ws = WebSocketService.getInstance();
  private cache = MessageCacheService.getInstance();

  async aboutToAppear() {
    await this.load();
  }

  private async load() {
    this.isLoading = true;
    try {
      this.accounts = await AuthAccountManager.listAccounts();
      this.currentUserId = await AuthAccountManager.getCurrentUserId();
    } finally {
      this.isLoading = false;
    }
  }

  private toProfile(auth: AppAuthData): MyProfileData {
    return {
      user_id: auth.user_id,
      nickname: auth.nickname,
      avatar_url: '',
      bio: '',
      tags: [],
      email: '',
      phone: '',
      location_name: '',
      is_online: true,
      last_online: '',
      created_at: new Date().toISOString()
    };
  }

  private async switchTo(auth: AppAuthData) {
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

      SystemToast.showSuccess(`已切换到：${next.nickname}`, this.getUIContext());
      await this.load();
    } catch {
      SystemToast.showError('切换失败', this.getUIContext());
    }
  }

  private async remove(auth: AppAuthData) {
    try {
      await AuthAccountManager.removeAccount(auth.user_id);
      if (this.currentUserId === auth.user_id) {
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

        Text('账号管理')
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

      if (this.isLoading) {
        Text('加载中...')
          .fontSize(14)
          .fontColor(DesignConstants.COLOR.TEXT_SECONDARY_LIGHT)
          .margin({ top: 24 })
      } else if (this.accounts.length === 0) {
        Column({ space: 12 }) {
          Text('暂无已保存账号')
            .fontSize(14)
            .fontColor(DesignConstants.COLOR.TEXT_SECONDARY_LIGHT)

          Button('去登录/创建访客')
            .type(ButtonType.Capsule)
            .backgroundColor(DesignConstants.COLOR.BRAND_PRIMARY)
            .fontColor(Color.White)
            .height(44)
            .onClick(() => {
              const routerInstance = this.getUIContext().getRouter();
              routerInstance.pushUrl({ url: `features/auth/pages/${PageNames.LOGIN}` });
            })
        }
        .width('100%')
        .padding(24)
      } else {
        List({ space: 12 }) {
          ForEach(this.accounts, (acc: AppAuthData) => {
            ListItem() {
              Row() {
                Column({ space: 4 }) {
                  Row({ space: 8 }) {
                    Text(acc.nickname)
                      .fontSize(16)
                      .fontColor(DesignConstants.COLOR.TEXT_PRIMARY_LIGHT)
                    if (this.currentUserId === acc.user_id) {
                      Text('当前')
                        .fontSize(12)
                        .fontColor(Color.White)
                        .backgroundColor(DesignConstants.COLOR.BRAND_PRIMARY)
                        .borderRadius(8)
                        .padding({ left: 8, right: 8, top: 2, bottom: 2 })
                    }
                  }
                  Text(`ID: ${acc.user_id}`)
                    .fontSize(12)
                    .fontColor(DesignConstants.COLOR.TEXT_TERTIARY_LIGHT)
                }
                .layoutWeight(1)
                .alignItems(HorizontalAlign.Start)

                Row({ space: 8 }) {
                  Button('切换')
                    .type(ButtonType.Capsule)
                    .fontSize(12)
                    .height(32)
                    .backgroundColor(DesignConstants.COLOR.TERTIARY_BG_LIGHT)
                    .fontColor(DesignConstants.COLOR.TEXT_PRIMARY_LIGHT)
                    .onClick(() => this.switchTo(acc))

                  Button('删除')
                    .type(ButtonType.Capsule)
                    .fontSize(12)
                    .height(32)
                    .backgroundColor(DesignConstants.COLOR.ERROR)
                    .fontColor(Color.White)
                    .onClick(() => this.remove(acc))
                }
              }
              .width('100%')
              .padding({ left: 16, right: 16, top: 12, bottom: 12 })
            }
            .backgroundColor(DesignConstants.COLOR.PRIMARY_BG_LIGHT)
            .borderRadius(DesignConstants.RADIUS.LG)
            .margin({ left: 16, right: 16 })
          }, (acc: AppAuthData) => acc.user_id.toString())
        }
        .width('100%')
        .layoutWeight(1)
        .scrollBar(BarState.Auto)
      }
    }
    .width('100%')
    .height('100%')
    .backgroundColor(DesignConstants.COLOR.SECONDARY_BG_LIGHT)
  }
}

