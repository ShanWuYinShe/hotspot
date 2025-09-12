/**
 * HotSpot 设计系统配置
 * 统一的UI设计规范，提升视觉质量和用户体验
 */

// 间距系统 - 基于8px网格系统
export const SPACING = {
  // 基础间距单位
  unit: 8, // 8px base unit
  
  // 内边距系统
  padding: {
    xs: 'p-2',     // 8px
    sm: 'p-3',     // 12px  
    md: 'p-4',     // 16px
    lg: 'p-6',     // 24px
    xl: 'p-8',     // 32px
    '2xl': 'p-12', // 48px
  },
  
  // 外边距系统
  margin: {
    xs: 'm-2',     // 8px
    sm: 'm-3',     // 12px
    md: 'm-4',     // 16px
    lg: 'm-6',     // 24px
    xl: 'm-8',     // 32px
    '2xl': 'm-12', // 48px
  },
  
  // 间隙系统
  gap: {
    xs: 'gap-2',   // 8px
    sm: 'gap-3',   // 12px
    md: 'gap-4',   // 16px
    lg: 'gap-6',   // 24px
    xl: 'gap-8',   // 32px
    '2xl': 'gap-12', // 48px
  }
} as const;

// 搜索框设计规范
export const SEARCH_INPUT = {
  // 图标间距 - 确保图标不紧贴边缘
  iconPadding: {
    leading: 'pl-4',  // 左侧图标16px间距
    trailing: 'pr-4', // 右侧图标16px间距
  },
  
  // 内容间距
  contentPadding: {
    x: 'px-12',       // 左右48px，为图标留出空间
    y: 'py-3',        // 上下12px
  },
  
  // 图标尺寸
  iconSize: {
    sm: 'w-4 h-4',    // 16x16px
    md: 'w-5 h-5',    // 20x20px
    lg: 'w-6 h-6',    // 24x24px
  },
  
  // 圆角
  borderRadius: {
    sm: 'rounded-md',  // 6px
    md: 'rounded-lg',  // 8px
    lg: 'rounded-xl',  // 12px
  },
  
  // 阴影
  shadow: {
    none: 'shadow-none',
    sm: 'shadow-sm',
    md: 'shadow-md',
    focus: 'focus:shadow-lg focus:shadow-primary-500/25',
  }
} as const;

// 按钮设计规范
export const BUTTON = {
  // 内边距
  padding: {
    xs: 'px-3 py-1.5',   // 12px x 6px
    sm: 'px-4 py-2',     // 16px x 8px
    md: 'px-6 py-3',     // 24px x 12px
    lg: 'px-8 py-4',     // 32px x 16px
  },
  
  // 图标间距
  iconSpacing: {
    left: 'mr-2',        // 图标在左侧时的右边距
    right: 'ml-2',       // 图标在右侧时的左边距
  },
  
  // 悬停效果（简化版）
  hover: {
    background: 'hover:bg-neutral-50 dark:hover:bg-neutral-800',
    shadow: 'hover:shadow-md',
    brightness: 'hover:brightness-105',
  },
  
  // 过渡动画（仅颜色和透明度）
  transition: 'transition-colors duration-200 ease-in-out',
} as const;

// 卡片设计规范
export const CARD = {
  // 内边距
  padding: {
    sm: 'p-4',           // 16px
    md: 'p-6',           // 24px
    lg: 'p-8',           // 32px
  },
  
  // 圆角
  borderRadius: {
    sm: 'rounded-lg',    // 8px
    md: 'rounded-xl',    // 12px
    lg: 'rounded-2xl',   // 16px
  },
  
  // 阴影
  shadow: {
    none: 'shadow-none',
    sm: 'shadow-sm',
    md: 'shadow-md',
    lg: 'shadow-lg',
    hover: 'hover:shadow-xl',
  },
  
  // 边框
  border: {
    none: 'border-0',
    light: 'border border-neutral-200 dark:border-neutral-700',
    medium: 'border-2 border-neutral-300 dark:border-neutral-600',
  }
} as const;

// 图标设计规范
export const ICON = {
  // 容器间距
  containerPadding: {
    xs: 'p-1.5',         // 6px
    sm: 'p-2',           // 8px
    md: 'p-3',           // 12px
    lg: 'p-4',           // 16px
    xl: 'p-5',           // 20px
  },
  
  // 图标尺寸
  size: {
    xs: 'w-3 h-3',       // 12x12px
    sm: 'w-4 h-4',       // 16x16px
    md: 'w-5 h-5',       // 20x20px
    lg: 'w-6 h-6',       // 24x24px
    xl: 'w-8 h-8',       // 32x32px
    '2xl': 'w-10 h-10',  // 40x40px
  },
  
  // 容器尺寸
  containerSize: {
    sm: 'w-8 h-8',       // 32x32px
    md: 'w-10 h-10',     // 40x40px
    lg: 'w-12 h-12',     // 48x48px
    xl: 'w-16 h-16',     // 64x64px
  }
} as const;

// 视觉层次系统
export const VISUAL_HIERARCHY = {
  // 字体大小
  fontSize: {
    xs: 'text-xs',       // 12px
    sm: 'text-sm',       // 14px
    base: 'text-base',   // 16px
    lg: 'text-lg',       // 18px
    xl: 'text-xl',       // 20px
    '2xl': 'text-2xl',   // 24px
    '3xl': 'text-3xl',   // 30px
  },
  
  // 字体粗细
  fontWeight: {
    normal: 'font-normal',     // 400
    medium: 'font-medium',     // 500
    semibold: 'font-semibold', // 600
    bold: 'font-bold',         // 700
  },
  
  // 颜色对比度
  textColor: {
    primary: 'text-neutral-900 dark:text-neutral-100',     // 主要文本
    secondary: 'text-neutral-700 dark:text-neutral-300',   // 次要文本
    muted: 'text-neutral-500 dark:text-neutral-400',       // 辅助文本
    disabled: 'text-neutral-400 dark:text-neutral-500',    // 禁用文本
  }
} as const;

// 交互反馈系统
export const INTERACTION = {
  // 悬停效果
  hover: {
    subtle: 'hover:bg-neutral-50 dark:hover:bg-neutral-800',
    medium: 'hover:bg-neutral-100 dark:hover:bg-neutral-700',
    strong: 'hover:bg-neutral-200 dark:hover:bg-neutral-600',
  },
  
  // 焦点效果
  focus: {
    ring: 'focus:ring-2 focus:ring-primary-500 focus:ring-offset-2',
    outline: 'focus:outline-none focus:ring-2 focus:ring-primary-500',
  },
  
  // 激活效果（简化版）
  active: {
    background: 'active:bg-neutral-100 dark:active:bg-neutral-700',
    brightness: 'active:brightness-95',
  },
  
  // 过渡动画
  transition: {
    fast: 'transition-colors duration-150 ease-in-out',
    normal: 'transition-colors duration-200 ease-in-out',
    slow: 'transition-colors duration-300 ease-in-out',
  }
} as const;

// 响应式断点
export const BREAKPOINTS = {
  sm: '640px',   // 手机横屏
  md: '768px',   // 平板竖屏
  lg: '1024px',  // 平板横屏/小桌面
  xl: '1280px',  // 桌面
  '2xl': '1536px', // 大桌面
} as const;

// 导出所有设计系统配置
export const DESIGN_SYSTEM = {
  SPACING,
  SEARCH_INPUT,
  BUTTON,
  CARD,
  ICON,
  VISUAL_HIERARCHY,
  INTERACTION,
  BREAKPOINTS,
} as const;

export default DESIGN_SYSTEM;
