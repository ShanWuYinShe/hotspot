/**
 * 智能通知服务
 * 整合性能告警、推荐内容、系统通知等多种通知类型
 */
import { hilog } from '@kit.PerformanceAnalysisKit';
import { BusinessError } from '@kit.BasicServicesKit';
import { StorageUtil } from '../utils/StorageUtil';
import { PerformanceOptimizer, PerformanceAlert } from '../performance/PerformanceOptimizer';
import { AIRecommendationService, RecommendationItem } from './AIRecommendationService';
import { AccessibilityManager } from '../accessibility/AccessibilityManager';

export interface SmartNotification {
  id: string;
  type: 'system' | 'performance' | 'recommendation' | 'social' | 'security';
  priority: 'low' | 'medium' | 'high' | 'critical';
  title: string;
  message: string;
  timestamp: number;
  read: boolean;
  actionable: boolean;
  actions?: NotificationAction[];
  metadata?: Record<string, any>;
  expiresAt?: number;
}

export interface NotificationAction {
  id: string;
  label: string;
  type: 'primary' | 'secondary' | 'danger';
  handler: () => Promise<void>;
}

export interface NotificationConfig {
  enablePerformanceAlerts: boolean;
  enableRecommendations: boolean;
  enableSocialNotifications: boolean;
  enableSecurityAlerts: boolean;
  quietHours: {
    enabled: boolean;
    startTime: string; // HH:MM
    endTime: string;   // HH:MM
  };
  maxNotifications: number;
  autoCleanupDays: number;
}

