/**
 * HarmonyOS 应用设计常量
 * 统一管理字体、间距、圆角等设计相关常量；
 * 业务色一律走 AppThemeManager palette（vibrant / forest，对齐前端 --cst-* token），
 * 本 COLOR 仅保留主题无关的系统/图片固定色。
 */

// ==================== 颜色系统接口（仅主题无关固定色）====================
export interface ColorPalette {
  /** 图片全屏预览固定反色文字（暗底上的浅色文字） */
  /** 媒体浏览器黑底上的铬层文字（关闭钮/页码），固定色非主题色 */
  MEDIA_CHROME_TEXT: string;
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
  /** 系统窗口默认底色（未接入主题前的暖灰基底，对齐设计稿壁纸色） */
  SYSTEM_SURFACE_BEIGE: string;
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
  XS2: number;
  LG_PLUS: number;
  HERO: number;
}

export interface FontLineHeights {
  XS: number; // 18 — 紧凑辅助文字
  SM: number; // 20 — 正文/卡片摘要
  MD: number; // 22 — 强调文字
  LG: number; // 24 — 标题
  XL: number; // 26 — 大标题
}

/**
 * 行高**比例**（对齐前端 theme-registry.css 的 --leading-* 标尺）。
 *
 * 多行/长文正文一律用比例制（配合 lineHeightFor），不要用上面的绝对档：
 * 绝对档只描述「某字号的某一种用法」，比例制才能保证换字号时行高跟着走。
 * 前端原注释：「Line-height scale — 正文行高档位。禁止 leading-[N] arbitrary 类与
 * 内联 line-height」，其中 card=1.7 是「原 1.55，阅读疲劳修复」后的值。
 */
export interface FontLineRatio {
  DEFAULT: number; // 1.5 — 基线（对齐前端 base.css 的 h1-h4/p/button 元素级 1.5）
  CARD: number; // 1.7 — 卡片/聊天气泡正文（对齐前端 --leading-card）
  COPY: number; // 1.75 — 长文正文（对齐前端 --leading-copy）
}

/**
 * 按字号与比例算行高（vp）。用于多行正文，避免内联魔法数字：
 *   .lineHeight(lineHeightFor(DesignConstants.FONT.SIZE.SM, DesignConstants.FONT.LINE_RATIO.COPY))
 */
export function lineHeightFor(fontSize: number, ratio: number): number {
  return Math.round(fontSize * ratio);
}

export interface FontLetterSpacing {
  EYEBROW: number;  // 1.4 — 分组标签/眉标（对齐前端 tracking-eyebrow 0.14em @10px）
  LABEL: number;    // 1.2 — 小标/在线标签（对齐前端 tracking-label 0.12em）
  FIELD: number;    // 1.28 — 表单字段标签（0.08em × 16px，对齐前端 tracking-field 0.08em）
  ICON_LABEL: number; // 0.6 — 图标下小字
  TITLE: number;    // -0.12 — 大标题微收紧（对齐前端 tracking-title -0.01em @12px+）
}

export interface FontWeights {
  REGULAR: number;
  MEDIUM: number;
  SEMIBOLD: number;
  BOLD: number;
}

export interface FontSystem {
  SIZE: FontSizes;
  WEIGHT: FontWeights;
  LINE_HEIGHT: FontLineHeights;
  /** 比例制行高（多行正文首选，见 FontLineRatio 注释） */
  LINE_RATIO: FontLineRatio;
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
  /** 轻量反馈/入场（150ms） */
  DURATION_FAST: number;
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
  /** 点赞计数弹跳放大（对齐前端 count-pop keyframes 1→1.3→1） */
  SCALE_LIKE_POP: number;
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
  /**
   * 弹簧动画 response / dampingFraction（面板吸附与高度过渡）。
   * 此前是散在地图位置选择器 3 处的裸值 `curves.springMotion(0.6, 0.9)`，
   * 与 PRESS(0.3/0.85)、TAB(0.4/0.75) 并列的第 3 套参数，提为语义 token。
   */
  SPRING_RESPONSE_PANEL: number;
  SPRING_DAMPING_PANEL: number;
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
  /** 次级内容淡化（对齐前端 opacity-50） */
  MUTED: number;
  /** 强调内容淡化（对齐前端 opacity-80） */
  EMPHASIZED: number;
}

