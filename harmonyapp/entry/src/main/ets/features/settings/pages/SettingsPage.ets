/**
 * 设置页面
 * 使用 HarmonyOS 原生 List 组件 + 系统颜色资源
 */

import { GlobalStateManager, KEY_CURRENT_USER, KEY_IS_LOGGED_IN, KEY_USER_TYPE } from '../../../core/state/GlobalStateManager';
import { ApiClient } from '../../../core/api/ApiClient';
import { SystemToast } from '../../../components/SystemToast';
import { DesignConstants } from '../../../constants/DesignConstants';
import { AppIcon, IconSize, IconButton } from '../../../components/SystemIcon';
import { HttpUtil } from '../../../core/network/HttpUtil';
import { WebSocketService } from '../../../core/network/WebSocketService';
import { MyProfile, ApiVersionMeta } from '../../../core/types/api';
import { AuthAccountManager } from '../../../core/state/AuthAccountManager';
import { MessageCacheService } from '../../../core/database/MessageCacheService';
import { AppRouter, ProfileRouteParam, LoginRouteParam } from '../../../core/router/AppRouter';
import { PageNames } from '../../../constants/AppConstants';
import { SystemUiController } from '../../../core/utils/SystemUiController';
import { SafeAreaUtils } from '../../../core/utils/SafeAreaUtils';
import { RefreshIndicator } from '../../../components/EmptyState';

@Preview
@Component
export struct SettingsPage {
  pathStack: NavPathStack = new NavPathStack();
  @StorageLink(KEY_CURRENT_USER) currentUser: MyProfile | null = null;
  @StorageLink(KEY_IS_LOGGED_IN) isLoggedIn: boolean = false;
  @StorageLink(KEY_USER_TYPE) userType: string = '';
  @StorageLink('safeAreaTop') safeAreaTop: number = 0;
  @State systemMeta: ApiVersionMeta | null = null;
  @State systemMetaLoading: boolean = false;
  @State systemMetaError: string = '';
  @State systemMetaLastChecked: string = '';
  @State metaRefreshing: boolean = false;
  private stateManager = GlobalStateManager.getInstance();
  private webSocketService = WebSocketService.getInstance();

  aboutToAppear(): void {
    SystemUiController.useSurface({
      baseColor: DesignConstants.COLOR.PRIMARY_BG_LIGHT,      preferDarkContent: true
    });
    this.loadSystemMeta();
  }

