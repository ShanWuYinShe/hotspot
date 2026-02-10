/**
 * 事件总线系统
 * 提供松耦合的服务间通信机制
 */

import { BaseSingleton } from '../base/Singleton';
import { createLogger } from '../utils/LogManager';
import { ErrorHandler, ErrorType, ErrorLevel } from '../utils/ErrorHandler';
import { TimerManager } from '../utils/TimerManager';

const logger = createLogger('EventBus');

/**
 * 事件接口
 */
export interface Event {
  type: string;
  data?: Object;
  timestamp: number;
  source?: string;
  id: string;
}

/**
 * 事件数据接口
 */
interface EventWithData<T extends Object = Object> {
  type: string;
  data: T;
  timestamp: number;
  source?: string;
  id: string;
}

/**
 * 带数据的事件实现类
 */
class EventWithDataImpl<T extends Object = Object> implements EventWithData<T> {
  type: string;
  data: T;
  timestamp: number;
  source?: string;
  id: string;

  constructor(id: string, type: string, timestamp: number, data: T, source?: string) {
    this.id = id;
    this.type = type;
    this.timestamp = timestamp;
    this.data = data;
    this.source = source;
  }
}

/**
 * 空数据对象类
 */
class EmptyDataObject implements Object {
  constructor() {
    // 空实现
  }
}

/**
 * 事件监听器类型
 */
export type EventListener<T extends Object = Object> = (event: EventWithData<T>) => void | Promise<void>;

/**
 * 事件过滤器类型
 */
export type EventFilter = (event: Event) => boolean;

/**
 * 事件监听器配置
 */
interface ListenerConfig {
  once?: boolean;           // 是否只执行一次
  priority?: number;        // 优先级（数字越大优先级越高）
  filter?: EventFilter;     // 事件过滤器
  async?: boolean;          // 是否异步执行
  timeout?: number;         // 超时时间（毫秒）
}

/**
 * 内部监听器信息
 */
interface ListenerInfo {
  id: string;
  listener: EventListener;
  config: ListenerConfig;
  addedAt: number;
  executionCount: number;
  lastExecuted?: number;
}

/**
 * 事件统计信息
 */
export interface EventStats {
  totalEvents: number;
  totalListeners: number;
  eventsByType: Record<string, number>;
  listenersByType: Record<string, number>;
  averageExecutionTime: number;
  errorCount: number;
}

/**
 * 事件总线实现
 */
export class EventBus extends BaseSingleton {
  private static instance: EventBus | null = null;
  private listeners: Map<string, ListenerInfo[]> = new Map();
  private eventHistory: Event[] = [];
  private stats: EventStats;
  private maxHistorySize: number = 1000;
  private nextListenerId: number = 1;
  private cleanupTimerId: number = -1;

  constructor() {
    super();
    const initialStats: EventStats = {
      totalEvents: 0,
      totalListeners: 0,
      eventsByType: {},
      listenersByType: {},
      averageExecutionTime: 0,
      errorCount: 0
    };
    this.stats = initialStats;
  }

  /**
   * 获取单例实例
   */
  static getInstance(): EventBus {
    if (!EventBus.instance) {
      EventBus.instance = new EventBus();
    }
    return EventBus.instance;
  }

  /**
   * 异步获取单例实例
   */
  static async getInstanceAsync(): Promise<EventBus> {
    const instance = EventBus.getInstance();
    await instance.initialize();
    return instance;
  }

  /**
   * 实现基类的初始化方法
   */
  protected async onInitialize(): Promise<void> {
    try {
      logger.info('事件总线初始化开始');
      
      // 启动清理任务
      this.startCleanupTask();
      
      logger.info('事件总线初始化完成');
    } catch (error) {
      ErrorHandler.handle(error, 'EventBus.onInitialize', ErrorType.SYSTEM, ErrorLevel.HIGH);
      throw new Error(String(error));
    }
  }

  /**
   * 实现基类的清理方法
   */
  protected async onCleanup(): Promise<void> {
    try {
      // 停止清理任务
      this.stopCleanupTask();
      
      // 清理所有监听器
      this.removeAllListeners();
      
      // 清理事件历史
      this.eventHistory = [];
      EventBus.instance = null;
      
      logger.info('事件总线清理完成');
    } catch (error) {
      ErrorHandler.handle(error, 'EventBus.onCleanup', ErrorType.SYSTEM);
    }
  }

  /**
   * 发布事件
   */
  emit<T extends Object = Object>(type: string, data?: T, source?: string): string {
    const event: Event = {
      type,
      data: data ? data as Object : undefined,
      timestamp: Date.now(),
      source,
      id: this.generateEventId()
    };

    return this.emitEvent(event);
  }

