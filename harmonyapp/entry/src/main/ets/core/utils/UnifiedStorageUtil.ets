/**
 * 统一存储工具类
 * 合并StorageUtil和PreferencesUtil的功能，提供统一的存储接口
 */

import { preferences } from '@kit.ArkData';
import { common } from '@kit.AbilityKit';
import { BaseSingleton } from '../base/Singleton';
import { createLogger } from './LogManager';
import { ErrorHandler, ErrorType, ErrorLevel } from './ErrorHandler';

const logger = createLogger('UnifiedStorageUtil');

/**
 * 存储配置接口
 */
interface StorageConfig {
  enableEncryption: boolean;
  enableCompression: boolean;
  maxRetryAttempts: number;
  retryDelay: number;
  cacheSize: number;
  enableBatch: boolean;
  batchSize: number;
  enableMetrics: boolean;
}

/**
 * 存储统计信息
 */
export interface StorageStats {
  totalOperations: number;
  successfulOperations: number;
  failedOperations: number;
  cacheHits: number;
  cacheMisses: number;
  averageResponseTime: number;
  lastOperationTime: number;
}

/**
 * 存储操作结果
 */
interface StorageResult<T> {
  success: boolean;
  data?: T;
  error?: string;
}

/**
 * 统一存储工具类
 * 使用单例模式，提供统一的存储接口
 */
export class UnifiedStorageUtil extends BaseSingleton {
  private static instance: UnifiedStorageUtil | null = null;
  private static readonly DEFAULT_STORE_NAME = 'hotspot_app_storage';
  private preferencesStore: preferences.Preferences | null = null;
  private config: StorageConfig;
  private stats: StorageStats;
  private context: common.Context | null = null;

  private constructor() {
    super();
    this.config = {
      enableEncryption: false,
      enableCompression: false,
      maxRetryAttempts: 3,
      retryDelay: 1000,
      cacheSize: 100,
      enableBatch: true,
      batchSize: 10,
      enableMetrics: true
    };
    this.stats = {
      totalOperations: 0,
      successfulOperations: 0,
      failedOperations: 0,
      cacheHits: 0,
      cacheMisses: 0,
      averageResponseTime: 0,
      lastOperationTime: Date.now()
    };
  }

  /**
   * 获取单例实例
   */
  static getInstance(): UnifiedStorageUtil {
    if (!UnifiedStorageUtil.instance) {
      UnifiedStorageUtil.instance = new UnifiedStorageUtil();
    }
    return UnifiedStorageUtil.instance;
  }

  /**
   * 异步获取单例实例
   */
  static async getInstanceAsync(): Promise<UnifiedStorageUtil> {
    const instance = UnifiedStorageUtil.getInstance();
    await instance.initialize();
    return instance;
  }

  /**
   * 初始化存储（需要传入Context）
   */
  async initializeWithContext(context: common.Context, config?: Partial<StorageConfig>): Promise<void> {
    this.context = context;
    if (config) {
      // 手动合并配置，避免使用展开语法
      const newConfig: StorageConfig = {
        enableEncryption: config.enableEncryption !== undefined ? config.enableEncryption : this.config.enableEncryption,
        enableCompression: config.enableCompression !== undefined ? config.enableCompression : this.config.enableCompression,
        maxRetryAttempts: config.maxRetryAttempts !== undefined ? config.maxRetryAttempts : this.config.maxRetryAttempts,
        retryDelay: config.retryDelay !== undefined ? config.retryDelay : this.config.retryDelay,
        cacheSize: config.cacheSize !== undefined ? config.cacheSize : this.config.cacheSize,
        enableBatch: config.enableBatch !== undefined ? config.enableBatch : this.config.enableBatch,
        batchSize: config.batchSize !== undefined ? config.batchSize : this.config.batchSize,
        enableMetrics: config.enableMetrics !== undefined ? config.enableMetrics : this.config.enableMetrics
      };
      this.config = newConfig;
    }
    await this.initialize();
  }

