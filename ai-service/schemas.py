from __future__ import annotations

from pydantic import BaseModel, Field


# ── Request schemas ────────────────────────────────────────────────────────────

class MetadataRequest(BaseModel):
    """One or more book photos as base64 strings OR public URLs."""

    images_base64: list[str] | None = Field(
        None, max_length=10, description="Base64-encoded book photo(s)"
    )
    image_urls: list[str] | None = Field(
        None, max_length=10, description="Publicly accessible image URL(s)"
    )


class ConditionRequest(BaseModel):
    """Condition classification request; one or more cover photos."""

    images_base64: list[str] | None = Field(
        None, max_length=10, description="Base64-encoded book photo(s)"
    )
    image_urls: list[str] | None = Field(
        None, max_length=10, description="Publicly accessible image URL(s)"
    )


# ── Response schemas ───────────────────────────────────────────────────────────

class MetadataResponse(BaseModel):
    title: str | None = None
    author: str | None = None
    isbn: str | None = None
    confidence: float = Field(..., ge=0.0, le=1.0)


class ConditionResponse(BaseModel):
    condition: str = Field(..., pattern="^(good|moderate|poor)$")
    score: float = Field(..., ge=0.0, le=1.0)
    confidence: float = Field(..., ge=0.0, le=1.0)


class HealthResponse(BaseModel):
    status: str
    model_loaded: bool
