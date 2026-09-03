/**
 * LucideIcon — 图标统一渲染组件（HarmonyOS 原生符号方案）
 *
 * 历史说明：组件名保留自旧版 lucide SVG 方案（rawfile/icons/*.svg +
 * Image ColorFilter 手写染色矩阵）。该方案存在两个根本问题：
 * 1. ColorFilter 矩阵值域按 0-255 传入，而 ArkUI 要求 0-1，非黑 RGB 全部
 *    被钳制为白色 → 浅色表面上图标/文字大面积"隐形"；
 * 2. 自维护 SVG 与系统符号双轨并行，同一 AppIcon 两种视觉。
 *
 * 现方案：直接渲染 HarmonyOS 系统符号（SymbolSpan + sys.symbol.*，AppIcon
 * 枚举值即符号名），fontColor 原生染色、随主题即时生效，与 SystemIcon 一致，
 * 组件 API（icon/iconSize/color/paletteRole）保持不变，调用点零改动。
 */

import { AppIcon } from './SystemIcon';
import { AppThemeManager } from '../core/theme/AppThemeManager';
import { KEY_RESOLVED_THEME_ID } from '../core/theme/AppThemeTypes';

const DEFAULT_ICON_SIZE: number = 24;
const DEFAULT_ICON_COLOR_SENTINEL: string = '__lucide_icon_default_color__';

/**
 * 图标组件参数：
 *   - icon: AppIcon 枚举（sys.symbol.* 系统符号名）
 *   - iconSize: 图标边长 px（默认 24）
 *   - color: 直接色值（hex string，与 paletteRole 互斥）
 *   - paletteRole: 调色板语义角色（accent / accentText / textPrimary /
 *     textSecondary / success / warning / error / tabActive / tabInactive）
 *   - overrideThemeId: 子组件显式传主题 ID（为空走 @StorageProp 全局）
 *
 * 用法：
 *   LucideIcon({ icon: AppIcon.HOME, iconSize: 24, paletteRole: 'accent' })
 *   LucideIcon({ icon: AppIcon.SEARCH, iconSize: 20, color: '#16233A' })
 */
@Component
export struct LucideIcon {
  @Prop icon: AppIcon;
  @Prop iconSize: number = DEFAULT_ICON_SIZE;
  @Prop color: ResourceColor = DEFAULT_ICON_COLOR_SENTINEL;
  @Prop paletteRole: string = '';
  @Prop overrideThemeId: string = '';

  @StorageProp(KEY_RESOLVED_THEME_ID) storageThemeId: string = 'vibrant_light';

  private getEffectiveThemeId(): string {
    return this.overrideThemeId !== '' ? this.overrideThemeId : this.storageThemeId;
  }

  private getPalette() {
    return AppThemeManager.getPaletteById(this.getEffectiveThemeId());
  }

  /** 解析最终色值（palette 为 hex string；color 支持 string/number(Color)/Resource） */
  private resolveColor(): ResourceColor {
    if (this.color !== DEFAULT_ICON_COLOR_SENTINEL) {
      if (typeof this.color === 'string') {
        return this.color as string;
      }
      if (typeof this.color === 'number') {
        return this.color;
      }
      return this.color;
    }
    if (this.paletteRole !== '') {
      const pal = this.getPalette();
      switch (this.paletteRole) {
        case 'accent': return pal.accentBackground;
        case 'accentText': return pal.accentText;
        case 'textPrimary': return pal.textPrimary;
        case 'textSecondary': return pal.textSecondary;
        case 'textTertiary': return pal.textTertiary;
        case 'success': return pal.success;
        case 'warning': return pal.warning;
        case 'error': return pal.error;
        case 'tabActive': return pal.accentBackground;
        case 'tabInactive': return pal.tabBarInactive;
        default: return pal.textSecondary;
      }
    }
    return this.getPalette().textSecondary;
  }

  build() {
    Text() {
      SymbolSpan($r(this.icon))
        .fontSize(this.iconSize)
        .fontColor([this.resolveColor()])
    }
  }
}

/**
 * 容器尺寸辅助：与前端 lucide 一致
 *   xs: 12  sm: 16  md: 20  lg: 24  xl: 32
 */
export const LUCIDE_ICON_SIZE: Record<string, number> = {
  'xs': 12,
  'sm': 16,
  'md': 20,
  'lg': 24,
  'xl': 32
};
