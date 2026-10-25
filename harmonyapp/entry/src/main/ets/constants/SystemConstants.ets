/**
 * 系统级配置常量
 * 统一管理所有硬编码的配置值和魔法数字
 */
import { hilog } from '@kit.PerformanceAnalysisKit';

/**
 * 缓存配置接口
 */
interface CacheConfig {
  DEFAULT_SIZE: number;
  MAX_SIZE: number;
  DEFAULT_EXPIRY: number;
  CLEANUP_INTERVAL: number;
  MAX_ITEM_SIZE: number;
}

/**
 * 阈值配置接口
 */
interface ThresholdConfig {
  MEMORY_WARNING: number;
  MEMORY_CRITICAL: number;
  CPU_WARNING: number;
  CPU_CRITICAL: number;
  NETWORK_SLOW: number;
  NETWORK_TIMEOUT: number;
  FRAME_RATE_MIN: number;
  BATTERY_LOW: number;
  ERROR_RATE_HIGH: number;
}

/**
 * 间隔配置接口
 */
interface IntervalConfig {
  HEALTH_CHECK: number;
  PERFORMANCE_CHECK: number;
  METRICS_UPDATE: number;
  ALERT_THROTTLE: number;
}

/**
 * 性能常量接口
 */
interface PerformanceConstants {
  CACHE: CacheConfig;
  THRESHOLDS: ThresholdConfig;
  INTERVALS: IntervalConfig;
}

/**
 * 超时配置接口
 */
interface TimeoutConfig {
  REQUEST: number;
  CONNECT: number;
  READ: number;
  WRITE: number;
}

/**
 * 重试配置接口
 */
interface RetryConfig {
  MAX_ATTEMPTS: number;
  DELAY: number;
  BACKOFF_MULTIPLIER: number;
  MAX_DELAY: number;
}

/**
 * 并发配置接口
 */
interface ConcurrencyConfig {
  MAX_REQUESTS: number;
  MAX_CONNECTIONS: number;
  POOL_SIZE: number;
}

/**
 * 网络常量接口
 */
interface NetworkConstants {
  TIMEOUTS: TimeoutConfig;
  RETRY: RetryConfig;
  CONCURRENCY: ConcurrencyConfig;
}

/**
 * 存储缓存配置接口
 */
interface StorageCacheConfig {
  MAX_SIZE: number;
  DEFAULT_TTL: number;
  CLEANUP_THRESHOLD: number;
}

/**
 * 存储限制配置接口
 */
interface StorageLimitConfig {
  MAX_KEY_LENGTH: number;
  MAX_VALUE_SIZE: number;
  MAX_ENTRIES: number;
}

/**
 * 批处理配置接口
 */
interface BatchConfig {
  SIZE: number;
  TIMEOUT: number;
  MAX_WAIT: number;
}

/**
 * 存储常量接口
 */
interface StorageConstants {
  CACHE: StorageCacheConfig;
  LIMITS: StorageLimitConfig;
  BATCH: BatchConfig;
}

/**
 * 日志级别配置接口
 */
interface LogLevelConfig {
  DEBUG: number;
  INFO: number;
  WARN: number;
  ERROR: number;
  FATAL: number;
}

/**
 * 日志配置接口
 */
interface LogConfig {
  MAX_FILE_SIZE: number;
  MAX_FILES: number;
  FLUSH_INTERVAL: number;
  BUFFER_SIZE: number;
}

/**
 * 日志域名配置接口
 */
interface LogDomainConfig {
  BASE: number;
  MAX_MODULES: number;
}

/**
 * 日志常量接口
 */
interface LogConstants {
  LEVELS: LogLevelConfig;
  CONFIG: LogConfig;
  DOMAINS: LogDomainConfig;
}

/**
 * 通知限制配置接口
 */
interface NotificationLimitConfig {
  MAX_NOTIFICATIONS: number;
  MAX_TITLE_LENGTH: number;
  MAX_MESSAGE_LENGTH: number;
  MAX_ACTIONS: number;
}

