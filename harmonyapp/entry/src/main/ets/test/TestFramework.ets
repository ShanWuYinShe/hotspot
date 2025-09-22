/**
 * 测试框架
 * 为GeoTalk HarmonyOS应用提供完整的测试支持
 */

import { hilog } from '@kit.PerformanceAnalysisKit';

const DOMAIN_NUMBER: number = 0xFF13;
const TAG: string = 'TestFramework';

/**
 * 测试结果接口
 */
export interface TestResult {
  name: string;
  passed: boolean;
  error?: string;
  duration: number;
  details?: Record<string, any>;
}

/**
 * 测试套件接口
 */
export interface TestSuite {
  name: string;
  tests: TestCase[];
  setup?: () => Promise<void>;
  teardown?: () => Promise<void>;
}

/**
 * 测试用例接口
 */
export interface TestCase {
  name: string;
  test: () => Promise<void>;
  timeout?: number;
}

/**
 * 断言工具类
 */
export class Assert {
  /**
   * 断言相等
   */
  static assertEqual<T>(actual: T, expected: T, message?: string): void {
    if (actual !== expected) {
      const errorMsg = message || `Expected ${expected}, but got ${actual}`;
      throw new Error(errorMsg);
    }
  }

  /**
   * 断言不相等
   */
  static assertNotEqual<T>(actual: T, expected: T, message?: string): void {
    if (actual === expected) {
      const errorMsg = message || `Expected not to be ${expected}, but got ${actual}`;
      throw new Error(errorMsg);
    }
  }

  /**
   * 断言为真
   */
  static assertTrue(condition: boolean, message?: string): void {
    if (!condition) {
      const errorMsg = message || 'Expected condition to be true';
      throw new Error(errorMsg);
    }
  }

  /**
   * 断言为假
   */
  static assertFalse(condition: boolean, message?: string): void {
    if (condition) {
      const errorMsg = message || 'Expected condition to be false';
      throw new Error(errorMsg);
    }
  }

  /**
   * 断言不为空
   */
  static assertNotNull<T>(value: T | null | undefined, message?: string): void {
    if (value === null || value === undefined) {
      const errorMsg = message || 'Expected value not to be null or undefined';
      throw new Error(errorMsg);
    }
  }

  /**
   * 断言为空
   */
  static assertNull<T>(value: T | null | undefined, message?: string): void {
    if (value !== null && value !== undefined) {
      const errorMsg = message || `Expected value to be null or undefined, but got ${value}`;
      throw new Error(errorMsg);
    }
  }

  /**
   * 断言数组包含元素
   */
  static assertContains<T>(array: T[], element: T, message?: string): void {
    if (!array.includes(element)) {
      const errorMsg = message || `Expected array to contain ${element}`;
      throw new Error(errorMsg);
    }
  }

  /**
   * 断言抛出异常
   */
  static async assertThrows(fn: () => Promise<void> | void, message?: string): Promise<void> {
    let threw = false;
    try {
      await fn();
    } catch (error) {
      threw = true;
    }
    
    if (!threw) {
      const errorMsg = message || 'Expected function to throw an error';
      throw new Error(errorMsg);
    }
  }

  /**
   * 断言字符串匹配正则表达式
   */
  static assertMatches(actual: string, pattern: RegExp, message?: string): void {
    if (!pattern.test(actual)) {
      const errorMsg = message || `Expected "${actual}" to match pattern ${pattern}`;
      throw new Error(errorMsg);
    }
  }
}

/**
 * 模拟工具类
 */
export class Mock {
  private static mocks: Map<string, any> = new Map();

  /**
   * 创建模拟函数
   */
  static fn<T extends (...args: any[]) => any>(implementation?: T): T & { calls: any[][]; } {
    const calls: any[][] = [];
    const mockFn = ((...args: any[]) => {
      calls.push(args);
      if (implementation) {
        return implementation(...args);
      }
    }) as T & { calls: any[][]; };
    
    mockFn.calls = calls;
    return mockFn;
  }

