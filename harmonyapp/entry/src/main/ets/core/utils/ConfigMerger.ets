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

declare const structuredClone: (<T>(value: T) => T) | undefined;

function deepClone(value: Record<string, Object>): Record<string, Object>;
function deepClone(value: Object[]): Object[];
function deepClone(value: Record<string, Object> | Object[]): Record<string, Object> | Object[] {
  try {
    if (typeof structuredClone === 'function') {
      return structuredClone(value);
    }
  } catch (_) {}
  if (Array.isArray(value)) {
    const result: Object[] = [];
    const sourceArray = value as Object[];
    for (let i = 0; i < sourceArray.length; i++) {
      const current = sourceArray[i];
      if (Array.isArray(current)) {
        result[i] = deepClone(current as Object[]);
      } else if (current !== null && typeof current === 'object') {
        result[i] = deepClone(current as Record<string, Object>);
      } else {
        result[i] = current;
      }
    }
    return result;
  }
  const sourceObject = value as Record<string, Object>;
  const clonedObject: Record<string, Object> = {};
  const keys = Object.keys(sourceObject);
  for (let i = 0; i < keys.length; i++) {
    const key = keys[i];
    const currentValue = sourceObject[key];
    if (Array.isArray(currentValue)) {
      clonedObject[key] = deepClone(currentValue as Object[]);
    } else if (currentValue !== null && typeof currentValue === 'object') {
      clonedObject[key] = deepClone(currentValue as Record<string, Object>);
    } else {
      clonedObject[key] = currentValue;
    }
  }
  return clonedObject;
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
  static merge(base: Object, override: Object): Object {
    const baseObj: Record<string, Object> = deepClone(base as Record<string, Object>);
    const overrideObj: Record<string, Object> = deepClone(override as Record<string, Object>);
    const keys = Object.keys(overrideObj);
    for (let i = 0; i < keys.length; i++) {
      const key = keys[i];
      baseObj[key] = overrideObj[key];
    }
    return baseObj as Object;
  }
  
  /**
   * 合并多个配置对象
   * @param target 目标配置对象
   * @param sources 多个源配置对象
   * @returns 合并后的新配置对象
   */
  static mergeMultiple(
    target: Object,
    sources: Object[]
  ): Object {
    let result: Object = target;
    
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
  static mergeIf(
    target: Object,
    source: Object,
    condition: boolean
  ): Object {
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
  static safeMerge(
    target: Object,
    source: Object
  ): Object {
    const baseObj: Record<string, Object> = deepClone(target as Record<string, Object>);
    const overrideObj: Record<string, Object> = deepClone(source as Record<string, Object>);
    const targetKeys = Object.keys(baseObj);
    const sourceKeys = Object.keys(overrideObj);
    
    // 安全合并源对象
    for (let i = 0; i < sourceKeys.length; i++) {
      const key = sourceKeys[i];
      const value = overrideObj[key];
      
      if (value !== undefined && value !== null) {
        // 检查目标对象是否有这个键
        if (targetKeys.includes(key)) {
          baseObj[key] = value;
        }
      }
    }
    
    return baseObj as Object;
  }
  
  /**
   * 创建配置对象的副本
   * @param config 配置对象
   * @returns 配置对象的副本
   */
  static clone(config: Record<string, Object>): Record<string, Object> {
    return deepClone(config);
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
  static diff(
    oldConfig: Record<string, Object>,
    newConfig: Record<string, Object>
  ): Record<string, Object> {
    const differences: Record<string, Object> = {};
    
    // 检查新配置中的变化
    const newKeys = Object.keys(newConfig);
    const oldKeys = Object.keys(oldConfig);
    for (let i = 0; i < newKeys.length; i++) {
      const key = newKeys[i];
      
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
  static hasChanges(
    oldConfig: Record<string, Object>,
    newConfig: Record<string, Object>
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
  static getChangedKeys(
    oldConfig: Record<string, Object>,
    newConfig: Record<string, Object>
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
  static validateRequired(
    config: Record<string, Object>,
    requiredKeys: string[]
  ): ValidationResult {
    const missing: string[] = [];
    const configKeys = Object.keys(config);
    
    for (let i = 0; i < requiredKeys.length; i++) {
      const key = requiredKeys[i];
      const configRecord = config as Record<string, Object>;
      if (!configKeys.includes(key) || configRecord[key] === undefined || configRecord[key] === null) {
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
  static validateTypes(
    config: Record<string, Object>,
    typeMap: Record<string, string>
  ): TypeValidationResult {
    const errors: string[] = [];
    const configKeys = Object.keys(config);
    const typeKeys = Object.keys(typeMap);
    const configRecord = config as Record<string, Object>;
    
    for (let i = 0; i < typeKeys.length; i++) {
      const key = typeKeys[i];
      if (configKeys.includes(key)) {
        const expectedType = typeMap[key];
        const actualType = typeof configRecord[key];
        
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
  static validateRanges(
    config: Record<string, Object>,
    ranges: Record<string, RangeConfig>
  ): RangeValidationResult {
    const errors: string[] = [];
    const configKeys = Object.keys(config);
    const rangeKeys = Object.keys(ranges);
    const configRecord = config as Record<string, Object>;
    
    for (let i = 0; i < rangeKeys.length; i++) {
      const key = rangeKeys[i];
      if (configKeys.includes(key)) {
        const value = configRecord[key];
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
