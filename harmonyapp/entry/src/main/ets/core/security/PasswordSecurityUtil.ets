/**
 * 密码安全工具类
 * 提供密码哈希计算等安全功能
 * 
 * 根据 api_spec.md 密码安全策略：
 * - **不做客户端哈希**：密码在 HTTPS/TLS 中明文传输，服务端进行慢哈希存储
 */

import { hilog } from '@kit.PerformanceAnalysisKit';

const DOMAIN_NUMBER: number = 0xFF01;
const TAG: string = 'PasswordSecurityUtil';

export class PasswordSecurityUtil {

  /**
   * 准备密码用于传输（不做哈希）
   *
   * @param password 用户的明文密码
   * @returns 原样返回 password（在 TLS 中传输）
   */
  static async calculatePasswordHash(password: string): Promise<string> {
    try {
      return password;
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `准备密码失败: ${JSON.stringify(error)}`);
      throw new Error('密码处理失败');
    }
  }

  /**
   * 生成随机字符串
   * @param length 字符串长度
   * @returns 随机字符串
   */
  static generateRandomString(length: number): string {
    const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
    let result = '';
    for (let i = 0; i < length; i++) {
      result += chars.charAt(Math.floor(Math.random() * chars.length));
    }
    return result;
  }

  /**
   * 验证密码强度
   * @param password 密码
   * @returns 密码强度评分 (0-100)
   */
  static validatePasswordStrength(password: string): number {
    if (!password) {
      return 0;
    }

    let score = 0;
    
    // 长度检查
    if (password.length >= 8) score += 25;
    if (password.length >= 12) score += 15;
    
    // 字符类型检查
    if (/[a-z]/.test(password)) score += 15; // 小写字母
    if (/[A-Z]/.test(password)) score += 15; // 大写字母
    if (/[0-9]/.test(password)) score += 15; // 数字
    if (/[^A-Za-z0-9]/.test(password)) score += 15; // 特殊字符
    
    return Math.min(score, 100);
  }

  /**
   * 获取密码强度描述
   * @param score 密码强度评分
   * @returns 强度描述
   */
  static getPasswordStrengthDescription(score: number): string {
    if (score < 30) {
      return '弱';
    } else if (score < 60) {
      return '中等';
    } else if (score < 80) {
      return '强';
    } else {
      return '很强';
    }
  }
}