  /**
   * 发布事件对象
   */
  emitEvent(event: Event): string {
    try {
      // 更新统计信息
      this.updateStatsForEmit(event);

      // 添加到历史记录
      this.addToHistory(event);

      // 获取监听器
      const listeners = this.getListenersForEvent(event);

      if (listeners.length === 0) {
        logger.debug(`没有监听器处理事件: ${event.type}`);
        return event.id;
      }

      // 按优先级排序
      listeners.sort((a, b) => (b.config.priority || 0) - (a.config.priority || 0));

      // 执行监听器
      this.executeListeners(event, listeners);

      logger.debug(`事件已发布: ${event.type}, 监听器数量: ${listeners.length}`);
      return event.id;

    } catch (error) {
      ErrorHandler.handle(error, `emitEvent(${event.type})`, ErrorType.SYSTEM, ErrorLevel.MEDIUM);
      return event.id;
    }
  }

  /**
   * 添加事件监听器
   */
  on<T extends Object = Object>(
    type: string, 
    listener: EventListener<T>, 
    config: ListenerConfig = {}
  ): string {
    const listenerId = this.generateListenerId();
    
    const listenerInfo: ListenerInfo = {
      id: listenerId,
      listener: listener as EventListener,
      config: {
        once: config.once,
        priority: config.priority,
        filter: config.filter,
        async: config.async,
        timeout: config.timeout
      },
      addedAt: Date.now(),
      executionCount: 0
    };

    // 添加到监听器列表
    if (!this.listeners.has(type)) {
      this.listeners.set(type, []);
    }
    this.listeners.get(type)!.push(listenerInfo);

    // 更新统计信息
    this.updateStatsForListener(type, 1);

    logger.debug(`添加监听器: ${type}, ID: ${listenerId}`);
    return listenerId;
  }

  /**
   * 添加一次性监听器
   */
  once<T extends Object = Object>(
    type: string, 
    listener: EventListener<T>, 
    config: ListenerConfig = {}
  ): string {
    const onceConfig: ListenerConfig = {
      once: true,
      priority: config.priority,
      filter: config.filter,
      async: config.async,
      timeout: config.timeout
    };
    return this.on(type, listener, onceConfig);
  }

  /**
   * 移除监听器
   */
  off(type: string, listenerId?: string): boolean {
    const listeners = this.listeners.get(type);
    if (!listeners) {
      return false;
    }

    if (listenerId) {
      // 移除特定监听器
      const index = listeners.findIndex(l => l.id === listenerId);
      if (index !== -1) {
        listeners.splice(index, 1);
        this.updateStatsForListener(type, -1);
        logger.debug(`移除监听器: ${type}, ID: ${listenerId}`);
        
        // 如果没有监听器了，删除整个类型
        if (listeners.length === 0) {
          this.listeners.delete(type);
        }
        return true;
      }
    } else {
      // 移除所有该类型的监听器
      const count = listeners.length;
      this.listeners.delete(type);
      this.updateStatsForListener(type, -count);
      logger.debug(`移除所有监听器: ${type}, 数量: ${count}`);
      return count > 0;
    }

    return false;
  }

  /**
   * 移除所有监听器
   */
  removeAllListeners(): void {
    const totalListeners = Array.from(this.listeners.values())
      .reduce((sum, listeners) => sum + listeners.length, 0);
    
    this.listeners.clear();
    this.stats.totalListeners = 0;
    this.stats.listenersByType = {};
    
    logger.info(`移除所有监听器，共 ${totalListeners} 个`);
  }

  /**
   * 获取监听器数量
   */
  listenerCount(type?: string): number {
    if (type) {
      return this.listeners.get(type)?.length || 0;
    }
    return Array.from(this.listeners.values())
      .reduce((sum, listeners) => sum + listeners.length, 0);
  }

  /**
   * 获取事件类型列表
   */
  eventTypes(): string[] {
    return Array.from(this.listeners.keys());
  }

  /**
   * 等待特定事件
   */
  waitFor<T extends Object = Object>(
    type: string, 
    timeout: number = 5000, 
    filter?: EventFilter
  ): Promise<EventWithData<T>> {
    return new Promise<EventWithData<T>>((resolve, reject) => {
      const timeoutId = TimerManager.setTimeout(() => {
        this.off(type, listenerId);
        reject(new Error(`等待事件超时: ${type}`));
      }, timeout, `等待事件: ${type}`);

      const listenerId = this.once(type, (event) => {
        TimerManager.clearTimeout(timeoutId);
        resolve(event as EventWithData<T>);
      }, { filter });
    });
  }

