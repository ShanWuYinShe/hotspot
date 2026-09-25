package users

import "errors"

var (
	ErrUnauthorized     = errors.New("users: unauthorized")
	ErrUserNotFound     = errors.New("users: user not found")
	ErrPermissionDenied = errors.New("权限不足")
	ErrConflict         = errors.New("users: conflict")
	ErrInvalidCursor    = errors.New("users: invalid cursor")
	// ErrInvalidRequest 参数错误（10000）。用于缺目标 ID 等必传参数缺失。
	ErrInvalidRequest = errors.New("users: invalid request")
	// ErrBlockedTag 用户资料 tags 含被屏蔽标签（10400）。
	ErrBlockedTag = errors.New("users: tags contain blocked tag")
	// ErrLoginIDTaken 升级时 login_id 已被占用（spec api_spec_user.md →
	// upgrade 错误码 10400）。由凭证写入实现（auth 模块）包装返回。
	ErrLoginIDTaken = errors.New("users: login_id already taken")
	// ErrUpgradeUnavailable 凭证写入依赖（auth 模块）未接线，升级被拒绝——
	// 只改 user_type 不落凭证会导致升级用户无法重新登录（spec 契约级语义）。
	ErrUpgradeUnavailable = errors.New("users: credential writer not wired")
	// ErrAvatarNotOwned avatar_file_id 不存在/不归属当前用户/非 avatar 类/
	// 未完成上传（spec api_spec_user.md update_account 错误码 10300）。
	ErrAvatarNotOwned = errors.New("users: avatar_file_id not owned or not ready")
	// ErrOwnsGroups 仍担任群主（10400）：注销会制造无人可管理的孤儿群，
	// 须先转让群主或解散群组（对齐退群/降级等路径拒绝制造孤儿的既有规则）。
	ErrOwnsGroups = errors.New("users: still owns groups")
	// ErrInvalidProfileField 资料字段违反契约约束（10400）：nickname 1-30 字符、
	// bio 0-500 字符、tags 每个标签 1-30 字符且最多 10 个、gender 枚举
	// （spec api_spec_base.md「通用字段约束」、api_spec_user.md Gender）。
	// 不校验会让 ~1MB bio/数千 tag 直通入库并在查看主页时原样放大回显。
	ErrInvalidProfileField = errors.New("users: profile field violates contract constraints")
)
