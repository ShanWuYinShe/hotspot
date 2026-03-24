import { StorageKeys } from '../../constants/AppConstants';
import { PreferencesUtil } from '../storage/PreferencesUtil';
import { SystemSurfaceStyle, SystemUiThemeStore } from '../utils/SystemUiThemeStore';
import {
  AppThemePalette,
  DEFAULT_RESOLVED_THEME_ID,
  DEFAULT_THEME_MODE,
  KEY_RESOLVED_THEME_ID,
  KEY_THEME_MODE,
  ResolvedThemeId,
  ThemeMode,
  ThemeOptionItem
} from './AppThemeTypes';
import { ThemeConstants } from './ThemeConstants';

const DEFAULT_CLEAN_LIGHT = {
  id: 'default_clean_light' as ResolvedThemeId,
  label: '当前默认',
  description: '清爽、干净、通用',
  pageBackground: ThemeConstants.DEFAULT_CLEAN_LIGHT_COLORS.pageBackground,
  surfaceBackground: ThemeConstants.DEFAULT_CLEAN_LIGHT_COLORS.surfaceBackground,
  surfaceSubtle: ThemeConstants.DEFAULT_CLEAN_LIGHT_COLORS.surfaceSubtle,
  borderColor: ThemeConstants.DEFAULT_CLEAN_LIGHT_COLORS.borderColor,
  sheetBackground: ThemeConstants.DEFAULT_CLEAN_LIGHT_COLORS.sheetBackground,
  accentBackground: ThemeConstants.DEFAULT_CLEAN_LIGHT_COLORS.accentBackground,
  accentText: ThemeConstants.DEFAULT_CLEAN_LIGHT_COLORS.accentText,
  chatBackground: ThemeConstants.DEFAULT_CLEAN_LIGHT_COLORS.chatBackground,
  systemSurfaceBaseColor: ThemeConstants.DEFAULT_CLEAN_LIGHT_COLORS.systemSurfaceBaseColor,
  preferDarkContent: true,
  isDarkTheme: ThemeConstants.DEFAULT_CLEAN_LIGHT_COLORS.isDarkTheme,
  textPrimary: ThemeConstants.DEFAULT_CLEAN_LIGHT_COLORS.textPrimary,
  textSecondary: ThemeConstants.DEFAULT_CLEAN_LIGHT_COLORS.textSecondary,
  textTertiary: ThemeConstants.DEFAULT_CLEAN_LIGHT_COLORS.textTertiary,
  iconMuted: ThemeConstants.DEFAULT_CLEAN_LIGHT_COLORS.iconMuted,
  buttonSecondaryBackground: ThemeConstants.DEFAULT_CLEAN_LIGHT_COLORS.buttonSecondaryBackground,
  buttonSecondaryText: ThemeConstants.DEFAULT_CLEAN_LIGHT_COLORS.buttonSecondaryText,
  tabBarBackground: ThemeConstants.DEFAULT_CLEAN_LIGHT_COLORS.tabBarBackground,
  tabBarInactive: ThemeConstants.DEFAULT_CLEAN_LIGHT_COLORS.tabBarInactive,
  surfaceShadow: ThemeConstants.SHADOWS.DEFAULT_CLEAN_LIGHT.surface,
  elevatedShadow: ThemeConstants.SHADOWS.DEFAULT_CLEAN_LIGHT.elevated,
  accentShadow: ThemeConstants.SHADOWS.DEFAULT_CLEAN_LIGHT.accent,
  controlShadow: ThemeConstants.SHADOWS.DEFAULT_CLEAN_LIGHT.control,
  tabBarShadow: ThemeConstants.SHADOWS.DEFAULT_CLEAN_LIGHT.tabBar,
  overlayStrong: ThemeConstants.DEFAULT_CLEAN_LIGHT_COLORS.overlayStrong
} as AppThemePalette;

