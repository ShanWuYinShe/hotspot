/**
 * 多账户会话管理（Harmony App）
 *
 * 目标：
 * - 本地可保存多个账户（每个账户保存 AppAuthData，其中包含 refresh_token）
 * - 可在账户之间切换（切换时更新当前 access_token/user_info，并重连 WS）
 * - “退出登录”默认只做本地锁定：清除当前账号的 access_token/user_info，但保留 accounts 列表
 *
 * 安全语义：
 * - 敏感数据（各账号 AuthData，含 refresh_token）存系统 Asset 存储
 *   （TokenAssetStore，按 user_id 独立别名）；Preferences 的 AUTH_ACCOUNTS
 *   只保存非敏感索引（user_id 列表 + current_user_id）
 * - 为避免“串数据”，切换/锁定时应清理本地消息缓存（由调用方触发）
 */

import { StorageKeys } from '../../constants/AppConstants';
import { PreferencesUtil } from '../storage/PreferencesUtil';
import { TokenAssetStore } from '../storage/TokenAssetStore';
import type { AuthData } from '../types/api';

/**
 * 多账号索引（非敏感）：仅 user_id 列表与当前活跃账号。
 * 各账号 AuthData 的敏感内容存 Asset（见 TokenAssetStore）。
 */
interface AuthAccountsIndex {
  version: 1;
  current_user_id?: string;
  user_ids: string[];
}

const DEFAULT_INDEX: AuthAccountsIndex = {
  version: 1,
  user_ids: []
};

export class AuthAccountManager {
  private static async getIndex(): Promise<AuthAccountsIndex> {
    const raw = await PreferencesUtil.getObject<AuthAccountsIndex>(StorageKeys.AUTH_ACCOUNTS);
    if (!raw || raw.version !== 1 || !Array.isArray(raw.user_ids)) {
      const defaultIndex: AuthAccountsIndex = {
        version: DEFAULT_INDEX.version,
        user_ids: []
      };
      return defaultIndex;
    }
    return raw;
  }

  private static async setIndex(index: AuthAccountsIndex): Promise<void> {
    await PreferencesUtil.setObject(StorageKeys.AUTH_ACCOUNTS, index);
  }

  /**
   * 把一个 user_id 并入索引列表（保持原有顺序，去重）
   */
  private static withUserId(index: AuthAccountsIndex, userId: string): AuthAccountsIndex {
    const user_ids: string[] = [];
    for (let i = 0; i < index.user_ids.length; i++) {
      if (index.user_ids[i] !== userId) {
        user_ids.push(index.user_ids[i]);
      }
    }
    user_ids.push(userId);
    const next: AuthAccountsIndex = {
      version: index.version,
      current_user_id: index.current_user_id,
      user_ids: user_ids
    };
    return next;
  }

  /**
   * 从索引列表移除一个 user_id
   */
  private static withoutUserId(index: AuthAccountsIndex, userId: string): AuthAccountsIndex {
    const user_ids: string[] = [];
    for (let i = 0; i < index.user_ids.length; i++) {
      if (index.user_ids[i] !== userId) {
        user_ids.push(index.user_ids[i]);
      }
    }
    const next: AuthAccountsIndex = {
      version: index.version,
      current_user_id: index.current_user_id === userId ? undefined : index.current_user_id,
      user_ids: user_ids
    };
    return next;
  }

  /**
   * 读取单账号的 AuthData（Asset 存储，不存在返回 null）
   */
  private static async readAccountAuth(userId: string): Promise<AuthData | null> {
    return TokenAssetStore.getObject<AuthData>(TokenAssetStore.accountAuthAlias(userId));
  }

  /**
   * 写入单账号的 AuthData（Asset 存储）
   */
  private static async writeAccountAuth(auth: AuthData): Promise<void> {
    await TokenAssetStore.set(TokenAssetStore.accountAuthAlias(String(auth.user_id)), JSON.stringify(auth));
  }

  /**
   * 更新当前会话快捷键（Asset 存储）
   */
  private static async writeSessionKeys(auth: AuthData): Promise<void> {
    await TokenAssetStore.set(TokenAssetStore.ALIAS_CURRENT_AUTH, JSON.stringify(auth));
    await TokenAssetStore.set(TokenAssetStore.ALIAS_CURRENT_ACCESS, auth.access_token);
  }

