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
import { SystemPageHeader } from '../../../components/SystemPageHeader';
import { RefreshIndicator } from '../../../components/EmptyState';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { AppThemeVisuals } from '../../../core/theme/AppThemeVisuals';
import { AppThemePalette } from '../../../core/theme/AppThemeTypes';
import { KEY_RESOLVED_THEME_ID, KEY_THEME_MODE, ThemeOptionItem, ThemeMode } from '../../../core/theme/AppThemeTypes';

@Preview
@Component
export struct SettingsPage {
  pathStack: NavPathStack = new NavPathStack();
  @StorageLink(KEY_CURRENT_USER) currentUser: MyProfile | null = null;
  @StorageLink(KEY_IS_LOGGED_IN) isLoggedIn: boolean = false;
  @StorageLink(KEY_USER_TYPE) userType: string = '';
  @StorageLink('safeAreaTop') safeAreaTop: number = 0;
  @StorageLink(KEY_THEME_MODE) themeMode: ThemeMode = 'follow_system';
  @StorageLink(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'default_clean_light';
  @State systemMeta: ApiVersionMeta | null = null;
  @State systemMetaLoading: boolean = false;
  @State systemMetaError: string = '';
  @State systemMetaLastChecked: string = '';
  @State metaRefreshing: boolean = false;
  @State showThemeSheet: boolean = false;
  private stateManager = GlobalStateManager.getInstance();
  private webSocketService = WebSocketService.getInstance();

  aboutToAppear(): void {
    SystemUiController.syncWithCurrentTheme();
    this.loadSystemMeta();
  }

  private getThemePreviewColor(): string {
    return this.getCurrentThemePalette().accentBackground;
  }

  private getThemeDisplayLabel(): string {
    return AppThemeManager.getThemeLabel(this.themeMode);
  }

  private getCurrentThemePalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  private getSurfaceBackground(): string {
    return this.getCurrentThemePalette().surfaceBackground;
  }

  private getSurfaceSubtle(): string {
    return this.getCurrentThemePalette().surfaceSubtle;
  }

  private getSurfaceBorder(): string {
    return this.getCurrentThemePalette().borderColor;
  }

  private getThemeSheetPalette(): AppThemePalette {
    return this.getCurrentThemePalette();
  }

  private getSoftCardShadow(): ShadowOptions {
    return this.getCurrentThemePalette().elevatedShadow;
  }

  private getThemeOptionPalette(option: ThemeOptionItem): AppThemePalette {
    if (this.themeMode === option.mode) {
      return this.getCurrentThemePalette();
    }
    return AppThemeManager.getPaletteForMode(option.mode);
  }

  private isThemeOptionActive(option: ThemeOptionItem): boolean {
    return this.themeMode === option.mode;
  }

  private async handleThemeChange(mode: ThemeMode): Promise<void> {
    if (mode === this.themeMode) {
      this.showThemeSheet = false;
      return;
    }
    await AppThemeManager.setThemeMode(mode);
    SystemUiController.syncWithCurrentTheme();
    this.showThemeSheet = false;
    SystemToast.show(`已切换为${AppThemeManager.getThemeLabel(mode)}`, this.getUIContext());
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
      SystemPageHeader({
        title: '设置',
        showBackButton: true,
        onBack: () => { AppRouter.pop(); }
      })

      // 设置列表
      Refresh({ refreshing: $$this.metaRefreshing, builder: RefreshIndicator }) {
        List() {
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
          .backgroundColor(this.getSurfaceBackground())
          .borderRadius(DesignConstants.SHAPE.CARD)
          .shadow(this.getSoftCardShadow())
          .border({ width: 1, color: this.getSurfaceBorder() })
          .margin({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, bottom: 14 })

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
              this.SettingItem(AppIcon.STAR_FILL, '外观主题', this.getThemeDisplayLabel(), () => {
                this.showThemeSheet = true;
              }, this.getThemePreviewColor())
            }
            .backgroundColor(Color.Transparent)

            ListItem() {
              this.SettingItem(AppIcon.SETTINGS, '高级设置', '开发者选项、网络诊断', () => {
                AppRouter.push(PageNames.ADVANCED_SETTINGS);
              })
            }
            .backgroundColor(Color.Transparent)
          }
          .backgroundColor(this.getSurfaceBackground())
          .borderRadius(DesignConstants.SHAPE.CARD)
          .shadow(this.getSoftCardShadow())
          .border({ width: 1, color: this.getSurfaceBorder() })
          .margin({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, bottom: 14 })

          // 关于组
          ListItemGroup({ header: this.ListHeader('关于') }) {
            ListItem() {
              this.SystemMetaCard()
            }
            .backgroundColor(Color.Transparent)
          }
          .backgroundColor(this.getSurfaceBackground())
          .borderRadius(DesignConstants.SHAPE.CARD)
          .shadow(this.getSoftCardShadow())
          .border({ width: 1, color: this.getSurfaceBorder() })
          .margin({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, bottom: 14 })

          // 退出登录
          if (this.isLoggedIn) {
            ListItem() {
              Button('退出登录')
                .width('100%')
                .height(50)
                .backgroundColor(Color.Transparent)
                .fontColor(this.getCurrentThemePalette().accentBackground)
                .fontSize(DesignConstants.FONT.SIZE.LG)
                .type(ButtonType.Normal)
                .borderRadius(DesignConstants.SHAPE.BUTTON)
                .shadow(this.getCurrentThemePalette().controlShadow)
                .border({ width: 1, color: this.getSurfaceBorder() })
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
                .backgroundColor(this.getCurrentThemePalette().accentBackground)
                .fontColor(this.getCurrentThemePalette().accentText)
                .fontSize(DesignConstants.FONT.SIZE.LG)
                .type(ButtonType.Normal)
                .borderRadius(DesignConstants.SHAPE.BUTTON)
                .shadow(this.getCurrentThemePalette().controlShadow)
                .border({ width: 1, color: this.getSurfaceBorder() })
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
    .backgroundColor(this.getCurrentThemePalette().pageBackground)
    .bindSheet($$this.showThemeSheet, this.ThemeSheet(), {
      height: SheetSize.FIT_CONTENT,
      showClose: true,
      dragBar: true,
      backgroundColor: this.getCurrentThemePalette().sheetBackground
    })
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
      .fontColor(this.getCurrentThemePalette().textSecondary)
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
            .fontColor(this.getCurrentThemePalette().textPrimary)

          Text(this.systemMeta?.status === 'sunset' ? '可能影响正常使用' : '与服务端版本保持同步')
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .fontColor(this.getCurrentThemePalette().textSecondary)
        }
        .layoutWeight(1)

        Text(this.getMetaStatusLabel())
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .fontWeight(FontWeight.Medium)
          .fontColor(this.getMetaStatusTextColor())
          .backgroundColor(this.getMetaStatusBackground())
          .padding({
            left: DesignConstants.SPACING.MD,
            right: DesignConstants.SPACING.MD,
            top: DesignConstants.SPACING.XS,
            bottom: DesignConstants.SPACING.XS
          })
          .borderRadius(DesignConstants.SHAPE.CHIP)

        Stack() {
          IconButton({
            icon: AppIcon.REFRESH,
            iconSize: IconSize.MEDIUM,
            iconColor: this.getCurrentThemePalette().textPrimary,
            bgColor: this.getSurfaceSubtle(),
            onButtonClick: () => { this.loadSystemMeta(true); }
          })
            .enabled(!this.systemMetaLoading)

          if (this.systemMetaError) {
            Circle({ width: 8, height: 8 })
              .fill(this.getCurrentThemePalette().accentBackground)
              .position({ x: 26, y: 2 })
          }
        }
      }

      if (this.systemMetaLoading) {
        Row({ space: DesignConstants.SPACING.SM }) {
          LoadingProgress().width(20).height(20).color(this.getCurrentThemePalette().accentBackground)
          Text('正在获取版本信息...')
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .fontColor(this.getCurrentThemePalette().textSecondary)
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
          .fontColor(this.getCurrentThemePalette().accentBackground)
      }
    }
    .width('100%')
    .padding(DesignConstants.SPACING.LG)
    .backgroundColor(Color.Transparent)
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
      return this.getCurrentThemePalette().textSecondary;
    }
    switch (this.systemMeta.status) {
      case 'deprecated':
      case 'sunset':
        return this.getCurrentThemePalette().accentBackground;
      default:
        return this.getCurrentThemePalette().textPrimary;
    }
  }

