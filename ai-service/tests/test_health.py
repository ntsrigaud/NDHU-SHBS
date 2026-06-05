"""Basic health-check tests for the AI microservice."""

from __future__ import annotations

import pytest
from fastapi.testclient import TestClient

from main import app


@pytest.fixture
def client() -> TestClient:
    return TestClient(app)


def test_health_returns_200(client: TestClient) -> None:
    response = client.get("/health")
    assert response.status_code == 200


def test_health_response_schema(client: TestClient) -> None:
    data = client.get("/health").json()
    assert data["status"] == "ok"
    assert isinstance(data["model_loaded"], bool)


def test_condition_stub_returns_200(client: TestClient) -> None:
    response = client.post(
        "/analyze/condition",
        json={"images_base64": ["dGVzdA=="]},  # base64("test")
    )
    assert response.status_code == 200


def test_condition_stub_schema(client: TestClient) -> None:
    data = client.post(
        "/analyze/condition",
        json={"images_base64": ["dGVzdA=="]},
    ).json()
    assert data["condition"] in ("good", "moderate", "poor")
    assert 0.0 <= data["score"] <= 1.0
    assert 0.0 <= data["confidence"] <= 1.0


def test_metadata_returns_200(client: TestClient) -> None:
    # "dGVzdA==" is valid base64 but not an image: no barcode, and OCR is
    # stubbed to empty by conftest, so this resolves to a null/0.0 response.
    response = client.post(
        "/analyze/metadata",
        json={"images_base64": ["dGVzdA=="]},
    )
    assert response.status_code == 200
