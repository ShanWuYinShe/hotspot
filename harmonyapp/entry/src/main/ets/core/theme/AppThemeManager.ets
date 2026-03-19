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
  surfaceShadow: ThemeConstants.SHADOWS.DEFAULT_CLEAN_DARK.surface,
  elevatedShadow: ThemeConstants.SHADOWS.DEFAULT_CLEAN_DARK.elevated,
  accentShadow: ThemeConstants.SHADOWS.DEFAULT_CLEAN_DARK.accent,
  controlShadow: ThemeConstants.SHADOWS.DEFAULT_CLEAN_DARK.control,
  tabBarShadow: ThemeConstants.SHADOWS.DEFAULT_CLEAN_DARK.tabBar,
  overlayStrong: ThemeConstants.DEFAULT_CLEAN_DARK_COLORS.overlayStrong
} as AppThemePalette;

const TACTILE_MINIMAL_LIGHT = {
  id: 'tactile_minimal_light' as ResolvedThemeId,
  label: 'Tactile Minimalist',
  description: '奶白、柔和、带轻触感',
  pageBackground: ThemeConstants.TACTILE_MINIMAL_LIGHT_COLORS.pageBackground,
  surfaceBackground: ThemeConstants.TACTILE_MINIMAL_LIGHT_COLORS.surfaceBackground,
  surfaceSubtle: ThemeConstants.TACTILE_MINIMAL_LIGHT_COLORS.surfaceSubtle,
  borderColor: ThemeConstants.TACTILE_MINIMAL_LIGHT_COLORS.borderColor,
  sheetBackground: ThemeConstants.TACTILE_MINIMAL_LIGHT_COLORS.sheetBackground,
  accentBackground: ThemeConstants.TACTILE_MINIMAL_LIGHT_COLORS.accentBackground,
  accentText: ThemeConstants.TACTILE_MINIMAL_LIGHT_COLORS.accentText,
  chatBackground: ThemeConstants.TACTILE_MINIMAL_LIGHT_COLORS.chatBackground,
  systemSurfaceBaseColor: ThemeConstants.TACTILE_MINIMAL_LIGHT_COLORS.systemSurfaceBaseColor,
  preferDarkContent: true,
  surfaceShadow: ThemeConstants.SHADOWS.TACTILE_MINIMAL_LIGHT.surface,
  elevatedShadow: ThemeConstants.SHADOWS.TACTILE_MINIMAL_LIGHT.elevated,
  accentShadow: ThemeConstants.SHADOWS.TACTILE_MINIMAL_LIGHT.accent,
  controlShadow: ThemeConstants.SHADOWS.TACTILE_MINIMAL_LIGHT.control,
  tabBarShadow: ThemeConstants.SHADOWS.TACTILE_MINIMAL_LIGHT.tabBar,
  overlayStrong: ThemeConstants.TACTILE_MINIMAL_LIGHT_COLORS.overlayStrong
} as AppThemePalette;

const TACTILE_MINIMAL_DARK = {
  id: 'tactile_minimal_dark' as ResolvedThemeId,
  label: 'Tactile Minimalist',
  description: '深海蓝、暖金、柔和泛光',
  pageBackground: ThemeConstants.TACTILE_MINIMAL_DARK_COLORS.pageBackground,
  surfaceBackground: ThemeConstants.TACTILE_MINIMAL_DARK_COLORS.surfaceBackground,
  surfaceSubtle: ThemeConstants.TACTILE_MINIMAL_DARK_COLORS.surfaceSubtle,
  borderColor: ThemeConstants.TACTILE_MINIMAL_DARK_COLORS.borderColor,
  sheetBackground: ThemeConstants.TACTILE_MINIMAL_DARK_COLORS.sheetBackground,
  accentBackground: ThemeConstants.TACTILE_MINIMAL_DARK_COLORS.accentBackground,
  accentText: ThemeConstants.TACTILE_MINIMAL_DARK_COLORS.accentText,
  chatBackground: ThemeConstants.TACTILE_MINIMAL_DARK_COLORS.chatBackground,
  systemSurfaceBaseColor: ThemeConstants.TACTILE_MINIMAL_DARK_COLORS.systemSurfaceBaseColor,
  preferDarkContent: false,
  surfaceShadow: ThemeConstants.SHADOWS.TACTILE_MINIMAL_DARK.surface,
  elevatedShadow: ThemeConstants.SHADOWS.TACTILE_MINIMAL_DARK.elevated,
  accentShadow: ThemeConstants.SHADOWS.TACTILE_MINIMAL_DARK.accent,
  controlShadow: ThemeConstants.SHADOWS.TACTILE_MINIMAL_DARK.control,
  tabBarShadow: ThemeConstants.SHADOWS.TACTILE_MINIMAL_DARK.tabBar,
  overlayStrong: ThemeConstants.TACTILE_MINIMAL_DARK_COLORS.overlayStrong
} as AppThemePalette;

