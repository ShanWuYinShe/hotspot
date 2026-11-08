package groups

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"hotspot/backend-go/internal/platform/httpx"
)

func TestCovHandlerListApplicationsErrorPaths(t *testing.T) {
	body := `{"group_id":"g1"}`
	apiErrRepo := covNewRepo(covOwnerGroup())
	apiErrRepo.getErr = httpx.InvalidRequest("bad")
	missingRepo := covNewRepo(covOwnerGroup())
	missingRepo.getFound = false
	memberRepo := covNewRepo(covOwnerGroup())
	memberRepo.getResult = covMemberGroup()
	cursorRepo := covNewRepo(covOwnerGroup())
	cursorRepo.listApplicationsErr = ErrInvalidCursor
	genericRepo := covNewRepo(covOwnerGroup())
	genericRepo.listApplicationsErr = errors.New("boom")
	covRunCases(t, []covCase{
		{name: "invalid json", repo: covNewRepo(covOwnerGroup()), fn: func(h *Handler) http.HandlerFunc { return h.ListApplications }, body: "{", want: httpx.CodeInvalidRequest},
		{name: "empty group id", repo: covNewRepo(covOwnerGroup()), fn: func(h *Handler) http.HandlerFunc { return h.ListApplications }, body: "{}", want: httpx.CodeInvalidRequest},
		{name: "api error", repo: apiErrRepo, fn: func(h *Handler) http.HandlerFunc { return h.ListApplications }, body: body, want: httpx.CodeInvalidRequest},
		{name: "not found", repo: missingRepo, fn: func(h *Handler) http.HandlerFunc { return h.ListApplications }, body: body, want: httpx.CodeNotFound},
		{name: "not admin", repo: memberRepo, fn: func(h *Handler) http.HandlerFunc { return h.ListApplications }, body: body, want: httpx.CodeForbidden},
		{name: "invalid cursor", repo: cursorRepo, fn: func(h *Handler) http.HandlerFunc { return h.ListApplications }, body: body, want: httpx.CodeInvalidRequest},
		{name: "generic error", repo: genericRepo, fn: func(h *Handler) http.HandlerFunc { return h.ListApplications }, body: body, want: httpx.CodeInternalUnavailable},
	})
}

func TestCovHandlerApproveApplicationErrorPaths(t *testing.T) {
	body := `{"group_id":"g1","application_id":"a1"}`
	repoWithApp := func(groupID string) *covRepo {
		repo := covNewRepo(covOwnerGroup())
		repo.listApplicationsItems = []GroupApplicationItem{{
			ApplicationID: "a1",
			GroupID:       groupID,
			Applicant:     UserInfo{UserID: "user_bob"},
			Status:        ApplicationStatusPending,
		}}
		return repo
	}
	mismatchRepo := repoWithApp("g2")
	bannedRepo := repoWithApp("g1")
	bannedRepo.isBannedResult = true
	genericRepo := repoWithApp("g1")
	genericRepo.approveApplicationErr = errors.New("boom")
	covRunCases(t, []covCase{
		{name: "invalid json", repo: covNewRepo(covOwnerGroup()), fn: func(h *Handler) http.HandlerFunc { return h.ApproveApplication }, body: "{", want: httpx.CodeInvalidRequest},
		{name: "missing application id", repo: covNewRepo(covOwnerGroup()), fn: func(h *Handler) http.HandlerFunc { return h.ApproveApplication }, body: `{"group_id":"g1"}`, want: httpx.CodeInvalidRequest},
		{name: "application group mismatch", repo: mismatchRepo, fn: func(h *Handler) http.HandlerFunc { return h.ApproveApplication }, body: body, want: httpx.CodeNotFound},
		{name: "banned applicant", repo: bannedRepo, fn: func(h *Handler) http.HandlerFunc { return h.ApproveApplication }, body: body, want: httpx.CodeForbidden},
		{name: "generic repo error", repo: genericRepo, fn: func(h *Handler) http.HandlerFunc { return h.ApproveApplication }, body: body, want: httpx.CodeInternalUnavailable},
	})
}

