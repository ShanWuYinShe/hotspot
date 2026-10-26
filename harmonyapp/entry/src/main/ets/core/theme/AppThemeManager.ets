import { StorageKeys } from '../../constants/AppConstants';
import { DesignConstants } from '../../constants/DesignConstants';
import { PreferencesUtil } from '../storage/PreferencesUtil';
import { SystemSurfaceStyle, SystemUiThemeStore } from '../utils/SystemUiThemeStore';
import {
  AppThemeEffectTokens,
  AppThemeMetricTokens,
  AppThemePalette,
  AppThemeShapeTokens,
  AppThemeShadow,
  AppThemeVisualTokens,
  DEFAULT_DARK_MODE_PREFERENCE,
  DEFAULT_RESOLVED_THEME_ID,
  DEFAULT_THEME_MODE,
  DarkModePreference,
  KEY_DARK_MODE_PREFERENCE,
  KEY_RESOLVED_THEME_ID,
  KEY_THEME_MODE,
  ResolvedThemeId,
  ThemeMode,
  ThemeOptionItem
} from './AppThemeTypes';
import { AmbientColorConfig, ColorConfig, ThemeConstants, ThemeShadows } from './ThemeConstants';
import { AppThemeVisuals } from './AppThemeVisuals';

const DEFAULT_THEME_SHAPE: AppThemeShapeTokens = {
  surfaceCard: DesignConstants.SHAPE.CARD,
  heroCard: DesignConstants.SHAPE.HERO_CARD,
  compactCard: DesignConstants.SHAPE.COMPACT_CARD,
  input: DesignConstants.SHAPE.INPUT,
  button: DesignConstants.SHAPE.BUTTON,
  chip: DesignConstants.SHAPE.CHIP,
  tabBar: DesignConstants.SHAPE.TAB_BAR,
  previewCard: DesignConstants.SHAPE.PREVIEW_CARD,
  listItem: DesignConstants.SHAPE.LIST_ITEM,
  iconTile: DesignConstants.SHAPE.ICON_TILE
};

const DEFAULT_THEME_METRICS: AppThemeMetricTokens = {
  cardPadding: DesignConstants.SPACING.LG,
  heroPadding: DesignConstants.SPACING.XL,
  metricCardPadding: DesignConstants.SPACING.LG,
  profilePrimaryButtonHeight: DesignConstants.COMPONENT.BUTTON_HEIGHT,
  profilePrimaryButtonHorizontalPadding: DesignConstants.SPACING.XXL,
  secondaryButtonHorizontalPadding: DesignConstants.SPACING.XL,
  profileAvatarSize: 68,
  profileTitleFontSize: DesignConstants.FONT.SIZE.XL,
  profileSubtitleFontSize: DesignConstants.FONT.SIZE.LG,
  profileMetricMinHeight: 164,
  profileMetricValueFontSize: DesignConstants.FONT.SIZE.XL,
  actionButtonHeight: 36,
  listItemMinHeight: 72,
  tabBarHeight: 56,
  createButtonWidth: 54,
  createButtonHeight: 34,
  tabBarFloatingMargin: DesignConstants.SPACING.SM,
  chatListRadius: DesignConstants.SHAPE.CARD,
  chatSendButtonHeight: DesignConstants.CHAT.SEND_BUTTON_HEIGHT,
  previewHeroAvatarSize: 28,
  previewPrimaryButtonHeight: 14,
  textureOpacity: DesignConstants.EFFECT.TEXTURE_SOFT
};


function cloneShadow(shadow: AppThemeShadow): AppThemeShadow {
  return new AppThemeShadow(shadow.radius, shadow.color, shadow.offsetX, shadow.offsetY);
}

function buildVisualTokens(
  shape: AppThemeShapeTokens,
  metrics: AppThemeMetricTokens,
  effects: AppThemeEffectTokens
): AppThemeVisualTokens {
  return {
    shape,
    metrics,
    effects
  };
}

