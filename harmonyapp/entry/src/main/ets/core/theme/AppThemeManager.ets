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
  DEFAULT_RESOLVED_THEME_ID,
  DEFAULT_THEME_MODE,
  KEY_RESOLVED_THEME_ID,
  KEY_THEME_MODE,
  ResolvedThemeId,
  ThemeMode,
  ThemeOptionItem
} from './AppThemeTypes';
import { ColorConfig, ThemeConstants, ThemeShadows } from './ThemeConstants';

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

const SOFT_THEME_SHAPE: AppThemeShapeTokens = {
  surfaceCard: DesignConstants.SHAPE.CARD + 6,
  heroCard: DesignConstants.SHAPE.HERO_CARD + 8,
  compactCard: DesignConstants.SHAPE.COMPACT_CARD + 8,
  input: DesignConstants.SHAPE.INPUT + 4,
  button: DesignConstants.SHAPE.BUTTON + 4,
  chip: DesignConstants.SHAPE.CHIP + 4,
  tabBar: DesignConstants.SHAPE.TAB_BAR + 8,
  previewCard: DesignConstants.SHAPE.PREVIEW_CARD + 8,
  listItem: DesignConstants.SHAPE.LIST_ITEM + 6,
  iconTile: DesignConstants.SHAPE.ICON_TILE + 6
};
const DEFAULT_THEME_METRICS: AppThemeMetricTokens = {
  cardPadding: DesignConstants.SPACING.LG,
  heroPadding: DesignConstants.SPACING.XL,
  metricCardPadding: DesignConstants.SPACING.LG,
  profilePrimaryButtonHeight: DesignConstants.COMPONENT.BUTTON_HEIGHT,
  profilePrimaryButtonHorizontalPadding: DesignConstants.SPACING.XXL,
  profileAvatarSize: 96,
  profileTitleFontSize: DesignConstants.FONT.SIZE.XL,
  profileSubtitleFontSize: DesignConstants.FONT.SIZE.LG,
  profileMetricMinHeight: 164,
  profileMetricValueFontSize: 22,
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

const SOFT_THEME_METRICS: AppThemeMetricTokens = {
  cardPadding: DesignConstants.SPACING.XL + 2,
  heroPadding: DesignConstants.SPACING.XXL + 4,
  metricCardPadding: DesignConstants.SPACING.XL + 2,
  profilePrimaryButtonHeight: 58,
  profilePrimaryButtonHorizontalPadding: DesignConstants.SPACING.XXL + 14,
  profileAvatarSize: 128,
  profileTitleFontSize: DesignConstants.FONT.SIZE.XXL + 4,
  profileSubtitleFontSize: DesignConstants.FONT.SIZE.XL,
  profileMetricMinHeight: 188,
  profileMetricValueFontSize: 28,
  actionButtonHeight: 48,
  listItemMinHeight: 92,
  tabBarHeight: 70,
  createButtonWidth: 82,
  createButtonHeight: 48,
  tabBarFloatingMargin: DesignConstants.SPACING.SM,
  chatListRadius: DesignConstants.SHAPE.CARD + 8,
  chatSendButtonHeight: DesignConstants.CHAT.SEND_BUTTON_HEIGHT + 6,
  previewHeroAvatarSize: 44,
  previewPrimaryButtonHeight: 22,
  textureOpacity: DesignConstants.EFFECT.TEXTURE_STRONG
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
    pageGlow: colors.pageGlow,
    heroGlow: colors.heroGlow,
    textureTint: colors.textureTint,
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
    preferDarkContent,
    isDarkTheme: colors.isDarkTheme,
    textPrimary: colors.textPrimary,
    textSecondary: colors.textSecondary,
    textTertiary: colors.textTertiary,
    textOnAccentMuted: colors.textOnAccentMuted,
    iconMuted: colors.iconMuted,
    buttonSecondaryBackground: colors.buttonSecondaryBackground,
    buttonSecondaryText: colors.buttonSecondaryText,
    tabBarBackground: colors.tabBarBackground,
    tabBarInactive: colors.tabBarInactive,
    tabBarActiveBackground: colors.tabBarActiveBackground,
    surfaceShadow: shadows.surface,
    elevatedShadow: shadows.elevated,
    accentShadow: shadows.accent,
    controlShadow: shadows.control,
    tabBarShadow: shadows.tabBar,
    overlayStrong: colors.overlayStrong,
    visuals
  } as AppThemePalette;
}

