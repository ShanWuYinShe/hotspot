/**
 * 深度集成HarmonyOS原生能力的应用入口
 * 利用HarmonyOS完整的系统服务和生命周期管理
 */

import {
  AbilityConstant,
  ConfigurationConstant,
  UIAbility,
  Want,
  Configuration,
  bundleManager,
  common
} from '@kit.AbilityKit';
import { window, display } from '@kit.ArkUI';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { notificationManager } from '@kit.NotificationKit';
import { deviceInfo, BusinessError } from '@kit.BasicServicesKit';
import { ServiceRegistry } from '../core/services/ServiceRegistry';
import { RdbManager } from '../core/database/RdbManager';
import { AnchorHistoryUtil } from '../core/utils/AnchorHistoryUtil';
import { DesignConstants } from '../constants/DesignConstants';
import { SystemSurfaceStyle, SystemUiThemeStore } from '../core/utils/SystemUiThemeStore';
import { KEY_IS_LOGGED_IN } from '../core/state/GlobalStateManager';
import { WebSocketService } from '../core/network/WebSocketService';
import { PreferencesUtil } from '../core/storage/PreferencesUtil';
import { StorageKeys } from '../constants/AppConstants';
import { ApiClient } from '../core/api/ApiClient';
import { AppThemeManager } from '../core/theme/AppThemeManager';

const DOMAIN_NUMBER: number = 0xFF00;
const TAG: string = 'GeoTalk';

export default class EntryAbility extends UIAbility {
  private windowStage: window.WindowStage | null = null;
  private mainWindow: window.Window | null = null;
  private avoidAreaCallback?: (options: window.AvoidAreaOptions) => void;
  private cachedDensityScale: number = 0;
  private surfaceStyleUnsubscribe?: () => void;
  private currentSurfaceStyle: SystemSurfaceStyle = SystemUiThemeStore.getStyle();
  private isDarkMode: boolean = false;
  private serviceRegistry = ServiceRegistry.getInstance();
  private rdbManager = RdbManager.getInstance();

  onCreate(want: Want, launchParam: AbilityConstant.LaunchParam): void {
    hilog.info(DOMAIN_NUMBER, TAG, 'GeoTalk应用启动 - 启动原因: %{public}s',
      JSON.stringify(launchParam.launchReason));

    // 初始化应用级配置
    this.initializeAppConfiguration();

    // 设置应用主题跟随系统
    this.setupThemeConfiguration();

    // 初始化通知管理
    this.initializeNotifications();

    // 获取设备信息
    this.logDeviceInfo();
    
    // 初始化数据库
    this.rdbManager.init(this.context);

    // 初始化锚点历史工具
    AnchorHistoryUtil.init(this.context);

    // 初始化智能服务
    this.initializeSmartServices();
  }

  onDestroy(): void {
    hilog.info(DOMAIN_NUMBER, TAG, 'GeoTalk应用销毁');
    this.cleanup();
  }

  onWindowStageCreate(windowStage: window.WindowStage): void {
    hilog.info(DOMAIN_NUMBER, TAG, 'WindowStage创建');
    this.windowStage = windowStage;

    // 配置窗口属性
    this.configureWindow(windowStage);

    // 加载主页面
    windowStage.loadContent('features/main/pages/Index')
      .then(() => {
        hilog.info(DOMAIN_NUMBER, TAG, '主页面加载成功');
        // 页面加载完成后的初始化
        this.onPageLoaded();
      })
      .catch((err: Error) => {
        hilog.error(DOMAIN_NUMBER, TAG, '主页面加载失败: %{public}s', err.message);
      });
  }

  onWindowStageDestroy(): void {
    hilog.info(DOMAIN_NUMBER, TAG, 'WindowStage销毁');
    this.detachAvoidAreaListener();
    this.surfaceStyleUnsubscribe?.();
    this.surfaceStyleUnsubscribe = undefined;
    this.mainWindow = null;
    this.cachedDensityScale = 0;
    this.windowStage = null;
  }

  onForeground(): void {
    hilog.info(DOMAIN_NUMBER, TAG, '应用切换到前台');
    // 应用回到前台时的处理
    this.handleAppForeground();
  }

  onBackground(): void {
    hilog.info(DOMAIN_NUMBER, TAG, '应用切换到后台');
    // 应用进入后台时的处理
    this.handleAppBackground();
  }