function buildPalette(
  id: ResolvedThemeId,
  label: string,
  description: string,
  colors: ColorConfig,
  shadows: ThemeShadows,
  visuals: AppThemeVisualTokens,
  preferDarkContent: boolean
): AppThemePalette {
  return {
    id,
    label,
    description,
    pageBackground: colors.pageBackground,
    surfaceBackground: colors.surfaceBackground,
    surfaceSubtle: colors.surfaceSubtle,
    surfaceElevated: colors.surfaceElevated,
    borderColor: colors.borderColor,
    floatingBorderColor: colors.floatingBorderColor,
    sheetBackground: colors.sheetBackground,
    accentBackground: colors.accentBackground,
    accentText: colors.accentText,
    accentMuted: colors.accentMuted,
    chatBackground: colors.chatBackground,
    systemSurfaceBaseColor: colors.systemSurfaceBaseColor,
    online: colors.online,
    preferDarkContent,
    isDarkTheme: colors.isDarkTheme,
    textPrimary: colors.textPrimary,
    textSecondary: colors.textSecondary,
    textTertiary: colors.textTertiary,
    iconMuted: colors.iconMuted,
    buttonSecondaryBackground: colors.buttonSecondaryBackground,
    buttonSecondaryText: colors.buttonSecondaryText,
    primaryButtonBorderColor: colors.primaryButtonBorderColor,
    inputBackgroundColor: colors.inputBackgroundColor,
    quickActionBackgroundColor: colors.quickActionBackgroundColor,
    tabBarBackground: colors.tabBarBackground,
    tabBarInactive: colors.tabBarInactive,
    tabBarActiveBackground: colors.tabBarActiveBackground,
    surfaceShadow: shadows.surface,
    elevatedShadow: shadows.elevated,

    controlShadow: shadows.control,
    tabBarShadow: shadows.tabBar,
    overlayStrong: colors.overlayStrong,
    warning: colors.warning,
    success: colors.success,
    error: colors.error,
    info: colors.info,
    mapUser: colors.mapUser,
    mapGroup: colors.mapGroup,
    mapMoment: colors.mapMoment,
    dangerSoft: colors.dangerSoft,
    warningSoft: colors.warningSoft,
    cardScrimTop: colors.cardScrimTop,
    cardScrimTopMid: colors.cardScrimTopMid,
    cardScrimBottomStrong: colors.cardScrimBottomStrong,
    cardScrimBottomMid: colors.cardScrimBottomMid,
    cardScrimBottomSoft: colors.cardScrimBottomSoft,
    cardFallbackStart: colors.cardFallbackStart,
    cardFallbackMid: colors.cardFallbackMid,
    visuals
  } as AppThemePalette;
}

const DEFAULT_CLEAN_LIGHT_VISUALS = buildVisualTokens(
  DEFAULT_THEME_SHAPE,
  DEFAULT_THEME_METRICS,
  {
    // 头像抬升阴影（对齐前端 --cst-elevation-profile-avatar 0 6px 20px 0.22）
    avatarShadow: new AppThemeShadow(20, '#38000000', 0, 6),
    previewShadow: new AppThemeShadow(24, '#29000000', 0, 8)
  }
);

const DEFAULT_CLEAN_DARK_VISUALS = buildVisualTokens(
  DEFAULT_THEME_SHAPE,
  DEFAULT_THEME_METRICS,
  {
    // 头像抬升阴影（深色增强透明度）
    avatarShadow: new AppThemeShadow(20, '#66000000', 0, 6),
    previewShadow: new AppThemeShadow(28, '#59000000', 0, 8)
  }
);

const VIBRANT_THEME_SHAPE: AppThemeShapeTokens = {
  surfaceCard: 22,
  heroCard: 28,
  compactCard: DesignConstants.RADIUS.XL,
  input: 18,
  button: 28,
  chip: DesignConstants.SHAPE.CHIP,
  tabBar: 28,
  previewCard: 22,
  listItem: 22,
  iconTile: DesignConstants.RADIUS.MD
};

