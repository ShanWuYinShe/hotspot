/**
 * 通知渲染器
 * 专门负责通知的UI显示、样式管理和用户交互
 */

import { BaseSingleton } from '../../base/Singleton';
import { createLogger } from '../../utils/LogManager';
import { ErrorHandler, ErrorType, ErrorLevel } from '../../utils/ErrorHandler';
import { ConfigMerger } from '../../utils/ConfigMerger';
import { UI_CONSTANTS } from '../../constants/SystemConstants';
import { SmartNotification, NotificationAction } from './NotificationDataManager';

const logger = createLogger('NotificationRenderer');

/**
 * 通知显示位置
 */
export enum NotificationPosition {
  TOP = 'top',
  BOTTOM = 'bottom',
  CENTER = 'center'
}

/**
 * 通知动画类型
 */
export enum NotificationAnimation {
  SLIDE = 'slide',
  FADE = 'fade',
  BOUNCE = 'bounce',
  NONE = 'none'
}

/**
 * 渲染配置
 */
export interface RendererConfig {
  position: NotificationPosition;
  animation: NotificationAnimation;
  duration: number;              // 显示持续时间（毫秒）
  maxVisible: number;            // 最大同时显示数量
  enableSound: boolean;          // 启用声音
  enableVibration: boolean;      // 启用震动
  enableAutoHide: boolean;       // 启用自动隐藏
  stackNotifications: boolean;   // 堆叠显示
  showProgress: boolean;         // 显示进度条
  theme: 'light' | 'dark' | 'auto'; // 主题
}

/**
 * 渲染配置更新类
 */
export class RendererConfigUpdate {
  position?: NotificationPosition;
  animation?: NotificationAnimation;
  duration?: number;
  maxVisible?: number;
  enableSound?: boolean;
  enableVibration?: boolean;
  enableAutoHide?: boolean;
  stackNotifications?: boolean;
  showProgress?: boolean;
  theme?: string;
}

/**
 * 通知样式
 */
interface NotificationStyle {
  backgroundColor: string;
  textColor: string;
  borderColor: string;
  iconColor: string;
  shadowColor: string;
  borderRadius: number;
  padding: number;
  margin: number;
}

/**
 * 通知元素接口
 */
interface NotificationElement {
  id: string;
  type: string;
  style: NotificationStyle;
  content: NotificationElementContent;
  show: () => Promise<void>;
  hide: () => Promise<void>;
  update: (content: Object) => Promise<void>;
  remove: () => Promise<void>;
  onClick?: () => void;
  onAction?: (actionId: string) => void;
  onClose?: () => void;
}

/**
 * 通知元素内容接口
 */
interface NotificationElementContent {
  title: string;
  message: string;
  actions: NotificationAction[];
  timestamp: number;
}

/**
 * 显示中的通知信息
 */
interface DisplayedNotification {
  id: string;
  notification: SmartNotification;
  element: NotificationElement;
  startTime: number;
  hideTimer?: number;
}

/**
 * 渲染统计
 */
export interface RendererStats {
  totalRendered: number;
  currentlyVisible: number;
  averageDisplayTime: number;
  userInteractions: number;
  autoHiddenCount: number;
  manuallyClosedCount: number;
}

/**
 * 通知渲染器
 */
export class NotificationRenderer extends BaseSingleton {
  private static instance: NotificationRenderer | null = null;
  
  /**
   * 渲染配置
   */
  private config: RendererConfig;
  
  /**
   * 当前显示的通知
   */
  private displayedNotifications = new Map<string, DisplayedNotification>();
  
  /**
   * 通知队列（等待显示）
   */
  private notificationQueue: SmartNotification[] = [];
  
  /**
   * 样式缓存
   */
  private styleCache = new Map<string, NotificationStyle>();
  
  /**
   * 事件监听器
   */
  private clickListeners = new Map<string, (notification: SmartNotification) => void>();
  private actionListeners = new Map<string, (action: NotificationAction, notification: SmartNotification) => void>();
  
