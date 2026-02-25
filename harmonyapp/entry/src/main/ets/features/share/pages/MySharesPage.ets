import { KEY_IS_LOGGED_IN } from '../../../core/state/GlobalStateManager';
import { ApiClient } from '../../../core/api/ApiClient';
import { ShareListItem, ShareListRequest, ShareType } from '../../../core/types/api';
import { DefaultConfig } from '../../../constants/AppConstants';
import { DesignConstants } from '../../../constants/DesignConstants';
import { SystemToast } from '../../../components/SystemToast';
import { SystemLoadingView } from '../../../components/SystemLoadingView';
import { EmptyState, LoadingMoreIndicator, RefreshIndicator } from '../../../components/EmptyState';
import { AppIcon, IconButton, IconSize } from '../../../components/SystemIcon';
import { AppRouter } from '../../../core/router/AppRouter';
import { BasicDataSource } from '../../../core/utils/BasicDataSource';
import { SystemUiController } from '../../../core/utils/SystemUiController';
import { SafeAreaUtils } from '../../../core/utils/SafeAreaUtils';
import { LoginPromptView } from '../../../components/LoginPromptView';

@Component
export struct MySharesPage {
  pathStack: NavPathStack = new NavPathStack();
  @StorageLink(KEY_IS_LOGGED_IN) @Watch('onLoginStateChanged') isLoggedIn: boolean = false;
  @StorageLink('safeAreaTop') safeAreaTop: number = 0;
  @StorageLink('safeAreaBottom') safeAreaBottom: number = 0;

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

  async aboutToAppear() {
    SystemUiController.useSurface({
      baseColor: DesignConstants.COLOR.PRIMARY_BG_LIGHT, preferDarkContent: true
    });
    if (this.isLoggedIn) {
      await this.loadShares();
    } else {
      this.resetState();
    }
  }

