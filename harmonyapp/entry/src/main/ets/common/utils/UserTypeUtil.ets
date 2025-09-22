/**
 * 用户ID类型判断工具类
 * 根据后端文档定义的用户ID编码规范实现类型判断功能
 */

import { hilog } from '@kit.PerformanceAnalysisKit';

const DOMAIN_NUMBER: number = 0xFF02;
const TAG: string = 'UserTypeUtil';

/**
 * 用户类型枚举
 * 对应后端文档中用户ID高8位的编码
 */
export enum UserType {
  UNKNOWN = 0x00,       // 未知用户类型
  TEMPORARY = 0x01,     // 临时用户
  REGULAR = 0x02,       // 普通用户
  SYSTEM = 0x0F,        // 系统用户
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
   * 获取用户ID的高8位作为用户类型编码
   * @param userId 64位用户ID字符串
   * @returns 用户类型编码（0-255）
   */
  static getUserTypeCode(userId: string): number {
    if (!userId || userId.length < 16) { // 64位ID至少16个十六进制字符
      hilog.warn(DOMAIN_NUMBER, TAG, `无效的用户ID格式: ${userId}`);
      return UserType.UNKNOWN;
    }
    try {
      // 假设userId是64位整数的字符串表示，高8位是用户类型
      // 取前两位十六进制字符转换为数字
      const typeCodeStr = userId.substring(0, 2);
      return parseInt(typeCodeStr, 16);
    } catch (e) {
      hilog.error(DOMAIN_NUMBER, TAG, `解析用户ID类型失败: ${JSON.stringify(e)}`);
      return UserType.UNKNOWN;
    }
  }

  /**
   * 获取用户类型
   * @param userId 64位用户ID字符串
   * @returns UserType枚举值
   */
  static getUserType(userId: string): UserType {
    const typeCode = UserTypeUtil.getUserTypeCode(userId);
    switch (typeCode) {
      case UserType.TEMPORARY:
        return UserType.TEMPORARY;
      case UserType.REGULAR:
        return UserType.REGULAR;
      case UserType.SYSTEM:
        return UserType.SYSTEM;
      default:
        return UserType.UNKNOWN;
    }
  }

  /**
   * 判断是否为临时用户
   * @param userId 用户ID
   */
  static isTempUser(userId: string): boolean {
    return UserTypeUtil.getUserType(userId) === UserType.TEMPORARY;
  }

  /**
   * 判断是否为普通用户
   * @param userId 用户ID
   */
  static isRegularUser(userId: string): boolean {
    return UserTypeUtil.getUserType(userId) === UserType.REGULAR;
  }

  /**
   * 判断是否为系统用户
   * @param userId 用户ID
   */
  static isSystemUser(userId: string): boolean {
    return UserTypeUtil.getUserType(userId) === UserType.SYSTEM;
  }

  /**
   * 获取用户类型名称
   * @param userId 用户ID
   */
  static getUserTypeName(userId: string): string {
    switch (UserTypeUtil.getUserType(userId)) {
      case UserType.TEMPORARY:
        return '临时用户';
      case UserType.REGULAR:
        return '普通用户';
      case UserType.SYSTEM:
        return '系统用户';
      case UserType.UNKNOWN:
      default:
        return '未知用户';
    }
  }

  /**
   * 验证用户ID格式是否有效
   * 简单的正则匹配，确保是16位十六进制字符串
   * @param userId 用户ID
   */
  static validateUserId(userId: string): boolean {
    return /^[0-9a-fA-F]{16}$/.test(userId);
  }

  /**
   * 检查用户是否拥有特定权限
   * 这是一个示例函数，实际权限逻辑可能更复杂，需要后端或本地权限配置
   * @param userId 用户ID
   * @param permission 权限字符串，例如 'can_create_group', 'can_send_message'
   */
  static hasPermission(userId: string, permission: string): boolean {
    const userType = UserTypeUtil.getUserType(userId);
    
    // 获取权限配置
    const permissions = UserTypeUtil.getPermissionConfig(userType);
    
    // 检查权限
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
  private static getPermissionConfig(userType: UserType): PermissionConfig {
    switch (userType) {
      case UserType.TEMPORARY:
        return {
          can_create_group: false,
          can_send_message: true,
          can_kick_member: false
        };
      case UserType.REGULAR:
        return {
          can_create_group: true,
          can_send_message: true,
          can_kick_member: true
        };
      case UserType.SYSTEM:
        return {
          can_create_group: false,
          can_send_message: false,
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