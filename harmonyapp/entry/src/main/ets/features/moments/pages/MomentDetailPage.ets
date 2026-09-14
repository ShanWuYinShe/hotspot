/**
 * 动态详情页
 * 支持查看详情、点赞、收藏、分享
 */

import { ApiClient } from '../../../core/api/ApiClient';
import { PageShellContainer } from '../../../components/PageShellContainer';
import type { BreakpointType } from '../../../core/responsive/Breakpoint';
import { GlobalStateManager, KEY_CURRENT_USER } from '../../../core/state/GlobalStateManager';
import { interactionActions } from '../../../core/state/InteractionActions';
import { InteractionStateStore, MomentSocialState, isMomentLikedByReactions, sumReactionCounts } from '../../../core/state/InteractionStateStore';
import { AppRouter, EditMomentRouteParam, PhotoPreviewRouteParam } from '../../../core/router/AppRouter';
import { PageNames } from '../../../constants/AppConstants';
import {
  MomentImage,
  MomentItem,
  MomentSharePayload,
  AccountInfo,
  CommentItem,
  FetchCommentsRequest,
  CreateCommentRequest,
  UserInfo,
  Reaction,
} from '../../../core/types/api';
import { SystemAvatar } from '../../../components/SystemAvatar';
import { SystemLoadingView } from '../../../components/SystemLoadingView';
import { SystemToast } from '../../../components/SystemToast';
import { LoginGate } from '../../../core/state/LoginGate';
import { SystemPageHeader } from '../../../components/SystemPageHeader';
import { SystemDialog } from '../../../components/SystemDialog';
import { SystemButton } from '../../../components/SystemButton';
import { SystemInput } from '../../../components/SystemInput';
import { EmptyState } from '../../../components/EmptyState';
import { Pressable } from '../../../components/Pressable';
import { AppIcon, IconButton, IconSize } from '../../../components/SystemIcon';
import { DesignConstants } from '../../../constants/DesignConstants';
import { LocationInfo, LocationUtil } from '../../../core/location/LocationUtil';
import { KEY_BROWSE_ANCHOR } from '../../../core/state/GlobalStateManager';
import { SafeAreaUtils } from '../../../core/utils/SafeAreaUtils';
import { SystemUiController } from '../../../core/utils/SystemUiController';
import { ShareActionHelper } from '../../../core/utils/ShareActionHelper';
import { SystemShareService } from '../../../core/services/SystemShareService';
import { MediaUrlUtil } from '../../../core/utils/MediaUrlUtil';
import { formatCount } from '../../../core/utils/FormatUtil';
import { common } from '@kit.AbilityKit';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { KEY_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';
import { getErrorMessage } from '../../../core/network/ApiErrorUtil';
import { LucideIcon } from '../../../components/LucideIcon';

const DOMAIN_NUMBER: number = 0xFF1A;
const TAG: string = 'MomentDetailPage';

interface MomentDetailParams {
  momentId?: string;
  moment_id?: string;
}

@Preview
@Component
export struct MomentDetailPage {
  @StorageProp('currentBreakpoint') currentBreakpoint: BreakpointType = 'sm';
  pathStack: NavPathStack = new NavPathStack();
  @StorageLink('safeAreaTop') safeAreaTop: number = 0;
  @StorageLink('safeAreaBottom') safeAreaBottom: number = 0;
  @StorageLink('interaction_moment_state') momentStateMap: Record<string, MomentSocialState> = {};
  @StorageLink(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'vibrant_light';

  private stateManager = GlobalStateManager.getInstance();
  private momentId: string = '';

  @State moment: MomentItem | null = null;
  @State isLoading: boolean = true;
  @State errorMessage: string = '';
  @State likePending: boolean = false;
  /** 点赞计数弹跳（count-pop：成功瞬间 1→1.3→1，对齐前端 cst-count-pop） */
  @State likePop: boolean = false;
  /** 操作条按压键（comment/share 按压缩放反馈） */
  @State actionBarPressedKey: string = '';
  @State favoritePending: boolean = false;
  @State comments: CommentItem[] = [];
  @State commentText: string = '';
  @State commentsLoading: boolean = false;
  @State commentsError: string = '';
  @State commentSending: boolean = false;
  /** 删除确认弹窗可见性（作者视角，SystemDialog danger 二次确认） */
  @State showDeleteConfirm: boolean = false;
  /** 删除请求 in-flight 守卫（防连点重复提交，对齐发布器 isPublishing 范式） */
  @State isDeleting: boolean = false;
  /** 本地新增评论数（服务端计数刷新前叠加展示，避免手动克隆 MomentItem） */
  @State commentAdditions: number = 0;
  /** 评论点赞 in-flight 去重（防连点双翻转） */
  private commentLikePending: Set<string> = new Set<string>();
  @StorageLink(KEY_CURRENT_USER) currentUser: AccountInfo | null = null;

  private isDestroyed: boolean = false;
  /** 内容区滚动控制器（评论键点击滚动到评论区，对齐前端 focusComments） */
  private contentScroller: Scroller = new Scroller();

  aboutToAppear(): void {
    SystemUiController.syncWithCurrentTheme();
  }

  aboutToDisappear(): void {
    this.isDestroyed = true;
  }

  private getCurrentThemePalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  private getSurfaceBackground(): string {
    return this.getCurrentThemePalette().surfaceBackground;
  }

  private getSurfaceSubtle(): string {
    return this.getCurrentThemePalette().surfaceSubtle;
  }

  private getSurfaceBorder(): string {
    return this.getCurrentThemePalette().borderColor;
  }

  private formatTime(timeStr: string): string {
    const time = new Date(timeStr);
    const now = new Date();
    const hhmm = `${time.getHours().toString().padStart(2, '0')}:${time.getMinutes().toString().padStart(2, '0')}`;
    if (time.toDateString() === now.toDateString()) {
      return hhmm;
    }
    // 对齐前端格式：YYYY-MM-DD HH:mm（此前 toLocaleDateString 输出美式 8/31/2026）
    const mm = (time.getMonth() + 1).toString().padStart(2, '0');
    const dd = time.getDate().toString().padStart(2, '0');
    return `${time.getFullYear()}-${mm}-${dd} ${hhmm}`;
  }


  private calcImageGridHeight(imageCount: number): number {
    if (imageCount === 0) return 0;
    const cols = imageCount === 1 ? 1 : 2;
    const rows = Math.ceil(Math.min(imageCount, 9) / cols);
    const gap = 8;
    const itemHeight = imageCount === 1 ? 240 : 180;
    return rows * itemHeight + (rows - 1) * gap;
  }

  private async loadMomentDetail(): Promise<void> {
    if (this.isDestroyed) return;
    if (!this.momentId) {
      this.errorMessage = '缺少动态ID';
      this.isLoading = false;
      return;
    }

    this.isLoading = true;
    this.errorMessage = '';
    try {
      const response = await ApiClient.getMomentDetail({ moment_id: this.momentId });
      if (this.isDestroyed) return;
      if (response.code === 0 && response.data) {
        this.moment = response.data;
        InteractionStateStore.getInstance().patchMomentFromItem(response.data);
        this.loadComments();
      } else {
        this.errorMessage = getErrorMessage(response, '加载动态详情失败');
      }
    } catch (error) {
      this.errorMessage = '加载动态详情失败';
      hilog.error(DOMAIN_NUMBER, TAG, `loadMomentDetail failed: ${JSON.stringify(error)}`);
    } finally {
      this.isLoading = false;
    }
  }

  private async loadComments(): Promise<void> {
    if (this.isDestroyed || !this.momentId) return;
    this.commentsLoading = true;
    try {
      const request: FetchCommentsRequest = { moment_id: this.momentId, pagination: { limit: 50 } };
      const response = await ApiClient.getComments(request);
      if (this.isDestroyed) return;
      if (response.code === 0 && response.data) {
        this.comments = (response.data.items ?? []).filter((c: CommentItem) => !c.is_deleted);
      }
    } catch (_) {
      // 加载失败显式提示：空列表会掩盖网络错误（「还没评论」误导）
      this.commentsError = '评论加载失败，请重试';
    } finally {
      this.commentsLoading = false;
    }
  }

  private async sendComment(): Promise<void> {
    const content = this.commentText.trim();
    if (!content || this.commentSending) return;
    if (!this.currentUser) {
      LoginGate.requireAction('评论', this.getUIContext());
      return;
    }
    this.commentSending = true;
    const tempId = `temp-${Date.now()}`;
    const user: UserInfo = {
      user_id: this.currentUser.user_id,
      nickname: this.currentUser.nickname ?? '',
      avatar_url: this.currentUser.avatar_url ?? ''
    };
    const tempComment: CommentItem = {
      comment_id: tempId,
      user: user,
      content: content,
      parent_id: null,
      mentions: [],
      created_at: new Date().toISOString(),
      replies_count: 0,
      is_deleted: false,
      reactions: [],
      top_replies: []
    };
    this.comments = [...this.comments, tempComment];
    this.commentText = '';
    try {
      const request: CreateCommentRequest = { moment_id: this.momentId, content: content };
      const response = await ApiClient.createComment(request);
      if (this.isDestroyed) return;
      if (response.code === 0 && response.data) {
        this.comments = this.comments.map((c: CommentItem) =>
          c.comment_id === tempId ? response.data as CommentItem : c
        );
        this.commentAdditions += 1;
      } else {
        this.comments = this.comments.filter((c: CommentItem) => c.comment_id !== tempId);
        SystemToast.showError(getErrorMessage(response, '评论失败'), this.getUIContext());
      }
    } catch (_) {
      this.comments = this.comments.filter((c: CommentItem) => c.comment_id !== tempId);
      SystemToast.showError('评论失败，请稍后重试', this.getUIContext());
    } finally {
      this.commentSending = false;
    }
  }

  private async toggleLike(): Promise<void> {
    if (this.isDestroyed) return;
    if (!this.moment) return;
    if (this.likePending) return;
    this.likePending = true;
    try {
      const success = await interactionActions.toggleMomentLike(this.moment.moment_id);
      if (this.isDestroyed) return;
      if (success) {
        // count-pop：点赞成功瞬间计数放大弹跳后回落
        this.likePop = true;
        setTimeout(() => {
          this.likePop = false;
        }, DesignConstants.ANIMATION.DURATION_FAST);
      } else {
        SystemToast.showError('点赞操作失败', this.getUIContext());
      }
    } finally {
      this.likePending = false;
    }
  }

  private async toggleFavorite(): Promise<void> {
    if (this.isDestroyed) return;
    if (!this.moment) return;
    if (this.favoritePending) return;
    this.favoritePending = true;
    try {
      const success = await interactionActions.toggleMomentFavorite(this.moment.moment_id);
      if (this.isDestroyed) return;
      if (!success) {
        SystemToast.showError('收藏操作失败', this.getUIContext());
      }
    } finally {
      this.favoritePending = false;
    }
  }

  private async shareMoment(): Promise<void> {
    if (this.isDestroyed) return;
    if (!this.moment) return;
    try {
      const context = this.getUIContext().getHostContext() as common.UIAbilityContext;
      const payload: MomentSharePayload = { shared_moment_id: this.moment.moment_id };
      await ShareActionHelper.shareResource(
        context,
        'moment',
        payload,
        `${this.moment.user?.nickname ?? '匿名'} 的动态`,
        this.moment.description.length > 100 ? this.moment.description.substring(0, 100) + '...' : this.moment.description
      );
    } catch {
      // 用户取消分享
    }
  }

  private isOwner(): boolean {
    if (!this.moment || !this.currentUser) return false;
    return this.moment.user.user_id === this.currentUser.user_id;
  }

  private editMoment(): void {
    if (!this.moment) return;
    const param = new EditMomentRouteParam();
    param.moment_id = this.moment.moment_id;
    param.description = this.moment.description;
    param.tags = [...this.moment.tags];
    param.existingImageUrls = this.moment.images?.map(img => img.url) ?? [];
    param.privacy_level = 'public';
    param.location_name = this.moment.location?.location_name ?? '';
    param.latitude = this.moment.location?.point.latitude ?? 0;
    param.longitude = this.moment.location?.point.longitude ?? 0;
    AppRouter.push(PageNames.CREATE_MOMENT, param);
  }

  private previewImage(index: number): void {
    if (!this.moment || !this.moment.images || this.moment.images.length === 0) return;
    const previewParams = new PhotoPreviewRouteParam();
    previewParams.momentId = this.moment.moment_id;
    previewParams.images = this.moment.images.map((image: MomentImage) => MediaUrlUtil.normalize(image.url));
    previewParams.index = index;
    AppRouter.push(PageNames.PHOTO_PREVIEW, previewParams);
  }

  /** 评论点赞态（由 reactions[].reacted_by_me 推导，对齐前端 c.liked） */
  private isCommentLiked(comment: CommentItem): boolean {
    return (comment.reactions ?? []).some((r: Reaction) => r.reacted_by_me);
  }

  /** 评论点赞计数（reactions[].count 求和，对齐前端 c.likes） */
  private getCommentLikesCount(comment: CommentItem): number {
    return (comment.reactions ?? []).reduce((sum: number, r: Reaction) => sum + (r.count ?? 0), 0);
  }

  /**
   * 评论点赞：乐观翻转（likes 同步 ±1）+ reactions/set(target_type='comment')，失败回滚。
   * 对齐前端 toggleCommentLike（登录闸 + in-flight 去重）。
   */
  private async toggleCommentLike(commentId: string): Promise<void> {
    const target = this.comments.find((c: CommentItem) => c.comment_id === commentId);
    if (!target || this.commentLikePending.has(commentId)) return;
    if (!this.currentUser) {
      LoginGate.requireAction('点赞', this.getUIContext());
      return;
    }
    const prevLiked = this.isCommentLiked(target);
    const prevLikes = this.getCommentLikesCount(target);
    // 乐观翻转：克隆评论更新 reactions（liked + count ±1）；ArkTS 无 spread，用 concat + 显式拷贝
    this.comments = this.comments.map((c: CommentItem) => {
      if (c.comment_id !== commentId) return c;
      const reactions: Reaction[] = (c.reactions ?? []).concat();
      const mine = reactions.find((r: Reaction) => r.reacted_by_me);
      if (prevLiked) {
        if (mine) {
          mine.reacted_by_me = false;
          mine.count = Math.max(0, (mine.count ?? 1) - 1);
        }
      } else {
        if (mine) {
          mine.reacted_by_me = true;
          mine.count = (mine.count ?? 0) + 1;
        } else {
          reactions.push({ emoji: ApiClient.getDefaultLikeEmoji(), count: 1, reacted_by_me: true } as Reaction);
        }
      }
      const updated: CommentItem = {
        comment_id: c.comment_id,
        user: c.user,
        content: c.content,
        parent_id: c.parent_id,
        mentions: c.mentions,
        created_at: c.created_at,
        replies_count: c.replies_count,
        is_deleted: c.is_deleted,
        reactions: reactions,
        top_replies: c.top_replies
      };
      return updated;
    });
    this.commentLikePending.add(commentId);
    try {
      const res = await ApiClient.setCommentReaction(commentId, ApiClient.getDefaultLikeEmoji(), !prevLiked);
      if (res.code !== 0) throw new Error(res.message || '操作失败');
    } catch (err) {
      // 失败回滚
      this.comments = this.comments.map((c: CommentItem) => {
        if (c.comment_id !== commentId) return c;
        const reactions: Reaction[] = (c.reactions ?? []).concat();
        const mine = reactions.find((r: Reaction) => r.reacted_by_me);
        if (prevLiked) {
          if (mine) {
            mine.reacted_by_me = true;
            mine.count = prevLikes;
          }
        } else {
          if (mine) {
            mine.reacted_by_me = false;
            mine.count = prevLikes;
          }
        }
        const updated: CommentItem = {
          comment_id: c.comment_id,
          user: c.user,
          content: c.content,
          parent_id: c.parent_id,
          mentions: c.mentions,
          created_at: c.created_at,
          replies_count: c.replies_count,
          is_deleted: c.is_deleted,
          reactions: reactions,
          top_replies: c.top_replies
        };
        return updated;
      });
      SystemToast.showError(err instanceof Error ? err.message : '操作失败，请重试', this.getUIContext());
    } finally {
      this.commentLikePending.delete(commentId);
    }
  }

  /**
   * 评论回复：输入栏置 @对方 并滚动到评论区（对齐前端 replyTo：
   * input = `@${user} ` + commentInputEl.focus()；鸿蒙端以滚动到输入栏替代聚焦）。
   */
  private replyToComment(userName: string): void {
    this.commentText = `@${userName} `;
    this.contentScroller.scrollEdge(Edge.Bottom);
  }

  private isMomentLiked(moment: MomentItem): boolean {
    // 点赞态由 reactions[].reacted_by_me 推导（响应数据驱动，优先本地最新聚合）
    const state = this.momentStateMap[moment.moment_id];
    return isMomentLikedByReactions(state?.reactions ?? moment.reactions);
  }

  private isMomentFavorited(moment: MomentItem): boolean {
    const state = this.momentStateMap[moment.moment_id];
    if (state) {
      return state.is_favorited;
    }
    return moment.is_favorited;
  }

  private getMomentLikesCount(moment: MomentItem): number {
    const state = this.momentStateMap[moment.moment_id];
    return state ? state.likes_count : sumReactionCounts(moment.reactions);
  }

  build() {
    NavDestination() {
    PageShellContainer({ variant: 'narrow', currentBreakpoint: this.currentBreakpoint, noHorizontalPadding: true }) {
      Column() {
        this.buildTopBar()
        if (this.isLoading) {
          SystemLoadingView({ message: '正在加载动态详情...' })
        } else if (!this.moment) {
          EmptyState({
            icon: AppIcon.WARNING,
            message: this.errorMessage || '动态不存在或已被删除',
            actionLabel: '重试',
            onAction: () => { this.loadMomentDetail(); }
          })
        } else {
          Scroll(this.contentScroller) {
            Column() {
              this.buildMomentContent(this.moment as MomentItem)
              this.buildCommentsSection()
            }
            .width('100%')
          }
          .layoutWeight(1)
          .width('100%')
          .align(Alignment.TopStart)

          this.buildActionBar(this.moment as MomentItem)
          this.buildCommentInput()
        }
      }
      .width('100%')
      .height('100%')
      }
      // 删除动态二次确认（作者视角，danger 范式对齐 GroupCard 退群确认）
      SystemDialog({
        visible: this.showDeleteConfirm,
        title: '删除动态',
        description: '删除后不可恢复，评论与点赞将一并清除',
        confirmText: '删除',
        confirmVariant: 'danger',
        onCancel: () => { this.showDeleteConfirm = false; },
        onConfirm: () => { this.deleteMoment(); }
      })
    }
    .hideTitleBar(true)
    .onReady((context: NavDestinationContext) => {
      this.pathStack = context.pathStack;
      const params = context.pathInfo?.param as MomentDetailParams | undefined;
      this.momentId = params?.momentId ?? params?.moment_id ?? '';
      this.loadMomentDetail();
    })
  }

  @Builder
  private buildTopBar() {
    // 统一走 SystemPageHeader（标题 + 返回 + 作者视角更多菜单）。
    // 对齐前端 moments/[momentId] 的 more-menu（编辑动态/删除动态）：作者视角
    // 用 trailing 更多钮 + bindMenu 承载，编辑入口从 rightAction 迁移至此，
    // 使删除入口有处可挂（此前端内无删除入口，而 CreateMomentPage 编辑态文案
    // 却引导用户「删除后重新发布」，形成死胡同，审计 P2-5）。
    SystemPageHeader({
      title: '动态详情',
      showBackButton: true,
      onBack: () => { AppRouter.pop(); },
      // 对齐前端 moments/[momentId]：PageHeader titleSize=base(16) + font-medium
      titleSize: DesignConstants.FONT.SIZE.LG,
      titleWeight: DesignConstants.FONT.WEIGHT.MEDIUM,
      backgroundVariant: 'transparent',
      rightAction: null
    }) {
      if (this.isOwner()) {
        Row() {
          LucideIcon({
            icon: AppIcon.MORE,
            iconSize: DesignConstants.FONT.SIZE.MD,
            paletteRole: 'textSecondary'
          })
        }
        .width(DesignConstants.LAYOUT.ICON_BUTTON_SIZE)
        .height(DesignConstants.LAYOUT.ICON_BUTTON_SIZE)
        .justifyContent(FlexAlign.Center)
        .borderRadius(DesignConstants.RADIUS.FULL)
        .backgroundColor(this.getCurrentThemePalette().surfaceSubtle)
        .responseRegion({
          x: -(DesignConstants.LAYOUT.MIN_HIT_TARGET - DesignConstants.LAYOUT.ICON_BUTTON_SIZE) / 2,
          y: -(DesignConstants.LAYOUT.MIN_HIT_TARGET - DesignConstants.LAYOUT.ICON_BUTTON_SIZE) / 2,
          width: DesignConstants.LAYOUT.MIN_HIT_TARGET,
          height: DesignConstants.LAYOUT.MIN_HIT_TARGET
        })
        .bindMenu([
          {
            value: '编辑动态',
            action: () => { this.editMoment(); }
          },
          {
            value: '删除动态',
            action: () => { this.showDeleteConfirm = true; }
          }
        ])
      }
    }
  }

  /**
   * 删除当前动态（作者视角）：SystemDialog danger 二次确认后调
   * ApiClient.deleteMoment。成功 toast + 返回上一页（列表页在下次
   * aboutToAppear 时重拉，自然收敛；与编辑流的返回语义一致）。
   */
  private async deleteMoment(): Promise<void> {
    if (!this.moment || this.isDeleting) return;
    this.isDeleting = true;
    try {
      const resp = await ApiClient.deleteMoment({ moment_id: this.moment.moment_id });
      if (resp.code === 0) {
        SystemToast.showSuccess('动态已删除', this.getUIContext());
        AppRouter.pop();
      } else {
        SystemToast.show(resp.message || '删除失败，请重试', this.getUIContext());
      }
    } catch {
      SystemToast.show('删除失败，请重试', this.getUIContext());
    } finally {
      this.isDeleting = false;
      this.showDeleteConfirm = false;
    }
  }

  @Builder
  private buildMomentContent(moment: MomentItem) {
    Column({ space: DesignConstants.SPACING.LG }) {
      Row({ space: DesignConstants.SPACING.MD }) {
        SystemAvatar({
          avatarUrl: moment.user?.avatar_url ?? '',
          name: moment.user?.nickname ?? '未知用户',
          avatarSize: DesignConstants.COMPONENT.AVATAR_SIZE.SMALL,
          shape: 'circle',
          userInfo: moment.user
        })

        Column({ space: DesignConstants.SPACING.XS }) {
          Text(moment.user?.nickname ?? '未知用户')
            .fontSize(DesignConstants.FONT.SIZE.LG)
            .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
            .fontColor(this.getCurrentThemePalette().textPrimary)

          Row({ space: DesignConstants.SPACING.SM }) {
            Text(this.formatTime(moment.created_at))
              .fontSize(DesignConstants.FONT.SIZE.SM)
              .fontColor(this.getCurrentThemePalette().textSecondary)
            if (moment.location?.distance !== undefined && moment.location?.distance !== null) {
              Text(LocationUtil.formatDistance(moment.location.distance))
                .fontSize(DesignConstants.FONT.SIZE.SM)
                .fontColor(this.getCurrentThemePalette().textSecondary)
            }
          }
        }
        .layoutWeight(1)
        .alignItems(HorizontalAlign.Start)
      }
      .width('100%')

      Text(moment.description)
        .width('100%')
        .fontSize(DesignConstants.FONT.SIZE.LG)
        .fontColor(this.getCurrentThemePalette().textPrimary)
        .lineHeight(DesignConstants.FONT.LINE_HEIGHT.XL)

      if (moment.images && moment.images.length > 0) {
        Grid() {
          ForEach(moment.images.slice(0, 9), (img: MomentImage, imgIndex: number) => {
            GridItem() {
              Pressable({ onPress: () => {
                this.previewImage(imgIndex);
              } }) {
                Image(MediaUrlUtil.normalize(img.url))
                  .width('100%')
                  .height('100%')
                  .objectFit(ImageFit.Cover)
                  .borderRadius(DesignConstants.RADIUS.LG)
                  .backgroundColor(this.getSurfaceBackground())
              }
            }
          }, (img: MomentImage, index: number) => `${img.url}_${index}`)
        }
        .columnsTemplate(moment.images.length === 1 ? '1fr' : '1fr 1fr')
        .columnsGap(DesignConstants.SPACING.SM)
        .rowsGap(DesignConstants.SPACING.SM)
        .height(this.calcImageGridHeight(moment.images.length))
        .width('100%')
      }

      if (moment.tags && moment.tags.length > 0) {
        Flex({ wrap: FlexWrap.Wrap }) {
          ForEach(moment.tags, (tag: string, index: number) => {
            // 对齐前端 MomentPostBody：accent 实底胶囊 + on-accent 文字，
            // 浅色下淡底淡字对比不足
            Text(`#${tag}`)
              .fontSize(DesignConstants.FONT.SIZE.XS)
              .fontColor(this.getCurrentThemePalette().accentText)
              .backgroundColor(this.getCurrentThemePalette().accentBackground)
              .padding({
                left: DesignConstants.SPACING.MD,
                right: DesignConstants.SPACING.MD,
                top: DesignConstants.SPACING.XS,
                bottom: DesignConstants.SPACING.XS
              })
              .borderRadius(DesignConstants.SHAPE.CHIP)
              .margin({ right: DesignConstants.SPACING.SM, bottom: DesignConstants.SPACING.SM })
          }, (tag: string, index: number) => `${tag}_${index}`)
        }
        .width('100%')
      }

      if (moment.location?.location_name) {
        Pressable({ onPress: () => {
          if (moment.location?.point) {
            const locationInfo: LocationInfo = {
              latitude: moment.location.point.latitude,
              longitude: moment.location.point.longitude,
              locationName: moment.location.location_name ?? ''
            };
            GlobalStateManager.setBrowseAnchor(locationInfo);
            SystemToast.showSuccess('已设为浏览锚点', this.getUIContext());
          }
        } }) {
          Row({ space: DesignConstants.SPACING.XS }) {
            LucideIcon({
                icon: AppIcon.LOCATION,
                iconSize: DesignConstants.FONT.SIZE.SM,
                color: this.getCurrentThemePalette().textSecondary
              })
            Text(moment.location.location_name)
              .fontSize(DesignConstants.FONT.SIZE.SM)
              .fontColor(this.getCurrentThemePalette().textSecondary)
              .maxLines(1)
              .textOverflow({ overflow: TextOverflow.Ellipsis })
          }
          .padding({ top: DesignConstants.SPACING.XS, bottom: DesignConstants.SPACING.XS })
        }
      }

      // 互动计数（点赞/评论/分享）不再在正文区重复展示：底部操作栏已带同值
      // 计数（更醒目且可直接操作），保留一处即可。
    }
    .width('100%')
    .padding(DesignConstants.SPACING.LG)
    .alignItems(HorizontalAlign.Start)
  }

  private getCommentCount(moment: MomentItem): number {
    return (moment.comments_count ?? 0) + this.commentAdditions;
  }

  @Builder
  private buildCommentRow(comment: CommentItem) {
    Row({ space: DesignConstants.SPACING.MD }) {
      SystemAvatar({
        avatarUrl: comment.user?.avatar_url ?? '',
        name: comment.user?.nickname ?? '匿名',
        avatarSize: DesignConstants.COMPONENT.AVATAR_SIZE.XS,
        shape: 'circle'
      })
      Column({ space: DesignConstants.SPACING.XS }) {
        Row({ space: DesignConstants.SPACING.SM }) {
          Text(comment.user?.nickname ?? '匿名')
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
            .fontColor(this.getCurrentThemePalette().textPrimary)
          Text(this.formatTime(comment.created_at))
            .fontSize(DesignConstants.FONT.SIZE.XS)
            .fontColor(this.getCurrentThemePalette().textSecondary)
        }
        .width('100%')
        Text(comment.content)
          .fontSize(DesignConstants.FONT.SIZE.MD)
          .fontColor(this.getCurrentThemePalette().textPrimary)
          .width('100%')
          .textAlign(TextAlign.Start)

        // 评论点赞 + 回复（对齐前端评论行：点赞 Heart+计数 / 回复按钮）
        Row({ space: DesignConstants.SPACING.MD }) {
          Row({ space: DesignConstants.SPACING.XS }) {
            LucideIcon({
              icon: this.isCommentLiked(comment) ? AppIcon.HEART_FILL : AppIcon.HEART,
              iconSize: DesignConstants.FONT.SIZE.SM,
              color: this.isCommentLiked(comment) ? this.getCurrentThemePalette().accentBackground : this.getCurrentThemePalette().textSecondary
            })
            Text(formatCount(this.getCommentLikesCount(comment)))
              .fontSize(DesignConstants.FONT.SIZE.XS)
              .fontColor(this.isCommentLiked(comment) ? this.getCurrentThemePalette().accentBackground : this.getCurrentThemePalette().textSecondary)
          }
          .padding({ left: DesignConstants.SPACING.XS, right: DesignConstants.SPACING.XS, top: DesignConstants.SPACING.XS, bottom: DesignConstants.SPACING.XS })
          .scale(this.commentLikePending.has(comment.comment_id) ?
            { x: DesignConstants.INTERACTION.SCALE_PRESSED_ICON, y: DesignConstants.INTERACTION.SCALE_PRESSED_ICON } :
            { x: 1, y: 1 })
          .animation({ duration: DesignConstants.ANIMATION.DURATION_FAST, curve: Curve.EaseOut })
          .onClick(() => { this.toggleCommentLike(comment.comment_id); })
          .accessibilityText(this.isCommentLiked(comment) ? '取消点赞评论' : '点赞评论')

          Text('回复')
            .fontSize(DesignConstants.FONT.SIZE.XS)
            .fontColor(this.getCurrentThemePalette().textSecondary)
            .padding({
              left: DesignConstants.SPACING.SM,
              right: DesignConstants.SPACING.SM,
              top: DesignConstants.SPACING.SM2,
              bottom: DesignConstants.SPACING.SM2
            })
            .onClick(() => { this.replyToComment(comment.user?.nickname ?? ''); })
            .accessibilityText('回复评论')
            .responseRegion({
              x: 0,
              y: -(DesignConstants.LAYOUT.MIN_HIT_TARGET - DesignConstants.COMPONENT.ACTION_BUTTON_HEIGHT) / 2,
              width: '100%',
              height: DesignConstants.LAYOUT.MIN_HIT_TARGET
            })
        }
        .alignItems(VerticalAlign.Center)
      }
      .alignItems(HorizontalAlign.Start)
      .layoutWeight(1)
    }
    .width('100%')
    .alignItems(VerticalAlign.Top)
  }

  @Builder
  private buildCommentsSection() {
    Column({ space: DesignConstants.SPACING.MD }) {
      Text(`评论 ${formatCount(this.getCommentCount(this.moment as MomentItem))}`)
        .fontSize(DesignConstants.FONT.SIZE.SM)
        .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
        .fontColor(this.getCurrentThemePalette().textPrimary)
        .width('100%')

      if (this.commentsLoading && this.comments.length === 0) {
        Text('加载评论中…')
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .fontColor(this.getCurrentThemePalette().textSecondary)
          .width('100%')
      } else if (this.commentsError.length > 0) {
        // 错误态与空态区分：失败可重试（点击错误行重新拉取）
        Text(`${this.commentsError}（点击重试）`)
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .fontColor(this.getCurrentThemePalette().error)
          .width('100%')
          .padding(DesignConstants.SPACING.SM)
          .onClick(() => { this.loadComments(); })
      } else if (this.comments.length === 0) {
        Text('还没有评论，来抢沙发')
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .fontColor(this.getCurrentThemePalette().textSecondary)
          .width('100%')
      } else {
        ForEach(this.comments, (comment: CommentItem) => {
          this.buildCommentRow(comment)
        }, (comment: CommentItem) => comment.comment_id)
      }
    }
    .width('100%')
    .padding(DesignConstants.SPACING.LG)
    .alignItems(HorizontalAlign.Start)
  }

  @Builder
  private buildCommentInput() {
    Row({ space: DesignConstants.SPACING.SM }) {
      SystemInput({
        value: this.commentText,
        placeholder: '写下你的评论…',
        showPrefixIcon: false,
        inputHeight: DesignConstants.COMPONENT.INPUT_HEIGHT,
        onChange: (value: string) => {
          this.commentText = value;
        }
      })
        .layoutWeight(1)

      IconButton({
        icon: AppIcon.SEND,
        iconSize: IconSize.SMALL,
        bgColor: this.commentText.trim().length > 0 ? this.getCurrentThemePalette().accentBackground : this.getSurfaceSubtle(),
        iconColor: this.commentText.trim().length > 0 ? this.getCurrentThemePalette().accentText : this.getCurrentThemePalette().textSecondary,
        radius: DesignConstants.RADIUS.FULL,
        minButtonSize: DesignConstants.LAYOUT.ICON_BUTTON_SIZE,
        onButtonClick: () => { this.sendComment(); }
      })
        .enabled(this.commentText.trim().length > 0 && !this.commentSending)
        .opacity(this.commentText.trim().length === 0 || this.commentSending ?
        DesignConstants.OPACITY.DISABLED : 1)
    }
    .width('100%')
    .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, bottom: DesignConstants.SPACING.SM })
  }

  @Builder
  /**
   * 移动端互动操作条（对齐前端 moments/[momentId] footer：四键图标 + 计数，
   * 44pt 拇指热区等分；图标与计数水平并排（同动态卡片操作行），保证四键
   * 图标基线一致；点赞/评论带计数，收藏/分享纯图标；激活态 accent 色）。
   */
  private buildActionBar(moment: MomentItem) {
    Row() {
      // 点赞（Heart + 计数，liked 时 accent）
      Row({ space: DesignConstants.SPACING.XS }) {
        LucideIcon({
          icon: this.isMomentLiked(moment) ? AppIcon.HEART_FILL : AppIcon.HEART,
          iconSize: DesignConstants.FONT.SIZE.LG,
          color: this.isMomentLiked(moment) ? this.getCurrentThemePalette().accentBackground : this.getCurrentThemePalette().textSecondary
        })
        Text(formatCount(this.getMomentLikesCount(moment)))
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .fontColor(this.isMomentLiked(moment) ? this.getCurrentThemePalette().accentBackground : this.getCurrentThemePalette().textSecondary)
          .scale(this.likePop ?
            { x: DesignConstants.INTERACTION.SCALE_LIKE_POP, y: DesignConstants.INTERACTION.SCALE_LIKE_POP } :
            { x: 1, y: 1 })
          .animation({
            duration: DesignConstants.ANIMATION.DURATION_FAST,
            curve: DesignConstants.ANIMATION.CURVE_EASE_OUT
          })
      }
      .layoutWeight(1)
      .justifyContent(FlexAlign.Center)
      .scale(this.likePending ? { x: DesignConstants.INTERACTION.SCALE_PRESSED_ICON, y: DesignConstants.INTERACTION.SCALE_PRESSED_ICON } : { x: 1, y: 1 })
      .animation({ duration: DesignConstants.ANIMATION.DURATION_FAST, curve: Curve.EaseOut })
      .onClick(() => { this.toggleLike(); })
      .accessibilityText(this.isMomentLiked(moment) ? '取消点赞' : '点赞')

      // 评论（MessageCircle + 计数，点击滚动到评论区并聚焦输入栏）
      Row({ space: DesignConstants.SPACING.XS }) {
        LucideIcon({
          icon: AppIcon.MESSAGE,
          iconSize: DesignConstants.FONT.SIZE.LG,
          color: this.getCurrentThemePalette().textSecondary
        })
        Text(formatCount(this.getCommentCount(moment)))
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .fontColor(this.getCurrentThemePalette().textSecondary)
      }
      .layoutWeight(1)
      .justifyContent(FlexAlign.Center)
      .onClick(() => { this.contentScroller.scrollEdge(Edge.Bottom); })
      .accessibilityText('查看评论')
      .onTouch((event: TouchEvent) => {
        if (event.type === TouchType.Down) {
          this.actionBarPressedKey = 'comment';
        } else if (event.type === TouchType.Up || event.type === TouchType.Cancel) {
          this.actionBarPressedKey = '';
        }
      })
      .scale(this.actionBarPressedKey === 'comment' ?
        { x: DesignConstants.INTERACTION.SCALE_PRESSED_ICON, y: DesignConstants.INTERACTION.SCALE_PRESSED_ICON } :
        { x: 1, y: 1 })
      .animation({ duration: DesignConstants.ANIMATION.DURATION_FAST, curve: Curve.EaseOut })

      // 收藏（Bookmark，saved 时 accent）
      Row() {
        LucideIcon({
          icon: this.isMomentFavorited(moment) ? AppIcon.STAR_FILL : AppIcon.STAR,
          iconSize: DesignConstants.FONT.SIZE.LG,
          color: this.isMomentFavorited(moment) ? this.getCurrentThemePalette().accentBackground : this.getCurrentThemePalette().textSecondary
        })
      }
      .layoutWeight(1)
      .justifyContent(FlexAlign.Center)
      .scale(this.favoritePending ? { x: DesignConstants.INTERACTION.SCALE_PRESSED_ICON, y: DesignConstants.INTERACTION.SCALE_PRESSED_ICON } : { x: 1, y: 1 })
      .animation({ duration: DesignConstants.ANIMATION.DURATION_FAST, curve: Curve.EaseOut })
      .onClick(() => { this.toggleFavorite(); })
      .accessibilityText(this.isMomentFavorited(moment) ? '取消收藏' : '收藏')

      // 分享（Share）
      Row() {
        LucideIcon({
          icon: AppIcon.SHARE,
          iconSize: DesignConstants.FONT.SIZE.LG,
          color: this.getCurrentThemePalette().textSecondary
        })
      }
      .layoutWeight(1)
      .justifyContent(FlexAlign.Center)
      .onClick(() => { this.shareMoment(); })
      .onTouch((event: TouchEvent) => {
        if (event.type === TouchType.Down) {
          this.actionBarPressedKey = 'share';
        } else if (event.type === TouchType.Up || event.type === TouchType.Cancel) {
          this.actionBarPressedKey = '';
        }
      })
      .scale(this.actionBarPressedKey === 'share' ?
        { x: DesignConstants.INTERACTION.SCALE_PRESSED_ICON, y: DesignConstants.INTERACTION.SCALE_PRESSED_ICON } :
        { x: 1, y: 1 })
      .animation({ duration: DesignConstants.ANIMATION.DURATION_FAST, curve: Curve.EaseOut })
      .accessibilityText('分享动态')
    }
    .width('100%')
    .height(DesignConstants.LAYOUT.MOBILE_BOTTOM_NAV_HEIGHT)
    .alignItems(VerticalAlign.Center)
    .padding({
      left: DesignConstants.SPACING.LG,
      right: DesignConstants.SPACING.LG
    })
    .backgroundColor(this.getSurfaceBackground())
    .border({ width: { top: DesignConstants.BORDER_WIDTH.HAIRLINE }, color: this.getSurfaceBorder() })
  }
}

@Builder
export function MomentDetailPageBuilder() {
  MomentDetailPage()
}