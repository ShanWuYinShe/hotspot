#!/usr/bin/env bash
# ── 接待机器人账号口令轮换（本地库）────────────────────────────────
# 用法（仓库根目录，本地 dev compose 在运行）：
#   bash reception-bot/scripts/rotate-credentials.sh
# 环境变量：
#   POSTGRES_CONTAINER  默认 hotspot-postgres-1
#   POSTGRES_DB/USER    默认 hotspot / postgres
#   BACKEND_URL         默认 http://localhost:8081（轮换后验证登录）
#   BOT_STATE_ENV       默认 reception-bot/state/reception-bot.env；在 worktree 里执行时
#                       自动回退主工作树同名路径（state/ 是 gitignored 运行时目录，
#                       不会随 worktree 复制）
#
# 做什么：为归档里的每个口令生成新随机值 → bcrypt(cost 12，与库中现有 hash 同参)
# → 更新 auth_accounts.password_hash → **吊销这些账号全部会话**（泄漏场景必须做，
# 否则旧 token 仍可用）→ 清理脚本缓存 token → 原子替换 0600 归档 → 逐个实测登录。
#
# 一致性保护：口令存在「数据库 hash」与「归档文件」两处，跨存储无法真正原子。脚本
# 在改库前快照旧 hash，只有在归档尚未发布时失败才回滚数据库——避免出现「库已改、
# 归档未改」把机器人锁死（2026-10-02 首次实现即踩到：awk -v 无法承载含换行的值）。
#
# 纪律：全程不回显任何口令值；归档保持 0600 且位于 gitignored 的 state/ 下。
set -euo pipefail
cd "$(dirname "$0")/../.."

CONTAINER="${POSTGRES_CONTAINER:-hotspot-postgres-1}"
DB="${POSTGRES_DB:-hotspot}"
DBUSER="${POSTGRES_USER:-postgres}"
BACKEND="${BACKEND_URL:-http://localhost:8081}"
ENVFILE="${BOT_STATE_ENV:-reception-bot/state/reception-bot.env}"
if [ ! -f "$ENVFILE" ] && [ -z "${BOT_STATE_ENV:-}" ]; then
  main_root="$(cd "$(git rev-parse --git-common-dir)/.." && pwd)"
  [ -f "$main_root/reception-bot/state/reception-bot.env" ] && ENVFILE="$main_root/reception-bot/state/reception-bot.env"
fi
STATE_DIR="$(dirname "$ENVFILE")"
WORK=".workwork/rotate-bot-$$"
HELPER="backend-go/tmp-pwhash"
mkdir -p "$WORK"; chmod 700 "$WORK"
# Go 构建缓存默认写 ~/Library/Caches/go-build：本机已由 DSH sandbox-extra-roots 插件
# 放行（2026-10-14 实测可写），直接构建即可；未配插件的机器如报 operation not
# permitted，可临时 export GOCACHE=<工作区内目录> 兜底（bcrypt 生成步骤曾无声失败的坑）。

cleanup() { rm -rf "$WORK" "$HELPER"; }
restore_db() {
  # 仅在「库已改、归档未发布」时回滚，避免把已发布的轮换又倒回去
  [ -f "$WORK/db-updated" ] && [ ! -f "$WORK/archive-published" ] || return 0
  echo "!! 归档尚未发布，回滚数据库到轮换前的 hash…" >&2
  docker exec -i "$CONTAINER" psql -q -v ON_ERROR_STOP=1 -U "$DBUSER" -d "$DB" < "$WORK/old-hashes.sql" >&2 || \
    echo "!! 回滚失败：库中口令已变但归档未更新，需人工执行恢复" >&2
}
on_exit() { local st=$?; restore_db || true; cleanup; exit "$st"; }
trap on_exit EXIT

[ -f "$ENVFILE" ] || { echo "!! 找不到口令归档 $ENVFILE"; exit 1; }
admin_id=$(sed -n 's/^WELCOME_ADMIN_LOGIN_ID=//p' "$ENVFILE")
[ -n "$admin_id" ] || { echo "!! 归档缺少 WELCOME_ADMIN_LOGIN_ID"; exit 1; }

