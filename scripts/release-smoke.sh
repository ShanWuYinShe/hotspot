#!/usr/bin/env bash
# 发版后线上冒烟：校验「版本三处对齐 + 前端入口可达 + Release 资产齐全且校验和正确」。
# 前置：发布流程已完成（tag 已推、Release workflow 成功；默认还要求 APK 已附）；
# gh 已登录且仓库可访问。
#
# 用法：
#   ./scripts/release-smoke.sh --version v1.15.4
#   ./scripts/release-smoke.sh --version v1.15.4 --base https://chaoset.com
# 参数：--version <vX.Y.Z>（必填）
#       --base <url>（默认 https://chaoset.com）
#       --repo <owner/name>
#       --skip-apk  跳过 APK 资产检查。APK 在 Release 工作流之后由本机
#                   scripts/release-android-apk.sh 附包，CI 的 post-deploy-smoke
#                   job 必须用它；手动发版后自查不要用。
# 退出码：任一检查失败 → 1；全过 → 0。

set -u

BASE="https://chaoset.com"
VERSION=""
REPO=""
SKIP_APK=0
while [ $# -gt 0 ]; do
  case "$1" in
    --version) VERSION="${2:-}"; shift ;;
    --base) BASE="${2:-$BASE}"; shift ;;
    --repo) REPO="${2:-}"; shift ;;
    --skip-apk) SKIP_APK=1 ;;
    -h|--help) sed -n '2,/^$/p' "$0" | sed -e 's/^# //' -e 's/^#$//'; exit 0 ;;
    *) echo "未知参数：$1（--help 查看用法）"; exit 2 ;;
  esac
  shift
done
if [ -z "$VERSION" ]; then
  echo "必须传 --version <vX.Y.Z>（--help 查看用法）" >&2
  exit 2
fi
case "$VERSION" in
  v[0-9]*.[0-9]*.[0-9]*) ;;
  *) echo "版本号形态不对：${VERSION}（应为 vX.Y.Z）" >&2; exit 2 ;;
esac

GH_REPO_ARG=""
if [ -n "$REPO" ]; then
  GH_REPO_ARG="--repo $REPO"
fi

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

FAIL=0
PASS=0
say() { printf '%s\n' "$1"; }
pass() { say "✓ $1"; PASS=$((PASS + 1)); }
fail() { say "✗ $1"; FAIL=1; }

# 取 JSON 字段（缺失/解析失败输出空串）。
json_field() { # $1=json $2=field
  printf '%s' "$1" | python3 -c 'import json,sys; d=json.load(sys.stdin); v=d.get(sys.argv[1]); print("" if v is None else v)' "$2" 2>/dev/null
}

say "HotSpot 发版后线上冒烟"
say "BASE=$BASE VERSION=$VERSION"
say ""

# 1) version.json：version / commit / project
# 同一类网络抖动：version.json / healthz 也重试 3 次（http_get 定义在后，先就地内联简单重试）
VJ=""
for _try in 1 2 3; do
  VJ="$(curl -sS -m 20 "$BASE/version.json" 2>/dev/null)" && [ -n "$VJ" ] && break
  sleep 2
done
VJ_VERSION="$(json_field "$VJ" version)"
VJ_COMMIT="$(json_field "$VJ" commit)"
VJ_PROJECT="$(json_field "$VJ" project)"
if [ "$VJ_VERSION" = "$VERSION" ]; then
  pass "version.json 版本对齐：$VJ_VERSION"
else
  fail "version.json 版本不符：期望 ${VERSION}，实际 ${VJ_VERSION:-（空/不可达）}"
fi
case "$VJ_COMMIT" in
  [0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f]*) pass "version.json commit：${VJ_COMMIT:0:9}" ;;
  *) fail "version.json commit 缺失/非法：${VJ_COMMIT:-（空）}" ;;
esac
if [ -n "$VJ_PROJECT" ]; then
  pass "version.json project：$VJ_PROJECT"
else
  fail "version.json 缺少 project 字段"
fi

# 2) healthz：code=0 / status=ok / version 与 commit 与前端一致
HZ=""
for _try in 1 2 3; do
  HZ="$(curl -sS -m 20 "$BASE/healthz" 2>/dev/null)" && [ -n "$HZ" ] && break
  sleep 2
done
HZ_DATA="$(printf '%s' "$HZ" | python3 -c 'import json,sys; print(json.dumps(json.load(sys.stdin).get("data", {})))' 2>/dev/null)"
HZ_CODE="$(json_field "$HZ" code)"
HZ_STATUS="$(json_field "$HZ_DATA" status)"
HZ_VERSION="$(json_field "$HZ_DATA" version)"
HZ_COMMIT="$(json_field "$HZ_DATA" commit)"
if [ "$HZ_CODE" = "0" ] && [ "$HZ_STATUS" = "ok" ]; then
  pass "healthz 健康：code=0 status=ok"
else
  fail "healthz 不健康：code=${HZ_CODE:-空} status=${HZ_STATUS:-空}"
fi
if [ "$HZ_VERSION" = "$VERSION" ]; then
  pass "healthz 版本对齐：$HZ_VERSION"
