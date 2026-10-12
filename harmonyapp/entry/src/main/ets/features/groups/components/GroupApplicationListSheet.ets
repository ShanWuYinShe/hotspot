import { DesignConstants } from '../../../constants/DesignConstants';
import { AppIcon } from '../../../components/SystemIcon';
import { SystemLoadingView } from '../../../components/SystemLoadingView';
import { EmptyState } from '../../../components/EmptyState';
import { LoadingMoreIndicator } from '../../../components/EmptyState';
import { ListEnd } from '../../../components/ListEnd';
import { GroupApplicationRow } from './GroupApplicationRow';
import { SafeAreaUtils } from '../../../core/utils/SafeAreaUtils';
import { BasicDataSource } from '../../../core/utils/BasicDataSource';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { DEFAULT_RESOLVED_THEME_ID, KEY_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';
import { GroupApplicationItem } from '../../../core/types/api';

/**
 * 入群申请列表面板（从 GroupDetailPage.buildApplicationListSheet 提取的子域组件）。
 *
 * 状态边界：列表数据/加载态经 @Prop 单向下发；数据源对象为页面持有实例的
 * 引用传递；批准/拒绝动作经回调携带 application 交回页面（权限与 API 在页面侧）。
 */
@Component
export struct GroupApplicationListSheet {
  @Prop applications: GroupApplicationItem[] = [];
  @Prop isLoadingApplications: boolean = false;
  @Prop applicationsErrorMessage: string = '';
  @Prop applicationsLoadingMore: boolean = false;
  @Prop applicationsHasMore: boolean = false;
  @Prop safeAreaBottom: number = 0;
  applicationsDataSource: BasicDataSource<GroupApplicationItem> = new BasicDataSource<GroupApplicationItem>([]);
  onRetryLoad: () => void = () => {};
  onLoadMore: () => void = () => {};
  handlingFor: (applicationId: string) => boolean = () => false;
  onApprove: (application: GroupApplicationItem) => void = () => {};
  onReject: (application: GroupApplicationItem) => void = () => {};

  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = DEFAULT_RESOLVED_THEME_ID;

  private getCurrentThemePalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  private getSurfaceBackground(): string {
    return this.getCurrentThemePalette().surfaceBackground;
  }

  build() {
    Column() {
      Row() {
        Text('入群申请')
          .fontSize(DesignConstants.FONT.SIZE.XL)
          .fontWeight(DesignConstants.FONT.WEIGHT.BOLD)
          .fontColor(this.getCurrentThemePalette().textPrimary)
          .layoutWeight(1)

        Text(`${this.applications.length} 条`)
          .fontSize(DesignConstants.FONT.SIZE.MD)
          .fontColor(this.getCurrentThemePalette().textSecondary)
      }
      .width('100%')
      .padding(DesignConstants.SPACING.LG)

      if (this.isLoadingApplications && this.applications.length === 0) {
        SystemLoadingView({ message: '加载入群申请中…' }).layoutWeight(1)
      } else if (this.applications.length === 0 && this.applicationsErrorMessage) {
        EmptyState({
          icon: AppIcon.WARNING,
          title: this.applicationsErrorMessage,
          actionLabel: '重试',
          fillMode: true,
          onAction: () => { this.onRetryLoad(); }
        })
      } else if (this.applications.length > 0) {
        List() {
          LazyForEach(this.applicationsDataSource, (application: GroupApplicationItem) => {
            ListItem() {
              GroupApplicationRow({
                application: application,
                handling: this.handlingFor(application.application_id),
                palette: this.getCurrentThemePalette(),
                surfaceBackground: this.getSurfaceBackground(),
                onApprove: () => { this.onApprove(application); },
                onReject: () => { this.onReject(application); }
              })
            }
          }, (application: GroupApplicationItem) => application.application_id)

          // 分页状态条：加载中原语 + 到底 ListEnd（原为手写 LoadingProgress + 「加载更多…」，
          // 样式与其它页不一致；四态契约要求「到底」有明确收尾）
          if (this.applicationsLoadingMore) {
            ListItem() {
              LoadingMoreIndicator()
            }
          } else if (!this.applicationsHasMore && this.applications.length > 0) {
            ListItem() {
              ListEnd()
            }
          }
        }
        .width('100%')
        .layoutWeight(1)
        // 横向留白由外层 PageShellContainer 统一提供，这里不再叠加
        .scrollBar(BarState.Off)
        .cachedCount(5)
        .onReachEnd(() => { this.onLoadMore(); })
      } else {
        EmptyState({
          icon: AppIcon.EMPTY,
          title: '暂无待处理申请',
          bare: true,
          fillMode: true
        })
      }
    }
    .padding({ bottom: SafeAreaUtils.resolveBottomPadding(this.safeAreaBottom, DesignConstants.SPACING.LG, 0) })
    .width('100%')
    .height('100%')
    .backgroundColor(this.getCurrentThemePalette().pageBackground)
  }
}
