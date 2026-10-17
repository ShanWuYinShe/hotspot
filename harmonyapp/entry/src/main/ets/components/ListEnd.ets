import { DesignConstants } from '../constants/DesignConstants';
import { AppThemeManager } from '../core/theme/AppThemeManager';
import { DEFAULT_RESOLVED_THEME_ID, KEY_RESOLVED_THEME_ID } from '../core/theme/AppThemeTypes';

/**
 * ListEnd —— 分页列表「已到底」收尾提示（对齐前端 components/data-display/ListEnd.svelte）。
 *
 * 为什么必须有：分页列表内容耗尽时若只是「加载更多」指示消失，用户无法区分
 * 「已到底」与「加载卡住」——前端 AGENTS.md 把四态列为硬性契约：
 * 空列表 EmptyState / 加载中 InlineLoading / 失败 LoadError / 到底 ListEnd，
 * 四者各司其职、不得互相替代。
 *
 * 用法（列表尾部）：
 *   if (this.hasMore) {
 *     LoadingMoreIndicator()
 *   } else {
 *     ListEnd()
 *   }
 *
 * 此前鸿蒙端 4 处各写一遍、padding 各不相同（PAGE_TOP_GAP/SECTION_GAP、LG/XS、
 * MD/LG 三种），7 处分页页面则完全没有收尾提示；统一收敛到本组件。
 */
@Component
export struct ListEnd {
  /** 自定义文案（默认「没有更多了」） */
  @Prop label: string = '没有更多了';
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = DEFAULT_RESOLVED_THEME_ID;

  build() {
    Text(this.label)
      .fontSize(DesignConstants.FONT.SIZE.XS)
      .fontColor(AppThemeManager.getPaletteById(this.resolvedThemeId).textTertiary)
      .width('100%')
      .textAlign(TextAlign.Center)
      // 状态提示需可被读屏播报（非装饰文本）
      .accessibilityText(this.label)
      // 对齐前端 ListEnd：py-6（上下各 24）
      .padding({
        top: DesignConstants.SPACING.XL,
        bottom: DesignConstants.SPACING.XL
      })
  }
}
