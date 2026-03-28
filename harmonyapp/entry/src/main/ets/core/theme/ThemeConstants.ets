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
    surface: new AppThemeShadow(24, '#1FDBCEBA', 0, 6),
    elevated: new AppThemeShadow(36, '#26B59269', 0, 12),
    accent: new AppThemeShadow(30, '#30C59E74', 0, 10),
    control: new AppThemeShadow(20, '#20D4C0A4', 0, 5),
    tabBar: new AppThemeShadow(30, '#20CBB08C', 0, -4)
  };

  readonly SOFT_GALLERY_DARK: ThemeShadows = {
    surface: new AppThemeShadow(22, '#244A6999', 0, 4),
    elevated: new AppThemeShadow(34, '#325A7DC4', 0, 10),
    accent: new AppThemeShadow(24, '#32D5AE7A', 0, 9),
    control: new AppThemeShadow(20, '#2A5A7CB4', 0, 5),
    tabBar: new AppThemeShadow(32, '#305D80C0', 0, -4)
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
    pageBackground: '#F6EFE6',
    surfaceBackground: '#FCF7F0',
    surfaceSubtle: '#F1E7DA',
    borderColor: '#E8DAC7',
    sheetBackground: '#F8F1E8',
    accentBackground: '#B68960',
    accentText: '#FFF8F1',
    chatBackground: '#F7EFE5',
    systemSurfaceBaseColor: '#F6EFE6',
    overlayStrong: '#785A4138',
    textPrimary: '#76563E',
    textSecondary: '#967B63',
    textTertiary: '#B5A08C',
    iconMuted: '#98A48F',
    buttonSecondaryBackground: '#F5ECE2',
    buttonSecondaryText: '#76563E',
    tabBarBackground: '#F7F0E6',
    tabBarInactive: '#887766',
    isDarkTheme: false
  };

  static readonly SOFT_GALLERY_DARK_COLORS: ColorConfig = {
    pageBackground: '#121A27',
    surfaceBackground: '#192233',
    surfaceSubtle: '#212D41',
    borderColor: '#2E4057',
    sheetBackground: '#18202F',
    accentBackground: '#D9B48A',
    accentText: '#192232',
    chatBackground: '#101725',
    systemSurfaceBaseColor: '#121A27',
    overlayStrong: '#BF05080D',
    textPrimary: '#E9EDF5',
    textSecondary: '#B4BECC',
    textTertiary: '#8998AB',
    iconMuted: '#97A9C0',
    buttonSecondaryBackground: '#202C40',
    buttonSecondaryText: '#E9EDF5',
    tabBarBackground: '#16202F',
    tabBarInactive: '#A4B1C2',
    isDarkTheme: true
  };

  static readonly SHADOWS: ThemeShadowRegistry = new ThemeShadowRegistry();
}
