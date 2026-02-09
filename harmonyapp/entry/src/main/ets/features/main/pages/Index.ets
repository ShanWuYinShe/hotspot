/**
 * 优化后的主页面
 * 使用HarmonyOS原生能力，简化架构，AppStorage 状态管理
 */

import { PageNames } from '../../../constants/AppConstants';
import { common } from '@kit.AbilityKit';
import { UIContext } from '@kit.ArkUI';
import { GlobalStateManager, UnreadCount, KEY_CURRENT_USER, KEY_IS_LOGGED_IN, KEY_CURRENT_LOCATION, KEY_UNREAD_COUNT } from '../../../core/state/GlobalStateManager';
import { LocationInfo } from '../../../core/location/LocationUtil';
import { PersistentStateManager } from '../../../core/state/PersistentStateManager';
import { PreferencesUtil } from '../../../core/storage/PreferencesUtil';
import { LocationUtil } from '../../../core/location/LocationUtil';
import { WebSocketService } from '../../../core/network/WebSocketService';
import { MyProfileData, UpdateLocationRequest } from '../../../core/types/api';
import type { ServerEvent } from '../../../core/types/WebSocketTypes';
import { PermissionManager } from '../../../core/permission/PermissionManager';
import { ApiClient } from '../../../core/api/ApiClient';
import { AppIcon, AppIconMap } from '../../../components/SystemIcon';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { SmartNotificationService, NotificationEvent, CreateNotificationInput, NotificationMetadata } from '../../../core/services/SmartNotificationService';

