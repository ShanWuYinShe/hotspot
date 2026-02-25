#!/bin/bash
set -e

# 确保脚本在项目根目录运行
cd "$(dirname "$0")/.."

echo "Preparing for Docker deployment..."

# 检查 sqlx-cli 是否安装
if ! command -v sqlx &> /dev/null; then
    echo "sqlx-cli could not be found. Installing..."
    cargo install sqlx-cli --no-default-features --features postgres
fi

# 检查是否有运行中的数据库用于 prepare
# 如果没有，提示用户
echo "Checking for running database..."
# 这里我们假设用户本地开发环境已经有一个 DB (如 docker-compose up postgres)
# 如果没有，sqlx prepare 会失败。
# 但我们可以尝试启动 docker-compose 中的 postgres 吗？太复杂。

# 尝试生成 sqlx-data.json
if [ -f "backend/Cargo.toml" ]; then
    echo "Generating sqlx-data.json for offline build..."
    cd backend
    # 确保 DATABASE_URL 设置正确（通常是 localhost）
    # 如果 .env 存在，sqlx 会读取
    cargo sqlx prepare --workspace
    cd ..
else
    echo "backend directory not found!"
    exit 1
fi

echo "Preparation complete. You can now run:"
echo "  docker-compose up --build"
