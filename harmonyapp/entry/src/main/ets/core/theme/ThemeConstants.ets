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

  static readonly SHADOWS: Record<string, ThemeShadows> = {
    DEFAULT_CLEAN_LIGHT: {
      surface: { radius: 6, color: '#00000000', offsetX: 0, offsetY: 0 },
      elevated: { radius: 8, color: '#14000000', offsetX: 0, offsetY: 3 },
      accent: { radius: 0, color: '#00000000', offsetX: 0, offsetY: 0 },
      control: { radius: 0, color: '#00000000', offsetX: 0, offsetY: 0 },
      tabBar: { radius: 6, color: '#12000000', offsetX: 0, offsetY: -1 }
    },
    DEFAULT_CLEAN_DARK: {
      surface: { radius: 8, color: '#26000000', offsetX: 0, offsetY: 3 },
      elevated: { radius: 10, color: '#2A000000', offsetX: 0, offsetY: 4 },
      accent: { radius: 0, color: '#00000000', offsetX: 0, offsetY: 0 },
      control: { radius: 0, color: '#00000000', offsetX: 0, offsetY: 0 },
      tabBar: { radius: 10, color: '#26000000', offsetX: 0, offsetY: -1 }
    },
    SOFT_GALLERY_LIGHT: {
      surface: { radius: 28, color: '#24D3C1A220', offsetX: 0, offsetY: 5 },
      elevated: { radius: 34, color: '#2CB89D742C', offsetX: 0, offsetY: 10 },
      accent: { radius: 24, color: '#34E2C39A', offsetX: 0, offsetY: 8 },
      control: { radius: 20, color: '#28CCB7A0', offsetX: 0, offsetY: 6 },
      tabBar: { radius: 28, color: '#2AD6C2A024', offsetX: 0, offsetY: -2 }
    },
    SOFT_GALLERY_DARK: {
      surface: { radius: 24, color: '#36577EBA', offsetX: 0, offsetY: 4 },
      elevated: { radius: 32, color: '#3B6288CE', offsetX: 0, offsetY: 10 },
      accent: { radius: 24, color: '#34E8CEA7', offsetX: 0, offsetY: 8 },
      control: { radius: 20, color: '#345F80B2', offsetX: 0, offsetY: 6 },
      tabBar: { radius: 28, color: '#386489CC', offsetX: 0, offsetY: -2 }
    }
  };
}
