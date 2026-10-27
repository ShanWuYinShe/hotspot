/**
 * BackToTopButton —— 长列表回顶浮动按钮（R9）
 *
 * 48vp 圆形，overlay 绝对定位（不参与布局 / 无 CLS）；显隐走 MotionUtil：
 * reduced 时直接显隐，否则 180ms opacity 过渡；不可见时禁触且移出无障碍树。
 */
import { DesignConstants } from '../constants/DesignConstants';
import { AppIcon } from './SystemIcon';
import { LucideIcon } from './LucideIcon';
import { MotionUtil } from '../core/accessibility/MotionUtil';
import { AppThemeManager } from '../core/theme/AppThemeManager';
import { DEFAULT_RESOLVED_THEME_ID, KEY_RESOLVED_THEME_ID } from '../core/theme/AppThemeTypes';

@Component
export struct BackToTopButton {
  @Prop visible: boolean = false;
  onTap: () => void = () => {};
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = DEFAULT_RESOLVED_THEME_ID;

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  build() {
    Row() {
      LucideIcon({
        icon: AppIcon.ARROW_UP,
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
    .opacity(this.visible ? 1 : 0)
    .animation({
      duration: MotionUtil.reduced() ? 0 : DesignConstants.ANIMATION.DURATION_SHORT,
      curve: DesignConstants.ANIMATION.CURVE_EASE_OUT
    })
    .enabled(this.visible)
    .hitTestBehavior(this.visible ? HitTestMode.Default : HitTestMode.None)
    .accessibilityLevel(this.visible ? 'auto' : 'no')
    .accessibilityText('回到顶部')
    .onClick(() => {
      if (this.visible) {
        this.onTap();
      }
    })
  }
}
