/**
 * 位置选择页面（地图选点）
 * - 顶部搜索框：输入地名快速跳转
 * - 拖拽地图移动 → 停止后自动加载附近地点列表
 * - 双击地图 → 将选择器快速移动到该位置
 * - 用户从列表中选择一个地点确认
 */

import { KEY_BROWSE_ANCHOR, KEY_CURRENT_LOCATION, KEY_LOCATION_PICK_RESULT } from '../../../core/state/GlobalStateManager';
import { LocationInfo, LocationUtil } from '../../../core/location/LocationUtil';
import { NativeMapComponent, MapRegion, MapCoordinate } from '../../../components/NativeMapComponent';
import { SystemToast } from '../../../components/SystemToast';
import { DesignConstants } from '../../../constants/DesignConstants';
import { AppIcon } from '../../../components/SystemIcon';
import { hilog } from '@kit.PerformanceAnalysisKit';

const DOMAIN_NUMBER: number = 0xFF13;
const TAG: string = 'LocationPickerPage';

@Entry
@Component
struct LocationPickerPage {
  @StorageLink(KEY_CURRENT_LOCATION) currentLocation: LocationInfo | null = null;

  @State mapRegion: MapRegion = {
    latitude: 35.154921,
    longitude: 112.68253,
    latitudeDelta: 0.01,
    longitudeDelta: 0.01
  };

  /** 当前解析出的位置名称 */
  @State locationName: string = '';
  /** 当前中心坐标 */
  @State centerLat: number = 0;
  @State centerLng: number = 0;
  @State isLoading: boolean = false;
  @State titleText: string = '地图选点';
  @State confirmText: string = '确认选择';
  @State isPinAnimating: boolean = false;

  /** 搜索相关 */
  @State searchText: string = '';
  @State isSearching: boolean = false;
  @State showSearchBar: boolean = false;

  private resultKey: string = KEY_LOCATION_PICK_RESULT;
  private mode: 'browse_anchor' | 'pick_location' = 'pick_location';
  private resolveTimerId: number = 0;
  private isFirstCameraIdle: boolean = true;

  async aboutToAppear() {
    const params: Object | undefined = this.getUIContext().getRouter().getParams() as Object | undefined;
    const mode = this.getStringParam(params, 'mode');
    this.mode = mode === 'browse_anchor' ? 'browse_anchor' : 'pick_location';

    const resultKey = this.getStringParam(params, 'resultKey');
    this.resultKey = typeof resultKey === 'string' && resultKey.length > 0 ? resultKey : KEY_LOCATION_PICK_RESULT;

    this.titleText = this.getStringParam(params, 'title') || (this.mode === 'browse_anchor' ? '设置浏览锚点' : '地图选点');
    this.confirmText = this.getStringParam(params, 'confirmText') || (this.mode === 'browse_anchor' ? '设为浏览锚点' : '确认选择');

    AppStorage.setOrCreate<LocationInfo | null>(this.resultKey, null);

    // 初始化中心位置
    if (this.mode === 'browse_anchor') {
      const anchor = AppStorage.get<LocationInfo | null>(KEY_BROWSE_ANCHOR);
      if (anchor) {
        this.setCenter(anchor.latitude, anchor.longitude);
        await this.resolveLocation(anchor.latitude, anchor.longitude);
        return;
      }
    }

    if (this.currentLocation) {
      this.setCenter(this.currentLocation.latitude, this.currentLocation.longitude);
      await this.resolveLocation(this.currentLocation.latitude, this.currentLocation.longitude);
      return;
    }

    const location = await LocationUtil.getCurrentLocation();
    if (!location) return;
    this.setCenter(location.latitude, location.longitude);
    await this.resolveLocation(location.latitude, location.longitude);
  }

  aboutToDisappear() {
    if (this.resolveTimerId > 0) {
      clearTimeout(this.resolveTimerId);
      this.resolveTimerId = 0;
    }
  }

