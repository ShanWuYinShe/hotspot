#!/bin/sh
# ── 迁移纪律校验（CI backend job 调用，本地门禁同款）────────────────
# 1) 序号唯一（历史遗留 009 重号白名单；新增重复即失败）；
# 2) 执行架构约束：禁止 CREATE INDEX CONCURRENTLY（pgx 简单协议整文件单消息
#    执行会落隐式事务，25001 直接让启动失败）；
# 3) expand-contract 兼容性：新迁移（序号 >= 058）禁止破坏性 DDL——旧镜像回滚
#    后仍要能跑在新 schema 上，DROP/RENAME/SET NOT NULL/TYPE 变更跨版本再清。
set -eu

cd "$(dirname "$0")/../backend-go/migrations"

# ── 1) 序号唯一 ─────────────────────────────────────────────────
dupes=$(ls -1 *.sql | grep -oE '^[0-9]{3}' | sort | uniq -d)
# 历史遗留：009 重号（009_admin_audit_app_config / 009_admin_moderation_tags_announcements），
# 已应用顺序由字典序确定，重命名会改变全新环境的执行顺序，故只拦新增。
dupes=$(printf '%s\n' "$dupes" | grep -v '^009$' || true)
if [ -n "$dupes" ]; then
  echo "!! 迁移序号重复：${dupes}（新增迁移不得与既有序号重复）"
  exit 1
fi

# 去掉 SQL 行注释后再匹配，避免把说明文字里的关键字当成 DDL。
strip_comments() {
  sed -e 's/--.*$//' "$1"
}

# ── 2) 禁止 CREATE INDEX CONCURRENTLY（架构约束，全量历史文件适用）──
for f in *.sql; do
  if strip_comments "$f" | grep -qiE 'create[[:space:]]+(unique[[:space:]]+)?index[[:space:]]+concurrently'; then
    echo "!! 迁移 $f 使用 CREATE INDEX CONCURRENTLY：当前迁移架构（整文件单消息/隐式事务）会报 25001。"
    echo "   移除 CONCURRENTLY；若需避免建索引停写，把建索引拆成单语句迁移文件（见 docs/backend-audit.md 与迁移 skill）。"
    exit 1
  fi
done

# ── 3) 新迁移的 expand-contract 兼容性（回滚 = 旧镜像 + 保留新 schema）──
# 本守卫落地时最新序号为 057；>=058 视为「新增迁移」受约束，历史文件不回改。
for f in *.sql; do
  seq=${f%%_*}
  # POSIX sh 不保证前导 0 的算术解释；去掉前导 0 后按十进制比较。
  seq=$(printf '%s' "$seq" | sed 's/^0*//')
  [ -n "$seq" ] || seq=0
  [ "$seq" -ge 58 ] || continue
  if strip_comments "$f" | grep -qiE 'drop[[:space:]]+(column|table|constraint)|rename[[:space:]]+column|set[[:space:]]+not[[:space:]]+null|alter[[:space:]]+column[[:space:]]+[a-z_]+[[:space:]]+type'; then
    echo "!! 迁移 $f 含破坏性/不兼容 DDL（expand-contract 纪律）："
    echo "   禁止 DROP COLUMN/TABLE/CONSTRAINT、RENAME COLUMN、SET NOT NULL、ALTER COLUMN TYPE。"
    echo "   做法：先加新列/新表并双写兼容，跨版本再清理旧结构（见 skill hotspot-backend-migrations）。"
    exit 1
  fi
done

echo "迁移序号与兼容性纪律校验 OK"
