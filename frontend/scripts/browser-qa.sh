#!/usr/bin/env bash
# HotSpot 前端浏览器巡检脚本（agent-browser）。
#
# 前置：
#   1. dev 环境已启动（仓库根：docker compose -f docker-compose.yml -f docker-compose.dev.yml up -d）
#   2. 已安装 agent-browser（npm i -g agent-browser && agent-browser install）
#
# 用法：
#   bun run browser:qa              # 桌面 20 页
#   bun run browser:qa:mobile       # 桌面 20 页 + 移动 5 页（390×844）
#   bash scripts/browser-qa.sh --base http://localhost:8889 [--mobile] [--session name]
#
# 输出：<仓库根>/.workwork/browser-check/qa-<时间戳>/（screenshots/ + report.txt）
# 检查项：横向溢出 / 坏图 / h1 数量 / TabBar 数量 / 控制台错误。

set -u

REPO_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"

AB="${AB:-}"
if [ -z "$AB" ]; then
  # 优先用户级 wrapper（沙盒自适应：socket/profile 重定向 + Chrome 沙盒判定）。
  # 沙盒内 command -v 可能解析到绕过 wrapper 的裸二进制，导致 socket 目录
  # 不可写而整轮全 FAIL（2026-10-04 实测），故 wrapper 存在时先于 command -v。
  if [ -x /Users/liyou/.local/bin/agent-browser ]; then AB=/Users/liyou/.local/bin/agent-browser;
  elif command -v agent-browser >/dev/null 2>&1; then AB="$(command -v agent-browser)";
  elif [ -x /Users/liyou/.bun/bin/agent-browser ]; then AB=/Users/liyou/.bun/bin/agent-browser;
  else echo "找不到 agent-browser，请安装或设置 AB 环境变量"; exit 1; fi
fi

NODE_BIN="${NODE_BIN:-}"
if [ -z "$NODE_BIN" ]; then
  if command -v node >/dev/null 2>&1; then NODE_BIN="$(dirname "$(command -v node)")";
  elif [ -x /Users/liyou/.nvm/versions/node/v26.2.0/bin/node ]; then NODE_BIN=/Users/liyou/.nvm/versions/node/v26.2.0/bin;
  fi
fi
[ -n "$NODE_BIN" ] && export PATH="$NODE_BIN:$PATH"

BASE="http://localhost:8888"
SESSION="hotspot-qa"
MOBILE=0
while [ $# -gt 0 ]; do
  case "$1" in
    --mobile) MOBILE=1 ;;
    --base) BASE="${2:-$BASE}"; shift ;;
    --session) SESSION="${2:-$SESSION}"; shift ;;
  esac
  shift
done

OUT="${QA_OUT_DIR:-$REPO_ROOT/.workwork/browser-check}/qa-$(date +%Y%m%d-%H%M%S)"
mkdir -p "$OUT/screenshots"
REPORT="$OUT/report.txt"
echo "HotSpot 浏览器巡检 $(date '+%Y-%m-%d %H:%M:%S')" | tee "$REPORT"
echo "BASE=$BASE SESSION=$SESSION MOBILE=$MOBILE" | tee -a "$REPORT"

ab() { "$AB" --session "$SESSION" "$@"; }

audit_js='(() => {
	const vw = document.documentElement.clientWidth;
	const issues = [];
	if (document.documentElement.scrollWidth > vw + 2) issues.push("overflow sw=" + document.documentElement.scrollWidth + " vw=" + vw);
	document.querySelectorAll("img").forEach((im) => {
		const s = im.getAttribute("src") || "";
		if (s && im.complete && im.naturalWidth === 0) issues.push("broken " + s.slice(0, 80));
	});
	const h1 = document.querySelectorAll("h1").length;
	if (h1 !== 1) issues.push("h1=" + h1);
	// 底部主导航 = MobilePortrait/TabletLayout 的 nav[aria-label="主导航"]；
	// 桌面（侧边栏）与手机横屏（无底栏）合法值为 0，故仅记录不做硬断言。
	// 注意 audit_js 整体是 shell 单引号字符串：JS 内一律用双引号（含转义），不得出现单引号。
	return { url: location.pathname, title: document.title, vw, tabBars: document.querySelectorAll("nav[aria-label=\"主导航\"]").length, issues };
})()'

