/**
 * 全局状态管理器
 * 解决@Entry组件不能使用@ObjectLink的问题
 */

import { UserResponse } from '../types/BackendApiTypes';
import { UnreadCount } from '../types/BackendApiTypes';
import { LocationInfo } from '../location/LocationUtil';
import { WebSocketService } from '../websocket/WebSocketService';
import { hilog } from '@kit.PerformanceAnalysisKit';

const DOMAIN_NUMBER: number = 0xFF08;
const TAG: string = 'GlobalStateManager';

export interface GlobalState {
  currentUser: UserResponse | null;
  isLoggedIn: boolean;
  networkConnected: boolean;
  locationEnabled: boolean;
  currentLocation: LocationInfo | null;
  unreadCount: UnreadCount;
  isNetworkConnected: boolean;
}

export class GlobalStateManager {
  private static instance: GlobalStateManager | null = null;
  private webSocketService: WebSocketService | null = null;

  private currentState: GlobalState = {
    currentUser: null,
    isLoggedIn: false,
    networkConnected: true,
    locationEnabled: false,
    currentLocation: null,
    unreadCount: {
      total_unread_count: 0
    },
    isNetworkConnected: true
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
   * 初始化全局状态管理器
   */
  async initialize(): Promise<void> {
    try {
      // 从本地存储恢复状态
      await this.loadStateFromStorage();
      
      // 初始化WebSocket服务
      this.webSocketService = WebSocketService.getInstance();
      
      hilog.info(DOMAIN_NUMBER, TAG, 'GlobalStateManager初始化成功');
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `GlobalStateManager初始化失败: ${JSON.stringify(error)}`);
      throw new Error('GlobalStateManager初始化失败');
    }
  }

  /**
   * 获取当前状态
   */
  getState(): GlobalState {
    const stateCopy: GlobalState = {
      currentUser: this.currentState.currentUser,
      isLoggedIn: this.currentState.isLoggedIn,
      networkConnected: this.currentState.networkConnected,
      locationEnabled: this.currentState.locationEnabled,
      currentLocation: this.currentState.currentLocation,
      unreadCount: this.currentState.unreadCount,
      isNetworkConnected: this.currentState.isNetworkConnected
    };
    return stateCopy;
  }

  /**
   * 更新用户信息
   */
  updateUser(user: UserResponse | null): void {
    this.currentState = {
      currentUser: user,
      isLoggedIn: user !== null,
      networkConnected: this.currentState.networkConnected,
      locationEnabled: this.currentState.locationEnabled,
      currentLocation: this.currentState.currentLocation,
      unreadCount: this.currentState.unreadCount,
      isNetworkConnected: this.currentState.isNetworkConnected
    };
    this.notifyListeners();
    this.saveStateToStorage();
  }

  /**
   * 更新位置信息
   */
  updateLocation(location: LocationInfo): void {
    this.currentState = {
      currentUser: this.currentState.currentUser,
      isLoggedIn: this.currentState.isLoggedIn,
      networkConnected: this.currentState.networkConnected,
      locationEnabled: true,
      currentLocation: location,
      unreadCount: this.currentState.unreadCount,
      isNetworkConnected: this.currentState.isNetworkConnected
    };
    this.notifyListeners();
  }

  /**
   * 更新网络状态
   */
  updateNetworkStatus(connected: boolean): void {
    this.currentState = {
      currentUser: this.currentState.currentUser,
      isLoggedIn: this.currentState.isLoggedIn,
      networkConnected: connected,
      locationEnabled: this.currentState.locationEnabled,
      currentLocation: this.currentState.currentLocation,
      unreadCount: this.currentState.unreadCount,
      isNetworkConnected: connected
    };
    this.notifyListeners();
  }

  /**
   * 更新未读消息数量
   */
  updateUnreadCount(unreadCount: UnreadCount): void {
    this.currentState = {
      currentUser: this.currentState.currentUser,
      isLoggedIn: this.currentState.isLoggedIn,
      networkConnected: this.currentState.networkConnected,
      locationEnabled: this.currentState.locationEnabled,
      currentLocation: this.currentState.currentLocation,
      unreadCount: unreadCount,
      isNetworkConnected: this.currentState.isNetworkConnected
    };
    this.notifyListeners();
  }

  /**
   * 订阅状态变化
   */
  subscribe(id: string, listener: (state: GlobalState) => void): void {
    this.listeners.set(id, listener);
    // 立即通知当前状态
    listener(this.getState());
  }

  /**
   * 取消订阅
   */
  unsubscribe(id: string): void {
    this.listeners.delete(id);
  }

  /**
   * 暂停非必要服务
   */
  pauseNonEssentialServices(): void {
    hilog.info(DOMAIN_NUMBER, TAG, '暂停非必要服务');
    // 实现暂停逻辑
  }

  /**
   * 从本地存储加载状态
   */
  private async loadStateFromStorage(): Promise<void> {
    try {
      hilog.info(DOMAIN_NUMBER, TAG, '从本地存储加载状态');
      // 实现从存储加载状态的逻辑
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `加载本地状态失败: ${JSON.stringify(error)}`);
    }
  }

  /**
   * 通知所有监听器
   */
  private notifyListeners(): void {
    const state = this.getState();
    this.listeners.forEach((listener) => {
      try {
        listener(state);
      } catch (error) {
        hilog.error(DOMAIN_NUMBER, TAG, `通知监听器失败: ${JSON.stringify(error)}`);
      }
    });
  }

  /**
   * 清理资源
   */
  clear(): void {
    try {
      this.listeners.clear();
      this.webSocketService = null;
      hilog.info(DOMAIN_NUMBER, TAG, 'GlobalStateManager资源已清理');
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `GlobalStateManager清理失败: ${JSON.stringify(error)}`);
    }
  }

  /**
   * 保存状态到本地存储
   */
  private async saveStateToStorage(): Promise<void> {
    try {
      hilog.info(DOMAIN_NUMBER, TAG, '保存状态到本地存储');
      // 实现保存状态到存储的逻辑
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `保存本地状态失败: ${JSON.stringify(error)}`);
    }
  }
}