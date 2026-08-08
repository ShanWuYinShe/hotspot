/**
 * 系统级Toast工具类
 * 使用官方推荐的UIContext.PromptAction#showToast替代废弃的promptAction.showToast
 * 需要通过参数传递UIContext
 */

import { UIContext } from '@kit.ArkUI';
import { hilog } from '@kit.PerformanceAnalysisKit';

const DOMAIN_NUMBER: number = 0xFF13;
const TAG: string = 'SystemToast';

export class SystemToast {
  /**
   * 显示基础Toast
   */
  static show(message: string, uiContext: UIContext, duration: number = 2000): void {
    try {
      uiContext.getPromptAction().showToast({
        message: message,
        duration: duration,
        bottom: 100
      });
      hilog.info(DOMAIN_NUMBER, TAG, `显示Toast: ${message}`);
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `Toast显示失败: ${JSON.stringify(error)}`);
    }
  }

  /**
   * 显示成功Toast
   */
  static showSuccess(message: string, uiContext: UIContext): void {
    SystemToast.show(`✅ ${message}`, uiContext);
  }

  /**
   * 显示错误Toast
   */
  static showError(message: string, uiContext: UIContext): void {
    SystemToast.show(`❌ ${message}`, uiContext);
  }

  /**
   * 显示警告Toast
   */
  static showWarning(message: string, uiContext: UIContext): void {
    SystemToast.show(`⚠️ ${message}`, uiContext);
  }

  /**
   * 显示信息Toast
   */
  static showInfo(message: string, uiContext: UIContext): void {
    SystemToast.show(`ℹ️ ${message}`, uiContext);
  }

  /**
   * 显示长时间Toast
   */
  static showLong(message: string, uiContext: UIContext): void {
    SystemToast.show(message, uiContext, 3500);
  }

  /**
   * 显示自定义位置Toast
   */
  static showAt(message: string, uiContext: UIContext, bottom: number = 100): void {
    try {
      uiContext.getPromptAction().showToast({
        message: message,
        duration: 2000, // Toast 显示时长（行为常量，非动画时长）
        bottom: bottom
      });
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `显示自定义Toast失败: ${JSON.stringify(error)}`);
    }
  }
}
