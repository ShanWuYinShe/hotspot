import { DesignConstants } from '../constants/DesignConstants';
import { MediaUrlUtil } from '../core/utils/MediaUrlUtil';
import { ImageUrlUtil } from '../core/utils/ImageUrlUtil';
import { MotionUtil } from '../core/accessibility/MotionUtil';
import { AppThemeManager } from '../core/theme/AppThemeManager';
import { AppThemeVisuals } from '../core/theme/AppThemeVisuals';
import { DEFAULT_RESOLVED_THEME_ID, KEY_RESOLVED_THEME_ID } from '../core/theme/AppThemeTypes';
import { AppIcon } from './SystemIcon';
import { curves } from '@kit.ArkUI';
import { LucideIcon } from '../components/LucideIcon';
import { CardScrimTop, CardScrimBottom, CardFallbackBackground } from './media/MediaOverlay';

@Builder
function emptyNearbyCardSlot() {
}

/**
 * NearbyImageCard — 图片优先的附近内容卡片统一原语。
 * 对齐前端 data-display/NearbyImageCard 契约：
 *   - 图片 objectFit Cover 铺满卡片，底部 footer 托盘（尾随闭包）承载元数据/操作，
 *     托盘 surfaceElevated 底 + hairline 顶边
 *   - 图片层叠 media scrim 渐变保证图上徽章可读；无图/加载失败回退
 *     accent 渐变占位层 + 首字
 *   - 图上角徽章参数化：pill（距离/精选文字胶囊）或 circle（隐私圆徽章），
 *     topRight / topLeft 对齐
 *   - 整卡按压 spring 缩放（SCALE_PRESSED_CARD）；id 以 CHILD_CONTROL_PREFIX
 *     开头的子控件（关注钮等行内操作）触摸不带动整卡按压，需自行处理反馈
 */
@Reusable
@Component
export struct NearbyImageCard {
  /** 行内子控件 id 前缀约定：设置 .id(`${CHILD_CONTROL_PREFIX}xxx`) 的子控件不触发整卡按压 */
  static readonly CHILD_CONTROL_PREFIX: string = 'nic-child-';

  @Watch('onImageUrlChange')
  @Prop imageUrl: string = '';
  /** R6：首屏优先（900）/非首屏降档（800），与 fe NearbyImageCard 对齐 */
  @Prop priority: boolean = false;
  @Prop imageAlt: string = '';
  /** 宽高比（宽/高，3:4 竖卡传 0.75）；为 0 时高度由 fixedHeight 或父容器决定 */
  @Prop cardAspectRatio: number = 0;
  /** 固定高度（aspectRatio 为 0 时生效；同为 0 则撑满父容器） */
  @Prop fixedHeight: number = 0;
  @Prop cardRadius: number = DesignConstants.SHAPE.COMPACT_CARD;
  /** 无图占位首字来源（昵称/群名） */
  @Prop placeholderText: string = '';
  /** overlay fallback 居中水印图标（opacity 0.10；用户卡用 PERSON、群卡用 GROUP、瞬间用 PHOTO） */
  @Prop fallbackIcon: AppIcon = AppIcon.PHOTO;
  /** 图上徽章：pill 文本（空 = 不渲染 pill） */
  @Prop badgeText: string = '';
  /** 图上徽章图标（pill 前缀图标 / circle 徽章主体） */
  @Prop badgeIcon: AppIcon = AppIcon.LOCATION;
  /** circle 徽章（仅图标）时置 true */
  @Prop badgeShape: 'pill' | 'circle' = 'pill';
  /** pill 底色变体：glass = 半透明玻璃（距离/精选）；accent = accent 实底白字（分类胶囊） */
  @Prop badgeVariant: 'glass' | 'accent' = 'glass';
  @Prop badgeAlign: 'topRight' | 'topLeft' = 'topRight';
  @Prop disabled: boolean = false;
  /**
   * R2 整卡图片卡模式（Image-first，BRIEF §A）：图片铺满圆角卡，footer 内容覆盖在
   * 底部遮罩上（白字/白图标），去掉不透明托盘与描边；no-image 时用主题 fallback
   * + 主题文字色。调用方 footer 需根据 onMediaStateChange 切换文字色。
   */
  /** R2 整卡图片卡模式（Image-first，BRIEF §A）：图片铺满圆角卡，footer 内容覆盖在
   * 底部遮罩上（白字/白图标），去掉不透明托盘与描边；no-image 时用主题 fallback
   * + 主题文字色。注意：不能命名 overlay（与 ArkUI 内置 overlay 属性冲突）。 */
  @Prop mediaOverlay: boolean = false;
  /** 卡片当前是否处于「图上有内容」状态（false = fallback，footer 应换主题文字色） */
  onMediaStateChange?: (mediaActive: boolean) => void;
  onPress?: () => void = () => {
  };
  @BuilderParam footer: () => void = emptyNearbyCardSlot;

