/**
 * 极简生活瞬间页面
 * 使用HarmonyOS原生能力，专注核心功能
 */

import { GlobalStateManager, GlobalState } from '../common/state/GlobalStateManager';
import { ApiClient } from '../common/api/ApiClient';
import { PageNames } from '../constants/AppConstants';
import { LikeMomentRequest, SearchMomentsByLocationResponse } from '../common/types/BackendApiTypes';
import { SystemAvatar } from '../components/SystemAvatar';
import { IntelligentLocationCard } from '../components/IntelligentLocationCard';
import { SystemLoadingView } from '../components/SystemLoadingView';
import { hilog } from '@kit.PerformanceAnalysisKit';

const DOMAIN_NUMBER: number = 0xFF05;
const TAG: string = 'MomentPage';

@Entry
@Component
struct MomentPage {
  @State globalState: GlobalState = GlobalStateManager.getInstance().getState();
  private stateManager = GlobalStateManager.getInstance();
  @State moments: SearchMomentsByLocationResponse[] = [];
  @State isLoading: boolean = false;
  @State errorMessage: string = '';

  async aboutToAppear() {
    // 订阅状态变化
    this.stateManager.subscribe('momentpage_page', (state: GlobalState) => {
      this.globalState = state;
    });
    await this.loadNearbyMoments();
  }

  aboutToDisappear() {
    // 取消订阅
    this.stateManager.unsubscribe('momentpage_page');
  }

  /**
   * 加载附近瞬间
   */
  private async loadNearbyMoments() {
    if (!this.globalState.currentLocation) {
      this.errorMessage = '请先获取位置信息';
      return;
    }

    this.isLoading = true;
    this.errorMessage = '';

    try {
      const response = await ApiClient.searchMomentByLocation({
        latitude: this.globalState.currentLocation.latitude,
        longitude: this.globalState.currentLocation.longitude,
        radius: 2000,
        limit: 50
      });
      
      if (response.code === 0 && response.data) {
        this.moments = response.data.items;
      } else {
        this.errorMessage = response.msg || '加载瞬间失败';
        this.moments = [];
      }
    } catch (error) {
      this.errorMessage = error instanceof Error ? error.message : '加载瞬间失败';
    } finally {
      this.isLoading = false;
    }
  }

