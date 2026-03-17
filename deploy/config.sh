#!/bin/bash
# HotSpot 数据库配置文件
# 
# 🔧 修改此文件可以调整所有脚本和文档中的配置
# 注意：修改配置后需要重新运行初始化脚本

# ============================================================================
# PostgreSQL 版本配置
# ============================================================================
export POSTGRES_VERSION="18"

# ============================================================================
# PostgreSQL 命令配置（自动检测）
# ============================================================================
# 尝试查找 pg_isready 命令（处理版本后缀和路径问题）
if command -v pg_isready >/dev/null 2>&1; then
    export PG_ISREADY="pg_isready"
elif command -v "pg_isready@${POSTGRES_VERSION}" >/dev/null 2>&1; then
    export PG_ISREADY="pg_isready@${POSTGRES_VERSION}"
elif [ -f "/opt/homebrew/opt/postgresql@${POSTGRES_VERSION}/bin/pg_isready" ]; then
    export PG_ISREADY="/opt/homebrew/opt/postgresql@${POSTGRES_VERSION}/bin/pg_isready"
elif [ -f "/usr/local/opt/postgresql@${POSTGRES_VERSION}/bin/pg_isready" ]; then
    export PG_ISREADY="/usr/local/opt/postgresql@${POSTGRES_VERSION}/bin/pg_isready"
else
    export PG_ISREADY="pg_isready"  # 回退到默认命令
fi

# 检测 psql 路径
PG_BIN_DIR=$(dirname "$PG_ISREADY")
if [ -f "$PG_BIN_DIR/psql" ]; then
    export PSQL="$PG_BIN_DIR/psql"
elif command -v psql >/dev/null 2>&1; then
    export PSQL="psql"
else
    export PSQL="psql"  # 回退到默认
fi

# 检测 PostgreSQL 超级用户
# Linux 通常是 postgres，macOS Homebrew 通常是当前用户
if id postgres >/dev/null 2>&1; then
    export PG_SUPERUSER="postgres"
else
    export PG_SUPERUSER=$(whoami)
fi

# ============================================================================
# 数据库配置
# ============================================================================
export DB_USER="liyou"
export DB_TEMPLATE="hotspot_template"
export DB_NAME="hotspot_db"
export DB_HOST="localhost"
export DB_PORT="5432"

# ============================================================================
# 连接字符串
# ============================================================================
export DATABASE_URL="postgresql://${DB_USER}@${DB_HOST}:${DB_PORT}/${DB_NAME}"

# Docker 开发栈（deploy/docker-compose.yml）使用的独立连接配置
export DOCKER_DB_USER="hotspot"
export DOCKER_DB_PASSWORD="hotspot"
export DOCKER_DB_NAME="hotspot"
export DOCKER_DATABASE_URL="postgresql://${DOCKER_DB_USER}:${DOCKER_DB_PASSWORD}@127.0.0.1:5432/${DOCKER_DB_NAME}"

# 测试环境使用独立数据库与独立 Redis，避免污染运行时实例
export TEST_DB_NAME="hotspot_test"
export TEST_DATABASE_URL="${TEST_DATABASE_URL:-postgresql://${DOCKER_DB_USER}:${DOCKER_DB_PASSWORD}@127.0.0.1:5432/${TEST_DB_NAME}}"
export TEST_REDIS_PORT="6380"
export TEST_REDIS_URL="${TEST_REDIS_URL:-redis://127.0.0.1:${TEST_REDIS_PORT}/0}"

# ============================================================================
# 函数：显示当前配置
# ============================================================================
show_config() {
    echo "HotSpot 数据库配置"
    echo "=================="
    echo "PostgreSQL 版本: ${POSTGRES_VERSION}"
    echo "PostgreSQL 超级用户: ${PG_SUPERUSER}"
    echo "PostgreSQL 工具: ${PG_ISREADY}"
    echo "psql 路径: ${PSQL}"
    echo "数据库用户: ${DB_USER}"
    echo "模板数据库: ${DB_TEMPLATE}"
    echo "用户数据库: ${DB_NAME}"
    echo "连接地址: ${DB_HOST}:${DB_PORT}"
    echo "连接字符串: ${DATABASE_URL}"
    echo "Docker 开发库: ${DOCKER_DATABASE_URL}"
    echo "测试数据库: ${TEST_DATABASE_URL}"
    echo "测试 Redis: ${TEST_REDIS_URL}"
}

