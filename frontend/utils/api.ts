/**
 * API工具类型和函数
 * 提供API调用相关的通用类型和工具函数
 */

import type { PaginatedResponse } from '~/types/api'
import { STATUS } from '~/config'

// API 错误类型
export interface ApiError {
  message: string
  code?: string | number
  details?: any
}

// 导出状态常量
export const ApiStatus = STATUS
