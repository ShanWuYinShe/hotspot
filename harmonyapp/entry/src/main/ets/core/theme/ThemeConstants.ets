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
    surface: new AppThemeShadow(6, '#00000000', 0, 0),
    elevated: new AppThemeShadow(8, '#14000000', 0, 3),
    accent: new AppThemeShadow(0, '#00000000', 0, 0),
    control: new AppThemeShadow(0, '#00000000', 0, 0),
    tabBar: new AppThemeShadow(6, '#12000000', 0, -1)
  };

  readonly DEFAULT_CLEAN_DARK: ThemeShadows = {
    surface: new AppThemeShadow(8, '#26000000', 0, 3),
    elevated: new AppThemeShadow(10, '#2A000000', 0, 4),
    accent: new AppThemeShadow(0, '#00000000', 0, 0),
    control: new AppThemeShadow(0, '#00000000', 0, 0),
    tabBar: new AppThemeShadow(10, '#26000000', 0, -1)
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
}

export class ThemeConstants {
  static readonly DEFAULT_CLEAN_LIGHT_COLORS: ColorConfig = {
    pageBackground: '#FFFFFF',
    pageGlow: '#FFF7EC',
    heroGlow: '#F2F8F2',
    textureTint: '#FFFFFF14',
    surfaceBackground: '#FFFFFF',
    surfaceSubtle: '#F3F4F6',
    surfaceElevated: '#FFFFFF',
    borderColor: '#E5E7EB',
    floatingBorderColor: '#F6F6F6',
    sheetBackground: '#FFFFFF',
    accentBackground: '#4A7A57',
    accentText: '#FFFFFF',
    accentMuted: '#E6F1E8',
    chatBackground: '#F5F6FA',
    systemSurfaceBaseColor: '#FFFFFF',
    overlayStrong: '#80000000',
    textPrimary: '#111827',
    textSecondary: '#6B7280',
    textTertiary: '#9CA3AF',
    textOnAccentMuted: '#2F4A35',
    iconMuted: '#6B7280',
    buttonSecondaryBackground: '#F3F4F6',
    buttonSecondaryText: '#111827',
    primaryButtonBorderColor: '#F6F6F6',
    inputBackgroundColor: '#F3F4F6',
    quickActionBackgroundColor: '#FFFFFF',
    tabBarBackground: '#FFFFFF',
    tabBarInactive: '#6B7280',
    tabBarActiveBackground: '#F6F8F7',
    isDarkTheme: false
  };

  static readonly DEFAULT_CLEAN_DARK_COLORS: ColorConfig = {
    pageBackground: '#0F1115',
    pageGlow: '#161A22',
    heroGlow: '#172023',
    textureTint: '#FFFFFF08',
    surfaceBackground: '#171A20',
    surfaceSubtle: '#222733',
    surfaceElevated: '#1B1F26',
    borderColor: '#2D3544',
    floatingBorderColor: '#263040',
    sheetBackground: '#171A20',
    accentBackground: '#6BAF86',
    accentText: '#0F1115',
    accentMuted: '#1D3426',
    chatBackground: '#11131D',
    systemSurfaceBaseColor: '#0F1115',
    overlayStrong: '#99000000',
    textPrimary: '#F9FAFB',
    textSecondary: '#9CA3AF',
    textTertiary: '#6B7280',
    textOnAccentMuted: '#D9F2E0',
    iconMuted: '#9CA3AF',
    buttonSecondaryBackground: '#222733',
    buttonSecondaryText: '#F9FAFB',
    primaryButtonBorderColor: '#263040',
    inputBackgroundColor: '#222733',
    quickActionBackgroundColor: '#0F1115',
    tabBarBackground: '#0F1115',
    tabBarInactive: '#9CA3AF',
    tabBarActiveBackground: '#1D242D',
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
}
