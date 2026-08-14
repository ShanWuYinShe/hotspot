# HotSpot API - 守卫与权限模型

> 定义 HotSpot 的统一守卫模型：节点划分、认证档位、资源约束、守卫子树规则、平台权限键与失败语义。
> 本文档是权限判定的唯一权威来源；各模块端点文档的「守卫/认证」字段只引用本文档，不重复定义规则。

## 契约引用

- 基础规范与约束：`api_spec_base.md`（节点与路径架构、命名规范、类型稳定性）
- 错误码与 ForbiddenReason：`api_spec_errors.md`

---

## 守卫位置总览（契约级）

| 位置 | 调用者 | 守卫 | 判定时机 |
|------|--------|------|----------|
| `/api/v1/` | 普通用户（含群成员） | 认证档位（public/optional/required/credential）+ 资源约束 | 认证在平台层中间件；资源约束在引擎 |
| `/api/v1/groups/admin/` | 群管理员 | 守卫子树：调用者必须是**请求目标群**的 owner/admin | 子树级统一中间件（惰性解析 group_id → 角色） |
| `/admin/v1/` | 平台管理员 | required + `admin.*` 权限键 | 节点级统一中间件 |

**判定顺序（契约级）**：认证 → 可见性（10300 防枚举）→ 守卫/授权（10200 + forbidden_reason）。可见性失败必须先于授权失败返回，不可见资源连「无权限」都不暴露。

**端点单角色原则（契约级）**：每个端点只有一种调用者角色与一种检查规则；双角色语义（「作者或管理员」）一律拆分为两个端点。角色层级（owner ⊃ admin ⊃ member）由守卫层级表达，不是端点内条件分支。

---

## 认证档位（仅四种，权威定义）

| 档位 | 语义 | 缺少/无效 Token 行为 |
|------|------|---------------------|
| `public` | 不涉及鉴权 | 无（仅账号生命周期端点可用） |
| `optional` | 匿名与登录态均可访问 | 视为匿名，忽略 Token，仅返回公开数据 |
| `required` | 必须登录 | 返回 `10100` |
| `credential` | 仅 `auth/refresh_token`：以 refresh_token 为凭证 | 返回 `10100`，不适用 optional 的匿名降级 |

- 「条件必须」档位与「目标缺省=当前用户」语义已废除：目标 ID 一律必传
- optional 端点：只要携带 Token 就必须验证（含黑名单检查）；验证失败按匿名处理
- 黑名单规则：access/refresh token 失效通过黑名单实现；条目有效期不得短于 Token 过期时间

---

## 资源约束（/api/v1 专用，由判定引擎按端点声明解析）

