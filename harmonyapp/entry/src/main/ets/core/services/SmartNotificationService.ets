/**
 * 智能通知服务
 * 协调通知数据管理、调度和渲染的统一服务
 * 使用组合模式，将功能拆分为专门的子服务
 */

import { BaseSingleton } from '../base/Singleton';
import { createLogger } from '../utils/LogManager';
import { TimerManager } from '../utils/TimerManager';
import { ErrorHandler, ErrorType, ErrorLevel } from '../utils/ErrorHandler';
import { ConfigMerger } from '../utils/ConfigMerger';
import { NOTIFICATION_CONSTANTS } from '../../constants/SystemConstants';

// 导入子服务
import { 
  NotificationDataManager, 
  SmartNotification, 
  NotificationAction, 
  NotificationFilter,
  NotificationStats
} from './notification/NotificationDataManager';
import { 
  NotificationScheduler, 
  SchedulerConfig,
  SchedulerConfigUpdate,
  SchedulerStats,
  QuietHours
} from './notification/NotificationScheduler';
import { 
  NotificationRenderer, 
  RendererConfig, 
  RendererConfigUpdate,
  NotificationPosition, 
  NotificationAnimation,
  RendererStats
} from './notification/NotificationRenderer';

// 重新导出类型供外部使用
export { 
  SmartNotification, 
  NotificationAction, 
  NotificationFilter, 
  NotificationStats 
} from './notification/NotificationDataManager';

const logger = createLogger('SmartNotificationService');

/**
 * 统一配置接口
 */
export interface NotificationConfig {
  // 功能开关
  enablePerformanceAlerts: boolean;
  enableSocialNotifications: boolean;
  enableSecurityAlerts: boolean;
  
  // 数据管理配置
  maxNotifications: number;
  autoCleanupDays: number;
  
  // 调度配置
  quietHours: QuietHours;
  batchDelay: number;
  enableThrottling: boolean;
  
  // 渲染配置
  position: NotificationPosition;
  animation: NotificationAnimation;
  duration: number;
  maxVisible: number;
  enableSound: boolean;
  enableVibration: boolean;
  theme: 'light' | 'dark' | 'auto';
}

/**
 * 综合统计信息
 */
export interface ComprehensiveStats {
  data: NotificationStats;
  scheduler: SchedulerStats;
  renderer: RendererStats;
  overall: OverallStatsData;
}

interface OverallStatsData {
  totalProcessed: number;
  successRate: number;
  averageProcessingTime: number;
  peakConcurrency: number;
}

export interface CreateNotificationOptions {
  immediate?: boolean;
  delay?: number;
  callback?: (success: boolean, notificationId?: string) => void;
}

interface PerformanceAlertInput {
  type: string;
  level: 'info' | 'warning' | 'critical';
  message: string;
  value: number;
  threshold: number;
}

interface SecurityAlertInput {
  type: 'login_attempt' | 'permission_change' | 'data_access';
  severity: 'low' | 'medium' | 'high' | 'critical';
  description: string;
  location?: string;
}

class AlertData {
  type: string = '';
  level?: string;
  severity?: string;
  value?: number;
  threshold?: number;
  location?: string;
}

/**
 * 通知事件类型
 */
export enum NotificationEvent {
  CREATED = 'created',
  UPDATED = 'updated',
  DELETED = 'deleted',
  SHOWN = 'shown',
  HIDDEN = 'hidden',
  CLICKED = 'clicked',
  ACTION_EXECUTED = 'action_executed'
}

/**
 * 通知元数据类
 */
export class NotificationMetadata {
  type?: string;
  user?: Object;
  group?: Object;
  moment_id?: string;
  group_id?: string;
  comment_id?: string;
  is_mutual?: boolean;
  alert?: Object;
  config?: Object;
  fromUser?: string;
  targetId?: string;
}

/**
 * 事件额外数据类
 */
class EventExtra {
  id?: string;
  notification?: SmartNotification;
}

/**
 * 性能指标接口
 */
interface PerformanceMetrics {
  totalProcessed: number;
  successCount: number;
  errorCount: number;
  totalProcessingTime: number;
  peakConcurrency: number;
  currentConcurrency: number;
}

/**
 * 创建通知输入参数（不包含 id 和 timestamp）
 */
