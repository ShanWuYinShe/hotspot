import { DesignConstants } from '../constants/DesignConstants';
import { AppThemeManager } from '../core/theme/AppThemeManager';
import { KEY_RESOLVED_THEME_ID } from '../core/theme/AppThemeTypes';
import { AppIcon } from './SystemIcon';
import { curves } from '@kit.ArkUI';

/**
 * 返回按钮（对齐前端 back-button.svelte）
 * 胶囊形 chevron-left，press 缩放 0.88（对齐前端 active:scale-[0.88]）。
 * 可选文字标签（默认纯图标）；默认 textSecondary 色。
 */
@Reusable
@Component
export struct BackButton {
  /** 按钮高度（圆形/胶囊宽度随内容） */
  @Prop buttonSize: number = 38;
  @Prop glyphSize: number = 20;
  /** 可选文字标签（如"返回"），默认纯图标 */
  @Prop label: string = '';
  onBack?: () => void;

  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'default_clean_light';
  @State isPressed: boolean = false;

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  build() {
    Row() {
      Text() {
        SymbolSpan($r(AppIcon.BACK))
          .fontSize(this.glyphSize)
          .fontColor([this.getPalette().textSecondary])
      }
      if (this.label.length > 0) {
        Text(this.label)
          .fontSize(DesignConstants.FONT.SIZE.MD_PLUS) // 15
          .fontColor(this.getPalette().textPrimary)
          .margin({ left: 2 })
      }
    }
    .height(this.buttonSize)
    .padding({ left: DesignConstants.SPACING.SM, right: DesignConstants.SPACING.MD })
    .borderRadius(DesignConstants.RADIUS.FULL)
    .justifyContent(FlexAlign.Center)
    .alignItems(VerticalAlign.Center)
    .scale(this.isPressed
      ? { x: 0.88, y: 0.88 }
      : { x: 1, y: 1 })
    .animation({ curve: curves.springMotion(0.3, 0.85) })
    .onTouch((event: TouchEvent) => {
      if (event.type === TouchType.Down) {
        this.isPressed = true;
      } else if (event.type === TouchType.Up || event.type === TouchType.Cancel) {
        this.isPressed = false;
      }
    })
    .onClick(() => {
      this.onBack?.();
    })
  }
}