  /**
   * 实现基类的初始化方法
   */
  protected async onInitialize(): Promise<void> {
    if (!this.context) {
      throw new Error('存储初始化失败：Context未设置');
    }

    try {
      this.preferencesStore = await preferences.getPreferences(
        this.context, 
        UnifiedStorageUtil.DEFAULT_STORE_NAME
      );
      
      // 更新统计信息
      await this.updateStats();
      
      logger.info(`存储初始化成功: ${UnifiedStorageUtil.DEFAULT_STORE_NAME}`);
    } catch (error) {
      const standardError = ErrorHandler.handle(
        error, 
        'UnifiedStorageUtil.onInitialize', 
        ErrorType.STORAGE, 
        ErrorLevel.CRITICAL
      );
      throw new Error(standardError.message);
    }
  }

  /**
   * 实现基类的清理方法
   */
  protected async onCleanup(): Promise<void> {
    try {
      if (this.preferencesStore) {
        // HarmonyOS Preferences 不需要显式关闭
        this.preferencesStore = null;
      }
      this.context = null;
      logger.info('存储清理完成');
    } catch (error) {
      ErrorHandler.handle(error, 'UnifiedStorageUtil.onCleanup', ErrorType.STORAGE);
    }
  }

  /**
   * 存储字符串值
   */
  async setString(key: string, value: string): Promise<boolean> {
    return this.executeWithRetry(async () => {
      this.ensureInitialized();
      await this.preferencesStore!.put(key, value);
      await this.preferencesStore!.flush();
      this.updateStatsAfterOperation();
      logger.debug(`存储字符串成功: ${key}`);
      return true;
    }, `setString(${key})`);
  }

  /**
   * 获取字符串值
   */
  async getString(key: string, defaultValue: string = ''): Promise<string> {
    return this.executeWithRetry(async () => {
      this.ensureInitialized();
      const result = await this.preferencesStore!.get(key, defaultValue);
      this.updateStatsAfterOperation();
      return result as string;
    }, `getString(${key})`, defaultValue);
  }

  /**
   * 存储数字值
   */
  async setNumber(key: string, value: number): Promise<boolean> {
    return this.executeWithRetry(async () => {
      this.ensureInitialized();
      await this.preferencesStore!.put(key, value);
      await this.preferencesStore!.flush();
      this.updateStatsAfterOperation();
      logger.debug(`存储数字成功: ${key}`);
      return true;
    }, `setNumber(${key})`);
  }

  /**
   * 获取数字值
   */
  async getNumber(key: string, defaultValue: number = 0): Promise<number> {
    return this.executeWithRetry(async () => {
      this.ensureInitialized();
      const result = await this.preferencesStore!.get(key, defaultValue);
      this.updateStatsAfterOperation();
      return result as number;
    }, `getNumber(${key})`, defaultValue);
  }

  /**
   * 存储布尔值
   */
  async setBoolean(key: string, value: boolean): Promise<boolean> {
    return this.executeWithRetry(async () => {
      this.ensureInitialized();
      await this.preferencesStore!.put(key, value);
      await this.preferencesStore!.flush();
      this.updateStatsAfterOperation();
      logger.debug(`存储布尔值成功: ${key}`);
      return true;
    }, `setBoolean(${key})`);
  }

  /**
   * 获取布尔值
   */
  async getBoolean(key: string, defaultValue: boolean = false): Promise<boolean> {
    return this.executeWithRetry(async () => {
      this.ensureInitialized();
      const result = await this.preferencesStore!.get(key, defaultValue);
      this.updateStatsAfterOperation();
      return result as boolean;
    }, `getBoolean(${key})`, defaultValue);
  }

  /**
   * 存储对象（自动序列化）
   */
  async setObject<T>(key: string, value: T): Promise<boolean> {
    try {
      const jsonString = JSON.stringify(value);
      return await this.setString(key, jsonString);
    } catch (error) {
      ErrorHandler.handle(error, `setObject(${key})`, ErrorType.STORAGE);
      return false;
    }
  }