/**
 * 通知清理配置接口
 */
interface NotificationCleanupConfig {
  AUTO_CLEANUP_DAYS: number;
  CLEANUP_INTERVAL: number;
  MAX_READ_NOTIFICATIONS: number;
}

/**
 * 通知优先级配置接口
 */
interface NotificationPriorityConfig {
  LOW: number;
  MEDIUM: number;
  HIGH: number;
  CRITICAL: number;
}

/**
 * 通知常量接口
 */
interface NotificationConstants {
  LIMITS: NotificationLimitConfig;
  CLEANUP: NotificationCleanupConfig;
  PRIORITY: NotificationPriorityConfig;
}

/**
 * 动画配置接口
 */
interface AnimationConfig {
  DURATION_FAST: number;
  DURATION_NORMAL: number;
  DURATION_SLOW: number;
  EASING: string;
}

/**
 * 尺寸配置接口
 */
interface SizeConfig {
  ICON_SMALL: number;
  ICON_MEDIUM: number;
  ICON_LARGE: number;
  AVATAR_SMALL: number;
  AVATAR_MEDIUM: number;
  AVATAR_LARGE: number;
}

/**
 * 间距配置接口
 */
interface SpacingConfig {
  XS: number;
  SM: number;
  MD: number;
  LG: number;
  XL: number;
}

/**
 * UI常量接口
 */

/**
 * 图片配置接口
 */
interface ImageConfig {
  MAX_SIZE: number;
  MAX_COUNT: number;
  QUALITY_HIGH: number;
  QUALITY_MEDIUM: number;
  QUALITY_LOW: number;
  THUMBNAIL_SIZE: number;
}

/**
 * 格式配置接口
 */
interface FormatConfig {
  IMAGE: string[];
  VIDEO: string[];
  AUDIO: string[];
}

/**
 * 媒体常量接口
 */
interface MediaConstants {
  IMAGE: ImageConfig;
  FORMATS: FormatConfig;
}

/**
 * 密码配置接口
 */
interface PasswordConfig {
  MIN_LENGTH: number;
  MAX_LENGTH: number;
  REQUIRE_UPPERCASE: boolean;
  REQUIRE_LOWERCASE: boolean;
  REQUIRE_NUMBERS: boolean;
  REQUIRE_SYMBOLS: boolean;
}

/**
 * 会话配置接口
 */
interface SessionConfig {
  TIMEOUT: number;
  REFRESH_THRESHOLD: number;
  MAX_CONCURRENT: number;
}

/**
 * 加密配置接口
 */
interface EncryptionConfig {
  KEY_SIZE: number;
  SALT_SIZE: number;
  ITERATIONS: number;
}

/**
 * 安全常量接口
 */
interface SecurityConstants {
  PASSWORD: PasswordConfig;
  SESSION: SessionConfig;
  ENCRYPTION: EncryptionConfig;
}

/**
 * 精度配置接口
 */
interface AccuracyConfig {
  HIGH: number;
  MEDIUM: number;
  LOW: number;
  HIGH_ACCURACY_THRESHOLD?: number;
}

/**
 * 位置缓存配置接口
 */
interface LocationCacheConfig {
  TTL: number;
  MAX_ENTRIES: number;
  MIN_DISTANCE: number;
}

/**
 * 搜索配置接口
 */
interface SearchConfig {
  DEFAULT_RADIUS: number;
  MAX_RADIUS: number;
  MIN_RADIUS: number;
}

/**
 * 位置常量接口
 */
interface LocationConstants {
  ACCURACY: AccuracyConfig;
  CACHE: LocationCacheConfig;
  SEARCH: SearchConfig;
}

/**
 * 系统常量接口
 */
interface SystemConstantsInterface {
  PERFORMANCE: PerformanceConstants;
  NETWORK: NetworkConstants;
  STORAGE: StorageConstants;
  LOG: LogConstants;
  NOTIFICATION: NotificationConstants;
  MEDIA: MediaConstants;
  SECURITY: SecurityConstants;
  LOCATION: LocationConstants;
}

