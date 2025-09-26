/**
 * SEO 数据源配置
 * 统一管理所有页面的 SEO 信息
 */

export interface SeoConfig {
  title: string
  description: string
  keywords?: string[]
  image?: string
  type?: 'website' | 'article' | 'profile'
}

// 基础 SEO 配置
export const baseSeoConfig = {
  siteName: 'HotSpot',
  siteDescription: '基于位置的社交平台',
  defaultImage: '/images/og-default.jpg',
  baseKeywords: ['HotSpot', '社交', '位置', '群组', '朋友', '附近'],
  themeColor: '#10b981',
  twitterSite: '@hotspot'
}

// 页面 SEO 配置
export const pageSeoConfig: Record<string, SeoConfig> = {
  // 首页
  home: {
    title: '首页',
    description: '发现附近的朋友，加入有趣的群组，分享精彩瞬间',
    keywords: ['首页', '发现', '社交', '附近'],
    type: 'website'
  },

  // 地图页面
  map: {
    title: '地图探索',
    description: '在地图上发现附近的用户、群组和精彩动态',
    keywords: ['地图', '探索', '附近', '位置'],
    type: 'website'
  },

  // 群组相关
  groups: {
    title: '我的群组',
    description: '管理和浏览您加入的群组，发现更多有趣的社区',
    keywords: ['群组', '社区', '管理'],
    type: 'website'
  },

  'groups-owned': {
    title: '管理群组',
    description: '管理您创建的群组，查看成员和活动统计',
    keywords: ['群组管理', '创建', '统计'],
    type: 'website'
  },

  'groups-create': {
    title: '创建群组',
    description: '创建一个新的群组，邀请朋友一起交流',
    keywords: ['创建群组', '邀请', '交流'],
    type: 'website'
  },

  // 消息聊天
  chat: {
    title: '消息中心',
    description: '与朋友和群组成员实时聊天交流',
    keywords: ['消息', '聊天', '实时'],
    type: 'website'
  },

  // 用户相关
  users: {
    title: '用户',
    description: '发现和连接有趣的用户',
    keywords: ['用户', '发现', '连接'],
    type: 'website'
  },

  'users-nearby': {
    title: '附近用户',
    description: '发现您附近的用户，扩展社交圈',
    keywords: ['附近用户', '发现', '社交圈'],
    type: 'website'
  },

  'users-search': {
    title: '搜索用户',
    description: '通过各种条件搜索和发现用户',
    keywords: ['搜索用户', '发现', '筛选'],
    type: 'website'
  },

  // 动态相关
  moments: {
    title: '动态分享',
    description: '浏览和分享生活中的精彩瞬间',
    keywords: ['动态', '分享', '瞬间'],
    type: 'website'
  },

  // 个人资料
  profile: {
    title: '个人资料',
    description: '查看和编辑您的个人资料信息',
    keywords: ['个人资料', '编辑', '信息'],
    type: 'profile'
  },

  // 设置页面
  settings: {
    title: '设置中心',
    description: '个性化定制您的体验，管理账户信息和偏好设置',
    keywords: ['设置', '个性化', '账户', '偏好'],
    type: 'website'
  },

  'settings-profile': {
    title: '个人资料设置',
    description: '管理您的个人信息和账户设置',
    keywords: ['个人资料', '设置', '管理'],
    type: 'website'
  },

  // 认证页面
  'auth-login': {
    title: '用户登录',
    description: '登录您的 HotSpot 账户，开始社交之旅',
    keywords: ['登录', '账户', '社交'],
    type: 'website'
  },

  'auth-register': {
    title: '用户注册',
    description: '创建您的 HotSpot 账户，加入我们的社交平台',
    keywords: ['注册', '创建账户', '加入'],
    type: 'website'
  },

  // 通知中心
  notifications: {
    title: '通知中心',
    description: '查看和管理您的所有通知消息',
    keywords: ['通知', '消息', '管理'],
    type: 'website'
  },

  // 仪表板
  dashboard: {
    title: '数据仪表板',
    description: '查看项目概览和关键指标统计',
    keywords: ['仪表板', '统计', '数据', '概览'],
    type: 'website'
  }
}

// 获取页面 SEO 配置的工具函数
export function getPageSeoConfig(pageKey: string): SeoConfig {
  return pageSeoConfig[pageKey] || {
    title: baseSeoConfig.siteName,
    description: baseSeoConfig.siteDescription,
    type: 'website'
  }
}

// 构建完整的 SEO meta 数据
export function buildSeoMeta(pageKey: string, customConfig?: Partial<SeoConfig>) {
  const config = getPageSeoConfig(pageKey)
  const finalConfig = { ...config, ...customConfig }
  
  const fullTitle = finalConfig.title === baseSeoConfig.siteName 
    ? baseSeoConfig.siteName 
    : `${finalConfig.title} - ${baseSeoConfig.siteName}`
  
  const keywords = [
    ...baseSeoConfig.baseKeywords,
    ...(finalConfig.keywords || [])
  ].join(', ')

  return {
    title: fullTitle,
    description: finalConfig.description,
    keywords,
    
    // Open Graph
    ogTitle: fullTitle,
    ogDescription: finalConfig.description,
    ogImage: finalConfig.image || baseSeoConfig.defaultImage,
    ogType: finalConfig.type || 'website',
    ogSiteName: baseSeoConfig.siteName,
    
    // Twitter Card
    twitterCard: 'summary_large_image' as const,
    twitterTitle: fullTitle,
    twitterDescription: finalConfig.description,
    twitterImage: finalConfig.image || baseSeoConfig.defaultImage,
    twitterSite: baseSeoConfig.twitterSite,
    
    // 基础 meta
    viewport: 'width=device-width, initial-scale=1',
    themeColor: baseSeoConfig.themeColor,
    robots: 'index, follow',
    
    // PWA 相关
    applicationName: baseSeoConfig.siteName,
    appleWebAppTitle: baseSeoConfig.siteName,
    appleWebAppCapable: 'yes',
    appleWebAppStatusBarStyle: 'default'
  }
}
