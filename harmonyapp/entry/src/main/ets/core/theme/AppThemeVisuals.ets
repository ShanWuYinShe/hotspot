import { DesignConstants } from '../../constants/DesignConstants';
import { AppThemeManager } from './AppThemeManager';

export interface ThemeCardVisualStyle {
  backgroundColor: ResourceColor;
  borderColor: ResourceColor;
  borderWidth: number;
  shadow: ShadowOptions;
  borderRadius: number;
  padding: number;
  minHeight?: number;
}

export interface ThemeButtonVisualStyle {
  backgroundColor: ResourceColor;
  textColor: ResourceColor;
  borderColor: ResourceColor;
  borderWidth: number;
  shadow: ShadowOptions;
  borderRadius: number;
  height: number;
  horizontalPadding: number;
}

export interface ThemeInputVisualStyle {
  backgroundColor: ResourceColor;
  borderColor: ResourceColor;
  borderWidth: number;
  borderRadius: number;
  shadow: ShadowOptions;
  height: number;
  horizontalPadding: number;
}

/** 输入框边框态解析结果（focus / error 态覆盖 borderColor + shadow） */
export interface InputBorderResolution {
  borderColor: ResourceColor;
  shadow: ShadowOptions;
}

export interface ThemeMetricVisualStyle extends ThemeCardVisualStyle {
  iconColor: ResourceColor;
  valueFontSize: number;
}

export interface ThemeProfileIdentityStyle {
  avatarSize: number;
  titleFontSize: number;
  subtitleFontSize: number;
  avatarShadow: ShadowOptions;
  haloShadow: ShadowOptions;
}

export interface ThemeProfileQuickActionStyle extends ThemeCardVisualStyle {
  iconSize: number;
  iconWrapperSize: number;
  labelFontSize: number;
}

export interface ThemeProfileIntentStyle extends ThemeCardVisualStyle {
  iconWrapperSize: number;
  titleFontSize: number;
  descriptionFontSize: number;
  minHeight: number;
}

export interface ThemeProfileVisualStyle {
  card: ThemeCardVisualStyle;
  metricCard: ThemeMetricVisualStyle;
  primaryButton: ThemeButtonVisualStyle;
  secondaryButton: ThemeButtonVisualStyle;
  input: ThemeInputVisualStyle;
  identity: ThemeProfileIdentityStyle;
  quickAction: ThemeProfileQuickActionStyle;
  intentCard: ThemeProfileIntentStyle;
}

export interface ThemeTabBarVisualStyle {
  backgroundColor: ResourceColor;
  activeBackgroundColor: ResourceColor;
  borderColor: ResourceColor;
  borderWidth: number;
  shadow: ShadowOptions;
  activeGlow: ShadowOptions;
  fabGlow: ShadowOptions;
  height: number;
  radius: number;
  floatingMargin: number;
  createButtonWidth: number;
  createButtonHeight: number;
  indicatorWidth: number;
  indicatorHeight: number;
  indicatorBorderRadius: number;
  itemHeight: number;
}

export interface ThemePreviewVisualStyle {
  card: ThemeCardVisualStyle;
  primaryButton: ThemeButtonVisualStyle;
  tileRadius: number;
  heroAvatarSize: number;
  haloShadow: ShadowOptions;
}

export interface ThemeChatVisualStyle {
  listRadius: number;
  composerRadius: number;
  input: ThemeInputVisualStyle;
  sendButton: ThemeButtonVisualStyle;
}

export class AppThemeVisuals {
  private static surfaceCardCache: Map<string, ThemeCardVisualStyle> = new Map();
  private static elevatedCardCache: Map<string, ThemeCardVisualStyle> = new Map();
  private static compactCardCache: Map<string, ThemeCardVisualStyle> = new Map();
  private static listItemCardCache: Map<string, ThemeCardVisualStyle> = new Map();
  private static primaryButtonCache: Map<string, ThemeButtonVisualStyle> = new Map();
  private static secondaryButtonCache: Map<string, ThemeButtonVisualStyle> = new Map();
  private static inputCache: Map<string, ThemeInputVisualStyle> = new Map();
  private static profileMetricCache: Map<string, ThemeMetricVisualStyle> = new Map();
  private static profileIdentityCache: Map<string, ThemeProfileIdentityStyle> = new Map();
  private static profileVisualCache: Map<string, ThemeProfileVisualStyle> = new Map();
  private static tabBarCache: Map<string, ThemeTabBarVisualStyle> = new Map();
  private static previewCache: Map<string, ThemePreviewVisualStyle> = new Map();
  private static chatVisualCache: Map<string, ThemeChatVisualStyle> = new Map();

