/**
 * 主题系统常量
 * 统一管理所有主题的颜色值，避免重复定义
 */

import { AppThemeShadow } from './AppThemeTypes';

export interface ColorConfig {
  pageBackground: string;
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

  control: AppThemeShadow;
  tabBar: AppThemeShadow;
}

class ThemeShadowRegistry {
  readonly DEFAULT_CLEAN_LIGHT: ThemeShadows = {
    // 卡片轻微抬升（对齐前端 elevation-card 的"浮起"感；ArkUI 无 spread 做不了硬边 ring，用轻量模糊近似）
    surface: new AppThemeShadow(10, '#143D7A55', 0, 2),
    // 弹层抬升（对齐前端 --cst-elevation-modal 0 24px 80px 0.24）
    elevated: new AppThemeShadow(40, '#3D000000', 0, 12),
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
    // 菜单浮层
    control: new AppThemeShadow(40, '#59000000', 0, 8),
    // 底部导航（对齐前端 --cst-elevation-nav 0 -4px 24px 0.40）
    tabBar: new AppThemeShadow(28, '#66000000', 0, -4)
  };

  readonly VIBRANT_LIGHT: ThemeShadows = {
    surface: new AppThemeShadow(8, '#0F000000', 0, 2),
    elevated: new AppThemeShadow(24, '#26000000', 0, 8),
    control: new AppThemeShadow(16, '#1F000000', 0, 4),
    tabBar: new AppThemeShadow(8, '#0A000000', 0, -1)
  };

  readonly VIBRANT_DARK: ThemeShadows = {
    surface: new AppThemeShadow(10, '#59000000', 0, 3),
    elevated: new AppThemeShadow(24, '#80000000', 0, 8),
    control: new AppThemeShadow(16, '#4D000000', 0, 4),
    tabBar: new AppThemeShadow(10, '#66000000', 0, -3)
  };

  readonly INKWASH_LIGHT: ThemeShadows = {
    // 宣纸扁平：卡片用极淡墨影近似前端「细墨线描边」表达层级（ArkUI 无 spread 做不了硬边 ring）
    surface: new AppThemeShadow(10, '#0F2B2A26', 0, 2),
    // 弹层抬升（对齐前端 --cst-elevation-modal 0 24px 80px 0.24，留白性格略收敛）
    elevated: new AppThemeShadow(40, '#2E000000', 0, 12),
    // 菜单浮层（对齐前端 --cst-elevation-menu 0 16px 40px 0.16）
    control: new AppThemeShadow(32, '#24000000', 0, 8),
    // 底部导航（对齐前端 --cst-elevation-nav，向上微光）
    tabBar: new AppThemeShadow(24, '#142B2A26', 0, -4)
  };

  readonly INKWASH_DARK: ThemeShadows = {
    // 深色下黑阴影对比弱，卡片抬升改用更亮的描边承载（border 单独加亮），阴影只做轻微加深
    surface: new AppThemeShadow(10, '#59000000', 0, 3),
    // 弹层抬升（对齐前端 --cst-elevation-modal 0 28px 88px 0.42）
    elevated: new AppThemeShadow(24, '#80000000', 0, 8),
    // 菜单浮层
    control: new AppThemeShadow(16, '#4D000000', 0, 4),
    // 底部导航（对齐前端 --cst-elevation-nav 0 -2px 10px 0.35）
    tabBar: new AppThemeShadow(10, '#66000000', 0, -3)
  };
}

export class ThemeConstants {
  static readonly DEFAULT_CLEAN_LIGHT_COLORS: ColorConfig = {
    pageBackground: '#F6F7F3',
    surfaceBackground: '#F6F7F3',
    surfaceSubtle: '#F0F1F0',
    surfaceElevated: '#F6F7F3',
    borderColor: '#3D2A4934',
    floatingBorderColor: '#142A4934',
    sheetBackground: '#F6F7F3',
    accentBackground: '#39714F',
    accentText: '#FFFFFF',
    accentMuted: '#E2E9E2',
    chatBackground: '#F2F3F2',
    systemSurfaceBaseColor: '#EDEDED',
    overlayStrong: '#6B141C14',
    textPrimary: '#152219',
    textSecondary: '#3F5547',
    // 审计 P3-4：原 #7A837E on 纸面 #F6F7F3 对比度仅 3.77(<4.5)，加深至 #5F6C64（5.12），
    // 色相同族微加深、仍保持比 textSecondary(#3F5547) 弱一档的层次
    textTertiary: '#5F6C64',
    iconMuted: '#3F5547',
    buttonSecondaryBackground: '#F0F1F0',
    buttonSecondaryText: '#152219',
    primaryButtonBorderColor: '#39714F',
    inputBackgroundColor: '#F0F1F0',
    quickActionBackgroundColor: '#FFFFFF',
    tabBarBackground: '#F9F9F5',
    tabBarInactive: '#3F5547',
    tabBarActiveBackground: '#DEE7DF',
    isDarkTheme: false,
    online: '#39714F',
    warning: '#7E602B',
    success: '#39714F',
    // danger 独立于 warning（对齐前端 forest light hot #B4543E）：原 error 误用暖棕与 warning 混淆
    error: '#A04B37',
    info: '#39714F',
    mapUser: '#39714F',
    mapGroup: '#4A703F',
    mapMoment: '#8A6426',
    // 淡底对齐前端 forest light danger 12% 预混（#B4543E × 12% over #F6F7F3）
    dangerSoft: '#EEE3DD',
    warningSoft: '#E9E6DB',
  };