  /**
   * 点赞瞬间
   */
  private async likeMoment(moment: SearchMomentsByLocationResponse, index: number) {
    try {
      const request: LikeMomentRequest = {
        moment_id: moment.moment_id
      };
      const response = await ApiClient.likeMoment(request);
      
      if (response.code === 0 && response.data) {
        // 更新本地状态
        this.moments[index] = {
          moment_id: moment.moment_id,
          user_id: moment.user_id,
          nickname: moment.nickname,
          user_avatar: moment.user_avatar,
          description: moment.description,
          latitude: moment.latitude,
          longitude: moment.longitude,
          location_name: moment.location_name,
          created_at: moment.created_at,
          comments_count: moment.comments_count,
          distance: moment.distance,
          tags: moment.tags,
          images: moment.images,
          liked_by_me: !moment.liked_by_me,
          likes_count: response.data.likes_count
        };
      } else {
        hilog.error(DOMAIN_NUMBER, TAG, `点赞失败: ${response.msg}`);
      }
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `点赞失败: ${JSON.stringify(error)}`);
    }
  }

  build() {
    Column() {
      // 标题栏
      Row() {
        Text('附近瞬间')
          .fontSize(20)
          .fontWeight(FontWeight.Bold)
          .fontColor($r('sys.color.ohos_id_color_text_primary'))
          .layoutWeight(1)

        Button('发布')
          .type(ButtonType.Capsule)
          .backgroundColor($r('sys.color.ohos_id_color_emphasize'))
          .fontColor(Color.White)
          .fontSize(14)
          .padding({ left: 16, right: 16, top: 8, bottom: 8 })
          .onClick(async () => {
            try {
              const routerInstance = this.getUIContext().getRouter();
              await routerInstance.pushUrl({ url: `pages/${PageNames.CREATE_MOMENT}` });
            } catch (error) {
              hilog.error(DOMAIN_NUMBER, TAG, `跳转失败: ${JSON.stringify(error)}`);
            }
          })
      }
      .width('100%')
      .padding(16)
      .backgroundColor($r('sys.color.ohos_id_color_background'))

      // 内容区域
      if (this.isLoading) {
        SystemLoadingView({ message: '正在加载附近瞬间...' })
      } else if (this.errorMessage) {
        // 内联错误视图
        Column({ space: 16 }) {
          Text('⚠️')
            .fontSize(48)
          
          Text(this.errorMessage)
            .fontSize(16)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
            .textAlign(TextAlign.Center)
          
          Button('重试')
            .type(ButtonType.Capsule)
            .backgroundColor($r('sys.color.ohos_id_color_emphasize'))
            .fontColor(Color.White)
            .onClick(() => this.loadNearbyMoments())
        }
        .width('100%')
        .height('100%')
        .justifyContent(FlexAlign.Center)
        .padding(24)
      } else if (this.moments.length === 0) {
        // 内联空状态视图
        Column({ space: 16 }) {
          Text('✨')
            .fontSize(48)
          
          Text('附近暂无瞬间\n快来分享第一个吧！')
            .fontSize(16)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
            .textAlign(TextAlign.Center)
        }
        .width('100%')
        .height('100%')
        .justifyContent(FlexAlign.Center)
      } else {
        this.buildMomentList()
      }
    }
    .width('100%')
    .height('100%')
    .backgroundColor($r('sys.color.ohos_id_color_sub_background'))
  }

  @Builder
  private buildMomentList() {
    List({ space: 12 }) {
      ForEach(this.moments, (moment: SearchMomentsByLocationResponse, index: number) => {
        ListItem() {
          this.buildMomentCard(moment, index)
        }
      }, (moment: SearchMomentsByLocationResponse) => moment.moment_id)
    }
    .width('100%')
    .layoutWeight(1)
    .padding({ left: 16, right: 16, top: 8, bottom: 16 })
    .scrollBar(BarState.Auto)
  }

  @Builder
  private buildMomentCard(moment: SearchMomentsByLocationResponse, index: number) {
    Column({ space: 12 }) {
      // 用户信息
      Row({ space: 12 }) {
        SystemAvatar({ 
          avatarUrl: moment.user_avatar || '',
          nickname: moment.nickname,
          avatarSize: 40
        })

        Column({ space: 2 }) {
          Text(moment.nickname)
            .fontSize(14)
            .fontWeight(FontWeight.Medium)
            .fontColor($r('sys.color.ohos_id_color_text_primary'))

          Text(this.formatTime(moment.created_at))
            .fontSize(12)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
        }
        .alignItems(HorizontalAlign.Start)
        .layoutWeight(1)
      }
      .width('100%')

      // 瞬间内容
      Text(moment.description)
        .fontSize(16)
        .fontColor($r('sys.color.ohos_id_color_text_primary'))
        .lineHeight(24)

      // 标签
      if (moment.tags && moment.tags.length > 0) {
        Flex({ wrap: FlexWrap.Wrap }) {
          ForEach(moment.tags, (tag: string) => {
            // 内联标签组件
            Text(`#${tag}`)
              .fontSize(12)
              .fontColor($r('sys.color.ohos_id_color_text_secondary'))
              .backgroundColor($r('sys.color.ohos_id_color_background_transparent'))
              .padding({ left: 12, right: 12, top: 6, bottom: 6 })
              .borderRadius(16)
              .border({
                width: 1,
                color: $r('sys.color.ohos_id_color_list_separator')
              })
              .margin({ right: 8, bottom: 8 })
          }, (tag: string) => tag)
        }
        .width('100%')
      }

      // 位置信息
      IntelligentLocationCard({
        latitude: moment.latitude,
        longitude: moment.longitude,
        locationName: moment.location_name,
        distance: moment.distance
      })

      // 互动区域
      Row({ space: 16 }) {
        Row({ space: 4 }) {
          Text(moment.liked_by_me ? '❤️' : '🤍')
            .fontSize(16)
            .onClick(() => this.likeMoment(moment, index))

          Text(moment.likes_count.toString())
            .fontSize(14)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
        }

        Row({ space: 4 }) {
          Text('💬')
            .fontSize(16)

          Text(moment.comments_count.toString())
            .fontSize(14)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
        }

        Blank()

        Text('分享')
          .fontSize(14)
          .fontColor($r('sys.color.ohos_id_color_text_secondary'))
      }
      .width('100%')
    }
    .width('100%')
    .padding(16)
    .backgroundColor($r('sys.color.ohos_id_color_background'))
    .borderRadius(12)
  }

  /**
   * 格式化时间
   */
  private formatTime(timeStr: string): string {
    const time = new Date(timeStr);
    const now = new Date();
    const diff = now.getTime() - time.getTime();
    
    const minutes = Math.floor(diff / (1000 * 60));
    const hours = Math.floor(diff / (1000 * 60 * 60));
    const days = Math.floor(diff / (1000 * 60 * 60 * 24));

    if (minutes < 1) return '刚刚';
    if (minutes < 60) return `${minutes}分钟前`;
    if (hours < 24) return `${hours}小时前`;
    if (days < 7) return `${days}天前`;
    
    return time.toLocaleDateString();
  }
}
