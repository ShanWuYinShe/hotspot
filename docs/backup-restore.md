# Backup & Restore Runbook

This is an **operational runbook** — manual procedures to back up and restore a
HotSpot deployment. It does not deploy an automated sidecar; the expectation is
that the operator (or an external scheduler) runs these on a schedule.

The two pieces of state that matter:

| State             | Where it lives                          | How to back up                          |
|-------------------|-----------------------------------------|-----------------------------------------|
| Application data  | `postgres_data` volume (PostgreSQL)     | `pg_dump`                               |
| Uploaded media    | `seaweedfs_data` volume (SeaweedFS S3)  | rsync / volume snapshot of `/data`      |
| Cache / sessions  | `redis_data` volume (Redis)             | **Not backed up** — ephemeral, rebuilds |

Everything else (container images, configs, migrations) is reproducible from
source and the `.env` file, which the operator must keep in a **separate**,
access-controlled secret store.

---

## 1. PostgreSQL (manual one-liner)

Run from the host that runs the compose stack. Output is a **custom-format**
dump (`pg_dump -Fc`, 自带压缩、支持 `pg_restore` 选择性恢复) that can be
restored into a fresh `postgres:18` container (compose pins
`POSTGRES_VERSION=18.4`; keep major versions aligned).

```bash
# Set these to match your deployment.
export COMPOSE_FILE="docker-compose.yml"   # add -f docker-compose.dev.yml if dev
export BACKUP_DIR="./backups"
export DB_CONTAINER="hotspot-postgres-1"   # docker compose ps to confirm
export PGUSER="postgres"
export PGDB="hotspot"

mkdir -p "$BACKUP_DIR"
STAMP=$(date -u +%Y%m%dT%H%M%SZ)
docker exec "$DB_CONTAINER" \
  pg_dump -U "$PGUSER" -d "$PGDB" -Fc --no-owner \
  > "$BACKUP_DIR/hotspot-db-$STAMP.dump"
```

Flags explained:

- `-Fc` — custom format: internally compressed (典型 5–10x), and unlike
  plain SQL it can be restored selectively (`pg_restore -t <table>`) and
  verified with `pg_restore -l`.
- `--no-owner` — restore works regardless of the local role name.

Verify the dump is non-empty and reloadable before trusting it:

```bash
test -s "$BACKUP_DIR/hotspot-db-$STAMP.dump" || echo "EMPTY DUMP"
docker exec -i "$DB_CONTAINER" pg_restore -l < "$BACKUP_DIR/hotspot-db-$STAMP.dump" | head -5
```

`pg_restore -l` 只读归档目录（TOC）——损坏的 dump 在这里直接报错，而不会
等到真正还原时才暴露。

---

## 2. 定时备份（服务器 cron，以仓库脚本为准）

定时备份的唯一来源是 `scripts/backup-postgres.sh`，用法与默认值（`-Fc`
custom format 输出 `.dump`、dump 后 `pg_restore -l` 完整性自检、保留最近
14 份、备份目录为仓库根 `backups/`，已 gitignore；`ALERT_WEBHOOK` 失败告警、
`OFFSITE_CMD` 可选异地上传）见脚本头注释。不要另写备份命令或 cron 片段——
两处维护必然漂移（本节早前引用过并不存在的 `scripts/backup-db.sh`，即此类
漂移的实例）。

服务器侧示例（以脚本头注释为准）：

```cron
30 3 * * * /home/workuser/hotspot/scripts/backup-postgres.sh >> /home/workuser/hotspot/backups/backup.log 2>&1
0 4 * * * /home/workuser/hotspot/scripts/backup-seaweedfs.sh >> /home/workuser/hotspot/backups/backup.log 2>&1
*/10 * * * * /home/workuser/hotspot/scripts/health-check.sh >> /home/workuser/hotspot/backups/health-check.log 2>&1
```

- `scripts/backup-seaweedfs.sh`：媒体卷定时打包（第 4 节的手动方案即由此脚本固化）。
- `scripts/health-check.sh`：健康巡检（后端 health 端点、核心容器运行状态、
  磁盘水位、**备份新鲜度（>25h 即告警）**、TLS 证书剩余有效期），异常时非零
  退出并留日志；**这是当前唯一的线上巡检通道**，新机器重建时必须重配 crontab。
