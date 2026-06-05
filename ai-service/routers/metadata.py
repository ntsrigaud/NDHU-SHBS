"""Metadata extraction router.

POST /analyze/metadata
  - Receives one or more base64-encoded book photos (front, back, pages).
  - Resolves authoritative title / author / ISBN from a real book catalog,
    so the values are looked up, not OCR-guessed.

Two-stage strategy (so the user never has to take a close-up of the barcode):

  1. Barcode pass (exact): try to decode an ISBN-13 from ANY of the images.
     If one of the normal shots happens to contain a legible barcode we get
     the exact ISBN for free, then look it up by ISBN.

  2. OCR pass (fallback): when no barcode is legible, OCR the photos to get
     noisy text lines (title, author, publisher…), use them as a catalog
     SEARCH query, then accept a result only if its title actually overlaps
     the OCR text (a relevance guard — otherwise loose search ranking can
     return a confidently-wrong book). The catalog resolves messy OCR text to
     a clean record — including the real ISBN.

Confidence levels:
  0.00  nothing usable (no barcode, no OCR text, or no catalog match)
  0.50  ISBN decoded from a barcode but no catalog metadata found
  0.75  metadata resolved from OCR text via catalog search (not exact)
  0.95  ISBN decoded from a barcode and matched to a catalog entry
"""

from __future__ import annotations

import base64
import binascii
import io
import logging
import re

import httpx
import zxingcpp
from fastapi import APIRouter, Request
from fastapi.concurrency import run_in_threadpool
from PIL import Image, UnidentifiedImageError

from schemas import MetadataRequest, MetadataResponse

logger = logging.getLogger(__name__)

router = APIRouter(prefix="/analyze", tags=["metadata"])

_OPENLIBRARY_LOOKUP_URL = "https://openlibrary.org/api/books"
_OPENLIBRARY_SEARCH_URL = "https://openlibrary.org/search.json"
_GOOGLE_BOOKS_URL = "https://www.googleapis.com/books/v1/volumes"

# (title, author) — either element may be None.
BookInfo = tuple[str | None, str | None]
# (title, author, isbn) resolved from a free-text catalog search.
SearchHit = tuple[str | None, str | None, str | None]

# Lazily-initialised singleton OCR engine (loading the ONNX models is costly,
# so we create it once and reuse it across requests).
_ocr_engine = None


# ── Image / barcode helpers ─────────────────────────────────────────────────

def _b64_to_bytes(image_b64: str) -> bytes | None:
    """Decode a base64 string (with optional data-URI prefix) to raw bytes."""
    if "," in image_b64 and image_b64.lstrip().startswith("data:"):
        image_b64 = image_b64.split(",", 1)[1]
    try:
        return base64.b64decode(image_b64.strip(), validate=True)
    except (binascii.Error, ValueError):
        return None


def _decode_isbn(image_b64: str) -> str | None:
    """Decode an ISBN-13 (EAN-13 Bookland) barcode from one base64 image."""
    raw = _b64_to_bytes(image_b64)
    if raw is None:
        return None
    try:
        image = Image.open(io.BytesIO(raw))
        image.load()
    except (UnidentifiedImageError, OSError) as exc:
        logger.warning("Could not open image for ISBN extraction: %s", exc)
        return None

    for result in zxingcpp.read_barcodes(image):
        text = result.text
        if result.valid and len(text) == 13 and text.startswith(("978", "979")):
            return text
    return None


# ── OCR ─────────────────────────────────────────────────────────────────────

def _get_ocr_engine():
    """Return the shared RapidOCR engine, creating it on first use."""
    global _ocr_engine
    if _ocr_engine is None:
        from rapidocr_onnxruntime import RapidOCR

        _ocr_engine = RapidOCR()
    return _ocr_engine


def prewarm_ocr() -> None:
    """Initialise the OCR engine ahead of the first request (called at startup)."""
    _get_ocr_engine()


