/**
 * 统一日志管理器
 * 提供统一的日志记录接口，避免重复的DOMAIN_NUMBER和TAG定义
 */

import { hilog } from '@kit.PerformanceAnalysisKit';

/**
 * 日志级别枚举
 */
export enum LogLevel {
  DEBUG = 0,
  INFO = 1,
  WARN = 2,
  ERROR = 3
}

/**
 * 日志配置接口
 */
interface LogConfig {
  level: LogLevel;
  enableConsole: boolean;
  enableFile: boolean;
  maxLogSize: number;
}

/**
 * 日志记录器接口
 */
export interface Logger {
  debug(message: string, ...args: Object[]): void;
  info(message: string, ...args: Object[]): void;
  warn(message: string, ...args: Object[]): void;
  error(message: string, ...args: Object[]): void;
  fatal(message: string, ...args: Object[]): void;
}

/**
 * 日志管理器实现
 */
export class LogManager {
  private static readonly DOMAIN_BASE = 0xFF00;
  private static domainCounter = 0;
  private static loggers: Map<string, Logger> = new Map();
  private static config: LogConfig = {
    level: LogLevel.DEBUG,
    enableConsole: true,
    enableFile: false,
    maxLogSize: 10 * 1024 * 1024 // 10MB
  };

  /**
   * 获取或创建日志记录器
   */
  static getLogger(moduleName: string): Logger {
    if (!LogManager.loggers.has(moduleName)) {
      const domain = LogManager.DOMAIN_BASE + (++LogManager.domainCounter);
      const logger = new HiLogLogger(domain, moduleName, LogManager.config);
      LogManager.loggers.set(moduleName, logger);
    }
    return LogManager.loggers.get(moduleName)!;
  }

  /**
   * 设置全局日志配置
   */
  static setConfig(config: Partial<LogConfig>): void {
    // 手动合并配置对象，避免使用展开语法
    const newConfig: LogConfig = {
      level: config.level !== undefined ? config.level : LogManager.config.level,
      enableConsole: config.enableConsole !== undefined ? config.enableConsole : LogManager.config.enableConsole,
      enableFile: config.enableFile !== undefined ? config.enableFile : LogManager.config.enableFile,
      maxLogSize: config.maxLogSize !== undefined ? config.maxLogSize : LogManager.config.maxLogSize
    };
    LogManager.config = newConfig;
    
    // 更新所有已存在的日志记录器
    LogManager.loggers.forEach((logger) => {
      if (logger instanceof HiLogLogger) {
        logger.updateConfig(LogManager.config);
      }
    });
  }

  /**
   * 获取当前日志配置
   */
  static getConfig(): LogConfig {
    // 手动创建配置副本，避免使用展开语法
    return {
      level: LogManager.config.level,
      enableConsole: LogManager.config.enableConsole,
      enableFile: LogManager.config.enableFile,
      maxLogSize: LogManager.config.maxLogSize
    };
  }

  /**
   * 清理所有日志记录器
   */
  static cleanup(): void {
    LogManager.loggers.clear();
    LogManager.domainCounter = 0;
  }

  /**
   * 格式化敏感信息
   */
  static sanitizeData(data: Object): Object {
    if (typeof data !== 'object' || data === null) {
      return data;
    }

    const sensitiveKeys = [
      'token', 'password', 'secret', 'key', 'auth', 'authorization',
      'credential', 'session', 'cookie', 'jwt', 'apikey'
    ];

    // 手动创建副本，避免使用展开语法
    let sanitized: Record<string, Object>;
    if (Array.isArray(data)) {
      sanitized = {};
      for (let i = 0; i < data.length; i++) {
        sanitized[i.toString()] = data[i];
      }
    } else {
      sanitized = {};
      const keys = Object.keys(data);
      for (let i = 0; i < keys.length; i++) {
        const key = keys[i];
        sanitized[key] = (data as Record<string, Object>)[key];
      }
    }

    // 使用传统for循环替代for...in
    const keys = Object.keys(sanitized);
    for (let i = 0; i < keys.length; i++) {
      const key = keys[i];
      const lowerKey = key.toLowerCase();
      if (sensitiveKeys.some(sensitive => lowerKey.includes(sensitive))) {
        const value: Object = sanitized[key];
        if (typeof value === 'string' && value.length > 0) {
          sanitized[key] = value.length > 10 
            ? `${value.substring(0, 4)}****${value.substring(value.length - 4)}`
            : '****';
        } else {
          sanitized[key] = '****';
        }
      } else if (typeof sanitized[key] === 'object' && sanitized[key] !== null) {
        sanitized[key] = LogManager.sanitizeData(sanitized[key]);
      }
    }

    return sanitized;
  }
}

/**
 * HiLog日志记录器实现
 */
class HiLogLogger implements Logger {
  private domain: number;
  private tag: string;
  private config: LogConfig;

  constructor(domain: number, tag: string, config: LogConfig) {
    this.domain = domain;
    this.tag = tag;
    this.config = config;
  }

  updateConfig(config: LogConfig): void {
    this.config = config;
  }

  debug(message: string, ...args: Object[]): void {
    if (this.config.level <= LogLevel.DEBUG && this.config.enableConsole) {
      const formattedMessage = this.formatMessage(message, args);
      hilog.debug(this.domain, this.tag, formattedMessage);
    }
  }

  info(message: string, ...args: Object[]): void {
    if (this.config.level <= LogLevel.INFO && this.config.enableConsole) {
      const formattedMessage = this.formatMessage(message, args);
      hilog.info(this.domain, this.tag, formattedMessage);
    }
  }

  warn(message: string, ...args: Object[]): void {
    if (this.config.level <= LogLevel.WARN && this.config.enableConsole) {
      const formattedMessage = this.formatMessage(message, args);
      hilog.warn(this.domain, this.tag, formattedMessage);
    }
  }

  error(message: string, ...args: Object[]): void {
    if (this.config.level <= LogLevel.ERROR && this.config.enableConsole) {
      const formattedMessage = this.formatMessage(message, args);
      hilog.error(this.domain, this.tag, formattedMessage);
    }
  }

  fatal(message: string, ...args: Object[]): void {
    if (this.config.enableConsole) {
      const formattedMessage = this.formatMessage(message, args);
      hilog.fatal(this.domain, this.tag, formattedMessage);
    }
  }

  private formatMessage(message: string, args: Object[]): string {
    if (args.length === 0) {
      return message;
    }

    // 处理参数，对敏感数据进行脱敏
    const sanitizedArgs = args.map(arg => {
      if (typeof arg === 'object') {
        return JSON.stringify(LogManager.sanitizeData(arg));
      }
      return String(arg);
    });

    // 简单的字符串格式化
    let formattedMessage = message;
    sanitizedArgs.forEach((arg, index) => {
      formattedMessage = formattedMessage.replace(`{${index}}`, arg);
    });

    // 如果没有占位符，直接拼接参数
    if (!message.includes('{') && sanitizedArgs.length > 0) {
      formattedMessage = `${message} ${sanitizedArgs.join(' ')}`;
    }

    return formattedMessage;
  }
}

/**
 * 便捷的日志记录函数
 */
export function createLogger(moduleName: string): Logger {
  return LogManager.getLogger(moduleName);
}

/**
 * 全局日志记录器（用于快速使用）
 */
export const globalLogger = LogManager.getLogger('Global');
