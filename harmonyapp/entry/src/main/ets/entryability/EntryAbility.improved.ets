/**
 * 改进后的应用入口（示例）
 * 使用新的应用管理器和基础设施
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
import { window } from '@kit.ArkUI';
import { notificationManager } from '@kit.NotificationKit';
import { deviceInfo, BusinessError } from '@kit.BasicServicesKit';
import { geoLocationManager } from '@kit.LocationKit';

// 使用新的基础设施
import { ApplicationManager, AppState, initializeApp } from '../common/core/ApplicationManager';
import { createLogger, LogLevel } from '../utils/LogManager';
import { ErrorHandler, ErrorType, ErrorLevel } from '../utils/ErrorHandler';
import { TimerManager } from '../utils/TimerManager';

const logger = createLogger('EntryAbility');

export default class ImprovedEntryAbility extends UIAbility {
  private windowStage: window.WindowStage | null = null;
  private appManager: ApplicationManager | null = null;

  async onCreate(want: Want, launchParam: AbilityConstant.LaunchParam): Promise<void> {
    try {
      logger.info('HotSpot应用启动', { 
        launchReason: launchParam.launchReason,
        want: want.action 
      });

      // 初始化应用管理器
      await this.initializeApplicationManager();

      // 初始化应用级配置
      await this.initializeAppConfiguration();

      // 设置应用主题跟随系统
      this.setupThemeConfiguration();

      // 初始化通知管理
      await this.initializeNotifications();

      // 获取设备信息
      await this.logDeviceInfo();

      logger.info('应用启动完成');

    } catch (error) {
      ErrorHandler.handle(error, 'EntryAbility.onCreate', ErrorType.SYSTEM, ErrorLevel.CRITICAL);
    }
  }

  async onDestroy(): Promise<void> {
    try {
      logger.info('应用开始销毁');
      
      if (this.appManager) {
        await this.appManager.onExit();
      }
      
      logger.info('应用销毁完成');
    } catch (error) {
      ErrorHandler.handle(error, 'EntryAbility.onDestroy', ErrorType.SYSTEM, ErrorLevel.HIGH);
    }
  }

  onWindowStageCreate(windowStage: window.WindowStage): void {
    logger.info('WindowStage创建');
    this.windowStage = windowStage;

    // 配置窗口属性
    this.configureWindow(windowStage);

    // 加载主页面
    windowStage.loadContent('pages/Index')
      .then(() => {
        logger.info('主页面加载成功');
        this.onPageLoaded();
      })
      .catch((err: Error) => {
        ErrorHandler.handle(err, 'loadContent', ErrorType.SYSTEM, ErrorLevel.HIGH);
      });
  }

  onWindowStageDestroy(): void {
    logger.info('WindowStage销毁');
    this.windowStage = null;
  }

  onForeground(): void {
    logger.info('应用切换到前台');
    if (this.appManager) {
      this.appManager.onResume();
    }
    this.handleAppForeground();
  }

  onBackground(): void {
    logger.info('应用切换到后台');
    if (this.appManager) {
      this.appManager.onPause();
    }
    this.handleAppBackground();
  }

  onConfigurationUpdate(newConfig: Configuration): void {
    logger.info('系统配置更新', { config: newConfig });
    this.handleConfigurationChange(newConfig);
  }

  onNewWant(want: Want, launchParam: AbilityConstant.LaunchParam): void {
    logger.info('接收到新的Want请求', { want: want.action });
    this.handleNewWant(want);
  }

  /**
   * 初始化应用管理器
   */
  private async initializeApplicationManager(): Promise<void> {
    try {
      this.appManager = await initializeApp(this.context, {
        logLevel: LogLevel.INFO,
        enablePerformanceMonitoring: true,
        enableErrorReporting: true,
        enableTimerHealthCheck: true,
        timerHealthCheckInterval: 300000, // 5分钟
        maxErrorHistory: 100,
        autoRecovery: true
      });

      // 添加应用状态监听
      this.appManager.addStateListener((state: AppState) => {
        logger.info(`应用状态变更: ${state}`);
      });

      // 添加错误监听
      this.appManager.addErrorListener((error) => {
        logger.error('全局错误', error);
      });

      logger.info('应用管理器初始化完成');

    } catch (error) {
      logger.error('应用管理器初始化失败', error);
      throw error;
    }
  }

  /**
   * 初始化应用配置
   */
  private async initializeAppConfiguration(): Promise<void> {
    return ErrorHandler.safeExecute(async () => {
      // 获取当前应用的Bundle信息
      const bundleInfo = await bundleManager.getBundleInfoForSelf(
        bundleManager.BundleFlag.GET_BUNDLE_INFO_WITH_APPLICATION
      );

      logger.info('应用信息', {
        name: bundleInfo.name,
        versionName: bundleInfo.versionName,
        versionCode: bundleInfo.versionCode
      });

    }, 'initializeAppConfiguration', undefined, ErrorType.SYSTEM);
  }

  /**
   * 设置主题配置
   */
  private setupThemeConfiguration(): void {
    ErrorHandler.safeExecuteSync(() => {
      const appContext = this.context.getApplicationContext();
      appContext.setColorMode(ConfigurationConstant.ColorMode.COLOR_MODE_NOT_SET);
      logger.info('主题配置设置完成');
    }, 'setupThemeConfiguration', undefined, ErrorType.SYSTEM);
  }

  /**
   * 初始化通知管理
   */
  private async initializeNotifications(): Promise<void> {
    return ErrorHandler.safeExecute(async () => {
      // 请求通知权限
      await notificationManager.isNotificationEnabled().then(async (enabled) => {
        if (!enabled) {
          logger.info('通知权限未开启，请在设置中开启');
        }
      });

      // 创建通知渠道
      await notificationManager.addSlot(notificationManager.SlotType.SOCIAL_COMMUNICATION);
      logger.info('通知系统初始化成功');

    }, 'initializeNotifications', undefined, ErrorType.PERMISSION);
  }

  /**
   * 获取并记录设备信息
   */
  private async logDeviceInfo(): Promise<void> {
    return ErrorHandler.safeExecute(async () => {
      const device = deviceInfo.deviceType;
      const osVersion = deviceInfo.osFullName;
      const brand = deviceInfo.brand;
      const model = deviceInfo.productModel;

      logger.info('设备信息', {
        type: device,
        system: osVersion,
        brand: brand,
        model: model
      });

    }, 'logDeviceInfo', undefined, ErrorType.SYSTEM);
  }

  /**
   * 配置窗口属性
   */
  private async configureWindow(windowStage: window.WindowStage): Promise<void> {
    return ErrorHandler.safeExecute(async () => {
      const mainWindow = await windowStage.getMainWindow();

      // 设置窗口全屏
      await mainWindow.setWindowLayoutFullScreen(true);

      // 设置状态栏和导航栏样式
      await mainWindow.setWindowSystemBarEnable(['status', 'navigation']);
      await mainWindow.setWindowSystemBarProperties({
        statusBarColor: '#00000000',
        navigationBarColor: '#00000000',
        statusBarContentColor: '#FFFFFF',
        navigationBarContentColor: '#FFFFFF'
      });

      // 设置窗口方向
      await mainWindow.setPreferredOrientation(window.Orientation.AUTO_ROTATION);

      logger.info('窗口配置完成');

    }, 'configureWindow', undefined, ErrorType.SYSTEM);
  }

  /**
   * 页面加载完成后的处理
   */
  private onPageLoaded(): void {
    logger.info('页面加载完成，开始应用初始化');
    
    // 启动定时器健康检查
    TimerManager.startHealthCheck(300000); // 5分钟检查一次
  }

  /**
   * 应用前台处理
   */
  private handleAppForeground(): void {
    logger.info('应用回到前台，刷新应用状态');
    
    // 可以在这里添加前台恢复逻辑
    // 例如：重新连接WebSocket、刷新数据等
  }

  /**
   * 应用后台处理
   */
  private handleAppBackground(): void {
    logger.info('应用进入后台，保存应用状态');
    
    // 可以在这里添加后台保存逻辑
    // 例如：断开非必要连接、保存用户数据等
  }

  /**
   * 处理系统配置变化
   */
  private handleConfigurationChange(config: Configuration): void {
    if (config.colorMode !== undefined) {
      logger.info('主题模式变化', { colorMode: config.colorMode });
    }
    if (config.language !== undefined) {
      logger.info('语言变化', { language: config.language });
    }
  }

  /**
   * 处理新的Want请求
   */
  private handleNewWant(want: Want): void {
    if (want.action === 'ohos.want.action.sendData') {
      logger.info('接收到分享请求');
    }
  }

  /**
   * 获取应用统计信息（用于调试）
   */
  getAppStats() {
    return this.appManager?.getStats();
  }

  /**
   * 获取应用状态（用于调试）
   */
  getAppState() {
    return this.appManager?.getState();
  }
}
