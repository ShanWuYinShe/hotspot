import { GlobalStateManager, KEY_CLIENT_CONFIG, KEY_CURRENT_USER, KEY_IS_LOGGED_IN, KEY_USER_TYPE } from '../../../core/state/GlobalStateManager';
import { curves } from '@kit.ArkUI';
import { ApiClient } from '../../../core/api/ApiClient';
import { SystemToast } from '../../../components/SystemToast';
import { DesignConstants } from '../../../constants/DesignConstants';
import { AccountInfo, ApiVersionMeta, ClientConfig, SystemAnnouncementItem, UserSettings } from '../../../core/types/api';
import { AuthAccountManager } from '../../../core/state/AuthAccountManager';
import { MessageCacheService } from '../../../core/database/MessageCacheService';
import { MomentCacheService } from '../../../core/database/MomentCacheService';
import { AppRouter, LoginRouteParam, AnnouncementsRouteParam } from '../../../core/router/AppRouter';
import { PageNames, StorageKeys } from '../../../constants/AppConstants';
import { SystemUiController } from '../../../core/utils/SystemUiController';
import { SafeAreaUtils } from '../../../core/utils/SafeAreaUtils';
import { SystemPageHeader } from '../../../components/SystemPageHeader';
import { SystemAvatar } from '../../../components/SystemAvatar';
import { SystemButton } from '../../../components/SystemButton';
import { SystemDialog } from '../../../components/SystemDialog';
import { RefreshIndicator } from '../../../components/EmptyState';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { AppThemeVisuals } from '../../../core/theme/AppThemeVisuals';
import { ThemeConstants } from '../../../core/theme/ThemeConstants';
import { AppThemePalette } from '../../../core/theme/AppThemeTypes';
import { KEY_RESOLVED_THEME_ID, KEY_THEME_MODE, KEY_DARK_MODE_PREFERENCE, ThemeOptionItem, ThemeMode, DarkModePreference } from '../../../core/theme/AppThemeTypes';
import { WebSocketService } from '../../../core/network/WebSocketService';
import { HttpUtil } from '../../../core/network/HttpUtil';
import { getErrorMessage } from '../../../core/network/ApiErrorUtil';
import {
  AnnouncementSummaryCard,
  SettingsEntryItem,
  SettingsEntryRow,
  SettingsGroupLabel,
  SettingsMetaCard,
  SettingsRowDivider,
  SettingsSectionCard,
  SettingsThemeSegmentCard,
  SettingsToggleRow,
  SupportConfigCard
} from './SettingsSupport';
import { AppIcon } from '../../../components/SystemIcon';
import { TabsPillGroup } from '../../../components/TabsPillGroup';
import { PageShellContainer } from '../../../components/PageShellContainer';
import type { BreakpointType } from '../../../core/responsive/Breakpoint';
import { common } from '@kit.AbilityKit';
import { ConfigurationConstant } from '@kit.AbilityKit';
import { PreferencesUtil } from '../../../core/storage/PreferencesUtil';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { LucideIcon } from '../../../components/LucideIcon';

const DOMAIN_NUMBER: number = 0xFF27;
const TAG: string = 'SettingsPage';

