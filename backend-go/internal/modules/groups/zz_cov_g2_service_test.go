package groups

import (
	"context"
	"errors"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"hotspot/backend-go/internal/platform/actor"
	"hotspot/backend-go/internal/platform/appconfig"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/geo"
	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/types"
)

// ── 测试替身 ─────────────────────────────────────────────────────────────

type covLocationProvider struct {
	point geo.Point
	found bool
}

func (p covLocationProvider) RecentLocation(context.Context, string) (geo.Point, bool) {
	return p.point, p.found
}

type covAppConfigSource struct{ cfg appconfig.AppConfig }

func (s covAppConfigSource) Get(context.Context) appconfig.AppConfig { return s.cfg }

type covActorProvider struct {
	info actor.Info
	err  error
}

func (p covActorProvider) GetActorInfo(context.Context, string) (actor.Info, error) {
	return p.info, p.err
}

type covAvatarResolver struct {
	err  error
	path string
}

func (r covAvatarResolver) ResolveGroupAvatarImage(context.Context, string, string) error {
	return r.err
}

func (r covAvatarResolver) AvatarPublicPath(fileID string) string {
	if r.path != "" {
		return r.path
	}
	return "avatar/" + fileID
}

type covNotifyWriter struct {
	err    error
	lastID string
}

func (w covNotifyWriter) AppendGroupInviteReceived(context.Context, string, types.UserInfo, string, string, string, *string) (string, error) {
	return w.lastID, w.err
}

func (w covNotifyWriter) AppendGroupApplicationReceived(context.Context, string, types.UserInfo, string, string, string, *string) (string, error) {
	return w.lastID, w.err
}

func (w covNotifyWriter) AppendGroupApplicationAccepted(context.Context, string, string, string, string) (string, error) {
	return w.lastID, w.err
}

func (w covNotifyWriter) AppendGroupApplicationRejected(context.Context, string, string, string, string, *string) (string, error) {
	return w.lastID, w.err
}

type covAuditWriter struct{ err error }

func (w covAuditWriter) AppendGroupDismissAudit(context.Context, string, string, string, string) error {
	return w.err
}

type covAppErrRepo struct {
	*covRepo
	getApplicationErr error
}

func (r *covAppErrRepo) GetApplication(context.Context, string) (GroupApplicationItem, bool, error) {
	return GroupApplicationItem{}, false, r.getApplicationErr
}

type covInviteErrRepo struct {
	*covRepo
	getInviteErr error
}

func (r *covInviteErrRepo) GetInvite(context.Context, string) (GroupInviteItem, string, bool, error) {
	return GroupInviteItem{}, "", false, r.getInviteErr
}

type covSeqGetRepo struct {
	*covRepo
	calls     int
	secondErr error
}

func (r *covSeqGetRepo) Get(ctx context.Context, groupID string) (GroupItem, bool, error) {
	r.calls++
	if r.calls == 2 {
		if r.secondErr != nil {
			return GroupItem{}, false, r.secondErr
		}
		return GroupItem{}, false, nil
	}
	return r.covRepo.Get(ctx, groupID)
}

type covSeqListMembersRepo struct {
	*covRepo
	calls      int
	firstItems []GroupMemberItem
	secondErr  error
}

func (r *covSeqListMembersRepo) ListMembers(context.Context, string, *GroupRole, int, string) ([]GroupMemberItem, string, error) {
	r.calls++
	if r.calls >= 2 {
		if r.secondErr != nil {
			return nil, "", r.secondErr
		}
		return nil, "", nil
	}
	return r.firstItems, "", nil
}

func covCtx(userID string) context.Context {
	return platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: userID})
}

// ── 基础 helper / setter / 边界 ─────────────────────────────────────────

