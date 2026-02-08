/**
 * 配置合并工具 - ArkTS兼容版本
 * 提供简化的配置合并功能
 */

/**
 * 验证结果接口
 */
interface ValidationResult {
  valid: boolean;
  missing: string[];
}

/**
 * 类型验证结果接口
 */
interface TypeValidationResult {
  valid: boolean;
  errors: string[];
}

/**
 * 范围验证结果接口
 */
interface RangeValidationResult {
  valid: boolean;
  errors: string[];
}

/**
 * 范围配置接口
 */
interface RangeConfig {
  min?: number;
  max?: number;
}

/**
 * 配置合并器
 */
export class ConfigMerger {
  
  /**
   * 合并两个配置对象（浅合并）
   * @param target 目标配置对象
   * @param source 源配置对象
   * @returns 合并后的新配置对象
   */
  static merge<T>(
    target: T, 
    source: Partial<T>
  ): T {
    const result: T = {} as T;
    
    // 复制目标对象的所有属性
    const targetKeys = Object.keys(target);
    for (let i = 0; i < targetKeys.length; i++) {
      const key = targetKeys[i];
      (result as Record<string, Object>)[key] = target[key];
    }
    
    // 合并源对象的属性
    const sourceKeys = Object.keys(source);
    for (let i = 0; i < sourceKeys.length; i++) {
      const key = sourceKeys[i];
      if (source[key] !== undefined) {
        (result as Record<string, Object>)[key] = source[key];
      }
    }
    
    return result;
  }
  
  /**
   * 合并多个配置对象
   * @param target 目标配置对象
   * @param sources 多个源配置对象
   * @returns 合并后的新配置对象
   */
  static mergeMultiple<T>(
    target: T,
    sources: Partial<T>[]
  ): T {
    let result = target;
    
    for (let i = 0; i < sources.length; i++) {
      result = ConfigMerger.merge(result, sources[i]);
    }
    
    return result;
  }
  
  /**
   * 条件合并：只有当条件为真时才合并
   * @param target 目标配置对象
   * @param source 源配置对象
   * @param condition 合并条件
   * @returns 合并后的新配置对象
   */
  static mergeIf<T>(
    target: T,
    source: Partial<T>,
    condition: boolean
  ): T {
    if (condition) {
      return ConfigMerger.merge(target, source);
    }
    return target;
  }
  
  /**
   * 安全合并：只合并存在的属性
   * @param target 目标配置对象
   * @param source 源配置对象
   * @returns 合并后的新配置对象
   */
  static safeMerge<T>(
    target: T,
    source: Partial<T>
  ): T {
    const result: T = {} as T;
    
    // 复制目标对象
    const targetKeys = Object.keys(target);
    for (let i = 0; i < targetKeys.length; i++) {
      const key = targetKeys[i];
      (result as Record<string, Object>)[key] = target[key];
    }
    
    // 安全合并源对象
    const sourceKeys = Object.keys(source);
    for (let i = 0; i < sourceKeys.length; i++) {
      const key = sourceKeys[i];
      const value = source[key];
      
      if (value !== undefined && value !== null) {
        // 检查目标对象是否有这个键
        if (targetKeys.includes(key)) {
          (result as Record<string, Object>)[key] = value;
        }
      }
    }
    
    return result;
  }
    
    // 合并源对象的属性
    const sourceKeys = Object.keys(source);
    for (let i = 0; i < sourceKeys.length; i++) {
      const key = sourceKeys[i];
      if (source[key] !== undefined) {
        result[key] = source[key];
      }
    }
    
    return result as T;
  }
  
  /**
   * 合并多个配置对象
   * @param target 目标配置对象
   * @param sources 多个源配置对象
   * @returns 合并后的新配置对象
   */
  static mergeMultiple<T extends Record<string, Object>>(
    target: T,
    sources: Record<string, Object>[]
  ): T {
    let result = target;
    
    for (let i = 0; i < sources.length; i++) {
      result = ConfigMerger.merge(result, sources[i]);
    }
    
    return result;
  }
  
  /**
   * 条件合并：只有当条件为真时才合并
   * @param target 目标配置对象
   * @param source 源配置对象
   * @param condition 合并条件
   * @returns 合并后的配置对象
   */
  static mergeIf<T extends Record<string, Object>>(
    target: T,
    source: Record<string, Object>,
    condition: boolean
  ): T {
    if (condition) {
      return ConfigMerger.merge(target, source);
    }
    return target;
  }
  
  /**
   * 安全合并：只合并存在的属性
   * @param target 目标配置对象
   * @param source 源配置对象
   * @returns 合并后的配置对象
   */
  static safeMerge<T extends Record<string, Object>>(
    target: T,
    source: Record<string, Object>
  ): T {
    const result: Record<string, Object> = {};
    
    // 复制目标对象
    const targetKeys = Object.keys(target);
    for (let i = 0; i < targetKeys.length; i++) {
      const key = targetKeys[i];
      result[key] = target[key];
    }
    
    // 安全合并源对象
    const sourceKeys = Object.keys(source);
    for (let i = 0; i < sourceKeys.length; i++) {
      const key = sourceKeys[i];
      const value = source[key];
      
      if (value !== undefined && value !== null) {
        // 检查目标对象是否有这个键
        if (targetKeys.includes(key)) {
          result[key] = value;
        }
      }
    }
    
    return result as T;
  }
  
  /**
   * 创建配置对象的副本
   * @param config 配置对象
   * @returns 配置对象的副本
   */
  static clone<T extends Record<string, Object>>(config: T): T {
    const result: Record<string, Object> = {};
    
    const keys = Object.keys(config);
    for (let i = 0; i < keys.length; i++) {
      const key = keys[i];
      result[key] = config[key];
    }
    
    return result as T;
  }
}

