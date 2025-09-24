/**
 * 懒加载服务管理器
 * 提供服务的懒加载和依赖注入功能，提升应用启动性能
 */

import { createLogger } from '../utils/LogManager';
import { ErrorHandler, ErrorType, ErrorLevel } from '../utils/ErrorHandler';
import { TimerManager } from '../utils/TimerManager';

const logger = createLogger('LazyServiceManager');

/**
 * 服务生命周期
 */
export enum ServiceLifecycle {
  SINGLETON = 'singleton',     // 单例模式
  TRANSIENT = 'transient',     // 每次创建新实例
  SCOPED = 'scoped'           // 作用域内单例
}

/**
 * 服务状态
 */
export enum ServiceStatus {
  UNINITIALIZED = 'uninitialized',
  INITIALIZING = 'initializing',
  INITIALIZED = 'initialized',
  ERROR = 'error',
  DISPOSED = 'disposed'
}

/**
 * 服务工厂函数类型
 */
export type ServiceFactory<T extends Object = Object> = () => Promise<T> | T;

/**
 * 服务配置
 */
interface ServiceConfig<T extends Object = Object> {
  /**
   * 服务名称
   */
  name: string;
  
  /**
   * 服务工厂函数
   */
  factory: ServiceFactory<T>;
  
  /**
   * 生命周期
   */
  lifecycle: ServiceLifecycle;
  
  /**
   * 依赖的其他服务
   */
  dependencies?: string[];
  
  /**
   * 是否延迟初始化
   */
  lazy?: boolean;
  
  /**
   * 初始化超时时间（毫秒）
   */
  timeout?: number;
  
  /**
   * 销毁函数
   */
  dispose?: (instance: T) => Promise<void> | void;
  
  /**
   * 服务标签（用于分组）
   */
  tags?: string[];
}

/**
 * 服务配置实现类
 */
class ServiceConfigImpl<T extends Object = Object> implements ServiceConfig<T> {
  name: string;
  factory: ServiceFactory<T>;
  lifecycle: ServiceLifecycle;
  dependencies?: string[];
  lazy?: boolean;
  timeout?: number;
  dispose?: (instance: T) => Promise<void> | void;
  tags?: string[];

  constructor(
    name: string,
    factory: ServiceFactory<T>,
    lifecycle: ServiceLifecycle,
    dependencies?: string[],
    lazy?: boolean,
    timeout?: number,
    dispose?: (instance: T) => Promise<void> | void,
    tags?: string[]
  ) {
    this.name = name;
    this.factory = factory;
    this.lifecycle = lifecycle;
    this.dependencies = dependencies;
    this.lazy = lazy;
    this.timeout = timeout;
    this.dispose = dispose;
    this.tags = tags;
  }
}

/**
 * 服务实例信息
 */
interface ServiceInstance<T extends Object = Object> {
  /**
   * 服务实例
   */
  instance: T;
  
  /**
   * 创建时间
   */
  createdAt: number;
  
  /**
   * 最后访问时间
   */
  lastAccessedAt: number;
  
  /**
   * 访问次数
   */
  accessCount: number;
  
  /**
   * 服务状态
   */
  status: ServiceStatus;
  
  /**
   * 作用域ID（用于scoped生命周期）
   */
  scopeId?: string;
}

/**
 * 服务实例实现类
 */
class ServiceInstanceImpl<T extends Object = Object> implements ServiceInstance<T> {
  instance: T;
  createdAt: number;
  lastAccessedAt: number;
  accessCount: number;
  status: ServiceStatus;
  scopeId?: string;
  
  constructor(
    instance: T | null,
    createdAt: number,
    lastAccessedAt: number,
    accessCount: number,
    status: ServiceStatus,
    scopeId?: string
  ) {
    this.instance = instance as T;
    this.createdAt = createdAt;
    this.lastAccessedAt = lastAccessedAt;
    this.accessCount = accessCount;
    this.status = status;
    this.scopeId = scopeId;
  }
}

/**
 * 内存使用统计接口
 */
interface MemoryUsageStats {
  singletons: number;
  transients: number;
  scoped: number;
}

/**
 * 性能统计接口
 */
