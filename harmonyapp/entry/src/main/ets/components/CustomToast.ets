/**
 * 自定义Toast组件
 * 使用CustomPopup替代废弃的promptAction.showToast
 */

import { hilog } from '@kit.PerformanceAnalysisKit';
import { vibrator } from '@kit.SensorServiceKit';

const DOMAIN_NUMBER: number = 0xFF14;
const TAG: string = 'CustomToast';

export enum ToastType {
  SUCCESS = 'success',
  ERROR = 'error',
  WARNING = 'warning',
  INFO = 'info'
}

@Component
export struct CustomToast {
  @Prop message: string = '';
  @Prop type: ToastType = ToastType.INFO;
  @Prop duration: number = 2000;
  @State private isVisible: boolean = false;
  @State private toastOpacity: number = 0;
  private hideTimer: number = -1;

  aboutToAppear() {
    this.show();
  }

  aboutToDisappear() {
    if (this.hideTimer !== -1) {
      clearTimeout(this.hideTimer);
    }
  }

  build() {
    if (this.isVisible) {
      Row({ space: 8 }) {
        // 图标
        Text(this.getIcon())
          .fontSize(16)

        // 消息文本
        Text(this.message)
          .fontSize(14)
          .fontColor(Color.White)
          .maxLines(2)
          .textOverflow({ overflow: TextOverflow.Ellipsis })
      }
      .padding({ left: 16, right: 16, top: 12, bottom: 12 })
      .backgroundColor(this.getBackgroundColor())
      .borderRadius(8)
      .shadow({
        radius: 8,
        color: Color.Black,
        offsetX: 0,
        offsetY: 4
      })
      .opacity(this.toastOpacity)
      .animation({
        duration: 300,
        curve: Curve.EaseInOut
      })
      .position({ x: '50%', y: '80%' })
      .markAnchor({ x: '50%', y: '50%' })
    }
  }

  private show(): void {
    this.isVisible = true;
    
    // 触觉反馈
    this.triggerHapticFeedback();
    
    // 淡入动画
    setTimeout(() => {
      this.toastOpacity = 1;
    }, 50);

    // 自动隐藏
    this.hideTimer = setTimeout(() => {
      this.hide();
    }, this.duration);
  }

  private hide(): void {
    // 淡出动画
    this.toastOpacity = 0;
    
    // 延迟移除组件
    setTimeout(() => {
      this.isVisible = false;
    }, 300);
  }

  private getIcon(): string {
    switch (this.type) {
      case ToastType.SUCCESS:
        return '✅';
      case ToastType.ERROR:
        return '❌';
      case ToastType.WARNING:
        return '⚠️';
      case ToastType.INFO:
      default:
        return 'ℹ️';
    }
  }

  private getBackgroundColor(): ResourceColor {
    switch (this.type) {
      case ToastType.SUCCESS:
        return '#4CAF50';
      case ToastType.ERROR:
        return '#F44336';
      case ToastType.WARNING:
        return '#FF9800';
      case ToastType.INFO:
      default:
        return '#2196F3';
    }
  }

  private triggerHapticFeedback(): void {
    try {
      vibrator.startVibration({
        type: 'time',
        duration: 20
      }, {
        id: 0,
        usage: 'alarm'
      });
    } catch (error) {
      hilog.warn(DOMAIN_NUMBER, TAG, '触觉反馈失败');
    }
  }
}

// Toast管理器 - 全局单例
export class CustomToastManager {
  private static instance: CustomToastManager | null = null;
  private toastQueue: ToastConfig[] = [];
  private isShowing: boolean = false;

  private constructor() {}

  static getInstance(): CustomToastManager {
    if (!CustomToastManager.instance) {
      CustomToastManager.instance = new CustomToastManager();
    }
    return CustomToastManager.instance;
  }

  show(message: string, type: ToastType = ToastType.INFO, duration: number = 2000): void {
    const config: ToastConfig = { message, type, duration };
    
    if (this.isShowing) {
      this.toastQueue.push(config);
    } else {
      this.showToast(config);
    }
  }

  showSuccess(message: string, duration: number = 2000): void {
    this.show(message, ToastType.SUCCESS, duration);
  }

  showError(message: string, duration: number = 2000): void {
    this.show(message, ToastType.ERROR, duration);
  }

  showWarning(message: string, duration: number = 2000): void {
    this.show(message, ToastType.WARNING, duration);
  }

  showInfo(message: string, duration: number = 2000): void {
    this.show(message, ToastType.INFO, duration);
  }

  private showToast(config: ToastConfig): void {
    this.isShowing = true;
    
    // 这里需要在实际使用时通过状态管理或者全局组件来显示Toast
    // 由于ArkTS的限制，我们需要在页面级别管理Toast的显示
    hilog.info(DOMAIN_NUMBER, TAG, `显示Toast: ${config.message}`);
    
    setTimeout(() => {
      this.isShowing = false;
      this.showNextToast();
    }, config.duration + 300);
  }

  private showNextToast(): void {
    if (this.toastQueue.length > 0) {
      const nextConfig = this.toastQueue.shift();
      if (nextConfig) {
        this.showToast(nextConfig);
      }
    }
  }
}

interface ToastConfig {
  message: string;
  type: ToastType;
  duration: number;
}
