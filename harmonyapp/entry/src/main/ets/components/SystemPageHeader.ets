/**
 * SystemPageHeader
 * - 统一页面标题栏样式（Apple-like / HarmonyOS Next）
 * - 覆盖全仓所有页面头部：标题 + 副标题 + 可选返回/左操作/尾随内容
 * - 结构特殊的页面（内联搜索、浮动卡片+头像）用 SearchPageHeader / ChatPageHeader
 *
 * 扩展 API（向后兼容 7 个旧消费者）：
 *   subtitle          标题下副文本（未读数/标语/计数）
 *   trailing          @BuilderParam 任意尾随内容（位置 chip/多按钮簇/头像），优先级高于 action/rightAction
 *   titleAlign        标题对齐（默认 Start；GroupDetailPage 需 Center）
 *   titleSize/Weight  可覆盖（XL_PLUS/LG 等）
 *   backgroundVariant 'bar'(默认全宽底栏) | 'transparent'(透明) | 'card'(浮动圆角卡)
 */

import { DesignConstants } from '../constants/DesignConstants';
import { AppThemeManager } from '../core/theme/AppThemeManager';
import { AppThemeShadow, KEY_RESOLVED_THEME_ID } from '../core/theme/AppThemeTypes';
import { AppIcon } from './SystemIcon';
import { SurfaceIconButton } from './SurfaceIconButton';
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
}

/** 背景变体：bar=全宽底栏(默认)，transparent=透明(继承页面背景)，card=浮动圆角卡 */
export type HeaderBackgroundVariant = 'bar' | 'transparent' | 'card';

@Component
export struct SystemPageHeader {
  @Prop title: string = '';
  @Prop subtitle: string = '';
  action: PageHeaderAction | null = null;
  @Prop showBackButton: boolean = false;
  onBack: (() => void) | null = null;
  leftAction: IconAction | null = null;
  rightAction: IconAction | null = null;
  /** 任意尾随内容（@BuilderParam，可选）。提供时优先于 action/rightAction。 */
  @BuilderParam trailing?: () => void;
  /** 标题对齐，默认 Start；详情页居中标题用 Center。 */
  @Prop titleAlign: TextAlign = TextAlign.Start;
  /** 标题字号，默认 FONT.SIZE.XL（20）。 */
  @Prop titleSize: number = DesignConstants.FONT.SIZE.XL;
  /** 标题字重，默认 600。 */
  @Prop titleWeight: number = DesignConstants.FONT.WEIGHT.SEMIBOLD;
  /** 背景变体：bar(默认全宽底栏) / transparent / card(浮动圆角卡)。 */
  @Prop backgroundVariant: HeaderBackgroundVariant = 'bar';
  @StorageLink('safeAreaTop') safeAreaTop: number = 0;
  @Prop surfaceColor: ResourceColor | null = null;
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'default_clean_light';

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  private getHeaderTopPadding(): number {
    return SafeAreaUtils.resolveTopPadding(this.safeAreaTop);
  }

  private getBackgroundColor(): ResourceColor {
    if (this.surfaceColor) {
      return this.surfaceColor;
    }
    if (this.backgroundVariant === 'transparent') {
      return DesignConstants.COLOR.TRANSPARENT;
    }
    return this.getPalette().sheetBackground;
  }

  private getHeaderShadow(): AppThemeShadow {
    // card 变体用表面阴影；其余归零（bar 靠底边线分隔，transparent 无边无影）
    if (this.backgroundVariant === 'card') {
      return this.getPalette().surfaceShadow;
    }
    return new AppThemeShadow(0, DesignConstants.COLOR.TRANSPARENT, 0, 0);
  }

  private getHeaderBorderColor(): ResourceColor {
    if (this.backgroundVariant === 'transparent' || this.backgroundVariant === 'card') {
      return DesignConstants.COLOR.TRANSPARENT;
    }
    return this.getPalette().floatingBorderColor;
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

  private hasTrailing(): boolean {
    // trailing BuilderParam 提供时优先；否则回落到 action/rightAction
    return this.trailing !== undefined && this.trailing !== null;
  }

  private hasRightAction(): boolean {
    return this.rightAction !== null || this.action !== null;
  }

  @Builder
  private buildIconActionButton(action: IconAction) {
    SurfaceIconButton({
      icon: action.icon,
      kind: action.kind ?? 'neutral',
      buttonSize: action.size ?? 40,
      glyphSize: action.iconSize ?? 20,
      disabled: action.disabled ?? false,
      onPress: action.onClick
    })
  }

  @Builder
  private buildActionTextButton(a: PageHeaderAction) {
    Button(a.label)
      .type(ButtonType.Normal)
      .backgroundColor(this.getPalette().accentBackground)
      .borderRadius(DesignConstants.RADIUS.FULL)
      .height(DesignConstants.COMPONENT.BUTTON_HEIGHT)
      .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG })
      .fontColor(this.getPalette().accentText)
      .fontSize(DesignConstants.FONT.SIZE.SM)
      .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
      .enabled(!a.disabled)
      .onClick(() => {
        a?.onClick();
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
                kind: 'neutral',
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
            .fontColor(this.getPalette().textPrimary)
            .maxLines(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
            .textAlign(this.titleAlign)

          if (this.subtitle.length > 0) {
            Text(this.subtitle)
              .fontSize(DesignConstants.FONT.SIZE.SM)
              .fontColor(this.getPalette().textSecondary)
              .margin({ top: DesignConstants.SPACING.XS })
              .maxLines(1)
              .textOverflow({ overflow: TextOverflow.Ellipsis })
          }
        }
        .layoutWeight(1)
        .alignItems(this.titleAlign === TextAlign.Center ? HorizontalAlign.Center : HorizontalAlign.Start)

        // 右侧：trailing(优先) → rightAction → action
        if (this.trailing) {
          this.trailing!()
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
      // 修复地基 token 泄漏：原硬编码 left:16/right:16/top:12/bottom:12
      .padding({
        left: DesignConstants.SPACING.LG,
        right: DesignConstants.SPACING.LG,
        top: DesignConstants.SPACING.MD,
        bottom: DesignConstants.SPACING.MD
      })
    }
    .width('100%')
    .padding({
      top: this.getHeaderTopPadding()
    })
    .backgroundColor(this.getBackgroundColor())
    .border({ width: { bottom: 1 }, color: this.getHeaderBorderColor() })
    .borderRadius(this.getRadius())
    .shadow(this.getHeaderShadow())
    .expandSafeArea([SafeAreaType.SYSTEM], [SafeAreaEdge.TOP])
  }
}