/**
 * 性能相关常量
 */
export const PERFORMANCE_CONSTANTS: PerformanceConstants = {
  // 缓存配置
  CACHE: {
    DEFAULT_SIZE: 100,
    MAX_SIZE: 1000,
    DEFAULT_EXPIRY: 5 * 60 * 1000,      // 5分钟
    CLEANUP_INTERVAL: 30 * 60 * 1000,   // 30分钟
    MAX_ITEM_SIZE: 1024 * 1024,         // 1MB
  },
  
  // 性能阈值
  THRESHOLDS: {
    MEMORY_WARNING: 80,      // 内存使用率警告阈值 (%)
    MEMORY_CRITICAL: 90,     // 内存使用率严重阈值 (%)
    CPU_WARNING: 70,         // CPU使用率警告阈值 (%)
    CPU_CRITICAL: 85,        // CPU使用率严重阈值 (%)
    NETWORK_SLOW: 500,       // 网络延迟慢阈值 (ms)
    NETWORK_TIMEOUT: 1000,   // 网络延迟超时阈值 (ms)
    FRAME_RATE_MIN: 30,      // 最低帧率
    BATTERY_LOW: 20,         // 低电量阈值 (%)
    ERROR_RATE_HIGH: 5,      // 高错误率阈值 (%)
  },
  
  // 监控间隔
  INTERVALS: {
    HEALTH_CHECK: 60 * 1000,           // 健康检查间隔 (1分钟)
    PERFORMANCE_CHECK: 5 * 60 * 1000,  // 性能检查间隔 (5分钟)
    METRICS_UPDATE: 30 * 1000,         // 指标更新间隔 (30秒)
    ALERT_THROTTLE: 2 * 60 * 1000,     // 告警节流间隔 (2分钟)
  }
};

/**
 * 网络相关常量
 */
class NetworkTimeoutsImpl implements TimeoutConfig {
  REQUEST: number = 30 * 1000;        // 请求超时 (30秒)
  CONNECT: number = 10 * 1000;        // 连接超时 (10秒)
  READ: number = 15 * 1000;           // 读取超时 (15秒)
  WRITE: number = 10 * 1000;          // 写入超时 (10秒)
}

const NETWORK_TIMEOUTS: TimeoutConfig = new NetworkTimeoutsImpl();

const NETWORK_RETRY: RetryConfig = {
  MAX_ATTEMPTS: 3,           // 最大重试次数
  DELAY: 1000,               // 重试延迟 (1秒)
  BACKOFF_MULTIPLIER: 2,     // 退避倍数
  MAX_DELAY: 10 * 1000,      // 最大重试延迟 (10秒)
};

const NETWORK_CONCURRENCY: ConcurrencyConfig = {
  MAX_REQUESTS: 5,           // 最大并发请求数
  MAX_CONNECTIONS: 10,       // 最大连接数
  POOL_SIZE: 20,             // 连接池大小
};

export const NETWORK_CONSTANTS: NetworkConstants = {
  // 超时配置
  TIMEOUTS: NETWORK_TIMEOUTS,
  
  // 重试配置
  RETRY: NETWORK_RETRY,
  
  // 并发配置
  CONCURRENCY: NETWORK_CONCURRENCY
};

/**
 * 存储相关常量
 */
export const STORAGE_CONSTANTS: StorageConstants = {
  // 缓存配置
  CACHE: {
    MAX_SIZE: 50 * 1024 * 1024,        // 最大缓存大小 (50MB)
    DEFAULT_TTL: 24 * 60 * 60 * 1000,  // 默认生存时间 (24小时)
    CLEANUP_THRESHOLD: 0.8,            // 清理阈值 (80%)
  },
  
  // 存储限制
  LIMITS: {
    MAX_KEY_LENGTH: 256,               // 最大键长度
    MAX_VALUE_SIZE: 10 * 1024 * 1024,  // 最大值大小 (10MB)
    MAX_ENTRIES: 10000,                // 最大条目数
  },
  
  // 批处理配置
  BATCH: {
    SIZE: 100,                         // 批处理大小
    TIMEOUT: 5 * 1000,                 // 批处理超时 (5秒)
    MAX_WAIT: 1000,                    // 最大等待时间 (1秒)
  }
};

