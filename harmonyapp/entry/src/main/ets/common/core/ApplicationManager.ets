/**
 * 应用管理器
 * 统一管理应用的生命周期、服务、资源等
 */

import { common } from '@kit.AbilityKit';
import { BaseSingleton, SingletonManager } from '../base/Singleton';
import { createLogger, LogManager, LogLevel } from '../utils/LogManager';
import { TimerManager } from '../utils/TimerManager';
import { ErrorHandler, ErrorType, ErrorLevel } from '../utils/ErrorHandler';
import { UnifiedStorageUtil } from '../utils/UnifiedStorageUtil';

const logger = createLogger('ApplicationManager');

/**
 * 应用状态枚举
 */
export enum AppState {
  INITIALIZING = 'initializing',
  RUNNING = 'running',
  PAUSED = 'paused',
  STOPPED = 'stopped',
  ERROR = 'error'
}

/**
 * 应用配置接口
 */
export interface AppConfig {
  logLevel: LogLevel;
  enablePerformanceMonitoring: boolean;
  enableErrorReporting: boolean;
  enableTimerHealthCheck: boolean;
  timerHealthCheckInterval: number;
  maxErrorHistory: number;
  autoRecovery: boolean;
}

/**
 * 应用统计信息
 */
export interface AppStats {
  uptime: number;
  state: AppState;
  totalErrors: number;
  totalTimers: number;
  memoryUsage: number;
  servicesCount: number;
  lastError?: string;
}

/**
 * 应用管理器
 * 负责整个应用的生命周期管理和资源协调
 */
export class ApplicationManager extends BaseSingleton {
  private context: common.Context | null = null;
  private state: AppState = AppState.INITIALIZING;
  private startTime: number = 0;
  private config: AppConfig;
  private healthCheckTimerId: number = -1;
  private errorListeners: ((error: any) => void)[] = [];
  private stateListeners: ((state: AppState) => void)[] = [];

  constructor() {
    super();
    this.config = {
      logLevel: LogLevel.INFO,
      enablePerformanceMonitoring: true,
      enableErrorReporting: true,
      enableTimerHealthCheck: true,
      timerHealthCheckInterval: 300000, // 5分钟
      maxErrorHistory: 100,
      autoRecovery: true
    };
  }

  /**
   * 获取单例实例
   */
  static getInstance(): ApplicationManager {
    return super.getInstance();
  }

  /**
   * 异步获取单例实例
   */
  static async getInstanceAsync(): Promise<ApplicationManager> {
    return super.getInstanceAsync();
  }

  /**
   * 初始化应用管理器
   */
  async initializeWithContext(context: common.Context, config?: Partial<AppConfig>): Promise<void> {
    this.context = context;
    if (config) {
      this.config = { ...this.config, ...config };
    }
    await this.initialize();
  }

  /**
   * 实现基类的初始化方法
   */
  protected async onInitialize(): Promise<void> {
    try {
      this.startTime = Date.now();
      this.setState(AppState.INITIALIZING);

      logger.info('应用管理器初始化开始');

      // 1. 配置日志系统
      this.setupLogging();

      // 2. 配置错误处理
      this.setupErrorHandling();

      // 3. 初始化存储系统
      await this.initializeStorage();

      // 4. 启动健康检查
      this.startHealthCheck();

      // 5. 设置应用状态监听
      this.setupStateListeners();

      this.setState(AppState.RUNNING);
      logger.info('应用管理器初始化完成');

    } catch (error) {
      this.setState(AppState.ERROR);
      ErrorHandler.handle(error, 'ApplicationManager.onInitialize', ErrorType.SYSTEM, ErrorLevel.CRITICAL);
      throw error;
    }
  }