  static readonly DEFAULT_CLEAN_DARK_COLORS: ColorConfig = {
    pageBackground: '#161A16',
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
    // 审计 P3-4：原 #5A6B60 on #161A16 对比度仅 3.10(<4.5)，提亮至 #7E8D82（5.04），
    // 仍保持比 textSecondary(#A8BDAE) 弱一档的层次
    textTertiary: '#7E8D82',
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
    surfaceBackground: '#FFFFFF',
    surfaceSubtle: '#EAF1FA',
    surfaceElevated: '#F7FAFF',
    borderColor: '#2E2B5EB2',
    floatingBorderColor: '#142B5EB2',
    sheetBackground: '#F7FAFF',
    accentBackground: '#2B5EB2',
    accentText: '#FFFFFF',
    accentMuted: '#DBE5F3',
    chatBackground: '#EDF3FC',
    systemSurfaceBaseColor: '#EDF3FC',
    overlayStrong: '#850C1930',
    textPrimary: '#16233A',
    textSecondary: '#52647F',
    // 审计 P3-4：原 #767F92 on 卡片白底 4.02 / on 画布 #EFF3F9 3.61（均 <4.5），
    // 加深至 #626C80（5.28 / 4.74），同族微加深、仍比 textSecondary(#52647F) 弱一档
    textTertiary: '#626C80',
    iconMuted: '#52647F',
    buttonSecondaryBackground: '#EDF3FC',
    buttonSecondaryText: '#16233A',
    primaryButtonBorderColor: '#2B5EB2',
    inputBackgroundColor: '#EAF1FA',
    quickActionBackgroundColor: '#FFFFFF',
    tabBarBackground: '#F7FAFF',
    tabBarInactive: '#52647F',
    tabBarActiveBackground: '#D6E0F0',
    isDarkTheme: false,
    online: '#0D6A62',
    // warning 两次加深（#F59E0B→#B45309→#9C4708，对齐前端 vibrant light 同名校准）：
    // 压白底 2.1→6.33、压 warning 淡底 4.21→4.72（均 ≥4.5）
    warning: '#9C4708',
    success: '#0D6A62',
    // danger 实底对比度：#F43F5E 配白字仅 3.67（配深字 4.28），均不达 4.5；加深至 #C81E4E
    //（白字 5.60、红字 on 浅底 4.92、红字 on dangerSoft 4.54，对齐前端 danger 实底同类修复）
    error: '#C81E4E',
    info: '#2B5EB2',
    mapUser: '#2B5EB2',
    mapGroup: '#0B7653',
    mapMoment: '#C2470B',
    dangerSoft: '#F7E2EA',
    // warningSoft = warning 13% 预合成叠 page-bg（随 warning 加深同步复算）
    warningSoft: '#EEE4DF',
  };

