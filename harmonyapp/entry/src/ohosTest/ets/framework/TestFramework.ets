/**
 * 轻量级测试框架
 * 专为HarmonyOS应用设计的单元测试框架
 */

import { createLogger } from '../../../main/ets/common/utils/LogManager';
import { ErrorHandler, ErrorType, ErrorLevel } from '../../../main/ets/common/utils/ErrorHandler';
import { TimerManager } from '../../../main/ets/common/utils/TimerManager';

const logger = createLogger('TestFramework');

/**
 * 测试结果枚举
 */
export enum TestResult {
  PASSED = 'PASSED',
  FAILED = 'FAILED',
  SKIPPED = 'SKIPPED',
  ERROR = 'ERROR'
}

/**
 * 测试用例接口
 */
export interface TestCase {
  name: string;
  description?: string;
  timeout?: number;
  skip?: boolean;
  setup?: () => void | Promise<void>;
  teardown?: () => void | Promise<void>;
  test: () => void | Promise<void>;
}

/**
 * 测试套件接口
 */
export interface TestSuite {
  name: string;
  description?: string;
  setup?: () => void | Promise<void>;
  teardown?: () => void | Promise<void>;
  tests: TestCase[];
}

/**
 * 测试执行结果
 */
export interface TestExecutionResult {
  name: string;
  result: TestResult;
  duration: number;
  error?: Error;
  message?: string;
}

/**
 * 测试套件执行结果
 */
export interface TestSuiteResult {
  name: string;
  totalTests: number;
  passedTests: number;
  failedTests: number;
  skippedTests: number;
  errorTests: number;
  duration: number;
  results: TestExecutionResult[];
}

/**
 * 测试报告
 */
export interface TestReport {
  totalSuites: number;
  totalTests: number;
  passedTests: number;
  failedTests: number;
  skippedTests: number;
  errorTests: number;
  duration: number;
  suiteResults: TestSuiteResult[];
  startTime: number;
  endTime: number;
}

/**
 * 断言类
 */
export class Assert {
  /**
   * 断言为真
   */
  static isTrue(value: boolean, message?: string): void {
    if (!value) {
      throw new Error(message || `期望值为 true，但实际为 ${value}`);
    }
  }

  /**
   * 断言为假
   */
  static isFalse(value: boolean, message?: string): void {
    if (value) {
      throw new Error(message || `期望值为 false，但实际为 ${value}`);
    }
  }

  /**
   * 断言相等
   */
  static equals<T>(actual: T, expected: T, message?: string): void {
    if (actual !== expected) {
      throw new Error(message || `期望值为 ${expected}，但实际为 ${actual}`);
    }
  }

  /**
   * 断言不相等
   */
  static notEquals<T>(actual: T, expected: T, message?: string): void {
    if (actual === expected) {
      throw new Error(message || `期望值不为 ${expected}，但实际为 ${actual}`);
    }
  }

  /**
   * 断言深度相等（对象比较）
   */
  static deepEquals(actual: any, expected: any, message?: string): void {
    const actualStr = JSON.stringify(actual);
    const expectedStr = JSON.stringify(expected);
    if (actualStr !== expectedStr) {
      throw new Error(message || `期望对象为 ${expectedStr}，但实际为 ${actualStr}`);
    }
  }

  /**
   * 断言为null
   */
  static isNull(value: any, message?: string): void {
    if (value !== null) {
      throw new Error(message || `期望值为 null，但实际为 ${value}`);
    }
  }

  /**
   * 断言不为null
   */
  static isNotNull(value: any, message?: string): void {
    if (value === null) {
      throw new Error(message || '期望值不为 null，但实际为 null');
    }
  }

  /**
   * 断言为undefined
   */
  static isUndefined(value: any, message?: string): void {
    if (value !== undefined) {
      throw new Error(message || `期望值为 undefined，但实际为 ${value}`);
    }
  }

  /**
   * 断言不为undefined
   */
  static isNotUndefined(value: any, message?: string): void {
    if (value === undefined) {
      throw new Error(message || '期望值不为 undefined，但实际为 undefined');
    }
  }

  /**
   * 断言数组包含元素
   */
  static contains<T>(array: T[], element: T, message?: string): void {
    if (!array.includes(element)) {
      throw new Error(message || `期望数组包含 ${element}，但实际不包含`);
    }
  }

  /**
   * 断言数组不包含元素
   */
  static notContains<T>(array: T[], element: T, message?: string): void {
    if (array.includes(element)) {
      throw new Error(message || `期望数组不包含 ${element}，但实际包含`);
    }
  }

  /**
   * 断言抛出异常
   */
  static async throws(fn: () => void | Promise<void>, message?: string): Promise<void> {
    let threw = false;
    try {
      await fn();
    } catch (error) {
      threw = true;
    }
    if (!threw) {
      throw new Error(message || '期望抛出异常，但没有抛出');
    }
  }

