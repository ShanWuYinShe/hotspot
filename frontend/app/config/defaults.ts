/**
 * 与后端默认设置保持一致的前端默认常量
 */

export const DEFAULTS = {
  // 后端 DEFAULT_LATITUDE / DEFAULT_LONGITUDE
  latitude: 35.154921,
  longitude: 112.68253,
  // 后端 DEFAULT_LOCATION_NAME
  locationName: '南天门',
  // 后端 DEFAULT_NOTIFICATION_RADIUS_KM 约定的常用半径（5km）
  nearbyRadiusMeters: 5000,
  // 通用分页默认值
  pageSize: 20,
  maxPageSize: 100,
  // 本地保存账号最大数量
  maxSavedAccounts: 200,
}

/**
 * 时间相关常量（毫秒）
 */
export const TIME_CONSTANTS = {
  SECOND: 1000,
  MINUTE: 60 * 1000,
  HOUR: 60 * 60 * 1000,
  DAY: 24 * 60 * 60 * 1000,
  WEEK: 7 * 24 * 60 * 60 * 1000,
  MONTH: 30 * 24 * 60 * 60 * 1000,
  YEAR: 365 * 24 * 60 * 60 * 1000,
}

/**
 * 群组相关常量
 */
export const GROUP_CONSTANTS = {
  MAX_MEMBERS_DEFAULT: 500,
  MAX_MEMBERS_LIMIT: 1000,
  MIN_MEMBERS: 2,
  INVITE_CODE_LENGTH: 8,
  REFRESH_INTERVAL: 30000, // 30秒
}

/**
 * 生活瞬间相关常量
 */
export const MOMENT_CONSTANTS = {
  MAX_DESCRIPTION_LENGTH: 500,
  MAX_IMAGES: 9,
  MAX_TAGS: 10,
  DRAFT_STORAGE_KEY: 'moment_draft',
}

/**
 * 用户相关常量
 */
export const USER_CONSTANTS = {
  MAX_NICKNAME_LENGTH: 50,
  MIN_PASSWORD_LENGTH: 6,
  MAX_BIO_LENGTH: 200,
}

/**
 * UI相关常量
 */
export const UI_CONSTANTS = {
  SKELETON_COUNT: 3,
  TOAST_TIMEOUT: 3000,
  DEBOUNCE_DELAY: 300,
  ANIMATION_DURATION: 200,
}

/**
 * 聊天相关常量
 */
export const CHAT_CONSTANTS = {
  COMMON_EMOJIS: [
    "😀", "😃", "😄", "😁", "😆", "😅", "😂", "🤣", "😊", "😇",
    "🙂", "🙃", "😉", "😌", "😍", "🥰", "😘", "😗", "😙", "😚",
    "😋", "😛", "😝", "😜", "🤪", "🤨", "🧐", "🤓", "😎", "🤩",
    "🥳", "😏", "😒", "😞", "😔", "😟", "😕", "🙁", "☹️", "😣",
    "😖", "😫", "😩", "🥺", "😢", "😭", "😤", "😠", "😡", "🤬",
    "🤯", "😳", "🥵", "🥶", "😱", "😨", "😰", "😥", "😓", "🤗",
    "🤔", "🤭", "🤫", "🤥",
  ] as const,
  ACCEPTED_ATTACHMENTS: "image/*,.pdf,.doc,.docx,.txt" as const,
}

/**
 * WebSocket 心跳配置（与后端 constants.rs 保持一致）
 */
export const WS_CONSTANTS = {
  /** 客户端心跳间隔（毫秒） */
  HEARTBEAT_INTERVAL: 45 * 1000,
  /** 心跳响应超时时间（毫秒） */
  HEARTBEAT_TIMEOUT: 10 * 1000,
  /** 最大允许丢失心跳次数 */
  MAX_MISSED_HEARTBEATS: 3,
  /** 重连间隔（毫秒） */
  RECONNECT_INTERVAL: 3000,
  /** 最大重连次数 */
  MAX_RECONNECT_ATTEMPTS: 5,
}

/**
 * 服务端连接超时时间（毫秒）
 * 计算公式: HEARTBEAT_INTERVAL * 2 + HEARTBEAT_TIMEOUT
 */
export const WS_CONNECTION_TIMEOUT = WS_CONSTANTS.HEARTBEAT_INTERVAL * 2 + WS_CONSTANTS.HEARTBEAT_TIMEOUT