  @State private imageFailed: boolean = false;
  /** R6：onComplete 渐显（reduced-motion 直接显示） */
  @State private imageLoaded: boolean = false;
  @State private isPressed: boolean = false;
  @State private slideOut: boolean = false;
  @State private cardWidth: number = 0;
  @State private cardHeight: number = 0;
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = DEFAULT_RESOLVED_THEME_ID;

  private onImageUrlChange(): void {
    this.imageFailed = false;
    this.imageLoaded = false;
    this.notifyMediaState();
  }

  aboutToAppear(): void {
    this.notifyMediaState();
  }

  private hasMedia(): boolean {
    return this.imageUrl.length > 0 && !this.imageFailed;
  }

  private notifyMediaState(): void {
    if (this.onMediaStateChange) {
      this.onMediaStateChange(this.hasMedia());
    }
  }

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  private getPlaceholderInitial(): string {
    if (!this.placeholderText) {
      return '?';
    }
    return Array.from(this.placeholderText)[0] ?? '?';
  }

  private isChildControlTouch(event: TouchEvent): boolean {
    const targetId: string = event.target?.id ?? '';
    return targetId.startsWith(NearbyImageCard.CHILD_CONTROL_PREFIX);
  }

  // topRight 用「右缘对齐 + markAnchor 右锚」表达（position 的 x 只从左缘起算，
  // 原两分支同坐标导致 topRight 徽章实际渲染在左上）
  private getBadgePosition(): Position {
    return this.badgeAlign === 'topRight'
      ? { x: '100%', y: DesignConstants.SPACING.MD2 }
      : { x: DesignConstants.SPACING.MD2, y: DesignConstants.SPACING.MD2 };
  }

  private getBadgeAnchor(): Position {
    return this.badgeAlign === 'topRight'
      ? { x: '100%', y: 0 }
      : { x: 0, y: 0 };
  }

  /** topRight 右缘对齐后向左平移回内边距（等价 left: 100% - MD2） */
  private getBadgeAlignTranslate(): TranslateOptions {
    return this.badgeAlign === 'topRight'
      ? { x: -DesignConstants.SPACING.MD2 }
      : { x: 0 };
  }

  @Builder
  private buildBadge() {
    if (this.badgeShape === 'circle') {
      Row() {
        LucideIcon({
            icon: this.badgeIcon,
            iconSize: DesignConstants.COMPONENT.ICON_SIZE_2XS,
            color: DesignConstants.COLOR.WHITE
          })
      }
      .width(DesignConstants.LAYOUT.UNREAD_BADGE_SIZE)
      .height(DesignConstants.LAYOUT.UNREAD_BADGE_SIZE)
      .borderRadius(DesignConstants.RADIUS.FULL)
      .backgroundColor(this.mediaOverlay ? DesignConstants.COLOR.MEDIA_SCRIM_STRONG : DesignConstants.COLOR.IMAGE_BADGE_BG)
      .border(this.mediaOverlay ?
        { width: 0 } :
        { width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: DesignConstants.COLOR.IMAGE_BADGE_BORDER })
      .justifyContent(FlexAlign.Center)
      .markAnchor(this.getBadgeAnchor())
      .position(this.getBadgePosition())
      .translate(this.getBadgeAlignTranslate())
    } else if (this.badgeText.length > 0) {
      Row({ space: DesignConstants.SPACING.HALF }) {
        if (this.badgeVariant === 'glass') {
          LucideIcon({
              icon: this.badgeIcon,
              iconSize: DesignConstants.COMPONENT.ICON_SIZE_2XS,
              color: DesignConstants.COLOR.WHITE
            })
        }
        Text(this.badgeText)
          .fontSize(DesignConstants.FONT.SIZE.XS2)
          .fontColor(DesignConstants.COLOR.WHITE)
      }
      .alignItems(VerticalAlign.Center)
      .padding({
        left: this.badgeVariant === 'accent' ? DesignConstants.SPACING.SM2 : DesignConstants.SPACING.SM,
        right: this.badgeVariant === 'accent' ? DesignConstants.SPACING.SM2 : DesignConstants.SPACING.SM,
        top: DesignConstants.SPACING.THREE,
        bottom: DesignConstants.SPACING.THREE
      })
      .borderRadius(DesignConstants.RADIUS.FULL)
      .backgroundColor(this.badgeVariant === 'accent' ?
        this.getPalette().accentBackground :
        (this.mediaOverlay ? DesignConstants.COLOR.MEDIA_SCRIM_STRONG : DesignConstants.COLOR.IMAGE_BADGE_BG))
      .border(this.badgeVariant === 'accent' || this.mediaOverlay ?
        { width: 0 } :
        { width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: DesignConstants.COLOR.IMAGE_BADGE_BORDER })
      .markAnchor(this.getBadgeAnchor())
      .position(this.getBadgePosition())
      .translate(this.getBadgeAlignTranslate())
    }
  }

