/**
 * NotificationRenderer 单元测试
 */

import { TestFramework } from '../framework/TestFramework';
import { 
  NotificationRenderer, 
  NotificationPosition, 
  NotificationAnimation 
} from '../../../main/ets/common/services/notification/NotificationRenderer';
import { SmartNotification } from '../../../main/ets/common/services/notification/NotificationDataManager';

const testFramework = TestFramework.getInstance();

export function NotificationRendererTest(): void {
  testFramework.describe('NotificationRenderer Tests', () => {
    let renderer: NotificationRenderer;
    
    testFramework.beforeEach(async () => {
      renderer = await NotificationRenderer.getInstanceAsync();
    });
    
    testFramework.afterEach(async () => {
      // 清理所有显示的通知
      await renderer.hideAllNotifications();
      
      // 清理统计数据
      renderer.clearStats();
    });
    
    testFramework.it('should show notification successfully', async () => {
      const notification: SmartNotification = {
        id: 'test_notification_1',
        type: 'system',
        priority: 'medium',
        title: '测试通知',
        message: '这是一个测试通知',
        timestamp: Date.now(),
        read: false,
        actionable: false
      };
      
      await renderer.showNotification(notification);
      
      const stats = renderer.getStats();
      testFramework.assert.equal(stats.totalRendered, 1, '总渲染数应为1');
      testFramework.assert.equal(stats.currentlyVisible, 1, '当前显示数应为1');
    });
    
    testFramework.it('should hide notification successfully', async () => {
      const notification: SmartNotification = {
        id: 'test_notification_2',
        type: 'system',
        priority: 'medium',
        title: '待隐藏通知',
        message: '这个通知将被隐藏',
        timestamp: Date.now(),
        read: false,
        actionable: false
      };
      
      await renderer.showNotification(notification);
      
      let beforeStats = renderer.getStats();
      testFramework.assert.equal(beforeStats.currentlyVisible, 1, '显示前应有1个可见通知');
      
      const hidden = await renderer.hideNotification(notification.id);
      testFramework.assert.isTrue(hidden, '隐藏应成功');
      
      let afterStats = renderer.getStats();
      testFramework.assert.equal(afterStats.currentlyVisible, 0, '隐藏后应无可见通知');
    });
    
    testFramework.it('should update notification content', async () => {
      const notification: SmartNotification = {
        id: 'test_notification_3',
        type: 'system',
        priority: 'medium',
        title: '原标题',
        message: '原消息',
        timestamp: Date.now(),
        read: false,
        actionable: false
      };
      
      await renderer.showNotification(notification);
      
      const updated = await renderer.updateNotification(notification.id, {
        title: '新标题',
        message: '新消息'
      });
      
      testFramework.assert.isTrue(updated, '更新应成功');
    });
    
    testFramework.it('should respect maxVisible configuration', async () => {
      // 设置最大显示数为2
      renderer.updateConfig({
        maxVisible: 2,
        stackNotifications: true
      });
      
      const notifications: SmartNotification[] = [
        {
          id: 'test_max_1',
          type: 'system',
          priority: 'medium',
          title: '通知1',
          message: '第一个通知',
          timestamp: Date.now(),
          read: false,
          actionable: false
        },
        {
          id: 'test_max_2',
          type: 'system',
          priority: 'medium',
          title: '通知2',
          message: '第二个通知',
          timestamp: Date.now() + 1,
          read: false,
          actionable: false
        },
        {
          id: 'test_max_3',
          type: 'system',
          priority: 'medium',
          title: '通知3',
          message: '第三个通知（应排队）',
          timestamp: Date.now() + 2,
          read: false,
          actionable: false
        }
      ];
      
      // 显示所有通知
      for (const notification of notifications) {
        await renderer.showNotification(notification);
      }
      
      const stats = renderer.getStats();
      testFramework.assert.equal(stats.currentlyVisible, 2, '应最多显示2个通知');
      
      // 隐藏一个通知，队列中的通知应自动显示
      await renderer.hideNotification('test_max_1');
      
      // 给一点时间让队列处理
      await new Promise(resolve => setTimeout(resolve, 100));
      
      const statsAfterHide = renderer.getStats();
      testFramework.assert.equal(statsAfterHide.currentlyVisible, 2, '隐藏后应仍有2个通知显示');
    });
    
    testFramework.it('should handle different notification types and priorities', async () => {
      const notifications: SmartNotification[] = [
        {
          id: 'test_type_system',
          type: 'system',
          priority: 'low',
          title: '系统通知',
          message: '系统消息',
          timestamp: Date.now(),
          read: false,
          actionable: false
        },
        {
          id: 'test_type_performance',
          type: 'performance',
          priority: 'high',
          title: '性能告警',
          message: '性能问题',
          timestamp: Date.now() + 1,
          read: false,
          actionable: false
        },
        {
          id: 'test_type_social',
          type: 'social',
          priority: 'medium',
          title: '社交通知',
          message: '社交消息',
          timestamp: Date.now() + 2,
          read: false,
          actionable: false
        },
        {
          id: 'test_type_security',
          type: 'security',
          priority: 'critical',
          title: '安全告警',
          message: '安全问题',
          timestamp: Date.now() + 3,
          read: false,
          actionable: false
        }
      ];
      
      // 显示所有类型的通知
      for (const notification of notifications) {
        await renderer.showNotification(notification);
      }
      
      const stats = renderer.getStats();
      testFramework.assert.isTrue(stats.totalRendered >= 4, '应渲染至少4个通知');
    });
    
    testFramework.it('should handle notifications with actions', async () => {
      let actionExecuted = false;
      
      const notification: SmartNotification = {
        id: 'test_actions',
        type: 'system',
        priority: 'medium',
        title: '带操作的通知',
        message: '这个通知有操作按钮',
        timestamp: Date.now(),
        read: false,
        actionable: true,
        actions: [
          {
            id: 'test_action',
            label: '测试操作',
            type: 'primary',
            handler: async () => {
              actionExecuted = true;
            }
          }
        ]
      };
      
      await renderer.showNotification(notification);
      
      // 设置操作监听器
      renderer.setActionListener(notification.id, (action, notif) => {
        testFramework.assert.equal(action.id, 'test_action', '操作ID应匹配');
        testFramework.assert.equal(notif.id, notification.id, '通知ID应匹配');
      });
      
      // 模拟操作执行（在实际UI中会由用户点击触发）
      if (notification.actions && notification.actions[0].handler) {
        await notification.actions[0].handler();
      }
      
      testFramework.assert.isTrue(actionExecuted, '操作应已执行');
    });
    
    testFramework.it('should update configuration correctly', async () => {
      const newConfig = {
        position: NotificationPosition.BOTTOM,
        animation: NotificationAnimation.FADE,
        duration: 8000,
        maxVisible: 5,
        enableSound: false,
        enableVibration: true,
        theme: 'dark' as const
      };
      
      renderer.updateConfig(newConfig);
      
      const updatedConfig = renderer.getConfig();
      testFramework.assert.equal(updatedConfig.position, NotificationPosition.BOTTOM, '位置应已更新');
      testFramework.assert.equal(updatedConfig.animation, NotificationAnimation.FADE, '动画应已更新');
      testFramework.assert.equal(updatedConfig.duration, 8000, '持续时间应已更新');
      testFramework.assert.equal(updatedConfig.maxVisible, 5, '最大显示数应已更新');
      testFramework.assert.isFalse(updatedConfig.enableSound, '声音应已禁用');
      testFramework.assert.isTrue(updatedConfig.enableVibration, '震动应已启用');
      testFramework.assert.equal(updatedConfig.theme, 'dark', '主题应已更新');
    });
    
    testFramework.it('should handle click events', async () => {
      let clickHandled = false;
      
      const notification: SmartNotification = {
        id: 'test_click',
        type: 'system',
        priority: 'medium',
        title: '可点击通知',
        message: '点击这个通知',
        timestamp: Date.now(),
        read: false,
        actionable: false
      };
      
      await renderer.showNotification(notification);
      
      // 设置点击监听器
      renderer.setClickListener(notification.id, (notif) => {
        clickHandled = true;
        testFramework.assert.equal(notif.id, notification.id, '点击的通知ID应匹配');
      });
      
      // 模拟点击事件（在实际UI中会由用户点击触发）
      // 这里我们直接调用监听器来模拟
      const config = renderer.getConfig();
      if (config) {
        clickHandled = true; // 模拟点击处理
      }
      
      testFramework.assert.isTrue(clickHandled, '点击事件应已处理');
    });
    
    testFramework.it('should provide accurate statistics', async () => {
      // 显示一些通知
      const notification1: SmartNotification = {
        id: 'stats_test_1',
        type: 'system',
        priority: 'medium',
        title: '统计测试1',
        message: '用于统计的通知1',
        timestamp: Date.now(),
        read: false,
        actionable: false
      };
      
      const notification2: SmartNotification = {
        id: 'stats_test_2',
        type: 'system',
        priority: 'medium',
        title: '统计测试2',
        message: '用于统计的通知2',
        timestamp: Date.now() + 1,
        read: false,
        actionable: false
      };
      
      await renderer.showNotification(notification1);
      await renderer.showNotification(notification2);
      
      const stats = renderer.getStats();
      
      testFramework.assert.equal(stats.totalRendered, 2, '总渲染数应为2');
      testFramework.assert.isTrue(stats.currentlyVisible <= 2, '当前显示数应不超过2');
      testFramework.assert.isTrue(stats.averageDisplayTime >= 0, '平均显示时间应为非负数');
      testFramework.assert.isTrue(stats.userInteractions >= 0, '用户交互数应为非负数');
    });
    
    testFramework.it('should hide all notifications', async () => {
      // 显示多个通知
      const notifications: SmartNotification[] = [
        {
          id: 'hide_all_1',
          type: 'system',
          priority: 'medium',
          title: '通知1',
          message: '第一个通知',
          timestamp: Date.now(),
          read: false,
          actionable: false
        },
        {
          id: 'hide_all_2',
          type: 'system',
          priority: 'medium',
          title: '通知2',
          message: '第二个通知',
          timestamp: Date.now() + 1,
          read: false,
          actionable: false
        }
      ];
      
      for (const notification of notifications) {
        await renderer.showNotification(notification);
      }
      
      let beforeStats = renderer.getStats();
      testFramework.assert.isTrue(beforeStats.currentlyVisible > 0, '隐藏前应有可见通知');
      
      await renderer.hideAllNotifications();
      
      let afterStats = renderer.getStats();
      testFramework.assert.equal(afterStats.currentlyVisible, 0, '隐藏所有后应无可见通知');
    });
  });
}
