#!/bin/bash
# HotSpot 数据库管理脚本
# 
# 用法：
#   ./setup_database.sh        # 创建数据库（首次会自动初始化模板）
#   ./setup_database.sh reset  # 重置数据库

set -e

# 加载配置
readonly SCRIPT_DIR="$(dirname "$0")"
source "$SCRIPT_DIR/config.sh"

# 颜色定义
readonly GREEN='\033[0;32m'
readonly BLUE='\033[0;34m'
readonly YELLOW='\033[1;33m'
readonly RED='\033[0;31m'
readonly NC='\033[0m'

# 检查 PostgreSQL 服务
check_postgres() {
    if ! $PG_ISREADY -q 2>/dev/null; then
        echo -e "${RED}❌ PostgreSQL 未运行${NC}"
        exit 1
    fi
}

# 检查模板是否存在
template_exists() {
    local exists
    if [ "$PG_SUPERUSER" = "$(whoami)" ]; then
        exists=$($PSQL -d postgres -t -c "SELECT COUNT(*) FROM pg_database WHERE datname = '$DB_TEMPLATE';" 2>/dev/null | tr -d ' \n' || echo "0")
    else
        exists=$(sudo -u "$PG_SUPERUSER" "$PSQL" -t -c "SELECT COUNT(*) FROM pg_database WHERE datname = '$DB_TEMPLATE';" 2>/dev/null | tr -d ' \n' || echo "0")
    fi
    [ "$exists" -eq 1 ]
}

# 检查数据库是否存在
database_exists() {
    local exists=$($PSQL -U "$DB_USER" -d postgres -t -c "SELECT COUNT(*) FROM pg_database WHERE datname = '$1';" 2>/dev/null | tr -d ' \n' || echo "0")
    [ "$exists" -eq 1 ]
}

# 初始化模板（仅首次）
init_template() {
    echo -e "${BLUE}🚀 初始化模板数据库...${NC}"
    
    if [ "$PG_SUPERUSER" = "$(whoami)" ]; then
        "$PSQL" -d postgres -f "$SCRIPT_DIR/init_template.sql" >/dev/null
    else
        if ! sudo -n true 2>/dev/null; then
            echo -e "${YELLOW}⚠️  需要 sudo 权限创建模板${NC}"
        fi
        sudo -u "$PG_SUPERUSER" "$PSQL" -f "$SCRIPT_DIR/init_template.sql" >/dev/null
    fi
    
    echo -e "${GREEN}✓${NC} 模板初始化完成"
}

# 创建数据库
create_database() {
    # 如果模板不存在，先初始化
    if ! template_exists; then
        init_template
    fi

    # 如果数据库已存在
    if database_exists "$DB_NAME"; then
        echo -e "${GREEN}✓${NC} 数据库 $DB_NAME 已存在"
        echo ""
        echo "使用 './setup_database.sh reset' 重置数据库"
        return
    fi

    echo -e "${BLUE}🔧 创建数据库: $DB_NAME${NC}"
    $PSQL -U "$DB_USER" -d postgres -c "CREATE DATABASE $DB_NAME TEMPLATE $DB_TEMPLATE;" >/dev/null

    echo -e "${GREEN}✓${NC} 数据库创建完成"
    echo ""
    echo -e "${YELLOW}💡 下一步：${NC}"
    echo "  cd backend && sqlx migrate run"
}

# 重置数据库
reset_database() {
    if ! database_exists "$DB_NAME"; then
        echo -e "${YELLOW}⚠️  数据库不存在，创建新数据库${NC}"
        create_database
        return
    fi

    echo -e "${YELLOW}⚠️  将删除 $DB_NAME 的所有数据！${NC}"
    read -p "确认继续？(y/N): " -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "已取消"
        exit 0
    fi

    echo -e "${BLUE}🔄 重置数据库...${NC}"
    
    # 终止连接
    $PSQL -U "$DB_USER" -d postgres -c "
        SELECT pg_terminate_backend(pid) 
        FROM pg_stat_activity 
        WHERE datname = '$DB_NAME' AND pid <> pg_backend_pid();
    " >/dev/null 2>&1

    # 删除并重建
    $PSQL -U "$DB_USER" -d postgres -c "DROP DATABASE $DB_NAME;" >/dev/null
    $PSQL -U "$DB_USER" -d postgres -c "CREATE DATABASE $DB_NAME TEMPLATE $DB_TEMPLATE;" >/dev/null

    echo -e "${GREEN}✓${NC} 数据库重置完成"
    echo ""
    echo -e "${YELLOW}💡 下一步：${NC}"
    echo "  cd backend && sqlx migrate run"
}

# 主流程
main() {
    check_postgres

    case "${1:-create}" in
        reset|r|-r|--reset)
            reset_database
            ;;
        create|c|-c|--create|"")
            create_database
            ;;
        help|h|-h|--help)
            echo "用法："
            echo "  $0        # 创建数据库（首次会自动初始化模板）"
            echo "  $0 reset  # 重置数据库"
            ;;
        *)
            echo -e "${RED}❌ 未知命令: $1${NC}"
            echo "使用 '$0 help' 查看帮助"
            exit 1
            ;;
    esac
}

main "$@"
