/**
 * 推荐内容卡片组件
 * 展示 AI 推荐的内容，包括动态、群组、用户等
 */
import { RecommendationItem, AIRecommendationService } from '../common/services/AIRecommendationService';
import { SystemIcon } from './SystemIcon';
import { AppIcon } from '../common/constants/AppIcon';
import { SystemToast } from '../common/utils/SystemToast';
import { AccessibilityManager } from '../common/accessibility/AccessibilityManager';

@Component
export struct RecommendationCard {
  @Prop recommendation: RecommendationItem;
  @State private isExpanded: boolean = false;
  @State private isLoading: boolean = false;

  private aiService = AIRecommendationService.getInstance();
  private accessibilityManager = AccessibilityManager.getInstance();

  /**
   * 处理推荐项点击
   */
  private async onRecommendationClick(): Promise<void> {
    try {
      this.isLoading = true;
      
      // 记录用户行为
      await this.aiService.recordUserBehavior({
        userId: 'current_user', // 这里应该从全局状态获取
        action: 'view',
        targetType: this.recommendation.type,
        targetId: this.recommendation.id,
        timestamp: Date.now()
      });

      // 无障碍播报
      await this.accessibilityManager.announceText(
        `正在查看推荐的${this.getTypeText()}：${this.recommendation.title}`
      );

      // 根据推荐类型执行相应操作
      switch (this.recommendation.type) {
        case 'moment':
          this.navigateToMoment();
          break;
        case 'group':
          this.navigateToGroup();
          break;
        case 'user':
          this.navigateToUser();
          break;
        case 'location':
          this.navigateToLocation();
          break;
      }
      
    } catch (error) {
      SystemToast.showError(`查看推荐失败: ${error}`);
    } finally {
      this.isLoading = false;
    }
  }

  /**
   * 处理点赞操作
   */
  private async onLike(): Promise<void> {
    try {
      await this.aiService.recordUserBehavior({
        userId: 'current_user',
        action: 'like',
        targetType: this.recommendation.type,
        targetId: this.recommendation.id,
        timestamp: Date.now()
      });

      SystemToast.showSuccess('已点赞');
      
      // 无障碍播报
      await this.accessibilityManager.announceText('已点赞该推荐内容');
      
    } catch (error) {
      SystemToast.showError(`点赞失败: ${error}`);
    }
  }

  /**
   * 处理分享操作
   */
  private async onShare(): Promise<void> {
    try {
      await this.aiService.recordUserBehavior({
        userId: 'current_user',
        action: 'share',
        targetType: this.recommendation.type,
        targetId: this.recommendation.id,
        timestamp: Date.now()
      });

      SystemToast.showSuccess('已分享');
      
      // 无障碍播报
      await this.accessibilityManager.announceText('已分享该推荐内容');
      
    } catch (error) {
      SystemToast.showError(`分享失败: ${error}`);
    }
  }

  /**
   * 获取类型文本
   */
  private getTypeText(): string {
    const typeMap = {
      'moment': '动态',
      'group': '群组',
      'user': '用户',
      'location': '位置'
    };
    return typeMap[this.recommendation.type] || '内容';
  }

  /**
   * 获取类型图标
   */
  private getTypeIcon(): AppIcon {
    const iconMap = {
      'moment': AppIcon.CAMERA,
      'group': AppIcon.GROUP,
      'user': AppIcon.PERSON,
      'location': AppIcon.MAP
    };
    return iconMap[this.recommendation.type] || AppIcon.INFO;
  }

  /**
   * 获取推荐分数颜色
   */
  private getScoreColor(): ResourceColor {
    if (this.recommendation.score >= 0.8) return Color.Green;
    if (this.recommendation.score >= 0.6) return Color.Orange;
    return Color.Gray;
  }

  /**
   * 导航方法
   */
  private navigateToMoment(): void {
    SystemToast.showInfo('跳转到动态详情');
  }

  private navigateToGroup(): void {
    SystemToast.showInfo('跳转到群组详情');
  }

  private navigateToUser(): void {
    SystemToast.showInfo('跳转到用户详情');
  }

  private navigateToLocation(): void {
    SystemToast.showInfo('跳转到位置详情');
  }

