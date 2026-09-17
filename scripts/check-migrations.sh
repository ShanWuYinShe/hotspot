#!/bin/sh
# ── 迁移序号校验（CI backend job 调用）───────────────────────────────
# 校验 migrations/ 下：序号唯一、无重复（历史遗留的 009 重号已由该脚本
# 登记为白名单；新增重复即失败）。顺序正确性当前依赖字典序，编号可靠化
# 见成熟度审计遗留项。
set -eu

cd "$(dirname "$0")/../backend-go/migrations"

dupes=$(ls -1 *.sql | grep -oE '^[0-9]{3}' | sort | uniq -d)
# 历史遗留：009 重号（009_admin_audit_app_config / 009_admin_moderation_tags_announcements），
# 已应用顺序由字典序确定，重命名会改变全新环境的执行顺序，故只拦新增。
dupes=$(printf '%s\n' "$dupes" | grep -v '^009$' || true)
if [ -n "$dupes" ]; then
  echo "!! 迁移序号重复：$dupes（新增迁移不得与既有序号重复）"
  exit 1
fi
echo "迁移序号校验 OK"
