/**
 * 全局通知管理 - 订阅和处理 WebSocket 实时事件
 * 
 * 功能：
 * - 订阅关注、点赞、评论、系统公告等通知事件
 * - 显示 Toast 通知
 * - 管理通知状态
 */

import type {
  NewFollowerEvent,
  MutualFollowEvent,
  MomentLikedEvent,
  MomentCommentedEvent,
  CommentReplyEvent,
  SystemAnnouncementEvent,
  GroupInviteReceivedEvent,
  GroupApplicationReceivedEvent,
  GroupApplicationResponseEvent,
  UserTypingEvent,
} from '~/types/websocket'

// 通知类型定义
export interface NotificationItem {
  id: string
  type: string
  title: string
  description: string
  timestamp: string
  read: boolean
  data?: Record<string, unknown>
}

// 单例状态
let isInitialized = false
const cleanupFunctions: (() => void)[] = []

/**
 * 全局通知管理组合式函数
 */
export function useNotifications() {
  const toast = useToast()
  const { on: wsOn, subscribeToNotifications } = useGlobalWebSocket()
  const authStore = useAuthStore()
  
  // 通知列表（响应式）
  const notifications = useState<NotificationItem[]>('global-notifications', () => [])
  const unreadCount = computed(() => notifications.value.filter(n => !n.read).length)

  /**
   * 添加通知到列表
   */
  const addNotification = (notification: Omit<NotificationItem, 'id' | 'timestamp' | 'read'>) => {
    const newNotification: NotificationItem = {
      ...notification,
      id: `notif_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`,
      timestamp: new Date().toISOString(),
      read: false,
    }
    notifications.value.unshift(newNotification)
    
    // 限制通知列表最多 100 条
    if (notifications.value.length > 100) {
      notifications.value = notifications.value.slice(0, 100)
    }
  }

  /**
   * 标记通知为已读
   */
  const markAsRead = (notificationId: string) => {
    const notification = notifications.value.find(n => n.id === notificationId)
    if (notification) {
      notification.read = true
    }
  }

  /**
   * 标记所有通知为已读
   */
  const markAllAsRead = () => {
    notifications.value.forEach(n => n.read = true)
  }

  /**
   * 清空通知列表
   */
  const clearNotifications = () => {
    notifications.value = []
  }

  /**
   * 初始化通知订阅（只执行一次）
   */
  const initializeSubscriptions = () => {
    if (isInitialized || !authStore.isAuthenticated) return
    isInitialized = true

    console.log('[Notifications] 初始化通知订阅...')

    // 订阅通知类型
    subscribeToNotifications([
      'new_follower',
      'mutual_follow',
      'moment_like',
      'moment_comment',
      'comment_reply',
      'group_invite',
      'group_application',
      'group_application_response',
      'system_announcement',
    ])

    // 监听新关注者事件
    const cleanupNewFollower = wsOn<NewFollowerEvent['payload']>('NewFollower', (data) => {
      const title = data.is_mutual ? '🎉 新好友' : '👤 新关注者'
      const description = data.is_mutual 
        ? `${data.user.nickname} 与你互相关注了，你们现在是好友了！`
        : `${data.user.nickname} 关注了你`
      
      toast.add({
        title,
        description,
        color: 'primary',
        duration: 5000,
      })
      
      addNotification({
        type: 'new_follower',
        title,
        description,
        data: { user: data.user, is_mutual: data.is_mutual },
      })
    })
    cleanupFunctions.push(cleanupNewFollower)

    // 监听互相关注事件
    const cleanupMutualFollow = wsOn<MutualFollowEvent['payload']>('MutualFollow', (data) => {
      toast.add({
        title: '🤝 成为好友',
        description: `你和 ${data.user.nickname} 互相关注了，现在是好友了！`,
        color: 'success',
        duration: 5000,
      })
      
      addNotification({
        type: 'mutual_follow',
        title: '成为好友',
        description: `你和 ${data.user.nickname} 成为了好友`,
        data: { user: data.user },
      })
    })
    cleanupFunctions.push(cleanupMutualFollow)

    // 监听瞬间被点赞事件
    const cleanupMomentLiked = wsOn<MomentLikedEvent['payload']>('MomentLiked', (data) => {
      toast.add({
        title: '❤️ 收到点赞',
        description: `${data.user.nickname} 赞了你的瞬间`,
        color: 'error',
        duration: 4000,
      })
      
      addNotification({
        type: 'moment_liked',
        title: '瞬间被点赞',
        description: `${data.user.nickname} 赞了你的瞬间`,
        data: { moment_id: data.moment_id, user: data.user },
      })
    })
    cleanupFunctions.push(cleanupMomentLiked)

    // 监听瞬间被评论事件
    const cleanupMomentCommented = wsOn<MomentCommentedEvent['payload']>('MomentCommented', (data) => {
      toast.add({
        title: '💬 新评论',
        description: `${data.user.nickname}: ${data.content.substring(0, 50)}${data.content.length > 50 ? '...' : ''}`,
        color: 'info',
        duration: 5000,
      })
      
      addNotification({
        type: 'moment_commented',
        title: '瞬间被评论',
        description: `${data.user.nickname}: ${data.content}`,
        data: { moment_id: data.moment_id, comment_id: data.comment_id, user: data.user },
      })
    })
    cleanupFunctions.push(cleanupMomentCommented)

    // 监听评论被回复事件
    const cleanupCommentReply = wsOn<CommentReplyEvent['payload']>('CommentReply', (data) => {
      toast.add({
        title: '💬 评论被回复',
        description: `${data.user.nickname}: ${data.content.substring(0, 50)}${data.content.length > 50 ? '...' : ''}`,
        color: 'info',
        duration: 5000,
      })
      
      addNotification({
        type: 'comment_reply',
        title: '评论被回复',
        description: `${data.user.nickname} 回复了你的评论`,
        data: { moment_id: data.moment_id, comment_id: data.comment_id, reply_id: data.reply_id, user: data.user },
      })
    })
    cleanupFunctions.push(cleanupCommentReply)

    // 监听系统公告事件
    const cleanupSystemAnnouncement = wsOn<SystemAnnouncementEvent['payload']>('SystemAnnouncement', (data) => {
      const colorMap: Record<string, 'info' | 'warning' | 'error'> = {
        'info': 'info',
        'warning': 'warning',
        'error': 'error',
      }
      
      toast.add({
        title: `📢 ${data.title}`,
        description: data.content,
        color: colorMap[data.severity] || 'info',
        duration: 8000,
      })
      
      addNotification({
        type: 'system_announcement',
        title: data.title,
        description: data.content,
        data: { id: data.id, severity: data.severity, action_url: data.action_url },
      })
    })
    cleanupFunctions.push(cleanupSystemAnnouncement)

    // 监听群组邀请事件
    const cleanupGroupInvite = wsOn<GroupInviteReceivedEvent['payload']>('GroupInviteReceived', (data) => {
      toast.add({
        title: '📨 群组邀请',
        description: `${data.inviter.nickname} 邀请你加入「${data.group_name}」`,
        color: 'primary',
        duration: 6000,
      })
      
      addNotification({
        type: 'group_invite',
        title: '群组邀请',
        description: `${data.inviter.nickname} 邀请你加入「${data.group_name}」`,
        data: { invite_id: data.id, group_chat_id: data.group_chat_id, group_name: data.group_name, inviter: data.inviter },
      })
    })
    cleanupFunctions.push(cleanupGroupInvite)

    // 监听群组申请事件（群主/管理员收到）
    const cleanupGroupApplication = wsOn<GroupApplicationReceivedEvent['payload']>('GroupApplicationReceived', (data) => {
      toast.add({
        title: '📝 入群申请',
        description: `${data.applicant.nickname} 申请加入群组`,
        color: 'warning',
        duration: 5000,
      })
      
      addNotification({
        type: 'group_application',
        title: '入群申请',
        description: `${data.applicant.nickname} 申请加入群组${data.message ? `: ${data.message}` : ''}`,
        data: { application_id: data.id, group_chat_id: data.group_chat_id, applicant: data.applicant },
      })
    })
    cleanupFunctions.push(cleanupGroupApplication)

    // 监听群组申请响应事件（申请者收到）
    const cleanupGroupApplicationResponse = wsOn<GroupApplicationResponseEvent['payload']>('GroupApplicationResponse', (data) => {
      const isAccepted = data.status === 'accepted'
      toast.add({
        title: isAccepted ? '✅ 申请已通过' : '❌ 申请被拒绝',
        description: isAccepted 
          ? `你已成功加入「${data.group_name}」`
          : `你的入群申请被拒绝`,
        color: isAccepted ? 'success' : 'error',
        duration: 5000,
      })
      
      addNotification({
        type: 'group_application_response',
        title: isAccepted ? '入群申请通过' : '入群申请被拒绝',
        description: isAccepted 
          ? `你已成功加入「${data.group_name}」`
          : `你的「${data.group_name}」入群申请被拒绝`,
        data: { application_id: data.id, group_chat_id: data.group_chat_id, group_name: data.group_name, status: data.status },
      })
    })
    cleanupFunctions.push(cleanupGroupApplicationResponse)

    console.log('[Notifications] 通知订阅初始化完成')
  }

  /**
   * 清理订阅
   */
  const cleanup = () => {
    cleanupFunctions.forEach(fn => fn())
    cleanupFunctions.length = 0
    isInitialized = false
    console.log('[Notifications] 通知订阅已清理')
  }

  return {
    // 状态
    notifications: readonly(notifications),
    unreadCount,
    
    // 方法
    initializeSubscriptions,
    cleanup,
    addNotification,
    markAsRead,
    markAllAsRead,
    clearNotifications,
  }
}

