/**
 * Layout — 顶层布局切换器
 *
 * 根据屏幕断点（GridRow.currentBreakpoint）选择 Phone / Desktop 布局。
 * - xs / sm / md：PhoneLayout（手机 + 平板）
 * - lg / xl / xxl：DesktopLayout（桌面 / 折叠屏 / PC）
 *
 * 使用方式（在 Entry / Index 组件的 build() 中）：
 *   Layout({ currentBreakpoint: this.currentBreakpoint })
 *
 * 设计说明：
 * - 阶段 2 状态：抽象层 + 占位实现
 * - 阶段 3 任务：
 *   1. 从 features/main/pages/Index.ets 抽取手机端业务内容到 PhoneLayout
 *   2. 桌面端业务内容接入 DesktopLayout
 *   3. Index.ets 简化为：
 *        build() { Layout({ currentBreakpoint: this.currentBreakpoint }) }
 */

import { type BreakpointType, useSidebarChrome } from '../../core/responsive/Breakpoint';
import { PhoneLayout } from './PhoneLayout';
import { DesktopLayout } from './DesktopLayout';

@Component
export struct Layout {
  @Prop currentBreakpoint: BreakpointType;

  build() {
    if (!useSidebarChrome(this.currentBreakpoint)) {
      PhoneLayout();
    } else {
      DesktopLayout();
    }
  }
}
