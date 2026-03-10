import { StorageKeys } from '../../constants/AppConstants';
import { PreferencesUtil } from '../storage/PreferencesUtil';
import { SystemSurfaceStyle, SystemUiThemeStore } from '../utils/SystemUiThemeStore';
import {
  AppThemePalette,
  DEFAULT_RESOLVED_THEME_ID,
  DEFAULT_THEME_MODE,
  KEY_RESOLVED_THEME_ID,
  KEY_THEME_MODE,
  ResolvedThemeId,
  ThemeMode,
  ThemeOptionItem
} from './AppThemeTypes';

const DEFAULT_CLEAN_LIGHT: AppThemePalette = {
  id: 'default_clean_light',
  label: '当前默认',
  description: '清爽、干净、通用',
  pageBackground: '#FFFFFF',
  surfaceBackground: '#FFFFFF',
  surfaceSubtle: '#F3F4F6',
  borderColor: '#E5E7EB',
  sheetBackground: '#FFFFFF',
  accentBackground: '#4A7A57',
  accentText: '#FFFFFF',
  chatBackground: '#F5F6FA',
  systemSurfaceBaseColor: '#FFFFFF',
  preferDarkContent: true,
  surfaceShadow: { radius: 6, color: '#00000000', offsetX: 0, offsetY: 0 },
  elevatedShadow: { radius: 8, color: '#14000000', offsetX: 0, offsetY: 3 },
  accentShadow: { radius: 0, color: '#00000000', offsetX: 0, offsetY: 0 },
  controlShadow: { radius: 0, color: '#00000000', offsetX: 0, offsetY: 0 },
  tabBarShadow: { radius: 6, color: '#12000000', offsetX: 0, offsetY: -1 },
  overlayStrong: '#80000000'
};

const DEFAULT_CLEAN_DARK: AppThemePalette = {
  id: 'default_clean_dark',
  label: '当前默认',
  description: '清爽、干净、通用',
  pageBackground: '#0F1115',
  surfaceBackground: '#171A20',
  surfaceSubtle: '#222733',
  borderColor: '#2D3544',
  sheetBackground: '#171A20',
  accentBackground: '#6BAF86',
  accentText: '#0F1115',
  chatBackground: '#11131D',
  systemSurfaceBaseColor: '#0F1115',
  preferDarkContent: false,
  surfaceShadow: { radius: 8, color: '#26000000', offsetX: 0, offsetY: 3 },
  elevatedShadow: { radius: 10, color: '#2A000000', offsetX: 0, offsetY: 4 },
  accentShadow: { radius: 0, color: '#00000000', offsetX: 0, offsetY: 0 },
  controlShadow: { radius: 0, color: '#00000000', offsetX: 0, offsetY: 0 },
  tabBarShadow: { radius: 10, color: '#26000000', offsetX: 0, offsetY: -1 },
  overlayStrong: '#99000000'
};

const TACTILE_MINIMAL_LIGHT: AppThemePalette = {
  id: 'tactile_minimal_light',
  label: 'Tactile Minimalist',
  description: '奶白、柔和、带轻触感',
  // 参考拟物截图：整体偏暖米白，背景略深、卡片略浅
  pageBackground: '#E9E1D4',      // 页面大背景：柔和奶油米色
  surfaceBackground: '#F6EFE4',   // 卡片 / 底栏背景：更亮一点的象牙白
  surfaceSubtle: '#DDD1C2',       // 次级区域 / 按钮背景：略深，用于凹槽对比
  borderColor: '#D0C4B4',         // 卡片描边：低对比暖灰
  sheetBackground: '#F4EDDF',     // 底部弹窗：介于背景和卡片之间
  accentBackground: '#E4C899',    // 主要操作按钮：金杏色高亮，模拟渐变中位色
  accentText: '#6C5234',          // 按钮文字：偏深棕色，保证可读性
  chatBackground: '#EDE3D6',      // 聊天列表背景：略浅于 pageBackground
  systemSurfaceBaseColor: '#E9E1D4',
  preferDarkContent: true,
  // 卡片希望是柔和的“整圈泛光”，不要只在下边缘有阴影
  surfaceShadow: { radius: 22, color: '#1F9A826418', offsetX: 0, offsetY: 0 },
  // 更高一层的浮起可以略微向下偏一点
  elevatedShadow: { radius: 26, color: '#2C9A82661C', offsetX: 0, offsetY: 6 },
  accentShadow: { radius: 18, color: '#2A9E7D621E', offsetX: 0, offsetY: 6 },
  controlShadow: { radius: 12, color: '#1E9E846818', offsetX: 0, offsetY: 4 },
  tabBarShadow: { radius: 18, color: '#24A18E7A1A', offsetX: 0, offsetY: -2 },
  overlayStrong: '#8A6F584D'
};

