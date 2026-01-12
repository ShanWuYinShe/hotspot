export default defineAppConfig({
  ui: {
    colors: {
      primary: "teal",
      neutral: "slate",
    },

    // Global
    icon: {
      base: "inline-block align-middle shrink-0 text-current",
    },

    // ========================================================================
    // Elements
    // ========================================================================

    // Button: Pill shaped, high interactivity
    button: {
      slots: {
        base: "inline-flex items-center justify-center relative font-medium transition-all duration-200 ease-out active:scale-[0.96] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 focus-visible:ring-offset-2 focus-visible:ring-offset-white dark:focus-visible:ring-offset-neutral-950",
        label: "whitespace-nowrap",
        leadingIcon: "shrink-0",
        trailingIcon: "shrink-0",
      },
      variants: {
        size: {
          xs: { base: "px-3 py-1.5 text-xs gap-1.5", leadingIcon: "size-4", trailingIcon: "size-4" },
          sm: { base: "px-3.5 py-2 text-sm gap-2", leadingIcon: "size-4", trailingIcon: "size-4" },
          md: { base: "px-5 py-2.5 text-sm gap-2", leadingIcon: "size-5", trailingIcon: "size-5" },
          lg: { base: "px-6 py-3 text-base gap-2.5", leadingIcon: "size-5", trailingIcon: "size-5" },
          xl: { base: "px-7 py-3.5 text-base gap-3", leadingIcon: "size-6", trailingIcon: "size-6" },
        },
        variant: {
          solid: "rounded-full shadow-sm hover:shadow-md text-white bg-primary-500 hover:bg-primary-600 disabled:bg-primary-500 disabled:opacity-75",
          outline: "rounded-full ring-1 ring-inset ring-current hover:bg-primary-50/10 dark:hover:bg-primary-950/10 disabled:opacity-75",
          soft: "rounded-full bg-primary-50 dark:bg-primary-950/30 text-primary-500 dark:text-primary-400 hover:bg-primary-100 dark:hover:bg-primary-900/50 disabled:opacity-75",
          subtle: "rounded-full ring-1 ring-inset ring-neutral-300 dark:ring-neutral-700 hover:ring-neutral-400 dark:hover:ring-neutral-600 bg-neutral-50 dark:bg-neutral-900/50 text-neutral-700 dark:text-neutral-200 hover:bg-neutral-100 dark:hover:bg-neutral-800 disabled:opacity-75",
          ghost: "rounded-full text-neutral-700 dark:text-neutral-200 hover:bg-neutral-100 dark:hover:bg-neutral-800 disabled:opacity-75",
          link: "rounded-full text-primary-500 hover:text-primary-600 underline-offset-4 hover:underline disabled:text-primary-500 disabled:no-underline disabled:opacity-75",
        },
        block: { true: { base: "w-full justify-center" } },
        square: { true: { base: "shrink-0" } },
        // Support for ButtonGroup
        group: {
          horizontal: { base: "group-not-only:group-first:rounded-e-none group-not-only:group-last:rounded-s-none group-not-last:group-not-first:rounded-none" },
          vertical: { base: "group-not-only:group-first:rounded-b-none group-not-only:group-last:rounded-t-none group-not-last:group-not-first:rounded-none" }
        }
      },
      compoundVariants: [
        { size: "xs", square: true, class: { base: "size-8 p-1 justify-center" } },
        { size: "sm", square: true, class: { base: "size-9 p-1.5 justify-center" } },
        { size: "md", square: true, class: { base: "size-10 p-2 justify-center" } },
        { size: "lg", square: true, class: { base: "size-11 p-2.5 justify-center" } },
        { size: "xl", square: true, class: { base: "size-12 p-3 justify-center" } },
      ],
      defaultVariants: {
        variant: "solid",
        size: "md",
        color: "primary",
      },
    },

    // Input: Super rounded (2xl), clean focus rings
    input: {
      slots: {
        root: "relative flex items-center w-full",
        base: "relative block w-full disabled:cursor-not-allowed disabled:opacity-75 rounded-2xl transition-all duration-200 placeholder:text-neutral-400 dark:placeholder:text-neutral-500 focus:outline-none autofill:shadow-[inset_0_0_0_1000px_white] dark:autofill:shadow-[inset_0_0_0_1000px_theme('colors.neutral.900')] autofill:text-neutral-900 dark:autofill:text-white",
        leading: "absolute inset-y-0 start-0 flex items-center",
        leadingIcon: "shrink-0 text-neutral-400 dark:text-neutral-500",
        trailing: "absolute inset-y-0 end-0 flex items-center",
        trailingIcon: "shrink-0 text-neutral-400 dark:text-neutral-500",
      },
      variants: {
        variant: {
          outline: "bg-white dark:bg-neutral-900 text-neutral-900 dark:text-white ring-1 ring-neutral-300/80 dark:ring-neutral-700/70 shadow-sm hover:ring-2 hover:ring-neutral-400/80 dark:hover:ring-neutral-500/70 focus:ring-2 focus:ring-primary-500 dark:focus:ring-primary-400",
          soft: "bg-neutral-50/80 dark:bg-neutral-900/55 text-neutral-900 dark:text-white ring-1 ring-transparent hover:ring-neutral-300/80 dark:hover:ring-neutral-700/70 focus:ring-2 focus:ring-primary-500 dark:focus:ring-primary-400",
          subtle: "bg-neutral-50 dark:bg-neutral-900 text-neutral-900 dark:text-white ring-1 ring-neutral-200 dark:ring-neutral-800 hover:ring-neutral-300 dark:hover:ring-neutral-700 focus:ring-2 focus:ring-primary-500 dark:focus:ring-primary-400",
          ghost: "bg-transparent text-neutral-900 dark:text-white ring-0 hover:bg-neutral-50 dark:hover:bg-neutral-900 focus:ring-2 focus:ring-primary-500 dark:focus:ring-primary-400",
          // Clean none variant for nested components (SelectMenu, InputMenu, CommandPalette)
          none: "bg-transparent border-0 ring-0 focus:ring-0 shadow-none focus:shadow-none p-0 rounded-none",
        },
        size: {
          xs: { base: "px-2.5 py-1.5 text-xs", leading: "ps-2.5", trailing: "pe-2.5", leadingIcon: "size-4", trailingIcon: "size-4" },
          sm: { base: "px-3 py-2 text-sm", leading: "ps-3", trailing: "pe-3", leadingIcon: "size-4", trailingIcon: "size-4" },
          md: { base: "px-3.5 py-2.5 text-sm", leading: "ps-3.5", trailing: "pe-3.5", leadingIcon: "size-5", trailingIcon: "size-5" },
          lg: { base: "px-4 py-3 text-base", leading: "ps-4", trailing: "pe-4", leadingIcon: "size-5", trailingIcon: "size-5" },
          xl: { base: "px-5 py-3.5 text-base", leading: "ps-5", trailing: "pe-5", leadingIcon: "size-6", trailingIcon: "size-6" },
        },
        // Support for ButtonGroup/InputGroup
        fieldGroup: {
          horizontal: { base: "group-not-only:group-first:rounded-e-none group-not-only:group-last:rounded-s-none group-not-last:group-not-first:rounded-none" },
          vertical: { base: "group-not-only:group-first:rounded-b-none group-not-only:group-last:rounded-t-none group-not-last:group-not-first:rounded-none" }
        },
        loading: { true: "" },
      },
      compoundVariants: [
        { leading: true, size: "xs", class: "ps-8" }, { leading: true, size: "sm", class: "ps-9" }, { leading: true, size: "md", class: "ps-10" }, { leading: true, size: "lg", class: "ps-11" }, { leading: true, size: "xl", class: "ps-12" },
        { trailing: true, size: "xs", class: "pe-8" }, { trailing: true, size: "sm", class: "pe-9" }, { trailing: true, size: "md", class: "pe-10" }, { trailing: true, size: "lg", class: "pe-11" }, { trailing: true, size: "xl", class: "pe-12" },
        { loading: true, leading: true, class: { leadingIcon: "animate-spin" } },
        { loading: true, leading: false, trailing: true, class: { trailingIcon: "animate-spin" } },
      ],
      defaultVariants: {
        variant: "outline",
        size: "md",
        color: "primary",
      },
    },

    // Textarea: Consistent with Input
    textarea: {
      slots: {
        root: "relative flex items-center w-full",
        base: "relative block w-full disabled:cursor-not-allowed disabled:opacity-75 rounded-2xl transition-all duration-200 placeholder:text-neutral-400 dark:placeholder:text-neutral-500 focus:outline-none resize-none",
      },
      variants: {
        variant: {
          outline: "bg-white dark:bg-neutral-900 text-neutral-900 dark:text-white ring-1 ring-neutral-300/80 dark:ring-neutral-700/70 shadow-sm hover:ring-2 hover:ring-neutral-400/80 dark:hover:ring-neutral-500/70 focus:ring-2 focus:ring-primary-500 dark:focus:ring-primary-400",
          soft: "bg-neutral-50/80 dark:bg-neutral-900/55 text-neutral-900 dark:text-white ring-1 ring-transparent hover:ring-neutral-300/80 dark:hover:ring-neutral-700/70 focus:ring-2 focus:ring-primary-500 dark:focus:ring-primary-400",
          subtle: "bg-neutral-50 dark:bg-neutral-900 text-neutral-900 dark:text-white ring-1 ring-neutral-200 dark:ring-neutral-800 hover:ring-neutral-300 dark:hover:ring-neutral-700 focus:ring-2 focus:ring-primary-500 dark:focus:ring-primary-400",
          ghost: "bg-transparent text-neutral-900 dark:text-white ring-0 hover:bg-neutral-50 dark:hover:bg-neutral-900 focus:ring-2 focus:ring-primary-500 dark:focus:ring-primary-400",
          none: "bg-transparent border-0 ring-0 focus:ring-0 shadow-none focus:shadow-none p-0 rounded-none",
        },
        size: {
          xs: { base: "px-2.5 py-1.5 text-xs" },
          sm: { base: "px-3 py-2 text-sm" },
          md: { base: "px-3.5 py-2.5 text-sm" },
          lg: { base: "px-4 py-3 text-base" },
          xl: { base: "px-5 py-3.5 text-base" },
        },
      },
      defaultVariants: {
        variant: "outline",
        size: "md",
        color: "primary",
      },
    },

    // ========================================================================
    // Form Components
    // ========================================================================

    // Select (Native): Consistent with Input
    select: {
      slots: {
        root: "relative flex items-center w-full",
        base: "relative block w-full disabled:cursor-not-allowed disabled:opacity-75 rounded-2xl transition-all duration-200 placeholder:text-neutral-400 dark:placeholder:text-neutral-500 focus:outline-none appearance-none",
        leading: "absolute inset-y-0 start-0 flex items-center",
        leadingIcon: "shrink-0 text-neutral-400 dark:text-neutral-500",
        trailing: "absolute inset-y-0 end-0 flex items-center pointer-events-none",
        trailingIcon: "shrink-0 text-neutral-400 dark:text-neutral-500",
      },
      variants: {
        variant: {
          outline: "bg-white dark:bg-neutral-900 text-neutral-900 dark:text-white ring-1 ring-neutral-300/80 dark:ring-neutral-700/70 shadow-sm hover:ring-2 hover:ring-neutral-400/80 dark:hover:ring-neutral-500/70 focus:ring-2 focus:ring-primary-500 dark:focus:ring-primary-400",
          soft: "bg-neutral-50/80 dark:bg-neutral-900/55 text-neutral-900 dark:text-white ring-1 ring-transparent hover:ring-neutral-300/80 dark:hover:ring-neutral-700/70 focus:ring-2 focus:ring-primary-500 dark:focus:ring-primary-400",
          subtle: "bg-neutral-50 dark:bg-neutral-900 text-neutral-900 dark:text-white ring-1 ring-neutral-200 dark:ring-neutral-800 hover:ring-neutral-300 dark:hover:ring-neutral-700 focus:ring-2 focus:ring-primary-500 dark:focus:ring-primary-400",
          ghost: "bg-transparent text-neutral-900 dark:text-white ring-0 hover:bg-neutral-50 dark:hover:bg-neutral-900 focus:ring-2 focus:ring-primary-500 dark:focus:ring-primary-400",
          none: "bg-transparent border-0 ring-0 focus:ring-0 shadow-none focus:shadow-none p-0 rounded-none",
        },
        size: {
          xs: { base: "px-2.5 py-1.5 text-xs", leading: "ps-2.5", trailing: "pe-2.5", leadingIcon: "size-4", trailingIcon: "size-4" },
          sm: { base: "px-3 py-2 text-sm", leading: "ps-3", trailing: "pe-3", leadingIcon: "size-4", trailingIcon: "size-4" },
          md: { base: "px-3.5 py-2.5 text-sm", leading: "ps-3.5", trailing: "pe-3.5", leadingIcon: "size-5", trailingIcon: "size-5" },
          lg: { base: "px-4 py-3 text-base", leading: "ps-4", trailing: "pe-4", leadingIcon: "size-5", trailingIcon: "size-5" },
          xl: { base: "px-5 py-3.5 text-base", leading: "ps-5", trailing: "pe-5", leadingIcon: "size-6", trailingIcon: "size-6" },
        },
      },
      compoundVariants: [
        { leading: true, size: "xs", class: "ps-8" }, { leading: true, size: "sm", class: "ps-9" }, { leading: true, size: "md", class: "ps-10" }, { leading: true, size: "lg", class: "ps-11" }, { leading: true, size: "xl", class: "ps-12" },
        { trailing: true, size: "xs", class: "pe-8" }, { trailing: true, size: "sm", class: "pe-9" }, { trailing: true, size: "md", class: "pe-10" }, { trailing: true, size: "lg", class: "pe-11" }, { trailing: true, size: "xl", class: "pe-12" },
      ],
      defaultVariants: {
        variant: "outline",
        size: "md",
        color: "primary",
      },
    },

    // SelectMenu: Coordinate Trigger with Input, Internal Input uses variant: none
    selectMenu: {
      slots: {
        // Trigger Base: Matches Input Outline variant exactly
        base: "relative group flex items-center justify-between w-full disabled:cursor-not-allowed disabled:opacity-75 rounded-2xl transition-all duration-200 focus:outline-none bg-transparent text-neutral-900 dark:text-white ring-1 ring-neutral-300/80 dark:ring-neutral-700/70 hover:ring-2 hover:ring-neutral-400/80 dark:hover:ring-neutral-500/70 focus:ring-2 focus:ring-primary-500 dark:focus:ring-primary-400",

        // Content Dropdown: Consistent with other overlays
        content: "min-w-[12rem] bg-white/95 dark:bg-neutral-900/95 backdrop-blur-xl shadow-xl shadow-neutral-900/10 dark:shadow-black/40 rounded-2xl ring-1 ring-neutral-200/80 dark:ring-neutral-800/70 overflow-hidden flex flex-col",

        // Internal Search Input: Uses Input variant: none, but needs proper styling
        // The input component inside SelectMenu will use variant: none, so we style it here
        input: "[&>input]:h-10 [&>input]:min-h-10 [&>input]:px-3 [&>input]:py-2 [&>input]:text-sm [&>input]:bg-transparent [&>input]:border-0 [&>input]:ring-0 [&>input]:focus:ring-0 [&>input]:rounded-none border-b border-neutral-200 dark:border-neutral-800",

        viewport: "p-1.5",
        label: "text-neutral-500 dark:text-neutral-400 font-semibold px-2 py-1.5 text-xs uppercase tracking-wider",
        item: "rounded-lg transition-colors cursor-pointer p-2 flex items-center gap-2 text-neutral-700 dark:text-neutral-200 data-[highlighted]:bg-primary-50 dark:data-[highlighted]:bg-primary-950/30 data-[highlighted]:text-primary-900 dark:data-[highlighted]:text-primary-100",
        empty: "py-3 px-3 text-sm text-neutral-500 dark:text-neutral-400 text-center",

        leadingIcon: "shrink-0 text-neutral-400 dark:text-neutral-500",
        trailingIcon: "shrink-0 text-neutral-400 dark:text-neutral-500",
      },
      variants: {
        size: {
          xs: { base: "px-2.5 py-1.5 text-xs gap-1.5", leadingIcon: "size-4", trailingIcon: "size-4" },
          sm: { base: "px-3 py-2 text-sm gap-2", leadingIcon: "size-4", trailingIcon: "size-4" },
          md: { base: "px-3.5 py-2.5 text-sm gap-2", leadingIcon: "size-5", trailingIcon: "size-5" },
          lg: { base: "px-4 py-3 text-base gap-2.5", leadingIcon: "size-5", trailingIcon: "size-5" },
          xl: { base: "px-5 py-3.5 text-base gap-3", leadingIcon: "size-6", trailingIcon: "size-6" },
        },
      }
    },

    // InputMenu: Coordinate Trigger with Input, Internal Input uses variant: none
    inputMenu: {
      slots: {
        // Trigger Base: Matches Input Outline variant exactly
        base: "relative group flex items-center justify-between w-full disabled:cursor-not-allowed disabled:opacity-75 rounded-2xl transition-all duration-200 focus:outline-none bg-transparent text-neutral-900 dark:text-white ring-1 ring-neutral-300/80 dark:ring-neutral-700/70 hover:ring-2 hover:ring-neutral-400/80 dark:hover:ring-neutral-500/70 focus:ring-2 focus:ring-primary-500 dark:focus:ring-primary-400",

        // Content Dropdown: Consistent with SelectMenu
        content: "min-w-[12rem] bg-white/95 dark:bg-neutral-900/95 backdrop-blur-xl shadow-xl shadow-neutral-900/10 dark:shadow-black/40 rounded-2xl ring-1 ring-neutral-200/80 dark:ring-neutral-800/70 overflow-hidden flex flex-col",

        viewport: "p-1.5",
        label: "text-neutral-500 dark:text-neutral-400 font-semibold px-2 py-1.5 text-xs uppercase tracking-wider",
        item: "rounded-lg transition-colors cursor-pointer p-2 flex items-center gap-2 text-neutral-700 dark:text-neutral-200 data-[highlighted]:bg-primary-50 dark:data-[highlighted]:bg-primary-950/30 data-[highlighted]:text-primary-900 dark:data-[highlighted]:text-primary-100",
        empty: "py-3 px-3 text-sm text-neutral-500 dark:text-neutral-400 text-center",

        leadingIcon: "shrink-0 text-neutral-400 dark:text-neutral-500",
        trailingIcon: "shrink-0 text-neutral-400 dark:text-neutral-500",
      },
      variants: {
        size: {
          xs: { base: "px-2.5 py-1.5 text-xs gap-1.5", leadingIcon: "size-4", trailingIcon: "size-4" },
          sm: { base: "px-3 py-2 text-sm gap-2", leadingIcon: "size-4", trailingIcon: "size-4" },
          md: { base: "px-3.5 py-2.5 text-sm gap-2", leadingIcon: "size-5", trailingIcon: "size-5" },
          lg: { base: "px-4 py-3 text-base gap-2.5", leadingIcon: "size-5", trailingIcon: "size-5" },
          xl: { base: "px-5 py-3.5 text-base gap-3", leadingIcon: "size-6", trailingIcon: "size-6" },
        },
      }
    },

    // CommandPalette: Optimized for use inside Modal, coordinate with Input style
    commandPalette: {
      slots: {
        // Root: Transparent when inside Modal, no duplicate container styling
        root: "flex flex-col min-h-0 min-w-0",
        // Internal Input: Ensure Input component's root slot (flex items-center) works correctly
        // We ensure the Input root div maintains flex items-center and has proper min-height
        // REMOVED: px-0/py-0 on input to allow default padding for icon spacing (leading icon)
        input: "flex items-center px-4 border-b border-neutral-200 dark:border-neutral-800 [&>div]:flex-1 [&>div]:min-h-12 [&>div>input]:h-12 [&>div>input]:min-h-12 [&>div>input]:text-base [&>div>input]:bg-transparent [&>div>input]:border-0 [&>div>input]:ring-0 [&>div>input]:focus:ring-0 [&>div>input]:rounded-none [&>div>input]:placeholder:text-neutral-400 [&>div>input]:dark:placeholder:text-neutral-500 [&>div>input]:text-neutral-900 [&>div>input]:dark:text-white [&_svg]:shrink-0 [&_svg]:size-5 [&_svg]:text-neutral-400 [&_svg]:dark:text-neutral-500",
        content: "relative overflow-hidden flex flex-col flex-1 min-h-0",
        viewport: "relative scroll-py-2 overflow-y-auto flex-1 focus:outline-none px-2 py-2",
        item: "group relative w-full flex items-center gap-3 px-3 py-2.5 select-none outline-none text-base font-medium text-neutral-700 dark:text-neutral-200 rounded-xl transition-colors duration-200 data-[highlighted]:bg-neutral-100 dark:data-[highlighted]:bg-neutral-800 data-[highlighted]:text-neutral-900 dark:data-[highlighted]:text-white",
        itemLabel: "truncate flex-1",
        itemLeadingIcon: "size-5 shrink-0 text-neutral-400 dark:text-neutral-500 group-data-[highlighted]:text-primary-500 dark:group-data-[highlighted]:text-primary-400 transition-colors",
        group: "p-2",
        label: "px-2 py-1.5 text-xs font-semibold text-neutral-500 dark:text-neutral-400 uppercase tracking-wider",
      },
    },

    // ========================================================================
    // Overlays
    // ========================================================================

    modal: {
      slots: {
        overlay: "backdrop-blur-md bg-neutral-900/40 dark:bg-neutral-950/60 transition-opacity",
        content: "bg-white/95 dark:bg-neutral-900/90 ring-1 ring-neutral-200/80 dark:ring-neutral-800/70 shadow-2xl shadow-neutral-900/10 dark:shadow-black/40 rounded-[2rem] flex flex-col w-11/12 sm:w-5/6 md:w-3/4 lg:w-2/3 xl:w-3/5 max-w-[95vw] transition-all",
        header: "p-5 sm:px-7 sm:py-6 border-b border-neutral-100 dark:border-neutral-800/50",
        body: "p-0 flex-1 overflow-y-auto",
        footer: "p-5 sm:px-7 sm:py-5 bg-neutral-50/50 dark:bg-neutral-900/30 rounded-b-[2rem] mt-auto backdrop-blur-sm",
        // Close button: Uses Button component, inherits Button styles
        close: "absolute top-4 right-4",
      },
    },

    slideover: {
      slots: {
        overlay: "backdrop-blur-sm bg-neutral-900/40 dark:bg-neutral-950/60 transition-opacity",
        content: "bg-white dark:bg-neutral-900 shadow-2xl shadow-neutral-900/10 dark:shadow-black/40 ring-1 ring-neutral-200/80 dark:ring-neutral-800/70 flex flex-col transition-transform",
        header: "p-5 border-b border-neutral-200 dark:border-neutral-800 shrink-0",
        body: "p-5 flex-1 overflow-y-auto",
        footer: "p-5 border-t border-neutral-200 dark:border-neutral-800 bg-neutral-50 dark:bg-neutral-900/50 shrink-0",
      },
      variants: {
        side: {
          right: { content: "w-full sm:w-[28rem] max-w-[100vw]" },
          left: { content: "w-full sm:w-[28rem] max-w-[100vw]" },
          top: { content: "max-h-[50vh] h-auto" },
          bottom: { content: "max-h-[50vh] h-auto" },
        },
      },
    },

    dropdownMenu: {
      slots: {
        content: "min-w-[12rem] w-56 bg-white/95 dark:bg-neutral-900/95 backdrop-blur-xl shadow-xl shadow-neutral-900/10 dark:shadow-black/40 rounded-2xl ring-1 ring-neutral-200/80 dark:ring-neutral-800/70 overflow-hidden p-1.5 flex flex-col",
        item: "group relative w-full flex items-center justify-between select-none outline-none rounded-xl px-2.5 py-2 text-sm transition-colors duration-200 data-[disabled]:cursor-not-allowed data-[disabled]:opacity-50 data-[highlighted]:bg-neutral-100 dark:data-[highlighted]:bg-neutral-800 data-[highlighted]:text-neutral-900 dark:data-[highlighted]:text-white",
        itemLabel: "truncate",
        itemLeadingIcon: "size-4 shrink-0 text-neutral-500 dark:text-neutral-400 group-data-[highlighted]:text-primary-500 dark:group-data-[highlighted]:text-primary-400 transition-colors",
      },
    },

    // ========================================================================
    // Containers
    // ========================================================================

    card: {
      slots: {
        root: "bg-white dark:bg-neutral-900 ring-1 ring-neutral-200 dark:ring-neutral-800 rounded-3xl shadow-sm transition-all duration-200 ease-out hover:shadow-md",
        header: "p-5 sm:p-7 border-b border-neutral-100 dark:border-neutral-800/50",
        body: "p-5 sm:p-7",
        footer: "p-5 sm:p-7 border-t border-neutral-100 dark:border-neutral-800/50 bg-neutral-50/30 dark:bg-neutral-900/30 rounded-b-3xl",
      },
      variants: {
        variant: {
          outline: { root: "bg-transparent ring-1 ring-neutral-200 dark:ring-neutral-800 shadow-none hover:shadow-none hover:bg-neutral-50/50 dark:hover:bg-neutral-900/50" },
          solid: { root: "bg-white dark:bg-neutral-900 shadow-sm hover:shadow-md" },
          soft: { root: "bg-neutral-50/80 dark:bg-neutral-900/50 ring-0 shadow-none hover:bg-neutral-100/80 dark:hover:bg-neutral-900/80" },
          subtle: { root: "bg-neutral-50/40 dark:bg-neutral-900/20 ring-1 ring-neutral-200/50 dark:ring-neutral-800/50 shadow-none" },
        },
      },
      defaultVariants: {
        variant: "solid",
      },
    },

    // ========================================================================
    // Display Components
    // ========================================================================

    // Badge: Pill shaped, consistent with Button
    badge: {
      slots: {
        base: "inline-flex items-center gap-1.5 font-medium transition-all duration-200",
        label: "truncate",
        leadingIcon: "shrink-0",
        leadingAvatar: "shrink-0",
        trailingIcon: "shrink-0",
      },
      variants: {
        size: {
          xs: { base: "px-2 py-0.5 text-xs gap-1", leadingIcon: "size-3", trailingIcon: "size-3" },
          sm: { base: "px-2.5 py-1 text-xs gap-1", leadingIcon: "size-3.5", trailingIcon: "size-3.5" },
          md: { base: "px-3 py-1 text-sm gap-1.5", leadingIcon: "size-4", trailingIcon: "size-4" },
          lg: { base: "px-3.5 py-1.5 text-sm gap-1.5", leadingIcon: "size-4", trailingIcon: "size-4" },
          xl: { base: "px-4 py-2 text-base gap-2", leadingIcon: "size-5", trailingIcon: "size-5" },
        },
        variant: {
          solid: "rounded-full text-white",
          outline: "rounded-full ring-1 ring-inset ring-current bg-transparent",
          soft: "rounded-full",
          subtle: "rounded-full ring-1 ring-inset ring-neutral-300 dark:ring-neutral-700 bg-neutral-50 dark:bg-neutral-900/50 text-neutral-700 dark:text-neutral-200",
        },
      },
      defaultVariants: {
        variant: "solid",
        size: "md",
        color: "primary",
      },
    },

    // Avatar: Circular, consistent with HotSpot style
    avatar: {
      slots: {
        root: "relative inline-flex shrink-0 items-center justify-center overflow-hidden rounded-full bg-neutral-100 dark:bg-neutral-800 text-neutral-900 dark:text-white ring-1 ring-neutral-200 dark:ring-neutral-700",
        image: "size-full object-cover",
        fallback: "size-full flex items-center justify-center font-medium",
        icon: "size-full flex items-center justify-center shrink-0",
      },
      variants: {
        size: {
          "3xs": { root: "size-4", fallback: "text-xs", icon: "size-3" },
          "2xs": { root: "size-5", fallback: "text-xs", icon: "size-3.5" },
          xs: { root: "size-6", fallback: "text-xs", icon: "size-4" },
          sm: { root: "size-8", fallback: "text-sm", icon: "size-4" },
          md: { root: "size-10", fallback: "text-sm", icon: "size-5" },
          lg: { root: "size-12", fallback: "text-base", icon: "size-6" },
          xl: { root: "size-16", fallback: "text-lg", icon: "size-8" },
          "2xl": { root: "size-20", fallback: "text-xl", icon: "size-10" },
          "3xl": { root: "size-24", fallback: "text-2xl", icon: "size-12" },
        },
      },
      defaultVariants: {
        size: "md",
      },
    },

    // Tooltip: Rounded, backdrop blur
    tooltip: {
      slots: {
        content: "z-50 rounded-xl bg-neutral-900 dark:bg-neutral-800 px-3 py-2 text-sm text-white shadow-lg backdrop-blur-sm ring-1 ring-neutral-800 dark:ring-neutral-700",
        arrow: "fill-neutral-900 dark:fill-neutral-800",
        text: "",
        kbds: "gap-1",
        kbdsSize: "text-xs",
      },
    },

    // Alert: Rounded, consistent with Card
    alert: {
      slots: {
        root: "rounded-2xl ring-1 ring-neutral-200 dark:ring-neutral-800 shadow-sm transition-all duration-200",
        wrapper: "flex items-start gap-3 p-4",
        title: "font-semibold text-neutral-900 dark:text-white",
        description: "text-sm text-neutral-600 dark:text-neutral-400 mt-1",
        icon: "shrink-0",
        avatar: "shrink-0",
        actions: "mt-2 flex items-center gap-2",
        close: "shrink-0",
      },
      variants: {
        variant: {
          solid: "",
          outline: "bg-transparent",
          soft: "bg-neutral-50 dark:bg-neutral-900/50",
          subtle: "bg-neutral-50/40 dark:bg-neutral-900/20",
        },
        color: {
          primary: { root: "bg-primary-50 dark:bg-primary-950/30 ring-primary-200 dark:ring-primary-900", title: "text-primary-900 dark:text-primary-100", description: "text-primary-700 dark:text-primary-300" },
          success: { root: "bg-success-50 dark:bg-success-950/30 ring-success-200 dark:ring-success-900", title: "text-success-900 dark:text-success-100", description: "text-success-700 dark:text-success-300" },
          warning: { root: "bg-warning-50 dark:bg-warning-950/30 ring-warning-200 dark:ring-warning-900", title: "text-warning-900 dark:text-warning-100", description: "text-warning-700 dark:text-warning-300" },
          error: { root: "bg-error-50 dark:bg-error-950/30 ring-error-200 dark:ring-error-900", title: "text-error-900 dark:text-error-100", description: "text-error-700 dark:text-error-300" },
          neutral: { root: "bg-neutral-50 dark:bg-neutral-900/50 ring-neutral-200 dark:ring-neutral-800", title: "text-neutral-900 dark:text-white", description: "text-neutral-600 dark:text-neutral-400" },
        },
      },
      defaultVariants: {
        variant: "soft",
        color: "primary",
      },
    },

    // Tabs: Pill shaped triggers, consistent with Button
    tabs: {
      slots: {
        root: "w-full",
        list: "inline-flex items-center gap-2 rounded-full bg-neutral-100 dark:bg-neutral-900 p-1",
        indicator: "absolute rounded-full bg-white dark:bg-neutral-700 shadow-sm transition-all duration-200",
        trigger: "group relative px-4 py-2 text-sm font-medium transition-all duration-200 rounded-full disabled:cursor-not-allowed disabled:opacity-50 data-[disabled]:cursor-not-allowed data-[disabled]:opacity-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500",
        leadingIcon: "shrink-0",
        label: "truncate",
        trailingBadge: "shrink-0",
        content: "mt-4 focus-visible:outline-none",
      },
      variants: {
        variant: {
          pill: { list: "bg-neutral-100 dark:bg-neutral-800", trigger: "data-[active]:text-primary-700 dark:data-[active]:text-primary-300 text-neutral-600 dark:text-neutral-400 hover:text-neutral-900 dark:hover:text-white" },
          link: { list: "border-b border-neutral-200 dark:border-neutral-800", trigger: "border-b-2 border-transparent data-[active]:border-primary-500 data-[active]:text-primary-600 dark:data-[active]:text-primary-400 text-neutral-600 dark:text-neutral-400 hover:text-neutral-900 dark:hover:text-white" },
        },
        size: {
          xs: { trigger: "px-2 py-1 text-xs", leadingIcon: "size-3" },
          sm: { trigger: "px-3 py-1.5 text-xs", leadingIcon: "size-3.5" },
          md: { trigger: "px-4 py-2 text-sm", leadingIcon: "size-4" },
          lg: { trigger: "px-5 py-2.5 text-sm", leadingIcon: "size-4" },
          xl: { trigger: "px-6 py-3 text-base", leadingIcon: "size-5" },
        },
      },
      defaultVariants: {
        variant: "pill",
        size: "md",
        color: "primary",
      },
    },

    // Chip: Circular, for numeric indicators
    chip: {
      slots: {
        root: "relative inline-flex shrink-0",
        base: "inline-flex items-center justify-center rounded-full font-semibold text-white shadow-sm ring-1 ring-transparent",
      },
      variants: {
        size: {
          "3xs": { base: "min-w-[0.875rem] h-[0.875rem] text-[0.625rem] px-0.5" },
          "2xs": { base: "min-w-[1rem] h-[1rem] text-[0.625rem] px-1" },
          xs: { base: "min-w-[1.125rem] h-[1.125rem] text-[0.625rem] px-1" },
          sm: { base: "min-w-[1.25rem] h-[1.25rem] text-[0.75rem] px-1.5" },
          md: { base: "min-w-[1.5rem] h-[1.5rem] text-xs px-2" },
          lg: { base: "min-w-[1.75rem] h-[1.75rem] text-xs px-2.5" },
          xl: { base: "min-w-[2rem] h-[2rem] text-sm px-3" },
          "2xl": { base: "min-w-[2.5rem] h-[2.5rem] text-base px-3.5" },
          "3xl": { base: "min-w-[3rem] h-[3rem] text-lg px-4" },
        },
        color: {
          primary: { base: "bg-primary-500 dark:bg-primary-600" },
          success: { base: "bg-success-500 dark:bg-success-600" },
          warning: { base: "bg-warning-500 dark:bg-warning-600" },
          error: { base: "bg-error-500 dark:bg-error-600" },
          neutral: { base: "bg-neutral-500 dark:bg-neutral-600" },
        },
      },
      defaultVariants: {
        size: "md",
        color: "primary",
      },
    },

    // Skeleton: Smooth animation, rounded
    skeleton: {
      slots: {
        root: "animate-pulse rounded-lg bg-neutral-200 dark:bg-neutral-800",
      },
    },

    // Progress: Rounded, smooth transition
    progress: {
      slots: {
        root: "w-full",
        base: "relative h-2 w-full overflow-hidden rounded-full bg-neutral-200 dark:bg-neutral-800 ring-1 ring-neutral-300 dark:ring-neutral-700",
        indicator: "h-full w-full flex-1 rounded-full bg-primary-500 dark:bg-primary-400 transition-all duration-300 ease-out",
        status: "mt-1 text-xs font-medium text-neutral-600 dark:text-neutral-400",
        steps: "flex items-center gap-1",
        step: "h-2 flex-1 rounded-full bg-neutral-200 dark:bg-neutral-800 ring-1 ring-neutral-300 dark:ring-neutral-700 data-[active]:bg-primary-500 dark:data-[active]:bg-primary-400 transition-colors duration-200",
      },
      variants: {
        size: {
          "2xs": { base: "h-1", indicator: "h-1" },
          xs: { base: "h-1.5", indicator: "h-1.5" },
          sm: { base: "h-2", indicator: "h-2" },
          md: { base: "h-2.5", indicator: "h-2.5" },
          lg: { base: "h-3", indicator: "h-3" },
          xl: { base: "h-4", indicator: "h-4" },
          "2xl": { base: "h-5", indicator: "h-5" },
        },
        color: {
          primary: { indicator: "bg-primary-500 dark:bg-primary-400", step: "data-[active]:bg-primary-500 dark:data-[active]:bg-primary-400" },
          success: { indicator: "bg-success-500 dark:bg-success-400", step: "data-[active]:bg-success-500 dark:data-[active]:bg-success-400" },
          warning: { indicator: "bg-warning-500 dark:bg-warning-400", step: "data-[active]:bg-warning-500 dark:data-[active]:bg-warning-400" },
          error: { indicator: "bg-error-500 dark:bg-error-400", step: "data-[active]:bg-error-500 dark:data-[active]:bg-error-400" },
          neutral: { indicator: "bg-neutral-500 dark:bg-neutral-400", step: "data-[active]:bg-neutral-500 dark:data-[active]:bg-neutral-400" },
        },
      },
      defaultVariants: {
        size: "md",
        color: "primary",
      },
    },

    // ========================================================================
    // Page Components
    // ========================================================================

    // Empty State
    empty: {
      slots: {
        root: "flex flex-col items-center justify-center gap-4 py-12 text-center",
        icon: "size-12 text-neutral-400 dark:text-neutral-500 mb-2",
        title: "text-base font-semibold text-neutral-900 dark:text-white",
        description: "text-sm text-neutral-500 dark:text-neutral-400 max-w-sm mx-auto",
        actions: "flex items-center justify-center gap-2 mt-4",
      },
      variants: {
        variant: {
          naked: { root: "py-0" },
        },
      },
    },

    // File Upload
    fileUpload: {
      slots: {
        root: "w-full",
        base: "relative flex items-center justify-center w-full rounded-2xl border-2 border-dashed border-neutral-300 dark:border-neutral-700 hover:bg-neutral-50 dark:hover:bg-neutral-800/50 transition-colors duration-200 cursor-pointer",
        icon: "size-8 text-neutral-400 dark:text-neutral-500",
        label: "text-sm font-medium text-neutral-700 dark:text-neutral-200 mt-2",
        description: "text-xs text-neutral-500 dark:text-neutral-400 mt-1",
        list: "mt-4 space-y-2",
        item: "flex items-center gap-3 p-3 rounded-xl bg-white dark:bg-neutral-900 ring-1 ring-neutral-200 dark:ring-neutral-800",
      },
    },

    pageCard: {
      slots: {
        root: "relative flex flex-col overflow-hidden rounded-3xl transition-all duration-200 ease-out",
        header: "px-6 py-4 border-b border-neutral-200 dark:border-neutral-800",
        wrapper: "flex flex-col flex-1",
        body: "flex-1 px-6 py-4",
        footer: "px-6 py-4 bg-neutral-50 dark:bg-neutral-900/50 border-t border-neutral-200 dark:border-neutral-800",
        title: "text-lg font-bold text-neutral-900 dark:text-white",
        description: "text-sm text-neutral-500 dark:text-neutral-400 mt-1",
      },
      variants: {
        variant: {
          solid: {
            root: "bg-white dark:bg-neutral-900 shadow-sm hover:shadow-md ring-1 ring-neutral-200 dark:ring-neutral-800",
          },
          outline: {
            root: "bg-transparent ring-1 ring-neutral-200 dark:ring-neutral-800",
          },
          soft: {
            root: "bg-neutral-50 dark:bg-neutral-900/50",
          },
          subtle: {
            root: "bg-neutral-50/50 dark:bg-neutral-900/20 ring-1 ring-neutral-200/50 dark:ring-neutral-800/50",
          },
          ghost: {
            root: "bg-transparent hover:bg-neutral-50 dark:hover:bg-neutral-900/50",
          },
          naked: {
            root: "bg-transparent ring-0 shadow-none hover:shadow-none",
          },
        },
      },
      defaultVariants: {
        variant: "solid",
      },
    },

    // ========================================================================
    // Dashboard Components
    // ========================================================================

    dashboardGroup: {
      base: "fixed inset-0 flex overflow-hidden bg-neutral-50 dark:bg-neutral-950",
    },
    dashboardPanel: {
      slots: {
        root: "relative flex flex-col min-w-0 h-full lg:not-last:border-e lg:not-last:border-neutral-200/80 dark:lg:not-last:border-neutral-800/80 shrink-0 bg-white dark:bg-neutral-900 z-0",
        body: "flex flex-col flex-1 overflow-y-auto px-4 sm:px-6 gap-4 sm:gap-6 py-0 sm:py-0",
        handle: "",
      },
      variants: {
        size: {
          true: {
            root: "w-full lg:w-[var(--width)]",
          },
          false: {
            root: "flex-1",
          },
        },
      },
    },
    dashboardNavbar: {
      slots: {
        root: "h-[var(--ui-header-height)] shrink-0 flex items-center justify-between border-b border-neutral-200/80 dark:border-neutral-800/80 px-4 sm:px-6 gap-1.5 bg-white/80 dark:bg-neutral-900/80 backdrop-blur-md z-10 sticky top-0",
        title: "flex items-center gap-1.5 font-semibold text-neutral-900 dark:text-white truncate",
      },
    },
    dashboardSidebar: {
      slots: {
        root: "relative hidden lg:flex flex-col min-h-svh min-w-16 w-[var(--width)] shrink-0 bg-neutral-50/50 dark:bg-neutral-950/50",
        header: "h-[var(--ui-header-height)] shrink-0 flex items-center gap-1.5 px-4",
        body: "flex flex-col gap-2 flex-1 overflow-y-auto px-3 py-2",
        footer: "shrink-0 flex items-center gap-1.5 px-3 py-2",
      },
    },
  },
});