  /**
   * 统计信息
   */
  private stats: RendererStats = {
    totalRendered: 0,
    currentlyVisible: 0,
    averageDisplayTime: 0,
    userInteractions: 0,
    autoHiddenCount: 0,
    manuallyClosedCount: 0
  };
  
  private constructor() {
    super();
    
    this.config = {
      position: NotificationPosition.TOP,
      animation: NotificationAnimation.SLIDE,
      duration: 5000,              // 5秒
      maxVisible: 3,               // 最多同时显示3个
      enableSound: true,
      enableVibration: false,
      enableAutoHide: true,
      stackNotifications: true,
      showProgress: true,
      theme: 'auto'
    };
  }
  
  /**
   * 获取单例实例
   */
  static getInstance(): NotificationRenderer {
    if (!NotificationRenderer.instance) {
      NotificationRenderer.instance = new NotificationRenderer();
    }
    return NotificationRenderer.instance;
  }
  
  /**
   * 异步获取单例实例
   */
  static async getInstanceAsync(): Promise<NotificationRenderer> {
    const instance = NotificationRenderer.getInstance();
    await instance.initialize();
    return instance;
  }
  
  /**
   * 初始化
   */
  protected async onInitialize(): Promise<void> {
    logger.info('通知渲染器初始化开始');
    
    try {
      // 初始化样式缓存
      this.initializeStyleCache();
      
      // 设置主题监听
      this.setupThemeListener();
      
      logger.info('通知渲染器初始化完成');
    } catch (error) {
      ErrorHandler.handle(error, 'NotificationRenderer.onInitialize', ErrorType.SYSTEM, ErrorLevel.HIGH);
      throw (error instanceof Error) ? error : new Error(String(error));
    }
  }
  
  /**
   * 清理
   */
  protected async onCleanup(): Promise<void> {
    logger.info('通知渲染器开始清理');
    
    // 隐藏所有显示中的通知
    await this.hideAllNotifications();
    
    // 清理数据
    this.displayedNotifications.clear();
    this.notificationQueue = [];
    this.styleCache.clear();
    this.clickListeners.clear();
    this.actionListeners.clear();
    
    NotificationRenderer.instance = null;
    logger.info('通知渲染器清理完成');
  }
  
  /**
   * 更新配置
   */
  updateConfig(config: RendererConfigUpdate): void {
    this.config = ConfigMerger.merge(this.config, config as Object as Record<string, Object>);
    
    // 重新初始化样式缓存
    this.initializeStyleCache();
    
    logger.debug('渲染器配置已更新');
  }
  
  /**
   * 获取配置
   */
  getConfig(): RendererConfig {
    const result: RendererConfig = {
      position: this.config.position,
      animation: this.config.animation,
      duration: this.config.duration,
      maxVisible: this.config.maxVisible,
      enableSound: this.config.enableSound,
      enableVibration: this.config.enableVibration,
      enableAutoHide: this.config.enableAutoHide,
      stackNotifications: this.config.stackNotifications,
      showProgress: this.config.showProgress,
      theme: this.config.theme
    };
    return result;
  }
  
  /**
   * 显示通知
   */
  async showNotification(notification: SmartNotification): Promise<void> {
    // 检查是否已经显示
    if (this.displayedNotifications.has(notification.id)) {
      logger.debug(`通知已在显示中: ${notification.id}`);
      return;
    }
    
    // 检查显示数量限制
    if (this.displayedNotifications.size >= this.config.maxVisible) {
      if (this.config.stackNotifications) {
        // 添加到队列
        this.notificationQueue.push(notification);
        logger.debug(`通知已加入队列: ${notification.id}`);
        return;
      } else {
        // 隐藏最旧的通知
        await this.hideOldestNotification();
      }
    }
    
    try {
      // 创建UI元素
      const element = await this.createNotificationElement(notification);
      
      // 记录显示信息
      const displayInfo: DisplayedNotification = {
        id: notification.id,
        notification,
        element,
        startTime: Date.now()
      };
      
      this.displayedNotifications.set(notification.id, displayInfo);
      
      // 播放声音和震动
      await this.playNotificationEffects(notification);
      
      // 显示动画
      await this.animateIn(element);
      
      // 设置自动隐藏
      if (this.config.enableAutoHide && notification.priority !== 'critical') {
        displayInfo.hideTimer = setTimeout(() => {
          this.hideNotification(notification.id, 'auto');
        }, this.config.duration);
      }
      
      // 更新统计
      this.stats.totalRendered++;
      this.stats.currentlyVisible = this.displayedNotifications.size;
      
      logger.debug(`通知显示成功: ${notification.id}`);
    } catch (error) {
      ErrorHandler.handle(error, 'NotificationRenderer.showNotification', ErrorType.BUSINESS, ErrorLevel.MEDIUM);
      logger.error(`显示通知失败: ${notification.id}`, error);
      throw (error instanceof Error) ? error : new Error(String(error));
    }
  }
  