interface PerformanceStats {
  averageInitTime: number;
  slowestService: string;
  fastestService: string;
}

/**
 * 服务性能统计接口
 */
interface ServicePerformanceStats {
  initTime: number;
  count: number;
}

/**
 * 服务统计信息
 */
export interface ServiceStats {
  /**
   * 总服务数
   */
  totalServices: number;
  
  /**
   * 已初始化服务数
   */
  initializedServices: number;
  
  /**
   * 正在初始化服务数
   */
  initializingServices: number;
  
  /**
   * 错误服务数
   */
  errorServices: number;
  
  /**
   * 内存使用情况
   */
  memoryUsage: MemoryUsageStats;
  
  /**
   * 性能统计
   */
  performance: PerformanceStats;
}

/**
 * 懒加载服务管理器
 */
export class LazyServiceManager {
  private static instance: LazyServiceManager | null = null;
  
  /**
   * 服务配置注册表
   */
  private serviceConfigs = new Map<string, ServiceConfig<Object>>();
  
  /**
   * 单例服务实例
   */
  private singletonInstances = new Map<string, ServiceInstance<Object>>();
  
  /**
   * 作用域服务实例
   */
  private scopedInstances = new Map<string, Map<string, ServiceInstance<Object>>>();
  
  /**
   * 正在初始化的服务
   */
  private initializingServices = new Map<string, Promise<Object>>();
  
  /**
   * 服务依赖图
   */
  private dependencyGraph = new Map<string, Set<string>>();
  
  /**
   * 当前作用域ID
   */
  private currentScopeId: string | null = null;
  
  /**
   * 性能统计
   */
  private performanceStats = new Map<string, ServicePerformanceStats>();
  
  private constructor() {
    logger.info('懒加载服务管理器初始化');
  }
  
  /**
   * 获取单例实例
   */
  static getInstance(): LazyServiceManager {
    if (!LazyServiceManager.instance) {
      LazyServiceManager.instance = new LazyServiceManager();
    }
    return LazyServiceManager.instance;
  }
  
  /**
   * 注册服务
   * @param config 服务配置
   */
  register<T extends Object>(config: ServiceConfig<T>): void {
    try {
      // 验证配置
      const objectConfig: ServiceConfig<Object> = {
        name: config.name,
        factory: config.factory as ServiceFactory<Object>,
        lifecycle: config.lifecycle,
        dependencies: config.dependencies,
        lazy: config.lazy,
        timeout: config.timeout,
        dispose: config.dispose as ((instance: Object) => Promise<void> | void) | undefined,
        tags: config.tags
      };
      this.validateServiceConfig(objectConfig);
      
      // 注册服务
      this.serviceConfigs.set(config.name, objectConfig);
      
      // 构建依赖图
      this.buildDependencyGraph(objectConfig);
      
      logger.debug(`服务 ${config.name} 注册成功`);
    } catch (error) {
      ErrorHandler.handle(error, `LazyServiceManager.register(${config.name})`, ErrorType.SYSTEM, ErrorLevel.HIGH);
      throw new Error(String(error));
    }
  }
  
  /**
   * 将ServiceConfig<Object>转换为ServiceConfig<T>
   */
  private convertServiceConfig<T extends Object>(config: ServiceConfig<Object>): ServiceConfig<T> {
    return new ServiceConfigImpl<T>(
      config.name,
      config.factory as ServiceFactory<T>,
      config.lifecycle,
      config.dependencies,
      config.lazy,
      config.timeout,
      config.dispose as ((instance: T) => Promise<void> | void) | undefined,
      config.tags
    );
  }

  /**
   * 获取服务实例
   * @param serviceName 服务名称
   * @returns 服务实例
   */
  async get<T extends Object>(serviceName: string): Promise<T> {
    try {
      const config = this.serviceConfigs.get(serviceName);
      if (!config) {
        throw new Error(`服务 ${serviceName} 未注册`);
      }
      
      // 根据生命周期获取实例
      switch (config.lifecycle) {
        case ServiceLifecycle.SINGLETON:
          return await this.getSingletonInstance<T>(serviceName, this.convertServiceConfig<T>(config));
        
        case ServiceLifecycle.TRANSIENT:
          return await this.createTransientInstance<T>(this.convertServiceConfig<T>(config));
        
        case ServiceLifecycle.SCOPED:
          return await this.getScopedInstance<T>(serviceName, this.convertServiceConfig<T>(config));
        
        default:
          throw new Error(`不支持的服务生命周期: ${config.lifecycle}`);
      }
    } catch (error) {
      ErrorHandler.handle(error, `LazyServiceManager.get(${serviceName})`, ErrorType.SYSTEM, ErrorLevel.HIGH);
      throw new Error(String(error));
    }
  }
  
