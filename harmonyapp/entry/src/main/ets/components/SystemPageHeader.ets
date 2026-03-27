/**
 * SystemPageHeader
 * - 统一页面标题栏样式（Apple-like / HarmonyOS Next）
 * - 只负责“标题 + 可选右侧操作”，避免每个页面重复写 Row/Text/Button 样式
 */

import { DesignConstants } from '../constants/DesignConstants';
import { AppThemeManager } from '../core/theme/AppThemeManager';
import { KEY_RESOLVED_THEME_ID } from '../core/theme/AppThemeTypes';
import { AppIcon } from './SystemIcon';
import { SurfaceIconButton } from './SurfaceIconButton';
import { SafeAreaUtils } from '../core/utils/SafeAreaUtils';

export interface PageHeaderAction {
  label: string;
  onClick: () => void | Promise<void>;
  // 轻量扩展：有需要再加 icon / disabled / loading
  disabled?: boolean;
}

export interface IconAction {
  icon: AppIcon;
  onClick: () => void;
  kind?: 'primary' | 'neutral';
  disabled?: boolean;
  size?: number;
  iconSize?: number;
}

@Component
export struct SystemPageHeader {
  @Prop title: string = '';
  action: PageHeaderAction | null = null;
  @Prop showBackButton: boolean = false;
  onBack: (() => void) | null = null;
  leftAction: IconAction | null = null;
  rightAction: IconAction | null = null;
  @StorageLink('safeAreaTop') safeAreaTop: number = 0;
  @Prop surfaceColor: ResourceColor | null = null;
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'default_clean_light';

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  private getHeaderTopPadding(): number {
    return SafeAreaUtils.resolveTopPadding(this.safeAreaTop);
  }

  private getBackgroundColor(): ResourceColor {
    return this.surfaceColor ? this.surfaceColor : this.getPalette().pageBackground;
  }

  private getHeaderShadow(): ShadowOptions {
    return this.getPalette().tabBarShadow;
  }

  private getHeaderBorderColor(): ResourceColor {
    return this.getPalette().borderColor;
  }

  private canShowBackButton(): boolean {
    return this.showBackButton && this.onBack !== null;
  }

  private hasLeftAction(): boolean {
    return this.canShowBackButton() || this.leftAction !== null;
  }

  private hasRightAction(): boolean {
    return this.rightAction !== null || this.action !== null;
  }

  @Builder
  private buildIconActionButton(action: IconAction) {
    SurfaceIconButton({
      icon: action.icon,
      kind: action.kind ?? 'neutral',
      buttonSize: action.size ?? 40,
      glyphSize: action.iconSize ?? 20,
      disabled: action.disabled ?? false,
      onPress: action.onClick
    })
  }

  build() {
    Column() {
      Row({ space: DesignConstants.SPACING.SM }) {
        if (this.hasLeftAction()) {
          Row({ space: DesignConstants.SPACING.SM }) {
            if (this.canShowBackButton()) {
              SurfaceIconButton({
                icon: AppIcon.BACK,
                kind: 'neutral',
                onPress: () => {
                  if (this.onBack) {
                    this.onBack();
                  }
                }
              })
            }

            if (this.leftAction) {
              this.buildIconActionButton(this.leftAction);
            }
          }
          .alignItems(VerticalAlign.Center)
        } else {
          Blank().width(40)
        }

        Text(this.title)
          .fontSize(DesignConstants.FONT.SIZE.XL)
          .fontWeight(FontWeight.Medium)
          .fontColor(this.getPalette().textPrimary)
          .layoutWeight(1)
          .textAlign(TextAlign.Center)

        if (this.hasRightAction()) {
          Row({ space: DesignConstants.SPACING.SM }) {
            if (this.rightAction) {
              this.buildIconActionButton(this.rightAction);
            }

            if (this.action) {
              Button(this.action.label)
                .type(ButtonType.Normal)
                .backgroundColor(this.getPalette().surfaceBackground)
                .borderRadius(DesignConstants.RADIUS.FULL)
                .border({ width: 1, color: this.getPalette().borderColor })
                .shadow(this.getPalette().controlShadow)
                .height(38)
                .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG })
                .fontColor(this.action.disabled
                  ? this.getPalette().textSecondary
                  : this.getPalette().accentBackground)
                .fontSize(DesignConstants.FONT.SIZE.SM)
                .enabled(!this.action.disabled)
                .onClick(async () => {
                  try {
                    await this.action?.onClick()
                  } catch (_) {
                  }
                })
            }
          }
          .alignItems(VerticalAlign.Center)
        } else {
          Blank().width(24)
        }
      }
      .width('100%')
      .alignItems(VerticalAlign.Center)
      .padding({
        left: DesignConstants.SPACING.LG,
        right: DesignConstants.SPACING.LG,
        top: DesignConstants.SPACING.SM,
        bottom: DesignConstants.SPACING.SM
      })
    }
    .width('100%')
    .padding({
      top: this.getHeaderTopPadding(),
      bottom: DesignConstants.SPACING.SM
    })
    .backgroundColor(this.getBackgroundColor())
    .shadow(this.getHeaderShadow())
    .border({ width: { bottom: 1 }, color: this.getHeaderBorderColor() })
    .expandSafeArea([SafeAreaType.SYSTEM], [SafeAreaEdge.TOP])
  }
}
