/**
 * 内置头像管理
 * 提供预设头像选择功能
 */

export interface Avatar {
  id: string
  name: string
  url: string
  color: string
}

// 内置头像配置 (使用 Nuxt UI v3 语义化颜色)
const BUILT_IN_AVATARS: Avatar[] = [
  { id: 'avatar-1', name: '主色用户', url: '/avatars/avatar-1.svg', color: 'primary' },
  { id: 'avatar-2', name: '成功色用户', url: '/avatars/avatar-2.svg', color: 'success' },
  { id: 'avatar-3', name: '紫色用户', url: '/avatars/avatar-3.svg', color: 'purple' },
  { id: 'avatar-4', name: '错误色用户', url: '/avatars/avatar-4.svg', color: 'error' },
  { id: 'avatar-5', name: '警告色用户', url: '/avatars/avatar-5.svg', color: 'warning' },
  { id: 'avatar-6', name: '粉色用户', url: '/avatars/avatar-6.svg', color: 'pink' },
  { id: 'avatar-7', name: '信息色用户', url: '/avatars/avatar-7.svg', color: 'info' },
  { id: 'avatar-8', name: '橙色用户', url: '/avatars/avatar-8.svg', color: 'orange' },
]

// 使用Heroicons作为默认头像（如果SVG不可用）- 使用 Nuxt UI v3 语义化颜色
const HEROICON_AVATARS: Avatar[] = [
  { id: 'hero-1', name: '默认用户', url: '', color: 'neutral' },
  { id: 'hero-2', name: '主色用户', url: '', color: 'primary' },
  { id: 'hero-3', name: '成功色用户', url: '', color: 'success' },
  { id: 'hero-4', name: '紫色用户', url: '', color: 'purple' },
  { id: 'hero-5', name: '错误色用户', url: '', color: 'error' },
  { id: 'hero-6', name: '警告色用户', url: '', color: 'warning' },
  { id: 'hero-7', name: '粉色用户', url: '', color: 'pink' },
  { id: 'hero-8', name: '信息色用户', url: '', color: 'info' },
  { id: 'hero-9', name: '橙色用户', url: '', color: 'orange' },
  { id: 'hero-10', name: '靛蓝用户', url: '', color: 'indigo' },
  { id: 'hero-11', name: '翠绿用户', url: '', color: 'emerald' },
  { id: 'hero-12', name: '琥珀用户', url: '', color: 'amber' }
]

export const useAvatars = () => {
  // 获取所有可用头像
  const getAvailableAvatars = (): Avatar[] => {
    // 优先使用内置SVG头像，如果不可用则使用Heroicons
    return [...BUILT_IN_AVATARS, ...HEROICON_AVATARS]
  }

  // 根据用户名生成默认头像
  const getDefaultAvatar = (nickname: string): Avatar => {
    const index = nickname.charCodeAt(0) % HEROICON_AVATARS.length
    return HEROICON_AVATARS[index] || HEROICON_AVATARS[0]!
  }

  // 生成头像URL（如果是Heroicon则使用用户名首字母）
  const getAvatarDisplay = (avatar: Avatar, nickname?: string) => {
    if (avatar.url) {
      return {
        type: 'image' as const,
        src: avatar.url,
        alt: avatar.name
      }
    } else {
      return {
        type: 'text' as const,
        text: nickname ? nickname.charAt(0).toUpperCase() : 'U',
        style: { backgroundColor: avatar.color, color: 'white' }
      }
    }
  }

  // 随机选择一个头像
  const getRandomAvatar = (): Avatar => {
    const avatars = getAvailableAvatars()
    const randomIndex = Math.floor(Math.random() * avatars.length)
    return avatars[randomIndex] || avatars[0]!
  }

  return {
    getAvailableAvatars,
    getDefaultAvatar,
    getAvatarDisplay,
    getRandomAvatar
  }
}