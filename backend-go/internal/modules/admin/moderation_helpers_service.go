package admin

import (
	"net/url"
	"strings"
	"time"

	systemmodule "hotspot/backend-go/internal/modules/system"
	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/types"
)

const (
	announcementTitleMaxLength   = 100
	announcementContentMaxLength = 5000
	keywordMaxLength             = 100
)

func trimOptional(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	return &trimmed
}

func validateAnnouncementCreate(req CreateAnnouncementRequest) *httpx.APIError {
	if !validAnnounceLevel(req.Severity) {
		return httpx.InvalidRequest("参数错误")
	}
	if !validAnnouncementTitle(req.Title) || !validAnnouncementContent(req.Content) {
		return httpx.InvalidRequest("参数错误")
	}
	if !validAnnouncementActionURL(req.ActionURL) {
		return httpx.InvalidRequest("参数错误")
	}
	return nil
}

func validateAnnouncementUpdate(req UpdateAnnouncementRequest) *httpx.APIError {
	if strings.TrimSpace(req.AnnouncementID) == "" {
		return httpx.InvalidRequest("参数错误")
	}
	if req.Title == nil && req.Content == nil && req.Severity == nil && req.ActionURL == nil {
		return httpx.InvalidRequest("参数错误")
	}
	if req.Title != nil && !validAnnouncementTitle(*req.Title) {
		return httpx.InvalidRequest("参数错误")
	}
	if req.Content != nil && !validAnnouncementContent(*req.Content) {
		return httpx.InvalidRequest("参数错误")
	}
	if req.Severity != nil && !validAnnounceLevel(*req.Severity) {
		return httpx.InvalidRequest("参数错误")
	}
	if req.ActionURL != nil && !validAnnouncementActionURL(*req.ActionURL) {
		return httpx.InvalidRequest("参数错误")
	}
	return nil
}

func validateTagSearchRequest(req AdminTagSearchRequest) *httpx.APIError {
	keyword := strings.TrimSpace(req.Keyword)
	if keyword != "" && len([]rune(keyword)) > keywordMaxLength {
		return httpx.InvalidRequest("参数错误")
	}
	return nil
}

func toAdminAnnouncementItem(item systemmodule.AnnouncementRecord) AdminAnnouncementItem {
	resp := AdminAnnouncementItem{
		AnnouncementID: item.AnnouncementID,
		Title:          item.Title,
		Content:        item.Content,
		Severity:       item.Severity,
		CreatedAt:      item.CreatedAt.UTC().Format(time.RFC3339),
		CreatedBy: UserInfo{
			UserID:    item.CreatedBy.UserID,
			Nickname:  item.CreatedBy.Nickname,
			AvatarURL: types.StrPtr(item.CreatedBy.AvatarURL),
		},
		SendPush:  item.SendPush,
		ViewCount: item.ViewCount,
	}
	resp.ActionURL = types.StrPtr(item.ActionURL)
	if item.PushSentAt != nil {
		pushSentAt := item.PushSentAt.UTC().Format(time.RFC3339)
		resp.PushSentAt = &pushSentAt
	}
	return resp
}

func validAnnounceLevel(severity AnnounceSeverity) bool {
	switch severity {
	case AnnounceLevelInfo, AnnounceLevelWarning, AnnounceLevelError:
		return true
	default:
		return false
	}
}

func validAnnouncementTitle(title string) bool {
	length := len([]rune(strings.TrimSpace(title)))
	return length >= 1 && length <= announcementTitleMaxLength
}

func validAnnouncementContent(content string) bool {
	length := len([]rune(strings.TrimSpace(content)))
	return length >= 1 && length <= announcementContentMaxLength
}

func validAnnouncementActionURL(raw string) bool {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return true
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return false
	}
	return parsed.Scheme == "http" || parsed.Scheme == "https"
}
