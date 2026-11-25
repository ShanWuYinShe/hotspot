#!/usr/bin/env bash
# 应急兜底：本机构建并上传 Android 签名 APK 到对应 GitHub Release。
# 正常发版由 GitHub Release workflow 的 android-apk job 构建并附包，不要在本地跑本脚本。
# 仅在 GitHub Actions 故障时使用；用完手动触发完整冒烟：
#   gh workflow run release-verify.yml -f version=vX.Y.Z
#
# 用法：scripts/release-android-apk.sh vX.Y.Z <versionCode>
#   例：scripts/release-android-apk.sh v1.14.0 5
#
# 前置条件：
#   - tag vX.Y.Z 已推送到 origin/main（Release 工作流跑完后 Release 才存在）；
#   - 本机有签名材料：~/.android/hotspot-release.jks + 钥匙串口令（service=hotspot-android-release）；
#   - 已登录 gh（有 repo 权限）。
# 流程：校验 tag/Release → assembleRelease（签名 + 版本号注入）→ apksigner/aapt2 校验 → gh release upload。
set -euo pipefail

usage() {
  echo "用法: $0 vX.Y.Z <versionCode>" >&2
  echo "示例: $0 v1.14.0 5" >&2
  exit 2
}

[[ $# -eq 2 ]] || usage
VERSION="$1"
CODE="$2"
[[ "${VERSION}" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]] || { echo "版本号必须是 vX.Y.Z: ${VERSION}" >&2; exit 1; }
[[ "${CODE}" =~ ^[0-9]+$ ]] || { echo "versionCode 必须是整数: ${CODE}" >&2; exit 1; }

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT}"

echo "==> 校验 tag 与 Release（${VERSION}）"
git fetch --tags --quiet
git rev-parse -q --verify "refs/tags/${VERSION}" >/dev/null || { echo "tag ${VERSION} 不存在：先打 tag 并在 Release 工作流跑完后再上传 APK" >&2; exit 1; }
git merge-base --is-ancestor "refs/tags/${VERSION}^{commit}" origin/main || { echo "tag ${VERSION} 不在 origin/main 上" >&2; exit 1; }
gh release view "${VERSION}" >/dev/null 2>&1 || { echo "Release ${VERSION} 尚不存在：等 Release 工作流完成后再执行" >&2; exit 1; }

KEYSTORE="${HOTSPOT_KEYSTORE_FILE:-${HOME}/.android/hotspot-release.jks}"
[[ -f "${KEYSTORE}" ]] || { echo "缺少 keystore: ${KEYSTORE}（无法签名，禁止用 debug 签名冒充发布包）" >&2; exit 1; }
PASS="$(security find-generic-password -a "${USER}" -s hotspot-android-release -w 2>/dev/null || true)"
[[ -n "${PASS}" ]] || { echo "钥匙串缺少 hotspot-android-release 口令（见 docs/release-runbook.md）" >&2; exit 1; }

export JAVA_HOME="${JAVA_HOME:-/Applications/Android Studio.app/Contents/jbr/Contents/Home}"
export ANDROID_HOME="${ANDROID_HOME:-${HOME}/Library/Android/sdk}"
export PATH="${JAVA_HOME}/bin:${PATH}"

echo "==> assembleRelease（versionName=${VERSION} versionCode=${CODE}）"
cd androidapp
HOTSPOT_KEYSTORE_FILE="${KEYSTORE}" \
HOTSPOT_KEYSTORE_PASSWORD="${PASS}" \
HOTSPOT_KEY_ALIAS=hotspot \
HOTSPOT_KEY_PASSWORD="${PASS}" \
HOTSPOT_VERSION_NAME="${VERSION}" \
HOTSPOT_VERSION_CODE="${CODE}" \
./gradlew --no-daemon assembleRelease

APK="$(pwd)/app/build/outputs/apk/release/app-release.apk"
[[ -f "${APK}" ]] || { echo "未生成 APK: ${APK}" >&2; exit 1; }

APKSIGNER="$(ls -d "${ANDROID_HOME}"/build-tools/*/apksigner 2>/dev/null | sort -V | tail -1)"
AAPT2="$(ls -d "${ANDROID_HOME}"/build-tools/*/aapt2 2>/dev/null | sort -V | tail -1)"
[[ -x "${APKSIGNER}" ]] || { echo "找不到 apksigner（ANDROID_HOME=${ANDROID_HOME}）" >&2; exit 1; }
[[ -x "${AAPT2}" ]] || { echo "找不到 aapt2（ANDROID_HOME=${ANDROID_HOME}）" >&2; exit 1; }

echo "==> 校验签名与版本"
"${APKSIGNER}" verify --print-certs "${APK}" | sed -n "1,2p"
BADGING="$("${AAPT2}" dump badging "${APK}" | sed -n "1p")"
echo "${BADGING}"
echo "${BADGING}" | grep -q "versionName='${VERSION#v}'" || { echo "APK versionName 与 ${VERSION} 不一致" >&2; exit 1; }
echo "${BADGING}" | grep -q "versionCode='${CODE}'" || { echo "APK versionCode 与 ${CODE} 不一致" >&2; exit 1; }

echo "==> 上传到 Release ${VERSION}"
cd "${ROOT}"
ARTIFACT="${ROOT}/.workwork/hotspot-android-${VERSION}.apk"
mkdir -p "${ROOT}/.workwork"
cp "${APK}" "${ARTIFACT}"
gh release upload "${VERSION}" "${ARTIFACT}#hotspot-android-${VERSION}.apk" --clobber
rm -f "${ARTIFACT}"
gh release view "${VERSION}" --json assets --jq '.assets[] | "\(.name)  \(.size) bytes"'
echo "✅ APK 已发布: hotspot-android-${VERSION}.apk"