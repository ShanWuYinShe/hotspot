import { DesignConstants } from '../constants/DesignConstants';
import { AppThemeManager } from '../core/theme/AppThemeManager';
import { KEY_RESOLVED_THEME_ID } from '../core/theme/AppThemeTypes';

/**
 * ListGroup —— 列表分组容器（对齐前端 components/list/ListContainer.svelte）
 *
 * 契约来源：前端 ListContainer 只有两种形态，鸿蒙端按同一语义提供：
 * - bordered=true （默认）：直接位于开放页面区域的集合（设置项、关系列表、搜索结果、
 *   群组/分享/订阅条目等）。**唯一的圆角外框卡 + 行间发丝分隔线**——分组感由容器
 *   承担，行本身不得再叠第二层卡片边框/圆角/阴影。
 * - bordered=false：嵌入面板 / Sheet / 侧栏内的集合（会话列表、地图附近栏等）。
 *   无外框，仅行间发丝分隔线，行 hover/按压填充直接落在画布上。
 *
 * 为什么需要它（本次「去悬浮卡片化」的根因）：
 * 此前每个列表行各自带 surfaceBackground + 圆角 + elevatedShadow，等于「一行一张浮卡」，
 * 整屏看起来所有东西都在漂浮（层级噪音），而前端对应位置是「一个分组容器 + 分隔线」。
 * 引用规范：Material Design 3 Cards —— 卡片用于「围绕单一主题的内容与操作」，
 * 列表条目用 List（容器 + divider）而不是逐行卡片
 * （https://m3.material.io/components/cards/overview，Card.md 容器属性表）；
 * NN/g 卡片定义同样要求卡片是「一个概念单元的简短摘要」，不是密集条目的默认容器
 * （https://www.nngroup.com/articles/cards-component/）。
 *
 * 用法：
 *   ListGroup() {
 *     this.buildRowA()          // 行：只写 padding + 按压反馈，不写背景/圆角/投影
 *     ListRowDivider()          // 行间分隔线
 *     this.buildRowB()
 *   }
 *
 * 注意：容器自带 .clip(true)，行内按压填充不会溢出圆角。
 */
@Component
export struct ListGroup {
  @BuilderParam content: () => void;
  /** true = 开放页面区域的分组外框卡；false = 嵌入面板/Sheet 内的无框列表 */
  @Prop bordered: boolean = true;
  /** 附加内边距（页面侧边距由 PageShellContainer 承担时留 0） */
  @Prop groupPadding: number = 0;
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'vibrant_light';

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  build() {
    Column({ space: 0 }) {
      this.content();
    }
    .width('100%')
    .padding(this.groupPadding)
    .backgroundColor(this.bordered ? this.getPalette().surfaceBackground : DesignConstants.COLOR.TRANSPARENT)
    .borderRadius(this.bordered ? DesignConstants.SHAPE.COMPACT_CARD : 0)
    .border({
      width: this.bordered ? DesignConstants.BORDER_WIDTH.HAIRLINE : 0,
      color: this.getPalette().borderColor
    })
    // 分组卡静态投影 = surfaceShadow（白卡档）；禁止用 elevatedShadow（弹层/FAB 档），
    // 否则分组卡在页面上「浮起」——正是本次要消除的观感。
    // bordered=false（嵌入面板/Sheet）时不画外框也不画投影，只保留行间分隔线。
    .shadow(this.bordered ? this.getPalette().surfaceShadow : DesignConstants.SHADOW.SMALL)
    .clip(true)
  }
}

/**
 * 分组内行间发丝分隔线（对齐前端 ListContainer 的 divide-y divide-border-subtle）。
 * 缩进由调用方按行内容对齐（一般 = 行左侧 padding）。
 */
@Component
export struct ListRowDivider {
  @Prop startMargin: number = 0;
  @Prop endMargin: number = 0;
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'vibrant_light';

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  build() {
    Divider()
      .strokeWidth(DesignConstants.BORDER_WIDTH.HAIRLINE)
      .color(this.getPalette().borderColor)
      .margin({ left: this.startMargin, right: this.endMargin })
  }
}
