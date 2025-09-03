/**
 * 统一错误工厂
 * 将各种类型的错误转换为标准化的 StandardError
 */

import { 
  ErrorType, 
  GeolocationErrorCode, 
  PermissionErrorCode,
  FileErrorCode,
  WebSocketErrorCode,
  type StandardError,
  type GeolocationErrorInfo,
  type PermissionErrorInfo,
  type FileErrorInfo,
  type WebSocketErrorInfo
} from '~/types/errors';

// =============================================================================
// 1. 地理位置错误工厂
// =============================================================================

export const createGeolocationError = (
  originalError: GeolocationPositionError | Error,
  source: string,
  operation: string = 'getCurrentPosition'
): StandardError => {
  let code: GeolocationErrorCode = GeolocationErrorCode.UNKNOWN;
  let message = '获取位置失败';
  let userFriendlyMessage = '无法获取您的位置信息';
  let canRetry = true;
  let requiresUserAction = false;

  // 处理 GeolocationPositionError
  if ('code' in originalError && typeof originalError.code === 'number') {
    switch (originalError.code) {
      case 1: // PERMISSION_DENIED
        code = GeolocationErrorCode.PERMISSION_DENIED;
        message = '位置权限被拒绝';
        userFriendlyMessage = '需要位置权限才能使用此功能，请在浏览器设置中允许位置访问';
        canRetry = false;
        requiresUserAction = true;
        break;
      case 2: // POSITION_UNAVAILABLE
        code = GeolocationErrorCode.POSITION_UNAVAILABLE;
        message = '位置信息不可用';
        userFriendlyMessage = '无法获取位置信息，请确保GPS已开启';
        canRetry = true;
        break;
      case 3: // TIMEOUT
        code = GeolocationErrorCode.TIMEOUT;
        message = '获取位置超时';
        userFriendlyMessage = '获取位置超时，请重试';
        canRetry = true;
        break;
    }
  } else {
    // 处理其他类型的错误
    message = originalError.message || message;
    userFriendlyMessage = originalError.message || userFriendlyMessage;
  }

  const geolocationInfo: GeolocationErrorInfo = {
    code,
    message,
    userFriendlyMessage,
    canRetry,
    requiresUserAction
  };

  return {
    type: ErrorType.GEOLOCATION_ERROR,
    code,
    message,
    userFriendlyMessage,
    timestamp: new Date(),
    source,
    operation,
    originalError,
    geolocation: geolocationInfo,
    canRetry,
    requiresUserAction,
    actionText: requiresUserAction ? '请允许位置权限' : undefined,
    shouldNotify: true,
    notificationLevel: requiresUserAction ? 'warning' : 'error',
    dedupeKey: `geolocation-${code}-${source}`
  };
};

// =============================================================================
// 2. 权限错误工厂  
// =============================================================================

export const createPermissionError = (
  permission: 'geolocation' | 'notification' | 'camera' | 'microphone',
  state: 'granted' | 'denied' | 'prompt' | 'unsupported',
  source: string,
  operation: string = 'requestPermission'
): StandardError => {
  let code: PermissionErrorCode = PermissionErrorCode.UNKNOWN;
  let message = '权限请求失败';
  let userFriendlyMessage = '无法获取所需权限';
  let actionRequired: string | null = null;

  switch (state) {
    case 'denied':
      code = PermissionErrorCode.DENIED;
      message = `${permission}权限被拒绝`;
      userFriendlyMessage = `需要${permission === 'geolocation' ? '位置' : permission}权限才能使用此功能`;
      actionRequired = '请在浏览器设置中允许权限访问';
      break;
    case 'unsupported':
      code = PermissionErrorCode.NOT_SUPPORTED;
      message = `浏览器不支持${permission}权限`;
      userFriendlyMessage = '您的浏览器不支持此功能';
      actionRequired = '请升级到最新版本的浏览器';
      break;
    case 'prompt':
      code = PermissionErrorCode.PROMPT;
      message = `需要用户授权${permission}权限`;
      userFriendlyMessage = '请允许权限访问';
      actionRequired = '点击浏览器权限提示中的"允许"';
      break;
  }

  const permissionInfo: PermissionErrorInfo = {
    permission,
    state,
    code,
    message,
    userFriendlyMessage,
    actionRequired
  };

  return {
    type: ErrorType.PERMISSION_ERROR,
    code,
    message,
    userFriendlyMessage,
    timestamp: new Date(),
    source,
    operation,
    permission: permissionInfo,
    canRetry: state !== 'unsupported',
    requiresUserAction: state === 'denied' || state === 'prompt',
    actionText: actionRequired || undefined,
    shouldNotify: true,
    notificationLevel: state === 'unsupported' ? 'error' : 'warning',
    dedupeKey: `permission-${permission}-${state}-${source}`
  };
};

