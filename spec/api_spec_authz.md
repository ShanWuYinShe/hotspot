# HotSpot API - 守卫与权限模型

> 定义 HotSpot 的统一守卫模型：节点划分、节点身份语义、资源约束、守卫子树规则、平台权限键与失败语义。
> 本文档是权限判定的唯一权威来源；各模块端点文档的守卫说明只引用本文档，不重复定义规则。

## 契约引用

- 基础规范与约束：`api_spec_base.md`（节点与路径架构、节点身份语义、命名规范、类型稳定性）
- public 节点端点清单：`api_spec_public.md`
- 错误码与 ForbiddenReason：`api_spec_errors.md`

---

## 守卫位置总览（契约级）

节点划分与守卫语义见 `api_spec_base.md`「节点与路径架构」；各位置的判定时机：

| 位置 | 判定时机 |
|------|----------|
| `/public/v1/` | 无身份判定；仅可见性过滤（资源约束在引擎） |
| `/api/v1/` | 认证在平台层中间件；资源约束在引擎 |
| `/api/v1/groups/admin/` | 子树级统一中间件（惰性解析 group_id → 角色） |
| `/admin/v1/` | 节点级统一中间件（身份 + `admin.*` 权限键） |

**节点即档位（契约级）**：身份要求由节点前缀唯一表达——`/public/v1` = 纯匿名（不解析 Token）、`/api/v1` = 注册用户、`/admin/v1` = 平台管理员。唯一特例：`/public/v1/auth/refresh_token` 以请求体中的 refresh_token 为凭证（凭证语义独立于节点匿名语义）。详见 `api_spec_base.md`「节点身份语义」。

**判定顺序（契约级）**：认证（仅 /api/v1、/admin/v1）→ 可见性（10300 防枚举）→ 守卫/授权（10200 + forbidden_reason）。可见性失败必须先于授权失败返回，不可见资源连「无权限」都不暴露。

**端点单角色原则（契约级）**：每个端点只有一种调用者角色与一种检查规则；双角色语义（「作者或管理员」）一律拆分为两个端点。角色层级（owner ⊃ admin ⊃ member）由守卫层级表达，不是端点内条件分支。

---

## 资源约束（由判定引擎按端点声明解析）

身份类资源约束仅存在于 `/api/v1`（含 groups/admin 子树）；`/public/v1` 仅适用可见性过滤（无身份类约束）。

