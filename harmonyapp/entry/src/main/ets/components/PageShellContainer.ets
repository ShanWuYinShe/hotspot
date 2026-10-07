import { DesignConstants } from '../constants/DesignConstants';
import { BreakpointType } from '../core/responsive/Breakpoint';
import { AppThemeManager } from '../core/theme/AppThemeManager';
import { KEY_RESOLVED_THEME_ID } from '../core/theme/AppThemeTypes';

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
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'vibrant_light';

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

  /** 横向留白：对齐前端 clamp，移动兜底 16，随视口增大 */
  private getHorizontalPadding(): number {
    switch (this.currentBreakpoint) {
      case 'xs':
      case 'sm':
        return DesignConstants.SPACING.LG; // 16
      case 'md':
        return DesignConstants.SPACING.LG2;
      case 'lg':
        return DesignConstants.SPACING.XL; // 24
      default:
        return DesignConstants.SPACING.XXL; // 32
    }
  }

  build() {
    Stack({ alignContent: Alignment.Top }) {

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