/**
 * 日志相关常量
 */
export const LOG_CONSTANTS: LogConstants = {
  // 日志级别
  LEVELS: {
    DEBUG: 0,
    INFO: 1,
    WARN: 2,
    ERROR: 3,
    FATAL: 4,
  },
  
  // 日志配置
  CONFIG: {
    MAX_FILE_SIZE: 10 * 1024 * 1024,   // 最大日志文件大小 (10MB)
    MAX_FILES: 5,                      // 最大日志文件数
    FLUSH_INTERVAL: 5 * 1000,          // 刷新间隔 (5秒)
    BUFFER_SIZE: 1024,                 // 缓冲区大小
  },
  
  // 域名分配
  DOMAINS: {
    BASE: 0xFF00,                      // 基础域名
    MAX_MODULES: 255,                  // 最大模块数
  }
};

/**
 * 通知相关常量
 */
export const NOTIFICATION_CONSTANTS: NotificationConstants = {
  // 通知限制
  LIMITS: {
    MAX_NOTIFICATIONS: 100,            // 最大通知数
    MAX_TITLE_LENGTH: 50,              // 最大标题长度
    MAX_MESSAGE_LENGTH: 200,           // 最大消息长度
    MAX_ACTIONS: 3,                    // 最大操作数
  },
  
  // 清理配置
  CLEANUP: {
    AUTO_CLEANUP_DAYS: 7,              // 自动清理天数
    CLEANUP_INTERVAL: 24 * 60 * 60 * 1000, // 清理间隔 (24小时)
    MAX_READ_NOTIFICATIONS: 50,        // 最大已读通知数
  },
  
  // 优先级配置
  PRIORITY: {
    LOW: 1,
    MEDIUM: 2,
    HIGH: 3,
    CRITICAL: 4,
  }
};

/**
 * UI相关常量
 */

/**
 * 媒体相关常量
 */
export const MEDIA_CONSTANTS: MediaConstants = {
  // 图片配置
  IMAGE: {
    MAX_SIZE: 10 * 1024 * 1024,        // 最大图片大小 (10MB)
    MAX_COUNT: 9,                      // 最大图片数量
    QUALITY_HIGH: 90,                  // 高质量压缩
    QUALITY_MEDIUM: 70,                // 中等质量压缩
    QUALITY_LOW: 50,                   // 低质量压缩
    THUMBNAIL_SIZE: 200,               // 缩略图尺寸
  },
  
  // 支持的格式
  FORMATS: {
    IMAGE: ['jpg', 'jpeg', 'png', 'webp'],
    VIDEO: ['mp4', 'mov', 'avi'],
    AUDIO: ['mp3', 'wav', 'aac'],
  }
};

/**
 * 安全相关常量
 */
export const SECURITY_CONSTANTS: SecurityConstants = {
  // 密码配置
  PASSWORD: {
    MIN_LENGTH: 8,                     // 最小密码长度
    MAX_LENGTH: 128,                   // 最大密码长度
    REQUIRE_UPPERCASE: true,           // 需要大写字母
    REQUIRE_LOWERCASE: true,           // 需要小写字母
    REQUIRE_NUMBERS: true,             // 需要数字
    REQUIRE_SYMBOLS: false,            // 需要符号
  },
  
  // 会话配置
  SESSION: {
    TIMEOUT: 30 * 60 * 1000,           // 会话超时 (30分钟)
    REFRESH_THRESHOLD: 5 * 60 * 1000,  // 刷新阈值 (5分钟)
    MAX_CONCURRENT: 3,                 // 最大并发会话
  },
  
  // 加密配置
  ENCRYPTION: {
    KEY_SIZE: 256,                     // 密钥大小 (bits)
    SALT_SIZE: 32,                     // 盐值大小 (bytes)
    ITERATIONS: 10000,                 // 迭代次数
  }
};

