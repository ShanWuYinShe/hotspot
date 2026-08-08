/**
 * HarmonyOS 应用设计常量
 * 统一管理字体、间距、圆角等设计相关常量；
 * 业务色一律走 AppThemeManager palette（vibrant / forest，对齐前端 --cst-* token），
 * 本 COLOR 仅保留主题无关的系统/图片固定色。
 */

// ==================== 颜色系统接口（仅主题无关固定色）====================
export interface ColorPalette {
  /** 图片全屏预览固定反色文字（暗底上的浅色文字） */
  TEXT_PRIMARY_DARK: string;
  /** 图片遮罩/全屏预览固定遮罩色 */
  OVERLAY_LIGHT: string;
  OVERLAY_STRONG: string;
  /** 图片底部渐变起始（极淡黑） */
  IMAGE_OVERLAY_START: string;
  /** 图片底部渐变结束（深黑，承载正文） */
  IMAGE_OVERLAY_END: string;
  /** 图片角标背景（深蓝黑半透明） */
  IMAGE_BADGE_BG: string;
  /** 图片角标描边（半透明白） */
  IMAGE_BADGE_BORDER: string;
  /** 图片上传遮罩描边（浅色模式） */
  OVERLAY_BORDER_LIGHT: string;
  /** 图片上传遮罩描边（深色模式） */
  OVERLAY_BORDER_DARK: string;
  // 系统常量色（所有主题一致，替代 Color.White/Transparent/Black）
  WHITE: string;
  TRANSPARENT: string;
  BLACK: string;
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

export interface FontLineHeights {
  XS: number; // 18 — 紧凑辅助文字
  SM: number; // 20 — 正文/卡片摘要
  MD: number; // 22 — 强调文字
  LG: number; // 24 — 标题
  XL: number; // 26 — 大标题
}

export interface FontLetterSpacing {
  EYEBROW: number;  // 1.4 — 分组标签/眉标（对齐前端 tracking-eyebrow 0.14em @10px）
  LABEL: number;    // 1.2 — 小标/在线标签（对齐前端 tracking-label 0.12em）
  FIELD: number;    // 1.28 — 表单字段标签（0.08em × 16px，对齐前端 tracking-field 0.08em）
  ICON_LABEL: number; // 0.6 — 图标下小字
}

export interface FontWeights {
  REGULAR: number;
  MEDIUM: number;
  SEMIBOLD: number;
  BOLD: number;
}

export interface FontSystem {
  FAMILY_PRIMARY: string;
  SIZE: FontSizes;
  WEIGHT: FontWeights;
  LINE_HEIGHT: FontLineHeights;
  LETTER_SPACING: FontLetterSpacing;
}

// ==================== 间距系统接口 ====================
export interface SpacingSystem {
  HALF: number; // 2 — 极小内边距（badge/头像文字内）
  XS: number;
  THREE: number; // 3 — 紧凑徽章/筛选 chip 内边距
  SM2: number; // 6 — 4 与 8 之间的微步（紧凑元素内边距）
  SM: number;
  MD2: number; // 10 — 8 与 12 之间的微步
  MD: number;
  MD3: number; // 14 — 12 与 16 之间的微步（表单字段间距）
  LG: number;
  LG2: number; // 20 — 16 与 24 之间的微步（弹层/分组间距）
  XL: number;
  XXL: number;
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
}

// ==================== 动画系统接口 ====================
export interface AnimationSystem {
  DURATION_SHORT: number;
  DURATION_MEDIUM: number;
  /** loading → content 交叉淡入淡出时长 */
  DURATION_CONTENT_SWITCH: number;
  /** bottom-sheet 转场统一时长（全应用 320ms，原散落魔法数收敛） */
  DURATION_SHEET: number;
  /** 面板/筛选过渡（250ms） */
  DURATION_250: number;
  /** 弹层高度过渡（280ms） */
  DURATION_280: number;
  /** 轻量反馈/入场（150ms） */
  DURATION_FAST: number;
  /** 即时反馈（100ms） */
  DURATION_INSTANT: number;
  /** 脉冲/shimmer 循环周期（1500ms） */
  DURATION_SLOW: number;
  /** 列表入场 stagger 延迟 */
  DURATION_STAGGER_DELAY: number;
  CURVE_EASE_IN_OUT: Curve;
  CURVE_EASE_OUT: Curve;
  CURVE_FAST_OUT_SLOW_IN: Curve;
  CURVE_SMOOTH: Curve;
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
  /** 输入行按压缩放 */
  SCALE_PRESSED_INPUT_ROW: number;
  /** 胶囊/标签按压缩放 */
  SCALE_PRESSED_PILL: number;
  /** 返回按钮按压缩放 */
  SCALE_PRESSED_BACK: number;
  /** 卡片 hover 抬升幅度（radius ×2 / offsetY +elevation） */
  HOVER_ELEVATION: number;
  /** 卡片 hover 静止态抬升 */
  HOVER_ELEVATION_REST: number;
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
  /** 空态图标淡化（对齐前端 opacity-40 标准档） */
  FADED: number;
  /** 次级内容淡化（对齐前端 opacity-50） */
  MUTED: number;
  /** 强调内容淡化（对齐前端 opacity-80） */
  EMPHASIZED: number;
}

// ==================== 组件特定常量接口 ====================
export interface AvatarSizes {
  XS: number;
  SMALL: number;
  /** 44 — 通知行/会话列表头像（对齐前端 size-11） */
  SMALL_PLUS: number;
  MEDIUM: number;
  LARGE: number;
  XLARGE: number;
}

export interface ComponentConstants {
  AVATAR_SIZE: AvatarSizes;
  BUTTON_HEIGHT: number;
  INPUT_HEIGHT: number;
  /** 紧凑操作按钮（通知操作/角标按钮 28） */
  ACTION_BUTTON_HEIGHT: number;
  /** 中等操作按钮（详情页操作行 40） */
  ACTION_BUTTON_HEIGHT_MD: number;
  /** 大号操作按钮（胶囊标签组/邀请行 34） */
  ACTION_BUTTON_HEIGHT_LG: number;
  /** sheet 内操作行按钮（44） */
  SHEET_ACTION_HEIGHT: number;
  /** 主按钮大号（48，登录/提交主操作） */
  BUTTON_HEIGHT_LG: number;
  /** 行内按钮（38，邀请操作等） */
  INLINE_BUTTON_HEIGHT: number;
  /** 筛选 chip 高度（30） */
  FILTER_CHIP_HEIGHT: number;
  /** 状态图标尺寸（28，定位/通知 banner） */
  STATUS_ICON_SIZE: number;
  /** 加载指示图标尺寸（20） */
  LOADING_ICON_SIZE: number;
  /** 空态图标容器尺寸（64） */
  EMPTY_ICON_SIZE: number;
  /** 加载容器尺寸（72） */
  LOADING_CONTAINER_SIZE: number;
  /** 品牌/logo 图标尺寸（38） */
  BRAND_ICON_SIZE: number;
  /** 资料页横幅高度（140） */
  BANNER_HEIGHT: number;
  /** 首页群组横滑卡片宽度（140） */
  GROUP_CARD_WIDTH: number;
  /** 通知中心加载空态区域高度（200） */
  LOADING_AREA_HEIGHT: number;
  /** 创建群组人数输入框宽度（64） */
  NUMBER_INPUT_WIDTH: number;
  /** 通用图标尺寸档（16/20/24） */
  ICON_SIZE_XS: number;
  ICON_SIZE_SM: number;
  ICON_SIZE_MD: number;
  /** 骨架屏卡片高度（100） */
  SKELETON_CARD_HEIGHT: number;
  /** 骨架屏头像尺寸（66） */
  SKELETON_AVATAR_SIZE: number;
  /** 内容占位区高度（120，加载/空态占位） */
  PLACEHOLDER_AREA_HEIGHT: number;
  /** 大图占位高度（110，创建页图片区） */
  IMAGE_PLACEHOLDER_HEIGHT: number;
  /** 分享卡高度（130） */
  SHARE_CARD_HEIGHT: number;
  /** 日志区高度（300，调试面板） */
  LOG_AREA_HEIGHT: number;
  /** 加载旋转图标尺寸（46，SystemLoadingView 旋转环） */
  LOADING_GLYPH_SIZE: number;
  /** 密码输入框高度（46） */
  PASSWORD_INPUT_HEIGHT: number;
  /** 退出登录按钮高度（50） */
  LOGOUT_BUTTON_HEIGHT: number;
  /** 主题预览条尺寸（54×16） */
  THEME_PREVIEW_BAR_WIDTH: number;
  THEME_PREVIEW_BAR_HEIGHT: number;
  /** 加载环描边宽度（4） */
  PROGRESS_STROKE_WIDTH: number;
}

// ==================== 聊天主题布局常量接口 ====================
export interface ChatLayoutConstants {
  /** 气泡圆角 */
  BUBBLE_RADIUS: number;
  /** 气泡内边距（水平） */
  BUBBLE_PADDING_H: number;
  /** 气泡内边距（垂直） */
  BUBBLE_PADDING_V: number;
  /** 图片消息气泡内边距（水平，14 — 图片内容紧凑） */
  IMAGE_BUBBLE_PADDING_H: number;
  /** 图片消息气泡内边距（垂直，6） */
  IMAGE_BUBBLE_PADDING_V: number;
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
  /** 图片消息最大宽度（200） */
  IMAGE_MAX_WIDTH: number;
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
  /** 底部 sheet 顶部圆角（对齐前端 radius-sheet 弹层专用大圆角） */
  SHEET_TOP_RADIUS: number;
}

// ==================== 设计常量聚合接口 ====================
export interface MaterialEffectSystem {
  TEXTURE_SOFT: number;
  /** 大号 glow 模糊（80，头像光晕） */
  GLOW_BLUR_LARGE: number;
  /** 中号 glow 模糊（64，头部光晕） */
  GLOW_BLUR_MEDIUM: number;
  /** 小号 glow 模糊（22，头像内光晕） */
  GLOW_BLUR_SMALL: number;
}

// ==================== 页面布局系统接口（对齐前端 navigation chrome）====================
export interface LayoutSystem {
  /** 桌面顶栏高度（对齐前端 --desktop-header-height: 52px） */
  DESKTOP_HEADER_HEIGHT: number;
  /** 桌面常驻侧边栏宽度（对齐前端 --cst-desktop-sidebar-width: 224px） */
  DESKTOP_SIDEBAR_WIDTH: number;
  /** feed 页右栏宽度（对齐前端发现等右栏 220px + border-left） */
  RIGHT_SIDEBAR_WIDTH: number;
  /** 移动端通栏底栏高度（对齐前端 MobileBottomNav） */
  MOBILE_BOTTOM_NAV_HEIGHT: number;
  /** 平板浮动玻璃药丸底栏最大宽度（对齐前端 TabletBottomNav） */
  TABLET_NAV_MAX_WIDTH: number;
  /** 中央发布 FAB 尺寸（对齐前端 52x52） */
  FAB_SIZE: number;
  /** 圆形图标按钮尺寸（对齐前端 36px，移动端可放大到 38px） */
  ICON_BUTTON_SIZE: number;
  /** 统计标签列宽（60，对齐前端明细标签列） */
  LABEL_WIDTH: number;
  /** 首页桌面双栏左区宽度（420） */
  HOME_COLUMN_WIDTH: number;
  /** 地图筛选面板宽度（180） */
  FILTER_PANEL_WIDTH: number;
  /** 个人页 glow 光晕尺寸（180/160/120） */
  GLOW_BLOB_LARGE: number;
  GLOW_BLOB_MEDIUM: number;
  GLOW_BLOB_SMALL: number;
  /** 个人页 glow 光晕右上定位偏移（64） */
  GLOW_BLOB_MARGIN_TOP: number;
}

// ==================== 页面内容栏档位接口（对齐前端 .page-shell / .feed-density-shell）====================
export interface PageShellSystem {
  /** 默认内容栏（feed/卡片列表/网格）— 对齐 .page-shell min(1180px, 100vw-32px) */
  SHELL_DEFAULT_MAX: number;
  /** 宽内容栏（数据表格/管理后台/仪表盘）— 对齐 .page-shell--wide min(1320px, ...) */
  SHELL_WIDE_MAX: number;
  /** 窄内容栏（详情/阅读/长文）— 对齐 .page-shell--narrow min(720px, ...) */
  SHELL_NARROW_MAX: number;
  /** 表单内容栏（设置/个人资料/编辑/搜索/登录）— 对齐 .page-shell--form min(560px, ...) */
  SHELL_FORM_MAX: number;
  /** feed 窄栏（moments 移动·窄平板）— 对齐 .feed-density-shell--narrow 640px */
  FEED_NARROW_MAX: number;
  /** feed 中栏（moments 宽平板）— 对齐 .feed-density-shell--medium 960px */
  FEED_MEDIUM_MAX: number;
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
  LAYOUT: LayoutSystem;
  PAGE_SHELL: PageShellSystem;
}

// ==================== 颜色系统（仅主题无关固定色）====================
const COLOR: ColorPalette = {
  TEXT_PRIMARY_DARK: '#E8F1E9',
  OVERLAY_LIGHT: '#59FFFFFF',
  OVERLAY_STRONG: '#80000000',
  IMAGE_OVERLAY_START: 'rgba(0, 0, 0, 0.04)',
  IMAGE_OVERLAY_END: 'rgba(0, 0, 0, 0.44)',
  IMAGE_BADGE_BG: 'rgba(15, 23, 42, 0.58)',
  IMAGE_BADGE_BORDER: 'rgba(255, 255, 255, 0.24)',
  OVERLAY_BORDER_LIGHT: '#40FFFFFF',
  OVERLAY_BORDER_DARK: '#26000000',
  WHITE: '#FFFFFF',
  TRANSPARENT: '#00000000',
  BLACK: '#000000',
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
    XS_MINUS: 9, // HarmonyOS 渲染适配档（前端最小档为 10，本端保留 9 用于极小角标）
    XS2: 10,
    SM_PLUS: 13, // HarmonyOS 渲染适配档（前端已把 13 归并到 12，本端保留为聊天气泡正文档）
    MD_PLUS: 15,
    LG_PLUS: 18,
    XL_PLUS: 22,
    XXL_PLUS: 28,
    XXXL_PLUS: 36,
    HERO: 48,
    HERO_LARGE: 56,
  },
  WEIGHT: {
    REGULAR: 400,
    MEDIUM: 500,
    SEMIBOLD: 600,
    BOLD: 700,
  },
  LINE_HEIGHT: {
    XS: 18,
    SM: 20,
    MD: 22,
    LG: 24,
    XL: 26,
  },
  LETTER_SPACING: {
    EYEBROW: 1.4,
    LABEL: 1.2,
    FIELD: 1.28,
    ICON_LABEL: 0.6,
  },
};