  private async loadSystemMeta(force: boolean = false): Promise<void> {
    if (this.systemMetaLoading && !force) {
      return;
    }
    this.systemMetaLoading = true;
    if (force) {
      this.metaRefreshing = true;
    }
    this.systemMetaError = '';
    try {
      const response = await ApiClient.getSystemMeta();
      if (response.code === 0 && response.data) {
        this.systemMeta = response.data;
        this.systemMetaLastChecked = new Date().toISOString();
      } else {
        this.systemMetaError = response.message || '获取系统信息失败';
      }
    } catch (error) {
      this.systemMetaError = error instanceof Error ? error.message : '获取系统信息失败';
      SystemToast.showError(this.systemMetaError, this.getUIContext());
    } finally {
      this.systemMetaLoading = false;
      this.metaRefreshing = false;
    }
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
      .backgroundColor($r('app.color.bg_primary'))
      .expandSafeArea([SafeAreaType.SYSTEM], [SafeAreaEdge.TOP])

      // 设置列表
      Refresh({ refreshing: $$this.metaRefreshing, builder: RefreshIndicator }) {
        List({ space: DesignConstants.SPACING.MD }) {
          // 账号设置组
          ListItemGroup({ header: this.ListHeader('账号') }) {
            ListItem() {
              this.SettingItem(AppIcon.PERSON, '账号管理', '切换/删除本地已保存账号', () => {
                AppRouter.push(PageNames.ACCOUNTS);
              })
            }
            .backgroundColor(Color.Transparent)

            ListItem() {
              this.SettingItem(AppIcon.EDIT, '个人资料', '修改头像、昵称', async () => {
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
            .backgroundColor(Color.Transparent)
          }
          .backgroundColor($r('app.color.bg_secondary'))
          .borderRadius(DesignConstants.RADIUS.XXL)
          .border({ width: 1, color: $r('sys.color.ohos_id_color_list_separator') })
          .shadow({ radius: 10, color: $r('app.color.color_shadow'), offsetX: 0, offsetY: 3 })
          .divider({ strokeWidth: 0.5, color: $r('sys.color.ohos_id_color_list_separator'), startMargin: 16, endMargin: 16 })
          .margin({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG })

          // 通用设置组
          ListItemGroup({ header: this.ListHeader('通用') }) {
            ListItem() {
              this.SettingItem(AppIcon.NOTIFICATION, '消息通知', '', () => {
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
            .backgroundColor(Color.Transparent)

            ListItem() {
              this.SettingItem(AppIcon.SETTINGS, '高级设置', '开发者选项、网络诊断', () => {
                AppRouter.push(PageNames.ADVANCED_SETTINGS);
              })
            }
            .backgroundColor(Color.Transparent)
          }
          .backgroundColor($r('app.color.bg_secondary'))
          .borderRadius(DesignConstants.RADIUS.XXL)
          .border({ width: 1, color: $r('sys.color.ohos_id_color_list_separator') })
          .shadow({ radius: 10, color: $r('app.color.color_shadow'), offsetX: 0, offsetY: 3 })
          .divider({ strokeWidth: 0.5, color: $r('sys.color.ohos_id_color_list_separator'), startMargin: 16, endMargin: 16 })
          .margin({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG })

          // 关于组
          ListItemGroup({ header: this.ListHeader('关于') }) {
            ListItem() {
              this.SystemMetaCard()
            }
            .backgroundColor(Color.Transparent)
          }
          .backgroundColor($r('app.color.bg_secondary'))
          .borderRadius(DesignConstants.RADIUS.XXL)
          .border({ width: 1, color: $r('sys.color.ohos_id_color_list_separator') })
          .shadow({ radius: 10, color: $r('app.color.color_shadow'), offsetX: 0, offsetY: 3 })
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
                .borderWidth(1)
                .borderColor($r('sys.color.ohos_id_color_list_separator'))
                .borderRadius(DesignConstants.RADIUS.XXL)
                .shadow({ radius: 8, color: $r('app.color.color_shadow'), offsetX: 0, offsetY: 3 })
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
                .type(ButtonType.Normal)
                .borderRadius(DesignConstants.RADIUS.XXL)
                .shadow({ radius: 8, color: $r('app.color.color_shadow'), offsetX: 0, offsetY: 3 })
                .onClick(() => {
                  AppRouter.push(PageNames.LOGIN);
                })
            }
            .margin({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, top: DesignConstants.SPACING.MD, bottom: DesignConstants.SPACING.XXL })
          }
        }
        .scrollBar(BarState.Auto)
        .sticky(StickyStyle.Header)
      }
      .onRefreshing(() => {
        if (!this.metaRefreshing) {
          this.metaRefreshing = true;
          this.loadSystemMeta(true);
        }
      })
      .width('100%')
      .layoutWeight(1)

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
  private SystemMetaCard() {
    Column({ space: DesignConstants.SPACING.MD }) {
      Row({ space: DesignConstants.SPACING.SM }) {
        Column({ space: 4 }) {
          Text('系统版本状态')
            .fontSize(DesignConstants.FONT.SIZE.LG)
            .fontWeight(FontWeight.Medium)
            .fontColor($r('sys.color.ohos_id_color_text_primary'))

          Text(this.systemMeta?.status === 'sunset' ? '可能影响正常使用' : '与服务端版本保持同步')
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
        }
        .layoutWeight(1)

        Text(this.getMetaStatusLabel())
          .fontSize(12)
          .fontWeight(FontWeight.Medium)
          .fontColor(this.getMetaStatusTextColor())
          .backgroundColor(this.getMetaStatusBackground())
          .padding({
            left: DesignConstants.SPACING.MD,
            right: DesignConstants.SPACING.MD,
            top: DesignConstants.SPACING.XS,
            bottom: DesignConstants.SPACING.XS
          })
          .borderRadius(DesignConstants.RADIUS.SM)

        Stack() {
          IconButton({
            icon: AppIcon.REFRESH,
            iconSize: IconSize.MEDIUM,
            iconColor: $r('sys.color.ohos_id_color_text_primary'),
            bgColor: Color.Transparent,
            onButtonClick: () => { this.loadSystemMeta(true); }
          })
            .enabled(!this.systemMetaLoading)

          if (this.systemMetaError) {
            Circle({ width: 8, height: 8 })
              .fill($r('app.color.color_danger'))
              .position({ x: 26, y: 2 })
          }
        }
      }

      if (this.systemMetaLoading) {
        Row({ space: DesignConstants.SPACING.SM }) {
          LoadingProgress().width(20).height(20).color($r('sys.color.ohos_id_color_emphasize'))
          Text('正在获取版本信息...')
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
        }
      } else {
        this.MetaInfoRow('当前版本', this.systemMeta?.version ?? '--')
        this.MetaInfoRow('推荐版本', this.systemMeta?.recommended_version ?? '已是最新版本')
        this.MetaInfoRow('提示', this.systemMeta?.message ?? '服务运行正常')
        this.MetaInfoRow('检查时间', this.formatMetaTime(this.systemMetaLastChecked))
      }

      if (this.systemMetaError) {
        Text(this.systemMetaError)
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .fontColor($r('sys.color.ohos_id_color_warning'))
      }
    }
    .width('100%')
    .padding(DesignConstants.SPACING.LG)
    .backgroundColor(Color.Transparent)
    .borderRadius(DesignConstants.RADIUS.XL)
  }

  private getMetaStatusLabel(): string {
    if (!this.systemMeta) {
      return '未获取';
    }
    switch (this.systemMeta.status) {
      case 'deprecated':
        return '建议升级';
      case 'sunset':
        return '维护中';
      default:
        return '正常';
    }
  }

  private getMetaStatusTextColor(): ResourceColor {
    if (!this.systemMeta) {
      return $r('sys.color.ohos_id_color_text_secondary');
    }
    switch (this.systemMeta.status) {
      case 'deprecated':
        return $r('sys.color.ohos_id_color_warning');
      case 'sunset':
        return $r('app.color.color_danger');
      default:
        return $r('app.color.color_success');
    }
  }

  private getMetaStatusBackground(): ResourceColor {
    if (!this.systemMeta) {
      return DesignConstants.COLOR.SURFACE_NEUTRAL_SOFT;
    }
    switch (this.systemMeta.status) {
      case 'deprecated':
        return DesignConstants.COLOR.SURFACE_WARNING_SOFT;
      case 'sunset':
        return DesignConstants.COLOR.SURFACE_DANGER_SOFT;
      default:
        return DesignConstants.COLOR.SURFACE_SUCCESS_SOFT;
    }
  }

  private formatMetaTime(value?: string): string {
    if (!value) {
      return '尚未检查';
    }
    try {
      const date = new Date(value);
      if (Number.isNaN(date.getTime())) {
        return value;
      }
      const month = (date.getMonth() + 1).toString().padStart(2, '0');
      const day = date.getDate().toString().padStart(2, '0');
      const hour = date.getHours().toString().padStart(2, '0');
      const minute = date.getMinutes().toString().padStart(2, '0');
      return `${month}-${day} ${hour}:${minute}`;
    } catch (_) {
      return value;
    }
  }

  @Builder
  private MetaInfoRow(label: string, value: string) {
    Row() {
      Text(label)
        .fontSize(DesignConstants.FONT.SIZE.SM)
        .fontColor($r('sys.color.ohos_id_color_text_secondary'))
        .layoutWeight(1)

      Text(value)
        .fontSize(DesignConstants.FONT.SIZE.SM)
        .fontColor($r('sys.color.ohos_id_color_text_primary'))
        .textAlign(TextAlign.End)
    }
  }

  @Builder
  SettingItem(icon: AppIcon, title: string, subtitle: string, onClick: () => void) {
    Row({ space: DesignConstants.SPACING.MD }) {
      Row() {
        Text() {
          SymbolSpan($r(icon))
            .fontSize(18)
            .fontColor([$r('sys.color.ohos_id_color_emphasize')])
        }
      }
      .width(38)
      .height(38)
      .borderRadius(DesignConstants.RADIUS.XL)
      .backgroundColor($r('app.color.bg_tertiary'))
      .justifyContent(FlexAlign.Center)
      .alignItems(VerticalAlign.Center)

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

      Row() {
        Text() {
          SymbolSpan($r(AppIcon.ARROW_RIGHT))
            .fontSize(15)
            .fontColor([$r('sys.color.ohos_id_color_text_secondary')])
        }
      }
      .width(28)
      .height(28)
      .borderRadius(DesignConstants.RADIUS.XL)
      .backgroundColor($r('app.color.bg_tertiary'))
      .justifyContent(FlexAlign.Center)
      .alignItems(VerticalAlign.Center)
    }
    .width('100%')
    .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, top: DesignConstants.SPACING.MD, bottom: DesignConstants.SPACING.MD })
    .borderRadius(DesignConstants.RADIUS.XL)
    .onClick(onClick)
  }
}

@Builder
export function SettingsPageBuilder() {
  SettingsPage()
}
