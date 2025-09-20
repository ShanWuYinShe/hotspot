/**
 * 极简应用常量
 * 只保留核心必要的常量定义
 */

// API配置
export const API_BASE_URL = 'http://192.168.1.6:3000';
export const WS_BASE_URL = 'ws://192.168.1.6:3000/ws';

// 业务状态码
export interface BusinessCodeType {
  SUCCESS: number;
  ERROR: number;
  UNAUTHORIZED: number;
  SERVER_ERROR: number;
}

export const BusinessCode: BusinessCodeType = {
  SUCCESS: 0,
  ERROR: 1000,
  UNAUTHORIZED: 1002,
  SERVER_ERROR: 5000
};

// 存储键名
export interface StorageKeysType {
  USER_INFO: string;
  ACCESS_TOKEN: string;
  LOCATION_CACHE: string;
}

export const StorageKeys: StorageKeysType = {
  USER_INFO: 'user_info',
  ACCESS_TOKEN: 'access_token',
  LOCATION_CACHE: 'location_cache'
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
  SEARCH: string;
  SETTINGS: string;
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
  SEARCH: 'SearchPage',
  SETTINGS: 'SettingsPage'
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