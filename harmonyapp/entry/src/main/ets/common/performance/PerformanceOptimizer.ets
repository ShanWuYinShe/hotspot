/**
 * 性能优化管理器
 * 提供内存管理、缓存优化、渲染优化等功能
 */

import { hilog } from '@kit.PerformanceAnalysisKit';
import { display } from '@kit.ArkUI';
import { deviceInfo } from '@kit.BasicServicesKit';
import { resourceManager } from '@kit.LocalizationKit';
import { StorageUtil } from '../utils/StorageUtil';

const DOMAIN_NUMBER: number = 0xFF12;
const TAG: string = 'PerformanceOptimizer';

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

/**
 * 性能监控数据接口
 */
export interface PerformanceMetrics {
  memoryUsage: number;
  cacheHitRate: number;
  averageResponseTime: number;
  errorRate: number;
  activeConnections: number;
  cpuUsage?: number;
  batteryLevel?: number;
  networkLatency?: number;
  frameRate?: number;
}

/**
 * 性能告警接口
 */
export interface PerformanceAlert {
  type: 'memory' | 'cpu' | 'network' | 'battery' | 'cache';
  level: 'warning' | 'critical';
  message: string;
  timestamp: number;
  metrics: Partial<PerformanceMetrics>;
  suggestions: string[];
}

/**
 * 自适应配置接口
 */
export interface AdaptiveConfig {
  imageQuality: 'low' | 'medium' | 'high';
  animationEnabled: boolean;
  cacheSize: number;
  networkTimeout: number;
  maxConcurrentRequests: number;
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

export class PerformanceOptimizer {
  private static instance: PerformanceOptimizer | null = null;
  private cache: Map<string, CacheItem<any>> = new Map();
  private maxCacheSize: number = 100;
  private defaultExpiry: number = 5 * 60 * 1000; // 5分钟
  private performanceMetrics: PerformanceMetrics;
  private requestTimes: number[] = [];
  private errorCount: number = 0;
  private totalRequests: number = 0;
  
  // 新增智能监控属性
  private performanceAlerts: PerformanceAlert[] = [];
  private adaptiveConfig: AdaptiveConfig;
  private monitoringInterval: number = -1;
  private performanceHistory: PerformanceMetrics[] = [];
  private readonly maxHistorySize = 100;
  private alertCallbacks: ((alert: PerformanceAlert) => void)[] = [];

  private constructor() {
    this.performanceMetrics = {
      memoryUsage: 0,
      cacheHitRate: 0,
      averageResponseTime: 0,
      errorRate: 0,
      activeConnections: 0
    };
    
    // 初始化自适应配置
    this.adaptiveConfig = {
      imageQuality: 'medium',
      animationEnabled: true,
      cacheSize: 100,
      networkTimeout: 10000,
      maxConcurrentRequests: 5,
      backgroundSyncEnabled: true
    };

    // 定期清理过期缓存
    this.startCacheCleanup();
    
    // 启动性能监控
    this.startPerformanceMonitoring();
  }

  static getInstance(): PerformanceOptimizer {
    if (!PerformanceOptimizer.instance) {
      PerformanceOptimizer.instance = new PerformanceOptimizer();
    }
    return PerformanceOptimizer.instance;
  }

  // ==================== 缓存管理 ====================

  /**
   * 设置缓存
   */
  setCache<T>(key: string, data: T, expiry?: number): void {
    try {
      // 如果缓存已满，清理最少使用的项
      if (this.cache.size >= this.maxCacheSize) {
        this.evictLeastUsed();
      }

      const cacheItem = {
        data,
        timestamp: Date.now(),
        expiry: expiry || this.defaultExpiry,
        accessCount: 0,
        lastAccess: Date.now()
      };

      this.cache.set(key, cacheItem);
      hilog.debug(DOMAIN_NUMBER, TAG, `缓存设置成功: ${key}`);
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `设置缓存失败: ${key}, error: ${JSON.stringify(error)}`);
    }
  }

