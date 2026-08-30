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
 * 资源：resources/base/media/icons/<name>.svg（来自 lucide-static v0.460）
 * 渲染：Image + ColorFilter.tint（line-icon 整图染为目标色）
 * Fallback：未支持的 AppIcon 走 SymbolSpan，调用点不报错
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
 *   - size: 图标边长 px（默认 24）
 *   - color: 直接色值（与 paletteRole 互斥）
 *   - paletteRole: 调色板语义角色
 *
 * 用法：
 *   LucideIcon({ icon: AppIcon.HOME, size: 24, paletteRole: 'accent' })
 *   LucideIcon({ icon: AppIcon.SEARCH, size: 20, color: '#16233A' })
 */
@Component
export struct LucideIcon {
  @Prop icon: AppIcon;
  @Prop size: number = DEFAULT_ICON_SIZE;
  @Prop color: string | ResourceColor = DEFAULT_ICON_COLOR_SENTINEL;
  @Prop paletteRole: string = '';
  /** 子组件（如 ProfileContentAtoms）显式传入主题 ID；为空时走 @StorageProp 全局 */
  @Prop overrideThemeId: string = '';

  @StorageProp(KEY_RESOLVED_THEME_ID) storageThemeId: string = 'vibrant_light';

  private getEffectiveThemeId(): string {
    return this.overrideThemeId !== '' ? this.overrideThemeId : this.storageThemeId;
  }

  private getPalette() {
    return AppThemeManager.getPaletteById(this.getEffectiveThemeId());
  }

  private resolveColor(): ResourceColor {
    if (this.color !== DEFAULT_ICON_COLOR_SENTINEL) {
      return this.color as ResourceColor;
    }
    if (this.paletteRole !== '') {
      const pal = this.getPalette();
      switch (this.paletteRole) {
        case 'accent': return pal.accentBackground as ResourceColor;
        case 'accentText': return pal.accentText as ResourceColor;
        case 'textPrimary': return pal.textPrimary as ResourceColor;
        case 'textSecondary': return pal.textSecondary as ResourceColor;
        case 'success': return pal.success as ResourceColor;
        case 'warning': return pal.warning as ResourceColor;
        case 'error': return pal.error as ResourceColor;
        case 'tabActive': return pal.accentBackground as ResourceColor;
        case 'tabInactive': return pal.tabBarInactive as ResourceColor;
        default: return pal.textSecondary as ResourceColor;
      }
    }
    return this.getPalette().textSecondary as ResourceColor;
  }