  private static getPalette(themeId: string) {
    return AppThemeManager.getPalette(themeId);
  }

  private static getThemeKey(themeId: string): string {
    return AppThemeVisuals.getPalette(themeId).id;
  }

  static getSurfaceShadow(themeId: string): ShadowOptions {
    return AppThemeVisuals.getPalette(themeId).surfaceShadow;
  }

  static getElevatedShadow(themeId: string): ShadowOptions {
    return AppThemeVisuals.getPalette(themeId).elevatedShadow;
  }

  static getAccentShadow(themeId: string): ShadowOptions {
    return AppThemeVisuals.getPalette(themeId).accentShadow;
  }

  static getControlShadow(themeId: string): ShadowOptions {
    return AppThemeVisuals.getPalette(themeId).controlShadow;
  }

  static getTabBarShadow(themeId: string): ShadowOptions {
    return AppThemeVisuals.getPalette(themeId).tabBarShadow;
  }

  static getOverlayStrong(themeId: string): string {
    return AppThemeVisuals.getPalette(themeId).overlayStrong;
  }

  static getSurfaceCardStyle(themeId: string): ThemeCardVisualStyle {
    const key = AppThemeVisuals.getThemeKey(themeId);
    const cached = AppThemeVisuals.surfaceCardCache.get(key);
    if (cached) {
      return cached;
    }
    const palette = AppThemeVisuals.getPalette(themeId);
    const style: ThemeCardVisualStyle = {
      backgroundColor: palette.surfaceBackground,
      borderColor: palette.borderColor,
      borderWidth: 1,
      shadow: palette.elevatedShadow,
      borderRadius: palette.visuals.shape.surfaceCard,
      padding: palette.visuals.metrics.cardPadding
    };
    AppThemeVisuals.surfaceCardCache.set(key, style);
    return style;
  }

  static getElevatedCardStyle(themeId: string): ThemeCardVisualStyle {
    const key = AppThemeVisuals.getThemeKey(themeId);
    const cached = AppThemeVisuals.elevatedCardCache.get(key);
    if (cached) {
      return cached;
    }
    const palette = AppThemeVisuals.getPalette(themeId);
    const style: ThemeCardVisualStyle = {
      backgroundColor: palette.surfaceElevated,
      borderColor: palette.floatingBorderColor,
      borderWidth: 1,
      shadow: palette.elevatedShadow,
      borderRadius: palette.visuals.shape.heroCard,
      padding: palette.visuals.metrics.heroPadding
    };
    AppThemeVisuals.elevatedCardCache.set(key, style);
    return style;
  }

  static getCompactCardStyle(themeId: string): ThemeCardVisualStyle {
    const key = AppThemeVisuals.getThemeKey(themeId);
    const cached = AppThemeVisuals.compactCardCache.get(key);
    if (cached) {
      return cached;
    }
    const palette = AppThemeVisuals.getPalette(themeId);
    const style: ThemeCardVisualStyle = {
      backgroundColor: palette.surfaceBackground,
      borderColor: palette.borderColor,
      borderWidth: 1,
      shadow: palette.elevatedShadow,
      borderRadius: palette.visuals.shape.compactCard,
      padding: palette.visuals.metrics.cardPadding
    };
    AppThemeVisuals.compactCardCache.set(key, style);
    return style;
  }

  static getListItemCardStyle(themeId: string): ThemeCardVisualStyle {
    const key = AppThemeVisuals.getThemeKey(themeId);
    const cached = AppThemeVisuals.listItemCardCache.get(key);
    if (cached) {
      return cached;
    }
    const palette = AppThemeVisuals.getPalette(themeId);
    const style: ThemeCardVisualStyle = {
      backgroundColor: palette.surfaceBackground,
      borderColor: palette.borderColor,
      borderWidth: 1,
      shadow: palette.elevatedShadow,
      borderRadius: palette.visuals.shape.listItem,
      padding: palette.visuals.metrics.cardPadding,
      minHeight: palette.visuals.metrics.listItemMinHeight
    };
    AppThemeVisuals.listItemCardCache.set(key, style);
    return style;
  }

