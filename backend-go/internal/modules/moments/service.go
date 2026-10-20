package moments

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"hotspot/backend-go/internal/modules/comments"
	"hotspot/backend-go/internal/modules/notify"
	"hotspot/backend-go/internal/platform/actor"
	"hotspot/backend-go/internal/platform/appconfig"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/observability"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/realtime"
	"hotspot/backend-go/internal/platform/resourceguard"
	"hotspot/backend-go/internal/platform/textutil"
	"hotspot/backend-go/internal/platform/types"

	"golang.org/x/sync/errgroup"
)

type Service struct {
	repo          Repository
	notifications momentNotificationWriter
	users         actor.Provider
	images        ImageResolver
	appConfig     appconfig.Source
	runInTx       persistence.TxRunner
	reactions     ReactionAggregator   // optional; nil => reactions[] not filled
	relations     FollowRelationReader // optional; nil => author_relation stays null
	locations     LocationProvider     // optional; nil => reference_point 回退链跳过用户最近位置一级
	// blockChecker 可选注入：caller 与瞬间作者间双向拉黑判定（对齐 comments 的
	// 依赖倒置模式，bootstrap 统一接线）。nil（dev/test 默认）时跳过拉黑检查。
	blockChecker func(callerID, authorID string) (bool, error)
	// 附近新瞬间通知（可选注入，bootstrap 接线）：目标查询（users 实现）+
	// 通知发布（notify 实现）。nil（未接线）时 Create 静默跳过附近通知。
	nearbyNotifyTargets   NearbyNotifyTargetsFn
	nearbyNotifyPublisher NearbyNotifyPublisher
}

// SetNearbyNotifyTargetsReader 注入附近通知目标查询（users 模块实现）。
func (s *Service) SetNearbyNotifyTargetsReader(fn NearbyNotifyTargetsFn) { s.nearbyNotifyTargets = fn }

// SetNearbyNotifyPublisher 注入附近通知发布器（notify 模块实现）。
func (s *Service) SetNearbyNotifyPublisher(p NearbyNotifyPublisher) { s.nearbyNotifyPublisher = p }

// ReactionAggregator computes per-emoji reaction aggregates for moments.
// Implemented by the reactions module (dependency inversion: moments never
// imports reactions). Backs MomentItem.reactions[]. The viewerID drives the
// @auth-only reacted_by_me flag (empty for anonymous).
type ReactionAggregator interface {
	AggregateMoments(ctx context.Context, momentIDs []string, viewerID string) (map[string][]Reaction, error)
}

// FollowRelationReader 批量读取「查看者 → 作者」的双向关注关系。
// 由 bootstrap 注入（与 ReactionAggregator 同模式，打破模块循环）；
// 未注入时 author_relation 恒 null（例如内存 repo 的单元测试）。
type FollowRelationReader interface {
	AuthorRelations(ctx context.Context, viewerID string, authorIDs []string) (map[string]UserRelationView, error)
}

type momentNotificationWriter interface {
	AppendMentionedBatch(ctx context.Context, targetUserIDs []string, actor types.UserInfo, targetType string, momentID string, momentSummary types.MomentSummary, commentID string, commentPreview string) (map[string]string, error)
}

func NewService() *Service {
	return NewServiceWithDepsAndImagesAndConfig(NewInMemoryRepository(), notify.NewService(), nil, nil, nil, nil)
}

func NewServiceWithRepo(repo Repository) *Service {
	return NewServiceWithDepsAndImagesAndConfig(repo, notify.NewService(), nil, nil, nil, nil)
}

func NewServiceWithDeps(repo Repository, notificationWriter momentNotificationWriter, actorProvider actor.Provider) *Service {
	return NewServiceWithDepsAndImagesAndConfig(repo, notificationWriter, actorProvider, nil, nil, nil)
}

func NewServiceWithDepsAndImages(repo Repository, notificationWriter momentNotificationWriter, actorProvider actor.Provider, images ImageResolver) *Service {
	return NewServiceWithDepsAndImagesAndConfig(repo, notificationWriter, actorProvider, images, nil, nil)
}

// NewServiceWithDepsAndImagesAndConfig wires all dependencies including app
// config for moment image limits and a transaction runner for atomic updates.
func NewServiceWithDepsAndImagesAndConfig(repo Repository, notificationWriter momentNotificationWriter, actorProvider actor.Provider, images ImageResolver, cfg appconfig.Source, runInTx persistence.TxRunner) *Service {
	return &Service{repo: repo, notifications: notificationWriter, users: actorProvider, images: images, appConfig: cfg, runInTx: runInTx}
}

