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
import { KEY_RESOLVED_THEME_ID } from '../core/theme/AppThemeTypes';
import { AppIcon } from './SystemIcon';
import { SafeAreaUtils } from '../core/utils/SafeAreaUtils';
import curves from '@ohos.curves';
import { LucideIcon } from '../components/LucideIcon';

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
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'vibrant_light';
  @State private isBackPressed: boolean = false;
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
    Button() {
      LucideIcon({
          icon: AppIcon.BACK,
          iconSize: DesignConstants.FONT.SIZE.MD,
          color: this.getPalette().textSecondary
        })
    }
    .type(ButtonType.Circle)
    .width(DesignConstants.LAYOUT.ICON_BUTTON_SIZE)
    .height(DesignConstants.LAYOUT.ICON_BUTTON_SIZE)
    .backgroundColor(this.getSurfaceBackground())
    .shadow(this.getPalette().surfaceShadow)
    .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getSurfaceBorder() })
    .scale(this.isBackPressed
      ? { x: DesignConstants.INTERACTION.SCALE_PRESSED_ICON, y: DesignConstants.INTERACTION.SCALE_PRESSED_ICON }
      : { x: 1, y: 1 })
    .animation({
      curve: curves.springMotion(
        DesignConstants.INTERACTION.SPRING_RESPONSE_PRESS,
        DesignConstants.INTERACTION.SPRING_DAMPING_PRESS
      )
    })
    .onTouch((event: TouchEvent) => {
      if (event.type === TouchType.Down) {
        this.isBackPressed = true;
      } else if (event.type === TouchType.Up || event.type === TouchType.Cancel) {
        this.isBackPressed = false;
      }
    })
    .onClick(() => {
      this.onBack();
    })
  }

  @Builder
  private SearchInputRow() {
    Row({ space: DesignConstants.SPACING.SM }) {
      LucideIcon({
          icon: AppIcon.SEARCH,
          iconSize: DesignConstants.FONT.SIZE.MD,
          color: this.getPalette().textSecondary
        })

      TextInput({ placeholder: this.placeholder, text: this.keyword })
        .layoutWeight(1)
        .height(DesignConstants.CHAT.INPUT_HEIGHT)
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
    .height(DesignConstants.CHAT.INPUT_HEIGHT)
    .borderRadius(DesignConstants.SHAPE.INPUT)
    .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG })
    .backgroundColor(this.getSurfaceBackground())
    .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.isFocused ? this.getPalette().accentText : this.getSurfaceBorder() })
    .shadow(this.isFocused ? this.getPalette().accentShadow : this.getPalette().surfaceShadow)
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
        left: DesignConstants.SPACING.LG2,
        right: DesignConstants.SPACING.LG2,
        top: SafeAreaUtils.resolveTopPadding(this.safeAreaTop, DesignConstants.SPACING.SM),
        bottom: DesignConstants.SPACING.MD2
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
