import { ChatDetailPage } from '@/app/pages/messaging/ChatDetailPage'

export function generateStaticParams() {
  return [{ chatId: '_' }]
}

export default function ChatDetailRoute() {
  return <ChatDetailPage />
}

export const dynamic = 'force-static'
