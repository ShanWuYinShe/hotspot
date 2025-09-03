/**
 * 前端统一错误类型定义
 * 规范化所有客户端错误处理
 */

// =============================================================================
// 1. 基础错误类型枚举
// =============================================================================

export enum ErrorType {
  // API相关错误
  API_ERROR = 'API_ERROR',
  NETWORK_ERROR = 'NETWORK_ERROR', 
  TIMEOUT_ERROR = 'TIMEOUT_ERROR',
  
  // 认证授权错误
  UNAUTHORIZED = 'UNAUTHORIZED',
  FORBIDDEN = 'FORBIDDEN',
  
  // 客户端特定错误
  GEOLOCATION_ERROR = 'GEOLOCATION_ERROR',
  PERMISSION_ERROR = 'PERMISSION_ERROR',
  FILE_ERROR = 'FILE_ERROR',
  WEBSOCKET_ERROR = 'WEBSOCKET_ERROR',
  
  // 业务逻辑错误
  VALIDATION_ERROR = 'VALIDATION_ERROR',
  BUSINESS_ERROR = 'BUSINESS_ERROR',
  
  // 系统错误
  UNKNOWN_ERROR = 'UNKNOWN_ERROR',
  CLIENT_ERROR = 'CLIENT_ERROR'
}

// =============================================================================
// 2. 地理位置错误细分
// =============================================================================

export enum GeolocationErrorCode {
  PERMISSION_DENIED = 1,
  POSITION_UNAVAILABLE = 2, 
  TIMEOUT = 3,
  NOT_SUPPORTED = 'NOT_SUPPORTED',
  UNKNOWN = 'UNKNOWN'
}

export interface GeolocationErrorInfo {
  code: GeolocationErrorCode;
  message: string;
  userFriendlyMessage: string;
  canRetry: boolean;
  requiresUserAction: boolean;
}

// =============================================================================
// 3. 权限错误细分
// =============================================================================

export enum PermissionErrorCode {
  DENIED = 'DENIED',
  NOT_SUPPORTED = 'NOT_SUPPORTED',
  PROMPT = 'PROMPT',
  UNKNOWN = 'UNKNOWN'
}

export interface PermissionErrorInfo {
  permission: 'geolocation' | 'notification' | 'camera' | 'microphone';
  state: 'granted' | 'denied' | 'prompt' | 'unsupported';
  code: PermissionErrorCode;
  message: string;
  userFriendlyMessage: string;
  actionRequired: string | null;
}

// =============================================================================
// 4. 文件操作错误
// =============================================================================

export enum FileErrorCode {
  TOO_LARGE = 'TOO_LARGE',
  INVALID_TYPE = 'INVALID_TYPE',
  READ_FAILED = 'READ_FAILED',
  UPLOAD_FAILED = 'UPLOAD_FAILED',
  NOT_SUPPORTED = 'NOT_SUPPORTED'
}

export interface FileErrorInfo {
  code: FileErrorCode;
  file?: File;
  maxSize?: number;
  allowedTypes?: string[];
  message: string;
  userFriendlyMessage: string;
}

// =============================================================================
// 5. WebSocket错误
// =============================================================================

export enum WebSocketErrorCode {
  CONNECTION_FAILED = 'CONNECTION_FAILED',
  CONNECTION_LOST = 'CONNECTION_LOST',
  AUTHENTICATION_FAILED = 'AUTHENTICATION_FAILED',
  MESSAGE_FAILED = 'MESSAGE_FAILED',
  PROTOCOL_ERROR = 'PROTOCOL_ERROR'
}

export interface WebSocketErrorInfo {
  code: WebSocketErrorCode;
  originalEvent?: Event;
  reconnectAvailable: boolean;
  message: string;
  userFriendlyMessage: string;
}

// =============================================================================
// 6. 统一错误对象
// =============================================================================

export interface StandardError {
  type: ErrorType;
  code: string | number;
  message: string;
  userFriendlyMessage: string;
  
  // 上下文信息
  timestamp: Date;
  source: string; // 错误来源（组件/页面/服务）
  operation?: string; // 正在执行的操作
  
  // 原始错误信息（调试用）
  originalError?: any;
  
  // 特定类型的详细信息
  geolocation?: GeolocationErrorInfo;
  permission?: PermissionErrorInfo;
  file?: FileErrorInfo;
  websocket?: WebSocketErrorInfo;
  
  // 错误处理选项
  canRetry: boolean;
  requiresUserAction: boolean;
  actionText?: string; // 用户需要采取的行动说明
  
  // 通知选项
  shouldNotify: boolean;
  notificationLevel: 'info' | 'warning' | 'error';
  dedupeKey?: string;
}

// =============================================================================
// 7. 错误工厂函数类型
// =============================================================================

export type ErrorFactory = (params: {
  message: string;
  source: string;
  operation?: string;
  originalError?: any;
  canRetry?: boolean;
  requiresUserAction?: boolean;
  shouldNotify?: boolean;
}) => StandardError;

// =============================================================================
// 8. 错误处理器类型
// =============================================================================

export interface ErrorHandler {
  handle(error: StandardError): Promise<void>;
  canHandle(error: any): boolean;
  priority: number; // 处理器优先级
}

// =============================================================================
// 9. 错误恢复策略
// =============================================================================

export interface RecoveryStrategy {
  canRecover(error: StandardError): boolean;
  recover(error: StandardError): Promise<boolean>;
  getRecoveryMessage(error: StandardError): string;
}
