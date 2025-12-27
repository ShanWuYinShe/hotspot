/**
 * 基础单例模式实现
 * 提供统一的单例模式基类，避免重复实现
 * 重构以符合ArkTS严格模式要求
 */

import { createLogger } from '../utils/LogManager';

const logger = createLogger('Singleton');

/**
 * 单例接口
 */
export interface ISingleton {
  /**
   * 初始化方法（可选）
   */
  initialize?(): void | Promise<void>;

  /**
   * 清理方法（可选）
   */
  cleanup?(): void | Promise<void>;
}

/**
 * 单例状态信息
 */
interface SingletonInfo {
  instance: ISingleton;
  isInitialized: boolean;
  isInitializing: boolean;
  initPromise?: Promise<void>;
}

/**
 * 单例统计信息
 */
export interface SingletonStats {
  totalInstances: number;
  instanceNames: string[];
  initializingInstances: string[];
}

/**
 * 单例管理器
 * 统一管理所有单例实例
 */
export class SingletonManager {
  private static instances: Map<string, SingletonInfo> = new Map();

  /**
   * 注册单例实例
   */
  static registerInstance(className: string, instance: ISingleton): void {
    if (!SingletonManager.instances.has(className)) {
      logger.debug(`注册单例实例: ${className}`);
      SingletonManager.instances.set(className, {
        instance: instance,
        isInitialized: false,
        isInitializing: false
      });
    }
  }

  /**
   * 获取单例实例
   */
  static getInstance(className: string): ISingleton | null {
    const info = SingletonManager.instances.get(className);
    return info ? info.instance : null;
  }

  /**
   * 异步获取单例实例（确保初始化完成）
   */
  static async getInstanceAsync(className: string): Promise<ISingleton | null> {
    const info = SingletonManager.instances.get(className);
    if (!info) {
      return null;
    }

    // 如果有初始化方法，确保初始化完成
    if (info.instance.initialize && !info.isInitialized) {
      if (!info.initPromise) {
        info.initPromise = SingletonManager.initializeInstance(className);
      }
      await info.initPromise;
    }

    return info.instance;
  }

  /**
   * 初始化实例
   */
  private static async initializeInstance(className: string): Promise<void> {
    const info = SingletonManager.instances.get(className);
    if (!info || info.isInitialized || info.isInitializing) {
      return;
    }

    info.isInitializing = true;
    try {
      logger.debug(`初始化单例实例: ${className}`);
      if (info.instance.initialize) {
        await info.instance.initialize();
      }
      info.isInitialized = true;
      logger.debug(`单例实例初始化完成: ${className}`);
    } catch (error) {
      logger.error(`单例实例初始化失败: ${className}`, error);
      throw new Error(`单例实例初始化失败: ${className}`);
    } finally {
      info.isInitializing = false;
    }
  }

  /**
   * 检查实例是否存在
   */
  static hasInstance(className: string): boolean {
    return SingletonManager.instances.has(className);
  }

  /**
   * 销毁单例实例
   */
  static async destroyInstance(className: string): Promise<void> {
    const info = SingletonManager.instances.get(className);
    if (!info) {
      return;
    }

    try {
      logger.debug(`销毁单例实例: ${className}`);
      if (info.instance.cleanup) {
        await info.instance.cleanup();
      }
      SingletonManager.instances.delete(className);
      logger.debug(`单例实例销毁完成: ${className}`);
    } catch (error) {
      logger.error(`单例实例销毁失败: ${className}`, error);
      throw new Error(`单例实例销毁失败: ${className}`);
    }
  }

  /**
   * 销毁所有单例实例
   */
  static async destroyAll(): Promise<void> {
    logger.info('开始销毁所有单例实例');
    const destroyPromises: Promise<void>[] = [];

    const classNames = Array.from(SingletonManager.instances.keys());
    for (let i = 0; i < classNames.length; i++) {
      destroyPromises.push(SingletonManager.destroyInstance(classNames[i]));
    }

    try {
      await Promise.all(destroyPromises);
      logger.info('所有单例实例销毁完成');
    } catch (error) {
      logger.error('销毁单例实例时出错', error);
      throw new Error('销毁单例实例时出错');
    }
  }

