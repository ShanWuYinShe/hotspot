/**
 * 统一错误处理机制
 * 提供标准化的错误处理、记录和恢复策略
 */

import { BusinessError } from '@kit.BasicServicesKit';
import { createLogger, LogLevel } from './LogManager';

const logger = createLogger('ErrorHandler');

/**
 * 错误级别枚举
 */
export enum ErrorLevel {
  LOW = 'low',           // 低级错误，不影响主要功能
  MEDIUM = 'medium',     // 中级错误，影响部分功能
  HIGH = 'high',         // 高级错误，影响主要功能
  CRITICAL = 'critical'  // 严重错误，应用无法正常运行
}

/**
 * 错误类型枚举
 */
export enum ErrorType {
  NETWORK = 'network',           // 网络错误
  STORAGE = 'storage',           // 存储错误
  PERMISSION = 'permission',     // 权限错误
  VALIDATION = 'validation',     // 数据验证错误
  BUSINESS = 'business',         // 业务逻辑错误
  SYSTEM = 'system',             // 系统错误
  UNKNOWN = 'unknown'            // 未知错误
}

/**
 * 标准化错误信息接口
 */
export interface StandardError {
  code: string;
  message: string;
  type: ErrorType;
  level: ErrorLevel;
  context: string;
  timestamp: number;
  stack?: string;
  originalError?: Object;
  userMessage?: string;
  recoveryAction?: string;
}

/**
 * 错误处理配置
 */
interface ErrorHandlerConfig {
  enableLogging: boolean;
  enableUserNotification: boolean;
  enableRecovery: boolean;
  maxErrorHistory: number;
  logLevel: LogLevel;
}

/**
 * 错误统计信息
 */
export interface ErrorStats {
  totalErrors: number;
  errorsByType: Record<ErrorType, number>;
  errorsByLevel: Record<ErrorLevel, number>;
  recentErrors: StandardError[];
  mostFrequentError: string;
}

/**
 * 统一错误处理器
 */
export class ErrorHandler {
  private static config: ErrorHandlerConfig = {
    enableLogging: true,
    enableUserNotification: false,
    enableRecovery: true,
    maxErrorHistory: 100,
    logLevel: LogLevel.ERROR
  };

  private static errorHistory: StandardError[] = [];
  private static errorListeners: ((error: StandardError) => void)[] = [];

  /**
   * 设置错误处理配置
   */
  static setConfig(config: Partial<ErrorHandlerConfig>): void {
    // 手动合并配置，避免类型兼容性问题
    if (config.enableLogging !== undefined) {
      ErrorHandler.config.enableLogging = config.enableLogging;
    }
    if (config.enableUserNotification !== undefined) {
      ErrorHandler.config.enableUserNotification = config.enableUserNotification;
    }
    if (config.enableRecovery !== undefined) {
      ErrorHandler.config.enableRecovery = config.enableRecovery;
    }
    if (config.maxErrorHistory !== undefined) {
      ErrorHandler.config.maxErrorHistory = config.maxErrorHistory;
    }
    if (config.logLevel !== undefined) {
      ErrorHandler.config.logLevel = config.logLevel;
    }
  }

  /**
   * 处理错误的主要方法
   */
  static handle(
    error: Object,
    context: string,
    type: ErrorType = ErrorType.UNKNOWN,
    level: ErrorLevel = ErrorLevel.MEDIUM
  ): StandardError {
    const standardError = ErrorHandler.standardizeError(error, context, type, level);
    
    // 记录错误
    if (ErrorHandler.config.enableLogging) {
      ErrorHandler.logError(standardError);
    }

    // 添加到历史记录
    ErrorHandler.addToHistory(standardError);

    // 通知监听器
    ErrorHandler.notifyListeners(standardError);

    // 尝试自动恢复
    if (ErrorHandler.config.enableRecovery) {
      ErrorHandler.attemptRecovery(standardError);
    }

    return standardError;
  }