func TestCovHandlerRejectApplicationErrorPaths(t *testing.T) {
	body := `{"group_id":"g1","application_id":"a1"}`
	pendingRepo := func() *covRepo {
		repo := covNewRepo(covOwnerGroup())
		repo.listApplicationsItems = []GroupApplicationItem{{
			ApplicationID: "a1",
			GroupID:       "g1",
			Applicant:     UserInfo{UserID: "user_bob"},
			Status:        ApplicationStatusPending,
		}}
		return repo
	}
	mismatchRepo := covNewRepo(covOwnerGroup())
	mismatchRepo.listApplicationsItems = []GroupApplicationItem{{ApplicationID: "a1", GroupID: "g2", Status: ApplicationStatusPending}}
	genericRepo := pendingRepo()
	genericRepo.rejectApplicationErr = errors.New("boom")
	covRunCases(t, []covCase{
		{name: "invalid json", repo: covNewRepo(covOwnerGroup()), fn: func(h *Handler) http.HandlerFunc { return h.RejectApplication }, body: "{", want: httpx.CodeInvalidRequest},
		{name: "missing application id", repo: covNewRepo(covOwnerGroup()), fn: func(h *Handler) http.HandlerFunc { return h.RejectApplication }, body: `{"group_id":"g1"}`, want: httpx.CodeInvalidRequest},
		{name: "application group mismatch", repo: mismatchRepo, fn: func(h *Handler) http.HandlerFunc { return h.RejectApplication }, body: body, want: httpx.CodeNotFound},
		{name: "invalid reason", repo: pendingRepo(), fn: func(h *Handler) http.HandlerFunc { return h.RejectApplication }, body: `{"group_id":"g1","application_id":"a1","reason":"` + strings.Repeat("a", 201) + `"}`, want: httpx.CodeInvalidRequest},
		{name: "generic repo error", repo: genericRepo, fn: func(h *Handler) http.HandlerFunc { return h.RejectApplication }, body: body, want: httpx.CodeInternalUnavailable},
	})
}

func TestCovHandlerSetMemberRoleErrorPaths(t *testing.T) {
	body := `{"group_id":"g1","user_id":"u2","role":"admin"}`
	missingRepo := covNewRepo(covOwnerGroup())
	missingRepo.getFound = false
	memberViewRepo := covNewRepo(covOwnerGroup())
	memberViewRepo.getResult = covMemberGroup()
	targetNotMemberRepo := covNewRepo(covOwnerGroup())
	targetNotMemberRepo.isMemberResult = false
	ownerTargetRepo := covNewRepo(covOwnerGroup())
	ownerID := "u2"
	ownerTargetRepo.getResult = GroupItem{Group: GroupInfo{GroupID: "g1"}, ViewerState: &GroupViewerState{Role: GroupRoleOwner}, OwnerID: &ownerID}
	ownerTargetRepo.isMemberResult = true
	genericRepo := covNewRepo(covOwnerGroup())
	genericRepo.isMemberResult = true
	genericRepo.setRoleErr = errors.New("boom")
	covRunCases(t, []covCase{
		{name: "invalid json", repo: covNewRepo(covOwnerGroup()), fn: func(h *Handler) http.HandlerFunc { return h.SetMemberRole }, body: "{", want: httpx.CodeInvalidRequest},
		{name: "missing user id", repo: covNewRepo(covOwnerGroup()), fn: func(h *Handler) http.HandlerFunc { return h.SetMemberRole }, body: `{"group_id":"g1","role":"admin"}`, want: httpx.CodeInvalidRequest},
		{name: "not found", repo: missingRepo, fn: func(h *Handler) http.HandlerFunc { return h.SetMemberRole }, body: body, want: httpx.CodeNotFound},
		{name: "not owner", repo: memberViewRepo, fn: func(h *Handler) http.HandlerFunc { return h.SetMemberRole }, body: body, want: httpx.CodeForbidden},
		{name: "target not member", repo: targetNotMemberRepo, fn: func(h *Handler) http.HandlerFunc { return h.SetMemberRole }, body: body, want: httpx.CodeConflict},
		{name: "owner cannot be demoted", repo: ownerTargetRepo, fn: func(h *Handler) http.HandlerFunc { return h.SetMemberRole }, body: body, want: httpx.CodeConflict},
		{name: "generic repo error", repo: genericRepo, fn: func(h *Handler) http.HandlerFunc { return h.SetMemberRole }, body: body, want: httpx.CodeInternalUnavailable},
	})
}

