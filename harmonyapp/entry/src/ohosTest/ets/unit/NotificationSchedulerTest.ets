/**
 * NotificationScheduler 单元测试
 */

import { TestFramework } from '../framework/TestFramework';
import { NotificationScheduler } from '../../../main/ets/common/services/notification/NotificationScheduler';
import { NotificationDataManager } from '../../../main/ets/common/services/notification/NotificationDataManager';

const testFramework = TestFramework.getInstance();

export function NotificationSchedulerTest(): void {
  testFramework.describe('NotificationScheduler Tests', () => {
    let scheduler: NotificationScheduler;
    let dataManager: NotificationDataManager;
    
    testFramework.beforeEach(async () => {
      scheduler = await NotificationScheduler.getInstanceAsync();
      dataManager = await NotificationDataManager.getInstanceAsync();
    });
    
    testFramework.afterEach(async () => {
      // 清理测试数据
      const notifications = dataManager.getNotifications();
      const ids = notifications.map(n => n.id);
      await dataManager.deleteNotifications(ids);
      
      // 清理统计数据
      scheduler.clearStats();
    });
    
    testFramework.it('should send notification immediately', async () => {
      const notificationData = {
        type: 'system' as const,
        priority: 'medium' as const,
        title: '立即发送测试',
        message: '这是立即发送的通知',
        read: false,
        actionable: false
      };
      
      let callbackCalled = false;
      let callbackSuccess = false;
      let callbackNotificationId: string | undefined;
      
      const notificationId = await scheduler.sendImmediately(
        notificationData,
        (success, id) => {
          callbackCalled = true;
          callbackSuccess = success;
          callbackNotificationId = id;
        }
      );
      
      testFramework.assert.isNotNull(notificationId, '应返回通知ID');
      testFramework.assert.isTrue(callbackCalled, '回调应被调用');
      testFramework.assert.isTrue(callbackSuccess, '回调应报告成功');
      testFramework.assert.equal(callbackNotificationId, notificationId, '回调中的ID应匹配');
      
      // 验证通知已创建
      const notification = dataManager.getNotification(notificationId!);
      testFramework.assert.isNotNull(notification, '通知应已创建');
      testFramework.assert.equal(notification!.title, notificationData.title, '标题应匹配');
    });
    
    testFramework.it('should schedule notification with delay', async () => {
      const notificationData = {
        type: 'system' as const,
        priority: 'low' as const,
        title: '延迟发送测试',
        message: '这是延迟发送的通知',
        read: false,
        actionable: false
      };
      
      let callbackCalled = false;
      const delay = 1000; // 1秒延迟
      
      const taskId = scheduler.schedule(
        notificationData,
        delay,
        (success, id) => {
          callbackCalled = true;
          testFramework.assert.isTrue(success, '延迟发送应成功');
          testFramework.assert.isNotNull(id, '应有通知ID');
        }
      );
      
      testFramework.assert.isNotNull(taskId, '应返回任务ID');
      
      // 立即检查，通知还不应该存在
      const immediateCheck = dataManager.getNotifications();
      testFramework.assert.equal(immediateCheck.length, 0, '通知不应立即创建');
      
      // 等待延迟时间
      await new Promise(resolve => setTimeout(resolve, delay + 500));
      
      // 现在应该有通知了
      const delayedCheck = dataManager.getNotifications();
      testFramework.assert.equal(delayedCheck.length, 1, '延迟后应有1个通知');
      testFramework.assert.isTrue(callbackCalled, '回调应已被调用');
    });
    
    testFramework.it('should cancel scheduled task', async () => {
      const notificationData = {
        type: 'system' as const,
        priority: 'low' as const,
        title: '取消测试',
        message: '这个通知应被取消',
        read: false,
        actionable: false
      };
      
      const taskId = scheduler.schedule(notificationData, 2000); // 2秒延迟
      
      // 取消任务
      const cancelled = scheduler.cancelScheduledTask(taskId);
      testFramework.assert.isTrue(cancelled, '取消应成功');
      
      // 等待原本的延迟时间
      await new Promise(resolve => setTimeout(resolve, 2500));
      
      // 通知不应被创建
      const notifications = dataManager.getNotifications();
      testFramework.assert.equal(notifications.length, 0, '被取消的通知不应被创建');
    });
    
    testFramework.it('should respect quiet hours', async () => {
      // 设置当前时间为静默时间
      const now = new Date();
      const currentTime = `${now.getHours().toString().padStart(2, '0')}:${now.getMinutes().toString().padStart(2, '0')}`;
      
      scheduler.updateConfig({
        quietHours: {
          enabled: true,
          startTime: currentTime,
          endTime: currentTime // 设置为当前时间，确保在静默时间内
        }
      });
      
      const notificationData = {
        type: 'system' as const,
        priority: 'medium' as const,
        title: '静默时间测试',
        message: '这个通知应被静默时间阻止',
        read: false,
        actionable: false
      };
      
      let callbackCalled = false;
      const notificationId = await scheduler.sendImmediately(
        notificationData,
        (success) => {
          callbackCalled = true;
          testFramework.assert.isFalse(success, '静默时间内发送应失败');
        }
      );
      
      testFramework.assert.isNull(notificationId, '静默时间内不应返回通知ID');
      testFramework.assert.isTrue(callbackCalled, '回调应被调用');
      
      const stats = scheduler.getStats();
      testFramework.assert.isTrue(stats.quietHoursBlocked > 0, '静默时间阻止统计应增加');
    });
    
    testFramework.it('should handle throttling', async () => {
      // 配置严格的限流
      scheduler.updateConfig({
        enableThrottling: true,
        throttleWindow: 5000, // 5秒窗口
        maxNotificationsPerWindow: 2 // 最多2个通知
      });
      
      const notificationData = {
        type: 'system' as const,
        priority: 'low' as const,
        title: '限流测试',
        message: '测试限流功能',
        read: false,
        actionable: false
      };
      
      // 发送前两个通知应该成功
      const id1 = await scheduler.sendImmediately(notificationData);
      const id2 = await scheduler.sendImmediately(notificationData);
      
      testFramework.assert.isNotNull(id1, '第1个通知应成功');
      testFramework.assert.isNotNull(id2, '第2个通知应成功');
      
      // 第三个通知应被限流
      let throttledCallbackCalled = false;
      const id3 = await scheduler.sendImmediately(
        notificationData,
        (success) => {
          throttledCallbackCalled = true;
          testFramework.assert.isFalse(success, '第3个通知应被限流');
        }
      );
      
      testFramework.assert.isNull(id3, '第3个通知应返回null');
      testFramework.assert.isTrue(throttledCallbackCalled, '限流回调应被调用');
      
      const stats = scheduler.getStats();
      testFramework.assert.isTrue(stats.throttledCount > 0, '限流统计应增加');
    });
    
    testFramework.it('should batch process notifications', async () => {
      const notifications = [
        {
          notification: {
            type: 'system' as const,
            priority: 'low' as const,
            title: '批量通知1',
            message: '批量处理测试1',
            read: false,
            actionable: false
          },
          delay: 500
        },
        {
          notification: {
            type: 'system' as const,
            priority: 'low' as const,
            title: '批量通知2',
            message: '批量处理测试2',
            read: false,
            actionable: false
          },
          delay: 600
        },
        {
          notification: {
            type: 'system' as const,
            priority: 'low' as const,
            title: '批量通知3',
            message: '批量处理测试3',
            read: false,
            actionable: false
          },
          delay: 700
        }
      ];
      
      const taskIds = scheduler.scheduleBatch(notifications);
      
      testFramework.assert.equal(taskIds.length, 3, '应返回3个任务ID');
      
      // 等待所有通知处理完成
      await new Promise(resolve => setTimeout(resolve, 1500));
      
      const createdNotifications = dataManager.getNotifications();
      testFramework.assert.equal(createdNotifications.length, 3, '应创建3个通知');
    });
    
    testFramework.it('should retry failed notifications', async () => {
      // 这个测试需要模拟失败情况，在实际环境中可能需要mock
      // 这里我们测试重试配置是否正确设置
      
      const config = scheduler.getConfig();
      testFramework.assert.isTrue(config.retryAttempts > 0, '应配置重试次数');
      testFramework.assert.isTrue(config.retryDelay > 0, '应配置重试延迟');
    });
    
    testFramework.it('should update configuration correctly', async () => {
      const newConfig = {
        batchDelay: 2000,
        maxBatchSize: 20,
        retryAttempts: 5,
        enableThrottling: false
      };
      
      scheduler.updateConfig(newConfig);
      
      const updatedConfig = scheduler.getConfig();
      testFramework.assert.equal(updatedConfig.batchDelay, 2000, '批量延迟应已更新');
      testFramework.assert.equal(updatedConfig.maxBatchSize, 20, '批量大小应已更新');
      testFramework.assert.equal(updatedConfig.retryAttempts, 5, '重试次数应已更新');
      testFramework.assert.isFalse(updatedConfig.enableThrottling, '限流应已禁用');
    });
    
    testFramework.it('should provide accurate statistics', async () => {
      // 发送一些通知来生成统计数据
      await scheduler.sendImmediately({
        type: 'system',
        priority: 'medium',
        title: '统计测试1',
        message: '用于统计的通知1',
        read: false,
        actionable: false
      });
      
      await scheduler.sendImmediately({
        type: 'system',
        priority: 'medium',
        title: '统计测试2',
        message: '用于统计的通知2',
        read: false,
        actionable: false
      });
      
      const stats = scheduler.getStats();
      
      testFramework.assert.isTrue(stats.totalScheduled >= 0, '总调度数应为非负数');
      testFramework.assert.isTrue(stats.totalSent >= 2, '总发送数应至少为2');
      testFramework.assert.isTrue(stats.totalFailed >= 0, '总失败数应为非负数');
      testFramework.assert.isTrue(stats.pendingTasks >= 0, '待处理任务数应为非负数');
    });
  });
}
