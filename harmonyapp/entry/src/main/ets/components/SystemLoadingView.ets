/**
 * 系统级加载组件
 * 使用 ArkUI 声明式动画替代 setInterval 轮询，降低 CPU 开销
 */

@Component
export struct SystemLoadingView {
  @Prop message: string = '加载中...';
  @Prop showProgress: boolean = true;
  @Prop showSystemInfo: boolean = false;
  @Prop progressType: 'indeterminate' | 'determinate' = 'indeterminate';
  @Prop maxProgress: number = 100;
  @Prop externalProgress: number = 0;
  @State private rotationAngle: number = 0;
  @State private systemTime: string = '';
  private timeUpdateId: number = -1;
  private cachedDate: Date = new Date();

  aboutToAppear() {
    this.rotationAngle = 360;
    if (this.showSystemInfo) {
      this.updateSystemTime();
      this.timeUpdateId = setInterval(() => {
        this.updateSystemTime();
      }, 1000);
    }
  }

  aboutToDisappear() {
    if (this.timeUpdateId !== -1) {
      clearInterval(this.timeUpdateId);
    }
  }

  build() {
    Column({ space: 24 }) {
      Stack() {
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

        if (this.showProgress && this.progressType === 'determinate') {
          Progress({
            value: this.externalProgress,
            total: this.maxProgress,
            type: ProgressType.Ring
          })
            .width(96)
            .height(96)
            .color($r('sys.color.ohos_id_color_emphasize'))
            .style({ strokeWidth: 3 })
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
          Text(`${Math.round(this.externalProgress)}%`)
            .fontSize(14)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
            .textAlign(TextAlign.Center)
        }
      }
    }
    .width('100%')
    .height('100%')
    .justifyContent(FlexAlign.Center)
    .backgroundColor($r('app.color.bg_secondary'))
    .padding(32)
  }

  private updateSystemTime(): void {
    try {
      this.cachedDate.setTime(Date.now());
      this.systemTime = this.cachedDate.toLocaleTimeString('zh-CN');
    } catch (_) {
      // 忽略时间格式化失败
    }
  }
}