// defaultMomentDescriptionMaxRunes 与 ClientConfig.moment_description_max_length
// 默认值一致；仅当 AppConfig 不可用或未配置时作为兜底。
const defaultMomentDescriptionMaxRunes = 2000

// ensureDescriptionLength 按 ClientConfig.moment_description_max_length（rune 数）
// 校验动态正文；Create 与 Update 共用，保证两端点执行口径一致。
func (s *Service) ensureDescriptionLength(ctx context.Context, description string) error {
	max := defaultMomentDescriptionMaxRunes
	if s.appConfig != nil {
		if n := s.appConfig.Get(ctx).Limits.MomentDescriptionMaxLength; n > 0 {
			max = n
		}
	}
	if utf8.RuneCountInString(description) > max {
		return ErrDescriptionTooLong
	}
	return nil
}

// mentionsMaxCount 契约级上限（api_spec_base.md 字段校验约束：mentions 最多 20 个）。
// 不限制会放大通知写入：每个 mention 触发一次通知落库 + 一次 WS/Redis 推送。
const mentionsMaxCount = 20

// ensureMentionsLimit 校验 mentions 数量上限；Create 与 Update 共用。
func ensureMentionsLimit(mentions []string) error {
	if len(mentions) > mentionsMaxCount {
		return ErrTooManyMentions
	}
	return nil
}

// tagsMaxCount 契约级上限（api_spec_base.md 字段校验约束：tags 默认最多 10 个）。
// tags 逐条 INSERT，不限制会让单条请求放大为数千次写。
const tagsMaxCount = 10

// ensureTagsLimit 校验 tags 数量上限；Create 与 Update 共用。
func ensureTagsLimit(tags []string) error {
	if len(tags) > tagsMaxCount {
		return ErrTooManyTags
	}
	return nil
}

func (s *Service) Create(ctx context.Context, req CreateMomentRequest) (OwnerMomentItem, error) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	// distance/bearing_degrees 是响应侧相对位置字段，由服务端基于 reference_point
	// 计算（spec api_spec_types.md「位置与地理信息」）；请求侧 location 仅承载
	// point/location_name，剥离客户端可能传入的相对字段避免原样回显。
	if req.Location != nil {
		req.Location.Distance = nil
		req.Location.BearingDegrees = nil
	}
	if err := s.ensureDescriptionLength(ctx, req.Description); err != nil {
		return OwnerMomentItem{}, err
	}
	if err := ensureMentionsLimit(req.Mentions); err != nil {
		return OwnerMomentItem{}, err
	}
	if err := ensureTagsLimit(req.Tags); err != nil {
		return OwnerMomentItem{}, err
	}
	if err := s.validateImageOwnership(ctx, principal.UserID, req.ImageFileIDs); err != nil {
		return OwnerMomentItem{}, err
	}
	item, err := s.repo.Create(ctx, principal.UserID, req)
	if err != nil {
		observability.L().Error("moments.create failed", "error", err)
		return OwnerMomentItem{}, fmt.Errorf("moments.Service.Create: repo: %w", err)
	}
	resourceguard.RecordMoment(ctx, item.MomentID)
	s.notifyMomentMentions(ctx, principal.UserID, item)
	// 附近新瞬间通知（spec api_spec_websocket.md NearbyNewMoment）：带位置的
	// 公开瞬间创建后，向开启 nearby_moment_notify 的附近用户推送。无位置或
	// 未接线时跳过；目标查询/发布失败不阻断创建主流程（尽力而为）。
	if req.Location != nil && (item.PrivacyLevel == "" || item.PrivacyLevel == "public") {
		s.notifyNearbyNewMoment(ctx, principal.UserID, item.MomentID,
			req.Location.Point.Latitude, req.Location.Point.Longitude,
			item.User, types.MomentSummary{
				MomentID:           item.MomentID,
				DescriptionPreview: summarizeMomentDescription(item.Description),
			})
	}
	s.hydrateImages(ctx, &item)
	return toOwnerMoment(item), nil
}

