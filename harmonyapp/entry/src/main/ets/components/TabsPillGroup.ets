/**
 * TabsPillGroup - 分类/筛选胶囊组（对齐前端 navigation/TabBar）
 * 两种形态：
 * - 默认（segmented）：等宽分段 + surfaceSubtle 灰底槽（系统分段控件视觉）
 * - freeWidth：透明容器 + 自适应内容宽左对齐（对齐前端 TabBar 胶囊行，
 *   激活项 accentMuted 淡底胶囊 + textPrimary 字（soft 弱高亮，亮 accent 实底
 *   收给主按钮，避免筛选态与主操作抢焦点），未激活项透明 + 次要色文字）
 *
 * 交互契约（四态）：
 * - 按压差异：选中胶囊按压 = 底色加深（active 加深反馈）；未选中按压 = SCALE_PRESSED_PILL 缩放
 * - hover（仅键鼠外接触发）：选中 = textPrimary 字已表达，无底色增量；未选中 = surfaceSubtle 淡底预览
 * - 触控热区：胶囊视觉高度不变，外层透明命中区扩到 LAYOUT.MIN_HIT_TARGET
 * - ForEach key 只含 label-index（拼 activeIndex 会导致切换激活时全部胶囊销毁重建、打断位移动画）
 */

import { DesignConstants } from '../constants/DesignConstants';
import { AppThemeManager } from '../core/theme/AppThemeManager';
import { DEFAULT_RESOLVED_THEME_ID, KEY_RESOLVED_THEME_ID } from '../core/theme/AppThemeTypes';
import { AppIcon } from './SystemIcon';
import { LucideIcon } from './LucideIcon';
import { curves } from '@kit.ArkUI';

export interface TabItem {
  label: string;
  icon?: AppIcon;
  /** 角标计数（如「未读 3」；对齐前端 TabBar { id, label, count? } 契约） */
  count?: number;
}

export interface TabsPillGroupActions {
  onChange?: (index: number) => void;
}

@Component
export struct TabsPillGroup {
  /** 当前按下胶囊的索引（-1 表示无）：按 index 记录，仅缩放被按的那颗，不再整组齐缩 */
  @State pressedIndex: number = -1;
  /** hover 中的胶囊索引（-1 表示无；onHover 仅键鼠外接触发，触屏不受影响） */
  @State hoveredIndex: number = -1;
  @Prop tabs: TabItem[] = [];
  @Prop activeIndex: number = 0;
  @Prop gap: number = DesignConstants.SPACING.XS;
  /** 自由宽度模式：透明容器 + 自适应内容宽（对齐前端 TabBar；缺省等宽分段） */
  @Prop freeWidth: boolean = false;
  actions?: TabsPillGroupActions;
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = DEFAULT_RESOLVED_THEME_ID;

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  private isActive(index: number): boolean {
    return this.activeIndex === index;
  }

  private dispatchChange(index: number) {
    const handler = this.actions?.onChange;
    if (handler) {
      handler(index);
    }
  }

  /** 胶囊底色：选中 = accentMuted 淡底（soft 弱高亮，对齐前端 TabBar 选中态——
   *  亮 accent 实底收给主按钮）；未选中 hover = surfaceSubtle 淡底（「将选中」预览，
   *  与选中态同语言不同档，避免 hover 即选中） */
  private getPillBackground(index: number, isActiveTab: boolean): ResourceColor {
    const palette = this.getPalette();
    if (isActiveTab) {
      return palette.accentMuted;
    }
    return this.hoveredIndex === index ? palette.surfaceSubtle : DesignConstants.COLOR.TRANSPARENT;
  }

  /**
   * 胶囊文字/图标色。
   *
   * 关键：选中态底色是 accentMuted（**淡色底**），前景必须是 textPrimary / accentBackground，
   * 不能用 accentText——accentText 的语义是「accentBackground 亮实底上的反色前景」
   * （深色主题 #071426），配到 accentMuted 深蓝底（#182943）上对比度极低，
   * 实测「距离 / 全部」等选中胶囊文字几乎不可见（附近页排序/筛选行整排看不清）。
   */
  private getPillForeground(index: number, isActiveTab: boolean): ResourceColor {
    const palette = this.getPalette();
    if (isActiveTab) {
      return palette.textPrimary;
    }
    // 未选中：hover 的时候底色会变 surfaceSubtle，文字随之提亮
    return this.hoveredIndex === index ? palette.textPrimary : palette.tabBarInactive;
  }

