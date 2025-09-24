/**
 * 智能通知服务（重构版）
 * 使用新的基础设施：单例基类、统一日志、定时器管理器、错误处理、统一存储
 */
import { BusinessError } from '@kit.BasicServicesKit';
import { BaseSingleton } from '../base/Singleton';
import { createLogger } from '../utils/LogManager';
import { TimerManager } from '../utils/TimerManager';
import { ErrorHandler, ErrorType, ErrorLevel } from '../utils/ErrorHandler';
import { UnifiedStorageUtil } from '../utils/UnifiedStorageUtil';
import { PerformanceOptimizer, PerformanceAlert } from '../performance/PerformanceOptimizer';
import { AccessibilityManager } from '../accessibility/AccessibilityManager';

const logger = createLogger('SmartNotificationService');

export interface SmartNotification {
  id: string;
  type: 'system' | 'performance' | 'social' | 'security';
  priority: 'low' | 'medium' | 'high' | 'critical';
  title: string;
  message: string;
  timestamp: number;
  read: boolean;
  actionable: boolean;
  actions?: NotificationAction[];
  metadata?: NotificationMetadata;
  expiresAt?: number;
}

export interface NotificationAction {
  id: string;
  label: string;
  type: 'primary' | 'secondary' | 'danger';
  handler: () => Promise<void>;
}

interface QuietHours {
  enabled: boolean;
  startTime: string; // HH:MM
  endTime: string;   // HH:MM
}

export interface NotificationFilter {
  type?: 'system' | 'performance' | 'social' | 'security';
  priority?: 'low' | 'medium' | 'high' | 'critical';
  read?: boolean;
}

interface NotificationMetadata {
  alert?: Object;
  recommendation?: Object;
  recommendations?: Object;
  user?: Object;
  config?: Object;
  type?: string;
  fromUser?: string;
  targetId?: string;
}

export interface NotificationConfig {
  enablePerformanceAlerts: boolean;
  enableRecommendations: boolean;
  enableSocialNotifications: boolean;
  enableSecurityAlerts: boolean;
  quietHours: QuietHours;
  maxNotifications: number;
  autoCleanupDays: number;
}

export class SmartNotificationService extends BaseSingleton {
  private static instance: SmartNotificationService | null = null;
  private notifications: Map<string, SmartNotification> = new Map();
  private config: NotificationConfig;
  private listeners: ((notification: SmartNotification) => void)[] = [];
  private performanceOptimizer: PerformanceOptimizer;
  private accessibilityManager: AccessibilityManager;
  private storage: UnifiedStorageUtil | null = null;
  private readonly storageKey = 'smart_notifications';
  private readonly configKey = 'notification_config';
  private cleanupTimerId: number = -1;
  private performanceCheckTimerId: number = -1;
  private cleanupInterval: number = -1;

  private constructor() {
    super();
    this.config = {
      enablePerformanceAlerts: true,
      enableRecommendations: true,
      enableSocialNotifications: true,
      enableSecurityAlerts: true,
      quietHours: {
        enabled: false,
        startTime: '22:00',
        endTime: '08:00'
      } as QuietHours,
      maxNotifications: 100,
      autoCleanupDays: 7
    } as NotificationConfig;

    this.performanceOptimizer = PerformanceOptimizer.getInstance();
    this.accessibilityManager = AccessibilityManager.getInstance();
  }

  /**
   * 获取单例实例
   */
  public static getInstance(): SmartNotificationService {
    if (!SmartNotificationService.instance) {
      SmartNotificationService.instance = new SmartNotificationService();
    }
    return SmartNotificationService.instance;
  }

  /**
   * 异步获取单例实例
   */
  public static async getInstanceAsync(): Promise<SmartNotificationService> {
    const instance = SmartNotificationService.getInstance();
    await instance.initialize();
    return instance;
  }

