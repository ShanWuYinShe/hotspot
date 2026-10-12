/**
 * 高级设置页面
 * 重构：使用原生 List 组件
 */

import { DesignConstants } from '../../../constants/DesignConstants';
import { PageShellContainer } from '../../../components/PageShellContainer';
import type { BreakpointType } from '../../../core/responsive/Breakpoint';
import { curves } from '@kit.ArkUI';
import { AppIcon } from '../../../components/SystemIcon';
import { SystemPageHeader } from '../../../components/SystemPageHeader';
import { SystemToast } from '../../../components/SystemToast';
import { AppRouter } from '../../../core/router/AppRouter';
import { PageNames, DEBUG_BUILD } from '../../../constants/AppConstants';
import { SystemUiController } from '../../../core/utils/SystemUiController';
import { SafeAreaUtils } from '../../../core/utils/SafeAreaUtils';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { DEFAULT_RESOLVED_THEME_ID, KEY_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';
import { SystemDialog } from '../../../components/SystemDialog';
import { CacheManager } from '../../../core/cache/CacheManager';
import { MessageCacheService } from '../../../core/database/MessageCacheService';
import { MomentCacheService } from '../../../core/database/MomentCacheService';
import { LucideIcon } from '../../../components/LucideIcon';
import { SettingsGroupLabel, SettingsRowDivider, SettingsSectionCard } from './SettingsSupport';

@Preview
@Component
export struct AdvancedSettingsPage {
  @StorageProp('currentBreakpoint') currentBreakpoint: BreakpointType = 'sm';
  pathStack: NavPathStack = new NavPathStack();
  @StorageLink('safeAreaTop') safeAreaTop: number = 0;
  @StorageLink(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = DEFAULT_RESOLVED_THEME_ID;
  @State private pressedSettingKey: string = '';
  @State private showClearCacheDialog: boolean = false;
  @State private isClearingCache: boolean = false;

  aboutToAppear(): void {
    SystemUiController.syncWithCurrentTheme();
  }

  private requestClearCache(): void {
    this.showClearCacheDialog = true;
  }

  private async confirmClearCache(): Promise<void> {
    this.showClearCacheDialog = false;
    if (this.isClearingCache) {
      return;
    }
    this.isClearingCache = true;
    try {
      // 运行时内存缓存（图片/网络等）
      CacheManager.getInstance().clearAll();
      // 本地消息与会话缓存（RDB）
      await MessageCacheService.getInstance().clearAll();
      // 本地动态缓存（RDB）
      await MomentCacheService.getInstance().clearAll();
      SystemToast.showSuccess('缓存已清理', this.getUIContext());
    } catch (error) {
      const message = error instanceof Error ? error.message : '未知错误';
      SystemToast.showError(`缓存清理失败：${message}`, this.getUIContext());
    } finally {
      this.isClearingCache = false;
    }
  }

  private getCurrentThemePalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  private getSurfaceBackground(): string {
    return this.getCurrentThemePalette().surfaceBackground;
  }

  build() {
    NavDestination() {
    Stack() {
  PageShellContainer({ variant: 'form', currentBreakpoint: this.currentBreakpoint}) {
    Column() {
      // 标题栏 —— 统一走 SystemPageHeader（居中标题 + 返回）
      SystemPageHeader({
        title: '高级设置',
        titleSize: DesignConstants.FONT.SIZE.XXL,
        titleWeight: DesignConstants.FONT.WEIGHT.MEDIUM,
        showBackButton: true,
        onBack: () => { AppRouter.pop(); },
        titleAlign: TextAlign.Center,
        backgroundVariant: 'transparent'
      })

      Column() {
        // 对齐主设置页分组样板：eyebrow 分组 + 白卡发丝描边（SettingsSectionCard）+ 行间分隔线
        SettingsGroupLabel({ label: '工具' })
        Column() {
          SettingsSectionCard() {
            this.SettingItem(AppIcon.CLEAN, '清理缓存', '清除本地缓存数据', () => {
              this.requestClearCache();
            })

            // 调试面板仅 debug 构建可达：面板暴露 Token 状态与网络工具，
            // release 构建不应给普通用户入口（审查 高-1）。
            if (DEBUG_BUILD) {
              SettingsRowDivider()

              this.SettingItem(AppIcon.DEBUG, '调试面板', '网络诊断、状态检查', () => {
                AppRouter.push(PageNames.DEBUG_PANEL);
              })
            }
          }
        }
        .padding({
          left: DesignConstants.SPACING.LG,
          right: DesignConstants.SPACING.LG,
          bottom: DesignConstants.SPACING.LG
        })
      }
      .width('100%')
      .layoutWeight(1)
      .alignItems(HorizontalAlign.Start)
      .backgroundColor(DesignConstants.COLOR.TRANSPARENT)
    }
    .width('100%')
    .height('100%')
    }

    SystemDialog({
      visible: this.showClearCacheDialog,
      title: '清理缓存',
      description: '将清除应用内存缓存、本地消息与动态缓存数据，清理后相关内容需重新加载，且不可恢复。账号信息与登录状态不受影响。',
      confirmText: '清理',
      cancelText: '取消',
      confirmVariant: 'danger',
      onCancel: () => { this.showClearCacheDialog = false; },
      onConfirm: () => { this.confirmClearCache(); }
    })
    }
    .width('100%')
    .height('100%')
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
        LucideIcon({
            icon: icon,
            iconSize: DesignConstants.COMPONENT.ICON_SIZE_XS,
            color: this.getCurrentThemePalette().accentBackground
          })
      }
      .width(DesignConstants.COMPONENT.BRAND_ICON_SIZE)
      .height(DesignConstants.COMPONENT.BRAND_ICON_SIZE)
      .borderRadius(DesignConstants.SHAPE.ICON_TILE)
      .backgroundColor(this.getCurrentThemePalette().accentMuted)
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

      LucideIcon({
          icon: AppIcon.ARROW_RIGHT,
          iconSize: DesignConstants.COMPONENT.ICON_SIZE_2XS,
          color: this.getCurrentThemePalette().textSecondary
        })
    }
    .width('100%')
    .padding({ top: DesignConstants.SPACING.MD, bottom: DesignConstants.SPACING.MD })
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