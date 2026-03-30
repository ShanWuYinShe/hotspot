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
    tabBarBackground: '#0F1115',
    tabBarInactive: '#9CA3AF',
    tabBarActiveBackground: '#1D242D',
    isDarkTheme: true
  };

  static readonly SHADOWS: ThemeShadowRegistry = new ThemeShadowRegistry();
}
