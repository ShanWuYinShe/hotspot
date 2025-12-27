/**
 * 多账户会话管理（Harmony App）
 *
 * 目标：
 * - 本地可保存多个账户（每个账户保存 AppAuthData，其中包含 refresh_token）
 * - 可在账户之间切换（切换时更新当前 access_token/user_info，并重连 WS）
 * - “退出登录”默认只做本地锁定：清除当前账号的 access_token/user_info，但保留 accounts 列表
 *
 * 安全语义：
 * - 为避免“串数据”，切换/锁定时应清理本地消息缓存（由调用方触发）
 */

import { StorageKeys } from '../../constants/AppConstants';
import { PreferencesUtil } from '../storage/PreferencesUtil';
import type { AppAuthData } from '../types/api';

interface AuthAccountsState {
  version: 1;
  current_user_id?: number;
  accounts: Record<string, AppAuthData>;
}

const DEFAULT_STATE: AuthAccountsState = {
  version: 1,
  accounts: {}
};

export class AuthAccountManager {
  private static async getState(): Promise<AuthAccountsState> {
    const raw = await PreferencesUtil.getObject<AuthAccountsState>(StorageKeys.AUTH_ACCOUNTS);
    if (!raw || raw.version !== 1 || !raw.accounts) return { ...DEFAULT_STATE };
    return raw;
  }

  private static async setState(state: AuthAccountsState): Promise<void> {
    await PreferencesUtil.setObject(StorageKeys.AUTH_ACCOUNTS, state);
  }

  static async listAccounts(): Promise<AppAuthData[]> {
    const state = await this.getState();
    return Object.values(state.accounts);
  }

  static async getCurrentUserId(): Promise<number | null> {
    const state = await this.getState();
    return typeof state.current_user_id === 'number' ? state.current_user_id : null;
  }

  static async getCurrentAuth(): Promise<AppAuthData | null> {
    const state = await this.getState();
    const currentId = typeof state.current_user_id === 'number' ? state.current_user_id : undefined;
    if (currentId == null) return null;
    return state.accounts[String(currentId)] ?? null;
  }

  /**
   * 保存/更新一个账号，并将其设为当前活跃账号（同时写回兼容键：access_token/user_info）
   */
  static async upsertAndActivate(auth: AppAuthData): Promise<void> {
    const state = await this.getState();
    state.accounts[String(auth.user_id)] = auth;
    state.current_user_id = auth.user_id;
    await this.setState(state);

    // 兼容旧逻辑：仍维持一份“当前账号”的 user_info/access_token
    await PreferencesUtil.setObject(StorageKeys.USER_INFO, auth);
    await PreferencesUtil.set(StorageKeys.ACCESS_TOKEN, auth.access_token);
  }

  /**
   * 切换到已保存账号（会更新 access_token/user_info）
   * 返回切换后的 auth（不存在则返回 null）
   */
  static async switchTo(userId: number): Promise<AppAuthData | null> {
    const state = await this.getState();
    const next = state.accounts[String(userId)];
    if (!next) return null;

    state.current_user_id = userId;
    await this.setState(state);

    await PreferencesUtil.setObject(StorageKeys.USER_INFO, next);
    await PreferencesUtil.set(StorageKeys.ACCESS_TOKEN, next.access_token);
    return next;
  }

  /**
   * 本地锁定当前账号：清除当前 access_token/user_info（不删除 accounts 列表）
   */
  static async lockCurrent(): Promise<void> {
    const state = await this.getState();
    delete state.current_user_id;
    await this.setState(state);

    await PreferencesUtil.remove(StorageKeys.USER_INFO);
    await PreferencesUtil.remove(StorageKeys.ACCESS_TOKEN);
  }

  /**
   * 删除一个账号（如果删除的是当前账号，会自动 lock）
   */
  static async removeAccount(userId: number): Promise<void> {
    const state = await this.getState();
    delete state.accounts[String(userId)];
    if (state.current_user_id === userId) {
      delete state.current_user_id;
      await PreferencesUtil.remove(StorageKeys.USER_INFO);
      await PreferencesUtil.remove(StorageKeys.ACCESS_TOKEN);
    }
    await this.setState(state);
  }

  /**
   * 清空所有账号（危险操作）
   */
  static async clearAllAccounts(): Promise<void> {
    await PreferencesUtil.remove(StorageKeys.AUTH_ACCOUNTS);
    await PreferencesUtil.remove(StorageKeys.USER_INFO);
    await PreferencesUtil.remove(StorageKeys.ACCESS_TOKEN);
  }
}

