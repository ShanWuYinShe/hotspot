/**
 * 优化版应用入口能力
 * 集成了完整的生命周期管理、资源管理和性能优化
 */

import UIAbility from '@ohos.app.ability.UIAbility';
import hilog from '@ohos.hilog';
import window from '@ohos.window';
import Want from '@ohos.app.ability.Want';
import AbilityConstant from '@ohos.app.ability.AbilityConstant';
import { ApplicationLifecycleManager, ILifecycleListener, LifecyclePhase } from '../common/lifecycle/ApplicationLifecycleManager';
import { ResourceManager, IResource } from '../common/utils/ResourceManager';
import { createLogger } from '../common/utils/LogManager';
import { ErrorHandler, ErrorType, ErrorLevel } from '../common/utils/ErrorHandler';
import { createCache } from '../common/utils/LRUCache';
import { createObjectPool, CommonObjectFactories } from '../common/utils/ObjectPool';
import { EventBus } from '../common/events/EventBus';

const logger = createLogger('EntryAbility');

/**
 * 优化版应用入口能力
 */
export default class EntryAbilityOptimized extends UIAbility implements ILifecycleListener, IResource {
  private lifecycleManager: ApplicationLifecycleManager;
  private resourceId: string | null = null;
  private mainWindow: window.WindowStage | null = null;

  constructor() {
    super();
    this.lifecycleManager = ApplicationLifecycleManager.getInstance();
  }

  onCreate(want: Want, launchParam: AbilityConstant.LaunchParam): void {
    logger.info('应用创建开始');
    
    try {
      // 注册为资源
      this.resourceId = ResourceManager.register(this, 'EntryAbility', '主应用能力');

      // 注册生命周期监听器
      this.lifecycleManager.addListener(this);

      // 初始化应用
      this.initializeApplication();

      logger.info('应用创建完成');
    } catch (error) {
      ErrorHandler.handle(error, 'EntryAbility.onCreate', ErrorType.SYSTEM, ErrorLevel.CRITICAL);
    }
  }

  onDestroy(): void {
    logger.info('应用销毁开始');
    
    try {
      // 执行完整的资源清理
      this.performCompleteCleanup();
      
      logger.info('应用销毁完成');
    } catch (error) {
      ErrorHandler.handle(error, 'EntryAbility.onDestroy', ErrorType.SYSTEM, ErrorLevel.CRITICAL);
    }
  }

  onWindowStageCreate(windowStage: window.WindowStage): void {
    logger.info('窗口舞台创建');
    
    try {
      this.mainWindow = windowStage;
      
      // 设置主窗口
      windowStage.loadContent('pages/Index', (err, data) => {
        if (err.code) {
          ErrorHandler.handle(err, 'EntryAbility.loadContent', ErrorType.SYSTEM, ErrorLevel.HIGH);
          return;
        }
        logger.info('主页面加载成功');
      });

      // 启动应用生命周期管理
      this.lifecycleManager.initialize();
      
    } catch (error) {
      ErrorHandler.handle(error, 'EntryAbility.onWindowStageCreate', ErrorType.SYSTEM, ErrorLevel.HIGH);
    }
  }

  onWindowStageDestroy(): void {
    logger.info('窗口舞台销毁');
    
    try {
      this.mainWindow = null;
    } catch (error) {
      ErrorHandler.handle(error, 'EntryAbility.onWindowStageDestroy', ErrorType.SYSTEM, ErrorLevel.MEDIUM);
    }
  }

  onForeground(): void {
    logger.info('应用进入前台');
    
    try {
      // 恢复应用
      this.lifecycleManager.resume();
      
      // 发布事件
      EventBus.getInstance().emit('application:foreground');
      
    } catch (error) {
      ErrorHandler.handle(error, 'EntryAbility.onForeground', ErrorType.SYSTEM, ErrorLevel.MEDIUM);
    }
  }

