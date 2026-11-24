import { KEY_IS_LOGGED_IN } from '../../../core/state/GlobalStateManager';
import { PageShellContainer } from '../../../components/PageShellContainer';
import type { BreakpointType } from '../../../core/responsive/Breakpoint';
import { ApiClient } from '../../../core/api/ApiClient';
import { ShareListItem, FetchCreatedSharesRequest, ShareType } from '../../../core/types/api';
import { DefaultConfig, PageNames } from '../../../constants/AppConstants';
import { DesignConstants, lineHeightFor, textLineHeight } from '../../../constants/DesignConstants';
import { ListEnd } from '../../../components/ListEnd';
import { SystemToast } from '../../../components/SystemToast';
import { SystemInput } from '../../../components/SystemInput';
import { SystemLoadingView } from '../../../components/SystemLoadingView';
import { ListSkeleton } from '../../../components/states/ListSkeleton';
import { EmptyState, LoadingMoreIndicator, RefreshIndicator } from '../../../components/EmptyState';
import { SystemPageHeader } from '../../../components/SystemPageHeader';
import { SystemButton } from '../../../components/SystemButton';
import { SystemDialog } from '../../../components/SystemDialog';
import { AppIcon } from '../../../components/SystemIcon';
import { AppRouter } from '../../../core/router/AppRouter';
import { BasicDataSource } from '../../../core/utils/BasicDataSource';
import { SystemUiController } from '../../../core/utils/SystemUiController';
import { SafeAreaUtils } from '../../../core/utils/SafeAreaUtils';
import { LoginPromptView } from '../../../components/LoginPromptView';
import { MediaUrlUtil } from '../../../core/utils/MediaUrlUtil';
import { ImageUrlUtil } from '../../../core/utils/ImageUrlUtil';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { DEFAULT_RESOLVED_THEME_ID, KEY_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';
import { getErrorMessage } from '../../../core/network/ApiErrorUtil';
import { MotionUtil } from '../../../core/accessibility/MotionUtil';
import { LucideIcon } from '../../../components/LucideIcon';
import { BackToTopButton } from '../../../components/BackToTopButton';

// ── 长列表回顶（分享较多时滚到底部后一键回到顶部）──
// 注意：overlay builder 不能返回空（空 overlay 节点会拦截点击/滚动手势）；
// 这里用 Stack 同级子节点 + HitTestMode.Transparent，空白区域手势穿透。
@Preview
@Component
export struct MySharesPage {
  @State private showBackToTop: boolean = false;
  private backToTopScroller: Scroller = new Scroller();

  private updateBackToTop(): void {
    const offset: OffsetResult | undefined = this.backToTopScroller.currentOffset();
    const y: number = offset ? offset.yOffset : 0;
    const next: boolean = y > 240;
    if (next !== this.showBackToTop) {
      this.showBackToTop = next;
    }
  }

  private scrollToTop(): void {
    this.showBackToTop = false;
    this.backToTopScroller.scrollTo({
      xOffset: 0,
      yOffset: 0,
      animation: { duration: MotionUtil.reduced() ? 0 : 300, curve: Curve.EaseInOut }
    });
  }

  @Builder
  private buildBackToTopOverlay() {
    Row() {
      if (this.showBackToTop) {
        BackToTopButton({
          visible: true,
          onTap: () => { this.scrollToTop(); }
        })
      }
    }
    .width('100%')
    .height('100%')
    .padding({ right: DesignConstants.SPACING.LG, bottom: DesignConstants.SPACING.XXL })
    .justifyContent(FlexAlign.End)
    .alignItems(VerticalAlign.Bottom)
    .hitTestBehavior(HitTestMode.Transparent)
  }
  @StorageProp('currentBreakpoint') currentBreakpoint: BreakpointType = 'sm';
  pathStack: NavPathStack = new NavPathStack();
  @StorageLink(KEY_IS_LOGGED_IN) @Watch('onLoginStateChanged') isLoggedIn: boolean = false;
  @StorageLink('safeAreaTop') safeAreaTop: number = 0;
  @StorageLink('safeAreaBottom') safeAreaBottom: number = 0;
  @StorageLink(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = DEFAULT_RESOLVED_THEME_ID;

  private sharesDataSource: BasicDataSource<ShareListItem> = new BasicDataSource<ShareListItem>([]);
  @State private shares: ShareListItem[] = [];
  @State private coverLoadedKeys: string[] = [];
  @State private coverFailedKeys: string[] = [];
  @State private isLoading: boolean = true;
  @State private isRefreshing: boolean = false;
  @State private isLoadingMore: boolean = false;
  @State private errorMessage: string = '';
  @State private hasMore: boolean = false;
  @State private nextCursor: string = '';
  @State private revokingShareId: string = '';
  // 搜索：契约 share/fetch_created 无 keyword，搜索时先加载完全部分页（最多 10 页）再本地筛选。
  @State private searchKeyword: string = '';
  @State private searchResults: ShareListItem[] = [];
  @State private searchLoading: boolean = false;
  @State private searchError: string = '';
  private searchTimer: number = -1;
  private searchSeq: number = 0;
  @State private showRevokeConfirm: boolean = false;
  @State private pendingRevokeShare: ShareListItem | null = null;
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

  private coverKeyOf(share: ShareListItem): string {
    return share.preview?.cover_url ?? '';
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
        this.handleLoadError(getErrorMessage(response, '加载我的分享失败'));
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
    if (!this.isLoggedIn || !this.hasMore || this.isLoadingMore || this.searchKeyword.trim()) {
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
        SystemToast.showError(getErrorMessage(response, '加载更多分享失败'), this.getUIContext());
      }
    } catch (error) {
      const message = error instanceof Error ? error.message : '加载更多分享失败';
      SystemToast.showError(message, this.getUIContext());
    } finally {
      this.isLoadingMore = false;
    }
  }

  private matchShare(share: ShareListItem, keyword: string): boolean {
    const hay = [
      share.share_code,
      share.preview?.title ?? '',
      share.preview?.subtitle ?? '',
      this.formatShareType(share.share_type),
      this.shareStatusText(share)
    ].join(' ').toLowerCase();
    return hay.includes(keyword.toLowerCase());
  }

  private onSearchChange(value: string): void {
    this.searchKeyword = value;
    if (this.searchTimer >= 0) {
      clearTimeout(this.searchTimer);
      this.searchTimer = -1;
    }
    const keyword = value.trim();
    if (!keyword) {
      this.searchSeq++;
      this.searchResults = [];
      this.searchError = '';
      this.searchLoading = false;
      this.sharesDataSource.setData(this.shares);
      return;
    }
    this.searchTimer = setTimeout(() => {
      this.searchTimer = -1;
      void this.runSearch(keyword);
    }, 350);
  }

  private async runSearch(keyword: string): Promise<void> {
    const seq = ++this.searchSeq;
    this.searchLoading = true;
    this.searchError = '';
    const all: ShareListItem[] = [];
    let cursor = '';
    for (let page = 0; page < 10; page++) {
      const request: FetchCreatedSharesRequest = {
        pagination: { limit: this.pageLimit }
      };
      if (cursor) {
        request.pagination = { limit: this.pageLimit, cursor: cursor };
      }
      const response = await ApiClient.listMyShares(request);
      if (seq !== this.searchSeq) return;
      if (response.code !== 0) {
        this.searchError = getErrorMessage(response, '搜索分享失败');
        break;
      }
      all.push(...(response.data?.items ?? []));
      cursor = response.data?.next_cursor ?? '';
      if (!cursor) break;
    }
    if (this.isDestroyed || seq !== this.searchSeq) return;
    this.searchResults = all.filter((item: ShareListItem) => this.matchShare(item, keyword));
    this.sharesDataSource.setData(this.searchResults);
    this.searchLoading = false;
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
        if (this.searchKeyword.trim()) {
          this.searchResults = this.searchResults.filter(item => item.share_id !== share.share_id);
          this.sharesDataSource.setData(this.searchResults);
        } else {
          this.sharesDataSource.setData(this.shares);
        }
        SystemToast.showSuccess('分享已撤销', this.getUIContext());
      } else {
        SystemToast.showError(getErrorMessage(response, '撤销分享失败'), this.getUIContext());
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
    Stack() {
  PageShellContainer({ variant: 'default', currentBreakpoint: this.currentBreakpoint}) {
    Column() {
      // 头部 —— 统一走 SystemPageHeader（居中标题 + 返回）
      SystemPageHeader({
        title: '我的分享',
        // 对齐 web 个人分区页 integrated 移动端页头 18/600
        titleSize: DesignConstants.FONT.SIZE.LG_PLUS,
        titleWeight: DesignConstants.FONT.WEIGHT.SEMIBOLD,
        showBackButton: true,
        onBack: () => { AppRouter.pop(); },
      })

      // 搜索框（分享数量较多时快速定位；契约无 keyword：加载全部分页后本地筛选）
      if (this.isLoggedIn && this.shares.length > 0) {
        SystemInput({
          value: this.searchKeyword,
          placeholder: '搜索分享代码 / 标题 / 类型…',
          prefixIcon: AppIcon.SEARCH,
          showPrefixIcon: true,
          showClearButton: true,
          inputHeight: DesignConstants.COMPONENT.SEARCH_INPUT_HEIGHT,
          onChange: (value: string) => { this.onSearchChange(value); }
        })
          .margin({
            left: DesignConstants.SPACING.LG,
            right: DesignConstants.SPACING.LG,
            bottom: DesignConstants.SPACING.MD
          })
      }

      if (!this.isLoggedIn) {
        LoginPromptView({
          title: '登录后查看我的分享',
          subtitle: '统一管理已生成的分享链接并随时撤销',
          icon: AppIcon.SHARE,
          redirectUrl: PageNames.MY_SHARES,
          loginAction: '查看我的分享',
          highlights: [
            { icon: AppIcon.SHARE, text: '统一管理已生成的分享链接' },
            { icon: AppIcon.LOCK, text: '随时撤销不再需要的分享' },
            { icon: AppIcon.TIME, text: '查看分享有效期与访问状态' }
          ]
        })
          .layoutWeight(1)
      } else if (this.searchKeyword.trim()) {
        if (this.searchLoading) {
          SystemLoadingView({ message: '正在搜索分享…' })
            .layoutWeight(1)
        } else if (this.searchError) {
          EmptyState({
            bare: true,
            icon: AppIcon.WARNING,
            message: this.searchError,
            actionLabel: '重试',
            onAction: () => { void this.runSearch(this.searchKeyword.trim()); }
          })
            .layoutWeight(1)
        } else if (this.searchResults.length === 0) {
          EmptyState({
            bare: true,
            icon: AppIcon.SHARE,
            title: '没有找到分享',
            message: '换个分享代码 / 标题 / 类型关键词试试，或清空搜索词。',
            actionLabel: '清空搜索',
            onAction: () => { this.onSearchChange(''); }
          })
            .layoutWeight(1)
        } else {
          Refresh({ refreshing: $$this.isRefreshing, builder: RefreshIndicator }) {
            this.buildShareList()
          }
          .layoutWeight(1)
          .pullToRefresh(false)
          .refreshOffset(64)
          .onRefreshing(() => { this.refreshShares(); })
        }
      } else if (this.isLoading) {
        // 列表型首屏等待用骨架屏（对齐通知中心/关系页）
        ListSkeleton({ rowCount: 5 })
      } else if (this.errorMessage && this.shares.length === 0) {
        EmptyState({
          bare: true,
          icon: AppIcon.WARNING,
          message: this.errorMessage,
          actionLabel: '重试',
          onAction: () => { this.loadShares(); }
        })
          .layoutWeight(1)
      } else if (this.shares.length === 0) {
        Refresh({ refreshing: $$this.isRefreshing, builder: RefreshIndicator }) {
          EmptyState({
            spotlight: true,
            icon: AppIcon.SHARE,
            title: '暂无分享记录',
            message: '创建分享后即可在这里管理链接、查看有效期与访问状态。'
          })
        }
        .layoutWeight(1)
        .pullToRefresh(false)
        .refreshOffset(64)
        .onRefreshing(() => { this.refreshShares(); })
      } else {
        Refresh({ refreshing: $$this.isRefreshing, builder: RefreshIndicator }) {
          this.buildShareList()
        }
        .layoutWeight(1)
        .pullToRefresh(false)
        .refreshOffset(64)
        .onRefreshing(() => { this.refreshShares(); })
      }
    }
    .width('100%')
    .height('100%')
    }

    // 回顶浮动按钮（Stack 子节点，手势穿透）
    this.buildBackToTopOverlay()

    // 撤销分享二次确认
    SystemDialog({
      visible: this.showRevokeConfirm,
      title: '撤销分享',
      description: '撤销后分享链接将失效，他人将无法再通过该链接访问',
      confirmText: '撤销分享',
      confirmVariant: 'danger',
      onCancel: () => {
        this.showRevokeConfirm = false;
        this.pendingRevokeShare = null;
      },
      onConfirm: () => {
        const target = this.pendingRevokeShare;
        this.showRevokeConfirm = false;
        this.pendingRevokeShare = null;
        if (target) {
          this.revokeShare(target);
        }
      }
    })
    }
    }
    .hideTitleBar(true)
  }

  @Builder
  private buildShareList() {
    List({ scroller: this.backToTopScroller }) {
      ListItemGroup() {
        LazyForEach(this.sharesDataSource, (share: ShareListItem) => {
          ListItem() {
            this.buildShareCard(share)
          }
        }, (share: ShareListItem) => share.share_id)
      }
      .padding(DesignConstants.SPACING.SM)
      .backgroundColor(this.getCurrentThemePalette().surfaceIsland)
      .borderRadius(DesignConstants.SHAPE.CARD)
      .shadow(this.getCurrentThemePalette().surfaceShadow)
      .clip(true)
      .divider({
        strokeWidth: DesignConstants.BORDER_WIDTH.HAIRLINE,
        color: this.getSurfaceBorder(),
        startMargin: DesignConstants.SPACING.MD,
        endMargin: DesignConstants.SPACING.MD
      })

      if (this.searchKeyword.trim()) {
        if (this.searchResults.length > 0) {
          ListItem() {
            ListEnd()
          }
        }
      } else if (this.isLoadingMore) {
        ListItem() {
          LoadingMoreIndicator()
        }
      } else if (!this.hasMore && this.shares.length > 0) {
        ListItem() {
          ListEnd()
        }
      }
    }
    .divider({ strokeWidth: 0 })
    .width('100%')
    .edgeEffect(EdgeEffect.None)
    .cachedCount(5)
    // 左右留白由 PageShellContainer gutter 承担（SKILL:44-48），List 不再自加横向外边距
    .padding({
      top: DesignConstants.SPACING.SM,
      bottom: SafeAreaUtils.resolveBottomPadding(this.safeAreaBottom, DesignConstants.SPACING.SM, DesignConstants.SPACING.SM)
    })
    .scrollBar(BarState.Off)
    .onReachEnd(() => { if (!this.searchKeyword.trim()) this.loadMoreShares(); })
    .onDidScroll(() => { this.updateBackToTop(); })
  }

  @Builder
  private buildShareCard(share: ShareListItem) {
    Column({ space: DesignConstants.SPACING.MD }) {
      Row({ space: DesignConstants.SPACING.SM }) {
        Column({ space: DesignConstants.SPACING.XS }) {
          Text(share.preview?.title ?? this.defaultShareTitle(share))
            .fontSize(DesignConstants.FONT.SIZE.LG)
            .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.LG))
            .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
            .fontColor(this.getCurrentThemePalette().textPrimary)
            .maxLines(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })

          Text(`分享代码 ${share.share_code}`)
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.SM))
            .fontColor(this.getCurrentThemePalette().textSecondary)
        }
        .layoutWeight(1)

        Text(this.shareStatusText(share))
          .fontSize(DesignConstants.FONT.SIZE.XS2)
          .lineHeight(DesignConstants.FONT.SIZE.XS2)
          .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
          .fontColor(this.shareStatusColor(share))
          .backgroundColor(this.shareStatusBackground(share))
          .borderRadius(DesignConstants.RADIUS.FULL)
          .padding({
            left: DesignConstants.SPACING.SM2,
            right: DesignConstants.SPACING.SM2,
            top: DesignConstants.SPACING.XS,
            bottom: DesignConstants.SPACING.XS
          })
      }

      if (share.preview?.subtitle) {
        Text(share.preview.subtitle)
          .fontSize(DesignConstants.FONT.SIZE.MD)
          .fontColor(this.getCurrentThemePalette().textSecondary)
          // 多行副标题走比例制行高（SKILL:22-27）
          .lineHeight(lineHeightFor(DesignConstants.FONT.SIZE.MD, DesignConstants.FONT.LINE_RATIO.CARD))
          .maxLines(2)
          .textOverflow({ overflow: TextOverflow.Ellipsis })
      }

      if (share.preview?.cover_url) {
        // R8：占位 icon 常驻底层 + 图片渐显 + 失败保留占位
        Stack() {
          Column() {
            LucideIcon({
              icon: AppIcon.IMAGES,
              iconSize: DesignConstants.COMPONENT.ICON_SIZE_MD,
              color: this.getCurrentThemePalette().textTertiary
            })
          }
          .width('100%')
          .height('100%')
          .justifyContent(FlexAlign.Center)
          .backgroundColor(this.getSurfaceSubtle())
          if (this.coverFailedKeys.indexOf(share.preview.cover_url ?? '') < 0) {
            Image(ImageUrlUtil.optimized(MediaUrlUtil.normalize(this.coverKeyOf(share)), 480))
              .width('100%')
              .height('100%')
              .objectFit(ImageFit.Cover)
              .opacity(this.coverLoadedKeys.indexOf(this.coverKeyOf(share)) >= 0 || MotionUtil.reduced() ? 1 : 0)
              .animation({
                duration: MotionUtil.reduced() ? 0 : DesignConstants.ANIMATION.DURATION_SHORT,
                curve: DesignConstants.ANIMATION.CURVE_EASE_OUT
              })
              .onComplete(() => {
                if (this.coverLoadedKeys.indexOf(this.coverKeyOf(share)) >= 0) return;
                const next: string[] = this.coverLoadedKeys.slice();
                next.push(this.coverKeyOf(share));
                this.coverLoadedKeys = next;
              })
              .onError(() => {
                if (this.coverFailedKeys.indexOf(this.coverKeyOf(share)) >= 0) return;
                const next: string[] = this.coverFailedKeys.slice();
                next.push(this.coverKeyOf(share));
                this.coverFailedKeys = next;
              })
          }
        }
        .width('100%')
        .height(DesignConstants.COMPONENT.SHARE_CARD_HEIGHT)
        .borderRadius(DesignConstants.RADIUS.LG)
        .clip(true)
        .accessibilityText('分享内容封面图')
      }

      Column({ space: DesignConstants.SPACING.XS }) {
        Text(`类型：${this.formatShareType(share.share_type)}`)
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.SM))
          .fontColor(this.getCurrentThemePalette().textSecondary)

        Text(`创建：${this.formatDate(share.created_at)}`)
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.SM))
          .fontColor(this.getCurrentThemePalette().textSecondary)

        Text(`有效期：${this.formatExpiration(share.expires_at)}`)
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.SM))
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
            .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.SM))
            .fontColor(this.getCurrentThemePalette().textSecondary)

          Text(share.is_revoked ? '已停止对外访问' : '对外链接可用')
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.SM))
            .fontColor(this.getCurrentThemePalette().textPrimary)
        }
        .layoutWeight(1)

        if (!share.is_revoked) {
          SystemButton({
            label: '撤销分享',
            variant: 'surface-danger',
            buttonSize: 'small',
            loading: this.revokingShareId === share.share_id,
            onPress: () => {
              this.pendingRevokeShare = share;
              this.showRevokeConfirm = true;
            }
          })
        } else {
          Text('已撤销')
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.SM))
            .fontColor(this.getCurrentThemePalette().textTertiary)
        }
      }
    }
    .width('100%')
    .padding({
      left: DesignConstants.SPACING.MD,
      right: DesignConstants.SPACING.MD,
      top: DesignConstants.SPACING.MD,
      bottom: DesignConstants.SPACING.MD
    })
    // 行级卡片（圆角 + elevatedShadow + 白底 + 行间距）移除：分组外框与分隔线由
    // ListItemGroup 承担（对齐前端 ListContainer 分组列表）
  }

  private defaultShareTitle(share: ShareListItem): string {
    return `分享 #${share.share_code}`;
  }

  private shareStatusText(share: ShareListItem): string {
    return share.is_revoked ? '已撤销' : '进行中';
  }

  private shareStatusColor(share: ShareListItem) {
    // R12：软底状态文字用 on-soft 色（对齐 web text-status-*-on-soft；状态语义保留）
    return share.is_revoked
      ? this.getCurrentThemePalette().textStatusWarningOnSoft
      : this.getCurrentThemePalette().textStatusInfoOnSoft;
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