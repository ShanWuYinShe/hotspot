/**
 * 系统图标组件
 * 提供统一的图标显示和交互功能
 */


import { AppThemeManager } from '../core/theme/AppThemeManager';
import { DEFAULT_RESOLVED_THEME_ID, KEY_RESOLVED_THEME_ID } from '../core/theme/AppThemeTypes';
import { DesignConstants } from '../constants/DesignConstants';

const DEFAULT_THEME_ICON_SENTINEL: string = '__theme_icon_default__';

/**
 * 应用图标枚举
 */
export enum AppIcon {
  // 导航图标
  HOME = 'sys.symbol.house',
  COMPASS = 'sys.symbol.compass',
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
  NEARBY = 'sys.symbol.discover', // 系统符号 id 保留 discover，语义为「附近」

  // 用户图标
  PERSON = 'sys.symbol.person',
  GROUP = 'sys.symbol.person_2',

  // 内容类型图标
  COFFEE = 'sys.symbol.cup',
  MUSIC = 'sys.symbol.music',
  /** ✨ 星光（对齐前端 lucide Sparkles；系统符号无 sparkles，用最接近的 wand_and_stars） */
  SPARKLES = 'sys.symbol.wand_and_stars',
  /** ☺ 表情（对齐前端 lucide Smile；系统符号 face_smiling） */
  EMOJI = 'sys.symbol.face_smiling',

  // 状态图标
  HEART = 'sys.symbol.heart',
  HEART_FILL = 'sys.symbol.heart_fill',
  STAR = 'sys.symbol.star',
  STAR_FILL = 'sys.symbol.star_fill',
  MOON = 'sys.symbol.moon',

  // 系统图标
  INFO = 'sys.symbol.info_circle',
  WARNING = 'sys.symbol.exclamationmark_triangle',
  ERROR = 'sys.symbol.xmark_circle',
  SUCCESS = 'sys.symbol.checkmark_circle',
  LOCK = 'sys.symbol.lock',

  // 新增智能功能图标
  NOTIFICATION = 'sys.symbol.bell',
  NOTIFICATION_FILL = 'sys.symbol.bell_fill',
  CHART = 'sys.symbol.histogram',
  PERFORMANCE = 'sys.symbol.bolt',
  AI = 'sys.symbol.robot',
  RECOMMEND = 'sys.symbol.lightbulb',
  SYNC = 'sys.symbol.repeat',
  DISTRIBUTED = 'sys.symbol.wifi',
  PALETTE = 'sys.symbol.paintpalette',

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

  // 置顶/图钉图标
  PIN = 'sys.symbol.pin',
  PIN_FILL = 'sys.symbol.pin_fill',

  // 通用图标
  ALL = 'sys.symbol.square_grid_2x2'
}

// AppIconMap 用于在运行时通过 AppIcon 枚举值获取对应的资源字符串
export const AppIconMap: Map<AppIcon, string> = new Map<AppIcon, string>([
  [AppIcon.HOME, 'house'],
  [AppIcon.COMPASS, 'compass'],
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
  [AppIcon.LOCATION, 'position'],
  [AppIcon.MAP, 'map'],
  [AppIcon.NEARBY, 'discover'],
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
  [AppIcon.LOCK, 'lock'],
  [AppIcon.NOTIFICATION, 'bell'],
  [AppIcon.NOTIFICATION_FILL, 'bell_fill'],
  [AppIcon.CHART, 'histogram'],
  [AppIcon.PERFORMANCE, 'bolt'],
  [AppIcon.AI, 'robot'],
  [AppIcon.RECOMMEND, 'lightbulb'],
  [AppIcon.SYNC, 'repeat'],
  [AppIcon.DISTRIBUTED, 'wifi'],
  [AppIcon.PALETTE, 'paintpalette'],
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
  [AppIcon.TIME, 'clock'],
  [AppIcon.MOON, 'moon'],
  [AppIcon.PIN, 'pin'],
  [AppIcon.PIN_FILL, 'pin_fill'],
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

// IconColor 已被基于主题调色板的颜色逻辑取代，保留按需使用 AppThemeManager 与 DesignConstants。

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
 * 系统图标组件
 */
@Reusable
@Component
export struct SystemIcon {
  @Prop icon: AppIcon;
  @Prop iconSize: IconSize = IconSize.MEDIUM;
  @Prop iconColor: string | ResourceColor = DEFAULT_THEME_ICON_SENTINEL;
  @Prop clickable: boolean = false;
  onIconClick?: () => void;
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = DEFAULT_RESOLVED_THEME_ID;

  private getDefaultIconColor(): string {
    return AppThemeManager.getPaletteById(this.resolvedThemeId).accentBackground;
  }

  private getResolvedIconColor(): string | ResourceColor {
    return this.iconColor === DEFAULT_THEME_ICON_SENTINEL ? this.getDefaultIconColor() : this.iconColor;
  }

  build() {
    Text() {
      SymbolSpan($r(this.icon))
        .fontSize(this.iconSize)
        .fontColor([this.getResolvedIconColor() as ResourceColor])
    }
    .onClick(() => {
      if (this.clickable && this.onIconClick) {
        this.onIconClick();
      }
    })
  }
}

// 旧 IconButton（SymbolSpan + 手写 48vp 底色钮）已删除：带交互的图标钮统一走
// SurfaceIconButton（LucideIcon + 44vp 命中区 + a11y + spring 按压，审查 中-1）。
