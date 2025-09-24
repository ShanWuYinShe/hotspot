/**
 * 测试运行器入口
 * 统一管理和运行所有单元测试
 */

import { common } from '@kit.AbilityKit';
import { TestRunner, TestReport, createLogger } from './framework/TestFramework';
import { runLogManagerTests } from './unit/LogManagerTest';
import { runTimerManagerTests } from './unit/TimerManagerTest';
import { runErrorHandlerTests } from './unit/ErrorHandlerTest';
import { runEventBusTests } from './unit/EventBusTest';
import { ApplicationManager } from '../../main/ets/common/core/ApplicationManager';
import { LogLevel } from '../../main/ets/common/utils/LogManager';

const logger = createLogger('TestRunner');

/**
 * 测试配置接口
 */
export interface TestConfig {
  enableLogManagerTests: boolean;
  enableTimerManagerTests: boolean;
  enableErrorHandlerTests: boolean;
  enableEventBusTests: boolean;
  enableIntegrationTests: boolean;
  logLevel: LogLevel;
  timeout: number;
}

/**
 * 默认测试配置
 */
const DEFAULT_TEST_CONFIG: TestConfig = {
  enableLogManagerTests: true,
  enableTimerManagerTests: true,
  enableErrorHandlerTests: true,
  enableEventBusTests: true,
  enableIntegrationTests: true,
  logLevel: LogLevel.INFO,
  timeout: 30000 // 30秒总超时
};

/**
 * 主测试运行器类
 */
export class MainTestRunner {
  private testRunner: TestRunner;
  private config: TestConfig;
  private appManager: ApplicationManager | null = null;

  constructor(config: Partial<TestConfig> = {}) {
    this.config = { ...DEFAULT_TEST_CONFIG, ...config };
    this.testRunner = new TestRunner();
  }

  /**
   * 初始化测试环境
   */
  async initialize(context: common.Context): Promise<void> {
    try {
      logger.info('初始化测试环境');

      // 初始化应用管理器
      this.appManager = ApplicationManager.getInstance();
      await this.appManager.initializeWithContext(context, {
        logLevel: this.config.logLevel,
        enablePerformanceMonitoring: false, // 测试时关闭性能监控
        enableErrorReporting: true,
        enableTimerHealthCheck: false, // 测试时关闭定时器健康检查
        maxErrorHistory: 1000,
        autoRecovery: false
      });

      logger.info('测试环境初始化完成');

    } catch (error) {
      logger.error('测试环境初始化失败:', error);
      throw error;
    }
  }

  /**
   * 运行所有测试
   */
  async runAllTests(): Promise<TestReport> {
    try {
      logger.info('开始运行所有测试');

      // 注册测试套件
      await this.registerTestSuites();

      // 运行测试
      const report = await this.testRunner.runAll();

      // 输出测试结果
      this.outputTestResults(report);

      return report;

    } catch (error) {
      logger.error('运行测试时出错:', error);
      throw error;
    }
  }

  /**
   * 运行特定的测试套件
   */
  async runSpecificTests(testNames: string[]): Promise<TestReport> {
    try {
      logger.info(`运行指定测试: ${testNames.join(', ')}`);

      // 只注册指定的测试
      await this.registerSpecificTestSuites(testNames);

      const report = await this.testRunner.runAll();
      this.outputTestResults(report);

      return report;

    } catch (error) {
      logger.error('运行指定测试时出错:', error);
      throw error;
    }
  }

  /**
   * 清理测试环境
   */
  async cleanup(): Promise<void> {
    try {
      logger.info('清理测试环境');

      // 清理测试运行器
      this.testRunner.cleanup();

      // 清理应用管理器
      if (this.appManager) {
        await this.appManager.cleanup();
      }

      logger.info('测试环境清理完成');

    } catch (error) {
      logger.error('清理测试环境时出错:', error);
    }
  }

  /**
   * 注册所有测试套件
   */
  private async registerTestSuites(): Promise<void> {
    if (this.config.enableLogManagerTests) {
      await runLogManagerTests();
      logger.debug('LogManager 测试已注册');
    }

    if (this.config.enableTimerManagerTests) {
      await runTimerManagerTests();
      logger.debug('TimerManager 测试已注册');
    }

    if (this.config.enableErrorHandlerTests) {
      await runErrorHandlerTests();
      logger.debug('ErrorHandler 测试已注册');
    }

    if (this.config.enableEventBusTests) {
      await runEventBusTests();
      logger.debug('EventBus 测试已注册');
    }

    logger.info('所有测试套件已注册');
  }

