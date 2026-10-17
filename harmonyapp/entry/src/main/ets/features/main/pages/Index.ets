/**
 * 优化后的主页面
 * 使用HarmonyOS原生能力，简化架构，AppStorage 状态管理
 */

import { PageNames } from '../../../constants/AppConstants';
import { AppRouter, LoginRouteParam } from '../../../core/router/AppRouter';
import { common } from '@kit.AbilityKit';
import { UIContext } from '@kit.ArkUI';
import { GlobalStateManager, UnreadCount, KEY_CURRENT_USER, KEY_IS_LOGGED_IN, KEY_CURRENT_LOCATION, KEY_BROWSE_ANCHOR, KEY_UNREAD_COUNT } from '../../../core/state/GlobalStateManager';
import { LocationInfo, isValidLocation } from '../../../core/location/LocationUtil';
import { PreferencesUtil } from '../../../core/storage/PreferencesUtil';
import { LocationUtil } from '../../../core/location/LocationUtil';
import { WebSocketService } from '../../../core/network/WebSocketService';
import { AccountInfo, UpdateLocationRequest, GeoPoint, AuthData } from '../../../core/types/api';
import { AuthAccountManager } from '../../../core/state/AuthAccountManager';
import { HttpUtil } from '../../../core/network/HttpUtil';
import type { ServerEvent } from '../../../core/types/WebSocketTypes';
import { PermissionManager } from '../../../core/permission/PermissionManager';
import { ApiClient } from '../../../core/api/ApiClient';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { SmartNotificationService, NotificationEvent, CreateNotificationInput, NotificationMetadata, CreateNotificationOptions } from '../../../core/services/SmartNotificationService';
import { NotificationSyncService } from '../../../core/services/NotificationSyncService';
import { CacheManager } from '../../../core/cache/CacheManager';
import { NotificationCenter } from '../../../components/NotificationCenter';
import { PerformanceIndicator } from '../../../components/PerformanceIndicator';
import { I18nManager } from '../../../core/i18n/I18nManager';
import { DesignConstants, textLineHeight } from '../../../constants/DesignConstants';
import { LoginPromptView } from '../../../components/LoginPromptView';
import { NotificationRenderer } from '../../../core/services/notification/NotificationRenderer';
import { SmartNotification } from '../../../core/services/notification/NotificationDataManager';
import { NotificationBanner } from '../../../components/NotificationBanner';
import { SystemButton } from '../../../components/SystemButton';
import { TimerManager } from '../../../core/utils/TimerManager';
import { SystemUiController } from '../../../core/utils/SystemUiController';
import { KeyboardDismissManager } from '../../../core/utils/KeyboardDismissManager';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { AppThemeVisuals } from '../../../core/theme/AppThemeVisuals';
import { DEFAULT_RESOLVED_THEME_ID, KEY_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';
import { useSidebarChrome, type BreakpointType } from '../../../core/responsive/Breakpoint';
import { PhoneLayout } from '../../../components/layout/PhoneLayout';
import { DesktopLayout } from '../../../components/layout/DesktopLayout';

const DOMAIN_NUMBER: number = 0xFF09;
const TAG: string = 'IndexPage';

// WS 通知桥订阅的事件清单——注册与清理共用同一份，杜绝两份清单各自漂移
//（历史上 MessageMentioned 幽灵事件与 ContentMentioned 更名即因此类双份清单漏改）。
// ConnectionEstablished/SessionRevoked 为桥内 no-op 分支（前者已由 HTTP 校准未读，
// 后者由 WebSocketService 统一处理），保留注册仅为走统一 try/catch 日志。
const WS_NOTIFICATION_EVENTS: string[] = [
  'ConnectionEstablished',
  'NewFollower',
  'MutualFollow',
  'MomentCommented',
  'CommentReply',
  'ContentMentioned',
  'ReactionAdded',
  'GroupInviteReceived',
  'GroupInviteResponse',
  'GroupInviteRejected',
  'GroupInviteRevoked',
  'GroupInviteExpired',
  'GroupApplicationReceived',
  'GroupApplicationResponse',
  'GroupApplicationCancelled',
  'GroupApplicationRejected',
  'GroupMessageReadCountUpdated',
  'GroupMemberRoleChanged',
  'ReportResolved',
  'ReportDismissed',
  'NearbyNewMoment',
  'SystemAnnouncement',
  'SessionRevoked',
];

type NotificationCategory = 'social' | 'system' | 'security';
type NotificationPriority = 'low' | 'medium' | 'high' | 'critical';

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

  @StorageLink(KEY_CURRENT_USER) currentUser: AccountInfo | null = null;
  @StorageLink(KEY_IS_LOGGED_IN) @Watch('onLoginStateChanged') isLoggedIn: boolean = false;
  @StorageLink(KEY_CURRENT_LOCATION) currentLocation: LocationInfo | null = null;
  @StorageLink(KEY_UNREAD_COUNT) unreadCount: UnreadCount = { total_unread_count: 0 };
  @StorageLink('safeAreaTop') safeAreaTop: number = 0;
  @StorageLink('safeAreaBottom') safeAreaBottom: number = 0;
  @StorageLink(STORAGE_KEY_VISIBLE_NOTIFICATIONS) visibleNotifications: SmartNotification[] = [];
  @StorageLink(STORAGE_KEY_OPEN_NOTIFICATION_CENTER) @Watch('onNotificationCenterRequested') openNotificationCenter: boolean = false;
  @StorageLink(STORAGE_KEY_OPEN_CREATE_SHEET) @Watch('onCreateSheetRequested') openCreateSheet: boolean = false;
  @StorageLink(KEY_RESOLVED_THEME_ID) @Watch('onThemeChanged') resolvedThemeId: string = DEFAULT_RESOLVED_THEME_ID;
  /** 主题切换淡入过渡用（ArkUI 无法对颜色插值，以整体 opacity 淡入缓和硬切，对齐前端 transition-colors 观感） */
  @State themeFadeOpacity: number = 1;
  @StorageProp('isTablet') isTablet: boolean = false;
  @StorageProp('currentBreakpoint') currentBreakpoint: BreakpointType = 'lg';

  @State currentTabIndex: number = 0;
  @State currentRoute: string = 'home';
  @State isInitialized: boolean = false;
  @State lastError: string = '';
  @State showNotificationCenter: boolean = false;
  @State showPerformanceDetails: boolean = false;
  @State showCreateSheet: boolean = false;
  @State isCreateTabPressed: boolean = false;

  private stateManager = GlobalStateManager.getInstance();
  private webSocketService = WebSocketService.getInstance();
  private smartNotificationService = SmartNotificationService.getInstance();
  private notificationSyncService = NotificationSyncService.getInstance();
  private cacheManager = CacheManager.getInstance();
  private i18nManager = I18nManager.getInstance();
  private uiContext: UIContext | null = null;
  private abilityContext: common.UIAbilityContext | null = null;
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
    // 清理 WebSocket 通知事件监听（与注册共用同一份清单）
    if (this.wsNotificationHandler) {
      WS_NOTIFICATION_EVENTS.forEach((t: string) => {
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

      // 定时同步位置到服务端（5分钟一次）。仅登录用户（含临时账号）上报：
      // update_location 是 /api/v1 认证端点，匿名用户上报必得 10100，
      // 每 5 分钟一次只会空转定位 + 触发认证失败链
      if (this.locationSyncTimerId === 0) {
        this.locationSyncTimerId = TimerManager.setInterval(async () => {
          if (!this.currentLocation) return;
          if (!this.isLoggedIn) return;
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

  private onThemeChanged(): void {
    // 主题已切换（resolvedThemeId 已更新为新主题）：整页微降后下一帧淡入，
    // 缓和主题硬切（ArkUI 无法做颜色插值过渡，这是最接近前端 transition-colors 500ms 的观感妥协）。
    this.themeFadeOpacity = 0.4;
    setTimeout(() => {
      this.getUIContext().animateTo({ duration: DesignConstants.ANIMATION.DURATION_MEDIUM, curve: DesignConstants.ANIMATION.CURVE_FAST_OUT_SLOW_IN }, () => {
        this.themeFadeOpacity = 1;
      });
    }, 16);
  }

  private async onLoginStateChanged(): Promise<void> {
    // 登录态切换会重建大量页面/弹窗层，此时重新解析一次主题：
    // 否则窗口底色与毛玻璃材质可能停留在浅色默认值，深色主题下出现浅灰底栏
    AppThemeManager.refreshResolvedThemePublic();
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
   * 已有锚点（用户手动选择或持久化恢复）时不覆盖；无定位/默认 (0,0) 视为无效不设锚
   * （UI 走查实测：模拟器无定位时曾把 (0,0) 写成锚点，创建群组继承后「距你 12443km」）。
   */
  private initBrowseAnchorIfNeeded(wgs84Location: LocationInfo): void {
    const currentAnchor = AppStorage.get<LocationInfo | null>(KEY_BROWSE_ANCHOR);
    if (currentAnchor) return;
    if (!isValidLocation(wgs84Location)) {
      hilog.warn(DOMAIN_NUMBER, TAG, '定位无效(缺省/0,0)，跳过初始浏览锚点设置');
      return;
    }

    const anchor: LocationInfo = {
      latitude: wgs84Location.latitude,
      longitude: wgs84Location.longitude,
      locationName: wgs84Location.locationName
    };
    GlobalStateManager.setBrowseAnchor(anchor);
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
      if (statusResp.code !== 0) {
        hilog.warn(DOMAIN_NUMBER, TAG, '冷启动：token 无效，清除登录态');
        this.stateManager.updateUser(null);
        await AuthAccountManager.clearAllAccounts();
        return;
      }

      const profileResp = await ApiClient.getAccountInfo();
      if (profileResp.code !== 0 || !profileResp.data) {
        hilog.warn(DOMAIN_NUMBER, TAG, '冷启动：获取 profile 失败，清除登录态');
        this.stateManager.updateUser(null);
        await AuthAccountManager.lockCurrent();
        return;
      }

      // 更新 profile 到内存
      this.stateManager.updateUser(profileResp.data);

      // 更新 token 过期时间（如果 checkStatus 返回了 expires_at）
      const statusData = statusResp.data;
      if (statusData && statusData.expires_at) {
        const currentAuth = await AuthAccountManager.getCurrentAuth();
        if (currentAuth) {
          const updatedAuth: AuthData = {
            client_type: 'app',
            user_id: currentAuth.user_id,
            access_token: currentAuth.access_token,
            refresh_token: currentAuth.refresh_token,
            access_token_expires_at: statusData.expires_at,
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
  /**
   * 通知构造工厂：统一 CreateNotificationInput 组装与投递。
   * 历史上每个事件分支重复 8 行构造块（17 处），字段一处改漏即静默漂移；
   * 收敛后各分支只负责「metadata 组装 + 文案」，组装/投递/未读同步在此单点。
   * @param syncUnread 投递后是否同步未读数（决定角标的高优事件为 true）
   */
  private async pushSmartNotification(
    category: NotificationCategory,
    priority: NotificationPriority,
    title: string,
    message: string,
    metadata: NotificationMetadata,
    syncUnread: boolean = false
  ): Promise<void> {
    const notifInput: CreateNotificationInput = {
      type: category,
      priority: priority,
      title: title,
      message: message,
      read: false,
      actionable: true,
      metadata: metadata
    };
    const options: CreateNotificationOptions = { immediate: true };
    await this.smartNotificationService.createNotification(notifInput, options);
    if (syncUnread) {
      await this.notificationSyncService.syncUnreadCount();
    }
  }

  /** 取 payload 中的嵌套对象（缺省返回空记录，调用方免空判）。 */
  private static payloadRecord(payload: Object, key: string): Record<string, Object> {
    const record = payload as Record<string, Object>;
    return (record[key] as Record<string, Object>) ?? {};
  }

  /** 取 payload 字符串字段。 */
  private static payloadString(payload: Object, key: string): string {
    const v = (payload as Record<string, Object>)[key];
    return v === undefined || v === null ? '' : String(v);
  }

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
            const payloadRecord = Index.payloadRecord(event.payload, '');
            const metadata = new NotificationMetadata();
            metadata.type = 'new_follower';
            metadata.user = payloadRecord.actor;
            await this.pushSmartNotification(
              'social', 'medium', '新关注者',
              `${(payloadRecord.actor as Record<string, Object>).nickname} 关注了你`, metadata);
            break;
          }
          case 'MutualFollow': {
            const payloadRecord = Index.payloadRecord(event.payload, '');
            const metadata = new NotificationMetadata();
            metadata.type = 'mutual_follow';
            metadata.user = payloadRecord.actor;
            await this.pushSmartNotification(
              'social', 'medium', '成为好友',
              `你和 ${(payloadRecord.actor as Record<string, Object>).nickname} 互相关注了`, metadata);
            break;
          }
          // Note: 'Unfollowed' event type is not in ServerEventType enum, removed
          // Note: 'MomentLiked'/'CommentLiked' 事件已从契约移除，点赞态由 reactions[].reacted_by_me 表达
          case 'MomentCommented': {
            const payloadRecord = Index.payloadRecord(event.payload, '');
            const metadata = new NotificationMetadata();
            metadata.type = 'moment_comment';
            metadata.targetId = String(payloadRecord.moment_id);
            await this.pushSmartNotification(
              'social', 'medium', '新评论',
              `${(payloadRecord.actor as Record<string, Object>).nickname}: ${String(payloadRecord.content)}`, metadata);
            break;
          }
          case 'CommentReply': {
            const payloadRecord = Index.payloadRecord(event.payload, '');
            const metadata = new NotificationMetadata();
            metadata.type = 'comment_reply';
            metadata.targetId = String(payloadRecord.moment_id);
            await this.pushSmartNotification(
              'social', 'medium', '评论被回复',
              `${(payloadRecord.actor as Record<string, Object>).nickname}: ${String(payloadRecord.content)}`, metadata);
            break;
          }
          case 'ContentMentioned': {
            const payloadRecord = Index.payloadRecord(event.payload, '');
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
            await this.pushSmartNotification(
              'social', 'medium', '被提及',
              `${(payloadRecord.actor as Record<string, Object>).nickname} 在${targetType === 'comment' ? '评论' : '动态'}中提到了你`, metadata);
            break;
          }
          case 'GroupInviteReceived': {
            const payloadRecord = Index.payloadRecord(event.payload, '');
            const groupRecord = Index.payloadRecord(event.payload, 'group');
            const metadata = new NotificationMetadata();
            metadata.type = 'group_invite';
            metadata.targetId = String(groupRecord.group_id);
            await this.pushSmartNotification(
              'social', 'high', '群组邀请',
              `${(payloadRecord.actor as Record<string, Object>).nickname} 邀请你加入「${String(groupRecord.name)}」`, metadata);
            break;
          }
          case 'GroupInviteResponse': {
            const payloadRecord = Index.payloadRecord(event.payload, '');
            const metadata = new NotificationMetadata();
            metadata.type = 'group_invite_response';
            metadata.targetId = String(payloadRecord.group_id);
            // 契约：GroupInviteResponse 仅接受时推送且无 status 字段，事件名即语义
            await this.pushSmartNotification(
              'social', 'medium', '邀请已接受', '对方已接受你的群组邀请', metadata);
            break;
          }
          case 'GroupInviteRejected': {
            const payloadRecord = Index.payloadRecord(event.payload, '');
            const metadata = new NotificationMetadata();
            metadata.type = 'group_invite_rejected';
            metadata.targetId = String(payloadRecord.group_id);
            const nickname = (Index.payloadRecord(event.payload, 'rejected_by').nickname as string) ?? '';
            const reason = payloadRecord.reason ? String(payloadRecord.reason) : '';
            await this.pushSmartNotification(
              'social', 'medium', '邀请被拒绝',
              reason ? `${nickname} 拒绝了你的群组邀请：${reason}` : `${nickname} 拒绝了你的群组邀请`,
              metadata, true);
            break;
          }
          case 'GroupApplicationReceived': {
            const payloadRecord = Index.payloadRecord(event.payload, '');
            const metadata = new NotificationMetadata();
            metadata.type = 'group_application';
            metadata.targetId = String(payloadRecord.group_id);
            await this.pushSmartNotification(
              'social', 'high', '入群申请',
              `${(payloadRecord.applicant as Record<string, Object>).nickname} 申请加入群组`, metadata);
            break;
          }
          case 'GroupApplicationResponse': {
            const payloadRecord = Index.payloadRecord(event.payload, '');
            const groupRecord = Index.payloadRecord(event.payload, 'group');
            const metadata = new NotificationMetadata();
            metadata.type = 'group_application_response';
            metadata.targetId = String(groupRecord.group_id);
            // 契约：GroupApplicationResponse 仅接受时推送且无 status 字段，事件名即语义
            await this.pushSmartNotification(
              'social', 'medium', '申请已通过', `你已加入「${String(groupRecord.name)}」`, metadata);
            break;
          }
          case 'SystemAnnouncement': {
            const payloadRecord = Index.payloadRecord(event.payload, '');
            const priority: NotificationPriority = payloadRecord.severity === 'error'
              ? 'critical'
              : (payloadRecord.severity === 'warning' ? 'high' : 'medium');
            const metadata = new NotificationMetadata();
            metadata.type = 'system_announcement';
            metadata.targetId = String(payloadRecord.announcement_id);
            await this.pushSmartNotification(
              'system', priority, String(payloadRecord.title), String(payloadRecord.content), metadata);
            break;
          }
          case 'ReactionAdded': {
            const payloadRecord = Index.payloadRecord(event.payload, '');
            const metadata = new NotificationMetadata();
            metadata.type = 'reaction_added';
            metadata.targetId = String(payloadRecord.target_id);
            const emoji = typeof payloadRecord.emoji === 'string' ? payloadRecord.emoji : '';
            const message = emoji
              ? `${(payloadRecord.actor as Record<string, Object>).nickname} 对你的内容送出了 ${emoji}`
              : `${(payloadRecord.actor as Record<string, Object>).nickname} 对你的内容添加了表情反应`;
            await this.pushSmartNotification(
              'social', 'low', '收到表情反应', message, metadata, true);
            break;
          }
          case 'GroupInviteRevoked':
          case 'GroupInviteExpired': {
            const payloadRecord = Index.payloadRecord(event.payload, '');
            const metadata = new NotificationMetadata();
            metadata.type = event.type === 'GroupInviteRevoked' ? 'group_invite_revoked' : 'group_invite_expired';
            metadata.targetId = String(payloadRecord.group_id);
            await this.pushSmartNotification(
              'social', 'medium', '群组邀请已失效', '你收到的群组邀请已被撤销或过期',
              metadata, true);
            break;
          }
          case 'GroupApplicationCancelled':
          case 'GroupApplicationExpired': {
            const payloadRecord = Index.payloadRecord(event.payload, '');
            const metadata = new NotificationMetadata();
            metadata.type = 'group_application_cancelled';
            metadata.targetId = String(payloadRecord.group_id);
            await this.pushSmartNotification(
              'social', 'medium', '入群申请已取消', '群成员的入群申请已被取消或过期',
              metadata, true);
            break;
          }
          case 'GroupApplicationRejected': {
            const payloadRecord = Index.payloadRecord(event.payload, '');
            const metadata = new NotificationMetadata();
            metadata.type = 'group_application_rejected';
            metadata.targetId = String(payloadRecord.application_id);
            const reason = payloadRecord.reason ? `：${String(payloadRecord.reason)}` : '';
            await this.pushSmartNotification(
              'social', 'medium', '入群申请被拒绝', `你的入群申请被拒绝${reason}`,
              metadata, true);
            break;
          }
          case 'ReportResolved':
          case 'ReportDismissed': {
            const payloadRecord = Index.payloadRecord(event.payload, '');
            const metadata = new NotificationMetadata();
            metadata.type = event.type === 'ReportResolved' ? 'report_resolved' : 'report_dismissed';
            metadata.targetId = String(payloadRecord.report_id);
            await this.pushSmartNotification(
              'social', 'medium',
              event.type === 'ReportResolved' ? '举报已处理' : '举报被驳回',
              event.type === 'ReportResolved' ? '你的举报已处理完成' : '你的举报未通过审核',
              metadata, true);
            break;
          }
          case 'NearbyNewMoment': {
            const payloadRecord = Index.payloadRecord(event.payload, '');
            const metadata = new NotificationMetadata();
            metadata.type = 'nearby_new_moment';
            metadata.targetId = String(payloadRecord.moment_id);
            await this.pushSmartNotification(
              'social', 'medium', '附近新动态',
              `${(payloadRecord.actor as Record<string, Object>).nickname} 在附近发布了新瞬间`,
              metadata, true);
            break;
          }
          // 契约事件登记：群消息已读数/成员角色变更由群详情/聊天页按需刷新，
          // 本端暂无精确展示，仅在类型层对齐 spec（见 WebSocketTypes.ets）。
          case 'GroupMessageReadCountUpdated':
          case 'GroupMemberRoleChanged': {
            break;
          }
          case 'SessionRevoked': {
            // 已由 WebSocketService.handleServerEvent 处理（清本地态 + 断开 + 提示），此处无需额外动作
            break;
          }
        }
      } catch (err) {
        hilog.warn(DOMAIN_NUMBER, TAG, `WS通知处理失败: ${JSON.stringify(err)}`);
      }
    };

    // 逐事件绑定（避免 '*' 里做大量无关 switch）；清单见 WS_NOTIFICATION_EVENTS
    WS_NOTIFICATION_EVENTS.forEach((t: string) => {
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
  build() {
    Stack({ alignContent: Alignment.Top }) {
      if (!this.isInitialized) {
        Column() {
          LoadingProgress()
            .width(DesignConstants.COMPONENT.AVATAR_SIZE.MEDIUM)
            .height(DesignConstants.COMPONENT.AVATAR_SIZE.MEDIUM)
            .color(this.getCurrentThemePalette().accentBackground)

          Text($r('app.string.common_loading'))
            .fontSize(DesignConstants.FONT.SIZE.LG)
            .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.LG))
            .fontColor(this.getCurrentThemePalette().textSecondary)
            .margin({ top: DesignConstants.SPACING.LG })
        }
        .width('100%')
        .height('100%')
        .justifyContent(FlexAlign.Center)
        .backgroundColor(this.getCurrentThemePalette().pageBackground)
        .expandSafeArea([SafeAreaType.SYSTEM], [SafeAreaEdge.TOP, SafeAreaEdge.BOTTOM])
        .transition(
          TransitionEffect.OPACITY.animation({ duration: DesignConstants.ANIMATION.DURATION_SHEET, curve: DesignConstants.ANIMATION.CURVE_FAST_OUT_SLOW_IN })
            .combine(TransitionEffect.move(TransitionEdge.BOTTOM).animation({ duration: DesignConstants.ANIMATION.DURATION_SHEET, curve: DesignConstants.ANIMATION.CURVE_FAST_OUT_SLOW_IN }))
        )
      } else if (this.lastError) {
        Column({ space: DesignConstants.SPACING.LG }) {
          Text('应用初始化失败')
            .fontSize(DesignConstants.FONT.SIZE.LG_PLUS)
            .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.LG_PLUS))
            .fontWeight(DesignConstants.FONT.WEIGHT.BOLD)
            .fontColor(this.getCurrentThemePalette().textPrimary)
          Text(this.lastError)
            .fontSize(DesignConstants.FONT.SIZE.MD)
            .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.MD))
            .fontColor(this.getCurrentThemePalette().textSecondary)
            .textAlign(TextAlign.Center)
            .constraintSize({ maxWidth: 280 })
          SystemButton({
            label: '重试',
            variant: 'primary',
            buttonSize: 'default',
            pill: true,
            onPress: () => {
              this.lastError = '';
              this.isInitialized = false;
              this.initializeApp();
            }
          })
        }
        .width('100%')
        .height('100%')
        .justifyContent(FlexAlign.Center)
        .backgroundColor(this.getCurrentThemePalette().pageBackground)
        .expandSafeArea([SafeAreaType.SYSTEM], [SafeAreaEdge.TOP, SafeAreaEdge.BOTTOM])
      } else {
        // Navigation 路由宿主：承载所有 AppRouter.push() 推入的页面。
        // route_map.json 已注册全部 28 个路由 + *Builder，系统据此解析 NavDestination，
        // 无需手写 .navDestination() 映射。根视图 = PhoneLayout / DesktopLayout（tab 容器）。
        // 缺少此宿主时 pushPathByName 只填栈不渲染，导致除 tab 外所有菜单点击静默失效。
        Navigation(this.navPathStack) {
          if (!useSidebarChrome(this.currentBreakpoint)) {
            PhoneLayout({
              currentTabIndex: $currentTabIndex,
              showCreateSheet: $showCreateSheet,
              showNotificationCenter: $showNotificationCenter,
              showPerformanceDetails: $showPerformanceDetails,
              isCreateTabPressed: $isCreateTabPressed,
              unreadCount: this.unreadCount.total_unread_count,
              visibleNotifications: this.visibleNotifications,
              isLoggedIn: this.isLoggedIn,
              safeAreaTop: this.safeAreaTop,
              safeAreaBottom: this.safeAreaBottom,
              isTablet: this.isTablet,
              resolvedThemeId: this.resolvedThemeId
            })
          } else {
            DesktopLayout({
              currentRoute: $currentRoute,
              showCreateSheet: $showCreateSheet,
              showNotificationCenter: $showNotificationCenter,
              showPerformanceDetails: $showPerformanceDetails,
              unreadCount: this.unreadCount.total_unread_count,
              isLoggedIn: this.isLoggedIn
            })
          }
        }
        .hideTitleBar(true)
        .mode(NavigationMode.Stack)
      }
    }
    .opacity(this.themeFadeOpacity)
  }

}