// ==================== 间距系统（对齐前端 16px 移动端基线，4px 网格 + 微步）====================
// HALF/SM2/MD2 为组件内部紧凑元素的精细内边距，消除散落的硬编码 2/6/10。
const SPACING: SpacingSystem = {
  HALF: 2,
  XS: 4,
  THREE: 3,
  SM2: 6,
  SM: 8,
  MD2: 10,
  MD: 12,
  MD3: 14,
  LG: 16,
  LG2: 20,
  XL: 24,
  XXL: 32,
};

// ==================== 圆角系统（HarmonyOS vp 适配档）====================
// 前端 --radius 已收敛 4 档（sm 12 / md 16 / lg 20 / xl 24，sheet 28）；
// harmonyapp 按 vp 渲染保留 8/12/16/18/22 近似档，语义圆角由 SHAPE 档位对齐前端用法
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
  // 主按钮统一圆角矩形（28vp，HarmonyOS 控制件常规圆角；前端主按钮为 rounded-full
  // 胶囊，本端用圆角矩形与胶囊 chip 区分主次操作）。
  BUTTON: 28,
  CHIP: RADIUS.FULL,
  TAB_BAR: 24,
  PREVIEW_CARD: 22,
  LIST_ITEM: 22,
  ICON_TILE: RADIUS.MD,
  SHEET_TOP_RADIUS: 20,
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
};

