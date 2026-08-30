/**
 * LucideIcon — 对齐前端 @lucide/svelte 线描 SVG 图标
 *
 * 与 SystemIcon 关系：
 * - SystemIcon 继续渲染 HarmonyOS 系统 SymbolFont（填充式）；保留兼容所有旧调用点
 * - LucideIcon 渲染 lucide 线描 SVG（24x24 stroke=2 fill=none），对齐前端视觉
 *
 * 选用规则：
 * - 关键位置（TabBar / PageHeader / Hero 卡 / 按钮图标）优先用 LucideIcon
 * - 长尾位置可继续用 SystemIcon，避免一次性迁移所有调用点
 *
 * 资源：resources/rawfile/icons/<name>.svg（来自 lucide-static v0.460）
 * 渲染：Image + ColorFilter 4x5 颜色矩阵 tint（line-icon 整图染为目标色，
 *       alpha 保留、RGB 替换为 palette 色）
 * Fallback：未支持的 AppIcon 走 SymbolSpan，调用点不报错
 *
 * 注意：prop 名用 iconSize（避免与 CustomComponent 基类 size 冲突）；
 *      if-else 链用 string 局部变量比较（避免 ArkTS enum 类型收窄误报）
 */

import { AppIcon } from './SystemIcon';
import { AppThemeManager } from '../core/theme/AppThemeManager';
import { KEY_RESOLVED_THEME_ID } from '../core/theme/AppThemeTypes';
import { DesignConstants } from '../constants/DesignConstants';

const DEFAULT_ICON_SIZE: number = 24;
const DEFAULT_ICON_COLOR_SENTINEL: string = '__lucide_icon_default_color__';

