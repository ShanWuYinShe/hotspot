/**
 * 简化的存储工具类
 * 直接使用HarmonyOS Preferences API，去除复杂的封装层
 */

import { preferences } from '@kit.ArkData';
import { common } from '@kit.AbilityKit';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { PreferenceStores } from '../../constants/AppConstants';

const DOMAIN_NUMBER: number = 0xFF37;
const TAG: string = 'PreferencesUtil';

export class PreferencesUtil {
  private static store: preferences.Preferences | null = null;

  /**
   * 初始化存储
   */
  static async init(context: common.UIAbilityContext) {
    if (!PreferencesUtil.store) {
      try {
        PreferencesUtil.store = await preferences.getPreferences(context, PreferenceStores.GEOTALK);
      } catch (error) {
        // Handle error
        hilog.error(DOMAIN_NUMBER, TAG, `Failed to get preferences: ${error}`);
      }
    }
  }

  /**
   * 保存数据
   */
  static async set(key: string, value: preferences.ValueType): Promise<void> {
    if (!PreferencesUtil.store) {
      // 未初始化直接降级返回（不抛出）：调用方遍布启动链路，抛出会中断
      // 初始化；记日志保证问题可排查
      hilog.error(DOMAIN_NUMBER, TAG, `Preferences not ready, drop set ${key}`);
      return;
    }
    try {
      await PreferencesUtil.store.put(key, value);
      await PreferencesUtil.store.flush();
    } catch (error) {
       hilog.error(DOMAIN_NUMBER, TAG, `Failed to set preference ${key}: ${error}`);
    }
  }

  /**
   * 获取数据
   */
  static async get<T extends preferences.ValueType>(key: string, defaultValue: T): Promise<T> {
    if (!PreferencesUtil.store) {
      hilog.error(DOMAIN_NUMBER, TAG, `Preferences not ready, fallback default for ${key}`);
      return defaultValue;
    }
    try {
      const result = await PreferencesUtil.store.get(key, defaultValue);
      return result as T;
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `Failed to get preference ${key}: ${error}`);
      return defaultValue;
    }
  }

  /**
   * 删除数据
   */
  static async remove(key: string): Promise<void> {
    if (!PreferencesUtil.store) {
      hilog.error(DOMAIN_NUMBER, TAG, `Preferences not ready, drop remove ${key}`);
      return;
    }
    try {
      await PreferencesUtil.store.delete(key);
      await PreferencesUtil.store.flush();
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `Failed to remove preference ${key}: ${error}`);
    }
  }

  /**
   * 清空所有数据
   */
  static async clear(): Promise<void> {
    if (!PreferencesUtil.store) {
      hilog.error(DOMAIN_NUMBER, TAG, 'Preferences not ready, drop clear');
      return;
    }
    try {
      await PreferencesUtil.store.clear();
      await PreferencesUtil.store.flush();
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `Failed to clear preferences: ${error}`);
    }
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
