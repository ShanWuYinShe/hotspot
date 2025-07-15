import { useUserManagerStore } from '~/stores/userManager'
import { useAuthStore } from '~/stores/auth'
import { h } from 'vue'

export const useMultiUserSystem = () => {
  const userManager = useUserManagerStore()
  const authStore = useAuthStore()
  const router = useRouter()

  // 初始化多用户系统
  const initializeSystem = () => {
    return async (options?: { message?: any; notification?: any }) => {
      if (!process.client) return

      console.log('🚀 初始化多用户系统...')

      try {
        // 1. 初始化用户管理器
        userManager.initialize()
        
        // 2. 自动选择用户
        const user = await userManager.autoSelectUser()
        
        if (user) {
          if (user.is_temporary) {
            console.log(`✅ 已使用临时用户: ${user.nickname} (ID: ${user.user_id})`)
            
            // 显示临时用户欢迎消息
            options?.notification?.success({
              title: '欢迎使用 HotSpot！',
              content: `已为您创建临时账户 "${user.nickname}"，您可以立即开始探索`,
              duration: 4000
            })
          } else {
            console.log(`✅ 已登录用户: ${user.nickname} (ID: ${user.user_id})`)
            
            options?.message?.success(`欢迎回来，${user.nickname}！`)
          }
        } else {
          console.warn('⚠️ 无法创建或加载任何用户')
        }
      } catch (error) {
        console.error('❌ 多用户系统初始化失败:', error)
        
        // 降级处理
        options?.message?.info('欢迎使用 HotSpot！您可以浏览公开内容或手动登录')
      }
    }
  }

  // 确保受保护路由有用户身份
  const ensureUserForProtectedRoutes = () => {
    return async (options?: { message?: any }) => {
      const protectedRoutes = ['/profile', '/my-groups', '/settings', '/chat']
      
      // 如果当前路由需要用户身份但没有用户，尝试自动创建
      const currentPath = router.currentRoute.value.path
      const requiresUser = protectedRoutes.some(path => currentPath.startsWith(path))
      
      if (requiresUser && !authStore.isAuthenticated) {
        console.log('🔐 受保护页面需要用户身份，尝试自动选择用户...')
        
        try {
          const user = await userManager.autoSelectUser()
          if (user) {
            console.log(`✅ 为受保护页面自动选择用户: ${user.nickname}`)
            return user
          }
        } catch (error) {
          console.error('❌ 为受保护页面创建用户失败:', error)
        }
        
        // 如果仍然没有用户，重定向到登录页面
        options?.message?.warning('此页面需要登录，正在跳转到登录页面...')
        await router.push('/auth')
        return null
      }
      
      return authStore.getUser
    }
  }

  // 设置路由守卫
  const setupRouteGuards = () => {
    router.beforeEach(async (to) => {
      const protectedRoutes = ['/profile', '/my-groups', '/settings', '/chat']
      const requiresUser = protectedRoutes.some(path => to.path.startsWith(path))
      
      if (requiresUser && !authStore.isAuthenticated) {
        console.log(`🔐 路由 ${to.path} 需要用户身份`)
        
        try {
          await userManager.autoSelectUser()
          
          if (!authStore.isAuthenticated) {
            console.log('❌ 无法获取用户身份，重定向到登录页面')
            return '/auth'
          }
        } catch (error) {
          console.error('❌ 路由守卫中获取用户身份失败:', error)
          return '/auth'
        }
      }
      
      return true
    })
  }

  // 切换用户
  const switchUser = () => {
    return async (userId: number, options?: { message?: any }) => {
      try {
        const user = await userManager.switchToUser(userId)
        
        options?.message?.success(`已切换到用户: ${user.nickname}`)
        
        // 触发页面刷新或更新
        await router.push('/')
        
        return user
      } catch (error: any) {
        options?.message?.error(error.message || '切换用户失败')
        throw error
      }
    }
  }

  // 添加新用户（通过登录）
  const addUserByLogin = () => {
    return async (credentials: { login_id: string; password: string }, options?: { message?: any }) => {
      try {
        const user = await userManager.addUserByLogin(credentials)
        
        options?.message?.success(`登录成功！欢迎 ${user.nickname}`)
        
        return user
      } catch (error: any) {
        options?.message?.error(error.message || '登录失败')
        throw error
      }
    }
  }

  // 添加新用户（通过注册）
  const addUserByRegister = () => {
    return async (userData: { login_id: string; nickname: string; password: string }, options?: { message?: any }) => {
      try {
        const user = await userManager.addUserByRegister(userData)
        
        options?.message?.success(`注册成功！欢迎加入 HotSpot，${user.nickname}！`)
        
        return user
      } catch (error: any) {
        options?.message?.error(error.message || '注册失败')
        throw error
      }
    }
  }

  // 创建新的临时用户
  const createNewTempUser = () => {
    return async (nickname?: string, options?: { message?: any }) => {
      try {
        const user = await userManager.createTempUser(nickname)
        
        options?.message?.success(`临时用户创建成功！欢迎 ${user.nickname}`)
        
        return user
      } catch (error: any) {
        options?.message?.error(error.message || '创建临时用户失败')
        throw error
      }
    }
  }

  // 移除用户账户
  const removeUser = () => {
    return async (userId: number, options?: { message?: any }) => {
      try {
        await userManager.removeAccount(userId)
        
        options?.message?.success('用户账户已移除')
      } catch (error: any) {
        options?.message?.error(error.message || '移除用户失败')
        throw error
      }
    }
  }

  // 获取当前用户信息
  const getCurrentUser = () => {
    return userManager.currentUser
  }

  // 获取所有用户账户
  const getAllUsers = () => {
    return userManager.accounts
  }

  // 获取最近使用的用户
  const getRecentUsers = () => {
    return userManager.recentUsers
  }

  // 生成友好的临时用户昵称
  const generateTempNickname = (): string => {
    const adjectives = ['探索', '漫游', '发现', '寻找', '遇见', '追寻', '冒险', '游历']
    const nouns = ['者', '客', '人', '员', '师', '家']
    const randomAdjective = adjectives[Math.floor(Math.random() * adjectives.length)]
    const randomNoun = nouns[Math.floor(Math.random() * nouns.length)]
    const randomNum = Math.floor(Math.random() * 9999) + 1
    
    return `${randomAdjective}${randomNoun}${randomNum}`
  }

  // 清理过期用户
  const cleanupExpiredUsers = () => {
    userManager.cleanupExpiredAccounts()
  }

  // 检查是否有用户
  const hasUsers = () => {
    return userManager.hasUsers
  }

  // 显示升级提示
  const showUpgradePrompt = () => {
    return (reason: 'time' | 'action' | 'limit' = 'action', options?: { notification?: any }) => {
      const currentUser = userManager.currentUser
      if (!currentUser?.is_temporary) return

      const messages = {
        time: '您已使用临时账户一段时间，升级为正式用户可享受更多功能！',
        action: '升级为正式用户，解锁完整的社交功能！',
        limit: '您已达到临时账户限制，升级后可享受无限制使用！'
      }

      options?.notification?.info({
        title: '升级账户',
        content: messages[reason],
        duration: 8000,
        closable: true,
        action: () => {
          return h(
            'button',
            {
              onClick: () => {
                router.push('/auth?mode=upgrade')
              },
              style: {
                color: '#18a058',
                cursor: 'pointer',
                background: 'none',
                border: 'none',
                textDecoration: 'underline'
              }
            },
            '立即升级'
          )
        }
      })
    }
  }

  // 检查升级机会
  const checkUpgradeOpportunities = () => {
    return (options?: { notification?: any }) => {
      const currentUser = userManager.currentUser
      if (!currentUser?.is_temporary) return

      // 基于使用时长的提醒
      const lastUsed = new Date(currentUser.last_used).getTime()
      const hoursActive = (Date.now() - lastUsed) / (1000 * 60 * 60)
      
      if (hoursActive > 2) { // 使用超过2小时
        setTimeout(() => {
          showUpgradePrompt()('time', options)
        }, 30000) // 30秒后显示提醒
      }
    }
  }

  return {
    // 初始化
    initializeSystem,
    setupRouteGuards,
    
    // 用户管理
    switchUser,
    addUserByLogin,
    addUserByRegister,
    createNewTempUser,
    removeUser,
    
    // 信息获取
    getCurrentUser,
    getAllUsers,
    getRecentUsers,
    hasUsers,
    
    // 工具方法
    generateTempNickname,
    cleanupExpiredUsers,
    ensureUserForProtectedRoutes,
    
    // 升级相关
    showUpgradePrompt,
    checkUpgradeOpportunities,
    
    // Store访问
    userManager,
    authStore
  }
} 