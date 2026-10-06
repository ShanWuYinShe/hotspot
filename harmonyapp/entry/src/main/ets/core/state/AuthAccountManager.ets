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
import type { AuthData } from '../types/api';

interface AuthAccountsState {
  version: 1;
  current_user_id?: string;
  accounts: Record<string, AuthData>;
}

const DEFAULT_STATE: AuthAccountsState = {
  version: 1,
  accounts: {}
};

export class AuthAccountManager {
  private static async getState(): Promise<AuthAccountsState> {
    const raw = await PreferencesUtil.getObject<AuthAccountsState>(StorageKeys.AUTH_ACCOUNTS);
    if (!raw || raw.version !== 1 || !raw.accounts) {
      const defaultState: AuthAccountsState = {
        version: DEFAULT_STATE.version,
        accounts: {}
      };
      return defaultState;
    }
    return raw;
  }

  private static async setState(state: AuthAccountsState): Promise<void> {
    await PreferencesUtil.setObject(StorageKeys.AUTH_ACCOUNTS, state);
  }

  static async listAccounts(): Promise<AuthData[]> {
    const state = await AuthAccountManager.getState();
    return Object.values(state.accounts);
  }

  static async getCurrentUserId(): Promise<string | null> {
    const state = await AuthAccountManager.getState();
    return typeof state.current_user_id === 'string' ? state.current_user_id : null;
  }

  static async getCurrentAuth(): Promise<AuthData | null> {
    const state = await AuthAccountManager.getState();
    const currentId = typeof state.current_user_id === 'string' ? state.current_user_id : undefined;
    if (currentId == null) return null;
    return state.accounts[String(currentId)] ?? null;
  }

  /**
   * 仅保存/更新一个账号（不改当前活跃账号、不写会话快捷键）。
   * 场景：后台 token 刷新完成时用户已切到其它账号——只落库，避免把旧账号
   * 重新激活造成串数据。
   */
  static async upsertOnly(auth: AuthData): Promise<void> {
    const state = await AuthAccountManager.getState();
    const newAccounts: Record<string, AuthData> = {};
    const accountKeys = Object.keys(state.accounts);
    for (let i = 0; i < accountKeys.length; i++) {
      const key = accountKeys[i];
      newAccounts[key] = state.accounts[key];
    }
    newAccounts[String(auth.user_id)] = auth;
    const newState: AuthAccountsState = {
      version: state.version,
      current_user_id: state.current_user_id,
      accounts: newAccounts
    };
    await AuthAccountManager.setState(newState);
  }

  /**
   * 保存/更新一个账号，并将其设为当前活跃账号
   * 同步更新当前会话键（access_token/user_info）
   */
  static async upsertAndActivate(auth: AuthData): Promise<void> {
    const state = await AuthAccountManager.getState();
    const newAccounts: Record<string, AuthData> = {};
    const accountKeys = Object.keys(state.accounts);
    for (let i = 0; i < accountKeys.length; i++) {
      const key = accountKeys[i];
      newAccounts[key] = state.accounts[key];
    }
    newAccounts[String(auth.user_id)] = auth;
    const newState: AuthAccountsState = {
      version: state.version,
      current_user_id: auth.user_id,
      accounts: newAccounts
    };
    await AuthAccountManager.setState(newState);

    // 维持一份“当前账号”的 user_info/access_token 作为会话快捷读取键
    await PreferencesUtil.setObject(StorageKeys.AUTH_DATA, auth);
    await PreferencesUtil.set(StorageKeys.ACCESS_TOKEN, auth.access_token);
  }

  /**
   * 切换到已保存账号（会更新 access_token/user_info）
   * 返回切换后的 auth（不存在则返回 null）
   */
  static async switchTo(userId: string): Promise<AuthData | null> {
    const state = await AuthAccountManager.getState();
    const next = state.accounts[String(userId)];
    if (!next) return null;

    const newState: AuthAccountsState = {
      version: state.version,
      current_user_id: userId,
      accounts: state.accounts
    };
    await AuthAccountManager.setState(newState);

    await PreferencesUtil.setObject(StorageKeys.AUTH_DATA, next);
    await PreferencesUtil.set(StorageKeys.ACCESS_TOKEN, next.access_token);
    return next;
  }

  /**
   * 本地锁定当前账号：清除当前 access_token/user_info（不删除 accounts 列表）
   */
  static async lockCurrent(): Promise<void> {
    const state = await AuthAccountManager.getState();
    const newState: AuthAccountsState = {
      version: state.version,
      accounts: state.accounts
    };
    await AuthAccountManager.setState(newState);

    await PreferencesUtil.remove(StorageKeys.AUTH_DATA);
    await PreferencesUtil.remove(StorageKeys.ACCESS_TOKEN);
  }

  /**
   * 删除一个账号（如果删除的是当前账号，会自动 lock）
   */
  static async removeAccount(userId: string): Promise<void> {
    const state = await AuthAccountManager.getState();
    const newAccounts: Record<string, AuthData> = {};
    const accountKeys = Object.keys(state.accounts);
    for (let i = 0; i < accountKeys.length; i++) {
      const key = accountKeys[i];
      if (key !== String(userId)) {
        newAccounts[key] = state.accounts[key];
      }
    }
    const newState: AuthAccountsState = {
      version: state.version,
      current_user_id: state.current_user_id === userId ? undefined : state.current_user_id,
      accounts: newAccounts
    };
    if (state.current_user_id === userId) {
      await PreferencesUtil.remove(StorageKeys.AUTH_DATA);
      await PreferencesUtil.remove(StorageKeys.ACCESS_TOKEN);
    }
    await AuthAccountManager.setState(newState);
  }

  /**
   * 清空所有账号（危险操作）
   */
  static async clearAllAccounts(): Promise<void> {
    await PreferencesUtil.remove(StorageKeys.AUTH_ACCOUNTS);
    await PreferencesUtil.remove(StorageKeys.AUTH_DATA);
    await PreferencesUtil.remove(StorageKeys.ACCESS_TOKEN);
  }
}

