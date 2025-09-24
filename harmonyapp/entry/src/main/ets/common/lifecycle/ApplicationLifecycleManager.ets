/**
 * 应用生命周期管理器
 * 统一管理应用的启动、运行和关闭过程，确保资源正确清理
 */

import { createLogger } from '../utils/LogManager';
import { ErrorHandler, ErrorType, ErrorLevel } from '../utils/ErrorHandler';
import { TimerManager } from '../utils/TimerManager';
import { ResourceManager } from '../utils/ResourceManager';
import { ObjectPoolManager } from '../utils/ObjectPool';
import { CacheManager } from '../utils/LRUCache';
import { EventBus } from '../events/EventBus';
import { Container } from '../di/Container';
import { SingletonManager } from '../base/Singleton';

const logger = createLogger('ApplicationLifecycleManager');

/**
 * 应用生命周期阶段
 */
export enum LifecyclePhase {
  INITIALIZING = 'initializing',
  RUNNING = 'running',
  PAUSING = 'pausing',
  PAUSED = 'paused',
  RESUMING = 'resuming',
  DESTROYING = 'destroying',
  DESTROYED = 'destroyed'
}

/**
 * 生命周期监听器接口
 */
export interface ILifecycleListener {
  onPhaseChange?(from: LifecyclePhase, to: LifecyclePhase): Promise<void> | void;
  onApplicationStart?(): Promise<void> | void;
  onApplicationPause?(): Promise<void> | void;
  onApplicationResume?(): Promise<void> | void;
  onApplicationDestroy?(): Promise<void> | void;
  onLowMemory?(): Promise<void> | void;
  onConfigurationChange?(): Promise<void> | void;
}

/**
 * 应用健康状态
 */
export interface ApplicationHealth {
  phase: LifecyclePhase;
  uptime: number;
  memoryUsage: {
    used: number;
    total: number;
    percentage: number;
  };
  resourceCounts: {
    timers: number;
    eventListeners: number;
    singletons: number;
    caches: number;
    objectPools: number;
  };
  performance: {
    averageResponseTime: number;
    errorRate: number;
    cacheHitRate: number;
  };
  lastHealthCheck: number;
}

/**
 * 应用生命周期管理器
 */
export class ApplicationLifecycleManager {
  private static instance: ApplicationLifecycleManager | null = null;
  private currentPhase: LifecyclePhase = LifecyclePhase.INITIALIZING;
  private listeners: ILifecycleListener[] = [];
  private startTime: number = Date.now();
  private healthCheckTimerId: number = -1;
  private lastHealthCheck: ApplicationHealth | null = null;
  private isShuttingDown: boolean = false;

  private constructor() {
    this.setupGlobalErrorHandling();
    this.startHealthMonitoring();
  }

  /**
   * 获取单例实例
   */
  static getInstance(): ApplicationLifecycleManager {
    if (!ApplicationLifecycleManager.instance) {
      ApplicationLifecycleManager.instance = new ApplicationLifecycleManager();
    }
    return ApplicationLifecycleManager.instance;
  }

  /**
   * 初始化应用
   */
  async initialize(): Promise<void> {
    try {
      logger.info('开始应用初始化...');
      await this.changePhase(LifecyclePhase.INITIALIZING);

      // 初始化核心服务
      await this.initializeCoreServices();

      // 通知监听器
      await this.notifyListeners('onApplicationStart');

      await this.changePhase(LifecyclePhase.RUNNING);
      logger.info('应用初始化完成');
    } catch (error) {
      ErrorHandler.handle(error, 'ApplicationLifecycleManager.initialize', ErrorType.SYSTEM, ErrorLevel.CRITICAL);
      throw new Error('应用初始化失败');
    }
  }

  /**
   * 暂停应用
   */
  async pause(): Promise<void> {
    if (this.currentPhase !== LifecyclePhase.RUNNING) {
      logger.warn(`应用不在运行状态，无法暂停。当前状态: ${this.currentPhase}`);
      return;
    }

    try {
      logger.info('开始暂停应用...');
      await this.changePhase(LifecyclePhase.PAUSING);

      // 通知监听器
      await this.notifyListeners('onApplicationPause');

      // 暂停非关键服务
      await this.pauseNonCriticalServices();

      await this.changePhase(LifecyclePhase.PAUSED);
      logger.info('应用已暂停');
    } catch (error) {
      ErrorHandler.handle(error, 'ApplicationLifecycleManager.pause', ErrorType.SYSTEM, ErrorLevel.HIGH);
    }
  }

  /**
   * 恢复应用
   */
  async resume(): Promise<void> {
    if (this.currentPhase !== LifecyclePhase.PAUSED) {
      logger.warn(`应用不在暂停状态，无法恢复。当前状态: ${this.currentPhase}`);
      return;
    }

    try {
      logger.info('开始恢复应用...');
      await this.changePhase(LifecyclePhase.RESUMING);

      // 恢复服务
      await this.resumeServices();

      // 通知监听器
      await this.notifyListeners('onApplicationResume');

      await this.changePhase(LifecyclePhase.RUNNING);
      logger.info('应用已恢复');
    } catch (error) {
      ErrorHandler.handle(error, 'ApplicationLifecycleManager.resume', ErrorType.SYSTEM, ErrorLevel.HIGH);
    }
  }