  onConfigurationUpdate(newConfig: Configuration): void {
    hilog.info(DOMAIN_NUMBER, TAG, '系统配置更新: %{public}s', JSON.stringify(newConfig));
    // 处理系统配置变化（如语言、主题等）
    this.handleConfigurationChange(newConfig);
  }

  onNewWant(want: Want, launchParam: AbilityConstant.LaunchParam): void {
    hilog.info(DOMAIN_NUMBER, TAG, '接收到新的Want请求: %{public}s', JSON.stringify(want));
    // 处理新的Intent请求（如分享、深度链接等）
    this.handleNewWant(want);
  }

  /**
   * 初始化应用配置
   */
  private async initializeAppConfiguration(): Promise<void> {
    try {
      // 获取当前应用的Bundle信息
      const bundleInfo = await bundleManager.getBundleInfoForSelf(
        bundleManager.BundleFlag.GET_BUNDLE_INFO_WITH_APPLICATION
      );

      // 提取关键信息
      const appName = bundleInfo.name; // 应用包名
      const versionName = bundleInfo.versionName; // 版本名称
      const versionCode = bundleInfo.versionCode; // 版本号

      // 打印日志
      hilog.info(DOMAIN_NUMBER, TAG, '当前应用信息: %{public}s', JSON.stringify({
        name: appName,
        bundleName: appName, // 包名和应用名通常一致
        versionName: versionName,
        versionCode: versionCode
      }));

    } catch (error) {
      // 错误处理
      const businessError = error as BusinessError;
      hilog.error(DOMAIN_NUMBER, TAG, '初始化应用配置失败: %{public}s', businessError.message);
      hilog.error(DOMAIN_NUMBER, TAG, '错误码: %{public}d', businessError.code);
    }
  }

