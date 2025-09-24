/**
 * 性能基准测试
 * 测试核心服务的性能表现
 */

import { createLogger } from '../../../main/ets/common/utils/LogManager';
import { TimerManager } from '../../../main/ets/common/utils/TimerManager';
import { ErrorHandler, ErrorType, ErrorLevel } from '../../../main/ets/common/utils/ErrorHandler';
import { EventBus } from '../../../main/ets/common/events/EventBus';
import { Container, ServiceLifetime } from '../../../main/ets/common/di/Container';

const logger = createLogger('PerformanceBenchmark');

/**
 * 基准测试结果接口
 */
export interface BenchmarkResult {
  name: string;
  iterations: number;
  totalTime: number;
  averageTime: number;
  minTime: number;
  maxTime: number;
  throughput: number; // 每秒操作数
  memoryUsage?: number;
}

/**
 * 基准测试套件结果
 */
export interface BenchmarkSuiteResult {
  suiteName: string;
  results: BenchmarkResult[];
  totalTime: number;
  timestamp: number;
}

/**
 * 性能基准测试类
 */
export class PerformanceBenchmark {
  private results: BenchmarkSuiteResult[] = [];

  /**
   * 运行单个基准测试
   */
  async runBenchmark(
    name: string,
    testFunction: () => void | Promise<void>,
    iterations: number = 1000
  ): Promise<BenchmarkResult> {
    logger.info(`开始基准测试: ${name} (${iterations} 次迭代)`);

    const times: number[] = [];
    let totalTime = 0;

    // 预热
    for (let i = 0; i < Math.min(100, iterations / 10); i++) {
      await testFunction();
    }

    // 正式测试
    for (let i = 0; i < iterations; i++) {
      const startTime = performance.now();
      await testFunction();
      const endTime = performance.now();
      
      const duration = endTime - startTime;
      times.push(duration);
      totalTime += duration;
    }

    const averageTime = totalTime / iterations;
    const minTime = Math.min(...times);
    const maxTime = Math.max(...times);
    const throughput = 1000 / averageTime; // 每秒操作数

    const result: BenchmarkResult = {
      name,
      iterations,
      totalTime,
      averageTime,
      minTime,
      maxTime,
      throughput
    };

    logger.info(`${name} 完成: 平均 ${averageTime.toFixed(3)}ms, 吞吐量 ${throughput.toFixed(0)} ops/sec`);
    return result;
  }

  /**
   * 运行 LogManager 基准测试
   */
  async runLogManagerBenchmarks(): Promise<BenchmarkResult[]> {
    const results: BenchmarkResult[] = [];

    // 测试日志记录性能
    results.push(await this.runBenchmark(
      'LogManager - 简单日志记录',
      () => {
        logger.info('这是一条测试日志');
      },
      10000
    ));

    // 测试带参数的日志记录
    results.push(await this.runBenchmark(
      'LogManager - 带参数日志记录',
      () => {
        logger.info('用户操作', { userId: '123', action: 'login', timestamp: Date.now() });
      },
      5000
    ));

    // 测试敏感数据脱敏性能
    results.push(await this.runBenchmark(
      'LogManager - 敏感数据脱敏',
      () => {
        const sensitiveData = {
          username: 'testuser',
          password: 'secretpassword123',
          token: 'abc123def456ghi789',
          data: { nested: { password: 'nested_secret' } }
        };
        logger.info('敏感数据处理', sensitiveData);
      },
      1000
    ));

    return results;
  }

  /**
   * 运行 TimerManager 基准测试
   */
  async runTimerManagerBenchmarks(): Promise<BenchmarkResult[]> {
    const results: BenchmarkResult[] = [];

    // 测试定时器创建和清除性能
    results.push(await this.runBenchmark(
      'TimerManager - 定时器创建和清除',
      () => {
        const timerId = TimerManager.setInterval(() => {}, 1000, '基准测试');
        TimerManager.clearInterval(timerId);
      },
      5000
    ));

    // 测试大量定时器管理
    results.push(await this.runBenchmark(
      'TimerManager - 批量定时器管理',
      () => {
        const timerIds: number[] = [];
        // 创建10个定时器
        for (let i = 0; i < 10; i++) {
          timerIds.push(TimerManager.setInterval(() => {}, 1000, `批量测试${i}`));
        }
        // 清除所有定时器
        timerIds.forEach(id => TimerManager.clearInterval(id));
      },
      1000
    ));

    // 测试统计信息获取性能
    results.push(await this.runBenchmark(
      'TimerManager - 统计信息获取',
      () => {
        TimerManager.getStats();
      },
      10000
    ));

    return results;
  }