  @Builder
  private buildTab(tab: TabItem, index: number) {
    // 激活态必须在 builder 体内内联经 this.isActive(index) 读取（不可作按值参数传入）：
    // ForEach key 不含 activeIndex（保位移动画），复用项只有自身状态依赖变化才会刷新，
    // 按值传参会让高亮冻结在首帧（实测设置页明暗三档切换高亮不动，须重进页面才更新）
    Column() {
      Row({ space: DesignConstants.SPACING.SM2 }) {
        if (tab.icon) {
          LucideIcon({
            icon: tab.icon,
            iconSize: DesignConstants.COMPONENT.ICON_SIZE_2XS,
            color: this.getPillForeground(index, this.isActive(index))
          })
        }

        Text(tab.label)
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .fontWeight(this.isActive(index) ? DesignConstants.FONT.WEIGHT.MEDIUM : DesignConstants.FONT.WEIGHT.REGULAR)
          .fontColor(this.getPillForeground(index, this.isActive(index)))

        if (tab.count !== undefined && tab.count > 0) {
          Text(`${tab.count}`)
            .fontSize(DesignConstants.FONT.SIZE.XS2)
            .fontColor(this.getPillForeground(index, this.isActive(index)))
            .opacity(this.isActive(index) ? DesignConstants.OPACITY.SUBTLE : DesignConstants.OPACITY.EMPHASIZED)
        }
      }
      .justifyContent(FlexAlign.Center)
      .alignItems(VerticalAlign.Center)
      // 自由宽度：内层 Row 不设宽，由 label 文本撑开（设 100% 会依赖容器宽形成循环，
      // 导致胶囊被拉满——此前 3 次滚动容器尝试失败的真正原因）；
      // 等宽分段：占满各自 layoutWeight 份额
      .width(this.freeWidth ? undefined : '100%')
      // 两态统一 30 档：对齐前端 TabBar pill(py-1.5+text ≈ 28-32)，原分段 34 为近邻异档
      .height(DesignConstants.COMPONENT.FILTER_CHIP_HEIGHT)
      .borderRadius(DesignConstants.RADIUS.FULL)
      // 选中态播报（对齐前端 TabBar 的 aria-pressed={isActive}）：屏幕阅读器需知道
      // 当前哪个分类/Tab 处于选中，否则只能靠颜色猜测
      .accessibilityText(this.isActive(index) ? `${tab.label}，已选中` : tab.label)
      .backgroundColor(this.getPillBackground(index, this.isActive(index)))
      .animation({
        curve: curves.springMotion(
          DesignConstants.INTERACTION.SPRING_RESPONSE_TAB,
          DesignConstants.INTERACTION.SPRING_DAMPING_TAB
        )
      })
    }
    // 自由宽度模式：水平 padding 承载胶囊内距（对齐前端 px-3.5=14），使胶囊收缩到内容宽
    .padding({
      left: this.freeWidth ? DesignConstants.SPACING.MD3 : 0,
      right: this.freeWidth ? DesignConstants.SPACING.MD3 : 0
    })
    // 等宽分段用 layoutWeight(1)；自由宽度收缩到内容宽
    .layoutWeight(this.freeWidth ? 0 : 1)
    // 按压差异：选中胶囊按压不再缩放（底色已表达选中），未选中按压缩放（「将选中」预览）
    .scale(!this.isActive(index) && this.pressedIndex === index ?
    {
      x: DesignConstants.INTERACTION.SCALE_PRESSED_PILL,
      y: DesignConstants.INTERACTION.SCALE_PRESSED_PILL
    } : { x: 1, y: 1 })
    .animation({
      curve: curves.springMotion(
        DesignConstants.INTERACTION.SPRING_RESPONSE_PRESS,
        DesignConstants.INTERACTION.SPRING_DAMPING_PRESS
      )
    })
    .onTouch((event: TouchEvent) => {
      if (event.type === TouchType.Down) {
        this.pressedIndex = index;
      } else if (event.type === TouchType.Up || event.type === TouchType.Cancel) {
        this.pressedIndex = -1;
      }
    })
    .onHover((isHover: boolean) => {
      this.hoveredIndex = isHover ? index : -1;
    })
    .onClick(() => {
      if (!this.isActive(index)) {
        this.dispatchChange(index);
      }
    })
    // 触控热区：外层 Column 透明，constraintSize 只扩命中区（胶囊视觉高度由内层 Row 固定）
    .constraintSize({ minHeight: DesignConstants.LAYOUT.MIN_HIT_TARGET })
    // 热区 Column 高于视觉胶囊（44 > 32），必须垂直居中——否则胶囊贴 Column 顶、
    // 热区空隙全落在下方，视觉上「分段内容靠上不居中」（实拍：群组详情 tabs）
    .justifyContent(FlexAlign.Center)
    .alignItems(HorizontalAlign.Center)
  }

  build() {
    Row({ space: this.gap }) {
      ForEach(this.tabs, (tab: TabItem, index: number) => {
        this.buildTab(tab, index)
      }, (tab: TabItem, index: number) => `${tab.label}-${index}`)
    }
    // 自由宽度不设 width：由内容撑开（外层可套 Scroll 横向滚动，对齐前端 overflow-x-auto 胶囊行）；
    // 等宽分段占满容器
    .width(this.freeWidth ? undefined : '100%')
    // 自由宽度模式：透明容器、无灰底槽、无内边距（对齐前端 TabBar 容器）
    .padding(this.freeWidth ? 0 : DesignConstants.SPACING.THREE)
    .borderRadius(this.freeWidth ? 0 : DesignConstants.RADIUS.FULL)
    .backgroundColor(this.freeWidth ? DesignConstants.COLOR.TRANSPARENT : this.getPalette().surfaceSubtle)
    .alignItems(VerticalAlign.Center)
  }
}