/**
 * 地理位置相关常量
 */
export const LOCATION_CONSTANTS: LocationConstants = {
  // 精度配置
  ACCURACY: {
    HIGH: 10,                          // 高精度 (米)
    MEDIUM: 100,                       // 中等精度 (米)
    LOW: 1000,                         // 低精度 (米)
  },
  
  // 缓存配置
  CACHE: {
    TTL: 5 * 60 * 1000,                // 缓存生存时间 (5分钟)
    MAX_ENTRIES: 100,                  // 最大缓存条目
    MIN_DISTANCE: 50,                  // 最小距离变化 (米)
  },
  
  // 搜索配置
  SEARCH: {
    DEFAULT_RADIUS: 2000,              // 默认搜索半径 (米)
    MAX_RADIUS: 50000,                 // 最大搜索半径 (50公里)
    MIN_RADIUS: 100,                   // 最小搜索半径 (米)
  }
};

/**
 * 获取所有常量的统一接口
 */
export const SYSTEM_CONSTANTS: SystemConstantsInterface = {
  PERFORMANCE: PERFORMANCE_CONSTANTS,
  NETWORK: NETWORK_CONSTANTS,
  STORAGE: STORAGE_CONSTANTS,
  LOG: LOG_CONSTANTS,
  NOTIFICATION: NOTIFICATION_CONSTANTS,
  MEDIA: MEDIA_CONSTANTS,
  SECURITY: SECURITY_CONSTANTS,
  LOCATION: LOCATION_CONSTANTS,
};

/**
 * 常量验证函数
 */
export class ConstantValidator {
  /**
   * 验证所有常量的有效性
   */
  static validateAll(): boolean {
    try {
      // 验证性能常量
      ConstantValidator.validatePerformanceConstants();
      
      // 验证网络常量
      ConstantValidator.validateNetworkConstants();
      
      // 验证存储常量
      ConstantValidator.validateStorageConstants();
      
      return true;
    } catch (error) {
      hilog.error(0xFF00, 'SystemConstants', '常量验证失败: %{public}s', String(error));
      return false;
    }
  }
  
  private static validatePerformanceConstants(): void {
    const perf = PERFORMANCE_CONSTANTS;
    
    if (perf.CACHE.DEFAULT_SIZE <= 0 || perf.CACHE.DEFAULT_SIZE > perf.CACHE.MAX_SIZE) {
      throw new Error('缓存大小配置无效');
    }
    
    if (perf.THRESHOLDS.MEMORY_WARNING >= perf.THRESHOLDS.MEMORY_CRITICAL) {
      throw new Error('内存阈值配置无效');
    }
    
    if (perf.THRESHOLDS.CPU_WARNING >= perf.THRESHOLDS.CPU_CRITICAL) {
      throw new Error('CPU阈值配置无效');
    }
  }
  
  private static validateNetworkConstants(): void {
    const network = NETWORK_CONSTANTS;
    
    if (network.RETRY.MAX_ATTEMPTS <= 0) {
      throw new Error('重试次数配置无效');
    }
    
    if (network.TIMEOUTS.CONNECT >= network.TIMEOUTS.REQUEST) {
      throw new Error('超时配置无效');
    }
  }
  
  private static validateStorageConstants(): void {
    const storage = STORAGE_CONSTANTS;
    
    if (storage.LIMITS.MAX_KEY_LENGTH <= 0) {
      throw new Error('存储键长度配置无效');
    }
    
    if (storage.CACHE.CLEANUP_THRESHOLD <= 0 || storage.CACHE.CLEANUP_THRESHOLD >= 1) {
      throw new Error('清理阈值配置无效');
    }
  }
}
