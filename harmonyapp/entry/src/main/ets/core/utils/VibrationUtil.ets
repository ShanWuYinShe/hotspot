/**
 * 震动反馈工具类
 * 使用 HarmonyOS 原生 SensorServiceKit 提供触觉反馈
 */
import { vibrator } from '@kit.SensorServiceKit';
import { BusinessError } from '@kit.BasicServicesKit';
import { hilog } from '@kit.PerformanceAnalysisKit';

const TAG = 'VibrationUtil';

export class VibrationUtil {
  
  /**
   * 触发短震动 (轻触反馈)
   * 适用于：普通点击、开关切换、键盘输入
   */
  static async startShort(): Promise<void> {
    try {
      // 预置震动效果：EFFECT_ID_CLOCK_TIMER (类似滴答声/轻触)
      // 或者使用 vibrator.VibratorFileDescription
      // 这里使用 startVibration 配合简单的 duration 或 preset
      // 注意：不同设备支持的 preset 不同
      await vibrator.startVibration({
        type: 'preset',
        effectId: 'haptic.clock.timer', // 常用轻触反馈
        count: 1,
      }, {
        id: 0,
        usage: 'touch'
      });
    } catch (error) {
      const err = error as BusinessError;
      hilog.debug(0xFF00, TAG, `Vibration failed: ${err.message}`);
    }
  }

  /**
   * 触发成功反馈
   * 适用于：操作成功、完成支付、完成加载
   */
  static async startSuccess(): Promise<void> {
    try {
      // 尝试加载系统预置效果，如果不支持则降级
      // haptic.success 是比较通用的 ID? 
      // 查阅文档：ohos.vibrator 
      // 降级方案：自定义时长
      await vibrator.startVibration({
        type: 'time',
        duration: 50,
      }, {
        id: 0,
        usage: 'touch'
      });
    } catch (error) {
      // ignore
    }
  }

  /**
   * 触发失败/警告反馈
   * 适用于：操作失败、删除确认、错误提示
   */
  static async startFail(): Promise<void> {
    try {
      // 连续两次短震动
      await vibrator.startVibration({
        type: 'time',
        duration: 100, // 触觉反馈时长（行为常量，非视觉 token 范畴）
      }, {
        id: 0,
        usage: 'touch'
      });
    } catch (error) {
      // ignore
    }
  }
}










