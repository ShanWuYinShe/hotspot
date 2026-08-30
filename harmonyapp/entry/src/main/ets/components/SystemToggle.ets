import { DesignConstants } from '../constants/DesignConstants';
import { AppThemeManager } from '../core/theme/AppThemeManager';
import { KEY_RESOLVED_THEME_ID } from '../core/theme/AppThemeTypes';

/**
 * SystemToggle — iOS 风格开关（对齐前端 forms/Toggle：50×28、thumb 22）。
 * 开启 = accent 实底；关闭 = inputBackground + hairline 边。
 * 状态由父组件持有（受控），onToggle 回调翻转；切换带动 thumb 位移动画。
 */
@Component
export struct SystemToggle {
  @Prop isOn: boolean = false;
  @Prop disabled: boolean = false;
  onToggle?: (next: boolean) => void = () => {
  };
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'vibrant_light';

  private static readonly TRACK_WIDTH: number = 50;
  private static readonly TRACK_HEIGHT: number = 28;
  private static readonly THUMB_SIZE: number = 22;
  /** thumb 距边 3vp（对齐前端 top:3px / left 2-23px 的近似收敛） */
  private static readonly THUMB_MARGIN: number = 3;

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  private getThumbOffset(): number {
    return this.isOn ?
      SystemToggle.TRACK_WIDTH - SystemToggle.THUMB_SIZE - SystemToggle.THUMB_MARGIN :
      SystemToggle.THUMB_MARGIN;
  }

  build() {
    Stack({ alignContent: Alignment.Start }) {
      // thumb（白圆 + hairline 边保证深色轨道上的轮廓）
      Column()
        .width(SystemToggle.THUMB_SIZE)
        .height(SystemToggle.THUMB_SIZE)
        .borderRadius(DesignConstants.RADIUS.FULL)
        .backgroundColor(DesignConstants.COLOR.WHITE)
        .border({
          width: DesignConstants.BORDER_WIDTH.HAIRLINE,
          color: this.getPalette().borderColor
        })
        .shadow(this.getPalette().controlShadow)
        .margin({ left: this.getThumbOffset() })
        .animation({
          duration: DesignConstants.ANIMATION.DURATION_MEDIUM,
          curve: DesignConstants.ANIMATION.CURVE_FAST_OUT_SLOW_IN
        })
    }
    .width(SystemToggle.TRACK_WIDTH)
    .height(SystemToggle.TRACK_HEIGHT)
    .borderRadius(DesignConstants.RADIUS.FULL)
    .backgroundColor(this.isOn ?
      this.getPalette().accentBackground :
      this.getPalette().inputBackgroundColor)
    .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getPalette().borderColor })
    .opacity(this.disabled ? DesignConstants.OPACITY.DISABLED : 1)
    .onClick(() => {
      if (this.disabled || !this.onToggle) {
        return;
      }
      this.onToggle(!this.isOn);
    })
    .accessibilityText(this.isOn ? '已开启' : '已关闭')
    .accessibilityLevel(this.disabled ? 'no' : 'auto')
  }
}
