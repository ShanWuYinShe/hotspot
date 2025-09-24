/**
 * 推送通知服务
 * 集成HarmonyOS系统通知能力，提供统一的推送通知管理
 */

import { notificationManager } from '@kit.NotificationKit';
import { wantAgent, WantAgent } from '@kit.AbilityKit';
import { common } from '@kit.AbilityKit';
import { BusinessError } from '@kit.BasicServicesKit';
import { createLogger } from '../utils/LogManager';
import { ErrorHandler, ErrorType, ErrorLevel } from '../utils/ErrorHandler';
import { BaseSingleton } from '../base/Singleton';

const logger = createLogger('PushNotificationService');

/**
 * 通知类型枚举
 */
export enum NotificationType {
  MESSAGE = 'message',           // 消息通知
  GROUP = 'group',              // 群组通知
  MOMENT = 'moment',            // 瞬间通知
  SYSTEM = 'system',            // 系统通知
  FRIEND = 'friend',            // 好友通知
  LOCATION = 'location'         // 位置通知
}

/**
 * 通知优先级
 */
export enum NotificationPriority {
  LOW = 'low',
  NORMAL = 'normal',
  HIGH = 'high',
  URGENT = 'urgent'
}

/**
 * 通知配置
 */
export interface NotificationConfig {
  id: number;                    // 通知ID
  title: string;                 // 通知标题
  content: string;               // 通知内容
  type: NotificationType;        // 通知类型
  priority: NotificationPriority; // 优先级
  autoCancel?: boolean;          // 是否自动取消
  showDeliveryTime?: boolean;    // 是否显示发送时间
  tapAction?: NotificationTapAction; // 点击操作
  buttons?: NotificationButton[]; // 操作按钮
  largeIcon?: string;            // 大图标
  bigText?: string;              // 长文本
  bigPicture?: string;           // 大图片
  sound?: string;                // 提示音
  vibration?: number[];          // 震动模式
  ledColor?: string;             // LED颜色
}

/**
 * 通知点击操作
 */
export interface NotificationTapAction {
  bundleName: string;
  abilityName: string;
  parameters?: Record<string, Object>;
}

/**
 * 通知按钮
 */
export interface NotificationButton {
  title: string;
  wantAgent: WantAgent;
}

/**
 * 通知统计信息
 */
export interface NotificationStats {
  totalSent: number;             // 总发送数
  totalClicked: number;          // 总点击数
  totalDismissed: number;        // 总取消数
  byType: Record<NotificationType, number>; // 按类型统计
  byPriority: Record<NotificationPriority, number>; // 按优先级统计
}

/**
 * 推送通知服务
 */
export class PushNotificationService extends BaseSingleton {
  private static instance: PushNotificationService | null = null;
  private context: common.UIAbilityContext | null = null;
  private isNotificationEnabled: boolean = false;
  private notificationStats: NotificationStats;
  private notificationQueue: NotificationConfig[] = [];
  private isProcessingQueue: boolean = false;

  // 通知ID计数器
  private notificationIdCounter: number = 1000;

  // 通知类型配置
  private readonly typeConfig: Record<NotificationType, {
    slotType: notificationManager.SlotType;
    defaultPriority: NotificationPriority;
    icon: string;
  }> = {
    [NotificationType.MESSAGE]: {
      slotType: notificationManager.SlotType.SOCIAL_COMMUNICATION,
      defaultPriority: NotificationPriority.HIGH,
      icon: 'sys.symbol.message'
    },
    [NotificationType.GROUP]: {
      slotType: notificationManager.SlotType.SOCIAL_COMMUNICATION,
      defaultPriority: NotificationPriority.NORMAL,
      icon: 'sys.symbol.person_2'
    },
    [NotificationType.MOMENT]: {
      slotType: notificationManager.SlotType.CONTENT_INFORMATION,
      defaultPriority: NotificationPriority.NORMAL,
      icon: 'sys.symbol.camera'
    },
    [NotificationType.SYSTEM]: {
      slotType: notificationManager.SlotType.SERVICE_INFORMATION,
      defaultPriority: NotificationPriority.HIGH,
      icon: 'sys.symbol.gear'
    },
    [NotificationType.FRIEND]: {
      slotType: notificationManager.SlotType.SOCIAL_COMMUNICATION,
      defaultPriority: NotificationPriority.NORMAL,
      icon: 'sys.symbol.person'
    },
    [NotificationType.LOCATION]: {
      slotType: notificationManager.SlotType.OTHER_TYPES,
      defaultPriority: NotificationPriority.LOW,
      icon: 'sys.symbol.location'
    }
  };

