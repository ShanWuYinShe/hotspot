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
  PRIMARY_BG_LIGHT: '#F5F5F7',
  PRIMARY_BG_DARK: '#000000',
  SECONDARY_BG_LIGHT: '#FFFFFF',
  SECONDARY_BG_DARK: '#1C1C1E',
  TERTIARY_BG_LIGHT: '#E5E5EA',
  TERTIARY_BG_DARK: '#2C2C2E',
  TEXT_PRIMARY_LIGHT: '#1C1C1E',
  TEXT_PRIMARY_DARK: '#FFFFFF',
  TEXT_SECONDARY_LIGHT: '#636366',
  TEXT_SECONDARY_DARK: '#AEAEB2',
  TEXT_TERTIARY_LIGHT: '#8E8E93',
  TEXT_TERTIARY_DARK: '#636366',
  BRAND_PRIMARY: '#007AFF',
  BRAND_PRIMARY_LIGHT: '#0A84FF',
  BRAND_PRIMARY_DARK: '#0A84FF',
  SUCCESS: '#34C759',
  WARNING: '#FFCC00',
  ERROR: '#FF3B30',
  INFO: '#007AFF',
  DIVIDER_LIGHT: '#E5E5EA',
  DIVIDER_DARK: '#3A3A3C',
  ONLINE_STATUS: '#34C759',
  OFFLINE_STATUS: '#AEAEB2',
};

// ==================== 字体系统 ====================
const FONT: FontSystem = {
  FAMILY_PRIMARY: 'sans-serif',
  SIZE: {
    XS: 10,
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
  SM: 4,
  MD: 8,
  LG: 12,
  XL: 16,
  XXL: 24,
  FULL: 999,
};

// ==================== 阴影系统 ====================
const SHADOW: ShadowSystem = {
  SMALL: {
    radius: 4,
    color: '#0000001A',
    offsetX: 0,
    offsetY: 2,
  },
  MEDIUM: {
    radius: 8,
    color: '#0000001A',
    offsetX: 0,
    offsetY: 4,
  },
  LARGE: {
    radius: 12,
    color: '#0000001A',
    offsetX: 0,
    offsetY: 6,
  },
};

// ==================== 动画系统 ====================
const ANIMATION: AnimationSystem = {
  DURATION_SHORT: 150,
  DURATION_MEDIUM: 300,
  DURATION_LONG: 500,
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
  BUTTON_HEIGHT: 44,
  INPUT_HEIGHT: 44,
};

export const DesignConstants: DesignConstantsType = {
  COLOR,
  FONT,
  SPACING,
  RADIUS,
  SHADOW,
  ANIMATION,
  COMPONENT,
};