const DEFAULT_CLEAN_LIGHT_VISUALS = buildVisualTokens(
  DEFAULT_THEME_SHAPE,
  DEFAULT_THEME_METRICS,
  {
    avatarShadow: new AppThemeShadow(0, '#00000000', 0, 0),
    haloShadow: cloneShadow(ThemeConstants.SHADOWS.DEFAULT_CLEAN_LIGHT.control),
    previewShadow: cloneShadow(ThemeConstants.SHADOWS.DEFAULT_CLEAN_LIGHT.elevated),
    insetShadow: new AppThemeShadow(0, '#00000000', 0, 0),
    highlightShadow: new AppThemeShadow(0, '#00000000', 0, 0),
    tabActiveGlow: new AppThemeShadow(0, '#00000000', 0, 0)
  }
);

const DEFAULT_CLEAN_DARK_VISUALS = buildVisualTokens(
  DEFAULT_THEME_SHAPE,
  DEFAULT_THEME_METRICS,
  {
    avatarShadow: new AppThemeShadow(0, '#00000000', 0, 0),
    haloShadow: cloneShadow(ThemeConstants.SHADOWS.DEFAULT_CLEAN_DARK.control),
    previewShadow: cloneShadow(ThemeConstants.SHADOWS.DEFAULT_CLEAN_DARK.elevated),
    insetShadow: new AppThemeShadow(0, '#00000000', 0, 0),
    highlightShadow: new AppThemeShadow(0, '#00000000', 0, 0),
    tabActiveGlow: new AppThemeShadow(0, '#00000000', 0, 0)
  }
);

const SOFT_GALLERY_LIGHT_VISUALS = buildVisualTokens(
  SOFT_THEME_SHAPE,
  SOFT_THEME_METRICS,
  {
    avatarShadow: cloneShadow(ThemeConstants.SHADOWS.SOFT_GALLERY_LIGHT.elevated),
    haloShadow: new AppThemeShadow(36, '#30D4AF37', 0, 0),
    previewShadow: cloneShadow(ThemeConstants.SHADOWS.SOFT_GALLERY_LIGHT.elevated),
    insetShadow: new AppThemeShadow(10, '#14CDB89F', 0, 4),
    highlightShadow: new AppThemeShadow(18, '#36FFFFFF', 0, -3),
    tabActiveGlow: new AppThemeShadow(24, '#26D4AF37', 0, 0)
  }
);

const SOFT_GALLERY_DARK_VISUALS = buildVisualTokens(
  SOFT_THEME_SHAPE,
  SOFT_THEME_METRICS,
  {
    avatarShadow: cloneShadow(ThemeConstants.SHADOWS.SOFT_GALLERY_DARK.elevated),
    haloShadow: new AppThemeShadow(28, '#30FFD700', 0, 0),
    previewShadow: cloneShadow(ThemeConstants.SHADOWS.SOFT_GALLERY_DARK.elevated),
    insetShadow: new AppThemeShadow(10, '#14000000', 0, 4),
    highlightShadow: new AppThemeShadow(16, '#22FFFFFF', 0, -2),
    tabActiveGlow: new AppThemeShadow(22, '#30FFD700', 0, 0)
  }
);