| 约束 | 语义 | 典型端点 |
|------|------|----------|
| `resource.owner` | 调用者是目标资源的作者/本人 | moments/update、comments/delete、private_messages/update、share/revoke、media/complete_upload |
| `chat.participant` | 调用者是私聊会话的双方之一 | private_chats/*、private_messages/* |
| `group.member` | 调用者是目标群的成员 | group_messages/send、fetch、mark_read、groups/update_preferences、groups/exit |
| `self` | 操作对象即当前用户本人 | users/account 系列、auth/fetch_sessions 系列、notify/*、各天生私有列表 |

资源约束判定所需的角色/关系数据由引擎惰性解析；业务 service 不再自行判断角色。

---

## 群管理守卫子树（/api/v1/groups/admin/）

**守卫（子树级统一中间件）**：解析请求中的 `group_id`，判定调用者在该群的角色，要求 `owner` 或 `admin`；不满足返回 `10200` + `forbidden_reason='not_group_admin'`（非成员为 `not_group_member`）。

**owner 专属端点（守卫内再判一次）**：
- `groups/admin/dismiss`：`not_group_owner`
- `groups/admin/transfer_ownership`：`not_group_owner`

**目标保护规则（10400，业务冲突）**：
- 不能对群主执行 set_mute / set_ban / remove（群主不受禁言/封禁/移除）
- set_role 不能降级最后一个管理员；owner 降级前须先转交所有权

**特殊规则（ABAC 条件，仅拒绝不授权）**：
- 禁言（mute）为成员资格上的临时条件约束：被禁言成员调用 `group_messages/send` 返回 `10400` + `data.mute`（含 muted_until）
- `groups/admin/messages/recall`：管理员撤回任意成员消息，无时间窗限制

群管理权限由本子树统一守卫表达（无独立权限键）；禁言状态 `muted_until` 位于 `GroupViewerState` 与 `AdminGroupMemberItem`。

---

## 平台权限键（/admin/v1 专用）

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
- 权限蕴含（契约级）：`/admin/v1/reports/handle` 在 `action='resolve'` 且携带 `ban_duration_days` 时，实际执行封禁——调用方除 `admin.reports.write` 外必须同时具备 `admin.users.write`，权限不足返回 `10200` + `insufficient_admin_permission`

---

## 交叉规则（契约级）

1. **登录用户调 /public/v1 得到与匿名完全相同的响应**：public 节点不解析 Token，无任何隐式身份增强；需要登录视角（viewer_state / relation 等）必须调 `/api/v1` 对应端点
2. **平台管理员在 /api/v1 上与普通用户完全同权**：平台权限不外溢到客户端 API
3. **群角色与 /admin/v1 无关**：群主/群管理员的动作全部在客户端 API（含守卫子树）执行并落审计；平台节点只做平台级动作
4. **/admin/v1 响应使用独立 `AdminXxxItem` 类型**：客户端类型的可见性注解与视角规则不适用于该节点；其字段可见性由 `admin.*` 权限键 + PII 键控制
5. **同一用户多角色分别判定、互不传递**：用户可同时是平台管理员与某群群主，两套判定按节点与上下文独立进行
6. **术语纪律**：文档禁止裸称「管理员」；一律写「平台管理员」或「群主/群管理员」

---

## 失败语义（契约级）

| 失败 | 码 | data | 适用节点 |
|------|-----|------|----------|
| 未认证（/api/v1、/admin/v1 及 refresh_token 凭证特例） | 10100 | 无 | /public/v1 之外 |
| 不存在/不可见（含隐私隐藏，防枚举） | 10300 | 无 | 全部 |
| 权限不足 | 10200 | `{ forbidden_reason: ForbiddenReason }`（**必填**） | 全部（public 节点为隐私类 reason：privacy_setting / not_group_member） |
| 业务冲突/前置不满足 | 10400 | 端点可显式声明结构化 data（如 `data.mute`） | 全部 |
| 账号封禁（强制登出） | 10201 | 无（中间件级） | /api/v1、/admin/v1 |

`ForbiddenReason` 枚举（11 项）的权威定义在 `api_spec_errors.md`。

**隐私拒绝**：资源存在但不可见（private/被拉黑/资料隐藏）→ 10300，先于授权判定；整表隐私拒绝（social_lists_visible 等=false）→ 10200 + `privacy_setting`。

---

## 权限矩阵（契约附件）

> 列说明：
> - **节点即档位**：块标题即节点，节点身份语义见上（/public/v1 纯匿名、/api/v1 注册用户、groups/admin 守卫子树、/admin/v1 平台管理员）
> - 守卫/约束 = 守卫子树（groups/admin）、平台权限键（admin.*）或资源约束（owner=resource.owner、part=chat.participant、gmember=group.member、self、可见性过滤）
> - reason = 10200 时的 forbidden_reason
> - 幂等 = 幂等键/自然去重/no-op 语义
> - 「登录视角」= /api/v1 登录版端点的可见性判定语义；其匿名镜像版在 `/public/v1`（类型独立，见 `api_spec_public.md`）
> - **镜像判定原则**：仅当登录版存在真实增量（查看者状态字段，或可见性判定依赖身份——如 `reactions/fetch` 的私聊消息）才拆双版本；无任何增量的端点仅保留 `/public/v1` 版（如 `map/canvas`、`groups/announcements/fetch`、system 的 meta/health/announcements/fetch/client_config），不设登录镜像

### /public/v1（未注册用户节点，纯匿名）

