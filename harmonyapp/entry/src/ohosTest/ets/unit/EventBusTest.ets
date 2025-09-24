/**
 * EventBus 单元测试
 */

import { 
  TestSuite, 
  TestCase, 
  Assert, 
  createTestSuite, 
  createTestCase 
} from '../framework/TestFramework';
import { EventBus, Event, EventListener } from '../../../main/ets/common/events/EventBus';

/**
 * EventBus 测试套件
 */
export function createEventBusTestSuite(): TestSuite {
  const suite = createTestSuite('EventBus', 'EventBus 事件总线测试');

  let eventBus: EventBus;

  // 设置和清理
  suite.setup = async () => {
    eventBus = EventBus.getInstance();
    await eventBus.initialize();
    eventBus.removeAllListeners();
  };

  suite.teardown = async () => {
    eventBus.removeAllListeners();
    await eventBus.cleanup();
  };

  // 测试用例：基本事件发布和监听
  suite.tests.push(createTestCase(
    '应该能够发布和监听事件',
    async () => {
      let receivedEvent: Event | null = null;
      
      const listenerId = eventBus.on('test:event', (event) => {
        receivedEvent = event;
      });

      const eventId = eventBus.emit('test:event', { message: 'Hello World' }, 'TestSource');

      // 等待事件处理
      await new Promise(resolve => setTimeout(resolve, 10));

      Assert.isNotNull(receivedEvent);
      Assert.equals(receivedEvent!.type, 'test:event');
      Assert.equals(receivedEvent!.data.message, 'Hello World');
      Assert.equals(receivedEvent!.source, 'TestSource');
      Assert.equals(receivedEvent!.id, eventId);

      // 清理
      eventBus.off('test:event', listenerId);
    }
  ));

  // 测试用例：一次性监听器
  suite.tests.push(createTestCase(
    '应该能够使用一次性监听器',
    async () => {
      let callCount = 0;
      
      eventBus.once('test:once', () => {
        callCount++;
      });

      // 发布多次事件
      eventBus.emit('test:once', {});
      eventBus.emit('test:once', {});
      eventBus.emit('test:once', {});

      // 等待事件处理
      await new Promise(resolve => setTimeout(resolve, 10));

      Assert.equals(callCount, 1); // 只应该执行一次
      Assert.equals(eventBus.listenerCount('test:once'), 0); // 监听器应该被自动移除
    }
  ));

  // 测试用例：多个监听器
  suite.tests.push(createTestCase(
    '应该能够为同一事件添加多个监听器',
    async () => {
      let count1 = 0;
      let count2 = 0;
      let count3 = 0;

      const listener1 = eventBus.on('test:multiple', () => { count1++; });
      const listener2 = eventBus.on('test:multiple', () => { count2++; });
      const listener3 = eventBus.on('test:multiple', () => { count3++; });

      Assert.equals(eventBus.listenerCount('test:multiple'), 3);

      eventBus.emit('test:multiple', {});

      // 等待事件处理
      await new Promise(resolve => setTimeout(resolve, 10));

      Assert.equals(count1, 1);
      Assert.equals(count2, 1);
      Assert.equals(count3, 1);

      // 清理
      eventBus.off('test:multiple', listener1);
      eventBus.off('test:multiple', listener2);
      eventBus.off('test:multiple', listener3);
    }
  ));

  // 测试用例：事件优先级
  suite.tests.push(createTestCase(
    '应该能够按优先级执行监听器',
    async () => {
      const executionOrder: number[] = [];

      eventBus.on('test:priority', () => { executionOrder.push(1); }, { priority: 1 });
      eventBus.on('test:priority', () => { executionOrder.push(3); }, { priority: 3 });
      eventBus.on('test:priority', () => { executionOrder.push(2); }, { priority: 2 });

      eventBus.emit('test:priority', {});

      // 等待事件处理
      await new Promise(resolve => setTimeout(resolve, 10));

      Assert.arrayLength(executionOrder, 3);
      Assert.equals(executionOrder[0], 3); // 最高优先级先执行
      Assert.equals(executionOrder[1], 2);
      Assert.equals(executionOrder[2], 1);

      // 清理
      eventBus.removeAllListeners();
    }
  ));

  // 测试用例：事件过滤器
  suite.tests.push(createTestCase(
    '应该能够使用事件过滤器',
    async () => {
      let filteredCount = 0;
      let unfilteredCount = 0;

      // 只处理包含特定数据的事件
      eventBus.on('test:filter', () => { filteredCount++; }, {
        filter: (event) => event.data && event.data.shouldProcess === true
      });

      eventBus.on('test:filter', () => { unfilteredCount++; });

      // 发布不同的事件
      eventBus.emit('test:filter', { shouldProcess: true });
      eventBus.emit('test:filter', { shouldProcess: false });
      eventBus.emit('test:filter', {});

      // 等待事件处理
      await new Promise(resolve => setTimeout(resolve, 10));

      Assert.equals(filteredCount, 1); // 只有一个事件通过过滤器
      Assert.equals(unfilteredCount, 3); // 无过滤器的监听器处理所有事件

      // 清理
      eventBus.removeAllListeners();
    }
  ));

  // 测试用例：异步监听器
  suite.tests.push(createTestCase(
    '应该能够处理异步监听器',
    async () => {
      let asyncResult: string | null = null;

      eventBus.on('test:async', async (event) => {
        // 模拟异步操作
        await new Promise(resolve => setTimeout(resolve, 50));
        asyncResult = `processed: ${event.data.message}`;
      }, { async: true });

      eventBus.emit('test:async', { message: 'async test' });

      // 等待异步处理完成
      await new Promise(resolve => setTimeout(resolve, 100));

      Assert.equals(asyncResult, 'processed: async test');

      // 清理
      eventBus.removeAllListeners();
    }
  ));

  // 测试用例：超时处理
  suite.tests.push(createTestCase(
    '应该能够处理监听器超时',
    async () => {
      let timeoutOccurred = false;

      eventBus.on('test:timeout', async () => {
        // 模拟长时间运行的操作
        await new Promise(resolve => setTimeout(resolve, 200));
      }, { 
        async: true, 
        timeout: 100 // 100ms 超时
      });

      eventBus.emit('test:timeout', {});

      // 等待超时处理
      await new Promise(resolve => setTimeout(resolve, 150));

      // 检查错误统计（超时应该被记录为错误）
      const stats = eventBus.getStats();
      Assert.isTrue(stats.errorCount > 0);

      // 清理
      eventBus.removeAllListeners();
    }
  ));

  // 测试用例：等待特定事件
  suite.tests.push(createTestCase(
    '应该能够等待特定事件',
    async () => {
      // 在另一个异步任务中发布事件
      setTimeout(() => {
        eventBus.emit('test:wait', { result: 'success' });
      }, 50);

      // 等待事件
      const event = await eventBus.waitFor('test:wait', 1000);
      
      Assert.equals(event.type, 'test:wait');
      Assert.equals(event.data.result, 'success');
    },
    { timeout: 2000 }
  ));

  // 测试用例：等待事件超时
  suite.tests.push(createTestCase(
    '等待事件应该能够超时',
    async () => {
      await Assert.throws(async () => {
        await eventBus.waitFor('test:never', 100); // 100ms 超时
      });
    },
    { timeout: 1000 }
  ));

  // 测试用例：事件历史记录
  suite.tests.push(createTestCase(
    '应该能够记录和获取事件历史',
    () => {
      eventBus.clearHistory();

      eventBus.emit('test:history1', { data: 1 });
      eventBus.emit('test:history2', { data: 2 });
      eventBus.emit('test:history1', { data: 3 });

      const allHistory = eventBus.getHistory();
      Assert.arrayLength(allHistory, 3);

      const history1 = eventBus.getHistory('test:history1');
      Assert.arrayLength(history1, 2);
      Assert.equals(history1[0].data.data, 1);
      Assert.equals(history1[1].data.data, 3);

      const limitedHistory = eventBus.getHistory('test:history1', 1);
      Assert.arrayLength(limitedHistory, 1);
      Assert.equals(limitedHistory[0].data.data, 3); // 最新的一个
    }
  ));

  // 测试用例：移除监听器
  suite.tests.push(createTestCase(
    '应该能够移除特定的监听器',
    () => {
      let count = 0;
      const listenerId = eventBus.on('test:remove', () => { count++; });

      eventBus.emit('test:remove', {});
      Assert.equals(count, 1);

      const removed = eventBus.off('test:remove', listenerId);
      Assert.isTrue(removed);

      eventBus.emit('test:remove', {});
      Assert.equals(count, 1); // 不应该再增加
    }
  ));

  // 测试用例：移除所有监听器
  suite.tests.push(createTestCase(
    '应该能够移除所有监听器',
    () => {
      eventBus.on('test:removeall1', () => {});
      eventBus.on('test:removeall2', () => {});
      eventBus.on('test:removeall1', () => {});

      Assert.equals(eventBus.listenerCount(), 3);

      eventBus.removeAllListeners();

      Assert.equals(eventBus.listenerCount(), 0);
      Assert.arrayLength(eventBus.eventTypes(), 0);
    }
  ));

  // 测试用例：事件统计
  suite.tests.push(createTestCase(
    '应该能够获取事件统计信息',
    async () => {
      eventBus.removeAllListeners();
      
      eventBus.on('test:stats1', () => {});
      eventBus.on('test:stats1', () => {});
      eventBus.on('test:stats2', () => {});

      eventBus.emit('test:stats1', {});
      eventBus.emit('test:stats1', {});
      eventBus.emit('test:stats2', {});

      // 等待事件处理
      await new Promise(resolve => setTimeout(resolve, 10));

      const stats = eventBus.getStats();
      Assert.equals(stats.totalListeners, 3);
      Assert.equals(stats.totalEvents, 3);
      Assert.equals(stats.eventsByType['test:stats1'], 2);
      Assert.equals(stats.eventsByType['test:stats2'], 1);
      Assert.equals(stats.listenersByType['test:stats1'], 2);
      Assert.equals(stats.listenersByType['test:stats2'], 1);

      // 清理
      eventBus.removeAllListeners();
    }
  ));

  return suite;
}

/**
 * 运行 EventBus 测试
 */
export async function runEventBusTests(): Promise<void> {
  const { testRunner } = await import('../framework/TestFramework');
  const suite = createEventBusTestSuite();
  testRunner.addSuite(suite);
}
