"""
FastAPI 主应用
重构后的精简版入口文件
"""
from contextlib import asynccontextmanager
from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
import uvicorn

from .config import settings
from .api.routes import router, initialize_service, shutdown_service
from .db.session import init_db


@asynccontextmanager
async def lifespan(app: FastAPI):
    """应用生命周期管理"""
    # 启动时加载模型
    print("=" * 60)
    print("🚀 Starting DeepSeek-OCR API Server")
    print("⚙️  Inference Engine: vllm_direct")
    print(f"📦 Model Path: {settings.model_path}")
    print(f"🎮 GPU Config: TP={settings.tensor_parallel_size}, Memory={settings.gpu_memory_utilization}")
    print(f"📏 Max Model Length: {settings.max_model_len}")
    print(f"🧩 OCR Mode: base_size={settings.base_size}, image_size={settings.image_size}, crop_mode={settings.crop_mode}")
    print(f"🧠 vLLM Engine Mode: {'v1' if settings.vllm_use_v1 else 'legacy'}")
    print("=" * 60)
    
    try:
        await init_db()
        await initialize_service()
        print("✅ Service initialized successfully!")
    except Exception as e:
        print(f"❌ Failed to initialize service: {e}")
        import traceback
        traceback.print_exc()
        raise
    
    yield
    
    # 关闭时清理
    print("🛑 Shutting down...")
    await shutdown_service()


# 创建 FastAPI 应用
app = FastAPI(
    title="DeepSeek-OCR API",
    description="Blazing fast OCR with DeepSeek-OCR model 🔥",
    version="4.0.0",
    lifespan=lifespan
)

# CORS 中间件
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# 注册路由
app.include_router(router)


if __name__ == "__main__":
    uvicorn.run(
        "app.main:app",
        host=settings.api_host,
        port=settings.api_port,
        reload=False
    )
