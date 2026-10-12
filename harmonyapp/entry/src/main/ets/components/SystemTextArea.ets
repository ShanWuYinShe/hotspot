import { DesignConstants } from '../constants/DesignConstants';
import { AppThemeManager } from '../core/theme/AppThemeManager';
import { DEFAULT_RESOLVED_THEME_ID, KEY_RESOLVED_THEME_ID } from '../core/theme/AppThemeTypes';

/**
 * 通用多行输入框（对齐前端 textarea）
 * inputBackgroundColor 底 + 16 圆角，自适应高度。
 * 受控组件：父组件维护 value，通过 onChange 回写。
 */
@Reusable
@Component
export struct SystemTextArea {
  @Prop value: string = '';
  @Prop placeholder: string = '';
  @Prop minHeight: number = 96;
  @Prop fontSize: number = DesignConstants.FONT.SIZE.SM; // 12
  /** 最大行数（约束高度），0 表示不限 */
  @Prop maxLines: number = 0;
  @Prop maxLength: number = 1000;
  @Prop hasError: boolean = false;
  onChange?: (value: string) => void;
  onFocusChange?: (focused: boolean) => void;

  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = DEFAULT_RESOLVED_THEME_ID;
  /** 聚焦态：边框 accent 化（对齐 SystemInput focus 契约） */
  @State isFocused: boolean = false;

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  build() {
    TextArea({
      placeholder: this.placeholder,
      text: this.value
    })
      .width('100%')
      .constraintSize({ minHeight: this.minHeight })
      .padding(DesignConstants.SPACING.MD)
      .backgroundColor(this.getPalette().inputBackgroundColor)
      .borderRadius(DesignConstants.SHAPE.INPUT) // 16
      .border({
        width: this.isFocused ? DesignConstants.BORDER_WIDTH.REGULAR : DesignConstants.BORDER_WIDTH.HAIRLINE,
        color: this.hasError ? this.getPalette().error :
          (this.isFocused ? this.getPalette().accentBackground : this.getPalette().borderColor)
      })
      .fontColor(this.getPalette().textPrimary)
      .fontSize(this.fontSize)
      .placeholderColor(this.getPalette().textSecondary)
      .placeholderFont({ size: this.fontSize })
      .caretColor(this.getPalette().accentBackground)
      .maxLines(this.maxLines > 0 ? this.maxLines : Number.MAX_SAFE_INTEGER)
      .maxLength(this.maxLength)
      .onChange((v: string) => {
        this.onChange?.(v);
      })
      .onFocus(() => {
        this.isFocused = true;
        this.onFocusChange?.(true);
      })
      .onBlur(() => {
        this.isFocused = false;
        this.onFocusChange?.(false);
      })
  }
}