const DEFAULT_CLEAN_DARK = {
  id: 'default_clean_dark' as ResolvedThemeId,
  label: '当前默认',
  description: '清爽、干净、通用',
  pageBackground: ThemeConstants.DEFAULT_CLEAN_DARK_COLORS.pageBackground,
  surfaceBackground: ThemeConstants.DEFAULT_CLEAN_DARK_COLORS.surfaceBackground,
  surfaceSubtle: ThemeConstants.DEFAULT_CLEAN_DARK_COLORS.surfaceSubtle,
  borderColor: ThemeConstants.DEFAULT_CLEAN_DARK_COLORS.borderColor,
  sheetBackground: ThemeConstants.DEFAULT_CLEAN_DARK_COLORS.sheetBackground,
  accentBackground: ThemeConstants.DEFAULT_CLEAN_DARK_COLORS.accentBackground,
  accentText: ThemeConstants.DEFAULT_CLEAN_DARK_COLORS.accentText,
  chatBackground: ThemeConstants.DEFAULT_CLEAN_DARK_COLORS.chatBackground,
  systemSurfaceBaseColor: ThemeConstants.DEFAULT_CLEAN_DARK_COLORS.systemSurfaceBaseColor,
  preferDarkContent: false,
  isDarkTheme: ThemeConstants.DEFAULT_CLEAN_DARK_COLORS.isDarkTheme,
  textPrimary: ThemeConstants.DEFAULT_CLEAN_DARK_COLORS.textPrimary,
  textSecondary: ThemeConstants.DEFAULT_CLEAN_DARK_COLORS.textSecondary,
  textTertiary: ThemeConstants.DEFAULT_CLEAN_DARK_COLORS.textTertiary,
  iconMuted: ThemeConstants.DEFAULT_CLEAN_DARK_COLORS.iconMuted,
  buttonSecondaryBackground: ThemeConstants.DEFAULT_CLEAN_DARK_COLORS.buttonSecondaryBackground,
  buttonSecondaryText: ThemeConstants.DEFAULT_CLEAN_DARK_COLORS.buttonSecondaryText,
  tabBarBackground: ThemeConstants.DEFAULT_CLEAN_DARK_COLORS.tabBarBackground,
  tabBarInactive: ThemeConstants.DEFAULT_CLEAN_DARK_COLORS.tabBarInactive,
  surfaceShadow: ThemeConstants.SHADOWS.DEFAULT_CLEAN_DARK.surface,
  elevatedShadow: ThemeConstants.SHADOWS.DEFAULT_CLEAN_DARK.elevated,
  accentShadow: ThemeConstants.SHADOWS.DEFAULT_CLEAN_DARK.accent,
  controlShadow: ThemeConstants.SHADOWS.DEFAULT_CLEAN_DARK.control,
  tabBarShadow: ThemeConstants.SHADOWS.DEFAULT_CLEAN_DARK.tabBar,
  overlayStrong: ThemeConstants.DEFAULT_CLEAN_DARK_COLORS.overlayStrong
} as AppThemePalette;

const SOFT_GALLERY_LIGHT = {
  id: 'soft_gallery_light' as ResolvedThemeId,
  label: 'Soft Gallery',
  description: '奶白、柔光、静谧质感',
  pageBackground: ThemeConstants.SOFT_GALLERY_LIGHT_COLORS.pageBackground,
  surfaceBackground: ThemeConstants.SOFT_GALLERY_LIGHT_COLORS.surfaceBackground,
  surfaceSubtle: ThemeConstants.SOFT_GALLERY_LIGHT_COLORS.surfaceSubtle,
  borderColor: ThemeConstants.SOFT_GALLERY_LIGHT_COLORS.borderColor,
  sheetBackground: ThemeConstants.SOFT_GALLERY_LIGHT_COLORS.sheetBackground,
  accentBackground: ThemeConstants.SOFT_GALLERY_LIGHT_COLORS.accentBackground,
  accentText: ThemeConstants.SOFT_GALLERY_LIGHT_COLORS.accentText,
  chatBackground: ThemeConstants.SOFT_GALLERY_LIGHT_COLORS.chatBackground,
  systemSurfaceBaseColor: ThemeConstants.SOFT_GALLERY_LIGHT_COLORS.systemSurfaceBaseColor,
  preferDarkContent: true,
  isDarkTheme: ThemeConstants.SOFT_GALLERY_LIGHT_COLORS.isDarkTheme,
  textPrimary: ThemeConstants.SOFT_GALLERY_LIGHT_COLORS.textPrimary,
  textSecondary: ThemeConstants.SOFT_GALLERY_LIGHT_COLORS.textSecondary,
  textTertiary: ThemeConstants.SOFT_GALLERY_LIGHT_COLORS.textTertiary,
  iconMuted: ThemeConstants.SOFT_GALLERY_LIGHT_COLORS.iconMuted,
  buttonSecondaryBackground: ThemeConstants.SOFT_GALLERY_LIGHT_COLORS.buttonSecondaryBackground,
  buttonSecondaryText: ThemeConstants.SOFT_GALLERY_LIGHT_COLORS.buttonSecondaryText,
  tabBarBackground: ThemeConstants.SOFT_GALLERY_LIGHT_COLORS.tabBarBackground,
  tabBarInactive: ThemeConstants.SOFT_GALLERY_LIGHT_COLORS.tabBarInactive,
  surfaceShadow: ThemeConstants.SHADOWS.SOFT_GALLERY_LIGHT.surface,
  elevatedShadow: ThemeConstants.SHADOWS.SOFT_GALLERY_LIGHT.elevated,
  accentShadow: ThemeConstants.SHADOWS.SOFT_GALLERY_LIGHT.accent,
  controlShadow: ThemeConstants.SHADOWS.SOFT_GALLERY_LIGHT.control,
  tabBarShadow: ThemeConstants.SHADOWS.SOFT_GALLERY_LIGHT.tabBar,
  overlayStrong: ThemeConstants.SOFT_GALLERY_LIGHT_COLORS.overlayStrong
} as AppThemePalette;

