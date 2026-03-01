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

// ==================== 颜色系统 ====================
const COLOR: ColorPalette = {
  PRIMARY_BG_LIGHT: '#F5F6F8',
  PRIMARY_BG_DARK: '#0F1115',
  SECONDARY_BG_LIGHT: '#FFFFFF',
  SECONDARY_BG_DARK: '#171A21',
  TERTIARY_BG_LIGHT: '#EEF1F5',
  TERTIARY_BG_DARK: '#202531',
  TEXT_PRIMARY_LIGHT: '#1C1C1E',
  TEXT_PRIMARY_DARK: '#FFFFFF',
  TEXT_SECONDARY_LIGHT: '#6A7280',
  TEXT_SECONDARY_DARK: '#B7BECA',
  TEXT_TERTIARY_LIGHT: '#9AA3B4',
  TEXT_TERTIARY_DARK: '#8D95A3',
  BRAND_PRIMARY: '#7D9FF7',
  BRAND_PRIMARY_LIGHT: '#7D9FF7',
  BRAND_PRIMARY_DARK: '#8AA5FF',
  SUCCESS: '#36B37E',
  WARNING: '#F4B400',
  ERROR: '#E14D4D',
  INFO: '#7D9FF7',
  DIVIDER_LIGHT: '#E2E5EA',
  DIVIDER_DARK: '#30384A',
  ONLINE_STATUS: '#34C759',
  OFFLINE_STATUS: '#A3A8B2',
  SURFACE_NEUTRAL_SOFT: '#FFF9FAFB',
  SURFACE_INFO_SOFT: '#FFECF2FF',
  SURFACE_SUCCESS_SOFT: '#FFEFF9F4',
  SURFACE_WARNING_SOFT: '#FFFFF8E9',
  SURFACE_DANGER_SOFT: '#FFFFF0F0',
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
  MD: 12,
  LG: 16,
  XL: 24,
  XXL: 32,
  XXXL: 48,
};

// ==================== 圆角系统 ====================
const RADIUS: RadiusSystem = {
  SM: 8,
  MD: 12,
  LG: 16,
  XL: 20,
  XXL: 26,
  FULL: 999,
};

// ==================== 阴影系统 ====================
const SHADOW: ShadowSystem = {
  SMALL: {
    radius: 6,
    color: '#14000000',
    offsetX: 0,
    offsetY: 2,
  },
  MEDIUM: {
    radius: 10,
    color: '#16000000',
    offsetX: 0,
    offsetY: 3,
  },
  LARGE: {
    radius: 14,
    color: '#18000000',
    offsetX: 0,
    offsetY: 5,
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
