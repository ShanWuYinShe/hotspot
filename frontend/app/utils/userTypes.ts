/**
 * 用户类型工具函数
 *
 * 用户类型通过 user_type 枚举字段区分：
 * - temp: 临时用户
 * - normal: 注册用户
 * - admin: 管理员用户
 * - system: 系统用户（官方账号等）
 *
 * 用户ID分配规则：
 * - 系统用户区间：1 - 1,000,000
 * - 临时用户和注册用户区间：1,000,001+
 */

/**
 * 用户类型枚举
 * 与后端 UserType 枚举保持一致
 */
export enum UserType {
  TEMP = 'temp', // 临时用户
  NORMAL = 'normal', // 注册用户
  ADMIN = 'admin', // 管理员用户
  SYSTEM = 'system', // 系统用户
}

/**
 * 用户类型字符串联合类型（用于类型约束）
 */
export type UserTypeValue = `${UserType}`;

/**
 * 将字符串解析为 UserType 枚举
 * @param value 用户类型字符串
 * @returns UserType 枚举值，无效值返回 NORMAL
 */
export function parseUserType(value: string): UserType {
  switch (value) {
    case UserType.TEMP:
      return UserType.TEMP;
    case UserType.ADMIN:
      return UserType.ADMIN;
    case UserType.SYSTEM:
      return UserType.SYSTEM;
    case UserType.NORMAL:
    default:
      return UserType.NORMAL;
  }
}

/**
 * 判断是否为管理员用户
 * @param userType 用户类型（字符串或枚举）
 * @returns 是否为管理员用户
 */
export function isAdminUser(userType: UserTypeValue | string): boolean {
  return userType === UserType.ADMIN;
}

/**
 * 判断是否为临时用户
 * @param userType 用户类型（字符串或枚举）
 * @returns 是否为临时用户
 */
export function isTempUser(userType: UserTypeValue | string): boolean {
  return userType === UserType.TEMP;
}

/**
 * 判断是否为普通用户（注册用户）
 * @param userType 用户类型（字符串或枚举）
 * @returns 是否为普通用户
 */
export function isNormalUser(userType: UserTypeValue | string): boolean {
  return userType === UserType.NORMAL;
}

/**
 * 判断是否为系统用户
 * @param userType 用户类型（字符串或枚举）
 * @returns 是否为系统用户
 */
export function isSystemUser(userType: UserTypeValue | string): boolean {
  return userType === UserType.SYSTEM;
}

/**
 * 获取用户类型显示名称
 * @param userType 用户类型（字符串或枚举）
 * @returns 用户类型显示名称
 */
export function getUserTypeDisplayName(userType: UserTypeValue | string): string {
  switch (userType) {
    case UserType.TEMP:
      return '临时用户';
    case UserType.NORMAL:
      return '普通用户';
    case UserType.ADMIN:
      return '管理员';
    case UserType.SYSTEM:
      return '官方账号';
    default:
      return '未知类型';
  }
}

/**
 * 获取用户类型标签样式
 * @param userType 用户类型（字符串或枚举）
 * @returns 标签样式配置
 */
/**
 * 获取用户类型对应的 Nuxt UI Badge 配置（避免在 Vue 里拼 Tailwind 颜色 class）
 */
export function getUserTypeBadge(userType: UserTypeValue | string): {
  label: string
  color: 'primary' | 'secondary' | 'success' | 'info' | 'warning' | 'error' | 'neutral'
} | null {
  switch (userType) {
    case UserType.TEMP:
      return { label: '临时', color: 'warning' }
    case UserType.ADMIN:
      return { label: '管理', color: 'secondary' }
    case UserType.SYSTEM:
      return { label: '官方', color: 'info' }
    default:
      return null
  }
}

/**
 * 系统用户ID常量
 */
export const SYSTEM_USER_IDS = {
  NOTIFICATION: '1', // 系统通知
  SECURITY_CENTER: '2', // 安全中心
  OPERATIONS_TEAM: '3', // 运营团队
} as const;

/**
 * 系统用户ID区间边界
 */
export const SYSTEM_USER_ID_MAX = 1000000;