func TestCovServiceSettersAndHelpers(t *testing.T) {
	svc := &Service{}
	svc.SetTxRunner(func(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) })
	svc.SetGroupAvatarMediaResolver(nil)
	svc.SetGroupDismissAuditWriter(nil)
	svc.SetLocationProvider(nil)

	var nilSvc *Service
	nilSvc.SetTxRunner(nil)
	nilSvc.SetGroupDismissAuditWriter(nil)

	redactGroupFields(nil)

	if !equalStringPtr(nil, nil) || equalStringPtr(nil, strPtrCov("a")) || equalStringPtr(strPtrCov("a"), nil) {
		t.Fatal("nil pointer equality contract broken")
	}
	if !equalStringPtr(strPtrCov("a"), strPtrCov("a")) || equalStringPtr(strPtrCov("a"), strPtrCov("b")) {
		t.Fatal("value pointer equality contract broken")
	}

	pending := defaultPendingStatus(nil)
	if pending == nil || *pending != ApplicationStatusPending {
		t.Fatalf("default status = %#v", pending)
	}
	if all := defaultPendingStatus(&[]ApplicationStatus{ApplicationStatus("all")}[0]); all != nil {
		t.Fatalf("all status = %#v, want nil", all)
	}
	accepted := ApplicationStatusAccepted
	if got := defaultPendingStatus(&accepted); got == nil || *got != ApplicationStatusAccepted {
		t.Fatalf("explicit status = %#v", got)
	}

	if cap := svc.groupMembersCap(context.Background()); cap != 0 {
		t.Fatalf("cap without appconfig = %d", cap)
	}
	svc.SetAppConfig(covAppConfigSource{cfg: appconfig.AppConfig{Limits: appconfig.Limits{MaxGroupMembers: 50}}})
	if cap := svc.groupMembersCap(context.Background()); cap != 50 {
		t.Fatalf("cap with appconfig = %d", cap)
	}
}

func TestCovServiceResolveReferencePoint(t *testing.T) {
	svc := NewService()
	svc.SetLocationProvider(covLocationProvider{point: geo.Point{Latitude: 11, Longitude: 22}, found: true})
	pt := svc.resolveReferencePoint(covCtx("user_alice"))
	if pt.Latitude != 11 || pt.Longitude != 22 {
		t.Fatalf("location provider point = %#v", pt)
	}

	// provider 未命中 → 回退 appConfig 默认位置。
	svc = NewService()
	svc.SetLocationProvider(covLocationProvider{found: false})
	svc.SetAppConfig(covAppConfigSource{cfg: appconfig.AppConfig{Defaults: appconfig.Defaults{DefaultLocation: appconfig.Location{Point: appconfig.GeoPoint{Latitude: 33, Longitude: 44}}}}})
	pt = svc.resolveReferencePoint(covCtx("user_alice"))
	if pt.Latitude != 33 || pt.Longitude != 44 {
		t.Fatalf("appconfig fallback point = %#v", pt)
	}
}

func TestCovServiceBuildGroupUpdatedFields(t *testing.T) {
	before := GroupItem{
		Group:          GroupInfo{GroupID: "g1", Name: "before", AvatarURL: strPtrCov("a1")},
		Description:    strPtrCov("d1"),
		Tags:           []string{"t1"},
		JoinMethod:     GroupJoinMethodDirect,
		NextActivityAt: strPtrCov("2026-01-01T00:00:00Z"),
	}
	after := GroupItem{
		Group:            GroupInfo{GroupID: "g1", Name: "after", AvatarURL: strPtrCov("a2")},
		Description:      strPtrCov("d2"),
		Tags:             []string{"t2"},
		JoinMethod:       GroupJoinMethodInviteOnly,
		RequiresApproval: true,
		NextActivityAt:   strPtrCov("2026-02-01T00:00:00Z"),
		Location:         &RelativeLocation{LocationName: "loc"},
	}
	fields := buildGroupUpdatedFields(before, after, UpdateGroupRequest{GroupID: "g1", Location: &Location{LocationName: "loc"}})
	for _, key := range []string{"name", "avatar_url", "description", "tags", "join_method", "requires_approval", "next_activity_at", "location"} {
		if _, ok := fields[key]; !ok {
			t.Fatalf("missing changed field %q in %#v", key, fields)
		}
	}
	// 无变化 + 未带 location → 空 map。
	empty := buildGroupUpdatedFields(after, after, UpdateGroupRequest{GroupID: "g1"})
	if len(empty) != 0 {
		t.Fatalf("unchanged fields = %#v", empty)
	}
	// description 由 nil 变有值。
	descFields := buildGroupUpdatedFields(GroupItem{}, GroupItem{Description: strPtrCov("x")}, UpdateGroupRequest{})
	if got, ok := descFields["description"]; !ok || got != "x" {
		t.Fatalf("description diff = %#v", descFields)
	}
	// next_activity_at 由有值变 nil。
	nextFields := buildGroupUpdatedFields(before, GroupItem{}, UpdateGroupRequest{})
	if got, ok := nextFields["next_activity_at"]; !ok || got != "" {
		t.Fatalf("next_activity diff = %#v", nextFields)
	}
}

