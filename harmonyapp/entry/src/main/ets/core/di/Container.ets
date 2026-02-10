/**
 * 简单的依赖注入容器
 * 提供服务注册、解析和生命周期管理
 */

import { BaseSingleton } from '../base/Singleton';
import { createLogger } from '../utils/LogManager';
import { ErrorHandler, ErrorType, ErrorLevel } from '../utils/ErrorHandler';

const logger = createLogger('DIContainer');

/**
 * 服务生命周期枚举
 */
export enum ServiceLifetime {
  SINGLETON = 'singleton',    // 单例
  TRANSIENT = 'transient',    // 每次创建新实例
  SCOPED = 'scoped'           // 作用域内单例
}

/**
 * 服务工厂函数类型
 */
export type ServiceFactory<T = any> = (container: Container) => T | Promise<T>;

/**
 * 服务构造函数类型
 */
export type ServiceConstructor<T = any> = new(...args: any[]) => T;

/**
 * 服务注册信息
 */
interface ServiceRegistration<T = any> {
  key: string;
  factory?: ServiceFactory<T>;
  constructor?: ServiceConstructor<T>;
  lifetime: ServiceLifetime;
  instance?: T;
  dependencies?: string[];
  metadata?: Record<string, any>;
  registeredAt: number;
}

/**
 * 依赖解析上下文
 */
interface ResolutionContext {
  resolving: Set<string>;
  scope: Map<string, any>;
}

/**
 * 容器统计信息
 */
export interface ContainerStats {
  totalServices: number;
  singletonServices: number;
  transientServices: number;
  scopedServices: number;
  resolvedCount: number;
  errorCount: number;
}

/**
 * 依赖注入容器
 */
export class Container extends BaseSingleton {
  private static instance: Container | null = null;
  private services: Map<string, ServiceRegistration> = new Map();
  private stats: ContainerStats;
  private globalScope: Map<string, any> = new Map();

  constructor() {
    super();
    this.stats = {
      totalServices: 0,
      singletonServices: 0,
      transientServices: 0,
      scopedServices: 0,
      resolvedCount: 0,
      errorCount: 0
    };
  }

  /**
   * 获取单例实例
   */
  static getInstance(): Container {
    if (!Container.instance) {
      Container.instance = new Container();
    }
    return Container.instance;
  }

  /**
   * 异步获取单例实例
   */
  static async getInstanceAsync(): Promise<Container> {
    const instance = Container.getInstance();
    await instance.initialize();
    return instance;
  }

  /**
   * 实现基类的初始化方法
   */
  protected async onInitialize(): Promise<void> {
    try {
      logger.info('依赖注入容器初始化开始');
      
      // 注册容器自身
      this.registerInstance('Container', this);
      
      logger.info('依赖注入容器初始化完成');
    } catch (error) {
      ErrorHandler.handle(error, 'Container.onInitialize', ErrorType.SYSTEM, ErrorLevel.HIGH);
      throw error;
    }
  }

  /**
   * 实现基类的清理方法
   */
  protected async onCleanup(): Promise<void> {
    try {
      // 清理所有单例实例
      await this.disposeAllSingletons();
      
      // 清理注册信息
      this.services.clear();
      this.globalScope.clear();
      Container.instance = null;
      
      logger.info('依赖注入容器清理完成');
    } catch (error) {
      ErrorHandler.handle(error, 'Container.onCleanup', ErrorType.SYSTEM);
    }
  }

  /**
   * 注册服务工厂
   */
  register<T>(
    key: string,
    factory: ServiceFactory<T>,
    lifetime: ServiceLifetime = ServiceLifetime.SINGLETON,
    dependencies?: string[]
  ): void {
    this.registerService({
      key,
      factory,
      lifetime,
      dependencies,
      registeredAt: Date.now()
    });
  }

  /**
   * 注册服务构造函数
   */
  registerType<T>(
    key: string,
    constructor: ServiceConstructor<T>,
    lifetime: ServiceLifetime = ServiceLifetime.SINGLETON,
    dependencies?: string[]
  ): void {
    this.registerService({
      key,
      constructor,
      lifetime,
      dependencies,
      registeredAt: Date.now()
    });
  }

