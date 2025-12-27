/**
 * 服务管理器
 * 统一管理所有智能服务的生命周期和状态
 */
import { hilog } from '@kit.PerformanceAnalysisKit';
import { BusinessError } from '@kit.BasicServicesKit';
import { PerformanceOptimizer } from '../performance/PerformanceOptimizer';
import { SmartNotificationService } from './SmartNotificationService';
import { DistributedSyncService } from './DistributedSyncService';
import { AccessibilityManager } from '../accessibility/AccessibilityManager';
import { common } from '@kit.AbilityKit';

export interface ServiceStatus {
  name: string;
  status: 'initializing' | 'running' | 'stopped' | 'error';
  lastUpdate: number;
  errorMessage?: string;
  metadata?: Record<string, Object>;
}

export interface ServiceManagerConfig {
  enablePerformanceOptimizer: boolean;
  enableAIRecommendation: boolean;
  enableSmartNotification: boolean;
  enableDistributedSync: boolean;
  enableAccessibility: boolean;
  autoRestart: boolean;
  healthCheckInterval: number;
}

interface PerformanceService {
  getPerformanceMetrics?: () => Object | null;
}

interface RecommendationService {
  getRecommendations?: () => Object;
}

interface ConfigurableService {
  getConfig?: () => Object;
}

interface SyncService {
  syncUserData?: () => Promise<void>;
}

interface CleanupService {
  cleanup?: () => Promise<void>;
}

interface ServiceStats {
  total: number;
  running: number;
  stopped: number;
  error: number;
  initializing: number;
}

interface ServiceManagerConfigUpdate {
  enablePerformanceOptimizer?: boolean;
  enableAIRecommendation?: boolean;
  enableSmartNotification?: boolean;
  enableDistributedSync?: boolean;
  enableAccessibility?: boolean;
  autoRestart?: boolean;
  healthCheckInterval?: number;
}

interface ServiceReport {
  timestamp: number;
  config: ServiceManagerConfig;
  services: ServiceStatus[];
  stats: ServiceStats;
}

export class ServiceManager {
  private static instance: ServiceManager;
  private services: Map<string, Object> = new Map();
  private serviceStatus: Map<string, ServiceStatus> = new Map();
  private config: ServiceManagerConfig;
  private healthCheckTimer: number = -1;
  private listeners: ((status: Map<string, ServiceStatus>) => void)[] = [];

  private constructor() {
    this.config = {
      enablePerformanceOptimizer: true,
      enableAIRecommendation: true,
      enableSmartNotification: true,
      enableDistributedSync: true,
      enableAccessibility: true,
      autoRestart: true,
      healthCheckInterval: 30000 // 30秒
    };
  }

  public static getInstance(): ServiceManager {
    if (!ServiceManager.instance) {
      ServiceManager.instance = new ServiceManager();
    }
    return ServiceManager.instance;
  }

  /**
   * 初始化所有服务
   * @param context 从页面组件传入的 Context
   */
  public async initializeServices(context: common.Context): Promise<void> {
    hilog.info(0x0000, 'ServiceManager', '开始初始化所有智能服务');

    try {
      // 按依赖顺序初始化服务
      await this.initializeService('AccessibilityManager', () => {
        if (!this.config.enableAccessibility) return null;
        return AccessibilityManager.getInstance();
      });

      await this.initializeService('PerformanceOptimizer', () => {
        if (!this.config.enablePerformanceOptimizer) return null;
        return PerformanceOptimizer.getInstance();
      });

      await this.initializeService('DistributedSyncService', async () => {
        if (!this.config.enableDistributedSync) return null;
        const service = DistributedSyncService.getInstance();
        // 使用传入的 context
        await service.initialize(context);
        return service;
      });


      await this.initializeService('SmartNotificationService', () => {
        if (!this.config.enableSmartNotification) return null;
        return SmartNotificationService.getInstance();
      });

      // 启动健康检查
      this.startHealthCheck();

      hilog.info(0x0000, 'ServiceManager', '所有智能服务初始化完成');
      this.notifyListeners();

    } catch (error) {
      const err = error as BusinessError;
      hilog.error(0x0000, 'ServiceManager', `服务初始化失败: ${err.message}`);
      throw new Error(`服务管理器初始化失败: ${JSON.stringify(error)}`);
    }
  }

  /**
   * 初始化单个服务
   */
  private async initializeService(
    name: string, 
    factory: () => Object | Promise<Object> | null
  ): Promise<void> {
    try {
      this.updateServiceStatus(name, 'initializing');

      const service = await factory();
      
      if (service) {
        this.services.set(name, service);
        const metadata: Record<string, Object> = {};
        metadata['initTime'] = Date.now();
        metadata['version'] = '1.0.0';
        this.updateServiceStatus(name, 'running', metadata);
        hilog.info(0x0000, 'ServiceManager', `服务 ${name} 初始化成功`);
      } else {
        this.updateServiceStatus(name, 'stopped', undefined, '服务已禁用');
        hilog.info(0x0000, 'ServiceManager', `服务 ${name} 已禁用`);
      }

    } catch (error) {
      const err = error as BusinessError;
      this.updateServiceStatus(name, 'error', undefined, err.message);
      hilog.error(0x0000, 'ServiceManager', `服务 ${name} 初始化失败: ${err.message}`);
      
      if (!this.config.autoRestart) {
        throw new Error(`服务管理器初始化失败: ${JSON.stringify(error)}`);
      }
    }
  }