/**
 * LucideIcon 组件参数：
 *   - icon: 复用现有 AppIcon 枚举（保持调用点类型一致）
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

  /** 解析最终 hex 色（palette 为 hex string；color 支持 string/number(Color)/Resource） */
  private resolveColorHex(): string {
    if (this.color !== DEFAULT_ICON_COLOR_SENTINEL) {
      if (typeof this.color === 'string') {
        return this.color as string;
      }
      if (typeof this.color === 'number') {
        // Color 枚举为 0xAARRGGBB
        const hex: string = (this.color as number).toString(16).padStart(8, '0');
        return '#' + hex.substring(2);
      }
      // Resource 无法解析为 hex：退中性灰
      return '#8A94A6';
    }
    if (this.paletteRole !== '') {
      const pal = this.getPalette();
      switch (this.paletteRole) {
        case 'accent': return pal.accentBackground;
        case 'accentText': return pal.accentText;
        case 'textPrimary': return pal.textPrimary;
        case 'textSecondary': return pal.textSecondary;
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

  /**
   * 生成 4x5 颜色矩阵（行优先）：RGB 替换为目标色、alpha 保留。
   * [0,0,0,0,R,  0,0,0,0,G,  0,0,0,0,B,  0,0,0,1,0]
   * 透明像素 outA=0 保持透明；stroke 像素 outA=1 染成目标色。
   */
  private tintColorFilter(): ColorFilter {
    const hex = this.resolveColorHex();
    const r = parseInt(hex.substring(1, 3), 16);
    const g = parseInt(hex.substring(3, 5), 16);
    const b = parseInt(hex.substring(5, 7), 16);
    const matrix: number[] = [
      0, 0, 0, 0, r,
      0, 0, 0, 0, g,
      0, 0, 0, 0, b,
      0, 0, 0, 1, 0
    ];
    return new ColorFilter(matrix);
  }

  /** 渲染 lucide SVG；不支持的图标走 SymbolSpan fallback */
  @Builder
  private renderLucide(iconName: string) {
    if (iconName === 'house') {
      Image($rawfile('icons/house.svg'))
        .width(this.iconSize).height(this.iconSize)
        .colorFilter(this.tintColorFilter())
        .interpolation(ImageInterpolation.High)
    } else if (iconName === 'map-pin') {
      Image($rawfile('icons/map-pin.svg'))
        .width(this.iconSize).height(this.iconSize)
        .colorFilter(this.tintColorFilter())
        .interpolation(ImageInterpolation.High)
    } else if (iconName === 'message-circle') {
      Image($rawfile('icons/message-circle.svg'))
        .width(this.iconSize).height(this.iconSize)
        .colorFilter(this.tintColorFilter())
        .interpolation(ImageInterpolation.High)
    } else if (iconName === 'circle-user') {
      Image($rawfile('icons/circle-user.svg'))
        .width(this.iconSize).height(this.iconSize)
        .colorFilter(this.tintColorFilter())
        .interpolation(ImageInterpolation.High)
    } else if (iconName === 'users') {
      Image($rawfile('icons/users.svg'))
        .width(this.iconSize).height(this.iconSize)
        .colorFilter(this.tintColorFilter())
        .interpolation(ImageInterpolation.High)
    } else if (iconName === 'map') {
      Image($rawfile('icons/map.svg'))
        .width(this.iconSize).height(this.iconSize)
        .colorFilter(this.tintColorFilter())
        .interpolation(ImageInterpolation.High)
    } else if (iconName === 'bell') {
      Image($rawfile('icons/bell.svg'))
        .width(this.iconSize).height(this.iconSize)
        .colorFilter(this.tintColorFilter())
        .interpolation(ImageInterpolation.High)
    } else if (iconName === 'settings') {
      Image($rawfile('icons/settings.svg'))
        .width(this.iconSize).height(this.iconSize)
        .colorFilter(this.tintColorFilter())
        .interpolation(ImageInterpolation.High)
    } else if (iconName === 'plus') {
      Image($rawfile('icons/plus.svg'))
        .width(this.iconSize).height(this.iconSize)
        .colorFilter(this.tintColorFilter())
        .interpolation(ImageInterpolation.High)
    } else if (iconName === 'x') {
      Image($rawfile('icons/x.svg'))
        .width(this.iconSize).height(this.iconSize)
        .colorFilter(this.tintColorFilter())
        .interpolation(ImageInterpolation.High)
    } else if (iconName === 'chevron-left') {
      Image($rawfile('icons/chevron-left.svg'))
        .width(this.iconSize).height(this.iconSize)
        .colorFilter(this.tintColorFilter())
        .interpolation(ImageInterpolation.High)
    } else if (iconName === 'chevron-right') {
      Image($rawfile('icons/chevron-right.svg'))
        .width(this.iconSize).height(this.iconSize)
        .colorFilter(this.tintColorFilter())
        .interpolation(ImageInterpolation.High)
    } else if (iconName === 'chevron-down') {
      Image($rawfile('icons/chevron-down.svg'))
        .width(this.iconSize).height(this.iconSize)
        .colorFilter(this.tintColorFilter())
        .interpolation(ImageInterpolation.High)
    } else if (iconName === 'chevron-up') {
      Image($rawfile('icons/chevron-up.svg'))
        .width(this.iconSize).height(this.iconSize)
        .colorFilter(this.tintColorFilter())
        .interpolation(ImageInterpolation.High)
    } else if (iconName === 'ellipsis') {
      Image($rawfile('icons/ellipsis.svg'))
        .width(this.iconSize).height(this.iconSize)
        .colorFilter(this.tintColorFilter())
        .interpolation(ImageInterpolation.High)
    } else if (iconName === 'check') {
      Image($rawfile('icons/check.svg'))
        .width(this.iconSize).height(this.iconSize)
        .colorFilter(this.tintColorFilter())
        .interpolation(ImageInterpolation.High)
    } else if (iconName === 'circle-check') {
      Image($rawfile('icons/circle-check.svg'))
        .width(this.iconSize).height(this.iconSize)
        .colorFilter(this.tintColorFilter())
        .interpolation(ImageInterpolation.High)
    } else if (iconName === 'search') {
      Image($rawfile('icons/search.svg'))
        .width(this.iconSize).height(this.iconSize)
        .colorFilter(this.tintColorFilter())
        .interpolation(ImageInterpolation.High)
    } else if (iconName === 'send') {
      Image($rawfile('icons/send.svg'))
        .width(this.iconSize).height(this.iconSize)
        .colorFilter(this.tintColorFilter())
        .interpolation(ImageInterpolation.High)
    } else if (iconName === 'share-2') {
      Image($rawfile('icons/share-2.svg'))
        .width(this.iconSize).height(this.iconSize)
        .colorFilter(this.tintColorFilter())
        .interpolation(ImageInterpolation.High)
    } else if (iconName === 'pencil') {
      Image($rawfile('icons/pencil.svg'))
        .width(this.iconSize).height(this.iconSize)
        .colorFilter(this.tintColorFilter())
        .interpolation(ImageInterpolation.High)
    } else if (iconName === 'trash-2') {
      Image($rawfile('icons/trash-2.svg'))
        .width(this.iconSize).height(this.iconSize)
        .colorFilter(this.tintColorFilter())
        .interpolation(ImageInterpolation.High)
    } else if (iconName === 'camera') {
      Image($rawfile('icons/camera.svg'))
        .width(this.iconSize).height(this.iconSize)
        .colorFilter(this.tintColorFilter())
        .interpolation(ImageInterpolation.High)
    } else if (iconName === 'image') {
      Image($rawfile('icons/image.svg'))
        .width(this.iconSize).height(this.iconSize)
        .colorFilter(this.tintColorFilter())
        .interpolation(ImageInterpolation.High)
    } else if (iconName === 'heart') {
      Image($rawfile('icons/heart.svg'))
        .width(this.iconSize).height(this.iconSize)
        .colorFilter(this.tintColorFilter())
        .interpolation(ImageInterpolation.High)
    } else if (iconName === 'star') {
      Image($rawfile('icons/star.svg'))
        .width(this.iconSize).height(this.iconSize)
        .colorFilter(this.tintColorFilter())
        .interpolation(ImageInterpolation.High)
    } else if (iconName === 'lock') {
      Image($rawfile('icons/lock.svg'))
        .width(this.iconSize).height(this.iconSize)
        .colorFilter(this.tintColorFilter())
        .interpolation(ImageInterpolation.High)
    } else if (iconName === 'info') {
      Image($rawfile('icons/info.svg'))
        .width(this.iconSize).height(this.iconSize)
        .colorFilter(this.tintColorFilter())
        .interpolation(ImageInterpolation.High)
    } else if (iconName === 'shield-alert') {
      Image($rawfile('icons/shield-alert.svg'))
        .width(this.iconSize).height(this.iconSize)
        .colorFilter(this.tintColorFilter())
        .interpolation(ImageInterpolation.High)
    } else if (iconName === 'circle-x') {
      Image($rawfile('icons/circle-x.svg'))
        .width(this.iconSize).height(this.iconSize)
        .colorFilter(this.tintColorFilter())
        .interpolation(ImageInterpolation.High)
    } else if (iconName === 'inbox') {
      Image($rawfile('icons/inbox.svg'))
        .width(this.iconSize).height(this.iconSize)
        .colorFilter(this.tintColorFilter())
        .interpolation(ImageInterpolation.High)
    } else if (iconName === 'clock') {
      Image($rawfile('icons/clock.svg'))
        .width(this.iconSize).height(this.iconSize)
        .colorFilter(this.tintColorFilter())
        .interpolation(ImageInterpolation.High)
    } else if (iconName === 'minus') {
      Image($rawfile('icons/minus.svg'))
        .width(this.iconSize).height(this.iconSize)
        .colorFilter(this.tintColorFilter())
        .interpolation(ImageInterpolation.High)
    } else if (iconName === 'check-check') {
      Image($rawfile('icons/check-check.svg'))
        .width(this.iconSize).height(this.iconSize)
        .colorFilter(this.tintColorFilter())
        .interpolation(ImageInterpolation.High)
    } else if (iconName === 'filter') {
      Image($rawfile('icons/filter.svg'))
        .width(this.iconSize).height(this.iconSize)
        .colorFilter(this.tintColorFilter())
        .interpolation(ImageInterpolation.High)
    } else if (iconName === 'rotate-cw') {
      Image($rawfile('icons/rotate-cw.svg'))
        .width(this.iconSize).height(this.iconSize)
        .colorFilter(this.tintColorFilter())
        .interpolation(ImageInterpolation.High)
    } else if (iconName === 'circle-help') {
      Image($rawfile('icons/circle-help.svg'))
        .width(this.iconSize).height(this.iconSize)
        .colorFilter(this.tintColorFilter())
        .interpolation(ImageInterpolation.High)
    } else if (iconName === 'shield') {
      Image($rawfile('icons/shield.svg'))
        .width(this.iconSize).height(this.iconSize)
        .colorFilter(this.tintColorFilter())
        .interpolation(ImageInterpolation.High)
    } else if (iconName === 'chart-column') {
      Image($rawfile('icons/chart-column.svg'))
        .width(this.iconSize).height(this.iconSize)
        .colorFilter(this.tintColorFilter())
        .interpolation(ImageInterpolation.High)
    } else if (iconName === 'code') {
      Image($rawfile('icons/code.svg'))
        .width(this.iconSize).height(this.iconSize)
        .colorFilter(this.tintColorFilter())
        .interpolation(ImageInterpolation.High)
    } else if (iconName === 'palette') {
      Image($rawfile('icons/palette.svg'))
        .width(this.iconSize).height(this.iconSize)
        .colorFilter(this.tintColorFilter())
        .interpolation(ImageInterpolation.High)
    } else if (iconName === 'sparkles') {
      Image($rawfile('icons/sparkles.svg'))
        .width(this.iconSize).height(this.iconSize)
        .colorFilter(this.tintColorFilter())
        .interpolation(ImageInterpolation.High)
    } else if (iconName === 'moon') {
      Image($rawfile('icons/moon.svg'))
        .width(this.iconSize).height(this.iconSize)
        .colorFilter(this.tintColorFilter())
        .interpolation(ImageInterpolation.High)
    } else if (iconName === 'menu') {
      Image($rawfile('icons/menu.svg'))
        .width(this.iconSize).height(this.iconSize)
        .colorFilter(this.tintColorFilter())
        .interpolation(ImageInterpolation.High)
    } else if (iconName === 'eye') {
      Image($rawfile('icons/eye.svg'))
        .width(this.iconSize).height(this.iconSize)
        .colorFilter(this.tintColorFilter())
        .interpolation(ImageInterpolation.High)
    } else if (iconName === 'eye-off') {
      Image($rawfile('icons/eye-off.svg'))
        .width(this.iconSize).height(this.iconSize)
        .colorFilter(this.tintColorFilter())
        .interpolation(ImageInterpolation.High)
    } else if (iconName === 'user-plus') {
      Image($rawfile('icons/user-plus.svg'))
        .width(this.iconSize).height(this.iconSize)
        .colorFilter(this.tintColorFilter())
        .interpolation(ImageInterpolation.High)
    } else if (iconName === 'user-check') {
      Image($rawfile('icons/user-check.svg'))
        .width(this.iconSize).height(this.iconSize)
        .colorFilter(this.tintColorFilter())
        .interpolation(ImageInterpolation.High)
    } else if (iconName === 'sliders-horizontal') {
      Image($rawfile('icons/sliders-horizontal.svg'))
        .width(this.iconSize).height(this.iconSize)
        .colorFilter(this.tintColorFilter())
        .interpolation(ImageInterpolation.High)
    }
  }

  /** AppIcon → lucide 资源名；未映射返回空串（走 SymbolSpan fallback） */
  private lucideName(icon: AppIcon): string {
    switch (icon) {
      case AppIcon.HOME: return 'house';
      case AppIcon.COMPASS: return 'map-pin';
      case AppIcon.MESSAGE: return 'message-circle';
      case AppIcon.PERSON: return 'circle-user';
      case AppIcon.GROUP: return 'users';
      case AppIcon.NEARBY: return 'map-pin';
      case AppIcon.LOCATION: return 'map-pin';
      case AppIcon.MAP: return 'map';
      case AppIcon.NOTIFICATION: return 'bell';
      case AppIcon.NOTIFICATION_FILL: return 'bell';
      case AppIcon.SETTINGS: return 'settings';
      case AppIcon.ADD: return 'plus';
      case AppIcon.CLOSE: return 'x';
      case AppIcon.CANCEL: return 'x';
      case AppIcon.BACK: return 'chevron-left';
      case AppIcon.ARROW_LEFT: return 'chevron-left';
      case AppIcon.FORWARD: return 'chevron-right';
      case AppIcon.ARROW_RIGHT: return 'chevron-right';
      case AppIcon.DOWN: return 'chevron-down';
      case AppIcon.ARROW_DOWN: return 'chevron-down';
      case AppIcon.UP: return 'chevron-up';
      case AppIcon.ARROW_UP: return 'chevron-up';
      case AppIcon.MORE: return 'ellipsis';
      case AppIcon.CHECK: return 'check';
      case AppIcon.SAVE: return 'check';
      case AppIcon.CONFIRM: return 'circle-check';
      case AppIcon.SEARCH: return 'search';
      case AppIcon.SEND: return 'send';
      case AppIcon.SHARE: return 'share-2';
      case AppIcon.EDIT: return 'pencil';
      case AppIcon.DELETE: return 'trash-2';
      case AppIcon.CLEAN: return 'trash-2';
      case AppIcon.CLEAR: return 'trash-2';
      case AppIcon.CAMERA: return 'camera';
      case AppIcon.PHOTO: return 'image';
      case AppIcon.LIKE: return 'heart';
      case AppIcon.HEART: return 'heart';
      case AppIcon.HEART_FILL: return 'heart';
      case AppIcon.STAR: return 'star';
      case AppIcon.STAR_FILL: return 'star';
      case AppIcon.LOCK: return 'lock';
      case AppIcon.INFO: return 'info';
      case AppIcon.WARNING: return 'shield-alert';
      case AppIcon.ERROR: return 'circle-x';
      case AppIcon.EMPTY: return 'inbox';
      case AppIcon.TIME: return 'clock';
      case AppIcon.REMOVE: return 'minus';
      case AppIcon.CHECK_ALL: return 'check-check';
      case AppIcon.FILTER: return 'filter';
      case AppIcon.REFRESH: return 'rotate-cw';
      case AppIcon.QUESTION: return 'circle-help';
      case AppIcon.SHIELD: return 'shield';
      case AppIcon.CHART: return 'chart-column';
      case AppIcon.CODE: return 'code';
      case AppIcon.PALETTE: return 'palette';
      case AppIcon.RECOMMEND: return 'sparkles';
      case AppIcon.MOON: return 'moon';
      default: return '';
    }
  }

  @Builder
  private renderForIcon(icon: AppIcon) {
    if (this.lucideName(icon).length > 0) {
      this.renderLucide(this.lucideName(icon))
    } else {
      // Fallback：未列入映射时降级到 SymbolSpan
      Text() {
        SymbolSpan($r(icon))
          .fontSize(this.iconSize)
          .fontColor([this.resolveColorHex()])
      }
    }
  }

  build() {
    Stack({ alignContent: Alignment.Center }) {
      this.renderForIcon(this.icon)
    }
    .width(this.iconSize)
    .height(this.iconSize)
  }
}

/**
 * 容器尺寸辅助：与前端 lucide 一致
 *   xs: 12  sm: 16  md: 20  lg: 24  xl: 32
 */
export const LUCIDE_ICON_SIZE: Record<string, number> = {
  'xs': DesignConstants.FONT.SIZE.XS2,
  'sm': 16,
  'md': DesignConstants.FONT.SIZE.LG,
  'lg': 24,
  'xl': 32
};
