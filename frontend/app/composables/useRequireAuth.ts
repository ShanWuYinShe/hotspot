/**
 * 功能级认证检查 Composable
 * 
 * 设计理念：
 * - 游客可以浏览所有公开内容（动态、用户、群组等）
 * - 只有在执行操作时（点赞、评论、发送消息等）才要求登录
 * - 提供友好的登录引导，而不是直接阻止访问
 * 
 * 使用场景：
 * - 点赞按钮
 * - 评论按钮
 * - 发送消息按钮
 * - 关注按钮
 * - 发布动态按钮
 * - 加入群组按钮
 * 
 * @example
 * ```vue
 * <script setup>
 * const { requireAuth } = useRequireAuth()
 * 
 * const handleLike = () => {
 *   if (!requireAuth('点赞')) return
 *   // 执行点赞逻辑
 * }
 * </script>
 * ```
 */

export function useRequireAuth() {
  const authStore = useAuthStore()
  const router = useRouter()
  const route = useRoute()
  const modal = useModal()

  /**
   * 检查是否已认证，未认证则引导登录
   * 
   * @param actionName 操作名称（用于提示）
   * @param options 配置选项
   * @returns 是否已认证
   */
  const requireAuth = (
    actionName: string = '执行此操作',
    options: {
      showModal?: boolean  // 是否显示模态框（默认true）
      redirectAfterLogin?: boolean  // 登录后是否回到当前页（默认true）
    } = {}
  ): boolean => {
    const { showModal = true, redirectAfterLogin = true } = options

    // 已登录，直接放行
    if (authStore.isAuthenticated) {
      return true
    }

    // 未登录，引导登录
    if (showModal) {
      // 使用模态框引导（更友好）
      modal.showLoginPrompt()
    } else {
      // 直接跳转到登录页
      const redirectPath = redirectAfterLogin ? route.fullPath : '/'
      router.push({
        path: '/auth/login',
        query: {
          redirect: redirectPath,
          reason: 'action_required',
          action: actionName
        }
      })
    }

    return false
  }

  /**
   * 检查是否已认证（不引导登录，仅返回状态）
   */
  const isAuthenticated = computed(() => authStore.isAuthenticated)

  /**
   * 批量检查操作权限
   * 
   * @example
   * ```ts
   * const canLike = checkPermission('like')
   * const canComment = checkPermission('comment')
   * ```
   */
  const checkPermission = (action: string): boolean => {
    // 游客可以浏览，但不能操作
    const guestAllowedActions: string[] = []
    
    if (guestAllowedActions.includes(action)) {
      return true
    }

    return authStore.isAuthenticated
  }

  return {
    requireAuth,
    isAuthenticated,
    checkPermission
  }
}



