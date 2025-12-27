/**
 * 通知调度器
 * 专门负责通知的调度、定时发送和静默时间管理
 */

import { BaseSingleton } from '../../base/Singleton';
import { createLogger } from '../../utils/LogManager';
import { TimerManager } from '../../utils/TimerManager';
import { ErrorHandler, ErrorType, ErrorLevel } from '../../utils/ErrorHandler';
import { ConfigMerger } from '../../utils/ConfigMerger';
import { NOTIFICATION_CONSTANTS } from '../../constants/SystemConstants';
import { SmartNotification, NotificationDataManager } from './NotificationDataManager';

const logger = createLogger('NotificationScheduler');

/**
 * 静默时间配置
 */
interface QuietHours {
  enabled: boolean;
  startTime: string; // HH:MM
  endTime: string;   // HH:MM
}

/**
 * 调度配置
 */
export interface SchedulerConfig {
  quietHours: QuietHours;
  batchDelay: number;           // 批量发送延迟（毫秒）
  maxBatchSize: number;         // 最大批量大小
  retryAttempts: number;        // 重试次数
  retryDelay: number;           // 重试延迟（毫秒）
  enableThrottling: boolean;    // 启用限流
  throttleWindow: number;       // 限流时间窗口（毫秒）
  maxNotificationsPerWindow: number; // 时间窗口内最大通知数
}

// Manual definition to avoid Partial<T> utility type
export interface SchedulerConfigUpdate {
  quietHours?: QuietHours;
  batchDelay?: number;
  maxBatchSize?: number;
  retryAttempts?: number;
  retryDelay?: number;
  enableThrottling?: boolean;
  throttleWindow?: number;
  maxNotificationsPerWindow?: number;
}

/**
 * 调度任务
 */
interface ScheduledTask {
  id: string;
  notification: SmartNotification;
  scheduledTime: number;
  attempts: number;
  maxAttempts: number;
  callback?: (success: boolean, notificationId?: string) => void;
}

/**
 * 调度统计
 */
export interface SchedulerStats {
  totalScheduled: number;
  totalSent: number;
  totalFailed: number;
  pendingTasks: number;
  averageDelay: number;
  throttledCount: number;
  quietHoursBlocked: number;
}

/**
 * 通知调度器
 */
export class NotificationScheduler extends BaseSingleton {
  private static instance: NotificationScheduler | null = null;
  
  /**
   * 调度配置
   */
  private config: SchedulerConfig;
  
  /**
   * 待调度的任务
   */
  private scheduledTasks = new Map<string, ScheduledTask>();
  
  /**
   * 批量发送队列
   */
  private batchQueue: ScheduledTask[] = [];
  
  /**
   * 限流记录
   */
  private throttleHistory: number[] = [];
  
  /**
   * 定时器ID
   */
  private batchTimerId: number = -1;
  private cleanupTimerId: number = -1;
  
  /**
   * 数据管理器
   */
  private dataManager: NotificationDataManager | null = null;
  
  /**
   * 统计信息
   */
  private stats: SchedulerStats = {
    totalScheduled: 0,
    totalSent: 0,
    totalFailed: 0,
    pendingTasks: 0,
    averageDelay: 0,
    throttledCount: 0,
    quietHoursBlocked: 0
  };
  
  private constructor() {
    super();
    
    this.config = {
      quietHours: {
        enabled: false,
        startTime: '22:00',
        endTime: '08:00'
      },
      batchDelay: 1000,                    // 1秒批量延迟
      maxBatchSize: 10,                    // 最大批量10个
      retryAttempts: 3,                    // 重试3次
      retryDelay: 2000,                    // 重试延迟2秒
      enableThrottling: true,              // 启用限流
      throttleWindow: 60000,               // 1分钟时间窗口
      maxNotificationsPerWindow: 20        // 每分钟最多20个通知
    };
  }
  
  /**
   * 获取单例实例
   */
  static getInstance(): NotificationScheduler {
    if (!NotificationScheduler.instance) {
      NotificationScheduler.instance = new NotificationScheduler();
    }
    return NotificationScheduler.instance;
  }
  
