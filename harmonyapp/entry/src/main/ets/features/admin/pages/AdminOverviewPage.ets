/**
 * 管理后台概览页（移动端降级方案）
 *
 * 移动端仅提供「权限感知 + 模块入口引导」。复杂表格化管理（批量操作、角色配置、
 * 审计查询等）引导到 Web 端完成——移动端触屏不适合密集表格操作，管理员日常用 Web。
 *
 * 对齐前端 admin 体系的存在感（桌面侧边栏可进入），但内容聚焦概览而非复刻表格。
 */

import { ApiClient } from '../../../core/api/ApiClient';
import { getErrorMessage } from '../../../core/network/ApiErrorUtil';
import { ApiError } from '../../../core/network/ApiError';
import { ErrorCode } from '../../../core/constants/ErrorCodes';
import { curves } from '@kit.ArkUI';
import { AdminPermissions } from '../../../core/types/api';
import { PageShellContainer } from '../../../components/PageShellContainer';
import { SystemPageHeader } from '../../../components/SystemPageHeader';
import { SystemLoadingView } from '../../../components/SystemLoadingView';
import { EmptyState } from '../../../components/EmptyState';
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

const DOMAIN: number = 0xFF20;
const TAG: string = 'AdminOverviewPage';

interface AdminModule {
  icon: AppIcon;
  title: string;
  desc: string;
  perm: string;
}

const ADMIN_MODULES: AdminModule[] = [
  { icon: AppIcon.PERSON, title: '用户管理', desc: '搜索、封禁与解封用户', perm: 'admin.users.read' },
  { icon: AppIcon.GROUP, title: '群组管理', desc: '搜索、解散群组', perm: 'admin.groups.read' },
  { icon: AppIcon.WARNING, title: '举报审核', desc: '处理用户举报', perm: 'admin.reports.read' },
  { icon: AppIcon.NOTIFICATION, title: '系统公告', desc: '发布与管理公告', perm: 'admin.announcements.read' },
  { icon: AppIcon.CODE, title: '审计日志', desc: '查看操作记录', perm: 'admin.audit.read' },
  { icon: AppIcon.SHIELD, title: '角色权限', desc: 'RBAC 角色与权限管理', perm: 'admin.rbac.manage' },
  { icon: AppIcon.SETTINGS, title: '系统配置', desc: '应用配置与功能开关', perm: 'admin.app_config.read' }
];

@Component
struct AdminOverviewPage {
  pathStack: NavPathStack = new NavPathStack();
  @StorageLink('safeAreaTop') safeAreaTop: number = 0;
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = DEFAULT_RESOLVED_THEME_ID;
  @StorageProp('currentBreakpoint') currentBreakpoint: BreakpointType = 'sm';

  @State permissions: string[] = [];
  @State isSuperAdmin: boolean = false;
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
        this.isSuperAdmin = this.permissions.some((p: string) => p === '*' || p.endsWith('.*'));
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

