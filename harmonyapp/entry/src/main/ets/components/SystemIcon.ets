/**
 * 系统图标组件
 * 提供统一的图标显示和交互功能
 */


/**
 * 应用图标枚举
 */
export enum AppIcon {
  // 导航图标
  BACK = 'sys.symbol.chevron_left',
  FORWARD = 'sys.symbol.chevron_right',
  UP = 'sys.symbol.chevron_up',
  DOWN = 'sys.symbol.chevron_down',
  
  // 操作图标
  ADD = 'sys.symbol.plus',
  REMOVE = 'sys.symbol.minus',
  DELETE = 'sys.symbol.trash',
  EDIT = 'sys.symbol.pencil',
  SAVE = 'sys.symbol.checkmark',
  CANCEL = 'sys.symbol.xmark',
  CLOSE = 'sys.symbol.xmark',
  
  // 功能图标
  SEARCH = 'sys.symbol.magnifyingglass',
  FILTER = 'sys.symbol.line_3_horizontal_decrease',
  SORT = 'sys.symbol.arrow_up_arrow_down',
  REFRESH = 'sys.symbol.arrow_clockwise',
  SETTINGS = 'sys.symbol.gear',
  MORE = 'sys.symbol.ellipsis',
  
  // 通信图标
  MESSAGE = 'sys.symbol.message',
  SEND = 'sys.symbol.paperplane',
  CALL = 'sys.symbol.phone',
  VIDEO = 'sys.symbol.video',
  
  // 媒体图标
  CAMERA = 'sys.symbol.camera',
  PHOTO = 'sys.symbol.photo',
  PLAY = 'sys.symbol.play',
  PAUSE = 'sys.symbol.pause',
  
  // 位置图标
  LOCATION = 'sys.symbol.location',
  MAP = 'sys.symbol.map',
  
  // 用户图标
  PERSON = 'sys.symbol.person',
  GROUP = 'sys.symbol.person_2',
  
  // 状态图标
  HEART = 'sys.symbol.heart',
  HEART_FILL = 'sys.symbol.heart_fill',
  STAR = 'sys.symbol.star',
  STAR_FILL = 'sys.symbol.star_fill',
  
  // 系统图标
  INFO = 'sys.symbol.info_circle',
  WARNING = 'sys.symbol.exclamationmark_triangle',
  ERROR = 'sys.symbol.xmark_circle',
  SUCCESS = 'sys.symbol.checkmark_circle',
  
  // 新增智能功能图标
  NOTIFICATION = 'sys.symbol.bell',
  NOTIFICATION_FILL = 'sys.symbol.bell_fill',
  CHART = 'sys.symbol.chart_bar',
  PERFORMANCE = 'sys.symbol.speedometer',
  AI = 'sys.symbol.brain',
  RECOMMEND = 'sys.symbol.lightbulb',
  SYNC = 'sys.symbol.arrow_triangle_2_circlepath',
  DISTRIBUTED = 'sys.symbol.network',
  
  // 箭头图标
  ARROW_UP = 'sys.symbol.chevron_up',
  ARROW_DOWN = 'sys.symbol.chevron_down',
  ARROW_LEFT = 'sys.symbol.chevron_left',
  ARROW_RIGHT = 'sys.symbol.chevron_right',
  
  // 操作图标补充
  LIKE = 'sys.symbol.heart',
  SHARE = 'sys.symbol.square_and_arrow_up',
  CHECK = 'sys.symbol.checkmark',
  CHECK_ALL = 'sys.symbol.checkmark_circle',
  CLEAN = 'sys.symbol.trash',
  CLEAR = 'sys.symbol.clear',
  
  // 状态图标补充
  EMPTY = 'sys.symbol.tray',
  LOADING = 'sys.symbol.arrow_clockwise',
  QUESTION = 'sys.symbol.questionmark_circle',
  SHIELD = 'sys.symbol.shield',
  
