/**
 * HarmonyOS 应用设计常量
 * 统一管理颜色、字体、间距、圆角等设计相关常量，以实现高级感和一致性
 * 遵循 Apple Design Guidelines 和 HarmonyOS Design 规范
 */

// ==================== 颜色系统接口 ====================
export interface ColorPalette {
  PRIMARY_BG_LIGHT: string;
  PRIMARY_BG_DARK: string;
  SECONDARY_BG_LIGHT: string;
  SECONDARY_BG_DARK: string;
  TERTIARY_BG_LIGHT: string;
  TERTIARY_BG_DARK: string;
  TEXT_PRIMARY_LIGHT: string;
  TEXT_PRIMARY_DARK: string;
  TEXT_SECONDARY_LIGHT: string;
  TEXT_SECONDARY_DARK: string;
  TEXT_TERTIARY_LIGHT: string;
  TEXT_TERTIARY_DARK: string;
  BRAND_PRIMARY: string;
  BRAND_PRIMARY_LIGHT: string;
  BRAND_PRIMARY_DARK: string;
  SUCCESS: string;
  WARNING: string;
  ERROR: string;
  INFO: string;
  DIVIDER_LIGHT: string;
  DIVIDER_DARK: string;
  ONLINE_STATUS: string;
  OFFLINE_STATUS: string;
  SURFACE_NEUTRAL_SOFT: string;
  SURFACE_INFO_SOFT: string;
  SURFACE_SUCCESS_SOFT: string;
  SURFACE_WARNING_SOFT: string;
  SURFACE_DANGER_SOFT: string;
}

// ==================== 字体系统接口 ====================
export interface FontSizes {
  XS: number;
  SM: number;
  MD: number;
  LG: number;
  XL: number;
  XXL: number;
  XXXL: number;
}

export interface FontWeights {
  LIGHT: number;
  REGULAR: number;
  MEDIUM: number;
  SEMIBOLD: number;
  BOLD: number;
}

export interface FontSystem {
  FAMILY_PRIMARY: string;
  SIZE: FontSizes;
  WEIGHT: FontWeights;
}

// ==================== 间距系统接口 ====================
export interface SpacingSystem {
  XS: number;
  SM: number;
  MD: number;
  LG: number;
  XL: number;
  XXL: number;
  XXXL: number;
}

// ==================== 圆角系统接口 ====================
export interface RadiusSystem {
  SM: number;
  MD: number;
  LG: number;
  XL: number;
  XXL: number;
  FULL: number;
}

// ==================== 阴影系统接口 ====================
export interface ShadowConfig {
  radius: number;
  color: string;
  offsetX: number;
  offsetY: number;
}

export interface ShadowSystem {
  SMALL: ShadowConfig;
  MEDIUM: ShadowConfig;
  LARGE: ShadowConfig;
  INSET: ShadowConfig;
  HIGHLIGHT: ShadowConfig;
}

// ==================== 动画系统接口 ====================
export interface AnimationSystem {
  DURATION_SHORT: number;
  DURATION_MEDIUM: number;
  DURATION_LONG: number;
  /** loading → content 交叉淡入淡出时长 */
  DURATION_CONTENT_SWITCH: number;
  CURVE_EASE_IN_OUT: Curve;
}

// ==================== 组件特定常量接口 ====================
export interface AvatarSizes {
  SMALL: number;
  MEDIUM: number;
  LARGE: number;
  XLARGE: number;
}

export interface ComponentConstants {
  AVATAR_SIZE: AvatarSizes;
  BUTTON_HEIGHT: number;
  INPUT_HEIGHT: number;
}

// ==================== 设计常量聚合接口 ====================
export interface DesignConstantsType {
  COLOR: ColorPalette;
  FONT: FontSystem;
  SPACING: SpacingSystem;
  RADIUS: RadiusSystem;
  SHADOW: ShadowSystem;
  ANIMATION: AnimationSystem;
  COMPONENT: ComponentConstants;
}

// ==================== 颜色系统（干净扁平极简风格）====================
const COLOR: ColorPalette = {
  PRIMARY_BG_LIGHT: '#FFFFFF',
  PRIMARY_BG_DARK: '#0F1115',
  SECONDARY_BG_LIGHT: '#FFFFFF',
  SECONDARY_BG_DARK: '#171A20',
  TERTIARY_BG_LIGHT: '#F3F4F6',
  TERTIARY_BG_DARK: '#222733',
  TEXT_PRIMARY_LIGHT: '#111827',
  TEXT_PRIMARY_DARK: '#F9FAFB',
  TEXT_SECONDARY_LIGHT: '#6B7280',
  TEXT_SECONDARY_DARK: '#9CA3AF',
  TEXT_TERTIARY_LIGHT: '#9CA3AF',
  TEXT_TERTIARY_DARK: '#6B7280',
  BRAND_PRIMARY: '#4A7A57',
  BRAND_PRIMARY_LIGHT: '#4A7A57',
  BRAND_PRIMARY_DARK: '#6BAF86',
  SUCCESS: '#22C55E',
  WARNING: '#FBBF24',
  ERROR: '#EF4444',
  INFO: '#3B82F6',
  DIVIDER_LIGHT: '#F3F4F6',
  DIVIDER_DARK: '#28303D',
  ONLINE_STATUS: '#22C55E',
  OFFLINE_STATUS: '#9CA3AF',
  SURFACE_NEUTRAL_SOFT: '#F9FAFB',
  SURFACE_INFO_SOFT: '#EFF6FF',
  SURFACE_SUCCESS_SOFT: '#F0FDF4',
  SURFACE_WARNING_SOFT: '#FFFBEB',
  SURFACE_DANGER_SOFT: '#FEF2F2',
};

