import { DesignConstants } from '../constants/DesignConstants';
import { UserInfo } from '../core/types/WebSocketTypes';
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
  @Prop size: number = 40; // Default size in vp
  @Prop isOnline: boolean = false;
  @Prop showBorder: boolean = false;
  @Prop shape: 'circle' | 'squircle' | 'square' = 'circle';
  @Prop userInfo: UserInfo | undefined;

  build() {
    const actualSrc: string | Resource | undefined = this.avatarUrl ?? this.src;
    const actualName: string = this.name;

    Stack({ alignContent: Alignment.BottomEnd }) {
      // Main Avatar Image
      if (actualSrc) {
        Image(actualSrc)
          .width(this.size)
          .height(this.size)
          .objectFit(ImageFit.Cover)
          .borderRadius(this.getBorderRadius())
          .border(this.showBorder ? { width: 2, color: DesignConstants.COLOR.SECONDARY_BG_LIGHT } : undefined)
          .backgroundColor(DesignConstants.COLOR.TERTIARY_BG_LIGHT)
      } else {
        // Initials Fallback
        Flex({ justifyContent: FlexAlign.Center, alignItems: ItemAlign.Center }) {
          Text(this.getInitials(actualName))
            .fontSize(this.size * 0.4)
            .fontWeight(DesignConstants.FONT.WEIGHT.SEMIBOLD)
            .fontColor(Color.White)
            .width(this.size)
            .height(this.size)
            .backgroundColor(this.generateAvatarColor(actualName))
            .borderRadius(this.getBorderRadius())
            .border(this.showBorder ? { width: 2, color: DesignConstants.COLOR.SECONDARY_BG_LIGHT } : undefined)
        }
      }

      // Online Status Indicator
      if (this.isOnline) {
        Row()
          .width(this.size * 0.28)
          .height(this.size * 0.28)
          .position({ x: '85%', y: '75%' })
          .backgroundColor(DesignConstants.COLOR.ONLINE_STATUS)
          .borderRadius(this.size * 0.5)
      }
    }
        .width(this.size)
        .height(this.size)
        .backgroundColor(this.generateAvatarColor(this.name))
        .borderRadius(this.getBorderRadius())
        .border(this.showBorder ? { width: 2, color: DesignConstants.COLOR.SECONDARY_BG_LIGHT } : undefined)
      }

      // Online Status Indicator
      if (this.isOnline) {
        Row()
          .width(this.size * 0.28)
          .height(this.size * 0.28)
          .backgroundColor(DesignConstants.COLOR.ONLINE_STATUS)
          .borderRadius(DesignConstants.RADIUS.FULL)
          .border({ width: 2, color: DesignConstants.COLOR.SECONDARY_BG_LIGHT })
          .margin({ bottom: -1, right: -1 })
      } else if (this.userInfo && UserTypeUtil.isTempUser(this.userInfo.user_type)) {
        Text('临时')
          .fontSize(this.size * 0.25) // Slightly larger than dot
          .fontColor(DesignConstants.COLOR.TEXT_SECONDARY_LIGHT)
          .backgroundColor(DesignConstants.COLOR.TERTIARY_BG_LIGHT)
          .borderRadius(4)
          .padding({ left: 2, right: 2 })
          .margin({ bottom: -2, right: -2 })
      } else if (this.userInfo && UserTypeUtil.isSystemUser(this.userInfo.user_type)) {
        Text('官方')
          .fontSize(this.size * 0.25)
          .fontColor(Color.White)
          .backgroundColor(DesignConstants.COLOR.BRAND_PRIMARY)
          .borderRadius(4)
          .padding({ left: 2, right: 2 })
          .margin({ bottom: -2, right: -2 })
      }
    }
    .width(this.size)
    .height(this.size)
  }

  private getBorderRadius(): Length {
    if (this.shape === 'circle') return DesignConstants.RADIUS.FULL;
    if (this.shape === 'squircle') return (this.size * 0.22) + 'vp'; // Apple-like squircle approx
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
