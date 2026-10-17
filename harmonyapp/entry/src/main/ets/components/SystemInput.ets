import { DesignConstants } from '../constants/DesignConstants';
import { AppThemeManager } from '../core/theme/AppThemeManager';
import { DEFAULT_RESOLVED_THEME_ID, KEY_RESOLVED_THEME_ID } from '../core/theme/AppThemeTypes';
import { AppThemeVisuals, InputBorderResolution } from '../core/theme/AppThemeVisuals';
import { AppIcon } from './SystemIcon';
import { LucideIcon } from '../components/LucideIcon';

/**
 * 通用单行输入框（对齐前端 input-surface）
 * inputBackgroundColor 底 + 16 圆角（SHAPE.INPUT）+ 可选前置图标。
 * 受控组件：父组件维护 value，通过 onChange 回写。
 */
@Reusable
@Component
export struct SystemInput {
  @Prop value: string = '';
  @Prop placeholder: string = '';
  @Prop prefixIcon: AppIcon = AppIcon.SEARCH;
  @Prop showPrefixIcon: boolean = false;
  @Prop passwordMode: boolean = false;
  /** 输入类型（默认普通；email/number 对齐登录与数字场景） */
  @Prop inputType: 'normal' | 'email' | 'number' = 'normal';
  @Prop maxLength: number = 1000;
  @Prop disabled: boolean = false;
  /** 输入框高度（不可命名为 height——与 CustomComponent 基类方法冲突，同 SystemButton.buttonSize） */
  @Prop inputHeight: number = DesignConstants.COMPONENT.INPUT_HEIGHT; // 42
  @Prop fontSize: number = DesignConstants.FONT.SIZE.SM; // 12
  /** 错误态：边框变 error 色 */
  @Prop hasError: boolean = false;
  /** 右侧清除按钮（value 非空时显示，对齐前端 moments 搜索栏的 X） */
  @Prop showClearButton: boolean = false;
  /** 清除按钮图标尺寸 */
  @Prop clearIconSize: number = 13;
  onChange?: (value: string) => void;
  onSubmit?: (value: string) => void;
  onFocusChange?: (focused: boolean) => void;

  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = DEFAULT_RESOLVED_THEME_ID;
  @State isFocused: boolean = false;

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  private getBorderResolution(): InputBorderResolution {
    return AppThemeVisuals.resolveInputBorder({
      backgroundColor: this.getPalette().inputBackgroundColor,
      borderColor: this.getPalette().borderColor,
      borderWidth: DesignConstants.BORDER_WIDTH.HAIRLINE,
      borderRadius: DesignConstants.SHAPE.INPUT,
      shadow: { radius: 0, color: DesignConstants.COLOR.TRANSPARENT, offsetX: 0, offsetY: 0 },
      height: this.inputHeight,
      horizontalPadding: DesignConstants.SPACING.LG
    }, this.isFocused, this.hasError, this.resolvedThemeId);
  }

  private getInputType(): InputType {
    if (this.passwordMode) return InputType.Password;
    if (this.inputType === 'email') return InputType.Email;
    if (this.inputType === 'number') return InputType.Number;
    return InputType.Normal;
  }

  /** 无障碍字段名：优先 placeholder（本仓输入框以 placeholder 承载字段名），
   *  否则退化为通用「输入框」。 */
  private getFieldLabel(): string {
    return this.placeholder.length > 0 ? this.placeholder : '输入框';
  }

  build() {
    Row() {
      if (this.showPrefixIcon) {
        LucideIcon({
            icon: this.prefixIcon,
            iconSize: this.fontSize + 2,
            color: this.getPalette().textSecondary
          })
        // 对齐前端 SearchInput 的 gap-2.5(10)：图标与输入文字间距 10vp（实测前端 33→57 即 10）
        .margin({ right: DesignConstants.SPACING.MD2 })
      }

      TextInput({ placeholder: this.placeholder, text: this.value })
        .layoutWeight(1)
        // 完全透明化：消除 TextInput 系统主题默认背景/圆角（否则会溢出外层
        // 圆角容器露出直角矩形，对齐前端 input 无自身视觉层）
        .backgroundColor(DesignConstants.COLOR.TRANSPARENT)
        .background(DesignConstants.COLOR.TRANSPARENT)
        .border({ width: 0 })
        .borderRadius(0)
        .padding({ top: 0, bottom: 0, left: 0, right: 0 })
        .fontColor(this.getPalette().textPrimary)
        .fontSize(this.fontSize)
        .placeholderColor(this.getPalette().textSecondary)
        .placeholderFont({ size: this.fontSize })
        .caretColor(this.getPalette().accentBackground)
        .enabled(!this.disabled)
        // 禁用时从无障碍树移除（对齐原生 disabled 语义）
        // 字段名 + 错误态播报：placeholder 不等于 label（屏幕阅读器不一定播），
        // 错误态需可听（对齐前端 aria-invalid + role="alert" 的就近错误行契约）
        .accessibilityText(this.hasError ? `${this.getFieldLabel()}，输入有误` : this.getFieldLabel())
        .accessibilityLevel(this.disabled ? 'no' : 'auto')
    .opacity(this.disabled ? DesignConstants.OPACITY.DISABLED : 1)
        .type(this.getInputType())
        .maxLength(this.maxLength)
        .onChange((v: string) => {
          this.onChange?.(v);
        })
        .onSubmit(() => {
          this.onSubmit?.(this.value);
        })
        .onFocus(() => {
          this.isFocused = true;
          this.onFocusChange?.(true);
        })
        .onBlur(() => {
          this.isFocused = false;
          this.onFocusChange?.(false);
        })

      if (this.showClearButton && this.value.length > 0) {
        // 清除钮用 MIN_HIT_TARGET(44) 命中区包裹小图标：触控达标 ≥44vp（WCAG 2.5.5），视觉尺寸不变
        Row() {
          LucideIcon({
              icon: AppIcon.CLOSE,
              iconSize: this.clearIconSize,
              color: this.getPalette().textSecondary
            })
        }
        .width(DesignConstants.LAYOUT.MIN_HIT_TARGET)
        .height(DesignConstants.LAYOUT.MIN_HIT_TARGET)
        .justifyContent(FlexAlign.Center)
        .alignItems(VerticalAlign.Center)
        .onClick(() => {
          this.onChange?.('');
        })
      }
    }
    .height(this.inputHeight)
    .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG })
    .backgroundColor(this.getPalette().inputBackgroundColor)
    .borderRadius(DesignConstants.SHAPE.INPUT) // 16
    .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getBorderResolution().borderColor })
    .shadow(this.getBorderResolution().shadow)
    // 裁剪子组件（TextInput 等），防止系统默认视觉溢出圆角
    .clip(true)
  }
}
