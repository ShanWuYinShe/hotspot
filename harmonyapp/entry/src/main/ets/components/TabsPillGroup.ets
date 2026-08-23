/**
 * TabsPillGroup - 分类/筛选胶囊组（对齐前端 navigation/TabBar）
 * 两种形态：
 * - 默认（segmented）：等宽分段 + surfaceSubtle 灰底槽（系统分段控件视觉）
 * - freeWidth：透明容器 + 自适应内容宽左对齐（对齐前端 TabBar 胶囊行，
 *   激活项 accent 实底胶囊 + glow，未激活项透明 + 次要色文字）
 */

import { DesignConstants } from '../constants/DesignConstants';
import { AppThemeManager } from '../core/theme/AppThemeManager';
import { KEY_RESOLVED_THEME_ID } from '../core/theme/AppThemeTypes';
import { AppIcon } from './SystemIcon';
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
  @State isTabPressed: boolean = false;
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

  @Builder
  private buildTab(tab: TabItem, index: number, isActiveTab: boolean) {
    Column() {
      Row({ space: DesignConstants.SPACING.SM2 }) {
        if (tab.icon) {
          Text() {
            SymbolSpan($r(tab.icon))
              .fontSize(DesignConstants.FONT.SIZE.MD)
              .fontColor(isActiveTab ? [this.getPalette().accentText] : [this.getPalette().tabBarInactive])
          }
        }

        Text(tab.label)
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .fontWeight(isActiveTab ? DesignConstants.FONT.WEIGHT.MEDIUM : DesignConstants.FONT.WEIGHT.REGULAR)
          .fontColor(isActiveTab ? this.getPalette().accentText : this.getPalette().tabBarInactive)

        if (tab.count !== undefined && tab.count > 0) {
          Text(`${tab.count}`)
            .fontSize(DesignConstants.FONT.SIZE.XS_MINUS)
            .fontColor(isActiveTab ? this.getPalette().accentText : this.getPalette().tabBarInactive)
            .opacity(isActiveTab ? DesignConstants.OPACITY.SUBTLE : DesignConstants.OPACITY.EMPHASIZED)
        }
      }
      .justifyContent(FlexAlign.Center)
      .alignItems(VerticalAlign.Center)
      .width('100%')
      .height(this.freeWidth ? DesignConstants.COMPONENT.FILTER_CHIP_HEIGHT : DesignConstants.COMPONENT.ACTION_BUTTON_HEIGHT_LG)
      .borderRadius(DesignConstants.RADIUS.FULL)
      .backgroundColor(isActiveTab ? this.getPalette().accentBackground : DesignConstants.COLOR.TRANSPARENT)
      .shadow(isActiveTab ? this.getPalette().accentShadow : {
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
    .scale(this.isTabPressed ? { x: DesignConstants.INTERACTION.SCALE_PRESSED_PILL, y: DesignConstants.INTERACTION.SCALE_PRESSED_PILL } : { x: 1, y: 1 })
    .animation({
      curve: curves.springMotion(
        DesignConstants.INTERACTION.SPRING_RESPONSE_PRESS,
        DesignConstants.INTERACTION.SPRING_DAMPING_PRESS
      )
    })
    .onTouch((event: TouchEvent) => {
      if (event.type === TouchType.Down) {
        this.isTabPressed = true;
      } else if (event.type === TouchType.Up || event.type === TouchType.Cancel) {
        this.isTabPressed = false;
      }
    })
    .onClick(() => {
      if (!isActiveTab) {
        this.dispatchChange(index);
      }
    })
  }

  build() {
    Row({ space: this.gap }) {
      ForEach(this.tabs, (tab: TabItem, index: number) => {
        this.buildTab(tab, index, this.isActive(index));
      }, (tab: TabItem, index: number) => `${tab.label}-${index}-${this.activeIndex}`)
    }
    .width('100%')
    // 自由宽度模式：透明容器、无灰底槽、无内边距（对齐前端 TabBar 容器）
    .padding(this.freeWidth ? 0 : DesignConstants.SPACING.THREE)
    .borderRadius(this.freeWidth ? 0 : DesignConstants.RADIUS.LG)
    .backgroundColor(this.freeWidth ? DesignConstants.COLOR.TRANSPARENT : this.getPalette().surfaceSubtle)
    .alignItems(VerticalAlign.Center)
  }
}
