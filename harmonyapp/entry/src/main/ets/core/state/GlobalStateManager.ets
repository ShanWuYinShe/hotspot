/**
 * 全局状态管理器 (基于 AppStorage 和 PersistentStorage)
 * 符合 HarmonyOS 最佳实践
 *
 * 用户三级体系：
 * - 匿名（无 token）：可浏览公开内容（瞬间、群组、地图等）
 * - 临时用户（temp，有 token）：通过 guest API 创建，可执行绝大部分写操作
 * - 注册用户（normal，有 token）：完整功能，含密码修改等
 */

import { AccountInfo, ClientConfig, AppLimits } from '../types/api';
import { PreferencesUtil } from '../storage/PreferencesUtil';
import type { UserType } from '../types/api';
import { LocationInfo, isValidLocation } from '../location/LocationUtil';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { InteractionStateStore, interactionEventHub } from './InteractionStateStore';
import { MessageSyncManager } from './MessageSyncManager';
import { AppThemeManager } from '../theme/AppThemeManager';
import { DEFAULT_RESOLVED_THEME_ID, DEFAULT_THEME_MODE, KEY_RESOLVED_THEME_ID, KEY_THEME_MODE } from '../theme/AppThemeTypes';

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
 * 全局状态接口定义（用于类型引用）
 */
export interface GlobalState {
  currentUser: AccountInfo | null;
  isLoggedIn: boolean;
  userType: string;
  networkConnected: boolean;
  locationEnabled: boolean;
  currentLocation: LocationInfo | null;
  unreadCount: UnreadCount;
}

// 定义 AppStorage 的 key
export const KEY_CURRENT_USER = 'currentUser';
export const KEY_IS_LOGGED_IN = 'isLoggedIn';
export const KEY_USER_TYPE = 'userType';
export const KEY_CURRENT_LOCATION = 'currentLocation';
export const KEY_LOCATION_PICK_RESULT = 'locationPickResult';
export const KEY_BROWSE_ANCHOR = 'browseAnchor';
export const KEY_UNREAD_COUNT = 'unreadCount';
export const KEY_NETWORK_CONNECTED = 'networkConnected';
export const KEY_LAST_MESSAGE_SYNC_TIME = 'lastMessageSyncTime';
export const KEY_CLIENT_CONFIG = 'clientConfig';
const BROWSE_ANCHOR_PREF_KEY = 'browse_anchor';

interface PersistedBrowseAnchor {
  latitude: number;
  longitude: number;
  locationName: string;
}

