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

// 内置头像配置
const BUILT_IN_AVATARS: Avatar[] = [
  { id: 'avatar-1', name: '蓝色用户', url: '/avatars/avatar-1.svg', color: '#3B82F6' },
  { id: 'avatar-2', name: '绿色用户', url: '/avatars/avatar-2.svg', color: '#10B981' },
  { id: 'avatar-3', name: '紫色用户', url: '/avatars/avatar-3.svg', color: '#8B5CF6' },
  { id: 'avatar-4', name: '红色用户', url: '/avatars/avatar-4.svg', color: '#EF4444' },
  { id: 'avatar-5', name: '黄色用户', url: '/avatars/avatar-5.svg', color: '#F59E0B' },
  { id: 'avatar-6', name: '粉色用户', url: '/avatars/avatar-6.svg', color: '#EC4899' },
  { id: 'avatar-7', name: '青色用户', url: '/avatars/avatar-7.svg', color: '#06B6D4' },
  { id: 'avatar-8', name: '橙色用户', url: '/avatars/avatar-8.svg', color: '#F97316' },
]

// 使用Heroicons作为默认头像（如果SVG不可用）
const HEROICON_AVATARS: Avatar[] = [
  { id: 'hero-1', name: '默认用户', url: '', color: '#6B7280' },
  { id: 'hero-2', name: '蓝色用户', url: '', color: '#3B82F6' },
  { id: 'hero-3', name: '绿色用户', url: '', color: '#10B981' },
  { id: 'hero-4', name: '紫色用户', url: '', color: '#8B5CF6' },
  { id: 'hero-5', name: '红色用户', url: '', color: '#EF4444' },
  { id: 'hero-6', name: '黄色用户', url: '', color: '#F59E0B' },
  { id: 'hero-7', name: '粉色用户', url: '', color: '#EC4899' },
  { id: 'hero-8', name: '青色用户', url: '', color: '#06B6D4' },
  { id: 'hero-9', name: '橙色用户', url: '', color: '#F97316' },
  { id: 'hero-10', name: '靛蓝用户', url: '', color: '#6366F1' },
  { id: 'hero-11', name: '翠绿用户', url: '', color: '#059669' },
  { id: 'hero-12', name: '琥珀用户', url: '', color: '#D97706' }
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