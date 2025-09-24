/**
 * 对象池管理器
 * 减少频繁对象创建和垃圾回收，提升性能
 */

import { createLogger } from './LogManager';
import { ErrorHandler, ErrorType, ErrorLevel } from './ErrorHandler';
import { TimerManager } from './TimerManager';

const logger = createLogger('ObjectPool');

/**
 * 可池化对象接口
 */
export interface IPoolable {
  reset(): void;
  isReusable(): boolean;
}

/**
 * 对象工厂接口
 */
export interface IObjectFactory<T> {
  create(): T;
  reset(obj: T): void;
  validate?(obj: T): boolean;
}

/**
 * 对象池配置
 */
export interface ObjectPoolConfig {
  initialSize: number;
  maxSize: number;
  maxIdleTime: number; // 最大空闲时间（毫秒）
  enableValidation: boolean;
  enableMetrics: boolean;
}

/**
 * 对象池统计信息
 */
export interface ObjectPoolStats {
  totalCreated: number;
  totalAcquired: number;
  totalReleased: number;
  currentPoolSize: number;
  currentActiveObjects: number;
  hitRate: number;
  averageAcquisitionTime: number;
}

/**
 * 池化对象信息
 */
interface PooledObjectInfo<T> {
  object: T;
  createdAt: number;
  lastUsed: number;
  useCount: number;
}

/**
 * 通用对象池
 */
export class ObjectPool<T> {
  private pool: PooledObjectInfo<T>[] = [];
  private activeObjects: Set<T> = new Set();
  private factory: IObjectFactory<T>;
  private config: ObjectPoolConfig;
  private stats: ObjectPoolStats;
  private cleanupTimerId: number = -1;

  constructor(factory: IObjectFactory<T>, config?: Partial<ObjectPoolConfig>) {
    this.factory = factory;
    this.config = {
      initialSize: 5,
      maxSize: 50,
      maxIdleTime: 30 * 60 * 1000, // 30分钟
      enableValidation: true,
      enableMetrics: true,
      ...config
    };

    this.stats = {
      totalCreated: 0,
      totalAcquired: 0,
      totalReleased: 0,
      currentPoolSize: 0,
      currentActiveObjects: 0,
      hitRate: 0,
      averageAcquisitionTime: 0
    };

    this.initialize();
  }

  /**
   * 初始化对象池
   */
  private initialize(): void {
    try {
      // 预创建初始对象
      for (let i = 0; i < this.config.initialSize; i++) {
        const obj = this.createObject();
        this.pool.push({
          object: obj,
          createdAt: Date.now(),
          lastUsed: Date.now(),
          useCount: 0
        });
      }

      this.updateStats();

      // 启动清理任务
      this.startCleanupTask();

      logger.debug(`对象池初始化完成，预创建 ${this.config.initialSize} 个对象`);
    } catch (error) {
      ErrorHandler.handle(error, 'ObjectPool.initialize', ErrorType.SYSTEM, ErrorLevel.HIGH);
    }
  }

  /**
   * 获取对象
   */
  acquire(): T {
    const startTime = performance.now();

    try {
      let pooledInfo = this.pool.pop();
      let obj: T;

      if (pooledInfo) {
        // 从池中获取对象
        obj = pooledInfo.object;
        
        // 验证对象是否可用
        if (this.config.enableValidation && this.factory.validate) {
          if (!this.factory.validate(obj)) {
            logger.warn('池中对象验证失败，创建新对象');
            obj = this.createObject();
          }
        }

        // 重置对象状态
        this.factory.reset(obj);
        pooledInfo.lastUsed = Date.now();
        pooledInfo.useCount++;
      } else {
        // 池中无可用对象，创建新对象
        obj = this.createObject();
      }

      this.activeObjects.add(obj);
      this.stats.totalAcquired++;
      this.updateStats();

      // 更新平均获取时间
      if (this.config.enableMetrics) {
        const acquisitionTime = performance.now() - startTime;
        this.updateAverageAcquisitionTime(acquisitionTime);
      }

      return obj;
    } catch (error) {
      ErrorHandler.handle(error, 'ObjectPool.acquire', ErrorType.SYSTEM, ErrorLevel.MEDIUM);
      // 发生错误时创建新对象
      return this.createObject();
    }
  }