  build() {
    Column({ space: 12 }) {
      // 推荐卡片头部
      Row({ space: 12 }) {
        // 类型图标
        SystemIcon({
          icon: this.getTypeIcon(),
          iconSize: 24,
          iconColor: $r('sys.color.ohos_id_color_emphasize')
        })

        // 标题和描述
        Column({ space: 4 }) {
          Text(this.recommendation.title)
            .fontSize(16)
            .fontWeight(FontWeight.Medium)
            .fontColor($r('sys.color.ohos_id_color_text_primary'))
            .maxLines(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
            .alignSelf(ItemAlign.Start)

          Text(this.recommendation.description)
            .fontSize(14)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
            .maxLines(2)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
            .alignSelf(ItemAlign.Start)
        }
        .alignItems(HorizontalAlign.Start)
        .layoutWeight(1)

        // 推荐分数
        Row({ space: 4 }) {
          Circle({ width: 8, height: 8 })
            .fill(this.getScoreColor())

          Text(`${(this.recommendation.score * 100).toFixed(0)}%`)
            .fontSize(12)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
        }
      }
      .width('100%')

      // 推荐原因
      if (this.recommendation.reason) {
        Row({ space: 6 }) {
          SystemIcon({
            icon: AppIcon.INFO,
            iconSize: 14,
            iconColor: $r('sys.color.ohos_id_color_text_secondary')
          })

          Text(this.recommendation.reason)
            .fontSize(12)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
            .layoutWeight(1)
        }
        .width('100%')
        .padding({ left: 8, right: 8 })
      }

      // 操作按钮
      Row({ space: 16 }) {
        Button(this.isLoading ? '加载中...' : '查看')
          .type(ButtonType.Capsule)
          .backgroundColor($r('sys.color.ohos_id_color_emphasize'))
          .fontColor(Color.White)
          .fontSize(14)
          .layoutWeight(1)
          .enabled(!this.isLoading)
          .onClick(() => this.onRecommendationClick())

        // 快捷操作按钮
        Row({ space: 12 }) {
          SystemIcon({
            icon: AppIcon.LIKE,
            iconSize: 20,
            onIconClick: () => this.onLike()
          })

          SystemIcon({
            icon: AppIcon.SHARE,
            iconSize: 20,
            onIconClick: () => this.onShare()
          })

          SystemIcon({
            icon: this.isExpanded ? AppIcon.ARROW_UP : AppIcon.ARROW_DOWN,
            iconSize: 20,
            onIconClick: () => {
              this.isExpanded = !this.isExpanded;
            }
          })
        }
      }
      .width('100%')

      // 展开的详细信息
      if (this.isExpanded && this.recommendation.metadata) {
        Column({ space: 8 }) {
          Divider()
            .color($r('sys.color.ohos_id_color_list_separator'))

          Text('详细信息')
            .fontSize(14)
            .fontWeight(FontWeight.Medium)
            .fontColor($r('sys.color.ohos_id_color_text_primary'))
            .alignSelf(ItemAlign.Start)

          // 显示元数据
          ForEach(Object.entries(this.recommendation.metadata), ([key, value]) => {
            if (key !== 'algorithm' && key !== 'targetId') {
              Row() {
                Text(`${key}:`)
                  .fontSize(12)
                  .fontColor($r('sys.color.ohos_id_color_text_secondary'))
                  .width(80)

                Text(String(value))
                  .fontSize(12)
                  .fontColor($r('sys.color.ohos_id_color_text_primary'))
                  .layoutWeight(1)
              }
              .width('100%')
            }
          })

          // 推荐算法信息
          if (this.recommendation.metadata.algorithm) {
            Text(`推荐算法: ${this.recommendation.metadata.algorithm}`)
              .fontSize(11)
              .fontColor($r('sys.color.ohos_id_color_text_tertiary'))
              .alignSelf(ItemAlign.Start)
          }
        }
        .width('100%')
        .padding({ top: 8 })
      }
    }
    .width('100%')
    .padding(16)
    .backgroundColor($r('sys.color.ohos_id_color_background'))
    .borderRadius(12)
    .shadow({
      radius: 8,
      color: $r('sys.color.ohos_id_color_shadow'),
      offsetX: 0,
      offsetY: 2
    })
    // 无障碍属性
    .accessibilityGroup(true)
    .accessibilityText(`推荐${this.getTypeText()}：${this.recommendation.title}。${this.recommendation.description}。推荐原因：${this.recommendation.reason}`)
    .accessibilityDescription(`推荐分数${(this.recommendation.score * 100).toFixed(0)}%，点击查看详情`)
  }
}