  // 系统功能图标
  MEMORY = 'sys.symbol.memorychip',
  CPU = 'sys.symbol.cpu',
  BATTERY = 'sys.symbol.battery_100',
  SPEED = 'sys.symbol.speedometer',
  CACHE = 'sys.symbol.externaldrive',
  
  // 调试和开发图标
  DEBUG = 'sys.symbol.hammer',
  CODE = 'sys.symbol.curlybraces',
  BUG = 'sys.symbol.ant',
  
  // 通用图标
  ALL = 'sys.symbol.square_grid_2x2',
  HOME = 'sys.symbol.house',
  PROFILE = 'sys.symbol.person_circle'
}

/**
 * 图标尺寸枚举
 */
export enum IconSize {
  SMALL = 16,
  MEDIUM = 24,
  LARGE = 32,
  EXTRA_LARGE = 48
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
 * 系统图标组件参数接口
 */
export interface SystemIconParams {
  icon: AppIcon;
  size?: IconSize;
  color?: string;
  clickable?: boolean;
  onClick?: () => void;
}

/**
 * 图标按钮组件参数接口
 */
export interface IconButtonParams {
  icon: AppIcon;
  size?: IconSize;
  iconColor?: string;
  backgroundColor?: ResourceColor;
  borderRadius?: number;
  padding?: number;
  onClick?: () => void;
}

/**
 * 系统图标组件
 */
@Component
export struct SystemIcon {
  @Prop icon: AppIcon;
  @Prop iconSize: IconSize = IconSize.MEDIUM;
  @Prop iconColor: string = IconColor.PRIMARY;
  @Prop clickable: boolean = false;
  onIconClick?: () => void;

  build() {
    Text() {
      SymbolSpan($r(this.icon))
        .fontSize(this.iconSize)
        .fontColor([this.iconColor as ResourceColor])
    }
    .onClick(() => {
      if (this.clickable && this.onIconClick) {
        this.onIconClick();
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
  @Prop iconSize: IconSize = IconSize.MEDIUM;
  @Prop iconColor: string = IconColor.PRIMARY;
  @Prop bgColor: ResourceColor = '#F2F2F7';
  @Prop radius: number = 8;
  @Prop paddingSize: number = 8;
  onButtonClick?: () => void;

  build() {
    Button() {
      Text() {
        SymbolSpan($r(this.icon))
          .fontSize(this.iconSize)
          .fontColor([this.iconColor as ResourceColor])
      }
    }
    .type(ButtonType.Normal)
    .backgroundColor(this.bgColor)
    .borderRadius(this.radius)
    .padding(this.paddingSize)
    .onClick(() => {
      if (this.onButtonClick) {
        this.onButtonClick();
      }
    })
  }
}

/**
 * 导航图标组件
 */
@Component
export struct NavIcon {
  @Prop icon: AppIcon;
  @Prop iconSize: IconSize = IconSize.LARGE;
  @Prop iconColor: string = IconColor.PRIMARY;
  onNavClick?: () => void;

  build() {
    Button() {
      Text() {
        SymbolSpan($r(this.icon))
          .fontSize(this.iconSize)
          .fontColor([this.iconColor as ResourceColor])
      }
    }
    .type(ButtonType.Normal)
    .backgroundColor('#00000000')
    .padding(8)
    .onClick(() => {
      if (this.onNavClick) {
        this.onNavClick();
      }
    })
  }
}

/**
 * 状态图标组件
 */
@Component
export struct StatusIcon {
  @Prop icon: AppIcon;
  @Prop iconSize: IconSize = IconSize.MEDIUM;
  @Prop status: 'normal' | 'success' | 'warning' | 'error' = 'normal';

  private getStatusColor(): string {
    switch (this.status) {
      case 'success':
        return IconColor.SUCCESS;
      case 'warning':
        return IconColor.WARNING;
      case 'error':
        return IconColor.ERROR;
      default:
        return IconColor.PRIMARY;
    }
  }

  build() {
    Text() {
      SymbolSpan($r(this.icon))
        .fontSize(this.iconSize)
        .fontColor([this.getStatusColor() as ResourceColor])
    }
  }
}