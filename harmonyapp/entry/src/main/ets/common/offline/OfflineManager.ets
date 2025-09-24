/**
 * 离线管理器
 * 处理离线数据存储、同步和网络状态管理
 */

import { connection } from '@kit.NetworkKit';
import { relationalStore } from '@kit.ArkData';
import { common } from '@kit.AbilityKit';
import { BusinessError } from '@kit.BasicServicesKit';
import { createLogger } from '../utils/LogManager';
import { ErrorHandler, ErrorType, ErrorLevel } from '../utils/ErrorHandler';
import { BaseSingleton } from '../base/Singleton';
import { EventBus } from '../events/EventBus';

const logger = createLogger('OfflineManager');

/**
 * 网络状态枚举
 */
export enum NetworkStatus {
  ONLINE = 'online',
  OFFLINE = 'offline',
  POOR = 'poor'        // 网络质量差
}

/**
 * 同步状态枚举
 */
export enum SyncStatus {
  IDLE = 'idle',           // 空闲
  SYNCING = 'syncing',     // 同步中
  SUCCESS = 'success',     // 同步成功
  FAILED = 'failed'        // 同步失败
}

/**
 * 离线数据类型
 */
export enum OfflineDataType {
  MESSAGE = 'message',
  MOMENT = 'moment',
  GROUP = 'group',
  USER = 'user',
  DRAFT = 'draft'
}

/**
 * 离线数据项
 */
export interface OfflineDataItem {
  id: string;                    // 唯一标识
  type: OfflineDataType;         // 数据类型
  action: 'create' | 'update' | 'delete'; // 操作类型
  data: Record<string, Object>;  // 数据内容
  timestamp: number;             // 时间戳
  retryCount: number;            // 重试次数
  priority: number;              // 优先级（1-10，数字越大优先级越高）
  userId: string;                // 用户ID
}

/**
 * 同步配置
 */
export interface SyncConfig {
  maxRetryCount: number;         // 最大重试次数
  retryInterval: number;         // 重试间隔（毫秒）
  batchSize: number;             // 批量同步大小
  syncInterval: number;          // 自动同步间隔（毫秒）
  enableAutoSync: boolean;       // 是否启用自动同步
}

/**
 * 网络质量信息
 */
export interface NetworkQuality {
  status: NetworkStatus;
  type: connection.NetBearType;
  strength: number;              // 信号强度 0-100
  latency: number;               // 延迟（毫秒）
}

/**
 * 同步统计信息
 */
export interface SyncStats {
  totalItems: number;            // 总数据项
  pendingItems: number;          // 待同步项
  successItems: number;          // 成功同步项
  failedItems: number;           // 失败项
  lastSyncTime: number;          // 最后同步时间
  syncDuration: number;          // 同步耗时（毫秒）
}

/**
 * 离线管理器
 */
export class OfflineManager extends BaseSingleton {
  private static instance: OfflineManager | null = null;
  private context: common.UIAbilityContext | null = null;
  private database: relationalStore.RdbStore | null = null;
  private eventBus: EventBus;
  
  private networkStatus: NetworkStatus = NetworkStatus.OFFLINE;
  private syncStatus: SyncStatus = SyncStatus.IDLE;
  private networkQuality: NetworkQuality | null = null;
  
  private syncConfig: SyncConfig = {
    maxRetryCount: 3,
    retryInterval: 5000,
    batchSize: 10,
    syncInterval: 30000,
    enableAutoSync: true
  };

  private syncTimer: number = -1;
  private networkMonitorTimer: number = -1;
  private syncStats: SyncStats = {
    totalItems: 0,
    pendingItems: 0,
    successItems: 0,
    failedItems: 0,
    lastSyncTime: 0,
    syncDuration: 0
  };

  // 数据库配置
  private readonly dbConfig = {
    name: 'hotspot_offline.db',
    version: 1,
    encrypted: false
  };

  private constructor() {
    super();
    this.eventBus = EventBus.getInstance();
  }

  public static getInstance(): OfflineManager {
    if (!OfflineManager.instance) {
      OfflineManager.instance = new OfflineManager();
    }
    return OfflineManager.instance;
  }

  public static async getInstanceAsync(): Promise<OfflineManager> {
    const instance = OfflineManager.getInstance();
    await instance.initialize();
    return instance;
  }