  /**
   * 更新服务状态
   */
  private updateServiceStatus(
    name: string, 
    status: 'running' | 'stopped' | 'error' | 'initializing', 
    metadata?: Record<string, Object>,
    errorMessage?: string
  ): void {
    this.serviceStatus.set(name, {
      name,
      status,
      lastUpdate: Date.now(),
      errorMessage,
      metadata
    });
  }

  /**
   * 启动健康检查
   */
  private startHealthCheck(): void {
    if (this.healthCheckTimer !== -1) {
      return;
    }

    this.healthCheckTimer = setInterval(() => {
      this.performHealthCheck();
    }, this.config.healthCheckInterval);

    hilog.info(0x0000, 'ServiceManager', '健康检查已启动');
  }

  /**
   * 执行健康检查
   */
  private async performHealthCheck(): Promise<void> {
    try {
      const serviceNames = Array.from(this.services.keys());
      for (let i = 0; i < serviceNames.length; i++) {
        const name = serviceNames[i];
        const service = this.services.get(name);
        if (!service) continue;
        
        const currentStatus = this.serviceStatus.get(name);
        
        if (!currentStatus || currentStatus.status === 'error') {
          continue;
        }

        // 检查服务是否响应
        const isHealthy = await this.checkServiceHealth(name, service);
        
        if (!isHealthy && this.config.autoRestart) {
          hilog.warn(0x0000, 'ServiceManager', `服务 ${name} 健康检查失败，尝试重启`);
          await this.restartService(name);
        }
      }
    } catch (error) {
      hilog.error(0x0000, 'ServiceManager', `健康检查失败: ${JSON.stringify(error)}`);
    }
  }

  /**
   * 检查单个服务健康状态
   */
  private async checkServiceHealth(name: string, service: Object): Promise<boolean> {
    try {
      // 根据服务类型执行不同的健康检查
      switch (name) {
        case 'PerformanceOptimizer':
          // 检查性能优化器是否能获取指标
          const performanceService = service as PerformanceService;
          const metrics = performanceService.getPerformanceMetrics?.();
          return metrics !== null;


        case 'SmartNotificationService':
          // 检查通知服务是否正常
          const notificationService = service as ConfigurableService;
          const config = notificationService.getConfig?.();
          return config !== null;

        case 'DistributedSyncService':
          // 检查分布式同步服务是否正常
          const syncService = service as SyncService;
          return typeof syncService.syncUserData === 'function';

        case 'AccessibilityManager':
          // 检查无障碍管理器是否正常
          const accessibilityService = service as ConfigurableService;
          const accessConfig = accessibilityService.getConfig?.();
          return accessConfig !== null;

        default:
          return true;
      }
    } catch (error) {
      hilog.error(0x0000, 'ServiceManager', `服务 ${name} 健康检查异常: ${JSON.stringify(error)}`);
      return false;
    }
  }

  /**
   * 重启服务
   */
  private async restartService(name: string): Promise<void> {
    try {
      this.updateServiceStatus(name, 'initializing');

      // 根据服务类型执行重启逻辑
      let newService: Object | null = null;

      switch (name) {
        case 'PerformanceOptimizer':
          newService = PerformanceOptimizer.getInstance();
          break;
        case 'SmartNotificationService':
          newService = SmartNotificationService.getInstance();
          break;
        case 'DistributedSyncService':
          // 重启服务时需要重新传入 Context
          throw new Error('DistributedSyncService 重启需要 Context，请通过 initializeServices(context) 重新初始化所有服务');
        case 'AccessibilityManager':
          newService = AccessibilityManager.getInstance();
          break;
      }

      if (newService) {
        this.services.set(name, newService);
        const currentStatus = this.serviceStatus.get(name);
        const currentRestartCount = currentStatus?.metadata?.restartCount as number || 0;
        const restartMetadata: Record<string, Object> = {};
        restartMetadata['restartTime'] = Date.now();
        restartMetadata['restartCount'] = currentRestartCount + 1;
        this.updateServiceStatus(name, 'running', restartMetadata);
        
        hilog.info(0x0000, 'ServiceManager', `服务 ${name} 重启成功`);
      }

    } catch (error) {
      const err = error as BusinessError;
      this.updateServiceStatus(name, 'error', undefined, err.message);
      hilog.error(0x0000, 'ServiceManager', `服务 ${name} 重启失败: ${err.message}`);
    }
  }

  /**
   * 获取服务实例
   */
  public getService<T>(name: string): T | null {
    const service = this.services.get(name);
    return service ? (service as T) : null;
  }

  /**
   * 获取所有服务状态
   */
  public getServiceStatus(): Map<string, ServiceStatus> {
    return new Map(this.serviceStatus);
  }

