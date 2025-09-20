/**
 * 持久化状态管理器
 * 使用 HarmonyOS Preferences 进行状态持久化存储
 */

import { PreferencesUtil } from '../storage/PreferencesUtil';
import { UserResponse } from '../types/BackendApiTypes';
import { UnreadCount } from '../types/BackendApiTypes';
import { LocationInfo } from '../location/LocationUtil';
import { hilog } from '@kit.PerformanceAnalysisKit';

const DOMAIN_NUMBER: number = 0xFF10;
const TAG: string = 'PersistentStateManager';

export interface GlobalState {
  currentUser: UserResponse | null;
  isLoggedIn: boolean;
  networkConnected: boolean;
  locationEnabled: boolean;
  unreadCounts: UnreadCount;
  currentLocation: LocationInfo | null;
}

export class PersistentStateManager {
  private static instance: PersistentStateManager | null = null;
  private state: GlobalState = {
    currentUser: null,
    isLoggedIn: false,
    networkConnected: true,
    locationEnabled: false,
    unreadCounts: {},
    currentLocation: null
  };
  private listeners: Map<string, (state: GlobalState) => void> = new Map();
  private isInitialized: boolean = false;

  private constructor() {
  }

  static getInstance(): PersistentStateManager {
    if (!PersistentStateManager.instance) {
      PersistentStateManager.instance = new PersistentStateManager();
    }
    return PersistentStateManager.instance;
  }

  /**
   * 初始化状态管理器，从持久化存储中恢复状态
   */
  async initialize(): Promise<void> {
    if (this.isInitialized) {
      return;
    }

    try {
      hilog.info(DOMAIN_NUMBER, TAG, '🔄 开始初始化持久化状态管理器');

      // 恢复用户信息
      const userInfo = await PreferencesUtil.getObject<UserResponse>('user_info');
      const token = await PreferencesUtil.get<string>('access_token', '');

      if (userInfo && token) {
        this.state.currentUser = userInfo;
        this.state.isLoggedIn = true;
        hilog.info(DOMAIN_NUMBER, TAG, `✅ 恢复用户状态: ${userInfo.nickname}`);
      }

      // 恢复位置信息
      const locationInfo = await PreferencesUtil.getObject<LocationInfo>('current_location');
      if (locationInfo) {
        this.state.currentLocation = locationInfo;
        this.state.locationEnabled = true;
        hilog.info(DOMAIN_NUMBER, TAG, '✅ 恢复位置状态');
      }

      // 恢复网络状态
      const networkConnected = await PreferencesUtil.get<boolean>('network_connected', true);
      this.state.networkConnected = networkConnected;

      // 恢复未读消息数
      const unreadCounts = await PreferencesUtil.getObject<UnreadCount>('unread_counts');
      if (unreadCounts) {
        this.state.unreadCounts = unreadCounts;
      }

      this.isInitialized = true;
      hilog.info(DOMAIN_NUMBER, TAG, '✅ 持久化状态管理器初始化完成');

      // 通知所有监听器
      this.notifyListeners();
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `❌ 初始化失败: ${error}`);
      this.isInitialized = true; // 即使失败也标记为已初始化
    }
  }

  /**
   * 获取当前状态
   */
  getState(): GlobalState {
    return {
      currentUser: this.state.currentUser,
      isLoggedIn: this.state.isLoggedIn,
      networkConnected: this.state.networkConnected,
      locationEnabled: this.state.locationEnabled,
      unreadCounts: this.state.unreadCounts,
      currentLocation: this.state.currentLocation ? this.state.currentLocation : null
    };
  }

  /**
   * 更新用户信息并持久化
   */
  async updateUser(user: UserResponse | null): Promise<void> {
    try {
      this.state.currentUser = user;
      this.state.isLoggedIn = !!user;

      if (user) {
        await PreferencesUtil.setObject('user_info', user);
        hilog.info(DOMAIN_NUMBER, TAG, `💾 用户信息已保存: ${user.nickname}`);
      } else {
        await PreferencesUtil.remove('user_info');
        await PreferencesUtil.remove('access_token');
        hilog.info(DOMAIN_NUMBER, TAG, '🗑️ 用户信息已清除');
      }

      this.notifyListeners();
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `❌ 保存用户信息失败: ${error}`);
    }
  }

  /**
   * 更新位置信息并持久化
   */
  async updateLocation(location: LocationInfo): Promise<void> {
    try {
      this.state.currentLocation = location;
      this.state.locationEnabled = true;

      await PreferencesUtil.setObject('current_location', location);
      hilog.info(DOMAIN_NUMBER, TAG, `💾 位置信息已保存: ${location.locationName || '南天门'}`);

      this.notifyListeners();
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `❌ 保存位置信息失败: ${error}`);
    }
  }

  /**
   * 更新未读消息数并持久化
   */
  async updateUnreadCounts(counts: UnreadCount): Promise<void> {
    try {
      this.state.unreadCounts = counts;

      await PreferencesUtil.setObject('unread_counts', counts);
      hilog.info(DOMAIN_NUMBER, TAG, '💾 未读消息数已保存');

      this.notifyListeners();
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `❌ 保存未读消息数失败: ${error}`);
    }
  }

  /**
   * 更新网络状态并持久化
   */
  async updateNetworkStatus(connected: boolean): Promise<void> {
    try {
      this.state.networkConnected = connected;

      await PreferencesUtil.set('network_connected', connected);
      hilog.info(DOMAIN_NUMBER, TAG, `💾 网络状态已保存: ${connected ? '已连接' : '已断开'}`);

      this.notifyListeners();
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `❌ 保存网络状态失败: ${error}`);
    }
  }

  /**
   * 清除所有状态和持久化数据
   */
  async clearAllState(): Promise<void> {
    try {
      // 重置内存状态
      this.state = {
        currentUser: null,
        isLoggedIn: false,
        networkConnected: true,
        locationEnabled: false,
        unreadCounts: {},
        currentLocation: null
      };

      // 清除持久化数据
      await PreferencesUtil.clear();
      hilog.info(DOMAIN_NUMBER, TAG, '🗑️ 所有状态已清除');

      this.notifyListeners();
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `❌ 清除状态失败: ${error}`);
    }
  }

  /**
   * 订阅状态变化
   */
  subscribe(key: string, callback: (state: GlobalState) => void): void {
    this.listeners.set(key, callback);
    hilog.debug(DOMAIN_NUMBER, TAG, `📡 订阅状态变化: ${key}`);
  }

  /**
   * 取消订阅
   */
  unsubscribe(key: string): void {
    this.listeners.delete(key);
    hilog.debug(DOMAIN_NUMBER, TAG, `📡 取消订阅: ${key}`);
  }

  /**
   * 通知所有监听器
   */
  private notifyListeners(): void {
    const currentState = this.getState();
    this.listeners.forEach((callback, key) => {
      try {
        callback(currentState);
      } catch (error) {
        hilog.error(DOMAIN_NUMBER, TAG, `❌ 通知监听器失败 ${key}: ${error}`);
      }
    });
  }

  /**
   * 获取初始化状态
   */
  isReady(): boolean {
    return this.isInitialized;
  }
}










