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
  EDIT = 'sys.symbol.square_and_pencil',
  SAVE = 'sys.symbol.checkmark',
  CANCEL = 'sys.symbol.xmark',
  CLOSE = 'sys.symbol.xmark',
  CONFIRM = 'sys.symbol.checkmark_circle',

  // 功能图标
  SEARCH = 'sys.symbol.magnifyingglass',
  FILTER = 'sys.symbol.line_3_horizontal',
  SORT = 'sys.symbol.sort',
  REFRESH = 'sys.symbol.arrow_clockwise',
  SETTINGS = 'sys.symbol.gearshape',
  MORE = 'sys.symbol.more',

  // 通信图标
  MESSAGE = 'sys.symbol.message',
  SEND = 'sys.symbol.paperplane',
  CALL = 'sys.symbol.phone',
  VIDEO = 'sys.symbol.video',

  // 媒体图标
  CAMERA = 'sys.symbol.camera',
  PHOTO = 'sys.symbol.picture',
  PLAY = 'sys.symbol.play',
  PAUSE = 'sys.symbol.pause',

  // 位置图标
  LOCATION = 'sys.symbol.position',
  MAP = 'sys.symbol.map',
  NEARBY = 'sys.symbol.discover',

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
  CHART = 'sys.symbol.histogram',
  PERFORMANCE = 'sys.symbol.bolt',
  AI = 'sys.symbol.robot',
  RECOMMEND = 'sys.symbol.lightbulb',
  SYNC = 'sys.symbol.repeat',
  DISTRIBUTED = 'sys.symbol.wifi',

  // 箭头图标
  ARROW_UP = 'sys.symbol.chevron_up',
  ARROW_DOWN = 'sys.symbol.chevron_down',
  ARROW_LEFT = 'sys.symbol.chevron_left',
  ARROW_RIGHT = 'sys.symbol.chevron_right',

  // 操作图标补充
  LIKE = 'sys.symbol.heart',
  SHARE = 'sys.symbol.share',
  CHECK = 'sys.symbol.checkmark',
  CHECK_ALL = 'sys.symbol.checkmark_circle',
  CLEAN = 'sys.symbol.trash',
  CLEAR = 'sys.symbol.trash',

  // 状态图标补充
  EMPTY = 'sys.symbol.inbox',
  LOADING = 'sys.symbol.arrow_clockwise',
  QUESTION = 'sys.symbol.questionmark_circle',
  SHIELD = 'sys.symbol.shield_checkered',

  // 系统功能图标
  MEMORY = 'sys.symbol.rectangle_stack',
  CPU = 'sys.symbol.circle_grid_2x2',
  BATTERY = 'sys.symbol.battery',
  SPEED = 'sys.symbol.dial',
  CACHE = 'sys.symbol.externaldrive',

  // 调试和开发图标
  DEBUG = 'sys.symbol.spanner',
  CODE = 'sys.symbol.doc_text',
  BUG = 'sys.symbol.exclamationmark',

  // 时间图标
  TIME = 'sys.symbol.clock',

  // 通用图标
  ALL = 'sys.symbol.square_grid_2x2'
}

