/**
 * 智能位置卡片组件
 * 集成天气、海拔、导航等多种HarmonyOS系统服务
 */

import { hilog } from '@kit.PerformanceAnalysisKit';
import { vibrator } from '@kit.SensorServiceKit';
import { DesignConstants } from '../constants/DesignConstants';
import { AppIcon } from './SystemIcon';
import { LocationUtil } from '../core/location/LocationUtil';

const DOMAIN_NUMBER: number = 0xFF11;
const TAG: string = 'IntelligentLocationCard';

@Component
export struct IntelligentLocationCard {
  @Prop latitude: number = 0;
  @Prop longitude: number = 0;
  @Prop locationName: string = '未知位置';
  @Prop distance: number = -1;
  @Prop altitude: number = 0; // 新增海拔属性
  @Prop showElevation: boolean = true;
  onLocationTap?: () => void;
  onNavigationTap?: () => void;
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
        SymbolSpan($r(AppIcon.LOCATION))
          .fontSize(DesignConstants.FONT.SIZE.XXL)
          .fontColor([$r('sys.color.ohos_id_color_emphasize')])
      }

      Column({ space: 6 }) {
        // 地址信息
        Text(this.locationName || '未知位置')
          .fontSize(DesignConstants.FONT.SIZE.LG)
          .fontColor($r('sys.color.ohos_id_color_text_primary'))
          .fontWeight(FontWeight.Medium)
          .maxLines(2)
          .textOverflow({ overflow: TextOverflow.Ellipsis })

        Row({ space: 12 }) {
          // 距离信息
          if (this.distance >= 0) {
              Row({ space: DesignConstants.SPACING.XS }) {
                Text() {
                  SymbolSpan($r(AppIcon.SEND))
                    .fontSize(DesignConstants.FONT.SIZE.SM)
                    .fontColor([$r('sys.color.ohos_id_color_text_secondary')])
                }
                Text(LocationUtil.formatDistance(this.distance))
                  .fontSize(DesignConstants.FONT.SIZE.SM)
                  .fontColor($r('sys.color.ohos_id_color_text_secondary'))
              }
          }

          // 海拔信息
          if (this.showElevation && this.elevationInfo) {
            Row({ space: DesignConstants.SPACING.XS }) {
              Text('⛰️')
                .fontSize(DesignConstants.FONT.SIZE.SM)
              
              Text(this.elevationInfo)
                .fontSize(DesignConstants.FONT.SIZE.SM)
                .fontColor($r('sys.color.ohos_id_color_text_secondary'))
            }
          }
        }

      }
      .alignItems(HorizontalAlign.Start)
      .layoutWeight(1)

      // 操作按钮组
      Column({ space: DesignConstants.SPACING.SM }) {
        // 查看详情按钮
        Button({ type: ButtonType.Normal, stateEffect: true }) {
          Text() {
            SymbolSpan($r(AppIcon.INFO))
              .fontSize(DesignConstants.FONT.SIZE.LG)
              .fontColor([Color.White])
          }
        }
        .width(32)
        .height(32)
        .backgroundColor($r('sys.color.ohos_id_color_emphasize'))
        .borderRadius(DesignConstants.RADIUS.XL)
        .onClick(() => {
          this.triggerHapticFeedback();
          if (this.onLocationTap) {
            this.onLocationTap();
          }
        })

        // 导航按钮
        Button({ type: ButtonType.Normal, stateEffect: true }) {
          Text() {
            SymbolSpan($r(AppIcon.UP))
              .fontSize(DesignConstants.FONT.SIZE.LG)
              .fontColor([$r('sys.color.ohos_id_color_emphasize')])
          }
        }
        .width(32)
        .height(32)
        .backgroundColor($r('app.color.bg_secondary'))
        .border({
          width: 1,
          color: $r('sys.color.ohos_id_color_emphasize')
        })
        .borderRadius(DesignConstants.RADIUS.XL)
        .onClick(() => {
          this.triggerHapticFeedback();
          if (this.onNavigationTap) {
            this.onNavigationTap();
          }
        })
      }
    }
    .width('100%')
    .padding(16)
    .backgroundColor($r('app.color.bg_secondary'))
    .borderRadius(DesignConstants.RADIUS.XL)
    .border({ width: 1, color: $r('sys.color.ohos_id_color_list_separator') })
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
          if (this.onLocationTap) {
            this.onLocationTap();
          }
        })
    )
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