  private getMetaStatusBackground(): ResourceColor {
    if (!this.systemMeta) {
      return AppThemeManager.getSoftStatusBackground('neutral');
    }
    switch (this.systemMeta.status) {
      case 'deprecated':
        return AppThemeManager.getSoftStatusBackground('warning');
      case 'sunset':
        return AppThemeManager.getSoftStatusBackground('danger');
      default:
        return AppThemeManager.getSoftStatusBackground('success');
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
        .fontColor(this.getCurrentThemePalette().textSecondary)
        .layoutWeight(1)

      Text(value)
        .fontSize(DesignConstants.FONT.SIZE.SM)
        .fontColor(this.getCurrentThemePalette().textPrimary)
        .textAlign(TextAlign.End)
    }
  }

  @Builder
  private ThemeSheet() {
    Column({ space: DesignConstants.SPACING.LG }) {
      Text('外观主题')
        .fontSize(DesignConstants.FONT.SIZE.XL)
        .fontWeight(FontWeight.Bold)
        .fontColor(this.getThemeSheetPalette().textPrimary)

      Text('切换整体观感，预览卡片、按钮与底栏风格。')
        .fontSize(DesignConstants.FONT.SIZE.SM)
        .fontColor(this.getThemeSheetPalette().textSecondary)
        .textAlign(TextAlign.Center)

      Column({ space: DesignConstants.SPACING.SM }) {
        ForEach(AppThemeManager.getThemeOptions(), (option: ThemeOptionItem) => {
          this.ThemeOptionCard(option)
        }, (option: ThemeOptionItem) => option.mode)
      }
      .width('100%')
    }
    .width('100%')
    .padding({
      left: DesignConstants.SPACING.LG,
      right: DesignConstants.SPACING.LG,
      top: DesignConstants.SPACING.LG,
      bottom: SafeAreaUtils.resolveBottomPadding(0, DesignConstants.SPACING.XXL, DesignConstants.SPACING.LG)
    })
    .backgroundColor(this.getThemeSheetPalette().sheetBackground)
  }

