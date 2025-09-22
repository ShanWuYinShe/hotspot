/**
 * 全局状态管理器
 * 解决@Entry组件不能使用@ObjectLink的问题
 */

import { UserResponse } from '../types/BackendApiTypes';
import { UnreadCount } from '../types/BackendApiTypes';
import { LocationInfo } from '../location/LocationUtil';
import { WebSocketService } from '../websocket/WebSocketService';

export interface GlobalState {
  currentUser: UserResponse | null;
  isLoggedIn: boolean;
  networkConnected: boolean;
  locationEnabled: boolean;
  unreadCounts: UnreadCount;
  currentLocation: LocationInfo | null;
  isNetworkConnected: boolean;
}

export class GlobalStateManager {
  private static instance: GlobalStateManager | null = null;
  private webSocketService: WebSocketService | null = null;
  private state: GlobalState = {
    currentUser: null,
    isLoggedIn: false,
    networkConnected: true,
    locationEnabled: false,
    unreadCounts: {},
    currentLocation: null,
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
      
      console.log('GlobalStateManager初始化成功');
    } catch (error) {
      console.error('GlobalStateManager初始化失败:', error);
      throw new Error('GlobalStateManager初始化失败');
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
      currentLocation: this.state.currentLocation,
      isNetworkConnected: this.state.isNetworkConnected
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
   * 从本地存储加载状态
   */
  private async loadStateFromStorage(): Promise<void> {
    try {
      // 这里应该从preferences加载保存的状态
      // 暂时使用默认状态
      console.log('从本地存储加载状态');
    } catch (error) {
      console.error('加载本地状态失败:', error);
    }
  }

  /**
   * 重新连接WebSocket（如果需要）
   */
  reconnectWebSocketIfNeeded(): void {
    if (this.state.isLoggedIn && this.state.isNetworkConnected && this.webSocketService) {
      this.webSocketService.connect('');
    }
  }

  /**
   * 暂停非必要服务
   */
  pauseNonEssentialServices(): void {
    // 暂停位置更新等非必要服务
    console.log('暂停非必要服务');
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
      currentLocation: null,
      isNetworkConnected: true
    };
    this.notifyListeners();
  }

  /**
   * 清理资源
   */
  async cleanup(): Promise<void> {
    try {
      // 断开WebSocket连接
      if (this.webSocketService) {
        this.webSocketService.disconnect();
      }
      
      // 保存状态到本地存储
      await this.saveStateToStorage();
      
      // 清理订阅者
      this.listeners.clear();
      
      // 重置状态
      this.clear();
      
      console.log('GlobalStateManager资源已清理');
    } catch (error) {
      console.error('GlobalStateManager清理失败:', error);
    }
  }

  /**
   * 保存状态到本地存储
   */
  private async saveStateToStorage(): Promise<void> {
    try {
      // 这里应该保存重要状态到preferences
      console.log('保存状态到本地存储');
    } catch (error) {
      console.error('保存本地状态失败:', error);
    }
  }
}