// AppIconMap 用于在运行时通过 AppIcon 枚举值获取对应的资源字符串
export const AppIconMap: Map<AppIcon, string> = new Map<AppIcon, string>([
  [AppIcon.BACK, 'chevron_left'],
  [AppIcon.FORWARD, 'chevron_right'],
  [AppIcon.UP, 'chevron_up'],
  [AppIcon.DOWN, 'chevron_down'],
  [AppIcon.ADD, 'plus'],
  [AppIcon.REMOVE, 'minus'],
  [AppIcon.DELETE, 'trash'],
  [AppIcon.EDIT, 'square_and_pencil'],
  [AppIcon.SAVE, 'checkmark'],
  [AppIcon.CANCEL, 'xmark'],
  [AppIcon.CLOSE, 'xmark'],
  [AppIcon.CONFIRM, 'checkmark_circle'],
  [AppIcon.SEARCH, 'magnifyingglass'],
  [AppIcon.FILTER, 'line_3_horizontal'],
  [AppIcon.SORT, 'sort'],
  [AppIcon.REFRESH, 'arrow_clockwise'],
  [AppIcon.SETTINGS, 'gearshape'],
  [AppIcon.MORE, 'more'],
  [AppIcon.MESSAGE, 'message'],
  [AppIcon.SEND, 'paperplane'],
  [AppIcon.CALL, 'phone'],
  [AppIcon.VIDEO, 'video'],
  [AppIcon.CAMERA, 'camera'],
  [AppIcon.PHOTO, 'picture'],
  [AppIcon.PLAY, 'play'],
  [AppIcon.PAUSE, 'pause'],
  [AppIcon.LOCATION, 'location'],
  [AppIcon.MAP, 'map'],
  [AppIcon.PERSON, 'person'],
  [AppIcon.GROUP, 'person_2'],
  [AppIcon.HEART, 'heart'],
  [AppIcon.HEART_FILL, 'heart_fill'],
  [AppIcon.STAR, 'star'],
  [AppIcon.STAR_FILL, 'star_fill'],
  [AppIcon.INFO, 'info_circle'],
  [AppIcon.WARNING, 'exclamationmark_triangle'],
  [AppIcon.ERROR, 'xmark_circle'],
  [AppIcon.SUCCESS, 'checkmark_circle'],
  [AppIcon.NOTIFICATION, 'bell'],
  [AppIcon.NOTIFICATION_FILL, 'bell_fill'],
  [AppIcon.CHART, 'histogram'],
  [AppIcon.PERFORMANCE, 'bolt'],
  [AppIcon.AI, 'robot'],
  [AppIcon.RECOMMEND, 'lightbulb'],
  [AppIcon.SYNC, 'repeat'],
  [AppIcon.DISTRIBUTED, 'wifi'],
  [AppIcon.ARROW_UP, 'chevron_up'],
  [AppIcon.ARROW_DOWN, 'chevron_down'],
  [AppIcon.ARROW_LEFT, 'chevron_left'],
  [AppIcon.ARROW_RIGHT, 'chevron_right'],
  [AppIcon.LIKE, 'heart'],
  [AppIcon.SHARE, 'share'],
  [AppIcon.CHECK, 'checkmark'],
  [AppIcon.CHECK_ALL, 'checkmark_circle'],
  [AppIcon.CLEAN, 'trash'],
  [AppIcon.CLEAR, 'trash'],
  [AppIcon.EMPTY, 'inbox'],
  [AppIcon.LOADING, 'arrow_clockwise'],
  [AppIcon.QUESTION, 'questionmark_circle'],
  [AppIcon.SHIELD, 'shield_checkered'],
  [AppIcon.MEMORY, 'rectangle_stack'],
  [AppIcon.CPU, 'circle_grid_2x2'],
  [AppIcon.BATTERY, 'battery'],
  [AppIcon.SPEED, 'dial'],
  [AppIcon.CACHE, 'externaldrive'],
  [AppIcon.DEBUG, 'spanner'],
  [AppIcon.CODE, 'doc_text'],
  [AppIcon.BUG, 'exclamationmark'],
  [AppIcon.ALL, 'square_grid_2x2']
]);

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
  PRIMARY = '#007AFF', // DesignConstants.COLOR.BRAND_PRIMARY
  SECONDARY = '#8E8E93', // DesignConstants.COLOR.TEXT_SECONDARY_LIGHT
  SUCCESS = '#34C759', // DesignConstants.COLOR.SUCCESS
  WARNING = '#FFCC00', // DesignConstants.COLOR.WARNING
  ERROR = '#FF3B30', // DesignConstants.COLOR.ERROR
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
  color?: string | ResourceColor;
  clickable?: boolean;
  onClick?: () => void;
}

/**
 * 图标按钮组件参数接口
 */
export interface IconButtonParams {
  icon: AppIcon;
  size?: IconSize;
  iconColor?: string | ResourceColor;
  backgroundColor?: ResourceColor;
  borderRadius?: number;
  padding?: number;
  onClick?: () => void;
}

/**
 * 系统图标组件
 */
@Reusable
@Component
export struct SystemIcon {
  @Prop icon: AppIcon;
  @Prop iconSize: IconSize = IconSize.MEDIUM;
  @Prop iconColor: string | ResourceColor = $r('app.color.brand_primary');
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
@Reusable
@Component
export struct IconButton {
  @Prop icon: AppIcon;
  @Prop iconSize: IconSize = IconSize.LARGE;
  @Prop iconColor: string | ResourceColor = $r('app.color.brand_primary');
  @Prop bgColor: ResourceColor = $r('app.color.bg_tertiary'); // Default background
  @Prop radius: number = 24;
  @Prop paddingSize: number = 12;
  @Prop minButtonSize: number = 48;
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
    .width(this.minButtonSize)
    .height(this.minButtonSize)
    .onClick(() => {
      if (this.onButtonClick) {
        this.onButtonClick();
      }
    })
  }
}
