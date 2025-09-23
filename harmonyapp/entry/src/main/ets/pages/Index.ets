/**
 * 优化后的主页面
 * 使用HarmonyOS原生能力，简化架构
 */

import { PageNames } from '../constants/AppConstants';
import { common } from '@kit.AbilityKit';
import { UIContext } from '@kit.ArkUI';
import { GlobalStateManager, GlobalState } from '../common/state/GlobalStateManager';
import { LocationInfo } from '../common/location/LocationUtil';
import { PersistentStateManager } from '../common/state/PersistentStateManager';
import { PreferencesUtil } from '../common/storage/PreferencesUtil';
import { LocationUtil } from '../common/location/LocationUtil';
import { WebSocketService } from '../common/websocket/WebSocketService';
import { UserResponse } from '../common/types/BackendApiTypes';
import { PermissionManager } from '../common/permission/PermissionManager';
import { NotificationComponent } from '../components/NotificationComponent';
import { NavIcon, AppIcon } from '../components/SystemIcon';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { SmartNotificationService } from '../common/services/SmartNotificationService';
import { PerformanceOptimizer } from '../common/performance/PerformanceOptimizer';
import { NotificationCenter } from '../components/NotificationCenter';
import { PerformanceIndicator } from '../components/PerformanceIndicator';
import { SystemToast } from '../components/SystemToast';

const DOMAIN_NUMBER: number = 0xFF09;
const TAG: string = 'IndexPage';

interface TabItem {
  name: string;
  icon: AppIcon;
  page: string;
}

@Entry
@Component
struct Index {
  @State globalState: GlobalState = GlobalStateManager.getInstance().getState();
  @State currentTabIndex: number = 0;
  @State isInitialized: boolean = false;
  @State lastError: string = '';
  @State wsConnectionStatus: string = '未连接';
  @State showNotificationCenter: boolean = false;
  @State showPerformanceDetails: boolean = false;
  private stateManager = GlobalStateManager.getInstance();
  private persistentStateManager = PersistentStateManager.getInstance();
  private webSocketService = WebSocketService.getInstance();
  private smartNotificationService = SmartNotificationService.getInstance();
  private performanceOptimizer = PerformanceOptimizer.getInstance();
  private uiContext: UIContext | null = null;
  private abilityContext: common.UIAbilityContext | null = null;
  // 标签配置
  private readonly tabs: TabItem[] = [
    { name: '群组', icon: AppIcon.GROUP, page: PageNames.GROUP },
    { name: '消息', icon: AppIcon.MESSAGE, page: PageNames.MESSAGE },
    { name: '瞬间', icon: AppIcon.CAMERA, page: PageNames.MOMENT },
    { name: '地图', icon: AppIcon.MAP, page: PageNames.MAP },
    { name: '个人', icon: AppIcon.PERSON, page: PageNames.PROFILE }
  ];

  async aboutToAppear() {
    // 初始化UIContext - 使用推荐的UIContext方式 (API Version 10+)
    try {
      this.uiContext = this.getUIContext();
      // 通过UIContext获取UIAbilityContext
      this.abilityContext = this.uiContext.getHostContext() as common.UIAbilityContext;
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `获取UIContext失败: ${JSON.stringify(error)}`);
    }

    // 订阅状态变化
    this.stateManager.subscribe('index_page', (state: GlobalState) => {
      this.globalState = state;
    });

