import { DesignConstants } from '../constants/DesignConstants';
import { BreakpointType } from '../core/responsive/Breakpoint';
import { AppThemeManager } from '../core/theme/AppThemeManager';
import { DEFAULT_RESOLVED_THEME_ID, KEY_RESOLVED_THEME_ID } from '../core/theme/AppThemeTypes';
import { AmbientBackdrop } from './layout/AmbientBackdrop';

/**
 * 页面内容栏容器（对齐前端 .page-shell + .page-pad）
 *
 * 核心原则（承前端 AGENTS.md「页面布局规范」）：
 * 内容居中收束在一条合理的测量栏内，宽屏不铺满全宽、不留大片空白；
 * 同时始终保留舒适的响应式横向留白，内容不紧贴容器边缘。
 *
 * 档位（对齐前端 .page-shell 变体）：
 * - default：feed / 卡片列表 / 网格（1180）
 * - wide：数据表格 / 管理后台 / 仪表盘（1320）
 * - narrow：详情 / 阅读 / 长文（720）
 * - form：设置 / 个人资料 / 编辑 / 搜索 / 登录（560）
 * - feed-narrow：moments 移动·窄平板（640）
 * - feed-medium：moments 宽平板（960）
 *
 * 用法：
 *   PageShellContainer({ variant: 'form', currentBreakpoint: this.currentBreakpoint }) {
 *     // 页面内容
 *   }
 *
 * 注意（承前端红线）：禁止页面直接用裸 max-width 各自为政，统一走本容器档位。
 */
export type PageShellVariant = 'default' | 'wide' | 'narrow' | 'form' | 'feed-narrow' | 'feed-medium';

/**
 * 页面 gutter 解析（唯一来源）：
 * 页面内需要**全宽元素**（封面图、通栏分区）时，用本函数取同一 gutter 做负 margin 抵消
 * （对齐前端 `-mx-[var(--cst-page-gutter)]`），而不是关掉容器留白或另写一套 padding。
 */
export function resolvePageGutter(breakpoint: BreakpointType): number {
  const gutter = DesignConstants.PAGE_SHELL;
  switch (breakpoint) {
    case 'xs':
    case 'sm':
      return gutter.GUTTER_MOBILE;
    case 'md':
      return gutter.GUTTER_TABLET;
    default:
      return gutter.GUTTER_DESKTOP;
  }
}

@Component
export struct PageShellContainer {
  @BuilderParam content: () => void;
  @Prop variant: PageShellVariant = 'default';
  @Prop currentBreakpoint: BreakpointType = 'sm';
  /** 关闭横向留白（父级已自行处理 padding 时用） */
  @Prop noHorizontalPadding: boolean = false;
  /** 是否占满父容器高度。滚动容器（Scroll/Refresh）内部必须传 false：
   *  height('100%') 会使内容高度恒等于视口，滚动范围归零、下方内容被裁剪 */
  @Prop fillHeight: boolean = true;
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = DEFAULT_RESOLVED_THEME_ID;

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  private getMaxWidth(): number {
    const shell = DesignConstants.PAGE_SHELL;
    switch (this.variant) {
      case 'wide':
        return shell.SHELL_WIDE_MAX; // 1320
      case 'narrow':
        return shell.SHELL_NARROW_MAX; // 720
      case 'form':
        return shell.SHELL_FORM_MAX; // 560
      case 'feed-narrow':
        return shell.FEED_NARROW_MAX; // 640
      case 'feed-medium':
        return shell.FEED_MEDIUM_MAX; // 960
      default:
        return shell.SHELL_DEFAULT_MAX; // 1180
    }
  }

  /**
   * 横向留白：**全站唯一 gutter 来源**（对齐前端 ContentColumn 的
   * --cst-page-gutter：移动 20 / 平板 24 / 桌面 28）。
   *
   * 契约（强制）：页面内部**不得**再为自己的正文加 left/right padding。
   * 此前 8 个页面在容器已带 16 的基础上又自加 16~20，实际留白变成 32~36，
   * 而其它页面只有 16~20 —— 各页正文左边缘不在同一条竖线上，滑动时内容
   * 左右边界跳动，是「页面看起来不协调」的主要来源。
   * 全宽元素（封面图、通栏列表）需要贴边时，用负 margin 抵消本 gutter
   * （对齐前端 -mx-[var(--cst-page-gutter)]），而不是关掉本容器留白。
   */
  private getHorizontalPadding(): number {
    return resolvePageGutter(this.currentBreakpoint);
  }

  build() {
    Stack({ alignContent: Alignment.Top }) {
      // 氛围画布：全站唯一装饰层（对齐 web 在外壳内容区挂 AmbientBackdrop）。
      // 只挂在「页面视口级」容器上（fillHeight=true）：装饰层固定于内容区、不随内部列表滚动，
      // 也不进入布局层级；页面自身不得再画第二套背景装饰。
      // fillHeight=false 的容器位于外页 Scroll 内（如设置页/管理后台），高度由内容撑开，
      // 不做整屏世界层（会随滚动拉伸）。**已登记平台差异（2026-10-26 复核 N3）**：这些页
      // 目前外层未再挂 AmbientBackdrop，即无氛围层（与 web 全站有氛围层不一致）；后续若要
      // 对齐，应由外层页面/外壳统一挂载单实例，而不是在滚动容器内挂。
      if (this.fillHeight) {
        AmbientBackdrop()
      }

      Row() {
        Column() {
          this.content()
        }
        .constraintSize({ maxWidth: this.getMaxWidth() })
        .width('100%')
      }
      .width('100%')
      .justifyContent(FlexAlign.Center)
      .padding({
        left: this.noHorizontalPadding ? 0 : this.getHorizontalPadding(),
        right: this.noHorizontalPadding ? 0 : this.getHorizontalPadding()
      })
    }
    .width('100%')
    .height(this.fillHeight ? '100%' : 'auto')
    .backgroundColor(this.getPalette().pageBackground)
  }
}
