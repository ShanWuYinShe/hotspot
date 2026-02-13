/**
 * 优化后的主页面
 * 使用HarmonyOS原生能力，简化架构，AppStorage 状态管理
 */

import { PageNames } from '../../../constants/AppConstants';
import { AppRouter, LoginRouteParam } from '../../../core/router/AppRouter';
import { common } from '@kit.AbilityKit';
import { UIContext } from '@kit.ArkUI';
import { GlobalStateManager, UnreadCount, KEY_CURRENT_USER, KEY_IS_LOGGED_IN, KEY_USER_TYPE, KEY_CURRENT_LOCATION, KEY_BROWSE_ANCHOR, KEY_UNREAD_COUNT } from '../../../core/state/GlobalStateManager';
import { CoordinateConverter } from '../../../core/geo/CoordinateConverter';
import { LocationInfo } from '../../../core/location/LocationUtil';
import { PersistentStateManager } from '../../../core/state/PersistentStateManager';
import { PreferencesUtil } from '../../../core/storage/PreferencesUtil';
import { LocationUtil } from '../../../core/location/LocationUtil';
import { WebSocketService } from '../../../core/network/WebSocketService';
import { MyProfile, UpdateLocationRequest, GeoPoint } from '../../../core/types/api';
import type { ServerEvent } from '../../../core/types/WebSocketTypes';
import { PermissionManager } from '../../../core/permission/PermissionManager';
import { ApiClient } from '../../../core/api/ApiClient';
import { AppIcon } from '../../../components/SystemIcon';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { SmartNotificationService, NotificationEvent, CreateNotificationInput, NotificationMetadata, CreateNotificationOptions } from '../../../core/services/SmartNotificationService';
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

const DOMAIN_NUMBER: number = 0xFF09;
const TAG: string = 'IndexPage';

// Define storage key locally since it's not exported from NotificationRenderer
const STORAGE_KEY_VISIBLE_NOTIFICATIONS = 'visibleNotifications';

@Entry
@Component
struct Index {
  /** Navigation 路由栈 */
  @Provide('NavPathStack') navPathStack: NavPathStack = new NavPathStack();

  @StorageLink(KEY_CURRENT_USER) currentUser: MyProfile | null = null;
  @StorageLink(KEY_IS_LOGGED_IN) isLoggedIn: boolean = false;
  @StorageLink(KEY_USER_TYPE) userType: string = '';
  @StorageLink(KEY_CURRENT_LOCATION) currentLocation: LocationInfo | null = null;
  @StorageLink(KEY_UNREAD_COUNT) unreadCount: UnreadCount = { total_unread_count: 0 };
  @StorageLink(STORAGE_KEY_VISIBLE_NOTIFICATIONS) visibleNotifications: SmartNotification[] = [];

  @State currentTabIndex: number = 0;
  @State previousTabIndex: number = 0;
  @State isInitialized: boolean = false;
  @State lastError: string = '';
  @State showNotificationCenter: boolean = false;
  @State showPerformanceDetails: boolean = false;
  @State showCreateSheet: boolean = false;
  
  private stateManager = GlobalStateManager.getInstance();
  private persistentStateManager = PersistentStateManager.getInstance();
  private webSocketService = WebSocketService.getInstance();
  private smartNotificationService = SmartNotificationService.getInstance();
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

  async aboutToAppear() {
    // 初始化全局路由
    AppRouter.init(this.navPathStack);

    try {
      this.uiContext = this.getUIContext();
      this.abilityContext = this.uiContext.getHostContext() as common.UIAbilityContext;
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `获取UIContext失败: ${JSON.stringify(error)}`);
    }

    if (this.abilityContext) {
      this.i18nManager.setContext(this.abilityContext);
      await this.i18nManager.initialize();
    }

    await this.initializeApp();
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

      await PreferencesUtil.init(this.abilityContext);
      hilog.info(DOMAIN_NUMBER, TAG, '✅ 存储初始化完成');

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

      await this.persistentStateManager.initialize();
      await this.stateManager.initialize();
      await this.initializeSmartServices();

