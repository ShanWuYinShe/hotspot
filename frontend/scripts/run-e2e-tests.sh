#!/bin/bash

# HotSpot E2E测试运行脚本
# 
# 自动化端到端测试执行，包括环境检查、测试运行和报告生成

set -e

echo "🚀 HotSpot E2E测试启动"
echo "======================="

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 函数：打印彩色消息
print_message() {
    local color=$1
    local message=$2
    echo -e "${color}${message}${NC}"
}

# 检查Node.js版本
check_node_version() {
    print_message $BLUE "📋 检查Node.js版本..."
    if command -v node &> /dev/null; then
        NODE_VERSION=$(node --version)
        print_message $GREEN "✅ Node.js版本: $NODE_VERSION"
    else
        print_message $RED "❌ 未找到Node.js，请先安装Node.js"
        exit 1
    fi
}

# 检查依赖
check_dependencies() {
    print_message $BLUE "📦 检查项目依赖..."
    
    if [ ! -d "node_modules" ]; then
        print_message $YELLOW "⚠️ 未找到node_modules，正在安装依赖..."
        npm install
    fi
    
    # 检查Playwright是否安装
    if [ ! -d "node_modules/@playwright" ]; then
        print_message $YELLOW "⚠️ 未找到Playwright，正在安装..."
        npm install @playwright/test
        npx playwright install
    fi
    
    print_message $GREEN "✅ 依赖检查完成"
}

# 检查后端服务
check_backend_service() {
    print_message $BLUE "🔌 检查后端服务..."
    
    BACKEND_URL="http://localhost:8080"
    
    if curl -s -f "$BACKEND_URL/health" > /dev/null 2>&1; then
        print_message $GREEN "✅ 后端服务运行正常"
    else
        print_message $YELLOW "⚠️ 后端服务未运行，测试将在模拟模式下进行"
        print_message $YELLOW "   如需完整测试，请先启动后端服务: cd ../backend && cargo run"
    fi
}

# 清理旧的测试结果
cleanup_old_results() {
    print_message $BLUE "🧹 清理旧的测试结果..."
    
    if [ -d "test-results" ]; then
        rm -rf test-results
        print_message $GREEN "✅ 旧测试结果已清理"
    fi
    
    mkdir -p test-results
}

# 运行测试
run_tests() {
    print_message $BLUE "🧪 开始运行E2E测试..."
    
    # 设置环境变量
    export NODE_ENV=test
    export BASE_URL=${BASE_URL:-"http://localhost:3000"}
    
    # 运行测试的选项
    local test_options=""
    
    # 检查命令行参数
    while [[ $# -gt 0 ]]; do
        case $1 in
            --headed)
                test_options="$test_options --headed"
                shift
                ;;
            --debug)
                test_options="$test_options --debug"
                shift
                ;;
            --project=*)
                test_options="$test_options --project=${1#*=}"
                shift
                ;;
            --grep=*)
                test_options="$test_options --grep=${1#*=}"
                shift
                ;;
            --workers=*)
                test_options="$test_options --workers=${1#*=}"
                shift
                ;;
            *)
                print_message $YELLOW "⚠️ 未知参数: $1"
                shift
                ;;
        esac
    done
    
    print_message $BLUE "测试选项: $test_options"
    
    # 运行Playwright测试
    if npx playwright test $test_options; then
        print_message $GREEN "✅ 测试运行完成"
        return 0
    else
        print_message $RED "❌ 测试运行失败"
        return 1
    fi
}

# 生成测试报告
generate_report() {
    print_message $BLUE "📊 生成测试报告..."
    
    if [ -f "test-results/results.json" ]; then
        # 显示测试摘要
        if command -v jq &> /dev/null; then
            print_message $BLUE "📈 测试结果摘要:"
            jq -r '.stats | "总测试数: \(.total // 0)\n通过: \(.passed // 0)\n失败: \(.failed // 0)\n跳过: \(.skipped // 0)"' test-results/results.json
        fi
        
        print_message $GREEN "✅ 测试报告生成完成"
        print_message $BLUE "📋 报告位置:"
        print_message $BLUE "   - HTML报告: test-results/html-report/index.html"
        print_message $BLUE "   - JSON结果: test-results/results.json"
        
        # 自动打开HTML报告（可选）
        if [[ "$OSTYPE" == "darwin"* ]]; then
            # macOS
            if [ -f "test-results/html-report/index.html" ]; then
                print_message $BLUE "🌐 自动打开测试报告..."
                open test-results/html-report/index.html
            fi
        elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
            # Linux
            if [ -f "test-results/html-report/index.html" ] && command -v xdg-open &> /dev/null; then
                print_message $BLUE "🌐 自动打开测试报告..."
                xdg-open test-results/html-report/index.html
            fi
        fi
        
    else
        print_message $YELLOW "⚠️ 未找到测试结果文件"
    fi
}

# 显示帮助信息
show_help() {
    echo "HotSpot E2E测试运行脚本"
    echo ""
    echo "用法: $0 [选项]"
    echo ""
    echo "选项:"
    echo "  --headed              在有界面模式下运行测试"
    echo "  --debug               启用调试模式"
    echo "  --project=PROJECT     只运行指定项目的测试"
    echo "  --grep=PATTERN        只运行匹配模式的测试"
    echo "  --workers=NUMBER      设置并行工作进程数"
    echo "  --help                显示此帮助信息"
    echo ""
    echo "示例:"
    echo "  $0                    运行所有测试"
    echo "  $0 --headed           在有界面模式下运行"
    echo "  $0 --project=chromium 只在Chrome中运行测试"
    echo "  $0 --grep=auth        只运行认证相关测试"
}

# 主函数
main() {
    # 检查是否请求帮助
    if [[ "$1" == "--help" ]] || [[ "$1" == "-h" ]]; then
        show_help
        exit 0
    fi
    
    # 记录开始时间
    START_TIME=$(date +%s)
    
    # 执行检查和测试
    check_node_version
    check_dependencies
    check_backend_service
    cleanup_old_results
    
    # 运行测试
    if run_tests "$@"; then
        TEST_SUCCESS=true
    else
        TEST_SUCCESS=false
    fi
    
    # 生成报告
    generate_report
    
    # 计算耗时
    END_TIME=$(date +%s)
    DURATION=$((END_TIME - START_TIME))
    
    print_message $BLUE "⏱️ 总耗时: ${DURATION}秒"
    
    # 显示最终结果
    if [ "$TEST_SUCCESS" = true ]; then
        print_message $GREEN "🎉 E2E测试完成！"
        exit 0
    else
        print_message $RED "💥 E2E测试失败！"
        exit 1
    fi
}

# 运行主函数
main "$@"