  protected async onInitialize(): Promise<void> {
    logger.info('离线管理器初始化');
    await this.initializeDatabase();
    await this.startNetworkMonitoring();
    await this.loadSyncStats();
    
    if (this.syncConfig.enableAutoSync) {
      this.startAutoSync();
    }
  }

  protected async onCleanup(): Promise<void> {
    logger.info('离线管理器清理');
    
    this.stopAutoSync();
    this.stopNetworkMonitoring();
    
    if (this.database) {
      await this.database.close();
      this.database = null;
    }
    
    this.context = null;
    OfflineManager.instance = null;
  }

  /**
   * 设置上下文
   */
  public setContext(context: common.UIAbilityContext): void {
    this.context = context;
    logger.info('离线管理器上下文已设置');
  }

  /**
   * 初始化数据库
   */
  private async initializeDatabase(): Promise<void> {
    try {
      if (!this.context) {
        throw new Error('Context未设置，请先调用setContext方法');
      }

      logger.info('初始化离线数据库');

      const config: relationalStore.StoreConfig = {
        name: this.dbConfig.name,
        securityLevel: relationalStore.SecurityLevel.S1,
        encrypt: this.dbConfig.encrypted
      };

      this.database = await relationalStore.getRdbStore(this.context, config);

      // 创建离线数据表
      await this.createTables();

      logger.info('离线数据库初始化完成');

    } catch (error) {
      logger.error('初始化离线数据库失败', error);
      ErrorHandler.handle(error, 'OfflineManager.initializeDatabase', ErrorType.SYSTEM, ErrorLevel.HIGH);
      throw error;
    }
  }

  /**
   * 创建数据库表
   */
  private async createTables(): Promise<void> {
    if (!this.database) {
      throw new Error('数据库未初始化');
    }

    // 离线数据表
    const createOfflineDataTable = `
      CREATE TABLE IF NOT EXISTS offline_data (
        id TEXT PRIMARY KEY,
        type TEXT NOT NULL,
        action TEXT NOT NULL,
        data TEXT NOT NULL,
        timestamp INTEGER NOT NULL,
        retry_count INTEGER DEFAULT 0,
        priority INTEGER DEFAULT 5,
        user_id TEXT NOT NULL,
        created_at INTEGER DEFAULT (strftime('%s', 'now') * 1000),
        updated_at INTEGER DEFAULT (strftime('%s', 'now') * 1000)
      )
    `;

    // 同步日志表
    const createSyncLogTable = `
      CREATE TABLE IF NOT EXISTS sync_log (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        data_id TEXT NOT NULL,
        sync_time INTEGER NOT NULL,
        status TEXT NOT NULL,
        error_message TEXT,
        duration INTEGER DEFAULT 0
      )
    `;

    await this.database.executeSql(createOfflineDataTable);
    await this.database.executeSql(createSyncLogTable);

    logger.info('数据库表创建完成');
  }

  /**
   * 开始网络监控
   */
  private async startNetworkMonitoring(): Promise<void> {
    try {
      logger.info('开始网络状态监控');

      // 获取初始网络状态
      await this.updateNetworkStatus();

      // 注册网络状态变化监听
      connection.on('netConnectionChange', async (netHandle) => {
        logger.info('网络状态发生变化');
        await this.updateNetworkStatus();
      });

      // 定期检查网络质量
      this.networkMonitorTimer = setInterval(async () => {
        await this.updateNetworkQuality();
      }, 10000); // 每10秒检查一次

      logger.info('网络状态监控已启动');

    } catch (error) {
      logger.error('启动网络监控失败', error);
      ErrorHandler.handle(error, 'OfflineManager.startNetworkMonitoring', ErrorType.SYSTEM, ErrorLevel.MEDIUM);
    }
  }

  /**
   * 停止网络监控
   */
  private stopNetworkMonitoring(): void {
    try {
      connection.off('netConnectionChange');
      
      if (this.networkMonitorTimer !== -1) {
        clearInterval(this.networkMonitorTimer);
        this.networkMonitorTimer = -1;
      }

      logger.info('网络状态监控已停止');

    } catch (error) {
      logger.error('停止网络监控失败', error);
    }
  }