func TestCovHandlerSetMuteErrorPaths(t *testing.T) {
	muteBody := `{"group_id":"g1","user_id":"u2","active":true,"duration_minutes":10}`
	unmuteBody := `{"group_id":"g1","user_id":"u2","active":false}`
	apiErrRepo := covNewRepo(covOwnerGroup())
	apiErrRepo.getFound = false
	apiErrRepo.getErr = httpx.InvalidRequest("bad")
	missingRepo := covNewRepo(covOwnerGroup())
	missingRepo.getFound = false
	memberViewRepo := covNewRepo(covOwnerGroup())
	memberViewRepo.getResult = covMemberGroup()
	targetNotMemberRepo := covNewRepo(covOwnerGroup())
	ownerID := "u2"
	ownerTargetRepo := covNewRepo(covOwnerGroup())
	ownerTargetRepo.getResult = GroupItem{Group: GroupInfo{GroupID: "g1"}, ViewerState: &GroupViewerState{Role: GroupRoleOwner}, OwnerID: &ownerID}
	ownerTargetRepo.isMemberResult = true
	genericMuteRepo := covNewRepo(covOwnerGroup())
	genericMuteRepo.isMemberResult = true
	genericMuteRepo.muteMemberErr = errors.New("boom")
	genericUnmuteRepo := covNewRepo(covOwnerGroup())
	genericUnmuteRepo.isMemberResult = true
	genericUnmuteRepo.unmuteMemberErr = errors.New("boom")
	covRunCases(t, []covCase{
		{name: "invalid json", repo: covNewRepo(covOwnerGroup()), fn: func(h *Handler) http.HandlerFunc { return h.SetMute }, body: "{", want: httpx.CodeInvalidRequest},
		{name: "missing user id", repo: covNewRepo(covOwnerGroup()), fn: func(h *Handler) http.HandlerFunc { return h.SetMute }, body: `{"group_id":"g1","active":true}`, want: httpx.CodeInvalidRequest},
		{name: "mute api error", repo: apiErrRepo, fn: func(h *Handler) http.HandlerFunc { return h.SetMute }, body: muteBody, want: httpx.CodeInvalidRequest},
		{name: "mute not found", repo: missingRepo, fn: func(h *Handler) http.HandlerFunc { return h.SetMute }, body: muteBody, want: httpx.CodeNotFound},
		{name: "mute not admin", repo: memberViewRepo, fn: func(h *Handler) http.HandlerFunc { return h.SetMute }, body: muteBody, want: httpx.CodeForbidden},
		{name: "mute target not member", repo: targetNotMemberRepo, fn: func(h *Handler) http.HandlerFunc { return h.SetMute }, body: muteBody, want: httpx.CodeConflict},
		{name: "mute owner target", repo: ownerTargetRepo, fn: func(h *Handler) http.HandlerFunc { return h.SetMute }, body: muteBody, want: httpx.CodeConflict},
		{name: "mute generic error", repo: genericMuteRepo, fn: func(h *Handler) http.HandlerFunc { return h.SetMute }, body: muteBody, want: httpx.CodeInternalUnavailable},
		{name: "unmute target not member", repo: targetNotMemberRepo, fn: func(h *Handler) http.HandlerFunc { return h.SetMute }, body: unmuteBody, want: httpx.CodeConflict},
		{name: "unmute generic error", repo: genericUnmuteRepo, fn: func(h *Handler) http.HandlerFunc { return h.SetMute }, body: unmuteBody, want: httpx.CodeInternalUnavailable},
	})
}

func TestCovHandlerSetMemberBanErrorPaths(t *testing.T) {
	body := `{"group_id":"g1","user_id":"u2","active":true}`
	missingRepo := covNewRepo(covOwnerGroup())
	missingRepo.getFound = false
	memberViewRepo := covNewRepo(covOwnerGroup())
	memberViewRepo.getResult = covMemberGroup()
	selfBanRepo := covNewRepo(covOwnerGroup())
	ownerID := "u2"
	ownerTargetRepo := covNewRepo(covOwnerGroup())
	ownerTargetRepo.getResult = GroupItem{Group: GroupInfo{GroupID: "g1"}, ViewerState: &GroupViewerState{Role: GroupRoleOwner}, OwnerID: &ownerID}
	genericBanRepo := covNewRepo(covOwnerGroup())
	genericBanRepo.banMemberErr = errors.New("boom")
	genericUnbanRepo := covNewRepo(covOwnerGroup())
	genericUnbanRepo.unbanMemberErr = errors.New("boom")
	covRunCases(t, []covCase{
		{name: "invalid json", repo: covNewRepo(covOwnerGroup()), fn: func(h *Handler) http.HandlerFunc { return h.SetMemberBan }, body: "{", want: httpx.CodeInvalidRequest},
		{name: "missing user id", repo: covNewRepo(covOwnerGroup()), fn: func(h *Handler) http.HandlerFunc { return h.SetMemberBan }, body: `{"group_id":"g1","active":true}`, want: httpx.CodeInvalidRequest},
		{name: "not found", repo: missingRepo, fn: func(h *Handler) http.HandlerFunc { return h.SetMemberBan }, body: body, want: httpx.CodeNotFound},
		{name: "not admin", repo: memberViewRepo, fn: func(h *Handler) http.HandlerFunc { return h.SetMemberBan }, body: body, want: httpx.CodeForbidden},
		{name: "cannot ban self", repo: selfBanRepo, fn: func(h *Handler) http.HandlerFunc { return h.SetMemberBan }, body: `{"group_id":"g1","user_id":"user_alice","active":true}`, userID: "user_alice", want: httpx.CodeForbidden},
		{name: "owner cannot be banned", repo: ownerTargetRepo, fn: func(h *Handler) http.HandlerFunc { return h.SetMemberBan }, body: body, want: httpx.CodeConflict},
		{name: "ban generic error", repo: genericBanRepo, fn: func(h *Handler) http.HandlerFunc { return h.SetMemberBan }, body: body, want: httpx.CodeInternalUnavailable},
		{name: "unban generic error", repo: genericUnbanRepo, fn: func(h *Handler) http.HandlerFunc { return h.SetMemberBan }, body: `{"group_id":"g1","user_id":"u2","active":false}`, want: httpx.CodeInternalUnavailable},
	})
}

