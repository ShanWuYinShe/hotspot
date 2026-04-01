/**
 * API 错误码判断工具
 *
 * 错误码严格遵循 spec/api_spec_errors.md 定义：
 * - 10000: 参数错误
 * - 10100: 认证失败（token 无效/过期/缺失）
 * - 10200: 权限不足/访问受限
 * - 10201: 账号被封禁（需强制登出+封禁页）
 * - 10300: 资源不存在
 * - 10400: 业务冲突
 * - 10500: 请求过于频繁
 * - 10600: 服务器错误
 */

export interface ApiErrorSnapshot {
  code?: number;
  status?: number;
  message?: string;
}

function isRecord(v: Object): boolean {
  return v !== null && typeof v === 'object';
}

export function extractApiErrorSnapshot(result: Object, responseCode?: number): ApiErrorSnapshot {
  let code: number | undefined;
  let message: string | undefined;

  if (isRecord(result)) {
    const resultRecord = result as Record<string, Object>;
    if (typeof resultRecord.code === 'number') code = resultRecord.code as number;
    if (typeof resultRecord.message === 'string') message = resultRecord.message as string;
    if (!message && typeof resultRecord.msg === 'string') message = resultRecord.msg as string;
  }

  return { code, status: typeof responseCode === 'number' ? responseCode : undefined, message };
}

/** 认证失败（token 无效/过期/缺失），客户端应尝试刷新 token */
export function isAuthErrorCode(code: number): boolean {
  return code === 10100;
}

/** 账号被封禁，客户端必须强制登出并显示封禁页 */
export function isAccountBannedCode(code: number): boolean {
  return code === 10201;
}

export function isBadRequestCode(code: number): boolean {
  return code === 10000;
}

export function isForbiddenCode(code: number): boolean {
  return code === 10200;
}

export function isNotFoundCode(code: number): boolean {
  return code === 10300;
}

export function isConflictCode(code: number): boolean {
  return code === 10400;
}

export function isRateLimitCode(code: number): boolean {
  return code === 10500;
}

export function isServerErrorCode(code: number): boolean {
  return code === 10600;
}

function isSafeUserMessage(message: Object): boolean {
  if (typeof message !== 'string') return false;
  const msg = (message as string).trim();
  if (!msg) return false;
  if (msg.includes('/api/') || msg.includes('baseURL')) return false;
  if (/^\[[A-Z]+\]\s+".*"\:\s+\d{3}\s+/.test(msg)) return false;
  if (msg.includes('Internal Server Error')) return false;
  return true;
}

export function guessUserMessageFromSnapshot(
  snapshot: ApiErrorSnapshot,
  defaultMessage: string = '操作失败'
): string {
  const code = snapshot.code;
  const status = snapshot.status;
  const message = snapshot.message;

  if (code !== undefined && isAuthErrorCode(code)) return '登录已过期，请重新登录';
  if (code !== undefined && isAccountBannedCode(code)) return '账号已被封禁';
  if (code !== undefined && isBadRequestCode(code)) return (message !== undefined && isSafeUserMessage(message)) ? message : '请求参数错误';
  if (code !== undefined && isForbiddenCode(code)) return '无权限执行此操作';
  if (code !== undefined && isNotFoundCode(code)) return '资源不存在或已被删除';
  if (code !== undefined && isConflictCode(code)) return (message !== undefined && isSafeUserMessage(message)) ? message : '操作冲突，请稍后重试';
  if ((code !== undefined && isRateLimitCode(code)) || status === 429) return '请求过于频繁，请稍后再试';

  if (message !== undefined && isSafeUserMessage(message)) return message;
  if ((code !== undefined && isServerErrorCode(code)) || (typeof status === 'number' && status >= 500)) return defaultMessage;
  return defaultMessage;
}

/**
 * 便捷方法：从 API 响应对象中提取友好错误消息
 * 供页面统一替换 response.message || 'fallback' 模式
 */
export function getErrorMessage(response: Object, fallback: string = '操作失败'): string {
  const snapshot = extractApiErrorSnapshot(response);
  return guessUserMessageFromSnapshot(snapshot, fallback);
}
