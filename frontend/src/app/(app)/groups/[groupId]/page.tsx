import { GroupDetailPage } from './GroupDetailPage'

export function generateStaticParams() {
  return [{ groupId: '_' }]
}

export default function GroupDetailRoute() {
  return <GroupDetailPage />
}

export const dynamic = 'force-static'