  /**
   * 实现基类的初始化方法
   */
  protected async onInitialize(): Promise<void> {
    try {
      logger.info('智能通知服务初始化开始');
      
      // 获取存储实例
      this.storage = UnifiedStorageUtil.getInstance();
      
      // 加载配置和通知
      await this.loadConfig();
      await this.loadNotifications();
      
      // 启动性能监听
      this.setupPerformanceListener();
      
      // 启动清理任务
      this.startCleanupTask();
      
      logger.info('智能通知服务初始化完成');
    } catch (error) {
      ErrorHandler.handle(error, 'SmartNotificationService.onInitialize', ErrorType.SYSTEM, ErrorLevel.HIGH);
      throw new Error('SmartNotificationService初始化失败');
    }
  }

  /**
   * 实现基类的清理方法
   */
  protected async onCleanup(): Promise<void> {
    try {
      // 停止定时任务
      this.stopCleanupTask();
      this.stopPerformanceListener();
      
      // 保存数据
      await this.saveNotifications();
      await this.saveConfig();
      
      // 清理资源
      this.notifications.clear();
      this.listeners = [];
      
      logger.info('智能通知服务清理完成');
    } catch (error) {
      ErrorHandler.handle(error, 'SmartNotificationService.onCleanup', ErrorType.SYSTEM);
    }
  }

  /**
   * 重写基类的初始化方法
   */
  async initialize(): Promise<void> {
    await super.initialize();
    
    try {
      await this.loadConfig();
      await this.loadNotifications();
      this.setupPerformanceAlertListener();
      this.startAutoCleanup();
      
      logger.info('智能通知服务初始化完成');
    } catch (error) {
      ErrorHandler.handle(error, 'SmartNotificationService.initialize', ErrorType.SYSTEM, ErrorLevel.HIGH);
    }
  }

  /**
   * 设置性能告警监听器
   */
  private setupPerformanceAlertListener(): void {
    if (!this.config.enablePerformanceAlerts) return;

    // 使用轮询方式检查性能告警
    setInterval(() => {
      const alerts = this.performanceOptimizer.getPerformanceAlerts();
      if (alerts.length > 0) {
        const latestAlert = alerts[alerts.length - 1];
        this.createPerformanceNotification(latestAlert);
      }
    }, 10000); // 每10秒检查一次
  }

  /**
   * 创建性能告警通知
   */
  private async createPerformanceNotification(alert: PerformanceAlert): Promise<void> {
    const actions: NotificationAction[] = [];

    // 根据告警类型添加相应的操作
    switch (alert.type) {
      case 'memory':
        actions.push({
          id: 'clear_cache',
          label: '清理缓存',
          type: 'primary',
          handler: async () => {
            this.performanceOptimizer.clearCache();
            await this.markAsHandled(alert.type + '_' + alert.timestamp);
          }
        });
        break;
      case 'network':
        actions.push({
          id: 'retry_requests',
          label: '重试请求',
          type: 'primary',
          handler: async () => {
            // 实现重试逻辑
            await this.markAsHandled(alert.type + '_' + alert.timestamp);
          }
        });
        break;
      case 'battery':
        actions.push({
          id: 'enable_power_save',
          label: '启用省电模式',
          type: 'primary',
          handler: async () => {
            // 注意：setAdaptiveConfig方法不存在，这里只记录处理
            logger.info('用户选择启用省电模式');
            await this.markAsHandled(alert.type + '_' + alert.timestamp);
          }
        });
        break;
    }

    const notification: SmartNotification = {
      id: alert.type + '_' + alert.timestamp,
      type: 'performance' as 'performance',
      priority: (alert.level === 'critical' ? 'critical' : 'high') as 'critical' | 'high',
      title: '性能告警',
      message: alert.message,
      timestamp: alert.timestamp,
      read: false,
      actionable: actions.length > 0,
      actions,
      metadata: { alert: alert } as NotificationMetadata,
      expiresAt: Date.now() + 24 * 60 * 60 * 1000 // 24小时后过期
    } as SmartNotification;

    await this.addNotification(notification);
  }


