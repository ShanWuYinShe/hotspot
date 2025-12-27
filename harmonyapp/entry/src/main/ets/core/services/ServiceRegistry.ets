/**
 * 服务注册中心
 * 使用LazyServiceManager管理所有服务
 */

import { LazyServiceManager, ServiceLifecycle, registerService } from './LazyServiceManager';
import { CacheManager } from '../cache/CacheManager';
import { SmartNotificationService } from './SmartNotificationService';
import { DistributedSyncService } from './DistributedSyncService';
import { AccessibilityManager } from '../accessibility/AccessibilityManager';
import { I18nManager } from '../i18n/I18nManager';
import { LogManager } from '../utils/LogManager';
import { TimerManager } from '../utils/TimerManager';
import { ErrorHandler } from '../utils/ErrorHandler';
import { EventBus } from '../events/EventBus';
import { UnifiedStorageUtil } from '../utils/UnifiedStorageUtil';
import { createLogger } from '../utils/LogManager';
import { common } from '@kit.AbilityKit';

const logger = createLogger('ServiceRegistry');

/**
 * 服务注册配置
 */
export interface ServiceRegistryConfig {
  enableCacheManager: boolean;
  enableSmartNotification: boolean;
  enableDistributedSync: boolean;
  enableAccessibility: boolean;
  enableI18n: boolean;
}

/**
 * 默认服务配置
 */
const DEFAULT_CONFIG: ServiceRegistryConfig = {
  enableCacheManager: true,
  enableSmartNotification: true,
  enableDistributedSync: true,
  enableAccessibility: true,
  enableI18n: true
};

/**
 * 服务注册中心
 */
export class ServiceRegistry {
  private static instance: ServiceRegistry | null = null;
  private lazyServiceManager: LazyServiceManager;
  private config: ServiceRegistryConfig;
  private isRegistered: boolean = false;

  private constructor() {
    this.lazyServiceManager = LazyServiceManager.getInstance();
    this.config = this.copyConfig(DEFAULT_CONFIG);
  }

  public static getInstance(): ServiceRegistry {
    if (!ServiceRegistry.instance) {
      ServiceRegistry.instance = new ServiceRegistry();
    }
    return ServiceRegistry.instance;
  }

  /**
   * 复制配置对象
   */
  private copyConfig(config: ServiceRegistryConfig): ServiceRegistryConfig {
    return {
      enableCacheManager: config.enableCacheManager,
      enableSmartNotification: config.enableSmartNotification,
      enableDistributedSync: config.enableDistributedSync,
      enableAccessibility: config.enableAccessibility,
      enableI18n: config.enableI18n
    };
  }

  /**
   * 合并配置对象
   */
  private mergeConfig(base: ServiceRegistryConfig, partial: Partial<ServiceRegistryConfig>): ServiceRegistryConfig {
    const result = this.copyConfig(base);
    if (partial.enableCacheManager !== undefined) {
      result.enableCacheManager = partial.enableCacheManager;
    }
    if (partial.enableSmartNotification !== undefined) {
      result.enableSmartNotification = partial.enableSmartNotification;
    }
    if (partial.enableDistributedSync !== undefined) {
      result.enableDistributedSync = partial.enableDistributedSync;
    }
    if (partial.enableAccessibility !== undefined) {
      result.enableAccessibility = partial.enableAccessibility;
    }
    if (partial.enableI18n !== undefined) {
      result.enableI18n = partial.enableI18n;
    }
    return result;
  }

  /**
   * 配置服务
   */
  public configure(config: Partial<ServiceRegistryConfig>): void {
    this.config = this.mergeConfig(this.config, config);
    logger.info('服务配置已更新', this.config);
  }

