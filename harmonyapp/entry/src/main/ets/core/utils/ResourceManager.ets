/**
 * 资源管理器
 * 提供统一的资源生命周期管理，防止内存泄漏
 */

import { createLogger } from './LogManager';
import { ErrorHandler, ErrorType, ErrorLevel } from './ErrorHandler';
import { TimerManager } from './TimerManager';

const logger = createLogger('ResourceManager');

/**
 * 资源接口
 */
export interface IResource {
  cleanup(): Promise<void> | void;
  isActive(): boolean;
}

/**
 * 资源信息
 */
interface ResourceInfo {
  id: string;
  resource: IResource;
  type: string;
  createdAt: number;
  lastAccess: number;
  description?: string;
}

/**
 * 资源泄漏报告
 */
export interface ResourceLeakReport {
  totalResources: number;
  leakedResources: Array<{
    id: string;
    type: string;
    age: number;
    description?: string;
  }>;
  oldestResource?: {
    id: string;
    age: number;
  };
}

/**
 * 资源统计信息
 */
export interface ResourceStats {
  totalResources: number;
  resourcesByType: Record<string, number>;
  averageAge: number;
  oldestResourceAge: number;
}

/**
 * 资源管理器
 */
export class ResourceManager {
  private static resources: Map<string, ResourceInfo> = new Map();
  private static nextId: number = 1;
  private static cleanupTimerId: number = -1;
  private static readonly MAX_RESOURCE_AGE = 24 * 60 * 60 * 1000; // 24小时
  private static readonly CLEANUP_INTERVAL = 30 * 60 * 1000; // 30分钟

  /**
   * 注册资源
   */
  static register(resource: IResource, type: string, description?: string): string {
    const id = `resource_${ResourceManager.nextId++}`;
    const now = Date.now();

    const resourceInfo: ResourceInfo = {
      id,
      resource,
      type,
      createdAt: now,
      lastAccess: now,
      description
    };

    ResourceManager.resources.set(id, resourceInfo);
    logger.debug(`注册资源: ${id} (${type}) - ${description || '无描述'}`);

    // 启动自动清理（如果还未启动）
    ResourceManager.startAutoCleanup();

    return id;
  }

  /**
   * 取消注册资源
   */
  static async unregister(id: string): Promise<boolean> {
    const resourceInfo = ResourceManager.resources.get(id);
    if (!resourceInfo) {
      logger.warn(`尝试取消注册不存在的资源: ${id}`);
      return false;
    }

    try {
      // 清理资源
      await resourceInfo.resource.cleanup();
      ResourceManager.resources.delete(id);
      logger.debug(`取消注册资源: ${id} (${resourceInfo.type})`);
      return true;
    } catch (error) {
      ErrorHandler.handle(error, `ResourceManager.unregister(${id})`, ErrorType.SYSTEM, ErrorLevel.MEDIUM);
      return false;
    }
  }

  /**
   * 更新资源访问时间
   */
  static touch(id: string): void {
    const resourceInfo = ResourceManager.resources.get(id);
    if (resourceInfo) {
      resourceInfo.lastAccess = Date.now();
    }
  }

  /**
   * 检测资源泄漏
   */
  static detectLeaks(): ResourceLeakReport {
    const now = Date.now();
    const leakedResources: Array<{
      id: string;
      type: string;
      age: number;
      description?: string;
    }> = [];

    let oldestResource: { id: string; age: number } | undefined;
    let maxAge = 0;

    for (const [id, info] of ResourceManager.resources.entries()) {
      const age = now - info.createdAt;
      
      // 检查是否为泄漏资源
      if (age > ResourceManager.MAX_RESOURCE_AGE || !info.resource.isActive()) {
        leakedResources.push({
          id,
          type: info.type,
          age,
          description: info.description
        });
      }

      // 记录最老的资源
      if (age > maxAge) {
        maxAge = age;
        oldestResource = { id, age };
      }
    }

    return {
      totalResources: ResourceManager.resources.size,
      leakedResources,
      oldestResource
    };
  }

  /**
   * 获取资源统计信息
   */
  static getStats(): ResourceStats {
    const now = Date.now();
    const resourcesByType: Record<string, number> = {};
    let totalAge = 0;
    let oldestResourceAge = 0;

    for (const info of ResourceManager.resources.values()) {
      const age = now - info.createdAt;
      
      // 按类型统计
      resourcesByType[info.type] = (resourcesByType[info.type] || 0) + 1;
      
      // 计算平均年龄
      totalAge += age;
      
      // 记录最老资源年龄
      if (age > oldestResourceAge) {
        oldestResourceAge = age;
      }
    }

    return {
      totalResources: ResourceManager.resources.size,
      resourcesByType,
      averageAge: ResourceManager.resources.size > 0 ? totalAge / ResourceManager.resources.size : 0,
      oldestResourceAge
    };
  }

