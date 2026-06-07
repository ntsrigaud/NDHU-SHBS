"""Condition classification router.

POST /analyze/condition
  - Calls Roboflow object-detection endpoint for each image.
  - Aggregates detections into a single condition rating.

Scoring:
  Per-image defect score = Σ(weight × confidence), capped at 1.0
  Worst-image score across all images determines the final condition.
  score (quality)  = 1.0 − worst_defect_score  (1.0 = perfect, 0.0 = destroyed)
  condition        = "good"     if defect_score < 0.25
                     "moderate" if defect_score < 0.60
                     "poor"     otherwise
  confidence       = mean of per-image avg-detection-confidences
                     (1.0 for an image that has no detections)
"""

from __future__ import annotations

import base64
import logging
import os
from typing import Any

import httpx
from fastapi import APIRouter, HTTPException, Request

from schemas import ConditionRequest, ConditionResponse

logger = logging.getLogger(__name__)

router = APIRouter(prefix="/analyze", tags=["condition"])

# Severity weight per Roboflow class.
# label and barcode are book identifiers, not physical defects — weight 0.0.
# annotations/crease/nick/spot have limited training data (faded UI boxes) — weight 0.3.
_SEVERITY: dict[str, float] = {
    "tear":        1.0,
    "chip":        1.0,
    "stain":       0.6,
    "mark":        0.5,
    "spot":        0.3,
    "annotations": 0.3,
    "crease":      0.3,
    "nick":        0.3,
    "label":       0.0,
    "barcode":     0.0,
}


def _score_predictions(predictions: list[dict[str, Any]]) -> tuple[float, float]:
    """Return (defect_score, avg_confidence) for one image's prediction list.

    defect_score   – weighted sum of detections, capped at 1.0
    avg_confidence – mean detection confidence; 1.0 when predictions is empty
                     (model is confidently seeing no defects)
    """
    if not predictions:
        return 0.0, 1.0

    defect_score = sum(
        _SEVERITY.get(p.get("class", ""), 0.0) * p.get("confidence", 0.0)
        for p in predictions
    )
    avg_conf = sum(p.get("confidence", 0.0) for p in predictions) / len(predictions)
    return min(defect_score, 1.0), avg_conf


@router.post("/condition", response_model=ConditionResponse)
async def analyze_condition(body: ConditionRequest, request: Request) -> ConditionResponse:
    """Classify the physical condition of a used textbook."""
    api_key = os.getenv("ROBOFLOW_API_KEY", "")
    model_endpoint = os.getenv(
        "MODEL_ENDPOINT",
        "book-defect-detection/10",
    )

    # Ensure the endpoint is a full URL. If it's just "model/version",
    # prepend the Roboflow base URL.
    if model_endpoint.startswith("http"):
        endpoint = model_endpoint
    else:
        endpoint = f"https://detect.roboflow.com/{model_endpoint.lstrip('/')}"

    client: httpx.AsyncClient = request.app.state.http_client

    images_base64 = body.images_base64 or []
    image_urls = body.image_urls or []
    
    # We need base64 for Roboflow API
    all_b64: list[str] = list(images_base64)
    for url in image_urls:
        try:
            resp = await client.get(url)
            resp.raise_for_status()
            # Convert bytes to base64
            all_b64.append(base64.b64encode(resp.content).decode("utf-8"))
        except httpx.HTTPError as exc:
            logger.warning("Could not download image from %s: %s", url, exc)

    if not all_b64:
        raise HTTPException(status_code=400, detail="No valid images provided")

    per_image_defect: list[float] = []
    per_image_confidence: list[float] = []

    for image_b64 in all_b64:
        # Roboflow's hosted API wants the raw base64 only. Strip a data-URI
        # prefix (e.g. "data:image/jpeg;base64,") if the caller included one,
        # plus any surrounding whitespace/newlines — otherwise Roboflow 400s.
        if "," in image_b64 and image_b64.lstrip().startswith("data:"):
            image_b64 = image_b64.split(",", 1)[1]
        image_b64 = image_b64.strip()

        try:
            resp = await client.post(
                endpoint,
                # confidence=5 lowers Roboflow's detection threshold to 5% so
                # weak/uncertain defect detections are not dropped before we
                # score them. Value is a percentage (0–100).
                params={"api_key": api_key, "confidence": 5},
                content=image_b64,
                headers={"Content-Type": "application/x-www-form-urlencoded"},
            )
            resp.raise_for_status()
            data = resp.json()
        except httpx.HTTPError as exc:
            logger.error("Roboflow inference failed: %s", exc)
            raise HTTPException(status_code=502, detail="AI inference unavailable") from exc

        defect, conf = _score_predictions(data.get("predictions", []))
        per_image_defect.append(defect)
        per_image_confidence.append(conf)

    worst = max(per_image_defect)
    overall_conf = sum(per_image_confidence) / len(per_image_confidence)

    if worst < 0.25:
        condition = "good"
    elif worst < 0.60:
        condition = "moderate"
    else:
        condition = "poor"

    return ConditionResponse(
        condition=condition,
        score=round(1.0 - worst, 6),
        confidence=round(overall_conf, 6),
    )