  private constructor() {
    super();
    this.initializeStats();
  }

  public static getInstance(): PushNotificationService {
    if (!PushNotificationService.instance) {
      PushNotificationService.instance = new PushNotificationService();
    }
    return PushNotificationService.instance;
  }

  public static async getInstanceAsync(): Promise<PushNotificationService> {
    const instance = PushNotificationService.getInstance();
    await instance.initialize();
    return instance;
  }

  protected async onInitialize(): Promise<void> {
    logger.info('推送通知服务初始化');
    await this.initializeNotificationSlots();
  }

  protected async onCleanup(): Promise<void> {
    logger.info('推送通知服务清理');
    this.context = null;
    this.notificationQueue = [];
    PushNotificationService.instance = null;
  }

  /**
   * 设置上下文
   */
  public setContext(context: common.UIAbilityContext): void {
    this.context = context;
    logger.info('推送通知服务上下文已设置');
  }

  /**
   * 初始化统计信息
   */
  private initializeStats(): void {
    this.notificationStats = {
      totalSent: 0,
      totalClicked: 0,
      totalDismissed: 0,
      byType: {
        [NotificationType.MESSAGE]: 0,
        [NotificationType.GROUP]: 0,
        [NotificationType.MOMENT]: 0,
        [NotificationType.SYSTEM]: 0,
        [NotificationType.FRIEND]: 0,
        [NotificationType.LOCATION]: 0
      },
      byPriority: {
        [NotificationPriority.LOW]: 0,
        [NotificationPriority.NORMAL]: 0,
        [NotificationPriority.HIGH]: 0,
        [NotificationPriority.URGENT]: 0
      }
    };
  }

  /**
   * 初始化通知槽
   */
  private async initializeNotificationSlots(): Promise<void> {
    try {
      logger.info('初始化通知槽');

      // 为每种通知类型创建通知槽
      for (const [type, config] of Object.entries(this.typeConfig)) {
        const slot: notificationManager.NotificationSlot = {
          type: config.slotType,
          level: this.getPriorityLevel(config.defaultPriority),
          desc: `${type}通知`,
          badgeFlag: true,
          bypassDnd: config.defaultPriority === NotificationPriority.URGENT,
          lockscreenVisibility: notificationManager.VisibilityType.PUBLIC,
          vibrationEnabled: true,
          sound: 'file://system/etc/audio/alarms/default.ogg',
          lightEnabled: true,
          lightColor: 0xFF0000FF
        };

        await notificationManager.addSlot(slot);
        logger.debug(`通知槽已创建: ${type}`);
      }

      logger.info('所有通知槽初始化完成');

    } catch (error) {
      logger.error('初始化通知槽失败', error);
      ErrorHandler.handle(error, 'PushNotificationService.initializeNotificationSlots', ErrorType.SYSTEM, ErrorLevel.HIGH);
    }
  }

  /**
   * 检查通知权限
   */
  public async checkNotificationPermission(): Promise<boolean> {
    try {
      if (!this.context) {
        logger.warn('Context未设置，无法检查通知权限');
        return false;
      }

      const bundleOption: notificationManager.BundleOption = {
        bundle: this.context.applicationInfo.name
      };

      this.isNotificationEnabled = await notificationManager.isNotificationEnabled(bundleOption);
      logger.info(`通知权限状态: ${this.isNotificationEnabled}`);
      
      return this.isNotificationEnabled;

    } catch (error) {
      logger.error('检查通知权限失败', error);
      ErrorHandler.handle(error, 'PushNotificationService.checkNotificationPermission', ErrorType.SYSTEM, ErrorLevel.MEDIUM);
      return false;
    }
  }

