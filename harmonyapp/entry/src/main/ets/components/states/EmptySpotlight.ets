/**
 * EmptySpotlight — 场景化空态装饰层（纯装饰，对齐 web EmptyState.spotlight）
 *
 * 跨端 spec（docs / fe report 同步）：超大空态（整页/整块内容区的访客引导、我的内容空态）
 * 的中心「信标」意象：中心光晕 + 内外双描边环 + 四向星点。小面板空态不要开启。
 *
 * 几何（与 web .empty-spotlight 1:1）：
 * - 中心光晕 320×320vp 圆：径向渐变 ambient-glow2(中心) → 透明 70%；
 * - 外环 288×288vp / 内环 168×168vp，1vp 发丝描边 ambient-line，无填充；
 * - 四向星点 4×4vp，位于外环半径 144vp 的 45° 等分位（相对中心 ±102, ±102），ambient-dot。
 * 颜色全部取 ThemeConstants.AMBIENT（web --cst-ambient-* 同源 token），无硬编码色值。
 * 约束：无动画、不参与布局、不拦截触摸、不进无障碍树、z 序在内容之下。
 */
import { DesignConstants } from '../../constants/DesignConstants';
import { AppThemeManager } from '../../core/theme/AppThemeManager';
import { AmbientColorConfig } from '../../core/theme/ThemeConstants';
import { DEFAULT_RESOLVED_THEME_ID, KEY_RESOLVED_THEME_ID } from '../../core/theme/AppThemeTypes';

const SPOTLIGHT_SIZE: number = 320;
const GLOW_SIZE: number = 320;
const RING_OUTER_SIZE: number = 288;
const RING_INNER_SIZE: number = 168;
const DOT_SIZE: number = 4;
const DOT_OFFSET: number = 102;
const RING_WIDTH: number = DesignConstants.BORDER_WIDTH.HAIRLINE;
const TRANSPARENT: string = '#00000000';

@Component
export struct EmptySpotlight {
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = DEFAULT_RESOLVED_THEME_ID;

  build() {
    Stack({ alignContent: Alignment.Center }) {
      // 中心光晕：径向渐变（中心 ambient-glow2 → 70% 处透明）
      Column()
        .width(GLOW_SIZE)
        .height(GLOW_SIZE)
        .borderRadius(DesignConstants.RADIUS.FULL)
        .radialGradient({
          center: ['50%', '50%'],
          radius: GLOW_SIZE / 2,
          colors: [
            [this.ambient().glow2, 0],
            [TRANSPARENT, 0.7],
            [TRANSPARENT, 1]
          ]
        })

      // 外环 / 内环：1vp 发丝描边，无填充
      Column()
        .width(RING_OUTER_SIZE)
        .height(RING_OUTER_SIZE)
        // ArkUI 陷阱：.border() 后设且未显式给 radius 会把 borderRadius 覆盖为 0（方框），
        // 因此 border 必须在 borderRadius 之前（见 hotspot-harmony-ui skill「布局陷阱 1」）。
        .border({ width: RING_WIDTH, color: this.ambient().line })
        .borderRadius(DesignConstants.RADIUS.FULL)
      Column()
        .width(RING_INNER_SIZE)
        .height(RING_INNER_SIZE)
        .border({ width: RING_WIDTH, color: this.ambient().line })
        .borderRadius(DesignConstants.RADIUS.FULL)

      // 四向星点：外环 45° 等分位（translate 不占布局，保持居中锚点）
      this.SpotlightDot(DOT_OFFSET, DOT_OFFSET)
      this.SpotlightDot(-DOT_OFFSET, DOT_OFFSET)
      this.SpotlightDot(-DOT_OFFSET, -DOT_OFFSET)
      this.SpotlightDot(DOT_OFFSET, -DOT_OFFSET)
    }
    .width(SPOTLIGHT_SIZE)
    .height(SPOTLIGHT_SIZE)
    // 装饰层：不拦截触摸、不进入无障碍树、不参与页面语义
    .hitTestBehavior(HitTestMode.None)
    .accessibilityLevel('no')
  }

  @Builder
  private SpotlightDot(offsetX: number, offsetY: number) {
    Column()
      .width(DOT_SIZE)
      .height(DOT_SIZE)
      .borderRadius(DOT_SIZE / 2)
      .backgroundColor(this.ambient().dot)
      .translate({ x: offsetX, y: offsetY })
  }

  private ambient(): AmbientColorConfig {
    return AppThemeManager.getAmbientById(this.resolvedThemeId);
  }
}
