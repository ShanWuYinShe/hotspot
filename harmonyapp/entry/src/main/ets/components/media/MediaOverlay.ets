/**
 * MediaOverlay — 整卡图片卡共享覆盖层原语（R2，跨端 spec 见 BRIEF §A）
 *
 * - CardScrimTop / CardScrimBottom：图片铺满模式下的顶/底遮罩（纯装饰、不拦截触摸、不进无障碍树）
 * - CardFallbackBackground：无图 fallback 主题渐变底（配合主题文字色，不使用白字遮罩）
 * - MediaOverlayBadge：深色强遮罩胶囊（对齐 web --cst-surface-media-scrim-strong）+ 白字
 * - MediaActionButton：图片上的白色图标操作键（热区 44，白色计数）
 *
 * token：遮罩/fallback 来自 palette.cardScrim* / cardFallback*（镜像 frontend themes/*.css，
 * 2026-10-27 实测值）；白字用 DesignConstants.COLOR.WHITE（「图片/遮罩上恒白」已登记例外）。
 * 禁止在卡片里硬编码 rgba。
 */
import { DesignConstants } from '../../constants/DesignConstants';
import { AppThemeManager } from '../../core/theme/AppThemeManager';
import { DEFAULT_RESOLVED_THEME_ID, KEY_RESOLVED_THEME_ID } from '../../core/theme/AppThemeTypes';
import { AppIcon } from '../SystemIcon';
import { LucideIcon } from '../LucideIcon';

const TRANSPARENT: string = '#00000000';

@Component
export struct CardScrimTop {
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = DEFAULT_RESOLVED_THEME_ID;

  build() {
    Column()
      .width('100%')
      .height(DesignConstants.LAYOUT.MEDIA_SCRIM_TOP_HEIGHT)
      // 三段（R2 复审修复，提升顶部作者行白字对比度）：0% 0.86 → 40% 0.64 → 100% 透明
      .linearGradient({
        direction: GradientDirection.Bottom,
        colors: [
          [AppThemeManager.getPaletteById(this.resolvedThemeId).cardScrimTop, 0],
          [AppThemeManager.getPaletteById(this.resolvedThemeId).cardScrimTopMid, 0.40],
          [TRANSPARENT, 1]
        ]
      })
      .hitTestBehavior(HitTestMode.None)
      .accessibilityLevel('no')
  }
}

@Component
export struct CardScrimBottom {
  /** 高度比例：feed 58%（BRIEF）；附近图卡 62%（对齐 web NearbyImageCard h-[62%]） */
  @Prop heightRatio: Length = '58%';
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = DEFAULT_RESOLVED_THEME_ID;

  build() {
    Column()
      .width('100%')
      .height(this.heightRatio)
      // bottom → top：底部最强，42% 处次强，78% 处透明（与 web 同名 token 一致）
      .linearGradient({
        direction: GradientDirection.Top,
        // web 四段：0/30/60/90%（light 0.94/0.86/0.64/透明，dark 0.96/0.88/0.68/透明）
        colors: [
          [AppThemeManager.getPaletteById(this.resolvedThemeId).cardScrimBottomStrong, 0],
          [AppThemeManager.getPaletteById(this.resolvedThemeId).cardScrimBottomMid, 0.30],
          [AppThemeManager.getPaletteById(this.resolvedThemeId).cardScrimBottomSoft, 0.60],
          [TRANSPARENT, 0.90]
        ]
      })
      .hitTestBehavior(HitTestMode.None)
      .accessibilityLevel('no')
  }
}

@Component
export struct CardFallbackBackground {
  /** 居中主题水印图标（对齐 fe 终稿：opacity ≤0.10，主题色，不抢文字） */
  @Prop icon: AppIcon = AppIcon.PHOTO;
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = DEFAULT_RESOLVED_THEME_ID;

  build() {
    Stack() {
      Column()
        .width('100%')
        .height('100%')
        .linearGradient({
          angle: 135,
          colors: [
            [AppThemeManager.getPaletteById(this.resolvedThemeId).cardFallbackStart, 0],
            [AppThemeManager.getPaletteById(this.resolvedThemeId).cardFallbackMid, 0.55],
            [AppThemeManager.getPaletteById(this.resolvedThemeId).surfaceBackground, 1]
          ]
        })

      LucideIcon({
        icon: this.icon,
        iconSize: DesignConstants.COMPONENT.EMPTY_ICON_SIZE,
        color: AppThemeManager.getPaletteById(this.resolvedThemeId).accentBackground
      })
      .opacity(0.08)
    }
    .width('100%')
    .height('100%')
  }
}