  /**
   * 获取所有实例信息
   */
  static getAllInstances(): string[] {
    return Array.from(SingletonManager.instances.keys());
  }

  /**
   * 获取实例统计信息
   */
  static getStats(): SingletonStats {
    const initializingInstances: string[] = [];
    const entries = Array.from(SingletonManager.instances.entries());
    
    for (let i = 0; i < entries.length; i++) {
      const entry = entries[i];
      const className = entry[0];
      const info = entry[1];
      if (info.isInitializing) {
        initializingInstances.push(className);
      }
    }

    return {
      totalInstances: SingletonManager.instances.size,
      instanceNames: Array.from(SingletonManager.instances.keys()),
      initializingInstances: initializingInstances
    };
  }
}

/**
 * 抽象单例基类
 * 提供标准的单例实现模板
 */
export abstract class BaseSingleton implements ISingleton {
  private _isInitialized: boolean = false;
  private _isInitializing: boolean = false;
  private _initPromise?: Promise<void>;

  protected constructor() {
    // 自动注册到单例管理器
    const className = this.constructor.name;
    SingletonManager.registerInstance(className, this);
  }

  /**
   * 获取单例实例的静态方法
   * 子类需要重写这个方法
   */
  public static getInstance(): BaseSingleton {
    throw new Error('子类必须实现 getInstance 静态方法');
  }

  /**
   * 异步获取单例实例
   */
  public static async getInstanceAsync(): Promise<BaseSingleton> {
    throw new Error('子类必须实现 getInstanceAsync 静态方法');
  }

  /**
   * 检查是否已初始化
   */
  get isInitialized(): boolean {
    return this._isInitialized;
  }

  /**
   * 检查是否正在初始化
   */
  get isInitializing(): boolean {
    return this._isInitializing;
  }

  /**
   * 初始化方法（子类可以重写）
   */
  async initialize(): Promise<void> {
    if (this._isInitialized || this._isInitializing) {
      if (this._initPromise) {
        await this._initPromise;
      }
      return;
    }

    this._isInitializing = true;
    this._initPromise = this.performInitialization();
    
    try {
      await this._initPromise;
    } finally {
      this._isInitializing = false;
    }
  }

  /**
   * 执行初始化
   */
  private async performInitialization(): Promise<void> {
    try {
      await this.onInitialize();
      this._isInitialized = true;
      logger.debug(`${this.constructor.name} 初始化完成`);
    } catch (error) {
      logger.error(`${this.constructor.name} 初始化失败`, error);
      throw new Error(`${this.constructor.name} 初始化失败`);
    }
  }

  /**
   * 清理方法（子类可以重写）
   */
  async cleanup(): Promise<void> {
    if (!this._isInitialized) {
      return;
    }

    try {
      await this.onCleanup();
      this._isInitialized = false;
      logger.debug(`${this.constructor.name} 清理完成`);
    } catch (error) {
      logger.error(`${this.constructor.name} 清理失败`, error);
      throw new Error(`${this.constructor.name} 清理失败`);
    }
  }

  /**
   * 子类实现的初始化逻辑
   */
  protected abstract onInitialize(): Promise<void> | void;

  /**
   * 子类实现的清理逻辑
   */
  protected abstract onCleanup(): Promise<void> | void;

  /**
   * 确保已初始化的辅助方法
   */
  protected ensureInitialized(): void {
    if (!this._isInitialized) {
      throw new Error(`${this.constructor.name} 尚未初始化`);
    }
  }
}


/**
 * 便捷函数：获取单例实例
 */
export function getSingleton(className: string): ISingleton | null {
  return SingletonManager.getInstance(className);
}

/**
 * 便捷函数：异步获取单例实例
 */
export async function getSingletonAsync(className: string): Promise<ISingleton | null> {
  return SingletonManager.getInstanceAsync(className);
}