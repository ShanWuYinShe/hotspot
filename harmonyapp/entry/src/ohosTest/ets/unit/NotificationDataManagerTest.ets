/**
 * NotificationDataManager 单元测试
 */

import { TestFramework } from '../framework/TestFramework';
import { NotificationDataManager, SmartNotification } from '../../../main/ets/common/services/notification/NotificationDataManager';

const testFramework = TestFramework.getInstance();

export function NotificationDataManagerTest(): void {
  testFramework.describe('NotificationDataManager Tests', () => {
    let dataManager: NotificationDataManager;
    
    testFramework.beforeEach(async () => {
      dataManager = await NotificationDataManager.getInstanceAsync();
    });
    
    testFramework.afterEach(async () => {
      // 清理测试数据
      const notifications = dataManager.getNotifications();
      const ids = notifications.map(n => n.id);
      await dataManager.deleteNotifications(ids);
    });
    
    testFramework.it('should create notification successfully', async () => {
      const notificationData = {
        type: 'system' as const,
        priority: 'medium' as const,
        title: '测试通知',
        message: '这是一个测试通知',
        read: false,
        actionable: false
      };
      
      const id = await dataManager.addNotification(notificationData);
      
      testFramework.assert.isNotNull(id, '通知ID不应为空');
      testFramework.assert.isString(id, '通知ID应为字符串');
      
      const notification = dataManager.getNotification(id);
      testFramework.assert.isNotNull(notification, '应能获取到创建的通知');
      testFramework.assert.equal(notification!.title, notificationData.title, '标题应匹配');
      testFramework.assert.equal(notification!.message, notificationData.message, '消息应匹配');
    });
    
    testFramework.it('should update notification successfully', async () => {
      const id = await dataManager.addNotification({
        type: 'system',
        priority: 'low',
        title: '原标题',
        message: '原消息',
        read: false,
        actionable: false
      });
      
      const success = await dataManager.updateNotification(id, {
        title: '新标题',
        read: true
      });
      
      testFramework.assert.isTrue(success, '更新应成功');
      
      const notification = dataManager.getNotification(id);
      testFramework.assert.equal(notification!.title, '新标题', '标题应已更新');
      testFramework.assert.isTrue(notification!.read, '已读状态应已更新');
      testFramework.assert.equal(notification!.message, '原消息', '未更新的字段应保持不变');
    });
    
    testFramework.it('should delete notification successfully', async () => {
      const id = await dataManager.addNotification({
        type: 'system',
        priority: 'low',
        title: '待删除通知',
        message: '这个通知将被删除',
        read: false,
        actionable: false
      });
      
      const success = await dataManager.deleteNotification(id);
      testFramework.assert.isTrue(success, '删除应成功');
      
      const notification = dataManager.getNotification(id);
      testFramework.assert.isNull(notification, '删除后应无法获取到通知');
    });
    
    testFramework.it('should filter notifications correctly', async () => {
      // 创建不同类型的通知
      await dataManager.addNotification({
        type: 'system',
        priority: 'high',
        title: '系统通知1',
        message: '系统消息',
        read: false,
        actionable: false
      });
      
      await dataManager.addNotification({
        type: 'performance',
        priority: 'medium',
        title: '性能通知1',
        message: '性能消息',
        read: true,
        actionable: false
      });
      
      await dataManager.addNotification({
        type: 'system',
        priority: 'low',
        title: '系统通知2',
        message: '另一个系统消息',
        read: false,
        actionable: false
      });
      
      // 测试类型过滤
      const systemNotifications = dataManager.getNotifications({ type: 'system' });
      testFramework.assert.equal(systemNotifications.length, 2, '应有2个系统通知');
      
      // 测试优先级过滤
      const highPriorityNotifications = dataManager.getNotifications({ priority: 'high' });
      testFramework.assert.equal(highPriorityNotifications.length, 1, '应有1个高优先级通知');
      
      // 测试已读状态过滤
      const unreadNotifications = dataManager.getNotifications({ read: false });
      testFramework.assert.equal(unreadNotifications.length, 2, '应有2个未读通知');
    });
    
    testFramework.it('should mark notifications as read', async () => {
      const id1 = await dataManager.addNotification({
        type: 'system',
        priority: 'medium',
        title: '通知1',
        message: '消息1',
        read: false,
        actionable: false
      });
      
      const id2 = await dataManager.addNotification({
        type: 'system',
        priority: 'medium',
        title: '通知2',
        message: '消息2',
        read: false,
        actionable: false
      });
      
      // 标记单个通知为已读
      const success = await dataManager.markAsRead(id1);
      testFramework.assert.isTrue(success, '标记已读应成功');
      
      const notification1 = dataManager.getNotification(id1);
      testFramework.assert.isTrue(notification1!.read, '通知1应已标记为已读');
      
      const notification2 = dataManager.getNotification(id2);
      testFramework.assert.isFalse(notification2!.read, '通知2应仍为未读');
      
      // 批量标记为已读
      const markedCount = await dataManager.markAllAsRead({ type: 'system' });
      testFramework.assert.equal(markedCount, 1, '应标记1个通知为已读');
      
      const notification2Updated = dataManager.getNotification(id2);
      testFramework.assert.isTrue(notification2Updated!.read, '通知2现在应已标记为已读');
    });
    
    testFramework.it('should get correct statistics', async () => {
      // 创建测试数据
      await dataManager.addNotification({
        type: 'system',
        priority: 'high',
        title: '系统通知',
        message: '系统消息',
        read: false,
        actionable: false
      });
      
      await dataManager.addNotification({
        type: 'performance',
        priority: 'medium',
        title: '性能通知',
        message: '性能消息',
        read: true,
        actionable: false
      });
      
      const stats = dataManager.getStats();
      
      testFramework.assert.equal(stats.total, 2, '总通知数应为2');
      testFramework.assert.equal(stats.unread, 1, '未读通知数应为1');
      testFramework.assert.equal(stats.byType['system'], 1, '系统通知数应为1');
      testFramework.assert.equal(stats.byType['performance'], 1, '性能通知数应为1');
      testFramework.assert.equal(stats.byPriority['high'], 1, '高优先级通知数应为1');
      testFramework.assert.equal(stats.byPriority['medium'], 1, '中优先级通知数应为1');
    });
    
    testFramework.it('should validate notification data', async () => {
      // 测试空标题
      try {
        await dataManager.addNotification({
          type: 'system',
          priority: 'medium',
          title: '',
          message: '有效消息',
          read: false,
          actionable: false
        });
        testFramework.assert.fail('空标题应抛出错误');
      } catch (error) {
        testFramework.assert.isTrue(true, '空标题正确抛出错误');
      }
      
      // 测试过长标题
      try {
        await dataManager.addNotification({
          type: 'system',
          priority: 'medium',
          title: 'a'.repeat(100), // 超过最大长度
          message: '有效消息',
          read: false,
          actionable: false
        });
        testFramework.assert.fail('过长标题应抛出错误');
      } catch (error) {
        testFramework.assert.isTrue(true, '过长标题正确抛出错误');
      }
    });
    
    testFramework.it('should cleanup expired notifications', async () => {
      const now = Date.now();
      
      // 创建过期通知
      const id = await dataManager.addNotification({
        type: 'system',
        priority: 'medium',
        title: '过期通知',
        message: '这个通知已过期',
        read: false,
        actionable: false,
        expiresAt: now - 1000 // 1秒前过期
      });
      
      // 创建未过期通知
      await dataManager.addNotification({
        type: 'system',
        priority: 'medium',
        title: '有效通知',
        message: '这个通知未过期',
        read: false,
        actionable: false,
        expiresAt: now + 60000 // 1分钟后过期
      });
      
      const cleanedCount = await dataManager.cleanupExpiredNotifications();
      testFramework.assert.equal(cleanedCount, 1, '应清理1个过期通知');
      
      const expiredNotification = dataManager.getNotification(id);
      testFramework.assert.isNull(expiredNotification, '过期通知应已被删除');
      
      const remainingNotifications = dataManager.getNotifications();
      testFramework.assert.equal(remainingNotifications.length, 1, '应剩余1个有效通知');
    });
  });
}