const EDITORIAL_GLOW_SHAPE: AppThemeShapeTokens = {
  surfaceCard: DesignConstants.SHAPE.CARD + 10,
  heroCard: DesignConstants.SHAPE.HERO_CARD + 12,
  compactCard: DesignConstants.SHAPE.COMPACT_CARD + 10,
  input: DesignConstants.SHAPE.INPUT + 8,
  button: DesignConstants.SHAPE.BUTTON + 10,
  chip: DesignConstants.SHAPE.CHIP + 8,
  tabBar: DesignConstants.SHAPE.TAB_BAR + 12,
  previewCard: DesignConstants.SHAPE.PREVIEW_CARD + 12,
  listItem: DesignConstants.SHAPE.LIST_ITEM + 10,
  iconTile: DesignConstants.SHAPE.ICON_TILE + 10
};

const EDITORIAL_GLOW_METRICS: AppThemeMetricTokens = {
  cardPadding: DesignConstants.SPACING.XL + 6,
  heroPadding: DesignConstants.SPACING.XXL + 10,
  metricCardPadding: DesignConstants.SPACING.XL + 4,
  profilePrimaryButtonHeight: 60,
  profilePrimaryButtonHorizontalPadding: DesignConstants.SPACING.XXL + 18,
  profileAvatarSize: 144,
  profileTitleFontSize: DesignConstants.FONT.SIZE.XXXL - 2,
  profileSubtitleFontSize: DesignConstants.FONT.SIZE.XXL - 2,
  profileMetricMinHeight: 204,
  profileMetricValueFontSize: 32,
  actionButtonHeight: 52,
  listItemMinHeight: 96,
  tabBarHeight: 78,
  createButtonWidth: 84,
  createButtonHeight: 52,
  tabBarFloatingMargin: DesignConstants.SPACING.MD,
  chatListRadius: DesignConstants.SHAPE.CARD + 10,
  chatSendButtonHeight: DesignConstants.CHAT.SEND_BUTTON_HEIGHT + 8,
  previewHeroAvatarSize: 46,
  previewPrimaryButtonHeight: 24,
  textureOpacity: DesignConstants.EFFECT.TEXTURE_STRONG
};

const EDITORIAL_GLOW_LIGHT_VISUALS = buildVisualTokens(
  EDITORIAL_GLOW_SHAPE,
  EDITORIAL_GLOW_METRICS,
  {
    avatarShadow: cloneShadow(ThemeConstants.SHADOWS.EDITORIAL_GLOW_LIGHT.elevated),
    haloShadow: new AppThemeShadow(42, '#54FFF7EB', 0, 0),
    previewShadow: cloneShadow(ThemeConstants.SHADOWS.EDITORIAL_GLOW_LIGHT.elevated),
    insetShadow: new AppThemeShadow(16, '#1BE3D7CA', 0, 6),
    highlightShadow: new AppThemeShadow(24, '#55FFFFFF', 0, -4),
    tabActiveGlow: new AppThemeShadow(36, '#42FFFFFF', 0, 0)
  }
);

const EDITORIAL_GLOW_DARK_VISUALS = buildVisualTokens(
  EDITORIAL_GLOW_SHAPE,
  EDITORIAL_GLOW_METRICS,
  {
    avatarShadow: cloneShadow(ThemeConstants.SHADOWS.EDITORIAL_GLOW_DARK.elevated),
    haloShadow: new AppThemeShadow(36, '#2FF1D0B0', 0, 0),
    previewShadow: cloneShadow(ThemeConstants.SHADOWS.EDITORIAL_GLOW_DARK.elevated),
    insetShadow: new AppThemeShadow(18, '#2A0C1320', 0, 8),
    highlightShadow: new AppThemeShadow(20, '#26C7D0DC', 0, -4),
    tabActiveGlow: new AppThemeShadow(28, '#38F1D0B0', 0, 0)
  }
);

const DEFAULT_CLEAN_LIGHT = buildPalette(
  'default_clean_light',
  '当前默认',
  '清爽、干净、通用',
  ThemeConstants.DEFAULT_CLEAN_LIGHT_COLORS,
  ThemeConstants.SHADOWS.DEFAULT_CLEAN_LIGHT,
  DEFAULT_CLEAN_LIGHT_VISUALS,
  true
);