guest_login() {
	ab open "$BASE/" >/dev/null 2>&1
	ab wait --load networkidle >/dev/null 2>&1
	local need_login
	need_login="$(ab eval '(() => document.body.innerText.includes("随便逛逛（访客）"))()' 2>/dev/null | tr -d '"')"
	[ "$need_login" != "true" ] && return 0
	ab eval '(() => { const b = [...document.querySelectorAll("button")].find((x) => x.textContent.includes("随便逛逛")); if (b) b.click(); return !!b; })()' >/dev/null 2>&1
	ab wait 500 >/dev/null 2>&1
	ab eval '(() => { const i = document.querySelector("input[autocomplete=nickname]") || [...document.querySelectorAll("input")].find((x) => (x.placeholder || "").includes("给自己起个名字")); if (!i) return false; Object.getOwnPropertyDescriptor(window.HTMLInputElement.prototype, "value").set.call(i, "QA巡检"); i.dispatchEvent(new Event("input", { bubbles: true })); return true; })()' >/dev/null 2>&1
	ab eval '(() => { const b = [...document.querySelectorAll("button")].find((x) => x.textContent.includes("以访客身份继续")); if (b) b.click(); return !!b; })()' >/dev/null 2>&1
	ab wait --load networkidle >/dev/null 2>&1
}

run_pages() {
	local label="$1"; shift
	local idx=0
	for path in "$@"; do
		idx=$((idx + 1))
		ab open "$BASE$path" >/dev/null 2>&1 || { echo "OPEN FAIL $path" | tee -a "$REPORT"; continue; }
		ab wait --load networkidle >/dev/null 2>&1
		echo "=== $label $path ===" | tee -a "$REPORT"
		ab eval "$audit_js" 2>&1 | tee -a "$REPORT"
		ab screenshot "$OUT/screenshots/$(printf '%02d' "$idx")-${path//\//_}.png" 2>&1 | tail -1
	done
}

guest_login

# 详情页样例 ID 运行时从匿名端点提取（DB 种子数据会变，不写死）；取不到则跳过该页。
api_first() { # $1=path $2=payload $3=json 键
	curl -s -X POST "$BASE$1" -H 'Content-Type: application/json' -d "$2" \
		| grep -o "\"$3\":\"[^\"]*\"" | head -1 | cut -d'"' -f4
}
moment_id="$(api_first /public/v1/moments/search '{"keyword":"a"}' moment_id)"
user_id="$(api_first /public/v1/feed/nearby_users '{}' user_id)"
group_id="$(api_first /public/v1/groups/search '{"keyword":"a"}' group_id)"
if [ -z "$group_id" ] && [ -n "${HOTSPOT_COMPOSE:-}" ]; then
	group_id="$(docker compose -f "$HOTSPOT_COMPOSE" exec -T postgres psql -U postgres -d hotspot -t -A -c "SELECT group_id FROM groups LIMIT 1;" 2>/dev/null)"
fi

DESKTOP_PAGES=(
	"/" "/nearby/people" "/nearby/groups" "/nearby/moments" "/notify"
	"/messages" "/me" "/me/edit" "/settings" "/map" "/search" "/me/relationships"
	"/me/moments" "/me/groups" "/me/friends" "/me/following" "/me/fans"
)
[ -n "$moment_id" ] && DESKTOP_PAGES+=("/moments/$moment_id")
[ -n "$user_id" ] && DESKTOP_PAGES+=("/users/$user_id")
[ -n "$group_id" ] && DESKTOP_PAGES+=("/groups/$group_id")
run_pages "desktop" "${DESKTOP_PAGES[@]}"

if [ "$MOBILE" = "1" ]; then
	ab set viewport 390 844 >/dev/null 2>&1
	MOBILE_PAGES=("/" "/nearby/moments" "/messages" "/map" "/settings")
	run_pages "mobile" "${MOBILE_PAGES[@]}"
fi

echo "=== console errors ===" | tee -a "$REPORT"
ab errors 2>&1 | tee -a "$REPORT" || true
ab close >/dev/null 2>&1

echo "完成：$OUT" | tee -a "$REPORT"
