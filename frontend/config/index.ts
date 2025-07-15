/**
 * 应用配置统一导出
 * 简化配置结构，提供所有配置的统一访问入口
 */

// 环境类型定义
export type Environment = 'development' | 'production' | 'staging' | 'test'

// 当前环境
export const NODE_ENV = (process.env.NODE_ENV || 'development') as Environment

// 地理位置默认值
export const DEFAULT_LATITUDE = 35.16
export const DEFAULT_LONGITUDE = 112.68
export const DEFAULT_LOCATION_NAME = '南天门'

// API相关常量 - 使用 useRuntimeConfig composable
export const getApiConfig = () => {
  const config = useRuntimeConfig()
  return {
    BASE_URL: config.public.apiBaseUrl,
    WS_BASE_URL: config.public.wsBaseUrl,
    TIMEOUT: 30000,
    RETRY_COUNT: 3,
  } as const
}

// 应用配置常量
export const API_CONFIG = {
  TIMEOUT: 30000,
  RETRY_COUNT: 3,
} as const

// 应用配置常量
export const APP_CONFIG = {
  NAME: 'HotSpot',
  VERSION: '1.0.0',
  DESCRIPTION: 'Location-based social application',
  
  // 分页配置
  PAGINATION: {
    DEFAULT_PAGE_SIZE: 20,
    MAX_PAGE_SIZE: 100,
  },
  
  // 地图配置
  MAP: {
    DEFAULT_ZOOM: 15,
    MIN_ZOOM: 3,
    MAX_ZOOM: 18,
    DEFAULT_CENTER: [DEFAULT_LONGITUDE, DEFAULT_LATITUDE] as const,
    DEFAULT_LOCATION: {
      latitude: DEFAULT_LATITUDE,
      longitude: DEFAULT_LONGITUDE,
      locationName: DEFAULT_LOCATION_NAME
    },
  },
  
  // 文件上传配置
  UPLOAD: {
    MAX_SIZE: 10 * 1024 * 1024, // 10MB
    ALLOWED_TYPES: ['image/jpeg', 'image/png', 'image/gif'],
  },
} as const

// 业务规则常量
export const BUSINESS_RULES = {
  // 用户相关
  USER: {
    MIN_NICKNAME_LENGTH: 2,
    MAX_NICKNAME_LENGTH: 20,
    MIN_PASSWORD_LENGTH: 8,
    MAX_PASSWORD_LENGTH: 64,
  },
  
  // 群组相关
  GROUP: {
    MIN_NAME_LENGTH: 2,
    MAX_NAME_LENGTH: 30,
    MAX_DESCRIPTION_LENGTH: 200,
    MAX_MEMBERS: 500,
  },
  
  // 消息相关
  MESSAGE: {
    MAX_CONTENT_LENGTH: 1000,
    MAX_HISTORY_DAYS: 30,
  },
  
  // 时刻动态相关
  MOMENT: {
    MAX_DESCRIPTION_LENGTH: 200,
    MAX_TAGS: 5,
    LOCATION_RADIUS: 100, // 米
  },
} as const

// UI相关常量
export const UI_CONFIG = {
  // 断点配置
  BREAKPOINTS: {
    XS: 480,
    SM: 640,
    MD: 768,
    LG: 1024,
    XL: 1280,
    '2XL': 1536,
  },
  
  // 动画配置
  ANIMATION: {
    DURATION: {
      FAST: 150,
      NORMAL: 300,
      SLOW: 500,
    },
    EASING: 'cubic-bezier(0.4, 0, 0.2, 1)',
  },
  
  // Z-index层级
  Z_INDEX: {
    DROPDOWN: 1000,
    STICKY: 1010,
    FIXED: 1020,
    MODAL_BACKDROP: 1030,
    MODAL: 1040,
    POPOVER: 1050,
    TOOLTIP: 1060,
    NOTIFICATION: 1070,
  },
  
  // 导航配置
  NAVIGATION: {
    MAIN_ITEMS: [
      {
        path: '/',
        label: '首页',
        icon: 'HomeFilled',
        showBadge: false
      },
      {
        path: '/my-groups',
        label: '群组',
        icon: 'UserFilled',
        showBadge: false
      },
      {
        path: '/dynamic',
        label: '动态',
        icon: 'LocationFilled',
        showBadge: false
      },
      {
        path: '/messages',
        label: '消息',
        icon: 'ChatDotRound',
        showBadge: true
      }
    ]
  },
} as const

