/**
 * WebSocket 事件落地插件（只在客户端运行）
 *
 * 目标：
 * - 将 WS 事件统一落到 Pinia store（会话列表/消息列表/在线状态/正在输入）
 * - 页面只读 store，避免每个页面重复写 wsOn 与状态同步逻辑
 */
import { webSocketService } from '~/services/websocket.service'
import type {
  PrivateMessageNewEvent,
  PrivateMessageEditedEvent,
  PrivateMessageRecalledEvent,
  PrivateMessageDeletedEvent,
  PrivateMessageReadEvent,
  PrivateMessageReactionEvent,
  GroupMessageNewEvent,
  GroupMessageEditedEvent,
  GroupMessageRecalledEvent,
  GroupMessageDeletedEvent,
  GroupMessageReadEvent,
  GroupMessageReactionEvent,
  GroupUserTypingEvent,
  PrivateUserTypingEvent,
  UserOfflineEvent,
  UserOnlineEvent
} from '~/types/websocket'

export default defineNuxtPlugin((nuxtApp) => {
  if (import.meta.server) return

  const authStore = useAuthStore()
  const messagesStore = useMessagesStore()

  // 订阅目标用户在线状态：基于会话列表里出现的私聊对象
  let lastSubscribedUserIds = new Set<number>()

  function syncUserPresenceSubscriptions() {
    if (webSocketService.getStatus() !== 'connected') return
    if (!authStore.isAuthenticated) return

    const next = new Set<number>()
    for (const c of messagesStore.conversationsList) {
      if (c.chat_type === 'private' && c.target_user_id) {
        next.add(c.target_user_id)
      }
    }

    const toSubscribe: number[] = []
    const toUnsubscribe: number[] = []
    for (const id of next) if (!lastSubscribedUserIds.has(id)) toSubscribe.push(id)
    for (const id of lastSubscribedUserIds) if (!next.has(id)) toUnsubscribe.push(id)

    if (toSubscribe.length) {
      webSocketService.subscribe([{ source_type: 'user', source_ids: toSubscribe }])
    }
    if (toUnsubscribe.length) {
      webSocketService.unsubscribe([{ source_type: 'user', source_ids: toUnsubscribe }])
    }

    lastSubscribedUserIds = next
  }

  function currentUserId() {
    return authStore.user?.user_id
  }

  nuxtApp.hook('app:mounted', () => {
    // 会话列表变化时，动态同步订阅（仅连接态）
    watch(
      () => messagesStore.conversationsList.map(c => `${c.chat_id}:${c.target_user_id ?? ''}`).join('|'),
      () => syncUserPresenceSubscriptions(),
      { immediate: true }
    )

    // 连接状态变化：connected 时补一次订阅
    webSocketService.onStatusChange((s) => {
      if (s === 'connected') syncUserPresenceSubscriptions()
    })

    // ====== 消息类事件：统一落 messagesStore（去重由 store 负责）======
    // 私聊
    webSocketService.on('PrivateMessageNew', (p: PrivateMessageNewEvent['payload']) => {
      messagesStore.handlePrivateMessageNew(p, currentUserId())
    })
    webSocketService.on('PrivateMessageEdited', (p: PrivateMessageEditedEvent['payload']) => {
      messagesStore.handlePrivateMessageEdited(p)
    })
    webSocketService.on('PrivateMessageRecalled', (p: PrivateMessageRecalledEvent['payload']) => {
      messagesStore.handlePrivateMessageRecalled(p)
    })
    webSocketService.on('PrivateMessageDeleted', (p: PrivateMessageDeletedEvent['payload']) => {
      messagesStore.handlePrivateMessageDeleted(p)
    })
    webSocketService.on('PrivateMessageRead', (p: PrivateMessageReadEvent['payload']) => {
      messagesStore.handlePrivateMessageRead(p, currentUserId())
    })
    webSocketService.on('PrivateMessageReaction', (p: PrivateMessageReactionEvent['payload']) => {
      messagesStore.handlePrivateMessageReaction(p)
    })

    // 群聊
    webSocketService.on('GroupMessageNew', (p: GroupMessageNewEvent['payload']) => {
      messagesStore.handleGroupMessageNew(p, currentUserId())
    })
    webSocketService.on('GroupMessageEdited', (p: GroupMessageEditedEvent['payload']) => {
      messagesStore.handleGroupMessageEdited(p)
    })
    webSocketService.on('GroupMessageRecalled', (p: GroupMessageRecalledEvent['payload']) => {
      messagesStore.handleGroupMessageRecalled(p)
    })
    webSocketService.on('GroupMessageDeleted', (p: GroupMessageDeletedEvent['payload']) => {
      messagesStore.handleGroupMessageDeleted(p)
    })
    webSocketService.on('GroupMessageRead', (p: GroupMessageReadEvent['payload']) => {
      messagesStore.handleGroupMessageRead(p, currentUserId())
    })
    webSocketService.on('GroupMessageReaction', (p: GroupMessageReactionEvent['payload']) => {
      messagesStore.handleGroupMessageReaction(p)
    })

    webSocketService.on('PrivateUserTyping', (p: PrivateUserTypingEvent['payload']) => {
      messagesStore.handlePrivateUserTyping(p)
    })
    webSocketService.on('GroupUserTyping', (p: GroupUserTypingEvent['payload']) => {
      messagesStore.handleGroupUserTyping(p, currentUserId())
    })

    // ====== 在线状态事件：落到会话列表（以及私聊详情页会自己处理更细节的 last_online）======
    webSocketService.on('UserOnline', (p: UserOnlineEvent['payload']) => {
      messagesStore.handleUserOnline(p)
    })
    webSocketService.on('UserOffline', (p: UserOfflineEvent['payload']) => {
      messagesStore.handleUserOffline(p)
    })
  })
})

