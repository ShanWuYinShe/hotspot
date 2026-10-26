export type ThemeMode = 'follow_system' | 'default_clean' | 'vibrant' | 'inkwash';

export type ResolvedThemeId =
  | 'default_clean_light'
  | 'default_clean_dark'
  | 'vibrant_light'
  | 'vibrant_dark'
  | 'inkwash_light'
  | 'inkwash_dark';

/** 明暗偏好三档（与 ThemeMode 主题族正交）：auto 跟随系统，light/dark 手动锁定 */
export type DarkModePreference = 'auto' | 'light' | 'dark';

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
  previewShadow: AppThemeShadow;
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
  surfaceBackground: string;
  /** 软岛底色：surfaceBackground 85% alpha（镜像 web bg-surface-primary/85，r3 内容岛语言） */
  surfaceIsland: string;
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
  /** 在线状态色（对齐前端 --cst-palette-online） */
  online: string;
  preferDarkContent: boolean;
  isDarkTheme: boolean;
  textPrimary: string;
  textSecondary: string;
  textTertiary: string;
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

  controlShadow: AppThemeShadow;
  tabBarShadow: AppThemeShadow;
  overlayStrong: string;
  // 语义色(随主题族/明暗变,对齐前端)
  warning: string;
  success: string;
  error: string;
  info: string;
  /** 地图标记主色（对齐前端 --cst-marker-*，随主题变） */
  mapUser: string;
  mapGroup: string;
  mapMoment: string;
  /** 状态色淡底（对齐前端 --cst-status-*-bg，随主题变） */
  dangerSoft: string;
  warningSoft: string;
  /** R2 整卡图片卡遮罩 / fallback（镜像前端 --cst-surface-card-scrim-* / -fallback） */
  cardScrimTop: string;
  cardScrimTopMid: string;
  cardScrimBottomStrong: string;
  cardScrimBottomMid: string;
  cardScrimBottomSoft: string;
  cardFallbackStart: string;
  cardFallbackMid: string;
  visuals: AppThemeVisualTokens;
}

export interface ThemeOptionItem {
  mode: ThemeMode;
  label: string;
  description: string;
}

export const KEY_THEME_MODE: string = 'themeMode';
export const KEY_RESOLVED_THEME_ID: string = 'resolvedThemeId';
export const KEY_DARK_MODE_PREFERENCE: string = 'darkModePreference';

// 默认主题对齐前端（前端默认 vibrant 星际；存量用户 AppStorage 已存偏好不受影响）
export const DEFAULT_THEME_MODE: ThemeMode = 'vibrant';
export const DEFAULT_RESOLVED_THEME_ID: ResolvedThemeId = 'vibrant_light';
/** 明暗偏好缺省跟随系统 */
export const DEFAULT_DARK_MODE_PREFERENCE: DarkModePreference = 'auto';
