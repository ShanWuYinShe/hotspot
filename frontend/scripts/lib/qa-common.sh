#!/usr/bin/env bash
# UI 质量门禁脚本共享库（browser-qa / browser-a11y / browser-touch）。
#
# 统一封装：agent-browser 定位（沙盒 wrapper 优先）、node/bun 定位、base/session 参数、
# 前置健康检查、真实账号登录（失败回退登录页表单 → 未登录态扫描）、详情页种子 ID
# 解析（public API / 首页链接兜底）、页面清单构建、输出目录。
#
# 登录账号（仅限本地 dev，契约见 seed-spec §2）：
#   QA_LOGIN_ID  默认 qa@hotspot.local
#   QA_PASSWORD  默认 Qa12345678
# 种子口令若在 .env 里用 SEED_QA_PASSWORD 覆盖，跑 QA 时需同步设置 QA_PASSWORD。
# 未注入 seed / 登录失败时降级为未登录态扫描，并提示先跑 ./scripts/seed-dev.sh。
#
# 详情页 / 聊天页 ID 覆盖（显式设置即生效；置空则跳过 seed 固定 ID、回退到原 API 发现）：
#   AUDIT_DETAIL_MOMENT / AUDIT_DETAIL_USER / AUDIT_DETAIL_GROUP（旧口径）
#   SEED_MOMENT_ID / SEED_USER_ID / SEED_GROUP_ID（seed-spec 口径别名）
#   AUDIT_CHAT_USER / AUDIT_CHAT_GROUP（默认 user_seed_sophia / group_seed_1）
#
# 用法：source "$(dirname "$0")/lib/qa-common.sh"
# 前置：dev 环境已启动（推荐已执行 ./scripts/seed-dev.sh）；已安装 agent-browser。

set -u

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# seed 固定 ID（见 worktree 内 .workwork/seed-spec.md §2/§3；均可用环境变量覆盖）。
SEED_DETAIL_MOMENT_DEFAULT="moment_seed_01"
SEED_DETAIL_USER_DEFAULT="user_seed_lumos"
SEED_DETAIL_GROUP_DEFAULT="group_seed_1"
SEED_CHAT_USER_DEFAULT="user_seed_sophia"
SEED_CHAT_GROUP_DEFAULT="group_seed_1"
# QA 主账号（seed-spec §2）。
QA_LOGIN_ID_DEFAULT="qa@hotspot.local"
QA_PASSWORD_DEFAULT="Qa12345678"
# 管理员种子账号（seed-spec §2；QA_LOGIN_ID 可显式覆盖成它，用来一次登录扫全部管理页）。
QA_ADMIN_LOGIN_ID_DEFAULT="admin@hotspot.local"
# seed 账号对应的固定 user_id（用于「复用已有会话」时判定是否种子账号）。
QA_SEED_USER_IDS="user_seed_lumos user_seed_admin"

find_ab() {
  AB_BIN=""
  if [ -n "${AB:-}" ]; then AB_BIN="$AB";
  elif [ -x /Users/liyou/.local/bin/agent-browser ]; then AB_BIN=/Users/liyou/.local/bin/agent-browser;
  elif command -v agent-browser >/dev/null 2>&1; then AB_BIN="$(command -v agent-browser)";
  elif [ -x /Users/liyou/.bun/bin/agent-browser ]; then AB_BIN=/Users/liyou/.bun/bin/agent-browser;
  else echo "找不到 agent-browser，请安装或设置 AB 环境变量"; exit 1; fi
  export AB_BIN
}

find_node() {
  if command -v node >/dev/null 2>&1; then NODE_BIN="$(dirname "$(command -v node)")";
  elif [ -x /Users/liyou/.nvm/versions/node/v26.2.0/bin/node ]; then NODE_BIN=/Users/liyou/.nvm/versions/node/v26.2.0/bin;
  else NODE_BIN=""; fi
  [ -n "$NODE_BIN" ] && export PATH="$NODE_BIN:$PATH"
  if command -v bun >/dev/null 2>&1; then RUNNER="bun";
  elif [ -n "$NODE_BIN" ]; then RUNNER="$NODE_BIN/node";
  else echo "找不到 bun/node，无法做报告汇总"; exit 1; fi
  export RUNNER
}

