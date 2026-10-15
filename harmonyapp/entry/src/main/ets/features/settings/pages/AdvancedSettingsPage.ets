/**
 * 高级设置页面
 * 重构：使用原生 List 组件
 */

import { DesignConstants } from '../../../constants/DesignConstants';
import { PageShellContainer } from '../../../components/PageShellContainer';
import type { BreakpointType } from '../../../core/responsive/Breakpoint';
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
import { SettingsGroupLabel, SettingsRowDivider, SettingsSectionCard, SettingsEntryRow } from './SettingsSupport';

@Preview
@Component
export struct AdvancedSettingsPage {
  @StorageProp('currentBreakpoint') currentBreakpoint: BreakpointType = 'sm';
  pathStack: NavPathStack = new NavPathStack();
  @StorageLink('safeAreaTop') safeAreaTop: number = 0;
  @StorageLink(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = DEFAULT_RESOLVED_THEME_ID;
  @State private showClearCacheDialog: boolean = false;
  @State private isClearingCache: boolean = false;
  // 缓存条目数外显（Gemini 审查 r32：清理操作缺当前占用感知；条目数为 CacheManager 现成口径）
  @State private cacheItems: number = 0;

  aboutToAppear(): void {
    SystemUiController.syncWithCurrentTheme();
    this.refreshCacheSummary();
  }

  private refreshCacheSummary(): void {
    this.cacheItems = CacheManager.getInstance().getStats().size;
  }

  private getCacheSubtitle(): string {
    return this.cacheItems > 0 ? `清除本地缓存数据 · 内存缓存 ${this.cacheItems} 项` : '清除本地缓存数据';
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
      this.refreshCacheSummary();
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
        backgroundVariant: 'transparent'
      })

      Column() {
        // 对齐主设置页分组样板：eyebrow 分组 + 白卡发丝描边（SettingsSectionCard）+ 行间分隔线。
        // 行统一走 SettingsEntryRow（与主设置页同一样式：LG 左右内边距、圆形图标底、SurfaceIconButton 箭头）
        // ——原页内自绘 Builder 缺左右内边距致内容贴边，且图标底为方角，同域两套行样式（Gemini 审查 r32）。
        SettingsGroupLabel({ label: '工具' })
        Column() {
          SettingsSectionCard() {
            SettingsEntryRow({ item: {
              icon: AppIcon.CLEAN,
              title: '清理缓存',
              subtitle: this.getCacheSubtitle(),
              action: (): void => {
                this.requestClearCache();
              }
            } })

            // 调试面板仅 debug 构建可达：面板暴露 Token 状态与网络工具，
            // release 构建不应给普通用户入口（审查 高-1）。
            if (DEBUG_BUILD) {
              // 分隔线内缩至文字起点（LG，与行内边距同值），不再贯穿图标区留白（r32）
              SettingsRowDivider({ startMargin: DesignConstants.SPACING.LG, endMargin: DesignConstants.SPACING.LG })

              SettingsEntryRow({ item: {
                icon: AppIcon.DEBUG,
                title: '调试面板',
                subtitle: '网络诊断、状态检查',
                action: (): void => {
                  AppRouter.push(PageNames.DEBUG_PANEL);
                }
              } })
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
}

@Builder
export function AdvancedSettingsPageBuilder() {
  AdvancedSettingsPage()
}