func TestCovHandlerListBannedMembersErrorPaths(t *testing.T) {
	body := `{"group_id":"g1"}`
	apiErrRepo := covNewRepo(covOwnerGroup())
	apiErrRepo.getFound = false
	apiErrRepo.getErr = httpx.InvalidRequest("bad")
	missingRepo := covNewRepo(covOwnerGroup())
	missingRepo.getFound = false
	memberViewRepo := covNewRepo(covOwnerGroup())
	memberViewRepo.getResult = covMemberGroup()
	cursorRepo := covNewRepo(covOwnerGroup())
	cursorRepo.bannedMemErr = ErrInvalidCursor
	genericRepo := covNewRepo(covOwnerGroup())
	genericRepo.bannedMemErr = errors.New("boom")
	covRunCases(t, []covCase{
		{name: "invalid json", repo: covNewRepo(covOwnerGroup()), fn: func(h *Handler) http.HandlerFunc { return h.ListBannedMembers }, body: "{", want: httpx.CodeInvalidRequest},
		{name: "empty group id", repo: covNewRepo(covOwnerGroup()), fn: func(h *Handler) http.HandlerFunc { return h.ListBannedMembers }, body: "{}", want: httpx.CodeInvalidRequest},
		{name: "api error", repo: apiErrRepo, fn: func(h *Handler) http.HandlerFunc { return h.ListBannedMembers }, body: body, want: httpx.CodeInvalidRequest},
		{name: "not found", repo: missingRepo, fn: func(h *Handler) http.HandlerFunc { return h.ListBannedMembers }, body: body, want: httpx.CodeNotFound},
		{name: "not admin", repo: memberViewRepo, fn: func(h *Handler) http.HandlerFunc { return h.ListBannedMembers }, body: body, want: httpx.CodeForbidden},
		{name: "invalid cursor", repo: cursorRepo, fn: func(h *Handler) http.HandlerFunc { return h.ListBannedMembers }, body: body, want: httpx.CodeInvalidRequest},
		{name: "generic error", repo: genericRepo, fn: func(h *Handler) http.HandlerFunc { return h.ListBannedMembers }, body: body, want: httpx.CodeInternalUnavailable},
	})
}

func TestCovHandlerRemoveMemberErrorPaths(t *testing.T) {
	body := `{"group_id":"g1","user_id":"u2"}`
	missingRepo := covNewRepo(covOwnerGroup())
	missingRepo.getFound = false
	memberViewRepo := covNewRepo(covOwnerGroup())
	memberViewRepo.getResult = covMemberGroup()
	targetNotMemberRepo := covNewRepo(covOwnerGroup())
	ownerID := "u2"
	ownerTargetRepo := covNewRepo(covOwnerGroup())
	ownerTargetRepo.getResult = GroupItem{Group: GroupInfo{GroupID: "g1"}, ViewerState: &GroupViewerState{Role: GroupRoleOwner}, OwnerID: &ownerID}
	ownerTargetRepo.isMemberResult = true
	genericRepo := covNewRepo(covOwnerGroup())
	genericRepo.isMemberResult = true
	genericRepo.removeMemberErr = errors.New("boom")
	covRunCases(t, []covCase{
		{name: "invalid json", repo: covNewRepo(covOwnerGroup()), fn: func(h *Handler) http.HandlerFunc { return h.RemoveMember }, body: "{", want: httpx.CodeInvalidRequest},
		{name: "missing user id", repo: covNewRepo(covOwnerGroup()), fn: func(h *Handler) http.HandlerFunc { return h.RemoveMember }, body: `{"group_id":"g1"}`, want: httpx.CodeInvalidRequest},
		{name: "not found", repo: missingRepo, fn: func(h *Handler) http.HandlerFunc { return h.RemoveMember }, body: body, want: httpx.CodeNotFound},
		{name: "not admin", repo: memberViewRepo, fn: func(h *Handler) http.HandlerFunc { return h.RemoveMember }, body: body, want: httpx.CodeForbidden},
		{name: "target not member", repo: targetNotMemberRepo, fn: func(h *Handler) http.HandlerFunc { return h.RemoveMember }, body: body, want: httpx.CodeConflict},
		{name: "owner cannot be removed", repo: ownerTargetRepo, fn: func(h *Handler) http.HandlerFunc { return h.RemoveMember }, body: body, want: httpx.CodeConflict},
		{name: "generic repo error", repo: genericRepo, fn: func(h *Handler) http.HandlerFunc { return h.RemoveMember }, body: body, want: httpx.CodeInternalUnavailable},
	})
}

