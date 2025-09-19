/**
 * 用户ID编码规范工具函数
 * 根据需求文档：用户ID为64位整数，高8位表示用户类型，低56位表示用户序号
 */

/**
 * 用户类型枚举
 */
export enum UserType {
  TEMP = 1,     // 临时用户
  REGULAR = 2,  // 普通用户
  SYSTEM = 3,   // 系统用户（官方账号等）
}

/**
 * 获取用户类型
 * @param userId 用户ID字符串（64位整数）
 * @returns 用户类型编码
 */
export function getUserType(userId: string): number {
  const userIdBigInt = BigInt(userId);
  return Number(userIdBigInt >> 56n);
}

/**
 * 判断是否为临时用户
 * @param userId 用户ID字符串
 * @returns 是否为临时用户
 */
export function isTempUser(userId: string): boolean {
  return getUserType(userId) === UserType.TEMP;
}

/**
 * 判断是否为普通用户
 * @param userId 用户ID字符串
 * @returns 是否为普通用户
 */
export function isRegularUser(userId: string): boolean {
  return getUserType(userId) === UserType.REGULAR;
}

/**
 * 判断是否为系统用户
 * @param userId 用户ID字符串
 * @returns 是否为系统用户
 */
export function isSystemUser(userId: string): boolean {
  return getUserType(userId) === UserType.SYSTEM;
}

/**
 * 获取用户类型显示名称
 * @param userId 用户ID字符串
 * @returns 用户类型显示名称
 */
export function getUserTypeDisplayName(userId: string): string {
  const userType = getUserType(userId);
  switch (userType) {
    case UserType.TEMP:
      return '临时用户';
    case UserType.REGULAR:
      return '普通用户';
    case UserType.SYSTEM:
      return '官方账号';
    default:
      return '未知类型';
  }
}

/**
 * 获取用户类型标签样式
 * @param userId 用户ID字符串
 * @returns 标签样式配置
 */
export function getUserTypeTagStyle(userId: string): {
  text: string;
  color: string;
  bgColor: string;
} {
  const userType = getUserType(userId);
  switch (userType) {
    case UserType.TEMP:
      return {
        text: '临时',
        color: 'text-orange-600',
        bgColor: 'bg-orange-100',
      };
    case UserType.SYSTEM:
      return {
        text: '官方',
        color: 'text-blue-600',
        bgColor: 'bg-blue-100',
      };
    default:
      // 普通用户不显示标签
      return {
        text: '',
        color: '',
        bgColor: '',
      };
  }
}

/**
 * 示例用户ID（用于测试和文档）
 */
export const EXAMPLE_USER_IDS = {
  TEMP: '72057594037927936',      // 0x0100000000000000
  REGULAR: '144115188075855872',  // 0x0200000000000000
  SYSTEM: '216172782113783808',   // 0x0300000000000000
} as const;