const ZEN_ARTISTIC_LIGHT = {
  id: 'zen_artistic_light' as ResolvedThemeId,
  label: 'Zen Artistic',
  description: '雾蓝、安静、艺术感',
  pageBackground: ThemeConstants.ZEN_ARTISTIC_LIGHT_COLORS.pageBackground,
  surfaceBackground: ThemeConstants.ZEN_ARTISTIC_LIGHT_COLORS.surfaceBackground,
  surfaceSubtle: ThemeConstants.ZEN_ARTISTIC_LIGHT_COLORS.surfaceSubtle,
  borderColor: ThemeConstants.ZEN_ARTISTIC_LIGHT_COLORS.borderColor,
  sheetBackground: ThemeConstants.ZEN_ARTISTIC_LIGHT_COLORS.sheetBackground,
  accentBackground: ThemeConstants.ZEN_ARTISTIC_LIGHT_COLORS.accentBackground,
  accentText: ThemeConstants.ZEN_ARTISTIC_LIGHT_COLORS.accentText,
  chatBackground: ThemeConstants.ZEN_ARTISTIC_LIGHT_COLORS.chatBackground,
  systemSurfaceBaseColor: ThemeConstants.ZEN_ARTISTIC_LIGHT_COLORS.systemSurfaceBaseColor,
  preferDarkContent: true,
  surfaceShadow: ThemeConstants.SHADOWS.ZEN_ARTISTIC_LIGHT.surface,
  elevatedShadow: ThemeConstants.SHADOWS.ZEN_ARTISTIC_LIGHT.elevated,
  accentShadow: ThemeConstants.SHADOWS.ZEN_ARTISTIC_LIGHT.accent,
  controlShadow: ThemeConstants.SHADOWS.ZEN_ARTISTIC_LIGHT.control,
  tabBarShadow: ThemeConstants.SHADOWS.ZEN_ARTISTIC_LIGHT.tabBar,
  overlayStrong: ThemeConstants.ZEN_ARTISTIC_LIGHT_COLORS.overlayStrong
} as AppThemePalette;

const ZEN_ARTISTIC_DARK = {
  id: 'zen_artistic_dark' as ResolvedThemeId,
  label: 'Zen Artistic',
  description: '深蓝、静谧、艺术感',
  pageBackground: ThemeConstants.ZEN_ARTISTIC_DARK_COLORS.pageBackground,
  surfaceBackground: ThemeConstants.ZEN_ARTISTIC_DARK_COLORS.surfaceBackground,
  surfaceSubtle: ThemeConstants.ZEN_ARTISTIC_DARK_COLORS.surfaceSubtle,
  borderColor: ThemeConstants.ZEN_ARTISTIC_DARK_COLORS.borderColor,
  sheetBackground: ThemeConstants.ZEN_ARTISTIC_DARK_COLORS.sheetBackground,
  accentBackground: ThemeConstants.ZEN_ARTISTIC_DARK_COLORS.accentBackground,
  accentText: ThemeConstants.ZEN_ARTISTIC_DARK_COLORS.accentText,
  chatBackground: ThemeConstants.ZEN_ARTISTIC_DARK_COLORS.chatBackground,
  systemSurfaceBaseColor: ThemeConstants.ZEN_ARTISTIC_DARK_COLORS.systemSurfaceBaseColor,
  preferDarkContent: false,
  surfaceShadow: ThemeConstants.SHADOWS.ZEN_ARTISTIC_DARK.surface,
  elevatedShadow: ThemeConstants.SHADOWS.ZEN_ARTISTIC_DARK.elevated,
  accentShadow: ThemeConstants.SHADOWS.ZEN_ARTISTIC_DARK.accent,
  controlShadow: ThemeConstants.SHADOWS.ZEN_ARTISTIC_DARK.control,
  tabBarShadow: ThemeConstants.SHADOWS.ZEN_ARTISTIC_DARK.tabBar,
  overlayStrong: ThemeConstants.ZEN_ARTISTIC_DARK_COLORS.overlayStrong
} as AppThemePalette;

const THEME_OPTIONS: ThemeOptionItem[] = [
  { mode: 'follow_system', label: '跟随系统', description: '自动切换深浅主题' } as ThemeOptionItem,
  { mode: 'default_clean', label: '当前默认', description: '清爽、干净、通用' } as ThemeOptionItem,
  { mode: 'tactile_minimal', label: 'Tactile Minimalist', description: '奶白、柔和、带轻触感' } as ThemeOptionItem,
  { mode: 'zen_artistic', label: 'Zen Artistic', description: '雾蓝、安静、艺术感' } as ThemeOptionItem
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
      case 'tactile_minimal_light':
        return TACTILE_MINIMAL_LIGHT;
      case 'tactile_minimal_dark':
        return TACTILE_MINIMAL_DARK;
      case 'zen_artistic_light':
        return ZEN_ARTISTIC_LIGHT;
      case 'zen_artistic_dark':
        return ZEN_ARTISTIC_DARK;
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
    switch (status) {
      case 'success':
        return palette.id === 'default_clean_dark' ? '#183428' : '#E7F6EC';
      case 'warning':
        return palette.id === 'default_clean_dark' ? '#3B2B16' : '#FFF4DF';
      case 'danger':
        return palette.id === 'default_clean_dark' ? '#3A1F24' : '#FDEAEC';
      case 'info':
        return palette.id === 'default_clean_dark' ? '#1D2B35' : '#E8F1FB';
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
      case 'tactile_minimal':
        return isDarkMode ? 'tactile_minimal_dark' : 'tactile_minimal_light';
      case 'zen_artistic':
        return isDarkMode ? 'zen_artistic_dark' : 'zen_artistic_light';
      default:
        return DEFAULT_RESOLVED_THEME_ID;
    }
  }

  private static normalizeThemeMode(themeMode: string): ThemeMode {
    switch (themeMode) {
      case 'default_clean':
      case 'tactile_minimal':
      case 'zen_artistic':
      case 'follow_system':
        return themeMode;
      default:
        return DEFAULT_THEME_MODE;
    }
  }

  private static normalizeResolvedThemeId(resolvedThemeId: string): ResolvedThemeId {
    switch (resolvedThemeId) {
      case 'default_clean_dark':
      case 'tactile_minimal_light':
      case 'tactile_minimal_dark':
      case 'zen_artistic_light':
      case 'zen_artistic_dark':
      case 'default_clean_light':
        return resolvedThemeId;
      default:
        return DEFAULT_RESOLVED_THEME_ID;
    }
  }
}