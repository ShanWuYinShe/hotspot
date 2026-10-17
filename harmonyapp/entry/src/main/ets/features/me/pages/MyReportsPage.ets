/**
 * 我的举报 - 独立列表页（对齐前端 /me/reports ReportsPage）
 * 消费 reports/fetch：举报人查看已提交举报的处理进度与结果（resolution）。
 * 页头（标题 + 「N 条」trailing）+ below 状态胶囊行（全部/待处理/已处理/已驳回）
 * + 列表（举报{目标} · {原因} + 状态胶囊 + 描述 + 处理结果 + 时间）。
 * 下拉刷新 + 「加载更多」按钮分页（对齐前端手动加载模式）。
 */

import { PageShellContainer } from '../../../components/PageShellContainer';
import { isDesktopForm, BreakpointType } from '../../../core/responsive/Breakpoint';
import { ApiClient } from '../../../core/api/ApiClient';
import { AppRouter } from '../../../core/router/AppRouter';
import { PageNames } from '../../../constants/AppConstants';
import { ReportItem, ReportStatus } from '../../../core/types/api';
import { SystemLoadingView } from '../../../components/SystemLoadingView';
import { DesignConstants, textLineHeight } from '../../../constants/DesignConstants';
import { ListEnd } from '../../../components/ListEnd';
import { AppIcon } from '../../../components/SystemIcon';
import { EmptyState, RefreshIndicator, LoadingMoreIndicator } from '../../../components/EmptyState';
import { SystemPageHeader } from '../../../components/SystemPageHeader';
import { TabsPillGroup, TabItem } from '../../../components/TabsPillGroup';
import { SystemButton } from '../../../components/SystemButton';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { DEFAULT_RESOLVED_THEME_ID, KEY_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';
import { getErrorMessage } from '../../../core/network/ApiErrorUtil';
import { formatDateTime } from '../../../core/utils/FormatUtil';

const PAGE_SIZE: number = 20;

/** 状态过滤 tab 键（'all' 不传 status，其余直传对应状态，对齐前端 statusParam） */
const TAB_KEYS: string[] = ['all', 'pending', 'resolved', 'dismissed'];
const TAB_LABELS: string[] = ['全部', '待处理', '已处理', '已驳回'];

@Preview
@Component
export struct MyReportsPage {
  @StorageProp('currentBreakpoint') currentBreakpoint: BreakpointType = 'sm';
  @StorageLink(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = DEFAULT_RESOLVED_THEME_ID;
  @StorageLink('safeAreaTop') safeAreaTop: number = 0;
  @StorageLink('safeAreaBottom') safeAreaBottom: number = 0;

  @State reports: ReportItem[] = [];
  @State isLoading: boolean = true;
  @State isRefreshing: boolean = false;
  @State isLoadingMore: boolean = false;
  @State hasMore: boolean = false;
  @State nextCursor: string = '';
  @State errorMessage: string = '';
  /** 当前状态过滤 tab（0=全部；其余为 TAB_KEYS 下标） */
  @State activeTabIndex: number = 0;
  private isDestroyed: boolean = false;

  private getCurrentThemePalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  private statusParam(): ReportStatus | undefined {
    const key = TAB_KEYS[this.activeTabIndex] ?? 'all';
    return key === 'all' ? undefined : (key as ReportStatus);
  }

  private targetLabel(targetType: string): string {
    switch (targetType) {
      case 'user': return '用户';
      case 'moment': return '瞬间';
      case 'comment': return '评论';
      case 'message': return '消息';
      case 'group': return '群组';
      default: return targetType;
    }
  }

  private reasonLabel(reason: string): string {
    switch (reason) {
      case 'spam': return '垃圾信息';
      case 'harassment': return '骚扰';
      case 'violence': return '暴力';
      case 'hate_speech': return '仇恨言论';
      case 'sexual_content': return '色情内容';
      case 'misinformation': return '不实信息';
      case 'impersonation': return '冒充他人';
      case 'self_harm': return '自伤';
      default: return '其他';
    }
  }

  private statusLabel(status: string): string {
    switch (status) {
      case 'pending': return '待处理';
      case 'resolved': return '已处理';
      case 'dismissed': return '已驳回';
      default: return status;
    }
  }

  aboutToAppear() {
    this.loadReports(false);
  }

  aboutToDisappear() {
    this.isDestroyed = true;
  }

  private async loadReports(refresh: boolean = false): Promise<void> {
    if (!refresh) { this.isLoading = true; }
    this.errorMessage = '';
    try {
      const status = this.statusParam();
      const response = await ApiClient.getMyReports({
        status: status,
        pagination: { limit: PAGE_SIZE, cursor: refresh ? '' : (this.nextCursor || '') }
      });
      if (this.isDestroyed) return;
      if (response.code === 0) {
        const items = response.data?.items ?? [];
        if (refresh) {
          this.reports = items;
        } else {
          const merged = [...this.reports, ...items];
          this.reports = merged;
        }
        this.hasMore = !!(response.data?.next_cursor);
        this.nextCursor = response.data?.next_cursor ?? '';
      } else {
        this.errorMessage = getErrorMessage(response, '加载失败');
        this.reports = [];
      }
    } catch (error) {
      if (!this.isDestroyed) {
        this.errorMessage = '加载举报记录失败';
      }
    } finally {
      this.isLoading = false;
      this.isRefreshing = false;
      this.isLoadingMore = false;
    }
  }

  private loadMore(): void {
    if (this.isLoadingMore || !this.hasMore) return;
    this.isLoadingMore = true;
    this.loadReports(false);
  }

  build() {
    NavDestination() {
    PageShellContainer({ variant: 'default', currentBreakpoint: this.currentBreakpoint }) {
      Column() {
        // ── 页头区块（标题 + 「N 条」trailing + below 状态胶囊行，分隔线在 below 之下）──
        Column() {
        SystemPageHeader({
          title: '我的举报',
          titleSize: DesignConstants.FONT.SIZE.XXL,
          titleWeight: DesignConstants.FONT.WEIGHT.MEDIUM,
          showBackButton: true,
          onBack: () => { AppRouter.pop(); },
          backgroundVariant: 'transparent',
          hasBelowContent: true,
          showDivider: false
        }) {
          // trailing：条数统计（对齐前端「N 条」，加载/错误态隐藏）
          if (!this.isLoading && !this.errorMessage) {
            Text(`${this.reports.length} 条`)
              .fontSize(DesignConstants.FONT.SIZE.XS)
              .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS))
              .fontColor(this.getCurrentThemePalette().textSecondary)
          }
        }

        // 状态胶囊行 = 页头 below（全部/待处理/已处理/已驳回）
        Scroll() {
        TabsPillGroup({
          tabs: TAB_LABELS.map((label: string): TabItem => ({ label })),
          freeWidth: true,
          activeIndex: this.activeTabIndex,
          actions: {
            onChange: (index: number) => {
              if (index === this.activeTabIndex) return;
              this.activeTabIndex = index;
              this.nextCursor = '';
              this.hasMore = false;
              this.loadReports(true);
            }
          }
        })
        }
        .scrollable(ScrollDirection.Horizontal)
        .scrollBar(BarState.Off)
        .align(Alignment.Start)
        .width('100%')
        .padding({ top: DesignConstants.SPACING.MD, bottom: DesignConstants.SPACING.SM })
        }
        .width('100%')
        .padding({ bottom: DesignConstants.SPACING.MD3 })
        .border({
          width: { bottom: DesignConstants.BORDER_WIDTH.HAIRLINE },
          color: this.getCurrentThemePalette().borderColor
        })

        // ── 内容区（对齐前端内容列 pt-4）──
        if (this.isLoading) {
          SystemLoadingView({ message: '加载举报记录…' })
            .layoutWeight(1)
        } else if (this.reports.length === 0 && this.errorMessage) {
          EmptyState({
            icon: AppIcon.WARNING,
            message: this.errorMessage,
            actionLabel: '重试',
            onAction: () => { this.loadReports(true); }
          })
            .layoutWeight(1)
        } else if (this.reports.length === 0) {
          EmptyState({
            icon: AppIcon.EMPTY,
            title: '暂无举报记录',
            message: '在动态、评论或聊天中遇到违规内容时，可通过「举报」提交，处理结果会在这里同步。',
            fillMode: true
          })
            .layoutWeight(1)
        } else {
          Refresh({ refreshing: $$this.isRefreshing, builder: RefreshIndicator }) {
            Scroll() {
              Column() {
                ForEach(this.reports, (item: ReportItem) => {
                  this.buildReportRow(item)
                }, (item: ReportItem) => item.report_id)

                // 分页：手动「加载更多」（对齐前端 LoadMoreButton）/ 到底提示
                if (this.hasMore) {
                  SystemButton({
                    label: '加载更多',
                    variant: 'surface',
                    buttonSize: 'default',
                    fullWidth: true,
                    onPress: () => { this.loadMore(); }
                  })
                    .margin({ top: DesignConstants.SPACING.LG })
                } else {
                  ListEnd()
                    .margin({ top: DesignConstants.SPACING.LG })
                }
              }
              .width('100%')
              // 内容列顶距 16（对齐前端 StandardPageLayout pt-4）
              .padding({ top: DesignConstants.SPACING.LG })
              .constraintSize({ minHeight: '100%' })
            }
            .scrollable(ScrollDirection.Vertical)
            .scrollBar(BarState.Off)
            .width('100%')
            .layoutWeight(1)
          }
          .layoutWeight(1)
          .onRefreshing(() => { this.loadReports(true); })
        }
      }
      .width('100%')
      .height('100%')
    }
    }
    .hideTitleBar(true)
  }

  @Builder
  private buildReportRow(item: ReportItem) {
    Column({ space: DesignConstants.SPACING.XS }) {
      Row({ space: DesignConstants.SPACING.SM2 }) {
        Text(`举报${this.targetLabel(item.target_type)} · ${this.reasonLabel(item.reason)}`)
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.SM))
          .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
          .fontColor(this.getCurrentThemePalette().textPrimary)
          .maxLines(1)
          .textOverflow({ overflow: TextOverflow.Ellipsis })
          .layoutWeight(1)

        // 状态胶囊（对齐前端：pending = accent-soft 底 + accent 字，其余 subtle 底 + 次要字）
        Text(this.statusLabel(item.status))
          .fontSize(DesignConstants.FONT.SIZE.XS2)
          .lineHeight(1)
          .fontColor(item.status === 'pending' ?
            this.getCurrentThemePalette().accentBackground : this.getCurrentThemePalette().textSecondary)
          .padding({
            left: DesignConstants.SPACING.SM2,
            right: DesignConstants.SPACING.SM2,
            top: DesignConstants.SPACING.XS,
            bottom: DesignConstants.SPACING.XS
          })
          .borderRadius(DesignConstants.RADIUS.FULL)
          .backgroundColor(item.status === 'pending' ?
            this.getCurrentThemePalette().accentMuted : this.getCurrentThemePalette().surfaceSubtle)
      }
      .width('100%')
      .alignItems(VerticalAlign.Center)

      if (item.description) {
        Text(item.description)
          .fontSize(DesignConstants.FONT.SIZE.XS)
          .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS))
          .fontColor(this.getCurrentThemePalette().textSecondary)
          .maxLines(2)
          .textOverflow({ overflow: TextOverflow.Ellipsis })
      }

      if (item.resolution) {
        Text(`处理结果：${item.resolution.action === 'resolve' ? '已确认处理' : '未构成违规'}${item.resolution.note ? ' · ' + item.resolution.note : ''}`)
          .fontSize(DesignConstants.FONT.SIZE.XS)
          .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS))
          .fontColor(this.getCurrentThemePalette().textSecondary)
          .maxLines(2)
          .textOverflow({ overflow: TextOverflow.Ellipsis })
      }

      Text(formatDateTime(item.created_at))
        .fontSize(DesignConstants.FONT.SIZE.XS2)
        .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS2))
        .fontColor(this.getCurrentThemePalette().textTertiary)
    }
    .alignItems(HorizontalAlign.Start)
    .width('100%')
    .padding({
      left: DesignConstants.SPACING.LG,
      right: DesignConstants.SPACING.LG,
      top: DesignConstants.SPACING.MD,
      bottom: DesignConstants.SPACING.MD
    })
    .border({
      width: { bottom: DesignConstants.BORDER_WIDTH.HAIRLINE },
      color: this.getCurrentThemePalette().borderColor
    })
  }
}

@Builder
export function MyReportsPageBuilder() {
  MyReportsPage()
}
