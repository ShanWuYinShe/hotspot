import { MomentDetailPage } from './MomentDetailPage'

export function generateStaticParams() {
  return [{ postId: '_' }]
}

export default function MomentDetailRoute() {
  return <MomentDetailPage />
}

export const dynamic = 'force-static'
