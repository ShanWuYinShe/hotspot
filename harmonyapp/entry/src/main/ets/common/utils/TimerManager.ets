/**
 * 定时器管理器
 * 统一管理所有定时器，防止内存泄漏
 */

import { createLogger } from './LogManager';

const logger = createLogger('TimerManager');

/**
 * 定时器信息接口
 */
interface TimerInfo {
  id: number;
  type: 'interval' | 'timeout';
  callback: () => void;
  delay: number;
  createdAt: number;
  lastExecuted?: number;
  executionCount: number;
  description?: string;
}

/**
 * 定时器统计信息
 */
export interface TimerStats {
  totalTimers: number;
  intervalTimers: number;
  timeoutTimers: number;
  oldestTimer: number;
  averageDelay: number;
}

/**
 * 定时器管理器
 */
export class TimerManager {
  private static timers: Map<number, TimerInfo> = new Map();
  private static nextId: number = 1;

  /**
   * 创建定时器（setInterval的替代）
   */
  static setInterval(
    callback: () => void, 
    delay: number, 
    description?: string
  ): number {
    const id = TimerManager.nextId++;
    const systemId = setInterval(() => {
      const timerInfo = TimerManager.timers.get(id);
      if (timerInfo) {
        timerInfo.lastExecuted = Date.now();
        timerInfo.executionCount++;
        try {
          callback();
        } catch (error) {
          logger.error(`定时器执行出错 [${id}]: ${error}`);
        }
      }
    }, delay);

    const timerInfo: TimerInfo = {
      id: systemId,
      type: 'interval',
      callback,
      delay,
      createdAt: Date.now(),
      executionCount: 0,
      description
    };

    TimerManager.timers.set(id, timerInfo);
    logger.debug(`创建定时器 [${id}]: ${description || '未命名'}, 间隔: ${delay}ms`);
    
    return id;
  }

  /**
   * 创建超时定时器（setTimeout的替代）
   */
  static setTimeout(
    callback: () => void, 
    delay: number, 
    description?: string
  ): number {
    const id = TimerManager.nextId++;
    const systemId = setTimeout(() => {
      const timerInfo = TimerManager.timers.get(id);
      if (timerInfo) {
        timerInfo.lastExecuted = Date.now();
        timerInfo.executionCount++;
        try {
          callback();
        } catch (error) {
          logger.error(`超时定时器执行出错 [${id}]: ${error}`);
        } finally {
          // 超时定时器执行后自动清理
          TimerManager.timers.delete(id);
        }
      }
    }, delay);

    const timerInfo: TimerInfo = {
      id: systemId,
      type: 'timeout',
      callback,
      delay,
      createdAt: Date.now(),
      executionCount: 0,
      description
    };

    TimerManager.timers.set(id, timerInfo);
    logger.debug(`创建超时定时器 [${id}]: ${description || '未命名'}, 延迟: ${delay}ms`);
    
    return id;
  }

  /**
   * 清除定时器
   */
  static clearInterval(id: number): void {
    const timerInfo = TimerManager.timers.get(id);
    if (timerInfo) {
      if (timerInfo.type === 'interval') {
        clearInterval(timerInfo.id);
      } else {
        clearTimeout(timerInfo.id);
      }
      TimerManager.timers.delete(id);
      logger.debug(`清除定时器 [${id}]: ${timerInfo.description || '未命名'}`);
    } else {
      logger.warn(`尝试清除不存在的定时器 [${id}]`);
    }
  }

  /**
   * 清除超时定时器（setTimeout的替代）
   */
  static clearTimeout(id: number): void {
    TimerManager.clearInterval(id); // 内部实现相同
  }

  /**
   * 清除所有定时器
   */
  static clearAll(): void {
    const count = TimerManager.timers.size;
    TimerManager.timers.forEach((timerInfo, id) => {
      if (timerInfo.type === 'interval') {
        clearInterval(timerInfo.id);
      } else {
        clearTimeout(timerInfo.id);
      }
    });
    TimerManager.timers.clear();
    logger.info(`清除了所有定时器，共 ${count} 个`);
  }

  /**
   * 获取定时器统计信息
   */
  static getStats(): TimerStats {
    const timers = Array.from(TimerManager.timers.values());
    const intervalTimers = timers.filter(t => t.type === 'interval').length;
    const timeoutTimers = timers.filter(t => t.type === 'timeout').length;
    
    const now = Date.now();
    const oldestTimer = timers.length > 0 
      ? Math.min(...timers.map(t => now - t.createdAt))
      : 0;
    
    const averageDelay = timers.length > 0
      ? timers.reduce((sum, t) => sum + t.delay, 0) / timers.length
      : 0;

    return {
      totalTimers: timers.length,
      intervalTimers,
      timeoutTimers,
      oldestTimer,
      averageDelay
    };
  }

  /**
   * 获取所有定时器信息（用于调试）
   */
  static getAllTimers(): TimerInfo[] {
    return Array.from(TimerManager.timers.values());
  }

  /**
   * 检查长时间运行的定时器
   */
  static checkLongRunningTimers(thresholdMs: number = 60000): TimerInfo[] {
    const now = Date.now();
    const longRunningTimers = Array.from(TimerManager.timers.values())
      .filter(timer => now - timer.createdAt > thresholdMs);
    
    if (longRunningTimers.length > 0) {
      logger.warn(`发现 ${longRunningTimers.length} 个长时间运行的定时器`);
      longRunningTimers.forEach(timer => {
        const runningTime = now - timer.createdAt;
        logger.warn(`长时间运行定时器: ${timer.description || '未命名'}, 运行时间: ${runningTime}ms, 执行次数: ${timer.executionCount}`);
      });
    }
    
    return longRunningTimers;
  }

  /**
   * 定期检查和清理（建议在应用生命周期中调用）
   */
  static startHealthCheck(intervalMs: number = 300000): number { // 5分钟检查一次
    return TimerManager.setInterval(() => {
      const stats = TimerManager.getStats();
      logger.info(`定时器健康检查: 总计 ${stats.totalTimers} 个定时器`);
      
      // 检查长时间运行的定时器
      TimerManager.checkLongRunningTimers();
      
      // 如果定时器过多，发出警告
      if (stats.totalTimers > 50) {
        logger.warn(`定时器数量过多: ${stats.totalTimers} 个，请检查是否存在内存泄漏`);
      }
    }, intervalMs, 'TimerManager健康检查');
  }

  /**
   * 应用退出时的清理方法
   */
  static cleanup(): void {
    logger.info('TimerManager开始清理...');
    TimerManager.clearAll();
    logger.info('TimerManager清理完成');
  }
}

/**
 * 便捷的定时器创建函数
 */
export function createInterval(
  callback: () => void, 
  delay: number, 
  description?: string
): number {
  return TimerManager.setInterval(callback, delay, description);
}

export function createTimeout(
  callback: () => void, 
  delay: number, 
  description?: string
): number {
  return TimerManager.setTimeout(callback, delay, description);
}

export function clearTimer(id: number): void {
  TimerManager.clearInterval(id);
}
