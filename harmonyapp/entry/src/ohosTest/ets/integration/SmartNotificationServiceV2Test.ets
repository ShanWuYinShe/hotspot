/**
 * SmartNotificationServiceV2 集成测试
 */

import { TestFramework } from '../framework/TestFramework';
import { 
  SmartNotificationServiceV2, 
  NotificationEvent 
} from '../../../main/ets/common/services/notification/SmartNotificationServiceV2';
import { NotificationPosition, NotificationAnimation } from '../../../main/ets/common/services/notification/NotificationRenderer';

const testFramework = TestFramework.getInstance();

export function SmartNotificationServiceV2Test(): void {
  testFramework.describe('SmartNotificationServiceV2 Integration Tests', () => {
    let notificationService: SmartNotificationServiceV2;
    
    testFramework.beforeEach(async () => {
      notificationService = await SmartNotificationServiceV2.getInstanceAsync();
    });
    
    testFramework.afterEach(async () => {
      // 清理所有通知
      await notificationService.hideAllNotifications();
      
      const notifications = notificationService.getNotifications();
      for (const notification of notifications) {
        await notificationService.deleteNotification(notification.id);
      }
    });
    
    testFramework.it('should create and manage notifications end-to-end', async () => {
      // 创建通知
      const notificationId = await notificationService.createNotification({
        type: 'system',
        priority: 'medium',
        title: '端到端测试',
        message: '这是一个完整的端到端测试',
        read: false,
        actionable: true,
        actions: [
          {
            id: 'test_action',
            label: '测试操作',
            type: 'primary',
            handler: async () => {
              // 测试操作处理
            }
          }
        ]
      }, { immediate: true });
      
      testFramework.assert.isNotNull(notificationId, '应创建通知成功');
      
      // 获取通知
      const notification = notificationService.getNotification(notificationId!);
      testFramework.assert.isNotNull(notification, '应能获取到通知');
      testFramework.assert.equal(notification!.title, '端到端测试', '标题应匹配');
      
      // 显示通知
      const shown = await notificationService.showNotification(notificationId!);
      testFramework.assert.isTrue(shown, '应能显示通知');
      
      // 更新通知
      const updated = await notificationService.updateNotification(notificationId!, {
        title: '更新后的标题'
      });
      testFramework.assert.isTrue(updated, '应能更新通知');
      
      // 标记为已读
      const marked = await notificationService.markAsRead(notificationId!);
      testFramework.assert.isTrue(marked, '应能标记为已读');
      
      // 隐藏通知
      const hidden = await notificationService.hideNotification(notificationId!);
      testFramework.assert.isTrue(hidden, '应能隐藏通知');
      
      // 删除通知
      const deleted = await notificationService.deleteNotification(notificationId!);
      testFramework.assert.isTrue(deleted, '应能删除通知');
    });
    
    testFramework.it('should handle configuration updates', async () => {
      const newConfig = {
        enablePerformanceAlerts: false,
        enableSocialNotifications: true,
        enableSecurityAlerts: true,
        maxNotifications: 50,
        autoCleanupDays: 14,
        quietHours: {
          enabled: true,
          startTime: '23:00',
          endTime: '07:00'
        },
        position: NotificationPosition.BOTTOM,
        animation: NotificationAnimation.BOUNCE,
        duration: 8000,
        maxVisible: 5,
        enableSound: false,
        enableVibration: true,
        theme: 'dark' as const
      };
      
      notificationService.updateConfig(newConfig);
      
      const updatedConfig = notificationService.getConfig();
      testFramework.assert.isFalse(updatedConfig.enablePerformanceAlerts, '性能告警应已禁用');
      testFramework.assert.isTrue(updatedConfig.enableSocialNotifications, '社交通知应已启用');
      testFramework.assert.equal(updatedConfig.maxNotifications, 50, '最大通知数应已更新');
      testFramework.assert.equal(updatedConfig.position, NotificationPosition.BOTTOM, '位置应已更新');
      testFramework.assert.equal(updatedConfig.animation, NotificationAnimation.BOUNCE, '动画应已更新');
    });
    
    testFramework.it('should create performance alerts', async () => {
      // 启用性能告警
      notificationService.updateConfig({
        enablePerformanceAlerts: true
      });
      
      const alertId = await notificationService.createPerformanceAlert({
        type: 'memory_usage',
        level: 'warning',
        message: '内存使用率过高',
        value: 85,
        threshold: 80
      });
      
      testFramework.assert.isNotNull(alertId, '应创建性能告警成功');
      
      const alert = notificationService.getNotification(alertId!);
      testFramework.assert.isNotNull(alert, '应能获取到性能告警');
      testFramework.assert.equal(alert!.type, 'performance', '类型应为performance');
      testFramework.assert.equal(alert!.priority, 'high', '优先级应为high');
    });
    
    testFramework.it('should create social notifications', async () => {
      // 启用社交通知
      notificationService.updateConfig({
        enableSocialNotifications: true
      });
      
      const socialId = await notificationService.createSocialNotification({
        fromUser: '张三',
        type: 'message',
        content: '张三给您发送了一条消息'
      });
      
      testFramework.assert.isNotNull(socialId, '应创建社交通知成功');
      
      const social = notificationService.getNotification(socialId!);
      testFramework.assert.isNotNull(social, '应能获取到社交通知');
      testFramework.assert.equal(social!.type, 'social', '类型应为social');
      testFramework.assert.equal(social!.priority, 'medium', '优先级应为medium');
    });
    
    testFramework.it('should create security alerts', async () => {
      // 启用安全告警
      notificationService.updateConfig({
        enableSecurityAlerts: true
      });
      
      const securityId = await notificationService.createSecurityAlert({
        type: 'login_attempt',
        severity: 'high',
        description: '检测到异常登录尝试',
        location: '北京市'
      });
      
      testFramework.assert.isNotNull(securityId, '应创建安全告警成功');
      
      const security = notificationService.getNotification(securityId!);
      testFramework.assert.isNotNull(security, '应能获取到安全告警');
      testFramework.assert.equal(security!.type, 'security', '类型应为security');
      testFramework.assert.equal(security!.priority, 'high', '优先级应为high');
    });
    
    testFramework.it('should handle delayed notifications', async () => {
      let callbackExecuted = false;
      
      const delayedId = await notificationService.createNotification({
        type: 'system',
        priority: 'low',
        title: '延迟通知测试',
        message: '这是一个延迟通知',
        read: false,
        actionable: false
      }, {
        delay: 1000, // 1秒延迟
        callback: (success, id) => {
          callbackExecuted = true;
          testFramework.assert.isTrue(success, '延迟通知应成功');
          testFramework.assert.isNotNull(id, '应有通知ID');
        }
      });
      
      testFramework.assert.isNotNull(delayedId, '应返回任务ID');
      
      // 等待延迟时间
      await new Promise(resolve => setTimeout(resolve, 1500));
      
      testFramework.assert.isTrue(callbackExecuted, '回调应已执行');
      
      // 检查通知是否已创建
      const notifications = notificationService.getNotifications();
      const delayedNotification = notifications.find(n => n.title === '延迟通知测试');
      testFramework.assert.isNotNull(delayedNotification, '延迟通知应已创建');
    });
    
    testFramework.it('should handle event listeners', async () => {
      let createdEventFired = false;
      let clickedEventFired = false;
      let actionExecutedEventFired = false;
      
      // 添加事件监听器
      notificationService.addEventListener(NotificationEvent.CREATED, (event, notification) => {
        createdEventFired = true;
        testFramework.assert.equal(event, NotificationEvent.CREATED, '事件类型应匹配');
        testFramework.assert.isNotNull(notification, '通知对象不应为空');
      });
      
      notificationService.addEventListener(NotificationEvent.CLICKED, (event, notification) => {
        clickedEventFired = true;
        testFramework.assert.equal(event, NotificationEvent.CLICKED, '事件类型应匹配');
      });
      
      notificationService.addEventListener(NotificationEvent.ACTION_EXECUTED, (event, notification, action) => {
        actionExecutedEventFired = true;
        testFramework.assert.equal(event, NotificationEvent.ACTION_EXECUTED, '事件类型应匹配');
        testFramework.assert.isNotNull(action, '操作对象不应为空');
      });
      
      // 创建通知
      const notificationId = await notificationService.createNotification({
        type: 'system',
        priority: 'medium',
        title: '事件测试通知',
        message: '用于测试事件的通知',
        read: false,
        actionable: true,
        actions: [
          {
            id: 'test_event_action',
            label: '测试事件操作',
            type: 'primary',
            handler: async () => {
              // 操作处理逻辑
            }
          }
        ]
      }, { immediate: true });
      
      testFramework.assert.isTrue(createdEventFired, '创建事件应已触发');
      
      // 注意：在实际环境中，点击和操作事件需要用户交互才能触发
      // 这里我们只验证监听器是否正确设置
      testFramework.assert.isNotNull(notificationId, '通知应创建成功');
    });
    
    testFramework.it('should filter notifications correctly', async () => {
      // 创建不同类型的通知
      await notificationService.createNotification({
        type: 'system',
        priority: 'high',
        title: '系统通知1',
        message: '系统消息1',
        read: false,
        actionable: false
      }, { immediate: true });
      
      await notificationService.createNotification({
        type: 'performance',
        priority: 'medium',
        title: '性能通知1',
        message: '性能消息1',
        read: true,
        actionable: false
      }, { immediate: true });
      
      await notificationService.createNotification({
        type: 'system',
        priority: 'low',
        title: '系统通知2',
        message: '系统消息2',
        read: false,
        actionable: false
      }, { immediate: true });
      
      // 测试类型过滤
      const systemNotifications = notificationService.getNotifications({ type: 'system' });
      testFramework.assert.equal(systemNotifications.length, 2, '应有2个系统通知');
      
      // 测试优先级过滤
      const highPriorityNotifications = notificationService.getNotifications({ priority: 'high' });
      testFramework.assert.equal(highPriorityNotifications.length, 1, '应有1个高优先级通知');
      
      // 测试未读通知
      const unreadNotifications = notificationService.getUnreadNotifications();
      testFramework.assert.equal(unreadNotifications.length, 2, '应有2个未读通知');
    });
    
    testFramework.it('should provide comprehensive statistics', async () => {
      // 创建一些通知来生成统计数据
      await notificationService.createNotification({
        type: 'system',
        priority: 'medium',
        title: '统计测试1',
        message: '用于统计的通知1',
        read: false,
        actionable: false
      }, { immediate: true });
      
      await notificationService.createNotification({
        type: 'performance',
        priority: 'high',
        title: '统计测试2',
        message: '用于统计的通知2',
        read: false,
        actionable: false
      }, { immediate: true });
      
      const stats = notificationService.getComprehensiveStats();
      
      // 验证数据统计
      testFramework.assert.isTrue(stats.data.total >= 2, '总通知数应至少为2');
      testFramework.assert.isTrue(stats.data.unread >= 2, '未读通知数应至少为2');
      
      // 验证调度统计
      testFramework.assert.isTrue(stats.scheduler.totalSent >= 2, '总发送数应至少为2');
      testFramework.assert.isTrue(stats.scheduler.totalScheduled >= 0, '总调度数应为非负数');
      
      // 验证渲染统计
      testFramework.assert.isTrue(stats.renderer.totalRendered >= 0, '总渲染数应为非负数');
      testFramework.assert.isTrue(stats.renderer.currentlyVisible >= 0, '当前显示数应为非负数');
      
      // 验证整体统计
      testFramework.assert.isTrue(stats.overall.totalProcessed >= 2, '总处理数应至少为2');
      testFramework.assert.isTrue(stats.overall.successRate >= 0, '成功率应为非负数');
      testFramework.assert.isTrue(stats.overall.averageProcessingTime >= 0, '平均处理时间应为非负数');
    });
    
    testFramework.it('should handle batch operations', async () => {
      // 批量标记为已读
      await notificationService.createNotification({
        type: 'system',
        priority: 'medium',
        title: '批量测试1',
        message: '批量操作测试1',
        read: false,
        actionable: false
      }, { immediate: true });
      
      await notificationService.createNotification({
        type: 'system',
        priority: 'medium',
        title: '批量测试2',
        message: '批量操作测试2',
        read: false,
        actionable: false
      }, { immediate: true });
      
      const markedCount = await notificationService.markAllAsRead({ type: 'system' });
      testFramework.assert.equal(markedCount, 2, '应标记2个通知为已读');
      
      const unreadAfterMark = notificationService.getUnreadNotifications();
      const systemUnread = unreadAfterMark.filter(n => n.type === 'system');
      testFramework.assert.equal(systemUnread.length, 0, '系统通知应全部已读');
    });
    
    testFramework.it('should respect feature toggles', async () => {
      // 禁用性能告警
      notificationService.updateConfig({
        enablePerformanceAlerts: false
      });
      
      const performanceAlertId = await notificationService.createPerformanceAlert({
        type: 'cpu_usage',
        level: 'warning',
        message: 'CPU使用率过高',
        value: 90,
        threshold: 80
      });
      
      testFramework.assert.isNull(performanceAlertId, '禁用时不应创建性能告警');
      
      // 禁用社交通知
      notificationService.updateConfig({
        enableSocialNotifications: false
      });
      
      const socialId = await notificationService.createSocialNotification({
        fromUser: '李四',
        type: 'friend_request',
        content: '李四请求添加您为好友'
      });
      
      testFramework.assert.isNull(socialId, '禁用时不应创建社交通知');
    });
  });
}