ab() { "$AB_BIN" --session "$SESSION" "$@"; }

# JSON 字符串转义：把值安全嵌入浏览器内 JS 的双引号字符串字面量（种子账号仅含字母数字/邮箱）。
js_escape() {
  printf '%s' "$1" | sed -e 's/\\/\\\\/g' -e 's/"/\\"/g'
}

# 从响应 / eval 结果中取值（兼容裸 JSON 与 agent-browser 对字符串结果加引号的形态）。
qa_json_code() { # $1=JSON 文本
  printf '%s' "$1" | sed -e 's/\\"/"/g' | sed -n 's/.*"code"[[:space:]]*:[[:space:]]*\([-0-9]*\).*/\1/p' | head -1
}

qa_json_message() { # $1=JSON 文本
  printf '%s' "$1" | sed -e 's/\\"/"/g' | sed -n 's/.*"message":"\([^"]*\)".*/\1/p' | head -1
}

qa_json_field() { # $1=JSON 文本 $2=键名（取字符串值）
  printf '%s' "$1" | sed -e 's/\\"/"/g' | sed -n "s/.*\"$2\":\"\([^\"]*\)\".*/\1/p" | head -1
}

# 前置健康检查（spec api_spec_public.md → system/health：HTTP 200 且 code=0 才可用）。
# 失败即明确报错并退出，避免浏览器巡检在「后端没起来」时产出成片假失败。
require_backend() {
  local body status code message
  body="$(curl -sS -f -m 8 -X POST -H 'Content-Type: application/json' -d '{}' "$BASE/public/v1/system/health" 2>/dev/null)"
  if [ $? -ne 0 ] || [ -z "$body" ]; then
    echo "✗ 后端不可达：POST $BASE/public/v1/system/health"
    echo "  请先确认 dev 环境已启动：docker compose -f docker-compose.yml -f docker-compose.dev.yml up -d"
    echo "  首次启动 / 数据缺失后执行：./scripts/seed-dev.sh"
    echo "  worktree 或自定义端口请用 --base 指向对应网关（如 http://localhost:8889）。"
    exit 1
  fi
  code="$(qa_json_code "$body")"
  if [ "$code" != "0" ]; then
    message="$(qa_json_message "$body")"
    echo "✗ 后端健康检查失败：code=${code:-未知} message=${message:-无}"
    echo "  修复：docker compose -f docker-compose.yml -f docker-compose.dev.yml ps 检查 backend/postgres/redis，"
    echo "        必要时重启 dev 环境后重跑；QA 登录依赖 seed 数据，请执行 ./scripts/seed-dev.sh。"
    exit 1
  fi
  status="$(qa_json_field "$body" status)"
  echo "✓ 后端健康：$BASE（status=${status:-unknown}）"
}

# 轮询浏览器内异步探针写入的 window 变量（最多约 12s）；拿到 JSON 后打印到 stdout。
# 必须经 JSON.stringify 取值：agent-browser 对对象返回值做 pretty-print 多行输出，
# 直接 eval 对象再 tail -1 只会拿到最后的 "}"（2026-10-29 实测：登录成功被误判为失败）。
qa_poll_window_json() { # $1=取值表达式，如 window.__qaLogin
  local expr="$1" i=0 raw
  while [ "$i" -lt 40 ]; do
    raw="$(ab eval "JSON.stringify($expr)" 2>/dev/null | tail -1)"
    case "$raw" in
      ""|null|'"null"'|undefined|'"undefined"') ;;
      *) printf '%s' "$raw"; return 0 ;;
    esac
    sleep 0.3
    i=$((i + 1))
  done
  return 1
}