func TestCovServiceAvatarInput(t *testing.T) {
	ctx := covCtx("user_alice")
	svc := NewService()
	// resolver 未注入：原样返回。
	if got, err := svc.resolveGroupAvatarInput(ctx, "file_1", "user_alice", "create"); err != nil || got != "file_1" {
		t.Fatalf("nil resolver = %q, %v", got, err)
	}
	// 空 fileID 直接返回。
	svc.SetGroupAvatarMediaResolver(covAvatarResolver{})
	if got, err := svc.resolveGroupAvatarInput(ctx, "", "user_alice", "create"); err != nil || got != "" {
		t.Fatalf("empty file = %q, %v", got, err)
	}
	// 校验失败 → APIError（10000）。
	svc.SetGroupAvatarMediaResolver(covAvatarResolver{err: errors.New("denied")})
	if _, err := svc.resolveGroupAvatarInput(ctx, "file_1", "user_alice", "create"); err == nil {
		t.Fatal("expected resolver error")
	} else if apiErr, ok := err.(*httpx.APIError); !ok || apiErr.Code != httpx.CodeInvalidRequest {
		t.Fatalf("resolver err = %#v", err)
	}
	// 校验成功 → 代理路径。
	svc.SetGroupAvatarMediaResolver(covAvatarResolver{path: "avatar/proxy"})
	if got, err := svc.resolveGroupAvatarInput(ctx, "file_1", "user_alice", "create"); err != nil || got != "avatar/proxy" {
		t.Fatalf("resolved avatar = %q, %v", got, err)
	}

	// Create：resolver 成功路径（含密码哈希）与失败路径。
	createRepo := covNewRepo(covOwnerGroup())
	createRepo.createResult = GroupItem{Group: GroupInfo{GroupID: "g1"}}
	createSvc := NewServiceWithRepo(createRepo)
	createSvc.SetGroupAvatarMediaResolver(covAvatarResolver{path: "avatar/proxy"})
	item, err := createSvc.Create(ctx, CreateGroupRequest{Name: "n", AvatarFileID: "file_1", Location: Location{LocationName: "loc"}, Password: "abcdefgh"})
	if err != nil || item.Group.GroupID != "g1" {
		t.Fatalf("create with avatar = %#v, %v", item, err)
	}
	failRepo := covNewRepo(covOwnerGroup())
	failSvc := NewServiceWithRepo(failRepo)
	failSvc.SetGroupAvatarMediaResolver(covAvatarResolver{err: errors.New("denied")})
	if _, err := failSvc.Create(ctx, CreateGroupRequest{Name: "n", AvatarFileID: "file_1", Location: Location{LocationName: "loc"}}); err == nil {
		t.Fatal("expected create avatar error")
	}
}

func TestCovServiceUpdateBranches(t *testing.T) {
	ctx := covCtx("user_alice")
	baseReq := UpdateGroupRequest{GroupID: "g1"}

	// 头像校验失败 → 直接返回错误。
	avatarFailRepo := covNewRepo(covOwnerGroup())
	avatarFailSvc := NewServiceWithRepo(avatarFailRepo)
	avatarFailSvc.SetGroupAvatarMediaResolver(covAvatarResolver{err: errors.New("denied")})
	if _, err := avatarFailSvc.Update(ctx, UpdateGroupRequest{GroupID: "g1", AvatarFileID: "file_1"}); err == nil {
		t.Fatal("expected avatar error on update")
	}

	// 密码哈希 + 校验通过（location 走校验分支）。
	password := "abcdefgh"
	passwordRepo := covNewRepo(covOwnerGroup())
	passwordSvc := NewServiceWithRepo(passwordRepo)
	if _, err := passwordSvc.Update(ctx, UpdateGroupRequest{GroupID: "g1", Password: &password, Location: &Location{LocationName: "loc"}}); err != nil {
		t.Fatalf("update with password: %v", err)
	}

	// runInTx 成功路径。
	txRepo := covNewRepo(covOwnerGroup())
	txSvc := NewServiceWithRepo(txRepo)
	txSvc.SetTxRunner(func(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) })
	if _, err := txSvc.Update(ctx, baseReq); err != nil {
		t.Fatalf("update with tx runner: %v", err)
	}
	// runInTx 失败路径。
	errRepo := covNewRepo(covOwnerGroup())
	errSvc := NewServiceWithRepo(errRepo)
	errSvc.SetTxRunner(func(context.Context, func(context.Context) error) error { return errors.New("tx failed") })
	if _, err := errSvc.Update(ctx, baseReq); err == nil {
		t.Fatal("expected tx runner error")
	}

	// 刷新阶段 Get 错误 / 未命中。
	refreshErrRepo := &covSeqGetRepo{covRepo: covNewRepo(covOwnerGroup()), secondErr: errors.New("refresh failed")}
	if _, err := NewServiceWithRepo(refreshErrRepo).Update(ctx, baseReq); err == nil {
		t.Fatal("expected refresh error")
	}
	refreshMissRepo := &covSeqGetRepo{covRepo: covNewRepo(covOwnerGroup())}
	if _, err := NewServiceWithRepo(refreshMissRepo).Update(ctx, baseReq); !errors.Is(err, ErrGroupNotFound) {
		t.Fatalf("refresh miss err = %v", err)
	}
}