accounts=""
for key in $(grep -oE '^WELCOME_PASSWORD_[A-Z]+=' "$ENVFILE" | sed 's/=$//'); do
  accounts="$accounts $(printf '%s' "$key" | sed 's/^WELCOME_PASSWORD_//' | tr 'A-Z' 'a-z')"
done
[ -n "$accounts" ] || { echo "!! 归档里没有 WELCOME_PASSWORD_<人格> 条目"; exit 1; }
echo "==> 待轮换账号:${accounts}（另含管理员 ${admin_id}）"

# 登录标识列表（供 SQL IN 使用）
in_list=""
for acc in $accounts "$admin_id"; do in_list="$in_list'$acc',"; done
in_list="${in_list%,}"

# 1) 新口令（hex：无 shell 元字符，set -a 注入安全）
: > "$WORK/pw.txt"; chmod 600 "$WORK/pw.txt"
i=0
for _ in $accounts "$admin_id"; do
  openssl rand -hex 24 >> "$WORK/pw.txt"
  i=$((i+1))
done
echo "==> 已生成 $i 组随机口令（48 hex 字符）"

# 2) 旧 hash 快照（回滚用）
docker exec "$CONTAINER" psql -tA -U "$DBUSER" -d "$DB" \
  -c "SELECT 'UPDATE auth_accounts SET password_hash=''' || password_hash || ''', updated_at=now() WHERE login_id=''' || login_id || ''';' FROM auth_accounts WHERE login_id IN ($in_list);" \
  > "$WORK/old-hashes.sql"
old_count=$(grep -c 'UPDATE' "$WORK/old-hashes.sql" || true)
[ "$old_count" -eq $i ] || { echo "!! 账号数与库中记录不一致（库 $old_count / 归档 $i），中止"; exit 1; }

# 3) bcrypt（借 backend-go 模块的 x/crypto/bcrypt，不引入新工具链）
mkdir -p "$HELPER"
cat > "$HELPER/main.go" <<'GO'
package main

import (
	"bufio"
	"fmt"
	"os"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	// verify 子命令：verify <bcrypt hash> <明文> → MATCH / MISMATCH（供脚本自检库中 hash）
	if len(os.Args) >= 4 && os.Args[1] == "verify" {
		if err := bcrypt.CompareHashAndPassword([]byte(os.Args[2]), []byte(os.Args[3])); err != nil {
			fmt.Println("MISMATCH")
			os.Exit(1)
		}
		fmt.Println("MATCH")
		return
	}
	sc := bufio.NewScanner(os.Stdin)
	for sc.Scan() {
		line := sc.Text()
		if line == "" {
			continue
		}
		h, err := bcrypt.GenerateFromPassword([]byte(line), 12)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println(string(h))
	}
}
GO
(cd backend-go && go run ./tmp-pwhash < "../$WORK/pw.txt") > "$WORK/hashes.txt"
[ "$(grep -c . "$WORK/hashes.txt")" -eq $i ] || { echo "!! bcrypt 生成数量不符"; exit 1; }
echo "==> bcrypt 完成（$i 条）"

# 4) 组装「新口令映射」（key=value 换行分隔，供归档替换）与更新 SQL
: > "$WORK/new-kv.txt"; : > "$WORK/update.sql"; chmod 600 "$WORK/new-kv.txt"
n=1
for acc in $accounts "$admin_id"; do
  pw=$(sed -n "${n}p" "$WORK/pw.txt")
  h=$(sed -n "${n}p" "$WORK/hashes.txt")
  if [ "$acc" = "$admin_id" ]; then key="WELCOME_ADMIN_PASSWORD"; else key="WELCOME_PASSWORD_$(printf '%s' "$acc" | tr 'a-z' 'A-Z')"; fi
  printf '%s=%s\n' "$key" "$pw" >> "$WORK/new-kv.txt"
  printf "UPDATE auth_accounts SET password_hash='%s', updated_at=now() WHERE login_id='%s';\n" "$h" "$acc" >> "$WORK/update.sql"
  n=$((n+1))
