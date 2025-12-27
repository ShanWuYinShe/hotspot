/**
 * 版本号工具类
 * 用于处理语义化版本号 (Semantic Versioning)
 */
export class VersionUtil {
  /**
   * 比较两个版本号
   * @param v1 版本号1 (如 "1.0.0")
   * @param v2 版本号2 (如 "1.0.1")
   * @returns 1: v1 > v2, -1: v1 < v2, 0: v1 == v2
   */
  public static compare(v1: string, v2: string): number {
    if (!v1 || !v2) return 0;
    
    // 移除可能存在的 v 前缀
    const cleanV1 = v1.replace(/^v/, '');
    const cleanV2 = v2.replace(/^v/, '');
    
    const parts1 = cleanV1.split('.').map((p) => {
      const n = parseInt(p, 10);
      return isNaN(n) ? 0 : n;
    });
    const parts2 = cleanV2.split('.').map((p) => {
      const n = parseInt(p, 10);
      return isNaN(n) ? 0 : n;
    });
    
    const maxLength = Math.max(parts1.length, parts2.length);
    
    for (let i = 0; i < maxLength; i++) {
      const num1 = i < parts1.length ? parts1[i] : 0;
      const num2 = i < parts2.length ? parts2[i] : 0;
      
      if (num1 > num2) return 1;
      if (num1 < num2) return -1;
    }
    
    return 0;
  }

  /**
   * 检查是否需要更新
   * @param currentVersion 当前版本
   * @param targetVersion 目标版本
   * @returns true 如果 targetVersion > currentVersion
   */
  public static hasNewVersion(currentVersion: string, targetVersion: string): boolean {
    return VersionUtil.compare(targetVersion, currentVersion) > 0;
  }
}