  /**
   * 设置主题配置
   */
  private setupThemeConfiguration(): void {
    try {
      const appContext = this.context.getApplicationContext();

      // 设置颜色模式跟随系统（使用 ColorMode.COLOR_MODE_NOT_SET）
      appContext.setColorMode(ConfigurationConstant.ColorMode.COLOR_MODE_NOT_SET);

      // 监听系统配置变化
      // 注意：HarmonyOS的应用上下文事件监听可能需要不同的实现方式
      hilog.info(DOMAIN_NUMBER, TAG, '主题配置设置完成');

    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, '设置主题配置失败: %{public}s', JSON.stringify(error));
    }
  }

  /**
   * 初始化通知管理
   */
  private async initializeNotifications(): Promise<void> {
    try {
      // 请求通知权限 - 使用最新API
      await notificationManager.isNotificationEnabled().then(async (enabled) => {
        if (!enabled) {
          // 引导用户到设置页面开启通知
          hilog.info(DOMAIN_NUMBER, TAG, '通知权限未开启，请在设置中开启');
        }
      });

      // 创建通知渠道
      await notificationManager.addSlot(notificationManager.SlotType.SOCIAL_COMMUNICATION);

      hilog.info(DOMAIN_NUMBER, TAG, '通知系统初始化成功');
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, '通知系统初始化失败: %{public}s', JSON.stringify(error));
    }
  }

  /**
   * 获取并记录设备信息
   */
  private async logDeviceInfo(): Promise<void> {
    try {
      const device = deviceInfo.deviceType;
      const osVersion = deviceInfo.osFullName;
      const brand = deviceInfo.brand;
      const model = deviceInfo.productModel;

      hilog.info(DOMAIN_NUMBER, TAG,
        '设备信息 - 类型: %{public}s, 系统: %{public}s, 品牌: %{public}s, 型号: %{public}s',
        device, osVersion, brand, model);

    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, '获取设备信息失败: %{public}s', JSON.stringify(error));
    }
  }

  /**
   * 配置窗口属性
   */
  private async configureWindow(windowStage: window.WindowStage): Promise<void> {
    try {
      const mainWindow = await windowStage.getMainWindow();
      this.mainWindow = mainWindow;
      this.cachedDensityScale = 0;
      this.isDarkMode = this.context.config.colorMode === ConfigurationConstant.ColorMode.COLOR_MODE_DARK;
      AppThemeManager.onSystemColorModeChange(this.isDarkMode);

      // 全屏布局：内容延伸到屏幕边缘
      // 配合 expandSafeArea 让页面可延伸到状态栏区域
      // 可交互组件通过设置 safeAreaInsets 确保不被遮挡
      await mainWindow.setWindowLayoutFullScreen(true);

      this.updateSafeAreaInsets(mainWindow, 'init');
      this.registerAvoidAreaListener(mainWindow);
      this.initializeSystemSurfaceSync(mainWindow);

      // 隐藏系统状态栏和导航栏（或设置为透明）
      await mainWindow.setWindowSystemBarEnable(['status', 'navigation']);

      // 设置状态栏和导航栏为透明
      await this.applySystemBarStyle(mainWindow, this.isDarkMode, this.currentSurfaceStyle);

      // 设置窗口方向
      await mainWindow.setPreferredOrientation(window.Orientation.AUTO_ROTATION);

      hilog.info(DOMAIN_NUMBER, TAG,
        '窗口配置完成（全屏布局 + 透明系统栏），safeAreaTop=%{public}f, safeAreaBottom=%{public}f',
        AppStorage.get<number>('safeAreaTop') ?? 0,
        AppStorage.get<number>('safeAreaBottom') ?? 0);
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, '窗口配置失败: %{public}s', JSON.stringify(error));
    }
  }

  /**
   * 更新 Safe Area 高度（vp）供前端使用
   */
  private updateSafeAreaInsets(mainWindow: window.Window, source: string = 'manual'): void {
    try {
      const densityScale = this.getDensityScale();
      const systemArea = mainWindow.getWindowAvoidArea(window.AvoidAreaType.TYPE_SYSTEM);
      const cutoutArea = mainWindow.getWindowAvoidArea(window.AvoidAreaType.TYPE_CUTOUT);
      const navIndicatorArea = mainWindow.getWindowAvoidArea(window.AvoidAreaType.TYPE_NAVIGATION_INDICATOR);

      const safeAreaTopPx = Math.max(systemArea.topRect.height, cutoutArea.topRect.height);
      const bottomCandidates = [
        systemArea.bottomRect.height,
        navIndicatorArea.bottomRect.height
      ].filter(value => value && value > 0);
      const safeAreaBottomPx = bottomCandidates.length > 0 ? Math.max(...bottomCandidates) : 0;

      const safeAreaTop = this.pxToVp(safeAreaTopPx, densityScale);
      const safeAreaBottom = this.pxToVp(safeAreaBottomPx, densityScale);

      const previousTop = AppStorage.get<number>('safeAreaTop') ?? 0;
      const previousBottom = AppStorage.get<number>('safeAreaBottom') ?? 0;

      if (previousTop !== safeAreaTop) {
        AppStorage.setOrCreate('safeAreaTop', safeAreaTop);
      }
      if (previousBottom !== safeAreaBottom) {
        AppStorage.setOrCreate('safeAreaBottom', safeAreaBottom);
      }

      if (previousTop !== safeAreaTop || previousBottom !== safeAreaBottom) {
        hilog.info(DOMAIN_NUMBER, TAG,
          'SafeArea更新(%{public}s): top=%{public}f, bottom=%{public}f',
          source, safeAreaTop, safeAreaBottom);
      }
    } catch (error) {
      hilog.warn(DOMAIN_NUMBER, TAG, '更新 SafeArea 失败: %{public}s', JSON.stringify(error));
      AppStorage.setOrCreate('safeAreaTop', 0);
      AppStorage.setOrCreate('safeAreaBottom', 0);
    }
  }

  /**
   * 获取当前设备密度倍数
   */
  private getDensityScale(): number {
    if (this.cachedDensityScale > 0) {
      return this.cachedDensityScale;
    }
    try {
      const displayInfo = display.getDefaultDisplaySync();
      const scale = displayInfo.densityDPI / 160;
      this.cachedDensityScale = scale > 0 ? scale : 1;
      return this.cachedDensityScale;
    } catch (error) {
      hilog.warn(DOMAIN_NUMBER, TAG, '获取设备密度失败，使用默认值: %{public}s', JSON.stringify(error));
      this.cachedDensityScale = 1;
      return this.cachedDensityScale;
    }
  }

  /**
   * 将像素转换为 vp（基于屏幕密度）
   */
  private pxToVp(px: number, densityScale?: number): number {
    const scale = densityScale && densityScale > 0 ? densityScale : this.getDensityScale();
    if (px <= 0) {
      return 0;
    }
    return Number((px / scale).toFixed(2));
  }

  private getWindowBaseBackgroundColor(isDark: boolean): string {
    return AppThemeManager.getCurrentSurfaceStyle().baseColor;
  }


  private registerAvoidAreaListener(mainWindow: window.Window): void {
    this.detachAvoidAreaListener();
    const callback = (_data?: window.AvoidAreaOptions) => {
      this.updateSafeAreaInsets(mainWindow, 'event');
    };
    try {
      mainWindow.on('avoidAreaChange', callback);
      this.avoidAreaCallback = callback;
      hilog.info(DOMAIN_NUMBER, TAG, '已监听 avoidAreaChange 事件');
    } catch (error) {
      hilog.warn(DOMAIN_NUMBER, TAG, '注册 avoidAreaChange 失败: %{public}s', JSON.stringify(error));
    }
  }

  private detachAvoidAreaListener(): void {
    if (this.mainWindow && this.avoidAreaCallback) {
      try {
        this.mainWindow.off('avoidAreaChange', this.avoidAreaCallback);
      } catch (error) {
        hilog.warn(DOMAIN_NUMBER, TAG, '移除 avoidAreaChange 监听失败: %{public}s', JSON.stringify(error));
      }
    }
    this.avoidAreaCallback = undefined;
  }

  /**
   * 根据深色/浅色模式统一设置窗口背景和系统栏样式。
   *
   * 全屏布局配色方案：
   * - 系统栏背景 = 透明，让页面内容可以延伸到状态栏区域
   * - 状态栏内容颜色根据深色/浅色模式设置
   * - 页面中需要避开系统栏的组件使用 safeAreaLayout 区域
   */
  private initializeSystemSurfaceSync(mainWindow: window.Window): void {
    this.surfaceStyleUnsubscribe?.();
    this.currentSurfaceStyle = SystemUiThemeStore.getStyle();
    this.surfaceStyleUnsubscribe = SystemUiThemeStore.subscribe((style: SystemSurfaceStyle) => {
      this.currentSurfaceStyle = style;
      this.applySystemBarStyle(mainWindow, this.isDarkMode, style).catch(() => {});
    });
    this.applySystemBarStyle(mainWindow, this.isDarkMode, this.currentSurfaceStyle).catch(() => {});
  }

  private async applySystemBarStyle(mainWindow: window.Window, isDark: boolean, style?: SystemSurfaceStyle): Promise<void> {
    try {
      const windowBgColor = this.getWindowBaseBackgroundColor(isDark);
      // 状态栏内容颜色完全由系统深浅模式决定：浅色模式用黑色图标，深色模式用白色图标
      // 不使用页面传入的 preferDarkContent，因为透明状态栏下背景已通过 $r() 自动适配深浅
      const contentUsesDarkColor = !isDark;
      const contentColor = contentUsesDarkColor ? '#000000' : '#FFFFFF';

      // 窗口背景色跟随深浅模式，确保任务视图下颜色正确
      mainWindow.setWindowBackgroundColor(windowBgColor);

      // 状态栏和导航栏设为透明，让页面内容（已通过 expandSafeArea 延伸）自然填充
      await mainWindow.setWindowSystemBarProperties({
        statusBarColor: '#00000000',
        navigationBarColor: '#00000000',
        statusBarContentColor: contentColor,
        navigationBarContentColor: contentColor,
        isStatusBarLightIcon: !contentUsesDarkColor,
        isNavigationBarLightIcon: !contentUsesDarkColor
      });
      hilog.info(DOMAIN_NUMBER, TAG,
        `系统栏样式已更新: isDark=${isDark}, transparent bars, contentDark=${contentUsesDarkColor}`);
    } catch {
      hilog.warn(DOMAIN_NUMBER, TAG, '更新系统栏样式失败');
    }
  }

  /**
   * 页面加载完成后的处理
   */
  private onPageLoaded(): void {
    // 可以在这里进行一些页面加载完成后的初始化工作
    hilog.info(DOMAIN_NUMBER, TAG, '页面加载完成，开始应用初始化');
  }

  /**
   * 应用前台处理：检查 token 有效性 + WS 重连
   */
  private handleAppForeground(): void {
    hilog.info(DOMAIN_NUMBER, TAG, '应用回到前台，检查 token 与 WS 状态');

    const isLoggedIn = AppStorage.get<boolean>(KEY_IS_LOGGED_IN) ?? false;
    if (!isLoggedIn) return;

    this.refreshTokenAndReconnectWs().catch((error: Error) => {
      hilog.warn(DOMAIN_NUMBER, TAG, `前台恢复失败: ${error.message}`);
    });
  }

  private async refreshTokenAndReconnectWs(): Promise<void> {
    // 调用 check_status 验证 token（HttpUtil 拦截器会自动 refresh 过期 token）
    const statusResp = await ApiClient.checkAuthStatus();
    if (statusResp.code !== 0 || !statusResp.data?.is_valid) {
      hilog.warn(DOMAIN_NUMBER, TAG, '前台恢复：token 已失效');
      return;
    }

    // 检查 WS 连接状态，断开则重连
    const ws = WebSocketService.getInstance();
    if (!ws.isConnected()) {
      const token = await PreferencesUtil.get<string>(StorageKeys.ACCESS_TOKEN, '');
      if (token) {
        hilog.info(DOMAIN_NUMBER, TAG, '前台恢复：WS 已断开，重新连接');
        await ws.connect(token);
      }
    }
  }

  /**
   * 应用后台处理
   */
  private handleAppBackground(): void {
    hilog.info(DOMAIN_NUMBER, TAG, '应用进入后台');
  }

  /**
   * 处理系统配置变化
   */
  private handleConfigurationChange(config: Configuration): void {
    // 处理深色/浅色主题切换 → 实时更新系统状态栏样式
    if (config.colorMode !== undefined && this.windowStage) {
      hilog.info(DOMAIN_NUMBER, TAG, '主题模式变化: %{public}d', config.colorMode);
      const isDark = config.colorMode === ConfigurationConstant.ColorMode.COLOR_MODE_DARK;
      this.isDarkMode = isDark;
      AppThemeManager.onSystemColorModeChange(isDark);
      this.windowStage.getMainWindow()
        .then((mainWindow: window.Window) => {
          this.applySystemBarStyle(mainWindow, isDark, this.currentSurfaceStyle);
          this.updateSafeAreaInsets(mainWindow, 'config_change');
        })
        .catch(() => {
          hilog.warn(DOMAIN_NUMBER, TAG, '更新系统栏样式失败');
        });
    }
    if (config.language !== undefined) {
      hilog.info(DOMAIN_NUMBER, TAG, '语言变化: %{public}s', config.language);
    }
  }

  /**
   * 处理新的Want请求
   */
  private handleNewWant(want: Want): void {
    // 处理分享、深度链接等
    if (want.action === 'ohos.want.action.sendData') {
      hilog.info(DOMAIN_NUMBER, TAG, '接收到分享请求');
    }
  }

  /**
   * 清理资源
   */
  /**
   * 初始化智能服务
   */
  private async initializeSmartServices(): Promise<void> {
    try {
      hilog.info(DOMAIN_NUMBER, TAG, '开始初始化智能服务');
      
      // 1. 注册所有服务
      this.serviceRegistry.registerServices();
      
      // 2. 预加载核心服务
      await this.serviceRegistry.preloadCoreServices();
      
      // 3. 初始化需要Context的服务
      await this.serviceRegistry.initializeContextServices(this.context);
      
      // 4. 预加载功能服务
      await this.serviceRegistry.preloadFeatureServices();
      
      // 5. 输出服务统计信息
      const stats = this.serviceRegistry.getServiceStats();
      hilog.info(DOMAIN_NUMBER, TAG, 
        `服务初始化完成: 总计${stats.totalServices}, 已初始化${stats.initializedServices}, 错误${stats.errorServices}`);
      
    } catch (error) {
      const err = error as BusinessError;
      hilog.error(DOMAIN_NUMBER, TAG, `智能服务初始化失败: ${err.message}`);
      
      // 即使智能服务初始化失败，也不影响应用基本功能
      // 可以在后台尝试重新初始化
      setTimeout(() => {
        this.retryServiceInitialization();
      }, 10000); // 10秒后重试
    }
  }

  /**
   * 重试服务初始化
   */
  private async retryServiceInitialization(): Promise<void> {
    try {
      hilog.info(DOMAIN_NUMBER, TAG, '重试智能服务初始化');
      await this.initializeSmartServices();
      hilog.info(DOMAIN_NUMBER, TAG, '智能服务重试初始化成功');
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `智能服务重试初始化失败: ${JSON.stringify(error)}`);
    }
  }

  private async cleanup(): Promise<void> {
    try {
      this.detachAvoidAreaListener();
      this.surfaceStyleUnsubscribe?.();
      this.surfaceStyleUnsubscribe = undefined;
      // 清理智能服务
      hilog.info(DOMAIN_NUMBER, TAG, '清理智能服务');
      await this.serviceRegistry.dispose();
      
      // 清理应用级资源
      hilog.info(DOMAIN_NUMBER, TAG, '清理应用级资源');

      hilog.info(DOMAIN_NUMBER, TAG, '资源清理完成');
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, '资源清理失败: %{public}s', JSON.stringify(error));
    }
  }
}