  /**
   * 隐藏通知
   */
  async hideNotification(notificationId: string, reason: 'auto' | 'manual' | 'action' = 'manual'): Promise<boolean> {
    const displayInfo = this.displayedNotifications.get(notificationId);
    if (!displayInfo) {
      return false;
    }
    
    try {
      // 清除自动隐藏定时器
      if (displayInfo.hideTimer) {
        clearTimeout(displayInfo.hideTimer);
      }
      
      // 隐藏动画
      await this.animateOut(displayInfo.element);
      
      // 移除UI元素
      await this.removeNotificationElement(displayInfo.element);
      
      // 从显示列表中移除
      this.displayedNotifications.delete(notificationId);
      
      // 更新统计
      this.updateHideStats(displayInfo, reason);
      this.stats.currentlyVisible = this.displayedNotifications.size;
      
      // 显示队列中的下一个通知
      await this.showNextQueuedNotification();
      
      logger.debug(`通知已隐藏: ${notificationId}, 原因: ${reason}`);
      return true;
    } catch (error) {
      ErrorHandler.handle(error, 'NotificationRenderer.hideNotification', ErrorType.BUSINESS, ErrorLevel.LOW);
      logger.error(`隐藏通知失败: ${notificationId}`, error);
      return false;
    }
  }
  
  /**
   * 隐藏所有通知
   */
  async hideAllNotifications(): Promise<void> {
    const notificationIds = Array.from(this.displayedNotifications.keys());
    
    const hidePromises = notificationIds.map(id => 
      this.hideNotification(id, 'manual')
    );
    
    await Promise.all(hidePromises);
    
    // 清空队列
    this.notificationQueue = [];
    
    logger.info(`隐藏所有通知: ${notificationIds.length}个`);
  }
  
  /**
   * 更新通知内容
   */
  async updateNotification(notificationId: string, updates: Partial<SmartNotification>): Promise<boolean> {
    const displayInfo = this.displayedNotifications.get(notificationId);
    if (!displayInfo) {
      return false;
    }
    
    try {
      // 更新通知数据
      const updatedNotification = ConfigMerger.merge(displayInfo.notification, updates);
      displayInfo.notification = updatedNotification;
      
      // 更新UI元素
      await this.updateNotificationElement(displayInfo.element, updatedNotification);
      
      logger.debug(`通知已更新: ${notificationId}`);
      return true;
    } catch (error) {
      ErrorHandler.handle(error, 'NotificationRenderer.updateNotification', ErrorType.BUSINESS, ErrorLevel.MEDIUM);
      logger.error(`更新通知失败: ${notificationId}`, error);
      return false;
    }
  }
  
  /**
   * 设置点击监听器
   */
  setClickListener(notificationId: string, listener: (notification: SmartNotification) => void): void {
    this.clickListeners.set(notificationId, listener);
  }
  
  /**
   * 设置操作监听器
   */
  setActionListener(notificationId: string, listener: (action: NotificationAction, notification: SmartNotification) => void): void {
    this.actionListeners.set(notificationId, listener);
  }
  
