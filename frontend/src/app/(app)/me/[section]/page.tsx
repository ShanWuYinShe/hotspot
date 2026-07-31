import { ProfilePage } from '../ProfileFeature'

export function generateStaticParams() {
  return [{ section: 'moments' }]
}

export default function ProfileSectionRoute() {
  return <ProfilePage />
}

export const dynamic = 'force-static'