  /**
   * 更新网络状态
   */
  private async updateNetworkStatus(): Promise<void> {
    try {
      const netHandle = await connection.getDefaultNet();
      const netCapabilities = await connection.getNetCapabilities(netHandle);
      
      const oldStatus = this.networkStatus;
      
      if (netCapabilities.hasCapability(connection.NetCap.NET_CAPABILITY_INTERNET)) {
        this.networkStatus = NetworkStatus.ONLINE;
      } else {
        this.networkStatus = NetworkStatus.OFFLINE;
      }

      // 如果网络状态发生变化，触发事件
      if (oldStatus !== this.networkStatus) {
        logger.info(`网络状态变化: ${oldStatus} -> ${this.networkStatus}`);
        
        this.eventBus.emit('networkStatusChanged', {
          oldStatus,
          newStatus: this.networkStatus
        });

        // 如果从离线变为在线，触发同步
        if (oldStatus === NetworkStatus.OFFLINE && this.networkStatus === NetworkStatus.ONLINE) {
          await this.triggerSync();
        }
      }

    } catch (error) {
      logger.error('更新网络状态失败', error);
      this.networkStatus = NetworkStatus.OFFLINE;
    }
  }

  /**
   * 更新网络质量
   */
  private async updateNetworkQuality(): Promise<void> {
    try {
      if (this.networkStatus === NetworkStatus.OFFLINE) {
        this.networkQuality = null;
        return;
      }

      const netHandle = await connection.getDefaultNet();
      const netCapabilities = await connection.getNetCapabilities(netHandle);
      
      // 获取网络类型
      let netBearType = connection.NetBearType.BEARER_CELLULAR;
      if (netCapabilities.bearerTypes.includes(connection.NetBearType.BEARER_WIFI)) {
        netBearType = connection.NetBearType.BEARER_WIFI;
      }

      // 简单的网络质量评估（实际项目中可能需要更复杂的算法）
      let strength = 80; // 默认强度
      let latency = 50;  // 默认延迟

      // 根据网络类型调整
      if (netBearType === connection.NetBearType.BEARER_WIFI) {
        strength = 90;
        latency = 20;
      } else if (netBearType === connection.NetBearType.BEARER_CELLULAR) {
        strength = 70;
        latency = 100;
      }

      this.networkQuality = {
        status: strength > 60 ? NetworkStatus.ONLINE : NetworkStatus.POOR,
        type: netBearType,
        strength,
        latency
      };

      // 更新网络状态
      if (this.networkQuality.status !== this.networkStatus) {
        this.networkStatus = this.networkQuality.status;
        this.eventBus.emit('networkQualityChanged', this.networkQuality);
      }

    } catch (error) {
      logger.error('更新网络质量失败', error);
    }
  }

  /**
   * 添加离线数据
   */
  public async addOfflineData(item: Omit<OfflineDataItem, 'retryCount'>): Promise<void> {
    try {
      if (!this.database) {
        throw new Error('数据库未初始化');
      }

      const offlineItem: OfflineDataItem = {
        ...item,
        retryCount: 0
      };

      logger.info(`添加离线数据: ${item.type} - ${item.action}`);

      const sql = `
        INSERT INTO offline_data (id, type, action, data, timestamp, retry_count, priority, user_id)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?)
      `;

      await this.database.executeSql(sql, [
        offlineItem.id,
        offlineItem.type,
        offlineItem.action,
        JSON.stringify(offlineItem.data),
        offlineItem.timestamp,
        offlineItem.retryCount,
        offlineItem.priority,
        offlineItem.userId
      ]);

      // 更新统计信息
      this.syncStats.totalItems++;
      this.syncStats.pendingItems++;

      // 如果网络可用，立即尝试同步
      if (this.networkStatus === NetworkStatus.ONLINE) {
        await this.triggerSync();
      }

      logger.info(`离线数据已添加: ${item.id}`);

    } catch (error) {
      logger.error('添加离线数据失败', error);
      ErrorHandler.handle(error, 'OfflineManager.addOfflineData', ErrorType.SYSTEM, ErrorLevel.HIGH);
      throw error;
    }
  }

