/**
 * 优化后的主页面
 * 使用HarmonyOS原生能力，简化架构，AppStorage 状态管理
 */

import { PageNames } from '../../../constants/AppConstants';
import { AppRouter, LoginRouteParam } from '../../../core/router/AppRouter';
import { common } from '@kit.AbilityKit';
import { UIContext } from '@kit.ArkUI';
import { GlobalStateManager, UnreadCount, KEY_CURRENT_USER, KEY_IS_LOGGED_IN, KEY_USER_TYPE, KEY_CURRENT_LOCATION, KEY_BROWSE_ANCHOR, KEY_UNREAD_COUNT } from '../../../core/state/GlobalStateManager';
import { LocationInfo } from '../../../core/location/LocationUtil';
import { PreferencesUtil } from '../../../core/storage/PreferencesUtil';
import { LocationUtil } from '../../../core/location/LocationUtil';
import { WebSocketService } from '../../../core/network/WebSocketService';
import { MyProfile, UpdateLocationRequest, GeoPoint, AuthData } from '../../../core/types/api';
import { AuthAccountManager } from '../../../core/state/AuthAccountManager';
import { HttpUtil } from '../../../core/network/HttpUtil';
import type { ServerEvent } from '../../../core/types/WebSocketTypes';
import { PermissionManager } from '../../../core/permission/PermissionManager';
import { ApiClient } from '../../../core/api/ApiClient';
import { AppIcon } from '../../../components/SystemIcon';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { SmartNotificationService, NotificationEvent, CreateNotificationInput, NotificationMetadata, CreateNotificationOptions } from '../../../core/services/SmartNotificationService';
import { NotificationSyncService } from '../../../core/services/NotificationSyncService';
import { CacheManager } from '../../../core/cache/CacheManager';
import { NotificationCenter } from '../../../components/NotificationCenter';
import { PerformanceIndicator } from '../../../components/PerformanceIndicator';
import { I18nManager } from '../../../core/i18n/I18nManager';
import { DesignConstants } from '../../../constants/DesignConstants';
import { NearbyPage } from '../../../features/nearby/pages/NearbyPage';
import { DiscoverPage } from '../../../features/discover/pages/DiscoverPage';
import { MessagePage } from '../../../features/message/pages/MessagePage';
import { ProfilePage } from '../../../features/profile/pages/ProfilePage';
import { LoginPromptView } from '../../../components/LoginPromptView';
import { NotificationRenderer } from '../../../core/services/notification/NotificationRenderer';
import { SmartNotification, NotificationAction } from '../../../core/services/notification/NotificationDataManager';
import { NotificationBanner } from '../../../components/NotificationBanner';
import { TimerManager } from '../../../core/utils/TimerManager';
import { SystemUiController } from '../../../core/utils/SystemUiController';
import { KeyboardDismissManager } from '../../../core/utils/KeyboardDismissManager';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { AppThemeVisuals } from '../../../core/theme/AppThemeVisuals';
import { KEY_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';

const DOMAIN_NUMBER: number = 0xFF09;
const TAG: string = 'IndexPage';

// Define storage key locally since it's not exported from NotificationRenderer
const STORAGE_KEY_VISIBLE_NOTIFICATIONS = 'visibleNotifications';
const STORAGE_KEY_OPEN_NOTIFICATION_CENTER = 'openNotificationCenter';
const STORAGE_KEY_OPEN_CREATE_SHEET = 'openCreateSheet';

@Entry
@Preview
@Component
struct Index {
  /** Navigation 路由栈 */
  @Provide('NavPathStack') navPathStack: NavPathStack = new NavPathStack();

  @StorageLink(KEY_CURRENT_USER) currentUser: MyProfile | null = null;
  @StorageLink(KEY_IS_LOGGED_IN) @Watch('onLoginStateChanged') isLoggedIn: boolean = false;
  @StorageLink(KEY_USER_TYPE) userType: string = '';
  @StorageLink(KEY_CURRENT_LOCATION) currentLocation: LocationInfo | null = null;
  @StorageLink(KEY_UNREAD_COUNT) unreadCount: UnreadCount = { total_unread_count: 0 };
  @StorageLink('safeAreaTop') safeAreaTop: number = 0;
  @StorageLink('safeAreaBottom') safeAreaBottom: number = 0;
  @StorageLink(STORAGE_KEY_VISIBLE_NOTIFICATIONS) visibleNotifications: SmartNotification[] = [];
  @StorageLink(STORAGE_KEY_OPEN_NOTIFICATION_CENTER) @Watch('onNotificationCenterRequested') openNotificationCenter: boolean = false;
  @StorageLink(STORAGE_KEY_OPEN_CREATE_SHEET) @Watch('onCreateSheetRequested') openCreateSheet: boolean = false;
  @StorageLink(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'default_clean_light';

  @State currentTabIndex: number = 0;
  @State previousTabIndex: number = 0;
  @State isInitialized: boolean = false;
  @State lastError: string = '';
  @State showNotificationCenter: boolean = false;
  @State showPerformanceDetails: boolean = false;
  @State showCreateSheet: boolean = false;

  private stateManager = GlobalStateManager.getInstance();
  private webSocketService = WebSocketService.getInstance();
  private smartNotificationService = SmartNotificationService.getInstance();
  private notificationSyncService = NotificationSyncService.getInstance();
  private cacheManager = CacheManager.getInstance();
  private i18nManager = I18nManager.getInstance();
  private uiContext: UIContext | null = null;
  private abilityContext: common.UIAbilityContext | null = null;
  private tabsController: TabsController = new TabsController();
  private wsNotificationsBound: boolean = false;
  private locationSyncTimerId: number = 0;
  private hasInitialLocationSynced: boolean = false;
  /** 缓存监控定时器 ID */
  private cacheMonitorTimerId: number = 0;
  /** 智能通知监听器引用（用于清理） */
  private smartNotificationListener: ((event: NotificationEvent, notification: SmartNotification, extra?: Object) => void) | null = null;
  /** WS 通知处理函数引用（用于 aboutToDisappear 解绑） */
  private wsNotificationHandler: ((event: ServerEvent) => Promise<void>) | null = null;

  async aboutToAppear() {
    SystemUiController.syncWithCurrentTheme();
    if (!AppStorage.has(STORAGE_KEY_OPEN_NOTIFICATION_CENTER)) {
      AppStorage.setOrCreate<boolean>(STORAGE_KEY_OPEN_NOTIFICATION_CENTER, false);
    }
    if (!AppStorage.has(STORAGE_KEY_OPEN_CREATE_SHEET)) {
      AppStorage.setOrCreate<boolean>(STORAGE_KEY_OPEN_CREATE_SHEET, false);
    }
    // 初始化全局路由
    AppRouter.init(this.navPathStack);

    try {
      this.uiContext = this.getUIContext();
      this.abilityContext = this.uiContext.getHostContext() as common.UIAbilityContext;
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `获取UIContext失败: ${JSON.stringify(error)}`);
    }

    if (this.uiContext) {
      const keyboardManager = KeyboardDismissManager.getInstance();
      keyboardManager.init(this.uiContext, 'GlobalKeyboardDismissTarget');
    }

    if (this.abilityContext) {
      this.i18nManager.setContext(this.abilityContext);
      await this.i18nManager.initialize();
    }

    await this.initializeApp();
  }

  private getCurrentThemePalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  private getBrandAccent(): string {
    return this.getCurrentThemePalette().accentBackground;
  }

  private getBrandAccentText(): string {
    return this.getCurrentThemePalette().accentText;
  }

  private getTabBarBackground(): string {
    return this.getCurrentThemePalette().tabBarBackground;
  }

  private getTabBarVisualStyle() {
    return AppThemeVisuals.getTabBarVisualStyle(this.resolvedThemeId);
  }

  private getTabInactiveColor(): ResourceColor {
    return this.getCurrentThemePalette().tabBarInactive;
  }

  aboutToDisappear() {
    // 清理定时任务（避免页面销毁后继续跑）
    if (this.locationSyncTimerId) {
      TimerManager.clearInterval(this.locationSyncTimerId);
      this.locationSyncTimerId = 0;
    }
    // 停止持续位置监听
    LocationUtil.stopLocationListening();
    // 清理缓存监控定时器
    if (this.cacheMonitorTimerId) {
      TimerManager.clearInterval(this.cacheMonitorTimerId);
      this.cacheMonitorTimerId = 0;
    }
    // 清理智能通知监听器
    if (this.smartNotificationListener) {
      this.smartNotificationService.removeEventListener(NotificationEvent.CREATED, this.smartNotificationListener);
      this.smartNotificationListener = null;
    }
    // 清理 WebSocket 通知事件监听
    if (this.wsNotificationHandler) {
      const wsEventTypes: string[] = [
        'ConnectionEstablished', 'NewFollower', 'MutualFollow',
        'MomentLiked', 'MomentCommented', 'CommentLiked', 'CommentReply',
        'MessageMentioned', 'GroupInviteReceived', 'GroupInviteResponse',
        'GroupApplicationReceived', 'GroupApplicationResponse', 'SystemAnnouncement',
      ];
      wsEventTypes.forEach((t: string) => {
        this.webSocketService.off(t, this.wsNotificationHandler!);
      });
      this.wsNotificationHandler = null;
      this.wsNotificationsBound = false;
    }
  }

  private async initializeApp() {
    try {
      hilog.info(DOMAIN_NUMBER, TAG, '🚀 开始应用初始化');

      if (!this.abilityContext) {
        const errorMsg = 'UIAbilityContext未初始化';
        hilog.error(DOMAIN_NUMBER, TAG, errorMsg);
        this.lastError = errorMsg;
        return;
      }

      try {
        const configResp = await ApiClient.getClientConfig();
        if (configResp.code === 0 && configResp.data) {
          this.stateManager.updateClientConfig(configResp.data);
          hilog.info(DOMAIN_NUMBER, TAG, `✅ 客户端配置已缓存: features.moments=${configResp.data.features?.enable_moments}`);
        }
      } catch (e) {
        hilog.warn(DOMAIN_NUMBER, TAG, `客户端配置加载失败: ${JSON.stringify(e)}`);
      }

      // 初始化权限管理器（必须在使用任何权限 API 之前）
      await PermissionManager.getInstance().initialize(this.abilityContext);
      hilog.info(DOMAIN_NUMBER, TAG, '✅ 权限管理器初始化完成');

      await this.stateManager.initialize();
      await this.initializeSmartServices();

      const persistedToken = await PreferencesUtil.get<string>('access_token', '');
      const persistedLocation = await PreferencesUtil.getObject<LocationInfo>('current_location');
      if (persistedLocation) {
        this.stateManager.updateLocation(persistedLocation);
        this.initBrowseAnchorIfNeeded(persistedLocation);
      }

      // 冷启动时验证 token 有效性并刷新 profile（与前端 restoreSessionFromServer 对齐）
      if (persistedToken) {
        await this.validateSessionOnColdStart();
      }
      // ===== 持续位置追踪（与 MapKit 蓝点共享 Location Kit 数据源） =====
      try {
        const hasPermission: boolean = await PermissionManager.getInstance().requestLocationPermission();
        if (hasPermission) {
          // 启动持续位置监听（5s 间隔），实时更新 currentLocation
          const started = await LocationUtil.startLocationListening((locInfo: LocationInfo) => {
            this.stateManager.updateLocation(locInfo);
            this.initBrowseAnchorIfNeeded(locInfo);
            // 首次获取到真实位置后，立即同步到服务器
            if (!this.hasInitialLocationSynced) {
              this.hasInitialLocationSynced = true;
              this.syncLocationToServer(locInfo);
            }
          });
          if (started) {
            hilog.info(DOMAIN_NUMBER, TAG, '✅ 持续位置监听已启动');
          } else {
            // 监听启动失败，回退到单次定位
            hilog.warn(DOMAIN_NUMBER, TAG, '持续监听启动失败，回退到单次定位');
            const location = await LocationUtil.getCurrentLocation();
            if (location) {
              this.stateManager.updateLocation(location);
              this.initBrowseAnchorIfNeeded(location);
              // 立即同步到服务器
              if (!this.hasInitialLocationSynced) {
                this.hasInitialLocationSynced = true;
                this.syncLocationToServer(location);
              }
            }
          }
        } else {
          hilog.warn(DOMAIN_NUMBER, TAG, '位置权限未授予');
        }
      } catch {
        hilog.warn(DOMAIN_NUMBER, TAG, '位置初始化失败');
      }

      // 定时同步位置到服务端（5分钟一次），所有用户（包括临时用户）均需上报
      if (this.locationSyncTimerId === 0) {
        this.locationSyncTimerId = TimerManager.setInterval(async () => {
          if (!this.currentLocation) return;
          this.syncLocationToServer(this.currentLocation);
        }, 5 * 60 * 1000, '定时同步用户位置');
      }

      await this.initializeWebSocket();

      this.isInitialized = true;
      hilog.info(DOMAIN_NUMBER, TAG, '✅ 应用初始化完成');
    } catch (error) {
      const errorMsg = `应用初始化失败: ${error}`;
      hilog.error(DOMAIN_NUMBER, TAG, errorMsg);
      this.lastError = errorMsg;
      this.isInitialized = true; 
    }
  }

  private async onLoginStateChanged(): Promise<void> {
    if (this.isLoggedIn) {
      await this.notificationSyncService.syncUnreadCount().catch(() => {});
    } else {
      this.notificationSyncService.reset();
      this.stateManager.updateUnreadCount({ total_unread_count: 0 });
    }
  }

  private onNotificationCenterRequested(): void {
    if (this.openNotificationCenter) {
      this.showNotificationCenter = true;
      AppStorage.setOrCreate<boolean>(STORAGE_KEY_OPEN_NOTIFICATION_CENTER, false);
    }
  }

  private onCreateSheetRequested(): void {
    if (this.openCreateSheet) {
      this.showCreateSheet = true;
      AppStorage.setOrCreate<boolean>(STORAGE_KEY_OPEN_CREATE_SHEET, false);
    }
  }

  /**
   * 同步真实位置到服务端。所有用户（登录 / 临时）均需上报。
   */
  private syncLocationToServer(location: LocationInfo): void {
    const geoPoint: GeoPoint = {
      latitude: location.latitude,
      longitude: location.longitude
    };
    const locationReq: UpdateLocationRequest = {
      location: {
        point: geoPoint,
        location_name: location.locationName || '未知位置'
      }
    };
    ApiClient.updateLocation(locationReq).then((resp) => {
      if (resp.code !== 0) {
        hilog.warn(DOMAIN_NUMBER, TAG, `位置上传失败: ${resp.message || resp.code}`);
      }
    }).catch(() => {
      hilog.warn(DOMAIN_NUMBER, TAG, '位置同步异常');
    });
  }

  /**
   * 首次定位后，若浏览锚点尚未设置，则将当前位置（WGS84 → GCJ02）设为初始浏览锚点。
   * 已有锚点（用户手动选择或持久化恢复）时不覆盖。
   */
  private initBrowseAnchorIfNeeded(wgs84Location: LocationInfo): void {
    const currentAnchor = AppStorage.get<LocationInfo | null>(KEY_BROWSE_ANCHOR);
    if (currentAnchor) return;

    const anchor: LocationInfo = {
      latitude: wgs84Location.latitude,
      longitude: wgs84Location.longitude,
      locationName: wgs84Location.locationName
    };
    AppStorage.setOrCreate<LocationInfo | null>(KEY_BROWSE_ANCHOR, anchor);
    hilog.info(DOMAIN_NUMBER, TAG, `✅ 初始浏览锚点已设置(WGS84): ${anchor.locationName ?? '未知'}`);
  }

  /**
   * 冷启动时向服务端验证当前 token 是否仍然有效，并刷新 profile。
   * 与前端 restoreSessionFromServer() 逻辑对齐。
   */
  private async validateSessionOnColdStart(): Promise<void> {
    try {
      hilog.info(DOMAIN_NUMBER, TAG, '🔑 冷启动：验证 token 有效性');
      const statusResp = await ApiClient.checkAuthStatus();
      if (statusResp.code !== 0 || !statusResp.data?.is_valid) {
        hilog.warn(DOMAIN_NUMBER, TAG, '冷启动：token 无效，清除登录态');
        this.stateManager.updateUser(null);
        await AuthAccountManager.clearAllAccounts();
        return;
      }

      const profileResp = await ApiClient.getMyProfile();
      if (profileResp.code !== 0 || !profileResp.data) {
        hilog.warn(DOMAIN_NUMBER, TAG, '冷启动：获取 profile 失败，清除登录态');
        this.stateManager.updateUser(null);
        await AuthAccountManager.lockCurrent();
        return;
      }

      // 更新 profile 到内存
      this.stateManager.updateUser(profileResp.data);

      // 更新 token 过期时间（如果 checkStatus 返回了 expires_at）
      if (statusResp.data.expires_at) {
        const currentAuth = await AuthAccountManager.getCurrentAuth();
        if (currentAuth) {
          const updatedAuth: AuthData = {
            user_id: currentAuth.user_id,
            access_token: currentAuth.access_token,
            refresh_token: currentAuth.refresh_token,
            access_token_expires_at: statusResp.data.expires_at,
            refresh_token_expires_at: currentAuth.refresh_token_expires_at
          };
          await HttpUtil.setToken(updatedAuth.access_token);
          await AuthAccountManager.upsertAndActivate(updatedAuth);
        }
      }

      hilog.info(DOMAIN_NUMBER, TAG, '✅ 冷启动：token 有效，profile 已刷新');
    } catch (error) {
      hilog.warn(DOMAIN_NUMBER, TAG, `冷启动验证失败: ${error}`);
    }
  }

  private async getAccessToken(): Promise<string | null> {
    try {
      return await PreferencesUtil.get('access_token', '');
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `获取访问令牌失败: ${error}`);
      return null;
    }
  }

  private async initializeWebSocket() {
    try {
      hilog.info(DOMAIN_NUMBER, TAG, '🔌 初始化WebSocket连接');
      if (this.isLoggedIn && this.currentUser) {
        const token = await this.getAccessToken();
        if (token) {
          await this.webSocketService.connect(token);
          this.bindWsNotificationsOnce();
        }
      }
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `WebSocket初始化失败: ${error}`);
    }
  }

  /**
   * WS 通知桥：订阅 notification 类型，并将事件转成 SmartNotification
   *
   * 说明：
   * - 后端推送侧会同时检查“持久化偏好 + 实时订阅”
   * - App 端这里订阅所有通知类型即可，最终是否推送由后端控制
   * - 避免重复注册：只绑定一次（重连后会通过 SubscribeAck/内部重订阅恢复）
   */
  private bindWsNotificationsOnce() {
    if (this.wsNotificationsBound) return;
    this.wsNotificationsBound = true;

    // 通知事件为自动推送，无需显式订阅

    // 通用处理：把“与我相关的通知事件”落到 SmartNotificationService
    const handle: (event: ServerEvent) => Promise<void> = async (event: ServerEvent) => {
      try {
        switch (event.type) {
          case 'ConnectionEstablished': {
            // Unread counts are now fetched via notifications/unread_count HTTP API
            break;
          }
          case 'NewFollower': {
            const p: Object = event.payload as Object;
            const payloadRecord = p as Record<string, Object>;
            const metadata = new NotificationMetadata();
            metadata.type = 'new_follower';
            metadata.user = payloadRecord.actor;
            const notifInput: CreateNotificationInput = {
              type: 'social',
              priority: 'medium',
              title: '新关注者',
              message: `${(payloadRecord.actor as Record<string, Object>).nickname} 关注了你`,
              read: false,
              actionable: true,
              metadata: metadata
            };
            const options: CreateNotificationOptions = { immediate: true };
            await this.smartNotificationService.createNotification(notifInput, options);
            break;
          }
          case 'MutualFollow': {
            const p: Object = event.payload as Object;
            const payloadRecord = p as Record<string, Object>;
            const metadata = new NotificationMetadata();
            metadata.type = 'mutual_follow';
            metadata.user = payloadRecord.actor;
            const notifInput: CreateNotificationInput = {
              type: 'social',
              priority: 'medium',
              title: '成为好友',
              message: `你和 ${(payloadRecord.actor as Record<string, Object>).nickname} 互相关注了`,
              read: false,
              actionable: true,
              metadata: metadata
            };
            const options: CreateNotificationOptions = { immediate: true };
            await this.smartNotificationService.createNotification(notifInput, options);
            break;
          }
          // Note: 'Unfollowed' event type is not in ServerEventType enum, removed
          case 'MomentLiked': {
            const p: Object = event.payload as Object;
            const payloadRecord = p as Record<string, Object>;
            const metadata = new NotificationMetadata();
            metadata.type = 'moment_like';
            metadata.targetId = String(payloadRecord.moment_id);
            const notifInput: CreateNotificationInput = {
              type: 'social',
              priority: 'low',
              title: '收到点赞',
              message: `${(payloadRecord.actor as Record<string, Object>).nickname} 赞了你的瞬间`,
              read: false,
              actionable: true,
              metadata: metadata
            };
            const options: CreateNotificationOptions = { immediate: true };
            await this.smartNotificationService.createNotification(notifInput, options);
            break;
          }
          case 'MomentCommented': {
            const p: Object = event.payload as Object;
            const payloadRecord = p as Record<string, Object>;
            const metadata = new NotificationMetadata();
            metadata.type = 'moment_comment';
            metadata.targetId = String(payloadRecord.moment_id);
            const notifInput: CreateNotificationInput = {
              type: 'social',
              priority: 'medium',
              title: '新评论',
              message: `${(payloadRecord.actor as Record<string, Object>).nickname}: ${String(payloadRecord.content)}`,
              read: false,
              actionable: true,
              metadata: metadata
            };
            const options: CreateNotificationOptions = { immediate: true };
            await this.smartNotificationService.createNotification(notifInput, options);
            break;
          }
          case 'CommentReply': {
            const p: Object = event.payload as Object;
            const payloadRecord = p as Record<string, Object>;
            const metadata = new NotificationMetadata();
            metadata.type = 'comment_reply';
            metadata.targetId = String(payloadRecord.moment_id);
            const notifInput: CreateNotificationInput = {
              type: 'social',
              priority: 'medium',
              title: '评论被回复',
              message: `${(payloadRecord.actor as Record<string, Object>).nickname}: ${String(payloadRecord.content)}`,
              read: false,
              actionable: true,
              metadata: metadata
            };
            const options: CreateNotificationOptions = { immediate: true };
            await this.smartNotificationService.createNotification(notifInput, options);
            break;
          }
          case 'CommentLiked': {
            const p: Object = event.payload as Object;
            const payloadRecord = p as Record<string, Object>;
            const metadata = new NotificationMetadata();
            metadata.type = 'comment_like';
            metadata.targetId = String(payloadRecord.comment_id);
            metadata.moment_id = String(payloadRecord.moment_id);
            const notifInput: CreateNotificationInput = {
              type: 'social',
              priority: 'low',
              title: '评论收到点赞',
              message: `${(payloadRecord.actor as Record<string, Object>).nickname} 赞了你的评论`,
              read: false,
              actionable: true,
              metadata: metadata
            };
            const options: CreateNotificationOptions = { immediate: true };
            await this.smartNotificationService.createNotification(notifInput, options);
            break;
          }
          case 'MessageMentioned': {
            const p: Object = event.payload as Object;
            const payloadRecord = p as Record<string, Object>;
            const targetType = String(payloadRecord.target_type);
            const metadata = new NotificationMetadata();
            metadata.type = 'message_mentioned';
            metadata.moment_id = String(payloadRecord.moment_id);
            if (typeof payloadRecord.comment_id === 'string') {
              metadata.comment_id = payloadRecord.comment_id as string;
            }
            metadata.targetId = targetType === 'comment'
              ? (metadata.comment_id ?? metadata.moment_id)
              : metadata.moment_id;
            const notifInput: CreateNotificationInput = {
              type: 'social',
              priority: 'medium',
              title: '被提及',
              message: `${(payloadRecord.actor as Record<string, Object>).nickname} 在${targetType === 'comment' ? '评论' : '动态'}中提到了你`,
              read: false,
              actionable: true,
              metadata: metadata
            };
            const options: CreateNotificationOptions = { immediate: true };
            await this.smartNotificationService.createNotification(notifInput, options);
            break;
          }
          case 'GroupInviteReceived': {
            const p: Object = event.payload as Object;
            const payloadRecord = p as Record<string, Object>;
            const groupRecord = payloadRecord.group as Record<string, Object>;
            const metadata = new NotificationMetadata();
            metadata.type = 'group_invite';
            metadata.targetId = String(groupRecord.group_id);
            const notifInput: CreateNotificationInput = {
              type: 'social',
              priority: 'high',
              title: '群组邀请',
              message: `${(payloadRecord.inviter as Record<string, Object>).nickname} 邀请你加入「${String(groupRecord.name)}」`,
              read: false,
              actionable: true,
              metadata: metadata
            };
            const options: CreateNotificationOptions = { immediate: true };
            await this.smartNotificationService.createNotification(notifInput, options);
            break;
          }
          case 'GroupInviteResponse': {
            const p: Object = event.payload as Object;
            const payloadRecord = p as Record<string, Object>;
            const metadata = new NotificationMetadata();
            metadata.type = 'group_invite_response';
            metadata.targetId = String(payloadRecord.group_id);
            const notifInput: CreateNotificationInput = {
              type: 'social',
              priority: payloadRecord.status === 'accepted' ? 'medium' : 'low',
              title: payloadRecord.status === 'accepted' ? '邀请已接受' : '邀请被拒绝',
              message: payloadRecord.status === 'accepted' ? '对方已接受你的群组邀请' : '对方拒绝了你的群组邀请',
              read: false,
              actionable: true,
              metadata: metadata
            };
            const options: CreateNotificationOptions = { immediate: true };
            await this.smartNotificationService.createNotification(notifInput, options);
            break;
          }
          case 'GroupApplicationReceived': {
            const p: Object = event.payload as Object;
            const payloadRecord = p as Record<string, Object>;
            const metadata = new NotificationMetadata();
            metadata.type = 'group_application';
            metadata.targetId = String(payloadRecord.group_id);
            const notifInput: CreateNotificationInput = {
              type: 'social',
              priority: 'high',
              title: '入群申请',
              message: `${(payloadRecord.applicant as Record<string, Object>).nickname} 申请加入群组`,
              read: false,
              actionable: true,
              metadata: metadata
            };
            const options: CreateNotificationOptions = { immediate: true };
            await this.smartNotificationService.createNotification(notifInput, options);
            break;
          }
          case 'GroupApplicationResponse': {
            const p: Object = event.payload as Object;
            const payloadRecord = p as Record<string, Object>;
            const groupRecord = payloadRecord.group as Record<string, Object>;
            const metadata = new NotificationMetadata();
            metadata.type = 'group_application_response';
            metadata.targetId = String(groupRecord.group_id);
            const notifInput: CreateNotificationInput = {
              type: 'social',
              priority: payloadRecord.status === 'accepted' ? 'medium' : 'low',
              title: payloadRecord.status === 'accepted' ? '申请已通过' : '申请被拒绝',
              message: payloadRecord.status === 'accepted' ? `你已加入「${String(groupRecord.name)}」` : `你的入群申请被拒绝`,
              read: false,
              actionable: true,
              metadata: metadata
            };
            const options: CreateNotificationOptions = { immediate: true };
            await this.smartNotificationService.createNotification(notifInput, options);
            break;
          }
          case 'SystemAnnouncement': {
            const p: Object = event.payload as Object;
            const payloadRecord = p as Record<string, Object>;
            const priority = payloadRecord.severity === 'error' ? 'critical' : (payloadRecord.severity === 'warning' ? 'high' : 'medium');
            const metadata = new NotificationMetadata();
            metadata.type = 'system_announcement';
            metadata.targetId = String(payloadRecord.announcement_id);
            const notifInput: CreateNotificationInput = {
              type: 'system',
              priority: priority as 'low' | 'medium' | 'high' | 'critical',
              title: String(payloadRecord.title),
              message: String(payloadRecord.content),
              read: false,
              actionable: true,
              metadata: metadata
            };
            const options: CreateNotificationOptions = { immediate: true };
            await this.smartNotificationService.createNotification(notifInput, options);
            break;
          }
        }
      } catch (err) {
        hilog.warn(DOMAIN_NUMBER, TAG, `WS通知处理失败: ${JSON.stringify(err)}`);
      }
    };

    // 逐事件绑定（避免 '*' 里做大量无关 switch）
    [
      'ConnectionEstablished',
      'NewFollower',
      'MutualFollow',
      'MomentLiked',
      'MomentCommented',
      'CommentLiked',
      'CommentReply',
      'MessageMentioned',
      'GroupInviteReceived',
      'GroupInviteResponse',
      'GroupApplicationReceived',
      'GroupApplicationResponse',
      'SystemAnnouncement',
    ].forEach((t: string) => {
      this.webSocketService.on(t, handle);
    });
    this.wsNotificationHandler = handle;
  }

  private async initializeSmartServices(): Promise<void> {
    try {
      // 缓存命中率监控（使用 TimerManager 以便统一清理）
      this.cacheMonitorTimerId = TimerManager.setInterval(() => {
        const stats = this.cacheManager.getStats();
        if (stats.hitRate < 50 && stats.totalRequests > 100) {
          hilog.warn(DOMAIN_NUMBER, TAG, `缓存命中率较低: ${stats.hitRate}%`);
        }
      }, 30000);

      // 存储监听器引用以便 aboutToDisappear 清理
      this.smartNotificationListener = (_event: NotificationEvent, notification: SmartNotification): void => {
        hilog.info(DOMAIN_NUMBER, TAG, `收到通知: ${notification.title}`);
        NotificationRenderer.getInstance().showNotification(notification);
        void this.notificationSyncService.syncUnreadCount().catch(() => {});
      };
      this.smartNotificationService.addEventListener(NotificationEvent.CREATED, this.smartNotificationListener);

      if (this.isLoggedIn) {
        await this.notificationSyncService.syncUnreadCount().catch(() => {});
      } else {
        this.notificationSyncService.reset();
      }

    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `智能服务初始化失败: ${JSON.stringify(error)}`);
    }
  }

  private handleTabChange(index: number) {
    // "+" 按钮 (index 2) 不是真实标签，拦截并弹出创建菜单
    if (index === 2) {
      this.tabsController.changeIndex(this.previousTabIndex);
      this.currentTabIndex = this.previousTabIndex;
      this.showCreateActionSheet();
      return;
    }
    this.previousTabIndex = this.currentTabIndex;
    this.currentTabIndex = index;
  }

  private showCreateActionSheet(): void {
    if (!this.isLoggedIn) {
      const loginParams = new LoginRouteParam();
      loginParams.reason = 'action_required';
      loginParams.action = '创建内容';
      loginParams.redirectUrl = PageNames.INDEX;
      loginParams.restoreAction = 'open_create_sheet';
      AppRouter.push(PageNames.LOGIN, loginParams);
      return;
    }
    this.showCreateSheet = true;
  }

  /**
   * 通用带图标标签 Builder（SymbolSpan）
   * 用于发现、消息、我的等确认可用的系统符号
   */
  @Builder
  TabBuilderIcon(title: string, targetIndex: number, icon: AppIcon) {
    Column({ space: 4 }) {
      Column({ space: 5 }) {
        Row() {
          Text() {
              SymbolSpan($r(icon))
                .fontSize(this.currentTabIndex === targetIndex ? 24 : 22)
                .fontColor([this.currentTabIndex === targetIndex ? this.getBrandAccent() : this.getTabInactiveColor()])
          }
        }
        .justifyContent(FlexAlign.Center)
        .padding({ left: 8, right: 8, top: 4, bottom: 4 })

        Row()
          .width(this.currentTabIndex === targetIndex ? this.getTabBarVisualStyle().indicatorWidth : 0)
          .height(this.getTabBarVisualStyle().indicatorHeight)
          .borderRadius(this.getTabBarVisualStyle().indicatorBorderRadius)
          .backgroundColor(this.currentTabIndex === targetIndex ? this.getBrandAccent() : Color.Transparent)
      }
      .alignItems(HorizontalAlign.Center)

      Text(title)
        .fontColor(this.currentTabIndex === targetIndex ? this.getBrandAccent() : this.getTabInactiveColor())
        .fontSize(DesignConstants.FONT.SIZE.XS)
        .fontWeight(this.currentTabIndex === targetIndex ? FontWeight.Bold : FontWeight.Medium)
    }
    .width('100%')
    .height(this.getTabBarVisualStyle().itemHeight)
    .justifyContent(FlexAlign.Center)
    .backgroundColor(Color.Transparent)
    .onClick(() => {
      this.handleTabChange(targetIndex);
      if (targetIndex !== 2) {
        this.tabsController.changeIndex(targetIndex);
      }
    })
  }

      /** 中间「+」发布按钮 Builder */
  @Builder
  CreateTabBuilder() {
    Column({ space: 4 }) {
      Row() {
        Text() {
          SymbolSpan($r(AppIcon.ADD))
            .fontSize(22)
            .fontColor([this.currentTabIndex === 2 ? this.getBrandAccent() : this.getBrandAccentText()])
        }
      }
      .width(this.getTabBarVisualStyle().createButtonWidth)
      .height(this.getTabBarVisualStyle().createButtonHeight)
      .justifyContent(FlexAlign.Center)
      .backgroundColor(this.currentTabIndex === 2 ? this.getTabBarVisualStyle().activeBackgroundColor : this.getBrandAccent())
      .borderRadius(DesignConstants.RADIUS.FULL)
      .shadow(this.currentTabIndex === 2 ? this.getTabBarVisualStyle().activeGlow : this.getTabBarVisualStyle().shadow)
      .border({
        width: this.getTabBarVisualStyle().borderWidth,
        color: this.currentTabIndex === 2 ? this.getBrandAccent() : this.getTabBarVisualStyle().borderColor
      })

      Text('发布')
        .fontColor(this.currentTabIndex === 2 ? this.getBrandAccent() : this.getTabInactiveColor())
        .fontSize(DesignConstants.FONT.SIZE.XS)
        .fontWeight(this.currentTabIndex === 2 ? FontWeight.Bold : FontWeight.Medium)
    }
    .width('100%')
    .height(this.getTabBarVisualStyle().itemHeight)
    .justifyContent(FlexAlign.Center)
    .backgroundColor(Color.Transparent)
    .onClick(() => {
      this.handleTabChange(2);
    })
  }

  build() {
    Stack({ alignContent: Alignment.Top }) {
      if (!this.isInitialized) {
        Column() {
          LoadingProgress()
            .width(48)
            .height(48)
            .color(this.getCurrentThemePalette().accentBackground)

          Text($r('app.string.common_loading'))
            .fontSize(DesignConstants.FONT.SIZE.LG)
            .fontColor(this.getCurrentThemePalette().textSecondary)
            .margin({ top: 16 })
        }
        .width('100%')
        .height('100%')
        .justifyContent(FlexAlign.Center)
        .backgroundColor(this.getCurrentThemePalette().pageBackground)
        .expandSafeArea([SafeAreaType.SYSTEM], [SafeAreaEdge.TOP, SafeAreaEdge.BOTTOM])
        .transition(
          TransitionEffect.OPACITY.animation({ duration: 320, curve: Curve.FastOutSlowIn })
            .combine(TransitionEffect.move(TransitionEdge.BOTTOM).animation({ duration: 320, curve: Curve.FastOutSlowIn }))
        )
      } else if (this.lastError) {
        Column({ space: 16 }) {
          Text('应用初始化失败')
            .fontSize(18)
            .fontWeight(FontWeight.Bold)
            .fontColor(this.getCurrentThemePalette().textPrimary)
          Text(this.lastError)
            .fontSize(DesignConstants.FONT.SIZE.MD)
            .fontColor(this.getCurrentThemePalette().textSecondary)
            .textAlign(TextAlign.Center)
            .constraintSize({ maxWidth: 280 })
          Button('重试')
            .type(ButtonType.Normal)
            .borderRadius(DesignConstants.RADIUS.XL)
            .onClick(() => {
              this.lastError = '';
              this.isInitialized = false;
              this.initializeApp();
            })
        }
        .width('100%')
        .height('100%')
        .justifyContent(FlexAlign.Center)
        .backgroundColor(this.getCurrentThemePalette().pageBackground)
        .expandSafeArea([SafeAreaType.SYSTEM], [SafeAreaEdge.TOP, SafeAreaEdge.BOTTOM])
      } else {
        Stack({ alignContent: Alignment.Bottom }) {
          // 底部背景容器：与页面背景无缝融合
          Column() {}
            .width('100%')
            .height(this.getTabBarVisualStyle().height + this.safeAreaBottom)
            .backgroundColor(this.getTabBarBackground())

          Column() {
        Navigation(this.navPathStack) {
        Tabs({ barPosition: BarPosition.End, controller: this.tabsController }) {
          // 1. 附近（含动态/用户/群组子标签）
          TabContent() {
            NearbyPage()
          }
          .tabBar(this.TabBuilderIcon('附近', 0, AppIcon.NEARBY))

          // 2. 发现（热门话题/推荐用户/热门瞬间）
          TabContent() {
            DiscoverPage()
          }
          .tabBar(this.TabBuilderIcon('发现', 1, AppIcon.RECOMMEND))

          // 3. 发布（+按钮，拦截后弹出 ActionSheet）
          TabContent() {
            // 占位：不会真正显示，点击时拦截弹出菜单
            Column() {}
          }
          .tabBar(this.CreateTabBuilder())

          // 4. 消息
          TabContent() {
            if (this.isLoggedIn) {
              Column() {
                if (this.userType === 'temp') {
                  this.buildUpgradeBanner()
                }
                MessagePage()
              }
              .width('100%')
              .height('100%')
            } else {
              LoginPromptView({
                title: '消息功能需要登录',
                subtitle: '登录后可与好友实时聊天、加入群组讨论',
                icon: AppIcon.MESSAGE
              })
            }
          }
          .tabBar(this.TabBuilderIcon('消息', 3, AppIcon.MESSAGE))

          // 5. 我的
          TabContent() {
            ProfilePage()
          }
          .tabBar(this.TabBuilderIcon('我的', 4, AppIcon.PERSON))
        }
        .barHeight(this.getTabBarVisualStyle().height)
        .scrollable(false)
        .barMode(BarMode.Fixed)
        .divider({ strokeWidth: 0, color: Color.Transparent, startMargin: 0, endMargin: 0 })
        .onChange((index: number) => this.handleTabChange(index))
        .barBackgroundColor(Color.Transparent)
        .border({ width: { top: 0 }, color: Color.Transparent })
        .backgroundColor(Color.Transparent)
        .bindSheet($$this.showCreateSheet, this.CreateSheetContent(), {
          height: SheetSize.FIT_CONTENT,
          dragBar: true,
          showClose: true,
          backgroundColor: this.getCurrentThemePalette().sheetBackground
        })
        }
        .mode(NavigationMode.Stack)
        .hideTitleBar(true)
        .hideBackButton(true)
        .backgroundColor(Color.Transparent)
        .transition(
          TransitionEffect.OPACITY.animation({ duration: 320, curve: Curve.FastOutSlowIn })
            .combine(TransitionEffect.move(TransitionEdge.BOTTOM).animation({ duration: 320, curve: Curve.FastOutSlowIn }))
        )
        }
        .width('100%')
        .height('100%')
        .padding({ bottom: this.safeAreaBottom })
        }
        .width('100%')
        .height('100%')
      }

      if (this.currentTabIndex === 1) {
        Column() {
          Row() {
            Blank().layoutWeight(1)
            Stack({ alignContent: Alignment.TopEnd }) {
              Button() {
                Text() {
                  SymbolSpan($r(this.unreadCount.total_unread_count > 0 ? AppIcon.NOTIFICATION_FILL : AppIcon.NOTIFICATION))
                    .fontSize(18)
                    .fontColor([this.getCurrentThemePalette().textPrimary])
                }
              }
              .type(ButtonType.Normal)
              .width(40)
              .height(40)
              .backgroundColor(this.getCurrentThemePalette().surfaceElevated)
              .borderRadius(DesignConstants.RADIUS.FULL)
              .border({ width: 1, color: this.getCurrentThemePalette().borderColor })
              .shadow(this.getCurrentThemePalette().controlShadow)
              .onClick(() => {
                this.showNotificationCenter = true;
              })

              if (this.unreadCount.total_unread_count > 0) {
                Text(this.unreadCount.total_unread_count > 99 ? '99+' : `${this.unreadCount.total_unread_count}`)
                  .fontSize(10)
                  .fontWeight(FontWeight.Bold)
                  .fontColor(Color.White)
                  .padding({ left: 6, right: 6, top: 2, bottom: 2 })
                  .backgroundColor(this.getCurrentThemePalette().accentBackground)
                  .borderRadius(DesignConstants.RADIUS.FULL)
                  .translate({ x: 8, y: -6 })
              }
            }
            .margin({ top: this.getSafeAreaTopPadding(), right: DesignConstants.SPACING.LG })
          }
          .width('100%')
        }
        .expandSafeArea([SafeAreaType.SYSTEM], [SafeAreaEdge.TOP])
      }

      if (this.visibleNotifications.length > 0) {
        Column({ space: DesignConstants.SPACING.SM }) {
          ForEach(this.visibleNotifications, (notification: SmartNotification) => {
            NotificationBanner({
              notification: notification,
              onClose: () => {
                NotificationRenderer.getInstance().hideNotification(notification.id, 'manual');
              },
              onBannerClick: () => {
                NotificationRenderer.getInstance().handleClick(notification);
              },
              onAction: (action: NotificationAction) => {
                NotificationRenderer.getInstance().handleAction(action, notification);
              }
            })
            .transition(
              TransitionEffect.OPACITY.animation({ duration: 350, curve: Curve.FastOutSlowIn })
                .combine(TransitionEffect.move(TransitionEdge.TOP).animation({ duration: 350, curve: Curve.FastOutSlowIn }))
            )
          }, (item: SmartNotification) => item.id)
        }
        .width('100%')
        .padding({ top: 12, left: 16, right: 16 })
        .expandSafeArea([SafeAreaType.SYSTEM], [SafeAreaEdge.TOP])
      }

      if (this.showNotificationCenter) {
        Stack() {
          Column()
            .width('100%')
            .height('100%')
            .backgroundColor('#66000000')
            .onClick(() => {
              this.showNotificationCenter = false;
            })

          Column() {
            NotificationCenter({
              onClose: () => {
                this.showNotificationCenter = false;
              }
            })
          }
          .width('100%')
          .height('100%')
          .backgroundColor(this.getCurrentThemePalette().pageBackground)
        }
        .width('100%')
        .height('100%')
        .expandSafeArea([SafeAreaType.SYSTEM], [SafeAreaEdge.TOP, SafeAreaEdge.BOTTOM])
        .transition(
          TransitionEffect.OPACITY.animation({ duration: 320, curve: Curve.EaseInOut })
            .combine(TransitionEffect.move(TransitionEdge.BOTTOM).animation({ duration: 320, curve: Curve.EaseInOut }))
        )
      }

      if (this.showPerformanceDetails) {
        Column() {
          PerformanceIndicator({
            showDetails: true,
            compact: false
          })
        }
        .width('100%')
        .height('100%')
        .backgroundColor(this.getCurrentThemePalette().pageBackground)
        .expandSafeArea([SafeAreaType.SYSTEM], [SafeAreaEdge.TOP, SafeAreaEdge.BOTTOM])
        .transition(
          TransitionEffect.OPACITY.animation({ duration: 320, curve: Curve.EaseInOut })
            .combine(TransitionEffect.move(TransitionEdge.BOTTOM).animation({ duration: 320, curve: Curve.EaseInOut }))
        )
        .onClick(() => {
          this.showPerformanceDetails = false;
        })
      }

      Button('')
        .id('GlobalKeyboardDismissTarget')
        .width(1)
        .height(1)
        .opacity(0)
        .focusable(true)
        .onClick(() => {})
    }
    .width('100%')
    .height('100%')
    .backgroundColor(this.getCurrentThemePalette().pageBackground)
    .expandSafeArea([SafeAreaType.SYSTEM], [SafeAreaEdge.BOTTOM])
  }

  /** 获取安全区域顶部内边距（状态栏高度 + 间距） */
  private getSafeAreaTopPadding(): number {
    return (this.safeAreaTop > 0 ? this.safeAreaTop : DesignConstants.SPACING.LG) + DesignConstants.SPACING.SM;
  }

  @Builder
  private buildUpgradeBanner() {
    Row({ space: DesignConstants.SPACING.SM }) {
      Text('当前为临时用户，注册账号可保留所有数据')
        .fontSize(DesignConstants.FONT.SIZE.SM)
        .fontColor(this.getCurrentThemePalette().accentText)
        .layoutWeight(1)
        .maxLines(1)
        .textOverflow({ overflow: TextOverflow.Ellipsis })

      Button('去注册')
        .type(ButtonType.Normal)
        .fontSize(DesignConstants.FONT.SIZE.SM)
        .height(28)
        .backgroundColor(this.getCurrentThemePalette().surfaceBackground)
        .fontColor(this.getCurrentThemePalette().accentBackground)
        .borderRadius(DesignConstants.RADIUS.XL)
        .onClick(() => {
          AppRouter.push(PageNames.LOGIN);
        })
    }
    .width('100%')
    .padding({
      left: DesignConstants.SPACING.LG,
      right: DesignConstants.SPACING.LG,
      top: this.getSafeAreaTopPadding(),
      bottom: DesignConstants.SPACING.SM
    })
    .backgroundColor(this.getCurrentThemePalette().accentBackground)
    .borderRadius({ bottomLeft: DesignConstants.RADIUS.LG, bottomRight: DesignConstants.RADIUS.LG })
    .expandSafeArea([SafeAreaType.SYSTEM], [SafeAreaEdge.TOP])
  }

  @Builder
  CreateSheetContent() {
    Column({ space: 0 }) {
      Text('创建内容')
        .fontSize(DesignConstants.FONT.SIZE.XL)
        .fontWeight(FontWeight.Bold)
        .fontColor(this.getCurrentThemePalette().textPrimary)
        .margin({ top: DesignConstants.SPACING.SM, bottom: 20 })

      // 发布瞬间
      Row({ space: DesignConstants.SPACING.LG }) {
        Column() {
          Text() {
            SymbolSpan($r(AppIcon.CAMERA))
              .fontSize(26)
              .fontColor([this.getCurrentThemePalette().accentBackground])
          }
        }
        .width(52)
        .height(52)
        .justifyContent(FlexAlign.Center)
        .backgroundColor(this.getCurrentThemePalette().surfaceSubtle)
        .borderRadius(DesignConstants.RADIUS.MD)

        Column({ space: DesignConstants.SPACING.XS }) {
          Text('发布瞬间')
            .fontSize(DesignConstants.FONT.SIZE.LG)
            .fontWeight(FontWeight.Medium)
            .fontColor(this.getCurrentThemePalette().textPrimary)
          Text('分享你身边的精彩时刻')
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .fontColor(this.getCurrentThemePalette().textSecondary)
        }
        .alignItems(HorizontalAlign.Start)
        .layoutWeight(1)

        Text() {
          SymbolSpan($r(AppIcon.FORWARD))
            .fontSize(18)
            .fontColor([this.getCurrentThemePalette().textSecondary])
        }
      }
      .width('100%')
      .padding(DesignConstants.SPACING.LG)
      .backgroundColor(this.getCurrentThemePalette().surfaceBackground)
      .borderRadius(DesignConstants.RADIUS.XXL)
      .shadow(this.getCurrentThemePalette().elevatedShadow)
      .border({ width: 1, color: this.getCurrentThemePalette().borderColor })
      .onClick(() => {
        this.showCreateSheet = false;
        AppRouter.push(PageNames.CREATE_MOMENT);
      })

      // 创建群组
      Row({ space: DesignConstants.SPACING.LG }) {
        Column() {
          Text() {
            SymbolSpan($r(AppIcon.GROUP))
              .fontSize(26)
              .fontColor([this.getCurrentThemePalette().accentBackground])
          }
        }
        .width(52)
        .height(52)
        .justifyContent(FlexAlign.Center)
        .backgroundColor(this.getCurrentThemePalette().surfaceSubtle)
        .borderRadius(DesignConstants.RADIUS.MD)

        Column({ space: DesignConstants.SPACING.XS }) {
          Text('创建群组')
            .fontSize(DesignConstants.FONT.SIZE.LG)
            .fontWeight(FontWeight.Medium)
            .fontColor(this.getCurrentThemePalette().textPrimary)
          Text('创建一个附近的群组讨论')
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .fontColor(this.getCurrentThemePalette().textSecondary)
        }
        .alignItems(HorizontalAlign.Start)
        .layoutWeight(1)

        Text() {
          SymbolSpan($r(AppIcon.FORWARD))
            .fontSize(18)
            .fontColor([this.getCurrentThemePalette().textSecondary])
        }
      }
      .width('100%')
      .padding(DesignConstants.SPACING.LG)
      .backgroundColor(this.getCurrentThemePalette().surfaceBackground)
      .borderRadius(DesignConstants.RADIUS.XXL)
      .shadow(this.getCurrentThemePalette().elevatedShadow)
      .border({ width: 1, color: this.getCurrentThemePalette().borderColor })
      .margin({ top: DesignConstants.SPACING.SM })
      .onClick(() => {
        this.showCreateSheet = false;
        AppRouter.push(PageNames.CREATE_GROUP);
      })
    }
    .width('100%')
    .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG,
      top: DesignConstants.SPACING.LG, bottom: 40 })
  }
}
