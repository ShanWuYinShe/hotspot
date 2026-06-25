/**
 * 极简应用常量
 * 只保留核心必要的常量定义
 */

import { DEBUG } from 'BuildProfile';

// API 配置：根据构建模式自动切换
// debug  → 开发服务器（docker compose 的 nginx 网关，局域网 IP，端口 = .env NGINX_PORT 默认 8888）
// release → 正式域名
const DEV_SERVER = 'http://192.168.0.101:8888';
const PROD_SERVER = 'https://api.chaoset.com';

export const SERVER_BASE_URL: string = DEBUG ? DEV_SERVER : PROD_SERVER;

// 由同一个 BaseUrl 推导出 API / WebSocket 地址（避免多处硬编码）
export const API_BASE_URL: string = SERVER_BASE_URL;
export const WS_BASE_URL: string = SERVER_BASE_URL.replace(/^http:/, 'ws:').replace(/^https:/, 'wss:') + '/ws/v1';

// 业务状态码
export interface BusinessCodeType {
  SUCCESS: number;
  ERROR: number;
  UNAUTHORIZED: number;
  SERVER_ERROR: number;
}

export const BusinessCode: BusinessCodeType = {
  SUCCESS: 0,
  ERROR: 40000,
  UNAUTHORIZED: 10100,
  SERVER_ERROR: 50000
};

// 存储键名
export interface StorageKeysType {
  USER_INFO: string;
  AUTH_DATA: string;
  ACCESS_TOKEN: string;
  LOCATION_CACHE: string;
  SERVER_BASE_URL: string;
  AUTH_ACCOUNTS: string;
  THEME_MODE: string;
}

export const StorageKeys: StorageKeysType = {
  USER_INFO: 'user_info',
  AUTH_DATA: 'auth_data',
  ACCESS_TOKEN: 'access_token',
  LOCATION_CACHE: 'location_cache',
  SERVER_BASE_URL: 'server_base_url',
  // 多账号：保存所有账户的 AppAuthData（含 refresh_token），并记录当前活跃账号
  AUTH_ACCOUNTS: 'auth_accounts',
  THEME_MODE: 'theme_mode'
};

// 页面名称
export interface PageNamesType {
  INDEX: string;
  LOGIN: string;
  GROUP: string;
  MESSAGE: string;
  MOMENT: string;
  MOMENT_DETAIL: string;
  PROFILE: string;
  GROUP_DETAIL: string;
  CREATE_GROUP: string;
  CREATE_MOMENT: string;
  CHAT: string;
  LOCATION_PICKER: string;
  SEARCH: string;
  SETTINGS: string;
  ADVANCED_SETTINGS: string;
  PRIVACY_SETTINGS: string;
  SAFETY_SETTINGS: string;
  RELATIONSHIP_HUB: string;
  ANNOUNCEMENTS: string;
  PHOTO_PREVIEW: string;
  DEBUG_PANEL: string;
  SUBSCRIPTIONS: string;
  ACCOUNTS: string;
  MY_SHARES: string;
  SHARE_RESOLUTION: string;
  EVENT_DETAIL: string;
  PROFILE_EDIT: string;
  NEARBY_USERS: string;
  ADMIN_OVERVIEW: string;
}

export const PageNames: PageNamesType = {
  INDEX: 'Index',
  LOGIN: 'LoginPage',
  GROUP: 'GroupPage',
  MESSAGE: 'MessagePage',
  MOMENT: 'MomentPage',
  MOMENT_DETAIL: 'MomentDetailPage',
  PROFILE: 'ProfilePage',
  GROUP_DETAIL: 'GroupDetailPage',
  CREATE_GROUP: 'CreateGroupPage',
  CREATE_MOMENT: 'CreateMomentPage',
  CHAT: 'ChatPage',
  LOCATION_PICKER: 'LocationPickerPage',
  SEARCH: 'SearchPage',
  SETTINGS: 'SettingsPage',
  ADVANCED_SETTINGS: 'AdvancedSettingsPage',
  PRIVACY_SETTINGS: 'PrivacySettingsPage',
  SAFETY_SETTINGS: 'SafetySettingsPage',
  RELATIONSHIP_HUB: 'RelationshipHubPage',
  ANNOUNCEMENTS: 'AnnouncementsPage',
  PHOTO_PREVIEW: 'PhotoPreviewPage',
  DEBUG_PANEL: 'DebugPanel',
  SUBSCRIPTIONS: 'SubscriptionsPage',
  ACCOUNTS: 'AccountsPage',
  MY_SHARES: 'MySharesPage',
  SHARE_RESOLUTION: 'ShareResolutionPage',
  EVENT_DETAIL: 'EventDetailPage',
  PROFILE_EDIT: 'ProfileEditPage',
  NEARBY_USERS: 'NearbyUsersListPage',
  ADMIN_OVERVIEW: 'AdminOverviewPage'
};

// 默认配置（服务端 ClientConfig 加载前的回退值）
export interface DefaultConfigType {
  NEARBY_RADIUS: number;
  MOMENT_RADIUS: number;
  PAGE_SIZE: number;
  PAGE_SIZE_LARGE: number;
  REQUEST_TIMEOUT: number;
  LOCATION_CACHE_TIME: number;
  MAX_MOMENT_IMAGES: number;
  MAX_MOMENT_TAGS: number;
}

export const DefaultConfig: DefaultConfigType = {
  NEARBY_RADIUS: 5000,         // 附近搜索默认半径 5km
  MOMENT_RADIUS: 2000,         // 瞬间搜索默认半径 2km
  PAGE_SIZE: 30,               // 默认分页大小
  PAGE_SIZE_LARGE: 50,         // 大分页大小（加载更多）
  REQUEST_TIMEOUT: 30000,      // 请求超时 30 秒
  LOCATION_CACHE_TIME: 300000, // 位置缓存 5 分钟
  MAX_MOMENT_IMAGES: 9,        // 瞬间最大图片数
  MAX_MOMENT_TAGS: 10          // 瞬间最大标签数
};
