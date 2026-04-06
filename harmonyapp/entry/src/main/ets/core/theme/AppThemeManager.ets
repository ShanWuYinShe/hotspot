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

const DEFAULT_THEME_METRICS: AppThemeMetricTokens = {
  cardPadding: DesignConstants.SPACING.LG,
  heroPadding: DesignConstants.SPACING.XL,
  metricCardPadding: DesignConstants.SPACING.LG,
  profilePrimaryButtonHeight: DesignConstants.COMPONENT.BUTTON_HEIGHT,
  profilePrimaryButtonHorizontalPadding: DesignConstants.SPACING.XXL,
  secondaryButtonHorizontalPadding: DesignConstants.SPACING.XL,
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

const NO_SHADOW: AppThemeShadow = new AppThemeShadow(0, '#00000000', 0, 0);

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
    avatarShadow: NO_SHADOW,
    haloShadow: cloneShadow(ThemeConstants.SHADOWS.DEFAULT_CLEAN_LIGHT.control),
    previewShadow: cloneShadow(ThemeConstants.SHADOWS.DEFAULT_CLEAN_LIGHT.elevated),
    insetShadow: NO_SHADOW,
    highlightShadow: NO_SHADOW,
    tabActiveGlow: NO_SHADOW
  }
);

const DEFAULT_CLEAN_DARK_VISUALS = buildVisualTokens(
  DEFAULT_THEME_SHAPE,
  DEFAULT_THEME_METRICS,
  {
    avatarShadow: NO_SHADOW,
    haloShadow: cloneShadow(ThemeConstants.SHADOWS.DEFAULT_CLEAN_DARK.control),
    previewShadow: cloneShadow(ThemeConstants.SHADOWS.DEFAULT_CLEAN_DARK.elevated),
    insetShadow: NO_SHADOW,
    highlightShadow: NO_SHADOW,
    tabActiveGlow: NO_SHADOW
  }
);

const FIGMA_NEU_THEME_SHAPE: AppThemeShapeTokens = {
  surfaceCard: DesignConstants.RADIUS.XXL,
  heroCard: 28,
  compactCard: DesignConstants.RADIUS.XL,
  input: 20,
  button: 22,
  chip: DesignConstants.SHAPE.CHIP,
  tabBar: 30,
  previewCard: 28,
  listItem: 22,
  iconTile: 18
};

const FIGMA_NEU_THEME_METRICS: AppThemeMetricTokens = {
  cardPadding: DesignConstants.SPACING.XL,
  heroPadding: DesignConstants.SPACING.XXL,
  metricCardPadding: DesignConstants.SPACING.LG,
  profilePrimaryButtonHeight: DesignConstants.COMPONENT.BUTTON_HEIGHT,
  profilePrimaryButtonHorizontalPadding: DesignConstants.SPACING.XXL,
  secondaryButtonHorizontalPadding: DesignConstants.SPACING.XXL,
  profileAvatarSize: 96,
  profileTitleFontSize: DesignConstants.FONT.SIZE.XL,
  profileSubtitleFontSize: DesignConstants.FONT.SIZE.LG,
  profileMetricMinHeight: 164,
  profileMetricValueFontSize: 22,
  actionButtonHeight: 40,
  listItemMinHeight: 72,
  tabBarHeight: 64,
  createButtonWidth: 54,
  createButtonHeight: 54,
  tabBarFloatingMargin: DesignConstants.SPACING.MD,
  chatListRadius: DesignConstants.SHAPE.CARD,
  chatSendButtonHeight: DesignConstants.CHAT.SEND_BUTTON_HEIGHT,
  previewHeroAvatarSize: 32,
  previewPrimaryButtonHeight: 18,
  textureOpacity: 0
};

