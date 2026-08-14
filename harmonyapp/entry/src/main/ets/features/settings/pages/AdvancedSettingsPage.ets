/**
 * 高级设置页面
 * 重构：使用原生 List 组件
 */

import { DesignConstants } from '../../../constants/DesignConstants';
import { ResponsiveContainer } from '../../../components/ResponsiveContainer';
import { curves } from '@kit.ArkUI';
import { AppIcon, IconSize, IconButton } from '../../../components/SystemIcon';
import { SystemPageHeader } from '../../../components/SystemPageHeader';
import { SystemToast } from '../../../components/SystemToast';
import { AppRouter } from '../../../core/router/AppRouter';
import { PageNames } from '../../../constants/AppConstants';
import { SystemUiController } from '../../../core/utils/SystemUiController';
import { SafeAreaUtils } from '../../../core/utils/SafeAreaUtils';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { KEY_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';

@Preview
@Component
export struct AdvancedSettingsPage {
  pathStack: NavPathStack = new NavPathStack();
  @StorageLink('safeAreaTop') safeAreaTop: number = 0;
  @StorageLink(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'vibrant_light';
  @State private pressedSettingKey: string = '';

  aboutToAppear(): void {
    SystemUiController.syncWithCurrentTheme();
  }

  private getCurrentThemePalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  private getSurfaceBackground(): string {
    return this.getCurrentThemePalette().surfaceBackground;
  }

  build() {
    NavDestination() {
  ResponsiveContainer() {
    Column() {
      // 标题栏 —— 统一走 SystemPageHeader（居中标题 + 返回）
      SystemPageHeader({
        title: '高级设置',
        showBackButton: true,
        onBack: () => { AppRouter.pop(); },
        titleAlign: TextAlign.Center,
        titleWeight: DesignConstants.FONT.WEIGHT.MEDIUM,
        backgroundVariant: 'transparent'
      })

      Column() {
        Column() {
          this.SettingItem(AppIcon.CLEAN, '清理缓存', '清除本地缓存数据', () => {
            SystemToast.show('缓存已清理', this.getUIContext());
          })

          this.SettingItem(AppIcon.DEBUG, '调试面板', '网络诊断、状态检查', () => {
            AppRouter.push(PageNames.DEBUG_PANEL);
          })
        }
        .backgroundColor(this.getSurfaceBackground())
        .borderRadius(DesignConstants.RADIUS.LG)
        .shadow(this.getCurrentThemePalette().elevatedShadow)
        .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getCurrentThemePalette().borderColor })
        .margin({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG,
          top: DesignConstants.SPACING.LG, bottom: DesignConstants.SPACING.MD3 })
      }
      .width('100%')
      .layoutWeight(1)
      .alignItems(HorizontalAlign.Start)
      .backgroundColor(DesignConstants.COLOR.TRANSPARENT)
    }
    .width('100%')
    .height('100%')
    .backgroundColor(this.getCurrentThemePalette().pageBackground)
    }
    }
    .hideTitleBar(true)
    .onReady((context: NavDestinationContext) => {
      this.pathStack = context.pathStack;
    })
  }

  @Builder
  SettingItem(icon: AppIcon, title: string, subtitle: string, onClick: () => void) {
    Row({ space: DesignConstants.SPACING.MD }) {
      Row() {
        Text() {
          SymbolSpan($r(icon))
            .fontSize(DesignConstants.FONT.SIZE.LG_PLUS)
            .fontColor([this.getCurrentThemePalette().accentBackground])
        }
      }
      .width(DesignConstants.COMPONENT.BRAND_ICON_SIZE)
      .height(DesignConstants.COMPONENT.BRAND_ICON_SIZE)
      .borderRadius(DesignConstants.SHAPE.ICON_TILE)
      .backgroundColor(this.getCurrentThemePalette().surfaceSubtle)
      .justifyContent(FlexAlign.Center)
      .alignItems(VerticalAlign.Center)

      Column({ space: DesignConstants.SPACING.XS }) {
        Text(title)
          .fontSize(DesignConstants.FONT.SIZE.LG)
          .fontColor(this.getCurrentThemePalette().textPrimary)

        if (subtitle) {
          Text(subtitle)
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .fontColor(this.getCurrentThemePalette().textSecondary)
        }
      }
      .layoutWeight(1)
      .alignItems(HorizontalAlign.Start)

      Row() {
        Text() {
          SymbolSpan($r(AppIcon.ARROW_RIGHT))
            .fontSize(DesignConstants.FONT.SIZE.MD)
            .fontColor([this.getCurrentThemePalette().textSecondary])
        }
      }
      .width(DesignConstants.COMPONENT.STATUS_ICON_SIZE)
      .height(DesignConstants.COMPONENT.STATUS_ICON_SIZE)
      .borderRadius(DesignConstants.SHAPE.ICON_TILE)
      .backgroundColor(this.getCurrentThemePalette().surfaceSubtle)
      .justifyContent(FlexAlign.Center)
      .alignItems(VerticalAlign.Center)
    }
    .width('100%')
    .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, top: DesignConstants.SPACING.MD, bottom: DesignConstants.SPACING.MD })
    // 整行按压反馈（对齐 tap-97 语义）
    .scale(this.pressedSettingKey === title ?
      { x: DesignConstants.INTERACTION.SCALE_PRESSED_CARD, y: DesignConstants.INTERACTION.SCALE_PRESSED_CARD } :
      { x: 1, y: 1 })
    .animation({ curve: curves.springMotion(
      DesignConstants.INTERACTION.SPRING_RESPONSE_PRESS,
      DesignConstants.INTERACTION.SPRING_DAMPING_PRESS
    ) })
    .onTouch((event: TouchEvent) => {
      if (event.type === TouchType.Down) {
        this.pressedSettingKey = title;
      } else if (event.type === TouchType.Up || event.type === TouchType.Cancel) {
        this.pressedSettingKey = '';
      }
    })
    .onClick(onClick)
  }
}

@Builder
export function AdvancedSettingsPageBuilder() {
  AdvancedSettingsPage()
}
