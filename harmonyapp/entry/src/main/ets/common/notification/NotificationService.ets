/**
 * HarmonyOS通知服务管理器
 * 集成HarmonyOS原生通知能力，支持本地通知和推送通知
 */

import notificationManager from '@ohos.notificationManager';
import wantAgent, { WantAgent } from '@ohos.app.ability.wantAgent';
import { common } from '@kit.AbilityKit';
import { hilog } from '@kit.PerformanceAnalysisKit';

const DOMAIN_NUMBER: number = 0xFF08;
const TAG: string = 'NotificationService';

export interface NotificationData {
  id: number;
  title: string;
  content: string;
  additionalText?: string;
  smallIcon?: string;
  largeIcon?: string;
  wantAgentInfo?: WantAgentInfo;
}

export interface WantAgentInfo {
  bundleName: string;
  abilityName: string;
  parameters?: Record<string, Object>;
}

export class NotificationService {
  private static instance: NotificationService | null = null;
  private context: common.UIAbilityContext | null = null;
  private notificationId: number = 1000;

  private constructor() {}

  static getInstance(): NotificationService {
    if (!NotificationService.instance) {
      NotificationService.instance = new NotificationService();
    }
    return NotificationService.instance;
  }

  /**
   * 初始化通知服务
   */
  async initialize(context: common.UIAbilityContext): Promise<void> {
    try {
      this.context = context;
      
      // 请求通知权限
      await this.requestNotificationPermission();
      
      hilog.info(DOMAIN_NUMBER, TAG, '✅ 通知服务初始化成功');
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `❌ 通知服务初始化失败: ${JSON.stringify(error)}`);
      throw error;
    }
  }

  /**
   * 请求通知权限
   */
  private async requestNotificationPermission(): Promise<void> {
    try {
      const isEnabled = await notificationManager.isNotificationEnabled();
      if (!isEnabled) {
        hilog.warn(DOMAIN_NUMBER, TAG, '通知权限未开启，请在设置中开启');
        // 可以引导用户去设置页面开启通知权限
        await notificationManager.requestEnableNotification();
      }
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `请求通知权限失败: ${JSON.stringify(error)}`);
      throw error;
    }
  }

  /**
   * 发送本地通知
   */
  async sendLocalNotification(data: NotificationData): Promise<void> {
    try {
      if (!this.context) {
        throw new Error('NotificationService未初始化');
      }

      let wantAgentObj: WantAgent | undefined;
      
      // 如果提供了WantAgent信息，创建WantAgent
      if (data.wantAgentInfo) {
        const wantAgentInfo = {
          wants: [{
            bundleName: data.wantAgentInfo.bundleName,
            abilityName: data.wantAgentInfo.abilityName,
            parameters: data.wantAgentInfo.parameters || {}
          }],
          operationType: wantAgent.OperationType.START_ABILITY,
          requestCode: data.id,
          wantAgentFlags: [wantAgent.WantAgentFlags.CONSTANT_FLAG]
        };

        wantAgentObj = await wantAgent.getWantAgent(wantAgentInfo);
      }

      // 构建通知内容
      const notificationRequest: notificationManager.NotificationRequest = {
        id: data.id,
        content: {
          contentType: notificationManager.ContentType.NOTIFICATION_CONTENT_BASIC_TEXT,
          normal: {
            title: data.title,
            text: data.content,
            additionalText: data.additionalText || ''
          }
        },
        wantAgent: wantAgentObj,
        slotType: notificationManager.SlotType.SOCIAL_COMMUNICATION,
        isOngoing: false,
        isUnremovable: false,
        deliveryTime: new Date().getTime(),
        tapDismissed: true,
        autoDeletedTime: new Date().getTime() + 24 * 60 * 60 * 1000, // 24小时后自动删除
        showDeliveryTime: true
      };

      // 发送通知
      await notificationManager.publish(notificationRequest);
      
      hilog.info(DOMAIN_NUMBER, TAG, `✅ 本地通知发送成功: ${data.title}`);
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `❌ 发送本地通知失败: ${JSON.stringify(error)}`);
      throw error;
    }
  }

  /**
   * 发送消息通知
   */
  async sendMessageNotification(senderName: string, message: string, groupName?: string): Promise<void> {
    const title = groupName ? `${groupName}` : senderName;
    const content = groupName ? `${senderName}: ${message}` : message;
    
    await this.sendLocalNotification({
      id: this.generateNotificationId(),
      title: title,
      content: content,
      wantAgentInfo: {
        bundleName: this.context?.abilityInfo.bundleName || '',
        abilityName: 'EntryAbility',
        parameters: {
          action: 'openChat',
          groupName: groupName,
          senderName: senderName
        }
      }
    });
  }

  /**
   * 发送系统通知
   */
  async sendSystemNotification(title: string, content: string): Promise<void> {
    await this.sendLocalNotification({
      id: this.generateNotificationId(),
      title: title,
      content: content,
      wantAgentInfo: {
        bundleName: this.context?.abilityInfo.bundleName || '',
        abilityName: 'EntryAbility'
      }
    });
  }

  /**
   * 取消指定通知
   */
  async cancelNotification(id: number): Promise<void> {
    try {
      await notificationManager.cancel(id);
      hilog.info(DOMAIN_NUMBER, TAG, `✅ 通知已取消: ${id}`);
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `❌ 取消通知失败: ${JSON.stringify(error)}`);
    }
  }

  /**
   * 取消所有通知
   */
  async cancelAllNotifications(): Promise<void> {
    try {
      await notificationManager.cancelAll();
      hilog.info(DOMAIN_NUMBER, TAG, '✅ 所有通知已取消');
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `❌ 取消所有通知失败: ${JSON.stringify(error)}`);
    }
  }

  /**
   * 检查通知权限状态
   */
  async isNotificationEnabled(): Promise<boolean> {
    try {
      return await notificationManager.isNotificationEnabled();
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `❌ 检查通知权限失败: ${JSON.stringify(error)}`);
      return false;
    }
  }

  /**
   * 生成通知ID
   */
  private generateNotificationId(): number {
    return ++this.notificationId;
  }

  /**
   * 清理资源
   */
  async cleanup(): Promise<void> {
    try {
      await this.cancelAllNotifications();
      this.context = null;
      hilog.info(DOMAIN_NUMBER, TAG, '✅ 通知服务清理完成');
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `❌ 通知服务清理失败: ${JSON.stringify(error)}`);
    }
  }
}
