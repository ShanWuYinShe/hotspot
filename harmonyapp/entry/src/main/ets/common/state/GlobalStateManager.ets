/**
 * 全局状态管理器
 * 解决@Entry组件不能使用@ObjectLink的问题
 */

import { UserResponse } from '../types/BackendApiTypes';
import { UnreadCount } from '../types/BackendApiTypes';
import { LocationInfo } from '../location/LocationUtil';

export interface GlobalState {
  currentUser: UserResponse | null;
  isLoggedIn: boolean;
  networkConnected: boolean;
  locationEnabled: boolean;
  unreadCounts: UnreadCount;
  currentLocation: LocationInfo | null;
}

export class GlobalStateManager {
  private static instance: GlobalStateManager | null = null;
  private state: GlobalState = {
    currentUser: null,
    isLoggedIn: false,
    networkConnected: true,
    locationEnabled: false,
    unreadCounts: {},
    currentLocation: null
  };
  
  private listeners: Map<string, (state: GlobalState) => void> = new Map();

  private constructor() {}

  static getInstance(): GlobalStateManager {
    if (!GlobalStateManager.instance) {
      GlobalStateManager.instance = new GlobalStateManager();
    }
    return GlobalStateManager.instance;
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
      currentLocation: this.state.currentLocation
    };
  }

  /**
   * 更新用户信息
   */
  updateUser(user: UserResponse | null): void {
    this.state.currentUser = user;
    this.state.isLoggedIn = !!user;
    this.notifyListeners();
  }

  /**
   * 更新位置信息
   */
  updateLocation(location: LocationInfo): void {
    this.state.currentLocation = location;
    this.state.locationEnabled = true;
    this.notifyListeners();
  }

  /**
   * 更新未读消息数
   */
  updateUnreadCounts(counts: UnreadCount): void {
    this.state.unreadCounts = counts;
    this.notifyListeners();
  }

  /**
   * 更新网络状态
   */
  updateNetworkStatus(connected: boolean): void {
    this.state.networkConnected = connected;
    this.notifyListeners();
  }

  /**
   * 注册状态监听器
   */
  subscribe(id: string, listener: (state: GlobalState) => void): void {
    this.listeners.set(id, listener);
  }

  /**
   * 取消状态监听器
   */
  unsubscribe(id: string): void {
    this.listeners.delete(id);
  }

  /**
   * 通知所有监听器
   */
  private notifyListeners(): void {
    const currentState = this.getState();
    for (const listener of this.listeners.values()) {
      listener(currentState);
    }
  }

  /**
   * 清理状态
   */
  clear(): void {
    this.state = {
      currentUser: null,
      isLoggedIn: false,
      networkConnected: true,
      locationEnabled: false,
      unreadCounts: {},
      currentLocation: null
    };
    this.notifyListeners();
  }
}