  @Builder
  private ThemeOptionCard(option: ThemeOptionItem) {
    const palette = this.getThemeOptionPalette(option);
    const preview = AppThemeVisuals.getThemePreviewStyle(palette.id);
    const tabBar = AppThemeVisuals.getTabBarVisualStyle(palette.id);
    const isActive = this.isThemeOptionActive(option);

    Column({ space: DesignConstants.SPACING.MD }) {
      Row({ space: DesignConstants.SPACING.SM }) {
        Column({ space: 4 }) {
          Text(option.label)
            .fontSize(DesignConstants.FONT.SIZE.LG)
            .fontWeight(FontWeight.Medium)
            .fontColor(palette.textPrimary)

          Text(option.description)
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .fontColor(palette.textSecondary)
        }
        .layoutWeight(1)
        .alignItems(HorizontalAlign.Start)

        Text(isActive ? '当前使用' : '点击切换')
          .fontSize(DesignConstants.FONT.SIZE.XS)
          .fontColor(isActive ? palette.accentText : palette.accentBackground)
          .padding({ left: 10, right: 10, top: 4, bottom: 4 })
          .backgroundColor(isActive ? palette.accentBackground : palette.surfaceSubtle)
          .borderRadius(DesignConstants.SHAPE.CHIP)
      }
      .width('100%')

      Column({ space: 8 }) {
        Stack({ alignContent: Alignment.Top }) {
          Column({ space: 8 }) {
            Blank()
              .width(preview.heroAvatarSize)
              .height(preview.heroAvatarSize)
              .borderRadius(DesignConstants.RADIUS.FULL)
              .backgroundColor(palette.surfaceSubtle)
              .border({ width: 1, color: palette.borderColor })
              .shadow(preview.haloShadow)

            Blank()
              .width('66%')
              .height(preview.primaryButton.height)
              .borderRadius(preview.primaryButton.borderRadius)
              .backgroundColor(preview.primaryButton.backgroundColor)
              .shadow(preview.primaryButton.shadow)

            Row({ space: 8 }) {
              Blank()
                .layoutWeight(1)
                .height(36)
                .borderRadius(preview.tileRadius)
                .backgroundColor(palette.surfaceSubtle)
              Blank()
                .layoutWeight(1)
                .height(36)
                .borderRadius(preview.tileRadius)
                .backgroundColor(palette.surfaceSubtle)
            }
            .width('100%')

            Blank()
              .width('100%')
              .height(18)
              .borderRadius(tabBar.radius)
              .backgroundColor(tabBar.backgroundColor)
              .shadow(tabBar.shadow)
          }
          .width('100%')
          .alignItems(HorizontalAlign.Center)
          .padding(DesignConstants.SPACING.MD)
        }
        .width('100%')
        .borderRadius(preview.card.borderRadius)
        .backgroundColor(preview.card.backgroundColor)
        .border({ width: preview.card.borderWidth, color: preview.card.borderColor })
        .shadow(preview.card.shadow)
      }
      .width('100%')
    }
    .width('100%')
    .padding(DesignConstants.SPACING.LG)
    .borderRadius(preview.card.borderRadius)
    .backgroundColor(isActive ? palette.surfaceBackground : palette.surfaceSubtle)
    .border({ width: isActive ? 2 : 1, color: isActive ? palette.accentBackground : palette.borderColor })
    .shadow(isActive ? palette.elevatedShadow : palette.surfaceShadow)
    .onClick(() => {
      this.handleThemeChange(option.mode).catch(() => {
        SystemToast.showError('切换主题失败', this.getUIContext());
      });
    })
  }