  /**
   * 请求通知权限
   */
  public async requestNotificationPermission(): Promise<boolean> {
    try {
      if (!this.context) {
        throw new Error('Context未设置，请先调用setContext方法');
      }

      logger.info('请求通知权限');

      // 检查当前权限状态
      const hasPermission = await this.checkNotificationPermission();
      if (hasPermission) {
        logger.info('通知权限已授权');
        return true;
      }

      // 请求通知权限
      await notificationManager.requestEnableNotification(this.context);
      
      // 再次检查权限状态
      const newPermissionStatus = await this.checkNotificationPermission();
      logger.info(`通知权限请求结果: ${newPermissionStatus}`);
      
      return newPermissionStatus;

    } catch (error) {
      logger.error('请求通知权限失败', error);
      ErrorHandler.handle(error, 'PushNotificationService.requestNotificationPermission', ErrorType.SYSTEM, ErrorLevel.HIGH);
      return false;
    }
  }

  /**
   * 发送通知
   */
  public async sendNotification(config: NotificationConfig): Promise<boolean> {
    try {
      if (!this.isNotificationEnabled) {
        logger.warn('通知权限未授权，将通知加入队列');
        this.notificationQueue.push(config);
        return false;
      }

      logger.info(`发送通知: ${config.title}`);

      // 创建WantAgent（用于点击跳转）
      let wantAgentObj: WantAgent | undefined;
      if (config.tapAction) {
        wantAgentObj = await this.createWantAgent(config.tapAction);
      }

      // 构建通知内容
      const notificationContent: notificationManager.NotificationContent = {
        contentType: notificationManager.ContentType.NOTIFICATION_CONTENT_BASIC_TEXT,
        normal: {
          title: config.title,
          text: config.content,
          additionalText: config.bigText
        }
      };

      // 构建通知请求
      const notificationRequest: notificationManager.NotificationRequest = {
        id: config.id,
        content: notificationContent,
        slotType: this.typeConfig[config.type].slotType,
        isOngoing: false,
        isUnremovable: false,
        deliveryTime: config.showDeliveryTime ? new Date().getTime() : undefined,
        tapDismissed: config.autoCancel !== false,
        autoDeletedTime: config.autoCancel !== false ? new Date().getTime() + 10000 : undefined,
        wantAgent: wantAgentObj,
        extraInfo: {
          type: config.type,
          priority: config.priority
        }
      };

      // 添加大图标
      if (config.largeIcon) {
        notificationRequest.largeIcon = config.largeIcon;
      }

      // 添加操作按钮
      if (config.buttons && config.buttons.length > 0) {
        notificationRequest.actionButtons = config.buttons.map(button => ({
          title: button.title,
          wantAgent: button.wantAgent
        }));
      }

      // 发送通知
      await notificationManager.publish(notificationRequest);

      // 更新统计信息
      this.updateStats(config.type, config.priority, 'sent');

      logger.info(`通知发送成功: ${config.id}`);
      return true;

    } catch (error) {
      logger.error('发送通知失败', error);
      ErrorHandler.handle(error, 'PushNotificationService.sendNotification', ErrorType.SYSTEM, ErrorLevel.HIGH);
      return false;
    }
  }

  /**
   * 发送消息通知
   */
  public async sendMessageNotification(
    title: string,
    content: string,
    senderId: string,
    conversationId: string
  ): Promise<boolean> {
    const config: NotificationConfig = {
      id: this.generateNotificationId(),
      title,
      content,
      type: NotificationType.MESSAGE,
      priority: NotificationPriority.HIGH,
      autoCancel: true,
      showDeliveryTime: true,
      tapAction: {
        bundleName: this.context?.applicationInfo.name || '',
        abilityName: 'EntryAbility',
        parameters: {
          page: 'ChatPage',
          conversationId,
          senderId
        }
      }
    };

    return await this.sendNotification(config);
  }

  /**
   * 发送群组通知
   */
  public async sendGroupNotification(
    title: string,
    content: string,
    groupId: string,
    groupName: string
  ): Promise<boolean> {
    const config: NotificationConfig = {
      id: this.generateNotificationId(),
      title,
      content,
      type: NotificationType.GROUP,
      priority: NotificationPriority.NORMAL,
      autoCancel: true,
      tapAction: {
        bundleName: this.context?.applicationInfo.name || '',
        abilityName: 'EntryAbility',
        parameters: {
          page: 'GroupDetailPage',
          groupId,
          groupName
        }
      }
    };

    return await this.sendNotification(config);
  }

  /**
   * 发送系统通知
   */
  public async sendSystemNotification(
    title: string,
    content: string,
    priority: NotificationPriority = NotificationPriority.NORMAL
  ): Promise<boolean> {
    const config: NotificationConfig = {
      id: this.generateNotificationId(),
      title,
      content,
      type: NotificationType.SYSTEM,
      priority,
      autoCancel: true,
      showDeliveryTime: true
    };

    return await this.sendNotification(config);
  }