const VIBRANT_THEME_METRICS: AppThemeMetricTokens = {
  cardPadding: 16,
  heroPadding: DesignConstants.SPACING.XL,
  metricCardPadding: DesignConstants.SPACING.LG,
  profilePrimaryButtonHeight: DesignConstants.COMPONENT.BUTTON_HEIGHT,
  profilePrimaryButtonHorizontalPadding: DesignConstants.SPACING.XXL,
  secondaryButtonHorizontalPadding: DesignConstants.SPACING.XXL,
  profileAvatarSize: 68,
  profileTitleFontSize: DesignConstants.FONT.SIZE.XL,
  profileSubtitleFontSize: DesignConstants.FONT.SIZE.LG,
  profileMetricMinHeight: 164,
  profileMetricValueFontSize: DesignConstants.FONT.SIZE.XL,
  actionButtonHeight: 40,
  listItemMinHeight: 72,
  tabBarHeight: 60,
  createButtonWidth: 48,
  createButtonHeight: 48,
  tabBarFloatingMargin: DesignConstants.SPACING.MD,
  chatListRadius: DesignConstants.SHAPE.CARD,
  chatSendButtonHeight: DesignConstants.CHAT.SEND_BUTTON_HEIGHT,
  previewHeroAvatarSize: 32,
  previewPrimaryButtonHeight: 18,
  textureOpacity: 0
};

const VIBRANT_LIGHT_VISUALS = buildVisualTokens(
  VIBRANT_THEME_SHAPE,
  VIBRANT_THEME_METRICS,
  {
    avatarShadow: cloneShadow(ThemeConstants.SHADOWS.VIBRANT_LIGHT.surface),
    previewShadow: cloneShadow(ThemeConstants.SHADOWS.VIBRANT_LIGHT.elevated)
  }
);

const VIBRANT_DARK_VISUALS = buildVisualTokens(
  VIBRANT_THEME_SHAPE,
  VIBRANT_THEME_METRICS,
  {
    avatarShadow: cloneShadow(ThemeConstants.SHADOWS.VIBRANT_DARK.surface),
    previewShadow: cloneShadow(ThemeConstants.SHADOWS.VIBRANT_DARK.elevated)
  }
);

const DEFAULT_CLEAN_LIGHT = buildPalette(
  'default_clean_light',
  '清野',
  '自然纸面、森林绿强调',
  ThemeConstants.DEFAULT_CLEAN_LIGHT_COLORS,
  ThemeConstants.SHADOWS.DEFAULT_CLEAN_LIGHT,
  DEFAULT_CLEAN_LIGHT_VISUALS,
  true
);

const DEFAULT_CLEAN_DARK = buildPalette(
  'default_clean_dark',
  '清野',
  '自然纸面、森林绿强调',
  ThemeConstants.DEFAULT_CLEAN_DARK_COLORS,
  ThemeConstants.SHADOWS.DEFAULT_CLEAN_DARK,
  DEFAULT_CLEAN_DARK_VISUALS,
  false
);

const VIBRANT_LIGHT = buildPalette(
  'vibrant_light',
  '星际',
  '冷色玻璃、蓝调强调',
  ThemeConstants.VIBRANT_LIGHT_COLORS,
  ThemeConstants.SHADOWS.VIBRANT_LIGHT,
  VIBRANT_LIGHT_VISUALS,
  true
);

const VIBRANT_DARK = buildPalette(
  'vibrant_dark',
  '星际',
  '深空蓝底、明快蓝调强调',
  ThemeConstants.VIBRANT_DARK_COLORS,
  ThemeConstants.SHADOWS.VIBRANT_DARK,
  VIBRANT_DARK_VISUALS,
  false
);

// 水墨：宣纸扁平语言，形制/尺寸沿用默认 tokens（对齐清野做法）
const INKWASH_LIGHT_VISUALS = buildVisualTokens(
  DEFAULT_THEME_SHAPE,
  DEFAULT_THEME_METRICS,
  {
    avatarShadow: cloneShadow(ThemeConstants.SHADOWS.INKWASH_LIGHT.surface),
    previewShadow: cloneShadow(ThemeConstants.SHADOWS.INKWASH_LIGHT.elevated)
  }
);

const INKWASH_DARK_VISUALS = buildVisualTokens(
  DEFAULT_THEME_SHAPE,
  DEFAULT_THEME_METRICS,
  {
    avatarShadow: cloneShadow(ThemeConstants.SHADOWS.INKWASH_DARK.surface),
    previewShadow: cloneShadow(ThemeConstants.SHADOWS.INKWASH_DARK.elevated)
  }
);