  private async onLoginStateChanged() {
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

  private getHeaderTopPadding(): number {
    return SafeAreaUtils.resolveTopPadding(this.safeAreaTop);
  }

  private async loadShares(refresh: boolean = false): Promise<void> {
    if (!this.isLoggedIn) {
      this.resetState();
      return;
    }

    if (!refresh) {
      this.isLoading = true;
    }
    this.errorMessage = '';

    try {
      const request: ShareListRequest = {
        pagination: {
          limit: this.pageLimit
        }
      };
      const response = await ApiClient.listMyShares(request);
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
    if (this.isRefreshing) {
      return;
    }
    this.isRefreshing = true;
    await this.loadShares(true);
  }

  private async loadMoreShares(): Promise<void> {
    if (!this.isLoggedIn || !this.hasMore || this.isLoadingMore) {
      return;
    }

    this.isLoadingMore = true;
    try {
      const request: ShareListRequest = {
        pagination: {
          limit: this.pageLimit,
          cursor: this.nextCursor
        }
      };
      const response = await ApiClient.listMyShares(request);
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
    if (!this.isLoggedIn || this.revokingShareId === share.share_id) {
      return;
    }

    this.revokingShareId = share.share_id;
    try {
      const response = await ApiClient.revokeShare({ share_id: share.share_id });
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
    Column() {
      Row() {
        IconButton({
          icon: AppIcon.BACK,
          iconSize: IconSize.LARGE,
          iconColor: $r('sys.color.ohos_id_color_text_primary'),
          bgColor: Color.Transparent,
          onButtonClick: () => AppRouter.pop()
        })

        Text('我的分享')
          .fontSize(DesignConstants.FONT.SIZE.XL)
          .fontWeight(FontWeight.Medium)
          .fontColor($r('sys.color.ohos_id_color_text_primary'))
          .layoutWeight(1)
          .textAlign(TextAlign.Center)

        Blank().width(48)
      }
      .width('100%')
      .alignItems(VerticalAlign.Center)
      .padding({
        left: DesignConstants.SPACING.LG,
        right: DesignConstants.SPACING.LG,
        top: this.getHeaderTopPadding(),
        bottom: DesignConstants.SPACING.SM
      })
      .backgroundColor($r('app.color.bg_secondary'))
      .expandSafeArea([SafeAreaType.SYSTEM], [SafeAreaEdge.TOP])

      if (!this.isLoggedIn) {
        LoginPromptView({
          title: '登录后查看我的分享',
          subtitle: '统一管理已生成的分享链接并随时撤销',
          icon: AppIcon.SHARE
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
    .backgroundColor($r('app.color.bg_primary'))
    }
  }

  @Builder
  private buildShareList() {
    List({ space: DesignConstants.SPACING.MD }) {
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
    .width('100%')
    .padding({
      left: DesignConstants.SPACING.LG,
      right: DesignConstants.SPACING.LG,
      bottom: SafeAreaUtils.resolveBottomPadding(this.safeAreaBottom, DesignConstants.SPACING.SM, DesignConstants.SPACING.SM)
    })
    .scrollBar(BarState.Auto)
    .onReachEnd(() => { this.loadMoreShares(); })
  }

  @Builder
  private buildShareCard(share: ShareListItem) {
    Column({ space: DesignConstants.SPACING.MD }) {
      Row({ space: DesignConstants.SPACING.SM }) {
        Column({ space: 4 }) {
          Text(share.preview?.title ?? this.defaultShareTitle(share))
            .fontSize(DesignConstants.FONT.SIZE.LG)
            .fontWeight(FontWeight.Medium)
            .fontColor($r('sys.color.ohos_id_color_text_primary'))
            .maxLines(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })

          Text(`分享代码 ${share.share_code}`)
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
        }
        .layoutWeight(1)

        Text(this.shareStatusText(share))
          .fontSize(12)
          .fontWeight(FontWeight.Medium)
          .fontColor(this.shareStatusColor(share))
          .backgroundColor(this.shareStatusBackground(share))
          .borderRadius(DesignConstants.RADIUS.SM)
          .padding({ left: 8, right: 8, top: 3, bottom: 3 })
      }

      if (share.preview?.subtitle) {
        Text(share.preview.subtitle)
          .fontSize(DesignConstants.FONT.SIZE.MD)
          .fontColor($r('sys.color.ohos_id_color_text_secondary'))
          .maxLines(2)
          .textOverflow({ overflow: TextOverflow.Ellipsis })
      }

      if (share.preview?.cover_url) {
        Image(share.preview.cover_url)
          .width('100%')
          .height(130)
          .objectFit(ImageFit.Cover)
          .borderRadius(DesignConstants.RADIUS.LG)
          .backgroundColor($r('app.color.bg_tertiary'))
      }

      Column({ space: 4 }) {
        Text(`类型：${this.formatShareType(share.share_type)}`)
          .fontSize(13)
          .fontColor($r('sys.color.ohos_id_color_text_secondary'))

        Text(`创建：${this.formatDate(share.created_at)}`)
          .fontSize(13)
          .fontColor($r('sys.color.ohos_id_color_text_secondary'))

        Text(`有效期：${this.formatExpiration(share.expires_at)}`)
          .fontSize(13)
          .fontColor($r('sys.color.ohos_id_color_text_secondary'))
      }

      Row({ space: DesignConstants.SPACING.MD }) {
        Column({ space: 2 }) {
          Text('分享状态')
            .fontSize(13)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))

          Text(share.is_revoked ? '已停止对外访问' : '对外链接可用')
            .fontSize(13)
            .fontColor($r('sys.color.ohos_id_color_text_primary'))
        }
        .layoutWeight(1)

        if (!share.is_revoked) {
          Button(this.revokingShareId === share.share_id ? '撤销中...' : '撤销分享')
            .type(ButtonType.Capsule)
            .backgroundColor($r('sys.color.ohos_id_color_emphasize'))
            .fontColor(Color.White)
            .fontSize(13)
            .height(34)
            .enabled(this.revokingShareId !== share.share_id)
            .onClick(() => this.revokeShare(share))
        } else {
          Text('已撤销')
            .fontSize(13)
            .fontColor($r('sys.color.ohos_id_color_warning'))
        }
      }
    }
    .width('100%')
    .padding(DesignConstants.SPACING.LG)
    .backgroundColor($r('app.color.bg_secondary'))
    .borderRadius(DesignConstants.RADIUS.LG)
  }

  private defaultShareTitle(share: ShareListItem): string {
    return `分享 #${share.share_code}`;
  }

  private shareStatusText(share: ShareListItem): string {
    return share.is_revoked ? '已撤销' : '进行中';
  }

  private shareStatusColor(share: ShareListItem) {
    return share.is_revoked ? $r('sys.color.ohos_id_color_warning') : $r('sys.color.ohos_id_color_emphasize');
  }

  private shareStatusBackground(share: ShareListItem): ResourceColor {
    return share.is_revoked ? '#FFFFF4F2' : '#FFE8F5FF';
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
