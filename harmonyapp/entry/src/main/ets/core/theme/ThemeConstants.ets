/**
 * 主题系统常量
 * 统一管理所有主题的颜色值，避免重复定义
 */

import { AppThemeShadow } from './AppThemeTypes';

export interface ColorConfig {
  pageBackground: string;
  /** 工作区舞台底色（比 page 深一档，对齐前端 --cst-palette-workspace-bg） */
  workspaceBackground: string;
  pageGlow: string;
  heroGlow: string;
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
  /** 在线状态色（对齐前端 --cst-palette-online，随主题变） */
  online: string;
  textPrimary: string;
  textSecondary: string;
  textTertiary: string;
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
  /** 地图标记主色（对齐前端 --cst-marker-user/group/moment，随主题族/明暗变） */
  mapUser: string;
  mapGroup: string;
  mapMoment: string;
  /** 状态色淡底（对齐前端 --cst-status-*-bg 的 color-mix 12~14% 预混近似，随主题变） */
  dangerSoft: string;
  warningSoft: string;
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
    // 卡片轻微抬升（对齐前端 elevation-card 的"浮起"感；ArkUI 无 spread 做不了硬边 ring，用轻量模糊近似）
    surface: new AppThemeShadow(10, '#143D7A55', 0, 2),
    // 弹层抬升（对齐前端 --cst-elevation-modal 0 24px 80px 0.24）
    elevated: new AppThemeShadow(40, '#3D000000', 0, 12),
    // 主操作光晕（对齐前端 --cst-elevation-fab 0 6px 18px glow；forest 扁平退化：近透明）
    accent: new AppThemeShadow(18, '#0A3D7A55', 0, 6),
    // 菜单浮层（对齐前端 --cst-elevation-menu 0 16px 40px 0.16）
    control: new AppThemeShadow(32, '#29000000', 0, 8),
    // 底部导航（对齐前端 --cst-elevation-nav，向上微光）
    tabBar: new AppThemeShadow(24, '#1A3D7A55', 0, -4)
  };

  readonly DEFAULT_CLEAN_DARK: ThemeShadows = {
    // 深色下黑阴影对比弱，卡片抬升改用更亮的描边承载（border 单独加亮），阴影只做轻微加深
    surface: new AppThemeShadow(14, '#4D000000', 0, 4),
    // 弹层抬升（对齐前端 --cst-elevation-modal 0 28px 88px 0.42）
    elevated: new AppThemeShadow(44, '#6B000000', 0, 14),
    // 主操作光晕（对齐前端 --cst-elevation-fab 0 6px 22px glow；深色 + forest 扁平退化：近透明）
    accent: new AppThemeShadow(22, '#0D52B878', 0, 6),
    // 菜单浮层
    control: new AppThemeShadow(40, '#59000000', 0, 8),
    // 底部导航（对齐前端 --cst-elevation-nav 0 -4px 24px 0.40）
    tabBar: new AppThemeShadow(28, '#66000000', 0, -4)
  };

  readonly VIBRANT_LIGHT: ThemeShadows = {
    surface: new AppThemeShadow(8, '#1A635BC8', 0, 3),
    elevated: new AppThemeShadow(14, '#30635BC8', 0, 6),
    accent: new AppThemeShadow(22, '#522E66C0', 0, 8),
    control: new AppThemeShadow(4, '#0D635BC8', 0, 1),
    tabBar: new AppThemeShadow(8, '#18635BC8', 0, -2)
  };

  readonly VIBRANT_DARK: ThemeShadows = {
    surface: new AppThemeShadow(10, '#A6000000', 0, 3),
    elevated: new AppThemeShadow(16, '#80000000', 0, 5),
    // 深色 accent 光晕退化：近透明（深色下有色发光显脏，对齐扁平宣言）
    accent: new AppThemeShadow(22, '#1469A8FF', 0, 8),
    control: new AppThemeShadow(4, '#0F69A8FF', 0, 0),
    tabBar: new AppThemeShadow(10, '#80000000', 0, -3)
  };
}

