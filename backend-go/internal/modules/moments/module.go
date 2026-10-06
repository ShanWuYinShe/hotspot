package moments

import (
	"context"
	"time"

	"github.com/go-chi/chi/v5"

	"hotspot/backend-go/internal/modules/comments"
	"hotspot/backend-go/internal/modules/notify"
	"hotspot/backend-go/internal/platform/actor"
	"hotspot/backend-go/internal/platform/appconfig"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/idempotency"
	"hotspot/backend-go/internal/platform/observability"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/ratelimit"
	"hotspot/backend-go/internal/platform/resourceguard"
)

type Module struct {
	momentsHandler  *Handler
	commentsHandler *comments.Handler
	Service         *Service
	Comments        *comments.Service
}

func NewModule(factory *persistence.Factory, actorProvider actor.Provider, appConfig appconfig.Source, imageResolver ImageResolver, notificationService *notify.Service) *Module {
	momentRepo := NewPostgresRepository(factory)
	ensureReactionCountBackfill(factory)
	txRunner := persistence.NewTxRunner(factory)
	momentService := NewServiceWithDepsAndImagesAndConfig(momentRepo, notificationService, actorProvider, imageResolver, appConfig, txRunner)
	commentService := comments.NewServiceWithDeps(comments.NewPostgresRepository(factory), momentService, notificationService, actorProvider, appConfig, txRunner)
	return &Module{
		momentsHandler:  NewHandler(momentService),
		commentsHandler: comments.NewHandler(commentService),
		Service:         momentService,
		Comments:        commentService,
	}
}

// CommentsService exposes the comments service for cross-module wiring (e.g.
// reactions target resolution). It returns the same instance used by the
// moments module's own /api/v1/comments routes.
func (m *Module) CommentsService() *comments.Service { return m.Comments }

func (m *Module) Mount(r chi.Router) {
	r.Route("/api/v1/moments", func(momentsRouter chi.Router) {
		momentsRouter.Group(func(group chi.Router) {
			group.Use(platformauth.RequiredAuth)
			group.Use(ratelimit.Middleware)
			group.Use(resourceguard.Middleware)
			group.Use(idempotency.Middleware)
			group.Post("/open_the_door", m.momentsHandler.OpenTheDoor)
			group.Post("/search", m.momentsHandler.Search)
			group.Post("/create", m.momentsHandler.Create)
			group.Post("/update", m.momentsHandler.Update)
			group.Post("/delete", m.momentsHandler.Delete)
			group.Post("/fetch_created", m.momentsHandler.ListCreated)
			group.Post("/fetch_following", m.momentsHandler.FetchFollowing)
			group.Post("/fetch_nearby", m.momentsHandler.FetchNearby)
			group.Post("/set_favorite", m.momentsHandler.Favorite)
			group.Post("/fetch_favorites", m.momentsHandler.ListFavorites)
		})
	})

	r.Route("/api/v1/comments", func(commentsRouter chi.Router) {
		commentsRouter.Group(func(group chi.Router) {
			group.Use(platformauth.RequiredAuth)
			group.Use(ratelimit.Middleware)
			group.Use(resourceguard.Middleware)
			group.Use(idempotency.Middleware)
			group.Post("/fetch", m.commentsHandler.List)
			group.Post("/fetch_replies", m.commentsHandler.ListReplies)
			group.Post("/open_the_door", m.commentsHandler.GetDetail)
			group.Post("/create", m.commentsHandler.Create)
			group.Post("/delete", m.commentsHandler.Delete)
		})
	})
}

// MountPublic mounts the anonymous moments/comments endpoints on the
// /public/v1 node (spec api_spec_public.md).
func (m *Module) MountPublic(r chi.Router) {
	r.Route("/moments", func(momentsRouter chi.Router) {
		momentsRouter.Post("/open_the_door", m.momentsHandler.OpenTheDoor)
		momentsRouter.Post("/search", m.momentsHandler.Search)
		// 附近公开瞬间的可见性不依赖身份（仅 privacy_level='public'），
		// 匿名版为拉新路径（spec api_spec_public.md）。
		momentsRouter.Post("/fetch_nearby", m.momentsHandler.FetchNearby)
	})
	r.Route("/comments", func(commentsRouter chi.Router) {
		commentsRouter.Post("/fetch", m.commentsHandler.List)
		commentsRouter.Post("/fetch_replies", m.commentsHandler.ListReplies)
		commentsRouter.Post("/open_the_door", m.commentsHandler.GetDetail)
	})
}

// reaction 计数冗余列（likes_count，语义已复活为 reactions 总数）的启动期自愈：
// 写路径经 Service.AdjustReactionCount 增量维护，这里把存量/漂移的行按 reactions
// 表实况重算（IS DISTINCT FROM 使稳态下零写入，代价为一次全表扫描，社区规模
// 可忽略）。不放迁移文件的原因：schema_migrations 版本表对「升级前已有全部表」
// 的存量库做一次性基线、不重放历史文件——与本列同版本的迁移会被基线吞掉
// （R47 版本表自身尚未随 tag 发布，时序上必然同版本）。
const ensureReactionCountSQL = `
UPDATE moments m SET likes_count = c.n
FROM (SELECT target_id, COUNT(*) AS n FROM reactions WHERE target_type = 'moment' GROUP BY target_id) c
WHERE m.moment_id = c.target_id AND m.likes_count IS DISTINCT FROM c.n`

const resetOrphanReactionCountSQL = `
UPDATE moments SET likes_count = 0
WHERE likes_count <> 0
  AND NOT EXISTS (SELECT 1 FROM reactions r WHERE r.target_type = 'moment' AND r.target_id = moments.moment_id)`

func ensureReactionCountBackfill(factory *persistence.Factory) {
	if factory == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	q, ok := persistence.RepoQueryable(ctx, factory)
	if !ok {
		return // 非 Postgres 后端（测试 stub / 内存）：无列可回填
	}
	for _, stmt := range []string{ensureReactionCountSQL, resetOrphanReactionCountSQL} {
		if err := q.Exec(ctx, stmt); err != nil {
			// 自愈失败不拒启：计数列只影响搜索热度排序的精度，下一轮启动重试。
			observability.L().Warn("moments: reaction count backfill failed", "error", err)
			return
		}
	}
	observability.L().Info("moments: reaction count backfill ensured")
}
