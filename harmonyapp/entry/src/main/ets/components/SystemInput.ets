import { DesignConstants } from '../constants/DesignConstants';
import { AppThemeManager } from '../core/theme/AppThemeManager';
import { KEY_RESOLVED_THEME_ID } from '../core/theme/AppThemeTypes';
import { AppIcon } from './SystemIcon';

/**
 * 通用单行输入框（对齐前端 input-surface）
 * inputBackgroundColor 底 + 18 圆角（SHAPE.INPUT）+ 可选前置图标。
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
  @Prop height: number = DesignConstants.COMPONENT.INPUT_HEIGHT; // 42
  @Prop fontSize: number = DesignConstants.FONT.SIZE.SM_PLUS; // 13
  /** 错误态：边框变 error 色 */
  @Prop hasError: boolean = false;
  /** 右侧清除按钮（value 非空时显示，对齐前端 moments 搜索栏的 X） */
  @Prop showClearButton: boolean = false;
  /** 清除按钮图标尺寸 */
  @Prop clearIconSize: number = 13;
  onChange?: (value: string) => void;
  onSubmit?: (value: string) => void;

  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'default_clean_light';

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  build() {
    Row() {
      if (this.showPrefixIcon) {
        Text() {
          SymbolSpan($r(this.prefixIcon))
            .fontSize(this.fontSize + 2)
            .fontColor([this.getPalette().textSecondary])
        }
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
        .type(this.passwordMode ? InputType.Password : InputType.Normal)
        .onChange((v: string) => {
          this.onChange?.(v);
        })
        .onSubmit((v: string) => {
          this.onSubmit?.(v);
        })

      if (this.showClearButton && this.value.length > 0) {
        Text() {
          SymbolSpan($r(AppIcon.CLOSE))
            .fontSize(this.clearIconSize)
            .fontColor([this.getPalette().textSecondary])
        }
        .margin({ left: DesignConstants.SPACING.SM })
        .onClick(() => {
          this.onChange?.('');
        })
      }
    }
    .height(this.height)
    .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG })
    .backgroundColor(this.getPalette().inputBackgroundColor)
    .borderRadius(DesignConstants.SHAPE.INPUT) // 18
    .border({
      width: 1,
      color: this.hasError ? this.getPalette().error : this.getPalette().borderColor
    })
  }
}
