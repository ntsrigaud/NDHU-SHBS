"""Metadata extraction router.

POST /analyze/metadata
  - Receives a base64-encoded book image (ideally the back cover barcode).
  - Decodes the ISBN-13 (EAN-13 Bookland) barcode with zxing-cpp.
  - Looks the ISBN up in a book catalog to obtain authoritative title and
    author — these come from a real database, not OCR guesses.

The catalog lookup tries OpenLibrary first (no API key, lenient rate limits)
and falls back to Google Books only if OpenLibrary has no match. This avoids
Google Books' aggressive per-IP 429 throttling on unauthenticated requests.

Confidence levels:
  0.00  no ISBN barcode found in the image
  0.50  ISBN decoded but no catalog match (or both providers unreachable)
  0.95  ISBN decoded and matched to a catalog entry
"""

from __future__ import annotations

import base64
import binascii
import io
import logging

import httpx
import zxingcpp
from fastapi import APIRouter, Request
from PIL import Image, UnidentifiedImageError

from schemas import MetadataRequest, MetadataResponse

logger = logging.getLogger(__name__)

router = APIRouter(prefix="/analyze", tags=["metadata"])

_OPENLIBRARY_URL = "https://openlibrary.org/api/books"
_GOOGLE_BOOKS_URL = "https://www.googleapis.com/books/v1/volumes"

# A (title, author) pair; either element may be None.
BookInfo = tuple[str | None, str | None]


def _decode_isbn(image_b64: str) -> str | None:
    """Decode an ISBN-13 barcode from a base64-encoded image.

    Returns the 13-digit ISBN string, or None if the image cannot be decoded
    or contains no valid Bookland (978/979) EAN-13 barcode.
    """
    # Strip a data-URI prefix if the caller included one.
    if "," in image_b64 and image_b64.lstrip().startswith("data:"):
        image_b64 = image_b64.split(",", 1)[1]

    try:
        raw = base64.b64decode(image_b64.strip(), validate=True)
        image = Image.open(io.BytesIO(raw))
        image.load()
    except (binascii.Error, ValueError, UnidentifiedImageError, OSError) as exc:
        logger.warning("Could not decode image for ISBN extraction: %s", exc)
        return None

    for result in zxingcpp.read_barcodes(image):
        text = result.text
        # Book ISBNs are EAN-13 barcodes in the Bookland range (978/979).
        if result.valid and len(text) == 13 and text.startswith(("978", "979")):
            return text
    return None


async def _lookup_openlibrary(client: httpx.AsyncClient, isbn: str) -> BookInfo | None:
    """Look an ISBN up in OpenLibrary. Returns (title, author) or None."""
    try:
        resp = await client.get(
            _OPENLIBRARY_URL,
            params={"bibkeys": f"ISBN:{isbn}", "format": "json", "jscmd": "data"},
        )
        resp.raise_for_status()
        data = resp.json()
    except httpx.HTTPError as exc:
        logger.warning("OpenLibrary lookup failed for %s: %s", isbn, exc)
        return None

    entry = data.get(f"ISBN:{isbn}")
    if not entry:
        return None
    authors = entry.get("authors")
    author = ", ".join(a["name"] for a in authors if a.get("name")) if authors else None
    return entry.get("title"), author


async def _lookup_google_books(client: httpx.AsyncClient, isbn: str) -> BookInfo | None:
    """Look an ISBN up in Google Books. Returns (title, author) or None."""
    try:
        resp = await client.get(_GOOGLE_BOOKS_URL, params={"q": f"isbn:{isbn}"})
        resp.raise_for_status()
        data = resp.json()
    except httpx.HTTPError as exc:
        logger.warning("Google Books lookup failed for %s: %s", isbn, exc)
        return None

    items = data.get("items", [])
    if not items:
        return None
    info = items[0].get("volumeInfo", {})
    authors = info.get("authors")
    return info.get("title"), (", ".join(authors) if authors else None)


@router.post("/metadata", response_model=MetadataResponse)
async def analyze_metadata(body: MetadataRequest, request: Request) -> MetadataResponse:
    """Extract book metadata (title, author, ISBN) from a cover/barcode image."""
    isbn = _decode_isbn(body.image_base64)
    if isbn is None:
        return MetadataResponse(title=None, author=None, isbn=None, confidence=0.0)

    client: httpx.AsyncClient = request.app.state.http_client

    # OpenLibrary first (avoids Google Books' 429 throttling), then fall back.
    result = await _lookup_openlibrary(client, isbn)
    if result is None:
        result = await _lookup_google_books(client, isbn)

    if result is None:
        # Barcode decoded but no metadata available — return the ISBN so the
        # seller can fill in title/author manually.
        return MetadataResponse(title=None, author=None, isbn=isbn, confidence=0.5)

    title, author = result
    return MetadataResponse(title=title, author=author, isbn=isbn, confidence=0.95)
