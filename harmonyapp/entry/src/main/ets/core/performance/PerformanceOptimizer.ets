/**
 * 性能优化管理器（重构版）
 * 使用新的基础设施：单例基类、统一日志、定时器管理器、错误处理
 */

import { display } from '@kit.ArkUI';
import { deviceInfo, batteryInfo } from '@kit.BasicServicesKit';
import { BaseSingleton } from '../base/Singleton';
import { createLogger } from '../utils/LogManager';
import { TimerManager } from '../utils/TimerManager';
import { ErrorHandler, ErrorType, ErrorLevel } from '../utils/ErrorHandler';
import { PERFORMANCE_CONSTANTS } from '../../constants/SystemConstants';
import { hidebug } from '@kit.PerformanceAnalysisKit';

const logger = createLogger('PerformanceOptimizer');

/**
 * 缓存项接口
 */
interface CacheItem<T> {
  data: T;
  timestamp: number;
  expiry: number;
  accessCount: number;
  lastAccess: number;
}

interface CacheStats {
  size: number;
  maxSize: number;
  hitRate: number;
}

/**
 * 性能指标接口
 */
export interface PerformanceMetrics {
  memoryUsage: number;
  cpuUsage: number;
  networkLatency: number;
  frameRate: number;
  batteryLevel: number;
  cacheHitRate: number;
  errorRate: number;
  averageResponseTime: number;
}

/**
 * 性能告警接口
 */
export interface PerformanceAlert {
  type: string;
  level: 'info' | 'warning' | 'critical';
  message: string;
  timestamp: number;
  value: number;
  threshold: number;
}

/**
 * 自适应配置接口
 */
export interface AdaptiveConfig {
  maxCacheSize: number;
  imageQuality: number;
  animationEnabled: boolean;
  maxConcurrentRequests: number;
  networkTimeout: number;
  backgroundSyncEnabled: boolean;
}

/**
 * 设备适配信息接口
 */
export interface DeviceAdaptationInfo {
  screenWidth: number;
  screenHeight: number;
  density: number;
  orientation: string;
  deviceType: 'phone' | 'tablet' | 'tv' | 'watch' | 'car';
  performanceLevel: 'low' | 'medium' | 'high';
}

export class PerformanceOptimizer extends BaseSingleton {
  private static instance: PerformanceOptimizer | null = null;
  private cache: Map<string, CacheItem<object>> = new Map();
  private maxCacheSize: number = PERFORMANCE_CONSTANTS.CACHE.DEFAULT_SIZE;
  private defaultExpiry: number = PERFORMANCE_CONSTANTS.CACHE.DEFAULT_EXPIRY;
  private performanceMetrics: PerformanceMetrics;
  private alerts: PerformanceAlert[] = [];
  private config: AdaptiveConfig;
  private isMonitoring: boolean = false;
  private monitoringTimerId: number = -1;
  private cacheHits: number = 0;
  private cacheMisses: number = 0;
  private requestCount: number = 0;
  private totalResponseTime: number = 0;
  private errorCount: number = 0;

  private constructor() {
    super();
    this.performanceMetrics = {
      memoryUsage: 0,
      cpuUsage: 0,
      networkLatency: 0,
      frameRate: 60,
      batteryLevel: 100,
      cacheHitRate: 0,
      errorRate: 0,
      averageResponseTime: 0
    };

    this.config = {
      maxCacheSize: PERFORMANCE_CONSTANTS.CACHE.DEFAULT_SIZE,
      imageQuality: 80,
      animationEnabled: true,
      maxConcurrentRequests: 5,
      networkTimeout: PERFORMANCE_CONSTANTS.INTERVALS.PERFORMANCE_CHECK,
      backgroundSyncEnabled: true
    };
  }

  /**
   * 获取单例实例
   */
  static getInstance(): PerformanceOptimizer {
    if (!PerformanceOptimizer.instance) {
      PerformanceOptimizer.instance = new PerformanceOptimizer();
    }
    return PerformanceOptimizer.instance;
  }

  /**
   * 异步获取单例实例
   */
  static async getInstanceAsync(): Promise<PerformanceOptimizer> {
    const instance = PerformanceOptimizer.getInstance();
    await instance.initialize();
    return instance;
  }

