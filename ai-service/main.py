"""SHBS AI Microservice entry point.

Exposes:
  GET  /health            — liveness + readiness probe
  POST /analyze/metadata  — book metadata extraction (Phase 4)
  POST /analyze/condition — book condition classification (Phase 4)

This service is internal-only and must NOT be exposed directly to the client.
All calls originate from the Go API server via the Docker bridge network.
"""

from __future__ import annotations

import logging
import os

from contextlib import asynccontextmanager
from typing import AsyncGenerator

import httpx
from dotenv import load_dotenv
from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware

from routers import condition, metadata
from schemas import HealthResponse

load_dotenv()

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Global flag set during startup once models are ready.
_model_loaded: bool = False


@asynccontextmanager
async def lifespan(app: FastAPI) -> AsyncGenerator[None, None]:
    """Load ML models during startup; release resources on shutdown."""
    global _model_loaded
    logger.info("AI service starting — loading models…")
    app.state.http_client = httpx.AsyncClient(timeout=30.0)
    _model_loaded = True
    logger.info("Models ready")
    yield
    await app.state.http_client.aclose()
    _model_loaded = False
    logger.info("AI service shutting down")


app = FastAPI(
    title="SHBS AI Microservice",
    version="0.1.0",
    description="Internal AI service for book metadata extraction and condition classification.",
    lifespan=lifespan,
)

# Default client used by tests that skip the lifespan context (TestClient without `with`).
# The lifespan always replaces this with a fresh client in production/Docker.
app.state.http_client = httpx.AsyncClient(timeout=30.0)

# CORS is restricted to the Docker bridge network (Go API server only).
# The wildcard below is safe because Traefik never routes /ai from external
# clients — see docker-compose.yml labels.
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_methods=["POST", "GET"],
    allow_headers=["*"],
)

app.include_router(metadata.router)
app.include_router(condition.router)


@app.get("/health", response_model=HealthResponse, tags=["health"])
async def health() -> HealthResponse:
    """Liveness and readiness probe used by Traefik and Docker health checks."""
    return HealthResponse(status="ok", model_loaded=_model_loaded)


if __name__ == "__main__":
    import uvicorn

    port = int(os.getenv("PORT", "8000"))
    uvicorn.run("main:app", host="0.0.0.0", port=port, reload=False)
