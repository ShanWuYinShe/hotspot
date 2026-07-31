import { ProfilePage } from '../ProfileFeature'

// 静态导出需要枚举全部 section 取值（ProfileSection：moments/groups/relationships）
export function generateStaticParams() {
  return [{ section: 'moments' }, { section: 'groups' }, { section: 'relationships' }]
}

export default function ProfileSectionRoute() {
  return <ProfilePage />
}

export const dynamic = 'force-static'
