package search

// toPublicSuggestResponse 把登录视角 typed payload 映射为匿名 Public 判别联合。
// 未识别的 payload（测试替身残留 map）直接跳过并保留类型序稳定性——生产仓储
// 只返回上方 typed 结构，不存在该分支。
func toPublicSuggestResponse(internal SearchSuggestResponse) PublicSearchSuggestResponse {
	items := make([]PublicSearchSuggestItem, 0, len(internal.Items))
	for _, item := range internal.Items {
		switch payload := item.Payload.(type) {
		case TagItem:
			items = append(items, PublicSearchSuggestItem{Type: item.Type, Payload: payload})
		case SearchSuggestUserPayload:
			items = append(items, PublicSearchSuggestItem{Type: item.Type, Payload: PublicSearchSuggestUserPayload{
				User:           payload.User,
				Bio:            payload.Bio,
				Tags:           payload.Tags,
				PresenceStatus: payload.PresenceStatus,
				LastOnlineAt:   payload.LastOnlineAt,
				Location:       payload.Location,
				Stats:          payload.Stats,
			}})
		case SearchSuggestGroupPayload:
			items = append(items, PublicSearchSuggestItem{Type: item.Type, Payload: PublicSearchSuggestGroupPayload{
				Group:              payload.Group,
				Description:        payload.Description,
				Tags:               payload.Tags,
				JoinMethod:         payload.JoinMethod,
				RequiresApproval:   payload.RequiresApproval,
				Location:           payload.Location,
				LastActivityAt:     payload.LastActivityAt,
				OnlineMembersCount: payload.OnlineMembersCount,
				NextActivityAt:     payload.NextActivityAt,
				CreatedAt:          payload.CreatedAt,
				OwnerID:            payload.OwnerID,
			}})
		case SearchSuggestMomentPayload:
			items = append(items, PublicSearchSuggestItem{Type: item.Type, Payload: payload})
		}
	}
	return PublicSearchSuggestResponse{Items: items}
}
