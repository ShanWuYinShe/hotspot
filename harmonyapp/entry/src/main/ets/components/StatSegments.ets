import { DesignConstants } from '../constants/DesignConstants';
import { AppThemeManager } from '../core/theme/AppThemeManager';
import { KEY_RESOLVED_THEME_ID } from '../core/theme/AppThemeTypes';

/**
 * 统计分段组件（对齐前端 stat-segments.svelte）
 * 全站「数值 + 标签」统计条的唯一实现（禁止再内联手搭）。
 *
 * layout：
 * - 'row'：横向分段，用竖分隔线区隔（适合概览卡右侧填充）
 * - 'grid'：网格分段，居中 + 间距（适合 hero-stats 三栏统计）
 *
 * 用 Row + layoutWeight 等分实现，规避 Grid columnsTemplate 在动态列数下的不确定性。
 */

export type StatSegmentsLayout = 'row' | 'grid';

export interface StatSegmentItem {
  value: string;
  label: string;
}

@Reusable
@Component
export struct StatSegments {
  /** segments 数据：JSON 字符串（ArkUI @Prop 复杂数组用字符串约定） */
  @Prop segmentsJson: string = '[]';
  @Prop layout: StatSegmentsLayout = 'row';

  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'default_clean_light';

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  private getSegments(): StatSegmentItem[] {
    try {
      const parsed: StatSegmentItem[] = JSON.parse(this.segmentsJson) as StatSegmentItem[];
      return Array.isArray(parsed) ? parsed : [];
    } catch (e) {
      return [];
    }
  }

  @Builder
  SegmentBuilder(item: StatSegmentItem) {
    Column() {
      Text(item.value)
        .fontSize(DesignConstants.FONT.SIZE.XL) // 20
        .fontWeight(FontWeight.Bold)
        .fontColor(this.getPalette().textPrimary)
        .maxLines(1)
        .textOverflow({ overflow: TextOverflow.Ellipsis })
      Text(item.label)
        .fontSize(DesignConstants.FONT.SIZE.XS2) // 10
        .fontColor(this.getPalette().textSecondary)
        .margin({ top: 2 })
        .maxLines(1)
        .textOverflow({ overflow: TextOverflow.Ellipsis })
    }
    .alignItems(this.layout === 'grid' ? HorizontalAlign.Center : HorizontalAlign.Start)
    .layoutWeight(1)
    .constraintSize({ minWidth: this.layout === 'grid' ? 0 : 56 })
  }

  build() {
    Row() {
      ForEach(this.getSegments(), (item: StatSegmentItem, index: number) => {
        Column() {
          this.SegmentBuilder(item)
        }
        .layoutWeight(1)

        if (this.layout === 'row' && index < this.getSegments().length - 1) {
          Divider()
            .vertical(true)
            .height(28)
            .color(this.getPalette().borderColor)
            .margin({ left: DesignConstants.SPACING.SM, right: DesignConstants.SPACING.SM })
        }
      }, (item: StatSegmentItem, index: number) => `${item.label}-${index}`)
    }
    .width('100%')
    .alignItems(VerticalAlign.Center)
    .justifyContent(this.layout === 'grid' ? FlexAlign.SpaceEvenly : FlexAlign.Start)
  }
}