  /**
   * 强制清理所有资源
   */
  static async cleanup(): Promise<void> {
    logger.info('开始清理所有资源...');
    const cleanupPromises: Promise<void>[] = [];

    for (const [id, info] of ResourceManager.resources.entries()) {
      cleanupPromises.push(
        (async () => {
          try {
            await info.resource.cleanup();
            logger.debug(`清理资源: ${id} (${info.type})`);
          } catch (error) {
            logger.error(`清理资源失败: ${id} - ${error}`);
          }
        })()
      );
    }

    try {
      await Promise.all(cleanupPromises);
      ResourceManager.resources.clear();
      ResourceManager.stopAutoCleanup();
      logger.info('所有资源清理完成');
    } catch (error) {
      ErrorHandler.handle(error, 'ResourceManager.cleanup', ErrorType.SYSTEM, ErrorLevel.HIGH);
    }
  }

  /**
   * 清理泄漏的资源
   */
  static async cleanupLeaks(): Promise<number> {
    const leakReport = ResourceManager.detectLeaks();
    let cleanedCount = 0;

    for (const leak of leakReport.leakedResources) {
      const success = await ResourceManager.unregister(leak.id);
      if (success) {
        cleanedCount++;
        logger.warn(`清理泄漏资源: ${leak.id} (${leak.type}), 年龄: ${leak.age}ms`);
      }
    }

    if (cleanedCount > 0) {
      logger.info(`清理了 ${cleanedCount} 个泄漏资源`);
    }

    return cleanedCount;
  }

  /**
   * 启动自动清理
   */
  private static startAutoCleanup(): void {
    if (ResourceManager.cleanupTimerId !== -1) {
      return; // 已经启动
    }

    ResourceManager.cleanupTimerId = TimerManager.setInterval(() => {
      ResourceManager.performAutoCleanup();
    }, ResourceManager.CLEANUP_INTERVAL, 'ResourceManager自动清理');

    logger.debug('资源自动清理已启动');
  }

  /**
   * 停止自动清理
   */
  private static stopAutoCleanup(): void {
    if (ResourceManager.cleanupTimerId !== -1) {
      TimerManager.clearInterval(ResourceManager.cleanupTimerId);
      ResourceManager.cleanupTimerId = -1;
      logger.debug('资源自动清理已停止');
    }
  }

  /**
   * 执行自动清理
   */
  private static async performAutoCleanup(): Promise<void> {
    try {
      const stats = ResourceManager.getStats();
      logger.debug(`资源健康检查: 总计 ${stats.totalResources} 个资源`);

      // 清理泄漏资源
      const cleanedCount = await ResourceManager.cleanupLeaks();
      
      // 如果资源过多，发出警告
      if (stats.totalResources > 100) {
        logger.warn(`资源数量过多: ${stats.totalResources} 个，请检查是否存在泄漏`);
      }

      // 如果有很老的资源，发出警告
      if (stats.oldestResourceAge > ResourceManager.MAX_RESOURCE_AGE) {
        logger.warn(`发现长期存在的资源，年龄: ${stats.oldestResourceAge}ms`);
      }

    } catch (error) {
      ErrorHandler.handle(error, 'ResourceManager.performAutoCleanup', ErrorType.SYSTEM, ErrorLevel.MEDIUM);
    }
  }

  /**
   * 使用资源的便捷方法（自动管理生命周期）
   */
  static async withResource<T extends IResource, R>(
    resourceFactory: () => Promise<T> | T,
    operation: (resource: T) => Promise<R> | R,
    resourceType: string = 'temporary',
    description?: string
  ): Promise<R> {
    let resource: T;
    let resourceId: string | null = null;

    try {
      // 创建资源
      resource = await resourceFactory();
      resourceId = ResourceManager.register(resource, resourceType, description);

      // 执行操作
      const result = await operation(resource);
      return result;

    } finally {
      // 确保资源被清理
      if (resourceId) {
        await ResourceManager.unregister(resourceId);
      }
    }
  }

  /**
   * 获取所有资源信息（用于调试）
   */
  static getAllResources(): Array<{
    id: string;
    type: string;
    age: number;
    description?: string;
    isActive: boolean;
  }> {
    const now = Date.now();
    const result: Array<{
      id: string;
      type: string;
      age: number;
      description?: string;
      isActive: boolean;
    }> = [];

    for (const [id, info] of ResourceManager.resources.entries()) {
      result.push({
        id,
        type: info.type,
        age: now - info.createdAt,
        description: info.description,
        isActive: info.resource.isActive()
      });
    }

    return result.sort((a, b) => b.age - a.age); // 按年龄降序排列
  }
}

/**
 * 便捷的资源创建函数
 */
export function createManagedResource<T extends IResource>(
  resource: T,
  type: string,
  description?: string
): string {
  return ResourceManager.register(resource, type, description);
}

/**
 * 便捷的资源清理函数
 */
export async function cleanupManagedResource(id: string): Promise<boolean> {
  return await ResourceManager.unregister(id);
}
