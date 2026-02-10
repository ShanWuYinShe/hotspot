import { DesignConstants } from '../constants/DesignConstants';
import { UserInfo } from '../core/types/api';
import { UserTypeUtil } from '../core/utils/UserTypeUtil';

/**
 * SystemAvatar - 头像组件
 * 支持图片、首字母、在线状态、用户类型标识
 */
@Component
export struct SystemAvatar {
  @Prop src: string | Resource | undefined;
  @Prop avatarUrl: string | undefined;
  @Prop name: string = '';
  @Prop avatarSize: number = 40;
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
      color: $r('sys.color.ohos_id_color_sub_background')
    };
    return borderOptions;
  }

  private getActualSrc(): string | Resource | undefined {
    return this.avatarUrl ?? this.src;
  }

  build() {
    Column() {
      Stack({ alignContent: Alignment.BottomEnd }) {
        if (this.getActualSrc()) {
          Image(this.getActualSrc()!)
            .width(this.avatarSize)
            .height(this.avatarSize)
            .objectFit(ImageFit.Cover)
            .borderRadius(this.getBorderRadius())
            .border(this.getBorderOptions())
            .backgroundColor($r('sys.color.ohos_id_color_component_normal'))
        } else {
          // 首字母 Fallback
          Flex({ justifyContent: FlexAlign.Center, alignItems: ItemAlign.Center }) {
            Text(this.getInitials(this.name))
              .fontSize(this.avatarSize * 0.4)
              .fontWeight(DesignConstants.FONT.WEIGHT.SEMIBOLD)
              .fontColor(Color.White)
              .textAlign(TextAlign.Center)
              .width(this.avatarSize)
              .height(this.avatarSize)
              .backgroundColor(this.generateAvatarColor(this.name))
              .borderRadius(this.getBorderRadius())
              .border(this.getBorderOptions())
          }
        }

        // 在线状态
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
            .fontSize(this.avatarSize * 0.25)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
            .backgroundColor($r('sys.color.ohos_id_color_component_normal'))
            .borderRadius(4)
            .padding({ left: 2, right: 2 })
            .margin({ bottom: -2, right: -2 })
        } else if (this.userInfo && UserTypeUtil.isSystemUser(this.userInfo.user_type)) {
          Text('官方')
            .fontSize(this.avatarSize * 0.25)
            .fontColor(Color.White)
            .backgroundColor($r('sys.color.ohos_id_color_emphasize'))
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
    if (this.shape === 'squircle') return (this.avatarSize * 0.22) + 'vp';
    return DesignConstants.RADIUS.SM;
  }

  private getInitials(name: string): string {
    if (!name) return '?';
    return name.substring(0, 2).toUpperCase();
  }

  // 根据名称生成头像颜色（保持 string 类型用于 backgroundColor）
  private generateAvatarColor(name: string): string {
    if (!name) return DesignConstants.COLOR.TEXT_TERTIARY_LIGHT;

    const colors: string[] = [
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