  /**
   * 获取渲染统计
   */
  getStats(): RendererStats {
    const result: RendererStats = {
      totalRendered: this.stats.totalRendered,
      currentlyVisible: this.displayedNotifications.size,
      averageDisplayTime: this.stats.averageDisplayTime,
      userInteractions: this.stats.userInteractions,
      autoHiddenCount: this.stats.autoHiddenCount,
      manuallyClosedCount: this.stats.manuallyClosedCount
    };
    return result;
  }
  
  /**
   * 清理统计数据
   */
  clearStats(): void {
    this.stats = {
      totalRendered: 0,
      currentlyVisible: this.displayedNotifications.size,
      averageDisplayTime: 0,
      userInteractions: 0,
      autoHiddenCount: 0,
      manuallyClosedCount: 0
    };
    
    logger.debug('渲染器统计数据已清理');
  }
  
  /**
   * 初始化样式缓存
   */
  private initializeStyleCache(): void {
    this.styleCache.clear();
    
    // 根据通知类型和优先级生成样式
    const types = ['system', 'performance', 'social', 'security'];
    const priorities = ['low', 'medium', 'high', 'critical'];
    
    for (const type of types) {
      for (const priority of priorities) {
        const key = `${type}_${priority}`;
        const style = this.generateNotificationStyle(type, priority);
        this.styleCache.set(key, style);
      }
    }
    
    logger.debug('样式缓存已初始化');
  }
  
  /**
   * 生成通知样式
   */
  private generateNotificationStyle(type: string, priority: string): NotificationStyle {
    const isDark = this.config.theme === 'dark' || 
      (this.config.theme === 'auto' && this.isSystemDarkMode());
    
    // 基础样式
    let backgroundColor = isDark ? '#2C2C2C' : '#FFFFFF';
    let textColor = isDark ? '#FFFFFF' : '#000000';
    let borderColor = '#E0E0E0';
    let iconColor = '#666666';
    
    // 根据类型调整颜色
    switch (type) {
      case 'system':
        borderColor = '#2196F3';
        iconColor = '#2196F3';
        break;
      case 'performance':
        borderColor = '#FF9800';
        iconColor = '#FF9800';
        break;
      case 'social':
        borderColor = '#4CAF50';
        iconColor = '#4CAF50';
        break;
      case 'security':
        borderColor = '#F44336';
        iconColor = '#F44336';
        break;
    }
    
    // 根据优先级调整样式
    switch (priority) {
      case 'critical':
        backgroundColor = isDark ? '#4A1A1A' : '#FFEBEE';
        borderColor = '#F44336';
        break;
      case 'high':
        backgroundColor = isDark ? '#4A3A1A' : '#FFF3E0';
        borderColor = '#FF9800';
        break;
    }
    
    return {
      backgroundColor,
      textColor,
      borderColor,
      iconColor,
      shadowColor: isDark ? 'rgba(0,0,0,0.5)' : 'rgba(0,0,0,0.1)',
      borderRadius: UI_CONSTANTS.SPACING.SM,
      padding: UI_CONSTANTS.SPACING.MD,
      margin: UI_CONSTANTS.SPACING.SM
    };
  }
  
  /**
   * 创建通知UI元素
   */
  private async createNotificationElement(notification: SmartNotification): Promise<NotificationElement> {
    // 这里应该创建实际的HarmonyOS UI组件
    // 由于这是架构设计，返回一个模拟的元素对象
    
    const styleKey = `${notification.type}_${notification.priority}`;
    const style = this.styleCache.get(styleKey) || this.generateNotificationStyle(notification.type, notification.priority);
    
    const content: NotificationElementContent = {
      title: notification.title,
      message: notification.message,
      actions: notification.actions || [],
      timestamp: notification.timestamp
    };
    
    const element: NotificationElement = {
      id: notification.id,
      type: 'notification',
      style: style,
      content: content,
      // 模拟的UI方法
      show: () => Promise.resolve(),
      hide: () => Promise.resolve(),
      update: (content: Object) => Promise.resolve(),
      remove: () => Promise.resolve()
    };
    
    // 设置事件处理
    this.setupElementEvents(element, notification);
    
    return element;
  }
  