export interface BorderWidthSystem {
  /** 0.5vp 亚像素行间分隔线（列表 divide-y 密度场景） */
  SUBPIXEL: number;
  /** 1vp 细线（默认边框，对齐前端 hairline） */
  HAIRLINE: number;
  /** 1.5vp 常规描边（头像/选中环，对齐前端 regular） */
  REGULAR: number;
  /** 2vp 强调边框（徽章/强分隔，对齐前端 strong） */
  STRONG: number;
  /** 3vp 重边框（头像环/定位点，对齐前端 heavy） */
  HEAVY: number;
}

// ==================== 组件特定常量接口 ====================
export interface AvatarSizes {
  /** 22 — 瀑布墙卡作者头像（对齐前端 MomentCard size 22px） */
  XXS: number;
  XS: number;
  /** 68 — 个人页身份卡头像（对齐前端 ProfileMobile Avatar size 68px） */
  PROFILE: number;
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
  /** 确认弹窗危险态图标圆底边长（44）。前端 ConfirmDialog 同容器为 size-10(40)，
   *  本档为存量平台适配值，仅收敛圆角（原 22 → RADIUS.FULL），不改尺寸 */
  DIALOG_ICON_TILE_SIZE: number;
  /** 主按钮大号（48，登录/提交主操作） */
  BUTTON_HEIGHT_LG: number;
  /** 次级按钮小号（36，列表/卡片内嵌入按钮） */
  BUTTON_HEIGHT_SM: number;
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
  /** 附近首页「身边的新朋友」横滑用户卡宽度（128）。前端 /nearby 聚合门户已删除
   *  （307 → /nearby/people，对端改为行列表），本档为本端门户专属，保留原值不变更布局 */
  NEARBY_USER_CARD_WIDTH: number;
  /** 首页「附近活动」行首图标圆底边长（34，对齐前端 ActivityRow size-8.5） */
  ACTIVITY_ICON_TILE_SIZE: number;
  /** 附近群组 featured 精选全宽卡高度（对齐前端 min-h-85=340px） */
  GROUP_FEATURED_HEIGHT: number;
  /** 通知中心加载空态区域高度（200） */
  LOADING_AREA_HEIGHT: number;
  /** 空态 fillMode 最小高度 */
  EMPTY_FILL_MIN_HEIGHT: number;
  /** 创建群组人数输入框宽度（64） */
  NUMBER_INPUT_WIDTH: number;
  /** 通用图标尺寸档（16/20/24） */
  ICON_SIZE_XS: number;
  ICON_SIZE_SM: number;
  ICON_SIZE_MD: number;
  /** 紧凑图标（14，徽章/小关闭钮；与 FONT.SIZE 解耦的独立 ICON 刻度） */
  ICON_SIZE_2XS: number;
  /** 快捷操作图标（22，profile quickAction；独立 ICON 刻度） */
  ICON_SIZE_SM_PLUS: number;
  /** 大图标（32，性能概览/空态强调；独立 ICON 刻度） */
  ICON_SIZE_LG: number;
  /** 空态大图标（48，错误/空态插画级图标；独立 ICON 刻度） */
  ICON_SIZE_EMPTY: number;
  /** 骨架屏卡片高度（100） */
  SKELETON_CARD_HEIGHT: number;
  /** 内容占位区高度（120，加载/空态占位） */
  PLACEHOLDER_AREA_HEIGHT: number;
  /** 大图占位高度（110，创建页图片区） */
  IMAGE_PLACEHOLDER_HEIGHT: number;
  /** 照片缩略图尺寸（80，CreateMomentSheet 图片选择器网格） */
  IMAGE_THUMBNAIL_SIZE: number;
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
  /** 主题预览卡 tab 选中指示条上移叠入 tab bar 的负 margin（-19，SettingsPage 预览卡） */
  THEME_PREVIEW_INDICATOR_OVERLAP: number;
  /** 资料编辑页头像区上移叠入 hero 头图的负 margin（-40，ProfileEditPage） */
  PROFILE_AVATAR_HEADER_OVERLAP: number;
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
  /** 图片消息最大高度（240 — 超高图限高防撑爆列表，对齐审查 P1 聊天三件套） */
  IMAGE_MAX_HEIGHT: number;
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
  /** 平板浮动玻璃药丸底栏最大宽度（对齐前端 --cst-tablet-dock-max 680px） */
  TABLET_NAV_MAX_WIDTH: number;
  /** 中央发布 FAB 尺寸（对齐前端 52x52） */
  FAB_SIZE: number;
  /** 圆形图标按钮尺寸（对齐前端 36px，移动端可放大到 38px） */
  ICON_BUTTON_SIZE: number;
  /** 页头返回按钮尺寸（40vp）：返回是导航关键动作，视觉与命中区大于普通图标钮（36），易点不误触 */
  BACK_BUTTON_SIZE: number;
  /** 最小触控命中区（44，对齐 WCAG 2.5.5 / HarmonyOS 规范最小热区；视觉尺寸不变时用透明 padding 扩展） */
  MIN_HIT_TARGET: number;
  /** 统计标签列宽（60，对齐前端明细标签列） */
  LABEL_WIDTH: number;
  /** 会话列表行尾未读徽章直径（对齐前端 size-5=20px） */
  UNREAD_BADGE_SIZE: number;
  /** 封面型页全幅封面高度（对齐前端 --cst-cover-height clamp 220→340px，移动端取 220） */
  COVER_HEIGHT: number;
  /** 头像条昵称截断宽度（布局几何，非主题风格） */
  NICKNAME_MAX_WIDTH: number;
  /** 首页桌面双栏左区宽度（420） */
  HOME_COLUMN_WIDTH: number;
  /** 地图筛选面板宽度（180） */
  FILTER_PANEL_WIDTH: number;
  /** 登录卡片最大宽度（对齐前端 /login max-w-100=400px） */
  LOGIN_CARD_MAX_WIDTH: number;
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
  /**
   * 页面横向 gutter 三档（对齐前端 ContentColumn 的 --cst-page-gutter）：
   * 移动 20 / 平板 24 / 桌面 28。**全站唯一横向留白来源**——页面内部不得再叠
   * 自己的 left/right padding，否则会出现「容器 16 + 页面 16 = 32」的双重留白，
   * 各页正文左边缘不在同一条竖线上，滑动时内容左右边界跳动。
   */
  GUTTER_MOBILE: number;
  GUTTER_TABLET: number;
  GUTTER_DESKTOP: number;
  /**
   * 页面纵向节奏（分区之间）：
   * - SECTION: 同页相邻分区（如首页「附近的人」与「附近群组」）的间距
   * - SECTION_INNER: 分区标题与其内容之间的间距
   * - PAGE_TOP: 页头/首个分区与页面顶部的间距
   * 对齐前端样板页 StandardPageLayout contentClassName="pt-4 pb-6"。
   */
  SECTION_GAP: number;
  SECTION_INNER_GAP: number;
  PAGE_TOP_GAP: number;
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
  BORDER_WIDTH: BorderWidthSystem;
  LAYOUT: LayoutSystem;
  PAGE_SHELL: PageShellSystem;
}

