import { DesignConstants } from '../constants/DesignConstants';
import { UserInfo } from '../core/types/api';
import { UserTypeUtil } from '../core/utils/UserTypeUtil';
import { MediaUrlUtil } from '../core/utils/MediaUrlUtil';

/**
 * SystemAvatar - 头像组件
 * 支持图片、首字母、在线状态、用户类型标识
 */
@Reusable
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
  @Prop userType: string = '';
  @Prop inset: boolean = false;
  @State private imageLoadFailed: boolean = false;

  private getBorderOptions(): BorderOptions | undefined {
    if (!this.showBorder) {
      return undefined;
    }
    const borderOptions: BorderOptions = {
      width: 2,
      color: $r('app.color.bg_primary')
    };
    return borderOptions;
  }

  private getActualSrc(): string | Resource | undefined {
    const source = this.avatarUrl ?? this.src;
    if (typeof source === 'string') {
      return MediaUrlUtil.normalize(source);
    }
    return source;
  }

  @Builder
  private buildAvatarContent() {
    Stack({ alignContent: Alignment.BottomEnd }) {
      if (this.getActualSrc() && !this.imageLoadFailed) {
        Image(this.getActualSrc()!)
          .width(this.avatarSize)
          .height(this.avatarSize)
          .objectFit(ImageFit.Cover)
          .borderRadius(this.getBorderRadius())
          .border(this.getBorderOptions())
          .backgroundColor($r('app.color.bg_secondary'))
          .onError(() => {
            this.imageLoadFailed = true;
          })
      } else {
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

      if (this.isOnline) {
        Row()
          .width(this.avatarSize * 0.28)
          .height(this.avatarSize * 0.28)
          .backgroundColor($r('app.color.status_online'))
          .borderRadius(DesignConstants.RADIUS.FULL)
          .border(this.getBorderOptions())
          .margin({ bottom: -1, right: -1 })
      } else if (this.userType && UserTypeUtil.isTempUser(this.userType)) {
        Text('临时')
          .fontSize(this.avatarSize * 0.25)
          .fontColor($r('sys.color.ohos_id_color_text_secondary'))
          .backgroundColor($r('app.color.bg_tertiary'))
          .borderRadius(DesignConstants.RADIUS.SM)
          .padding({ left: 2, right: 2 })
          .margin({ bottom: -2, right: -2 })
      }
    }
    .width(this.avatarSize)
    .height(this.avatarSize)
  }

  build() {
    Column() {
      this.buildAvatarContent()
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

  private generateAvatarColor(name: string): ResourceColor {
    if (!name) return $r('sys.color.ohos_id_color_text_secondary');

    const colors: ResourceColor[] = [
      $r('app.color.brand_primary'),
      $r('app.color.brand_primary'),
      $r('app.color.color_success'),
      $r('app.color.color_warning'),
      $r('app.color.color_danger')
    ];

    let hash = 0;
    for (let i = 0; i < name.length; i++) {
      hash = name.charCodeAt(i) + ((hash << 5) - hash);
    }

    const index = Math.abs(hash % colors.length);
    return colors[index];
  }
}
