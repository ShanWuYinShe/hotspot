/**
 * 系统级Toast工具类
 * 使用推荐的CustomPopup组件替代废弃的promptAction.showToast
 */

import { CustomToastManager, ToastType } from './CustomToast';
import { hilog } from '@kit.PerformanceAnalysisKit';

const DOMAIN_NUMBER: number = 0xFF13;
const TAG: string = 'SystemToast';

export class SystemToast {
  private static toastManager = CustomToastManager.getInstance();

  /**
   * 显示系统级Toast - 使用推荐的CustomPopup组件
   */
  static show(message: string, duration: number = 2000): void {
    try {
      SystemToast.toastManager.show(message, ToastType.INFO, duration);
      hilog.info(DOMAIN_NUMBER, TAG, `显示Toast: ${message}`);
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `Toast显示失败: ${JSON.stringify(error)}`);
    }
  }

  /**
   * 显示成功Toast
   */
  static showSuccess(message: string): void {
    SystemToast.toastManager.showSuccess(message);
  }

  /**
   * 显示错误Toast
   */
  static showError(message: string): void {
    SystemToast.toastManager.showError(message);
  }

  /**
   * 显示警告Toast
   */
  static showWarning(message: string): void {
    SystemToast.toastManager.showWarning(message);
  }

  /**
   * 显示信息Toast
   */
  static showInfo(message: string): void {
    SystemToast.toastManager.showInfo(message);
  }
}
