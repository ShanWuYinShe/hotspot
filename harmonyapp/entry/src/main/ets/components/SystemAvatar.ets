/**
 * 系统级用户头像组件
 * 集成触觉反馈、系统缓存、在线状态等HarmonyOS原生能力
 */

import { curves } from '@kit.ArkUI';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { vibrator } from '@kit.SensorServiceKit';

const DOMAIN_NUMBER: number = 0xFF10;
const TAG: string = 'SystemAvatar';

@Component
export struct SystemAvatar {
  @Prop avatarUrl: string = '';
  @Prop avatarSize: number = 40;
  @Prop nickname: string = '';
  @Prop isOnline: boolean = false;
  @Prop showBorder: boolean = true;
  @Prop avatarBorderColor: ResourceColor = $r('sys.color.ohos_id_color_emphasize');
  @Prop enableHapticFeedback: boolean = true;
  @Prop onTap: () => void = () => {};
  @State private isPressed: boolean = false;
  @State private scaleValue: number = 1;
  @State private shadowRadius: number = 4;

  build() {
    Stack({ alignContent: Alignment.BottomEnd }) {
      // 主头像 - 使用系统图片缓存和优化
      if (this.avatarUrl) {
        Image(this.avatarUrl)
          .width(this.avatarSize)
          .height(this.avatarSize)
          .borderRadius(this.avatarSize / 2)
          .objectFit(ImageFit.Cover)
          .border(this.showBorder ? {
            width: 2,
            color: this.avatarBorderColor
          } : undefined)
          .scale({ x: this.scaleValue, y: this.scaleValue })
          .shadow({
            radius: this.shadowRadius,
            color: $r('sys.color.ohos_id_color_tertiary'),
            offsetX: 0,
            offsetY: 2
          })
          .animation({
            duration: 200,
            curve: curves.springMotion(0.6, 0.8)
          })
          // 使用系统级图片加载优化
          .syncLoad(false)
          .autoResize(true)
      } else {
        Text(this.nickname.charAt(0).toUpperCase())
          .width(this.avatarSize)
          .height(this.avatarSize)
          .fontSize(this.avatarSize * 0.4)
          .fontColor(Color.White)
          .textAlign(TextAlign.Center)
          .backgroundColor($r('sys.color.ohos_id_color_emphasize'))
          .borderRadius(this.avatarSize / 2)
          .border(this.showBorder ? {
            width: 2,
            color: this.avatarBorderColor
          } : undefined)
          .scale({ x: this.scaleValue, y: this.scaleValue })
          .shadow({
            radius: this.shadowRadius,
            color: $r('sys.color.ohos_id_color_tertiary'),
            offsetX: 0,
            offsetY: 2
          })
          .animation({
            duration: 200,
            curve: curves.springMotion(0.6, 0.8)
          })
      }

      // 在线状态指示器 - 使用系统状态颜色
      if (this.isOnline !== undefined) {
        Circle({ width: this.avatarSize * 0.25, height: this.avatarSize * 0.25 })
          .fill(this.isOnline ? $r('sys.color.ohos_id_color_emphasize') : $r('sys.color.ohos_id_color_tertiary'))
          .stroke($r('sys.color.ohos_id_color_background'))
          .strokeWidth(2)
          .offset({ x: -2, y: -2 })
      }
    }
    .onTouch((event: TouchEvent) => {
      if (event.type === TouchType.Down) {
        this.isPressed = true;
        this.scaleValue = 0.95;
        this.shadowRadius = 8;
        
        // 系统触觉反馈
        if (this.enableHapticFeedback) {
          this.triggerHapticFeedback();
        }
      } else if (event.type === TouchType.Up || event.type === TouchType.Cancel) {
        this.isPressed = false;
        this.scaleValue = 1;
        this.shadowRadius = 4;
        
        if (event.type === TouchType.Up) {
          this.onTap();
        }
      }
    })
  }

  private triggerHapticFeedback(): void {
    try {
      vibrator.startVibration({
        type: 'time',
        duration: 10
      }, {
        id: 0,
        usage: 'alarm'
      });
    } catch (error) {
      hilog.warn(DOMAIN_NUMBER, TAG, '触觉反馈失败');
    }
  }
}