  /**
   * 获取对象（自动反序列化）
   */
  async getObject<T>(key: string, defaultValue?: T): Promise<T | null> {
    try {
      const jsonString = await this.getString(key);
      if (!jsonString) {
        return defaultValue || null;
      }
      return JSON.parse(jsonString) as T;
    } catch (error) {
      ErrorHandler.handle(error, `getObject(${key})`, ErrorType.STORAGE);
      return defaultValue || null;
    }
  }

  /**
   * 删除指定键
   */
  async remove(key: string): Promise<boolean> {
    return this.executeWithRetry(async () => {
      this.ensureInitialized();
      await this.preferencesStore!.delete(key);
      await this.preferencesStore!.flush();
      this.updateStatsAfterOperation();
      logger.debug(`删除键成功: ${key}`);
      return true;
    }, `remove(${key})`);
  }

  /**
   * 检查键是否存在
   */
  async has(key: string): Promise<boolean> {
    return this.executeWithRetry(async () => {
      this.ensureInitialized();
      const result = await this.preferencesStore!.has(key);
      this.updateStatsAfterOperation();
      return result;
    }, `has(${key})`, false);
  }

  /**
   * 清空所有数据
   */
  async clear(): Promise<boolean> {
    return this.executeWithRetry(async () => {
      this.ensureInitialized();
      await this.preferencesStore!.clear();
      await this.preferencesStore!.flush();
      this.stats.totalOperations += 1;
      this.stats.successfulOperations += 1;
      this.updateStatsAfterOperation();
      logger.info('清空所有存储数据');
      return true;
    }, 'clear()');
  }

  /**
   * 获取所有键（注意：HarmonyOS Preferences API可能不支持此功能）
   */
  async getAllKeys(): Promise<string[]> {
    try {
      this.ensureInitialized();
      // HarmonyOS Preferences API 不直接支持获取所有键
      // 这里返回空数组并记录警告
      logger.warn('HarmonyOS Preferences API 不支持获取所有键');
      return [];
    } catch (error) {
      ErrorHandler.handle(error, 'getAllKeys()', ErrorType.STORAGE);
      return [];
    }
  }

  /**
   * 批量操作：批量设置
   */
  async setBatch(data: Record<string, preferences.ValueType>): Promise<boolean> {
    return this.executeWithRetry(async () => {
      this.ensureInitialized();
      
      const operations: Promise<void>[] = [];
      const entries = Object.entries(data);
      for (let i = 0; i < entries.length; i++) {
        const entry = entries[i];
        const key = entry[0];
        const value = entry[1];
        operations.push(this.preferencesStore!.put(key, value));
      }
      
      await Promise.all<void>(operations);
      await this.preferencesStore!.flush();
      
      this.updateStatsAfterOperation();
      logger.debug(`批量设置完成，共 ${Object.keys(data).length} 项`);
      return true;
    }, 'setBatch()');
  }

  /**
   * 批量操作：批量删除
   */
  async removeBatch(keys: string[]): Promise<boolean> {
    return this.executeWithRetry(async () => {
      this.ensureInitialized();
      
      const operations: Promise<void>[] = [];
      for (const key of keys) {
        operations.push(this.preferencesStore!.delete(key));
      }
      
      await Promise.all<void>(operations);
      await this.preferencesStore!.flush();
      
      this.updateStatsAfterOperation();
      logger.debug(`批量删除完成，共 ${keys.length} 项`);
      return true;
    }, 'removeBatch()');
  }

  /**
   * 获取存储统计信息
   */
  getStats(): StorageStats {
    // 手动创建副本，避免使用展开语法
    return {
      totalOperations: this.stats.totalOperations,
      successfulOperations: this.stats.successfulOperations,
      failedOperations: this.stats.failedOperations,
      cacheHits: this.stats.cacheHits,
      cacheMisses: this.stats.cacheMisses,
      averageResponseTime: this.stats.averageResponseTime,
      lastOperationTime: this.stats.lastOperationTime
    };
  }

