/**
 * 简化的快捷键管理 - 功能完整但代码简洁
 */

interface Shortcut {
  key: string
  handler: () => void
  description: string
  category: string
}

export function useShortcuts() {
  const shortcuts = ref<Shortcut[]>([])
  const isHelpVisible = ref(false)

  const register = (key: string, handler: () => void, description: string, category = '通用') => {
    shortcuts.value.push({ key, handler, description, category })
  }

  const unregister = (key: string) => {
    const index = shortcuts.value.findIndex(s => s.key === key)
    if (index > -1) {
      shortcuts.value.splice(index, 1)
    }
  }

  const cleanup = () => {
    shortcuts.value = []
  }

  const getByCategory = () => {
    const categories: Record<string, Shortcut[]> = {}
    shortcuts.value.forEach(shortcut => {
      if (!categories[shortcut.category]) {
        categories[shortcut.category] = []
      }
      categories[shortcut.category].push(shortcut)
    })
    return categories
  }

  const showHelp = () => {
    isHelpVisible.value = true
  }

  const hideHelp = () => {
    isHelpVisible.value = false
  }

  const toggleHelp = () => {
    isHelpVisible.value = !isHelpVisible.value
  }

  const formatShortcut = (key: string) => {
    return key.replace(/\+/g, ' + ').replace('meta', '⌘').replace('alt', '⌥').replace('ctrl', '⌃')
  }

  const formatShortcutKeys = (key: string) => {
    return key.split('+').map(k => k.trim().replace('meta', '⌘').replace('alt', '⌥').replace('ctrl', '⌃'))
  }

  // 设置基本快捷键
  const setupShortcuts = () => {
    return defineShortcuts({
      'meta_slash': {
        usingInput: true,
        handler: toggleHelp,
      },
      'escape': {
        usingInput: false,
        handler: hideHelp,
      },
    })
  }

  return {
    shortcuts: readonly(shortcuts),
    isHelpVisible: readonly(isHelpVisible),
    register,
    unregister,
    cleanup,
    getByCategory,
    showHelp,
    hideHelp,
    toggleHelp,
    formatShortcut,
    formatShortcutKeys,
    setupShortcuts
  }
}