    await this.initializeApp();
  }

  aboutToDisappear() {
    // 取消订阅
    this.stateManager.unsubscribe('index_page');
  }

  /**
   * 应用初始化 - 简化版本
   */
  private async initializeApp() {
    try {
      hilog.info(DOMAIN_NUMBER, TAG, '🚀 开始应用初始化');

      // 检查UIAbilityContext是否已初始化
      if (!this.abilityContext) {
        const errorMsg = 'UIAbilityContext未初始化';
        hilog.error(DOMAIN_NUMBER, TAG, errorMsg);
        this.lastError = errorMsg;
        return;
      }

      // 1. 初始化存储 - 使用UIContext获取的UIAbilityContext
      await PreferencesUtil.init(this.abilityContext);
      hilog.info(DOMAIN_NUMBER, TAG, '✅ 存储初始化完成');

      // 2. 初始化持久化状态管理器
      await this.persistentStateManager.initialize();
      hilog.info(DOMAIN_NUMBER, TAG, '✅ 持久化状态管理器初始化完成');

      // 3. 初始化智能服务
      await this.initializeSmartServices();
      hilog.info(DOMAIN_NUMBER, TAG, '✅ 智能服务初始化完成');

      // 初始化导航工具
      // NavigationUtil 已删除，无需初始化
      hilog.info(DOMAIN_NUMBER, TAG, '✅ 导航工具初始化完成');


      // 2. 同步状态到旧的状态管理器（向后兼容）
      const persistentState = this.persistentStateManager.getState();
      if (persistentState.currentUser) {
        this.stateManager.updateUser(persistentState.currentUser);
      }
      if (persistentState.currentLocation) {
        this.stateManager.updateLocation(persistentState.currentLocation as LocationInfo);
      }

      // 3. 请求位置权限并获取位置信息
      try {
        const hasPermission: boolean = await PermissionManager.getInstance().checkLocationPermission();
        if (hasPermission) {
          const location = await LocationUtil.getCurrentLocation();
          if (location) {
            // 获取位置名称
            const locationName: string | null = await LocationUtil.getLocationName(location.latitude, location.longitude);
            const locationInfo: LocationInfo = {
              latitude: location.latitude,
              longitude: location.longitude,
              locationName: locationName || '未知位置',
              accuracy: location.accuracy || 0,
              altitude: location.altitude || 0,
              speed: location.speed || 0,
              bearing: location.bearing || 0,
              timestamp: location.timestamp || Date.now()
            };
            this.stateManager.updateLocation(locationInfo);
          }
        } else {
          hilog.warn(DOMAIN_NUMBER, TAG, '位置权限未授予');
        }
      } catch (error) {
        hilog.warn(DOMAIN_NUMBER, TAG, `获取位置失败: ${JSON.stringify(error)}`);
      }

      // 初始化WebSocket连接
      await this.initializeWebSocket();

      // 加载调试信息

      this.isInitialized = true;
      hilog.info(DOMAIN_NUMBER, TAG, '✅ 应用初始化完成');
    } catch (error) {
      const errorMsg = `应用初始化失败: ${error}`;
      hilog.error(DOMAIN_NUMBER, TAG, errorMsg);
      this.lastError = errorMsg;
      this.isInitialized = true; // 即使失败也要显示界面
    }
  }

  /**
   * 获取访问令牌
   */
  private async getAccessToken(): Promise<string | null> {
    try {
      return await PreferencesUtil.get('access_token', '');
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `获取访问令牌失败: ${error}`);
      return null;
    }
  }

  /**
   * 初始化WebSocket连接
   */
  private async initializeWebSocket() {
    try {
      this.wsConnectionStatus = '连接中...';
      hilog.info(DOMAIN_NUMBER, TAG, '🔌 初始化WebSocket连接');

      // 如果用户已登录，尝试连接WebSocket
      if (this.globalState.isLoggedIn && this.globalState.currentUser) {
        // 假设我们有一个方法来获取当前的访问令牌
        const token = await this.getAccessToken();
        if (token) {
          await this.webSocketService.connect(token);
          this.wsConnectionStatus = '已连接';
        } else {
          this.wsConnectionStatus = '无令牌';
        }
      } else {
        this.wsConnectionStatus = '未登录';
      }
    } catch (error) {
      this.wsConnectionStatus = '连接失败';
      hilog.error(DOMAIN_NUMBER, TAG, `WebSocket初始化失败: ${error}`);
    }
  }


  /**
   * 切换标签页
   */
  private async switchTab(index: number) {
    if (this.currentTabIndex === index) {
      return;
    }

    // 检查登录状态
    if (!this.globalState.isLoggedIn && index !== 0) {
      await this.getUIContext().getRouter().pushUrl({ url: `pages/${PageNames.LOGIN}` });
      return;
    }

    this.currentTabIndex = index;
    // 标签页切换不需要路由跳转，直接更新状态即可
  }

  build() {
    Stack() {
      Column() {
        if (!this.isInitialized) {
          // 加载状态
          this.buildSystemLoadingView();
        } else if (!this.globalState.isLoggedIn) {
          // 未登录状态
          this.buildWelcomeView();
        } else {
          // 已登录状态
          this.buildMainContent();
        }

        // 底部导航栏
        this.buildTabBar();
      }
      .width('100%')
      .height('100%')
      .backgroundColor($r('sys.color.ohos_id_color_sub_background'))

      // 通知中心弹窗
      if (this.showNotificationCenter) {
        Column() {
          NotificationCenter()
        }
        .width('100%')
        .height('100%')
        .backgroundColor($r('sys.color.ohos_id_color_background'))
        .onClick(() => {
          this.showNotificationCenter = false;
        })
      }

      // 性能详情弹窗
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
        .onClick(() => {
          this.showPerformanceDetails = false;
        })
      }
    }
    .width('100%')
    .height('100%')
  }

  @Builder
  private buildSystemLoadingView() {
    Column() {
      LoadingProgress()
        .width(48)
        .height(48)
        .color($r('sys.color.ohos_id_color_emphasize'))

      Text('正在初始化...')
        .fontSize(16)
        .fontColor($r('sys.color.ohos_id_color_text_secondary'))
        .margin({ top: 16 })
    }
    .width('100%')
    .layoutWeight(1)
    .justifyContent(FlexAlign.Center)
  }

  @Builder
  private buildWelcomeView() {
    Column({ space: 24 }) {
      Text('欢迎使用 GeoTalk')
        .fontSize(28)
        .fontWeight(FontWeight.Bold)
        .fontColor($r('sys.color.ohos_id_color_text_primary'))

      Text('基于地理位置的社交应用')
        .fontSize(16)
        .fontColor($r('sys.color.ohos_id_color_text_secondary'))

      Button('开始使用')
        .type(ButtonType.Capsule)
        .backgroundColor($r('sys.color.ohos_id_color_emphasize'))
        .fontColor(Color.White)
        .width(200)
        .height(44)
        .onClick(async () => {
          try {
            const routerInstance = this.getUIContext().getRouter();
            await routerInstance.pushUrl({ url: `pages/${PageNames.LOGIN}` });
          } catch (error) {
            hilog.error(DOMAIN_NUMBER, TAG, `跳转登录页面失败: ${error}`);
          }
        })
    }
    .width('100%')
    .layoutWeight(1)
    .justifyContent(FlexAlign.Center)
    .padding(24)
  }

  @Builder
  private buildMainContent() {
    Column() {
      // 状态栏
      Row() {
        Text(`欢迎回来，${this.globalState.currentUser?.nickname}`)
          .fontSize(16)
          .fontColor($r('sys.color.ohos_id_color_text_primary'))
          .layoutWeight(1)

        // 搜索按钮
        Button('🔍')
          .width(36)
          .height(36)
          .fontSize(16)
          .backgroundColor(Color.Transparent)
          .fontColor($r('sys.color.ohos_id_color_text_primary'))
          .margin({ right: 8 })
          .onClick(async () => {
            await this.getUIContext().getRouter().pushUrl({
              url: `pages/${PageNames.SEARCH}`
            });
          })

        // 通知组件
        NotificationComponent()
          .margin({ right: 8 })

        if (this.globalState.currentLocation) {
          Text(`📍 ${this.globalState.currentLocation.locationName || '南天门'}`)
            .fontSize(12)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
        }
      }
      .width('100%')
      .padding({
        left: 16,
        right: 16,
        top: 12,
        bottom: 12
      })
      .backgroundColor($r('sys.color.ohos_id_color_background'))

      // 主要内容区域
      Scroll() {
        Column({ space: 16 }) {
          // 性能监控指示器
          Row() {
            Text('系统状态')
              .fontSize(16)
              .fontWeight(FontWeight.Medium)
              .fontColor($r('sys.color.ohos_id_color_text_primary'))
              .layoutWeight(1)

            PerformanceIndicator({
              compact: true,
              showDetails: this.showPerformanceDetails
            })
            .onClick(() => this.togglePerformanceDetails())
          }
          .width('100%')
          .padding({ left: 16, right: 16 })


          // 快捷功能区域
          Column({ space: 12 }) {
            Text('快捷功能')
              .fontSize(18)
              .fontWeight(FontWeight.Medium)
              .fontColor($r('sys.color.ohos_id_color_text_primary'))
              .alignSelf(ItemAlign.Start)
              .margin({ left: 16 })

            Grid() {
              GridItem() {
                this.buildQuickActionCard(
                  '通知中心',
                  AppIcon.NOTIFICATION,
                  () => this.toggleNotificationCenter()
                )
              }

              GridItem() {
                this.buildQuickActionCard(
                  '性能监控',
                  AppIcon.CHART,
                  () => this.togglePerformanceDetails()
                )
              }

              GridItem() {
                this.buildQuickActionCard(
                  '高级设置',
                  AppIcon.SETTINGS,
                  () => {
                    // 跳转到高级设置页面
                    SystemToast.showInfo('跳转到高级设置', this.getUIContext());
                  }
                )
              }

              GridItem() {
                this.buildQuickActionCard(
                  '调试面板',
                  AppIcon.DEBUG,
                  () => {
                    // 跳转到调试面板
                    this.getUIContext().getRouter().pushUrl({
                      url: 'pages/DebugPanel'
                    });
                  }
                )
              }

              GridItem() {
                this.buildQuickActionCard(
                  '服务状态',
                  AppIcon.CHART,
                  () => {
                    // 跳转到服务状态页面
                    this.getUIContext().getRouter().pushUrl({
                      url: 'pages/ServiceStatusPage'
                    });
                  }
                )
              }

              GridItem() {
                this.buildQuickActionCard(
                  '推荐中心',
                  AppIcon.RECOMMEND,
                  () => {
                    // 跳转到推荐页面
                    this.getUIContext().getRouter().pushUrl({
                      url: 'pages/RecommendationPage'
                    });
                  }
                )
              }
            }
            .columnsTemplate('1fr 1fr 1fr')
            .rowsGap(12)
            .columnsGap(12)
            .width('calc(100% - 32vp)')
            .margin({ left: 16, right: 16 })
          }
          .width('100%')

          // 当前标签页内容
          Column({ space: 16 }) {
            Text(`${this.tabs[this.currentTabIndex].name}页面`)
              .fontSize(18)
              .fontWeight(FontWeight.Medium)
              .fontColor($r('sys.color.ohos_id_color_text_primary'))
              .alignSelf(ItemAlign.Start)
              .margin({ left: 16 })

            Button(`进入${this.tabs[this.currentTabIndex].name}`)
              .fontSize(16)
              .backgroundColor($r('sys.color.ohos_id_color_emphasize'))
              .width('calc(100% - 32vp)')
              .margin({ left: 16, right: 16 })
              .onClick(async () => {
                await this.switchTab(this.currentTabIndex);
              })
          }
          .width('100%')
        }
        .padding({ bottom: 16 })
      }
      .layoutWeight(1)
      .scrollBar(BarState.Auto)
    }
    .layoutWeight(1)
  }

  @Builder
  private buildTabBar() {
    Row() {
      ForEach(this.tabs, (tab: TabItem, index: number) => {
        NavIcon({
          icon: tab.icon,
          onNavClick: () => {
            this.switchTab(index);
          }
        })
        .layoutWeight(1)
        .padding({ top: 8, bottom: 8 })
      }, (tab: TabItem, index: number) => `${tab.name}_${index}`)
    }
    .width('100%')
    .height(64)
    .backgroundColor($r('sys.color.ohos_id_color_background'))
    .border({
      width: { top: 1 },
      color: $r('sys.color.ohos_id_color_list_separator')
    })
  }

  /**
   * 初始化智能服务
   */
  private async initializeSmartServices(): Promise<void> {
    try {

      // 设置性能告警监听 - 使用轮询方式检查告警
      setInterval(() => {
        const alerts = this.performanceOptimizer.getPerformanceAlerts();
        if (alerts.length > 0) {
          const latestAlert = alerts[alerts.length - 1];
          hilog.warn(DOMAIN_NUMBER, TAG, `性能告警: ${latestAlert.message}`);
        }
      }, 5000); // 每5秒检查一次

      // 设置通知监听
      this.smartNotificationService.addListener((notification) => {
        hilog.info(DOMAIN_NUMBER, TAG, `收到通知: ${notification.title}`);
      });


    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `智能服务初始化失败: ${JSON.stringify(error)}`);
    }
  }


  /**
   * 切换通知中心显示
   */
  private toggleNotificationCenter(): void {
    this.showNotificationCenter = !this.showNotificationCenter;
  }

  /**
   * 切换性能详情显示
   */
  private togglePerformanceDetails(): void {
    this.showPerformanceDetails = !this.showPerformanceDetails;
  }

  /**
   * 构建快捷操作卡片
   */
  @Builder
  private buildQuickActionCard(title: string, icon: AppIcon, onClick: () => void) {
    Column({ space: 8 }) {
              Text(icon)
                .fontSize(24)
                .fontColor($r('sys.color.ohos_id_color_emphasize'))

      Text(title)
        .fontSize(14)
        .fontColor($r('sys.color.ohos_id_color_text_primary'))
        .textAlign(TextAlign.Center)
    }
    .width('100%')
    .height(80)
    .justifyContent(FlexAlign.Center)
    .backgroundColor($r('sys.color.ohos_id_color_background'))
    .borderRadius(12)
    .shadow({
      radius: 4,
      color: $r('sys.color.ohos_id_color_foreground_contrary'),
      offsetX: 0,
      offsetY: 2
    })
    .onClick(onClick)
    // 无障碍属性
    .accessibilityGroup(true)
    .accessibilityText(`${title}功能`)
    .accessibilityDescription(`点击进入${title}`)
  }

}