@Preview
@Component
export struct SettingsPage {
  @StorageProp('currentBreakpoint') currentBreakpoint: BreakpointType = 'sm';
  pathStack: NavPathStack = new NavPathStack();
  @StorageLink(KEY_CURRENT_USER) currentUser: AccountInfo | null = null;
  @StorageLink(KEY_IS_LOGGED_IN) isLoggedIn: boolean = false;
  @StorageLink(KEY_USER_TYPE) userType: string = '';
  @StorageLink(KEY_CLIENT_CONFIG) clientConfig: ClientConfig | null = null;
  @StorageLink('safeAreaTop') safeAreaTop: number = 0;
  @StorageLink(KEY_THEME_MODE) themeMode: ThemeMode = 'follow_system';
  @StorageLink(KEY_DARK_MODE_PREFERENCE) darkModePreference: DarkModePreference = 'auto';
  @StorageLink(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'vibrant_light';
  @State systemMeta: ApiVersionMeta | null = null;
  @State systemMetaLoading: boolean = false;
  @State systemMetaError: string = '';
  @State systemMetaLastChecked: string = '';
  @State announcements: SystemAnnouncementItem[] = [];
  @State announcementsLoading: boolean = false;
  @State metaRefreshing: boolean = false;
  @State showThemeSheet: boolean = false;
  /** 主题卡按压态（sheet 主交互元素；原零按压反馈） */
  @State private themeCardPressed: string = '';
  @State showLogoutConfirm: boolean = false;
  // 用户偏好开关（对齐前端 settings：私密账号 / 位置共享 / 推送通知 / 深色）
  @State privacyPrivate: boolean = false;
  @State locationVisible: boolean = true;
  @State pushNotifyEnabled: boolean = true;
  @State privacyPending: boolean = false;
  @State locationPending: boolean = false;
  private static readonly KEY_PUSH_NOTIFY: string = 'settings_push_notify_enabled';
  private stateManager = GlobalStateManager.getInstance();
  private webSocketService = WebSocketService.getInstance();
  private isDestroyed: boolean = false;

  aboutToAppear(): void {
    SystemUiController.syncWithCurrentTheme();
    this.refreshAll(false);
    this.loadLocalPreferences();
    this.loadUserSettings();
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

  // ==================== 偏好开关（对齐前端 settings 乐观更新契约） ====================

  private async loadLocalPreferences(): Promise<void> {
    this.pushNotifyEnabled = await PreferencesUtil.get(SettingsPage.KEY_PUSH_NOTIFY, true);
  }

  private async loadUserSettings(): Promise<void> {
    if (!this.isLoggedIn) {
      return;
    }
    try {
      const response = await ApiClient.getSettings();
      if (this.isDestroyed || response.code !== 0 || !response.data) {
        return;
      }
      this.privacyPrivate = response.data.privacy_level === 'private';
      this.locationVisible = response.data.location_visible;
    } catch (error) {
      hilog.warn(DOMAIN_NUMBER, TAG, `加载用户设置失败: ${JSON.stringify(error)}`);
    }
  }

  private async togglePrivacyPrivate(next: boolean): Promise<void> {
    if (!this.requireLogin('管理隐私设置', PageNames.PRIVACY_SETTINGS)) {
      return;
    }
    const snapshot = this.privacyPrivate;
    this.privacyPrivate = next;
    this.privacyPending = true;
    try {
      const response = await ApiClient.updateSettings({ settings: { privacy_level: next ? 'private' : 'public' } });
      if (response.code !== 0) {
        throw new Error(response.message || 'update failed');
      }
      SystemToast.show(next ? '已切换为私密账号' : '已切换为公开账号', this.getUIContext());
    } catch (error) {
      this.privacyPrivate = snapshot;
      SystemToast.showWarning('设置失败，请稍后重试', this.getUIContext());
    } finally {
      this.privacyPending = false;
    }
  }

  private async toggleLocationVisible(next: boolean): Promise<void> {
    if (!this.requireLogin('管理隐私设置', PageNames.PRIVACY_SETTINGS)) {
      return;
    }
    const snapshot = this.locationVisible;
    this.locationVisible = next;
    this.locationPending = true;
    try {
      const response = await ApiClient.updateSettings({ settings: { location_visible: next } });
      if (response.code !== 0) {
        throw new Error(response.message || 'update failed');
      }
      SystemToast.show(next ? '已开启位置共享' : '已关闭位置共享', this.getUIContext());
    } catch (error) {
      this.locationVisible = snapshot;
      SystemToast.showWarning('设置失败，请稍后重试', this.getUIContext());
    } finally {
      this.locationPending = false;
    }
  }

  private async togglePushNotify(next: boolean): Promise<void> {
    this.pushNotifyEnabled = next;
    await PreferencesUtil.set(SettingsPage.KEY_PUSH_NOTIFY, next);
  }

  /** 明暗三档单通道：偏好经 AppThemeManager 持久化 + 重解主题，再按档位 setColorMode */
  private getDarkModeIndex(): number {
    if (this.darkModePreference === 'light') {
      return 1;
    }
    if (this.darkModePreference === 'dark') {
      return 2;
    }
    return 0;
  }

  private darkModeFromIndex(index: number): DarkModePreference {
    if (index === 1) {
      return 'light';
    }
    if (index === 2) {
      return 'dark';
    }
    return 'auto';
  }

  private async handleDarkModeChange(pref: DarkModePreference): Promise<void> {
    if (pref === this.darkModePreference) {
      return;
    }
    await AppThemeManager.setDarkModePreference(pref);
    SystemUiController.syncWithCurrentTheme();
    try {
      const context = this.getUIContext().getHostContext() as common.UIAbilityContext;
      if (pref === 'dark') {
        context.getApplicationContext().setColorMode(ConfigurationConstant.ColorMode.COLOR_MODE_DARK);
      } else if (pref === 'light') {
        context.getApplicationContext().setColorMode(ConfigurationConstant.ColorMode.COLOR_MODE_LIGHT);
      } else {
        context.getApplicationContext().setColorMode(ConfigurationConstant.ColorMode.COLOR_MODE_NOT_SET);
      }
    } catch (error) {
      hilog.warn(DOMAIN_NUMBER, TAG, `切换颜色模式失败: ${JSON.stringify(error)}`);
    }
    let label: string = '跟随系统';
    if (pref === 'light') {
      label = '浅色';
    } else if (pref === 'dark') {
      label = '深色';
    }
    SystemToast.show(`明暗模式：${label}`, this.getUIContext());
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
        this.systemMetaError = getErrorMessage(response, '获取系统信息失败');
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
      // 「我的资料」不 push 全屏 ProfilePage：自己看自己走非外部模式（无返回钮），
      // NavDestination 全屏覆盖 tab 层后底栏假死、用户被卡死；前端 settings 用户卡
      // 也不跳转。改为回 tab 根视图并切「我的」，由用户在个人中心点「编辑资料」。
      AppRouter.clear();
      AppStorage.setOrCreate<number>(StorageKeys.SWITCH_TAB_REQUEST, 4);
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
      }
    ];
  }