  /**
   * 设置元素事件
   */
  private setupElementEvents(element: NotificationElement, notification: SmartNotification): void {
    // 点击事件
    element.onClick = () => {
      this.stats.userInteractions++;
      
      const listener = this.clickListeners.get(notification.id);
      if (listener) {
        listener(notification);
      }
    };
    
    // 操作按钮事件
    element.onAction = (actionId: string) => {
      this.stats.userInteractions++;
      
      const action = notification.actions?.find(a => a.id === actionId);
      if (action) {
        const listener = this.actionListeners.get(notification.id);
        if (listener) {
          listener(action, notification);
        }
        
        // 执行操作处理器
        if (action.handler) {
          action.handler().catch((error: Object) => {
            logger.error(`通知操作执行失败: ${actionId}`, error);
          });
        }
      }
    };
    
    // 关闭事件
    element.onClose = () => {
      this.hideNotification(notification.id, 'manual');
    };
  }
  
  /**
   * 播放通知效果
   */
  private async playNotificationEffects(notification: SmartNotification): Promise<void> {
    try {
      // 播放声音
      if (this.config.enableSound) {
        await this.playNotificationSound(notification);
      }
      
      // 震动
      if (this.config.enableVibration) {
        await this.playNotificationVibration(notification);
      }
    } catch (error) {
      logger.warn('播放通知效果失败', error);
      // 不抛出错误，继续显示通知
    }
  }
  
  /**
   * 播放通知声音
   */
  private async playNotificationSound(notification: SmartNotification): Promise<void> {
    // 这里应该调用HarmonyOS的音频API
    // 根据通知类型和优先级选择不同的声音
    logger.debug(`播放通知声音: ${notification.type}_${notification.priority}`);
  }
  
  /**
   * 播放通知震动
   */
  private async playNotificationVibration(notification: SmartNotification): Promise<void> {
    // 这里应该调用HarmonyOS的震动API
    // 根据通知优先级选择不同的震动模式
    logger.debug(`播放通知震动: ${notification.priority}`);
  }
  
  /**
   * 入场动画
   */
  private async animateIn(element: NotificationElement): Promise<void> {
    switch (this.config.animation) {
      case NotificationAnimation.SLIDE:
        await this.slideInAnimation(element);
        break;
      case NotificationAnimation.FADE:
        await this.fadeInAnimation(element);
        break;
      case NotificationAnimation.BOUNCE:
        await this.bounceInAnimation(element);
        break;
      case NotificationAnimation.NONE:
        await element.show();
        break;
    }
  }
  
  /**
   * 出场动画
   */
  private async animateOut(element: any): Promise<void> {
    switch (this.config.animation) {
      case NotificationAnimation.SLIDE:
        await this.slideOutAnimation(element);
        break;
      case NotificationAnimation.FADE:
        await this.fadeOutAnimation(element);
        break;
      case NotificationAnimation.BOUNCE:
        await this.bounceOutAnimation(element);
        break;
      case NotificationAnimation.NONE:
        await element.hide();
        break;
    }
  }
  
  /**
   * 滑入动画
   */
  private async slideInAnimation(element: any): Promise<void> {
    // 实现滑入动画
    logger.debug('执行滑入动画');
    await element.show();
  }
  
  /**
   * 滑出动画
   */
  private async slideOutAnimation(element: any): Promise<void> {
    // 实现滑出动画
    logger.debug('执行滑出动画');
    await element.hide();
  }
  
  /**
   * 淡入动画
   */
  private async fadeInAnimation(element: any): Promise<void> {
    // 实现淡入动画
    logger.debug('执行淡入动画');
    await element.show();
  }
  
  /**
   * 淡出动画
   */
  private async fadeOutAnimation(element: NotificationElement): Promise<void> {
    // 实现淡出动画
    logger.debug('执行淡出动画');
    await element.hide();
  }
  
  /**
   * 弹跳入场动画
   */
  private async bounceInAnimation(element: NotificationElement): Promise<void> {
    // 实现弹跳入场动画
    logger.debug('执行弹跳入场动画');
    await element.show();
  }
  
