/**
 * 群组聊天页面类型定义
 */

export interface Message {
  id: string
  content: string
  content_type: 'text' | 'image' | 'location' | 'file' | 'voice' | 'system'
  sender_id: string
  group_id: string
  created_at: string
  updated_at: string
  sender?: User | null
}

export interface User {
  id: string
  nickname: string
  avatar_url?: string
  email?: string
  bio?: string
}

export interface LocationContent {
  name: string
  address: string
  latitude: number
  longitude: number
}

export interface MessageGroup {
  id: string
  type: 'date' | 'message-group'
  date?: string
  senderId?: string
  sender?: User | null
  messages?: Message[]
  time?: string
}

export interface TypingUser {
  id: string
  name: string
  timestamp: number
  avatar?: string
}

export interface ChatPageData {
  groupId: string
  messages: Message[]
  typingUsers: TypingUser[]
  hasMoreHistory: boolean
  loading: boolean
  loadingHistory: boolean
  sending: boolean
}

export interface ShareOptions {
  title: string
  desc: string
  link: string
  imgUrl: string
  messages: Message[]
}
