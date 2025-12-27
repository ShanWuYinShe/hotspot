export default defineAppConfig({
  ui: {
    colors: {
      primary: 'emerald',
      neutral: 'slate'
    },

    /* =================================================
     * Global Layout
     * ================================================= */
    dashboardPanel: {
      slots: {
        body:
          'flex-1 overflow-y-auto ' +
          'bg-slate-100 dark:bg-slate-950'
      }
    },

    dashboardNavbar: {
      slots: {
        root:
          'h-(--ui-header-height) flex items-center px-4 gap-3 ' +
          'bg-slate-100 dark:bg-slate-900 ' +
          'border-b border-slate-200 dark:border-slate-800'
      }
    },

    dashboardSidebar: {
      slots: {
        root:
          'bg-slate-100 dark:bg-slate-900 ' +
          'border-r border-slate-200 dark:border-slate-800'
      }
    },

    /* =================================================
     * NavigationMenu
     * ================================================= */
    navigationMenu: {
      slots: {
        link:
          'group w-full flex items-center gap-3 px-4 py-3 ' +
          'rounded-2xl text-sm font-medium ' +
          'text-slate-700 dark:text-slate-300 ' +

          'bg-slate-50 ring-1 ring-slate-200 ' +

          'transition-all duration-150 ' +
          'hover:ring-slate-400 hover:-translate-y-[0.5px] ' +
          'hover:shadow-[0_4px_12px_-6px_rgba(0,0,0,0.15)] ' +

          'active:ring-primary-400 active:translate-y-0 active:shadow-none',

        linkLeadingIcon:
          'size-5 text-slate-400 ' +
          'group-hover:text-slate-700 dark:group-hover:text-white ' +
          'transition-colors'
      },

      variants: {
        active: {
          true: {
            link:
              'ring-2 ring-primary-400 bg-slate-50 ' +
              'text-primary-700 dark:bg-slate-800 dark:text-primary-300',
            linkLeadingIcon: 'text-primary'
          }
        }
      }
    },

    /* =================================================
     * Button
     * ================================================= */
    button: {
      slots: {
        base:
          'inline-flex items-center justify-center gap-2 ' +
          'rounded-full font-medium ' +
          'transition-all duration-150 ' +
          'hover:-translate-y-[0.5px] ' +
          'active:translate-y-0 active:scale-[0.97]'
      },

      compoundVariants: [
        {
          color: 'primary',
          variant: 'solid',
          class: {
            base:
              'bg-primary-500 text-white ' +
              'ring-1 ring-primary-400 ' +
              'hover:ring-primary-500 hover:shadow-md'
          }
        },
        {
          color: 'neutral',
          variant: 'ghost',
          class: {
            base:
              'bg-slate-50 text-slate-700 ' +
              'ring-1 ring-slate-200 ' +
              'hover:ring-slate-400 hover:shadow-sm'
          }
        }
      ]
    },

    /* =================================================
     * Card
     * ================================================= */
    card: {
      slots: {
        root:
          'rounded-3xl bg-slate-50 ' +
          'ring-1 ring-slate-200 ' +
          'transition-all duration-150',
        body: 'p-4'
      },

      variants: {
        variant: {
          interactive: {
            root:
              'cursor-pointer ' +
              'hover:ring-slate-400 hover:-translate-y-[0.5px] ' +
              'hover:shadow-[0_6px_16px_-8px_rgba(0,0,0,0.18)] ' +
              'active:ring-primary-400 active:shadow-none active:translate-y-0'
          }
        }
      }
    },

    /* =================================================
     * PageCard / PageGrid
     * ================================================= */
    pageCard: {
      slots: {
        root:
          'rounded-3xl bg-slate-50 ' +
          'ring-1 ring-slate-200 ' +
          'transition-all duration-150',
        body: 'p-6'
      },

      variants: {
        interactive: {
          true:
            'hover:ring-slate-400 hover:-translate-y-[0.5px] ' +
            'hover:shadow-[0_8px_20px_-10px_rgba(0,0,0,0.18)] ' +
            'active:ring-primary-400 active:shadow-none'
        }
      }
    },

    pageGrid: {
      slots: {
        root: 'grid gap-4 bg-transparent'
      }
    },

    /* =================================================
     * Input / Textarea
     * ================================================= */
    input: {
      slots: {
        base:
          'w-full rounded-full px-4 py-2 ' +
          'bg-slate-50 text-slate-900 ' +
          'ring-1 ring-slate-300 ' +
          'transition-all duration-150 ' +
          'hover:ring-slate-400 ' +
          'focus-visible:ring-2 focus-visible:ring-primary-500'
      }
    },

    textarea: {
      slots: {
        base:
          'w-full rounded-3xl p-3 ' +
          'bg-slate-50 ' +
          'ring-1 ring-slate-300 ' +
          'transition-all duration-150 ' +
          'hover:ring-slate-400 ' +
          'focus-visible:ring-2 focus-visible:ring-primary-500'
      }
    },

    /* =================================================
     * Table（列表 hover 稳定）
     * ================================================= */
    table: {
      slots: {
        tr:
          'transition-all duration-150 ' +
          'hover:bg-slate-50 ' +
          'hover:shadow-sm ' +
          'hover:ring-1 hover:ring-slate-200'
      }
    },

    /* =================================================
     * Modal
     * ================================================= */
    modal: {
      slots: {
        overlay:
          'fixed inset-0 bg-slate-900/40 backdrop-blur-sm',
        content:
          'bg-slate-50 rounded-3xl ' +
          'ring-1 ring-slate-200 shadow-xl'
      }
    }
  }
})
