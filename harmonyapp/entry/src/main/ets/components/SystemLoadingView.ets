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
  @State private rotationAngle: number = 0;
  @State private systemTime: string = '';
  @State private loadingProgress: number = 0;
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
      // 系统级加载动画
      Stack() {
        // 外圈旋转动画
        Circle({ width: 80, height: 80 })
          .stroke($r('sys.color.ohos_id_color_emphasize'))
          .strokeWidth(3)
          .strokeDashArray([15, 8])
          .rotate({ angle: this.rotationAngle })
          .animation({
            duration: 2000,
            iterations: -1,
            curve: Curve.Linear
          })

        // 内圈进度
        if (this.showProgress) {
          Progress({
            value: this.loadingProgress,
            total: 100,
            type: ProgressType.Ring
          })
          .width(60)
          .height(60)
          .color($r('sys.color.ohos_id_color_emphasize'))
          .style({
            strokeWidth: 4,
            scaleCount: 0,
            enableScanEffect: true
          })
        }

        // 中心图标
        Text() {
          SymbolSpan($r('sys.symbol.arrow_clockwise'))
            .fontSize(24)
            .fontColor([$r('sys.color.ohos_id_color_emphasize')])
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

        if (this.showProgress) {
          Text(`${Math.round(this.loadingProgress)}%`)
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
      this.rotationAngle += 10;
      if (this.rotationAngle >= 360) {
        this.rotationAngle = 0;
      }
      
      // 模拟进度更新
      if (this.showProgress) {
        this.loadingProgress = (this.loadingProgress + 2) % 100;
      }
    }, 50);
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