  static readonly VIBRANT_DARK_COLORS: ColorConfig = {
    pageBackground: '#0B1424',
    // 卡面提档对齐前端 vibrant dark（UI 复审 D2 token 级微调，#111D31→#152134）
    surfaceBackground: '#152134',
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
    // 审计 P3-4：原 #5E6C85 on #111D31 对比度仅 3.24(<4.5)，提亮至 #7D8BA3（4.90），
    // 仍保持比 textSecondary(#A5B6CE) 弱一档的层次
    textTertiary: '#7D8BA3',
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

  static readonly INKWASH_LIGHT_COLORS: ColorConfig = {
    // 表面分层（对齐前端 inkwash.css）：画布宣纸暖底 + 卡片宣纸白，层级靠细墨线描边表达
    pageBackground: '#F5F2EA',
    surfaceBackground: '#FBF9F3',
    surfaceSubtle: '#F0EBE0',
    surfaceElevated: '#FBF9F3',
    // borderColor = CSS rgba(43,42,38,0.20)，ArkTS #AARRGGBB alpha 前置（跨端校验直比）
    borderColor: '#332B2A26',
    floatingBorderColor: '#142B2A26',
    sheetBackground: '#F5F2EA',
    accentBackground: '#A63A2E',
    accentText: '#FFFFFF',
    // accentMuted = accent-soft（10% 朱砂）预合成到 #F5F2EA（跨端校验直比）
    accentMuted: '#EDDFD7',
    chatBackground: '#F1EDE3',
    systemSurfaceBaseColor: '#F1EDE3',
    // overlayStrong = CSS rgba(26,24,20,0.42)，ArkTS #AARRGGBB alpha 前置
    overlayStrong: '#6B1A1814',
    textPrimary: '#2B2A26',
    textSecondary: '#6B675D',
    // 宣纸灰 tertiary：on 画布 #F5F2EA 对比 4.68（≥4.5），比 textSecondary(#6E6A60, 4.82) 弱一档
    textTertiary: '#706C62',
    iconMuted: '#6E6A60',
    buttonSecondaryBackground: '#F0EBE0',
    buttonSecondaryText: '#2B2A26',
    primaryButtonBorderColor: '#A63A2E',
    inputBackgroundColor: '#F0EBE0',
    quickActionBackgroundColor: '#FFFFFF',
    tabBarBackground: '#FAF8F2',
    tabBarInactive: '#6E6A60',
    // tabBarActiveBackground = accent @ 0.13 预合成到 #F5F2EA（跨端校验直比）
    tabBarActiveBackground: '#EBDAD2',
    isDarkTheme: false,
    online: '#3F7256',
    warning: '#7F6422',
    success: '#3F7256',
    error: '#9E2140',
    info: '#A63A2E',
    mapUser: '#A63A2E',
    mapGroup: '#4A7048',
    mapMoment: '#8A6426',
    // 淡底对齐前端 inkwash light danger/warning 12% 预混（#9E2140 / #8F7026 × 12% over #F5F2EA）
    dangerSoft: '#EAD9D5',
    warningSoft: '#E9E2D2',
  };

  static readonly INKWASH_DARK_COLORS: ColorConfig = {
    pageBackground: '#181A1C',
    surfaceBackground: '#212426',
    surfaceSubtle: '#26292B',
    surfaceElevated: '#24272A',
    // borderColor = CSS rgba(234,230,218,0.16)，ArkTS #AARRGGBB alpha 前置（跨端校验直比）
    borderColor: '#29EAE6DA',
    floatingBorderColor: '#14EAE6DA',
    sheetBackground: '#181A1C',
    accentBackground: '#DF7F6A',
    accentText: '#1F1512',
    // accentMuted = accent-soft（12% 暖赭）预合成到 #181A1C（跨端校验直比）
    accentMuted: '#302625',
    chatBackground: '#1D2022',
    systemSurfaceBaseColor: '#181A1C',
    // overlayStrong = CSS rgba(0,0,0,0.58)，ArkTS #AARRGGBB alpha 前置
    overlayStrong: '#94000000',
    textPrimary: '#EAE6DA',
    textSecondary: '#A8A49A',
    // 宣纸灰 tertiary：on 画布 #181A1C 对比 5.20（≥4.5），比 textSecondary(#A8A49A, 7.01) 弱一档
    textTertiary: '#908C82',
    iconMuted: '#A8A49A',
    buttonSecondaryBackground: '#26292B',
    buttonSecondaryText: '#EAE6DA',
    primaryButtonBorderColor: '#DF7F6A',
    inputBackgroundColor: '#26292B',
    quickActionBackgroundColor: '#141617',
    tabBarBackground: '#181A1C',
    tabBarInactive: '#A8A49A',
    // tabBarActiveBackground = accent @ 0.13 预合成到 #181A1C（跨端校验直比）
    tabBarActiveBackground: '#322726',
    isDarkTheme: true,
    online: '#82B297',
    warning: '#D6A54E',
    success: '#82B297',
    error: '#DE8496',
    info: '#DF7F6A',
    mapUser: '#DF7F6A',
    mapGroup: '#7FA88B',
    mapMoment: '#C89B5A',
    // 淡底对齐前端 inkwash dark danger/warning 14% 预混（#DE8496 / #D6A54E × 14% over #181A1C）
    dangerSoft: '#34292D',
    warningSoft: '#332E23',
  };
}