  /**
   * 注册特定的测试套件
   */
  private async registerSpecificTestSuites(testNames: string[]): Promise<void> {
    for (const testName of testNames) {
      switch (testName.toLowerCase()) {
        case 'logmanager':
          if (this.config.enableLogManagerTests) {
            await runLogManagerTests();
            logger.debug('LogManager 测试已注册');
          }
          break;
        case 'timermanager':
          if (this.config.enableTimerManagerTests) {
            await runTimerManagerTests();
            logger.debug('TimerManager 测试已注册');
          }
          break;
        case 'errorhandler':
          if (this.config.enableErrorHandlerTests) {
            await runErrorHandlerTests();
            logger.debug('ErrorHandler 测试已注册');
          }
          break;
        case 'eventbus':
          if (this.config.enableEventBusTests) {
            await runEventBusTests();
            logger.debug('EventBus 测试已注册');
          }
          break;
        default:
          logger.warn(`未知的测试套件: ${testName}`);
      }
    }
  }

  /**
   * 输出测试结果
   */
  private outputTestResults(report: TestReport): void {
    logger.info('=== 详细测试报告 ===');
    
    // 总体统计
    logger.info(`测试套件总数: ${report.totalSuites}`);
    logger.info(`测试用例总数: ${report.totalTests}`);
    logger.info(`通过: ${report.passedTests} (${this.getPercentage(report.passedTests, report.totalTests)}%)`);
    logger.info(`失败: ${report.failedTests} (${this.getPercentage(report.failedTests, report.totalTests)}%)`);
    logger.info(`跳过: ${report.skippedTests} (${this.getPercentage(report.skippedTests, report.totalTests)}%)`);
    logger.info(`错误: ${report.errorTests} (${this.getPercentage(report.errorTests, report.totalTests)}%)`);
    logger.info(`总耗时: ${report.duration}ms`);

    // 各套件详情
    report.suiteResults.forEach(suiteResult => {
      logger.info(`\n--- ${suiteResult.name} ---`);
      logger.info(`测试数量: ${suiteResult.totalTests}`);
      logger.info(`通过: ${suiteResult.passedTests}, 失败: ${suiteResult.failedTests}, 跳过: ${suiteResult.skippedTests}, 错误: ${suiteResult.errorTests}`);
      logger.info(`耗时: ${suiteResult.duration}ms`);

      // 显示失败的测试
      const failedTests = suiteResult.results.filter(r => 
        r.result === 'FAILED' || r.result === 'ERROR'
      );

      if (failedTests.length > 0) {
        logger.error(`失败的测试:`);
        failedTests.forEach(test => {
          logger.error(`  ❌ ${test.name}: ${test.message || test.error?.message}`);
          if (test.error?.stack) {
            logger.debug(`     堆栈: ${test.error.stack}`);
          }
        });
      }

      // 显示通过的测试
      const passedTests = suiteResult.results.filter(r => r.result === 'PASSED');
      if (passedTests.length > 0) {
        logger.info(`通过的测试:`);
        passedTests.forEach(test => {
          logger.info(`  ✅ ${test.name} (${test.duration}ms)`);
        });
      }
    });

    // 测试结果总结
    const success = report.failedTests === 0 && report.errorTests === 0;
    if (success) {
      logger.info('\n🎉 所有测试通过！');
    } else {
      logger.error(`\n❌ 测试失败：${report.failedTests + report.errorTests} 个测试未通过`);
    }

    logger.info('=== 测试报告结束 ===');
  }

  /**
   * 计算百分比
   */
  private getPercentage(part: number, total: number): string {
    if (total === 0) return '0';
    return ((part / total) * 100).toFixed(1);
  }

  /**
   * 获取测试配置
   */
  getConfig(): TestConfig {
    return { ...this.config };
  }

  /**
   * 更新测试配置
   */
  updateConfig(config: Partial<TestConfig>): void {
    this.config = { ...this.config, ...config };
  }

  /**
   * 获取测试运行器实例
   */
  getTestRunner(): TestRunner {
    return this.testRunner;
  }
}

/**
 * 便捷函数：运行所有测试
 */
export async function runAllTests(
  context: common.Context, 
  config?: Partial<TestConfig>
): Promise<TestReport> {
  const runner = new MainTestRunner(config);
  
  try {
    await runner.initialize(context);
    const report = await runner.runAllTests();
    return report;
  } finally {
    await runner.cleanup();
  }
}

/**
 * 便捷函数：运行特定测试
 */
export async function runSpecificTests(
  context: common.Context,
  testNames: string[],
  config?: Partial<TestConfig>
): Promise<TestReport> {
  const runner = new MainTestRunner(config);
  
  try {
    await runner.initialize(context);
    const report = await runner.runSpecificTests(testNames);
    return report;
  } finally {
    await runner.cleanup();
  }
}

/**
 * 便捷函数：快速测试（用于开发调试）
 */
export async function quickTest(context: common.Context): Promise<boolean> {
  try {
    const report = await runAllTests(context, {
      logLevel: LogLevel.WARN, // 减少日志输出
      timeout: 10000 // 10秒快速测试
    });
    
    return report.failedTests === 0 && report.errorTests === 0;
  } catch (error) {
    logger.error('快速测试失败:', error);
    return false;
  }
}
