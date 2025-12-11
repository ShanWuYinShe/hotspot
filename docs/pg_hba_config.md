# PostgreSQL 本地认证配置指南

## 概述

为了让 `hotspot_user` 能够无密码访问本地 PostgreSQL 数据库，需要配置 PostgreSQL 的客户端认证文件 `pg_hba.conf`。本文档说明如何进行配置。

## pg_hba.conf 文件位置

根据您的操作系统和 PostgreSQL 安装方式，`pg_hba.conf` 文件位置可能不同：

### Linux

| 发行版 | 默认位置 |
|--------|----------|
| Ubuntu/Debian | `/etc/postgresql/{version}/main/pg_hba.conf` |
| CentOS/RHEL/Fedora | `/var/lib/pgsql/{version}/data/pg_hba.conf` |
| Arch Linux | `/var/lib/postgres/data/pg_hba.conf` |

### macOS

| 安装方式 | 位置 |
|----------|------|
| Homebrew | `/opt/homebrew/var/postgresql@{version}/pg_hba.conf` 或 `/usr/local/var/postgresql@{version}/pg_hba.conf` |
| Postgres.app | `~/Library/Application Support/Postgres/var-{version}/pg_hba.conf` |

### 查找 pg_hba.conf

如果不确定位置，可以使用以下命令查找：

```bash
# 方法 1：通过 psql 查询
sudo -u postgres psql -c "SHOW hba_file;"

# 方法 2：通过 find 命令
sudo find /etc /var -name pg_hba.conf 2>/dev/null
```

## 推荐配置

在 `pg_hba.conf` 文件中添加以下配置行（添加到文件开头，在其他规则之前）：

```conf
# TYPE  DATABASE        USER              ADDRESS                 METHOD

# HotSpot 开发环境配置
# 允许 hotspot_user 通过 Unix socket 本地连接（peer 认证）
local   all             hotspot_user                              peer

# 允许 hotspot_user 通过本地 TCP 连接（trust 认证，仅本地回环）
host    all             hotspot_user      127.0.0.1/32            trust
host    all             hotspot_user      ::1/128                 trust
```

### 配置说明

| 字段 | 说明 |
|------|------|
| **TYPE** | 连接类型 (`local` = Unix socket, `host` = TCP/IP) |
| **DATABASE** | 允许访问的数据库 (`all` = 所有数据库) |
| **USER** | 允许连接的用户 (`hotspot_user`) |
| **ADDRESS** | 允许连接的地址（仅对 `host` 类型有效） |
| **METHOD** | 认证方法 (`peer` = 操作系统用户认证, `trust` = 无需密码) |

### 认证方法详解

1. **peer**（推荐用于 Unix socket）
   - 检查操作系统用户名与数据库用户名是否匹配
   - 如果您的 Linux/macOS 用户名与 `hotspot_user` 不同，需要额外配置
   - 最安全的本地认证方式

2. **trust**（用于本地 TCP）
   - 完全信任本地连接，不需要密码
   - 仅适用于本地开发环境
   - 确保只允许 `127.0.0.1` 和 `::1`（本地回环地址）

3. **md5/scram-sha-256**（生产环境推荐）
   - 需要密码认证
   - 如果需要在生产环境使用，应改为这些方法

## 配置步骤

### 1. 备份原配置文件

```bash
sudo cp /etc/postgresql/15/main/pg_hba.conf /etc/postgresql/15/main/pg_hba.conf.backup
```

### 2. 编辑 pg_hba.conf

```bash
sudo nano /etc/postgresql/15/main/pg_hba.conf
```

或使用 `vim`：

```bash
sudo vim /etc/postgresql/15/main/pg_hba.conf
```

### 3. 添加配置行

将推荐配置添加到文件开头（在 `# TYPE DATABASE USER ADDRESS METHOD` 注释行之后）：

```conf
# HotSpot 开发环境配置
local   all             hotspot_user                              peer
host    all             hotspot_user      127.0.0.1/32            trust
host    all             hotspot_user      ::1/128                 trust
```

**重要**：这些规则应该添加在其他规则之前，因为 PostgreSQL 按照从上到下的顺序匹配规则。

### 4. 保存并退出

- **nano**: 按 `Ctrl+O` 保存，`Ctrl+X` 退出
- **vim**: 按 `Esc`，输入 `:wq` 保存退出

### 5. 重新加载 PostgreSQL 配置