  /**
   * 获取单个服务状态
   */
  public getServiceStatusByName(name: string): ServiceStatus | null {
    return this.serviceStatus.get(name) || null;
  }

  /**
   * 停止服务
   */
  public async stopService(name: string): Promise<void> {
    try {
      const service = this.services.get(name);
      if (service) {
        // 如果服务有cleanup方法，调用它
        const serviceWithCleanup = service as CleanupService;
        if (typeof serviceWithCleanup.cleanup === 'function') {
          await serviceWithCleanup.cleanup();
        }
        
        this.services.delete(name);
        this.updateServiceStatus(name, 'stopped');
        
        hilog.info(0x0000, 'ServiceManager', `服务 ${name} 已停止`);
      }
    } catch (error) {
      const err = error as BusinessError;
      hilog.error(0x0000, 'ServiceManager', `停止服务 ${name} 失败: ${err.message}`);
    }
  }

  /**
   * 停止所有服务
   */
  public async stopAllServices(): Promise<void> {
    hilog.info(0x0000, 'ServiceManager', '开始停止所有服务');

    // 停止健康检查
    if (this.healthCheckTimer !== -1) {
      clearInterval(this.healthCheckTimer);
      this.healthCheckTimer = -1;
    }

    // 按相反顺序停止服务
    const serviceNames = Array.from(this.services.keys()).reverse();
    
    for (const name of serviceNames) {
      await this.stopService(name);
    }

    this.services.clear();
    this.serviceStatus.clear();
    
    hilog.info(0x0000, 'ServiceManager', '所有服务已停止');
    this.notifyListeners();
  }

  /**
   * 更新配置
   */
  public updateConfig(config: ServiceManagerConfigUpdate): void {
    // 手动合并配置对象
    this.config = {
      enablePerformanceOptimizer: config.enablePerformanceOptimizer ?? this.config.enablePerformanceOptimizer,
      enableAIRecommendation: config.enableAIRecommendation ?? this.config.enableAIRecommendation,
      enableSmartNotification: config.enableSmartNotification ?? this.config.enableSmartNotification,
      enableDistributedSync: config.enableDistributedSync ?? this.config.enableDistributedSync,
      enableAccessibility: config.enableAccessibility ?? this.config.enableAccessibility,
      autoRestart: config.autoRestart ?? this.config.autoRestart,
      healthCheckInterval: config.healthCheckInterval ?? this.config.healthCheckInterval
    };
    
    // 如果健康检查间隔改变，重启健康检查
    if (config.healthCheckInterval && this.healthCheckTimer !== -1) {
      clearInterval(this.healthCheckTimer);
      this.healthCheckTimer = -1;
      this.startHealthCheck();
    }
    
    hilog.info(0x0000, 'ServiceManager', `配置已更新: ${JSON.stringify(config)}`);
  }

  /**
   * 获取配置
   */
  public getConfig(): ServiceManagerConfig {
    // 手动复制配置对象
    return {
      enablePerformanceOptimizer: this.config.enablePerformanceOptimizer,
      enableAIRecommendation: this.config.enableAIRecommendation,
      enableSmartNotification: this.config.enableSmartNotification,
      enableDistributedSync: this.config.enableDistributedSync,
      enableAccessibility: this.config.enableAccessibility,
      autoRestart: this.config.autoRestart,
      healthCheckInterval: this.config.healthCheckInterval
    };
  }

  /**
   * 添加状态监听器
   */
  public addStatusListener(listener: (status: Map<string, ServiceStatus>) => void): void {
    this.listeners.push(listener);
  }

  /**
   * 移除状态监听器
   */
  public removeStatusListener(listener: (status: Map<string, ServiceStatus>) => void): void {
    const index = this.listeners.indexOf(listener);
    if (index > -1) {
      this.listeners.splice(index, 1);
    }
  }

  /**
   * 通知监听器
   */
  private notifyListeners(): void {
    const status = this.getServiceStatus();
    this.listeners.forEach(listener => {
      try {
        listener(status);
      } catch (error) {
        hilog.error(0x0000, 'ServiceManager', `状态监听器执行失败: ${JSON.stringify(error)}`);
      }
    });
  }

  /**
   * 获取服务统计信息
   */
  public getServiceStats(): ServiceStats {
    const stats: ServiceStats = {} as ServiceStats;
    stats.total = this.serviceStatus.size;
    stats.running = 0;
    stats.stopped = 0;
    stats.error = 0;
    stats.initializing = 0;

    for (const status of this.serviceStatus.values()) {
      switch (status.status) {
        case 'running':
          stats.running++;
          break;
        case 'stopped':
          stats.stopped++;
          break;
        case 'error':
          stats.error++;
          break;
        case 'initializing':
          stats.initializing++;
          break;
      }
    }

    return stats;
  }

  /**
   * 生成服务报告
   */
  public generateServiceReport(): ServiceReport {
    const report: ServiceReport = {
      timestamp: Date.now(),
      config: this.getConfig(),
      services: Array.from(this.serviceStatus.values()),
      stats: this.getServiceStats()
    };
    return report;
  }
}
