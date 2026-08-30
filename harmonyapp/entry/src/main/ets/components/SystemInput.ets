import { DesignConstants } from '../constants/DesignConstants';
import { AppThemeManager } from '../core/theme/AppThemeManager';
import { KEY_RESOLVED_THEME_ID } from '../core/theme/AppThemeTypes';
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

  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'vibrant_light';
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

  build() {
    Row() {
      if (this.showPrefixIcon) {
        LucideIcon({
            icon: this.prefixIcon,
            size: this.fontSize + 2,
            color: this.getPalette().textSecondary
          })
        .margin({ right: DesignConstants.SPACING.SM })
      }

      TextInput({ placeholder: this.placeholder, text: this.value })
        .layoutWeight(1)
        .backgroundColor(DesignConstants.COLOR.TRANSPARENT)
        .padding({ top: 0, bottom: 0, left: 0, right: 0 })
        .fontColor(this.getPalette().textPrimary)
        .fontSize(this.fontSize)
        .placeholderColor(this.getPalette().textSecondary)
        .placeholderFont({ size: this.fontSize })
        .caretColor(this.getPalette().accentBackground)
        .enabled(!this.disabled)
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
        // 清除钮用 ICON_BUTTON_SIZE(36) 命中区包裹小图标：触控达标 ≥36vp，视觉尺寸不变
        Row() {
          LucideIcon({
              icon: AppIcon.CLOSE,
              size: this.clearIconSize,
              color: this.getPalette().textSecondary
            })
        }
        .width(DesignConstants.LAYOUT.ICON_BUTTON_SIZE)
        .height(DesignConstants.LAYOUT.ICON_BUTTON_SIZE)
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
  }
}
