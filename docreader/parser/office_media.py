"""Rasterize vector media emitted by Office document parsers."""

from __future__ import annotations

import base64
import io
import logging
import os
import re
import shutil
import subprocess
import tempfile
import uuid
from pathlib import Path
from typing import Dict, List, Sequence, Tuple

from PIL import Image, ImageChops

logger = logging.getLogger(__name__)

_VECTOR_DATA_IMAGE = re.compile(
    r"!\[([^\]]*)\]\(data:image/(x-wmf|wmf|x-emf|emf|svg\+xml);base64,"
    r"([A-Za-z0-9+/=\s]+)\)",
    re.IGNORECASE,
)
_RASTER_EXT = {".png", ".jpg", ".jpeg", ".gif", ".webp", ".bmp"}
_OFFICE_VECTOR_EXT = {".wmf", ".emf", ".x-wmf", ".x-emf"}
_VECTOR_MIME_SUFFIX = {
    "x-wmf": ".wmf",
    "wmf": ".wmf",
    "x-emf": ".emf",
    "emf": ".emf",
    "svg+xml": ".svg",
}


def _image_as_png(data: bytes, *, trim_white: bool) -> bytes | None:
    try:
        source = Image.open(io.BytesIO(data)).convert("RGBA")
    except Exception as exc:
        logger.debug("Pillow could not open media bytes: %s", exc)
        return None

    flattened = Image.new("RGB", source.size, "white")
    flattened.paste(source, mask=source.getchannel("A"))
    if trim_white:
        white = Image.new("RGB", flattened.size, "white")
        difference = ImageChops.difference(flattened, white).convert("L")
        bbox = difference.point(lambda pixel: 255 if pixel > 8 else 0).getbbox()
        if bbox is None:
            return None
        flattened = flattened.crop(bbox)
        padded = Image.new(
            "RGB",
            (flattened.width + 16, flattened.height + 16),
            "white",
        )
        padded.paste(flattened, (8, 8))
        flattened = padded

    output = io.BytesIO()
    flattened.save(output, format="PNG", optimize=True)
    return output.getvalue()


def _rasterize_with_imagemagick(data: bytes, suffix: str) -> bytes | None:
    convert = shutil.which("convert")
    if not convert:
        return None
    with tempfile.TemporaryDirectory() as temp_dir:
        source = os.path.join(temp_dir, f"input{suffix}")
        destination = os.path.join(temp_dir, "output.png")
        Path(source).write_bytes(data)
        try:
            result = subprocess.run(
                [convert, source, destination],
                capture_output=True,
                timeout=60,
                check=False,
            )
        except (OSError, subprocess.TimeoutExpired) as exc:
            logger.warning("ImageMagick conversion failed: %s", exc)
            return None
        if result.returncode != 0 or not os.path.isfile(destination):
            return None
        return _image_as_png(Path(destination).read_bytes(), trim_white=True)


def _run_soffice_batch(items: Sequence[Tuple[str, bytes]]) -> List[bytes | None]:
    soffice = shutil.which("soffice") or shutil.which("libreoffice")
    if not soffice or not items:
        return [None] * len(items)

    with tempfile.TemporaryDirectory() as temp_dir:
        root = Path(temp_dir)
        output_dir = root / "output"
        profile_dir = root / "profile"
        output_dir.mkdir()
        profile_dir.mkdir()
        sources: List[Path] = []
        for index, (suffix, data) in enumerate(items):
            source = root / f"input_{index:04d}{suffix}"
            source.write_bytes(data)
            sources.append(source)
        command = [
            soffice,
            "--headless",
            f"-env:UserInstallation={profile_dir.as_uri()}",
            "--convert-to",
            "png",
            "--outdir",
            str(output_dir),
            *(str(source) for source in sources),
        ]
        try:
            result = subprocess.run(command, capture_output=True, timeout=180, check=False)
        except (OSError, subprocess.TimeoutExpired) as exc:
            logger.warning("LibreOffice vector conversion failed: %s", exc)
            return [None] * len(items)
        if result.returncode != 0:
            logger.warning(
                "LibreOffice vector conversion exited %s: %s",
                result.returncode,
                result.stderr.decode("utf-8", errors="ignore"),
            )
        return [
            _image_as_png((output_dir / f"{source.stem}.png").read_bytes(), trim_white=True)
            if (output_dir / f"{source.stem}.png").is_file()
            else None
            for source in sources
        ]


def _normalized_suffix(name: str) -> str:
    suffix = os.path.splitext(name)[1].lower() or ".bin"
    if suffix == ".x-wmf":
        return ".wmf"
    if suffix == ".x-emf":
        return ".emf"
    return suffix


def rasterize_media_bytes(name: str, data: bytes) -> bytes | None:
    suffix = _normalized_suffix(name)
    if suffix in _RASTER_EXT:
        return _image_as_png(data, trim_white=False)
    if suffix in {".wmf", ".emf"}:
        converted = _rasterize_with_imagemagick(data, suffix)
        return converted or _run_soffice_batch([(suffix, data)])[0]
    return _rasterize_with_imagemagick(data, suffix)


def rasterize_media_batch(items: Sequence[Tuple[str, bytes]]) -> List[bytes | None]:
    results: List[bytes | None] = [None] * len(items)
    office_items: List[Tuple[str, bytes]] = []
    office_indexes: List[int] = []
    for index, (name, data) in enumerate(items):
        suffix = os.path.splitext(name)[1].lower()
        if suffix in _OFFICE_VECTOR_EXT:
            office_items.append((_normalized_suffix(name), data))
            office_indexes.append(index)
        else:
            results[index] = rasterize_media_bytes(name, data)
    for index, converted in zip(office_indexes, _run_soffice_batch(office_items)):
        if converted is None:
            name, data = items[index]
            converted = _rasterize_with_imagemagick(data, _normalized_suffix(name))
        results[index] = converted
    return results


def attach_vector_data_uris_to_markdown(
    markdown: str,
) -> Tuple[str, Dict[str, str]]:
    matches = list(_VECTOR_DATA_IMAGE.finditer(markdown or ""))
    if not matches:
        return markdown, {}

    media: List[Tuple[str, bytes]] = []
    valid_matches: List[re.Match[str]] = []
    for match in matches:
        try:
            payload = base64.b64decode(re.sub(r"\s+", "", match.group(3)), validate=True)
        except ValueError:
            logger.warning("Skipping invalid Office vector data URI")
            continue
        suffix = _VECTOR_MIME_SUFFIX[match.group(2).lower()]
        media.append((f"vector{suffix}", payload))
        valid_matches.append(match)

    replacements: Dict[int, str] = {}
    images: Dict[str, str] = {}
    for match, png in zip(valid_matches, rasterize_media_batch(media)):
        if not png:
            continue
        ref = f"images/{uuid.uuid4()}.png"
        images[ref] = base64.b64encode(png).decode()
        replacements[match.start()] = f"![{match.group(1)}]({ref})"

    rendered = _VECTOR_DATA_IMAGE.sub(
        lambda match: replacements.get(match.start(), match.group(0)),
        markdown,
    )
    return rendered, images
