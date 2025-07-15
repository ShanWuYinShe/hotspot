/**
 * API related type definitions
 */

// 分页参数
export interface Pagination {
  limit?: number
  cursor?: number
}

// 分页元数据
export interface PaginationMeta {
  has_more: boolean
  next_cursor?: number
}

// 分页响应
export interface PaginatedResponse<T = any> {
  items: T[]
  has_more: boolean
  next_cursor?: number
}
