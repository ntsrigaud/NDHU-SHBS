"""Metadata extraction router.

POST /analyze/metadata
  - Receives a base64-encoded book cover image.
  - Returns predicted title, author, ISBN, and confidence.

Phase 0: returns a stub response. Phase 4 will integrate the Roboflow model.
"""

from __future__ import annotations

from fastapi import APIRouter

from schemas import MetadataRequest, MetadataResponse

router = APIRouter(prefix="/analyze", tags=["metadata"])


@router.post("/metadata", response_model=MetadataResponse)
async def analyze_metadata(body: MetadataRequest) -> MetadataResponse:
    """Extract book metadata from a cover image.

    Phase 0 stub — returns fixed placeholder values so the server integration
    can be developed and tested without a live ML model.
    """
    # TODO (Phase 4): call Roboflow / OCR pipeline with body.image_base64
    return MetadataResponse(
        title=None,
        author=None,
        isbn=None,
        confidence=0.0,
    )
