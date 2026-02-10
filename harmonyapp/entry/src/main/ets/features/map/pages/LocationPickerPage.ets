/**
 * 位置选择页面（地图选点）
 * - 通过点击地图选择一个坐标点
 * - 自动进行反向地理编码生成 locationName
 * - 结果写入 AppStorage: KEY_LOCATION_PICK_RESULT，调用方返回后读取并消费
 */

import { KEY_BROWSE_ANCHOR, KEY_CURRENT_LOCATION, KEY_LOCATION_PICK_RESULT } from '../../../core/state/GlobalStateManager';
import { LocationInfo, LocationUtil } from '../../../core/location/LocationUtil';
import { NativeMapComponent, MapRegion } from '../../../components/NativeMapComponent';
import { SystemToast } from '../../../components/SystemToast';
import { DesignConstants } from '../../../constants/DesignConstants';
import { IconButton, AppIcon, IconSize } from '../../../components/SystemIcon';

interface MapCenter {
  latitude: number;
  longitude: number;
}

@Entry
@Component
struct LocationPickerPage {
  @StorageLink(KEY_CURRENT_LOCATION) currentLocation: LocationInfo | null = null;

  @State mapCenter: MapCenter | null = null;
  @State mapRegion: MapRegion = {
    latitude: 35.154921,
    longitude: 112.68253,
    latitudeDelta: 0.01,
    longitudeDelta: 0.01
  };

  @State selected: LocationInfo | null = null;
  @State isResolving: boolean = false;
  @State titleText: string = '地图选点';
  @State confirmText: string = '确认选择';
  @State hintText: string = '点击地图选择位置';

  private resultKey: string = KEY_LOCATION_PICK_RESULT;
  private mode: 'browse_anchor' | 'pick_location' = 'pick_location';

  async aboutToAppear() {
    // 从路由参数读取用途（默认：业务选点）
    const params = this.getUIContext().getRouter().getParams() as unknown;
    const mode = this.getStringParam(params, 'mode');
    this.mode = mode === 'browse_anchor' ? 'browse_anchor' : 'pick_location';

    const resultKey = this.getStringParam(params, 'resultKey');
    this.resultKey = typeof resultKey === 'string' && resultKey.length > 0 ? resultKey : KEY_LOCATION_PICK_RESULT;

    this.titleText = this.getStringParam(params, 'title') || (this.mode === 'browse_anchor' ? '设置浏览锚点' : '地图选点');
    this.confirmText = this.getStringParam(params, 'confirmText') || (this.mode === 'browse_anchor' ? '设为浏览锚点' : '确认选择');
    this.hintText = this.getStringParam(params, 'hintText') || (this.mode === 'browse_anchor' ? '点击地图设置浏览锚点' : '点击地图选择位置');

    // 清空旧结果，避免误用（清当前用途对应的 key）
    AppStorage.setOrCreate<LocationInfo | null>(this.resultKey, null);

    // 默认中心：浏览锚点模式优先用已设置的锚点；否则用真实位置
    if (this.mode === 'browse_anchor') {
      const anchor = AppStorage.get<LocationInfo | null>(KEY_BROWSE_ANCHOR);
      if (anchor) {
        this.setCenter(anchor.latitude, anchor.longitude);
        this.selected = {
          latitude: anchor.latitude,
          longitude: anchor.longitude,
          locationName: anchor.locationName || '浏览锚点'
        };
        await this.resolveName(this.selected.latitude, this.selected.longitude);
        return;
      }
    }

    if (this.currentLocation) {
      this.setCenter(this.currentLocation.latitude, this.currentLocation.longitude);
      this.selected = {
        latitude: this.currentLocation.latitude,
        longitude: this.currentLocation.longitude,
        locationName: this.currentLocation.locationName || '当前位置'
      };
      await this.resolveName(this.selected.latitude, this.selected.longitude);
      return;
    }

    const location = await LocationUtil.getCurrentLocation();
    if (!location) return;

    this.setCenter(location.latitude, location.longitude);
    this.selected = {
      latitude: location.latitude,
      longitude: location.longitude,
      locationName: location.locationName || '当前位置'
    };
    await this.resolveName(location.latitude, location.longitude);
  }

  private getStringParam(params: unknown, key: string): string | undefined {
    if (!params || typeof params !== 'object') return undefined;
    const dict = params as Record<string, unknown>;
    const v = dict[key];
    return typeof v === 'string' ? v : undefined;
  }

