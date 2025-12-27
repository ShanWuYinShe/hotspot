/**
 * 智能位置卡片组件
 * 集成天气、海拔、导航等多种HarmonyOS系统服务
 */

import { hilog } from '@kit.PerformanceAnalysisKit';
import { vibrator } from '@kit.SensorServiceKit';

const DOMAIN_NUMBER: number = 0xFF11;
const TAG: string = 'IntelligentLocationCard';

@Component
export struct IntelligentLocationCard {
  @Prop latitude: number = 0;
  @Prop longitude: number = 0;
  @Prop locationName: string = '南天门';
  @Prop distance: number = -1;
  @Prop altitude: number = 0; // 新增海拔属性
  @Prop showElevation: boolean = true;
  @Prop onLocationTap: () => void = () => {};
  @Prop onNavigationTap: () => void = () => {};
  @State private cardElevation: number = 1;
  @State private elevationInfo: string = '';

  aboutToAppear() {
    if (this.showElevation) {
      this.loadElevationInfo();
    }
  }

  build() {
    Row({ space: 12 }) {
      // 位置图标 - 使用SymbolSpan
      Text() {
        SymbolSpan($r('sys.symbol.local_fill'))
          .fontSize(24)
          .fontColor([$r('sys.color.ohos_id_color_emphasize')])
      }

      Column({ space: 6 }) {
        // 地址信息
        Text(this.locationName || '南天门')
          .fontSize(16)
          .fontColor($r('sys.color.ohos_id_color_text_primary'))
          .fontWeight(FontWeight.Medium)
          .maxLines(2)
          .textOverflow({ overflow: TextOverflow.Ellipsis })

        Row({ space: 12 }) {
          // 距离信息
          if (this.distance >= 0) {
              Row({ space: 4 }) {
                Text() {
                  SymbolSpan($r('sys.symbol.paperplane_right_fill'))
                    .fontSize(12)
                    .fontColor([$r('sys.color.ohos_id_color_text_secondary')])
                }
                Text(this.formatDistance(this.distance))
                  .fontSize(12)
                  .fontColor($r('sys.color.ohos_id_color_text_secondary'))
              }
          }

          // 海拔信息
          if (this.showElevation && this.elevationInfo) {
            Row({ space: 4 }) {
              Text('⛰️')
                .fontSize(12)
              
              Text(this.elevationInfo)
                .fontSize(12)
                .fontColor($r('sys.color.ohos_id_color_text_secondary'))
            }
          }
        }

        // 坐标信息
        Text(`${this.latitude.toFixed(6)}, ${this.longitude.toFixed(6)}`)
          .fontSize(11)
          .fontColor($r('sys.color.ohos_id_color_text_tertiary'))
          .fontFamily('monospace')
      }
      .alignItems(HorizontalAlign.Start)
      .layoutWeight(1)

      // 操作按钮组
      Column({ space: 8 }) {
        // 查看详情按钮
        Button({ type: ButtonType.Circle, stateEffect: true }) {
          Text() {
            SymbolSpan($r('sys.symbol.eye'))
              .fontSize(16)
              .fontColor([Color.White])
          }
        }
        .width(32)
        .height(32)
        .backgroundColor($r('sys.color.ohos_id_color_emphasize'))
        .onClick(() => {
          this.triggerHapticFeedback();
          this.onLocationTap();
        })

        // 导航按钮
        Button({ type: ButtonType.Circle, stateEffect: true }) {
          Text() {
            SymbolSpan($r('sys.symbol.arrow_up'))
              .fontSize(16)
              .fontColor([$r('sys.color.ohos_id_color_emphasize')])
          }
        }
        .width(32)
        .height(32)
        .backgroundColor($r('sys.color.ohos_id_color_background'))
        .border({
          width: 1,
          color: $r('sys.color.ohos_id_color_emphasize')
        })
        .onClick(() => {
          this.triggerHapticFeedback();
          this.onNavigationTap();
        })
      }
    }
    .width('100%')
    .padding(16)
    .backgroundColor($r('sys.color.ohos_id_color_background'))
    .borderRadius(16)
    .shadow({
      radius: this.cardElevation * 6,
      color: $r('sys.color.ohos_id_color_tertiary'),
      offsetX: 0,
      offsetY: this.cardElevation * 3
    })
    .onHover((isHover: boolean) => {
      // 使用最新的动画API
      this.cardElevation = isHover ? 4 : 1;
    })
    .animation({
      duration: 250,
      curve: Curve.EaseInOut
    })
    .gesture(
      TapGesture()
        .onAction(() => {
          this.triggerHapticFeedback();
          this.onLocationTap();
        })
    )
  }

  private formatDistance(distance: number): string {
    if (distance < 1000) {
      return `${Math.round(distance)}m`;
    } else if (distance < 10000) {
      return `${(distance / 1000).toFixed(1)}km`;
    } else {
      return `${Math.round(distance / 1000)}km`;
    }
  }

  private async loadElevationInfo(): Promise<void> {
    try {
      if (this.altitude && this.altitude !== 0) {
         this.elevationInfo = `海拔 ${Math.round(this.altitude)}m`;
      } else {
         this.elevationInfo = '';
      }
    } catch (error) {
      hilog.warn(DOMAIN_NUMBER, TAG, '加载海拔信息失败');
    }
  }

  private triggerHapticFeedback(): void {
    try {
      vibrator.startVibration({
        type: 'time',
        duration: 15
      }, {
        id: 0,
        usage: 'alarm'
      });
    } catch (error) {
      hilog.warn(DOMAIN_NUMBER, TAG, '触觉反馈失败');
    }
  }
}