  /**
   * 创建社交通知
   */
  public async createSocialNotification(
    type: 'like' | 'comment' | 'follow' | 'mention' | 'group_invite',
    fromUser: string,
    content: string,
    targetId?: string
  ): Promise<void> {
    if (!this.config.enableSocialNotifications) return;

    const titleMap: Record<string, string> = {
      'like': '有人点赞了您',
      'comment': '有人评论了您',
      'follow': '有人关注了您',
      'mention': '有人提到了您',
      'group_invite': '群组邀请'
    };

    const notification: SmartNotification = {
      id: `social_${type}_${Date.now()}`,
      type: 'social' as 'social',
      priority: (type === 'mention' ? 'high' : 'medium') as 'high' | 'medium',
      title: titleMap[type] || '社交通知',
      message: `${fromUser}: ${content}`,
      timestamp: Date.now(),
      read: false,
      actionable: true,
      actions: [{
        id: 'view_detail',
        label: '查看详情',
        type: 'primary' as 'primary',
        handler: async () => {
          // 跳转到相关页面
          logger.info(`查看社交通知详情: ${targetId}`);
          await this.markAsRead(`social_${type}_${Date.now()}`);
        }
      } as NotificationAction],
      metadata: { type: type, fromUser: fromUser, targetId: targetId } as NotificationMetadata,
      expiresAt: Date.now() + 7 * 24 * 60 * 60 * 1000 // 7天后过期
    } as SmartNotification;

    await this.addNotification(notification);
  }

  /**
   * 创建安全通知
   */
  public async createSecurityNotification(
    type: 'login_attempt' | 'password_change' | 'suspicious_activity',
    message: string,
    metadata?: Record<string, Object>
  ): Promise<void> {
    if (!this.config.enableSecurityAlerts) return;

    const titleMap: Record<string, string> = {
      'login_attempt': '登录尝试',
      'password_change': '密码变更',
      'suspicious_activity': '可疑活动'
    };

    const notification: SmartNotification = {
      id: `security_${type}_${Date.now()}`,
      type: 'security' as 'security',
      priority: 'critical' as 'critical',
      title: titleMap[type],
      message,
      timestamp: Date.now(),
      read: false,
      actionable: true,
      actions: [{
        id: 'review_security',
        label: '查看安全设置',
        type: 'danger' as 'danger',
        handler: async () => {
          // 跳转到安全设置页面
          logger.info('查看安全设置');
          await this.markAsRead(`security_${type}_${Date.now()}`);
        }
      }],
      metadata: metadata,
      expiresAt: Date.now() + 30 * 24 * 60 * 60 * 1000 // 30天后过期
    };

    await this.addNotification(notification);
  }

  /**
   * 添加通知
   */
  private async addNotification(notification: SmartNotification): Promise<void> {
    // 检查是否在静默时间
    if (this.isInQuietHours() && notification.priority !== 'critical') {
      logger.info('当前为静默时间，跳过非关键通知');
      return;
    }

    // 限制通知数量
    if (this.notifications.size >= this.config.maxNotifications) {
      await this.cleanupOldNotifications();
    }

    this.notifications.set(notification.id, notification);
    
    // 保存到本地存储
    await this.saveNotifications();
    
    // 无障碍支持
    if (this.accessibilityManager.getConfig().screenReaderEnabled) {
      await this.accessibilityManager.announceText(
        `新通知：${notification.title} - ${notification.message}`,
        notification.priority === 'critical' ? 'high' : 'medium'
      );
    }
    
    // 通知监听器
    this.notifyListeners(notification);
    
    logger.info(`添加通知: ${notification.title}`);
  }

