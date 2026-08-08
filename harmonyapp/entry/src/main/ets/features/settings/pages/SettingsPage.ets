import { GlobalStateManager, KEY_CLIENT_CONFIG, KEY_CURRENT_USER, KEY_IS_LOGGED_IN, KEY_USER_TYPE } from '../../../core/state/GlobalStateManager';
import { ApiClient } from '../../../core/api/ApiClient';
import { SystemToast } from '../../../components/SystemToast';
import { DesignConstants } from '../../../constants/DesignConstants';
import { MyProfile, ApiVersionMeta, ClientConfig, SystemAnnouncementItem } from '../../../core/types/api';
import { AuthAccountManager } from '../../../core/state/AuthAccountManager';
import { MessageCacheService } from '../../../core/database/MessageCacheService';
import { MomentCacheService } from '../../../core/database/MomentCacheService';
import { AppRouter, ProfileRouteParam, LoginRouteParam, AnnouncementsRouteParam } from '../../../core/router/AppRouter';
import { PageNames } from '../../../constants/AppConstants';
import { SystemUiController } from '../../../core/utils/SystemUiController';
import { SafeAreaUtils } from '../../../core/utils/SafeAreaUtils';
import { SystemPageHeader } from '../../../components/SystemPageHeader';
import { RefreshIndicator } from '../../../components/EmptyState';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { AppThemeVisuals } from '../../../core/theme/AppThemeVisuals';
import { AppThemePalette } from '../../../core/theme/AppThemeTypes';
import { KEY_RESOLVED_THEME_ID, KEY_THEME_MODE, ThemeOptionItem, ThemeMode } from '../../../core/theme/AppThemeTypes';
import { WebSocketService } from '../../../core/network/WebSocketService';
import { HttpUtil } from '../../../core/network/HttpUtil';
import { AnnouncementSummaryCard, SettingsEntryItem, SettingsEntryRow, SettingsMetaCard, SettingsSectionCard, SupportConfigCard } from './SettingsSupport';
import { AppIcon } from '../../../components/SystemIcon';
import { ResponsiveContainer } from '../../../components/ResponsiveContainer';
import { hilog } from '@kit.PerformanceAnalysisKit';

const DOMAIN_NUMBER: number = 0xFF27;
const TAG: string = 'SettingsPage';