  /**
   * 获取缓存
   */
  getCache<T>(key: string): T | null {
    try {
      const cacheItem = this.cache.get(key);
      if (!cacheItem) {
        return null;
      }

      const now = Date.now();
      
      // 检查是否过期
      if (now - cacheItem.timestamp > cacheItem.expiry) {
        this.cache.delete(key);
        hilog.debug(DOMAIN_NUMBER, TAG, `缓存已过期: ${key}`);
        return null;
      }

      // 更新访问统计
      cacheItem.accessCount++;
      cacheItem.lastAccess = now;

      hilog.debug(DOMAIN_NUMBER, TAG, `缓存命中: ${key}`);
      return cacheItem.data as T;
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `获取缓存失败: ${key}, error: ${JSON.stringify(error)}`);
      return null;
    }
  }

  /**
   * 删除缓存
   */
  deleteCache(key: string): boolean {
    try {
      const deleted = this.cache.delete(key);
      if (deleted) {
        hilog.debug(DOMAIN_NUMBER, TAG, `缓存删除成功: ${key}`);
      }
      return deleted;
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `删除缓存失败: ${key}, error: ${JSON.stringify(error)}`);
      return false;
    }
  }

  /**
   * 清空所有缓存
   */
  clearCache(): void {
    try {
      this.cache.clear();
      hilog.info(DOMAIN_NUMBER, TAG, '所有缓存已清空');
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `清空缓存失败: ${JSON.stringify(error)}`);
    }
  }

  /**
   * 清理过期缓存
   */
  private cleanupExpiredCache(): void {
    try {
      const now = Date.now();
      const expiredKeys: string[] = [];

      for (const [key, cacheItem] of this.cache.entries()) {
        if (now - cacheItem.timestamp > cacheItem.expiry) {
          expiredKeys.push(key);
        }
      }

      expiredKeys.forEach(key => this.cache.delete(key));
      
      if (expiredKeys.length > 0) {
        hilog.info(DOMAIN_NUMBER, TAG, `清理过期缓存: ${expiredKeys.length}个`);
      }
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `清理过期缓存失败: ${JSON.stringify(error)}`);
    }
  }

  /**
   * 驱逐最少使用的缓存项
   */
  private evictLeastUsed(): void {
    try {
      let leastUsedKey: string | null = null;
      let minAccessCount = Infinity;
      let oldestAccess = Infinity;

      for (const [key, cacheItem] of this.cache.entries()) {
        if (cacheItem.accessCount < minAccessCount || 
            (cacheItem.accessCount === minAccessCount && cacheItem.lastAccess < oldestAccess)) {
          leastUsedKey = key;
          minAccessCount = cacheItem.accessCount;
          oldestAccess = cacheItem.lastAccess;
        }
      }

      if (leastUsedKey) {
        this.cache.delete(leastUsedKey);
        hilog.debug(DOMAIN_NUMBER, TAG, `驱逐最少使用缓存: ${leastUsedKey}`);
      }
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `驱逐缓存失败: ${JSON.stringify(error)}`);
    }
  }

  /**
   * 开始缓存清理定时器
   */
  private startCacheCleanup(): void {
    setInterval(() => {
      this.cleanupExpiredCache();
      this.updateCacheMetrics();
    }, 60000); // 每分钟清理一次
  }

  // ==================== 性能监控 ====================

  /**
   * 记录请求开始时间
   */
  startRequest(): number {
    const startTime = Date.now();
    this.totalRequests++;
    return startTime;
  }

  /**
   * 记录请求结束时间
   */
  endRequest(startTime: number, success: boolean = true): void {
    try {
      const endTime = Date.now();
      const responseTime = endTime - startTime;
      
      this.requestTimes.push(responseTime);
      
      // 只保留最近100次请求的时间
      if (this.requestTimes.length > 100) {
        this.requestTimes.shift();
      }

      if (!success) {
        this.errorCount++;
      }

      this.updatePerformanceMetrics();
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `记录请求结束失败: ${JSON.stringify(error)}`);
    }
  }

  /**
   * 更新性能指标
   */
  private updatePerformanceMetrics(): void {
    try {
      // 计算平均响应时间
      if (this.requestTimes.length > 0) {
        const sum = this.requestTimes.reduce((a, b) => a + b, 0);
        this.performanceMetrics.averageResponseTime = sum / this.requestTimes.length;
      }

      // 计算错误率
      this.performanceMetrics.errorRate = this.totalRequests > 0 ? 
        (this.errorCount / this.totalRequests) * 100 : 0;

      // 更新内存使用情况（简化版）
      this.performanceMetrics.memoryUsage = this.cache.size;
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `更新性能指标失败: ${JSON.stringify(error)}`);
    }
  }

  /**
   * 更新缓存指标
   */
  private updateCacheMetrics(): void {
    try {
      // 计算缓存命中率（简化版）
      const totalAccess = Array.from(this.cache.values())
        .reduce((sum, item) => sum + item.accessCount, 0);
      
      this.performanceMetrics.cacheHitRate = totalAccess > 0 ? 
        (this.cache.size / totalAccess) * 100 : 0;
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `更新缓存指标失败: ${JSON.stringify(error)}`);
    }
  }

  /**
   * 获取性能指标
   */
  getPerformanceMetrics(): PerformanceMetrics {
    return { ...this.performanceMetrics };
  }

  // ==================== 设备适配 ====================

  /**
   * 获取设备适配信息
   */
  getDeviceAdaptationInfo(): DeviceAdaptationInfo {
    try {
      // 从系统API获取真实的设备信息
      const display = display.getDefaultDisplaySync();
      const deviceInfo = deviceInfo.deviceType;
      
      const adaptationInfo = {
        screenWidth: display.width,
        screenHeight: display.height,
        density: display.densityDPI / 160, // 转换为密度倍数
        orientation: display.width > display.height ? 'landscape' : 'portrait',
        deviceType: deviceInfo === 'phone' ? 'phone' : deviceInfo === 'tablet' ? 'tablet' : 'unknown',
        performanceLevel: this.getPerformanceLevel()
      };

      hilog.debug(DOMAIN_NUMBER, TAG, `设备适配信息: ${JSON.stringify(adaptationInfo)}`);
      return adaptationInfo;
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `获取设备适配信息失败: ${JSON.stringify(error)}`);
      // 返回默认值
      return {
        screenWidth: 1080,
        screenHeight: 2340,
        density: 3.0,
        orientation: 'portrait',
        deviceType: 'phone',
        performanceLevel: 'medium'
      };
    }
  }

  /**
   * 获取设备性能等级
   */
  private getPerformanceLevel(): 'low' | 'medium' | 'high' {
    try {
      // 基于设备信息判断性能等级
      const totalMemory = deviceInfo.totalMemory || 0;
      
      if (totalMemory > 8 * 1024 * 1024 * 1024) { // 8GB以上
        return 'high';
      } else if (totalMemory > 4 * 1024 * 1024 * 1024) { // 4GB以上
        return 'medium';
      } else {
        return 'low';
      }
    } catch (error) {
      return 'medium';
    }
  }

  /**
   * 根据设备性能调整配置
   */
  getOptimizedConfig(): Record<string, any> {
    const deviceInfo = this.getDeviceAdaptationInfo();
    const config = {};

    try {
      // 根据性能等级调整配置
      switch (deviceInfo.performanceLevel) {
        case 'low':
          config.maxCacheSize = 50;
          config.imageQuality = 'low';
          config.animationEnabled = false;
          config.maxConcurrentRequests = 2;
          break;
        case 'medium':
          config.maxCacheSize = 100;
          config.imageQuality = 'medium';
          config.animationEnabled = true;
          config.maxConcurrentRequests = 4;
          break;
        case 'high':
          config.maxCacheSize = 200;
          config.imageQuality = 'high';
          config.animationEnabled = true;
          config.maxConcurrentRequests = 8;
          break;
      }

      // 根据设备类型调整配置
      switch (deviceInfo.deviceType) {
        case 'phone':
          config.listItemHeight = 60;
          config.fontSize = 16;
          break;
        case 'tablet':
          config.listItemHeight = 80;
          config.fontSize = 18;
          break;
        case 'tv':
          config.listItemHeight = 100;
          config.fontSize = 24;
          break;
      }

      // 根据屏幕密度调整配置
      if (deviceInfo.density >= 3.0) {
        config.imageScale = 3;
      } else if (deviceInfo.density >= 2.0) {
        config.imageScale = 2;
      } else {
        config.imageScale = 1;
      }

      this.maxCacheSize = config.maxCacheSize;
      hilog.info(DOMAIN_NUMBER, TAG, `优化配置: ${JSON.stringify(config)}`);
      return config;
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `获取优化配置失败: ${JSON.stringify(error)}`);
      return {};
    }
  }

  // ==================== 内存管理 ====================

  /**
   * 内存压力检测
   */
  checkMemoryPressure(): 'low' | 'medium' | 'high' {
    try {
      // 获取系统内存信息
      const availableMemory = deviceInfo.availableMemory || 0;
      const totalMemory = deviceInfo.totalMemory || 0;
      
      if (totalMemory > 0) {
        const memoryUsageRatio = (totalMemory - availableMemory) / totalMemory;
        
        if (memoryUsageRatio > 0.85) {
          return 'high';
        } else if (memoryUsageRatio > 0.65) {
          return 'medium';
        } else {
          return 'low';
        }
      } else {
        // 如果无法获取系统内存信息，基于缓存大小进行判断
        const cacheSize = this.cache.size;
        
        if (cacheSize > this.maxCacheSize * 0.8) {
          return 'high';
        } else if (cacheSize > this.maxCacheSize * 0.5) {
          return 'medium';
        } else {
          return 'low';
        }
      }
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `检测内存压力失败: ${JSON.stringify(error)}`);
      return 'medium';
    }
  }

  /**
   * 内存优化
   */
  optimizeMemory(): void {
    try {
      const memoryPressure = this.checkMemoryPressure();
      
      switch (memoryPressure) {
        case 'high':
          // 清理一半的缓存
          this.clearOldCache(0.5);
          hilog.info(DOMAIN_NUMBER, TAG, '高内存压力，执行激进清理');
          break;
        case 'medium':
          // 清理过期缓存
          this.cleanupExpiredCache();
          hilog.info(DOMAIN_NUMBER, TAG, '中等内存压力，清理过期缓存');
          break;
        case 'low':
          // 无需特殊处理
          hilog.debug(DOMAIN_NUMBER, TAG, '内存压力正常');
          break;
      }
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `内存优化失败: ${JSON.stringify(error)}`);
    }
  }

  /**
   * 清理旧缓存
   */
  private clearOldCache(ratio: number): void {
    try {
      const targetSize = Math.floor(this.cache.size * (1 - ratio));
      const cacheEntries = Array.from(this.cache.entries());
      
      // 按最后访问时间排序
      cacheEntries.sort((a, b) => a[1].lastAccess - b[1].lastAccess);
      
      // 删除最旧的缓存项
      for (let i = 0; i < cacheEntries.length - targetSize; i++) {
        this.cache.delete(cacheEntries[i][0]);
      }
      
      hilog.info(DOMAIN_NUMBER, TAG, `清理旧缓存: ${cacheEntries.length - targetSize}个`);
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `清理旧缓存失败: ${JSON.stringify(error)}`);
    }
  }

  // ==================== 渲染优化 ====================

  /**
   * 获取列表渲染优化配置
   */
  getListRenderConfig(): Record<string, any> {
    const deviceInfo = this.getDeviceAdaptationInfo();
    
    return {
      // 虚拟滚动配置
      virtualScrollEnabled: deviceInfo.performanceLevel !== 'high',
      itemHeight: deviceInfo.deviceType === 'tablet' ? 80 : 60,
      bufferSize: deviceInfo.performanceLevel === 'low' ? 5 : 10,
      
      // 图片懒加载配置
      lazyLoadEnabled: true,
      lazyLoadThreshold: deviceInfo.performanceLevel === 'low' ? 100 : 200,
      
      // 动画配置
      animationDuration: deviceInfo.performanceLevel === 'low' ? 150 : 300,
      animationEnabled: deviceInfo.performanceLevel !== 'low'
    };
  }

  // ==================== 工具方法 ====================

  /**
   * 获取缓存统计信息
   */
  getCacheStats(): Record<string, any> {
    const totalAccess = Array.from(this.cache.values())
      .reduce((sum, item) => sum + item.accessCount, 0);
    
    return {
      size: this.cache.size,
      maxSize: this.maxCacheSize,
      totalAccess,
      hitRate: this.performanceMetrics.cacheHitRate,
      memoryUsage: this.cache.size * 1024 // 简化计算
    };
  }

  /**
   * 重置性能统计
   */
  resetPerformanceStats(): void {
    this.requestTimes = [];
    this.errorCount = 0;
    this.totalRequests = 0;
    this.performanceMetrics = {
      memoryUsage: 0,
      cacheHitRate: 0,
      averageResponseTime: 0,
      errorRate: 0,
      activeConnections: 0
    };
    hilog.info(DOMAIN_NUMBER, TAG, '性能统计已重置');
  }

  /**
   * 启动性能监控
   */
  private startPerformanceMonitoring(): void {
    if (this.monitoringInterval !== -1) {
      return; // 已经启动
    }
    
    this.monitoringInterval = setInterval(() => {
      this.collectPerformanceMetrics();
      this.analyzePerformance();
      this.adaptConfiguration();
    }, 30000); // 每30秒监控一次
    
    hilog.info(DOMAIN_NUMBER, TAG, '性能监控已启动');
  }

  /**
   * 停止性能监控
   */
  private stopPerformanceMonitoring(): void {
    if (this.monitoringInterval !== -1) {
      clearInterval(this.monitoringInterval);
      this.monitoringInterval = -1;
      hilog.info(DOMAIN_NUMBER, TAG, '性能监控已停止');
    }
  }

  /**
   * 收集性能指标
   */
  private async collectPerformanceMetrics(): Promise<void> {
    try {
      // 获取内存使用情况
      const memoryInfo = await this.getMemoryInfo();
      
      // 计算缓存命中率
      const totalCacheAccess = this.getCacheHitRate();
      
      // 计算平均响应时间
      const avgResponseTime = this.requestTimes.length > 0 
        ? this.requestTimes.reduce((a, b) => a + b, 0) / this.requestTimes.length 
        : 0;
      
      // 计算错误率
      const errorRate = this.totalRequests > 0 ? this.errorCount / this.totalRequests : 0;
      
      // 更新性能指标
      this.performanceMetrics = {
        ...this.performanceMetrics,
        memoryUsage: memoryInfo.used / memoryInfo.total,
        cacheHitRate: totalCacheAccess,
        averageResponseTime: avgResponseTime,
        errorRate: errorRate,
        cpuUsage: await this.getCpuUsage(),
        batteryLevel: await this.getBatteryLevel(),
        networkLatency: await this.getNetworkLatency(),
        frameRate: await this.getFrameRate()
      };
      
      // 保存到历史记录
      this.performanceHistory.push({ ...this.performanceMetrics });
      if (this.performanceHistory.length > this.maxHistorySize) {
        this.performanceHistory.shift();
      }
      
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `收集性能指标失败: ${JSON.stringify(error)}`);
    }
  }

  /**
   * 分析性能并生成告警
   */
  private analyzePerformance(): void {
    const metrics = this.performanceMetrics;
    const alerts: PerformanceAlert[] = [];
    
    // 内存使用告警
    if (metrics.memoryUsage > 0.8) {
      alerts.push({
        type: 'memory',
        level: metrics.memoryUsage > 0.9 ? 'critical' : 'warning',
        message: `内存使用率过高: ${(metrics.memoryUsage * 100).toFixed(1)}%`,
        timestamp: Date.now(),
        metrics: { memoryUsage: metrics.memoryUsage },
        suggestions: [
          '清理缓存数据',
          '减少图片质量',
          '关闭后台同步',
          '重启应用'
        ]
      });
    }
    
    // 网络延迟告警
    if (metrics.networkLatency && metrics.networkLatency > 2000) {
      alerts.push({
        type: 'network',
        level: metrics.networkLatency > 5000 ? 'critical' : 'warning',
        message: `网络延迟过高: ${metrics.networkLatency}ms`,
        timestamp: Date.now(),
        metrics: { networkLatency: metrics.networkLatency },
        suggestions: [
          '检查网络连接',
          '切换到更快的网络',
          '启用离线模式',
          '减少并发请求'
        ]
      });
    }
    
    // 错误率告警
    if (metrics.errorRate > 0.1) {
      alerts.push({
        type: 'network',
        level: metrics.errorRate > 0.2 ? 'critical' : 'warning',
        message: `错误率过高: ${(metrics.errorRate * 100).toFixed(1)}%`,
        timestamp: Date.now(),
        metrics: { errorRate: metrics.errorRate },
        suggestions: [
          '检查网络连接',
          '重试失败的请求',
          '联系技术支持'
        ]
      });
    }
    
    // 缓存命中率告警
    if (metrics.cacheHitRate < 0.3 && this.cache.size > 10) {
      alerts.push({
        type: 'cache',
        level: 'warning',
        message: `缓存命中率过低: ${(metrics.cacheHitRate * 100).toFixed(1)}%`,
        timestamp: Date.now(),
        metrics: { cacheHitRate: metrics.cacheHitRate },
        suggestions: [
          '调整缓存策略',
          '增加缓存大小',
          '优化数据访问模式'
        ]
      });
    }
    
    // 电池电量告警
    if (metrics.batteryLevel && metrics.batteryLevel < 0.2) {
      alerts.push({
        type: 'battery',
        level: metrics.batteryLevel < 0.1 ? 'critical' : 'warning',
        message: `电池电量不足: ${(metrics.batteryLevel * 100).toFixed(0)}%`,
        timestamp: Date.now(),
        metrics: { batteryLevel: metrics.batteryLevel },
        suggestions: [
          '启用省电模式',
          '关闭动画效果',
          '减少后台活动',
          '降低屏幕亮度'
        ]
      });
    }
    
    // 触发告警回调
    alerts.forEach(alert => {
      this.performanceAlerts.push(alert);
      this.alertCallbacks.forEach(callback => {
        try {
          callback(alert);
        } catch (error) {
          hilog.error(DOMAIN_NUMBER, TAG, `告警回调执行失败: ${JSON.stringify(error)}`);
        }
      });
    });
    
    // 限制告警历史数量
    if (this.performanceAlerts.length > 50) {
      this.performanceAlerts = this.performanceAlerts.slice(-50);
    }
  }

  /**
   * 自适应配置调整
   */
  private adaptConfiguration(): void {
    const metrics = this.performanceMetrics;
    let configChanged = false;
    
    // 根据内存使用情况调整
    if (metrics.memoryUsage > 0.8) {
      if (this.adaptiveConfig.imageQuality !== 'low') {
        this.adaptiveConfig.imageQuality = 'low';
        configChanged = true;
      }
      if (this.adaptiveConfig.cacheSize > 50) {
        this.adaptiveConfig.cacheSize = 50;
        this.maxCacheSize = 50;
        configChanged = true;
      }
    } else if (metrics.memoryUsage < 0.5) {
      if (this.adaptiveConfig.imageQuality !== 'high') {
        this.adaptiveConfig.imageQuality = 'high';
        configChanged = true;
      }
      if (this.adaptiveConfig.cacheSize < 200) {
        this.adaptiveConfig.cacheSize = 200;
        this.maxCacheSize = 200;
        configChanged = true;
      }
    }
    
    // 根据网络延迟调整
    if (metrics.networkLatency && metrics.networkLatency > 3000) {
      if (this.adaptiveConfig.networkTimeout < 20000) {
        this.adaptiveConfig.networkTimeout = 20000;
        configChanged = true;
      }
      if (this.adaptiveConfig.maxConcurrentRequests > 2) {
        this.adaptiveConfig.maxConcurrentRequests = 2;
        configChanged = true;
      }
    } else if (metrics.networkLatency && metrics.networkLatency < 500) {
      if (this.adaptiveConfig.maxConcurrentRequests < 8) {
        this.adaptiveConfig.maxConcurrentRequests = 8;
        configChanged = true;
      }
    }
    
    // 根据电池电量调整
    if (metrics.batteryLevel && metrics.batteryLevel < 0.2) {
      if (this.adaptiveConfig.animationEnabled) {
        this.adaptiveConfig.animationEnabled = false;
        configChanged = true;
      }
      if (this.adaptiveConfig.backgroundSyncEnabled) {
        this.adaptiveConfig.backgroundSyncEnabled = false;
        configChanged = true;
      }
    } else if (metrics.batteryLevel && metrics.batteryLevel > 0.5) {
      if (!this.adaptiveConfig.animationEnabled) {
        this.adaptiveConfig.animationEnabled = true;
        configChanged = true;
      }
      if (!this.adaptiveConfig.backgroundSyncEnabled) {
        this.adaptiveConfig.backgroundSyncEnabled = true;
        configChanged = true;
      }
    }
    
    if (configChanged) {
      hilog.info(DOMAIN_NUMBER, TAG, `自适应配置已更新: ${JSON.stringify(this.adaptiveConfig)}`);
      this.saveAdaptiveConfig();
    }
  }

  /**
   * 获取内存信息
   */
  private async getMemoryInfo(): Promise<{ used: number; total: number }> {
    try {
      // 使用 HarmonyOS 设备信息 API
      const totalMemory = deviceInfo.totalMemory || 4 * 1024 * 1024 * 1024; // 默认4GB
      const availableMemory = deviceInfo.availableMemory || totalMemory * 0.5;
      const usedMemory = totalMemory - availableMemory;
      
      return {
        used: usedMemory,
        total: totalMemory
      };
    } catch (error) {
      hilog.warn(DOMAIN_NUMBER, TAG, '获取内存信息失败，使用默认值');
      return { used: 2 * 1024 * 1024 * 1024, total: 4 * 1024 * 1024 * 1024 };
    }
  }

  /**
   * 获取CPU使用率（模拟）
   */
  private async getCpuUsage(): Promise<number> {
    // HarmonyOS 暂时没有直接的CPU使用率API，这里模拟
    return Math.random() * 0.8; // 0-80%
  }

  /**
   * 获取电池电量
   */
  private async getBatteryLevel(): Promise<number> {
    try {
      // 这里需要使用 HarmonyOS 电池API，暂时模拟
      return Math.random() * 0.8 + 0.2; // 20-100%
    } catch (error) {
      return 1.0; // 默认满电
    }
  }

  /**
   * 获取网络延迟
   */
  private async getNetworkLatency(): Promise<number> {
    const startTime = Date.now();
    try {
      // 简单的网络延迟测试
      await fetch('https://www.baidu.com', { 
        method: 'HEAD',
        mode: 'no-cors'
      });
      return Date.now() - startTime;
    } catch (error) {
      return 5000; // 默认5秒超时
    }
  }

  /**
   * 获取帧率（模拟）
   */
  private async getFrameRate(): Promise<number> {
    // HarmonyOS 暂时没有直接的帧率API，这里模拟
    return 60 - Math.random() * 10; // 50-60 FPS
  }

  /**
   * 保存自适应配置
   */
  private async saveAdaptiveConfig(): Promise<void> {
    try {
      await StorageUtil.setItem('adaptive_config', JSON.stringify(this.adaptiveConfig));
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `保存自适应配置失败: ${JSON.stringify(error)}`);
    }
  }

  /**
   * 加载自适应配置
   */
  private async loadAdaptiveConfig(): Promise<void> {
    try {
      const configStr = await StorageUtil.getItem('adaptive_config');
      if (configStr) {
        this.adaptiveConfig = { ...this.adaptiveConfig, ...JSON.parse(configStr) };
        this.maxCacheSize = this.adaptiveConfig.cacheSize;
      }
    } catch (error) {
      hilog.warn(DOMAIN_NUMBER, TAG, '加载自适应配置失败，使用默认配置');
    }
  }

  /**
   * 注册性能告警回调
   */
  public onPerformanceAlert(callback: (alert: PerformanceAlert) => void): void {
    this.alertCallbacks.push(callback);
  }

  /**
   * 移除性能告警回调
   */
  public offPerformanceAlert(callback: (alert: PerformanceAlert) => void): void {
    const index = this.alertCallbacks.indexOf(callback);
    if (index > -1) {
      this.alertCallbacks.splice(index, 1);
    }
  }

  /**
   * 获取当前自适应配置
   */
  public getAdaptiveConfig(): AdaptiveConfig {
    return { ...this.adaptiveConfig };
  }

  /**
   * 手动设置自适应配置
   */
  public setAdaptiveConfig(config: Partial<AdaptiveConfig>): void {
    this.adaptiveConfig = { ...this.adaptiveConfig, ...config };
    if (config.cacheSize) {
      this.maxCacheSize = config.cacheSize;
    }
    this.saveAdaptiveConfig();
    hilog.info(DOMAIN_NUMBER, TAG, `手动更新自适应配置: ${JSON.stringify(config)}`);
  }

  /**
   * 获取性能告警历史
   */
  public getPerformanceAlerts(): PerformanceAlert[] {
    return [...this.performanceAlerts];
  }

  /**
   * 获取性能历史数据
   */
  public getPerformanceHistory(): PerformanceMetrics[] {
    return [...this.performanceHistory];
  }

  /**
   * 清理资源
   */
  cleanup(): void {
    try {
      this.stopPerformanceMonitoring();
      this.clearCache();
      this.resetPerformanceStats();
      this.performanceAlerts = [];
      this.performanceHistory = [];
      this.alertCallbacks = [];
      hilog.info(DOMAIN_NUMBER, TAG, '性能优化器资源清理完成');
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `清理资源失败: ${JSON.stringify(error)}`);
    }
  }
}

