/**
 * WebSocket composable：提供组件内的“响应式状态 + 自动清理监听”。
 * 
 * 说明：
 * - 连接/重连/token 刷新等全局生命周期由 `plugins/04.websocket.client.ts` 负责
 * - 这里仅做 Vue 侧适配，避免组件里手写清理逻辑
 */

import { tryOnBeforeUnmount } from '@vueuse/core'
import { webSocketService } from '~/services/websocket.service'
import type { ClientEvent, ServerEventType, ServerPayload, SubscriptionItem, WebSocketState } from '~/types/websocket'

export function useWebSocket() {
  const status = ref<WebSocketState>(webSocketService.getStatus())
  const isConnected = computed(() => status.value === 'connected')
  const isConnecting = computed(() => status.value === 'connecting')

  const cleanupFns = new Set<() => void>()

  cleanupFns.add(
    webSocketService.onStatusChange((newStatus) => {
      status.value = newStatus
    })
  )

  tryOnBeforeUnmount(() => {
    cleanupFns.forEach(fn => fn())
    cleanupFns.clear()
  })

  const on = <K extends ServerEventType>(type: K, handler: (payload: ServerPayload<K>) => void) => {
    const cleanup = webSocketService.on(type, handler)
    cleanupFns.add(cleanup)
    return cleanup
  }

  const send = <T extends ClientEvent>(type: T['type'], payload: T['payload']) => {
    return webSocketService.send(type, payload)
  }

  const subscribe = (items: SubscriptionItem[]) => webSocketService.subscribe(items)
  const unsubscribe = (items: SubscriptionItem[]) => webSocketService.unsubscribe(items)

  const useSubscription = (items: SubscriptionItem[]) => {
    subscribe(items)
    cleanupFns.add(() => unsubscribe(items))
  }

  const subscribeToGroupChats = (ids: string[]) => subscribe([{ source_type: 'group_chat', source_ids: ids }])
  const unsubscribeFromGroupChats = (ids: string[]) => unsubscribe([{ source_type: 'group_chat', source_ids: ids }])

  const subscribeToUsers = (ids: string[]) => subscribe([{ source_type: 'user', source_ids: ids }])
  const unsubscribeFromUsers = (ids: string[]) => unsubscribe([{ source_type: 'user', source_ids: ids }])

  const subscribeToNotifications = (types: string[]) =>
    subscribe([{ source_type: 'notification', source_ids: types }])
  const unsubscribeFromNotifications = (types: string[]) =>
    unsubscribe([{ source_type: 'notification', source_ids: types }])

  return {
    status: readonly(status),
    isConnected,
    isConnecting,

    on,
    send,

    subscribe,
    unsubscribe,
    useSubscription,

    subscribeToGroupChats,
    unsubscribeFromGroupChats,
    subscribeToUsers,
    unsubscribeFromUsers,
    subscribeToNotifications,
    unsubscribeFromNotifications
  }
}