  /**
   * 销毁应用
   */
  async destroy(): Promise<void> {
    if (this.isShuttingDown) {
      logger.warn('应用已在关闭过程中');
      return;
    }

    this.isShuttingDown = true;

    try {
      logger.info('开始销毁应用...');
      await this.changePhase(LifecyclePhase.DESTROYING);

      // 通知监听器
      await this.notifyListeners('onApplicationDestroy');

      // 执行完整清理
      await this.performCompleteCleanup();

      await this.changePhase(LifecyclePhase.DESTROYED);
      logger.info('应用已销毁');
    } catch (error) {
      ErrorHandler.handle(error, 'ApplicationLifecycleManager.destroy', ErrorType.SYSTEM, ErrorLevel.CRITICAL);
    } finally {
      this.isShuttingDown = false;
    }
  }

  /**
   * 处理低内存警告
   */
  async handleLowMemory(): Promise<void> {
    try {
      logger.warn('收到低内存警告，开始清理...');

      // 通知监听器
      await this.notifyListeners('onLowMemory');

      // 执行内存清理
      await this.performMemoryCleanup();

      logger.info('低内存清理完成');
    } catch (error) {
      ErrorHandler.handle(error, 'ApplicationLifecycleManager.handleLowMemory', ErrorType.SYSTEM, ErrorLevel.HIGH);
    }
  }

  /**
   * 处理配置变更
   */
  async handleConfigurationChange(): Promise<void> {
    try {
      logger.info('处理配置变更...');

      // 通知监听器
      await this.notifyListeners('onConfigurationChange');

      logger.info('配置变更处理完成');
    } catch (error) {
      ErrorHandler.handle(error, 'ApplicationLifecycleManager.handleConfigurationChange', ErrorType.SYSTEM, ErrorLevel.MEDIUM);
    }
  }

  /**
   * 添加生命周期监听器
   */
  addListener(listener: ILifecycleListener): void {
    this.listeners.push(listener);
    logger.debug('添加生命周期监听器');
  }

  /**
   * 移除生命周期监听器
   */
  removeListener(listener: ILifecycleListener): boolean {
    const index = this.listeners.indexOf(listener);
    if (index > -1) {
      this.listeners.splice(index, 1);
      logger.debug('移除生命周期监听器');
      return true;
    }
    return false;
  }

  /**
   * 获取当前生命周期阶段
   */
  getCurrentPhase(): LifecyclePhase {
    return this.currentPhase;
  }

  /**
   * 获取应用健康状态
   */
  getHealth(): ApplicationHealth {
    return this.lastHealthCheck || this.collectHealthData();
  }

  /**
   * 获取应用运行时间
   */
  getUptime(): number {
    return Date.now() - this.startTime;
  }

  /**
   * 检查应用是否健康
   */
  isHealthy(): boolean {
    const health = this.getHealth();
    
    // 检查各项健康指标
    const memoryOk = health.memoryUsage.percentage < 80;
    const performanceOk = health.performance.errorRate < 5;
    const resourcesOk = health.resourceCounts.timers < 100;

    return memoryOk && performanceOk && resourcesOk;
  }

  /**
   * 改变生命周期阶段
   */
  private async changePhase(newPhase: LifecyclePhase): Promise<void> {
    const oldPhase = this.currentPhase;
    this.currentPhase = newPhase;

    logger.info(`生命周期阶段变更: ${oldPhase} -> ${newPhase}`);

    // 通知监听器阶段变更
    for (const listener of this.listeners) {
      try {
        if (listener.onPhaseChange) {
          await listener.onPhaseChange(oldPhase, newPhase);
        }
      } catch (error) {
        ErrorHandler.handle(error, 'LifecycleListener.onPhaseChange', ErrorType.SYSTEM, ErrorLevel.MEDIUM);
      }
    }

    // 发布事件
    EventBus.getInstance().emit('lifecycle:phase-change', { from: oldPhase, to: newPhase });
  }

  /**
   * 通知监听器
   */
  private async notifyListeners(method: keyof ILifecycleListener): Promise<void> {
    const promises: Promise<void>[] = [];

    for (const listener of this.listeners) {
      const fn = listener[method] as Function;
      if (fn) {
        try {
          const result = fn.call(listener);
          if (result instanceof Promise) {
            promises.push(result);
          }
        } catch (error) {
          ErrorHandler.handle(error, `LifecycleListener.${method}`, ErrorType.SYSTEM, ErrorLevel.MEDIUM);
        }
      }
    }

    if (promises.length > 0) {
      await Promise.all(promises);
    }
  }

  /**
   * 初始化核心服务
   */
  private async initializeCoreServices(): Promise<void> {
    // 这里可以初始化各种核心服务
    logger.debug('核心服务初始化完成');
  }