else
  fail "healthz 版本不符：期望 ${VERSION}，实际 ${HZ_VERSION:-（空）}"
fi
if [ -n "$VJ_COMMIT" ] && [ "$HZ_COMMIT" = "$VJ_COMMIT" ]; then
  pass "前后端 commit 一致：${HZ_COMMIT:0:9}"
else
  fail "前后端 commit 不一致：frontend=${VJ_COMMIT:-空} backend=${HZ_COMMIT:-空}"
fi

# 3) 前端入口可达（SPA index.html）
# 本机网络（尤其经系统代理/跨境链路）偶发超时会让单次 curl 返回 000，导致误报；
# 入口检查做 3 次重试（每次 20s 超时），只有连续失败才判失败。
http_get() { # $1=url $2=out_file [$3=attempts]；输出最后一次 http_code
  local url="$1" out="$2" attempts="$3" i code
  attempts="${attempts:-3}"
  for i in $(seq 1 "$attempts"); do
    code="$(curl -sS -m 20 -o "$out" -w '%{http_code}' "$url" 2>/dev/null)" || true
    if [ "$code" = "200" ]; then
      echo "$code"
      return 0
    fi
    if [ "$i" -lt "$attempts" ]; then sleep 2; fi
  done
  echo "${code:-}"
}
INDEX_CODE="$(http_get "$BASE/" "$TMP_DIR/index.html" 3)"
if [ "$INDEX_CODE" = "200" ] && grep -qi '<!doctype html' "$TMP_DIR/index.html"; then
  pass "前端入口 200 + HTML：$BASE/"
else
  fail "前端入口异常：HTTP ${INDEX_CODE:-空}（$BASE/）"
fi

# 4) Release 资产齐全 + 镜像包校验和正确
if ! command -v gh >/dev/null 2>&1; then
  fail "找不到 gh，无法核对 Release 资产"
else
  ASSET_NAMES="$(gh release view "$VERSION" $GH_REPO_ARG --json assets --jq '.assets[].name' 2>/dev/null)"
  if [ -z "$ASSET_NAMES" ]; then
    fail "Release $VERSION 不存在或不可读"
  else
    # APK 由本机在 Release 工作流之后附包，CI 的 post-deploy-smoke 用
    # --skip-apk 跳过；镜像三类资产任何时候都必须齐全。
    REQUIRED_ASSETS=(
      "hotspot-images-$VERSION.tar.gz"
      "hotspot-images-$VERSION.tar.gz.sha256"
      "hotspot-images-$VERSION.ids.txt"
    )
    if [ "$SKIP_APK" = "0" ]; then
      REQUIRED_ASSETS=("hotspot-android-$VERSION.apk" "${REQUIRED_ASSETS[@]}")
    else
      say "· 跳过 APK 资产检查（--skip-apk）"
    fi
    for required in "${REQUIRED_ASSETS[@]}"; do
      if printf '%s\n' "$ASSET_NAMES" | grep -qx "$required"; then
        pass "Release 资产存在：$required"
      else
        fail "Release 缺少资产：$required"
      fi
    done
    # Release 资产走 GitHub 下载，偶发 unexpected EOF（网络抖动）；重试 3 次。
    DOWNLOAD_OK=0
    ATTEMPT=1
    while [ "$ATTEMPT" -le 3 ]; do
      rm -rf "$TMP_DIR/assets"
      mkdir -p "$TMP_DIR/assets"
      if gh release download "$VERSION" $GH_REPO_ARG --pattern "hotspot-images-$VERSION.tar.gz*" -D "$TMP_DIR/assets" >/dev/null 2>&1; then
        DOWNLOAD_OK=1
        break
      fi
      ATTEMPT=$((ATTEMPT + 1))
      sleep 2
    done
    if [ "$DOWNLOAD_OK" = "1" ]; then
      # Linux CI 用 sha256sum，macOS 本机用 shasum -a 256（两者输出格式一致）。
      if command -v sha256sum >/dev/null 2>&1; then
        ACTUAL_SHA="$(sha256sum "$TMP_DIR/assets/hotspot-images-$VERSION.tar.gz" 2>/dev/null | awk '{print $1}')"
      else
        ACTUAL_SHA="$(shasum -a 256 "$TMP_DIR/assets/hotspot-images-$VERSION.tar.gz" 2>/dev/null | awk '{print $1}')"
      fi
      EXPECTED_SHA="$(awk '{print $1}' "$TMP_DIR/assets/hotspot-images-$VERSION.tar.gz.sha256" 2>/dev/null)"
      if [ -n "$ACTUAL_SHA" ] && [ "$ACTUAL_SHA" = "$EXPECTED_SHA" ]; then
        pass "镜像包校验和一致：${ACTUAL_SHA:0:12}…"
      else
        fail "镜像包校验和不一致：actual=${ACTUAL_SHA:-空} expected=${EXPECTED_SHA:-空}"
      fi
    else
      fail "镜像包下载失败（Release 资产不可读，已重试 3 次）"
    fi
  fi
fi

say ""
if [ "$FAIL" -eq 0 ]; then
  say "结果: PASS（通过 ${PASS}）"
  exit 0
else
  say "结果: FAIL（通过 ${PASS}，存在失败项）"
  exit 1
fi