func (s *Service) OpenTheDoor(ctx context.Context, momentID string) (MomentItem, bool, error) {
	item, ok, err := s.repo.Get(ctx, momentID)
	if err != nil || !ok {
		return MomentItem{}, false, err
	}
	// 软删瞬间一律按不存在处理，含作者本人（spec api_spec_moment.md「删除
	// 语义」：软删占位只出现在 fetch_created/fetch_favorites，open_the_door
	// 对所有调用方按不存在处理）。
	if item.IsDeleted {
		return MomentItem{}, false, nil
	}
	principal, _ := platformauth.PrincipalFromContext(ctx)
	isOwner := principal.UserID != "" && item.User.UserID == principal.UserID
	if item.PrivacyLevel == "private" && !isOwner {
		return MomentItem{}, false, nil
	}
	if principal.UserID != "" {
		if st := s.repo.LoadMyState(ctx, momentID, principal.UserID); st != nil {
			item.IsFavorited = st.IsFavorited
		}
	}
	s.hydrateImages(ctx, &item)
	s.fillReactions(ctx, &item, principal.UserID)
	s.fillAuthorRelation(ctx, &item, principal.UserID)
	return item, true, nil
}

func (s *Service) NotificationContext(ctx context.Context, momentID string) (comments.MomentNotificationContext, bool, error) {
	item, ok, err := s.repo.Get(ctx, momentID)
	if err != nil || !ok {
		return comments.MomentNotificationContext{}, ok, err
	}
	return comments.MomentNotificationContext{
		UserID: item.User.UserID,
		Summary: types.MomentSummary{
			MomentID:           item.MomentID,
			DescriptionPreview: summarizeMomentDescription(item.Description),
		},
	}, true, nil
}

func (s *Service) ListCreated(ctx context.Context, req ListCreatedMomentsRequest) (PaginatedOwnerMoments, error) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	limit := 20
	cursor := ""
	if req.Pagination != nil {
		if req.Pagination.Limit > 0 {
			limit = req.Pagination.Limit
		}
		if limit > types.MaxPageSize {
			limit = types.MaxPageSize
		}
		cursor = req.Pagination.Cursor
	}
	result, err := s.repo.ListMine(ctx, principal.UserID, limit, cursor)
	if err != nil {
		return PaginatedOwnerMoments{}, err
	}
	items := make([]OwnerMomentItem, 0, len(result.Items))
	// 图片批量 hydrate（单次查询，与 feed 路径一致），再转拥有者视角。
	s.hydrateImagesBatch(ctx, result.Items)
	for i := range result.Items {
		item := result.Items[i]
		ownerItem := toOwnerMoment(item)
		// 恒定数组形状：images/tags/mentions 为空时序列化 [] 而非 null（spec 恒定形状，
		// 与 feed 的 ensureMomentArraysBatch 对齐；OwnerMomentItem 此前缺失该步骤，
		// 无图动态下发 images:null 会被客户端渲染成空 src 图片）。
		ensureOwnerMomentArrays(&ownerItem)
		items = append(items, ownerItem)
	}
	s.fillReactionsBatchOwnerMoment(ctx, items, principal.UserID)
	return PaginatedOwnerMoments{Items: items, NextCursor: result.NextCursor}, nil
}

func (s *Service) Exists(ctx context.Context, momentID string) (bool, error) {
	return s.repo.Exists(ctx, momentID)
}

// SetBlockChecker 注入 caller↔作者双向拉黑判定（依赖倒置：moments 不直接
// import users，由 bootstrap 接线；查询出错 fail-closed 向上返回）。
func (s *Service) SetBlockChecker(fn func(callerID, authorID string) (bool, error)) {
	s.blockChecker = fn
}

// blockedEitherWay caller 与作者任一方拉黑即视为阻断（feed/交互全链路口径对齐
// 私聊与 WS presence）。返回 (是否阻断, 错误)。
func (s *Service) blockedEitherWay(callerID, authorID string) (bool, error) {
	if s.blockChecker == nil || callerID == "" || authorID == "" || callerID == authorID {
		return false, nil
	}
	return s.blockChecker(callerID, authorID)
}

// IsReactionTargetVisible 判定瞬间对当前调用方可见（spec reaction.md 目标可见性）：
// 软删瞬间与 private 瞬间（非拥有者）统一按不存在处理（found=false → 10300，
// 不因报错路径差异泄漏存在性）；双向拉黑按不可见处理（reaction 不得击穿拉黑
// 关系触发对方通知）。
func (s *Service) IsReactionTargetVisible(ctx context.Context, momentID string) (found bool, visible bool, err error) {
	item, ok, err := s.repo.Get(ctx, momentID)
	if err != nil || !ok {
		return ok, false, err
	}
	if item.IsDeleted {
		return false, false, nil
	}
	principal, _ := platformauth.PrincipalFromContext(ctx)
	if item.PrivacyLevel == "private" && item.User.UserID != principal.UserID {
		return true, false, nil
	}
	blocked, err := s.blockedEitherWay(principal.UserID, item.User.UserID)
	if err != nil {
		return true, false, err
	}
	if blocked {
		return true, false, nil
	}
	return true, true, nil
}