export interface CreateNotificationInput {
  type: 'system' | 'performance' | 'social' | 'security';
  priority: 'low' | 'medium' | 'high' | 'critical';
  title: string;
  message: string;
  read: boolean;
  actionable: boolean;
  actions?: NotificationAction[];
  metadata?: NotificationMetadata;
  expiresAt?: number;
}

/**
 * 事件监听器类型
 */
export type NotificationEventListener = (
  event: NotificationEvent,
  notification: SmartNotification,
  extra?: Object
) => void;

/**
 * 智能通知服务
 */
export class SmartNotificationService extends BaseSingleton {
  private static instance: SmartNotificationService | null = null;
  
  /**
   * 子服务实例
   */
  private dataManager: NotificationDataManager | null = null;
  private scheduler: NotificationScheduler | null = null;
  private renderer: NotificationRenderer | null = null;
  
  /**
   * 统一配置
   */
  private config: NotificationConfig;
  
  /**
   * 事件监听器
   */
  private eventListeners = new Map<NotificationEvent, NotificationEventListener[]>();
  
  /**
   * 性能监控
   */
  private performanceMetrics: PerformanceMetrics = {
    totalProcessed: 0,
    successCount: 0,
    errorCount: 0,
    totalProcessingTime: 0,
    peakConcurrency: 0,
    currentConcurrency: 0
  };
  
  /**
   * 清理定时器
   */
  private cleanupTimerId: number = -1;
  
  private constructor() {
    super();
    
    // 初始化默认配置
    this.config = {
      // 功能开关
      enablePerformanceAlerts: true,
      enableSocialNotifications: true,
      enableSecurityAlerts: true,
      
      // 数据管理配置
      maxNotifications: NOTIFICATION_CONSTANTS.LIMITS.MAX_NOTIFICATIONS,
      autoCleanupDays: NOTIFICATION_CONSTANTS.CLEANUP.AUTO_CLEANUP_DAYS,
      
      // 调度配置
      quietHours: {
        enabled: false,
        startTime: '22:00',
        endTime: '08:00'
      },
      batchDelay: 1000,
      enableThrottling: true,
      
      // 渲染配置
      position: NotificationPosition.TOP,
      animation: NotificationAnimation.SLIDE,
      duration: 5000,
      maxVisible: 3,
      enableSound: true,
      enableVibration: false,
      theme: 'auto'
    };
  }
  
  /**
   * 获取单例实例
   */
  static getInstance(): SmartNotificationService {
    if (!SmartNotificationService.instance) {
      SmartNotificationService.instance = new SmartNotificationService();
    }
    return SmartNotificationService.instance;
  }
  
  /**
   * 异步获取单例实例
   */
  static async getInstanceAsync(): Promise<SmartNotificationService> {
    const instance = SmartNotificationService.getInstance();
    await instance.initialize();
    return instance;
  }
  
  /**
   * 初始化
   */
  protected async onInitialize(): Promise<void> {
    logger.info('智能通知服务初始化开始');
    
    try {
      // 初始化子服务
      await this.initializeSubServices();
      
      // 配置子服务
      this.configureSubServices();
      
      // 设置事件监听
      this.setupEventListeners();
      
      // 启动清理定时器
      this.startCleanupTimer();
      
      logger.info('智能通知服务初始化完成');
    } catch (error) {
      logger.error('智能通知服务初始化失败', error);
      ErrorHandler.handle(error, 'SmartNotificationService.onInitialize', ErrorType.SYSTEM, ErrorLevel.CRITICAL);
      throw (error instanceof Error) ? error : new Error(String(error));
    }
  }
  
  /**
   * 清理
   */
  protected async onCleanup(): Promise<void> {
    logger.info('智能通知服务开始清理');
    
    try {
      // 停止清理定时器
      if (this.cleanupTimerId !== -1) {
        TimerManager.clearInterval(this.cleanupTimerId);
        this.cleanupTimerId = -1;
      }
      
      // 清理子服务
      if (this.renderer) {
        await this.renderer.cleanup();
        this.renderer = null;
      }
      
      if (this.scheduler) {
        await this.scheduler.cleanup();
        this.scheduler = null;
      }
      
      if (this.dataManager) {
        await this.dataManager.cleanup();
        this.dataManager = null;
      }
      
      // 清理事件监听器
      this.eventListeners.clear();
      
      SmartNotificationService.instance = null;
      logger.info('智能通知服务清理完成');
    } catch (error) {
      logger.error('智能通知服务清理失败', error);
    }
  }
  
