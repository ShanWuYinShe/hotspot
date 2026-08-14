import { DesignConstants } from '../constants/DesignConstants';
import { curves } from '@kit.ArkUI';

/**
 * Pressable — 统一按压反馈容器（对齐前端 active:scale 语义与 SystemCard 按压三件套）。
 *
 * 用法：
 *   Pressable({ onPress: () => this.handleTap() }) {
 *     Row() { /* 行内容 *\/ }
 *     .width('100%')
 *   }
 *
 * 内部处理 isPressed + spring 缩放 + Down/Up/Cancel 状态；页面内联 onClick 元素
 * 一律收敛到本组件，禁止再手写 onTouch + scale + animation 三件套。
 * 默认点击在 TouchType.Up 触发（与按钮一致），disabled 时无按压态且不触发。
 * 手指滑出组件区域后松开视为取消：按压态提前释放且不触发 onPress
 * （TouchObject.x/y 为相对组件左上角的触点坐标，出负值或超界即滑出）。
 * 属性命名 onPress：onClick 是 CustomComponent 保留属性（事件链），不可覆盖。
 */
@Component
export struct Pressable {
  @BuilderParam content: () => void;
  /** 按压/点击回调（Up 触发）。不传则仅保留按压反馈（纯装饰）。 */
  onPress: () => void = () => {
  };
  /** 按压缩放比例（默认对齐 SystemCard 的 SCALE_PRESSED_CARD）。 */
  pressScale: number = DesignConstants.INTERACTION.SCALE_PRESSED_CARD;
  /** 禁用态：无按压反馈、不触发点击。 */
  disabled: boolean = false;

  @State isPressed: boolean = false;
  @State slideOut: boolean = false;
  @State pressWidth: number = 0;
  @State pressHeight: number = 0;

  build() {
    Column() {
      this.content();
    }
    .width('100%')
    .onAreaChange((_oldValue: Area, newValue: Area) => {
      this.pressWidth = newValue.width as number;
      this.pressHeight = newValue.height as number;
    })
    .scale(this.isPressed && !this.disabled ?
      { x: this.pressScale, y: this.pressScale } :
      { x: 1, y: 1 })
    .animation({
      curve: curves.springMotion(
        DesignConstants.INTERACTION.SPRING_RESPONSE_PRESS,
        DesignConstants.INTERACTION.SPRING_DAMPING_PRESS
      )
    })
    .onTouch((event: TouchEvent) => {
      if (this.disabled) {
        this.isPressed = false;
        return;
      }
      if (event.type === TouchType.Down) {
        this.isPressed = true;
        this.slideOut = false;
      } else if (event.type === TouchType.Move && event.touches.length > 0) {
        const touch = event.touches[0];
        if (this.isPressed &&
          (touch.x < 0 || touch.y < 0 || touch.x > this.pressWidth || touch.y > this.pressHeight)) {
          this.isPressed = false;
          this.slideOut = true;
        }
      } else if (event.type === TouchType.Up) {
        if (this.isPressed && !this.slideOut) {
          this.onPress();
        }
        this.isPressed = false;
        this.slideOut = false;
      } else if (event.type === TouchType.Cancel) {
        this.isPressed = false;
        this.slideOut = false;
      }
    })
  }
}
