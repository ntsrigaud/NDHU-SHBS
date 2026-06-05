"""Tests for the metadata extraction endpoint.

Layers:
  - _decode_isbn is tested for real by generating an EAN-13 barcode image and
    decoding it back (no mocking — proves the zxing-cpp + Pillow path works).
  - Endpoint branches are tested by patching _decode_isbn and _run_ocr and
    mocking app.state.http_client.get (no real network or OCR model).

The autouse _no_real_ocr fixture in conftest.py stubs _run_ocr to [] by
default, so only tests that opt in exercise the OCR path.
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
_IMG = "dGVzdA=="  # placeholder base64 (not a real image)


def _barcode_png_base64(isbn: str) -> str:
    """Generate an EAN-13 barcode for `isbn` and return it as base64 PNG."""
    zimg = zxingcpp.write_barcode(zxingcpp.BarcodeFormat.EAN13, isbn)
    h, w = zimg.shape[0], zimg.shape[1]
    pil = Image.frombytes("L", (w, h), bytes(zimg))
    buf = io.BytesIO()
    pil.save(buf, "PNG")
    return base64.b64encode(buf.getvalue()).decode()


def _ol_lookup(isbn: str, title: str, authors: list[str]) -> MagicMock:
    mock = MagicMock()
    mock.json.return_value = {
        f"ISBN:{isbn}": {"title": title, "authors": [{"name": a} for a in authors]}
    }
    mock.raise_for_status = MagicMock()
    return mock


def _ol_search(title: str, authors: list[str], isbns: list[str]) -> MagicMock:
    mock = MagicMock()
    mock.json.return_value = {
        "docs": [{"title": title, "author_name": authors, "isbn": isbns}]
    }
    mock.raise_for_status = MagicMock()
    return mock


def _empty() -> MagicMock:
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
    assert _decode_isbn(_IMG) is None


# ── Stage 1: barcode pass ───────────────────────────────────────────────────

def test_barcode_match_returns_full_metadata(
    client: TestClient, mock_http_client: AsyncMock, monkeypatch
) -> None:
    monkeypatch.setattr("routers.metadata._decode_isbn", lambda _: _ISBN)
    mock_http_client.get.return_value = _ol_lookup(
        _ISBN, "The C Programming Language", ["Brian W. Kernighan"]
    )

    resp = client.post("/analyze/metadata", json={"images_base64": [_IMG]})

    assert resp.status_code == 200
    data = resp.json()
    assert data["isbn"] == _ISBN
    assert data["title"] == "The C Programming Language"
    assert data["author"] == "Brian W. Kernighan"
    assert data["confidence"] == pytest.approx(0.95)


def test_barcode_found_in_second_image(
    client: TestClient, mock_http_client: AsyncMock, monkeypatch
) -> None:
    # First image has no barcode, second does — the pass should scan all images.
    monkeypatch.setattr(
        "routers.metadata._decode_isbn",
        lambda img: _ISBN if img == "second" else None,
    )
    mock_http_client.get.return_value = _ol_lookup(_ISBN, "Title", ["Author"])

    resp = client.post(
        "/analyze/metadata", json={"images_base64": ["first", "second"]}
    )

    assert resp.status_code == 200
    assert resp.json()["isbn"] == _ISBN


def test_barcode_but_no_catalog_metadata(
    client: TestClient, mock_http_client: AsyncMock, monkeypatch
) -> None:
    monkeypatch.setattr("routers.metadata._decode_isbn", lambda _: _ISBN)
    mock_http_client.get.return_value = _empty()

    resp = client.post("/analyze/metadata", json={"images_base64": [_IMG]})

    data = resp.json()
    assert data["isbn"] == _ISBN
    assert data["title"] is None
    assert data["confidence"] == pytest.approx(0.5)


# ── Stage 2: OCR fallback ───────────────────────────────────────────────────

def test_ocr_fallback_resolves_via_search(
    client: TestClient, mock_http_client: AsyncMock, monkeypatch
) -> None:
    monkeypatch.setattr("routers.metadata._decode_isbn", lambda _: None)
    monkeypatch.setattr(
        "routers.metadata._run_ocr", lambda _raw: ["FLUENT PYTHON", "Luciano Ramalho"]
    )
    mock_http_client.get.return_value = _ol_search(
        "Fluent Python", ["Luciano Ramalho"], ["9781491946008"]
    )

    resp = client.post("/analyze/metadata", json={"images_base64": [_IMG]})

    assert resp.status_code == 200
    data = resp.json()
    assert data["title"] == "Fluent Python"
    assert data["author"] == "Luciano Ramalho"
    assert data["isbn"] == "9781491946008"
    assert data["confidence"] == pytest.approx(0.75)


def test_ocr_rejects_irrelevant_search_result(
    client: TestClient, mock_http_client: AsyncMock, monkeypatch
) -> None:
    # OCR clearly reads a deep-learning book, but the catalog's loose ranking
    # returns an unrelated title. The relevance guard must reject it rather than
    # returning confidently-wrong metadata.
    monkeypatch.setattr("routers.metadata._decode_isbn", lambda _: None)
    monkeypatch.setattr(
        "routers.metadata._run_ocr",
        lambda _raw: ["GENERATIVE DEEP LEARNING", "Second Edition", "David Foster"],
    )
    mock_http_client.get.return_value = _ol_search(
        "The Two Towers", ["J.R.R. Tolkien"], ["9780261102361"]
    )

    resp = client.post("/analyze/metadata", json={"images_base64": [_IMG]})

    assert resp.status_code == 200
    data = resp.json()
    assert data["title"] is None
    assert data["isbn"] is None
    assert data["confidence"] == pytest.approx(0.0)


def test_no_barcode_and_no_ocr_text(
    client: TestClient, monkeypatch
) -> None:
    monkeypatch.setattr("routers.metadata._decode_isbn", lambda _: None)
    # conftest already stubs _run_ocr -> [] (no text recognised)

    resp = client.post("/analyze/metadata", json={"images_base64": [_IMG]})

    assert resp.status_code == 200
    data = resp.json()
    assert data == {"title": None, "author": None, "isbn": None, "confidence": 0.0}


def test_ocr_text_but_no_search_match(
    client: TestClient, mock_http_client: AsyncMock, monkeypatch
) -> None:
    monkeypatch.setattr("routers.metadata._decode_isbn", lambda _: None)
    monkeypatch.setattr("routers.metadata._run_ocr", lambda _raw: ["gibberish xyz"])
    mock_http_client.get.return_value = _empty()  # both searches return nothing

    resp = client.post("/analyze/metadata", json={"images_base64": [_IMG]})

    data = resp.json()
    assert data["confidence"] == pytest.approx(0.0)
    assert data["title"] is None


def test_empty_images_list_rejected(client: TestClient) -> None:
    resp = client.post("/analyze/metadata", json={"images_base64": []})
    assert resp.status_code == 422