// ==================== 动画系统 ====================
const ANIMATION: AnimationSystem = {
  DURATION_SHORT: 200,
  DURATION_MEDIUM: 300,
  DURATION_CONTENT_SWITCH: 400,
  DURATION_SHEET: 320,
  DURATION_250: 250,
  DURATION_280: 280,
  DURATION_FAST: 150,
  DURATION_INSTANT: 100,
  DURATION_SLOW: 1500,
  DURATION_STAGGER_DELAY: 150,
  CURVE_EASE_IN_OUT: Curve.EaseInOut,
  CURVE_EASE_OUT: Curve.EaseOut,
  CURVE_FAST_OUT_SLOW_IN: Curve.FastOutSlowIn,
  CURVE_SMOOTH: Curve.Smooth,
};

// ==================== 交互反馈系统 ====================
const INTERACTION: InteractionSystem = {
  SCALE_PRESSED_CARD: 0.97,
  SCALE_PRESSED_MASONRY: 0.96,
  SCALE_PRESSED_BUTTON: 0.95,
  SCALE_PRESSED_ICON: 0.90,
  SCALE_PRESSED_INPUT_ROW: 0.98,
  SCALE_PRESSED_PILL: 0.94,
  SCALE_PRESSED_BACK: 0.88,
  HOVER_ELEVATION: 4,
  HOVER_ELEVATION_REST: 1,
  SPRING_RESPONSE_PRESS: 0.3,
  SPRING_DAMPING_PRESS: 0.85,
  SPRING_RESPONSE_TAB: 0.4,
  SPRING_DAMPING_TAB: 0.75,
};

