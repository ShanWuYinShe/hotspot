/**
 * HarmonyOS 应用设计常量
 * 统一管理颜色、字体、间距、圆角等设计相关常量
 * 对齐前端 SvelteKit 项目的 forest 设计系统
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
  CHIP_ACTIVE_BG: string;
  CHIP_ACTIVE_TEXT: string;
  CHIP_INACTIVE_TEXT: string;
  HEADER_ICON_BG: string;
  HEADER_ICON_BORDER: string;
  HEADER_ICON_TEXT: string;
  CHAT_TITLE_ICON_BG: string;
  OVERLAY_LIGHT: string;
  OVERLAY_STRONG: string;
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
  // 扩展值（覆盖高频实际使用的字号）
  XS_MINUS: number;
  XS2: number;
  SM_PLUS: number;
  MD_PLUS: number;
  LG_PLUS: number;
  XL_PLUS: number;
  XXL_PLUS: number;
  XXXL_PLUS: number;
  HERO: number;
  HERO_LARGE: number;
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
  CURVE_EMPHASIZED: Curve;
}

// ==================== 交互反馈系统接口 ====================
export interface InteractionSystem {
  /** 卡片按压缩放 */
  SCALE_PRESSED_CARD: number;
  /** 瀑布流卡片按压缩放 */
  SCALE_PRESSED_MASONRY: number;
  /** 按钮按压缩放 */
  SCALE_PRESSED_BUTTON: number;
  /** 图标按钮按压缩放 */
  SCALE_PRESSED_ICON: number;
  /** 弹簧动画 response（按压） */
  SPRING_RESPONSE_PRESS: number;
  /** 弹簧动画 dampingFraction（按压） */
  SPRING_DAMPING_PRESS: number;
  /** 弹簧动画 response（标签切换） */
  SPRING_RESPONSE_TAB: number;
  /** 弹簧动画 dampingFraction（标签切换） */
  SPRING_DAMPING_TAB: number;
}

// ==================== 透明度系统接口 ====================
export interface OpacitySystem {
  /** 禁用态 */
  DISABLED: number;
  /** 按压/不可操作态 */
  PRESSED: number;
  /** 加载/pending 态 */
  LOADING: number;
  /** 轻微淡化 */
  SUBTLE: number;
}

// ==================== 地图标记颜色接口 ====================
export interface MapMarkerColors {
  PICKER: string;
  USER: string;
  GROUP: string;
  MOMENT: string;
  DEFAULT: string;
}