  /**
   * 异步获取单例实例
   */
  static async getInstanceAsync(): Promise<NotificationScheduler> {
    const instance = NotificationScheduler.getInstance();
    await instance.initialize();
    return instance;
  }
  
  /**
   * 初始化
   */
  protected async onInitialize(): Promise<void> {
    logger.info('通知调度器初始化开始');
    
    try {
      // 获取数据管理器
      this.dataManager = await NotificationDataManager.getInstanceAsync();
      
      // 启动批量处理定时器
      this.startBatchProcessor();
      
      // 启动清理定时器
      this.startCleanupTimer();
      
      logger.info('通知调度器初始化完成');
    } catch (error) {
      ErrorHandler.handle(error, 'NotificationScheduler.onInitialize', ErrorType.SYSTEM, ErrorLevel.HIGH);
      throw new Error(String(error));
    }
  }
  
  /**
   * 清理
   */
  protected async onCleanup(): Promise<void> {
    logger.info('通知调度器开始清理');
    
    // 停止定时器
    if (this.batchTimerId !== -1) {
      TimerManager.clearInterval(this.batchTimerId);
      this.batchTimerId = -1;
    }
    
    if (this.cleanupTimerId !== -1) {
      TimerManager.clearInterval(this.cleanupTimerId);
      this.cleanupTimerId = -1;
    }
    
    // 处理剩余任务
    await this.processPendingTasks();
    
    // 清理数据
    this.scheduledTasks.clear();
    this.batchQueue = [];
    this.throttleHistory = [];
    this.dataManager = null;
    
    NotificationScheduler.instance = null;
    logger.info('通知调度器清理完成');
  }
  
  /**
   * 更新配置
   */
  updateConfig(config: SchedulerConfigUpdate): void {
    this.config = ConfigMerger.merge(this.config, config);
    logger.debug('调度器配置已更新');
  }
  
  /**
   * 获取配置
   */
  getConfig(): SchedulerConfig {
    return ConfigMerger.merge({} as SchedulerConfig, this.config);
  }
  
  /**
   * 立即发送通知
   */
  async sendImmediately(
    notification: SmartNotification,
    callback?: (success: boolean, notificationId?: string) => void
  ): Promise<string | null> {
    const startTime = Date.now();
    try {
      // 检查静默时间
      if (this.isInQuietHours()) {
        this.stats.quietHoursBlocked++;
        logger.debug('当前处于静默时间，通知被阻止');
        
        if (callback) {
          callback(false);
        }
        return null;
      }
      
      // 检查限流
      if (this.isThrottled()) {
        this.stats.throttledCount++;
        logger.debug('通知被限流');
        
        if (callback) {
          callback(false);
        }
        return null;
      }
      
      // 发送通知
      const notificationId = await this.dataManager!.addNotification(notification);
      
      // 记录限流历史
      this.recordThrottleHistory();
      
      // 更新统计
      this.stats.totalSent++;
      
      logger.debug(`通知立即发送成功: ${notificationId}`);
      
      if (callback) {
        callback(true, notificationId);
      }
      
      return notificationId;
    } catch (error) {
      this.stats.totalFailed++;
      ErrorHandler.handle(error, 'NotificationScheduler.sendImmediately', ErrorType.BUSINESS, ErrorLevel.MEDIUM);
      logger.error('立即发送通知失败', error);
      
      if (callback) {
        callback(false);
      }
      
      throw error;
    } finally {
      // 简单性能日志
      const duration = Date.now() - startTime;
      if (duration > 500) {
        logger.warn(`sendImmediately took ${duration}ms`);
      }
    }
  }
  
