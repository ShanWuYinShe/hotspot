import { DesignConstants } from '../constants/DesignConstants';
import { AppThemeManager } from '../core/theme/AppThemeManager';
import { KEY_RESOLVED_THEME_ID } from '../core/theme/AppThemeTypes';

/**
 * 角标组件（对齐前端 unread badge）
 * - count > 0：数字徽章（> maxCount 显示 "N+"），accentBackground 底 + accentText 字
 * - count <= 0 且 dotOnly=true：纯小红点
 * - count <= 0 且 dotOnly=false：不渲染
 */
@Reusable
@Component
export struct SystemBadge {
  @Prop count: number = 0;
  @Prop maxCount: number = 99;
  /** count<=0 时是否降级为红点 */
  @Prop dotOnly: boolean = false;

  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'vibrant_light';

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  private getDisplayText(): string {
    if (this.count > this.maxCount) {
      return `${this.maxCount}+`;
    }
    return String(this.count);
  }

  build() {
    if (this.count > 0) {
      // 对齐前端徽标契约（22cbd552）：紧凑徽标 h-5 px-2 text-xxs + leading-none，
      // 行高锁字号防文字下沉，minFontScale 保数字溢出可读
      Text(this.getDisplayText())
        .fontSize(DesignConstants.FONT.SIZE.XS2)
        .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
        .fontColor(this.getPalette().accentText)
        .backgroundColor(this.getPalette().accentBackground)
        .borderRadius(DesignConstants.RADIUS.FULL)
        .padding({ left: DesignConstants.SPACING.SM2, right: DesignConstants.SPACING.SM2, top: 1, bottom: 1 })
        .lineHeight(DesignConstants.FONT.SIZE.XS2) // leading-none：行高锁字号
        .minFontScale(0.8)
        .maxLines(1)
    } else if (this.dotOnly) {
      Row()
        .width(DesignConstants.SPACING.SM)
        .height(DesignConstants.SPACING.SM)
        .borderRadius(DesignConstants.RADIUS.FULL)
        .backgroundColor(this.getPalette().accentBackground)
    }
  }
}
