/**
 * 推荐页面
 * 展示完整的智能推荐内容，包括不同类型的推荐和个性化设置
 */
import { AIRecommendationService, RecommendationItem, UserPreferences } from '../common/services/AIRecommendationService';
import { SmartNotificationService } from '../common/services/SmartNotificationService';
import { AccessibilityManager } from '../common/accessibility/AccessibilityManager';
import { GlobalStateManager, GlobalState } from '../common/state/GlobalStateManager';
import { RecommendationCard } from '../components/RecommendationCard';
import { SystemIcon } from '../components/SystemIcon';
import { AppIcon } from '../common/constants/AppIcon';
import { SystemToast } from '../common/utils/SystemToast';
import { hilog } from '@kit.PerformanceAnalysisKit';

const DOMAIN_NUMBER: number = 0xFF20;
const TAG: string = 'RecommendationPage';

@Entry
@Component
struct RecommendationPage {
  @State private globalState: GlobalState = GlobalStateManager.getInstance().getState();
  @State private recommendations: RecommendationItem[] = [];
  @State private userPreferences: UserPreferences | null = null;
  @State private selectedFilter: 'all' | 'moment' | 'group' | 'user' | 'location' = 'all';
  @State private isLoading: boolean = false;
  @State private showPreferences: boolean = false;
  @State private refreshing: boolean = false;

  private aiService = AIRecommendationService.getInstance();
  private notificationService = SmartNotificationService.getInstance();
  private accessibilityManager = AccessibilityManager.getInstance();
  private stateManager = GlobalStateManager.getInstance();

  aboutToAppear() {
    this.loadRecommendations();
    this.loadUserPreferences();
  }

  /**
   * 加载推荐内容
   */
  private async loadRecommendations(): Promise<void> {
    if (!this.globalState.currentUser?.user_id) {
      SystemToast.showWarning('请先登录');
      return;
    }

    try {
      this.isLoading = true;
      
      // 获取推荐内容
      const allRecommendations = await this.aiService.getRecommendations(
        this.globalState.currentUser.user_id,
        20
      );

      // 根据过滤器筛选
      if (this.selectedFilter === 'all') {
        this.recommendations = allRecommendations;
      } else {
        this.recommendations = allRecommendations.filter(r => r.type === this.selectedFilter);
      }

      // 记录用户行为
      await this.aiService.recordUserBehavior({
        userId: this.globalState.currentUser.user_id,
        action: 'view',
        targetType: 'user',
        targetId: 'recommendation_page',
        timestamp: Date.now()
      });

      // 无障碍播报
      await this.accessibilityManager.announceText(
        `已加载 ${this.recommendations.length} 条推荐内容`
      );

      hilog.info(DOMAIN_NUMBER, TAG, `加载了 ${this.recommendations.length} 条推荐内容`);
      
    } catch (error) {
      SystemToast.showError(`加载推荐失败: ${error}`);
      hilog.error(DOMAIN_NUMBER, TAG, `加载推荐失败: ${JSON.stringify(error)}`);
    } finally {
      this.isLoading = false;
    }
  }

