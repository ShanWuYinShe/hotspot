/**
 * LRU (Least Recently Used) 缓存实现
 * 提供高效的缓存管理，自动淘汰最少使用的项
 */

import { createLogger } from './LogManager';
import { ErrorHandler, ErrorType, ErrorLevel } from './ErrorHandler';
import { TimerManager } from './TimerManager';

const logger = createLogger('LRUCache');

/**
 * 缓存节点
 */
class CacheNode<K, V> {
  key: K;
  value: V;
  prev: CacheNode<K, V> | null = null;
  next: CacheNode<K, V> | null = null;
  timestamp: number;
  accessCount: number = 0;
  expireTime?: number;

  constructor(key: K, value: V, expireTime?: number) {
    this.key = key;
    this.value = value;
    this.timestamp = Date.now();
    this.expireTime = expireTime;
  }

  isExpired(): boolean {
    return this.expireTime !== undefined && Date.now() > this.expireTime;
  }
}

/**
 * LRU缓存配置
 */
export interface LRUCacheConfig {
  maxSize: number;
  defaultTTL?: number; // 默认生存时间（毫秒）
  enableMetrics: boolean;
  cleanupInterval: number; // 清理间隔（毫秒）
  onEvict?: (key: any, value: any) => void; // 淘汰回调
}

/**
 * 缓存统计信息
 */
export interface CacheStats {
  size: number;
  maxSize: number;
  hitCount: number;
  missCount: number;
  evictionCount: number;
  hitRate: number;
  averageAccessTime: number;
  oldestEntryAge: number;
}

/**
 * LRU缓存实现
 */
export class LRUCache<K, V> {
  private capacity: number;
  private cache: Map<K, CacheNode<K, V>> = new Map();
  private head: CacheNode<K, V>;
  private tail: CacheNode<K, V>;
  private config: LRUCacheConfig;
  private stats: CacheStats;
  private cleanupTimerId: number = -1;

  constructor(capacity: number, config?: Partial<LRUCacheConfig>) {
    this.capacity = capacity;
    this.config = {
      maxSize: capacity,
      enableMetrics: true,
      cleanupInterval: 5 * 60 * 1000, // 5分钟
      ...config
    };

    // 创建虚拟头尾节点
    this.head = new CacheNode(null as any, null as any);
    this.tail = new CacheNode(null as any, null as any);
    this.head.next = this.tail;
    this.tail.prev = this.head;

    this.stats = {
      size: 0,
      maxSize: capacity,
      hitCount: 0,
      missCount: 0,
      evictionCount: 0,
      hitRate: 0,
      averageAccessTime: 0,
      oldestEntryAge: 0
    };

    this.startCleanupTask();
  }

  /**
   * 获取缓存值
   */
  get(key: K): V | undefined {
    const startTime = performance.now();

    try {
      const node = this.cache.get(key);
      
      if (!node) {
        this.stats.missCount++;
        this.updateStats();
        return undefined;
      }

      // 检查是否过期
      if (node.isExpired()) {
        this.delete(key);
        this.stats.missCount++;
        this.updateStats();
        return undefined;
      }

      // 移动到头部（最近使用）
      this.moveToHead(node);
      node.accessCount++;
      node.timestamp = Date.now();

      this.stats.hitCount++;
      this.updateStats();

      // 更新平均访问时间
      if (this.config.enableMetrics) {
        const accessTime = performance.now() - startTime;
        this.updateAverageAccessTime(accessTime);
      }

      return node.value;
    } catch (error) {
      ErrorHandler.handle(error, `LRUCache.get(${key})`, ErrorType.SYSTEM, ErrorLevel.LOW);
      return undefined;
    }
  }

  /**
   * 设置缓存值
   */
  set(key: K, value: V, ttl?: number): void {
    try {
      const existingNode = this.cache.get(key);
      const expireTime = ttl ? Date.now() + ttl : 
                        (this.config.defaultTTL ? Date.now() + this.config.defaultTTL : undefined);

      if (existingNode) {
        // 更新现有节点
        existingNode.value = value;
        existingNode.timestamp = Date.now();
        existingNode.expireTime = expireTime;
        this.moveToHead(existingNode);
      } else {
        // 创建新节点
        const newNode = new CacheNode(key, value, expireTime);
        
        // 检查容量
        if (this.cache.size >= this.capacity) {
          this.evictLRU();
        }

        this.cache.set(key, newNode);
        this.addToHead(newNode);
      }

      this.updateStats();
    } catch (error) {
      ErrorHandler.handle(error, `LRUCache.set(${key})`, ErrorType.SYSTEM, ErrorLevel.LOW);
    }
  }