  /**
   * 释放对象回池中
   */
  release(obj: T): void {
    try {
      if (!this.activeObjects.has(obj)) {
        logger.warn('尝试释放不属于此池的对象');
        return;
      }

      this.activeObjects.delete(obj);

      // 检查对象是否可重用
      if (obj && typeof obj === 'object' && 'isReusable' in obj) {
        const poolable = obj as unknown as IPoolable;
        if (!poolable.isReusable()) {
          logger.debug('对象不可重用，直接丢弃');
          this.stats.totalReleased++;
          this.updateStats();
          return;
        }
      }

      // 如果池未满，将对象放回池中
      if (this.pool.length < this.config.maxSize) {
        this.pool.push({
          object: obj,
          createdAt: Date.now(),
          lastUsed: Date.now(),
          useCount: 0
        });
      }

      this.stats.totalReleased++;
      this.updateStats();
    } catch (error) {
      ErrorHandler.handle(error, 'ObjectPool.release', ErrorType.SYSTEM, ErrorLevel.LOW);
    }
  }

  /**
   * 获取统计信息
   */
  getStats(): ObjectPoolStats {
    return { ...this.stats };
  }

  /**
   * 清空对象池
   */
  clear(): void {
    try {
      this.pool = [];
      this.activeObjects.clear();
      this.updateStats();
      logger.debug('对象池已清空');
    } catch (error) {
      ErrorHandler.handle(error, 'ObjectPool.clear', ErrorType.SYSTEM, ErrorLevel.LOW);
    }
  }

  /**
   * 销毁对象池
   */
  destroy(): void {
    try {
      this.stopCleanupTask();
      this.clear();
      logger.debug('对象池已销毁');
    } catch (error) {
      ErrorHandler.handle(error, 'ObjectPool.destroy', ErrorType.SYSTEM, ErrorLevel.LOW);
    }
  }

  /**
   * 预热对象池
   */
  warmup(count: number): void {
    try {
      const objectsToCreate = Math.min(count, this.config.maxSize - this.pool.length);
      
      for (let i = 0; i < objectsToCreate; i++) {
        const obj = this.createObject();
        this.pool.push({
          object: obj,
          createdAt: Date.now(),
          lastUsed: Date.now(),
          useCount: 0
        });
      }

      this.updateStats();
      logger.debug(`对象池预热完成，创建 ${objectsToCreate} 个对象`);
    } catch (error) {
      ErrorHandler.handle(error, 'ObjectPool.warmup', ErrorType.SYSTEM, ErrorLevel.LOW);
    }
  }

  /**
   * 创建新对象
   */
  private createObject(): T {
    const obj = this.factory.create();
    this.stats.totalCreated++;
    return obj;
  }

  /**
   * 更新统计信息
   */
  private updateStats(): void {
    this.stats.currentPoolSize = this.pool.length;
    this.stats.currentActiveObjects = this.activeObjects.size;
    
    // 计算命中率
    if (this.stats.totalAcquired > 0) {
      const hits = this.stats.totalAcquired - this.stats.totalCreated;
      this.stats.hitRate = (hits / this.stats.totalAcquired) * 100;
    }
  }

  /**
   * 更新平均获取时间
   */
  private updateAverageAcquisitionTime(acquisitionTime: number): void {
    if (this.stats.averageAcquisitionTime === 0) {
      this.stats.averageAcquisitionTime = acquisitionTime;
    } else {
      // 使用移动平均
      this.stats.averageAcquisitionTime = 
        (this.stats.averageAcquisitionTime * 0.9) + (acquisitionTime * 0.1);
    }
  }

  /**
   * 启动清理任务
   */
  private startCleanupTask(): void {
    this.cleanupTimerId = TimerManager.setInterval(() => {
      this.performCleanup();
    }, 5 * 60 * 1000, '对象池清理'); // 5分钟清理一次
  }

  /**
   * 停止清理任务
   */
  private stopCleanupTask(): void {
    if (this.cleanupTimerId !== -1) {
      TimerManager.clearInterval(this.cleanupTimerId);
      this.cleanupTimerId = -1;
    }
  }