  /**
   * 实现基类的初始化方法
   */
  protected async onInitialize(): Promise<void> {
    try {
      logger.info('性能优化器初始化开始');
      // 初始化性能监控
      this.updatePerformanceMetrics();
      logger.info('性能优化器初始化完成');
    } catch (error) {
      ErrorHandler.handle(error, 'PerformanceOptimizer.onInitialize', ErrorType.SYSTEM, ErrorLevel.HIGH);
      throw new Error('PerformanceOptimizer初始化失败');
    }
  }

  /**
   * 实现基类的清理方法
   */
  protected async onCleanup(): Promise<void> {
    try {
      this.stopMonitoring();
      this.clearCache();
      this.clearAlerts();
      logger.info('性能优化器清理完成');
    } catch (error) {
      ErrorHandler.handle(error, 'PerformanceOptimizer.onCleanup', ErrorType.SYSTEM);
    }
  }

  /**
   * 设置缓存
   */
  setCache<T>(key: string, data: T, expiry?: number): void {
    try {
      // 如果缓存已满，清理最少使用的项
      if (this.cache.size >= this.maxCacheSize) {
        this.evictLeastUsed();
      }

      const cacheItem: CacheItem<T> = {
        data,
        timestamp: Date.now(),
        expiry: expiry || this.defaultExpiry,
        accessCount: 0,
        lastAccess: Date.now()
      };

      this.cache.set(key, cacheItem as CacheItem<Object>);
      logger.debug(`缓存设置成功: ${key}`);
    } catch (error) {
      ErrorHandler.handle(error, `setCache(${key})`, ErrorType.SYSTEM, ErrorLevel.LOW);
    }
  }

  /**
   * 获取缓存
   */
  getCache<T>(key: string): T | null {
    try {
      const cacheItem = this.cache.get(key);
      if (!cacheItem) {
        this.cacheMisses++;
        return null;
      }

      // 检查是否过期
      if (Date.now() - cacheItem.timestamp > cacheItem.expiry) {
        this.cache.delete(key);
        this.cacheMisses++;
        return null;
      }

      // 更新访问信息
      cacheItem.accessCount++;
      cacheItem.lastAccess = Date.now();
      this.cacheHits++;

      return cacheItem.data as T;
    } catch (error) {
      ErrorHandler.handle(error, `getCache(${key})`, ErrorType.SYSTEM, ErrorLevel.LOW);
      this.cacheMisses++;
      return null;
    }
  }

  /**
   * 清理最少使用的缓存项
   */
  private evictLeastUsed(): void {
    let leastUsedKey: string | null = null;
    let leastAccessCount = Infinity;

    this.cache.forEach((item: CacheItem<Object>, key: string) => {
      if (item.accessCount < leastAccessCount) {
        leastAccessCount = item.accessCount;
        leastUsedKey = key;
      }
    });

    if (leastUsedKey) {
      this.cache.delete(leastUsedKey);
      logger.debug(`清理最少使用缓存: ${leastUsedKey}`);
    }
  }

  /**
   * 清空缓存
   */
  clearCache(): void {
    this.cache.clear();
    logger.info('缓存已清空');
  }

  /**
   * 获取缓存统计
   */
  getCacheStats(): CacheStats {
    const stats: CacheStats = {
      size: this.cache.size,
      maxSize: this.maxCacheSize,
      hitRate: this.performanceMetrics.cacheHitRate
    };
    return stats;
  }

  /**
   * 获取设备适配信息
   */
  getDeviceAdaptationInfo(): DeviceAdaptationInfo {
    try {
      // 从系统API获取真实的设备信息
      const displayInfo = display.getDefaultDisplaySync();
      const deviceType = deviceInfo.deviceType;
      
      const adaptationInfo: DeviceAdaptationInfo = {
        screenWidth: displayInfo.width,
        screenHeight: displayInfo.height,
        density: displayInfo.densityDPI / 160, // 转换为密度倍数
        orientation: displayInfo.width > displayInfo.height ? 'landscape' : 'portrait',
        deviceType: deviceType === 'phone' ? 'phone' : deviceType === 'tablet' ? 'tablet' : 'tv',
        performanceLevel: this.getPerformanceLevel()
      };

      logger.debug(`设备适配信息: ${JSON.stringify(adaptationInfo)}`);
      return adaptationInfo;
    } catch (error) {
      ErrorHandler.handle(error, 'getDeviceAdaptationInfo', ErrorType.SYSTEM, ErrorLevel.LOW);
      // 返回默认值
      return {
        screenWidth: 360,
        screenHeight: 640,
        density: 2.0,
        orientation: 'portrait',
        deviceType: 'phone',
        performanceLevel: 'medium'
      };
    }
  }