| # | 端点 | 守卫/约束 | 10200 reason | 特有失败 | 幂等/no-op |
|---|------|-----------|--------------|----------|------------|
| 1 | `/public/v1/auth/register` | —（login_id 唯一） | — | 10400 login_id 占用 | 幂等 |
| 2 | `/public/v1/auth/signin` | — | — | 10100（防枚举） | — |
| 3 | `/public/v1/auth/guest` | — | — | — | 幂等 |
| 4 | `/public/v1/auth/refresh_token` | 凭证特例（refresh 归属+黑名单） | — | 10100 | — |
| 5 | `/public/v1/auth/forgot_password` | — | — | 限流 | — |
| 6 | `/public/v1/auth/reset_password` | 一次性验证码 | — | 10300 | 自然去重 |
| 7 | `/public/v1/users/search` | searchable 系列过滤 | — | — | — |
| 8 | `/public/v1/users/open_the_door` | 隐私过滤 | — | 10300 | — |
| 9 | `/public/v1/users/fetch_friends` | social_lists_visible | privacy_setting | 10200/10300 | — |
| 10 | `/public/v1/users/fetch_following` | social_lists_visible | privacy_setting | 10200/10300 | — |
| 11 | `/public/v1/users/fetch_followers` | social_lists_visible | privacy_setting | 10200/10300 | — |
| 12 | `/public/v1/users/fetch_groups` | groups_visible | privacy_setting | 10200/10300 | — |
| 13 | `/public/v1/users/fetch_moments` | 公开瞬间过滤 | — | 10300 | — |
| 14 | `/public/v1/users/fetch_reacted_moments` | reacted_moments_visible | privacy_setting | 10200/10300 | — |
| 15 | `/public/v1/users/batch_fetch` | 可见性过滤 | — | — | — |
| 16 | `/public/v1/moments/search` | 可见性过滤 | — | — | — |
| 17 | `/public/v1/moments/open_the_door` | 可见性过滤 | — | 10300 | — |
| 18 | `/public/v1/comments/fetch` | 可见性过滤 | — | 10300 | — |
| 19 | `/public/v1/comments/fetch_replies` | 可见性过滤 | — | 10300 | — |
| 20 | `/public/v1/comments/open_the_door` | 可见性过滤 | — | 10300 | — |
| 21 | `/public/v1/groups/search` | is_searchable 过滤 | — | — | — |
| 22 | `/public/v1/groups/open_the_door` | 可见性过滤 | — | 10300 | — |
| 23 | `/public/v1/groups/announcements/fetch` | 可见性过滤 | — | 10300 | — |
| 24 | `/public/v1/groups/members/fetch` | members_searchable 条件 | not_group_member | 10300 | — |
| 25 | `/public/v1/groups/members/search` | members_searchable 条件 | not_group_member | 10300 | — |
| 26 | `/public/v1/feed/nearby_users` | searchable 系列过滤 | — | — | — |
| 27 | `/public/v1/feed/nearby_groups` | is_searchable 过滤 | — | — | — |
| 28 | `/public/v1/feed/trending` | — | — | — | — |
| 29 | `/public/v1/search/suggest` | 可见性过滤 | — | — | — |
| 30 | `/public/v1/map/canvas` | 公开对象过滤 | — | — | — |
| 31 | `/public/v1/share/fetch` | 撤销/过期 | — | 10300 | 反滥用限流 |
| 32 | `/public/v1/reactions/fetch` | 目标可见性 | — | 10300 | — |
| 33 | `/public/v1/system/meta` | — | — | — | — |
| 34 | `/public/v1/system/health` | — | — | 10600 | — |
| 35 | `/public/v1/system/announcements/fetch` | — | — | — | — |
| 36 | `/public/v1/system/client_config` | — | — | — | — |
| 37 | `/public/v1/system/bootstrap` | — | — | — | If-None-Match |

### /api/v1（注册用户节点，必须有效 Token）

