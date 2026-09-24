"""Regenerate the Windows application icon from its SVG source."""

from io import BytesIO
from pathlib import Path

import resvg_py
from PIL import Image


ROOT = Path(__file__).resolve().parents[1]
SOURCE = ROOT / "assets/icons/app/eqm-logo.svg"
OUTPUT = ROOT / "internal/resources/icons/app/eqm.ico"

png = resvg_py.svg_to_bytes(svg_path=str(SOURCE), width=512, height=512)
OUTPUT.parent.mkdir(parents=True, exist_ok=True)
with Image.open(BytesIO(png)) as image:
    image.save(OUTPUT, format="ICO", sizes=[(size, size) for size in (16, 32, 48, 256)])
