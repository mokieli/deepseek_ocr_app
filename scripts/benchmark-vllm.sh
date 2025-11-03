#!/bin/bash
# vLLM 性能对比测试脚本
# 用途: 对比不同版本的vLLM Dockerfile性能差异

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 配置
ENDPOINT="${1:-http://localhost:8000}"
TEST_IMAGE="${2:-./test-images/sample.jpg}"
ITERATIONS="${3:-20}"
WARMUP_ITERATIONS="${4:-3}"

echo -e "${GREEN}=== vLLM 性能基准测试 ===${NC}"
echo "端点: $ENDPOINT"
echo "测试图像: $TEST_IMAGE"
echo "迭代次数: $ITERATIONS"
echo ""

# 检查测试图像是否存在
if [ ! -f "$TEST_IMAGE" ]; then
    echo -e "${RED}错误: 测试图像不存在: $TEST_IMAGE${NC}"
    echo "请提供有效的图像路径"
    exit 1
fi

# 检查服务是否可用
echo -e "${YELLOW}检查服务健康状态...${NC}"
if ! curl -f -s "${ENDPOINT}/health" > /dev/null; then
    echo -e "${RED}错误: 服务不可用或健康检查失败${NC}"
    echo "请确保服务正在运行: docker compose -f docker-compose.vllm.yml up -d"
    exit 1
fi
echo -e "${GREEN}✓ 服务健康${NC}"
echo ""

# 预热
echo -e "${YELLOW}预热中 (${WARMUP_ITERATIONS}次)...${NC}"
for i in $(seq 1 $WARMUP_ITERATIONS); do
    curl -X POST "${ENDPOINT}/ocr" \
        -F "file=@${TEST_IMAGE}" \
        -o /dev/null -s -w ""
    echo -ne "预热进度: $i/$WARMUP_ITERATIONS\r"
done
echo -e "${GREEN}✓ 预热完成${NC}"
echo ""

# 创建结果文件
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
RESULT_FILE="benchmark_results_${TIMESTAMP}.txt"

echo -e "${YELLOW}开始性能测试...${NC}"
echo "开始时间: $(date)" | tee "$RESULT_FILE"
echo "===============================================" | tee -a "$RESULT_FILE"
echo "" | tee -a "$RESULT_FILE"

# 性能测试
declare -a TIMES
TOTAL_TIME=0
MIN_TIME=999999
MAX_TIME=0

for i in $(seq 1 $ITERATIONS); do
    # 测量时间（包括HTTP往返）
    TIME=$(curl -X POST "${ENDPOINT}/ocr" \
        -F "file=@${TEST_IMAGE}" \
        -o /dev/null -s -w "%{time_total}\n")
    
    TIMES+=($TIME)
    TOTAL_TIME=$(echo "$TOTAL_TIME + $TIME" | bc)
    
    # 更新最小/最大时间
    if (( $(echo "$TIME < $MIN_TIME" | bc -l) )); then
        MIN_TIME=$TIME
    fi
    if (( $(echo "$TIME > $MAX_TIME" | bc -l) )); then
        MAX_TIME=$TIME
    fi
    
    echo -ne "进度: $i/$ITERATIONS | 当前: ${TIME}s | 平均: $(echo "scale=3; $TOTAL_TIME / $i" | bc)s\r"
done

echo ""
echo ""

# 计算统计数据
AVG_TIME=$(echo "scale=3; $TOTAL_TIME / $ITERATIONS" | bc)
THROUGHPUT=$(echo "scale=2; $ITERATIONS / $TOTAL_TIME" | bc)

# 计算中位数和百分位数
sorted_times=($(printf '%s\n' "${TIMES[@]}" | sort -n))
p50_index=$((ITERATIONS / 2))
p95_index=$((ITERATIONS * 95 / 100))
p99_index=$((ITERATIONS * 99 / 100))

P50=${sorted_times[$p50_index]}
P95=${sorted_times[$p95_index]}
P99=${sorted_times[$p99_index]}

# 计算标准差
sum_squared_diff=0
for time in "${TIMES[@]}"; do
    diff=$(echo "$time - $AVG_TIME" | bc)
    squared=$(echo "$diff * $diff" | bc)
    sum_squared_diff=$(echo "$sum_squared_diff + $squared" | bc)
done
variance=$(echo "scale=6; $sum_squared_diff / $ITERATIONS" | bc)
stddev=$(echo "scale=3; sqrt($variance)" | bc -l)

# 输出结果
echo -e "${GREEN}=== 测试结果 ===${NC}" | tee -a "$RESULT_FILE"
echo "" | tee -a "$RESULT_FILE"
echo "总请求数: $ITERATIONS" | tee -a "$RESULT_FILE"
echo "总耗时: ${TOTAL_TIME}s" | tee -a "$RESULT_FILE"
echo "" | tee -a "$RESULT_FILE"
echo "响应时间统计:" | tee -a "$RESULT_FILE"
echo "  平均值: ${AVG_TIME}s" | tee -a "$RESULT_FILE"
echo "  中位数 (P50): ${P50}s" | tee -a "$RESULT_FILE"
echo "  P95: ${P95}s" | tee -a "$RESULT_FILE"
echo "  P99: ${P99}s" | tee -a "$RESULT_FILE"
echo "  最小值: ${MIN_TIME}s" | tee -a "$RESULT_FILE"
echo "  最大值: ${MAX_TIME}s" | tee -a "$RESULT_FILE"
echo "  标准差: ${stddev}s" | tee -a "$RESULT_FILE"
echo "" | tee -a "$RESULT_FILE"
echo "吞吐量: ${THROUGHPUT} req/s" | tee -a "$RESULT_FILE"
echo "" | tee -a "$RESULT_FILE"
echo "===============================================" | tee -a "$RESULT_FILE"
echo "结束时间: $(date)" | tee -a "$RESULT_FILE"

echo ""
echo -e "${GREEN}结果已保存到: $RESULT_FILE${NC}"

# 获取系统信息
echo ""
echo -e "${YELLOW}收集系统信息...${NC}"
SYSTEM_INFO_FILE="system_info_${TIMESTAMP}.txt"

{
    echo "=== 系统信息 ==="
    echo ""
    echo "Docker版本:"
    docker --version
    echo ""
    echo "Docker Compose版本:"
    docker compose version
    echo ""
    echo "NVIDIA驱动信息:"
    nvidia-smi --query-gpu=name,driver_version,memory.total --format=csv
    echo ""
    echo "容器信息:"
    docker compose -f docker-compose.vllm.yml ps
    echo ""
    echo "容器资源使用:"
    docker stats --no-stream --format "table {{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}"
} > "$SYSTEM_INFO_FILE"

echo -e "${GREEN}系统信息已保存到: $SYSTEM_INFO_FILE${NC}"
echo ""
echo -e "${GREEN}✓ 测试完成${NC}"

