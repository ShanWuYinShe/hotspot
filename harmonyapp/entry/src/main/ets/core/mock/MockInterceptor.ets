/**
 * DEV mock interceptor — plugs into HttpUtil.request() to return canned
 * ApiResponse data when enabled. State persists via Preferences.
 *
 * Enable via AdvancedSettings dev panel or:
 *   await MockInterceptor.setEnabled(true);
 */

import { PreferencesUtil } from '../storage/PreferencesUtil';
import { getCannedResponse, setLoggedIn, setScenarioStatic, isLoggedIn, getScenario } from './MockData';
import { ApiResponse } from '../types/api';
import { hilog } from '@kit.PerformanceAnalysisKit';

const DOMAIN_NUMBER: number = 0xFF37;
const TAG: string = 'MockInterceptor';
const KEY_ENABLED = 'mock_enabled';
const KEY_SCENARIO = 'mock_scenario';

export class MockInterceptor {
  private static _enabled: boolean = false;
  private static _initialized: boolean = false;

  /** Call once after PreferencesUtil.init. */
  static async init(): Promise<void> {
    if (MockInterceptor._initialized) return;
    try {
      MockInterceptor._enabled = await PreferencesUtil.get<boolean>(KEY_ENABLED, false);
      const s = await PreferencesUtil.get<string>(KEY_SCENARIO, 'ok');
      setScenarioStatic(s);
    } catch {}
    MockInterceptor._initialized = true;
    hilog.info(DOMAIN_NUMBER, TAG, `Mock interceptor init (enabled=${MockInterceptor._enabled})`);
  }

  static isEnabled(): boolean { return MockInterceptor._enabled; }

  static async setEnabled(v: boolean): Promise<void> {
    MockInterceptor._enabled = v;
    try { await PreferencesUtil.set(KEY_ENABLED, v); } catch {}
  }

  static async setScenario(s: string): Promise<void> {
    setScenarioStatic(s);
    try { await PreferencesUtil.set(KEY_SCENARIO, s); } catch {}
  }

  static getCurrentScenario(): string { return getScenario(); }

  /**
   * Called at the top of HttpUtil.request(). Returns null if mock is disabled
   * or the endpoint has no canned response, so HttpUtil falls through to real
   * network timing.
   */
  static match(url: string): ApiResponse<Object> | null {
    if (!MockInterceptor._enabled) return null;
    return getCannedResponse(url, isLoggedIn(), getScenario());
  }
}
