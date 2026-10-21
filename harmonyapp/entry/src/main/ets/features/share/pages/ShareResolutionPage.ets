/**
 * 分享码解析中间页
 * 接收 share_code → 调 API 获取详情 → 根据类型路由跳转
 */
import { ApiClient } from '../../../core/api/ApiClient';
import { PageShellContainer } from '../../../components/PageShellContainer';
import type { BreakpointType } from '../../../core/responsive/Breakpoint';
import { ShareDetail, LocationSharePayload, UserSharePayload, GroupSharePayload, MomentSharePayload } from '../../../core/types/api';
import { AppRouter, ProfileRouteParam, GroupDetailRouteParam, MomentDetailRouteParam, ShareResolutionRouteParam } from '../../../core/router/AppRouter';
import { PageNames } from '../../../constants/AppConstants';
import { SystemToast } from '../../../components/SystemToast';
import { SystemLoadingView } from '../../../components/SystemLoadingView';
import { EmptyState } from '../../../components/EmptyState';
import { AppIcon } from '../../../components/SystemIcon';
import { DesignConstants } from '../../../constants/DesignConstants';
import { GlobalStateManager, KEY_BROWSE_ANCHOR } from '../../../core/state/GlobalStateManager';
import { LocationInfo } from '../../../core/location/LocationUtil';
import { SystemUiController } from '../../../core/utils/SystemUiController';
import { SafeAreaUtils } from '../../../core/utils/SafeAreaUtils';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { DEFAULT_RESOLVED_THEME_ID, KEY_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';
import { getErrorMessage } from '../../../core/network/ApiErrorUtil';
import { hilog } from '@kit.PerformanceAnalysisKit';

const DOMAIN_NUMBER: number = 0xFF31;
const TAG: string = 'ShareResolutionPage';

@Preview
@Component
export struct ShareResolutionPage {
  @StorageProp('currentBreakpoint') currentBreakpoint: BreakpointType = 'sm';
  pathStack: NavPathStack = new NavPathStack();
  @State isLoading: boolean = true;
  @State errorMessage: string = '';
  @State shareCode: string = '';
  @StorageLink('safeAreaTop') safeAreaTop: number = 0;
  @StorageLink('safeAreaBottom') safeAreaBottom: number = 0;
  @StorageLink(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = DEFAULT_RESOLVED_THEME_ID;

  private isDestroyed: boolean = false;

  aboutToAppear(): void {
    SystemUiController.syncWithCurrentTheme();
  }

  aboutToDisappear(): void {
    this.isDestroyed = true;
  }

  private getCurrentThemePalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  private async resolveShare(): Promise<void> {
    if (this.isDestroyed) return;
    if (!this.shareCode) {
      this.errorMessage = '缺少分享码';
      this.isLoading = false;
      return;
    }

    try {
      const response = await ApiClient.getShare({ share_code: this.shareCode });
      if (this.isDestroyed) return;
      if (response.code === 0 && response.data) {
        this.routeByType(response.data);
      } else {
        this.errorMessage = getErrorMessage(response, '分享不存在或已过期');
        this.isLoading = false;
      }
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `resolveShare failed: ${JSON.stringify(error)}`);
      this.errorMessage = '加载分享失败';
      this.isLoading = false;
    }
  }

  private routeByType(detail: ShareDetail): void {
    const shareType = detail.share.share_type;

    if (shareType === 'location') {
      const payload = detail.payload as LocationSharePayload;
      const loc = payload.shared_location;
      const anchor: LocationInfo = {
        latitude: loc.point.latitude,
        longitude: loc.point.longitude,
        locationName: loc.location_name
      };
      GlobalStateManager.setBrowseAnchor(anchor);
      SystemToast.showSuccess('浏览锚点已更新', this.getUIContext());
      AppRouter.clear();
    } else if (shareType === 'user') {
      const payload = detail.payload as UserSharePayload;
      const param = new ProfileRouteParam();
      param.userId = payload.shared_user_id;
      AppRouter.clear();
      AppRouter.push(PageNames.PROFILE, param);
    } else if (shareType === 'group') {
      const payload = detail.payload as GroupSharePayload;
      const param = new GroupDetailRouteParam();
      param.groupId = payload.shared_group_id;
      AppRouter.clear();
      AppRouter.push(PageNames.GROUP_DETAIL, param);
    } else if (shareType === 'moment') {
      const payload = detail.payload as MomentSharePayload;
      const param = new MomentDetailRouteParam();
      param.momentId = payload.shared_moment_id;
      AppRouter.clear();
      AppRouter.push(PageNames.MOMENT_DETAIL, param);
    } else {
      this.errorMessage = '不支持的分享类型';
      this.isLoading = false;
    }
  }

  build() {
    NavDestination() {
    PageShellContainer({ variant: 'form', currentBreakpoint: this.currentBreakpoint }) {
      Column() {
        if (this.isLoading) {
          SystemLoadingView({ message: '正在加载分享内容…' })
            .layoutWeight(1)
        } else {
          EmptyState({
            bare: true,
            icon: AppIcon.WARNING,
            message: this.errorMessage,
            actionLabel: '返回首页',
            onAction: () => { AppRouter.clear(); }
          })
            .layoutWeight(1)
        }
      }
      .width('100%')
      .height('100%')
      .padding({
        top: SafeAreaUtils.resolveTopPadding(this.safeAreaTop),
        bottom: SafeAreaUtils.resolveBottomPadding(this.safeAreaBottom, 0, 0)
      })
      }
    }
    .hideTitleBar(true)
    .onReady((context: NavDestinationContext) => {
      this.pathStack = context.pathStack;
      const param = context.pathInfo?.param as ShareResolutionRouteParam | undefined;
      this.shareCode = param?.share_code ?? '';
      this.resolveShare();
    })
  }
}

@Builder
export function ShareResolutionPageBuilder() {
  ShareResolutionPage()
}