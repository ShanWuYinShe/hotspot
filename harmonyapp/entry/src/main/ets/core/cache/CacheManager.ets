/**
 * 缓存管理器
 * 简化版本，专注于核心缓存功能
 */

import { BaseSingleton } from '../base/Singleton';
import { createLogger } from '../utils/LogManager';
import { TimerManager } from '../utils/TimerManager';
import { ErrorHandler, ErrorType, ErrorLevel } from '../utils/ErrorHandler';
import { PERFORMANCE_CONSTANTS } from '../../constants/SystemConstants';

const logger = createLogger('CacheManager');

/**
 * 缓存项接口
 */
interface CacheItem<T> {
  data: T;
  timestamp: number;
  expiry: number;
  accessCount: number;
  lastAccess: number;
}

/**
 * 缓存统计信息
 */
export interface CacheStats {
  size: number;
  maxSize: number;
  hitRate: number;
  totalHits: number;
  totalMisses: number;
  totalRequests: number;
}

/**
 * 内部缓存统计信息
 */
interface CacheStatsInternal {
  hits: number;
  misses: number;
  cleanupTimerId: number;
}

/**
 * 缓存项信息接口
 */
interface CacheItemInfo {
  exists: boolean;
  accessCount?: number;
  lastAccess?: number;
  expiry?: number;
}

/**
 * 缓存配置
 */
export interface CacheConfig {
  maxSize: number;
  defaultExpiry: number;
  cleanupInterval: number;
  enableStats: boolean;
}

/**
 * 缓存管理器
 */
export class CacheManager extends BaseSingleton {
  private static instance: CacheManager | null = null;
  private cache: Map<string, CacheItem<Object>> = new Map();
  private config: CacheConfig;
  private stats: CacheStatsInternal = {
    hits: 0,
    misses: 0,
    cleanupTimerId: -1
  };

  private constructor() {
    super();
    this.config = {
      maxSize: PERFORMANCE_CONSTANTS.CACHE.DEFAULT_SIZE,
      defaultExpiry: PERFORMANCE_CONSTANTS.CACHE.DEFAULT_EXPIRY,
      cleanupInterval: 5 * 60 * 1000, // 5分钟清理一次
      enableStats: true
    };
  }

  /**
   * 获取单例实例
   */
  static getInstance(): CacheManager {
    if (!CacheManager.instance) {
      CacheManager.instance = new CacheManager();
    }
    return CacheManager.instance;
  }

  /**
   * 异步获取单例实例
   */
  static async getInstanceAsync(): Promise<CacheManager> {
    const instance = CacheManager.getInstance();
    await instance.initialize();
    return instance;
  }

  /**
   * 初始化缓存管理器
   */
  protected async onInitialize(): Promise<void> {
    logger.info('缓存管理器初始化开始');
    this.startCleanupTimer();
    logger.info('缓存管理器初始化完成');
  }

  /**
   * 清理缓存管理器
   */
  protected async onCleanup(): Promise<void> {
    this.stopCleanupTimer();
    this.clearAll();
    logger.info('缓存管理器清理完成');
  }

  /**
   * 设置缓存
   */
  set<T>(key: string, data: T, expiry?: number): void {
    try {
      // 如果缓存已满，清理最少使用的项
      if (this.cache.size >= this.config.maxSize) {
        this.evictLeastUsed();
      }

      const cacheItem: CacheItem<T> = {
        data,
        timestamp: Date.now(),
        expiry: expiry || this.config.defaultExpiry,
        accessCount: 0,
        lastAccess: Date.now()
      };

      this.cache.set(key, cacheItem as CacheItem<Object>);
      logger.debug(`缓存设置成功: ${key}`);
    } catch (error) {
      ErrorHandler.handle(error, `CacheManager.set(${key})`, ErrorType.SYSTEM, ErrorLevel.LOW);
    }
  }

  /**
   * 获取缓存
   */
  get<T>(key: string): T | null {
    try {
      const cacheItem = this.cache.get(key);
      if (!cacheItem) {
        this.updateStats('miss');
        return null;
      }

      // 检查是否过期
      if (Date.now() - cacheItem.timestamp > cacheItem.expiry) {
        this.cache.delete(key);
        this.updateStats('miss');
        logger.debug(`缓存已过期: ${key}`);
        return null;
      }

      // 更新访问信息
      cacheItem.accessCount++;
      cacheItem.lastAccess = Date.now();
      this.updateStats('hit');

      return cacheItem.data as T;
    } catch (error) {
      ErrorHandler.handle(error, `CacheManager.get(${key})`, ErrorType.SYSTEM, ErrorLevel.LOW);
      this.updateStats('miss');
      return null;
    }
  }

  /**
   * 检查缓存是否存在
   */
  has(key: string): boolean {
    const item = this.cache.get(key);
    if (!item) return false;
    
    // 检查是否过期
    if (Date.now() - item.timestamp > item.expiry) {
      this.cache.delete(key);
      return false;
    }
    
    return true;
  }

  /**
   * 删除缓存
   */
  delete(key: string): boolean {
    const result = this.cache.delete(key);
    if (result) {
      logger.debug(`缓存删除成功: ${key}`);
    }
    return result;
  }