// 错误码定义
export const ERROR_CODES = {
  // 网络错误
  NETWORK_ERROR: 'NETWORK_ERROR',
  TIMEOUT: 'TIMEOUT',
  
  // 认证错误
  UNAUTHORIZED: 'UNAUTHORIZED',
  FORBIDDEN: 'FORBIDDEN',
  TOKEN_EXPIRED: 'TOKEN_EXPIRED',
  
  // 业务错误
  VALIDATION_ERROR: 'VALIDATION_ERROR',
  RESOURCE_NOT_FOUND: 'RESOURCE_NOT_FOUND',
  RESOURCE_CONFLICT: 'RESOURCE_CONFLICT',
  
  // 系统错误
  INTERNAL_ERROR: 'INTERNAL_ERROR',
  SERVICE_UNAVAILABLE: 'SERVICE_UNAVAILABLE',
} as const

// 状态常量
export const STATUS = {
  IDLE: 'idle',
  LOADING: 'loading',
  SUCCESS: 'success',
  ERROR: 'error',
} as const

// 事件名称常量
export const EVENTS = {
  // 认证事件
  USER_LOGIN: 'user:login',
  USER_LOGOUT: 'user:logout',
  TOKEN_REFRESH: 'token:refresh',
  
  // 消息事件
  MESSAGE_RECEIVED: 'message:received',
  MESSAGE_SENT: 'message:sent',
  CONVERSATION_UPDATED: 'conversation:updated',
  
  // 群组事件
  GROUP_JOINED: 'group:joined',
  GROUP_LEFT: 'group:left',
  GROUP_UPDATED: 'group:updated',
  
  // 位置事件
  LOCATION_UPDATED: 'location:updated',
  NEARBY_USERS_UPDATED: 'nearby:users:updated',
  
  // 时刻动态事件
  MOMENT_CREATED: 'moment:created',
  MOMENT_LIKED: 'moment:liked',
} as const

// 本地存储键名
export const STORAGE_KEYS = {
  AUTH_TOKEN: 'auth:token',
  REFRESH_TOKEN: 'auth:refresh_token',
  USER_DATA: 'user:data',
  USER_PREFERENCES: 'user:preferences',
  LOCATION_PERMISSION: 'location:permission',
  THEME: 'ui:theme',
  LANGUAGE: 'ui:language',
} as const

// 环境配置
interface EnvConfig {
  name: string
  api: {
    baseUrl: string
    timeout: number
  }
  features: {
    enableDevTools: boolean
    enableDebugger: boolean
    enableConsoleLog: boolean
    enableAnalytics: boolean
  }
  performance: {
    enableServiceWorker: boolean
    enablePreloading: boolean
    enableLazyLoading: boolean
  }
}

// 开发环境配置
const developmentConfig: EnvConfig = {
  name: '开发环境',
  api: {
    baseUrl: '',  // 将在运行时通过 getApiConfig() 获取
    timeout: API_CONFIG.TIMEOUT,
  },
  features: {
    enableDevTools: true,
    enableDebugger: true,
    enableConsoleLog: true,
    enableAnalytics: false,
  },
  performance: {
    enableServiceWorker: false,
    enablePreloading: false,
    enableLazyLoading: true,
  },
}

// 生产环境配置
const productionConfig: EnvConfig = {
  name: '生产环境',
  api: {
    baseUrl: '',  // 将在运行时通过 getApiConfig() 获取
    timeout: 15000,
  },
  features: {
    enableDevTools: false,
    enableDebugger: false,
    enableConsoleLog: false,
    enableAnalytics: true,
  },
  performance: {
    enableServiceWorker: true,
    enablePreloading: true,
    enableLazyLoading: true,
  },
}

// 根据环境选择配置
export const envConfig = NODE_ENV === 'production' ? productionConfig : developmentConfig

// 功能标志
export const featureFlags = {
  enableChat: true,
  enableGroupChat: true,
  enableFileUpload: true,
  enableNotifications: true,
  enableGeolocation: true,
  enableOfflineMode: false,
  enablePWA: NODE_ENV === 'production',
}

// 日志工具函数
export const debugLog = (...args: any[]) => {
  if (envConfig.features.enableDebugger) {
    console.debug('[DEBUG]', ...args)
  }
}

export const errorLog = (...args: any[]) => {
  if (envConfig.features.enableConsoleLog) {
    console.error('[ERROR]', ...args)
  }
}

export const warnLog = (...args: any[]) => {
  if (envConfig.features.enableConsoleLog) {
    console.warn('[WARN]', ...args)
  }
}

// 统一的应用配置对象
export const appConfig = {
  app: APP_CONFIG,
  api: API_CONFIG,
  business: BUSINESS_RULES,
  ui: UI_CONFIG,
  env: envConfig,
  features: featureFlags,
  errorCodes: ERROR_CODES,
  status: STATUS,
  events: EVENTS,
  storageKeys: STORAGE_KEYS,
  utils: {
    debugLog,
    errorLog,
    warnLog,
  },
}

// 默认导出
export default appConfig
