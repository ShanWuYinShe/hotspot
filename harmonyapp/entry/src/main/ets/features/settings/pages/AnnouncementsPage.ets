import { AppRouter, AnnouncementsRouteParam } from '../../../core/router/AppRouter';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { KEY_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';
import { DesignConstants } from '../../../constants/DesignConstants';
import { SystemPageHeader } from '../../../components/SystemPageHeader';
import { SystemToast } from '../../../components/SystemToast';
import { ApiClient } from '../../../core/api/ApiClient';
import { SystemAnnouncementItem } from '../../../core/types/api';
import { AnnouncementSummaryCard, SettingsSectionCard } from './SettingsSupport';
import { SystemUiController } from '../../../core/utils/SystemUiController';
import { RefreshIndicator } from '../../../components/EmptyState';
import { SafeAreaUtils } from '../../../core/utils/SafeAreaUtils';

@Preview
@Component
export struct AnnouncementsPage {
  pathStack: NavPathStack = new NavPathStack();
  @StorageLink(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'default_clean_light';
  @State announcements: SystemAnnouncementItem[] = [];
  @State loading: boolean = false;
  @State refreshing: boolean = false;

  aboutToAppear(): void {
    SystemUiController.syncWithCurrentTheme();
  }

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  private async loadAnnouncements(force: boolean): Promise<void> {
    if (this.loading && !force) {
      return;
    }
    this.loading = true;
    try {
      const response = await ApiClient.getSystemAnnouncements({});
      if (response.code === 0 && response.data) {
        this.announcements = response.data.items ?? [];
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
      Column() {
        SystemPageHeader({
          title: '系统公告',
          showBackButton: true,
          onBack: () => { AppRouter.pop(); }
        })

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
      .width('100%')
      .height('100%')
      .backgroundColor(this.getPalette().pageBackground)
    }
    .hideTitleBar(true)
    .onReady((context: NavDestinationContext) => {
      this.pathStack = context.pathStack;
      const params = context.pathInfo?.param as AnnouncementsRouteParam | undefined;
      const routeItems = params?.items as SystemAnnouncementItem[] | undefined;
      if (routeItems && routeItems.length > 0) {
        this.announcements = routeItems;
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