  /**
   * 调度通知（延迟发送）
   */
  schedule(
    notification: SmartNotification,
    delay: number,
    callback?: (success: boolean, notificationId?: string) => void
  ): string {
    try {
      const taskId = this.generateTaskId();
      const scheduledTime = Date.now() + delay;
      
      const task: ScheduledTask = {
        id: taskId,
        notification,
        scheduledTime,
        attempts: 0,
        maxAttempts: this.config.retryAttempts,
        callback
      };
      
      this.scheduledTasks.set(taskId, task);
      this.stats.totalScheduled++;
      this.stats.pendingTasks++;
      
      logger.debug(`通知已调度: ${taskId}, 延迟: ${delay}ms`);
      return taskId;
    } catch (error) {
      ErrorHandler.handle(error, 'NotificationScheduler.schedule', ErrorType.BUSINESS, ErrorLevel.MEDIUM);
      throw error;
    }
  }
  
  /**
   * 批量调度通知
   */
  scheduleBatch(
    notifications: Array<{
      notification: SmartNotification;
      delay: number;
      callback?: (success: boolean, notificationId?: string) => void;
    }>
  ): string[] {
    try {
      const taskIds: string[] = [];
      
      for (const item of notifications) {
        const taskId = this.schedule(item.notification, item.delay, item.callback);
        taskIds.push(taskId);
      }
      
      logger.debug(`批量调度通知: ${taskIds.length}个`);
      return taskIds;
    } catch (error) {
      ErrorHandler.handle(error, 'NotificationScheduler.scheduleBatch', ErrorType.BUSINESS, ErrorLevel.MEDIUM);
      throw error;
    }
  }
  
  /**
   * 取消调度任务
   */
  cancelScheduledTask(taskId: string): boolean {
    const task = this.scheduledTasks.get(taskId);
    if (task) {
      this.scheduledTasks.delete(taskId);
      this.stats.pendingTasks--;
      
      logger.debug(`调度任务已取消: ${taskId}`);
      return true;
    }
    
    return false;
  }
  
  /**
   * 获取调度统计
   */
  getStats(): SchedulerStats {
    return {
      ...this.stats,
      pendingTasks: this.scheduledTasks.size
    };
  }
  
  /**
   * 清理统计数据
   */
  clearStats(): void {
    this.stats = {
      totalScheduled: 0,
      totalSent: 0,
      totalFailed: 0,
      pendingTasks: this.scheduledTasks.size,
      averageDelay: 0,
      throttledCount: 0,
      quietHoursBlocked: 0
    };
    
    logger.debug('调度器统计数据已清理');
  }
  
  /**
   * 检查是否在静默时间
   */
  private isInQuietHours(): boolean {
    if (!this.config.quietHours.enabled) {
      return false;
    }
    
    const now = new Date();
    const currentTime = `${now.getHours().toString().padStart(2, '0')}:${now.getMinutes().toString().padStart(2, '0')}`;
    
    const startTime = this.config.quietHours.startTime;
    const endTime = this.config.quietHours.endTime;
    
    // 处理跨天的情况
    if (startTime > endTime) {
      return currentTime >= startTime || currentTime <= endTime;
    } else {
      return currentTime >= startTime && currentTime <= endTime;
    }
  }
  
  /**
   * 检查是否被限流
   */
  private isThrottled(): boolean {
    if (!this.config.enableThrottling) {
      return false;
    }
    
    const now = Date.now();
    const windowStart = now - this.config.throttleWindow;
    
    // 清理过期的限流记录
    this.throttleHistory = this.throttleHistory.filter(time => time > windowStart);
    
    // 检查是否超过限制
    return this.throttleHistory.length >= this.config.maxNotificationsPerWindow;
  }
  
  /**
   * 记录限流历史
   */
  private recordThrottleHistory(): void {
    if (this.config.enableThrottling) {
      this.throttleHistory.push(Date.now());
    }
  }
  
  /**
   * 启动批量处理器
   */
  private startBatchProcessor(): void {
    this.batchTimerId = TimerManager.setInterval(() => {
      this.processBatch();
    }, this.config.batchDelay, 'notification_batch_processor');
  }
  
  /**
   * 启动清理定时器
   */
  private startCleanupTimer(): void {
    // 每5分钟清理一次过期任务
    this.cleanupTimerId = TimerManager.setInterval(() => {
      this.cleanupExpiredTasks();
    }, 5 * 60 * 1000, 'notification_cleanup');
  }
  