func TestCovHandlerTransferOwnershipErrorPaths(t *testing.T) {
	body := `{"group_id":"g1","new_owner_id":"u2"}`
	missingRepo := covNewRepo(covOwnerGroup())
	missingRepo.getFound = false
	memberViewRepo := covNewRepo(covOwnerGroup())
	memberViewRepo.getResult = covMemberGroup()
	targetNotMemberRepo := covNewRepo(covOwnerGroup())
	genericRepo := covNewRepo(covOwnerGroup())
	genericRepo.isMemberResult = true
	genericRepo.transferOwnershipErr = errors.New("boom")
	covRunCases(t, []covCase{
		{name: "invalid json", repo: covNewRepo(covOwnerGroup()), fn: func(h *Handler) http.HandlerFunc { return h.TransferOwnership }, body: "{", want: httpx.CodeInvalidRequest},
		{name: "missing new owner", repo: covNewRepo(covOwnerGroup()), fn: func(h *Handler) http.HandlerFunc { return h.TransferOwnership }, body: `{"group_id":"g1"}`, want: httpx.CodeInvalidRequest},
		{name: "not found", repo: missingRepo, fn: func(h *Handler) http.HandlerFunc { return h.TransferOwnership }, body: body, want: httpx.CodeNotFound},
		{name: "not owner", repo: memberViewRepo, fn: func(h *Handler) http.HandlerFunc { return h.TransferOwnership }, body: body, want: httpx.CodeForbidden},
		{name: "target not member", repo: targetNotMemberRepo, fn: func(h *Handler) http.HandlerFunc { return h.TransferOwnership }, body: body, want: httpx.CodeConflict},
		{name: "generic repo error", repo: genericRepo, fn: func(h *Handler) http.HandlerFunc { return h.TransferOwnership }, body: body, want: httpx.CodeInternalUnavailable},
	})
}

