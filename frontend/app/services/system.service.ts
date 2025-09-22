/**
 * 系统状态服务 - Composition API 风格
 * 提供系统状态检查相关的所有 API 调用功能
 * 
 * 基于 backend_doc.md 5.2.9 系统状态API
 */

import { 
  STATUS_ENDPOINTS, 
  buildStatusUrl 
} from "~/config/api";
import type { 
  ApiResponse,
  HealthCheckRequest,
  HealthCheckResponse
} from "~/types/api";

export const useSystemService = () => {
  const api = useApi();

  /**
   * 系统健康检查
   */
  const healthCheck = async (): Promise<HealthCheckResponse> => {
    const request: HealthCheckRequest = {};

    const response = await api.post<ApiResponse<HealthCheckResponse>>(
      buildStatusUrl(STATUS_ENDPOINTS.PUBLIC.HEALTH, true),
      request
    );

    return response.data!;
  };

  /**
   * 获取系统状态摘要
   */
  const getSystemSummary = async (): Promise<{
    status: string;
    uptime: number;
    version: string;
    environment: string;
  }> => {
    try {
      const health = await healthCheck();
      return {
        status: health.status,
        uptime: health.uptime || 0,
        version: health.version || 'unknown',
        environment: health.environment || 'unknown'
      };
    } catch (error) {
      return {
        status: 'error',
        uptime: 0,
        version: 'unknown',
        environment: 'unknown'
      };
    }
  };

  /**
   * 检查系统是否可用
   */
  const isSystemAvailable = async (): Promise<boolean> => {
    try {
      const health = await healthCheck();
      return health.status === 'healthy';
    } catch (error) {
      return false;
    }
  };

  /**
   * 获取系统状态显示文本
   */
  const getStatusDisplayText = (status: string): string => {
    const statusMap: Record<string, string> = {
      'healthy': '正常',
      'degraded': '降级',
      'unhealthy': '异常',
      'maintenance': '维护中',
      'error': '错误'
    };
    return statusMap[status] || '未知';
  };

  /**
   * 获取系统状态显示样式
   */
  const getStatusDisplayStyle = (status: string) => {
    const statusStyles: Record<string, { color: string; bgColor: string; icon: string }> = {
      'healthy': { 
        color: '#34c759', 
        bgColor: '#e6f7ea', 
        icon: '✅' 
      },
      'degraded': { 
        color: '#ff9500', 
        bgColor: '#fff5e6', 
        icon: '⚠️' 
      },
      'unhealthy': { 
        color: '#ff3b30', 
        bgColor: '#ffe6e6', 
        icon: '❌' 
      },
      'maintenance': { 
        color: '#007aff', 
        bgColor: '#e6f3ff', 
        icon: '🔧' 
      },
      'error': { 
        color: '#ff3b30', 
        bgColor: '#ffe6e6', 
        icon: '💥' 
      }
    };
    return statusStyles[status] || statusStyles.error;
  };

  /**
   * 格式化运行时间
   */
  const formatUptime = (uptimeSeconds: number): string => {
    if (uptimeSeconds < 60) {
      return `${Math.floor(uptimeSeconds)}秒`;
    } else if (uptimeSeconds < 3600) {
      const minutes = Math.floor(uptimeSeconds / 60);
      return `${minutes}分钟`;
    } else if (uptimeSeconds < 86400) {
      const hours = Math.floor(uptimeSeconds / 3600);
      const minutes = Math.floor((uptimeSeconds % 3600) / 60);
      return `${hours}小时${minutes}分钟`;
    } else {
      const days = Math.floor(uptimeSeconds / 86400);
      const hours = Math.floor((uptimeSeconds % 86400) / 3600);
      return `${days}天${hours}小时`;
    }
  };

  /**
   * 定期健康检查
   */
  const startHealthMonitoring = (
    intervalMs: number = 30000, // 默认30秒
    onStatusChange?: (status: HealthCheckResponse) => void
  ): () => void => {
    let lastStatus: string | null = null;
    
    const checkHealth = async () => {
      try {
        const health = await healthCheck();
        
        // 如果状态发生变化，触发回调
        if (onStatusChange && health.status !== lastStatus) {
          onStatusChange(health);
          lastStatus = health.status;
        }
      } catch (error) {
        console.warn('健康检查失败:', error);
        
        // 如果之前是正常状态，现在失败了，触发回调
        if (onStatusChange && lastStatus !== 'error') {
          onStatusChange({
            status: 'error',
            message: '系统连接失败',
            timestamp: new Date().toISOString()
          });
          lastStatus = 'error';
        }
      }
    };

    // 立即执行一次检查
    checkHealth();
    
    // 设置定期检查
    const intervalId = setInterval(checkHealth, intervalMs);
    
    // 返回停止监控的函数
    return () => {
      clearInterval(intervalId);
    };
  };

  return {
    // 核心API方法
    healthCheck,
    
    // 便捷方法
    getSystemSummary,
    isSystemAvailable,
    
    // 工具方法
    getStatusDisplayText,
    getStatusDisplayStyle,
    formatUptime,
    
    // 监控方法
    startHealthMonitoring
  };
};

/**
 * 系统状态相关的常量定义
 */
export const SYSTEM_CONSTANTS = {
  // 系统状态
  STATUSES: {
    HEALTHY: 'healthy',
    DEGRADED: 'degraded',
    UNHEALTHY: 'unhealthy',
    MAINTENANCE: 'maintenance',
    ERROR: 'error'
  },
  
  // 默认监控间隔（毫秒）
  DEFAULT_MONITORING_INTERVAL: 30000, // 30秒
  
  // 健康检查超时时间（毫秒）
  HEALTH_CHECK_TIMEOUT: 5000, // 5秒
  
  // 系统环境
  ENVIRONMENTS: {
    DEVELOPMENT: 'development',
    STAGING: 'staging',
    PRODUCTION: 'production'
  }
} as const;