const INKWASH_LIGHT = buildPalette(
  'inkwash_light',
  '水墨',
  '宣纸墨韵、朱砂强调',
  ThemeConstants.INKWASH_LIGHT_COLORS,
  ThemeConstants.SHADOWS.INKWASH_LIGHT,
  INKWASH_LIGHT_VISUALS,
  true
);

const INKWASH_DARK = buildPalette(
  'inkwash_dark',
  '水墨',
  '墨夜留白、暖赭强调',
  ThemeConstants.INKWASH_DARK_COLORS,
  ThemeConstants.SHADOWS.INKWASH_DARK,
  INKWASH_DARK_VISUALS,
  false
);

const THEME_OPTIONS: ThemeOptionItem[] = [
  { mode: 'follow_system', label: '跟随系统', description: '自动匹配系统明暗模式' } as ThemeOptionItem,
  { mode: 'vibrant', label: '星际', description: '冷色玻璃、蓝调强调' } as ThemeOptionItem,
  { mode: 'default_clean', label: '清野', description: '自然纸面、森林绿强调' } as ThemeOptionItem,
  { mode: 'inkwash', label: '水墨', description: '宣纸墨韵、朱砂强调' } as ThemeOptionItem
];

export class AppThemeManager {
  static async initialize(): Promise<void> {
    const storedThemeMode = await PreferencesUtil.get<string>(StorageKeys.THEME_MODE, DEFAULT_THEME_MODE);
    const themeMode = AppThemeManager.normalizeThemeMode(storedThemeMode);
    AppStorage.setOrCreate<string>(KEY_THEME_MODE, themeMode);
    // 明暗偏好迁移：旧 settings_dark_forced=true 的用户落到 dark，并清理旧键（单通道唯一来源）
    let storedDarkPref = await PreferencesUtil.get<string>(StorageKeys.DARK_MODE_PREFERENCE, '');
    if (!storedDarkPref) {
      const forcedDark = await PreferencesUtil.get<boolean>('settings_dark_forced', false);
      if (forcedDark) {
        storedDarkPref = 'dark';
        await PreferencesUtil.set(StorageKeys.DARK_MODE_PREFERENCE, 'dark');
      }
      await PreferencesUtil.remove('settings_dark_forced');
    }
    AppStorage.setOrCreate<string>(KEY_DARK_MODE_PREFERENCE,
      AppThemeManager.normalizeDarkModePreference(storedDarkPref));
    AppThemeManager.refreshResolvedTheme();
  }

  static async setThemeMode(themeMode: ThemeMode): Promise<void> {
    const normalizedThemeMode = AppThemeManager.normalizeThemeMode(themeMode);
    await PreferencesUtil.set(StorageKeys.THEME_MODE, normalizedThemeMode);
    AppStorage.setOrCreate<string>(KEY_THEME_MODE, normalizedThemeMode);
    AppThemeManager.refreshResolvedTheme();
  }

  static getThemeMode(): ThemeMode {
    return AppThemeManager.normalizeThemeMode(AppStorage.get<string>(KEY_THEME_MODE) ?? DEFAULT_THEME_MODE);
  }

  /** 明暗三档单通道写入口（主题族走 setThemeMode，明暗走这里，双通道旧键已迁移删除） */
  static async setDarkModePreference(pref: DarkModePreference): Promise<void> {
    const normalized = AppThemeManager.normalizeDarkModePreference(pref);
    await PreferencesUtil.set(StorageKeys.DARK_MODE_PREFERENCE, normalized);
    AppStorage.setOrCreate<string>(KEY_DARK_MODE_PREFERENCE, normalized);
    AppThemeManager.refreshResolvedTheme();
  }

  static getDarkModePreference(): DarkModePreference {
    return AppThemeManager.normalizeDarkModePreference(
      AppStorage.get<string>(KEY_DARK_MODE_PREFERENCE) ?? DEFAULT_DARK_MODE_PREFERENCE);
  }

  /** 生效的深色状态：auto 看系统，manual 直接取偏好 */
  static isEffectiveDarkMode(): boolean {
    const pref = AppThemeManager.getDarkModePreference();
    if (pref === 'dark') {
      return true;
    }
    if (pref === 'light') {
      return false;
    }
    return AppStorage.get<boolean>('isDarkMode') === true;
  }