export class SmartNotificationService {
  private static instance: SmartNotificationService;
  private notifications: Map<string, SmartNotification> = new Map();
  private config: NotificationConfig;
  private listeners: ((notification: SmartNotification) => void)[] = [];
  private performanceOptimizer: PerformanceOptimizer;
  private aiRecommendationService: AIRecommendationService;
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
      },
      maxNotifications: 100,
      autoCleanupDays: 7
    };

    this.performanceOptimizer = PerformanceOptimizer.getInstance();
    this.aiRecommendationService = AIRecommendationService.getInstance();
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

    this.performanceOptimizer.onPerformanceAlert((alert: PerformanceAlert) => {
      this.createPerformanceNotification(alert);
    });
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
            this.performanceOptimizer.setAdaptiveConfig({
              animationEnabled: false,
              backgroundSyncEnabled: false,
              imageQuality: 'low'
            });
            await this.markAsHandled(alert.type + '_' + alert.timestamp);
          }
        });
        break;
    }

    const notification = {
      id: alert.type + '_' + alert.timestamp,
      type: 'performance',
      priority: alert.level === 'critical' ? 'critical' : 'high',
      title: '性能告警',
      message: alert.message,
      timestamp: alert.timestamp,
      read: false,
      actionable: actions.length > 0,
      actions,
      metadata: { alert },
      expiresAt: Date.now() + 24 * 60 * 60 * 1000 // 24小时后过期
    };

    await this.addNotification(notification);
  }

  /**
   * 创建推荐内容通知
   */
  public async createRecommendationNotification(userId: string): Promise<void> {
    if (!this.config.enableRecommendations) return;

    try {
      const recommendations = await this.aiRecommendationService.getRecommendations(userId, 3);
      
      if (recommendations.length === 0) return;

      const topRecommendation = recommendations[0];
      
      const notification = {
        id: 'recommendation_' + Date.now(),
        type: 'recommendation',
        priority: 'medium',
        title: '为您推荐',
        message: `发现了您可能感兴趣的内容：${topRecommendation.title}`,
        timestamp: Date.now(),
        read: false,
        actionable: true,
        actions: [{
          id: 'view_recommendation',
          label: '查看推荐',
          type: 'primary',
          handler: async () => {
            // 跳转到推荐内容
            hilog.info(0x0000, 'SmartNotification', `查看推荐内容: ${topRecommendation.id}`);
            await this.markAsRead('recommendation_' + Date.now());
          }
        }],
        metadata: { recommendations },
        expiresAt: Date.now() + 48 * 60 * 60 * 1000 // 48小时后过期
      };

      await this.addNotification(notification);
    } catch (error) {
      hilog.error(0x0000, 'SmartNotification', `创建推荐通知失败: ${JSON.stringify(error)}`);
    }
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

    const titleMap = {
      'like': '有人点赞了您',
      'comment': '有人评论了您',
      'follow': '有人关注了您',
      'mention': '有人提到了您',
      'group_invite': '群组邀请'
    };

    const notification = {
      id: `social_${type}_${Date.now()}`,
      type: 'social',
      priority: type === 'mention' ? 'high' : 'medium',
      title: titleMap[type],
      message: `${fromUser}: ${content}`,
      timestamp: Date.now(),
      read: false,
      actionable: true,
      actions: [{
        id: 'view_detail',
        label: '查看详情',
        type: 'primary',
        handler: async () => {
          // 跳转到相关页面
          hilog.info(0x0000, 'SmartNotification', `查看社交通知详情: ${targetId}`);
          await this.markAsRead(`social_${type}_${Date.now()}`);
        }
      }],
      metadata: { type, fromUser, targetId },
      expiresAt: Date.now() + 7 * 24 * 60 * 60 * 1000 // 7天后过期
    };

    await this.addNotification(notification);
  }

  /**
   * 创建安全通知
   */
  public async createSecurityNotification(
    type: 'login_attempt' | 'password_change' | 'suspicious_activity',
    message: string,
    metadata?: Record<string, any>
  ): Promise<void> {
    if (!this.config.enableSecurityAlerts) return;

    const titleMap = {
      'login_attempt': '登录尝试',
      'password_change': '密码变更',
      'suspicious_activity': '可疑活动'
    };

    const notification = {
      id: `security_${type}_${Date.now()}`,
      type: 'security',
      priority: 'critical',
      title: titleMap[type],
      message,
      timestamp: Date.now(),
      read: false,
      actionable: true,
      actions: [{
        id: 'review_security',
        label: '查看安全设置',
        type: 'danger',
        handler: async () => {
          // 跳转到安全设置页面
          hilog.info(0x0000, 'SmartNotification', '查看安全设置');
          await this.markAsRead(`security_${type}_${Date.now()}`);
        }
      }],
      metadata,
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
    
    const [startHour, startMin] = this.config.quietHours.startTime.split(':').map(Number);
    const [endHour, endMin] = this.config.quietHours.endTime.split(':').map(Number);
    
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
  public getNotifications(filter?: {
    type?: SmartNotification['type'];
    priority?: SmartNotification['priority'];
    read?: boolean;
  }): SmartNotification[] {
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
  public getUnreadCount(type?: SmartNotification['type']): number {
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
  public async markAllAsRead(type?: SmartNotification['type']): Promise<void> {
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
    
    for (const [id, notification] of this.notifications.entries()) {
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
    
    for (const [id] of toDelete) {
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
    this.config = { ...this.config, ...updates };
    await this.saveConfig();
    
    // 重新设置性能告警监听器
    if ('enablePerformanceAlerts' in updates) {
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
    return { ...this.config };
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
      await StorageUtil.setItem(this.storageKey, JSON.stringify(notificationsArray));
    } catch (error) {
      hilog.error(0x0000, 'SmartNotification', `保存通知失败: ${JSON.stringify(error)}`);
    }
  }

  /**
   * 从本地存储加载通知
   */
  private async loadNotifications(): Promise<void> {
    try {
      const data = await StorageUtil.getItem(this.storageKey);
      if (data) {
        const notificationsArray = JSON.parse(data);
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
      await StorageUtil.setItem(this.configKey, JSON.stringify(this.config));
    } catch (error) {
      hilog.error(0x0000, 'SmartNotification', `保存配置失败: ${JSON.stringify(error)}`);
    }
  }

  /**
   * 从本地存储加载配置
   */
  private async loadConfig(): Promise<void> {
    try {
      const data = await StorageUtil.getItem(this.configKey);
      if (data) {
        const savedConfig = JSON.parse(data);
        this.config = { ...this.config, ...savedConfig };
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