func TestCovServiceDismissAuditAndAdmin(t *testing.T) {
	ctx := covCtx("user_alice")
	auditErrSvc := NewServiceWithRepo(covNewRepo(covOwnerGroup()))
	auditErrSvc.SetGroupDismissAuditWriter(covAuditWriter{err: errors.New("audit down")})
	if err := auditErrSvc.Dismiss(ctx, DismissGroupRequest{GroupID: "g1"}); err != nil {
		t.Fatalf("dismiss with failing audit must succeed: %v", err)
	}

	// DismissAsAdmin：Get 错误。
	getErrRepo := covNewRepo(covOwnerGroup())
	getErrRepo.getFound = false
	getErrRepo.getErr = errors.New("load failed")
	if _, err := NewServiceWithRepo(getErrRepo).DismissAsAdmin(ctx, "op", "g1", "reason"); err == nil {
		t.Fatal("expected dismiss as admin load error")
	}
	// 群不存在。
	missingRepo := covNewRepo(covOwnerGroup())
	missingRepo.getFound = false
	if _, err := NewServiceWithRepo(missingRepo).DismissAsAdmin(ctx, "op", "g1", "reason"); !errors.Is(err, ErrGroupNotFound) {
		t.Fatalf("missing group err = %v", err)
	}
	// 仓储错误。
	repoErrRepo := covNewRepo(covOwnerGroup())
	repoErrRepo.dismissAsAdminErr = errors.New("db down")
	if _, err := NewServiceWithRepo(repoErrRepo).DismissAsAdmin(ctx, "op", "g1", "reason"); err == nil {
		t.Fatal("expected dismiss as admin repo error")
	}
	// 未命中（已解散）。
	notHitRepo := covNewRepo(covOwnerGroup())
	if _, err := NewServiceWithRepo(notHitRepo).DismissAsAdmin(ctx, "op", "g1", "reason"); !errors.Is(err, ErrGroupNotFound) {
		t.Fatalf("not hit err = %v", err)
	}
	// 审计失败不影响返回。
	auditFailRepo := covNewRepo(covOwnerGroup())
	auditFailRepo.dismissAsAdminHit = true
	auditFailSvc := NewServiceWithRepo(auditFailRepo)
	auditFailSvc.SetGroupDismissAuditWriter(covAuditWriter{err: errors.New("audit down")})
	info, err := auditFailSvc.DismissAsAdmin(ctx, "op", "g1", "reason")
	if err != nil || info.GroupID != "g1" {
		t.Fatalf("dismiss as admin with failing audit = %#v, %v", info, err)
	}
}

func TestCovServiceJoinGroupPasswordBranches(t *testing.T) {
	ctx := covCtx("user_alice")
	passwordGroup := func() GroupItem {
		return GroupItem{Group: GroupInfo{GroupID: "g1"}, JoinMethod: GroupJoinMethodPassword}
	}

	// 群无密码哈希 → ErrInvalidPassword。
	noHashRepo := covNewRepo(passwordGroup())
	plain := "abcdefgh"
	if _, err := NewServiceWithRepo(noHashRepo).JoinGroup(ctx, EnterGroupRequest{GroupID: "g1", Password: &plain}); !errors.Is(err, ErrInvalidPassword) {
		t.Fatalf("missing hash err = %v", err)
	}
	// 取哈希失败。
	hashErrRepo := covNewRepo(passwordGroup())
	hashErrRepo.passwordHashErr = errors.New("hash load failed")
	if _, err := NewServiceWithRepo(hashErrRepo).JoinGroup(ctx, EnterGroupRequest{GroupID: "g1", Password: &plain}); err == nil {
		t.Fatal("expected hash load error")
	}
	// 哈希比对失败。
	hash, err := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	hashStr := string(hash)
	wrongRepo := covNewRepo(passwordGroup())
	wrongRepo.passwordHash = &hashStr
	if _, err := NewServiceWithRepo(wrongRepo).JoinGroup(ctx, EnterGroupRequest{GroupID: "g1", Password: &plain}); !errors.Is(err, ErrInvalidPassword) {
		t.Fatalf("wrong password err = %v", err)
	}
	// 哈希比对成功 → 入群。
	okRepo := covNewRepo(passwordGroup())
	okRepo.passwordHash = &hashStr
	okRepo.joinResult = EnterGroupResult{Status: EnterGroupStatusJoined}
	correct := "correct-password"
	result, err := NewServiceWithRepo(okRepo).JoinGroup(ctx, EnterGroupRequest{GroupID: "g1", Password: &correct})
	if err != nil || result.Status != EnterGroupStatusJoined {
		t.Fatalf("join with password = %#v, %v", result, err)
	}
}

