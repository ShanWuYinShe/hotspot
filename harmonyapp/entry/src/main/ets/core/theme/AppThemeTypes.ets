export type ThemeMode = 'follow_system' | 'default_clean' | 'tactile_minimal' | 'zen_artistic';

export type ResolvedThemeId =
  | 'default_clean_light'
  | 'default_clean_dark'
  | 'tactile_minimal_light'
  | 'tactile_minimal_dark'
  | 'zen_artistic_light'
  | 'zen_artistic_dark';

export interface AppThemeShadow {
  radius: number;
  color: string;
  offsetX: number;
  offsetY: number;
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