  /**
   * 注册所有服务
   */
  public registerServices(): void {
    if (this.isRegistered) {
      logger.warn('服务已经注册过了');
      return;
    }

    logger.info('开始注册服务');

    // 1. 注册基础工具服务
    registerService({
      name: 'LogManager',
      factory: (): LogManager => new LogManager(),
      lifecycle: ServiceLifecycle.SINGLETON,
      tags: ['core', 'utility']
    });

    registerService({
      name: 'TimerManager',
      factory: (): TimerManager => new TimerManager(),
      lifecycle: ServiceLifecycle.SINGLETON,
      tags: ['core', 'utility']
    });

    registerService({
      name: 'ErrorHandler',
      factory: (): ErrorHandler => new ErrorHandler(),
      lifecycle: ServiceLifecycle.SINGLETON,
      tags: ['core', 'utility']
    });

    registerService({
      name: 'EventBus',
      factory: () => EventBus.getInstance(),
      lifecycle: ServiceLifecycle.SINGLETON,
      dependencies: ['LogManager'],
      tags: ['core', 'communication']
    });

    registerService({
      name: 'UnifiedStorageUtil',
      factory: () => UnifiedStorageUtil.getInstance(),
      lifecycle: ServiceLifecycle.SINGLETON,
      dependencies: ['LogManager', 'ErrorHandler'],
      tags: ['core', 'storage']
    });

    // 2. 注册国际化服务
    if (this.config.enableI18n) {
      registerService({
        name: 'I18nManager',
        factory: () => I18nManager.getInstance(),
        lifecycle: ServiceLifecycle.SINGLETON,
        dependencies: ['LogManager', 'ErrorHandler', 'EventBus', 'UnifiedStorageUtil'],
        tags: ['feature', 'i18n']
      });
    }

    // 3. 注册缓存管理服务
    if (this.config.enableCacheManager) {
      registerService({
        name: 'CacheManager',
        factory: () => CacheManager.getInstance(),
        lifecycle: ServiceLifecycle.SINGLETON,
        dependencies: ['LogManager', 'TimerManager', 'ErrorHandler'],
        tags: ['feature', 'cache']
      });
    }

    // 4. 注册通知服务
    if (this.config.enableSmartNotification) {
      registerService({
        name: 'SmartNotificationService',
        factory: () => SmartNotificationService.getInstance(),
        lifecycle: ServiceLifecycle.SINGLETON,
        dependencies: ['LogManager', 'TimerManager', 'ErrorHandler', 'UnifiedStorageUtil'],
        tags: ['feature', 'notification']
      });
    }

    // 5. 注册分布式同步服务
    if (this.config.enableDistributedSync) {
      registerService({
        name: 'DistributedSyncService',
        factory: () => DistributedSyncService.getInstance(),
        lifecycle: ServiceLifecycle.SINGLETON,
        dependencies: ['LogManager', 'ErrorHandler', 'EventBus'],
        tags: ['feature', 'distributed']
      });
    }

    // 6. 注册无障碍服务
    if (this.config.enableAccessibility) {
      registerService({
        name: 'AccessibilityManager',
        factory: () => AccessibilityManager.getInstance(),
        lifecycle: ServiceLifecycle.SINGLETON,
        dependencies: ['LogManager'],
        tags: ['feature', 'accessibility']
      });
    }

    this.isRegistered = true;
    logger.info('所有服务注册完成');
  }

  /**
   * 预加载核心服务
   */
  public async preloadCoreServices(): Promise<void> {
    logger.info('开始预加载核心服务');
    
    try {
      await this.lazyServiceManager.preloadByTag('core');
      logger.info('核心服务预加载完成');
    } catch (error) {
      logger.error('核心服务预加载失败', error);
      throw new Error(String(error));
    }
  }

  /**
   * 预加载功能服务
   */
  public async preloadFeatureServices(): Promise<void> {
    logger.info('开始预加载功能服务');
    
    try {
      await this.lazyServiceManager.preloadByTag('feature');
      logger.info('功能服务预加载完成');
    } catch (error) {
      logger.error('功能服务预加载失败', error);
      throw new Error(String(error));
    }
  }

  /**
   * 初始化需要Context的服务
   */
  public async initializeContextServices(context: common.UIAbilityContext): Promise<void> {
    logger.info('开始初始化需要Context的服务');

    try {
      // 初始化国际化服务
      if (this.config.enableI18n) {
        const i18nManager = await this.lazyServiceManager.get<I18nManager>('I18nManager');
        i18nManager.setContext(context);
        await i18nManager.initialize();
      }

      // 初始化分布式同步服务
      if (this.config.enableDistributedSync) {
        const distributedService = await this.lazyServiceManager.get<DistributedSyncService>('DistributedSyncService');
        await distributedService.initialize(context);
      }

      logger.info('Context服务初始化完成');
    } catch (error) {
      logger.error('Context服务初始化失败', error);
      throw new Error(String(error));
    }
  }

  /**
   * 获取服务
   */
  public async getService<T extends Object>(serviceName: string): Promise<T> {
    return this.lazyServiceManager.get<T>(serviceName);
  }

  /**
   * 检查服务是否已注册
   */
  public isServiceRegistered(serviceName: string): boolean {
    return this.lazyServiceManager.isRegistered(serviceName);
  }

  /**
   * 检查服务是否已初始化
   */
  public isServiceInitialized(serviceName: string): boolean {
    return this.lazyServiceManager.isInitialized(serviceName);
  }

  /**
   * 获取服务统计信息
   */
  public getServiceStats() {
    return this.lazyServiceManager.getStats();
  }

  /**
   * 创建作用域
   */
  public createScope(): string {
    return this.lazyServiceManager.createScope();
  }

  /**
   * 设置当前作用域
   */
  public setCurrentScope(scopeId: string | null): void {
    this.lazyServiceManager.setCurrentScope(scopeId);
  }

  /**
   * 销毁作用域
   */
  public async disposeScope(scopeId: string): Promise<void> {
    await this.lazyServiceManager.disposeScope(scopeId);
  }

  /**
   * 清理未使用的服务
   */
  public async cleanup(maxIdleTime?: number): Promise<void> {
    await this.lazyServiceManager.cleanup(maxIdleTime);
  }

  /**
   * 销毁所有服务
   */
  public async dispose(): Promise<void> {
    await this.lazyServiceManager.dispose();
    this.isRegistered = false;
    ServiceRegistry.instance = null;
  }
}

/**
 * 便捷函数：获取服务注册中心
 */
export function getServiceRegistry(): ServiceRegistry {
  return ServiceRegistry.getInstance();
}

/**
 * 便捷函数：获取服务
 */
export async function getService<T extends Object>(serviceName: string): Promise<T> {
  return ServiceRegistry.getInstance().getService<T>(serviceName);
}