  /**
   * 获取存储配置
   */
  getConfig(): StorageConfig {
    // 手动创建副本，避免使用展开语法
    const configCopy: StorageConfig = {
      enableEncryption: this.config.enableEncryption,
      enableCompression: this.config.enableCompression,
      maxRetryAttempts: this.config.maxRetryAttempts,
      retryDelay: this.config.retryDelay,
      cacheSize: this.config.cacheSize,
      enableBatch: this.config.enableBatch,
      batchSize: this.config.batchSize,
      enableMetrics: this.config.enableMetrics
    };
    return configCopy;
  }

  /**
   * 更新配置
   */
  updateConfig(config: Partial<StorageConfig>): void {
    // 手动合并配置，避免使用展开语法
    const newConfig: StorageConfig = {
      enableEncryption: config.enableEncryption !== undefined ? config.enableEncryption : this.config.enableEncryption,
      enableCompression: config.enableCompression !== undefined ? config.enableCompression : this.config.enableCompression,
      maxRetryAttempts: config.maxRetryAttempts !== undefined ? config.maxRetryAttempts : this.config.maxRetryAttempts,
      retryDelay: config.retryDelay !== undefined ? config.retryDelay : this.config.retryDelay,
      cacheSize: config.cacheSize !== undefined ? config.cacheSize : this.config.cacheSize,
      enableBatch: config.enableBatch !== undefined ? config.enableBatch : this.config.enableBatch,
      batchSize: config.batchSize !== undefined ? config.batchSize : this.config.batchSize,
      enableMetrics: config.enableMetrics !== undefined ? config.enableMetrics : this.config.enableMetrics
    };
    this.config = newConfig;
    logger.debug('存储配置已更新');
  }

  /**
   * 带重试机制的执行方法
   */
  private async executeWithRetry<T>(
    operation: () => Promise<T>,
    operationName: string,
    fallbackValue?: T
  ): Promise<T> {
    let lastError: Object = new Error('初始错误');
    
    for (let attempt = 1; attempt <= this.config.maxRetryAttempts; attempt++) {
      try {
        return await operation();
      } catch (error) {
        lastError = error;
        logger.warn(`存储操作失败 (${attempt}/${this.config.maxRetryAttempts}): ${operationName}`);
        
        if (attempt < this.config.maxRetryAttempts) {
          await new Promise<void>(resolve => setTimeout(resolve, this.config.retryDelay));
        }
      }
    }
    
    // 所有重试都失败了
    ErrorHandler.handle(lastError, operationName, ErrorType.STORAGE, ErrorLevel.HIGH);
    
    if (fallbackValue !== undefined) {
      return fallbackValue;
    }
    
    throw new Error(`存储操作最终失败: ${operationName}`);
  }

  /**
   * 更新统计信息
   */
  private async updateStats(): Promise<void> {
    try {
      // 由于HarmonyOS API限制，无法准确获取键数量和大小
      // 这里提供估算值
      this.stats.lastOperationTime = Date.now();
      // 更新统计信息
    } catch (error) {
      logger.warn(`更新存储统计信息失败: ${error}`);
    }
  }

  /**
   * 操作后更新统计信息
   */
  private updateStatsAfterOperation(): void {
    this.stats.lastOperationTime = Date.now();
    this.stats.totalOperations++;
  }
}

/**
 * 便捷函数：获取存储实例
 */
export async function getStorage(): Promise<UnifiedStorageUtil> {
  return UnifiedStorageUtil.getInstanceAsync();
}

/**
 * 便捷函数：初始化存储
 */
export async function initializeStorage(
  context: common.Context, 
  config?: Partial<StorageConfig>
): Promise<UnifiedStorageUtil> {
  const storage = UnifiedStorageUtil.getInstance();
  await storage.initializeWithContext(context, config);
  return storage;
}
