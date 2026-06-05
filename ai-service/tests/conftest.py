"""Shared test fixtures for the ai-service test suite."""

from __future__ import annotations

from unittest.mock import AsyncMock, MagicMock

import pytest

from main import app


def _default_mock_response() -> MagicMock:
    """An httpx response mock that returns empty predictions (no defects)."""
    mock = MagicMock()
    mock.json.return_value = {"predictions": []}
    mock.raise_for_status = MagicMock()
    return mock


@pytest.fixture(autouse=True)
def mock_http_client() -> AsyncMock:
    """Inject a mock HTTP client into app.state before every test.

    The default response is empty predictions so condition tests in test_health.py
    receive a valid 200 response without making real network calls. Tests in
    test_condition.py override mock_http_client.post.return_value as needed.
    """
    mock = AsyncMock()
    mock.post.return_value = _default_mock_response()
    app.state.http_client = mock
    return mock