  private buildSecurityEntry(): SettingsEntryItem {
    // 密码与会话安全归属隐私与安全组（通知组只留通知相关行）
    return {
      icon: AppIcon.LOCK,
      title: '密码与会话安全',
      subtitle: '修改密码、查看登录设备并撤销其他会话',
      action: () => {
        if (this.requireLogin('管理密码与会话安全', PageNames.SAFETY_SETTINGS)) {
          AppRouter.push(PageNames.SAFETY_SETTINGS);
        }
      }
    };
  }

  // ==================== 用户卡（对齐前端 settings 顶部 hero 用户卡） ====================
  // currentUser 判空走可选链：登出等场景下 currentUser 先于 isLoggedIn 置空，
  // 脏更新窗口内 if 守卫尚未重算，直接属性访问会崩（faultlogger jscrash 实录）
  @Builder
  private buildUserCard() {
    if (this.isLoggedIn && this.currentUser) {
      Row({ space: DesignConstants.SPACING.MD }) {
        SystemAvatar({
          avatarUrl: this.currentUser?.avatar_url ?? '',
          name: this.currentUser?.nickname ?? '',
          avatarSize: DesignConstants.COMPONENT.AVATAR_SIZE.XS
        })

        Column({ space: DesignConstants.SPACING.THREE }) {
          Text(this.currentUser?.nickname ?? '未命名用户')
            .fontSize(DesignConstants.FONT.SIZE.MD)
            .fontWeight(DesignConstants.FONT.WEIGHT.SEMIBOLD)
            .fontColor(this.getCurrentThemePalette().textPrimary)
            .maxLines(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
          Text(this.currentUser?.email || (this.userType === 'temp' ? '游客身份 · 未绑定邮箱' : '未绑定邮箱'))
            .fontSize(DesignConstants.FONT.SIZE.XS)
            .fontColor(this.getCurrentThemePalette().textSecondary)
            .maxLines(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
        }
        .layoutWeight(1)
        .alignItems(HorizontalAlign.Start)

        LucideIcon({
            icon: AppIcon.ARROW_RIGHT,
            iconSize: DesignConstants.FONT.SIZE.MD,
            color: this.getCurrentThemePalette().textSecondary
          })
      }
      .width('100%')
      .padding(DesignConstants.SPACING.LG)
      .borderRadius(DesignConstants.SHAPE.CARD)
      .backgroundColor(this.getCurrentThemePalette().surfaceBackground)
      .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getCurrentThemePalette().borderColor })
      .margin({ top: DesignConstants.SPACING.LG, bottom: DesignConstants.SPACING.SM })
      .onClick(() => this.openProfile())
    } else {
      Row({ space: DesignConstants.SPACING.MD }) {
        Text('登录后同步你的设置与资料')
          .fontSize(DesignConstants.FONT.SIZE.MD)
          .fontColor(this.getCurrentThemePalette().textSecondary)
          .layoutWeight(1)

        SystemButton({
          label: '登录',
          variant: 'primary',
          buttonSize: 'small',
          onPress: () => {
            const loginParams = new LoginRouteParam();
            loginParams.reason = 'action_required';
            loginParams.action = '登录';
            AppRouter.push(PageNames.LOGIN, loginParams);
          }
        })
      }
      .width('100%')
      .padding(DesignConstants.SPACING.LG)
      .borderRadius(DesignConstants.SHAPE.CARD)
      .backgroundColor(this.getCurrentThemePalette().surfaceBackground)
      .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getCurrentThemePalette().borderColor })
      .margin({ top: DesignConstants.SPACING.LG, bottom: DesignConstants.SPACING.SM })
    }
  }

  // ==================== 主题分段卡（对齐前端主题三选一：星际 / 清野 / 水墨） ====================
  // 独立组件 SettingsThemeSegmentCard（@Builder 经 @BuilderParam 跨组件渲染时
  // onClick 丢失，见 SettingsSupport 内组件注释）；三卡各 layoutWeight(1) 横向均分
  private getActiveSegmentModes(): ThemeMode[] {
    // 跟随系统时默认解析为 vibrant，星旗舰同步高亮
    return this.themeMode === 'follow_system' ? ['vibrant'] : [this.themeMode];
  }

  build() {
    NavDestination() {
      Stack() {
      Column() {
        SystemPageHeader({
          title: '设置',
          titleSize: DesignConstants.FONT.SIZE.XXL,
          titleWeight: DesignConstants.FONT.WEIGHT.MEDIUM,
          showBackButton: true,
          onBack: () => { AppRouter.pop(); }
        })

        Refresh({ refreshing: $$this.metaRefreshing, builder: RefreshIndicator }) {
          Scroll() {
            PageShellContainer({ variant: 'form', currentBreakpoint: this.currentBreakpoint, noHorizontalPadding: true, fillHeight: false }) {
              Column() {
                // === 用户卡 ===
                this.buildUserCard()

                // === 外观 ===
                SettingsGroupLabel({ label: '外观' })
                Column() {
                  SettingsSectionCard() {
                    Column() {
                      Row({ space: DesignConstants.SPACING.SM }) {
                        SettingsThemeSegmentCard({
                          mode: 'vibrant',
                          label: '星际',
                          swatch: ThemeConstants.VIBRANT_LIGHT_COLORS.accentBackground,
                          selected: this.getActiveSegmentModes().indexOf('vibrant') >= 0,
                          onSelect: (mode: string) => { this.handleThemeChange(mode as ThemeMode); }
                        })
                        SettingsThemeSegmentCard({
                          mode: 'default_clean',
                          label: '清野',
                          swatch: ThemeConstants.DEFAULT_CLEAN_LIGHT_COLORS.accentBackground,
                          selected: this.getActiveSegmentModes().indexOf('default_clean') >= 0,
                          onSelect: (mode: string) => { this.handleThemeChange(mode as ThemeMode); }
                        })
                        SettingsThemeSegmentCard({
                          mode: 'inkwash',
                          label: '水墨',
                          swatch: ThemeConstants.INKWASH_LIGHT_COLORS.accentBackground,
                          selected: this.getActiveSegmentModes().indexOf('inkwash') >= 0,
                          onSelect: (mode: string) => { this.handleThemeChange(mode as ThemeMode); }
                        })
                      }
                      .width('100%')
                      .padding({
                        left: DesignConstants.SPACING.LG,
                        right: DesignConstants.SPACING.LG,
                        top: DesignConstants.SPACING.MD,
                        bottom: DesignConstants.SPACING.SM
                      })
                      SettingsRowDivider()
                      // 明暗三档（跟随系统/浅色/深色 → setColorMode NOT_SET/LIGHT/DARK，走 TabsPillGroup 原语）
                      Column({ space: DesignConstants.SPACING.SM }) {
                        Row({ space: DesignConstants.SPACING.MD }) {
                          Row() {
                            LucideIcon({
                              icon: AppIcon.MOON,
                              iconSize: DesignConstants.FONT.SIZE.LG,
                              color: this.getCurrentThemePalette().accentBackground
                            })
                          }
                          .width(DesignConstants.LAYOUT.ICON_BUTTON_SIZE)
                          .height(DesignConstants.LAYOUT.ICON_BUTTON_SIZE)
                          .borderRadius(DesignConstants.RADIUS.FULL)
                          .backgroundColor(this.getCurrentThemePalette().accentMuted)
                          .justifyContent(FlexAlign.Center)
                          .alignItems(VerticalAlign.Center)

                          Column({ space: DesignConstants.SPACING.XS }) {
                            Text('明暗模式')
                              .fontSize(DesignConstants.FONT.SIZE.MD)
                              .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
                              .fontColor(this.getCurrentThemePalette().textPrimary)
                            Text('浅色/深色手动锁定，或跟随系统')
                              .fontSize(DesignConstants.FONT.SIZE.XS)
                              .fontColor(this.getCurrentThemePalette().textSecondary)
                              .maxLines(2)
                              .textOverflow({ overflow: TextOverflow.Ellipsis })
                          }
                          .layoutWeight(1)
                          .alignItems(HorizontalAlign.Start)
                        }
                        .width('100%')

                        TabsPillGroup({
                          tabs: [{ label: '跟随系统' }, { label: '浅色' }, { label: '深色' }],
                          activeIndex: this.getDarkModeIndex(),
                          actions: {
                            onChange: (index: number) => {
                              this.handleDarkModeChange(this.darkModeFromIndex(index)).catch(() => {
                                SystemToast.showError('切换明暗模式失败', this.getUIContext());
                              });
                            }
                          }
                        })
                      }
                      .width('100%')
                      .padding({
                        left: DesignConstants.SPACING.LG,
                        right: DesignConstants.SPACING.LG,
                        top: DesignConstants.SPACING.MD,
                        bottom: DesignConstants.SPACING.MD
                      })
                      SettingsEntryRow({ item: {
                        icon: AppIcon.STAR_FILL,
                        title: '更多外观模式',
                        subtitle: '跟随系统 / 星际 / 清野 / 水墨（大图预览）',
                        action: () => { this.showThemeSheet = true; }
                      } })
                    }
                  }
                }
                .padding(this.getSectionMargin(0))

                // === 通知 ===
                SettingsGroupLabel({ label: '通知' })
                Column() {
                  SettingsSectionCard() {
                    Column() {
                      SettingsToggleRow({
                        icon: AppIcon.NOTIFICATION,
                        title: '推送通知',
                        subtitle: '当前设备提醒总开关（本地偏好）',
                        isOn: this.pushNotifyEnabled,
                        onToggle: (next: boolean) => { this.togglePushNotify(next); }
                      })
                      SettingsRowDivider()
                      ForEach(this.buildSystemEntries(), (item: SettingsEntryItem, index: number) => {
                        if (index > 0) {
                          SettingsRowDivider()
                        }
                        SettingsEntryRow({ item })
                      }, (item: SettingsEntryItem) => item.title)
                    }
                  }
                }
                .padding(this.getSectionMargin(0))

                // === 隐私与安全 ===
                SettingsGroupLabel({ label: '隐私与安全' })
                Column() {
                  SettingsSectionCard() {
                    Column() {
                      SettingsToggleRow({
                        icon: AppIcon.LOCK,
                        title: '私密账号',
                        subtitle: '开启后动态仅对关注者可见',
                        isOn: this.privacyPrivate,
                        pending: this.privacyPending,
                        onToggle: (next: boolean) => { this.togglePrivacyPrivate(next); }
                      })
                      SettingsRowDivider()
                      SettingsToggleRow({
                        icon: AppIcon.LOCATION,
                        title: '位置共享',
                        subtitle: '控制附近的人是否可以看到你的距离',
                        isOn: this.locationVisible,
                        pending: this.locationPending,
                        onToggle: (next: boolean) => { this.toggleLocationVisible(next); }
                      })
                      SettingsRowDivider()
                      ForEach(this.buildPrivacyEntries(), (item: SettingsEntryItem, index: number) => {
                        if (index > 0) {
                          SettingsRowDivider()
                        }
                        SettingsEntryRow({ item })
                      }, (item: SettingsEntryItem) => item.title)
                      SettingsRowDivider()
                      SettingsEntryRow({ item: this.buildSecurityEntry() })
                    }
                  }
                }
                .padding(this.getSectionMargin(0))

                // === 通用 ===
                SettingsGroupLabel({ label: '通用' })
                Column() {
                  SettingsSectionCard() {
                    Column() {
                      ForEach(this.buildAccountEntries(), (item: SettingsEntryItem, index: number) => {
                        if (index > 0) {
                          SettingsRowDivider()
                        }
                        SettingsEntryRow({ item })
                      }, (item: SettingsEntryItem) => item.title)
                      SettingsRowDivider()
                      SettingsEntryRow({ item: {
                        icon: AppIcon.SETTINGS,
                        title: '高级设置',
                        subtitle: '开发者选项、诊断与缓存工具',
                        action: () => AppRouter.push(PageNames.ADVANCED_SETTINGS)
                      } })
                      SettingsRowDivider()
                      SettingsEntryRow({ item: {
                        icon: AppIcon.INFO,
                        title: '关于',
                        subtitle: this.systemMeta ? `版本 ${this.systemMeta.version}` : '版本信息加载中…',
                        action: () => { this.loadSystemMeta(true); }
                      } })
                    }
                  }
                }
                .padding(this.getSectionMargin(0))

                // === 平台信息（本端补充运维卡，前端无对应分组） ===
                SettingsGroupLabel({ label: '平台信息' })
                Column() {
                  SettingsSectionCard() {
                    Column() {
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
                      AnnouncementSummaryCard({
                        items: this.announcements,
                        openAll: () => {
                          const params = new AnnouncementsRouteParam();
                          params.items = this.announcements as Object[];
                          AppRouter.push(PageNames.ANNOUNCEMENTS, params);
                        }
                      })
                      SupportConfigCard({ config: this.clientConfig })
                    }
                  }
                }
                .padding(this.getSectionMargin(0))

                if (this.isLoggedIn) {
                  Button('退出登录')
                    .width('100%')
                    .height(DesignConstants.COMPONENT.LOGOUT_BUTTON_HEIGHT)
                    .backgroundColor(DesignConstants.COLOR.TRANSPARENT)
                    .fontColor(this.getCurrentThemePalette().error)
                    .fontSize(DesignConstants.FONT.SIZE.LG)
                    .type(ButtonType.Normal)
                    .borderRadius(DesignConstants.SHAPE.BUTTON)
                    .margin({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, top: DesignConstants.SPACING.LG, bottom: DesignConstants.SPACING.XXL })
                    .onClick(() => { this.showLogoutConfirm = true; })
                }
              }
              .width('100%')
            }
          }
          .scrollBar(BarState.Off)
          .padding({ bottom: SafeAreaUtils.resolveBottomPadding(0, DesignConstants.SPACING.XXL, DesignConstants.SPACING.LG) })
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
      // 画布契约样板：页面整底 pageBackground 连续（header 默认 bar 变体同画布底）
      .backgroundColor(this.getCurrentThemePalette().pageBackground)
      .bindSheet($$this.showThemeSheet, this.ThemeSheet(), {
        height: SheetSize.FIT_CONTENT,
        showClose: true,
        dragBar: true,
        backgroundColor: this.getCurrentThemePalette().sheetBackground
      })

      // 退出登录二次确认
      SystemDialog({
        visible: this.showLogoutConfirm,
        title: '退出登录',
        description: '退出后需要重新登录',
        confirmText: '退出登录',
        confirmVariant: 'danger',
        onCancel: () => { this.showLogoutConfirm = false; },
        onConfirm: () => {
          this.showLogoutConfirm = false;
          this.logout();
        }
      })
      }
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
              .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getThemeOptionPalette(option).borderColor })

            Blank()
              .width('70%')
              .height(AppThemeVisuals.getThemePreviewStyle(this.getThemeOptionPalette(option).id).primaryButton.height)
              .borderRadius(AppThemeVisuals.getThemePreviewStyle(this.getThemeOptionPalette(option).id).primaryButton.borderRadius)
              .backgroundColor(AppThemeVisuals.getThemePreviewStyle(this.getThemeOptionPalette(option).id).primaryButton.backgroundColor)

            Row({ space: DesignConstants.SPACING.MD2 }) {
              Blank().layoutWeight(1).height(DesignConstants.COMPONENT.ACTION_BUTTON_HEIGHT_MD).borderRadius(AppThemeVisuals.getThemePreviewStyle(this.getThemeOptionPalette(option).id).tileRadius).backgroundColor(this.getThemeOptionPalette(option).surfaceSubtle)
              Blank().layoutWeight(1).height(DesignConstants.COMPONENT.ACTION_BUTTON_HEIGHT_MD).borderRadius(AppThemeVisuals.getThemePreviewStyle(this.getThemeOptionPalette(option).id).tileRadius).backgroundColor(this.getThemeOptionPalette(option).surfaceSubtle)
            }
            .width('100%')

            Column() {
              Row() {}
                .width('100%')
                .height(DesignConstants.FONT.SIZE.XL)
                .borderRadius(AppThemeVisuals.getTabBarVisualStyle(this.getThemeOptionPalette(option).id).radius)
                .backgroundColor(AppThemeVisuals.getTabBarVisualStyle(this.getThemeOptionPalette(option).id).backgroundColor)

              Row() {}
                .width(DesignConstants.COMPONENT.THEME_PREVIEW_BAR_WIDTH)
                .height(DesignConstants.COMPONENT.THEME_PREVIEW_BAR_HEIGHT)
                .margin({
                  top: DesignConstants.COMPONENT.THEME_PREVIEW_INDICATOR_OVERLAP,
                  left: DesignConstants.SPACING.SM
                })
                .borderRadius(DesignConstants.RADIUS.FULL)
                .backgroundColor(AppThemeVisuals.getTabBarVisualStyle(this.getThemeOptionPalette(option).id).activeBackgroundColor)
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
    .border({ width: this.isThemeOptionActive(option) ? DesignConstants.BORDER_WIDTH.REGULAR : DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.isThemeOptionActive(option) ? this.getThemeOptionPalette(option).accentBackground : this.getThemeOptionPalette(option).borderColor })
    .shadow(this.isThemeOptionActive(option) ? this.getThemeOptionPalette(option).elevatedShadow : this.getThemeOptionPalette(option).surfaceShadow)
    // 选中底色/边框/阴影切换过渡与按压缩放合并为一处 spring 动画
    // （对齐仓内统一 springMotion 手感，P3-1/P3-3：消除重复 .animation）
    .scale(this.themeCardPressed === option.mode && !this.isThemeOptionActive(option) ?
      { x: DesignConstants.INTERACTION.SCALE_PRESSED_INPUT_ROW, y: DesignConstants.INTERACTION.SCALE_PRESSED_INPUT_ROW } :
      { x: 1, y: 1 })
    .animation({
      curve: curves.springMotion(
        DesignConstants.INTERACTION.SPRING_RESPONSE_PRESS,
        DesignConstants.INTERACTION.SPRING_DAMPING_PRESS
      )
    })
    .onTouch((event: TouchEvent) => {
      if (event.type === TouchType.Down) {
        this.themeCardPressed = option.mode;
      } else if (event.type === TouchType.Up || event.type === TouchType.Cancel) {
        this.themeCardPressed = '';
      }
    })
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