  /**
   * 弹跳出场动画
   */
  private async bounceOutAnimation(element: NotificationElement): Promise<void> {
    // 实现弹跳出场动画
    logger.debug('执行弹跳出场动画');
    await element.hide();
  }
  
  /**
   * 更新通知元素
   */
  private async updateNotificationElement(element: NotificationElement, notification: SmartNotification): Promise<void> {
    const content: NotificationElementContent = {
      title: notification.title,
      message: notification.message,
      actions: notification.actions || [],
      timestamp: notification.timestamp
    };
    
    await element.update(content);
  }
  
  /**
   * 移除通知元素
   */
  private async removeNotificationElement(element: NotificationElement): Promise<void> {
    await element.remove();
  }
  
  /**
   * 隐藏最旧的通知
   */
  private async hideOldestNotification(): Promise<void> {
    let oldestId: string | null = null;
    let oldestTime = Infinity;
    
    for (const entry of this.displayedNotifications.entries()) {
      const id = entry[0];
      const displayInfo = entry[1];
      if (displayInfo.startTime < oldestTime) {
        oldestTime = displayInfo.startTime;
        oldestId = id;
      }
    }
    
    if (oldestId) {
      await this.hideNotification(oldestId, 'auto');
    }
  }
  
  /**
   * 显示队列中的下一个通知
   */
  private async showNextQueuedNotification(): Promise<void> {
    if (this.notificationQueue.length > 0 && 
        this.displayedNotifications.size < this.config.maxVisible) {
      const nextNotification = this.notificationQueue.shift()!;
      await this.showNotification(nextNotification);
    }
  }
  
  /**
   * 更新隐藏统计
   */
  private updateHideStats(displayInfo: DisplayedNotification, reason: string): void {
    const displayTime = Date.now() - displayInfo.startTime;
    
    // 更新平均显示时间
    const totalDisplayTime = this.stats.averageDisplayTime * this.stats.totalRendered + displayTime;
    this.stats.averageDisplayTime = totalDisplayTime / (this.stats.totalRendered + 1);
    
    // 更新隐藏原因统计
    switch (reason) {
      case 'auto':
        this.stats.autoHiddenCount++;
        break;
      case 'manual':
        this.stats.manuallyClosedCount++;
        break;
    }
  }
  
  /**
   * 设置主题监听
   */
  private setupThemeListener(): void {
    // 这里应该监听系统主题变化
    // 当主题改变时重新初始化样式缓存
    logger.debug('主题监听器已设置');
  }
  
  /**
   * 检查是否为系统暗色模式
   */
  private isSystemDarkMode(): boolean {
    // 这里应该检查系统主题设置
    // 暂时返回false
    return false;
  }
  
  /**
   * 处理通知点击
   */
  handleClick(notification: SmartNotification): void {
    try {
      this.stats.userInteractions++;
      const listener = this.clickListeners.get(notification.id);
      if (listener) {
        listener(notification);
      }
      logger.debug(`通知被点击: ${notification.id}`);
    } catch (error) {
      ErrorHandler.handle(error, 'NotificationRenderer.handleClick', ErrorType.BUSINESS, ErrorLevel.MEDIUM);
      logger.error(`处理通知点击失败: ${notification.id}`, error);
    }
  }
  
  /**
   * 处理通知操作
   */
  handleAction(action: NotificationAction, notification: SmartNotification): void {
    try {
      this.stats.userInteractions++;
      const listener = this.actionListeners.get(notification.id);
      if (listener) {
        listener(action, notification);
      }
      
      // 执行操作处理器
      if (action.handler) {
        action.handler().catch((error: Object) => {
          logger.error(`通知操作执行失败: ${action.id}`, error);
        });
      }
      
      logger.debug(`通知操作执行: ${action.id} for notification ${notification.id}`);
    } catch (error) {
      ErrorHandler.handle(error, 'NotificationRenderer.handleAction', ErrorType.BUSINESS, ErrorLevel.MEDIUM);
      logger.error(`处理通知操作失败: ${action.id}`, error);
    }
  }
}
