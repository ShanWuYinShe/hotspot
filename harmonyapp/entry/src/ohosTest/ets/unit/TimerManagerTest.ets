/**
 * TimerManager 单元测试
 */

import { 
  TestSuite, 
  TestCase, 
  Assert, 
  createTestSuite, 
  createTestCase 
} from '../framework/TestFramework';
import { TimerManager, TimerStats } from '../../../main/ets/common/utils/TimerManager';

/**
 * TimerManager 测试套件
 */
export function createTimerManagerTestSuite(): TestSuite {
  const suite = createTestSuite('TimerManager', 'TimerManager 定时器管理器测试');

  // 设置和清理
  suite.setup = async () => {
    // 确保开始测试前清理所有定时器
    TimerManager.clearAll();
  };

  suite.teardown = async () => {
    // 测试结束后清理所有定时器
    TimerManager.clearAll();
  };

  // 测试用例：创建和清除定时器
  suite.tests.push(createTestCase(
    '应该能够创建和清除定时器',
    async () => {
      let executed = false;
      const timerId = TimerManager.setInterval(() => {
        executed = true;
      }, 100, '测试定时器');

      Assert.isType(timerId, 'number');
      Assert.isTrue(timerId > 0);

      // 等待一段时间确保定时器执行
      await new Promise(resolve => setTimeout(resolve, 150));
      Assert.isTrue(executed);

      // 清除定时器
      TimerManager.clearInterval(timerId);
      
      const stats = TimerManager.getStats();
      Assert.equals(stats.totalTimers, 0);
    },
    { timeout: 1000 }
  ));

  // 测试用例：超时定时器
  suite.tests.push(createTestCase(
    '应该能够创建超时定时器',
    async () => {
      let executed = false;
      const timerId = TimerManager.setTimeout(() => {
        executed = true;
      }, 100, '测试超时定时器');

      Assert.isFalse(executed);
      
      // 等待定时器执行
      await new Promise(resolve => setTimeout(resolve, 150));
      Assert.isTrue(executed);

      // 超时定时器应该自动清理
      const stats = TimerManager.getStats();
      Assert.equals(stats.totalTimers, 0);
    },
    { timeout: 1000 }
  ));

  // 测试用例：定时器统计
  suite.tests.push(createTestCase(
    '应该能够获取定时器统计信息',
    () => {
      const timer1 = TimerManager.setInterval(() => {}, 1000, '定时器1');
      const timer2 = TimerManager.setInterval(() => {}, 2000, '定时器2');
      const timer3 = TimerManager.setTimeout(() => {}, 3000, '超时定时器');

      const stats = TimerManager.getStats();
      Assert.equals(stats.totalTimers, 3);
      Assert.equals(stats.intervalTimers, 2);
      Assert.equals(stats.timeoutTimers, 1);
      Assert.isTrue(stats.averageDelay > 0);

      // 清理
      TimerManager.clearInterval(timer1);
      TimerManager.clearInterval(timer2);
      TimerManager.clearTimeout(timer3);
    }
  ));

  // 测试用例：清除所有定时器
  suite.tests.push(createTestCase(
    '应该能够清除所有定时器',
    () => {
      // 创建多个定时器
      TimerManager.setInterval(() => {}, 1000, '定时器A');
      TimerManager.setInterval(() => {}, 2000, '定时器B');
      TimerManager.setTimeout(() => {}, 3000, '超时定时器C');

      let stats = TimerManager.getStats();
      Assert.isTrue(stats.totalTimers > 0);

      // 清除所有定时器
      TimerManager.clearAll();

      stats = TimerManager.getStats();
      Assert.equals(stats.totalTimers, 0);
    }
  ));

  // 测试用例：获取所有定时器信息
  suite.tests.push(createTestCase(
    '应该能够获取所有定时器的详细信息',
    () => {
      const timer1 = TimerManager.setInterval(() => {}, 1000, '详细信息测试1');
      const timer2 = TimerManager.setTimeout(() => {}, 2000, '详细信息测试2');

      const allTimers = TimerManager.getAllTimers();
      Assert.arrayLength(allTimers, 2);

      const timer1Info = allTimers.find(t => t.description === '详细信息测试1');
      const timer2Info = allTimers.find(t => t.description === '详细信息测试2');

      Assert.isNotNull(timer1Info);
      Assert.isNotNull(timer2Info);
      Assert.equals(timer1Info!.type, 'interval');
      Assert.equals(timer2Info!.type, 'timeout');
      Assert.equals(timer1Info!.delay, 1000);
      Assert.equals(timer2Info!.delay, 2000);

      // 清理
      TimerManager.clearInterval(timer1);
      TimerManager.clearTimeout(timer2);
    }
  ));

  // 测试用例：长时间运行的定时器检查
  suite.tests.push(createTestCase(
    '应该能够检测长时间运行的定时器',
    async () => {
      const timer1 = TimerManager.setInterval(() => {}, 1000, '长时间运行测试');
      
      // 等待一小段时间
      await new Promise(resolve => setTimeout(resolve, 50));
      
      // 检查长时间运行的定时器（阈值设为10ms，确保能检测到）
      const longRunningTimers = TimerManager.checkLongRunningTimers(10);
      Assert.arrayLength(longRunningTimers, 1);
      Assert.equals(longRunningTimers[0].description, '长时间运行测试');

      // 清理
      TimerManager.clearInterval(timer1);
    },
    { timeout: 1000 }
  ));

  // 测试用例：健康检查
  suite.tests.push(createTestCase(
    '应该能够启动和停止健康检查',
    async () => {
      const healthCheckId = TimerManager.startHealthCheck(100); // 100ms间隔
      Assert.isType(healthCheckId, 'number');

      // 等待健康检查执行
      await new Promise(resolve => setTimeout(resolve, 150));

      // 停止健康检查
      TimerManager.clearInterval(healthCheckId);

      const stats = TimerManager.getStats();
      Assert.equals(stats.totalTimers, 0);
    },
    { timeout: 1000 }
  ));

  // 测试用例：定时器执行计数
  suite.tests.push(createTestCase(
    '应该能够正确计算定时器执行次数',
    async () => {
      let executionCount = 0;
      const timerId = TimerManager.setInterval(() => {
        executionCount++;
      }, 50, '执行计数测试');

      // 等待定时器执行几次
      await new Promise(resolve => setTimeout(resolve, 200));

      Assert.isTrue(executionCount >= 3); // 至少执行3次

      const allTimers = TimerManager.getAllTimers();
      const timerInfo = allTimers.find(t => t.description === '执行计数测试');
      Assert.isNotNull(timerInfo);
      Assert.isTrue(timerInfo!.executionCount >= 3);

      // 清理
      TimerManager.clearInterval(timerId);
    },
    { timeout: 1000 }
  ));

  // 测试用例：错误处理
  suite.tests.push(createTestCase(
    '应该能够处理定时器回调中的错误',
    async () => {
      let errorThrown = false;
      const timerId = TimerManager.setInterval(() => {
        errorThrown = true;
        throw new Error('测试错误');
      }, 50, '错误处理测试');

      // 等待定时器执行
      await new Promise(resolve => setTimeout(resolve, 100));

      Assert.isTrue(errorThrown);
      
      // 定时器应该仍然存在（错误不应该导致定时器被清除）
      const stats = TimerManager.getStats();
      Assert.equals(stats.totalTimers, 1);

      // 清理
      TimerManager.clearInterval(timerId);
    },
    { timeout: 1000 }
  ));

  return suite;
}

/**
 * 运行 TimerManager 测试
 */
export async function runTimerManagerTests(): Promise<void> {
  const { testRunner } = await import('../framework/TestFramework');
  const suite = createTimerManagerTestSuite();
  testRunner.addSuite(suite);
}
