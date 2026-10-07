/**
 * 敏感凭据存储工具类（HarmonyOS Asset Store Kit 封装）
 *
 * 将 access_token、auth_data（含 refresh_token）、多账号 AuthData 等敏感数据
 * 从 Preferences 明文存储迁移到系统级 Asset 存储（加密、按应用沙箱隔离、
 * 默认卸载即清除——不设置 IS_PERSISTENT）。
 *
 * 设计：
 * - 每个敏感项一个 asset 别名，value 为 JSON 字符串（存为 Uint8Array）；
 * - set/get/remove 均为异步静态方法，风格对齐 PreferencesUtil：
 *   失败记 hilog 日志不抛出；get 失败或条目不存在返回 null；
 * - 不做旧明文数据迁移（破坏性替换，快速迭代期已授权）。
 */

import asset from '@ohos.security.asset';
import util from '@ohos.util';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { PreferencesUtil } from './PreferencesUtil';
import { StorageKeys } from '../../constants/AppConstants';

const DOMAIN_NUMBER: number = 0xFF37;
const TAG: string = 'TokenAssetStore';

export class TokenAssetStore {
  /** 当前会话 access_token 快捷键 */
  static readonly ALIAS_CURRENT_ACCESS: string = 'geotalk_current_access';
  /** 当前会话 AuthData（含 refresh_token）快捷键 */
  static readonly ALIAS_CURRENT_AUTH: string = 'geotalk_current_auth';

  /**
   * 一次性清理升级前 Preferences 明文时代的 token 残留：96563d4a 迁 Asset 后
   * 旧键（auth_data/access_token）不再被读取，但升级设备上的旧明文值不会被
   * 自动清除，会一直留在应用沙盒内。幂等（键不存在时 remove 无害）；在
   * PreferencesUtil.init 之后调用（EntryAbility）。
   */
  static async removeLegacyPlaintextTokens(): Promise<void> {
    try {
      await PreferencesUtil.remove(StorageKeys.AUTH_DATA);
      await PreferencesUtil.remove(StorageKeys.ACCESS_TOKEN);
      hilog.info(DOMAIN_NUMBER, TAG, 'Preferences 明文 token 残留已清理（如存在）');
    } catch (error) {
      hilog.warn(DOMAIN_NUMBER, TAG, `明文 token 清理失败: ${JSON.stringify(error)}`);
    }
  }

  /** 多账号：每个账号的 AuthData 按独立别名存储 */
  static accountAuthAlias(userId: string): string {
    return `geotalk_account_${userId}_auth`;
  }

  /**
   * UTF-8 字符串 → 字节
   */
  private static encode(value: string): Uint8Array {
    return new util.TextEncoder().encodeInto(value);
  }

  /**
   * 字节 → UTF-8 字符串
   */
  private static decode(value: Uint8Array): string {
    return util.TextDecoder.create('utf-8').decodeToString(value);
  }

  /**
   * 写入（别名不存在则新增，已存在则覆盖）
   */
  static async set(alias: string, value: string): Promise<void> {
    try {
      const attributes: asset.AssetMap = new Map<asset.Tag, asset.Value>();
      attributes.set(asset.Tag.ALIAS, TokenAssetStore.encode(alias));
      attributes.set(asset.Tag.SECRET, TokenAssetStore.encode(value));
      attributes.set(asset.Tag.CONFLICT_RESOLUTION, asset.ConflictResolution.OVERWRITE);
      await asset.add(attributes);
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `Failed to set asset ${alias}: ${error}`);
    }
  }

  /**
   * 读取。条目不存在（24000002）或读取失败返回 null。
   */
  static async get(alias: string): Promise<string | null> {
    try {
      const query: asset.AssetMap = new Map<asset.Tag, asset.Value>();
      query.set(asset.Tag.ALIAS, TokenAssetStore.encode(alias));
      query.set(asset.Tag.RETURN_TYPE, asset.ReturnType.ALL);
      query.set(asset.Tag.RETURN_LIMIT, 1);
      const results: Array<asset.AssetMap> = await asset.query(query);
      if (results.length === 0) return null;
      const secret = results[0].get(asset.Tag.SECRET);
      if (!(secret instanceof Uint8Array)) return null;
      return TokenAssetStore.decode(secret);
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `Failed to get asset ${alias}: ${error}`);
      return null;
    }
  }

  /**
   * 删除。条目不存在视为已删除（幂等），其余失败记日志不抛出。
   */
  static async remove(alias: string): Promise<void> {
    try {
      const query: asset.AssetMap = new Map<asset.Tag, asset.Value>();
      query.set(asset.Tag.ALIAS, TokenAssetStore.encode(alias));
      await asset.remove(query);
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `Failed to remove asset ${alias}: ${error}`);
    }
  }

  /**
   * 读取 JSON 对象（便捷封装）
   */
  static async getObject<T>(alias: string): Promise<T | null> {
    const raw = await TokenAssetStore.get(alias);
    if (!raw) return null;
    try {
      return JSON.parse(raw) as T;
    } catch {
      return null;
    }
  }
}
