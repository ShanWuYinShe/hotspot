/**
 * 全局快捷键管理
 * 提供统一的快捷键注册和管理功能
 */

export interface ShortcutConfig {
  key: string
  description: string
  handler: () => void
  category?: string
  visible?: boolean // 是否在快捷键帮助中显示
}

export const useGlobalShortcuts = () => {
  const shortcuts = ref<Map<string, ShortcutConfig>>(new Map())
  const isHelpVisible = ref(false)

  // 注册快捷键
  const registerShortcut = (config: ShortcutConfig) => {
    shortcuts.value.set(config.key, config)
  }

  // 注销快捷键
  const unregisterShortcut = (key: string) => {
    shortcuts.value.delete(key)
  }

  // 获取所有快捷键（按分类分组）
  const getShortcutsByCategory = computed(() => {
    const result: Record<string, ShortcutConfig[]> = {}
    
    shortcuts.value.forEach((config) => {
      if (config.visible !== false) {
        const category = config.category || '通用'
        if (!result[category]) {
          result[category] = []
        }
        result[category].push(config)
      }
    })
    
    return result
  })

  // 显示快捷键帮助
  const showHelp = () => {
    isHelpVisible.value = true
  }

  // 隐藏快捷键帮助
  const hideHelp = () => {
    isHelpVisible.value = false
  }

  // 切换快捷键帮助显示状态
  const toggleHelp = () => {
    isHelpVisible.value = !isHelpVisible.value
  }

  // 格式化快捷键显示
  const formatShortcut = (key: string): string[] => {
    const parts = key.split('_')
    const formatted: string[] = []
    
    parts.forEach(part => {
      switch (part) {
        case 'meta':
          formatted.push(navigator.platform.includes('Mac') ? '⌘' : 'Ctrl')
          break
        case 'alt':
          formatted.push(navigator.platform.includes('Mac') ? '⌥' : 'Alt')
          break
        case 'shift':
          formatted.push('⇧')
          break
        case 'ctrl':
          formatted.push('Ctrl')
          break
        default:
          formatted.push(part.toUpperCase())
      }
    })
    
    return formatted
  }

  // 注册全局快捷键监听
  const setupGlobalShortcuts = () => {
    // 注册帮助快捷键
    registerShortcut({
      key: 'meta_slash',
      description: '显示快捷键帮助',
      handler: toggleHelp,
      category: '系统',
      visible: true
    })

    // 使用 Nuxt 的 defineShortcuts
    const dispose = defineShortcuts({
      meta_slash: {
        usingInput: true,
        handler: toggleHelp,
      },
      escape: {
        usingInput: false,
        handler: () => {
          if (isHelpVisible.value) {
            hideHelp()
          }
        },
      },
    })

    return dispose
  }

  return {
    shortcuts: readonly(shortcuts),
    isHelpVisible: readonly(isHelpVisible),
    registerShortcut,
    unregisterShortcut,
    getShortcutsByCategory,
    showHelp,
    hideHelp,
    toggleHelp,
    formatShortcut,
    setupGlobalShortcuts,
  }
}
