/**
 * FocusRing —— 统一键盘焦点环（r4）
 *
 * 口径（对齐 web focus-ring：ring-accent 2px offset 2）：
 * - strokeWidth 2vp、margin 2vp（系统 focusBox 在组件外扩 2vp，需父容器允许溢出）
 * - 颜色取主题 accentBackground（不新造颜色档）
 *
 * 用法：Button/Column/Row 等可聚焦节点 `.focusable(true)` + `.focusBox(FocusRing.box(accentHex))`。
 * 行/列表：ListGroup padding 8 保证 2vp 外扩环仍在岛内（离岛边 ≥6）。
 */
import { LengthMetrics, ColorMetrics } from '@kit.ArkUI';

export class FocusRing {
  /** #RRGGBB / #AARRGGBB → FocusBoxStyle；非法值回退 accent 默认蓝 */
  static box(accentHex: string): FocusBoxStyle {
    let body: string = accentHex.startsWith('#') ? accentHex.slice(1) : accentHex;
    if (body.length === 8) {
      body = body.slice(2);
    }
    const parsed: number = parseInt(body.length === 6 ? body : '3478F6', 16);
    return {
      margin: LengthMetrics.vp(2),
      strokeWidth: LengthMetrics.vp(2),
      strokeColor: ColorMetrics.numeric(parsed)
    };
  }
}