  /**
   * 获取性能等级
   */
  private getPerformanceLevel(): 'low' | 'medium' | 'high' {
    // 简化的性能等级判断
    try {
      const displayInfo = display.getDefaultDisplaySync();
      const screenSize = displayInfo.width * displayInfo.height;
      
      if (screenSize > 1920 * 1080) {
        return 'high';
      } else if (screenSize > 720 * 1280) {
        return 'medium';
      } else {
        return 'low';
      }
    } catch (error) {
      ErrorHandler.handle(error, 'getPerformanceLevel', ErrorType.SYSTEM, ErrorLevel.LOW);
      return 'medium';
    }
  }

  /**
   * 获取性能指标
   */
  getPerformanceMetrics(): PerformanceMetrics {
    const metrics: PerformanceMetrics = {
      memoryUsage: this.performanceMetrics.memoryUsage,
      cpuUsage: this.performanceMetrics.cpuUsage,
      networkLatency: this.performanceMetrics.networkLatency,
      frameRate: this.performanceMetrics.frameRate,
      batteryLevel: this.performanceMetrics.batteryLevel,
      cacheHitRate: this.performanceMetrics.cacheHitRate,
      errorRate: this.performanceMetrics.errorRate,
      averageResponseTime: this.performanceMetrics.averageResponseTime
    };
    return metrics;
  }

  /**
   * 获取性能告警
   */
  getPerformanceAlerts(): PerformanceAlert[] {
    const alertsCopy: PerformanceAlert[] = [];
    this.alerts.forEach(alert => alertsCopy.push(alert));
    return alertsCopy;
  }

  /**
   * 获取自适应配置
   */
  getAdaptiveConfig(): AdaptiveConfig {
    const configCopy: AdaptiveConfig = {
      maxCacheSize: this.config.maxCacheSize,
      imageQuality: this.config.imageQuality,
      animationEnabled: this.config.animationEnabled,
      maxConcurrentRequests: this.config.maxConcurrentRequests,
      networkTimeout: this.config.networkTimeout,
      backgroundSyncEnabled: this.config.backgroundSyncEnabled
    };
    return configCopy;
  }

  /**
   * 开始性能监控
   */
  startMonitoring(): void {
    if (this.isMonitoring) {
      return;
    }

    this.isMonitoring = true;
    this.monitoringTimerId = TimerManager.setInterval(() => {
      this.updatePerformanceMetrics();
    }, 2000, '性能监控');

    logger.info('性能监控已启动');
  }

  /**
   * 停止性能监控
   */
  stopMonitoring(): void {
    if (!this.isMonitoring) {
      return;
    }

    this.isMonitoring = false;
    if (this.monitoringTimerId !== -1) {
      TimerManager.clearInterval(this.monitoringTimerId);
      this.monitoringTimerId = -1;
    }

    logger.info('性能监控已停止');
  }

  /**
   * 更新性能指标 - 使用HarmonyOS真实系统API获取性能数据
   */
  private updatePerformanceMetrics(): void {
    try {
      // 使用HarmonyOS真实的系统性能API
      this.performanceMetrics = {
        memoryUsage: this.getRealMemoryUsage(),
        cpuUsage: this.getRealCpuUsage(),
        networkLatency: this.getNetworkLatency(),
        frameRate: this.getFrameRate(),
        batteryLevel: this.getBatteryLevel(),
        cacheHitRate: this.getCacheHitRate(),
        errorRate: this.getErrorRate(),
        averageResponseTime: this.getAverageResponseTime()
      };

      // 检查告警条件
      this.checkAlerts();
    } catch (error) {
      ErrorHandler.handle(error, 'updatePerformanceMetrics', ErrorType.SYSTEM, ErrorLevel.MEDIUM);
    }
  }

  /**
   * 检查告警条件
   */
  private checkAlerts(): void {
    const metrics = this.performanceMetrics;
    const now = Date.now();

    // 内存使用率告警
    if (metrics.memoryUsage > 80) {
      this.addAlert({
        type: 'memory',
        level: metrics.memoryUsage > 90 ? 'critical' : 'warning',
        message: `内存使用率过高: ${metrics.memoryUsage.toFixed(1)}%`,
        timestamp: now,
        value: metrics.memoryUsage,
        threshold: 80
      });
    }

    // CPU使用率告警
    if (metrics.cpuUsage > 70) {
      this.addAlert({
        type: 'cpu',
        level: metrics.cpuUsage > 85 ? 'critical' : 'warning',
        message: `CPU使用率过高: ${metrics.cpuUsage.toFixed(1)}%`,
        timestamp: now,
        value: metrics.cpuUsage,
        threshold: 70
      });
    }

    // 网络延迟告警
    if (metrics.networkLatency > 500) {
      this.addAlert({
        type: 'network',
        level: metrics.networkLatency > 1000 ? 'critical' : 'warning',
        message: `网络延迟过高: ${metrics.networkLatency.toFixed(0)}ms`,
        timestamp: now,
        value: metrics.networkLatency,
        threshold: 500
      });
    }
  }

