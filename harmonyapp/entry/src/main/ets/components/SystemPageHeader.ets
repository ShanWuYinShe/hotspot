/**
 * SystemPageHeader
 * - 统一页面标题栏样式（Apple-like / HarmonyOS Next）
 * - 覆盖全仓所有页面头部：标题 + 副标题 + 可选返回/左操作/尾随内容
 * - 结构特殊的页面（内联搜索、浮动卡片+头像）用 SearchPageHeader / ChatPageHeader
 *
 * 扩展 API（向后兼容 7 个旧消费者）：
 *   subtitle          标题下副文本（未读数/标语/计数）
 *   trailing          @BuilderParam 任意尾随内容（位置 chip/多按钮簇/头像），优先级高于 action/rightAction
 *   titleAlign        标题对齐（默认 Start，全站统一左对齐；Center 变体保留给 hero 封面页，2026-10-13 起标准二级页不再使用）
 *   titleSize/Weight  可覆盖（如 XL/LG/XXL 等 FONT.SIZE 档位）
 *   backgroundVariant 'bar'(默认全宽底栏) | 'transparent'(透明) | 'card'(浮动圆角卡)
 */

import { DesignConstants } from '../constants/DesignConstants';
import { AppThemeManager } from '../core/theme/AppThemeManager';
import { AppThemeShadow, DEFAULT_RESOLVED_THEME_ID, KEY_RESOLVED_THEME_ID } from '../core/theme/AppThemeTypes';
import { AppIcon } from './SystemIcon';
import { LucideIcon } from './LucideIcon';
import { SurfaceIconButton } from './SurfaceIconButton';
import { SystemButton } from './SystemButton';
import { SafeAreaUtils } from '../core/utils/SafeAreaUtils';

export interface PageHeaderAction {
  label: string;
  onClick: () => void | Promise<void>;
  // 轻量扩展：有需要再加 icon / disabled / loading
  disabled?: boolean;
}

export interface IconAction {
  icon: AppIcon;
  onClick: () => void;
  kind?: 'primary' | 'neutral';
  disabled?: boolean;
  size?: number;
  iconSize?: number;
  /** 无障碍语义描述，透传给 SurfaceIconButton（图标钮读屏必需） */
  accessibilityLabel?: string;
}

/** 背景变体：bar=全宽底栏(默认)，transparent=透明(继承页面背景)，card=浮动圆角卡 */
export type HeaderBackgroundVariant = 'bar' | 'transparent' | 'card';

@Component
export struct SystemPageHeader {
  @Prop title: string = '';
  @Prop subtitle: string = '';
  /** 副文本颜色（缺省 textSecondary；如消息页「全部已读」用 success 色，对齐前端） */
  @Prop subtitleColor: string = '';
  action: PageHeaderAction | null = null;
  @Prop showBackButton: boolean = false;
  onBack: (() => void) | null = null;
  leftAction: IconAction | null = null;
  rightAction: IconAction | null = null;
  /** 页面在页头下方自带内容（如搜索框）：页头底部留白转由该内容自行承担，
   *  对齐前端 PageHeader「below 存在时 pt-4(16) pb-0 + below 自带 pb-3.5(14)」的节奏。 */
  @Prop hasBelowContent: boolean = false;
  /** 任意尾随内容（@BuilderParam，可选）。提供时优先于 action/rightAction。 */
  @BuilderParam trailing?: () => void;
  /** 副标题前置图标（可选；对齐前端 PageHeader subtitleIcon，如附近页 MapPin 位置行）。
   *  注：不用 @BuilderParam——ArkTS trailing lambda 要求唯一 BuilderParam，多参数会破坏全部现有调用。 */
  @Prop subtitleIcon: AppIcon | null = null;
  /** 标题对齐，默认 Start；详情页居中标题用 Center。 */
  @Prop titleAlign: TextAlign = TextAlign.Start;
  /** 标题字号，默认 FONT.SIZE.XL（20）。顶级 Tab 页对齐前端传 XXL（24）。 */
  @Prop titleSize: number = DesignConstants.FONT.SIZE.XL;
  /** 标题字重，默认 500（对齐前端 PageHeader font-medium）。 */
  @Prop titleWeight: number = DesignConstants.FONT.WEIGHT.SEMIBOLD;
  /** 背景变体：bar(默认全宽底栏) / transparent / card(浮动圆角卡)。 */
  @Prop backgroundVariant: HeaderBackgroundVariant = 'bar';
  /** 滚动态（页面接滚动态传入）：静止无边线与页面融合，滚动后恢复边线并出现投影。 */
  @Prop elevated: boolean = false;
  @StorageLink('safeAreaTop') safeAreaTop: number = 0;
  @Prop surfaceColor: ResourceColor | null = null;
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = DEFAULT_RESOLVED_THEME_ID;

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  private getHeaderTopPadding(): number {
    // extra 0：市面惯例是头部内容紧贴状态栏下缘（行内 MD=12 提供呼吸感）。
    // 此前 resolveTopPadding 默认再 +SM(8)，实测返回钮距状态栏 35vp，观感「离
    // 边缘太远」（2026-10-10 模拟器 dumpLayout 实测 206px≈62vp）。
    return SafeAreaUtils.resolveTopPadding(this.safeAreaTop, 0);
  }