| 约束 | 语义 | 典型端点 |
|------|------|----------|
| `resource.owner` | 调用者是目标资源的作者/本人 | moments/update、comments/delete、private_messages/update、share/revoke、media/complete_upload |
| `chat.participant` | 调用者是私聊会话的双方之一 | private_chats/*、private_messages/* |
| `group.member` | 调用者是目标群的成员 | group_messages/send、list、mark_read、groups/update_preferences、groups/leave |
| `self` | 操作对象即当前用户本人 | users/get_account 系列、auth/list_sessions 系列、notify/*、各天生私有列表 |

资源约束判定所需的角色/关系数据由引擎惰性解析；业务 service 不再自行判断角色。

---

## 群管理守卫子树（/api/v1/groups/admin/）

**守卫（子树级统一中间件）**：解析请求中的 `group_id`，判定调用者在该群的角色，要求 `owner` 或 `admin`；不满足返回 `10200` + `forbidden_reason='not_group_admin'`（非成员为 `not_group_member`）。

**owner 专属端点（守卫内再判一次，仅 2 个）**：
- `groups/admin/dismiss`：`not_group_owner`
- `groups/admin/transfer_ownership`：`not_group_owner`

**目标保护规则（10400，业务冲突）**：
- 不能对群主执行 set_mute / set_ban / remove（群主不受禁言/封禁/移除）
- set_role 不能降级最后一个管理员；owner 降级前须先转交所有权

**特殊规则（ABAC 条件，仅拒绝不授权）**：
- 禁言（mute）为成员资格上的临时条件约束：被禁言成员调用 `group_messages/send` 返回 `10400` + `data.mute`（含 muted_until）
- `groups/admin/messages/recall`：管理员撤回任意成员消息，无时间窗限制

原 14 个 `group.*` 权限键已删除，由本子树守卫取代；「自身例外」规则已删除（muted_until 归位到 `GroupViewerState` 与 `AdminGroupMemberItem`）。

---

## 平台权限键（/admin/v1 专用，17 个）

| 权限键 | 说明 |
|--------|------|
| `admin.rbac.manage` | 管理员权限管理（授权/角色/权限） |
| `admin.reports.read` | 查看举报列表 |
| `admin.reports.write` | 处理举报 |
| `admin.app_config.read` | 查看应用配置（AppConfig） |
| `admin.app_config.write` | 修改应用配置（AppConfig） |
| `admin.tags.read` | 查看标签列表 |
| `admin.tags.write` | 删除/屏蔽标签 |
| `admin.announcements.read` | 查看公告列表 |
| `admin.announcements.write` | 发布/删除公告 |
| `admin.users.read` | 查看用户列表/详情 |
| `admin.users.read_ip` | 查看用户详情中的 IP 字段（敏感 PII，需同时具备 admin.users.read；权限不足时字段为 null） |
| `admin.users.write` | 封禁/解封用户 |
| `admin.groups.read` | 查看群组列表/详情 |
| `admin.groups.write` | 解散群组 |
| `admin.audit.read` | 查看审计日志（不含 ip_address） |
| `admin.audit.read_ip` | 查看审计日志 IP 地址（需同时具备 admin.audit.read） |

**角色模型**：
- **superadmin**：`permissions=['*']`，拥有全部权限键；仅 PII 键（`read_ip` ×2）要求显式持有
- **admin**：绑定 `AdminRole`（roles/* 机制），permissions = 角色集合并集
- 权限蕴含（契约级）：`admin/reports/handle` 在 `action='resolve'` 且携带 `ban_duration_days` 时，实际执行封禁——调用方除 `admin.reports.write` 外必须同时具备 `admin.users.write`，权限不足返回 `10200` + `insufficient_admin_permission`

---

## 交叉规则（契约级）

1. **平台管理员在 /api/v1 上与普通用户完全同权**：平台权限不外溢到客户端 API
2. **群角色与 /admin/v1 无关**：群主/群管理员的动作全部在客户端 API（含守卫子树）执行并落审计；平台节点只做平台级动作
3. **/admin/v1 响应使用独立 `AdminXxxItem` 类型**：客户端类型的可见性注解与视角规则不适用于该节点；其字段可见性由 `admin.*` 权限键 + PII 键控制
4. **同一用户多角色分别判定、互不传递**：用户可同时是平台管理员与某群群主，两套判定按节点与上下文独立进行
5. **术语纪律**：文档禁止裸称「管理员」；一律写「平台管理员」或「群主/群管理员」

---

## 失败语义（契约级）

| 失败 | 码 | data |
|------|-----|------|
| 未认证（required/credential） | 10100 | 无 |
| 不存在/不可见（含隐私隐藏，防枚举） | 10300 | 无 |
| 权限不足 | 10200 | `{ forbidden_reason: ForbiddenReason }`（**必填**） |
| 业务冲突/前置不满足 | 10400 | 端点可显式声明结构化 data（如 `data.mute`） |
| 账号封禁（强制登出） | 10201 | 无（中间件级） |

`ForbiddenReason` 枚举（12 项）的权威定义在 `api_spec_errors.md`；`BlockReason` 已并入。

**隐私拒绝**：资源存在但不可见（private/被拉黑/资料隐藏）→ 10300，先于授权判定；整表隐私拒绝（social_lists_visible 等=false）→ 10200 + `privacy_setting`。

---

## 权限矩阵（契约附件，148 行）

> 列说明：
> - 认证 = 档位（public/optional/required/credential）
> - 守卫 = 守卫子树（groups/admin）或平台权限键（admin.*）
> - 约束 = 资源约束（owner=resource.owner、part=chat.participant、gmember=group.member、self）
> - reason = 10200 时的 forbidden_reason
> - 幂等 = 幂等键/自然去重/no-op 语义