  static getSecondaryButtonStyle(themeId: string): ThemeButtonVisualStyle {
    const key = AppThemeVisuals.getThemeKey(themeId);
    const cached = AppThemeVisuals.secondaryButtonCache.get(key);
    if (cached) {
      return cached;
    }
    const palette = AppThemeVisuals.getPalette(themeId);
    const style: ThemeButtonVisualStyle = {
      backgroundColor: palette.buttonSecondaryBackground,
      textColor: palette.buttonSecondaryText,
      borderColor: palette.floatingBorderColor,
      borderWidth: 1,
      shadow: palette.visuals.effects.previewShadow,
      borderRadius: palette.visuals.shape.button,
      height: palette.visuals.metrics.actionButtonHeight,
      horizontalPadding: palette.visuals.metrics.secondaryButtonHorizontalPadding
    };
    AppThemeVisuals.secondaryButtonCache.set(key, style);
    return style;
  }

  static getPrimaryButtonStyle(themeId: string): ThemeButtonVisualStyle {
    const key = AppThemeVisuals.getThemeKey(themeId);
    const cached = AppThemeVisuals.primaryButtonCache.get(key);
    if (cached) {
      return cached;
    }
    const palette = AppThemeVisuals.getPalette(themeId);
    const style: ThemeButtonVisualStyle = {
      backgroundColor: palette.accentBackground,
      textColor: palette.accentText,
      borderColor: palette.primaryButtonBorderColor,
      borderWidth: 1,
      shadow: palette.accentShadow,
      borderRadius: palette.visuals.shape.button,
      height: palette.visuals.metrics.profilePrimaryButtonHeight,
      horizontalPadding: palette.visuals.metrics.profilePrimaryButtonHorizontalPadding + DesignConstants.SPACING.XS
    };
    AppThemeVisuals.primaryButtonCache.set(key, style);
    return style;
  }

  static getInputStyle(themeId: string): ThemeInputVisualStyle {
    const key = AppThemeVisuals.getThemeKey(themeId);
    const cached = AppThemeVisuals.inputCache.get(key);
    if (cached) {
      return cached;
    }
    const palette = AppThemeVisuals.getPalette(themeId);
    const style: ThemeInputVisualStyle = {
      backgroundColor: palette.inputBackgroundColor,
      borderColor: palette.borderColor,
      borderWidth: 1,
      borderRadius: palette.visuals.shape.input,
      shadow: palette.controlShadow,
      height: DesignConstants.COMPONENT.INPUT_HEIGHT,
      horizontalPadding: DesignConstants.SPACING.LG
    };
    AppThemeVisuals.inputCache.set(key, style);
    return style;
  }

  static getProfileMetricCardStyle(themeId: string): ThemeMetricVisualStyle {
    const key = AppThemeVisuals.getThemeKey(themeId);
    const cached = AppThemeVisuals.profileMetricCache.get(key);
    if (cached) {
      return cached;
    }
    const palette = AppThemeVisuals.getPalette(themeId);
    const baseStyle = AppThemeVisuals.getSurfaceCardStyle(themeId);
    const style: ThemeMetricVisualStyle = {
      backgroundColor: baseStyle.backgroundColor,
      borderColor: baseStyle.borderColor,
      borderWidth: baseStyle.borderWidth,
      shadow: palette.elevatedShadow,
      borderRadius: palette.visuals.shape.heroCard,
      padding: palette.visuals.metrics.metricCardPadding,
      minHeight: palette.visuals.metrics.profileMetricMinHeight,
      iconColor: palette.accentBackground,
      valueFontSize: palette.visuals.metrics.profileMetricValueFontSize
    };
    AppThemeVisuals.profileMetricCache.set(key, style);
    return style;
  }

  static getProfileIdentityStyle(themeId: string): ThemeProfileIdentityStyle {
    const key = AppThemeVisuals.getThemeKey(themeId);
    const cached = AppThemeVisuals.profileIdentityCache.get(key);
    if (cached) {
      return cached;
    }
    const palette = AppThemeVisuals.getPalette(themeId);
    const style: ThemeProfileIdentityStyle = {
      avatarSize: palette.visuals.metrics.profileAvatarSize,
      titleFontSize: palette.visuals.metrics.profileTitleFontSize,
      subtitleFontSize: palette.visuals.metrics.profileSubtitleFontSize,
      avatarShadow: palette.visuals.effects.avatarShadow,
      haloShadow: palette.visuals.effects.haloShadow
    };
    AppThemeVisuals.profileIdentityCache.set(key, style);
    return style;
  }

