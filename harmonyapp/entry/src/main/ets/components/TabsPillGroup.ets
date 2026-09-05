/**
 * TabsPillGroup - 分类/筛选胶囊组（对齐前端 navigation/TabBar）
 * 两种形态：
 * - 默认（segmented）：等宽分段 + surfaceSubtle 灰底槽（系统分段控件视觉）
 * - freeWidth：透明容器 + 自适应内容宽左对齐（对齐前端 TabBar 胶囊行，
 *   激活项 accent 实底胶囊 + glow，未激活项透明 + 次要色文字）
 *
 * 交互契约（四态）：
 * - 按压差异：选中胶囊按压 = 底色加深（active 加深反馈）；未选中按压 = SCALE_PRESSED_PILL 缩放
 * - hover（仅键鼠外接触发）：选中 = accentShadow 增强；未选中 = 淡底填充 + 主文字色预览
 * - 触控热区：胶囊视觉高度不变，外层透明命中区扩到 LAYOUT.MIN_HIT_TARGET
 * - ForEach key 只含 label-index（拼 activeIndex 会导致切换激活时全部胶囊销毁重建、打断位移动画）
 */

import { DesignConstants } from '../constants/DesignConstants';
import { AppThemeManager } from '../core/theme/AppThemeManager';
import { KEY_RESOLVED_THEME_ID, AppThemeShadow } from '../core/theme/AppThemeTypes';
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
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'vibrant_light';

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

  /** 胶囊底色：选中 hover 保持 accent 实底（阴影增强表达，见下方 shadow 分支）；
   *  未选中 hover = accentMuted 淡底（「将选中」预览） */
  private getPillBackground(index: number, isActiveTab: boolean): ResourceColor {
    const palette = this.getPalette();
    if (isActiveTab) {
      return palette.accentBackground;
    }
    return this.hoveredIndex === index ? palette.accentMuted : DesignConstants.COLOR.TRANSPARENT;
  }

  /** 选中胶囊 hover 阴影增强（同色系光晕抬档，表达已选中项的可交互增量） */
  private getPillShadow(isActiveTab: boolean): AppThemeShadow {
    const base = this.getPalette().accentShadow;
    if (!isActiveTab || this.hoveredIndex < 0) {
      return base;
    }
    return new AppThemeShadow(
      Math.round(base.radius * 1.6),
      base.color,
      base.offsetX,
      base.offsetY
    );
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
            iconSize: DesignConstants.FONT.SIZE.MD,
            color: this.isActive(index) || this.hoveredIndex === index ?
            this.getPalette().accentText : this.getPalette().tabBarInactive
          })
        }

        Text(tab.label)
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .fontWeight(this.isActive(index) ? DesignConstants.FONT.WEIGHT.MEDIUM : DesignConstants.FONT.WEIGHT.REGULAR)
          .fontColor(this.isActive(index) || this.hoveredIndex === index ?
          this.getPalette().accentText : this.getPalette().tabBarInactive)

        if (tab.count !== undefined && tab.count > 0) {
          Text(`${tab.count}`)
            .fontSize(DesignConstants.FONT.SIZE.XS2)
            .fontColor(this.isActive(index) ? this.getPalette().accentText : this.getPalette().tabBarInactive)
            .opacity(this.isActive(index) ? DesignConstants.OPACITY.SUBTLE : DesignConstants.OPACITY.EMPHASIZED)
        }
      }
      .justifyContent(FlexAlign.Center)
      .alignItems(VerticalAlign.Center)
      .width('100%')
      .height(this.freeWidth ?
      DesignConstants.COMPONENT.FILTER_CHIP_HEIGHT :
      DesignConstants.COMPONENT.ACTION_BUTTON_HEIGHT_LG)
      .borderRadius(DesignConstants.RADIUS.FULL)
      .backgroundColor(this.getPillBackground(index, this.isActive(index)))
      .shadow(this.isActive(index) ? this.getPillShadow(true) : {
        radius: 0,
        color: DesignConstants.COLOR.TRANSPARENT,
        offsetX: 0,
        offsetY: 0
      })
      .animation({
        curve: curves.springMotion(
          DesignConstants.INTERACTION.SPRING_RESPONSE_TAB,
          DesignConstants.INTERACTION.SPRING_DAMPING_TAB
        )
      })
    }
    // 自由宽度模式：水平 padding 由外层承载，使胶囊收缩到内容宽
    .padding({
      left: this.freeWidth ? DesignConstants.SPACING.LG : 0,
      right: this.freeWidth ? DesignConstants.SPACING.LG : 0
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
  }

  build() {
    Row({ space: this.gap }) {
      ForEach(this.tabs, (tab: TabItem, index: number) => {
        this.buildTab(tab, index)
      }, (tab: TabItem, index: number) => `${tab.label}-${index}`)
    }
    .width('100%')
    // 自由宽度模式：透明容器、无灰底槽、无内边距（对齐前端 TabBar 容器）
    .padding(this.freeWidth ? 0 : DesignConstants.SPACING.THREE)
    .borderRadius(this.freeWidth ? 0 : DesignConstants.RADIUS.LG)
    .backgroundColor(this.freeWidth ? DesignConstants.COLOR.TRANSPARENT : this.getPalette().surfaceSubtle)
    .alignItems(VerticalAlign.Center)
  }
}