  /**
   * 获取事件历史
   */
  getHistory(type?: string, limit?: number): Event[] {
    let history = type 
      ? this.eventHistory.filter(e => e.type === type)
      : this.eventHistory;
    
    if (limit) {
      history = history.slice(-limit);
    }
    
    const result: Event[] = [];
    for (const event of history) {
      result.push(event);
    }
    return result;
  }

  /**
   * 清理事件历史
   */
  clearHistory(type?: string): void {
    if (type) {
      this.eventHistory = this.eventHistory.filter(e => e.type !== type);
    } else {
      this.eventHistory = [];
    }
    logger.debug(`清理事件历史: ${type || '全部'}`);
  }

  /**
   * 获取统计信息
   */
  getStats(): EventStats {
    const statsResult: EventStats = {
      totalEvents: this.stats.totalEvents,
      totalListeners: this.stats.totalListeners,
      eventsByType: this.copyRecord(this.stats.eventsByType),
      listenersByType: this.copyRecord(this.stats.listenersByType),
      averageExecutionTime: this.stats.averageExecutionTime,
      errorCount: this.stats.errorCount
    };
    return statsResult;
  }

  /**
   * 重置统计信息
   */
  resetStats(): void {
    const newStats: EventStats = {
      totalEvents: 0,
      totalListeners: this.listenerCount(),
      eventsByType: {},
      listenersByType: this.copyRecord(this.stats.listenersByType),
      averageExecutionTime: 0,
      errorCount: 0
    };
    this.stats = newStats;
    logger.debug('统计信息已重置');
  }