  /**
   * 加载用户偏好
   */
  private loadUserPreferences(): void {
    if (!this.globalState.currentUser?.user_id) return;

    try {
      this.userPreferences = this.aiService.getUserPreferencesAnalysis(
        this.globalState.currentUser.user_id
      );
      
      hilog.info(DOMAIN_NUMBER, TAG, '用户偏好分析已加载');
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `加载用户偏好失败: ${JSON.stringify(error)}`);
    }
  }

  /**
   * 刷新推荐
   */
  private async onRefresh(): Promise<void> {
    this.refreshing = true;
    await this.loadRecommendations();
    
    setTimeout(() => {
      this.refreshing = false;
      SystemToast.showSuccess('推荐内容已刷新');
    }, 500);
  }

  /**
   * 切换过滤器
   */
  private onFilterChange(filter: typeof this.selectedFilter): void {
    this.selectedFilter = filter;
    this.loadRecommendations();
    
    // 无障碍播报
    this.accessibilityManager.announceText(
      `已切换到${this.getFilterText(filter)}推荐`
    );
  }

  /**
   * 获取过滤器文本
   */
  private getFilterText(filter: typeof this.selectedFilter): string {
    const filterMap = {
      'all': '全部',
      'moment': '动态',
      'group': '群组',
      'user': '用户',
      'location': '位置'
    };
    return filterMap[filter];
  }

  /**
   * 获取过滤器图标
   */
  private getFilterIcon(filter: typeof this.selectedFilter): AppIcon {
    const iconMap = {
      'all': AppIcon.ALL,
      'moment': AppIcon.CAMERA,
      'group': AppIcon.GROUP,
      'user': AppIcon.PERSON,
      'location': AppIcon.MAP
    };
    return iconMap[filter];
  }

  /**
   * 返回上一页
   */
  private onBack(): void {
    const routerInstance = this.getUIContext().getRouter();
    routerInstance.back();
  }

  /**
   * 清除推荐数据
   */
  private async onClearRecommendationData(): Promise<void> {
    if (!this.globalState.currentUser?.user_id) return;

    try {
      await this.aiService.clearUserData(this.globalState.currentUser.user_id);
      this.recommendations = [];
      this.userPreferences = null;
      
      SystemToast.showSuccess('推荐数据已清除');
      
      // 无障碍播报
      await this.accessibilityManager.announceText('推荐数据已清除');
      
    } catch (error) {
      SystemToast.showError(`清除数据失败: ${error}`);
    }
  }

  build() {
    Column() {
      // 标题栏
      Row() {
        SystemIcon({
          icon: AppIcon.BACK,
          iconSize: 24,
          onIconClick: () => this.onBack()
        })

        Text('智能推荐')
          .fontSize(20)
          .fontWeight(FontWeight.Bold)
          .fontColor($r('sys.color.ohos_id_color_text_primary'))
          .margin({ left: 16 })
          .layoutWeight(1)

        // 操作按钮
        Row({ space: 12 }) {
          SystemIcon({
            icon: AppIcon.REFRESH,
            iconSize: 20,
            onIconClick: () => this.onRefresh()
          })

          SystemIcon({
            icon: AppIcon.SETTINGS,
            iconSize: 20,
            onIconClick: () => {
              this.showPreferences = !this.showPreferences;
            }
          })
        }
      }
      .width('100%')
      .height(56)
      .padding({ left: 16, right: 16 })
      .backgroundColor($r('sys.color.ohos_id_color_background'))

      // 用户偏好分析（可展开）
      if (this.showPreferences && this.userPreferences) {
        Column({ space: 12 }) {
          Row() {
            Text('个性化分析')
              .fontSize(16)
              .fontWeight(FontWeight.Medium)
              .fontColor($r('sys.color.ohos_id_color_text_primary'))
              .layoutWeight(1)

            SystemIcon({
              icon: AppIcon.ARROW_UP,
              iconSize: 16,
              onIconClick: () => {
                this.showPreferences = false;
              }
            })
          }
          .width('100%')

          // 兴趣标签
          if (this.userPreferences.interests.length > 0) {
            Column({ space: 8 }) {
              Text('兴趣偏好')
                .fontSize(14)
                .fontColor($r('sys.color.ohos_id_color_text_secondary'))
                .alignSelf(ItemAlign.Start)

              Flex({ wrap: FlexWrap.Wrap, space: { main: 8, cross: 8 } }) {
                ForEach(this.userPreferences.interests.slice(0, 10), (interest: string) => {
                  Text(interest)
                    .fontSize(12)
                    .fontColor($r('sys.color.ohos_id_color_emphasize'))
                    .padding({ left: 8, right: 8, top: 4, bottom: 4 })
                    .backgroundColor($r('sys.color.ohos_id_color_background_light'))
                    .borderRadius(12)
                })
              }
            }
            .width('100%')
            .alignItems(HorizontalAlign.Start)
          }

          // 其他偏好信息
          Grid() {
            GridItem() {
              this.buildPreferenceItem('位置偏好', this.getLocationPreferenceText())
            }
            GridItem() {
              this.buildPreferenceItem('社交程度', this.getSocialLevelText())
            }
            GridItem() {
              this.buildPreferenceItem('活跃时段', this.userPreferences.activeTimeRanges.join(', '))
            }
            GridItem() {
              this.buildPreferenceItem('内容类型', this.userPreferences.contentTypes.join(', '))
            }
          }
          .columnsTemplate('1fr 1fr')
          .rowsGap(8)
          .columnsGap(8)
          .width('100%')

          // 管理按钮
          Button('清除推荐数据')
            .type(ButtonType.Normal)
            .fontSize(14)
            .fontColor(Color.Red)
            .backgroundColor($r('sys.color.ohos_id_color_background_light'))
            .onClick(() => this.onClearRecommendationData())
        }
        .width('100%')
        .padding(16)
        .backgroundColor($r('sys.color.ohos_id_color_background'))
        .margin({ bottom: 8 })
      }

      // 过滤器
      Scroll() {
        Row({ space: 8 }) {
          ForEach([
            { key: 'all', label: '全部' },
            { key: 'moment', label: '动态' },
            { key: 'group', label: '群组' },
            { key: 'user', label: '用户' },
            { key: 'location', label: '位置' }
          ], (filter) => {
            Button() {
              Row({ space: 6 }) {
                SystemIcon({
                  icon: this.getFilterIcon(filter.key as any),
                  iconSize: 16,
                  iconColor: this.selectedFilter === filter.key 
                    ? Color.White 
                    : $r('sys.color.ohos_id_color_text_primary')
                })

                Text(filter.label)
                  .fontSize(14)
                  .fontColor(this.selectedFilter === filter.key 
                    ? Color.White 
                    : $r('sys.color.ohos_id_color_text_primary'))
              }
            }
            .type(this.selectedFilter === filter.key ? ButtonType.Capsule : ButtonType.Normal)
            .backgroundColor(this.selectedFilter === filter.key 
              ? $r('sys.color.ohos_id_color_emphasize') 
              : $r('sys.color.ohos_id_color_background'))
            .padding({ left: 16, right: 16, top: 8, bottom: 8 })
            .onClick(() => this.onFilterChange(filter.key as any))
          })
        }
        .padding({ left: 16, right: 16 })
      }
      .scrollable(ScrollDirection.Horizontal)
      .scrollBar(BarState.Off)
      .margin({ bottom: 16 })

      // 推荐内容列表
      if (this.isLoading || this.refreshing) {
        Column() {
          LoadingProgress()
            .width(48)
            .height(48)
            .color($r('sys.color.ohos_id_color_emphasize'))
          
          Text(this.refreshing ? '刷新中...' : '加载中...')
            .fontSize(14)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
            .margin({ top: 16 })
        }
        .width('100%')
        .layoutWeight(1)
        .justifyContent(FlexAlign.Center)
      } else if (this.recommendations.length === 0) {
        Column() {
          SystemIcon({
            icon: AppIcon.EMPTY,
            iconSize: 64,
            iconColor: $r('sys.color.ohos_id_color_text_tertiary')
          })
          
          Text('暂无推荐内容')
            .fontSize(16)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
            .margin({ top: 16 })

          Text('多使用应用功能，我们会为您推荐更多感兴趣的内容')
            .fontSize(14)
            .fontColor($r('sys.color.ohos_id_color_text_tertiary'))
            .textAlign(TextAlign.Center)
            .margin({ top: 8 })
            .padding({ left: 32, right: 32 })

          Button('刷新推荐')
            .margin({ top: 24 })
            .onClick(() => this.onRefresh())
        }
        .width('100%')
        .layoutWeight(1)
        .justifyContent(FlexAlign.Center)
      } else {
        List({ space: 16 }) {
          ForEach(this.recommendations, (recommendation: RecommendationItem) => {
            ListItem() {
              RecommendationCard({ recommendation: recommendation })
            }
          })
        }
        .width('100%')
        .layoutWeight(1)
        .padding({ left: 16, right: 16, bottom: 16 })
        .scrollBar(BarState.Auto)
      }
    }
    .width('100%')
    .height('100%')
    .backgroundColor($r('sys.color.ohos_id_color_background_light'))
    // 无障碍属性
    .accessibilityGroup(true)
    .accessibilityText('智能推荐页面')
    .accessibilityDescription(`当前显示${this.getFilterText(this.selectedFilter)}推荐，共${this.recommendations.length}条内容`)
  }

  /**
   * 构建偏好项
   */
  @Builder
  private buildPreferenceItem(title: string, value: string) {
    Column({ space: 4 }) {
      Text(title)
        .fontSize(12)
        .fontColor($r('sys.color.ohos_id_color_text_secondary'))
        .alignSelf(ItemAlign.Start)

      Text(value || '未设置')
        .fontSize(14)
        .fontColor($r('sys.color.ohos_id_color_text_primary'))
        .maxLines(1)
        .textOverflow({ overflow: TextOverflow.Ellipsis })
        .alignSelf(ItemAlign.Start)
    }
    .width('100%')
    .alignItems(HorizontalAlign.Start)
    .padding(8)
    .backgroundColor($r('sys.color.ohos_id_color_background_light'))
    .borderRadius(6)
  }

  /**
   * 获取位置偏好文本
   */
  private getLocationPreferenceText(): string {
    if (!this.userPreferences) return '未设置';
    
    const map = {
      'nearby': '附近',
      'city': '同城',
      'global': '全球'
    };
    return map[this.userPreferences.locationPreference] || '未设置';
  }

  /**
   * 获取社交程度文本
   */
  private getSocialLevelText(): string {
    if (!this.userPreferences) return '未设置';
    
    const map = {
      'introvert': '内向',
      'moderate': '适中',
      'extrovert': '外向'
    };
    return map[this.userPreferences.socialLevel] || '未设置';
  }
}