  private getStringParam(params: Object | undefined, key: string): string | undefined {
    if (!params || typeof params !== 'object') return undefined;
    const dict = params as Record<string, Object>;
    const v = dict[key];
    return typeof v === 'string' ? v as string : undefined;
  }

  private setCenter(lat: number, lng: number) {
    this.mapRegion = { latitude: lat, longitude: lng, latitudeDelta: 0.01, longitudeDelta: 0.01 };
  }

  // ========== 地图事件 ==========

  private onCameraIdle(center: MapCoordinate): void {
    if (this.isFirstCameraIdle) {
      this.isFirstCameraIdle = false;
      return;
    }

    if (this.resolveTimerId > 0) {
      clearTimeout(this.resolveTimerId);
    }

    this.isLoading = true;

    this.resolveTimerId = setTimeout(() => {
      this.resolveLocation(center.latitude, center.longitude);
    }, 400);
  }

  private onMapDoubleClick(coord: MapCoordinate): void {
    this.getUIContext().animateTo({ duration: 300, curve: Curve.FastOutSlowIn }, () => {
      this.isPinAnimating = true;
    });
    setTimeout(() => {
      this.getUIContext().animateTo({ duration: 400, curve: Curve.EaseOut }, () => {
        this.isPinAnimating = false;
      });
    }, 300);
    this.setCenter(coord.latitude, coord.longitude);
  }

  // ========== 位置解析 ==========

  private async resolveLocation(lat: number, lng: number): Promise<void> {
    this.isLoading = true;
    this.centerLat = lat;
    this.centerLng = lng;
    try {
      const addr = await LocationUtil.reverseGeocode(lat, lng);
      this.locationName = addr?.fullAddress || '未知位置';
    } catch (e) {
      hilog.warn(DOMAIN_NUMBER, TAG, `解析位置失败: ${JSON.stringify(e)}`);
      // 失败时保留旧名称
      if (this.locationName.length === 0) {
        this.locationName = '未知位置';
      }
    } finally {
      this.isLoading = false;
    }
  }

  // ========== 搜索 ==========

  private async doSearch(): Promise<void> {
    const keyword = this.searchText.trim();
    if (keyword.length === 0) return;

    this.isSearching = true;
    try {
      const result = await LocationUtil.geocode(keyword);
      if (result && LocationUtil.isValidCoordinate(result.latitude, result.longitude)) {
        this.setCenter(result.latitude, result.longitude);
        // cameraIdle 会自动触发 loadNearbyPlaces
        this.showSearchBar = false;
        // 收起键盘
        this.getUIContext().getFocusController().clearFocus();
      } else {
        SystemToast.show('未找到该地点，请换个关键词试试', this.getUIContext());
      }
    } catch (e) {
      hilog.warn(DOMAIN_NUMBER, TAG, `搜索地点失败: ${JSON.stringify(e)}`);
      SystemToast.show('搜索失败，请稍后重试', this.getUIContext());
    } finally {
      this.isSearching = false;
    }
  }

  // ========== 其他操作 ==========

  private relocateToCurrentPosition() {
    if (this.currentLocation) {
      this.setCenter(this.currentLocation.latitude, this.currentLocation.longitude);
    } else {
      SystemToast.show('无法获取当前位置', this.getUIContext());
    }
  }

  private confirm() {
    if (this.locationName.length === 0 || this.isLoading) {
      SystemToast.show('请等待位置识别完成', this.getUIContext());
      return;
    }
    AppStorage.setOrCreate<LocationInfo | null>(this.resultKey, {
      latitude: this.centerLat,
      longitude: this.centerLng,
      locationName: this.locationName
    });
    this.getUIContext().getRouter().back();
  }