// ==================== 组件特定常量 ====================
const EFFECT: MaterialEffectSystem = {
  TEXTURE_SOFT: 0.12,
  GLOW_BLUR_LARGE: 80,
  GLOW_BLUR_MEDIUM: 64,
  GLOW_BLUR_SMALL: 22,
};

const COMPONENT: ComponentConstants = {
  AVATAR_SIZE: {
    XS: 32,
    SMALL: 40,
    SMALL_PLUS: 44,
    MEDIUM: 48,
    LARGE: 80,
    XLARGE: 96,
  },
  BUTTON_HEIGHT: 42,
  INPUT_HEIGHT: 42,
  ACTION_BUTTON_HEIGHT: 28,
  ACTION_BUTTON_HEIGHT_MD: 40,
  ACTION_BUTTON_HEIGHT_LG: 34,
  SHEET_ACTION_HEIGHT: 44,
  BUTTON_HEIGHT_LG: 48,
  INLINE_BUTTON_HEIGHT: 38,
  FILTER_CHIP_HEIGHT: 30,
  STATUS_ICON_SIZE: 28,
  LOADING_ICON_SIZE: 20,
  EMPTY_ICON_SIZE: 64,
  LOADING_CONTAINER_SIZE: 72,
  BRAND_ICON_SIZE: 38,
  BANNER_HEIGHT: 140,
  GROUP_CARD_WIDTH: 140,
  LOADING_AREA_HEIGHT: 200,
  NUMBER_INPUT_WIDTH: 64,
  ICON_SIZE_XS: 16,
  ICON_SIZE_SM: 20,
  ICON_SIZE_MD: 24,
  SKELETON_CARD_HEIGHT: 100,
  SKELETON_AVATAR_SIZE: 66,
  PLACEHOLDER_AREA_HEIGHT: 120,
  IMAGE_PLACEHOLDER_HEIGHT: 110,
  SHARE_CARD_HEIGHT: 130,
  LOG_AREA_HEIGHT: 300,
  LOADING_GLYPH_SIZE: 46,
  PASSWORD_INPUT_HEIGHT: 46,
  LOGOUT_BUTTON_HEIGHT: 50,
  THEME_PREVIEW_BAR_WIDTH: 54,
  THEME_PREVIEW_BAR_HEIGHT: 16,
  PROGRESS_STROKE_WIDTH: 4,
};

