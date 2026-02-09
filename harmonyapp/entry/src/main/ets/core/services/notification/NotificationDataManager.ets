/**
 * 通知数据管理器
 * 专门负责通知的CRUD操作和数据持久化
 */

import { BaseSingleton } from '../../base/Singleton';
import { createLogger } from '../../utils/LogManager';
import { ErrorHandler, ErrorType, ErrorLevel } from '../../utils/ErrorHandler';
import { UnifiedStorageUtil } from '../../utils/UnifiedStorageUtil';
import { ConfigMerger } from '../../utils/ConfigMerger';
import { NOTIFICATION_CONSTANTS } from '../../constants/SystemConstants';
import { NotificationMetadata } from '../SmartNotificationService';

const logger = createLogger('NotificationDataManager');

/**
 * 通知接口
 */
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

export interface SmartNotificationInput {
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

export interface SmartNotificationUpdate {
  type?: 'system' | 'performance' | 'social' | 'security';
  priority?: 'low' | 'medium' | 'high' | 'critical';
  title?: string;
  message?: string;
  read?: boolean;
  actionable?: boolean;
  actions?: NotificationAction[];
  metadata?: NotificationMetadata;
  expiresAt?: number;
}

export interface NotificationFilter {
  type?: 'system' | 'performance' | 'social' | 'security';
  priority?: 'low' | 'medium' | 'high' | 'critical';
  read?: boolean;
  startDate?: number;
  endDate?: number;
}

/**
 * 通知统计信息
 */
export interface NotificationStats {
  total: number;
  unread: number;
  byType: Record<string, number>;
  byPriority: Record<string, number>;
  oldestUnread?: SmartNotification;
  newestNotification?: SmartNotification;
}

/**
 * 通知数据管理器
 */
export class NotificationDataManager extends BaseSingleton {
  private static instance: NotificationDataManager | null = null;
  
  /**
   * 内存中的通知存储
   */
  private notifications = new Map<string, SmartNotification>();
  
  /**
   * 持久化存储
   */
  private storage: UnifiedStorageUtil | null = null;
  
  /**
   * 存储键
   */
  private readonly storageKey = 'smart_notifications';
  
  /**
   * 数据变更监听器
   */
  private changeListeners: ((notifications: SmartNotification[]) => void)[] = [];
  
  private constructor() {
    super();
  }
  
  /**
   * 获取单例实例
   */
  static getInstance(): NotificationDataManager {
    if (!NotificationDataManager.instance) {
      NotificationDataManager.instance = new NotificationDataManager();
    }
    return NotificationDataManager.instance;
  }
  
  /**
   * 异步获取单例实例
   */
  static async getInstanceAsync(): Promise<NotificationDataManager> {
    const instance = NotificationDataManager.getInstance();
    await instance.initialize();
    return instance;
  }
  
  /**
   * 初始化
   */
  protected async onInitialize(): Promise<void> {
    logger.info('通知数据管理器初始化开始');
    
    try {
      // 初始化存储
      this.storage = UnifiedStorageUtil.getInstance();
      await this.storage.initialize();
      
      // 加载持久化的通知
      await this.loadNotificationsFromStorage();
      
      logger.info('通知数据管理器初始化完成');
    } catch (error) {
      ErrorHandler.handle(error, 'NotificationDataManager.onInitialize', ErrorType.SYSTEM, ErrorLevel.HIGH);
      throw new Error(String(error));
    }
  }
  
  /**
   * 清理
   */
  protected async onCleanup(): Promise<void> {
    logger.info('通知数据管理器开始清理');
    
    // 保存数据
    await this.saveNotificationsToStorage();
    
    // 清理内存
    this.notifications.clear();
    this.changeListeners = [];
    this.storage = null;
    
    NotificationDataManager.instance = null;
    logger.info('通知数据管理器清理完成');
  }
  
