import { DesignConstants } from '../constants/DesignConstants';
import { AppThemeManager } from '../core/theme/AppThemeManager';
import { KEY_RESOLVED_THEME_ID } from '../core/theme/AppThemeTypes';
import { AppIcon } from './SystemIcon';

export type SurfaceIconButtonKind = 'primary' | 'neutral';

@Component
export struct SurfaceIconButton {
  @Prop icon: AppIcon = AppIcon.BACK;
  @Prop kind: SurfaceIconButtonKind = 'neutral';
  @Prop buttonSize: number = 40;
  @Prop glyphSize: number = 20;
  @Prop disabled: boolean = false;
  onPress?: () => void;
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'default_clean_light';

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  private getBackgroundColor(): ResourceColor | Color {
    if (this.kind === 'primary') {
      return this.getPalette().accentBackground;
    }
    return this.getPalette().surfaceBackground;
  }

  private getBorderColor(): ResourceColor | Color {
    if (this.kind === 'primary') {
      return this.getPalette().accentBackground;
    }
    return this.getPalette().borderColor;
  }

  private getShadowColor(): ShadowOptions {
    return this.kind === 'primary' ? this.getPalette().accentShadow : this.getPalette().controlShadow;
  }

  private getIconColor(): ResourceColor | Color {
    if (this.kind === 'primary') {
      return this.getPalette().accentText;
    }
    return this.getPalette().textPrimary;
  }

  build() {
    Button() {
      Text() {
        SymbolSpan($r(this.icon))
          .fontSize(this.glyphSize)
          .fontColor([this.getIconColor()])
      }
    }
    .type(ButtonType.Normal)
    .stateEffect(!this.disabled)
    .width(this.buttonSize)
    .height(this.buttonSize)
    .backgroundColor(this.getBackgroundColor())
    .borderRadius(DesignConstants.RADIUS.FULL)
    .border({ width: 1, color: this.getBorderColor() })
    .shadow(this.getShadowColor())
    .enabled(!this.disabled)
    .opacity(this.disabled ? DesignConstants.OPACITY.LOADING : 1)
    .onClick(() => {
      if (this.disabled) {
        return;
      }
      this.onPress?.();
    })
  }
}