| # | 端点 | 守卫/约束 | 10200 reason | 特有失败 | 幂等/no-op |
|---|------|-----------|--------------|----------|------------|
| 38 | `/api/v1/auth/signout` | self.session | — | — | no-op |
| 39 | `/api/v1/auth/session` | self | — | — | — |
| 40 | `/api/v1/auth/change_password` | self+旧密码 | — | 10401 | 自然去重 |
| 41 | `/api/v1/auth/fetch_sessions` | self | — | — | — |
| 42 | `/api/v1/auth/revoke_session` | self.session | — | 10300/10400(当前会话) | — |
| 43 | `/api/v1/auth/revoke_other_sessions` | self.session | — | — | — |
| 44 | `/api/v1/users/search` | searchable 系列过滤（登录视角） | — | — | — |
| 45 | `/api/v1/users/open_the_door` | 隐私过滤（登录视角） | — | 10300 | — |
| 46 | `/api/v1/users/fetch_friends` | social_lists_visible（登录视角） | privacy_setting | 10200/10300 | — |
| 47 | `/api/v1/users/fetch_following` | social_lists_visible（登录视角） | privacy_setting | 10200/10300 | — |
| 48 | `/api/v1/users/fetch_followers` | social_lists_visible（登录视角） | privacy_setting | 10200/10300 | — |
| 49 | `/api/v1/users/fetch_groups` | groups_visible（登录视角） | privacy_setting | 10200/10300 | — |
| 50 | `/api/v1/users/fetch_moments` | 公开瞬间过滤（登录视角） | — | 10300 | — |
| 51 | `/api/v1/users/fetch_reacted_moments` | reacted_moments_visible（登录视角） | privacy_setting | 10200/10300 | — |
| 52 | `/api/v1/users/batch_fetch` | 可见性过滤（登录视角） | — | — | — |
| 53 | `/api/v1/users/account` | self | — | — | — |
| 54 | `/api/v1/users/update_account` | self | — | 10300(avatar)/10400(屏蔽标签) | — |
| 55 | `/api/v1/users/update_location` | self | — | — | — |
| 56 | `/api/v1/users/upgrade` | self(temp) | — | 10400 | — |
| 57 | `/api/v1/users/settings` | self | — | — | — |
| 58 | `/api/v1/users/update_settings` | self | — | — | — |
| 59 | `/api/v1/users/delete_account` | self | — | 10000(确认串) | — |
| 60 | `/api/v1/users/set_follow` | 对方 allow_follow | privacy_setting | 10300/10400(自己) | no-op |
| 61 | `/api/v1/users/set_block` | — | — | 10300/10400(自己) | no-op |
| 62 | `/api/v1/users/fetch_blocked` | self | — | — | — |
| 63 | `/api/v1/users/set_remark` | 已关注条件 | — | 10300/10400(未关注) | no-op |
| 64 | `/api/v1/moments/search` | 可见性过滤（登录视角） | — | — | — |
| 65 | `/api/v1/moments/open_the_door` | 可见性过滤（登录视角） | — | 10300 | — |
| 66 | `/api/v1/moments/create` | — | — | 10300/10400(屏蔽标签) | 幂等 |
| 67 | `/api/v1/moments/update` | owner | not_resource_owner | 10200/10300 | — |
| 68 | `/api/v1/moments/delete` | owner | not_resource_owner | 10200/10300 | — |
| 69 | `/api/v1/moments/set_favorite` | 可见性过滤 | — | 10300 | no-op |
| 70 | `/api/v1/moments/fetch_created` | self(owner) | — | — | — |
| 71 | `/api/v1/moments/fetch_favorites` | self | — | — | — |
| 72 | `/api/v1/comments/create` | 评论限制+黑名单 | privacy_setting/blocked_by_user | 10300 | 幂等 |
| 73 | `/api/v1/comments/delete` | owner | not_resource_owner | 10200/10300 | — |
| 74 | `/api/v1/comments/fetch` | 可见性过滤（登录视角） | — | 10300 | — |
| 75 | `/api/v1/comments/fetch_replies` | 可见性过滤（登录视角） | — | 10300 | — |
| 76 | `/api/v1/comments/open_the_door` | 可见性过滤（登录视角） | — | 10300 | — |
| 77 | `/api/v1/reactions/set` | 目标可见性 | — | 10300 | no-op 语义 |
| 78 | `/api/v1/reactions/fetch` | 目标可见性（登录视角） | — | 10300 | — |
| 79 | `/api/v1/private_chats/fetch` | self | — | — | — |
| 80 | `/api/v1/private_chats/open_the_door` | 对方 ChatPerm | privacy_setting | 10300 | — |
| 81 | `/api/v1/private_chats/update_preferences` | part | — | 10300 | no-op |
| 82 | `/api/v1/private_chats/delete` | self | — | 10300 | — |
| 83 | `/api/v1/private_messages/send` | part+ChatPerm/黑名单/配额 | blocked_by_user/privacy_setting/stranger_quota_exceeded | 10300/10400 | 幂等 |
| 84 | `/api/v1/private_messages/fetch` | part | — | 10300 | — |
| 85 | `/api/v1/private_messages/update` | owner | not_resource_owner | 10300/10400(非文本/已撤回) | — |
| 86 | `/api/v1/private_messages/recall` | owner | not_resource_owner | 10300/10400(超时) | — |
| 87 | `/api/v1/private_messages/mark_read` | part | — | 10300 | — |
| 88 | `/api/v1/group_messages/send` | gmember+禁言 | not_group_member | 10300/10400(禁言+data.mute) | 幂等 |
| 89 | `/api/v1/group_messages/fetch` | gmember | not_group_member | 10300/10400 | — |
| 90 | `/api/v1/group_messages/update` | owner | not_resource_owner | 10300/10400 | — |
| 91 | `/api/v1/group_messages/recall` | owner | not_resource_owner | 10300/10400(超时) | — |
| 92 | `/api/v1/group_messages/mark_read` | gmember | not_group_member | 10300/10400 | — |
| 93 | `/api/v1/groups/search` | is_searchable 过滤（登录视角） | — | — | — |
| 94 | `/api/v1/groups/open_the_door` | 可见性过滤（登录视角） | — | 10300 | — |
| 95 | `/api/v1/groups/create` | — | — | 10300/10400(屏蔽标签) | 幂等 |
| 96 | `/api/v1/groups/enter` | join_method+封禁 | banned_in_group | 10300/10400 | 自然去重 |
| 97 | `/api/v1/groups/exit` | gmember | not_group_member | 10300/10400(群主) | — |
| 98 | `/api/v1/groups/fetch_joined` | self | — | — | — |
| 99 | `/api/v1/groups/update_preferences` | gmember | not_group_member | 10300/10400 | no-op |
| 100 | `/api/v1/groups/members/fetch` | members_searchable 条件（登录视角） | not_group_member | 10300 | — |
| 101 | `/api/v1/groups/members/search` | members_searchable 条件（登录视角） | not_group_member | 10300 | — |
| 102 | `/api/v1/groups/applications/fetch_submitted` | self(申请人) | — | — | — |
| 103 | `/api/v1/groups/applications/cancel` | self(申请人) | not_resource_owner | 10300/10400 | — |
| 104 | `/api/v1/groups/invites/fetch_received` | self | — | — | — |
| 105 | `/api/v1/groups/invites/accept` | self(受邀人) | — | 10300/10400 | — |
| 106 | `/api/v1/groups/invites/reject` | self(受邀人) | — | 10300/10400 | — |
| 107 | `/api/v1/feed/following` | — | — | — | — |
| 108 | `/api/v1/feed/nearby_moments` | — | — | — | — |
| 109 | `/api/v1/feed/nearby_users` | searchable 系列过滤（登录视角） | — | — | — |
| 110 | `/api/v1/feed/nearby_groups` | is_searchable 过滤（登录视角） | — | — | — |
| 111 | `/api/v1/feed/trending` | —（登录视角） | — | — | — |
| 112 | `/api/v1/search/suggest` | 可见性过滤（登录视角） | — | — | — |
| 113 | `/api/v1/media/presign_upload` | — | — | 限流 | — |
| 114 | `/api/v1/media/complete_upload` | owner(upload_id) | not_resource_owner | 10300/10400 | 自然去重 |
| 115 | `/api/v1/share/create` | 目标可见 | — | 10300 | 幂等 |
| 116 | `/api/v1/share/fetch` | self | — | — | — |
| 117 | `/api/v1/share/revoke` | owner | — | 10300(隐私一致性) | no-op |
| 118 | `/api/v1/reports/create` | 目标存在+非自举报 | — | 10300/10400 | 自然去重 |
| 119 | `/api/v1/reports/fetch` | self | — | — | — |
| 120 | `/api/v1/notify/fetch` | self | — | — | — |
| 121 | `/api/v1/notify/unread_count` | self | — | — | — |
| 122 | `/api/v1/notify/mark_read` | self(scope 显式) | — | — | — |
| 123 | `/api/v1/system/bootstrap` | self（登录态聚合） | — | — | If-None-Match |