  /**
   * 添加通知
   */
  async addNotification(notification: SmartNotificationInput): Promise<string> {
    try {
      // 生成ID和时间戳
      const id = this.generateNotificationId();
      const timestamp = Date.now();
      
      const fullNotification: SmartNotification = {
        id: id,
        type: notification.type,
        priority: notification.priority,
        title: notification.title,
        message: notification.message,
        timestamp: timestamp,
        read: notification.read,
        actionable: notification.actionable,
        actions: notification.actions,
        metadata: notification.metadata,
        expiresAt: notification.expiresAt
      };
      
      // 验证通知
      this.validateNotification(fullNotification);
      
      // 检查数量限制
      await this.enforceNotificationLimit();
      
      // 添加到内存
      this.notifications.set(id, fullNotification);
      
      // 持久化
      await this.saveNotificationsToStorage();
      
      // 通知监听器
      this.notifyChangeListeners();
      
      logger.debug(`通知已添加: ${id}`);
      return id;
    } catch (error) {
      ErrorHandler.handle(error, 'NotificationDataManager.addNotification', ErrorType.BUSINESS, ErrorLevel.MEDIUM);
      throw new Error(String(error));
    }
  }
  
  /**
   * 获取通知
   */
  getNotification(id: string): SmartNotification | null {
    return this.notifications.get(id) || null;
  }
  
  /**
   * 更新通知
   */
  async updateNotification(id: string, updates: SmartNotificationUpdate): Promise<boolean> {
    try {
      const notification = this.notifications.get(id);
      if (!notification) {
        return false;
      }
      
      // 合并更新
      const updatedNotification = ConfigMerger.merge(notification, updates as Partial<SmartNotification>);
      
      // 验证更新后的通知
      this.validateNotification(updatedNotification);
      
      // 更新内存
      this.notifications.set(id, updatedNotification);
      
      // 持久化
      await this.saveNotificationsToStorage();
      
      // 通知监听器
      this.notifyChangeListeners();
      
      logger.debug(`通知已更新: ${id}`);
      return true;
    } catch (error) {
      ErrorHandler.handle(error, 'NotificationDataManager.updateNotification', ErrorType.BUSINESS, ErrorLevel.MEDIUM);
      return false;
    }
  }
  
  /**
   * 删除通知
   */
  async deleteNotification(id: string): Promise<boolean> {
    try {
      const deleted = this.notifications.delete(id);
      
      if (deleted) {
        // 持久化
        await this.saveNotificationsToStorage();
        
        // 通知监听器
        this.notifyChangeListeners();
        
        logger.debug(`通知已删除: ${id}`);
      }
      
      return deleted;
    } catch (error) {
      ErrorHandler.handle(error, 'NotificationDataManager.deleteNotification', ErrorType.BUSINESS, ErrorLevel.LOW);
      return false;
    }
  }
  
  /**
   * 批量删除通知
   */
  async deleteNotifications(ids: string[]): Promise<number> {
    try {
      let deletedCount = 0;
      
      for (const id of ids) {
        if (this.notifications.delete(id)) {
          deletedCount++;
        }
      }
      
      if (deletedCount > 0) {
        // 持久化
        await this.saveNotificationsToStorage();
        
        // 通知监听器
        this.notifyChangeListeners();
        
        logger.debug(`批量删除通知: ${deletedCount}个`);
      }
      
      return deletedCount;
    } catch (error) {
      ErrorHandler.handle(error, 'NotificationDataManager.deleteNotifications', ErrorType.BUSINESS, ErrorLevel.LOW);
      return 0;
    }
  }
  
  /**
   * 标记通知为已读
   */
  async markAsRead(id: string): Promise<boolean> {
    return this.updateNotification(id, { read: true });
  }
  
  /**
   * 批量标记为已读
   */
  async markAllAsRead(filter?: NotificationFilter): Promise<number> {
    const notifications = this.getNotifications(filter);
    const unreadIds = notifications
      .filter(n => !n.read)
      .map(n => n.id);
    
    let updatedCount = 0;
    for (const id of unreadIds) {
      if (await this.markAsRead(id)) {
        updatedCount++;
      }
    }
    
    return updatedCount;
  }
  