@Preview
@Component
export struct SettingsPage {
  pathStack: NavPathStack = new NavPathStack();
  @StorageLink(KEY_CURRENT_USER) currentUser: MyProfile | null = null;
  @StorageLink(KEY_IS_LOGGED_IN) isLoggedIn: boolean = false;
  @StorageLink(KEY_USER_TYPE) userType: string = '';
  @StorageLink(KEY_CLIENT_CONFIG) clientConfig: ClientConfig | null = null;
  @StorageLink('safeAreaTop') safeAreaTop: number = 0;
  @StorageLink(KEY_THEME_MODE) themeMode: ThemeMode = 'follow_system';
  @StorageLink(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'default_clean_light';
  @State systemMeta: ApiVersionMeta | null = null;
  @State systemMetaLoading: boolean = false;
  @State systemMetaError: string = '';
  @State systemMetaLastChecked: string = '';
  @State announcements: SystemAnnouncementItem[] = [];
  @State announcementsLoading: boolean = false;
  @State metaRefreshing: boolean = false;
  @State showThemeSheet: boolean = false;
  private stateManager = GlobalStateManager.getInstance();
  private webSocketService = WebSocketService.getInstance();
  private isDestroyed: boolean = false;

  aboutToAppear(): void {
    SystemUiController.syncWithCurrentTheme();
    this.refreshAll(false);
  }

  aboutToDisappear(): void {
    this.isDestroyed = true;
  }

  private getCurrentThemePalette(): AppThemePalette {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  private getThemePreviewColor(): string {
    return this.getCurrentThemePalette().accentBackground;
  }

  private getThemeDisplayLabel(): string {
    return AppThemeManager.getThemeLabel(this.themeMode);
  }

  private getThemeSheetPalette(): AppThemePalette {
    return this.getCurrentThemePalette();
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

  private getSectionMargin(top: number = 0): Padding {
    return {
      left: DesignConstants.SPACING.LG,
      right: DesignConstants.SPACING.LG,
      top: top,
      bottom: DesignConstants.SPACING.LG
    };
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

  private async refreshAll(force: boolean): Promise<void> {
    await Promise.all([
      this.loadSystemMeta(force),
      this.loadAnnouncements(force)
    ]);
  }

  private async loadSystemMeta(force: boolean = false): Promise<void> {
    if (this.systemMetaLoading && !force) {
      return;
    }
    this.systemMetaLoading = true;
    this.systemMetaError = '';
    try {
      const response = await ApiClient.getSystemMeta();
      if (this.isDestroyed) return;
      if (response.code === 0 && response.data) {
        this.systemMeta = response.data;
        this.systemMetaLastChecked = new Date().toISOString();
      } else {
        this.systemMetaError = response.message || '获取系统信息失败';
      }
    } catch (error) {
      this.systemMetaError = error instanceof Error ? error.message : '获取系统信息失败';
    } finally {
      this.systemMetaLoading = false;
    }
  }

  private async loadAnnouncements(force: boolean = false): Promise<void> {
    if (this.announcementsLoading && !force) {
      return;
    }
    this.announcementsLoading = true;
    try {
      const response = await ApiClient.getSystemAnnouncements({});
      if (this.isDestroyed) return;
      if (response.code === 0 && response.data) {
        this.announcements = response.data.items ?? [];
      }
    } catch (error) {
      hilog.warn(DOMAIN_NUMBER, TAG, `加载系统公告失败: ${JSON.stringify(error)}`);
    } finally {
      this.announcementsLoading = false;
      this.metaRefreshing = false;
    }
  }

  private async logout(): Promise<void> {
    try {
      this.webSocketService.disconnect();
      await MessageCacheService.getInstance().clearAll();
      await MomentCacheService.getInstance().clearAll();
      await AuthAccountManager.lockCurrent();
      await HttpUtil.clearToken();
      this.stateManager.updateUser(null);
      AppRouter.clear();
      SystemToast.show('已退出登录', this.getUIContext());
    } catch (_) {
      SystemToast.show('退出失败', this.getUIContext());
    }
  }

  private openProfile(): void {
    if (this.currentUser) {
      AuthAccountManager.getCurrentUserId().then((userId: string | null) => {
        const profileParams = new ProfileRouteParam();
        profileParams.userId = userId ?? '';
        AppRouter.push(PageNames.PROFILE, profileParams);
      });
      return;
    }
    SystemToast.showWarning('请先登录', this.getUIContext());
    const loginParams = new LoginRouteParam();
    loginParams.reason = 'action_required';
    loginParams.action = '查看个人资料';
    loginParams.redirectUrl = PageNames.PROFILE;
    AppRouter.push(PageNames.LOGIN, loginParams);
  }

  private requireLogin(action: string, redirect: string): boolean {
    if (this.isLoggedIn) {
      return true;
    }
    SystemToast.showWarning('请先登录', this.getUIContext());
    const loginParams = new LoginRouteParam();
    loginParams.reason = 'action_required';
    loginParams.action = action;
    loginParams.redirectUrl = redirect;
    AppRouter.push(PageNames.LOGIN, loginParams);
    return false;
  }

  private buildAccountEntries(): SettingsEntryItem[] {
    return [
      {
        icon: AppIcon.PERSON,
        title: '账号管理',
        subtitle: '切换或删除本地已保存账号',
        action: () => AppRouter.push(PageNames.ACCOUNTS)
      },
      {
        icon: AppIcon.EDIT,
        title: '个人资料',
        subtitle: '修改头像、昵称、简介与资料卡片',
        action: () => this.openProfile()
      }
    ];
  }

  private buildPrivacyEntries(): SettingsEntryItem[] {
    return [
      {
        icon: AppIcon.SHIELD,
        title: '隐私设置',
        subtitle: '控制资料、位置、动态与可搜索范围',
        action: () => {
          if (this.requireLogin('管理隐私设置', PageNames.PRIVACY_SETTINGS)) {
            AppRouter.push(PageNames.PRIVACY_SETTINGS);
          }
        }
      },
      {
        icon: AppIcon.GROUP,
        title: '关系与黑名单',
        subtitle: '查看好友、关注、粉丝与黑名单',
        action: () => {
          if (this.requireLogin('管理关系与黑名单', PageNames.RELATIONSHIP_HUB)) {
            AppRouter.push(PageNames.RELATIONSHIP_HUB);
          }
        }
      }
    ];
  }

  private buildSystemEntries(): SettingsEntryItem[] {
    return [
      {
        icon: AppIcon.NOTIFICATION_FILL,
        title: '通知中心',
        subtitle: '查看互动消息、群组提醒和未读通知',
        action: () => {
          if (this.requireLogin('查看通知中心', PageNames.INDEX)) {
            AppStorage.setOrCreate<boolean>('openNotificationCenter', true);
            AppRouter.pop();
          }
        }
      },
      {
        icon: AppIcon.NOTIFICATION,
        title: '通知偏好',
        subtitle: '管理当前设备上的提醒类型与订阅偏好',
        action: () => {
          if (this.requireLogin('管理通知偏好', PageNames.SUBSCRIPTIONS)) {
            AppRouter.push(PageNames.SUBSCRIPTIONS);
          }
        }
      },
      {
        icon: AppIcon.INFO,
        title: '系统公告',
        subtitle: this.announcements.length > 0 ? `最近 ${this.announcements.length} 条平台通知` : '查看平台公告、维护提醒与系统消息',
        action: () => {
          const params = new AnnouncementsRouteParam();
          params.items = this.announcements as Object[];
          AppRouter.push(PageNames.ANNOUNCEMENTS, params);
        },
        accentColor: this.announcements.length > 0 ? this.getCurrentThemePalette().accentBackground : undefined
      },
      {
        icon: AppIcon.LOCK,
        title: '密码与会话安全',
        subtitle: '修改密码、查看登录设备并撤销其他会话',
        action: () => {
          if (this.requireLogin('管理密码与会话安全', PageNames.SAFETY_SETTINGS)) {
            AppRouter.push(PageNames.SAFETY_SETTINGS);
          }
        }
      }
    ];
  }

  build() {
    NavDestination() {
      Column() {
        SystemPageHeader({
          title: '设置',
          showBackButton: true,
          onBack: () => { AppRouter.pop(); }
        })

        Refresh({ refreshing: $$this.metaRefreshing, builder: RefreshIndicator }) {
          Scroll() {
            ResponsiveContainer() {
              // === Section 1: 外观与主题 ===
              Column() {
                SettingsSectionCard({ title: '外观与主题' }) {
                  Column() {
                    SettingsEntryRow({ item: {
                      icon: AppIcon.STAR_FILL,
                      title: '外观主题',
                      subtitle: this.getThemeDisplayLabel(),
                      action: () => { this.showThemeSheet = true; },
                      accentColor: this.getThemePreviewColor()
                    } })
                  }
                }
              }
              .padding(this.getSectionMargin(DesignConstants.SPACING.LG))

              // === Section 2: 隐私、安全与通知 ===
              Column() {
                SettingsSectionCard({ title: '隐私、安全与通知' }) {
                  Column() {
                    ForEach(this.buildPrivacyEntries(), (item: SettingsEntryItem) => {
                      SettingsEntryRow({ item })
                    }, (item: SettingsEntryItem) => item.title)
                    ForEach(this.buildSystemEntries(), (item: SettingsEntryItem) => {
                      SettingsEntryRow({ item })
                    }, (item: SettingsEntryItem) => item.title)
                  }
                }
              }
              .padding(this.getSectionMargin())

              // === Section 3: 账号、记录与高级 ===
              Column() {
                SettingsSectionCard({ title: '账号、记录与高级' }) {
                  Column() {
                    ForEach(this.buildAccountEntries(), (item: SettingsEntryItem) => {
                      SettingsEntryRow({ item })
                    }, (item: SettingsEntryItem) => item.title)
                    SettingsEntryRow({ item: {
                      icon: AppIcon.SETTINGS,
                      title: '高级设置',
                      subtitle: '开发者选项、诊断与缓存工具',
                      action: () => AppRouter.push(PageNames.ADVANCED_SETTINGS)
                    } })
                  }
                }
              }
              .padding(this.getSectionMargin())

              // === Supplementary: 平台状态 ===
              Column() {
                SettingsSectionCard({ title: '平台状态' }) {
                  SettingsMetaCard({
                    meta: this.systemMeta,
                    loading: this.systemMetaLoading,
                    error: this.systemMetaError,
                    lastChecked: this.systemMetaLastChecked,
                    refreshAction: () => {
                      this.loadSystemMeta(true);
                      this.loadAnnouncements(true);
                    }
                  })
                }
              }
              .padding(this.getSectionMargin())

              // === Supplementary: 系统公告速览 ===
              Column() {
                SettingsSectionCard({ title: '系统公告速览' }) {
                  AnnouncementSummaryCard({
                    items: this.announcements,
                    openAll: () => {
                      const params = new AnnouncementsRouteParam();
                      params.items = this.announcements as Object[];
                      AppRouter.push(PageNames.ANNOUNCEMENTS, params);
                    }
                  })
                }
              }
              .padding(this.getSectionMargin())

              // === Supplementary: 公开配置与帮助 ===
              Column() {
                SettingsSectionCard({ title: '公开配置与帮助' }) {
                  SupportConfigCard({ config: this.clientConfig })
                }
              }
              .padding(this.getSectionMargin())

              if (this.isLoggedIn) {
                Button('退出登录')
                  .width('100%')
                  .height(DesignConstants.COMPONENT.LOGOUT_BUTTON_HEIGHT)
                  .backgroundColor(DesignConstants.COLOR.TRANSPARENT)
                  .fontColor(this.getCurrentThemePalette().accentBackground)
                  .fontSize(DesignConstants.FONT.SIZE.LG)
                  .type(ButtonType.Normal)
                  .borderRadius(DesignConstants.SHAPE.BUTTON)
                  .shadow(this.getCurrentThemePalette().controlShadow)
                  .border({ width: 1, color: this.getCurrentThemePalette().borderColor })
                  .margin({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, top: DesignConstants.SPACING.MD, bottom: DesignConstants.SPACING.XXL })
                  .onClick(() => { this.logout(); })
              }
            }
          }
          .scrollBar(BarState.Auto)
          .padding({ top: DesignConstants.SPACING.LG, bottom: SafeAreaUtils.resolveBottomPadding(0, DesignConstants.SPACING.XXL, DesignConstants.SPACING.LG) })
        }
        .onRefreshing(() => {
          if (!this.metaRefreshing) {
            this.metaRefreshing = true;
            this.refreshAll(true);
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
  private ThemeSheet() {
    Column({ space: DesignConstants.SPACING.LG }) {
      Text('外观主题')
        .fontSize(DesignConstants.FONT.SIZE.XL)
        .fontWeight(DesignConstants.FONT.WEIGHT.BOLD)
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
    Column({ space: DesignConstants.SPACING.MD }) {
      Row({ space: DesignConstants.SPACING.SM }) {
        Column({ space: DesignConstants.SPACING.XS }) {
          Text(option.label)
            .fontSize(DesignConstants.FONT.SIZE.LG)
            .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
            .fontColor(this.getThemeOptionPalette(option).textPrimary)
          Text(option.description)
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .fontColor(this.getThemeOptionPalette(option).textSecondary)
        }
        .layoutWeight(1)
        .alignItems(HorizontalAlign.Start)

        Text(this.isThemeOptionActive(option) ? '当前使用' : '点击切换')
          .fontSize(DesignConstants.FONT.SIZE.XS)
          .fontColor(this.isThemeOptionActive(option) ? this.getThemeOptionPalette(option).accentText : this.getThemeOptionPalette(option).accentBackground)
          .padding({ left: DesignConstants.SPACING.MD2, right: DesignConstants.SPACING.MD2, top: DesignConstants.SPACING.XS, bottom: DesignConstants.SPACING.XS })
          .backgroundColor(this.isThemeOptionActive(option) ? this.getThemeOptionPalette(option).accentBackground : this.getThemeOptionPalette(option).surfaceSubtle)
          .borderRadius(DesignConstants.SHAPE.CHIP)
      }
      .width('100%')

      Column({ space: DesignConstants.SPACING.SM }) {
        Stack({ alignContent: Alignment.Top }) {
          Column({ space: DesignConstants.SPACING.SM }) {
            Blank()
              .width(AppThemeVisuals.getThemePreviewStyle(this.getThemeOptionPalette(option).id).heroAvatarSize)
              .height(AppThemeVisuals.getThemePreviewStyle(this.getThemeOptionPalette(option).id).heroAvatarSize)
              .borderRadius(DesignConstants.RADIUS.FULL)
              .backgroundColor(this.getThemeOptionPalette(option).surfaceSubtle)
              .border({ width: 1, color: this.getThemeOptionPalette(option).borderColor })
              .shadow(AppThemeVisuals.getThemePreviewStyle(this.getThemeOptionPalette(option).id).haloShadow)

            Blank()
              .width('70%')
              .height(AppThemeVisuals.getThemePreviewStyle(this.getThemeOptionPalette(option).id).primaryButton.height)
              .borderRadius(AppThemeVisuals.getThemePreviewStyle(this.getThemeOptionPalette(option).id).primaryButton.borderRadius)
              .backgroundColor(AppThemeVisuals.getThemePreviewStyle(this.getThemeOptionPalette(option).id).primaryButton.backgroundColor)
              .shadow(AppThemeVisuals.getThemePreviewStyle(this.getThemeOptionPalette(option).id).primaryButton.shadow)

            Row({ space: DesignConstants.SPACING.MD2 }) {
              Blank().layoutWeight(1).height(DesignConstants.COMPONENT.ACTION_BUTTON_HEIGHT_MD).borderRadius(AppThemeVisuals.getThemePreviewStyle(this.getThemeOptionPalette(option).id).tileRadius).backgroundColor(this.getThemeOptionPalette(option).surfaceSubtle)
              Blank().layoutWeight(1).height(DesignConstants.COMPONENT.ACTION_BUTTON_HEIGHT_MD).borderRadius(AppThemeVisuals.getThemePreviewStyle(this.getThemeOptionPalette(option).id).tileRadius).backgroundColor(this.getThemeOptionPalette(option).surfaceSubtle)
            }
            .width('100%')

            Column() {
              Row() {}
                .width('100%')
                .height(DesignConstants.FONT.SIZE.XL_PLUS)
                .borderRadius(AppThemeVisuals.getTabBarVisualStyle(this.getThemeOptionPalette(option).id).radius)
                .backgroundColor(AppThemeVisuals.getTabBarVisualStyle(this.getThemeOptionPalette(option).id).backgroundColor)
                .shadow(AppThemeVisuals.getTabBarVisualStyle(this.getThemeOptionPalette(option).id).shadow)

              Row() {}
                .width(DesignConstants.COMPONENT.THEME_PREVIEW_BAR_WIDTH)
                .height(DesignConstants.COMPONENT.THEME_PREVIEW_BAR_HEIGHT)
                .margin({ top: -19, left: DesignConstants.SPACING.SM })
                .borderRadius(DesignConstants.RADIUS.FULL)
                .backgroundColor(AppThemeVisuals.getTabBarVisualStyle(this.getThemeOptionPalette(option).id).activeBackgroundColor)
                .shadow(AppThemeVisuals.getTabBarVisualStyle(this.getThemeOptionPalette(option).id).activeGlow)
            }
            .width('100%')
            .alignItems(HorizontalAlign.Start)
          }
          .width('100%')
          .alignItems(HorizontalAlign.Center)
          .padding(DesignConstants.SPACING.MD)
        }
        .width('100%')
        .borderRadius(AppThemeVisuals.getThemePreviewStyle(this.getThemeOptionPalette(option).id).card.borderRadius)
        .backgroundColor(AppThemeVisuals.getThemePreviewStyle(this.getThemeOptionPalette(option).id).card.backgroundColor)
        .border({ width: AppThemeVisuals.getThemePreviewStyle(this.getThemeOptionPalette(option).id).card.borderWidth, color: AppThemeVisuals.getThemePreviewStyle(this.getThemeOptionPalette(option).id).card.borderColor })
        .shadow(AppThemeVisuals.getThemePreviewStyle(this.getThemeOptionPalette(option).id).card.shadow)
      }
      .width('100%')
    }
    .width('100%')
    .padding(DesignConstants.SPACING.LG)
    .borderRadius(AppThemeVisuals.getThemePreviewStyle(this.getThemeOptionPalette(option).id).card.borderRadius)
    .backgroundColor(this.isThemeOptionActive(option) ? this.getThemeOptionPalette(option).surfaceBackground : this.getThemeOptionPalette(option).surfaceSubtle)
    .border({ width: this.isThemeOptionActive(option) ? 1.5 : 1, color: this.isThemeOptionActive(option) ? this.getThemeOptionPalette(option).accentBackground : this.getThemeOptionPalette(option).borderColor })
    .shadow(this.isThemeOptionActive(option) ? this.getThemeOptionPalette(option).accentShadow : this.getThemeOptionPalette(option).surfaceShadow)
    .onClick(() => {
      this.handleThemeChange(option.mode).catch(() => {
        SystemToast.showError('切换主题失败', this.getUIContext());
      });
    })
  }
}

@Builder
export function SettingsPageBuilder() {
  SettingsPage()
}