func (s *Service) CheckCommentable(ctx context.Context, momentID string) (found bool, canComment bool, err error) {
	item, ok, err := s.repo.Get(ctx, momentID)
	if err != nil || !ok {
		return ok, false, err
	}
	// 软删瞬间按不存在处理（spec api_spec_moment.md：评论挂载按不存在处理；
	// 对齐 IsReactionTargetVisible 的 IsDeleted 口径）。
	if item.IsDeleted {
		return false, false, nil
	}
	principal, _ := platformauth.PrincipalFromContext(ctx)
	// 私有瞬间非拥有者按不存在处理（spec 防枚举：private 内容不暴露存在性 → 10300）。
	if item.PrivacyLevel == "private" && item.User.UserID != principal.UserID {
		return false, false, nil
	}
	// 可预留在 moments 侧表达「作者限制评论」（如仅好友可评论）→ canComment=false
	// 由 comments.create 映射为 10200 + privacy_setting。
	return true, true, nil
}

func (s *Service) IncrementCommentCount(ctx context.Context, momentID string) error {
	return s.repo.IncrementCommentCount(ctx, momentID)
}

func (s *Service) DecrementCommentCount(ctx context.Context, momentID string) error {
	return s.repo.DecrementCommentCount(ctx, momentID)
}

func (s *Service) IncrementSharesCount(ctx context.Context, momentID string) error {
	return s.repo.IncrementSharesCount(ctx, momentID)
}

func (s *Service) Search(ctx context.Context, req SearchMomentsRequest) (PaginatedMoments, error) {
	result, err := s.repo.Search(ctx, req, viewerUserID(ctx))
	if err != nil {
		return result, err
	}
	s.hydrateImagesBatch(ctx, result.Items)
	s.fillReactionsBatch(ctx, result.Items, viewerUserID(ctx))
	s.fillAuthorRelations(ctx, result.Items, viewerUserID(ctx))
	return result, nil
}

func (s *Service) Update(ctx context.Context, req UpdateMomentRequest) (OwnerMomentItem, error) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	item, ok, err := s.repo.Get(ctx, req.MomentID)
	if !ok {
		if err != nil {
			return OwnerMomentItem{}, err
		}
		return OwnerMomentItem{}, ErrMomentNotFound
	}
	if item.User.UserID != principal.UserID {
		return OwnerMomentItem{}, ErrPermissionDenied
	}
	// 同 Create：请求侧 location 剥离服务端计算的相对位置字段（spec api_spec_types.md）。
	if req.Location != nil {
		req.Location.Distance = nil
		req.Location.BearingDegrees = nil
	}
	if req.Description != nil {
		if err := s.ensureDescriptionLength(ctx, *req.Description); err != nil {
			return OwnerMomentItem{}, err
		}
	}
	// mentions 是 patch 语义：nil=未传保持不变；显式传数组（含空数组=清空）才校验上限。
	if req.Mentions != nil {
		if err := ensureMentionsLimit(req.Mentions); err != nil {
			return OwnerMomentItem{}, err
		}
	}
	if req.Tags != nil {
		if err := ensureTagsLimit(req.Tags); err != nil {
			return OwnerMomentItem{}, err
		}
	}
	if err := s.validateImageOwnership(ctx, principal.UserID, req.ImageFileIDs); err != nil {
		return OwnerMomentItem{}, err
	}
	updateFn := func(ctx context.Context) error {
		return s.repo.Update(ctx, principal.UserID, req)
	}
	if s.runInTx != nil {
		if err := s.runInTx(ctx, updateFn); err != nil {
			observability.L().Error("moments.update failed", "error", err, "moment_id", req.MomentID)
			return OwnerMomentItem{}, fmt.Errorf("moments.Service.Update: tx: %w", err)
		}
	} else {
		if err := updateFn(ctx); err != nil {
			observability.L().Error("moments.update failed", "error", err, "moment_id", req.MomentID)
			return OwnerMomentItem{}, fmt.Errorf("moments.Service.Update: repo: %w", err)
		}
	}
	updated, _, err := s.repo.Get(ctx, req.MomentID)
	if err != nil {
		observability.L().Error("moments.update reload failed", "error", err, "moment_id", req.MomentID)
		return OwnerMomentItem{}, err
	}
	// 仅对本次编辑新增的 mention 通知：编辑前已存在的 mention 不重复推送。
	previousMentions := make(map[string]struct{}, len(item.Mentions))
	for _, mentionedID := range normalizeMomentMentions(item.Mentions) {
		previousMentions[mentionedID] = struct{}{}
	}
	newlyMentioned := make([]string, 0)
	for _, mentionedID := range normalizeMomentMentions(updated.Mentions) {
		if _, seen := previousMentions[mentionedID]; !seen {
			newlyMentioned = append(newlyMentioned, mentionedID)
		}
	}
	s.notifyMentionUserIDs(ctx, principal.UserID, updated, newlyMentioned)
	s.hydrateImages(ctx, &updated)
	// 瞬间被编辑 → 推送给作者的其他在线设备（多端同步，spec api_spec_websocket.md）
	realtime.DefaultHub().BroadcastToUser(principal.UserID, realtime.Event{Type: realtime.EventTypeMomentUpdated, Payload: MomentUpdatedPayload{
		MomentID:  req.MomentID,
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	}})
	return toOwnerMoment(updated), nil
}