  /**
   * 暂停非关键服务
   */
  private async pauseNonCriticalServices(): Promise<void> {
    // 暂停定时器
    // 暂停后台任务
    // 保存状态
    logger.debug('非关键服务已暂停');
  }

  /**
   * 恢复服务
   */
  private async resumeServices(): Promise<void> {
    // 恢复定时器
    // 恢复后台任务
    // 恢复状态
    logger.debug('服务已恢复');
  }

  /**
   * 执行完整清理
   */
  private async performCompleteCleanup(): Promise<void> {
    logger.info('开始完整资源清理...');

    const cleanupTasks = [
      () => this.stopHealthMonitoring(),
      () => TimerManager.cleanup(),
      () => ResourceManager.cleanup(),
      () => ObjectPoolManager.cleanup(),
      () => CacheManager.cleanup(),
      () => EventBus.getInstance().removeAllListeners(),
      () => Container.getInstance().cleanup(),
      () => SingletonManager.destroyAll()
    ];

    for (const task of cleanupTasks) {
      try {
        await task();
      } catch (error) {
        logger.error(`清理任务执行失败: ${error}`);
      }
    }

    logger.info('完整资源清理完成');
  }

  /**
   * 执行内存清理
   */
  private async performMemoryCleanup(): Promise<void> {
    logger.info('开始内存清理...');

    // 清理缓存
    const cacheStats = CacheManager.getAllStats();
    for (const [name, stats] of Object.entries(cacheStats)) {
      const cache = CacheManager.getCache(name);
      if (cache && stats.size > stats.maxSize * 0.8) {
        cache.clear();
        logger.debug(`清理缓存: ${name}`);
      }
    }

    // 清理对象池
    ObjectPoolManager.cleanup();

    // 清理资源泄漏
    await ResourceManager.cleanupLeaks();

    // 强制垃圾回收（如果支持）
    if (global.gc) {
      global.gc();
    }

    logger.info('内存清理完成');
  }

  /**
   * 收集健康数据
   */
  private collectHealthData(): ApplicationHealth {
    const now = Date.now();
    const timerStats = TimerManager.getStats();
    const resourceStats = ResourceManager.getStats();
    const cacheStats = CacheManager.getAllStats();

    // 计算总缓存命中率
    let totalHits = 0;
    let totalRequests = 0;
    for (const stats of Object.values(cacheStats)) {
      totalHits += stats.hitCount;
      totalRequests += stats.hitCount + stats.missCount;
    }
    const overallCacheHitRate = totalRequests > 0 ? (totalHits / totalRequests) * 100 : 0;

    return {
      phase: this.currentPhase,
      uptime: now - this.startTime,
      memoryUsage: {
        used: 0, // 需要实际的内存API
        total: 0,
        percentage: 0
      },
      resourceCounts: {
        timers: timerStats.totalTimers,
        eventListeners: EventBus.getInstance().getListenerCount(),
        singletons: SingletonManager.getAllInstances().length,
        caches: Object.keys(cacheStats).length,
        objectPools: Object.keys(ObjectPoolManager.getAllStats()).length
      },
      performance: {
        averageResponseTime: 0, // 需要从性能监控获取
        errorRate: 0, // 需要从错误统计获取
        cacheHitRate: overallCacheHitRate
      },
      lastHealthCheck: now
    };
  }

  /**
   * 启动健康监控
   */
  private startHealthMonitoring(): void {
    this.healthCheckTimerId = TimerManager.setInterval(() => {
      this.performHealthCheck();
    }, 60 * 1000, '应用健康检查'); // 每分钟检查一次
  }

  /**
   * 停止健康监控
   */
  private stopHealthMonitoring(): void {
    if (this.healthCheckTimerId !== -1) {
      TimerManager.clearInterval(this.healthCheckTimerId);
      this.healthCheckTimerId = -1;
    }
  }

  /**
   * 执行健康检查
   */
  private performHealthCheck(): void {
    try {
      this.lastHealthCheck = this.collectHealthData();
      
      if (!this.isHealthy()) {
        logger.warn('应用健康检查发现问题', this.lastHealthCheck);
        EventBus.getInstance().emit('application:health-warning', this.lastHealthCheck);
      }
    } catch (error) {
      ErrorHandler.handle(error, 'ApplicationLifecycleManager.performHealthCheck', ErrorType.SYSTEM, ErrorLevel.MEDIUM);
    }
  }

  /**
   * 设置全局错误处理
   */
  private setupGlobalErrorHandling(): void {
    // 设置未捕获异常处理
    process.on?.('uncaughtException', (error) => {
      ErrorHandler.handle(error, 'UncaughtException', ErrorType.SYSTEM, ErrorLevel.CRITICAL);
    });

    // 设置未处理的Promise拒绝
    process.on?.('unhandledRejection', (reason) => {
      ErrorHandler.handle(reason, 'UnhandledRejection', ErrorType.SYSTEM, ErrorLevel.HIGH);
    });
  }
}

/**
 * 便捷函数：获取应用生命周期管理器
 */
export function getApplicationLifecycleManager(): ApplicationLifecycleManager {
  return ApplicationLifecycleManager.getInstance();
}
