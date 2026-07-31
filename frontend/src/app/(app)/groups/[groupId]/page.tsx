import { GroupDetailPage } from '@/app/pages/groups/GroupDetailPage'

export function generateStaticParams() {
  return [{ groupId: '_' }]
}

export default function GroupDetailRoute() {
  return <GroupDetailPage />
}

export const dynamic = 'force-static'