  /**
   * 模拟对象方法
   */
  static spyOn<T extends Record<string, any>, K extends keyof T>(
    object: T, 
    method: K, 
    implementation?: T[K]
  ): T[K] & { calls: any[][]; restore: () => void; } {
    const original = object[method];
    const calls: any[][] = [];
    
    const spy = ((...args: any[]) => {
      calls.push(args);
      if (implementation) {
        return implementation(...args);
      } else if (typeof original === 'function') {
        return original.apply(object, args);
      }
    }) as T[K] & { calls: any[][]; restore: () => void; };
    
    spy.calls = calls;
    spy.restore = () => {
      object[method] = original;
    };
    
    object[method] = spy;
    return spy;
  }

  /**
   * 清除所有模拟
   */
  static clearAll(): void {
    Mock.mocks.clear();
  }
}

/**
 * 测试运行器
 */
export class TestRunner {
  private results: TestResult[] = [];
  private currentSuite: string = '';

  /**
   * 运行单个测试用例
   */
  async runTest(testCase: TestCase): Promise<TestResult> {
    const startTime = Date.now();
    let result: TestResult;

    try {
      // 设置超时
      const timeout = testCase.timeout || 5000;
      const timeoutPromise = new Promise<never>((_, reject) => {
        setTimeout(() => reject(new Error('Test timeout')), timeout);
      });

      await Promise.race([testCase.test(), timeoutPromise]);
      
      result = {
        name: testCase.name,
        passed: true,
        duration: Date.now() - startTime
      };
      
      hilog.info(DOMAIN_NUMBER, TAG, `✓ ${testCase.name} (${result.duration}ms)`);
    } catch (error) {
      result = {
        name: testCase.name,
        passed: false,
        error: error instanceof Error ? error.message : String(error),
        duration: Date.now() - startTime
      };
      
      hilog.error(DOMAIN_NUMBER, TAG, `✗ ${testCase.name} - ${result.error}`);
    }

    this.results.push(result);
    return result;
  }

  /**
   * 运行测试套件
   */
  async runSuite(suite: TestSuite): Promise<TestResult[]> {
    this.currentSuite = suite.name;
    hilog.info(DOMAIN_NUMBER, TAG, `Running test suite: ${suite.name}`);

    const suiteResults: TestResult[] = [];

    try {
      // 执行setup
      if (suite.setup) {
        await suite.setup();
      }

      // 运行所有测试
      for (const testCase of suite.tests) {
        const result = await this.runTest(testCase);
        suiteResults.push(result);
      }

      // 执行teardown
      if (suite.teardown) {
        await suite.teardown();
      }
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `Test suite setup/teardown failed: ${error}`);
    }

    return suiteResults;
  }

  /**
   * 运行多个测试套件
   */
  async runSuites(suites: TestSuite[]): Promise<void> {
    hilog.info(DOMAIN_NUMBER, TAG, `Starting test run with ${suites.length} suites`);
    
    const startTime = Date.now();
    let totalTests = 0;
    let passedTests = 0;

    for (const suite of suites) {
      const results = await this.runSuite(suite);
      totalTests += results.length;
      passedTests += results.filter(r => r.passed).length;
    }

    const duration = Date.now() - startTime;
    const failedTests = totalTests - passedTests;

    hilog.info(DOMAIN_NUMBER, TAG, `Test run completed in ${duration}ms`);
    hilog.info(DOMAIN_NUMBER, TAG, `Total: ${totalTests}, Passed: ${passedTests}, Failed: ${failedTests}`);

    if (failedTests > 0) {
      hilog.error(DOMAIN_NUMBER, TAG, 'Some tests failed!');
      this.printFailedTests();
    } else {
      hilog.info(DOMAIN_NUMBER, TAG, 'All tests passed!');
    }
  }

  /**
   * 打印失败的测试
   */
  private printFailedTests(): void {
    const failedTests = this.results.filter(r => !r.passed);
    
    hilog.error(DOMAIN_NUMBER, TAG, '\nFailed Tests:');
    failedTests.forEach(test => {
      hilog.error(DOMAIN_NUMBER, TAG, `  ✗ ${test.name}: ${test.error}`);
    });
  }

  /**
   * 获取测试结果
   */
  getResults(): TestResult[] {
    return [...this.results];
  }

  /**
   * 清除测试结果
   */
  clearResults(): void {
    this.results = [];
  }
}

/**
 * 测试工具函数
 */
export class TestUtils {
  /**
   * 等待指定时间
   */
  static async sleep(ms: number): Promise<void> {
    return new Promise(resolve => setTimeout(resolve, ms));
  }

