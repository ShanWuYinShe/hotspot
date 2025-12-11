# HotSpot 数据库脚本

本目录包含 HotSpot 项目的数据库管理脚本。

> **📌 版本配置**  
> 本项目使用 **PostgreSQL 17**。如需更改版本，请编辑 `config.sh` 文件中的 `POSTGRES_VERSION` 变量。

## 📦 脚本列表

| 脚本 | 用途 | 职责 |
|------|------|------|
| `config.sh` | 配置文件 | 统一管理版本号和数据库配置 |
| `setup_database.sh` | 数据库管理脚本 | 初始化模板 + 创建/重置用户数据库 |
| `init_template.sql` | 模板初始化 SQL | 创建模板数据库（由脚本调用） |

## ⚙️ 配置管理

### 修改 PostgreSQL 版本

编辑 `config.sh` 文件：

```bash
# 修改版本号
export POSTGRES_VERSION="17"  # 改为你安装的版本

# 其他可配置项
export DB_USER="hotspot_user"
export DB_TEMPLATE="hotspot_template"
export DB_NAME="hotspot_db"
```

修改后重新运行初始化脚本即可。

> **💡 自动检测**  
> `config.sh` 会自动检测 PostgreSQL 工具路径和超级用户，实现跨平台兼容：
> - **工具路径**：支持标准安装、版本后缀命令、Homebrew 路径
> - **超级用户**：Linux 使用 `postgres` 用户，macOS Homebrew 使用当前用户
> - **无需修改**：同一套脚本可在 Linux 和 macOS 上运行

### 查看当前配置

```bash
source ./scripts/config.sh
show_config
```

## 🎯 设计理念

**职责分离**：
- **配置文件** (`config.sh`) - 统一管理所有配置（一处修改，处处生效）
- **SQL 脚本** (`init_template.sql`) - 只创建模板数据库（一次性操作）
- **Shell 脚本** (`setup_database.sh`) - 管理用户数据库（可重复执行）

这种设计的好处：
- ✅ 版本号集中管理，修改一处即可
- ✅ 模板创建是一次性的，不会重复执行
- ✅ 用户数据库可以反复创建/删除/重置
- ✅ 职责单一，更符合 Unix 哲学

## 🚀 快速开始

### 自动模式（推荐）

```bash
# 脚本会自动检测并执行需要的操作
# macOS: 直接运行
./scripts/setup_database.sh

# Linux: 需要 sudo（切换到 postgres 用户）
sudo ./scripts/setup_database.sh

# 配置认证（详见 docs/pg_hba_config.md）
# macOS: 通常不需要修改 pg_hba.conf
# Linux: 需要配置后重新加载
sudo -u postgres pg_ctl reload

# 运行迁移
cd backend
export DATABASE_URL="postgresql://hotspot_user@localhost:5432/hotspot_db"
sqlx migrate run
```

### 手动模式

```bash
# 1. 初始化模板数据库
# macOS: 直接运行
./scripts/setup_database.sh --init-template
# Linux: 需要 sudo
sudo ./scripts/setup_database.sh --init-template

# 2. 配置 pg_hba.conf 认证（详见 docs/pg_hba_config.md）
# macOS: 通常不需要
# Linux: 需要重新加载配置
sudo -u postgres pg_ctl reload

# 3. 创建用户数据库
./scripts/setup_database.sh --create

# 4. 运行迁移
cd backend && sqlx migrate run
```

## 🔄 日常开发

### 使用脚本重置

```bash
# 重置数据库（删除 + 从模板重建）
./scripts/setup_database.sh --reset
```

### 使用 SQLx 管理

```bash
cd backend

# 方式 1：分步操作
sqlx database drop     # 删除数据库（会提示确认）
sqlx database create   # 从模板重新创建
sqlx migrate run       # 运行迁移

# 方式 2：一步到位
sqlx database reset    # 等同于 drop + create + migrate
```

## 💡 使用场景

| 场景 | macOS | Linux |
|------|-------|-------|
| 首次设置 | `./setup_database.sh` | `sudo ./setup_database.sh` |
| 重建模板 | `./setup_database.sh --init-template` | `sudo ./setup_database.sh --init-template` |
| 创建数据库 | `./setup_database.sh --create` | `./setup_database.sh --create` |
| 重置数据库 | `./setup_database.sh --reset` | `./setup_database.sh --reset` |
| 查看帮助 | `./setup_database.sh --help` | `./setup_database.sh --help` |

## 📚 详细文档

- 🔐 [认证配置指南](../docs/pg_hba_config.md) - pg_hba.conf 配置详解（仅 Linux 需要）

## ⚙️ 配置摘要

- **用户**：`hotspot_user`（CREATEDB 权限，无密码本地认证）
- **模板数据库**：`hotspot_template`（包含 PostGIS 等扩展，不可连接）
- **用户数据库**：`hotspot_db`（基于模板创建）

## ⚠️ 注意事项

这些脚本仅用于**开发环境**。
