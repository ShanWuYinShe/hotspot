/**
 * 视图过渡 Composable
 * 利用 Nuxt v4 的 View Transition API
 */

interface TransitionOptions {
  name?: string
  duration?: number
  easing?: string
  delay?: number
}

export function useViewTransition() {
  // 检查浏览器是否支持 View Transition API
  const isSupported = computed(() => {
    return typeof document !== 'undefined' && 'startViewTransition' in document
  })
  
  // 执行视图过渡
  const startTransition = async (
    callback: () => void | Promise<void>,
    options: TransitionOptions = {}
  ) => {
    if (!isSupported.value) {
      // 如果不支持，直接执行回调
      await callback()
      return
    }
    
    try {
      // 使用 View Transition API
      const transition = (document as any).startViewTransition(async () => {
        await callback()
      })
      
      // 如果提供了自定义选项，应用它们
      if (options.name) {
        transition.name = options.name
      }
      
      await transition.finished
    } catch (error) {
      console.warn('View transition failed:', error)
      // 降级处理
      await callback()
    }
  }
  
  // 页面导航过渡
  const navigateWithTransition = async (
    to: string,
    options: TransitionOptions = {}
  ) => {
    await startTransition(async () => {
      await navigateTo(to)
    }, options)
  }
  
  // 模态框过渡
  const toggleModalWithTransition = async (
    toggleFn: () => void | Promise<void>,
    options: TransitionOptions = { name: 'modal' }
  ) => {
    await startTransition(toggleFn, options)
  }
  
  // 列表项过渡
  const updateListWithTransition = async (
    updateFn: () => void | Promise<void>,
    options: TransitionOptions = { name: 'list-update' }
  ) => {
    await startTransition(updateFn, options)
  }
  
  // 表单提交过渡
  const submitWithTransition = async (
    submitFn: () => void | Promise<void>,
    options: TransitionOptions = { name: 'form-submit' }
  ) => {
    await startTransition(submitFn, options)
  }
  
  return {
    isSupported,
    startTransition,
    navigateWithTransition,
    toggleModalWithTransition,
    updateListWithTransition,
    submitWithTransition
  }
}

// 预定义的过渡效果
export function usePageTransitions() {
  const { startTransition } = useViewTransition()
  
  // 淡入淡出过渡
  const fadeTransition = (callback: () => void | Promise<void>) => {
    return startTransition(callback, {
      name: 'fade',
      duration: 300,
      easing: 'ease-in-out'
    })
  }
  
  // 滑动过渡
  const slideTransition = (callback: () => void | Promise<void>) => {
    return startTransition(callback, {
      name: 'slide',
      duration: 400,
      easing: 'cubic-bezier(0.4, 0, 0.2, 1)'
    })
  }
  
  // 缩放过渡
  const scaleTransition = (callback: () => void | Promise<void>) => {
    return startTransition(callback, {
      name: 'scale',
      duration: 250,
      easing: 'ease-out'
    })
  }
  
  // 弹性过渡
  const bounceTransition = (callback: () => void | Promise<void>) => {
    return startTransition(callback, {
      name: 'bounce',
      duration: 500,
      easing: 'cubic-bezier(0.68, -0.55, 0.265, 1.55)'
    })
  }
  
  return {
    fadeTransition,
    slideTransition,
    scaleTransition,
    bounceTransition
  }
}

// 路由过渡 composable
export function useRouteTransitions() {
  const { navigateWithTransition } = useViewTransition()
  
  // 前进导航（向右滑动）
  const navigateForward = (to: string) => {
    return navigateWithTransition(to, {
      name: 'slide-left',
      duration: 300
    })
  }
  
  // 后退导航（向左滑动）
  const navigateBack = (to?: string) => {
    if (to) {
      return navigateWithTransition(to, {
        name: 'slide-right',
        duration: 300
      })
    } else {
      // 使用浏览器后退
      history.back()
    }
  }
  
  // 向上导航（用于模态框或详情页）
  const navigateUp = (to: string) => {
    return navigateWithTransition(to, {
      name: 'slide-up',
      duration: 400
    })
  }
  
  // 向下导航（关闭模态框或返回）
  const navigateDown = (to: string) => {
    return navigateWithTransition(to, {
      name: 'slide-down',
      duration: 400
    })
  }
  
  return {
    navigateForward,
    navigateBack,
    navigateUp,
    navigateDown
  }
}