// Define CreateNotificationOptions type locally
interface CreateNotificationOptions {
  immediate?: boolean;
  delay?: number;
  callback?: (success: boolean, notificationId?: string) => void;
}
import { CacheManager } from '../../../core/cache/CacheManager';
import { NotificationCenter } from '../../../components/NotificationCenter';
import { PerformanceIndicator } from '../../../components/PerformanceIndicator';
import { I18nManager } from '../../../core/i18n/I18nManager';
import { DesignConstants } from '../../../constants/DesignConstants';
import { GroupPage } from '../../../features/group/pages/GroupPage';
import { MessagePage } from '../../../features/message/pages/MessagePage';
import { MomentPage } from '../../../features/moment/pages/MomentPage';
import { MapPage } from '../../../features/map/pages/MapPage';
import { ProfilePage } from '../../../features/profile/pages/ProfilePage';
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
  @StorageLink(KEY_CURRENT_USER) currentUser: MyProfileData | null = null;
  @StorageLink(KEY_IS_LOGGED_IN) isLoggedIn: boolean = false;
  @StorageLink(KEY_CURRENT_LOCATION) currentLocation: LocationInfo | null = null;
  @StorageLink(KEY_UNREAD_COUNT) unreadCount: UnreadCount = { total_unread_count: 0 };
  @StorageLink(STORAGE_KEY_VISIBLE_NOTIFICATIONS) visibleNotifications: SmartNotification[] = [];

  @State currentTabIndex: number = 0;
  @State isInitialized: boolean = false;
  @State lastError: string = '';
  @State showNotificationCenter: boolean = false;
  @State showPerformanceDetails: boolean = false;
  
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

  async aboutToAppear() {
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

      await this.persistentStateManager.initialize();
      await this.stateManager.initialize();
      await this.initializeSmartServices();

      const persistentState = this.persistentStateManager.getState();
      if (persistentState.currentUser) {
        this.stateManager.updateUser(persistentState.currentUser);
      }
      if (persistentState.currentLocation) {
        this.stateManager.updateLocation(persistentState.currentLocation as LocationInfo);
      }

      try {
        const hasPermission: boolean = await PermissionManager.getInstance().checkLocationPermission();
        if (hasPermission) {
          const location = await LocationUtil.getCurrentLocation();
          if (location) {
            const locationName: string | null = await LocationUtil.getLocationName(location.latitude, location.longitude);
            const locationInfo: LocationInfo = {
              latitude: location.latitude,
              longitude: location.longitude,
              locationName: locationName || '位置不可用',
              accuracy: location.accuracy || 0,
              altitude: location.altitude || 0,
              speed: location.speed || 0,
              bearing: location.bearing || 0,
              timestamp: location.timestamp || Date.now()
            };
            this.stateManager.updateLocation(locationInfo);

            // 登录态：同步到服务端（与 Web 端保持一致，不依赖“手动点更新位置”）
            if (this.isLoggedIn) {
              try {
                const locationReq: UpdateLocationRequest = {
                  location: {
                    latitude: locationInfo.latitude,
                    longitude: locationInfo.longitude,
                    location_name: locationInfo.locationName || '位置不可用'
                  }
                };
                await ApiClient.updateLocation(locationReq);
              } catch (e) {
                hilog.warn(DOMAIN_NUMBER, TAG, `位置上传失败: ${JSON.stringify(e)}`);
              }
            }
          }
        } else {
          hilog.warn(DOMAIN_NUMBER, TAG, '位置权限未授予');
        }
      } catch (error) {
        hilog.warn(DOMAIN_NUMBER, TAG, `获取位置失败: ${JSON.stringify(error)}`);
      }

      // 定时同步位置（5分钟一次）：仅登录态启用，节省电量与网络
      if (this.isLoggedIn && this.locationSyncTimerId === 0) {
        this.locationSyncTimerId = TimerManager.setInterval(async () => {
          try {
            if (!this.isLoggedIn) return;
            const hasPermission: boolean = await PermissionManager.getInstance().checkLocationPermission();
            if (!hasPermission) return;

            const location = await LocationUtil.getCurrentLocation();
            if (!location) return;

            const locationName: string | null = await LocationUtil.getLocationName(location.latitude, location.longitude);
            const locationInfo: LocationInfo = {
              latitude: location.latitude,
              longitude: location.longitude,
              locationName: locationName || '位置不可用',
              accuracy: location.accuracy || 0,
              altitude: location.altitude || 0,
              speed: location.speed || 0,
              bearing: location.bearing || 0,
              timestamp: location.timestamp || Date.now()
            };
            this.stateManager.updateLocation(locationInfo);

            const locationReq: UpdateLocationRequest = {
              location: {
                latitude: locationInfo.latitude,
                longitude: locationInfo.longitude,
                location_name: locationInfo.locationName || '位置不可用'
              }
            };
            const resp = await ApiClient.updateLocation(locationReq);
            if (resp.code !== 0) {
              hilog.warn(DOMAIN_NUMBER, TAG, `定时位置上传失败: ${resp.message || resp.code}`);
            }
          } catch (e) {
            hilog.warn(DOMAIN_NUMBER, TAG, `定时位置同步异常: ${JSON.stringify(e)}`);
          }
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
          case 'NewFollower': {
            const p: Object = event.payload as Object;
            const payloadRecord = p as Record<string, Object>;
            const metadata = new NotificationMetadata();
            metadata.type = 'new_follower';
            metadata.user = payloadRecord.user;
            const notifInput: CreateNotificationInput = {
              type: 'social',
              priority: 'medium',
              title: payloadRecord.is_mutual ? '新好友' : '新关注者',
              message: payloadRecord.is_mutual ? `${(payloadRecord.user as Record<string, Object>).nickname} 与你互相关注了` : `${(payloadRecord.user as Record<string, Object>).nickname} 关注了你`,
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
      setInterval(() => {
        const stats = this.cacheManager.getStats();
        if (stats.hitRate < 50 && stats.totalRequests > 100) {
          hilog.warn(DOMAIN_NUMBER, TAG, `缓存命中率较低: ${stats.hitRate}%`);
        }
      }, 30000); 

      this.smartNotificationService.addEventListener(NotificationEvent.CREATED, (event, notification) => {
        hilog.info(DOMAIN_NUMBER, TAG, `收到通知: ${notification.title}`);
        NotificationRenderer.getInstance().showNotification(notification);
      });

    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `智能服务初始化失败: ${JSON.stringify(error)}`);
    }
  }

  private handleTabChange(index: number) {
    this.currentTabIndex = index;
    if (!this.isLoggedIn && index !== 0 && index !== 3) { 
       // Logic for unauthorized access
    }
  }

  @Builder
  TabBuilderIcon(title: string, targetIndex: number, icon: string) {
    Column() {
      // 使用 SymbolGlyph 替代 Text 渲染图标
      SymbolGlyph($r(`sys.symbol.${icon}`))
        .fontSize(24)
        .fontColor([this.currentTabIndex === targetIndex ? DesignConstants.COLOR.BRAND_PRIMARY : DesignConstants.COLOR.TEXT_SECONDARY_LIGHT])
        .renderingStrategy(SymbolRenderingStrategy.SINGLE)
      
      Text(title)
        .fontColor(this.currentTabIndex === targetIndex ? DesignConstants.COLOR.BRAND_PRIMARY : DesignConstants.COLOR.TEXT_SECONDARY_LIGHT)
        .fontSize(10)
        .margin({ top: 4 })
        .fontWeight(this.currentTabIndex === targetIndex ? 500 : 400)
    }
    .width('100%')
    .height(56)
    .justifyContent(FlexAlign.Center)
    .onClick(() => {
      this.currentTabIndex = targetIndex;
      this.tabsController.changeIndex(targetIndex);
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
        .backgroundColor(DesignConstants.COLOR.PRIMARY_BG_LIGHT)
      } else {
        Tabs({ barPosition: BarPosition.End, controller: this.tabsController }) {
          // 1. 群组/发现
          TabContent() {
            // 游客允许浏览附近群组；加入/创建等操作在页面内做登录引导
            GroupPage()
          }
          .tabBar(this.TabBuilderIcon('群组', 0, AppIconMap[AppIcon.GROUP]))

          // 2. 消息
          TabContent() {
            if (this.isLoggedIn) {
              MessagePage()
            } else {
              this.buildWelcomeView()
            }
          }
          .tabBar(this.TabBuilderIcon('消息', 1, AppIconMap[AppIcon.MESSAGE]))

          // 3. 瞬间
          TabContent() {
            // 游客允许浏览附近瞬间；发布/点赞等操作在页面内做登录引导
            MomentPage()
          }
          .tabBar(this.TabBuilderIcon('瞬间', 2, AppIconMap[AppIcon.CAMERA]))

          // 4. 地图
          TabContent() {
            MapPage()
          }
          .tabBar(this.TabBuilderIcon('地图', 3, AppIconMap[AppIcon.MAP]))

          // 5. 我的
          TabContent() {
            ProfilePage()
          }
          .tabBar(this.TabBuilderIcon('我的', 4, AppIconMap[AppIcon.PERSON]))
        }
        .scrollable(false)
        .onChange((index: number) => this.handleTabChange(index))
        .backgroundColor(DesignConstants.COLOR.SECONDARY_BG_LIGHT)
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
            .transition(TransitionEffect.OPACITY.animation({ duration: 300 }).combine(TransitionEffect.move(TransitionEdge.TOP)))
          }, (item: SmartNotification) => item.id)
        }
        .width('100%')
        .padding({ top: 12 + 36, left: 16, right: 16 })
        .hitTestBehavior(HitTestMode.Transparent)
      }

      if (this.showNotificationCenter) {
        Column() {
          NotificationCenter()
        }
        .width('100%')
        .height('100%')
        .backgroundColor(DesignConstants.COLOR.PRIMARY_BG_LIGHT)
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
        .backgroundColor(DesignConstants.COLOR.PRIMARY_BG_LIGHT)
        .onClick(() => {
          this.showPerformanceDetails = false;
        })
      }
    }
    .width('100%')
    .height('100%')
  }

  @Builder
  private buildWelcomeView() {
    Column({ space: 24 }) {
      Text($r('app.string.module_desc'))
        .fontSize(28)
        .fontWeight(FontWeight.Bold)
        .fontColor(DesignConstants.COLOR.TEXT_PRIMARY_LIGHT)

      Text($r('app.string.app_description'))
        .fontSize(16)
        .fontColor(DesignConstants.COLOR.TEXT_SECONDARY_LIGHT)

      Button($r('app.string.auth_start_experience'))
        .type(ButtonType.Capsule)
        .backgroundColor(DesignConstants.COLOR.BRAND_PRIMARY)
        .fontColor(Color.White)
        .width(200)
        .height(44)
        .onClick(async () => {
          try {
            const routerInstance = this.getUIContext().getRouter();
            await routerInstance.pushUrl({ url: `features/auth/pages/${PageNames.LOGIN}` });
          } catch (error) {
            hilog.error(DOMAIN_NUMBER, TAG, `跳转登录页面失败: ${error}`);
          }
        })
    }
    .width('100%')
    .height('100%')
    .justifyContent(FlexAlign.Center)
    .padding(24)
    .backgroundColor(DesignConstants.COLOR.SECONDARY_BG_LIGHT)
  }
}
