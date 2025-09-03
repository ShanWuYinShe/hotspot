export default defineAppConfig({
  ui: {
    // Base palette mapping (align with Nuxt UI design system)
    // https://ui.nuxt.com/getting-started/theme
    primary: "blue",
    gray: "neutral",
    colors: {
      primary: "blue",
      neutral: "zinc",
      info: "blue",
      success: "green",
      warning: "yellow",
      error: "red",
    },

    // UButton theme (use slots/defaultVariants)
    button: {
      slots: {
        base: "rounded-xl focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/60",
        label: "",
        leadingIcon: "",
        trailingIcon: "",
      },
      defaultVariants: {
        color: "primary",
        variant: "solid",
        size: "lg",
      },
    },

    // UInput theme
    input: {
      slots: {
        base: "rounded-xl border border-neutral-200/60 dark:border-neutral-800/60 text-neutral-900 dark:text-neutral-100 placeholder:text-neutral-500/70 focus-visible:ring-2 focus-visible:ring-primary-500/60 focus:border-primary-500/60",
        leading: "",
        leadingIcon: "",
        trailing: "",
        trailingIcon: "",
      },
      defaultVariants: {
        size: "lg",
      },
    },

    // UCard theme
    card: {
      slots: {
        root: "bg-white/80 dark:bg-neutral-900/70 backdrop-blur border border-neutral-200/60 dark:border-neutral-800/60 rounded-2xl shadow-sm text-neutral-900 dark:text-neutral-100",
        header: "pb-4 border-b border-black/5 dark:border-white/10 text-neutral-900 dark:text-neutral-100",
        body: "p-4 sm:p-6",
        footer: "border-t border-black/5 dark:border-white/10",
      },
    },

    // UModal theme
    modal: {
      slots: {
        overlay: "backdrop-blur-sm",
        wrapper: "",
        content:
          "bg-white/80 dark:bg-neutral-900/70 backdrop-blur border border-neutral-200/60 dark:border-neutral-800/60 rounded-2xl shadow-sm text-neutral-900 dark:text-neutral-100",
        header: "border-b border-black/5 dark:border-white/10",
        body: "",
        footer: "border-t border-black/5 dark:border-white/10",
        title: "",
        description: "",
        close: "",
      },
    },

    // UTooltip theme
    tooltip: {
      slots: {
        content: "backdrop-blur bg-neutral-900/80 text-white",
        arrow: "",
        text: "",
        kbds: "",
        kbdsSize: "",
      },
    },

    // UAlert theme
    alert: {
      slots: {
        root: "rounded-xl border border-neutral-200/60 dark:border-neutral-800/60 bg-white/70 dark:bg-neutral-900/60 backdrop-blur text-neutral-900 dark:text-neutral-100",
        wrapper: "",
        title: "font-medium text-neutral-900 dark:text-neutral-100",
        description: "text-sm text-neutral-600 dark:text-neutral-300",
        icon: "opacity-90",
        avatar: "",
        avatarSize: "",
        actions: "",
        close: "opacity-70 hover:opacity-100",
      },
    },

    // UBadge theme
    badge: {
      slots: {
        base: "rounded-full px-2.5 py-0.5 text-xs font-medium",
        label: "",
        leadingIcon: "",
        trailingIcon: "",
        leadingAvatar: "",
        leadingAvatarSize: "",
      },
    },

    // UAvatar theme
    avatar: {
      slots: {
        root: "rounded-2xl",
        image: "",
        fallback:
          "bg-neutral-200 dark:bg-neutral-800 text-neutral-500 dark:text-neutral-400",
        icon: "",
      },
    },

    // UPopover theme
    popover: {
      slots: {
        content:
          "border border-neutral-200/60 dark:border-neutral-800/60 rounded-xl shadow-sm bg-white/95 dark:bg-neutral-900/90 backdrop-blur text-neutral-900 dark:text-neutral-100",
        arrow: "",
      },
    },

    // UDropdown theme
    dropdown: {
      slots: {
        content:
          "border border-neutral-200/60 dark:border-neutral-800/60 rounded-xl shadow-sm bg-white/95 dark:bg-neutral-900/90 backdrop-blur text-neutral-900 dark:text-neutral-100 p-1",
        item: "rounded-lg px-3 py-2 text-sm hover:bg-neutral-100 dark:hover:bg-neutral-800",
        divider: "my-1 border-t border-neutral-200 dark:border-neutral-700",
      },
    },

    // UTabs theme

    // UTable theme
    table: {
      slots: {
        root: "min-w-full table-fixed text-sm text-neutral-900 dark:text-neutral-100",
        base: "",
        caption: "",
        thead: "text-neutral-500 dark:text-neutral-400",
        tbody: "",
        tr: "border-b border-neutral-200/60 dark:border-neutral-800/60",
        th: "text-left font-medium py-2 px-3",
        td: "py-2.5 px-3",
        tfoot: "",
        progress: "",
        empty: "",
        loading: "",
      },
    },

    // UPagination theme
    pagination: {
      slots: {
        root: "inline-flex items-center gap-1",
        list: "",
        ellipsis: "",
        label: "",
        first:
          "min-w-9 h-9 rounded-lg border border-neutral-200/60 dark:border-neutral-800/60 hover:bg-neutral-100 dark:hover:bg-neutral-800 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/60",
        prev: "min-w-9 h-9 rounded-lg border border-neutral-200/60 dark:border-neutral-800/60 hover:bg-neutral-100 dark:hover:bg-neutral-800 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/60",
        item: "min-w-9 h-9 rounded-lg border border-neutral-200/60 dark:border-neutral-800/60 hover:bg-neutral-100 dark:hover:bg-neutral-800 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/60",
        next: "min-w-9 h-9 rounded-lg border border-neutral-200/60 dark:border-neutral-800/60 hover:bg-neutral-100 dark:hover:bg-neutral-800 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/60",
        last: "min-w-9 h-9 rounded-lg border border-neutral-200/60 dark:border-neutral-800/60 hover:bg-neutral-100 dark:hover:bg-neutral-800 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/60",
      },
    },

    // UToast theme - 配置位置避免遮挡导航栏
    toast: {
      slots: {
        root: "pointer-events-auto relative overflow-hidden rounded-xl border border-neutral-200/60 dark:border-neutral-800/60 bg-white/95 dark:bg-neutral-900/95 backdrop-blur shadow-lg",
        wrapper: "p-4",
        title: "font-semibold text-neutral-900 dark:text-neutral-100",
        description: "text-sm text-neutral-600 dark:text-neutral-300 mt-1",
        close: "absolute right-2 top-2 opacity-70 hover:opacity-100",
        actions: "flex gap-2 mt-3",
      },
      defaultVariants: {
        color: "primary"
      }
    },

    // UFormField theme
    formField: {
      slots: {
        root: "",
        wrapper: "",
        labelWrapper: "",
        label: "text-sm font-medium text-neutral-700 dark:text-neutral-300",
        container: "",
        description: "text-sm text-neutral-600 dark:text-neutral-300",
        error: "text-xs text-red-600 dark:text-red-400",
        hint: "",
        help: "text-xs text-neutral-500 dark:text-neutral-400",
      },
    },
  },
});
