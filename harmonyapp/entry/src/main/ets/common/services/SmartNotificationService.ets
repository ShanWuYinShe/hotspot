/**
 * 智能通知服务
 * 整合性能告警、推荐内容、系统通知等多种通知类型
 */
import { hilog } from '@kit.PerformanceAnalysisKit';
import { BusinessError } from '@kit.BasicServicesKit';
import { StorageUtil } from '../utils/StorageUtil';
import { PerformanceOptimizer, PerformanceAlert } from '../performance/PerformanceOptimizer';
import { AccessibilityManager } from '../accessibility/AccessibilityManager';

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

export class SmartNotificationService {
  private static instance: SmartNotificationService;
  private notifications: Map<string, SmartNotification> = new Map();
  private config: NotificationConfig;
  private listeners: ((notification: SmartNotification) => void)[] = [];
  private performanceOptimizer: PerformanceOptimizer;
  private accessibilityManager: AccessibilityManager;
  private readonly storageKey = 'smart_notifications';
  private readonly configKey = 'notification_config';
  private cleanupInterval: number = -1;

  private constructor() {
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

    this.initialize();
  }

  public static getInstance(): SmartNotificationService {
    if (!SmartNotificationService.instance) {
      SmartNotificationService.instance = new SmartNotificationService();
    }
    return SmartNotificationService.instance;
  }

  /**
   * 初始化服务
   */
  private async initialize(): Promise<void> {
    try {
      await this.loadConfig();
      await this.loadNotifications();
      this.setupPerformanceAlertListener();
      this.startAutoCleanup();
      
      hilog.info(0x0000, 'SmartNotification', '智能通知服务初始化完成');
    } catch (error) {
      hilog.error(0x0000, 'SmartNotification', `初始化失败: ${JSON.stringify(error)}`);
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
            hilog.info(0x0000, 'SmartNotification', '用户选择启用省电模式');
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
          hilog.info(0x0000, 'SmartNotification', `查看社交通知详情: ${targetId}`);
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
          hilog.info(0x0000, 'SmartNotification', '查看安全设置');
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
      hilog.info(0x0000, 'SmartNotification', '当前为静默时间，跳过非关键通知');
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
    
    hilog.info(0x0000, 'SmartNotification', `添加通知: ${notification.title}`);
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
      hilog.info(0x0000, 'SmartNotification', `标记通知为已读: ${notificationId}`);
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
    hilog.info(0x0000, 'SmartNotification', `标记所有${type || ''}通知为已读`);
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
      hilog.info(0x0000, 'SmartNotification', `删除通知: ${notificationId}`);
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
      hilog.info(0x0000, 'SmartNotification', `清理过期通知: ${cleanedCount} 条`);
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
      hilog.info(0x0000, 'SmartNotification', `清理旧通知: ${toDelete.length} 条`);
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
    
    hilog.info(0x0000, 'SmartNotification', `通知配置已更新: ${JSON.stringify(updates)}`);
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
        hilog.error(0x0000, 'SmartNotification', `通知监听器执行失败: ${JSON.stringify(error)}`);
      }
    });
  }

  /**
   * 保存通知到本地存储
   */
  private async saveNotifications(): Promise<void> {
    try {
      const notificationsArray = Array.from(this.notifications.entries());
      await StorageUtil.setString(this.storageKey, JSON.stringify(notificationsArray));
    } catch (error) {
      hilog.error(0x0000, 'SmartNotification', `保存通知失败: ${JSON.stringify(error)}`);
    }
  }

  /**
   * 从本地存储加载通知
   */
  private async loadNotifications(): Promise<void> {
    try {
      const data = await StorageUtil.getString(this.storageKey, '');
      if (data) {
        const notificationsArray: [string, SmartNotification][] = JSON.parse(data) as [string, SmartNotification][];
        this.notifications = new Map(notificationsArray);
        
        // 清理过期通知
        await this.cleanupExpiredNotifications();
      }
    } catch (error) {
      hilog.warn(0x0000, 'SmartNotification', '加载通知失败，使用空列表');
    }
  }

  /**
   * 保存配置到本地存储
   */
  private async saveConfig(): Promise<void> {
    try {
      await StorageUtil.setString(this.configKey, JSON.stringify(this.config));
    } catch (error) {
      hilog.error(0x0000, 'SmartNotification', `保存配置失败: ${JSON.stringify(error)}`);
    }
  }

  /**
   * 从本地存储加载配置
   */
  private async loadConfig(): Promise<void> {
    try {
      const data = await StorageUtil.getString(this.configKey, '');
      if (data) {
        const savedConfig: Record<string, Object> = JSON.parse(data) as Record<string, Object>;
        // 手动合并配置
        const mergedConfig: NotificationConfig = {
          enablePerformanceAlerts: (savedConfig.enablePerformanceAlerts as boolean) !== undefined ? (savedConfig.enablePerformanceAlerts as boolean) : this.config.enablePerformanceAlerts,
          enableRecommendations: (savedConfig.enableRecommendations as boolean) !== undefined ? (savedConfig.enableRecommendations as boolean) : this.config.enableRecommendations,
          enableSocialNotifications: (savedConfig.enableSocialNotifications as boolean) !== undefined ? (savedConfig.enableSocialNotifications as boolean) : this.config.enableSocialNotifications,
          enableSecurityAlerts: (savedConfig.enableSecurityAlerts as boolean) !== undefined ? (savedConfig.enableSecurityAlerts as boolean) : this.config.enableSecurityAlerts,
          quietHours: (savedConfig.quietHours as QuietHours) || this.config.quietHours,
          maxNotifications: (savedConfig.maxNotifications as number) !== undefined ? (savedConfig.maxNotifications as number) : this.config.maxNotifications,
          autoCleanupDays: (savedConfig.autoCleanupDays as number) !== undefined ? (savedConfig.autoCleanupDays as number) : this.config.autoCleanupDays
        };
        this.config = mergedConfig;
      }
    } catch (error) {
      hilog.warn(0x0000, 'SmartNotification', '加载配置失败，使用默认配置');
    }
  }

  /**
   * 清理资源
   */
  public cleanup(): void {
    this.stopAutoCleanup();
    this.notifications.clear();
    this.listeners = [];
    hilog.info(0x0000, 'SmartNotification', '智能通知服务资源清理完成');
  }
}