  static getProfileVisualStyle(themeId: string): ThemeProfileVisualStyle {
    const key = AppThemeVisuals.getThemeKey(themeId);
    const cached = AppThemeVisuals.profileVisualCache.get(key);
    if (cached) {
      return cached;
    }
    const palette = AppThemeVisuals.getPalette(themeId);
    const style: ThemeProfileVisualStyle = {
      card: AppThemeVisuals.getElevatedCardStyle(themeId),
      metricCard: AppThemeVisuals.getProfileMetricCardStyle(themeId),
      primaryButton: AppThemeVisuals.getPrimaryButtonStyle(themeId),
      secondaryButton: AppThemeVisuals.getSecondaryButtonStyle(themeId),
      input: AppThemeVisuals.getInputStyle(themeId),
      identity: AppThemeVisuals.getProfileIdentityStyle(themeId),
      quickAction: {
        backgroundColor: palette.quickActionBackgroundColor,
        borderColor: palette.floatingBorderColor,
        borderWidth: 1,
        shadow: palette.elevatedShadow,
        borderRadius: 18,
        padding: DesignConstants.SPACING.MD,
        iconSize: 22,
        iconWrapperSize: 56,
        labelFontSize: DesignConstants.FONT.SIZE.SM
      },
      intentCard: {
        backgroundColor: palette.surfaceElevated,
        borderColor: palette.floatingBorderColor,
        borderWidth: 1,
        shadow: palette.elevatedShadow,
        borderRadius: palette.visuals.shape.heroCard,
        padding: palette.visuals.metrics.cardPadding,
        iconWrapperSize: 40,
        titleFontSize: DesignConstants.FONT.SIZE.XL,
        descriptionFontSize: DesignConstants.FONT.SIZE.MD,
        minHeight: 120
      }
    };
    AppThemeVisuals.profileVisualCache.set(key, style);
    return style;
  }

  static getTabBarVisualStyle(themeId: string): ThemeTabBarVisualStyle {
    const key = AppThemeVisuals.getThemeKey(themeId);
    const cached = AppThemeVisuals.tabBarCache.get(key);
    if (cached) {
      return cached;
    }
    const palette = AppThemeVisuals.getPalette(themeId);
    const style: ThemeTabBarVisualStyle = {
      backgroundColor: palette.tabBarBackground,
      activeBackgroundColor: palette.tabBarActiveBackground,
      borderColor: palette.floatingBorderColor,
      borderWidth: 1,
      shadow: palette.tabBarShadow,
      activeGlow: palette.visuals.effects.tabActiveGlow,
      fabGlow: palette.accentShadow,
      height: palette.visuals.metrics.tabBarHeight,
      radius: palette.visuals.shape.tabBar,
      floatingMargin: palette.visuals.metrics.tabBarFloatingMargin,
      createButtonWidth: palette.visuals.metrics.createButtonWidth,
      createButtonHeight: palette.visuals.metrics.createButtonHeight,
      indicatorWidth: 22,
      indicatorHeight: 2.5,
      indicatorBorderRadius: 2,
      itemHeight: palette.visuals.metrics.tabBarHeight
    };
    AppThemeVisuals.tabBarCache.set(key, style);
    return style;
  }

  static getThemePreviewStyle(themeId: string): ThemePreviewVisualStyle {
    const key = AppThemeVisuals.getThemeKey(themeId);
    const cached = AppThemeVisuals.previewCache.get(key);
    if (cached) {
      return cached;
    }
    const palette = AppThemeVisuals.getPalette(themeId);
    const style: ThemePreviewVisualStyle = {
      card: {
        backgroundColor: palette.surfaceElevated,
        borderColor: palette.floatingBorderColor,
        borderWidth: 1,
        shadow: palette.visuals.effects.previewShadow,
        borderRadius: palette.visuals.shape.previewCard,
        padding: palette.visuals.metrics.heroPadding
      },
      primaryButton: {
        backgroundColor: palette.accentBackground,
        textColor: palette.accentText,
        borderColor: palette.borderColor,
        borderWidth: 1,
        shadow: palette.accentShadow,
        borderRadius: palette.visuals.shape.button,
        height: palette.visuals.metrics.previewPrimaryButtonHeight,
        horizontalPadding: DesignConstants.SPACING.LG
      },
      tileRadius: palette.visuals.shape.compactCard,
      heroAvatarSize: palette.visuals.metrics.previewHeroAvatarSize,
      haloShadow: palette.visuals.effects.haloShadow
    };
    AppThemeVisuals.previewCache.set(key, style);
    return style;
  }

