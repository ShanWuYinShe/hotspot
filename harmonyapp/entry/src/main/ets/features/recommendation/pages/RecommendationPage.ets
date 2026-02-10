/**
 * 推荐页面
 * 展示推荐内容，使用原生 List 和 Grid 布局
 */

import { ApiClient } from '../../../core/api/ApiClient';
import { FeedUserItem, TrendingTagsData, TagItem } from '../../../core/types/api';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { SystemToast } from '../../../components/SystemToast';
import { SystemLoadingView } from '../../../components/SystemLoadingView';
import { SystemAvatar } from '../../../components/SystemAvatar';
import { DesignConstants } from '../../../constants/DesignConstants';


const DOMAIN_NUMBER: number = 0xFF09;
const TAG: string = 'RecommendationPage';

@Entry
@Component
struct RecommendationPage {
  @State isLoading: boolean = true;
  @State trendingTags: TagItem[] = [];
  @State recommendedUsers: FeedUserItem[] = [];

  async aboutToAppear() {
    await this.loadData();
  }

  private async loadData() {
    this.isLoading = true;
    try {
      // 并行加载推荐数据
      const results = await Promise.all([
        ApiClient.getTrendingTags({ limit: 10, timeframe: 'weekly' }),
        ApiClient.getFeedUsers({ limit: 5 })
      ]);
      const tagsResponse = results[0];
      const usersResponse = results[1];

      if (tagsResponse.code === 0 && tagsResponse.data) {
        this.trendingTags = tagsResponse.data.items;
      }

      if (usersResponse.code === 0 && usersResponse.data) {
        this.recommendedUsers = usersResponse.data.items;
      }
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `加载推荐数据失败: ${JSON.stringify(error)}`);
      SystemToast.showError('加载失败，请重试', this.getUIContext());
    } finally {
      this.isLoading = false;
    }
  }

  build() {
    Column() {
      // 标题栏
      Row() {
        Text('推荐')
          .fontSize(DesignConstants.FONT.SIZE.XL)
          .fontWeight(FontWeight.Bold)
          .fontColor($r('sys.color.ohos_id_color_text_primary'))
          .layoutWeight(1)
          .padding({ left: DesignConstants.SPACING.LG })
      }
      .width('100%')
      .height(56)
      .backgroundColor($r('sys.color.ohos_id_color_background'))

      if (this.isLoading) {
        SystemLoadingView({ message: '加载推荐内容...' })
      } else {
        List({ space: 16 }) {
          // 热门话题
          ListItem() {
            Column({ space: 12 }) {
              Text('热门话题')
                .fontSize(DesignConstants.FONT.SIZE.LG)
                .fontWeight(FontWeight.Bold)
                .width('100%')
                .padding({ left: DesignConstants.SPACING.SM })
              
              Flex({ wrap: FlexWrap.Wrap }) {
                if (this.trendingTags.length > 0) {
                  ForEach(this.trendingTags, (tag: TagItem) => {
                    Text(`#${tag.name}`)
                      .padding(DesignConstants.SPACING.SM)
                      .backgroundColor($r('sys.color.ohos_id_color_component_normal'))
                      .borderRadius(DesignConstants.RADIUS.XL)
                      .margin({ right: DesignConstants.SPACING.SM, bottom: DesignConstants.SPACING.SM })
                  }, (tag: TagItem) => tag.name)
                } else {
                  Text('暂无热门话题')
                    .fontSize(DesignConstants.FONT.SIZE.MD)
                    .fontColor($r('sys.color.ohos_id_color_text_secondary'))
                    .padding(DesignConstants.SPACING.SM)
                }
              }
              .padding({ left: 8, right: 8 })
            }
          }
          .width('100%')
          .backgroundColor($r('sys.color.ohos_id_color_background'))
          .padding(DesignConstants.SPACING.LG)
          .borderRadius(DesignConstants.RADIUS.LG)


          // 推荐用户
          if (this.recommendedUsers.length > 0) {
            ListItemGroup({ header: this.ListHeader('推荐用户') }) {
              ForEach(this.recommendedUsers, (user: FeedUserItem) => {
                ListItem() {
                  Row() {
                    SystemAvatar({
                      avatarUrl: user.avatar_url,
                      name: user.nickname,
                      avatarSize: 40
                    })
                    .margin({ right: 12 })
                    
                    Column() {
                      Text(user.nickname)
                        .fontSize(DesignConstants.FONT.SIZE.LG)
                        .fontWeight(FontWeight.Medium)
                      Text('可能会感兴趣')
                        .fontSize(DesignConstants.FONT.SIZE.SM)
                        .fontColor($r('sys.color.ohos_id_color_text_secondary'))
                    }
                    .alignItems(HorizontalAlign.Start)
                    .layoutWeight(1)

                    Button('关注')
                      .type(ButtonType.Capsule)
                      .height(28)
                      .fontSize(DesignConstants.FONT.SIZE.SM)
                      .backgroundColor($r('sys.color.ohos_id_color_emphasize'))
                      .onClick(async () => {
                        try {
                          await ApiClient.followUser({ user_id: user.user_id });
                          SystemToast.showSuccess('已关注', this.getUIContext());
                        } catch (error) {
                          SystemToast.showError('关注失败', this.getUIContext());
                        }
                      })
                  }
                  .width('100%')
                  .padding(DesignConstants.SPACING.LG)
                  .backgroundColor($r('sys.color.ohos_id_color_background'))
                }
              }, (user: FeedUserItem) => String(user.user_id))
            }
            .borderRadius(DesignConstants.RADIUS.LG)
            .backgroundColor($r('sys.color.ohos_id_color_background'))
            .divider({ strokeWidth: 1, color: $r('sys.color.ohos_id_color_component_normal'), startMargin: 68, endMargin: 16 })
          }
        }
        .width('100%')
        .layoutWeight(1)
        .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, top: DesignConstants.SPACING.LG, bottom: DesignConstants.SPACING.LG })
        .scrollBar(BarState.Auto)
      }
    }
    .width('100%')
    .height('100%')
    .backgroundColor($r('sys.color.ohos_id_color_sub_background'))
  }

  @Builder
  ListHeader(title: string) {
    Text(title)
      .fontSize(DesignConstants.FONT.SIZE.LG)
      .fontWeight(FontWeight.Bold)
      .width('100%')
      .padding({ left: DesignConstants.SPACING.SM, bottom: DesignConstants.SPACING.SM, top: DesignConstants.SPACING.SM })
      .fontColor($r('sys.color.ohos_id_color_text_primary'))
  }
}
