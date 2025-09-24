/**
 * 错误处理装饰器
 * 提供统一的错误处理装饰器，减少重复的try-catch代码
 */

import { ErrorHandler, ErrorType, ErrorLevel } from '../utils/ErrorHandler';
import { createLogger } from '../utils/LogManager';

const logger = createLogger('ErrorDecorator');

/**
 * 错误处理选项
 */
interface ErrorHandlingOptions {
  /**
   * 错误上下文
   */
  context?: string;
  
  /**
   * 错误类型
   */
  type?: ErrorType;
  
  /**
   * 错误级别
   */
  level?: ErrorLevel;
  
  /**
   * 是否重新抛出错误
   */
  rethrow?: boolean;
  
  /**
   * 是否记录错误
   */
  log?: boolean;
  
  /**
   * 默认返回值（当不重新抛出错误时）
   */
  defaultValue?: any;
  
  /**
   * 自定义错误处理函数
   */
  customHandler?: (error: any, context: string) => any;
}

/**
 * 重试选项
 */
interface RetryOptions {
  /**
   * 最大重试次数
   */
  maxAttempts?: number;
  
  /**
   * 重试延迟（毫秒）
   */
  delay?: number;
  
  /**
   * 退避倍数
   */
  backoffMultiplier?: number;
  
  /**
   * 最大延迟（毫秒）
   */
  maxDelay?: number;
  
  /**
   * 重试条件函数
   */
  shouldRetry?: (error: any, attempt: number) => boolean;
}

/**
 * 方法装饰器：错误处理
 * @param options 错误处理选项
 */
export function withErrorHandling(options: ErrorHandlingOptions = {}) {
  return function (target: any, propertyName: string, descriptor: PropertyDescriptor) {
    const originalMethod = descriptor.value;
    const context = options.context || `${target.constructor.name}.${propertyName}`;
    
    descriptor.value = async function (...args: any[]) {
      try {
        const result = await originalMethod.apply(this, args);
        return result;
      } catch (error) {
        // 记录错误
        if (options.log !== false) {
          logger.error(`方法 ${context} 执行失败`, error);
        }
        
        // 自定义错误处理
        if (options.customHandler) {
          return options.customHandler(error, context);
        }
        
        // 使用ErrorHandler处理
        const standardError = ErrorHandler.handle(
          error,
          context,
          options.type || ErrorType.SYSTEM,
          options.level || ErrorLevel.MEDIUM
        );
        
        // 是否重新抛出错误
        if (options.rethrow !== false) {
          throw standardError;
        }
        
        // 返回默认值
        return options.defaultValue;
      }
    };
    
    return descriptor;
  };
}

/**
 * 方法装饰器：带重试的错误处理
 * @param retryOptions 重试选项
 * @param errorOptions 错误处理选项
 */
export function withRetry(
  retryOptions: RetryOptions = {},
  errorOptions: ErrorHandlingOptions = {}
) {
  return function (target: any, propertyName: string, descriptor: PropertyDescriptor) {
    const originalMethod = descriptor.value;
    const context = errorOptions.context || `${target.constructor.name}.${propertyName}`;
    
    const maxAttempts = retryOptions.maxAttempts || 3;
    const delay = retryOptions.delay || 1000;
    const backoffMultiplier = retryOptions.backoffMultiplier || 2;
    const maxDelay = retryOptions.maxDelay || 10000;
    
    descriptor.value = async function (...args: any[]) {
      let lastError: any;
      let currentDelay = delay;
      
      for (let attempt = 1; attempt <= maxAttempts; attempt++) {
        try {
          const result = await originalMethod.apply(this, args);
          
          // 成功时记录重试信息
          if (attempt > 1) {
            logger.info(`方法 ${context} 在第 ${attempt} 次尝试后成功`);
          }
          
          return result;
        } catch (error) {
          lastError = error;
          
          // 检查是否应该重试
          const shouldRetry = retryOptions.shouldRetry 
            ? retryOptions.shouldRetry(error, attempt)
            : attempt < maxAttempts;
          
          if (!shouldRetry) {
            break;
          }
          
          // 记录重试信息
          logger.warn(`方法 ${context} 第 ${attempt} 次尝试失败，${currentDelay}ms 后重试`, error);
          
          // 等待重试
          await new Promise(resolve => setTimeout(resolve, currentDelay));
          
          // 增加延迟（指数退避）
          currentDelay = Math.min(currentDelay * backoffMultiplier, maxDelay);
        }
      }
      
      // 所有重试都失败了
      logger.error(`方法 ${context} 在 ${maxAttempts} 次尝试后仍然失败`, lastError);
      
      // 使用错误处理装饰器
      const errorHandler = withErrorHandling(errorOptions);
      const errorDescriptor = {
        value: () => { throw lastError; }
      };
      
      errorHandler(target, propertyName, errorDescriptor);
      return errorDescriptor.value();
    };
    
    return descriptor;
  };
}

/**
 * 方法装饰器：性能监控
 * @param threshold 性能阈值（毫秒）
 */
export function withPerformanceMonitoring(threshold: number = 1000) {
  return function (target: any, propertyName: string, descriptor: PropertyDescriptor) {
    const originalMethod = descriptor.value;
    const context = `${target.constructor.name}.${propertyName}`;
    
    descriptor.value = async function (...args: any[]) {
      const startTime = Date.now();
      
      try {
        const result = await originalMethod.apply(this, args);
        const duration = Date.now() - startTime;
        
        // 记录性能信息
        if (duration > threshold) {
          logger.warn(`方法 ${context} 执行时间过长: ${duration}ms (阈值: ${threshold}ms)`);
        } else {
          logger.debug(`方法 ${context} 执行时间: ${duration}ms`);
        }
        
        return result;
      } catch (error) {
        const duration = Date.now() - startTime;
        logger.error(`方法 ${context} 执行失败，耗时: ${duration}ms`, error);
        throw error;
      }
    };
    
    return descriptor;
  };
}

