"""Condition classification router.

POST /analyze/condition
  - Receives one or more base64-encoded book photo(s).
  - Returns predicted condition (good | moderate | poor), numeric score,
    and model confidence.

Phase 0: returns a stub response. Phase 4 will integrate the Roboflow model.
"""

from __future__ import annotations

from fastapi import APIRouter

from schemas import ConditionRequest, ConditionResponse

router = APIRouter(prefix="/analyze", tags=["condition"])


@router.post("/condition", response_model=ConditionResponse)
async def analyze_condition(body: ConditionRequest) -> ConditionResponse:
    """Classify the physical condition of a used textbook.

    Phase 0 stub — always returns 'good' with zero confidence so downstream
    code can be integrated before the model is available.
    """
    # TODO (Phase 4): run inference against Roboflow model endpoint
    return ConditionResponse(
        condition="good",
        score=0.0,
        confidence=0.0,
    )