  /**
   * 处理批量任务
   */
  private async processBatch(): Promise<void> {
    try {
      const now = Date.now();
      const readyTasks: ScheduledTask[] = [];
      
      // 找出准备执行的任务
      for (const [taskId, task] of this.scheduledTasks.entries()) {
        if (task.scheduledTime <= now) {
          readyTasks.push(task);
          this.scheduledTasks.delete(taskId);
          this.stats.pendingTasks--;
        }
      }
      
      if (readyTasks.length === 0) {
        return;
      }
      
      // 按批量大小分组处理
      const batches = this.chunkArray(readyTasks, this.config.maxBatchSize);
      
      for (const batch of batches) {
        await this.processBatchGroup(batch);
      }
    } catch (error) {
      ErrorHandler.handle(error, 'NotificationScheduler.processBatch', ErrorType.SYSTEM, ErrorLevel.LOW);
    }
  }
  
  /**
   * 处理批量组
   */
  private async processBatchGroup(tasks: ScheduledTask[]): Promise<void> {
    const promises = tasks.map(task => this.executeTask(task));
    await Promise.all(promises);
  }
  
  /**
   * 执行单个任务
   */
  private async executeTask(task: ScheduledTask): Promise<void> {
    try {
      task.attempts++;
      
      const notificationId = await this.sendImmediately(task.notification);
      
      if (notificationId) {
        // 成功
        if (task.callback) {
          task.callback(true, notificationId);
        }
      } else {
        // 失败但不重试（静默时间或限流）
        if (task.callback) {
          task.callback(false);
        }
      }
    } catch (error) {
      // 执行失败，考虑重试
      if (task.attempts < task.maxAttempts) {
        // 重新调度重试
        const retryDelay = this.config.retryDelay * Math.pow(2, task.attempts - 1); // 指数退避
        task.scheduledTime = Date.now() + retryDelay;
        
        this.scheduledTasks.set(task.id, task);
        this.stats.pendingTasks++;
        
        logger.warn(`任务执行失败，将重试: ${task.id}, 尝试次数: ${task.attempts}/${task.maxAttempts}`);
      } else {
        // 重试次数用完，最终失败
        this.stats.totalFailed++;
        
        if (task.callback) {
          task.callback(false);
        }
        
        logger.error(`任务最终失败: ${task.id}`, error);
      }
    }
  }
  
  /**
   * 处理待处理任务（清理时调用）
   */
  private async processPendingTasks(): Promise<void> {
    const pendingTasks = Array.from(this.scheduledTasks.values());
    
    if (pendingTasks.length > 0) {
      logger.info(`处理剩余待调度任务: ${pendingTasks.length}个`);
      
      for (const task of pendingTasks) {
        try {
          await this.executeTask(task);
        } catch (error) {
          logger.error(`处理待调度任务失败: ${task.id}`, error);
        }
      }
    }
  }
  
  /**
   * 清理过期任务
   */
  private cleanupExpiredTasks(): void {
    const now = Date.now();
    const maxAge = 24 * 60 * 60 * 1000; // 24小时
    const expiredTaskIds: string[] = [];
    
    for (const [taskId, task] of this.scheduledTasks.entries()) {
      if (now - task.scheduledTime > maxAge) {
        expiredTaskIds.push(taskId);
      }
    }
    
    for (const taskId of expiredTaskIds) {
      this.scheduledTasks.delete(taskId);
      this.stats.pendingTasks--;
    }
    
    if (expiredTaskIds.length > 0) {
      logger.info(`清理过期调度任务: ${expiredTaskIds.length}个`);
    }
  }
  
  /**
   * 生成任务ID
   */
  private generateTaskId(): string {
    return `task_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
  }
  
  /**
   * 数组分块
   */
  private chunkArray<T>(array: T[], chunkSize: number): T[][] {
    const chunks: T[][] = [];
    
    for (let i = 0; i < array.length; i += chunkSize) {
      chunks.push(array.slice(i, i + chunkSize));
    }
    
    return chunks;
  }
}
