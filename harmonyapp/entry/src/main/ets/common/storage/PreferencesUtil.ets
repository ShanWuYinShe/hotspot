/**
 * 简化的存储工具类
 * 直接使用HarmonyOS Preferences API，去除复杂的封装层
 */

import preferences from '@ohos.data.preferences';
import { common } from '@kit.AbilityKit';

export class PreferencesUtil {
  private static store: preferences.Preferences | null = null;

  /**
   * 初始化存储
   */
  static async init(context: common.UIAbilityContext) {
    if (!PreferencesUtil.store) {
      PreferencesUtil.store = await preferences.getPreferences(context, 'geotalk');
    }
  }

  /**
   * 保存数据
   */
  static async set(key: string, value: preferences.ValueType): Promise<void> {
    if (!PreferencesUtil.store) throw new Error('Storage not initialized');
    await PreferencesUtil.store.put(key, value);
    await PreferencesUtil.store.flush();
  }

  /**
   * 获取数据
   */
  static async get<T extends preferences.ValueType>(key: string, defaultValue: T): Promise<T> {
    if (!PreferencesUtil.store) throw new Error('Storage not initialized');
    const result = await PreferencesUtil.store.get(key, defaultValue);
    return result as T;
  }

  /**
   * 删除数据
   */
  static async remove(key: string): Promise<void> {
    if (!PreferencesUtil.store) throw new Error('Storage not initialized');
    await PreferencesUtil.store.delete(key);
    await PreferencesUtil.store.flush();
  }

  /**
   * 清空所有数据
   */
  static async clear(): Promise<void> {
    if (!PreferencesUtil.store) throw new Error('Storage not initialized');
    await PreferencesUtil.store.clear();
    await PreferencesUtil.store.flush();
  }

  /**
   * 保存对象（自动序列化）
   */
  static async setObject(key: string, value: Object): Promise<void> {
    await PreferencesUtil.set(key, JSON.stringify(value));
  }

  /**
   * 获取对象（自动反序列化）
   */
  static async getObject<T>(key: string): Promise<T | null> {
    const value: string = await PreferencesUtil.get<string>(key, '');
    if (!value) return null;
    try {
      return JSON.parse(value) as T;
    } catch {
      return null;
    }
  }
}
