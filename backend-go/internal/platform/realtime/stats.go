package realtime

import (
	"github.com/prometheus/client_golang/prometheus"
)

// stats.go — Hub 运行状态暴露（Prometheus 采集器，装配见 bootstrap）。

// Stats 返回当前连接/订阅规模快照（持读锁一致读取）：
//   - sessions: 在线 WS 连接数（按 session，即设备数）
//   - users:    有在线连接的去重用户数
//   - groups:   被订阅的群组数
//   - userSubs: user 类型订阅的 (user, sink) 对数
func (h *Hub) Stats() (sessions, users, groups, userSubs int) {
	for i := range h.shards {
		shard := &h.shards[i]
		shard.mu.RLock()
		for _, sinks := range shard.bySession {
			sessions += len(sinks)
		}
		users += len(shard.byUser)
		groups += len(shard.byGroup)
		for _, sinks := range shard.byUserWatchers {
			userSubs += len(sinks)
		}
		shard.mu.RUnlock()
	}
	return sessions, users, groups, userSubs
}

// NewHubCollector 把 Hub 状态暴露为 Prometheus 指标（gauges）：
//
//	realtime_sessions_total       在线 WS 连接数（按设备）
//	realtime_online_users_total   在线去重用户数
//	realtime_group_subscriptions  被订阅群组数
//	realtime_user_subscriptions   user 类型订阅对数
//
// 采集时实时读取 Hub（持读锁），无后台轮询 goroutine。
func NewHubCollector(h *Hub) prometheus.Collector {
	sessions := prometheus.NewDesc(
		"realtime_sessions_total", "在线 WS 连接数（按 session/设备）", nil, nil,
	)
	onlineUsers := prometheus.NewDesc(
		"realtime_online_users_total", "有在线连接的去重用户数", nil, nil,
	)
	groupSubs := prometheus.NewDesc(
		"realtime_group_subscriptions", "被订阅的群组数", nil, nil,
	)
	userSubs := prometheus.NewDesc(
		"realtime_user_subscriptions", "user 类型订阅的 (user, sink) 对数", nil, nil,
	)
	rejected := prometheus.NewDesc(
		"realtime_connections_rejected_total", "因全局连接上限被拒绝的 WS 连接累计数", nil, nil,
	)
	return hubCollector{h: h, sessions: sessions, onlineUsers: onlineUsers, groupSubs: groupSubs, userSubs: userSubs, rejectedDesc: rejected}
}

type hubCollector struct {
	h                                                        *Hub
	sessions, onlineUsers, groupSubs, userSubs, rejectedDesc *prometheus.Desc
}

func (c hubCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.sessions
	ch <- c.onlineUsers
	ch <- c.groupSubs
	ch <- c.userSubs
	ch <- c.rejectedDesc
}

func (c hubCollector) Collect(ch chan<- prometheus.Metric) {
	sessions, users, groups, userSubs := c.h.Stats()
	ch <- prometheus.MustNewConstMetric(c.sessions, prometheus.GaugeValue, float64(sessions))
	ch <- prometheus.MustNewConstMetric(c.onlineUsers, prometheus.GaugeValue, float64(users))
	ch <- prometheus.MustNewConstMetric(c.groupSubs, prometheus.GaugeValue, float64(groups))
	ch <- prometheus.MustNewConstMetric(c.userSubs, prometheus.GaugeValue, float64(userSubs))
	ch <- prometheus.MustNewConstMetric(c.rejectedDesc, prometheus.CounterValue, float64(c.h.RejectedConnections()))
}