// PersistentStorage 不支持 null 默认值，可空类型改用 AppStorage 初始化
// 需要持久化的扩展状态统一通过 PreferencesUtil 处理
AppStorage.setOrCreate<AccountInfo | null>(KEY_CURRENT_USER, null);
PersistentStorage.persistProp<boolean>(KEY_IS_LOGGED_IN, false);
PersistentStorage.persistProp<string>(KEY_USER_TYPE, '');
AppStorage.setOrCreate<LocationInfo | null>(KEY_BROWSE_ANCHOR, null);
AppStorage.setOrCreate<string>(KEY_THEME_MODE, DEFAULT_THEME_MODE);
AppStorage.setOrCreate<string>(KEY_RESOLVED_THEME_ID, DEFAULT_RESOLVED_THEME_ID);

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
    await GlobalStateManager.restoreBrowseAnchor();
    if (!AppStorage.has(KEY_UNREAD_COUNT)) {
      AppStorage.setOrCreate<UnreadCount>(KEY_UNREAD_COUNT, { total_unread_count: 0 });
    }
    if (!AppStorage.has(KEY_NETWORK_CONNECTED)) {
      AppStorage.setOrCreate<boolean>(KEY_NETWORK_CONNECTED, true);
    }
    if (!AppStorage.has(KEY_LAST_MESSAGE_SYNC_TIME)) {
      AppStorage.setOrCreate<number>(KEY_LAST_MESSAGE_SYNC_TIME, 0);
    }
    if (!AppStorage.has(KEY_CLIENT_CONFIG)) {
      AppStorage.setOrCreate<ClientConfig | null>(KEY_CLIENT_CONFIG, null);
    }
    if (!AppStorage.has(KEY_THEME_MODE)) {
      AppStorage.setOrCreate<string>(KEY_THEME_MODE, DEFAULT_THEME_MODE);
    }
    if (!AppStorage.has(KEY_RESOLVED_THEME_ID)) {
      AppStorage.setOrCreate<string>(KEY_RESOLVED_THEME_ID, DEFAULT_RESOLVED_THEME_ID);
    }

    await AppThemeManager.initialize();

    // 初始化交互状态存储
    InteractionStateStore.getInstance();

    // 初始化消息同步管理器
    MessageSyncManager.getInstance().initialize();

    hilog.info(DOMAIN_NUMBER, TAG, 'GlobalStateManager (AppStorage) 初始化成功');
  }

  /**
   * 更新用户信息
   * isLoggedIn = (user !== null)，表示有 token（含临时用户和注册用户）
   * userType 仅区分 'temp' | 'normal'（管理员身份由 RBAC 获取）
   */
  updateUser(user: AccountInfo | null): void {
    AppStorage.setOrCreate(KEY_CURRENT_USER, user);
    AppStorage.setOrCreate(KEY_IS_LOGGED_IN, user !== null);
    AppStorage.setOrCreate(KEY_USER_TYPE, user?.user_type || '');
    // 登录/登出/切号后常驻 Tab 页（如个人中心）不会重新 mount，
    // 经事件通知订阅方刷新依赖登录身份的派生状态（管理员行等）。
    interactionEventHub.notifyAuthUserChanged(user?.user_id ?? '');
  }

  /**
   * 获取当前用户 (供非UI组件使用)
   */
  getCurrentUser(): AccountInfo | null | undefined {
    return AppStorage.get<AccountInfo | null>(KEY_CURRENT_USER);
  }

  /**
   * 获取当前用户类型
   */
  getUserType(): string {
    return AppStorage.get<string>(KEY_USER_TYPE) || '';
  }

  /**
   * 是否已认证（有 token，含临时用户和注册用户）
   */
  isAuthenticated(): boolean {
    return AppStorage.get<boolean>(KEY_IS_LOGGED_IN) === true;
  }

  /**
   * 是否为临时用户
   */
  isTempUser(): boolean {
    return this.getUserType() === 'temp';
  }

  /**
   * 是否为注册用户
   */
  isRegisteredUser(): boolean {
    return this.getUserType() === 'normal';
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
   * 更新客户端配置
   */
  updateClientConfig(config: ClientConfig): void {
    AppStorage.setOrCreate<ClientConfig | null>(KEY_CLIENT_CONFIG, config);
    hilog.info(DOMAIN_NUMBER, TAG, `客户端配置已更新`);
  }

  /**
   * 获取 AppLimits（服务端配置的限制参数）
   */
  getLimits(): AppLimits | null {
    const config = AppStorage.get<ClientConfig | null>(KEY_CLIENT_CONFIG);
    return config?.limits ?? null;
  }

  /**
   * 写入浏览锚点（AppStorage + 持久化）。所有锚点写入统一走此入口，
   * 此前锚点仅存 AppStorage，重启即丢失，用户每次启动都要重新设置。
   */
  static setBrowseAnchor(loc: LocationInfo | null): void {
    AppStorage.setOrCreate<LocationInfo | null>(KEY_BROWSE_ANCHOR, loc);
    if (loc) {
      const persisted: PersistedBrowseAnchor = {
        latitude: loc.latitude, longitude: loc.longitude, locationName: loc.locationName ?? ''
      };
      PreferencesUtil.setObject(BROWSE_ANCHOR_PREF_KEY, persisted).catch(() => {});
    } else {
      PreferencesUtil.remove(BROWSE_ANCHOR_PREF_KEY).catch(() => {});
    }
  }

  /** 启动时恢复持久化的浏览锚点（无效数据如 (0,0) 缺省值直接丢弃并清持久化） */
  static async restoreBrowseAnchor(): Promise<void> {
    try {
      const saved = await PreferencesUtil.getObject<PersistedBrowseAnchor>(BROWSE_ANCHOR_PREF_KEY);
      if (saved && typeof saved.latitude === 'number' && typeof saved.longitude === 'number') {
        const loc: LocationInfo = { latitude: saved.latitude, longitude: saved.longitude, locationName: saved.locationName ?? '' };
        if (!isValidLocation(loc)) {
          hilog.warn(DOMAIN_NUMBER, TAG, '持久化浏览锚点无效(如 (0,0))，丢弃');
          PreferencesUtil.remove(BROWSE_ANCHOR_PREF_KEY).catch(() => {});
          return;
        }
        AppStorage.setOrCreate<LocationInfo | null>(KEY_BROWSE_ANCHOR, loc);
      }
    } catch {
      hilog.warn(DOMAIN_NUMBER, TAG, '恢复浏览锚点失败');
    }
  }

  /**
   * 清理资源 (登出时调用)
   */
  clear(): void {
    this.updateUser(null);
    this.updateUnreadCount({ total_unread_count: 0 });
    GlobalStateManager.setBrowseAnchor(null);
    AppStorage.setOrCreate<LocationInfo | null>(KEY_LOCATION_PICK_RESULT, null);
    MessageSyncManager.getInstance().destroy();
    hilog.info(DOMAIN_NUMBER, TAG, 'GlobalStateManager 状态已重置');
  }
}