  /**
   * 更新配置
   */
  updateConfig(config: Object): void {
    try {
      this.config = ConfigMerger.merge(this.config as Object as Record<string, Object>, config as Record<string, Object>) as Object as NotificationConfig;
      
      // 更新子服务配置
      this.configureSubServices();
      
      logger.debug('通知服务配置已更新');
    } catch (error) {
      ErrorHandler.handle(error, 'SmartNotificationService.updateConfig', ErrorType.BUSINESS, ErrorLevel.MEDIUM);
    }
  }
  
  /**
   * 获取配置
   */
  getConfig(): NotificationConfig {
    const result: NotificationConfig = {
      enablePerformanceAlerts: this.config.enablePerformanceAlerts,
      enableSocialNotifications: this.config.enableSocialNotifications,
      enableSecurityAlerts: this.config.enableSecurityAlerts,
      maxNotifications: this.config.maxNotifications,
      autoCleanupDays: this.config.autoCleanupDays,
      quietHours: this.config.quietHours,
      batchDelay: this.config.batchDelay,
      enableThrottling: this.config.enableThrottling,
      position: this.config.position,
      animation: this.config.animation,
      duration: this.config.duration,
      maxVisible: this.config.maxVisible,
      enableSound: this.config.enableSound,
      enableVibration: this.config.enableVibration,
      theme: this.config.theme
    };
    return result;
  }
  
  /**
   * 创建通知
   */
  async createNotification(
    notification: CreateNotificationInput,
    options?: CreateNotificationOptions
  ): Promise<string | null> {
    const startTime = Date.now();
    this.performanceMetrics.currentConcurrency++;
    this.performanceMetrics.peakConcurrency = Math.max(
      this.performanceMetrics.peakConcurrency,
      this.performanceMetrics.currentConcurrency
    );
    
    try {
      // 检查功能开关
      if (!this.isNotificationTypeEnabled(notification.type)) {
        logger.debug(`通知类型已禁用: ${notification.type}`);
        return null;
      }
      
      let notificationId: string | null = null;
      let fullNotification: SmartNotification | null = null;
      
      // 确保调度器已初始化
      if (!this.scheduler) {
        await this.initializeSubServices();
        if (!this.scheduler) {
          throw new Error('NotificationScheduler not initialized');
        }
      }

      if (options?.immediate) {
        // 立即发送
        fullNotification = {
          id: this.generateId(),
          timestamp: Date.now(),
          type: notification.type,
          priority: notification.priority,
          title: notification.title,
          message: notification.message,
          read: notification.read,
          actionable: notification.actionable,
          actions: notification.actions,
          metadata: notification.metadata,
          expiresAt: notification.expiresAt
        };
        notificationId = await this.scheduler!.sendImmediately(fullNotification, options.callback);
      } else if (options?.delay) {
        // 延迟发送
        fullNotification = {
          id: this.generateId(),
          timestamp: Date.now(),
          type: notification.type,
          priority: notification.priority,
          title: notification.title,
          message: notification.message,
          read: notification.read,
          actionable: notification.actionable,
          actions: notification.actions,
          metadata: notification.metadata,
          expiresAt: notification.expiresAt
        };
        const taskId = this.scheduler!.schedule(fullNotification, options.delay, options.callback);
        // 对于调度任务，我们返回任务ID作为通知ID
        notificationId = taskId;
      } else {
        // 默认立即发送
        fullNotification = {
          id: this.generateId(),
          timestamp: Date.now(),
          type: notification.type,
          priority: notification.priority,
          title: notification.title,
          message: notification.message,
          read: notification.read,
          actionable: notification.actionable,
          actions: notification.actions,
          metadata: notification.metadata,
          expiresAt: notification.expiresAt
        };
        notificationId = await this.scheduler!.sendImmediately(fullNotification, options?.callback);
      }
      
      if (notificationId && fullNotification) {
        // 触发创建事件
        const eventExtra = new EventExtra();
        eventExtra.id = notificationId;
        eventExtra.notification = fullNotification;
        this.emitEvent(NotificationEvent.CREATED, fullNotification, eventExtra);
        
        // 更新性能指标
        this.performanceMetrics.successCount++;
      } else {
        this.performanceMetrics.errorCount++;
      }
      
      this.performanceMetrics.totalProcessed++;
      
      return notificationId;
    } catch (error) {
      this.performanceMetrics.errorCount++;
      logger.error('创建通知失败', error);
      ErrorHandler.handle(error, 'SmartNotificationService.createNotification', ErrorType.BUSINESS, ErrorLevel.MEDIUM);
      throw (error instanceof Error) ? error : new Error(String(error));
    } finally {
      this.performanceMetrics.currentConcurrency--;
      const duration = Date.now() - startTime;
      this.performanceMetrics.totalProcessingTime += duration;
      if (duration > 1000) {
        logger.warn(`创建通知执行时间过长: ${duration}ms`);
      }
    }
  }
  
