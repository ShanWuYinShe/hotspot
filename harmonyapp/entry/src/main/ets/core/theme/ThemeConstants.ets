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

  readonly SOFT_GALLERY_LIGHT: ThemeShadows = {
    surface: new AppThemeShadow(4, '#06000000', 0, 1),
    elevated: new AppThemeShadow(6, '#08000000', 0, 2),
    accent: new AppThemeShadow(0, '#00000000', 0, 0),
    control: new AppThemeShadow(0, '#00000000', 0, 0),
    tabBar: new AppThemeShadow(4, '#06000000', 0, -1)
  };

  readonly SOFT_GALLERY_DARK: ThemeShadows = {
    surface: new AppThemeShadow(4, '#10000000', 0, 1),
    elevated: new AppThemeShadow(8, '#16000000', 0, 2),
    accent: new AppThemeShadow(0, '#00000000', 0, 0),
    control: new AppThemeShadow(0, '#00000000', 0, 0),
    tabBar: new AppThemeShadow(6, '#12000000', 0, -1)
  };

  readonly EDITORIAL_GLOW_LIGHT: ThemeShadows = {
    surface: new AppThemeShadow(28, '#160B1220', 0, 10),
    elevated: new AppThemeShadow(44, '#220C1726', 0, 18),
    accent: new AppThemeShadow(30, '#2EA88466', 0, 8),
    control: new AppThemeShadow(18, '#140E1722', 0, 6),
    tabBar: new AppThemeShadow(36, '#1E0F1826', 0, 10)
  };

  readonly EDITORIAL_GLOW_DARK: ThemeShadows = {
    surface: new AppThemeShadow(24, '#50000000', 0, 8),
    elevated: new AppThemeShadow(42, '#72000000', 0, 18),
    accent: new AppThemeShadow(34, '#40F5CDA3', 0, 10),
    control: new AppThemeShadow(18, '#42000000', 0, 8),
    tabBar: new AppThemeShadow(32, '#66000000', 0, 10)
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

  static readonly SOFT_GALLERY_LIGHT_COLORS: ColorConfig = {
    pageBackground: '#F8F5F2',
    pageGlow: '#FBF7F3',
    heroGlow: '#FBF5EF',
    textureTint: '#FFFDFB14',
    surfaceBackground: '#FFFFFF',
    surfaceSubtle: '#F0EBE3',
    surfaceElevated: '#FFFFFF',
    borderColor: '#E8E0D4',
    floatingBorderColor: '#F0E8DC',
    sheetBackground: '#F8F5F2',
    accentBackground: '#E8B89C',
    accentText: '#FFFFFF',
    accentMuted: '#EDE5DA',
    chatBackground: '#F5F0EA',
    systemSurfaceBaseColor: '#F8F5F2',
    overlayStrong: '#80000000',
    textPrimary: '#333333',
    textSecondary: '#666666',
    textTertiary: '#999999',
    textOnAccentMuted: '#5A4A3E',
    iconMuted: '#999999',
    buttonSecondaryBackground: '#FFFFFF',
    buttonSecondaryText: '#333333',
    tabBarBackground: '#F8F5F2',
    tabBarInactive: '#B0B0B0',
    tabBarActiveBackground: '#F5F0EB',
    isDarkTheme: false
  };

  static readonly SOFT_GALLERY_DARK_COLORS: ColorConfig = {
    pageBackground: '#121212',
    pageGlow: '#1A1A1A',
    heroGlow: '#1E1E1E',
    textureTint: '#FFFFFF08',
    surfaceBackground: '#1E1E1E',
    surfaceSubtle: '#2A2A2A',
    surfaceElevated: '#23262B',
    borderColor: '#333333',
    floatingBorderColor: '#3A3A3A',
    sheetBackground: '#1A1A1A',
    accentBackground: '#FFA500',
    accentText: '#121212',
    accentMuted: '#FFA500',
    chatBackground: '#0E0E0E',
    systemSurfaceBaseColor: '#121212',
    overlayStrong: '#99000000',
    textPrimary: '#FFFFFF',
    textSecondary: '#B0B0B0',
    textTertiary: '#808080',
    textOnAccentMuted: '#121212',
    iconMuted: '#808080',
    buttonSecondaryBackground: '#2A2A2A',
    buttonSecondaryText: '#FFFFFF',
    tabBarBackground: '#121212',
    tabBarInactive: '#808080',
    tabBarActiveBackground: '#1E2222',
    isDarkTheme: true
  };

  static readonly EDITORIAL_GLOW_LIGHT_COLORS: ColorConfig = {
    pageBackground: '#EEE5DA',
    pageGlow: '#FFF5EA',
    heroGlow: '#FFF6ED',
    textureTint: '#FFFFFF26',
    surfaceBackground: '#F7EFE6',
    surfaceSubtle: '#EFE5D8',
    surfaceElevated: '#FBF7F1',
    borderColor: '#F5ECDD',
    floatingBorderColor: '#FFF8EF',
    sheetBackground: '#EFE6DA',
    accentBackground: '#8E6544',
    accentText: '#FFF7F0',
    accentMuted: '#E7D8C7',
    chatBackground: '#F4ECE2',
    systemSurfaceBaseColor: '#EEE5DA',
    overlayStrong: '#66362112',
    textPrimary: '#855E41',
    textSecondary: '#947760',
    textTertiary: '#B7A28E',
    textOnAccentMuted: '#7D604B',
    iconMuted: '#98A28E',
    buttonSecondaryBackground: '#F8F2EA',
    buttonSecondaryText: '#7C6F63',
    tabBarBackground: '#EEE5DA',
    tabBarInactive: '#8C977E',
    tabBarActiveBackground: '#F8F3EA',
    isDarkTheme: false
  };

  static readonly EDITORIAL_GLOW_DARK_COLORS: ColorConfig = {
    pageBackground: '#141C27',
    pageGlow: '#182336',
    heroGlow: '#1C2940',
    textureTint: '#FFFFFF0D',
    surfaceBackground: '#1A2331',
    surfaceSubtle: '#202B3C',
    surfaceElevated: '#202B3C',
    borderColor: '#243246',
    floatingBorderColor: '#2A3950',
    sheetBackground: '#141C27',
    accentBackground: '#F1D0B0',
    accentText: '#1A2230',
    accentMuted: '#2E3B4F',
    chatBackground: '#111825',
    systemSurfaceBaseColor: '#141C27',
    overlayStrong: '#99242E3F',
    textPrimary: '#D8DDE5',
    textSecondary: '#A8B0BC',
    textTertiary: '#7E8794',
    textOnAccentMuted: '#F4D7BE',
    iconMuted: '#C8D0DB',
    buttonSecondaryBackground: '#1C2636',
    buttonSecondaryText: '#D7DDE8',
    tabBarBackground: '#141C27',
    tabBarInactive: '#BEC6D1',
    tabBarActiveBackground: '#202C3F',
    isDarkTheme: true
  };

  static readonly SHADOWS: ThemeShadowRegistry = new ThemeShadowRegistry();
}
