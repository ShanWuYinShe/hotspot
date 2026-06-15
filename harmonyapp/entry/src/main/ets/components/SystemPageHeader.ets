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
    return this.surfaceColor ? this.surfaceColor : this.getPalette().sheetBackground;
  }

  private getHeaderShadow(): ShadowOptions {
    return this.getPalette().surfaceShadow;
  }

  private getHeaderBorderColor(): ResourceColor {
    return this.getPalette().floatingBorderColor;
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
        // 左侧：返回按钮 / 左操作
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
        }

        // 标题：左对齐，对齐前端 H1 样式
        Text(this.title)
          .fontSize(DesignConstants.FONT.SIZE.XL)
          .fontWeight(600)
          .fontColor(this.getPalette().textPrimary)
          .layoutWeight(1)
          .textAlign(TextAlign.Start)

        // 右侧操作
        if (this.hasRightAction()) {
          Row({ space: DesignConstants.SPACING.SM }) {
            if (this.rightAction) {
              this.buildIconActionButton(this.rightAction);
            }

            if (this.action) {
              Button(this.action.label)
                .type(ButtonType.Normal)
                .backgroundColor(this.getPalette().accentBackground)
                .borderRadius(DesignConstants.RADIUS.FULL)
                .height(36)
                .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG })
                .fontColor(this.getPalette().accentText)
                .fontSize(DesignConstants.FONT.SIZE.SM)
                .fontWeight(FontWeight.Medium)
                .enabled(!this.action.disabled)
                .onClick(() => {
                  this.action?.onClick();
                })
            }
          }
          .alignItems(VerticalAlign.Center)
        }
      }
      .width('100%')
      .alignItems(VerticalAlign.Center)
      .padding({
        left: 16,
        right: 16,
        top: 12,
        bottom: 12
      })
    }
    .width('100%')
    .padding({
      top: this.getHeaderTopPadding()
    })
    .backgroundColor(this.getBackgroundColor())
    .border({ width: { bottom: 1 }, color: this.getHeaderBorderColor() })
    .expandSafeArea([SafeAreaType.SYSTEM], [SafeAreaEdge.TOP])
  }
}
