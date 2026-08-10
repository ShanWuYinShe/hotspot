/**
 * 智能位置卡片组件
 * 集成天气、海拔、导航等多种HarmonyOS系统服务
 */

import { hilog } from '@kit.PerformanceAnalysisKit';
import { vibrator } from '@kit.SensorServiceKit';
import { DesignConstants } from '../constants/DesignConstants';
import { AppIcon } from './SystemIcon';
import { LocationUtil } from '../core/location/LocationUtil';
import { AppThemeManager } from '../core/theme/AppThemeManager';
import { KEY_RESOLVED_THEME_ID } from '../core/theme/AppThemeTypes';

const DOMAIN_NUMBER: number = 0xFF11;
const TAG: string = 'IntelligentLocationCard';

@Component
export struct IntelligentLocationCard {
  @Prop latitude: number = 0;
  @Prop longitude: number = 0;
  @Prop locationName: string = '未知位置';
  @Prop distance: number = -1;
  @Prop altitude: number = 0; // 新增海拔属性
  @Prop showElevation: boolean = true;
  onLocationTap?: () => void;
  onNavigationTap?: () => void;
  @State private cardElevation: number = 1;
  @State private elevationInfo: string = '';
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'vibrant_light';

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  aboutToAppear() {
    if (this.showElevation) {
      this.loadElevationInfo();
    }
  }

  build() {
    Row({ space: DesignConstants.SPACING.MD }) {
      // 位置图标 - 使用SymbolSpan
      Text() {
        SymbolSpan($r(AppIcon.LOCATION))
          .fontSize(DesignConstants.FONT.SIZE.XXL)
          .fontColor([this.getPalette().accentBackground])
      }

      Column({ space: DesignConstants.SPACING.SM2 }) {
        // 地址信息
        Text(this.locationName || '未知位置')
          .fontSize(DesignConstants.FONT.SIZE.LG)
          .fontColor(this.getPalette().textPrimary)
          .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
          .maxLines(2)
          .textOverflow({ overflow: TextOverflow.Ellipsis })

        Row({ space: DesignConstants.SPACING.MD }) {
          // 距离信息
          if (this.distance >= 0) {
              Row({ space: DesignConstants.SPACING.XS }) {
                Text() {
                  SymbolSpan($r(AppIcon.SEND))
                    .fontSize(DesignConstants.FONT.SIZE.SM)
                    .fontColor([this.getPalette().textSecondary])
                }
                Text(LocationUtil.formatDistance(this.distance))
                  .fontSize(DesignConstants.FONT.SIZE.SM)
                  .fontColor(this.getPalette().textSecondary)
              }
          }

          // 海拔信息
          if (this.showElevation && this.elevationInfo) {
            Row({ space: DesignConstants.SPACING.XS }) {
              Text('⛰️')
                .fontSize(DesignConstants.FONT.SIZE.SM)
              
              Text(this.elevationInfo)
                .fontSize(DesignConstants.FONT.SIZE.SM)
                .fontColor(this.getPalette().textSecondary)
            }
          }
        }

      }
      .alignItems(HorizontalAlign.Start)
      .layoutWeight(1)

      // 操作按钮组
      Column({ space: DesignConstants.SPACING.SM }) {
        // 查看详情按钮
        Button({ type: ButtonType.Normal, stateEffect: true }) {
          Text() {
            SymbolSpan($r(AppIcon.INFO))
              .fontSize(DesignConstants.FONT.SIZE.LG)
              .fontColor([this.getPalette().accentText as ResourceColor])
          }
        }
        .width(DesignConstants.COMPONENT.AVATAR_SIZE.XS)
        .height(DesignConstants.COMPONENT.AVATAR_SIZE.XS)
        .backgroundColor(this.getPalette().accentBackground)
        .borderRadius(DesignConstants.RADIUS.XL)
        .onClick(() => {
          this.triggerHapticFeedback();
          if (this.onLocationTap) {
            this.onLocationTap();
          }
        })

        // 导航按钮
        Button({ type: ButtonType.Normal, stateEffect: true }) {
          Text() {
            SymbolSpan($r(AppIcon.UP))
              .fontSize(DesignConstants.FONT.SIZE.LG)
              .fontColor([this.getPalette().accentBackground])
          }
        }
        .width(DesignConstants.COMPONENT.AVATAR_SIZE.XS)
        .height(DesignConstants.COMPONENT.AVATAR_SIZE.XS)
        .backgroundColor(this.getPalette().surfaceBackground)
        .border({
          width: 1,
          color: this.getPalette().accentBackground
        })
        .borderRadius(DesignConstants.RADIUS.XL)
        .onClick(() => {
          this.triggerHapticFeedback();
          if (this.onNavigationTap) {
            this.onNavigationTap();
          }
        })
      }
    }
    .width('100%')
    .padding(DesignConstants.SPACING.LG)
    .backgroundColor(this.getPalette().surfaceBackground)
    .borderRadius(DesignConstants.RADIUS.XL)
    .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getPalette().borderColor })
    .shadow({
      radius: this.getPalette().surfaceShadow.radius + this.cardElevation * 2,
      color: this.getPalette().surfaceShadow.color,
      offsetX: this.getPalette().surfaceShadow.offsetX,
      offsetY: this.getPalette().surfaceShadow.offsetY + this.cardElevation
    })
    .onHover((isHover: boolean) => {
      // 使用最新的动画API
      this.cardElevation = isHover ? DesignConstants.INTERACTION.HOVER_ELEVATION : DesignConstants.INTERACTION.HOVER_ELEVATION_REST;
    })
    .animation({
      duration: DesignConstants.ANIMATION.DURATION_250,
      curve: DesignConstants.ANIMATION.CURVE_EASE_IN_OUT
    })
    .gesture(
      TapGesture()
        .onAction(() => {
          this.triggerHapticFeedback();
          if (this.onLocationTap) {
            this.onLocationTap();
          }
        })
    )
  }


  private async loadElevationInfo(): Promise<void> {
    try {
      if (this.altitude && this.altitude !== 0) {
         this.elevationInfo = `海拔 ${Math.round(this.altitude)}m`;
      } else {
         this.elevationInfo = '';
      }
    } catch (error) {
      hilog.warn(DOMAIN_NUMBER, TAG, '加载海拔信息失败');
    }
  }

  private triggerHapticFeedback(): void {
    try {
      vibrator.startVibration({
        type: 'time',
        duration: 15
      }, {
        id: 0,
        usage: 'alarm'
      });
    } catch (error) {
      hilog.warn(DOMAIN_NUMBER, TAG, '触觉反馈失败');
    }
  }
}
