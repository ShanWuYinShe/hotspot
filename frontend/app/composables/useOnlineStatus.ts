/**
 * 用户在线状态管理 - 遵循 Nuxt 最佳实践
 * 专注于在线状态查询
 */

export function useOnlineStatus() {
  const checkUserOnline = async (userId: string): Promise<boolean> => {
    if (!userId) return false
    
    try {
      const api = useApi()
      const response = await api.post('/protected/user/online_status', { user_id: userId }, { auth: true })
      return response.is_online || false
    } catch (error) {
      return false
    }
  }
  
  const checkMultipleUsers = async (userIds: string[]): Promise<Record<string, boolean>> => {
    if (userIds.length === 0) return {}
    
    try {
      const api = useApi()
      const response = await api.post('/protected/user/batch_online_status', { user_ids: userIds }, { auth: true })
      return response.statuses || {}
    } catch (error) {
      return {}
    }
  }
  
  return {
    checkUserOnline,
    checkMultipleUsers
  }
}
