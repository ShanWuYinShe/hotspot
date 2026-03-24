export type ThemeMode = 'follow_system' | 'default_clean' | 'soft_gallery';

export type ResolvedThemeId =
  | 'default_clean_light'
  | 'default_clean_dark'
  | 'soft_gallery_light'
  | 'soft_gallery_dark';

export class AppThemeShadow {
  radius: number;
  color: string;
  offsetX: number;
  offsetY: number;

  constructor(radius: number, color: string, offsetX: number, offsetY: number) {
    this.radius = radius;
    this.color = color;
    this.offsetX = offsetX;
    this.offsetY = offsetY;
  }
}

export interface AppThemePalette {
  id: ResolvedThemeId;
  label: string;
  description: string;
  pageBackground: string;
  surfaceBackground: string;
  surfaceSubtle: string;
  borderColor: string;
  sheetBackground: string;
  accentBackground: string;
  accentText: string;
  chatBackground: string;
  systemSurfaceBaseColor: string;
  preferDarkContent: boolean;
  isDarkTheme: boolean;
  textPrimary: string;
  textSecondary: string;
  textTertiary: string;
  iconMuted: string;
  buttonSecondaryBackground: string;
  buttonSecondaryText: string;
  tabBarBackground: string;
  tabBarInactive: string;
  surfaceShadow: AppThemeShadow;
  elevatedShadow: AppThemeShadow;
  accentShadow: AppThemeShadow;
  controlShadow: AppThemeShadow;
  tabBarShadow: AppThemeShadow;
  overlayStrong: string;
}

export interface ThemeOptionItem {
  mode: ThemeMode;
  label: string;
  description: string;
}

export const KEY_THEME_MODE: string = 'themeMode';
export const KEY_RESOLVED_THEME_ID: string = 'resolvedThemeId';

export const DEFAULT_THEME_MODE: ThemeMode = 'follow_system';
export const DEFAULT_RESOLVED_THEME_ID: ResolvedThemeId = 'default_clean_light';