| # | 位置 | 端点 | 认证/守卫 | 资源约束 | 10200 reason | 特有失败 | 幂等/no-op |
|---|------|------|-----------|----------|--------------|----------|------------|
| 1 | /api/v1 | `/api/v1/auth/register` | public | — | — | 10400 login_id 占用 | 幂等 |
| 2 | /api/v1 | `/api/v1/auth/login` | public | — | — | 10100（防枚举） | — |
| 3 | /api/v1 | `/api/v1/auth/logout` | required | self.session | — | — | no-op |
| 4 | /api/v1 | `/api/v1/auth/guest` | public | — | — | — | 幂等 |
| 5 | /api/v1 | `/api/v1/auth/refresh_token` | credential | refresh 归属+黑名单 | — | 10100 | — |
| 6 | /api/v1 | `/api/v1/auth/get_status` | optional | — | — | — | — |
| 7 | /api/v1 | `/api/v1/auth/request_password_reset` | public | — | — | 限流 | — |
| 8 | /api/v1 | `/api/v1/auth/reset_password` | public | 一次性验证码 | — | 10300 | 自然去重 |
| 9 | /api/v1 | `/api/v1/auth/change_password` | required | self+旧密码 | — | 10401 | 自然去重 |
| 10 | /api/v1 | `/api/v1/auth/list_sessions` | required | self | — | — | — |
| 11 | /api/v1 | `/api/v1/auth/revoke_session` | required | self.session | — | 10300/10400(当前会话) | — |
| 12 | /api/v1 | `/api/v1/auth/revoke_all_sessions` | required | self.session | — | — | — |
| 13 | /api/v1 | `/api/v1/users/search` | optional | searchable 系列过滤 | — | — | — |
| 14 | /api/v1 | `/api/v1/users/open_the_door` | optional | 隐私过滤 | — | 10300 | — |
| 15 | /api/v1 | `/api/v1/users/list_friends` | optional | social_lists_visible | privacy_setting | 10200/10300 | — |
| 16 | /api/v1 | `/api/v1/users/list_following` | optional | social_lists_visible | privacy_setting | 10200/10300 | — |
| 17 | /api/v1 | `/api/v1/users/list_followers` | optional | social_lists_visible | privacy_setting | 10200/10300 | — |
| 18 | /api/v1 | `/api/v1/users/list_groups` | optional | groups_visible | privacy_setting | 10200/10300 | — |
| 19 | /api/v1 | `/api/v1/users/list_moments` | optional | 公开瞬间过滤 | — | 10300 | — |
| 20 | /api/v1 | `/api/v1/users/list_reacted_moments` | optional | reacted_moments_visible | privacy_setting | 10200/10300 | — |
| 21 | /api/v1 | `/api/v1/users/batch_get` | optional | 可见性过滤 | — | — | — |
| 22 | /api/v1 | `/api/v1/users/get_account` | required | self | — | — | — |
| 23 | /api/v1 | `/api/v1/users/update_account` | required | self | — | 10300(avatar)/10400(屏蔽标签) | — |
| 24 | /api/v1 | `/api/v1/users/update_location` | required | self | — | — | — |
| 25 | /api/v1 | `/api/v1/users/upgrade` | required | self(temp) | — | 10400 | — |
| 26 | /api/v1 | `/api/v1/users/get_settings` | required | self | — | — | — |
| 27 | /api/v1 | `/api/v1/users/update_settings` | required | self | — | — | — |
| 28 | /api/v1 | `/api/v1/users/delete_account` | required | self | — | 10000(确认串) | — |
| 29 | /api/v1 | `/api/v1/users/set_follow` | required | 对方 allow_follow | privacy_setting | 10300/10400(自己) | no-op |
| 30 | /api/v1 | `/api/v1/users/set_block` | required | — | — | 10300/10400(自己) | no-op |
| 31 | /api/v1 | `/api/v1/users/list_blocked` | required | self | — | — | — |
| 32 | /api/v1 | `/api/v1/users/set_remark` | required | 已关注条件 | — | 10300/10400(未关注) | no-op |
| 33 | /api/v1 | `/api/v1/moments/search` | optional | 可见性过滤 | — | — | — |
| 34 | /api/v1 | `/api/v1/moments/open_the_door` | optional | 可见性过滤 | — | 10300 | — |
| 35 | /api/v1 | `/api/v1/moments/create` | required | — | — | 10300/10400(屏蔽标签) | 幂等 |
| 36 | /api/v1 | `/api/v1/moments/update` | required | owner | not_resource_owner | 10200/10300 | — |
| 37 | /api/v1 | `/api/v1/moments/delete` | required | owner | not_resource_owner | 10200/10300 | — |
| 38 | /api/v1 | `/api/v1/moments/set_favorite` | required | 可见性过滤 | — | 10300 | no-op |
| 39 | /api/v1 | `/api/v1/moments/list_created` | required | self(owner) | — | — | — |
| 40 | /api/v1 | `/api/v1/moments/list_favorites` | required | self | — | — | — |
| 41 | /api/v1 | `/api/v1/comments/create` | required | 评论限制+黑名单 | privacy_setting/blocked_by_user | 10300 | 幂等 |
| 42 | /api/v1 | `/api/v1/comments/delete` | required | owner | not_resource_owner | 10200/10300 | — |
| 43 | /api/v1 | `/api/v1/comments/list` | optional | 可见性过滤 | — | 10300 | — |
| 44 | /api/v1 | `/api/v1/comments/list_replies` | optional | 可见性过滤 | — | 10300 | — |
| 45 | /api/v1 | `/api/v1/comments/open_the_door` | optional | 可见性过滤 | — | 10300 | — |
| 46 | /api/v1 | `/api/v1/reactions/set` | required | 目标可见性 | — | 10300 | no-op 语义 |
| 47 | /api/v1 | `/api/v1/reactions/list` | optional | 目标可见性 | — | 10300 | — |
| 48 | /api/v1 | `/api/v1/private_chats/list` | required | self | — | — | — |
| 49 | /api/v1 | `/api/v1/private_chats/open_the_door` | required | 对方 ChatPerm | privacy_setting | 10300 | — |
| 50 | /api/v1 | `/api/v1/private_chats/update_preferences` | required | part | — | 10300 | no-op |
| 51 | /api/v1 | `/api/v1/private_chats/delete` | required | self | — | 10300 | — |
| 52 | /api/v1 | `/api/v1/private_messages/send` | required | part+ChatPerm/黑名单/配额 | blocked_by_user/privacy_setting/stranger_quota_exceeded | 10300/10400 | 幂等 |
| 53 | /api/v1 | `/api/v1/private_messages/list` | required | part | — | 10300 | — |
| 54 | /api/v1 | `/api/v1/private_messages/update` | required | owner | not_resource_owner | 10300/10400(非文本/已撤回) | — |
| 55 | /api/v1 | `/api/v1/private_messages/recall` | required | owner | not_resource_owner | 10300/10400(超时) | — |
| 56 | /api/v1 | `/api/v1/private_messages/mark_read` | required | part | — | 10300 | — |
| 57 | /api/v1 | `/api/v1/group_messages/send` | required | gmember+禁言 | not_group_member | 10300/10400(禁言+data.mute) | 幂等 |
| 58 | /api/v1 | `/api/v1/group_messages/list` | required | gmember | not_group_member | 10300/10400 | — |
| 59 | /api/v1 | `/api/v1/group_messages/update` | required | owner | not_resource_owner | 10300/10400 | — |
| 60 | /api/v1 | `/api/v1/group_messages/recall` | required | owner | not_resource_owner | 10300/10400(超时) | — |
| 61 | /api/v1 | `/api/v1/group_messages/mark_read` | required | gmember | not_group_member | 10300/10400 | — |
| 62 | /api/v1 | `/api/v1/groups/search` | optional | is_searchable 过滤 | — | — | — |
| 63 | /api/v1 | `/api/v1/groups/open_the_door` | optional | 可见性过滤 | — | 10300 | — |
| 64 | /api/v1 | `/api/v1/groups/create` | required | — | — | 10300/10400(屏蔽标签) | 幂等 |
| 65 | /api/v1 | `/api/v1/groups/join` | required | join_method+封禁 | banned_in_group | 10300/10400 | 自然去重 |
| 66 | /api/v1 | `/api/v1/groups/leave` | required | gmember | not_group_member | 10300/10400(群主) | — |
| 67 | /api/v1 | `/api/v1/groups/list_joined` | required | self | — | — | — |
| 68 | /api/v1 | `/api/v1/groups/update_preferences` | required | gmember | not_group_member | 10300/10400 | no-op |
| 69 | /api/v1 | `/api/v1/groups/announcements/list` | optional | 可见性过滤 | — | 10300 | — |
| 70 | /api/v1 | `/api/v1/groups/members/list` | optional | members_searchable 条件 | not_group_member | 10300 | — |
| 71 | /api/v1 | `/api/v1/groups/members/search` | optional | members_searchable 条件 | not_group_member | 10300 | — |
| 72 | /api/v1 | `/api/v1/groups/applications/list_submitted` | required | self(申请人) | — | — | — |
| 73 | /api/v1 | `/api/v1/groups/applications/cancel` | required | self(申请人) | not_resource_owner | 10300/10400 | — |
| 74 | /api/v1 | `/api/v1/groups/invites/list_received` | required | self | — | — | — |
| 75 | /api/v1 | `/api/v1/groups/invites/accept` | required | self(受邀人) | — | 10300/10400 | — |
| 76 | /api/v1 | `/api/v1/groups/invites/reject` | required | self(受邀人) | — | 10300/10400 | — |
| 77 | /api/v1 | `/api/v1/feed/following` | required | — | — | — | — |
| 78 | /api/v1 | `/api/v1/feed/nearby` | required | — | — | — | — |
| 79 | /api/v1 | `/api/v1/feed/nearby_users` | optional | searchable 系列过滤 | — | — | — |
| 80 | /api/v1 | `/api/v1/feed/nearby_groups` | optional | is_searchable 过滤 | — | — | — |
| 81 | /api/v1 | `/api/v1/feed/trending` | optional | — | — | — | — |
| 82 | /api/v1 | `/api/v1/search/suggest` | optional | 可见性过滤 | — | — | — |
| 83 | /api/v1 | `/api/v1/map/canvas` | optional | 公开对象过滤 | — | — | — |
| 84 | /api/v1 | `/api/v1/media/presign_upload` | required | — | — | 限流 | — |
| 85 | /api/v1 | `/api/v1/media/batch_presign_upload` | required | — | — | — | — |
| 86 | /api/v1 | `/api/v1/media/complete_upload` | required | owner(upload_id) | not_resource_owner | 10300/10400 | 自然去重 |
| 87 | /api/v1 | `/api/v1/media/batch_complete_upload` | required | owner(upload_id) | not_resource_owner | 10300/10400 | 自然去重 |
| 88 | /api/v1 | `/api/v1/share/create` | required | 目标可见 | — | 10300 | 幂等 |
| 89 | /api/v1 | `/api/v1/share/get` | optional | 撤销/过期 | — | 10300 | 反滥用限流 |
| 90 | /api/v1 | `/api/v1/share/list` | required | self | — | — | — |
| 91 | /api/v1 | `/api/v1/share/revoke` | required | owner | — | 10300(隐私一致性) | no-op |
| 92 | /api/v1 | `/api/v1/reports/create` | required | 目标存在+非自举报 | — | 10300/10400 | 自然去重 |
| 93 | /api/v1 | `/api/v1/reports/list` | required | self | — | — | — |
| 94 | /api/v1 | `/api/v1/notify/list` | required | self | — | — | — |
| 95 | /api/v1 | `/api/v1/notify/get_unread_count` | required | self | — | — | — |
| 96 | /api/v1 | `/api/v1/notify/mark_read` | required | self(scope 显式) | — | — | — |
| 97 | /api/v1 | `/api/v1/system/get_meta` | optional | — | — | — | — |
| 98 | /api/v1 | `/api/v1/system/health` | optional | — | — | 10600 | — |
| 99 | /api/v1 | `/api/v1/system/announcements/list` | optional | — | — | — | — |
| 100 | /api/v1 | `/api/v1/system/get_client_config` | optional | — | — | — | — |
| 101 | /api/v1 | `/api/v1/system/bootstrap` | optional | self(登录态附加) | — | — | If-None-Match |
| 102 | /api/v1/groups/admin | `/api/v1/groups/admin/update` | 守卫(owner/admin) | gmember | not_group_admin | 10200/10300 | — |
| 103 | /api/v1/groups/admin | `/api/v1/groups/admin/get_settings` | 守卫(owner/admin) | gmember | not_group_admin | 10300 | — |
| 104 | /api/v1/groups/admin | `/api/v1/groups/admin/dismiss` | owner 专属 | gmember | not_group_owner | 10200/10300 | — |
| 105 | /api/v1/groups/admin | `/api/v1/groups/admin/invites/create` | 守卫(owner/admin) | gmember | not_group_admin | 10300/10400 | 自然去重 |
| 106 | /api/v1/groups/admin | `/api/v1/groups/admin/invites/list_sent` | 守卫(owner/admin) | gmember | not_group_admin | 10300/10400 | — |
| 107 | /api/v1/groups/admin | `/api/v1/groups/admin/invites/revoke` | 守卫(owner/admin) | gmember | not_group_admin | 10300/10400 | — |
| 108 | /api/v1/groups/admin | `/api/v1/groups/admin/members/list` | 守卫(owner/admin) | gmember | not_group_admin | 10300 | — |
| 109 | /api/v1/groups/admin | `/api/v1/groups/admin/members/set_role` | 守卫(owner/admin) | gmember | not_group_admin | 10300/10400 | no-op 同角色 |
| 110 | /api/v1/groups/admin | `/api/v1/groups/admin/members/set_mute` | 守卫(owner/admin) | gmember | not_group_admin | 10300/10400(群主保护) | no-op |
| 111 | /api/v1/groups/admin | `/api/v1/groups/admin/members/set_ban` | 守卫(owner/admin) | gmember | not_group_admin | 10300/10400(群主保护) | no-op 解封 |
| 112 | /api/v1/groups/admin | `/api/v1/groups/admin/members/list_banned` | 守卫(owner/admin) | gmember | not_group_admin | 10300 | — |
| 113 | /api/v1/groups/admin | `/api/v1/groups/admin/members/remove` | 守卫(owner/admin) | gmember | not_group_admin | 10300/10400(群主保护) | — |
| 114 | /api/v1/groups/admin | `/api/v1/groups/admin/applications/list` | 守卫(owner/admin) | gmember | not_group_admin | 10300 | — |
| 115 | /api/v1/groups/admin | `/api/v1/groups/admin/applications/approve` | 守卫(owner/admin) | gmember | not_group_admin | 10300/10400 | no-op 已接受 |
| 116 | /api/v1/groups/admin | `/api/v1/groups/admin/applications/reject` | 守卫(owner/admin) | gmember | not_group_admin | 10300/10400 | — |
| 117 | /api/v1/groups/admin | `/api/v1/groups/admin/announcements/create` | 守卫(owner/admin) | gmember | not_group_admin | 10300/10400 | 幂等 |
| 118 | /api/v1/groups/admin | `/api/v1/groups/admin/announcements/update` | 守卫(owner/admin) | gmember | not_group_admin | 10200/10300 | — |
| 119 | /api/v1/groups/admin | `/api/v1/groups/admin/announcements/delete` | 守卫(owner/admin) | gmember | not_group_admin | 10200/10300 | — |
| 120 | /api/v1/groups/admin | `/api/v1/groups/admin/transfer_ownership` | owner 专属 | gmember | not_group_owner | 10300/10400(目标非成员) | 自然去重 |
| 121 | /api/v1/groups/admin | `/api/v1/groups/admin/messages/recall` | 守卫(owner/admin) | gmember | not_group_admin | 10300/10400 | — |
| 122 | /admin/v1 | `/admin/v1/rbac/get_me` | required+平台管理员 | — | not_admin | — | — |
| 123 | /admin/v1 | `/admin/v1/rbac/list_permissions` | admin.rbac.manage | — | not_admin/insufficient_admin_permission | — | — |
| 124 | /admin/v1 | `/admin/v1/rbac/roles/list` | admin.rbac.manage | — | not_admin/insufficient_admin_permission | — | — |
| 125 | /admin/v1 | `/admin/v1/rbac/roles/create` | admin.rbac.manage | — | not_admin/insufficient_admin_permission | 10400 重名 | — |
| 126 | /admin/v1 | `/admin/v1/rbac/roles/update` | admin.rbac.manage | — | not_admin/insufficient_admin_permission | 10300/10400 | — |
| 127 | /admin/v1 | `/admin/v1/rbac/roles/delete` | admin.rbac.manage | — | not_admin/insufficient_admin_permission | 10300/10400 | — |
| 128 | /admin/v1 | `/admin/v1/rbac/roles/set_permissions` | admin.rbac.manage | — | not_admin/insufficient_admin_permission | 10300；* 仅超管 | — |
| 129 | /admin/v1 | `/admin/v1/rbac/admins/list` | admin.rbac.manage | — | not_admin/insufficient_admin_permission | — | — |
| 130 | /admin/v1 | `/admin/v1/rbac/admins/grant` | admin.rbac.manage | — | not_admin/insufficient_admin_permission | 10300；grant_all 仅超管 | — |
| 131 | /admin/v1 | `/admin/v1/rbac/admins/revoke` | admin.rbac.manage | — | not_admin/insufficient_admin_permission | 10300；撤超管仅超管 | — |
| 132 | /admin/v1 | `/admin/v1/audit/list` | admin.audit.read | — | not_admin/insufficient_admin_permission | — | — |
| 133 | /admin/v1 | `/admin/v1/app_config/get` | admin.app_config.read | — | not_admin/insufficient_admin_permission | — | — |
| 134 | /admin/v1 | `/admin/v1/app_config/update` | admin.app_config.write | — | not_admin/insufficient_admin_permission | — | — |
| 135 | /admin/v1 | `/admin/v1/users/search` | admin.users.read | — | not_admin/insufficient_admin_permission | — | — |
| 136 | /admin/v1 | `/admin/v1/users/open_the_door` | admin.users.read | — | not_admin/insufficient_admin_permission | 10300 | — |
| 137 | /admin/v1 | `/admin/v1/users/set_ban` | admin.users.write | — | not_admin/insufficient_admin_permission/admin_protected_target | 10300/10400 | no-op 解封 |
| 138 | /admin/v1 | `/admin/v1/groups/search` | admin.groups.read | — | not_admin/insufficient_admin_permission | — | — |
| 139 | /admin/v1 | `/admin/v1/groups/open_the_door` | admin.groups.read | — | not_admin/insufficient_admin_permission | 10300 | — |
| 140 | /admin/v1 | `/admin/v1/groups/dismiss` | admin.groups.write | — | not_admin/insufficient_admin_permission | 10300 | — |
| 141 | /admin/v1 | `/admin/v1/reports/list` | admin.reports.read | — | not_admin/insufficient_admin_permission | — | — |
| 142 | /admin/v1 | `/admin/v1/reports/handle` | admin.reports.write(+蕴含 users.write) | — | not_admin/insufficient_admin_permission | 10300/10400 | — |
| 143 | /admin/v1 | `/admin/v1/tags/search` | admin.tags.read | — | not_admin/insufficient_admin_permission | — | — |
| 144 | /admin/v1 | `/admin/v1/tags/delete` | admin.tags.write | — | not_admin/insufficient_admin_permission | 10300 | — |
| 145 | /admin/v1 | `/admin/v1/announcements/list` | admin.announcements.read | — | not_admin/insufficient_admin_permission | — | — |
| 146 | /admin/v1 | `/admin/v1/announcements/create` | admin.announcements.write | — | not_admin/insufficient_admin_permission | — | 幂等 |
| 147 | /admin/v1 | `/admin/v1/announcements/update` | admin.announcements.write | — | not_admin/insufficient_admin_permission | 10300 | — |
| 148 | /admin/v1 | `/admin/v1/announcements/delete` | admin.announcements.write | — | not_admin/insufficient_admin_permission | 10300 | — |

（共 148 行，与端点总数一致。）