export class ThemeConstants {
  static readonly DEFAULT_CLEAN_LIGHT_COLORS: ColorConfig = {
    pageBackground: '#F6F7F3',
    workspaceBackground: '#EBEEE5',
    pageGlow: '#F2F4F2',
    heroGlow: '#E8EDE8',
    surfaceBackground: '#F6F7F3',
    surfaceSubtle: '#F0F1F0',
    surfaceElevated: '#F6F7F3',
    borderColor: '#3D2A4934',
    floatingBorderColor: '#142A4934',
    sheetBackground: '#F6F7F3',
    accentBackground: '#3D7A55',
    accentText: '#FFFFFF',
    accentMuted: '#E2E9E2',
    chatBackground: '#F2F3F2',
    systemSurfaceBaseColor: '#EDEDED',
    overlayStrong: '#6B141C14',
    textPrimary: '#152219',
    textSecondary: '#3F5547',
    textTertiary: '#7A837E',
    iconMuted: '#3F5547',
    buttonSecondaryBackground: '#F0F1F0',
    buttonSecondaryText: '#152219',
    primaryButtonBorderColor: '#3D7A55',
    inputBackgroundColor: '#F0F1F0',
    quickActionBackgroundColor: '#FFFFFF',
    tabBarBackground: '#F9F9F5',
    tabBarInactive: '#3F5547',
    tabBarActiveBackground: '#DEE7DF',
    isDarkTheme: false,
    online: '#3D7A55',
    warning: '#8A6A2F',
    success: '#3D7A55',
    // danger 独立于 warning（对齐前端 forest light hot #B4543E）：原 error 误用暖棕与 warning 混淆
    error: '#B4543E',
    info: '#3D7A55',
    mapUser: '#3D7A55',
    mapGroup: '#4A703F',
    mapMoment: '#8A6426',
    // 淡底对齐前端 forest light danger 12% 预混（#B4543E × 12% over #F6F7F3）
    dangerSoft: '#EEE3DD',
    warningSoft: '#E9E6DB',
  };

  static readonly DEFAULT_CLEAN_DARK_COLORS: ColorConfig = {
    pageBackground: '#161A16',
    workspaceBackground: '#0F130E',
    pageGlow: '#171A17',
    heroGlow: '#1A201A',
    surfaceBackground: '#161A16',
    surfaceSubtle: '#242624',
    surfaceElevated: '#181C18',
    borderColor: '#2EA9C7B2',
    floatingBorderColor: '#14A9C7B2',
    sheetBackground: '#161A16',
    accentBackground: '#52B878',
    accentText: '#0C1911',
    accentMuted: '#1D2D22',
    chatBackground: '#0E100E',
    systemSurfaceBaseColor: '#111311',
    overlayStrong: '#8A000000',
    textPrimary: '#E8F1E9',
    textSecondary: '#A8BDAE',
    textTertiary: '#5A6B60',
    iconMuted: '#A8BDAE',
    buttonSecondaryBackground: '#242624',
    buttonSecondaryText: '#E8F1E9',
    primaryButtonBorderColor: '#52B878',
    inputBackgroundColor: '#242624',
    quickActionBackgroundColor: '#111311',
    tabBarBackground: '#161A16',
    tabBarInactive: '#A8BDAE',
    tabBarActiveBackground: '#1E2E23',
    isDarkTheme: true,
    online: '#52B878',
    warning: '#C4A75A',
    success: '#52B878',
    // danger 独立于 warning（对齐前端 forest dark hot #D97C5A）：原 error 与 warning 完全同色
    error: '#D97C5A',
    info: '#52B878',
    mapUser: '#52B878',
    mapGroup: '#88B85A',
    mapMoment: '#C4A75A',
    // 淡底对齐前端 forest dark danger 14% 预混（#D97C5A × 14% over #161A16）
    dangerSoft: '#31271F',
    warningSoft: '#2E2E20',
  };

