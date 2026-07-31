import { UserProfilePage } from './UserProfilePage'

export function generateStaticParams() {
  return [{ userId: '_' }]
}

export default function UserProfileRoute() {
  return <UserProfilePage />
}

export const dynamic = 'force-static'
