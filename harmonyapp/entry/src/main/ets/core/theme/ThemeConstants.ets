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
    pageBackground: '#EDECEA',
    pageGlow: '#F2F4F2',
    heroGlow: '#E8EDE8',
    textureTint: '#FFFFFF0A',
    surfaceBackground: '#F9F9F5',
    surfaceSubtle: '#F0F1F0',
    surfaceElevated: '#F9F9F5',
    borderColor: '#3D7A5521',
    floatingBorderColor: '#3D7A5514',
    sheetBackground: '#F9F9F5',
    accentBackground: '#3D7A55',
    accentText: '#FFFFFF',
    accentMuted: '#3D7A551A',
    chatBackground: '#F2F3F2',
    systemSurfaceBaseColor: '#EDEDED',
    overlayStrong: '#80000000',
    textPrimary: '#1B1F1B',
    textSecondary: '#566B5E',
    textTertiary: '#AAB0AF',
    textOnAccentMuted: '#2D5040',
    iconMuted: '#566B5E',
    buttonSecondaryBackground: '#F0F1F0',
    buttonSecondaryText: '#1B1F1B',
    primaryButtonBorderColor: '#3D7A55',
    inputBackgroundColor: '#F0F1F0',
    quickActionBackgroundColor: '#FFFFFF',
    tabBarBackground: '#F9F9F5',
    tabBarInactive: '#566B5E',
    tabBarActiveBackground: '#F0F2F0',
    isDarkTheme: false,
    warning: '#D97706',
    success: '#059669',
    error: '#EF4444',
    info: '#3D7A55'
  };

  static readonly DEFAULT_CLEAN_DARK_COLORS: ColorConfig = {
    pageBackground: '#0E110E',
    pageGlow: '#171A17',
    heroGlow: '#1A201A',
    textureTint: '#FFFFFF06',
    surfaceBackground: '#161A16',
    surfaceSubtle: '#242624',
    surfaceElevated: '#181C18',
    borderColor: '#52B87821',
    floatingBorderColor: '#52B87814',
    sheetBackground: '#161A16',
    accentBackground: '#52B878',
    accentText: '#FFFFFF',
    accentMuted: '#52B8781A',
    chatBackground: '#0E100E',
    systemSurfaceBaseColor: '#111311',
    overlayStrong: '#99000000',
    textPrimary: '#DCE8DE',
    textSecondary: '#7A9080',
    textTertiary: '#5A6B60',
    textOnAccentMuted: '#D0F0D5',
    iconMuted: '#7A9080',
    buttonSecondaryBackground: '#242624',
    buttonSecondaryText: '#DCE8DE',
    primaryButtonBorderColor: '#52B878',
    inputBackgroundColor: '#242624',
    quickActionBackgroundColor: '#111311',
    tabBarBackground: '#161A16',
    tabBarInactive: '#7A9080',
    tabBarActiveBackground: '#242624',
    isDarkTheme: true,
    warning: '#FBBF24',
    success: '#34D399',
    error: '#F87171',
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
    pageBackground: '#EEF2FF',
    pageGlow: '#E8EDFF',
    heroGlow: '#DDE4FF',
    textureTint: '#6366f108',
    surfaceBackground: '#FAFAFE',
    surfaceSubtle: '#F0F0FF',
    surfaceElevated: '#FAFAFE',
    borderColor: '#6366F121',
    floatingBorderColor: '#6366F114',
    sheetBackground: '#FAFAFE',
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
    tabBarBackground: '#FAFAFE',
    tabBarInactive: '#5b5792',
    tabBarActiveBackground: '#EEF2FF',
    isDarkTheme: false,
    warning: '#F59E0B',
    success: '#059669',
    error: '#EF4444',
    info: '#6366F1'
  };

  static readonly VIBRANT_DARK_COLORS: ColorConfig = {
    pageBackground: '#07040F',
    pageGlow: '#1A1530',
    heroGlow: '#221B3D',
    textureTint: '#FFFFFF08',
    surfaceBackground: '#130F26',
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
    textPrimary: '#EAE6FF',
    textSecondary: '#857FBD',
    textTertiary: '#5A5490',
    textOnAccentMuted: '#C7D2FE',
    iconMuted: '#857FBD',
    buttonSecondaryBackground: '#1E1840',
    buttonSecondaryText: '#e8e4ff',
    primaryButtonBorderColor: '#818cf8',
    inputBackgroundColor: '#1A1432',
    quickActionBackgroundColor: '#141028',
    tabBarBackground: '#141028',
    tabBarInactive: '#857FBD',
    tabBarActiveBackground: '#1E1840',
    isDarkTheme: true,
    warning: '#FBBF24',
    success: '#34D399',
    error: '#F87171',
    info: '#818CF8'
  };
}
