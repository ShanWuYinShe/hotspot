/**
 * 主题系统常量
 * 统一管理所有主题的颜色值，避免重复定义
 */

import { AppThemeShadow } from './AppThemeTypes';

export interface ColorConfig {
  pageBackground: string;
  surfaceBackground: string;
  surfaceSubtle: string;
  borderColor: string;
  sheetBackground: string;
  accentBackground: string;
  accentText: string;
  chatBackground: string;
  systemSurfaceBaseColor: string;
  overlayStrong: string;
  textPrimary: string;
  textSecondary: string;
  textTertiary: string;
  iconMuted: string;
  buttonSecondaryBackground: string;
  buttonSecondaryText: string;
  tabBarBackground: string;
  tabBarInactive: string;
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

  readonly SOFT_GALLERY_LIGHT: ThemeShadows = {
    surface: new AppThemeShadow(16, '#0D000000', 0, 2),
    elevated: new AppThemeShadow(24, '#14000000', 0, 4),
    accent: new AppThemeShadow(20, '#18D4AF37', 0, 3),
    control: new AppThemeShadow(12, '#0A000000', 0, 2),
    tabBar: new AppThemeShadow(20, '#10000000', 0, -2)
  };

  readonly SOFT_GALLERY_DARK: ThemeShadows = {
    surface: new AppThemeShadow(14, '#20000000', 0, 2),
    elevated: new AppThemeShadow(22, '#30000000', 0, 4),
    accent: new AppThemeShadow(16, '#26FFD700', 0, 3),
    control: new AppThemeShadow(12, '#1A000000', 0, 2),
    tabBar: new AppThemeShadow(18, '#26000000', 0, -2)
  };
}

export class ThemeConstants {
  static readonly DEFAULT_CLEAN_LIGHT_COLORS: ColorConfig = {
    pageBackground: '#FFFFFF',
    surfaceBackground: '#FFFFFF',
    surfaceSubtle: '#F3F4F6',
    borderColor: '#E5E7EB',
    sheetBackground: '#FFFFFF',
    accentBackground: '#4A7A57',
    accentText: '#FFFFFF',
    chatBackground: '#F5F6FA',
    systemSurfaceBaseColor: '#FFFFFF',
    overlayStrong: '#80000000',
    textPrimary: '#111827',
    textSecondary: '#6B7280',
    textTertiary: '#9CA3AF',
    iconMuted: '#6B7280',
    buttonSecondaryBackground: '#F3F4F6',
    buttonSecondaryText: '#111827',
    tabBarBackground: '#FFFFFF',
    tabBarInactive: '#6B7280',
    isDarkTheme: false
  };

  static readonly DEFAULT_CLEAN_DARK_COLORS: ColorConfig = {
    pageBackground: '#0F1115',
    surfaceBackground: '#171A20',
    surfaceSubtle: '#222733',
    borderColor: '#2D3544',
    sheetBackground: '#171A20',
    accentBackground: '#6BAF86',
    accentText: '#0F1115',
    chatBackground: '#11131D',
    systemSurfaceBaseColor: '#0F1115',
    overlayStrong: '#99000000',
    textPrimary: '#F9FAFB',
    textSecondary: '#9CA3AF',
    textTertiary: '#6B7280',
    iconMuted: '#9CA3AF',
    buttonSecondaryBackground: '#222733',
    buttonSecondaryText: '#F9FAFB',
    tabBarBackground: '#171A20',
    tabBarInactive: '#9CA3AF',
    isDarkTheme: true
  };

  static readonly SOFT_GALLERY_LIGHT_COLORS: ColorConfig = {
    pageBackground: '#F8F5F2',
    surfaceBackground: '#FFFFFF',
    surfaceSubtle: '#F0EBE3',
    borderColor: '#E8E0D4',
    sheetBackground: '#F8F5F2',
    accentBackground: '#D4AF37',
    accentText: '#FFFFFF',
    chatBackground: '#F5F0EA',
    systemSurfaceBaseColor: '#F8F5F2',
    overlayStrong: '#80000000',
    textPrimary: '#2D2D2D',
    textSecondary: '#6E6E6E',
    textTertiary: '#9E9E9E',
    iconMuted: '#9E9E9E',
    buttonSecondaryBackground: '#FFFFFF',
    buttonSecondaryText: '#2D2D2D',
    tabBarBackground: '#F8F5F2',
    tabBarInactive: '#8A8A8A',
    isDarkTheme: false
  };

  static readonly SOFT_GALLERY_DARK_COLORS: ColorConfig = {
    pageBackground: '#121212',
    surfaceBackground: '#1E1E1E',
    surfaceSubtle: '#2A2A2A',
    borderColor: '#333333',
    sheetBackground: '#1A1A1A',
    accentBackground: '#FFD700',
    accentText: '#121212',
    chatBackground: '#0E0E0E',
    systemSurfaceBaseColor: '#121212',
    overlayStrong: '#99000000',
    textPrimary: '#FFFFFF',
    textSecondary: '#B0B0B0',
    textTertiary: '#808080',
    iconMuted: '#808080',
    buttonSecondaryBackground: '#2A2A2A',
    buttonSecondaryText: '#FFFFFF',
    tabBarBackground: '#181818',
    tabBarInactive: '#8A8A8A',
    isDarkTheme: true
  };

  static readonly SHADOWS: ThemeShadowRegistry = new ThemeShadowRegistry();
}