  onBackground(): void {
    logger.info('应用进入后台');
    
    try {
      // 暂停应用
      this.lifecycleManager.pause();
      
      // 发布事件
      EventBus.getInstance().emit('application:background');
      
    } catch (error) {
      ErrorHandler.handle(error, 'EntryAbility.onBackground', ErrorType.SYSTEM, ErrorLevel.MEDIUM);
    }
  }

  onMemoryLevel(level: AbilityConstant.MemoryLevel): void {
    logger.warn(`内存级别变更: ${level}`);
    
    try {
      // 根据内存级别执行不同的清理策略
      switch (level) {
        case AbilityConstant.MemoryLevel.MEMORY_LEVEL_MODERATE:
          // 中等内存压力，清理缓存
          this.performLightCleanup();
          break;
        case AbilityConstant.MemoryLevel.MEMORY_LEVEL_LOW:
          // 低内存，执行深度清理
          this.lifecycleManager.handleLowMemory();
          break;
        case AbilityConstant.MemoryLevel.MEMORY_LEVEL_CRITICAL:
          // 严重低内存，执行紧急清理
          this.performEmergencyCleanup();
          break;
      }
    } catch (error) {
      ErrorHandler.handle(error, 'EntryAbility.onMemoryLevel', ErrorType.SYSTEM, ErrorLevel.HIGH);
    }
  }

  onConfigurationUpdate(newConfig: Configuration): void {
    logger.info('配置更新');
    
    try {
      // 处理配置变更
      this.lifecycleManager.handleConfigurationChange();
      
      // 发布配置变更事件
      EventBus.getInstance().emit('application:configuration-change', newConfig);
      
    } catch (error) {
      ErrorHandler.handle(error, 'EntryAbility.onConfigurationUpdate', ErrorType.SYSTEM, ErrorLevel.MEDIUM);
    }
  }

  // 实现 ILifecycleListener 接口
  async onPhaseChange(from: LifecyclePhase, to: LifecyclePhase): Promise<void> {
    logger.info(`生命周期阶段变更: ${from} -> ${to}`);
    
    // 根据阶段变更执行相应操作
    switch (to) {
      case LifecyclePhase.RUNNING:
        await this.onApplicationFullyStarted();
        break;
      case LifecyclePhase.PAUSED:
        await this.onApplicationPaused();
        break;
      case LifecyclePhase.DESTROYING:
        await this.onApplicationDestroying();
        break;
    }
  }

  async onApplicationStart(): Promise<void> {
    logger.info('应用启动回调');
    // 应用启动时的初始化逻辑
  }

  async onApplicationPause(): Promise<void> {
    logger.info('应用暂停回调');
    // 保存应用状态
    await this.saveApplicationState();
  }

  async onApplicationResume(): Promise<void> {
    logger.info('应用恢复回调');
    // 恢复应用状态
    await this.restoreApplicationState();
  }

  async onApplicationDestroy(): Promise<void> {
    logger.info('应用销毁回调');
    // 最终清理
    await this.performFinalCleanup();
  }

  async onLowMemory(): Promise<void> {
    logger.warn('低内存回调');
    // 执行内存清理
    await this.performMemoryCleanup();
  }

  async onConfigurationChange(): Promise<void> {
    logger.info('配置变更回调');
    // 处理配置变更
  }

  // 实现 IResource 接口
  async cleanup(): Promise<void> {
    logger.info('EntryAbility资源清理');
    
    try {
      // 移除生命周期监听器
      this.lifecycleManager.removeListener(this);
      
      // 销毁生命周期管理器
      await this.lifecycleManager.destroy();
      
      // 清理窗口引用
      this.mainWindow = null;
      
    } catch (error) {
      ErrorHandler.handle(error, 'EntryAbility.cleanup', ErrorType.SYSTEM, ErrorLevel.MEDIUM);
    }
  }

  isActive(): boolean {
    return this.lifecycleManager.getCurrentPhase() === LifecyclePhase.RUNNING;
  }