  @Builder
  private buildFallbackLayer() {
    Stack() {
      // 强调色透明叠层占位（对齐前端 surface-accent-gradient）：以 surfaceElevated 实底
      // 托住 accent 两档透明渐变。直接用 accentMuted→surfaceBackground 在深色主题下
      // 两个近黑色相融，卡片会渲染成一片近乎全黑的空白
      Column()
        .width('100%')
        .height('100%')
        .backgroundColor(this.getPalette().surfaceElevated)

      Column()
        .width('100%')
        .height('100%')
        .linearGradient({
          angle: 160,
          colors: [
            [AppThemeVisuals.withAlpha(this.getPalette().accentBackground, 0.30), 0.0],
            [AppThemeVisuals.withAlpha(this.getPalette().accentBackground, 0.06), 1.0]
          ] as [ResourceColor, number][]
        })

      // 占位对齐前端 NearbyImageCard.svelte：accent-soft 圆(size-16)内 text-3xl 半粗首字母，
      // 固定尺寸不随卡片放大（原自适应大字在图片位喧宾夺主，UI 审查 D7）。
      // 前景色：**淡底（accentSoft/accentMuted）上的文字必须用 accent 本色**——
      // accentText 的语义是「accent 亮实底上的反色前景」（深色主题 #071426），
      // 配到同主题 accentMuted 深蓝底（#182943）上对比度约 1.2:1，首字母几乎不可见。
      //
      // `placeholderText` 为空时**不画占位圆**：调用方用空值表达"这张卡没有可展示的图片、
      // 只要渐变底"（如无封面的精选群卡）。否则会画出 '?' 圆，在矮卡上还会被底部托盘裁切
      // （2026-10-21 端上实测）。
      if (this.placeholderText) {
        Column() {
          Text(this.getPlaceholderInitial())
            .fontSize(DesignConstants.FONT.SIZE.XXXL)
            .fontWeight(DesignConstants.FONT.WEIGHT.SEMIBOLD)
            .fontColor(this.getPalette().textAccentOnSoft)
        }
        // web 占位圆 size-16 = 64vp（原 48 偏小，首字母显得局促）
        .width(DesignConstants.COMPONENT.EMPTY_ICON_SIZE)
        .height(DesignConstants.COMPONENT.EMPTY_ICON_SIZE)
        .borderRadius(DesignConstants.RADIUS.FULL)
        .backgroundColor(this.getPalette().accentMuted)
        // 视觉居中于上方可见图片区（避开底部托盘遮挡）
        .margin({ bottom: DesignConstants.SPACING.XXL * 2 })
      }
    }
    .width('100%')
    .height('100%')
  }

  @Builder
  private buildImageLayer() {
    Stack() {
      if (this.imageUrl && !this.imageFailed) {
        // R8：fallback 层（主题渐变 + 首字母/图标）常驻图片底层，加载中不再出现纯灰块；
        // 图片失败时仅隐藏 Image，占位层仍在。
        if (this.mediaOverlay) {
          CardFallbackBackground({ icon: this.fallbackIcon })
        } else {
          this.buildFallbackLayer()
        }
        Image(ImageUrlUtil.optimized(MediaUrlUtil.normalize(this.imageUrl), this.priority ? 900 : 800))
          .width('100%')
          .height('100%')
          .objectFit(ImageFit.Cover)
          .draggable(false)
          .accessibilityText(this.imageAlt)
          .opacity(this.imageLoaded || MotionUtil.reduced() ? 1 : 0)
          .animation({
            duration: MotionUtil.reduced() ? 0 : DesignConstants.ANIMATION.DURATION_SHORT,
            curve: DesignConstants.ANIMATION.CURVE_EASE_OUT
          })
          .onComplete(() => { this.imageLoaded = true; })
          .onError(() => {
            this.imageFailed = true;
          })

        // overlay 模式用 R2 顶/底遮罩（下方统一叠加）；旧托盘模式保留轻渐变
        if (!this.mediaOverlay) {
          Column()
            .width('100%')
            .height('100%')
            .linearGradient({
              angle: 180,
              colors: [
                [DesignConstants.COLOR.IMAGE_OVERLAY_START, 0.0],
                [DesignConstants.COLOR.IMAGE_OVERLAY_END, 0.55]
              ] as [ResourceColor, number][]
            })
            .hitTestBehavior(HitTestMode.None)
        }
      } else {
        if (this.mediaOverlay) {
          CardFallbackBackground({ icon: this.fallbackIcon })
        } else {
          this.buildFallbackLayer()
        }
      }

      // 非 overlay：badge 直接叠在图上；overlay 时 badge 要画在遮罩之上（见 build()）
      if (!this.mediaOverlay) {
        this.buildBadge()
      }
    }
    .width('100%')
    .height('100%')
  }

