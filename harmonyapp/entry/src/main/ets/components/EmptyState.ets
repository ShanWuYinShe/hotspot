/**
 * 通用空状态组件
 * 用于列表无数据、加载错误等场景的统一展示
 */

import { DesignConstants } from '../constants/DesignConstants';
import { AppIcon } from './SystemIcon';
import { AppThemeManager } from '../core/theme/AppThemeManager';
import { SystemButton } from './SystemButton';
import { DEFAULT_RESOLVED_THEME_ID, KEY_RESOLVED_THEME_ID } from '../core/theme/AppThemeTypes';
import { LucideIcon } from './LucideIcon';

@Reusable
@Component
export struct EmptyState {
  @Prop icon: AppIcon = AppIcon.EMPTY;
  /** 主标题（大字，对齐前端 EmptyState title） */
  @Prop title: string = '';
  /** 说明文字（小字，向后兼容：等价于前端 description） */
  @Prop message: string = '';
  @Prop actionLabel: string = '';
  /** 整页空态：撑开 min-height 垂直居中（对齐前端 fill 模式，用于下方无并列内容的空态） */
  @Prop fillMode: boolean = false;
  /** 极简空态：不画卡片背景/边框/阴影（对齐前端 bare 形态，用于 tab 内容区/列表内空态） */
  @Prop bare: boolean = false;
  onAction?: () => void;

  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = DEFAULT_RESOLVED_THEME_ID;

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  build() {
    Column() {
      // 卡内呼吸感：非 bare 空态图标-文案-按钮间距提档到 XL（r45：层级紧凑、
      // 视觉重心偏低）；bare 列表内空态保持 LG 紧凑
      Column({ space: this.bare ? DesignConstants.SPACING.LG : DesignConstants.SPACING.XL }) {
        // bare：无圆形容器，图标直接渲染 + muted 文案（对齐前端 EmptyState bare 契约）
        if (this.bare) {
          LucideIcon({
            icon: this.icon,
            iconSize: DesignConstants.COMPONENT.ICON_SIZE_MD,
            color: this.getPalette().textSecondary
          })
          .opacity(DesignConstants.OPACITY.MUTED)
        } else {
          Stack() {
            // 对齐前端 EmptyState 非 bare 契约：圆形容器 = 透明底 + 发丝描边
            // （frontend rounded-full border border-border-default shadow-raised-sm，
            // 无填充色；原 accentMuted 实底为单端发散，已收敛）
            Column()
              .width(DesignConstants.COMPONENT.EMPTY_ICON_SIZE)
              .height(DesignConstants.COMPONENT.EMPTY_ICON_SIZE)
              .borderRadius(DesignConstants.RADIUS.FULL)
              .backgroundColor(DesignConstants.COLOR.TRANSPARENT)
              .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getPalette().borderColor })
              .shadow(this.getPalette().surfaceShadow)

            LucideIcon({
              icon: this.icon,
              iconSize: DesignConstants.COMPONENT.ICON_SIZE_MD,
              color: this.getPalette().accentBackground
            })
            .opacity(DesignConstants.OPACITY.EMPHASIZED)
          }
          .width(DesignConstants.COMPONENT.EMPTY_ICON_SIZE)
          .height(DesignConstants.COMPONENT.EMPTY_ICON_SIZE)
        }

        // 主标题 —— 对齐前端 EmptyState title（text-sm = FONT.SIZE.SM，与描述同尺寸靠
        // 字重 + 颜色区分层级：title font-medium + textPrimary，description regular + textSecondary）。
        // 注意：前端 --text-sm 在 theme-registry.css 覆盖为 12px（非 Tailwind 默认 14），
        // 故此处取 SM 而非 MD。
        // bare 变体 title 用 textSecondary（对齐前端 bare 契约 muted 文案）
        if ((this.title ?? '').length > 0) {
          Text(this.title)
            .fontSize(DesignConstants.FONT.SIZE.SM) // 12 对齐前端 text-sm
            .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
            .fontColor(this.bare ? this.getPalette().textSecondary : this.getPalette().textPrimary)
            .textAlign(TextAlign.Center)
        }

        // 说明文字 —— 对齐前端 EmptyState description（text-sm = FONT.SIZE.SM，同尺寸二级）
        if ((this.message ?? '').length > 0) {
          Text(this.message)
            .fontSize(DesignConstants.FONT.SIZE.SM) // 12 对齐前端 text-sm
            .fontColor(this.getPalette().textSecondary)
            .textAlign(TextAlign.Center)
            .lineHeight(DesignConstants.FONT.LINE_HEIGHT.SM)
        }

        if ((this.actionLabel ?? '').length > 0 && this.onAction) {
          SystemButton({
            label: this.actionLabel ?? '',
            variant: 'primary',
            buttonSize: 'default',
            onPress: () => {
              this.onAction?.();
            }
          })
        }
      }
      .width('100%')
      .padding(this.bare ? 0 : DesignConstants.SPACING.XXL)
      .borderRadius(this.bare ? 0 : DesignConstants.SHAPE.HERO_CARD)
      .backgroundColor(this.bare ? DesignConstants.COLOR.TRANSPARENT : this.getPalette().surfaceBackground)
      .border(this.bare
        ? { width: 0 }
        : { width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getPalette().floatingBorderColor })
      .shadow(this.bare
        ? { radius: 0, color: DesignConstants.COLOR.TRANSPARENT, offsetX: 0, offsetY: 0 }
        : this.getPalette().surfaceShadow)
    }
    .width('100%')
    .height('100%')
    .constraintSize({ minHeight: this.fillMode ? DesignConstants.COMPONENT.EMPTY_FILL_MIN_HEIGHT : 0 })
    .justifyContent(FlexAlign.Center)
    .padding(DesignConstants.SPACING.XL)
  }
}