  build() {
    Column() {
      // ===== 顶部导航栏 =====
      Row() {
        Column() {
          Text() {
            SymbolSpan($r(AppIcon.BACK))
              .fontSize(22)
              .fontColor([$r('sys.color.ohos_id_color_text_primary')])
          }
        }
        .width(44)
        .height(44)
        .justifyContent(FlexAlign.Center)
        .onClick(() => { this.getUIContext().getRouter().back(); })

        if (this.showSearchBar) {
          // 搜索输入框（带入场动画）
          TextInput({ placeholder: '搜索地点名称', text: this.searchText })
            .layoutWeight(1)
            .height(36)
            .borderRadius(DesignConstants.RADIUS.LG)
            .backgroundColor($r('sys.color.ohos_id_color_sub_background'))
            .fontSize(DesignConstants.FONT.SIZE.MD)
            .onChange((value: string) => { this.searchText = value; })
            .onSubmit(() => { this.doSearch(); })
            .transition(TransitionEffect.OPACITY.combine(
              TransitionEffect.translate({ x: 40 })
            ).animation({ duration: 250, curve: Curve.FastOutSlowIn }))

          // 搜索 / 取消按钮
          if (this.isSearching) {
            LoadingProgress()
              .width(24)
              .height(24)
              .color($r('sys.color.ohos_id_color_emphasize'))
              .margin({ left: DesignConstants.SPACING.SM })
          } else {
            Text('取消')
              .fontSize(DesignConstants.FONT.SIZE.MD)
              .fontColor($r('sys.color.ohos_id_color_emphasize'))
              .margin({ left: DesignConstants.SPACING.SM })
              .transition(TransitionEffect.OPACITY.animation({ duration: 200 }))
              .onClick(() => {
                this.getUIContext().animateTo({ duration: 250, curve: Curve.FastOutSlowIn }, () => {
                  this.showSearchBar = false;
                  this.searchText = '';
                });
              })
          }
        } else {
          // 标题（带入场动画）
          Text(this.titleText)
            .fontSize(DesignConstants.FONT.SIZE.XL)
            .fontWeight(FontWeight.Bold)
            .fontColor($r('sys.color.ohos_id_color_text_primary'))
            .layoutWeight(1)
            .textAlign(TextAlign.Center)
            .transition(TransitionEffect.OPACITY.animation({ duration: 200 }))

          // 搜索按钮
          Column() {
            Text() {
              SymbolSpan($r(AppIcon.SEARCH))
                .fontSize(22)
                .fontColor([$r('sys.color.ohos_id_color_text_primary')])
            }
          }
          .width(44)
          .height(44)
          .justifyContent(FlexAlign.Center)
          .onClick(() => {
            this.getUIContext().animateTo({ duration: 250, curve: Curve.FastOutSlowIn }, () => {
              this.showSearchBar = true;
            });
          })
        }
      }
      .width('100%')
      .height(56)
      .padding({ left: DesignConstants.SPACING.SM, right: DesignConstants.SPACING.SM })
      .backgroundColor($r('sys.color.ohos_id_color_background'))

      // ===== 地图区域 =====
      Stack({ alignContent: Alignment.Center }) {
        NativeMapComponent({
          mapRegion: this.mapRegion,
          markers: [],
          showCompass: true,
          showScale: true,
          showZoomControls: true,
          onCameraIdle: (center: MapCoordinate) => { this.onCameraIdle(center); },
          onMapDoubleClick: (coord: MapCoordinate) => { this.onMapDoubleClick(coord); }
        })

        // 中心选择器 — 地图钉
        Column() {
          Stack({ alignContent: Alignment.Center }) {
            Column()
              .width(40)
              .height(40)
              .borderRadius(20)
              .backgroundColor('rgba(0, 122, 255, 0.12)')
            Column()
              .width(24)
              .height(24)
              .borderRadius(12)
              .backgroundColor($r('sys.color.ohos_id_color_emphasize'))
              .border({ width: 3, color: Color.White })
              .shadow({ radius: 6, color: 'rgba(0,0,0,0.25)', offsetX: 0, offsetY: 2 })
          }

          Column()
            .width(3)
            .height(14)
            .borderRadius(1.5)
            .backgroundColor($r('sys.color.ohos_id_color_emphasize'))
            .margin({ top: -2 })

          Column()
            .width(10)
            .height(4)
            .borderRadius(5)
            .backgroundColor('rgba(0,0,0,0.12)')
            .margin({ top: 6 })
        }
        .offset({ y: this.isPinAnimating ? -30 : -18 })
        .scale({ x: this.isPinAnimating ? 1.15 : 1.0, y: this.isPinAnimating ? 1.15 : 1.0 })
        .animation({ duration: 350, curve: Curve.FastOutSlowIn })

        // 回到当前位置
        Column() {
          Text() {
            SymbolSpan($r(AppIcon.MAP))
              .fontSize(20)
              .fontColor([$r('sys.color.ohos_id_color_emphasize')])
          }
        }
        .width(44)
        .height(44)
        .justifyContent(FlexAlign.Center)
        .backgroundColor($r('sys.color.ohos_id_color_background'))
        .borderRadius(DesignConstants.RADIUS.FULL)
        .shadow({ radius: 8, color: 'rgba(0,0,0,0.15)', offsetX: 0, offsetY: 2 })
        .position({ right: 16, bottom: 16 })
        .onClick(() => { this.relocateToCurrentPosition(); })
      }
      .width('100%')
      .layoutWeight(1)

      // ===== 底部面板：位置名称 + 确认按钮 =====
      Column() {
        // 拖拽指示条
        Column()
          .width(36)
          .height(4)
          .borderRadius(2)
          .backgroundColor($r('sys.color.ohos_id_color_component_normal'))
          .margin({ top: DesignConstants.SPACING.SM, bottom: DesignConstants.SPACING.MD })

        // 位置名称（固定高度，内容原地替换，不改变布局）
        Row({ space: DesignConstants.SPACING.SM }) {
          if (this.isLoading) {
            LoadingProgress()
              .width(18)
              .height(18)
              .color($r('sys.color.ohos_id_color_emphasize'))
          }

          Text(this.isLoading ? '正在识别位置...' : (this.locationName || '移动地图选择位置'))
            .fontSize(DesignConstants.FONT.SIZE.LG)
            .fontWeight(this.locationName.length > 0 && !this.isLoading ? FontWeight.Medium : FontWeight.Normal)
            .fontColor(this.locationName.length > 0 && !this.isLoading
              ? $r('sys.color.ohos_id_color_text_primary')
              : $r('sys.color.ohos_id_color_text_secondary'))
            .layoutWeight(1)
            .maxLines(2)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
        }
        .width('100%')
        .constraintSize({ minHeight: 44 })
        .alignItems(VerticalAlign.Center)

        // 确认按钮
        Button(this.confirmText)
          .type(ButtonType.Capsule)
          .width('100%')
          .height(48)
          .margin({ top: DesignConstants.SPACING.MD })
          .backgroundColor(this.locationName.length > 0 && !this.isLoading
            ? $r('sys.color.ohos_id_color_emphasize')
            : $r('sys.color.ohos_id_color_component_normal'))
          .fontColor(this.locationName.length > 0 && !this.isLoading
            ? Color.White
            : $r('sys.color.ohos_id_color_text_secondary'))
          .fontSize(DesignConstants.FONT.SIZE.LG)
          .fontWeight(FontWeight.Medium)
          .enabled(this.locationName.length > 0 && !this.isLoading)
          .onClick(() => this.confirm())
      }
      .width('100%')
      .padding({
        left: DesignConstants.SPACING.XL,
        right: DesignConstants.SPACING.XL,
        bottom: DesignConstants.SPACING.XL
      })
      .backgroundColor($r('sys.color.ohos_id_color_background'))
      .borderRadius({ topLeft: DesignConstants.RADIUS.XL, topRight: DesignConstants.RADIUS.XL })
      .shadow({ radius: 16, color: 'rgba(0,0,0,0.08)', offsetX: 0, offsetY: -4 })
    }
    .width('100%')
    .height('100%')
    .backgroundColor($r('sys.color.ohos_id_color_sub_background'))
  }
}
