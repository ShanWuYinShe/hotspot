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
    surface: new AppThemeShadow(26, '#24E3D3BE', 0, 5),
    elevated: new AppThemeShadow(34, '#30B48D63', 0, 10),
    accent: new AppThemeShadow(28, '#34D9B98F', 0, 8),
    control: new AppThemeShadow(18, '#22D6C1A4', 0, 4),
    tabBar: new AppThemeShadow(26, '#24D0B898', 0, -3)
  };

  readonly SOFT_GALLERY_DARK: ThemeShadows = {
    surface: new AppThemeShadow(24, '#384E72AA', 0, 4),
    elevated: new AppThemeShadow(32, '#40597EC7', 0, 10),
    accent: new AppThemeShadow(24, '#42D2A56D', 0, 8),
    control: new AppThemeShadow(18, '#30557BAA', 0, 5),
    tabBar: new AppThemeShadow(30, '#3A6C8FD8', 0, -3)
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
    pageBackground: '#F4EDE3',
    surfaceBackground: '#FAF4EC',
    surfaceSubtle: '#EFE5D8',
    borderColor: '#E5D7C5',
    sheetBackground: '#F7EFE5',
    accentBackground: '#B88C63',
    accentText: '#FFF8F1',
    chatBackground: '#F6EEE4',
    systemSurfaceBaseColor: '#F4EDE3',
    overlayStrong: '#7A5F4638',
    textPrimary: '#7B5B43',
    textSecondary: '#9A8068',
    textTertiary: '#B59F8D',
    iconMuted: '#98A690',
    buttonSecondaryBackground: '#F3EADF',
    buttonSecondaryText: '#7B5B43',
    tabBarBackground: '#F5EDE2',
    tabBarInactive: '#8C7868',
    isDarkTheme: false
  };

  static readonly SOFT_GALLERY_DARK_COLORS: ColorConfig = {
    pageBackground: '#141B28',
    surfaceBackground: '#1B2434',
    surfaceSubtle: '#212D40',
    borderColor: '#314158',
    sheetBackground: '#182130',
    accentBackground: '#D6B38A',
    accentText: '#1A2434',
    chatBackground: '#111826',
    systemSurfaceBaseColor: '#141B28',
    overlayStrong: '#BF05070C',
    textPrimary: '#E7EBF1',
    textSecondary: '#B1BBC8',
    textTertiary: '#8794A7',
    iconMuted: '#95A6BD',
    buttonSecondaryBackground: '#202B3E',
    buttonSecondaryText: '#E7EBF1',
    tabBarBackground: '#17202E',
    tabBarInactive: '#A0ADBE',
    isDarkTheme: true
  };

  static readonly SHADOWS: ThemeShadowRegistry = new ThemeShadowRegistry();
}
