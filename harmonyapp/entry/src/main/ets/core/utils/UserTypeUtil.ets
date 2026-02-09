/**
 * 用户类型工具类
 *
 * 用户类型通过 user_type 字段区分：
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
  TEMP = 'temp',     // 临时用户
  NORMAL = 'normal', // 注册用户
  ADMIN = 'admin',   // 管理员用户
  SYSTEM = 'system', // 系统用户
}

/**
 * 权限配置接口
 */
interface PermissionConfig {
  can_create_group: boolean;
  can_send_message: boolean;
  can_kick_member: boolean;
}

export class UserTypeUtil {
  /**
   * 将字符串解析为 UserType 枚举
   * @param value 用户类型字符串
   * @returns UserType 枚举值，无效值返回 NORMAL
   */
  static parseUserType(value: string): UserType {
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
   * 判断是否为临时用户
   * @param userType 用户类型
   */
  static isTempUser(userType: string): boolean {
    return userType === UserType.TEMP;
  }

  /**
   * 判断是否为普通用户
   * @param userType 用户类型
   */
  static isNormalUser(userType: string): boolean {
    return userType === UserType.NORMAL;
  }

  /**
   * 判断是否为管理员用户
   * @param userType 用户类型
   */
  static isAdminUser(userType: string): boolean {
    return userType === UserType.ADMIN;
  }

  /**
   * 判断是否为系统用户
   * @param userType 用户类型
   */
  static isSystemUser(userType: string): boolean {
    return userType === UserType.SYSTEM;
  }

  /**
   * 获取用户类型名称
   * @param userType 用户类型
   */
  static getUserTypeName(userType: string): string {
    switch (userType) {
      case UserType.TEMP:
        return '临时用户';
      case UserType.NORMAL:
        return '普通用户';
      case UserType.ADMIN:
        return '管理员';
      case UserType.SYSTEM:
        return '系统用户';
      default:
        return '未知用户';
    }
  }

  /**
   * 验证用户类型是否有效
   * @param userType 用户类型
   */
  static isValidUserType(userType: string): boolean {
    return userType === UserType.TEMP ||
      userType === UserType.NORMAL ||
      userType === UserType.ADMIN ||
      userType === UserType.SYSTEM;
  }

  /**
   * 检查用户是否拥有特定权限
   * @param userType 用户类型
   * @param permission 权限字符串
   */
  static hasPermission(userType: string, permission: string): boolean {
    const permissions = UserTypeUtil.getPermissionConfig(userType);

    switch (permission) {
      case 'can_create_group':
        return permissions.can_create_group;
      case 'can_send_message':
        return permissions.can_send_message;
      case 'can_kick_member':
        return permissions.can_kick_member;
      default:
        return false;
    }
  }

  /**
   * 获取用户类型的权限配置
   * @param userType 用户类型
   * @returns 权限配置对象
   */
  private static getPermissionConfig(userType: string): PermissionConfig {
    switch (userType) {
      case UserType.TEMP:
        return {
          can_create_group: false,
          can_send_message: true,
          can_kick_member: false
        };
      case UserType.NORMAL:
        return {
          can_create_group: true,
          can_send_message: true,
          can_kick_member: true
        };
      case UserType.ADMIN:
        return {
          can_create_group: true,
          can_send_message: true,
          can_kick_member: true
        };
      case UserType.SYSTEM:
        return {
          can_create_group: false,
          can_send_message: true,
          can_kick_member: false
        };
      default:
        return {
          can_create_group: false,
          can_send_message: false,
          can_kick_member: false
        };
    }
  }
}

/**
 * 系统用户ID常量接口
 */
export interface SystemUserIds {
  NOTIFICATION: string;
  SECURITY_CENTER: string;
  OPERATIONS_TEAM: string;
}

/**
 * 系统用户ID常量
 */
export const SYSTEM_USER_IDS: SystemUserIds = {
  NOTIFICATION: '1',       // 系统通知
  SECURITY_CENTER: '2',    // 安全中心
  OPERATIONS_TEAM: '3',    // 运营团队
};

/**
 * 系统用户ID区间边界
 */
export const SYSTEM_USER_ID_MAX = 1000000;
