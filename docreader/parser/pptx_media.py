"""Extract and rasterize images embedded in PPTX (e.g. WMF) when MarkItDown cannot inline them."""

from __future__ import annotations

import base64
import io
import logging
import os
import re
import uuid
import zipfile
from typing import Dict, List, Tuple

from docreader.parser.office_media import rasterize_media_bytes

logger = logging.getLogger(__name__)

_MARKDOWN_IMAGE = re.compile(r"!\[([^\]]*)\]\(([^)]+)\)")


def list_pptx_media(pptx_bytes: bytes) -> List[Tuple[str, bytes]]:
    """Return (zip path, raw bytes) for each file under ppt/media/, in archive order."""
    items: List[Tuple[str, bytes]] = []
    with zipfile.ZipFile(io.BytesIO(pptx_bytes)) as archive:
        for name in archive.namelist():
            if not name.startswith("ppt/media/"):
                continue
            base = os.path.basename(name)
            if not base or base.startswith("."):
                continue
            items.append((name, archive.read(name)))
    return items


def extract_pptx_media_rasterized(pptx_bytes: bytes) -> List[bytes]:
    """Rasterize all ppt/media assets to PNG bytes, skipping failures."""
    rasterized: List[bytes] = []
    for path, raw in list_pptx_media(pptx_bytes):
        png = rasterize_media_bytes(os.path.basename(path), raw)
        if png:
            rasterized.append(png)
            logger.info("Rasterized pptx media %s (%d -> %d bytes)", path, len(raw), len(png))
        else:
            logger.warning("Failed to rasterize pptx media %s", path)
    return rasterized


def _is_unresolved_image_ref(url: str) -> bool:
    if not url or url.startswith("data:") or url.startswith("images/"):
        return False
    if url.startswith(("http://", "https://")):
        return False
    return True


def attach_pptx_media_to_markdown(
    markdown: str, pptx_bytes: bytes
) -> Tuple[str, Dict[str, str]]:
    """Replace unresolved ![](...) refs with images/ paths and inline image payloads."""
    media = extract_pptx_media_rasterized(pptx_bytes)
    if not media:
        return markdown, {}

    images: Dict[str, str] = {}
    media_iter = iter(media)

    def repl(match: re.Match[str]) -> str:
        alt, url = match.group(1), match.group(2)
        if not _is_unresolved_image_ref(url):
            return match.group(0)
        try:
            png = next(media_iter)
        except StopIteration:
            return match.group(0)
        ref = f"images/{uuid.uuid4()}.png"
        images[ref] = base64.b64encode(png).decode()
        return f"![{alt}]({ref})"

    return _MARKDOWN_IMAGE.sub(repl, markdown), images


def markdown_needs_pptx_media_attach(markdown: str) -> bool:
    return any(_is_unresolved_image_ref(m.group(2)) for m in _MARKDOWN_IMAGE.finditer(markdown))
