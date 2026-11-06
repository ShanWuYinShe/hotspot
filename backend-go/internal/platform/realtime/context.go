package realtime

import "context"

// hubContextKey 是请求级 Hub 注入的 context key。
type hubContextKey struct{}

// WithHub 把进程内 Hub 实例注入 context。组合根（bootstrap）以中间件形式
// 在请求入口注入一次，模块内所有广播/会话操作统一 HubFromContext(ctx) 获取，
// 不再直接引用进程级 DefaultHub 单例——同进程可装配多套 App 的测试/嵌入场景
// 不会互相串台，依赖关系也从「隐式全局」变成「显式 context 依赖」。
func WithHub(ctx context.Context, hub *Hub) context.Context {
	if hub == nil {
		return ctx
	}
	return context.WithValue(ctx, hubContextKey{}, hub)
}

// HubFromContext 返回请求级 Hub；未注入时回退到 DefaultHub（后台任务/直调
// 服务方法的测试路径仍可用 SetDefaultHub 注入替身）。生产请求路径由
// bootstrap 的 hub 中间件保证恒有值。
func HubFromContext(ctx context.Context) *Hub {
	if hub, ok := ctx.Value(hubContextKey{}).(*Hub); ok && hub != nil {
		return hub
	}
	return DefaultHub()
}
