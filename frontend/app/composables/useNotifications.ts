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
  UnfollowedEvent,
  MomentLikedEvent,
  MomentCommentedEvent,
  CommentReplyEvent,
  SystemAnnouncementEvent,
  GroupInviteReceivedEvent,
  GroupInviteResponseEvent,
  GroupApplicationReceivedEvent,
  GroupApplicationResponseEvent,
  UserTypingEvent,
} from '~/types/websocket'
import { webSocketService } from '~/services/websocket.service'

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

const NOTIFICATION_SOURCE_IDS = [
  'new_follower',
  'mutual_follow',
  'unfollowed',
  'moment_like',
  'moment_comment',
  'comment_reply',
  'group_invite',
  'group_invite_response',
  'group_application',
  'group_application_response',
  'system_announcement',
] as const

function trunc(s: string, n: number) {
  if (s.length <= n) return s
  return `${s.substring(0, n)}...`
}

function createNotificationId() {
  return `notif_${Date.now()}_${Math.random().toString(36).slice(2, 11)}`
}

function pushNotification(
  notifications: Ref<NotificationItem[]>,
  notification: Omit<NotificationItem, 'id' | 'timestamp' | 'read'>
) {
  const item: NotificationItem = {
    ...notification,
    id: createNotificationId(),
    timestamp: new Date().toISOString(),
    read: false,
  }

  notifications.value.unshift(item)
  if (notifications.value.length > 100) notifications.value = notifications.value.slice(0, 100)
}

/**
 * 全局通知管理组合式函数
 */
