/**
 * 系统级加载组件
 * 集成系统时间、进度显示、系统动画等HarmonyOS能力
 */

import { curves } from '@kit.ArkUI';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { systemDateTime } from '@kit.BasicServicesKit';

const DOMAIN_NUMBER: number = 0xFF12;
const TAG: string = 'SystemLoadingView';

@Component
export struct SystemLoadingView {
  @Prop message: string = '加载中...';
  @Prop showProgress: boolean = true;
  @Prop showSystemInfo: boolean = true;
  @Prop progressType: 'indeterminate' | 'determinate' = 'indeterminate';
  @Prop maxProgress: number = 100;
  @Prop externalProgress: number;
  @State private rotationAngle: number = 0;
  @State private systemTime: string = '';
  @State private loadingProgress: number = 0;
  @State private progressDirection: number = 1; // 1 for forward, -1 for backward
  private animationId: number = -1;
  private timeUpdateId: number = -1;

  aboutToAppear() {
    this.startAnimation();
    if (this.showSystemInfo) {
      this.startTimeUpdate();
    }
  }

  aboutToDisappear() {
    if (this.animationId !== -1) {
      clearInterval(this.animationId);
    }
    if (this.timeUpdateId !== -1) {
      clearInterval(this.timeUpdateId);
    }
  }

  build() {
    Column({ space: 24 }) {
      // 龙形 logo 旋转动画 —— 游龙戏珠效果
      Stack() {
        // logo 旋转
        Image($r('app.media.logo'))
          .width(80)
          .height(80)
          .objectFit(ImageFit.Contain)
          .rotate({ angle: this.rotationAngle })
          .animation({
            duration: 2000,
            iterations: -1,
            curve: Curve.Linear
          })

        // 确定进度模式：在 logo 下方显示进度环
        if (this.showProgress && this.progressType === 'determinate') {
          Progress({
            value: this.externalProgress !== undefined ? this.externalProgress : this.loadingProgress,
            total: this.maxProgress,
            type: ProgressType.Ring
          })
            .width(96)
            .height(96)
            .color($r('sys.color.ohos_id_color_emphasize'))
            .style({
              strokeWidth: 3,
              scaleCount: 0,
              enableScanEffect: false
            })
        }
      }

      Column({ space: 8 }) {
        Text(this.message)
          .fontSize(16)
          .fontColor($r('sys.color.ohos_id_color_text_primary'))
          .textAlign(TextAlign.Center)
          .fontWeight(FontWeight.Medium)

        if (this.showSystemInfo && this.systemTime) {
          Text(this.systemTime)
            .fontSize(12)
            .fontColor($r('sys.color.ohos_id_color_text_tertiary'))
            .textAlign(TextAlign.Center)
        }

        if (this.showProgress && this.progressType === 'determinate') {
          Text(`${Math.round(this.externalProgress !== undefined ? this.externalProgress : this.loadingProgress)}%`)
            .fontSize(14)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
            .textAlign(TextAlign.Center)
        }
      }
    }
    .width('100%')
    .height('100%')
    .justifyContent(FlexAlign.Center)
    .backgroundColor($r('sys.color.ohos_id_color_background'))
    .padding(32)
  }

  private startAnimation(): void {
    this.animationId = setInterval(() => {
      // 旋转动画
      this.rotationAngle += 10;
      if (this.rotationAngle >= 360) {
        this.rotationAngle = 0;
      }
      
      // 进度更新逻辑
      if (this.showProgress) {
        if (this.progressType === 'indeterminate') {
          // 不确定进度 - 使用呼吸效果（0-100来回）
          this.loadingProgress += this.progressDirection * 3;
          if (this.loadingProgress >= 100) {
            this.loadingProgress = 100;
            this.progressDirection = -1;
          } else if (this.loadingProgress <= 0) {
            this.loadingProgress = 0;
            this.progressDirection = 1;
          }
        }
        // 确定进度模式下，进度由外部控制，这里不自动更新
      }
    }, 50);
  }

  /**
   * 设置确定进度值（仅在 determinate 模式下有效）
   */
  setProgress(progress: number): void {
    if (this.progressType === 'determinate') {
      this.loadingProgress = Math.min(Math.max(progress, 0), this.maxProgress);
    }
  }

  /**
   * 重置进度
   */
  resetProgress(): void {
    this.loadingProgress = 0;
    this.progressDirection = 1;
  }

  private startTimeUpdate(): void {
    this.updateSystemTime();
    this.timeUpdateId = setInterval(() => {
      this.updateSystemTime();
    }, 1000);
  }

  private async updateSystemTime(): Promise<void> {
    try {
      // 使用推荐的Date.now()或new Date()替代util.Time.getCurrentTime()
      const now = Date.now();
      const date = new Date(now);
      this.systemTime = date.toLocaleTimeString('zh-CN');
    } catch (error) {
      hilog.warn(DOMAIN_NUMBER, TAG, '获取系统时间失败');
    }
  }
}
