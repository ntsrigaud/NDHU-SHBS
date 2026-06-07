"""Tests for the condition classification endpoint.

Strategy: assign AsyncMock to app.state.http_client via an autouse fixture
before every test. TestClient is used without a context manager (matching
test_health.py style), so the lifespan does not run — the mock is injected
directly onto app.state instead.
"""

from __future__ import annotations

from unittest.mock import AsyncMock, MagicMock

import httpx
import pytest
from fastapi.testclient import TestClient

from main import app

_TEST_B64 = "dGVzdA=="  # base64("test") — a valid base64 string, not a real image


def _mock_response(predictions: list[dict]) -> MagicMock:
    """Build a mock httpx.Response returning the given predictions list."""
    mock = MagicMock()
    mock.json.return_value = {"predictions": predictions, "time": 0.045}
    mock.raise_for_status = MagicMock()
    return mock


def _pred(cls: str, confidence: float) -> dict:
    return {
        "class": cls,
        "confidence": confidence,
        "x": 100,
        "y": 100,
        "width": 50,
        "height": 50,
        "class_id": 0,
        "detection_id": "abc",
    }


@pytest.fixture
def client() -> TestClient:
    return TestClient(app)


# ── Tests ─────────────────────────────────────────────────────────────────────


def test_no_defects_returns_good(
    client: TestClient, mock_http_client: AsyncMock
) -> None:
    mock_http_client.post.return_value = _mock_response([])

    resp = client.post("/analyze/condition", json={"images_base64": [_TEST_B64]})

    assert resp.status_code == 200
    data = resp.json()
    assert data["condition"] == "good"
    assert data["score"] == pytest.approx(1.0)
    assert data["confidence"] == pytest.approx(1.0)


def test_severe_defect_returns_poor(
    client: TestClient, mock_http_client: AsyncMock
) -> None:
    # tear @ 0.9 → defect = 1.0 × 0.9 = 0.9 → condition "poor", score = 0.1
    mock_http_client.post.return_value = _mock_response([_pred("tear", 0.9)])

    resp = client.post("/analyze/condition", json={"images_base64": [_TEST_B64]})

    assert resp.status_code == 200
    data = resp.json()
    assert data["condition"] == "poor"
    assert data["score"] == pytest.approx(0.1, abs=1e-5)
    assert data["confidence"] == pytest.approx(0.9, abs=1e-5)


def test_label_and_barcode_not_counted_as_defects(
    client: TestClient, mock_http_client: AsyncMock
) -> None:
    # label and barcode have weight 0.0 — should not affect defect score
    mock_http_client.post.return_value = _mock_response(
        [
            _pred("label", 0.99),
            _pred("barcode", 0.95),
        ]
    )

    resp = client.post("/analyze/condition", json={"images_base64": [_TEST_B64]})

    assert resp.status_code == 200
    data = resp.json()
    assert data["condition"] == "good"
    assert data["score"] == pytest.approx(1.0)


def test_worst_image_score_wins(
    client: TestClient, mock_http_client: AsyncMock
) -> None:
    # Image 1: clean → defect 0.0
    # Image 2: chip @ 0.8 → defect 0.8 → overall "poor"
    mock_http_client.post.side_effect = [
        _mock_response([]),
        _mock_response([_pred("chip", 0.8)]),
    ]

    resp = client.post(
        "/analyze/condition",
        json={"images_base64": [_TEST_B64, _TEST_B64]},
    )

    assert resp.status_code == 200
    data = resp.json()
    assert data["condition"] == "poor"
    assert data["score"] == pytest.approx(0.2, abs=1e-5)
    # confidence: image1 no-detection→1.0, image2 avg→0.8; mean = 0.9
    assert data["confidence"] == pytest.approx(0.9, abs=1e-5)


def test_response_schema_always_valid(
    client: TestClient, mock_http_client: AsyncMock
) -> None:
    mock_http_client.post.return_value = _mock_response(
        [
            _pred("stain", 0.5),
            _pred("mark", 0.4),
        ]
    )

    resp = client.post("/analyze/condition", json={"images_base64": [_TEST_B64]})

    assert resp.status_code == 200
    data = resp.json()
    assert data["condition"] in ("good", "moderate", "poor")
    assert 0.0 <= data["score"] <= 1.0
    assert 0.0 <= data["confidence"] <= 1.0


def test_roboflow_http_error_returns_502(
    client: TestClient, mock_http_client: AsyncMock
) -> None:
    mock_http_client.post.side_effect = httpx.ConnectError("connection refused")

    resp = client.post("/analyze/condition", json={"images_base64": [_TEST_B64]})

    assert resp.status_code == 502
    assert resp.json()["detail"] == "AI inference unavailable"


def test_condition_via_image_url(
    client: TestClient, mock_http_client: AsyncMock
) -> None:
    # 1. Mock image download
    image_resp = MagicMock()
    image_resp.content = b"fake-image-bytes"
    image_resp.status_code = 200
    image_resp.raise_for_status = MagicMock()

    mock_http_client.get.return_value = image_resp

    # 2. Mock Roboflow inference
    mock_http_client.post.return_value = _mock_response([])

    resp = client.post(
        "/analyze/condition", json={"image_urls": ["http://example.com/book.jpg"]}
    )

    assert resp.status_code == 200
    data = resp.json()
    assert data["condition"] == "good"
    assert mock_http_client.get.called
    assert mock_http_client.post.called