export function useNotifications() {
  const feedback = useFeedbackStore()
  const authStore = useAuthStore()
  
  // 通知列表（响应式）
  const notifications = useState<NotificationItem[]>('global-notifications', () => [])
  const unreadCount = computed(() => notifications.value.filter(n => !n.read).length)

  /**
   * 添加通知到列表
   */
  const addNotification = (notification: Omit<NotificationItem, 'id' | 'timestamp' | 'read'>) => {
    pushNotification(notifications, notification)
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
    webSocketService.subscribe([{
      source_type: 'notification',
      source_ids: [...NOTIFICATION_SOURCE_IDS]
    }])

    const cleanupNewFollower = webSocketService.on('NewFollower', (data) => {
      const title = data.is_mutual ? '🎉 新好友' : '👤 新关注者'
      const description = data.is_mutual
        ? `${data.user.nickname} 与你互相关注了，你们现在是好友了！`
        : `${data.user.nickname} 关注了你`

      // 统一使用 GlobalFeedback（移动端更友好）
      if (data.is_mutual) feedback.success(title, description, 5000)
      else feedback.info(title, description, 5000)
      addNotification({ type: 'new_follower', title, description, data: { user: data.user, is_mutual: data.is_mutual } })
    })
    cleanupFunctions.push(cleanupNewFollower)

    const cleanupMutualFollow = webSocketService.on('MutualFollow', (data) => {
      feedback.success('🤝 成为好友', `你和 ${data.user.nickname} 互相关注了，现在是好友了！`, 5000)
      addNotification({
        type: 'mutual_follow',
        title: '成为好友',
        description: `你和 ${data.user.nickname} 成为了好友`,
        data: { user: data.user },
      })
    })
    cleanupFunctions.push(cleanupMutualFollow)

    const cleanupUnfollowed = webSocketService.on('Unfollowed', (data) => {
      // 这是“有人取消关注我”的通知；若之前是好友，语义是“好友关系解除”
      if (data.was_mutual) {
        feedback.warning('👋 好友关系解除', `对方已取消关注，你们不再是好友`, 5000)
      } else {
        feedback.info('👋 取消关注', `有人取消关注了你`, 4000)
      }
      addNotification({
        type: 'unfollowed',
        title: data.was_mutual ? '好友关系解除' : '取消关注',
        description: data.was_mutual ? '对方已取消关注，你们不再是好友' : '有人取消关注了你',
        data: { user_id: data.user_id, was_mutual: data.was_mutual },
      })
    })
    cleanupFunctions.push(cleanupUnfollowed)

    const cleanupMomentLiked = webSocketService.on('MomentLiked', (data) => {
      feedback.info('❤️ 收到点赞', `${data.user.nickname} 赞了你的瞬间`, 4000)
      addNotification({
        type: 'moment_liked',
        title: '瞬间被点赞',
        description: `${data.user.nickname} 赞了你的瞬间`,
        data: { moment_id: data.moment_id, user: data.user },
      })
    })
    cleanupFunctions.push(cleanupMomentLiked)

    const cleanupMomentCommented = webSocketService.on('MomentCommented', (data) => {
      feedback.info('💬 新评论', `${data.user.nickname}: ${trunc(data.content, 50)}`, 5000)
      addNotification({
        type: 'moment_commented',
        title: '瞬间被评论',
        description: `${data.user.nickname}: ${data.content}`,
        data: { moment_id: data.moment_id, comment_id: data.comment_id, user: data.user },
      })
    })
    cleanupFunctions.push(cleanupMomentCommented)

    const cleanupCommentReply = webSocketService.on('CommentReply', (data) => {
      feedback.info('💬 评论被回复', `${data.user.nickname}: ${trunc(data.content, 50)}`, 5000)
      addNotification({
        type: 'comment_reply',
        title: '评论被回复',
        description: `${data.user.nickname} 回复了你的评论`,
        data: { moment_id: data.moment_id, comment_id: data.comment_id, reply_id: data.reply_id, user: data.user },
      })
    })
    cleanupFunctions.push(cleanupCommentReply)

    const cleanupSystemAnnouncement = webSocketService.on('SystemAnnouncement', (data) => {
      // 最佳实践：不要每条通知都弹“打断式提示”，只对高优先级（warning/error）使用顶部 banner
      if (data.severity === 'warning') {
        feedback.warning(data.title, data.content, 8000)
      } else if (data.severity === 'error') {
        feedback.error(data.title, data.content, 10000)
      }
      addNotification({
        type: 'system_announcement',
        title: data.title,
        description: data.content,
        data: { id: data.id, severity: data.severity, action_url: data.action_url },
      })
    })
    cleanupFunctions.push(cleanupSystemAnnouncement)

    const cleanupGroupInvite = webSocketService.on('GroupInviteReceived', (data) => {
      addNotification({
        type: 'group_invite',
        title: '群组邀请',
        description: `${data.inviter.nickname} 邀请你加入「${data.group_name}」`,
        data: { invite_id: data.id, group_chat_id: data.group_chat_id, group_name: data.group_name, inviter: data.inviter },
      })
    })
    cleanupFunctions.push(cleanupGroupInvite)

    const cleanupGroupInviteResponse = webSocketService.on('GroupInviteResponse', (data) => {
      const isAccepted = data.status === 'accepted'
      if (isAccepted) {
        feedback.success('✅ 邀请已接受', `对方已接受你的群组邀请`, 5000)
      } else {
        feedback.warning('❌ 邀请被拒绝', `对方拒绝了你的群组邀请`, 5000)
      }
      addNotification({
        type: 'group_invite_response',
        title: isAccepted ? '群组邀请已接受' : '群组邀请被拒绝',
        description: isAccepted ? '对方已接受你的群组邀请' : '对方拒绝了你的群组邀请',
        data: { invite_id: data.id, group_chat_id: data.group_chat_id, user: data.user, status: data.status },
      })
    })
    cleanupFunctions.push(cleanupGroupInviteResponse)

    const cleanupGroupApplication = webSocketService.on('GroupApplicationReceived', (data) => {
      feedback.warning('📝 入群申请', `${data.applicant.nickname} 申请加入群组`, 5000)
      addNotification({
        type: 'group_application',
        title: '入群申请',
        description: `${data.applicant.nickname} 申请加入群组${data.message ? `: ${data.message}` : ''}`,
        data: { application_id: data.id, group_chat_id: data.group_chat_id, applicant: data.applicant },
      })
    })
    cleanupFunctions.push(cleanupGroupApplication)

    const cleanupGroupApplicationResponse = webSocketService.on('GroupApplicationResponse', (data) => {
      const isAccepted = data.status === 'accepted'
      if (isAccepted) {
        feedback.success('✅ 申请已通过', `你已成功加入「${data.group_name}」`, 5000)
      } else {
        feedback.error('❌ 申请被拒绝', `你的入群申请被拒绝`, 5000)
      }
      addNotification({
        type: 'group_application_response',
        title: isAccepted ? '入群申请通过' : '入群申请被拒绝',
        description: isAccepted ? `你已成功加入「${data.group_name}」` : `你的「${data.group_name}」入群申请被拒绝`,
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