  /**
   * 生成事件ID
   */
  private generateEventId(): string {
    return `event_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
  }

  /**
   * 生成监听器ID
   */
  private generateListenerId(): string {
    return `listener_${this.nextListenerId++}`;
  }

  /**
   * 将Event转换为EventWithData类型
   */
  private convertToEventWithData(event: Event): EventWithData<Object> {
    return new EventWithDataImpl<Object>(
      event.id || this.generateEventId(),
      event.type,
      event.timestamp,
      event.data || new EmptyDataObject(),
      event.source
    );
  }

  /**
   * 获取事件的监听器
   */
  private getListenersForEvent(event: Event): ListenerInfo[] {
    const listeners = this.listeners.get(event.type) || [];
    
    return listeners.filter(listenerInfo => {
      // 应用过滤器
      if (listenerInfo.config.filter && !listenerInfo.config.filter(event)) {
        return false;
      }
      return true;
    });
  }

  /**
   * 执行监听器
   */
  private executeListeners(event: Event, listeners: ListenerInfo[]): void {
    const startTime = Date.now();
    
    listeners.forEach(listenerInfo => {
      this.executeListener(event, listenerInfo);
    });

    // 更新平均执行时间
    const executionTime = Date.now() - startTime;
    this.updateAverageExecutionTime(executionTime);
  }

  /**
   * 复制Record对象
   */
  private copyRecord(source: Record<string, number>): Record<string, number> {
    const result: Record<string, number> = {};
    const keys = Object.keys(source);
    for (const key of keys) {
      result[key] = source[key];
    }
    return result;
  }

  /**
   * 执行单个监听器
   */
  private executeListener(event: Event, listenerInfo: ListenerInfo): void {
    try {
      const listener = listenerInfo.listener;
      const config = listenerInfo.config;
      
      // 更新执行信息
      listenerInfo.executionCount++;
      listenerInfo.lastExecuted = Date.now();

      if (config.async) {
        // 异步执行
        this.executeAsyncListener(event, listenerInfo);
      } else {
        // 同步执行
        const eventWithData = this.convertToEventWithData(event);
        const result = listener(eventWithData);
        if (result instanceof Promise) {
          result.catch((error: Error) => {
            ErrorHandler.handle(error, `listener(${event.type})`, ErrorType.SYSTEM, ErrorLevel.LOW);
            this.stats.errorCount++;
          });
        }
      }

      // 如果是一次性监听器，移除它
      if (config.once) {
        this.off(event.type, listenerInfo.id);
      }

    } catch (error) {
      ErrorHandler.handle(error, `executeListener(${event.type})`, ErrorType.SYSTEM, ErrorLevel.LOW);
      this.stats.errorCount++;
    }
  }

  /**
   * 异步执行监听器
   */
  private async executeAsyncListener(event: Event, listenerInfo: ListenerInfo): Promise<void> {
    const listener = listenerInfo.listener;
    const config = listenerInfo.config;
    
    try {
      const eventWithData = this.convertToEventWithData(event);
      const promise = Promise.resolve(listener(eventWithData));
      
      if (config.timeout) {
        // 带超时的执行
        const timeoutPromise = new Promise<void>((_, reject) => {
          TimerManager.setTimeout(() => {
            reject(new Error(`监听器执行超时: ${event.type}`));
          }, config.timeout!, `监听器超时: ${event.type}`);
        });
        
        await Promise.race([promise, timeoutPromise]);
      } else {
        await promise;
      }
      
    } catch (error) {
      ErrorHandler.handle(error, `asyncListener(${event.type})`, ErrorType.SYSTEM, ErrorLevel.LOW);
      this.stats.errorCount++;
    }
  }

  /**
   * 添加到历史记录
   */
  private addToHistory(event: Event): void {
    this.eventHistory.push(event);
    
    // 保持历史记录在限制范围内
    if (this.eventHistory.length > this.maxHistorySize) {
      this.eventHistory = this.eventHistory.slice(-this.maxHistorySize);
    }
  }

  /**
   * 更新发布事件的统计信息
   */
  private updateStatsForEmit(event: Event): void {
    this.stats.totalEvents++;
    this.stats.eventsByType[event.type] = (this.stats.eventsByType[event.type] || 0) + 1;
  }

  /**
   * 更新监听器的统计信息
   */
  private updateStatsForListener(type: string, delta: number): void {
    this.stats.totalListeners += delta;
    this.stats.listenersByType[type] = Math.max(0, (this.stats.listenersByType[type] || 0) + delta);
    
    // 如果监听器数量为0，删除该类型
    if (this.stats.listenersByType[type] === 0) {
      const newListenersByType: Record<string, number> = {};
      const keys = Object.keys(this.stats.listenersByType);
      for (const key of keys) {
        if (key !== type) {
          newListenersByType[key] = this.stats.listenersByType[key];
        }
      }
      this.stats.listenersByType = newListenersByType;
    }
  }

  /**
   * 更新平均执行时间
   */
  private updateAverageExecutionTime(executionTime: number): void {
    if (this.stats.totalEvents === 1) {
      this.stats.averageExecutionTime = executionTime;
    } else {
      // 使用移动平均
      this.stats.averageExecutionTime = 
        (this.stats.averageExecutionTime * 0.9) + (executionTime * 0.1);
    }
  }

  /**
   * 启动清理任务
   */
  private startCleanupTask(): void {
    // 每10分钟清理一次过期的监听器和历史记录
    this.cleanupTimerId = TimerManager.setInterval(() => {
      this.performCleanup();
    }, 10 * 60 * 1000, '事件总线清理');
  }

  /**
   * 停止清理任务
   */
  private stopCleanupTask(): void {
    if (this.cleanupTimerId !== -1) {
      TimerManager.clearInterval(this.cleanupTimerId);
      this.cleanupTimerId = -1;
    }
  }

  /**
   * 执行清理
   */
  private performCleanup(): void {
    const now = Date.now();
    const maxAge = 60 * 60 * 1000; // 1小时
    let cleanedListeners = 0;

    // 清理长时间未执行的一次性监听器
    const listenerEntries = Array.from(this.listeners.entries());
    for (const entry of listenerEntries) {
      const type = entry[0];
      const listeners = entry[1];
      const originalLength = listeners.length;
      
      // 保留活跃的监听器
      const activeListeners = listeners.filter(listener => {
        if (listener.config.once && listener.executionCount === 0) {
          // 一次性监听器超过1小时未执行，可能是泄漏
          if (now - listener.addedAt > maxAge) {
            return false;
          }
        }
        return true;
      });
      
      if (activeListeners.length !== originalLength) {
        this.listeners.set(type, activeListeners);
        cleanedListeners += originalLength - activeListeners.length;
        
        if (activeListeners.length === 0) {
          this.listeners.delete(type);
        }
      }
    }

    if (cleanedListeners > 0) {
      logger.info(`清理了 ${cleanedListeners} 个过期监听器`);
    }
  }
}

/**
 * 便捷函数：获取事件总线实例
 */
export function getEventBus(): EventBus {
  return EventBus.getInstance();
}

/**
 * 便捷函数：发布事件
 */
export function emit<T extends Object = Object>(type: string, data?: T, source?: string): string {
  return getEventBus().emit(type, data, source);
}

/**
 * 便捷函数：监听事件
 */
export function on<T extends Object = Object>(
  type: string, 
  listener: EventListener<T>, 
  config?: ListenerConfig
): string {
  return getEventBus().on(type, listener, config);
}

/**
 * 便捷函数：一次性监听
 */
export function once<T extends Object = Object>(
  type: string, 
  listener: EventListener<T>, 
  config?: ListenerConfig
): string {
  return getEventBus().once(type, listener, config);
}

/**
 * 便捷函数：移除监听器
 */
export function off(type: string, listenerId?: string): boolean {
  return getEventBus().off(type, listenerId);
}
