/**
 * LocationResultRow —— 地点选择器搜索结果行（定位图标 + 地点名，tint 行 +
 * 按压反馈）。从 LocationPickerPage 抽取；按压态组件自持，选中回调上抛。
 */

import { curves } from '@kit.ArkUI';
import { DesignConstants } from '../../../constants/DesignConstants';
import { AppIcon } from '../../../components/SystemIcon';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { AppThemePalette } from '../../../core/theme/AppThemeTypes';
import { LocationInfo } from '../../../core/location/LocationUtil';
import { LucideIcon } from '../../../components/LucideIcon';

@Component
export struct LocationResultRow {
  @Prop item: LocationInfo;
  @Prop resolvedThemeId: string = 'vibrant_light';
  @State pressed: boolean = false;
  onSelect: (result: LocationInfo) => void = () => {};

  private palette(): AppThemePalette {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  build() {
    Row({ space: DesignConstants.SPACING.MD }) {
      Column() {
        LucideIcon({
            icon: AppIcon.LOCATION,
            size: DesignConstants.FONT.SIZE.LG_PLUS,
            color: this.palette().accentBackground
          })
      }
      .width(DesignConstants.COMPONENT.AVATAR_SIZE.XS)
      .height(DesignConstants.COMPONENT.AVATAR_SIZE.XS)
      .justifyContent(FlexAlign.Center)
      .backgroundColor(this.palette().surfaceSubtle)
      .borderRadius(DesignConstants.RADIUS.MD)

      Text(this.item.locationName || '未知地点')
        .fontSize(DesignConstants.FONT.SIZE.MD)
        .fontColor(this.palette().textPrimary)
        .layoutWeight(1)
        .maxLines(2)
        .textOverflow({ overflow: TextOverflow.Ellipsis })
    }
    .width('100%')
    .padding({
      left: DesignConstants.SPACING.MD,
      right: DesignConstants.SPACING.MD,
      top: DesignConstants.SPACING.MD,
      bottom: DesignConstants.SPACING.MD
    })
    // 面板内行去三件套（surface 叠 surface + 边框 + 阴影），改 tint 行 + 按压反馈
    .backgroundColor(this.palette().surfaceSubtle)
    .borderRadius(DesignConstants.RADIUS.LG)
    .scale(this.pressed ?
      { x: DesignConstants.INTERACTION.SCALE_PRESSED_CARD, y: DesignConstants.INTERACTION.SCALE_PRESSED_CARD } :
      { x: 1, y: 1 })
    .animation({ curve: curves.springMotion(
      DesignConstants.INTERACTION.SPRING_RESPONSE_PRESS,
      DesignConstants.INTERACTION.SPRING_DAMPING_PRESS
    ) })
    .onTouch((event: TouchEvent) => {
      if (event.type === TouchType.Down) {
        this.pressed = true;
      } else if (event.type === TouchType.Up || event.type === TouchType.Cancel) {
        this.pressed = false;
      }
    })
    .onClick(() => { this.onSelect(this.item); })
  }
}
