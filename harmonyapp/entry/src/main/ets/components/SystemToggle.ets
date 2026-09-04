import { DesignConstants } from '../constants/DesignConstants';
import { AppThemeManager } from '../core/theme/AppThemeManager';
import { KEY_RESOLVED_THEME_ID } from '../core/theme/AppThemeTypes';

/**
 * SystemToggle — iOS 风格开关（对齐前端 forms/Toggle：50×28、thumb 22）。
 * 开启 = accent 实底；关闭 = inputBackground + hairline 边。
 * 状态由父组件持有（受控），onToggle 回调翻转；切换带动 thumb 位移动画。
 *
 * 交互契约（四态）：
 * - 按压：thumb 轻缩（按压即「将切换」意图反馈）
 * - hover（仅键鼠外接触发）：on → 轨道整体微提亮；off → 边框 accent 化（两态 hover 反馈不同）
 * - disabled：opacity DISABLED + 不响应
 * - 触控热区：外层透明命中容器扩到 LAYOUT.MIN_HIT_TARGET（视觉 50×28 不变）
 */
@Component
export struct SystemToggle {
  @Prop isOn: boolean = false;
  @Prop disabled: boolean = false;
  onToggle?: (next: boolean) => void = () => {
  };
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'vibrant_light';
  @State private isPressed: boolean = false;
  @State private isHovered: boolean = false;

  private static readonly TRACK_WIDTH: number = 50;
  private static readonly TRACK_HEIGHT: number = 28;
  private static readonly THUMB_SIZE: number = 22;
  /** thumb 距边 3vp（对齐前端 top:3px / left 2-23px 的近似收敛） */
  private static readonly THUMB_MARGIN: number = 3;
  /** 按压 thumb 缩放（M3 switch：按压缩小反馈） */
  private static readonly THUMB_PRESSED_SCALE: number = 0.88;

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  private getThumbOffset(): number {
    return this.isOn ?
    SystemToggle.TRACK_WIDTH - SystemToggle.THUMB_SIZE - SystemToggle.THUMB_MARGIN :
    SystemToggle.THUMB_MARGIN;
  }

  @Builder
  private buildTrack() {
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
        .scale(this.isPressed ?
        {
          x: SystemToggle.THUMB_PRESSED_SCALE,
          y: SystemToggle.THUMB_PRESSED_SCALE
        } : { x: 1, y: 1 })
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
    .border({
      width: DesignConstants.BORDER_WIDTH.HAIRLINE,
      // hover 差异化：off 态边框 accent 化预示可切换；on 态保持轨道自身描边
      color: !this.isOn && this.isHovered && !this.disabled ?
      this.getPalette().accentBackground :
      this.getPalette().borderColor
    })
    .opacity(this.disabled ? DesignConstants.OPACITY.DISABLED : 1)
    .animation({
      duration: DesignConstants.ANIMATION.DURATION_FAST,
      curve: DesignConstants.ANIMATION.CURVE_EASE_OUT
    })
  }

  build() {
    // 透明热区容器（≥ MIN_HIT_TARGET）：交互事件挂这里，视觉轨道居中不变
    Column() {
      this.buildTrack()
    }
    .justifyContent(FlexAlign.Center)
    .alignItems(HorizontalAlign.Center)
    .constraintSize({
      minWidth: DesignConstants.LAYOUT.MIN_HIT_TARGET,
      minHeight: DesignConstants.LAYOUT.MIN_HIT_TARGET
    })
    .opacity(this.disabled ?
    DesignConstants.OPACITY.DISABLED :
    (this.isHovered && this.isOn ? DesignConstants.OPACITY.SUBTLE : 1))
    .onClick(() => {
      if (this.disabled || !this.onToggle) {
        return;
      }
      this.onToggle(!this.isOn);
    })
    .onTouch((event: TouchEvent) => {
      if (this.disabled) {
        this.isPressed = false;
        return;
      }
      if (event.type === TouchType.Down) {
        this.isPressed = true;
      } else if (event.type === TouchType.Up || event.type === TouchType.Cancel) {
        this.isPressed = false;
      }
    })
    .onHover((isHover: boolean) => {
      if (this.disabled) {
        this.isHovered = false;
        return;
      }
      this.isHovered = isHover;
    })
    .accessibilityText(this.isOn ? '已开启' : '已关闭')
    .accessibilityLevel(this.disabled ? 'no' : 'auto')
  }
}
