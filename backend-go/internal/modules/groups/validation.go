package groups

import (
	"unicode/utf8"

	"hotspot/backend-go/internal/platform/httpx"
)

// 群组模块长度约束（spec api_spec_base.md「通用字段约束」/「群组字段约束」/
// 「公告字段约束」）：这些限制此前只有非空校验，长度上限完全未实现——TEXT 列
// 不拦超长输入，单字段可塞到 1MiB 请求体上限，列表/详情响应随之放大，且违反
// 契约。校验放在 Service 入口（与 moments/comments 的长度校验同层），返回 10000。
const (
	groupNameMaxRunes           = 50
	groupDescriptionMaxRunes    = 500
	groupTagsMaxCount           = 10
	groupTagMaxRunes            = 30
	groupLocationNameMaxRunes   = 100
	groupPasswordMinRunes       = 8
	groupPasswordMaxRunes       = 32
	groupMaxMembersMin          = 2
	groupMaxMembersMax          = 2000
	groupAnnouncementTitleMax   = 100
	groupAnnouncementContentMax = 5000
	groupInviteMessageMaxRunes  = 200
	groupJoinMessageMaxRunes    = 200
)

func invalidGroupField() *httpx.APIError {
	return httpx.InvalidRequest("参数错误")
}

// validateGroupCoreFields 校验 Create/Update 共用的文本字段；空值表示「不携带」
// 由调用方分别处理（Create 要求 name/location 非空，Update 允许缺省）。
func validateGroupCoreFields(name *string, description *string, tags []string, locationName *string) *httpx.APIError {
	if name != nil {
		// 非空由 Handler 校验（Create 必填 / Update 空串=不修改），此处只兜底长度上限。
		if utf8.RuneCountInString(*name) > groupNameMaxRunes {
			return invalidGroupField()
		}
	}
	if description != nil {
		if utf8.RuneCountInString(*description) > groupDescriptionMaxRunes {
			return invalidGroupField()
		}
	}
	if tags != nil {
		if len(tags) > groupTagsMaxCount {
			return invalidGroupField()
		}
		for _, tag := range tags {
			if utf8.RuneCountInString(tag) > groupTagMaxRunes {
				return invalidGroupField()
			}
		}
	}
	if locationName != nil && *locationName != "" {
		if utf8.RuneCountInString(*locationName) > groupLocationNameMaxRunes {
			return invalidGroupField()
		}
	}
	return nil
}

func validateGroupPassword(password *string) *httpx.APIError {
	if password == nil || *password == "" {
		return nil
	}
	if n := utf8.RuneCountInString(*password); n < groupPasswordMinRunes || n > groupPasswordMaxRunes {
		return invalidGroupField()
	}
	return nil
}

func validateGroupMaxMembers(maxMembers int) *httpx.APIError {
	if maxMembers == 0 {
		return nil // 0 = 使用服务端默认值
	}
	if maxMembers < groupMaxMembersMin || maxMembers > groupMaxMembersMax {
		return invalidGroupField()
	}
	return nil
}

func validateCreateGroupRequest(req CreateGroupRequest) *httpx.APIError {
	name := req.Name
	description := req.Description
	locationName := req.Location.LocationName
	if apiErr := validateGroupCoreFields(&name, &description, req.Tags, &locationName); apiErr != nil {
		return apiErr
	}
	if apiErr := validateGroupPassword(&req.Password); apiErr != nil {
		return apiErr
	}
	return validateGroupMaxMembers(req.MaxMembers)
}

func validateUpdateGroupRequest(req UpdateGroupRequest) *httpx.APIError {
	var name, description, locationName *string
	if req.Name != "" {
		name = &req.Name
	}
	if req.Description != "" {
		description = &req.Description
	}
	var tags []string
	if req.Tags != nil {
		tags = req.Tags
	}
	if req.Location != nil {
		locationName = &req.Location.LocationName
	}
	if apiErr := validateGroupCoreFields(name, description, tags, locationName); apiErr != nil {
		return apiErr
	}
	if apiErr := validateGroupPassword(req.Password); apiErr != nil {
		return apiErr
	}
	if req.MaxMembers != nil {
		return validateGroupMaxMembers(*req.MaxMembers)
	}
	return nil
}

// validateGroupAnnouncementContent 校验公告标题/内容；传 nil 表示不修改该字段。
func validateGroupAnnouncementContent(title, content *string) *httpx.APIError {
	if title != nil && utf8.RuneCountInString(*title) > groupAnnouncementTitleMax {
		return invalidGroupField()
	}
	if content != nil && utf8.RuneCountInString(*content) > groupAnnouncementContentMax {
		return invalidGroupField()
	}
	return nil
}

// validateGroupMessage 校验邀请/申请附言上限（0-200 字符；nil 视为未携带）。
func validateGroupMessage(message *string) *httpx.APIError {
	if message == nil {
		return nil
	}
	if utf8.RuneCountInString(*message) > groupInviteMessageMaxRunes {
		return invalidGroupField()
	}
	return nil
}
