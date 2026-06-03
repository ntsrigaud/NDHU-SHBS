from __future__ import annotations

from pydantic import BaseModel, Field


# ── Request schemas ────────────────────────────────────────────────────────────

class MetadataRequest(BaseModel):
    """Image bytes are supplied as a base64-encoded string."""

    image_base64: str = Field(..., description="Base64-encoded book cover image")


class ConditionRequest(BaseModel):
    """Condition classification request; one or more cover photos."""

    images_base64: list[str] = Field(
        ..., min_length=1, max_length=10, description="Base64-encoded book photo(s)"
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