// ==================== 组件特定常量接口 ====================
export interface AvatarSizes {
  XS: number;
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

// ==================== 聊天主题布局常量接口 ====================
export interface ChatLayoutConstants {
  /** 气泡圆角 */
  BUBBLE_RADIUS: number;
  /** 气泡内边距（水平） */
  BUBBLE_PADDING_H: number;
  /** 气泡内边距（垂直） */
  BUBBLE_PADDING_V: number;
  /** 气泡最大宽度占比 */
  BUBBLE_MAX_WIDTH: string;
  /** 消息间距 */
  MESSAGE_GAP: number;
  /** 气泡文字大小 */
  BUBBLE_FONT_SIZE: number;
  /** 输入框高度 */
  INPUT_HEIGHT: number;
  /** 发送按钮高度 */
  SEND_BUTTON_HEIGHT: number;
  /** 头像尺寸 */
  AVATAR_SIZE: number;
}

// ==================== 语义化形状 token ====================
export interface ShapeSemanticTokens {
  CARD: number;
  HERO_CARD: number;
  COMPACT_CARD: number;
  INPUT: number;
  BUTTON: number;
  CHIP: number;
  TAB_BAR: number;
  PREVIEW_CARD: number;
  LIST_ITEM: number;
  ICON_TILE: number;
}

// ==================== 设计常量聚合接口 ====================
export interface MaterialEffectSystem {
  BLUR_LIGHT: number;
  BLUR_HEAVY: number;
  TEXTURE_SOFT: number;
  TEXTURE_STRONG: number;
  GLOW_SOFT: number;
  GLOW_STRONG: number;
}

export interface DesignConstantsType {
  COLOR: ColorPalette;
  FONT: FontSystem;
  SPACING: SpacingSystem;
  RADIUS: RadiusSystem;
  SHAPE: ShapeSemanticTokens;
  SHADOW: ShadowSystem;
  ANIMATION: AnimationSystem;
  INTERACTION: InteractionSystem;
  EFFECT: MaterialEffectSystem;
  COMPONENT: ComponentConstants;
  CHAT: ChatLayoutConstants;
  OPACITY: OpacitySystem;
  MAP_MARKER: MapMarkerColors;
}

// ==================== 颜色系统（forest 扁平风格，对齐前端设计系统）====================
const COLOR: ColorPalette = {
  PRIMARY_BG_LIGHT: '#EDECEA',
  PRIMARY_BG_DARK: '#0E110E',
  SECONDARY_BG_LIGHT: '#F9F9F5',
  SECONDARY_BG_DARK: '#161A16',
  TERTIARY_BG_LIGHT: '#F0F1F0',
  TERTIARY_BG_DARK: '#242624',
  TEXT_PRIMARY_LIGHT: '#1B1F1B',
  TEXT_PRIMARY_DARK: '#DCE8DE',
  TEXT_SECONDARY_LIGHT: '#566B5E',
  TEXT_SECONDARY_DARK: '#7A9080',
  TEXT_TERTIARY_LIGHT: '#AAB0AF',
  TEXT_TERTIARY_DARK: '#5A6B60',
  BRAND_PRIMARY: '#3D7A55',
  BRAND_PRIMARY_LIGHT: '#3D7A55',
  BRAND_PRIMARY_DARK: '#52B878',
  SUCCESS: '#059669',
  WARNING: '#D97706',
  ERROR: '#EF4444',
  INFO: '#3D7A55',
  DIVIDER_LIGHT: '#00000014',
  DIVIDER_DARK: '#FFFFFF12',
  ONLINE_STATUS: '#3D7A55',
  OFFLINE_STATUS: '#9CA3AF',
  SURFACE_NEUTRAL_SOFT: '#F0F1F0',
  SURFACE_INFO_SOFT: '#E8F0E9',
  SURFACE_SUCCESS_SOFT: '#F0FDF4',
  SURFACE_WARNING_SOFT: '#FFFBEB',
  SURFACE_DANGER_SOFT: '#FEF2F2',
  CHIP_ACTIVE_BG: '#3D7A55',
  CHIP_ACTIVE_TEXT: '#FFFFFF',
  CHIP_INACTIVE_TEXT: '#566B5E',
  HEADER_ICON_BG: '#E8F0E9',
  HEADER_ICON_BORDER: '#3D7A5521',
  HEADER_ICON_TEXT: '#3D7A55',
  CHAT_TITLE_ICON_BG: '#F9F9F5',
  OVERLAY_LIGHT: '#59FFFFFF',
  OVERLAY_STRONG: '#80000000',
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
    XS_MINUS: 9,
    XS2: 10,
    SM_PLUS: 13,
    MD_PLUS: 15,
    LG_PLUS: 18,
    XL_PLUS: 22,
    XXL_PLUS: 28,
    XXXL_PLUS: 36,
    HERO: 48,
    HERO_LARGE: 56,
  },
  WEIGHT: {
    LIGHT: 300,
    REGULAR: 400,
    MEDIUM: 500,
    SEMIBOLD: 600,
    BOLD: 700,
  },
};

// ==================== 间距系统（对齐前端 16px 移动端基线，4px 网格）====================
const SPACING: SpacingSystem = {
  XS: 4,
  SM: 8,
  MD: 12,
  LG: 16,
  XL: 24,
  XXL: 32,
  XXXL: 48,
};

// ==================== 圆角系统（对齐前端 --radius: 0.625rem 基线）====================
const RADIUS: RadiusSystem = {
  SM: 8,
  MD: 12,
  LG: 16,
  XL: 18,
  XXL: 22,
  FULL: 999,
};

