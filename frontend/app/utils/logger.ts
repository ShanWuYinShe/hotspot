/**
 * 统一的日志系统
 * 在开发环境显示详细日志，生产环境只记录错误
 */

// Sentry 类型声明
declare global {
  interface Window {
    Sentry?: {
      captureException: (error: Error, options?: any) => void
    }
  }
}

export enum LogLevel {
  DEBUG = 0,
  INFO = 1,
  WARN = 2,
  ERROR = 3
}

interface LogEntry {
  level: LogLevel
  message: string
  context?: string
  data?: any
  timestamp: Date
}

class Logger {
  private isDev = import.meta.dev
  private minLevel = this.isDev ? LogLevel.DEBUG : LogLevel.ERROR

  private formatMessage(entry: LogEntry): string {
    const timestamp = entry.timestamp.toISOString()
    const levelName = LogLevel[entry.level]
    const context = entry.context ? `[${entry.context}]` : ''
    return `${timestamp} ${levelName} ${context} ${entry.message}`
  }

  private shouldLog(level: LogLevel): boolean {
    return level >= this.minLevel
  }

  private log(level: LogLevel, message: string, context?: string, data?: any) {
    if (!this.shouldLog(level)) return

    const entry: LogEntry = {
      level,
      message,
      context,
      data,
      timestamp: new Date()
    }

    const formattedMessage = this.formatMessage(entry)

    switch (level) {
      case LogLevel.DEBUG:
        if (this.isDev) {
          console.debug(formattedMessage, data)
        }
        break
      case LogLevel.INFO:
        if (this.isDev) {
          console.info(formattedMessage, data)
        }
        break
      case LogLevel.WARN:
        console.warn(formattedMessage, data)
        break
      case LogLevel.ERROR:
        console.error(formattedMessage, data)
        // 在生产环境可以发送到错误监控服务
        if (!this.isDev) {
          this.sendToErrorService(entry)
        }
        break
    }
  }

  private sendToErrorService(entry: LogEntry) {
    // 集成错误监控服务
    if (process.client && window.Sentry) {
      window.Sentry.captureException(new Error(entry.message), {
        level: entry.level,
        contexts: {
          logger: {
            context: entry.context,
            timestamp: entry.timestamp,
            data: entry.data
          }
        }
      })
    }
  }

  debug(message: string, context?: string, data?: any) {
    this.log(LogLevel.DEBUG, message, context, data)
  }

  info(message: string, context?: string, data?: any) {
    this.log(LogLevel.INFO, message, context, data)
  }

  warn(message: string, context?: string, data?: any) {
    this.log(LogLevel.WARN, message, context, data)
  }

  error(message: string, context?: string, data?: any) {
    this.log(LogLevel.ERROR, message, context, data)
  }

  // 便捷方法
  api(message: string, data?: any) {
    this.debug(message, 'API', data)
  }

  store(message: string, data?: any) {
    this.debug(message, 'Store', data)
  }

  component(message: string, component: string, data?: any) {
    this.debug(message, `Component:${component}`, data)
  }

  service(message: string, service: string, data?: any) {
    this.debug(message, `Service:${service}`, data)
  }
}

// 导出单例实例
export const logger = new Logger()

// 导出便捷方法
export const { debug, info, warn, error, api, store, component, service } = logger