  build() {
    Stack({ alignContent: Alignment.Center }) {
      // TabBar / 导航
      if (this.icon === AppIcon.HOME) {
        Image($r('app.media.icons.house'))
          .width(this.size).height(this.size)
          .colorFilter(ColorFilter.tint(this.resolveColor()))
          .interpolation(ImageInterpolation.High)
      } else if (this.icon === AppIcon.COMPASS) {
        Image($r('app.media.icons.map-pin'))
          .width(this.size).height(this.size)
          .colorFilter(ColorFilter.tint(this.resolveColor()))
          .interpolation(ImageInterpolation.High)
      } else if (this.icon === AppIcon.MESSAGE) {
        Image($r('app.media.icons.message-circle'))
          .width(this.size).height(this.size)
          .colorFilter(ColorFilter.tint(this.resolveColor()))
          .interpolation(ImageInterpolation.High)
      } else if (this.icon === AppIcon.PERSON) {
        Image($r('app.media.icons.circle-user'))
          .width(this.size).height(this.size)
          .colorFilter(ColorFilter.tint(this.resolveColor()))
          .interpolation(ImageInterpolation.High)
      } else if (this.icon === AppIcon.GROUP) {
        Image($r('app.media.icons.users'))
          .width(this.size).height(this.size)
          .colorFilter(ColorFilter.tint(this.resolveColor()))
          .interpolation(ImageInterpolation.High)
      } else if (this.icon === AppIcon.NEARBY) {
        Image($r('app.media.icons.map-pin'))
          .width(this.size).height(this.size)
          .colorFilter(ColorFilter.tint(this.resolveColor()))
          .interpolation(ImageInterpolation.High)
      } else if (this.icon === AppIcon.LOCATION) {
        Image($r('app.media.icons.map-pin'))
          .width(this.size).height(this.size)
          .colorFilter(ColorFilter.tint(this.resolveColor()))
          .interpolation(ImageInterpolation.High)
      } else if (this.icon === AppIcon.MAP) {
        Image($r('app.media.icons.map'))
          .width(this.size).height(this.size)
          .colorFilter(ColorFilter.tint(this.resolveColor()))
          .interpolation(ImageInterpolation.High)
      } else if (this.icon === AppIcon.NOTIFICATION) {
        Image($r('app.media.icons.bell'))
          .width(this.size).height(this.size)
          .colorFilter(ColorFilter.tint(this.resolveColor()))
          .interpolation(ImageInterpolation.High)
      } else if (this.icon === AppIcon.SETTINGS) {
        Image($r('app.media.icons.settings'))
          .width(this.size).height(this.size)
          .colorFilter(ColorFilter.tint(this.resolveColor()))
          .interpolation(ImageInterpolation.High)
      // 通用操作
      } else if (this.icon === AppIcon.ADD) {
        Image($r('app.media.icons.plus'))
          .width(this.size).height(this.size)
          .colorFilter(ColorFilter.tint(this.resolveColor()))
          .interpolation(ImageInterpolation.High)
      } else if (this.icon === AppIcon.CLOSE || this.icon === AppIcon.CANCEL) {
        Image($r('app.media.icons.x'))
          .width(this.size).height(this.size)
          .colorFilter(ColorFilter.tint(this.resolveColor()))
          .interpolation(ImageInterpolation.High)
      } else if (this.icon === AppIcon.BACK || this.icon === AppIcon.ARROW_LEFT) {
        Image($r('app.media.icons.chevron-left'))
          .width(this.size).height(this.size)
          .colorFilter(ColorFilter.tint(this.resolveColor()))
          .interpolation(ImageInterpolation.High)
      } else if (this.icon === AppIcon.FORWARD || this.icon === AppIcon.ARROW_RIGHT) {
        Image($r('app.media.icons.chevron-right'))
          .width(this.size).height(this.size)
          .colorFilter(ColorFilter.tint(this.resolveColor()))
          .interpolation(ImageInterpolation.High)
      } else if (this.icon === AppIcon.DOWN || this.icon === AppIcon.ARROW_DOWN) {
        Image($r('app.media.icons.chevron-down'))
          .width(this.size).height(this.size)
          .colorFilter(ColorFilter.tint(this.resolveColor()))
          .interpolation(ImageInterpolation.High)
      } else if (this.icon === AppIcon.UP || this.icon === AppIcon.ARROW_UP) {
        Image($r('app.media.icons.chevron-up'))
          .width(this.size).height(this.size)
          .colorFilter(ColorFilter.tint(this.resolveColor()))
          .interpolation(ImageInterpolation.High)
      } else if (this.icon === AppIcon.MORE) {
        Image($r('app.media.icons.ellipsis'))
          .width(this.size).height(this.size)
          .colorFilter(ColorFilter.tint(this.resolveColor()))
          .interpolation(ImageInterpolation.High)
      } else if (this.icon === AppIcon.CHECK || this.icon === AppIcon.SAVE) {
        Image($r('app.media.icons.check'))
          .width(this.size).height(this.size)
          .colorFilter(ColorFilter.tint(this.resolveColor()))
          .interpolation(ImageInterpolation.High)
      } else if (this.icon === AppIcon.CONFIRM) {
        Image($r('app.media.icons.circle-check'))
          .width(this.size).height(this.size)
          .colorFilter(ColorFilter.tint(this.resolveColor()))
          .interpolation(ImageInterpolation.High)
      } else if (this.icon === AppIcon.SEARCH) {
        Image($r('app.media.icons.search'))
          .width(this.size).height(this.size)
          .colorFilter(ColorFilter.tint(this.resolveColor()))
          .interpolation(ImageInterpolation.High)
      } else if (this.icon === AppIcon.SEND) {
        Image($r('app.media.icons.send'))
          .width(this.size).height(this.size)
          .colorFilter(ColorFilter.tint(this.resolveColor()))
          .interpolation(ImageInterpolation.High)
      } else if (this.icon === AppIcon.SHARE) {
        Image($r('app.media.icons.share-2'))
          .width(this.size).height(this.size)
          .colorFilter(ColorFilter.tint(this.resolveColor()))
          .interpolation(ImageInterpolation.High)
      } else if (this.icon === AppIcon.EDIT) {
        Image($r('app.media.icons.pencil'))
          .width(this.size).height(this.size)
          .colorFilter(ColorFilter.tint(this.resolveColor()))
          .interpolation(ImageInterpolation.High)
      } else if (this.icon === AppIcon.DELETE || this.icon === AppIcon.CLEAN || this.icon === AppIcon.CLEAR) {
        Image($r('app.media.icons.trash-2'))
          .width(this.size).height(this.size)
          .colorFilter(ColorFilter.tint(this.resolveColor()))
          .interpolation(ImageInterpolation.High)
      } else if (this.icon === AppIcon.CAMERA) {
        Image($r('app.media.icons.camera'))
          .width(this.size).height(this.size)
          .colorFilter(ColorFilter.tint(this.resolveColor()))
          .interpolation(ImageInterpolation.High)
      } else if (this.icon === AppIcon.PHOTO) {
        Image($r('app.media.icons.image'))
          .width(this.size).height(this.size)
          .colorFilter(ColorFilter.tint(this.resolveColor()))
          .interpolation(ImageInterpolation.High)
      } else if (this.icon === AppIcon.LIKE || this.icon === AppIcon.HEART) {
        Image($r('app.media.icons.heart'))
          .width(this.size).height(this.size)
          .colorFilter(ColorFilter.tint(this.resolveColor()))
          .interpolation(ImageInterpolation.High)
      } else if (this.icon === AppIcon.STAR) {
        Image($r('app.media.icons.star'))
          .width(this.size).height(this.size)
          .colorFilter(ColorFilter.tint(this.resolveColor()))
          .interpolation(ImageInterpolation.High)
      } else if (this.icon === AppIcon.LOCK) {
        Image($r('app.media.icons.lock'))
          .width(this.size).height(this.size)
          .colorFilter(ColorFilter.tint(this.resolveColor()))
          .interpolation(ImageInterpolation.High)
      } else if (this.icon === AppIcon.INFO) {
        Image($r('app.media.icons.info'))
          .width(this.size).height(this.size)
          .colorFilter(ColorFilter.tint(this.resolveColor()))
          .interpolation(ImageInterpolation.High)
      } else if (this.icon === AppIcon.WARNING) {
        Image($r('app.media.icons.shield-alert'))
          .width(this.size).height(this.size)
          .colorFilter(ColorFilter.tint(this.resolveColor()))
          .interpolation(ImageInterpolation.High)
      } else if (this.icon === AppIcon.RECOMMEND) {
        Image($r('app.media.icons.sparkles'))
          .width(this.size).height(this.size)
          .colorFilter(ColorFilter.tint(this.resolveColor()))
          .interpolation(ImageInterpolation.High)
      } else if (this.icon === AppIcon.MOON) {
        Image($r('app.media.icons.moon'))
          .width(this.size).height(this.size)
          .colorFilter(ColorFilter.tint(this.resolveColor()))
          .interpolation(ImageInterpolation.High)
      } else if (this.icon === AppIcon.SUN) {
        Image($r('app.media.icons.sun'))
          .width(this.size).height(this.size)
          .colorFilter(ColorFilter.tint(this.resolveColor()))
          .interpolation(ImageInterpolation.High)
      } else {
        // Fallback：未列入映射时降级到 SymbolSpan
        Text() {
          SymbolSpan($r(this.icon))
            .fontSize(this.size)
            .fontColor([this.resolveColor()])
        }
      }
    }
    .width(this.size)
    .height(this.size)
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