  /**
   * 获取通知
   */
  getNotification(id: string): SmartNotification | null {
    return this.dataManager ? this.dataManager.getNotification(id) : null;
  }
  
  /**
   * 更新通知
   */
  async updateNotification(id: string, updates: Partial<SmartNotification>): Promise<boolean> {
    try {
      if (!this.dataManager) return false;
      const success = await this.dataManager.updateNotification(id, updates);
      
      if (success) {
        const updatedNotification = this.dataManager.getNotification(id);
        if (updatedNotification) {
          // 更新渲染
          if (this.renderer) {
              await this.renderer.updateNotification(id, updates);
          }
          
          // 触发更新事件
          this.emitEvent(NotificationEvent.UPDATED, updatedNotification, updates);
        }
      }
      
      return success;
    } catch (error) {
      ErrorHandler.handle(error, 'SmartNotificationService.updateNotification', ErrorType.BUSINESS, ErrorLevel.MEDIUM);
      return false;
    }
  }
  
  /**
   * 删除通知
   */
  async deleteNotification(id: string): Promise<boolean> {
    try {
      if (!this.dataManager) return false;
      const notification = this.dataManager.getNotification(id);
      
      // 先隐藏通知
      if (this.renderer) {
          await this.renderer.hideNotification(id, 'manual');
      }
      
      // 然后从数据中删除
      const success = await this.dataManager.deleteNotification(id);
      
      if (success && notification) {
        // 触发删除事件
        this.emitEvent(NotificationEvent.DELETED, notification);
      }
      
      return success;
    } catch (error) {
      ErrorHandler.handle(error, 'SmartNotificationService.deleteNotification', ErrorType.BUSINESS, ErrorLevel.LOW);
      return false;
    }
  }
  
  /**
   * 获取通知列表
   */
  getNotifications(filter?: NotificationFilter): SmartNotification[] {
    return this.dataManager ? this.dataManager.getNotifications(filter) : [];
  }
  
  /**
   * 获取未读通知
   */
  getUnreadNotifications(): SmartNotification[] {
    return this.dataManager ? this.dataManager.getUnreadNotifications() : [];
  }

  /**
   * 标记为已读
   */
  async markAsRead(id: string): Promise<boolean> {
    return this.dataManager ? this.dataManager.markAsRead(id) : false;
  }
  
  /**
   * 批量标记为已读
   */
  async markAllAsRead(filter?: NotificationFilter): Promise<number> {
    return this.dataManager ? this.dataManager.markAllAsRead(filter) : 0;
  }
  
  /**
   * 显示通知
   */
  async showNotification(id: string): Promise<boolean> {
    try {
      if (!this.dataManager || !this.renderer) return false;

      const notification = this.dataManager.getNotification(id);
      if (!notification) {
        return false;
      }
      
      await this.renderer.showNotification(notification);
      
      // 触发显示事件
      this.emitEvent(NotificationEvent.SHOWN, notification);
      
      return true;
    } catch (error) {
      logger.error(`显示通知失败: ${id}`, error);
      ErrorHandler.handle(error, 'SmartNotificationService.showNotification', ErrorType.BUSINESS, ErrorLevel.MEDIUM);
      return false;
    }
  }
  
  /**
   * 隐藏通知
   */
  async hideNotification(id: string): Promise<boolean> {
    if (!this.dataManager || !this.renderer) return false;

    const notification = this.dataManager.getNotification(id);
    const success = await this.renderer.hideNotification(id, 'manual');
    
    if (success && notification) {
      // 触发隐藏事件
      this.emitEvent(NotificationEvent.HIDDEN, notification);
    }
    
    return success;
  }
  
