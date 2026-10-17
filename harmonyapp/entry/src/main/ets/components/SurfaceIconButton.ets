import { DesignConstants } from '../constants/DesignConstants';
import { AppThemeManager } from '../core/theme/AppThemeManager';
import { AppThemeShadow, DEFAULT_RESOLVED_THEME_ID, KEY_RESOLVED_THEME_ID } from '../core/theme/AppThemeTypes';
import { AppIcon } from './SystemIcon';
import { LucideIcon } from './LucideIcon';
import { curves } from '@kit.ArkUI';

export type SurfaceIconButtonKind = 'primary' | 'neutral' | 'surface';

@Component
export struct SurfaceIconButton {
  @Prop icon: AppIcon = AppIcon.BACK;
  @Prop kind: SurfaceIconButtonKind = 'primary';
  @Prop buttonSize: number = DesignConstants.LAYOUT.ICON_BUTTON_SIZE; // 36
  /**
   * 图标视觉尺寸。**0 = 按按钮尺寸自动推导**（缺省推荐）：
   * 前端 ActionIconButton 契约是 `<Icon size={Math.round(size * 0.42)} />`，
   * 即 42 钮配 18、36 钮配 15。此前本组件默认 16 且多处调用传 ICON_SIZE_SM(20)，
   * 20/36 ≈ 56% 的图标占比明显重于前端的 42%，页头图标看起来「比标题还抢眼」
   * （2026-10-17 截图复核）。显式传值仅在需要刻意放大/缩小时使用。
   */
  @Prop glyphSize: number = 0;
  @Prop disabled: boolean = false;
  /** 无障碍语义描述（图标钮无文字，读屏依赖此标签，如全局返回钮「返回」） */
  @Prop accessibilityLabel: string = '';
  /** 图标色逃逸口（媒体镀铬白字、品牌色等特殊着色）；空串走 kind 主题色。
   *  仅限媒体浮层/hero 封面等主题色不可达的场景，常规页面禁止使用。 */
  @Prop iconColor: string | ResourceColor = '';
  onPress?: () => void;
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = DEFAULT_RESOLVED_THEME_ID;

  @State isPressed: boolean = false;
  /** hover（仅键鼠外接触发） */
  @State isHovered: boolean = false;

  /** 图标尺寸：显式值优先；否则按按钮尺寸 × 0.42（对齐前端 ActionIconButton 契约） */
  private getGlyphSize(): number {
    if (this.glyphSize > 0) {
      return this.glyphSize;
    }
    return Math.round(this.buttonSize * DesignConstants.COMPONENT.ICON_GLYPH_RATIO);
  }

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  private getBackgroundColor(): ResourceColor | Color {
    if (this.kind === 'primary') {
      return this.getPalette().accentBackground;
    }
    // surface → 凸起中性钮（对齐前端 ActionIconButton variant=surface 的浅底圆钮）
    if (this.kind === 'surface') {
      return this.getPalette().surfaceBackground;
    }
    // neutral → 对齐前端 ActionIconButton variant=ghost：透明底、无边框、无阴影
    return DesignConstants.COLOR.TRANSPARENT;
  }

  private getBorderColor(): ResourceColor | Color {
    if (this.kind === 'primary') {
      return this.getPalette().accentBackground;
    }
    // surface → hairline 常规边框（凸起钮的描边语义）
    if (this.kind === 'surface') {
      return this.getPalette().borderColor;
    }
    return DesignConstants.COLOR.TRANSPARENT;
  }

  private getShadowColor(): ShadowOptions {
    // 对齐前端 header-action / ui/button：accent 实底图标钮用中性 raised 投影，
    // hover 抬档 raised-md（无彩色光晕）
    if (this.kind === 'primary') {
      if (this.isHovered && !this.disabled) {
        return this.getPalette().elevatedShadow;
      }
      return this.getPalette().surfaceShadow;
    }
    if (this.kind === 'surface') {
      return this.getPalette().surfaceShadow;
    }
    return new AppThemeShadow(0, DesignConstants.COLOR.TRANSPARENT, 0, 0);
  }

  /** neutral/surface hover 淡底填充（键鼠）；primary 保持实底（阴影承载 hover） */
  private getHoverOverlayBackground(): ResourceColor {
    if (this.kind !== 'primary' && this.isHovered && !this.disabled) {
      return this.getPalette().accentMuted;
    }
    return this.getBackgroundColor();
  }

  private getIconColor(): ResourceColor | Color {
    // 逃逸口显式着色优先（媒体镀铬等）
    if (this.iconColor !== '') {
      return this.iconColor;
    }
    if (this.kind === 'primary') {
      return this.getPalette().accentText;
    }
    // ghost/surface 图标用主文字色（对齐前端 PageHeader 末尾透明钮——深色保证在亮底上的可见性）
    return this.getPalette().textPrimary;
  }

  build() {
    // 触控命中区 ≥44vp（对齐 WCAG 2.5.5 / HarmonyOS 最小热区）：外层透明扩展，
    // 内层保持视觉尺寸（buttonSize 默认 36 不变），按压/点击绑定在外层。
    Stack() {
      Button() {
        LucideIcon({
          icon: this.icon,
          iconSize: this.getGlyphSize(),
          color: this.getIconColor()
        })
      }
      // Circle：同 SystemButton，Normal 类型忽略 borderRadius
      .type(ButtonType.Circle)
      .stateEffect(false)
      .width(this.buttonSize)
      .height(this.buttonSize)
      .backgroundColor(this.getHoverOverlayBackground())
      .borderRadius(DesignConstants.RADIUS.FULL)
      .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getBorderColor() })
      .shadow(this.getShadowColor())
    }
    .width(Math.max(this.buttonSize, DesignConstants.LAYOUT.MIN_HIT_TARGET))
    .height(Math.max(this.buttonSize, DesignConstants.LAYOUT.MIN_HIT_TARGET))
    .alignContent(Alignment.Center)
    .enabled(!this.disabled)
    .opacity(this.disabled ? DesignConstants.OPACITY.DISABLED : 1)
    .accessibilityText(this.accessibilityLabel)
    .scale(this.isPressed ?
      { x: DesignConstants.INTERACTION.SCALE_PRESSED_ICON, y: DesignConstants.INTERACTION.SCALE_PRESSED_ICON } :
      { x: 1, y: 1 })
    .animation({
      curve: curves.springMotion(
        DesignConstants.INTERACTION.SPRING_RESPONSE_PRESS,
        DesignConstants.INTERACTION.SPRING_DAMPING_PRESS
      )
    })
    .onTouch((event: TouchEvent) => {
      if (this.disabled) {
        return;
      }
      if (event.type === TouchType.Down) {
        this.isPressed = true;
      } else if (event.type === TouchType.Up || event.type === TouchType.Cancel) {
        this.isPressed = false;
      }
    })
    .onClick(() => {
      if (this.disabled) {
        return;
      }
      this.onPress?.();
    })
    .onHover((isHover: boolean) => {
      if (!this.disabled) {
        this.isHovered = isHover;
      }
    })
  }
}
