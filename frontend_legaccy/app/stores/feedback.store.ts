import { defineStore } from 'pinia'

export type FeedbackColor = 'primary' | 'success' | 'info' | 'warning' | 'error' | 'neutral'

export interface FeedbackMessage {
  id: string
  title: string
  description?: string
  color: FeedbackColor
  createdAt: number
  timeoutMs?: number
}

const genId = () => `${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 8)}`

export const useFeedbackStore = defineStore('feedback', () => {
  // 已按产品要求禁用 UI 反馈：只输出到 console，避免遮挡/样式问题。
  // 保留 store API，减少业务代码改动面。
  const messages = ref<FeedbackMessage[]>([])

  const push = (input: Omit<FeedbackMessage, 'id' | 'createdAt'>) => {
    const msg: FeedbackMessage = {
      id: genId(),
      createdAt: Date.now(),
      ...input,
    }
    const prefix = `[Feedback:${msg.color}]`
    const text = msg.description ? `${msg.title} - ${msg.description}` : msg.title
    if (msg.color === 'error') console.error(prefix, text)
    else if (msg.color === 'warning') console.warn(prefix, text)
    else if (msg.color === 'success') console.info(prefix, text)
    else console.log(prefix, text)

    return msg.id
  }

  const dismiss = (id: string) => {
    // UI 已禁用：保留 API 但不做任何事
    void id
  }

  const clear = () => {
    messages.value = []
  }

  const success = (title: string, description?: string, timeoutMs = 2500) =>
    push({ title, description, color: 'success', timeoutMs })
  const info = (title: string, description?: string, timeoutMs = 2500) =>
    push({ title, description, color: 'info', timeoutMs })
  const warning = (title: string, description?: string, timeoutMs = 3500) =>
    push({ title, description, color: 'warning', timeoutMs })
  const error = (title: string, description?: string, timeoutMs = 5000) =>
    push({ title, description, color: 'error', timeoutMs })

  return {
    messages,
    push,
    dismiss,
    clear,
    success,
    info,
    warning,
    error,
  }
})