  /**
   * 取消通知
   */
  public async cancelNotification(notificationId: number): Promise<void> {
    try {
      await notificationManager.cancel(notificationId);
      logger.info(`通知已取消: ${notificationId}`);
    } catch (error) {
      logger.error(`取消通知失败: ${notificationId}`, error);
      ErrorHandler.handle(error, 'PushNotificationService.cancelNotification', ErrorType.SYSTEM, ErrorLevel.MEDIUM);
    }
  }

  /**
   * 取消所有通知
   */
  public async cancelAllNotifications(): Promise<void> {
    try {
      await notificationManager.cancelAll();
      logger.info('所有通知已取消');
    } catch (error) {
      logger.error('取消所有通知失败', error);
      ErrorHandler.handle(error, 'PushNotificationService.cancelAllNotifications', ErrorType.SYSTEM, ErrorLevel.MEDIUM);
    }
  }

  /**
   * 处理通知队列
   */
  public async processNotificationQueue(): Promise<void> {
    if (this.isProcessingQueue || this.notificationQueue.length === 0) {
      return;
    }

    this.isProcessingQueue = true;
    logger.info(`开始处理通知队列，待处理: ${this.notificationQueue.length}`);

    try {
      // 检查权限
      const hasPermission = await this.checkNotificationPermission();
      if (!hasPermission) {
        logger.warn('通知权限未授权，无法处理队列');
        return;
      }

      // 处理队列中的通知
      const queue = [...this.notificationQueue];
      this.notificationQueue = [];

      for (const config of queue) {
        try {
          await this.sendNotification(config);
          // 添加延迟，避免通知过于频繁
          await new Promise(resolve => setTimeout(resolve, 500));
        } catch (error) {
          logger.error('处理队列通知失败', error);
        }
      }

      logger.info('通知队列处理完成');

    } finally {
      this.isProcessingQueue = false;
    }
  }

  /**
   * 获取通知统计信息
   */
  public getNotificationStats(): NotificationStats {
    return { ...this.notificationStats };
  }

  /**
   * 清除通知统计信息
   */
  public clearNotificationStats(): void {
    this.initializeStats();
    logger.info('通知统计信息已清除');
  }

  /**
   * 创建WantAgent
   */
  private async createWantAgent(tapAction: NotificationTapAction): Promise<WantAgent> {
    const wantAgentInfo: wantAgent.WantAgentInfo = {
      wants: [{
        bundleName: tapAction.bundleName,
        abilityName: tapAction.abilityName,
        parameters: tapAction.parameters
      }],
      operationType: wantAgent.OperationType.START_ABILITY,
      requestCode: 0,
      wantAgentFlags: [wantAgent.WantAgentFlags.UPDATE_PRESENT_FLAG]
    };

    return await wantAgent.getWantAgent(wantAgentInfo);
  }

  /**
   * 生成通知ID
   */
  private generateNotificationId(): number {
    return this.notificationIdCounter++;
  }

  /**
   * 获取优先级对应的通知级别
   */
  private getPriorityLevel(priority: NotificationPriority): notificationManager.SlotLevel {
    switch (priority) {
      case NotificationPriority.LOW:
        return notificationManager.SlotLevel.LEVEL_MIN;
      case NotificationPriority.NORMAL:
        return notificationManager.SlotLevel.LEVEL_LOW;
      case NotificationPriority.HIGH:
        return notificationManager.SlotLevel.LEVEL_DEFAULT;
      case NotificationPriority.URGENT:
        return notificationManager.SlotLevel.LEVEL_HIGH;
      default:
        return notificationManager.SlotLevel.LEVEL_DEFAULT;
    }
  }

  /**
   * 更新统计信息
   */
  private updateStats(
    type: NotificationType,
    priority: NotificationPriority,
    action: 'sent' | 'clicked' | 'dismissed'
  ): void {
    switch (action) {
      case 'sent':
        this.notificationStats.totalSent++;
        break;
      case 'clicked':
        this.notificationStats.totalClicked++;
        break;
      case 'dismissed':
        this.notificationStats.totalDismissed++;
        break;
    }

    this.notificationStats.byType[type]++;
    this.notificationStats.byPriority[priority]++;
  }
}
