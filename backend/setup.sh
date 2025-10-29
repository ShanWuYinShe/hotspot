#!/bin/bash

echo "🚀 HotSpot Backend Setup Script"
echo "================================"

# 检查 Docker
if ! command -v docker &> /dev/null; then
    echo "❌ Docker not found. Please install Docker first."
    exit 1
fi

# 检查 Rust
if ! command -v cargo &> /dev/null; then
    echo "❌ Rust not found. Please install Rust first."
    exit 1
fi

# 1. 复制环境变量文件
if [ ! -f .env ]; then
    echo "📝 Creating .env file..."
    cp env.example .env
    echo "✅ .env file created. Please edit it with your configuration."
else
    echo "✅ .env file already exists."
fi

# 2. 启动 Docker 服务
echo "🐳 Starting PostgreSQL and Redis..."
docker-compose up -d

# 等待服务启动
echo "⏳ Waiting for services to be ready..."
sleep 5

# 3. 安装 sqlx-cli（如果未安装）
if ! command -v sqlx &> /dev/null; then
    echo "📦 Installing sqlx-cli..."
    cargo install sqlx-cli --no-default-features --features postgres
fi

# 4. 创建数据库
echo "🗄️  Creating database..."
sqlx database create

# 5. 运行迁移
echo "📊 Running migrations..."
sqlx migrate run

echo ""
echo "✅ Setup complete!"
echo ""
echo "To start the server, run:"
echo "  cargo run"
echo ""
echo "The server will be available at http://localhost:8080"