const SOFT_GALLERY_DARK = {
  id: 'soft_gallery_dark' as ResolvedThemeId,
  label: 'Soft Gallery',
  description: '深蓝、冷光、暖金点缀',
  pageBackground: ThemeConstants.SOFT_GALLERY_DARK_COLORS.pageBackground,
  surfaceBackground: ThemeConstants.SOFT_GALLERY_DARK_COLORS.surfaceBackground,
  surfaceSubtle: ThemeConstants.SOFT_GALLERY_DARK_COLORS.surfaceSubtle,
  borderColor: ThemeConstants.SOFT_GALLERY_DARK_COLORS.borderColor,
  sheetBackground: ThemeConstants.SOFT_GALLERY_DARK_COLORS.sheetBackground,
  accentBackground: ThemeConstants.SOFT_GALLERY_DARK_COLORS.accentBackground,
  accentText: ThemeConstants.SOFT_GALLERY_DARK_COLORS.accentText,
  chatBackground: ThemeConstants.SOFT_GALLERY_DARK_COLORS.chatBackground,
  systemSurfaceBaseColor: ThemeConstants.SOFT_GALLERY_DARK_COLORS.systemSurfaceBaseColor,
  preferDarkContent: false,
  isDarkTheme: ThemeConstants.SOFT_GALLERY_DARK_COLORS.isDarkTheme,
  textPrimary: ThemeConstants.SOFT_GALLERY_DARK_COLORS.textPrimary,
  textSecondary: ThemeConstants.SOFT_GALLERY_DARK_COLORS.textSecondary,
  textTertiary: ThemeConstants.SOFT_GALLERY_DARK_COLORS.textTertiary,
  iconMuted: ThemeConstants.SOFT_GALLERY_DARK_COLORS.iconMuted,
  buttonSecondaryBackground: ThemeConstants.SOFT_GALLERY_DARK_COLORS.buttonSecondaryBackground,
  buttonSecondaryText: ThemeConstants.SOFT_GALLERY_DARK_COLORS.buttonSecondaryText,
  tabBarBackground: ThemeConstants.SOFT_GALLERY_DARK_COLORS.tabBarBackground,
  tabBarInactive: ThemeConstants.SOFT_GALLERY_DARK_COLORS.tabBarInactive,
  surfaceShadow: ThemeConstants.SHADOWS.SOFT_GALLERY_DARK.surface,
  elevatedShadow: ThemeConstants.SHADOWS.SOFT_GALLERY_DARK.elevated,
  accentShadow: ThemeConstants.SHADOWS.SOFT_GALLERY_DARK.accent,
  controlShadow: ThemeConstants.SHADOWS.SOFT_GALLERY_DARK.control,
  tabBarShadow: ThemeConstants.SHADOWS.SOFT_GALLERY_DARK.tabBar,
  overlayStrong: ThemeConstants.SOFT_GALLERY_DARK_COLORS.overlayStrong
} as AppThemePalette;

const THEME_OPTIONS: ThemeOptionItem[] = [
  { mode: 'follow_system', label: '跟随系统', description: '自动切换深浅主题' } as ThemeOptionItem,
  { mode: 'default_clean', label: '当前默认', description: '清爽、干净、通用' } as ThemeOptionItem,
  { mode: 'soft_gallery', label: 'Soft Gallery', description: '柔光、奶油感、软拟物' } as ThemeOptionItem
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
    return AppThemeManager.getPaletteById(AppThemeManager.getResolvedThemeId());
  }

  static getSafeCurrentPalette(): AppThemePalette {
    try {
      return AppThemeManager.getCurrentPalette();
    } catch (_) {
      return DEFAULT_CLEAN_LIGHT;
    }
  }

  static getPaletteById(resolvedThemeId: string): AppThemePalette {
    switch (AppThemeManager.normalizeResolvedThemeId(resolvedThemeId)) {
      case 'default_clean_dark':
        return DEFAULT_CLEAN_DARK;
      case 'soft_gallery_light':
        return SOFT_GALLERY_LIGHT;
      case 'soft_gallery_dark':
        return SOFT_GALLERY_DARK;
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

  static isSoftGalleryTheme(themeId: string): boolean {
    const normalizedThemeId = AppThemeManager.normalizeResolvedThemeId(themeId);
    return normalizedThemeId === 'soft_gallery_light' || normalizedThemeId === 'soft_gallery_dark';
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
      default:
        return DEFAULT_RESOLVED_THEME_ID;
    }
  }

  private static normalizeThemeMode(themeMode: string): ThemeMode {
    switch (themeMode) {
      case 'default_clean':
      case 'soft_gallery':
      case 'follow_system':
        return themeMode;
      case 'tactile_minimal':
      case 'zen_artistic':
        return 'soft_gallery';
      default:
        return DEFAULT_THEME_MODE;
    }
  }

  private static normalizeResolvedThemeId(resolvedThemeId: string): ResolvedThemeId {
    switch (resolvedThemeId) {
      case 'default_clean_dark':
      case 'soft_gallery_light':
      case 'soft_gallery_dark':
      case 'default_clean_light':
        return resolvedThemeId;
      case 'tactile_minimal_light':
      case 'zen_artistic_light':
        return 'soft_gallery_light';
      case 'tactile_minimal_dark':
      case 'zen_artistic_dark':
        return 'soft_gallery_dark';
      default:
        return DEFAULT_RESOLVED_THEME_ID;
    }
  }
}
