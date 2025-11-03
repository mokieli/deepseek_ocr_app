#!/bin/bash
# DeepSeek-OCR vLLM Direct 快速启动脚本

set -e

echo "=================================="
echo "DeepSeek-OCR vLLM Direct 启动脚本"
echo "=================================="
echo ""

# 检查配置文件
if [ ! -f .env ]; then
    echo "⚠️  未找到 .env 文件，复制默认配置..."
    cp .env.vllm-direct .env
    echo "✅ 已创建 .env 文件，请根据需要修改配置"
    echo ""
fi

# 显示当前配置
echo "📋 当前配置："
echo "-----------------------------------"
if [ -f .env ]; then
    grep -E "^(MODEL_PATH|TENSOR_PARALLEL_SIZE|GPU_MEMORY_UTILIZATION|MAX_MODEL_LEN|BASE_SIZE|IMAGE_SIZE|CROP_MODE|VLLM_USE_V1)=" .env || true
fi
echo "-----------------------------------"
echo ""

# 检查 Docker 和 NVIDIA runtime
echo "🔍 检查环境..."
if ! command -v docker &> /dev/null; then
    echo "❌ Docker 未安装"
    exit 1
fi

if ! docker info | grep -q nvidia; then
    echo "⚠️  警告：未检测到 NVIDIA runtime，GPU 可能无法使用"
fi

echo "✅ 环境检查通过"
echo ""

# 询问是否构建镜像
read -p "是否重新构建 Docker 镜像？(y/N): " -n 1 -r
echo ""
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo "🔨 构建 Docker 镜像..."
    docker-compose -f docker-compose.vllm-direct.yml build
    echo "✅ 镜像构建完成"
    echo ""
fi

# 启动服务
echo "🚀 启动服务..."
docker-compose -f docker-compose.vllm-direct.yml up -d

echo ""
echo "⏳ 等待服务启动..."
sleep 5

# 检查服务状态
echo ""
echo "📊 服务状态："
docker-compose -f docker-compose.vllm-direct.yml ps

echo ""
echo "=================================="
echo "✅ 启动完成！"
echo "=================================="
echo ""
echo "📖 使用指南："
echo "  - API 文档: http://localhost:8001/docs"
echo "  - 健康检查: http://localhost:8001/health"
echo "  - 前端界面: http://localhost:3000 (如果启用)"
echo ""
echo "📝 查看日志："
echo "  docker-compose -f docker-compose.vllm-direct.yml logs -f backend-direct"
echo ""
echo "🛑 停止服务："
echo "  docker-compose -f docker-compose.vllm-direct.yml down"
echo ""