  /**
   * 断言不抛出异常
   */
  static async notThrows(fn: () => void | Promise<void>, message?: string): Promise<void> {
    try {
      await fn();
    } catch (error) {
      throw new Error(message || `期望不抛出异常，但抛出了: ${error}`);
    }
  }

  /**
   * 断言字符串包含子串
   */
  static stringContains(str: string, substring: string, message?: string): void {
    if (!str.includes(substring)) {
      throw new Error(message || `期望字符串包含 "${substring}"，但实际不包含`);
    }
  }

  /**
   * 断言数组长度
   */
  static arrayLength<T>(array: T[], expectedLength: number, message?: string): void {
    if (array.length !== expectedLength) {
      throw new Error(message || `期望数组长度为 ${expectedLength}，但实际为 ${array.length}`);
    }
  }

  /**
   * 断言类型
   */
  static isType(value: any, expectedType: string, message?: string): void {
    const actualType = typeof value;
    if (actualType !== expectedType) {
      throw new Error(message || `期望类型为 ${expectedType}，但实际为 ${actualType}`);
    }
  }

  /**
   * 断言实例类型
   */
  static isInstanceOf(value: any, expectedClass: any, message?: string): void {
    if (!(value instanceof expectedClass)) {
      throw new Error(message || `期望为 ${expectedClass.name} 的实例，但不是`);
    }
  }
}

/**
 * 模拟对象工具
 */
export class Mock {
  private calls: any[][] = [];
  private returnValues: any[] = [];
  private throwErrors: Error[] = [];

  /**
   * 设置返回值
   */
  returns(value: any): Mock {
    this.returnValues.push(value);
    return this;
  }

  /**
   * 设置抛出错误
   */
  throws(error: Error): Mock {
    this.throwErrors.push(error);
    return this;
  }

  /**
   * 创建模拟函数
   */
  createFunction(): (...args: any[]) => any {
    return (...args: any[]) => {
      this.calls.push(args);
      
      if (this.throwErrors.length > 0) {
        throw this.throwErrors.shift();
      }
      
      if (this.returnValues.length > 0) {
        return this.returnValues.shift();
      }
      
      return undefined;
    };
  }

  /**
   * 获取调用次数
   */
  getCallCount(): number {
    return this.calls.length;
  }

  /**
   * 获取调用参数
   */
  getCall(index: number): any[] {
    return this.calls[index] || [];
  }

  /**
   * 获取所有调用
   */
  getAllCalls(): any[][] {
    return [...this.calls];
  }

  /**
   * 验证是否被调用
   */
  wasCalled(): boolean {
    return this.calls.length > 0;
  }

  /**
   * 验证调用次数
   */
  wasCalledTimes(times: number): boolean {
    return this.calls.length === times;
  }

  /**
   * 验证调用参数
   */
  wasCalledWith(...args: any[]): boolean {
    return this.calls.some(call => 
      call.length === args.length && 
      call.every((arg, index) => arg === args[index])
    );
  }

  /**
   * 重置模拟对象
   */
  reset(): void {
    this.calls = [];
    this.returnValues = [];
    this.throwErrors = [];
  }
}

/**
 * 测试运行器
 */
export class TestRunner {
  private suites: TestSuite[] = [];
  private report: TestReport | null = null;

  /**
   * 添加测试套件
   */
  addSuite(suite: TestSuite): void {
    this.suites.push(suite);
  }

  /**
   * 运行所有测试
   */
  async runAll(): Promise<TestReport> {
    const startTime = Date.now();
    logger.info('开始运行测试');

    const suiteResults: TestSuiteResult[] = [];
    let totalTests = 0;
    let passedTests = 0;
    let failedTests = 0;
    let skippedTests = 0;
    let errorTests = 0;

    for (const suite of this.suites) {
      const suiteResult = await this.runSuite(suite);
      suiteResults.push(suiteResult);
      
      totalTests += suiteResult.totalTests;
      passedTests += suiteResult.passedTests;
      failedTests += suiteResult.failedTests;
      skippedTests += suiteResult.skippedTests;
      errorTests += suiteResult.errorTests;
    }

    const endTime = Date.now();
    
    this.report = {
      totalSuites: this.suites.length,
      totalTests,
      passedTests,
      failedTests,
      skippedTests,
      errorTests,
      duration: endTime - startTime,
      suiteResults,
      startTime,
      endTime
    };

    this.logReport();
    return this.report;
  }