  /**
   * 注册服务实例
   */
  registerInstance<T>(key: string, instance: T): void {
    this.registerService({
      key,
      lifetime: ServiceLifetime.SINGLETON,
      instance,
      registeredAt: Date.now()
    });
  }

  /**
   * 解析服务
   */
  resolve<T>(key: string): T {
    const context: ResolutionContext = {
      resolving: new Set(),
      scope: new Map()
    };
    
    return this.resolveWithContext<T>(key, context);
  }

  /**
   * 异步解析服务
   */
  async resolveAsync<T>(key: string): Promise<T> {
    const context: ResolutionContext = {
      resolving: new Set(),
      scope: new Map()
    };
    
    return this.resolveWithContextAsync<T>(key, context);
  }

  /**
   * 尝试解析服务（不抛出异常）
   */
  tryResolve<T>(key: string): T | null {
    try {
      return this.resolve<T>(key);
    } catch (error) {
      ErrorHandler.handle(error, `tryResolve(${key})`, ErrorType.SYSTEM, ErrorLevel.LOW);
      return null;
    }
  }

  /**
   * 检查服务是否已注册
   */
  isRegistered(key: string): boolean {
    return this.services.has(key);
  }

  /**
   * 获取已注册的服务键列表
   */
  getRegisteredKeys(): string[] {
    return Array.from(this.services.keys());
  }

  /**
   * 取消注册服务
   */
  unregister(key: string): boolean {
    const registration = this.services.get(key);
    if (!registration) {
      return false;
    }

    // 如果是单例且已实例化，尝试清理
    if (registration.lifetime === ServiceLifetime.SINGLETON && registration.instance) {
      this.disposeInstance(registration.instance);
    }

    this.services.delete(key);
    this.updateStatsForUnregister(registration.lifetime);
    
    logger.debug(`取消注册服务: ${key}`);
    return true;
  }

  /**
   * 创建子容器（作用域）
   */
  createScope(): ScopedContainer {
    return new ScopedContainer(this);
  }

  /**
   * 获取容器统计信息
   */
  getStats(): ContainerStats {
    return { ...this.stats };
  }

  /**
   * 获取服务信息
   */
  getServiceInfo(key: string): ServiceRegistration | null {
    const registration = this.services.get(key);
    return registration ? { ...registration } : null;
  }

  /**
   * 验证依赖关系
   */
  validateDependencies(): string[] {
    const errors: string[] = [];
    
    for (const [key, registration] of this.services.entries()) {
      if (registration.dependencies) {
        for (const dep of registration.dependencies) {
          if (!this.services.has(dep)) {
            errors.push(`服务 '${key}' 依赖未注册的服务 '${dep}'`);
          }
        }
      }
    }
    
    return errors;
  }

  /**
   * 检测循环依赖
   */
  detectCircularDependencies(): string[] {
    const errors: string[] = [];
    const visited = new Set<string>();
    const visiting = new Set<string>();

    const visit = (key: string, path: string[]): void => {
      if (visiting.has(key)) {
        const cycle = [...path, key].join(' -> ');
        errors.push(`检测到循环依赖: ${cycle}`);
        return;
      }

      if (visited.has(key)) {
        return;
      }

      visiting.add(key);
      const registration = this.services.get(key);
      
      if (registration?.dependencies) {
        for (const dep of registration.dependencies) {
          visit(dep, [...path, key]);
        }
      }

      visiting.delete(key);
      visited.add(key);
    };

    for (const key of this.services.keys()) {
      if (!visited.has(key)) {
        visit(key, []);
      }
    }

    return errors;
  }

  /**
   * 注册服务的内部方法
   */
  private registerService<T>(registration: ServiceRegistration<T>): void {
    // 检查是否已注册
    if (this.services.has(registration.key)) {
      logger.warn(`服务已存在，将被覆盖: ${registration.key}`);
      this.unregister(registration.key);
    }

    this.services.set(registration.key, registration);
    this.updateStatsForRegister(registration.lifetime);
    
    logger.debug(`注册服务: ${registration.key} (${registration.lifetime})`);
  }

