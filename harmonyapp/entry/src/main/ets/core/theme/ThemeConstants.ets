/**
 * 主题系统常量
 * 统一管理所有主题的颜色值，避免重复定义
 */

import { AppThemeShadow } from './AppThemeTypes';

export interface ColorConfig {
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
  overlayStrong: string;
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
  isDarkTheme: boolean;
}

export interface ThemeShadows {
  surface: AppThemeShadow;
  elevated: AppThemeShadow;
  accent: AppThemeShadow;
  control: AppThemeShadow;
  tabBar: AppThemeShadow;
}

class ThemeShadowRegistry {
  readonly DEFAULT_CLEAN_LIGHT: ThemeShadows = {
    surface: new AppThemeShadow(0, '#00000000', 0, 0),
    elevated: new AppThemeShadow(0, '#00000000', 0, 0),
    accent: new AppThemeShadow(14, '#3D7A5526', 0, 4),
    control: new AppThemeShadow(0, '#00000000', 0, 0),
    tabBar: new AppThemeShadow(0, '#00000000', 0, 0)
  };

  readonly DEFAULT_CLEAN_DARK: ThemeShadows = {
    surface: new AppThemeShadow(0, '#00000000', 0, 0),
    elevated: new AppThemeShadow(0, '#00000000', 0, 0),
    accent: new AppThemeShadow(14, '#52B8782E', 0, 4),
    control: new AppThemeShadow(0, '#00000000', 0, 0),
    tabBar: new AppThemeShadow(0, '#00000000', 0, 0)
  };

  readonly FIGMA_NEU_LIGHT: ThemeShadows = {
    surface: new AppThemeShadow(20, '#BFB7AC55', 8, 8),
    elevated: new AppThemeShadow(28, '#C3BAAE55', 12, 12),
    accent: new AppThemeShadow(22, '#D4845A40', 0, 8),
    control: new AppThemeShadow(12, '#FFFFFFBB', -4, -4),
    tabBar: new AppThemeShadow(18, '#B4AB9E55', 0, -6)
  };

  readonly FIGMA_NEU_DARK: ThemeShadows = {
    surface: new AppThemeShadow(20, '#0C0D1280', 8, 8),
    elevated: new AppThemeShadow(28, '#0B0C1090', 12, 12),
    accent: new AppThemeShadow(22, '#E8A87C3D', 0, 8),
    control: new AppThemeShadow(12, '#282A31BB', -4, -4),
    tabBar: new AppThemeShadow(18, '#0B0C10AA', 0, -6)
  };

  readonly VIBRANT_LIGHT: ThemeShadows = {
    surface: new AppThemeShadow(8, '#635BC81A', 0, 3),
    elevated: new AppThemeShadow(14, '#635BC830', 0, 6),
    accent: new AppThemeShadow(22, '#6366F152', 0, 8),
    control: new AppThemeShadow(4, '#635BC80D', 0, 1),
    tabBar: new AppThemeShadow(8, '#635BC818', 0, -2)
  };

  readonly VIBRANT_DARK: ThemeShadows = {
    surface: new AppThemeShadow(10, '#000000A6', 0, 3),
    elevated: new AppThemeShadow(16, '#00000080', 0, 5),
    accent: new AppThemeShadow(22, '#818CF842', 0, 8),
    control: new AppThemeShadow(4, '#818CF80F', 0, 0),
    tabBar: new AppThemeShadow(10, '#00000080', 0, -3)
  };
}