  /**
   * 等待条件满足
   */
  static async waitFor(
    condition: () => boolean, 
    timeout: number = 5000, 
    interval: number = 100
  ): Promise<void> {
    const startTime = Date.now();
    
    while (!condition()) {
      if (Date.now() - startTime > timeout) {
        throw new Error('Timeout waiting for condition');
      }
      await TestUtils.sleep(interval);
    }
  }

  /**
   * 生成随机字符串
   */
  static randomString(length: number = 10): string {
    const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
    let result = '';
    for (let i = 0; i < length; i++) {
      result += chars.charAt(Math.floor(Math.random() * chars.length));
    }
    return result;
  }

  /**
   * 生成随机数字
   */
  static randomNumber(min: number = 0, max: number = 100): number {
    return Math.floor(Math.random() * (max - min + 1)) + min;
  }

  /**
   * 生成测试用户ID
   */
  static generateTestUserId(userType: number = 2): string {
    const sequence = BigInt(Math.floor(Math.random() * 1000000));
    const typeBigInt = BigInt(userType) << 56n;
    return (typeBigInt | sequence).toString();
  }

  /**
   * 生成测试位置
   */
  static generateTestLocation(): { latitude: number; longitude: number } {
    return {
      latitude: 39.9042 + (Math.random() - 0.5) * 0.1, // 北京附近
      longitude: 116.4074 + (Math.random() - 0.5) * 0.1
    };
  }
}

/**
 * 性能测试工具
 */
export class PerformanceTest {
  /**
   * 测量函数执行时间
   */
  static async measureTime<T>(fn: () => Promise<T> | T): Promise<{ result: T; duration: number }> {
    const startTime = Date.now();
    const result = await fn();
    const duration = Date.now() - startTime;
    
    return { result, duration };
  }

  /**
   * 压力测试
   */
  static async stressTest<T>(
    fn: () => Promise<T> | T, 
    iterations: number = 100
  ): Promise<{ 
    totalTime: number; 
    averageTime: number; 
    minTime: number; 
    maxTime: number; 
    successCount: number;
    errorCount: number;
  }> {
    const times: number[] = [];
    let successCount = 0;
    let errorCount = 0;
    
    const startTime = Date.now();
    
    for (let i = 0; i < iterations; i++) {
      try {
        const { duration } = await PerformanceTest.measureTime(fn);
        times.push(duration);
        successCount++;
      } catch (error) {
        errorCount++;
        hilog.warn(DOMAIN_NUMBER, TAG, `Stress test iteration ${i} failed: ${error}`);
      }
    }
    
    const totalTime = Date.now() - startTime;
    const averageTime = times.length > 0 ? times.reduce((a, b) => a + b, 0) / times.length : 0;
    const minTime = times.length > 0 ? Math.min(...times) : 0;
    const maxTime = times.length > 0 ? Math.max(...times) : 0;
    
    return {
      totalTime,
      averageTime,
      minTime,
      maxTime,
      successCount,
      errorCount
    };
  }
}

// 导出便捷的测试函数
export const describe = (name: string, tests: TestCase[]): TestSuite => ({
  name,
  tests
});

export const it = (name: string, test: () => Promise<void>, timeout?: number): TestCase => ({
  name,
  test,
  timeout
});

export const expect = <T>(actual: T) => ({
  toBe: (expected: T) => Assert.assertEqual(actual, expected),
  not: {
    toBe: (expected: T) => Assert.assertNotEqual(actual, expected)
  },
  toBeNull: () => Assert.assertNull(actual),
  toBeUndefined: () => Assert.assertEqual(actual, undefined),
  toBeTruthy: () => Assert.assertTrue(!!actual),
  toBeFalsy: () => Assert.assertFalse(!!actual),
  toContain: (element: any) => {
    if (Array.isArray(actual)) {
      Assert.assertContains(actual, element);
    } else {
      throw new Error('Expected array for toContain');
    }
  },
  toMatch: (pattern: RegExp) => {
    if (typeof actual === 'string') {
      Assert.assertMatches(actual, pattern);
    } else {
      throw new Error('Expected string for toMatch');
    }
  }
});

export const beforeEach = (fn: () => Promise<void> | void) => fn;
export const afterEach = (fn: () => Promise<void> | void) => fn;

