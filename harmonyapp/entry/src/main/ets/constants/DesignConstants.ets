/**
 * HarmonyOS 应用设计常量
 * 统一管理颜色、字体、间距、圆角等设计相关常量，以实现高级感和一致性
 * 遵循 Apple Design Guidelines 和 HarmonyOS Design 规范
 */

export const DesignConstants = {
  // ==================== 颜色系统 (Apple-like Palette) ====================
  // 参考 Apple Human Interface Guidelines 中的颜色使用
  COLOR: {
    // 基础灰度
    PRIMARY_BG_LIGHT: '#F5F5F7', // 浅色模式主背景 (Apple's Light Gray)
    PRIMARY_BG_DARK: '#000000', // 深色模式主背景 (Apple's Black)
    SECONDARY_BG_LIGHT: '#FFFFFF', // 浅色模式次背景 (Cards, Modals)
    SECONDARY_BG_DARK: '#1C1C1E', // 深色模式次背景 (Cards, Modals)
    TERTIARY_BG_LIGHT: '#E5E5EA', // 浅色模式三级背景 (Dividers, subtle elements)
    TERTIARY_BG_DARK: '#2C2C2E', // 深色模式三级背景

    TEXT_PRIMARY_LIGHT: '#1C1C1E', // 浅色模式主文本
    TEXT_PRIMARY_DARK: '#FFFFFF', // 深色模式主文本
    TEXT_SECONDARY_LIGHT: '#636366', // 浅色模式次文本
    TEXT_SECONDARY_DARK: '#AEAEB2', // 深色模式次文本
    TEXT_TERTIARY_LIGHT: '#8E8E93', // 浅色模式三级文本
    TEXT_TERTIARY_DARK: '#636366', // 深色模式三级文本

    // 品牌色 (接近 Apple Blue)
    BRAND_PRIMARY: '#007AFF',
    BRAND_PRIMARY_LIGHT: '#0A84FF',
    BRAND_PRIMARY_DARK: '#0A84FF',

    // 状态色
    SUCCESS: '#34C759', // 绿色
    WARNING: '#FFCC00', // 黄色
    ERROR: '#FF3B30', // 红色
    INFO: '#007AFF', // 蓝色

    // 分隔线
    DIVIDER_LIGHT: '#E5E5EA',
    DIVIDER_DARK: '#3A3A3C',

    // 强调色 (用于特殊元素，如在线状态)
    ONLINE_STATUS: '#34C759', // 绿色
    OFFLINE_STATUS: '#AEAEB2', // 灰色
  },

  // ==================== 字体系统 ====================
  FONT: {
    FAMILY_PRIMARY: 'sans-serif', // HarmonyOS 默认字体或 SF Pro Text
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
  },

  // ==================== 间距系统 ====================
  SPACING: {
    XS: 4,
    SM: 8,
    MD: 12,
    LG: 16,
    XL: 24,
    XXL: 32,
    XXXL: 48,
  },

  // ==================== 圆角系统 ====================
  RADIUS: {
    SM: 4,
    MD: 8,
    LG: 12,
    XL: 16, // 用于卡片、头像等主要元素
    XXL: 24,
    FULL: 999, // 用于圆形元素
  },

  // ==================== 阴影系统 ====================
  SHADOW: {
    // 模拟 iOS 的轻微阴影效果
    SMALL: {
      radius: 4,
      color: '#0000001A', // 10% black
      offsetX: 0,
      offsetY: 2,
    },
    MEDIUM: {
      radius: 8,
      color: '#0000001A', // 10% black
      offsetX: 0,
      offsetY: 4,
    },
    LARGE: {
      radius: 12,
      color: '#0000001A', // 10% black
      offsetX: 0,
      offsetY: 6,
    },
  },

  // ==================== 动画系统 ====================
  ANIMATION: {
    DURATION_SHORT: 150, // ms
    DURATION_MEDIUM: 300, // ms
    DURATION_LONG: 500, // ms
    CURVE_EASE_IN_OUT: Curve.EaseInOut,
  },

  // ==================== 组件特定常量 ====================
  COMPONENT: {
    AVATAR_SIZE: {
      SMALL: 32,
      MEDIUM: 48,
      LARGE: 64,
      XLARGE: 80,
    },
    BUTTON_HEIGHT: 44,
    INPUT_HEIGHT: 44,
  },
};
