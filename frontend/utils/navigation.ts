/**
 * 导航工具函数
 * 
 * 从 useNavigation composable 迁移而来的简化版本
 * 将页面级别的导航逻辑移动到这里作为工具函数
 */

import type { RouteLocationRaw } from 'vue-router'

/**
 * 安全导航函数，带有错误处理
 */
export function safeNavigate(
  router: any,
  to: RouteLocationRaw,
  options: {
    replace?: boolean
    external?: boolean
    onError?: (error: Error) => void
  } = {}
) {
  try {
    if (options.external && typeof to === 'string') {
      // 外部链接
      window.open(to, '_blank')
      return
    }

    // 内部导航
    if (options.replace) {
      router.replace(to)
    } else {
      router.push(to)
    }
  } catch (error) {
    console.error('导航失败:', error)
    options.onError?.(error as Error)
  }
}

/**
 * 带有确认的导航
 */
export function confirmNavigate(
  router: any,
  to: RouteLocationRaw,
  message: string = '确定要离开当前页面吗？'
) {
  if (confirm(message)) {
    safeNavigate(router, to)
  }
}

/**
 * 条件导航 - 根据条件决定是否导航
 */
export function conditionalNavigate(
  router: any,
  to: RouteLocationRaw,
  condition: boolean | (() => boolean),
  fallback?: () => void
) {
  const shouldNavigate = typeof condition === 'function' ? condition() : condition
  
  if (shouldNavigate) {
    safeNavigate(router, to)
  } else {
    fallback?.()
  }
}

/**
 * 获取当前路由信息的工具函数
 */
export function getCurrentRouteInfo(route: any) {
  return {
    path: route.path,
    name: route.name,
    params: route.params,
    query: route.query,
    hash: route.hash,
    fullPath: route.fullPath
  }
}

/**
 * 检查是否为特定路径
 */
export function isCurrentPath(route: any, targetPath: string | string[]): boolean {
  const paths = Array.isArray(targetPath) ? targetPath : [targetPath]
  return paths.some(path => route.path.startsWith(path))
}

/**
 * 生成带参数的路由
 */
export function buildRoute(
  name: string,
  params: Record<string, any> = {},
  query: Record<string, any> = {}
): RouteLocationRaw {
  return {
    name,
    params,
    query
  }
}

/**
 * 页面返回功能
 */
export function goBack(router: any, fallbackRoute: RouteLocationRaw = '/') {
  if (window.history.length > 1) {
    router.back()
  } else {
    safeNavigate(router, fallbackRoute)
  }
}

/**
 * 面包屑导航生成器
 */
export function generateBreadcrumbs(route: any, routeConfig: Record<string, string>) {
  const breadcrumbs = []
  const pathSegments = route.path.split('/').filter(Boolean)
  
  let currentPath = ''
  for (const segment of pathSegments) {
    currentPath += `/${segment}`
    if (routeConfig[currentPath]) {
      breadcrumbs.push({
        path: currentPath,
        name: routeConfig[currentPath],
        active: currentPath === route.path
      })
    }
  }
  
  return breadcrumbs
}
