import { DesignConstants } from '../constants/DesignConstants';
import { UserInfo } from '../core/types/api';
import { UserTypeUtil } from '../core/utils/UserTypeUtil';

/**
 * SystemAvatar - Premium Avatar Component
 * Supports initials fallback, online status, and Apple-style sizing
 */
@Component
export struct SystemAvatar {
  @Prop src: string | Resource | undefined;
  @Prop avatarUrl: string | undefined;
  @Prop name: string = '';
  @Prop avatarSize: number = 40; // Default size in vp
  @Prop isOnline: boolean = false;
  @Prop showBorder: boolean = false;
  @Prop shape: 'circle' | 'squircle' | 'square' = 'circle';
  @Prop userInfo: UserInfo | undefined;

  private getBorderOptions(): BorderOptions | undefined {
    if (!this.showBorder) {
      return undefined;
    }
    const borderOptions: BorderOptions = {
      width: 2,
      color: DesignConstants.COLOR.SECONDARY_BG_LIGHT
    };
    return borderOptions;
  }

  build() {
    const actualSrc: string | Resource | undefined = this.avatarUrl ?? this.src;
    const actualName: string = this.name;

    Column() {
      Stack({ alignContent: Alignment.BottomEnd }) {
        // Main Avatar Image
        if (actualSrc) {
          Image(actualSrc)
            .width(this.avatarSize)
            .height(this.avatarSize)
            .objectFit(ImageFit.Cover)
            .borderRadius(this.getBorderRadius())
            .border(this.getBorderOptions())
            .backgroundColor(DesignConstants.COLOR.TERTIARY_BG_LIGHT)
        } else {
          // Initials Fallback
          Flex({ justifyContent: FlexAlign.Center, alignItems: ItemAlign.Center }) {
            Text(this.getInitials(actualName))
              .fontSize(this.avatarSize * 0.4)
              .fontWeight(DesignConstants.FONT.WEIGHT.SEMIBOLD)
              .fontColor(Color.White)
              .width(this.avatarSize)
              .height(this.avatarSize)
              .backgroundColor(this.generateAvatarColor(actualName))
              .borderRadius(this.getBorderRadius())
              .border(this.getBorderOptions())
          }
        }

        // Online Status Indicator
        if (this.isOnline) {
          Row()
            .width(this.avatarSize * 0.28)
            .height(this.avatarSize * 0.28)
            .backgroundColor(DesignConstants.COLOR.ONLINE_STATUS)
            .borderRadius(DesignConstants.RADIUS.FULL)
            .border(this.getBorderOptions())
            .margin({ bottom: -1, right: -1 })
        } else if (this.userInfo && UserTypeUtil.isTempUser(this.userInfo.user_type)) {
          Text('临时')
            .fontSize(this.avatarSize * 0.25) // Slightly larger than dot
            .fontColor(DesignConstants.COLOR.TEXT_SECONDARY_LIGHT)
            .backgroundColor(DesignConstants.COLOR.TERTIARY_BG_LIGHT)
            .borderRadius(4)
            .padding({ left: 2, right: 2 })
            .margin({ bottom: -2, right: -2 })
        } else if (this.userInfo && UserTypeUtil.isSystemUser(this.userInfo.user_type)) {
          Text('官方')
            .fontSize(this.avatarSize * 0.25)
            .fontColor(Color.White)
            .backgroundColor(DesignConstants.COLOR.BRAND_PRIMARY)
            .borderRadius(4)
            .padding({ left: 2, right: 2 })
            .margin({ bottom: -2, right: -2 })
        }
      }
      .width(this.avatarSize)
      .height(this.avatarSize)
    }
  }

  private getBorderRadius(): Length {
    if (this.shape === 'circle') return DesignConstants.RADIUS.FULL;
    if (this.shape === 'squircle') return (this.avatarSize * 0.22) + 'vp'; // Apple-like squircle approx
    return DesignConstants.RADIUS.SM;
  }

  private getInitials(name: string): string {
    if (!name) return '?';
    return name.substring(0, 2).toUpperCase();
  }

  private generateAvatarColor(name: string): string {
    if (!name) return DesignConstants.COLOR.TEXT_TERTIARY_LIGHT;
    
    const colors = [
      DesignConstants.COLOR.BRAND_PRIMARY,
      DesignConstants.COLOR.INFO,
      DesignConstants.COLOR.SUCCESS,
      DesignConstants.COLOR.WARNING,
      DesignConstants.COLOR.ERROR
    ];
    
    let hash = 0;
    for (let i = 0; i < name.length; i++) {
      hash = name.charCodeAt(i) + ((hash << 5) - hash);
    }
    
    const index = Math.abs(hash % colors.length);
    return colors[index];
  }
}