  /**
   * 检查服务是否已注册
   * @param serviceName 服务名称
   */
  isRegistered(serviceName: string): boolean {
    return this.serviceConfigs.has(serviceName);
  }
  
  /**
   * 检查服务是否已初始化
   * @param serviceName 服务名称
   */
  isInitialized(serviceName: string): boolean {
    const singletonInstance = this.singletonInstances.get(serviceName);
    return singletonInstance?.status === ServiceStatus.INITIALIZED;
  }
  
  /**
   * 预加载服务
   * @param serviceNames 要预加载的服务名称列表
   */
  async preload(serviceNames: string[]): Promise<void> {
    try {
      logger.info(`开始预加载 ${serviceNames.length} 个服务`);
      
      const preloadPromises = serviceNames.map((serviceName: string) => {
        return new Promise<void>(async (resolve) => {
          try {
            await this.get<Object>(serviceName);
            logger.debug(`服务 ${serviceName} 预加载成功`);
          } catch (error) {
            logger.error(`服务 ${serviceName} 预加载失败`, error);
          }
          resolve();
        });
      });
      
      await Promise.all<void>(preloadPromises);
      logger.info('服务预加载完成');
    } catch (error) {
      ErrorHandler.handle(error, 'LazyServiceManager.preload', ErrorType.SYSTEM, ErrorLevel.MEDIUM);
    }
  }
  
  /**
   * 按标签预加载服务
   * @param tag 服务标签
   */
  async preloadByTag(tag: string): Promise<void> {
    const entries = Array.from(this.serviceConfigs.entries());
    const filteredEntries = entries.filter(entry => entry[1].tags?.includes(tag));
    const serviceNames = filteredEntries.map(entry => entry[0]);
    
    await this.preload(serviceNames);
  }
  
