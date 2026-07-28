/**
 * 持久化 device_id 工具（spec: DeviceId = UUID v4）。
 *
 * spec 要求每个 auth 请求体都携带 device_id，且该值需跨会话稳定——它驱动
 * 「单设备单会话」踢人语义与 refresh_token 的设备校验。因此不能像幂等键
 * 那样每次启动随机生成，必须持久化。
 *
 * 使用方式：
 *   - 应用启动时（Preferences 初始化后）调用 ensureInitialized() 预加载
 *   - 之后用同步的 getDeviceId() 取值；未初始化时退化为进程内稳定的临时 UUID
 *     （不会跨进程持久，但保证同一次运行内所有请求一致）
 */

import { util } from '@kit.ArkTS';
import { PreferencesUtil } from '../storage/PreferencesUtil';
import { StorageKeys } from '../../constants/AppConstants';
import { hilog } from '@kit.PerformanceAnalysisKit';

const DOMAIN_NUMBER: number = 0xFF37;
const TAG: string = 'DeviceIdUtil';

export class DeviceIdUtil {
  /** 内存缓存：ensureInitialized 成功后与持久化值一致。 */
  private static cached: string | null = null;

  /**
   * 应用启动时调用（必须在 PreferencesUtil.init 之后）。读取已持久化的
   * device_id，若不存在则生成 UUID v4 并落盘。
   */
  static async ensureInitialized(): Promise<string> {
    if (DeviceIdUtil.cached) return DeviceIdUtil.cached;
    try {
      let id = await PreferencesUtil.get<string>(StorageKeys.DEVICE_ID, '');
      if (!id) {
        id = util.generateRandomUUID(false);
        await PreferencesUtil.set(StorageKeys.DEVICE_ID, id);
        hilog.info(DOMAIN_NUMBER, TAG, '已生成并持久化 device_id');
      }
      DeviceIdUtil.cached = id;
      return id;
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `初始化 device_id 失败: ${error}`);
      return DeviceIdUtil.getDeviceId();
    }
  }

  /**
   * 同步获取 device_id。若尚未 ensureInitialized，返回进程内稳定的临时值
   * （保证本次运行所有请求一致），避免阻塞请求构造。
   */
  static getDeviceId(): string {
    if (DeviceIdUtil.cached) return DeviceIdUtil.cached;
    // 兜底：进程内稳定，但跨重启会变（仅当未正确初始化时发生）。
    const fallback = util.generateRandomUUID(false);
    DeviceIdUtil.cached = fallback;
    return fallback;
  }
}
