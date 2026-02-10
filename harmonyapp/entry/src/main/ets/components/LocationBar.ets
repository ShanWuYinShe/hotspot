/**
 * LocationBar - 位置/浏览锚点选择栏
 * 显示当前位置或自定义锚点，提供搜索和地图入口
 */

import { LocationInfo } from '../core/location/LocationUtil';
import { KEY_BROWSE_ANCHOR, KEY_CURRENT_LOCATION } from '../core/state/GlobalStateManager';
import { PageNames } from '../constants/AppConstants';
import { DesignConstants } from '../constants/DesignConstants';
import { AppIcon } from './SystemIcon';

@Component
export struct LocationBar {
  @StorageLink(KEY_BROWSE_ANCHOR) browseAnchor: LocationInfo | null = null;
  @StorageLink(KEY_CURRENT_LOCATION) currentLocation: LocationInfo | null = null;

  /** 是否显示地图入口按钮 */
  @Prop showMapButton: boolean = true;
  /** 是否显示搜索按钮 */
  @Prop showSearchButton: boolean = true;

  private getDisplayLocation(): string {
    if (this.browseAnchor && this.browseAnchor.locationName) {
      return this.browseAnchor.locationName;
    }
    if (this.currentLocation && this.currentLocation.locationName) {
      return this.currentLocation.locationName;
    }
    return '定位中...';
  }

  private isUsingAnchor(): boolean {
    return this.browseAnchor !== null && this.browseAnchor !== undefined;
  }

  build() {
    Row() {
      // 位置选择区域
      Row({ space: 6 }) {
        Text() {
          SymbolSpan($r(AppIcon.LOCATION))
            .fontSize(18)
            .fontColor([$r('sys.color.ohos_id_color_emphasize')])
        }

        Text(this.getDisplayLocation())
          .fontSize(DesignConstants.FONT.SIZE.MD)
          .fontColor($r('sys.color.ohos_id_color_text_primary'))
          .maxLines(1)
          .textOverflow({ overflow: TextOverflow.Ellipsis })
          .constraintSize({ maxWidth: '50%' })

        // 下拉箭头
        Text() {
          SymbolSpan($r(AppIcon.DOWN))
            .fontSize(12)
            .fontColor([$r('sys.color.ohos_id_color_text_secondary')])
        }

        // 锚点清除按钮
        if (this.isUsingAnchor()) {
          Text() {
            SymbolSpan($r(AppIcon.CLOSE))
              .fontSize(14)
              .fontColor([$r('sys.color.ohos_id_color_text_secondary')])
          }
          .backgroundColor($r('sys.color.ohos_id_color_component_normal'))
          .borderRadius(DesignConstants.RADIUS.FULL)
          .width(22)
          .height(22)
          .textAlign(TextAlign.Center)
          .onClick(() => {
            this.browseAnchor = null;
          })
        }
      }
      .layoutWeight(1)
      .height('100%')
      .onClick(() => {
        this.openLocationPicker();
      })

      // 右侧按钮区
      Row({ space: 4 }) {
        if (this.showSearchButton) {
          this.buildIconButton(AppIcon.SEARCH, () => {
            this.navigateToSearch();
          })
        }

        if (this.showMapButton) {
          this.buildIconButton(AppIcon.MAP, () => {
            this.navigateToMap();
          })
        }
      }
    }
    .width('100%')
    .height(48)
    .padding({ left: 16, right: 12 })
    .backgroundColor($r('sys.color.ohos_id_color_background'))
  }

  @Builder
  private buildIconButton(icon: AppIcon, action: () => void) {
    Column() {
      Text() {
        SymbolSpan($r(icon))
          .fontSize(22)
          .fontColor([$r('sys.color.ohos_id_color_text_primary')])
      }
    }
    .width(40)
    .height(40)
    .justifyContent(FlexAlign.Center)
    .borderRadius(DesignConstants.RADIUS.FULL)
    .onClick(action)
  }

  private openLocationPicker(): void {
    try {
      const routerInstance = this.getUIContext().getRouter();
      routerInstance.pushUrl({
        url: `features/map/pages/${PageNames.LOCATION_PICKER}`,
        params: { resultKey: KEY_BROWSE_ANCHOR }
      });
    } catch (_e) {
      // ignore
    }
  }

  private navigateToSearch(): void {
    try {
      const routerInstance = this.getUIContext().getRouter();
      routerInstance.pushUrl({ url: `features/search/pages/${PageNames.SEARCH}` });
    } catch (_e) {
      // ignore
    }
  }

  private navigateToMap(): void {
    try {
      const routerInstance = this.getUIContext().getRouter();
      routerInstance.pushUrl({ url: `features/map/pages/${PageNames.MAP}` });
    } catch (_e) {
      // ignore
    }
  }
}