# 清理 MSW 时代残留（本重构已删除前端 mock）：agent-browser 各会话共享 owner Chrome 的
# profile，历史 run 可能留下已注册的 mock service worker / localStorage 标记。清掉后重载，
# 避免真实登录请求被旧 SW 拦截而拿到假的成功响应。
qa_cleanup_legacy_mock_state() {
  ab eval '(() => {
    try {
      localStorage.removeItem("hotspot:mock-auth-logged-in");
      localStorage.removeItem("hotspot:mock-enabled");
      if (navigator.serviceWorker && navigator.serviceWorker.getRegistrations) {
        navigator.serviceWorker.getRegistrations().then((rs) => { rs.forEach((r) => { r.unregister(); }); }).catch(() => {});
      }
    } catch (e) {}
    return true;
  })()' >/dev/null 2>&1
}

# 当前浏览器会话是否已登录：POST /api/v1/auth/session（Cookie 认证），code=0 即已登录。
qa_browser_session_ok() {
  local js raw code
  js='(() => {
    window.__qaSession = null;
    fetch("/api/v1/auth/session", { method: "POST", credentials: "include", headers: { "Content-Type": "application/json" }, body: "{}" })
      .then((r) => r.json().catch(() => ({ code: -1 })))
      .then((j) => { window.__qaSession = j; })
      .catch(() => { window.__qaSession = { code: -1 }; });
    return true;
  })()'
  ab eval "$js" >/dev/null 2>&1
  raw="$(qa_poll_window_json 'window.__qaSession')" || return 1
  code="$(qa_json_code "$raw")"
  [ "$code" = "0" ]
}

# 已登录会话对应的 user_id（POST /api/v1/users/account），用于「复用已有会话」时识别种子账号。
qa_browser_account_user_id() {
  local js raw
  js='(() => {
    window.__qaAccount = null;
    fetch("/api/v1/users/account", { method: "POST", credentials: "include", headers: { "Content-Type": "application/json" }, body: "{}" })
      .then((r) => r.json().catch(() => ({ code: -1 })))
      .then((j) => { window.__qaAccount = j; })
      .catch(() => { window.__qaAccount = { code: -1 }; });
    return true;
  })()'
  ab eval "$js" >/dev/null 2>&1
  raw="$(qa_poll_window_json 'window.__qaAccount')" || return 1
  [ "$(qa_json_code "$raw")" = "0" ] || return 1
  qa_json_field "$raw" user_id
}

# 登录账号是否为 seed 账号（默认 qa / admin）：成功登录时可就地判定，无需再探测。
qa_mark_seed_account() { # $1=login_id
  case "$1" in
    "$QA_LOGIN_ID_DEFAULT"|"$QA_ADMIN_LOGIN_ID_DEFAULT") QA_SEED_ACCOUNT=1 ;;
  esac
}

# 真实账号登录：浏览器内同源 POST /public/v1/auth/signin（credentials: include → HttpOnly Cookie）。
# signin 的 Web 口径依赖 bootstrap 下发的 device_id Cookie，故先补一次（幂等）。
qa_browser_signin() { # $1=login_id $2=password
  local login_id password js
  login_id="$(js_escape "$1")"
  password="$(js_escape "$2")"
  js='(() => {
    window.__qaLogin = null;
    const done = (v) => { window.__qaLogin = v; };
    (async () => {
      try {
        await fetch("/public/v1/system/bootstrap", { method: "POST", credentials: "include", headers: { "Content-Type": "application/json" }, body: "{}" }).catch(() => null);
        const res = await fetch("/public/v1/auth/signin", { method: "POST", credentials: "include", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ login_id: "'"$login_id"'", password: "'"$password"'", client_type: "web", device_id: null }) });
        let json = null;
        try { json = await res.json(); } catch (e) { json = { code: -1, message: "signin 响应非 JSON（HTTP " + res.status + "）" }; }
        done(json || { code: -1, message: "signin 空响应" });
      } catch (e) {
        done({ code: -1, message: "signin 请求失败：" + String(e) });
      }
    })();
    return true;
  })()'
  ab eval "$js" >/dev/null 2>&1
}