func (s *Service) Delete(ctx context.Context, momentID string) error {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	item, ok, err := s.repo.Get(ctx, momentID)
	if !ok {
		if err != nil {
			return err
		}
		return ErrMomentNotFound
	}
	if item.User.UserID != principal.UserID {
		return ErrPermissionDenied
	}
	if item.IsDeleted {
		// 重复删除（已软删）视为 no-op 成功
		return nil
	}
	if err := s.repo.Delete(ctx, momentID); err != nil {
		return fmt.Errorf("moments.Service.Delete: repo: %w", err)
	}
	// 软删后推送 MomentDeleted（spec api_spec_websocket.md）：作者其他在线设备 + 收藏者
	// 据此将本地列表/收藏更新为"已删除"占位。
	now := time.Now().UTC().Format(time.RFC3339)
	realtime.DefaultHub().BroadcastToUser(principal.UserID, realtime.Event{Type: realtime.EventTypeMomentDeleted, Payload: MomentDeletedPayload{
		MomentID:  momentID,
		DeletedAt: now,
	}})
	if err := s.broadcastMomentDeletedToFavorites(ctx, momentID, principal.UserID, now); err != nil {
		observability.L().Warn("moments.delete favorite broadcast failed", "error", err, "moment_id", momentID)
	}
	return nil
}

// ensureInteractable 校验瞬间对当前用户可交互：必须存在、未软删且可见
// （public 或拥有者），且与作者无双向拉黑。不可见统一 ErrMomentNotFound
// （10300 防枚举）；拉黑查询失败 fail-closed 向上返回。
func (s *Service) ensureInteractable(ctx context.Context, momentID string) error {
	item, ok, err := s.repo.Get(ctx, momentID)
	if err != nil {
		return err
	}
	if !ok || item.IsDeleted {
		return ErrMomentNotFound
	}
	principal, _ := platformauth.PrincipalFromContext(ctx)
	if item.PrivacyLevel == "private" && item.User.UserID != principal.UserID {
		return ErrMomentNotFound
	}
	blocked, err := s.blockedEitherWay(principal.UserID, item.User.UserID)
	if err != nil {
		return err
	}
	if blocked {
		return ErrMomentNotFound
	}
	return nil
}

func (s *Service) Favorite(ctx context.Context, momentID string) error {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	if err := s.ensureInteractable(ctx, momentID); err != nil {
		observability.L().Warn("moments.favorite target not interactable", "moment_id", momentID)
		return err
	}
	return s.repo.Favorite(ctx, principal.UserID, momentID)
}

func (s *Service) Unfavorite(ctx context.Context, momentID string) error {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	if err := s.ensureInteractable(ctx, momentID); err != nil {
		observability.L().Warn("moments.unfavorite target not interactable", "moment_id", momentID)
		return err
	}
	return s.repo.Unfavorite(ctx, principal.UserID, momentID)
}

func (s *Service) ListFavorites(ctx context.Context, req MyFavoritesRequest) (PaginatedMoments, error) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	limit := 20
	if req.Pagination != nil && req.Pagination.Limit > 0 {
		limit = req.Pagination.Limit
	}
	if limit > types.MaxPageSize {
		limit = types.MaxPageSize
	}
	cursor := ""
	if req.Pagination != nil {
		cursor = req.Pagination.Cursor
	}
	result, err := s.repo.ListFavorites(ctx, principal.UserID, limit, cursor)
	if err != nil {
		return result, err
	}
	s.hydrateImagesBatch(ctx, result.Items)
	s.fillReactionsBatch(ctx, result.Items, principal.UserID)
	s.fillAuthorRelations(ctx, result.Items, principal.UserID)
	return result, nil
}

