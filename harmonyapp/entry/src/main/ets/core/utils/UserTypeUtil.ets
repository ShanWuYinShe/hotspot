/**
 * 用户类型工具类
 *
 * 用户类型通过 user_type 字段区分（按 spec）：
 * - temp: 临时用户
 * - normal: 注册用户
 *
 * 管理员身份不通过 user_type 体现，需通过 admin/rbac/me 获取。
 */

/**
 * 用户类型枚举
 * 与后端 UserType 枚举保持一致
 */
export enum UserType {
  TEMP = 'temp',     // 临时用户
  NORMAL = 'normal', // 注册用户
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
   * 获取用户类型名称
   * @param userType 用户类型
   */
  static getUserTypeName(userType: string): string {
    switch (userType) {
      case UserType.TEMP:
        return '临时用户';
      case UserType.NORMAL:
        return '普通用户';
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
      userType === UserType.NORMAL;
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
      default:
        return {
          can_create_group: false,
          can_send_message: false,
          can_kick_member: false
        };
    }
  }

}
