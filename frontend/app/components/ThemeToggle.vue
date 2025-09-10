<template>
  <!-- 主题切换按钮 - 使用 Nuxt UI v3 最佳实践 -->
  <UButton 
    @click="enhancedToggleTheme"
    :size="size" 
    variant="ghost"
    color="neutral"
    square
    class="relative overflow-hidden"
    :class="buttonClass"
  >
    <!-- 太阳图标 (浅色模式) -->
    <UIcon 
      name="i-heroicons-sun" 
      :class="iconClass"
      class="transition-all duration-300 absolute inset-0 m-auto"
      :style="{ 
        opacity: isDark ? 0 : 1, 
        transform: isDark ? 'rotate(90deg) scale(0.8)' : 'rotate(0deg) scale(1)' 
      }"
    />
    
    <!-- 月亮图标 (深色模式) -->
    <UIcon 
      name="i-heroicons-moon" 
      :class="iconClass"
      class="transition-all duration-300 absolute inset-0 m-auto"
      :style="{ 
        opacity: isDark ? 1 : 0, 
        transform: isDark ? 'rotate(0deg) scale(1)' : 'rotate(-90deg) scale(0.8)' 
      }"
    />
    
    <!-- 占位元素保持按钮尺寸 -->
    <div class="opacity-0" :class="iconClass">
      <UIcon name="i-heroicons-sun" :class="iconClass" />
    </div>

    <!-- 切换动画效果 -->
    <div 
      v-if="isToggling"
      class="absolute inset-0 bg-gradient-to-r from-orange-400 to-yellow-400 dark:from-blue-600 dark:to-purple-600 opacity-20 animate-pulse rounded-md"
    />
  </UButton>
</template>

<script setup lang="ts">
interface Props {
  size?: 'xs' | 'sm' | 'md' | 'lg' | 'xl'
}

const props = withDefaults(defineProps<Props>(), {
  size: 'lg'
})

// SSR 安全：提供默认值，客户端挂载后再注入真实实现，避免水合不匹配
const isDark = ref(false)
const isToggling = ref(false)
let toggleTheme: () => void = () => {}

// 根据尺寸计算样式类
const iconClass = computed(() => {
  const sizeMap = {
    xs: 'w-3 h-3',
    sm: 'w-4 h-4', 
    md: 'w-5 h-5',
    lg: 'w-5 h-5',
    xl: 'w-6 h-6'
  }
  return sizeMap[props.size]
})

const buttonClass = computed(() => {
  const sizeMap = {
    xs: 'p-1',
    sm: 'p-1.5',
    md: 'p-2', 
    lg: 'p-2',
    xl: 'p-2.5'
  }
  return sizeMap[props.size]
})

// 增强的切换函数，添加动画效果
const enhancedToggleTheme = async () => {
  if (isToggling.value) return
  
  isToggling.value = true
  
  // 执行主题切换
  toggleTheme()
  
  // 动画持续时间
  setTimeout(() => {
    isToggling.value = false
  }, 300)
}

onMounted(() => {
  try {
    const settingsStore = useSettingsStore()
    
    // 初始化主题监听器
    settingsStore.initThemeWatcher()
    
    // 设置切换函数
    toggleTheme = () => {
      const currentTheme = settingsStore.theme
      if (currentTheme === 'auto') {
        // 从 auto 切换到与当前系统主题相反的主题
        const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches
        settingsStore.setTheme(prefersDark ? 'light' : 'dark')
      } else if (currentTheme === 'light') {
        settingsStore.setTheme('dark')
      } else {
        settingsStore.setTheme('light')
      }
    }
    
    // 监听主题变化
    watch(() => settingsStore.theme, (newTheme) => {
      if (newTheme === 'auto') {
        const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches
        isDark.value = prefersDark
      } else {
        isDark.value = newTheme === 'dark'
      }
    }, { immediate: true })
    
    // 监听系统主题变化（当主题为 auto 时）
    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)')
    const handleMediaChange = () => {
      if (settingsStore.theme === 'auto') {
        isDark.value = mediaQuery.matches
      }
    }
    mediaQuery.addEventListener('change', handleMediaChange)
    
    // 初始设置
    handleMediaChange()
    
    // 组件销毁时清理事件监听器
    onBeforeUnmount(() => {
      mediaQuery.removeEventListener('change', handleMediaChange)
    })
    
  } catch (error) {
    console.warn('Theme system not available, using default light theme:', error)
  }
})

// 导出增强的切换函数
const toggleThemeWithAnimation = enhancedToggleTheme
defineExpose({ toggleTheme: toggleThemeWithAnimation })
</script>