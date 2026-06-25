import { DesignConstants } from '../constants/DesignConstants';
import { AppThemeManager } from '../core/theme/AppThemeManager';
import { KEY_RESOLVED_THEME_ID } from '../core/theme/AppThemeTypes';
import { AppIcon } from './SystemIcon';
import { curves } from '@kit.ArkUI';

/**
 * 通用按钮组件（对齐前端 button 设计系统）
 *
 * 变体（variant）：
 * - primary：品牌色填充 + 白字 + 品牌阴影（对齐前端 .brand + --text-on-accent + --shadow-brand-btn）
 * - secondary：次级填充面 + 主文字色（对齐前端 surface-flat 二级操作）
 * - outline：透明底 + 品牌色描边 + 品牌色文字
 * - ghost：透明底 + 主文字色（对齐前端 ghost）
 * - danger：错误色填充 + 白字（删除/危险操作）
 *
 * 尺寸（size）：small(36) / default(42) / large(48)
 *
 * 交互：press 缩放 0.95（对齐前端 active:scale-[0.95]），springMotion 回弹
 */
export type ButtonVariant = 'primary' | 'secondary' | 'outline' | 'ghost' | 'danger';
export type ButtonSize = 'small' | 'default' | 'large';

@Reusable
@Component
export struct SystemButton {
  @Prop label: string = '';
  /** 前置图标（showIcon=true 时渲染） */
  @Prop icon: AppIcon = AppIcon.ADD;
  @Prop showIcon: boolean = false;
  @Prop variant: ButtonVariant = 'primary';
  // 注：不可命名为 size——与 CustomComponent 基类的 size() 方法冲突
  @Prop buttonSize: ButtonSize = 'default';
  @Prop loading: boolean = false;
  @Prop disabled: boolean = false;
  @Prop fullWidth: boolean = false;
  onPress?: () => void;

  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'default_clean_light';
  @State isPressed: boolean = false;

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  private getHeight(): number {
    if (this.buttonSize === 'small') return 36;
    if (this.buttonSize === 'large') return 48;
    return DesignConstants.COMPONENT.BUTTON_HEIGHT; // 42
  }

  private getFontSize(): number {
    if (this.buttonSize === 'small') return DesignConstants.FONT.SIZE.SM; // 12
    if (this.buttonSize === 'large') return DesignConstants.FONT.SIZE.MD_PLUS; // 15
    return DesignConstants.FONT.SIZE.SM_PLUS; // 13
  }

  private getIconSize(): number {
    return this.getFontSize() + 3;
  }

  private getHorizontalPadding(): number {
    if (this.buttonSize === 'small') return DesignConstants.SPACING.MD; // 12
    if (this.buttonSize === 'large') return DesignConstants.SPACING.XXL; // 32
    return DesignConstants.SPACING.XL; // 24
  }

  private getBackgroundColor(): ResourceColor {
    const p = this.getPalette();
    switch (this.variant) {
      case 'primary':
        return p.accentBackground;
      case 'secondary':
        return p.buttonSecondaryBackground;
      case 'outline':
        return DesignConstants.COLOR.TRANSPARENT;
      case 'ghost':
        return DesignConstants.COLOR.TRANSPARENT;
      case 'danger':
        return p.error;
      default:
        return p.accentBackground;
    }
  }

  private getTextColor(): ResourceColor {
    const p = this.getPalette();
    switch (this.variant) {
      case 'primary':
        return p.accentText; // --text-on-accent
      case 'danger':
        return p.accentText;
      case 'secondary':
        return p.buttonSecondaryText;
      case 'outline':
        return p.accentBackground;
      case 'ghost':
        return p.textPrimary;
      default:
        return p.accentText;
    }
  }

  private getBorderWidth(): number {
    return this.variant === 'outline' ? 1.5 : 0;
  }

  private getBorderColor(): ResourceColor {
    const p = this.getPalette();
    if (this.variant === 'outline') {
      return p.accentBackground;
    }
    return DesignConstants.COLOR.TRANSPARENT;
  }

  private getShadow(): ShadowOptions | undefined {
    const p = this.getPalette();
    // 扁平风格：仅 primary / danger 带品牌阴影（对齐前端 --shadow-brand-btn）
    if (this.variant === 'primary' || this.variant === 'danger') {
      return p.accentShadow;
    }
    return undefined;
  }

  private isInteractive(): boolean {
    return !this.disabled && !this.loading;
  }

  build() {
    Button() {
      Row() {
        if (this.loading) {
          LoadingProgress()
            .width(this.getIconSize())
            .height(this.getIconSize())
            .color(this.getTextColor() as ResourceColor)
            .margin({ right: this.showIcon || this.label.length === 0 ? 0 : DesignConstants.SPACING.XS })
        } else if (this.showIcon) {
          Text() {
            SymbolSpan($r(this.icon))
              .fontSize(this.getIconSize())
              .fontColor([this.getTextColor()])
          }
          .margin({ right: this.label.length === 0 ? 0 : DesignConstants.SPACING.XS })
        }
        if (this.label.length > 0) {
          Text(this.label)
            .fontSize(this.getFontSize())
            .fontWeight(FontWeight.Medium)
            .fontColor(this.getTextColor())
            .maxLines(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
        }
      }
      .justifyContent(FlexAlign.Center)
      .alignItems(VerticalAlign.Center)
    }
    .type(ButtonType.Normal)
    .stateEffect(false)
    .height(this.getHeight())
    .width(this.fullWidth ? '100%' : undefined)
    .padding({ left: this.getHorizontalPadding(), right: this.getHorizontalPadding() })
    .backgroundColor(this.getBackgroundColor())
    .borderRadius(DesignConstants.SHAPE.BUTTON) // 28
    .border({ width: this.getBorderWidth(), color: this.getBorderColor() })
    .shadow(this.getShadow())
    .enabled(this.isInteractive())
    .opacity(this.disabled ? DesignConstants.OPACITY.DISABLED : 1)
    .scale(this.isPressed
      ? { x: DesignConstants.INTERACTION.SCALE_PRESSED_BUTTON, y: DesignConstants.INTERACTION.SCALE_PRESSED_BUTTON }
      : { x: 1, y: 1 })
    .animation({
      curve: curves.springMotion(
        DesignConstants.INTERACTION.SPRING_RESPONSE_PRESS,
        DesignConstants.INTERACTION.SPRING_DAMPING_PRESS
      )
    })
    .onTouch((event: TouchEvent) => {
      if (!this.isInteractive()) {
        return;
      }
      if (event.type === TouchType.Down) {
        this.isPressed = true;
      } else if (event.type === TouchType.Up || event.type === TouchType.Cancel) {
        this.isPressed = false;
      }
    })
    .onClick(() => {
      if (!this.isInteractive()) {
        return;
      }
      this.onPress?.();
    })
  }
}
