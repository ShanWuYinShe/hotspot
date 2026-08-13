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

  build() {
    Column() {
      this.content();
    }
    .width('100%')
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
      } else if (event.type === TouchType.Up || event.type === TouchType.Cancel) {
        if (this.isPressed && event.type === TouchType.Up) {
          this.onPress();
        }
        this.isPressed = false;
      }
    })
  }
}
