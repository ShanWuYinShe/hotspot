package moments

import (
	"context"
	"math"

	"hotspot/backend-go/internal/platform/observability"
	"hotspot/backend-go/internal/platform/types"
)

// NearbyNotifyTarget 附近新瞬间通知的目标（依赖倒置：moments 不 import
// users，由 bootstrap 在接线处完成 users.NearbyNotifyTarget 的转换）。
type NearbyNotifyTarget struct {
	UserID    string
	Latitude  float64
	Longitude float64
}

// NearbyNotifyTargetsFn 查询开启「附近新瞬间通知」且带最新位置的用户
// （users 模块实现）。
type NearbyNotifyTargetsFn func(ctx context.Context, centerLat, centerLng float64, radiusMeters int, excludeUserID string) ([]NearbyNotifyTarget, error)

// NearbyNotifyPublisher 发布附近新瞬间通知（notify 模块实现：
// PublishNearbyNewMomentPreChecked 落库通知 + WS 推送 NearbyNewMoment；
// PreChecked 变体跳过逐目标 settings 检查——本链路的目标查询已按开关过滤）。
type NearbyNotifyPublisher interface {
	PublishNearbyNewMomentPreChecked(ctx context.Context, targetUserID, actorID, momentID string, distance int, actorPayload types.UserInfo, momentSummaryPayload types.MomentSummary) error
}

// nearbyNotifyRadiusMeters 「附近」半径：与首页实时附近（5km 内）口径一致。
const nearbyNotifyRadiusMeters = 5000

// nearbyNotifyMaxTargets 单次创建的附近通知扇出上限。spec 未定义该数值
// （api_spec_websocket.md NearbyNewMoment：推送给配置开启的用户，节流等
// 细节为服务端实现），此处为防止 moments/create 请求路径延迟随目标数线性
// 放大而设的防线（每个目标 = 1 次通知落库 + 1 次 WS 广播，且当前为同步
// 串行发布）；量级参照 mentions 上限 20 的防放大先例，取 200 兼顾附近
// 通知的到达面。超出部分静默截断（通知为尽力而为语义）。
const nearbyNotifyMaxTargets = 200

// notifyNearbyNewMoment 在带位置的瞬间创建成功后，向开启通知的附近用户推送。
// 各目标失败不阻断主流程（通知为尽力而为）；无位置/未接线/无目标时静默跳过。
func (s *Service) notifyNearbyNewMoment(ctx context.Context, authorID, momentID string, lat, lng float64, actor types.UserInfo, summary types.MomentSummary) {
	if s.nearbyNotifyPublisher == nil || s.nearbyNotifyTargets == nil {
		return
	}
	targets, err := s.nearbyNotifyTargets(ctx, lat, lng, nearbyNotifyRadiusMeters, authorID)
	if err != nil {
		observability.L().Warn("moments.nearby_notify targets failed", "error", err)
		return
	}
	// 目标查询（users 侧）无 LIMIT，密集区域可能返回大量目标；截断到
	// nearbyNotifyMaxTargets，防止 create 请求路径同步串行发布的延迟线性放大。
	if len(targets) > nearbyNotifyMaxTargets {
		targets = targets[:nearbyNotifyMaxTargets]
	}
	for _, t := range targets {
		distance := haversineMeters(lat, lng, t.Latitude, t.Longitude)
		if err := s.nearbyNotifyPublisher.PublishNearbyNewMomentPreChecked(ctx, t.UserID, authorID, momentID, int(math.Round(distance)), actor, summary); err != nil {
			observability.L().Warn("moments.nearby_notify publish failed", "error", err, "target", t.UserID)
		}
	}
}

// haversineMeters 两点球面距离（米）。半径 6371008.8m（IUGG 平均半径）。
func haversineMeters(lat1, lng1, lat2, lng2 float64) float64 {
	const earthRadius = 6371008.8
	rad := math.Pi / 180
	dLat := (lat2 - lat1) * rad
	dLng := (lng2 - lng1) * rad
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*rad)*math.Cos(lat2*rad)*math.Sin(dLng/2)*math.Sin(dLng/2)
	return 2 * earthRadius * math.Asin(math.Min(1, math.Sqrt(a)))
}