  /**
   * 处理并抛出错误
   */
  static handleAndThrow(
    error: Object,
    context: string,
    type: ErrorType = ErrorType.UNKNOWN,
    level: ErrorLevel = ErrorLevel.MEDIUM
  ): never {
    const standardError = ErrorHandler.handle(error, context, type, level);
    throw new Error(standardError.message);
  }

  /**
   * 安全执行函数，捕获并处理错误
   */
  static async safeExecute<T>(
    fn: () => Promise<T>,
    context: string,
    fallbackValue?: T,
    type: ErrorType = ErrorType.UNKNOWN
  ): Promise<T | undefined> {
    try {
      return await fn();
    } catch (error) {
      ErrorHandler.handle(error, context, type, ErrorLevel.MEDIUM);
      return fallbackValue;
    }
  }

  /**
   * 同步版本的安全执行
   */
  static safeExecuteSync<T>(
    fn: () => T,
    context: string,
    fallbackValue?: T,
    type: ErrorType = ErrorType.UNKNOWN
  ): T | undefined {
    try {
      return fn();
    } catch (error) {
      ErrorHandler.handle(error, context, type, ErrorLevel.MEDIUM);
      return fallbackValue;
    }
  }

  /**
   * 标准化错误信息
   */
  private static standardizeError(
    error: Object,
    context: string,
    type: ErrorType,
    level: ErrorLevel
  ): StandardError {
    let message = '未知错误';
    let code = 'UNKNOWN_ERROR';
    let stack: string | undefined;
    let originalError: Object = error;

    if (error instanceof Error) {
      message = error.message;
      stack = error.stack;
      code = error.name || 'ERROR';
    } else if (error && typeof error === 'object') {
      const businessError = error as BusinessError;
      if (businessError.code !== undefined) {
        code = String(businessError.code);
        message = businessError.message || message;
      } else {
        message = JSON.stringify(error);
      }
    } else if (typeof error === 'string') {
      message = error;
    }

    // 根据错误类型生成用户友好的消息
    const userMessage = ErrorHandler.generateUserMessage(type, message);
    const recoveryAction = ErrorHandler.generateRecoveryAction(type);

    return {
      code,
      message,
      type,
      level,
      context,
      timestamp: Date.now(),
      stack,
      originalError,
      userMessage,
      recoveryAction
    };
  }

  /**
   * 生成用户友好的错误消息
   */
  private static generateUserMessage(type: ErrorType, originalMessage: string): string {
    switch (type) {
      case ErrorType.NETWORK:
        return '网络连接出现问题，请检查网络设置';
      case ErrorType.STORAGE:
        return '数据存储出现问题，请重试';
      case ErrorType.PERMISSION:
        return '缺少必要权限，请在设置中开启相关权限';
      case ErrorType.VALIDATION:
        return '输入的数据格式不正确，请检查后重试';
      case ErrorType.BUSINESS:
        return originalMessage; // 业务错误通常已经是用户友好的
      case ErrorType.SYSTEM:
        return '系统出现问题，请稍后重试';
      default:
        return '操作失败，请重试';
    }
  }

  /**
   * 生成恢复建议
   */
  private static generateRecoveryAction(type: ErrorType): string {
    switch (type) {
      case ErrorType.NETWORK:
        return '检查网络连接或稍后重试';
      case ErrorType.STORAGE:
        return '清理应用缓存或重启应用';
      case ErrorType.PERMISSION:
        return '前往设置页面开启相关权限';
      case ErrorType.VALIDATION:
        return '检查输入格式是否正确';
      case ErrorType.BUSINESS:
        return '按照提示信息进行操作';
      case ErrorType.SYSTEM:
        return '重启应用或联系技术支持';
      default:
        return '重试操作或重启应用';
    }
  }

