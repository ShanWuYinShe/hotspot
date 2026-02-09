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

export function isAuthErrorCode(code: Object): boolean {
  return typeof code === 'number' && (code as number) >= 40100 && (code as number) <= 40199;
}

export function isTokenInvalidOrExpiredCode(code: Object): boolean {
  return code === 40101;
}

export function isTokenMissingCode(code: Object): boolean {
  return code === 40102;
}

export function isReauthRequiredCode(code: Object): boolean {
  return code === 40106;
}

export function isSoftTokenRotation(message: Object): boolean {
  return typeof message === 'string' && (message as string).includes('已被轮换');
}

export function shouldForceLogoutOnAuthError(code: Object, message?: Object): boolean {
  if (!isAuthErrorCode(code)) return false;
  if (isReauthRequiredCode(code)) return true;
  if (isTokenMissingCode(code)) return true;
  if (isTokenInvalidOrExpiredCode(code)) return !isSoftTokenRotation(message);
  return true;
}

export function isBadRequestCode(code: Object): boolean {
  return typeof code === 'number' && (code as number) >= 40000 && (code as number) <= 40099;
}

export function isForbiddenCode(code: Object): boolean {
  return typeof code === 'number' && (code as number) >= 40300 && (code as number) <= 40399;
}

export function isNotFoundCode(code: Object): boolean {
  return typeof code === 'number' && (code as number) >= 40400 && (code as number) <= 40499;
}

export function isConflictCode(code: Object): boolean {
  return typeof code === 'number' && (code as number) >= 40900 && (code as number) <= 40999;
}

export function isCursorInvalidOrExpiredCode(code: Object): boolean {
  return code === 41000;
}

export function isRateLimitCode(code: Object): boolean {
  return typeof code === 'number' && (code as number) >= 42900 && (code as number) <= 42999;
}

export function isServerErrorCode(code: Object): boolean {
  return typeof code === 'number' && (code as number) >= 50000 && (code as number) <= 59999;
}

function isSafeUserMessage(message: Object): boolean {
  if (typeof message !== 'string') return false;
  const msg = (message as string).trim();
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
  const code = snapshot.code;
  const status = snapshot.status;
  const message = snapshot.message;

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