  /**
   * 删除缓存项
   */
  delete(key: K): boolean {
    try {
      const node = this.cache.get(key);
      if (!node) {
        return false;
      }

      this.cache.delete(key);
      this.removeNode(node);
      this.updateStats();

      // 调用淘汰回调
      if (this.config.onEvict) {
        this.config.onEvict(key, node.value);
      }

      return true;
    } catch (error) {
      ErrorHandler.handle(error, `LRUCache.delete(${key})`, ErrorType.SYSTEM, ErrorLevel.LOW);
      return false;
    }
  }

  /**
   * 检查是否存在
   */
  has(key: K): boolean {
    const node = this.cache.get(key);
    return node !== undefined && !node.isExpired();
  }

  /**
   * 清空缓存
   */
  clear(): void {
    try {
      // 调用所有项的淘汰回调
      if (this.config.onEvict) {
        for (const [key, node] of this.cache.entries()) {
          this.config.onEvict(key, node.value);
        }
      }

      this.cache.clear();
      this.head.next = this.tail;
      this.tail.prev = this.head;
      this.updateStats();

      logger.debug('LRU缓存已清空');
    } catch (error) {
      ErrorHandler.handle(error, 'LRUCache.clear', ErrorType.SYSTEM, ErrorLevel.LOW);
    }
  }

  /**
   * 获取缓存大小
   */
  size(): number {
    return this.cache.size;
  }

  /**
   * 获取所有键
   */
  keys(): K[] {
    return Array.from(this.cache.keys());
  }

  /**
   * 获取所有值
   */
  values(): V[] {
    return Array.from(this.cache.values()).map(node => node.value);
  }

  /**
   * 获取统计信息
   */
  getStats(): CacheStats {
    return { ...this.stats };
  }

  /**
   * 获取缓存使用情况
   */
  getUsageInfo(): Array<{
    key: K;
    accessCount: number;
    age: number;
    isExpired: boolean;
  }> {
    const now = Date.now();
    const result: Array<{
      key: K;
      accessCount: number;
      age: number;
      isExpired: boolean;
    }> = [];

    for (const [key, node] of this.cache.entries()) {
      result.push({
        key,
        accessCount: node.accessCount,
        age: now - node.timestamp,
        isExpired: node.isExpired()
      });
    }

    return result.sort((a, b) => b.accessCount - a.accessCount);
  }

  /**
   * 预热缓存
   */
  warmup(entries: Array<{ key: K; value: V; ttl?: number }>): void {
    try {
      for (const entry of entries) {
        this.set(entry.key, entry.value, entry.ttl);
      }
      logger.debug(`缓存预热完成，加载 ${entries.length} 个条目`);
    } catch (error) {
      ErrorHandler.handle(error, 'LRUCache.warmup', ErrorType.SYSTEM, ErrorLevel.LOW);
    }
  }

  /**
   * 销毁缓存
   */
  destroy(): void {
    try {
      this.stopCleanupTask();
      this.clear();
      logger.debug('LRU缓存已销毁');
    } catch (error) {
      ErrorHandler.handle(error, 'LRUCache.destroy', ErrorType.SYSTEM, ErrorLevel.LOW);
    }
  }

  /**
   * 淘汰最少使用的项
   */
  private evictLRU(): void {
    const lru = this.tail.prev;
    if (lru && lru !== this.head) {
      const key = lru.key;
      const value = lru.value;
      
      this.cache.delete(key);
      this.removeNode(lru);
      this.stats.evictionCount++;

      // 调用淘汰回调
      if (this.config.onEvict) {
        this.config.onEvict(key, value);
      }

      logger.debug(`淘汰LRU项: ${key}`);
    }
  }

  /**
   * 移动节点到头部
   */
  private moveToHead(node: CacheNode<K, V>): void {
    this.removeNode(node);
    this.addToHead(node);
  }

  /**
   * 添加节点到头部
   */
  private addToHead(node: CacheNode<K, V>): void {
    node.prev = this.head;
    node.next = this.head.next;
    
    if (this.head.next) {
      this.head.next.prev = node;
    }
    this.head.next = node;
  }

