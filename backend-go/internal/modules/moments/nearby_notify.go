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
// PublishNearbyNewMoment 落库通知 + WS 推送 NearbyNewMoment）。
type NearbyNotifyPublisher interface {
	PublishNearbyNewMoment(ctx context.Context, targetUserID, actorID, momentID string, distance int, actorPayload types.UserInfo, momentSummaryPayload types.MomentSummary) error
}

// nearbyNotifyRadiusMeters 「附近」半径：与首页实时附近（5km 内）口径一致。
const nearbyNotifyRadiusMeters = 5000

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
	for _, t := range targets {
		distance := haversineMeters(lat, lng, t.Latitude, t.Longitude)
		if err := s.nearbyNotifyPublisher.PublishNearbyNewMoment(ctx, t.UserID, authorID, momentID, int(math.Round(distance)), actor, summary); err != nil {
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
