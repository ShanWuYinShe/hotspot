import { DesignConstants } from '../constants/DesignConstants';
import { AppThemeManager } from '../core/theme/AppThemeManager';
import { KEY_RESOLVED_THEME_ID } from '../core/theme/AppThemeTypes';
import { KEY_BROWSE_ANCHOR } from '../core/state/GlobalStateManager';
import { LocationInfo } from '../core/location/LocationUtil';
import { DefaultConfig, PageNames } from '../constants/AppConstants';
import { AppRouter } from '../core/router/AppRouter';
import { AppIcon } from './SystemIcon';
import { curves } from '@kit.ArkUI';

/**
 * 浏览锚点入口（对齐前端 anchor-entry.svelte）
 * 位置与半径调整的唯一入口；点击打开 LocationPickerPage。
 *
 * variant：
 * - 'card'：桌面侧边栏底部的完整卡片块（图标圆 + 锚点标签 + 位置名 + 半径胶囊 + chevron）
 * - 'pill'：移动 / 平板头部或概览行的紧凑 chip
 */
export type AnchorEntryVariant = 'card' | 'pill';

@Reusable
@Component
export struct AnchorEntry {
  @Prop variant: AnchorEntryVariant = 'pill';
  @Prop labelOverride: string = '';
  /** 半径（公里），<=0 表示用默认 NEARBY_RADIUS */
  @Prop radiusKm: number = 0;
  /** 自定义点击回调；不传则内部跳 LocationPickerPage */
  onAdjust?: () => void;

  @StorageLink(KEY_BROWSE_ANCHOR) browseAnchor: LocationInfo | null = null;
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'default_clean_light';
  @State isPressed: boolean = false;

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  private getDisplayName(): string {
    return this.browseAnchor?.locationName || '当前位置';
  }

  private getRadiusText(): string {
    const km = this.radiusKm > 0 ? this.radiusKm : DefaultConfig.NEARBY_RADIUS / 1000;
    return `${km}km`;
  }

  private getLabel(): string {
    return this.labelOverride.length > 0 ? this.labelOverride : '浏览锚点';
  }

  private handleClick(): void {
    if (this.onAdjust) {
      this.onAdjust();
    } else {
      AppRouter.push(PageNames.LOCATION_PICKER);
    }
  }

  build() {
    if (this.variant === 'card') {
      this.CardVariant()
    } else {
      this.PillVariant()
    }
  }

  @Builder
  CardVariant() {
    Row() {
      Row() {
        Text() {
          SymbolSpan($r(AppIcon.NEARBY))
            .fontSize(14)
            .fontColor([this.getPalette().accentText])
        }
      }
      .width(34)
      .height(34)
      .borderRadius(DesignConstants.RADIUS.FULL)
      .backgroundColor(this.getPalette().accentBackground)
      .justifyContent(FlexAlign.Center)

      Column() {
        Text(this.getLabel())
          .fontSize(DesignConstants.FONT.SIZE.XS2) // 10
          .fontColor(this.getPalette().textSecondary)
          .letterSpacing(0.6)
        Row() {
          Text(this.getDisplayName())
            .fontSize(DesignConstants.FONT.SIZE.SM_PLUS) // 13
            .fontColor(this.getPalette().textPrimary)
            .maxLines(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
            .layoutWeight(1)
          Text(this.getRadiusText())
            .fontSize(DesignConstants.FONT.SIZE.XS2) // 10
            .fontColor(this.getPalette().accentBackground)
            .padding({ left: DesignConstants.SPACING.SM2, right: DesignConstants.SPACING.SM2 })
            .borderRadius(DesignConstants.RADIUS.FULL)
            .backgroundColor(this.getPalette().accentMuted)
            .margin({ left: DesignConstants.SPACING.XS })
        }
        .width('100%')
        .margin({ top: 2 })
      }
      .layoutWeight(1)
      .margin({ left: DesignConstants.SPACING.SM })
      .alignItems(HorizontalAlign.Start)

      Text() {
        SymbolSpan($r(AppIcon.ARROW_RIGHT))
          .fontSize(14)
          .fontColor([this.getPalette().textSecondary])
      }
    }
    .width('100%')
    .padding({ left: DesignConstants.SPACING.MD, right: DesignConstants.SPACING.MD, top: DesignConstants.SPACING.MD2, bottom: DesignConstants.SPACING.MD2 })
    .backgroundColor(this.getPalette().surfaceBackground)
    .borderRadius(DesignConstants.SHAPE.INPUT) // 18
    .border({ width: 1, color: this.getPalette().borderColor })
    .alignItems(VerticalAlign.Center)
    .scale(this.isPressed ? { x: 0.98, y: 0.98 } : { x: 1, y: 1 }) // 对齐前端输入行按压强度
    .animation({ curve: curves.springMotion(0.3, 0.85) })
    .onTouch((event: TouchEvent) => {
      if (event.type === TouchType.Down) {
        this.isPressed = true;
      } else if (event.type === TouchType.Up || event.type === TouchType.Cancel) {
        this.isPressed = false;
      }
    })
    .onClick(() => this.handleClick())
  }

  @Builder
  PillVariant() {
    Row() {
      Text() {
        SymbolSpan($r(AppIcon.LOCATION))
          .fontSize(12)
          .fontColor([this.getPalette().accentBackground])
      }
      Text(this.getDisplayName())
        .fontSize(DesignConstants.FONT.SIZE.XS) // 11
        .fontColor(this.getPalette().textSecondary)
        .maxLines(1)
        .textOverflow({ overflow: TextOverflow.Ellipsis })
        .constraintSize({ maxWidth: '55%' })
        .margin({ left: 6 })
      Text(`· ${this.getRadiusText()}`)
        .fontSize(DesignConstants.FONT.SIZE.XS) // 11
        .fontColor(this.getPalette().accentBackground)
    }
    .padding({ left: DesignConstants.SPACING.MD, right: DesignConstants.SPACING.MD, top: DesignConstants.SPACING.SM2, bottom: DesignConstants.SPACING.SM2 })
    .backgroundColor(this.getPalette().surfaceBackground)
    .border({ width: 1, color: this.getPalette().borderColor })
    .borderRadius(DesignConstants.RADIUS.FULL)
    .alignItems(VerticalAlign.Center)
    .scale(this.isPressed ? { x: 0.94, y: 0.94 } : { x: 1, y: 1 }) // 对齐前端 pill 按钮按压强度
    .animation({ curve: curves.springMotion(0.3, 0.85) })
    .onTouch((event: TouchEvent) => {
      if (event.type === TouchType.Down) {
        this.isPressed = true;
      } else if (event.type === TouchType.Up || event.type === TouchType.Cancel) {
        this.isPressed = false;
      }
    })
    .onClick(() => this.handleClick())
  }
}