export class ThemeConstants {
  static readonly DEFAULT_CLEAN_LIGHT_COLORS: ColorConfig = {
    pageBackground: '#EDEDED',
    pageGlow: '#F2F4F2',
    heroGlow: '#E8EDE8',
    textureTint: '#FFFFFF0A',
    surfaceBackground: '#FFFFFF',
    surfaceSubtle: '#F0F1F0',
    surfaceElevated: '#FFFFFF',
    borderColor: '#00000014',
    floatingBorderColor: '#0000000A',
    sheetBackground: '#FFFFFF',
    accentBackground: '#3D7A55',
    accentText: '#FFFFFF',
    accentMuted: '#E8F0E9',
    chatBackground: '#F2F3F2',
    systemSurfaceBaseColor: '#EDEDED',
    overlayStrong: '#80000000',
    textPrimary: '#1A1A1A',
    textSecondary: '#8A9190',
    textTertiary: '#AAB0AF',
    textOnAccentMuted: '#2D5040',
    iconMuted: '#8A9190',
    buttonSecondaryBackground: '#F0F1F0',
    buttonSecondaryText: '#1A1A1A',
    primaryButtonBorderColor: '#3D7A55',
    inputBackgroundColor: '#F0F1F0',
    quickActionBackgroundColor: '#FFFFFF',
    tabBarBackground: '#FFFFFF',
    tabBarInactive: '#8A9190',
    tabBarActiveBackground: '#F0F2F0',
    isDarkTheme: false
  };

  static readonly DEFAULT_CLEAN_DARK_COLORS: ColorConfig = {
    pageBackground: '#111311',
    pageGlow: '#171A17',
    heroGlow: '#1A201A',
    textureTint: '#FFFFFF06',
    surfaceBackground: '#1A1C1A',
    surfaceSubtle: '#242624',
    surfaceElevated: '#1E201E',
    borderColor: '#FFFFFF12',
    floatingBorderColor: '#FFFFFF0D',
    sheetBackground: '#1A1C1A',
    accentBackground: '#52B878',
    accentText: '#FFFFFF',
    accentMuted: '#1A3020',
    chatBackground: '#0E100E',
    systemSurfaceBaseColor: '#111311',
    overlayStrong: '#99000000',
    textPrimary: '#E4EAE5',
    textSecondary: '#8AA096',
    textTertiary: '#5A6B60',
    textOnAccentMuted: '#D0F0D5',
    iconMuted: '#8AA096',
    buttonSecondaryBackground: '#242624',
    buttonSecondaryText: '#E4EAE5',
    primaryButtonBorderColor: '#52B878',
    inputBackgroundColor: '#242624',
    quickActionBackgroundColor: '#111311',
    tabBarBackground: '#1A1C1A',
    tabBarInactive: '#8AA096',
    tabBarActiveBackground: '#242624',
    isDarkTheme: true
  };

  static readonly FIGMA_NEU_LIGHT_COLORS: ColorConfig = {
    pageBackground: '#D0CBC3',
    pageGlow: '#E6DDD2',
    heroGlow: '#F4E9DE',
    textureTint: '#FFFFFF16',
    surfaceBackground: '#EAE5DE',
    surfaceSubtle: '#E1DBD3',
    surfaceElevated: '#F2EEE8',
    borderColor: '#F7F5F2',
    floatingBorderColor: '#DFD9D1',
    sheetBackground: '#EAE5DE',
    accentBackground: '#D4845A',
    accentText: '#FFFFFF',
    accentMuted: '#F3E0D4',
    chatBackground: '#DED8D0',
    systemSurfaceBaseColor: '#D0CBC3',
    overlayStrong: '#6B22140F',
    textPrimary: '#4A4036',
    textSecondary: '#7A7269',
    textTertiary: '#9B9590',
    textOnAccentMuted: '#7F533C',
    iconMuted: '#7A7269',
    buttonSecondaryBackground: '#E6E0D8',
    buttonSecondaryText: '#3D3832',
    primaryButtonBorderColor: '#D4845A',
    inputBackgroundColor: '#EAE5DE',
    quickActionBackgroundColor: '#EAE5DE',
    tabBarBackground: '#EAE5DE',
    tabBarInactive: '#8A8075',
    tabBarActiveBackground: '#F2EEE8',
    isDarkTheme: false
  };

