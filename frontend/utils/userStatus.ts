// /Users/liyou/Work/hotspot/frontend/utils/userStatus.ts

/**
 * 用户在线状态工具函数
 * 基于 last_active_at 计算用户在线状态
 */

export type UserOnlineStatus = 'online' | 'away' | 'offline'

/**
 * 根据最后活跃时间计算用户在线状态
 * @param lastActiveAt 最后活跃时间 (ISO string 或 Date)
 * @returns 在线状态
 */
export function calculateUserOnlineStatus(lastActiveAt?: string | Date): UserOnlineStatus {
  if (!lastActiveAt) {
    return 'offline'
  }

  const lastActive = new Date(lastActiveAt)
  const now = new Date()
  const timeDiff = now.getTime() - lastActive.getTime()
  
  // 5分钟内活跃 = 在线
  if (timeDiff <= 5 * 60 * 1000) {
    return 'online'
  }
  
  // 30分钟内活跃 = 离开
  if (timeDiff <= 30 * 60 * 1000) {
    return 'away'
  }
  
  // 超过30分钟 = 离线
  return 'offline'
}

/**
 * 检查用户是否在线
 * @param lastActiveAt 最后活跃时间
 * @returns 是否在线
 */
export function isUserOnline(lastActiveAt?: string | Date): boolean {
  return calculateUserOnlineStatus(lastActiveAt) === 'online'
}

/**
 * 获取在线状态的显示文本
 * @param status 在线状态
 * @returns 显示文本
 */
export function getOnlineStatusText(status: UserOnlineStatus): string {
  switch (status) {
    case 'online':
      return '在线'
    case 'away':
      return '离开'
    case 'offline':
      return '离线'
    default:
      return '未知'
  }
}

/**
 * 获取在线状态对应的CSS类名
 * @param status 在线状态
 * @returns CSS类名
 */
export function getOnlineStatusClass(status: UserOnlineStatus): string {
  switch (status) {
    case 'online':
      return 'status-online'
    case 'away':
      return 'status-away'
    case 'offline':
      return 'status-offline'
    default:
      return 'status-unknown'
  }
}

/**
 * 获取最后活跃时间的友好显示
 * @param lastActiveAt 最后活跃时间
 * @returns 友好显示文本
 */
export function getLastSeenText(lastActiveAt?: string | Date): string {
  if (!lastActiveAt) {
    return '从未在线'
  }

  const lastActive = new Date(lastActiveAt)
  const now = new Date()
  const timeDiff = now.getTime() - lastActive.getTime()
  
  // 小于1分钟
  if (timeDiff < 60 * 1000) {
    return '刚刚活跃'
  }
  
  // 小于1小时
  if (timeDiff < 60 * 60 * 1000) {
    const minutes = Math.floor(timeDiff / (60 * 1000))
    return `${minutes}分钟前活跃`
  }
  
  // 小于1天
  if (timeDiff < 24 * 60 * 60 * 1000) {
    const hours = Math.floor(timeDiff / (60 * 60 * 1000))
    return `${hours}小时前活跃`
  }
  
  // 小于7天
  if (timeDiff < 7 * 24 * 60 * 60 * 1000) {
    const days = Math.floor(timeDiff / (24 * 60 * 60 * 1000))
    return `${days}天前活跃`
  }
  
  // 超过7天，显示具体日期
  return lastActive.toLocaleDateString('zh-CN')
}