  /**
   * 运行 ErrorHandler 基准测试
   */
  async runErrorHandlerBenchmarks(): Promise<BenchmarkResult[]> {
    const results: BenchmarkResult[] = [];

    // 测试错误处理性能
    results.push(await this.runBenchmark(
      'ErrorHandler - 基本错误处理',
      () => {
        ErrorHandler.handle(
          new Error('基准测试错误'),
          'BenchmarkTest',
          ErrorType.SYSTEM,
          ErrorLevel.LOW
        );
      },
      5000
    ));

    // 测试安全执行性能
    results.push(await this.runBenchmark(
      'ErrorHandler - 安全执行（成功）',
      async () => {
        await ErrorHandler.safeExecute(
          async () => 'success',
          'BenchmarkSafeExecute'
        );
      },
      3000
    ));

    // 测试安全执行性能（失败情况）
    results.push(await this.runBenchmark(
      'ErrorHandler - 安全执行（失败）',
      async () => {
        await ErrorHandler.safeExecute(
          async () => { throw new Error('测试失败'); },
          'BenchmarkSafeExecuteFail',
          'default'
        );
      },
      1000
    ));

    return results;
  }

  /**
   * 运行 EventBus 基准测试
   */
  async runEventBusBenchmarks(): Promise<BenchmarkResult[]> {
    const results: BenchmarkResult[] = [];
    const eventBus = EventBus.getInstance();

    // 设置测试监听器
    let eventCount = 0;
    eventBus.on('benchmark:test', () => { eventCount++; });

    // 测试事件发布性能
    results.push(await this.runBenchmark(
      'EventBus - 事件发布',
      () => {
        eventBus.emit('benchmark:test', { data: 'test' });
      },
      10000
    ));

    // 测试多监听器事件发布
    const listeners: string[] = [];
    for (let i = 0; i < 10; i++) {
      listeners.push(eventBus.on('benchmark:multi', () => {}));
    }

    results.push(await this.runBenchmark(
      'EventBus - 多监听器事件发布',
      () => {
        eventBus.emit('benchmark:multi', { data: 'multi test' });
      },
      5000
    ));

    // 清理监听器
    listeners.forEach(id => eventBus.off('benchmark:multi', id));

    // 测试监听器添加和移除性能
    results.push(await this.runBenchmark(
      'EventBus - 监听器添加和移除',
      () => {
        const listenerId = eventBus.on('benchmark:temp', () => {});
        eventBus.off('benchmark:temp', listenerId);
      },
      5000
    ));

    return results;
  }

  /**
   * 运行 Container 基准测试
   */
  async runContainerBenchmarks(): Promise<BenchmarkResult[]> {
    const results: BenchmarkResult[] = [];
    const container = Container.getInstance();

    // 注册测试服务
    class TestService {
      getValue(): string { return 'test'; }
    }

    container.registerType('TestService', TestService, ServiceLifetime.TRANSIENT);
    container.registerType('SingletonService', TestService, ServiceLifetime.SINGLETON);

    // 测试服务解析性能（瞬态）
    results.push(await this.runBenchmark(
      'Container - 瞬态服务解析',
      () => {
        container.resolve<TestService>('TestService');
      },
      5000
    ));

    // 测试服务解析性能（单例）
    results.push(await this.runBenchmark(
      'Container - 单例服务解析',
      () => {
        container.resolve<TestService>('SingletonService');
      },
      10000
    ));

    // 测试服务注册性能
    results.push(await this.runBenchmark(
      'Container - 服务注册',
      () => {
        const serviceName = `TempService_${Math.random()}`;
        container.registerType(serviceName, TestService, ServiceLifetime.TRANSIENT);
        container.unregister(serviceName);
      },
      1000
    ));

    return results;
  }

  /**
   * 运行所有基准测试
   */
  async runAllBenchmarks(): Promise<BenchmarkSuiteResult[]> {
    const suiteResults: BenchmarkSuiteResult[] = [];

    logger.info('开始运行所有性能基准测试');

    // LogManager 基准测试
    const logManagerStart = performance.now();
    const logManagerResults = await this.runLogManagerBenchmarks();
    const logManagerTime = performance.now() - logManagerStart;
    
    suiteResults.push({
      suiteName: 'LogManager',
      results: logManagerResults,
      totalTime: logManagerTime,
      timestamp: Date.now()
    });

    // TimerManager 基准测试
    const timerManagerStart = performance.now();
    const timerManagerResults = await this.runTimerManagerBenchmarks();
    const timerManagerTime = performance.now() - timerManagerStart;
    
    suiteResults.push({
      suiteName: 'TimerManager',
      results: timerManagerResults,
      totalTime: timerManagerTime,
      timestamp: Date.now()
    });

    // ErrorHandler 基准测试
    const errorHandlerStart = performance.now();
    const errorHandlerResults = await this.runErrorHandlerBenchmarks();
    const errorHandlerTime = performance.now() - errorHandlerStart;
    
    suiteResults.push({
      suiteName: 'ErrorHandler',
      results: errorHandlerResults,
      totalTime: errorHandlerTime,
      timestamp: Date.now()
    });

    // EventBus 基准测试
    const eventBusStart = performance.now();
    const eventBusResults = await this.runEventBusBenchmarks();
    const eventBusTime = performance.now() - eventBusStart;
    
    suiteResults.push({
      suiteName: 'EventBus',
      results: eventBusResults,
      totalTime: eventBusTime,
      timestamp: Date.now()
    });

    // Container 基准测试
    const containerStart = performance.now();
    const containerResults = await this.runContainerBenchmarks();
    const containerTime = performance.now() - containerStart;
    
    suiteResults.push({
      suiteName: 'Container',
      results: containerResults,
      totalTime: containerTime,
      timestamp: Date.now()
    });

    this.results = suiteResults;
    this.generateReport();

    logger.info('所有性能基准测试完成');
    return suiteResults;
  }