// validateImageOwnership ensures every file_id in the list belongs to userID,
// has category 'moment_image' and is completed, and that the count does not
// exceed the configured max. A nil slice (no update) and an empty slice
// (clear) are both valid. Returns ErrImagesNotOwned on mismatch.
func (s *Service) validateImageOwnership(ctx context.Context, userID string, fileIDs []string) error {
	if len(fileIDs) == 0 {
		return nil
	}
	if s.appConfig != nil {
		maxImages := s.appConfig.Get(ctx).Limits.MaxMomentImages
		if maxImages > 0 && len(fileIDs) > maxImages {
			return ErrTooManyImages
		}
	}
	if s.images == nil {
		return nil
	}
	if _, err := s.images.ResolveMomentImages(ctx, fileIDs, userID); err != nil {
		observability.L().Warn("moments.image_ownership denied", "error", err, "user_id", userID)
		return ErrImagesNotOwned
	}
	return nil
}

// hydrateImages fills a single moment's images in place via the shared helper.
func (s *Service) hydrateImages(ctx context.Context, item *MomentItem) {
	if item == nil {
		return
	}
	if s.images != nil {
		HydrateSingleMoment(ctx, s.repo, s.images, item)
	}
	// 单条出口（detail/create/update）统一保证恒定数组形状：images/tags/mentions
	// 为空时序列化 [] 而非 null（spec 恒定形状，与 feed 的 ensureMomentArraysBatch 对齐）。
	ensureMomentArrays(item)
}

// hydrateImagesBatch fills a slice of moments' images in place via one query.
func (s *Service) hydrateImagesBatch(ctx context.Context, items []MomentItem) {
	if s.images == nil || len(items) == 0 {
		return
	}
	HydrateMomentImages(ctx, s.repo, s.images, items)
}

// SetReactionAggregator wires the reactions aggregator (called by bootstrap
// after both modules are constructed, breaking the import cycle).
func (s *Service) SetReactionAggregator(ra ReactionAggregator) { s.reactions = ra }

// fillReactions populates MomentItem.reactions[] for a single item via the
// aggregator. No-op when no aggregator is wired or the moment has no reactions.
func (s *Service) fillReactions(ctx context.Context, item *MomentItem, viewerID string) {
	if item == nil {
		return
	}
	if s.reactions == nil {
		item.Reactions = emptyReactions()
		return
	}
	byID, err := s.reactions.AggregateMoments(ctx, []string{item.MomentID}, viewerID)
	if err != nil {
		observability.L().Warn("moments.fillReactions failed", "error", err, "moment_id", item.MomentID)
		item.Reactions = emptyReactions()
		return
	}
	item.Reactions = reactionsFor(byID[item.MomentID])
}

// fillReactionsBatch populates MomentItem.reactions[] for a list in one query.
func (s *Service) fillReactionsBatch(ctx context.Context, items []MomentItem, viewerID string) {
	if len(items) == 0 {
		return
	}
	if s.reactions == nil {
		for i := range items {
			items[i].Reactions = emptyReactions()
		}
		return
	}
	ids := make([]string, 0, len(items))
	for i := range items {
		ids = append(ids, items[i].MomentID)
	}
	byID, err := s.reactions.AggregateMoments(ctx, ids, viewerID)
	if err != nil {
		observability.L().Warn("moments.fillReactionsBatch failed", "error", err)
		for i := range items {
			items[i].Reactions = emptyReactions()
		}
		return
	}
	for i := range items {
		items[i].Reactions = reactionsFor(byID[items[i].MomentID])
	}
}

// SetFollowRelationReader wires the follow-relation reader (bootstrap 注入).
func (s *Service) SetFollowRelationReader(r FollowRelationReader) { s.relations = r }

// fillAuthorRelations 批量填充 MomentItem.author_relation（一次查询，按作者去重）。
// 仅登录视角（viewerID 非空）填充；读取器缺失或出错时保持 null —— UI 只是少一个
// "已关注"标注，绝不因此让列表失败。
func (s *Service) fillAuthorRelations(ctx context.Context, items []MomentItem, viewerID string) {
	if len(items) == 0 || viewerID == "" || s.relations == nil {
		return
	}
	seen := make(map[string]struct{}, len(items))
	authorIDs := make([]string, 0, len(items))
	for i := range items {
		id := items[i].User.UserID
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		authorIDs = append(authorIDs, id)
	}
	if len(authorIDs) == 0 {
		return
	}
	byID, err := s.relations.AuthorRelations(ctx, viewerID, authorIDs)
	if err != nil {
		observability.L().Warn("moments.fillAuthorRelations failed", "error", err)
		return
	}
	for i := range items {
		if rel, ok := byID[items[i].User.UserID]; ok {
			r := rel
			items[i].AuthorRelation = &r
		}
	}
}

