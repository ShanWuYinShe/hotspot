/**
 * 主题系统常量
 * 统一管理所有主题的颜色值，避免重复定义
 */

import { AppThemeShadow } from './AppThemeTypes';

interface ColorConfig {
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

interface ThemeShadows {
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
    surface: new AppThemeShadow(28, '#24D3C1A2', 0, 5),
    elevated: new AppThemeShadow(34, '#2CB89D74', 0, 10),
    accent: new AppThemeShadow(24, '#34E2C39A', 0, 8),
    control: new AppThemeShadow(20, '#28CCB7A0', 0, 6),
    tabBar: new AppThemeShadow(28, '#2AD6C2A0', 0, -2)
  };

  readonly SOFT_GALLERY_DARK: ThemeShadows = {
    surface: new AppThemeShadow(24, '#36577EBA', 0, 4),
    elevated: new AppThemeShadow(32, '#3B6288CE', 0, 10),
    accent: new AppThemeShadow(24, '#34E8CEA7', 0, 8),
    control: new AppThemeShadow(20, '#345F80B2', 0, 6),
    tabBar: new AppThemeShadow(28, '#386489CC', 0, -2)
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
    pageBackground: '#F7F1E9',
    surfaceBackground: '#FCF8F2',
    surfaceSubtle: '#F0E7DC',
    borderColor: '#E8DDD0',
    sheetBackground: '#F9F3EC',
    accentBackground: '#E9D1AF',
    accentText: '#6D5845',
    chatBackground: '#F5EEE6',
    systemSurfaceBaseColor: '#F7F1E9',
    overlayStrong: '#8A6B553E',
    textPrimary: '#7A5E49',
    textSecondary: '#9A826D',
    textTertiary: '#B7A797',
    iconMuted: '#A7B3A3',
    buttonSecondaryBackground: '#F3EBE1',
    buttonSecondaryText: '#7A5E49',
    tabBarBackground: '#F8F2EA',
    tabBarInactive: '#8C7867',
    isDarkTheme: false
  };

  static readonly SOFT_GALLERY_DARK_COLORS: ColorConfig = {
    pageBackground: '#111A26',
    surfaceBackground: '#182332',
    surfaceSubtle: '#1E2B3B',
    borderColor: '#2A3A4F',
    sheetBackground: '#172230',
    accentBackground: '#E8CEA7',
    accentText: '#182433',
    chatBackground: '#0F1824',
    systemSurfaceBaseColor: '#111A26',
    overlayStrong: '#B3000000',
    textPrimary: '#D9E0E8',
    textSecondary: '#A3AFBB',
    textTertiary: '#7D8A99',
    iconMuted: '#8A9CAF',
    buttonSecondaryBackground: '#223142',
    buttonSecondaryText: '#D9E0E8',
    tabBarBackground: '#16212E',
    tabBarInactive: '#9AA7B5',
    isDarkTheme: true
  };

  static readonly SHADOWS: ThemeShadowRegistry = new ThemeShadowRegistry();
}
