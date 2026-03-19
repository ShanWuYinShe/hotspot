/**
 * 主题系统常量
 * 统一管理所有主题的颜色值，避免重复定义
 */

import { AppThemeShadow } from './AppThemeTypes';

// ==================== 颜色配置接口 ====================
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
}

// ==================== 阴影配置接口 ====================
interface ThemeShadows {
  surface: AppThemeShadow;
  elevated: AppThemeShadow;
  accent: AppThemeShadow;
  control: AppThemeShadow;
  tabBar: AppThemeShadow;
}

export class ThemeConstants {
  // ==================== 默认清爽主题颜色 ====================
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
    overlayStrong: '#80000000'
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
    overlayStrong: '#99000000'
  };

  // ==================== 触感极简主题颜色 ====================
  static readonly TACTILE_MINIMAL_LIGHT_COLORS: ColorConfig = {
    pageBackground: '#E9E1D4',
    surfaceBackground: '#F6EFE4',
    surfaceSubtle: '#DDD1C2',
    borderColor: '#D0C4B4',
    sheetBackground: '#F4EDDF',
    accentBackground: '#E4C899',
    accentText: '#6C5234',
    chatBackground: '#EDE3D6',
    systemSurfaceBaseColor: '#E9E1D4',
    overlayStrong: '#8A6F584D'
  };

  static readonly TACTILE_MINIMAL_DARK_COLORS: ColorConfig = {
    pageBackground: '#09131E',
    surfaceBackground: '#102030',
    surfaceSubtle: '#0C1A29',
    borderColor: '#24384B',
    sheetBackground: '#112334',
    accentBackground: '#E3C9A5',
    accentText: '#182638',
    chatBackground: '#0B1725',
    systemSurfaceBaseColor: '#09131E',
    overlayStrong: '#B0000000'
  };

  // ==================== 禅意艺术主题颜色 ====================
  static readonly ZEN_ARTISTIC_LIGHT_COLORS: ColorConfig = {
    pageBackground: '#C7D2D8',
    surfaceBackground: '#D3DDE2',
    surfaceSubtle: '#B7C4CB',
    borderColor: '#ACBAC2',
    sheetBackground: '#D5E0E4',
    accentBackground: '#E3D5C4',
    accentText: '#705743',
    chatBackground: '#C1CDD3',
    systemSurfaceBaseColor: '#C7D2D8',
    overlayStrong: '#8A7D7166'
  };

  static readonly ZEN_ARTISTIC_DARK_COLORS: ColorConfig = {
    pageBackground: '#0A1A23',
    surfaceBackground: '#132634',
    surfaceSubtle: '#0E202D',
    borderColor: '#2A4A5A',
    sheetBackground: '#142938',
    accentBackground: '#B8A089',
    accentText: '#1A2B38',
    chatBackground: '#0D1C27',
    systemSurfaceBaseColor: '#0A1A23',
    overlayStrong: '#99000000'
  };

  // ==================== 阴影配置 ====================
  static readonly SHADOWS: Record<string, ThemeShadows> = {
    // 默认清爽主题阴影
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
    // 触感极简主题阴影
    TACTILE_MINIMAL_LIGHT: {
      surface: { radius: 22, color: '#1F9A826418', offsetX: 0, offsetY: 0 },
      elevated: { radius: 26, color: '#2C9A82661C', offsetX: 0, offsetY: 6 },
      accent: { radius: 18, color: '#2A9E7D621E', offsetX: 0, offsetY: 6 },
      control: { radius: 12, color: '#1E9E846818', offsetX: 0, offsetY: 4 },
      tabBar: { radius: 18, color: '#24A18E7A1A', offsetX: 0, offsetY: -2 }
    },
    TACTILE_MINIMAL_DARK: {
      surface: { radius: 18, color: '#2A5A84A6', offsetX: 0, offsetY: 0 },
      elevated: { radius: 24, color: '#304F74C0', offsetX: 0, offsetY: 8 },
      accent: { radius: 22, color: '#28E3C9A5', offsetX: 0, offsetY: 7 },
      control: { radius: 14, color: '#264A6B96', offsetX: 0, offsetY: 4 },
      tabBar: { radius: 20, color: '#30516D9E', offsetX: 0, offsetY: -2 }
    },
    // 禅意艺术主题阴影
    ZEN_ARTISTIC_LIGHT: {
      surface: { radius: 20, color: '#1E4F677018', offsetX: 0, offsetY: 0 },
      elevated: { radius: 24, color: '#28566B7A24', offsetX: 0, offsetY: 6 },
      accent: { radius: 16, color: '#245A6D7A22', offsetX: 0, offsetY: 6 },
      control: { radius: 10, color: '#1E4F677018', offsetX: 0, offsetY: 4 },
      tabBar: { radius: 16, color: '#224F67701C', offsetX: 0, offsetY: -2 }
    },
    ZEN_ARTISTIC_DARK: {
      surface: { radius: 20, color: '#2A3A4A60', offsetX: 0, offsetY: 0 },
      elevated: { radius: 24, color: '#30405070', offsetX: 0, offsetY: 6 },
      accent: { radius: 16, color: '#2A3A4A60', offsetX: 0, offsetY: 6 },
      control: { radius: 10, color: '#2A3A4A60', offsetX: 0, offsetY: 4 },
      tabBar: { radius: 16, color: '#2A3A4A60', offsetX: 0, offsetY: -2 }
    }
  };
}