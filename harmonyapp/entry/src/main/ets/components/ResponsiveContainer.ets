/**
 * ResponsiveContainer — 路由页内容限宽居中原语
 *
 * NavDestination 默认铺满全屏，桌面/平板宽屏下内容会被拉成超长行。
 * 经此包裹后，内容居中收束在合理最大宽度内（对齐 PhoneLayout/DesktopLayout
 * 的 CONTAINER_MAX_WIDTH_VP 与 frontend .page-shell），手机端(xs/sm)充满。
 *
 * 用法（路由页 Scroll / 根内容外层）：
 *   Scroll() {
 *     ResponsiveContainer() {
 *       Column() { ...内容... }
 *     }
 *   }
 *
 * 断点来源：AppStorage('currentBreakpoint')，由 EntryAbility 实时写入。
 */
import { BreakpointType, CONTAINER_MAX_WIDTH_VP } from '../core/responsive/Breakpoint';

@Component
export struct ResponsiveContainer {
  @StorageProp('currentBreakpoint') currentBreakpoint: BreakpointType = 'sm';
  /** true = 内层列撑满高度（全屏页根限宽，如 NavDestination 直接子，配合 List/Tabs 占满）；
   *  false = 高度由内容（Scroll 内限宽）。 */
  @Prop fill: boolean = false;
  @BuilderParam content: () => void;

  build() {
    Row() {
      Column() {
        this.content();
      }
      .width('100%')
      .height(this.fill ? '100%' : undefined)
      .constraintSize({ maxWidth: this.getContainerMaxWidth() })
    }
    .width('100%')
    .justifyContent(FlexAlign.Center)
  }

  /** xs/sm 充满；md 起按断点限宽（720/960/1200/1440 vp）。 */
  private getContainerMaxWidth(): Length {
    switch (this.currentBreakpoint) {
      case 'md':
        return CONTAINER_MAX_WIDTH_VP.md;
      case 'lg':
        return CONTAINER_MAX_WIDTH_VP.lg;
      case 'xl':
        return CONTAINER_MAX_WIDTH_VP.xl;
      case 'xxl':
        return CONTAINER_MAX_WIDTH_VP.xxl;
      default: // xs / sm
        return '100%';
    }
  }
}