const TACTILE_MINIMAL_DARK: AppThemePalette = {
  id: 'tactile_minimal_dark',
  label: 'Tactile Minimalist',
  description: '深海蓝、暖金、柔和泛光',
  pageBackground: '#09131E',
  surfaceBackground: '#102030',
  surfaceSubtle: '#0C1A29',
  borderColor: '#24384B',
  sheetBackground: '#112334',
  accentBackground: '#E3C9A5',
  accentText: '#182638',
  chatBackground: '#0B1725',
  systemSurfaceBaseColor: '#09131E',
  preferDarkContent: false,
  surfaceShadow: { radius: 18, color: '#2A5A84A6', offsetX: 0, offsetY: 0 },
  elevatedShadow: { radius: 24, color: '#304F74C0', offsetX: 0, offsetY: 8 },
  accentShadow: { radius: 22, color: '#28E3C9A5', offsetX: 0, offsetY: 7 },
  controlShadow: { radius: 14, color: '#264A6B96', offsetX: 0, offsetY: 4 },
  tabBarShadow: { radius: 20, color: '#30516D9E', offsetX: 0, offsetY: -2 },
  overlayStrong: '#B0000000'
};

const ZEN_ARTISTIC_LIGHT: AppThemePalette = {
  id: 'zen_artistic_light',
  label: 'Zen Artistic',
  description: '雾蓝、安静、艺术感',
  // 参考禅意截图：丝绒质感的雾蓝背景，凹槽区域略深
  pageBackground: '#C7D2D8',      // 页面大背景：偏灰的雾蓝
  surfaceBackground: '#D3DDE2',   // 卡片 / 底栏：比背景更亮一点
  surfaceSubtle: '#B7C4CB',       // 凹槽 / 次级区域：更深的蓝灰
  borderColor: '#ACBAC2',         // 轮廓线：低对比冷灰
  sheetBackground: '#D5E0E4',     // 底部弹窗：介于背景与卡片之间
  accentBackground: '#E3D5C4',    // 主要操作按钮：温暖米金色，与蓝灰形成对比
  accentText: '#705743',          // 按钮文字：暖棕色，保持柔和
  chatBackground: '#C1CDD3',      // 聊天背景：略深，避免过曝
  systemSurfaceBaseColor: '#C7D2D8',
  preferDarkContent: true,
  // 雾蓝主题卡片同样使用“整圈泛光”，偏冷一点的柔光
  surfaceShadow: { radius: 20, color: '#1E4F677018', offsetX: 0, offsetY: 0 },
  elevatedShadow: { radius: 24, color: '#28566B7A24', offsetX: 0, offsetY: 6 },
  accentShadow: { radius: 16, color: '#245A6D7A22', offsetX: 0, offsetY: 6 },
  controlShadow: { radius: 10, color: '#1E4F677018', offsetX: 0, offsetY: 4 },
  tabBarShadow: { radius: 16, color: '#224F67701C', offsetX: 0, offsetY: -2 },
  overlayStrong: '#8A7D7166'
};

const THEME_OPTIONS: ThemeOptionItem[] = [
  { mode: 'follow_system', label: '跟随系统', description: '自动切换深浅主题' },
  { mode: 'default_clean', label: '当前默认', description: '清爽、干净、通用' },
  { mode: 'tactile_minimal', label: 'Tactile Minimalist', description: '奶白、柔和、带轻触感' },
  { mode: 'zen_artistic', label: 'Zen Artistic', description: '雾蓝、安静、艺术感' }
];

export class AppThemeManager {
  static async initialize(): Promise<void> {
    const storedThemeMode = await PreferencesUtil.get<string>(StorageKeys.THEME_MODE, DEFAULT_THEME_MODE);
    const themeMode = AppThemeManager.normalizeThemeMode(storedThemeMode);
    AppStorage.setOrCreate<string>(KEY_THEME_MODE, themeMode);
    AppThemeManager.refreshResolvedTheme();
  }

  static async setThemeMode(themeMode: ThemeMode): Promise<void> {
    const normalizedThemeMode = AppThemeManager.normalizeThemeMode(themeMode);
    await PreferencesUtil.set(StorageKeys.THEME_MODE, normalizedThemeMode);
    AppStorage.setOrCreate<string>(KEY_THEME_MODE, normalizedThemeMode);
    AppThemeManager.refreshResolvedTheme();
  }

  static getThemeMode(): ThemeMode {
    return AppThemeManager.normalizeThemeMode(AppStorage.get<string>(KEY_THEME_MODE) ?? DEFAULT_THEME_MODE);
  }

  static getResolvedThemeId(): ResolvedThemeId {
    const resolvedThemeId = AppStorage.get<string>(KEY_RESOLVED_THEME_ID) ?? DEFAULT_RESOLVED_THEME_ID;
    return AppThemeManager.normalizeResolvedThemeId(resolvedThemeId);
  }

