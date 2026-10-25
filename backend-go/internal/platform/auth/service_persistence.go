package auth

import (
	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/persistence"
)

func NewServiceWithPersistence(cfg config.Config, factory *persistence.Factory) *Service {
	if factory == nil {
		return NewService(cfg)
	}
	// 会话读取加进程内缓存：认证中间件每请求一次会话查询是认证路径固定开销
	//（压测实测 1.5k RPS 时连接池被该查询与资料查询打满）。撤销类写入经装饰器
	// 立即失效；跨实例/直连改库的撤销延迟上限 sessionCacheTTL（部署说明见
	// session_cache.go 注释）。
	sessions := NewCachedSessionStore(NewPostgresSessionStore(factory, nil), sessionCacheTTL, sessionCacheMaxEntries)
	return NewServiceWithStores(cfg, sessions, NewPostgresRefreshStore(factory, nil))
}
