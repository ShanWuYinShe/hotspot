export default defineAppConfig({
  ui: {
    primary: "blue",
    gray: "neutral",
    
    colors: {
      primary: "blue",
      secondary: "slate", 
      neutral: "zinc",
      info: "blue",
      success: "green",
      warning: "orange",
      error: "red",
    },

    button: {
      slots: {
        base: "rounded-lg focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/50 transition-colors duration-150 ease-out font-medium cursor-pointer",
      },
      defaultVariants: {
        color: "primary",
        variant: "solid",
        size: "md",
      },
    },

    card: {
      slots: {
        root: "bg-white dark:bg-neutral-900 border border-neutral-200 dark:border-neutral-700 rounded-lg shadow-sm",
        body: "p-4",
      },
    },

    modal: {
      slots: {
        overlay: "backdrop-blur-sm bg-black/30",
        content: "bg-white dark:bg-neutral-900 rounded-lg shadow-xl border border-neutral-200 dark:border-neutral-700",
        header: "px-4 py-3 border-b border-neutral-200 dark:border-neutral-700",
        body: "p-4",
        footer: "px-4 py-3 border-t border-neutral-200 dark:border-neutral-700 flex gap-2 justify-end",
      },
    },
  }
});