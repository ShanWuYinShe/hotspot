/**
 * API服务工厂
 * 使用 Composition API 格式
 */

import { 
  useApiRequest,
  useSafeApiRequest
} from './core/api.service'

/**
 * 获取API服务相关的Composition API函数
 * 这个函数应该在Vue组件、composables或Nuxt插件中调用
 */
export function useApiService() {
  return {
    request: useApiRequest(),
    safeRequest: useSafeApiRequest()
  }
}