// fillAuthorRelation 单条版本（详情出口）。
func (s *Service) fillAuthorRelation(ctx context.Context, item *MomentItem, viewerID string) {
	if item == nil {
		return
	}
	items := []MomentItem{*item}
	s.fillAuthorRelations(ctx, items, viewerID)
	*item = items[0]
}

// fillReactionsBatchOwnerMoment populates OwnerMomentItem.reactions[] for a list.
func (s *Service) fillReactionsBatchOwnerMoment(ctx context.Context, items []OwnerMomentItem, viewerID string) {
	if len(items) == 0 {
		return
	}
	if s.reactions == nil {
		for i := range items {
			items[i].Reactions = emptyReactions()
		}
		return
	}
	ids := make([]string, 0, len(items))
	for i := range items {
		ids = append(ids, items[i].MomentID)
	}
	byID, err := s.reactions.AggregateMoments(ctx, ids, viewerID)
	if err != nil {
		observability.L().Warn("moments.fillReactionsBatchOwnerMoment failed", "error", err)
		for i := range items {
			items[i].Reactions = emptyReactions()
		}
		return
	}
	for i := range items {
		items[i].Reactions = reactionsFor(byID[items[i].MomentID])
	}
}

// AdjustReactionCount 供 reactions 模块在 set/unreact 成功后维护瞬间 reaction
// 计数冗余列（bootstrap 把本 Service 以 ReactionCounter 接口注入 reactions）。
// 计数列（likes_count，语义已复活为 reactions 总数）消除搜索 popular 排序的
// 每行 reactions 聚合子查询。写失败仅 Warn 不阻断反应主流程：列与 reactions
// 表的最终一致由启动期差异回填自愈（见 ensureReactionCountColumn）。
func (s *Service) AdjustReactionCount(ctx context.Context, momentID string, delta int) {
	if err := s.repo.AdjustReactionCount(ctx, momentID, delta); err != nil {
		observability.L().Warn("moments.AdjustReactionCount failed", "error", err, "moment_id", momentID, "delta", delta)
	}
}

// toOwnerMoment 将公开 MomentItem 转为拥有者视角 OwnerMomentItem（含 privacy_level）。
func toOwnerMoment(item MomentItem) OwnerMomentItem {
	return OwnerMomentItem{MomentItem: item, PrivacyLevel: item.PrivacyLevel}
}

func emptyReactions() []Reaction { return []Reaction{} }

func ensureMomentArrays(item *MomentItem) {
	if item.Images == nil {
		item.Images = []MomentImage{}
	}
	if item.Tags == nil {
		item.Tags = []string{}
	}
	if item.Mentions == nil {
		item.Mentions = []string{}
	}
}

func ensureOwnerMomentArrays(item *OwnerMomentItem) {
	ensureMomentArrays(&item.MomentItem)
}

func reactionsFor(recs []Reaction) []Reaction {
	if recs == nil {
		return emptyReactions()
	}
	return recs
}

// viewerUserID returns the authenticated user id, or "" for anonymous callers.
func viewerUserID(ctx context.Context) string {
	principal, ok := platformauth.PrincipalFromContext(ctx)
	if !ok {
		return ""
	}
	return principal.UserID
}

func (s *Service) buildMomentActor(ctx context.Context, userID string) (types.UserInfo, error) {
	if s.users != nil {
		info, err := s.users.GetActorInfo(ctx, userID)
		if err != nil {
			return types.UserInfo{}, err
		}
		return actorFromInfo(info), nil
	}
	return fallbackActor(userID), nil
}

func fallbackActor(userID string) types.UserInfo {
	userID = strings.TrimSpace(userID)
	return types.UserInfo{UserID: userID, Nickname: userID}
}

func actorFromInfo(info actor.Info) types.UserInfo {
	return types.UserInfo{UserID: info.UserID, Nickname: info.Nickname, AvatarURL: types.StrPtr(info.AvatarURL)}
}

func momentSummaryForItem(item MomentItem) types.MomentSummary {
	return types.MomentSummary{
		MomentID:           item.MomentID,
		DescriptionPreview: summarizeMomentDescription(item.Description),
	}
}

func summarizeMomentDescription(description string) string {
	return textutil.TruncateRunes(description, 80)
}

func (s *Service) notifyMomentMentions(ctx context.Context, actorUserID string, item MomentItem) {
	s.notifyMentionUserIDs(ctx, actorUserID, item, normalizeMomentMentions(item.Mentions))
}

