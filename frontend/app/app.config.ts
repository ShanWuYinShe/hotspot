export default defineAppConfig({
  ui: {
    // 主题配置
    primary: "blue",
    gray: "neutral",
    // 禁用 Dashboard 搜索功能
    dashboard: {
      search: false
    }
  },

  // 主题切换配置
  theme: {
    default: 'system',
    options: ['light', 'dark', 'system'],
    transition: {
      duration: '300ms',
      timing: 'ease-out',
    },
  },

  // 布局配置
  layout: {
    default: 'dashboard',
    breakpoints: {
      sm: '640px',
      md: '768px',
      lg: '1024px',
      xl: '1280px',
      '2xl': '1536px',
    },
  },

  // 动画配置
  animation: {
    pageTransition: {
      name: 'page',
      mode: 'out-in',
      duration: 300,
    },
    layoutTransition: {
      name: 'layout',
      mode: 'out-in',
      duration: 300,
    },
  },

  // 通知配置
  notification: {
    position: 'bottom-right',
    duration: 5000,
    progress: true,
    expand: true,
  },
});