# 回退登录：登录页表单（DOM 契约见 src/components/auth/AuthForm.svelte：
# #auth-field-loginId / #auth-field-password / form button[type=submit]）。
qa_browser_ui_login() { # $1=login_id $2=password
  local login_id password js raw
  login_id="$(js_escape "$1")"
  password="$(js_escape "$2")"
  ab open "$BASE/login" >/dev/null 2>&1
  ab wait --load networkidle >/dev/null 2>&1
  js='(() => {
    const setValue = (sel, val) => {
      const el = document.querySelector(sel);
      if (!el) return false;
      const setter = Object.getOwnPropertyDescriptor(window.HTMLInputElement.prototype, "value").set;
      setter.call(el, val);
      el.dispatchEvent(new Event("input", { bubbles: true }));
      return true;
    };
    const idOk = setValue("#auth-field-loginId", "'"$login_id"'");
    const pwOk = setValue("#auth-field-password", "'"$password"'");
    const btn = document.querySelector("form button[type=submit]");
    if (btn) btn.click();
    return { idOk: idOk, pwOk: pwOk, submit: !!btn };
  })()'
  # 同 qa_poll_window_json：对象返回值必须 JSON.stringify 成单行再匹配。
  raw="$(ab eval "JSON.stringify($js)" 2>/dev/null | tail -1)"
  case "$raw" in
    *'"submit":true'*) ;;
    *) echo "  登录页未找到表单提交按钮（页面结构可能已变更）"; return 1 ;;
  esac
  ab wait 800 >/dev/null 2>&1
  ab wait --load networkidle >/dev/null 2>&1
  qa_browser_session_ok
}

# 登录准备：真实账号（QA_LOGIN_ID/QA_PASSWORD）→ 登录页表单回退 → 未登录态扫描回退。
# 结果写入 SCAN_LOGGED_IN（1=已登录，0=未登录），供调用脚本判断/记录。
ensure_login() {
  local login_id="${QA_LOGIN_ID:-$QA_LOGIN_ID_DEFAULT}"
  local password="${QA_PASSWORD:-$QA_PASSWORD_DEFAULT}"
  local raw code message account_user_id
  # 供 resolve_detail_ids 判断「seed 账号已登录 = seed 数据已注入」（默认账号复用/登录成功时置 1）。
  QA_SEED_ACCOUNT=0

  # 1) 打开首页：应用冷启动会 POST /public/v1/system/bootstrap 下发 device_id Cookie（signin 前置）。
  ab open "$BASE/" >/dev/null 2>&1
  ab wait --load networkidle >/dev/null 2>&1
  qa_cleanup_legacy_mock_state
  ab open "$BASE/" >/dev/null 2>&1
  ab wait --load networkidle >/dev/null 2>&1

  # 2) 复用已有会话（agent-browser 各 session 共享 owner Chrome 的 Cookie）。
  if qa_browser_session_ok; then
    SCAN_LOGGED_IN=1
    account_user_id="$(qa_browser_account_user_id 2>/dev/null || true)"
    case " $QA_SEED_USER_IDS " in
      *" $account_user_id "*) [ -n "$account_user_id" ] && QA_SEED_ACCOUNT=1 ;;
    esac
    echo "登录态：复用已有会话（SCAN_LOGGED_IN=1，user_id=${account_user_id:-未知}）"
    return 0
  fi

  # 3) 真实账号登录，成功后 reload 让 SPA 重新引导会话。
  echo "登录态：尝试真实账号 $login_id（POST /public/v1/auth/signin）"
  qa_browser_signin "$login_id" "$password"
  if raw="$(qa_poll_window_json 'window.__qaLogin')"; then
    code="$(qa_json_code "$raw")"
    message="$(qa_json_message "$raw")"
    if [ "$code" = "0" ]; then
      ab open "$BASE/" >/dev/null 2>&1
      ab wait --load networkidle >/dev/null 2>&1
      if qa_browser_session_ok; then
        SCAN_LOGGED_IN=1
        qa_mark_seed_account "$login_id"
        echo "登录态：登录成功 $login_id（SCAN_LOGGED_IN=1）"
        return 0
      fi
      echo "真实账号登录返回成功，但会话校验未通过；回退界面登录"
    else
      echo "真实账号登录失败：code=${code:-未知} message=${message:-无}"
    fi
  else
    echo "真实账号登录超时（浏览器内 fetch 无响应）"
  fi

  # 4) 回退：登录页表单登录（与前端 AuthForm 提交路径一致）。
  echo "登录态：回退登录页表单登录"
  if qa_browser_ui_login "$login_id" "$password"; then
    SCAN_LOGGED_IN=1
    qa_mark_seed_account "$login_id"
    echo "登录态：表单登录成功 $login_id（SCAN_LOGGED_IN=1）"
    return 0
  fi

  # 5) 回退：未登录态扫描（公开页面仍可巡检；个人/管理页面会呈现未登录态）。
  ab open "$BASE/" >/dev/null 2>&1
  ab wait --load networkidle >/dev/null 2>&1
  SCAN_LOGGED_IN=0
  echo "登录失败：请先确认 dev 后端已启动并执行 ./scripts/seed-dev.sh"
  echo "（本次按未登录态扫描；账号可用 QA_LOGIN_ID / QA_PASSWORD 覆盖，见 frontend/AGENTS.md）"
  return 1
}

