#!/bin/bash

# ============================================================================
# HotSpot 开发数据库重置脚本
# ============================================================================
# 
# 功能：快速重置开发环境数据库
# 用法：./reset_dev_db.sh [数据库名]
# 
# ============================================================================

set -e  # 遇到错误立即退出

# 配置
DB_NAME=${1:-hotspot}
DB_USER=${DB_USER:-postgres}
DB_HOST=${DB_HOST:-localhost}
DB_PORT=${DB_PORT:-5432}
APP_USER=${APP_USER:-hotspot_app}

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 日志函数
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查依赖
check_dependencies() {
    log_info "检查依赖..."
    
    if ! command -v psql &> /dev/null; then
        log_error "psql 未安装或不在 PATH 中"
        exit 1
    fi
    
    if ! psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d postgres -c "SELECT 1;" &> /dev/null; then
        log_error "无法连接到 PostgreSQL 服务器"
        log_error "请检查连接参数：$DB_USER@$DB_HOST:$DB_PORT"
        exit 1
    fi
    
    log_success "依赖检查通过"
}

# 备份现有数据库（可选）
backup_database() {
    if psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d postgres -lqt | cut -d \| -f 1 | grep -qw $DB_NAME; then
        log_warning "数据库 $DB_NAME 已存在"
        
        read -p "是否要备份现有数据库？(y/N): " -n 1 -r
        echo
        
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            BACKUP_FILE="${DB_NAME}_backup_$(date +%Y%m%d_%H%M%S).sql"
            log_info "备份数据库到 $BACKUP_FILE..."
            
            pg_dump -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME > $BACKUP_FILE
            log_success "备份完成：$BACKUP_FILE"
        fi
    fi
}

# 重置数据库
reset_database() {
    log_info "重置数据库 $DB_NAME..."
    
    # 删除现有数据库
    psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d postgres -c "DROP DATABASE IF EXISTS $DB_NAME;" 2>/dev/null || true
    
    # 创建新数据库
    psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d postgres -c "CREATE DATABASE $DB_NAME;"
    
    log_success "数据库 $DB_NAME 已重置"
}

# 执行迁移
run_migration() {
    log_info "执行数据库迁移..."
    
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    MIGRATION_FILE="$SCRIPT_DIR/01_complete_schema.sql"
    
    if [ ! -f "$MIGRATION_FILE" ]; then
        log_error "迁移文件不存在：$MIGRATION_FILE"
        exit 1
    fi
    
    psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f "$MIGRATION_FILE"
    
    log_success "数据库迁移完成"
}

# 设置权限
setup_permissions() {
    log_info "设置应用权限..."
    
    # 创建应用用户（如果不存在）
    psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d postgres -c "
        DO \$\$
        BEGIN
            IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = '$APP_USER') THEN
                CREATE ROLE $APP_USER LOGIN PASSWORD '$APP_USER';
            END IF;
        END
        \$\$;
    " 2>/dev/null || true
    
    # 设置权限
    psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "
        GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO $APP_USER;
        GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO $APP_USER;
        GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO $APP_USER;
        GRANT CREATE ON SCHEMA public TO $APP_USER;
    "
    
    log_success "权限设置完成"
}

# 验证安装
verify_installation() {
    log_info "验证安装..."
    
    # 检查表数量
    TABLE_COUNT=$(psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -t -c "
        SELECT COUNT(*) FROM information_schema.tables 
        WHERE table_schema = 'public' AND table_type = 'BASE TABLE';
    " | xargs)
    
    if [ "$TABLE_COUNT" -ge 20 ]; then
        log_success "表创建成功：$TABLE_COUNT 个表"
    else
        log_error "表创建可能有问题：只有 $TABLE_COUNT 个表"
        exit 1
    fi
    
    # 检查扩展
    EXTENSIONS=$(psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -t -c "
        SELECT COUNT(*) FROM pg_extension 
        WHERE extname IN ('postgis', 'pgcrypto', 'uuid-ossp');
    " | xargs)
    
    if [ "$EXTENSIONS" -eq 3 ]; then
        log_success "扩展安装成功：$EXTENSIONS 个扩展"
    else
        log_warning "扩展可能缺失：只有 $EXTENSIONS 个扩展"
    fi
    
    # 测试用户ID编码
    USER_ID=$(psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -t -c "
        SELECT generate_user_id(2);
    " | xargs)
    
    if [ ! -z "$USER_ID" ]; then
        log_success "用户ID编码功能正常：$USER_ID"
    else
        log_error "用户ID编码功能异常"
        exit 1
    fi
    
    log_success "数据库验证通过！"
}

# 显示连接信息
show_connection_info() {
    echo
    log_info "数据库连接信息："
    echo "  数据库名: $DB_NAME"
    echo "  主机地址: $DB_HOST:$DB_PORT"
    echo "  管理用户: $DB_USER"
    echo "  应用用户: $APP_USER"
    echo
    log_info "连接命令："
    echo "  psql -h $DB_HOST -p $DB_PORT -U $APP_USER -d $DB_NAME"
    echo
}

# 主函数
main() {
    echo "============================================================================"
    echo "🚀 HotSpot 开发数据库重置脚本"
    echo "============================================================================"
    echo
    
    check_dependencies
    backup_database
    reset_database
    run_migration
    setup_permissions
    verify_installation
    show_connection_info
    
    echo "============================================================================"
    log_success "🎉 数据库重置完成！现在可以开始开发了！"
    echo "============================================================================"
}

# 执行主函数
main "$@"