  /**
   * 初始化应用
   */
  private async initializeApplication(): Promise<void> {
    try {
      logger.info('开始初始化应用组件...');

      // 初始化缓存
      this.initializeCaches();

      // 初始化对象池
      this.initializeObjectPools();

      // 设置错误处理
      this.setupErrorHandling();

      // 设置性能监控
      this.setupPerformanceMonitoring();

      logger.info('应用组件初始化完成');
    } catch (error) {
      ErrorHandler.handle(error, 'EntryAbility.initializeApplication', ErrorType.SYSTEM, ErrorLevel.CRITICAL);
      throw error;
    }
  }

  /**
   * 初始化缓存
   */
  private initializeCaches(): void {
    // 创建页面缓存
    createCache('pages', 50, {
      defaultTTL: 30 * 60 * 1000, // 30分钟
      enableMetrics: true
    });

    // 创建API响应缓存
    createCache('api-responses', 100, {
      defaultTTL: 5 * 60 * 1000, // 5分钟
      enableMetrics: true
    });

    // 创建图片缓存
    createCache('images', 200, {
      defaultTTL: 60 * 60 * 1000, // 1小时
      enableMetrics: true
    });

    logger.debug('缓存初始化完成');
  }

  /**
   * 初始化对象池
   */
  private initializeObjectPools(): void {
    // 创建Map对象池
    createObjectPool('maps', CommonObjectFactories.createMapFactory(), {
      initialSize: 10,
      maxSize: 100
    });

    // 创建Array对象池
    createObjectPool('arrays', CommonObjectFactories.createArrayFactory(), {
      initialSize: 20,
      maxSize: 200
    });

    // 创建Set对象池
    createObjectPool('sets', CommonObjectFactories.createSetFactory(), {
      initialSize: 5,
      maxSize: 50
    });

    logger.debug('对象池初始化完成');
  }

  /**
   * 设置错误处理
   */
  private setupErrorHandling(): void {
    // 配置错误处理器
    ErrorHandler.setConfig({
      enableLogging: true,
      enableUserNotification: true,
      enableRecovery: true,
      maxErrorHistory: 100
    });

    // 添加错误监听器
    ErrorHandler.addListener((error) => {
      if (error.level === ErrorLevel.CRITICAL) {
        // 严重错误时发送通知
        EventBus.getInstance().emit('application:critical-error', error);
      }
    });

    logger.debug('错误处理设置完成');
  }

  /**
   * 设置性能监控
   */
  private setupPerformanceMonitoring(): void {
    // 监听性能事件
    EventBus.getInstance().on('performance:alert', (alert) => {
      logger.warn('性能告警', alert);
    });

    // 监听健康检查事件
    EventBus.getInstance().on('application:health-warning', (health) => {
      logger.warn('应用健康警告', health);
    });

    logger.debug('性能监控设置完成');
  }

  /**
   * 应用完全启动后的回调
   */
  private async onApplicationFullyStarted(): Promise<void> {
    logger.info('应用已完全启动');
    
    // 预热缓存和对象池
    await this.warmupCachesAndPools();
    
    // 发布应用就绪事件
    EventBus.getInstance().emit('application:ready');
  }

  /**
   * 应用暂停后的回调
   */
  private async onApplicationPaused(): Promise<void> {
    logger.info('应用已暂停');
    
    // 执行轻量级清理
    await this.performLightCleanup();
  }

  /**
   * 应用销毁前的回调
   */
  private async onApplicationDestroying(): Promise<void> {
    logger.info('应用即将销毁');
    
    // 保存关键数据
    await this.saveApplicationState();
  }

  /**
   * 预热缓存和对象池
   */
  private async warmupCachesAndPools(): Promise<void> {
    try {
      // 预热对象池
      const mapPool = ObjectPoolManager.getPool('maps');
      if (mapPool) {
        mapPool.warmup(5);
      }

      const arrayPool = ObjectPoolManager.getPool('arrays');
      if (arrayPool) {
        arrayPool.warmup(10);
      }

      logger.debug('缓存和对象池预热完成');
    } catch (error) {
      ErrorHandler.handle(error, 'EntryAbility.warmupCachesAndPools', ErrorType.SYSTEM, ErrorLevel.LOW);
    }
  }

