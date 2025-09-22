/**
 * 密码安全工具类
 * 提供密码哈希计算、nonce验证等安全功能
 */

import cryptoFramework from '@ohos.security.cryptoFramework';
import { util } from '@kit.ArkTS';
import { hilog } from '@kit.PerformanceAnalysisKit';

const DOMAIN_NUMBER: number = 0xFF01;
const TAG: string = 'PasswordSecurityUtil';

export class PasswordSecurityUtil {

  /**
   * 计算传输哈希: SHA256(password + nonce + timestamp)
   * @param password 用户的明文密码
   * @param nonce 服务器返回的随机数
   * @param timestamp 服务器返回的时间戳
   * @returns 计算出的SHA-256哈希值
   */
  static async calculateTransportHash(password: string, nonce: string, timestamp: number): Promise<string> {
    try {
      const dataToHash = password + nonce + timestamp.toString();
      
      // 将字符串转换为Uint8Array
      const encoder = new util.TextEncoder();
      const dataBuffer = encoder.encodeInto(dataToHash);

      // 创建MD算法实例
      const md = cryptoFramework.createMd('SHA256');
      await md.update({ data: dataBuffer });
      const hashResult = await md.digest();

      // 将DataBlob转换为十六进制字符串
      const hashArray = new Uint8Array(hashResult.data);
      let hexString = '';
      for (let i = 0; i < hashArray.length; i++) {
        const hex = hashArray[i].toString(16).padStart(2, '0');
        hexString += hex;
      }
      
      return hexString;
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `计算密码哈希失败: ${JSON.stringify(error)}`);
      throw new Error('密码哈希计算失败');
    }
  }

  /**
   * 验证nonce的有效性
   * @param nonce 服务器返回的随机数
   * @param timestamp 服务器返回的时间戳
   * @param expiresIn nonce的有效期（秒）
   * @returns 如果nonce有效则返回true，否则返回false
   */
  static validateNonce(nonce: string, timestamp: number, expiresIn: number): boolean {
    if (!nonce || !timestamp || !expiresIn) {
      hilog.warn(DOMAIN_NUMBER, TAG, 'nonce、timestamp或expiresIn缺失');
      return false;
    }

    const currentTime = Math.floor(Date.now() / 1000); // 当前Unix时间戳
    const nonceExpiryTime = timestamp + expiresIn;

    if (currentTime > nonceExpiryTime) {
      hilog.warn(DOMAIN_NUMBER, TAG, `nonce已过期。当前时间: ${currentTime}, 过期时间: ${nonceExpiryTime}`);
      return false;
    }

    // 可以在这里添加更多nonce的验证逻辑，例如检查nonce格式等
    return true;
  }

  /**
   * 清除敏感字符串（例如密码）
   * 尝试覆盖内存中的敏感数据，降低泄露风险
   * @param sensitiveString 敏感字符串
   */
  static clearSensitiveString(sensitiveString: string): void {
    // 在JavaScript/TypeScript中，直接操作内存以"清除"字符串是不可行的
    // 因为字符串是不可变的，并且垃圾回收机制会管理内存。
    // 这里的实现仅为示意，实际效果有限。
    // 对于真正的安全敏感数据，应考虑使用更底层的语言或安全硬件模块。
    sensitiveString = ''; // 赋值为空字符串，使其尽快被垃圾回收
    hilog.debug(DOMAIN_NUMBER, TAG, '敏感字符串已尝试清理');
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