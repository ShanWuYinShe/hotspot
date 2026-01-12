export interface ApiErrorSnapshot {
  code?: number;
  status?: number;
  message?: string;
}

const isRecord = (v: unknown): v is Record<string, unknown> => typeof v === 'object' && v !== null;

export function extractApiErrorSnapshot(result: unknown, responseCode?: number): ApiErrorSnapshot {
  let code: number | undefined;
  let message: string | undefined;

  if (isRecord(result)) {
    if (typeof result.code === 'number') code = result.code;
    if (typeof result.message === 'string') message = result.message;
    if (!message && typeof result.msg === 'string') message = result.msg;
  }

  return { code, status: typeof responseCode === 'number' ? responseCode : undefined, message };
}

export function isAuthErrorCode(code: unknown): boolean {
  return typeof code === 'number' && code >= 40100 && code <= 40199;
}

export function isTokenInvalidOrExpiredCode(code: unknown): boolean {
  return code === 40101;
}

export function isTokenMissingCode(code: unknown): boolean {
  return code === 40102;
}

export function isReauthRequiredCode(code: unknown): boolean {
  return code === 40106;
}

export function isSoftTokenRotation(message: unknown): boolean {
  return typeof message === 'string' && message.includes('已被轮换');
}

export function shouldForceLogoutOnAuthError(code: unknown, message?: unknown): boolean {
  if (!isAuthErrorCode(code)) return false;
  if (isReauthRequiredCode(code)) return true;
  if (isTokenMissingCode(code)) return true;
  if (isTokenInvalidOrExpiredCode(code)) return !isSoftTokenRotation(message);
  return true;
}

export function isBadRequestCode(code: unknown): boolean {
  return typeof code === 'number' && code >= 40000 && code <= 40099;
}

export function isForbiddenCode(code: unknown): boolean {
  return typeof code === 'number' && code >= 40300 && code <= 40399;
}

export function isNotFoundCode(code: unknown): boolean {
  return typeof code === 'number' && code >= 40400 && code <= 40499;
}

export function isConflictCode(code: unknown): boolean {
  return typeof code === 'number' && code >= 40900 && code <= 40999;
}

export function isCursorInvalidOrExpiredCode(code: unknown): boolean {
  return code === 41000;
}

export function isRateLimitCode(code: unknown): boolean {
  return typeof code === 'number' && code >= 42900 && code <= 42999;
}

export function isServerErrorCode(code: unknown): boolean {
  return typeof code === 'number' && code >= 50000 && code <= 59999;
}

function isSafeUserMessage(message: unknown): message is string {
  if (typeof message !== 'string') return false;
  const msg = message.trim();
  if (!msg) return false;
  if (msg.includes('/api/') || msg.includes('baseURL')) return false;
  if (/^\[[A-Z]+\]\s+\".*\"\:\s+\d{3}\s+/.test(msg)) return false;
  if (msg.includes('Internal Server Error')) return false;
  return true;
}

export function guessUserMessageFromSnapshot(
  snapshot: ApiErrorSnapshot,
  defaultMessage: string = '操作失败'
): string {
  const { code, status, message } = snapshot;

  if (isTokenInvalidOrExpiredCode(code) || (typeof message === 'string' && message.includes('ExpiredSignature'))) {
    return '登录已过期，请重新登录';
  }

  if (isTokenMissingCode(code) || isAuthErrorCode(code) || status === 401) {
    return '请先登录';
  }

  if (isBadRequestCode(code)) return isSafeUserMessage(message) ? message : '请求参数错误';
  if (isForbiddenCode(code)) return '无权限执行此操作';
  if (isNotFoundCode(code)) return '资源不存在或已被删除';
  if (isConflictCode(code)) return isSafeUserMessage(message) ? message : '操作冲突，请稍后重试';
  if (isCursorInvalidOrExpiredCode(code)) return '列表已过期，请刷新后重试';
  if (isRateLimitCode(code) || status === 429) return '请求过于频繁，请稍后再试';

  if (isSafeUserMessage(message)) return message;
  if (isServerErrorCode(code) || (typeof status === 'number' && status >= 500)) return defaultMessage;
  return defaultMessage;
}
