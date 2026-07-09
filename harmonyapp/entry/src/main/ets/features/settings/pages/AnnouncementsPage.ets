import { AppRouter, AnnouncementsRouteParam } from '../../../core/router/AppRouter';
import { ResponsiveContainer } from '../../../components/ResponsiveContainer';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { KEY_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';
import { DesignConstants } from '../../../constants/DesignConstants';
import { SystemPageHeader } from '../../../components/SystemPageHeader';
import { SystemToast } from '../../../components/SystemToast';
import { AppIcon } from '../../../components/SystemIcon';
import { ApiClient } from '../../../core/api/ApiClient';
import { SystemAnnouncementItem } from '../../../core/types/api';
import { AnnouncementSummaryCard, SettingsSectionCard } from './SettingsSupport';
import { SystemUiController } from '../../../core/utils/SystemUiController';
import { RefreshIndicator, EmptyState } from '../../../components/EmptyState';
import { SystemLoadingView } from '../../../components/SystemLoadingView';
import { SafeAreaUtils } from '../../../core/utils/SafeAreaUtils';

@Preview
@Component
export struct AnnouncementsPage {
  pathStack: NavPathStack = new NavPathStack();
  @StorageLink(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'default_clean_light';
  @State announcements: SystemAnnouncementItem[] = [];
  @State loading: boolean = false;
  @State refreshing: boolean = false;
  private focusAnnouncementId: string = '';

  private isDestroyed: boolean = false;

  aboutToAppear(): void {
    SystemUiController.syncWithCurrentTheme();
  }

  aboutToDisappear(): void {
    this.isDestroyed = true;
  }

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  private applyAnnouncementFocus(items: SystemAnnouncementItem[]): SystemAnnouncementItem[] {
    if (!this.focusAnnouncementId) {
      return items;
    }
    const focused = items.find((item) => item.announcement_id === this.focusAnnouncementId);
    if (!focused) {
      return items;
    }
    return [focused, ...items.filter((item) => item.announcement_id !== this.focusAnnouncementId)];
  }

  private async loadAnnouncements(force: boolean): Promise<void> {
    if (this.isDestroyed) return;
    if (this.loading && !force) {
      return;
    }
    this.loading = true;
    try {
      const response = await ApiClient.getSystemAnnouncements({});
      if (this.isDestroyed) return;
      if (response.code === 0 && response.data) {
        this.announcements = this.applyAnnouncementFocus(response.data.items ?? []);
      } else {
        SystemToast.showError(response.message || '获取系统公告失败', this.getUIContext());
      }
    } catch (_) {
      SystemToast.showError('获取系统公告失败', this.getUIContext());
    } finally {
      this.loading = false;
      this.refreshing = false;
    }
  }

  build() {
    NavDestination() {
    ResponsiveContainer() {
      Column() {
        SystemPageHeader({
          title: '系统公告',
          showBackButton: true,
          onBack: () => { AppRouter.pop(); }
        })

        // 首次网络加载（无数据）时显示加载反馈，避免页面空白静默；下拉刷新由 Refresh 自身指示器反馈
        if (this.loading && this.announcements.length === 0) {
          SystemLoadingView({ message: '加载中...' })
            .layoutWeight(1)
            .width('100%')
        } else if (!this.loading && this.announcements.length === 0) {
          EmptyState({
            icon: AppIcon.NOTIFICATION,
            message: '暂无系统公告',
            actionLabel: '刷新',
            onAction: () => { this.loadAnnouncements(true); }
          })
            .layoutWeight(1)
            .width('100%')
        } else {
        Refresh({ refreshing: $$this.refreshing, builder: RefreshIndicator }) {
          Scroll() {
            Column({ space: 0 }) {
              Column() {
              SettingsSectionCard({ title: '平台通知' }) {
                AnnouncementSummaryCard({
                  items: this.announcements,
                  maxItems: 0,
                  showOpenAll: false
                })
              }
            }
            .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, top: DesignConstants.SPACING.LG, bottom: DesignConstants.SPACING.LG })
            }
            .width('100%')
            .padding({ bottom: SafeAreaUtils.resolveBottomPadding(0, DesignConstants.SPACING.XXL, DesignConstants.SPACING.LG) })
          }
        }
        .onRefreshing(() => {
          this.refreshing = true;
          this.loadAnnouncements(true);
        })
        .layoutWeight(1)
        .width('100%')
        }
      }
      .width('100%')
      .height('100%')
      .backgroundColor(this.getPalette().pageBackground)
    }
    }
    .hideTitleBar(true)
    .onReady((context: NavDestinationContext) => {
      this.pathStack = context.pathStack;
      const params = context.pathInfo?.param as AnnouncementsRouteParam | undefined;
      this.focusAnnouncementId = params?.focusAnnouncementId ?? '';
      const routeItems = params?.items as SystemAnnouncementItem[] | undefined;
      if (routeItems && routeItems.length > 0) {
        this.announcements = this.applyAnnouncementFocus(routeItems);
      } else {
        this.loadAnnouncements(false);
      }
    })
  }
}

@Builder
export function AnnouncementsPageBuilder() {
  AnnouncementsPage()
}