const DEFAULT_CLEAN_DARK = buildPalette(
  'default_clean_dark',
  '当前默认',
  '清爽、干净、通用',
  ThemeConstants.DEFAULT_CLEAN_DARK_COLORS,
  ThemeConstants.SHADOWS.DEFAULT_CLEAN_DARK,
  DEFAULT_CLEAN_DARK_VISUALS,
  false
);

const SOFT_GALLERY_LIGHT = buildPalette(
  'soft_gallery_light',
  'Tactile Minimalist',
  '奶油柔光、轻拟物、静谧触感',
  ThemeConstants.SOFT_GALLERY_LIGHT_COLORS,
  ThemeConstants.SHADOWS.SOFT_GALLERY_LIGHT,
  SOFT_GALLERY_LIGHT_VISUALS,
  true
);

const SOFT_GALLERY_DARK = buildPalette(
  'soft_gallery_dark',
  'Tactile Minimalist',
  '冷蓝深夜、柔光边缘、暖金焦点',
  ThemeConstants.SOFT_GALLERY_DARK_COLORS,
  ThemeConstants.SHADOWS.SOFT_GALLERY_DARK,
  SOFT_GALLERY_DARK_VISUALS,
  false
);

const EDITORIAL_GLOW_LIGHT = buildPalette(
  'editorial_glow_light',
  'Editorial Glow',
  '奶霜纸感、暖调浮雕、柔焦光晕',
  ThemeConstants.EDITORIAL_GLOW_LIGHT_COLORS,
  ThemeConstants.SHADOWS.EDITORIAL_GLOW_LIGHT,
  EDITORIAL_GLOW_LIGHT_VISUALS,
  true
);

const EDITORIAL_GLOW_DARK = buildPalette(
  'editorial_glow_dark',
  'Editorial Glow',
  '深海夜幕、微光描边、暖金聚焦',
  ThemeConstants.EDITORIAL_GLOW_DARK_COLORS,
  ThemeConstants.SHADOWS.EDITORIAL_GLOW_DARK,
  EDITORIAL_GLOW_DARK_VISUALS,
  false
);

const THEME_OPTIONS: ThemeOptionItem[] = [
  { mode: 'follow_system', label: '跟随系统', description: '自动匹配系统明暗模式' } as ThemeOptionItem,
  { mode: 'default_clean', label: '当前默认', description: '清爽、中性、稳妥的基础主题' } as ThemeOptionItem,
  { mode: 'soft_gallery', label: 'Tactile Minimalist', description: '浅色奶油柔光 / 深色夜蓝暖金' } as ThemeOptionItem,
  { mode: 'editorial_glow', label: 'Editorial Glow', description: '高保真纹理、柔光拟物、设计稿复刻基线' } as ThemeOptionItem
];

export class AppThemeManager {
  static async initialize(): Promise<void> {
    const storedThemeMode = await PreferencesUtil.get<string>(StorageKeys.THEME_MODE, DEFAULT_THEME_MODE);
    const themeMode = AppThemeManager.normalizeThemeMode(storedThemeMode);
    AppStorage.setOrCreate<string>(KEY_THEME_MODE, themeMode);
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

  static getResolvedThemeId(): ResolvedThemeId {
    const resolvedThemeId = AppStorage.get<string>(KEY_RESOLVED_THEME_ID) ?? DEFAULT_RESOLVED_THEME_ID;
    return AppThemeManager.normalizeResolvedThemeId(resolvedThemeId);
  }

  static onSystemColorModeChange(isDarkMode: boolean): void {
    AppStorage.setOrCreate<boolean>('isDarkMode', isDarkMode);
    AppThemeManager.refreshResolvedTheme();
  }

  static getCurrentPalette(): AppThemePalette {
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
      return DEFAULT_CLEAN_LIGHT;
    }
  }

