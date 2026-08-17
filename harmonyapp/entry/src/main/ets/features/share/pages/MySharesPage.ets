import { KEY_IS_LOGGED_IN } from '../../../core/state/GlobalStateManager';
import { PageShellContainer } from '../../../components/PageShellContainer';
import type { BreakpointType } from '../../../core/responsive/Breakpoint';
import { ApiClient } from '../../../core/api/ApiClient';
import { ShareListItem, FetchCreatedSharesRequest, ShareType } from '../../../core/types/api';
import { DefaultConfig, PageNames } from '../../../constants/AppConstants';
import { DesignConstants } from '../../../constants/DesignConstants';
import { SystemToast } from '../../../components/SystemToast';
import { SystemLoadingView } from '../../../components/SystemLoadingView';
import { EmptyState, LoadingMoreIndicator, RefreshIndicator } from '../../../components/EmptyState';
import { SystemPageHeader } from '../../../components/SystemPageHeader';
import { SystemButton } from '../../../components/SystemButton';
import { AppIcon, IconButton, IconSize } from '../../../components/SystemIcon';
import { AppRouter } from '../../../core/router/AppRouter';
import { BasicDataSource } from '../../../core/utils/BasicDataSource';
import { SystemUiController } from '../../../core/utils/SystemUiController';
import { SafeAreaUtils } from '../../../core/utils/SafeAreaUtils';
import { LoginPromptView } from '../../../components/LoginPromptView';
import { MediaUrlUtil } from '../../../core/utils/MediaUrlUtil';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { KEY_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';

@Preview
@Component
export struct MySharesPage {
  @StorageProp('currentBreakpoint') currentBreakpoint: BreakpointType = 'sm';
  pathStack: NavPathStack = new NavPathStack();
  @StorageLink(KEY_IS_LOGGED_IN) @Watch('onLoginStateChanged') isLoggedIn: boolean = false;
  @StorageLink('safeAreaTop') safeAreaTop: number = 0;
  @StorageLink('safeAreaBottom') safeAreaBottom: number = 0;
  @StorageLink(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'vibrant_light';

  private sharesDataSource: BasicDataSource<ShareListItem> = new BasicDataSource<ShareListItem>([]);
  @State private shares: ShareListItem[] = [];
  @State private isLoading: boolean = true;
  @State private isRefreshing: boolean = false;
  @State private isLoadingMore: boolean = false;
  @State private errorMessage: string = '';
  @State private hasMore: boolean = false;
  @State private nextCursor: string = '';
  @State private revokingShareId: string = '';
  private readonly pageLimit: number = DefaultConfig.PAGE_SIZE;

  private isDestroyed: boolean = false;

  async aboutToAppear() {
    SystemUiController.syncWithCurrentTheme();
    if (this.isLoggedIn) {
      await this.loadShares();
    } else {
      this.resetState();
    }
  }

  aboutToDisappear(): void {
    this.isDestroyed = true;
  }

  private async onLoginStateChanged() {
    if (this.isDestroyed) return;
    if (this.isLoggedIn) {
      await this.loadShares();
    } else {
      this.resetState();
    }
  }

  private resetState(): void {
    this.shares = [];
    this.sharesDataSource.clear();
    this.hasMore = false;
    this.nextCursor = '';
    this.errorMessage = '';
    this.isLoading = false;
    this.isRefreshing = false;
    this.isLoadingMore = false;
  }

  private getCurrentThemePalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  private getSurfaceBackground(): string {
    return this.getCurrentThemePalette().surfaceBackground;
  }

  private getSurfaceSubtle(): string {
    return this.getCurrentThemePalette().surfaceSubtle;
  }

  private getSurfaceBorder(): string {
    return this.getCurrentThemePalette().borderColor;
  }

  private async loadShares(refresh: boolean = false): Promise<void> {
    if (this.isDestroyed) return;
    if (!this.isLoggedIn) {
      this.resetState();
      return;
    }

    if (!refresh) {
      this.isLoading = true;
    }
    this.errorMessage = '';

    try {
      const request: FetchCreatedSharesRequest = {
        pagination: {
          limit: this.pageLimit
        }
      };
      const response = await ApiClient.listMyShares(request);
      if (this.isDestroyed) return;
      if (response.code === 0) {
        const items = response.data?.items ?? [];
        this.shares = items;
        this.sharesDataSource.setData(items);
        this.hasMore = !!response.data?.next_cursor;
        this.nextCursor = response.data?.next_cursor ?? '';
      } else {
        this.handleLoadError(response.message || '加载我的分享失败');
      }
    } catch (error) {
      const message = error instanceof Error ? error.message : '加载我的分享失败';
      this.handleLoadError(message);
    } finally {
      this.isLoading = false;
      this.isRefreshing = false;
    }
  }

  private handleLoadError(message: string): void {
    this.errorMessage = message;
    this.shares = [];
    this.sharesDataSource.clear();
  }

  private async refreshShares(): Promise<void> {
    if (this.isDestroyed) return;
    if (this.isRefreshing) {
      return;
    }
    this.isRefreshing = true;
    await this.loadShares(true);
  }

  private async loadMoreShares(): Promise<void> {
    if (this.isDestroyed) return;
    if (!this.isLoggedIn || !this.hasMore || this.isLoadingMore) {
      return;
    }

    this.isLoadingMore = true;
    try {
      const request: FetchCreatedSharesRequest = {
        pagination: {
          limit: this.pageLimit,
          cursor: this.nextCursor
        }
      };
      const response = await ApiClient.listMyShares(request);
      if (this.isDestroyed) return;
      if (response.code === 0) {
        const items = response.data?.items ?? [];
        if (items.length > 0) {
          this.shares = this.shares.concat(items);
          this.sharesDataSource.appendData(items);
        }
        this.hasMore = !!response.data?.next_cursor;
        this.nextCursor = response.data?.next_cursor ?? '';
      } else {
        SystemToast.showError(response.message || '加载更多分享失败', this.getUIContext());
      }
    } catch (error) {
      const message = error instanceof Error ? error.message : '加载更多分享失败';
      SystemToast.showError(message, this.getUIContext());
    } finally {
      this.isLoadingMore = false;
    }
  }

  private async revokeShare(share: ShareListItem): Promise<void> {
    if (this.isDestroyed) return;
    if (!this.isLoggedIn || this.revokingShareId === share.share_id) {
      return;
    }

    this.revokingShareId = share.share_id;
    try {
      const response = await ApiClient.revokeShare({ share_id: share.share_id });
      if (this.isDestroyed) return;
      if (response.code === 0) {
        this.shares = this.shares.filter(item => item.share_id !== share.share_id);
        this.sharesDataSource.setData(this.shares);
        SystemToast.showSuccess('分享已撤销', this.getUIContext());
      } else {
        SystemToast.showError(response.message || '撤销分享失败', this.getUIContext());
      }
    } catch (error) {
      const message = error instanceof Error ? error.message : '撤销分享失败';
      SystemToast.showError(message, this.getUIContext());
    } finally {
      this.revokingShareId = '';
    }
  }

  build() {
    NavDestination() {
  PageShellContainer({ variant: 'form', currentBreakpoint: this.currentBreakpoint, noHorizontalPadding: true }) {
    Column() {
      // 头部 —— 统一走 SystemPageHeader（居中标题 + 返回）
      SystemPageHeader({
        title: '我的分享',
        showBackButton: true,
        onBack: () => { AppRouter.pop(); },
        titleAlign: TextAlign.Center,
        titleWeight: DesignConstants.FONT.WEIGHT.MEDIUM
      })

      if (!this.isLoggedIn) {
        LoginPromptView({
          title: '登录后查看我的分享',
          subtitle: '统一管理已生成的分享链接并随时撤销',
          icon: AppIcon.SHARE,
          redirectUrl: PageNames.MY_SHARES,
          loginAction: '查看我的分享'
        })
          .layoutWeight(1)
      } else if (this.isLoading) {
        SystemLoadingView({ message: '正在加载我的分享...' })
          .layoutWeight(1)
      } else if (this.errorMessage && this.shares.length === 0) {
        EmptyState({
          icon: AppIcon.WARNING,
          message: this.errorMessage,
          actionLabel: '重试',
          onAction: () => { this.loadShares(); }
        })
          .layoutWeight(1)
      } else if (this.shares.length === 0) {
        Refresh({ refreshing: $$this.isRefreshing, builder: RefreshIndicator }) {
          EmptyState({ icon: AppIcon.SHARE, message: '暂无分享记录\n创建分享后即可在这里管理' })
        }
        .layoutWeight(1)
        .onRefreshing(() => { this.refreshShares(); })
      } else {
        Refresh({ refreshing: $$this.isRefreshing, builder: RefreshIndicator }) {
          this.buildShareList()
        }
        .layoutWeight(1)
        .onRefreshing(() => { this.refreshShares(); })
      }
    }
    .width('100%')
    .height('100%')
    .backgroundColor(this.getCurrentThemePalette().pageBackground)
    }
    }
    .hideTitleBar(true)
  }

  @Builder
  private buildShareList() {
    List() {
      LazyForEach(this.sharesDataSource, (share: ShareListItem) => {
        ListItem() {
          this.buildShareCard(share)
        }
      }, (share: ShareListItem) => share.share_id)

      if (this.isLoadingMore) {
        ListItem() {
          LoadingMoreIndicator()
        }
      }
    }
    .divider({ strokeWidth: 0 })
    .width('100%')
    .cachedCount(5)
    .padding({
      left: DesignConstants.SPACING.LG,
      right: DesignConstants.SPACING.LG,
      top: DesignConstants.SPACING.SM,
      bottom: SafeAreaUtils.resolveBottomPadding(this.safeAreaBottom, DesignConstants.SPACING.SM, DesignConstants.SPACING.SM)
    })
    .scrollBar(BarState.Auto)
    .onReachEnd(() => { this.loadMoreShares(); })
  }

  @Builder
  private buildShareCard(share: ShareListItem) {
    Column({ space: DesignConstants.SPACING.MD }) {
      Row({ space: DesignConstants.SPACING.SM }) {
        Column({ space: DesignConstants.SPACING.XS }) {
          Text(share.preview?.title ?? this.defaultShareTitle(share))
            .fontSize(DesignConstants.FONT.SIZE.LG)
            .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
            .fontColor(this.getCurrentThemePalette().textPrimary)
            .maxLines(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })

          Text(`分享代码 ${share.share_code}`)
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .fontColor(this.getCurrentThemePalette().textSecondary)
        }
        .layoutWeight(1)

        Text(this.shareStatusText(share))
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
          .fontColor(this.shareStatusColor(share))
          .backgroundColor(this.shareStatusBackground(share))
          .borderRadius(DesignConstants.RADIUS.MD)
          .padding({
            left: DesignConstants.SPACING.SM,
            right: DesignConstants.SPACING.SM,
            top: DesignConstants.SPACING.XS,
            bottom: DesignConstants.SPACING.XS
          })
      }

      if (share.preview?.subtitle) {
        Text(share.preview.subtitle)
          .fontSize(DesignConstants.FONT.SIZE.MD)
          .fontColor(this.getCurrentThemePalette().textSecondary)
          .maxLines(2)
          .textOverflow({ overflow: TextOverflow.Ellipsis })
      }

      if (share.preview?.cover_url) {
        Image(MediaUrlUtil.normalize(share.preview.cover_url))
          .width('100%')
          .height(DesignConstants.COMPONENT.SHARE_CARD_HEIGHT)
          .objectFit(ImageFit.Cover)
          .borderRadius(DesignConstants.RADIUS.LG)
          .backgroundColor(this.getSurfaceSubtle())
      }

      Column({ space: DesignConstants.SPACING.XS }) {
        Text(`类型：${this.formatShareType(share.share_type)}`)
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .fontColor(this.getCurrentThemePalette().textSecondary)

        Text(`创建：${this.formatDate(share.created_at)}`)
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .fontColor(this.getCurrentThemePalette().textSecondary)

        Text(`有效期：${this.formatExpiration(share.expires_at)}`)
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .fontColor(this.getCurrentThemePalette().textSecondary)
      }
      .padding({
        left: DesignConstants.SPACING.MD,
        right: DesignConstants.SPACING.MD,
        top: DesignConstants.SPACING.SM,
        bottom: DesignConstants.SPACING.SM
      })
      .backgroundColor(this.getSurfaceSubtle())
      .borderRadius(DesignConstants.RADIUS.MD)

      Row({ space: DesignConstants.SPACING.MD }) {
        Column({ space: DesignConstants.SPACING.HALF }) {
          Text('分享状态')
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .fontColor(this.getCurrentThemePalette().textSecondary)

          Text(share.is_revoked ? '已停止对外访问' : '对外链接可用')
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .fontColor(this.getCurrentThemePalette().textPrimary)
        }
        .layoutWeight(1)

        if (!share.is_revoked) {
          SystemButton({
            label: '撤销分享',
            variant: 'surface-danger',
            buttonSize: 'small',
            loading: this.revokingShareId === share.share_id,
            onPress: () => { this.revokeShare(share); }
          })
        } else {
          Text('已撤销')
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .fontColor(this.getCurrentThemePalette().textTertiary)
        }
      }
    }
    .width('100%')
    .padding(DesignConstants.SPACING.LG)
    .borderRadius(DesignConstants.RADIUS.LG)
    .shadow(this.getCurrentThemePalette().elevatedShadow)
    .backgroundColor(this.getSurfaceBackground())
    .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getSurfaceBorder() })
    .margin({ bottom: DesignConstants.SPACING.MD })
  }

  private defaultShareTitle(share: ShareListItem): string {
    return `分享 #${share.share_code}`;
  }

  private shareStatusText(share: ShareListItem): string {
    return share.is_revoked ? '已撤销' : '进行中';
  }

  private shareStatusColor(share: ShareListItem) {
    return share.is_revoked ? this.getCurrentThemePalette().textTertiary : this.getCurrentThemePalette().accentBackground;
  }

  private shareStatusBackground(share: ShareListItem): ResourceColor {
    return share.is_revoked
      ? AppThemeManager.getSoftStatusBackground('warning')
      : AppThemeManager.getSoftStatusBackground('info');
  }

  private formatShareType(type: ShareType): string {
    switch (type) {
      case 'user':
        return '用户资料';
      case 'group':
        return '群组';
      case 'moment':
        return '动态内容';
      case 'location':
        return '位置';
      default:
        return '分享';
    }
  }

  private formatDate(value: string): string {
    if (!value) {
      return '--';
    }
    try {
      const date = new Date(value);
      if (Number.isNaN(date.getTime())) {
        return value;
      }
      const year = date.getFullYear();
      const month = this.padNumber(date.getMonth() + 1);
      const day = this.padNumber(date.getDate());
      const hour = this.padNumber(date.getHours());
      const minute = this.padNumber(date.getMinutes());
      return `${year}-${month}-${day} ${hour}:${minute}`;
    } catch (_) {
      return value;
    }
  }

  private formatExpiration(value?: string): string {
    if (!value) {
      return '未设置（永久有效）';
    }
    return this.formatDate(value);
  }

  private padNumber(num: number): string {
    return num < 10 ? `0${num}` : `${num}`;
  }
}

@Builder
export function MySharesPageBuilder() {
  MySharesPage()
}