const SHAPE: ShapeSemanticTokens = {
  CARD: 22,
  HERO_CARD: 24,
  COMPACT_CARD: 18,
  INPUT: 18,
  // 主按钮统一圆角矩形（28vp），对齐前端 SvelteKit Button base `rounded-[28px]`；
  // 与 chip/头像的 RADIUS.FULL 胶囊区分，主操作走圆角矩形。
  BUTTON: 28,
  CHIP: RADIUS.FULL,
  TAB_BAR: 24,
  PREVIEW_CARD: 22,
  LIST_ITEM: 22,
  ICON_TILE: RADIUS.MD,
};

// ==================== 阴影系统（扁平风格，默认归零）====================
const SHADOW: ShadowSystem = {
  SMALL: {
    radius: 0,
    color: '#00000000',
    offsetX: 0,
    offsetY: 0,
  },
  MEDIUM: {
    radius: 0,
    color: '#00000000',
    offsetX: 0,
    offsetY: 0,
  },
  LARGE: {
    radius: 0,
    color: '#00000000',
    offsetX: 0,
    offsetY: 0,
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
  CURVE_EMPHASIZED: Curve.FastOutSlowIn,
};

// ==================== 交互反馈系统 ====================
const INTERACTION: InteractionSystem = {
  SCALE_PRESSED_CARD: 0.97,
  SCALE_PRESSED_MASONRY: 0.96,
  SCALE_PRESSED_BUTTON: 0.95,
  SCALE_PRESSED_ICON: 0.90,
  SPRING_RESPONSE_PRESS: 0.3,
  SPRING_DAMPING_PRESS: 0.85,
  SPRING_RESPONSE_TAB: 0.4,
  SPRING_DAMPING_TAB: 0.75,
};

// ==================== 组件特定常量 ====================
const EFFECT: MaterialEffectSystem = {
  BLUR_LIGHT: 12,
  BLUR_HEAVY: 24,
  TEXTURE_SOFT: 0.12,
  TEXTURE_STRONG: 0.22,
  GLOW_SOFT: 16,
  GLOW_STRONG: 32,
};

const COMPONENT: ComponentConstants = {
  AVATAR_SIZE: {
    XS: 32,
    SMALL: 40,
    MEDIUM: 48,
    LARGE: 80,
    XLARGE: 96,
  },
  BUTTON_HEIGHT: 42,
  INPUT_HEIGHT: 42,
};

const CHAT: ChatLayoutConstants = {
  BUBBLE_RADIUS: 18,
  BUBBLE_PADDING_H: 16,
  BUBBLE_PADDING_V: 10,
  BUBBLE_MAX_WIDTH: '75%',
  MESSAGE_GAP: 16,
  BUBBLE_FONT_SIZE: 13,
  INPUT_HEIGHT: 38,
  SEND_BUTTON_HEIGHT: 36,
  AVATAR_SIZE: 40,
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

  // 卡片圆角(对齐前端瀑布流卡片 --radius-lg:22)
  static readonly CARD_RADIUS: number = 22;

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

// ==================== 透明度系统 ====================
const OPACITY: OpacitySystem = {
  DISABLED: 0.38,
  PRESSED: 0.72,
  LOADING: 0.58,
  SUBTLE: 0.85,
};

// ==================== 地图标记颜色（对齐前端 --map-* tokens）====================
const MAP_MARKER: MapMarkerColors = {
  PICKER: '#3D7A55',
  USER: '#3D7A55',
  GROUP: '#638B46',
  MOMENT: '#A57C38',
  DEFAULT: '#566B5E',
};

export const DesignConstants: DesignConstantsType = {
  COLOR,
  FONT,
  SPACING,
  RADIUS,
  SHAPE,
  SHADOW,
  ANIMATION,
  INTERACTION,
  EFFECT,
  COMPONENT,
  CHAT,
  OPACITY,
  MAP_MARKER,
};
