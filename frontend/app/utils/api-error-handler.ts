/**
 * API错误处理工具
 * 提供准确的错误信息反馈
 */

export interface ApiErrorInfo {
  title: string;
  message: string;
  type: 'network' | 'client' | 'server' | 'auth' | 'validation';
  canRetry: boolean;
  actionText?: string;
}

/**
 * 根据HTTP状态码和错误信息生成用户友好的错误描述
 */
export const getApiErrorInfo = (
  status: number, 
  statusText: string, 
  url?: string,
  responseData?: any
): ApiErrorInfo => {
  
  // 网络连接问题
  if (status === 0 || !status) {
    return {
      title: '网络连接失败',
      message: '无法连接到服务器，请检查网络连接',
      type: 'network',
      canRetry: true,
      actionText: '重试'
    };
  }

  // 客户端错误 (4xx)
  if (status >= 400 && status < 500) {
    switch (status) {
      case 400:
        return {
          title: '请求参数错误',
          message: responseData?.msg || '请求参数不正确，请检查输入内容',
          type: 'validation',
          canRetry: false
        };
        
      case 401:
        return {
          title: '身份验证失败',
          message: '登录已过期，请重新登录',
          type: 'auth',
          canRetry: false,
          actionText: '重新登录'
        };
        
      case 403:
        return {
          title: '权限不足',
          message: '您没有权限执行此操作',
          type: 'auth',
          canRetry: false
        };
        
      case 404:
        return {
          title: 'API接口无效',
          message: url ? `接口地址不存在：${getApiPath(url)}` : 'API接口地址错误',
          type: 'client',
          canRetry: false
        };
        
      case 409:
        return {
          title: '数据冲突',
          message: responseData?.msg || '数据已存在或发生冲突',
          type: 'validation',
          canRetry: false
        };
        
      case 422:
        return {
          title: '数据验证失败',
          message: responseData?.msg || '提交的数据格式不正确',
          type: 'validation',
          canRetry: false
        };
        
      case 429:
        return {
          title: '请求过于频繁',
          message: '操作太频繁，请稍后再试',
          type: 'client',
          canRetry: true,
          actionText: '稍后重试'
        };
        
      default:
        return {
          title: '请求错误',
          message: responseData?.msg || `客户端错误 (${status})`,
          type: 'client',
          canRetry: false
        };
    }
  }

  // 服务器错误 (5xx)
  if (status >= 500) {
    switch (status) {
      case 500:
        return {
          title: '服务器内部错误',
          message: '服务器处理请求时发生错误，请稍后重试',
          type: 'server',
          canRetry: true,
          actionText: '重试'
        };
        
      case 502:
      case 503:
      case 504:
        return {
          title: '服务器暂时不可用',
          message: '服务器正在维护或负载过高，请稍后重试',
          type: 'server',
          canRetry: true,
          actionText: '重试'
        };
        
      default:
        return {
          title: '服务器错误',
          message: `服务器错误 (${status})，请稍后重试`,
          type: 'server',
          canRetry: true,
          actionText: '重试'
        };
    }
  }

  // 其他未知错误
  return {
    title: '未知错误',
    message: statusText || '发生了未知错误，请稍后重试',
    type: 'client',
    canRetry: true,
    actionText: '重试'
  };
};

/**
 * 从完整URL中提取API路径
 */
const getApiPath = (url: string): string => {
  try {
    const urlObj = new URL(url);
    return urlObj.pathname;
  } catch {
    return url;
  }
};

/**
 * 处理业务错误码（后端返回的code字段）
 */
export const getBusinessErrorInfo = (code: number, msg: string): ApiErrorInfo => {
  // 根据业务错误码返回具体的错误信息
  switch (code) {
    case 1000:
      return {
        title: '参数错误',
        message: msg || '请求参数不正确',
        type: 'validation',
        canRetry: false
      };
      
    case 1001:
      return {
        title: '验证失败',
        message: msg || '数据验证失败',
        type: 'validation',
        canRetry: false
      };
      
    case 1002:
      return {
        title: '未授权访问',
        message: '登录已过期，请重新登录',
        type: 'auth',
        canRetry: false,
        actionText: '重新登录'
      };
      
    case 1003:
      return {
        title: '权限不足',
        message: msg || '您没有权限执行此操作',
        type: 'auth',
        canRetry: false
      };
      
    case 1004:
      return {
        title: '资源不存在',
        message: msg || '请求的资源不存在',
        type: 'client',
        canRetry: false
      };
      
    case 1005:
      return {
        title: '资源冲突',
        message: msg || '数据已存在或发生冲突',
        type: 'validation',
        canRetry: false
      };
      
    case 1006:
      return {
        title: '功能未实现',
        message: msg || '该功能暂未开放',
        type: 'client',
        canRetry: false
      };
      
    case 1007:
      return {
        title: '请求超时',
        message: msg || '请求处理超时，请稍后重试',
        type: 'timeout',
        canRetry: true,
        actionText: '重试'
      };
      
    case 5000:
      return {
        title: '服务器内部错误',
        message: '服务器处理请求时发生错误，请稍后重试',
        type: 'server',
        canRetry: true,
        actionText: '重试'
      };
      
    case 5001:
      return {
        title: '数据库错误',
        message: '数据库操作失败，请稍后重试',
        type: 'server',
        canRetry: true,
        actionText: '重试'
      };
      
    case 5002:
      return {
        title: '外部服务错误',
        message: '外部服务暂时不可用，请稍后重试',
        type: 'server',
        canRetry: true,
        actionText: '重试'
      };
      
    default:
      return {
        title: '操作失败',
        message: msg || `操作失败 (错误码: ${code})`,
        type: 'server',
        canRetry: true,
        actionText: '重试'
      };
  }
};

/**
 * 统一的错误处理函数
 */
export const handleApiError = (error: any, url?: string): ApiErrorInfo => {
  // 如果是业务错误（后端返回的错误响应）
  if (error?.data?.code && error.data.code !== 0) {
    return getBusinessErrorInfo(error.data.code, error.data.msg);
  }
  
  // 如果是HTTP错误
  if (error?.status !== undefined) {
    return getApiErrorInfo(error.status, error.statusText, url, error.data);
  }
  
  // 网络错误或其他错误
  if (error?.message) {
    if (error.message.includes('fetch') || error.message.includes('network')) {
      return {
        title: '网络连接失败',
        message: '无法连接到服务器，请检查网络连接',
        type: 'network',
        canRetry: true,
        actionText: '重试'
      };
    }
  }
  
  // 默认错误
  return {
    title: '操作失败',
    message: error?.message || '发生了未知错误，请稍后重试',
    type: 'client',
    canRetry: true,
    actionText: '重试'
  };
};