  private hasPermission(perm: string): boolean {
    if (this.isSuperAdmin) {
      return true;
    }
    return this.permissions.indexOf(perm) >= 0;
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
      Column() {
        // Header —— 统一走 SystemPageHeader（替代手搓 Row + BackButton）
        SystemPageHeader({
          title: '管理后台',
          showBackButton: true,
          onBack: () => { this.pathStack.pop(); },
          titleSize: DesignConstants.FONT.SIZE.XXL,
          titleWeight: DesignConstants.FONT.WEIGHT.MEDIUM
        })

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
            PageShellContainer({ variant: 'default', currentBreakpoint: this.currentBreakpoint, fillHeight: false }) {
              Column({ space: DesignConstants.SPACING.LG }) {
                this.buildPermissionCard()
                this.buildModuleSection()
                this.buildWebGuideCard()
              }
              .width('100%')
              .padding({ top: DesignConstants.SPACING.LG, bottom: DesignConstants.SPACING.XXL })
            }
          }
          .layoutWeight(1)
          .scrollBar(BarState.Off)
          // 短内容顶对齐：Scroll 对小于视口的内容默认居中（SKILL:286-291）
          .align(Alignment.Top)
        .opacity(this.enterOpacity)
        .translate({ y: this.enterOffset })
        }
      }
      .width('100%')
      .height('100%')
    }
    .hideTitleBar(true)
    .onReady((context: NavDestinationContext) => {
      this.pathStack = context.pathStack;
    })
  }

  @Builder
  buildPermissionCard() {
    Row({ space: DesignConstants.SPACING.MD }) {
      Row() {
        LucideIcon({
            icon: AppIcon.SHIELD,
            iconSize: DesignConstants.COMPONENT.ICON_SIZE_SM,
            color: this.getPalette().accentText
          })
      }
      .width(DesignConstants.COMPONENT.AVATAR_SIZE.SMALL_PLUS)
      .height(DesignConstants.COMPONENT.AVATAR_SIZE.SMALL_PLUS)
      .borderRadius(DesignConstants.RADIUS.FULL)
      .backgroundColor(this.getPalette().accentBackground)
      .justifyContent(FlexAlign.Center)

      Column({ space: DesignConstants.SPACING.HALF }) {
        Text(this.isSuperAdmin ? '超级管理员' : '管理员')
          .fontSize(DesignConstants.FONT.SIZE.LG_PLUS) // 18
          .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.LG_PLUS))
          .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
          .fontColor(this.getPalette().textPrimary)
        Text(`拥有 ${this.permissions.length} 项管理权限`)
          .fontSize(DesignConstants.FONT.SIZE.SM) // 12
          .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.SM))
          .fontColor(this.getPalette().textSecondary)
      }
      .alignItems(HorizontalAlign.Start)
      .layoutWeight(1)
    }
    .width('100%')
    .padding(DesignConstants.SPACING.LG)
    // r4 软岛：去 1px 描边，surfaceIsland + 轻投影 + 圆角 20
    .backgroundColor(this.getPalette().surfaceIsland)
    .borderRadius(DesignConstants.SHAPE.CARD)
    .shadow(this.getPalette().surfaceShadow)
    .alignItems(VerticalAlign.Center)
  }

  @Builder
  buildModuleSection() {
    Column({ space: DesignConstants.SPACING.SM }) {
      Text('管理模块')
        .fontSize(DesignConstants.FONT.SIZE.XS2) // 10
        .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS2))
        .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
        .fontColor(this.getPalette().textSecondary)
        .letterSpacing(DesignConstants.FONT.LETTER_SPACING.LABEL)

      ForEach(ADMIN_MODULES, (m: AdminModule) => {
        if (this.hasPermission(m.perm)) {
          this.buildModuleItem(m)
        }
      }, (m: AdminModule) => m.perm)
    }
    .width('100%')
    .alignItems(HorizontalAlign.Start)
  }

  @Builder
  buildModuleItem(m: AdminModule) {
    Row({ space: DesignConstants.SPACING.SM }) {
      Row() {
        LucideIcon({
            icon: m.icon,
            iconSize: DesignConstants.COMPONENT.ICON_SIZE_XS,
            color: this.getPalette().accentBackground
          })
      }
      .width(DesignConstants.COMPONENT.AVATAR_SIZE.SMALL)
      .height(DesignConstants.COMPONENT.AVATAR_SIZE.SMALL)
      .borderRadius(DesignConstants.SHAPE.ICON_TILE)
      .backgroundColor(this.getPalette().accentMuted)
      .justifyContent(FlexAlign.Center)

      Column({ space: DesignConstants.SPACING.HALF }) {
        Text(m.title)
          .fontSize(DesignConstants.FONT.SIZE.SM) // 12
          .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.SM))
          .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
          .fontColor(this.getPalette().textPrimary)
        Text(m.desc)
          .fontSize(DesignConstants.FONT.SIZE.XS) // 11
          .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS))
          .fontColor(this.getPalette().textSecondary)
      }
      .alignItems(HorizontalAlign.Start)
      .layoutWeight(1)

      LucideIcon({
          icon: AppIcon.ARROW_RIGHT,
          iconSize: DesignConstants.COMPONENT.ICON_SIZE_2XS,
          color: this.getPalette().textSecondary
        })
    }
    .width('100%')
    .padding(DesignConstants.SPACING.MD)
    // r4 软岛：去 1px 描边；可聚焦 + focus 环 + 按压反馈
    .backgroundColor(this.getPalette().surfaceIsland)
    .borderRadius(DesignConstants.SHAPE.CARD)
    .shadow(this.getPalette().surfaceShadow)
    .alignItems(VerticalAlign.Center)
    .focusable(true)
    .focusOnTouch(false)
    .focusBox(FocusRing.box(this.getPalette().accentBackground))
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
      SystemToast.showWarning(`「${m.title}」请在 Web 管理后台完成`, this.getUIContext());
    })
  }

  @Builder
  buildWebGuideCard() {
    Column({ space: DesignConstants.SPACING.XS }) {
      Row({ space: DesignConstants.SPACING.XS }) {
        LucideIcon({
            icon: AppIcon.INFO,
            iconSize: DesignConstants.COMPONENT.ICON_SIZE_2XS,
            color: this.getPalette().textSecondary
          })
        Text('完整管理请在 Web 端进行')
          .fontSize(DesignConstants.FONT.SIZE.SM) // 12
          .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.SM))
          .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
          .fontColor(this.getPalette().textPrimary)
      }
      Text('移动端提供概览与引导。表格化的批量管理、角色配置、审计查询等复杂操作，请在电脑端访问 Web 管理后台以获得最佳体验。')
        .fontSize(DesignConstants.FONT.SIZE.XS) // 11
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