      const persistentState = this.persistentStateManager.getState();
      if (persistentState.currentUser) {
        this.stateManager.updateUser(persistentState.currentUser);
      }
      if (persistentState.currentLocation) {
        const restoredLoc = persistentState.currentLocation as LocationInfo;
        this.stateManager.updateLocation(restoredLoc);
        // 若浏览锚点尚未设置，用恢复的位置初始化
        this.initBrowseAnchorIfNeeded(restoredLoc);
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

    const converted = CoordinateConverter.convertWgs84ToGcj02(
      wgs84Location.latitude, wgs84Location.longitude
    );
    const anchor: LocationInfo = {
      latitude: converted.latitude,
      longitude: converted.longitude,
      locationName: wgs84Location.locationName
    };
    AppStorage.setOrCreate<LocationInfo | null>(KEY_BROWSE_ANCHOR, anchor);
    hilog.info(DOMAIN_NUMBER, TAG, `✅ 初始浏览锚点已设置: ${anchor.locationName ?? '未知'}`);
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
    const handle = async (event: ServerEvent) => {
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
            metadata.user = payloadRecord.user;
            const notifInput: CreateNotificationInput = {
              type: 'social',
              priority: 'medium',
              title: '新关注者',
              message: `${(payloadRecord.user as Record<string, Object>).nickname} 关注了你`,
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
            metadata.user = payloadRecord.user;
            const notifInput: CreateNotificationInput = {
              type: 'social',
              priority: 'medium',
              title: '成为好友',
              message: `你和 ${(payloadRecord.user as Record<string, Object>).nickname} 互相关注了`,
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
              message: `${(payloadRecord.user as Record<string, Object>).nickname} 赞了你的瞬间`,
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
              message: `${(payloadRecord.user as Record<string, Object>).nickname}: ${String(payloadRecord.content)}`,
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
              message: `${(payloadRecord.user as Record<string, Object>).nickname}: ${String(payloadRecord.content)}`,
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
            const metadata = new NotificationMetadata();
            metadata.type = 'group_invite';
            metadata.targetId = String(payloadRecord.group_id);
            const notifInput: CreateNotificationInput = {
              type: 'social',
              priority: 'high',
              title: '群组邀请',
              message: `${(payloadRecord.inviter as Record<string, Object>).nickname} 邀请你加入「${String(payloadRecord.group_name)}」`,
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
            const metadata = new NotificationMetadata();
            metadata.type = 'group_application_response';
            metadata.targetId = String(payloadRecord.group_id);
            const notifInput: CreateNotificationInput = {
              type: 'social',
              priority: payloadRecord.status === 'accepted' ? 'medium' : 'low',
              title: payloadRecord.status === 'accepted' ? '申请已通过' : '申请被拒绝',
              message: payloadRecord.status === 'accepted' ? `你已加入「${String(payloadRecord.group_name)}」` : `你的入群申请被拒绝`,
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
            metadata.targetId = String(payloadRecord.id);
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
      'CommentReply',
      'GroupInviteReceived',
      'GroupInviteResponse',
      'GroupApplicationReceived',
      'GroupApplicationResponse',
      'SystemAnnouncement',
    ].forEach((t: string) => {
      this.webSocketService.on(t, handle);
    });
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
      };
      this.smartNotificationService.addEventListener(NotificationEvent.CREATED, this.smartNotificationListener);

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
    Column() {
      Text() {
        SymbolSpan($r(icon))
          .fontSize(24)
          .fontColor([this.currentTabIndex === targetIndex
            ? $r('sys.color.ohos_id_color_emphasize')
            : $r('sys.color.ohos_id_color_text_secondary')])
      }

      Text(title)
        .fontColor(this.currentTabIndex === targetIndex
          ? $r('sys.color.ohos_id_color_emphasize')
          : $r('sys.color.ohos_id_color_text_secondary'))
        .fontSize(10)
        .margin({ top: 4 })
        .fontWeight(this.currentTabIndex === targetIndex ? FontWeight.Medium : FontWeight.Normal)
    }
    .width('100%')
    .height(56)
    .justifyContent(FlexAlign.Center)
    .onClick(() => {
      this.handleTabChange(targetIndex);
      if (targetIndex !== 2) {
        this.tabsController.changeIndex(targetIndex);
      }
    })
  }

  /**
   * 附近标签 Builder — 使用 Image + 系统媒体图标
   * sys.symbol.location 在部分设备不可用，改用 sys.media 图标
   */
  @Builder
  NearbyTabBuilder() {
    Column() {
      Image($r('sys.media.ohos_ic_public_location'))
        .width(24)
        .height(24)
        .fillColor(this.currentTabIndex === 0
          ? $r('sys.color.ohos_id_color_emphasize')
          : $r('sys.color.ohos_id_color_text_secondary'))

      Text('附近')
        .fontColor(this.currentTabIndex === 0
          ? $r('sys.color.ohos_id_color_emphasize')
          : $r('sys.color.ohos_id_color_text_secondary'))
        .fontSize(10)
        .margin({ top: 4 })
        .fontWeight(this.currentTabIndex === 0 ? FontWeight.Medium : FontWeight.Normal)
    }
    .width('100%')
    .height(56)
    .justifyContent(FlexAlign.Center)
    .onClick(() => {
      this.handleTabChange(0);
      this.tabsController.changeIndex(0);
    })
  }

  /** 中间「+」发布按钮 Builder */
  @Builder
  CreateTabBuilder() {
    Column() {
      Column() {
        Text() {
          SymbolSpan($r(AppIcon.ADD))
            .fontSize(24)
            .fontColor([Color.White])
        }
      }
      .width(44)
      .height(28)
      .justifyContent(FlexAlign.Center)
      .backgroundColor($r('sys.color.ohos_id_color_emphasize'))
      .borderRadius(DesignConstants.RADIUS.MD)

      Text('发布')
        .fontColor($r('sys.color.ohos_id_color_text_secondary'))
        .fontSize(10)
        .margin({ top: 4 })
    }
    .width('100%')
    .height(56)
    .justifyContent(FlexAlign.Center)
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
            .color($r('sys.color.ohos_id_color_emphasize'))
          
          Text($r('app.string.common_loading'))
            .fontSize(16)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
            .margin({ top: 16 })
        }
        .width('100%')
        .height('100%')
        .justifyContent(FlexAlign.Center)
        .backgroundColor($r('sys.color.ohos_id_color_background'))
        .transition(TransitionEffect.OPACITY.animation({ duration: 300, curve: Curve.FastOutSlowIn }))
      } else {
        Navigation(this.navPathStack) {
        Tabs({ barPosition: BarPosition.End, controller: this.tabsController }) {
          // 1. 附近（含动态/用户/群组子标签）
          TabContent() {
            NearbyPage()
          }
          .tabBar(this.NearbyTabBuilder())

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
        .scrollable(false)
        .onChange((index: number) => this.handleTabChange(index))
        .backgroundColor($r('sys.color.ohos_id_color_sub_background'))
        .bindSheet($$this.showCreateSheet, this.CreateSheetContent(), {
          height: SheetSize.FIT_CONTENT,
          dragBar: true,
          showClose: true
        })
        }
        .mode(NavigationMode.Stack)
        .hideTitleBar(true)
        .hideBackButton(true)
        .transition(TransitionEffect.OPACITY.animation({ duration: 300, curve: Curve.FastOutSlowIn }))
      }

      if (this.visibleNotifications.length > 0) {
        Column({ space: 8 }) {
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
        .hitTestBehavior(HitTestMode.Transparent)
      }

      if (this.showNotificationCenter) {
        Column() {
          NotificationCenter()
        }
        .width('100%')
        .height('100%')
        .backgroundColor($r('sys.color.ohos_id_color_background'))
        .transition(TransitionEffect.OPACITY.animation({ duration: 300, curve: Curve.EaseInOut }))
        .onClick(() => {
          this.showNotificationCenter = false;
        })
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
        .backgroundColor($r('sys.color.ohos_id_color_background'))
        .transition(TransitionEffect.OPACITY.animation({ duration: 300, curve: Curve.EaseInOut }))
        .onClick(() => {
          this.showPerformanceDetails = false;
        })
      }
    }
    .width('100%')
    .height('100%')
  }

  @Builder
  private buildUpgradeBanner() {
    Row({ space: 8 }) {
      Text('当前为临时用户，注册账号可保留所有数据')
        .fontSize(12)
        .fontColor($r('sys.color.ohos_id_color_text_primary_contrary'))
        .layoutWeight(1)
        .maxLines(1)
        .textOverflow({ overflow: TextOverflow.Ellipsis })

      Button('去注册')
        .type(ButtonType.Capsule)
        .fontSize(12)
        .height(28)
        .backgroundColor($r('sys.color.ohos_id_color_background'))
        .fontColor($r('sys.color.ohos_id_color_emphasize'))
        .onClick(() => {
          AppRouter.push(PageNames.LOGIN);
        })
    }
    .width('100%')
    .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, top: 8, bottom: 8 })
    .backgroundColor($r('sys.color.ohos_id_color_emphasize'))
  }