  /**
   * 带上下文的服务解析
   */
  private resolveWithContext<T>(key: string, context: ResolutionContext): T {
    // 检查循环依赖
    if (context.resolving.has(key)) {
      throw new Error(`检测到循环依赖: ${Array.from(context.resolving).join(' -> ')} -> ${key}`);
    }

    const registration = this.services.get(key);
    if (!registration) {
      throw new Error(`服务未注册: ${key}`);
    }

    try {
      context.resolving.add(key);
      const instance = this.createInstance<T>(registration, context);
      this.stats.resolvedCount++;
      return instance;
    } catch (error) {
      this.stats.errorCount++;
      throw error;
    } finally {
      context.resolving.delete(key);
    }
  }

  /**
   * 异步服务解析
   */
  private async resolveWithContextAsync<T>(key: string, context: ResolutionContext): Promise<T> {
    // 检查循环依赖
    if (context.resolving.has(key)) {
      throw new Error(`检测到循环依赖: ${Array.from(context.resolving).join(' -> ')} -> ${key}`);
    }

    const registration = this.services.get(key);
    if (!registration) {
      throw new Error(`服务未注册: ${key}`);
    }

    try {
      context.resolving.add(key);
      const instance = await this.createInstanceAsync<T>(registration, context);
      this.stats.resolvedCount++;
      return instance;
    } catch (error) {
      this.stats.errorCount++;
      throw error;
    } finally {
      context.resolving.delete(key);
    }
  }

  /**
   * 创建服务实例
   */
  private createInstance<T>(registration: ServiceRegistration<T>, context: ResolutionContext): T {
    // 如果已有实例（单例）
    if (registration.instance) {
      return registration.instance;
    }

    // 检查作用域缓存
    if (registration.lifetime === ServiceLifetime.SCOPED) {
      const scopedInstance = context.scope.get(registration.key);
      if (scopedInstance) {
        return scopedInstance;
      }
    }

    let instance: T;

    if (registration.factory) {
      // 使用工厂函数创建
      instance = registration.factory(this);
    } else if (registration.constructor) {
      // 使用构造函数创建
      const dependencies = this.resolveDependencies(registration.dependencies || [], context);
      instance = new registration.constructor(...dependencies);
    } else {
      throw new Error(`服务 '${registration.key}' 没有有效的创建方式`);
    }

    // 根据生命周期处理实例
    this.handleInstanceLifetime(registration, instance, context);

    return instance;
  }

  /**
   * 异步创建服务实例
   */
  private async createInstanceAsync<T>(registration: ServiceRegistration<T>, context: ResolutionContext): Promise<T> {
    // 如果已有实例（单例）
    if (registration.instance) {
      return registration.instance;
    }

    // 检查作用域缓存
    if (registration.lifetime === ServiceLifetime.SCOPED) {
      const scopedInstance = context.scope.get(registration.key);
      if (scopedInstance) {
        return scopedInstance;
      }
    }

    let instance: T;

    if (registration.factory) {
      // 使用工厂函数创建
      instance = await Promise.resolve(registration.factory(this));
    } else if (registration.constructor) {
      // 使用构造函数创建
      const dependencies = await this.resolveDependenciesAsync(registration.dependencies || [], context);
      instance = new registration.constructor(...dependencies);
    } else {
      throw new Error(`服务 '${registration.key}' 没有有效的创建方式`);
    }

    // 根据生命周期处理实例
    this.handleInstanceLifetime(registration, instance, context);

    return instance;
  }

  /**
   * 解析依赖项
   */
  private resolveDependencies(dependencies: string[], context: ResolutionContext): any[] {
    return dependencies.map(dep => this.resolveWithContext(dep, context));
  }

  /**
   * 异步解析依赖项
   */
  private async resolveDependenciesAsync(dependencies: string[], context: ResolutionContext): Promise<any[]> {
    const promises = dependencies.map(dep => this.resolveWithContextAsync(dep, context));
    return Promise.all(promises);
  }