  /**
   * 移除节点
   */
  private removeNode(node: CacheNode<K, V>): void {
    if (node.prev) {
      node.prev.next = node.next;
    }
    if (node.next) {
      node.next.prev = node.prev;
    }
  }

  /**
   * 更新统计信息
   */
  private updateStats(): void {
    this.stats.size = this.cache.size;
    
    // 计算命中率
    const total = this.stats.hitCount + this.stats.missCount;
    this.stats.hitRate = total > 0 ? (this.stats.hitCount / total) * 100 : 0;

    // 计算最老条目年龄
    let oldestAge = 0;
    const now = Date.now();
    for (const node of this.cache.values()) {
      const age = now - node.timestamp;
      if (age > oldestAge) {
        oldestAge = age;
      }
    }
    this.stats.oldestEntryAge = oldestAge;
  }

  /**
   * 更新平均访问时间
   */
  private updateAverageAccessTime(accessTime: number): void {
    if (this.stats.averageAccessTime === 0) {
      this.stats.averageAccessTime = accessTime;
    } else {
      // 使用移动平均
      this.stats.averageAccessTime = 
        (this.stats.averageAccessTime * 0.9) + (accessTime * 0.1);
    }
  }

  /**
   * 启动清理任务
   */
  private startCleanupTask(): void {
    this.cleanupTimerId = TimerManager.setInterval(() => {
      this.performCleanup();
    }, this.config.cleanupInterval, 'LRU缓存清理');
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
      const initialSize = this.cache.size;
      const expiredKeys: K[] = [];

      // 查找过期的键
      for (const [key, node] of this.cache.entries()) {
        if (node.isExpired()) {
          expiredKeys.push(key);
        }
      }

      // 删除过期的键
      for (const key of expiredKeys) {
        this.delete(key);
      }

      const cleanedCount = expiredKeys.length;
      if (cleanedCount > 0) {
        logger.debug(`清理了 ${cleanedCount} 个过期缓存项`);
      }

      // 记录缓存状态
      if (this.config.enableMetrics) {
        logger.debug(`缓存状态: 大小=${this.cache.size}/${this.capacity}, 命中率=${this.stats.hitRate.toFixed(1)}%`);
      }
    } catch (error) {
      ErrorHandler.handle(error, 'LRUCache.performCleanup', ErrorType.SYSTEM, ErrorLevel.LOW);
    }
  }
}

/**
 * 缓存管理器
 */
export class CacheManager {
  private static caches: Map<string, LRUCache<any, any>> = new Map();

  /**
   * 创建并注册缓存
   */
  static createCache<K, V>(
    name: string,
    capacity: number,
    config?: Partial<LRUCacheConfig>
  ): LRUCache<K, V> {
    const cache = new LRUCache<K, V>(capacity, config);
    CacheManager.caches.set(name, cache);
    logger.debug(`创建缓存: ${name}, 容量: ${capacity}`);
    return cache;
  }

  /**
   * 获取缓存
   */
  static getCache<K, V>(name: string): LRUCache<K, V> | undefined {
    return CacheManager.caches.get(name);
  }

  /**
   * 移除缓存
   */
  static removeCache(name: string): boolean {
    const cache = CacheManager.caches.get(name);
    if (cache) {
      cache.destroy();
      CacheManager.caches.delete(name);
      logger.debug(`移除缓存: ${name}`);
      return true;
    }
    return false;
  }

  /**
   * 获取所有缓存统计信息
   */
  static getAllStats(): Record<string, CacheStats> {
    const stats: Record<string, CacheStats> = {};
    for (const [name, cache] of CacheManager.caches.entries()) {
      stats[name] = cache.getStats();
    }
    return stats;
  }

  /**
   * 清理所有缓存
   */
  static cleanup(): void {
    logger.info('开始清理所有缓存...');
    for (const [name, cache] of CacheManager.caches.entries()) {
      try {
        cache.destroy();
        logger.debug(`清理缓存: ${name}`);
      } catch (error) {
        logger.error(`清理缓存失败: ${name} - ${error}`);
      }
    }
    CacheManager.caches.clear();
    logger.info('所有缓存清理完成');
  }
}

/**
 * 便捷函数：创建缓存
 */
export function createCache<K, V>(
  name: string,
  capacity: number,
  config?: Partial<LRUCacheConfig>
): LRUCache<K, V> {
  return CacheManager.createCache(name, capacity, config);
}
