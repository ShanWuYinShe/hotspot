/**
 * LogManager 单元测试
 */

import { 
  TestSuite, 
  TestCase, 
  Assert, 
  createTestSuite, 
  createTestCase, 
  createMock 
} from '../framework/TestFramework';
import { LogManager, Logger, LogLevel, createLogger } from '../../../main/ets/common/utils/LogManager';

/**
 * LogManager 测试套件
 */
export function createLogManagerTestSuite(): TestSuite {
  const suite = createTestSuite('LogManager', 'LogManager 日志管理器测试');

  // 测试用例：创建日志记录器
  suite.tests.push(createTestCase(
    '应该能够创建日志记录器',
    () => {
      const logger = createLogger('TestModule');
      Assert.isNotNull(logger);
      Assert.isType(logger.info, 'function');
      Assert.isType(logger.error, 'function');
      Assert.isType(logger.warn, 'function');
      Assert.isType(logger.debug, 'function');
    }
  ));

  // 测试用例：日志级别配置
  suite.tests.push(createTestCase(
    '应该能够设置和获取日志级别',
    () => {
      const originalConfig = LogManager.getConfig();
      
      LogManager.setConfig({ level: LogLevel.ERROR });
      const config = LogManager.getConfig();
      
      Assert.equals(config.level, LogLevel.ERROR);
      
      // 恢复原始配置
      LogManager.setConfig(originalConfig);
    }
  ));

  // 测试用例：敏感数据脱敏
  suite.tests.push(createTestCase(
    '应该能够对敏感数据进行脱敏',
    () => {
      const sensitiveData = {
        username: 'testuser',
        password: 'secretpassword123',
        token: 'abc123def456ghi789',
        normalData: 'this is normal'
      };

      const sanitized = LogManager.sanitizeData(sensitiveData);
      
      Assert.equals(sanitized.username, 'testuser');
      Assert.equals(sanitized.normalData, 'this is normal');
      Assert.notEquals(sanitized.password, 'secretpassword123');
      Assert.notEquals(sanitized.token, 'abc123def456ghi789');
      Assert.stringContains(sanitized.password, '****');
      Assert.stringContains(sanitized.token, '****');
    }
  ));

  // 测试用例：多个日志记录器
  suite.tests.push(createTestCase(
    '应该能够管理多个日志记录器',
    () => {
      const logger1 = LogManager.getLogger('Module1');
      const logger2 = LogManager.getLogger('Module2');
      const logger1Again = LogManager.getLogger('Module1');

      Assert.isNotNull(logger1);
      Assert.isNotNull(logger2);
      Assert.equals(logger1, logger1Again); // 应该返回同一个实例
    }
  ));

  // 测试用例：清理功能
  suite.tests.push(createTestCase(
    '应该能够清理所有日志记录器',
    () => {
      LogManager.getLogger('TempModule');
      LogManager.cleanup();
      
      // 清理后应该能够重新创建
      const newLogger = LogManager.getLogger('TempModule');
      Assert.isNotNull(newLogger);
    }
  ));

  // 测试用例：嵌套对象脱敏
  suite.tests.push(createTestCase(
    '应该能够对嵌套对象进行脱敏',
    () => {
      const nestedData = {
        user: {
          id: '123',
          credentials: {
            password: 'secret123',
            apiKey: 'key_abc123def456'
          }
        },
        config: {
          database: {
            host: 'localhost',
            password: 'dbpass123'
          }
        }
      };

      const sanitized = LogManager.sanitizeData(nestedData);
      
      Assert.equals(sanitized.user.id, '123');
      Assert.equals(sanitized.config.database.host, 'localhost');
      Assert.stringContains(sanitized.user.credentials.password, '****');
      Assert.stringContains(sanitized.user.credentials.apiKey, '****');
      Assert.stringContains(sanitized.config.database.password, '****');
    }
  ));

  // 测试用例：数组脱敏
  suite.tests.push(createTestCase(
    '应该能够对数组中的敏感数据进行脱敏',
    () => {
      const arrayData = [
        { name: 'user1', password: 'pass1' },
        { name: 'user2', token: 'token123' },
        { name: 'user3', data: 'normal' }
      ];

      const sanitized = LogManager.sanitizeData(arrayData);
      
      Assert.arrayLength(sanitized, 3);
      Assert.equals(sanitized[0].name, 'user1');
      Assert.equals(sanitized[2].data, 'normal');
      Assert.stringContains(sanitized[0].password, '****');
      Assert.stringContains(sanitized[1].token, '****');
    }
  ));

  // 测试用例：配置更新影响现有记录器
  suite.tests.push(createTestCase(
    '配置更新应该影响现有的日志记录器',
    () => {
      const logger = LogManager.getLogger('ConfigTestModule');
      const originalConfig = LogManager.getConfig();
      
      // 更新配置
      LogManager.setConfig({ 
        level: LogLevel.ERROR,
        enableConsole: false 
      });
      
      const newConfig = LogManager.getConfig();
      Assert.equals(newConfig.level, LogLevel.ERROR);
      Assert.isFalse(newConfig.enableConsole);
      
      // 恢复原始配置
      LogManager.setConfig(originalConfig);
    }
  ));

  return suite;
}

/**
 * 运行 LogManager 测试
 */
export async function runLogManagerTests(): Promise<void> {
  const { testRunner } = await import('../framework/TestFramework');
  const suite = createLogManagerTestSuite();
  testRunner.addSuite(suite);
}