// =============================================================================
// 3. 文件操作错误工厂
// =============================================================================

export const createFileError = (
  errorCode: FileErrorCode,
  source: string,
  options: {
    file?: File;
    maxSize?: number;
    allowedTypes?: string[];
    originalError?: Error;
    operation?: string;
  } = {}
): StandardError => {
  let message = '文件操作失败';
  let userFriendlyMessage = '文件处理出现问题';
  let canRetry = true;

  switch (errorCode) {
    case FileErrorCode.TOO_LARGE:
      message = '文件过大';
      userFriendlyMessage = `文件大小超过限制${options.maxSize ? ` (${Math.round(options.maxSize / 1024 / 1024)}MB)` : ''}`;
      canRetry = false;
      break;
    case FileErrorCode.INVALID_TYPE:
      message = '文件类型不支持';
      userFriendlyMessage = `请选择支持的文件类型${options.allowedTypes ? `: ${options.allowedTypes.join(', ')}` : ''}`;
      canRetry = false;
      break;
    case FileErrorCode.READ_FAILED:
      message = '文件读取失败';
      userFriendlyMessage = '无法读取文件内容，请重试';
      canRetry = true;
      break;
    case FileErrorCode.UPLOAD_FAILED:
      message = '文件上传失败';
      userFriendlyMessage = '上传失败，请检查网络连接后重试';
      canRetry = true;
      break;
    case FileErrorCode.NOT_SUPPORTED:
      message = '不支持文件操作';
      userFriendlyMessage = '您的浏览器不支持此文件操作';
      canRetry = false;
      break;
  }

  const fileInfo: FileErrorInfo = {
    code: errorCode,
    file: options.file,
    maxSize: options.maxSize,
    allowedTypes: options.allowedTypes,
    message,
    userFriendlyMessage
  };

  return {
    type: ErrorType.FILE_ERROR,
    code: errorCode,
    message,
    userFriendlyMessage,
    timestamp: new Date(),
    source,
    operation: options.operation || 'fileOperation',
    originalError: options.originalError,
    file: fileInfo,
    canRetry,
    requiresUserAction: !canRetry,
    shouldNotify: true,
    notificationLevel: 'error',
    dedupeKey: `file-${errorCode}-${source}`
  };
};

// =============================================================================
// 4. WebSocket错误工厂
// =============================================================================

export const createWebSocketError = (
  errorCode: WebSocketErrorCode,
  source: string,
  options: {
    originalEvent?: Event;
    operation?: string;
  } = {}
): StandardError => {
  let message = 'WebSocket连接错误';
  let userFriendlyMessage = '连接出现问题';
  let canRetry = true;
  let reconnectAvailable = true;

  switch (errorCode) {
    case WebSocketErrorCode.CONNECTION_FAILED:
      message = 'WebSocket连接失败';
      userFriendlyMessage = '无法建立连接，请检查网络';
      canRetry = true;
      break;
    case WebSocketErrorCode.CONNECTION_LOST:
      message = 'WebSocket连接丢失';
      userFriendlyMessage = '连接已断开，正在重新连接...';
      canRetry = true;
      break;
    case WebSocketErrorCode.AUTHENTICATION_FAILED:
      message = 'WebSocket认证失败';
      userFriendlyMessage = '连接认证失败，请重新登录';
      canRetry = false;
      reconnectAvailable = false;
      break;
    case WebSocketErrorCode.MESSAGE_FAILED:
      message = 'WebSocket消息发送失败';
      userFriendlyMessage = '消息发送失败，请重试';
      canRetry = true;
      break;
    case WebSocketErrorCode.PROTOCOL_ERROR:
      message = 'WebSocket协议错误';
      userFriendlyMessage = '连接协议异常';
      canRetry = false;
      break;
  }

  const websocketInfo: WebSocketErrorInfo = {
    code: errorCode,
    originalEvent: options.originalEvent,
    reconnectAvailable,
    message,
    userFriendlyMessage
  };

  return {
    type: ErrorType.WEBSOCKET_ERROR,
    code: errorCode,
    message,
    userFriendlyMessage,
    timestamp: new Date(),
    source,
    operation: options.operation || 'websocket',
    originalError: options.originalEvent,
    websocket: websocketInfo,
    canRetry,
    requiresUserAction: !canRetry,
    shouldNotify: errorCode !== WebSocketErrorCode.CONNECTION_LOST, // 连接丢失时不通知（会自动重连）
    notificationLevel: errorCode === WebSocketErrorCode.CONNECTION_LOST ? 'info' : 'error',
    dedupeKey: `websocket-${errorCode}-${source}`
  };
};

