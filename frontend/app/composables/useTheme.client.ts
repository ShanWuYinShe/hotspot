/**
 * 主题管理 Composable - 客户端专用
 * 自动处理系统主题检测，无需手动环境检查
 */
export const useTheme = () => {
  const settingsStore = useSettingsStore()

  // 系统主题媒体查询
  const systemThemeQuery = window.matchMedia('(prefers-color-scheme: dark)')

  // 计算当前是否为深色主题
  const isDark = computed(() => {
    switch (settingsStore.theme) {
      case 'dark':
        return true
      case 'light':
        return false
      case 'auto':
      default:
        return systemThemeQuery.matches
    }
  })

  // 检测系统主题支持
  const isSupported = 'matchMedia' in window

  // 切换主题逻辑
  const toggleTheme = () => {
    const currentTheme = settingsStore.theme
    
    if (currentTheme === 'auto') {
      // 从 auto 切换到与当前系统主题相反的主题
      settingsStore.setTheme(isDark.value ? 'light' : 'dark')
    } else if (currentTheme === 'light') {
      settingsStore.setTheme('dark')
    } else {
      settingsStore.setTheme('light')
    }
  }

  // 循环切换：light -> dark -> auto
  const cycleTheme = () => {
    const currentTheme = settingsStore.theme
    const nextTheme = {
      light: 'dark',
      dark: 'auto', 
      auto: 'light'
    }[currentTheme] as 'light' | 'dark' | 'auto'
    
    settingsStore.setTheme(nextTheme)
  }

  // 主题图标
  const themeIcon = computed(() => {
    if (settingsStore.theme === 'auto') {
      return isDark.value ? 'i-heroicons-moon' : 'i-heroicons-sun'
    }
    return isDark.value ? 'i-heroicons-moon' : 'i-heroicons-sun'
  })

  // 主题标签
  const themeLabel = computed(() => {
    const labels = {
      light: '浅色主题',
      dark: '深色主题',
      auto: '跟随系统'
    }
    return labels[settingsStore.theme] || '跟随系统'
  })

  return {
    isDark: readonly(isDark),
    themeIcon: readonly(themeIcon),
    themeLabel: readonly(themeLabel),
    isSupported,
    toggleTheme,
    cycleTheme,
    setTheme: settingsStore.setTheme,
    currentTheme: computed(() => settingsStore.theme)
  }
}