  /**
   * 检查是否在静默时间
   */
  private isInQuietHours(): boolean {
    if (!this.config.quietHours.enabled) return false;

    const now = new Date();
    const currentTime = now.getHours() * 60 + now.getMinutes();
    
    const startTimeParts = this.config.quietHours.startTime.split(':').map(Number);
    const startHour = startTimeParts[0];
    const startMin = startTimeParts[1];
    
    const endTimeParts = this.config.quietHours.endTime.split(':').map(Number);
    const endHour = endTimeParts[0];
    const endMin = endTimeParts[1];
    
    const startTime = startHour * 60 + startMin;
    const endTime = endHour * 60 + endMin;
    
    if (startTime <= endTime) {
      return currentTime >= startTime && currentTime <= endTime;
    } else {
      // 跨天的情况
      return currentTime >= startTime || currentTime <= endTime;
    }
  }

  /**
   * 获取所有通知
   */
  public getNotifications(filter?: NotificationFilter): SmartNotification[] {
    let notifications = Array.from(this.notifications.values());
    
    if (filter) {
      if (filter.type) {
        notifications = notifications.filter(n => n.type === filter.type);
      }
      if (filter.priority) {
        notifications = notifications.filter(n => n.priority === filter.priority);
      }
      if (filter.read !== undefined) {
        notifications = notifications.filter(n => n.read === filter.read);
      }
    }
    
    // 按时间戳降序排列
    return notifications.sort((a, b) => b.timestamp - a.timestamp);
  }

  /**
   * 获取未读通知数量
   */
  public getUnreadCount(type?: 'system' | 'performance' | 'social' | 'security'): number {
    const notifications = this.getNotifications({ read: false, type });
    return notifications.length;
  }

  /**
   * 标记通知为已读
   */
  public async markAsRead(notificationId: string): Promise<void> {
    const notification = this.notifications.get(notificationId);
    if (notification) {
      notification.read = true;
      await this.saveNotifications();
      logger.info(`标记通知为已读: ${notificationId}`);
    }
  }

  /**
   * 标记所有通知为已读
   */
  public async markAllAsRead(type?: 'system' | 'performance' | 'social' | 'security'): Promise<void> {
    for (const notification of this.notifications.values()) {
      if (!type || notification.type === type) {
        notification.read = true;
      }
    }
    await this.saveNotifications();
    logger.info(`标记所有${type || ''}通知为已读`);
  }

  /**
   * 标记通知为已处理
   */
  public async markAsHandled(notificationId: string): Promise<void> {
    await this.markAsRead(notificationId);
    // 可以添加额外的处理逻辑
  }

  /**
   * 删除通知
   */
  public async deleteNotification(notificationId: string): Promise<void> {
    if (this.notifications.delete(notificationId)) {
      await this.saveNotifications();
      logger.info(`删除通知: ${notificationId}`);
    }
  }

  /**
   * 清理过期通知
   */
  private async cleanupExpiredNotifications(): Promise<void> {
    const now = Date.now();
    let cleanedCount = 0;
    
    const entries = Array.from(this.notifications.entries());
    for (let i = 0; i < entries.length; i++) {
      const entry = entries[i];
      const id = entry[0];
      const notification = entry[1];
      if (notification.expiresAt && notification.expiresAt < now) {
        this.notifications.delete(id);
        cleanedCount++;
      }
    }
    
    if (cleanedCount > 0) {
      await this.saveNotifications();
      logger.info(`清理过期通知: ${cleanedCount} 条`);
    }
  }

  /**
   * 清理旧通知
   */
  private async cleanupOldNotifications(): Promise<void> {
    const notifications = Array.from(this.notifications.entries())
      .sort((a, b) => a[1].timestamp - b[1].timestamp);
    
    // 保留最新的通知，删除超出限制的旧通知
    const toDelete = notifications.slice(0, notifications.length - this.config.maxNotifications + 10);
    
    for (let i = 0; i < toDelete.length; i++) {
      const entry = toDelete[i];
      const id = entry[0];
      this.notifications.delete(id);
    }
    
    if (toDelete.length > 0) {
      await this.saveNotifications();
      logger.info(`清理旧通知: ${toDelete.length} 条`);
    }
  }