  /**
   * 获取通知列表
   */
  getNotifications(filter?: NotificationFilter): SmartNotification[] {
    let notifications = Array.from(this.notifications.values());
    
    if (filter) {
      notifications = notifications.filter(notification => {
        // 类型过滤
        if (filter.type && notification.type !== filter.type) {
          return false;
        }
        
        // 优先级过滤
        if (filter.priority && notification.priority !== filter.priority) {
          return false;
        }
        
        // 已读状态过滤
        if (filter.read !== undefined && notification.read !== filter.read) {
          return false;
        }
        
        // 时间范围过滤
        if (filter.startDate && notification.timestamp < filter.startDate) {
          return false;
        }
        
        if (filter.endDate && notification.timestamp > filter.endDate) {
          return false;
        }
        
        return true;
      });
    }
    
    // 按时间倒序排列
    return notifications.sort((a, b) => b.timestamp - a.timestamp);
  }
  
  /**
   * 获取未读通知
   */
  getUnreadNotifications(): SmartNotification[] {
    return this.getNotifications({ read: false });
  }
  
  /**
   * 获取通知统计
   */
  getStats(): NotificationStats {
    const notifications = Array.from(this.notifications.values());
    const unreadNotifications = notifications.filter(n => !n.read);
    
    // 按类型统计
    const byType: Record<string, number> = {};
    for (const notification of notifications) {
      byType[notification.type] = (byType[notification.type] || 0) + 1;
    }
    
    // 按优先级统计
    const byPriority: Record<string, number> = {};
    for (const notification of notifications) {
      byPriority[notification.priority] = (byPriority[notification.priority] || 0) + 1;
    }
    
    // 最旧的未读通知
    const oldestUnread = unreadNotifications
      .sort((a, b) => a.timestamp - b.timestamp)[0];
    
    // 最新的通知
    const newestNotification = notifications
      .sort((a, b) => b.timestamp - a.timestamp)[0];
    
    return {
      total: notifications.length,
      unread: unreadNotifications.length,
      byType,
      byPriority,
      oldestUnread,
      newestNotification
    };
  }
  
  /**
   * 清理过期通知
   */
  async cleanupExpiredNotifications(): Promise<number> {
    try {
      const now = Date.now();
      const expiredIds: string[] = [];
      
      // 遍历通知 Map
      this.notifications.forEach((notification, id) => {
        if (notification.expiresAt && notification.expiresAt < now) {
          expiredIds.push(id);
        }
      });
      
      const deletedCount = await this.deleteNotifications(expiredIds);
      
      if (deletedCount > 0) {
        logger.info(`清理过期通知: ${deletedCount}个`);
      }
      
      return deletedCount;
    } catch (error) {
      ErrorHandler.handle(error, 'NotificationDataManager.cleanupExpiredNotifications', ErrorType.SYSTEM, ErrorLevel.LOW);
      return 0;
    }
  }
  
  /**
   * 清理旧通知
   */
  async cleanupOldNotifications(maxAge: number = NOTIFICATION_CONSTANTS.CLEANUP.AUTO_CLEANUP_DAYS * 24 * 60 * 60 * 1000): Promise<number> {
    try {
      const cutoffTime = Date.now() - maxAge;
      const oldIds: string[] = [];
      
      // 遍历通知 Map
      this.notifications.forEach((notification, id) => {
        if (notification.timestamp < cutoffTime && notification.read) {
          oldIds.push(id);
        }
      });
      
      const deletedCount = await this.deleteNotifications(oldIds);
      
      if (deletedCount > 0) {
        logger.info(`清理旧通知: ${deletedCount}个`);
      }
      
      return deletedCount;
    } catch (error) {
      ErrorHandler.handle(error, 'NotificationDataManager.cleanupOldNotifications', ErrorType.SYSTEM, ErrorLevel.LOW);
      return 0;
    }
  }
  
