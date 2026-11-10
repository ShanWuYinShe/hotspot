/**
 * 管理后台概览页（移动端降级方案）
 *
 * 移动端仅提供「权限感知 + 全站 KPI + 模块入口引导」。复杂表格化管理（批量操作、
 * 角色配置、审计查询等）引导到 Web 端完成——移动端触屏不适合密集表格操作。
 *
 * 对齐前端 admin 概览移动端（routes/(app)/admin/+page.svelte）：
 * 页头 18/600 + 「平台管理」副标题、模块导航胶囊、身份行 + KPI 数字条、
 * 模块入口卡（图标砖 + 标题 + 说明 + chevron）、Web 端引导卡。
 */

import { ApiClient } from '../../../core/api/ApiClient';
import { getErrorMessage } from '../../../core/network/ApiErrorUtil';
import { ApiError } from '../../../core/network/ApiError';
import { ErrorCode } from '../../../core/constants/ErrorCodes';
import { AdminPermissions, AdminRoleBasic, AdminGeoStatsTotals } from '../../../core/types/api';
import { PageShellContainer } from '../../../components/PageShellContainer';
import { SystemPageHeader } from '../../../components/SystemPageHeader';
import { SystemLoadingView } from '../../../components/SystemLoadingView';
import { EmptyState } from '../../../components/EmptyState';
import { Skeleton } from '../../../components/states/Skeleton';
import { FocusRing } from '../../../core/accessibility/FocusRing';
import { MotionUtil } from '../../../core/accessibility/MotionUtil';
import { SystemToast } from '../../../components/SystemToast';
import { DesignConstants, lineHeightFor, textLineHeight } from '../../../constants/DesignConstants';
import { AppIcon } from '../../../components/SystemIcon';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { DEFAULT_RESOLVED_THEME_ID, KEY_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';
import { BreakpointType } from '../../../core/responsive/Breakpoint';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { LucideIcon } from '../../../components/LucideIcon';
import { TabsPillGroup, TabItem } from '../../../components/TabsPillGroup';

const DOMAIN: number = 0xFF20;
const TAG: string = 'AdminOverviewPage';

interface AdminModule {
  icon: AppIcon;
  title: string;
  desc: string;
  perm: string;
}

interface KpiItem {
  label: string;
  value: string;
  warning: boolean;
}

/** 对齐前端 admin/+page.svelte MODULES（移动端单列卡片，9 个模块） */
const ADMIN_MODULES: AdminModule[] = [
  { icon: AppIcon.CHART, title: '地理统计', desc: '全站分布与热门地点', perm: 'admin.stats.read' },
  { icon: AppIcon.PERSON, title: '用户管理', desc: '搜索、封禁与解封用户', perm: 'admin.users.read' },
  { icon: AppIcon.GROUP, title: '群组管理', desc: '搜索与解散群组', perm: 'admin.groups.read' },
  { icon: AppIcon.FLAG, title: '举报审核', desc: '裁决用户举报', perm: 'admin.reports.read' },
  { icon: AppIcon.FILTER, title: '标签管理', desc: '屏蔽与恢复标签', perm: 'admin.tags.read' },
  { icon: AppIcon.NOTIFICATION, title: '系统公告', desc: '发布与管理公告', perm: 'admin.announcements.read' },
  { icon: AppIcon.SHIELD, title: '角色权限', desc: 'RBAC 角色与管理员', perm: 'admin.rbac.manage' },
  { icon: AppIcon.CODE, title: '审计日志', desc: '平台操作留痕查询', perm: 'admin.audit.read' },
  { icon: AppIcon.SETTINGS, title: '系统配置', desc: '应用配置与功能开关', perm: 'admin.app_config.read' }
];

/** 模块导航（对齐前端 +layout.svelte MODULES，概览永远排第一） */
const MODULE_TABS: string[] = ['概览', '地理统计', '用户', '群组', '举报', '标签', '公告', '角色权限', '审计日志', '系统配置'];

@Component
struct AdminOverviewPage {
  pathStack: NavPathStack = new NavPathStack();
  @StorageLink('safeAreaTop') safeAreaTop: number = 0;
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = DEFAULT_RESOLVED_THEME_ID;
  @StorageProp('currentBreakpoint') currentBreakpoint: BreakpointType = 'sm';

  @State permissions: string[] = [];
  @State roles: AdminRoleBasic[] = [];
  @State isSuperAdmin: boolean = false;
  @State totals: AdminGeoStatsTotals | null = null;
  @State pendingReports: number | null = null;
  @State kpiLoading: boolean = false;
  @State isLoading: boolean = true;
  @State private enterOpacity: number = 1;
  @State private enterOffset: number = 0;
  private enterPlayed: boolean = false;

  @State errorMessage: string = '';
  @State private pressedModuleKey: string = '';

  private isDestroyed: boolean = false;

  async aboutToAppear() {
    await this.loadPermissions();
  }

  aboutToDisappear(): void {
    this.isDestroyed = true;
  }

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  private async loadPermissions() {
    if (this.isDestroyed) return;
    this.isLoading = true;
    this.errorMessage = '';
    try {
      const resp = await ApiClient.getMyAdminPermissions();
      if (this.isDestroyed) return;
      if (resp.code === 0 && resp.data) {
        const data = resp.data as AdminPermissions;
        this.permissions = data.permissions ?? [];
        this.roles = data.roles ?? [];
        this.isSuperAdmin = this.permissions.some((p: string) => p === '*' || p.endsWith('.*'));
        this.loadKpi();
      } else {
        this.errorMessage = getErrorMessage(resp, '无管理员权限');
      }
    } catch (e) {
      // 10200 非管理员是业务终态（HttpUtil 以 ApiError 抛出，非传输失败），
      // 必须与网络错误区分展示，否则“无权限”会被误导为“检查网络”。
      if (e instanceof ApiError && e.code === ErrorCode.PERMISSION_DENIED) {
        this.errorMessage = e.message || '无管理员权限';
      } else {
        hilog.warn(DOMAIN, TAG, `加载权限失败: ${e}`);
        this.errorMessage = '加载权限失败，请检查网络';
      }
    } finally {
      this.isLoading = false;
      this.resetContentEnter();
      this.playContentEnter();
    }
  }

  private async loadKpi(): Promise<void> {
    const canStats = this.hasPermission('admin.stats.read');
    const canReports = this.hasPermission('admin.reports.read');
    if (!canStats && !canReports) {
      return;
    }
    this.kpiLoading = true;
    if (canStats) {
      try {
        const res = await ApiClient.getAdminGeoStats({ level: 8, limit: 1 });
        if (!this.isDestroyed && res.code === 0 && res.data) {
          this.totals = res.data.totals;
        }
      } catch (_) {
        // KPI 失败降级为「—」，不阻断模块入口（对齐 web：KPI 缺数据不报错）
      }
    }
    if (canReports) {
      try {
        const res = await ApiClient.listAdminReports({ status: 'pending', pagination: { limit: 1 } });
        if (!this.isDestroyed && res.code === 0 && res.data) {
          this.pendingReports = res.data.total;
        }
      } catch (_) {
        // 同上
      }
    }
    if (!this.isDestroyed) {
      this.kpiLoading = false;
    }
  }

  private hasPermission(perm: string): boolean {
    if (this.isSuperAdmin) {
      return true;
    }
    return this.permissions.indexOf(perm) >= 0;
  }

  private hasAnyKpiPermission(): boolean {
    return this.hasPermission('admin.stats.read') || this.hasPermission('admin.reports.read');
  }

  private getRoleLabel(): string {
    if (this.isSuperAdmin) {
      return '超级管理员';
    }
    if (this.roles.length > 0) {
      return this.roles.map((role: AdminRoleBasic) => role.name).join('、');
    }
    return '平台管理员';
  }

  private getPermissionSummary(): string {
    return this.isSuperAdmin ? '拥有全部平台权限' : `持有 ${this.permissions.length} 项权限`;
  }

  private getKpiItems(): KpiItem[] {
    const items: KpiItem[] = [];
    if (this.hasPermission('admin.stats.read')) {
      items.push({ label: '用户', value: this.formatKpi(this.totals?.user ?? null), warning: false });
      items.push({ label: '群组', value: this.formatKpi(this.totals?.group ?? null), warning: false });
      items.push({ label: '动态', value: this.formatKpi(this.totals?.moment ?? null), warning: false });
    }
    if (this.hasPermission('admin.reports.read')) {
      items.push({
        label: '待处理',
        value: this.formatKpi(this.pendingReports),
        warning: this.pendingReports !== null && this.pendingReports > 0
      });
    }
    return items;
  }

  private formatKpi(value: number | null): string {
    return value === null ? '—' : `${value}`;
  }

  private playContentEnter(): void {
    if (this.enterPlayed) return;
    this.enterPlayed = true;
    if (MotionUtil.reduced()) {
      this.enterOpacity = 1;
      this.enterOffset = 0;
      return;
    }
    this.enterOpacity = 0;
    this.enterOffset = MotionUtil.enterOffset();
    MotionUtil.contentEnter(this.getUIContext(), () => {
      this.enterOpacity = 1;
      this.enterOffset = 0;
    });
  }

  private resetContentEnter(): void {
    this.enterPlayed = false;
    this.enterOpacity = 1;
    this.enterOffset = 0;
  }

  build() {
    NavDestination() {
      PageShellContainer({ variant: 'default', currentBreakpoint: this.currentBreakpoint }) {
        Column() {
          SystemPageHeader({
            title: '管理后台',
            subtitle: '平台管理',
            showBackButton: true,
            onBack: () => { this.pathStack.pop(); },
            titleSize: DesignConstants.FONT.SIZE.LG_PLUS,
            titleWeight: DesignConstants.FONT.WEIGHT.SEMIBOLD,
            backgroundVariant: 'transparent',
            hasBelowContent: !this.isLoading && this.errorMessage.length === 0
          })

          if (!this.isLoading && this.errorMessage.length === 0 && this.permissions.length > 0) {
            this.buildModuleNav()
          }

          if (this.isLoading) {
            SystemLoadingView({ message: '加载管理权限…' }).layoutWeight(1)
          } else if (this.errorMessage.length > 0 && this.permissions.length === 0) {
            EmptyState({
              bare: true,
              icon: AppIcon.SHIELD,
              message: this.errorMessage,
              actionLabel: '重试',
              onAction: () => { this.loadPermissions(); }
            }).layoutWeight(1)
          } else {
            Scroll() {
              Column({ space: DesignConstants.PAGE_SHELL.SECTION_GAP }) {
                this.buildOverviewCard()
                this.buildModuleSection()
                this.buildWebGuideCard()
              }
              .width('100%')
              .padding({
                top: DesignConstants.PAGE_SHELL.PAGE_TOP_GAP,
                bottom: DesignConstants.SPACING.XXL
              })
            }
            .layoutWeight(1)
            .scrollBar(BarState.Off)
            .align(Alignment.Top)
            .opacity(this.enterOpacity)
            .translate({ y: this.enterOffset })
          }
        }
        .width('100%')
        .height('100%')
      }
    }
    .hideTitleBar(true)
    .onReady((context: NavDestinationContext) => {
      this.pathStack = context.pathStack;
    })
  }

  @Builder
  private buildModuleNav() {
    Scroll() {
      TabsPillGroup({
        variant: 'pills',
        freeWidth: true,
        tabs: MODULE_TABS.map((label: string): TabItem => ({ label })),
        activeIndex: 0,
        actions: {
          onChange: (index: number) => {
            if (index === 0) {
              return;
            }
            SystemToast.showWarning(`「${MODULE_TABS[index]}」请在 Web 管理后台完成`, this.getUIContext());
          }
        }
      })
    }
    .scrollable(ScrollDirection.Horizontal)
    .scrollBar(BarState.Off)
    .align(Alignment.Start)
    .width('100%')
    .padding({
      top: DesignConstants.SPACING.SM,
      bottom: DesignConstants.SPACING.MD
    })
  }

  @Builder
  private buildOverviewCard() {
    Column() {
      Row({ space: DesignConstants.SPACING.SM }) {
        LucideIcon({
          icon: AppIcon.SHIELD,
          iconSize: DesignConstants.COMPONENT.ICON_SIZE_XS,
          color: this.getPalette().accentBackground
        })

        Row({ space: DesignConstants.SPACING.XS }) {
          Text(this.getRoleLabel())
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.SM))
            .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
            .fontColor(this.getPalette().textPrimary)
          Text('·')
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.SM))
            .fontColor(this.getPalette().textSecondary)
          Text(this.getPermissionSummary())
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.SM))
            .fontColor(this.getPalette().textSecondary)
            .maxLines(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
            .layoutWeight(1)
        }
        .layoutWeight(1)
        .alignItems(VerticalAlign.Center)
      }
      .width('100%')
      .alignItems(VerticalAlign.Center)

      if (this.hasAnyKpiPermission()) {
        Divider()
          .strokeWidth(DesignConstants.BORDER_WIDTH.HAIRLINE)
          .color(this.getPalette().borderColor)
          .margin({
            top: DesignConstants.SPACING.MD,
            bottom: DesignConstants.SPACING.MD
          })

        if (this.kpiLoading) {
          Row({ space: DesignConstants.SPACING.SM }) {
            ForEach([0, 1, 2, 3], (_index: number) => {
              Column({ space: DesignConstants.SPACING.SM }) {
                Skeleton({
                  blockWidth: '44%',
                  blockHeight: DesignConstants.SPACING.XL,
                  cornerRadius: DesignConstants.RADIUS.SM
                })
                Skeleton({
                  blockWidth: '30%',
                  blockHeight: DesignConstants.FONT.SIZE.XS,
                  cornerRadius: DesignConstants.RADIUS.SM
                })
              }
              .layoutWeight(1)
              .alignItems(HorizontalAlign.Center)
            }, (index: number) => `kpi-skeleton-${index}`)
          }
          .width('100%')
          .alignItems(VerticalAlign.Center)
          .accessibilityText('正在加载平台指标')
        } else {
          Row() {
            ForEach(this.getKpiItems(), (item: KpiItem, index: number) => {
              if (index > 0) {
                Column()
                  .width(DesignConstants.BORDER_WIDTH.HAIRLINE)
                  .height(DesignConstants.SPACING.XL)
                  .backgroundColor(this.getPalette().borderColor)
                  .margin({
                    left: DesignConstants.SPACING.SM,
                    right: DesignConstants.SPACING.SM
                  })
              }
              Column({ space: DesignConstants.SPACING.SM }) {
                Text(item.value)
                  .fontSize(DesignConstants.FONT.SIZE.XXL)
                  .lineHeight(DesignConstants.FONT.SIZE.XXL)
                  .fontWeight(DesignConstants.FONT.WEIGHT.SEMIBOLD)
                  .fontColor(item.warning ? this.getPalette().warning : this.getPalette().textPrimary)
                  .maxLines(1)
                  .textOverflow({ overflow: TextOverflow.Ellipsis })
                Text(item.label)
                  .fontSize(DesignConstants.FONT.SIZE.SM)
                  .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.SM))
                  .fontColor(this.getPalette().textSecondary)
                  .maxLines(1)
              }
              .layoutWeight(1)
              .alignItems(HorizontalAlign.Center)
            }, (item: KpiItem) => item.label)
          }
          .width('100%')
          .alignItems(VerticalAlign.Center)
        }
      }
    }
    .width('100%')
    .padding(DesignConstants.SPACING.LG)
    .backgroundColor(this.getPalette().surfaceIsland)
    .borderRadius(DesignConstants.SHAPE.CARD)
    .shadow(this.getPalette().surfaceShadow)
    .alignItems(HorizontalAlign.Start)
  }

  @Builder
  private buildModuleSection() {
    Column({ space: DesignConstants.PAGE_SHELL.SECTION_INNER_GAP }) {
      Row() {
        Text('管理模块')
          .fontSize(DesignConstants.FONT.SIZE.MD)
          .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.MD))
          .fontWeight(DesignConstants.FONT.WEIGHT.SEMIBOLD)
          .fontColor(this.getPalette().textPrimary)
          .layoutWeight(1)

        Text(`共 ${ADMIN_MODULES.length} 个模块`)
          .fontSize(DesignConstants.FONT.SIZE.XS)
          .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS))
          .fontColor(this.getPalette().textSecondary)
      }
      .width('100%')
      .alignItems(VerticalAlign.Center)

      ForEach(ADMIN_MODULES, (m: AdminModule) => {
        this.buildModuleItem(m)
      }, (m: AdminModule) => m.perm)
    }
    .width('100%')
    .alignItems(HorizontalAlign.Start)
  }

  @Builder
  private buildModuleItem(m: AdminModule) {
    Row({ space: DesignConstants.SPACING.MD }) {
      Row() {
        LucideIcon({
          icon: m.icon,
          iconSize: DesignConstants.COMPONENT.ICON_SIZE_SM,
          color: this.hasPermission(m.perm) ? this.getPalette().textAccentOnSoft : this.getPalette().textSecondary
        })
      }
      .width(DesignConstants.COMPONENT.AVATAR_SIZE.SMALL)
      .height(DesignConstants.COMPONENT.AVATAR_SIZE.SMALL)
      .borderRadius(DesignConstants.RADIUS.MD)
      .backgroundColor(this.hasPermission(m.perm) ? this.getPalette().accentMuted : this.getPalette().surfaceSubtle)
      .justifyContent(FlexAlign.Center)
      .alignItems(VerticalAlign.Center)

      Column({ space: DesignConstants.SPACING.XS }) {
        Text(m.title)
          .fontSize(DesignConstants.FONT.SIZE.MD)
          .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.MD))
          .fontWeight(DesignConstants.FONT.WEIGHT.SEMIBOLD)
          .fontColor(this.getPalette().textPrimary)
          .maxLines(1)
          .textOverflow({ overflow: TextOverflow.Ellipsis })
        Text(this.hasPermission(m.perm) ? m.desc : '无访问权限')
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .lineHeight(lineHeightFor(DesignConstants.FONT.SIZE.SM, DesignConstants.FONT.LINE_RATIO.CARD))
          .fontColor(this.getPalette().textSecondary)
          .maxLines(2)
          .textOverflow({ overflow: TextOverflow.Ellipsis })
      }
      .alignItems(HorizontalAlign.Start)
      .layoutWeight(1)

      LucideIcon({
        icon: AppIcon.ARROW_RIGHT,
        iconSize: DesignConstants.COMPONENT.ICON_SIZE_XS,
        color: this.getPalette().textSecondary
      })
    }
    .width('100%')
    .padding(DesignConstants.SPACING.LG)
    .backgroundColor(this.getPalette().surfaceIsland)
    .borderRadius(DesignConstants.SHAPE.CARD)
    .shadow(this.getPalette().surfaceShadow)
    .alignItems(VerticalAlign.Center)
    .opacity(this.hasPermission(m.perm) ? 1 : DesignConstants.OPACITY.MUTED)
    .focusable(true)
    .focusOnTouch(false)
    .focusBox(FocusRing.box(this.getPalette().accentBackground))
    .accessibilityText(`${m.title}，${this.hasPermission(m.perm) ? m.desc : '无访问权限'}`)
    // 按压反馈（对齐侧栏管理后台行 tap-97 语义；减少动态时不缩放）
    .scale({
      x: MotionUtil.pressScale(this.pressedModuleKey === m.title, DesignConstants.INTERACTION.SCALE_PRESSED_CARD),
      y: MotionUtil.pressScale(this.pressedModuleKey === m.title, DesignConstants.INTERACTION.SCALE_PRESSED_CARD)
    })
    .animation(MotionUtil.spring(
      DesignConstants.INTERACTION.SPRING_RESPONSE_PRESS,
      DesignConstants.INTERACTION.SPRING_DAMPING_PRESS
    ))
    .onTouch((event: TouchEvent) => {
      if (event.type === TouchType.Down) {
        this.pressedModuleKey = m.title;
      } else if (event.type === TouchType.Up || event.type === TouchType.Cancel) {
        this.pressedModuleKey = '';
      }
    })
    .onClick(() => {
      if (!this.hasPermission(m.perm)) {
        SystemToast.showWarning('当前账号没有该模块的访问权限', this.getUIContext());
        return;
      }
      SystemToast.showWarning(`「${m.title}」请在 Web 管理后台完成`, this.getUIContext());
    })
  }

  @Builder
  private buildWebGuideCard() {
    Column({ space: DesignConstants.SPACING.XS }) {
      Row({ space: DesignConstants.SPACING.XS }) {
        LucideIcon({
          icon: AppIcon.INFO,
          iconSize: DesignConstants.COMPONENT.ICON_SIZE_2XS,
          color: this.getPalette().textSecondary
        })
        Text('完整管理请在 Web 端进行')
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.SM))
          .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
          .fontColor(this.getPalette().textPrimary)
      }
      Text('移动端提供概览与引导。表格化的批量管理、角色配置、审计查询等复杂操作，请在电脑端访问 Web 管理后台以获得最佳体验。')
        .fontSize(DesignConstants.FONT.SIZE.XS)
        .fontColor(this.getPalette().textSecondary)
        // 多行正文走比例制行高（SKILL:22-27），绝对档仅限单行紧凑场景
        .lineHeight(lineHeightFor(DesignConstants.FONT.SIZE.XS, DesignConstants.FONT.LINE_RATIO.COPY))
    }
    .width('100%')
    .alignItems(HorizontalAlign.Start)
    .padding(DesignConstants.SPACING.LG)
    .backgroundColor(this.getPalette().accentMuted)
    .borderRadius(DesignConstants.SHAPE.CARD)
  }
}

@Builder
export function AdminOverviewPageBuilder() {
  AdminOverviewPage()
}