  private static getResolvedThemeId(): ResolvedThemeId {
    const resolvedThemeId = AppStorage.get<string>(KEY_RESOLVED_THEME_ID) ?? DEFAULT_RESOLVED_THEME_ID;
    return AppThemeManager.normalizeResolvedThemeId(resolvedThemeId);
  }

  static onSystemColorModeChange(isDarkMode: boolean): void {
    AppStorage.setOrCreate<boolean>('isDarkMode', isDarkMode);
    // refresh 按偏好解析：auto 时结果跟随系统变化，manual 时结果不变
    AppThemeManager.refreshResolvedTheme();
  }

  private static getCurrentPalette(): AppThemePalette {
    return AppThemeManager.getPalette();
  }

  static getPalette(themeId?: string): AppThemePalette {
    if (themeId) {
      return AppThemeManager.getPaletteById(themeId);
    }
    return AppThemeManager.getPaletteById(AppThemeManager.getResolvedThemeId());
  }

  static getSafeCurrentPalette(): AppThemePalette {
    try {
      return AppThemeManager.getCurrentPalette();
    } catch (_) {
      return VIBRANT_LIGHT;
    }
  }

  static getPaletteForMode(themeMode: ThemeMode): AppThemePalette {
    const resolvedThemeId = AppThemeManager.resolveThemeId(
      AppThemeManager.normalizeThemeMode(themeMode)
    );
    return AppThemeManager.getPaletteById(resolvedThemeId);
  }

  static getPaletteById(resolvedThemeId: string): AppThemePalette {
    switch (AppThemeManager.normalizeResolvedThemeId(resolvedThemeId)) {
      case 'vibrant_dark':
        return VIBRANT_DARK;
      case 'vibrant_light':
        return VIBRANT_LIGHT;
      case 'inkwash_dark':
        return INKWASH_DARK;
      case 'inkwash_light':
        return INKWASH_LIGHT;
      case 'default_clean_dark':
        return DEFAULT_CLEAN_DARK;
      case 'default_clean_light':
      default:
        return DEFAULT_CLEAN_LIGHT;
    }
  }

  /** 氛围画布色板（对齐 web --cst-ambient-*，按解析后的主题 id 取） */
  static getAmbientById(resolvedThemeId: string): AmbientColorConfig {
    switch (AppThemeManager.normalizeResolvedThemeId(resolvedThemeId)) {
      case 'vibrant_dark':
        return ThemeConstants.AMBIENT.VIBRANT_DARK;
      case 'vibrant_light':
        return ThemeConstants.AMBIENT.VIBRANT_LIGHT;
      case 'inkwash_dark':
        return ThemeConstants.AMBIENT.INKWASH_DARK;
      case 'inkwash_light':
        return ThemeConstants.AMBIENT.INKWASH_LIGHT;
      case 'default_clean_dark':
        return ThemeConstants.AMBIENT.DEFAULT_CLEAN_DARK;
      case 'default_clean_light':
      default:
        return ThemeConstants.AMBIENT.DEFAULT_CLEAN_LIGHT;
    }
  }

  static getAmbient(): AmbientColorConfig {
    return AppThemeManager.getAmbientById(AppThemeManager.getResolvedThemeId());
  }

  /** 主题世界语言（对齐 web AmbientBackdrop：vibrant 星际 / forest 清野 / inkwash 水墨） */
  static getAmbientWorld(resolvedThemeId: string): string {
    switch (AppThemeManager.normalizeResolvedThemeId(resolvedThemeId)) {
      case 'inkwash_dark':
      case 'inkwash_light':
        return 'inkwash';
      case 'default_clean_dark':
      case 'default_clean_light':
        return 'forest';
      default:
        return 'vibrant';
    }
  }

  static getCurrentSurfaceStyle(): SystemSurfaceStyle {
    const palette = AppThemeManager.getSafeCurrentPalette();
    return {
      baseColor: palette.systemSurfaceBaseColor,
      overlayOpacity: 1,
      preferDarkContent: palette.preferDarkContent
    };
  }

  private static syncSystemSurface(): void {
    SystemUiThemeStore.setStyle(AppThemeManager.getCurrentSurfaceStyle());
  }