  /**
   * 添加数据变更监听器
   */
  addChangeListener(listener: (notifications: SmartNotification[]) => void): void {
    this.changeListeners.push(listener);
  }
  
  /**
   * 移除数据变更监听器
   */
  removeChangeListener(listener: (notifications: SmartNotification[]) => void): void {
    const index = this.changeListeners.indexOf(listener);
    if (index > -1) {
      this.changeListeners.splice(index, 1);
    }
  }
  
  /**
   * 生成通知ID
   */
  private generateNotificationId(): string {
    return `notification_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
  }
  
  /**
   * 验证通知
   */
  private validateNotification(notification: SmartNotification): void {
    if (!notification.id) {
      throw new Error('通知ID不能为空');
    }
    
    if (!notification.title || notification.title.length > NOTIFICATION_CONSTANTS.LIMITS.MAX_TITLE_LENGTH) {
      throw new Error(`通知标题长度必须在1-${NOTIFICATION_CONSTANTS.LIMITS.MAX_TITLE_LENGTH}字符之间`);
    }
    
    if (!notification.message || notification.message.length > NOTIFICATION_CONSTANTS.LIMITS.MAX_MESSAGE_LENGTH) {
      throw new Error(`通知消息长度必须在1-${NOTIFICATION_CONSTANTS.LIMITS.MAX_MESSAGE_LENGTH}字符之间`);
    }
    
    if (notification.actions && notification.actions.length > NOTIFICATION_CONSTANTS.LIMITS.MAX_ACTIONS) {
      throw new Error(`通知操作数量不能超过${NOTIFICATION_CONSTANTS.LIMITS.MAX_ACTIONS}个`);
    }
  }
  
  /**
   * 强制执行通知数量限制
   */
  private async enforceNotificationLimit(): Promise<void> {
    const maxNotifications = NOTIFICATION_CONSTANTS.LIMITS.MAX_NOTIFICATIONS;
    
    if (this.notifications.size >= maxNotifications) {
      // 删除最旧的已读通知
      const notifications = Array.from(this.notifications.values())
        .filter((n: SmartNotification) => n.read)
        .sort((a: SmartNotification, b: SmartNotification) => a.timestamp - b.timestamp);
      
      const toDelete = notifications.slice(0, notifications.length - maxNotifications + 10);
      const idsToDelete = toDelete.map((n: SmartNotification) => n.id);
      
      await this.deleteNotifications(idsToDelete);
      
      logger.info(`删除旧通知以维持数量限制: ${idsToDelete.length}个`);
    }
  }
  
  /**
   * 从存储加载通知
   */
  private async loadNotificationsFromStorage(): Promise<void> {
    try {
      if (!this.storage) return;
      
      const data = await this.storage.getObject<SmartNotification[]>(this.storageKey);
      if (data && Array.isArray(data)) {
        this.notifications.clear();
        
        for (const notification of data) {
          this.notifications.set(notification.id, notification);
        }
        
        logger.debug(`从存储加载通知: ${data.length}个`);
      }
    } catch (error) {
      logger.error('加载通知数据失败', error);
      // 不抛出错误，使用空的通知列表
    }
  }
  
  /**
   * 保存通知到存储
   */
  private async saveNotificationsToStorage(): Promise<void> {
    try {
      if (!this.storage) return;
      
      const notifications = Array.from(this.notifications.values());
      await this.storage.setObject(this.storageKey, notifications);
      
      logger.debug(`保存通知到存储: ${notifications.length}个`);
    } catch (error) {
      logger.error('保存通知数据失败', error);
      // 不抛出错误，继续执行
    }
  }
  
  /**
   * 通知变更监听器
   */
  private notifyChangeListeners(): void {
    const notifications = this.getNotifications();
    
    for (const listener of this.changeListeners) {
      try {
        listener(notifications);
      } catch (error) {
        logger.error('通知变更监听器执行失败', error);
      }
    }
  }
}