  private getBackgroundColor(): ResourceColor {
    if (this.surfaceColor) {
      return this.surfaceColor;
    }
    if (this.backgroundVariant === 'transparent') {
      return DesignConstants.COLOR.TRANSPARENT;
    }
    // 对齐前端 PageHeader：背景 = --cst-surface-page（页面画布同底）
    return this.getPalette().pageBackground;
  }

  private getHeaderShadow(): AppThemeShadow {
    // 滚动态（elevated）出现表面投影：Gemini 审查「静止时分割线显生硬，应改为
    // 滑动时以 elevation 区分层级」——prop 由页面接滚动态传入
    if (this.elevated) {
      return this.getPalette().surfaceShadow;
    }
    // card 变体用表面阴影；其余归零（bar 靠底边线分隔，transparent 无边无影）
    if (this.backgroundVariant === 'card') {
      return this.getPalette().surfaceShadow;
    }
    return new AppThemeShadow(0, DesignConstants.COLOR.TRANSPARENT, 0, 0);
  }

  private getHeaderBorderColor(): ResourceColor {
    if (this.backgroundVariant === 'card') {
      return DesignConstants.COLOR.TRANSPARENT;
    }
    // 对齐前端 PageHeader：任何页面头部在任何状态都画 border-bottom hairline
    // （2026-10-17 逐像素实测 9 个移动端页面：头部最后一行像素均比画布亮，
    //  如 /messages y=117 是 rgb(42,60,89)、画布 rgb(21,33,52)）。
    // 此前 transparent 变体完全不画线、bar 变体仅在滚动态画线，与前端不一致。
    return this.getPalette().borderColor;
  }

  private getRadius(): number {
    // card 变体底栏圆角（对齐 SHAPE.TAB_BAR）；bar/transparent 无圆角
    if (this.backgroundVariant === 'card') {
      return DesignConstants.SHAPE.TAB_BAR;
    }
    return 0;
  }

  private canShowBackButton(): boolean {
    return this.showBackButton && this.onBack !== null;
  }

  private hasLeftAction(): boolean {
    return this.canShowBackButton() || this.leftAction !== null;
  }

  private hasRightAction(): boolean {
    return this.rightAction !== null || this.action !== null;
  }

  @Builder
  private buildIconActionButton(action: IconAction) {
    SurfaceIconButton({
      icon: action.icon,
      kind: action.kind ?? 'neutral',
      // 右侧图标钮与返回钮同尺寸（BACK_BUTTON_SIZE 40），页头左右视觉重量对称
      buttonSize: action.size ?? DesignConstants.LAYOUT.BACK_BUTTON_SIZE,
      glyphSize: action.iconSize ?? 20,
      disabled: action.disabled ?? false,
      accessibilityLabel: action.accessibilityLabel ?? '',
      onPress: action.onClick
    })
  }

  @Builder
  private buildActionTextButton(a: PageHeaderAction) {
    // 原语强制：文本操作按钮统一走 SystemButton。
    // primary+pill 复刻原手写样式（accent 实底胶囊 42vp + SM 字号 + FULL 圆角），保持视觉不变；
    // disabled 由 SystemButton 内聚 enabled(false) + OPACITY.DISABLED。ghost 为透明底文本钮，
    // 会丢掉 accent 实底视觉，故不采用。
    SystemButton({
      label: a.label,
      variant: 'primary',
      pill: true,
      disabled: a.disabled ?? false,
      onPress: () => {
        a.onClick();
      }
    })
  }

