/**
 * 配置合并工具 - ArkTS兼容版本
 * 提供简化的配置合并功能
 */

import { hilog } from '@kit.PerformanceAnalysisKit';

declare const structuredClone: (<T>(value: T) => T) | undefined;

const DOMAIN_NUMBER: number = 0xFF22;
const TAG: string = 'ConfigMerger';

function deepClone(value: Record<string, Object>): Record<string, Object>;
function deepClone(value: Object[]): Object[];
function deepClone(value: Record<string, Object> | Object[]): Record<string, Object> | Object[] {
  try {
    if (typeof structuredClone === 'function') {
      return structuredClone(value);
    }
  } catch (error) {
    hilog.warn(DOMAIN_NUMBER, TAG, `structuredClone failed, falling back to manual clone: ${JSON.stringify(error)}`);
  }
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