// ==================== 字体系统 ====================
const FONT: FontSystem = {
  FAMILY_PRIMARY: 'sans-serif',
  SIZE: {
    XS: 11,
    SM: 12,
    MD: 14,
    LG: 16,
    XL: 20,
    XXL: 24,
    XXXL: 32,
  },
  WEIGHT: {
    LIGHT: 300,
    REGULAR: 400,
    MEDIUM: 500,
    SEMIBOLD: 600,
    BOLD: 700,
  },
};

// ==================== 间距系统 ====================
const SPACING: SpacingSystem = {
  XS: 4,
  SM: 8,
  MD: 14,
  LG: 18,
  XL: 24,
  XXL: 32,
  XXXL: 48,
};

// ==================== 圆角系统（干净现代风格）====================
const RADIUS: RadiusSystem = {
  SM: 8,
  MD: 12,
  LG: 16,
  XL: 20,
  XXL: 24,
  FULL: 999,
};

// ==================== 阴影系统（干净扁平风格：极轻微阴影）====================
const SHADOW: ShadowSystem = {
  SMALL: {
    radius: 4,
    color: '#0A000000',
    offsetX: 0,
    offsetY: 1,
  },
  MEDIUM: {
    radius: 8,
    color: '#0A000000',
    offsetX: 0,
    offsetY: 2,
  },
  LARGE: {
    radius: 16,
    color: '#0F000000',
    offsetX: 0,
    offsetY: 4,
  },
  INSET: {
    radius: 0,
    color: '#00000000',
    offsetX: 0,
    offsetY: 0,
  },
  HIGHLIGHT: {
    radius: 0,
    color: '#00000000',
    offsetX: 0,
    offsetY: 0,
  },
};

// ==================== 动画系统 ====================
const ANIMATION: AnimationSystem = {
  DURATION_SHORT: 200,
  DURATION_MEDIUM: 300,
  DURATION_LONG: 500,
  DURATION_CONTENT_SWITCH: 400,
  CURVE_EASE_IN_OUT: Curve.EaseInOut,
};

// ==================== 组件特定常量 ====================
const COMPONENT: ComponentConstants = {
  AVATAR_SIZE: {
    SMALL: 32,
    MEDIUM: 48,
    LARGE: 64,
    XLARGE: 80,
  },
  BUTTON_HEIGHT: 42,
  INPUT_HEIGHT: 42,
};

// ==================== 小红书风格布局常量 ====================
export class MasonryLayout {
  // 列宽配置
  static readonly COLUMN_WIDTH_SM: number = 140;  // 小屏设备
  static readonly COLUMN_WIDTH_MD: number = 160;  // 标准尺寸
  static readonly COLUMN_WIDTH_LG: number = 180;  // 大屏设备

  // 间距配置
  static readonly GAP_XS: number = 4;
  static readonly GAP_SM: number = 8;
  static readonly GAP_MD: number = 12;

  // 图片高度限制
  static readonly IMAGE_HEIGHT_MIN: number = 120;
  static readonly IMAGE_HEIGHT_DEFAULT: number = 180;
  static readonly IMAGE_HEIGHT_MAX: number = 280;

  // 卡片圆角
  static readonly CARD_RADIUS: number = 20;

  // 预加载数量
  static readonly CACHED_COUNT: number = 8;
}

// ==================== 瀑布流布局辅助函数 ====================
export class MasonryHelper {
  /**
   * 计算瀑布流图片高度
   * @param originalWidth 原始宽度
   * @param originalHeight 原始高度
   * @param columnWidth 列宽
   * @returns 计算后的高度
   */
  static calculateImageHeight(
    originalWidth: number,
    originalHeight: number,
    columnWidth: number = MasonryLayout.COLUMN_WIDTH_MD
  ): number {
    if (!originalWidth || !originalHeight || originalWidth <= 0) {
      return MasonryLayout.IMAGE_HEIGHT_DEFAULT;
    }

    const aspectRatio = originalHeight / originalWidth;
    const calculatedHeight = columnWidth * aspectRatio;

    return Math.max(
      MasonryLayout.IMAGE_HEIGHT_MIN,
      Math.min(calculatedHeight, MasonryLayout.IMAGE_HEIGHT_MAX)
    );
  }

  /**
   * 获取列模板字符串
   * @param width 列宽
   * @returns columnsTemplate 字符串
   */
  static getColumnsTemplate(width: number = MasonryLayout.COLUMN_WIDTH_MD): string {
    return `repeat(auto-fill, ${width}vp)`;
  }
}

export const DesignConstants: DesignConstantsType = {
  COLOR,
  FONT,
  SPACING,
  RADIUS,
  SHADOW,
  ANIMATION,
  COMPONENT,
};