func TestCovHandlerInviteErrorPaths(t *testing.T) {
	withListInvitesErr := func(err error) *covRepo {
		repo := covNewRepo(covOwnerGroup())
		repo.listMyInvitesErr = err
		return repo
	}
	pendingInviteRepo := func() *covRepo {
		repo := covNewRepo(covOwnerGroup())
		repo.listMyInvitesItems = []GroupInviteItem{{
			InviteID: "inv_1",
			Group:    GroupInfo{GroupID: "g1"},
			Status:   ApplicationStatusPending,
		}}
		return repo
	}
	bannedRepo := pendingInviteRepo()
	bannedRepo.isBannedResult = true
	expiredRepo := covNewRepo(covOwnerGroup())
	expiredRepo.listMyInvitesItems = []GroupInviteItem{{
		InviteID: "inv_1",
		Group:    GroupInfo{GroupID: "g1"},
		Status:   ApplicationStatusRejected,
	}}
	genericAcceptRepo := pendingInviteRepo()
	genericAcceptRepo.acceptInviteErr = errors.New("boom")
	genericRejectRepo := pendingInviteRepo()
	genericRejectRepo.rejectInviteErr = errors.New("boom")
	missingRepo := covNewRepo(covOwnerGroup())
	missingRepo.getFound = false
	memberViewRepo := covNewRepo(covOwnerGroup())
	memberViewRepo.getResult = covMemberGroup()
	alreadyMemberRepo := covNewRepo(covOwnerGroup())
	alreadyMemberRepo.createErr = ErrAlreadyMember // Service 将 ErrAlreadyMember 归一为 no-op 成功
	inviteBannedRepo := covNewRepo(covOwnerGroup())
	inviteBannedRepo.isBannedResult = true
	inviteGenericRepo := covNewRepo(covOwnerGroup())
	inviteGenericRepo.createErr = errors.New("boom")
	sentCursorRepo := covNewRepo(covOwnerGroup())
	sentCursorRepo.sentInvitesErr = ErrInvalidCursor
	sentGenericRepo := covNewRepo(covOwnerGroup())
	sentGenericRepo.sentInvitesErr = errors.New("boom")
	sentAPIErrRepo := covNewRepo(covOwnerGroup())
	sentAPIErrRepo.getFound = false
	sentAPIErrRepo.getErr = httpx.InvalidRequest("bad")
	sentMissingRepo := covNewRepo(covOwnerGroup())
	sentMissingRepo.getFound = false
	sentMemberRepo := covNewRepo(covOwnerGroup())
	sentMemberRepo.getResult = covMemberGroup()
	revokeGenericRepo := covNewRepo(covOwnerGroup())
	revokeGenericRepo.revokeInviteErr = errors.New("boom")

	covRunCases(t, []covCase{
		// ListMyInvites
		{name: "invites invalid json", repo: covNewRepo(covOwnerGroup()), fn: func(h *Handler) http.HandlerFunc { return h.ListMyInvites }, body: "{", want: httpx.CodeInvalidRequest},
		{name: "invites api error", repo: withListInvitesErr(httpx.InvalidRequest("bad")), fn: func(h *Handler) http.HandlerFunc { return h.ListMyInvites }, body: "{}", userID: "user_alice", want: httpx.CodeInvalidRequest},
		{name: "invites invalid cursor", repo: withListInvitesErr(ErrInvalidCursor), fn: func(h *Handler) http.HandlerFunc { return h.ListMyInvites }, body: "{}", userID: "user_alice", want: httpx.CodeInvalidRequest},
		{name: "invites generic error", repo: withListInvitesErr(errors.New("boom")), fn: func(h *Handler) http.HandlerFunc { return h.ListMyInvites }, body: "{}", userID: "user_alice", want: httpx.CodeInternalUnavailable},
		// AcceptInvite
		{name: "accept invalid json", repo: covNewRepo(covOwnerGroup()), fn: func(h *Handler) http.HandlerFunc { return h.AcceptInvite }, body: "{", want: httpx.CodeInvalidRequest},
		{name: "accept missing invite id", repo: covNewRepo(covOwnerGroup()), fn: func(h *Handler) http.HandlerFunc { return h.AcceptInvite }, body: "{}", want: httpx.CodeInvalidRequest},
		{name: "accept pending invite of other user", repo: missingRepo, fn: func(h *Handler) http.HandlerFunc { return h.AcceptInvite }, body: `{"invite_id":"inv_1"}`, userID: "user_alice", want: httpx.CodeNotFound},
		{name: "accept banned", repo: bannedRepo, fn: func(h *Handler) http.HandlerFunc { return h.AcceptInvite }, body: `{"invite_id":"inv_1"}`, userID: "user_alice", want: httpx.CodeForbidden},
		{name: "accept expired", repo: expiredRepo, fn: func(h *Handler) http.HandlerFunc { return h.AcceptInvite }, body: `{"invite_id":"inv_1"}`, userID: "user_alice", want: httpx.CodeConflict},
		{name: "accept generic error", repo: genericAcceptRepo, fn: func(h *Handler) http.HandlerFunc { return h.AcceptInvite }, body: `{"invite_id":"inv_1"}`, userID: "user_alice", want: httpx.CodeInternalUnavailable},
		// RejectInvite
		{name: "reject invalid json", repo: covNewRepo(covOwnerGroup()), fn: func(h *Handler) http.HandlerFunc { return h.RejectInvite }, body: "{", want: httpx.CodeInvalidRequest},
		{name: "reject missing invite id", repo: covNewRepo(covOwnerGroup()), fn: func(h *Handler) http.HandlerFunc { return h.RejectInvite }, body: "{}", want: httpx.CodeInvalidRequest},
		{name: "reject generic error", repo: genericRejectRepo, fn: func(h *Handler) http.HandlerFunc { return h.RejectInvite }, body: `{"invite_id":"inv_1"}`, userID: "user_alice", want: httpx.CodeInternalUnavailable},
		// InviteMember
		{name: "invite invalid json", repo: covNewRepo(covOwnerGroup()), fn: func(h *Handler) http.HandlerFunc { return h.InviteMember }, body: "{", want: httpx.CodeInvalidRequest},
		{name: "invite missing user id", repo: covNewRepo(covOwnerGroup()), fn: func(h *Handler) http.HandlerFunc { return h.InviteMember }, body: `{"group_id":"g1"}`, want: httpx.CodeInvalidRequest},
		{name: "invite message too long", repo: covNewRepo(covOwnerGroup()), fn: func(h *Handler) http.HandlerFunc { return h.InviteMember }, body: `{"group_id":"g1","user_id":"u2","message":"` + strings.Repeat("a", 201) + `"}`, want: httpx.CodeInvalidRequest},
		{name: "invite not found", repo: missingRepo, fn: func(h *Handler) http.HandlerFunc { return h.InviteMember }, body: `{"group_id":"g1","user_id":"u2"}`, want: httpx.CodeNotFound},
		{name: "invite not admin", repo: memberViewRepo, fn: func(h *Handler) http.HandlerFunc { return h.InviteMember }, body: `{"group_id":"g1","user_id":"u2"}`, want: httpx.CodeForbidden},
		{name: "invite already member noop success", repo: alreadyMemberRepo, fn: func(h *Handler) http.HandlerFunc { return h.InviteMember }, body: `{"group_id":"g1","user_id":"u2"}`, want: httpx.CodeOK},
		{name: "invite banned user", repo: inviteBannedRepo, fn: func(h *Handler) http.HandlerFunc { return h.InviteMember }, body: `{"group_id":"g1","user_id":"u2"}`, want: httpx.CodeConflict},
		{name: "invite generic repo error", repo: inviteGenericRepo, fn: func(h *Handler) http.HandlerFunc { return h.InviteMember }, body: `{"group_id":"g1","user_id":"u2"}`, want: httpx.CodeInternalUnavailable},
		// ListSentInvites
		{name: "sent invalid json", repo: covNewRepo(covOwnerGroup()), fn: func(h *Handler) http.HandlerFunc { return h.ListSentInvites }, body: "{", want: httpx.CodeInvalidRequest},
		{name: "sent api error", repo: sentAPIErrRepo, fn: func(h *Handler) http.HandlerFunc { return h.ListSentInvites }, body: `{"group_id":"g1"}`, want: httpx.CodeInvalidRequest},
		{name: "sent not found", repo: sentMissingRepo, fn: func(h *Handler) http.HandlerFunc { return h.ListSentInvites }, body: `{"group_id":"g1"}`, want: httpx.CodeNotFound},
		{name: "sent not admin", repo: sentMemberRepo, fn: func(h *Handler) http.HandlerFunc { return h.ListSentInvites }, body: `{"group_id":"g1"}`, want: httpx.CodeForbidden},
		{name: "sent invalid cursor", repo: sentCursorRepo, fn: func(h *Handler) http.HandlerFunc { return h.ListSentInvites }, body: `{"group_id":"g1"}`, want: httpx.CodeInvalidRequest},
		{name: "sent generic error", repo: sentGenericRepo, fn: func(h *Handler) http.HandlerFunc { return h.ListSentInvites }, body: `{"group_id":"g1"}`, want: httpx.CodeInternalUnavailable},
		// RevokeInvite
		{name: "revoke invalid json", repo: covNewRepo(covOwnerGroup()), fn: func(h *Handler) http.HandlerFunc { return h.RevokeInvite }, body: "{", want: httpx.CodeInvalidRequest},
		{name: "revoke missing invite id", repo: covNewRepo(covOwnerGroup()), fn: func(h *Handler) http.HandlerFunc { return h.RevokeInvite }, body: `{"group_id":"g1"}`, want: httpx.CodeInvalidRequest},
		{name: "revoke generic error", repo: revokeGenericRepo, fn: func(h *Handler) http.HandlerFunc { return h.RevokeInvite }, body: `{"group_id":"g1","invite_id":"inv_1"}`, want: httpx.CodeInternalUnavailable},
	})
}