func TestCovServiceMemberFaultBranches(t *testing.T) {
	ctx := covCtx("user_alice")
	ownerID := "u2"

	// SetMemberRole：Get 错误 / IsMember 错误。
	roleGetErrRepo := covNewRepo(covOwnerGroup())
	roleGetErrRepo.getFound = false
	roleGetErrRepo.getErr = errors.New("load failed")
	if err := NewServiceWithRepo(roleGetErrRepo).SetMemberRole(ctx, SetMemberRoleRequest{GroupID: "g1", UserID: "u2", Role: GroupRoleAdmin}); err == nil {
		t.Fatal("expected set role load error")
	}
	roleMemberErrRepo := covNewRepo(covOwnerGroup())
	roleMemberErrRepo.isMemberErr = errors.New("probe failed")
	if err := NewServiceWithRepo(roleMemberErrRepo).SetMemberRole(ctx, SetMemberRoleRequest{GroupID: "g1", UserID: "u2", Role: GroupRoleAdmin}); err == nil {
		t.Fatal("expected set role member probe error")
	}

	// MuteMember：Get 错误 / IsMember 错误。
	muteGetErrRepo := covNewRepo(covOwnerGroup())
	muteGetErrRepo.getFound = false
	muteGetErrRepo.getErr = errors.New("load failed")
	if err := NewServiceWithRepo(muteGetErrRepo).MuteMember(ctx, MuteMemberRequest{GroupID: "g1", UserID: "u2", DurationMinutes: 10}); err == nil {
		t.Fatal("expected mute load error")
	}
	muteMemberErrRepo := covNewRepo(covOwnerGroup())
	muteMemberErrRepo.isMemberErr = errors.New("probe failed")
	if err := NewServiceWithRepo(muteMemberErrRepo).MuteMember(ctx, MuteMemberRequest{GroupID: "g1", UserID: "u2", DurationMinutes: 10}); err == nil {
		t.Fatal("expected mute member probe error")
	}

	// UnmuteMember：Get 错误 / 不存在 / IsMember 错误 / 正常成功广播。
	unmuteGetErrRepo := covNewRepo(covOwnerGroup())
	unmuteGetErrRepo.getFound = false
	unmuteGetErrRepo.getErr = errors.New("load failed")
	if err := NewServiceWithRepo(unmuteGetErrRepo).UnmuteMember(ctx, UnmuteMemberRequest{GroupID: "g1", UserID: "u2"}); err == nil {
		t.Fatal("expected unmute load error")
	}
	unmuteMissingRepo := covNewRepo(covOwnerGroup())
	unmuteMissingRepo.getFound = false
	if err := NewServiceWithRepo(unmuteMissingRepo).UnmuteMember(ctx, UnmuteMemberRequest{GroupID: "g1", UserID: "u2"}); !errors.Is(err, ErrGroupNotFound) {
		t.Fatalf("unmute missing err = %v", err)
	}
	unmuteMemberErrRepo := covNewRepo(covOwnerGroup())
	unmuteMemberErrRepo.isMemberErr = errors.New("probe failed")
	if err := NewServiceWithRepo(unmuteMemberErrRepo).UnmuteMember(ctx, UnmuteMemberRequest{GroupID: "g1", UserID: "u2"}); err == nil {
		t.Fatal("expected unmute member probe error")
	}
	unmuteOKRepo := covNewRepo(covOwnerGroup())
	unmuteOKRepo.isMemberResult = true
	if err := NewServiceWithRepo(unmuteOKRepo).UnmuteMember(ctx, UnmuteMemberRequest{GroupID: "g1", UserID: "u2"}); err != nil {
		t.Fatalf("unmute success: %v", err)
	}

	// SetMemberBan：Get 错误。
	banGetErrRepo := covNewRepo(covAdminGroup())
	banGetErrRepo.getFound = false
	banGetErrRepo.getErr = errors.New("load failed")
	if err := NewServiceWithRepo(banGetErrRepo).SetMemberBan(ctx, SetMemberBanRequest{GroupID: "g1", UserID: "u2", Active: true}); err == nil {
		t.Fatal("expected ban load error")
	}

	// ListBannedMembers：Get 错误。
	bannedGetErrRepo := covNewRepo(covOwnerGroup())
	bannedGetErrRepo.getFound = false
	bannedGetErrRepo.getErr = errors.New("load failed")
	if _, _, err := NewServiceWithRepo(bannedGetErrRepo).ListBannedMembers(ctx, BannedMembersRequest{GroupID: "g1"}); err == nil {
		t.Fatal("expected banned members load error")
	}

	// RemoveMember：Get 错误。
	removeGetErrRepo := covNewRepo(covOwnerGroup())
	removeGetErrRepo.getFound = false
	removeGetErrRepo.getErr = errors.New("load failed")
	if err := NewServiceWithRepo(removeGetErrRepo).RemoveMember(ctx, RemoveMemberRequest{GroupID: "g1", UserID: "u2"}); err == nil {
		t.Fatal("expected remove load error")
	}

	// TransferOwnership：Get 错误 / IsMember 错误。
	transferGetErrRepo := covNewRepo(covOwnerGroup())
	transferGetErrRepo.getFound = false
	transferGetErrRepo.getErr = errors.New("load failed")
	if err := NewServiceWithRepo(transferGetErrRepo).TransferOwnership(ctx, TransferOwnershipRequest{GroupID: "g1", NewOwnerID: ownerID}); err == nil {
		t.Fatal("expected transfer load error")
	}
	transferMemberErrRepo := covNewRepo(covOwnerGroup())
	transferMemberErrRepo.isMemberErr = errors.New("probe failed")
	if err := NewServiceWithRepo(transferMemberErrRepo).TransferOwnership(ctx, TransferOwnershipRequest{GroupID: "g1", NewOwnerID: ownerID}); err == nil {
		t.Fatal("expected transfer member probe error")
	}
}