  @Builder
  SettingItem(icon: AppIcon, title: string, subtitle: string, onClick: () => void, accentColor?: string) {
    Row({ space: DesignConstants.SPACING.MD }) {
      Row() {
        Text() {
          SymbolSpan($r(icon))
            .fontSize(18)
            .fontColor([accentColor ? accentColor : this.getCurrentThemePalette().iconMuted])
        }
      }
      .width(38)
      .height(38)
      .borderRadius(DesignConstants.SHAPE.ICON_TILE)
      .backgroundColor(accentColor ? Color.Transparent : this.getSurfaceSubtle())
      .border(accentColor ? { width: 1, color: accentColor } : { width: 0, color: Color.Transparent })
      .justifyContent(FlexAlign.Center)
      .alignItems(VerticalAlign.Center)

      Column({ space: 4 }) {
        Text(title)
          .fontSize(DesignConstants.FONT.SIZE.LG)
          .fontColor(this.getCurrentThemePalette().textPrimary)

        if (subtitle) {
          Text(subtitle)
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .fontColor(this.getCurrentThemePalette().textSecondary)
        }
      }
      .layoutWeight(1)
      .alignItems(HorizontalAlign.Start)

      Row() {
        Text() {
          SymbolSpan($r(AppIcon.ARROW_RIGHT))
            .fontSize(15)
            .fontColor([this.getCurrentThemePalette().textSecondary])
        }
      }
      .width(28)
      .height(28)
      .borderRadius(DesignConstants.SHAPE.ICON_TILE)
      .backgroundColor(this.getSurfaceSubtle())
      .justifyContent(FlexAlign.Center)
      .alignItems(VerticalAlign.Center)
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
