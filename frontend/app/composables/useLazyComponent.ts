/**
 * 高级组件懒加载 Composable
 * 提供智能预加载、错误重试、加载状态管理
 */

import type { AsyncComponentLoader, Component } from 'vue'

interface LazyComponentOptions {
  /** 延迟加载时间 (ms) */
  delay?: number
  /** 超时时间 (ms) */
  timeout?: number
  /** 最大重试次数 */
  maxRetries?: number
  /** 是否启用预加载 */
  preload?: boolean
  /** 预加载触发条件 */
  preloadTrigger?: 'hover' | 'visible' | 'idle'
}

interface LazyComponentState {
  loading: Ref<boolean>
  error: Ref<Error | null>
  retryCount: Ref<number>
  component: Ref<Component | null>
}

export function useLazyComponent(
  loader: AsyncComponentLoader,
  options: LazyComponentOptions = {}
): LazyComponentState & {
  load: () => Promise<Component>
  retry: () => Promise<Component>
  preload: () => void
} {
  const {
    delay = 200,
    timeout = 10000,
    maxRetries = 3,
    preload = false,
    preloadTrigger = 'hover'
  } = options

  const loading = ref(false)
  const error = ref<Error | null>(null)
  const retryCount = ref(0)
  const component = ref<Component | null>(null)

  // 组件加载缓存
  const componentCache = new Map<string, Promise<Component>>()
  const cacheKey = loader.toString()

  const loadComponent = async (): Promise<Component> => {
    // 检查缓存
    if (componentCache.has(cacheKey)) {
      return componentCache.get(cacheKey)!
    }

    loading.value = true
    error.value = null

    const loadPromise = new Promise<Component>((resolve, reject) => {
      const timeoutId = setTimeout(() => {
        reject(new Error(`组件加载超时 (${timeout}ms)`))
      }, timeout)

      Promise.resolve(loader())
        .then((comp) => {
          clearTimeout(timeoutId)
          component.value = comp
          resolve(comp)
        })
        .catch((err) => {
          clearTimeout(timeoutId)
          reject(err)
        })
    })

    // 缓存 Promise
    componentCache.set(cacheKey, loadPromise)

    try {
      const comp = await loadPromise
      loading.value = false
      retryCount.value = 0
      return comp
    } catch (err) {
      loading.value = false
      error.value = err as Error
      componentCache.delete(cacheKey) // 失败时清除缓存
      throw err
    }
  }

  const load = async (): Promise<Component> => {
    if (component.value) {
      return component.value
    }

    if (delay > 0) {
      await new Promise(resolve => setTimeout(resolve, delay))
    }

    return loadComponent()
  }

  const retry = async (): Promise<Component> => {
    if (retryCount.value >= maxRetries) {
      throw new Error(`组件加载失败，已达到最大重试次数 (${maxRetries})`)
    }

    retryCount.value++
    componentCache.delete(cacheKey) // 清除缓存以重新加载
    return loadComponent()
  }

  const preloadComponent = () => {
    if (!component.value && !loading.value) {
      loadComponent().catch(() => {
        // 预加载失败时静默处理
      })
    }
  }

  // 自动预加载
  if (preload) {
    if (preloadTrigger === 'idle') {
      // 浏览器空闲时预加载
      if ('requestIdleCallback' in window) {
        requestIdleCallback(preloadComponent)
      } else {
        setTimeout(preloadComponent, 100)
      }
    } else if (preloadTrigger === 'visible') {
      // 组件可见时预加载（需要配合 Intersection Observer）
      onMounted(() => {
        preloadComponent()
      })
    }
  }

  return {
    loading: readonly(loading),
    error: readonly(error),
    retryCount: readonly(retryCount),
    component: readonly(component),
    load,
    retry,
    preload: preloadComponent
  }
}

/**
 * 创建懒加载组件的便捷函数
 */
export function defineLazyComponent(
  loader: AsyncComponentLoader,
  options?: LazyComponentOptions
) {
  return defineAsyncComponent({
    loader,
    delay: options?.delay || 200,
    timeout: options?.timeout || 10000,
    errorComponent: defineComponent({
      props: {
        error: Object,
        retry: Function
      },
      setup(props) {
        return () => h('div', {
          class: 'flex flex-col items-center justify-center p-8 text-center'
        }, [
          h('div', {
            class: 'w-16 h-16 mb-4 rounded-full bg-red-100 dark:bg-red-900/20 flex items-center justify-center'
          }, [
            h('svg', {
              class: 'w-8 h-8 text-red-500',
              fill: 'none',
              stroke: 'currentColor',
              viewBox: '0 0 24 24'
            }, [
              h('path', {
                'stroke-linecap': 'round',
                'stroke-linejoin': 'round',
                'stroke-width': '2',
                d: 'M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L3.732 16.5c-.77.833.192 2.5 1.732 2.5z'
              })
            ])
          ]),
          h('h3', {
            class: 'text-lg font-semibold text-gray-900 dark:text-gray-100 mb-2'
          }, '组件加载失败'),
          h('p', {
            class: 'text-gray-600 dark:text-gray-400 mb-4'
          }, props.error?.message || '未知错误'),
          h('button', {
            class: 'px-4 py-2 bg-primary-500 text-white rounded-lg hover:bg-primary-600 transition-colors',
            onClick: props.retry
          }, '重试')
        ])
      }
    }),
    loadingComponent: defineComponent({
      setup() {
        return () => h('div', {
          class: 'flex items-center justify-center p-8'
        }, [
          h('div', {
            class: 'animate-spin rounded-full h-8 w-8 border-b-2 border-primary-500'
          })
        ])
      }
    })
  })
}

/**
 * 路由级别的代码分割
 */
export function defineAsyncPage(loader: AsyncComponentLoader) {
  return defineLazyComponent(loader, {
    delay: 0,
    timeout: 15000,
    preload: true,
    preloadTrigger: 'idle'
  })
}