func TestCovServiceApplicationFaultBranches(t *testing.T) {
	ctx := covCtx("user_alice")
	expiredApp := GroupApplicationItem{
		ApplicationID: "a1",
		GroupID:       "g1",
		Applicant:     UserInfo{UserID: "user_bob"},
		Status:        ApplicationStatusPending,
		CreatedAt:     time.Now().UTC().Add(-100 * time.Hour).Format(time.RFC3339Nano),
	}

	// GetApplication 错误。
	appErrRepo := &covAppErrRepo{covRepo: covNewRepo(covOwnerGroup()), getApplicationErr: errors.New("lookup failed")}
	if err := NewServiceWithRepo(appErrRepo).ApproveApplication(ctx, ApproveGroupApplicationRequest{GroupID: "g1", ApplicationID: "a1"}); err == nil {
		t.Fatal("expected application lookup error")
	}
	// 群 Get 错误。
	groupErrRepo := covNewRepo(covOwnerGroup())
	groupErrRepo.getFound = false
	groupErrRepo.getErr = errors.New("load failed")
	groupErrRepo.listApplicationsItems = []GroupApplicationItem{expiredApp}
	if _, _, err := NewServiceWithRepo(groupErrRepo).findApplicationForApprover(ctx, "user_alice", "a1"); err == nil {
		t.Fatal("expected approver group load error")
	}
	// 群不存在。
	groupMissingRepo := covNewRepo(covOwnerGroup())
	groupMissingRepo.getFound = false
	groupMissingRepo.listApplicationsItems = []GroupApplicationItem{expiredApp}
	if _, _, err := NewServiceWithRepo(groupMissingRepo).findApplicationForApprover(ctx, "user_alice", "a1"); !errors.Is(err, ErrGroupNotFound) {
		t.Fatalf("approver missing group err = %v", err)
	}
	// 申请过期且撤销失败。
	cancelErrRepo := covNewRepo(GroupItem{Group: GroupInfo{GroupID: "g1"}, ViewerState: &GroupViewerState{Role: GroupRoleOwner}})
	cancelErrRepo.listApplicationsItems = []GroupApplicationItem{expiredApp}
	cancelErrRepo.cancelApplicationErr = errors.New("cancel failed")
	if _, _, err := NewServiceWithRepo(cancelErrRepo).findApplicationForApprover(ctx, "user_alice", "a1"); err == nil {
		t.Fatal("expected expired application cancel error")
	}

	// CancelApplication：GetApplication 错误 / 群锚点不匹配。
	cancelAppErrRepo := &covAppErrRepo{covRepo: covNewRepo(covOwnerGroup()), getApplicationErr: errors.New("lookup failed")}
	if err := NewServiceWithRepo(cancelAppErrRepo).CancelApplication(ctx, CancelApplicationRequest{GroupID: "g1", ApplicationID: "a1"}); err == nil {
		t.Fatal("expected cancel application lookup error")
	}
	mismatchRepo := covNewRepo(covOwnerGroup())
	mismatchRepo.listApplicationsItems = []GroupApplicationItem{{ApplicationID: "a1", GroupID: "g2", Applicant: UserInfo{UserID: "user_alice"}, Status: ApplicationStatusPending}}
	if err := NewServiceWithRepo(mismatchRepo).CancelApplication(ctx, CancelApplicationRequest{GroupID: "g1", ApplicationID: "a1"}); !errors.Is(err, ErrApplicationNotFound) {
		t.Fatalf("group anchor mismatch err = %v", err)
	}

	// UpdatePreferences：Get 错误。
	prefErrRepo := covNewRepo(covOwnerGroup())
	prefErrRepo.getFound = false
	prefErrRepo.getErr = errors.New("load failed")
	if err := NewServiceWithRepo(prefErrRepo).UpdatePreferences(ctx, UpdateGroupPreferencesRequest{GroupID: "g1"}); err == nil {
		t.Fatal("expected preferences load error")
	}
}

