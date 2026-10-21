/**
 * SearchPageHeader
 * - 内联搜索框头部（替代 SearchPage 手搓头部）
 * - 结构：圆形返回按钮 + 占满剩余宽度的搜索输入框（带 search glyph）
 * - 复用 SystemPageHeader 同套主题/安全区/DesignConstants 原语
 *
 * 与 SystemPageHeader 的差异：头部主体是 TextInput 而非标题文本，故独立组件。
 */

import { DesignConstants } from '../constants/DesignConstants';
import { AppThemeManager } from '../core/theme/AppThemeManager';
import { DEFAULT_RESOLVED_THEME_ID, KEY_RESOLVED_THEME_ID } from '../core/theme/AppThemeTypes';
import { AppIcon } from './SystemIcon';
import { SafeAreaUtils } from '../core/utils/SafeAreaUtils';
import { LucideIcon } from '../components/LucideIcon';
import { SurfaceIconButton } from './SurfaceIconButton';

@Component
export struct SearchPageHeader {
  /** 搜索关键词（双向绑定）。 */
  @Link keyword: string;
  /** 页面标题（对齐前端「搜索」大标题模式：标题行 + 下方输入框；空串走内联旧模式）。 */
  @Prop title: string = '';
  /** 占位提示文案。 */
  @Prop placeholder: string = '搜索用户、动态、地点或话题';
  /** 返回回调（通常 AppRouter.pop）。 */
  onBack: () => void = () => {};
  /** 聚焦态变化回调。 */
  onFocusChange: ((focused: boolean) => void) | null = null;
  /** 输入变化回调。 */
  onChange: ((value: string) => void) | null = null;
  /** 提交回调。 */
  onSubmit: (() => void) | null = null;
  @StorageLink('safeAreaTop') safeAreaTop: number = 0;
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = DEFAULT_RESOLVED_THEME_ID;
  @State private isFocused: boolean = false;

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  private getSurfaceBackground(): ResourceColor {
    return this.getPalette().surfaceBackground;
  }

  private getSurfaceBorder(): ResourceColor {
    return this.getPalette().floatingBorderColor;
  }

  @Builder
  private BackCircleButton() {
    // 返回钮统一走 SurfaceIconButton（与 SystemPageHeader 同款凸起中性 40vp：
    // a11y + spring 按压 + 44 命中区由组件内聚，审查 中-1/低-4）。
    // isBackPressed 手写按压态由组件的 spring 缩放替代，一并移除。
    SurfaceIconButton({
      icon: AppIcon.BACK,
      kind: 'surface',
      buttonSize: DesignConstants.LAYOUT.BACK_BUTTON_SIZE,
      glyphSize: DesignConstants.FONT.SIZE.LG_PLUS,
      accessibilityLabel: '返回',
      onPress: () => {
        this.onBack();
      }
    })
  }

  @Builder
  private SearchInputRow() {
    // 对齐前端 SearchInput：gap-2.5(10) + 框高 39.14 → SEARCH_INPUT_HEIGHT
    Row({ space: DesignConstants.SPACING.MD }) {
      LucideIcon({
          icon: AppIcon.SEARCH,
          iconSize: DesignConstants.COMPONENT.ICON_SIZE_2XS,
          color: this.getPalette().textSecondary
        })

      TextInput({ placeholder: this.placeholder, text: this.keyword })
        .layoutWeight(1)
        .height(DesignConstants.COMPONENT.SEARCH_INPUT_HEIGHT)
        // 完全透明化：消除 TextInput 系统主题默认背景/圆角溢出
        .backgroundColor(DesignConstants.COLOR.TRANSPARENT)
        .background(DesignConstants.COLOR.TRANSPARENT)
        .fontSize(DesignConstants.FONT.SIZE.SM)
        .placeholderColor(this.getPalette().textTertiary)
        .fontColor(this.getPalette().textPrimary)
        .border({ width: 0 })
        .borderRadius(0)
        .padding(0)
        .onFocus(() => {
          this.isFocused = true;
          if (this.onFocusChange) {
            this.onFocusChange(true);
          }
        })
        .onChange((value: string) => {
          this.keyword = value;
          if (this.onChange) {
            this.onChange(value);
          }
        })
        .onSubmit(() => {
          if (this.onSubmit) {
            this.onSubmit();
          }
        })
        .onBlur(() => {
          this.isFocused = false;
          if (this.onFocusChange) {
            this.onFocusChange(false);
          }
        })
    }
    .height(DesignConstants.COMPONENT.SEARCH_INPUT_HEIGHT)
    .borderRadius(DesignConstants.SHAPE.INPUT)
    .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG })
    .backgroundColor(this.getSurfaceBackground())
    .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.isFocused ? this.getPalette().accentText : this.getSurfaceBorder() })
    // 对齐前端 input-surface 聚焦契约：边框加深承载焦点（内凹阴影），无彩色光晕
    .shadow(this.getPalette().surfaceShadow)
    // 裁剪子组件，防止 TextInput 系统默认视觉溢出圆角
    .clip(true)
  }

  build() {
    if (this.title.length > 0) {
      // 标题模式（对齐前端搜索页）：标题行（返回 + 大标题）+ 下方通栏输入框
      Column() {
        Row({ space: DesignConstants.SPACING.MD }) {
          this.BackCircleButton()
          Text(this.title)
            .fontSize(DesignConstants.FONT.SIZE.XXL)
            .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
            .fontColor(this.getPalette().textPrimary)
            .layoutWeight(1)
        }
        .width('100%')
        .alignItems(VerticalAlign.Center)

        this.SearchInputRow()
      }
      .width('100%')
      .padding({
        left: DesignConstants.SPACING.LG,
        right: DesignConstants.SPACING.LG,
        top: SafeAreaUtils.resolveTopPadding(this.safeAreaTop, 0),
        bottom: DesignConstants.SPACING.MD
      })
      .backgroundColor(this.getPalette().pageBackground)
      .border({ width: { bottom: DesignConstants.BORDER_WIDTH.HAIRLINE }, color: this.getPalette().borderColor })
      .expandSafeArea([SafeAreaType.SYSTEM], [SafeAreaEdge.TOP])
    } else {
      // 内联模式（返回 + 输入框同排）
      Row({ space: DesignConstants.SPACING.MD }) {
        this.BackCircleButton()
        Row() {
          this.SearchInputRow()
        }
        .layoutWeight(1)
      }
      .width('100%')
      .padding({
        left: DesignConstants.SPACING.LG,
        right: DesignConstants.SPACING.LG,
        top: SafeAreaUtils.resolveTopPadding(this.safeAreaTop),
        bottom: DesignConstants.SPACING.SM
      })
      .backgroundColor(DesignConstants.COLOR.TRANSPARENT)
      .border({ width: { bottom: DesignConstants.BORDER_WIDTH.HAIRLINE }, color: this.getSurfaceBorder() })
      .expandSafeArea([SafeAreaType.SYSTEM], [SafeAreaEdge.TOP])
    }
  }
}