  private setCenter(lat: number, lng: number) {
    this.mapCenter = { latitude: lat, longitude: lng };
    this.mapRegion = {
      latitude: lat,
      longitude: lng,
      latitudeDelta: 0.01,
      longitudeDelta: 0.01
    };
  }

  private async resolveName(lat: number, lng: number): Promise<void> {
    this.isResolving = true;
    try {
      const addr = await LocationUtil.reverseGeocode(lat, lng);
      const name = addr?.fullAddress || `${lat.toFixed(4)}, ${lng.toFixed(4)}`;
      this.selected = {
        latitude: lat,
        longitude: lng,
        locationName: name
      };
    } catch {
      this.selected = {
        latitude: lat,
        longitude: lng,
        locationName: `${lat.toFixed(4)}, ${lng.toFixed(4)}`
      };
    } finally {
      this.isResolving = false;
    }
  }

  private async onMapClick(coordinate: { latitude: number; longitude: number }) {
    await this.resolveName(coordinate.latitude, coordinate.longitude);
  }

  private confirm() {
    if (!this.selected) {
      SystemToast.show(this.mode === 'browse_anchor' ? '请在地图上点击选择锚点' : '请在地图上点击选择一个位置', this.getUIContext());
      return;
    }

    AppStorage.setOrCreate<LocationInfo | null>(this.resultKey, {
      latitude: this.selected.latitude,
      longitude: this.selected.longitude,
      locationName: this.selected.locationName || '未知位置'
    });

    const routerInstance = this.getUIContext().getRouter();
    routerInstance.back();
  }

  build() {
    Column() {
      // 顶部栏
      Row() {
        IconButton({
          icon: AppIcon.BACK,
          iconSize: IconSize.LARGE,
          iconColor: DesignConstants.COLOR.TEXT_PRIMARY_LIGHT,
          bgColor: Color.Transparent,
          onButtonClick: () => {
            const routerInstance = this.getUIContext().getRouter();
            routerInstance.back();
          }
        })

        Text(this.titleText)
          .fontSize(DesignConstants.FONT.SIZE.XL)
          .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
          .fontColor($r('sys.color.ohos_id_color_text_primary'))
          .layoutWeight(1)
          .textAlign(TextAlign.Center)

        Blank().width(48)
      }
      .width('100%')
      .height(56)
      .padding({ left: DesignConstants.SPACING.MD, right: DesignConstants.SPACING.MD })
      .backgroundColor($r('sys.color.ohos_id_color_background'))

      // 地图 + 底部信息
      Stack() {
        NativeMapComponent({
          mapRegion: this.mapRegion,
          markers: [],
          showCompass: true,
          showScale: true,
          showZoomControls: false,
          onMapClick: (pos) => {
            this.onMapClick(pos);
          }
        })

        // 中心提示（视觉引导）
        Column() {
          Text() { SymbolSpan($r('sys.symbol.local_fill')).fontSize(24).fontColor([$r('sys.color.ohos_id_color_emphasize')]) }
        }
        .position({ x: '50%', y: '50%' })
        .translate({ x: -12, y: -32 })

        // 底部卡片
        Column() {
          Column() {
          Text(this.selected?.locationName || this.hintText)
              .fontSize(DesignConstants.FONT.SIZE.MD)
              .fontColor($r('sys.color.ohos_id_color_text_primary'))
              .maxLines(2)
              .textOverflow({ overflow: TextOverflow.Ellipsis })

            if (this.selected) {
              Text(`${this.selected.latitude.toFixed(6)}, ${this.selected.longitude.toFixed(6)}`)
                .fontSize(DesignConstants.FONT.SIZE.SM)
                .fontColor($r('sys.color.ohos_id_color_text_secondary'))
                .margin({ top: 4 })
            }

            if (this.isResolving) {
              Text('正在解析地址…')
                .fontSize(DesignConstants.FONT.SIZE.SM)
                .fontColor($r('sys.color.ohos_id_color_text_secondary'))
                .margin({ top: 4 })
            }
          }
          .alignItems(HorizontalAlign.Start)
          .padding(16)

        Button(this.confirmText)
            .type(ButtonType.Capsule)
            .width('100%')
            .height(44)
            .margin({ left: 16, right: 16, bottom: 16 })
            .backgroundColor($r('sys.color.ohos_id_color_emphasize'))
            .fontColor(Color.White)
            .onClick(() => this.confirm())
        }
        .width('100%')
        .position({ x: 0, y: '100%' })
        .translate({ x: 0, y: -140 })
      }
      .layoutWeight(1)
    }
    .width('100%')
    .height('100%')
    .backgroundColor($r('sys.color.ohos_id_color_sub_background'))
  }
}