  /**
   * 隐藏所有通知
   */
  async hideAllNotifications(): Promise<void> {
    if (this.renderer) {
        await this.renderer.hideAllNotifications();
    }
  }
  
  /**
   * 获取综合统计信息
   */
  getComprehensiveStats(): ComprehensiveStats | null {
    if (!this.dataManager || !this.scheduler || !this.renderer) return null;

    const dataStats = this.dataManager.getStats();
    const schedulerStats = this.scheduler.getStats();
    const rendererStats = this.renderer.getStats();
    
    const successRate = this.performanceMetrics.totalProcessed > 0
      ? (this.performanceMetrics.successCount / this.performanceMetrics.totalProcessed) * 100
      : 0;
    
    const averageProcessingTime = this.performanceMetrics.totalProcessed > 0
      ? this.performanceMetrics.totalProcessingTime / this.performanceMetrics.totalProcessed
      : 0;
    
    const overallData: OverallStatsData = {
      totalProcessed: this.performanceMetrics.totalProcessed,
      successRate: successRate,
      averageProcessingTime: averageProcessingTime,
      peakConcurrency: this.performanceMetrics.peakConcurrency
    };
    const stats: ComprehensiveStats = {
      data: dataStats,
      scheduler: schedulerStats,
      renderer: rendererStats,
      overall: overallData
    };
    return stats;
  }
  
  /**
   * 清理过期通知
   */
  async cleanup(): Promise<void> {
    logger.info('开始清理过期通知');
    
    try {
      if (!this.dataManager) return;

      // 清理过期通知
      const expiredCount = await this.dataManager.cleanupExpiredNotifications();
      
      // 清理旧通知
      const oldCount = await this.dataManager.cleanupOldNotifications();
      
      logger.info(`清理完成: 过期通知 ${expiredCount}个, 旧通知 ${oldCount}个`);
    } catch (error) {
      ErrorHandler.handle(error, 'SmartNotificationService.cleanup', ErrorType.SYSTEM, ErrorLevel.LOW);
    }
  }
  
  /**
   * 添加事件监听器
   */
  addEventListener(event: NotificationEvent, listener: NotificationEventListener): void {
    if (!this.eventListeners.has(event)) {
      this.eventListeners.set(event, []);
    }
    
    this.eventListeners.get(event)!.push(listener);
  }
  
  /**
   * 移除事件监听器
   */
  removeEventListener(event: NotificationEvent, listener: NotificationEventListener): void {
    const listeners = this.eventListeners.get(event);
    if (listeners) {
      const index = listeners.indexOf(listener);
      if (index > -1) {
        listeners.splice(index, 1);
      }
    }
  }

  /**
   * 创建性能告警通知
   */
  async createPerformanceAlert(alert: PerformanceAlertInput): Promise<string | null> {
    if (!this.config.enablePerformanceAlerts) {
      return null;
    }
    
    const priority: 'low' | 'medium' | 'high' | 'critical' = alert.level === 'critical' ? 'critical' : 
                    alert.level === 'warning' ? 'high' : 'medium';
    
    const metadata = new NotificationMetadata();
    const alertData = new AlertData();
    alertData.type = alert.type;
    alertData.level = alert.level;
    alertData.value = alert.value;
    alertData.threshold = alert.threshold;
    metadata.alert = alertData as Object;
    
    const opts: CreateNotificationOptions = { immediate: true };
    return this.createNotification({
      type: 'performance',
      priority: priority,
      title: `性能告警: ${alert.type}`,
      message: alert.message,
      read: false,
      actionable: true,
      actions: [
        {
          id: 'view_details',
          label: '查看详情',
          type: 'primary',
          handler: async () => {
            logger.info(`查看性能告警详情: ${alert.type}`);
          }
        }
      ],
      metadata: metadata
    }, opts);
  }
  
