/**
 * LoginPromptView - 可复用的未登录内联引导（对齐 web components/auth/NeedsAuthNotice.svelte）
 *
 * 形态（R13 访客引导面板）：图标圆 64（accent-soft 底 + accent 图标）
 *   → 标题 14/600 → 说明 12 次色
 *   → 可选 3 条能力行（圆形小图标 32 + 11/500 文本）
 *   → 主 CTA「登录 / 注册」+ 次 CTA「先随便逛逛」（ghost，去首页公共内容）。
 *
 * 约定：
 * - 本组件只负责「整页/整块只读内容」的未登录占位；写操作拦截仍走 LoginGate 弹层。
 * - 「先随便逛逛」全站唯一语义：跳过登录、去可浏览的公共内容继续逛，不创建任何账号
 *   （登录页里的「访客模式」是另一个动作：创建临时账号）。
 * - 不做 spotlight 装饰层：web NeedAuthNotice 的 spotlight 层实际只渲染 glow 且 display:none，
 *   观感是干净的页内引导，不是带双环的空态插画。
 * - 旧调用兼容：title/subtitle/icon/redirectUrl/loginAction/surfaceColor/spotlight 保留；
 *   spotlight 已无视觉（调用方无需改动，后续可移除）。
 */

import { DesignConstants, lineHeightFor } from '../constants/DesignConstants';
import { PageNames, StorageKeys } from '../constants/AppConstants';
import { AppRouter, LoginRouteParam } from '../core/router/AppRouter';
import { AppIcon } from './SystemIcon';
import { SystemButton } from './SystemButton';
import { LucideIcon } from './LucideIcon';
import { AppThemeManager } from '../core/theme/AppThemeManager';
import { DEFAULT_RESOLVED_THEME_ID, KEY_RESOLVED_THEME_ID } from '../core/theme/AppThemeTypes';

/** 能力行（icon + 文案，最多 3 条；对齐 web NeedsAuthNotice highlights） */
export interface LoginPromptHighlight {
  icon: AppIcon;
  text: string;
}

@Component
export struct LoginPromptView {
  /** 主标题，如 "登录后查看消息" */
  @Prop title: string = '此功能需要登录';
  /** 副标题说明 */
  @Prop subtitle: string = '登录后即可查看完整内容';
  /** 底部提示文字（可选） */
  @Prop hint: string = '';
  /** 图标（默认人形图标，对齐 web UserRound） */
  @Prop icon: AppIcon = AppIcon.PERSON;
  /** 登录后跳转的页面路由 */
  @Prop redirectUrl: string = '';
  /** 登录原因描述（传给 LoginPage） */
  @Prop loginAction: string = '';
  /**
   * 能力行（登录后能得到什么）：web 消息/通知访客态各 3 条。
   * 不传则不渲染（通用兜底页面保持精简）。
   */
  @Prop highlights: LoginPromptHighlight[] = [];
  /** 主 CTA 文案（对齐 web primaryLabel，默认「登录 / 注册」） */
  @Prop primaryLabel: string = '登录 / 注册';
  /** 次 CTA 文案（ghost；默认「先随便逛逛」，语义 = 跳过登录去浏览公共内容） */
  @Prop secondaryLabel: string = '先随便逛逛';
  /** 次 CTA 行为：缺省回首页（可浏览的公共内容） */
  onSecondary?: () => void = undefined;
  /** 作为独立全屏页时由调用方注入背景；嵌入场景不传则透明（让父容器背景透出）。 */
  @Prop surfaceColor: ResourceColor | null = null;
  /** 已废弃：web NeedAuthNotice 的 spotlight 装饰层实际不可见，本组件不再绘制装饰层。 */
  @Prop spotlight: boolean = false;

  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = DEFAULT_RESOLVED_THEME_ID;

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  /** 能力行图标圆底：web size-8(32) 圆形 surface-island + shadow-xs */
  @Builder
  private buildHighlightRow(item: LoginPromptHighlight) {
    Row({ space: DesignConstants.SPACING.MD2 }) {
      Row() {
        LucideIcon({
          icon: item.icon,
          iconSize: DesignConstants.COMPONENT.ICON_SIZE_XS,
          color: this.getPalette().accentBackground
        })
      }
      .width(DesignConstants.COMPONENT.AVATAR_SIZE.XS)
      .height(DesignConstants.COMPONENT.AVATAR_SIZE.XS)
      .borderRadius(DesignConstants.RADIUS.FULL)
      .backgroundColor(this.getPalette().surfaceIsland)
      .shadow(this.getPalette().surfaceShadow)

      Text(item.text)
        .fontSize(DesignConstants.FONT.SIZE.XS)
        .lineHeight(lineHeightFor(DesignConstants.FONT.SIZE.XS, 1.625))
        .fontColor(this.getPalette().textSecondary)
        .layoutWeight(1)
        .maxLines(2)
        .textOverflow({ overflow: TextOverflow.Ellipsis })
    }
    .width('100%')
    .alignItems(VerticalAlign.Center)
  }