  static getPaletteForMode(themeMode: ThemeMode, isDarkMode?: boolean): AppThemePalette {
    const resolvedThemeId = AppThemeManager.resolveThemeId(
      AppThemeManager.normalizeThemeMode(themeMode),
      isDarkMode ?? (AppStorage.get<boolean>('isDarkMode') === true)
    );
    return AppThemeManager.getPaletteById(resolvedThemeId);
  }

  static getPaletteById(resolvedThemeId: string): AppThemePalette {
    switch (AppThemeManager.normalizeResolvedThemeId(resolvedThemeId)) {
      case 'default_clean_dark':
        return DEFAULT_CLEAN_DARK;
      case 'soft_gallery_light':
        return SOFT_GALLERY_LIGHT;
      case 'soft_gallery_dark':
        return SOFT_GALLERY_DARK;
      case 'editorial_glow_light':
        return EDITORIAL_GLOW_LIGHT;
      case 'editorial_glow_dark':
        return EDITORIAL_GLOW_DARK;
      case 'default_clean_light':
      default:
        return DEFAULT_CLEAN_LIGHT;
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

  static syncSystemSurface(): void {
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
    switch (status) {
      case 'success':
        return isDark ? '#1B352A' : '#EAF6ED';
      case 'warning':
        return isDark ? '#403019' : '#FFF4DF';
      case 'danger':
        return isDark ? '#40242A' : '#FDEAEC';
      case 'info':
        return isDark ? '#223241' : '#EAF0F7';
      case 'neutral':
      default:
        return palette.surfaceSubtle;
    }
  }

  static getBrandAccent(): string {
    return AppThemeManager.getSafeCurrentPalette().accentBackground;
  }

  static getBrandAccentText(): string {
    return AppThemeManager.getSafeCurrentPalette().accentText;
  }

  private static refreshResolvedTheme(): void {
    const resolvedThemeId = AppThemeManager.resolveThemeId(
      AppThemeManager.getThemeMode(),
      AppStorage.get<boolean>('isDarkMode') === true
    );
    AppStorage.setOrCreate<string>(KEY_RESOLVED_THEME_ID, resolvedThemeId);
    AppThemeManager.syncSystemSurface();
  }

  private static resolveThemeId(themeMode: ThemeMode, isDarkMode: boolean): ResolvedThemeId {
    switch (themeMode) {
      case 'follow_system':
      case 'default_clean':
        return isDarkMode ? 'default_clean_dark' : 'default_clean_light';
      case 'soft_gallery':
        return isDarkMode ? 'soft_gallery_dark' : 'soft_gallery_light';
      case 'editorial_glow':
        return isDarkMode ? 'editorial_glow_dark' : 'editorial_glow_light';
      default:
        return DEFAULT_RESOLVED_THEME_ID;
    }
  }

  private static normalizeThemeMode(themeMode: string): ThemeMode {
    switch (themeMode) {
      case 'default_clean':
      case 'soft_gallery':
      case 'editorial_glow':
      case 'follow_system':
        return themeMode;
      case 'tactile_minimal':
      case 'zen_artistic':
        return 'soft_gallery';
      case 'tactile_editorial':
      case 'profile_signature':
        return 'editorial_glow';
      default:
        return DEFAULT_THEME_MODE;
    }
  }

  private static normalizeResolvedThemeId(resolvedThemeId: string): ResolvedThemeId {
    switch (resolvedThemeId) {
      case 'default_clean_dark':
      case 'soft_gallery_light':
      case 'soft_gallery_dark':
      case 'editorial_glow_light':
      case 'editorial_glow_dark':
      case 'default_clean_light':
        return resolvedThemeId;
      case 'tactile_minimal_light':
      case 'zen_artistic_light':
        return 'soft_gallery_light';
      case 'tactile_minimal_dark':
      case 'zen_artistic_dark':
        return 'soft_gallery_dark';
      case 'tactile_editorial_light':
      case 'profile_signature_light':
        return 'editorial_glow_light';
      case 'tactile_editorial_dark':
      case 'profile_signature_dark':
        return 'editorial_glow_dark';
      default:
        return DEFAULT_RESOLVED_THEME_ID;
    }
  }
}