  /**
   * 清空所有缓存
   */
  clearAll(): void {
    this.cache.clear();
    this.resetStats();
    logger.info('所有缓存已清空');
  }

  /**
   * 清理过期缓存
   */
  cleanupExpired(): number {
    let cleanedCount = 0;
    const now = Date.now();
    
    const cacheEntries = Array.from(this.cache.entries());
    for (const entry of cacheEntries) {
      const key = entry[0];
      const item = entry[1];
      if (now - item.timestamp > item.expiry) {
        this.cache.delete(key);
        cleanedCount++;
      }
    }
    
    if (cleanedCount > 0) {
      logger.debug(`清理了 ${cleanedCount} 个过期缓存项`);
    }
    
    return cleanedCount;
  }

  /**
   * 获取缓存统计信息
   */
  getStats(): CacheStats {
    const totalRequests = this.stats.hits + this.stats.misses;
    const hitRate = totalRequests > 0 ? (this.stats.hits / totalRequests) * 100 : 0;
    
    return {
      size: this.cache.size,
      maxSize: this.config.maxSize,
      hitRate: Math.round(hitRate * 100) / 100,
      totalHits: this.stats.hits,
      totalMisses: this.stats.misses,
      totalRequests
    };
  }

  /**
   * 获取缓存配置
   */
  getConfig(): CacheConfig {
    return {
      maxSize: this.config.maxSize,
      defaultExpiry: this.config.defaultExpiry,
      cleanupInterval: this.config.cleanupInterval,
      enableStats: this.config.enableStats
    };
  }

  /**
   * 更新缓存配置
   */
  updateConfig(newConfig: Partial<CacheConfig>): void {
    if (newConfig.maxSize !== undefined) {
      this.config.maxSize = newConfig.maxSize;
    }
    if (newConfig.defaultExpiry !== undefined) {
      this.config.defaultExpiry = newConfig.defaultExpiry;
    }
    if (newConfig.cleanupInterval !== undefined) {
      this.config.cleanupInterval = newConfig.cleanupInterval;
    }
    if (newConfig.enableStats !== undefined) {
      this.config.enableStats = newConfig.enableStats;
    }
    logger.info('缓存配置已更新', this.config);
    
    // 如果最大大小减小了，需要清理缓存
    if (newConfig.maxSize && newConfig.maxSize < this.cache.size) {
      this.evictToSize(newConfig.maxSize);
    }
    
    // 重启清理定时器
    if (newConfig.cleanupInterval) {
      this.stopCleanupTimer();
      this.startCleanupTimer();
    }
  }

  /**
   * 获取所有缓存键
   */
  getKeys(): string[] {
    return Array.from(this.cache.keys());
  }

  /**
   * 获取缓存项详细信息
   */
  getItemInfo(key: string): CacheItemInfo {
    const item = this.cache.get(key);
    if (!item) {
      const itemInfo: CacheItemInfo = { exists: false };
      return itemInfo;
    }
    
    const itemInfo: CacheItemInfo = {
      exists: true,
      accessCount: item.accessCount,
      lastAccess: item.lastAccess,
      expiry: item.expiry
    };
    return itemInfo;
  }

  /**
   * 清理最少使用的缓存项
   */
  private evictLeastUsed(): void {
    let leastUsedKey: string | null = null;
    let leastAccessCount = Infinity;
    let oldestAccess = Infinity;

    const cacheEntries = Array.from(this.cache.entries());
    for (const entry of cacheEntries) {
      const key = entry[0];
      const item = entry[1];
      // 优先清理访问次数最少的，如果相同则清理最久未访问的
      if (item.accessCount < leastAccessCount || 
          (item.accessCount === leastAccessCount && item.lastAccess < oldestAccess)) {
        leastAccessCount = item.accessCount;
        oldestAccess = item.lastAccess;
        leastUsedKey = key;
      }
    }

    if (leastUsedKey) {
      this.cache.delete(leastUsedKey);
      logger.debug(`清理最少使用缓存: ${leastUsedKey}`);
    }
  }

  /**
   * 清理缓存到指定大小
   */
  private evictToSize(targetSize: number): void {
    while (this.cache.size > targetSize) {
      this.evictLeastUsed();
    }
  }

  /**
   * 更新统计信息
   */
  private updateStats(type: 'hit' | 'miss'): void {
    if (!this.config.enableStats) return;
    
    if (type === 'hit') {
      this.stats.hits++;
    } else {
      this.stats.misses++;
    }
  }

  /**
   * 重置统计信息
   */
  private resetStats(): void {
    this.stats.hits = 0;
    this.stats.misses = 0;
  }

  /**
   * 启动清理定时器
   */
  private startCleanupTimer(): void {
    if (this.stats.cleanupTimerId !== -1) return;
    
    this.stats.cleanupTimerId = TimerManager.setInterval(() => {
      this.cleanupExpired();
    }, this.config.cleanupInterval, 'CacheCleanup');
    
    logger.debug('缓存清理定时器已启动');
  }

  /**
   * 停止清理定时器
   */
  private stopCleanupTimer(): void {
    if (this.stats.cleanupTimerId !== -1) {
      TimerManager.clearInterval(this.stats.cleanupTimerId);
      this.stats.cleanupTimerId = -1;
      logger.debug('缓存清理定时器已停止');
    }
  }
}