  build() {
    Column() {
      Row({ space: DesignConstants.SPACING.SM }) {
        // 左侧：返回按钮 / 左操作
        if (this.hasLeftAction()) {
          Row({ space: DesignConstants.SPACING.SM }) {
            if (this.canShowBackButton()) {
              SurfaceIconButton({
                icon: AppIcon.BACK,
                // 返回钮统一凸起中性样式（surface 浅底 + hairline 边框），与
                // SearchPageHeader / 前端 ActionIconButton surface 同视觉；
                // 40vp 大于普通图标钮（36）：导航关键动作易点不误触
                kind: 'surface',
                buttonSize: DesignConstants.LAYOUT.BACK_BUTTON_SIZE,
                // 箭头字号 18（对齐前端 42 钮 * 0.42 ≈ 18 的图标比例）
                glyphSize: DesignConstants.FONT.SIZE.LG_PLUS,
                // 读屏标签：纯图标钮必须播报用途（缺省空串读屏只读「按钮」）
                accessibilityLabel: '返回',
                onPress: () => {
                  if (this.onBack) {
                    this.onBack();
                  }
                }
              })
            }

            if (this.leftAction) {
              this.buildIconActionButton(this.leftAction);
            }
          }
          .alignItems(VerticalAlign.Center)
        }

        // 标题区：标题 + 可选副标题
        // layoutWeight(1) 占据中间空间；居中标题时容器整体居中。
        Column() {
          Text(this.title)
            .fontSize(this.titleSize)
            .fontWeight(this.titleWeight)
            // leading-none：前端标题行高恒等于字号，行盒不被字体默认行高撑高
            .lineHeight(this.titleSize)
            .fontColor(this.getPalette().textPrimary)
            .maxLines(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
            .textAlign(this.titleAlign)

          if (this.subtitle.length > 0) {
            Row({ space: DesignConstants.SPACING.XS }) {
              if (this.subtitleIcon !== null) {
                LucideIcon({
                  icon: this.subtitleIcon,
                  iconSize: DesignConstants.COMPONENT.ICON_SIZE_2XS,
                  color: this.getPalette().accentBackground
                })
              }
              Text(this.subtitle)
                .fontSize(DesignConstants.FONT.SIZE.XS)
                .fontColor(this.subtitleColor.length > 0 ? this.subtitleColor : this.getPalette().textSecondary)
                .maxLines(1)
                .textOverflow({ overflow: TextOverflow.Ellipsis })
            }
            .alignItems(VerticalAlign.Center)
            .margin({ top: DesignConstants.SPACING.XS })
          }
        }
        .layoutWeight(1)
        .alignItems(this.titleAlign === TextAlign.Center ? HorizontalAlign.Center : HorizontalAlign.Start)

        // 右侧：trailing(优先) → rightAction → action
        // trailing 包一层 flexShrink(0)：窄屏下操作簇不被压缩折行，由标题列
        // layoutWeight(1) + Text 省略先让位（对齐前端 PageHeader trailing shrink-0 契约）
        if (this.trailing) {
          Row() {
            this.trailing!()
          }
          .flexShrink(0)
          .alignItems(VerticalAlign.Center)
        } else if (this.hasRightAction()) {
          Row({ space: DesignConstants.SPACING.SM }) {
            if (this.rightAction) {
              this.buildIconActionButton(this.rightAction);
            }
            if (this.action) {
              this.buildActionTextButton(this.action);
            }
          }
          .alignItems(VerticalAlign.Center)
        }
      }
      .width('100%')
      .alignItems(VerticalAlign.Center)
      // 只保留纵向节奏（对齐前端 PageHeader）：无 below 时 pt-4.5/pb-4.5(18/18)，
      // 有 below（页面自带搜索框等）时 pt-4(16) 且底部留白交给该内容承担。
      // **横向留白一律由 PageShellContainer 的 gutter 提供**：页头此前自带
      // LG2(20) 横向 padding，与容器 gutter 叠加成 20+16=36，而正文只有 16，
      // 导致「页头标题比正文内容多缩进一截」——同一页面内左边缘都不齐。
      .padding({
        top: this.hasBelowContent ? DesignConstants.SPACING.LG : DesignConstants.SPACING.LG_PLUS,
        bottom: this.hasBelowContent ? 0 : DesignConstants.SPACING.LG_PLUS
      })
    }
    .width('100%')
    .padding({
      top: this.getHeaderTopPadding()
    })
    .backgroundColor(this.getBackgroundColor())
    .border({ width: { bottom: DesignConstants.BORDER_WIDTH.HAIRLINE }, color: this.getHeaderBorderColor() })
    .borderRadius(this.getRadius())
    .shadow(this.getHeaderShadow())
    .expandSafeArea([SafeAreaType.SYSTEM], [SafeAreaEdge.TOP])
  }
}