/**
 * 方法装饰器：参数验证
 * @param validator 验证函数
 */
export function withValidation(
  validator: (...args: any[]) => boolean | string
) {
  return function (target: any, propertyName: string, descriptor: PropertyDescriptor) {
    const originalMethod = descriptor.value;
    const context = `${target.constructor.name}.${propertyName}`;
    
    descriptor.value = function (...args: any[]) {
      // 执行参数验证
      const validationResult = validator(...args);
      
      if (validationResult === false) {
        const error = new Error(`方法 ${context} 参数验证失败`);
        ErrorHandler.handle(error, context, ErrorType.VALIDATION, ErrorLevel.MEDIUM);
        throw error;
      }
      
      if (typeof validationResult === 'string') {
        const error = new Error(`方法 ${context} 参数验证失败: ${validationResult}`);
        ErrorHandler.handle(error, context, ErrorType.VALIDATION, ErrorLevel.MEDIUM);
        throw error;
      }
      
      return originalMethod.apply(this, args);
    };
    
    return descriptor;
  };
}

/**
 * 方法装饰器：缓存结果
 * @param ttl 缓存生存时间（毫秒）
 * @param keyGenerator 缓存键生成函数
 */
export function withCache(
  ttl: number = 5 * 60 * 1000,
  keyGenerator?: (...args: any[]) => string
) {
  const cache = new Map<string, { value: any; expiry: number }>();
  
  return function (target: any, propertyName: string, descriptor: PropertyDescriptor) {
    const originalMethod = descriptor.value;
    const context = `${target.constructor.name}.${propertyName}`;
    
    descriptor.value = async function (...args: any[]) {
      // 生成缓存键
      const cacheKey = keyGenerator 
        ? keyGenerator(...args)
        : `${context}:${JSON.stringify(args)}`;
      
      // 检查缓存
      const cached = cache.get(cacheKey);
      if (cached && cached.expiry > Date.now()) {
        logger.debug(`方法 ${context} 命中缓存`);
        return cached.value;
      }
      
      try {
        // 执行原方法
        const result = await originalMethod.apply(this, args);
        
        // 缓存结果
        cache.set(cacheKey, {
          value: result,
          expiry: Date.now() + ttl
        });
        
        logger.debug(`方法 ${context} 结果已缓存`);
        return result;
      } catch (error) {
        logger.error(`方法 ${context} 执行失败，无法缓存结果`, error);
        throw error;
      }
    };
    
    return descriptor;
  };
}

/**
 * 方法装饰器：限流
 * @param maxCalls 最大调用次数
 * @param timeWindow 时间窗口（毫秒）
 */
export function withRateLimit(maxCalls: number = 10, timeWindow: number = 60000) {
  const callHistory = new Map<string, number[]>();
  
  return function (target: any, propertyName: string, descriptor: PropertyDescriptor) {
    const originalMethod = descriptor.value;
    const context = `${target.constructor.name}.${propertyName}`;
    
    descriptor.value = function (...args: any[]) {
      const now = Date.now();
      const calls = callHistory.get(context) || [];
      
      // 清理过期的调用记录
      const validCalls = calls.filter(time => now - time < timeWindow);
      
      // 检查是否超过限制
      if (validCalls.length >= maxCalls) {
        const error = new Error(`方法 ${context} 调用频率过高，已被限流`);
        ErrorHandler.handle(error, context, ErrorType.SYSTEM, ErrorLevel.MEDIUM);
        throw error;
      }
      
      // 记录本次调用
      validCalls.push(now);
      callHistory.set(context, validCalls);
      
      return originalMethod.apply(this, args);
    };
    
    return descriptor;
  };
}

/**
 * 类装饰器：为类的所有方法添加错误处理
 * @param options 错误处理选项
 */
export function withClassErrorHandling(options: ErrorHandlingOptions = {}) {
  return function <T extends { new(...args: any[]): {} }>(constructor: T) {
    const prototype = constructor.prototype;
    const propertyNames = Object.getOwnPropertyNames(prototype);
    
    for (const propertyName of propertyNames) {
      if (propertyName === 'constructor') continue;
      
      const descriptor = Object.getOwnPropertyDescriptor(prototype, propertyName);
      if (descriptor && typeof descriptor.value === 'function') {
        const errorHandler = withErrorHandling({
          ...options,
          context: options.context || `${constructor.name}.${propertyName}`
        });
        
        errorHandler(prototype, propertyName, descriptor);
        Object.defineProperty(prototype, propertyName, descriptor);
      }
    }
    
    return constructor;
  };
}

/**
 * 工具函数：手动应用错误处理
 * @param fn 要包装的函数
 * @param options 错误处理选项
 */
export function wrapWithErrorHandling<T extends (...args: any[]) => any>(
  fn: T,
  options: ErrorHandlingOptions = {}
): T {
  const context = options.context || fn.name || 'anonymous';
  
  return (async (...args: any[]) => {
    try {
      return await fn(...args);
    } catch (error) {
      if (options.log !== false) {
        logger.error(`函数 ${context} 执行失败`, error);
      }
      
      if (options.customHandler) {
        return options.customHandler(error, context);
      }
      
      const standardError = ErrorHandler.handle(
        error,
        context,
        options.type || ErrorType.SYSTEM,
        options.level || ErrorLevel.MEDIUM
      );
      
      if (options.rethrow !== false) {
        throw standardError;
      }
      
      return options.defaultValue;
    }
  }) as T;
}
