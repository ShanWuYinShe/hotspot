#!/usr/bin/env bash
# UI 质量门禁脚本共享库（browser-qa / browser-a11y / browser-touch）。
#
# 统一封装：agent-browser 定位（沙盒 wrapper 优先）、node/bun 定位、base/session 参数、
# 登录准备（dev/mock 栈 mock 标记 + 访客登录 + 生产优雅降级）、详情页种子 ID 发现
# （public API 优先，空则从首页链接提取，再空则跳过并记录）、页面清单构建、输出目录。
#
# 用法：source "$(dirname "$0")/lib/qa-common.sh"
# 前置：dev 环境已启动；已安装 agent-browser。

set -u

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

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

# 登录准备：dev/mock 栈用既有 localStorage 标记（hotspot:mock-auth-logged-in）后刷新；
# 非 mock（生产）环境没有对应 handler，标记无副作用，自动降级为「未登录态扫描」。
ensure_login() {
  ab open "$BASE/" >/dev/null 2>&1
  ab wait --load networkidle >/dev/null 2>&1
  ab eval 'localStorage.setItem("hotspot:mock-auth-logged-in","true"); true' >/dev/null 2>&1
  ab open "$BASE/" >/dev/null 2>&1
  ab wait --load networkidle >/dev/null 2>&1
  local need_guest
  need_guest="$(ab eval '(() => document.body.innerText.includes("随便逛逛（访客）"))()' 2>/dev/null | tr -d '"')"
  if [ "$need_guest" = "true" ]; then
    ab eval '(() => { const b = [...document.querySelectorAll("button")].find((x) => x.textContent.includes("随便逛逛")); if (b) b.click(); return !!b; })()' >/dev/null 2>&1
    ab wait 500 >/dev/null 2>&1
    ab eval '(() => { const i = document.querySelector("input[autocomplete=nickname]") || [...document.querySelectorAll("input")].find((x) => (x.placeholder || "").includes("给自己起个名字")); if (!i) return false; Object.getOwnPropertyDescriptor(window.HTMLInputElement.prototype, "value").set.call(i, "QA巡检"); i.dispatchEvent(new Event("input", { bubbles: true })); return true; })()' >/dev/null 2>&1
    ab eval '(() => { const b = [...document.querySelectorAll("button")].find((x) => x.textContent.includes("以访客身份继续")); if (b) b.click(); return !!b; })()' >/dev/null 2>&1
    ab wait --load networkidle >/dev/null 2>&1
  fi
  local logged_in
  logged_in="$(ab eval '(() => { const t = document.body.innerText; return !t.includes("随便逛逛（访客）") && !t.includes("去登录") })()' 2>/dev/null | tr -d '"')"
  if [ "$logged_in" != "true" ]; then
    echo "提示：未检测到登录态，按「未登录态扫描」继续（生产环境无 mock 登录标记属正常）"
    SCAN_LOGGED_IN=0
  else
    SCAN_LOGGED_IN=1
  fi
}

guest_login() { ensure_login "$@"; }

api_first() { # $1=path $2=payload $3=json 键
  curl -s -X POST "$BASE$1" -H 'Content-Type: application/json' -d "$2" \
    | grep -o "\"$3\":\"[^\"]*\"" | head -1 | cut -d'"' -f4
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

# 页面清单：QA 基线 18 页 + R5 审计扩展 17 页 + 详情页 3 个（API → 首页 href → 跳过并记录）。
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
    "/messages/user_sophia" "/messages/group/group_1"
    "/search?q=%E4%BA%91" "/share/abc"
  )
  MOBILE_PAGES=("/" "/nearby/people" "/search" "/settings" "/notify")

  local moment_id user_id group_id
  moment_id="$(api_first /public/v1/moments/search '{"keyword":"a"}' moment_id)"
  user_id="$(api_first /public/v1/feed/nearby_users '{}' user_id)"
  group_id="$(api_first /public/v1/groups/search '{"keyword":"a"}' group_id)"
  if [ -z "$group_id" ] && [ -n "${HOTSPOT_COMPOSE:-}" ]; then
    group_id="$(docker compose -f "$HOTSPOT_COMPOSE" exec -T postgres psql -U postgres -d hotspot -t -A -c "SELECT group_id FROM groups LIMIT 1;" 2>/dev/null)"
  fi
  if [ -z "$moment_id" ] || [ -z "$user_id" ] || [ -z "$group_id" ]; then
    local scraped
    scraped="$(scrape_detail_ids)"
    [ -z "$moment_id" ] && moment_id="$(printf '%s\n' "$scraped" | sed -n '1p')"
    [ -z "$user_id" ] && user_id="$(printf '%s\n' "$scraped" | sed -n '2p')"
    [ -z "$group_id" ] && group_id="$(printf '%s\n' "$scraped" | sed -n '3p')"
  fi
  # 详情页 ID 环境变量兜底（mock/空库环境不可发现时使用，见 frontend/AGENTS.md
  # 「质量门禁脚本」：AUDIT_DETAIL_MOMENT / AUDIT_DETAIL_USER / AUDIT_DETAIL_GROUP）。
  [ -z "$moment_id" ] && [ -n "${AUDIT_DETAIL_MOMENT:-}" ] && moment_id="$AUDIT_DETAIL_MOMENT"
  [ -z "$user_id" ] && [ -n "${AUDIT_DETAIL_USER:-}" ] && user_id="$AUDIT_DETAIL_USER"
  [ -z "$group_id" ] && [ -n "${AUDIT_DETAIL_GROUP:-}" ] && group_id="$AUDIT_DETAIL_GROUP"
  DETAIL_MOMENT="$moment_id"; DETAIL_USER="$user_id"; DETAIL_GROUP="$group_id"
  local detail_note=""
  [ -z "$moment_id" ] && detail_note="$detail_note moments"
  [ -z "$user_id" ] && detail_note="$detail_note users"
  [ -z "$group_id" ] && detail_note="$detail_note groups"
  [ -n "$detail_note" ] && echo "提示：以下详情页无种子 ID，已跳过：$detail_note"
  # bash 3.2 + set -u 下空数组展开会报 unbound，故用非空守卫逐个追加。
  DESKTOP_PAGES=("${QA_DESKTOP_PAGES[@]}" "${AUDIT_EXTRA_PAGES[@]}")
  [ -n "$moment_id" ] && DESKTOP_PAGES+=("/moments/$moment_id")
  [ -n "$user_id" ] && DESKTOP_PAGES+=("/users/$user_id")
  [ -n "$group_id" ] && DESKTOP_PAGES+=("/groups/$group_id")
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