/**
 * 加载更多指示器
 * 用于列表底部的加载更多状态展示
 */
@Reusable
@Component
export struct LoadingMoreIndicator {
  @Prop message: string = '加载更多…';
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = DEFAULT_RESOLVED_THEME_ID;

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  build() {
    Row({ space: DesignConstants.SPACING.SM }) {
      LoadingProgress()
        .width(DesignConstants.COMPONENT.LOADING_ICON_SIZE)
        .height(DesignConstants.COMPONENT.LOADING_ICON_SIZE)
        .color(this.getPalette().accentBackground)
        .accessibilityText('正在加载')
      Text(this.message)
        .fontSize(DesignConstants.FONT.SIZE.SM)
        .fontColor(this.getPalette().textSecondary)
    }
    .width('100%')
    .justifyContent(FlexAlign.Center)
    // 对齐前端 InlineLoading：列表内的加载态是「居中 spinner + 次要色文案」，
    // 不带卡片底色/描边/投影——原实现是一条带浮起投影的胶囊条，
    // 在列表底部与周围内容格格不入（附近的人/群组页尤其明显）
    .padding({ top: DesignConstants.SPACING.MD, bottom: DesignConstants.SPACING.MD })
    .accessibilityText(this.message)
  }
}

/**
 * 下拉刷新指示器
 * 用于 Refresh 组件的 builder 参数
 */
@Builder
export function RefreshIndicator() {
  Row() {
    LoadingProgress()
      .width(DesignConstants.COMPONENT.ICON_SIZE_MD)
      .height(DesignConstants.COMPONENT.ICON_SIZE_MD)
      .color(AppThemeManager.getSafeCurrentPalette().accentBackground)
      .accessibilityText('正在刷新')
    Text('正在刷新…')
      .fontSize(DesignConstants.FONT.SIZE.MD)
      .margin({ left: DesignConstants.SPACING.SM })
      .fontColor(AppThemeManager.getSafeCurrentPalette().textSecondary)
  }
  .width('100%')
  .justifyContent(FlexAlign.Center)
  .padding(DesignConstants.SPACING.SM)
  .accessibilityText('正在刷新')
}
