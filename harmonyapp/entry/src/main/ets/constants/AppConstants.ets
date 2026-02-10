/**
 * 极简应用常量
 * 只保留核心必要的常量定义
 */

import { DEBUG } from 'BuildProfile';

// API 配置：根据构建模式自动切换
// debug  → 开发服务器（局域网 IP）
// release → 正式域名
const DEV_SERVER = 'http://192.168.0.109:8888';
const PROD_SERVER = 'https://api.chaoset.com';

export const SERVER_BASE_URL: string = DEBUG ? DEV_SERVER : PROD_SERVER;

// 由同一个 BaseUrl 推导出 API / WebSocket 地址（避免多处硬编码）
export const API_BASE_URL: string = SERVER_BASE_URL;
export const WS_BASE_URL: string = SERVER_BASE_URL.replace(/^http:/, 'ws:').replace(/^https:/, 'wss:') + '/ws';

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
  UNAUTHORIZED: 40100,
  SERVER_ERROR: 50000
};

// 存储键名
export interface StorageKeysType {
  USER_INFO: string;
  ACCESS_TOKEN: string;
  LOCATION_CACHE: string;
  SERVER_BASE_URL: string;
  AUTH_ACCOUNTS: string;
  AUTH_CURRENT_USER_ID: string;
}

export const StorageKeys: StorageKeysType = {
  USER_INFO: 'user_info',
  ACCESS_TOKEN: 'access_token',
  LOCATION_CACHE: 'location_cache',
  SERVER_BASE_URL: 'server_base_url',
  // 多账号：保存所有账户的 AppAuthData（含 refresh_token），并记录当前活跃账号
  AUTH_ACCOUNTS: 'auth_accounts',
  AUTH_CURRENT_USER_ID: 'auth_current_user_id'
};

// 页面名称
export interface PageNamesType {
  INDEX: string;
  LOGIN: string;
  GROUP: string;
  MESSAGE: string;
  MOMENT: string;
  PROFILE: string;
  GROUP_DETAIL: string;
  CREATE_GROUP: string;
  CREATE_MOMENT: string;
  CHAT: string;
  MAP: string;
  LOCATION_PICKER: string;
  SEARCH: string;
  SETTINGS: string;
  ADVANCED_SETTINGS: string;
  PHOTO_PREVIEW: string;
  DEBUG_PANEL: string;
  SUBSCRIPTIONS: string;
  ACCOUNTS: string;
}

export const PageNames: PageNamesType = {
  INDEX: 'Index',
  LOGIN: 'LoginPage',
  GROUP: 'GroupPage',
  MESSAGE: 'MessagePage',
  MOMENT: 'MomentPage',
  PROFILE: 'ProfilePage',
  GROUP_DETAIL: 'GroupDetailPage',
  CREATE_GROUP: 'CreateGroupPage',
  CREATE_MOMENT: 'CreateMomentPage',
  CHAT: 'ChatPage',
  MAP: 'MapPage',
  LOCATION_PICKER: 'LocationPickerPage',
  SEARCH: 'SearchPage',
  SETTINGS: 'SettingsPage',
  ADVANCED_SETTINGS: 'AdvancedSettingsPage',
  PHOTO_PREVIEW: 'PhotoPreviewPage',
  DEBUG_PANEL: 'DebugPanel',
  SUBSCRIPTIONS: 'SubscriptionsPage',
  ACCOUNTS: 'AccountsPage'
};

// 默认配置
export interface DefaultConfigType {
  SEARCH_RADIUS: number;
  PAGE_SIZE: number;
  REQUEST_TIMEOUT: number;
  LOCATION_CACHE_TIME: number;
}

export const DefaultConfig: DefaultConfigType = {
  SEARCH_RADIUS: 2000,        // 默认搜索半径2km
  PAGE_SIZE: 20,              // 默认分页大小
  REQUEST_TIMEOUT: 30000,     // 请求超时30秒
  LOCATION_CACHE_TIME: 300000 // 位置缓存5分钟
};