  /**
   * 创建安全告警通知
   */
  async createSecurityAlert(security: SecurityAlertInput): Promise<string | null> {
    if (!this.config.enableSecurityAlerts) {
      return null;
    }
    
    const opts: CreateNotificationOptions = { immediate: true };
    return this.createNotification({
      type: 'security',
      priority: security.severity,
      title: '安全告警',
      message: security.description,
      read: false,
      actionable: true,
      actions: [
        {
          id: 'secure_account',
          label: '保护账户',
          type: 'danger',
          handler: async () => {
            logger.info(`执行账户保护: ${security.type}`);
          }
        }
      ],
      metadata: (() => {
        const meta = new NotificationMetadata();
        const ad = new AlertData();
        ad.type = security.type;
        ad.severity = security.severity;
        ad.location = security.location;
        meta.alert = ad as Object;
        return meta;
      })()
    }, opts);
  }

  /**
   * 初始化子服务
   */
  private async initializeSubServices(): Promise<void> {
    // 初始化数据管理器
    this.dataManager = await NotificationDataManager.getInstanceAsync();
    
    // 初始化调度器
    this.scheduler = await NotificationScheduler.getInstanceAsync();
    
    // 初始化渲染器
    this.renderer = await NotificationRenderer.getInstanceAsync();
    
    logger.debug('子服务初始化完成');
  }
  
  /**
   * 生成唯一通知ID
   */
  private generateId(): string {
    const timestamp = Date.now().toString(36);
    const random = Math.random().toString(36).substring(2, 8);
    return `notif_${timestamp}_${random}`;
  }

  /**
   * 配置子服务
   */
  private configureSubServices(): void {
    if (!this.dataManager || !this.scheduler || !this.renderer) {
      return;
    }
    
    // 配置调度器
    const schedulerConfig = new SchedulerConfigUpdate();
    schedulerConfig.quietHours = this.config.quietHours;
    schedulerConfig.batchDelay = this.config.batchDelay;
    schedulerConfig.enableThrottling = this.config.enableThrottling;
    this.scheduler.updateConfig(schedulerConfig);
    
    // 配置渲染器
    const rendererConfig = new RendererConfigUpdate();
    rendererConfig.position = this.config.position;
    rendererConfig.animation = this.config.animation;
    rendererConfig.duration = this.config.duration;
    rendererConfig.maxVisible = this.config.maxVisible;
    rendererConfig.enableSound = this.config.enableSound;
    rendererConfig.enableVibration = this.config.enableVibration;
    rendererConfig.theme = this.config.theme;
    this.renderer.updateConfig(rendererConfig);
    
    logger.debug('子服务配置完成');
  }
  
  /**
   * 设置事件监听
   */
  private setupEventListeners(): void {
    if (!this.dataManager || !this.renderer) {
      return;
    }
    
    // 监听数据变更，自动显示新通知
    this.dataManager.addChangeListener((notifications) => {
      // 这里可以添加自动显示逻辑
      logger.debug(`通知数据变更: ${notifications.length}个通知`);
    });
    
    // 设置渲染器事件监听
    this.renderer.setClickListener('*', (notification) => {
      this.emitEvent(NotificationEvent.CLICKED, notification);
    });
    
    this.renderer.setActionListener('*', (action, notification) => {
      this.emitEvent(NotificationEvent.ACTION_EXECUTED, notification, action);
    });
    
    logger.debug('事件监听设置完成');
  }
  
  /**
   * 启动清理定时器
   */
  private startCleanupTimer(): void {
    // 每小时执行一次清理
    this.cleanupTimerId = TimerManager.setInterval(() => {
      this.cleanup().catch(error => {
        logger.error('定时清理失败', error);
      });
    }, 60 * 60 * 1000, 'notification_cleanup');
  }
  
  /**
   * 检查通知类型是否启用
   */
  private isNotificationTypeEnabled(type: string): boolean {
    switch (type) {
      case 'performance':
        return this.config.enablePerformanceAlerts;
      case 'social':
        return this.config.enableSocialNotifications;
      case 'security':
        return this.config.enableSecurityAlerts;
      case 'system':
        return true; // 系统通知总是启用
      default:
        return true;
    }
  }
  
  /**
   * 触发事件
   */
  private emitEvent(event: NotificationEvent, notification: SmartNotification, extra?: Object): void {
    const listeners = this.eventListeners.get(event);
    if (listeners) {
      for (const listener of listeners) {
        try {
          listener(event, notification, extra);
        } catch (error) {
          logger.error(`事件监听器执行失败: ${event}`, error);
        }
      }
    }
  }
}
