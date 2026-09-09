import { DesignConstants } from '../constants/DesignConstants';
import { AppThemeManager } from '../core/theme/AppThemeManager';
import { KEY_RESOLVED_THEME_ID } from '../core/theme/AppThemeTypes';
import { AppIcon } from './SystemIcon';
import { curves } from '@kit.ArkUI';
import { LucideIcon } from './LucideIcon';

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
 * 尺寸（size）：compact(28，卡片内小按钮) / small(36) / default(42) / large(48) /
 *   sheet(44，底部弹层操作行)
 * 变体补充：surface（subtle 底 + 主文字 + 细边框，对齐 Sheet 操作行）、
 *   surface-danger（同上但文字为 error 色，用于举报/危险操作）
 *
 * 交互：press 缩放 0.95（对齐前端 active:scale-[0.95]），springMotion 回弹
 */
export type ButtonVariant = 'primary' | 'secondary' | 'outline' | 'ghost' | 'danger' | 'surface' | 'surface-accent' | 'surface-danger';
export type ButtonSize = 'compact' | 'small' | 'default' | 'large' | 'sheet';

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
  /** surface/surface-danger 变体是否带控制阴影（卡片内嵌小按钮用，弹层操作行不带） */
  @Prop shadowed: boolean = false;
  /** 强制胶囊圆角（主操作 42vp 胶囊等场景） */
  @Prop pill: boolean = false;
  onPress?: () => void;

  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'vibrant_light';
  @State isPressed: boolean = false;
  /** hover（仅键鼠外接触发；onHover 在触屏设备不会触发） */
  @State isHovered: boolean = false;

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  private getHeight(): number {
    if (this.buttonSize === 'compact') return DesignConstants.COMPONENT.ACTION_BUTTON_HEIGHT; // 28
    if (this.buttonSize === 'sheet') return DesignConstants.COMPONENT.SHEET_ACTION_HEIGHT; // 44
    if (this.buttonSize === 'small') return DesignConstants.COMPONENT.BUTTON_HEIGHT_SM; // 36
    if (this.buttonSize === 'large') return DesignConstants.COMPONENT.BUTTON_HEIGHT_LG; // 48
    return DesignConstants.COMPONENT.BUTTON_HEIGHT; // 42
  }

  private getFontSize(): number {
    if (this.buttonSize === 'compact') return DesignConstants.FONT.SIZE.XS2; // 10
    if (this.buttonSize === 'sheet') return DesignConstants.FONT.SIZE.MD; // 14
    if (this.buttonSize === 'small') return DesignConstants.FONT.SIZE.SM; // 12
    if (this.buttonSize === 'large') return DesignConstants.FONT.SIZE.MD; // 14
    return DesignConstants.FONT.SIZE.SM; // 12
  }

  private getIconSize(): number {
    if (this.buttonSize === 'compact') return DesignConstants.COMPONENT.ICON_SIZE_XS; // 16
    if (this.buttonSize === 'sheet' || this.buttonSize === 'large') return DesignConstants.COMPONENT.ICON_SIZE_MD; // 24
    return DesignConstants.COMPONENT.ICON_SIZE_SM; // 20
  }

  private getHorizontalPadding(): number {
    if (this.buttonSize === 'compact') return DesignConstants.SPACING.SM; // 8
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
      case 'surface':
      case 'surface-accent':
      case 'surface-danger':
        return p.surfaceSubtle;
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
        // danger 实底文字：恒定白色（vibrant_light error #C81E4E 配白字 5.60 ≥ 4.5）
        return DesignConstants.COLOR.WHITE;
      case 'secondary':
        return p.buttonSecondaryText;
      case 'outline':
        return p.accentBackground;
      case 'ghost':
        return p.textPrimary;
      case 'surface':
        return p.textPrimary;
      case 'surface-accent':
        return p.accentBackground;
      case 'surface-danger':
        return p.error;
      default:
        return p.accentText;
    }
  }

  private getBorderWidth(): number {
    if (this.variant === 'surface' || this.variant === 'surface-accent' || this.variant === 'surface-danger') {
      return DesignConstants.BORDER_WIDTH.HAIRLINE;
    }
    return this.variant === 'outline' ? DesignConstants.BORDER_WIDTH.REGULAR : 0;
  }

  private getBorderColor(): ResourceColor {
    const p = this.getPalette();
    if (this.variant === 'outline') {
      return p.accentBackground;
    }
    if (this.variant === 'surface') {
      return p.borderColor;
    }
    if (this.variant === 'surface-accent') {
      return p.floatingBorderColor;
    }
    if (this.variant === 'surface-danger') {
      return p.borderColor;
    }
    return DesignConstants.COLOR.TRANSPARENT;
  }

  private getRadius(): number {
    if (this.pill || this.buttonSize === 'compact' || this.buttonSize === 'sheet' ||
      this.variant === 'surface' || this.variant === 'surface-accent' || this.variant === 'surface-danger') {
      return DesignConstants.RADIUS.FULL;
    }
    return DesignConstants.SHAPE.BUTTON;
  }

  private getShadow(): ShadowOptions {
    const p = this.getPalette();
    // 对齐前端 ui/button.svelte 当前契约：全部 variant 用 raised 双向投影
    // （raised-sm 静态 / raised-md hover 抬档），无品牌光晕（前端已删除 accent glow）；
    // 无阴影时返回零阴影而非 undefined（undefined 实参会破坏属性链导致直角）
    if (this.variant === 'primary' || this.variant === 'danger') {
      if (this.isHovered && this.isInteractive()) {
        return p.elevatedShadow;
      }
      return p.surfaceShadow;
    }
    if (this.shadowed) {
      return p.controlShadow;
    }
    const none: ShadowOptions = { radius: 0, color: DesignConstants.COLOR.TRANSPARENT, offsetX: 0, offsetY: 0 };
    return none;
  }

  /** hover 背景/边框（键鼠）：透明变体（ghost/outline）hover = accentMuted 淡填充 + 边框提亮 */
  private getHoverOverlayBackground(): ResourceColor {
    const p = this.getPalette();
    const transparentVariants = ['ghost', 'outline'];
    if (this.isHovered && this.isInteractive() && transparentVariants.indexOf(this.variant) >= 0) {
      return p.accentMuted;
    }
    return this.getBackgroundColor();
  }

  private getHoverBorderColor(): ResourceColor {
    const p = this.getPalette();
    if (this.isHovered && this.isInteractive() && this.variant === 'ghost') {
      return p.floatingBorderColor;
    }
    return this.getBorderColor();
  }

  private isInteractive(): boolean {
    return !this.disabled && !this.loading;
  }

  /** compact(28)/small(36) 视觉高度的垂直热区扩展到 44vp */
  private getResponseRegion(): Rectangle {
    const height = this.getHeight();
    const extend = Math.max(0, DesignConstants.LAYOUT.MIN_HIT_TARGET - height) / 2;
    return {
      x: 0,
      y: -extend,
      width: '100%',
      height: height + extend * 2
    };
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
          LucideIcon({
            icon: this.icon,
            iconSize: this.getIconSize(),
            color: this.getTextColor()
          })
          .margin({ right: this.label.length === 0 ? 0 : DesignConstants.SPACING.XS })
        }
        if (this.label.length > 0) {
          Text(this.label)
            .fontSize(this.getFontSize())
            .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
            .fontColor(this.getTextColor())
            .maxLines(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
        }
      }
      .justifyContent(FlexAlign.Center)
      .alignItems(VerticalAlign.Center)
    }
    // Capsule：该 ArkUI 版本 ButtonType.Normal 会忽略 borderRadius 渲染直角；
    // 全部 radius 档均为 FULL，胶囊等价
    .type(ButtonType.Capsule)
    .stateEffect(false)
    .height(this.getHeight())
    .width(this.fullWidth ? '100%' : 'auto')
    .padding({ left: this.getHorizontalPadding(), right: this.getHorizontalPadding() })
    .backgroundColor(this.getHoverOverlayBackground())
    .borderRadius(this.getRadius())
    .border({ width: this.getBorderWidth(), color: this.getHoverBorderColor() })
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
    .onHover((isHover: boolean) => {
      if (this.isInteractive()) {
        this.isHovered = isHover;
      }
    })
    // 小尺寸按钮触控热区：垂直方向外扩到 LAYOUT.MIN_HIT_TARGET（responseRegion
    // 不影响布局与视觉；水平方向因 auto 宽度不可计算，保持原生命中区）
    .responseRegion(this.getResponseRegion())
    // 键盘焦点环（stateEffect 已关，系统焦点不显示；Tab 导航需要可感知焦点）
    .onFocus(() => {
      if (this.isInteractive()) {
        this.isHovered = true;
      }
    })
    .onBlur(() => {
      this.isHovered = false;
    })
  }
}