func TestCovServiceInviteExpiredBranches(t *testing.T) {
	ctx := covCtx("user_alice")
	expiredAt := time.Now().UTC().Add(-time.Hour).Format(time.RFC3339Nano)
	pastInvite := func() *covRepo {
		repo := covNewRepo(covOwnerGroup())
		repo.listMyInvitesItems = []GroupInviteItem{{
			InviteID:  "inv_1",
			Group:     GroupInfo{GroupID: "g1"},
			Inviter:   UserInfo{UserID: "user_bob"},
			Status:    ApplicationStatusPending,
			ExpiresAt: &expiredAt,
		}}
		return repo
	}
	// 接受邀请：已过期 → 撤销 + ErrInviteExpired。
	if err := NewServiceWithRepo(pastInvite()).AcceptInvite(ctx, AcceptInviteRequest{InviteID: "inv_1"}); !errors.Is(err, ErrInviteExpired) {
		t.Fatalf("accept expired err = %v", err)
	}
	// 拒绝邀请：已过期 → 撤销 + ErrInviteExpired。
	if err := NewServiceWithRepo(pastInvite()).RejectInvite(ctx, RejectInviteRequest{InviteID: "inv_1"}); !errors.Is(err, ErrInviteExpired) {
		t.Fatalf("reject expired err = %v", err)
	}
	// GetInvite 错误透传。
	inviteErrRepo := &covInviteErrRepo{covRepo: covNewRepo(covOwnerGroup()), getInviteErr: errors.New("lookup failed")}
	if err := NewServiceWithRepo(inviteErrRepo).AcceptInvite(ctx, AcceptInviteRequest{InviteID: "inv_1"}); err == nil {
		t.Fatal("expected accept invite lookup error")
	}
	if err := NewServiceWithRepo(inviteErrRepo).RevokeInvite(ctx, RevokeSentInviteRequest{GroupID: "g1", InviteID: "inv_1"}); err == nil {
		t.Fatal("expected revoke invite lookup error")
	}
}