  private goBrowse(): void {
    if (this.onSecondary) {
      this.onSecondary();
      return;
    }
    // 默认回首页 Tab（公共可浏览内容）：
    // 先清空导航栈回到 Tabs 根视图（独立推入的页面如「我的分享」需要先退出），
    // 再发出切 tab 信号由 PhoneLayout 消费；无 Tab 壳的桌面壳里切 tab 信号无副作用。
    AppRouter.clear();
    AppStorage.setOrCreate<number>(StorageKeys.SWITCH_TAB_REQUEST, 0);
  }

  private navigateToLogin(): void {
    const loginParams = new LoginRouteParam();
    loginParams.reason = 'action_required';
    loginParams.action = this.loginAction || this.title;
    loginParams.redirectUrl = this.redirectUrl;
    AppRouter.push(PageNames.LOGIN, loginParams);
  }

  build() {
    Column() {
      // 图标圆 64（web EmptyState 非 bare：accent-soft 底 + accent 图标）
      Row() {
        LucideIcon({
          icon: this.icon,
          iconSize: DesignConstants.COMPONENT.ICON_SIZE_LG,
          color: this.getPalette().textAccentOnSoft
        })
      }
      .width(DesignConstants.COMPONENT.EMPTY_ICON_SIZE)
      .height(DesignConstants.COMPONENT.EMPTY_ICON_SIZE)
      .borderRadius(DesignConstants.RADIUS.FULL)
      .backgroundColor(this.getPalette().accentMuted)
      .justifyContent(FlexAlign.Center)
      .alignItems(VerticalAlign.Center)

      Text(this.title)
        .fontSize(DesignConstants.FONT.SIZE.MD)
        .lineHeight(DesignConstants.FONT.LINE_HEIGHT.MD)
        .fontWeight(DesignConstants.FONT.WEIGHT.SEMIBOLD)
        .fontColor(this.getPalette().textPrimary)
        .textAlign(TextAlign.Center)
        .margin({ top: DesignConstants.SPACING.MD })

      Text(this.subtitle)
        .fontSize(DesignConstants.FONT.SIZE.SM)
        .lineHeight(lineHeightFor(DesignConstants.FONT.SIZE.SM, 1.625))
        .fontColor(this.getPalette().textSecondary)
        .textAlign(TextAlign.Center)
        .margin({ top: DesignConstants.SPACING.MD })

      // 能力行（web extra slot：mt-1 + gap-2 + 左对齐 + max-w-90）
      if (this.highlights.length > 0) {
        Column({ space: DesignConstants.SPACING.SM }) {
          ForEach(this.highlights, (item: LoginPromptHighlight) => {
            this.buildHighlightRow(item)
          }, (item: LoginPromptHighlight, index: number) => `${item.text}-${index}`)
        }
        .width('100%')
        .padding({ left: DesignConstants.SPACING.XS, right: DesignConstants.SPACING.XS })
        .margin({ top: DesignConstants.SPACING.XS })
      }

      // CTA 行：主「登录 / 注册」+ ghost「先随便逛逛」（web mt-2 + gap-2.5 居中）
      Row({ space: DesignConstants.SPACING.MD2 }) {
        SystemButton({
          label: this.primaryLabel,
          variant: 'primary',
          buttonSize: 'default',
          onPress: () => {
            this.navigateToLogin();
          }
        })
        SystemButton({
          label: this.secondaryLabel,
          variant: 'ghost',
          buttonSize: 'default',
          onPress: () => {
            this.goBrowse();
          }
        })
      }
      .alignItems(VerticalAlign.Center)
      .justifyContent(FlexAlign.Center)
      .margin({ top: DesignConstants.SPACING.SM })

      if (this.hint.length > 0) {
        Text(this.hint)
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .lineHeight(DesignConstants.FONT.LINE_HEIGHT.SM)
          .fontColor(this.getPalette().textSecondary)
          .textAlign(TextAlign.Center)
          .margin({ top: DesignConstants.SPACING.XS })
      }
    }
    .width('100%')
    .alignItems(HorizontalAlign.Center)
    .padding({
      left: DesignConstants.PAGE_SHELL.GUTTER_MOBILE,
      right: DesignConstants.PAGE_SHELL.GUTTER_MOBILE,
      // 对齐 web EmptyState py-20(80)：内容在页头下方留出页级呼吸再开始（32×2+16 = 80）
      top: DesignConstants.SPACING.XXL * 2 + DesignConstants.SPACING.LG,
      bottom: DesignConstants.SPACING.XXL * 2 + DesignConstants.SPACING.LG
    })
    // 嵌入场景透明（父容器 pageBackground/AmbientBackdrop 透出）；独立全屏页由调用方注入 surfaceColor
    .backgroundColor(this.surfaceColor ?? DesignConstants.COLOR.TRANSPARENT)
  }
}