// =============================================================================
// 5. API错误工厂（新版，返回StandardError）
// =============================================================================

export const createApiStandardError = (
  originalError: any,
  source: string,
  operation: string = 'apiCall'
): StandardError => {
  let type = ErrorType.API_ERROR;
  let code = originalError?.code || originalError?.status || 'API_ERROR';
  let message = originalError?.message || 'API请求失败';
  let userFriendlyMessage = '操作失败，请重试';
  let canRetry = true;
  let notificationLevel: 'info' | 'warning' | 'error' = 'error';

  // 根据HTTP状态码或业务错误码判断错误类型
  if (originalError?.status === 401) {
    type = ErrorType.UNAUTHORIZED;
    message = '未授权访问';
    userFriendlyMessage = '需要登录后才能进行该操作';
    canRetry = false;
  } else if (originalError?.status === 403) {
    type = ErrorType.FORBIDDEN;
    message = '权限不足';
    userFriendlyMessage = '您没有权限执行此操作';
    canRetry = false;
  } else if (originalError?.status >= 500 || originalError?.status === 0) {
    type = ErrorType.NETWORK_ERROR;
    message = '网络错误';
    userFriendlyMessage = '网络连接异常，请检查网络后重试';
    canRetry = true;
  } else if (originalError?.status === 408 || originalError?.name === 'TimeoutError') {
    type = ErrorType.TIMEOUT_ERROR;
    message = '请求超时';
    userFriendlyMessage = '请求超时，请重试';
    canRetry = true;
  }

  return {
    type,
    code,
    message,
    userFriendlyMessage,
    timestamp: new Date(),
    source,
    operation,
    originalError,
    canRetry,
    requiresUserAction: !canRetry && type === ErrorType.UNAUTHORIZED,
    actionText: type === ErrorType.UNAUTHORIZED ? '请重新登录' : undefined,
    shouldNotify: true,
    notificationLevel,
    dedupeKey: `api-${code}-${source}-${operation}`
  };
};

// =============================================================================
// 6. 通用错误工厂（兜底）
// =============================================================================

export const createGenericError = (
  originalError: any,
  source: string,
  operation: string = 'unknown'
): StandardError => {
  const message = originalError?.message || '未知错误';
  const userFriendlyMessage = '操作失败，请重试';

  return {
    type: ErrorType.UNKNOWN_ERROR,
    code: 'UNKNOWN',
    message,
    userFriendlyMessage,
    timestamp: new Date(),
    source,
    operation,
    originalError,
    canRetry: true,
    requiresUserAction: false,
    shouldNotify: true,
    notificationLevel: 'error',
    dedupeKey: `unknown-${source}-${operation}`
  };
};

// =============================================================================
// 7. 智能错误识别与创建
// =============================================================================

export const createStandardError = (
  originalError: any,
  source: string,
  operation: string = 'unknown'
): StandardError => {
  // 地理位置错误
  if (originalError?.code && [1, 2, 3].includes(originalError.code)) {
    return createGeolocationError(originalError, source, operation);
  }

  // API错误（有status或来自fetch）
  if (originalError?.status || originalError?.response) {
    return createApiStandardError(originalError, source, operation);
  }

  // WebSocket错误
  if (originalError?.target instanceof WebSocket || originalError?.type?.includes('websocket')) {
    return createWebSocketError(WebSocketErrorCode.CONNECTION_FAILED, source, { 
      originalEvent: originalError,
      operation 
    });
  }

  // 文件错误
  if (originalError?.name === 'FileError' || (originalError?.target instanceof FileReader)) {
    return createFileError(FileErrorCode.READ_FAILED, source, { 
      originalError,
      operation 
    });
  }

  // 网络错误
  if (originalError?.name === 'TypeError' && originalError?.message?.includes('fetch')) {
    return createApiStandardError({ 
      status: 0, 
      message: '网络连接失败' 
    }, source, operation);
  }

  // 兜底处理
  return createGenericError(originalError, source, operation);
};