  /**
   * 处理实例生命周期
   */
  private handleInstanceLifetime<T>(
    registration: ServiceRegistration<T>, 
    instance: T, 
    context: ResolutionContext
  ): void {
    switch (registration.lifetime) {
      case ServiceLifetime.SINGLETON:
        registration.instance = instance;
        this.globalScope.set(registration.key, instance);
        break;
      case ServiceLifetime.SCOPED:
        context.scope.set(registration.key, instance);
        break;
      case ServiceLifetime.TRANSIENT:
        // 不缓存
        break;
    }
  }

  /**
   * 清理实例
   */
  private disposeInstance(instance: any): void {
    try {
      if (instance && typeof instance.dispose === 'function') {
        instance.dispose();
      } else if (instance && typeof instance.cleanup === 'function') {
        instance.cleanup();
      }
    } catch (error) {
      ErrorHandler.handle(error, 'disposeInstance', ErrorType.SYSTEM, ErrorLevel.LOW);
    }
  }

  /**
   * 清理所有单例实例
   */
  private async disposeAllSingletons(): Promise<void> {
    for (const instance of this.globalScope.values()) {
      this.disposeInstance(instance);
    }
    this.globalScope.clear();
  }

  /**
   * 更新注册统计
   */
  private updateStatsForRegister(lifetime: ServiceLifetime): void {
    this.stats.totalServices++;
    switch (lifetime) {
      case ServiceLifetime.SINGLETON:
        this.stats.singletonServices++;
        break;
      case ServiceLifetime.TRANSIENT:
        this.stats.transientServices++;
        break;
      case ServiceLifetime.SCOPED:
        this.stats.scopedServices++;
        break;
    }
  }

  /**
   * 更新取消注册统计
   */
  private updateStatsForUnregister(lifetime: ServiceLifetime): void {
    this.stats.totalServices--;
    switch (lifetime) {
      case ServiceLifetime.SINGLETON:
        this.stats.singletonServices--;
        break;
      case ServiceLifetime.TRANSIENT:
        this.stats.transientServices--;
        break;
      case ServiceLifetime.SCOPED:
        this.stats.scopedServices--;
        break;
    }
  }
}

/**
 * 作用域容器
 */
export class ScopedContainer {
  private scope: Map<string, any> = new Map();

  constructor(private parentContainer: Container) {}

  /**
   * 解析服务（作用域内）
   */
  resolve<T>(key: string): T {
    const context: ResolutionContext = {
      resolving: new Set(),
      scope: this.scope
    };
    
    return this.parentContainer['resolveWithContext']<T>(key, context);
  }

  /**
   * 异步解析服务（作用域内）
   */
  async resolveAsync<T>(key: string): Promise<T> {
    const context: ResolutionContext = {
      resolving: new Set(),
      scope: this.scope
    };
    
    return this.parentContainer['resolveWithContextAsync']<T>(key, context);
  }

  /**
   * 清理作用域
   */
  dispose(): void {
    for (const instance of this.scope.values()) {
      this.parentContainer['disposeInstance'](instance);
    }
    this.scope.clear();
  }
}

/**
 * 便捷函数：获取容器实例
 */
export function getContainer(): Container {
  return Container.getInstance();
}

/**
 * 便捷函数：注册服务
 */
export function register<T>(
  key: string,
  factory: ServiceFactory<T>,
  lifetime?: ServiceLifetime,
  dependencies?: string[]
): void {
  getContainer().register(key, factory, lifetime, dependencies);
}

/**
 * 便捷函数：注册类型
 */
export function registerType<T>(
  key: string,
  constructor: ServiceConstructor<T>,
  lifetime?: ServiceLifetime,
  dependencies?: string[]
): void {
  getContainer().registerType(key, constructor, lifetime, dependencies);
}

/**
 * 便捷函数：注册实例
 */
export function registerInstance<T>(key: string, instance: T): void {
  getContainer().registerInstance(key, instance);
}

/**
 * 便捷函数：解析服务
 */
export function resolve<T>(key: string): T {
  return getContainer().resolve<T>(key);
}

/**
 * 便捷函数：异步解析服务
 */
export function resolveAsync<T>(key: string): Promise<T> {
  return getContainer().resolveAsync<T>(key);
}