func TestCovHandlerAdminErrorPaths(t *testing.T) {
	apiErrRepo := covNewRepo(covOwnerGroup())
	apiErrRepo.getErr = httpx.InvalidRequest("bad")
	genericRepo := covNewRepo(covOwnerGroup())
	genericRepo.getErr = errors.New("boom")
	missingRepo := covNewRepo(covOwnerGroup())
	missingRepo.getFound = false

	adminAPIErrRepo := covNewRepo(covOwnerGroup())
	adminAPIErrRepo.listAdminMembersErr = httpx.InvalidRequest("bad")
	adminMissingRepo := covNewRepo(covOwnerGroup())
	adminMissingRepo.listAdminMembersErr = ErrGroupNotFound
	adminCursorRepo := covNewRepo(covOwnerGroup())
	adminCursorRepo.listAdminMembersErr = ErrInvalidCursor
	adminGenericRepo := covNewRepo(covOwnerGroup())
	adminGenericRepo.listAdminMembersErr = errors.New("boom")

	covRunCases(t, []covCase{
		// GetSettings
		{name: "settings invalid json", repo: covNewRepo(covOwnerGroup()), fn: func(h *Handler) http.HandlerFunc { return h.GetSettings }, body: "{", want: httpx.CodeInvalidRequest},
		{name: "settings api error", repo: apiErrRepo, fn: func(h *Handler) http.HandlerFunc { return h.GetSettings }, body: `{"group_id":"g1"}`, want: httpx.CodeInvalidRequest},
		{name: "settings generic error", repo: genericRepo, fn: func(h *Handler) http.HandlerFunc { return h.GetSettings }, body: `{"group_id":"g1"}`, want: httpx.CodeInternalUnavailable},
		{name: "settings not found", repo: missingRepo, fn: func(h *Handler) http.HandlerFunc { return h.GetSettings }, body: `{"group_id":"g1"}`, want: httpx.CodeNotFound},
		// ListAdminMembers
		{name: "admin members invalid json", repo: covNewRepo(covOwnerGroup()), fn: func(h *Handler) http.HandlerFunc { return h.ListAdminMembers }, body: "{", want: httpx.CodeInvalidRequest},
		{name: "admin members api error", repo: adminAPIErrRepo, fn: func(h *Handler) http.HandlerFunc { return h.ListAdminMembers }, body: `{"group_id":"g1"}`, want: httpx.CodeInvalidRequest},
		{name: "admin members not found", repo: adminMissingRepo, fn: func(h *Handler) http.HandlerFunc { return h.ListAdminMembers }, body: `{"group_id":"g1"}`, want: httpx.CodeNotFound},
		{name: "admin members invalid cursor", repo: adminCursorRepo, fn: func(h *Handler) http.HandlerFunc { return h.ListAdminMembers }, body: `{"group_id":"g1"}`, want: httpx.CodeInvalidRequest},
		{name: "admin members generic error", repo: adminGenericRepo, fn: func(h *Handler) http.HandlerFunc { return h.ListAdminMembers }, body: `{"group_id":"g1"}`, want: httpx.CodeInternalUnavailable},
	})
}