// ==================== 颜色系统（仅主题无关固定色）====================
const COLOR: ColorPalette = {
  MEDIA_CHROME_TEXT: '#F2F6F3',
  OVERLAY_LIGHT: '#FF59FFFF',
  OVERLAY_STRONG: '#00800000',
  IMAGE_OVERLAY_START: 'rgba(0, 0, 0, 0.04)',
  IMAGE_OVERLAY_END: 'rgba(0, 0, 0, 0.44)',
  IMAGE_BADGE_BG: 'rgba(15, 23, 42, 0.58)',
  IMAGE_BADGE_BORDER: 'rgba(255, 255, 255, 0.24)',
  OVERLAY_BORDER_LIGHT: '#FF40FFFF',
  OVERLAY_BORDER_DARK: '#00260000',
  WHITE: '#FFFFFF',
  TRANSPARENT: '#00000000',
  BLACK: '#000000',
  /** 系统窗口默认底色（未接入主题前的暖灰基底，对齐设计稿壁纸色） */
  SYSTEM_SURFACE_BEIGE: '#D0CBC3',
};

// ==================== 字体系统 ====================
const FONT: FontSystem = {
  SIZE: {
    XS: 11,
    SM: 12,
    MD: 14,
    LG: 16,
    XL: 20,
    XXL: 24,
    XXXL: 30, // 对齐前端 text-3xl(30)；原 32 为 2px 近邻档已归并
    XS2: 10, // 最小档（对齐前端 --text-xxs；原 9fp 的 XS_MINUS 已按可读性下限收敛并档）
    // 近邻档（13/15/22/28）已按前端口径归并：13→12、15→14、22→20、28→24，
    // 消除 1-2px 近邻档位（前端 theme-registry 同原则）
    LG_PLUS: 18,
    HERO: 36, // 对齐前端 text-4xl(36，前端最大 text 档)；原 48/56 超前端刻度已归并
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
  // 比例制行高（对齐前端 --leading-*）：多行正文用它，绝对档只用于单行紧凑场景
  LINE_RATIO: {
    DEFAULT: 1.5,
    CARD: 1.7,
    COPY: 1.75,
  },
  LETTER_SPACING: {
    EYEBROW: 1.4,
    LABEL: 1.2,
    FIELD: 1.28,
    ICON_LABEL: 0.6,
    TITLE: -0.12, // 大标题微收紧（对齐前端 tracking-title -0.01em @12px+）
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

// ==================== 圆角系统（HarmonyOS vp 适配档，向前端 --radius 收敛）====================
// 前端刻度：sm 12 / md 16 / lg 20 / xl 24（sheet 28），禁止近邻档位（8/18/22 已删除）。
// 语义圆角由 SHAPE 档位表达（CARD/INPUT/BUTTON/SHEET 等），页面禁止再出现数值字面量。
const RADIUS: RadiusSystem = {
  SM: 12,
  MD: 16,
  LG: 20,
  XL: 24,
  FULL: 999,
};

const SHAPE: ShapeSemanticTokens = {
  CARD: RADIUS.LG,          // 20：普通卡片（前端 rounded-lg 主流档）
  HERO_CARD: RADIUS.XL,     // 24：封面/hero 区
  COMPACT_CARD: RADIUS.MD,  // 16：紧凑卡片/列表容器（前端 rounded-md）
  INPUT: RADIUS.MD,         // 16：输入框（对齐前端 input-surface）
  // 主按钮胶囊（对齐前端 rounded-full 主按钮；原 28 圆角矩形档已删除）
  BUTTON: RADIUS.FULL,
  CHIP: RADIUS.FULL,
  TAB_BAR: RADIUS.XL,
  PREVIEW_CARD: RADIUS.LG,
  LIST_ITEM: RADIUS.LG,
  ICON_TILE: RADIUS.SM,
  SHEET_TOP_RADIUS: 28,     // 对齐前端 --radius-sheet（弹层专用大圆角）
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
  DURATION_FAST: 150,
  DURATION_SLOW: 1500,
  DURATION_STAGGER_DELAY: 150,
  CURVE_EASE_IN_OUT: Curve.EaseInOut,
  CURVE_EASE_OUT: Curve.EaseOut,
  CURVE_FAST_OUT_SLOW_IN: Curve.FastOutSlowIn,
  CURVE_SMOOTH: Curve.Smooth,
};

// ==================== 交互反馈系统 ====================
const INTERACTION: InteractionSystem = {
  SCALE_LIKE_POP: 1.3,
  SCALE_PRESSED_CARD: 0.97,
  SCALE_PRESSED_MASONRY: 0.96,
  SCALE_PRESSED_BUTTON: 0.95,
  SCALE_PRESSED_ICON: 0.90,
  SCALE_PRESSED_INPUT_ROW: 0.98,
  SCALE_PRESSED_PILL: 0.94,
  HOVER_ELEVATION: 4,
  HOVER_ELEVATION_REST: 1,
  SPRING_RESPONSE_PRESS: 0.3,
  SPRING_DAMPING_PRESS: 0.85,
  SPRING_RESPONSE_TAB: 0.4,
  SPRING_DAMPING_TAB: 0.75,
  SPRING_RESPONSE_PANEL: 0.6,
  SPRING_DAMPING_PANEL: 0.9,
};

// ==================== 组件特定常量 ====================
const EFFECT: MaterialEffectSystem = {
  TEXTURE_SOFT: 0.12
};

const COMPONENT: ComponentConstants = {
  AVATAR_SIZE: {
    XXS: 22,
    XS: 32,
    PROFILE: 68,
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
  DIALOG_ICON_TILE_SIZE: 44,
  BUTTON_HEIGHT_LG: 48,
  BUTTON_HEIGHT_SM: 36,
  FILTER_CHIP_HEIGHT: 30,
  STATUS_ICON_SIZE: 28,
  LOADING_ICON_SIZE: 20,
  EMPTY_ICON_SIZE: 64,
  LOADING_CONTAINER_SIZE: 72,
  BRAND_ICON_SIZE: 38,
  BANNER_HEIGHT: 140,
  GROUP_CARD_WIDTH: 140,
  NEARBY_USER_CARD_WIDTH: 128,
  ACTIVITY_ICON_TILE_SIZE: 34,
  /** 附近群组 featured 精选全宽卡高度（对齐前端 min-h-85=340px） */
  GROUP_FEATURED_HEIGHT: 340,
  LOADING_AREA_HEIGHT: 200,
  /** 空态 fillMode 最小高度（对齐前端页面级空态撑高） */
  EMPTY_FILL_MIN_HEIGHT: 420,
  NUMBER_INPUT_WIDTH: 64,
  ICON_SIZE_XS: 16,
  ICON_SIZE_SM: 20,
  ICON_SIZE_MD: 24,
  ICON_SIZE_2XS: 14,
  ICON_SIZE_SM_PLUS: 22,
  ICON_SIZE_LG: 32,
  ICON_SIZE_EMPTY: 48,
  SKELETON_CARD_HEIGHT: 100,
  PLACEHOLDER_AREA_HEIGHT: 120,
  IMAGE_PLACEHOLDER_HEIGHT: 110,
  /** 照片缩略图尺寸（80，CreateMomentSheet 图片选择器网格） */
  IMAGE_THUMBNAIL_SIZE: 80,
  SHARE_CARD_HEIGHT: 130,
  LOG_AREA_HEIGHT: 300,
  LOADING_GLYPH_SIZE: 46,
  PASSWORD_INPUT_HEIGHT: 46,
  LOGOUT_BUTTON_HEIGHT: 50,
  THEME_PREVIEW_BAR_WIDTH: 54,
  THEME_PREVIEW_BAR_HEIGHT: 16,
  PROGRESS_STROKE_WIDTH: 4,
  // 负 margin 特例：均为「内容叠入上方区块」的既成视觉，命名以说明用途，
  // 禁止新增裸负值（审查 P3-3）
  /** 主题预览卡 tab 选中指示条上移叠入 tab bar（SettingsPage 预览卡） */
  THEME_PREVIEW_INDICATOR_OVERLAP: -19,
  /** 资料编辑页头像区上移叠入 hero 头图（ProfileEditPage） */
  PROFILE_AVATAR_HEADER_OVERLAP: -40,
};

const CHAT: ChatLayoutConstants = {
  // 气泡主体圆角对齐前端 shape-md（16）；原 18 为已删除近邻档（RADIUS 刻度 12/16/20/24）
  BUBBLE_RADIUS: 16,
  BUBBLE_PADDING_H: 16,
  BUBBLE_PADDING_V: 10,
  IMAGE_BUBBLE_PADDING_H: 14,
  IMAGE_BUBBLE_PADDING_V: 6,
  BUBBLE_MAX_WIDTH: '75%',
  MESSAGE_GAP: 16,
  // 气泡正文字号归并到 SM（12）；原 13 为已删除近邻档（FONT.SIZE 刻度）
  BUBBLE_FONT_SIZE: 12,
  INPUT_HEIGHT: 38,
  SEND_BUTTON_HEIGHT: 36,
  AVATAR_SIZE: 40,
  IMAGE_MAX_WIDTH: 200,
  IMAGE_MAX_HEIGHT: 240,
};

// ==================== 页面布局系统（对齐前端 navigation chrome）====================
const LAYOUT: LayoutSystem = {
  DESKTOP_HEADER_HEIGHT: 52,
  DESKTOP_SIDEBAR_WIDTH: 224,
  RIGHT_SIDEBAR_WIDTH: 220,
  MOBILE_BOTTOM_NAV_HEIGHT: 56,
  TABLET_NAV_MAX_WIDTH: 680,
  FAB_SIZE: 52,
  ICON_BUTTON_SIZE: 36,
  BACK_BUTTON_SIZE: 40,
  MIN_HIT_TARGET: 44,
  LABEL_WIDTH: 60,
  /** 会话列表行尾未读徽章直径（对齐前端 size-5=20px） */
  UNREAD_BADGE_SIZE: 20,
  COVER_HEIGHT: 220,
  /** 头像条昵称截断宽度（布局几何，非主题风格） */
  NICKNAME_MAX_WIDTH: 56,
  HOME_COLUMN_WIDTH: 420,
  FILTER_PANEL_WIDTH: 180,
  LOGIN_CARD_MAX_WIDTH: 400,
};

// ==================== 页面内容栏档位（对齐前端 .page-shell / .feed-density-shell）====================
const PAGE_SHELL: PageShellSystem = {
  SHELL_DEFAULT_MAX: 1180,
  SHELL_WIDE_MAX: 1320,
  SHELL_NARROW_MAX: 720,
  SHELL_FORM_MAX: 560,
  FEED_NARROW_MAX: 640,
  FEED_MEDIUM_MAX: 960,
  // 对齐前端 ContentColumn --cst-page-gutter（1.25rem / 1.5rem / 1.75rem）
  GUTTER_MOBILE: 20,
  GUTTER_TABLET: 24,
  GUTTER_DESKTOP: 28,
  // 纵向节奏：分区之间必须明显大于「分区标题→其内容」，
  // 标题才会在视觉上归属下方内容（否则标题飘在两块之间，整页读不出分组）。
  // 24 : 12 = 2:1 是移动端列表页的常用层次比。
  SECTION_GAP: 24,        // SPACING.XL：分区之间
  SECTION_INNER_GAP: 12,  // SPACING.MD：分区标题与其内容
  PAGE_TOP_GAP: 16,       // SPACING.LG：页头与首个分区
};

// ==================== 透明度系统 ====================
const OPACITY: OpacitySystem = {
  DISABLED: 0.38,
  PRESSED: 0.72,
  LOADING: 0.58,
  SUBTLE: 0.85,
  MUTED: 0.5,
  EMPHASIZED: 0.8,
};

// ==================== 边框宽度系统（对齐前端 --cst-border-width-* 档位）====================
const BORDER_WIDTH: BorderWidthSystem = {
  SUBPIXEL: 0.5,
  HAIRLINE: 1,
  REGULAR: 1.5,
  STRONG: 2,
  HEAVY: 3,
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
  BORDER_WIDTH,
  LAYOUT,
  PAGE_SHELL,
};
