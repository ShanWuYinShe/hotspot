export default defineAppConfig({
  ui: {
    // ============================================================================
    // 主题配置
    // ============================================================================
    // 主题色：blue（蓝色）- 经典、专业、高对比度
    // 注：绿色系(teal/emerald/green)在浅色背景上对比度不足，蓝色是最安全的选择
    primary: "blue",
    // 中性色：neutral - 用于文字和背景
    gray: "neutral",
    
    // 禁用 Dashboard 搜索功能
    dashboard: {
      search: false
    },

    // 组件样式覆盖 - 提升对比度
    pageCard: {
      slots: {
        // 图标使用中性色，确保在浅色背景上清晰可见
        icon: 'text-neutral-700 dark:text-neutral-300'
      }
    },

    // ============================================================================
    // 全局视觉设计系统
    // ============================================================================
    
    // 圆角系统（增强视觉层次）
    radius: {
      sm: '6px',
      md: '10px',
      lg: '14px',
      xl: '18px',
      '2xl': '24px',
      full: '9999px',
    },

    // 阴影系统（增强深度感）
    shadow: {
      sm: '0 1px 2px 0 rgba(0, 0, 0, 0.05)',
      md: '0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -1px rgba(0, 0, 0, 0.06)',
      lg: '0 10px 15px -3px rgba(0, 0, 0, 0.1), 0 4px 6px -2px rgba(0, 0, 0, 0.05)',
      xl: '0 20px 25px -5px rgba(0, 0, 0, 0.1), 0 10px 10px -5px rgba(0, 0, 0, 0.04)',
      '2xl': '0 25px 50px -12px rgba(0, 0, 0, 0.25)',
      inner: 'inset 0 2px 4px 0 rgba(0, 0, 0, 0.06)',
    },

    // 过渡动画（统一动画时长和缓动函数）
    transition: {
      fast: '150ms cubic-bezier(0.4, 0, 0.2, 1)',
      base: '250ms cubic-bezier(0.4, 0, 0.2, 1)',
      slow: '400ms cubic-bezier(0.4, 0, 0.2, 1)',
      bounce: '500ms cubic-bezier(0.68, -0.55, 0.265, 1.55)',
    },

    // 间距系统（统一的间距标准）
    spacing: {
      xs: '0.5rem',    // 8px
      sm: '0.75rem',   // 12px
      md: '1rem',      // 16px
      lg: '1.5rem',    // 24px
      xl: '2rem',      // 32px
      '2xl': '3rem',   // 48px
      '3xl': '4rem',   // 64px
    },

    // 字体大小系统（增强可读性）
    fontSize: {
      xs: '0.75rem',   // 12px
      sm: '0.875rem',  // 14px
      base: '1rem',    // 16px
      lg: '1.125rem',  // 18px
      xl: '1.25rem',   // 20px
      '2xl': '1.5rem', // 24px
      '3xl': '1.875rem', // 30px
      '4xl': '2.25rem',  // 36px
    },

    // 图标大小系统
    iconSize: {
      xs: '14px',
      sm: '16px',
      md: '20px',
      lg: '24px',
      xl: '32px',
    },
  },

  // ============================================================================
  // 主题切换配置
  // ============================================================================
  theme: {
    default: 'system',
    options: ['light', 'dark', 'system'],
    transition: {
      duration: '300ms',
      timing: 'cubic-bezier(0.4, 0, 0.2, 1)',
    },
  },

  // ============================================================================
  // 布局配置
  // ============================================================================
  layout: {
    default: 'dashboard',
    breakpoints: {
      sm: '640px',
      md: '768px',
      lg: '1024px',
      xl: '1280px',
      '2xl': '1536px',
    },
    container: {
      padding: {
        sm: '1rem',
        md: '1.5rem',
        lg: '2rem',
      },
      maxWidth: {
        sm: '640px',
        md: '768px',
        lg: '1024px',
        xl: '1280px',
        '2xl': '1536px',
      },
    },
  },

  // ============================================================================
  // 动画配置
  // ============================================================================
  animation: {
    // 页面切换动画
    pageTransition: {
      name: 'page',
      mode: 'out-in',
      duration: 300,
    },
    // 布局切换动画
    layoutTransition: {
      name: 'layout',
      mode: 'out-in',
      duration: 300,
    },
    // 微交互动画配置
    microInteraction: {
      scale: {
        hover: 1.02,
        active: 0.98,
      },
      opacity: {
        hover: 0.9,
        disabled: 0.5,
      },
      duration: 200,
    },
  },

  // ============================================================================
  // 通知配置
  // ============================================================================
  notification: {
    position: 'bottom-right',
    duration: 5000,
    progress: true,
    expand: true,
  },

  // ============================================================================
  // 用户体验增强
  // ============================================================================
  ux: {
    // 骨架屏配置
    skeleton: {
      baseColor: 'rgba(0, 0, 0, 0.05)',
      highlightColor: 'rgba(0, 0, 0, 0.02)',
      animationDuration: '1.5s',
    },
    // 加载配置
    loading: {
      debounce: 200,  // 防抖时间（毫秒）
      minDisplay: 300, // 最小显示时间（毫秒）
    },
    // 滚动配置
    scroll: {
      behavior: 'smooth',
      threshold: 100, // 触发"返回顶部"按钮的阈值
    },
  },
});