  /**
   * 启动自动清理
   */
  private startAutoCleanup(): void {
    if (this.cleanupInterval !== -1) return;
    
    // 每小时清理一次
    this.cleanupInterval = setInterval(() => {
      this.cleanupExpiredNotifications();
    }, 60 * 60 * 1000);
  }

  /**
   * 停止自动清理
   */
  private stopAutoCleanup(): void {
    if (this.cleanupInterval !== -1) {
      clearInterval(this.cleanupInterval);
      this.cleanupInterval = -1;
    }
  }

  /**
   * 更新配置
   */
  public async updateConfig(updates: Partial<NotificationConfig>): Promise<void> {
    // 手动合并配置
    const newConfig: NotificationConfig = {
      enablePerformanceAlerts: updates.enablePerformanceAlerts !== undefined ? updates.enablePerformanceAlerts : this.config.enablePerformanceAlerts,
      enableRecommendations: updates.enableRecommendations !== undefined ? updates.enableRecommendations : this.config.enableRecommendations,
      enableSocialNotifications: updates.enableSocialNotifications !== undefined ? updates.enableSocialNotifications : this.config.enableSocialNotifications,
      enableSecurityAlerts: updates.enableSecurityAlerts !== undefined ? updates.enableSecurityAlerts : this.config.enableSecurityAlerts,
      quietHours: updates.quietHours || this.config.quietHours,
      maxNotifications: updates.maxNotifications !== undefined ? updates.maxNotifications : this.config.maxNotifications,
      autoCleanupDays: updates.autoCleanupDays !== undefined ? updates.autoCleanupDays : this.config.autoCleanupDays
    };
    this.config = newConfig;
    await this.saveConfig();
    
    // 重新设置性能告警监听器
    if (updates.enablePerformanceAlerts !== undefined) {
      if (updates.enablePerformanceAlerts) {
        this.setupPerformanceAlertListener();
      }
    }
    
    logger.info(`通知配置已更新`, updates);
  }

  /**
   * 获取配置
   */
  public getConfig(): NotificationConfig {
    const configCopy: NotificationConfig = {
      enablePerformanceAlerts: this.config.enablePerformanceAlerts,
      enableRecommendations: this.config.enableRecommendations,
      enableSocialNotifications: this.config.enableSocialNotifications,
      enableSecurityAlerts: this.config.enableSecurityAlerts,
      quietHours: this.config.quietHours,
      maxNotifications: this.config.maxNotifications,
      autoCleanupDays: this.config.autoCleanupDays
    };
    return configCopy;
  }

  /**
   * 添加通知监听器
   */
  public addListener(listener: (notification: SmartNotification) => void): void {
    this.listeners.push(listener);
  }

  /**
   * 移除通知监听器
   */
  public removeListener(listener: (notification: SmartNotification) => void): void {
    const index = this.listeners.indexOf(listener);
    if (index > -1) {
      this.listeners.splice(index, 1);
    }
  }

  /**
   * 通知监听器
   */
  private notifyListeners(notification: SmartNotification): void {
    this.listeners.forEach(listener => {
      try {
        listener(notification);
      } catch (error) {
        ErrorHandler.handle(error, 'notifyListeners', ErrorType.SYSTEM, ErrorLevel.LOW);
      }
    });
  }

  /**
   * 保存通知到本地存储
   */
  private async saveNotifications(): Promise<void> {
    return ErrorHandler.safeExecute(async () => {
      if (!this.storage) {
        throw new Error('存储未初始化');
      }
      const notificationsArray = Array.from(this.notifications.entries());
      await this.storage.setObject(this.storageKey, notificationsArray);
      logger.debug(`保存了 ${notificationsArray.length} 条通知`);
    }, 'saveNotifications', undefined, ErrorType.STORAGE);
  }