  static readonly SHADOWS: ThemeShadowRegistry = new ThemeShadowRegistry();

  static readonly VIBRANT_LIGHT_COLORS: ColorConfig = {
    // 表面分层（对齐前端 vibrant.css）：画布灰蓝底 + 卡片纯白，卡片靠底色差与阴影浮起
    pageBackground: '#EFF3F9',
    workspaceBackground: '#E6EBF4',
    pageGlow: '#E8EDFF',
    heroGlow: '#DDE4FF',
    surfaceBackground: '#FFFFFF',
    surfaceSubtle: '#EAF1FA',
    surfaceElevated: '#F7FAFF',
    borderColor: '#2E2E66C0',
    floatingBorderColor: '#142E66C0',
    sheetBackground: '#F7FAFF',
    accentBackground: '#2E66C0',
    accentText: '#FFFFFF',
    accentMuted: '#DBE5F3',
    chatBackground: '#EDF3FC',
    systemSurfaceBaseColor: '#EDF3FC',
    overlayStrong: '#850C1930',
    textPrimary: '#16233A',
    textSecondary: '#52647F',
    textTertiary: '#767F92',
    iconMuted: '#52647F',
    buttonSecondaryBackground: '#EDF3FC',
    buttonSecondaryText: '#16233A',
    primaryButtonBorderColor: '#2E66C0',
    inputBackgroundColor: '#EAF1FA',
    quickActionBackgroundColor: '#FFFFFF',
    tabBarBackground: '#F7FAFF',
    tabBarInactive: '#52647F',
    tabBarActiveBackground: '#D6E1F2',
    isDarkTheme: false,
    online: '#14B8A6',
    warning: '#F59E0B',
    success: '#14B8A6',
    // danger 实底对比度：#F43F5E 配白字仅 3.67（配深字 4.28），均不达 4.5；加深至 #C81E4E
    //（白字 5.60、红字 on 浅底 4.92、红字 on dangerSoft 4.54，对齐前端 danger 实底同类修复）
    error: '#C81E4E',
    info: '#2E66C0',
    mapUser: '#2E66C0',
    mapGroup: '#0B7A56',
    mapMoment: '#C2470B',
    dangerSoft: '#F7E2EA',
    warningSoft: '#F7EEDF',
  };

  static readonly VIBRANT_DARK_COLORS: ColorConfig = {
    pageBackground: '#0B1424',
    workspaceBackground: '#060D19',
    pageGlow: '#122036',
    heroGlow: '#16263E',
    surfaceBackground: '#111D31',
    surfaceSubtle: '#14243A',
    surfaceElevated: '#16273C',
    borderColor: '#2E88B7FF',
    floatingBorderColor: '#1488B7FF',
    sheetBackground: '#0B1424',
    accentBackground: '#69A8FF',
    accentText: '#071426',
    accentMuted: '#182943',
    chatBackground: '#0D1726',
    systemSurfaceBaseColor: '#0B1424',
    overlayStrong: '#9E000814',
    textPrimary: '#E8F1FF',
    textSecondary: '#A5B6CE',
    textTertiary: '#5E6C85',
    iconMuted: '#A5B6CE',
    buttonSecondaryBackground: '#14243A',
    buttonSecondaryText: '#E8F1FF',
    primaryButtonBorderColor: '#69A8FF',
    inputBackgroundColor: '#14243A',
    quickActionBackgroundColor: '#0B1424',
    tabBarBackground: '#0B1424',
    tabBarInactive: '#A5B6CE',
    tabBarActiveBackground: '#172740',
    isDarkTheme: true,
    online: '#37C99A',
    warning: '#FBBF24',
    success: '#37C99A',
    error: '#FB7185',
    info: '#69A8FF',
    mapUser: '#69A8FF',
    mapGroup: '#34D399',
    mapMoment: '#FB923C',
    dangerSoft: '#2C2131',
    warningSoft: '#2C2B24',
  };
}
