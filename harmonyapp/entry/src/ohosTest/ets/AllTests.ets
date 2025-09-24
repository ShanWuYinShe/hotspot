/**
 * 统一测试套件入口
 * 整合所有测试类型：单元测试、集成测试、性能测试
 */

// Hypium 官方测试
import abilityTest from './test/Ability.test';
import localUnitTest from './integration/LocalUnitTest';
import networkConnectionTest from './integration/NetworkConnectionTest';

// 自定义测试框架
import { runLogManagerTests } from './unit/LogManagerTest';
import { runTimerManagerTests } from './unit/TimerManagerTest';
import { runErrorHandlerTests } from './unit/ErrorHandlerTest';
import { runEventBusTests } from './unit/EventBusTest';
import { NotificationDataManagerTest } from './unit/NotificationDataManagerTest';
import { NotificationSchedulerTest } from './unit/NotificationSchedulerTest';
import { NotificationRendererTest } from './unit/NotificationRendererTest';
import { SmartNotificationServiceV2Test } from './integration/SmartNotificationServiceV2Test';
import { runPerformanceBenchmarks } from './performance/PerformanceBenchmark';

// 测试运行器
import { MainTestRunner, TestConfig } from './TestRunner';
import { common } from '@kit.AbilityKit';
import { LogLevel } from '../../main/ets/common/utils/LogManager';

/**
 * 运行所有 Hypium 测试
 */
export function runHypiumTests() {
  console.info('=== 开始运行 Hypium 测试套件 ===');
  
  // 运行官方测试框架的测试
  abilityTest();
  localUnitTest();
  networkConnectionTest();
  
  console.info('=== Hypium 测试套件完成 ===');
}

/**
 * 运行自定义框架测试
 */
export async function runCustomTests(context: common.Context) {
  console.info('=== 开始运行自定义测试框架 ===');
  
  const testConfig: Partial<TestConfig> = {
    enableLogManagerTests: true,
    enableTimerManagerTests: true,
    enableErrorHandlerTests: true,
    enableEventBusTests: true,
    logLevel: LogLevel.INFO,
    timeout: 30000
  };
  
  const runner = new MainTestRunner(testConfig);
  
  try {
    await runner.initialize(context);
    
    // 注册所有测试
    await runLogManagerTests();
    await runTimerManagerTests();
    await runErrorHandlerTests();
    await runEventBusTests();
    
    // 运行通知服务测试
    NotificationDataManagerTest();
    NotificationSchedulerTest();
    NotificationRendererTest();
    SmartNotificationServiceV2Test();
    
    // 运行测试
    const report = await runner.runAllTests();
    
    console.info('=== 自定义测试框架完成 ===');
    return report;
    
  } finally {
    await runner.cleanup();
  }
}

/**
 * 运行性能基准测试
 */
export async function runPerformanceTests() {
  console.info('=== 开始运行性能基准测试 ===');
  
  try {
    const results = await runPerformanceBenchmarks();
    console.info('=== 性能基准测试完成 ===');
    return results;
  } catch (error) {
    console.error('性能测试失败:', error);
    throw error;
  }
}

/**
 * 运行所有测试
 */
export async function runAllTests(context: common.Context) {
  console.info('🚀 开始运行完整测试套件');
  
  const results = {
    hypiumTests: null as any,
    customTests: null as any,
    performanceTests: null as any,
    success: false,
    totalTime: 0
  };
  
  const startTime = Date.now();
  
  try {
    // 1. 运行 Hypium 测试
    console.info('\n📋 第一阶段：Hypium 官方测试');
    results.hypiumTests = runHypiumTests();
    
    // 2. 运行自定义测试
    console.info('\n🧪 第二阶段：自定义单元测试');
    results.customTests = await runCustomTests(context);
    
    // 3. 运行性能测试
    console.info('\n⚡ 第三阶段：性能基准测试');
    results.performanceTests = await runPerformanceTests();
    
    results.success = true;
    results.totalTime = Date.now() - startTime;
    
    // 输出总结
    console.info('\n🎉 === 测试套件执行完成 ===');
    console.info(`总耗时: ${results.totalTime}ms`);
    console.info('✅ 所有测试阶段执行完成');
    
    if (results.customTests) {
      const { totalTests, passedTests, failedTests, errorTests } = results.customTests;
      console.info(`自定义测试结果: ${passedTests}/${totalTests} 通过`);
      if (failedTests > 0 || errorTests > 0) {
        console.warn(`⚠️ 有 ${failedTests + errorTests} 个测试未通过`);
      }
    }
    
    return results;
    
  } catch (error) {
    results.success = false;
    results.totalTime = Date.now() - startTime;
    
    console.error('❌ 测试套件执行失败:', error);
    console.info(`执行时间: ${results.totalTime}ms`);
    
    throw error;
  }
}

/**
 * 快速测试（仅核心功能）
 */
export async function runQuickTests(context: common.Context) {
  console.info('⚡ 开始快速测试');
  
  const testConfig: Partial<TestConfig> = {
    enableLogManagerTests: true,
    enableTimerManagerTests: false, // 跳过耗时的定时器测试
    enableErrorHandlerTests: true,
    enableEventBusTests: false, // 跳过复杂的事件测试
    logLevel: LogLevel.WARN, // 减少日志输出
    timeout: 10000 // 10秒快速测试
  };
  
  const runner = new MainTestRunner(testConfig);
  
  try {
    await runner.initialize(context);
    
    // 只运行核心测试
    await runLogManagerTests();
    await runErrorHandlerTests();
    
    const report = await runner.runAllTests();
    
    const success = report.failedTests === 0 && report.errorTests === 0;
    console.info(`⚡ 快速测试完成: ${success ? '✅ 通过' : '❌ 失败'}`);
    
    return success;
    
  } finally {
    await runner.cleanup();
  }
}

/**
 * 默认导出：运行所有测试
 */
export default function testsuite() {
  console.info('📋 运行基础 Hypium 测试套件');
  runHypiumTests();
}
