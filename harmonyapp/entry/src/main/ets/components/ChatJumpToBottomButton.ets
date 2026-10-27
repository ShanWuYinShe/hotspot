/**
 * ChatJumpToBottomButton —— 聊天「回到底部」浮动按钮（R10）
 *
 * 48vp 圆钮 + ARROW_DOWN；count>0 时右上角计数徽标（≥100 显示 99+）。
 * 显隐走 MotionUtil：reduced 直接显隐，否则 150ms opacity；不可见时禁触且移出无障碍树。
 * 由页面 overlay 绝对定位（不参与布局 / 无 CLS），位于消息列表右下、输入栏上方。
 */
import { DesignConstants } from '../constants/DesignConstants';
import { AppIcon } from './SystemIcon';
import { LucideIcon } from './LucideIcon';
import { MotionUtil } from '../core/accessibility/MotionUtil';
import { AppThemeManager } from '../core/theme/AppThemeManager';
import { DEFAULT_RESOLVED_THEME_ID, KEY_RESOLVED_THEME_ID } from '../core/theme/AppThemeTypes';

@Component
export struct ChatJumpToBottomButton {
  @Prop visible: boolean = false;
  /** 非近底部时累计的新消息数（0 = 只显示箭头） */
  @Prop count: number = 0;
  onTap: () => void = () => {};
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = DEFAULT_RESOLVED_THEME_ID;

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  private getBadgeText(): string {
    return this.count > 99 ? '99+' : `${this.count}`;
  }

  private getAccessibilityLabel(): string {
    return this.count > 0 ? `回到底部，${this.count} 条新消息` : '回到底部';
  }

  build() {
    Stack({ alignContent: Alignment.TopEnd }) {
      Row() {
        LucideIcon({
          icon: AppIcon.ARROW_DOWN,
          iconSize: DesignConstants.COMPONENT.ICON_SIZE_SM,
          color: this.getPalette().textSecondary
        })
      }
      .width(48)
      .height(48)
      .borderRadius(DesignConstants.RADIUS.FULL)
      .backgroundColor(this.getPalette().surfaceBackground)
      .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getPalette().borderColor })
      .shadow(this.getPalette().controlShadow)
      .justifyContent(FlexAlign.Center)
      .alignItems(VerticalAlign.Center)

      if (this.count > 0) {
        Row() {
          Text(this.getBadgeText())
            .fontSize(DesignConstants.FONT.SIZE.XS2)
            .fontWeight(DesignConstants.FONT.WEIGHT.SEMIBOLD)
            .fontColor(this.getPalette().accentText)
            .maxLines(1)
        }
        .constraintSize({ minWidth: 18 })
        .height(18)
        .padding({ left: DesignConstants.SPACING.XS, right: DesignConstants.SPACING.XS })
        .borderRadius(DesignConstants.RADIUS.FULL)
        .backgroundColor(this.getPalette().accentBackground)
        .justifyContent(FlexAlign.Center)
        .translate({ x: 6, y: -6 })
        .accessibilityLevel('no')
      }
    }
    .width(48)
    .height(48)
    .opacity(this.visible ? 1 : 0)
    .animation({
      duration: MotionUtil.reduced() ? 0 : DesignConstants.ANIMATION.DURATION_FAST,
      curve: DesignConstants.ANIMATION.CURVE_EASE_OUT
    })
    .enabled(this.visible)
    .hitTestBehavior(this.visible ? HitTestMode.Default : HitTestMode.None)
    .accessibilityLevel(this.visible ? 'auto' : 'no')
    .accessibilityText(this.getAccessibilityLabel())
    .onClick(() => {
      if (this.visible) {
        this.onTap();
      }
    })
  }
}
