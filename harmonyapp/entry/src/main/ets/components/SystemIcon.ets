/**
 * 系统图标组件
 * 基于SymbolSpan的统一图标系统
 */

/**
 * 应用图标枚举
 */
export enum AppIcon {
  // 导航图标
  GROUP = 'sys.symbol.person_3_fill',
  MESSAGE = 'sys.symbol.message_fill',
  MOMENT = 'sys.symbol.camera_fill',
  MAP = 'sys.symbol.map_fill',
  PROFILE = 'sys.symbol.person_crop_circle_fill',
  
  // 操作图标
  BACK = 'sys.symbol.chevron_left',
  MORE = 'sys.symbol.ellipsis',
  SEND = 'sys.symbol.paperplane_right_fill',
  ADD = 'sys.symbol.plus',
  SEARCH = 'sys.symbol.magnifyingglass',
  CLOSE = 'sys.symbol.xmark',
  EDIT = 'sys.symbol.pencil',
  DELETE = 'sys.symbol.trash',
  SHARE = 'sys.symbol.square_and_arrow_up',
  SETTINGS = 'sys.symbol.gearshape_fill',
  
  // 功能图标
  LOCATION = 'sys.symbol.location_fill',
  CAMERA = 'sys.symbol.camera_fill',
  PHOTO = 'sys.symbol.photo_fill',
  VOICE = 'sys.symbol.mic_fill',
  VIDEO = 'sys.symbol.video_fill',
  FILE = 'sys.symbol.doc_fill',
  LINK = 'sys.symbol.link',
  
  // 状态图标
  ONLINE = 'sys.symbol.circle_fill',
  OFFLINE = 'sys.symbol.circle',
  TYPING = 'sys.symbol.ellipsis_bubble_fill',
  READ = 'sys.symbol.checkmark_circle_fill',
  UNREAD = 'sys.symbol.circle_badge_fill',
  NOTIFICATION = 'sys.symbol.bell_fill',
  MUTE = 'sys.symbol.bell_slash_fill',
  
  // 社交图标
  LIKE = 'sys.symbol.heart_fill',
  UNLIKE = 'sys.symbol.heart',
  COMMENT = 'sys.symbol.bubble_left_fill',
  MENTION = 'sys.symbol.at',
  FOLLOW = 'sys.symbol.person_badge_plus_fill',
  UNFOLLOW = 'sys.symbol.person_badge_minus_fill',
  BLOCK = 'sys.symbol.person_crop_circle_badge_xmark',
  
  // 系统图标
  WARNING = 'sys.symbol.exclamationmark_triangle_fill',
  ERROR = 'sys.symbol.xmark_circle_fill',
  SUCCESS = 'sys.symbol.checkmark_circle_fill',
  INFO = 'sys.symbol.info_circle_fill',
  LOADING = 'sys.symbol.arrow_clockwise',
  
  // 权限图标
  LOCK = 'sys.symbol.lock_fill',
  UNLOCK = 'sys.symbol.lock_open_fill',
  PRIVACY = 'sys.symbol.eye_slash_fill',
  PUBLIC = 'sys.symbol.eye_fill'
}

/**
 * 图标大小枚举
 */
export enum IconSize {
  SMALL = 16,
  MEDIUM = 20,
  LARGE = 24,
  XLARGE = 32
}

/**
 * 图标颜色枚举
 */
export enum IconColor {
  PRIMARY = '#007AFF',
  SECONDARY = '#8E8E93',
  SUCCESS = '#34C759',
  WARNING = '#FF9500',
  ERROR = '#FF3B30',
  WHITE = '#FFFFFF',
  BLACK = '#000000',
  GRAY = '#8E8E93'
}

/**
 * 系统图标组件
 */
@Component
export struct SystemIcon {
  @Prop icon: AppIcon;
  @Prop size: IconSize = IconSize.MEDIUM;
  @Prop color: string = IconColor.PRIMARY;
  @Prop clickable: boolean = false;
  onClick?: () => void;

  build() {
    SymbolSpan($r(this.icon))
      .fontSize(this.size)
      .fontColor(this.color)
      .onClick(() => {
        if (this.clickable && this.onClick) {
          this.onClick();
        }
      })
  }
}

/**
 * 带背景的图标按钮组件
 */
@Component
export struct IconButton {
  @Prop icon: AppIcon;
  @Prop size: IconSize = IconSize.MEDIUM;
  @Prop iconColor: string = IconColor.PRIMARY;
  @Prop backgroundColor: string = '#F2F2F7';
  @Prop borderRadius: number = 8;
  @Prop padding: number = 8;
  onClick?: () => void;

  build() {
    Row() {
      SystemIcon({
        icon: this.icon,
        size: this.size,
        color: this.iconColor,
        clickable: true,
        onClick: this.onClick
      })
    }
    .width(this.size + this.padding * 2)
    .height(this.size + this.padding * 2)
    .backgroundColor(this.backgroundColor)
    .borderRadius(this.borderRadius)
    .justifyContent(FlexAlign.Center)
    .alignItems(VerticalAlign.Center)
    .onClick(() => {
      if (this.onClick) {
        this.onClick();
      }
    })
  }
}

/**
 * 导航栏图标组件
 */
@Component
export struct NavIcon {
  @Prop icon: AppIcon;
  @Prop label: string;
  @Prop isActive: boolean = false;
  @Prop activeColor: string = IconColor.PRIMARY;
  @Prop inactiveColor: string = IconColor.SECONDARY;
  onClick?: () => void;

  build() {
    Column({ space: 4 }) {
      SystemIcon({
        icon: this.icon,
        size: IconSize.LARGE,
        color: this.isActive ? this.activeColor : this.inactiveColor,
        clickable: true,
        onClick: this.onClick
      })
      
      Text(this.label)
        .fontSize(12)
        .fontColor(this.isActive ? this.activeColor : this.inactiveColor)
        .fontWeight(this.isActive ? FontWeight.Medium : FontWeight.Normal)
    }
    .alignItems(HorizontalAlign.Center)
    .onClick(() => {
      if (this.onClick) {
        this.onClick();
      }
    })
  }
}

/**
 * 状态指示图标组件
 */
@Component
export struct StatusIcon {
  @Prop status: 'online' | 'offline' | 'typing' | 'read' | 'unread';
  @Prop size: IconSize = IconSize.SMALL;

  private getIconAndColor(): { icon: AppIcon, color: string } {
    switch (this.status) {
      case 'online':
        return { icon: AppIcon.ONLINE, color: IconColor.SUCCESS };
      case 'offline':
        return { icon: AppIcon.OFFLINE, color: IconColor.SECONDARY };
      case 'typing':
        return { icon: AppIcon.TYPING, color: IconColor.PRIMARY };
      case 'read':
        return { icon: AppIcon.READ, color: IconColor.SUCCESS };
      case 'unread':
        return { icon: AppIcon.UNREAD, color: IconColor.ERROR };
      default:
        return { icon: AppIcon.OFFLINE, color: IconColor.SECONDARY };
    }
  }

  build() {
    const { icon, color } = this.getIconAndColor();
    
    SystemIcon({
      icon: icon,
      size: this.size,
      color: color
    })
  }
}
