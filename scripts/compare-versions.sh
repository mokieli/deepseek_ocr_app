#!/bin/bash
# vLLM 版本对比测试脚本
# 用途: 自动化对比不同Dockerfile版本的性能

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

echo -e "${BLUE}=== vLLM 版本对比测试 ===${NC}"
echo ""

# 检查必需文件
if [ ! -f "$PROJECT_ROOT/backend/Dockerfile.vllm-direct" ]; then
    echo -e "${RED}错误: Dockerfile.vllm-direct 不存在${NC}"
    exit 1
fi

# 配置
TEST_IMAGE="${1:-$PROJECT_ROOT/test-images/sample.jpg}"
ITERATIONS=20

if [ ! -f "$TEST_IMAGE" ]; then
    echo -e "${RED}错误: 测试图像不存在: $TEST_IMAGE${NC}"
    echo "用法: $0 <test-image-path>"
    exit 1
fi

# 创建结果目录
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
RESULTS_DIR="$PROJECT_ROOT/benchmark_results_$TIMESTAMP"
mkdir -p "$RESULTS_DIR"

echo -e "${YELLOW}结果将保存到: $RESULTS_DIR${NC}"
echo ""

# 测试函数
test_version() {
    local VERSION_NAME=$1
    local DOCKERFILE=$2
    
    echo -e "${BLUE}=== 测试 $VERSION_NAME ===${NC}"
    echo ""
    
    # 备份当前Dockerfile
    cp "$PROJECT_ROOT/backend/Dockerfile.vllm-direct" "$PROJECT_ROOT/backend/Dockerfile.vllm-direct.tmp"
    
    # 使用指定版本
    if [ -f "$DOCKERFILE" ]; then
        cp "$DOCKERFILE" "$PROJECT_ROOT/backend/Dockerfile.vllm-direct"
    else
        echo -e "${YELLOW}警告: $DOCKERFILE 不存在，使用当前版本${NC}"
    fi
    
    echo -e "${YELLOW}停止现有容器...${NC}"
    docker compose down
    
    echo -e "${YELLOW}构建镜像...${NC}"
    docker compose build --no-cache
    
    echo -e "${YELLOW}启动容器...${NC}"
    docker compose up -d
    
    # 等待服务启动
    echo -e "${YELLOW}等待服务启动...${NC}"
    MAX_WAIT=300  # 5分钟
    ELAPSED=0
    while [ $ELAPSED -lt $MAX_WAIT ]; do
        if curl -f -s http://localhost:8001/health > /dev/null 2>&1; then
            echo -e "${GREEN}✓ 服务已就绪${NC}"
            break
        fi
        sleep 5
        ELAPSED=$((ELAPSED + 5))
        echo -ne "已等待 ${ELAPSED}s / ${MAX_WAIT}s\r"
    done
    
    if [ $ELAPSED -ge $MAX_WAIT ]; then
        echo -e "${RED}错误: 服务启动超时${NC}"
        docker compose logs
        return 1
    fi
    
    # 额外等待以确保模型完全加载
    echo -e "${YELLOW}等待模型加载...${NC}"
    sleep 30
    
    # 运行性能测试
    echo ""
    cd "$SCRIPT_DIR"
    ./benchmark-vllm.sh http://localhost:8001 "$TEST_IMAGE" $ITERATIONS 5
    
    # 移动结果文件
    mv benchmark_results_*.txt "$RESULTS_DIR/${VERSION_NAME}_results.txt" 2>/dev/null || true
    mv system_info_*.txt "$RESULTS_DIR/${VERSION_NAME}_system_info.txt" 2>/dev/null || true
    
    # 收集容器日志
    docker compose logs > "$RESULTS_DIR/${VERSION_NAME}_logs.txt"
    
    # 恢复Dockerfile
    mv "$PROJECT_ROOT/backend/Dockerfile.vllm-direct.tmp" "$PROJECT_ROOT/backend/Dockerfile.vllm-direct"
    
    echo ""
    echo -e "${GREEN}✓ $VERSION_NAME 测试完成${NC}"
    echo ""
}

# 测试各个版本
cd "$PROJECT_ROOT"

# 1. 测试当前版本
echo -e "${BLUE}===========================================${NC}"
test_version "current" "$PROJECT_ROOT/backend/Dockerfile.vllm-direct"

# 2. 测试渐进式更新版本（如果存在）
if [ -f "$PROJECT_ROOT/backend/Dockerfile.vllm-direct.updated" ]; then
    echo -e "${BLUE}===========================================${NC}"
    read -p "是否测试渐进式更新版本？(y/n) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        test_version "updated" "$PROJECT_ROOT/backend/Dockerfile.vllm-direct.updated"
    fi
fi

# 3. 测试最新版本（如果存在）
if [ -f "$PROJECT_ROOT/backend/Dockerfile.vllm-direct.latest" ]; then
    echo -e "${BLUE}===========================================${NC}"
    read -p "是否测试最新版本？(y/n) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        test_version "latest" "$PROJECT_ROOT/backend/Dockerfile.vllm-direct.latest"
    fi
fi

# 生成对比报告
echo -e "${BLUE}===========================================${NC}"
echo -e "${YELLOW}生成对比报告...${NC}"

REPORT_FILE="$RESULTS_DIR/comparison_report.md"

{
    echo "# vLLM 版本性能对比报告"
    echo ""
    echo "测试时间: $(date)"
    echo "测试图像: $TEST_IMAGE"
    echo "迭代次数: $ITERATIONS"
    echo ""
    echo "## 测试结果"
    echo ""
    echo "| 版本 | 平均响应时间 | P95延迟 | P99延迟 | 吞吐量 |"
    echo "|------|-------------|---------|---------|--------|"
    
    for result in "$RESULTS_DIR"/*_results.txt; do
        if [ -f "$result" ]; then
            version=$(basename "$result" | sed 's/_results.txt//')
            avg=$(grep "平均值:" "$result" | awk '{print $2}')
            p95=$(grep "P95:" "$result" | awk '{print $2}')
            p99=$(grep "P99:" "$result" | awk '{print $2}')
            throughput=$(grep "吞吐量:" "$result" | awk '{print $2, $3}')
            echo "| $version | $avg | $p95 | $p99 | $throughput |"
        fi
    done
    
    echo ""
    echo "## 详细结果"
    echo ""
    
    for result in "$RESULTS_DIR"/*_results.txt; do
        if [ -f "$result" ]; then
            version=$(basename "$result" | sed 's/_results.txt//')
            echo "### $version"
            echo ""
            echo '```'
            cat "$result"
            echo '```'
            echo ""
        fi
    done
} > "$REPORT_FILE"

echo -e "${GREEN}✓ 对比报告已生成: $REPORT_FILE${NC}"
echo ""

# 清理
echo -e "${YELLOW}清理环境...${NC}"
docker compose down

echo ""
echo -e "${GREEN}=== 所有测试完成 ===${NC}"
echo -e "${GREEN}结果目录: $RESULTS_DIR${NC}"
echo ""
echo "建议查看以下文件:"
echo "  - $REPORT_FILE (对比报告)"
echo "  - $RESULTS_DIR/*_results.txt (详细结果)"
echo "  - $RESULTS_DIR/*_logs.txt (容器日志)"
