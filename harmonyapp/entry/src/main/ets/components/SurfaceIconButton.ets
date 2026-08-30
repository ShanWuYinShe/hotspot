import { DesignConstants } from '../constants/DesignConstants';
import { AppThemeManager } from '../core/theme/AppThemeManager';
import { KEY_RESOLVED_THEME_ID } from '../core/theme/AppThemeTypes';
import { AppIcon } from './SystemIcon';
import { LucideIcon } from './LucideIcon';
import { curves } from '@kit.ArkUI';

export type SurfaceIconButtonKind = 'primary' | 'neutral';

@Component
export struct SurfaceIconButton {
  @Prop icon: AppIcon = AppIcon.BACK;
  @Prop kind: SurfaceIconButtonKind = 'primary';
  @Prop buttonSize: number = DesignConstants.LAYOUT.ICON_BUTTON_SIZE; // 36
  @Prop glyphSize: number = DesignConstants.COMPONENT.ICON_SIZE_XS; // 16
  @Prop disabled: boolean = false;
  /** 无障碍语义描述（图标钮无文字，读屏依赖此标签，如全局返回钮「返回」） */
  @Prop accessibilityLabel: string = '';
  onPress?: () => void;
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'vibrant_light';

  @State isPressed: boolean = false;

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  private getBackgroundColor(): ResourceColor | Color {
    if (this.kind === 'primary') {
      return this.getPalette().accentBackground;
    }
    return this.getPalette().surfaceBackground;
  }

  private getBorderColor(): ResourceColor | Color {
    if (this.kind === 'primary') {
      return this.getPalette().accentBackground;
    }
    return this.getPalette().borderColor;
  }

  private getShadowColor(): ShadowOptions {
    return this.kind === 'primary' ? this.getPalette().accentShadow : this.getPalette().surfaceShadow;
  }

  private getIconColor(): ResourceColor | Color {
    if (this.kind === 'primary') {
      return this.getPalette().accentText;
    }
    return this.getPalette().textPrimary;
  }

  build() {
    Button() {
      LucideIcon({
        icon: this.icon,
        iconSize: this.glyphSize,
        color: this.getIconColor()
      })
    }
    .type(ButtonType.Normal)
    .stateEffect(false)
    .width(this.buttonSize)
    .height(this.buttonSize)
    .backgroundColor(this.getBackgroundColor())
    .borderRadius(DesignConstants.RADIUS.FULL)
    .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getBorderColor() })
    .shadow(this.getShadowColor())
    .enabled(!this.disabled)
    .opacity(this.disabled ? DesignConstants.OPACITY.DISABLED : 1)
    .accessibilityText(this.accessibilityLabel)
    .scale(this.isPressed ?
      { x: DesignConstants.INTERACTION.SCALE_PRESSED_ICON, y: DesignConstants.INTERACTION.SCALE_PRESSED_ICON } :
      { x: 1, y: 1 })
    .animation({
      curve: curves.springMotion(
        DesignConstants.INTERACTION.SPRING_RESPONSE_PRESS,
        DesignConstants.INTERACTION.SPRING_DAMPING_PRESS
      )
    })
    .onTouch((event: TouchEvent) => {
      if (this.disabled) {
        return;
      }
      if (event.type === TouchType.Down) {
        this.isPressed = true;
      } else if (event.type === TouchType.Up || event.type === TouchType.Cancel) {
        this.isPressed = false;
      }
    })
    .onClick(() => {
      if (this.disabled) {
        return;
      }
      this.onPress?.();
    })
  }
}