func TestCovHandlerRecallGroupMessageAsAdminErrorPaths(t *testing.T) {
	orig := RecallMessageAsAdmin
	t.Cleanup(func() { RecallMessageAsAdmin = orig })

	// 未注入实现 → 服务返回 10600。
	RecallMessageAsAdmin = nil
	code := covRunHandler(t, covNewRepo(covOwnerGroup()), func(h *Handler) http.HandlerFunc { return h.RecallGroupMessageAsAdmin },
		`{"group_id":"g1","message_id":"m1"}`, "user_alice", false)
	if code != httpx.CodeInternalUnavailable {
		t.Fatalf("code = %d, want 10600", code)
	}

	// 实现返回 APIError → 原样透传。
	RecallMessageAsAdmin = func(context.Context, string, string, string) error {
		return httpx.InvalidRequest("bad")
	}
	code = covRunHandler(t, covNewRepo(covOwnerGroup()), func(h *Handler) http.HandlerFunc { return h.RecallGroupMessageAsAdmin },
		`{"group_id":"g1","message_id":"m1"}`, "user_alice", false)
	if code != httpx.CodeInvalidRequest {
		t.Fatalf("code = %d, want 10000", code)
	}

	// 实现返回普通错误 → 10600。
	RecallMessageAsAdmin = func(context.Context, string, string, string) error {
		return errors.New("boom")
	}
	code = covRunHandler(t, covNewRepo(covOwnerGroup()), func(h *Handler) http.HandlerFunc { return h.RecallGroupMessageAsAdmin },
		`{"group_id":"g1","message_id":"m1"}`, "user_alice", false)
	if code != httpx.CodeInternalUnavailable {
		t.Fatalf("code = %d, want 10600", code)
	}

	// 参数校验分支
	covRunCases(t, []covCase{
		{name: "recall invalid json", repo: covNewRepo(covOwnerGroup()), fn: func(h *Handler) http.HandlerFunc { return h.RecallGroupMessageAsAdmin }, body: "{", want: httpx.CodeInvalidRequest},
		{name: "recall missing message id", repo: covNewRepo(covOwnerGroup()), fn: func(h *Handler) http.HandlerFunc { return h.RecallGroupMessageAsAdmin }, body: `{"group_id":"g1"}`, want: httpx.CodeInvalidRequest},
	})
}