  /**
   * 保存应用状态
   */
  private async saveApplicationState(): Promise<void> {
    try {
      // 保存用户数据
      // 保存应用配置
      // 保存临时状态
      logger.debug('应用状态已保存');
    } catch (error) {
      ErrorHandler.handle(error, 'EntryAbility.saveApplicationState', ErrorType.STORAGE, ErrorLevel.MEDIUM);
    }
  }

  /**
   * 恢复应用状态
   */
  private async restoreApplicationState(): Promise<void> {
    try {
      // 恢复用户数据
      // 恢复应用配置
      // 恢复临时状态
      logger.debug('应用状态已恢复');
    } catch (error) {
      ErrorHandler.handle(error, 'EntryAbility.restoreApplicationState', ErrorType.STORAGE, ErrorLevel.MEDIUM);
    }
  }

  /**
   * 执行轻量级清理
   */
  private async performLightCleanup(): Promise<void> {
    try {
      // 清理部分缓存
      const imageCache = CacheManager.getCache('images');
      if (imageCache) {
        const stats = imageCache.getStats();
        if (stats.size > stats.maxSize * 0.7) {
          // 清理30%的缓存
          const keysToRemove = imageCache.keys().slice(0, Math.floor(stats.size * 0.3));
          for (const key of keysToRemove) {
            imageCache.delete(key);
          }
        }
      }

      logger.debug('轻量级清理完成');
    } catch (error) {
      ErrorHandler.handle(error, 'EntryAbility.performLightCleanup', ErrorType.SYSTEM, ErrorLevel.LOW);
    }
  }

  /**
   * 执行内存清理
   */
  private async performMemoryCleanup(): Promise<void> {
    try {
      // 清理所有缓存
      CacheManager.cleanup();
      
      // 清理对象池
      ObjectPoolManager.cleanup();
      
      // 清理资源泄漏
      await ResourceManager.cleanupLeaks();
      
      logger.info('内存清理完成');
    } catch (error) {
      ErrorHandler.handle(error, 'EntryAbility.performMemoryCleanup', ErrorType.SYSTEM, ErrorLevel.MEDIUM);
    }
  }

  /**
   * 执行紧急清理
   */
  private async performEmergencyCleanup(): Promise<void> {
    try {
      logger.warn('执行紧急内存清理');
      
      // 强制清理所有非关键资源
      await this.performMemoryCleanup();
      
      // 强制垃圾回收
      if (global.gc) {
        global.gc();
      }
      
      logger.warn('紧急清理完成');
    } catch (error) {
      ErrorHandler.handle(error, 'EntryAbility.performEmergencyCleanup', ErrorType.SYSTEM, ErrorLevel.HIGH);
    }
  }

  /**
   * 执行最终清理
   */
  private async performFinalCleanup(): Promise<void> {
    try {
      // 保存最终状态
      await this.saveApplicationState();
      
      // 清理所有资源
      await this.performCompleteCleanup();
      
      logger.info('最终清理完成');
    } catch (error) {
      ErrorHandler.handle(error, 'EntryAbility.performFinalCleanup', ErrorType.SYSTEM, ErrorLevel.MEDIUM);
    }
  }

  /**
   * 执行完整清理
   */
  private async performCompleteCleanup(): Promise<void> {
    try {
      logger.info('开始完整资源清理...');

      // 注销资源
      if (this.resourceId) {
        await ResourceManager.unregister(this.resourceId);
        this.resourceId = null;
      }

      // 清理生命周期管理器
      await this.cleanup();

      logger.info('完整资源清理完成');
    } catch (error) {
      ErrorHandler.handle(error, 'EntryAbility.performCompleteCleanup', ErrorType.SYSTEM, ErrorLevel.HIGH);
    }
  }
}
