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
import { window } from '@kit.ArkUI';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { notificationManager } from '@kit.NotificationKit';
import { deviceInfo, BusinessError } from '@kit.BasicServicesKit';
import { geoLocationManager } from '@kit.LocationKit';
import { ServiceRegistry } from '../core/services/ServiceRegistry';
import { RdbManager } from '../core/database/RdbManager';

const DOMAIN_NUMBER: number = 0xFF00;
const TAG: string = 'GeoTalk';

export default class EntryAbility extends UIAbility {
  private windowStage: window.WindowStage | null = null;
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

      hilog.info(DOMAIN_NUMBER, TAG, '窗口配置完成');
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, '窗口配置失败: %{public}s', JSON.stringify(error));
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
   * 应用前台处理
   */
  private handleAppForeground(): void {
    // 应用回到前台时可以刷新数据、重新连接等
    hilog.info(DOMAIN_NUMBER, TAG, '应用回到前台，刷新应用状态');
  }

  /**
   * 应用后台处理
   */
  private handleAppBackground(): void {
    // 应用进入后台时可以保存状态、断开连接等
    hilog.info(DOMAIN_NUMBER, TAG, '应用进入后台，保存应用状态');
  }

  /**
   * 处理系统配置变化
   */
  private handleConfigurationChange(config: Configuration): void {
    // 处理语言、主题、屏幕方向等变化
    if (config.colorMode !== undefined) {
      hilog.info(DOMAIN_NUMBER, TAG, '主题模式变化: %{public}d', config.colorMode);
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