/**
 * 配置差异检测器
 */
export class ConfigDiffer {
  
  /**
   * 检测两个配置对象的差异
   * @param oldConfig 旧配置
   * @param newConfig 新配置
   * @returns 差异对象
   */
  static diff<T extends Record<string, Object>>(
    oldConfig: T,
    newConfig: T
  ): Record<string, Object> {
    const differences: Record<string, Object> = {};
    
    // 检查新配置中的变化
    const newKeys = Object.keys(newConfig);
    for (let i = 0; i < newKeys.length; i++) {
      const key = newKeys[i];
      const oldKeys = Object.keys(oldConfig);
      
      if (!oldKeys.includes(key) || oldConfig[key] !== newConfig[key]) {
        differences[key] = newConfig[key];
      }
    }
    
    return differences;
  }
  
  /**
   * 检测配置是否有变化
   * @param oldConfig 旧配置
   * @param newConfig 新配置
   * @returns 是否有变化
   */
  static hasChanges<T extends Record<string, Object>>(
    oldConfig: T,
    newConfig: T
  ): boolean {
    const diff = ConfigDiffer.diff(oldConfig, newConfig);
    return Object.keys(diff).length > 0;
  }
  
  /**
   * 获取变化的键列表
   * @param oldConfig 旧配置
   * @param newConfig 新配置
   * @returns 变化的键数组
   */
  static getChangedKeys<T extends Record<string, Object>>(
    oldConfig: T,
    newConfig: T
  ): string[] {
    const diff = ConfigDiffer.diff(oldConfig, newConfig);
    return Object.keys(diff);
  }
}

/**
 * 配置验证器
 */
export class ConfigValidator {
  
  /**
   * 验证配置对象的必需字段
   * @param config 配置对象
   * @param requiredKeys 必需的键
   * @returns 验证结果
   */
  static validateRequired<T extends Record<string, Object>>(
    config: T,
    requiredKeys: string[]
  ): ValidationResult {
    const missing: string[] = [];
    const configKeys = Object.keys(config);
    
    for (let i = 0; i < requiredKeys.length; i++) {
      const key = requiredKeys[i];
      if (!configKeys.includes(key) || config[key] === undefined || config[key] === null) {
        missing.push(key);
      }
    }
    
    const result: ValidationResult = {
      valid: missing.length === 0,
      missing
    };
    return result;
  }
  
  /**
   * 验证配置值的类型
   * @param config 配置对象
   * @param typeMap 类型映射
   * @returns 验证结果
   */
  static validateTypes<T extends Record<string, Object>>(
    config: T,
    typeMap: Record<string, string>
  ): TypeValidationResult {
    const errors: string[] = [];
    const configKeys = Object.keys(config);
    const typeKeys = Object.keys(typeMap);
    
    for (let i = 0; i < typeKeys.length; i++) {
      const key = typeKeys[i];
      if (configKeys.includes(key)) {
        const expectedType = typeMap[key];
        const actualType = typeof config[key];
        
        if (actualType !== expectedType) {
          errors.push(`键 "${key}" 期望类型 ${expectedType}，实际类型 ${actualType}`);
        }
      }
    }
    
    const result: TypeValidationResult = {
      valid: errors.length === 0,
      errors
    };
    return result;
  }
  
  /**
   * 验证数值范围
   * @param config 配置对象
   * @param ranges 范围映射
   * @returns 验证结果
   */
  static validateRanges<T extends Record<string, Object>>(
    config: T,
    ranges: Record<string, RangeConfig>
  ): RangeValidationResult {
    const errors: string[] = [];
    const configKeys = Object.keys(config);
    const rangeKeys = Object.keys(ranges);
    
    for (let i = 0; i < rangeKeys.length; i++) {
      const key = rangeKeys[i];
      if (configKeys.includes(key)) {
        const value = config[key];
        const range = ranges[key];
        
        if (typeof value === 'number') {
          if (range.min !== undefined && value < range.min) {
            errors.push(`键 "${key}" 的值 ${value} 小于最小值 ${range.min}`);
          }
          
          if (range.max !== undefined && value > range.max) {
            errors.push(`键 "${key}" 的值 ${value} 大于最大值 ${range.max}`);
          }
        }
      }
    }
    
    const result: RangeValidationResult = {
      valid: errors.length === 0,
      errors
    };
    return result;
  }
}