  static getAvatarShadow(themeId: string): ShadowOptions {
    return AppThemeVisuals.getPalette(themeId).visuals.effects.avatarShadow;
  }

  static getAvatarHaloShadow(themeId: string): ShadowOptions {
    return AppThemeVisuals.getPalette(themeId).visuals.effects.haloShadow;
  }

  static getNeuInsetShadow(themeId: string): ShadowOptions {
    return AppThemeVisuals.getPalette(themeId).visuals.effects.insetShadow;
  }

  static getNeuHighlightShadow(themeId: string): ShadowOptions {
    return AppThemeVisuals.getPalette(themeId).visuals.effects.highlightShadow;
  }

  static getTabActiveGlow(themeId: string): ShadowOptions {
    return AppThemeVisuals.getPalette(themeId).visuals.effects.tabActiveGlow;
  }

  static getChatVisualStyle(themeId: string): ThemeChatVisualStyle {
    const key = AppThemeVisuals.getThemeKey(themeId);
    const cached = AppThemeVisuals.chatVisualCache.get(key);
    if (cached) {
      return cached;
    }
    const palette = AppThemeVisuals.getPalette(themeId);
    const style: ThemeChatVisualStyle = {
      listRadius: palette.visuals.metrics.chatListRadius,
      composerRadius: palette.visuals.shape.heroCard,
      input: {
        backgroundColor: palette.surfaceSubtle,
        borderColor: palette.borderColor,
        borderWidth: 1,
        borderRadius: palette.visuals.shape.input,
        shadow: palette.controlShadow,
        height: DesignConstants.CHAT.INPUT_HEIGHT,
        horizontalPadding: DesignConstants.CHAT.BUBBLE_PADDING_H
      },
      sendButton: {
        backgroundColor: palette.accentBackground,
        textColor: palette.accentText,
        borderColor: palette.borderColor,
        borderWidth: 1,
        shadow: palette.accentShadow,
        borderRadius: palette.visuals.shape.button,
        height: palette.visuals.metrics.chatSendButtonHeight,
        horizontalPadding: DesignConstants.SPACING.LG
      }
    };
    AppThemeVisuals.chatVisualCache.set(key, style);
    return style;
  }

  /**
   * 输入框边框态解析，对齐前端 focus（borderColor=brand + glow）/ error（红边 + 红色软发光）。
   * ArkUI 的 ShadowOptions 没有 spread 参数，无法表达 CSS `0 0 0 2px` 硬边环，
   * 这里降级为「提色边框 + 软发光阴影」的组合，是最接近前端 focus 观感的写法。
   */
  static resolveInputBorder(base: ThemeInputVisualStyle, focused: boolean, error: boolean, themeId: string): InputBorderResolution {
    const palette = AppThemeVisuals.getPalette(themeId);
    if (error) {
      return {
        borderColor: palette.error,
        shadow: { radius: 8, color: AppThemeVisuals.withAlpha(palette.error, 0.10), offsetX: 0, offsetY: 0 } as ShadowOptions
      };
    }
    if (focused) {
      return {
        borderColor: palette.accentBackground,
        shadow: { radius: 8, color: AppThemeVisuals.withAlpha(palette.accentBackground, 0.18), offsetX: 0, offsetY: 0 } as ShadowOptions
      };
    }
    return {
      borderColor: base.borderColor,
      shadow: base.shadow
    };
  }

  /** 把 #RRGGBB（6 位）拼接 alpha（0-1）→ #RRGGBBAA（8 位），用于 shadow 软发光色 */
  static withAlpha(hex: string, alpha: number): string {
    const a: number = Math.round(alpha * 255);
    const aHex: string = a < 16 ? '0' + a.toString(16).toUpperCase() : a.toString(16).toUpperCase();
    return hex + aHex;
  }
}
