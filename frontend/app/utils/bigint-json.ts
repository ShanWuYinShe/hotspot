/**
 * BigInt JSON 序列化/反序列化工具
 * 解决 JavaScript 大整数精度问题和 JSON 兼容性
 */

// 定义需要作为 BigInt 处理的字段
const BIGINT_FIELDS = new Set([
  'user_id',
  'sender_id', 
  'recipient_id',
  'peer_id',
  'owner_id',
  'id',
  'message_id', // 如果消息ID也可能很大
  // 可以根据需要添加更多字段
]);

// 检查字段是否应该被处理为 BigInt
function shouldBeBigInt(key: string, value: any): boolean {
  return BIGINT_FIELDS.has(key) && 
         (typeof value === 'string' || typeof value === 'number') &&
         value !== null && 
         value !== undefined &&
         value !== '';
}

/**
 * 自定义 JSON.stringify 替换器
 * 将 BigInt 转换为字符串以便序列化
 */
export function bigIntReplacer(key: string, value: any): any {
  if (typeof value === 'bigint') {
    return value.toString();
  }
  return value;
}

/**
 * 自定义 JSON.parse 恢复器
 * 将特定字段的字符串转换回 BigInt（如果需要）或保持为字符串
 */
export function bigIntReviver(key: string, value: any): any {
  if (shouldBeBigInt(key, value)) {
    // 检查是否是有效的大整数字符串
    if (typeof value === 'string' && /^\d+$/.test(value)) {
      const num = Number(value);
      // 如果超过安全整数范围，保持为字符串；否则转为数字
      return num > Number.MAX_SAFE_INTEGER ? value : num;
    }
    // 如果是数字且在安全范围内，保持为数字
    if (typeof value === 'number' && Number.isSafeInteger(value)) {
      return value;
    }
    // 其他情况转为字符串确保安全
    return String(value);
  }
  return value;
}

/**
 * 安全的 JSON 序列化
 */
export function safeStringify(obj: any): string {
  return JSON.stringify(obj, bigIntReplacer);
}

/**
 * 安全的 JSON 反序列化
 */
export function safeParse<T = any>(json: string): T {
  return JSON.parse(json, bigIntReviver);
}

/**
 * 处理 API 响应数据，确保 ID 字段的类型安全
 */
export function processApiResponse<T>(data: T): T {
  if (!data || typeof data !== 'object') {
    return data;
  }

  // 递归处理对象和数组
  if (Array.isArray(data)) {
    return data.map(item => processApiResponse(item)) as T;
  }

  const processed = { ...data } as any;
  
  // 处理对象的每个属性
  for (const [key, value] of Object.entries(processed)) {
    if (shouldBeBigInt(key, value)) {
      // 确保 ID 字段为字符串类型以避免精度问题
      processed[key] = String(value);
    } else if (value && typeof value === 'object') {
      // 递归处理嵌套对象
      processed[key] = processApiResponse(value);
    }
  }

  return processed;
}

/**
 * 处理发送给后端的数据
 */
export function processApiRequest<T>(data: T): T {
  if (!data || typeof data !== 'object') {
    return data;
  }

  if (Array.isArray(data)) {
    return data.map(item => processApiRequest(item)) as T;
  }

  const processed = { ...data } as any;
  
  for (const [key, value] of Object.entries(processed)) {
    if (shouldBeBigInt(key, value)) {
      // 发送给后端时，确保是字符串格式
      processed[key] = String(value);
    } else if (value && typeof value === 'object') {
      processed[key] = processApiRequest(value);
    }
  }

  return processed;
}

/**
 * 类型安全的 ID 比较函数
 */
export function compareIds(id1: string | number | bigint, id2: string | number | bigint): boolean {
  return String(id1) === String(id2);
}

/**
 * 类型安全的 ID 转换函数
 */
export function toSafeId(id: string | number | bigint | null | undefined): string | null {
  if (id === null || id === undefined || id === '') {
    return null;
  }
  return String(id);
}