// notifyMentionUserIDs 对明确列出的提及用户批量落库通知 + 并行推送。
// notifyMomentMentions（Create）通知全部 mentions；Update 只传新旧差集，
// 避免每次编辑都给已通知过的用户重复推送（每用户 = 1 次通知落库 + 1 次 WS 广播）。
func (s *Service) notifyMentionUserIDs(ctx context.Context, actorUserID string, item MomentItem, mentionedUserIDs []string) {
	if s.notifications == nil {
		return
	}
	targets := make([]string, 0, len(mentionedUserIDs))
	for _, mentionedUserID := range mentionedUserIDs {
		if mentionedUserID != actorUserID {
			targets = append(targets, mentionedUserID)
		}
	}
	if len(targets) == 0 {
		return
	}
	momentSummary := momentSummaryForItem(item)
	actor, err := s.buildMomentActor(ctx, actorUserID)
	if err != nil {
		observability.L().Error("moments.mentions build actor failed", "error", err, "user_id", actorUserID)
		actor = fallbackActor(actorUserID)
	}
	// 批量落库（单次往返）：失败时记日志并按通知缺失降级（notify_id 置空），
	// 广播仍照常进行——与此前逐人失败只影响该人的口径一致，只是失败粒度为整批。
	notifyIDs, err := s.notifications.AppendMentionedBatch(ctx, targets, actor, "moment", item.MomentID, momentSummary, "", "")
	if err != nil {
		observability.L().Error("moments.mentions batch append notifications failed", "error", err, "moment_id", item.MomentID, "count", len(targets))
	}
	// 并行推送（有界并发）：每条 BroadcastToUser 内部含 resume Record + 跨实例
	// publish 两次网络往返，逐条串行在多 @ 场景放大发送延迟（mentions 上限 20）。
	broadcast := func(mentionedUserID, notificationID string) func() error {
		return func() error {
			realtime.DefaultHub().BroadcastToUser(mentionedUserID, realtime.Event{Type: realtime.EventTypeContentMentioned, Payload: ContentMentionedPayload{
				NotifyID:      notificationID,
				TargetType:    "moment",
				MomentID:      item.MomentID,
				MomentSummary: momentSummary,
				Actor:         actor,
				MentionedAt:   item.CreatedAt,
			}})
			return nil
		}
	}
	group, _ := errgroup.WithContext(context.Background())
	group.SetLimit(8)
	for _, mentionedUserID := range targets {
		group.Go(broadcast(mentionedUserID, notifyIDs[mentionedUserID]))
	}
	_ = group.Wait()
}

// broadcastMomentDeletedToFavorites 推送 MomentDeleted 给收藏该瞬间的用户
// （spec api_spec_websocket.md：瞬间被软删除时收藏者据此更新收藏列表为"已删除"占位）。
func (s *Service) broadcastMomentDeletedToFavorites(ctx context.Context, momentID, exceptUserID, deletedAt string) error {
	userIDs, err := s.repo.ListFavoriteUserIDs(ctx, momentID)
	if err != nil {
		return err
	}
	for _, userID := range userIDs {
		if userID == exceptUserID {
			continue
		}
		realtime.DefaultHub().BroadcastToUser(userID, realtime.Event{Type: realtime.EventTypeMomentDeleted, Payload: MomentDeletedPayload{
			MomentID:  momentID,
			DeletedAt: deletedAt,
		}})
	}
	return nil
}

// ListMomentsByAuthor 返回某用户的非删除瞬间（created_at DESC keyset）。
// includePrivate=false 限定公开瞬间（公开个人主页视角）；true 保留给作者
// 本人视角。users 模块个人主页聚合经 Service 接口消费（原 users repo 直查
// moments 表的实现归位本模块）。
func (s *Service) ListMomentsByAuthor(ctx context.Context, userID string, limit int, cursor string, includePrivate bool) (PaginatedMoments, error) {
	return s.repo.ListMomentsByAuthor(ctx, userID, limit, cursor, includePrivate)
}

// ListReactedByUser 返回某用户反应过的瞬间（reacted_at DESC keyset）。
// viewerUserID 为空表示匿名；非空时结果包含该作者自己的私密瞬间
// （spec api_spec_moment.md → moments/list_reacted）。
func (s *Service) ListReactedByUser(ctx context.Context, userID, viewerUserID string, limit int, cursor string) (PaginatedMoments, error) {
	return s.repo.ListReactedByUser(ctx, userID, viewerUserID, limit, cursor)
}