  static readonly FIGMA_NEU_DARK_COLORS: ColorConfig = {
    pageBackground: '#0F1015',
    pageGlow: '#161821',
    heroGlow: '#1B1E28',
    textureTint: '#FFFFFF0A',
    surfaceBackground: '#1A1C23',
    surfaceSubtle: '#232731',
    surfaceElevated: '#20242D',
    borderColor: '#2A2C35',
    floatingBorderColor: '#22252D',
    sheetBackground: '#1A1C23',
    accentBackground: '#E8A87C',
    accentText: '#FFFFFF',
    accentMuted: '#35261E',
    chatBackground: '#14161D',
    systemSurfaceBaseColor: '#0F1015',
    overlayStrong: '#A6000000',
    textPrimary: '#E5E7EB',
    textSecondary: '#8D94A0',
    textTertiary: '#5A5550',
    textOnAccentMuted: '#F3CCB4',
    iconMuted: '#7A7670',
    buttonSecondaryBackground: '#232731',
    buttonSecondaryText: '#E8E4DF',
    primaryButtonBorderColor: '#E8A87C',
    inputBackgroundColor: '#1A1C23',
    quickActionBackgroundColor: '#1A1C23',
    tabBarBackground: '#1A1C23',
    tabBarInactive: '#8D94A0',
    tabBarActiveBackground: '#232731',
    isDarkTheme: true
  };

  static readonly SHADOWS: ThemeShadowRegistry = new ThemeShadowRegistry();

  static readonly VIBRANT_LIGHT_COLORS: ColorConfig = {
    pageBackground: '#EEF2FF',
    pageGlow: '#E8EDFF',
    heroGlow: '#DDE4FF',
    textureTint: '#6366f108',
    surfaceBackground: '#FFFFFF',
    surfaceSubtle: '#F0F0FF',
    surfaceElevated: '#FFFFFF',
    borderColor: '#6366F121',
    floatingBorderColor: '#6366F114',
    sheetBackground: '#FFFFFF',
    accentBackground: '#6366f1',
    accentText: '#FFFFFF',
    accentMuted: '#E0E7FF',
    chatBackground: '#EEF2FF',
    systemSurfaceBaseColor: '#EEF2FF',
    overlayStrong: '#80000000',
    textPrimary: '#1e1b4b',
    textSecondary: '#5b5792',
    textTertiary: '#8B87C4',
    textOnAccentMuted: '#C7D2FE',
    iconMuted: '#5b5792',
    buttonSecondaryBackground: '#EEF2FF',
    buttonSecondaryText: '#1e1b4b',
    primaryButtonBorderColor: '#6366f1',
    inputBackgroundColor: '#F5F3FF',
    quickActionBackgroundColor: '#FFFFFF',
    tabBarBackground: '#FFFFFF',
    tabBarInactive: '#5b5792',
    tabBarActiveBackground: '#EEF2FF',
    isDarkTheme: false
  };

  static readonly VIBRANT_DARK_COLORS: ColorConfig = {
    pageBackground: '#09060F',
    pageGlow: '#1A1530',
    heroGlow: '#221B3D',
    textureTint: '#FFFFFF08',
    surfaceBackground: '#141028',
    surfaceSubtle: '#1E1840',
    surfaceElevated: '#1A1432',
    borderColor: '#818CF821',
    floatingBorderColor: '#818CF814',
    sheetBackground: '#141028',
    accentBackground: '#818cf8',
    accentText: '#FFFFFF',
    accentMuted: '#2D2650',
    chatBackground: '#0E0B20',
    systemSurfaceBaseColor: '#09060F',
    overlayStrong: '#AA000000',
    textPrimary: '#e8e4ff',
    textSecondary: '#7a75b0',
    textTertiary: '#5A5490',
    textOnAccentMuted: '#C7D2FE',
    iconMuted: '#7a75b0',
    buttonSecondaryBackground: '#1E1840',
    buttonSecondaryText: '#e8e4ff',
    primaryButtonBorderColor: '#818cf8',
    inputBackgroundColor: '#1A1432',
    quickActionBackgroundColor: '#141028',
    tabBarBackground: '#141028',
    tabBarInactive: '#7a75b0',
    tabBarActiveBackground: '#1E1840',
    isDarkTheme: true
  };
}