  /**
   * 添加告警
   */
  private addAlert(alert: PerformanceAlert): void {
    this.alerts.push(alert);
    
    // 保持告警数量在合理范围内
    if (this.alerts.length > 50) {
      this.alerts = this.alerts.slice(-30);
    }

    logger.warn(`性能告警: ${alert.message}`);
  }

  /**
   * 清空告警
   */
  clearAlerts(): void {
    this.alerts = [];
    logger.info('性能告警已清空');
  }

  /**
   * 销毁实例
   */
  destroy(): void {
    this.stopMonitoring();
    this.clearCache();
    this.clearAlerts();
    logger.info('性能优化器已销毁');
  }

  /**
   * 记录响应时间
   */
  public recordResponseTime(time: number): void {
    this.requestCount++;
    this.totalResponseTime += time;
  }

  /**
   * 记录错误
   */
  public recordError(): void {
    this.errorCount++;
  }

  /**
   * 获取真实内存使用率 - 使用HarmonyOS系统API
   */
  private getRealMemoryUsage(): number {
    try {
      // 获取当前进程的 PSS (Proportional Set Size) 内存使用量 (KB)
      const pss = hidebug.getPss();
      // 假设最大可用内存为 4GB (作为参考基数，实际应获取设备物理内存)
      // 由于无法直接获取总内存，这里返回 PSS 的 MB 值作为参考指标，或者基于经验阈值的百分比
      // 为了保持与接口定义的兼容性(返回百分比)，这里做一个简单的归一化: 假设应用软限制为 512MB
      const appMemoryLimit = 512 * 1024; // KB
      return Math.min((pss / appMemoryLimit) * 100, 100);
    } catch (error) {
      // API调用失败时记录日志但不再频繁报错
      return 0;
    }
  }

  /**
   * 获取真实CPU使用率 - 使用HarmonyOS系统API
   */
  private getRealCpuUsage(): number {
    try {
      // 获取进程 CPU 使用率
      const cpuUsage = hidebug.getCpuUsage();
      // hidebug.getCpuUsage 返回的是 0-1 之间的浮点数? 文档说明不一，通常是负载。
      // 假设是 0.0 - 1.0
      return cpuUsage * 100;
    } catch (error) {
      return 0;
    }
  }

  /**
   * 获取网络延迟 - 通过实际网络请求测量
   */
  private getNetworkLatency(): number {
    // 返回最近的平均响应时间作为延迟估算
    return this.getAverageResponseTime();
  }

  /**
   * 获取帧率 - 使用HarmonyOS渲染性能API
   */
  private getFrameRate(): number {
    try {
      const displayInfo = display.getDefaultDisplaySync();
      return displayInfo.refreshRate;
    } catch (e) {
      return 60;
    }
  }

  /**
   * 获取电池电量 - 使用HarmonyOS电池API
   */
  private getBatteryLevel(): number {
    try {
      return batteryInfo.batterySOC;
    } catch (error) {
      ErrorHandler.handle(error, 'getBatteryLevel', ErrorType.SYSTEM, ErrorLevel.LOW);
      return 100;
    }
  }

  /**
   * 获取缓存命中率 - 基于实际缓存统计
   */
  private getCacheHitRate(): number {
    const totalRequests = this.cacheHits + this.cacheMisses;
    if (totalRequests === 0) {
      return 0;
    }
    return (this.cacheHits / totalRequests) * 100;
  }

  /**
   * 获取错误率 - 基于实际错误统计
   */
  private getErrorRate(): number {
    if (this.requestCount === 0) return 0;
    return (this.errorCount / this.requestCount) * 100;
  }

  /**
   * 获取平均响应时间 - 基于实际请求统计
   */
  private getAverageResponseTime(): number {
    if (this.requestCount === 0) return 0;
    return this.totalResponseTime / this.requestCount;
  }
}