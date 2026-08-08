export type ThemeMode = 'follow_system' | 'default_clean' | 'figma_neu' | 'vibrant';

export type ResolvedThemeId =
  | 'default_clean_light'
  | 'default_clean_dark'
  | 'figma_neu_light'
  | 'figma_neu_dark'
  | 'vibrant_light'
  | 'vibrant_dark';

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

export interface AppThemeShapeTokens {
  surfaceCard: number;
  heroCard: number;
  compactCard: number;
  input: number;
  button: number;
  chip: number;
  tabBar: number;
  previewCard: number;
  listItem: number;
  iconTile: number;
}

export interface AppThemeMetricTokens {
  cardPadding: number;
  heroPadding: number;
  metricCardPadding: number;
  profilePrimaryButtonHeight: number;
  profilePrimaryButtonHorizontalPadding: number;
  secondaryButtonHorizontalPadding: number;
  profileAvatarSize: number;
  profileTitleFontSize: number;
  profileSubtitleFontSize: number;
  profileMetricMinHeight: number;
  profileMetricValueFontSize: number;
  actionButtonHeight: number;
  listItemMinHeight: number;
  tabBarHeight: number;
  createButtonWidth: number;
  createButtonHeight: number;
  tabBarFloatingMargin: number;
  chatListRadius: number;
  chatSendButtonHeight: number;
  previewHeroAvatarSize: number;
  previewPrimaryButtonHeight: number;
  textureOpacity: number;
}

export interface AppThemeEffectTokens {
  avatarShadow: AppThemeShadow;
  haloShadow: AppThemeShadow;
  previewShadow: AppThemeShadow;
  insetShadow: AppThemeShadow;
  highlightShadow: AppThemeShadow;
  tabActiveGlow: AppThemeShadow;
}

export interface AppThemeVisualTokens {
  shape: AppThemeShapeTokens;
  metrics: AppThemeMetricTokens;
  effects: AppThemeEffectTokens;
}

export interface AppThemePalette {
  id: ResolvedThemeId;
  label: string;
  description: string;
  pageBackground: string;
  pageGlow: string;
  heroGlow: string;
  textureTint: string;
  surfaceBackground: string;
  surfaceSubtle: string;
  surfaceElevated: string;
  borderColor: string;
  floatingBorderColor: string;
  sheetBackground: string;
  accentBackground: string;
  accentText: string;
  accentMuted: string;
  chatBackground: string;
  systemSurfaceBaseColor: string;
  preferDarkContent: boolean;
  isDarkTheme: boolean;
  textPrimary: string;
  textSecondary: string;
  textTertiary: string;
  textOnAccentMuted: string;
  iconMuted: string;
  buttonSecondaryBackground: string;
  buttonSecondaryText: string;
  primaryButtonBorderColor: string;
  inputBackgroundColor: string;
  quickActionBackgroundColor: string;
  tabBarBackground: string;
  tabBarInactive: string;
  tabBarActiveBackground: string;
  surfaceShadow: AppThemeShadow;
  elevatedShadow: AppThemeShadow;
  accentShadow: AppThemeShadow;
  controlShadow: AppThemeShadow;
  tabBarShadow: AppThemeShadow;
  overlayStrong: string;
  // 语义色(随主题族/明暗变,对齐前端)
  warning: string;
  success: string;
  error: string;
  info: string;
  visuals: AppThemeVisualTokens;
}

export interface ThemeOptionItem {
  mode: ThemeMode;
  label: string;
  description: string;
}

export const KEY_THEME_MODE: string = 'themeMode';
export const KEY_RESOLVED_THEME_ID: string = 'resolvedThemeId';

// 默认主题对齐前端（前端默认 vibrant 星际；存量用户 AppStorage 已存偏好不受影响）
export const DEFAULT_THEME_MODE: ThemeMode = 'vibrant';
export const DEFAULT_RESOLVED_THEME_ID: ResolvedThemeId = 'vibrant_light';
