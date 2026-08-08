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
  // 语义色(对齐前端每主题族值,随主题变)
  warning: string;
  success: string;
  error: string;
  info: string;
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
    pageBackground: '#F6F7F3',
    pageGlow: '#F2F4F2',
    heroGlow: '#E8EDE8',
    textureTint: '#FFFFFF0A',
    surfaceBackground: '#F6F7F3',
    surfaceSubtle: '#F0F1F0',
    surfaceElevated: '#F6F7F3',
    borderColor: '#2A49343D',
    floatingBorderColor: '#2A493414',
    sheetBackground: '#F6F7F3',
    accentBackground: '#3D7A55',
    accentText: '#FFFFFF',
    accentMuted: '#3D7A551A',
    chatBackground: '#F2F3F2',
    systemSurfaceBaseColor: '#EDEDED',
    overlayStrong: '#80000000',
    textPrimary: '#152219',
    textSecondary: '#3F5547',
    textTertiary: '#AAB0AF',
    textOnAccentMuted: '#2D5040',
    iconMuted: '#3F5547',
    buttonSecondaryBackground: '#F0F1F0',
    buttonSecondaryText: '#152219',
    primaryButtonBorderColor: '#3D7A55',
    inputBackgroundColor: '#F0F1F0',
    quickActionBackgroundColor: '#FFFFFF',
    tabBarBackground: '#F9F9F5',
    tabBarInactive: '#3F5547',
    tabBarActiveBackground: '#F0F2F0',
    isDarkTheme: false,
    warning: '#8A6A2F',
    success: '#059669',
    error: '#8E6F35',
    info: '#3D7A55'
  };

  static readonly DEFAULT_CLEAN_DARK_COLORS: ColorConfig = {
    pageBackground: '#161A16',
    pageGlow: '#171A17',
    heroGlow: '#1A201A',
    textureTint: '#FFFFFF06',
    surfaceBackground: '#161A16',
    surfaceSubtle: '#242624',
    surfaceElevated: '#181C18',
    borderColor: '#A9C7B22E',
    floatingBorderColor: '#A9C7B214',
    sheetBackground: '#161A16',
    accentBackground: '#52B878',
    accentText: '#0C1911',
    accentMuted: '#52B8781A',
    chatBackground: '#0E100E',
    systemSurfaceBaseColor: '#111311',
    overlayStrong: '#99000000',
    textPrimary: '#E8F1E9',
    textSecondary: '#A8BDAE',
    textTertiary: '#5A6B60',
    textOnAccentMuted: '#D0F0D5',
    iconMuted: '#A8BDAE',
    buttonSecondaryBackground: '#242624',
    buttonSecondaryText: '#E8F1E9',
    primaryButtonBorderColor: '#52B878',
    inputBackgroundColor: '#242624',
    quickActionBackgroundColor: '#111311',
    tabBarBackground: '#161A16',
    tabBarInactive: '#A8BDAE',
    tabBarActiveBackground: '#242624',
    isDarkTheme: true,
    warning: '#C4A75A',
    success: '#34D399',
    error: '#C4A75A',
    info: '#52B878'
  };

  static readonly FIGMA_NEU_LIGHT_COLORS: ColorConfig = {
    pageBackground: '#E2D6C5',
    pageGlow: '#EFE7DA',
    heroGlow: '#F4E9DE',
    textureTint: '#FFFFFF16',
    surfaceBackground: '#EFE7DA',
    surfaceSubtle: '#E2D6C5',
    surfaceElevated: '#F4E9DE',
    borderColor: '#5D483417',
    floatingBorderColor: '#5D48342E',
    sheetBackground: '#EFE7DA',
    accentBackground: '#B86F4B',
    accentText: '#FFF8F4',
    accentMuted: '#B86F4B1A',
    chatBackground: '#E2D6C5',
    systemSurfaceBaseColor: '#E2D6C5',
    overlayStrong: '#2B211975',
    textPrimary: '#3F342A',
    textSecondary: '#65594A',
    textTertiary: '#8C7F6E',
    textOnAccentMuted: '#7F533C',
    iconMuted: '#65594A',
    buttonSecondaryBackground: '#E2D6C5',
    buttonSecondaryText: '#3F342A',
    primaryButtonBorderColor: '#B86F4B',
    inputBackgroundColor: '#E2D6C5',
    quickActionBackgroundColor: '#EFE7DA',
    tabBarBackground: '#EFE7DA',
    tabBarInactive: '#65594A',
    tabBarActiveBackground: '#F4E9DE',
    isDarkTheme: false,
    warning: '#B8843F',
    success: '#66835D',
    error: '#C0392B',
    info: '#66835D'
  };

  static readonly FIGMA_NEU_DARK_COLORS: ColorConfig = {
    pageBackground: '#15110E',
    pageGlow: '#211B17',
    heroGlow: '#2A2017',
    textureTint: '#FFFFFF0A',
    surfaceBackground: '#211B17',
    surfaceSubtle: '#2A231C',
    surfaceElevated: '#26201A',
    borderColor: '#FFDCB714',
    floatingBorderColor: '#FFDCB729',
    sheetBackground: '#211B17',
    accentBackground: '#D18A5D',
    accentText: '#FFF8F4',
    accentMuted: '#D18A5D1F',
    chatBackground: '#1A1511',
    systemSurfaceBaseColor: '#15110E',
    overlayStrong: '#0000008F',
    textPrimary: '#F1E4D2',
    textSecondary: '#A99682',
    textTertiary: '#8C7F6E',
    textOnAccentMuted: '#F3CCB4',
    iconMuted: '#A99682',
    buttonSecondaryBackground: '#2A231C',
    buttonSecondaryText: '#F1E4D2',
    primaryButtonBorderColor: '#D18A5D',
    inputBackgroundColor: '#2A231C',
    quickActionBackgroundColor: '#1A1511',
    tabBarBackground: '#211B17',
    tabBarInactive: '#A99682',
    tabBarActiveBackground: '#2A231C',
    isDarkTheme: true,
    warning: '#D1A35D',
    success: '#8CA877',
    error: '#E06F6F',
    info: '#8CA877'
  };

  static readonly SHADOWS: ThemeShadowRegistry = new ThemeShadowRegistry();

  static readonly VIBRANT_LIGHT_COLORS: ColorConfig = {
    pageBackground: '#F7FAFF',
    pageGlow: '#E8EDFF',
    heroGlow: '#DDE4FF',
    textureTint: '#2E66C008',
    surfaceBackground: '#F7FAFF',
    surfaceSubtle: '#EAF1FA',
    surfaceElevated: '#F7FAFF',
    borderColor: '#2E66C02E',
    floatingBorderColor: '#2E66C014',
    sheetBackground: '#F7FAFF',
    accentBackground: '#2E66C0',
    accentText: '#FFFFFF',
    accentMuted: '#E2EAF8',
    chatBackground: '#EDF3FC',
    systemSurfaceBaseColor: '#EDF3FC',
    overlayStrong: '#80000000',
    textPrimary: '#16233A',
    textSecondary: '#52647F',
    textTertiary: '#8A97AD',
    textOnAccentMuted: '#C9DCFF',
    iconMuted: '#52647F',
    buttonSecondaryBackground: '#EDF3FC',
    buttonSecondaryText: '#16233A',
    primaryButtonBorderColor: '#2E66C0',
    inputBackgroundColor: '#EAF1FA',
    quickActionBackgroundColor: '#FFFFFF',
    tabBarBackground: '#F7FAFF',
    tabBarInactive: '#52647F',
    tabBarActiveBackground: '#EDF3FC',
    isDarkTheme: false,
    warning: '#F59E0B',
    success: '#14B8A6',
    error: '#F43F5E',
    info: '#2E66C0'
  };

  static readonly VIBRANT_DARK_COLORS: ColorConfig = {
    pageBackground: '#0B1424',
    pageGlow: '#122036',
    heroGlow: '#16263E',
    textureTint: '#FFFFFF08',
    surfaceBackground: '#0B1424',
    surfaceSubtle: '#14243A',
    surfaceElevated: '#16273C',
    borderColor: '#88B7FF2E',
    floatingBorderColor: '#88B7FF14',
    sheetBackground: '#0B1424',
    accentBackground: '#69A8FF',
    accentText: '#071426',
    accentMuted: '#2D2650',
    chatBackground: '#0D1726',
    systemSurfaceBaseColor: '#0B1424',
    overlayStrong: '#AA000000',
    textPrimary: '#E8F1FF',
    textSecondary: '#A5B6CE',
    textTertiary: '#5E6C85',
    textOnAccentMuted: '#B9CFFF',
    iconMuted: '#A5B6CE',
    buttonSecondaryBackground: '#14243A',
    buttonSecondaryText: '#E8F1FF',
    primaryButtonBorderColor: '#69A8FF',
    inputBackgroundColor: '#14243A',
    quickActionBackgroundColor: '#0B1424',
    tabBarBackground: '#0B1424',
    tabBarInactive: '#A5B6CE',
    tabBarActiveBackground: '#14243A',
    isDarkTheme: true,
    warning: '#FBBF24',
    success: '#37C99A',
    error: '#FB7185',
    info: '#69A8FF'
  };
}