const CHAT: ChatLayoutConstants = {
  BUBBLE_RADIUS: 18,
  BUBBLE_PADDING_H: 16,
  BUBBLE_PADDING_V: 10,
  IMAGE_BUBBLE_PADDING_H: 14,
  IMAGE_BUBBLE_PADDING_V: 6,
  BUBBLE_MAX_WIDTH: '75%',
  MESSAGE_GAP: 16,
  BUBBLE_FONT_SIZE: 13,
  INPUT_HEIGHT: 38,
  SEND_BUTTON_HEIGHT: 36,
  AVATAR_SIZE: 40,
  IMAGE_MAX_WIDTH: 200,
};

// ==================== 页面布局系统（对齐前端 navigation chrome）====================
const LAYOUT: LayoutSystem = {
  DESKTOP_HEADER_HEIGHT: 52,
  DESKTOP_SIDEBAR_WIDTH: 224,
  RIGHT_SIDEBAR_WIDTH: 220,
  MOBILE_BOTTOM_NAV_HEIGHT: 56,
  TABLET_NAV_MAX_WIDTH: 520,
  FAB_SIZE: 52,
  ICON_BUTTON_SIZE: 36,
  LABEL_WIDTH: 60,
  HOME_COLUMN_WIDTH: 420,
  FILTER_PANEL_WIDTH: 180,
  GLOW_BLOB_LARGE: 180,
  GLOW_BLOB_MEDIUM: 160,
  GLOW_BLOB_SMALL: 120,
  GLOW_BLOB_MARGIN_TOP: 64,
};

// ==================== 页面内容栏档位（对齐前端 .page-shell / .feed-density-shell）====================
const PAGE_SHELL: PageShellSystem = {
  SHELL_DEFAULT_MAX: 1180,
  SHELL_WIDE_MAX: 1320,
  SHELL_NARROW_MAX: 720,
  SHELL_FORM_MAX: 560,
  FEED_NARROW_MAX: 640,
  FEED_MEDIUM_MAX: 960,
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
  FADED: 0.4,
  MUTED: 0.5,
  EMPHASIZED: 0.8,
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
  LAYOUT,
  PAGE_SHELL,
};