  /**
   * 实现基类的清理方法
   */
  protected async onCleanup(): Promise<void> {
    try {
      logger.info('应用管理器开始清理');
      this.setState(AppState.STOPPED);

      // 停止健康检查
      this.stopHealthCheck();

      // 清理所有定时器
      TimerManager.cleanup();

      // 清理所有单例
      await SingletonManager.destroyAll();

      // 清理错误历史
      ErrorHandler.clearHistory();

      logger.info('应用管理器清理完成');

    } catch (error) {
      ErrorHandler.handle(error, 'ApplicationManager.onCleanup', ErrorType.SYSTEM);
    }
  }

  /**
   * 配置日志系统
   */
  private setupLogging(): void {
    LogManager.setConfig({
      level: this.config.logLevel,
      enableConsole: true,
      enableFile: false,
      maxLogSize: 10 * 1024 * 1024
    });
    logger.debug('日志系统配置完成');
  }

  /**
   * 配置错误处理
   */
  private setupErrorHandling(): void {
    ErrorHandler.setConfig({
      enableLogging: true,
      enableReporting: this.config.enableErrorReporting,
      maxErrorHistory: this.config.maxErrorHistory,
      autoRecovery: this.config.autoRecovery
    });

    // 添加全局错误监听器
    ErrorHandler.addErrorListener((error) => {
      this.notifyErrorListeners(error);
      
      // 严重错误时切换应用状态
      if (error.level === ErrorLevel.CRITICAL) {
        this.setState(AppState.ERROR);
      }
    });

    logger.debug('错误处理系统配置完成');
  }

  /**
   * 初始化存储系统
   */
  private async initializeStorage(): Promise<void> {
    if (!this.context) {
      throw new Error('Context未设置，无法初始化存储系统');
    }

    const storage = UnifiedStorageUtil.getInstance();
    await storage.initializeWithContext(this.context, {
      storeName: 'hotspot_app_main',
      enableEncryption: false,
      maxRetries: 3,
      retryDelay: 1000
    });

    logger.debug('存储系统初始化完成');
  }

  /**
   * 启动健康检查
   */
  private startHealthCheck(): void {
    if (!this.config.enableTimerHealthCheck) {
      return;
    }

    this.healthCheckTimerId = TimerManager.setInterval(() => {
      this.performHealthCheck();
    }, this.config.timerHealthCheckInterval, '应用健康检查');

    logger.debug('健康检查已启动');
  }

  /**
   * 停止健康检查
   */
  private stopHealthCheck(): void {
    if (this.healthCheckTimerId !== -1) {
      TimerManager.clearInterval(this.healthCheckTimerId);
      this.healthCheckTimerId = -1;
      logger.debug('健康检查已停止');
    }
  }

  /**
   * 执行健康检查
   */
  private performHealthCheck(): void {
    try {
      // 检查定时器状态
      const timerStats = TimerManager.getStats();
      if (timerStats.totalTimers > 50) {
        logger.warn(`定时器数量过多: ${timerStats.totalTimers}`);
      }

      // 检查错误统计
      const errorStats = ErrorHandler.getStats();
      if (errorStats.totalErrors > 100) {
        logger.warn(`错误数量过多: ${errorStats.totalErrors}`);
      }

      // 检查单例状态
      const singletonStats = SingletonManager.getStats();
      logger.debug(`健康检查完成 - 定时器: ${timerStats.totalTimers}, 错误: ${errorStats.totalErrors}, 单例: ${singletonStats.totalInstances}`);

    } catch (error) {
      ErrorHandler.handle(error, 'performHealthCheck', ErrorType.SYSTEM, ErrorLevel.MEDIUM);
    }
  }

  /**
   * 设置状态监听
   */
  private setupStateListeners(): void {
    // 可以在这里添加状态变化的处理逻辑
    this.addStateListener((state) => {
      logger.info(`应用状态变更: ${state}`);
    });
  }

  /**
   * 设置应用状态
   */
  private setState(state: AppState): void {
    if (this.state !== state) {
      const oldState = this.state;
      this.state = state;
      logger.info(`应用状态从 ${oldState} 变更为 ${state}`);
      this.notifyStateListeners(state);
    }
  }

