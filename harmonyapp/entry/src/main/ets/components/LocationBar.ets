/**
 * LocationBar - 位置/浏览锚点选择栏
 * 显示当前位置或自定义锚点，提供搜索和地图入口
 *
 * 设计语言：
 * - 轻量中性底 + 柔和蓝强调，贴近 Stitch 的简洁层级
 * - 当前位置模式：脉冲动画传达「实时定位」
 * - 浏览锚点模式：浅蓝状态块明确区分
 */

import { LocationInfo } from '../core/location/LocationUtil';
import { KEY_BROWSE_ANCHOR, KEY_CURRENT_LOCATION } from '../core/state/GlobalStateManager';
import { DesignConstants } from '../constants/DesignConstants';
import { PageNames } from '../constants/AppConstants';
import { AppRouter, LocationPickerRouteParam } from '../core/router/AppRouter';
import { AppIcon } from './SystemIcon';
import { AppThemeManager } from '../core/theme/AppThemeManager';
import { KEY_RESOLVED_THEME_ID } from '../core/theme/AppThemeTypes';

@Component
export struct LocationBar {
  @StorageLink(KEY_BROWSE_ANCHOR) browseAnchor: LocationInfo | null = null;
  @StorageLink(KEY_CURRENT_LOCATION) currentLocation: LocationInfo | null = null;
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'default_clean_light';

  /** 是否显示搜索按钮 */
  @Prop showSearchButton: boolean = true;

  /** 脉冲呼吸动画状态 */
  @State private pulseScale: number = 1.0;
  @State private pulseOpacity: number = 0.7;
  private pulseTimerId: number = 0;

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

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  /** 启动脉冲呼吸动画 */
  private startPulseAnimation(): void {
    this.stopPulseAnimation();
    this.doPulse();
    this.pulseTimerId = setInterval(() => {
      this.pulseScale = 1.0;
      this.pulseOpacity = 0.7;
      setTimeout(() => {
        this.doPulse();
      }, 50);
    }, 2400);
  }

  private doPulse(): void {
    try {
      this.getUIContext().animateTo({
        duration: 1500,
        curve: Curve.EaseOut
      }, () => {
        this.pulseScale = 2.2;
        this.pulseOpacity = 0;
      });
    } catch (_e) {
      // 组件未挂载时忽略
    }
  }

  /** 停止脉冲动画 */
  private stopPulseAnimation(): void {
    if (this.pulseTimerId > 0) {
      clearInterval(this.pulseTimerId);
      this.pulseTimerId = 0;
    }
  }

  aboutToDisappear(): void {
    this.stopPulseAnimation();
  }

  build() {
    Row({ space: DesignConstants.SPACING.SM }) {

      // ===== 位置卡片（悬浮圆角，主交互区域）=====
      Row({ space: DesignConstants.SPACING.SM }) {
        // 左侧定位指示器
        if (this.isUsingAnchor()) {
          // 浏览锚点模式：蓝色背景圆形图标
          Column() {
            Text() {
              SymbolSpan($r(AppIcon.LOCATION))
                .fontSize(14)
                .fontColor([Color.White])
            }
          }
          .width(28)
          .height(28)
          .justifyContent(FlexAlign.Center)
          .backgroundColor($r('sys.color.ohos_id_color_emphasize'))
          .borderRadius(DesignConstants.RADIUS.XXL)
          .transition(TransitionEffect.OPACITY.animation({ duration: 250, curve: Curve.FastOutSlowIn }))
        } else {
          // 当前位置模式：脉冲动画指示器
          Stack({ alignContent: Alignment.Center }) {
            // 脉冲扩散环
            Column()
              .width(10)
              .height(10)
              .borderRadius(DesignConstants.RADIUS.FULL)
              .backgroundColor($r('sys.color.ohos_id_color_emphasize'))
              .scale({ x: this.pulseScale, y: this.pulseScale })
              .opacity(this.pulseOpacity)

            // 核心圆点（白色描边 + 阴影，营造立体感）
            Column()
              .width(10)
              .height(10)
              .borderRadius(DesignConstants.RADIUS.FULL)
              .backgroundColor($r('sys.color.ohos_id_color_emphasize'))
              .border({ width: 2, color: this.getPalette().pageBackground })
          }
          .width(28)
          .height(28)
          .transition(TransitionEffect.OPACITY.animation({ duration: 250, curve: Curve.FastOutSlowIn }))
          .onAppear(() => {
            this.startPulseAnimation();
          })
        }

        // 位置文本
        Column({ space: 1 }) {
          Text(this.isUsingAnchor() ? '浏览位置' : '当前位置')
            .fontSize(DesignConstants.FONT.SIZE.XS)
            .fontColor(this.isUsingAnchor()
              ? $r('sys.color.ohos_id_color_emphasize')
              : $r('sys.color.ohos_id_color_text_tertiary'))

          Text(this.getDisplayLocation())
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .fontWeight(FontWeight.Medium)
            .fontColor($r('sys.color.ohos_id_color_text_primary'))
            .maxLines(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
        }
        .layoutWeight(1)
        .alignItems(HorizontalAlign.Start)

        // 右侧前进箭头（点击进入位置选择）
        Text() {
          SymbolSpan($r(AppIcon.FORWARD))
            .fontSize(14)
            .fontColor([$r('sys.color.ohos_id_color_text_tertiary')])
        }
      }
      .layoutWeight(1)
      .height(40)
      .padding({ left: DesignConstants.SPACING.MD, right: DesignConstants.SPACING.SM })
      .backgroundColor(this.isUsingAnchor()
        ? $r('app.color.color_emphasize_bg')
        : this.getPalette().surfaceSubtle)
      .borderRadius(DesignConstants.RADIUS.XL)
      .border({ width: 1, color: $r('sys.color.ohos_id_color_list_separator') })
      .onClick(() => {
        this.openLocationPicker();
      })

      // ===== 右侧功能按钮（悬浮圆形）=====
      if (this.showSearchButton) {
        this.buildIconButton(AppIcon.SEARCH, () => {
          this.navigateToSearch();
        })
      }
    }
    .width('100%')
    .height(48)
    .padding({
      left: 0,
      right: 0
    })
  }

  @Builder
  private buildIconButton(icon: AppIcon, action: () => void) {
    Column() {
      Text() {
        SymbolSpan($r(icon))
          .fontSize(20)
          .fontColor([$r('sys.color.ohos_id_color_text_primary')])
      }
    }
    .width(40)
    .height(40)
    .justifyContent(FlexAlign.Center)
    .backgroundColor(this.getPalette().surfaceSubtle)
    .borderRadius(DesignConstants.RADIUS.XL)
    .border({ width: 1, color: $r('sys.color.ohos_id_color_list_separator') })
    .onClick(action)
  }

  private openLocationPicker(): void {
    const param = new LocationPickerRouteParam();
    param.mode = 'anchor';
    AppRouter.push(PageNames.LOCATION_PICKER, param);
  }

  private navigateToSearch(): void {
    AppRouter.push(PageNames.SEARCH);
  }
}