guest_login() { ensure_login "$@"; }  # 兼容旧调用名：登录语义已改为真实账号

api_first() { # $1=path $2=payload $3=json 键
  curl -s -m 8 -X POST "$BASE$1" -H 'Content-Type: application/json' -d "$2" 2>/dev/null \
    | grep -o "\"$3\":\"[^\"]*\"" | head -1 | cut -d'"' -f4
}

# seed 账号（默认 qa/admin）能登录成功 = seed 数据必然已注入（该账号由 seed 创建）。
# 此时直接采用 seed 固定 ID，避免 anonymous public 探测在非公开资源（如仅登录可见的详情）
# 上误判为「不存在」后错误回退到 API 发现。
qa_seed_data_assumed() {
  [ "${SCAN_LOGGED_IN:-0}" = "1" ] && [ "${QA_SEED_ACCOUNT:-0}" = "1" ]
}

# 检测 seed 固定详情 ID 是否真实存在（public open_the_door code=0）。
qa_detail_exists() { # $1=moment|user|group $2=id
  local kind="$1" id="$2" path body
  [ -n "$id" ] || return 1
  case "$kind" in
    moment) path="/public/v1/moments/open_the_door"; body="{\"moment_id\":\"$id\"}" ;;
    user) path="/public/v1/users/open_the_door"; body="{\"user_id\":\"$id\",\"include_recent_moments\":false}" ;;
    group) path="/public/v1/groups/open_the_door"; body="{\"group_id\":\"$id\"}" ;;
    *) return 1 ;;
  esac
  curl -s -m 8 -X POST "$BASE$path" -H 'Content-Type: application/json' -d "$body" 2>/dev/null \
    | grep -q '"code":0'
}