done
printf "DELETE FROM auth_sessions WHERE user_id IN (SELECT user_id FROM auth_accounts WHERE login_id IN ($in_list));\n" >> "$WORK/update.sql"

# 5) 先算好新归档内容（不落位），再改库——缩短不一致窗口
umask 077
while IFS= read -r line; do
  case "$line" in
    *=*)
      key="${line%%=*}"
      val=$(sed -n "s|^${key}=||p" "$WORK/new-kv.txt" | head -1)
      if [ -n "$val" ]; then printf '%s=%s\n' "$key" "$val"; else printf '%s\n' "$line"; fi
      ;;
    *) printf '%s\n' "$line" ;;
  esac
done < "$ENVFILE" > "$WORK/new.env"
chmod 600 "$WORK/new.env"
[ "$(grep -c . "$WORK/new.env")" -eq "$(grep -c . "$ENVFILE")" ] || { echo "!! 新归档行数与原文不符，中止（未动库）"; exit 1; }

# 6) 改库 → 发布归档（两步之间失败由 restore_db 回滚）
docker exec -i "$CONTAINER" psql -q -v ON_ERROR_STOP=1 -U "$DBUSER" -d "$DB" < "$WORK/update.sql"
touch "$WORK/db-updated"
# 自检：库中第一个账号的 hash 必须能验证刚生成的口令。缺了这步，SQL 组装错误
# （如占位符写错导致 WHERE 不匹配、0 行更新却零报错）会被当成成功——
# 2026-10-02 实测踩过：脚本报「已更新」，实际 0 行受影响。
first_acc=$(printf '%s\n' $accounts | head -1)
first_pw=$(sed -n 1p "$WORK/pw.txt")
db_hash=$(docker exec "$CONTAINER" psql -tA -U "$DBUSER" -d "$DB" -c "SELECT password_hash FROM auth_accounts WHERE login_id='$first_acc';")
if ! (cd backend-go && go run ./tmp-pwhash verify "$db_hash" "$first_pw") | grep -q MATCH; then
  echo "!! 库中 hash 与刚生成的口令不匹配（UPDATE 未生效），中止且不发布归档"; exit 1
fi
echo "==> 数据库已更新并吊销旧会话（自检通过）"
mv -f "$WORK/new.env" "$ENVFILE"
touch "$WORK/archive-published"
echo "==> 归档已更新（$(stat -f '%Sp' "$ENVFILE" 2>/dev/null || stat -c '%A' "$ENVFILE")）"

# 7) 清理脚本缓存 token（旧会话已吊销，留着只会让脚本反复失败）
export STATE_DIR
python3 - <<'PY'
import glob, json, os
for f in glob.glob(os.path.join(os.environ['STATE_DIR'], 'state.*.json')):
    d = json.load(open(f))
    if 'auth' in d:
        d.pop('auth')
        json.dump(d, open(f, 'w'), ensure_ascii=False, indent=2)
PY
rm -f "$STATE_DIR"/ws-auth.*.json
echo "==> 已清理缓存 token"

# 8) 验证新口令登录（仅显示返回码与 user_id）
echo "==> 验证登录"
n=1
for acc in $accounts "$admin_id"; do
  pw=$(sed -n "${n}p" "$WORK/pw.txt")
  body=$(python3 -c 'import json,sys;print(json.dumps({"login_id":sys.argv[1],"password":sys.argv[2],"client_type":"app","device_id":"00000000-0000-4000-8000-0000000000ff"}))' "$acc" "$pw")
  curl -s -X POST "$BACKEND/public/v1/auth/signin" -H 'Content-Type: application/json' -d "$body" \
    | python3 -c 'import json,sys
d=json.load(sys.stdin)
print("    %-12s code=%s user_id=%s" % (sys.argv[1], d.get("code"), (d.get("data") or {}).get("user_id","")))' "$acc"
  n=$((n+1))
done
echo "✅ 轮换完成（命中过 gitleaks 时记得同步登记指纹并写明轮换依据）"