  /**
   * 运行单个测试套件
   */
  async runSuite(suite: TestSuite): Promise<TestSuiteResult> {
    const startTime = Date.now();
    logger.info(`运行测试套件: ${suite.name}`);

    const results: TestExecutionResult[] = [];
    let passedTests = 0;
    let failedTests = 0;
    let skippedTests = 0;
    let errorTests = 0;

    try {
      // 执行套件设置
      if (suite.setup) {
        await suite.setup();
      }

      // 运行所有测试用例
      for (const testCase of suite.tests) {
        const result = await this.runTest(testCase);
        results.push(result);

        switch (result.result) {
          case TestResult.PASSED:
            passedTests++;
            break;
          case TestResult.FAILED:
            failedTests++;
            break;
          case TestResult.SKIPPED:
            skippedTests++;
            break;
          case TestResult.ERROR:
            errorTests++;
            break;
        }
      }

      // 执行套件清理
      if (suite.teardown) {
        await suite.teardown();
      }

    } catch (error) {
      logger.error(`测试套件 ${suite.name} 执行失败:`, error);
      ErrorHandler.handle(error, `TestSuite.${suite.name}`, ErrorType.SYSTEM, ErrorLevel.HIGH);
    }

    const endTime = Date.now();

    return {
      name: suite.name,
      totalTests: suite.tests.length,
      passedTests,
      failedTests,
      skippedTests,
      errorTests,
      duration: endTime - startTime,
      results
    };
  }

  /**
   * 运行单个测试用例
   */
  async runTest(testCase: TestCase): Promise<TestExecutionResult> {
    const startTime = Date.now();

    if (testCase.skip) {
      return {
        name: testCase.name,
        result: TestResult.SKIPPED,
        duration: 0,
        message: '测试被跳过'
      };
    }

    try {
      // 执行测试设置
      if (testCase.setup) {
        await testCase.setup();
      }

      // 执行测试（带超时）
      if (testCase.timeout) {
        await this.runWithTimeout(testCase.test, testCase.timeout);
      } else {
        await testCase.test();
      }

      // 执行测试清理
      if (testCase.teardown) {
        await testCase.teardown();
      }

      const endTime = Date.now();
      
      return {
        name: testCase.name,
        result: TestResult.PASSED,
        duration: endTime - startTime
      };

    } catch (error) {
      const endTime = Date.now();
      
      // 尝试执行清理（即使测试失败）
      try {
        if (testCase.teardown) {
          await testCase.teardown();
        }
      } catch (teardownError) {
        logger.warn(`测试清理失败: ${testCase.name}`, teardownError);
      }

      return {
        name: testCase.name,
        result: error instanceof Error ? TestResult.FAILED : TestResult.ERROR,
        duration: endTime - startTime,
        error: error instanceof Error ? error : new Error(String(error)),
        message: error instanceof Error ? error.message : String(error)
      };
    }
  }

  /**
   * 带超时的执行
   */
  private async runWithTimeout(fn: () => void | Promise<void>, timeout: number): Promise<void> {
    return new Promise((resolve, reject) => {
      const timerId = TimerManager.setTimeout(() => {
        reject(new Error(`测试超时: ${timeout}ms`));
      }, timeout, '测试超时');

      Promise.resolve(fn())
        .then(() => {
          TimerManager.clearTimeout(timerId);
          resolve();
        })
        .catch((error) => {
          TimerManager.clearTimeout(timerId);
          reject(error);
        });
    });
  }

  /**
   * 记录测试报告
   */
  private logReport(): void {
    if (!this.report) return;

    logger.info('=== 测试报告 ===');
    logger.info(`总测试套件: ${this.report.totalSuites}`);
    logger.info(`总测试用例: ${this.report.totalTests}`);
    logger.info(`通过: ${this.report.passedTests}`);
    logger.info(`失败: ${this.report.failedTests}`);
    logger.info(`跳过: ${this.report.skippedTests}`);
    logger.info(`错误: ${this.report.errorTests}`);
    logger.info(`总耗时: ${this.report.duration}ms`);

    // 记录失败的测试
    this.report.suiteResults.forEach(suiteResult => {
      const failedTests = suiteResult.results.filter(r => 
        r.result === TestResult.FAILED || r.result === TestResult.ERROR
      );
      
      if (failedTests.length > 0) {
        logger.error(`套件 ${suiteResult.name} 中的失败测试:`);
        failedTests.forEach(test => {
          logger.error(`  - ${test.name}: ${test.message || test.error?.message}`);
        });
      }
    });

    logger.info('=== 测试完成 ===');
  }

  /**
   * 获取最新的测试报告
   */
  getReport(): TestReport | null {
    return this.report;
  }

  /**
   * 清理测试运行器
   */
  cleanup(): void {
    this.suites = [];
    this.report = null;
  }
}

/**
 * 便捷函数：创建测试套件
 */
export function createTestSuite(name: string, description?: string): TestSuite {
  return {
    name,
    description,
    tests: []
  };
}

/**
 * 便捷函数：创建测试用例
 */
export function createTestCase(
  name: string, 
  test: () => void | Promise<void>,
  options?: {
    description?: string;
    timeout?: number;
    skip?: boolean;
    setup?: () => void | Promise<void>;
    teardown?: () => void | Promise<void>;
  }
): TestCase {
  return {
    name,
    test,
    description: options?.description,
    timeout: options?.timeout,
    skip: options?.skip,
    setup: options?.setup,
    teardown: options?.teardown
  };
}

/**
 * 便捷函数：创建模拟对象
 */
export function createMock(): Mock {
  return new Mock();
}

/**
 * 全局测试运行器实例
 */
export const testRunner = new TestRunner();