/** 图片上徽章：深色强遮罩胶囊（onMedia）/ 主题淡底（fallback）；白字或主题文字色 */
@Component
export struct MediaOverlayBadge {
  @Prop text: string = '';
  @Prop icon: AppIcon = AppIcon.LOCATION;
  @Prop showIcon: boolean = true;
  @Prop shape: string = 'pill';
  /** true = 图片上（白字 + 强遮罩）；false = fallback 主题底（主题字 + 淡底） */
  @Prop onMedia: boolean = true;
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = DEFAULT_RESOLVED_THEME_ID;

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  private foreground(): ResourceColor {
    return this.onMedia ? DesignConstants.COLOR.WHITE : this.getPalette().textSecondary;
  }

  build() {
    Row({ space: DesignConstants.SPACING.HALF }) {
      if (this.showIcon) {
        LucideIcon({
          icon: this.icon,
          iconSize: DesignConstants.COMPONENT.ICON_SIZE_2XS,
          color: this.foreground()
        })
      }
      if (this.text.length > 0) {
        Text(this.text)
          .fontSize(DesignConstants.FONT.SIZE.XS)
          .lineHeight(DesignConstants.FONT.LINE_HEIGHT.XS)
          .fontColor(this.foreground())
          .maxLines(1)
      }
    }
    .padding({
      left: this.shape === 'pill' ? DesignConstants.SPACING.SM : 0,
      right: this.shape === 'pill' ? DesignConstants.SPACING.SM : 0,
      top: DesignConstants.SPACING.XS,
      bottom: DesignConstants.SPACING.XS
    })
    .constraintSize({
      minWidth: this.shape === 'circle' ? DesignConstants.LAYOUT.ICON_BUTTON_SIZE : undefined,
      minHeight: this.shape === 'circle' ? DesignConstants.LAYOUT.ICON_BUTTON_SIZE : undefined
    })
    .justifyContent(FlexAlign.Center)
    .alignItems(VerticalAlign.Center)
    .borderRadius(DesignConstants.RADIUS.FULL)
    .backgroundColor(this.onMedia ? DesignConstants.COLOR.MEDIA_SCRIM_STRONG : this.getPalette().surfaceSubtle)
  }
}

/** 图片上操作键：图标 + 文字；热区 44（responseRegion 不占布局） */
@Component
export struct MediaActionButton {
  @Prop icon: AppIcon = AppIcon.HEART;
  @Prop label: string = '';
  @Prop active: boolean = false;
  /** 选中态颜色（点赞=error / 收藏=accent），空则仅图标形状变化 */
  @Prop activeColor: string = '';
  /** true = 图片上（白图标+白字）；false = fallback（主题文字色） */
  @Prop onMedia: boolean = true;
  @Prop accessibilityLabel: string = '';
  onPress: () => void = () => {};

  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = DEFAULT_RESOLVED_THEME_ID;

  /** 图标色：选中态用 activeColor 填充（点赞红/收藏主色），否则白/主题色 */
  private iconForeground(): string {
    if (this.active && this.activeColor !== '') {
      return this.activeColor;
    }
    return this.onMedia ? DesignConstants.COLOR.WHITE :
      AppThemeManager.getPaletteById(this.resolvedThemeId).textSecondary;
  }

  /** 文字/计数色：压图恒白（web text-content-inverse），fallback 主题色；不随选中态变色 */
  private labelForeground(): string {
    return this.onMedia ? DesignConstants.COLOR.WHITE :
      AppThemeManager.getPaletteById(this.resolvedThemeId).textSecondary;
  }

  build() {
    Row({ space: DesignConstants.SPACING.XS }) {
      LucideIcon({
        icon: this.icon,
        // web：icon size-4.5=18、count text-sm=12；选中只换图标填充色，文字保持白/主题色
        iconSize: DesignConstants.COMPONENT.ICON_SIZE_MEDIA_ACTION,
        color: this.iconForeground()
      })
      if (this.label.length > 0) {
        Text(this.label)
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
          .fontColor(this.labelForeground())
          .opacity(this.onMedia ? DesignConstants.OPACITY.SUBTLE : 1)
      }
    }
    .height(DesignConstants.LAYOUT.MIN_HIT_TARGET)
    .alignItems(VerticalAlign.Center)
    .accessibilityText(this.accessibilityLabel)
    .onClick(() => {
      this.onPress();
    })
  }
}