  /**
   * 清除当前会话快捷键（Asset 存储）
   */
  private static async clearSessionKeys(): Promise<void> {
    await TokenAssetStore.remove(TokenAssetStore.ALIAS_CURRENT_AUTH);
    await TokenAssetStore.remove(TokenAssetStore.ALIAS_CURRENT_ACCESS);
  }

  static async listAccounts(): Promise<AuthData[]> {
    const index = await AuthAccountManager.getIndex();
    const accounts: AuthData[] = [];
    for (let i = 0; i < index.user_ids.length; i++) {
      const auth = await AuthAccountManager.readAccountAuth(index.user_ids[i]);
      if (auth) {
        accounts.push(auth);
      }
    }
    return accounts;
  }

  static async getCurrentUserId(): Promise<string | null> {
    const index = await AuthAccountManager.getIndex();
    return typeof index.current_user_id === 'string' ? index.current_user_id : null;
  }

  static async getCurrentAuth(): Promise<AuthData | null> {
    const index = await AuthAccountManager.getIndex();
    const currentId = typeof index.current_user_id === 'string' ? index.current_user_id : undefined;
    if (currentId == null) return null;
    return await AuthAccountManager.readAccountAuth(String(currentId));
  }

  /**
   * 仅保存/更新一个账号（不改当前活跃账号、不写会话快捷键）。
   * 场景：后台 token 刷新完成时用户已切到其它账号——只落库，避免把旧账号
   * 重新激活造成串数据。
   */
  static async upsertOnly(auth: AuthData): Promise<void> {
    const index = await AuthAccountManager.getIndex();
    await AuthAccountManager.writeAccountAuth(auth);
    await AuthAccountManager.setIndex(AuthAccountManager.withUserId(index, String(auth.user_id)));
  }

  /**
   * 保存/更新一个账号，并将其设为当前活跃账号
   * 同步更新当前会话快捷键（access_token/当前会话 AuthData）
   */
  static async upsertAndActivate(auth: AuthData): Promise<void> {
    await AuthAccountManager.upsertOnly(auth);
    const index = await AuthAccountManager.getIndex();
    const activated: AuthAccountsIndex = {
      version: index.version,
      current_user_id: auth.user_id,
      user_ids: index.user_ids
    };
    await AuthAccountManager.setIndex(activated);

    // 维持一份“当前账号”的 AuthData/access_token 作为会话快捷读取键
    await AuthAccountManager.writeSessionKeys(auth);
  }

  /**
   * 切换到已保存账号（会更新 access_token/user_info）
   * 返回切换后的 auth（不存在则返回 null）
   */
  static async switchTo(userId: string): Promise<AuthData | null> {
    const index = await AuthAccountManager.getIndex();
    const next = await AuthAccountManager.readAccountAuth(userId);
    if (!next) return null;

    const newState: AuthAccountsIndex = {
      version: index.version,
      current_user_id: userId,
      user_ids: index.user_ids
    };
    await AuthAccountManager.setIndex(newState);

    await AuthAccountManager.writeSessionKeys(next);
    return next;
  }

  /**
   * 本地锁定当前账号：清除当前 access_token/user_info（不删除 accounts 列表）
   */
  static async lockCurrent(): Promise<void> {
    const index = await AuthAccountManager.getIndex();
    const newState: AuthAccountsIndex = {
      version: index.version,
      user_ids: index.user_ids
    };
    await AuthAccountManager.setIndex(newState);

    await AuthAccountManager.clearSessionKeys();
  }

  /**
   * 删除一个账号（如果删除的是当前账号，会自动 lock）
   */
  static async removeAccount(userId: string): Promise<void> {
    const index = await AuthAccountManager.getIndex();
    // 先删该账号的 Asset 条目，不残留 refresh_token
    await TokenAssetStore.remove(TokenAssetStore.accountAuthAlias(userId));
    if (index.current_user_id === userId) {
      await AuthAccountManager.clearSessionKeys();
    }
    await AuthAccountManager.setIndex(AuthAccountManager.withoutUserId(index, userId));
  }

  /**
   * 清空所有账号（危险操作）：逐个删除各账号的 Asset 条目与会话快捷键
   */
  static async clearAllAccounts(): Promise<void> {
    const index = await AuthAccountManager.getIndex();
    for (let i = 0; i < index.user_ids.length; i++) {
      await TokenAssetStore.remove(TokenAssetStore.accountAuthAlias(index.user_ids[i]));
    }
    await AuthAccountManager.clearSessionKeys();
    await PreferencesUtil.remove(StorageKeys.AUTH_ACCOUNTS);
  }
}