const FIGMA_NEU_LIGHT_VISUALS = buildVisualTokens(
  FIGMA_NEU_THEME_SHAPE,
  FIGMA_NEU_THEME_METRICS,
  {
    avatarShadow: cloneShadow(ThemeConstants.SHADOWS.FIGMA_NEU_LIGHT.surface),
    haloShadow: cloneShadow(ThemeConstants.SHADOWS.FIGMA_NEU_LIGHT.control),
    previewShadow: cloneShadow(ThemeConstants.SHADOWS.FIGMA_NEU_LIGHT.elevated),
    insetShadow: new AppThemeShadow(10, '#FFFFFFC8', -5, -5),
    highlightShadow: cloneShadow(ThemeConstants.SHADOWS.FIGMA_NEU_LIGHT.accent),
    tabActiveGlow: new AppThemeShadow(14, '#D4845A40', 0, 6)
  }
);

const FIGMA_NEU_DARK_VISUALS = buildVisualTokens(
  FIGMA_NEU_THEME_SHAPE,
  FIGMA_NEU_THEME_METRICS,
  {
    avatarShadow: cloneShadow(ThemeConstants.SHADOWS.FIGMA_NEU_DARK.surface),
    haloShadow: cloneShadow(ThemeConstants.SHADOWS.FIGMA_NEU_DARK.control),
    previewShadow: cloneShadow(ThemeConstants.SHADOWS.FIGMA_NEU_DARK.elevated),
    insetShadow: new AppThemeShadow(12, '#2D313B99', -5, -5),
    highlightShadow: cloneShadow(ThemeConstants.SHADOWS.FIGMA_NEU_DARK.accent),
    tabActiveGlow: new AppThemeShadow(16, '#D4845A38', 0, 7)
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

const FIGMA_NEU_LIGHT = buildPalette(
  'figma_neu_light',
  '浮雕暖调',
  '暖灰底、柔和阴影、轻浮雕边框',
  ThemeConstants.FIGMA_NEU_LIGHT_COLORS,
  ThemeConstants.SHADOWS.FIGMA_NEU_LIGHT,
  FIGMA_NEU_LIGHT_VISUALS,
  true
);

const FIGMA_NEU_DARK = buildPalette(
  'figma_neu_dark',
  '浮雕暖调',
  '暗色暖调、柔和立体层次',
  ThemeConstants.FIGMA_NEU_DARK_COLORS,
  ThemeConstants.SHADOWS.FIGMA_NEU_DARK,
  FIGMA_NEU_DARK_VISUALS,
  false
);

const THEME_OPTIONS: ThemeOptionItem[] = [
  { mode: 'follow_system', label: '跟随系统', description: '自动匹配系统明暗模式' } as ThemeOptionItem,
  { mode: 'default_clean', label: '当前默认', description: '清爽、中性、稳妥的基础主题' } as ThemeOptionItem,
  { mode: 'figma_neu', label: '浮雕暖调', description: '柔和压边、暖灰底与立体按钮感' } as ThemeOptionItem
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
      case 'figma_neu_dark':
        return FIGMA_NEU_DARK;
      case 'figma_neu_light':
        return FIGMA_NEU_LIGHT;
      case 'default_clean_dark':
        return DEFAULT_CLEAN_DARK;
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
      case 'figma_neu':
        return isDarkMode ? 'figma_neu_dark' : 'figma_neu_light';
      default:
        return DEFAULT_RESOLVED_THEME_ID;
    }
  }

  private static normalizeThemeMode(themeMode: string): ThemeMode {
    switch (themeMode) {
      case 'default_clean':
      case 'follow_system':
      case 'figma_neu':
        return themeMode;
      default:
        return DEFAULT_THEME_MODE;
    }
  }

  private static normalizeResolvedThemeId(resolvedThemeId: string): ResolvedThemeId {
    switch (resolvedThemeId) {
      case 'default_clean_dark':
      case 'default_clean_light':
      case 'figma_neu_dark':
      case 'figma_neu_light':
        return resolvedThemeId;
      default:
        return DEFAULT_RESOLVED_THEME_ID;
    }
  }
}