  /**
   * 获取待同步数据
   */
  public async getPendingData(limit: number = 10): Promise<OfflineDataItem[]> {
    try {
      if (!this.database) {
        throw new Error('数据库未初始化');
      }

      const sql = `
        SELECT * FROM offline_data 
        WHERE retry_count < ? 
        ORDER BY priority DESC, timestamp ASC 
        LIMIT ?
      `;

      const resultSet = await this.database.querySql(sql, [this.syncConfig.maxRetryCount, limit]);
      const items: OfflineDataItem[] = [];

      if (resultSet.goToFirstRow()) {
        do {
          const item: OfflineDataItem = {
            id: resultSet.getString(resultSet.getColumnIndex('id')),
            type: resultSet.getString(resultSet.getColumnIndex('type')) as OfflineDataType,
            action: resultSet.getString(resultSet.getColumnIndex('action')) as 'create' | 'update' | 'delete',
            data: JSON.parse(resultSet.getString(resultSet.getColumnIndex('data'))),
            timestamp: resultSet.getLong(resultSet.getColumnIndex('timestamp')),
            retryCount: resultSet.getLong(resultSet.getColumnIndex('retry_count')),
            priority: resultSet.getLong(resultSet.getColumnIndex('priority')),
            userId: resultSet.getString(resultSet.getColumnIndex('user_id'))
          };
          items.push(item);
        } while (resultSet.goToNextRow());
      }

      resultSet.close();
      return items;

    } catch (error) {
      logger.error('获取待同步数据失败', error);
      ErrorHandler.handle(error, 'OfflineManager.getPendingData', ErrorType.SYSTEM, ErrorLevel.MEDIUM);
      return [];
    }
  }

  /**
   * 触发同步
   */
  public async triggerSync(): Promise<void> {
    if (this.syncStatus === SyncStatus.SYNCING || this.networkStatus !== NetworkStatus.ONLINE) {
      return;
    }

    try {
      this.syncStatus = SyncStatus.SYNCING;
      const startTime = Date.now();

      logger.info('开始数据同步');

      // 获取待同步数据
      const pendingItems = await this.getPendingData(this.syncConfig.batchSize);
      
      if (pendingItems.length === 0) {
        this.syncStatus = SyncStatus.SUCCESS;
        logger.info('没有待同步数据');
        return;
      }

      logger.info(`开始同步 ${pendingItems.length} 项数据`);

      let successCount = 0;
      let failedCount = 0;

      // 逐个同步数据
      for (const item of pendingItems) {
        try {
          await this.syncDataItem(item);
          await this.removeOfflineData(item.id);
          successCount++;
          
          // 记录同步日志
          await this.logSyncResult(item.id, 'success', null, Date.now() - startTime);

        } catch (error) {
          failedCount++;
          await this.incrementRetryCount(item.id);
          
          // 记录同步日志
          await this.logSyncResult(item.id, 'failed', error.message, Date.now() - startTime);
          
          logger.error(`同步数据项失败: ${item.id}`, error);
        }
      }

      // 更新统计信息
      this.syncStats.successItems += successCount;
      this.syncStats.failedItems += failedCount;
      this.syncStats.pendingItems -= successCount;
      this.syncStats.lastSyncTime = Date.now();
      this.syncStats.syncDuration = Date.now() - startTime;

      this.syncStatus = failedCount === 0 ? SyncStatus.SUCCESS : SyncStatus.FAILED;

      logger.info(`数据同步完成: 成功 ${successCount}, 失败 ${failedCount}`);

      // 触发同步完成事件
      this.eventBus.emit('syncCompleted', {
        successCount,
        failedCount,
        duration: this.syncStats.syncDuration
      });

    } catch (error) {
      this.syncStatus = SyncStatus.FAILED;
      logger.error('数据同步失败', error);
      ErrorHandler.handle(error, 'OfflineManager.triggerSync', ErrorType.SYSTEM, ErrorLevel.HIGH);
    }
  }

  /**
   * 同步单个数据项（需要根据实际业务实现）
   */
  private async syncDataItem(item: OfflineDataItem): Promise<void> {
    // 这里需要根据实际的API调用来实现
    // 示例实现：
    logger.info(`同步数据项: ${item.type} - ${item.action} - ${item.id}`);
    
    // 模拟API调用
    await new Promise(resolve => setTimeout(resolve, 100));
    
    // 实际实现中，这里应该调用相应的API
    // 例如：
    // switch (item.type) {
    //   case OfflineDataType.MESSAGE:
    //     await this.syncMessage(item);
    //     break;
    //   case OfflineDataType.MOMENT:
    //     await this.syncMoment(item);
    //     break;
    //   // ... 其他类型
    // }
  }

  /**
   * 移除离线数据
   */
  private async removeOfflineData(id: string): Promise<void> {
    if (!this.database) {
      return;
    }

    const sql = 'DELETE FROM offline_data WHERE id = ?';
    await this.database.executeSql(sql, [id]);
  }

