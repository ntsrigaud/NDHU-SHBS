"""Tests for the metadata extraction endpoint.

Two layers:
  - _decode_isbn is tested for real by generating an EAN-13 barcode image and
    decoding it back (no mocking — proves the zxing-cpp + Pillow path works).
  - The endpoint's catalog-lookup branches are tested by patching _decode_isbn
    and mocking app.state.http_client.get (no real network calls).
"""

from __future__ import annotations

import base64
import io
from unittest.mock import AsyncMock, MagicMock

import pytest
import zxingcpp
from fastapi.testclient import TestClient
from PIL import Image

from main import app
from routers.metadata import _decode_isbn

_ISBN = "9780131103627"  # The C Programming Language


def _barcode_png_base64(isbn: str) -> str:
    """Generate an EAN-13 barcode for `isbn` and return it as base64 PNG."""
    zimg = zxingcpp.write_barcode(zxingcpp.BarcodeFormat.EAN13, isbn)
    h, w = zimg.shape[0], zimg.shape[1]
    pil = Image.frombytes("L", (w, h), bytes(zimg))
    buf = io.BytesIO()
    pil.save(buf, "PNG")
    return base64.b64encode(buf.getvalue()).decode()


def _google_response(items: list[dict]) -> MagicMock:
    mock = MagicMock()
    mock.json.return_value = {"items": items} if items else {}
    mock.raise_for_status = MagicMock()
    return mock


def _openlibrary_response(isbn: str, title: str, authors: list[str]) -> MagicMock:
    mock = MagicMock()
    mock.json.return_value = {
        f"ISBN:{isbn}": {"title": title, "authors": [{"name": a} for a in authors]}
    }
    mock.raise_for_status = MagicMock()
    return mock


def _empty_response() -> MagicMock:
    mock = MagicMock()
    mock.json.return_value = {}
    mock.raise_for_status = MagicMock()
    return mock


@pytest.fixture
def client() -> TestClient:
    return TestClient(app)


# ── _decode_isbn (real decode, no mocks) ────────────────────────────────────

def test_decode_isbn_from_real_barcode() -> None:
    assert _decode_isbn(_barcode_png_base64(_ISBN)) == _ISBN


def test_decode_isbn_returns_none_for_non_image() -> None:
    # "dGVzdA==" is valid base64 but not an image.
    assert _decode_isbn("dGVzdA==") is None


def test_decode_isbn_returns_none_for_garbage() -> None:
    assert _decode_isbn("not-valid-base64!!!") is None


# ── Endpoint branches ───────────────────────────────────────────────────────

def test_no_barcode_returns_nulls(client: TestClient, monkeypatch) -> None:
    monkeypatch.setattr("routers.metadata._decode_isbn", lambda _: None)

    resp = client.post("/analyze/metadata", json={"image_base64": "dGVzdA=="})

    assert resp.status_code == 200
    data = resp.json()
    assert data == {"title": None, "author": None, "isbn": None, "confidence": 0.0}


def test_openlibrary_match_returns_full_metadata(
    client: TestClient, mock_http_client: AsyncMock, monkeypatch
) -> None:
    monkeypatch.setattr("routers.metadata._decode_isbn", lambda _: _ISBN)
    mock_http_client.get.return_value = _openlibrary_response(
        _ISBN, "The C Programming Language", ["Brian W. Kernighan", "Dennis M. Ritchie"]
    )

    resp = client.post("/analyze/metadata", json={"image_base64": "x"})

    assert resp.status_code == 200
    data = resp.json()
    assert data["isbn"] == _ISBN
    assert data["title"] == "The C Programming Language"
    assert data["author"] == "Brian W. Kernighan, Dennis M. Ritchie"
    assert data["confidence"] == pytest.approx(0.95)


def test_falls_back_to_google_books(
    client: TestClient, mock_http_client: AsyncMock, monkeypatch
) -> None:
    monkeypatch.setattr("routers.metadata._decode_isbn", lambda _: _ISBN)
    # First call (OpenLibrary) finds nothing; second call (Google Books) matches.
    mock_http_client.get.side_effect = [
        _empty_response(),
        _google_response([{"volumeInfo": {"title": "Fluent Python", "authors": ["Luciano Ramalho"]}}]),
    ]

    resp = client.post("/analyze/metadata", json={"image_base64": "x"})

    assert resp.status_code == 200
    data = resp.json()
    assert data["title"] == "Fluent Python"
    assert data["author"] == "Luciano Ramalho"
    assert data["confidence"] == pytest.approx(0.95)


def test_isbn_decoded_but_no_catalog_match(
    client: TestClient, mock_http_client: AsyncMock, monkeypatch
) -> None:
    monkeypatch.setattr("routers.metadata._decode_isbn", lambda _: _ISBN)
    # Both providers return empty.
    mock_http_client.get.return_value = _empty_response()

    resp = client.post("/analyze/metadata", json={"image_base64": "x"})

    assert resp.status_code == 200
    data = resp.json()
    assert data["isbn"] == _ISBN
    assert data["title"] is None
    assert data["author"] is None
    assert data["confidence"] == pytest.approx(0.5)


def test_isbn_decoded_but_both_providers_fail(
    client: TestClient, mock_http_client: AsyncMock, monkeypatch
) -> None:
    import httpx

    monkeypatch.setattr("routers.metadata._decode_isbn", lambda _: _ISBN)
    mock_http_client.get.side_effect = httpx.ConnectError("offline")

    resp = client.post("/analyze/metadata", json={"image_base64": "x"})

    assert resp.status_code == 200
    data = resp.json()
    assert data["isbn"] == _ISBN
    assert data["title"] is None
    assert data["confidence"] == pytest.approx(0.5)