  /**
   * 创建新的作用域
   * @returns 作用域ID
   */
  createScope(): string {
    const scopeId = `scope_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
    this.scopedInstances.set(scopeId, new Map());
    logger.debug(`创建新作用域: ${scopeId}`);
    return scopeId;
  }
  
  /**
   * 设置当前作用域
   * @param scopeId 作用域ID
   */
  setCurrentScope(scopeId: string | null): void {
    this.currentScopeId = scopeId;
    logger.debug(`设置当前作用域: ${scopeId}`);
  }
  
  /**
   * 销毁作用域
   * @param scopeId 作用域ID
   */
  async disposeScope(scopeId: string): Promise<void> {
    try {
      const scopeInstances = this.scopedInstances.get(scopeId);
      if (!scopeInstances) {
        return;
      }
      
      // 销毁作用域内的所有服务实例
      const disposePromises = Array.from(scopeInstances.entries()).map((entry: [string, ServiceInstance<Object>]) => {
        return new Promise<void>(async (resolve) => {
          const serviceName = entry[0];
          const instanceInfo = entry[1];
          try {
            const config = this.serviceConfigs.get(serviceName);
            if (config?.dispose) {
              await config.dispose(instanceInfo.instance);
            }
            instanceInfo.status = ServiceStatus.DISPOSED;
          } catch (error) {
            logger.error(`销毁作用域服务 ${serviceName} 失败`, error);
          }
          resolve();
        });
      });
      
      await Promise.all<void>(disposePromises);
      
      // 删除作用域
      this.scopedInstances.delete(scopeId);
      
      logger.info(`作用域 ${scopeId} 已销毁`);
    } catch (error) {
      ErrorHandler.handle(error, `LazyServiceManager.disposeScope(${scopeId})`, ErrorType.SYSTEM, ErrorLevel.MEDIUM);
    }
  }
  
  /**
   * 获取服务统计信息
   */
  getStats(): ServiceStats {
    const totalServices = this.serviceConfigs.size;
    let initializedServices = 0;
    let initializingServices = 0;
    let errorServices = 0;
    
    // 统计单例服务状态
    for (const instanceInfo of this.singletonInstances.values()) {
      switch (instanceInfo.status) {
        case ServiceStatus.INITIALIZED:
          initializedServices++;
          break;
        case ServiceStatus.INITIALIZING:
          initializingServices++;
          break;
        case ServiceStatus.ERROR:
          errorServices++;
          break;
      }
    }
    
    // 计算性能统计
    const perfStats = Array.from(this.performanceStats.entries());
    const averageInitTime = perfStats.length > 0
      ? perfStats.reduce((sum: number, entry: [string, ServicePerformanceStats]) => {
          const stats = entry[1];
          return sum + stats.initTime;
        }, 0) / perfStats.length
      : 0;
    
    const slowestService = perfStats.length > 0
      ? perfStats.reduce((slowest: string, entry: [string, ServicePerformanceStats]) => {
          const name = entry[0];
          const stats = entry[1];
          const slowestStats = this.performanceStats.get(slowest);
          return stats.initTime > (slowestStats ? slowestStats.initTime : 0) ? name : slowest;
        }, perfStats[0][0])
      : '';
    
    const fastestService = perfStats.length > 0
      ? perfStats.reduce((fastest: string, entry: [string, ServicePerformanceStats]) => {
          const name = entry[0];
          const stats = entry[1];
          const fastestStats = this.performanceStats.get(fastest);
          return stats.initTime < (fastestStats ? fastestStats.initTime : Infinity) ? name : fastest;
        }, perfStats[0][0])
      : '';
    
    return {
      totalServices,
      initializedServices,
      initializingServices,
      errorServices,
      memoryUsage: {
        singletons: this.singletonInstances.size,
        transients: 0, // 瞬态服务不保存实例
        scoped: Array.from(this.scopedInstances.values()).reduce((sum, scope) => sum + scope.size, 0)
      },
      performance: {
        averageInitTime,
        slowestService,
        fastestService
      }
    };
  }
  
  /**
   * 清理未使用的服务实例
   * @param maxIdleTime 最大空闲时间（毫秒）
   */
  async cleanup(maxIdleTime: number = 30 * 60 * 1000): Promise<void> {
    try {
      const now = Date.now();
      const toCleanup: string[] = [];
      
      // 检查单例服务
      const singletonEntries = Array.from(this.singletonInstances.entries());
      for (const entry of singletonEntries) {
        const serviceName = entry[0];
        const instanceInfo = entry[1];
        if (now - instanceInfo.lastAccessedAt > maxIdleTime) {
          toCleanup.push(serviceName);
        }
      }
      
      // 清理服务
      for (const serviceName of toCleanup) {
        await this.disposeSingletonService(serviceName);
      }
      
      logger.info(`清理了 ${toCleanup.length} 个未使用的服务实例`);
    } catch (error) {
      ErrorHandler.handle(error, 'LazyServiceManager.cleanup', ErrorType.SYSTEM, ErrorLevel.LOW);
    }
  }
  
  /**
   * 销毁所有服务
   */
  async dispose(): Promise<void> {
    try {
      logger.info('开始销毁所有服务');
      
      // 销毁所有作用域
      const scopeIds = Array.from(this.scopedInstances.keys());
      for (const scopeId of scopeIds) {
        await this.disposeScope(scopeId);
      }
      
      // 销毁所有单例服务
      const singletonNames = Array.from(this.singletonInstances.keys());
      for (const serviceName of singletonNames) {
        await this.disposeSingletonService(serviceName);
      }
      
      // 清理数据
      this.serviceConfigs.clear();
      this.singletonInstances.clear();
      this.scopedInstances.clear();
      this.initializingServices.clear();
      this.dependencyGraph.clear();
      this.performanceStats.clear();
      
      logger.info('所有服务已销毁');
    } catch (error) {
      ErrorHandler.handle(error, 'LazyServiceManager.dispose', ErrorType.SYSTEM, ErrorLevel.HIGH);
    }
  }
  
  /**
   * 获取单例服务实例
   */
  private async getSingletonInstance<T extends Object>(serviceName: string, config: ServiceConfig<T>): Promise<T> {
    // 检查是否已有实例
    let instanceInfo = this.singletonInstances.get(serviceName);
    if (instanceInfo) {
      if (instanceInfo.status === ServiceStatus.INITIALIZED) {
        instanceInfo.lastAccessedAt = Date.now();
        instanceInfo.accessCount++;
        return instanceInfo.instance as T;
      } else if (instanceInfo.status === ServiceStatus.ERROR) {
        throw new Error(`服务 ${serviceName} 初始化失败`);
      }
    }
    
    // 检查是否正在初始化
    const initializingPromise = this.initializingServices.get(serviceName);
    if (initializingPromise) {
      return await initializingPromise as T;
    }
    
    // 开始初始化
    const initPromise = this.initializeSingletonService<T>(serviceName, config);
    this.initializingServices.set(serviceName, initPromise as Promise<Object>);
    
    try {
      const instance = await initPromise;
      return instance;
    } finally {
      this.initializingServices.delete(serviceName);
    }
  }
  
  /**
   * 初始化单例服务
   */
  private async initializeSingletonService<T extends Object>(serviceName: string, config: ServiceConfig<T>): Promise<T> {
    const startTime = Date.now();
    
    try {
      // 创建实例信息（初始时为空，后续会设置实际实例）
      const instanceInfo: ServiceInstance<T> = new ServiceInstanceImpl<T>(
        null,
        startTime,
        startTime,
        1,
        ServiceStatus.INITIALIZING
      );
      
      this.singletonInstances.set(serviceName, instanceInfo as ServiceInstance<Object>);
      
      // 初始化依赖
      await this.initializeDependencies(config.dependencies || []);
      
      // 创建服务实例
      const instance = await this.createServiceInstance<T>(config);
      
      // 更新实例信息
      instanceInfo.instance = instance;
      instanceInfo.status = ServiceStatus.INITIALIZED;
      
      // 记录性能统计
      const initTime = Date.now() - startTime;
      this.recordPerformanceStats(serviceName, initTime);
      
      logger.debug(`服务 ${serviceName} 初始化成功，耗时 ${initTime}ms`);
      return instance;
      
    } catch (error) {
      // 更新错误状态
      const instanceInfo = this.singletonInstances.get(serviceName);
      if (instanceInfo) {
        instanceInfo.status = ServiceStatus.ERROR;
      }
      
      logger.error(`服务 ${serviceName} 初始化失败`, error);
      throw new Error(String(error));
    }
  }
  
  /**
   * 创建瞬态服务实例
   */
  private async createTransientInstance<T extends Object>(config: ServiceConfig<T>): Promise<T> {
    // 初始化依赖
    await this.initializeDependencies(config.dependencies || []);
    
    // 创建实例
    return await this.createServiceInstance<T>(config);
  }
  
  /**
   * 获取作用域服务实例
   */
  private async getScopedInstance<T extends Object>(serviceName: string, config: ServiceConfig<T>): Promise<T> {
    const scopeId = this.currentScopeId;
    if (!scopeId) {
      throw new Error(`获取作用域服务 ${serviceName} 时没有设置当前作用域`);
    }
    
    let scopeInstances = this.scopedInstances.get(scopeId);
    if (!scopeInstances) {
      scopeInstances = new Map();
      this.scopedInstances.set(scopeId, scopeInstances);
    }
    
    let instanceInfo = scopeInstances.get(serviceName);
    if (instanceInfo && instanceInfo.status === ServiceStatus.INITIALIZED) {
      instanceInfo.lastAccessedAt = Date.now();
      instanceInfo.accessCount++;
      return instanceInfo.instance as T;
    }
    
    // 创建新的作用域实例
    await this.initializeDependencies(config.dependencies || []);
    const instance = await this.createServiceInstance<T>(config);
    
    instanceInfo = {
      instance,
      createdAt: Date.now(),
      lastAccessedAt: Date.now(),
      accessCount: 1,
      status: ServiceStatus.INITIALIZED,
      scopeId
    };
    
    scopeInstances.set(serviceName, instanceInfo as ServiceInstance<Object>);
    return instance;
  }
  
  /**
   * 创建服务实例
   */
  private async createServiceInstance<T extends Object>(config: ServiceConfig<T>): Promise<T> {
    const timeout = config.timeout || 30000; // 默认30秒超时
    
    return new Promise<T>(async (resolve, reject) => {
      // 设置超时
      const timeoutId = TimerManager.setTimeout(() => {
        reject(new Error(`服务 ${config.name} 初始化超时`));
      }, timeout, `service_init_${config.name}`);
      
      try {
        const instance = await config.factory();
        TimerManager.clearTimeout(timeoutId);
        resolve(instance);
      } catch (error) {
        TimerManager.clearTimeout(timeoutId);
        reject(error);
      }
    });
  }
  
  /**
   * 初始化依赖服务
   */
  private async initializeDependencies(dependencies: string[]): Promise<void> {
    if (dependencies.length === 0) return;
    
    // 检查循环依赖
    this.checkCircularDependencies(dependencies);
    
    // 并行初始化依赖
    const dependencyPromises = dependencies.map((dep: string) => this.get<Object>(dep));
    await Promise.all<Object>(dependencyPromises);
  }
  
  /**
   * 检查循环依赖
   */
  private checkCircularDependencies(dependencies: string[]): void {
    // 简单的循环依赖检测，可以进一步优化
    for (const dep of dependencies) {
      if (this.initializingServices.has(dep)) {
        throw new Error(`检测到循环依赖: ${dep}`);
      }
    }
  }
  
  /**
   * 构建依赖图
   */
  private buildDependencyGraph(config: ServiceConfig<Object>): void {
    if (config.dependencies && config.dependencies.length > 0) {
      this.dependencyGraph.set(config.name, new Set(config.dependencies));
    }
  }
  
  /**
   * 验证服务配置
   */
  private validateServiceConfig(config: ServiceConfig<Object>): void {
    if (!config.name) {
      throw new Error('服务名称不能为空');
    }
    
    if (!config.factory) {
      throw new Error('服务工厂函数不能为空');
    }
    
    if (this.serviceConfigs.has(config.name)) {
      throw new Error(`服务 ${config.name} 已经注册`);
    }
    
    // 验证依赖是否存在
    if (config.dependencies) {
      for (const dep of config.dependencies) {
        if (!this.serviceConfigs.has(dep)) {
          logger.warn(`服务 ${config.name} 依赖的服务 ${dep} 尚未注册`);
        }
      }
    }
  }
  
  /**
   * 记录性能统计
   */
  private recordPerformanceStats(serviceName: string, initTime: number): void {
    const existing = this.performanceStats.get(serviceName);
    if (existing) {
      existing.initTime = (existing.initTime * existing.count + initTime) / (existing.count + 1);
      existing.count++;
    } else {
      this.performanceStats.set(serviceName, { initTime, count: 1 });
    }
  }
  
  /**
   * 销毁单例服务
   */
  private async disposeSingletonService(serviceName: string): Promise<void> {
    const instanceInfo = this.singletonInstances.get(serviceName);
    if (!instanceInfo) return;
    
    try {
      const config = this.serviceConfigs.get(serviceName);
      if (config?.dispose) {
        await config.dispose(instanceInfo.instance);
      }
      
      instanceInfo.status = ServiceStatus.DISPOSED;
      this.singletonInstances.delete(serviceName);
      
      logger.debug(`服务 ${serviceName} 已销毁`);
    } catch (error) {
      logger.error(`销毁服务 ${serviceName} 失败`, error);
    }
  }
}

/**
 * 便捷函数：注册服务
 */
export function registerService<T extends Object>(config: ServiceConfig<T>): void {
  LazyServiceManager.getInstance().register(config);
}

/**
 * 便捷函数：获取服务
 */
export async function getService<T extends Object>(serviceName: string): Promise<T> {
  return LazyServiceManager.getInstance().get<T>(serviceName);
}

/**
 * 便捷函数：检查服务是否已注册
 */
export function isServiceRegistered(serviceName: string): boolean {
  return LazyServiceManager.getInstance().isRegistered(serviceName);
}
