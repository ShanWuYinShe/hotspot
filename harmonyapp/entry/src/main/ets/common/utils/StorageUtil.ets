/**
 * 存储工具类
 * 提供统一的本地存储功能
 */
import { preferences } from '@kit.ArkData';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { common } from '@kit.AbilityKit';

const DOMAIN_NUMBER: number = 0xFF00;
const TAG: string = 'StorageUtil';
const STORE_NAME: string = 'app_preferences';

export class StorageUtil {
  private static preferencesStore: preferences.Preferences | null = null;

  /**
   * 初始化存储
   */
  static async initialize(context: common.Context): Promise<void> {
    try {
      if (!StorageUtil.preferencesStore) {
        StorageUtil.preferencesStore = await preferences.getPreferences(context, STORE_NAME);
        hilog.info(DOMAIN_NUMBER, TAG, '存储初始化成功');
      }
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `存储初始化失败: ${JSON.stringify(error)}`);
      throw new Error(`存储操作失败: ${JSON.stringify(error)}`);
    }
  }

  /**
   * 存储字符串值
   */
  static async setString(key: string, value: string): Promise<void> {
    try {
      if (!StorageUtil.preferencesStore) {
        throw new Error('存储未初始化');
      }
      await StorageUtil.preferencesStore.put(key, value);
      await StorageUtil.preferencesStore.flush();
      hilog.debug(DOMAIN_NUMBER, TAG, `存储字符串: ${key}`);
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `存储字符串失败: ${key}, ${JSON.stringify(error)}`);
      throw new Error(`存储操作失败: ${JSON.stringify(error)}`);
    }
  }

  /**
   * 获取字符串值
   */
  static async getString(key: string, defaultValue: string = ''): Promise<string> {
    try {
      if (!StorageUtil.preferencesStore) {
        throw new Error('存储未初始化');
      }
      const value = await StorageUtil.preferencesStore.get(key, defaultValue);
      return value as string;
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `获取字符串失败: ${key}, ${JSON.stringify(error)}`);
      return defaultValue;
    }
  }

  /**
   * 存储数字值
   */
  static async setNumber(key: string, value: number): Promise<void> {
    try {
      if (!StorageUtil.preferencesStore) {
        throw new Error('存储未初始化');
      }
      await StorageUtil.preferencesStore.put(key, value);
      await StorageUtil.preferencesStore.flush();
      hilog.debug(DOMAIN_NUMBER, TAG, `存储数字: ${key}`);
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `存储数字失败: ${key}, ${JSON.stringify(error)}`);
      throw new Error(`存储操作失败: ${JSON.stringify(error)}`);
    }
  }

  /**
   * 获取数字值
   */
  static async getNumber(key: string, defaultValue: number = 0): Promise<number> {
    try {
      if (!StorageUtil.preferencesStore) {
        throw new Error('存储未初始化');
      }
      const value = await StorageUtil.preferencesStore.get(key, defaultValue);
      return value as number;
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `获取数字失败: ${key}, ${JSON.stringify(error)}`);
      return defaultValue;
    }
  }

  /**
   * 存储布尔值
   */
  static async setBoolean(key: string, value: boolean): Promise<void> {
    try {
      if (!StorageUtil.preferencesStore) {
        throw new Error('存储未初始化');
      }
      await StorageUtil.preferencesStore.put(key, value);
      await StorageUtil.preferencesStore.flush();
      hilog.debug(DOMAIN_NUMBER, TAG, `存储布尔值: ${key}`);
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `存储布尔值失败: ${key}, ${JSON.stringify(error)}`);
      throw new Error(`存储操作失败: ${JSON.stringify(error)}`);
    }
  }

  /**
   * 获取布尔值
   */
  static async getBoolean(key: string, defaultValue: boolean = false): Promise<boolean> {
    try {
      if (!StorageUtil.preferencesStore) {
        throw new Error('存储未初始化');
      }
      const value = await StorageUtil.preferencesStore.get(key, defaultValue);
      return value as boolean;
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `获取布尔值失败: ${key}, ${JSON.stringify(error)}`);
      return defaultValue;
    }
  }

  /**
   * 存储对象
   */
  static async setObject<T>(key: string, value: T): Promise<void> {
    try {
      const jsonString = JSON.stringify(value);
      await StorageUtil.setString(key, jsonString);
      hilog.debug(DOMAIN_NUMBER, TAG, `存储对象: ${key}`);
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `存储对象失败: ${key}, ${JSON.stringify(error)}`);
      throw new Error(`存储操作失败: ${JSON.stringify(error)}`);
    }
  }

  /**
   * 获取对象
   */
  static async getObject<T>(key: string, defaultValue: T | null = null): Promise<T | null> {
    try {
      const jsonString = await StorageUtil.getString(key, '');
      if (!jsonString) {
        return defaultValue;
      }
      return JSON.parse(jsonString) as T;
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `获取对象失败: ${key}, ${JSON.stringify(error)}`);
      return defaultValue;
    }
  }

  /**
   * 删除键值
   */
  static async remove(key: string): Promise<void> {
    try {
      if (!StorageUtil.preferencesStore) {
        throw new Error('存储未初始化');
      }
      await StorageUtil.preferencesStore.delete(key);
      await StorageUtil.preferencesStore.flush();
      hilog.debug(DOMAIN_NUMBER, TAG, `删除键值: ${key}`);
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `删除键值失败: ${key}, ${JSON.stringify(error)}`);
      throw new Error(`存储操作失败: ${JSON.stringify(error)}`);
    }
  }

  /**
   * 清空所有数据
   */
  static async clear(): Promise<void> {
    try {
      if (!StorageUtil.preferencesStore) {
        throw new Error('存储未初始化');
      }
      await StorageUtil.preferencesStore.clear();
      await StorageUtil.preferencesStore.flush();
      hilog.info(DOMAIN_NUMBER, TAG, '清空所有存储数据');
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `清空存储失败: ${JSON.stringify(error)}`);
      throw new Error(`存储操作失败: ${JSON.stringify(error)}`);
    }
  }

  /**
   * 检查键是否存在
   */
  static async has(key: string): Promise<boolean> {
    try {
      if (!StorageUtil.preferencesStore) {
        throw new Error('存储未初始化');
      }
      return await StorageUtil.preferencesStore.has(key);
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `检查键存在失败: ${key}, ${JSON.stringify(error)}`);
      return false;
    }
  }

  /**
   * 获取所有键
   */
  static async getAllKeys(): Promise<string[]> {
    try {
      if (!StorageUtil.preferencesStore) {
        throw new Error('存储未初始化');
      }
      // HarmonyOS Preferences API 没有 getAllKeys 方法
      // 返回空数组或实现自定义键管理
      hilog.warn(DOMAIN_NUMBER, TAG, 'getAllKeys方法不可用，返回空数组');
      return [];
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `获取所有键失败: ${JSON.stringify(error)}`);
      return [];
    }
  }
}