  build() {
    Stack({ alignContent: Alignment.Bottom }) {
      this.buildImageLayer()

      if (this.mediaOverlay) {
        if (this.hasMedia()) {
          // 顶/底遮罩（纯装饰，不拦截触摸）；fallback 不用白字遮罩（BRIEF 例外）
          Column() {
            CardScrimTop()
            Blank()
            CardScrimBottom({ heightRatio: '62%' })
          }
          .width('100%')
          .height('100%')
          .hitTestBehavior(HitTestMode.None)
          .accessibilityLevel('no')
        }

        // badge 画在遮罩之上，保证深遮罩徽章/白字不被二次压暗
        Stack() {
          this.buildBadge()
        }
        .width('100%')
        .height('100%')
        .hitTestBehavior(HitTestMode.None)

        // 内容覆盖在遮罩上（footer 由调用方按 onMediaStateChange 切白字/主题字）
        Column() {
          this.footer()
        }
        .width('100%')
        .alignItems(HorizontalAlign.Start)
      } else {
        // 底部托盘（surfaceElevated 底 + hairline 顶边，对齐前端 card-menu tray）
        Column() {
          this.footer()
        }
        .width('100%')
        .backgroundColor(this.getPalette().surfaceElevated)
        .border({
          width: { top: DesignConstants.BORDER_WIDTH.HAIRLINE },
          color: { top: this.getPalette().borderColor }
        })
      }
    }
    .width('100%')
    .aspectRatio(this.cardAspectRatio > 0 ? this.cardAspectRatio : undefined)
    .height(this.cardAspectRatio > 0 ? undefined : (this.fixedHeight > 0 ? this.fixedHeight : '100%'))
    .borderRadius(this.cardRadius)
    .clip(true)
    .onAreaChange((_oldValue: Area, newValue: Area) => {
      this.cardWidth = newValue.width as number;
      this.cardHeight = newValue.height as number;
    })
    .scale(this.isPressed && !this.disabled ?
      { x: DesignConstants.INTERACTION.SCALE_PRESSED_CARD, y: DesignConstants.INTERACTION.SCALE_PRESSED_CARD } :
      { x: 1, y: 1 })
    .animation({
      curve: curves.springMotion(
        DesignConstants.INTERACTION.SPRING_RESPONSE_PRESS,
        DesignConstants.INTERACTION.SPRING_DAMPING_PRESS
      )
    })
    .onTouch((event: TouchEvent) => {
      if (this.disabled) {
        this.isPressed = false;
        return;
      }
      // 行内子控件（CHILD_CONTROL_PREFIX 约定）自行处理按压，不带动整卡缩放
      if (this.isChildControlTouch(event)) {
        this.isPressed = false;
        return;
      }
      if (event.type === TouchType.Down) {
        this.isPressed = true;
        this.slideOut = false;
      } else if (event.type === TouchType.Move && event.touches.length > 0) {
        const touch = event.touches[0];
        if (this.isPressed &&
          (touch.x < 0 || touch.y < 0 || touch.x > this.cardWidth || touch.y > this.cardHeight)) {
          this.isPressed = false;
          this.slideOut = true;
        }
      } else if (event.type === TouchType.Up) {
        // 点击语义交由 onClick 触发（键盘/读屏可达）；此处只复位按压视觉
        this.isPressed = false;
        this.slideOut = false;
      } else if (event.type === TouchType.Cancel) {
        this.isPressed = false;
        this.slideOut = false;
      }
    })
    .onClick(() => {
      if (!this.disabled && this.onPress) {
        this.onPress();
      }
    })
  }
}
