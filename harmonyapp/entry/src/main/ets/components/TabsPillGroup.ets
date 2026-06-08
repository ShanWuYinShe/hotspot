/**
 * TabsPillGroup - 分段控制式子标签组件
 * 参照 Soul / 陌陌等主流社交 App 的子标签切换设计：
 * - 等宽分段布局
 * - 选中项：品牌色背景 + 白字 + 微阴影（浮起感）
 * - 未选中项：透明背景 + 次要色文字
 */

import { DesignConstants } from '../constants/DesignConstants';
import { AppThemeManager } from '../core/theme/AppThemeManager';
import { KEY_RESOLVED_THEME_ID } from '../core/theme/AppThemeTypes';
import { AppIcon } from './SystemIcon';
import { curves } from '@kit.ArkUI';

export interface TabItem {
  label: string;
  icon?: AppIcon;
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
  actions?: TabsPillGroupActions;
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'figma_neu_light';

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
      Row({ space: 6 }) {
        if (tab.icon) {
          Text() {
            SymbolSpan($r(tab.icon))
              .fontSize(14)
              .fontColor(isActiveTab ? [this.getPalette().accentText] : [this.getPalette().tabBarInactive])
          }
        }

        Text(tab.label)
          .fontSize(DesignConstants.FONT.SIZE.MD)
          .fontWeight(isActiveTab ? FontWeight.Medium : FontWeight.Regular)
          .fontColor(isActiveTab ? this.getPalette().accentText : this.getPalette().tabBarInactive)
      }
      .justifyContent(FlexAlign.Center)
      .alignItems(VerticalAlign.Center)
      .width('100%')
      .height(34)
      .borderRadius(DesignConstants.RADIUS.XL)
      .backgroundColor(isActiveTab ? this.getPalette().accentBackground : Color.Transparent)
      .shadow(isActiveTab ? this.getPalette().accentShadow : {
        radius: 0,
        color: '#00000000',
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
    .layoutWeight(1)
    .scale(this.isTabPressed ? { x: DesignConstants.INTERACTION.SCALE_PRESSED_BUTTON, y: DesignConstants.INTERACTION.SCALE_PRESSED_BUTTON } : { x: 1, y: 1 })
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
    .padding(3)
    .borderRadius(DesignConstants.RADIUS.XXL)
    .backgroundColor(this.getPalette().surfaceSubtle)
    .alignItems(VerticalAlign.Center)
  }
}
