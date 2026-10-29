#!/usr/bin/env bash
# HotSpot 前端浏览器巡检脚本（agent-browser）。
#
# 前置：
#   1. dev 环境已启动（仓库根：docker compose -f docker-compose.yml -f docker-compose.dev.yml up -d），
#      并已执行 ./scripts/seed-dev.sh 注入种子数据（未注入时降级为未登录态扫描）
#   2. 已安装 agent-browser（npm i -g agent-browser && agent-browser install）
#
# 登录：真实种子账号（QA_LOGIN_ID 默认 qa@hotspot.local / QA_PASSWORD 默认 Qa12345678）；
#       失败回退登录页表单，再失败按未登录态扫描（SCAN_LOGGED_IN=0）并提示先跑 seed-dev.sh。
# 详情页 ID：seed 固定 ID（moment_seed_01 / user_seed_lumos / group_seed_1）优先，
#       AUDIT_DETAIL_MOMENT / AUDIT_DETAIL_USER / AUDIT_DETAIL_GROUP 可覆盖。
#
# 用法：
#   bun run browser:qa              # 桌面 21 页
#   bun run browser:qa:mobile       # 桌面 21 页 + 移动 5 页（390×844）
#   bash scripts/browser-qa.sh --base http://localhost:8889 [--mobile] [--session name]
#
# 输出：<仓库根>/.workwork/browser-check/qa-<时间戳>/（screenshots/ + report.txt）
# 检查项：横向溢出 / 坏图 / h1 数量 / TabBar 数量 / 控制台错误。

set -u

REPO_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/lib/qa-common.sh"
find_ab
find_node
# agent-browser 定位（沙盒 wrapper 优先）与 node 定位统一收敛在 scripts/lib/qa-common.sh。

BASE="http://localhost:8888"
SESSION="hotspot-qa-$$"  # PID 后缀：主检出与 worktree 并发跑互不抢同一实例（2026-10-27）
MOBILE=0
while [ $# -gt 0 ]; do
  case "$1" in
    --mobile) MOBILE=1 ;;
    --base) BASE="${2:-$BASE}"; shift ;;
    --session) SESSION="${2:-$SESSION}"; shift ;;
  esac
  shift
done

# 前置健康检查与登录/详情页 ID 统一由 qa-common 处理（真实种子账号登录）。
require_backend

OUT="${QA_OUT_DIR:-$REPO_ROOT/.workwork/browser-check}/qa-$(date +%Y%m%d-%H%M%S)"
mkdir -p "$OUT/screenshots"
REPORT="$OUT/report.txt"
echo "HotSpot 浏览器巡检 $(date '+%Y-%m-%d %H:%M:%S')" | tee "$REPORT"
echo "BASE=$BASE SESSION=$SESSION MOBILE=$MOBILE" | tee -a "$REPORT"

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

run_pages() {
	local label="$1"; shift
	local idx=0
	for path in "$@"; do
		idx=$((idx + 1))
		ab open "$BASE$path" >/dev/null 2>&1 || { echo "OPEN FAIL $path" | tee -a "$REPORT"; continue; }
		ab wait --load networkidle >/dev/null 2>&1
		echo "=== $label $path ===" | tee -a "$REPORT"
		ab eval "$audit_js" 2>&1 | tee -a "$REPORT"
		ab screenshot "$OUT/screenshots/$(printf '%02d' "$idx")-${label}-${path//\//_}.png" 2>&1 | tail -1
	done
}

# 登录：qa-common.ensure_login（真实 seed 账号 → 登录页表单回退 → 未登录态扫描）。
ensure_login
# 页面清单与详情页 ID 解析统一收敛在 scripts/lib/qa-common.sh（seed 固定 ID 优先，
# 不可用时回退 public API / 首页 href，仍空则跳过并记录）。QA 保持 18 页基线 + 详情页。
build_page_lists
echo "SCAN_LOGGED_IN=$SCAN_LOGGED_IN DETAIL_MOMENT=$DETAIL_MOMENT DETAIL_USER=$DETAIL_USER DETAIL_GROUP=$DETAIL_GROUP" | tee -a "$REPORT"
if [ "$SCAN_LOGGED_IN" = "0" ]; then
	echo "WARN: 未登录态扫描（seed 未注入或 QA 账号登录失败；先执行 ./scripts/seed-dev.sh）" | tee -a "$REPORT"
fi
QA_PAGES=("${QA_DESKTOP_PAGES[@]}")
[ -n "$DETAIL_MOMENT" ] && QA_PAGES+=("/moments/$DETAIL_MOMENT")
[ -n "$DETAIL_USER" ] && QA_PAGES+=("/users/$DETAIL_USER")
[ -n "$DETAIL_GROUP" ] && QA_PAGES+=("/groups/$DETAIL_GROUP")
run_pages "desktop" "${QA_PAGES[@]}"

if [ "$MOBILE" = "1" ]; then
	ab set viewport 390 844 >/dev/null 2>&1
	MOBILE_PAGES=("/" "/nearby/moments" "/messages" "/map" "/settings")
	run_pages "mobile" "${MOBILE_PAGES[@]}"
fi

echo "=== console errors ===" | tee -a "$REPORT"
ab errors 2>&1 | tee -a "$REPORT" || true
ab close >/dev/null 2>&1

echo "完成：$OUT" | tee -a "$REPORT"