- 另有部署兜底：`scripts/deploy-production.sh` 在滚动更新前自动执行一次
  pg_dump（`-Fc`，服务器 `backups/` + 本机 `.workwork/` 各一份，完整性校验
  失败即中止部署）；版本校验失败时自动回滚到上一版本并经 `ALERT_WEBHOOK`
  告警（仅回滚镜像，数据面见第 3 节警告）。

---

## 3. Restore procedure

Restoring replaces **all** application data. Run against a stopped backend to
avoid writes racing the restore.

```bash
# 1. Stop the API (keep postgres up so we can pipe into it).
docker compose stop backend

# 2. Confirm the target DB name + container.
docker compose ps postgres

# 3. Restore from a dump. --clean drops conflicting objects first.
docker exec -i hotspot-postgres-1 \
  pg_restore -U postgres -d hotspot --no-owner --clean --if-exists \
  < ./backups/hotspot_20260101_020000.dump

# 4. Sanity check row counts, then bring the API back.
docker compose start backend
docker compose ps   # wait for backend to report healthy
```

> **历史格式**：2026-09 之前的定时备份是 gzip plain SQL（`.sql.gz`）。还原
> 旧文件用：
> `zcat ./backups/hotspot-db-*.sql.gz | docker exec -i hotspot-postgres-1 psql -U postgres -d hotspot`

If restoring into a **brand-new** postgres container (disaster recovery):

```bash
# Stand up a fresh stack first so the volume exists, then restore.
docker compose up -d postgres
# Wait for healthy, then run step 3 above.
```

---

## 4. SeaweedFS media volume

SeaweedFS stores uploaded images/videos as files under `/data` in the
`seaweedfs_data` named volume. Back them up by snapshotting that volume — do
**not** rely on S3-level tooling, because the volume also holds SeaweedFS's
internal metadata that is not exposed via the S3 API.

### Option A: 定时脚本（推荐，`scripts/backup-seaweedfs.sh`）

定时打包见第 2 节 crontab；脚本经一次性只读挂载容器执行 tar，不写入业务卷。
以下为手动方案的等价说明。

### Option B: rsync via a temporary mount

```bash
# Copy the entire volume out to the host filesystem.
docker run --rm \
  -v hotspot_seaweedfs_data:/data:ro \
  -v "$(pwd)/backups/seaweedfs-$(date -u +%Y%m%d):/backup" \
  alpine:3.24.1 \
  sh -c "cp -a /data/. /backup/"
```

To restore, reverse the mount and copy back, then restart seaweedfs:

```bash
docker compose stop seaweedfs
docker run --rm \
  -v hotspot_seaweedfs_data:/data \
  -v "$(pwd)/backups/seaweedfs-20260101:/backup:ro" \
  alpine:3.24.1 \
  sh -c "cp -a /backup/. /data/"
docker compose start seaweedfs
```

### Option C: volume snapshot (Docker / cloud)

If the Docker volume is backed by a block device or cloud volume (zfs, EBS,
etc.), use the host's snapshot mechanism — it is faster and more consistent
than file-level rsync:

```bash
# Example: zfs snapshot
zfs snapshot tank/hotspot_seaweedfs_data@backup-$(date -u +%Y%m%d)
```

Document whatever mechanism your storage driver uses here.

### Restore verification

After restoring, confirm the S3 bucket and object count look right:

```bash
# Requires aws-cli or mc (MinIO client) pointed at SeaweedFS.
mc ls local/hotspot/ | head
mc find local/hotspot --count | tail -1
```

Then log in to the app and open a few user-uploaded images to confirm they
resolve through the nginx `/s3/` proxy.

---

## 5. What is **not** backed up (and why)

- **Redis (`redis_data`)** — holds rate-limit counters, WebSocket fanout state,
  and short-lived session data. All of it is safe to lose; the system
  degrades to single-instance mode and rebuilds state on the next request.
- **`.env` secrets** — the operator keeps these out of band (e.g. a secrets
  manager). Never commit `.env` to git or to the backup bucket in plaintext.
- **Container images** — reproducible from source via
  `docker compose --profile prod build`.

---

## 6. Restore drill

Run a restore drill **at least once per quarter** against a throwaway stack:

1. `docker compose -f docker-compose.yml -p hotspot-drill up -d postgres seaweedfs`
2. Restore the latest DB dump (section 3) and media volume (section 4).
3. Start the backend and confirm the API boots and serves a known-good page.
4. Tear down: `docker compose -p hotspot-drill down -v`.

If the drill fails, the backup is useless — fix the procedure before you need
it for real.
