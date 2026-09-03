import { DesignConstants } from '../constants/DesignConstants';
import { AppThemeManager } from '../core/theme/AppThemeManager';
import { AppThemeShadow, KEY_RESOLVED_THEME_ID } from '../core/theme/AppThemeTypes';
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
    // neutral → 对齐前端 ActionIconButton variant=ghost：透明底、无边框、无阴影
    return DesignConstants.COLOR.TRANSPARENT;
  }

  private getBorderColor(): ResourceColor | Color {
    if (this.kind === 'primary') {
      return this.getPalette().accentBackground;
    }
    return DesignConstants.COLOR.TRANSPARENT;
  }

  private getShadowColor(): ShadowOptions {
    if (this.kind === 'primary') {
      return this.getPalette().accentShadow;
    }
    return new AppThemeShadow(0, DesignConstants.COLOR.TRANSPARENT, 0, 0);
  }

  private getIconColor(): ResourceColor | Color {
    if (this.kind === 'primary') {
      return this.getPalette().accentText;
    }
    // ghost 图标用主文字色（对齐前端 PageHeader 末尾透明钮——深色保证在亮底上的可见性）
    return this.getPalette().textPrimary;
  }

  build() {
    // 触控命中区 ≥44vp（对齐 WCAG 2.5.5 / HarmonyOS 最小热区）：外层透明扩展，
    // 内层保持视觉尺寸（buttonSize 默认 36 不变），按压/点击绑定在外层。
    Stack() {
      Button() {
        LucideIcon({
          icon: this.icon,
          iconSize: this.glyphSize,
          color: this.getIconColor()
        })
      }
      // Circle：同 SystemButton，Normal 类型忽略 borderRadius
      .type(ButtonType.Circle)
      .stateEffect(false)
      .width(this.buttonSize)
      .height(this.buttonSize)
      .backgroundColor(this.getBackgroundColor())
      .borderRadius(DesignConstants.RADIUS.FULL)
      .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getBorderColor() })
      .shadow(this.getShadowColor())
    }
    .width(Math.max(this.buttonSize, DesignConstants.LAYOUT.MIN_HIT_TARGET))
    .height(Math.max(this.buttonSize, DesignConstants.LAYOUT.MIN_HIT_TARGET))
    .alignContent(Alignment.Center)
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