  static onSystemColorModeChange(isDarkMode: boolean): void {
    AppStorage.setOrCreate<boolean>('isDarkMode', isDarkMode);
    AppThemeManager.refreshResolvedTheme();
  }

  static getCurrentPalette(): AppThemePalette {
    return AppThemeManager.getPaletteById(AppThemeManager.getResolvedThemeId());
  }

  static getSafeCurrentPalette(): AppThemePalette {
    try {
      return AppThemeManager.getCurrentPalette();
    } catch (_) {
      return DEFAULT_CLEAN_LIGHT;
    }
  }

  static getPaletteById(resolvedThemeId: string): AppThemePalette {
    switch (AppThemeManager.normalizeResolvedThemeId(resolvedThemeId)) {
      case 'default_clean_dark':
        return DEFAULT_CLEAN_DARK;
      case 'tactile_minimal_light':
        return TACTILE_MINIMAL_LIGHT;
      case 'tactile_minimal_dark':
        return TACTILE_MINIMAL_DARK;
      case 'zen_artistic_light':
        return ZEN_ARTISTIC_LIGHT;
      case 'default_clean_light':
      default:
        return DEFAULT_CLEAN_LIGHT;
    }
  }

  static getCurrentSurfaceStyle(): SystemSurfaceStyle {
    const palette = AppThemeManager.getSafeCurrentPalette();
    return {
      baseColor: palette.systemSurfaceBaseColor,
      overlayOpacity: 1,
      preferDarkContent: palette.preferDarkContent
    };
  }

  static syncSystemSurface(): void {
    SystemUiThemeStore.setStyle(AppThemeManager.getCurrentSurfaceStyle());
  }

  static getThemeOptions(): ThemeOptionItem[] {
    return THEME_OPTIONS;
  }

  static getThemeLabel(themeMode: string): string {
    const normalizedThemeMode = AppThemeManager.normalizeThemeMode(themeMode);
    const match = THEME_OPTIONS.find((item: ThemeOptionItem) => item.mode === normalizedThemeMode);
    return match ? match.label : '当前默认';
  }

  static getSoftStatusBackground(status: 'neutral' | 'success' | 'warning' | 'danger' | 'info'): string {
    const palette = AppThemeManager.getSafeCurrentPalette();
    switch (status) {
      case 'success':
        return palette.id === 'default_clean_dark' ? '#183428' : '#E7F6EC';
      case 'warning':
        return palette.id === 'default_clean_dark' ? '#3B2B16' : '#FFF4DF';
      case 'danger':
        return palette.id === 'default_clean_dark' ? '#3A1F24' : '#FDEAEC';
      case 'info':
        return palette.id === 'default_clean_dark' ? '#1D2B35' : '#E8F1FB';
      case 'neutral':
      default:
        return palette.surfaceSubtle;
    }
  }

  static getBrandAccent(): string {
    return AppThemeManager.getSafeCurrentPalette().accentBackground;
  }

  static getBrandAccentText(): string {
    return AppThemeManager.getSafeCurrentPalette().accentText;
  }

  private static refreshResolvedTheme(): void {
    const resolvedThemeId = AppThemeManager.resolveThemeId(
      AppThemeManager.getThemeMode(),
      AppStorage.get<boolean>('isDarkMode') === true
    );
    AppStorage.setOrCreate<string>(KEY_RESOLVED_THEME_ID, resolvedThemeId);
    AppThemeManager.syncSystemSurface();
  }

  private static resolveThemeId(themeMode: ThemeMode, isDarkMode: boolean): ResolvedThemeId {
    switch (themeMode) {
      case 'follow_system':
      case 'default_clean':
        return isDarkMode ? 'default_clean_dark' : 'default_clean_light';
      case 'tactile_minimal':
        return isDarkMode ? 'tactile_minimal_dark' : 'tactile_minimal_light';
      case 'zen_artistic':
        return 'zen_artistic_light';
      default:
        return DEFAULT_RESOLVED_THEME_ID;
    }
  }

  private static normalizeThemeMode(themeMode: string): ThemeMode {
    switch (themeMode) {
      case 'default_clean':
      case 'tactile_minimal':
      case 'zen_artistic':
      case 'follow_system':
        return themeMode;
      default:
        return DEFAULT_THEME_MODE;
    }
  }

  private static normalizeResolvedThemeId(resolvedThemeId: string): ResolvedThemeId {
    switch (resolvedThemeId) {
      case 'default_clean_dark':
      case 'tactile_minimal_light':
      case 'tactile_minimal_dark':
      case 'zen_artistic_light':
      case 'default_clean_light':
        return resolvedThemeId;
      default:
        return DEFAULT_RESOLVED_THEME_ID;
    }
  }
}