### /api/v1/groups/admin（群管理守卫子树）

| # | 端点 | 守卫 | 10200 reason | 特有失败 | 幂等/no-op |
|---|------|------|--------------|----------|------------|
| 124 | `/api/v1/groups/admin/update` | 守卫(owner/admin) | not_group_admin | 10200/10300 | — |
| 125 | `/api/v1/groups/admin/settings` | 守卫(owner/admin) | not_group_admin | 10300 | — |
| 126 | `/api/v1/groups/admin/dismiss` | owner 专属 | not_group_owner | 10200/10300 | — |
| 127 | `/api/v1/groups/admin/invites/create` | 守卫(owner/admin) | not_group_admin | 10300/10400 | 自然去重 |
| 128 | `/api/v1/groups/admin/invites/fetch_sent` | 守卫(owner/admin) | not_group_admin | 10300/10400 | — |
| 129 | `/api/v1/groups/admin/invites/revoke` | 守卫(owner/admin) | not_group_admin | 10300/10400 | — |
| 130 | `/api/v1/groups/admin/members/fetch` | 守卫(owner/admin) | not_group_admin | 10300 | — |
| 131 | `/api/v1/groups/admin/members/set_role` | 守卫(owner/admin) | not_group_admin | 10300/10400 | no-op 同角色 |
| 132 | `/api/v1/groups/admin/members/set_mute` | 守卫(owner/admin) | not_group_admin | 10300/10400(群主保护) | no-op |
| 133 | `/api/v1/groups/admin/members/set_ban` | 守卫(owner/admin) | not_group_admin | 10300/10400(群主保护) | no-op 解封 |
| 134 | `/api/v1/groups/admin/members/fetch_banned` | 守卫(owner/admin) | not_group_admin | 10300 | — |
| 135 | `/api/v1/groups/admin/members/remove` | 守卫(owner/admin) | not_group_admin | 10300/10400(群主保护) | — |
| 136 | `/api/v1/groups/admin/applications/fetch` | 守卫(owner/admin) | not_group_admin | 10300 | — |
| 137 | `/api/v1/groups/admin/applications/approve` | 守卫(owner/admin) | not_group_admin | 10300/10400 | no-op 已接受 |
| 138 | `/api/v1/groups/admin/applications/reject` | 守卫(owner/admin) | not_group_admin | 10300/10400 | — |
| 139 | `/api/v1/groups/admin/announcements/create` | 守卫(owner/admin) | not_group_admin | 10300/10400 | 幂等 |
| 140 | `/api/v1/groups/admin/announcements/update` | 守卫(owner/admin) | not_group_admin | 10200/10300 | — |
| 141 | `/api/v1/groups/admin/announcements/delete` | 守卫(owner/admin) | not_group_admin | 10200/10300 | — |
| 142 | `/api/v1/groups/admin/transfer_ownership` | owner 专属 | not_group_owner | 10300/10400(目标非成员) | 自然去重 |
| 143 | `/api/v1/groups/admin/messages/recall` | 守卫(owner/admin) | not_group_admin | 10300/10400 | — |