  /**
   * 从本地存储加载通知
   */
  private async loadNotifications(): Promise<void> {
    return ErrorHandler.safeExecute(async () => {
      if (!this.storage) {
        throw new Error('存储未初始化');
      }
      
      const notificationsArray = await this.storage.getObject<[string, SmartNotification][]>(this.storageKey);
      if (notificationsArray && Array.isArray(notificationsArray)) {
        this.notifications = new Map(notificationsArray);
        logger.debug(`加载了 ${notificationsArray.length} 条通知`);
        
        // 清理过期通知
        await this.cleanupExpiredNotifications();
      }
    }, 'loadNotifications', undefined, ErrorType.STORAGE);
  }

  /**
   * 保存配置到本地存储
   */
  private async saveConfig(): Promise<void> {
    return ErrorHandler.safeExecute(async () => {
      if (!this.storage) {
        throw new Error('存储未初始化');
      }
      await this.storage.setObject(this.configKey, this.config);
      logger.debug('配置已保存');
    }, 'saveConfig', undefined, ErrorType.STORAGE);
  }

  /**
   * 从本地存储加载配置
   */
  private async loadConfig(): Promise<void> {
    return ErrorHandler.safeExecute(async () => {
      if (!this.storage) {
        throw new Error('存储未初始化');
      }
      
      const savedConfig = await this.storage.getObject<NotificationConfig>(this.configKey);
      if (savedConfig) {
        // 合并配置，保留默认值
        this.config = {
          enablePerformanceAlerts: savedConfig.enablePerformanceAlerts ?? this.config.enablePerformanceAlerts,
          enableRecommendations: savedConfig.enableRecommendations ?? this.config.enableRecommendations,
          enableSocialNotifications: savedConfig.enableSocialNotifications ?? this.config.enableSocialNotifications,
          enableSecurityAlerts: savedConfig.enableSecurityAlerts ?? this.config.enableSecurityAlerts,
          quietHours: savedConfig.quietHours || this.config.quietHours,
          maxNotifications: savedConfig.maxNotifications ?? this.config.maxNotifications,
          autoCleanupDays: savedConfig.autoCleanupDays ?? this.config.autoCleanupDays
        };
        logger.debug('配置已加载');
      }
    }, 'loadConfig', undefined, ErrorType.STORAGE);
  }

  /**
   * 设置性能监听器
   */
  private setupPerformanceListener(): void {
    if (!this.config.enablePerformanceAlerts) return;

    // 使用TimerManager管理定时器
    this.performanceCheckTimerId = TimerManager.setInterval(() => {
      const alerts = this.performanceOptimizer.getPerformanceAlerts();
      if (alerts.length > 0) {
        const latestAlert = alerts[alerts.length - 1];
        this.createPerformanceNotification(latestAlert);
      }
    }, 5000, '性能告警检查');
    
    logger.debug('性能监听器已启动');
  }

  /**
   * 停止性能监听器
   */
  private stopPerformanceListener(): void {
    if (this.performanceCheckTimerId !== -1) {
      TimerManager.clearInterval(this.performanceCheckTimerId);
      this.performanceCheckTimerId = -1;
      logger.debug('性能监听器已停止');
    }
  }

  /**
   * 启动清理任务
   */
  private startCleanupTask(): void {
    if (this.cleanupTimerId !== -1) return;
    
    // 每小时清理一次过期通知
    this.cleanupTimerId = TimerManager.setInterval(() => {
      this.cleanupExpiredNotifications();
      this.cleanupOldNotifications();
    }, 60 * 60 * 1000, '通知清理任务');
    
    logger.debug('清理任务已启动');
  }

  /**
   * 停止清理任务
   */
  private stopCleanupTask(): void {
    if (this.cleanupTimerId !== -1) {
      TimerManager.clearInterval(this.cleanupTimerId);
      this.cleanupTimerId = -1;
      logger.debug('清理任务已停止');
    }
  }

  /**
   * 清理资源
   */
  public async cleanup(): Promise<void> {
    this.stopAutoCleanup();
    this.notifications.clear();
    this.listeners = [];
    logger.info('智能通知服务资源清理完成');
  }
}