func TestCovServiceNotificationFaultBranches(t *testing.T) {
	ctx := context.Background()
	notifyErr := covNotifyWriter{err: errors.New("notify down")}
	application := GroupApplicationItem{ApplicationID: "a1", GroupID: "g1", Applicant: UserInfo{UserID: "user_bob"}}
	group := GroupItem{Group: GroupInfo{GroupID: "g1", Name: "G"}}
	ownerMember := GroupMemberItem{User: UserInfo{UserID: "user_owner"}}

	// Cancelled：群加载失败。
	cancelledLoadErrRepo := covNewRepo(group)
	cancelledLoadErrRepo.getFound = false
	cancelledLoadErrRepo.getErr = errors.New("load failed")
	NewServiceWithRepo(cancelledLoadErrRepo).notifyGroupApplicationCancelled(ctx, application, "")
	// Cancelled：owner 列表失败。
	cancelledListErrRepo := covNewRepo(group)
	cancelledListErrRepo.listMembersErr = errors.New("list failed")
	NewServiceWithRepo(cancelledListErrRepo).notifyGroupApplicationCancelled(ctx, application, "x")
	// Cancelled：admin 列表失败 + 正常带 reason 广播。
	NewServiceWithRepo(&covSeqListMembersRepo{covRepo: covNewRepo(group), secondErr: errors.New("admin list failed")}).notifyGroupApplicationCancelled(ctx, application, "")
	okRepo := covNewRepo(group)
	okRepo.listMembers = []GroupMemberItem{ownerMember}
	NewServiceWithRepo(okRepo).notifyGroupApplicationCancelled(ctx, application, "reason")

	// Expired：owner 列表失败 / admin 列表失败 / 正常广播。
	expiredListErrRepo := covNewRepo(group)
	expiredListErrRepo.listMembersErr = errors.New("list failed")
	NewServiceWithRepo(expiredListErrRepo).notifyGroupApplicationExpired(ctx, group, application)
	NewServiceWithRepo(&covSeqListMembersRepo{covRepo: covNewRepo(group), secondErr: errors.New("admin list failed")}).notifyGroupApplicationExpired(ctx, group, application)
	NewServiceWithRepo(okRepo).notifyGroupApplicationExpired(ctx, group, application)

	// Rejected：通知写入失败；reason 非空。
	NewServiceWithDeps(covNewRepo(group), notifyErr, nil).notifyGroupApplicationRejected(ctx, group, application, strPtrCov("bad"))
	// Invite received：通知写入失败；message 非空。
	req := InviteMemberRequest{GroupID: "g1", UserID: "user_bob", Message: strPtrCov("hi")}
	NewServiceWithDeps(covNewRepo(group), notifyErr, nil).notifyGroupInviteReceived(ctx, group, "user_alice", req, Invitation{})
	// Application received：applicationID 为空 → 早早返回。
	NewServiceWithDeps(covNewRepo(group), notifyErr, nil).notifyGroupApplicationReceived(ctx, group, "user_bob", nil, nil)
	// Application received：owner 列表失败 / admin 列表失败 / 写入失败。
	receivedListErrRepo := covNewRepo(group)
	receivedListErrRepo.listMembersErr = errors.New("list failed")
	NewServiceWithDeps(receivedListErrRepo, notifyErr, nil).notifyGroupApplicationReceived(ctx, group, "user_bob", strPtrCov("a1"), strPtrCov("msg"))
	NewServiceWithDeps(&covSeqListMembersRepo{covRepo: covNewRepo(group), secondErr: errors.New("admin list failed")}, notifyErr, nil).notifyGroupApplicationReceived(ctx, group, "user_bob", strPtrCov("a1"), nil)
	receivedOKRepo := covNewRepo(group)
	receivedOKRepo.listMembers = []GroupMemberItem{ownerMember}
	NewServiceWithDeps(receivedOKRepo, notifyErr, nil).notifyGroupApplicationReceived(ctx, group, "user_bob", strPtrCov("a1"), strPtrCov("msg"))
	// Application accepted：通知写入失败。
	NewServiceWithDeps(covNewRepo(group), notifyErr, nil).notifyGroupApplicationAccepted(ctx, group, application)
	// resolveUserInfo：actor provider 错误回退 userID。
	if info := NewServiceWithDeps(covNewRepo(group), notifyErr, covActorProvider{err: errors.New("actor down")}).resolveUserInfo(ctx, "user_x"); info.UserID != "user_x" || info.Nickname != "user_x" {
		t.Fatalf("resolve user info fallback = %#v", info)
	}
	if info := NewServiceWithDeps(covNewRepo(group), notifyErr, covActorProvider{info: actor.Info{UserID: "user_x", Nickname: "Nick", AvatarURL: "a"}}).resolveUserInfo(ctx, "user_x"); info.Nickname != "Nick" || info.AvatarURL == nil || *info.AvatarURL != "a" {
		t.Fatalf("resolve user info = %#v", info)
	}
}

func TestCovServiceTxRunnerGuard(t *testing.T) {
	// SetTxRunner 空实现：Update 直接执行（回退路径）。
	svc := NewServiceWithRepo(covNewRepo(covOwnerGroup()))
	svc.SetTxRunner(nil)
	if _, err := svc.Update(covCtx("user_alice"), UpdateGroupRequest{GroupID: "g1"}); err != nil {
		t.Fatalf("update without tx runner: %v", err)
	}
	var _ persistence.TxRunner
}