  @Builder
  CreateSheetContent() {
    Column({ space: 0 }) {
      Text('创建内容')
        .fontSize(DesignConstants.FONT.SIZE.XL)
        .fontWeight(FontWeight.Bold)
        .fontColor($r('sys.color.ohos_id_color_text_primary'))
        .margin({ top: 8, bottom: 20 })

      // 发布瞬间
      Row({ space: DesignConstants.SPACING.LG }) {
        Column() {
          Text() {
            SymbolSpan($r(AppIcon.CAMERA))
              .fontSize(28)
              .fontColor([$r('sys.color.ohos_id_color_emphasize')])
          }
        }
        .width(56)
        .height(56)
        .justifyContent(FlexAlign.Center)
        .backgroundColor($r('sys.color.ohos_id_color_component_normal'))
        .borderRadius(DesignConstants.RADIUS.LG)

        Column({ space: 4 }) {
          Text('发布瞬间')
            .fontSize(DesignConstants.FONT.SIZE.LG)
            .fontWeight(FontWeight.Medium)
            .fontColor($r('sys.color.ohos_id_color_text_primary'))
          Text('分享你身边的精彩时刻')
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
        }
        .alignItems(HorizontalAlign.Start)
        .layoutWeight(1)

        Text() {
          SymbolSpan($r(AppIcon.FORWARD))
            .fontSize(18)
            .fontColor([$r('sys.color.ohos_id_color_text_secondary')])
        }
      }
      .width('100%')
      .padding(DesignConstants.SPACING.LG)
      .backgroundColor($r('sys.color.ohos_id_color_background'))
      .borderRadius(DesignConstants.RADIUS.LG)
      .onClick(() => {
        this.showCreateSheet = false;
        AppRouter.push(PageNames.CREATE_MOMENT);
      })

      // 创建群组
      Row({ space: DesignConstants.SPACING.LG }) {
        Column() {
          Text() {
            SymbolSpan($r(AppIcon.GROUP))
              .fontSize(28)
              .fontColor([$r('sys.color.ohos_id_color_emphasize')])
          }
        }
        .width(56)
        .height(56)
        .justifyContent(FlexAlign.Center)
        .backgroundColor($r('sys.color.ohos_id_color_component_normal'))
        .borderRadius(DesignConstants.RADIUS.LG)

        Column({ space: 4 }) {
          Text('创建群组')
            .fontSize(DesignConstants.FONT.SIZE.LG)
            .fontWeight(FontWeight.Medium)
            .fontColor($r('sys.color.ohos_id_color_text_primary'))
          Text('创建一个附近的群组讨论')
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
        }
        .alignItems(HorizontalAlign.Start)
        .layoutWeight(1)

        Text() {
          SymbolSpan($r(AppIcon.FORWARD))
            .fontSize(18)
            .fontColor([$r('sys.color.ohos_id_color_text_secondary')])
        }
      }
      .width('100%')
      .padding(DesignConstants.SPACING.LG)
      .backgroundColor($r('sys.color.ohos_id_color_background'))
      .borderRadius(DesignConstants.RADIUS.LG)
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

  /** 通过路由名跳转页面（兼容旧调用） */
  private navigateToPage(name: string): void {
    AppRouter.push(name);
  }
}