```bash
# 方法 1：使用 systemctl（推荐）
sudo systemctl reload postgresql

# 方法 2：使用 pg_ctl
sudo -u postgres pg_ctl reload

# 方法 3：通过 SQL
sudo -u postgres psql -c "SELECT pg_reload_conf();"
```

### 6. 验证配置

测试连接是否成功：

```bash
# 通过 Unix socket 连接
psql -U hotspot_user -d hotspot_db

# 通过 TCP 连接
psql -U hotspot_user -h localhost -d hotspot_db
```

如果连接成功且无需密码，说明配置正确。

## 操作系统用户名与数据库用户名不匹配的情况

如果您的 Linux/macOS 用户名不是 `hotspot_user`，peer 认证会失败。有两种解决方案：

### 方案 1：创建映射（推荐）

在 `pg_ident.conf` 中创建用户映射：

```bash
sudo nano /etc/postgresql/15/main/pg_ident.conf
```

添加以下行：

```conf
# MAPNAME       SYSTEM-USERNAME         PG-USERNAME
hotspot_map     your_linux_username     hotspot_user
```

然后修改 `pg_hba.conf` 中的 peer 规则：

```conf
local   all             hotspot_user                              peer map=hotspot_map
```

### 方案 2：使用 trust 代替 peer

将 `peer` 改为 `trust`：

```conf
local   all             hotspot_user                              trust
```

**注意**：此方法安全性较低，仅适用于开发环境。

## 常见问题排查

### 问题 1：连接被拒绝

```
psql: error: connection to server on socket "/var/run/postgresql/.s.PGSQL.5432" failed: FATAL: Peer authentication failed for user "hotspot_user"
```

**原因**：操作系统用户名与数据库用户名不匹配。

**解决**：使用方案 1 或方案 2（见上文）。

### 问题 2：找不到 socket 文件

```
psql: error: connection to server on socket "/var/run/postgresql/.s.PGSQL.5432" failed: No such file or directory
```

**原因**：PostgreSQL 未运行或 socket 文件位置不同。

**解决**：
```bash
# 启动 PostgreSQL
sudo systemctl start postgresql

# 检查 PostgreSQL 状态
sudo systemctl status postgresql

# 查看实际 socket 位置
sudo -u postgres psql -c "SHOW unix_socket_directories;"
```

### 问题 3：修改后不生效

**原因**：配置未重新加载。

**解决**：
```bash
sudo systemctl reload postgresql
```

### 问题 4：规则顺序错误

**原因**：`pg_hba.conf` 中其他规则先匹配。

**解决**：确保 `hotspot_user` 的规则在其他规则之前。

## 生产环境注意事项

**⚠️ 警告**：以上配置仅适用于开发和测试环境。

在生产环境中，应该：

1. **禁用 trust 认证**：改用 `md5` 或 `scram-sha-256`
2. **设置强密码**：为 `hotspot_user` 设置复杂密码
3. **限制网络访问**：不要允许远程连接，或使用严格的防火墙规则
4. **使用 SSL/TLS**：加密数据库连接
5. **最小权限原则**：根据实际需求限制数据库权限

示例生产环境配置：

```conf
# 生产环境配置（示例）
local   hotspot_db      hotspot_user                              scram-sha-256
host    hotspot_db      hotspot_user      127.0.0.1/32            scram-sha-256
host    hotspot_db      hotspot_user      ::1/128                 scram-sha-256
hostssl hotspot_db      hotspot_user      10.0.0.0/8              scram-sha-256
```

## 相关资源

- [PostgreSQL 官方文档 - Client Authentication](https://www.postgresql.org/docs/current/auth-pg-hba-conf.html)
- [PostgreSQL 官方文档 - User Name Maps](https://www.postgresql.org/docs/current/auth-username-maps.html)
- [PostgreSQL 安全检查清单](https://www.postgresql.org/docs/current/security.html)

## 总结

完成以上配置后，`hotspot_user` 将能够：

- ✅ 无密码连接本地 PostgreSQL 数据库
- ✅ 使用 `CREATEDB` 权限创建新数据库
- ✅ 通过模板快速创建测试数据库
- ✅ 与 SQLx 测试框架无缝集成

如果遇到问题，请检查：
1. PostgreSQL 服务是否运行
2. `pg_hba.conf` 配置是否正确
3. 配置是否已重新加载
4. 规则顺序是否正确

