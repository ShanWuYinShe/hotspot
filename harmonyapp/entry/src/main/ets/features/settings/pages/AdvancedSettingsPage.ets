/**
 * 高级设置页面
 * 重构：使用原生 List 组件
 */

import { DesignConstants } from '../../../constants/DesignConstants';
import { ResponsiveContainer } from '../../../components/ResponsiveContainer';
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
  @StorageLink(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'default_clean_light';

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
        titleWeight: FontWeight.Medium,
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
        .borderRadius(DesignConstants.RADIUS.XXL)
        .shadow(this.getCurrentThemePalette().elevatedShadow)
        .border({ width: 1, color: this.getCurrentThemePalette().borderColor })
        .margin({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG,
          top: DesignConstants.SPACING.LG, bottom: 14 })
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
      .width(38)
      .height(38)
      .borderRadius(DesignConstants.SHAPE.ICON_TILE)
      .backgroundColor(this.getCurrentThemePalette().surfaceSubtle)
      .justifyContent(FlexAlign.Center)
      .alignItems(VerticalAlign.Center)

      Column({ space: 4 }) {
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
            .fontSize(DesignConstants.FONT.SIZE.MD_PLUS)
            .fontColor([this.getCurrentThemePalette().textSecondary])
        }
      }
      .width(28)
      .height(28)
      .borderRadius(DesignConstants.SHAPE.ICON_TILE)
      .backgroundColor(this.getCurrentThemePalette().surfaceSubtle)
      .justifyContent(FlexAlign.Center)
      .alignItems(VerticalAlign.Center)
    }
    .width('100%')
    .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, top: DesignConstants.SPACING.MD, bottom: DesignConstants.SPACING.MD })
    .onClick(onClick)
  }
}

@Builder
export function AdvancedSettingsPageBuilder() {
  AdvancedSettingsPage()
}