  /**
   * 执行清理
   */
  private performCleanup(): void {
    try {
      const now = Date.now();
      const initialSize = this.pool.length;
      
      // 清理长时间未使用的对象
      this.pool = this.pool.filter(info => {
        const idleTime = now - info.lastUsed;
        return idleTime < this.config.maxIdleTime;
      });

      const cleanedCount = initialSize - this.pool.length;
      if (cleanedCount > 0) {
        logger.debug(`清理了 ${cleanedCount} 个长时间未使用的对象`);
        this.updateStats();
      }

      // 记录池状态
      if (this.config.enableMetrics) {
        logger.debug(`对象池状态: 池大小=${this.pool.length}, 活跃对象=${this.activeObjects.size}, 命中率=${this.stats.hitRate.toFixed(1)}%`);
      }
    } catch (error) {
      ErrorHandler.handle(error, 'ObjectPool.performCleanup', ErrorType.SYSTEM, ErrorLevel.LOW);
    }
  }
}

/**
 * 对象池管理器
 */
export class ObjectPoolManager {
  private static pools: Map<string, ObjectPool<any>> = new Map();

  /**
   * 注册对象池
   */
  static registerPool<T>(name: string, pool: ObjectPool<T>): void {
    ObjectPoolManager.pools.set(name, pool);
    logger.debug(`注册对象池: ${name}`);
  }

  /**
   * 获取对象池
   */
  static getPool<T>(name: string): ObjectPool<T> | undefined {
    return ObjectPoolManager.pools.get(name);
  }

  /**
   * 移除对象池
   */
  static removePool(name: string): boolean {
    const pool = ObjectPoolManager.pools.get(name);
    if (pool) {
      pool.destroy();
      ObjectPoolManager.pools.delete(name);
      logger.debug(`移除对象池: ${name}`);
      return true;
    }
    return false;
  }

  /**
   * 获取所有池的统计信息
   */
  static getAllStats(): Record<string, ObjectPoolStats> {
    const stats: Record<string, ObjectPoolStats> = {};
    for (const [name, pool] of ObjectPoolManager.pools.entries()) {
      stats[name] = pool.getStats();
    }
    return stats;
  }

  /**
   * 清理所有对象池
   */
  static cleanup(): void {
    logger.info('开始清理所有对象池...');
    for (const [name, pool] of ObjectPoolManager.pools.entries()) {
      try {
        pool.destroy();
        logger.debug(`清理对象池: ${name}`);
      } catch (error) {
        logger.error(`清理对象池失败: ${name} - ${error}`);
      }
    }
    ObjectPoolManager.pools.clear();
    logger.info('所有对象池清理完成');
  }
}

/**
 * 常用对象工厂
 */
export class CommonObjectFactories {
  /**
   * 通用对象工厂
   */
  static createGenericFactory<T>(
    createFn: () => T,
    resetFn: (obj: T) => void,
    validateFn?: (obj: T) => boolean
  ): IObjectFactory<T> {
    return {
      create: createFn,
      reset: resetFn,
      validate: validateFn
    };
  }

  /**
   * Map对象工厂
   */
  static createMapFactory<K, V>(): IObjectFactory<Map<K, V>> {
    return {
      create: () => new Map<K, V>(),
      reset: (map: Map<K, V>) => map.clear(),
      validate: (map: Map<K, V>) => map instanceof Map
    };
  }

  /**
   * Set对象工厂
   */
  static createSetFactory<T>(): IObjectFactory<Set<T>> {
    return {
      create: () => new Set<T>(),
      reset: (set: Set<T>) => set.clear(),
      validate: (set: Set<T>) => set instanceof Set
    };
  }

  /**
   * Array对象工厂
   */
  static createArrayFactory<T>(): IObjectFactory<T[]> {
    return {
      create: () => [],
      reset: (arr: T[]) => arr.length = 0,
      validate: (arr: T[]) => Array.isArray(arr)
    };
  }
}

/**
 * 便捷函数：创建并注册对象池
 */
export function createObjectPool<T>(
  name: string,
  factory: IObjectFactory<T>,
  config?: Partial<ObjectPoolConfig>
): ObjectPool<T> {
  const pool = new ObjectPool(factory, config);
  ObjectPoolManager.registerPool(name, pool);
  return pool;
}