  /**
   * 记录错误日志
   */
  private static logError(error: StandardError): void {
    const logMessage = `[${error.type.toUpperCase()}] ${error.context}: ${error.message}`;
    
    switch (error.level) {
      case ErrorLevel.LOW:
        logger.debug(logMessage);
        break;
      case ErrorLevel.MEDIUM:
        logger.warn(logMessage);
        break;
      case ErrorLevel.HIGH:
        logger.error(logMessage);
        break;
      case ErrorLevel.CRITICAL:
        logger.fatal(logMessage);
        break;
    }

    if (error.stack) {
      logger.debug(`错误堆栈: ${error.stack}`);
    }
  }

  /**
   * 添加到错误历史记录
   */
  private static addToHistory(error: StandardError): void {
    ErrorHandler.errorHistory.push(error);
    
    // 保持历史记录在限制范围内
    if (ErrorHandler.errorHistory.length > ErrorHandler.config.maxErrorHistory) {
      ErrorHandler.errorHistory = ErrorHandler.errorHistory.slice(-ErrorHandler.config.maxErrorHistory);
    }
  }

  /**
   * 通知错误监听器
   */
  private static notifyListeners(error: StandardError): void {
    ErrorHandler.errorListeners.forEach(listener => {
      try {
        listener(error);
      } catch (listenerError) {
        logger.error(`错误监听器执行失败: ${listenerError}`);
      }
    });
  }

  /**
   * 尝试自动恢复
   */
  private static attemptRecovery(error: StandardError): void {
    // 根据错误类型尝试不同的恢复策略
    switch (error.type) {
      case ErrorType.NETWORK:
        // 网络错误可以尝试重连
        logger.info(`尝试网络恢复: ${error.context}`);
        break;
      case ErrorType.STORAGE:
        // 存储错误可以尝试清理缓存
        logger.info(`尝试存储恢复: ${error.context}`);
        break;
      default:
        // 其他错误暂不自动恢复
        break;
    }
  }

  /**
   * 添加错误监听器
   */
  static addErrorListener(listener: (error: StandardError) => void): void {
    ErrorHandler.errorListeners.push(listener);
  }

  /**
   * 移除错误监听器
   */
  static removeErrorListener(listener: (error: StandardError) => void): void {
    const index = ErrorHandler.errorListeners.indexOf(listener);
    if (index > -1) {
      ErrorHandler.errorListeners.splice(index, 1);
    }
  }

  /**
   * 获取错误统计信息
   */
  static getStats(): ErrorStats {
    const errorsByType: Record<ErrorType, number> = {
      [ErrorType.NETWORK]: 0,
      [ErrorType.STORAGE]: 0,
      [ErrorType.PERMISSION]: 0,
      [ErrorType.VALIDATION]: 0,
      [ErrorType.BUSINESS]: 0,
      [ErrorType.SYSTEM]: 0,
      [ErrorType.UNKNOWN]: 0
    };

    const errorsByLevel: Record<ErrorLevel, number> = {
      [ErrorLevel.LOW]: 0,
      [ErrorLevel.MEDIUM]: 0,
      [ErrorLevel.HIGH]: 0,
      [ErrorLevel.CRITICAL]: 0
    };

    const errorCounts: Record<string, number> = {};

    ErrorHandler.errorHistory.forEach(error => {
      errorsByType[error.type]++;
      errorsByLevel[error.level]++;
      errorCounts[error.code] = (errorCounts[error.code] || 0) + 1;
    });

    const mostFrequentError = Object.keys(errorCounts).reduce((a, b) => 
      errorCounts[a] > errorCounts[b] ? a : b, '');

    return {
      totalErrors: ErrorHandler.errorHistory.length,
      errorsByType,
      errorsByLevel,
      recentErrors: ErrorHandler.errorHistory.slice(-10),
      mostFrequentError
    };
  }

  /**
   * 清除错误历史记录
   */
  static clearHistory(): void {
    ErrorHandler.errorHistory = [];
    logger.info('错误历史记录已清除');
  }

  /**
   * 获取所有错误历史记录
   */
  static getHistory(): StandardError[] {
    return [...ErrorHandler.errorHistory];
  }
}
