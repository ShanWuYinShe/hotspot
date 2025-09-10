export default defineAppConfig({
  ui: {
    // 现代化设计系统 - 基于人机界面设计原则
    // 参考: https://developer.apple.com/design/human-interface-guidelines/
    primary: "blue",
    gray: "neutral",
    
    // Nuxt UI v3 颜色系统配置
    colors: {
      // 基础颜色映射
      primary: "blue",
      secondary: "slate", 
      neutral: "zinc",
      info: "blue",
      success: "green",
      warning: "orange",
      error: "red",
    },

    // 现代按钮设计 - 44pt 最小触摸目标
    button: {
      slots: {
        base: "rounded-lg focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/50 transition-all duration-200 ease-out min-h-[44px] font-semibold hover:-translate-y-0.5",
        label: "font-medium",
        leadingIcon: "",
        trailingIcon: "",
      },
      defaultVariants: {
        color: "primary",
        variant: "solid",
        size: "md",
      },
      variants: {
        variant: {
          solid: "shadow-sm hover:shadow-lg",
          outline: "border-2 hover:bg-{color}-50 dark:hover:bg-{color}-950/20",
          ghost: "border-0 hover:bg-{color}-100 dark:hover:bg-{color}-900/30",
          soft: "bg-{color}-100 hover:bg-{color}-200 dark:bg-{color}-900/30 dark:hover:bg-{color}-800/40",
          subtle: "bg-transparent hover:bg-neutral-100 dark:hover:bg-neutral-800/50"
        },
        size: {
          xs: "min-h-[32px] px-3 text-sm",
          sm: "min-h-[36px] px-4 text-sm", 
          md: "min-h-[44px] px-6 text-base", // 标准触摸目标
          lg: "min-h-[50px] px-8 text-lg",
          xl: "min-h-[56px] px-10 text-xl"
        }
      }
    },

    // 现代输入框设计 - 44pt 最小触摸目标
    input: {
      slots: {
        base: "rounded-lg border border-neutral-200 dark:border-neutral-700 bg-white dark:bg-neutral-900 text-neutral-900 dark:text-neutral-100 placeholder:text-neutral-500 focus:border-primary-500 focus:ring-2 focus:ring-primary-500/20 transition-all duration-200 ease-out min-h-[44px]",
        leading: "",
        leadingIcon: "text-neutral-500 dark:text-neutral-400",
        trailing: "",
        trailingIcon: "text-neutral-500 dark:text-neutral-400",
      },
      defaultVariants: {
        size: "md",
      },
      variants: {
        size: {
          sm: "min-h-[36px] px-3 text-sm",
          md: "min-h-[44px] px-4 text-base", // 标准触摸目标
          lg: "min-h-[50px] px-5 text-lg"
        }
      }
    },

    // 现代卡片设计 - 分组背景风格
    card: {
      slots: {
        root: "bg-white dark:bg-neutral-900 border border-neutral-200 dark:border-neutral-700 rounded-xl text-neutral-900 dark:text-neutral-100 shadow-sm hover:shadow-lg hover:-translate-y-1 transition-all duration-200 ease-out",
        header: "pb-4 border-b border-neutral-200 dark:border-neutral-700 font-semibold text-lg",
        body: "p-6 text-base leading-relaxed",
        footer: "pt-4 border-t border-neutral-200 dark:border-neutral-700",
      },
      variants: {
        variant: {
          default: "",
          elevated: "shadow-md hover:shadow-xl",
          outlined: "border-2 border-neutral-300 dark:border-neutral-600",
          filled: "bg-neutral-50 dark:bg-neutral-800 border-0"
        }
      }
    },

    // 现代模态框设计
    modal: {
      slots: {
        overlay: "backdrop-blur-md bg-black/30",
        wrapper: "",
        content: "bg-white dark:bg-neutral-900 rounded-2xl shadow-2xl border border-neutral-200 dark:border-neutral-700 text-neutral-900 dark:text-neutral-100 overflow-hidden",
        header: "px-6 py-4 border-b border-neutral-200 dark:border-neutral-700 text-center",
        body: "px-6 py-6",
        footer: "px-6 py-4 border-t border-neutral-200 dark:border-neutral-700 flex gap-3 justify-end",
        title: "font-semibold text-lg text-neutral-900 dark:text-neutral-100",
        description: "text-sm text-neutral-600 dark:text-neutral-400 mt-2",
        close: "text-neutral-500 hover:text-neutral-700 dark:text-neutral-400 dark:hover:text-neutral-200 transition-colors duration-200",
      },
    },

    // 现代工具提示设计
    tooltip: {
      slots: {
        content: "backdrop-blur-sm bg-neutral-900/90 dark:bg-neutral-800/90 text-white rounded-lg px-3 py-2 text-sm font-medium shadow-lg",
        arrow: "",
        text: "",
        kbds: "bg-neutral-700 text-neutral-200 rounded px-1.5 py-0.5 text-xs font-mono",
        kbdsSize: "",
      },
    },

    // 现代警告框设计
    alert: {
      slots: {
        root: "rounded-xl border border-neutral-200 dark:border-neutral-700 bg-white dark:bg-neutral-900 text-neutral-900 dark:text-neutral-100 shadow-sm",
        wrapper: "p-4",
        title: "font-semibold text-base",
        description: "text-sm text-neutral-600 dark:text-neutral-400 mt-1",
        icon: "flex-shrink-0",
        avatar: "",
        avatarSize: "",
        actions: "flex gap-2 mt-3",
        close: "text-neutral-500 hover:text-neutral-700 dark:text-neutral-400 dark:hover:text-neutral-200 transition-colors duration-200",
      },
    },

    // 现代徽章设计
    badge: {
      slots: {
        base: "rounded-full px-2.5 py-1 text-xs font-semibold inline-flex items-center gap-1",
        label: "",
        leadingIcon: "w-3 h-3",
        trailingIcon: "w-3 h-3",
        leadingAvatar: "",
        leadingAvatarSize: "",
      },
      variants: {
        variant: {
          solid: "",
          outline: "border-2 bg-transparent",
          soft: "bg-opacity-20",
          subtle: "bg-opacity-10"
        }
      }
    },

    // 现代头像设计 - 增强交互效果
    avatar: {
      slots: {
        root: "rounded-xl transition-all duration-200 hover:scale-105 hover:shadow-lg ring-2 ring-transparent hover:ring-primary-500/20",
        image: "object-cover",
        fallback: "bg-gradient-to-br from-primary-500 to-primary-600 text-white font-semibold flex items-center justify-center",
        icon: "",
      },
      variants: {
        size: {
          xs: "w-6 h-6 text-xs",
          sm: "w-8 h-8 text-sm",
          md: "w-10 h-10 text-base",
          lg: "w-12 h-12 text-lg",
          xl: "w-16 h-16 text-xl"
        }
      }
    },

    // 现代弹出框设计
    popover: {
      slots: {
        content: "border border-neutral-200 dark:border-neutral-700 rounded-xl shadow-lg bg-white dark:bg-neutral-900 backdrop-blur-sm text-neutral-900 dark:text-neutral-100 p-1",
        arrow: "fill-white dark:fill-neutral-900",
      },
    },

    // 现代下拉菜单设计 - 增强交互效果
    dropdown: {
      slots: {
        content: "border border-neutral-200 dark:border-neutral-700 rounded-xl shadow-lg bg-white dark:bg-neutral-900 backdrop-blur-sm text-neutral-900 dark:text-neutral-100 p-1 min-w-[160px]",
        item: "rounded-lg px-3 py-2 text-sm hover:bg-neutral-100 dark:hover:bg-neutral-800 transition-all duration-150 cursor-pointer flex items-center gap-2",
        divider: "my-1 border-t border-neutral-200 dark:border-neutral-700",
        label: "px-3 py-2 text-xs font-semibold text-neutral-500 dark:text-neutral-400 uppercase tracking-wider",
      },
    },

    // 现代标签页设计
    tabs: {
      slots: {
        root: "",
        list: "flex gap-1 p-1 bg-neutral-100 dark:bg-neutral-800 rounded-lg",
        trigger: "rounded-md px-3 py-2 text-sm font-medium transition-all duration-200 data-[state=active]:bg-white data-[state=active]:shadow-sm dark:data-[state=active]:bg-neutral-900",
        content: "mt-4",
      },
    },

    // 现代表格设计
    table: {
      slots: {
        root: "min-w-full table-fixed text-sm text-neutral-900 dark:text-neutral-100 border-collapse",
        base: "border border-neutral-200 dark:border-neutral-700 rounded-lg overflow-hidden",
        caption: "text-sm text-neutral-600 dark:text-neutral-400 mb-4",
        thead: "bg-neutral-50 dark:bg-neutral-800",
        tbody: "divide-y divide-neutral-200 dark:divide-neutral-700",
        tr: "hover:bg-neutral-50 dark:hover:bg-neutral-800/50 transition-colors duration-150",
        th: "text-left font-semibold py-3 px-4 text-neutral-900 dark:text-neutral-100",
        td: "py-3 px-4 text-neutral-700 dark:text-neutral-300",
        tfoot: "bg-neutral-50 dark:bg-neutral-800",
        progress: "h-1 bg-primary-500",
        empty: "text-center py-8 text-neutral-500 dark:text-neutral-400",
        loading: "text-center py-8",
      },
    },

    // 现代分页设计 - 增强交互效果
    pagination: {
      slots: {
        root: "inline-flex items-center gap-2",
        list: "flex items-center gap-1",
        ellipsis: "px-2 text-neutral-500",
        label: "text-sm text-neutral-600 dark:text-neutral-400",
        first: "min-w-10 h-10 rounded-lg border border-neutral-200 dark:border-neutral-700 hover:bg-neutral-100 dark:hover:bg-neutral-800 hover:border-neutral-300 dark:hover:border-neutral-600 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/50 transition-all duration-200 flex items-center justify-center",
        prev: "min-w-10 h-10 rounded-lg border border-neutral-200 dark:border-neutral-700 hover:bg-neutral-100 dark:hover:bg-neutral-800 hover:border-neutral-300 dark:hover:border-neutral-600 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/50 transition-all duration-200 flex items-center justify-center",
        item: "min-w-10 h-10 rounded-lg border border-neutral-200 dark:border-neutral-700 hover:bg-neutral-100 dark:hover:bg-neutral-800 hover:border-neutral-300 dark:hover:border-neutral-600 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/50 transition-all duration-200 flex items-center justify-center data-[active=true]:bg-primary-500 data-[active=true]:text-white data-[active=true]:border-primary-500",
        next: "min-w-10 h-10 rounded-lg border border-neutral-200 dark:border-neutral-700 hover:bg-neutral-100 dark:hover:bg-neutral-800 hover:border-neutral-300 dark:hover:border-neutral-600 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/50 transition-all duration-200 flex items-center justify-center",
        last: "min-w-10 h-10 rounded-lg border border-neutral-200 dark:border-neutral-700 hover:bg-neutral-100 dark:hover:bg-neutral-800 hover:border-neutral-300 dark:hover:border-neutral-600 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/50 transition-all duration-200 flex items-center justify-center",
      },
    },

    // 现代通知设计 - 智能定位避免遮挡
    toast: {
      slots: {
        root: "pointer-events-auto relative overflow-hidden rounded-xl border border-neutral-200 dark:border-neutral-700 bg-white dark:bg-neutral-900 backdrop-blur-sm shadow-xl",
        wrapper: "p-4",
        title: "font-semibold text-base text-neutral-900 dark:text-neutral-100",
        description: "text-sm text-neutral-600 dark:text-neutral-400 mt-1 leading-relaxed",
        close: "absolute right-3 top-3 text-neutral-500 hover:text-neutral-700 dark:text-neutral-400 dark:hover:text-neutral-200 transition-colors duration-200",
        actions: "flex gap-2 mt-4",
        progress: "absolute bottom-0 left-0 h-1 bg-primary-500 transition-all duration-100",
      },
      defaultVariants: {
        color: "primary"
      }
    },

    // 现代表单字段设计
    formField: {
      slots: {
        root: "space-y-2",
        wrapper: "",
        labelWrapper: "flex items-center justify-between",
        label: "text-sm font-semibold text-neutral-900 dark:text-neutral-100",
        container: "",
        description: "text-sm text-neutral-600 dark:text-neutral-400 leading-relaxed",
        error: "text-sm text-red-600 dark:text-red-400 font-medium flex items-center gap-1",
        hint: "text-xs text-neutral-500 dark:text-neutral-400",
        help: "text-xs text-neutral-500 dark:text-neutral-400 leading-relaxed",
      },
    },
  },
});