### /admin/v1（平台管理员节点）

| # | 端点 | 守卫 | 10200 reason | 特有失败 | 幂等/no-op |
|---|------|------|--------------|----------|------------|
| 144 | `/admin/v1/rbac/me` | 平台管理员 | not_admin | — | — |
| 145 | `/admin/v1/rbac/fetch_permissions` | admin.rbac.manage | not_admin/insufficient_admin_permission | — | — |
| 146 | `/admin/v1/rbac/roles/fetch` | admin.rbac.manage | not_admin/insufficient_admin_permission | — | — |
| 147 | `/admin/v1/rbac/roles/create` | admin.rbac.manage | not_admin/insufficient_admin_permission | 10400 重名 | — |
| 148 | `/admin/v1/rbac/roles/update` | admin.rbac.manage | not_admin/insufficient_admin_permission | 10300/10400 | — |
| 149 | `/admin/v1/rbac/roles/delete` | admin.rbac.manage | not_admin/insufficient_admin_permission | 10300/10400 | — |
| 150 | `/admin/v1/rbac/roles/set_permissions` | admin.rbac.manage | not_admin/insufficient_admin_permission | 10300；* 仅超管 | — |
| 151 | `/admin/v1/rbac/admins/fetch` | admin.rbac.manage | not_admin/insufficient_admin_permission | — | — |
| 152 | `/admin/v1/rbac/admins/award` | admin.rbac.manage | not_admin/insufficient_admin_permission | 10300；award_all 仅超管 | — |
| 153 | `/admin/v1/rbac/admins/revoke` | admin.rbac.manage | not_admin/insufficient_admin_permission | 10300；撤超管仅超管 | — |
| 154 | `/admin/v1/audit/fetch` | admin.audit.read | not_admin/insufficient_admin_permission | — | — |
| 155 | `/admin/v1/app_config` | admin.app_config.read | not_admin/insufficient_admin_permission | — | — |
| 156 | `/admin/v1/app_config/update` | admin.app_config.write | not_admin/insufficient_admin_permission | — | — |
| 157 | `/admin/v1/users/search` | admin.users.read | not_admin/insufficient_admin_permission | — | — |
| 158 | `/admin/v1/users/open_the_door` | admin.users.read | not_admin/insufficient_admin_permission | 10300 | — |
| 159 | `/admin/v1/users/set_ban` | admin.users.write | not_admin/insufficient_admin_permission/admin_protected_target | 10300/10400 | no-op 解封 |
| 160 | `/admin/v1/groups/search` | admin.groups.read | not_admin/insufficient_admin_permission | — | — |
| 161 | `/admin/v1/groups/open_the_door` | admin.groups.read | not_admin/insufficient_admin_permission | 10300 | — |
| 162 | `/admin/v1/groups/dismiss` | admin.groups.write | not_admin/insufficient_admin_permission | 10300 | — |
| 163 | `/admin/v1/reports/fetch` | admin.reports.read | not_admin/insufficient_admin_permission | — | — |
| 164 | `/admin/v1/reports/handle` | admin.reports.write(+蕴含 users.write) | not_admin/insufficient_admin_permission | 10300/10400 | — |
| 165 | `/admin/v1/tags/search` | admin.tags.read | not_admin/insufficient_admin_permission | — | — |
| 166 | `/admin/v1/tags/delete` | admin.tags.write | not_admin/insufficient_admin_permission | 10300 | — |
| 167 | `/admin/v1/announcements/fetch` | admin.announcements.read | not_admin/insufficient_admin_permission | — | — |
| 168 | `/admin/v1/announcements/create` | admin.announcements.write | not_admin/insufficient_admin_permission | — | 幂等 |
| 169 | `/admin/v1/announcements/update` | admin.announcements.write | not_admin/insufficient_admin_permission | 10300 | — |
| 170 | `/admin/v1/announcements/delete` | admin.announcements.write | not_admin/insufficient_admin_permission | 10300 | — |