  /**
   * 生成性能报告
   */
  private generateReport(): void {
    logger.info('=== 性能基准测试报告 ===');

    this.results.forEach(suiteResult => {
      logger.info(`\n--- ${suiteResult.suiteName} 性能报告 ---`);
      logger.info(`套件总耗时: ${suiteResult.totalTime.toFixed(2)}ms`);

      suiteResult.results.forEach(result => {
        logger.info(`\n${result.name}:`);
        logger.info(`  迭代次数: ${result.iterations}`);
        logger.info(`  总耗时: ${result.totalTime.toFixed(2)}ms`);
        logger.info(`  平均耗时: ${result.averageTime.toFixed(3)}ms`);
        logger.info(`  最小耗时: ${result.minTime.toFixed(3)}ms`);
        logger.info(`  最大耗时: ${result.maxTime.toFixed(3)}ms`);
        logger.info(`  吞吐量: ${result.throughput.toFixed(0)} ops/sec`);
      });
    });

    // 性能总结
    logger.info('\n=== 性能总结 ===');
    const allResults = this.results.flatMap(suite => suite.results);
    const fastestTest = allResults.reduce((fastest, current) => 
      current.averageTime < fastest.averageTime ? current : fastest
    );
    const slowestTest = allResults.reduce((slowest, current) => 
      current.averageTime > slowest.averageTime ? current : slowest
    );

    logger.info(`最快操作: ${fastestTest.name} (${fastestTest.averageTime.toFixed(3)}ms)`);
    logger.info(`最慢操作: ${slowestTest.name} (${slowestTest.averageTime.toFixed(3)}ms)`);

    // 性能建议
    this.generatePerformanceRecommendations();

    logger.info('=== 性能报告结束 ===');
  }

  /**
   * 生成性能建议
   */
  private generatePerformanceRecommendations(): void {
    logger.info('\n=== 性能建议 ===');

    this.results.forEach(suiteResult => {
      suiteResult.results.forEach(result => {
        if (result.averageTime > 10) { // 超过10ms的操作
          logger.warn(`⚠️  ${result.name} 平均耗时较长 (${result.averageTime.toFixed(3)}ms)，建议优化`);
        }
        
        if (result.throughput < 100) { // 吞吐量低于100 ops/sec
          logger.warn(`⚠️  ${result.name} 吞吐量较低 (${result.throughput.toFixed(0)} ops/sec)，建议优化`);
        }

        if (result.maxTime > result.averageTime * 10) { // 最大耗时是平均耗时的10倍以上
          logger.warn(`⚠️  ${result.name} 性能不稳定，最大耗时过高`);
        }
      });
    });
  }

  /**
   * 获取测试结果
   */
  getResults(): BenchmarkSuiteResult[] {
    return [...this.results];
  }

  /**
   * 清理测试数据
   */
  cleanup(): void {
    this.results = [];
    // 清理错误历史，避免影响后续测试
    ErrorHandler.clearHistory();
  }
}

/**
 * 便捷函数：运行性能基准测试
 */
export async function runPerformanceBenchmarks(): Promise<BenchmarkSuiteResult[]> {
  const benchmark = new PerformanceBenchmark();
  try {
    return await benchmark.runAllBenchmarks();
  } finally {
    benchmark.cleanup();
  }
}

/**
 * 便捷函数：运行特定服务的基准测试
 */
export async function runSpecificBenchmark(serviceName: string): Promise<BenchmarkResult[]> {
  const benchmark = new PerformanceBenchmark();
  
  try {
    switch (serviceName.toLowerCase()) {
      case 'logmanager':
        return await benchmark.runLogManagerBenchmarks();
      case 'timermanager':
        return await benchmark.runTimerManagerBenchmarks();
      case 'errorhandler':
        return await benchmark.runErrorHandlerBenchmarks();
      case 'eventbus':
        return await benchmark.runEventBusBenchmarks();
      case 'container':
        return await benchmark.runContainerBenchmarks();
      default:
        throw new Error(`未知的服务名称: ${serviceName}`);
    }
  } finally {
    benchmark.cleanup();
  }
}
