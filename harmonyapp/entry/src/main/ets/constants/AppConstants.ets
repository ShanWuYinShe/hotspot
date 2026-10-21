/**
 * 极简应用常量
 * 只保留核心必要的常量定义
 */

import { DEBUG } from 'BuildProfile';

// API 配置：根据构建模式自动切换
// debug  → 开发服务器（docker compose 的 nginx 网关，端口 = .env NGINX_PORT 默认 8888）
// release → 正式域名（与 Web 网关同域同构；api 子域未建 DNS，禁止引用）
// 模拟器与真机统一走 127.0.0.1:8888 + hdc 反向转发（等价 adb reverse，不依赖
// SLIRP NAT——实测部分模拟器实例 10.0.2.2 用户态 NAT 不可达或延迟极高）：
// - 模拟器：`hdc -t 127.0.0.1:5555 rport tcp:8888 tcp:8888`
// - 真机：`hdc -t <设备序列号> fport tcp:8888 tcp:8888`（USB 反向转发）
// 开发环境默认指向 QEMU 宿主别名 10.0.2.2：
// - 模拟器里 127.0.0.1 指模拟器自身，**到达不了宿主**，此前导致所有页面停在加载/错误态；
// - 真机同样用不了 127.0.0.1，需要改成本机 LAN IP（DebugPanel 可覆盖，见 PreferencesUtil 的 SERVER_BASE_URL）；
// - 宿主 nginx 监听 0.0.0.0:8888，模拟器内 `ping 10.0.2.2` 实测可达（2026-10-21 验证）。
const DEV_SERVER = 'http://10.0.2.2:8888';
const PROD_SERVER = 'https://chaoset.com';

export const SERVER_BASE_URL: string = DEBUG ? DEV_SERVER : PROD_SERVER;

// Web 端站点入口（nginx 网关同域服务前端静态页），分享链接指向此域名。
// 路径与 Web 路由表保持一致：/users/:id、/groups/:id、/moments/:id、/share/:code
// （frontend/src/lib/routes.ts）。debug 走本地网关，dev compose 同样服务 Web 前端。
const DEV_WEB_SERVER = 'http://127.0.0.1:8888';
const PROD_WEB_SERVER = 'https://chaoset.com';

export const WEB_BASE_URL: string = DEBUG ? DEV_WEB_SERVER : PROD_WEB_SERVER;

// 构建旗标透出：调试类入口（调试面板等）按此门控，release 构建不可达。
export const DEBUG_BUILD: boolean = DEBUG;

// 由同一个 BaseUrl 推导出 API / WebSocket 地址（避免多处硬编码）
export const API_BASE_URL: string = SERVER_BASE_URL;
export const WS_BASE_URL: string = SERVER_BASE_URL.replace(/^http:/, 'ws:').replace(/^https:/, 'wss:') + '/ws/v1';

// 存储键名
export interface StorageKeysType {
  USER_INFO: string;
  AUTH_DATA: string;
  ACCESS_TOKEN: string;
  LOCATION_CACHE: string;
  SERVER_BASE_URL: string;
  AUTH_ACCOUNTS: string;
  THEME_MODE: string;
  DARK_MODE_PREFERENCE: string;
  DEVICE_ID: string;
  SWITCH_TAB_REQUEST: string;
}

export const StorageKeys: StorageKeysType = {
  USER_INFO: 'user_info',
  // AUTH_DATA / ACCESS_TOKEN 敏感值已迁至 Asset 存储（TokenAssetStore）；
  // 键名常量保留供索引/兼容引用，不再写入 Preferences 明文
  AUTH_DATA: 'auth_data',
  ACCESS_TOKEN: 'access_token',
  LOCATION_CACHE: 'location_cache',
  SERVER_BASE_URL: 'server_base_url',
  // 多账号：仅保存非敏感索引（user_id 列表 + current_user_id）；
  // 各账号 AuthData（含 refresh_token）存 Asset 存储（见 TokenAssetStore）
  AUTH_ACCOUNTS: 'auth_accounts',
  THEME_MODE: 'theme_mode',
  DARK_MODE_PREFERENCE: 'dark_mode_preference',
  // spec: 持久化 device_id（UUID v4），驱动单设备单会话踢人语义
  DEVICE_ID: 'device_id',
  // 路由页请求切底部 tab 的一次性信号（如设置页「个人资料」→ 我的 tab）；
  // PhoneLayout 消费后立即复位为 -1
  SWITCH_TAB_REQUEST: 'switch_tab_request'
};

/**
 * Preferences **库名**（不是键名——键名见上方 StorageKeys）。
 *
 * HarmonyOS Preferences 按「库」隔离，本项目按用途分库：
 * - GEOTALK：业务偏好（用户信息、主题、服务器地址、浏览锚点等），PreferencesUtil 单例持有；
 * - ANCHOR_HISTORY：浏览锚点历史，AnchorHistoryUtil 单例持有（数据量大、与业务偏好
 *   生命周期不同，故独立分库）。
 *
 * 此前两处库名是散落的字面量（'geotalk' / 'anchor_history'），集中登记避免拼写漂移。
 */
export interface PreferenceStoresType {
  GEOTALK: string;
  ANCHOR_HISTORY: string;
}

export const PreferenceStores: PreferenceStoresType = {
  GEOTALK: 'geotalk',
  ANCHOR_HISTORY: 'anchor_history'
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
  PROFILE_EDIT: string;
  NEARBY_USERS: string;
  MY_REPORTS: string;
  NEARBY_GROUPS: string;
  NEARBY: string;
  NEARBY_MOMENTS: string;
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
  PROFILE_EDIT: 'ProfileEditPage',
  NEARBY_USERS: 'NearbyUsersListPage',
  MY_REPORTS: 'MyReportsPage',
  NEARBY_GROUPS: 'NearbyGroupsListPage',
  NEARBY: 'NearbyPage',
  NEARBY_MOMENTS: 'MomentsFeedPage',
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
