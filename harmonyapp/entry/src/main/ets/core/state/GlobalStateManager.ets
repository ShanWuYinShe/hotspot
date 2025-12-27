/**
 * 全局状态管理器 (基于 AppStorage 和 PersistentStorage)
 * 符合 HarmonyOS 最佳实践
 */

import { MyProfileData } from '../types/api';
import { LocationInfo } from '../location/LocationUtil';
import { hilog } from '@kit.PerformanceAnalysisKit';

const DOMAIN_NUMBER: number = 0xFF08;
const TAG: string = 'GlobalStateManager';

/**
 * 未读计数
 */
export interface UnreadCount {
  total_unread_count: number;
  private_chat_count?: number;
  group_chat_count?: number;
}

/**
 * 全局状态接口定义 (主要用于类型引用兼容)
 */
export interface GlobalState {
  currentUser: MyProfileData | null;
  isLoggedIn: boolean;
  networkConnected: boolean;
  locationEnabled: boolean;
  currentLocation: LocationInfo | null;
  unreadCount: UnreadCount;
}

// 定义 AppStorage 的 key
export const KEY_CURRENT_USER = 'currentUser';
export const KEY_IS_LOGGED_IN = 'isLoggedIn';
export const KEY_CURRENT_LOCATION = 'currentLocation';
export const KEY_LOCATION_PICK_RESULT = 'locationPickResult';
export const KEY_BROWSE_ANCHOR = 'browseAnchor';
export const KEY_UNREAD_COUNT = 'unreadCount';
export const KEY_NETWORK_CONNECTED = 'networkConnected';
export const KEY_LAST_MESSAGE_SYNC_TIME = 'lastMessageSyncTime';

// 初始化 PersistentStorage (必须在 AppStorage 使用之前)
PersistentStorage.persistProp<MyProfileData | null>(KEY_CURRENT_USER, null);
PersistentStorage.persistProp<boolean>(KEY_IS_LOGGED_IN, false);
PersistentStorage.persistProp<LocationInfo | null>(KEY_BROWSE_ANCHOR, null);

export class GlobalStateManager {
  private static instance: GlobalStateManager | null = null;

  private constructor() {}

  static getInstance(): GlobalStateManager {
    if (!GlobalStateManager.instance) {
      GlobalStateManager.instance = new GlobalStateManager();
    }
    return GlobalStateManager.instance;
  }

  /**
   * 初始化全局状态管理器
   */
  async initialize(): Promise<void> {
    // 基础状态已经在 PersistentStorage 中初始化
    // 初始化内存状态
    if (!AppStorage.has(KEY_CURRENT_LOCATION)) {
      AppStorage.setOrCreate<LocationInfo | null>(KEY_CURRENT_LOCATION, null);
    }
    if (!AppStorage.has(KEY_LOCATION_PICK_RESULT)) {
      AppStorage.setOrCreate<LocationInfo | null>(KEY_LOCATION_PICK_RESULT, null);
    }
    if (!AppStorage.has(KEY_BROWSE_ANCHOR)) {
      AppStorage.setOrCreate<LocationInfo | null>(KEY_BROWSE_ANCHOR, null);
    }
    if (!AppStorage.has(KEY_UNREAD_COUNT)) {
      AppStorage.setOrCreate<UnreadCount>(KEY_UNREAD_COUNT, { total_unread_count: 0 });
    }
    if (!AppStorage.has(KEY_NETWORK_CONNECTED)) {
      AppStorage.setOrCreate<boolean>(KEY_NETWORK_CONNECTED, true);
    }
    if (!AppStorage.has(KEY_LAST_MESSAGE_SYNC_TIME)) {
      AppStorage.setOrCreate<number>(KEY_LAST_MESSAGE_SYNC_TIME, 0);
    }
    
    hilog.info(DOMAIN_NUMBER, TAG, 'GlobalStateManager (AppStorage) 初始化成功');
  }

  /**
   * 更新用户信息
   */
  updateUser(user: MyProfileData | null): void {
    AppStorage.setOrCreate(KEY_CURRENT_USER, user);
    AppStorage.setOrCreate(KEY_IS_LOGGED_IN, user !== null);
  }

  /**
   * 获取当前用户 (供非UI组件使用)
   */
  getCurrentUser(): MyProfileData | null | undefined {
    return AppStorage.get<MyProfileData | null>(KEY_CURRENT_USER);
  }

  /**
   * 更新位置信息
   */
  updateLocation(location: LocationInfo): void {
    AppStorage.setOrCreate(KEY_CURRENT_LOCATION, location);
  }

  /**
   * 更新网络状态
   */
  updateNetworkStatus(connected: boolean): void {
    AppStorage.setOrCreate(KEY_NETWORK_CONNECTED, connected);
  }

  /**
   * 更新未读消息数量
   */
  updateUnreadCount(unreadCount: UnreadCount): void {
    AppStorage.setOrCreate(KEY_UNREAD_COUNT, unreadCount);
  }

  /**
   * 通知消息同步
   */
  notifyMessageSync(): void {
    AppStorage.setOrCreate(KEY_LAST_MESSAGE_SYNC_TIME, Date.now());
  }

  /**
   * 清理资源 (登出时调用)
   */
  clear(): void {
    this.updateUser(null);
    this.updateUnreadCount({ total_unread_count: 0 });
    hilog.info(DOMAIN_NUMBER, TAG, 'GlobalStateManager 状态已重置');
  }
}