  /**
   * 增加重试次数
   */
  private async incrementRetryCount(id: string): Promise<void> {
    if (!this.database) {
      return;
    }

    const sql = 'UPDATE offline_data SET retry_count = retry_count + 1, updated_at = ? WHERE id = ?';
    await this.database.executeSql(sql, [Date.now(), id]);
  }

  /**
   * 记录同步日志
   */
  private async logSyncResult(
    dataId: string,
    status: string,
    errorMessage: string | null,
    duration: number
  ): Promise<void> {
    if (!this.database) {
      return;
    }

    const sql = `
      INSERT INTO sync_log (data_id, sync_time, status, error_message, duration)
      VALUES (?, ?, ?, ?, ?)
    `;

    await this.database.executeSql(sql, [
      dataId,
      Date.now(),
      status,
      errorMessage,
      duration
    ]);
  }

  /**
   * 加载同步统计信息
   */
  private async loadSyncStats(): Promise<void> {
    try {
      if (!this.database) {
        return;
      }

      // 获取总数据项
      const totalResult = await this.database.querySql('SELECT COUNT(*) as count FROM offline_data');
      if (totalResult.goToFirstRow()) {
        this.syncStats.totalItems = totalResult.getLong(0);
      }
      totalResult.close();

      // 获取待同步项
      const pendingResult = await this.database.querySql(
        'SELECT COUNT(*) as count FROM offline_data WHERE retry_count < ?',
        [this.syncConfig.maxRetryCount]
      );
      if (pendingResult.goToFirstRow()) {
        this.syncStats.pendingItems = pendingResult.getLong(0);
      }
      pendingResult.close();

      logger.info(`同步统计加载完成: 总计 ${this.syncStats.totalItems}, 待同步 ${this.syncStats.pendingItems}`);

    } catch (error) {
      logger.error('加载同步统计失败', error);
    }
  }

  /**
   * 开始自动同步
   */
  private startAutoSync(): void {
    if (this.syncTimer !== -1) {
      return;
    }

    this.syncTimer = setInterval(async () => {
      if (this.networkStatus === NetworkStatus.ONLINE && this.syncStatus !== SyncStatus.SYNCING) {
        await this.triggerSync();
      }
    }, this.syncConfig.syncInterval);

    logger.info(`自动同步已启动，间隔: ${this.syncConfig.syncInterval}ms`);
  }

  /**
   * 停止自动同步
   */
  private stopAutoSync(): void {
    if (this.syncTimer !== -1) {
      clearInterval(this.syncTimer);
      this.syncTimer = -1;
      logger.info('自动同步已停止');
    }
  }

  /**
   * 获取网络状态
   */
  public getNetworkStatus(): NetworkStatus {
    return this.networkStatus;
  }

  /**
   * 获取网络质量
   */
  public getNetworkQuality(): NetworkQuality | null {
    return this.networkQuality;
  }

  /**
   * 获取同步状态
   */
  public getSyncStatus(): SyncStatus {
    return this.syncStatus;
  }

  /**
   * 获取同步统计信息
   */
  public getSyncStats(): SyncStats {
    return { ...this.syncStats };
  }

  /**
   * 更新同步配置
   */
  public updateSyncConfig(config: Partial<SyncConfig>): void {
    this.syncConfig = { ...this.syncConfig, ...config };
    
    // 重启自动同步
    if (this.syncConfig.enableAutoSync) {
      this.stopAutoSync();
      this.startAutoSync();
    } else {
      this.stopAutoSync();
    }

    logger.info('同步配置已更新', this.syncConfig);
  }

  /**
   * 清理过期的同步日志
   */
  public async cleanupSyncLogs(daysToKeep: number = 7): Promise<void> {
    try {
      if (!this.database) {
        return;
      }

      const cutoffTime = Date.now() - (daysToKeep * 24 * 60 * 60 * 1000);
      const sql = 'DELETE FROM sync_log WHERE sync_time < ?';
      
      await this.database.executeSql(sql, [cutoffTime]);
      logger.info(`同步日志清理完成，保留 ${daysToKeep} 天`);

    } catch (error) {
      logger.error('清理同步日志失败', error);
      ErrorHandler.handle(error, 'OfflineManager.cleanupSyncLogs', ErrorType.SYSTEM, ErrorLevel.LOW);
    }
  }
}
