/**
 * ErrorHandler 单元测试
 */

import { 
  TestSuite, 
  TestCase, 
  Assert, 
  createTestSuite, 
  createTestCase,
  createMock
} from '../framework/TestFramework';
import { 
  ErrorHandler, 
  ErrorType, 
  ErrorLevel, 
  StandardError 
} from '../../../main/ets/common/utils/ErrorHandler';

/**
 * ErrorHandler 测试套件
 */
export function createErrorHandlerTestSuite(): TestSuite {
  const suite = createTestSuite('ErrorHandler', 'ErrorHandler 错误处理器测试');

  // 设置和清理
  suite.setup = () => {
    ErrorHandler.clearHistory();
  };

  suite.teardown = () => {
    ErrorHandler.clearHistory();
  };

  // 测试用例：基本错误处理
  suite.tests.push(createTestCase(
    '应该能够处理基本错误',
    () => {
      const error = new Error('测试错误');
      const standardError = ErrorHandler.handle(
        error, 
        'TestContext', 
        ErrorType.SYSTEM, 
        ErrorLevel.MEDIUM
      );

      Assert.equals(standardError.message, '测试错误');
      Assert.equals(standardError.context, 'TestContext');
      Assert.equals(standardError.type, ErrorType.SYSTEM);
      Assert.equals(standardError.level, ErrorLevel.MEDIUM);
      Assert.isNotNull(standardError.timestamp);
      Assert.isNotNull(standardError.code);
    }
  ));

  // 测试用例：字符串错误处理
  suite.tests.push(createTestCase(
    '应该能够处理字符串错误',
    () => {
      const standardError = ErrorHandler.handle(
        '这是一个字符串错误', 
        'StringContext', 
        ErrorType.VALIDATION
      );

      Assert.equals(standardError.message, '这是一个字符串错误');
      Assert.equals(standardError.context, 'StringContext');
      Assert.equals(standardError.type, ErrorType.VALIDATION);
    }
  ));

  // 测试用例：对象错误处理
  suite.tests.push(createTestCase(
    '应该能够处理对象错误',
    () => {
      const errorObj = { code: 500, message: '服务器错误' };
      const standardError = ErrorHandler.handle(
        errorObj, 
        'ObjectContext', 
        ErrorType.NETWORK
      );

      Assert.equals(standardError.code, '500');
      Assert.equals(standardError.message, '服务器错误');
      Assert.equals(standardError.type, ErrorType.NETWORK);
    }
  ));

  // 测试用例：用户友好消息生成
  suite.tests.push(createTestCase(
    '应该能够生成用户友好的错误消息',
    () => {
      const networkError = ErrorHandler.handle(
        new Error('Connection failed'), 
        'NetworkTest', 
        ErrorType.NETWORK
      );
      Assert.stringContains(networkError.userMessage!, '网络连接');

      const storageError = ErrorHandler.handle(
        new Error('Storage full'), 
        'StorageTest', 
        ErrorType.STORAGE
      );
      Assert.stringContains(storageError.userMessage!, '数据存储');

      const permissionError = ErrorHandler.handle(
        new Error('Permission denied'), 
        'PermissionTest', 
        ErrorType.PERMISSION
      );
      Assert.stringContains(permissionError.userMessage!, '权限');
    }
  ));

  // 测试用例：恢复建议生成
  suite.tests.push(createTestCase(
    '应该能够生成恢复建议',
    () => {
      const networkError = ErrorHandler.handle(
        new Error('Network error'), 
        'NetworkTest', 
        ErrorType.NETWORK
      );
      Assert.stringContains(networkError.recoveryAction!, '网络');

      const validationError = ErrorHandler.handle(
        new Error('Validation error'), 
        'ValidationTest', 
        ErrorType.VALIDATION
      );
      Assert.stringContains(validationError.recoveryAction!, '输入');
    }
  ));

  // 测试用例：安全执行
  suite.tests.push(createTestCase(
    '应该能够安全执行可能失败的函数',
    async () => {
      // 成功的情况
      const result1 = await ErrorHandler.safeExecute(
        async () => 'success',
        'SafeExecuteTest'
      );
      Assert.equals(result1, 'success');

      // 失败的情况，使用默认值
      const result2 = await ErrorHandler.safeExecute(
        async () => { throw new Error('test error'); },
        'SafeExecuteTest',
        'default'
      );
      Assert.equals(result2, 'default');

      // 失败的情况，无默认值
      const result3 = await ErrorHandler.safeExecute(
        async () => { throw new Error('test error'); },
        'SafeExecuteTest'
      );
      Assert.isUndefined(result3);
    }
  ));

  // 测试用例：同步安全执行
  suite.tests.push(createTestCase(
    '应该能够安全执行同步函数',
    () => {
      // 成功的情况
      const result1 = ErrorHandler.safeExecuteSync(
        () => 42,
        'SyncTest'
      );
      Assert.equals(result1, 42);

      // 失败的情况
      const result2 = ErrorHandler.safeExecuteSync(
        () => { throw new Error('sync error'); },
        'SyncTest',
        'fallback'
      );
      Assert.equals(result2, 'fallback');
    }
  ));

  // 测试用例：错误历史记录
  suite.tests.push(createTestCase(
    '应该能够记录和获取错误历史',
    () => {
      ErrorHandler.clearHistory();

      ErrorHandler.handle(new Error('错误1'), 'Context1', ErrorType.SYSTEM);
      ErrorHandler.handle(new Error('错误2'), 'Context2', ErrorType.NETWORK);
      ErrorHandler.handle(new Error('错误3'), 'Context3', ErrorType.STORAGE);

      const history = ErrorHandler.getHistory();
      Assert.arrayLength(history, 3);
      Assert.equals(history[0].message, '错误1');
      Assert.equals(history[1].message, '错误2');
      Assert.equals(history[2].message, '错误3');
    }
  ));

  // 测试用例：错误统计
  suite.tests.push(createTestCase(
    '应该能够获取错误统计信息',
    () => {
      ErrorHandler.clearHistory();

      // 添加不同类型和级别的错误
      ErrorHandler.handle(new Error('系统错误1'), 'Context1', ErrorType.SYSTEM, ErrorLevel.HIGH);
      ErrorHandler.handle(new Error('系统错误2'), 'Context2', ErrorType.SYSTEM, ErrorLevel.LOW);
      ErrorHandler.handle(new Error('网络错误1'), 'Context3', ErrorType.NETWORK, ErrorLevel.MEDIUM);
      ErrorHandler.handle(new Error('存储错误1'), 'Context4', ErrorType.STORAGE, ErrorLevel.CRITICAL);

      const stats = ErrorHandler.getStats();
      Assert.equals(stats.totalErrors, 4);
      Assert.equals(stats.errorsByType[ErrorType.SYSTEM], 2);
      Assert.equals(stats.errorsByType[ErrorType.NETWORK], 1);
      Assert.equals(stats.errorsByType[ErrorType.STORAGE], 1);
      Assert.equals(stats.errorsByLevel[ErrorLevel.HIGH], 1);
      Assert.equals(stats.errorsByLevel[ErrorLevel.LOW], 1);
      Assert.equals(stats.errorsByLevel[ErrorLevel.MEDIUM], 1);
      Assert.equals(stats.errorsByLevel[ErrorLevel.CRITICAL], 1);
      Assert.arrayLength(stats.recentErrors, 4);
    }
  ));

  // 测试用例：错误监听器
  suite.tests.push(createTestCase(
    '应该能够添加和移除错误监听器',
    () => {
      let capturedError: StandardError | null = null;
      
      const listener = (error: StandardError) => {
        capturedError = error;
      };

      ErrorHandler.addErrorListener(listener);

      // 触发错误
      ErrorHandler.handle(new Error('监听器测试'), 'ListenerTest', ErrorType.SYSTEM);

      Assert.isNotNull(capturedError);
      Assert.equals(capturedError!.message, '监听器测试');

      // 移除监听器
      ErrorHandler.removeErrorListener(listener);
      capturedError = null;

      // 再次触发错误，应该不会被捕获
      ErrorHandler.handle(new Error('监听器测试2'), 'ListenerTest2', ErrorType.SYSTEM);
      Assert.isNull(capturedError);
    }
  ));

  // 测试用例：配置设置
  suite.tests.push(createTestCase(
    '应该能够设置错误处理配置',
    () => {
      ErrorHandler.setConfig({
        enableLogging: false,
        enableReporting: true,
        maxErrorHistory: 50,
        autoRecovery: false
      });

      // 这里主要测试配置设置不会抛出错误
      // 实际的配置效果需要在集成测试中验证
      Assert.isTrue(true);
    }
  ));

  // 测试用例：处理并抛出错误
  suite.tests.push(createTestCase(
    '应该能够处理错误并重新抛出',
    async () => {
      await Assert.throws(async () => {
        ErrorHandler.handleAndThrow(
          new Error('需要抛出的错误'),
          'ThrowTest',
          ErrorType.BUSINESS,
          ErrorLevel.HIGH
        );
      });

      // 验证错误被记录到历史中
      const history = ErrorHandler.getHistory();
      const lastError = history[history.length - 1];
      Assert.equals(lastError.message, '需要抛出的错误');
      Assert.equals(lastError.context, 'ThrowTest');
    }
  ));

  // 测试用例：清理历史记录
  suite.tests.push(createTestCase(
    '应该能够清理错误历史记录',
    () => {
      // 添加一些错误
      ErrorHandler.handle(new Error('错误1'), 'Context1', ErrorType.SYSTEM);
      ErrorHandler.handle(new Error('错误2'), 'Context2', ErrorType.NETWORK);

      let history = ErrorHandler.getHistory();
      Assert.isTrue(history.length > 0);

      // 清理历史
      ErrorHandler.clearHistory();

      history = ErrorHandler.getHistory();
      Assert.arrayLength(history, 0);

      const stats = ErrorHandler.getStats();
      Assert.equals(stats.totalErrors, 0);
    }
  ));

  return suite;
}

/**
 * 运行 ErrorHandler 测试
 */
export async function runErrorHandlerTests(): Promise<void> {
  const { testRunner } = await import('../framework/TestFramework');
  const suite = createErrorHandlerTestSuite();
  testRunner.addSuite(suite);
}