def _run_ocr(image_bytes: bytes) -> list[str]:
    """OCR one image; return text lines ordered by prominence (tallest first).

    The largest text on a cover is almost always the title, so sorting by the
    detected box height puts the most useful lines first.
    """
    engine = _get_ocr_engine()
    result, _ = engine(image_bytes)
    if not result:
        return []

    def box_height(box: list[list[float]]) -> float:
        ys = [point[1] for point in box]
        return max(ys) - min(ys)

    ordered = sorted(result, key=lambda item: box_height(item[0]), reverse=True)
    return [text for _box, text, _score in ordered]


# ── Catalog lookups (by exact ISBN) ─────────────────────────────────────────

async def _lookup_openlibrary(client: httpx.AsyncClient, isbn: str) -> BookInfo | None:
    """Look an exact ISBN up in OpenLibrary. Returns (title, author) or None."""
    try:
        resp = await client.get(
            _OPENLIBRARY_LOOKUP_URL,
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
    """Look an exact ISBN up in Google Books. Returns (title, author) or None."""
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


# ── Catalog searches (by free-text OCR query) ───────────────────────────────

async def _search_openlibrary(
    client: httpx.AsyncClient, params: dict[str, str | int], limit: int = 10
) -> list[SearchHit]:
    """Search OpenLibrary with the given field params (e.g. {"title": ...} or
    {"q": ...}). Returns up to `limit` candidate hits."""
    try:
        resp = await client.get(
            _OPENLIBRARY_SEARCH_URL,
            # `fields` must be requested explicitly or isbn is omitted.
            params={**params, "limit": limit, "fields": "title,author_name,isbn"},
        )
        resp.raise_for_status()
        data = resp.json()
    except httpx.HTTPError as exc:
        logger.warning("OpenLibrary search failed for %r: %s", params, exc)
        return []

    hits: list[SearchHit] = []
    for doc in data.get("docs", []):
        authors = doc.get("author_name")
        isbns = doc.get("isbn")
        isbn13 = next((i for i in isbns if len(i) == 13), None) if isbns else None
        hits.append((doc.get("title"), ", ".join(authors) if authors else None, isbn13))
    return hits


async def _search_google_books(
    client: httpx.AsyncClient, query: str, limit: int = 10
) -> list[SearchHit]:
    """Free-text search of Google Books. Returns up to `limit` candidate hits."""
    try:
        resp = await client.get(
            _GOOGLE_BOOKS_URL, params={"q": query, "maxResults": limit}
        )
        resp.raise_for_status()
        data = resp.json()
    except httpx.HTTPError as exc:
        logger.warning("Google Books search failed for %r: %s", query, exc)
        return []

    hits: list[SearchHit] = []
    for item in data.get("items", []):
        info = item.get("volumeInfo", {})
        authors = info.get("authors")
        ids = info.get("industryIdentifiers", [])
        isbn13 = next((i["identifier"] for i in ids if i.get("type") == "ISBN_13"), None)
        hits.append((info.get("title"), ", ".join(authors) if authors else None, isbn13))
    return hits


# ── Query building & relevance guard ────────────────────────────────────────

# Common words dropped before comparing titles so they don't inflate overlap.
_STOPWORDS = frozenset(
    {"the", "a", "an", "of", "and", "or", "to", "in", "on", "for", "with", "by"}
)
# Edition/printing markers that wreck the catalog's conjunctive search if left
# in the query (e.g. "second edition" returns zero hits). Dropped from queries
# only — they are still kept in the OCR token set used for relevance matching.
_EDITION_NOISE = frozenset(
    {
        "edition", "ed", "first", "second", "third", "fourth", "fifth", "sixth",
        "revised", "updated", "expanded", "reprint", "reprinted", "volume", "vol",
    }
)
_SUBTITLE_SPLIT = re.compile(r"[:–—\-]")  # colon / en-dash / em-dash / hyphen


def _tokens(text: str | None) -> set[str]:
    """Lowercase alphanumeric tokens, minus stopwords and 1-char noise."""
    if not text:
        return set()
    words = re.sub(r"[^a-z0-9]+", " ", text.lower()).split()
    return {w for w in words if len(w) >= 2 and w not in _STOPWORDS}


def _content_words(lines: list[str]) -> list[str]:
    """Significant query words in prominence order, de-duplicated.

    Drops stopwords and edition markers so the catalog's conjunctive search
    isn't driven to zero hits by noise.
    """
    seen: set[str] = set()
    out: list[str] = []
    for line in lines:
        for word in re.sub(r"[^a-z0-9]+", " ", line.lower()).split():
            if len(word) < 2 or word in _STOPWORDS or word in _EDITION_NOISE:
                continue
            if word not in seen:
                seen.add(word)
                out.append(word)
    return out


def _best_candidate(
    candidates: list[SearchHit], ocr_tokens: set[str], threshold: float = 0.6
) -> SearchHit | None:
    """Pick the catalog result that best matches the OCR text.

    A candidate qualifies when most of its MAIN-title tokens (subtitle after a
    ":"/"-" dropped) appear in the OCR text — this rejects unrelated books that
    loose search ranking returns. Among qualifying candidates we prefer the
    most *specific* title (most matched tokens), so "Generative Deep Learning"
    beats the subset title "Deep Learning" when the cover shows "generative".
    """
    best: SearchHit | None = None
    best_key = (0.0, 0)
    for hit in candidates:
        main_tokens = _tokens(_SUBTITLE_SPLIT.split(hit[0], 1)[0]) if hit[0] else set()
        if not main_tokens:
            continue
        matched = len(main_tokens & ocr_tokens)
        precision = matched / len(main_tokens)
        if precision < threshold:
            continue
        key = (precision, matched)
        if key > best_key:
            best_key, best = key, hit
    return best


# ── Endpoint ────────────────────────────────────────────────────────────────

@router.post("/metadata", response_model=MetadataResponse)
async def analyze_metadata(body: MetadataRequest, request: Request) -> MetadataResponse:
    """Extract book metadata (title, author, ISBN) from one or more photos."""
    client: httpx.AsyncClient = request.app.state.http_client

    # Stage 1 — barcode pass across all images (exact when a barcode is legible).
    for image_b64 in body.images_base64:
        isbn = _decode_isbn(image_b64)
        if isbn is not None:
            result = await _lookup_openlibrary(client, isbn)
            if result is None:
                result = await _lookup_google_books(client, isbn)
            if result is None:
                return MetadataResponse(title=None, author=None, isbn=isbn, confidence=0.5)
            title, author = result
            return MetadataResponse(title=title, author=author, isbn=isbn, confidence=0.95)

    # Stage 2 — OCR fallback: read text from the photos and search the catalog.
    text_lines: list[str] = []
    for image_b64 in body.images_base64:
        raw = _b64_to_bytes(image_b64)
        if raw is None:
            continue
        # OCR is CPU-bound; run it off the event loop.
        text_lines.extend(await run_in_threadpool(_run_ocr, raw))

    if not text_lines:
        return MetadataResponse(title=None, author=None, isbn=None, confidence=0.0)

    ocr_tokens = _tokens(" ".join(text_lines))
    words = _content_words(text_lines)
    if not words:
        return MetadataResponse(title=None, author=None, isbn=None, confidence=0.0)

    short_query = " ".join(words[:3])  # core title — survives conjunctive search
    full_query = " ".join(words[:8])   # + author/publisher words to disambiguate

    # The catalog's search is conjunctive, so a long noisy query returns nothing.
    # Try a short title-field query first, then progressively broader fallbacks.
    # OpenLibrary is tried before Google Books (which 429s unauthenticated calls).
    # Each result is accepted only if its title overlaps the OCR text.
    attempts: list[tuple] = [
        (_search_openlibrary, ({"title": short_query},)),
        (_search_openlibrary, ({"q": full_query},)),
        (_search_google_books, (full_query,)),
    ]
    for search, args in attempts:
        best = _best_candidate(await search(client, *args), ocr_tokens)
        if best is not None:
            title, author, isbn = best
            return MetadataResponse(
                title=title, author=author, isbn=isbn, confidence=0.75
            )

    # OCR text recognised, but no catalog result matched it confidently.
    return MetadataResponse(title=None, author=None, isbn=None, confidence=0.0)
