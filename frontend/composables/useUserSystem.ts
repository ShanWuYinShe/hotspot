import { defineStore } from 'pinia'
import { useAuthStore } from '~/stores/auth'
import { useMessage, useNotification } from 'naive-ui'
import { h } from 'vue'

// 用户权限管理 Store
export const useUserCapabilityStore = defineStore('userCapability', () => {
  const authStore = useAuthStore()

  // 根据用户类型计算权限
  const capabilities = computed(() => {
    const user = authStore.getUser

    if (!user) {
      // 游客权限（未登录）
      return {
        canViewPublicContent: true,
        canCreateGroups: false,
        canSendMessages: false,
        canCreateMoments: false,
        canLikeContent: false,
        canJoinGroups: false,
        maxGroupJoins: 0,
        needsUpgrade: true,
        userType: 'guest' as const
      }
    }

    if (user.is_temp) {
      // 临时用户权限
      return {
        canViewPublicContent: true,
        canCreateGroups: true,
        canSendMessages: true,
        canCreateMoments: true,
        canLikeContent: true,
        canJoinGroups: true,
        maxGroupJoins: 3, // 限制加入群组数量
        hasExpiryReminder: true, // 显示升级提醒
        needsUpgrade: true,
        userType: 'temporary' as const
      }
    }

    // 正式用户权限
    return {
      canViewPublicContent: true,
      canCreateGroups: true,
      canSendMessages: true,
      canCreateMoments: true,
      canLikeContent: true,
      canJoinGroups: true,
      canInviteUsers: true,
      canManageGroups: true,
      maxGroupJoins: Infinity,
      needsUpgrade: false,
      userType: 'regular' as const
    }
  })

  // 检查特定功能权限
  const checkPermission = (feature: string): boolean => {
    switch (feature) {
      case 'createGroup':
        return capabilities.value.canCreateGroups
      case 'sendMessage':
        return capabilities.value.canSendMessages
      case 'createMoment':
        return capabilities.value.canCreateMoments
      case 'likeContent':
        return capabilities.value.canLikeContent
      case 'joinGroup':
        return capabilities.value.canJoinGroups
      default:
        return true
    }
  }

  return {
    capabilities,
    checkPermission
  }
})

// 用户升级引导功能
export const useUserUpgrade = () => {
  const authStore = useAuthStore()
  const { capabilities } = useUserCapabilityStore()

  const showUpgradePrompt = () => {
    return (trigger: 'time' | 'action' | 'limit' = 'action', options?: { message?: any; notification?: any }) => {
      const currentUser = authStore.getUser
      if (!currentUser?.is_temp) return

      const messages = {
        time: '您已使用临时账户一段时间，升级为正式用户可享受更多功能！',
        action: '升级为正式用户，解锁完整的社交功能！',
        limit: '您已达到临时账户限制，升级后可享受无限制使用！'
      }

      options?.notification?.info({
        content: messages[trigger],
        duration: 0, // 不自动关闭
        closable: true,
        action: () => {
          return h(
            'button',
            {
              onClick: () => {
                navigateTo('/auth?mode=upgrade')
              },
              style: {
                color: '#18a058',
                cursor: 'pointer',
                background: 'none',
                border: 'none',
                textDecoration: 'underline'
              }
            },            '立即升级'
          )
        }
      })
    }
  }

  const checkUpgradeOpportunities = () => {
    return (options?: { message?: any; notification?: any }) => {
      const currentUser = authStore.getUser
      if (!currentUser?.is_temp) return

      // 基于使用时长的提醒
      const registerTime = new Date(currentUser.created_at || Date.now()).getTime()
      const hoursSinceRegister = (Date.now() - registerTime) / (1000 * 60 * 60)

      if (hoursSinceRegister > 2) { // 使用超过2小时
        showUpgradePrompt()('time', options)
      }
    }
  }

  return {
    showUpgradePrompt,
    checkUpgradeOpportunities
  }
}

// 临时用户数据清理功能
export const useTempUserCleanup = () => {
  const cleanupExpiredData = () => {
    const tempUserData = localStorage.getItem('temp_user_backup')
    if (tempUserData) {
      try {
        const userData = JSON.parse(tempUserData)
        const createTime = new Date(userData.created_at).getTime()
        const daysSinceCreate = (Date.now() - createTime) / (1000 * 60 * 60 * 24)

        // 临时用户数据超过7天自动清理
        if (daysSinceCreate > 7) {
          localStorage.removeItem('temp_user_backup')
          return true
        }

        return false
      } catch (error) {
        localStorage.removeItem('temp_user_backup')
        return true
      }
    }

    return false
  }

  const clearAllTempUserData = () => {
    // 清理备份数据
    localStorage.removeItem('temp_user_backup')

    // 清理其他可能的临时数据
    const keysToClean = [
      'temp_user_settings',
      'temp_user_preferences',
      'temp_user_cache'
    ]

    keysToClean.forEach(key => {
      localStorage.removeItem(key)
    })
  }

  const getTempUserStats = () => {
    const tempUserData = localStorage.getItem('temp_user_backup')
    if (!tempUserData) return null

    try {
      const userData = JSON.parse(tempUserData)
      const createTime = new Date(userData.created_at).getTime()
      const daysSinceCreate = (Date.now() - createTime) / (1000 * 60 * 60 * 24)

      return {
        nickname: userData.nickname,
        userId: userData.user_id,
        createTime,
        daysSinceCreate: Math.floor(daysSinceCreate)
      }
    } catch (error) {
      return null
    }
  }

  return {
    cleanupExpiredData,
    clearAllTempUserData,
    getTempUserStats
  }
}

// 权限守卫功能
export const usePermissionGuard = () => {
  const capabilityStore = useUserCapabilityStore()
  const { showUpgradePrompt } = useUserUpgrade()

  const requirePermission = () => {
    return (feature: string, showPrompt = true, options?: { message?: any; notification?: any }): boolean => {
      const hasPermission = capabilityStore.checkPermission(feature)

      if (!hasPermission && showPrompt) {
        showUpgradePrompt()('action', options)
      }

      return hasPermission
    }
  }

  const withPermission = () => {
    return async (feature: string, action: () => Promise<void>, options?: { message?: any; notification?: any }) => {
      if (requirePermission()(feature, true, options)) {
        await action()
      }
    }
  }

  return {
    requirePermission,
    withPermission
  }
}