  static getThemeOptions(): ThemeOptionItem[] {
    return THEME_OPTIONS;
  }

  static getThemeLabel(themeMode: string): string {
    const normalizedThemeMode = AppThemeManager.normalizeThemeMode(themeMode);
    const match = THEME_OPTIONS.find((item: ThemeOptionItem) => item.mode === normalizedThemeMode);
    return match ? match.label : '当前默认';
  }

  static getSoftStatusBackground(status: 'neutral' | 'success' | 'warning' | 'danger' | 'info'): string {
    const palette = AppThemeManager.getSafeCurrentPalette();
    const isDark = palette.isDarkTheme;
    // 主题化派生：palette 语义色 + withAlpha 叠加，等效前端 color-mix(in srgb, 色 12~14%, transparent)
    switch (status) {
      case 'success':
        return AppThemeVisuals.withAlpha(palette.success, isDark ? 0.14 : 0.12);
      case 'warning':
        return AppThemeVisuals.withAlpha(palette.warning, isDark ? 0.14 : 0.12);
      case 'danger':
        return AppThemeVisuals.withAlpha(palette.error, isDark ? 0.14 : 0.12);
      case 'info':
        return AppThemeVisuals.withAlpha(palette.info, isDark ? 0.14 : 0.12);
      case 'neutral':
      default:
        return palette.surfaceSubtle;
    }
  }

  private static refreshResolvedTheme(): void {
    const resolvedThemeId = AppThemeManager.resolveThemeId(AppThemeManager.getThemeMode());
    AppStorage.setOrCreate<string>(KEY_RESOLVED_THEME_ID, resolvedThemeId);
    AppThemeManager.syncSystemSurface();
  }

  /** 供外部在大范围 UI 重建（如登录态切换）后强制重解主题并同步窗口/材质底色 */
  static refreshResolvedThemePublic(): void {
    AppThemeManager.refreshResolvedTheme();
  }

  private static resolveThemeId(themeMode: ThemeMode): ResolvedThemeId {
    // 明暗偏好三档：auto 看系统 isDarkMode，manual 直接取偏好（与主题族正交）
    const effectiveDark = AppThemeManager.isEffectiveDarkMode();
    switch (themeMode) {
      case 'follow_system':
        // 跟随系统 = 默认主题族（vibrant 星际，对齐前端默认）随生效明暗切换
        return effectiveDark ? 'vibrant_dark' : 'vibrant_light';
      case 'vibrant':
        return effectiveDark ? 'vibrant_dark' : 'vibrant_light';
      case 'default_clean':
        return effectiveDark ? 'default_clean_dark' : 'default_clean_light';
      case 'inkwash':
        return effectiveDark ? 'inkwash_dark' : 'inkwash_light';
      default:
        return DEFAULT_RESOLVED_THEME_ID;
    }
  }

  private static normalizeThemeMode(themeMode: string): ThemeMode {
    switch (themeMode) {
      case 'default_clean':
      case 'follow_system':
      case 'vibrant':
      case 'inkwash':
        return themeMode;
      case 'figma_neu':
        // 已删除主题族迁移：存量偏好一律映射 vibrant（明暗由偏好/系统重解，与原明暗对应）
        return 'vibrant';
      default:
        return DEFAULT_THEME_MODE;
    }
  }

  static normalizeDarkModePreference(pref: string): DarkModePreference {
    switch (pref) {
      case 'light':
      case 'dark':
      case 'auto':
        return pref;
      default:
        return DEFAULT_DARK_MODE_PREFERENCE;
    }
  }

  private static normalizeResolvedThemeId(resolvedThemeId: string): ResolvedThemeId {
    switch (resolvedThemeId) {
      case 'default_clean_dark':
      case 'default_clean_light':
      case 'vibrant_dark':
      case 'vibrant_light':
      case 'inkwash_dark':
      case 'inkwash_light':
        return resolvedThemeId;
      case 'figma_neu_dark':
        // 已删除主题族迁移：resolved id 按明暗映射 vibrant 对应档
        return 'vibrant_dark';
      case 'figma_neu_light':
        return 'vibrant_light';
      default:
        return DEFAULT_RESOLVED_THEME_ID;
    }
  }
}