# 详情页 ID 发现：public API 优先；空则从首页链接提取（/moments/x、/users/x、/groups/x）。
scrape_detail_ids() {
  local js
  js='(() => {
    var pick = function (re) {
      var as = [...document.querySelectorAll("a[href]")].map(function (x) { return x.getAttribute("href"); });
      for (var i = 0; i < as.length; i++) { var m = (as[i] || "").match(re); if (m) return m[1]; }
      return "";
    };
    return JSON.stringify({
      moment: pick(/^\/moments\/([^/?#]+)/),
      user: pick(/^\/users\/([^/?#]+)/),
      group: pick(/^\/groups\/([^/?#]+)/)
    });
  })()'
  local raw
  raw="$(ab eval "$js" 2>/dev/null | tail -1 | tr -d '"')"
  printf '%s' "$raw" | sed -n 's/.*"moment":"\([^"]*\)".*/\1/p'
  printf '%s' "$raw" | sed -n 's/.*"user":"\([^"]*\)".*/\1/p'
  printf '%s' "$raw" | sed -n 's/.*"group":"\([^"]*\)".*/\1/p'
}

# 详情页 ID 解析（每个 ID 独立）：env 覆盖 > seed 固定 ID（seed 账号已登录，或 public 探测存在）
# > 原 API 发现 > 首页链接 > 跳过。
resolve_detail_ids() {
  DETAIL_MOMENT=""
  DETAIL_USER=""
  DETAIL_GROUP=""
  if [ -n "${AUDIT_DETAIL_MOMENT+x}" ]; then
    DETAIL_MOMENT="$AUDIT_DETAIL_MOMENT"
  elif [ -n "${SEED_MOMENT_ID+x}" ]; then
    DETAIL_MOMENT="$SEED_MOMENT_ID"
  elif qa_seed_data_assumed || qa_detail_exists moment "$SEED_DETAIL_MOMENT_DEFAULT"; then
    DETAIL_MOMENT="$SEED_DETAIL_MOMENT_DEFAULT"
  fi
  if [ -n "${AUDIT_DETAIL_USER+x}" ]; then
    DETAIL_USER="$AUDIT_DETAIL_USER"
  elif [ -n "${SEED_USER_ID+x}" ]; then
    DETAIL_USER="$SEED_USER_ID"
  elif qa_seed_data_assumed || qa_detail_exists user "$SEED_DETAIL_USER_DEFAULT"; then
    DETAIL_USER="$SEED_DETAIL_USER_DEFAULT"
  fi
  if [ -n "${AUDIT_DETAIL_GROUP+x}" ]; then
    DETAIL_GROUP="$AUDIT_DETAIL_GROUP"
  elif [ -n "${SEED_GROUP_ID+x}" ]; then
    DETAIL_GROUP="$SEED_GROUP_ID"
  elif qa_seed_data_assumed || qa_detail_exists group "$SEED_DETAIL_GROUP_DEFAULT"; then
    DETAIL_GROUP="$SEED_DETAIL_GROUP_DEFAULT"
  fi

  # 回退：现有 public API 发现（无 seed 数据的真实环境仍能取到详情页 ID）。
  if [ -z "$DETAIL_MOMENT" ]; then
    DETAIL_MOMENT="$(api_first /public/v1/moments/search '{"sort_by":"popular"}' moment_id)"
  fi
  if [ -z "$DETAIL_USER" ]; then
    DETAIL_USER="$(api_first /public/v1/users/search '{"radius":50000}' user_id)"
    [ -z "$DETAIL_USER" ] && DETAIL_USER="$(api_first /public/v1/moments/search '{"sort_by":"popular"}' user_id)"
  fi
  if [ -z "$DETAIL_GROUP" ]; then
    DETAIL_GROUP="$(api_first /public/v1/groups/search '{"radius":50000}' group_id)"
    [ -z "$DETAIL_GROUP" ] && DETAIL_GROUP="$(api_first /public/v1/groups/search '{"keyword":"a"}' group_id)"
  fi
  if [ -z "$DETAIL_GROUP" ] && [ -n "${HOTSPOT_COMPOSE:-}" ]; then
    DETAIL_GROUP="$(docker compose -f "$HOTSPOT_COMPOSE" exec -T postgres psql -U postgres -d hotspot -t -A -c "SELECT group_id FROM groups LIMIT 1;" 2>/dev/null | tr -d '[:space:]')"
  fi
  if [ -z "$DETAIL_MOMENT" ] || [ -z "$DETAIL_USER" ] || [ -z "$DETAIL_GROUP" ]; then
    local scraped
    scraped="$(scrape_detail_ids)"
    [ -z "$DETAIL_MOMENT" ] && DETAIL_MOMENT="$(printf '%s\n' "$scraped" | sed -n '1p')"
    [ -z "$DETAIL_USER" ] && DETAIL_USER="$(printf '%s\n' "$scraped" | sed -n '2p')"
    [ -z "$DETAIL_GROUP" ] && DETAIL_GROUP="$(printf '%s\n' "$scraped" | sed -n '3p')"
  fi
  local detail_note=""
  [ -z "$DETAIL_MOMENT" ] && detail_note="$detail_note moments"
  [ -z "$DETAIL_USER" ] && detail_note="$detail_note users"
  [ -z "$DETAIL_GROUP" ] && detail_note="$detail_note groups"
  [ -n "$detail_note" ] && echo "提示：以下详情页无可用 ID，已跳过：$detail_note"
  echo "详情页 ID：moment=${DETAIL_MOMENT:-（跳过）} user=${DETAIL_USER:-（跳过）} group=${DETAIL_GROUP:-（跳过）}"
  return 0
}

# 页面清单：QA 基线 18 页 + R5 审计扩展 17 页 + 详情页 3 个（seed 固定 ID → API → 首页 href → 跳过）。
# 详情可取时桌面 38 页；a11y 桌面 38 + 移动 5 = 43 项检查（≥ 39 项门禁目标）。
build_page_lists() {
  QA_DESKTOP_PAGES=(
    "/" "/nearby/people" "/nearby/groups" "/nearby/moments" "/notify"
    "/messages" "/me" "/me/edit" "/settings" "/map" "/search" "/me/relationships"
    "/me/moments" "/me/groups" "/me/reports" "/me/friends" "/me/following" "/me/fans"
  )
  AUDIT_EXTRA_PAGES=(
    "/nearby" "/login" "/register"
    "/admin" "/admin/announcements" "/admin/audit" "/admin/config" "/admin/groups"
    "/admin/rbac" "/admin/reports" "/admin/stats" "/admin/tags" "/admin/users"
    "/messages/${AUDIT_CHAT_USER:-$SEED_CHAT_USER_DEFAULT}" "/messages/group/${AUDIT_CHAT_GROUP:-$SEED_CHAT_GROUP_DEFAULT}"
    "/search?q=%E4%BA%91" "/share/SEEDSUNSET23"
  )
  MOBILE_PAGES=("/" "/nearby/people" "/search" "/settings" "/notify")

  resolve_detail_ids
  # bash 3.2 + set -u 下空数组展开会报 unbound，故用非空守卫逐个追加。
  DESKTOP_PAGES=("${QA_DESKTOP_PAGES[@]}" "${AUDIT_EXTRA_PAGES[@]}")
  [ -n "$DETAIL_MOMENT" ] && DESKTOP_PAGES+=("/moments/$DETAIL_MOMENT")
  [ -n "$DETAIL_USER" ] && DESKTOP_PAGES+=("/users/$DETAIL_USER")
  [ -n "$DETAIL_GROUP" ] && DESKTOP_PAGES+=("/groups/$DETAIL_GROUP")
}

# 触控门禁基线页面（r5 20 页口径 + QA 详情页）：默认只扫这批；--full 追加审计扩展页
# （管理后台 audit/stats/tags/groups/rbac/reports/announcements、登录/注册、聊天、搜索词、分享）。
# 扩展页存在 r5 未登记的离群（2026-10-27 实测：admin/audit 文字链、admin/stats 图表/明细、
# 聊天头部整行、share CTA），登记为待修 backlog，修完前不纳入默认门禁（见 docs/frontend-ui-audit.md）。
build_touch_pages() {
  TOUCH_PAGES=("${QA_DESKTOP_PAGES[@]}" "/login" "/admin" "/admin/users" "/admin/config")
  [ -n "${DETAIL_MOMENT:-}" ] && TOUCH_PAGES+=("/moments/$DETAIL_MOMENT")
  [ -n "${DETAIL_USER:-}" ] && TOUCH_PAGES+=("/users/$DETAIL_USER")
  [ -n "${DETAIL_GROUP:-}" ] && TOUCH_PAGES+=("/groups/$DETAIL_GROUP")
}

make_out_dir() { # $1=prefix
  local out
  out="${AUDIT_OUT_DIR:-$REPO_ROOT/.workwork/browser-check}/$1-$(date +%Y%m%d-%H%M%S)"
  mkdir -p "$out/raw"
  echo "$out"
}