  /**
   * 通知状态监听器
   */
  private notifyStateListeners(state: AppState): void {
    this.stateListeners.forEach(listener => {
      try {
        listener(state);
      } catch (error) {
        logger.error(`状态监听器执行失败: ${error}`);
      }
    });
  }

  /**
   * 通知错误监听器
   */
  private notifyErrorListeners(error: any): void {
    this.errorListeners.forEach(listener => {
      try {
        listener(error);
      } catch (listenerError) {
        logger.error(`错误监听器执行失败: ${listenerError}`);
      }
    });
  }

  /**
   * 添加状态监听器
   */
  addStateListener(listener: (state: AppState) => void): void {
    this.stateListeners.push(listener);
  }

  /**
   * 移除状态监听器
   */
  removeStateListener(listener: (state: AppState) => void): void {
    const index = this.stateListeners.indexOf(listener);
    if (index > -1) {
      this.stateListeners.splice(index, 1);
    }
  }

  /**
   * 添加错误监听器
   */
  addErrorListener(listener: (error: any) => void): void {
    this.errorListeners.push(listener);
  }

  /**
   * 移除错误监听器
   */
  removeErrorListener(listener: (error: any) => void): void {
    const index = this.errorListeners.indexOf(listener);
    if (index > -1) {
      this.errorListeners.splice(index, 1);
    }
  }

  /**
   * 获取应用统计信息
   */
  getStats(): AppStats {
    const timerStats = TimerManager.getStats();
    const errorStats = ErrorHandler.getStats();
    const singletonStats = SingletonManager.getStats();

    return {
      uptime: Date.now() - this.startTime,
      state: this.state,
      totalErrors: errorStats.totalErrors,
      totalTimers: timerStats.totalTimers,
      memoryUsage: 0, // 需要实际的内存使用率API
      servicesCount: singletonStats.totalInstances,
      lastError: errorStats.recentErrors.length > 0 
        ? errorStats.recentErrors[errorStats.recentErrors.length - 1].message 
        : undefined
    };
  }

  /**
   * 获取当前应用状态
   */
  getState(): AppState {
    return this.state;
  }

  /**
   * 获取应用配置
   */
  getConfig(): AppConfig {
    return { ...this.config };
  }

  /**
   * 更新应用配置
   */
  updateConfig(config: Partial<AppConfig>): void {
    this.config = { ...this.config, ...config };
    
    // 应用新配置
    if (config.logLevel !== undefined) {
      LogManager.setConfig({ level: config.logLevel });
    }
    
    if (config.enableErrorReporting !== undefined) {
      ErrorHandler.setConfig({ enableReporting: config.enableErrorReporting });
    }

    logger.info('应用配置已更新');
  }

  /**
   * 强制垃圾回收（如果支持）
   */
  forceGarbageCollection(): void {
    try {
      // HarmonyOS可能不支持手动GC，这里只是占位
      logger.info('尝试执行垃圾回收');
    } catch (error) {
      logger.warn('垃圾回收不支持或失败');
    }
  }

  /**
   * 应用暂停处理
   */
  onPause(): void {
    this.setState(AppState.PAUSED);
    logger.info('应用已暂停');
  }

  /**
   * 应用恢复处理
   */
  onResume(): void {
    this.setState(AppState.RUNNING);
    logger.info('应用已恢复');
  }

  /**
   * 应用退出处理
   */
  async onExit(): Promise<void> {
    logger.info('应用准备退出');
    await this.cleanup();
  }
}

/**
 * 便捷函数：获取应用管理器实例
 */
export async function getAppManager(): Promise<ApplicationManager> {
  return ApplicationManager.getInstanceAsync();
}

/**
 * 便捷函数：初始化应用管理器
 */
export async function initializeApp(
  context: common.Context, 
  config?: Partial<AppConfig>
): Promise<ApplicationManager> {
  const appManager = ApplicationManager.getInstance();
  await appManager.initializeWithContext(context, config);
  return appManager;
}
